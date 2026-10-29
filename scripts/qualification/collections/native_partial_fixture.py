"""Private native rollback before a complete candidate loaded receipt.

Characterized startup cohorts use actual transport cleanup and original-owner
recovery. No production candidate-recovery command exists.
"""
from pathlib import Path
import json,os,signal,subprocess,sys,threading,time,traceback,uuid
sys.path.insert(0,'/lab/source')
import journal as j
import shared_admission as admission
import runtime_authority as authority
from native_connected_fixture import Probe
from native_shared_runtime import NativeAdapter,initialize,adopted
from journal_runtime_fixture import ROOT,guard,read,save,sha,identity
from supervised_shell import SupervisedShell

def run():
 guard();case=read(ROOT/'partial-candidate-mode.json')['case'];assert case in ('before-issue','dms-only','dms-cleanup-refused','theme-cleanup-refused','all-cleanup-refused');recovering=case.endswith('cleanup-refused');replacement_started=False
 roles=() if case=='before-issue' else ('theme',) if case=='theme-cleanup-refused' else j.ROLES if case=='all-cleanup-refused' else ('dms',)
 owners=['dev-'+r for r in roles if r!='omabit'];app_count=3+len(roles);expected_subscriptions=10+sum({'omabit':0,'theme':1,'dms':4}[r] for r in roles)
 probe=Probe(ROOT);guardians=[];store=None;thread=None;stop=threading.Event();report={'passed':False,'case':case,'status':'starting'}
 def report_save():save(ROOT/'native-partial-result.json',report)
 report_save()
 def raw(proc,target,method,*args):
  q=subprocess.run(['/usr/bin/quickshell','ipc','--pid',str(proc.pid),'call','--',target,method,*args],capture_output=True,text=True,timeout=4)
  assert q.returncode==0 and q.stdout.strip()!='Target not found.',q.stderr;return q.stdout.strip()
 def launch():
  proc=SupervisedShell(env=dict(os.environ,OMABIT_SUPERVISOR_QML_LOG='/lab/shell-'+str(len(guardians))+'.log'));guardians.append(proc)
  row=identity(proc.pid);row.pop('args');save(ROOT/'coordinator-input.json',{'shell':row,'supervisor':proc.owner,'specs':specs})
  a=NativeAdapter()
  a.wait(lambda:subprocess.run(['/usr/bin/quickshell','ipc','--pid',str(proc.pid),'call','--','artifactProbe','state'],capture_output=True,text=True,timeout=4).stdout.strip().startswith('{'))
  a.wait(lambda:len(a.snapshot()['selected'])==9 and not a.snapshot()['scanning'])
  return proc,a
 def record(event):
  with store.lease() as lease:return store.record(lease,event)
 def pending():
  state=store.status();return state['requests'][state['pending']] if state['pending'] else None
 def event(kind,**fields):
  req=pending();return {'type':kind,'request':req['id'],'phase':req['phase'],'effect':j.effect(req,req['phase']),**fields}
 def worker(expected=0,recover=False):
  probe.check();q=subprocess.run([sys.executable,'-B','/lab/native_shared_runtime.py']+(['--recover-candidate'] if recover else []),capture_output=True,text=True,timeout=90)
  with (ROOT/'native-worker.log').open('a') as f:f.write(json.dumps({'exit':q.returncode})+'\n'+q.stdout+q.stderr)
  assert q.returncode==expected,(q.returncode,q.stderr[-2200:]);probe.check()
 def count_apps():return sum((ROOT/('shell-'+str(i)+'.log')).read_text().count('APPLICATION_CREATED ') for i in range(len(guardians)))
 def state_files(spec,role):
  base=Path(spec['members'][role]['paths']['state']);result={}
  if base.exists():
   for path in sorted(base.rglob('*')):
    assert not path.is_symlink()
    if path.is_file():result[str(path.relative_to(base))]=sha(path)
  return result
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
  save(ROOT/'coordinator-fault.json',{})
  epoch=uuid.uuid4().hex;loaded=a.enable(specs['stable'],epoch,uuid.uuid4().hex)
  record({'type':'adopt','observation':a.healthy(specs['stable'],loaded)});adopted(a,store);probe.admitted('stable',proc,raw)
  pending_dm=probe.pending['sha256'];candidate_before={r:state_files(specs['dev'],r) for r in j.ROLES}
  request=uuid.uuid4().hex;record({'type':'begin','request':request,'candidate':j.digest(specs['dev']),'expected':j.digest(store.status()['committed'])})
  for _ in range(12):
   if pending()['phase']=='enable-candidate':break
   worker()
  assert pending()['phase']=='enable-candidate' and not a.snapshot()['loaded'];probe.no_admission();assert count_apps()==3
  save(ROOT/'coordinator-fault.json',{'phase':'after-handoff' if not roles else 'partial-all' if len(roles)==3 else 'partial-'+roles[0],'role':None})
  worker(-signal.SIGKILL);save(ROOT/'coordinator-fault.json',{})
  reservation=read(ROOT/'runtime'/authority.ANCHOR/'reservation.json');assert reservation['binding'] is None
  candidate=reservation['reservation'];gate=Path(candidate['gate']['path']);assert candidate['permit']['epoch']==j.effect(pending(),'enable-candidate')
  assert candidate['owner']['guardian']['owner']['token']==proc.owner['token']
  bindings={}
  if case=='before-issue':
   assert not list(gate.iterdir()) and not a.snapshot()['loaded'] and count_apps()==3
  else:
   partial=read(ROOT/'partial-loaded.json');assert partial['reservation']==j.digest(candidate)
   assert set(partial['bindings'])==set(roles) and count_apps()==app_count
   bindings=partial['bindings'];assert set(read(gate/'admission.json')['claims'])==set(roles)
   if 'theme' in roles:assert partial['appearance_applied']
  record(event('rollback',reason='loader-unavailable'));assert pending()['phase']=='drain-candidate' and pending()['loaded'] is None
  if recovering:probe.hub.blocked_cleanup_owners=owners
  original=proc;original.kill();assert original.wait(5)==-signal.SIGKILL
  # EOF cleanup is independently observed before the guardian closes the remaining
  # descendants. A vanished process never substitutes for remote channel closure.
  if not recovering:probe.no_admission()
  closure=original.close_owned();proof=read(original.path)
  assert proof['proof']=='waitpid-echild' and sha(original.path)==closure['receipt_sha256']
  assert proof['owner']['token']==candidate['owner']['guardian']['owner']['token']
  assert {k:proof['root'][k] for k in ('pid','start')}==candidate['owner']['guardian']['root']
  if recovering:
   record(event('problem',reason='reader-owner-lost'));fenced=sha(ROOT/'collection-journal/journal.json')
   assert all(any(row['owner']==owner and not row['deleted'] for row in probe.hub.snapshot()) for owner in owners)
   proc,a=launch();replacement_started=True;worker(expected=1)
   assert sha(ROOT/'collection-journal/journal.json')==fenced
   control={'request':request,'recovery':uuid.uuid4().hex,'closure':closure,'fault':''}
   def recover(fault='',expected=0,wrong=False):
    value=dict(control,fault=fault)
    if wrong:value['closure']={'owner':proc.owner,'root':proc.root,'request':uuid.uuid4().hex,'receipt':str(proc.path),'receipt_sha256':sha(proc.path)}
    save(ROOT/'recovery-control.json',value);worker(expected,recover=True)
   recover(expected=1,wrong=True);assert sha(ROOT/'collection-journal/journal.json')==fenced
   recover('after-prepared',-signal.SIGKILL);prepared_hash=sha(ROOT/'collection-journal/journal.json')
   assert pending()['candidate_recovery']['verified'] is None and pending()['uncertain']
   refused_before=sum(bool(row.get('refused')) for row in probe.hub.requests)
   recover(expected=1);assert sha(ROOT/'collection-journal/journal.json')==prepared_hash
   assert sum(bool(row.get('refused')) for row in probe.hub.requests)>refused_before
   assert pending()['uncertain'] and not a.snapshot()['loaded'] and count_apps()==app_count
   probe.hub.blocked_cleanup_owners=[]
   recover('after-cleanup',-signal.SIGKILL);effects=len(probe.hub.requests)
   assert sha(ROOT/'collection-journal/journal.json')==prepared_hash and pending()['uncertain']
   recover();assert pending()['phase']=='disable-candidate' and not pending()['uncertain']
   assert len(probe.hub.requests)==effects
   # Driver replay remains effect-free after candidate phase advancement.
   with store.lease() as lease:
    import coordinator as coordinator
    before=sha(ROOT/'collection-journal/journal.json')
    coordinator.recover_candidate_step(store,lease,object(),request,control['recovery'])
    assert sha(ROOT/'collection-journal/journal.json')==before
   report.update({'wrong_guardian_refused':True,'failed_cleanup_kept_fence':True,'recovery_controller_kills':2,'verified_retry_no_http':True,'cleanup_refusals':[row for row in probe.hub.requests if row.get('refused')]})
  assert all(x['deleted'] and not x['active_streams'] for x in probe.hub.snapshot())
  for role in roles:
   if role=='omabit':continue
   transport=read(Path(specs['dev']['members'][role]['paths']['state'])/'transport/channels.json')
   assert transport['channels'] and all(row['phase']=='closed' for row in transport['channels'])
   assert sum(row['subscriptions'] for row in probe.hub.snapshot() if row['owner']=='dev-'+role)==(4 if role=='dms' else 1)
  for role in j.ROLES:
   retained=state_files(specs['dev'],role)
   if role not in bindings:assert retained==candidate_before[role]
   if not recovering:record(event('drained',role=role,observation={'epoch':candidate['permit']['epoch'],'binding':bindings.get(role),
    'absent':role not in bindings,'closed':True,'retained_state':j.digest(retained),'pending_ids':[]}))
  assert pending()['phase']=='disable-candidate'
  record(event('completed',observation={'epoch':candidate['permit']['epoch'],'absent':True,'helpers_exited':True}))
  assert pending()['phase']=='restore-appearance'
  retained_gate={p.name:sha(p) for p in gate.iterdir() if p.is_file()}
  if not replacement_started:proc,a=launch()
  for _ in range(8):
   if pending() is None:break
   worker()
  state=store.status();assert state['pending'] is None and state['requests'][request]['outcome']=='rolled-back'
  assert {p.name:sha(p) for p in gate.iterdir() if p.is_file()}==retained_gate
  reservation=read(ROOT/'runtime'/authority.ANCHOR/'reservation.json')
  assert len(reservation['history'])==2 and reservation['history'][-1]['handoff']['mode']=='partial-rollback'
  assert reservation['history'][-1]['handoff']['closure']['sha256']==closure['receipt_sha256']
  assert reservation['binding']['loaded']==state['committed']
  probe.admitted('stable',proc,raw)
  assert sha(ROOT/'state/omarchy-urbit-dms.qualification-stable/outbound-messages.json')==pending_dm
  assert count_apps()==app_count+3
  committed=state['committed'];g=admission.Gate(ROOT/'gates'/committed['epoch'])
  try:g.stop_admission(committed['epoch'],uuid.uuid4().hex)
  finally:g.close()
  a.wait(lambda:all(row['host']['transport']['closed'] for row in a.snapshot()['loaded'].values()),75)
  a.release_guards(specs['stable'],read(a.bound(committed['epoch'])));a.set_enabled(specs['stable'],False);a.wait(lambda:not a.snapshot()['loaded'])
  raw(proc,'artifactProbe','stop');assert proc.wait(5)==0;proc.close_owned();probe.no_admission()
  assert probe.credentials()==probe.initial_credentials and sum(x['action']=='post' for x in probe.hub.requests)==probe.initial_logins
  assert len([x for x in probe.hub.pokes if x.get('uncertain') and x['owner']=='stable-dms'])==1
  subscriptions=sum(x['subscriptions'] for x in probe.hub.snapshot());assert subscriptions==expected_subscriptions
  report.update({'passed':True,'status':'complete','application_creations':count_apps(),'subscriptions':subscriptions,
   'channels':probe.hub.snapshot(),'unknown_dm_preserved':True,'no_new_logins':True,'credentials_unchanged':True,
   'credential_fingerprints':probe.initial_credentials,'original_guardian':closure,'guardians':[p.closed for p in guardians],
   'candidate_bindings':bindings,'candidate_appearance_applied':'theme' in roles,'stable_appearance_restored':True,'authority_history':reservation['history'],'abandoned_gate_unchanged':True,
   'candidate_transport_cleanup':'durable-original-owner-recovery' if recovering else 'observed-existing-helper-EOF' if bindings else 'no-candidate-constructor-admitted',
   'partial_native_recovery_api':False,'full_native_shell':False,'synthetic_eyre':True})
  print('PASS: actual native partial rollback '+case+'; original guardian, retained gate and connected stable state verified')
 except BaseException:
  report.update({'status':'failed','failure':traceback.format_exc()});raise
 finally:
  report_save()
  for p in guardians:
   if p.closed is None:p.close_owned()
  stop.set()
  if thread:thread.join(2)
  if store:store.close()
  probe.close()
