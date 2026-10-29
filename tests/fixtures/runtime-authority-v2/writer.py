"""Explicit shared desktop reservation, independent of per-profile XDG roots.

Version2 reservation history binds the original live runtime and permits handoff
only after canonical journal cleanup evidence. Vanished processes and ended guards
alone cannot establish cleanup. Provisioning remains a one-time explicit operation
for a fresh runtime/journal, never an automatic repair.
"""
import contextlib
import copy
import fcntl
import hashlib
import os
from pathlib import Path
import socket
import stat
import struct
import uuid

import admission as admission
import artifacts as artifacts
import journal as journal

FORMAT = 1
RESERVATION_FORMAT = 2
MAX_HANDOFFS = 128
ANCHOR = '.omabit-collection-authority'
ANCHOR_SEAL = ANCHOR + '.json'
MIRROR = 'runtime-authority.json'
MAX_BYTES = 4 << 20


def object_identity(path, kind):
    path = journal.absolute(str(path))
    parent = artifacts.directory(path.parent)
    try:
        info = os.stat(path.name, dir_fd=parent, follow_symlinks=False)
    finally:
        os.close(parent)
    predicate = {'directory': stat.S_ISDIR, 'socket': stat.S_ISSOCK,
                 'file': stat.S_ISREG}[kind]
    journal.require(predicate(info.st_mode) and info.st_uid == os.geteuid(),
                    'runtime resource has changed owner or type')
    if kind == 'directory':
        journal.require(stat.S_IMODE(info.st_mode) == 0o700, 'private runtime directory required')
    if kind == 'file':
        journal.require(info.st_nlink == 1 and stat.S_IMODE(info.st_mode) == 0o600,
                        'private unlinked runtime file required')
    return {'path': str(path), 'device': info.st_dev, 'inode': info.st_ino,
            'owner': info.st_uid, 'kind': kind}


def check_object(value, kind):
    journal.exact(value, ('path', 'device', 'inode', 'owner', 'kind'))
    journal.require(value['kind'] == kind)
    for name in ('device', 'inode', 'owner'):
        journal.require(type(value[name]) is int and value[name] >= 0)
    journal.require(object_identity(value['path'], kind) == value,
                    'bound runtime filesystem object was replaced')


def context(path):
    """Bind an explicitly selected Unix display socket to its real peer owner."""
    path = journal.absolute(str(path))
    directory = object_identity(path.parent, 'directory')
    before = object_identity(path, 'socket')
    with socket.socket(socket.AF_UNIX, socket.SOCK_STREAM) as client:
        client.settimeout(2)
        client.connect(str(path))
        pid, uid, _ = struct.unpack('3i', client.getsockopt(socket.SOL_SOCKET, socket.SO_PEERCRED, 12))
    server = admission.identity(pid)
    journal.require(uid == os.geteuid() and server is not None, 'display peer is not a live owned process')
    value = {'directory': directory, 'socket': before, 'server': server}
    check_context(value)
    return value


def check_context(value):
    journal.exact(value, ('directory', 'socket', 'server'))
    check_object(value['directory'], 'directory')
    check_object(value['socket'], 'socket')
    journal.require(Path(value['socket']['path']).parent == Path(value['directory']['path']))
    admission.process(value['server'])
    journal.require(admission.identity(value['server']['pid']) == value['server'],
                    'display owner is gone; preserve prior reservation')


def read(fd, name):
    file = os.open(name, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK | os.O_CLOEXEC, dir_fd=fd)
    with os.fdopen(file, 'rb') as stream:
        info = os.fstat(stream.fileno())
        journal.require(stat.S_ISREG(info.st_mode) and info.st_nlink == 1 and
                        info.st_uid == os.geteuid() and stat.S_IMODE(info.st_mode) == 0o600 and
                        info.st_size <= MAX_BYTES, 'unsafe runtime authority evidence')
        data = stream.read(MAX_BYTES + 1)
        journal.require(len(data) <= MAX_BYTES)
    return artifacts.object_json(data)


def write_new(fd, name, value):
    data = journal.canonical(value) + b'\n'
    journal.require(len(data) <= MAX_BYTES)
    file = os.open(name, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW | os.O_CLOEXEC,
                   0o600, dir_fd=fd)
    with os.fdopen(file, 'wb') as stream:
        stream.write(data)
        stream.flush()
        os.fsync(stream.fileno())
    os.fsync(fd)


def lease_check(store, lease):
    journal.require(lease.store is store and lease.live and lease.pid == os.getpid(),
                    'live canonical coordinator lease required')
    bound = object_identity(store.root, 'directory')
    held = os.fstat(store.fd)
    journal.require((held.st_dev, held.st_ino) == (bound['device'], bound['inode']),
                    'coordinator directory was replaced while open')
    return bound


def process_pair(value):
    journal.require(type(value) is dict and 'pid' in value and 'start' in value)
    result = {key: value[key] for key in ('pid', 'start')}
    journal.process_identity(result)
    return result


def pair_live(value):
    pair = process_pair(value)
    live = admission.identity(pair['pid'])
    return live is not None and live['start'] == pair['start']


def parent_of(value):
    journal.require(admission.identity(value['pid']) == value, 'bound process is gone')
    row = (Path('/proc') / str(value['pid']) / 'stat').read_text().rsplit(') ', 1)[1].split()
    journal.require(admission.identity(value['pid']) == value, 'bound process changed')
    return int(row[1])


def guardian_process(value):
    return {key: value['owner'][key] for key in ('pid', 'start', 'boot', 'namespace')}


def read_private_json(path):
    path = journal.absolute(str(path))
    fd = artifacts.directory(path.parent)
    try:
        file = os.open(path.name, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK | os.O_CLOEXEC, dir_fd=fd)
    finally:
        os.close(fd)
    with os.fdopen(file, 'rb') as stream:
        info = os.fstat(stream.fileno())
        journal.require(stat.S_ISREG(info.st_mode) and info.st_nlink == 1 and info.st_uid == os.geteuid() and
                        stat.S_IMODE(info.st_mode) == 0o600 and info.st_size <= MAX_BYTES, 'unsafe runtime proof file')
        data = stream.read(MAX_BYTES + 1)
        after = os.fstat(stream.fileno())
        journal.require(len(data) == info.st_size and
                        (info.st_mtime_ns, info.st_ctime_ns) == (after.st_mtime_ns, after.st_ctime_ns),
                        'runtime proof changed while reading')
    return artifacts.object_json(data), hashlib.sha256(data).hexdigest()


def check_guardian_receipt(receipt, guardian):
    journal.exact(receipt, ('version', 'owner', 'phase', 'request', 'root', 'reaped', 'signals', 'proof'))
    journal.require(type(receipt['version']) is int and receipt['version'] == 1)
    journal.guardian_identity(guardian)
    journal.require({key: receipt['owner'][key] for key in guardian['owner']} == guardian['owner'] and
                    process_pair(receipt['root']) == guardian['root'], 'receipt is from another guardian or shell')
    journal.require(type(receipt['reaped']) is list and len(receipt['reaped']) <= 4096 and
                    type(receipt['signals']) is list and len(receipt['signals']) <= 16384)


def reservation_shape(value, *, with_owner=False):
    journal.exact(value, ('permit', 'gate', 'intent') + (('owner',) if with_owner else ()))
    admission.permit(value['permit'])
    journal.require(value['intent'] in ('initial', 'transition'))
    if with_owner:owner_shape(value['owner'], value['permit']['shell'])
    journal.exact(value['gate'], ('path', 'device', 'inode', 'owner', 'kind'))
    journal.absolute(value['gate']['path'])
    journal.require(value['gate']['kind'] == 'directory')
    for key in ('device', 'inode', 'owner'):
        journal.require(type(value['gate'][key]) is int and value['gate'][key] >= 0)


def binding_shape(value, reservation):
    journal.exact(value, ('runtime', 'loaded', 'guardian', 'guardian_receipt', 'claims'))
    journal.exact(value['runtime'], ('object', 'sha256'))
    journal.token(value['runtime']['sha256'], journal.HASH)
    journal.exact(value['runtime']['object'], ('path', 'device', 'inode', 'owner', 'kind'))
    journal.absolute(value['runtime']['object']['path'])
    journal.absolute(value['guardian_receipt'])
    journal.loaded(value['loaded'], reservation['permit']['specification'], reservation['permit']['epoch'])
    journal.guardian_identity(value['guardian'])
    journal.require(value['guardian']['root'] == process_pair(reservation['permit']['shell']))
    journal.require(value['guardian'] == reservation['owner']['guardian'] and value['guardian_receipt'] == reservation['owner']['receipt'],
                    'loaded binding changed original reservation ownership')
    journal.exact(value['claims'], journal.ROLES)
    for claim in value['claims'].values():
        journal.exact(claim, ('instance', 'owner', 'state'))
        journal.token(claim['instance'])
        admission.process(claim['owner'])
        journal.require(claim['state'] == 'admitted')


def owner_for_shell(shell, receipt_path):
    receipt_path = str(journal.absolute(str(receipt_path)))
    receipt, _ = read_private_json(receipt_path)
    guardian = {'owner': {key: receipt['owner'][key] for key in ('token', 'pid', 'start', 'boot', 'namespace')},
                'root': process_pair(shell)}
    value = {'guardian': guardian, 'receipt': receipt_path}
    owner_live(value, shell)
    return value


def owner_shape(owner, shell):
    journal.exact(owner, ('guardian', 'receipt'))
    journal.guardian_identity(owner['guardian'])
    journal.absolute(owner['receipt'])
    journal.require(owner['guardian']['root'] == process_pair(shell), 'guardian belongs to another shell')


def owner_live(owner, shell):
    owner_shape(owner, shell)
    guardian = owner['guardian']
    journal.require(admission.identity(shell['pid']) == shell and
                    admission.identity(guardian['owner']['pid']) == guardian_process(guardian) and
                    parent_of(shell) == guardian['owner']['pid'], 'original guardian no longer owns the live shell')
    receipt, _ = read_private_json(owner['receipt'])
    check_guardian_receipt(receipt, guardian)
    journal.require(receipt['phase'] == 'running' and receipt['proof'] is None,
                    'original guardian is no longer admitting work')



class Authority:
    """One fixed anchor per physical display directory; no profile defaults."""
    def __init__(self, root):
        self.root = journal.absolute(str(root))
        journal.require(self.root.name == ANCHOR, 'authority must use the canonical runtime anchor')
        self.fd = artifacts.directory(self.root)
        try:
            self.descriptor = read(self.fd, 'authority.json')
            d = self.descriptor
            journal.exact(d, ('format', 'id', 'root', 'context', 'coordinator', 'lock', 'coordinator_lock'))
            journal.require(type(d['format']) is int and d['format'] == FORMAT,
                            'unsupported runtime authority; preserve evidence')
            journal.token(d['id'])
            journal.require(Path(d['context']['directory']['path']) / ANCHOR == self.root)
            self._verify()
            journal.require(read(self.fd, 'ready.json') == {'format': FORMAT, 'descriptor': journal.digest(d)},
                            'authority provisioning is unfinished or changed')
            self._read_reservation()
        except BaseException:
            self.close()
            raise

    def close(self):
        if self.fd is not None:
            os.close(self.fd)
            self.fd = None

    @classmethod
    def provision(cls, display, store, lease):
        """Create once. Any partial or preexisting setup is retained and refused."""
        coordinator = lease_check(store, lease)
        state = store.status()
        journal.require(state == journal.initial(), 'initial provisioning needs a fresh empty journal')
        # Never treat a missing runtime anchor as permission to reset this journal.
        journal.require(MIRROR not in os.listdir(store.fd), 'journal already has runtime authority evidence')
        runtime = context(display)
        parent = artifacts.directory(Path(display).parent)
        try:
            journal.require(ANCHOR_SEAL not in os.listdir(parent),
                            'runtime retains authority evidence; missing anchors are not fresh setup')
            os.mkdir(ANCHOR, 0o700, dir_fd=parent)
            os.fsync(parent)
        finally:
            os.close(parent)
        root = Path(display).parent / ANCHOR
        fd = artifacts.directory(root)
        try:
            write_new(fd, 'authority.lock', {})
            d = {'format': FORMAT, 'id': uuid.uuid4().hex,
                 'root': object_identity(root, 'directory'), 'context': runtime,
                 'coordinator': coordinator, 'lock': object_identity(root / 'authority.lock', 'file'),
                 'coordinator_lock': object_identity(store.root / 'coordinator.lock', 'file')}
            write_new(fd, 'authority.json', d)
            parent = artifacts.directory(root.parent)
            try:
                write_new(parent, ANCHOR_SEAL, d)
            finally:
                os.close(parent)
            write_new(store.fd, MIRROR, d)
            write_new(fd, 'reservation.json', {'format': FORMAT, 'authority': d['id'], 'reservation': None})
            write_new(fd, 'ready.json', {'format': FORMAT, 'descriptor': journal.digest(d)})
        finally:
            os.close(fd)
        return cls(root)

    def _verify(self):
        journal.require(self.fd is not None, 'authority is closed')
        d = self.descriptor
        check_object(d['root'], 'directory')
        held = os.fstat(self.fd)
        journal.require((held.st_dev, held.st_ino) == (d['root']['device'], d['root']['inode']))
        check_context(d['context'])
        parent = artifacts.directory(self.root.parent)
        try:
            journal.require(read(parent, ANCHOR_SEAL) == d, 'runtime authority anchor seal is missing or changed')
        finally:
            os.close(parent)
        check_object(d['coordinator'], 'directory')
        check_object(d['lock'], 'file')
        check_object(d['coordinator_lock'], 'file')
        journal.require(read(self.fd, 'authority.json') == d, 'authority descriptor changed')
        journal.require(read(self.fd, 'ready.json') == {'format': FORMAT, 'descriptor': journal.digest(d)},
                        'authority provisioning seal is missing or changed')
        fd = artifacts.directory(d['coordinator']['path'])
        try:
            journal.require(read(fd, MIRROR) == d, 'canonical coordinator binding is missing or changed')
        finally:
            os.close(fd)

    @contextlib.contextmanager
    def transaction(self):
        self._verify()
        fd = os.open('authority.lock', os.O_RDWR | os.O_NOFOLLOW | os.O_NONBLOCK | os.O_CLOEXEC, dir_fd=self.fd)
        try:
            held = os.fstat(fd)
            bound = self.descriptor['lock']
            journal.require((held.st_dev, held.st_ino) == (bound['device'], bound['inode']))
            fcntl.flock(fd, fcntl.LOCK_EX | fcntl.LOCK_NB)
            self._verify()
            yield
        finally:
            os.close(fd)

    def _read_reservation(self):
        value = read(self.fd, 'reservation.json')
        version = value.get('format') if type(value) is dict else None
        journal.require(type(version) is int and version in (1, RESERVATION_FORMAT), 'unknown reservation contract')
        journal.exact(value, ('format', 'authority', 'reservation') + (() if version == 1 else ('binding', 'history')))
        journal.require(value['authority'] == self.descriptor['id'])
        current = value['reservation']
        if current is not None:
            reservation_shape(current, with_owner=version == RESERVATION_FORMAT)
        if version == 1:
            journal.require(current is None or current['intent'] == 'initial')
            return value
        journal.require(current is not None, 'version2 history cannot erase its initial reservation')
        if value['binding'] is not None:
            binding_shape(value['binding'], current)
        history = value['history']
        journal.require(type(history) is list and len(history) <= MAX_HANDOFFS)
        rows = [item['reservation'] for item in history] + [current]
        journal.require(rows[0]['intent'] == 'initial' and all(row['intent'] == 'transition' for row in rows[1:]))
        for index, item in enumerate(history):
            journal.exact(item, ('reservation', 'binding', 'handoff'))
            reservation_shape(item['reservation'], with_owner=True)
            if item['binding'] is not None:binding_shape(item['binding'], item['reservation'])
            proof = item['handoff']
            journal.exact(proof, ('request', 'phase', 'to', 'mode', 'journal_events', 'journal_sha256', 'closure', 'abandoned_admission'))
            journal.token(proof['request']);journal.token(proof['to'], journal.HASH);journal.token(proof['journal_sha256'], journal.HASH)
            journal.require(proof['mode'] in ('loaded-clean', 'loaded-recovery', 'partial-rollback'))
            journal.require((item['binding'] is None) == (proof['mode'] == 'partial-rollback'))
            if proof['mode'] == 'partial-rollback':journal.require(proof['phase'] == 'enable-previous' and proof['closure'] is not None)
            if proof['mode'] == 'partial-rollback':
                journal.exact(proof['abandoned_admission'], ('missing', 'sha256'))
                abandoned = proof['abandoned_admission']
                journal.require(type(abandoned['missing']) is list and type(abandoned['sha256']) is dict)
                journal.require(set(abandoned['missing']).isdisjoint(abandoned['sha256']) and
                                set(abandoned['missing']) | set(abandoned['sha256']) == {'admission.json', 'issued.json'})
                for digest in abandoned['sha256'].values():journal.token(digest, journal.HASH)
            else:journal.require(proof['abandoned_admission'] is None)
            journal.require(proof['phase'] in ('enable-candidate', 'enable-previous') and
                            type(proof['journal_events']) is int and 0 < proof['journal_events'] <= journal.MAX_EVENTS)
            journal.require(proof['to'] == journal.digest(rows[index + 1]) and
                            proof['request'] == rows[index + 1]['permit']['request'], 'handoff chain changed')
            if proof['closure'] is not None:
                journal.exact(proof['closure'], ('path', 'sha256', 'request'))
                journal.absolute(proof['closure']['path']);journal.token(proof['closure']['sha256'], journal.HASH);journal.token(proof['closure']['request'])
        journal.require(len({row['permit']['id'] for row in rows}) == len(rows) and
                        len({row['permit']['epoch'] for row in rows}) == len(rows) and
                        len({(row['gate']['device'], row['gate']['inode']) for row in rows}) == len(rows),
                        'one-use reservation identity was reused')
        return value

    def _save_reservation(self, value):
        temporary = '.pending-' + uuid.uuid4().hex
        write_new(self.fd, temporary, value)
        os.replace(temporary, 'reservation.json', src_dir_fd=self.fd, dst_dir_fd=self.fd)
        os.fsync(self.fd)

    def status(self):
        self._verify()
        return copy.deepcopy(self._read_reservation())

    def reserve(self, store, lease, gate, permit, *, guardian_receipt=None):
        """Persist before issuance. Exact retry never erases a previous reservation."""
        canonical = lease_check(store, lease)
        journal.require(canonical == self.descriptor['coordinator'],
                        'another coordinator journal cannot own this runtime')
        permit = admission.permit(permit)
        candidate = {'permit': permit, 'gate': object_identity(gate, 'directory'), 'intent': 'initial'}
        if guardian_receipt is not None:candidate['owner'] = owner_for_shell(permit['shell'], guardian_receipt)
        with self.transaction():
            value = self._read_reservation()
            if value['reservation'] is not None:
                journal.require(value['reservation'] == candidate,
                                'runtime remains reserved; cleanup and handoff are not inferred')
                os.fsync(self.fd)
            else:
                state = store.status()
                key = journal.digest(permit['specification'])
                journal.require(state['committed'] is None and not state['requests'] and
                                state['pending'] is None and state['collections'].get(key) == permit['specification'],
                                'initial admission requires the registered canonical collection')
                journal.require(admission.identity(permit['shell']['pid']) == permit['shell'], 'initial shell is gone')
                fd = artifacts.directory(gate)
                try:
                    journal.require(not os.listdir(fd), 'initial admission root must be fresh and empty')
                finally:
                    os.close(fd)
                value = ({'format': RESERVATION_FORMAT, 'authority': self.descriptor['id'],
                          'reservation': candidate, 'binding': None, 'history': []}
                         if guardian_receipt is not None else
                         {'format': FORMAT, 'authority': self.descriptor['id'], 'reservation': candidate})
                self._save_reservation(value)
            return {'format': FORMAT, 'root': str(self.root), 'authority': self.descriptor['id'],
                    'reservation': journal.digest(candidate)}

    def verify_reservation(self, ticket, gate, permit):
        """Read-only admission prerequisite; this does not itself grant a claim."""
        # Publication is atomic. A handoff requires old admission already closed and
        # every original guard exited; no live old constructor can race this read.
        journal.exact(ticket, ('format', 'root', 'authority', 'reservation'))
        journal.require(type(ticket['format']) is int and ticket['format'] == FORMAT, 'unsupported authority ticket')
        self._verify()
        value = self._read_reservation()['reservation']
        journal.require(value is not None, 'runtime has no durable reservation')
        expected = {'format': FORMAT, 'root': str(self.root), 'authority': self.descriptor['id'],
                    'reservation': journal.digest(value)}
        journal.require(ticket == expected and value['permit'] == admission.permit(permit) and
                        value['gate'] == object_identity(gate, 'directory'), 'admission differs from the reserved activation')
        if 'owner' in value:owner_live(value['owner'], value['permit']['shell'])
        self._verify()
        return copy.deepcopy(value)

    def _ticket(self, reservation):
        return {'format': FORMAT, 'root': str(self.root), 'authority': self.descriptor['id'],
                'reservation': journal.digest(reservation)}

    def _canonical(self, store, lease):
        journal.require(lease_check(store, lease) == self.descriptor['coordinator'],
                        'handoff requires the canonical coordinator journal')

    def _history(self, value, events):
        for old in value.get('history', []):
            proof = old['handoff']
            count = proof['journal_events']
            journal.require(count <= len(events) and journal.digest(events[:count]) == proof['journal_sha256'],
                            'canonical journal no longer contains the retained handoff evidence')

    def _gate(self, reservation):
        # Shared admission imports this module; defer the reciprocal import until
        # an explicit controller operation, never during module initialization.
        from shared_admission import Gate
        check_object(reservation['gate'], 'directory')
        gate = Gate(reservation['gate']['path'])
        try:
            row = gate.status()['record']
            journal.require(row['permit'] == reservation['permit'] and
                            row['authority'] == self._ticket(reservation), 'bound activation permit changed')
            return row
        finally:
            gate.close()

    def _binding_file(self, binding):
        reference = binding['runtime']
        check_object(reference['object'], 'file')
        body, digest = read_private_json(reference['object']['path'])
        journal.require(digest == reference['sha256'], 'original loaded-runtime binding changed')
        return body

    def bind_loaded(self, store, lease, ticket, binding_path, guardian_receipt):
        """Pin original live runtime/guardian before a transition or owner loss.

        The runtime adapter supplies its immutable loaded/helper observation.
        This operation verifies it against canonical journal intent, live admitted
        guards and the actual shell-parent relationship. No caller closure flag is
        accepted, and later replacement guardians cannot create this evidence.
        """
        self._canonical(store, lease)
        journal.exact(ticket, ('format', 'root', 'authority', 'reservation'))
        journal.require(type(ticket['format']) is int and ticket['format'] == FORMAT)
        with self.transaction():
            value = self._read_reservation()
            journal.require(value['format'] == RESERVATION_FORMAT and value['reservation'] is not None,
                            'legacy reservations are readable but cannot acquire a new handoff contract')
            reservation = value['reservation']
            journal.require(ticket == self._ticket(reservation), 'wrong reservation binding')
            events, state = store._load()
            self._history(value, events)
            body, digest = read_private_json(binding_path)
            reference = {'object': object_identity(binding_path, 'file'), 'sha256': digest}
            receipt_path = str(journal.absolute(str(guardian_receipt)))
            if value['binding'] is not None:
                saved = value['binding']
                journal.require(saved['runtime'] == reference and saved['guardian_receipt'] == receipt_path,
                                'a different runtime observation already owns this activation')
                self._binding_file(saved)
                os.fsync(self.fd)
                return copy.deepcopy(saved)

            journal.exact(body, ('loaded', 'helpers', 'shell', 'supervisor', 'admission'))
            permit = reservation['permit']
            loaded = body['loaded']
            journal.loaded(loaded, permit['specification'], permit['epoch'])
            pending = state['requests'].get(state['pending']) if state['pending'] else None
            journal.require(state['committed'] == loaded or
                            (pending is not None and pending['loaded'] == loaded and
                             pending['phase'] in ('verify-candidate', 'verify-previous') and not pending['uncertain']),
                            'loaded binding is not durably recorded in the canonical journal')
            journal.exact(body['admission'], ('root', 'permit'))
            journal.require(body['admission'] == {'root': reservation['gate']['path'], 'permit': permit})
            journal.require(process_pair(body['shell']) == process_pair(permit['shell']))
            guardian = {'owner': {key: body['supervisor'][key] for key in ('token', 'pid', 'start', 'boot', 'namespace')},
                        'root': process_pair(permit['shell'])}
            journal.guardian_identity(guardian)
            journal.require(reservation['owner'] == {'guardian': guardian, 'receipt': receipt_path},
                            'loaded observation cannot replace the guardian pinned before construction')
            journal.require(admission.identity(permit['shell']['pid']) == permit['shell'] and
                            admission.identity(guardian['owner']['pid']) == guardian_process(guardian) and
                            parent_of(permit['shell']) == guardian['owner']['pid'],
                            'original live guardian must own the admitted shell')
            receipt, _ = read_private_json(receipt_path)
            check_guardian_receipt(receipt, guardian)
            journal.require(receipt['phase'] == 'running' and receipt['proof'] is None,
                            'guardian must be bound before its closure or loss')
            row = self._gate(reservation)
            journal.require(row['state'] == 'open' and set(row['claims']) == set(journal.ROLES),
                            'all three original claims must still be admitted')
            journal.exact(body['helpers'], journal.ROLES)
            for role in journal.ROLES:
                claim = row['claims'][role]
                journal.require(claim['state'] == 'admitted' and admission.identity(claim['owner']['pid']) == claim['owner'],
                                'cannot bind an already lost admission owner')
                helpers = body['helpers'][role]
                journal.require(type(helpers) is list and 0 < len(helpers) <= 256)
                identities = [process_pair(helper) for helper in helpers]
                journal.require(process_pair(claim['owner']) in identities and
                                len({(item['pid'], item['start']) for item in identities}) == len(identities),
                                'runtime observation must retain the exact role guard')
                for helper in identities:
                    live = admission.identity(helper['pid'])
                    if live is not None and live['start'] == helper['start']:
                        journal.require(parent_of(live) == permit['shell']['pid'], 'observed helper belongs to another shell')
            binding = {'runtime': reference, 'loaded': copy.deepcopy(loaded), 'guardian': guardian,
                       'guardian_receipt': receipt_path, 'claims': copy.deepcopy(row['claims'])}
            value['binding'] = binding
            self._save_reservation(value)
            return copy.deepcopy(binding)

    def _closure_owner(self, owner):
        receipt, digest = read_private_json(owner['receipt'])
        guardian = owner['guardian']
        check_guardian_receipt(receipt, guardian)
        journal.require(receipt['phase'] == 'closed' and receipt['proof'] == 'waitpid-echild',
                        'the original guardian has no completed descendant-closure proof')
        journal.token(receipt['request'])
        journal.require(any(item.get('pid') == guardian['root']['pid'] for item in receipt['reaped']),
                        'guardian closure does not include its bound shell')
        journal.require(admission.identity(guardian['owner']['pid']) != guardian_process(guardian) and
                        not pair_live(guardian['root']), 'original guardian or shell remains live')
        return {'path': owner['receipt'], 'sha256': digest, 'request': receipt['request']}

    def _partial_gate(self, reservation):
        from shared_admission import Gate
        check_object(reservation['gate'], 'directory')
        gate = Gate(reservation['gate']['path'])
        expected = {'permit': reservation['permit'], 'authority': self._ticket(reservation)}
        observed = {'missing': [], 'sha256': {}}
        try:
            try:row = gate._read(sealed=False)
            except FileNotFoundError:row = None
            if row is not None:
                journal.require(row['permit'] == expected['permit'] and row['authority'] == expected['authority'],
                                'partial permit belongs to another activation')
            for name in ('admission.json', 'issued.json'):
                try:body, digest = read_private_json(Path(reservation['gate']['path']) / name)
                except FileNotFoundError:observed['missing'].append(name);continue
                observed['sha256'][name] = digest
                if name == 'issued.json':journal.require(body == gate._seal_value(expected), 'partial issuance seal changed')
            return row, observed
        finally:
            gate.close()

    def handoff(self, store, lease, gate, permit, *, guardian_receipt=None):
        """Reserve a canonical next activation after verified outgoing cleanup.

        New shells must name their already-running original guardian. Its identity
        is durable before the next guard can be issued, including when candidate
        construction never completes. This method performs no cleanup itself.
        """
        self._canonical(store, lease)
        permit = admission.permit(permit)
        candidate = {'permit': permit, 'gate': object_identity(gate, 'directory'), 'intent': 'transition'}
        with self.transaction():
            value = self._read_reservation()
            journal.require(value['format'] == RESERVATION_FORMAT and value['reservation'] is not None,
                            'handoff needs a versioned reservation history with original ownership')
            events, state = store._load()
            self._history(value, events)
            old, binding = value['reservation'], value['binding']
            if permit['shell'] == old['permit']['shell'] and (guardian_receipt is None or str(guardian_receipt) == old['owner']['receipt']):
                candidate['owner'] = copy.deepcopy(old['owner'])
            else:
                journal.require(guardian_receipt is not None, 'a new shell requires its original guardian before issuance')
                candidate['owner'] = owner_for_shell(permit['shell'], guardian_receipt)
            if old == candidate:
                journal.require(value['history'] and value['history'][-1]['handoff']['to'] == journal.digest(candidate))
                os.fsync(self.fd)
                return self._ticket(candidate)
            journal.require(len(value['history']) < MAX_HANDOFFS, 'authority history is full; retain it')
            journal.require(state['pending'] == permit['request'], 'no matching pending collection transition')
            request = state['requests'][state['pending']]
            phase = request['phase']
            journal.require(phase in ('enable-candidate', 'enable-previous') and not request['uncertain'],
                            'cleanup, unload and appearance receipts must precede authority handoff')
            journal.require(permit['epoch'] == journal.effect(request, phase) and
                            permit['specification'] == state['collections'][journal.target(request, phase)],
                            'candidate differs from canonical enable intent')
            outgoing = 'previous' if phase == 'enable-candidate' else 'candidate'
            drain_phase, disable_phase = 'drain-' + outgoing, 'disable-' + outgoing
            previous = request['previous'] if outgoing == 'previous' else request['loaded']
            partial = binding is None
            if partial:
                journal.require(outgoing == 'candidate' and old['intent'] == 'transition' and
                                old['permit']['request'] == request['id'] and
                                old['permit']['epoch'] == journal.effect(request, 'enable-candidate') and
                                journal.digest(old['permit']['specification']) == request['candidate'],
                                'an unbound reservation can only roll back its own candidate')
            else:
                journal.require(previous is not None and previous == binding['loaded'] and
                                previous['epoch'] == old['permit']['epoch'] and
                                previous['collection'] == journal.digest(old['permit']['specification']),
                                'handoff does not target the reserved loaded activation')
                body = self._binding_file(binding)
                journal.require(body['loaded'] == binding['loaded'] and body['admission']['permit'] == old['permit'])

            abandoned = None
            if partial:
                partial_closure = self._closure_owner(old['owner'])
                row, abandoned = self._partial_gate(old)
            else:
                row = self._gate(old)
                journal.require(row['state'] == 'closing' and row['close_request'] == journal.effect(request, drain_phase),
                                'outgoing admission must be irreversibly closed by this transition')
            if not partial:
                journal.require(set(row['claims']) == set(binding['claims']))
                for role, original in binding['claims'].items():
                    journal.require(all(row['claims'][role][key] == original[key] for key in ('instance', 'owner')),
                                    'outgoing admission cohort changed')
                journal.require(all(not pair_live(helper) for helpers in body['helpers'].values() for helper in helpers),
                                'an original helper is still live')
            journal.require(row is None or all(admission.identity(claim['owner']['pid']) != claim['owner'] for claim in row['claims'].values()),
                            'an outgoing guard has not exited')
            disable = request['receipts'].get(journal.effect(request, disable_phase) + '/completed')
            journal.require(disable is not None and disable['observation'] ==
                            {'epoch': old['permit']['epoch'], 'absent': True, 'helpers_exited': True},
                            'canonical unload/exit receipt is missing')
            appearance_phase = 'checkpoint-appearance' if outgoing == 'previous' else 'restore-appearance'
            journal.require(request['receipts'].get(journal.effect(request, appearance_phase) + '/completed') is not None and
                            request['checkpoint'] is not None, 'appearance transition receipt is missing')

            recovery = request['recovery'] if outgoing == 'previous' else request['candidate_recovery']
            closure = None
            if recovery is not None:
                plan, verified = recovery['plan'], recovery['verified']
                journal.require(verified is not None and plan['guardian'] == old['owner']['guardian'],
                                'recovery is not bound to the original guardian')
                if outgoing == 'previous':
                    journal.require(not partial and plan['loaded'] == binding['loaded'] and
                                    plan['runtime_binding_sha256'] == binding['runtime']['sha256'],
                                    'recovery is not bound to the original loaded runtime')
                else:
                    journal.require(plan['loaded'] == previous and
                                    plan['activation'] == {'collection': request['candidate'], 'epoch': old['permit']['epoch']} and
                                    plan['authority'] == self.descriptor['id'] and plan['reservation_sha256'] == journal.digest(old),
                                    'candidate recovery is not bound to the original reservation')
                journal.recovered(verified, plan)
                closure = self._closure_owner(old['owner'])
                journal.require(closure['sha256'] == verified['closure_receipt_sha256'] and
                                closure['request'] == plan['closure_request'], 'guardian closure differs from journal recovery')
                mode = 'partial-rollback' if partial else 'loaded-recovery'
            else:
                if not partial:
                    journal.require(all(claim['state'] == 'ended' for claim in row['claims'].values()),
                                    'lost loaded guard claims require original-guardian recovery')
                for role in journal.ROLES:
                    drained = request['receipts'].get(journal.effect(request, drain_phase) + '/' + role)
                    journal.require(drained is not None and drained['observation']['closed'] is True,
                                    'canonical member drain proof is missing')
                    observation = drained['observation']
                    if not partial:
                        journal.require(observation['absent'] is False and observation['binding'] == binding['loaded']['members'][role],
                                        'member drain belongs to another loaded instance')
                    elif previous is not None:
                        journal.require(observation['absent'] is False and observation['binding'] == previous['members'][role])
                # Partial construction has no complete helper cohort. Only the
                # already-pinned guardian's full ECHILD proof can close that gap;
                # all three journal drain receipts still attest transport cleanup.
                if partial or admission.identity(old['permit']['shell']['pid']) != old['permit']['shell']:
                    closure = partial_closure if partial else self._closure_owner(old['owner'])
                else:
                    owner_live(old['owner'], old['permit']['shell'])
                mode = 'partial-rollback' if partial else 'loaded-clean'

            owner_live(candidate['owner'], permit['shell'])
            from shared_admission import display_for_shell
            journal.require(str(display_for_shell(permit['shell'])) == self.descriptor['context']['socket']['path'],
                            'next shell belongs to another display')
            fd = artifacts.directory(gate)
            try:
                journal.require(not os.listdir(fd), 'new handoff admission root must be empty')
            finally:
                os.close(fd)
            prior = [item['reservation'] for item in value['history']] + [old]
            journal.require(all(permit['id'] != item['permit']['id'] and permit['epoch'] != item['permit']['epoch'] and
                                (candidate['gate']['device'], candidate['gate']['inode']) != (item['gate']['device'], item['gate']['inode'])
                                for item in prior), 'a historical one-use reservation cannot be reused')
            proof = {'request': request['id'], 'phase': phase, 'to': journal.digest(candidate), 'mode': mode,
                     'journal_events': len(events), 'journal_sha256': journal.digest(events), 'closure': closure,
                     'abandoned_admission': abandoned}
            value['history'].append({'reservation': old, 'binding': binding, 'handoff': proof})
            value['reservation'], value['binding'] = candidate, None
            self._save_reservation(value)
            return self._ticket(candidate)
