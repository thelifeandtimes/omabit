#!/usr/bin/python3
"""Public collector + real receivers/consumers/Secret Service, synthetic ship.

No installed source, host bus/keyring, runner, network or desktop is mounted.
Both capability declarations are explicitly staged for qualification only.
"""
import argparse
import gzip
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import resource
import shutil
import socket
import subprocess
import sys
import time


def save(path, value):
    path.write_text(json.dumps(value, indent=2) + '\n')


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


WRAPPER = '''import contextlib,io,sys
from pathlib import Path
import application_connection as receiver
from instance import INSTANCE
fault=Path('/lab/faults')/INSTANCE.plugin_id
with (Path('/lab/evidence')/('wire-'+INSTANCE.plugin_id+'.log')).open('a') as log:
    log.write((sys.argv[1] if len(sys.argv)>1 else 'initial')+'\\n')
if sys.argv[1:]==['--replacement-status'] and fault.exists():
    raise SystemExit(75)
with contextlib.redirect_stdout(io.StringIO()) as output:
    code=receiver.main()
if sys.argv[1:]==['--replace'] and fault.exists():
    raise SystemExit(75)
print(output.getvalue(),end='',flush=True)
raise SystemExit(code)
'''


def inside():
    resource.setrlimit(resource.RLIMIT_CORE, (0, 0))
    root = Path('/lab')
    assert Path.home() == root / 'home' and not Path('/home').exists() and not Path('/run/user').exists()
    assert os.environ.get('OMABIT_REAL_RECONNECT') == 'private-v1'
    assert [name for _, name in socket.if_nameindex()] == ['lo']
    for name, original in json.loads((root / 'namespaces.json').read_text()).items():
        assert os.readlink('/proc/self/ns/' + name) != original
    keyring = subprocess.Popen(['/usr/bin/gnome-keyring-daemon', '--foreground', '--unlock', '--components=secrets'],
                               stdin=subprocess.PIPE, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    try:
        keyring.stdin.write(os.urandom(32).hex().encode())
        keyring.stdin.close()
        def bus(method):
            return subprocess.run(['/usr/bin/busctl', '--user', 'call', 'org.freedesktop.DBus',
                                   '/org/freedesktop/DBus', 'org.freedesktop.DBus', method, 's',
                                   'org.freedesktop.secrets'], capture_output=True, text=True, timeout=3)
        deadline = time.monotonic() + 10
        while bus('NameHasOwner').stdout.strip() != 'b true':
            assert keyring.poll() is None and time.monotonic() < deadline
            time.sleep(.05)
        assert bus('GetConnectionUnixProcessID').stdout.strip() == 'u ' + str(keyring.pid)
        save(root / 'keyring-startup.json', {'private_owned_daemon': True, 'pid': keyring.pid})
        result = subprocess.run(['/test', '-test.run', '^TestFleetRealReceiverReconnect$', '-test.v',
                                 '-test.timeout', '5m', '-test.count=1'], timeout=310)
        return result.returncode
    finally:
        keyring.terminate()
        keyring.wait(5)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--inside', action='store_true')
    parser.add_argument('--theme')
    parser.add_argument('--dms')
    parser.add_argument('--out')
    args = parser.parse_args()
    if args.inside:
        return inside()
    root = Path(__file__).resolve().parents[3]
    out = Path(args.out).absolute()
    assert out.parent == Path('/tmp') and out.name.startswith('omabit-real-reconnect-') and not out.exists()
    spec = importlib.util.spec_from_file_location('storage', root / 'scripts/qualification/app-development/storage.py')
    storage = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(storage)
    admission = storage.admit(400 * 1024**2)
    out.mkdir(mode=0o700)
    save(out / 'storage-admission.json', admission)
    for name in ('home', 'config', 'data', 'state', 'cache', 'runtime', 'etc', 'faults', 'evidence'):
        (out / name).mkdir(mode=0o700)
    installed = out / 'home/.config/omarchy/plugins'
    installed.mkdir(parents=True)
    provenance = {'sources': {}, 'adaptations': ['private manifest/profile', 'declaration version2',
        'bound lost-ack wrapper in retained inventory', 'Theme inert desktop adapter and explicitly enabled fixture account'],
        'scope': 'actual collector public CLI, real receivers, Theme sync loop, DMs main loop, private Secret Service; synthetic runner and Eyre'}
    for source, plugin in ((Path(args.theme).resolve(), 'thelifeandtimes.urbit-theme'),
                           (Path(args.dms).resolve(), 'urbit.dms')):
        plugin += '.reconnect-lab'
        target = installed / plugin
        target.mkdir()
        files = list((source / 'client').glob('*.py')) if 'theme' in plugin else list(source.glob('*.py'))
        files += [source / 'manifest.json', source / 'application-connection.json']
        for file in files:
            assert file.is_file() and not file.is_symlink()
            dest = target / file.relative_to(source)
            dest.parent.mkdir(exist_ok=True, parents=True)
            shutil.copy2(file, dest)
        provenance['sources'][str(source)] = {'head': subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=source, text=True).strip(),
            'files': {str(file.relative_to(source)): sha(file) for file in files}}
        manifest = json.loads((target / 'manifest.json').read_text())
        manifest.update(id=plugin, name=manifest['name'] + ' (private reconnect qualification)')
        save(target / 'manifest.json', manifest)
        save(target / 'application-profile.json', {'version': 1, 'profile': 'reconnect-lab'})
        declaration = json.loads((target / 'application-connection.json').read_text())
        wrapper = Path(declaration['entry']).with_name('qualification_connection.py')
        (target / wrapper).write_text(WRAPPER)
        declaration.update(version=2, entry=str(wrapper))
        declaration['files'].append(str(wrapper))
        save(target / 'application-connection.json', declaration)
    # Portable profile fixture, not a desktop source or keyring stub.
    theme = Path(args.theme).resolve()
    profile_code = "import sys,json;sys.path[:0]=sys.argv[1:];from test_profile import profile;print(json.dumps(profile()))"
    profile = subprocess.check_output(['/usr/bin/python3', '-B', '-c', profile_code, str(theme), str(theme / 'tests')], text=True)
    (out / 'profile.json').write_text(profile)
    consumer = root / 'scripts/qualification/fleet/reconnect_consumer.py'
    shutil.copy2(consumer, out / 'consumer.py')
    shutil.copy2(__file__, out / 'driver.py')
    provenance.update(collector_head=subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=root, text=True).strip(),
                      fixture_sha256=sha(Path(__file__)), consumer_sha256=sha(consumer),
                      test_sha256=sha(root / 'backend/cmd/urbit-collector/fleet_real_reconnect_test.go'))
    save(out / 'source.json', provenance)
    save(out / 'namespaces.json', {name: os.readlink('/proc/self/ns/' + name) for name in ('user', 'net', 'pid', 'mnt', 'ipc', 'uts')})
    (out / 'etc/passwd').write_text(f'lab:x:{os.getuid()}:{os.getgid()}:Lab:/lab/home:/bin/sh\n')
    (out / 'etc/group').write_text(f'lab:x:{os.getgid()}:\n')
    binary = out / 'collector.test'
    build = subprocess.run(['go', 'test', '-race', '-c', '-o', str(binary), './backend/cmd/urbit-collector'], cwd=root,
        env={**os.environ, 'GOCACHE': '/tmp/omabit-040-go-cache', 'GOTOOLCHAIN': 'local', 'GOPROXY': 'off'},
        capture_output=True, text=True)
    (out / 'build.log').write_text(build.stdout + build.stderr)
    if build.returncode:
        print(build.stdout + build.stderr)
        return build.returncode
    command = ['bwrap', '--unshare-all', '--die-with-parent', '--clearenv', '--ro-bind', '/usr', '/usr',
        '--symlink', 'usr/bin', '/bin', '--symlink', 'usr/lib', '/lib', '--symlink', 'usr/lib', '/lib64',
        '--proc', '/proc', '--dev', '/dev', '--tmpfs', '/tmp', '--ro-bind', str(out / 'etc'), '/etc',
        '--bind', str(out), '/lab', '--ro-bind', str(binary), '/test', '--chdir', '/lab']
    env = {'HOME': '/lab/home', 'PATH': '/usr/bin:/bin', 'LANG': 'C.UTF-8', 'PYTHONDONTWRITEBYTECODE': '1',
           'OMABIT_REAL_RECONNECT': 'private-v1', 'XDG_RUNTIME_DIR': '/lab/runtime'}
    for name in ('CONFIG', 'DATA', 'STATE', 'CACHE'):
        env['XDG_' + name + '_HOME'] = '/lab/' + name.lower()
    for key, value in env.items():
        command += ['--setenv', key, value]
    command += ['--', '/usr/bin/dbus-run-session', '--', '/usr/bin/python3', '-B', '/lab/driver.py', '--inside']
    save(out / 'command.json', command)
    with (out / 'test.log').open('w') as log:
        result = subprocess.run(command, stdout=log, stderr=subprocess.STDOUT, timeout=330)
    digest = sha(binary)
    with binary.open('rb') as src, gzip.open(str(binary) + '.gz', 'wb') as dst:
        shutil.copyfileobj(src, dst)
    with gzip.open(str(binary) + '.gz', 'rb') as stream:
        assert hashlib.file_digest(stream, 'sha256').hexdigest() == digest
    binary.unlink()
    save(out / 'result.json', {'exit': result.returncode, 'binary_sha256': digest, 'scope': provenance['scope'],
                              'storage_after': storage.admit(0)['charged_bytes']})
    print((out / 'test.log').read_text())
    print('Evidence:', out)
    return result.returncode


if __name__ == '__main__':
    raise SystemExit(main())
