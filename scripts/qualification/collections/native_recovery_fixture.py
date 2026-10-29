"""Original-guardian recovery and resumed connected native profiles; lab only."""
from pathlib import Path
import json,os,signal,subprocess,sys,threading,time,traceback,uuid
sys.path.insert(0,'/lab/source')
import journal as j
import admission
from native_connected_fixture import Probe
from native_recovery_runtime import NativeAdapter
from journal_runtime_fixture import ROOT,guard,read,save,sha,identity,gone
from supervised_shell import SupervisedShell

def run(*,runtime=None,unbound_initial=False):
 if runtime is None:
  import native_recovery_runtime as runtime
 NativeAdapter=runtime.NativeAdapter
 admission=runtime.admission
 guard();probe=Probe(ROOT);guardians=[];store=None;thread=None;stop=threading.Event();report={'passed':False,'status':'starting'}
 def report_save():save(ROOT/'native-recovery-result.json',report)
 report_save()
 def check():probe.check()
 def worker(recover=False,expected=0):
  check();q=subprocess.run([sys.executable,'-B',runtime.__file__]+(['--recover'] if recover else []),capture_output=True,text=True,timeout=90)
  with (ROOT/'native-worker.log').open('a') as f:f.write(json.dumps({'recovery':recover,'exit':q.returncode})+'\n'+q.stdout+q.stderr)
  assert q.returncode==expected,(q.returncode,q.stderr[-1600:]);check();return q
 def raw(proc,target,method,*args):
  q=subprocess.run(['/usr/bin/quickshell','ipc','--pid',str(proc.pid),'call','--',target,method,*args],capture_output=True,text=True,timeout=4)
  assert q.returncode==0 and q.stdout.strip()!='Target not found.';return q.stdout.strip()
 def launch(initial=None):
  env=dict(os.environ,OMABIT_SUPERVISOR_QML_LOG='/lab/shell-'+str(len(guardians))+'.log')
  if initial:env['GATE_INITIAL_CONFIG']=json.dumps({'version':1,'plugins':[{'id':m['plugin_id']} for m in specs[initial]['members'].values()],'bar':{'layout':{'left':[],'center':[],'right':[]}}})
  proc=SupervisedShell(env=env);guardians.append(proc)
  row=identity(proc.pid);row.pop('args');save(ROOT/'coordinator-input.json',{'shell':row,'supervisor':proc.owner,'specs':specs})
  adapter=NativeAdapter()
  adapter.wait(lambda:subprocess.run(['/usr/bin/quickshell','ipc','--pid',str(proc.pid),'call','--','artifactProbe','state'],capture_output=True,text=True,timeout=4).stdout.strip().startswith('{'))
  adapter.wait(lambda:len(adapter.snapshot()['selected'])==9 and not adapter.snapshot()['scanning'])
  return proc,adapter
 def record(event):
  with store.lease() as lease:return store.record(lease,event)
 def pending():
  state=store.status();return state['requests'][state['pending']] if state['pending'] else None
 def begin(name):
  req=uuid.uuid4().hex;record({'type':'begin','request':req,'candidate':j.digest(specs[name]),'expected':j.digest(store.status()['committed'])});return req
 def finish(kill_enable=False,kill_disable=False):
  killed=False;disabled=False
  for _ in range(30):
   req=pending()
   if req is None:return
   assert not req['problem'],req['problem']
   if kill_enable and not killed and req['phase']=='enable-candidate':
    if hasattr(runtime,'before_enable'):runtime.before_enable(worker,count_apps)
    save(ROOT/'coordinator-fault.json',{'phase':req['phase'],'role':None});worker(expected=-signal.SIGKILL);killed=True
    original=read(ROOT/'lost-reply.json')['observation'];before=count_apps()
    save(ROOT/'coordinator-fault.json',{});worker();assert count_apps()==before and pending()['loaded']==original
    if hasattr(runtime,'after_enable'):runtime.after_enable()
    report['enable_lost_reply_no_duplicate_construction']=True
   elif kill_disable and not disabled and req['phase']=='disable-previous':
    save(ROOT/'coordinator-fault.json',{'phase':req['phase'],'role':None});worker(expected=-signal.SIGKILL);disabled=True
    assert not NativeAdapter().snapshot()['loaded'];before=count_apps()
    save(ROOT/'coordinator-fault.json',{});worker();assert count_apps()==before
    report['disable_lost_reply_no_duplicate_effect']=True
   else:worker()
  raise AssertionError('too many native coordinator phases')
 def count_apps():return sum((ROOT/('shell-'+str(i)+'.log')).read_text().count('APPLICATION_CREATED ') for i in range(len(guardians)))
 try:
  prepared=read(ROOT/'prepared.json');(ROOT/'gates').mkdir(mode=0o700);(ROOT/'runtime-observations').mkdir(mode=0o700)
  probe.begin(ROOT,prepared)
  def observe():
   while not stop.is_set():save(ROOT/'hub-observation.json',{'url':probe.hub.url,'channels':probe.hub.snapshot()});stop.wait(.05)
  thread=threading.Thread(target=observe,daemon=True);thread.start()
  specs={}
  for name,items in prepared.items():
   members={}
   for role,row in items.items():
    d=row['1']['descriptor'];paths={k:d['environment']['XDG_'+k.upper()+'_HOME'] if role=='omabit' else str(ROOT/k/('omarchy-urbit-'+role+'.'+d['profile'])) for k in j.PATHS}
    members[role]={'plugin_id':d['plugin_id'],'profile':d['profile'],'artifact_sha256':row['1']['artifact_sha256'],'paths':paths,'endpoint_ref':None if role=='omabit' else j.digest({'url':probe.hub.url,'ship':'~zod','profile':name,'role':role})}
   specs[name]={'format':1,'name':name,'members':members};j.specification(specs[name])
  proc,a=launch();store=j.Store(ROOT/'collection-journal',create=True)
  if hasattr(runtime,'initialize'):runtime.initialize(store)
  for spec in specs.values():record({'type':'register','spec':spec})
  save(ROOT/'coordinator-fault.json',{})
  epoch=uuid.uuid4().hex
  if unbound_initial:
   assert getattr(runtime,'SHARED',False)
   save(ROOT/'initial-control.json',{'collection':'stable','epoch':epoch,'request':uuid.uuid4().hex,'fault':'after-adopt'})
   q=subprocess.run([sys.executable,'-B','/lab/native_initial_fixture.py','--controller'],capture_output=True,text=True,timeout=90)
   with (ROOT/'native-worker.log').open('a') as f:f.write(json.dumps({'initial_controller':True,'exit':q.returncode})+'\n'+q.stdout+q.stderr)
   assert q.returncode==-signal.SIGKILL,(q.returncode,q.stderr[-1600:])
   loaded=store.status()['committed'];assert loaded and read(a.bound(epoch))['loaded']==loaded
   assert read(ROOT/'runtime'/runtime.authority.ANCHOR/'reservation.json')['binding'] is None
  else:
   loaded=a.enable(specs['stable'],epoch,uuid.uuid4().hex);ready=a.healthy(specs['stable'],loaded);record({'type':'adopt','observation':ready})
   if hasattr(runtime,'adopted'):runtime.adopted(a,store)
  probe.admitted('stable',proc,raw);before_dm=probe.pending['sha256'];assert count_apps()==3
  if hasattr(runtime,'before_transition'):runtime.before_transition(a,store,probe,count_apps)
  req=begin('dev');probe.hub.block_cleanup=True
  # Close all new native admissions before losing the shell; transport closure
  # remains a separate fact and failed deletes must keep its recovery fence.
  gate=admission.Gate(ROOT/'gates'/epoch)
  try:gate.stop_admission(epoch,j.effect(pending(),'drain-previous'))
  finally:gate.close()
  old=proc;old.kill();assert old.wait(5)==-signal.SIGKILL
  worker();assert pending()['uncertain']==[{'phase':'drain-previous','reason':'reader-owner-lost'}]
  fenced=sha(ROOT/'collection-journal/journal.json')
  proc,a=launch('stable')
  a.wait(lambda:len(a.snapshot()['loaded'])==3 and all(v['host']['phase']=='fenced' and v['host']['loaded'] is None for v in a.snapshot()['loaded'].values()))
  assert not a.bindings() and count_apps()==3
  q=worker(expected=1);assert 'unconfirmed helper lineage prevents runtime effects' in q.stderr and sha(ROOT/'collection-journal/journal.json')==fenced
  closure=old.close_owned();assert closure['owner']!=proc.owner and closure['root']['pid']!=proc.pid
  worker(expected=1);assert sha(ROOT/'collection-journal/journal.json')==fenced and count_apps()==3
  recovery=uuid.uuid4().hex;control={'request':req,'recovery':recovery,'closure':closure,'fault':''}
  def recovery_step(fault='',expected=0,wrong=False):
   value=dict(control,fault=fault)
   if wrong:value['closure']={'owner':proc.owner,'root':proc.root,'request':uuid.uuid4().hex,'receipt':str(proc.path),'receipt_sha256':sha(proc.path)}
   save(ROOT/'recovery-control.json',value);return worker(True,expected)
  recovery_step(expected=1,wrong=True);assert sha(ROOT/'collection-journal/journal.json')==fenced
  recovery_step('after-prepared',-signal.SIGKILL);prepared_hash=sha(ROOT/'collection-journal/journal.json');assert pending()['recovery']['verified'] is None
  recovery_step(expected=1);assert sha(ROOT/'collection-journal/journal.json')==prepared_hash and pending()['uncertain']
  assert len([x for x in probe.hub.snapshot() if not x['deleted']])>=2
  probe.hub.block_cleanup=False
  recovery_step('after-cleanup',-signal.SIGKILL);assert sha(ROOT/'collection-journal/journal.json')==prepared_hash and pending()['uncertain']
  probe.no_admission();effects=len(probe.hub.requests)
  if unbound_initial:
   recovery_step('after-binding',expected=-signal.SIGKILL)
   binding=read(ROOT/'runtime'/runtime.authority.ANCHOR/'reservation.json')['binding']
   assert binding and binding['loaded']==store.status()['committed'] and binding['guardian']['owner']['token']==closure['owner']['token']
   save(ROOT/'closed-initial-binding.json',{'binding':binding,'request':req,'authority_sha256':sha(ROOT/'runtime'/runtime.authority.ANCHOR/'reservation.json')})
  recovery_step();assert pending()['phase']=='disable-previous' and not pending()['uncertain'] and len(probe.hub.requests)==effects
  if unbound_initial:assert sha(ROOT/'runtime'/runtime.authority.ANCHOR/'reservation.json')==read(ROOT/'closed-initial-binding.json')['authority_sha256']
  verified=sha(ROOT/'collection-journal/journal.json');recovery_step();assert sha(ROOT/'collection-journal/journal.json')==verified and len(probe.hub.requests)==effects
  assert sha(ROOT/'state/omarchy-urbit-dms.qualification-stable/outbound-messages.json')==before_dm
  report.update({'stale_enabled_replacement_inert':True,'wrong_guardian_refused':True,'original_guardian_closure':closure,'coordinator_kills':2,'failed_cleanup_retains_fence':True,'verified_retry_no_http':True})
  finish(kill_enable=True);assert store.status()['requests'][req]['outcome']=='committed';a=NativeAdapter();probe.admitted('dev',proc,raw)
  returned=begin('stable');finish(kill_disable=True);assert store.status()['requests'][returned]['outcome']=='committed';probe.admitted('stable',proc,raw);assert count_apps()==9
  # Last close is an explicit qualification teardown, not a fabricated committed
  # journal transition. Its retained journal reports the last successful profile.
  committed=store.status()['committed'];gate=admission.Gate(ROOT/'gates'/committed['epoch'])
  try:gate.stop_admission(committed['epoch'],uuid.uuid4().hex)
  finally:gate.close()
  a.wait(lambda:all(x['host']['transport']['closed'] for x in a.snapshot()['loaded'].values()),75)
  a.release_guards(specs['stable'],read(a.bound(committed['epoch'])))
  a.set_enabled(specs['stable'],False);a.wait(lambda:not a.snapshot()['loaded']);raw(proc,'artifactProbe','stop');assert proc.wait(5)==0;proc.close_owned();probe.finish(unclean_recovery=True,coordinator_recovery_integrated=True)
  for i in range(len(guardians)):
   log=(ROOT/('shell-'+str(i)+'.log')).read_text();assert not any(x in log for x in ('TypeError','ReferenceError','Binding loop','Unable to assign','Cannot assign','failed to load','service plugin load failed','WIDGET_BAD_ID','APPLICATION_BAD_ID'))
  report.update({'passed':True,'status':'complete','application_creations':count_apps(),'positive_subscriptions':sum(x['subscriptions'] for x in probe.hub.snapshot()),'connected_sessions_unchanged':True,'unresolved_dm_byte_identical':True,'guardians':[p.closed for p in guardians],'full_native_shell':False,'production_adapter':False,'cross_root_runtime_admission':getattr(runtime,'SHARED',False),'recovery_coordinator_kills':2,'coordinator_kills':7 if unbound_initial else 5 if getattr(runtime,'SHARED',False) else 4,'initial_binding_reconciled':unbound_initial,'closed_binding_controller_kills':1 if unbound_initial else 0})
  print('PASS: native stale admission fence, original-guardian recovery, lost-reply replay and resumed subscriptions')
 except BaseException:
  report.update({'passed':False,'status':'failed','failure':traceback.format_exc()})
  try:report['runtime']=NativeAdapter().snapshot()
  except Exception:pass
  raise
 finally:
  report_save()
  for p in guardians:
   if p.closed is None:p.close_owned()
  stop.set()
  if thread:thread.join(2)
  if store:store.close()
  probe.close()
