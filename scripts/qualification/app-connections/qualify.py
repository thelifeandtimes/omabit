#!/usr/bin/env python3
"""Fresh fake-ship authentication qualification, without host ship access."""

import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import resource
import secrets
import shutil
import signal
import stat
import subprocess
import sys
import threading
import time
import urllib.request
import uuid


def digest(path):
    with path.open('rb') as stream:
        return hashlib.file_digest(stream, 'sha256').hexdigest()


def regular(path):
    info = path.lstat()
    if not stat.S_ISREG(info.st_mode) or info.st_nlink != 1:
        raise ValueError('artifact must be a regular file, without links: ' + str(path))


def check_namespace():
    guard = json.loads(Path('/run/omabit-auth-lab.json').read_text())
    for kind in ('network', 'pid'):
        ns = 'net' if kind == 'network' else kind
        if os.readlink('/proc/self/ns/' + ns) == guard['host_' + kind + '_namespace']:
            raise ValueError('refusing host ' + kind + ' namespace')
    if not guard['run_id'].startswith('omabit-auth-'):
        raise ValueError('invalid qualification run')
    if any(Path(p).exists() for p in ('/home', '/run/user', '/var/run/docker.sock')):
        raise ValueError('host resources are mounted')
    return guard


def runtime_limits():
    os.nice(15)
    resource.setrlimit(resource.RLIMIT_CPU, (240, 240))
    resource.setrlimit(resource.RLIMIT_CORE, (0, 0))


def inside(bus=False):
    guard = check_namespace()  # Must precede filesystem or process mutations.
    lab = Path('/lab')
    os.umask(0o077)
    plugins = Path('/artifacts/plugins').is_dir()
    if not bus:
        for name in ('home', 'state', 'config', 'data', 'cache', 'runtime'):
            (lab / name).mkdir(mode=0o700)
        if plugins:
            return subprocess.run(['dbus-run-session', '--', '/usr/bin/python3', '/artifacts/qualify.py', '--inside-bus'],
                                  check=False, timeout=480).returncode
    keyring = None
    if plugins:
        if not bus or not os.environ.get('DBUS_SESSION_BUS_ADDRESS'):
            raise ValueError('plugin tests require the isolated session bus')
        keyring = subprocess.Popen(['gnome-keyring-daemon', '--foreground', '--unlock', '--components=secrets'],
                                   stdin=subprocess.PIPE, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        keyring.stdin.write(secrets.token_hex(32).encode())
        keyring.stdin.close()
        saved = subprocess.run(['secret-tool', 'store', '--label=Omabit disposable lab', 'application', 'omabit-auth-lab'],
                               input=b'synthetic-lab-value', stdout=subprocess.PIPE, stderr=subprocess.DEVNULL, timeout=10)
        loaded = subprocess.run(['secret-tool', 'lookup', 'application', 'omabit-auth-lab'],
                                stdout=subprocess.PIPE, stderr=subprocess.DEVNULL, timeout=10)
        if saved.returncode or loaded.returncode or loaded.stdout.strip() != b'synthetic-lab-value':
            raise ValueError('private Secret Service did not become ready')
    result = dict(run_id=guard['run_id'], boot_ready=False, test_exit=None,
                  clean_shutdown=False, log_truncated=False, plugin_handoff=plugins)
    args = ['/artifacts/urbit', '-t', '-L', '--no-conn', '--no-dock', '--loom', '31',
            '--snap-time', '60', '-b', '127.0.0.1', '--http-port', '18081',
            '-F', 'zod', '-B', '/artifacts/boot.pill', '-c', '/lab/zod']
    (lab / 'launch.json').write_text(json.dumps(args, indent=2) + '\n')
    child = subprocess.Popen(args, stdin=subprocess.DEVNULL, stdout=subprocess.PIPE,
                             stderr=subprocess.STDOUT, start_new_session=True,
                             preexec_fn=runtime_limits)
    overflow = threading.Event()

    def capture():
        total = 0
        with (lab / 'vere.log').open('wb') as out:
            while chunk := child.stdout.read(65536):
                remaining = max(0, (8 << 20) - total)
                out.write(chunk[:remaining])
                total += len(chunk)
                if total > 8 << 20:
                    overflow.set()

    reader = threading.Thread(target=capture, daemon=True)
    reader.start()
    opener = urllib.request.build_opener(urllib.request.ProxyHandler({}))
    test = None
    try:
        deadline = time.monotonic() + 240
        while time.monotonic() < deadline and child.poll() is None and not overflow.is_set():
            try:
                with opener.open('http://127.0.0.1:18081/~/host', timeout=1) as response:
                    if response.read(256).strip() == b'~zod':
                        result['boot_ready'] = True
                        break
            except (OSError, ValueError):
                pass
            time.sleep(1)
        if not result['boot_ready']:
            if child.poll() is not None:
                raise RuntimeError('fresh fake exited before readiness')
            raise RuntimeError('fresh fake did not become ready within the lab deadline')
        print('Fresh isolated fake ready; running session lifecycle gate.', flush=True)
        with (lab / 'tests.log').open('wb') as out:
            runs = [['/artifacts/eyre.test', '-test.run', '^Test(IndependentSessions|ApplicationPluginHandoff)IsolatedLive$', '-test.v', '-test.timeout=180s']]
            if plugins:
                runs.append(['/artifacts/applicationauth.test', '-test.run', '^TestDurablePluginHandoffIsolatedLive$', '-test.v', '-test.timeout=180s'])
                runs.append(['/usr/bin/python3', '-B', '/artifacts/public_flow.py'])
            deadline = time.monotonic() + 185
            for test_command in runs:
                test = subprocess.Popen(test_command,
                                        stdout=out, stderr=subprocess.STDOUT,
                                        env=dict(os.environ, OMABIT_AUTH_LAB='1', OMABIT_PLUGIN_LAB='1' if plugins else '0', OMABIT_TEST_FAKE_ZOD='0'))
                while test.poll() is None:
                    if overflow.is_set() or time.monotonic() >= deadline or child.poll() is not None:
                        test.kill()
                        break
                    time.sleep(0.25)
                result['test_exit'] = test.wait()
                if result['test_exit']:
                    break
        print((lab / 'tests.log').read_text()[:16384], end='', flush=True)
    except (RuntimeError, OSError) as error:
        result['error'] = str(error)
    finally:
        if test is not None and test.poll() is None:
            test.kill()
            test.wait()
        was_running = child.poll() is None
        if was_running:
            child.send_signal(signal.SIGTERM)
        try:
            code = child.wait(timeout=45)
            result['clean_shutdown'] = was_running and code == 0
        except subprocess.TimeoutExpired:
            os.killpg(child.pid, signal.SIGKILL)  # Only this namespace's fresh child.
            child.wait()
        reader.join(timeout=3)
        if keyring is not None:
            keyring.terminate()
            try:
                keyring.wait(timeout=3)
            except subprocess.TimeoutExpired:
                keyring.kill()
                keyring.wait()
        result['log_truncated'] = overflow.is_set()
        (lab / 'result.json').write_text(json.dumps(result, indent=2) + '\n')
    print(json.dumps(result), flush=True)
    return 0 if result['test_exit'] == 0 and result['clean_shutdown'] and not result['log_truncated'] else 1


def stage_plugin(source, destination, kind):
    source = source.absolute()
    if (source / 'application-profile.json').exists():
        raise ValueError('stage from an ordinary source checkout, not an already profiled artifact')
    canonical = 'thelifeandtimes.urbit-theme' if kind == 'theme' else 'urbit.dms'
    paths = [source / 'manifest.json', source / 'application-connection.json']
    if kind == 'theme':
        paths += [source / 'Service.qml', source / 'Model.js']
    paths += sorted((source / 'client').glob('*.py')) if kind == 'theme' else sorted(p for p in source.glob('*.py') if not p.name.startswith('test_'))
    before = {p.relative_to(source).as_posix(): digest(p) for p in paths}
    destination.mkdir(parents=True)
    for path in paths:
        regular(path)
        out = destination / path.relative_to(source)
        out.parent.mkdir(parents=True, exist_ok=True)
        shutil.copyfile(path, out)
    manifest = json.loads((destination / 'manifest.json').read_text())
    if manifest['id'] != canonical:
        raise ValueError('unexpected plugin source identity')
    manifest['id'] += '.auth-lab'
    manifest['name'] += ' (Auth Lab)'
    if 'barWidget' in manifest:
        manifest['barWidget']['displayName'] += ' (Auth Lab)'
    (destination / 'manifest.json').write_text(json.dumps(manifest, indent=2) + '\n')
    (destination / 'application-profile.json').write_text('{"version":1,"profile":"auth-lab"}\n')
    if before != {p.relative_to(source).as_posix(): digest(p) for p in paths}:
        raise ValueError('plugin source changed while staging')
    return {'source': str(source), 'source_sha256': before, 'plugin_id': manifest['id'], 'staged_ui': ['Service.qml', 'Model.js'] if kind == 'theme' else [], 'desktop_adapter_enabled': False}


UI_COMPONENTS = ('ApplicationConnections.qml', 'ApplicationsPane.qml', 'ActionButton.qml', 'SelectField.qml', 'Field.qml', 'CheckField.qml', 'Label.qml', 'Hint.qml', 'Design.qml')


def stage_ui(root, destination):
    plugin = destination / 'Plugin'
    plugin.mkdir(parents=True)
    for name in UI_COMPONENTS:
        shutil.copyfile(root / 'plugins/omabit' / name, plugin / name)
    (plugin / 'qmldir').write_text('singleton Design 1.0 Design.qml\n' + ''.join(name[:-4] + ' 1.0 ' + name + '\n' for name in UI_COMPONENTS if name != 'Design.qml'))
    commons = destination / 'Commons'
    commons.mkdir()
    (commons / 'qmldir').write_text('singleton Style 1.0 Style.qml\nsingleton Color 1.0 Color.qml\n')
    (commons / 'Style.qml').write_text('pragma Singleton\nimport QtQuick\nQtObject { function space(n) {return n} property var font: ({family:"sans-serif",caption:12,bodySmall:14,body:14}) }\n')
    (commons / 'Color.qml').write_text('pragma Singleton\nimport QtQuick\nQtObject {property color background:"#171a1e";property color foreground:"#e7e9eb";property color accent:"#8ab4f8";property color urgent:"#ed8994"}\n')
    (destination / 'fonts.conf').write_text('<?xml version="1.0"?><!DOCTYPE fontconfig SYSTEM "urn:fontconfig:fonts.dtd"><fontconfig><dir>/usr/share/fonts</dir><cachedir>/lab/ui-lab/cache/fontconfig</cachedir></fontconfig>\n')
    shutil.copyfile(Path(__file__).with_name('approval.qml'), destination / 'shell.qml')


def host(args):
    root = Path(__file__).resolve().parents[3]
    output = args.output.absolute()
    if output.parent != Path('/tmp') or not re.fullmatch(r'omabit-auth-run-[a-zA-Z0-9-]{1,80}', output.name):
        raise ValueError('output must be a fresh /tmp/omabit-auth-run-NAME directory')
    if not output.parent.is_dir() or output.exists() or output.is_symlink():
        raise ValueError('output must be a new directory beneath an existing parent')
    for path in (args.vere, args.pill):
        regular(path)
    metadata = (root / 'internal/runtimecandidate/metadata.go').read_text()
    expected = re.search(r'PillSHA256\s*=\s*"([a-f0-9]{64})"', metadata).group(1)
    size = int(re.search(r'PillBytes\s*=\s*int64\((\d+)\)', metadata).group(1))
    if args.pill.stat().st_size != size or digest(args.pill) != expected:
        raise ValueError('pill does not match repository-pinned bytes and SHA-256')
    os.umask(0o077)
    output.mkdir(mode=0o700)
    artifacts = output / 'artifacts'
    artifacts.mkdir()
    (output / 'lab').mkdir()
    shutil.copyfile(args.vere, artifacts / 'urbit')
    (artifacts / 'urbit').chmod(0o700)
    shutil.copyfile(args.pill, artifacts / 'boot.pill')
    if (artifacts / 'boot.pill').stat().st_size != size or digest(artifacts / 'boot.pill') != expected:
        raise ValueError('pill changed while copying; refusing to boot')
    shutil.copyfile(__file__, artifacts / 'qualify.py')
    plugin_sources = {}
    if args.theme_source is not None:
        plugin_sources['theme'] = stage_plugin(args.theme_source, artifacts / 'plugins/theme', 'theme')
        plugin_sources['dms'] = stage_plugin(args.dms_source, artifacts / 'plugins/dms', 'dms')
        shutil.copyfile(Path(__file__).with_name('plugin_probe.py'), artifacts / 'plugin_probe.py')
        shutil.copyfile(Path(__file__).with_name('public_flow.py'), artifacts / 'public_flow.py')
        shutil.copyfile(Path(__file__).with_name('running_bridge.py'), artifacts / 'running_bridge.py')
        stage_ui(root, artifacts / 'ui')
    (artifacts / 'passwd').write_text(f'root:x:0:0:root:/root:/bin/false\nlab:x:{os.getuid()}:{os.getgid()}:Omabit Lab:/lab/home:/bin/false\n')
    (artifacts / 'group').write_text(f'root:x:0:\nlab:x:{os.getgid()}:\n')
    source_files = [p.relative_to(root).as_posix() for p in sorted((root / 'internal').rglob('*.go'))]
    source_files += [p.relative_to(root).as_posix() for p in sorted((root / 'backend/cmd/urbit-collector').glob('*.go'))]
    source_files += ['go.mod', 'go.sum']
    source_files += [p.relative_to(root).as_posix() for p in sorted(Path(__file__).parent.glob('*.py'))]
    source_files += ['scripts/qualification/app-connections/approval.qml']
    source_files += ['plugins/omabit/' + name for name in UI_COMPONENTS]
    before = {p: digest(root / p) for p in source_files}
    subprocess.run(['go', 'test', '-c', '-o', str(artifacts / 'eyre.test'), './internal/eyre'],
                   cwd=root, check=True, env=dict(os.environ, CGO_ENABLED='0', OMABIT_TEST_FAKE_ZOD='0'))
    if plugin_sources:
        subprocess.run(['go', 'test', '-c', '-o', str(artifacts / 'applicationauth.test'), './internal/applicationauth'],
                       cwd=root, check=True, env=dict(os.environ, CGO_ENABLED='0', OMABIT_TEST_FAKE_ZOD='0'))
    if plugin_sources:
        subprocess.run(['go', 'build', '-o', str(artifacts / 'urbit-collector'), './backend/cmd/urbit-collector'],
                       cwd=root, check=True, env=dict(os.environ, CGO_ENABLED='0', OMABIT_TEST_FAKE_ZOD='0'))
    if before != {p: digest(root / p) for p in source_files}:
        raise ValueError('qualification source changed during build; refusing to boot')
    guard = dict(run_id='omabit-auth-' + uuid.uuid4().hex,
                 host_network_namespace=os.readlink('/proc/self/ns/net'),
                 host_pid_namespace=os.readlink('/proc/self/ns/pid'))
    (output / 'guard.json').write_text(json.dumps(guard) + '\n')
    evidence = dict(guard, artifact_sha256={p.relative_to(artifacts).as_posix(): digest(p) for p in artifacts.rglob('*') if p.is_file()},
                    source_sha256=before, plugin_sources=plugin_sources)
    (output / 'inputs.json').write_text(json.dumps(evidence, indent=2) + '\n')
    command = ['systemd-run', '--user', '--scope', '--unit=' + guard['run_id'],
               '--property=MemoryMax=4G', '--property=MemorySwapMax=0',
               '--property=CPUQuota=100%', '--property=TasksMax=128',
               '--', 'bwrap', '--unshare-all', '--die-with-parent', '--clearenv',
               '--ro-bind', '/usr', '/usr', '--symlink', 'usr/bin', '/bin',
               '--symlink', 'usr/lib', '/lib', '--symlink', 'usr/lib', '/lib64',
               '--proc', '/proc', '--dev', '/dev', '--tmpfs', '/tmp', '--dir', '/run',
               '--ro-bind', str(artifacts), '/artifacts',
               '--ro-bind', str(artifacts / 'passwd'), '/etc/passwd',
               '--ro-bind', str(artifacts / 'group'), '/etc/group',
               '--ro-bind', str(output / 'guard.json'), '/run/omabit-auth-lab.json',
               '--bind', str(output / 'lab'), '/lab', '--chdir', '/lab',
               '--setenv', 'HOME', '/lab/home', '--setenv', 'PATH', '/usr/bin:/bin',
               '--setenv', 'LANG', 'C.UTF-8', '--setenv', 'XDG_CONFIG_HOME', '/lab/config',
               '--setenv', 'XDG_STATE_HOME', '/lab/state', '--setenv', 'XDG_DATA_HOME', '/lab/data',
               '--setenv', 'XDG_CACHE_HOME', '/lab/cache', '--setenv', 'XDG_RUNTIME_DIR', '/lab/runtime',
               '--', '/usr/bin/python3', '/artifacts/qualify.py', '--inside']
    (output / 'sandbox.json').write_text(json.dumps(command, indent=2) + '\n')
    print('Qualification evidence: ' + str(output), flush=True)
    return subprocess.run(command, check=False, timeout=500).returncode


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--inside', action='store_true', help=argparse.SUPPRESS)
    parser.add_argument('--inside-bus', action='store_true', help=argparse.SUPPRESS)
    parser.add_argument('--theme-source', type=Path)
    parser.add_argument('--dms-source', type=Path)
    parser.add_argument('--vere', type=Path)
    parser.add_argument('--pill', type=Path)
    parser.add_argument('--output', type=Path)
    args = parser.parse_args()
    if args.inside or args.inside_bus:
        return inside(bus=args.inside_bus)
    if (args.theme_source is None) != (args.dms_source is None):
        parser.error('supply both --theme-source and --dms-source')
    if any(value is None for value in (args.vere, args.pill, args.output)):
        parser.error('--vere, --pill and --output are required')
    return host(args)


if __name__ == '__main__':
    try:
        sys.exit(main())
    except (ValueError, OSError, subprocess.SubprocessError) as error:
        sys.exit(str(error))
