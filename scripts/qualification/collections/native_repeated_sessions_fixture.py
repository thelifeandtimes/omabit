"""Three actual private desktop recoveries with one immutable package set.

Only the namespace-guarded qualification driver calls this fixture. Each round
owns its compositor and supervised shell; the baseline private keyring outlives
all rounds. No installed profile or host display is accepted.
"""
from pathlib import Path
import contextlib,copy,json,os,signal,subprocess,sys,time,traceback,uuid
sys.path.insert(0,'/lab/source')
import admission
import journal as j
import coordinator as c
import runtime_authority as original
import repeated_session as session_successor
import repeated_authority as linked
import repeated_admission as successor_admission
import repeated_artifacts as successor_artifacts
import repeated_preflight as successor_preflight
import repeated_checkpoint as checkpoint
import repeated_cleanup as cleanup
import repeated_inspection as inspection
import checkpoint_lineage
import session_checkpoint as checkpoint1
from journal_runtime_fixture import ROOT,guard,read,save,sha,identity,gone
from native_shared_runtime import NativeAdapter as Shared
from native_candidate_recovery import CandidateRecovery
from supervised_shell import SupervisedShell

class SuccessorCohort(CandidateRecovery):
 def __init__(self,control,runtime,store,lease):
  self.control=control;self.runtime=runtime;self.store=store;self.lease=lease
  self.stack=contextlib.ExitStack();self.roles=set(j.ROLES);self.spec=control['specification']
  self.original=None;self.admission=None
 def verify_owner(self):
  archived=checkpoint_lineage.read(self.control['checkpoint'],self.store.root)['head']
  receipt=checkpoint.parsed(archived,archived['cleanup']['object']['path'],reference=archived['cleanup'])
  assert receipt['plan']['specification']==self.spec
  self.loaded=receipt['plan']['loaded'];self.authority_id=receipt['plan']['authority']
  for role in j.ROLES:self.runtime.verify_artifact(self.spec,role)
  return copy.deepcopy(receipt['plan']['guardian'])
 def inspect_successor(self,plan):
  assert plan['specification']==self.spec and plan['loaded']==self.loaded
  value=self.snapshot(closed=True)
  for role in j.ROLES:value[role]['binding']=self.loaded['members'][role]
  return value

class Display:
 def __init__(self,index):self.index=index;self.root=ROOT/('repeated-runtime-'+str(index))
 def start(self):
  guard();self.root.mkdir(mode=0o700)
  env=dict(os.environ,XDG_RUNTIME_DIR=str(self.root),WLR_BACKENDS='headless',WLR_RENDERER='pixman',WLR_HEADLESS_OUTPUTS='1',XKB_CONFIG_ROOT='/tools/usr/share/X11/xkb')
  env.pop('WAYLAND_DISPLAY',None);env.pop('SWAYSOCK',None)
  self.log=(ROOT/('sway-repeat-'+str(self.index)+'.log')).open('wb')
  self.proc=subprocess.Popen(['/tools/usr/lib/ld-linux-x86-64.so.2','--library-path','/tools/usr/lib','/tools/usr/bin/sway','--config','/lab/sway.conf'],env=env,stdout=self.log,stderr=subprocess.STDOUT)
  deadline=time.monotonic()+15
  while time.monotonic()<deadline:
   assert self.proc.poll() is None,'private repeated compositor exited'
   paths=[p for p in self.root.glob('wayland-*') if p.is_socket()]
   if len(paths)==1:self.path=paths[0];self.context=original.context(self.path);return
   time.sleep(.05)
  raise AssertionError('private repeated compositor deadline')
 def close(self):
  if getattr(self,'proc',None):
   if self.proc.poll() is None:
    self.proc.terminate()
    try:self.proc.wait(timeout=8)
    except subprocess.TimeoutExpired:self.proc.kill();self.proc.wait(timeout=4)
   save(ROOT/('compositor-repeat-'+str(self.index)+'-exit.json'),{'pid':self.proc.pid,'exit':self.proc.returncode,'renderer':'pixman','backend':'headless'})
  if getattr(self,'log',None):self.log.close()

class Adapter(Shared):
 artifact_module=successor_artifacts
 admission_module=successor_admission
 def __init__(self,store=None,lease=None):
  super().__init__(store,lease);self.control=read(ROOT/'repeated-control.json')
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
   save(ROOT/'repeated-reservation-loss.json',{'ticket':ticket,'reservation_sha256':sha(self.authority_root/'reservation.json'),'gate':str(path)})
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


class DesktopCleanup(CandidateRecovery):
 effect_name='repeated-cleanup-effect.json'
 def __init__(self,control,runtime,store,lease):
  guard();self.control=control;self.runtime=runtime;self.store=store;self.lease=lease
  self.request={'recovery':control['recovery']};self.stack=contextlib.ExitStack();self.spec=control['specification']
  self.roles=set(j.ROLES);self.closure=control['closure'];self.original=None;self.admission=None
 def verify_owner(self):
  root=self.runtime.authority_root;expected=cleanup.context(root,self.store,self.lease)
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
 guard();control=read(ROOT/'repeated-control.json');fault=control['fault'];action=control['action']
 os.environ.update(XDG_RUNTIME_DIR=str(Path(control['display']).parent),WAYLAND_DISPLAY=Path(control['display']).name)
 store=j.Store(ROOT/'collection-journal');root=Path(control['display']).parent/original.ANCHOR
 try:
  with store.lease() as lease:
   runtime=Adapter(store,lease)
   if action in ('claim','publish'):
    with runtime.cohort(store,lease) as cohort:
     if action=='claim':result=session_successor.step(control['display'],store,lease,cohort,control['checkpoint'],control['attempt'])
     else:result=linked.step(control['display'],store,lease,cohort,control['attempt'])
    if fault=='after-step':os.kill(os.getpid(),signal.SIGKILL)
   elif action=='cleanup':
    previous=cleanup.read_record(store,control['cleanup_id'])
    if previous is not None and previous['verified'] is not None:result=cleanup.step(root,store,lease,object(),control['cleanup_id'])
    else:
     control['recovery']=control['cleanup_id']
     with DesktopCleanup(control,runtime,store,lease) as observer:result=cleanup.step(root,store,lease,observer,control['cleanup_id'])
    if fault=='after-prepared':os.kill(os.getpid(),signal.SIGKILL)
   elif action=='checkpoint':
    if fault=='before-publication':
     def interrupted(*args):os.kill(os.getpid(),signal.SIGKILL)
     checkpoint.provisioning.rename_no_replace=interrupted
    result=checkpoint.capture(root,store,lease,control['cleanup_id'])
    if fault=='after-publication':os.kill(os.getpid(),signal.SIGKILL)
   else:
    assert action=='step';result=c.step(store,lease,runtime)
   print(json.dumps({'action':action,'completed':True,'phase':result.get('phase')}))
 finally:store.close()


def metadata(root):
 roots=[ROOT/'collection-journal',root,ROOT/'gates',ROOT/'runtime-observations',ROOT/'receipts']
 files=[root.parent/original.ANCHOR_SEAL,root.parent/original.PROVISION_CLAIM]
 files += [p for directory in roots for p in directory.rglob('*') if p.is_file()]
 return {str(p.relative_to(ROOT)):{'inode':p.stat().st_ino,'sha256':sha(p)} for p in files
         if cleanup.PREFIX not in p.name and not p.name.startswith('.pending-desktop-repeat-cleanup-')}


def roundtrip(index,reference,probe):
 guard();display=Display(index);proc=None;store=j.Store(ROOT/'collection-journal')
 report={'passed':False,'round':index};old_input=read(ROOT/'coordinator-input.json')
 oldenv={key:os.environ.get(key) for key in ('XDG_RUNTIME_DIR','WAYLAND_DISPLAY')}
 log=ROOT/('shell-repeat-'+str(index)+'.log');attempt=uuid.uuid4().hex
 control={'checkpoint':reference,'attempt':attempt,'action':'claim','fault':'','round':index}
 def count_apps():return log.read_text().count('APPLICATION_CREATED ') if log.exists() else 0
 def child(action='step',fault='',expected=0,**extra):
  save(ROOT/'repeated-control.json',dict(control,action=action,fault=fault,**extra))
  save(ROOT/'coordinator-fault.json',{'phase':fault,'role':None} if fault in ('enable-candidate','verify-candidate') else {})
  q=subprocess.run([sys.executable,'-B',__file__,'--worker'],capture_output=True,text=True,timeout=90)
  with (ROOT/('repeated-worker-'+str(index)+'.log')).open('a') as stream:stream.write(json.dumps({'action':action,'fault':fault,'exit':q.returncode})+'\n'+q.stdout+q.stderr)
  assert q.returncode==expected,(action,fault,q.returncode,q.stderr[-3000:]);probe.check();return q
 def raw(shell,target,method,*args):
  env=dict(os.environ,XDG_RUNTIME_DIR=str(display.root),WAYLAND_DISPLAY=display.path.name)
  q=subprocess.run(['/usr/bin/quickshell','ipc','--pid',str(shell.pid),'call','--',target,method,*args],env=env,capture_output=True,text=True,timeout=4)
  assert q.returncode==0 and q.stdout.strip()!='Target not found.',(target,method,q.stderr[-600:]);return q.stdout.strip()
 try:
  chain=checkpoint_lineage.read(reference,store.root);prior=chain['head'];spec=prior['inspection']['reservation']['reservation']['permit']['specification']
  old_loaded=prior['inspection']['reservation']['binding']['loaded'];assert spec==old_input['specs']['stable']
  control.update(specification=spec,candidate=spec,recovery=prior['id'])
  original_refs=copy.deepcopy(chain['references']);prefix=read(store.root/'journal.json')['events']
  oldmirror=sha(store.root/original.MIRROR);requests=len(probe.hub.requests);channels=copy.deepcopy(probe.hub.snapshot())
  display.start();control['display']=str(display.path);root=display.root/original.ANCHOR
  for _ in range(4):child('claim','after-step',-signal.SIGKILL)
  child('claim');intent=session_successor.read(store,attempt);session_successor.check_claimed(intent)
  claim_reference=session_successor.reference(store.root/session_successor.filename(attempt))
  child('claim');assert session_successor.reference(store.root/session_successor.filename(attempt))==claim_reference
  assert len(probe.hub.requests)==requests and probe.hub.snapshot()==channels and count_apps()==0
  for _ in range(4):child('publish','after-step',-signal.SIGKILL)
  child('publish');publication=session_successor.reference(store.root/linked.filename(attempt))
  child('publish');assert session_successor.reference(store.root/linked.filename(attempt))==publication
  assert len(probe.hub.requests)==requests and probe.hub.snapshot()==channels and count_apps()==0
  observations={}
  def inspect(label):
   before=metadata(root);value=inspection.inspect(root,store.root);assert metadata(root)==before
   assert value['format']==4 and not value['activation_authorized']
   observations[label]={'inspection':value,'unchanged':before};return value
  assert inspect('published-empty')['activations']==[]
  os.environ.update(XDG_RUNTIME_DIR=str(display.root),WAYLAND_DISPLAY=display.path.name)
  proc=SupervisedShell(env=dict(os.environ,OMABIT_SUPERVISOR_QML_LOG=str(log)))
  row=identity(proc.pid);row.pop('args');save(ROOT/'coordinator-input.json',dict(old_input,shell=row,supervisor=proc.owner))
  adapter=Adapter();adapter.wait(lambda:raw(proc,'artifactProbe','state').startswith('{'))
  adapter.wait(lambda:len(adapter.snapshot()['selected'])==9 and not adapter.snapshot()['scanning'])
  assert not adapter.snapshot()['loaded'] and count_apps()==0
  ready=adapter.preflight(spec);assert ready['ready'] and ready['required']['authority']==4
  request=uuid.uuid4().hex
  with store.lease() as lease:store.record(lease,{'type':'begin','request':request,'candidate':j.digest(spec),'expected':j.digest(old_loaded)})
  for _ in range(5):child()
  assert store.status()['requests'][request]['phase']=='enable-candidate' and count_apps()==0
  child(fault='after-reservation',expected=-signal.SIGKILL);lost=read(ROOT/'repeated-reservation-loss.json')
  assert not list(Path(lost['gate']).iterdir()) and count_apps()==0
  child(fault='enable-candidate',expected=-signal.SIGKILL);assert count_apps()==3
  assert sha(root/'reservation.json')==lost['reservation_sha256'];binding=read(ROOT/'lost-reply.json')['observation']
  active_paths={x['path'] for x in probe.hub.snapshot()};child()
  assert store.status()['requests'][request]['loaded']==binding and count_apps()==3
  assert {x['path'] for x in probe.hub.snapshot()}==active_paths
  child(fault='verify-candidate',expected=-signal.SIGKILL);assert count_apps()==3
  a=linked.Authority(root)
  try:reservation=a.status();assert reservation['binding']['loaded']==binding
  finally:a.close()
  child();state=store.status();assert state['pending'] is None and state['committed']==binding
  canonical=sha(store.root/'journal.json');child();assert sha(store.root/'journal.json')==canonical and count_apps()==3
  a=linked.Authority(root)
  try:assert a.status()==reservation
  finally:a.close()
  assert inspect('committed-live')['context']['live_context']
  probe.admitted('stable',proc,raw);ui=json.loads(raw(proc,'connectedProbe','state'));assert len(ui['dms']['unresolved'])==1
  active=copy.deepcopy(probe.hub.active());assert len(active)==2 and sum(x['subscriptions'] for x in active)==5
  assert len(probe.hub.snapshot())==len(channels)+2
  probe.hub.block_cleanup=True;display.close();shell_exit=proc.wait(10);closure=proc.close_owned()
  deadline=time.monotonic()+8
  while any(x['active_streams'] for x in probe.hub.snapshot()) and time.monotonic()<deadline:time.sleep(.05)
  after_loss=copy.deepcopy(probe.hub.snapshot());assert all(not x['active_streams'] for x in after_loss)
  assert len([x for x in after_loss if not x['deleted']])==2
  value=inspect('compositor-closed');assert value['context']['server']=='gone' and value['activations'][-1]['closure']['verified_process_closure']
  control.update(cleanup_id=uuid.uuid4().hex,closure=closure);path=store.root/cleanup.filename(control['cleanup_id'])
  before=metadata(root);requests=len(probe.hub.requests)
  child('cleanup',expected=1,closure=read(ROOT/'desktop-cleanup-result.json')['original_guardian'])
  assert not path.exists() and len(probe.hub.requests)==requests
  child('cleanup','after-prepared',-signal.SIGKILL);prepared=sha(path);assert read(path)['verified'] is None
  assert len(probe.hub.requests)==requests
  child('cleanup',expected=1);assert sha(path)==prepared and any(x.get('refused') for x in probe.hub.requests[requests:])
  probe.hub.block_cleanup=False;child('cleanup','after-cleanup',-signal.SIGKILL)
  assert sha(path)==prepared;probe.no_admission();requests=len(probe.hub.requests)
  child('cleanup');verified=read(path);j.recovered(verified['verified'],verified['plan']);receipt=session_successor.reference(path)
  child('cleanup');assert session_successor.reference(path)==receipt and len(probe.hub.requests)==requests
  assert metadata(root)==before and count_apps()==3
  capture_path=store.root/checkpoint.filename(control['cleanup_id'])
  child('checkpoint','before-publication',-signal.SIGKILL);assert not capture_path.exists()
  child('checkpoint','after-publication',-signal.SIGKILL);next_reference=session_successor.reference(capture_path)
  child('checkpoint');assert session_successor.reference(capture_path)==next_reference
  captured=read(capture_path);checkpoint.validate(captured)
  assert captured['format']==3 and captured['inspection']['format']==4
  assert not any(ref['object']['path'] in captured['files'] for ref in original_refs)
  retired=ROOT/('retained-repeated-runtime-'+str(index));display.root.rename(retired)
  child('checkpoint');assert session_successor.reference(capture_path)==next_reference and not display.root.exists()
  following=checkpoint_lineage.read(next_reference,store.root)
  assert len(following['records'])==index+1 and following['references'][1:]==original_refs
  assert [x['format'] for x in following['records']]==[3]*index+[1]
  assert sha(store.root/'journal.json')==canonical and sha(store.root/original.MIRROR)==oldmirror
  assert read(store.root/'journal.json')['events'][:len(prefix)]==prefix
  assert probe.credentials()==probe.initial_credentials
  assert sha(ROOT/'state/omarchy-urbit-dms.qualification-stable/outbound-messages.json')==probe.pending['sha256']
  assert sum(x['action']=='post' for x in probe.hub.requests)==probe.initial_logins
  assert len([x for x in probe.hub.pokes if x.get('uncertain') and x['owner']=='stable-dms'])==1
  report.update(passed=True,attempt=attempt,request=request,context=display.context,specification=spec,
   intent=claim_reference,publication=publication,cleanup=receipt,checkpoint=next_reference,
   controller_kills={'ownership':4,'publication':4,'activation':3,'cleanup':2,'checkpoint':2},
   new_application_constructions=3,fresh_subscriptions=5,sessions_preserved=6,unknown_dm_preserved=True,
   no_new_login_or_message_replay=True,ready_ui=ui,ready_channels=active,channels_after_loss=after_loss,
   final_channels=probe.hub.snapshot(),original_journal_prefix=j.digest(prefix),journal_sha256=canonical,
   guardian_closure=closure,shell_loss_exit=shell_exit,retained_runtime=str(retired),lineage_usage=following['usage'],
   observations=observations,same_immutable_generations=True,preflight=ready)
  print('PASS: repeated native session',index,'preserves credentials and pending DM with five fresh subscriptions and15 controller kills',flush=True)
  return next_reference,report
 except BaseException:
  report.update(failure=traceback.format_exc());raise
 finally:
  if proc and proc.closed is None:proc.close_owned()
  display.close()
  for key,value in oldenv.items():
   if value is None:os.environ.pop(key,None)
   else:os.environ[key]=value
  save(ROOT/'coordinator-input.json',old_input);store.close();save(ROOT/('repeated-session-'+str(index)+'-result.json'),report)


def run(recovery,probe):
 guard();reference=session_successor.reference(ROOT/'collection-journal'/checkpoint1.filename(recovery))
 initial_channels=len(probe.hub.snapshot());reports=[];report={'passed':False,'sessions':reports}
 try:
  for index in range(1,4):reference,result=roundtrip(index,reference,probe);reports.append(result)
  assert all(row['specification']==reports[0]['specification'] for row in reports)
  assert len({j.digest(row['context']) for row in reports})==3
  assert len(probe.hub.snapshot())==initial_channels+6;probe.no_admission()
  assert sum(row['new_application_constructions'] for row in reports)==9
  assert all(read(ROOT/('compositor-repeat-'+str(i)+'-exit.json'))['exit']==0 for i in range(1,4))
  report.update(passed=True,final_checkpoint=reference,sessions_preserved=6,unknown_dm_preserved=True,
   new_application_constructions=9,controller_kills=45,initial_channels=initial_channels,channels=probe.hub.snapshot(),
   artifact_format=7,authority_format=4,intent_format=3,checkpoint_format=3,cleanup_format=4,
   actual_native_plugins=True,synthetic_eyre=True,public_installed_activation=False)
 finally:save(ROOT/'repeated-sessions-result.json',report)

if __name__=='__main__':worker()
