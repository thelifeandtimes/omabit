"""Actual private native recovery after unadopted initial shell loss."""
from pathlib import Path
import json,os,signal,subprocess,sys,threading,traceback,uuid
sys.path.insert(0,'/lab/source')
import journal as j
import runtime_authority as authority
import startup_recovery as recovery
from native_shared_runtime import NativeAdapter,Base,initialize,admission
from native_connected_fixture import Probe
from journal_runtime_fixture import ROOT,guard,read,save,sha,identity
from supervised_shell import SupervisedShell

def controller():
 guard();control=read(ROOT/'initial-owner-control.json');store=j.Store(ROOT/'collection-journal')
 try:
  with store.lease() as lease:
   adapter=NativeAdapter(store,lease);a=authority.Authority(ROOT/'runtime'/authority.ANCHOR)
   try:
    if control['command']=='cleanup':
     saved=recovery.read_record(a,control['recovery'])
     if saved is not None and saved['verified'] is not None:
      result=recovery.step(a,store,lease,object(),control['recovery'])
     else:
      from native_initial_cleanup import InitialCleanup
      with InitialCleanup(control,adapter,store,lease) as cleanup:result=recovery.step(a,store,lease,cleanup,control['recovery'])
     if control['fault']=='after-prepared':os.kill(os.getpid(),signal.SIGKILL)
     print(json.dumps({'verified':result['verified'] is not None}));return
    assert control['command']=='restart';spec=adapter.input['specs']['stable'];epoch=control['epoch']
    adapter.preflight(spec)
    gate=ROOT/'gates'/epoch;gate.mkdir(mode=0o700,exist_ok=True)
    permit={'id':epoch,'epoch':epoch,'request':control['recovery'],'shell':authority.admission.identity(adapter.pid),'specification':spec}
    ticket=a.restart_initial(store,lease,gate,permit,control['recovery'],guardian_receipt=adapter.guardian_receipt())
    if control['fault']=='after-reservation':
     save(ROOT/'initial-restart-effect.json',{'ticket':ticket,'reservation_sha256':sha(a.root/'reservation.json')});os.kill(os.getpid(),signal.SIGKILL)
    state=store.status()
    if state['committed'] is None:
     loaded=adapter.enable(spec,epoch,control['recovery'])
     if control['fault']=='after-loaded':os.kill(os.getpid(),signal.SIGKILL)
     store.record(lease,{'type':'adopt','observation':Base.healthy(adapter,spec,loaded)})
    else:
     loaded=state['committed'];assert loaded['epoch']==epoch and loaded['collection']==j.digest(spec)
     assert adapter.loaded(spec,epoch)==loaded
    adapter.bind(store,lease,epoch);Base.healthy(adapter,spec,loaded)
    print(json.dumps({'adopted':loaded}))
   finally:a.close()
 finally:store.close()

def run():
 guard();case=read(ROOT/'initial-owner-mode.json').get('cohort','all');roles={'all':set(j.ROLES),'before-issue':set(),'after-issue':set(),'omabit-only':{'omabit'},'theme-only':{'theme'},'dms-only':{'dms'}}[case];initial_apps=len(roles)
 probe=Probe(ROOT);guardians=[];store=None;thread=None;stop=threading.Event();report={'passed':False,'status':'starting'}
 save(ROOT/'native-initial-owner-result.json',report)
 def raw(p,target,method,*args):
  q=subprocess.run(['/usr/bin/quickshell','ipc','--pid',str(p.pid),'call','--',target,method,*args],capture_output=True,text=True,timeout=4)
  assert q.returncode==0 and q.stdout.strip()!='Target not found.',q.stderr;return q.stdout.strip()
 def count_apps():return sum((ROOT/('shell-'+str(i)+'.log')).read_text().count('APPLICATION_CREATED ') for i in range(len(guardians)))
 def launch():
  p=SupervisedShell(env=dict(os.environ,OMABIT_SUPERVISOR_QML_LOG='/lab/shell-'+str(len(guardians))+'.log'));guardians.append(p)
  row=identity(p.pid);row.pop('args');save(ROOT/'coordinator-input.json',{'shell':row,'supervisor':p.owner,'specs':specs})
  adapter=NativeAdapter();adapter.wait(lambda:subprocess.run(['/usr/bin/quickshell','ipc','--pid',str(p.pid),'call','--','artifactProbe','state'],capture_output=True,text=True,timeout=4).stdout.strip().startswith('{'))
  adapter.wait(lambda:len(adapter.snapshot()['selected'])==9 and not adapter.snapshot()['scanning']);return p,adapter
 def record(event):
  with store.lease() as lease:return store.record(lease,event)
 def worker(command='cleanup',fault='',expected=0,wrong=False):
  value=dict(control,command=command,fault=fault)
  if wrong:value['closure']={'owner':proc.owner,'root':proc.root,'request':uuid.uuid4().hex,'receipt':str(proc.path),'receipt_sha256':sha(proc.path)}
  save(ROOT/'initial-owner-control.json',value)
  q=subprocess.run([sys.executable,'-B',__file__,'--controller'],capture_output=True,text=True,timeout=90)
  with (ROOT/'native-worker.log').open('a') as f:f.write(json.dumps({'command':command,'fault':fault,'wrong_guardian':wrong,'exit':q.returncode})+'\n'+q.stdout+q.stderr)
  assert q.returncode==expected,(q.returncode,q.stderr[-1800:]);probe.check()
 def initial_worker(expected):
  q=subprocess.run([sys.executable,'-B','/lab/native_initial_fixture.py','--controller'],capture_output=True,text=True,timeout=90)
  with (ROOT/'native-worker.log').open('a') as f:f.write(json.dumps({'command':'initial','exit':q.returncode})+'\n'+q.stdout+q.stderr)
  assert q.returncode==expected,(q.returncode,q.stderr[-1800:]);probe.check()
 try:
  prepared=read(ROOT/'prepared.json');(ROOT/'gates').mkdir(mode=0o700);(ROOT/'runtime-observations').mkdir(mode=0o700)
  probe.begin(ROOT,prepared)
  def observe():
   while not stop.is_set():save(ROOT/'hub-observation.json',{'url':probe.hub.url,'channels':probe.hub.snapshot()});stop.wait(.05)
  thread=threading.Thread(target=observe,daemon=True);thread.start();specs={}
  for name,role_versions in prepared.items():
   members={}
   for role,versions in role_versions.items():
    row=versions['1'];d=row['descriptor'];paths={k:d['environment']['XDG_'+k.upper()+'_HOME'] if role=='omabit' else str(ROOT/k/('omarchy-urbit-'+role+'.'+d['profile'])) for k in j.PATHS}
    members[role]={'plugin_id':d['plugin_id'],'profile':d['profile'],'artifact_sha256':row['artifact_sha256'],'paths':paths,
     'endpoint_ref':None if role=='omabit' else j.digest({'url':probe.hub.url,'ship':'~zod','profile':name,'role':role})}
   specs[name]={'format':1,'name':name,'members':members};j.specification(specs[name])
  proc,a=launch();store=j.Store(ROOT/'collection-journal',create=True);initialize(store)
  for spec in specs.values():record({'type':'register','spec':spec})
  phase={'all':None,'before-issue':'after-handoff','after-issue':'after-initial-issue','dms-only':'initial-partial-dms','theme-only':'initial-partial-theme','omabit-only':'initial-partial-omabit'}[case]
  save(ROOT/'coordinator-fault.json',{'phase':phase,'role':None} if phase else {});first_epoch=uuid.uuid4().hex
  save(ROOT/'initial-control.json',{'collection':'stable','epoch':first_epoch,'request':uuid.uuid4().hex,'fault':'after-loaded'})
  initial_worker(-signal.SIGKILL);assert count_apps()==initial_apps and store.status()['committed'] is None
  if case=='all':probe.admitted('stable',proc,raw)
  elif 'dms' in roles:probe.seed_pending(proc,raw)
  pending_path=ROOT/'state/omarchy-urbit-dms.qualification-stable/outbound-messages.json'
  unknown=sha(pending_path) if pending_path.exists() else None
  assert (unknown is not None)==('dms' in roles)
  original_value=read(ROOT/'runtime'/authority.ANCHOR/'reservation.json');assert original_value['binding'] is None
  original_reservation=original_value['reservation'];gate=Path(original_reservation['gate']['path'])
  probe.hub.blocked_cleanup_owners=['stable-'+role for role in ('theme','dms') if role in roles]
  old=proc;old.kill();assert old.wait(5)==-signal.SIGKILL;closure=old.close_owned()
  assert all(any(x['owner']==owner and not x['deleted'] for x in probe.hub.snapshot()) for owner in probe.hub.blocked_cleanup_owners)
  original_gate={p.name:sha(p) for p in gate.iterdir() if p.is_file()}
  save(ROOT/'coordinator-fault.json',{})
  proc,a=launch();before=sha(ROOT/'collection-journal/journal.json');original_hash=sha(ROOT/'runtime'/authority.ANCHOR/'reservation.json')
  initial_worker(1);assert sha(ROOT/'collection-journal/journal.json')==before and sha(ROOT/'runtime'/authority.ANCHOR/'reservation.json')==original_hash and count_apps()==initial_apps
  control={'recovery':uuid.uuid4().hex,'epoch':uuid.uuid4().hex,'closure':closure,'fault':''}
  path=ROOT/'runtime'/authority.ANCHOR/recovery.filename(control['recovery'])
  worker(expected=1,wrong=True);assert not path.exists()
  worker(fault='after-prepared',expected=-signal.SIGKILL);prepared_hash=sha(path)
  assert read(path)['verified'] is None and store.status()['committed'] is None
  if probe.hub.blocked_cleanup_owners:
   refusals=sum(bool(x.get('refused')) for x in probe.hub.requests);worker(expected=1)
   assert sum(bool(x.get('refused')) for x in probe.hub.requests)>refusals and sha(path)==prepared_hash
  worker('restart',expected=1);assert sha(ROOT/'runtime'/authority.ANCHOR/'reservation.json')==original_hash and count_apps()==initial_apps
  probe.hub.blocked_cleanup_owners=[];worker(fault='after-cleanup',expected=-signal.SIGKILL)
  effects=len(probe.hub.requests);assert sha(path)==prepared_hash
  worker();assert read(path)['verified'] is not None and len(probe.hub.requests)==effects
  worker();assert len(probe.hub.requests)==effects and count_apps()==initial_apps
  assert (sha(pending_path) if pending_path.exists() else None)==unknown
  worker('restart','after-reservation',-signal.SIGKILL);assert count_apps()==initial_apps and not a.snapshot()['loaded']
  next_value=read(ROOT/'runtime'/authority.ANCHOR/'reservation.json');assert next_value['format']==3 and len(next_value['history'])==1
  assert next_value['history'][0]['reservation']==original_reservation and next_value['history'][0]['handoff']['mode']=='initial-recovery'
  assert next_value['history'][0]['handoff']['closure']['sha256']==closure['receipt_sha256']
  worker('restart','after-loaded',-signal.SIGKILL);assert count_apps()==initial_apps+3 and store.status()['committed'] is None
  assert sha(ROOT/'runtime'/authority.ANCHOR/'reservation.json')==read(ROOT/'initial-restart-effect.json')['reservation_sha256']
  worker('restart');assert count_apps()==initial_apps+3 and store.status()['committed']['epoch']==control['epoch']
  proof_hash=(sha(ROOT/'collection-journal/journal.json'),sha(ROOT/'runtime'/authority.ANCHOR/'reservation.json'));effects=len(probe.hub.requests)
  worker('restart');worker();assert len(probe.hub.requests)==effects and count_apps()==initial_apps+3
  assert proof_hash==(sha(ROOT/'collection-journal/journal.json'),sha(ROOT/'runtime'/authority.ANCHOR/'reservation.json'))
  assert (sha(pending_path) if pending_path.exists() else None)==unknown
  probe.admitted('stable',proc,raw)
  if unknown is not None:assert sha(pending_path)==unknown
  assert {p.name:sha(p) for p in gate.iterdir() if p.is_file()}==original_gate
  committed=store.status()['committed'];g=admission.Gate(ROOT/'gates'/committed['epoch'])
  try:g.stop_admission(committed['epoch'],uuid.uuid4().hex)
  finally:g.close()
  a.wait(lambda:all(x['host']['transport']['closed'] for x in a.snapshot()['loaded'].values()),75)
  a.release_guards(specs['stable'],read(a.bound(committed['epoch'])));a.set_enabled(specs['stable'],False);a.wait(lambda:not a.snapshot()['loaded'])
  raw(proc,'artifactProbe','stop');assert proc.wait(5)==0;proc.close_owned();probe.no_admission()
  assert probe.credentials()==probe.initial_credentials and sum(x['action']=='post' for x in probe.hub.requests)==probe.initial_logins
  assert len([x for x in probe.hub.pokes if x.get('uncertain') and x['owner']=='stable-dms'])==1
  subscriptions=5+(4 if 'dms' in roles else 0)+(1 if 'theme' in roles else 0)
  assert sum(x['subscriptions'] for x in probe.hub.snapshot())==subscriptions
  report.update({'passed':True,'status':'complete','controller_kills':5,'original_shell_killed':True,'application_creations':initial_apps+3,'subscriptions':subscriptions,'initial_cohort':case,'initial_roles':sorted(roles),'initial_admission_missing':['admission.json','issued.json'] if case=='before-issue' else [],
   'channels':probe.hub.snapshot(),'cleanup_refusals':[x for x in probe.hub.requests if x.get('refused')],
   'ordinary_replacement_refused':True,'unverified_restart_refused':True,'wrong_guardian_refused':True,'failed_cleanup_preserved_intent':bool(roles & {'theme','dms'}),
   'cleanup_retry_no_http':True,'restart_retry_no_duplicate_construction':True,'completed_replay_no_effects':True,
   'unknown_dm_preserved':unknown is not None,'initial_outbound_absence_preserved':unknown is None,'no_new_logins':True,'credentials_unchanged':True,'credential_fingerprints':probe.initial_credentials,
   'original_guardian':closure,'guardians':[p.closed for p in guardians],'recovery':control['recovery'],'abandoned_gate_unchanged':True,
   'authority_history':read(ROOT/'runtime'/authority.ANCHOR/'reservation.json')['history'],
   'initial_owner_loss_recovery':True,'same_physical_compositor':True,'cross_session_bootstrap':False,'public_profile_command':False})
  print('PASS: initial owner loss ('+case+'), five controller crashes and original-cohort cleanup recover one replacement activation')
 except BaseException:
  report.update({'status':'failed','failure':traceback.format_exc()});raise
 finally:
  save(ROOT/'native-initial-owner-result.json',report)
  for p in guardians:
   if p.closed is None:p.close_owned()
  stop.set()
  if thread:thread.join(2)
  if store:store.close()
  probe.close()

if __name__=='__main__':controller() if '--controller' in sys.argv else run()
