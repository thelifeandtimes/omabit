#!/usr/bin/env python3
"""Audit one stopped private Docker qualification disk in a read-only no-NIC VM.

The input must be a completed private-chain qualification run. Only bounded
public metadata and hashes leave the guest. No ship or Docker daemon is started.
"""
import argparse
import importlib.util
import json
import os
from pathlib import Path
import re
import signal
import subprocess
import time
import uuid
from docker_appliance import HERE,REPO,IMPORTS,sha,read_json,write_json,cpio_entries,write_cpio,relative
import storage


def inactive(unit):
    if not re.fullmatch(r'omabit-(?:import-guest|private-inspect)-[0-9a-f]{32}',unit):raise ValueError('qualification scope identity required')
    result=subprocess.run(['systemctl','--user','show',unit+'.scope','-p','ActiveState','--value'],capture_output=True,text=True,timeout=10)
    return result.returncode==0 and result.stdout.strip()=='inactive'


def admit_run(run):
    run=run.absolute()
    if run.parent!=Path('/tmp') or not re.fullmatch('omabit-import-guest-run-private-chain-[A-Za-z0-9-]+',run.name) or run.is_symlink():
        raise ValueError('explicit private-chain qualification run required')
    info=run.stat()
    if not run.is_dir() or info.st_uid!=os.getuid() or info.st_mode&0o077:raise ValueError('owned private run directory required')
    inputs=read_json(run/'inputs.json');completed=read_json(run/'result.json')
    if not inactive(inputs['unit']):raise ValueError('original qualification scope must be inactive')
    if completed.get('scope_admitted') is not True or completed.get('exit')!=0 or completed.get('timed_out') is not False or completed.get('clean_shutdown') is not True:
        raise ValueError('completed cleanly stopped qualification required')
    build=inputs['build']
    if build.get('exercise_private_chain') is not True:raise ValueError('private-chain appliance required')
    if type(build.get('gateway_resume_qualification',False)) is not bool:raise ValueError('invalid gateway resume audit selection')
    if type(build.get('gateway_peer_recovery_qualification',False)) is not bool or build.get('gateway_peer_recovery_qualification',False) and not build.get('gateway_resume_qualification',False):raise ValueError('invalid gateway peer audit selection')
    if type(build.get('http_publication_qualification',False)) is not bool or build.get('http_publication_qualification',False) and not build.get('gateway_peer_recovery_qualification',False):raise ValueError('invalid private publication audit selection')
    isolation=build.get('private_project_isolation_qualification',False)
    if type(isolation) is not bool or isolation and any(build.get(k,False) for k in ('gateway_resume_qualification','gateway_peer_recovery_qualification','http_publication_qualification')):raise ValueError('invalid private isolation audit selection')
    initial=build.get('initial_http_publication_qualification',False)
    if type(initial) is not bool or initial and any(build.get(k,False) for k in ('gateway_resume_qualification','gateway_peer_recovery_qualification','http_publication_qualification','private_project_isolation_qualification')):raise ValueError('invalid initial publication audit selection')
    appliance=Path(inputs['appliance'])
    for name,digest in build['artifacts'].items():
        relative(name)
        if sha(appliance/'artifacts'/name)!=digest:raise ValueError('appliance artifact differs')
    if sha(Path(build['base_source']))!=build['base_source_sha256']:raise ValueError('pristine backing differs')
    disk=run/'lab/root.qcow2'
    if disk.is_symlink() or not disk.is_file():raise ValueError('regular private overlay required')
    return inputs,build,appliance


def command(tools,artifacts,appliance,build,run,scope):
    return ['systemd-run','--user','--scope','--unit='+scope,'--property=MemoryMax=2G','--property=MemorySwapMax=0','--property=CPUQuota=100%','--property=TasksMax=64','--',
        'bwrap','--unshare-all','--die-with-parent','--clearenv','--ro-bind','/usr','/usr','--symlink','usr/bin','/bin','--symlink','usr/lib','/lib','--symlink','usr/lib','/lib64','--proc','/proc','--dev','/dev','--tmpfs','/tmp','--dir','/run',
        '--ro-bind',str(tools/'tools'),'/tools','--ro-bind',str(artifacts),'/artifacts','--ro-bind',build['base_source'],'/artifacts/base.raw','--ro-bind',str(appliance/'artifacts/vmlinuz'),'/artifacts/vmlinuz','--ro-bind',str(run/'lab'),'/guest',
        '--setenv','HOME','/tmp','--setenv','PATH','/usr/bin:/bin','--setenv','LD_LIBRARY_PATH','/tools/usr/lib','--setenv','QEMU_MODULE_DIR','/tools/usr/lib/qemu','--',
        '/tools/usr/bin/qemu-system-x86_64','-accel','tcg,thread=single','-cpu','max','-machine','q35','-m','1024M','-smp','1','-nodefaults','-no-user-config','-display','none','-monitor','none','-serial','stdio','-nic','none','-no-reboot','-L','/tools/usr/share/qemu','-bios','/tools/usr/share/qemu/bios-256k.bin',
        '-kernel','/artifacts/vmlinuz','-initrd','/artifacts/initramfs.cpio','-append','console=ttyS0 panic=-1 quiet rdinit=/init omabit.disposable-guest=offline-private-inspect-v1'+(' omabit.gateway-resume-audit=required-v1' if build.get('gateway_resume_qualification') is True else '')+(' omabit.gateway-peer-audit=required-v1' if build.get('gateway_peer_recovery_qualification') is True else '')+(' omabit.private-publication-audit=required-v1' if build.get('http_publication_qualification') is True else '')+(' omabit.private-isolation-audit=required-v1' if build.get('private_project_isolation_qualification') is True else '')+(' omabit.initial-publication-audit=required-v1' if build.get('initial_http_publication_qualification') is True else ''),
        '-drive','file=/guest/root.qcow2,format=qcow2,if=none,id=rootdisk,readonly=on','-device','virtio-blk-pci,drive=rootdisk']


def inspect(args):
    run=args.run.absolute();work=args.output.absolute();tools=args.tools.absolute()
    if work.parent!=Path('/tmp') or not re.fullmatch('omabit-private-chain-inspection-[A-Za-z0-9-]+',work.name) or work.exists() or work.is_symlink():
        raise ValueError('fresh /tmp/omabit-private-chain-inspection-NAME output required')
    inputs,build,appliance=admit_run(run)
    admission=storage.admit('/tmp',256*1024**2)
    pristine=Path(build['pristine_parent']);parent=read_json(pristine/'build.json')
    archive=pristine/'artifacts/initramfs.cpio'
    if sha(archive)!=parent['artifacts']['initramfs.cpio'] or sha(pristine/'build.json')!=build['pristine_parent_build_sha256']:
        raise ValueError('pristine module archive differs')
    entries=cpio_entries(archive)
    spec=importlib.util.spec_from_file_location('private_inspection_tools',REPO/'scripts/qualify-powercut.py');helper=importlib.util.module_from_spec(spec);spec.loader.exec_module(helper)
    tool_evidence=helper.tool_evidence(tools)
    os.umask(0o077);work.mkdir(mode=0o700);artifacts=work/'artifacts';artifacts.mkdir()
    names=('main.go','inspect.go','project.go','resume_audit.go','gateway_peers_audit.go','publication_audit.go','isolation_audit.go','initial_publication_audit.go')
    for name in names:(work/name).write_bytes((HERE/'inspect-init'/name).read_bytes())
    env=dict(os.environ,CGO_ENABLED='0',GOOS='linux',GOARCH='amd64',GOTOOLCHAIN='local',GOPROXY='off',GOSUMDB='off',GOFLAGS='',GOENV='off')
    subprocess.run(['go','build','-trimpath','-o',str(artifacts/'init'),*[str(work/n) for n in names]],cwd=REPO,env=env,check=True,timeout=180)
    with (artifacts/'initramfs.cpio').open('xb') as out:
        for i,(name,data,mode) in enumerate(entries,1):write_cpio(out,name,(artifacts/'init').read_bytes() if name=='init' else data,mode,i)
        write_cpio(out,'TRAILER!!!',b'',0,len(entries)+1)
    for name in ('vmlinuz','base.raw'):(artifacts/name).touch(exist_ok=False)
    scope='omabit-private-inspect-'+uuid.uuid4().hex
    before=sha(run/'lab/root.qcow2');cmd=command(tools,artifacts,appliance,build,run,scope)
    write_json(work/'inputs.json',{'run':str(run),'original_run_scope_inactive':inputs['unit'],'scope':scope,'command':cmd,
        'source_sha256':{n:sha(work/n) for n in names},'driver_sha256':sha(Path(__file__)),'archive_driver_sha256':sha(HERE/'docker_appliance.py'),
        'init_sha256':sha(artifacts/'init'),'initramfs_sha256':sha(artifacts/'initramfs.cpio'),'overlay_before_sha256':before,
        'base_sha256':build['base_source_sha256'],'storage_admission':admission,'tools':tool_evidence})
    started=time.monotonic();admitted=False;timed_out=False;observed={}
    with (work/'console.log').open('xb') as log:
        process=subprocess.Popen(cmd,stdout=log,stderr=subprocess.STDOUT,start_new_session=True)
        try:
            while process.poll() is None and time.monotonic()<started+8:
                result=subprocess.run(['systemctl','--user','show',scope+'.scope','-p','ActiveState,MemoryMax,MemorySwapMax,TasksMax,CPUQuotaPerSecUSec'],capture_output=True,text=True,timeout=5)
                observed=dict(line.split('=',1) for line in result.stdout.splitlines() if '=' in line)
                if observed.get('ActiveState')=='active':
                    admitted=observed=={'ActiveState':'active','MemoryMax':str(2*1024**3),'MemorySwapMax':'0','TasksMax':'64','CPUQuotaPerSecUSec':'1s'};break
                time.sleep(.1)
            write_json(work/'scope-admission.json',observed)
            while admitted and process.poll() is None:
                if time.monotonic()>started+240 or log.tell()>2*1024**2:timed_out=True;break
                time.sleep(.25)
        finally:
            if process.poll() is None:
                os.killpg(process.pid,signal.SIGTERM)
                try:process.wait(timeout=5)
                except subprocess.TimeoutExpired:os.killpg(process.pid,signal.SIGKILL);process.wait()
    raw=(work/'console.log').read_text(errors='replace')
    rows=[json.loads(line.split('OMABIT_PRIVATE_INSPECTION ',1)[1]) for line in raw.splitlines() if 'OMABIT_PRIVATE_INSPECTION ' in line]
    overlay_after=sha(run/'lab/root.qcow2');base_unchanged=sha(Path(build['base_source']))==build['base_source_sha256'];scope_inactive=inactive(scope)
    report={'ok':False,'scope_admitted':admitted,'scope_inactive':scope_inactive,'exit':process.returncode,'timed_out':timed_out,'seconds':time.monotonic()-started,
        'overlay_unchanged':overlay_after==before,'overlay_after_sha256':overlay_after,'base_unchanged':base_unchanged,
        'audit':rows[0] if len(rows)==1 else None,'clean_shutdown':'Power down' in raw,'console_sha256':sha(work/'console.log')}
    report['resume_required']=build.get('gateway_resume_qualification') is True
    report['resume_passed']=bool(len(rows)==1 and rows[0].get('gateway_resume',{}).get('ok') is True)
    report['peer_resume_required']=build.get('gateway_peer_recovery_qualification') is True
    report['peer_resume_passed']=bool(len(rows)==1 and rows[0].get('gateway_resume',{}).get('peer_recovery',{}).get('ok') is True)
    report['publication_required']=build.get('http_publication_qualification') is True
    report['publication_passed']=bool(len(rows)==1 and rows[0].get('gateway_publication',{}).get('ok') is True)
    report['isolation_required']=build.get('private_project_isolation_qualification') is True
    report['isolation_passed']=bool(len(rows)==1 and rows[0].get('private_project_isolation') is True and rows[0].get('ok') is True)
    report['initial_publication_required']=build.get('initial_http_publication_qualification') is True
    report['initial_publication_passed']=bool(len(rows)==1 and rows[0].get('initial_publication',{}).get('ok') is True)
    report['ok']=(not report['initial_publication_required'] or report['initial_publication_passed']) and (not report['isolation_required'] or report['isolation_passed']) and (not report['publication_required'] or report['publication_passed']) and (not report['peer_resume_required'] or report['peer_resume_passed']) and (not report['resume_required'] or report['resume_passed']) and admitted and scope_inactive and process.returncode==0 and not timed_out and report['clean_shutdown'] and report['overlay_unchanged'] and base_unchanged and len(rows)==1 and rows[0].get('ok') is True
    write_json(work/'result.json',report)
    print(json.dumps({'inspection':str(work),'ok':report['ok'],'scope':scope,'scope_inactive':scope_inactive}),flush=True)
    if not report['ok']:raise SystemExit(1)


def main():
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--run',type=Path,required=True,help='completed private-chain qualification run')
    parser.add_argument('--tools',type=Path,required=True,help='verified QEMU tools')
    parser.add_argument('--output',type=Path,required=True,help='fresh /tmp/omabit-private-chain-inspection-NAME')
    inspect(parser.parse_args())

if __name__=='__main__':main()
