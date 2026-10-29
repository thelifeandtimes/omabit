#!/usr/bin/env python3
"""Test copied Theme/DMs stores with a private locked/restarted Secret Service.

Creates a new owner-only lab, never touches installed plugins, host credentials,
ships or a Docker engine. Raw private keyring files remain in the lab.
"""
import argparse,hashlib,json,os,re,shutil,signal,subprocess,time,uuid
from pathlib import Path

def run(args):
    root=args.output.absolute()
    if root.parent!=Path('/tmp') or not re.fullmatch('omabit-keyring-lock-[A-Za-z0-9-]+',root.name) or root.exists() or root.is_symlink():raise ValueError('fresh private /tmp/omabit-keyring-lock-NAME required')
    os.umask(0o077);root.mkdir(mode=0o700)
    for name in ['home','runtime','cache','config','data','state','artifacts','etc']:(root/name).mkdir(mode=0o700)
    (root/'runtime/keyring').mkdir(mode=0o700)
    (root/'etc/passwd').write_text('lab:x:'+str(os.getuid())+':'+str(os.getgid())+':Private lab:/lab/home:/bin/sh\n');(root/'etc/group').write_text('lab:x:'+str(os.getgid())+':\n');(root/'etc/nsswitch.conf').write_text('passwd: files\ngroup: files\n')
    def sha(p):return hashlib.sha256(p.read_bytes()).hexdigest()
    source={}
    for profile in ('stable','dev'):
     for role in ('theme','dms'):
      src=(args.theme_source if role=='theme' else args.dms_source).resolve(strict=True);dst=root/'artifacts'/profile/role;dst.mkdir(parents=True)
      names=['client/support.py','client/instance.py','manifest.json'] if role=='theme' else ['session_store.py','instance.py','manifest.json']
      for name in names:
       p=src/name;assert p.is_file() and not p.is_symlink();target=dst/name;target.parent.mkdir(parents=True,exist_ok=True);shutil.copyfile(p,target);source[str(p)]=sha(p)
      m=json.loads((dst/'manifest.json').read_text());m['id']+='.keyring-'+profile;m['name']+=' (private keyring '+profile+')';(dst/'manifest.json').write_text(json.dumps(m)+'\n');(dst/'application-profile.json').write_text(json.dumps({'version':1,'profile':'keyring-'+profile})+'\n')
    shutil.copyfile(Path(__file__).with_name('keyring_lock_fixture.py'),root/'fixture.py')
    unit='omabit-keyring-lock-'+uuid.uuid4().hex
    cmd=['systemd-run','--user','--scope','--unit='+unit,'--property=MemoryMax=256M','--property=MemorySwapMax=0','--property=CPUQuota=100%','--property=TasksMax=64','--','bwrap','--unshare-all','--die-with-parent','--clearenv','--ro-bind','/usr','/usr','--symlink','usr/bin','/bin','--symlink','usr/lib','/lib','--symlink','usr/lib','/lib64','--proc','/proc','--dev','/dev','--tmpfs','/tmp','--ro-bind',str(root/'etc'),'/etc','--bind',str(root),'/lab','--ro-bind',str(root/'artifacts'),'/lab/artifacts','--chdir','/lab']
    env={'HOME':'/lab/home','XDG_RUNTIME_DIR':'/lab/runtime','XDG_CONFIG_HOME':'/lab/config','XDG_DATA_HOME':'/lab/data','XDG_STATE_HOME':'/lab/state','XDG_CACHE_HOME':'/lab/cache','PATH':'/usr/bin:/bin','LANG':'C.UTF-8','OMABIT_KEYRING_LAB':'private-v1','PYTHONDONTWRITEBYTECODE':'1'}
    for k,v in env.items():cmd+=['--setenv',k,v]
    cmd+=['--','/usr/bin/dbus-run-session','--','/usr/bin/python3','-B','/lab/fixture.py']
    (root/'inputs.json').write_text(json.dumps({'scope':unit+'.scope','command':cmd,'source_sha256':source,'fixture_sha256':sha(root/'fixture.py'),'namespaces':{n:os.readlink('/proc/self/ns/'+n) for n in ['user','net','pid','mnt','ipc']}},indent=2)+'\n')
    start=time.monotonic();admitted=False;timed_out=False
    with (root/'private-console.log').open('xb') as log:
     p=subprocess.Popen(cmd,stdout=log,stderr=subprocess.STDOUT,start_new_session=True)
     try:
      while p.poll() is None and time.monotonic()<start+8:
       q=subprocess.run(['systemctl','--user','show',unit+'.scope','-p','ActiveState,MemoryMax,MemorySwapMax,TasksMax,CPUQuotaPerSecUSec'],capture_output=True,text=True,timeout=5)
       seen=dict(row.split('=',1) for row in q.stdout.splitlines() if '=' in row)
       if seen.get('ActiveState')=='active':
        admitted=seen=={'ActiveState':'active','MemoryMax':str(256*1024**2),'MemorySwapMax':'0','TasksMax':'64','CPUQuotaPerSecUSec':'1s'};break
       time.sleep(.1)
      (root/'scope-admission.json').write_text(json.dumps(seen,indent=2)+'\n')
      while admitted and p.poll() is None:
       if time.monotonic()>start+180 or log.tell()>1024**2:timed_out=True;break
       time.sleep(.2)
     finally:
      if p.poll() is None:
       os.killpg(p.pid,signal.SIGTERM)
       try:p.wait(timeout=5)
       except subprocess.TimeoutExpired:os.killpg(p.pid,signal.SIGKILL);p.wait()
    q=subprocess.run(['systemctl','--user','show',unit+'.scope','-p','ActiveState','--value'],capture_output=True,text=True,timeout=5);inactive=q.returncode==0 and q.stdout.strip()=='inactive'
    result={'exit':p.returncode,'scope_admitted':admitted,'scope_inactive':inactive,'timed_out':timed_out,'seconds':time.monotonic()-start,'private_console_sha256':sha(root/'private-console.log'),'source_unchanged':all(sha(Path(k))==v for k,v in source.items())}
    (root/'outer-result.json').write_text(json.dumps(result,indent=2)+'\n');print(json.dumps(result),flush=True)
    raise SystemExit(0 if p.returncode==0 and admitted and inactive and not timed_out else 1)

def main():
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--theme-source',type=Path,required=True)
    parser.add_argument('--dms-source',type=Path,required=True)
    parser.add_argument('--output',type=Path,required=True)
    run(parser.parse_args())

if __name__=='__main__':main()
