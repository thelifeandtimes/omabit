#!/usr/bin/env python3
"""Compare collectors over real OpenSSH in a disposable, no-external-network lab.

The runner RPC reports are synthetic and read-only. The forced-command bridge,
SSH clients/server and collectors are real binaries. No live ships or host SSH
configuration are mounted. CPU totals include enrollment and teardown and are not
steady-state laptop battery measurements. Output holds only fixture credentials.
"""
import argparse
import copy
from datetime import datetime, timedelta, timezone
import hashlib
import http.server
import json
import os
from pathlib import Path
import re
import resource
import selectors
import shutil
import signal
import socket
import socketserver
import subprocess
import threading
import time

ROOT = Path(__file__).resolve().parents[2] if os.environ.get("OMABIT_SSH_PERFORMANCE_LAB") != "private-v1" else Path("/lab")


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def run(args, **kwargs):
    return subprocess.run(args, check=True, capture_output=True, text=True, timeout=15, **kwargs).stdout.strip()


def inside(args):
    root = Path('/lab')
    assert os.environ.get('OMABIT_SSH_PERFORMANCE_LAB') == 'private-v1'
    assert os.environ['HOME'] == '/lab/home' and not Path('/home').exists() and not Path('/run/user').exists()
    ns = json.loads((root / 'host-namespaces.json').read_text())
    assert all(os.readlink('/proc/self/ns/' + n) != ns[n] for n in ns)
    assert [n for _, n in socket.if_nameindex()] == ['lo']
    assert 'UP' in json.loads(run(['/usr/bin/ip', '-j', 'address', 'show', 'lo']))[0]['flags']
    assert not run(['/usr/bin/ip', 'route', 'show'])
    features = ['ssh-key-grants', 'host-compatibility-v1', 'project-reservations-v1']
    report_template = json.loads((root / 'host-report.json').read_text())
    result = {'passed': False, 'isolated_namespaces': list(ns), 'synthetic_read_only_rpc': True,
              'real_openssh': True, 'real_forced_command_bridge': True, 'cases': []}
    def save():
        (root / 'result.json').write_text(json.dumps(result, indent=2) + '\n')
    save()
    for label, binary, interval, controlled, duration in [
        ('baseline-visible', '/lab/artifacts/baseline-collector', '2s', False, args.seconds),
        ('candidate-visible', '/lab/artifacts/candidate-collector', '2s', True, args.seconds),
        ('candidate-hidden', '/lab/artifacts/candidate-collector', '30s', True, max(35, args.seconds)),
    ]:
        case = root / label
        case.mkdir(mode=0o700)
        for name in ('host', 'client'):
            run(['/usr/bin/ssh-keygen', '-q', '-t', 'ed25519', '-N', '', '-f', str(case / name)])
        grant = run(['/lab/artifacts/runner', 'ssh-authorized-key', '--public-key', str(case / 'client.pub'), '--role', 'observer'])
        fingerprint = re.search(r'SHA256:[A-Za-z0-9+/]{43}', grant).group()
        (case / 'grants').write_text(grant + '\n')
        public = ' '.join((case / 'client.pub').read_text().split()[:2])
        forced = f'/lab/artifacts/runner ssh-bridge --socket {case}/rpc.sock --authorized-keys {case}/grants --fingerprint {fingerprint}'
        (case / 'authorized_keys').write_text(f'restrict,command="{forced}" {public}\n')
        counters = {'connections': 0, 'requests': {}}
        counts_lock = threading.Lock()
        class Server(socketserver.ThreadingMixIn, socketserver.UnixStreamServer):
            daemon_threads = True
        class Handler(http.server.BaseHTTPRequestHandler):
            protocol_version = 'HTTP/1.1'
            def setup(self):
                super().setup()
                with counts_lock:
                    counters['connections'] += 1
            def log_message(self, *_):
                pass
            def do_GET(self):
                assert self.headers.get('Urbit-Runner-SSH-Grant') == fingerprint
                with counts_lock:
                    counters['requests'][self.path] = counters['requests'].get(self.path, 0) + 1
                now = datetime.now(timezone.utc)
                stamp = now.isoformat().replace('+00:00', 'Z')
                value = {'runner_id': 'performance-fixture', 'mode': 'development-managed'}
                if self.path == '/hello':
                    value.update(protocol_version=1, features=features, actor='ssh-key:' + fingerprint, access_role='observer')
                elif self.path == '/v1/snapshot':
                    value.update(instances=[], operations=[], incidents=[])
                elif self.path == '/v1/projects':
                    value.update(schema_version=1, read_at=stamp, basis='stored-reservations', runtime_observation='not-performed', projects=[])
                elif self.path == '/v1/host-compatibility':
                    value = copy.deepcopy(report_template)
                    value['runner_id'] = 'performance-fixture'
                    value['report']['observed_at'] = stamp
                    value['metadata_observed_at'] = stamp
                    value['expires_at'] = (now + timedelta(seconds=5)).isoformat().replace('+00:00', 'Z')
                else:
                    raise AssertionError('Unexpected benchmark route ' + self.path)
                if self.path != '/hello':
                    time.sleep(.01)  # bounded response overlap, not a throughput benchmark
                data = json.dumps(value).encode()
                self.send_response(200)
                self.send_header('Urbit-Runner-Protocol', '1')
                self.send_header('Content-Type', 'application/json')
                self.send_header('Content-Length', str(len(data)))
                self.end_headers()
                self.wfile.write(data)
        server = Server(str(case / 'rpc.sock'), Handler)
        serving = threading.Thread(target=server.serve_forever, daemon=True)
        serving.start()
        probe = socket.socket()
        probe.bind(('127.0.0.1', 0))
        port = probe.getsockname()[1]
        probe.close()
        (case / 'sshd_config').write_text(f'''Port {port}
ListenAddress 127.0.0.1
HostKey {case}/host
PidFile {case}/sshd.pid
AuthorizedKeysFile {case}/authorized_keys
UsePAM no
PasswordAuthentication no
KbdInteractiveAuthentication no
AuthenticationMethods publickey
PermitRootLogin no
AllowUsers lab
DisableForwarding yes
PermitTTY no
PermitUserEnvironment no
LogLevel VERBOSE
''')
        host_public = ' '.join((case / 'host.pub').read_text().split()[:2])
        (case / 'known_hosts').write_text(f'[127.0.0.1]:{port} {host_public}\n')
        (case / 'ssh_config').write_text(f'''Host fixture
  HostName 127.0.0.1
  Port {port}
  User lab
  IdentityFile {case}/client
  IdentitiesOnly yes
  IdentityAgent none
  UserKnownHostsFile {case}/known_hosts
  GlobalKnownHostsFile /dev/null
''')
        run(['/usr/bin/sshd', '-t', '-f', str(case / 'sshd_config')])
        initial = resource.getrusage(resource.RUSAGE_CHILDREN)
        own_cpu = time.process_time()
        observations = []
        wake_seconds = None
        collector = None
        with (case / 'sshd.log').open('w') as log, (case / 'collector.log').open('w') as errors:
            daemon = subprocess.Popen(['/usr/bin/sshd', '-D', '-e', '-f', str(case / 'sshd_config')], stderr=log)
            try:
                end = time.monotonic() + 5
                while True:
                    try:
                        with socket.create_connection(('127.0.0.1', port), timeout=.1):
                            break
                    except OSError:
                        assert daemon.poll() is None and time.monotonic() < end, (case / 'sshd.log').read_text()
                        time.sleep(.02)
                prefix = [binary, '--config-dir', str(case / 'profile')]
                run(prefix + ['enroll', 'remote', '--ssh', 'fixture', '--ssh-config', str(case / 'ssh_config')])
                counts_before = copy.deepcopy(counters)
                sessions_before = (case / 'sshd.log').read_text().count('Accepted publickey')
                command = prefix + ['inventory', '--watch', '--interval', interval]
                if controlled:
                    command.append('--control-stdin')
                started = time.monotonic()
                collector = subprocess.Popen(command, stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=errors, text=True)
                selector = selectors.DefaultSelector()
                selector.register(collector.stdout, selectors.EVENT_READ)
                while time.monotonic() - started < duration:
                    assert collector.poll() is None, (case / 'collector.log').read_text()
                    if not selector.select(timeout=.1):
                        continue
                    row = json.loads(collector.stdout.readline())
                    assert row['status'] == 'available' and row['project_inventory_status'] == 'available' and row['host_compatibility_status'] == 'available', row
                    observations.append(time.monotonic() - started)
                watch_counts = {key: val - counts_before['requests'].get(key, 0) for key, val in counters['requests'].items()}
                ssh_sessions = (case / 'sshd.log').read_text().count('Accepted publickey') - sessions_before
                if label == 'candidate-hidden':
                    waking = time.monotonic()
                    collector.stdin.write(json.dumps({'interval':'2s','refresh':True}) + '\n')
                    collector.stdin.flush()
                    assert selector.select(timeout=3), 'opening did not wake the retained watch'
                    refreshed = json.loads(collector.stdout.readline())
                    wake_seconds = round(time.monotonic() - waking, 6)
                    assert refreshed['status'] == 'available' and wake_seconds < 1, refreshed
                    assert (case / 'sshd.log').read_text().count('Accepted publickey') - sessions_before == ssh_sessions
                selector.close()
                if controlled:
                    collector.stdin.close()
                else:
                    collector.send_signal(signal.SIGTERM)
                collector.wait(timeout=8)
                assert collector.returncode in (0, 1), collector.returncode
            finally:
                if collector and collector.poll() is None:
                    collector.kill()
                    collector.wait()
                daemon.terminate()
                daemon.wait(timeout=5)
                server.shutdown()
                server.server_close()
                serving.join(timeout=3)
        usage = resource.getrusage(resource.RUSAGE_CHILDREN)
        record = {'label': label, 'watch_seconds': round(time.monotonic() - started, 3), 'interval': interval,
                  'observation_seconds': [round(n, 3) for n in observations], 'watch_ssh_sessions': ssh_sessions,
                  'watch_rpc_requests': watch_counts, 'watch_local_connections_including_wake': counters['connections'] - counts_before['connections'],
                  'hidden_to_visible_refresh_seconds': wake_seconds,
                  'children_cpu_seconds_including_enrollment_teardown': round(usage.ru_utime + usage.ru_stime - initial.ru_utime - initial.ru_stime, 6),
                  'synthetic_server_driver_cpu_seconds': round(time.process_time() - own_cpu, 6)}
        assert observations and observations[0] < 5
        if label == 'candidate-visible':
            assert ssh_sessions <= 3, record
        if label == 'candidate-hidden':
            assert len(observations) <= 1 + int(duration / 30), record
        result['cases'].append(record)
        save()
    result['passed'] = True
    save()
    print(json.dumps(result, indent=2))


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--inside', action='store_true')
    p.add_argument('--baseline', type=Path)
    p.add_argument('--candidate', type=Path)
    p.add_argument('--runner', type=Path)
    p.add_argument('--output', type=Path)
    p.add_argument('--seconds', type=int, default=30)
    args = p.parse_args()
    assert 6 <= args.seconds <= 300
    if args.inside:
        return inside(args)
    if not all((args.baseline, args.candidate, args.runner, args.output)):
        p.error('baseline, candidate, runner and output are required')
    out = args.output.absolute()
    assert out.parent == Path('/tmp') and re.fullmatch(r'omabit-ssh-performance-[A-Za-z0-9-]+', out.name) and not out.exists()
    out.mkdir(mode=0o700)
    for name in ('etc', 'home', 'runtime', 'config', 'data', 'state', 'cache', 'artifacts'):
        (out / name).mkdir(mode=0o700)
    for name, path in [('baseline-collector', args.baseline), ('candidate-collector', args.candidate), ('runner', args.runner)]:
        assert path.is_file() and not path.is_symlink()
        shutil.copy2(path, out / 'artifacts' / name)
    (out / 'etc/passwd').write_text(f'lab:x:{os.getuid()}:{os.getgid()}:Lab:/lab/home:/bin/sh\n')
    (out / 'etc/group').write_text(f'lab:x:{os.getgid()}:\n')
    (out / 'etc/nsswitch.conf').write_text('passwd: files\ngroup: files\nhosts: files\n')
    (out / 'host-namespaces.json').write_text(json.dumps({n: os.readlink('/proc/self/ns/' + n) for n in ('user', 'net', 'pid', 'mnt')}))
    fixture = json.loads(run(['go', 'run', 'scripts/test-host-compatibility-fixtures.go'], cwd=ROOT))
    (out / 'host-report.json').write_text(json.dumps(fixture['pending']['host_compatibility']))
    shutil.copy2(__file__, out / 'driver.py')
    (out / 'sources.json').write_text(json.dumps({p.name: sha(p) for p in (out / 'artifacts').iterdir()}, indent=2) + '\n')
    cmd = ['bwrap', '--unshare-all', '--die-with-parent', '--clearenv',
           '--ro-bind', '/usr', '/usr', '--symlink', 'usr/bin', '/bin', '--symlink', 'usr/lib', '/lib', '--symlink', 'usr/lib', '/lib64',
           '--proc', '/proc', '--dev', '/dev', '--tmpfs', '/tmp', '--ro-bind', str(out / 'etc'), '/etc',
           '--bind', str(out), '/lab', '--ro-bind', str(out / 'artifacts'), '/lab/artifacts', '--chdir', '/lab']
    env = {'HOME': '/lab/home', 'XDG_RUNTIME_DIR': '/lab/runtime', 'XDG_CONFIG_HOME': '/lab/config', 'XDG_STATE_HOME': '/lab/state',
           'XDG_CACHE_HOME': '/lab/cache', 'XDG_DATA_HOME': '/lab/data', 'PATH': '/usr/bin:/bin', 'LANG': 'C.UTF-8', 'OMABIT_SSH_PERFORMANCE_LAB': 'private-v1'}
    for key, value in env.items():
        cmd += ['--setenv', key, value]
    cmd += ['--', '/usr/bin/python3', '-B', '/lab/driver.py', '--inside', '--seconds', str(args.seconds)]
    (out / 'command.json').write_text(json.dumps(cmd, indent=2) + '\n')
    completed = subprocess.run(cmd, capture_output=True, text=True, timeout=3 * args.seconds + 70)
    (out / 'driver.log').write_text(completed.stdout + completed.stderr)
    print(completed.stdout + completed.stderr)
    raise SystemExit(completed.returncode)


if __name__ == '__main__':
    main()
