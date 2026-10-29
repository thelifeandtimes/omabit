#!/usr/bin/env python3
"""Seal inert runtime inputs from a separately reviewed private-chain appliance.

No dependency, database library, Docker command, ship or VM is executed. A result
still requires independent private-chain-inputs inspection and runtime admission.
Failed output is retained; never reuse or repair it in place.
"""
import argparse
import hashlib
import io
import json
import os
from pathlib import Path, PurePosixPath
import re
import shutil
import stat
import tarfile

HERE = Path(__file__).resolve().parent
REPO = HERE.parents[2]
MAX_ARCHIVE = 512 * 1024**2
MAX_FILES = 40000
MAX_BYTES = 1024**3
PILL_BYTES = 217421172
PILL_SHA = 'f8b08c53dd6d0aa9c4f7654f0a7bcd11e94089222888a3c99c0a10eedcbb377b'
IMAGE = 'docker.io/tloncorp/vere@sha256:47fbcf0710ded939931ce3c6408864cee5e8848c5765da0b51a90ca0b1ab1afd'


def require(ok, message):
    if not ok:
        raise ValueError(message)


def sha(data):
    return hashlib.sha256(data).hexdigest()


def file_hash(path):
    with path.open('rb') as stream:
        return hashlib.file_digest(stream, 'sha256').hexdigest()


def digest(value):
    return isinstance(value, str) and re.fullmatch('[0-9a-f]{64}', value) is not None


def relative(name):
    require(isinstance(name, str) and 0 < len(name) <= 1024, 'invalid input name')
    p = PurePosixPath(name)
    require(name != '.' and not p.is_absolute() and str(p) == name and '..' not in p.parts
            and '\\' not in name and all(ord(c) >= 32 and ord(c) != 127 for c in name), 'invalid input path')
    return name


def unique(pairs):
    result = {}
    for name, value in pairs:
        require(name not in result, 'duplicate input metadata')
        result[name] = value
    return result


def read_file(path, maximum, expected=None):
    fd = os.open(path, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK | os.O_CLOEXEC)
    try:
        before = os.fstat(fd)
        require(stat.S_ISREG(before.st_mode) and before.st_nlink == 1 and before.st_size <= maximum, 'bounded regular input required')
        with os.fdopen(os.dup(fd), 'rb') as stream:
            data = stream.read(maximum + 1)
        after = os.fstat(fd)
    finally:
        os.close(fd)
    require(len(data) == before.st_size and len(data) <= maximum
            and (before.st_dev, before.st_ino, before.st_size, before.st_mtime_ns, before.st_ctime_ns)
            == (after.st_dev, after.st_ino, after.st_size, after.st_mtime_ns, after.st_ctime_ns), 'input changed during read')
    if expected is not None:
        require(digest(expected) and sha(data) == expected, 'input review hash differs')
    return data


def payload_from_cpio(raw, expected):
    require(len(raw) <= MAX_ARCHIVE, 'initramfs exceeds limit')
    image = io.BytesIO(raw)
    names, payload = set(), None
    while True:
        header = image.read(110)
        require(len(header) == 110 and header[:6] == b'070701' and re.fullmatch(b'[0-9a-fA-F]{104}', header[6:]) is not None, 'unsupported initramfs')
        fields = [int(header[6+n*8:14+n*8], 16) for n in range(13)]
        length, size = fields[11], fields[6]
        require(1 < length <= 4096 and size <= MAX_ARCHIVE, 'initramfs entry exceeds limit')
        encoded = image.read(length)
        require(len(encoded) == length and encoded[-1:] == b'\0', 'invalid initramfs name')
        name = encoded[:-1].decode()
        require(name not in names and len(names) < MAX_FILES, 'duplicate or excess initramfs entry')
        names.add(name)
        image.read(-(110+length) % 4)
        if name == 'TRAILER!!!':
            require(size == 0, 'invalid initramfs trailer')
            break
        require(image.tell()+size <= len(raw), 'truncated initramfs data')
        if name == 'payload.tar.gz':
            require(stat.S_ISREG(fields[1]) and fields[4] == 1, 'regular payload required')
            payload = image.read(size)
        else:
            image.seek(size, 1)
        image.read(-size % 4)
    require(payload is not None and digest(expected) and sha(payload) == expected, 'payload review hash differs')
    return payload


def inventory(raw):
    value = json.loads(raw, object_pairs_hook=unique)
    require(isinstance(value, dict) and 0 < len(value) <= MAX_FILES, 'bounded input inventory required')
    total = 0
    for name, row in value.items():
        relative(name)
        require(isinstance(row, dict) and set(row) == {'bytes', 'sha256'}
                and type(row['bytes']) is int and 0 <= row['bytes'] <= MAX_BYTES-total
                and digest(row['sha256']), 'invalid input inventory entry')
        total += row['bytes']
    return value


def selection(entries):
    mapping = {'src/node-closure.json': 'node-closure.json'}
    for name in ['docker_gateway.cjs', 'docker_gateway_resume.cjs', 'gateway_resume_contract.cjs']:
        mapping['src/'+name] = 'gateway/'+name
    for name in entries:
        if name in ('vere', 'source/bridge/package.json') or name.startswith(('node/', 'tools/', 'compiled/', 'deployment/')):
            mapping[name] = name
    require(all(name in entries for name in mapping), 'required runtime input is absent')
    require(len(set(mapping.values())) == len(mapping), 'runtime input collision')
    for name in mapping.values():relative(name)
    return mapping


def output_path(value):
    path = value.absolute()
    parent = path.parent
    info = parent.stat()
    require(parent.resolve(strict=True) == parent and stat.S_ISDIR(info.st_mode)
            and info.st_uid == os.getuid() and stat.S_IMODE(info.st_mode) == 0o700,
            'output parent must be canonical, owned and mode0700')
    require(not os.path.lexists(path), 'fresh output required; preserve any previous attempt')
    return path


def encode(value):
    return json.dumps(value, separators=(',', ':'), ensure_ascii=False).encode()


def prepare(args):
    out = output_path(args.output)
    appliance = args.appliance.absolute()
    require(appliance.resolve(strict=True) == appliance, 'canonical appliance path required')
    build = json.loads(read_file(appliance/'build.json', 1024**2, args.build_sha256), object_pairs_hook=unique)
    require(build.get('exercise_private_chain') is True and build.get('gateway_resume_qualification') is True, 'reviewed private gateway appliance required')
    entries = inventory(read_file(appliance/'payload-inventory.json', 8*1024**2, build['payload_inventory_sha256']))
    raw = read_file(appliance/'artifacts/initramfs.cpio', MAX_ARCHIVE, build['artifacts']['initramfs.cpio'])
    payload = payload_from_cpio(raw, build['payload_sha256'])
    del raw
    pill = read_file(args.pill, PILL_BYTES, PILL_SHA)
    require(len(pill) == PILL_BYTES, 'bootstrap pill size differs')
    mapping = selection(entries)
    require(shutil.disk_usage(out.parent).free > sum(entries[n]['bytes'] for n in mapping)+len(pill)+2*1024**3, 'insufficient private output headroom')
    # This is public pinned review metadata, not evidence that a DB is unkeyed.
    foundation = json.loads(read_file(REPO/'internal/privatechain/testdata/qualified-foundation.json', 65536), object_pairs_hook=unique)
    foundation = {k:foundation[k] for k in ('version', 'profile', 'chain_id', 'genesis', 'contracts')}
    foundation['contracts'] = [{k:c[k] for k in ('name','address','code_sha256')} for c in sorted(foundation['contracts'], key=lambda c:c['name'])]
    os.umask(0o077)
    out.mkdir(mode=0o700)
    files, seen = [], set()
    with tarfile.open(fileobj=io.BytesIO(payload), mode='r|gz') as archive:
        for entry in archive:
            relative(entry.name)
            require(entry.isfile() and entry.name in entries and entry.name not in seen
                    and entry.size == entries[entry.name]['bytes'], 'archive inventory differs')
            seen.add(entry.name)
            if entry.name not in mapping:continue
            name = mapping[entry.name]
            target = out/name
            target.parent.mkdir(mode=0o700, parents=True, exist_ok=True)
            with archive.extractfile(entry) as source, target.open('xb') as sink:
                shutil.copyfileobj(source, sink, 1024**2)
                sink.flush();os.fsync(sink.fileno())
            require(file_hash(target) == entries[entry.name]['sha256'], 'copied input differs')
            executable = bool(entry.mode & 0o111)
            target.chmod(0o500 if executable else 0o400)
            files.append({'path':name,'bytes':entry.size,'sha256':entries[entry.name]['sha256'],'executable':executable})
    require(seen == set(entries) and len(files) == len(mapping), 'incomplete runtime inputs')
    with (out/'boot.pill').open('xb') as stream:
        stream.write(pill);stream.flush();os.fsync(stream.fileno())
    del pill
    (out/'boot.pill').chmod(0o400)
    files.append({'path':'boot.pill','bytes':PILL_BYTES,'sha256':PILL_SHA,'executable':False})
    manifest = {'version':1,'format':'private-chain-runtime-inputs-v1','platform':'linux/amd64','image':IMAGE,'foundation':foundation,'files':sorted(files,key=lambda f:f['path'])}
    encoded = encode(manifest)
    with (out/'manifest.json').open('xb') as stream:
        stream.write(encoded);stream.flush();os.fsync(stream.fileno())
    (out/'manifest.json').chmod(0o400)
    dirs = [out, *[p for p in out.rglob('*') if p.is_dir()]]
    for path in sorted(dirs, key=lambda p:len(p.parts), reverse=True):
        path.chmod(0o500)
        fd = os.open(path, os.O_DIRECTORY)
        try:os.fsync(fd)
        finally:os.close(fd)
    return {'directory':str(out),'manifest_sha256':sha(encoded),'foundation_digest':sha(b'omabit-private-chain-foundation-v1\0'+encode(foundation)),
            'files':len(files),'bytes':sum(f['bytes'] for f in files),'runtime_authorized':False,'requires_independent_inspection':True,'raw_inputs_private':True}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--appliance',type=Path,required=True)
    parser.add_argument('--build-sha256',required=True,help='separately reviewed exact build.json hash')
    parser.add_argument('--pill',type=Path,required=True)
    parser.add_argument('--output',type=Path,required=True,help='new directory under a private mode0700 parent')
    args = parser.parse_args()
    print(json.dumps(prepare(args),indent=2))


if __name__ == '__main__':main()
