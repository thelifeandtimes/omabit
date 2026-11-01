#!/usr/bin/env python3
"""Run the existing real OpenSSH authorization suite in an offline namespace."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import socket
import subprocess


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--inside', action='store_true')
    parser.add_argument('--bin-dir', type=Path)
    parser.add_argument('--output', type=Path)
    args = parser.parse_args()
    if args.inside:
        assert os.environ.get('OMABIT_SSH_REGRESSION_LAB') == 'private-v1'
        assert os.environ['HOME'] == '/lab/home' and not Path('/home').exists() and not Path('/run/user').exists()
        expected = json.loads(Path('/lab/host-namespaces.json').read_text())
        assert all(os.readlink('/proc/self/ns/' + n) != expected[n] for n in expected)
        assert [n for _, n in socket.if_nameindex()] == ['lo']
        routes = subprocess.check_output(['/usr/bin/ip', 'route', 'show'], text=True)
        assert not routes
        result = subprocess.run(['/lab/bin/managed.test', '-test.v', '-test.run=^TestOpenSSHIntegration$', '-test.timeout=90s'],
                                capture_output=True, text=True, timeout=100)
        Path('/lab/test.log').write_text(result.stdout + result.stderr)
        assert result.returncode == 0 and 'actual OpenSSH' in result.stdout and 'SKIP' not in result.stdout, result.stdout + result.stderr
        Path('/lab/result.json').write_text(json.dumps({'passed': True, 'actual_openssh': True,
            'actual_managed_server_fake_backend': True, 'namespaces': list(expected),
            'cases': ['two independent operator keys', 'observer restrictions', 'actor-scoped replay',
                      'client disconnect/reconnect', 'host-key mismatch', 'runner-ID mismatch',
                      'forwarding and arbitrary command refusal', 'live revocation on retained sessions']}, indent=2) + '\n')
        print(result.stdout)
        return
    if not args.bin_dir or not args.output:
        parser.error('--bin-dir and --output required')
    out = args.output.absolute()
    assert out.parent == Path('/tmp') and re.fullmatch(r'omabit-ssh-regression-[A-Za-z0-9-]+', out.name) and not out.exists()
    out.mkdir(mode=0o700)
    for name in ('etc', 'home', 'runtime', 'config', 'state', 'cache', 'data', 'bin'):
        (out / name).mkdir(mode=0o700)
    sources = {}
    for name in ('managed.test', 'urbit-collector', 'urbit-runner'):
        source = args.bin_dir / name
        assert source.is_file() and not source.is_symlink()
        shutil.copy2(source, out / 'bin' / name)
        sources[name] = hashlib.sha256(source.read_bytes()).hexdigest()
    (out / 'sources.json').write_text(json.dumps(sources, indent=2) + '\n')
    (out / 'etc/passwd').write_text(f'lab:x:{os.getuid()}:{os.getgid()}:Lab:/lab/home:/bin/sh\n')
    (out / 'etc/group').write_text(f'lab:x:{os.getgid()}:\n')
    (out / 'etc/nsswitch.conf').write_text('passwd: files\ngroup: files\nhosts: files\n')
    (out / 'host-namespaces.json').write_text(json.dumps({n:os.readlink('/proc/self/ns/' + n) for n in ('user', 'net', 'pid', 'mnt')}))
    shutil.copy2(__file__, out / 'driver.py')
    command = ['bwrap', '--unshare-all', '--die-with-parent', '--clearenv', '--ro-bind', '/usr', '/usr',
        '--symlink', 'usr/bin', '/bin', '--symlink', 'usr/lib', '/lib', '--symlink', 'usr/lib', '/lib64',
        '--proc', '/proc', '--dev', '/dev', '--tmpfs', '/tmp', '--ro-bind', str(out / 'etc'), '/etc',
        '--bind', str(out), '/lab', '--chdir', '/lab']
    environment = {'HOME':'/lab/home', 'PATH':'/usr/bin:/bin', 'LANG':'C.UTF-8',
        'XDG_RUNTIME_DIR':'/lab/runtime', 'XDG_CONFIG_HOME':'/lab/config', 'XDG_STATE_HOME':'/lab/state',
        'XDG_DATA_HOME':'/lab/data', 'XDG_CACHE_HOME':'/lab/cache', 'OMABIT_SSH_REGRESSION_LAB':'private-v1',
        'OMABIT_TEST_OPENSSH':'1', 'OMABIT_RUNNER_BINARY':'/lab/bin/urbit-runner'}
    for key, value in environment.items():
        command += ['--setenv', key, value]
    command += ['--', '/usr/bin/python3', '-B', '/lab/driver.py', '--inside']
    (out / 'command.json').write_text(json.dumps(command, indent=2) + '\n')
    result = subprocess.run(command, capture_output=True, text=True, timeout=110)
    (out / 'driver.log').write_text(result.stdout + result.stderr)
    print(result.stdout + result.stderr)
    raise SystemExit(result.returncode)


if __name__ == '__main__':
    main()
