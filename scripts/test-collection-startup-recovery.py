#!/usr/bin/env python3
"""Initial cleanup journal tests: real original guardian, synthetic app cohorts."""
import copy,hashlib,importlib.util,json,os,signal,subprocess,sys,unittest,uuid
from pathlib import Path
from unittest.mock import patch
ROOT=Path(__file__).resolve().parents[1];sys.path.insert(0,str(ROOT/'native/collections'))
import journal as j
import runtime_authority as authority
import startup_recovery as recovery
import shared_admission as shared

def module(name,path):
 s=importlib.util.spec_from_file_location(name,path);m=importlib.util.module_from_spec(s);s.loader.exec_module(m);return m
h=module('startup_handoff_helpers',Path(__file__).with_name('test-collection-authority-handoff.py'))

def members():
 return {role:{'binding':None,'application_sha256':j.digest([role,'app']),
  'pending_ids':['a'*32] if role=='dms' else [],'credential_refs':[j.digest([role,'session'])] if role!='omabit' else [],
  'transport_sha256':j.digest([role,'open']),'channel_ids':['b'*32] if role!='omabit' else []} for role in j.ROLES}

def observed(plan,runtime):
 value=runtime.status();receipt=value['reservation']['owner']['receipt']
 return {'plan_sha256':j.digest(plan),'guardian':plan['guardian'],'closure_request':plan['closure_request'],
  'closure_receipt_sha256':h.sha(Path(receipt)),'proof':'waitpid-echild',
  'members':{role:dict(value,closed=True,transport_sha256=j.digest([role,'closed'])) for role,value in plan['members'].items()}}

class StartupRecoveryTests(unittest.TestCase):
 provision=h.HandoffTests.provision;receive=h.HandoffTests.receive;cleanup=h.HandoffTests.cleanup
 send=h.HandoffTests.send;launch_guardian=h.HandoffTests.launch_guardian;start_claims=h.HandoffTests.start_claims
 record=h.HandoffTests.record
 pending=h.HandoffTests.pending;envelope=h.HandoffTests.envelope;write_binding=h.HandoffTests.write_binding;bind=h.HandoffTests.bind
 def setUp(self):
  h.fixture.AuthorityTests.setUp(self);self.gates=[];self.guardians=[];self.provision()
  self.guardian,self.owner,self.shell,self.receipt=self.launch_guardian();self.permit['shell']=self.shell
  with self.store.lease() as lease:self.ticket=self.authority.reserve(self.store,lease,self.gate,self.permit,guardian_receipt=self.receipt)
  self.current=shared.Gate(self.gate);self.gates.append(self.current);self.current.issue(self.permit,self.ticket);self.start_claims()
  self.rid=uuid.uuid4().hex;self.calls=[]
 def close_owner(self):
  self.send(self.guardian,{'command':'kill-shell'})
  self.send(self.guardian,{'command':'stop','request':uuid.uuid4().hex});self.assertEqual(0,self.guardian.wait(timeout=5))
 def step(self,adapter=None,rid=None):
  outer=self
  class Adapter:
   def prepare_initial(self,expected,spec):outer.calls.append('prepare');outer.assertEqual(spec,outer.spec);return members()
   def recover_initial(self,plan,spec):outer.calls.append('recover');outer.assertEqual(spec,outer.spec);return observed(plan,outer.authority)
  with self.store.lease() as lease:return recovery.step(self.authority,self.store,lease,adapter or Adapter(),rid or self.rid)
 def path(self):return self.authority.root/recovery.filename(self.rid)
 def plan(self):
  with self.store.lease() as lease:
   expected,_,_=recovery.context(self.authority,self.store,lease);return dict(expected,members=members())
 def test_live_original_owner_refuses_before_adapter(self):
  with self.assertRaises(j.Conflict):self.step()
  self.assertEqual([],self.calls);self.assertFalse(self.path().exists())
 def test_prepare_verify_and_completed_replay_preserve_all_cohorts(self):
  self.close_owner();before=(self.store.root/'journal.json').read_bytes();reservation=(self.authority.root/'reservation.json').read_bytes()
  record=self.step();self.assertIsNone(record['verified']);self.assertEqual(['prepare'],self.calls)
  self.assertEqual(record,recovery.read_record(self.authority,self.rid))
  done=self.step();self.assertIsNotNone(done['verified']);data=self.path().read_bytes()
  self.assertEqual(done,self.step(object()));self.assertEqual(['prepare','recover'],self.calls)
  self.assertEqual(data,self.path().read_bytes());self.assertEqual(before,(self.store.root/'journal.json').read_bytes())
  self.assertEqual(reservation,(self.authority.root/'reservation.json').read_bytes())
 def test_no_prepared_intent_means_no_verified_receipt(self):
  self.close_owner();plan=self.plan()
  with self.store.lease() as lease:
   with self.assertRaises(j.Conflict):recovery.verify(self.authority,self.store,lease,self.rid,observed(plan,self.authority))
  self.assertFalse(self.path().exists())
 def test_second_recovery_cannot_take_original_reservation(self):
  self.close_owner();self.step();data=self.path().read_bytes()
  with self.assertRaises(j.Conflict):self.step(rid=uuid.uuid4().hex)
  self.assertEqual(data,self.path().read_bytes())
 def test_bad_plan_fields_refuse_without_publication(self):
  self.close_owner();plan=self.plan()
  changes=[(('authority',),'f'*32),(('reservation_sha256',),'0'*64),(('activation','epoch'),'e'*32),
   (('journal','sha256'),'0'*64),(('guardian','owner','token'),'e'*32),(('closure_request',),'e'*32),
   (('admission','missing'),['issued.json']),(('members','theme','binding'),{}),(('members','dms','pending_ids'),['a'*32,'a'*32])]
  for keys,value in changes:
   changed=copy.deepcopy(plan);target=changed
   for key in keys[:-1]:target=target[key]
   target[keys[-1]]=value
   with self.subTest(keys=keys),self.store.lease() as lease:
    with self.assertRaises(j.Conflict):recovery.prepare(self.authority,self.store,lease,self.rid,changed)
   self.assertFalse(self.path().exists())
 def test_changed_registration_history_blocks_cleanup(self):
  self.close_owner();self.step();spec=copy.deepcopy(self.spec);spec['members']['dms']['artifact_sha256']=j.digest(['new'])
  self.record({'type':'register','spec':spec});data=self.path().read_bytes()
  with self.assertRaises(j.Conflict):self.step()
  self.assertEqual(['prepare'],self.calls);self.assertEqual(data,self.path().read_bytes())
 def test_changed_admission_bytes_block_cleanup(self):
  self.close_owner();self.step();p=self.gate/'admission.json';row=json.loads(p.read_text());row['claims']['dms']['instance']='f'*32;h.save(p,row)
  with self.assertRaises(j.Conflict):self.step()
  self.assertEqual(['prepare'],self.calls)
 def test_changed_original_closure_blocks_cleanup(self):
  self.close_owner();self.step();row=json.loads(self.receipt.read_text());row['request']='e'*32;h.save(self.receipt,row)
  with self.assertRaises(j.Conflict):self.step()
  self.assertEqual(['prepare'],self.calls)
 def test_adoption_prevents_initial_recovery(self):
  self.close_owner();loaded=h.model.loaded(self.spec,self.permit['epoch']);self.record({'type':'adopt','observation':h.model.ready(self.spec,loaded)})
  with self.assertRaises(j.Conflict):self.step()
  self.assertEqual([],self.calls)
 def test_cleanup_error_retains_prepared_record(self):
  self.close_owner();self.step();data=self.path().read_bytes()
  class Adapter:
   def recover_initial(self,*args):raise RuntimeError('cleanup refused')
  with self.assertRaises(RuntimeError):self.step(Adapter())
  self.assertEqual(data,self.path().read_bytes());self.assertIsNone(recovery.read_record(self.authority,self.rid)['verified'])
 def test_changed_intent_credential_or_channels_refuse(self):
  self.close_owner();record=self.step();good=observed(record['plan'],self.authority);data=self.path().read_bytes()
  changes=[(('members','dms','pending_ids'),[]),(('members','dms','credential_refs'),[]),
   (('members','theme','channel_ids'),[]),(('members','theme','application_sha256'),'0'*64),
   (('closure_receipt_sha256',),'0'*64),(('guardian','owner','token'),'f'*32)]
  for keys,value in changes:
   changed=copy.deepcopy(good);target=changed
   for key in keys[:-1]:target=target[key]
   target[keys[-1]]=value
   with self.store.lease() as lease:
    with self.assertRaises(j.Conflict):recovery.verify(self.authority,self.store,lease,self.rid,changed)
   self.assertEqual(data,self.path().read_bytes())
 def test_prepare_publication_failure_never_dispatches_cleanup(self):
  self.close_owner()
  with patch.object(recovery,'write_record',side_effect=OSError('disk')):
   with self.assertRaises(OSError):self.step()
  self.assertEqual(['prepare'],self.calls);self.assertFalse(self.path().exists())
 def test_lost_prepared_reply_resyncs_before_cleanup(self):
  self.close_owner();write=recovery.write_record
  def lost(*args):write(*args);raise OSError('reply lost')
  with patch.object(recovery,'write_record',side_effect=lost):
   with self.assertRaises(OSError):self.step()
  with patch.object(recovery,'sync_record',side_effect=OSError('fsync')):
   with self.assertRaises(OSError):self.step()
  self.assertEqual(['prepare'],self.calls);self.step();self.assertEqual(['prepare','recover'],self.calls)
 def test_lost_verified_reply_never_repeats_cleanup(self):
  self.close_owner();self.step();write=recovery.write_record
  def lost(*args):write(*args);raise OSError('reply lost')
  with patch.object(recovery,'write_record',side_effect=lost):
   with self.assertRaises(OSError):self.step()
  data=self.path().read_bytes();self.step(object());self.assertEqual(data,self.path().read_bytes())
  self.assertEqual(['prepare','recover'],self.calls)
 def test_closed_coordinator_lease_refuses(self):
  self.close_owner()
  with self.store.lease() as lease:pass
  with self.assertRaises(j.Conflict):recovery.step(self.authority,self.store,lease,object(),self.rid)
 def test_unknown_format_preserves_evidence(self):
  self.close_owner();self.step();record=json.loads(self.path().read_text());record['format']=99;h.save(self.path(),record);data=self.path().read_bytes()
  with self.assertRaises(j.Conflict):self.step()
  self.assertEqual(data,self.path().read_bytes())
 def test_original_reservation2_fixture_is_frozen(self):
  root=ROOT/'tests/fixtures/runtime-authority-v2';meta=json.loads((root/'provenance.json').read_text())
  for name,digest in meta['sha256'].items():self.assertEqual(digest,h.sha(root/name))


 def replacement(self):
  self.new_guardian,self.new_owner,self.new_shell,self.new_receipt=self.launch_guardian()
  path=self.root/('replacement-'+uuid.uuid4().hex);path.mkdir(mode=0o700)
  return path,{'id':uuid.uuid4().hex,'epoch':uuid.uuid4().hex,'request':self.rid,'shell':self.new_shell,'specification':self.spec}
 def restart(self,candidate):
  with self.store.lease() as lease:return self.authority.restart_initial(self.store,lease,*candidate,self.rid,guardian_receipt=self.new_receipt)
 def initial_ready(self):
  self.close_owner();self.step();self.step();return self.replacement()
 def adopt_replacement(self,candidate,ticket):
  self.gate,self.permit=candidate;self.guardian,self.owner,self.shell,self.receipt=self.new_guardian,self.new_owner,self.new_shell,self.new_receipt
  self.ticket=ticket;self.current=shared.Gate(self.gate);self.gates.append(self.current);self.current.issue(self.permit,ticket);self.start_claims()
  self.loaded=h.model.loaded(self.spec,self.permit['epoch']);self.record({'type':'adopt','observation':h.model.ready(self.spec,self.loaded)})
  self.binding_path=h.HandoffTests.write_binding(self)
  h.HandoffTests.bind(self)
 def test_initial_restart_requires_verified_cleanup(self):
  self.close_owner();self.step();candidate=self.replacement();before=(self.authority.root/'reservation.json').read_bytes()
  with self.assertRaises(j.Conflict):self.restart(candidate)
  self.assertEqual(before,(self.authority.root/'reservation.json').read_bytes());self.assertFalse(list(candidate[0].iterdir()))
 def test_initial_restart_preserves_history_and_replays_exactly(self):
  candidate=self.initial_ready();old=self.authority.status()['reservation'];ticket=self.restart(candidate)
  value=self.authority.status();self.assertEqual(3,value['format']);self.assertIsNone(value['binding'])
  self.assertEqual(old,value['history'][0]['reservation']);self.assertEqual('initial-recovery',value['history'][0]['handoff']['mode'])
  before=(self.authority.root/'reservation.json').read_bytes();self.assertEqual(ticket,self.restart(candidate));self.assertEqual(before,(self.authority.root/'reservation.json').read_bytes())
  with self.store.lease() as lease:
   self.assertEqual(ticket,self.authority.reserve(self.store,lease,*candidate,guardian_receipt=self.new_receipt))
  self.assertEqual(before,(self.authority.root/'reservation.json').read_bytes())
  self.adopt_replacement(candidate,ticket);self.assertEqual(self.loaded,self.authority.status()['binding']['loaded'])
 def test_initial_restart_rejects_changed_collection_or_request(self):
  candidate=self.initial_ready();before=(self.authority.root/'reservation.json').read_bytes()
  for field,value in [('request','f'*32),('specification',dict(self.spec,name='changed'))]:
   permit=dict(candidate[1]);permit[field]=value
   with self.assertRaises(j.Conflict):self.restart((candidate[0],permit))
   self.assertEqual(before,(self.authority.root/'reservation.json').read_bytes())
 def test_initial_restart_rejects_reused_gate_or_epoch(self):
  candidate=self.initial_ready()
  with self.assertRaises(j.Conflict):self.restart((self.gate,candidate[1]))
  permit=dict(candidate[1],epoch=self.permit['epoch'])
  with self.assertRaises(j.Conflict):self.restart((candidate[0],permit))
 def test_initial_restart_rejects_changed_closed_cohort_record(self):
  candidate=self.initial_ready();record=json.loads(self.path().read_text());record['verified']['members']['dms']['pending_ids']=[];h.save(self.path(),record)
  with self.assertRaises(j.Conflict):self.restart(candidate)
 def test_initial_restart_lost_reply_does_not_replace_twice(self):
  candidate=self.initial_ready();write=self.authority._save_reservation
  def lost(value):write(value);raise OSError('reply lost')
  with patch.object(self.authority,'_save_reservation',side_effect=lost):
   with self.assertRaises(OSError):self.restart(candidate)
  before=(self.authority.root/'reservation.json').read_bytes();self.restart(candidate)
  self.assertEqual(before,(self.authority.root/'reservation.json').read_bytes());self.assertEqual(1,len(self.authority.status()['history']))
 def test_original_reservation2_reader_refuses_initial_recovery_history(self):
  original=h.module('original_reservation2',ROOT/'tests/fixtures/runtime-authority-v2/writer.py')
  old=original.Authority(self.authority.root)
  try:self.assertEqual(self.authority.status(),old.status())
  finally:old.close()
  candidate=self.initial_ready();self.restart(candidate);before=(self.authority.root/'reservation.json').read_bytes()
  with self.assertRaises(j.Conflict):original.Authority(self.authority.root)
  self.assertEqual(before,(self.authority.root/'reservation.json').read_bytes())
 def test_changed_retained_initial_receipt_blocks_replay(self):
  candidate=self.initial_ready();self.restart(candidate);record=json.loads(self.path().read_text());h.save(self.path(),record)
  # Even equal JSON in a replacement inode cannot replace the bound proof file.
  with self.assertRaises(j.Conflict):self.restart(candidate)
 def test_ordinary_handoff_after_initial_recovery_keeps_both_histories(self):
  candidate=self.initial_ready();ticket=self.restart(candidate);self.adopt_replacement(candidate,ticket)
  h.HandoffTests.begin(self,self.spec);h.HandoffTests.close_claims(self);h.HandoffTests.drain_events(self);h.HandoffTests.unload_and_appearance(self)
  next_candidate=h.HandoffTests.candidate(self)
  with self.store.lease() as lease:ticket=self.authority.handoff(self.store,lease,*next_candidate)
  h.HandoffTests.activate(self,next_candidate,ticket)
  value=self.authority.status();self.assertEqual(3,value['format']);self.assertEqual(['initial-recovery','loaded-clean'],[x['handoff']['mode'] for x in value['history']])

if __name__=='__main__':unittest.main()
