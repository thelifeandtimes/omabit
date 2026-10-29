"""Characterize physical compositor replacement; no bootstrap/adoption shortcut."""
from pathlib import Path
import json,os,subprocess,sys,threading,time,traceback,uuid
sys.path.insert(0,'/lab/source')
import authority_provisioning as provisioning
import runtime_authority as authority
import journal as j
from native_shared_runtime import NativeAdapter,initialize,adopted
from native_connected_fixture import Probe
from journal_runtime_fixture import ROOT,guard,read,save,sha,identity
from supervised_shell import SupervisedShell


def run(display):
 guard();probe=Probe(ROOT);shells=[];store=None;thread=None;stop=threading.Event();report={'passed':False,'status':'starting'}
 def raw(proc,target,method,*args):
  q=subprocess.run(['/usr/bin/quickshell','ipc','--pid',str(proc.pid),'call','--',target,method,*args],capture_output=True,text=True,timeout=4)
  assert q.returncode==0 and q.stdout.strip()!='Target not found.';return q.stdout.strip()
 def count_apps():return sum((ROOT/('shell-'+str(i)+'.log')).read_text().count('APPLICATION_CREATED ') for i in range(len(shells)))
 def launch(stale=False):
  env=dict(os.environ,OMABIT_SUPERVISOR_QML_LOG='/lab/shell-'+str(len(shells))+'.log')
  if stale:env['GATE_INITIAL_CONFIG']=json.dumps({'version':1,'plugins':[{'id':m['plugin_id']} for m in specs['stable']['members'].values()],'bar':{'layout':{'left':[],'center':[],'right':[]}}})
  proc=SupervisedShell(env=env);shells.append(proc);row=identity(proc.pid);row.pop('args')
  save(ROOT/'coordinator-input.json',{'shell':row,'supervisor':proc.owner,'specs':specs})
  adapter=NativeAdapter();adapter.wait(lambda:subprocess.run(['/usr/bin/quickshell','ipc','--pid',str(proc.pid),'call','--','artifactProbe','state'],capture_output=True,text=True,timeout=4).stdout.strip().startswith('{'))
  adapter.wait(lambda:len(adapter.snapshot()['selected'])==9 and not adapter.snapshot()['scanning'])
  return proc,adapter
 def snapshot():
  files=[ROOT/'collection-journal/journal.json',ROOT/'collection-journal'/provisioning.RECORD,ROOT/'collection-journal'/authority.MIRROR,
   ROOT/'runtime'/authority.ANCHOR_SEAL,ROOT/'runtime'/provisioning.CLAIM]
  files.extend(p for p in (ROOT/'runtime'/authority.ANCHOR).rglob('*') if p.is_file())
  return {str(p.relative_to(ROOT)):{'inode':p.stat().st_ino,'sha256':sha(p)} for p in files}
 def refused(name,fn):
  before=snapshot()
  try:fn()
  except (OSError,ValueError) as error:report['refusals'].append({'operation':name,'error':str(error)})
  else:raise AssertionError('stale desktop authority unexpectedly admitted '+name)
  assert snapshot()==before
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
    members[role]={'plugin_id':d['plugin_id'],'profile':d['profile'],'artifact_sha256':row['artifact_sha256'],'paths':paths,'endpoint_ref':None if role=='omabit' else j.digest({'url':probe.hub.url,'ship':'~zod','profile':name,'role':role})}
   specs[name]={'format':1,'name':name,'members':members};j.specification(specs[name])
  proc,adapter=launch();store=j.Store(ROOT/'collection-journal',create=True);initialize(store)
  for spec in specs.values():
   with store.lease() as lease:store.record(lease,{'type':'register','spec':spec})
  save(ROOT/'coordinator-fault.json',{});epoch=uuid.uuid4().hex
  loaded=adapter.enable(specs['stable'],epoch,uuid.uuid4().hex);ready=adapter.healthy(specs['stable'],loaded)
  with store.lease() as lease:store.record(lease,{'type':'adopt','observation':ready})
  adopted(adapter,store);probe.admitted('stable',proc,raw);assert count_apps()==3
  old=read(ROOT/'runtime'/authority.ANCHOR/'authority.json');before=snapshot();pending=probe.pending['sha256']
  report.update({'old_context':old['context'],'authority':old['id'],'before':before,'refusals':[]})
  if (ROOT/'authority-inspection-mode.json').exists():
   from native_authority_inspection_fixture import observe
   value=observe('original-live');assert value['context']['live_context'] and not value['activations'][-1]['closure']['verified_process_closure']
  probe.hub.block_cleanup=True
  display.close();save(ROOT/'compositor-original-exit.json',read(ROOT/'compositor-exit.json'))
  report['original_shell_exit']=proc.wait(10);report['original_guardian']=proc.close_owned()
  # A complete process-tree closure is evidence of process death, not of remote
  # channel deletion. Capture both independently before starting a new compositor.
  deadline=time.monotonic()+8
  while any(x['active_streams'] for x in probe.hub.snapshot()) and time.monotonic()<deadline:time.sleep(.05)
  old_channels=probe.hub.snapshot();assert all(not x['active_streams'] for x in old_channels)
  assert any(not x['deleted'] for x in old_channels)
  assert snapshot()==before and count_apps()==3
  report['channels_after_original_closure']=old_channels
  if (ROOT/'authority-inspection-mode.json').exists():
   value=observe('original-closed');assert value['context']['server']=='gone' and value['activations'][-1]['closure']['verified_process_closure']
  refused('open-after-compositor-exit',lambda:authority.Authority(ROOT/'runtime'/authority.ANCHOR))
  os.environ.pop('WAYLAND_DISPLAY')
  (ROOT/'sway.log').rename(ROOT/'sway-original.log');(ROOT/'wayland-outputs.json').rename(ROOT/'wayland-original-outputs.json')
  display.start();new=authority.context(ROOT/'runtime'/os.environ['WAYLAND_DISPLAY'])
  assert new['server']!=old['context']['server']
  report['new_context']=new;report['socket_name_reused']=new['socket']['path']==old['context']['socket']['path']
  if (ROOT/'authority-inspection-mode.json').exists():
   value=observe('replacement-compositor');assert value['context']['server']=='gone' and not value['context']['live_context']
   assert value['descriptor']==old and value['activations'][-1]['closure']['verified_process_closure']
  refused('open-on-new-compositor',lambda:authority.Authority(ROOT/'runtime'/authority.ANCHOR))
  with store.lease() as lease:
   refused('resume-original-provisioning',lambda:provisioning.provision(ROOT/'runtime'/os.environ['WAYLAND_DISPLAY'],store,lease,read(store.root/provisioning.RECORD)['id']))
   refused('legacy-provision-existing-history',lambda:authority.Authority.provision(ROOT/'runtime'/os.environ['WAYLAND_DISPLAY'],store,lease))
  other=j.Store(ROOT/'other-coordinator',create=True)
  try:
   with other.lease() as lease:refused('new-coordinator-over-retained-anchor',lambda:provisioning.step(ROOT/'runtime'/os.environ['WAYLAND_DISPLAY'],other,lease,uuid.uuid4().hex))
   assert not (other.root/provisioning.RECORD).exists()
  finally:other.close()
  proc,adapter=launch(stale=True)
  adapter.wait(lambda:len(adapter.snapshot()['loaded'])==3 and all(x['host']['phase']=='fenced' and x['host']['loaded'] is None for x in adapter.snapshot()['loaded'].values()))
  assert not adapter.bindings() and count_apps()==3
  assert snapshot()==before and sha(ROOT/'state/omarchy-urbit-dms.qualification-stable/outbound-messages.json')==pending
  assert probe.credentials()==probe.initial_credentials and sum(x['action']=='post' for x in probe.hub.requests)==probe.initial_logins
  assert len([x for x in probe.hub.pokes if x.get('uncertain') and x['owner']=='stable-dms'])==1
  assert probe.hub.snapshot()==old_channels
  adapter.set_enabled(specs['stable'],False);adapter.wait(lambda:not adapter.snapshot()['loaded'])
  raw(proc,'artifactProbe','stop');assert proc.wait(5)==0;report['replacement_guardian']=proc.close_owned()
  if (ROOT/'session-cleanup-mode.json').exists():
   from native_session_cleanup_fixture import run as clean_desktop
   clean_desktop(probe,report['original_guardian'],report['replacement_guardian'],snapshot,count_apps)
  report.update({'passed':True,'status':'characterized','original_history_unchanged':True,'stale_selected_plugins_fenced':True,'application_creations':count_apps(),
   'unknown_dm_preserved':True,'credential_fingerprints':probe.initial_credentials,'credentials_unchanged':True,'no_new_logins':True,
   'original_transport_cleanup_unresolved':True,'old_channels_not_adopted':True,'same_socket_name_does_not_authorize_reuse':True,'public_cross_session_recovery':False,'final_snapshot':snapshot()})
  print('PASS: compositor replacement preserves/fences old authority; process closure does not erase unresolved remote channels')
 except BaseException:
  report.update({'status':'failed','failure':traceback.format_exc()});raise
 finally:
  save(ROOT/'compositor-loss-result.json',report)
  for proc in shells:
   if proc.closed is None:proc.close_owned()
  stop.set()
  if thread:thread.join(2)
  if store:store.close()
  probe.close()
