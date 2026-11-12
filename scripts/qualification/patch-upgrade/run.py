#!/usr/bin/env python3
"""Run one offline release upgrade guest, reusing a verified pristine review base.

No arguments performs no work. Explicit paths and --run are required. The runner
never opens an existing guest overlay or any host Docker/runner/socket/keyring.
"""
import argparse,base64,hashlib,json,os,re,shutil,subprocess,time,sys
from pathlib import Path
ROOT=Path(__file__).resolve().parents[3]
GIB=1024**3
sys.path.insert(0,str(ROOT/"scripts/qualification/app-development"))
import storage

def run(args,**kw):return subprocess.run([str(a) for a in args],check=True,**kw)
def digest(p):
 with p.open('rb') as f:return hashlib.file_digest(f,'sha256').hexdigest()
def allocated(p):
 return storage.measure(p)['allocated_bytes']
def cpio(f,name,data,mode,ino):
 name=name.encode()+b'\0';fields=[ino,mode,0,0,1,0,len(data),0,0,0,0,len(name),0]
 head=b'070701'+b''.join(('%08x'%v).encode() for v in fields)
 f.write(head+name+b'\0'*((-(len(head)+len(name)))%4));f.write(data+b'\0'*((-len(data))%4))
def main():
 p=argparse.ArgumentParser(description=__doc__)
 for n in ('portable','previous','candidate','rootless','output'):p.add_argument('--'+n,type=Path)
 p.add_argument('--namespace-tests',type=Path)
 p.add_argument('--theme-source',type=Path)
 p.add_argument('--dms-source',type=Path)
 p.add_argument('--storage-budget-gib',type=int,choices=(40,48),default=40,help='48 requires explicit temporary operator authorization')
 p.add_argument('--run',action='store_true');a=p.parse_args()
 if not a.run:p.print_help();return
 if os.geteuid()==0:raise RuntimeError('Run as ordinary owner')
 if bool(a.theme_source)!=bool(a.dms_source):p.error('both receiver sources required')
 if any(x is None for x in (a.portable,a.previous,a.candidate,a.rootless,a.output)):p.error('all paths required')
 a.portable=a.portable.resolve();a.previous=a.previous.resolve();a.rootless=a.rootless.resolve();a.candidate=a.candidate.resolve()
 labs=Path.home()/'Documents/ChatGPT/omabit-private-docker-labs';evidence=labs.parent/'omabit-qualification-evidence'
 if a.output.parent.resolve()!=labs.resolve() or a.output.exists() or not re.fullmatch(r'patch-upgrade-[0-9]{3}-[a-z0-9-]+',a.output.name):p.error('fresh patch-upgrade-NNN-NAME in the existing labs root required')
 previous=json.loads((a.previous/'release.json').read_text())['version']
 target=json.loads((a.candidate/'release.json').read_text())['version']
 if not all(re.fullmatch(r'[0-9]+\.[0-9]+\.[0-9]+(?:-rc\.[0-9]+)?',v) for v in (previous,target)) or previous==target:p.error('distinct release versions required')
 budget=4*GIB;storage_limit=a.storage_budget_gib*GIB
 admission=storage.admit(budget,budget_bytes=storage_limit)
 used=admission['charged_bytes']
 meta=json.loads((a.portable/'bundle.json').read_text())
 for n in ('base.qcow2','boot/vmlinuz','boot/initramfs.cpio'):
  if (a.portable/n).is_symlink() or digest(a.portable/n)!=meta['sha256'][n]:raise RuntimeError('Pristine input hash differs: '+n)
 for v,path in [(previous,a.previous),(target,a.candidate)]:
  release=json.loads((path/'release.json').read_text());assert release['version']==v
  hashes={line.split('  ',1)[1]:line.split('  ',1)[0] for line in (path/'SHA256SUMS').read_text().splitlines()}
  for tool in ('omabit','urbit-collector','urbit-runner'):
   assert digest(path/(tool+'-linux-amd64'))==hashes[tool+'-linux-amd64']
 work=a.output;work.mkdir(mode=0o700);(work/'control').mkdir(mode=0o700)
 # No copies of live state. The payload consists only of verified public tools,
 # a fresh test script, one pinned OCI image and one pinned bootstrap pill.
 payload=work/'payload';payload.mkdir()
 for v,path in [(previous,a.previous),(target,a.candidate)]:
  dest=payload/v;dest.mkdir()
  for n in ('release.json','SHA256SUMS','omabit-linux-amd64','urbit-collector-linux-amd64','urbit-runner-linux-amd64'):shutil.copy2(path/n,dest/n)
  if v==previous:shutil.copytree(a.rootless,dest/'rootless-bundle')
  archive=path/('omabit-'+v+'-linux-amd64.tar.gz')
  asset=next(x for x in json.loads((path/'release.json').read_text())['assets'] if x['name']==archive.name)
  assert digest(archive)==asset['sha256'] and archive.stat().st_size==asset['bytes']
  os.link(archive,dest/archive.name)
 runtime=evidence/'2026-10-25-private-docker-code-cache/omabit-import-runtime-20261022-01'
 pill_hash=json.loads((a.candidate/'release.json').read_text())['pill']['sha256']
 pill=a.candidate/json.loads((a.candidate/'release.json').read_text())['pill']['name']
 assert digest(pill)==pill_hash
 os.link(pill,payload/'boot.pill')
 os.link(runtime/'runtime.oci.tar',payload/'runtime.oci.tar')
 for n in ('guest.py','shell.qml','desktop.py','fleet_guest.py','fleet_client.py'):shutil.copy2(Path(__file__).parent/n,payload/n)
 namespace_tests={}
 if a.namespace_tests:
  for name in ('pierstore.test','rootless.test'):
   source=a.namespace_tests/name;assert source.is_file() and not source.is_symlink()
   namespace_tests[name]=digest(source);shutil.copy2(source,payload/name)
 receiver_provenance={}
 if a.theme_source:
  plugins=payload/'fleet-plugins';plugins.mkdir()
  for source,plugin in [(a.theme_source,'thelifeandtimes.urbit-theme'),(a.dms_source,'urbit.dms')]:
   source=source.resolve();dest=plugins/(plugin+'.qualification040');dest.mkdir()
   declaration=json.loads((source/'application-connection.json').read_text());assert declaration['version']==2
   names=set(declaration['files'])|{'manifest.json','application-connection.json'}
   receiver_provenance[plugin]={'head':subprocess.check_output(['git','rev-parse','HEAD'],cwd=source,text=True).strip(),'files':{}}
   for name in sorted(names):
    relative=Path(name);assert not relative.is_absolute() and '..' not in relative.parts
    original=source/relative;assert original.is_file() and not original.is_symlink()
    copied=dest/relative;copied.parent.mkdir(parents=True,exist_ok=True);shutil.copy2(original,copied)
    receiver_provenance[plugin]['files'][name]=digest(original)
   manifest=json.loads((dest/'manifest.json').read_text());manifest.update(id=dest.name,name=manifest['name']+' (Release qualification)')
   (dest/'manifest.json').write_text(json.dumps(manifest)+'\n')
   (dest/'application-profile.json').write_text(json.dumps({'version':1,'profile':'qualification040'})+'\n')
 shutil.copytree(ROOT/'native/bootstrap',payload/'bootstrap',ignore=shutil.ignore_patterns('__pycache__'))
 (payload/'scenario.json').write_text(json.dumps({'desktop':previous,'collector':previous,'runner':previous,'target':target,'fleet':bool(a.theme_source),'namespace_tests':bool(a.namespace_tests)}))
 disk=work/'payload.ext4'
 with disk.open('xb') as f:f.truncate(768*1024**2)
 run(['mkfs.ext4','-q','-F','-d',payload,disk],stdout=subprocess.DEVNULL)
 # Staging files are generated copies/hardlinks; the immutable payload disk is
 # now complete. This is the only automatic removal in the harness.
 shutil.rmtree(payload)
 init=work/'init';env=dict(os.environ,CGO_ENABLED='0')
 run(['go','build','-trimpath','-o',init,'./scripts/qualification/release-upgrade/init'],cwd=ROOT,env=env)
 boot=work/'boot';boot.mkdir()
 # Extract only the existing module list/modules to a private staging directory.
 run(['bsdtar','-xf',a.portable/'boot/initramfs.cpio','-C',boot,'guest-modules','modules'])
 with (work/'initramfs.cpio').open('xb') as out:
  ino=0
  for n in ('proc','sys','dev','newroot','modules'):ino+=1;cpio(out,n,b'',0o40755,ino)
  for path in [init,boot/'guest-modules',*sorted((boot/'modules').glob('*.ko'))]:
   ino+=1;n='init' if path==init else str(path.relative_to(boot));cpio(out,n,path.read_bytes(),0o100755 if n=='init' else 0o100644,ino)
  cpio(out,'TRAILER!!!',b'',0,ino+1)
 tools=a.portable/'tools/qemu';tenv=dict(os.environ,LD_LIBRARY_PATH=str(tools/'usr/lib'),QEMU_MODULE_DIR=str(tools/'usr/lib/qemu'))
 base_info=json.loads(subprocess.check_output([tools/'usr/bin/qemu-img','info','--output=json',a.portable/'base.qcow2'],env=tenv,text=True))
 assert base_info['format']=='qcow2' and not base_info.get('backing-filename')
 run([tools/'usr/bin/qemu-img','create','-q','-u','-f','qcow2','-F','qcow2','-b','/base.qcow2',work/'root.qcow2',str(base_info['virtual-size'])],env=tenv)
 cmd=['systemd-run','--user','--scope','--unit=omabit-'+work.name,'-p','MemoryMax=10G','-p','MemorySwapMax=0','-p','CPUQuota=300%','--',
 'bwrap','--unshare-all','--die-with-parent','--clearenv','--ro-bind','/usr','/usr','--symlink','usr/bin','/bin','--symlink','usr/lib','/lib','--symlink','usr/lib','/lib64','--proc','/proc','--dev','/dev','--dev-bind','/dev/kvm','/dev/kvm','--tmpfs','/tmp','--dir','/run','--ro-bind',str(tools),'/tools','--ro-bind',str(a.portable/'base.qcow2'),'/base.qcow2','--ro-bind',str(a.portable/'boot/vmlinuz'),'/vmlinuz','--bind',str(work),'/lab','--chdir','/lab','--setenv','HOME','/lab','--setenv','PATH','/usr/bin:/bin','--setenv','LD_LIBRARY_PATH','/tools/usr/lib','--setenv','QEMU_MODULE_DIR','/tools/usr/lib/qemu','--','/tools/usr/bin/qemu-system-x86_64','-accel','kvm','-cpu','host','-machine','q35','-m','8G','-smp','3','-nodefaults','-no-user-config','-display','none','-monitor','none','-nic','none','-no-reboot','-L','/tools/usr/share/qemu','-bios','/tools/usr/share/qemu/bios-256k.bin','-kernel','/vmlinuz','-initrd','/lab/initramfs.cpio','-append','console=ttyS0 panic=-1 quiet rdinit=/init omabit.disposable-guest=release-upgrade-v1 systemd.unit=multi-user.target','-drive','file=/lab/root.qcow2,format=qcow2,if=none,id=rootdisk','-device','virtio-blk-pci,drive=rootdisk','-drive','file=/lab/payload.ext4,format=raw,if=none,id=payload,readonly=on','-device','virtio-blk-pci,drive=payload','-qmp','unix:/lab/control/qmp.sock,server=on,wait=off','-serial','stdio']
 receipt={'storage_admission':admission,'receiver_sources':receiver_provenance,'namespace_tests':namespace_tests,'harness_sha256':{n:digest(Path(__file__).parent/n) for n in ('run.py','guest.py','desktop.py','shell.qml','fleet_guest.py','fleet_client.py')},'scenario':previous+' desktop and services to '+target,'source_commit':subprocess.check_output(['git','rev-parse','HEAD'],cwd=ROOT,text=True).strip(),'used_before':used,'scratch_limit':budget,'base_sha256':meta['sha256']['base.qcow2'],'base_path':str(a.portable/'base.qcow2'),'network':'none','host_mounts':'OS files, QEMU, immutable base and fresh lab only'}
 (work/'host.json').write_text(json.dumps(receipt,indent=2)+'\n')
 with (work/'serial.log').open('wb') as log:
  proc=subprocess.Popen(cmd,stdout=log,stderr=subprocess.STDOUT)
  started=time.monotonic()
  try:
   while proc.poll() is None:
    time.sleep(10)
    storage.admit(0,budget_bytes=storage_limit)
    if allocated(work)>budget or (work/'serial.log').stat().st_size>32*1024**2 or time.monotonic()-started>2700:
     raise RuntimeError('Guest budget/deadline exceeded; retained bounded failure evidence')
  except BaseException as error:
   receipt['interruption']=str(error)
   (work/'host.json').write_text(json.dumps(receipt,indent=2)+'\n')
   if proc.poll() is None:
    subprocess.run(['systemctl','--user','kill','--signal=TERM','omabit-'+work.name+'.scope'],check=True)
    proc.wait(timeout=15)
   raise
  receipt['exit']=proc.returncode
 receipt['used_after']=storage.admit(0,budget_bytes=storage_limit)['charged_bytes']
 (work/'host.json').write_text(json.dumps(receipt,indent=2)+'\n')
 lines=(work/'serial.log').read_text(errors='replace').splitlines()
 for line in lines:
  if 'OMABIT_GUIDED_IMAGE ' in line:
   data=json.loads(line.split('OMABIT_GUIDED_IMAGE ',1)[1])
   assert data['name'] in ('offer','retry','complete')
   png=base64.b64decode(data['png']);assert len(png)<200000
   (work/(data['name']+'.png')).write_bytes(png)
 result=next((json.loads(x.split('OMABIT_RELEASE_UPGRADE_RESULT ',1)[1]) for x in reversed(lines) if 'OMABIT_RELEASE_UPGRADE_RESULT ' in x),None)
 (work/'result.json').write_text(json.dumps(result,indent=2)+'\n')
 print(json.dumps({'passed':bool(result and result.get('passed')),'output':str(work),'error':result.get('error') if result else 'No completed guest result'}))
 if not result or not result.get('passed'):raise SystemExit(1)
if __name__=='__main__':main()
