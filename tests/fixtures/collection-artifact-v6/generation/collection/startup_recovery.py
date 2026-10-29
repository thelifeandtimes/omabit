"""Durable cleanup of an initial activation before canonical adoption.

This module performs no loader operations. A bound adapter verifies private app
cohorts and exact cleanup; the original runtime authority and canonical lease
serialize it. A verified record alone never grants a replacement reservation.
"""
import copy
import os
from pathlib import Path
import uuid

import journal as j
import runtime_authority as authority

FORMAT = 1
PREFIX = 'initial-recovery-'


def filename(recovery):
    j.token(recovery)
    return PREFIX + recovery + '.json'


admission_snapshot = authority.initial_admission_snapshot
plan_shape = authority.initial_plan_shape
record_shape = authority.initial_recovery_shape


def read_record(runtime, recovery):
    try:record, digest = authority.read_private_json(runtime.root / filename(recovery))
    except FileNotFoundError:return None
    record_shape(record)
    j.require(record['id'] == recovery and record['plan']['authority'] == runtime.descriptor['id'],
              'initial recovery belongs to another authority or request')
    return record


def sync_record(runtime, recovery):
    fd = os.open(filename(recovery), os.O_RDONLY | os.O_NOFOLLOW | os.O_CLOEXEC, dir_fd=runtime.fd)
    try:os.fsync(fd)
    finally:os.close(fd)
    os.fsync(runtime.fd)


def write_record(runtime, record):
    record_shape(record)
    temporary = '.pending-initial-recovery-' + record['id'] + '-' + uuid.uuid4().hex
    authority.write_new(runtime.fd, temporary, record)
    os.replace(temporary, filename(record['id']), src_dir_fd=runtime.fd, dst_dir_fd=runtime.fd)
    os.fsync(runtime.fd)


def context(runtime, store, lease):
    """Verify the exact unadopted activation and its original complete closure."""
    runtime._canonical(store, lease)
    value = runtime.status()
    j.require(value['format'] >= 2 and value['reservation'] is not None and value['binding'] is None,
              'initial recovery requires an owned, unbound reservation')
    original = value['reservation']
    j.require(original['intent'] in ('initial', 'initial-retry'), 'not an initial activation')
    events, state = store._load()
    runtime._history(value, events)
    j.require(state['committed'] is None and state['pending'] is None and not state['requests'],
              'initial recovery cannot reset adopted collection history')
    spec = original['permit']['specification'];key = j.digest(spec)
    j.require(state['collections'].get(key) == spec, 'initial collection registration changed')
    closure = runtime._closure_owner(original['owner'])
    row, snapshot = runtime._partial_gate(original)
    j.require(row is None or all(authority.admission.identity(claim['owner']['pid']) != claim['owner']
                                for claim in row['claims'].values()), 'an initial guard remains live')
    return {'authority': runtime.descriptor['id'], 'reservation_sha256': j.digest(original),
            'activation': {'collection': key, 'epoch': original['permit']['epoch']},
            'journal': {'events': len(events), 'sha256': j.digest(events)},
            'guardian': copy.deepcopy(original['owner']['guardian']),
            'closure_request': closure['request'], 'admission': snapshot}, closure, spec


def validate_context(runtime, store, lease, plan):
    plan_shape(plan)
    expected, closure, spec = context(runtime, store, lease)
    j.require(all(plan[key] == value for key, value in expected.items()),
              'initial reservation, guardian, registration history or admission changed')
    return closure, spec


def prepare(runtime, store, lease, recovery, plan):
    filename(recovery)
    validate_context(runtime, store, lease, plan)
    old = read_record(runtime, recovery)
    if old is not None:
        j.require(old['plan'] == plan, 'initial recovery replay changed')
        sync_record(runtime, recovery)
        return old
    names = [name for name in os.listdir(runtime.fd) if name.startswith(PREFIX)]
    j.require(len(names) < authority.MAX_HANDOFFS, 'initial recovery history is full; preserve it')
    for name in names:
        j.require(name.endswith('.json'), 'unknown initial recovery evidence')
        other = read_record(runtime, name[len(PREFIX):-5])
        j.require(other is not None and other['plan']['reservation_sha256'] != plan['reservation_sha256'],
                  'another recovery already owns this initial reservation')
    record = {'format': FORMAT, 'id': recovery, 'plan': copy.deepcopy(plan), 'verified': None}
    write_record(runtime, record)
    return record


def verify(runtime, store, lease, recovery, observation):
    record = read_record(runtime, recovery)
    j.require(record is not None, 'initial cleanup has no durable prepared intent')
    closure, _ = validate_context(runtime, store, lease, record['plan'])
    j.recovered(observation, record['plan'])
    j.require(observation['closure_receipt_sha256'] == closure['sha256'], 'initial closure receipt changed')
    if record['verified'] is not None:
        j.require(record['verified'] == observation, 'initial verified replay changed')
        sync_record(runtime, recovery)
        return record
    record['verified'] = copy.deepcopy(observation)
    write_record(runtime, record)
    return record


def step(runtime, store, lease, adapter, recovery):
    """Prepare, then resync and clean up on retry; completed replay has no effects."""
    runtime._canonical(store, lease)
    filename(recovery)
    with runtime.transaction():
        record = read_record(runtime, recovery)
        if record is not None and record['verified'] is not None:
            sync_record(runtime, recovery)
            return record
        expected, _, spec = context(runtime, store, lease)
        if record is None:
            members = adapter.prepare_initial(copy.deepcopy(expected), copy.deepcopy(spec))
            return prepare(runtime, store, lease, recovery, dict(expected, members=members))
        validate_context(runtime, store, lease, record['plan'])
        prepare(runtime, store, lease, recovery, record['plan'])
        observation = adapter.recover_initial(copy.deepcopy(record['plan']), copy.deepcopy(spec))
        return verify(runtime, store, lease, recovery, observation)
