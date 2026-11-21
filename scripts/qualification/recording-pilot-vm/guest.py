#!/usr/bin/python3
"""Installed ordinary capture/restart/drain; fresh offline guest only."""
import hashlib
import base64
import importlib.util
import json
import os
from pathlib import Path
import subprocess
import sys
import threading
import stat

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
    if value.get('kind')!='issues20-recording-policy-upgrade-v1' or value.get('target_schema')!=65 or not value.get('development_snapshot'):
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

def core(operation,fields):
    args={'name':'local','runner_id':report['runner_id'],**fields}
    result=common.run([I/'candidate/omabit','call',operation,'--input-json',json.dumps(args)],True)
    out=json.loads(result.stdout)
    if out.get('error') or out.get('operation')!=operation:raise RuntimeError('typed native recording operation refused')
    return out['data']

def verify_api_prefix(row):
    selected={'instance_id':'recording-zod','job_id':row['status']['job_id'],'job_sha256':row['status']['job_sha256'],'representation':'original','include_private_data':True}
    counts={'stdout':0,'stderr':0};hashes={'stdout':hashlib.sha256(),'stderr':hashlib.sha256()};exits=0;cursor=None
    for _ in range(256):
        if cursor is not None:selected['cursor']=cursor
        page=core('runtime.recording-page.v1',selected)
        if page['complete_export'] or page['status']['session_id']!=row['status']['session_id']:raise RuntimeError('raw window binding changed')
        for frame in page.get('original',[]):
            kind={1:'stdout',2:'stderr',4:'gap',5:'exit'}.get(frame['frame']['Kind'],'unsupported')
            if kind in counts:
                data=base64.b64decode(frame['original_bytes_base64'],validate=True);counts[kind]+=len(data);hashes[kind].update(data)
            elif kind=='exit':exits+=1
            elif kind=='gap':raise RuntimeError('native recording API exposed a gap')
        cursor=page.get('next')
        if page['tail']!='not-scanned':break
    else:raise RuntimeError('bounded native API scan exceeded')
    if counts['stdout']!=row['stdout_bytes'] or counts['stderr']!=row['stderr_bytes'] or any(hashes[k].hexdigest()!=row[k+'_sha256'] for k in counts) or exits!=1:raise RuntimeError('API bytes differ from independently decoded retained frames')

def withdraw_fixture_override(directory,expected):
    if directory!=HOME/'.config/systemd/user/urbit-runner.service.d' or directory.is_symlink() or not directory.is_dir() or directory.stat().st_uid!=1000:
        raise RuntimeError('only the exact owned fixture override may be withdrawn')
    file=directory/'recording-pilot.conf'
    if sorted(p.name for p in directory.iterdir())!=[file.name] or file.is_symlink() or not file.is_file() or file.stat().st_uid!=1000 or file.read_text()!=expected:
        raise RuntimeError('fixture override changed; preserve it')
    mode=stat.S_IMODE(file.stat().st_mode)
    file.unlink();directory.rmdir()
    return mode

def restore_fixture_override(directory,expected,mode):
    if directory!=HOME/'.config/systemd/user/urbit-runner.service.d':raise RuntimeError('wrong fixture override destination')
    directory.mkdir(mode=0o700)
    file=directory/'recording-pilot.conf'
    with file.open('x')as stream:stream.write(expected)
    os.chmod(file,mode)
    for path in (directory,file):os.chown(path,1000,1000)

def upgrade_with_fixture_override(override,override_content):
    # The installer deliberately preserves/refuses all user overrides. This
    # exact file is created by this fresh fixture; it is not a user deployment.
    override_mode=withdraw_fixture_override(override,override_content)
    report['fixture_override']={'exact_content_sha256':hashlib.sha256(override_content.encode()).hexdigest(),'withdrawn':True,'restored':False,'scope':'only this fresh fixture file; generic user overrides remain unsupported and protected'}
    common.run(['systemctl','--user','daemon-reload'],True)
    try:
        # Normal management updates require the original live listener for
        # online quiescence. Reopen the old manager under its stock invocation;
        # the independently supervised recorder remains unchanged.
        common.unit('start')
        upgraded=common.install('candidate')
        common.unit('stop')
    finally:
        restore_fixture_override(override,override_content,override_mode)
        report['fixture_override']['restored']=True
        common.run(['systemctl','--user','daemon-reload'],True)
    common.unit('start')
    return upgraded

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
    if sha(I/'predecessor-runner')!=scenario['predecessor']['runner_sha256']:raise RuntimeError('predecessor input changed')
    result=common.run([I/'predecessor-runner','install-host','--owner','review','--bundle',I/'previous/rootless-bundle','--public-key',HOME/'operator.pub','--acknowledge-host-changes'],timeout=600)
    host=next(json.loads(line)['host'] for line in result.stdout.splitlines() if json.loads(line).get('host'))
    report['runner_id']=host['runner_id'];common.wait(lambda:common.hello())
    common.docker('load','--input',I/'runtime.oci.tar')
    if common.db_projection()['schema']!=64:raise RuntimeError('wrong installed schema')
    start_guest_trim()
    common.unit('stop')
    override=HOME/'.config/systemd/user/urbit-runner.service.d'
    override.mkdir(mode=0o700,parents=True)
    f=override/'recording-pilot.conf'
    override_content='[Service]\nExecStart=\nExecStart=/usr/bin/env -i HOME=/home/review PATH=/usr/bin:/bin LANG=C XDG_RUNTIME_DIR=/run/user/1000 /usr/local/lib/urbit-runner/current/urbit-runner serve-development --acknowledge-development --record-runtime\n'
    f.write_text(override_content)
    for p in (override,override.parent,override.parent.parent,f):os.chown(p,1000,1000)
    common.run(['systemctl','--user','daemon-reload'],True)
    common.unit('start')
    hello=common.hello()
    if hello['build']['sha256']!=scenario['predecessor']['runner_sha256'] or 'runtime-recording-pilot-v1' not in hello['features']:raise RuntimeError('pilot build/selection mismatch')
    common.run([I/'candidate/urbit-collector','enroll','local','--socket',common.SOCK],True)
    report['hello']=hello
    common.collector('candidate','create-fake','recording-zod','--ship','~zod','--http-port','19480','--request-id','recording-create','--bootstrap','recommended','--pill',str(I/'boot.pill'))
    common.operation('candidate','recording-zod','start','recording-first-start')
    first=common.wait(lambda:common.instance('candidate','recording-zod','running'),600)
    original_container=common.cid(first);original_native=common.inspect(original_container)['State']
    rows=snapshot('first-running')
    if len(rows)!=1 or rows[0]['status']['metadata']['container_id']!=original_container or not rows[0]['status']['writer_present']:raise RuntimeError('missing exact initial capture')
    original_worker=worker(rows[0]);original_claim=rows[0]['status']['worker']
    old_projection=common.projection('candidate');old_runtime=common.runtime('candidate')
    common.unit('stop')
    detached=snapshot('manager-stopped')
    if worker(detached[0])!=original_worker or detached[0]['status']['worker']!=original_claim or common.inspect(original_container)['State']!=original_native:raise RuntimeError('manager stop changed native/recorder process')
    upgraded=upgrade_with_fixture_override(override,override_content)
    if upgraded['runner_id']!=report['runner_id'] or common.db_projection()['schema']!=65:raise RuntimeError('upgrade changed runner or schema')
    if common.projection('candidate')!=old_projection or common.runtime('candidate')!=old_runtime:raise RuntimeError('upgrade changed history, engine or native process')
    if common.hello()['build']['sha256']!=scenario['candidate']['urbit-runner']['sha256']:raise RuntimeError('candidate not activated')
    reopened=snapshot('manager-reopened')
    if len(reopened)!=1 or worker(reopened[0])!=original_worker or common.cid(common.instance('candidate','recording-zod','running'))!=original_container:raise RuntimeError('manager restart redispatched capture/runtime')
    control=core('runtime.recording-controls.v1',{'include_private_data':True})
    if control['capacity_status']!='available' or control['capacity']['jobs']!=1 or control['state']['revision']!=0:raise RuntimeError('original worker capacity attribution changed')
    selected=core('runtime.recordings.v1',{'instance_id':'recording-zod','include_private_data':True})['sessions']
    if len(selected)!=1 or selected[0]['job_id']!=rows[0]['status']['job_id']:raise RuntimeError('native history API differs from exact saved worker')
    common.operation('candidate','recording-zod','stop','recording-first-stop')
    ended=snapshot('first-stopped')
    closed(ended[0]);original_bytes=retained_session(ended[0])
    missing=common.docker('inspect',original_container,ok=False)
    if missing.returncode==0:raise RuntimeError('ordinary Stop did not remove its ephemeral container')
    verify_api_prefix(ended[0])
    original_control=control['state']
    policy=dict(original_control['policy']);policy.update(max_jobs=32,max_session_bytes=32*1024**2,max_segment_bytes=4*1024**2)
    request={'runner_id':report['runner_id'],'request_id':'configure-recording-limits','expected_revision':0,'expected_digest':'','kind':'configure','policy':policy}
    configured=core('runtime.recording-control-apply.v1',{'request':request})
    replayed=core('runtime.recording-control-apply.v1',{'request':request})
    lookup=core('runtime.recording-control-lookup.v1',{'request_id':request['request_id'],'include_private_data':True})
    if configured['replayed'] or not replayed['replayed'] or replayed['receipt']!=configured['receipt'] or lookup.get('receipt')!=configured['receipt']:raise RuntimeError('native control replay/lookup changed receipt')
    head=configured['receipt']['state']
    pin={'runner_id':report['runner_id'],'request_id':'pin-original-recording','expected_revision':head['revision'],'expected_digest':head['digest'],'kind':'pin','job_id':ended[0]['status']['job_id'],'job_sha256':ended[0]['status']['job_sha256'],'pin_reason':'incident'}
    pinned=core('runtime.recording-control-apply.v1',{'request':pin})
    common.operation('candidate','recording-zod','start','recording-second-start')
    second=common.wait(lambda:common.instance('candidate','recording-zod','running'),600)
    if common.cid(second)==original_container or second['pier_path']!=first['pier_path'] or second['ship']!=first['ship']:raise RuntimeError('same-pier replacement mismatch')
    common.operation('candidate','recording-zod','stop','recording-second-stop')
    final=snapshot('second-stopped')
    if len(final)!=2:raise RuntimeError('wrong retained session count')
    for row in final:closed(row)
    if retained_session(ended[0])!=original_bytes:raise RuntimeError('later launch changed original recording bytes')
    verify_api_prefix(next(row for row in final if row['status']['job_id']!=ended[0]['status']['job_id']))
    jobs={p.parent.name:json.loads(p.read_text()) for p in (HOME/'.local/share/urbit-runner/recording-jobs').glob('*/job.json')}
    if jobs[ended[0]['status']['job_id']]['limits']['max_session_bytes']!=64*1024**2:raise RuntimeError('original worker limits changed')
    new_job=next(j for j in jobs.values() if j['id']!=ended[0]['status']['job_id'])
    if new_job['limits']['max_session_bytes']!=32*1024**2 or new_job['limits']['max_segment_bytes']!=4*1024**2:raise RuntimeError('new launch ignored selected limits')
    final_control=core('runtime.recording-controls.v1',{'include_private_data':True})
    if final_control['state']!=pinned['receipt']['state'] or final_control['capacity']['committed_bytes']!=96*1024**2:raise RuntimeError('pins or immutable capacity commitments changed')
    common.unit('stop')
    dbroot=HOME/'.local/share/urbit-runner/managed'
    before_refusal={p.name:sha(p) for p in dbroot.glob('runner.sqlite*') if p.is_file()}
    if set(before_refusal)!={'runner.sqlite'}:raise RuntimeError('closed database not checkpointed for old-reader gate')
    refused=common.run([I/'predecessor-runner','serve-development','--acknowledge-development','--record-runtime'],True,ok=False,timeout=20)
    if refused.returncode==0 or 'state database requires compatible runner support (schema 64)' not in refused.stderr:raise RuntimeError('old native reader did not refuse schema65')
    after_refusal={p.name:sha(p) for p in dbroot.glob('runner.sqlite*') if p.is_file()}
    if before_refusal!=after_refusal:raise RuntimeError('old reader changed refused closed database')
    common.unit('start')
    if core('runtime.recording-controls.v1',{'include_private_data':True})['state']!=pinned['receipt']['state']:raise RuntimeError('reopen changed control history')
    report['control_qualification']={'schema64_to65_active_worker_upgrade':True,'actual_old_reader_refused_unchanged':True,'engine_runtime_worker_preserved':True,'native_history_and_raw_page_exact':True,'original_control_replay_lookup':True,'incident_pin_preserved':True,'new_job_uses_selected_limits':True,'old_job_limits_unchanged':True,'committed_bytes':final_control['capacity']['committed_bytes']}

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
