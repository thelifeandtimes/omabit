"""Offline anchor recovery ordering/refusal tests; no Docker or network calls."""
import copy
import json
from pathlib import Path
from types import SimpleNamespace
import tempfile
import unittest
from anchor_recovery import recover, endpoint_probe


def fixture(root):
    bindings={8081:18281,8082:18282};records={};spaces={};actions=[]
    ports={str(p)+'/tcp':[{'HostIp':'127.0.0.1','HostPort':str(h)}] for p,h in bindings.items()}
    def record(network,bindings):
        return {'State':{'Running':True,'ExitCode':0,'OOMKilled':False},'HostConfig':{'NetworkMode':network,'PortBindings':copy.deepcopy(bindings)},'NetworkSettings':{'Ports':copy.deepcopy(bindings)}}
    records['anchor']=record('bridge',ports);spaces['anchor']='old'
    report={'project':'omabit-network-a','anchor':'anchor','network_namespace':'old','ships':{}}
    for ship,port in [('zod',8081),('nec',8082)]:
        p=root/ship/ship;p.mkdir(parents=True)
        report['ships'][ship]={'container':ship,'pier':str(p),'http_port':port}
        records[ship]=record('container:anchor',{});spaces[ship]='old'
    class State:
        channel=None;cursor=None;count=4
        def expected(self):return {'revision':self.count,'entries':list(range(self.count))}
        def subscribe(self):self.channel='channel';self.cursor=1
        def write(self,phase):actions.append(('write',phase));self.count+=1
        def verify_state(self,status,body):
            if status!=200 or json.loads(body)!=self.expected():raise RuntimeError('state changed')
        def receive(self):self.cursor+=1
        def close(self):self.channel=None
    state=State()
    context={'report':report,'published':SimpleNamespace(bindings=bindings),'recovery':state,'save':lambda:None}
    context['inspect']=lambda identity:copy.deepcopy(records[identity])
    context['netns']=lambda identity:spaces[identity]
    def execute(args,**kw):
        actions.append(tuple(args));identity=args[-1]
        if args[0]=='stop':records[identity]['State']['Running']=False
        if args[0]=='start':
            records[identity]['State']['Running']=True
            if identity=='anchor':spaces['anchor']='new'
        return b''
    context['execute']=execute
    def stop(ship):
        actions.append(('stop-member',ship));records[report['ships'][ship]['container']]['State']['Running']=False
    context['stop']=stop
    def replace(ship,port,parent):
        if any(records[name]['State']['Running'] for name in ('zod','nec')):raise AssertionError('replacement started before all members stopped')
        actions.append(('replace',ship));identity=ship+'-new';records[identity]=record('container:anchor',{});records[identity]['State']['Running']=False;spaces[identity]=spaces['anchor'];return identity
    context['replacement']=replace
    context['ready']=lambda ship,timeout:actions.append(('ready',ship))
    context['state_http']=lambda *args:(200,json.dumps(state.expected()).encode())
    context['refuse_endpoint']=lambda port:endpoint_probe(7,b'\n000',b'')
    return context,records,spaces,actions


class EndpointProbeTests(unittest.TestCase):
    def test_transport_failures_require_no_http_response(self):
        for code in (7,28,52,56):
            self.assertTrue(endpoint_probe(code,b'\n000',b'')['unavailable'])
        for code,body,error in [(0,b'~zod\n200',b''),(56,b'\n200',b''),(52,b'partial\n000',b''),(28,b'\n404',b''),(52,b'',b''),(52,b'\n000',b'x'*8193),(60,b'\n000',b'')]:
            self.assertFalse(endpoint_probe(code,body,error)['unavailable'])

    def test_evidence_never_retains_response_or_error_text(self):
        facts=endpoint_probe(52,b'synthetic-sensitive-body\n000',b'synthetic-sensitive-error')
        self.assertNotIn('synthetic-sensitive',json.dumps(facts))
        self.assertEqual(set(facts),{'exit','status','response_bytes','stderr_bytes','unavailable'})


class AnchorRecoveryTests(unittest.TestCase):
    def test_all_old_members_stop_before_replacement_and_state_recovers(self):
        with tempfile.TemporaryDirectory() as temp:
            c,records,spaces,actions=fixture(Path(temp));recover(c);r=c['report']['anchor_recovery']
            self.assertTrue(r['ok']);self.assertEqual(r['old_namespace'],'old');self.assertEqual(r['new_namespace'],'new')
            self.assertEqual(r['old_members_retained'],['zod','nec']);self.assertEqual(len(r['endpoint_refusals']),4)
            self.assertEqual(r['final_state']['revision'],6);self.assertEqual(r['cursor_recovered'],2)
            self.assertEqual([a for a in actions if a[0] in ('stop-member','replace')],[('stop-member','nec'),('stop-member','zod'),('replace','zod'),('replace','nec')])
            self.assertFalse(records['zod']['State']['Running']);self.assertFalse(records['nec']['State']['Running'])
            self.assertEqual(c['recovery'].channel,None)

    def test_unchanged_namespace_does_not_count_as_anchor_replacement(self):
        with tempfile.TemporaryDirectory() as temp:
            c,records,spaces,actions=fixture(Path(temp));execute=c['execute']
            def unchanged(args,**kw):
                execute(args,**kw)
                if args==['start','anchor']:spaces['anchor']='old'
            c['execute']=unchanged
            with self.assertRaisesRegex(RuntimeError,'distinct namespace'):recover(c)
            self.assertFalse(any(a[0]=='replace' for a in actions))

    def test_old_member_drift_refuses_before_stop_or_adoption(self):
        with tempfile.TemporaryDirectory() as temp:
            c,records,spaces,actions=fixture(Path(temp));execute=c['execute']
            def drift(args,**kw):
                execute(args,**kw)
                if args==['start','anchor']:spaces['nec']='new'
            c['execute']=drift
            with self.assertRaisesRegex(RuntimeError,'unexpectedly migrated'):recover(c)
            self.assertFalse(any(a[0] in ('replace','stop-member') for a in actions))

    def test_member_that_did_not_stop_cannot_be_replaced(self):
        with tempfile.TemporaryDirectory() as temp:
            c,records,spaces,actions=fixture(Path(temp));c['stop']=lambda ship:None
            with self.assertRaisesRegex(RuntimeError,'remains live'):recover(c)
            self.assertFalse(any(a[0]=='replace' for a in actions))

if __name__=='__main__':unittest.main()
