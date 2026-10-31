#!/usr/bin/python3
"""Offline published-binary upgrade check, only in the guarded release guest."""
import hashlib,json,os,sqlite3,subprocess,sys,time,shutil,base64
from pathlib import Path
I=Path('/opt/omabit-release-input'); O=Path('/opt/omabit-release-results')
SOCK='/run/user/1000/urbit-runner/runner.sock'
VERSIONS=('0.3.3','0.3.4')
report={'format':1,'passed':False,'steps':[]}
created=False

def run(args,owner=False,timeout=90,ok=True):
 args=[str(a) for a in args]
 if owner:args=['runuser','-u','review','--','env','HOME=/home/review','XDG_RUNTIME_DIR=/run/user/1000','DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/1000/bus','PATH=/usr/bin:/bin',*args]
 if 'snapshot' not in args and 'hello' not in args:
  print('UPGRADE_STEP '+json.dumps(args),flush=True)
 r=subprocess.run(args,capture_output=True,text=True,timeout=timeout)
 with (O/'commands.jsonl').open('a') as f:f.write(json.dumps({'args':args,'exit':r.returncode,'out':r.stdout[-16000:],'err':r.stderr[-16000:]})+'\n')
 if ok and r.returncode:raise RuntimeError('Command failed: '+' '.join(args)+'\n'+r.stderr[-1000:])
 return r

def tool(v,n):return I/v/(n+'-linux-amd64')
def collector(v,*args):return json.loads(run([tool(v,'urbit-collector'),'--socket',SOCK,*args],True).stdout)
def wait(fn,seconds=180):
 end=time.monotonic()+seconds
 while time.monotonic()<end:
  try:
   val=fn()
   if val:return val
  except (RuntimeError,ValueError,KeyError):pass
  time.sleep(1)
 raise RuntimeError('readiness deadline exceeded')
def ready(v,name,state):
 row=next(i for i in collector(v,'snapshot')['instances'] if i['id']==name)
 if row.get('active_operation_id'):return None
 if row['observed_state']!=state:return None
 if state=='running' and not row.get('ship'):return None
 return row
def operation(v,name,action,request):
 row=next(i for i in collector(v,'snapshot')['instances'] if i['id']==name)
 collector(v,'operate',name,action,'--revision',str(row['config_revision']),'--request-id',request)
 def terminal():
  op=next((x for x in collector(v,'snapshot')['operations'] if x['request_id']==request),None)
  if op and op['status'] not in ('queued','pending','running'):
   if op['status']!='succeeded':raise AssertionError('operation did not succeed: '+str(op))
   return op
 result=wait(terminal,600)
 return result

def project(v):
 s=collector(v,'snapshot')
 return {'runner_id':s['runner_id'],'instances':[{k:r.get(k) for k in ('id','ship','mode','label','config','config_revision','applied_revision','desired_state','pier_path','bootstrap')} for r in s['instances']], 'operations':s['operations']}
def runtime(v):
 s=collector(v,'snapshot')
 ids=[r['runtime_status']['container_id'] for r in s['instances'] if r['observed_state']=='running']
 return {'engine':run(['systemctl','--user','show','urbit-docker.service','-p','MainPID,InvocationID'],True).stdout,
 'containers':run(['docker','--host','unix:///run/user/1000/urbit-docker.sock','inspect','--format','{{.Id}} {{.State.Pid}} {{.State.StartedAt}}',*ids],True).stdout if ids else ''}
def db_projection():
 with sqlite3.connect('file:/home/review/.local/share/urbit-runner/managed/runner.sqlite?mode=ro',uri=True) as db:
  return {'schema':db.execute('pragma user_version').fetchone()[0], 'dump':hashlib.sha256('\n'.join(db.iterdump()).encode()).hexdigest()}
def install(v):
 result=run([tool(v,'urbit-runner'),'install-host','--owner','review','--bundle',I/v/'rootless-bundle','--public-key','/home/review/release-operator.pub','--acknowledge-host-changes'],timeout=600)
 wait(lambda:collector(v,'hello'))
 return next(json.loads(x)['host'] for x in result.stdout.splitlines() if json.loads(x).get('host'))

def main():
 global created
 if sys.argv[1:]:raise RuntimeError('No arguments accepted')
 assert os.geteuid()==0 and Path('/sys/class/dmi/id/sys_vendor').read_text().strip()=='QEMU'
 assert 'omabit.disposable-guest=release-upgrade-v1' in Path('/proc/cmdline').read_text().split()
 assert not O.exists(),'Never reset an existing test'
 O.mkdir(mode=0o700)
 created=True
 for p in ('/etc/urbit-runner','/srv/urbit/piers','/home/review/.local/share/urbit-runner'):
  assert not Path(p).exists(), 'not pristine: '+p
 assert run(['pgrep','-x','dockerd'],ok=False).returncode==1
 for version in VERSIONS:
  meta=json.loads((I/version/'release.json').read_text());assert meta['version']==version
  checks={line.split('  ',1)[1]:line.split('  ',1)[0] for line in (I/version/'SHA256SUMS').read_text().splitlines()}
  for name in ('omabit','urbit-collector','urbit-runner'):
   assert hashlib.sha256(tool(version,name).read_bytes()).hexdigest()==checks[name+'-linux-amd64']
 report['sources']={v:json.loads((I/v/'release.json').read_text())['source_commit'] for v in VERSIONS}
 run(['loginctl','enable-linger','review'])
 run(['systemctl','start','user@1000.service'])
 scenario=json.loads((I/'scenario.json').read_text());report['starting_versions']=scenario
 setup=Path('/home/review/.config/omabit/setup')
 setup.mkdir(parents=True,mode=0o700)
 run(['chown','-R','review:review','/home/review/.config'])
 run(['ssh-keygen','-q','-t','ed25519','-N','','-f',setup/'id_ed25519'],True)
 shutil.copy2(setup/'id_ed25519.pub',Path('/home/review/release-operator.pub'))
 host=install('0.3.3')
 # Reproduce the published setup receipt format from actual installer evidence.
 # Initial setup/download itself is outside this upgrade rehearsal.
 contents={
  'operator.pub':(setup/'id_ed25519.pub').read_text(),
  'known_hosts':'[127.0.0.1]:22222 '+host['host_public_key']+'\n',
  'ssh_config':f'Host omabit-local\n  HostName 127.0.0.1\n  Port 22222\n  User review\n  IdentityFile "{setup}/id_ed25519"\n  IdentityAgent none\n  IdentitiesOnly yes\n  UserKnownHostsFile "{setup}/known_hosts"\n  GlobalKnownHostsFile /dev/null\n  StrictHostKeyChecking yes\n  BatchMode yes\n',
  'ready.json':json.dumps({'runner_id':host['runner_id'],'connection':'local','host_public_key':host['host_public_key']})}
 for name,data in contents.items():
  path=setup/name;path.write_text(data);path.chmod(0o600);os.chown(path,1000,1000)
 run([tool('0.3.3','urbit-collector'),'enroll','local','--ssh','omabit-local','--ssh-config',setup/'ssh_config'],True)
 old_status=json.loads(run([tool(scenario['desktop'],'omabit'),'setup-status'],True).stdout)
 assert old_status['ready'] and old_status['connection']['runner_id']==host['runner_id']
 report['receipt_fixture']='published format reconstructed from genuine 0.3.3 installer result and 0.3.3 SSH enrollment; old desktop setup-status verified ready'
 def access_hashes():
  return {name:hashlib.sha256((setup/name).read_bytes()).hexdigest() for name in contents.keys()|{'id_ed25519'}}
 original_access=access_hashes()
 run(['docker','--host','unix:///run/user/1000/urbit-docker.sock','load','--input',I/'runtime.oci.tar'],True)
 for name,ship,port in (('upgrade-zod','~zod',19180),('upgrade-nec','~nec',19181)):
  collector('0.3.3','create-fake',name,'--ship',ship,'--http-port',str(port),'--request-id',name+'-create','--bootstrap','recommended','--pill',str(I/'boot.pill'))
  operation('0.3.3',name,'start',name+'-start')
  wait(lambda:ready('0.3.3',name,'running'),600)
  if name=='upgrade-nec':operation('0.3.3',name,'stop',name+'-stop');wait(lambda:ready('0.3.3',name,'stopped'))
 # A stopped fake created only by this offline guest supplies real Vere state.
 # Its separate test copy receives synthetic macOS metadata, never user data.
 nec=next(x for x in collector('0.3.3','snapshot')['instances'] if x['id']=='upgrade-nec')
 import_source=Path('/home/review/macos-import-source');assert not import_source.exists()
 run(['cp','-a','--reflink=auto',Path('/srv/urbit/piers')/nec['pier_path'],import_source],True,timeout=180)
 epoch=next(x.name for x in (import_source/'.urb/log').iterdir() if x.is_dir() and x.name.startswith('0i'))
 companions={}
 for name in ['.urb/log/._0i391592','.urb/log/._data.mdb','.urb/log/._lock.mdb','.urb/log/.DS_Store','.urb/log/'+epoch+'/._data.mdb','.urb/log/'+epoch+'/.DS_Store','.urb/chk/.DS_Store']:
  data=b'\x00\x05\x16\x07\x00\x02\x00\x00synthetic metadata: '+name.encode()
  path=import_source/name;path.write_bytes(data);path.chmod(0o600);os.chown(path,1000,1000)
  companions[name]=hashlib.sha256(data).hexdigest()
 def storage(v,*args,ok=True):
  return run([tool(v,'urbit-collector'),'--runner','local','--expected-runner-id',host['runner_id'],*args],True,ok=ok)
 old=storage('0.3.3','preview-pier','--source',import_source,'--storage-root','default',ok=False)
 assert old.returncode and 'unrecognized epoch-log entry' in old.stderr,old.stderr
 report['macos_import']={'previous_refused_metadata':True,'synthetic_companion_sha256':companions}
 before=project('0.3.3'); runtime_before=runtime('0.3.3')
 report['baseline']=before;report['runtime_before']=runtime_before;report['schema_before']=db_projection()['schema']
 # Import and verify actual release bundles with the native desktop loader.
 staged=json.loads(run(['python3','-B',I/'desktop.py',scenario['desktop']],True).stdout)
 report['desktop_transition']=staged
 assert before==project('0.3.3') and runtime_before==runtime('0.3.3'),'desktop activation changed ships'
 candidate=Path(staged['directory']);ui=Path('/home/review/upgrade-ui');ui.mkdir(mode=0o700)
 shutil.copytree(candidate/'ui',ui/'Plugin')
 shutil.copy2(I/'shell.qml',ui/'shell.qml')
 source=(ui/'shell.qml').read_text().replace('@HELPER@',str(candidate/'bin/omabit'))
 (ui/'shell.qml').write_text(source)
 commons=ui/'Commons';commons.mkdir()
 (commons/'qmldir').write_text('singleton Style 1.0 Style.qml\nsingleton Color 1.0 Color.qml\n')
 (commons/'Style.qml').write_text('pragma Singleton\nimport QtQuick\nQtObject { function space(n) {return n} property var font: ({family:"sans-serif",caption:12,bodySmall:14,body:14}) }\n')
 (commons/'Color.qml').write_text('pragma Singleton\nimport QtQuick\nQtObject {property color background:"#171a1e";property color foreground:"#e7e9eb";property color accent:"#8ab4f8";property color urgent:"#ed8994"}\n')
 run(['chown','-R','review:review',ui])
 # Guest-only authorization fixture: real pkexec/polkit denies once, then grants
 # this precise retained installer. No host policy or service is available here.
 rule=Path('/etc/polkit-1/rules.d/00-omabit-rehearsal.rules')
 def authorize(result):
  rule.write_text('polkit.addRule(function(action, subject) { if (action.id === "org.freedesktop.policykit.exec" && subject.user === "review" && action.lookup("program") === "/home/review/.config/omabit/setup/service-upgrade/urbit-runner") return polkit.Result.'+result+'; });\n')
  time.sleep(2)
 authorize('NO')
 qlog=(O/'qml.log').open('w')
 proc=subprocess.Popen(['runuser','-u','review','--','env','HOME=/home/review','PATH=/usr/bin:/bin','XDG_RUNTIME_DIR=/run/user/1000','DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/1000/bus','QT_QPA_PLATFORM=offscreen','QT_QUICK_BACKEND=software','QSG_RHI_BACKEND=software','QT_QUICK_CONTROLS_STYLE=Basic','QML_DISABLE_DISK_CACHE=1','quickshell','--no-color','-p',str(ui/'shell.qml')],stdout=qlog,stderr=subprocess.STDOUT)
 def capture(name):
  path=ui/'review.png';path.unlink(missing_ok=True)
  control('capture');wait(lambda:path.exists(),10)
  data=path.read_bytes();assert len(data)<200000
  print('OMABIT_GUIDED_IMAGE '+json.dumps({'name':name,'png':base64.b64encode(data).decode()}),flush=True)
 def settled(need='status'):
  x=control('state')
  return x if not x['busy'] and (not need or x.get(need)) else None
 def control(method):
  response=run(['quickshell','ipc','-p',ui/'shell.qml','call','upgradeReview',method],True,timeout=10)
  return json.loads(response.stdout)
 try:
  wait(lambda:control('state'))
  assert control('state')['commands']==0,'passive card dispatched a helper'
  control('pressCheck');state=wait(lambda:settled())
  assert state['status']['available'] and not state['status']['pending']
  target=state['status']['target_sha256'];report['offered']=state['status'];capture('offer')
  control('pressUpdate');state=wait(lambda:settled('error'))
  assert before==project('0.3.3') and runtime_before==runtime('0.3.3') and original_access==access_hashes()
  report['denied_authorization']={'error':state['error'],'preserved':True}
  attempt=json.loads((setup/'service-upgrade.json').read_text());assert not attempt['complete'] and attempt['target_sha256']==target
  control('pressCheck');state=wait(lambda:settled())
  assert state['apply_text']=='Retry service update' and state['status']['pending'] and state['status']['target_sha256']==target
  capture('retry')
  authorize('YES')
  control('pressUpdate')
  state=wait(lambda:settled(''),660)
  if state['error']:raise AssertionError('Actual upgrade button failed: '+state['error'])
  assert state['status'] and not state['status']['available'] and not state['status']['pending'] and state['refreshes']==1,state
  assert json.loads((setup/'service-upgrade.json').read_text())['complete']
  assert before==project('0.3.4'),'fleet configuration/history changed'
  assert runtime_before==runtime('0.3.4'),'engine or live ship restarted'
  assert original_access==access_hashes(),'desktop access changed'
  report['upgraded']={'state':state,'schema_after':db_projection()['schema'],'ship_continuity':True,'access_preserved':True}
  control('pressCheck');state=wait(lambda:settled())
  assert not state['status']['available']
  commands=state['commands'];control('pressUpdate');assert control('state')['commands']==commands,'completed hidden button redispatched'
  report['recheck_noop']=True
  capture('complete')
 finally:
  try:control('stop')
  except Exception:proc.terminate()
  proc.wait(timeout=15);qlog.close()
  text=(O/'qml.log').read_text();print('UPGRADE_QML_LOG '+text[-12000:],flush=True)
  rule.unlink(missing_ok=True)
 assert collector('0.3.3','snapshot')['runner_id']==before['runner_id']
 run(['systemctl','--user','stop','urbit-runner.service'],True)
 assert db_projection()['schema']==report['schema_before']==47
 run(['systemctl','--user','start','urbit-runner.service'],True)
 wait(lambda:collector('0.3.4','hello'))
 assert before==project('0.3.4') and runtime_before==runtime('0.3.4')
 report['manager_restart_preserved_runtime']=True
 preview=json.loads(storage('0.3.4','preview-pier','--source',import_source,'--storage-root','default').stdout)
 assert preview['ship']=='~nec' and preview['network']=='fakenet',preview
 request='macos-import-nec'
 storage('0.3.4','import-pier',preview['instance_id'],'--source',import_source,'--path',preview['destination'],'--ship',preview['ship'],'--fingerprint',preview['source']['fingerprint'],'--http-port',str(preview['http_port']),'--request-id',request,'--confirm-stopped','--identify')
 def imported():
  snapshot=collector('0.3.4','snapshot')
  op=next((x for x in snapshot['operations'] if x['request_id']==request),None)
  if not op or op['status'] in ('queued','pending','running'):return None
  assert op['status']=='succeeded',op
  row=next(x for x in snapshot['instances'] if x['id']==preview['instance_id'])
  return row if row['observed_state']=='stopped' and not row.get('active_operation_id') else None
 imported_row=wait(imported,180)
 target=Path('/srv/urbit/piers')/imported_row['pier_path']
 assert not import_source.exists()
 for name,digest in companions.items():assert hashlib.sha256((target/name).read_bytes()).hexdigest()==digest,name
 assert runtime_before==runtime('0.3.4'),'import interrupted the unrelated running fake'
 assert ready('0.3.4','upgrade-nec','stopped')
 report['macos_import'].update(passed=True,identity=preview['ship'],stayed_stopped=True,metadata_preserved=True,unrelated_runtime_preserved=True)

 operation('0.3.4','upgrade-zod','stop','release-final-stop')
 wait(lambda:ready('0.3.4','upgrade-zod','stopped'))
 report['passed']=True

if __name__=='__main__':
 if os.geteuid()!=0 or not Path('/sys/class/dmi/id/sys_vendor').exists() or Path('/sys/class/dmi/id/sys_vendor').read_text().strip()!='QEMU' or 'omabit.disposable-guest=release-upgrade-v1' not in Path('/proc/cmdline').read_text().split():
  raise SystemExit('Refused: requires the explicitly provisioned offline release guest')
 try:main()
 except Exception as exc:
  report['error']=str(exc)[:3000]
  if created:
   print('UPGRADE_LAST_COMMANDS '+(O/'commands.jsonl').read_text()[-12000:],flush=True)
   diagnostic=subprocess.run(['journalctl','--no-pager','-n','30','-u','user@1000.service'],capture_output=True,text=True,timeout=15)
   print('UPGRADE_DIAGNOSTIC '+diagnostic.stdout[-8000:],flush=True)
 finally:
  if created:(O/'report.json').write_text(json.dumps(report,indent=2)+'\n')
  print('OMABIT_RELEASE_UPGRADE_RESULT '+json.dumps(report),flush=True)
  # Only the guest marker allows poweroff; a host invocation must be inert.
  if os.geteuid()==0 and 'omabit.disposable-guest=release-upgrade-v1' in Path('/proc/cmdline').read_text().split():
   subprocess.run(['systemctl','--no-block','poweroff'],check=True)
