"""Offline routing checks for the fresh two-ship application fixture."""
import json
from pathlib import Path
import re
import tempfile
import unittest
from network_case import cue_atom, jam
from state_case import Recovery

class StateTargets(unittest.TestCase):
    def test_install_subscription_and_write_use_selected_disposable_target(self):
        for target,sender in [('nec','zod'),('zod','nec')]:
            with self.subTest(target=target),tempfile.TemporaryDirectory(prefix='omabit-state-target-') as root:
                path=Path(root);(path/'base/app').mkdir(parents=True);source=path/'fixture.hoon';source.write_text(':: inert source fixture\n')
                calls=[];entries=[];sequence=0
                def http(ship,route,data=None,**kwargs):
                    calls.append((ship,route,data,kwargs))
                    if route.startswith('/spider/'):
                        code=cue_atom(data);code=code.to_bytes((code.bit_length()+7)//8,'little').decode()
                        receipt=re.search(r'omabit-receipt-[0-9a-f]{32}',code)[0]
                        if route.endswith('/json'):
                            self.assertEqual(ship,target)
                            self.assertIn('~'+target+' %hood',code)
                            self.assertNotIn('~'+sender+' %hood',code)
                            return 200,json.dumps(receipt).encode()
                        self.assertEqual(ship,sender)
                        self.assertIn('~'+target+' %omabit-state',code)
                        entries.append(re.search(r'omabit-state-[0-9a-f]{24}',code)[0])
                        return 200,jam(int.from_bytes(receipt.encode(),'little'))
                    self.assertEqual(ship,target)
                    if kwargs.get('admin'):
                        expression=json.loads(data)['source']['dojo']
                        if '/domes' in expression:
                            self.assertIn('/~'+target+'//',expression)
                            if 'ren.d' in expression:self.assertIn('[~'+target+' %base]',expression)
                        return 200,json.dumps('~' if 'ren.d' in expression else '%.y').encode()
                    if '/scry/' in route:return 200,json.dumps({'revision':len(entries),'entries':entries}).encode()
                    if kwargs.get('method')=='PUT':
                        for action in json.loads(data):
                            if action['action']=='subscribe':self.assertEqual(action['ship'],target)
                        return 204,b''
                    raise AssertionError(route)
                def events(channel,cursor,expected,initial):
                    nonlocal sequence
                    rows=[]
                    if initial:
                        sequence+=1;rows.append((sequence,{'id':1,'response':'subscribe','ok':'ok'}))
                    sequence+=1;rows.append((sequence,{'id':1,'response':'diff','json':expected}))
                    return rows
                r=Recovery(http,events,lambda:None,lambda *a,**kw:None,target=target,sender=sender)
                r.install(path,source);r.subscribe();r.write('selected-target');r.receive();r.close()
                self.assertEqual((path/'base/app/omabit-state.hoon').read_bytes(),source.read_bytes())
                self.assertEqual(r.entries,entries);self.assertEqual(len(entries),1)
                self.assertEqual(len([c for c in calls if c[1].endswith('/khan-eval/noun')]),1)
    def test_default_roles_remain_nec_target_zod_sender(self):
        r=Recovery(None,None,None,None)
        self.assertEqual((r.target,r.sender),('nec','zod'))
    def test_other_ship_names_refused_before_any_effect(self):
        for target,sender in [('~zod','nec'),('zod','~nec'),('other','nec'),('nec','other'),("zod';!!",'nec')]:
            with self.subTest(target=target,sender=sender),self.assertRaises(ValueError):
                Recovery(None,None,None,None,target=target,sender=sender)

if __name__=='__main__':unittest.main()
