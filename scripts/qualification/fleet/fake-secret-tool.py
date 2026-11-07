#!/usr/bin/python3
"""Synthetic Secret Service transport; usable only inside the offline CLI lab."""
import os
import re
import sys
from pathlib import Path

assert Path('/omabit-fleet-cli-fixture').read_text() == 'offline synthetic credential transport\n'
home = Path(os.environ['HOME'])
assert home.is_relative_to('/tmp') and not home.is_symlink()
args = sys.argv[1:]
assert args[0] in ('store', 'lookup', 'clear')
assert args[-6:-1] == ['application', 'urbit-collector', 'kind', 'eyre-session', 'scope']
key = args[-1]
assert re.fullmatch('[0-9a-f]{64}', key)
root = home / 'fixture-keyring'
root.mkdir(mode=0o700, exist_ok=True)
path = root / key
if args[0] == 'store':
    if (home / 'fail-secret-store').exists():
        sys.exit(2)
    data = sys.stdin.buffer.read(16385)
    assert 0 < len(data) <= 16384
    fd = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
    with os.fdopen(fd, 'wb') as f:
        f.write(data)
        f.flush()
        os.fsync(f.fileno())
elif args[0] == 'lookup':
    if not path.exists():
        sys.exit(1)
    sys.stdout.buffer.write(path.read_bytes())
else:
    path.unlink(missing_ok=True)
