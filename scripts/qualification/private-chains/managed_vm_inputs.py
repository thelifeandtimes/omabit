"""Fixture-only delivery of reviewed inert inputs inside the fresh no-NIC VM.

This is not a supported dependency installer or publisher trust mechanism.
Only a newly created destination is changed; every partial output is retained.
The real Go backend must independently verify the installed tree before use.
"""
import hashlib
import json
import os
from pathlib import Path, PurePosixPath
import stat

MANIFEST = 'fd4c0739be071ef2ad88fb10bcd17cbc93c712e6dc7544ca8a2c9f0232f7832a'
ROOT = Path('/home/lab/.local/share/urbit-runner/managed')
SOURCE = Path('/opt/omabit-lab/private-chain/src/runtime-inputs')


def require_guest():
    if (os.geteuid() != 0
        or Path('/opt/omabit-lab/guest-marker').read_text() != 'offline-import-guest-v1\n'
        or Path('/sys/class/dmi/id/sys_vendor').read_text().strip() != 'QEMU'
        or Path('/proc/1/comm').read_text().strip() != 'systemd'
        or sorted(p.name for p in Path('/sys/class/net').iterdir()) != ['lo']):
        raise ValueError('fresh no-NIC guest required before input delivery')


def sync_directory(path):
    fd = os.open(path, os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW | os.O_CLOEXEC)
    try:
        os.fsync(fd)
    finally:
        os.close(fd)


def copy_file(source, target, expected, size, executable):
    if source.resolve(strict=True) != source:
        raise ValueError('indirect input refused')
    fd = os.open(source, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK | os.O_CLOEXEC)
    try:
        before = os.fstat(fd)
        if not stat.S_ISREG(before.st_mode) or before.st_nlink != 1 or before.st_size != size:
            raise ValueError('input metadata differs')
        # Exclusively create only this fresh fixture destination; never repair it.
        with os.fdopen(os.dup(fd), 'rb') as incoming, target.open('xb') as outgoing:
            total = 0
            h = hashlib.sha256()
            while chunk := incoming.read(1024*1024):
                total += len(chunk)
                if total > size:
                    raise ValueError('input exceeds bound')
                h.update(chunk)
                outgoing.write(chunk)
            after = os.fstat(fd)
            fields = ('st_dev','st_ino','st_size','st_mtime_ns','st_ctime_ns')
            if (total != size or h.hexdigest() != expected
                or any(getattr(before,n) != getattr(after,n) for n in fields)):
                raise ValueError('input content changed')
            outgoing.flush()
            os.fchown(outgoing.fileno(), 1000, 1000)
            os.fchmod(outgoing.fileno(), 0o500 if executable else 0o400)
            os.fsync(outgoing.fileno())
    finally:
        os.close(fd)


def deliver():
    require_guest()
    os.umask(0o077)
    root = ROOT.lstat()
    if (ROOT.resolve(strict=True) != ROOT or not stat.S_ISDIR(root.st_mode)
        or root.st_uid != 1000 or stat.S_IMODE(root.st_mode) != 0o700):
        raise ValueError('fresh installed managed root required')
    path = SOURCE/'manifest.json'
    with path.open('rb') as f:
        raw = f.read(8*1024*1024+1)
    if len(raw) > 8*1024*1024 or hashlib.sha256(raw).hexdigest() != MANIFEST:
        raise ValueError('reviewed manifest differs')
    m = json.loads(raw)
    files = m['files']
    if len(files) != 12352 or sum(v['bytes'] for v in files) != 834044028:
        raise ValueError('reviewed inventory differs')
    parent = ROOT/'private-runtime-inputs'
    parent.mkdir(mode=0o700)
    os.chown(parent, 1000, 1000)
    dest = parent/MANIFEST
    dest.mkdir(mode=0o700)
    directories = {dest}
    names = set()
    for entry in files:
        name = entry['path']
        rel = PurePosixPath(name)
        if (name in names or not name or name != str(rel) or rel.is_absolute()
            or '..' in rel.parts or '\\' in name or name == 'manifest.json'):
            raise ValueError('reviewed inventory path differs')
        names.add(name)
        parts = rel.parts[:-1]
        at = dest
        for part in parts:
            at = at/part
            if at not in directories:
                at.mkdir(mode=0o700)
                directories.add(at)
        copy_file(SOURCE/name, dest/name, entry['sha256'], entry['bytes'], entry['executable'])
    copy_file(path, dest/'manifest.json', MANIFEST, len(raw), False)
    for directory in sorted(directories, key=lambda p: len(p.parts), reverse=True):
        os.chown(directory, 1000, 1000)
        os.chmod(directory, 0o500)
        sync_directory(directory)
    sync_directory(parent)
    sync_directory(ROOT)
    return {'manifest_sha256': MANIFEST, 'files': len(files), 'bytes': 834044028}
