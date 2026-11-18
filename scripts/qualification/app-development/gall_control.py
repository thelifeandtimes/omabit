"""Fixed restart control for one supervisor-owned disposable native fake.

This module performs no signal or process creation. The supervisor supplies its
own observed child identity, validates before effects, and records each transition.
It accepts no path, executable, command, runtime flag or target selection.
"""
import hashlib
import json
import os
from pathlib import Path
import re
import stat

KEYS = {'version', 'id', 'action', 'pid', 'start', 'vere_sha256', 'device', 'inode'}

def validate_restart(request, owned):
    if not isinstance(request, dict) or set(request) != KEYS:
        raise ValueError('invalid owned restart fields')
    if (type(request['version']) is not int or request['version'] != 1
            or not isinstance(request['id'], str) or not re.fullmatch('[a-f0-9]{32}', request['id'])
            or request['action'] != 'restart'):
        raise ValueError('invalid owned restart action')
    for key in ('pid', 'device', 'inode'):
        if type(request[key]) is not int or request[key] != owned[key]:
            raise ValueError('owned restart object differs')
    if request['pid'] < 2 or request['device'] < 0 or request['inode'] < 1:
        raise ValueError('invalid owned process or pier')
    if (not isinstance(request['start'], str) or not re.fullmatch('[0-9]+', request['start'])
            or request['start'] != owned['start'] or request['vere_sha256'] != owned['vere_sha256']
            or owned['executable'] != '/artifacts/vere' or owned['running'] is not True):
        raise ValueError('owned restart process differs')

def read_request(path):
    fd = os.open(path, os.O_RDONLY | os.O_CLOEXEC | os.O_NOFOLLOW | os.O_NONBLOCK)
    with os.fdopen(fd, 'rb') as stream:
        info = os.fstat(stream.fileno())
        if (not stat.S_ISREG(info.st_mode) or info.st_nlink != 1 or info.st_uid != os.geteuid()
                or info.st_size > 4096 or info.st_mode & 0o077):
            raise ValueError('owned restart request must be a bounded private ordinary file')
        data = stream.read(4097)
        if len(data) > 4096:
            raise ValueError('owned restart request exceeds bound')
    return json.loads(data), hashlib.sha256(data).hexdigest()

def write_receipt(lab, receipt):
    data = (json.dumps(receipt, sort_keys=True) + '\n').encode()
    history = Path(lab) / 'gall-control-history.jsonl'
    fd = os.open(history, os.O_WRONLY | os.O_CREAT | os.O_APPEND | os.O_NOFOLLOW | os.O_CLOEXEC, 0o600)
    with os.fdopen(fd, 'ab') as stream:
        info = os.fstat(stream.fileno())
        if not stat.S_ISREG(info.st_mode) or info.st_nlink != 1:
            raise ValueError('owned restart history is not an ordinary file')
        stream.write(data); stream.flush(); os.fsync(stream.fileno())
    temporary = Path(lab) / 'gall-control-receipt.tmp'
    fd = os.open(temporary, os.O_WRONLY | os.O_CREAT | os.O_TRUNC | os.O_NOFOLLOW | os.O_CLOEXEC, 0o600)
    with os.fdopen(fd, 'wb') as stream:
        stream.write(data); stream.flush(); os.fsync(stream.fileno())
    temporary.replace(Path(lab) / 'gall-control-receipt.json')
    fd = os.open(lab, os.O_RDONLY | os.O_DIRECTORY | os.O_CLOEXEC)
    try:
        os.fsync(fd)
    finally:
        os.close(fd)
