"""Private VM fixture: retain client intent across a whole gateway replacement.

Cookies and application values remain on the disposable guest disk. Reports sent
outside the VM must use a separately audited public projection, never these files.
"""
from dataclasses import dataclass
import hashlib
import json
import os
from pathlib import Path
import re
import stat
import subprocess
import time

from docker_peer import Transport
from docker_runtime import DOCKER, Engine, Ship, identity, require_guest
from state_case import Recovery

ROOT = Path('/opt/omabit-lab')


def private_json(path, value):
    temporary = path.with_suffix('.tmp')
    with temporary.open('x') as stream:
        os.chmod(temporary, 0o600)
        json.dump(value, stream, indent=2)
        stream.write('\n')
        stream.flush()
        os.fsync(stream.fileno())
    temporary.replace(path)
    fd = os.open(path.parent, os.O_DIRECTORY)
    try:
        os.fsync(fd)
    finally:
        os.close(fd)


def private_read(path, limit):
    fd = os.open(path, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK)
    try:
        st = os.fstat(fd)
        if not stat.S_ISREG(st.st_mode) or st.st_nlink != 1 or st.st_uid != 0 or stat.S_IMODE(st.st_mode) != 0o600 or st.st_size > limit:
            raise RuntimeError('private client checkpoint file differs')
        with os.fdopen(os.dup(fd), 'rb') as stream:
            data = stream.read(limit + 1)
        if len(data) != st.st_size:
            raise RuntimeError('private client checkpoint changed')
        return data
    finally:
        os.close(fd)


def checkpoint(value, project, previous):
    if not isinstance(value, dict) or value.get('version') != 1 or value.get('project') != project or value.get('previous_gateway') != identity(previous):
        raise RuntimeError('private client checkpoint binding differs')
    if not isinstance(project, str) or not re.fullmatch('[0-9a-f]{32}', project):
        raise RuntimeError('private project checkpoint required')
    if value.get('ready') is not True or not re.fullmatch('/~/channel/omabit-state-[0-9a-f]{32}', value.get('channel', '')):
        raise RuntimeError('private subscription checkpoint missing')
    if type(value.get('cursor')) is not int or value['cursor'] < 0:
        raise RuntimeError('private subscription cursor differs')
    if not re.fullmatch('net:\\[[0-9]{1,20}\\]', value.get('previous_namespace', '')):
        raise RuntimeError('private namespace checkpoint differs')
    entries = value.get('entries')
    if not isinstance(entries, list) or len(entries) != 5 or any(not isinstance(e, str) or not re.fullmatch('omabit-state-[0-9a-f]{24}', e) for e in entries) or len(set(entries)) != 5:
        raise RuntimeError('private application checkpoint differs')
    if set(value.get('ships', {})) != {'zod', 'nec'}:
        raise RuntimeError('private member checkpoint differs')
    for ship in ('zod', 'nec'):
        row = value['ships'][ship]
        identity(row.get('container'))
        if row.get('resumed') is not (ship == 'nec') or not isinstance(row.get('pier_identity'), list) or len(row['pier_identity']) != 2 or any(type(n) is not int or n <= 0 for n in row['pier_identity']):
            raise RuntimeError('private pier checkpoint differs')
        if not re.fullmatch('[0-9a-f]{64}', row.get('cookie_sha256', '')):
            raise RuntimeError('private cookie binding differs')
    recovery = value.get('recovery')
    if not isinstance(recovery, dict) or recovery.get('ok') is not False or recovery.get('channels') != [value['channel']] or recovery.get('channels_deleted') != []:
        raise RuntimeError('private channel history differs')
    writes, observations = recovery.get('writes'), recovery.get('observations')
    if not isinstance(writes, list) or len(writes) != 1 or not isinstance(observations, list) or len(observations) != 2:
        raise RuntimeError('private application history differs')
    write, initial, queued = writes[0], observations[0], observations[1]
    if write.get('state') != 'acknowledged' or write.get('http_status') != 200 or write.get('phase') != 'queued-before-gateway-replacement' or write.get('entry') != entries[-1]:
        raise RuntimeError('private write outcome unresolved')
    if initial.get('via') != 'subscription' or initial.get('channel') != value['channel'] or initial.get('cursor') != value['cursor'] or initial.get('state') != {'revision': 4, 'entries': entries[:4]}:
        raise RuntimeError('private subscription observation differs')
    if queued.get('via') != 'scry' or queued.get('state') != {'revision': 5, 'entries': entries}:
        raise RuntimeError('private queued state differs')
    return value


def capture(engine, specs, transport, previous_recovery, initial_result):
    require_guest()
    project = specs['zod'].project
    previous = specs['zod'].anchor
    root = ROOT / 'results' / ('gateway-clients-' + project)
    root.mkdir(mode=0o700)
    value = {'version': 1, 'project': project, 'previous_gateway': previous, 'previous_namespace': initial_result['network_namespace'], 'ready': False, 'ships': {}, 'entries': []}
    path = root / 'checkpoint.json'
    recovery = None

    def save():
        private_json(path, value)

    def emit(stage, **unused):
        print('OMABIT_PRIVATE_CHAIN ' + json.dumps({'stage': 'gateway-client-' + stage}), flush=True)

    try:
        for ship, spec in specs.items():
            container = initial_result['ships'][ship]['container']
            spec.check(engine.inspect(container), container)
            engine.check_anchor(spec)
            cookie = engine.execute(['exec', previous, '/bin/cat', '/tmp/omabit-' + ship + '-cookie'])
            if not 0 < len(cookie) <= 8192 or b'\x00' in cookie:
                raise RuntimeError('private cookie checkpoint bound')
            with (root / (ship + '.cookie')).open('xb') as stream:
                os.chmod(stream.name, 0o600)
                stream.write(cookie)
                stream.flush()
                os.fsync(stream.fileno())
            st = spec.pier.stat()
            value['ships'][ship] = {'container': container, 'resumed': spec.resumed, 'pier_identity': [st.st_dev, st.st_ino], 'cookie_sha256': hashlib.sha256(cookie).hexdigest()}
            del cookie
        recovery = Recovery(transport.call, transport.events, save, emit)
        recovery.entries = previous_recovery.entries.copy()
        if len(recovery.entries) != 4 or previous_recovery.report.get('ok') is not True or previous_recovery.channel is not None:
            raise RuntimeError('original application recovery incomplete')
        value['recovery'] = recovery.report
        recovery.subscribe()
        recovery.write('queued-before-gateway-replacement')
        value.update(ready=True, channel=recovery.channel, cursor=recovery.cursor, entries=recovery.entries.copy())
        checkpoint(value, project, previous)
        save()
        # Intentionally retain this distinct channel for the replacement client.
        # The original Recovery instance has already deleted its own channels.
        emit('checkpointed')
    except Exception:
        if recovery and recovery.channel:
            recovery.close()
        raise


@dataclass(frozen=True)
class GatewayShip(Ship):
    operation: str = ''

    def __post_init__(self):
        super().__post_init__()
        if not self.resumed or not re.fullmatch('[0-9a-f]{32}', self.operation):
            raise ValueError('retained gateway member generation required')

    @property
    def name(self):
        return 'omabit-private-' + self.project + '-' + self.ship + '-gateway-' + self.operation


def replacement_namespace(current, guest, previous):
    # Namespace inode numbers identify live objects, not permanent generations.
    # The predecessor is already stopped; Docker gives the new exact container
    # network=none. A recycled historical number is not evidence of host access.
    for value in (current, guest, previous):
        if not isinstance(value, str) or not re.fullmatch(r'net:\[[0-9]{1,20}\]', value):
            raise RuntimeError('private namespace observation malformed')
    if current == guest or previous == guest:
        raise RuntimeError('retained gateway shares guest namespace')
    return current == previous


def exercise(environment, project, previous, anchor, operation, publication=None):
    require_guest()
    identity(previous)
    identity(anchor)
    if previous == anchor:
        raise ValueError('distinct retained gateway container required')
    if not isinstance(project, str) or not re.fullmatch('[0-9a-f]{32}', project) or not isinstance(operation, str) or not re.fullmatch('[0-9a-f]{32}', operation):
        raise ValueError('exact private project and operation required')
    root = ROOT / 'results' / ('gateway-clients-' + project)
    saved = checkpoint(json.loads(private_read(root / 'checkpoint.json', 1024 * 1024)), project, previous)
    engine = Engine(environment)
    ship_type = GatewayShip
    if publication is not None:
        from docker_publication import Publication, PublishedEngine
        from docker_published_client import PublishedGatewayShip
        if not isinstance(publication, Publication) or publication.project != project:
            raise ValueError('retained peer publication binding differs')
        engine = PublishedEngine(environment, publication)
        ship_type = PublishedGatewayShip
    specs = {s: ship_type(project, s, identity(anchor), 25432, True, operation) for s in ('zod', 'nec')}
    report = {'version': 1, 'ok': False, 'project': project, 'previous_gateway': previous, 'gateway': anchor, 'operation': operation, 'ships': {}, 'stop_records': [], 'cleanup_errors': [], 'writes_replayed': False}
    transport = Transport(engine, specs)
    if publication is not None:
        from docker_published_client import PublishedTransport
        transport = PublishedTransport(engine, specs, publication, root)
        report['publication'] = {'network': publication.network, 'ports': publication.ports, 'requests': transport.requests, 'direct_client': True}
    recovery = None

    def save():
        private_json(ROOT / 'results/gateway-peer-recovery.json', report)

    def emit(stage, **unused):
        print('OMABIT_PRIVATE_CHAIN ' + json.dumps({'stage': 'gateway-peer-' + stage}), flush=True)

    def ready(ship):
        spec = specs[ship]
        container = report['ships'][ship]['container']
        deadline = time.monotonic() + 240
        while time.monotonic() < deadline:
            state = spec.check(engine.inspect(container), container)['State']
            if not state['Running'] or state['OOMKilled']:
                raise RuntimeError('retained gateway peer exited')
            try:
                status, body = transport.call(ship, '/~/host', timeout=3)
            except (RuntimeError, subprocess.TimeoutExpired):
                status, body = 0, b''
            if status == 200 and body.strip() == ('~' + ship).encode():
                break
            time.sleep(3)
        else:
            raise RuntimeError('retained gateway peer readiness deadline')
        ports = [int(row[0]) for row in (line.split() for line in (spec.pier / '.http.ports').read_text().splitlines()) if row[1:] == ['insecure', 'loopback']]
        if len(ports) != 1:
            raise RuntimeError('retained gateway peer administrative port differs')
        transport.admin[ship] = ports[0]
        status, body = transport.call(ship, '/', json.dumps({'source': {'dojo': 'our'}, 'sink': {'stdout': None}}).encode(), content='application/json', admin=True)
        if status != 200 or json.loads(body).strip() != '~' + ship:
            raise RuntimeError('retained peer administrative identity differs')
        status, _ = transport.call(ship, '/~/scry/hood/kiln/pikes.json')
        if status != 200:
            raise RuntimeError('original peer session did not survive')
        namespace = engine.execute(['exec', container, '/usr/bin/readlink', '/proc/self/ns/net']).decode().strip()
        if namespace != report['network_namespace']:
            raise RuntimeError('retained peer namespace differs')
        report['ships'][ship]['ready'] = True
        save()
        emit('ready-' + ship)

    try:
        report['network_namespace'] = engine.execute(['exec', anchor, '/usr/bin/readlink', '/proc/self/ns/net']).decode().strip()
        report['guest_namespace'] = os.readlink('/proc/self/ns/net')
        report['previous_namespace'] = saved['previous_namespace']
        report['historical_namespace_number_repeated'] = replacement_namespace(report['network_namespace'], report['guest_namespace'], report['previous_namespace'])
        engine.check_anchor(specs['zod'])
        for ship, spec in specs.items():
            old = saved['ships'][ship]
            original = Ship(project, ship, previous, 25432, old['resumed'])
            old_state = original.check(engine.inspect(old['container']), old['container'])['State']
            if old_state['Running'] or old_state['ExitCode'] != 0 or old_state['OOMKilled']:
                raise RuntimeError('original peer is not cleanly stopped')
            st = spec.pier.stat()
            if [st.st_dev, st.st_ino] != old['pier_identity']:
                raise RuntimeError('retained pier object changed')
            cookie = private_read(root / (ship + '.cookie'), 8192)
            if hashlib.sha256(cookie).hexdigest() != old['cookie_sha256']:
                raise RuntimeError('retained session changed')
            engine.check_anchor(spec)
            command = DOCKER + ['exec', '-i', anchor, '/bin/sh', '-c', 'umask 077; cat > "$1"', 'private-client-cookie', '/tmp/omabit-' + ship + '-cookie']
            copied = subprocess.run(command, input=cookie, capture_output=True, env=engine.environment, timeout=15)
            del cookie
            if copied.returncode or copied.stdout or copied.stderr:
                raise RuntimeError('private session restoration failed')
            container = engine.create(spec)
            report['ships'][ship] = {'container': container, 'previous_container': old['container'], 'pier_identity': old['pier_identity'], 'ready': False}
            save()
            engine.start(spec, container)
            ready(ship)
        if publication is not None:
            report['publication']['refusals'] = transport.check_unpublished()
        report['original_sessions_survived'] = True
        for ship, other in [('zod', 'nec'), ('nec', 'zod')]:
            status, _ = transport.call(ship, '/~/scry/hood/kiln/pikes.json', auth=other)
            if status not in (401, 403):
                raise RuntimeError('crossed retained session accepted')
        report['cross_ship_sessions_refused'] = True
        recovery = Recovery(transport.call, transport.events, save, emit)
        recovery.entries = saved['entries'].copy()
        recovery.channel = saved['channel']
        recovery.cursor = saved['cursor']
        recovery.report = saved['recovery']
        recovery.report['resume_channel'] = saved['channel']
        recovery.report['resume_cursor'] = saved['cursor']
        report['application_state'] = recovery.report
        report['baseline_entries'] = saved['entries'][:4]
        recovery.after_restart()
        report['transport_passed'] = True
    except Exception as error:
        report['error_type'] = type(error).__name__
        report['error_sha256'] = hashlib.sha256(str(error).encode()).hexdigest()
    finally:
        if recovery and recovery.channel:
            try:
                recovery.close()
            except Exception as error:
                report['cleanup_errors'].append({'operation': 'channel-delete', 'type': type(error).__name__})
        for ship in reversed(list(report['ships'])):
            try:
                record = {}
                report['stop_records'].append(record)
                engine.stop(specs[ship], report['ships'][ship]['container'], record, save, lambda: False)
                st = specs[ship].pier.stat()
                if [st.st_dev, st.st_ino] != saved['ships'][ship]['pier_identity']:
                    raise RuntimeError('retained pier changed after stop')
            except Exception as error:
                report['cleanup_errors'].append({'operation': 'stop-' + ship, 'type': type(error).__name__})
        report['ok'] = report.get('transport_passed') is True and 'error_type' not in report and not report['cleanup_errors']
        save()
    if not report['ok']:
        raise RuntimeError('retained gateway peer recovery failed')
    emit('complete')
    return {key: report[key] for key in ('ok', 'project', 'previous_gateway', 'gateway', 'operation', 'original_sessions_survived', 'cross_ship_sessions_refused', 'writes_replayed')}
