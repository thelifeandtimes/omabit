import copy
import importlib.util
from pathlib import Path
import sys
import unittest
from types import SimpleNamespace
from unittest.mock import patch
sys.path.insert(0,str(Path(__file__).resolve().parents[1]/'fake-networks'))
import runner_app_probe as probe

def fixture():
    runner='runner-'+'a'*32;project='project-'+'b'*32
    return {'format':1,'runner_id':runner,'project_id':project,'anchor_id':'c'*64,'namespace':'net:[123]', 'anchor_labels':{'org.urbit-runner.project':'true','org.urbit-runner.project.runner':runner,'org.urbit-runner.project.project':project},'ships':{ship:{'http_port':8080+i,'pier':'/home/lab/piers/fakeships/vm-'+ship+'/'+ship,'identity':{'device':1,'inode':i+2,'urb_device':1,'urb_inode':i+8}} for i,ship in enumerate(('zod','nec'))}}

class ProbeTests(unittest.TestCase):
    def test_spec_refuses_foreign_identity_paths_and_ports(self):
        value=fixture();self.assertIs(probe.validate_spec(value),value)
        custom=fixture();custom['ships']['zod']['pier']='/srv/urbit/piers/fakeships/vm-zod/zod';probe.validate_spec(custom)
        for mutate in [lambda s:s.update(format=True),lambda s:s.update(peer_cycle=True),lambda s:s.update(peer_cycle=1),lambda s:s.update(peer_mode="unknown"),lambda s:s.update(namespace='host'),lambda s:s.update(anchor_id='name'),lambda s:s['anchor_labels'].update({'org.urbit-runner.project.runner':'other'}),lambda s:s['ships']['zod'].update(pier='/home/lab/../other'),lambda s:s['ships']['zod'].update(http_port=8081),lambda s:s['ships']['zod']['identity'].update(inode=0),lambda s:s['ships'].update(zod=None)]:
            broken=copy.deepcopy(value);mutate(broken)
            with self.subTest(spec=broken),self.assertRaises(ValueError):probe.validate_spec(broken)
    def test_host_guard_precedes_private_reads_and_engine_calls(self):
        with patch.object(sys,'argv',['probe','setup-before-first']),patch.object(probe.os,'geteuid',return_value=999),patch.object(probe,'private_json') as read,patch.object(probe.subprocess,'run') as run:
            with self.assertRaisesRegex(ValueError,'guest owner'):probe.main()
            read.assert_not_called();run.assert_not_called()
    def test_login_secret_only_stdin_and_redacted_journal(self):
        state={};saved=[];client=probe.Probe(fixture(),state,lambda:saved.append(copy.deepcopy(state)))
        secret=b'private-test-password'
        with patch.object(probe.subprocess,'run',return_value=SimpleNamespace(returncode=0,stdout=b'\n204',stderr=b'')) as run:
            self.assertEqual(client.http('zod','/~/login',secret,login=True),(204,b''))
            call=run.call_args;self.assertEqual(call.kwargs['input'],secret)
            self.assertNotIn(secret.decode(),repr(call.args));self.assertNotIn(secret.decode(),repr(state))
            self.assertIn('password@-',call.args[0]);self.assertIn('/urbit/omabit-runner-app-zod-cookie',call.args[0]);self.assertIn('unix:///run/user/1000/urbit-docker.sock',call.args[0])
        self.assertEqual(len(saved),1);self.assertEqual(state['requests'][0]['status'],204)
    def test_command_errors_do_not_include_output_or_secret(self):
        client=probe.Probe(fixture(),{},lambda:None)
        with patch.object(probe.subprocess,'run',return_value=SimpleNamespace(returncode=1,stdout=b'secret',stderr=b'private')):
            with self.assertRaisesRegex(RuntimeError,'^bounded private command failed$'):client.run(['inspect'])
    def test_exact_anchor_and_namespace_required(self):
        import json
        client=probe.Probe(fixture(),{},lambda:None)
        row={'Id':client.spec['anchor_id'],'Config':{'Labels':client.spec['anchor_labels']},'State':{'Running':True},'HostConfig':{'ReadonlyRootfs':True,'Mounts':[{'Type':'tmpfs','Target':'/urbit','TmpfsOptions':{'SizeBytes':1048576,'Mode':448}}]},'Mounts':[{'Type':'tmpfs','Destination':'/urbit','RW':True}]}
        with patch.object(client,'run',side_effect=[json.dumps([row]).encode(),b'net:[123]\n']):client.verify_anchor()
        for mutate in [lambda r:r.update(Id='d'*64),lambda r:r['Config'].update(Labels={}),lambda r:r['State'].update(Running=False),lambda r:r['Mounts'][0].update(Destination='/tmp'),lambda r:r['HostConfig'].update(ReadonlyRootfs=False)]:
            bad=copy.deepcopy(row);mutate(bad)
            with patch.object(client,'run',return_value=json.dumps([bad]).encode()),self.assertRaises(ValueError):client.verify_anchor()
        with patch.object(client,'run',side_effect=[json.dumps([row]).encode(),b'net:[124]\n']),self.assertRaises(ValueError):client.verify_anchor()
    def test_four_phases_login_once_then_reuse_retained_recovery(self):
        self.check_four_phases(False)
    def test_running_peer_sends_after_both_replacements(self):
        self.check_four_phases(True)
    def test_sender_replacement_reuses_sessions_and_existing_subscription(self):
        self.check_four_phases(True,True)
    def test_anchor_repair_reuses_both_sessions_and_subscription(self):
        self.check_four_phases(True,True,True)
    def check_four_phases(self,live_peer,peer_cycle=False,anchor_cycle=False):
        calls=[]
        class FakeProbe:
            def __init__(self,*args):pass
            def verify_anchor(self):calls.append('anchor')
            def login_once(self,ship):calls.append('login-'+ship)
            def http(self,*a,**kw):raise AssertionError('unexpected real request')
            def events(self,*a,**kw):raise AssertionError('unexpected real stream')
        class FakeRecovery:
            def __init__(self,*a,**kw):
                calls.append((kw['target'],kw['sender']));self.entries=[];self.channel=None;self.cursor=None;self.report={'writes':[]}
            def install(self,*a):calls.append('install')
            def before_restart(self):calls.append('before')
            def after_restart(self):calls.append('after');self.report['ok']=True
        spec=fixture()
        if live_peer:spec['peer_mode']='running'
        state={'format':1,'attempts':[]};commits=[]
        if peer_cycle:spec['peer_cycle']=True;state['peer_cycle']=True
        if anchor_cycle:spec['anchor_cycle']=True;state['anchor_cycle']=True
        cycles=4 if anchor_cycle else 3 if peer_cycle else 2
        with patch.object(probe,'Probe',FakeProbe),patch.object(probe,'Recovery',FakeRecovery):
            for phase in (probe.ANCHOR_PHASES if anchor_cycle else probe.PEER_PHASES if peer_cycle else probe.PHASES):
                state=probe.execute_phase(state,phase,commits.append,lambda current:probe.run_phase_work(spec,current,phase,commits.append))
        self.assertTrue(state['ok']);self.assertEqual(len(state['completed_cycles']),cycles)
        self.assertEqual(calls.count('login-zod'),1);self.assertEqual(calls.count('login-nec'),1);self.assertEqual(calls.count('install'),1)
        self.assertEqual(calls.count('anchor'),cycles*2);self.assertEqual(calls.count('before'),cycles);self.assertEqual(calls.count('after'),cycles)
        self.assertEqual(calls.count(('zod','nec')),cycles*2 if live_peer else 1);self.assertEqual(calls.count(('zod','zod')),0 if live_peer else 3)

class PublishedProbeTests(unittest.TestCase):
    def spec(self):
        value=fixture();value['http_transport']='published-loopback'
        value['ships']['zod']['published_port']=18381;value['ships']['nec']['published_port']=18382
        return value
    def test_published_spec_binds_exact_guest_ports(self):
        probe.validate_spec(self.spec())
        for mutate in [lambda s:s['ships']['zod'].update(published_port=18382),lambda s:s['ships']['nec'].pop('published_port'),lambda s:s['ships']['nec'].update(published_port=True),lambda s:s.update(http_transport='external')]:
            value=self.spec();mutate(value)
            with self.assertRaises(ValueError):probe.validate_spec(value)
    def test_direct_login_uses_stdin_and_session_survives_new_probe(self):
        import tempfile,os,json
        with tempfile.TemporaryDirectory() as tmp:
            root=Path(tmp);state={};response=SimpleNamespace(returncode=0,stdout=b'\n204',stderr=b'')
            client=probe.Probe(self.spec(),state,lambda:None)
            with patch.object(client,'session_root',return_value=root),patch.object(probe.subprocess,'run',return_value=response) as run:
                client.http('nec','/~/login',b'private-test-code',login=True)
                argv=run.call_args.args[0];self.assertEqual(argv[0],'/usr/bin/curl')
                self.assertNotIn('docker',' '.join(argv));self.assertIn('http://127.0.0.1:18382/~/login',argv)
                self.assertEqual(run.call_args.kwargs['input'],b'private-test-code');self.assertNotIn('private-test-code',repr(argv))
                self.assertEqual((root/'nec.cookie').stat().st_mode&0o777,0o600)
                (root/'nec.cookie').write_bytes(b'private-test-cookie')
                with self.assertRaises(FileExistsError):client.http('nec','/~/login',b'no-relogin',login=True)
                self.assertEqual(run.call_count,1)
            restarted=probe.Probe(self.spec(),state,lambda:None)
            with patch.object(restarted,'session_root',return_value=root),patch.object(probe.subprocess,'run',return_value=SimpleNamespace(returncode=0,stdout=b'{}\n200',stderr=b'')) as run:
                restarted.http('nec','/~/scry/hood/kiln/pikes.json')
                self.assertIn(str(root/'nec.cookie'),run.call_args.args[0]);self.assertNotIn('--cookie-jar',run.call_args.args[0])
            self.assertNotIn('private-test',json.dumps(state))
    def test_untrusted_cookie_files_never_invoke_client(self):
        import tempfile,os
        with tempfile.TemporaryDirectory() as tmp:
            root=Path(tmp);client=probe.Probe(self.spec(),{},lambda:None)
            with patch.object(client,'session_root',return_value=root),patch.object(probe.subprocess,'run') as run:
                path=root/'zod.cookie';path.write_text('cookie');path.chmod(0o644)
                with self.assertRaises(ValueError):client.http('zod','/~/host')
                path.chmod(0o600);os.link(path,root/'linked')
                with self.assertRaises(ValueError):client.http('zod','/~/host')
                path.unlink();path.symlink_to(root/'linked')
                with self.assertRaises(ValueError):client.http('zod','/~/host')
                path.unlink();path.mkdir()
                with self.assertRaises(ValueError):client.http('zod','/~/host')
                run.assert_not_called()
    def test_runtime_storage_refuses_non_tmpfs_before_creating_directory(self):
        import tempfile
        with tempfile.TemporaryDirectory() as tmp:
            root=Path(tmp);root.chmod(0o700);client=probe.Probe(self.spec(),{},lambda:None)
            with patch.object(probe,'SESSION_ROOT',root/'session'),patch.object(Path,'read_text',return_value='1 0 0:1 / / rw - ext4 /dev/root rw\n'):
                with self.assertRaisesRegex(ValueError,'tmpfs'):client.session_root()
            self.assertFalse((root/'session').exists())
            mounts='1 0 0:1 / / rw - ext4 /dev/root rw\n2 1 0:2 / '+str(root)+' rw - tmpfs tmpfs rw\n'
            with patch.object(probe,'SESSION_ROOT',root/'session'),patch.object(Path,'read_text',return_value=mounts):
                self.assertEqual(client.session_root(),root/'session');self.assertEqual(client.session_root(),root/'session')
                self.assertEqual((root/'session').stat().st_mode&0o777,0o700)
    def test_admin_stays_in_anchor_and_direct_failure_never_replays(self):
        client=probe.Probe(self.spec(),{},lambda:None);client.spec['ships']['zod']['admin_port']=12345
        with patch.object(client,'run',return_value=b'~zod\n200') as engine,patch.object(client,'run_guest_curl') as direct:
            client.http('zod','/',b'read-only-dojo',admin=True)
            direct.assert_not_called();self.assertIn('http://127.0.0.1:12345/',engine.call_args.args[0])
        with patch.object(probe.subprocess,'run',return_value=SimpleNamespace(returncode=7,stdout=b'private-cookie',stderr=b'private-code')) as run:
            with self.assertRaisesRegex(RuntimeError,'not replayed') as error:client.run_guest_curl(['/usr/bin/curl','--data-binary','@-'],data=b'private-poke')
            self.assertEqual(run.call_count,1);self.assertNotIn('private-cookie',str(error.exception));self.assertNotIn('private-code',str(error.exception))
    def test_direct_sse_retains_cursor_and_cookie_without_engine_exec(self):
        client=probe.Probe(self.spec(),{},lambda:None)
        expected={'revision':1,'entries':['entry']};channel='/~/channel/omabit-state-'+'a'*32
        import json
        event={'id':1,'response':'diff','json':expected}
        response=('id: 8\ndata: '+json.dumps(event)+'\n\n\n200').encode()
        with patch.object(client,'published_cookie',return_value='/run/user/1000/private-cookie'),patch.object(client,'run_guest_curl',return_value=response) as direct,patch.object(client,'run') as engine:
            rows=client.events(channel,7,expected,False)
            self.assertEqual(rows,[(8,event)]);engine.assert_not_called()
            argv=direct.call_args.args[0];self.assertIn('Last-Event-ID: 7',argv);self.assertIn('http://127.0.0.1:18381'+channel,argv)
            self.assertIn('/run/user/1000/private-cookie',argv);self.assertEqual(direct.call_args.kwargs['allowed'],(0,28))


class SuccessionTests(unittest.TestCase):
    def values(self):
        spec=fixture();spec.update(peer_mode='running',peer_cycle=True,anchor_cycle=True)
        state={'spec_sha256':'d'*64,'attempts':[{'phase':'after-anchor','status':'running'}]}
        receipt={key:spec[key] for key in ('runner_id','project_id','anchor_id')}
        receipt.update(format=1,spec_sha256='d'*64,previous_namespace=spec['namespace'],namespace='net:[124]',operation_id='op-'+'e'*32)
        return spec,state,receipt
    def test_only_exact_completed_recovery_receipt_can_change_namespace(self):
        spec,state,receipt=self.values()
        before=copy.deepcopy(spec)
        self.assertEqual(probe.validate_succession(spec,state,receipt),'net:[124]');self.assertEqual(spec,before)
        for mutate in [lambda r:r.update(format=True),lambda r:r.update(project_id='foreign'),lambda r:r.update(spec_sha256='a'*64),lambda r:r.update(previous_namespace='net:[122]'),lambda r:r.update(namespace='net:[123]'),lambda r:r.update(namespace='host'),lambda r:r.update(operation_id='foreign'),lambda r:r.update(extra=True)]:
            bad=copy.deepcopy(receipt);mutate(bad)
            with self.assertRaises(ValueError):probe.validate_succession(spec,state,bad)
        for phase in probe.PEER_PHASES+('before-anchor',):
            bad=copy.deepcopy(state);bad['attempts'][-1]['phase']=phase
            with self.assertRaises(ValueError):probe.validate_succession(spec,bad,receipt)
    def test_probe_accepts_only_observed_successor_and_retains_receipt_digest(self):
        import json
        spec,state,receipt=self.values();client=probe.Probe(spec,state,lambda:None)
        row={'Id':spec['anchor_id'],'Config':{'Labels':spec['anchor_labels']},'State':{'Running':True},'HostConfig':{'ReadonlyRootfs':True,'Mounts':[{'Type':'tmpfs','Target':'/urbit','TmpfsOptions':{'SizeBytes':1048576,'Mode':448}}]},'Mounts':[{'Type':'tmpfs','Destination':'/urbit','RW':True}]}
        for namespace in ('net:[124]','net:[125]'):
            with patch.object(probe,'private_json',return_value=(receipt,'f'*64)),patch.object(client,'run',side_effect=[json.dumps([row]).encode(),namespace.encode()]):
                if namespace=='net:[124]':client.verify_anchor()
                else:
                    with self.assertRaises(ValueError):client.verify_anchor()
        self.assertEqual(state['succession'],{'receipt':receipt,'sha256':'f'*64})

if __name__=='__main__':unittest.main()
