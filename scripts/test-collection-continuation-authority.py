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
import continuation_authority as linked
import successor_authority as previous
import continuation_admission as shared
import session_continuation as session_successor
spec=importlib.util.spec_from_file_location('linked_successor_fixture',Path(__file__).with_name('test-collection-session-continuation.py'))
f=importlib.util.module_from_spec(spec);spec.loader.exec_module(f)
h=f.f.f.f.f.h
save=h.save

class ContinuedAuthorityTests(unittest.TestCase):
    def setUp(self):
        f.ContinuationTests.setUp(self);self.continue_step();self.continue_step();self.continued_authority=None
        self.addCleanup(lambda:self.continued_authority.close() if self.continued_authority else None)
    def preserved(self):
        value=f.ContinuationTests.preserved(self);intent=session_successor.read(self.store,self.continuation_id)
        generated={linked.filename(self.continuation_id),'runtime-continuation-'+intent['authority']+'.json'}
        stages=('.pending-desktop-activation-'+self.continuation_id+'-','.pending-desktop-activation-record-'+self.continuation_id+'-')
        value={k:v for k,v in value.items() if Path(k).name not in generated and not Path(k).name.startswith(stages)}
        for path in (self.continuation_path(),self.continuation_claim(),self.root/'new-binding.json'):
            value[str(path)]=(path.stat().st_ino,path.read_bytes())
        return value
    def publish_continuation(self):
        with self.store.lease() as lease:return linked.step(self.display,self.store,lease,self.continuation_observer,self.continuation_id)
    def finish_continuation(self):
        for _ in range(6):
            result=self.publish_continuation()
            if result['phase']=='complete':
                if self.continued_authority is None:self.continued_authority=linked.Authority(result['root'])
                return self.continued_authority
        self.fail('successor publication did not finish')
    def activation_record_path(self):return self.store.root/linked.filename(self.continuation_id)
    def continuation_event(self,**values):
        request=self.store.status()['requests'][self.continued_transition]
        return dict(request=self.continued_transition,phase=request['phase'],effect=j.effect(request,request['phase']),**values)
    def ready_continuation(self):
        self.finish_continuation();self.continued_spec=copy.deepcopy(self.next_spec)
        for role in j.ROLES:self.continued_spec['members'][role]['artifact_sha256']=j.digest(['continued-generation',role])
        self.assertNotEqual(j.digest(self.continued_spec),j.digest(self.next_spec))
        self.record({'type':'register','spec':self.continued_spec});self.continued_transition=uuid.uuid4().hex
        self.record({'type':'begin','request':self.continued_transition,'candidate':j.digest(self.continued_spec),'expected':j.digest(self.replacement)})
        for role in j.ROLES:
            row=self.continuation_observer.members[role]
            self.record(self.continuation_event(type='drained',role=role,observation={'epoch':self.replacement['epoch'],'binding':self.replacement['members'][role],
                'absent':False,'closed':True,'retained_state':row['application_sha256'],'pending_ids':row['pending_ids']}))
        self.record(self.continuation_event(type='completed',observation={'epoch':self.replacement['epoch'],'absent':True,'helpers_exited':True}))
        self.record(self.continuation_event(type='completed',observation={'checkpoint':j.digest(['synthetic-appearance'])}))
        self.guardian,self.owner,self.shell,self.continued_receipt=self.launch_guardian()
        request=self.store.status()['requests'][self.continued_transition];self.continued_gate_root=self.root/('gate-'+uuid.uuid4().hex);self.continued_gate_root.mkdir(mode=0o700)
        self.continued_permit={'id':uuid.uuid4().hex,'request':self.continued_transition,'epoch':j.effect(request,'enable-candidate'),'specification':self.continued_spec,'shell':self.shell}
    def reserve_continued(self):
        with self.store.lease() as lease:return self.continued_authority.reserve(self.store,lease,self.continued_gate_root,self.continued_permit,guardian_receipt=self.continued_receipt,adapter=self.continuation_observer)
    def issue_continued(self):
        self.continued_ticket=self.reserve_continued();self.continued_gate=shared.Gate(self.continued_gate_root);self.gates.append(self.continued_gate);self.continued_gate.issue(self.continued_permit,self.continued_ticket)
    def start_continued(self):
        rows=self.send(self.guardian,{'command':'start','permit':self.continued_permit,'gate':str(self.continued_gate_root)})['guards']
        self.assertEqual(len(rows),3);self.assertTrue(all(x['ok'] for x in rows),rows)
    def adopt_continued(self):
        self.issue_continued();self.start_continued();loaded=h.model.loaded(self.continued_spec,self.continued_permit['epoch'])
        self.record(self.continuation_event(type='completed',observation=loaded));row=self.continued_gate.status()['record']
        binding=self.root/'continued-binding.json';save(binding,{'loaded':loaded,'helpers':{role:[claim['owner']] for role,claim in row['claims'].items()},
            'shell':self.shell,'supervisor':self.owner,'admission':{'root':str(self.continued_gate_root),'permit':self.continued_permit}})
        with self.store.lease() as lease:self.continued_authority.bind_loaded(self.store,lease,self.continued_ticket,binding,self.continued_receipt)
        self.record(self.continuation_event(type='completed',observation=h.model.ready(self.continued_spec,loaded)));return loaded

    def test_five_publication_steps_preserve_original_objects(self):
        before=self.preserved();phases=[self.publish_continuation()['phase'] for _ in range(5)]
        self.assertEqual(phases,['prepared','seal-published','mirror-published','root-published','complete']);self.finish_continuation()
        after=self.snapshot();self.assertEqual(self.publish_continuation()['phase'],'complete');self.assertEqual(after,self.snapshot());self.assertEqual(before,self.preserved())
        self.assertEqual(self.continued_authority.descriptor['format'],3);self.assertIsNone(self.continued_authority.status()['reservation'])

    def test_original_authority_reader_refuses_successor_without_mutation(self):
        self.finish_continuation();before=self.snapshot()
        with self.assertRaises(ValueError):authority.Authority(self.continued_authority.root)
        self.assertEqual(before,self.snapshot())

    def test_missing_published_mirror_is_never_recreated(self):
        self.finish_continuation();Path(self.continued_authority.descriptor['mirror']).unlink();before=self.snapshot()
        with self.assertRaises(OSError):self.publish_continuation()
        self.assertEqual(before,self.snapshot())

    def test_lost_reply_at_each_publication_keeps_same_descriptor(self):
        original=linked.provisioning.rename_no_replace
        def lost(*args):original(*args);raise OSError('reply lost')
        for _ in range(4):
            with patch.object(linked.provisioning,'rename_no_replace',lost),self.assertRaises(OSError):self.publish_continuation()
        saved=linked.read(self.store,self.continuation_id);self.finish_continuation();self.assertEqual(self.continued_authority.descriptor,saved['descriptor'])

    def test_failed_initial_publication_retains_unbound_stages(self):
        with patch.object(linked.provisioning,'rename_no_replace',side_effect=OSError('publication failed')),self.assertRaises(OSError):self.publish_continuation()
        self.assertFalse(self.activation_record_path().exists());stages=list(self.runtime.glob('.pending-desktop-activation-*-root-*'));self.assertEqual(len(stages),1)
        self.finish_continuation();self.assertTrue(stages[0].is_dir());self.assertNotEqual(stages[0].stat().st_ino,self.continued_authority.root.stat().st_ino)

    def test_missing_bound_publication_record_cannot_create_another_authority(self):
        self.publish_continuation();self.activation_record_path().unlink();before=self.snapshot()
        with self.assertRaisesRegex(ValueError,'bound publication record is missing'):self.publish_continuation()
        self.assertEqual(before,self.snapshot());self.assertFalse(self.activation_record_path().exists())

    def test_missing_bound_root_stage_refuses(self):
        self.publish_continuation();record=linked.read(self.store,self.continuation_id);stage=Path(record['rows']['root']['staging']['path']);stage.rename(stage.with_name(stage.name+'.retained'))
        self.publish_continuation();self.publish_continuation();before=self.snapshot()
        with self.assertRaises(OSError):self.publish_continuation()
        self.assertEqual(before,self.snapshot())

    def test_unknown_publication_format_is_preserved(self):
        self.publish_continuation();p=self.activation_record_path();value=json.loads(p.read_text());value['format']=99;p.write_bytes(j.canonical(value)+b'\n');before=self.snapshot()
        with self.assertRaises(ValueError):self.publish_continuation()
        self.assertEqual(before,self.snapshot())

    def test_current_cohort_change_blocks_unfinished_publication(self):
        self.publish_continuation();self.continuation_observer.members['dms']['pending_ids']=[];before=self.snapshot()
        with self.assertRaises(ValueError):self.publish_continuation()
        self.assertEqual(before,self.snapshot())

    def test_future_journal_activity_blocks_unfinished_publication(self):
        self.publish_continuation();self.record({'type':'begin','request':uuid.uuid4().hex,'candidate':j.digest(self.next_spec),'expected':j.digest(self.replacement)});before=self.snapshot()
        with self.assertRaises(ValueError):self.publish_continuation()
        self.assertEqual(before,self.snapshot())

    def test_three_real_guards_and_bound_replacement_keep_prior_history(self):
        self.ready_continuation();prefix=list(self.second_archived['inspection']['files']);loaded=self.adopt_continued()
        state=self.store.status();self.assertEqual(state['committed'],loaded);self.assertIsNone(state['pending']);self.assertEqual(state['requests'][self.continued_transition]['previous'],self.replacement)
        self.assertEqual(self.continued_authority.status()['binding']['loaded'],loaded)
        before=self.snapshot();self.assertEqual(self.publish_continuation()['phase'],'complete');self.assertEqual(before,self.snapshot())
        self.assertTrue(prefix);self.assertNotEqual(self.continued_authority.descriptor['id'],self.second_archived['inspection']['authority'])
        exits=self.send(self.guardian,{'command':'end'})['exits'];self.assertEqual(exits,[0,0,0])
        self.assertTrue(self.send(self.guardian,{'command':'stop','request':uuid.uuid4().hex})['closed']);self.assertEqual(self.guardian.wait(timeout=5),0)
        if ROOT==Path('/lab/source'):
            save(Path('/lab/continued-admission-result.json'),{'passed':True,'authority':self.continued_authority.descriptor,
                'publication':linked.read(self.store,self.continuation_id),'intent':session_successor.read(self.store,self.continuation_id),
                'checkpoint':self.second_archived,'journal':json.loads((self.store.root/'journal.json').read_text()),
                'reservation':self.continued_authority.status(),'ticket':self.continued_ticket,'gate':self.continued_gate.status()['record'],
                'binding':json.loads((self.root/'continued-binding.json').read_text()),'guardian':json.loads(self.continued_receipt.read_text()),
                'original_loaded':self.replacement,'replacement_loaded':loaded,'guard_exits':exits,'guardian_exit':0,
                'application_observations':'synthetic','new_application_constructions':0})

    def test_live_guards_survive_atomic_journal_appends(self):
        self.ready_continuation();self.issue_continued();self.start_continued()
        for index in range(48):
            spec=copy.deepcopy(self.continued_spec)
            for role in j.ROLES:spec['members'][role]['artifact_sha256']=j.digest(['append',index,role])
            self.record({'type':'register','spec':spec})
        time.sleep(.6)
        self.assertEqual(self.send(self.guardian,{'command':'end'})['exits'],[0,0,0])

    def test_four_publication_controller_kills_reuse_original_objects(self):
        control=self.root/'publication-controller.json';save(control,{'store':str(self.store.root),'display':str(self.display),
            'attempt':self.continuation_id,'members':self.continuation_observer.members})
        original=self.preserved();outcomes=[];saved=None
        for index in range(5):
            q=subprocess.run([sys.executable,'-B',__file__,'--publication-worker',str(control),'kill' if index<4 else ''],capture_output=True,text=True,timeout=8)
            self.assertEqual(q.returncode,-signal.SIGKILL if index<4 else 0,q.stderr);outcomes.append(q.returncode)
            record=linked.read(self.store,self.continuation_id)
            if saved is None:saved=record
            self.assertEqual(record,saved)
        opened=self.finish_continuation();self.assertEqual(original,self.preserved());self.assertEqual(opened.descriptor,saved['descriptor'])
        if ROOT==Path('/lab/source'):
            save(Path('/lab/continued-publication-process-result.json'),{'passed':True,'outcomes':outcomes,'controller_kills':4,
                'publication':saved,'intent':session_successor.read(self.store,self.continuation_id),'checkpoint':self.second_archived,
                'authority':opened.descriptor,'reservation':opened.status(),'original_evidence_unchanged':True,'new_application_constructions':0})

    def test_reservation_requires_transition_at_construction_phase(self):
        self.ready_continuation();state=self.store.status();request=state['requests'][self.continued_transition]
        # A different request cannot use the pending transition's epoch.
        self.continued_permit['request']=uuid.uuid4().hex;before=self.snapshot()
        with self.assertRaises(ValueError):self.reserve_continued()
        self.assertEqual(before,self.snapshot());self.assertIsNone(self.continued_authority.status()['reservation'])

    def test_changed_profile_refuses_initial_successor_reservation(self):
        self.ready_continuation();self.continued_permit['specification']=copy.deepcopy(self.continued_spec);self.continued_permit['specification']['members']['dms']['profile']='other';before=self.snapshot()
        with self.assertRaises(ValueError):self.reserve_continued()
        self.assertEqual(before,self.snapshot())

    def test_changed_closed_cohort_refuses_before_reservation(self):
        self.ready_continuation();self.continuation_observer.members['dms']['pending_ids']=[];before=self.snapshot()
        with self.assertRaises(ValueError):self.reserve_continued()
        self.assertEqual(before,self.snapshot());self.assertIsNone(self.continued_authority.status()['reservation'])

    def test_reservation_retry_preserves_exact_bytes_after_apps_start(self):
        self.ready_continuation();self.issue_continued();self.start_continued();before=self.snapshot();self.continuation_observer.members['dms']['application_sha256']='0'*64
        self.assertEqual(self.reserve_continued(),self.continued_ticket);self.assertEqual(before,self.snapshot())
        self.assertEqual(self.send(self.guardian,{'command':'end'})['exits'],[0,0,0])

    def test_old_shared_reader_cannot_issue_successor_ticket(self):
        self.ready_continuation();ticket=self.reserve_continued();import shared_admission
        gate=shared_admission.Gate(self.continued_gate_root)
        try:
            with gate.transaction():pass  # Original reader initializes its own lock.
            before=self.snapshot()
            with self.assertRaises(ValueError):gate.issue(self.continued_permit,ticket)
            self.assertEqual(before,self.snapshot())
        finally:gate.close()

    def test_missing_checkpoint_fences_live_guards_but_allows_end(self):
        self.ready_continuation();self.issue_continued();self.start_continued();self.linked_checkpoint_path().unlink()
        result=self.send(self.guardian,{'command':'await-refusal'});self.assertEqual(result['exits'],[1,1,1]);self.assertTrue(all(not row['ok'] for row in result['responses']))
        self.assertTrue(all(row['state']=='ended' for row in self.continued_gate.status()['record']['claims'].values()))

    def test_prior_authority2_reader_refuses3_without_mutation(self):
        self.finish_continuation();before=self.snapshot()
        with self.assertRaises(ValueError):previous.Authority(self.continued_authority.root)
        self.assertEqual(before,self.snapshot())
    def test_missing_grandparent_fences_guards_without_reconstructing_ancestry(self):
        self.ready_continuation();self.issue_continued();self.start_continued();self.ancestor_checkpoint.unlink()
        result=self.send(self.guardian,{'command':'await-refusal'});self.assertEqual(result['exits'],[1,1,1])
        self.assertTrue(all(not row['ok'] for row in result['responses']))
        self.assertFalse(self.ancestor_checkpoint.exists())

for name in dir(f.ContinuationTests):
    value=getattr(f.ContinuationTests,name)
    if callable(value) and not name.startswith(('test','_')) and not hasattr(ContinuedAuthorityTests,name):setattr(ContinuedAuthorityTests,name,value)

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

ContinuedAuthorityTests.launch_guardian=launch_guardian

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
        child = subprocess.Popen([sys.executable, '-B', str(Path(shared.__file__).with_name('continuation_admission_guard.py')),
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
