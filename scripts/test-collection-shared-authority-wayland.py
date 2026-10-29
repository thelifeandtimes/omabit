#!/usr/bin/env python3
"""Shared admission against a private compositor; synthetic shell parents only."""
from pathlib import Path
import argparse
import hashlib
import importlib.util
import json
import os
import select
import shutil
import socket
import subprocess
import sys
import uuid


def save(path, value):
    path.write_text(json.dumps(value, indent=2) + '\n')


def receive(child):
    assert select.select([child.stdout], [], [], 8)[0], 'owned child response deadline'
    line = child.stdout.readline()
    assert line, 'owned child exited without response'
    return json.loads(line)


def send(child, value):
    child.stdin.write(json.dumps(value) + '\n')
    child.stdin.flush()


def guard():
    assert Path.home() == Path('/lab/home') and not Path('/home').exists() and not Path('/run/user').exists()
    assert [name for _, name in socket.if_nameindex()] == ['lo']
    assert all(os.readlink('/proc/self/ns/' + n) != v for n, v in json.loads(Path('/lab/host-namespaces.json').read_text()).items())


def worker():
    guard()
    sys.path.insert(0, '/lab/source')
    import admission as old
    import journal as j
    import runtime_authority as authority
    import shared_admission as shared
    children = []
    def emit(value):
        print(json.dumps(value), flush=True)
    emit({'shell': old.identity(os.getpid())})
    try:
        for line in sys.stdin:
            request = json.loads(line)
            command = request['command']
            if command == 'start':
                permit = request['permit']
                for role in j.ROLES:
                    member = permit['specification']['members'][role]
                    binding = {'permit': permit['id'], 'epoch': permit['epoch'],
                               'collection': j.digest(permit['specification']),
                               **{key: member[key] for key in ('plugin_id', 'profile', 'artifact_sha256')}}
                    child = subprocess.Popen([sys.executable, '-B', '/lab/source/shared_admission_guard.py', '--root', request['gate']],
                                             stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
                    children.append(child)
                    send(child, {'expected': binding, 'role': role, 'instance': uuid.uuid4().hex})
                emit({'guards': [receive(c) for c in children]})
            elif command == 'compete':
                base = Path('/lab/competitor')
                base.mkdir(mode=0o700)
                store = j.Store(base / 'coordinator', create=True)
                gate_path = base / 'gate'
                gate_path.mkdir(mode=0o700)
                gate = shared.Gate(gate_path)
                a = authority.Authority(request['ticket']['root'])
                failures = {}
                permit = dict(request['permit'], id=uuid.uuid4().hex, shell=old.identity(os.getpid()))
                try:
                    with store.lease() as lease:
                        try:
                            authority.Authority.provision(request['display'], store, lease)
                        except (OSError, ValueError) as error:
                            failures['provision'] = type(error).__name__
                        store.record(lease, {'type': 'register', 'spec': permit['specification']})
                        try:
                            a.reserve(store, lease, gate_path, permit)
                        except (OSError, ValueError) as error:
                            failures['reserve'] = type(error).__name__
                        try:
                            gate.issue(permit, request['ticket'])
                        except (OSError, ValueError) as error:
                            failures['issue'] = type(error).__name__
                    emit({'failures': failures, 'gate_written': (gate_path / 'admission.json').exists(), 'permit': permit})
                finally:
                    a.close()
                    gate.close()
                    store.close()
            elif command == 'await-fence':
                responses = [receive(c) for c in children]
                emit({'responses': responses, 'exits': [c.wait(timeout=5) for c in children]})
            elif command == 'quit':
                emit({'stopped': True})
                return
            else:
                raise ValueError('unknown qualification command')
    finally:
        for child in children:
            if child.poll() is None:
                child.kill()
                child.wait(timeout=5)
            for stream in (child.stdin, child.stdout, child.stderr):
                if stream:
                    stream.close()


def inside():
    guard()
    sys.path.insert(0, '/lab/source')
    import admission as old
    import journal as j
    import runtime_authority as authority
    import shared_admission as shared
    from wayland_runtime import Runtime
    runtime = Runtime()
    workers = []
    store = a = gate = None
    root = Path('/lab')
    try:
        runtime.start()
        display = root / 'runtime' / os.environ['WAYLAND_DISPLAY']
        store = j.Store(root / 'canonical-coordinator', create=True)
        spec = {'format': 1, 'name': 'stable', 'members': {
            role: {'plugin_id': old.CANONICAL[role] + '.qualification-stable', 'profile': 'qualification-stable',
                   'artifact_sha256': j.digest(['fixture', role]), 'endpoint_ref': None,
                   'paths': {kind: '/lab/shared/' + role + '/' + kind for kind in j.PATHS}}
            for role in j.ROLES}}
        with store.lease() as lease:
            a = authority.Authority.provision(display, store, lease)
            store.record(lease, {'type': 'register', 'spec': spec})
        assert a.descriptor['context']['server'] == old.identity(runtime.proc.pid)
        for slot in ('a', 'b'):
            log = (root / (slot + '.log')).open('w')
            child = subprocess.Popen([sys.executable, '-B', '/lab/driver.py', '--worker'],
                                     stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=log, text=True)
            workers.append((child, log, receive(child)['shell']))
        first, second = workers[0][0], workers[1][0]
        permit = {'id': uuid.uuid4().hex, 'request': uuid.uuid4().hex, 'epoch': uuid.uuid4().hex,
                  'specification': spec, 'shell': workers[0][2]}
        gate_path = root / 'first-gate'
        gate_path.mkdir(mode=0o700)
        def snapshot(stage):
            target = root / 'writer-states' / stage
            target.mkdir(parents=True, mode=0o700)
            for prefix, directory, names in (
                    ('authority', a.root, ('authority.json', 'ready.json', 'reservation.json')),
                    ('coordinator', store.root, ('journal.json', authority.MIRROR)),
                    ('gate', gate_path, ('admission.json', 'issued.json'))):
                for name in names:
                    path = directory / name
                    if path.exists():
                        shutil.copy2(path, target / (prefix + '-' + name))
        with store.lease() as lease:
            ticket = a.reserve(store, lease, gate_path, permit)
        snapshot('reserved-before-issue')
        gate = shared.Gate(gate_path)
        gate.issue(permit, ticket)
        snapshot('issued-before-claims')
        send(first, {'command': 'start', 'permit': permit, 'gate': str(gate_path)})
        admitted = receive(first)
        assert len(admitted['guards']) == 3 and all(v['ok'] and v['admission']['format'] == 2 for v in admitted['guards'])
        snapshot('three-active-claims')
        send(second, {'command': 'compete', 'permit': permit, 'ticket': ticket, 'display': str(display)})
        refused = receive(second)
        assert set(refused['failures']) == {'provision', 'reserve', 'issue'} and not refused['gate_written']
        assert refused['permit']['specification'] == permit['specification']
        assert refused['permit']['shell'] != permit['shell']
        live = gate.status()
        assert all(live['claim_owners_live'].values())
        # Retain, rather than delete, the exact marker to model missing authority.
        (a.root / 'ready.json').rename(a.root / 'retained-ready.json')
        send(first, {'command': 'await-fence'})
        fenced = receive(first)
        assert fenced['exits'] == [1, 1, 1] and all(not row['ok'] for row in fenced['responses'])
        gate.stop_admission(permit['id'], uuid.uuid4().hex)
        closed = gate.status()
        assert all(row['state'] == 'ended' for row in closed['record']['claims'].values())
        assert not any(closed['claim_owners_live'].values())
        snapshot('fenced-and-ended')
        for child, log, identity in workers:
            send(child, {'command': 'quit'})
            assert receive(child)['stopped'] and child.wait(timeout=5) == 0
        save(root / 'result.json', {'passed': True, 'applications_started': False, 'actual_private_compositor': True,
             'runtime_bound_to_compositor': a.descriptor['context']['server'], 'permit': permit, 'ticket': ticket,
             'admitted': admitted, 'competitor': refused, 'first_remained_live': live, 'fenced': fenced,
             'closed': closed, 'scope': 'initial-admission-only', 'handoff_supported': False})
        print('PASS: three shared guards, independent journal refused, authority loss fenced all guards; original formats retained')
    finally:
        for child, log, identity in workers:
            if child.poll() is None:
                child.kill()
                child.wait(timeout=5)
            log.close()
        if gate:
            gate.close()
        if a:
            a.close()
        if store:
            store.close()
        runtime.close()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--inside', action='store_true')
    parser.add_argument('--worker', action='store_true')
    parser.add_argument('--output', type=Path)
    parser.add_argument('--wayland-runtime', type=Path)
    args = parser.parse_args()
    if args.worker:
        return worker()
    if args.inside:
        return inside()
    out = args.output
    assert out and out.parent == Path('/tmp') and out.name.startswith('omabit-shared-authority-wayland-') and not out.exists()
    out.mkdir(mode=0o700)
    for name in ('home', 'source', 'runtime', 'config', 'state', 'data', 'cache'):
        (out / name).mkdir(mode=0o700)
    root = Path(__file__).resolve().parents[1]
    hashes = {}
    sources = [root / 'native/collections' / name for name in
               ('admission.py', 'admission_guard.py', 'artifacts.py', 'journal.py', 'runtime_authority.py',
                'shared_admission.py', 'shared_admission_guard.py')]
    sources += [root / 'scripts/qualification/collections/wayland_runtime.py']
    for path in sources:
        shutil.copy2(path, out / 'source' / path.name)
        hashes[str(path)] = hashlib.sha256(path.read_bytes()).hexdigest()
    shutil.copy2(__file__, out / 'driver.py')
    hashes[str(Path(__file__))] = hashlib.sha256(Path(__file__).read_bytes()).hexdigest()
    save(out / 'source.json', hashes)
    save(out / 'host-namespaces.json', {n: os.readlink('/proc/self/ns/' + n) for n in ('user', 'net', 'pid', 'mnt')})
    command = ['bwrap', '--unshare-all', '--die-with-parent', '--clearenv', '--ro-bind', '/usr', '/usr',
               '--symlink', 'usr/bin', '/bin', '--symlink', 'usr/lib', '/lib', '--symlink', 'usr/lib', '/lib64',
               '--proc', '/proc', '--dev', '/dev', '--tmpfs', '/tmp', '--bind', str(out), '/lab', '--chdir', '/lab']
    env = {'HOME': '/lab/home', 'PATH': '/usr/bin:/bin', 'XDG_RUNTIME_DIR': '/lab/runtime',
           'PYTHONDONTWRITEBYTECODE': '1', 'LANG': 'C.UTF-8', 'QT_QPA_PLATFORM': 'offscreen'}
    for name in ('CONFIG', 'STATE', 'DATA', 'CACHE'):
        env['XDG_' + name + '_HOME'] = '/lab/' + name.lower()
    for key, value in env.items():
        command += ['--setenv', key, value]
    command += ['--', '/usr/bin/python3', '-B', '/lab/driver.py', '--inside']
    spec = importlib.util.spec_from_file_location('private_wayland_runtime', sources[-1])
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    module.prepare(out, args.wayland_runtime, command, hashes)
    save(out / 'command.json', command)
    result = subprocess.run(command, capture_output=True, text=True, timeout=75)
    (out / 'driver.log').write_text(result.stdout + result.stderr)
    print(result.stdout + result.stderr)
    raise SystemExit(result.returncode)


if __name__ == '__main__':
    main()
