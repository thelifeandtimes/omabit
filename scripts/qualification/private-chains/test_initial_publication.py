"""Inert initial gateway publication selection; never a guest/engine launcher."""
import os
import unittest
from unittest.mock import patch
import docker_case
import docker_peer
import docker_runtime
from docker_publication import Publication,PublishedEngine,PublishedShip

class InitialPublication(unittest.TestCase):
    def test_first_bridge_routes_must_match_the_owned_project(self):
        from test_docker_publication import contract,network,ANCHOR
        publication=contract();row=network(ANCHOR)
        import ipaddress
        config=row['IPAM']['Config'][0];subnet=ipaddress.ip_network(config['Subnet'])
        little=lambda a:ipaddress.ip_address(a).packed[::-1].hex().upper()
        header='Iface Destination Gateway Flags RefCnt Use Metric Mask MTU Window IRTT\n'
        default='eth0 00000000 '+little(config['Gateway'])+' 0003 0 0 0 00000000 0 0 0\n'
        connected='eth0 '+little(subnet.network_address)+' 00000000 0001 0 0 0 '+little(subnet.netmask)+' 0 0 0\n'
        routes=header+default+connected;devices='header\nlo: counters\neth0: counters\n'
        docker_peer.check_peer_network(routes,devices,publication,row,ANCHOR)
        docker_peer.check_peer_network(header,'lo: counters\n')
        for bad_routes,bad_devices in [(routes,devices+'eth1: counters\n'),(header+default,devices),(routes.replace(little(config['Gateway']),'01020304'),devices),(routes.replace('0003','0007'),devices),(header+default+default,devices),(routes.replace('eth0','eth1'),devices)]:
            with self.subTest(routes=bad_routes),self.assertRaises(RuntimeError):docker_peer.check_peer_network(bad_routes,bad_devices,publication,row,ANCHOR)
        with self.assertRaises(RuntimeError):docker_peer.check_peer_network(routes,devices)
    def test_two_explicit_distinct_unprivileged_ports(self):
        self.assertIsNone(docker_case.initial_publication_ports(None))
        ports=(18381,18382)
        self.assertIs(docker_case.initial_publication_ports(ports),ports)
        for bad in [[],[18381,18382],(),(18381,),(18381,18381),(1023,18382),(18381,65536),(True,18382),('18381',18382),(18381,18382,18383)]:
            with self.subTest(bad=bad),self.assertRaises(ValueError):docker_case.initial_publication_ports(bad)
    def test_default_peers_keep_network_none_and_loopback_http(self):
        engine,specs,ship_type=docker_peer.peer_runtime({},'a'*32,'b'*64,25432)
        self.assertIs(type(engine),docker_runtime.Engine);self.assertIs(ship_type,docker_runtime.Ship)
        for spec in specs.values():
            args=spec.arguments();self.assertEqual(args[args.index('-b')+1],'127.0.0.1')
    def test_published_peers_use_exact_project_and_preserve_first_boot_and_resume(self):
        publication=Publication('a'*32,'c'*64,18381,18382)
        engine,specs,ship_type=docker_peer.peer_runtime({},'a'*32,'b'*64,25432,publication)
        self.assertIsInstance(engine,PublishedEngine);self.assertIs(ship_type,PublishedShip)
        for spec in specs.values():
            self.assertFalse(spec.resumed);self.assertIn('-k',spec.arguments())
            args=spec.arguments();self.assertEqual(args[args.index('-b')+1],'0.0.0.0')
            create=spec.create_command();self.assertEqual(create[create.index('--network')+1],'container:'+'b'*64);self.assertNotIn('--publish',create)
        resumed=ship_type('a'*32,'nec','b'*64,25432,True)
        self.assertNotIn('-k',resumed.arguments());self.assertNotIn('-B',resumed.arguments());self.assertEqual(resumed.arguments()[resumed.arguments().index('-b')+1],'0.0.0.0')
        self.assertEqual(publication.gateway_network_arguments(),['--network','c'*64,'--publish','127.0.0.1:18381:18081/tcp','--publish','127.0.0.1:18382:18082/tcp'])
    def test_wrong_publication_never_selects_engine(self):
        for bad in [False,{},Publication('d'*32,'c'*64,18381,18382)]:
            with self.subTest(bad=type(bad).__name__),patch('docker_publication.PublishedEngine') as factory:
                with self.assertRaises(ValueError):docker_peer.peer_runtime({},'a'*32,'b'*64,25432,bad)
                factory.assert_not_called()
    def test_host_entry_refuses_before_engine_files_or_traffic(self):
        publication=Publication('a'*32,'c'*64,18381,18382)
        for call in [lambda:docker_case.exercise({},publication_ports=(18381,18382)),lambda:docker_peer.exercise({},'a'*32,'b'*64,publication=publication)]:
            with patch.dict(os.environ,{},clear=True),patch.object(docker_runtime.os,'geteuid',return_value=1000),patch.object(docker_runtime.subprocess,'run') as run,patch.object(docker_case,'Engine') as case_engine,patch.object(docker_peer,'peer_runtime') as peer_factory:
                with self.assertRaisesRegex(RuntimeError,'private keyed Docker guest admission required'):call()
                run.assert_not_called();case_engine.assert_not_called();peer_factory.assert_not_called()

if __name__=='__main__':unittest.main()
