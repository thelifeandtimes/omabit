"""Inert network contract for the next guest-only private-chain publication case.

Only ordinary Eyre is published on the no-NIC VM's IPv4 loopback. The gateway's
RPC and Vere's administrative listener remain unpublished. These builders do not
create a network or authorize a runtime; the existing guest guard is still needed.
"""
from dataclasses import dataclass
import ipaddress
import re

from docker_runtime import Engine, LABEL, Ship, identity, image_matches


def network_create_command(project):
    if not isinstance(project, str) or not re.fullmatch('[0-9a-f]{32}', project):
        raise ValueError('private project required')
    # An ordinary bridge is needed for rootless port forwarding. The whole
    # qualification VM has no NIC; this is not an Internet-access contract.
    return ['network', 'create', '--driver', 'bridge', '--internal=false',
            '--ipv4=true', '--ipv6=false', '--label', LABEL + '=' + project,
            '--opt', 'com.docker.network.bridge.host_binding_ipv4=127.0.0.1',
            '--opt', 'com.docker.network.bridge.gateway_mode_ipv4=nat', 'omabit-private-' + project]


@dataclass(frozen=True)
class Publication:
    project: str
    network: str
    zod_port: int
    nec_port: int

    def __post_init__(self):
        if not isinstance(self.project, str) or not re.fullmatch('[0-9a-f]{32}', self.project):
            raise ValueError('private project required')
        identity(self.network)
        if any(type(p) is not int or not 1024 <= p <= 65535 for p in self.ports.values()) or self.zod_port == self.nec_port:
            raise ValueError('two distinct unprivileged loopback ports required')

    @property
    def name(self):
        return 'omabit-private-' + self.project

    @property
    def ports(self):
        return {18081: self.zod_port, 18082: self.nec_port}

    def gateway_network_arguments(self):
        args = ['--network', self.network]
        for port, host in self.ports.items():
            args += ['--publish', '127.0.0.1:' + str(host) + ':' + str(port) + '/tcp']
        return args

    def check_network(self, row, anchor=None):
        if (row.get('Id') != self.network or row.get('Name') != self.name
            or row.get('Driver') != 'bridge' or row.get('Scope') != 'local'
            or row.get('Internal') is not False or row.get('EnableIPv6') is not False
            or row.get('EnableIPv4') is not True or row.get('Ingress') is not False
            or row.get('Labels', {}).get(LABEL) != self.project):
            raise RuntimeError('private publication network differs')
        wanted = {'com.docker.network.bridge.host_binding_ipv4': '127.0.0.1',
                  'com.docker.network.bridge.gateway_mode_ipv4': 'nat'}
        if row.get('Options') != wanted:
            raise RuntimeError('private bridge options differ')
        ipam = row.get('IPAM', {})
        configs = ipam.get('Config')
        if ipam.get('Driver') != 'default' or ipam.get('Options') not in (None, {}) or not isinstance(configs, list) or len(configs) != 1:
            raise RuntimeError('private bridge allocation differs')
        config = configs[0]
        if not isinstance(config, dict) or set(config) != {'Subnet', 'Gateway'}:
            raise RuntimeError('private bridge address metadata differs')
        subnet = ipaddress.ip_network(config['Subnet'], strict=True)
        gateway = ipaddress.ip_address(config['Gateway'])
        private = [ipaddress.ip_network(s) for s in ('10.0.0.0/8', '172.16.0.0/12', '192.168.0.0/16')]
        if (subnet.version != 4 or not any(subnet.subnet_of(block) for block in private)
            or gateway not in subnet or gateway in (subnet.network_address, subnet.broadcast_address)):
            raise RuntimeError('private bridge address is not supported')
        containers = row.get('Containers') or {}
        if not isinstance(containers, dict) or set(containers) != ({identity(anchor)} if anchor is not None else set()):
            raise RuntimeError('unexpected private bridge participant')
        return row

    def check_gateway_configuration(self, row, anchor):
        identity(anchor)
        if (row.get('Id') != anchor or not image_matches(row)
            or row.get('Config', {}).get('Labels', {}).get(LABEL) != self.project
            or row.get('HostConfig', {}).get('NetworkMode') != self.network):
            raise RuntimeError('published gateway ownership differs')
        expected = {str(port) + '/tcp': [{'HostIp': '127.0.0.1', 'HostPort': str(host)}] for port, host in self.ports.items()}
        if row.get('HostConfig', {}).get('PortBindings') != expected:
            raise RuntimeError('gateway publication differs')
        return row

    def check_gateway_network(self, row, anchor, *, running=True):
        self.check_gateway_configuration(row, anchor)
        expected = row["HostConfig"]["PortBindings"]
        state = row.get('State', {})
        if running:
            if state.get('Running') is not True or state.get('OOMKilled') is not False:
                raise RuntimeError('published gateway is not running')
            actual = row.get('NetworkSettings', {}).get('Ports') or {}
            if {key: value for key, value in actual.items() if value} != expected:
                raise RuntimeError('active gateway publication differs')
            networks = row.get('NetworkSettings', {}).get('Networks') or {}
            if set(networks) != {self.name} or networks[self.name].get('NetworkID') != self.network:
                raise RuntimeError('gateway joined an unexpected network')
        elif state.get('Status') != 'created' or state.get('Running') is not False:
            raise RuntimeError('fresh published gateway is not created')
        return row


@dataclass(frozen=True)
class PublishedShip(Ship):
    def arguments(self):
        args = super().arguments()
        at = args.index('-b') + 1
        if args[at] != '127.0.0.1':
            raise RuntimeError('ordinary HTTP binding source changed')
        args[at] = '0.0.0.0'
        return args


class PublishedEngine(Engine):
    def __init__(self, environment, publication):
        super().__init__(environment)
        if not isinstance(publication, Publication):
            raise ValueError('exact private publication contract required')
        self.publication = publication

    def check_anchor(self, spec):
        # Engine.execute retains the no-NIC QEMU/rootless-engine guard. Member
        # creation still has all Ship mount, key, image and resource checks.
        if not isinstance(spec, PublishedShip) or spec.project != self.publication.project:
            raise RuntimeError('published member belongs to another contract')
        import json
        rows = json.loads(self.execute(['network', 'inspect', self.publication.network]))
        if not isinstance(rows, list) or len(rows) != 1:
            raise RuntimeError('private network observation differs')
        self.publication.check_network(rows[0], spec.anchor)
        self.publication.check_gateway_network(self.inspect(spec.anchor), spec.anchor)
