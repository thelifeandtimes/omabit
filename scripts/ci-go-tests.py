#!/usr/bin/env python3
"""Run Go tests; annotate only identifiers already declared in public source."""
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
    known.setdefault(package, set()).update(re.findall(r'^func (Test[A-Za-z0-9_]+)\(', path.read_text(), re.M))
args = ['go', 'test', '-json', *sys.argv[1:]]
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
