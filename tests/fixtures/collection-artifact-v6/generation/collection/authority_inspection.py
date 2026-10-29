"""Read retained runtime evidence without constructing an active Authority.

No admission, cleanup, process control or repair is exposed. A coherent snapshot
can describe a dead compositor but never authorizes successor activation. Original
private filesystem objects must still exist; copied archives are not live evidence.
"""
import contextlib
import copy
import fcntl
import hashlib
import os
from pathlib import Path
import stat
from types import SimpleNamespace

import admission
import artifacts
import journal as j
import runtime_authority as authority
import shared_admission


def reference_shape(value, kind):
    j.exact(value, ('path', 'device', 'inode', 'owner', 'kind'))
    j.absolute(value['path']);j.require(value['kind'] == kind)
    j.require(all(type(value[key]) is int and value[key] >= 0 for key in ('device', 'inode', 'owner')))


class Reader:
    def __init__(self):self.files = {};self.absent = set();self.directories = {}

    def directory(self, reference):
        reference_shape(reference, 'directory');authority.check_object(reference, 'directory')
        self.directories[reference['path']] = copy.deepcopy(reference)

    def read(self, path, *, optional=False, maximum=authority.MAX_BYTES):
        path = j.absolute(str(path))
        try:reference = authority.object_identity(path, 'file')
        except FileNotFoundError:
            if optional:self.absent.add(str(path));return None
            raise
        j.require(str(path) not in self.absent, 'retained evidence appeared during inspection')
        fd = os.open(path, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK | os.O_CLOEXEC)
        with os.fdopen(fd, 'rb') as stream:
            before = os.fstat(stream.fileno())
            j.require(stat.S_ISREG(before.st_mode) and before.st_nlink == 1 and stat.S_IMODE(before.st_mode) == 0o600 and
                      (before.st_dev, before.st_ino, before.st_uid) == tuple(reference[k] for k in ('device', 'inode', 'owner')) and
                      before.st_size <= maximum, 'unsafe or replaced retained evidence')
            raw = stream.read(maximum + 1);after = os.fstat(stream.fileno())
            j.require(len(raw) == before.st_size and
                      (before.st_mtime_ns, before.st_ctime_ns, before.st_nlink) == (after.st_mtime_ns, after.st_ctime_ns, after.st_nlink),
                      'retained evidence changed during inspection')
        authority.check_object(reference, 'file')
        row = {'object': reference, 'sha256': hashlib.sha256(raw).hexdigest(), 'maximum': maximum}
        old = self.files.get(str(path));j.require(old is None or old == row, 'retained evidence changed between reads')
        self.files[str(path)] = row
        return artifacts.object_json(raw)

    def verify(self):
        for reference in self.directories.values():authority.check_object(reference, 'directory')
        for path in self.absent:j.require(not os.path.lexists(path), 'missing evidence appeared during inspection')
        for path, row in list(self.files.items()):self.read(path, maximum=row['maximum'])


@contextlib.contextmanager
def read_lock(reference):
    reference_shape(reference, 'file');authority.check_object(reference, 'file')
    fd = os.open(reference['path'], os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK | os.O_CLOEXEC)
    try:
        held = os.fstat(fd)
        j.require((held.st_dev, held.st_ino) == (reference['device'], reference['inode']), 'retained lock replaced while opening')
        fcntl.flock(fd, fcntl.LOCK_SH | fcntl.LOCK_NB)
        authority.check_object(reference, 'file')
        yield
        authority.check_object(reference, 'file')
    finally:os.close(fd)


def process_state(value):
    """Never treat a PID from another process domain as locally absent."""
    admission.process(value)
    current = admission.identity(os.getpid());j.require(current is not None)
    if value['boot'] != current['boot']:return 'previous-boot'
    if value['namespace'] != current['namespace']:return 'unobservable'
    return 'live' if admission.identity(value['pid']) == value else 'gone'


def context_state(value):
    j.exact(value, ('directory', 'socket', 'server'))
    reference_shape(value['directory'], 'directory');reference_shape(value['socket'], 'socket')
    j.require(Path(value['socket']['path']).parent == Path(value['directory']['path']))
    server = process_state(value['server'])
    try:
        authority.check_object(value['directory'], 'directory');authority.check_object(value['socket'], 'socket');objects = 'original'
    except FileNotFoundError:objects = 'missing'
    except (OSError, ValueError):objects = 'changed-or-unreadable'
    return {'server': server, 'filesystem': objects, 'live_context': server == 'live' and objects == 'original'}


def binding(reader, value, reservation, recorded_loaded):
    if value is None:return None
    authority.binding_shape(value, reservation)
    reference = value['runtime'];authority.check_object(reference['object'], 'file')
    body = reader.read(reference['object']['path'])
    j.require(reader.files[reference['object']['path']]['object'] == reference['object'] and
              reader.files[reference['object']['path']]['sha256'] == reference['sha256'], 'retained runtime observation changed')
    j.exact(body, ('loaded', 'helpers', 'shell', 'supervisor', 'admission'))
    j.require(body['loaded'] == value['loaded'] and j.digest(value['loaded']) in recorded_loaded,
              'binding is absent from canonical collection history')
    permit = reservation['permit']
    j.require(body['admission'] == {'root': reservation['gate']['path'], 'permit': permit})
    j.require(authority.process_pair(body['shell']) == authority.process_pair(permit['shell']))
    j.require(all(body['supervisor'][key] == item for key, item in value['guardian']['owner'].items()))
    j.exact(body['helpers'], j.ROLES)
    for role, claim in value['claims'].items():
        helpers = body['helpers'][role];j.require(type(helpers) is list and 0 < len(helpers) <= 256)
        pairs = [authority.process_pair(helper) for helper in helpers]
        j.require(authority.process_pair(claim['owner']) in pairs and len({(x['pid'], x['start']) for x in pairs}) == len(pairs))
    return body


def gate(reader, descriptor, reservation, bound):
    reader.directory(reservation['gate']);root = Path(reservation['gate']['path'])
    row = reader.read(root/'admission.json', optional=True);seal = reader.read(root/'issued.json', optional=True)
    ticket = {'format': 1, 'root': descriptor['root']['path'], 'authority': descriptor['id'], 'reservation': j.digest(reservation)}
    expected = {'permit': reservation['permit'], 'authority': ticket}
    if row is not None:
        shared_admission.validate(row)
        j.require(all(row[k] == v for k, v in expected.items()), 'retained gate belongs to another reservation')
    if seal is not None:j.require(seal == shared_admission.Gate._seal_value(expected), 'retained issuance seal changed')
    if bound is not None:
        j.require(row is not None and seal is not None and set(row['claims']) == set(j.ROLES), 'bound activation lost its complete admission')
        for role, claim in row['claims'].items():
            j.require(all(claim[k] == bound['claims'][role][k] for k in ('owner', 'instance')), 'bound admission owner changed')
    return {'missing': [name for name, value in [('admission.json', row), ('issued.json', seal)] if value is None],
            'state': row['state'] if row else None,
            'claims': copy.deepcopy(row['claims']) if row else {}}


def closure(reader, reservation):
    owner = reservation.get('owner')
    if owner is None:return {'state': 'unowned-legacy', 'verified_process_closure': False}
    guardian = owner['guardian'];receipt = reader.read(owner['receipt'], optional=True)
    shell = process_state(reservation['permit']['shell']);supervisor = process_state(authority.guardian_process(guardian))
    if receipt is None:return {'state': 'receipt-missing', 'shell': shell, 'guardian': supervisor, 'verified_process_closure': False}
    authority.check_guardian_receipt(receipt, guardian)
    completed = receipt['phase'] == 'closed' and receipt['proof'] == 'waitpid-echild'
    if completed:
        j.token(receipt['request']);j.require(any(x.get('pid') == guardian['root']['pid'] for x in receipt['reaped']), 'closure omits original shell')
        j.require(shell != 'live' and supervisor != 'live', 'closed receipt contradicts a live original owner')
    return {'state': 'closed' if completed else 'unconfirmed', 'shell': shell, 'guardian': supervisor,
            'receipt_sha256': reader.files[owner['receipt']]['sha256'],
            'verified_process_closure': completed and shell == supervisor == 'gone'}


def inspect(root, coordinator, *, lease=None):
    """Return a coherent read-only snapshot; never instantiate active authority."""
    root = j.absolute(str(root));coordinator = j.absolute(str(coordinator))
    j.require(root.name == authority.ANCHOR, 'explicit canonical authority root required')
    reader = Reader();descriptor = reader.read(root/'authority.json')
    j.exact(descriptor, ('format', 'id', 'root', 'context', 'coordinator', 'lock', 'coordinator_lock'))
    j.require(type(descriptor['format']) is int and descriptor['format'] == authority.FORMAT, 'unknown retained authority format')
    j.token(descriptor['id']);j.require(descriptor['root']['path'] == str(root) and descriptor['coordinator']['path'] == str(coordinator))
    reader.directory(descriptor['root']);reader.directory(descriptor['coordinator'])
    j.require(Path(descriptor['context']['directory']['path'])/authority.ANCHOR == root)
    j.require(descriptor['lock']['path'] == str(root/'authority.lock') and descriptor['coordinator_lock']['path'] == str(coordinator/'coordinator.lock'))
    if lease is not None:
        j.require(str(lease.store.root) == str(coordinator) and
                  authority.lease_check(lease.store, lease) == descriptor['coordinator'], 'inspection lease belongs to another coordinator')
        authority.check_object(descriptor['coordinator_lock'], 'file')
    coordinator_lock = contextlib.nullcontext() if lease is not None else read_lock(descriptor['coordinator_lock'])
    with coordinator_lock, read_lock(descriptor['lock']):
        before_context = context_state(descriptor['context'])
        j.require(reader.read(root/'authority.json') == descriptor)
        j.require(reader.read(root/'ready.json') == {'format': 1, 'descriptor': j.digest(descriptor)}, 'retained ready seal changed')
        j.require(reader.read(root.parent/authority.ANCHOR_SEAL) == descriptor and reader.read(coordinator/authority.MIRROR) == descriptor,
                  'retained authority mirrors disagree')
        reader.read(root/'authority.lock')
        body = reader.read(coordinator/'journal.json', maximum=j.MAX_BYTES)
        store = j.Store(coordinator)
        try:events, state = store._load()
        finally:store.close()
        j.require(body['events'] == events)
        replay = j.initial();recorded_loaded = set()
        for event in events:
            j.require(j.apply(replay, event))
            for value in [replay['committed']] + [r[k] for r in replay['requests'].values() for k in ('previous', 'loaded')]:
                if value is not None:recorded_loaded.add(j.digest(value))
        value = reader.read(root/'reservation.json')
        # These two existing validators only parse/read. The lightweight holder
        # deliberately has none of Authority's admission or mutation methods.
        fd = artifacts.directory(root)
        try:
            holder = SimpleNamespace(fd=fd, root=root, descriptor=descriptor)
            j.require(authority.Authority._read_reservation(holder) == value)
            authority.Authority._history(holder, value, events)
        finally:os.close(fd)
        rows = list(value.get('history', []))
        if value['reservation'] is not None:rows.append({'reservation': value['reservation'], 'binding': value.get('binding')})
        observations = []
        for entry in rows:
            reservation = entry['reservation'];spec = reservation['permit']['specification']
            j.require(state['collections'].get(j.digest(spec)) == spec, 'retained reservation registration changed')
            observed = binding(reader, entry['binding'], reservation, recorded_loaded)
            admission_row = gate(reader, descriptor, reservation, entry['binding'])
            owner_closure = closure(reader, reservation)
            if 'handoff' in entry:
                proof = entry['handoff'];reference = proof.get('initial_recovery')
                if reference is not None:
                    reader.read(reference['object']['path'])
                    j.require(reader.files[reference['object']['path']]['sha256'] == reference['sha256'])
                if proof['closure'] is not None:
                    row = proof['closure'];reader.read(row['path'])
                    j.require(reader.files[row['path']]['sha256'] == row['sha256'], 'historical closure changed')
            observations.append({'reservation_sha256': j.digest(reservation), 'admission': admission_row, 'closure': owner_closure,
                                 'loaded': entry['binding']['loaded'] if entry['binding'] else None,
                                 'runtime_observation_sha256': entry['binding']['runtime']['sha256'] if observed else None})
        reader.verify();j.require(context_state(descriptor['context']) == before_context, 'compositor identity changed during inspection')
        authority.check_object(descriptor['coordinator_lock'], 'file')
        if lease is not None:j.require(authority.lease_check(lease.store, lease) == descriptor['coordinator'])
        return {'format': 1, 'mode': 'retained-evidence-only', 'authority': descriptor['id'], 'context': before_context,
                'descriptor': descriptor, 'reservation': value, 'journal': {'format': body['format'], 'events': len(events), 'sha256': j.digest(events),
                'committed': state['committed'], 'pending': state['pending']}, 'activations': observations,
                'files': {path: {k: row[k] for k in ('object', 'sha256')} for path, row in reader.files.items()},
                'missing_files': sorted(reader.absent), 'activation_authorized': False, 'transport_cleanup_verified': False}
