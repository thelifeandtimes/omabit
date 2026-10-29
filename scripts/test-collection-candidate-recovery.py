#!/usr/bin/env python3
"""Candidate recovery model/driver; synthetic proofs do not certify runtime cleanup."""
import copy,hashlib,importlib.util,json,os,signal,subprocess,sys,time,unittest,uuid
from pathlib import Path
from unittest.mock import patch
ROOT=Path(__file__).resolve().parents[1];sys.path.insert(0,str(ROOT/'native/collections'))
import journal as j
import coordinator as c
def module(name,path):
 s=importlib.util.spec_from_file_location(name,path);m=importlib.util.module_from_spec(s);s.loader.exec_module(m);return m
m=module('candidate_journal_fixture',ROOT/'scripts/test-collection-journal.py')
prior=module('candidate_old_recovery_fixture',ROOT/'scripts/test-collection-recovery.py')

def plan_for(request):
 old=prior.plan_for(request)
 return {'activation':{'collection':request['candidate'],'epoch':j.effect(request,'enable-candidate')},
  'loaded':copy.deepcopy(request['loaded']),'fences':copy.deepcopy(request['uncertain']),
  'guardian':old['guardian'],'closure_request':old['closure_request'],'authority':'e'*32,
  'reservation_sha256':j.digest(['candidate-reservation',request['candidate']]),
  'members':{r:{**v,'binding':copy.deepcopy(request['loaded']['members'][r]) if request['loaded'] else None} for r,v in old['members'].items()}}

def observed(plan):return prior.observed(plan)

class CandidateRecoveryTests(unittest.TestCase):
 def setUp(self):
  self.case=m.JournalTests();self.case.setUp();self.addCleanup(self.case.doCleanups)
  self.case.write(self.case.begin);self.case.until('enable-candidate');self.rid=uuid.uuid4().hex;self.calls=[]
 def fence(self,complete=False):
  if complete:self.case.until('verify-candidate')
  self.case.write(self.case.envelope('problem',reason='reader-owner-lost'))
  self.case.write(self.case.envelope('rollback-fenced',reason='loader-unavailable'))
 def data(self):return (self.case.root/'journal.json').read_bytes()
 def prepare(self):return self.case.envelope('candidate-recovery-prepared',recovery=self.rid,plan=plan_for(self.case.pending()))
 def verified(self,plan):return self.case.envelope('candidate-recovery-verified',recovery=self.rid,observation=observed(plan))
 def driver(self,adapter=None,rid=None):
  outer=self
  class Adapter:
   def prepare_recovery(self,request,specification,recovery):
    outer.calls.append('prepare');outer.assertEqual(specification,outer.case.dev);return plan_for(request)
   def recover(self,request,plan):outer.calls.append('recover');return observed(plan)
  with self.case.store.lease() as lease:
   return c.recover_candidate_step(self.case.store,lease,adapter or Adapter(),self.case.request,rid or self.rid)
 def test_fenced_rollback_preserves_uncertainty_until_verified(self):
  self.fence();old=json.loads(self.data())['events'];self.assertEqual(self.case.pending()['phase'],'drain-candidate')
  self.driver();self.case.reopen();self.assertTrue(self.case.pending()['uncertain'])
  self.assertIsNone(self.case.pending()['candidate_recovery']['verified'])
  self.driver();self.case.reopen();self.assertFalse(self.case.pending()['uncertain'])
  self.assertEqual(self.case.pending()['phase'],'disable-candidate')
  self.assertEqual(json.loads(self.data())['events'][:len(old)],old)
  saved=copy.deepcopy(self.case.pending()['candidate_recovery']);self.case.finish()
  self.assertEqual(self.case.pending()['outcome'],'rolled-back');self.assertEqual(self.case.pending()['candidate_recovery'],saved)
 def test_unfenced_and_wrong_phase_never_dispatch(self):
  before=self.data()
  with self.assertRaises(j.Conflict):self.driver()
  self.assertEqual(self.calls,[]);self.assertEqual(self.data(),before)
  with self.assertRaises(j.Conflict):self.case.write(self.case.envelope('rollback-fenced',reason='loader-unavailable'))
  self.assertEqual(self.data(),before)
 def test_prepared_recovery_keeps_ordinary_steps_fenced(self):
  self.fence();self.driver();before=self.data()
  class Adapter:
   def perform(self,*args):raise AssertionError('ordinary effects must remain fenced')
  with self.case.store.lease() as lease:
   with self.assertRaises(j.Conflict):c.step(self.case.store,lease,Adapter())
  self.assertEqual(self.data(),before)
 def test_complete_candidate_keeps_exact_loaded_instances(self):
  self.fence(complete=True);loaded=copy.deepcopy(self.case.pending()['loaded']);p=self.prepare();self.case.write(p)
  self.assertEqual(p['plan']['loaded'],loaded)
  self.case.write(self.verified(p['plan']));self.case.finish()
  self.assertEqual(self.case.pending()['candidate_recovery']['plan']['loaded'],loaded)
 def test_verification_without_plan_refuses(self):
  self.fence();before=self.data()
  with self.assertRaises(j.Conflict):self.case.write(self.verified(plan_for(self.case.pending())))
  self.assertEqual(self.data(),before)
 def test_invalid_plan_preserves_exact_bytes(self):
  self.fence();good=self.prepare();before=self.data()
  changes=[(('activation','collection'),'0'*64),(('activation','epoch'),'f'*32),(('loaded',),self.case.old),
   (('fences',),[]),(('authority',),'raw-cookie'),(('reservation_sha256',),'bad'),
   (('members','dms','binding'),self.case.old['members']['dms']),(('members','theme','channel_ids'),['a'*32,'a'*32])]
  for path,value in changes:
   with self.subTest(path=path):
    event=copy.deepcopy(good);target=event['plan']
    for key in path[:-1]:target=target[key]
    target[path[-1]]=value
    with self.assertRaises(j.Conflict):self.case.write(event)
    self.assertEqual(self.data(),before)
  event=copy.deepcopy(good);event['plan']['credentials']='do-not-persist'
  with self.assertRaises(j.Conflict):self.case.write(event)
  self.assertEqual(self.data(),before)
 def test_changed_guardian_channels_intent_or_session_refuses(self):
  self.fence();p=self.prepare();self.case.write(p);good=self.verified(p['plan']);before=self.data()
  changes=[(('plan_sha256',),'0'*64),(('guardian','owner','token'),'f'*32),(('closure_request',),'f'*32),
   (('proof',),'replacement-process'),(('members','theme','application_sha256'),'0'*64),
   (('members','dms','pending_ids'),[]),(('members','dms','credential_refs'),[]),
   (('members','dms','channel_ids'),[]),(('members','omabit','closed'),False)]
  for path,value in changes:
   event=copy.deepcopy(good);target=event['observation']
   for key in path[:-1]:target=target[key]
   target[path[-1]]=value
   with self.assertRaises(j.Conflict):self.case.write(event)
   self.assertEqual(self.data(),before)
 def test_new_fence_refuses_before_recovery_effect(self):
  self.fence();self.driver();self.case.write(self.case.envelope('problem',reason='helper-descendants-unconfirmed'));before=self.data()
  with self.assertRaises(j.Conflict):self.driver()
  self.assertEqual(self.calls,['prepare']);self.assertEqual(self.data(),before)
 def test_another_recovery_id_cannot_replace_intent(self):
  self.fence();self.driver();before=self.data()
  with self.assertRaises(j.Conflict):self.driver(rid=uuid.uuid4().hex)
  self.assertEqual(self.calls,['prepare']);self.assertEqual(self.data(),before)
 def test_exact_replay_after_rollback_has_no_effects(self):
  self.fence();p=self.prepare();self.case.write(p);v=self.verified(p['plan']);self.case.write(v);self.case.finish();before=self.data()
  self.case.write(p);self.case.write(v);self.driver()
  self.assertEqual(self.data(),before);self.assertEqual(self.calls,[])
 def test_busy_or_error_preserves_durable_intent(self):
  self.fence();self.driver();before=self.data()
  for error in (c.Busy(),RuntimeError('failed cleanup')):
   class Adapter:
    def recover(self,*args):raise error
   if isinstance(error,c.Busy):self.driver(Adapter())
   else:
    with self.assertRaises(RuntimeError):self.driver(Adapter())
   self.assertEqual(self.data(),before)
 def test_closed_lease_refuses_before_adapter(self):
  self.fence();before=self.data()
  with self.case.store.lease() as lease:pass
  with self.assertRaises(j.Conflict):c.recover_candidate_step(self.case.store,lease,object(),self.case.request,self.rid)
  self.assertEqual(self.data(),before)
 def test_failed_prepared_publication_does_not_cleanup(self):
  self.fence();before=self.data()
  with patch.object(j.os,'replace',side_effect=OSError('publish failed')):
   with self.assertRaises(OSError):self.driver()
  self.assertEqual(self.calls,['prepare']);self.assertEqual(self.data(),before)
  self.driver();self.assertEqual(self.calls,['prepare','prepare'])
 def test_lost_prepared_reply_resync_precedes_effect(self):
  self.fence();record=self.case.store.record
  def lost(*args):record(*args);raise OSError('lost durable response')
  with patch.object(self.case.store,'record',side_effect=lost):
   with self.assertRaises(OSError):self.driver()
  before=self.data();self.assertEqual(self.calls,['prepare'])
  with patch.object(j.os,'fsync',side_effect=OSError('not durable')):
   with self.assertRaises(OSError):self.driver()
  self.assertEqual(self.calls,['prepare']);self.assertEqual(self.data(),before)
  self.driver();self.assertEqual(self.calls,['prepare','recover'])
 def test_lost_verified_reply_retries_without_cleanup(self):
  self.fence();self.driver();record=self.case.store.record
  def lost(*args):
   result=record(*args)
   if args[1]['type']=='candidate-recovery-verified':raise OSError('lost verified reply')
   return result
  with patch.object(self.case.store,'record',side_effect=lost):
   with self.assertRaises(OSError):self.driver()
  before=self.data();self.driver();self.assertEqual(self.calls,['prepare','recover']);self.assertEqual(self.data(),before)
 def test_original_v2_writer_reproduces_and_refuses_successor(self):
  fixture=ROOT/'tests/fixtures/collection-journal-v2';old=module('candidate_original_journal2',fixture/'writer.py')
  for name,digest in json.loads((fixture/'provenance.json').read_text())['sha256'].items():
   self.assertEqual(hashlib.sha256((fixture/name).read_bytes()).hexdigest(),digest)
  original=json.loads((fixture/'journal.json').read_text());root=self.case.root.parent/'original-v2';writer=old.Store(root,create=True)
  try:
   with writer.lease() as lease:
    for event in original['events']:writer.record(lease,event)
   self.assertEqual(json.loads((root/'journal.json').read_text()),original);before=(root/'journal.json').read_bytes()
   reader=j.Store(root)
   try:
    normalized=reader.status()
    for request in normalized['requests'].values():self.assertIsNone(request.pop('candidate_recovery'))
    self.assertEqual(normalized,writer.status());self.assertEqual((root/'journal.json').read_bytes(),before)
   finally:reader.close()
  finally:writer.close()
  self.fence();self.assertEqual(json.loads(self.data())['format'],3);before=self.data();old_reader=old.Store(self.case.root)
  try:
   with self.assertRaises(old.Conflict):old_reader.status()
   self.assertEqual(self.data(),before)
  finally:old_reader.close()
  value=json.loads(before);value['format']=2;(self.case.root/'journal.json').write_text(json.dumps(value));bad=self.data()
  with self.assertRaises(j.Conflict):self.case.store.status()
  self.assertEqual(self.data(),bad)
 def test_previous_and_candidate_recoveries_coexist_without_rewriting(self):
  case=m.JournalTests();case.setUp();self.addCleanup(case.doCleanups);self.case=case
  case.write(case.begin);case.write(case.envelope('problem',reason='reader-owner-lost'))
  old_id=uuid.uuid4().hex
  class OldAdapter:
   def prepare_recovery(self,request,*args):return prior.plan_for(request)
   def recover(self,request,plan):return prior.observed(plan)
  for _ in range(2):
   with case.store.lease() as lease:c.recover_step(case.store,lease,OldAdapter(),case.request,old_id)
  original=copy.deepcopy(case.pending()['recovery']);case.until('enable-candidate')
  self.fence();self.driver();self.driver();case.finish()
  self.assertEqual(case.pending()['recovery'],original)
  self.assertNotEqual(case.pending()['candidate_recovery']['id'],old_id)
  before=self.data()
  with case.store.lease() as lease:c.recover_step(case.store,lease,object(),case.request,old_id)
  self.driver();self.assertEqual(self.data(),before)
 def test_actual_controller_kill_after_prepare_and_after_cleanup(self):
  self.fence();work=self.case.root.parent;child=work/'candidate-worker.py'
  child.write_text('''import importlib.util,json,os,sys,time
from pathlib import Path
s=importlib.util.spec_from_file_location('candidate_test',sys.argv[1]);m=importlib.util.module_from_spec(s);s.loader.exec_module(m)
j,c=m.j,m.c;root=Path(sys.argv[2]);rid,request,mode=sys.argv[3:];marker=root.parent/'boundary';saved=root.parent/'effect.json'
def stop():
 marker.write_text(mode)
 while True:time.sleep(.1)
class Adapter:
 def prepare_recovery(self,request,*args):return m.plan_for(request)
 def recover(self,request,plan):
  with saved.open('x') as f:f.write(json.dumps(m.observed(plan)));f.flush();os.fsync(f.fileno())
  stop()
store=j.Store(root);record=store.record
def boundary(*args):
 result=record(*args)
 if mode=='prepared':stop()
 return result
store.record=boundary
with store.lease() as lease:c.recover_candidate_step(store,lease,Adapter(),request,rid)
''')
  marker=work/'boundary'
  for mode in ('prepared','effect'):
   proc=subprocess.Popen([sys.executable,'-B',str(child),str(Path(__file__).resolve()),str(self.case.root),self.rid,self.case.request,mode],stdout=subprocess.DEVNULL,stderr=subprocess.PIPE)
   try:
    end=time.monotonic()+8
    while not marker.exists() and proc.poll() is None and time.monotonic()<end:time.sleep(.02)
    self.assertTrue(marker.exists(),proc.stderr.read().decode() if proc.poll() is not None else 'deadline')
    proc.kill();self.assertEqual(proc.wait(5),-signal.SIGKILL);marker.unlink()
   finally:
    if proc.poll() is None:proc.kill();proc.wait(5)
    proc.stderr.close()
   self.case.reopen();self.assertTrue(self.case.pending()['uncertain']);self.assertIsNone(self.case.pending()['candidate_recovery']['verified'])
  effect=work/'effect.json';before=effect.read_bytes()
  class Adapter:
   def recover(self,request,plan):return json.loads(effect.read_text())
  self.driver(Adapter());self.driver(Adapter());self.assertEqual(effect.read_bytes(),before)
  self.assertEqual(self.case.pending()['phase'],'disable-candidate')

if __name__=='__main__':unittest.main(verbosity=2)
