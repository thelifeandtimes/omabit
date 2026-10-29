#!/usr/bin/env python3
"""Format3 shared hosts through native loaders in an isolated Wayland runtime.

Synthetic app bodies isolate constructor admission. Real supported app bodies,
verified handoff and connected crash recovery are separate qualification gates.
"""
from pathlib import Path
import hashlib
import importlib.util
import json
import os
import shutil
import socket
import subprocess
import sys
import time
import uuid


def module(name, path):
    spec = importlib.util.spec_from_file_location(name, path)
    result = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(result)
    return result


def read(path):
    return json.loads(Path(path).read_text())


def save(path, value):
    Path(path).write_text(json.dumps(value, indent=2) + '\n')


def sha(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


def inside():
    root = Path('/lab')
    assert Path.home() == root / 'home' and not Path('/home').exists() and not Path('/run/user').exists()
    assert all(os.readlink('/proc/self/ns/' + n) != v for n, v in read(root / 'host-namespaces.json').items())
    assert [name for _, name in socket.if_nameindex()] == ['lo']
    sys.path.insert(0, '/lab/source')
    import admission as old
    import journal as j
    import runtime_authority as authority
    import shared_admission as admission
    import shared_artifacts as artifacts
    from wayland_runtime import Runtime
    runtime = Runtime()
    shells = []
    gate = a = store = legacy_gate = None
    prepared = read(root / 'prepared.json')['stable']
    events = []
    def start(enabled=False):
        env = dict(os.environ)
        if enabled:
            env['GATE_INITIAL_CONFIG'] = json.dumps({'version': 1,
                'plugins': [{'id': prepared[role]['1']['manifest']['id']} for role in j.ROLES],
                'bar': {'layout': {'left': [], 'center': [], 'right': []}}})
        log = (root / ('shell-' + str(len(shells)) + '.log')).open('wb')
        child = subprocess.Popen(['/usr/bin/quickshell', '--no-color', '--path', '/lab/shell.qml'],
                                 env=env, stdout=log, stderr=subprocess.STDOUT)
        shells.append((child, log))
        return child
    def call(child, target, method, *args, required=True):
        result = subprocess.run(['/usr/bin/quickshell', 'ipc', '--pid', str(child.pid), 'call', '--', target, method, *args],
                                capture_output=True, text=True, timeout=4)
        if required:
            assert result.returncode == 0, (method, result.stderr)
        return result.stdout.strip()
    def state(child):
        return json.loads(call(child, 'artifactProbe', 'state'))
    def wait(child, function):
        deadline = time.monotonic() + 12
        while time.monotonic() < deadline:
            assert child.poll() is None and runtime.proc.poll() is None
            value = function()
            if value:
                return value
            time.sleep(.04)
        save(root / 'failure-snapshot.json', state(child))
        raise AssertionError('shared native admission deadline')
    def ready(child):
        wait(child, lambda: call(child, 'artifactProbe', 'state', required=False).startswith('{'))
        wait(child, lambda: len(state(child)['selected']) == 9 and not state(child)['scanning'])
    def enable(child, value):
        for role in j.ROLES:
            assert call(child, 'shell', 'setPluginEnabled', prepared[role]['1']['manifest']['id'], str(value).lower()) == 'ok'
    def phase(child, expected, loaded):
        rows = state(child)['loaded']
        return rows if len(rows) == 3 and all(v['host']['phase'] == expected and (v['host']['loaded'] is not None) == loaded for v in rows.values()) else None
    def count(child):
        index = [item[0] for item in shells].index(child)
        return (root / ('shell-' + str(index) + '.log')).read_text().count('APPLICATION_CREATED ')
    def publish(child, selected_gate, include_authority=True):
        for role in j.ROLES:
            record = prepared[role]['1']
            manifest = dict(record['manifest'])
            activation = {'root': str(selected_gate.root), 'permit': permit['id'], 'epoch': permit['epoch'],
                          'collection': j.digest(permit['specification'])}
            if include_authority:
                activation['authority'] = ticket
            manifest['collectionActivation'] = activation
            save(root / 'home/.config/omarchy/plugins' / manifest['id'] / 'manifest.json', manifest)
        call(child, 'artifactProbe', 'rescan')
        wait(child, lambda: not state(child)['scanning'])
    try:
        runtime.start()
        first = start()
        ready(first)
        enable(first, True)
        rows = wait(first, lambda: phase(first, 'waiting', False))
        assert count(first) == 0
        events.append({'case': 'no-activation', 'hosts': rows, 'apps': count(first)})
        enable(first, False)
        wait(first, lambda: not state(first)['loaded'])
        store = j.Store(root / 'canonical-coordinator', create=True)
        with store.lease() as lease:
            a = authority.Authority.provision(root / 'runtime' / os.environ['WAYLAND_DISPLAY'], store, lease)
        members = {}
        for role in j.ROLES:
            record = prepared[role]['1']
            members[role] = {'plugin_id': record['manifest']['id'], 'profile': record['descriptor']['profile'],
                             'artifact_sha256': record['artifact_sha256'], 'endpoint_ref': None,
                             'paths': {kind: '/lab/app/stable/' + role + '/' + kind for kind in j.PATHS}}
        spec = {'format': 1, 'name': 'stable', 'members': members}
        permit = {'id': uuid.uuid4().hex, 'request': uuid.uuid4().hex, 'epoch': uuid.uuid4().hex,
                  'specification': spec, 'shell': old.identity(first.pid)}
        path = root / 'shared-gate'
        path.mkdir(mode=0o700)
        with store.lease() as lease:
            store.record(lease, {'type': 'register', 'spec': spec})
            ticket = a.reserve(store, lease, path, permit)
        gate = admission.Gate(path)
        gate.issue(permit, ticket)
        publish(first, gate, include_authority=False)
        enable(first, True)
        rows = wait(first, lambda: phase(first, 'fenced', False))
        assert count(first) == 0 and not gate.status()['record']['claims']
        events.append({'case': 'missing-authority-metadata', 'hosts': rows, 'apps': count(first)})
        enable(first, False)
        wait(first, lambda: not state(first)['loaded'])
        path = root / 'legacy-gate'
        path.mkdir(mode=0o700)
        legacy_gate = old.Gate(path)
        legacy_gate.issue(permit)
        publish(first, legacy_gate)
        enable(first, True)
        rows = wait(first, lambda: phase(first, 'fenced', False))
        assert count(first) == 0 and not legacy_gate.status()['record']['claims']
        events.append({'case': 'legacy-permit-refused', 'hosts': rows, 'apps': count(first)})
        enable(first, False)
        wait(first, lambda: not state(first)['loaded'])
        publish(first, gate)
        enable(first, True)
        rows = wait(first, lambda: phase(first, 'loaded', True))
        wait(first, lambda: all(row['helper'] == '1' for row in state(first)['loaded'].values()))
        wait(first, lambda: len(state(first)['widgets']) == 3 and all(v['loaded'] and v['admitted'] for v in state(first)['widgets'].values()))
        assert count(first) == 3
        for role in j.ROLES:
            member = members[role]
            host = rows[member['plugin_id']]['host']
            assert host['grant']['format'] == 2 and host['grant']['authority'] == ticket
            assert old.identity(host['grant']['owner']['pid']) == host['grant']['owner']
            generation = root / 'home/.config/omarchy/plugins' / member['plugin_id'] / 'generations' / member['artifact_sha256']
            assert artifacts.verify(generation, member['artifact_sha256'])['shared_authority_capable']
            artifacts.observation(generation, member['artifact_sha256'], host['loaded'])
        events.append({'case': 'shared-permit-admitted', 'hosts': rows, 'widgets': state(first)['widgets'], 'apps': count(first)})
        second = start(enabled=True)
        ready(second)
        denied = wait(second, lambda: phase(second, 'fenced', False))
        assert count(second) == 0 and count(first) == 3
        assert all(gate.status()['claim_owners_live'].values())
        events.append({'case': 'competing-native-shell-refused', 'hosts': denied, 'apps': count(second)})
        call(second, 'artifactProbe', 'stop')
        assert second.wait(timeout=5) == 0
        (a.root / 'ready.json').rename(a.root / 'retained-ready.json')
        fenced = wait(first, lambda: phase(first, 'fenced', True))
        wait(first, lambda: all(v['host']['transport']['closed'] for v in state(first)['loaded'].values()))
        wait(first, lambda: all(not v['admitted'] for v in state(first)['widgets'].values()))
        wait(first, lambda: all(v['state'] == 'ended' for v in gate.status()['record']['claims'].values()))
        assert count(first) == 3
        events.append({'case': 'authority-loss-fences-loaded-apps', 'hosts': fenced, 'widgets': state(first)['widgets'], 'apps': count(first)})
        gate.stop_admission(permit['id'], uuid.uuid4().hex)
        enable(first, False)
        wait(first, lambda: not state(first)['loaded'] and not state(first)['widgets'])
        call(first, 'artifactProbe', 'stop')
        assert first.wait(timeout=5) == 0
        for index in range(len(shells)):
            text = (root / ('shell-' + str(index) + '.log')).read_text()
            assert not any(word in text for word in ('TypeError', 'ReferenceError', 'Binding loop', 'service plugin load failed', 'APPLICATION_BAD_ID', 'WIDGET_BAD_ID')), text[-5000:]
        save(root / 'result.json', {'passed': True, 'events': events, 'artifact_format': 3, 'admission_format': 2,
                                  'native_registry_loaders': True, 'synthetic_bodies': True, 'real_plugins': False,
                                  'shared_authority': True, 'handoff_supported': False,
                                  'final_gate': gate.status(), 'shell_exits': [p.returncode for p, _ in shells]})
        print('PASS: format3 native hosts load zero apps without shared admission; valid three load, competing shell and authority loss fence')
    finally:
        for child, log in shells:
            if child.poll() is None:
                child.terminate()
                try:
                    child.wait(timeout=5)
                except subprocess.TimeoutExpired:
                    child.kill()
                    child.wait(timeout=4)
            log.close()
        for item in (gate, legacy_gate, a, store):
            if item:
                item.close()
        runtime.close()


def main():
    if '--inside' in sys.argv:
        return inside()
    at = sys.argv.index('--wayland-runtime')
    bundle = Path(sys.argv[at + 1])
    del sys.argv[at:at + 2]
    root = Path(__file__).resolve().parents[1]
    sys.path.insert(0, str(root / 'native/collections'))
    import shared_artifacts
    base = module('shared_native_base', Path(__file__).with_name('test-collection-admission-native.py'))
    runtime = module('shared_native_wayland', root / 'scripts/qualification/collections/wayland_runtime.py')
    original = subprocess.run
    def launch(command, *args, **kwargs):
        assert command[0] == 'bwrap'
        out = Path(command[command.index('--bind') + 1])
        hashes = read(out / 'source.json')
        for name in ('shared_artifacts.py', 'runtime_authority.py', 'shared_admission.py', 'shared_admission_guard.py'):
            source = root / 'native/collections' / name
            shutil.copy2(source, out / 'source' / name)
            hashes[str(source)] = sha(source)
        source = root / 'scripts/qualification/collections/wayland_runtime.py'
        shutil.copy2(source, out / 'source' / source.name)
        hashes[str(source)] = sha(source)
        source = Path(__file__).with_name('test-collection-admission-native.py')
        hashes[str(source)] = sha(source)
        shutil.copy2(source, out / 'admission-native-base.py')
        # The base supplies its command delimiter after the environment.
        runtime.prepare(out, bundle, command, hashes)
        save(out / 'source.json', hashes)
        save(out / 'command.json', command)
        return original(command, *args, **dict(kwargs, timeout=110))
    subprocess.run = launch
    base.__file__ = __file__
    sys.argv.append('--guarded')
    base.main(artifact_module=shared_artifacts)


if __name__ == '__main__':
    main()
