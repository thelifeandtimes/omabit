#!/usr/bin/env python3
"""Read fixed saved result files without copying or booting a failed VM's root."""
import argparse
import base64
import importlib.util
import json
import os
from pathlib import Path
import re
import subprocess
import time

spec=importlib.util.spec_from_file_location('issues_run',Path(__file__).with_name('run.py'))
host=importlib.util.module_from_spec(spec)
spec.loader.exec_module(host)


def main():
    p=argparse.ArgumentParser(description=__doc__)
    for n in ('capsule','portable','output'):
        p.add_argument('--'+n,type=Path)
    p.add_argument('--storage-budget-gib',type=int,choices=(40,48),default=40)
    p.add_argument('--run',action='store_true')
    a=p.parse_args()
    if not a.run:
        p.print_help();return
    if os.geteuid()==0 or any(x is None for x in (a.capsule,a.portable,a.output)):
        p.error('ordinary owner and all explicit inputs required')
    if not re.fullmatch('omabit-issues-vm-[0-9]{2}',a.capsule.name) or a.capsule.parent!=Path('/tmp') or a.capsule.is_symlink():
        p.error('exact physical task capsule required')
    if a.output.parent!=Path('/tmp') or not re.fullmatch('omabit-issues-vm-inspect-[0-9]{2}',a.output.name) or a.output.exists():
        p.error('fresh task inspection output required')
    unit='omabit-'+a.capsule.name
    state=subprocess.run(['systemctl','--user','is-active',unit+'.scope'],capture_output=True,text=True,timeout=10)
    if state.stdout.strip() not in ('inactive','failed','unknown') or not (a.capsule/'result.json').is_file():
        raise RuntimeError('original task guest is not proven stopped')
    receipt=json.loads((a.capsule/'host.json').read_text())
    if receipt['exit']!=0:
        raise RuntimeError('original host driver did not finish')
    cap=a.storage_budget_gib*host.GIB
    admission=host.charge(64*1024**2,cap)
    portable=a.portable.resolve(strict=True)
    meta=json.loads((portable/'bundle.json').read_text())
    for n,expected in meta['sha256'].items():
        if host.util.digest(host.ordinary(portable/n))!=expected:
            raise RuntimeError('immutable inspection input changed')
    root=host.ordinary(a.capsule/'root.qcow2')
    root_hash=host.util.digest(root)
    out=a.output;out.mkdir(mode=0o700)
    host.util.run(['go','build','-trimpath','-o',out/'init','./scripts/qualification/issues-recovery/inspect-init'],cwd=host.ROOT,env={**os.environ,'CGO_ENABLED':'0'})
    boot=out/'boot';boot.mkdir()
    host.util.run(['bsdtar','-xf',portable/'boot/initramfs.cpio','-C',boot,'guest-modules','modules'])
    with (out/'initramfs.cpio').open('xb') as f:
        ino=0
        for n in ('proc','sys','dev','newroot','modules'):
            ino+=1;host.util.cpio(f,n,b'',0o40755,ino)
        for path in [out/'init',boot/'guest-modules',*sorted((boot/'modules').glob('*.ko'))]:
            ino+=1;n='init' if path==out/'init' else str(path.relative_to(boot));host.util.cpio(f,n,path.read_bytes(),0o100755 if n=='init' else 0o100644,ino)
        host.util.cpio(f,'TRAILER!!!',b'',0,ino+1)
    tools=portable/'tools/qemu';scope='omabit-'+out.name
    command=['systemd-run','--user','--scope','--unit='+scope,'-p','MemoryMax=1G','-p','MemorySwapMax=0','-p','CPUQuota=100%','-p','TasksMax=64','--',
        'bwrap','--unshare-all','--die-with-parent','--clearenv','--ro-bind','/usr','/usr','--symlink','usr/bin','/bin','--symlink','usr/lib','/lib','--symlink','usr/lib','/lib64',
        '--proc','/proc','--dev','/dev','--dev-bind','/dev/kvm','/dev/kvm','--tmpfs','/tmp','--dir','/run','--ro-bind',str(tools),'/tools','--ro-bind',str(portable/'base.qcow2'),'/base.qcow2',
        '--ro-bind',str(root),'/inspected.qcow2','--ro-bind',str(portable/'boot/vmlinuz'),'/vmlinuz','--bind',str(out),'/lab','--chdir','/lab','--setenv','HOME','/lab','--setenv','PATH','/usr/bin:/bin',
        '--setenv','LD_LIBRARY_PATH','/tools/usr/lib','--setenv','QEMU_MODULE_DIR','/tools/usr/lib/qemu','--','/tools/usr/bin/qemu-system-x86_64','-accel','kvm','-cpu','host','-machine','q35','-m','512M','-smp','1',
        '-nodefaults','-no-user-config','-display','none','-monitor','none','-nic','none','-no-reboot','-L','/tools/usr/share/qemu','-bios','/tools/usr/share/qemu/bios-256k.bin',
        '-kernel','/vmlinuz','-initrd','/lab/initramfs.cpio','-append','console=ttyS0 panic=-1 quiet rdinit=/init omabit.disposable-guest=issues-readonly-inspection-v1',
        '-drive','file=/inspected.qcow2,format=qcow2,if=none,id=inspected,readonly=on','-device','virtio-blk-pci,drive=inspected','-serial','stdio']
    with (out/'serial.log').open('wb') as log:
        child=subprocess.Popen(command,stdout=log,stderr=subprocess.STDOUT)
        started=time.monotonic()
        try:
            while child.poll() is None:
                time.sleep(1)
                host.charge(0,cap)
                if host.storage.measure(out)['allocated_bytes']>64*1024**2 or (out/'serial.log').stat().st_size>16*1024**2 or time.monotonic()-started>90:
                    raise RuntimeError('bounded read-only inspection expired')
        finally:
            if child.poll() is None:
                subprocess.run(['systemctl','--user','kill','--signal=TERM',scope+'.scope'],check=True)
                child.wait(timeout=10)
    if child.returncode or host.util.digest(root)!=root_hash:
        raise RuntimeError('inspection process or original disk changed')
    lines=(out/'serial.log').read_text(errors='replace').splitlines()
    result=next(json.loads(line.split('OMABIT_ISSUES_INSPECTION ',1)[1]) for line in reversed(lines) if 'OMABIT_ISSUES_INSPECTION ' in line)
    if not result.get('passed') or set(result['files'])!={'commands.jsonl','result.json'}:
        raise RuntimeError('read-only inspection did not complete')
    files={}
    for n,b in result.pop('files').items():
        raw=base64.b64decode(b,validate=True)
        if len(raw)>4*1024**2:
            raise RuntimeError('extracted file exceeded bound')
        (out/n).write_bytes(raw)
        files[n]={'bytes':len(raw),'sha256':host.util.digest(out/n)}
    (out/'inspection.json').write_text(json.dumps({'admission':admission,'original_overlay_sha256':root_hash,'files':files,'result':result,'command':command},indent=2)+'\n')
    print(json.dumps({'passed':True,'read_only':True,'original_overlay_unchanged':True,'output':str(out)}))


if __name__=='__main__':
    main()
