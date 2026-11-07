#!/usr/bin/python3
"""Public fleet CLI in private namespaces with a synthetic keyring transport."""
import gzip
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import re
import subprocess
import sys

root = Path(__file__).resolve().parents[1]
label = sys.argv[1]
assert re.fullmatch('[a-z0-9-]+', label)
spec = importlib.util.spec_from_file_location('storage', root / 'scripts/qualification/app-development/storage.py')
storage = importlib.util.module_from_spec(spec)
spec.loader.exec_module(storage)
admission = storage.admit(400 * 1024**2)  # warm cache, collector dependency closure
out = Path('/tmp/omabit-fleet-connect-' + label)
out.mkdir(mode=0o700)
scratch = out / 'scratch'
scratch.mkdir(mode=0o700)
for name in ('home', 'config', 'state', 'data', 'cache', 'run'):
    (scratch / name).mkdir(mode=0o700)
(out / 'storage-admission.json').write_text(json.dumps(admission, indent=2) + '\n')
marker = out / 'fixture-marker'
marker.write_text('offline synthetic credential transport\n')
binary = out / 'collector.test'
build = subprocess.run(['go', 'test', '-race', '-c', '-o', str(binary), './backend/cmd/urbit-collector'], cwd=root,
                       env={**os.environ, 'GOCACHE': '/tmp/omabit-040-go-cache', 'GOTOOLCHAIN': 'local', 'GOPROXY': 'off'},
                       stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
(out / 'build.log').write_text(build.stdout)
if build.returncode:
    print(build.stdout)
    sys.exit(build.returncode)
fake = root / 'scripts/qualification/fleet/fake-secret-tool.py'
command = ['bwrap', '--unshare-all', '--die-with-parent', '--clearenv', '--ro-bind', '/usr', '/usr',
           '--ro-bind', str(fake), '/usr/bin/secret-tool', '--symlink', 'usr/bin', '/bin', '--symlink', 'usr/lib', '/lib',
           '--symlink', 'usr/lib', '/lib64', '--proc', '/proc', '--dev', '/dev', '--dir', '/etc', '--tmpfs', '/run',
           '--bind', str(scratch), '/tmp', '--ro-bind', str(marker), '/omabit-fleet-cli-fixture',
           '--ro-bind', str(root), str(root), '--ro-bind', str(binary), '/test', '--chdir', str(root / 'backend/cmd/urbit-collector')]
for key, value in {'HOME': '/tmp/home', 'XDG_CONFIG_HOME': '/tmp/config', 'XDG_STATE_HOME': '/tmp/state',
                   'XDG_DATA_HOME': '/tmp/data', 'XDG_CACHE_HOME': '/tmp/cache', 'XDG_RUNTIME_DIR': '/tmp/run',
                   'PATH': '/usr/bin:/bin', 'LANG': 'C.UTF-8', 'OMABIT_FLEET_CLI_FIXTURE': '1'}.items():
    command += ['--setenv', key, value]
for name in ('passwd', 'group', 'nsswitch.conf', 'hosts'):
    command += ['--ro-bind', '/etc/' + name, '/etc/' + name]
pattern = '^Test(FleetApplicationCLI|ApplicationCLI|ApplicationShipCLI)'
command += ['--', '/test', '-test.run', pattern, '-test.timeout', '5m', '-test.count=1', '-test.v']
(out / 'command.json').write_text(json.dumps(command, indent=2) + '\n')
with (out / 'test.log').open('w') as log:
    result = subprocess.run(command, stdout=log, stderr=subprocess.STDOUT, timeout=330)
sha = hashlib.sha256(binary.read_bytes()).hexdigest()
report = {'exit': result.returncode, 'test_pattern': pattern, 'binary_sha256': sha,
          'source_base': subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=root, text=True).strip(),
          'fake_secret_tool_sha256': hashlib.sha256(fake.read_bytes()).hexdigest(),
          'scope': 'offline synthetic runner, Eyre, receiver and keyring transport; no installed/live runtime or real keyring qualification'}
(out / 'result.json').write_text(json.dumps(report, indent=2) + '\n')
with binary.open('rb') as src, gzip.open(str(binary) + '.gz', 'wb') as dst:
    while data := src.read(1024 * 1024):
        dst.write(data)
with gzip.open(str(binary) + '.gz', 'rb') as src:
    assert hashlib.file_digest(src, 'sha256').hexdigest() == sha
binary.unlink()
print((out / 'test.log').read_text())
print('Evidence:', out)
sys.exit(result.returncode)
