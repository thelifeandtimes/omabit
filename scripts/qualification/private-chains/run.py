#!/usr/bin/env python3
"""Opt-in private chain qualification. No source fetching or access to existing chains or ships; native-peers creates fresh isolated ships."""
import argparse,hashlib,json,os,re,shutil,subprocess,time,uuid
from pathlib import Path
HERE=Path(__file__).resolve().parent

def sha(p):
 with p.open('rb') as f:return hashlib.file_digest(f,'sha256').hexdigest()

def main():
 p=argparse.ArgumentParser(description=__doc__)
 p.add_argument('--stage',choices=['compile','deploy','boot-requests','key-validation','native-peers'],required=True);p.add_argument('--sources',type=Path,required=True);p.add_argument('--tools',type=Path,required=True);p.add_argument('--compiled',type=Path);p.add_argument('--vere',type=Path);p.add_argument('--deployment',type=Path);p.add_argument('--pill',type=Path);p.add_argument('--output',type=Path,required=True);a=p.parse_args()
 out=a.output.absolute()
 if out.parent!=Path('/tmp') or not re.fullmatch('omabit-private-chain-[A-Za-z0-9-]+',out.name) or out.exists():p.error('fresh /tmp/omabit-private-chain-NAME output required')
 if (a.stage in ('deploy','key-validation','native-peers'))!=(a.compiled is not None):p.error('deploy/key-validation/native-peers require --compiled')
 src=a.sources.resolve(strict=True);tools=a.tools.resolve(strict=True);pins=json.loads((HERE/'inputs.json').read_text())
 for rel,record in pins['sources'].items():
  f=src/rel
  if not f.is_file() or f.is_symlink() or sha(f)!=record['sha256']:raise ValueError('source mismatch: '+rel)
 if sha(tools/'package-lock.json')!=pins['package_lock_sha256']:raise ValueError('dependency lock mismatch')
 for name,version in pins['packages'].items():
  if json.loads((tools/'node_modules'/name/'package.json').read_text())['version']!=version:raise ValueError('package version mismatch: '+name)
 pill=None
 if (a.stage=='native-peers')!=(a.pill is not None):p.error('only native-peers requires --pill')
 if a.pill:
  pill=a.pill.resolve(strict=True)
  if pill.stat().st_size!=217421172 or sha(pill)!='f8b08c53dd6d0aa9c4f7654f0a7bcd11e94089222888a3c99c0a10eedcbb377b':raise ValueError('bootstrap pill mismatch')
 deployment=None
 if (a.stage in ('key-validation','native-peers'))!=(a.deployment is not None):p.error('key-validation/native-peers require --deployment')
 if a.deployment:
  deployment=a.deployment.resolve(strict=True)
  if not json.loads((deployment/'result.json').read_text())['passed'] or not json.loads((deployment/'outer-result.json').read_text())['scope_inactive']:raise ValueError('inactive verified deployment required')
 vere=None
 if (a.stage in ('boot-requests','key-validation','native-peers'))!=(a.vere is not None):p.error('boot-requests/key-validation/native-peers require --vere')
 if a.vere:
  vere=a.vere.resolve(strict=True)
  if sha(vere)!='b5c54439ae5b9f0ff94d75319e67e04f04369776fa5e2180a5ab1a5ebd4c2ff0':raise ValueError('Vere binary mismatch')
 compiled=None
 if a.compiled:
  compiled=a.compiled.resolve(strict=True);j=json.loads((compiled/'result.json').read_text())
  if not j['passed'] or sha(compiled/'compiler-output.json')!=j['compilerOutputSha256']:raise ValueError('unverified compilation')
 if a.stage=='native-peers':
  mem={line.split(':',1)[0]:int(line.split()[1])*1024 for line in Path('/proc/meminfo').read_text().splitlines() if line.startswith('MemAvailable:')}
  if mem.get('MemAvailable',0)<8*1024**3:raise RuntimeError('two-peer fixture requires 8GiB available before its 6GiB cap')
 os.umask(0o077);out.mkdir(mode=0o700);(out/'home').mkdir(mode=0o700)
 token='private-chain-'+{'compile':'compile','deploy':'deployment','boot-requests':'boot-requests','key-validation':'key-validation','native-peers':'peers'}[a.stage]+'-v1'
 (out/'marker').write_text(token+'\n');(out/'host-namespaces.json').write_text(json.dumps({n:os.readlink('/proc/self/ns/'+n) for n in ('user','net','pid','mnt')}))
 if a.stage=='native-peers':
  helpers={'peer_probe.py':HERE/'native_peer.py',**{name:HERE.parent/'fake-networks'/name for name in ('thread_probe.py','network_case.py','state_case.py','omabit-state.hoon')}}
  if sha(helpers['omabit-state.hoon'])!='110a20b47c2e34eca322eda691bb9cb1897c305156e3dc13f02cf4ac1fcd042f':raise ValueError('persistent state fixture mismatch')
  manifest={}
  for name,path in helpers.items():
   if not path.is_file() or path.is_symlink():raise ValueError('regular native-peer helper required')
   shutil.copy2(path,out/name);manifest[name]={'source':str(path),'sha256':sha(path)}
  (out/'native-inputs.json').write_text(json.dumps(manifest,indent=2)+'\n')
 shutil.copy2(HERE/(a.stage+'.cjs'),out/'driver.cjs');shutil.copy2(__file__,out/'outer-driver.py');shutil.copy2(HERE/'inputs.json',out/'inputs.json')
 gate="""import json,os,time
from pathlib import Path
assert not Path('/home').exists() and not Path('/run/user').exists()
for _ in range(200):
 p=Path('/lab/scope-admission.json')
 if p.exists():
  assert json.loads(p.read_text())['admitted']
  os.execv('/usr/bin/node',['node','/lab/driver.cjs'])
 time.sleep(.05)
raise RuntimeError('scope admission unavailable')
"""
 (out/'scope-gate.py').write_text(gate)
 cmd=['bwrap','--unshare-all','--die-with-parent','--clearenv','--ro-bind','/usr','/usr','--symlink','usr/bin','/bin','--symlink','usr/lib','/lib','--symlink','usr/lib','/lib64','--proc','/proc','--dev','/dev','--tmpfs','/tmp','--ro-bind',str(src),'/source','--ro-bind',str(tools),'/tools','--bind',str(out),'/lab','--chdir','/lab','--setenv','PATH','/usr/bin:/bin','--setenv','HOME','/lab/home','--setenv','PYTHONDONTWRITEBYTECODE','1','--setenv','OMABIT_PRIVATE_CHAIN',token]
 if pill:cmd+=['--ro-bind',str(pill),'/boot.pill','--dir','/artifacts','--ro-bind',str(out/'omabit-state.hoon'),'/artifacts/omabit-state.hoon']
 if deployment:cmd+=['--ro-bind',str(deployment),'/deployment']
 if vere:cmd+=['--ro-bind',str(vere),'/vere']
 if compiled:cmd+=['--ro-bind',str(compiled),'/compiled']
 cmd+=['--','/usr/bin/python3','-B','/lab/scope-gate.py']
 unit='omabit-private-chain-'+uuid.uuid4().hex
 outer=['systemd-run','--user','--scope','--quiet','--unit',unit,'-p',('MemoryMax=6G' if a.stage=='native-peers' else 'MemoryMax=4G'),'-p','MemorySwapMax=0','-p','CPUQuota=100%','-p','TasksMax=256','timeout','--signal=TERM','--kill-after=5',('900' if a.stage=='native-peers' else '180')]+cmd
 (out/'command.json').write_text(json.dumps({'unit':unit,'command':outer,'node_sha256':sha(Path('/usr/bin/node'))},indent=2)+'\n')
 with (out/'driver.log').open('w') as f:
  proc=subprocess.Popen(outer,stdout=f,stderr=subprocess.STDOUT)
  for _ in range(100):
   r=subprocess.run(['systemctl','--user','show',unit+'.scope','-p','ActiveState','-p','MemoryMax','-p','MemorySwapMax','-p','CPUQuotaPerSecUSec','-p','TasksMax'],capture_output=True,text=True,timeout=3)
   actual=dict(line.split('=',1) for line in r.stdout.splitlines() if '=' in line)
   if actual.get('ActiveState')=='active':break
   if proc.poll() is not None:break
   time.sleep(.05)
  expected={'ActiveState':'active','MemoryMax':str((6 if a.stage=='native-peers' else 4)*1024**3),'MemorySwapMax':'0','CPUQuotaPerSecUSec':'1s','TasksMax':'256'}
  admitted=actual==expected
  if admitted:
   tmp=out/'scope-admission.tmp';tmp.write_text(json.dumps({'admitted':True,'observed':actual}));tmp.rename(out/'scope-admission.json')
  code=proc.wait(timeout=920 if a.stage=='native-peers' else 200)
 state=subprocess.run(['systemctl','--user','is-active',unit+'.scope'],capture_output=True,text=True,timeout=5).stdout.strip()
 result={'exit':code,'scope_admitted':admitted,'scope_inactive':state in ('inactive','failed','unknown')}
 (out/'outer-result.json').write_text(json.dumps(result,indent=2)+'\n');print(json.dumps(result));print((out/'driver.log').read_text()[-2000:])
 if code or not admitted or not result['scope_inactive']:raise SystemExit(1)

if __name__=='__main__':main()
