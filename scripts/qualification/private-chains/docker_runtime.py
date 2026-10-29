"""Guest-only keyed container lifecycle for private-chain qualification.

This is not a runner backend or a host Docker command. The caller owns a fresh
no-NIC VM, its exact rootless engine and a dedicated network-none anchor. Ship
keys are mounted only at creation; reports contain paths/IDs, never key contents.
"""
from dataclasses import dataclass
import json
import os
from pathlib import Path
import re
import subprocess

IMAGE = 'docker.io/tloncorp/vere@sha256:47fbcf0710ded939931ce3c6408864cee5e8848c5765da0b51a90ca0b1ab1afd'
IMAGE_ID = 'sha256:d6cc5107ed7fa92174e336af9c7f25ee433073bd9961e1c3ed23a225b866b245'
LABEL = 'org.omabit.qualification.private-chain'
DOCKER = ['runuser', '-u', 'lab', '--', 'docker', '--host', 'unix:///run/user/1000/urbit-docker.sock']


def image_matches(row):
    # Docker's classic store reports the config digest; its containerd store may
    # report the manifest digest. Both must name this exact verified image ref.
    return (row.get('Image') in (IMAGE_ID, IMAGE.split('@', 1)[1])
            and row.get('Config', {}).get('Image') == IMAGE)


def require_guest():
    if os.geteuid() != 0 or os.environ.get('OMABIT_PRIVATE_CHAIN_DOCKER') != 'guest-v1':
        raise RuntimeError('private keyed Docker guest admission required')
    if (Path('/opt/omabit-lab/guest-marker').read_text() != 'offline-import-guest-v1\n'
        or Path('/sys/class/dmi/id/sys_vendor').read_text().strip() != 'QEMU'
        or Path('/proc/1/comm').read_text().strip() != 'systemd'
        or sorted(p.name for p in Path('/sys/class/net').iterdir()) != ['lo']):
        raise RuntimeError('fresh no-NIC QEMU guest required')


def receipt_name(kind, label=None):
    if kind not in ('chain', 'peers', 'log') or label not in (None, 'a', 'b'):
        raise ValueError('fixed private qualification receipt label required')
    suffix = '' if label is None else '-' + label
    return {'chain':'private-chain'+suffix+'.json',
            'peers':'private-chain-peers'+suffix+'.json',
            'log':'gateway'+suffix+'.log'}[kind]


def identity(value):
    if not isinstance(value, str) or not re.fullmatch('[0-9a-f]{64}', value):
        raise ValueError('an exact recorded container ID is required')
    return value


@dataclass(frozen=True)
class Ship:
    project: str
    ship: str
    anchor: str
    gateway_port: int
    resumed: bool = False

    def __post_init__(self):
        if not isinstance(self.project, str) or not re.fullmatch('[0-9a-f]{32}', self.project):
            raise ValueError('private project token required')
        if self.ship not in ('zod', 'nec') or type(self.resumed) is not bool:
            raise ValueError('only the two fresh fixture identities are supported')
        identity(self.anchor)
        if type(self.gateway_port) is not int or not 20000 <= self.gateway_port <= 29999:
            raise ValueError('private gateway port outside fixture range')

    @property
    def root(self):return Path('/home/lab') / ('omabit-private-chain-' + self.project)
    @property
    def number(self):return 0 if self.ship == 'zod' else 1
    @property
    def pier_parent(self):return self.root / 'piers' / ('private-' + self.ship)
    @property
    def pier(self):return self.pier_parent / self.ship
    @property
    def key(self):return self.root / 'secrets' / (self.ship + '.key')
    @property
    def name(self):return 'omabit-private-' + self.project + '-' + self.ship + ('-resume' if self.resumed else '-boot')

    def arguments(self):
        args = ['-t', '-L', '--no-conn', '--no-dock', '--loom', '31', '--snap-time', '60',
                '-b', '127.0.0.1', '--http-port', str(18081 + self.number), '-p', str(31337 + self.number)]
        if not self.resumed:
            args += ['-H', 'galaxies.omabit.invalid', '-w', self.ship, '-k', '/run/omabit-boot/key',
                     '-e', 'http://127.0.0.1:' + str(self.gateway_port), '-B', '/run/omabit-boot/boot.pill', '-c']
        return args + ['/urbit/' + self.ship]

    def mounts(self):
        value = {str(self.pier_parent):('/urbit', True)}
        if not self.resumed:
            value[str(self.key)] = ('/run/omabit-boot/key', False)
            value['/opt/omabit-lab/runtime/boot.pill'] = ('/run/omabit-boot/boot.pill', False)
        return value

    def create_command(self):
        args = ['create', '--name', self.name, '--label', LABEL + '=' + self.project,
                '--network', 'container:' + self.anchor, '--pull', 'never', '--init', '--read-only',
                '--tmpfs', '/tmp:rw,nosuid,nodev,mode=1777,size=64m', '--user', '0:0',
                '--cap-drop', 'ALL', '--security-opt', 'no-new-privileges', '--cgroupns', 'private',
                '--ipc', 'private', '--restart', 'no', '--memory', '4g', '--memory-swap', '4g',
                '--pids-limit', '256', '--cpus', '2', '--entrypoint', '/bin/urbit']
        for source, (target, writable) in self.mounts().items():
            args += ['--mount', 'type=bind,src=' + source + ',dst=' + target + ('' if writable else ',readonly')]
        return args + [IMAGE] + self.arguments()

    def check(self, row, container):
        identity(container)
        if (row.get('Id') != container or not image_matches(row) or row.get('Name') != '/' + self.name
            or row.get('Config', {}).get('Labels', {}).get(LABEL) != self.project
            or row.get('Config', {}).get('Entrypoint') != ['/bin/urbit']
            or row.get('Config', {}).get('Cmd') != self.arguments()):
            raise RuntimeError('keyed container identity or command changed')
        host = row.get('HostConfig', {})
        if (host.get('NetworkMode') != 'container:' + self.anchor or host.get('Init') is not True
            or host.get('ReadonlyRootfs') is not True or host.get('RestartPolicy', {}).get('Name') != 'no'
            or host.get('PortBindings') or host.get('Privileged') or host.get('Memory') != 4*1024**3
            or host.get('MemorySwap') != 4*1024**3 or host.get('NanoCpus') != 2000000000
            or host.get('PidsLimit') != 256 or host.get('CapAdd') or host.get('CapDrop') != ['ALL']
            or host.get('CgroupnsMode') != 'private' or host.get('IpcMode') != 'private'
            or row.get('Config', {}).get('User') != '0:0'
            or host.get('SecurityOpt') not in (['no-new-privileges'], ['no-new-privileges=true'])):
            raise RuntimeError('keyed container isolation or resource binding changed')
        mounts = row.get('Mounts', [])
        actual = {m.get('Source'):(m.get('Destination'), m.get('RW')) for m in mounts if m.get('Type') == 'bind'}
        if actual != self.mounts() or len(actual) != sum(m.get('Type') == 'bind' for m in mounts):
            raise RuntimeError('keyed container mounts changed')
        if any(m.get('Type') != 'bind' and not (m.get('Type') == 'tmpfs' and m.get('Destination') == '/tmp') for m in mounts):
            raise RuntimeError('unexpected keyed container filesystem')
        return row


class Engine:
    def __init__(self, environment):self.environment = dict(environment)

    def execute(self, arguments, timeout=30, *, capture_stderr=False):
        require_guest()
        if (self.environment.get('HOME') != '/home/lab'
            or self.environment.get('XDG_RUNTIME_DIR') != '/run/user/1000'):
            raise RuntimeError('private engine environment required')
        if capture_stderr:
            if len(arguments) != 4 or arguments[:3] != ['logs', '--tail', '200']:
                raise ValueError('combined diagnostics require bounded logs for one exact container')
            identity(arguments[3])
            result = subprocess.run(DOCKER + arguments, env=self.environment, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, timeout=timeout)
        else:
            result = subprocess.run(DOCKER + arguments, env=self.environment, capture_output=True, timeout=timeout)
        if result.returncode != 0 or len(result.stdout) > 4*1024*1024 or len(result.stderr or b'') > 65536:
            # Do not put Docker stderr, command arguments or bootstrap logs in a report.
            raise RuntimeError('private Docker ' + arguments[0] + ' failed; exit=' + str(result.returncode))
        return result.stdout

    def inspect(self, container):
        rows = json.loads(self.execute(['inspect', identity(container)]))
        if len(rows) != 1 or rows[0].get('Id') != container:raise RuntimeError('container identity changed')
        return rows[0]

    def check_anchor(self, spec):
        row = self.inspect(spec.anchor)
        if (not image_matches(row) or row.get('Config', {}).get('Labels', {}).get(LABEL) != spec.project
            or row.get('HostConfig', {}).get('NetworkMode') != 'none'
            or row.get('HostConfig', {}).get('PortBindings') or not row.get('State', {}).get('Running')):
            raise RuntimeError('private network-none anchor is unavailable')

    def create(self, spec):
        require_guest();self.check_anchor(spec)
        for source in spec.mounts():
            p = Path(source)
            if p.resolve(strict=True) != p:raise RuntimeError('private mount path is indirect')
        if spec.resumed != spec.pier.is_dir():raise RuntimeError('boot/reopen pier precondition changed')
        container = identity(self.execute(spec.create_command()).decode().strip())
        spec.check(self.inspect(container), container)
        return container

    def start(self, spec, container):
        require_guest();self.check_anchor(spec)
        before = spec.check(self.inspect(container), container)
        if before.get('State', {}).get('Status') != 'created':raise RuntimeError('only the fresh recorded container may start')
        if self.execute(['start', container]).decode().strip() != container:raise RuntimeError('start reply differs from owned container')
        return spec.check(self.inspect(container), container)

    def stop(self, spec, container, record, persist, lens_exit):
        """One owned shutdown dispatch; uncertain replies never trigger a resend.

        persist() must durably save record before it returns. The fixture retains
        an uncertain record on any exception. It never uses Docker stop's forced
        kill fallback, removes a container, or interprets dispatch as completion.
        """
        require_guest();mode = 'sigterm' if spec.resumed else 'lens-exit'
        expected = {'container':identity(container), 'project':spec.project, 'mode':mode}
        if record and any(record.get(k) != v for k, v in expected.items()):raise RuntimeError('shutdown record belongs to another container')
        row = spec.check(self.inspect(container), container);state = row['State']
        if record.get('passed'):
            if state['Running'] or state['ExitCode'] != 0 or state['OOMKilled']:raise RuntimeError('completed shutdown changed')
            return record
        if not record:
            if not state['Running']:raise RuntimeError('stopped container has no owned shutdown receipt')
            record.update(expected, dispatched=False, acknowledged=False, passed=False);persist()
        if not record['dispatched']:
            if not state['Running']:raise RuntimeError('container exited before owned shutdown dispatch')
            record['dispatched'] = True;persist()
            if spec.resumed:
                reply = self.execute(['kill', '--signal', 'TERM', container]).decode().strip()
                record['acknowledged'] = reply == container
            else:record['acknowledged'] = lens_exit() is True
            persist()
        # A prior unknown dispatch is observed, not repeated. A completed but
        # unacknowledged shutdown remains explicit uncertainty in this first gate.
        exit_code = self.execute(['wait', container], timeout=75).decode().strip()
        final = spec.check(self.inspect(container), container)['State']
        record.update(exit=final['ExitCode'], oom=final['OOMKilled'], running=final['Running']);persist()
        if (exit_code != '0' or final['Running'] or final['OOMKilled'] or final['ExitCode'] != 0
            or not record['acknowledged']):raise RuntimeError('owned shutdown lacks acknowledged clean exit')
        record['passed'] = True;persist();return record
