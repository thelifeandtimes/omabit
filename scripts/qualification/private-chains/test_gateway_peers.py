"""Inert whole-gateway client checkpoint tests. Never opens Docker or a ship."""
import copy
import os
from pathlib import Path
import sys
import unittest
from unittest.mock import patch

HERE=Path(__file__).resolve().parent
sys.path.insert(0,str(HERE.parent/'fake-networks'))
import docker_gateway_peers as peers
import docker_appliance as prep

PROJECT='a'*32
OLD='b'*64
NEW='c'*64
OPERATION='d'*32

def fixture():
    entries=['omabit-state-'+f'{n:024x}' for n in range(5)]
    channel='/~/channel/omabit-state-'+'e'*32
    recovery={'ok':False,'channels':[channel],'channels_deleted':[],
      'writes':[{'state':'acknowledged','http_status':200,'phase':'queued-before-gateway-replacement','entry':entries[-1]}],
      'observations':[{'via':'subscription','channel':channel,'cursor':1,'state':{'revision':4,'entries':entries[:4]}},{'via':'scry','state':{'revision':5,'entries':entries.copy()}}]}
    return {'version':1,'project':PROJECT,'previous_gateway':OLD,'previous_namespace':'net:[12]','ready':True,'channel':channel,'cursor':1,'entries':entries,'recovery':recovery,
      'ships':{s:{'container':str(n)*64,'resumed':s=='nec','pier_identity':[1,n+1],'cookie_sha256':'f'*64} for n,s in enumerate(['zod','nec'])}}

class GatewayPeerTests(unittest.TestCase):
    def test_host_refuses_before_reading_private_checkpoint_or_constructing_engine(self):
        with patch.dict(os.environ,{},clear=True),patch.object(peers,'Engine') as engine,patch.object(peers,'private_read') as read:
            with self.assertRaisesRegex(RuntimeError,'guest admission'):peers.exercise({},PROJECT,OLD,NEW,OPERATION)
            with self.assertRaisesRegex(RuntimeError,'guest admission'):peers.capture(None,None,None,None,None)
            engine.assert_not_called();read.assert_not_called()

    def test_checkpoint_preserves_acknowledged_write_and_unread_subscription(self):
        value=fixture();self.assertEqual(peers.checkpoint(value,PROJECT,OLD),value)
        changes={
          'project':lambda v:v.update(project='c'*32),
          'previous':lambda v:v.update(previous_gateway=NEW),
          'ready':lambda v:v.update(ready=False),
          'channel':lambda v:v.update(channel='/other'),
          'cursor':lambda v:v.update(cursor=True),
          'entries':lambda v:v['entries'].pop(),
          'duplicate-entry':lambda v:v['entries'].__setitem__(4,v['entries'][0]),
          'namespace':lambda v:v.update(previous_namespace='host'),
          'foreign-pier':lambda v:v['ships']['nec'].update(pier_identity=[1,0]),
          'boot-again':lambda v:v['ships']['nec'].update(resumed=False),
          'cookie':lambda v:v['ships']['zod'].update(cookie_sha256='secret'),
          'pending-write':lambda v:v['recovery']['writes'][0].update(state='pending'),
          'different-write':lambda v:v['recovery']['writes'][0].update(entry=v['entries'][0]),
          'already-deleted':lambda v:v['recovery'].update(channels_deleted=[v['channel']]),
          'different-observation':lambda v:v['recovery']['observations'][0].update(cursor=2),
          'different-state':lambda v:v['recovery']['observations'][1]['state'].update(revision=4)}
        for name,change in changes.items():
            with self.subTest(name=name):
                value=fixture();change(value)
                with self.assertRaises((ValueError,RuntimeError)):peers.checkpoint(value,PROJECT,OLD)

    def test_replacement_members_have_distinct_names_and_never_remount_boot_material(self):
        for ship in ('zod','nec'):
            spec=peers.GatewayShip(PROJECT,ship,NEW,25432,True,OPERATION)
            self.assertIn(OPERATION,spec.name)
            self.assertNotEqual(spec.name,peers.Ship(PROJECT,ship,OLD,25432,True).name)
            self.assertEqual(list(spec.mounts()),[str(spec.pier_parent)])
            for forbidden in ('-w','-k','-B','-c','-e'):self.assertNotIn(forbidden,spec.arguments())
            self.assertIn('container:'+NEW,spec.create_command())
        for resumed,operation in [(False,OPERATION),(True,''),(True,'bad')]:
            with self.subTest(resumed=resumed,operation=operation),self.assertRaises(ValueError):peers.GatewayShip(PROJECT,'nec',NEW,25432,resumed,operation)

    def test_retired_namespace_number_is_not_a_permanent_generation(self):
        self.assertTrue(peers.replacement_namespace('net:[12]', 'net:[1]', 'net:[12]'))
        self.assertFalse(peers.replacement_namespace('net:[13]', 'net:[1]', 'net:[12]'))
        for current,guest,previous in [('net:[1]','net:[1]','net:[12]'),('net:[12]','net:[1]','net:[1]'),('bad','net:[1]','net:[12]'),('net:[12]',None,'net:[12]')]:
            with self.subTest(current=current,guest=guest,previous=previous),self.assertRaises(RuntimeError):
                peers.replacement_namespace(current,guest,previous)
        with patch.object(peers,'require_guest'),patch.object(peers,'private_read') as read:
            with self.assertRaisesRegex(ValueError,'distinct retained gateway'):
                peers.exercise({},PROJECT,OLD,OLD,OPERATION)
            read.assert_not_called()

    def test_peer_variant_requires_explicit_resume_and_changes_only_hooks(self):
        with self.assertRaises(ValueError):prep.gateway_sources(False,True)
        files=prep.gateway_sources(True,True)
        self.assertEqual(files['guest.py'],prep.HERE/'docker_gateway_peers_guest.py')
        self.assertEqual(set(files)-set(prep.gateway_sources(True)),{'docker_gateway_peers.py'})
        old=(prep.HERE/'docker_resume_guest.py').read_text()
        new=(prep.HERE/'docker_gateway_peers_guest.py').read_text()
        new=new.replace('        import docker_gateway_peers\n','').replace('docker_case.exercise(environment,peer_checkpoint=docker_gateway_peers.capture)','docker_case.exercise(environment)').replace('docker_resume_case.exercise(environment,proof,peer_resume=docker_gateway_peers.exercise)','docker_resume_case.exercise(environment,proof)')
        self.assertEqual(old,new)

if __name__=='__main__':unittest.main()
