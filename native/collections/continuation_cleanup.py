"""Explicit transport cleanup for a committed checkpoint2 continuation authority after desktop loss.

The canonical lease serializes a separate prepared/verified receipt. The original
runtime evidence remains in place and no successor admission is created. A bound
adapter must inspect actual plugin cohorts and execute transport-only cleanup.
"""
import copy
import os
from pathlib import Path
import uuid
import continuation_inspection as inspection
from authority_inspection import process_state
import journal as j
import runtime_authority as authority

FORMAT = 3
PREFIX = 'desktop-chain-cleanup-'
MAX_RECORDS = 128


def filename(recovery):
    j.token(recovery);return PREFIX + recovery + '.json'


def plan_shape(value):
    j.exact(value, ('authority', 'descriptor_sha256', 'reservation_sha256', 'evidence_sha256', 'journal',
                   'specification', 'loaded', 'guardian', 'closure_request', 'closure_receipt_sha256', 'members'))
    j.token(value['authority']);j.token(value['closure_request'])
    for name in ('descriptor_sha256', 'reservation_sha256', 'evidence_sha256', 'closure_receipt_sha256'):j.token(value[name], j.HASH)
    j.exact(value['journal'], ('events', 'sha256'));j.token(value['journal']['sha256'], j.HASH)
    j.require(type(value['journal']['events']) is int and 0 < value['journal']['events'] <= j.MAX_EVENTS)
    j.specification(value['specification']);j.loaded(value['loaded'], value['specification'], value['loaded']['epoch'])
    j.guardian_identity(value['guardian']);j.exact(value['members'], j.ROLES)
    for role in j.ROLES:j.recovery_member(value['members'][role], value['loaded']['members'][role])
    return value


def record_shape(value):
    j.exact(value, ('format', 'id', 'plan', 'verified'))
    j.require(type(value['format']) is int and value['format'] == FORMAT, 'unknown desktop cleanup record; preserve it')
    j.token(value['id']);plan_shape(value['plan'])
    if value['verified'] is not None:
        j.recovered(value['verified'], value['plan'])
        j.require(value['verified']['closure_receipt_sha256'] == value['plan']['closure_receipt_sha256'])
    return value


def read_record(store, recovery):
    try:value = authority.read(store.fd, filename(recovery))
    except FileNotFoundError:return None
    record_shape(value);j.require(value['id'] == recovery);return value


def sync_record(store, recovery):
    fd = os.open(filename(recovery), os.O_RDONLY | os.O_NOFOLLOW | os.O_CLOEXEC, dir_fd=store.fd)
    try:os.fsync(fd)
    finally:os.close(fd)
    os.fsync(store.fd)


def save_record(store, value):
    record_shape(value);name = '.pending-desktop-chain-cleanup-' + value['id'] + '-' + uuid.uuid4().hex
    authority.write_new(store.fd, name, value)
    os.replace(name, filename(value['id']), src_dir_fd=store.fd, dst_dir_fd=store.fd);os.fsync(store.fd)


def context(root, store, lease):
    authority.lease_check(store, lease)
    snapshot = inspection.inspect(root, store.root, lease=lease)
    j.require(snapshot['format'] == 3 and snapshot['descriptor']['format'] == 3, 'continuation cleanup requires authority3; use the matching historical cleanup for earlier authorities')
    j.require(snapshot['context']['server'] == 'gone', 'original compositor must be gone in the observable process domain')
    state = snapshot['reservation'];bound = state.get('binding');original = state['reservation']
    j.require(bound is not None and snapshot['journal']['pending'] is None and
              snapshot['journal']['committed'] == bound['loaded'], 'desktop cleanup requires a bound committed collection without a pending switch')
    closure = snapshot['activations'][-1]['closure']
    j.require(closure['verified_process_closure'], 'original complete guardian closure is required')
    j.require(all(process_state(row['owner']) == 'gone' for row in snapshot['activations'][-1]['admission']['claims'].values()),
              'an original admission owner remains live or unobservable')
    receipt, digest = authority.read_private_json(original['owner']['receipt'])
    j.require(digest == closure['receipt_sha256'])
    evidence = {key: snapshot[key] for key in ('descriptor', 'reservation', 'journal', 'activations', 'files', 'missing_files', 'lineage')}
    return {'authority': snapshot['authority'], 'descriptor_sha256': j.digest(snapshot['descriptor']),
            'reservation_sha256': j.digest(original), 'evidence_sha256': j.digest(evidence),
            'journal': {key: snapshot['journal'][key] for key in ('events', 'sha256')},
            'specification': copy.deepcopy(original['permit']['specification']), 'loaded': copy.deepcopy(bound['loaded']),
            'guardian': copy.deepcopy(original['owner']['guardian']), 'closure_request': receipt['request'], 'closure_receipt_sha256': digest}


def validate_context(root, store, lease, plan):
    plan_shape(plan);expected = context(root, store, lease)
    j.require(all(plan[key] == value for key, value in expected.items()), 'original desktop authority, history or closure changed')
    return expected


def step(root, store, lease, adapter, recovery):
    """Prepare first, then repeatable cleanup; verified replay dispatches nothing."""
    filename(recovery);expected = context(root, store, lease);old = read_record(store, recovery)
    if old is not None:
        validate_context(root, store, lease, old['plan']);sync_record(store, recovery)
        if old['verified'] is not None:return old
        observation = adapter.recover_desktop(copy.deepcopy(old['plan']))
        validate_context(root, store, lease, old['plan']);j.recovered(observation, old['plan'])
        j.require(observation['closure_receipt_sha256'] == old['plan']['closure_receipt_sha256'], 'original guardian closure changed during cleanup')
        value = dict(old, verified=copy.deepcopy(observation));save_record(store, value);return value
    names = [name for name in os.listdir(store.fd) if name.startswith(PREFIX)]
    j.require(len(names) < MAX_RECORDS, 'desktop cleanup history is full; preserve it')
    for name in names:
        j.require(name.endswith('.json'), 'unknown desktop cleanup evidence')
        other = read_record(store, name[len(PREFIX):-5]);j.require(other is not None)
        j.require((other['plan']['authority'], other['plan']['reservation_sha256']) != (expected['authority'], expected['reservation_sha256']),
                  'another cleanup intent owns this original activation')
    members = adapter.prepare_desktop(copy.deepcopy(expected))
    plan = dict(expected, members=copy.deepcopy(members));validate_context(root, store, lease, plan)
    value = {'format': FORMAT, 'id': recovery, 'plan': plan, 'verified': None};save_record(store, value);return value
