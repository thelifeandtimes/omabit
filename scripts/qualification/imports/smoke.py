#!/usr/bin/env python3
"""Verify a private QEMU/TCG guest boundary before adding the import runtime."""
import argparse
import re
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import shutil
import signal
import subprocess
import uuid

repo=Path(__file__).resolve().parents[3]
parser=argparse.ArgumentParser(description=__doc__)
parser.add_argument('--tools',type=Path,required=True,help='private signed QEMU package bundle prepared by qualify-powercut.py tools()')
parser.add_argument('--kernel',type=Path,required=True,help='copied/pinned x86-64 kernel to boot')
parser.add_argument('--output',type=Path,required=True,help='fresh /tmp/omabit-import-vm-smoke-NAME')
args=parser.parse_args()
bundle=args.tools.absolute()
work=args.output.absolute()
if work.parent!=Path('/tmp') or not re.fullmatch('omabit-import-vm-smoke-[A-Za-z0-9-]+',work.name) or work.exists():
    parser.error('use a fresh /tmp/omabit-import-vm-smoke-NAME directory')
if os.uname().machine!='x86_64' or not args.kernel.is_file() or not (bundle/'packages.json').is_file():
    parser.error('x86-64 host, explicit kernel and verified QEMU package bundle required')
work.mkdir(mode=0o700)
artifacts=work/'artifacts';artifacts.mkdir()
(work/'lab').mkdir()
spec=importlib.util.spec_from_file_location('powercut_tools',repo/'scripts/qualify-powercut.py')
module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
subprocess.run(['go','build','-trimpath','-o',str(artifacts/'init'),str(repo/'scripts/qualification/imports/smoke-init/main.go')],check=True,env=dict(os.environ,CGO_ENABLED='0',GOCACHE='/tmp/omabit-go-build',GOMODCACHE='/tmp/omabit-go-mod'))
shutil.copyfile(args.kernel,artifacts/'vmlinuz')
shutil.copyfile(repo/'scripts/qualification/imports/smoke-init/main.go',artifacts/'init.go')
verified_tools=module.tool_evidence(bundle)
with (artifacts/'initramfs.cpio').open('wb') as f:
    for n,name in enumerate(('proc','sys','dev'),1):module.cpio_file(f,name,b'',0o40755,n)
    module.cpio_file(f,'init',(artifacts/'init').read_bytes(),0o100755,4)
    module.cpio_file(f,'TRAILER!!!',b'',0,5)
unit='omabit-vm-smoke-'+uuid.uuid4().hex
command=['systemd-run','--user','--scope','--unit='+unit,'--property=MemoryMax=768M','--property=MemorySwapMax=0','--property=CPUQuota=100%','--property=TasksMax=64','--',
         'bwrap','--unshare-all','--die-with-parent','--clearenv','--ro-bind','/usr','/usr','--symlink','usr/bin','/bin','--symlink','usr/lib','/lib','--symlink','usr/lib','/lib64','--proc','/proc','--dev','/dev','--tmpfs','/tmp','--dir','/run',
         '--ro-bind',str(bundle/'tools'),'/tools','--ro-bind',str(artifacts),'/artifacts','--bind',str(work/'lab'),'/lab','--chdir','/lab',
         '--setenv','PATH','/usr/bin:/bin','--setenv','HOME','/lab','--setenv','LD_LIBRARY_PATH','/tools/usr/lib','--setenv','QEMU_MODULE_DIR','/tools/usr/lib/qemu','--',
         '/tools/usr/bin/qemu-system-x86_64','-accel','tcg,thread=single','-cpu','max','-machine','q35','-m','512M','-smp','1','-nodefaults','-no-user-config','-display','none','-monitor','none','-serial','stdio','-nic','none','-no-reboot',
         '-L','/tools/usr/share/qemu','-bios','/tools/usr/share/qemu/bios-256k.bin','-kernel','/artifacts/vmlinuz','-initrd','/artifacts/initramfs.cpio','-append','console=ttyS0 panic=-1 quiet rdinit=/init omabit.disposable-guest=smoke-v1']
(work/'command.json').write_text(json.dumps(command,indent=2)+'\n')
process=subprocess.Popen(command,stdout=subprocess.PIPE,stderr=subprocess.STDOUT,start_new_session=True)
timed_out=False
try:
    output,_=process.communicate(timeout=90)
except subprocess.TimeoutExpired:
    timed_out=True
    os.killpg(process.pid,signal.SIGTERM)
    try:output,_=process.communicate(timeout=3)
    except subprocess.TimeoutExpired:
        os.killpg(process.pid,signal.SIGKILL);output,_=process.communicate()
truncated=len(output)>1048576
(work/'console.log').write_bytes(output[:1048576])
report={'exit':process.returncode,'ready':b'OMABIT_VM_READY' in output,'timed_out':timed_out,'log_truncated':truncated,'unit':unit,'acceleration':'TCG single thread','host_filesystems_mounted_in_guest':False,'network_device':False,'qemu':verified_tools,'artifact_sha256':{p.name:hashlib.sha256(p.read_bytes()).hexdigest() for p in artifacts.iterdir() if p.is_file()}}
report['source_sha256']={str(path.relative_to(repo)):hashlib.sha256(path.read_bytes()).hexdigest() for path in (Path(__file__).resolve(),repo/'scripts/qualification/imports/smoke-init/main.go',repo/'scripts/qualify-powercut.py')}
(work/'result.json').write_text(json.dumps(report,indent=2)+'\n')
print(json.dumps({key:value for key,value in report.items() if key not in ('qemu','artifact_sha256','source_sha256')}))
if process.returncode or not report['ready'] or timed_out or truncated:
    print(output.decode(errors='replace')[-5000:]);raise SystemExit(1)
