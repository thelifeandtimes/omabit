#!/usr/bin/env python3
"""Linked authority publication and real private guard continuation tests.

Plugin state, appearance and readiness observations are synthetic here; no native
application activation or compatible generation publication is claimed.
"""
import copy,ctypes,importlib.util,json,os,select,signal,subprocess,sys,time,unittest,uuid
from pathlib import Path
from unittest.mock import patch
ROOT=Path(__file__).resolve().parents[1];sys.path.insert(0,str(ROOT/'native/collections'))
import journal as j
import runtime_authority as authority
import admission as old
import successor_authority as linked
import successor_admission as shared
import session_successor
spec=importlib.util.spec_from_file_location('linked_successor_fixture',Path(__file__).with_name('test-collection-session-successor.py'))
f=importlib.util.module_from_spec(spec);spec.loader.exec_module(f)
h=f.f.f.f.h
save=h.save

class LinkedAuthorityTests(unittest.TestCase):
    def setUp(self):
        f.SuccessorTests.setUp(self);self.advance();self.advance();self.next_authority=None
        self.addCleanup(lambda:self.next_authority.close() if self.next_authority else None)
    def publish(self):
        with self.store.lease() as lease:return linked.step(self.display,self.store,lease,self.observer,self.attempt)
    def finish(self):
        for _ in range(6):
            result=self.publish()
            if result['phase']=='complete':
                if self.next_authority is None:self.next_authority=linked.Authority(result['root'])
                return self.next_authority
        self.fail('successor publication did not finish')
    def record_path(self):return self.store.root/linked.filename(self.attempt)
    def event(self,**values):
        request=self.store.status()['requests'][self.transition]
        return dict(request=self.transition,phase=request['phase'],effect=j.effect(request,request['phase']),**values)
    def ready_transition(self):
        self.finish();self.next_spec=copy.deepcopy(self.spec)
        for role in j.ROLES:self.next_spec['members'][role]['artifact_sha256']=j.digest(['next-generation',role])
        self.record({'type':'register','spec':self.next_spec});self.transition=uuid.uuid4().hex
        self.record({'type':'begin','request':self.transition,'candidate':j.digest(self.next_spec),'expected':j.digest(self.loaded)})
        for role in j.ROLES:
            row=self.observer.members[role]
            self.record(self.event(type='drained',role=role,observation={'epoch':self.loaded['epoch'],'binding':self.loaded['members'][role],
                'absent':False,'closed':True,'retained_state':row['application_sha256'],'pending_ids':row['pending_ids']}))
        self.record(self.event(type='completed',observation={'epoch':self.loaded['epoch'],'absent':True,'helpers_exited':True}))
        self.record(self.event(type='completed',observation={'checkpoint':j.digest(['synthetic-appearance'])}))
        self.guardian,self.owner,self.shell,self.next_receipt=self.launch_guardian()
        request=self.store.status()['requests'][self.transition];self.next_gate=self.root/('gate-'+uuid.uuid4().hex);self.next_gate.mkdir(mode=0o700)
        self.next_permit={'id':uuid.uuid4().hex,'request':self.transition,'epoch':j.effect(request,'enable-candidate'),'specification':self.next_spec,'shell':self.shell}
    def reserve_new(self):
        with self.store.lease() as lease:return self.next_authority.reserve(self.store,lease,self.next_gate,self.next_permit,guardian_receipt=self.next_receipt,adapter=self.observer)
    def issue_new(self):
        self.next_ticket=self.reserve_new();self.new_gate=shared.Gate(self.next_gate);self.gates.append(self.new_gate);self.new_gate.issue(self.next_permit,self.next_ticket)
    def start_new(self):
        rows=self.send(self.guardian,{'command':'start','permit':self.next_permit,'gate':str(self.next_gate)})['guards']
        self.assertEqual(len(rows),3);self.assertTrue(all(x['ok'] for x in rows),rows)
    def adopt_new(self):
        self.issue_new();self.start_new();loaded=h.model.loaded(self.next_spec,self.next_permit['epoch'])
        self.record(self.event(type='completed',observation=loaded));row=self.new_gate.status()['record']
        binding=self.root/'new-binding.json';save(binding,{'loaded':loaded,'helpers':{role:[claim['owner']] for role,claim in row['claims'].items()},
            'shell':self.shell,'supervisor':self.owner,'admission':{'root':str(self.next_gate),'permit':self.next_permit}})
        with self.store.lease() as lease:self.next_authority.bind_loaded(self.store,lease,self.next_ticket,binding,self.next_receipt)
        self.record(self.event(type='completed',observation=h.model.ready(self.next_spec,loaded)));return loaded

    def test_five_publication_steps_preserve_original_objects(self):
        before=self.original();phases=[self.publish()['phase'] for _ in range(5)]
        self.assertEqual(phases,['prepared','seal-published','mirror-published','root-published','complete']);self.finish()
        after=self.snapshot();self.assertEqual(self.publish()['phase'],'complete');self.assertEqual(after,self.snapshot());self.assertEqual(before,self.original())
        self.assertEqual(self.next_authority.descriptor['format'],2);self.assertIsNone(self.next_authority.status()['reservation'])

    def test_original_authority_reader_refuses_successor_without_mutation(self):
        self.finish();before=self.snapshot()
        with self.assertRaises(ValueError):authority.Authority(self.next_authority.root)
        self.assertEqual(before,self.snapshot())

    def test_missing_published_mirror_is_never_recreated(self):
        self.finish();Path(self.next_authority.descriptor['mirror']).unlink();before=self.snapshot()
        with self.assertRaises(OSError):self.publish()
        self.assertEqual(before,self.snapshot())

    def test_lost_reply_at_each_publication_keeps_same_descriptor(self):
        original=linked.provisioning.rename_no_replace
        def lost(*args):original(*args);raise OSError('reply lost')
        for _ in range(4):
            with patch.object(linked.provisioning,'rename_no_replace',lost),self.assertRaises(OSError):self.publish()
        saved=linked.read(self.store,self.attempt);self.finish();self.assertEqual(self.next_authority.descriptor,saved['descriptor'])

    def test_failed_initial_publication_retains_unbound_stages(self):
        with patch.object(linked.provisioning,'rename_no_replace',side_effect=OSError('publication failed')),self.assertRaises(OSError):self.publish()
        self.assertFalse(self.record_path().exists());stages=list(self.runtime.glob('.pending-desktop-authority-*-root-*'));self.assertEqual(len(stages),1)
        self.finish();self.assertTrue(stages[0].is_dir());self.assertNotEqual(stages[0].stat().st_ino,self.next_authority.root.stat().st_ino)

    def test_missing_bound_publication_record_cannot_create_another_authority(self):
        self.publish();self.record_path().unlink();before=self.snapshot()
        with self.assertRaisesRegex(ValueError,'bound publication record is missing'):self.publish()
        self.assertEqual(before,self.snapshot());self.assertFalse(self.record_path().exists())

    def test_missing_bound_root_stage_refuses(self):
        self.publish();record=linked.read(self.store,self.attempt);stage=Path(record['rows']['root']['staging']['path']);stage.rename(stage.with_name(stage.name+'.retained'))
        self.publish();self.publish();before=self.snapshot()
        with self.assertRaises(OSError):self.publish()
        self.assertEqual(before,self.snapshot())

    def test_unknown_publication_format_is_preserved(self):
        self.publish();p=self.record_path();value=json.loads(p.read_text());value['format']=99;p.write_bytes(j.canonical(value)+b'\n');before=self.snapshot()
        with self.assertRaises(ValueError):self.publish()
        self.assertEqual(before,self.snapshot())

    def test_current_cohort_change_blocks_unfinished_publication(self):
        self.publish();self.observer.members['dms']['pending_ids']=[];before=self.snapshot()
        with self.assertRaises(ValueError):self.publish()
        self.assertEqual(before,self.snapshot())

    def test_future_journal_activity_blocks_unfinished_publication(self):
        self.publish();self.record({'type':'begin','request':uuid.uuid4().hex,'candidate':j.digest(self.spec),'expected':j.digest(self.loaded)});before=self.snapshot()
        with self.assertRaises(ValueError):self.publish()
        self.assertEqual(before,self.snapshot())

    def test_three_real_guards_and_bound_replacement_keep_prior_history(self):
        self.ready_transition();prefix=list(self.archived['inspection']['files']);loaded=self.adopt_new()
        state=self.store.status();self.assertEqual(state['committed'],loaded);self.assertIsNone(state['pending']);self.assertEqual(state['requests'][self.transition]['previous'],self.loaded)
        self.assertEqual(self.next_authority.status()['binding']['loaded'],loaded)
        before=self.snapshot();self.assertEqual(self.publish()['phase'],'complete');self.assertEqual(before,self.snapshot())
        self.assertTrue(prefix);self.assertNotEqual(self.next_authority.descriptor['id'],self.archived['inspection']['authority'])
        exits=self.send(self.guardian,{'command':'end'})['exits'];self.assertEqual(exits,[0,0,0])
        self.assertTrue(self.send(self.guardian,{'command':'stop','request':uuid.uuid4().hex})['closed']);self.assertEqual(self.guardian.wait(timeout=5),0)
        if ROOT==Path('/lab/source'):
            save(Path('/lab/linked-admission-result.json'),{'passed':True,'authority':self.next_authority.descriptor,
                'publication':linked.read(self.store,self.attempt),'intent':session_successor.read(self.store,self.attempt),
                'checkpoint':self.archived,'journal':json.loads((self.store.root/'journal.json').read_text()),
                'reservation':self.next_authority.status(),'ticket':self.next_ticket,'gate':self.new_gate.status()['record'],
                'binding':json.loads((self.root/'new-binding.json').read_text()),'guardian':json.loads(self.next_receipt.read_text()),
                'original_loaded':self.loaded,'replacement_loaded':loaded,'guard_exits':exits,'guardian_exit':0,
                'application_observations':'synthetic','new_application_constructions':0})

    def test_live_guards_survive_atomic_journal_appends(self):
        self.ready_transition();self.issue_new();self.start_new()
        for index in range(48):
            spec=copy.deepcopy(self.next_spec)
            for role in j.ROLES:spec['members'][role]['artifact_sha256']=j.digest(['append',index,role])
            self.record({'type':'register','spec':spec})
        time.sleep(.6)
        self.assertEqual(self.send(self.guardian,{'command':'end'})['exits'],[0,0,0])

    def test_four_publication_controller_kills_reuse_original_objects(self):
        control=self.root/'publication-controller.json';save(control,{'store':str(self.store.root),'display':str(self.display),
            'attempt':self.attempt,'members':self.observer.members})
        original=self.original();outcomes=[];saved=None
        for index in range(5):
            q=subprocess.run([sys.executable,'-B',__file__,'--publication-worker',str(control),'kill' if index<4 else ''],capture_output=True,text=True,timeout=8)
            self.assertEqual(q.returncode,-signal.SIGKILL if index<4 else 0,q.stderr);outcomes.append(q.returncode)
            record=linked.read(self.store,self.attempt)
            if saved is None:saved=record
            self.assertEqual(record,saved)
        opened=self.finish();self.assertEqual(original,self.original());self.assertEqual(opened.descriptor,saved['descriptor'])
        if ROOT==Path('/lab/source'):
            save(Path('/lab/linked-publication-process-result.json'),{'passed':True,'outcomes':outcomes,'controller_kills':4,
                'publication':saved,'intent':session_successor.read(self.store,self.attempt),'checkpoint':self.archived,
                'authority':opened.descriptor,'reservation':opened.status(),'original_evidence_unchanged':True,'new_application_constructions':0})

    def test_reservation_requires_transition_at_construction_phase(self):
        self.ready_transition();state=self.store.status();request=state['requests'][self.transition]
        # A different request cannot use the pending transition's epoch.
        self.next_permit['request']=uuid.uuid4().hex;before=self.snapshot()
        with self.assertRaises(ValueError):self.reserve_new()
        self.assertEqual(before,self.snapshot());self.assertIsNone(self.next_authority.status()['reservation'])

    def test_changed_profile_refuses_initial_successor_reservation(self):
        self.ready_transition();self.next_permit['specification']=copy.deepcopy(self.next_spec);self.next_permit['specification']['members']['dms']['profile']='other';before=self.snapshot()
        with self.assertRaises(ValueError):self.reserve_new()
        self.assertEqual(before,self.snapshot())

    def test_changed_closed_cohort_refuses_before_reservation(self):
        self.ready_transition();self.observer.members['dms']['pending_ids']=[];before=self.snapshot()
        with self.assertRaises(ValueError):self.reserve_new()
        self.assertEqual(before,self.snapshot());self.assertIsNone(self.next_authority.status()['reservation'])

    def test_reservation_retry_preserves_exact_bytes_after_apps_start(self):
        self.ready_transition();self.issue_new();self.start_new();before=self.snapshot();self.observer.members['dms']['application_sha256']='0'*64
        self.assertEqual(self.reserve_new(),self.next_ticket);self.assertEqual(before,self.snapshot())
        self.assertEqual(self.send(self.guardian,{'command':'end'})['exits'],[0,0,0])

    def test_old_shared_reader_cannot_issue_successor_ticket(self):
        self.ready_transition();ticket=self.reserve_new();import shared_admission
        gate=shared_admission.Gate(self.next_gate)
        try:
            with gate.transaction():pass  # Original reader initializes its own lock.
            before=self.snapshot()
            with self.assertRaises(ValueError):gate.issue(self.next_permit,ticket)
            self.assertEqual(before,self.snapshot())
        finally:gate.close()

    def test_missing_checkpoint_fences_live_guards_but_allows_end(self):
        self.ready_transition();self.issue_new();self.start_new();self.checkpoint_path().unlink()
        result=self.send(self.guardian,{'command':'await-refusal'});self.assertEqual(result['exits'],[1,1,1]);self.assertTrue(all(not row['ok'] for row in result['responses']))
        self.assertTrue(all(row['state']=='ended' for row in self.new_gate.status()['record']['claims'].values()))

for name in ('cleanup','provision','receive','send','start_claims','record','write_binding','bind','start_display','stop_display','close_guardian','snapshot','ready','step','core_snapshot','completed_cleanup','capture','read_checkpoint','lose_runtime','checkpoint_path','advance','intent_path','claim_path','original'):
    setattr(LinkedAuthorityTests,name,getattr(f.SuccessorTests,name))

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

LinkedAuthorityTests.launch_guardian=launch_guardian

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
        child = subprocess.Popen([sys.executable, '-B', str(Path(shared.__file__).with_name('successor_admission_guard.py')),
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


def publication_worker():
    value=json.loads(Path(sys.argv[2]).read_text());store=j.Store(value['store'])
    try:
        with store.lease() as lease:result=linked.step(value['display'],store,lease,f.Observer(value['members']),value['attempt'])
        if sys.argv[3]=='kill':os.kill(os.getpid(),signal.SIGKILL)
        print(json.dumps(result))
    finally:store.close()


if __name__=='__main__':
    if len(sys.argv)==4 and sys.argv[1]=='--publication-worker':publication_worker()
    elif len(sys.argv)==4 and sys.argv[1]=='--guardian':guardian_worker(sys.argv[2],sys.argv[3])
    elif sys.argv[1:]==['--shell']:shell_worker()
    else:unittest.main()
