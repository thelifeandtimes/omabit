"""Reject false isolation evidence; no Docker/runtime or network access."""
import copy
import json
import unittest
from types import SimpleNamespace
from project_isolation import verify


def pair():
    contexts=[]
    for project in ('a','b'):
        anchor='anchor-'+project
        report={'project':'omabit-network-'+project,'anchor':anchor,'network_namespace':'net-'+project,
                'ships':{name:{'container':project+'-'+name,'http_port':port} for name,port in [('zod',8081),('nec',8082)]}}
        context={'report':report,'cookie_status':401,'actual':{'revision':4,'entries':[project+str(n) for n in range(4)]},'cross_to':None}
        class Recovery:
            def __init__(self,owner):self.owner=owner;self.state=copy.deepcopy(owner['actual'])
            def expected(self):return copy.deepcopy(self.state)
            def verify_state(self,status,body):
                if status!=200 or json.loads(body)!=self.state:raise RuntimeError('durable state mismatch')
            def write(self,phase):
                self.state['revision']+=1;self.state['entries'].append(phase)
                self.owner['actual']=copy.deepcopy(self.state)
                if self.owner['cross_to'] is not None:self.owner['cross_to']['actual']=copy.deepcopy(self.state)
        context['recovery']=Recovery(context)
        def inspect(identity,c=context):
            assert identity in [v['container'] for v in c['report']['ships'].values()]
            return {'State':{'Running':True,'OOMKilled':False},'HostConfig':{'PortBindings':{},'NetworkMode':'container:'+c['report']['anchor']}}
        context['inspect']=inspect
        def execute(args,data=None,c=context):
            if '/bin/cat' in args:return b'private-test-cookie'
            return b''
        context['execute']=execute
        context['http']=lambda *a,c=context,**kw:(c['cookie_status'],b'')
        context['state_http']=lambda *a,c=context,**kw:(200,json.dumps(c['actual']).encode())
        contexts.append(context)
    return contexts


class ProjectIsolationTests(unittest.TestCase):
    def test_live_separate_pairs_and_independent_mutations(self):
        a,b=pair();result=verify(a,b)
        self.assertTrue(result['ok']);self.assertEqual(len(result['cross_cookie_checks']),4)
        self.assertEqual(a['actual']['revision'],5);self.assertEqual(b['actual']['revision'],5)
        self.assertNotEqual(a['actual']['entries'],b['actual']['entries'])

    def test_shared_namespace_is_refused_before_mutation(self):
        a,b=pair();b['report']['network_namespace']=a['report']['network_namespace']
        with self.assertRaisesRegex(RuntimeError,'share a namespace'):verify(a,b)
        self.assertEqual(a['actual']['revision'],4)

    def test_same_identity_cookie_acceptance_is_not_isolation(self):
        a,b=pair();b['cookie_status']=200
        with self.assertRaisesRegex(RuntimeError,'session crossed'):verify(a,b)
        self.assertEqual(a['actual']['revision'],4)

    def test_crossed_state_fails_even_with_separate_namespace_ids(self):
        a,b=pair();a['cross_to']=b
        with self.assertRaisesRegex(RuntimeError,'durable state mismatch'):verify(a,b)

    def test_inactive_peer_does_not_count_as_concurrent_isolation(self):
        a,b=pair();b['inspect']=lambda _:{'State':{'Running':False,'OOMKilled':False}}
        with self.assertRaisesRegex(RuntimeError,'both live'):verify(a,b)


def published_pair():
    contexts=pair()
    for i,context in enumerate(contexts):
        bindings={8081:18281,8082:18282} if i==0 else {8081:18291,8082:18292}
        context['report']['publication']={'bridge':'bridge-'+str(i)}
        context['foreign_checks']=[]
        def reject(source,ship,c=context):
            c['foreign_checks'].append(ship)
            if c['cookie_status'] not in (401,403):raise RuntimeError('session crossed')
            return c['cookie_status']
        context['published']=SimpleNamespace(bindings=bindings,reject_foreign_session=reject)
        previous=context['inspect']
        def inspect(identity,c=context,old=previous):
            if identity!=c['report']['anchor']:return old(identity)
            ports={str(p)+'/tcp':[{'HostIp':'127.0.0.1','HostPort':str(h)}] for p,h in c['published'].bindings.items()}
            return {'HostConfig':{'PortBindings':ports},'NetworkSettings':{'Ports':ports}}
        context['inspect']=inspect
        def no_transfer(*a,**kw):raise AssertionError('published pair attempted cookie file transfer')
        context['execute']=no_transfer
    return contexts

class PublishedProjectIsolationTests(unittest.TestCase):
    def test_published_pair_uses_distinct_ports_and_memory_sessions(self):
        a,b=published_pair();result=verify(a,b)
        self.assertTrue(result['ok']);self.assertTrue(result['direct_publication']);self.assertFalse(result['no_published_ports'])
        self.assertEqual(a['foreign_checks'],['zod','nec']);self.assertEqual(b['foreign_checks'],['zod','nec'])
        self.assertNotEqual(a['actual']['entries'],b['actual']['entries'])

    def test_shared_endpoints_bridge_or_mixed_mode_refused(self):
        for mode in ('ports','bridge','mixed'):
            a,b=published_pair()
            if mode=='ports':b['published'].bindings=a['published'].bindings
            elif mode=='bridge':b['report']['publication']['bridge']=a['report']['publication']['bridge']
            else:b['published']=None
            with self.subTest(mode=mode),self.assertRaises(RuntimeError):verify(a,b)
            self.assertEqual(a['actual']['revision'],4)
            self.assertEqual(a['foreign_checks'],[])

if __name__=='__main__':unittest.main()
