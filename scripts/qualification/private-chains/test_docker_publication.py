"""Inert exposure/refusal tests; no engine, network, ship or HTTP is used."""
import copy
import os
from pathlib import Path
import sys
import unittest
from unittest.mock import patch

HERE = Path(__file__).resolve().parent
import docker_publication as p
from docker_runtime import IMAGE, IMAGE_ID, LABEL, Ship

PROJECT = 'a' * 32
NETWORK = 'b' * 64
ANCHOR = 'c' * 64


def contract():
    return p.Publication(PROJECT, NETWORK, 18381, 18382)


def network(anchor=ANCHOR):
    return {'Id': NETWORK, 'Name': 'omabit-private-' + PROJECT, 'Driver': 'bridge', 'Scope': 'local',
            'Internal': False, 'EnableIPv4': True, 'EnableIPv6': False, 'Ingress': False,
            'Labels': {LABEL: PROJECT},
            'Options': {'com.docker.network.bridge.host_binding_ipv4': '127.0.0.1', 'com.docker.network.bridge.gateway_mode_ipv4': 'nat'},
            'IPAM': {'Driver': 'default', 'Options': None, 'Config': [{'Subnet': '172.30.0.0/16', 'Gateway': '172.30.0.1'}]},
            'Containers': {anchor: {}} if anchor else {}}


def gateway():
    bindings = {'18081/tcp': [{'HostIp': '127.0.0.1', 'HostPort': '18381'}], '18082/tcp': [{'HostIp': '127.0.0.1', 'HostPort': '18382'}]}
    return {'Id': ANCHOR, 'Image': IMAGE_ID, 'Config': {'Image': IMAGE, 'Labels': {LABEL: PROJECT}},
            'HostConfig': {'NetworkMode': NETWORK, 'PortBindings': bindings},
            'NetworkSettings': {'Ports': copy.deepcopy(bindings), 'Networks': {'omabit-private-' + PROJECT: {'NetworkID': NETWORK}}},
            'State': {'Running': True, 'OOMKilled': False, 'Status': 'running'}}


class PublicationTests(unittest.TestCase):
    def test_only_two_distinct_unprivileged_ports(self):
        for ports in [(18381, 18381), (80, 18382), (True, 18382), (18381, 65536), ('18381', 18382)]:
            with self.subTest(ports=ports), self.assertRaises(ValueError):
                p.Publication(PROJECT, NETWORK, *ports)
        self.assertEqual(contract().gateway_network_arguments(), ['--network', NETWORK, '--publish', '127.0.0.1:18381:18081/tcp', '--publish', '127.0.0.1:18382:18082/tcp'])

    def test_network_command_needs_no_fabricated_observed_id(self):
        args = p.network_create_command(PROJECT)
        self.assertEqual(args[-1], 'omabit-private-' + PROJECT)
        self.assertIn('--internal=false', args)
        self.assertIn('--ipv6=false', args)
        for invalid in ['', 'a' * 31, 'A' * 32, '../lab']:
            with self.subTest(project=invalid), self.assertRaises(ValueError):
                p.network_create_command(invalid)

    def test_owned_bridge_and_only_gateway_member(self):
        c = contract();c.check_network(network(), ANCHOR);c.check_network(network(None))
        for key, value in [('Id', 'd' * 64), ('Internal', True), ('EnableIPv6', True), ('Driver', 'host'), ('Ingress', True), ('EnableIPv4', False), ('Scope', 'swarm')]:
            row = network();row[key] = value
            with self.subTest(key=key), self.assertRaises(RuntimeError):c.check_network(row, ANCHOR)
        for participants in [{}, {ANCHOR: {}, 'd' * 64: {}}, {'d' * 64: {}}]:
            row = network();row['Containers'] = participants
            with self.subTest(participants=participants), self.assertRaises(RuntimeError):c.check_network(row, ANCHOR)

    def test_bridge_options_cannot_enable_routing_or_wildcard_binding(self):
        for key, value in [('com.docker.network.bridge.host_binding_ipv4', '0.0.0.0'), ('com.docker.network.bridge.gateway_mode_ipv4', 'routed'), ('com.docker.network.bridge.trusted_host_interfaces', 'eth0')]:
            row = network();row['Options'][key] = value
            with self.subTest(key=key), self.assertRaises(RuntimeError):contract().check_network(row, ANCHOR)

    def test_public_ipv6_and_auxiliary_ipam_refuse(self):
        for config in [{'Subnet': '203.0.113.0/24', 'Gateway': '203.0.113.1'}, {'Subnet': '::/0', 'Gateway': '::1'}, {'Subnet': '172.30.0.0/16', 'Gateway': '172.31.0.1'}, {'Subnet': '172.30.0.0/16', 'Gateway': '172.30.0.0'}, {'Subnet': '172.30.0.0/16', 'Gateway': '172.30.0.1', 'AuxiliaryAddresses': {'peer': '172.30.0.2'}}]:
            row = network();row['IPAM']['Config'] = [config]
            with self.subTest(config=config), self.assertRaises((RuntimeError, ValueError)):contract().check_network(row, ANCHOR)

    def test_active_publication_and_image_expose_without_mapping(self):
        row = gateway();row['NetworkSettings']['Ports']['34343/udp'] = None
        contract().check_gateway_network(row, ANCHOR)
        row['State'].update(Running=False, Status='created')
        contract().check_gateway_network(row, ANCHOR, running=False)
        with self.assertRaises(RuntimeError):contract().check_gateway_network(row, ANCHOR)

    def test_wildcard_ipv6_rpc_ames_extra_ports_refuse(self):
        for where, field in [('HostConfig', 'PortBindings'), ('NetworkSettings', 'Ports')]:
            for ip in ['0.0.0.0', '::', '::1', '']:
                row = gateway();row[where][field]['18081/tcp'][0]['HostIp'] = ip
                with self.subTest(where=where, ip=ip), self.assertRaises(RuntimeError):contract().check_gateway_network(row, ANCHOR)
            for port in ['25432/tcp', '31337/udp', '12321/tcp']:
                row = gateway();row[where][field][port] = [{'HostIp': '127.0.0.1', 'HostPort': '19999'}]
                with self.subTest(where=where, port=port), self.assertRaises(RuntimeError):contract().check_gateway_network(row, ANCHOR)

    def test_host_wrong_project_image_and_extra_network_refuse(self):
        mutations = [lambda r: r['HostConfig'].update(NetworkMode='host'), lambda r: r['Config']['Labels'].update({LABEL: 'd' * 32}), lambda r: r['Config'].update(Image='vere:latest'), lambda r: r['NetworkSettings']['Networks'].update(other={'NetworkID': 'd' * 64}), lambda r: r['State'].update(OOMKilled=True)]
        for change in mutations:
            row = gateway();change(row)
            with self.assertRaises(RuntimeError):contract().check_gateway_network(row, ANCHOR)

    def test_only_ordinary_http_bind_changes_for_fresh_and_resumed_piers(self):
        for resumed in [False, True]:
            original = Ship(PROJECT, 'nec', ANCHOR, 25432, resumed)
            published = p.PublishedShip(PROJECT, 'nec', ANCHOR, 25432, resumed)
            a, b = original.arguments(), published.arguments()
            self.assertEqual([(x, y) for x, y in zip(a, b) if x != y], [('127.0.0.1', '0.0.0.0')])
            self.assertEqual(original.mounts(), published.mounts())
            self.assertNotIn('--publish', published.create_command())
            if resumed:
                for flag in ['-k', '-w', '-B', '-c']:
                    self.assertNotIn(flag, b)
            else:
                self.assertEqual(b[b.index('-e') + 1], 'http://127.0.0.1:25432')

    def test_host_entry_refuses_before_any_process(self):
        engine = p.PublishedEngine({'HOME': '/home/lab', 'XDG_RUNTIME_DIR': '/run/user/1000'}, contract())
        spec = p.PublishedShip(PROJECT, 'zod', ANCHOR, 25432)
        with patch.dict(os.environ, {}, clear=True), patch('subprocess.run', side_effect=AssertionError('process must not run')):
            with self.assertRaises(RuntimeError):engine.check_anchor(spec)
        with self.assertRaises(RuntimeError):engine.check_anchor(Ship(PROJECT, 'zod', ANCHOR, 25432))


class PublicationIntegrationTests(unittest.TestCase):
    def test_opt_in_source_selection_cannot_omit_recovery(self):
        import docker_appliance as prep
        for values in [(False, False, True), (True, False, True), (True, True, 1)]:
            with self.subTest(values=values), self.assertRaises(ValueError):
                prep.gateway_sources(*values)
        files = prep.gateway_sources(True, True, True)
        self.assertEqual(files['guest.py'], prep.HERE / 'docker_publication_guest.py')
        self.assertIn('docker_published_client.py', files)
        self.assertNotIn('docker_published_client.py', prep.gateway_sources(True, True))

    def test_publication_does_not_weaken_gateway_mounts_or_resources(self):
        import test_docker_resume_case as fixture
        import docker_resume_case as case
        c = p.Publication(fixture.PROJECT, NETWORK, 18381, 18382)
        row = fixture.row()
        row['HostConfig'].update(NetworkMode=NETWORK, PortBindings=gateway()['HostConfig']['PortBindings'])
        case.check_gateway(row, fixture.CONTAINER, fixture.PROJECT, fixture.NAME, fixture.MOUNTS, fixture.COMMAND, fixture.VARS, c)
        for change in [lambda r: r['Mounts'].pop(), lambda r: r['HostConfig'].update(Privileged=True), lambda r: r['HostConfig'].update(MemorySwap=-1)]:
            altered = copy.deepcopy(row);change(altered)
            with self.assertRaises(RuntimeError):
                case.check_gateway(altered, fixture.CONTAINER, fixture.PROJECT, fixture.NAME, fixture.MOUNTS, fixture.COMMAND, fixture.VARS, c)

    def test_publication_cannot_be_started_without_guest_admission(self):
        import docker_resume_case as case
        with patch.dict(os.environ, {}, clear=True), patch.object(case, 'Engine') as engine, patch.object(case, 'directory') as directory:
            with self.assertRaises(RuntimeError):
                case.exercise({}, {'ok': True, 'project': PROJECT}, peer_resume=lambda *a: None, publication_ports=(18381, 18382))
            engine.assert_not_called();directory.assert_not_called()


if __name__ == '__main__':
    unittest.main()
