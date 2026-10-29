#!/usr/bin/env python3
"""Characterize actual receiver group lifetime inside a fresh no-NIC namespace."""
import argparse,hashlib,json,os,re,shutil,signal,socket,subprocess,time
from pathlib import Path
RECEIVER=r'''import json,os,subprocess,time
from pathlib import Path
root=Path(os.environ['HOME'])
childcode="import json,os,time;from pathlib import Path;p=Path(os.environ['HOME']);p.joinpath('child.json').write_text(json.dumps({'pid':os.getpid(),'start':Path('/proc/self/stat').read_text().split(') ',1)[1].split()[19]}));time.sleep(60)"
p=subprocess.Popen(['/usr/bin/python3','-I','-S','-B','-c',childcode],stdin=subprocess.DEVNULL,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
root.joinpath('receiver.json').write_text(json.dumps({'pid':os.getpid(),'group':os.getpgrp(),'start':Path('/proc/self/stat').read_text().split(') ',1)[1].split()[19]}))
while not root.joinpath('release').exists():time.sleep(.02)
print('{}')
'''
def sha(p):return hashlib.sha256(p.read_bytes()).hexdigest()
def inside():
 root=Path('/lab')
 assert os.environ.get('OMABIT_RECEIVER_LIFECYCLE')=='private-v1' and os.environ.get('HOME')=='/lab/home'
 assert not Path('/home').exists() and not Path('/run/user').exists() and (root/'marker').read_text()=='private-receiver-v1\n'
 guard=json.loads((root/'host-namespaces.json').read_text());assert all(os.readlink('/proc/self/ns/'+n)!=v for n,v in guard.items())
 assert [n for _,n in socket.if_nameindex()]==['lo']
 report={'passed':False,'actual_receiver_adapter':True,'actual_collector_cli':False,'credential_access':False,'cases':[]}
 def save():(root/'result.json').write_text(json.dumps(report,indent=2)+'\n')
 def ident(pid):
  try:
   parts=Path('/proc/'+str(pid)+'/stat').read_text().split(') ',1)[1].split()
   return {'pid':pid,'start':parts[19],'state':parts[0]}
  except FileNotFoundError:return None
 def alive(old):
  now=ident(old['pid']);return bool(now and now['start']==old['start'] and now['state'] not in ('Z','X'))
 def wait(fn,seconds=5):
  end=time.monotonic()+seconds
  while time.monotonic()<end:
   value=fn()
   if value:return value
   time.sleep(.03)
  raise AssertionError('receiver lifecycle deadline')
 save()
 try:
  for mode in ('normal','cancel','crash'):
   case=root/'cases'/mode;case.mkdir();(case/'state').mkdir();artifact=case/'artifact';artifact.mkdir()
   (artifact/'manifest.json').write_text(json.dumps({'id':'fixture.receiver-lab'}))
   (artifact/'application-profile.json').write_text(json.dumps({'version':1,'profile':'receiver-lab'}))
   (artifact/'receiver.py').write_text(RECEIVER)
   owner=None;receiver=None;child=None
   try:
    with (case/'owner.log').open('w') as log:
     owner=subprocess.Popen(['/lab/artifacts/receiver-probe',mode],stdout=log,stderr=subprocess.STDOUT,start_new_session=True)
     wait(lambda:(case/'receiver.json').exists() and (case/'child.json').exists())
     receiver=json.loads((case/'receiver.json').read_text());child=json.loads((case/'child.json').read_text())
     assert receiver['pid']==receiver['group'] and alive(receiver) and alive(child)
     if mode=='normal':(case/'release').touch()
     elif mode=='cancel':owner.send_signal(signal.SIGTERM)
     else:owner.kill()
     code=owner.wait(timeout=5)
     assert code=={'normal':0,'cancel':1,'crash':-signal.SIGKILL}[mode],(mode,code)
     if mode=='crash':
      time.sleep(.4);assert alive(receiver) and alive(child),'abrupt owner lifetime changed; review characterization'
      observed={'owner_exit_proves_children_exited':False,'receiver_survived':True,'descendant_survived':True}
      assert os.getpgid(receiver['pid'])==receiver['group']
      os.killpg(receiver['group'],signal.SIGKILL)
     else:observed={'owner_exit_proves_children_exited':True}
     wait(lambda:not alive(receiver) and not alive(child))
     report['cases'].append({'mode':mode,'owner_exit':code,'receiver':receiver,'child':child,'observation':observed,'exact_children_cleaned':True});save()
   finally:
    if owner is not None and owner.poll() is None:owner.kill();owner.wait(timeout=3)
    if receiver and alive(receiver) and os.getpgid(receiver['pid'])==receiver['group']:os.killpg(receiver['group'],signal.SIGKILL)
    if child and alive(child):os.kill(child['pid'],signal.SIGKILL)
  report.update(passed=True,abrupt_owner_exit_is_not_collection_closure=True);save()
  print('PASS characterization: normal/cancel reap the receiver group; abrupt owner death leaves it alive until exact fixture cleanup')
 except Exception as e:report['error']=type(e).__name__+': '+str(e);save();raise
def main():
 p=argparse.ArgumentParser(description=__doc__);p.add_argument('--inside',action='store_true');p.add_argument('--source',type=Path);p.add_argument('--output',type=Path);a=p.parse_args()
 if a.inside:return inside()
 if not a.source or not a.output:p.error('--source and --output required')
 source=a.source.resolve(strict=True);out=a.output.absolute()
 assert out.parent==Path('/tmp') and re.fullmatch('omabit-receiver-lifecycle-[A-Za-z0-9-]+',out.name) and not out.exists();out.mkdir(mode=0o700)
 for n in ('home','runtime','config','state','cache','data','artifacts','etc','cases'):(out/n).mkdir(mode=0o700)
 (out/'etc/passwd').write_text('lab:x:'+str(os.getuid())+':'+str(os.getgid())+':Lab:/lab/home:/bin/sh\n');(out/'etc/group').write_text('lab:x:'+str(os.getgid())+':\n');(out/'etc/nsswitch.conf').write_text('passwd: files\ngroup: files\n')
 (out/'marker').write_text('private-receiver-v1\n');(out/'host-namespaces.json').write_text(json.dumps({n:os.readlink('/proc/self/ns/'+n) for n in ('user','net','pid','mnt')}))
 files=[*source.glob('internal/**/*.go'),source/'scripts/qualification/collections/receiverprobe/main.go',source/'go.mod',source/'go.sum']
 hashes={str(f.relative_to(source)):sha(f) for f in files if not f.name.endswith('_test.go')}
 env=dict(os.environ,OMABIT_TEST_FAKE_ZOD='0',GOCACHE='/tmp/omabit-go-build',GOMODCACHE='/tmp/omabit-go-mod',GOPROXY='off',GOMAXPROCS='2')
 q=subprocess.run(['go','build','-o',str(out/'artifacts/receiver-probe'),'./scripts/qualification/collections/receiverprobe'],cwd=source,env=env,capture_output=True,text=True,timeout=90)
 (out/'build.log').write_text(q.stdout+q.stderr);assert q.returncode==0,'probe build failed'
 assert all(sha(source/f)==h for f,h in hashes.items()),'source changed during build'
 refusal=subprocess.run([str(out/'artifacts/receiver-probe'),'normal'],capture_output=True,text=True,timeout=5);assert refusal.returncode==2
 (out/'host-refusal.log').write_text(refusal.stdout+refusal.stderr)
 shutil.copy2(__file__,out/'driver.py');(out/'source.json').write_text(json.dumps({'sha256':hashes,'fixture_sha256':sha(Path(__file__)),'binary_sha256':sha(out/'artifacts/receiver-probe')},indent=2)+'\n')
 cmd=['bwrap','--unshare-all','--die-with-parent','--clearenv','--ro-bind','/usr','/usr','--symlink','usr/bin','/bin','--symlink','usr/lib','/lib','--symlink','usr/lib','/lib64','--proc','/proc','--dev','/dev','--tmpfs','/tmp','--ro-bind',str(out/'etc'),'/etc','--bind',str(out),'/lab','--ro-bind',str(out/'artifacts'),'/lab/artifacts','--chdir','/lab']
 env={'HOME':'/lab/home','XDG_RUNTIME_DIR':'/lab/runtime','XDG_CONFIG_HOME':'/lab/config','XDG_DATA_HOME':'/lab/data','XDG_STATE_HOME':'/lab/state','XDG_CACHE_HOME':'/lab/cache','PATH':'/usr/bin:/bin','LANG':'C.UTF-8','PYTHONDONTWRITEBYTECODE':'1','OMABIT_RECEIVER_LIFECYCLE':'private-v1'}
 for k,v in env.items():cmd+=['--setenv',k,v]
 cmd+=['--','/usr/bin/python3','-B','/lab/driver.py','--inside'];(out/'command.json').write_text(json.dumps(cmd,indent=2)+'\n')
 q=subprocess.run(cmd,capture_output=True,text=True,timeout=35);(out/'driver.log').write_text(q.stdout+q.stderr);print(q.stdout+q.stderr);raise SystemExit(q.returncode)
if __name__=='__main__':main()
