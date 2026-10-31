#!/usr/bin/env python3
"""Run one offline release upgrade guest, reusing a verified pristine review base.

No arguments performs no work. Explicit paths and --run are required. The runner
never opens an existing guest overlay or any host Docker/runner/socket/keyring.
"""
import argparse,base64,hashlib,json,os,re,shutil,subprocess,time
from pathlib import Path
ROOT=Path(__file__).resolve().parents[3]
GIB=1024**3
MEASUREMENT_LIMITS={}

def run(args,**kw):return subprocess.run([str(a) for a in args],check=True,**kw)
def digest(p):
 with p.open('rb') as f:return hashlib.file_digest(f,'sha256').hexdigest()
def allocated(p):
 r=subprocess.run(['du','-sx','-B1',str(p)],capture_output=True,text=True)
 if not r.stdout:raise RuntimeError('Cannot measure '+str(p))
 visible=int(r.stdout.split()[0])
 if r.returncode:
  # Rootless overlay work directories are inaccessible to the ordinary owner.
  # Record the incomplete count explicitly and reserve extra budget for them.
  errors=r.stderr.splitlines()
  pattern=r"du: cannot read directory '/srv/dev/urbit/state/\.local/share/urbit-runner/docker/containerd/daemon/io\.containerd\.snapshotter\.v1\.overlayfs/snapshots/[0-9]+/work/work': Permission denied"
  if str(p)!='/srv/dev/urbit' or not errors or not all(re.fullmatch(pattern,x) for x in errors):
   raise RuntimeError('Cannot fully measure '+str(p)+': '+r.stderr)
  MEASUREMENT_LIMITS[str(p)]={'visible_bytes':visible,'unreadable_work_directories':len(errors),'unmeasured_reserve':4*GIB}
  return visible+4*GIB
 return visible
def cpio(f,name,data,mode,ino):
 name=name.encode()+b'\0';fields=[ino,mode,0,0,1,0,len(data),0,0,0,0,len(name),0]
 head=b'070701'+b''.join(('%08x'%v).encode() for v in fields)
 f.write(head+name+b'\0'*((-(len(head)+len(name)))%4));f.write(data+b'\0'*((-len(data))%4))
def main():
 p=argparse.ArgumentParser(description=__doc__)
 for n in ('portable','previous','candidate','rootless','output'):p.add_argument('--'+n,type=Path)
 p.add_argument('--run',action='store_true');a=p.parse_args()
 if not a.run:p.print_help();return
 if os.geteuid()==0:raise RuntimeError('Run as ordinary owner')
 if any(x is None for x in (a.portable,a.previous,a.candidate,a.rootless,a.output)):p.error('all paths required')
 a.portable=a.portable.resolve();a.previous=a.previous.resolve();a.rootless=a.rootless.resolve();a.candidate=a.candidate.resolve()
 labs=Path.home()/'Documents/ChatGPT/omabit-private-docker-labs';evidence=labs.parent/'omabit-qualification-evidence'
 if a.output.parent.resolve()!=labs.resolve() or a.output.exists() or not re.fullmatch(r'patch-upgrade-034-[a-z0-9-]+',a.output.name):p.error('fresh patch-upgrade-034-NAME in the existing labs root required')
 roots=[Path('/srv/dev/urbit'),labs,evidence]
 used=sum(allocated(x) for x in roots);budget=4*GIB
 if used+budget>40*GIB:raise RuntimeError('Insufficient headroom for declared 4 GiB scratch cap')
 meta=json.loads((a.portable/'bundle.json').read_text())
 for n in ('base.qcow2','boot/vmlinuz','boot/initramfs.cpio'):
  if (a.portable/n).is_symlink() or digest(a.portable/n)!=meta['sha256'][n]:raise RuntimeError('Pristine input hash differs: '+n)
 for v,path in [('0.3.3',a.previous),('0.3.4',a.candidate)]:
  release=json.loads((path/'release.json').read_text());assert release['version']==v
  hashes={line.split('  ',1)[1]:line.split('  ',1)[0] for line in (path/'SHA256SUMS').read_text().splitlines()}
  for tool in ('omabit','urbit-collector','urbit-runner'):
   assert digest(path/(tool+'-linux-amd64'))==hashes[tool+'-linux-amd64']
 work=a.output;work.mkdir(mode=0o700);(work/'control').mkdir(mode=0o700)
 # No copies of live state. The payload consists only of verified public tools,
 # a fresh test script, one pinned OCI image and one pinned bootstrap pill.
 payload=work/'payload';payload.mkdir()
 for v,path in [('0.3.3',a.previous),('0.3.4',a.candidate)]:
  dest=payload/v;dest.mkdir()
  for n in ('release.json','SHA256SUMS','omabit-linux-amd64','urbit-collector-linux-amd64','urbit-runner-linux-amd64'):shutil.copy2(path/n,dest/n)
  if v=='0.3.3':shutil.copytree(a.rootless,dest/'rootless-bundle')
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
 for n in ('guest.py','shell.qml','desktop.py'):shutil.copy2(Path(__file__).parent/n,payload/n)
 shutil.copytree(ROOT/'native/bootstrap',payload/'bootstrap',ignore=shutil.ignore_patterns('__pycache__'))
 (payload/'scenario.json').write_text(json.dumps({'desktop':'0.3.3','collector':'0.3.3','runner':'0.3.3'}))
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
 cmd=['systemd-run','--user','--scope','--unit=omabit-patch-upgrade-034','-p','MemoryMax=10G','-p','MemorySwapMax=0','-p','CPUQuota=300%','--',
 'bwrap','--unshare-all','--die-with-parent','--clearenv','--ro-bind','/usr','/usr','--symlink','usr/bin','/bin','--symlink','usr/lib','/lib','--symlink','usr/lib','/lib64','--proc','/proc','--dev','/dev','--dev-bind','/dev/kvm','/dev/kvm','--tmpfs','/tmp','--dir','/run','--ro-bind',str(tools),'/tools','--ro-bind',str(a.portable/'base.qcow2'),'/base.qcow2','--ro-bind',str(a.portable/'boot/vmlinuz'),'/vmlinuz','--bind',str(work),'/lab','--chdir','/lab','--setenv','HOME','/lab','--setenv','PATH','/usr/bin:/bin','--setenv','LD_LIBRARY_PATH','/tools/usr/lib','--setenv','QEMU_MODULE_DIR','/tools/usr/lib/qemu','--','/tools/usr/bin/qemu-system-x86_64','-accel','kvm','-cpu','host','-machine','q35','-m','8G','-smp','3','-nodefaults','-no-user-config','-display','none','-monitor','none','-nic','none','-no-reboot','-L','/tools/usr/share/qemu','-bios','/tools/usr/share/qemu/bios-256k.bin','-kernel','/vmlinuz','-initrd','/lab/initramfs.cpio','-append','console=ttyS0 panic=-1 quiet rdinit=/init omabit.disposable-guest=release-upgrade-v1 systemd.unit=multi-user.target','-drive','file=/lab/root.qcow2,format=qcow2,if=none,id=rootdisk','-device','virtio-blk-pci,drive=rootdisk','-drive','file=/lab/payload.ext4,format=raw,if=none,id=payload,readonly=on','-device','virtio-blk-pci,drive=payload','-qmp','unix:/lab/control/qmp.sock,server=on,wait=off','-serial','stdio']
 receipt={'measurement_limits':MEASUREMENT_LIMITS,'harness_sha256':{n:digest(Path(__file__).parent/n) for n in ('run.py','guest.py','desktop.py','shell.qml')},'scenario':'0.3.3 desktop and services to 0.3.4','source_commit':subprocess.check_output(['git','rev-parse','HEAD'],cwd=ROOT,text=True).strip(),'used_before':used,'scratch_limit':budget,'base_sha256':meta['sha256']['base.qcow2'],'base_path':str(a.portable/'base.qcow2'),'network':'none','host_mounts':'OS files, QEMU, immutable base and fresh lab only'}
 (work/'host.json').write_text(json.dumps(receipt,indent=2)+'\n')
 with (work/'serial.log').open('wb') as log:
  proc=subprocess.Popen(cmd,stdout=log,stderr=subprocess.STDOUT)
  started=time.monotonic()
  while proc.poll() is None:
   time.sleep(10)
   if allocated(work)>budget or sum(allocated(x) for x in roots)>40*GIB or (work/'serial.log').stat().st_size>32*1024**2 or time.monotonic()-started>2700:
    subprocess.run(['systemctl','--user','kill','--signal=TERM','omabit-patch-upgrade-034.scope'],check=True);proc.wait(timeout=15);raise RuntimeError('Guest budget/deadline exceeded; retained bounded failure evidence')
  receipt['exit']=proc.returncode
 receipt['used_after']=sum(allocated(x) for x in roots)
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
