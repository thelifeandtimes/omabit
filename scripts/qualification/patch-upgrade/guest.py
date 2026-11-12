#!/usr/bin/python3
"""Offline published-binary upgrade check, only in the guarded release guest."""
import hashlib,json,os,sqlite3,subprocess,sys,time,shutil,base64
from pathlib import Path
I=Path('/opt/omabit-release-input'); O=Path('/opt/omabit-release-results')
SOCK='/run/user/1000/urbit-runner/runner.sock'
if os.geteuid()!=0 or not Path('/sys/class/dmi/id/sys_vendor').exists() or Path('/sys/class/dmi/id/sys_vendor').read_text().strip()!='QEMU' or 'omabit.disposable-guest=release-upgrade-v1' not in Path('/proc/cmdline').read_text().split():
 raise SystemExit('Refused: requires the explicitly provisioned offline release guest')
scenario=json.loads((I/'scenario.json').read_text())
PREVIOUS=scenario['runner']; TARGET=scenario['target']
VERSIONS=(PREVIOUS,TARGET)
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
 if scenario.get('namespace_tests'):
  report['kernel_semantics_tests']={}
  for binary,pattern in [('pierstore.test','^TestHostPathAllowsOnlyTheRunnersNarrowACLGrant$'),('rootless.test','^TestPrepareRefusesUnsafeOutputs$')]:
   result=run([I/binary,'-test.run',pattern,'-test.timeout=2m','-test.count=1','-test.v'],True)
   report['kernel_semantics_tests'][binary]={'passed':result.returncode==0,'sha256':hashlib.sha256((I/binary).read_bytes()).hexdigest()}
 report['sources']={v:json.loads((I/v/'release.json').read_text())['source_commit'] for v in VERSIONS}
 run(['loginctl','enable-linger','review'])
 run(['systemctl','start','user@1000.service'])
 report['starting_versions']=scenario
 setup=Path('/home/review/.config/omabit/setup')
 setup.mkdir(parents=True,mode=0o700)
 run(['chown','-R','review:review','/home/review/.config'])
 run(['ssh-keygen','-q','-t','ed25519','-N','','-f',setup/'id_ed25519'],True)
 shutil.copy2(setup/'id_ed25519.pub',Path('/home/review/release-operator.pub'))
 host=install(PREVIOUS)
 # Reproduce the published setup receipt format from actual installer evidence.
 # Initial setup/download itself is outside this upgrade rehearsal.
 contents={
  'operator.pub':(setup/'id_ed25519.pub').read_text(),
  'known_hosts':'[127.0.0.1]:22222 '+host['host_public_key']+'\n',
  'ssh_config':f'Host omabit-local\n  HostName 127.0.0.1\n  Port 22222\n  User review\n  IdentityFile "{setup}/id_ed25519"\n  IdentityAgent none\n  IdentitiesOnly yes\n  UserKnownHostsFile "{setup}/known_hosts"\n  GlobalKnownHostsFile /dev/null\n  StrictHostKeyChecking yes\n  BatchMode yes\n',
  'ready.json':json.dumps({'runner_id':host['runner_id'],'connection':'local','host_public_key':host['host_public_key']})}
 for name,data in contents.items():
  path=setup/name;path.write_text(data);path.chmod(0o600);os.chown(path,1000,1000)
 run([tool(PREVIOUS,'urbit-collector'),'enroll','local','--ssh','omabit-local','--ssh-config',setup/'ssh_config'],True)
 old_status=json.loads(run([tool(scenario['desktop'],'omabit'),'setup-status'],True).stdout)
 assert old_status['ready'] and old_status['connection']['runner_id']==host['runner_id']
 report['receipt_fixture']='published format reconstructed from genuine '+PREVIOUS+' installer result and SSH enrollment; old desktop setup-status verified ready'
 def access_hashes():
  return {name:hashlib.sha256((setup/name).read_bytes()).hexdigest() for name in contents.keys()|{'id_ed25519'}}
 original_access=access_hashes()
 run(['docker','--host','unix:///run/user/1000/urbit-docker.sock','load','--input',I/'runtime.oci.tar'],True)
 for name,ship,port in (('upgrade-zod','~zod',19180),('upgrade-nec','~nec',19181)):
  collector(PREVIOUS,'create-fake',name,'--ship',ship,'--http-port',str(port),'--request-id',name+'-create','--bootstrap','recommended','--pill',str(I/'boot.pill'))
  operation(PREVIOUS,name,'start',name+'-start')
  wait(lambda:ready(PREVIOUS,name,'running'),600)
  if name=='upgrade-nec':operation(PREVIOUS,name,'stop',name+'-stop');wait(lambda:ready(PREVIOUS,name,'stopped'))
 # A stopped fake created only by this offline guest supplies real Vere state.
 # Its separate test copy receives synthetic macOS metadata, never user data.
 nec=next(x for x in collector(PREVIOUS,'snapshot')['instances'] if x['id']=='upgrade-nec')
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
 old=storage(PREVIOUS,'preview-pier','--source',import_source,'--storage-root','default',ok=False)
 if PREVIOUS=='0.3.3':
  assert old.returncode and 'unrecognized epoch-log entry' in old.stderr,old.stderr
 else:
  assert old.returncode==0 and json.loads(old.stdout)['ship']=='~nec',old.stderr
 report['macos_import']={'previous_refused_metadata':bool(old.returncode),'synthetic_companion_sha256':companions}
 large_preview_test=tuple(map(int,TARGET.split('-')[0].split('.'))) >= (0,3,6)
 if large_preview_test:
  # A sparse file exercises real collector/runner preview validation without
  # allocating or copying 100 GB. It is removed before the actual small import.
  large_inventory_before=project(PREVIOUS)
  large_fixture=import_source/'large-preview-fixture.bin'
  with large_fixture.open('xb') as f:f.truncate(100_000_000_001)
  large_fixture.chmod(0o600);os.chown(large_fixture,1000,1000)
  large_size=large_fixture.stat().st_size
  large_old=storage(PREVIOUS,'preview-pier','--source',import_source,'--storage-root','default',ok=False)
  if PREVIOUS in ('0.3.4','0.3.5'):
   assert large_old.returncode and 'guided import identity, bounds or destination changed' in large_old.stderr,large_old.stderr
  large_new=json.loads(storage(TARGET,'preview-pier','--source',import_source,'--storage-root','default').stdout)
  assert large_new['source']['bytes']>=large_size and large_new['ship']=='~nec',large_new
  assert project(PREVIOUS)==large_inventory_before,'preview enrolled or changed a ship'
  report['large_local_preview']={'previous_rejected':bool(large_old.returncode),'target_collector_with_previous_runner_passed':True,'fixture_logical_bytes':large_size,'fixture_allocated_bytes':large_fixture.stat().st_blocks*512,'source_logical_bytes':large_new['source']['bytes'],'full_large_copy_tested':False}
 before=project(PREVIOUS); runtime_before=runtime(PREVIOUS)
 report['baseline']=before;report['runtime_before']=runtime_before;report['schema_before']=db_projection()['schema']
 # Import and verify actual release bundles with the native desktop loader.
 staged=json.loads(run(['python3','-B',I/'desktop.py',scenario['desktop']],True).stdout)
 report['desktop_transition']=staged
 assert before==project(PREVIOUS) and runtime_before==runtime(PREVIOUS),'desktop activation changed ships'
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
  assert before==project(PREVIOUS) and runtime_before==runtime(PREVIOUS) and original_access==access_hashes()
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
  assert before==project(TARGET),'fleet configuration/history changed'
  assert runtime_before==runtime(TARGET),'engine or live ship restarted'
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
 assert collector(PREVIOUS,'snapshot')['runner_id']==before['runner_id']
 run(['systemctl','--user','stop','urbit-runner.service'],True)
 expected_schemas={v:json.loads((I/v/'release.json').read_text())['schema'] for v in VERSIONS}
 assert report['schema_before']==expected_schemas[PREVIOUS]
 assert db_projection()['schema']==expected_schemas[TARGET]
 if expected_schemas[TARGET]>expected_schemas[PREVIOUS]:
  before_old_writer=db_projection()
  refused=run([tool(PREVIOUS,'urbit-runner'),'serve-development','--acknowledge-development'],True,timeout=20,ok=False)
  assert refused.returncode!=0 and 'schema' in (refused.stdout+refused.stderr).lower(), 'old writer did not refuse the newer schema'
  assert db_projection()==before_old_writer, 'old writer changed newer state'
  report['older_writer_refused_without_state_change']=True
 run(['systemctl','--user','start','urbit-runner.service'],True)
 wait(lambda:collector(TARGET,'hello'))
 assert before==project(TARGET) and runtime_before==runtime(TARGET)
 report['manager_restart_preserved_runtime']=True
 if large_preview_test:
  large_upgraded=json.loads(storage(TARGET,'preview-pier','--source',import_source,'--storage-root','default').stdout)
  assert large_upgraded['source']['bytes']==large_new['source']['bytes'] and large_upgraded['ship']=='~nec'
  assert large_fixture.stat().st_size==large_size
  report['large_local_preview'].update(target_collector_with_target_runner_passed=True,preview_preserved_source=True,preview_did_not_enroll=True,passed=True)
  large_fixture.unlink() # Only the synthetic fixture created above, never pier data.
 preview=json.loads(storage(TARGET,'preview-pier','--source',import_source,'--storage-root','default').stdout)
 assert preview['ship']=='~nec' and preview['network']=='fakenet',preview
 request='macos-import-nec'
 storage(TARGET,'import-pier',preview['instance_id'],'--source',import_source,'--path',preview['destination'],'--ship',preview['ship'],'--fingerprint',preview['source']['fingerprint'],'--http-port',str(preview['http_port']),'--request-id',request,'--confirm-stopped','--identify')
 def imported():
  snapshot=collector(TARGET,'snapshot')
  op=next((x for x in snapshot['operations'] if x['request_id']==request),None)
  if not op or op['status'] in ('queued','pending','running'):return None
  assert op['status']=='succeeded',op
  row=next(x for x in snapshot['instances'] if x['id']==preview['instance_id'])
  return row if row['observed_state']=='stopped' and not row.get('active_operation_id') else None
 imported_row=wait(imported,180)
 target=Path('/srv/urbit/piers')/imported_row['pier_path']
 assert not import_source.exists()
 for name,digest in companions.items():assert hashlib.sha256((target/name).read_bytes()).hexdigest()==digest,name
 assert runtime_before==runtime(TARGET),'import interrupted the unrelated running fake'
 assert ready(TARGET,'upgrade-nec','stopped')
 report['macos_import'].update(passed=True,identity=preview['ship'],stayed_stopped=True,metadata_preserved=True,unrelated_runtime_preserved=True)

 if scenario.get('fleet'):
  import fleet_guest
  report['fleet']=fleet_guest.exercise(run,tool,TARGET,host['runner_id'],host['host_public_key'])
 operation(TARGET,'upgrade-zod','stop','release-final-stop')
 wait(lambda:ready(TARGET,'upgrade-zod','stopped'))
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
