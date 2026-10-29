#!/usr/bin/env python3
"""Characterize current per-root admission scope; no application/ship is started."""
from pathlib import Path
import argparse,hashlib,json,os,select,shutil,socket,subprocess,sys,uuid

def save(p,v):p.write_text(json.dumps(v,indent=2)+'\n')
def guard_namespace():
 assert Path.home()==Path('/lab/home') and not Path('/home').exists() and not Path('/run/user').exists()
 assert [n for _,n in socket.if_nameindex()]==['lo']
 assert all(os.readlink('/proc/self/ns/'+n)!=v for n,v in json.loads(Path('/lab/host-namespaces.json').read_text()).items())
def receive(p):
 assert select.select([p.stdout],[],[],8)[0],'owned child response deadline'
 row=p.stdout.readline();assert row,'owned child exited without response';return json.loads(row)
def send(p,v):p.stdin.write(v+'\n');p.stdin.flush()
def worker(slot):
 guard_namespace();assert slot in ('a','b');sys.path.insert(0,'/lab/source');import admission as a;import journal as j
 base=Path('/lab')/slot;base.mkdir(mode=0o700);store=j.Store(base/'coordinator',create=True);(base/'gate').mkdir(mode=0o700);gate=a.Gate(base/'gate');children=[]
 spec={'format':1,'name':'stable','members':{role:{'plugin_id':a.CANONICAL[role]+'.qualification-stable','profile':'qualification-stable','artifact_sha256':j.digest(['fixture',role]),'endpoint_ref':None,'paths':{kind:'/lab/shared/'+role+'/'+kind for kind in j.PATHS}} for role in j.ROLES}}
 permit={'id':uuid.uuid4().hex,'request':uuid.uuid4().hex,'epoch':uuid.uuid4().hex,'specification':spec,'shell':a.identity(os.getpid())}
 def emit(v):print(json.dumps(v),flush=True)
 try:
  with store.lease():
   gate.issue(permit);rows=[]
   for role in j.ROLES:
    member=spec['members'][role];expected={'permit':permit['id'],'epoch':permit['epoch'],'collection':j.digest(spec),**{k:member[k] for k in ('plugin_id','profile','artifact_sha256')}}
    child=subprocess.Popen([sys.executable,'-B','/lab/source/admission_guard.py','--root',str(gate.root)],stdin=subprocess.PIPE,stdout=subprocess.PIPE,stderr=subprocess.PIPE,text=True);children.append(child)
    send(child,json.dumps({'expected':expected,'role':role,'instance':uuid.uuid4().hex}));value=receive(child);assert value['ok'] and value['admission']['admitted'];rows.append(value['admission'])
   emit({'slot':slot,'permit':permit,'claims':rows,'coordinator_lease_held':True})
   while True:
    cmd=sys.stdin.readline().strip();assert cmd in ('status','close'),cmd
    if cmd=='status':emit(gate.status());continue
    gate.stop_admission(permit['id'],uuid.uuid4().hex)
    for child in children:
     assert receive(child)['admission']['state']=='closing';child.stdin.close();assert child.wait(timeout=5)==0
    value=gate.status();assert all(c['state']=='ended' for c in value['record']['claims'].values());assert not any(value['claim_owners_live'].values());emit(value);break
 finally:
  for child in children:
   if child.poll() is None:child.kill();child.wait(timeout=4)
  gate.close();store.close()
def inside():
 guard_namespace();sys.path.insert(0,'/lab/source');import journal as j
 children=[]
 try:
  for slot in ('a','b'):
   log=Path('/lab',slot+'.log').open('w');p=subprocess.Popen([sys.executable,'-B','/lab/driver.py','--worker',slot],stdin=subprocess.PIPE,stdout=subprocess.PIPE,stderr=log,text=True);children.append((p,log))
  a,b=[receive(p) for p,_ in children]
  assert a['permit']['specification']==b['permit']['specification'] and a['permit']['shell']!=b['permit']['shell']
  assert len(a['claims'])==len(b['claims'])==3
  shared=j.Store(Path('/lab/a/coordinator'))
  try:
   try:
    with shared.lease():raise AssertionError('same-root coordinator overlap admitted')
   except BlockingIOError:pass
  finally:shared.close()
  send(children[0][0],'close');closed=receive(children[0][0]);assert children[0][0].wait(timeout=5)==0
  send(children[1][0],'status');still=receive(children[1][0]);assert still['record']['state']=='open' and all(still['claim_owners_live'].values())
  send(children[1][0],'close');ended=receive(children[1][0]);assert children[1][0].wait(timeout=5)==0
  save(Path('/lab/result.json'),{'passed':True,'characterization_only':True,'cross_root_exclusion':False,'same_root_lease_refuses_competitor':True,'simultaneous_admitted_guards':6,'same_application_specification':True,'two_live_shell_owners':True,'close_a_leaves_b_admitted':True,'all_guards_ended':True,'first':a,'second':b,'first_closed':closed,'second_still_open':still,'second_closed':ended,'applications_started':False})
  print('PASS characterization: per-root leases exclude locally; separate roots admit six guards for the same specification')
 finally:
  for p,log in children:
   if p.poll() is None:p.kill();p.wait(timeout=4)
   log.close()
def main():
 p=argparse.ArgumentParser(description=__doc__);p.add_argument('--inside',action='store_true');p.add_argument('--worker');p.add_argument('--output',type=Path);args=p.parse_args()
 if args.worker:return worker(args.worker)
 if args.inside:return inside()
 out=args.output;assert out and out.parent==Path('/tmp') and out.name.startswith('omabit-admission-roots-') and not out.exists();out.mkdir(mode=0o700)
 for name in ('home','source','runtime','config','state','cache','data'):(out/name).mkdir(mode=0o700)
 root=Path(__file__).resolve().parents[1];hashes={}
 for name in ('admission.py','admission_guard.py','artifacts.py','journal.py'):
  src=root/'native/collections'/name;shutil.copy2(src,out/'source'/name);hashes[str(src)]=hashlib.sha256(src.read_bytes()).hexdigest()
 shutil.copy2(__file__,out/'driver.py');hashes[str(Path(__file__))]=hashlib.sha256(Path(__file__).read_bytes()).hexdigest();save(out/'source.json',hashes)
 save(out/'host-namespaces.json',{n:os.readlink('/proc/self/ns/'+n) for n in ('user','net','pid','mnt')})
 cmd=['bwrap','--unshare-all','--die-with-parent','--clearenv','--ro-bind','/usr','/usr','--symlink','usr/bin','/bin','--symlink','usr/lib','/lib','--symlink','usr/lib','/lib64','--proc','/proc','--dev','/dev','--tmpfs','/tmp','--bind',str(out),'/lab','--chdir','/lab']
 env={'HOME':'/lab/home','PATH':'/usr/bin:/bin','XDG_RUNTIME_DIR':'/lab/runtime','PYTHONDONTWRITEBYTECODE':'1','LANG':'C.UTF-8'}
 for name in ('CONFIG','STATE','CACHE','DATA'):env['XDG_'+name+'_HOME']='/lab/'+name.lower()
 for k,v in env.items():cmd+=['--setenv',k,v]
 cmd+=['--','/usr/bin/python3','-B','/lab/driver.py','--inside'];save(out/'command.json',cmd)
 q=subprocess.run(cmd,capture_output=True,text=True,timeout=45);(out/'driver.log').write_text(q.stdout+q.stderr);print(q.stdout+q.stderr);raise SystemExit(q.returncode)
if __name__=='__main__':main()
