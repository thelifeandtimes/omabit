"""Private artifact6 native checkpoint2 continuation; never an installed-profile command."""
from pathlib import Path
import copy,json,os,signal,subprocess,sys,time,traceback,uuid
sys.path.insert(0,'/lab/source')
import admission
import journal as j
import coordinator as c
import runtime_authority as original
import session_continuation as session_successor
import successor_checkpoint as session_checkpoint
import continuation_authority as linked
import continuation_admission as successor_admission
import continuation_artifacts as successor_artifacts
import continuation_preflight as successor_preflight
from journal_runtime_fixture import ROOT,guard,read,save,sha,identity,gone
from native_shared_runtime import NativeAdapter as Shared
from native_recovery_runtime import NativeAdapter as Native
from native_session_continuation_fixture import SuccessorCohort
from supervised_shell import SupervisedShell


class Adapter(Shared):
 artifact_module=successor_artifacts
 admission_module=successor_admission
 def __init__(self,store=None,lease=None):
  super().__init__(store,lease);self.control=read(ROOT/'continuation-activation-control.json')
  self.authority_root=Path(self.control['display']).parent/original.ANCHOR
 def preflight(self,spec):
  return successor_preflight.require_ready(spec,{role:str(self.artifact_path(spec,role)) for role in j.ROLES})
 def cohort(self,store,lease):return SuccessorCohort(self.control,self,store,lease)
 def predecessor(self,store,lease):
  value=session_successor.read(store,self.control['attempt']);assert value
  # This verifies the original cleanup, checkpoint and canonical history links
  # without reopening its retired compositor authority.
  a=linked.Authority(self.authority_root)
  try:linked.linked(a.descriptor)
  finally:a.close()
  with self.cohort(store,lease) as cohort:
   assert cohort.inspect_successor({k:value[k] for k in ('specification','loaded','members')})==value['members']
  return value
 def issue(self,spec,epoch,request):
  self.preflight(spec);path=ROOT/'gates'/epoch;path.mkdir(mode=0o700,exist_ok=True)
  permit={'id':epoch,'request':request,'epoch':epoch,'specification':spec,'shell':admission.identity(self.pid)}
  with self.controller() as (store,lease):
   a=linked.Authority(self.authority_root)
   try:
    if a.status()['reservation'] is None:
     with self.cohort(store,lease) as observer:ticket=a.reserve(store,lease,path,permit,guardian_receipt=self.guardian_receipt(),adapter=observer)
    else:ticket=a.reserve(store,lease,path,permit,guardian_receipt=self.guardian_receipt(),adapter=object())
   finally:a.close()
  if self.control['fault']=='after-reservation':
   save(ROOT/'continuation-reservation-loss.json',{'ticket':ticket,'reservation_sha256':sha(self.authority_root/'reservation.json'),'gate':str(path)})
   os.kill(os.getpid(),signal.SIGKILL)
  gate=self.gate(path)
  try:gate.issue(permit,ticket)
  finally:gate.close()
  return path,permit,{'root':str(path),'permit':epoch,'epoch':epoch,'collection':j.digest(spec),'authority':ticket}
 def bind(self,store,lease,epoch):
  a=linked.Authority(self.authority_root)
  try:return a.bind_loaded(store,lease,a._ticket(a.status()['reservation']),self.bound(epoch),self.guardian_receipt())
  finally:a.close()
 def perform(self,phase,request,spec,role):
  if phase in ('drain-previous','disable-previous'):
   self.preflight(self.control['candidate'])
   with self.controller() as (store,lease):value=self.predecessor(store,lease)
   assert request['previous']==value['loaded'] and spec==value['specification']
   assert not self.snapshot()['loaded'] and all(gone(p) for rows in read(self.bound(value['loaded']['epoch']))['helpers'].values() for p in rows)
   if phase=='drain-previous':
    row=value['members'][role]
    return {'epoch':value['loaded']['epoch'],'binding':value['loaded']['members'][role],
     'absent':False,'closed':True,'retained_state':row['application_sha256'],'pending_ids':row['pending_ids']}
   return {'epoch':value['loaded']['epoch'],'absent':True,'helpers_exited':True}
  return super().perform(phase,request,spec,role)


def worker():
 guard();control=read(ROOT/'continuation-activation-control.json')
 os.environ.update(XDG_RUNTIME_DIR=str(Path(control['display']).parent),WAYLAND_DISPLAY=Path(control['display']).name)
 store=j.Store(ROOT/'collection-journal')
 try:
  with store.lease() as lease:
   adapter=Adapter(store,lease)
   if control['action']=='publish':
    with adapter.cohort(store,lease) as cohort:result=linked.step(control['display'],store,lease,cohort,control['attempt'])
    if control['fault']=='after-publication-step':os.kill(os.getpid(),signal.SIGKILL)
    print(json.dumps(result))
   else:
    result=c.step(store,lease,adapter);print(json.dumps({'pending':result['pending']}))
 finally:store.close()


def run(recovery,attempt,display,probe):
 guard();store=j.Store(ROOT/'collection-journal');proc=None;report={'passed':False,'status':'starting'}
 control={'recovery':recovery,'attempt':attempt,'display':str(display.path),'action':'publish','fault':''}
 intent=session_successor.read(store,attempt);original_spec=intent['specification'];spec=copy.deepcopy(original_spec);old_loaded=intent['loaded']
 upgrade=(ROOT/'continuation-activation-mode.json').exists()
 if upgrade:
  rows=read(ROOT/'continuation-upgrade-prepared.json')['stable']
  for role in j.ROLES:
   assert rows[role]['descriptor']['format']==6
   spec['members'][role]['artifact_sha256']=rows[role]['artifact_sha256']
   assert spec['members'][role]['artifact_sha256']!=original_spec['members'][role]['artifact_sha256']
 control['candidate']=spec;control['specification']=original_spec
 old_input=read(ROOT/'coordinator-input.json');prefix=read(store.root/'journal.json')['events'];oldmirror=sha(store.root/original.MIRROR)
 oldruntime={str(p.relative_to(ROOT)):sha(p) for name in ('runtime','successor-runtime') for p in (ROOT/name/original.ANCHOR).rglob('*') if p.is_file()}
 initial_requests=len(probe.hub.requests);initial_channels=copy.deepcopy(probe.hub.snapshot())
 def child(action='step',fault='',expected=0):
  value=dict(control,action=action,fault=fault);save(ROOT/'continuation-activation-control.json',value)
  save(ROOT/'coordinator-fault.json',{'phase':fault,'role':None} if fault in ('enable-candidate','verify-candidate') else {})
  q=subprocess.run([sys.executable,'-B',__file__,'--worker'],capture_output=True,text=True,timeout=75)
  with (ROOT/'continuation-activation-worker.log').open('a') as stream:stream.write(json.dumps({'action':action,'fault':fault,'exit':q.returncode})+'\n'+q.stdout+q.stderr)
  assert q.returncode==expected,(q.returncode,q.stderr[-2500:]);probe.check();return q
 def count_apps():
  p=ROOT/'shell-continuation.log';return p.read_text().count('APPLICATION_CREATED ') if p.exists() else 0
 def raw(shell,target,method,*args):
  env=dict(os.environ,XDG_RUNTIME_DIR=str(display.root),WAYLAND_DISPLAY=display.path.name)
  q=subprocess.run(['/usr/bin/quickshell','ipc','--pid',str(shell.pid),'call','--',target,method,*args],env=env,capture_output=True,text=True,timeout=4)
  assert q.returncode==0 and q.stdout.strip()!='Target not found.',(target,method,q.stderr[-600:]);return q.stdout.strip()
 oldenv={key:os.environ.get(key) for key in ('XDG_RUNTIME_DIR','WAYLAND_DISPLAY')}
 try:
  for _ in range(4):child('publish','after-publication-step',-signal.SIGKILL)
  child('publish');assert len(probe.hub.requests)==initial_requests and probe.hub.snapshot()==initial_channels and count_apps()==0
  publication=linked.read(store,attempt);pub_hash=sha(store.root/linked.filename(attempt));pub_inode=(store.root/linked.filename(attempt)).stat().st_ino
  child('publish');assert sha(store.root/linked.filename(attempt))==pub_hash and (store.root/linked.filename(attempt)).stat().st_ino==pub_inode
  if (ROOT/'continuation-inspection-mode.json').exists():
   from native_continuation_inspection_fixture import observe
   result=observe('published-empty');assert not result['activations'] and result['context']['live_context']
  os.environ.update(XDG_RUNTIME_DIR=str(display.root),WAYLAND_DISPLAY=display.path.name)
  env=dict(os.environ,OMABIT_SUPERVISOR_QML_LOG='/lab/shell-continuation.log')
  proc=SupervisedShell(env=env);row=identity(proc.pid);row.pop('args')
  selected=copy.deepcopy(old_input['specs']);selected['continuation']=spec
  save(ROOT/'coordinator-input.json',{'shell':row,'supervisor':proc.owner,'specs':selected})
  adapter=Adapter()
  adapter.wait(lambda:subprocess.run(['/usr/bin/quickshell','ipc','--pid',str(proc.pid),'call','--','artifactProbe','state'],capture_output=True,text=True,timeout=4).stdout.strip().startswith('{'))
  adapter.wait(lambda:len(adapter.snapshot()['selected'])==9 and not adapter.snapshot()['scanning'])
  assert not adapter.snapshot()['loaded'] and count_apps()==0
  blocked=None
  if upgrade:
   before={'journal':sha(store.root/'journal.json'),'apps':count_apps(),'gates':sorted(p.name for p in (ROOT/'gates').iterdir())}
   blocked=successor_preflight.check(original_spec,{role:str(adapter.artifact_path(original_spec,role)) for role in j.ROLES})
   assert not blocked['ready'] and all(x['reason']=='runtime-protocol-unsupported' for x in blocked['members'].values())
   assert before=={'journal':sha(store.root/'journal.json'),'apps':count_apps(),'gates':sorted(p.name for p in (ROOT/'gates').iterdir())}
  ready=adapter.preflight(spec);assert ready['required']['authority']==3 and ready['ready']
  request=uuid.uuid4().hex
  with store.lease() as lease:
   if upgrade:store.record(lease,{'type':'register','spec':spec})
   store.record(lease,{'type':'begin','request':request,'candidate':j.digest(spec),'expected':j.digest(old_loaded)})
  for _ in range(5):child()
  assert store.status()['requests'][request]['phase']=='enable-candidate' and count_apps()==0
  child(fault='after-reservation',expected=-signal.SIGKILL)
  lost=read(ROOT/'continuation-reservation-loss.json');assert not list(Path(lost['gate']).iterdir()) and count_apps()==0
  child(fault='enable-candidate',expected=-signal.SIGKILL);assert count_apps()==3
  assert sha(display.root/original.ANCHOR/'reservation.json')==lost['reservation_sha256']
  active_binding=read(ROOT/'lost-reply.json')['observation'];channels=copy.deepcopy(probe.hub.snapshot())
  child();assert store.status()['requests'][request]['phase']=='verify-candidate' and count_apps()==3
  assert store.status()['requests'][request]['loaded']==active_binding
  assert {x['path'] for x in probe.hub.snapshot()}=={x['path'] for x in channels}
  child(fault='verify-candidate',expected=-signal.SIGKILL);assert count_apps()==3
  a=linked.Authority(display.root/original.ANCHOR)
  try:before=a.status();assert before['binding']['loaded']==active_binding
  finally:a.close()
  child();state=store.status();assert state['pending'] is None and state['committed']==active_binding
  journal_hash=sha(store.root/'journal.json');child();assert sha(store.root/'journal.json')==journal_hash and count_apps()==3
  a=linked.Authority(display.root/original.ANCHOR)
  try:assert a.status()==before
  finally:a.close()
  if (ROOT/'continuation-inspection-mode.json').exists():
   result=observe('committed-live');assert result['journal']['committed']==active_binding and result['context']['live_context']
   assert not result['activations'][-1]['closure']['verified_process_closure']
  probe.admitted('stable',proc,raw)
  ui=json.loads(raw(proc,'connectedProbe','state'));assert len(ui['dms']['unresolved'])==1
  assert sha(ROOT/'state/omarchy-urbit-dms.qualification-stable/outbound-messages.json')==probe.pending['sha256']
  assert probe.credentials()==probe.initial_credentials and sum(x['action']=='post' for x in probe.hub.requests)==probe.initial_logins
  assert len([x for x in probe.hub.pokes if x.get('uncertain') and x['owner']=='stable-dms'])==1
  active=probe.hub.active();assert len(active)==2 and sum(x['subscriptions'] for x in active)==5
  assert all(x['deleted'] and not x['active_streams'] for x in probe.hub.snapshot() if x['path'] in {v['path'] for v in initial_channels})
  events=read(store.root/'journal.json')['events'];assert events[:len(prefix)]==prefix and len(events)>len(prefix)
  assert sha(store.root/original.MIRROR)==oldmirror and all(sha(ROOT/name)==digest for name,digest in oldruntime.items())
  cleanup_result=None;shell_loss_exit=None
  cleanup_mode=(ROOT/'continuation-cleanup-mode.json').exists()
  if cleanup_mode:
   probe.hub.block_cleanup=True;display.close();shell_loss_exit=proc.wait(10);closure=proc.close_owned()
   deadline=time.monotonic()+8
   while any(x['active_streams'] for x in probe.hub.snapshot()) and time.monotonic()<deadline:time.sleep(.05)
   unresolved=copy.deepcopy(probe.hub.snapshot());assert all(not x['active_streams'] for x in unresolved)
   assert any(not x['deleted'] for x in unresolved if x['path'] in {row['path'] for row in active})
   save(ROOT/'continuation-transport-after-loss.json',{'channels':unresolved,'closure':closure,'shell_exit':shell_loss_exit})
   result=observe('guards-closed');assert result['context']['server']=='gone' and result['activations'][-1]['closure']['verified_process_closure']
   from native_continuation_cleanup_fixture import run as clean
   cleanup_result=clean(probe,closure,spec,count_apps)
  else:
   # Explicit lab shutdown closes its granted transport independently of any
   # future forward transition. It neither rolls back nor erases the ready commit.
   gate=adapter.gate(ROOT/'gates'/active_binding['epoch'])
   try:gate.stop_admission(active_binding['epoch'],uuid.uuid4().hex)
   finally:gate.close()
   adapter.wait(lambda:all(x['host']['transport'].get('closed') for x in adapter.snapshot()['loaded'].values()),75)
   adapter.release_guards(spec,read(adapter.bound(active_binding['epoch'])));adapter.set_enabled(spec,False)
   adapter.wait(lambda:not adapter.snapshot()['loaded']);probe.no_admission()
   raw(proc,'artifactProbe','stop');assert proc.wait(5)==0;closure=proc.close_owned()
   if (ROOT/'continuation-inspection-mode.json').exists():
    result=observe('guards-closed');assert result['context']['live_context'] and result['activations'][-1]['closure']['verified_process_closure']
  gate=adapter.gate(ROOT/'gates'/active_binding['epoch'])
  try:
   gate_value=gate.status()['record'];assert set(gate_value['claims'])==set(j.ROLES)
   assert all(gone(x['owner']) for x in gate_value['claims'].values())
   if not cleanup_mode:assert all(x['state']=='ended' for x in gate_value['claims'].values())
  finally:gate.close()
  assert sha(store.root/'journal.json')==journal_hash and count_apps()==3
  report={'passed':True,'status':'activated-and-closed','attempt':attempt,'request':request,'recovery':recovery,
   'publication_controller_kills':4,'activation_controller_kills':3,'publication_sha256':pub_hash,'publication_inode':pub_inode,
   'old_loaded':old_loaded,'replacement_loaded':active_binding,'preflight':ready,'reservation':before,
   'new_application_constructions':3,'completed_retry_no_effect':True,'early_reservation_preserved':True,
   'loaded_retry_preserved_instances':True,'readiness_binding_retry_unchanged':True,'ready_ui':ui,'ready_channels':active,
   'final_channels':probe.hub.snapshot(),'fresh_subscriptions':5,'sessions_preserved':6,'credential_fingerprints':probe.initial_credentials,
   'unknown_dm_preserved':True,'pending':probe.pending,'no_new_login_or_message_replay':True,
   'original_mirror_sha256':oldmirror,'original_runtime_files':oldruntime,'original_journal_prefix':j.digest(prefix),
   'canonical_journal_sha256':journal_hash,'guardian_closure':closure,'gate':gate_value,
   'same_profile_and_immutable_generations':not upgrade,'artifact_upgrade':{'from':5,'to':6,'old_set_refusal':blocked} if upgrade else None,
   'synthetic_eyre':True,'actual_native_plugins':True,'public_activation':False,
   'linked_desktop_cleanup':cleanup_result,'shell_loss_exit':shell_loss_exit}
  print('PASS: actual Theme/DMs reconnect under linked authority3; seven controller kills preserve sessions, pending DM and one three-app activation')
 except BaseException:
  report.update(status='failed',failure=traceback.format_exc());raise
 finally:
  if proc and proc.closed is None:proc.close_owned()
  for key,value in oldenv.items():
   if value is None:os.environ.pop(key,None)
   else:os.environ[key]=value
  save(ROOT/'coordinator-input.json',old_input);store.close();save(ROOT/'continuation-activation-result.json',report)
 return report

if __name__=='__main__':worker()
