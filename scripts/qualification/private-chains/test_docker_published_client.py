import os
import sys
from pathlib import Path
import unittest
from unittest.mock import Mock, patch

sys.path.insert(0, str(Path(__file__).resolve().parent.parent / 'fake-networks'))
import docker_published_client as client
from docker_publication import Publication

P='a'*32;A='b'*64;N='c'*64;O='d'*32
COOKIE=b'# Netscape HTTP Cookie File\n#HttpOnly_127.0.0.1\tFALSE\t/\tFALSE\t0\turbauth-~zod\t0v.synthetic-token\n'

class DirectClientTests(unittest.TestCase):
    def test_private_cookie_origin_identity_and_duplicates(self):
        self.assertEqual(client.cookie_header(COOKIE,'zod'),'urbauth-~zod=0v.synthetic-token')
        for value in [COOKIE+COOKIE,COOKIE.replace(b'127.0.0.1',b'example.com'),COOKIE.replace(b'FALSE\t/',b'TRUE\t/'),COOKIE.replace(b'~zod',b'~nec'),COOKIE.replace(b'0v.synthetic-token',b'x; y=secret'),b'x'*8193]:
            with self.subTest(case=value[:20]),self.assertRaises(ValueError):client.cookie_header(value,'zod')

    def test_actual_gateway_ship_passes_published_engine_binding(self):
        import json
        from docker_publication import PublishedEngine, PublishedShip
        from test_docker_publication import contract, network, gateway, PROJECT, ANCHOR, NETWORK
        spec=client.PublishedGatewayShip(PROJECT,'zod',ANCHOR,25432,True,O)
        self.assertIsInstance(spec,PublishedShip)
        engine=PublishedEngine({},contract())
        engine.execute=Mock(return_value=json.dumps([network()]).encode())
        engine.inspect=Mock(return_value=gateway())
        engine.check_anchor(spec)
        engine.execute.assert_called_once_with(['network','inspect',NETWORK])
        engine.inspect.assert_called_once_with(ANCHOR)

    def test_no_boot_inputs_on_published_reopen(self):
        spec=client.PublishedGatewayShip(P,'zod',A,25432,True,O)
        args=spec.arguments()
        self.assertEqual(args[args.index('-b')+1],'0.0.0.0')
        self.assertEqual(list(spec.mounts().values()),[('/urbit',True)])
        for flag in ['-k','-B','-c','-w']:self.assertNotIn(flag,args)

    def fixture(self):
        specs={s:client.PublishedGatewayShip(P,s,A,25432,True,O) for s in ('zod','nec')}
        return client.PublishedTransport(Mock(),specs,Publication(P,N,18381,18382),Path('/synthetic'))

    def test_host_guard_precedes_cookie_or_connection(self):
        obj=self.fixture()
        with patch.dict(os.environ,{},clear=True),patch.object(client,'private_read') as read,patch.object(client.http.client,'HTTPConnection') as connection:
            for action in [lambda:obj.call('zod','/~/host'),lambda:obj.events('/bad',None,{},False),obj.check_unpublished]:
                with self.assertRaises(RuntimeError):action()
            read.assert_not_called();connection.assert_not_called();obj.engine.check_anchor.assert_not_called()

    def test_unknown_write_outcome_is_never_resent_or_exported(self):
        obj=self.fixture();connection=Mock();connection.request.side_effect=OSError('private secret')
        with patch.object(client,'require_guest'),patch.object(client,'private_read',return_value=COOKIE),patch.object(client.http.client,'HTTPConnection',return_value=connection) as build:
            with self.assertRaisesRegex(RuntimeError,'request not replayed') as error:obj.call('zod','/write',b'synthetic-value')
        self.assertNotIn('private secret',str(error.exception));connection.request.assert_called_once();connection.close.assert_called_once()
        build.assert_called_once_with('127.0.0.1',18381,timeout=90)
        self.assertEqual(obj.requests,[])

    def test_no_relogin_or_redirect_following(self):
        obj=self.fixture();connection=Mock();response=connection.getresponse.return_value;response.status=302;response.read.return_value=b''
        with patch.object(client,'require_guest'),patch.object(client,'private_read',return_value=COOKIE),patch.object(client.http.client,'HTTPConnection',return_value=connection):
            with self.assertRaisesRegex(RuntimeError,'must not log in'):obj.call('zod','/~/login',b'code',login=True)
            self.assertEqual(obj.call('zod','/~/host'),(302,b''))
        connection.request.assert_called_once()
        self.assertEqual(obj.requests[0]['status'],302)
        self.assertNotIn('Cookie',str(obj.requests))

    def test_only_exact_unpublished_endpoints_are_probed(self):
        obj=self.fixture();obj.admin={'zod':19001,'nec':19002}
        with patch.object(client,'require_guest'),patch.object(client.socket,'create_connection',side_effect=ConnectionRefusedError) as connect:
            rows=obj.check_unpublished()
        self.assertEqual(len(rows),5)
        self.assertEqual([v.args[0] for v in connect.call_args_list],[('127.0.0.1',25432),('127.0.0.1',19001),('127.0.0.1',19002),('::1',18381),('::1',18382)])

if __name__=='__main__':unittest.main()
