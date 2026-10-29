#!/usr/bin/env python3
"""Linked successor intent is exclusive and never becomes live admission."""
import copy,importlib.util,json,os,signal,subprocess
from pathlib import Path
import sys,unittest,uuid
from unittest.mock import patch
ROOT=Path(__file__).resolve().parents[1];sys.path.insert(0,str(ROOT/'native/collections'))
import session_successor as successor
import session_checkpoint as checkpoint
import authority_provisioning as provisioning
import runtime_authority as authority
import journal as j
spec=importlib.util.spec_from_file_location('successor_checkpoint_fixture',Path(__file__).with_name('test-collection-session-checkpoint.py'))
f=importlib.util.module_from_spec(spec);spec.loader.exec_module(f)

class Observer:
    def __init__(self,members):self.members=copy.deepcopy(members);self.calls=0;self.hook=None
    def inspect_successor(self,plan):
        self.calls+=1
        if self.hook:self.hook()
        return copy.deepcopy(self.members)

class SuccessorTests(unittest.TestCase):
    def setUp(self):
        f.CheckpointTests.setUp(self);self.completed_cleanup();self.archived=self.capture();self.lose_runtime()
        receipt=checkpoint.parsed(self.archived,self.archived['cleanup']['object']['path'])
        self.observer=Observer(receipt['verified']['members']);self.attempt=uuid.uuid4().hex
        self.compositor=self.start_display()
    def advance(self,attempt=None,recovery=None):
        with self.store.lease() as lease:return successor.step(self.display,self.store,lease,self.observer,recovery or self.request,attempt or self.attempt)
    def intent_path(self):return self.store.root/successor.filename(self.attempt)
    def claim_path(self):return self.runtime/authority.PROVISION_CLAIM
    def original(self):
        return {str(p.relative_to(self.root)):(p.stat().st_ino,p.read_bytes()) for p in (self.root/'retained-runtime').rglob('*') if p.is_file()}|{str(self.store.root/name):(self.store.root/name).read_bytes() for name in ('journal.json',authority.MIRROR,checkpoint.filename(self.request))}

    def test_prepared_then_claimed_preserves_original_history_and_never_admits(self):
        original=self.original();first=self.advance();self.assertEqual(first['phase'],'prepared');self.assertFalse(self.claim_path().exists())
        value=first['intent'];raw=self.intent_path().read_bytes();inode=self.intent_path().stat().st_ino
        self.assertEqual(value['predecessor'],self.archived['inspection']['authority']);self.assertNotEqual(value['authority'],value['predecessor'])
        second=self.advance();self.assertEqual(second['phase'],'claimed');self.assertFalse(second['activation_authorized']);self.assertEqual(second['intent'],value)
        claim=(self.claim_path().stat().st_ino,self.claim_path().read_bytes());self.assertEqual(self.advance(),second)
        self.assertEqual((self.intent_path().stat().st_ino,self.intent_path().read_bytes()),(inode,raw));self.assertEqual((self.claim_path().stat().st_ino,self.claim_path().read_bytes()),claim)
        self.assertEqual(self.original(),original);self.assertFalse((self.runtime/authority.ANCHOR).exists())
        with self.assertRaises(OSError):authority.Authority(self.runtime/authority.ANCHOR)

    def test_changed_plugin_cohorts_refuse_before_any_file_write(self):
        original=copy.deepcopy(self.observer.members)
        for field,value in [('application_sha256','0'*64),('pending_ids',[]),('credential_refs',[]),('transport_sha256','0'*64),('channel_ids',[]),('closed',False)]:
            with self.subTest(field=field):
                self.observer.members=copy.deepcopy(original);self.observer.members['dms'][field]=value;before=self.snapshot();calls=self.observer.calls
                with self.assertRaises(ValueError):self.advance()
                self.assertEqual(self.snapshot(),before);self.assertEqual(self.observer.calls,calls+1)
        self.observer.members=original

    def test_pending_switch_refuses_before_observer(self):
        self.record({'type':'begin','request':uuid.uuid4().hex,'candidate':j.digest(self.spec),'expected':j.digest(self.loaded)})
        before=self.snapshot()
        with self.assertRaises(ValueError):self.advance()
        self.assertEqual(before,self.snapshot());self.assertEqual(self.observer.calls,0)

    def test_second_attempt_cannot_take_predecessor(self):
        self.advance();before=self.snapshot();calls=self.observer.calls
        with self.assertRaises(ValueError):self.advance(attempt=uuid.uuid4().hex)
        self.assertEqual(before,self.snapshot());self.assertEqual(calls,self.observer.calls)

    def test_claimed_runtime_blocks_fresh_authority_from_another_coordinator(self):
        self.advance();self.advance();other=j.Store(self.root/'foreign-coordinator',create=True)
        try:
            with other.lease() as lease:
                with self.assertRaises(ValueError):authority.Authority.provision(self.display,other,lease)
                with self.assertRaises(ValueError):provisioning.step(self.display,other,lease,uuid.uuid4().hex)
            self.assertFalse((self.runtime/authority.ANCHOR).exists())
        finally:other.close()

    def test_failed_intent_publication_retains_unbound_stage(self):
        original=self.original()
        with patch.object(successor.provisioning,'rename_no_replace',side_effect=OSError('intent publication failed')),self.assertRaises(OSError):self.advance()
        self.assertFalse(self.intent_path().exists());self.assertFalse(self.claim_path().exists())
        abandoned=list(self.runtime.glob('.pending-desktop-successor-*'));self.assertEqual(len(abandoned),1);raw=abandoned[0].read_bytes()
        self.advance();self.advance();self.assertEqual(abandoned[0].read_bytes(),raw);self.assertEqual(original,self.original())

    def test_lost_intent_publication_reply_reuses_original_identity(self):
        original=successor.provisioning.rename_no_replace
        def lost(*args):original(*args);raise OSError('lost intent reply')
        with patch.object(successor.provisioning,'rename_no_replace',lost),self.assertRaises(OSError):self.advance()
        value=successor.read(self.store,self.attempt);raw=self.intent_path().read_bytes();inode=self.intent_path().stat().st_ino
        self.assertEqual(self.advance()['intent'],value);self.assertEqual((self.intent_path().stat().st_ino,self.intent_path().read_bytes()),(inode,raw))

    def test_failed_claim_publication_leaves_original_bound_stage(self):
        self.advance();before=self.snapshot()
        with patch.object(successor.provisioning,'rename_no_replace',side_effect=OSError('claim publication failed')),self.assertRaises(OSError):self.advance()
        self.assertEqual(before,self.snapshot());self.assertEqual(self.advance()['phase'],'claimed')

    def test_lost_claim_publication_reply_keeps_exact_inode(self):
        self.advance();original=successor.provisioning.rename_no_replace
        def lost(*args):original(*args);raise OSError('lost claim reply')
        with patch.object(successor.provisioning,'rename_no_replace',lost),self.assertRaises(OSError):self.advance()
        before=(self.claim_path().stat().st_ino,self.claim_path().read_bytes());self.assertEqual(self.advance()['phase'],'claimed');self.assertEqual((self.claim_path().stat().st_ino,self.claim_path().read_bytes()),before)

    def test_missing_bound_claim_is_not_recreated(self):
        self.advance();self.advance();self.claim_path().unlink();before=self.snapshot()
        with self.assertRaises(OSError):self.advance()
        self.assertEqual(before,self.snapshot())

    def test_missing_bound_stage_is_not_recreated(self):
        value=self.advance()['intent'];Path(value['claim']['staging']['path']).unlink();before=self.snapshot()
        with self.assertRaises(OSError):self.advance()
        self.assertEqual(before,self.snapshot());self.assertFalse(self.claim_path().exists())

    def test_replaced_claim_with_same_bytes_refuses(self):
        self.advance();self.advance();raw=self.claim_path().read_bytes();self.claim_path().rename(self.runtime/'retained-claim');self.claim_path().write_bytes(raw);self.claim_path().chmod(0o600);before=self.snapshot()
        with self.assertRaises(ValueError):self.advance()
        self.assertEqual(before,self.snapshot())

    def test_conflicting_claim_is_never_overwritten(self):
        self.advance();self.claim_path().write_text('{}\n');self.claim_path().chmod(0o600);before=self.snapshot()
        with self.assertRaises(ValueError):self.advance()
        self.assertEqual(before,self.snapshot())

    def test_checkpoint_replaced_with_same_bytes_refuses(self):
        self.advance();path=self.checkpoint_path();raw=path.read_bytes();path.rename(path.with_suffix('.retained'));path.write_bytes(raw);path.chmod(0o600);before=self.snapshot()
        with self.assertRaises(ValueError):self.advance()
        self.assertEqual(before,self.snapshot())

    def test_unknown_intent_format_preserved(self):
        self.advance();path=self.intent_path();value=json.loads(path.read_text());value['format']=99;path.write_bytes(j.canonical(value)+b'\n');before=self.snapshot()
        with self.assertRaises(ValueError):self.advance()
        self.assertEqual(before,self.snapshot())

    def test_display_replacement_does_not_rebind_prepared_intent(self):
        self.advance();self.stop_display();self.display.unlink();self.compositor=self.start_display();before=self.snapshot()
        with self.assertRaises(ValueError):self.advance()
        self.assertEqual(before,self.snapshot())

    def test_changed_cohort_on_retry_preserves_pending_claim(self):
        self.advance();self.observer.members['dms']['pending_ids']=[];before=self.snapshot()
        with self.assertRaises(ValueError):self.advance()
        self.assertEqual(before,self.snapshot());self.assertFalse(self.claim_path().exists())

    def test_original_mirror_change_refuses_before_observation(self):
        self.advance();path=self.store.root/authority.MIRROR;body=json.loads(path.read_text());body['id']=uuid.uuid4().hex;path.write_bytes(j.canonical(body)+b'\n');before=self.snapshot();calls=self.observer.calls
        with self.assertRaises(ValueError):self.advance()
        self.assertEqual(before,self.snapshot());self.assertEqual(calls,self.observer.calls)

    def test_context_change_during_observation_refuses_before_staging(self):
        self.observer.hook=self.stop_display;before=self.snapshot()
        with self.assertRaises(ValueError):self.advance()
        self.assertEqual(before,self.snapshot());self.assertEqual(self.observer.calls,1)

    def test_wrong_recovery_retry_refuses_unchanged(self):
        self.advance();before=self.snapshot()
        with self.assertRaises(ValueError):self.advance(recovery=uuid.uuid4().hex)
        self.assertEqual(before,self.snapshot())

    def test_three_real_controller_kills_preserve_one_intent_and_claim(self):
        control=self.root/'successor-controller.json'
        control.write_bytes(j.canonical({'display':str(self.display),'store':str(self.store.root),'recovery':self.request,
                                        'attempt':self.attempt,'members':self.observer.members})+b'\n');control.chmod(0o600)
        original=self.original();outcomes=[]
        for fault in ('before-intent','after-intent','after-claim',''):
            result=subprocess.run([sys.executable,'-B',__file__,'--successor-worker',str(control),fault],capture_output=True,text=True,timeout=8)
            self.assertEqual(result.returncode,-signal.SIGKILL if fault else 0,result.stderr)
            outcomes.append({'fault':fault,'exit':result.returncode})
            if fault=='before-intent':
                self.assertFalse(self.intent_path().exists());abandoned=list(self.runtime.glob('.pending-desktop-successor-*'))
                self.assertEqual(len(abandoned),1);abandoned_path=abandoned[0];abandoned_raw=abandoned_path.read_bytes()
            elif fault=='after-intent':
                self.assertFalse(self.claim_path().exists());intent=(self.intent_path().stat().st_ino,self.intent_path().read_bytes())
            elif fault=='after-claim':claim=(self.claim_path().stat().st_ino,self.claim_path().read_bytes())
        self.assertEqual((self.intent_path().stat().st_ino,self.intent_path().read_bytes()),intent)
        self.assertEqual((self.claim_path().stat().st_ino,self.claim_path().read_bytes()),claim)
        self.assertEqual(abandoned_path.read_bytes(),abandoned_raw);self.assertEqual(self.original(),original)
        self.assertFalse((self.runtime/authority.ANCHOR).exists())
        if ROOT==Path('/lab/source'):
            record=successor.read(self.store,self.attempt)
            evidence={'passed':True,'controller_kills':3,'outcomes':outcomes,'intent':record,
                      'intent_inode':intent[0],'claim_inode':claim[0],'claim':json.loads(claim[1]),
                      'abandoned_claim':json.loads(abandoned_raw),'checkpoint':self.archived,
                      'journal':json.loads((self.store.root/'journal.json').read_text()),
                      'original_mirror':json.loads((self.store.root/authority.MIRROR).read_text()),
                      'original_evidence_unchanged':True,'activation_authorized':False,'cohort_adapter':'synthetic closed cohort'}
            Path('/lab/successor-process-result.json').write_bytes(j.canonical(evidence)+b'\n')

    def test_expired_lease_refuses_before_observation(self):
        with self.store.lease() as lease:pass
        before=self.snapshot()
        with self.assertRaises(ValueError):successor.step(self.display,self.store,lease,self.observer,self.request,self.attempt)
        self.assertEqual(before,self.snapshot());self.assertEqual(self.observer.calls,0)

for name in ('cleanup','provision','receive','launch_guardian','send','start_claims','record','write_binding','bind','start_display','stop_display','close_guardian','snapshot','ready','step','core_snapshot','completed_cleanup','capture','read_checkpoint','lose_runtime','checkpoint_path'):
    setattr(SuccessorTests,name,getattr(f.CheckpointTests,name))
def worker():
    control=json.loads(Path(sys.argv[2]).read_text());fault=sys.argv[3];store=j.Store(control['store'])
    original=successor.save
    if fault in ('before-intent','after-intent'):
        def save(*args):
            if fault=='after-intent':original(*args)
            os.kill(os.getpid(),signal.SIGKILL)
        successor.save=save
    if fault=='after-claim':
        rename=successor.provisioning.rename_no_replace
        def publish(*args):
            rename(*args);os.kill(os.getpid(),signal.SIGKILL)
        successor.provisioning.rename_no_replace=publish
    try:
        with store.lease() as lease:
            result=successor.step(control['display'],store,lease,Observer(control['members']),control['recovery'],control['attempt'])
        print(json.dumps({'phase':result['phase'],'activation_authorized':result['activation_authorized']}))
    finally:store.close()

if __name__=='__main__':
    if len(sys.argv)>1 and sys.argv[1]=='--successor-worker':worker()
    else:unittest.main()
