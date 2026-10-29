"""Format2 admission requires the canonical desktop authority reservation.

The original format1 reader/writer stays independent. This successor refuses old
permits rather than upgrading their contract. Closing and ending an existing claim
remain possible when the external authority is unavailable; neither proves drain.
"""
import copy
import os
from pathlib import Path
import uuid

import admission as legacy
import artifacts as artifacts
import journal as journal
import runtime_authority as authority

FORMAT = 2


def display_for_shell(identity):
    """Read only the exact shell's two display variables, never discover shells."""
    journal.require(legacy.identity(identity['pid']) == identity, 'bound shell is gone')
    with (Path('/proc') / str(identity['pid']) / 'environ').open('rb') as stream:
        raw = stream.read((1 << 20) + 1)
    journal.require(len(raw) <= 1 << 20, 'shell environment exceeds supported bound')
    values = {}
    for item in raw.split(b'\0'):
        name, separator, value = item.partition(b'=')
        if separator and name in (b'WAYLAND_DISPLAY', b'XDG_RUNTIME_DIR'):
            journal.require(name not in values, 'ambiguous shell display environment')
            values[name] = value.decode('utf-8')
    journal.require(legacy.identity(identity['pid']) == identity, 'shell changed while binding display')
    name = values.get(b'WAYLAND_DISPLAY', '')
    journal.require(name and len(name) <= 4096 and not any(ord(c) < 32 for c in name),
                    'an explicit Wayland display is required')
    if name.startswith('/'):
        return journal.absolute(name)
    journal.require(Path(name).name == name and name not in ('.', '..'), 'unsupported relative display path')
    return journal.absolute(values.get(b'XDG_RUNTIME_DIR', '')) / name


def ticket(value):
    journal.exact(value, ('format', 'root', 'authority', 'reservation'))
    journal.require(type(value['format']) is int and value['format'] == 1)
    journal.absolute(value['root'])
    journal.token(value['authority'])
    journal.token(value['reservation'], journal.HASH)
    return copy.deepcopy(value)


def validate(value):
    journal.exact(value, ('format', 'permit', 'authority', 'state', 'close_request', 'claims'))
    journal.require(type(value['format']) is int and value['format'] == FORMAT,
                    'unsupported shared admission; legacy permits need their original cleanup reader')
    ticket(value['authority'])
    old = {k: v for k, v in value.items() if k != 'authority'}
    old['format'] = 1
    legacy.validate(old)
    return value


class Gate(legacy.Gate):
    def _read(self, *, sealed=True):
        row = validate(artifacts.object_json(self._read_bytes('admission.json')))
        if sealed:
            seal = artifacts.object_json(self._read_bytes('issued.json'))
            journal.require(seal == self._seal_value(row), 'shared issuance seal differs')
        return row

    @staticmethod
    def _seal_value(row):
        return {'format': FORMAT, 'permit_sha256': journal.digest(row['permit']),
                'authority_sha256': journal.digest(row['authority'])}

    def _save(self, value):
        validate(value)
        data = journal.canonical(value) + b'\n'
        journal.require(len(data) <= legacy.MAX_BYTES)
        name = '.pending-' + uuid.uuid4().hex
        fd = self._open(name, os.O_WRONLY | os.O_CREAT | os.O_EXCL)
        with os.fdopen(fd, 'wb') as stream:
            stream.write(data)
            stream.flush()
            os.fsync(stream.fileno())
        os.replace(name, 'admission.json', src_dir_fd=self.fd, dst_dir_fd=self.fd)
        self._sync()

    def _seal(self, row):
        expected = self._seal_value(row)
        try:
            fd = self._open('issued.json', os.O_RDONLY)
        except FileNotFoundError:
            journal.require(not row['claims'] and row['state'] == 'open', 'active shared permit lost its seal')
            name = '.seal-' + uuid.uuid4().hex
            authority.write_new(self.fd, name, expected)
            os.replace(name, 'issued.json', src_dir_fd=self.fd, dst_dir_fd=self.fd)
        else:
            with os.fdopen(fd, 'rb') as stream:
                self._regular(stream.fileno())
                journal.require(artifacts.object_json(stream.read(legacy.MAX_BYTES + 1)) == expected,
                                'shared admission seal changed')
        self._sync()

    def _check_authority(self, row, *, shell=False):
        a = authority.Authority(row['authority']['root'])
        try:
            a.verify_reservation(row['authority'], self.root, row['permit'])
            if shell:
                display = display_for_shell(row['permit']['shell'])
                journal.require(str(display) == a.descriptor['context']['socket']['path'] and
                                display.parent / authority.ANCHOR == a.root,
                                'permit authority is not the shell display authority')
        finally:
            a.close()

    def issue(self, value, reservation):
        value = legacy.permit(value)
        reservation = ticket(reservation)
        desired = {'format': FORMAT, 'permit': value, 'authority': reservation,
                   'state': 'open', 'close_request': None, 'claims': {}}
        with self.transaction():
            self._check_authority(desired, shell=True)
            try:
                old = self._read(sealed=False)
            except FileNotFoundError:
                try:
                    os.stat('issued.json', dir_fd=self.fd, follow_symlinks=False)
                except FileNotFoundError:
                    old = None
                else:
                    raise journal.Conflict('issued shared record is missing; preserve the fence')
            if old is not None:
                journal.require(old['permit'] == value and old['authority'] == reservation,
                                'another reserved activation owns this gate')
                self._seal(old)
                return copy.deepcopy(old)
            self._save(desired)
            self._seal(desired)
            return copy.deepcopy(desired)

    def claim(self, expected, role, instance):
        self._check_authority(self._read(), shell=True)
        claimed = super().claim(expected, role, instance)
        return Claim(claimed)


class Claim:
    def __init__(self, inner):
        self.inner = inner

    @property
    def fd(self):
        return self.inner.fd

    def status(self):
        value = self.inner.status()
        row = self.inner.gate._read()
        self.inner.gate._check_authority(row, shell=True)
        value['format'] = FORMAT
        value['authority'] = copy.deepcopy(row['authority'])
        return value

    def end(self):
        # Authority loss fences new effects but must not obstruct guard cleanup.
        return self.inner.end()
