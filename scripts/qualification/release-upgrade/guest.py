#!/usr/bin/python3
"""Offline published-binary upgrade check, only in the guarded release guest."""
import hashlib,json,os,sqlite3,subprocess,sys,time
from pathlib import Path
I=Path('/opt/omabit-release-input'); O=Path('/opt/omabit-release-results')
SOCK='/run/user/1000/urbit-runner/runner.sock'
VERSIONS=('0.1.0','0.2.0','0.3.0')
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
 run([tool(v,'urbit-runner'),'install-host','--owner','review','--bundle',I/v/'rootless-bundle','--public-key','/home/review/release-operator.pub','--acknowledge-host-changes'],timeout=600)
 wait(lambda:collector(v,'hello'))

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
 run(['ssh-keygen','-q','-t','ed25519','-N','','-f','/home/review/release-operator'],True)
 install('0.1.0')
 run(['docker','--host','unix:///run/user/1000/urbit-docker.sock','load','--input',I/'runtime.oci.tar'],True)
 for name,ship,port in (('upgrade-zod','~zod',19180),('upgrade-nec','~nec',19181)):
  collector('0.1.0','create-fake',name,'--ship',ship,'--http-port',str(port),'--request-id',name+'-create','--bootstrap','recommended','--pill',str(I/'boot.pill'))
  operation('0.1.0',name,'start',name+'-start')
  wait(lambda:ready('0.1.0',name,'running'),600)
  if name=='upgrade-nec':operation('0.1.0',name,'stop',name+'-stop');wait(lambda:ready('0.1.0',name,'stopped'))
 before=project('0.1.0'); runtime_before=runtime('0.1.0')
 report['baseline']=before;report['runtime_before']=runtime_before
 for version in VERSIONS[1:]:
  install(version)
  after=project(version)
  assert before==after, 'identity, settings, intended state or operation history changed'
  assert runtime_before==runtime(version), 'engine or running ship restarted'
  report['steps'].append({'version':version,'preserved':True,'schema':db_projection()['schema']})
  print('UPGRADE_PRESERVED '+version,flush=True)
  # Old clients must still read compatible observation endpoints.
  assert collector('0.1.0','snapshot')['runner_id']==before['runner_id']
  assert collector('0.2.0','snapshot')['runner_id']==before['runner_id']
 # Same target reinstall is idempotent and does not restart the engine/ships.
 install('0.3.0')
 assert before==project('0.3.0') and runtime_before==runtime('0.3.0')
 run(['systemctl','--user','stop','urbit-runner.service'],True)
 db_before=db_projection()
 for old in VERSIONS[:2]:
  rejected=run([tool(old,'urbit-runner'),'serve-development','--acknowledge-development'],True,timeout=20,ok=False)
  assert rejected.returncode!=0 and 'schema' in rejected.stderr, 'old reader did not refuse newer schema'
  assert db_before==db_projection(),'old reader altered state'
 run(['systemctl','--user','start','urbit-runner.service'],True)
 wait(lambda:collector('0.3.0','hello'))
 assert before==project('0.3.0') and runtime_before==runtime('0.3.0')
 # Explicitly stop only the freshly created test identity before guest poweroff.
 operation('0.3.0','upgrade-zod','stop','release-final-stop')
 wait(lambda:ready('0.3.0','upgrade-zod','stopped'))
 report['passed']=True

if __name__=='__main__':
 if os.geteuid()!=0 or not Path('/sys/class/dmi/id/sys_vendor').exists() or Path('/sys/class/dmi/id/sys_vendor').read_text().strip()!='QEMU' or 'omabit.disposable-guest=release-upgrade-v1' not in Path('/proc/cmdline').read_text().split():
  raise SystemExit('Refused: requires the explicitly provisioned offline release guest')
 try:main()
 except Exception as exc:
  report['error']=str(exc)[:3000]
  if created:
   diagnostic=subprocess.run(['journalctl','--no-pager','-n','30','-u','user@1000.service'],capture_output=True,text=True,timeout=15)
   print('UPGRADE_DIAGNOSTIC '+diagnostic.stdout[-8000:],flush=True)
 finally:
  if created:(O/'report.json').write_text(json.dumps(report,indent=2)+'\n')
  print('OMABIT_RELEASE_UPGRADE_RESULT '+json.dumps(report),flush=True)
  # Only the guest marker allows poweroff; a host invocation must be inert.
  if os.geteuid()==0 and 'omabit.disposable-guest=release-upgrade-v1' in Path('/proc/cmdline').read_text().split():
   subprocess.run(['systemctl','--no-block','poweroff'],check=True)
