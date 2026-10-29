"""Explicit, restartable creation of a fresh shared desktop authority.

All mutation requires the canonical coordinator lease and an explicit attempt ID.
Unbound staging objects are retained, never adopted or removed. Only a fully
recorded inode can be published to a fixed name, with atomic no-replace rename.
This does not repair historical partial authorities or change desktop sessions.
"""
import ctypes
import errno
import os
from pathlib import Path
import uuid

import artifacts
import journal as j
import runtime_authority as authority

FORMAT = 1
RECORD = 'authority-provision.json'
CLAIM = authority.ANCHOR + '.provision.json'
KINDS = ('claim', 'root', 'seal', 'mirror')
MAX_ATTEMPTS = 64


def ref_shape(value, kind):
    j.exact(value, ('path', 'device', 'inode', 'owner', 'kind'))
    j.absolute(value['path']);j.require(value['kind'] == kind)
    j.require(all(type(value[key]) is int and value[key] >= 0 for key in ('device', 'inode', 'owner')))


def target(record, kind):
    runtime = Path(record['context']['directory']['path'])
    return {'claim': runtime / CLAIM, 'root': runtime / authority.ANCHOR,
            'seal': runtime / authority.ANCHOR_SEAL,
            'mirror': Path(record['coordinator']['path']) / authority.MIRROR}[kind]


def shape(value):
    j.exact(value, ('format', 'id', 'authority', 'context', 'coordinator', 'coordinator_lock', 'attempts', 'active', 'complete'))
    j.require(type(value['format']) is int and value['format'] == FORMAT, 'unknown provisioning attempt; preserve it')
    j.token(value['id']);j.token(value['authority']);ref_shape(value['coordinator'], 'directory');ref_shape(value['coordinator_lock'], 'file')
    j.exact(value['context'], ('directory', 'socket', 'server'))
    ref_shape(value['context']['directory'], 'directory');ref_shape(value['context']['socket'], 'socket');authority.admission.process(value['context']['server'])
    j.require(type(value['attempts']) is list and len(value['attempts']) <= MAX_ATTEMPTS)
    j.exact(value['active'], KINDS);j.require(type(value['complete']) is bool)
    paths = []
    for index, row in enumerate(value['attempts']):
        j.exact(row, ('kind', 'path', 'object', 'descriptor', 'published', 'abandoned'))
        j.require(row['kind'] in KINDS and type(row['published']) is bool and type(row['abandoned']) is bool)
        path = j.absolute(row['path']);prefix = '.omabit-provision-' + value['id'] + '-' + row['kind'] + '-' + str(index) + '-'
        j.require(path.parent == target(value, row['kind']).parent and path.name.startswith(prefix))
        j.token(path.name[len(prefix):]);paths.append(str(path))
        if row['object'] is None:
            j.require(not row['published'] and row['descriptor'] is None)
        else:
            ref_shape(row['object'], 'directory' if row['kind'] == 'root' else 'file')
            j.require(row['object']['path'] == str(path) and not row['abandoned'])
            if row['kind'] == 'root':
                d = row['descriptor'];j.exact(d, ('format', 'id', 'root', 'context', 'coordinator', 'lock', 'coordinator_lock'))
                j.require(type(d['format']) is int and d['format'] == authority.FORMAT and d['id'] == value['authority'] and d['context'] == value['context'] and
                          d['coordinator'] == value['coordinator'] and d['coordinator_lock'] == value['coordinator_lock'])
                ref_shape(d['root'], 'directory');ref_shape(d['lock'], 'file')
                j.require(d['root'] == dict(row['object'], path=str(target(value, 'root'))) and
                          d['lock']['path'] == str(target(value, 'root') / 'authority.lock'))
            else:j.require(row['descriptor'] is None)
        if row['abandoned']:j.require(row['object'] is None and value['active'][row['kind']] != index)
    j.require(len(paths) == len(set(paths)), 'provisioning stage name was reused')
    for kind, index in value['active'].items():
        if index is not None:
            j.require(type(index) is int and 0 <= index < len(value['attempts']))
            j.require(value['attempts'][index]['kind'] == kind and not value['attempts'][index]['abandoned'])
    j.require(all(row['abandoned'] or value['active'][row['kind']] == index for index, row in enumerate(value['attempts'])))
    if value['complete']:j.require(all(active(value, kind) is not None and active(value, kind)['published'] for kind in KINDS))
    return value


def active(record, kind):
    index = record['active'][kind]
    return None if index is None else record['attempts'][index]


def read(store):
    try:value = authority.read(store.fd, RECORD)
    except FileNotFoundError:return None
    return shape(value)


def save(store, value):
    shape(value)
    name = '.provision-pending-' + uuid.uuid4().hex
    authority.write_new(store.fd, name, value)
    os.replace(name, RECORD, src_dir_fd=store.fd, dst_dir_fd=store.fd)
    os.fsync(store.fd)


def sync(path):
    fd = os.open(path, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK | os.O_CLOEXEC)
    try:os.fsync(fd)
    finally:os.close(fd)


def rename_no_replace(parent, source, destination):
    """Linux renameat2: an existing destination is never removed or replaced."""
    library = ctypes.CDLL(None, use_errno=True)
    function = getattr(library, 'renameat2', None)
    j.require(function is not None, 'atomic no-replace publication is unavailable')
    function.argtypes = [ctypes.c_int, ctypes.c_char_p, ctypes.c_int, ctypes.c_char_p, ctypes.c_uint]
    function.restype = ctypes.c_int
    if function(parent, os.fsencode(source), parent, os.fsencode(destination), 1) != 0:
        number = ctypes.get_errno()
        if number == errno.EEXIST:raise j.Conflict('fixed authority target already exists; preserve it')
        raise OSError(number, os.strerror(number))
    os.fsync(parent)


def environment(record, display, store, lease):
    j.require(authority.lease_check(store, lease) == record['coordinator'], 'another coordinator owns provisioning')
    authority.check_object(record['coordinator_lock'], 'file')
    j.require(str(j.absolute(str(display))) == record['context']['socket']['path'], 'another display was requested')
    authority.check_context(record['context'])


def body(record, kind):
    if kind == 'claim':
        return {'format': FORMAT, 'attempt': record['id'], 'authority': record['authority'],
                'context': record['context'], 'coordinator': record['coordinator'], 'coordinator_lock': record['coordinator_lock']}
    root = active(record, 'root');j.require(root is not None and root['descriptor'] is not None)
    return root['descriptor']


def moved(row, path):
    j.require(row is not None and row['object'] is not None, 'unrecorded authority object cannot be adopted')
    reference = dict(row['object'], path=str(path))
    authority.check_object(reference, reference['kind'])


def check_file(record, kind, path):
    row = active(record, kind);moved(row, path)
    observed, _ = authority.read_private_json(path)
    j.require(observed == body(record, kind), 'published authority evidence differs from original intent')


def check_stage(record):
    row = active(record, 'root');path = Path(row['path']);moved(row, path);d = row['descriptor']
    fd = artifacts.directory(path)
    try:
        j.require(set(os.listdir(fd)) == {'authority.lock', 'authority.json', 'reservation.json', 'ready.json'}, 'prepared authority stage changed')
        actual_lock = authority.object_identity(path / 'authority.lock', 'file')
        j.require(dict(actual_lock, path=str(target(record, 'root') / 'authority.lock')) == d['lock'])
        j.require(authority.read(fd, 'authority.lock') == {})
        j.require(authority.read(fd, 'authority.json') == d)
        j.require(authority.read(fd, 'reservation.json') == {'format': authority.FORMAT, 'authority': record['authority'], 'reservation': None})
        j.require(authority.read(fd, 'ready.json') == {'format': authority.FORMAT, 'descriptor': j.digest(d)})
    finally:os.close(fd)


def allocate(record, kind, store):
    old = active(record, kind)
    if old is not None:
        j.require(old['object'] is None and not old['published'], 'bound stages cannot be replaced')
        old['abandoned'] = True  # Preserve its path without opening, moving or deleting it.
    j.require(len(record['attempts']) < MAX_ATTEMPTS, 'provisioning attempt history is full; preserve it')
    index = len(record['attempts'])
    path = target(record, kind).parent / ('.omabit-provision-' + record['id'] + '-' + kind + '-' + str(index) + '-' + uuid.uuid4().hex)
    row = {'kind': kind, 'path': str(path), 'object': None, 'descriptor': None, 'published': False, 'abandoned': False}
    record['attempts'].append(row);record['active'][kind] = index;save(store, record)
    if kind == 'root':
        path.mkdir(mode=0o700);sync(path.parent);fd = artifacts.directory(path)
        try:
            authority.write_new(fd, 'authority.lock', {})
            reference = authority.object_identity(path, 'directory')
            d = {'format': authority.FORMAT, 'id': record['authority'], 'root': dict(reference, path=str(target(record, kind))),
                 'context': record['context'], 'coordinator': record['coordinator'],
                 'lock': dict(authority.object_identity(path / 'authority.lock', 'file'), path=str(target(record, kind) / 'authority.lock')),
                 'coordinator_lock': record['coordinator_lock']}
            authority.write_new(fd, 'authority.json', d)
            authority.write_new(fd, 'reservation.json', {'format': authority.FORMAT, 'authority': record['authority'], 'reservation': None})
            authority.write_new(fd, 'ready.json', {'format': authority.FORMAT, 'descriptor': j.digest(d)})
            row['object'], row['descriptor'] = reference, d
        finally:os.close(fd)
        check_stage(record)
    else:
        parent = artifacts.directory(path.parent)
        try:authority.write_new(parent, path.name, body(record, kind))
        finally:os.close(parent)
        row['object'] = authority.object_identity(path, 'file');check_file(record, kind, path)
    save(store, record)


def publish(record, kind, store):
    row = active(record, kind);source = Path(row['path']);destination = target(record, kind)
    if os.path.lexists(destination):
        moved(row, destination)
        j.require(not os.path.lexists(source), 'both staged and published authority objects exist')
        if kind != 'root':check_file(record, kind, destination)
    else:
        j.require(not row['published'], 'previously published authority evidence is missing')
        if kind == 'root':check_stage(record)
        else:check_file(record, kind, source)
        parent = artifacts.directory(destination.parent)
        try:rename_no_replace(parent, source.name, destination.name)
        finally:os.close(parent)
        moved(row, destination)
    sync(destination);sync(destination.parent)
    row['published'] = True;save(store, record)


def summary(record, phase):
    return {'format': FORMAT, 'attempt': record['id'], 'authority': record['authority'],
            'root': str(target(record, 'root')), 'phase': phase, 'complete': record['complete']}


def step(display, store, lease, attempt):
    """Persist one bounded setup step; retry only the same explicit intent."""
    coordinator = authority.lease_check(store, lease);j.token(attempt)
    record = read(store)
    if record is None:
        j.require(store.status() == j.initial(), 'initial provisioning requires a fresh empty journal')
        context = authority.context(display);runtime = Path(context['directory']['path'])
        j.require(not any(os.path.lexists(runtime / name) for name in (authority.ANCHOR, authority.ANCHOR_SEAL, CLAIM)),
                  'existing runtime evidence is not fresh provisioning')
        j.require(not os.path.lexists(store.root / authority.MIRROR), 'coordinator already has runtime authority evidence')
        record = {'format': FORMAT, 'id': attempt, 'authority': uuid.uuid4().hex, 'context': context,
                  'coordinator': coordinator, 'coordinator_lock': authority.object_identity(store.root / 'coordinator.lock', 'file'),
                  'attempts': [], 'active': {kind: None for kind in KINDS}, 'complete': False}
        # Fresh Store objects can represent an empty history in memory. A second
        # controller must be able to reopen that exact empty history after death.
        if 'journal.json' not in os.listdir(store.fd):
            j.require(store.fresh, 'missing prior journal cannot be recreated')
            authority.write_new(store.fd, 'journal.json', {'format': 2, 'events': []})
            store.fresh = False
        save(store, record);return summary(record, 'planned')
    j.require(record['id'] == attempt, 'another provisioning attempt already owns this coordinator')
    environment(record, display, store, lease)
    root = target(record, 'root');prepared = active(record, 'root')
    if os.path.lexists(root):
        moved(prepared, root)
        j.require(not os.path.lexists(prepared['path']), 'staged root remains after publication')
        for kind in ('claim', 'seal', 'mirror'):
            row = active(record, kind);j.require(row is not None and row['published'])
            check_file(record, kind, target(record, kind))
        opened = authority.Authority(root)
        try:j.require(opened.descriptor == prepared['descriptor'], 'completed authority differs from provisioning intent')
        finally:opened.close()
        sync(root);sync(root.parent)
        if not record['complete']:
            prepared['published'] = True;record['complete'] = True;save(store, record)
        else:sync(store.root / RECORD);sync(store.root)
        return summary(record, 'complete')
    j.require(not record['complete'] and (prepared is None or not prepared['published']), 'published authority is missing; never recreate it')
    j.require(store.status() == j.initial(), 'unfinished provisioning cannot reset changed journal state')
    for kind in ('claim', 'seal', 'mirror'):
        row = active(record, kind)
        if row is not None and row['published']:check_file(record, kind, target(record, kind))
    claim = active(record, 'claim')
    if claim is None or claim['object'] is None:
        allocate(record, 'claim', store);return summary(record, 'claim-prepared')
    if not claim['published']:
        publish(record, 'claim', store);return summary(record, 'claim-published')
    if prepared is None or prepared['object'] is None:
        allocate(record, 'root', store);return summary(record, 'root-prepared')
    check_stage(record)
    for kind in ('seal', 'mirror'):
        row = active(record, kind)
        if row is None or row['object'] is None:
            allocate(record, kind, store);return summary(record, kind + '-prepared')
        if not row['published']:
            publish(record, kind, store);return summary(record, kind + '-published')
    publish(record, 'root', store)
    return summary(record, 'root-published')


def provision(display, store, lease, attempt):
    """Finish or resume this one attempt; never choose an intent implicitly."""
    for _ in range(16):
        result = step(display, store, lease, attempt)
        if result['complete']:return authority.Authority(result['root'])
    raise j.Conflict('provisioning exceeded its bounded step count; retain and inspect the attempt')
