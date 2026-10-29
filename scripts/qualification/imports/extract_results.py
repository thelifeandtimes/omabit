#!/usr/bin/env python3
"""Extract private result files from a completed lab disk, without mounting it."""
import storage
import argparse
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import re
import subprocess

HERE=Path(__file__).resolve().parent

def sha(path):
    with path.open('rb') as f:return hashlib.file_digest(f,'sha256').hexdigest()

def admit_conversion(directory, info):
    # A raw conversion can allocate every virtual byte. Sparse source usage and
    # a fixed 4GiB estimate undercount grown lifecycle disks. Leave 2GiB for
    # extracted files and independent checkpoint hashing; serialize large tests
    # with conversion/verification because quota admission is a point-in-time read.
    size=info.get('virtual-size')
    if info.get('format')!='qcow2' or type(size) is not int or size<=0 or size%512 or size>64*1024**3:
        raise ValueError('invalid private overlay geometry')
    result=storage.admit(directory,size+2*1024**3)
    result['conversion_virtual_bytes']=size
    return result

def main():
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--run',type=Path,required=True)
    parser.add_argument('--tools',type=Path,required=True)
    parser.add_argument('--output',type=Path,required=True)
    args=parser.parse_args();run=args.run.absolute();work=args.output.absolute()
    if work.parent!=Path('/tmp') or not re.fullmatch('omabit-import-inspection-[A-Za-z0-9-]+',work.name) or work.exists():parser.error('use a fresh /tmp/omabit-import-inspection-NAME')
    inputs=json.loads((run/'inputs.json').read_text());result=json.loads((run/'result.json').read_text())
    # The boot driver publishes result.json only after waiting for its child.
    # Also refuse any still-loaded active scope; never inspect a live disk.
    unit=inputs['unit']
    if not re.fullmatch('omabit-import-guest-[0-9a-f]{32}',unit):raise ValueError('unexpected scope')
    observed=subprocess.run(['systemctl','--user','is-active',unit+'.scope'],capture_output=True,text=True,timeout=10)
    if observed.stdout.strip() not in ('inactive','failed','unknown'):raise ValueError('guest scope may still be running')
    artifacts=Path(inputs['appliance'])/'artifacts'
    if any(sha(artifacts/name)!=digest for name,digest in inputs['build']['artifacts'].items()):raise ValueError('guest backing artifacts changed')
    spec=importlib.util.spec_from_file_location('appliance',HERE/'appliance.py')
    module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
    bundle=args.tools.absolute();tools=module.helper().tool_evidence(bundle)
    readonly=['bwrap','--unshare-all','--die-with-parent','--clearenv','--ro-bind','/usr','/usr','--symlink','usr/bin','/bin','--symlink','usr/lib','/lib','--symlink','usr/lib','/lib64','--proc','/proc','--dev','/dev','--tmpfs','/tmp','--ro-bind',str(bundle/'tools'),'/tools','--ro-bind',str(artifacts),'/artifacts','--ro-bind',str(run/'lab'),'/guest','--setenv','PATH','/usr/bin:/bin','--setenv','LD_LIBRARY_PATH','/tools/usr/lib']
    geometry=subprocess.run(readonly+['--chdir','/','--','/tools/usr/bin/qemu-img','info','--output=json','-f','qcow2','/guest/root.qcow2'],capture_output=True,check=True,timeout=15)
    if len(geometry.stdout)>65536 or len(geometry.stderr)>8192:raise ValueError('excessive overlay geometry output')
    storage_admission=admit_conversion(work.parent,json.loads(geometry.stdout))
    os.umask(0o077);work.mkdir(mode=0o700)
    namespace=readonly+['--bind',str(work),'/output','--chdir','/output','--']
    with (work/'conversion.log').open('wb') as log:
        subprocess.run(namespace+['/tools/usr/bin/qemu-img','convert','-f','qcow2','-O','raw','-S','4096','/guest/root.qcow2','/output/guest.raw'],stdout=log,stderr=subprocess.STDOUT,check=True,timeout=120)
    with (work/'extraction.log').open('wb') as log:
        subprocess.run(namespace+['/usr/bin/debugfs','-R','rdump /opt/omabit-lab/results /output','/output/guest.raw'],stdout=log,stderr=subprocess.STDOUT,check=True,timeout=60)
    with (work/'extra-extraction.log').open('wb') as log:
        requests=['rdump /var/log/journal /output','dump /home/lab/.local/share/urbit-runner/managed/runner.sqlite /output/runner.sqlite']
        if inputs['build'].get('exercise_project_preparation') or inputs['build'].get('exercise_project_anchor_start') or inputs['build'].get('exercise_project_member_storage') or inputs['build'].get('exercise_project_member_launch'):
            project_store='/home/lab/.local/share/urbit-runner/managed' if inputs['build'].get('exercise_project_service_restart') else '/home/lab/project-qualification-store'
            requests.append('dump '+project_store+'/runner.sqlite /output/project-runner.sqlite')
        for request in requests:
            subprocess.run(namespace+['/usr/bin/debugfs','-R',request,'/output/guest.raw'],stdout=log,stderr=subprocess.STDOUT,check=True,timeout=60)
    files={}
    for path in (work/'results').rglob('*'):
        if path.is_symlink():raise ValueError('unexpected result symlink')
        if path.is_file():files[str(path.relative_to(work))]={'bytes':path.stat().st_size,'sha256':sha(path)}
    if not files or len(files)>256 or sum(item['bytes'] for item in files.values())>256*1024**2:raise ValueError('missing or excessive extracted result files')
    report={'run':str(run),'storage_admission':storage_admission,'converted_disk_sha256':sha(work/'guest.raw'),'guest_result':result.get('guest_result'),'source_overlay_sha256':sha(run/'lab/root.qcow2'),'all_disk_sha256':{p.name:sha(p) for p in sorted((run/'lab').glob('*.qcow2'))},'tools':tools,'files':files,'driver_sha256':sha(Path(__file__))}
    (work/'extraction.json').write_text(json.dumps(report,indent=2)+'\n')
    print(json.dumps({'output':str(work),'files':len(files),'bytes':sum(item['bytes'] for item in files.values())}),flush=True)

if __name__=='__main__':main()
