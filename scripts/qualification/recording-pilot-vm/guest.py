#!/usr/bin/python3
"""Installed ordinary capture/restart/drain; fresh offline guest only."""
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import subprocess
import sys
import threading

I=Path('/opt/omabit-release-input')
O=Path('/opt/omabit-issues-results')
HOME=Path('/home/review')
report={'format':1,'passed':False,'scope':'one fresh offline fake; installed ordinary recording pilot only; not comprehensive recording or upgrade qualification'}
common=None
trim_stop=threading.Event()
trim_thread=None

def start_guest_trim():
    # The caller has verified QEMU, the exact fresh scenario and pristine root.
    # Reclaim only blocks the guest filesystem has already freed. No host path
    # or device is visible, and no journal/pier bytes are edited or repaired.
    global trim_thread
    report['guest_trim']={'runs':0,'failures':0}
    def trim():
        while not trim_stop.wait(5):
            try:
                result=subprocess.run(['/usr/bin/fstrim','/'],capture_output=True,timeout=10)
                report['guest_trim']['runs']+=1
                report['guest_trim']['failures']+=int(result.returncode!=0)
            except subprocess.TimeoutExpired:
                report['guest_trim']['failures']+=1
    trim_thread=threading.Thread(target=trim,daemon=True)
    trim_thread.start()

def sha(path):
    with Path(path).open('rb') as f:return hashlib.file_digest(f,'sha256').hexdigest()

def guard():
    if os.geteuid()!=0 or Path('/sys/class/dmi/id/sys_vendor').read_text().strip()!='QEMU' or 'omabit.disposable-guest=release-upgrade-v1' not in Path('/proc/cmdline').read_text().split() or sys.argv[1:]:
        raise RuntimeError('refused outside fresh offline recording guest')
    value=json.loads((I/'scenario.json').read_text())
    if value.get('kind')!='issues20-recording-pilot-v1' or value.get('target_schema')!=64 or not value.get('development_snapshot'):
        raise RuntimeError('wrong explicit frozen scenario')
    return value

def snapshot(label):
    result=common.run(['env','OMABIT_RECORDING_PILOT_VM=snapshot',I/'managed.test','-test.run=^TestIssuesRecordingSnapshotDisposableVM$','-test.timeout=60s','-test.v'],True,timeout=70)
    values=[json.loads(line.split('OMABIT_RECORDING_VM_SNAPSHOT ',1)[1]) for line in result.stdout.splitlines() if 'OMABIT_RECORDING_VM_SNAPSHOT ' in line]
    if len(values)!=1:raise RuntimeError('no exact helper snapshot')
    (O/(label+'.json')).write_text(json.dumps(values[0],indent=2)+'\n')
    return values[0]

def worker(row):
    status=row['status'];claim=status['worker'];unit=row['unit']
    text=common.run(['systemctl','--user','show',unit,'-p','Id,Type,NotifyAccess,ActiveState,SubState,MainPID,InvocationID,ControlGroup'],True).stdout
    value=dict(line.split('=',1) for line in text.splitlines())
    if value['Id']!=unit or value['Type']!='notify' or value['NotifyAccess']!='main' or value['ActiveState']!='active' or value['SubState']!='running' or int(value['MainPID'])!=claim['pid'] or value['InvocationID']!=claim['invocation_id'] or not value['ControlGroup'].endswith('/'+unit):
        raise RuntimeError('exact notify worker changed')
    proc=Path('/proc')/str(claim['pid'])
    ticks=int((proc/'stat').read_text().rsplit(')',1)[1].split()[19])
    if ticks!=claim['start_ticks'] or sha(proc/'exe')!=claim['executable_sha256']:
        raise RuntimeError('exact worker kernel/image binding changed')
    return value

def retained_session(row):
    root=HOME/'.local/share/urbit-runner/recordings'/row['status']['session_id']
    return {str(p.relative_to(root)):sha(p) for p in sorted(root.rglob('*')) if p.is_file()}

def closed(row):
    s=row['status'];r=s.get('result') or {};c=r.get('capture') or {}
    if s['state']!='capture-finished' or s['writer_present'] or r.get('code')!='capture-finished' or not all(c.get(k) is True for k in ('attachment_acknowledged','ready_acknowledged','closed_durably','exit_saved')) or row['exit_frames']!=1 or row['gap_frames'] or row['tail']!='snapshot-eof' or row['frames']<2 or row['stdout_bytes']+row['stderr_bytes']<100:
        raise RuntimeError('recording did not retain a complete native capture/exit')

def main():
    global common
    scenario=guard()
    if O.exists():raise RuntimeError('never resume an existing guest')
    O.mkdir(mode=0o700)
    for name in ('/etc/urbit-runner','/srv/urbit/piers','/home/review/.local/share/urbit-runner'):
        if Path(name).exists():raise RuntimeError('guest not pristine')
    for name in ('omabit','urbit-collector','urbit-runner'):
        if sha(I/'candidate'/name)!=scenario['candidate'][name]['sha256']:raise RuntimeError('candidate changed')
    for name,pin in [('managed.test',scenario['candidate']['managed.test']['sha256']),('runtime.oci.tar',scenario['oci_sha256']),('boot.pill',scenario['pill_sha256'])]:
        if sha(I/name)!=pin:raise RuntimeError('frozen input changed')
    spec=importlib.util.spec_from_file_location('common',I/'issues-common.py');common=importlib.util.module_from_spec(spec);spec.loader.exec_module(common)
    common.O,common.report,common.scenario=O,report,scenario
    report.update(inputs=scenario,source_commit=scenario['source_commit'])
    common.run(['loginctl','enable-linger','review'])
    common.run(['systemctl','start','user@1000.service'])
    common.run(['ssh-keygen','-q','-t','ed25519','-N','','-f',HOME/'operator'],True)
    host=common.install('candidate');report['runner_id']=host['runner_id']
    common.run([I/'candidate/urbit-collector','enroll','local','--socket',common.SOCK],True)
    common.docker('load','--input',I/'runtime.oci.tar')
    if common.db_projection()['schema']!=64:raise RuntimeError('wrong installed schema')
    start_guest_trim()
    common.unit('stop')
    override=HOME/'.config/systemd/user/urbit-runner.service.d'
    override.mkdir(mode=0o700,parents=True)
    f=override/'recording-pilot.conf'
    f.write_text('[Service]\nExecStart=\nExecStart=/usr/bin/env -i HOME=/home/review PATH=/usr/bin:/bin LANG=C XDG_RUNTIME_DIR=/run/user/1000 /usr/local/lib/urbit-runner/current/urbit-runner serve-development --acknowledge-development --record-runtime\n')
    for p in (override,override.parent,override.parent.parent,f):os.chown(p,1000,1000)
    common.run(['systemctl','--user','daemon-reload'],True)
    common.unit('start')
    hello=common.hello()
    if hello['build']['sha256']!=scenario['candidate']['urbit-runner']['sha256'] or 'runtime-recording-pilot-v1' not in hello['features']:raise RuntimeError('pilot build/selection mismatch')
    report['hello']=hello
    common.collector('candidate','create-fake','recording-zod','--ship','~zod','--http-port','19480','--request-id','recording-create','--bootstrap','recommended','--pill',str(I/'boot.pill'))
    common.operation('candidate','recording-zod','start','recording-first-start')
    first=common.wait(lambda:common.instance('candidate','recording-zod','running'),600)
    original_container=common.cid(first);original_native=common.inspect(original_container)['State']
    rows=snapshot('first-running')
    if len(rows)!=1 or rows[0]['status']['metadata']['container_id']!=original_container or not rows[0]['status']['writer_present']:raise RuntimeError('missing exact initial capture')
    original_worker=worker(rows[0]);original_claim=rows[0]['status']['worker']
    common.unit('stop')
    detached=snapshot('manager-stopped')
    if worker(detached[0])!=original_worker or detached[0]['status']['worker']!=original_claim or common.inspect(original_container)['State']!=original_native:raise RuntimeError('manager stop changed native/recorder process')
    common.unit('start')
    reopened=snapshot('manager-reopened')
    if len(reopened)!=1 or worker(reopened[0])!=original_worker or common.cid(common.instance('candidate','recording-zod','running'))!=original_container:raise RuntimeError('manager restart redispatched capture/runtime')
    common.operation('candidate','recording-zod','stop','recording-first-stop')
    ended=snapshot('first-stopped')
    closed(ended[0]);original_bytes=retained_session(ended[0])
    missing=common.docker('inspect',original_container,ok=False)
    if missing.returncode==0:raise RuntimeError('ordinary Stop did not remove its ephemeral container')
    common.operation('candidate','recording-zod','start','recording-second-start')
    second=common.wait(lambda:common.instance('candidate','recording-zod','running'),600)
    if common.cid(second)==original_container or second['pier_path']!=first['pier_path'] or second['ship']!=first['ship']:raise RuntimeError('same-pier replacement mismatch')
    common.operation('candidate','recording-zod','stop','recording-second-stop')
    final=snapshot('second-stopped')
    if len(final)!=2:raise RuntimeError('wrong retained session count')
    for row in final:closed(row)
    if retained_session(ended[0])!=original_bytes:raise RuntimeError('later launch changed original recording bytes')
    report.update(passed=True,qualification={'installed_native_prestart_capture':True,'exact_notify_worker_outside_container':True,'manager_stop_and_reopen_preserve_processes':True,'single_worker_per_launch':True,'terminal_drain_before_container_removal':True,'same_pier_new_session':True,'original_session_unchanged':True,'final_clean_ship_stop':True},final=final)

if __name__=='__main__':
    try:main()
    except BaseException as error:report['error']=type(error).__name__+': '+str(error)
    finally:
        trim_stop.set()
        if trim_thread is not None:trim_thread.join(timeout=12)
        if common is not None:
            try:
                report['manager_unit']=common.run(['systemctl','--user','show','urbit-runner.service','-p','ActiveState,SubState,Result,ExecMainStatus'],True,ok=False).stdout
                common.run(['journalctl','--user','-u','urbit-runner.service','-n','60','--no-pager'],True,ok=False)
                root=HOME/'.local/share/urbit-runner/recording-jobs'
                report['saved_jobs']={p.parent.name+'/'+p.name:json.loads(p.read_text()) for p in root.glob('*/*.json') if p.stat().st_size<=65536}
            except BaseException as error:report['diagnostic_error']=type(error).__name__+': '+str(error)
        if O.is_dir():(O/'result.json').write_text(json.dumps(report,indent=2)+'\n')
        print('OMABIT_RECORDING_VM_RESULT '+json.dumps(report,separators=(',',':')),flush=True)
        try:
            guard();subprocess.run(['fstrim','/'],timeout=60,check=False);subprocess.run(['systemctl','poweroff'],check=True)
        except BaseException:pass
