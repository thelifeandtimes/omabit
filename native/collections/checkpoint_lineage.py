"""Bounded, non-leasing reads of retained desktop checkpoint ancestry.

Legacy validators remain authoritative for their own records. This reader checks
links and canonical evidence without opening a retired desktop or granting any
runtime permission. New checkpoint versions must be explicitly supported here.
"""
from dataclasses import dataclass
from pathlib import Path

import authority_inspection as inspection
import journal as j
import runtime_authority as authority
import session_checkpoint as first
import session_successor as successor
import successor_checkpoint as second


MAX_NODES = 64
MAX_FILES = 4096
MAX_BYTES = 64 << 20


@dataclass(frozen=True)
class Limits:
    nodes: int = MAX_NODES
    files: int = MAX_FILES
    bytes: int = MAX_BYTES

    def check(self):
        for value, ceiling in ((self.nodes, MAX_NODES), (self.files, MAX_FILES),
                               (self.bytes, MAX_BYTES)):
            j.require(type(value) is int and 0 < value <= ceiling,
                      'invalid checkpoint ancestry limit')


class _Evidence:
    def __init__(self, limits):
        limits.check()
        self.limits = limits
        self.reader = inspection.Reader()
        self.bodies = {}
        self.bytes = 0
        # The mutable journal is loaded atomically by Store, never pinned to an
        # inode. Reserve its entire existing maximum before allocating a body.
        j.require(limits.bytes >= j.MAX_BYTES, 'checkpoint ancestry byte limit: preserve evidence')
        self.remaining = limits.bytes - j.MAX_BYTES

    def bound(self, reference, maximum=authority.MAX_BYTES):
        successor.reference_shape(reference)
        path = reference['object']['path']
        old = self.reader.files.get(path)
        if old is not None:
            j.require({k:old[k] for k in ('object', 'sha256')} == reference,
                      'conflicting checkpoint ancestry reference')
            j.require(old['maximum'] <= maximum, 'checkpoint evidence exceeds file limit')
            return self.bodies[path]
        j.require(len(self.bodies) + 1 < self.limits.files,
                  'checkpoint ancestry file limit: preserve evidence')
        authority.check_object(reference['object'], 'file')
        size = Path(path).lstat().st_size
        j.require(size <= maximum, 'checkpoint evidence exceeds file limit')
        j.require(size <= self.remaining, 'checkpoint ancestry byte limit: preserve evidence')
        # Pin the size before Reader opens/parses it. Growth is rejected before
        # allocation; a concurrent shrink cannot increase the reserved budget.
        value = self.reader.read(path, maximum=size)
        row = self.reader.files[path]
        j.require({k:row[k] for k in ('object', 'sha256')} == reference,
                  'checkpoint ancestry reference changed')
        self.bytes += size
        self.remaining -= size
        self.bodies[path] = value
        return value


def _node(record):
    version = record.get('format')
    j.require(type(version) is int, 'unknown checkpoint ancestry format; preserve it')
    if version == 1:
        first.validate(record)
        return first.filename(record['id']), None
    if version == 2:
        second.validate(record)
        return second.filename(record['id']), record['ancestor_checkpoint']
    if version == 3:
        import repeated_checkpoint
        repeated_checkpoint.validate(record)
        return repeated_checkpoint.filename(record['id']), record['ancestor_checkpoint']
    raise j.Conflict('unknown checkpoint ancestry format; preserve it')


def _edge(child, parent, parent_reference):
    intent_reader = successor
    if child['format'] == 2:
        j.require(parent['format'] == 1, 'unsupported checkpoint ancestry edge; preserve it')
    else:
        j.require(child['format'] == 3, 'unsupported checkpoint ancestry edge; preserve it')
        if child['inspection']['format'] == 3:
            j.require(parent['format'] == 2, 'unsupported checkpoint ancestry edge; preserve it')
            import session_continuation as intent_reader
        else:
            j.require(child['inspection']['format'] == 4 and parent['format'] in (1,2,3),
                      'unsupported checkpoint ancestry edge; preserve it')
            import repeated_session as intent_reader
    descriptor = child['inspection']['descriptor']
    intent = intent_reader.shape(second.parsed(child, descriptor['successor']['object']['path'],
                                          reference=descriptor['successor']))
    j.require(intent['checkpoint'] == parent_reference and
              parent['id'] == intent['recovery'] and
              parent['inspection']['authority'] == intent['predecessor'],
              'checkpoint predecessor changed')
    j.require(parent['cleanup'] == child['inspection']['lineage']['cleanup'],
              'checkpoint cleanup link changed')
    receipt = first.parsed(parent, parent['cleanup']['object']['path'], reference=parent['cleanup'])
    j.require(intent['members'] == receipt['verified']['members'] and
              intent['loaded'] == receipt['plan']['loaded'] and
              intent['specification'] == receipt['plan']['specification'],
              'checkpoint closed cohort changed')
    j.require(intent['journal'] == {k:parent['inspection']['journal'][k]
                                   for k in ('events', 'sha256')},
              'checkpoint predecessor journal changed')


def read(reference, coordinator, *, limits=Limits()):
    """Read a coherent immutable chain plus one current journal snapshot.

    A caller may already hold the canonical lease. This function never takes a
    lease, touches an ancestor runtime, or pins the appendable journal inode.
    Limits may only narrow the defaults. Refusal never modifies retained files.
    """
    evidence = _Evidence(limits)
    coordinator = j.absolute(str(coordinator))
    canonical = authority.object_identity(coordinator, 'directory')
    lock = authority.object_identity(coordinator/'coordinator.lock', 'file')
    evidence.reader.directory(canonical)
    records, references, seen_paths, seen_objects = [], [], set(), set()
    current = reference
    while current is not None:
        successor.reference_shape(current)
        path = current['object']['path']
        identity = tuple(current['object'][k] for k in ('device', 'inode'))
        j.require(path not in seen_paths and identity not in seen_objects,
                  'repeated checkpoint ancestry node; preserve evidence')
        j.require(len(records) < limits.nodes,
                  'checkpoint ancestry node limit: preserve evidence')
        # Validate the canonical location before following any parent path.
        j.require(Path(path).parent == coordinator, 'checkpoint is outside its coordinator')
        seen_paths.add(path)
        seen_objects.add(identity)
        record = evidence.bound(current, maximum=first.MAX_BYTES)
        name, parent = _node(record)
        j.require(path == str(coordinator/name) and record['coordinator'] == canonical and
                  record['coordinator_lock'] == lock, 'checkpoint coordinator binding changed')
        if records:
            _edge(records[-1], record, current)
        records.append(record)
        references.append(current)
        # Canonical metadata survives desktop retirement. Runtime metadata is
        # validated from copied bytes by the version-specific validator only.
        for source, row in record['files'].items():
            if Path(source).parent == coordinator and source != str(coordinator/'journal.json'):
                evidence.bound({k:row[k] for k in ('object', 'sha256')})
        current = parent
    j.require(records, 'checkpoint ancestry is empty')
    for index, record in enumerate(records):
        if record['format'] == 3:
            j.require(record['inspection']['lineage']['ancestors'] == references[index+2:],
                      'checkpoint ancestry inventory changed')
    store = j.Store(coordinator)
    try:
        events, state = store._load()
    finally:
        store.close()
    for record in records:
        prefix = record['inspection']['journal']
        j.require(len(events) >= prefix['events'] and
                  j.digest(events[:prefix['events']]) == prefix['sha256'],
                  'checkpoint canonical prefix changed')
    evidence.reader.verify()
    authority.check_object(canonical, 'directory')
    authority.check_object(lock, 'file')
    return {'head': records[0], 'records': records, 'references': references,
            'events': events, 'state': state,
            'usage': {'nodes': len(records), 'files': len(evidence.bodies) + 1,
                      'immutable_bytes': evidence.bytes, 'journal_byte_reserve': j.MAX_BYTES}}
