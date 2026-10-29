#!/usr/bin/env python3
"""Portable, persistent Omabit onboarding VM. No host runner or Docker operations."""
import argparse,fcntl,hashlib,json,os,platform,shutil,signal,socket,subprocess,sys,time
from pathlib import Path
ROOT=Path(__file__).resolve().parent
STATE=ROOT/'state'
TAG=hashlib.sha256(str(ROOT).encode()).hexdigest()[:12]
UNIT='omabit-review-'+TAG
RUN=Path(os.environ.get('XDG_RUNTIME_DIR','/run/user/'+str(os.getuid())))/('omabit-vm-'+TAG)
PORT=6081
URL=f'http://127.0.0.1:{PORT}/vnc.html?autoconnect=1&resize=scale'
GIB=1024**3

def require(ok,message):
 if not ok:raise RuntimeError(message)
def command(args,**kw):return subprocess.run(args,check=True,**kw)
def capture(args,**kw):return subprocess.check_output(args,text=True,**kw).strip()
def active():
 p=subprocess.run(['systemctl','--user','show',UNIT+'.service','-p','ActiveState','--value'],capture_output=True,text=True)
 return p.stdout.strip() or 'inactive'
def qmp(name,args=None):
 with socket.socket(socket.AF_UNIX,socket.SOCK_STREAM) as s:
  s.settimeout(8);s.connect(str(RUN/'qmp.sock'));f=s.makefile('rwb',buffering=0)
  json.loads(f.readline())
  def call(n,a=None):
   f.write((json.dumps({'execute':n,**({'arguments':a} if a else {})})+'\n').encode())
   while True:
    row=json.loads(f.readline())
    if 'error' in row:raise RuntimeError(str(row['error']))
    if 'return' in row:return row['return']
  call('qmp_capabilities');return call(name,args)
def kvm_available():
 try:
  import fcntl
  fd=os.open('/dev/kvm',os.O_RDWR|os.O_CLOEXEC)
  try:return fcntl.ioctl(fd,0xAE00,0)==12
  finally:os.close(fd)
 except OSError:return False
def tool(name):
 # Prefer distribution-installed QEMU. The verified bundled copy is a fallback.
 native=Path('/usr/bin')/name
 return native if native.exists() else ROOT/'tools/qemu/usr/bin'/name
def toolenv(exe):
 env=dict(os.environ,PYTHONDONTWRITEBYTECODE='1')
 if str(exe).startswith(str(ROOT/'tools')):
  env.update(LD_LIBRARY_PATH=str(ROOT/'tools/qemu/usr/lib'),QEMU_MODULE_DIR=str(ROOT/'tools/qemu/usr/lib/qemu'))
 return env
def allocated(p):return int(capture(['du','-sx','-B1',str(p)]).split()[0])
def storage():
 # Honor sourcebox's existing shared budget if running inside those roots.
 lab=next((p for p in ROOT.parents if p.name=='omabit-private-docker-labs'),None)
 if lab and (lab.parent/'omabit-qualification-evidence').is_dir():
  roots=[lab,lab.parent/'omabit-qualification-evidence'];limit=40*GIB
 else:roots=[ROOT];limit=12*GIB
 used=sum(allocated(p) for p in roots)
 return {'bytes':used,'limit':limit,'free_bytes':shutil.disk_usage(ROOT).free}
def verify():
 data=json.loads((ROOT/'bundle.json').read_text());require(data['format']==1,'Unsupported bundle version')
 for name,expected in data['sha256'].items():
  p=ROOT/name
  require(not p.is_symlink() and p.is_file() and p.resolve().is_relative_to(ROOT),'Missing/unsafe bundle file: '+name)
  with p.open('rb') as f:actual=hashlib.file_digest(f,'sha256').hexdigest()
  require(actual==expected,'Bundle checksum mismatch: '+name+'; preserve state and re-fetch this exact bundle.')
 for name,target in data.get('symlinks',{}).items():
  p=ROOT/name
  require(p.parent.resolve().is_relative_to(ROOT) and p.is_symlink() and os.readlink(p)==target,'Missing/changed bundle symlink: '+name)
 return data
def doctor():
 require(os.geteuid()!=0,'Run as your normal desktop user, not sudo/root.')
 require(platform.system()=='Linux' and platform.machine()=='x86_64','This build supports x86-64 Linux/Omarchy.')
 for name in ['python3','bwrap','systemd-run','systemctl','du']:
  require(shutil.which(name),'Missing '+name+'. Install its native Omarchy package first; the launcher never installs host packages.')
 require(Path('/sys/fs/cgroup/cgroup.controllers').exists(),'cgroup v2 is required.')
 p=subprocess.run(['systemctl','--user','show-environment'],capture_output=True)
 require(p.returncode==0,'A user systemd session is required; log into the laptop normally.')
 for name in ['qemu-system-x86_64','qemu-img']:
  e=tool(name);require(e.exists(),'QEMU missing; install qemu-base on the host.')
  p=subprocess.run([str(e),'--version'],env=toolenv(e),capture_output=True,text=True)
  require(p.returncode==0,'QEMU cannot load on this host. Install qemu-base and qemu-hw-display-virtio-vga from your Omarchy packages. '+p.stderr[:350])
 p=subprocess.run(['bwrap','--unshare-all','--ro-bind','/usr','/usr','--symlink','usr/bin','/bin','--symlink','usr/lib','/lib','--symlink','usr/lib','/lib64','--proc','/proc','--dev','/dev','--','/usr/bin/true'],capture_output=True,text=True)
 require(p.returncode==0,'Unprivileged bubblewrap is unavailable: '+p.stderr[:250])
 st=storage();require(st['bytes']<st['limit'],'Review storage budget reached; preserve the disk and inspect usage.')
 require(st['free_bytes']>2*GIB,'Need at least 2 GiB free disk for safe runtime writes.')
 mem=int(next(x.split()[1] for x in Path('/proc/meminfo').read_text().splitlines() if x.startswith('MemAvailable:')))*1024
 return {'architecture':platform.machine(),'accelerator':'kvm' if kvm_available() else 'tcg','qemu':str(tool('qemu-system-x86_64')),'available_ram_GiB':round(mem/GIB,1),'storage_GiB':round(st['bytes']/GIB,2),'budget_GiB':st['limit']//GIB,'viewer':URL}
def qemu_command(config,disk=None,prepare=False):
 exe=tool('qemu-system-x86_64');bundled=str(exe).startswith(str(ROOT/'tools'))
 cmd=['bwrap','--unshare-all']
 if config['network']=='internet':cmd+=['--share-net','--dir','/etc','--ro-bind','/etc/resolv.conf','/etc/resolv.conf']
 cmd+=['--die-with-parent','--clearenv','--ro-bind','/usr','/usr','--symlink','usr/bin','/bin','--symlink','usr/lib','/lib','--symlink','usr/lib','/lib64','--proc','/proc','--dev','/dev','--tmpfs','/tmp','--dir','/run','--ro-bind',str(ROOT/'boot'),'/boot','--ro-bind',str(ROOT/'tools/qemu'),'/tools','--bind',str(RUN),'/control','--setenv','HOME','/control','--setenv','PATH','/usr/bin:/bin']
 if prepare:cmd+=['--bind',str(disk),'/base.qcow2']
 else:cmd+=['--ro-bind',str(ROOT/'base.qcow2'),'/base.qcow2','--bind',str(STATE),'/state']
 if config['accelerator']=='kvm':cmd+=['--dev-bind','/dev/kvm','/dev/kvm']
 if bundled:cmd+=['--setenv','LD_LIBRARY_PATH','/tools/usr/lib','--setenv','QEMU_MODULE_DIR','/tools/usr/lib/qemu']
 cmd+=['--chdir','/control','--',('/tools/usr/bin/qemu-system-x86_64' if bundled else '/usr/bin/qemu-system-x86_64'),'-name','Omabit Fresh Onboarding REVIEW','-accel',('kvm' if config['accelerator']=='kvm' else 'tcg,thread=multi'),'-cpu',('host' if config['accelerator']=='kvm' else 'max'),'-machine','q35','-m',str(config['memory'])+'G','-smp',str(config['cpus']),'-nodefaults','-no-user-config','-display','none','-monitor','none']
 if bundled:cmd+=['-L','/tools/usr/share/qemu','-bios','/tools/usr/share/qemu/bios-256k.bin']
 if config['network']=='internet':cmd+=['-netdev','user,id=internet,net=192.0.2.0/24,host=192.0.2.2,dns=192.0.2.3,dhcpstart=192.0.2.15,ipv6=off','-device','virtio-net-pci,netdev=internet']
 else:cmd+=['-nic','none']
 append='console=ttyS0 panic=-1 quiet rdinit=/init omabit.review=manual-v1'
 if prepare:append+=' omabit.onboarding=prepare-v1 systemd.unit=omabit-onboarding-prepare.target'
 cmd+=['-kernel','/boot/vmlinuz','-initrd','/boot/initramfs.cpio','-append',append,'-drive','file='+('/base.qcow2' if prepare else '/state/session.qcow2')+',format=qcow2,if=none,id=rootdisk,discard=unmap','-device','virtio-blk-pci,drive=rootdisk','-device','virtio-vga','-device','qemu-xhci','-device','usb-tablet','-device','usb-kbd','-vnc','unix:/control/vnc.sock,password=on','-qmp','unix:/control/qmp.sock,server=on,wait=off','-chardev','socket,id=serial,path=/control/serial.sock,server=on,wait=off,logfile=/control/serial.log','-serial','chardev:serial']
 return cmd
def run_vm():
 os.umask(0o077);RUN.mkdir(mode=0o700,parents=True,exist_ok=True)
 lock=(RUN.parent/'omabit-portable.lock').open('a');fcntl.flock(lock,fcntl.LOCK_EX|fcntl.LOCK_NB)
 for name in ['qmp.sock','vnc.sock','serial.sock','stop-requested']:(RUN/name).unlink(missing_ok=True)
 logs=STATE/'logs';logs.mkdir(exist_ok=True)
 for name in ['qemu.log','viewer.log','serial.log']:
  p=logs/name
  if p.exists():p.replace(logs/(name+'.previous'))
 config=json.loads((STATE/'config.json').read_text());reason=None;deadline=None;forced=False
 def stop(*_):
  nonlocal reason
  reason=reason or 'host requested shutdown'
 signal.signal(signal.SIGTERM,stop);signal.signal(signal.SIGINT,stop)
 qlog=(logs/'qemu.log').open('wb');vlog=(logs/'viewer.log').open('wb');p=subprocess.Popen(qemu_command(config),stdout=qlog,stderr=subprocess.STDOUT);v=None
 try:
  for _ in range(150):
   require(p.poll() is None,'QEMU exited; read state/logs/qemu.log')
   if (RUN/'qmp.sock').exists():break
   time.sleep(.2)
  qmp('set_password',{'protocol':'vnc','password':'review26'})
  env={'PATH':'/usr/bin:/bin','HOME':str(RUN),'PYTHONPATH':str(ROOT/'tools/websockify-0.13.0'),'PYTHONDONTWRITEBYTECODE':'1'}
  v=subprocess.Popen(['/usr/bin/python3','-B','-m','websockify','--web',str(ROOT/'tools/noVNC-1.7.0'),'--unix-target',str(RUN/'vnc.sock'),'127.0.0.1:'+str(PORT)],env=env,stdout=vlog,stderr=subprocess.STDOUT)
  print('Desktop: '+URL,flush=True);last=0
  while p.poll() is None:
   require(v.poll() is None,'Viewer exited; read state/logs/viewer.log')
   if (RUN/'stop-requested').exists():stop()
   if time.monotonic()-last>=30:
    last=time.monotonic();st=storage()
    if st['bytes']>=st['limit'] or st['free_bytes']<GIB or any(x.stat().st_size>32*1024**2 for x in [logs/'qemu.log',logs/'viewer.log',RUN/'serial.log'] if x.exists()):reason='storage or log limit'
   if reason and deadline is None:qmp('system_powerdown');deadline=time.monotonic()+150
   if deadline and time.monotonic()>deadline:forced=True;raise RuntimeError('Guest shutdown deadline; inspect logs before reopening.')
   time.sleep(1)
 finally:
  if v and v.poll() is None:v.terminate();v.wait(timeout=10)
  if p.poll() is None:forced=True;p.terminate();p.wait(timeout=10)
  qlog.close();vlog.close()
  if (RUN/'serial.log').exists():shutil.copyfile(RUN/'serial.log',logs/'serial.log')
  (STATE/'last-exit.json').write_text(json.dumps({'qemu_exit':p.returncode,'forced':forced,'reason':reason,'time':time.time()})+'\n')
def main():
 ap=argparse.ArgumentParser(description=__doc__);ap.add_argument('command',choices=['doctor','verify','start','stop','status','open','_run']);ap.add_argument('--memory',type=int,default=8,choices=range(6,13));ap.add_argument('--cpus',type=int,default=4,choices=range(2,9));ap.add_argument('--accel',choices=['auto','kvm','tcg'],default='auto');ap.add_argument('--network',choices=['internet','off'],default='internet');a=ap.parse_args()
 if a.command=='_run':run_vm();return
 if a.command=='doctor':print(json.dumps(doctor(),indent=2));return
 if a.command=='verify':print('Verified '+verify()['build']);return
 if a.command=='status':
  info={'state':active(),'viewer':URL,'disk':str(STATE/'session.qcow2')}
  if (STATE/'last-exit.json').exists():info['last_exit']=json.loads((STATE/'last-exit.json').read_text())
  print(json.dumps(info,indent=2));return
 if a.command=='stop':
  if active()=='active':(RUN/'stop-requested').write_text('stop\n');print('Guest shutdown requested. Wait for status inactive; data is retained.')
  else:print('Already stopped.')
  return
 if a.command=='open':
  require(active()=='active','Start the VM first.');command(['xdg-open',URL]);return
 if active()=='active':print(URL+' (already running)');return
 report=doctor();verify();accel=report['accelerator'] if a.accel=='auto' else a.accel
 require(accel!='kvm' or kvm_available(),'KVM is not accessible. Use --accel tcg or configure KVM yourself; no host changes were made.')
 require(report['available_ram_GiB']>=a.memory+1,'Insufficient available RAM. Close apps or lower --memory to 6.')
 with socket.socket() as check:
  check.setsockopt(socket.SOL_SOCKET,socket.SO_REUSEADDR,1)
  try:check.bind(('127.0.0.1',PORT))
  except OSError:raise RuntimeError('Port 6081 is occupied. Stop your existing review viewer first.')
 STATE.mkdir(mode=0o700,exist_ok=True);RUN.mkdir(mode=0o700,parents=True,exist_ok=True)
 image=STATE/'session.qcow2';exe=tool('qemu-img')
 if not image.exists():command([str(exe),'create','-q','-f','qcow2','-F','qcow2','-b','../base.qcow2',str(image)],env=toolenv(exe));image.chmod(0o600)
 info=json.loads(capture([str(exe),'info','--output=json',str(image)],env=toolenv(exe)))
 require(info.get('backing-filename')=='../base.qcow2' and info['format']=='qcow2','Unexpected disk backing chain; preserve it and inspect before launch.')
 (STATE/'config.json').write_text(json.dumps({'memory':a.memory,'cpus':a.cpus,'accelerator':accel,'network':a.network})+'\n')
 command(['systemd-run','--user','--unit='+UNIT,'--property=MemoryMax='+str(a.memory+2)+'G','--property=MemorySwapMax=0','--property=CPUQuota='+str(min(a.cpus,4)*100)+'%','--property=TasksMax=512','--property=TimeoutStopSec=180','--property=KillMode=mixed','--collect','--','/usr/bin/python3','-B',str(__file__),'_run'])
 print('Starting with '+accel.upper()+'. Desktop: '+URL+'\nViewer password: review26. Guest password: omabit-review.\nAllow startup to finish. Repeated start resumes this disk; it never resets onboarding.')
if __name__=='__main__':
 try:main()
 except (RuntimeError,OSError,subprocess.CalledProcessError) as e:print('Omabit review: '+str(e),file=sys.stderr);sys.exit(1)
