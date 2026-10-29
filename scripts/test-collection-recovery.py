#!/usr/bin/env python3
"""Recovery model/driver tests. Fake observations never certify real processes."""
import copy
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import signal
import subprocess
import sys
import time
import unittest
import uuid
from unittest.mock import patch

ROOT=Path(__file__).resolve().parents[1]
sys.path.insert(0,str(ROOT/'native/collections'))
import journal as j
import coordinator as c
spec=importlib.util.spec_from_file_location('journal_tests',ROOT/'scripts/test-collection-journal.py')
m=importlib.util.module_from_spec(spec);spec.loader.exec_module(m)


def plan_for(request):
    return {'loaded':copy.deepcopy(request['previous']),'fences':copy.deepcopy(request['uncertain']),
      'guardian':{'owner':{'token':'a'*32,'pid':30,'start':'12345','boot':'12345678-1234-1234-1234-123456789abc','namespace':'pid:[1234]'},'root':{'pid':31,'start':'12346'}},
      'closure_request':'b'*32,'runtime_binding_sha256':j.digest(['original-runtime-binding',request['previous']]),
      'members':{r:{'binding':copy.deepcopy(request['previous']['members'][r]),'application_sha256':j.digest([r,'application-state']),
        'pending_ids':(['c'*32] if r!='omabit' else []),'credential_refs':([j.digest([r,'session'])] if r!='omabit' else []),
        'transport_sha256':j.digest([r,'transport-before']),'channel_ids':(['d'*32] if r!='omabit' else [])} for r in j.ROLES}}


def observed(plan):
    return {'plan_sha256':j.digest(plan),'guardian':copy.deepcopy(plan['guardian']),'closure_request':plan['closure_request'],
      'closure_receipt_sha256':j.digest(['original-guardian-closure',plan['guardian']]),'proof':'waitpid-echild',
      'members':{r:{**copy.deepcopy(v),'transport_sha256':j.digest([r,'transport-after']),'closed':True} for r,v in plan['members'].items()}}


class RecoveryTests(unittest.TestCase):
    def setUp(self):
        self.case=m.JournalTests();self.case.setUp();self.addCleanup(self.case.doCleanups)
        self.store=self.case.store;self.rid=uuid.uuid4().hex;self.calls=[]
        self.case.write(self.case.begin)
        # Preserve a real ordinary partial-drain receipt alongside the fence.
        self.case.write(self.case.next_events()[0])
        self.problem=self.case.envelope('problem',reason='reader-owner-lost');self.case.write(self.problem)
    def prepare(self):
        return self.case.envelope('recovery-prepared',recovery=self.rid,plan=plan_for(self.case.pending()))
    def verify(self,plan):
        return self.case.envelope('recovery-verified',recovery=self.rid,observation=observed(plan))
    def data(self):return (self.case.root/'journal.json').read_bytes()
    def driver(self,adapter=None,rid=None):
        self.store=self.case.store
        outer=self
        class Adapter:
            def prepare_recovery(self,request,specification,recovery_id):
                outer.calls.append('prepare');return plan_for(request)
            def recover(self,request,plan):
                outer.calls.append('recover');return observed(plan)
        with self.store.lease() as lease:return c.recover_step(self.store,lease,adapter or Adapter(),self.case.request,rid or self.rid)
    def test_reopen_preserves_old_receipts_and_pending_app_intent(self):
        old=json.loads(self.data())['events'];p=self.prepare();self.case.write(p);self.case.reopen();self.store=self.case.store
        self.assertEqual(self.case.pending()['uncertain'],p['plan']['fences'])
        v=self.verify(p['plan']);self.case.write(v);self.case.reopen();self.store=self.case.store
        pending=self.case.pending();self.assertEqual(pending['phase'],'disable-previous');self.assertEqual(pending['uncertain'],[])
        self.assertEqual(json.loads(self.data())['events'][:len(old)],old)
        self.assertEqual(pending['receipts'][self.problem['effect']+'/problem/reader-owner-lost'],self.problem)
        self.assertEqual(pending['recovery']['verified']['members']['dms']['pending_ids'],['c'*32])
        self.assertEqual(pending['recovery']['verified']['members']['theme']['pending_ids'],['c'*32])
        self.case.finish();self.assertEqual(self.case.pending()['outcome'],'committed')
    def test_verification_without_prepared_intent_refuses_unchanged(self):
        before=self.data()
        with self.assertRaises(j.Conflict):self.case.write(self.verify(self.prepare()['plan']))
        self.assertEqual(self.data(),before)
    def test_wrong_activation_or_invalid_prepared_cohort_refuses_unchanged(self):
        good=self.prepare();before=self.data()
        for path,value in [(('loaded','epoch'),'f'*32),(('loaded','members','theme','instance'),'f'*32),(('fences',),[]),
            (('guardian','owner','pid'),True),(('guardian','owner','namespace'),'host'),
            (('members','dms','pending_ids'),['c'*32,'c'*32]),(('members','theme','credential_refs'),['raw-cookie']),
            (('members','theme','channel_ids'),['f'*32,'a'*32]),(('members','omabit','binding'),None)]:
            with self.subTest(path=path):
                event=copy.deepcopy(good);target=event['plan']
                for key in path[:-1]:target=target[key]
                target[path[-1]]=value
                with self.assertRaises(j.Conflict):self.case.write(event)
                self.assertEqual(self.data(),before)
        event=copy.deepcopy(good);event['plan']['cookie']='never-persist'
        with self.assertRaises(j.Conflict):self.case.write(event)
        self.assertEqual(self.data(),before)
    def test_changed_state_credentials_channel_or_guardian_refuses_unchanged(self):
        p=self.prepare();self.case.write(p);good=self.verify(p['plan']);before=self.data()
        changes=[(('plan_sha256',),'0'*64),(('guardian','owner','token'),'f'*32),(('closure_request',),'f'*32),
          (('proof',),'new-empty-shell'),(('members','theme','application_sha256'),'0'*64),
          (('members','dms','pending_ids'),[]),(('members','theme','credential_refs'),[]),
          (('members','theme','channel_ids'),[]),(('members','omabit','closed'),False),
          (('members','dms','binding','instance'),'f'*32)]
        for path,value in changes:
            with self.subTest(path=path):
                event=copy.deepcopy(good);target=event['observation']
                for key in path[:-1]:target=target[key]
                target[path[-1]]=value
                with self.assertRaises(j.Conflict):self.case.write(event)
                self.assertEqual(self.data(),before)
    def test_completed_exact_replay_is_read_only_after_phase_and_commit(self):
        p=self.prepare();self.case.write(p);v=self.verify(p['plan']);self.case.write(v);self.case.finish();before=self.data()
        self.case.write(p);self.case.write(v);self.driver();self.assertEqual(self.calls,[]);self.assertEqual(self.data(),before)
        bad=copy.deepcopy(v);bad['observation']['closure_receipt_sha256']='f'*64
        with self.assertRaises(j.Conflict):self.case.write(bad)
        with self.assertRaises(j.Conflict):self.driver(rid='f'*32)
        self.assertEqual(self.data(),before)
    def test_ordinary_step_stays_fenced_after_recovery_prepared(self):
        self.driver();before=self.data()
        class Adapter:
            def perform(self,*args):raise AssertionError('ordinary effects must stay fenced')
        with self.store.lease() as lease:
            with self.assertRaises(j.Conflict):c.step(self.store,lease,Adapter())
        self.assertEqual(self.data(),before)
    def test_new_fence_invalidates_prepared_plan_before_adapter_effect(self):
        self.driver();self.case.write(self.case.envelope('problem',reason='helper-descendants-unconfirmed'));before=self.data()
        with self.assertRaises(j.Conflict):self.driver()
        self.assertEqual(self.calls,['prepare']);self.assertEqual(self.data(),before)
    def test_different_recovery_id_cannot_replace_prepared_plan(self):
        self.driver();before=self.data()
        with self.assertRaises(j.Conflict):self.driver(rid='e'*32)
        self.assertEqual(self.calls,['prepare']);self.assertEqual(self.data(),before)
    def test_closed_lease_refuses_before_any_adapter_call(self):
        with self.store.lease() as lease:pass
        with self.assertRaises(j.Conflict):c.recover_step(self.store,lease,object(),self.case.request,self.rid)
    def test_driver_prepares_before_effect_and_copies_context(self):
        outer=self
        class Adapter:
            def prepare_recovery(self,request,specification,recovery_id):
                plan=plan_for(request);request['previous']['epoch']='f'*32;specification['name']='mutated';return plan
            def recover(self,request,plan):
                durable=outer.store.status()['requests'][outer.case.request]
                outer.assertEqual(durable['recovery']['plan'],plan)
                result=observed(plan);plan['members']['dms']['pending_ids']=[];request['uncertain']=[];return result
        self.driver(Adapter());self.assertTrue(self.case.pending()['uncertain']);self.driver(Adapter())
        self.assertEqual(self.case.pending()['recovery']['plan']['members']['dms']['pending_ids'],['c'*32])
    def test_busy_exception_and_invalid_receipt_preserve_prepared_intent(self):
        for prepared in (False,True):
            if prepared:self.driver()
            for error in (c.Busy(),RuntimeError('lost observation')):
                before=self.data()
                class Adapter:
                    def prepare_recovery(self,*args):raise error
                    def recover(self,*args):raise error
                if isinstance(error,c.Busy):self.driver(Adapter())
                else:
                    with self.assertRaises(RuntimeError):self.driver(Adapter())
                self.assertEqual(self.data(),before)
        class Invalid:
            def recover(self,*args):return {'closed':True}
        before=self.data()
        with self.assertRaises(j.Conflict):self.driver(Invalid())
        self.assertEqual(self.data(),before)
    def test_failed_prepare_publication_never_dispatches_recovery(self):
        before=self.data()
        with patch.object(j.os,'replace',side_effect=OSError('interrupted publication')):
            with self.assertRaises(OSError):self.driver()
        self.assertEqual(self.data(),before);self.assertEqual(self.calls,['prepare'])
        self.assertIsNone(self.case.pending()['recovery']);self.assertTrue(list(self.case.root.glob('.pending-*')))
        self.driver();self.driver();self.assertEqual(self.calls,['prepare','prepare','recover'])
    def test_closed_store_never_falls_back_to_current_directory(self):
        self.store.close()
        with patch.object(j.os,'open',side_effect=AssertionError('closed store must not open relative files')):
            with self.assertRaises(j.Conflict):self.store.status()
            with self.assertRaises(j.Conflict):
                with self.store.lease():pass
    def test_no_fence_and_unsupported_candidate_phase_never_call_adapter(self):
        other=m.JournalTests();other.setUp();self.addCleanup(other.doCleanups)
        other.write(other.begin)
        with other.store.lease() as lease:
            with self.assertRaises(j.Conflict):c.recover_step(other.store,lease,object(),other.request,self.rid)
        other.until('verify-candidate');other.write(other.envelope('problem',reason='reader-owner-lost'))
        before=(other.root/'journal.json').read_bytes()
        with other.store.lease() as lease:
            with self.assertRaises(j.Conflict):c.recover_step(other.store,lease,object(),other.request,self.rid)
        event=other.envelope('recovery-prepared',recovery=self.rid,plan=plan_for(other.pending()))
        with self.assertRaises(j.Conflict):other.write(event)
        self.assertEqual((other.root/'journal.json').read_bytes(),before)
    def test_prepared_intent_resync_failure_blocks_effect_after_lost_publish_reply(self):
        real=j.os.fsync
        def failed_directory(fd):
            if fd==self.store.fd:raise OSError('lost directory durability')
            return real(fd)
        with patch.object(j.os,'fsync',side_effect=failed_directory):
            with self.assertRaises(OSError):self.driver()
        self.assertIsNotNone(self.case.pending()['recovery']);before=self.data()
        with patch.object(j.os,'fsync',side_effect=failed_directory):
            with self.assertRaises(OSError):self.driver()
        self.assertEqual(self.calls,['prepare']);self.assertEqual(self.data(),before)
        self.driver();self.assertEqual(self.calls,['prepare','recover'])
    def test_verified_receipt_lost_reply_retries_durability_without_effect(self):
        self.driver();real=j.os.fsync;directories=[]
        def fail_second(fd):
            if fd==self.store.fd:
                directories.append(fd)
                if len(directories)==2:raise OSError('lost verified publication reply')
            return real(fd)
        with patch.object(j.os,'fsync',side_effect=fail_second):
            with self.assertRaises(OSError):self.driver()
        self.assertEqual(self.case.pending()['phase'],'disable-previous');before=self.data()
        self.driver();self.assertEqual(self.calls,['prepare','recover']);self.assertEqual(self.data(),before)
    def test_original_writer_fixture_read_only_migration_and_downgrade_refusal(self):
        fixture=ROOT/'tests/fixtures/collection-journal-v1';provenance=json.loads((fixture/'provenance.json').read_text())
        for name,h in provenance['sha256'].items():self.assertEqual(hashlib.sha256((fixture/name).read_bytes()).hexdigest(),h)
        spec=importlib.util.spec_from_file_location('original_writer',fixture/'writer.py');old=importlib.util.module_from_spec(spec);spec.loader.exec_module(old)
        original=json.loads((fixture/'journal.json').read_text());reproduced=self.case.root.parent/'original';reproduced.mkdir(mode=0o700)
        reproduced.rmdir();writer=old.Store(reproduced,create=True)
        try:
            with writer.lease() as lease:
                for event in original['events']:writer.record(lease,event)
            self.assertEqual(json.loads((reproduced/'journal.json').read_text()),original)
            old_state=writer.status();new=j.Store(reproduced)
            try:
                before=(reproduced/'journal.json').read_bytes();state=new.status()
                self.assertEqual((reproduced/'journal.json').read_bytes(),before)
                for request in state['requests'].values():self.assertIsNone(request['recovery'])
                normalized=copy.deepcopy(state)
                for request in normalized['requests'].values():request.pop('recovery');request.pop('candidate_recovery')
                self.assertEqual(normalized,old_state)
                request=state['requests'][state['pending']];plan=plan_for(request)
                event=dict(type='recovery-prepared',request=request['id'],phase=request['phase'],effect=j.effect(request,request['phase']),recovery=self.rid,plan=plan)
                with new.lease() as lease:new.record(lease,event)
                migrated=(reproduced/'journal.json').read_bytes();value=json.loads(migrated)
                self.assertEqual(value['format'],2);self.assertEqual(value['events'][:-1],original['events'])
                with self.assertRaises(old.Conflict):writer.status()
                self.assertEqual((reproduced/'journal.json').read_bytes(),migrated)
                with new.lease() as lease:new.record(lease,event)
                self.assertEqual((reproduced/'journal.json').read_bytes(),migrated)
                value['format']=1;(reproduced/'journal.json').write_text(json.dumps(value));bad=(reproduced/'journal.json').read_bytes()
                with self.assertRaises(j.Conflict):new.status()
                self.assertEqual((reproduced/'journal.json').read_bytes(),bad)
            finally:new.close()
        finally:writer.close()
    def test_actual_coordinator_kill_after_prepare_and_after_effect_before_receipt(self):
        work=self.case.root.parent;module_path=Path(__file__).resolve()
        child=work/'worker.py';child.write_text('''import importlib.util,json,os,signal,sys,time
from pathlib import Path
spec=importlib.util.spec_from_file_location('recovery_test',sys.argv[1]);m=importlib.util.module_from_spec(spec);spec.loader.exec_module(m)
j,c=m.j,m.c;root=Path(sys.argv[2]);rid,request,mode=sys.argv[3:];marker=root.parent/'killed-boundary';saved=root.parent/'effect.json'
def stop():
 marker.write_text(mode)
 while True:time.sleep(.1)
class Adapter:
 def prepare_recovery(self,request,specification,recovery_id):return m.plan_for(request)
 def recover(self,request,plan):
  observation=m.observed(plan)
  with saved.open('x') as f:f.write(json.dumps(observation));f.flush();os.fsync(f.fileno())
  stop()
store=j.Store(root);record=store.record
def boundary(*args):
 result=record(*args)
 if mode=='prepared':stop()
 return result
store.record=boundary
with store.lease() as lease:c.recover_step(store,lease,Adapter(),request,rid)
''')
        marker=work/'killed-boundary'
        for mode in ('prepared','effect'):
            proc=subprocess.Popen([sys.executable,'-B',str(child),str(module_path),str(self.case.root),self.rid,self.case.request,mode],stdout=subprocess.DEVNULL,stderr=subprocess.PIPE)
            try:
                end=time.monotonic()+8
                while not marker.exists() and proc.poll() is None and time.monotonic()<end:time.sleep(.02)
                self.assertTrue(marker.exists(),proc.stderr.read().decode() if proc.poll() is not None else 'timeout')
                proc.kill();self.assertEqual(proc.wait(timeout=5),-signal.SIGKILL);marker.unlink()
            finally:
                if proc.poll() is None:proc.kill();proc.wait(timeout=5)
                proc.stderr.close()
            self.case.reopen();self.store=self.case.store
            self.assertIsNotNone(self.case.pending()['recovery']);self.assertTrue(self.case.pending()['uncertain'])
        outer=self;effect=work/'effect.json';effect_bytes=effect.read_bytes()
        class Adapter:
            def prepare_recovery(self,*args):raise AssertionError('must use durable prepared intent')
            def recover(self,request,plan):
                value=json.loads(effect.read_text());outer.assertEqual(value,observed(plan));return value
        self.driver(Adapter());self.driver(Adapter());self.assertEqual(effect.read_bytes(),effect_bytes)
        self.assertEqual(self.case.pending()['phase'],'disable-previous');self.assertFalse(self.case.pending()['uncertain'])

if __name__=='__main__':unittest.main(verbosity=2)
