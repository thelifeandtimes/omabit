#!/usr/bin/env python3
"""Run Go tests; annotate only identifiers already declared in public source."""
import hashlib
import json
from pathlib import Path
import re
import subprocess
import sys

root = Path(__file__).resolve().parents[1]
module = re.search(r'^module (\S+)', (root / 'go.mod').read_text(), re.M).group(1)
known = {}
for path in root.rglob('*_test.go'):
    if '.git' in path.parts:
        continue
    package = module + '/' + path.parent.relative_to(root).as_posix()
    known.setdefault(package, set()).update(re.findall(r'^func ((?:Test|Example|Fuzz)[A-Za-z0-9_]*)\(', path.read_text(), re.M))
options = sys.argv[1:]
if options[:1] == ['--managed-shard']:
    index, count = map(int, options[1].split('/'))
    if not 0 <= index < count <= 16:
        raise SystemExit('invalid managed test shard')
    options = options[2:]
    package = module + '/internal/managed'
    # Discover the executable test inventory, including examples/fuzz seeds if
    # added later. Every name must be in source before it can be annotated.
    listing = subprocess.check_output(['go', 'test', '-list', '.', './internal/managed'], cwd=root, text=True)
    names = [line for line in listing.splitlines() if re.fullmatch(r'(?:Test|Example|Fuzz)[A-Za-z0-9_]*', line)]
    if not names or len(names) != len(set(names)) or not set(names) <= known[package]:
        raise SystemExit('managed test inventory differs from declared source')
    groups = [[name for name in names if int.from_bytes(hashlib.sha256(name.encode()).digest(), 'big') % count == shard] for shard in range(count)]
    if set().union(*map(set, groups)) != set(names) or sum(map(len, groups)) != len(names) or any(not group for group in groups):
        raise SystemExit('managed test shards do not cover the inventory exactly once')
    print(f'Managed race shard {index + 1}/{count}: {len(groups[index])}/{len(names)} tests', flush=True)
    options += ['-run', '^(' + '|'.join(map(re.escape, groups[index])) + ')$', './internal/managed']
args = ['go', 'test', '-json', *options]
process = subprocess.Popen(args, cwd=root, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
failures = set()
for line in process.stdout:
    print(line, end='', flush=True)
    try:
        row = json.loads(line)
    except ValueError:
        continue
    if row.get('Action') != 'fail' or row.get('Package') not in known:
        continue
    name = row.get('Test', '').split('/', 1)[0]
    if not name or name in known[row['Package']]:
        failures.add((row['Package'], name))
process.stdout.close()
status = process.wait()
for package, name in sorted(failures):
    print('::error title=Go fixture failed::' + package + (' ' + name if name else ''), flush=True)
raise SystemExit(status)
