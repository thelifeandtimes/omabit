#!/usr/bin/env python3
"""Manifest and retire generated disks from one ended issue22 qualification VM.

The default only inventories. --run requires an inactive scope, the shared VM
lock, complete host/result evidence, and unchanged regular files. Sources, logs,
reports, immutable inputs and every other directory remain in place.
"""
import argparse
import fcntl
import hashlib
import json
import os
from pathlib import Path
import re
import stat
import subprocess


def digest(path):
    with path.open('rb') as stream:
        return hashlib.file_digest(stream, 'sha256').hexdigest()


def inventory(path):
    node = path.lstat()
    if (not stat.S_ISREG(node.st_mode) or node.st_uid != os.getuid()
            or path.resolve() != path.absolute() or node.st_nlink != 1):
        raise RuntimeError('unsafe generated file: ' + str(path))
    return dict(path=str(path), sha256=digest(path), bytes=node.st_size,
                allocated_bytes=node.st_blocks * 512, inode=node.st_ino,
                device=node.st_dev)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--run', action='store_true')
    parser.add_argument('--interrupted', action='store_true',
                        help='explicitly retire an interrupted host run with a retained interruption receipt')
    args = parser.parse_args()
    lab = args.output.absolute()
    labs = Path.home() / 'Documents/ChatGPT/omabit-private-docker-labs'
    if (lab.parent != labs or lab.resolve() != lab
            or not re.fullmatch(r'patch-upgrade-060-(?:runtime|upgrade)[0-9]{2}', lab.name)):
        parser.error('exact physical issue22 runtimeNN or upgradeNN lab required')
    with open('/tmp/omabit-issues-vm-runtime.lock', 'rb') as lock:
        fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
        state = subprocess.check_output(
            ['systemctl', '--user', 'show', 'omabit-' + lab.name + '.scope',
             '-p', 'ActiveState', '-p', 'ControlGroup'], text=True)
        if 'ActiveState=inactive\n' not in state or 'ControlGroup=\n' not in state:
            raise RuntimeError('disposable scope has not ended')
        host = json.loads((lab / 'host.json').read_text())
        evidence_names = ['host.json', 'serial.log']
        if args.interrupted:
            interruption = json.loads((lab / 'host-interruption.json').read_text())
            if interruption.get('passed') is not False or interruption.get('output') != str(lab):
                raise RuntimeError('matching host interruption receipt required')
            evidence_names.append('host-interruption.json')
            result = {'passed': False}
            if (lab / 'result.json').exists():
                evidence_names.append('result.json')
        else:
            result = json.loads((lab / 'result.json').read_text())
            if host.get('exit') != 0 or not isinstance(result.get('passed'), bool):
                raise RuntimeError('complete ended-guest evidence required')
            evidence_names.append('result.json')
        base = Path(host['base_path'])
        if digest(base) != host['base_sha256']:
            raise RuntimeError('immutable base changed')
        names = ['root.qcow2', 'payload.ext4', 'init', 'initramfs.cpio', 'boot/guest-modules']
        files = [lab / name for name in names] + sorted((lab / 'boot/modules').glob('*.ko'))
        rows = [inventory(path) for path in files]
        retained = {name: digest(lab / name) for name in evidence_names}
        receipt = dict(scope=__doc__, guest_passed=result['passed'], files=rows,
                       retained=retained, base_sha256=host['base_sha256'], complete=False)
        if not args.run:
            print(json.dumps(receipt, indent=2))
            return
        target = lab / 'generated-retirement.json'
        with target.open('x') as stream:
            stream.write(json.dumps(receipt, indent=2) + '\n')
        for row in rows:
            path = Path(row['path'])
            if inventory(path) != row:
                raise RuntimeError('generated file changed; preserve partial retirement receipt')
            path.unlink()
        if any(digest(lab / name) != value for name, value in retained.items()) or digest(base) != host['base_sha256']:
            raise RuntimeError('retained evidence or base changed')
        receipt['complete'] = True
        target.write_text(json.dumps(receipt, indent=2) + '\n')
        print(json.dumps(dict(complete=True, files=len(rows),
                             retired_bytes=sum(row['allocated_bytes'] for row in rows))))


if __name__ == '__main__':
    main()
