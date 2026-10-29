"""Private actual-plugin cleanup after the linked desktop itself exits."""
import contextlib,copy,json,os,signal,subprocess,sys,uuid
from pathlib import Path
sys.path.insert(0,'/lab/source')
import successor_cleanup as session_cleanup
import successor_inspection as authority_inspection
import runtime_authority as authority
import journal as j
from native_candidate_recovery import CandidateRecovery
from native_shared_runtime import NativeAdapter
from journal_runtime_fixture import ROOT,guard,read,save,sha

class DesktopCleanup(CandidateRecovery):
 effect_name='successor-cleanup-effect.json'
 def __init__(self,control,runtime,store,lease):
  guard();self.control=control;self.runtime=runtime;self.store=store;self.lease=lease
  self.request={'recovery':control['recovery']};self.stack=contextlib.ExitStack();self.spec=control['specification']
  self.roles=set(j.ROLES);self.closure=control['closure'];self.original=None;self.admission=None
 def verify_owner(self):
  root=ROOT/'successor-runtime'/authority.ANCHOR;expected=session_cleanup.context(root,self.store,self.lease)
  assert expected['specification']==self.spec and expected['closure_request']==self.closure['request'] and expected['closure_receipt_sha256']==self.closure['receipt_sha256']
  original=read(root/'reservation.json')['reservation'];assert j.digest(original)==expected['reservation_sha256']
  proof=read(Path(original['owner']['receipt']));assert proof['owner']==self.closure['owner'] and proof['root']==self.closure['root']
  assert original['owner']['receipt']==self.closure['receipt']
  if self.original is None:self.original=original;self.authority_id=expected['authority'];self.loaded=expected['loaded']
  else:assert self.original==original and self.authority_id==expected['authority'] and self.loaded==expected['loaded']
  for role in j.ROLES:self.runtime.verify_artifact(self.spec,role)
  return copy.deepcopy(expected['guardian'])
 def snapshot(self,closed=False):
  members=super().snapshot(closed)
  for role in j.ROLES:members[role]['binding']=self.loaded['members'][role]
  return members
 def prepare_desktop(self,expected):
  assert expected['authority']==self.authority_id and expected['loaded']==self.loaded and expected['guardian']==self.guardian
  return self.snapshot()
 def recover_desktop(self,plan):return self.recover(self.request,plan)


def worker():
 guard();control=read(ROOT/'successor-cleanup-control.json');store=j.Store(ROOT/'collection-journal');root=ROOT/'successor-runtime'/authority.ANCHOR
 try:
  with store.lease() as lease:
   old=session_cleanup.read_record(store,control['recovery'])
   if old is not None and old['verified'] is not None:result=session_cleanup.step(root,store,lease,object(),control['recovery'])
   else:
    runtime=NativeAdapter(store,lease)
    with DesktopCleanup(control,runtime,store,lease) as adapter:result=session_cleanup.step(root,store,lease,adapter,control['recovery'])
   if control['fault']=='after-prepared':os.kill(os.getpid(),signal.SIGKILL)
   print(json.dumps({'verified':result['verified'] is not None}))
 finally:store.close()


def run(probe,original_guardian,specification,count_apps):
 from native_successor_inspection_fixture import snapshot as all_metadata
 def snapshot():return {k:v for k,v in all_metadata().items() if session_cleanup.PREFIX not in k and '.pending-desktop-successor-recovery-' not in k}
 replacement_guardian=read(ROOT/'desktop-cleanup-result.json')['original_guardian']
 guard();recovery=uuid.uuid4().hex;root=ROOT/'collection-journal';path=root/session_cleanup.filename(recovery)
 before=snapshot();pending=probe.pending['sha256'];requests=len(probe.hub.requests);assert probe.hub.block_cleanup
 def child(fault='',expected=0,wrong=False):
  save(ROOT/'successor-cleanup-control.json',{'recovery':recovery,'specification':specification,'closure':replacement_guardian if wrong else original_guardian,'fault':fault})
  q=subprocess.run([sys.executable,'-B',__file__,'--worker'],capture_output=True,text=True,timeout=90)
  with (ROOT/'successor-cleanup-worker.log').open('a') as stream:stream.write(json.dumps({'fault':fault,'wrong_guardian':wrong,'exit':q.returncode})+'\n'+q.stdout+q.stderr)
  assert q.returncode==expected,(q.returncode,q.stderr[-1800:]);probe.check()
 child(expected=1,wrong=True);assert not path.exists() and len(probe.hub.requests)==requests
 child('after-prepared',-signal.SIGKILL);prepared=read(path);assert prepared['verified'] is None and len(probe.hub.requests)==requests
 original_hash=sha(path);child(expected=1);assert sha(path)==original_hash and read(path)['verified'] is None
 refused=[dict(x) for x in probe.hub.requests if x['action']=='delete' and x.get('refused')];assert refused
 assert any(not x['deleted'] for x in probe.hub.snapshot())
 probe.hub.block_cleanup=False;child('after-cleanup',-signal.SIGKILL)
 assert sha(path)==original_hash and read(path)['verified'] is None;probe.no_admission();after_cleanup=len(probe.hub.requests)
 child();verified=read(path);j.recovered(verified['verified'],verified['plan']);assert len(probe.hub.requests)==after_cleanup
 verified_hash=sha(path);child();assert sha(path)==verified_hash and len(probe.hub.requests)==after_cleanup
 assert snapshot()==before and count_apps()==3
 from successor_authority import Authority
 try:opened=Authority(ROOT/'successor-runtime'/authority.ANCHOR)
 except (OSError,ValueError):pass
 else:
  opened.close();raise AssertionError('transport cleanup reopened old compositor authority')
 assert snapshot()==before
 assert sha(ROOT/'state/omarchy-urbit-dms.qualification-stable/outbound-messages.json')==pending
 assert probe.credentials()==probe.initial_credentials and sum(x['action']=='post' for x in probe.hub.requests)==probe.initial_logins
 assert len([x for x in probe.hub.pokes if x.get('uncertain') and x['owner']=='stable-dms'])==1
 save(ROOT/'successor-cleanup-result.json',{'passed':True,'recovery':recovery,'record_sha256':verified_hash,'controller_kills':2,
  'wrong_guardian_refused_before_effects':True,'failed_delete_preserved_prepared_intent':True,'prepared_sha256':original_hash,
  'verified_retry_no_http':True,'completed_retry_unchanged':True,'original_authority_and_journal_unchanged':before,
  'channels':probe.hub.snapshot(),'cleanup_refusals':refused,'application_creations':count_apps(),'replacement_applications':3,
  'credentials_unchanged':True,'credential_fingerprints':probe.initial_credentials,'unknown_dm_preserved':True,'no_new_logins':True,
  'original_guardian':original_guardian,'pending':probe.pending,'successor_activation_authorized':False,'old_authority_still_refuses_after_cleanup':True})
 print('PASS: linked desktop transport cleanup survives two controller kills and rejected DELETE; six sessions and unknown DM preserved')
 if (ROOT/'session-continuation-mode.json').exists():
  from native_successor_checkpoint_fixture import capture
  from native_session_continuation_fixture import run as continue_session
  capture(recovery);continue_session(recovery,probe,specification)
 return read(ROOT/'successor-cleanup-result.json')

if __name__=='__main__':worker()
