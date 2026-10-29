#!/usr/bin/env python3
"""Repeated format4 authorities with actual private guards and synthetic apps."""
import copy,ctypes,importlib.util,json,os,select,signal,subprocess,sys,time,unittest,uuid
from pathlib import Path
from unittest.mock import patch
ROOT=Path(__file__).resolve().parents[1];sys.path.insert(0,str(ROOT/'native/collections'))
import admission as old
import runtime_authority as authority
import repeated_authority as linked
import repeated_admission as shared
import repeated_session as session
import repeated_checkpoint as checkpoint
import repeated_inspection as inspection
import repeated_cleanup as cleanup
import journal as j

def fixture(name):
    spec=importlib.util.spec_from_file_location(name.replace('-','_'),Path(__file__).with_name(name+'.py'))
    module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module);return module
f=fixture('test-collection-repeated-session');model=fixture('test-collection-journal')
save=f.current.save

class CleanupAdapter:
    def __init__(self,members):self.members=copy.deepcopy(members);self.recovers=0;self.fail=False
    def prepare_desktop(self,plan):
        members=copy.deepcopy(self.members)
        for role,row in members.items():
            row.pop('closed',None);row['binding']=plan['loaded']['members'][role]
            row['channel_ids']=[uuid.uuid4().hex] if role!='omabit' else []
            row['transport_sha256']=j.digest(['new-session',role,plan['loaded']['epoch']])
        return members
    def recover_desktop(self,plan):
        self.recovers+=1
        if self.fail:raise RuntimeError('synthetic cleanup refusal')
        members=copy.deepcopy(plan['members'])
        for role,row in members.items():row.update(closed=True,transport_sha256=j.digest(['closed-repeat',role]))
        return {'plan_sha256':j.digest(plan),'guardian':plan['guardian'],'closure_request':plan['closure_request'],
                'closure_receipt_sha256':plan['closure_receipt_sha256'],'proof':'waitpid-echild','members':members}

class RuntimeFlow:
    def runtime_setup(self):
        self.repeat_authority=None;self.repeat_authorities=[];self.repeat_generation=0
        self.addCleanup(lambda:[x.close() for x in self.repeat_authorities])
    def publish_repeat(self):
        with self.store.lease() as lease:return linked.step(self.display,self.store,lease,self.repeat_observer,self.repeat_attempt)
    def open_repeat(self):
        self.finish_repeat()
        for _ in range(5):self.publish_repeat()
        self.repeat_authority=linked.Authority(self.runtime/authority.ANCHOR);self.repeat_authorities.append(self.repeat_authority)
        return self.repeat_authority
    def repeat_event(self,**values):
        request=self.store.status()['requests'][self.repeat_transition]
        return dict(request=self.repeat_transition,phase=request['phase'],effect=j.effect(request,request['phase']),**values)
    def prepare_repeat_activation(self):
        self.open_repeat();intent=session.read(self.store,self.repeat_attempt)
        self.repeat_spec=copy.deepcopy(intent['specification']);self.repeat_transition=uuid.uuid4().hex
        self.record({'type':'begin','request':self.repeat_transition,'candidate':j.digest(self.repeat_spec),'expected':j.digest(intent['loaded'])})
        for role,row in intent['members'].items():
            self.record(self.repeat_event(type='drained',role=role,observation={'epoch':intent['loaded']['epoch'],'binding':intent['loaded']['members'][role],
                'absent':False,'closed':True,'retained_state':row['application_sha256'],'pending_ids':row['pending_ids']}))
        self.record(self.repeat_event(type='completed',observation={'epoch':intent['loaded']['epoch'],'absent':True,'helpers_exited':True}))
        self.record(self.repeat_event(type='completed',observation={'checkpoint':j.digest(['repeat-appearance'])}))
        self.guardian,self.owner,self.shell,self.repeat_receipt=self.launch_guardian()
        request=self.store.status()['requests'][self.repeat_transition]
        self.repeat_gate_root=self.root/('gate-repeat-'+uuid.uuid4().hex);self.repeat_gate_root.mkdir(mode=0o700)
        self.repeat_permit={'id':uuid.uuid4().hex,'request':self.repeat_transition,'epoch':j.effect(request,'enable-candidate'),'specification':self.repeat_spec,'shell':self.shell}
    def reserve_repeat(self):
        with self.store.lease() as lease:return self.repeat_authority.reserve(self.store,lease,self.repeat_gate_root,self.repeat_permit,guardian_receipt=self.repeat_receipt,adapter=self.repeat_observer)
    def bind_repeat(self):
        self.repeat_ticket=self.reserve_repeat();self.repeat_gate=shared.Gate(self.repeat_gate_root);self.gates.append(self.repeat_gate)
        self.repeat_gate.issue(self.repeat_permit,self.repeat_ticket)
        guards=self.send(self.guardian,{'command':'start','permit':self.repeat_permit,'gate':str(self.repeat_gate_root)})['guards']
        self.assertEqual(len(guards),3);self.assertTrue(all(x['ok'] for x in guards),guards)
        loaded=model.loaded(self.repeat_spec,self.repeat_permit['epoch']);self.record(self.repeat_event(type='completed',observation=loaded))
        row=self.repeat_gate.status()['record'];binding=self.root/('binding-repeat-'+uuid.uuid4().hex+'.json')
        save(binding,{'loaded':loaded,'helpers':{role:[claim['owner']] for role,claim in row['claims'].items()},'shell':self.shell,
                      'supervisor':self.owner,'admission':{'root':str(self.repeat_gate_root),'permit':self.repeat_permit}})
        with self.store.lease() as lease:self.repeat_authority.bind_loaded(self.store,lease,self.repeat_ticket,binding,self.repeat_receipt)
        self.record(self.repeat_event(type='completed',observation=model.ready(self.repeat_spec,loaded)))
        return loaded
    def close_repeat(self):
        self.assertEqual(self.send(self.guardian,{'command':'end'})['exits'],[0,0,0]);self.close_guardian();self.stop_display()
        self.repeat_cleanup_request=uuid.uuid4().hex;self.repeat_cleanup_adapter=CleanupAdapter(self.repeat_observer.members)
    def repeat_cleanup_step(self):
        with self.store.lease() as lease:return cleanup.step(self.repeat_authority.root,self.store,lease,self.repeat_cleanup_adapter,self.repeat_cleanup_request)
    def repeat_capture(self):
        with self.store.lease() as lease:return checkpoint.capture(self.repeat_authority.root,self.store,lease,self.repeat_cleanup_request)
    def retire_repeat(self,record):
        path=self.store.root/checkpoint.filename(record['id']);self.repeat_authority.close();self.repeat_generation+=1
        self.runtime.rename(self.root/('repeat-retained-runtime-'+str(self.repeat_generation)));self.runtime.mkdir(mode=0o700)
        self.initialize_repeat(record,path)

class RepeatedRuntimeTests(RuntimeFlow,unittest.TestCase):
    def setUp(self):f.RepeatedIntentTests.setUp(self);self.runtime_setup()
    def test_three_sessions_reuse_formats_and_artifact_identifiers(self):
        original_members=copy.deepcopy(self.repeat_observer.members);original_spec=copy.deepcopy(self.repeat_record['inspection']['reservation']['reservation']['permit']['specification'])
        summaries=[]
        for index in range(3):
            self.prepare_repeat_activation();loaded=self.bind_repeat();self.assertEqual(self.repeat_spec,original_spec)
            live=inspection.inspect(self.repeat_authority.root,self.store.root);self.assertEqual(live['format'],4);self.assertFalse(live['activation_authorized'])
            self.assertEqual(live['journal']['committed'],loaded)
            self.close_repeat();self.assertIsNone(self.repeat_cleanup_step()['verified']);clean=self.repeat_cleanup_step()
            before=self.snapshot();self.assertEqual(self.repeat_cleanup_step(),clean);self.assertEqual(before,self.snapshot())
            record=self.repeat_capture();self.assertEqual(record['format'],3)
            chain=checkpoint.checkpoint_lineage.read(session.reference(self.store.root/checkpoint.filename(record['id'])),self.store.root)
            self.assertEqual(chain['usage']['nodes'],4+index)
            for ref in chain['references'][1:]:self.assertNotIn(ref['object']['path'],record['files'])
            for role,row in clean['verified']['members'].items():
                for key in ('application_sha256','pending_ids','credential_refs'):self.assertEqual(row[key],original_members[role][key])
            summaries.append({'authority':4,'intent':3,'checkpoint':3,'cleanup':4,'depth':chain['usage']['nodes'],'loaded':loaded})
            self.retire_repeat(record)
            with self.store.lease() as lease:self.assertEqual(checkpoint.read(self.store,record['id'],lease),record)
        if ROOT==Path('/lab/source'):
            save(Path('/lab/repeated-runtime-result.json'),{'passed':True,'sessions':summaries,'same_specification':original_spec,
                'real_guard_admissions':9,'new_application_instances':0,'application_observations':'synthetic','all_ancestor_bodies_excluded':True,
                'checkpoints':chain['records'],'references':chain['references'],'canonical_events':self.store._load()[0],
                'original_cohort':original_members})
    def test_publication_steps_and_exact_retry_preserve_identity(self):
        self.finish_repeat();phases=[self.publish_repeat()['phase'] for _ in range(5)]
        self.assertEqual(phases,['prepared','seal-published','mirror-published','root-published','complete'])
        before=self.snapshot();self.assertEqual(self.publish_repeat()['phase'],'complete');self.assertEqual(before,self.snapshot())
    def test_earlier_authority_reader_refuses_new_descriptor(self):
        self.open_repeat();before=self.snapshot()
        with self.assertRaises(ValueError):linked.previous.Authority(self.repeat_authority.root)
        self.assertEqual(before,self.snapshot())
    def test_unfinished_fences_do_not_implicitly_advance_from_publication(self):
        self.repeat_step();before=self.snapshot()
        with self.assertRaises(OSError):self.publish_repeat()
        self.assertEqual(before,self.snapshot());self.assertFalse((self.runtime/authority.ANCHOR).exists())
    def test_missing_fence_blocks_publication_before_new_authority(self):
        value=self.finish_repeat()['intent'];Path(value['fences']['successor']['target']['path']).unlink();before=self.snapshot()
        with self.assertRaises(OSError):self.publish_repeat()
        self.assertEqual(before,self.snapshot());self.assertFalse((self.runtime/authority.ANCHOR).exists())
    def test_four_lost_publication_replies_preserve_one_descriptor(self):
        self.finish_repeat();original=linked.provisioning.rename_no_replace;saved=None
        def lost(*args):original(*args);raise OSError('lost authority reply')
        for _ in range(4):
            with patch.object(linked.provisioning,'rename_no_replace',lost),self.assertRaises(OSError):self.publish_repeat()
            value=linked.read(self.store,self.repeat_attempt)
            if saved is None:saved=value
            self.assertEqual(value,saved)
        self.assertEqual(self.publish_repeat()['phase'],'complete')
    def test_missing_published_mirror_is_not_recreated(self):
        self.open_repeat();Path(self.repeat_authority.descriptor['mirror']).unlink();before=self.snapshot()
        with self.assertRaises(OSError):self.publish_repeat()
        self.assertEqual(before,self.snapshot())
    def test_changed_cohort_blocks_reservation(self):
        self.prepare_repeat_activation();self.repeat_observer.members['dms']['pending_ids']=[];before=self.snapshot()
        with self.assertRaises(ValueError):self.reserve_repeat()
        self.assertEqual(before,self.snapshot());self.assertIsNone(self.repeat_authority.status()['reservation'])
    def test_real_guards_survive_later_canonical_appends(self):
        self.prepare_repeat_activation();self.bind_repeat()
        for index in range(24):
            spec=copy.deepcopy(self.repeat_spec)
            for role in j.ROLES:spec['members'][role]['artifact_sha256']=j.digest(['repeated-append',index,role])
            self.record({'type':'register','spec':spec})
        time.sleep(.6);self.assertEqual(self.send(self.guardian,{'command':'end'})['exits'],[0,0,0])
    def test_failed_cleanup_preserves_prepared_receipt(self):
        self.prepare_repeat_activation();self.bind_repeat();self.close_repeat();value=self.repeat_cleanup_step();self.repeat_cleanup_adapter.fail=True
        before=self.snapshot()
        with self.assertRaises(RuntimeError):self.repeat_cleanup_step()
        self.assertEqual(before,self.snapshot());self.repeat_cleanup_adapter.fail=False
        self.assertEqual(self.repeat_cleanup_step()['plan'],value['plan'])
    def test_missing_fence_blocks_inspection_without_repair(self):
        self.open_repeat();value=session.read(self.store,self.repeat_attempt);Path(value['fences']['continuation']['target']['path']).unlink()
        before=self.snapshot()
        with self.assertRaises(OSError):inspection.inspect(self.repeat_authority.root,self.store.root)
        self.assertEqual(before,self.snapshot())
    def test_unknown_nested_authority_in_checkpoint_refuses(self):
        self.prepare_repeat_activation();self.bind_repeat();self.close_repeat();self.repeat_cleanup_step();self.repeat_cleanup_step();value=self.repeat_capture()
        value['inspection']['format']=99;f.current.redigest(value)
        with self.assertRaises(ValueError):checkpoint.validate(value)

class FirstCheckpointRuntimeTests(RuntimeFlow,unittest.TestCase):
    def setUp(self):f.FirstCheckpointImportTests.setUp(self);self.runtime_setup()
    def test_direct_first_checkpoint_activation_and_checkpoint3(self):
        self.prepare_repeat_activation();self.bind_repeat();self.close_repeat();self.repeat_cleanup_step();self.repeat_cleanup_step()
        value=self.repeat_capture();self.assertEqual(value['format'],3);self.assertEqual(value['inspection']['lineage']['ancestors'],[])

class SecondCheckpointRuntimeTests(RuntimeFlow,unittest.TestCase):
    def setUp(self):f.SecondCheckpointImportTests.setUp(self);self.runtime_setup()
    def test_direct_second_checkpoint_activation_and_checkpoint3(self):
        self.prepare_repeat_activation();self.bind_repeat();self.close_repeat();self.repeat_cleanup_step();self.repeat_cleanup_step()
        value=self.repeat_capture();self.assertEqual(value['format'],3);self.assertEqual(len(value['inspection']['lineage']['ancestors']),1)

for target,source in ((RepeatedRuntimeTests,f.RepeatedIntentTests),(FirstCheckpointRuntimeTests,f.FirstCheckpointImportTests),
                       (SecondCheckpointRuntimeTests,f.SecondCheckpointImportTests)):
    for name in dir(source):
        value=getattr(source,name)
        if callable(value) and not name.startswith(('test','_')) and not hasattr(target,name):setattr(target,name,value)
del target,source


def launch_guardian(self):
    token = uuid.uuid4().hex
    receipt = self.root / ('guardian-' + token + '.json')
    env = dict(os.environ, WAYLAND_DISPLAY=self.display.name, XDG_RUNTIME_DIR=str(self.runtime))
    child = subprocess.Popen([sys.executable, '-B', __file__, '--guardian', token, str(receipt)], env=env,
                             stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    self.children.append(child)
    value = self.receive(child)
    self.assertEqual(child.pid, value['owner']['pid'])
    item = (child, value['owner'], value['shell'], receipt)
    self.guardians.append(item)
    return item

for target in (RepeatedRuntimeTests,FirstCheckpointRuntimeTests,SecondCheckpointRuntimeTests):target.launch_guardian=launch_guardian
del target

def guardian_worker(token, path):
    path = Path(path)
    libc = ctypes.CDLL(None, use_errno=True)
    assert libc.prctl(36, 1, 0, 0, 0) == 0
    flag = ctypes.c_int()
    assert libc.prctl(37, ctypes.byref(flag), 0, 0, 0) == 0 and flag.value == 1
    child = subprocess.Popen([sys.executable, '-B', __file__, '--shell'],
                             stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    assert select.select([child.stdout], [], [], 5)[0]
    shell = json.loads(child.stdout.readline())['identity']
    owner = dict(old.identity(os.getpid()), token=token)
    receipt = {'version': 1, 'owner': owner, 'phase': 'running', 'request': uuid.uuid4().hex,
               'root': authority.process_pair(shell), 'reaped': [], 'signals': [], 'proof': None}
    save(path, receipt)
    def emit(value):
        print(json.dumps(value), flush=True)
    emit({'owner': owner, 'shell': shell})
    def close(request):
        receipt['phase'], receipt['request'] = 'draining', request
        save(path, receipt)
        if not child.stdin.closed:
            child.stdin.close()
        try:
            code = child.wait(timeout=4)
        except subprocess.TimeoutExpired:
            child.kill();code = child.wait(timeout=4)
        receipt['reaped'].append({'pid': child.pid, 'status': code})
        until = time.monotonic() + 5
        while True:
            try:
                pid, status = os.waitpid(-1, os.WNOHANG)
            except ChildProcessError:
                break
            if pid:
                receipt['reaped'].append({'pid': pid, 'status': os.waitstatus_to_exitcode(status)})
            else:
                assert time.monotonic() < until, 'owned descendant has not exited'
                time.sleep(.02)
        receipt['phase'], receipt['proof'] = 'closed', 'waitpid-echild'
        save(path, receipt)
        emit({'closed': True})
    try:
        for raw in sys.stdin.buffer:
            request = json.loads(raw)
            if request['command'] == 'stop':
                close(request['request']);return
            if request['command'] == 'kill-shell':
                child.kill()
                receipt['signals'].append({'pid': child.pid, 'signal': int(signal.SIGKILL)})
                save(path, receipt);emit({'signaled': True});continue
            child.stdin.write(j.canonical(request) + b'\n');child.stdin.flush()
            assert select.select([child.stdout], [], [], 6)[0]
            emit(json.loads(child.stdout.readline()))
        close(uuid.uuid4().hex)
    finally:
        if child.poll() is None:
            child.kill();child.wait(timeout=4)
        for stream in (child.stdin, child.stdout, child.stderr):
            if stream and not stream.closed:stream.close()


def expected(permit,role):
    return {'permit':permit['id'],'epoch':permit['epoch'],'collection':j.digest(permit['specification']),
            **{key:permit['specification']['members'][role][key] for key in ('plugin_id','profile','artifact_sha256')}}


def shell_worker():
    children = []
    def emit(value):
        print(json.dumps(value), flush=True)
    def receive(child):
        if not select.select([child.stdout], [], [], 5)[0]:
            raise RuntimeError('guard response deadline')
        return json.loads(child.stdout.readline())
    def launch(request, role):
        child = subprocess.Popen([sys.executable, '-B', str(Path(shared.__file__).with_name('repeated_admission_guard.py')),
                                  '--root', request['gate']], stdin=subprocess.PIPE,
                                 stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        children.append(child)
        child.stdin.write(j.canonical({'expected': expected(request['permit'], role), 'role': role,
                                            'instance': uuid.uuid4().hex}) + b'\n')
        child.stdin.flush()
        return child
    emit({'identity': old.identity(os.getpid())})
    try:
        for line in sys.stdin:
            request = json.loads(line)
            command = request['command']
            if command == 'start':
                group = [launch(request, role) for role in j.ROLES]
                emit({'guards': [receive(c) for c in group]})
            elif command == 'replace-one':
                child = launch(request, 'omabit')
                result = receive(child)
                if result['ok']:
                    child.stdin.close()
                emit({'grant': result, 'exit': child.wait(timeout=5)})
            elif command == 'kill-one':
                children[0].kill()
                emit({'exit': children[0].wait(timeout=5)})
            elif command == 'await-refusal':
                live = [c for c in children if c.returncode is None]
                responses = [receive(c) for c in live]
                emit({'responses': responses, 'exits': [c.wait(timeout=5) for c in live]})
            elif command == 'end':
                for child in children:
                    if child.poll() is None:
                        child.stdin.close()
                emit({'exits': [c.wait(timeout=5) for c in children]})
            else:
                raise ValueError('unknown fixture command')
    finally:
        for child in children:
            if child.poll() is None:
                child.kill()
                child.wait(timeout=5)
            for stream in (child.stdin, child.stdout, child.stderr):
                if stream and not stream.closed:
                    stream.close()


if __name__=='__main__':
    if len(sys.argv)==4 and sys.argv[1]=='--guardian':guardian_worker(sys.argv[2],sys.argv[3])
    elif sys.argv[1:]==['--shell']:shell_worker()
    else:unittest.main()
