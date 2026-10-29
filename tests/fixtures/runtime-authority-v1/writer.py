"""Explicit shared desktop reservation, independent of per-profile XDG roots.

Internal first-reservation contract only. There is deliberately no release,
replacement, adoption or handoff operation: vanished processes and ended guards
cannot establish transport or appearance cleanup. Provisioning is a one-time
explicit operation for a fresh runtime/journal, never an automatic repair.
"""
import contextlib
import copy
import fcntl
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
ANCHOR = '.omabit-collection-authority'
ANCHOR_SEAL = ANCHOR + '.json'
MIRROR = 'runtime-authority.json'
MAX_BYTES = 256 << 10


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
        journal.exact(value, ('format', 'authority', 'reservation'))
        journal.require(type(value['format']) is int and value['format'] == FORMAT and
                        value['authority'] == self.descriptor['id'], 'unknown reservation contract')
        r = value['reservation']
        if r is not None:
            journal.exact(r, ('permit', 'gate', 'intent'))
            admission.permit(r['permit'])
            journal.require(r['intent'] == 'initial', 'unsupported authority handoff')
            journal.exact(r['gate'], ('path', 'device', 'inode', 'owner', 'kind'))
        return value

    def _save_reservation(self, value):
        temporary = '.pending-' + uuid.uuid4().hex
        write_new(self.fd, temporary, value)
        os.replace(temporary, 'reservation.json', src_dir_fd=self.fd, dst_dir_fd=self.fd)
        os.fsync(self.fd)

    def status(self):
        self._verify()
        return copy.deepcopy(self._read_reservation())

    def reserve(self, store, lease, gate, permit):
        """Persist before issuance. Exact retry never erases a previous reservation."""
        canonical = lease_check(store, lease)
        journal.require(canonical == self.descriptor['coordinator'],
                        'another coordinator journal cannot own this runtime')
        permit = admission.permit(permit)
        candidate = {'permit': permit, 'gate': object_identity(gate, 'directory'), 'intent': 'initial'}
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
                value['reservation'] = candidate
                self._save_reservation(value)
            return {'format': FORMAT, 'root': str(self.root), 'authority': self.descriptor['id'],
                    'reservation': journal.digest(candidate)}

    def verify_reservation(self, ticket, gate, permit):
        """Read-only admission prerequisite; this does not itself grant a claim."""
        # Reservations cannot change after the first publication in this version.
        # Atomic reads avoid fencing simultaneous guards on mere lock contention.
        journal.exact(ticket, ('format', 'root', 'authority', 'reservation'))
        journal.require(type(ticket['format']) is int and ticket['format'] == FORMAT, 'unsupported authority ticket')
        self._verify()
        value = self._read_reservation()['reservation']
        journal.require(value is not None, 'runtime has no durable reservation')
        expected = {'format': FORMAT, 'root': str(self.root), 'authority': self.descriptor['id'],
                    'reservation': journal.digest(value)}
        journal.require(ticket == expected and value['permit'] == admission.permit(permit) and
                        value['gate'] == object_identity(gate, 'directory'), 'admission differs from the reserved activation')
        self._verify()
        return copy.deepcopy(value)
