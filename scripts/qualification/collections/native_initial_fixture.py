"""Private same-shell initial activation replay across controller loss.

The original live authority reservation supplies the frozen startup intent. This
qualifies bounded retry; it cannot adopt another shell, reset authority or recover
initial owner/compositor loss. No public bootstrap/profile command is provided.
"""
from pathlib import Path
import json,os,signal,subprocess,sys,threading,traceback,uuid
sys.path.insert(0,'/lab/source')
import journal as j
import runtime_authority as authority
from native_shared_runtime import NativeAdapter,Base,initialize,admission
from native_connected_fixture import Probe
from journal_runtime_fixture import ROOT,guard,read,save,sha,identity
from supervised_shell import SupervisedShell

def controller():
 guard();control=read(ROOT/'initial-control.json');store=j.Store(ROOT/'collection-journal')
 try:
  with store.lease() as lease:
   adapter=NativeAdapter(store,lease);spec=adapter.input['specs'][control['collection']]
   state=store.status();assert not state['requests'] and state['pending'] is None
   a=authority.Authority(ROOT/'runtime'/authority.ANCHOR)
   try:
    value=a.status();saved=value['reservation']
    if saved is not None:
     assert saved['intent']=='initial' and value['history']==[]
     assert saved['permit']['epoch']==control['epoch'] and saved['permit']['request']==control['request']
     assert saved['permit']['specification']==spec and saved['permit']['shell']==authority.admission.identity(adapter.pid)
     assert authority.process_pair(saved['permit']['shell'])==authority.process_pair(adapter.input['shell'])
     assert saved['owner']==authority.owner_for_shell(saved['permit']['shell'],adapter.guardian_receipt())
    else:assert state['committed'] is None
   finally:a.close()
   if state['committed'] is None:
    loaded=adapter.enable(spec,control['epoch'],control['request'])
    if control['fault']=='after-loaded':os.kill(os.getpid(),signal.SIGKILL)
    ready=Base.healthy(adapter,spec,loaded)
    store.record(lease,{'type':'adopt','observation':ready})
    if control['fault']=='after-adopt':os.kill(os.getpid(),signal.SIGKILL)
   else:
    loaded=state['committed'];assert loaded['collection']==j.digest(spec) and loaded['epoch']==control['epoch']
    assert adapter.loaded(spec,control['epoch'])==loaded
    Base.healthy(adapter,spec,loaded)
   adapter.bind(store,lease,control['epoch'])
   if control['fault']=='after-bind':os.kill(os.getpid(),signal.SIGKILL)
   print(json.dumps({'adopted':loaded}))
 finally:store.close()

def run():
 guard();probe=Probe(ROOT);proc=None;store=None;thread=None;stop=threading.Event();report={'passed':False,'status':'starting'}
 save(ROOT/'native-initial-result.json',report)
 def raw(p,target,method,*args):
  q=subprocess.run(['/usr/bin/quickshell','ipc','--pid',str(p.pid),'call','--',target,method,*args],capture_output=True,text=True,timeout=4)
  assert q.returncode==0 and q.stdout.strip()!='Target not found.',q.stderr;return q.stdout.strip()
 def count_apps():return (ROOT/'shell-0.log').read_text().count('APPLICATION_CREATED ')
 def record(event):
  with store.lease() as lease:return store.record(lease,event)
 def worker(fault='',expected=0,changed=None):
  value=dict(control,fault=fault)
  if changed:value.update(changed)
  save(ROOT/'initial-control.json',value)
  q=subprocess.run([sys.executable,'-B',__file__,'--controller'],capture_output=True,text=True,timeout=90)
  with (ROOT/'native-worker.log').open('a') as f:f.write(json.dumps({'fault':fault,'changed':bool(changed),'exit':q.returncode})+'\n'+q.stdout+q.stderr)
  assert q.returncode==expected,(q.returncode,q.stderr[-1800:]);probe.check()
 try:
  prepared=read(ROOT/'prepared.json');(ROOT/'gates').mkdir(mode=0o700);(ROOT/'runtime-observations').mkdir(mode=0o700)
  probe.begin(ROOT,prepared)
  def observe():
   while not stop.is_set():save(ROOT/'hub-observation.json',{'url':probe.hub.url,'channels':probe.hub.snapshot()});stop.wait(.05)
  thread=threading.Thread(target=observe,daemon=True);thread.start();specs={}
  for name,roles in prepared.items():
   members={}
   for role,versions in roles.items():
    row=versions['1'];d=row['descriptor'];paths={k:d['environment']['XDG_'+k.upper()+'_HOME'] if role=='omabit' else str(ROOT/k/('omarchy-urbit-'+role+'.'+d['profile'])) for k in j.PATHS}
    members[role]={'plugin_id':d['plugin_id'],'profile':d['profile'],'artifact_sha256':row['artifact_sha256'],'paths':paths,
     'endpoint_ref':None if role=='omabit' else j.digest({'url':probe.hub.url,'ship':'~zod','profile':name,'role':role})}
   specs[name]={'format':1,'name':name,'members':members};j.specification(specs[name])
  proc=SupervisedShell(env=dict(os.environ,OMABIT_SUPERVISOR_QML_LOG='/lab/shell-0.log'))
  row=identity(proc.pid);row.pop('args');save(ROOT/'coordinator-input.json',{'shell':row,'supervisor':proc.owner,'specs':specs})
  a=NativeAdapter();a.wait(lambda:subprocess.run(['/usr/bin/quickshell','ipc','--pid',str(proc.pid),'call','--','artifactProbe','state'],capture_output=True,text=True,timeout=4).stdout.strip().startswith('{'))
  a.wait(lambda:len(a.snapshot()['selected'])==9 and not a.snapshot()['scanning'])
  store=j.Store(ROOT/'collection-journal',create=True);initialize(store)
  for spec in specs.values():record({'type':'register','spec':spec})
  control={'collection':'stable','epoch':uuid.uuid4().hex,'request':uuid.uuid4().hex,'fault':''}
  save(ROOT/'coordinator-fault.json',{'phase':'after-handoff','role':None})
  worker(expected=-signal.SIGKILL);assert count_apps()==0 and store.status()['committed'] is None
  frozen=read(ROOT/'runtime'/authority.ANCHOR/'reservation.json')['reservation'];initial_hash=j.digest(frozen)
  assert frozen['intent']=='initial' and not list(Path(frozen['gate']['path']).iterdir())
  save(ROOT/'coordinator-fault.json',{})
  before=sha(ROOT/'collection-journal/journal.json');owner_before=sha(ROOT/'runtime'/authority.ANCHOR/'reservation.json')
  worker(expected=1,changed={'epoch':uuid.uuid4().hex})
  worker(expected=1,changed={'collection':'dev'})
  assert count_apps()==0 and sha(ROOT/'collection-journal/journal.json')==before and sha(ROOT/'runtime'/authority.ANCHOR/'reservation.json')==owner_before
  worker('after-loaded',-signal.SIGKILL);assert count_apps()==3 and store.status()['committed'] is None
  assert read(ROOT/'runtime'/authority.ANCHOR/'reservation.json')['binding'] is None
  loaded=a.loaded(specs['stable'],control['epoch']);assert loaded
  probe.admitted('stable',proc,raw);pending_dm=probe.pending['sha256'];channels=len(probe.hub.snapshot())
  worker('after-adopt',-signal.SIGKILL);assert count_apps()==3 and store.status()['committed']==loaded
  assert read(ROOT/'runtime'/authority.ANCHOR/'reservation.json')['binding'] is None
  worker('after-bind',-signal.SIGKILL);assert count_apps()==3
  value=read(ROOT/'runtime'/authority.ANCHOR/'reservation.json');assert value['binding']['loaded']==loaded
  assert j.digest(value['reservation'])==initial_hash and value['history']==[]
  stable=(sha(ROOT/'collection-journal/journal.json'),sha(ROOT/'runtime'/authority.ANCHOR/'reservation.json'))
  effects=len(probe.hub.requests);worker();worker()
  assert (sha(ROOT/'collection-journal/journal.json'),sha(ROOT/'runtime'/authority.ANCHOR/'reservation.json'))==stable
  assert len(probe.hub.requests)==effects and len(probe.hub.snapshot())==channels and count_apps()==3
  assert sha(ROOT/'state/omarchy-urbit-dms.qualification-stable/outbound-messages.json')==pending_dm
  assert probe.credentials()==probe.initial_credentials and sum(x['action']=='post' for x in probe.hub.requests)==probe.initial_logins
  assert len([x for x in probe.hub.pokes if x.get('uncertain') and x['owner']=='stable-dms'])==1
  gate=admission.Gate(ROOT/'gates'/control['epoch'])
  try:gate.stop_admission(control['epoch'],uuid.uuid4().hex)
  finally:gate.close()
  a.wait(lambda:all(row['host']['transport']['closed'] for row in a.snapshot()['loaded'].values()),75)
  a.release_guards(specs['stable'],read(a.bound(control['epoch'])));a.set_enabled(specs['stable'],False);a.wait(lambda:not a.snapshot()['loaded'])
  raw(proc,'artifactProbe','stop');assert proc.wait(5)==0;closure=proc.close_owned();probe.no_admission()
  assert sum(row['subscriptions'] for row in probe.hub.snapshot())==5
  report.update({'passed':True,'status':'complete','controller_kills':4,'application_creations':3,'subscriptions':5,'channels':probe.hub.snapshot(),
   'changed_epoch_refused':True,'changed_collection_refused':True,'original_reservation_preserved':True,'reservation_sha256':initial_hash,
   'completed_replay_no_http':True,'completed_replay_byte_identical':True,'unknown_dm_preserved':True,'no_new_logins':True,
   'credentials_unchanged':True,'credential_fingerprints':probe.initial_credentials,'original_guardian':closure,'loaded':loaded,
   'same_original_shell':True,'initial_owner_loss_recovery':False,'persistent_provisioning_recovery':False,'public_profile_command':False})
  print('PASS: same-shell initial activation survives four controller kills; exact reservation, single construction and retained application state')
 except BaseException:
  report.update({'status':'failed','failure':traceback.format_exc()});raise
 finally:
  save(ROOT/'native-initial-result.json',report)
  if proc and proc.closed is None:proc.close_owned()
  stop.set()
  if thread:thread.join(2)
  if store:store.close()
  probe.close()

if __name__=='__main__':controller() if '--controller' in sys.argv else run()
