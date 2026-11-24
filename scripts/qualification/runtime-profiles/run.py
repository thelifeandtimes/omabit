#!/usr/bin/env python3
"""Run core runtime-profile or previous-stable upgrade checks in one offline VM.

No arguments performs no work. Explicit paths and --run are required. The runner
never opens an existing guest overlay or any host Docker/runner/socket/keyring.
"""
import argparse,base64,errno,fcntl,hashlib,importlib.util,json,os,re,shutil,signal,subprocess,time,sys
from pathlib import Path
ROOT=Path(__file__).resolve().parents[3]
HERE=Path(__file__).resolve().parent
SUPPORT=HERE.parent/'patch-upgrade'
GIB=1024**3
sys.path.insert(0,str(ROOT/"scripts/qualification/app-development"))
import storage

def run(args,**kw):return subprocess.run([str(a) for a in args],check=True,**kw)
def digest(p):
 with p.open('rb') as f:return hashlib.file_digest(f,'sha256').hexdigest()
def stage_immutable(source,destination):
 if source.is_symlink() or not source.is_file() or destination.exists():raise RuntimeError('unsafe immutable staging input')
 before=digest(source)
 try:os.link(source,destination)
 except OSError as error:
  if error.errno!=errno.EXDEV:raise
  with source.open('rb') as src,destination.open('xb') as dst:shutil.copyfileobj(src,dst)
 if digest(destination)!=before or digest(source)!=before:raise RuntimeError('immutable staging input changed')
def allocated(p):
 return storage.measure(p)['allocated_bytes']
def cpio(f,name,data,mode,ino):
 name=name.encode()+b'\0';fields=[ino,mode,0,0,1,0,len(data),0,0,0,0,len(name),0]
 head=b'070701'+b''.join(('%08x'%v).encode() for v in fields)
 f.write(head+name+b'\0'*((-(len(head)+len(name)))%4));f.write(data+b'\0'*((-len(data))%4))
def full_charge(additional,cap):
 # Import lazily: issues-recovery shares only this module's inert file helpers.
 spec=importlib.util.spec_from_file_location('release_upgrade_budget',ROOT/'scripts/qualification/issues-recovery/run.py')
 budget_module=importlib.util.module_from_spec(spec);spec.loader.exec_module(budget_module)
 return budget_module.charge(additional,cap)
def verify_boot(path):
 if path.is_symlink() or not path.is_file() or path.resolve()!=path.absolute() or digest(path)!='63a86300f9ba7ca08bf7df1fe28eb8b1b070d93ab67e868c7a51194e6bce1ec7':raise RuntimeError('exact separately frozen PID1 required')
 if digest(ROOT/'scripts/qualification/release-upgrade/init/main.go')!='1327c9480aab4a0d6287ec6a2cf86609e6d8182045e59abc811ffaebe2baf643':raise RuntimeError('frozen PID1 source changed')
 info=subprocess.check_output(['go','version','-m',path],text=True)
 if '\tvcs.revision=d2bb3ba6c152396c9938c44a28cf90a0c902de72' not in info or '\tvcs.modified=false' not in info:raise RuntimeError('frozen PID1 provenance differs')
 return {'sha256':digest(path),'source_sha256':digest(ROOT/'scripts/qualification/release-upgrade/init/main.go'),'build_info':info}
original_charge=full_charge
def full_charge(additional,cap):
 report=original_charge(additional,cap)
 extra=Path.home()/'.config/omarchy/plugins/thelifeandtimes.omabit.review040-v050'
 if not any(row['path']==str(extra) for row in report['measurements']):report['measurements'].append(storage.measure(extra))
 return storage.admit(additional,report['measurements'],budget_bytes=cap)
def read_report(lines):
 frames={};end=None
 for line in lines:
  if 'OMABIT_PROFILE_REPORT_CHUNK ' in line:
   frame=json.loads(line.split('OMABIT_PROFILE_REPORT_CHUNK ',1)[1])
   if frame['i'] in frames:raise RuntimeError('duplicate framed guest evidence')
   frames[frame['i']]=frame['data']
  if 'OMABIT_PROFILE_REPORT_END ' in line:
   if end is not None:raise RuntimeError('duplicate guest evidence footer')
   end=json.loads(line.split('OMABIT_PROFILE_REPORT_END ',1)[1])
 if not end or len(frames)!=end['chunks']:raise RuntimeError('incomplete framed guest evidence')
 raw=base64.b64decode(''.join(frames[i] for i in range(end['chunks'])),validate=True)
 if hashlib.sha256(raw).hexdigest()!=end['sha256']:raise RuntimeError('framed evidence hash mismatch')
 return json.loads(raw)

def validate_frozen_changes(changed):
 sys.path.insert(0,str(ROOT/'scripts'));from release_policy import validate_distribution_changes
 validate_distribution_changes([n for n in changed if not n.startswith('scripts/qualification/runtime-profiles/')])

def main():
 p=argparse.ArgumentParser(description=__doc__)
 for n in ('portable','previous','candidate','rootless','output','init'):p.add_argument('--'+n,type=Path)
 p.add_argument('--case',choices=('runtime','upgrade'),default='runtime')
 p.add_argument('--namespace-tests',type=Path)
 p.add_argument('--app-workflow',action='store_true',help='create original stable app history before upgrade and verify preservation')
 p.add_argument('--theme-source',type=Path)
 p.add_argument('--dms-source',type=Path)
 p.add_argument('--storage-budget-gib',type=int,choices=(40,48),default=40,help='48 requires explicit temporary operator authorization')
 p.add_argument('--run',action='store_true');a=p.parse_args()
 if not a.run:p.print_help();return
 if int(next(x.split()[1] for x in Path('/proc/meminfo').read_text().splitlines() if x.startswith('MemAvailable:')))*1024<12*GIB:raise RuntimeError('12GiB memory headroom required')
 if os.geteuid()==0:raise RuntimeError('Run as ordinary owner')
 if bool(a.theme_source)!=bool(a.dms_source):p.error('both receiver sources required')
 if any(x is None for x in (a.portable,a.previous,a.candidate,a.rootless,a.output,a.init)):p.error('all paths required')
 a.portable=a.portable.resolve();a.previous=a.previous.resolve();a.rootless=a.rootless.resolve();a.candidate=a.candidate.resolve()
 labs=Path.home()/'Documents/ChatGPT/omabit-private-docker-labs';evidence=labs.parent/'omabit-qualification-evidence'
 if a.output.parent.resolve()!=labs.resolve() or a.output.exists() or not re.fullmatch(r'patch-upgrade-[0-9]{3}-[a-z0-9-]+',a.output.name):p.error('fresh patch-upgrade-NNN-NAME in the existing labs root required')
 previous=json.loads((a.previous/'release.json').read_text())['version']
 target=json.loads((a.candidate/'release.json').read_text())['version']
 if not all(re.fullmatch(r'[0-9]+\.[0-9]+\.[0-9]+(?:-rc\.[0-9]+)?',v) for v in (previous,target)) or previous==target:p.error('distinct release versions required')
 if subprocess.check_output(['git','status','--porcelain'],cwd=ROOT):raise RuntimeError('clean release qualification source required')
 source=json.loads((a.candidate/'release.json').read_text())['source_commit']
 if not re.fullmatch('[0-9a-f]{40}',source):raise RuntimeError('exact candidate source required')
 subprocess.run(['git','merge-base','--is-ancestor',source,'HEAD'],cwd=ROOT,check=True)
 # Only this private harness and normal distribution documentation may differ.
 # Runtime, installer, UI, bootstrap and package inputs remain frozen.
 changed=subprocess.check_output(['git','diff','--name-only',source,'HEAD'],cwd=ROOT,text=True).splitlines()
 validate_frozen_changes(changed)
 boot_provenance=verify_boot(a.init)
 lock=os.open('/tmp/omabit-issues-vm-runtime.lock',os.O_RDWR|os.O_CREAT|os.O_NOFOLLOW,0o600);fcntl.flock(lock,fcntl.LOCK_EX|fcntl.LOCK_NB)
 budget=4*GIB;storage_limit=a.storage_budget_gib*GIB
 admission=full_charge(budget,storage_limit)
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
  # Downloaded release assets need not carry executable Unix mode bits. Change
  # only the guest staging copies; their verified content remains identical.
  for tool in ('omabit','urbit-collector','urbit-runner'):(dest/(tool+'-linux-amd64')).chmod(0o755)
  shutil.copytree(a.rootless,dest/'rootless-bundle')
  archive=path/('omabit-'+v+'-linux-amd64.tar.gz')
  asset=next(x for x in json.loads((path/'release.json').read_text())['assets'] if x['name']==archive.name)
  assert digest(archive)==asset['sha256'] and archive.stat().st_size==asset['bytes']
  stage_immutable(archive,dest/archive.name)
 runtime=evidence/'2026-10-25-private-docker-code-cache/omabit-import-runtime-20261022-01'
 pill_hash=json.loads((a.candidate/'release.json').read_text())['pill']['sha256']
 pill=a.candidate/json.loads((a.candidate/'release.json').read_text())['pill']['name']
 assert digest(pill)==pill_hash
 stage_immutable(pill,payload/'boot.pill')
 stage_immutable(runtime/'runtime.oci.tar',payload/'runtime.oci.tar')
 stage_immutable(a.candidate/'vere-4.6-jsonfix-0774857c.tar.gz',payload/'jsonfix.tar.gz')
 for n in ('shell.qml','desktop.py','fleet_guest.py','fleet_client.py','desk_guest.py'):shutil.copy2(SUPPORT/n,payload/n)
 guest=(HERE/'guest.py').read_text() if a.case=='runtime' else (SUPPORT/'guest.py').read_text()
 if a.case=='upgrade':
  original="  print('OMABIT_RELEASE_UPGRADE_RESULT '+json.dumps(report),flush=True)"
  framed="  raw=json.dumps(report,separators=(',',':')).encode();encoded=base64.b64encode(raw).decode();chunks=[encoded[i:i+3000] for i in range(0,len(encoded),3000)]\n  for i,data in enumerate(chunks):print('OMABIT_PROFILE_REPORT_CHUNK '+json.dumps({'i':i,'data':data}),flush=True)\n  print('OMABIT_PROFILE_REPORT_END '+json.dumps({'chunks':len(chunks),'sha256':hashlib.sha256(raw).hexdigest()}),flush=True)"
  if guest.count(original)!=1:raise RuntimeError('upgrade report adapter no longer matches')
  guest=guest.replace(original,framed)
 (payload/'guest.py').write_text(guest)
 harness_hashes={n:digest(payload/n) for n in ('guest.py','desktop.py','shell.qml','fleet_guest.py','fleet_client.py','desk_guest.py')}
 harness_hashes['run.py']=digest(Path(__file__))
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
 (payload/'scenario.json').write_text(json.dumps({'desktop':previous,'collector':previous,'runner':previous,'target':target,'fleet':bool(a.theme_source),'namespace_tests':bool(a.namespace_tests),'app_workflow':a.app_workflow}))
 disk=work/'payload.ext4'
 with disk.open('xb') as f:f.truncate(768*1024**2)
 run(['mkfs.ext4','-q','-F','-d',payload,disk],stdout=subprocess.DEVNULL)
 # Staging files are generated copies/hardlinks; the immutable payload disk is
 # now complete. This is the only automatic removal in the harness.
 shutil.rmtree(payload)
 init=work/'init';shutil.copy2(a.init,init)
 if digest(init)!=boot_provenance['sha256']:raise RuntimeError('boot copy changed')
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
 root_limit=((budget-allocated(work)-64*1024**2)//(64*1024**2))*(64*1024**2)
 if root_limit<2*GIB:raise RuntimeError('fixed inputs leave insufficient root scratch')
 cmd=['systemd-run','--user','--scope','--unit=omabit-'+work.name,'-p','MemoryMax=10G','-p','MemorySwapMax=0','-p','CPUQuota=100%','-p','TasksMax=512','--',
 'bwrap','--unshare-all','--die-with-parent','--clearenv','--ro-bind','/usr','/usr','--symlink','usr/bin','/bin','--symlink','usr/lib','/lib','--symlink','usr/lib','/lib64','--proc','/proc','--dev','/dev','--dev-bind','/dev/kvm','/dev/kvm','--tmpfs','/tmp','--dir','/run','--ro-bind',str(tools),'/tools','--ro-bind',str(a.portable/'base.qcow2'),'/base.qcow2','--ro-bind',str(a.portable/'boot/vmlinuz'),'/vmlinuz','--bind',str(work),'/lab','--chdir','/lab','--setenv','HOME','/lab','--setenv','PATH','/usr/bin:/bin','--setenv','LD_LIBRARY_PATH','/tools/usr/lib','--setenv','QEMU_MODULE_DIR','/tools/usr/lib/qemu','--','/usr/bin/prlimit','--fsize='+str(root_limit)+':'+str(root_limit),'--','/tools/usr/bin/qemu-system-x86_64','-accel','kvm','-cpu','host','-machine','q35','-m','8G','-smp','3','-nodefaults','-no-user-config','-display','none','-monitor','none','-nic','none','-no-reboot','-L','/tools/usr/share/qemu','-bios','/tools/usr/share/qemu/bios-256k.bin','-kernel','/vmlinuz','-initrd','/lab/initramfs.cpio','-append','console=ttyS0 panic=-1 quiet rdinit=/init omabit.disposable-guest=release-upgrade-v1 systemd.unit=multi-user.target','-drive','file=/lab/root.qcow2,format=qcow2,if=none,id=rootdisk','-device','virtio-blk-pci,drive=rootdisk','-drive','file=/lab/payload.ext4,format=raw,if=none,id=payload,readonly=on','-device','virtio-blk-pci,drive=payload','-qmp','unix:/lab/control/qmp.sock,server=on,wait=off','-serial','stdio']
 receipt={'boot_helper':boot_provenance,'root_logical_file_max':root_limit,'storage_admission':admission,'receiver_sources':receiver_provenance,'namespace_tests':namespace_tests,'harness_sha256':harness_hashes,'case':a.case,'artifact_source_commit':source,'scenario':('runtime profiles '+target if a.case=='runtime' else previous+' desktop and services to '+target),'source_commit':subprocess.check_output(['git','rev-parse','HEAD'],cwd=ROOT,text=True).strip(),'used_before':used,'scratch_limit':budget,'base_sha256':meta['sha256']['base.qcow2'],'base_path':str(a.portable/'base.qcow2'),'network':'none','host_mounts':'OS files, QEMU, immutable base and fresh lab only'}
 (work/'host.json').write_text(json.dumps(receipt,indent=2)+'\n')
 with (work/'serial.log').open('wb') as log:
  proc=subprocess.Popen(cmd,stdout=log,stderr=subprocess.STDOUT)
  started=time.monotonic()
  def terminated(signum,frame):raise InterruptedError('qualification driver received signal '+str(signum))
  previous_handler=signal.signal(signal.SIGTERM,terminated)
  try:
   while proc.poll() is None:
    time.sleep(10)
    full_charge(0,storage_limit)
    if (work/'root.qcow2').stat().st_size>=root_limit or allocated(work)>budget or (work/'serial.log').stat().st_size>32*1024**2 or time.monotonic()-started>2700:
     raise RuntimeError('Guest budget/deadline exceeded; retained bounded failure evidence')
  except BaseException as error:
   receipt['interruption']=str(error)
   (work/'host.json').write_text(json.dumps(receipt,indent=2)+'\n')
   if proc.poll() is None:
    subprocess.run(['systemctl','--user','kill','--signal=TERM','omabit-'+work.name+'.scope'],check=True)
    proc.wait(timeout=15)
   raise
  finally:signal.signal(signal.SIGTERM,previous_handler)
  receipt['exit']=proc.returncode
 receipt['used_after']=full_charge(0,storage_limit)['charged_bytes']
 (work/'host.json').write_text(json.dumps(receipt,indent=2)+'\n')
 lines=(work/'serial.log').read_text(errors='replace').splitlines()
 for line in lines:
  if 'OMABIT_GUIDED_IMAGE ' in line:
   data=json.loads(line.split('OMABIT_GUIDED_IMAGE ',1)[1])
   assert data['name'] in ('offer','retry','complete')
   png=base64.b64decode(data['png']);assert len(png)<200000
   (work/(data['name']+'.png')).write_bytes(png)
 if proc.returncode!=0:raise RuntimeError('VM process did not exit successfully')
 result=read_report(lines)
 (work/'result.json').write_text(json.dumps(result,indent=2)+'\n')
 print(json.dumps({'passed':bool(result and result.get('passed')),'output':str(work),'error':result.get('error') if result else 'No completed guest result'}))
 if not result or not result.get('passed'):raise SystemExit(1)
if __name__=='__main__':main()
