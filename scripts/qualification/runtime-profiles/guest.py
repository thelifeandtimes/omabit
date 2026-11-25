#!/usr/bin/python3
import base64,hashlib,http.client,json,os,shutil,socket,sqlite3,subprocess,time,traceback
from pathlib import Path
I=Path('/opt/omabit-release-input');O=Path('/opt/omabit-release-results');SOCK='/run/user/1000/urbit-runner/runner.sock'
assert os.geteuid()==0 and Path('/sys/class/dmi/id/sys_vendor').read_text().strip()=='QEMU' and 'omabit.disposable-guest=release-upgrade-v1' in Path('/proc/cmdline').read_text().split()
scenario=json.loads((I/'scenario.json').read_text());V=scenario['target'];created=False
report={'format':1,'passed':False,'checks':{},'scope':'core functionality only','source_commit':json.loads((I/V/'release.json').read_text())['source_commit']}
PROFILE='vere-4.6-jsonfix-0774857c';STOCK='sha256:d6cc5107ed7fa92174e336af9c7f25ee433073bd9961e1c3ed23a225b866b245';PATCH='sha256:9aff0a010f4bfecff2233cc5508a0a6deec078239038456c94bbe0666b7a6f58'
STOCK_MANIFEST='sha256:47fbcf0710ded939931ce3c6408864cee5e8848c5765da0b51a90ca0b1ab1afd';PATCH_MANIFEST='sha256:2b6b7939b09aa0be68bf5a80b796f89382040aeb0d3469c032063a4c93fb5b1d'
def run(args,owner=False,timeout=90,ok=True,input=None):
 args=list(map(str,args))
 if owner:args=['runuser','-u','review','--','env','HOME=/home/review','XDG_RUNTIME_DIR=/run/user/1000','DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/1000/bus','PATH=/usr/bin:/bin',*args]
 if 'snapshot' not in args:print('PROFILE_STEP '+json.dumps(args[:18]),flush=True)
 r=subprocess.run(args,input=input,capture_output=True,text=True,timeout=timeout)
 with (O/'commands.jsonl').open('a') as f:f.write(json.dumps({'args':args,'input':input,'exit':r.returncode,'out':r.stdout[-16000:],'err':r.stderr[-16000:]})+'\n')
 if ok and r.returncode:raise RuntimeError('Command failed: '+' '.join(args[:15])+'\n'+r.stderr[-1500:])
 return r

def tool(n):return I/V/(n+'-linux-amd64')
def col(*args,ok=True):
 r=run([tool('urbit-collector'),'--socket',SOCK,*args],True,ok=ok)
 return json.loads(r.stdout) if ok else r

def docker(*args,**kw):return run(['docker','--host','unix:///run/user/1000/urbit-docker.sock',*args],True,**kw)
def wait(fn,seconds=300):
 end=time.monotonic()+seconds
 while time.monotonic()<end:
  val=fn()
  if val:return val
  time.sleep(1)
 raise RuntimeError('readiness deadline exceeded')
def row(name):return next(x for x in col('snapshot')['instances'] if x['id']==name)
def operation(name,action,request,success=True):
 col('operate',name,action,'--revision',str(row(name)['config_revision']),'--request-id',request)
 def terminal():
  op=next((x for x in col('snapshot')['operations'] if x['request_id']==request),None)
  return op if op and op['status'] not in ('queued','pending','running') else None
 op=wait(terminal,600)
 assert (op['status']=='succeeded')==success,op
 return op

def endpoint(path,method='GET',body=None):
 # Run as the enrolled ordinary operator so peer credentials follow real CLI semantics.
 args=['curl','--silent','--show-error','--unix-socket',SOCK,'-X',method,'-H','Urbit-Runner-Protocol: 1','-w','\n%{http_code}','http://localhost'+path]
 if body is not None:args+=['-H','Content-Type: application/json','--data',json.dumps(body)]
 r=run(args,True);data,code=r.stdout.rsplit('\n',1);return int(code),json.loads(data)
def check(name):report['checks'][name]=True;print('PROFILE_CHECK '+name,flush=True)
def api_preflight():return endpoint('/v1/runtime-profiles/'+PROFILE)
def audit_profile(name,expected,images):
 r=row(name);assert r['config'].get('runtime_profile','')==expected,r['config']
 status=r['runtime_status'];assert status['observed_image_id'] in images,status
 if expected:
  assert status['requested_runtime']['id']==expected and status['launch_runtime']['id']==expected,status
  assert status['requested_runtime']['requirement']=='vere-json-array-error-1145-v1',status
 return r

def main():
 global created
 assert not O.exists();O.mkdir(mode=0o700);created=True
 for p in ('/etc/urbit-runner','/srv/urbit/piers','/home/review/.local/share/urbit-runner'):assert not Path(p).exists()
 assert run(['pgrep','-x','dockerd'],ok=False).returncode==1
 checks={s.split('  ',1)[1]:s.split('  ',1)[0] for s in (I/V/'SHA256SUMS').read_text().splitlines()}
 for n in ('omabit','urbit-runner','urbit-collector'):assert hashlib.sha256(tool(n).read_bytes()).hexdigest()==checks[n+'-linux-amd64']
 assert hashlib.sha256((I/'jsonfix.tar.gz').read_bytes()).hexdigest()=='c8a412c42d53fa0c9a8c97aa55a580ddf2864bbdf32f80074dfb6bdfd837f510'
 run(['sysctl','-w','vm.overcommit_memory=1']);run(['loginctl','enable-linger','review']);run(['systemctl','start','user@1000.service'])
 run(['ssh-keygen','-q','-t','ed25519','-N','','-f','/home/review/profile-operator'],True)
 run([tool('urbit-runner'),'install-host','--owner','review','--bundle',I/V/'rootless-bundle','--public-key','/home/review/profile-operator.pub','--acknowledge-host-changes'],timeout=600)
 hello=wait(lambda:col('hello'));assert 'runtime-profiles-v1' in hello['features']
 run([tool('urbit-collector'),'enroll','local','--socket',SOCK],True)
 docker('load','--input',I/'runtime.oci.tar')
 events_log=(O/'container-events.jsonl').open('w')
 events=subprocess.Popen(['runuser','-u','review','--','env','HOME=/home/review','XDG_RUNTIME_DIR=/run/user/1000','PATH=/usr/bin:/bin','docker','--host','unix:///run/user/1000/urbit-docker.sock','events','--filter','type=container','--format','{{json .}}'],stdout=events_log,stderr=subprocess.DEVNULL)
 if scenario.get('runtime_case')!='local-runtime':
  assert api_preflight()[0]==409
  col('create-fake','profile-patched','--ship','~nec','--http-port','19181','--runtime-profile',PROFILE,'--loom','34','--request-id','profile-patched-create','--bootstrap','recommended','--pill',str(I/'boot.pill'))
  operation('profile-patched','start','missing-artifact-start',success=False)
  assert not row('profile-patched')['runtime_status'].get('initialized',False)
  assert not docker('ps','-q').stdout.strip()
  check('missing_artifact_refuses_start_before_pier_execution')
  corrupt=Path('/home/review/corrupt-runtime.tar.gz');shutil.copyfile(I/'jsonfix.tar.gz',corrupt)
  with corrupt.open('r+b') as f:f.write(b'BAD')
  os.chown(corrupt,1000,1000)
  fail=run([tool('urbit-runner'),'runtime-profile-install','--profile',PROFILE,'--archive',corrupt],True,ok=False)
  assert fail.returncode and 'checksum' in fail.stderr
  archive=Path('/home/review/.local/share/urbit-runner/runtime-profiles')/PROFILE/'image.tar.gz'
  assert not archive.exists();corrupt.unlink();check('corrupt_archive_install_refuses_without_publication')
  run([tool('urbit-runner'),'runtime-profile-install','--profile',PROFILE,'--archive',I/'jsonfix.tar.gz'],True)
  code,preflight=api_preflight();assert code==200 and preflight['available'] and preflight['profile']['id']==PROFILE
  run([tool('urbit-runner'),'runtime-profile-install','--profile',PROFILE,'--archive',I/'jsonfix.tar.gz'],True)
  check('exact_archive_install_idempotent_and_native_preflight')
  call=run([tool('omabit'),'call','runtime.profile-preflight.v1','--input-json',json.dumps({'name':'local','runner_id':hello['runner_id'],'runtime_profile':PROFILE})],True,ok=False)
  assert call.returncode==0,(call.stdout,call.stderr)
  report['shared_cli_preflight']={'exit':call.returncode,'out':call.stdout[-2500:],'err':call.stderr[-800:]}
  # Exact image metadata and binaries before any pier creation.
  for label,image,binary,version in [('stock',STOCK_MANIFEST,'b5c54439ae5b9f0ff94d75319e67e04f04369776fa5e2180a5ab1a5ebd4c2ff0','4.6-5a7614f'),('patched',PATCH_MANIFEST,'0774857cdb4460fc288bfa55e778249456a7d8bd7b6203922455d887a92fc96c','4.6-82f4558')]:
   value=docker('run','--rm','--network','none','--read-only','--entrypoint','/bin/sh',image,'-c','sha256sum /bin/urbit; /bin/urbit --version').stdout
   assert binary in value and version in value,value
  check('container_binary_checksums_and_versions')
  # The genuine manager executes and maintains two different pinned runtime images.
  for name,ship,port,profile in [('profile-stock','~zod',19180,''),('profile-patched','~nec',19181,PROFILE)]:
   args=['create-fake',name,'--ship',ship,'--http-port',str(port),'--request-id',name+'-create','--loom','34','--bootstrap','recommended','--pill',str(I/'boot.pill')]
   if profile:args+=['--runtime-profile',profile]
  
   if not profile:col(*args)
   operation(name,'start',name+'-start');wait(lambda:row(name)['observed_state']=='running',600)
   audit_profile(name,profile,[PATCH,PATCH_MANIFEST] if profile else [STOCK,STOCK_MANIFEST])
  check('two_fresh_ships_different_pinned_runtimes')
  before={name:row(name)['runtime_status']['container_id'] for name in ('profile-stock','profile-patched')}
  run(['systemctl','--user','restart','urbit-runner.service'],True);wait(lambda:col('hello'))
  assert before=={name:row(name)['runtime_status']['container_id'] for name in before};check('manager_restart_preserves_both_containers_and_pins')
  # Loss of the archive must not prevent a safe stop or select stock on restart.
  holding=archive.with_name('image.retained-for-test.tar.gz');archive.rename(holding)
  operation('profile-patched','stop','patched-missing-stop');assert row('profile-stock')['runtime_status']['container_id']==before['profile-stock']
  operation('profile-patched','start','patched-missing-start',success=False)
  assert row('profile-patched')['observed_state']!='running' and api_preflight()[0]==409
  holding.rename(archive)
  operation('profile-patched','stop','patched-restored-artifact-reconcile')
  assert not row('profile-patched').get('recovery_required', False)
  check('missing_archive_allows_stop_refuses_restart_no_fallback')
  # A changed runtime selection on an enrolled pier must refuse without a write.
  prior=row('profile-patched');config=dict(prior['config']);config['runtime_profile']='vere-4.6-5a7614f'
  code,error=endpoint('/v1/instances/profile-patched/configuration','PUT',{'expected_revision':prior['config_revision'],'config':config})
  assert code==409 and row('profile-patched')['config']==prior['config'];report['transition_refusal']=error;check('enrolled_downgrade_refuses_preserving_configuration')
  # Old CLI's omitted profile field must preserve the selection.
  col('configure','profile-patched','--revision',str(prior['config_revision']),'--http-port','19181','--ames-port','34343','--loom','34')
  assert row('profile-patched')['config']['runtime_profile']==PROFILE
  operation('profile-patched','start','patched-replacement-start');wait(lambda:row('profile-patched')['observed_state']=='running',600)
  patched=audit_profile('profile-patched',PROFILE,[PATCH,PATCH_MANIFEST]);assert patched['runtime_status']['container_id']!=before['profile-patched']
  check('replacement_and_omitted_profile_preserve_pin')
  operation('profile-patched','stop','patched-before-pack');operation('profile-patched','pack','patched-pack')
  operation('profile-patched','start','patched-after-pack');wait(lambda:row('profile-patched')['observed_state']=='running',600)
  audit_profile('profile-patched',PROFILE,[PATCH,PATCH_MANIFEST]);check('pack_and_restart_preserve_pin')
  # Only the fresh disposable patched fake receives a controlled crash.
  crashed=row('profile-patched')['runtime_status']['container_id']
  docker('kill','--signal','KILL',crashed)
  wait(lambda:row('profile-patched')['observed_state']!='running')
  operation('profile-patched','stop','patched-crash-reconcile')
  operation('profile-patched','start','patched-crash-recovery');wait(lambda:row('profile-patched')['observed_state']=='running',600)
  audit_profile('profile-patched',PROFILE,[PATCH,PATCH_MANIFEST]);check('controlled_crash_recovery_preserves_pin')
  # Import a stopped copy of this new fake, never a user pier or live identity.
  operation('profile-patched','stop','patched-before-import-copy')
  original=row('profile-patched');source=Path('/home/review/reviewed-import-source')
  run(['cp','-a','--reflink=auto',Path('/srv/urbit/piers')/original['pier_path'],source],True,timeout=180)
  def storage(*args,ok=True):return run([tool('urbit-collector'),'--runner','local','--expected-runner-id',hello['runner_id'],*args],True,ok=ok,timeout=180)
  refused=storage('preview-pier','--source',source,'--storage-root','default',ok=False)
  assert refused.returncode and source.exists(),(refused.stdout,refused.stderr)
  preview=json.loads(storage('preview-pier','--source',source,'--storage-root','default','--runtime-profile',PROFILE).stdout)
  assert preview['ship']=='~nec' and preview['network']=='fakenet',preview
  imported=preview['instance_id']
  storage('import-pier',imported,'--source',source,'--path',preview['destination'],'--ship',preview['ship'],'--fingerprint',preview['source']['fingerprint'],'--http-port',str(preview['http_port']),'--runtime-profile',PROFILE,'--request-id','reviewed-import','--confirm-stopped','--identify')
  def imported_ready():
   snap=col('snapshot');op=next((x for x in snap['operations'] if x['request_id']=='reviewed-import'),None)
   if not op or op['status'] in ('queued','pending','running'):return False
   assert op['status']=='succeeded',op
   return row(imported)['observed_state']=='stopped'
  wait(imported_ready,300)
  assert row(imported)['config']['runtime_profile']==PROFILE and not source.exists()
  current=row(imported)
  col('configure',imported,'--revision',str(current['config_revision']),'--http-port',str(current['config']['http_port']),'--ames-port',str(current['config']['ames_port']),'--loom','34')
  operation(imported,'start','reviewed-import-first-start');wait(lambda:row(imported)['observed_state']=='running',600)
  audit_profile(imported,PROFILE,[PATCH,PATCH_MANIFEST]);operation(imported,'stop','reviewed-import-stop')
  report['imported_instance']=imported;check('reviewed_import_refuses_stock_and_preserves_pin_through_start')
  operation('profile-patched','start','patched-after-import-copy');wait(lambda:row('profile-patched')['observed_state']=='running',600)
  for name in ('profile-patched','profile-stock'):
   if row(name)['desired_state']!='stopped' or row(name)['observed_state']=='running':operation(name,'stop',name+'-final-stop')
 # Expand the ordinary workflows with a separately packaged local executable.
 local=scenario.get('local_runtime')
 if local:
  local_id=local['id'];local_images=[local['config_digest'],local['manifest_digest']]
  run([tool('urbit-runner'),'runtime-profile-install','--manifest',I/'local-profile.json','--archive',I/'local-runtime.tar.gz'],True)
  code,value=endpoint('/v1/runtime-profiles/'+local_id)
  expected={k:v for k,v in local.items() if v!=''}
  assert code==200 and value['available'] and value['profile']==expected,(code,value,expected)
  col('create-fake','profile-local','--ship','~zod','--http-port','19185','--runtime-profile',local_id,'--loom','34','--request-id','profile-local-create','--bootstrap','recommended','--pill',str(I/'boot.pill'))
  operation('profile-local','start','local-first-start');wait(lambda:row('profile-local')['observed_state']=='running',600)
  audit_profile('profile-local',local_id,local_images)
  operation('profile-local','stop','local-first-stop');check('local_manifest_install_and_fresh_boot')
  if scenario.get('runtime_case')=='local-runtime':
   col('create-fake','profile-stock','--ship','~nec','--http-port','19180','--loom','34','--request-id','profile-stock-create','--bootstrap','recommended','--pill',str(I/'boot.pill'))
   operation('profile-stock','start','local-case-stock-start');wait(lambda:row('profile-stock')['observed_state']=='running',600)
   operation('profile-stock','stop','local-case-stock-stop')
  before_switch=row('profile-stock');config=dict(before_switch['config']);config['runtime_profile']=local_id
  # Confirmation is required, and changing intent does not automatically boot.
  code,error=endpoint('/v1/instances/profile-stock/configuration','PUT',{'expected_revision':before_switch['config_revision'],'config':config})
  assert code==409 and row('profile-stock')['config']==before_switch['config']
  col('configure','profile-stock','--revision',str(before_switch['config_revision']),'--http-port',str(config['http_port']),'--ames-port',str(config['ames_port']),'--loom',str(config['loom']),'--runtime-profile',local_id,'--confirm-runtime-change')
  assert row('profile-stock')['observed_state']=='stopped' and row('profile-stock')['pier_path']==before_switch['pier_path']
  operation('profile-stock','start','stock-to-local-start');wait(lambda:row('profile-stock')['observed_state']=='running',600)
  audit_profile('profile-stock',local_id,local_images);operation('profile-stock','stop','stock-to-local-stop')
  check('explicit_stopped_stock_to_local_transition_preserves_pier')
  local_source=Path('/home/review/local-import-source')
  run(['cp','-a','--reflink=auto',Path('/srv/urbit/piers')/row('profile-local')['pier_path'],local_source],True,timeout=180)
  plan=json.loads(storage('preview-pier','--source',local_source,'--storage-root','default','--runtime-profile',local_id).stdout)
  local_import=plan['instance_id']
  storage('import-pier',local_import,'--source',local_source,'--path',plan['destination'],'--ship',plan['ship'],'--fingerprint',plan['source']['fingerprint'],'--http-port',str(plan['http_port']),'--runtime-profile',local_id,'--request-id','local-import','--confirm-stopped','--identify')
  def local_import_ready():
   snap=col('snapshot');op=next((x for x in snap['operations'] if x['request_id']=='local-import'),None)
   if not op or op['status'] in ('queued','pending','running'):return False
   assert op['status']=='succeeded',op
   return row(local_import)['observed_state']=='stopped'
  wait(local_import_ready,300)
  operation(local_import,'start','local-import-start');wait(lambda:row(local_import)['observed_state']=='running',600)
  pinned=row(local_import)['runtime_status']['container_id']
  run(['systemctl','--user','restart','urbit-runner.service'],True);wait(lambda:col('hello'))
  assert row(local_import)['runtime_status']['container_id']==pinned
  audit_profile(local_import,local_id,local_images)
  operation(local_import,'stop','local-import-stop');check('local_import_and_manager_restart_preserve_pin')
  report['local_runtime']=local
 events.terminate();events.wait(timeout=10);events_log.close()
 with sqlite3.connect('file:/home/review/.local/share/urbit-runner/managed/runner.sqlite?mode=ro',uri=True) as db:
  saved={id:json.loads(body) for id,body in db.execute('select id,body from instances')}
 report['native_launches']=[]
 for line in (O/'container-events.jsonl').read_text().splitlines():
  event=json.loads(line)
  if event.get('Action')!='create':continue
  attrs=event['Actor']['Attributes'];name=attrs.get('name','')
  for id,item in saved.items():
   if item['runtime']['id'] in name:
    profiles={p['id']:p for p in col('hello')['runtime_profiles']}
    pins={item['public']['config'].get('runtime_profile') or 'vere-4.6-5a7614f'}
    for transition in item['runtime'].get('runtime_history',[]):pins.update((transition['from']['id'],transition['to']['id']))
    expected=[digest for pin in pins for digest in (profiles[pin]['manifest_digest'],profiles[pin]['config_digest'])]
    assert attrs['image'] in expected,(id,attrs)
    report['native_launches'].append({'instance':id,'name':name,'image':attrs['image']})
 if scenario.get('runtime_case')!='local-runtime':assert any(x['instance']=='profile-patched' and x['name'].endswith('-pack') for x in report['native_launches']),report['native_launches']
 check('native_utility_and_lifecycle_image_audit')
 assert all(x['public']['desired_state']=='stopped' for x in saved.values());check('both_test_ships_gracefully_stopped')
 report['final_snapshot']=col('snapshot')
 report['passed']=True

try:main()
except Exception as exc:
 report['error']=str(exc)[:3500];report['traceback']=traceback.format_exc()[-3500:]
 if created:print('PROFILE_LAST_COMMANDS '+(O/'commands.jsonl').read_text()[-12000:],flush=True)
finally:
 if created:(O/'report.json').write_text(json.dumps(report,indent=2)+'\n')
 raw=json.dumps(report,separators=(',',':')).encode();encoded=base64.b64encode(raw).decode();chunks=[encoded[i:i+3000] for i in range(0,len(encoded),3000)]
 for i,data in enumerate(chunks):print('OMABIT_PROFILE_REPORT_CHUNK '+json.dumps({'i':i,'data':data}),flush=True)
 print('OMABIT_PROFILE_REPORT_END '+json.dumps({'chunks':len(chunks),'sha256':hashlib.sha256(raw).hexdigest()}),flush=True)
 subprocess.run(['systemctl','--no-block','poweroff'],check=True)
