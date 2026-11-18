#!/usr/bin/python3
"""Exact development snapshot/real Stop gate; root only in the fresh offline VM."""
import hashlib
import json
import os
from pathlib import Path
import shutil
import signal
import sqlite3
import subprocess
import sys
import time

I = Path('/opt/omabit-release-input')
O = Path('/opt/omabit-issues-results')
HOME = Path('/home/review')
CONTROL = HOME/'issues-recovery-helper'
SOCK = '/run/user/1000/urbit-runner/runner.sock'
report = {'format':1,'passed':False,'scope':'offline Linux/AMD64 development snapshot; fresh fakes only; not a stable release or production/DEV update'}
scenario = None
helper = None


def sha(path):
    with Path(path).open('rb') as f:
        return hashlib.file_digest(f,'sha256').hexdigest()


def guard():
    if os.geteuid()!=0 or Path('/sys/class/dmi/id/sys_vendor').read_text().strip()!='QEMU' or 'omabit.disposable-guest=release-upgrade-v1' not in Path('/proc/cmdline').read_text().split():
        raise RuntimeError('refused outside the fresh offline VM')
    value = json.loads((I/'scenario.json').read_text())
    if value.get('kind')!='issues16-21-development-v1' or not value.get('development_snapshot') or sys.argv[1:]:
        raise RuntimeError('wrong explicit scenario')
    return value


def owner_args(args):
    return ['runuser','-u','review','--','env','HOME=/home/review','XDG_RUNTIME_DIR=/run/user/1000','DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/1000/bus','PATH=/usr/bin:/bin',*[str(x) for x in args]]


def run(args, owner=False, timeout=90, ok=True):
    args = [str(x) for x in args]
    print('ISSUES_VM_STEP '+json.dumps(args[:5]),flush=True)
    result = subprocess.run(owner_args(args) if owner else args,capture_output=True,text=True,timeout=timeout)
    with (O/'commands.jsonl').open('a') as f:
        f.write(json.dumps({'args':args,'exit':result.returncode,'out':result.stdout[-24000:],'err':result.stderr[-16000:]})+'\n')
    if ok and result.returncode:
        raise RuntimeError('command failed: '+args[0]+'; private commands.jsonl retains diagnostics')
    return result


def collector(version, *args, pin=False, ok=True, timeout=90):
    selection = ['--runner',report.get('connection','local'),'--expected-runner-id',report['runner_id']] if pin else ['--socket',SOCK]
    result = run([I/version/'urbit-collector',*selection,*args],True,timeout,ok)
    return json.loads(result.stdout) if result.returncode==0 else result


def wait(fn, seconds=180):
    end = time.monotonic()+seconds
    while time.monotonic()<end:
        value = fn()
        if value:
            return value
        time.sleep(1)
    raise RuntimeError('bounded guest readiness expired')


def hello(version='candidate'):
    # Readiness tolerates only transient transport unavailability, not mutations.
    result = run([I/version/'urbit-collector','--socket',SOCK,'hello'],True,ok=False,timeout=20)
    return json.loads(result.stdout) if result.returncode==0 else None


def instance(version, name, state=None):
    row = next(x for x in collector(version,'snapshot')['instances'] if x['id']==name)
    if state and (row['observed_state']!=state or row.get('active_operation_id')):
        return None
    return row


def operation(version, name, action, request):
    row = instance(version,name)
    collector(version,'operate',name,action,'--revision',str(row['config_revision']),'--request-id',request)
    def done():
        op = next((x for x in collector(version,'snapshot')['operations'] if x['request_id']==request),None)
        if not op or op['status'] in ('queued','running','pending'):
            return None
        if op['status']!='succeeded':
            report['failed_operation'] = op
            raise RuntimeError('native standalone operation did not succeed: '+request)
        return op
    return wait(done,600)


def projection(version):
    s = collector(version,'snapshot')
    fields = ('id','ship','mode','label','config','config_revision','applied_revision','desired_state','pier_path','bootstrap')
    return {'runner_id':s['runner_id'],'instances':[{k:r.get(k) for k in fields} for r in s['instances']],'operations':s['operations']}


def cid(row):
    return (row.get('runtime_status') or row.get('resources') or {})['container_id']


def docker(*args, **kwargs):
    return run(['docker','--host','unix:///run/user/1000/urbit-docker.sock',*args],True,**kwargs)


def inspect(container):
    return json.loads(docker('inspect',container).stdout)[0]


def runtime(version):
    rows = collector(version,'snapshot')['instances']
    return {'engine':run(['systemctl','--user','show','urbit-docker.service','-p','MainPID,InvocationID'],True).stdout,
            'containers':{x['id']:{k:inspect(cid(x))['State'][k] for k in ('Pid','StartedAt','Running')} for x in rows}}


def db_projection():
    with sqlite3.connect('file:/home/review/.local/share/urbit-runner/managed/runner.sqlite?mode=ro',uri=True) as db:
        return {'schema':db.execute('pragma user_version').fetchone()[0], 'sha256':hashlib.sha256('\n'.join(db.iterdump()).encode()).hexdigest()}


def install(version):
    result = run([I/version/'urbit-runner','install-host','--owner','review','--bundle',I/'previous/rootless-bundle','--public-key',HOME/'operator.pub','--acknowledge-host-changes'],timeout=600)
    host = next(json.loads(line)['host'] for line in result.stdout.splitlines() if json.loads(line).get('host'))
    wait(lambda:hello(version))
    return host


def unit(action):
    run(['systemctl','--user',action,'urbit-runner.service'],True)
    if action=='start':
        wait(lambda:hello())


def start_helper(mode):
    global helper
    if helper is not None:
        raise RuntimeError('one owned helper at a time')
    log = (O/(mode+'.log')).open('wb')
    helper = subprocess.Popen(owner_args(['env','OMABIT_ISSUES_VM_HELPER='+mode,I/'managed.test','-test.run','^TestIssuesStopRecoveryDisposableVM$','-test.count=1','-test.timeout=10m','-test.v']),stdout=log,stderr=subprocess.STDOUT,start_new_session=True)
    def ready():
        if helper.poll() is not None:
            raise RuntimeError('owned VM helper exited before readiness')
        path = CONTROL/(mode+'-ready.json')
        return json.loads(path.read_text()) if path.exists() else None
    proof = wait(ready,30)
    # runuser stays as parent; the exact Go child is the one publishing readiness.
    pid = proof['pid']
    stat = Path('/proc/'+str(pid)+'/status').read_text()
    if '\nUid:\t1000\t1000\t1000\t1000\n' not in stat or sha('/proc/'+str(pid)+'/exe')!=scenario['candidate']['managed.test']['sha256']:
        raise RuntimeError('owned helper executable/UID evidence changed')
    helper._owned_child = pid
    helper._owned_start = Path('/proc/'+str(pid)+'/stat').read_text().rsplit(')',1)[1].split()[19]
    return proof


def stop_helper(sig):
    global helper
    if helper is None or helper.poll() is not None:
        raise RuntimeError('owned helper is not active')
    pid = helper._owned_child
    if sha('/proc/'+str(pid)+'/exe')!=scenario['candidate']['managed.test']['sha256'] or Path('/proc/'+str(pid)+'/stat').read_text().rsplit(')',1)[1].split()[19]!=helper._owned_start:
        raise RuntimeError('owned helper executable changed before signal')
    os.kill(pid,sig)
    status = helper.wait(timeout=20)
    helper = None
    if sig==signal.SIGTERM and status!=0:
        raise RuntimeError('VM helper did not shut down cleanly')
    return status


def project_status():
    return collector('candidate','project-status',report['project_id'],pin=True)


def project_action(kind, member, request, container):
    status = project_status()
    return collector('candidate','project-operate',report['project_id'],kind,'--member',member,'--container-id',container,'--request-id',request,'--revision',str(status['project']['revision']),'--review-digest',status['project']['review_digest'],'--confirm-action',pin=True)


def project_done(request):
    def done():
        op = next((x for x in project_status()['operations'] if x['request_id']==request),None)
        if not op or op['status'] in ('queued','running'):
            return None
        if op['status']!='succeeded':
            report['failed_operation'] = op
            raise RuntimeError('project phase did not succeed: '+request)
        return op
    return wait(done,600)


def main():
    global scenario
    scenario = guard()
    if O.exists():
        raise RuntimeError('never reset or resume a guest')
    O.mkdir(mode=0o700)
    for path in ('/etc/urbit-runner','/srv/urbit/piers','/home/review/.local/share/urbit-runner'):
        if Path(path).exists():
            raise RuntimeError('guest is not pristine: '+path)
    for version in ('previous','candidate'):
        for name in ('omabit','urbit-collector','urbit-runner'):
            expected = scenario['previous'][name+'-linux-amd64'] if version=='previous' else scenario['candidate'][name]['sha256']
            assert sha(I/version/name)==expected
    assert sha(I/'managed.test')==scenario['candidate']['managed.test']['sha256']
    assert sha(I/'runtime.oci.tar')==scenario['oci_sha256'] and sha(I/'boot.pill')==scenario['pill_sha256'] and sha(I/'recommended.pill')==scenario['recommended_sha256']
    report['source_commit'] = scenario['source_commit']
    report['inputs'] = scenario
    run(['loginctl','enable-linger','review'])
    run(['systemctl','start','user@1000.service'])
    run(['ssh-keygen','-q','-t','ed25519','-N','','-f',HOME/'operator'],True)
    previous_host = install('previous')
    report['runner_id'] = previous_host['runner_id']
    run([I/'previous/urbit-collector','enroll','local','--socket',SOCK],True)
    docker('load','--input',I/'runtime.oci.tar')
    report['guest_virtual_cpus'] = json.loads(docker('info','--format','{{json .NCPU}}').stdout)
    for name, ship, port in (('upgrade-zod','~zod',19180),('upgrade-nec','~nec',19181)):
        collector('previous','create-fake',name,'--ship',ship,'--http-port',str(port),'--request-id',name+'-create','--bootstrap','recommended','--pill',str(I/'recommended.pill'))
        operation('previous',name,'start',name+'-start')
        wait(lambda:instance('previous',name,'running'),600)
        if name=='upgrade-nec':
            operation('previous',name,'stop',name+'-stop')
            wait(lambda:instance('previous',name,'stopped'))
    before, runtime_before = projection('previous'), runtime('previous')
    access = {p.name:sha(p) for p in (HOME/'operator',HOME/'operator.pub',Path('/etc/urbit-runner/authorized_keys'))}
    report['schema_before'] = db_projection()['schema']
    assert report['schema_before']==59
    install('candidate')
    assert projection('candidate')==before and runtime('candidate')==runtime_before
    assert access=={p.name:sha(p) for p in (HOME/'operator',HOME/'operator.pub',Path('/etc/urbit-runner/authorized_keys'))}
    exact_hello = hello()
    assert exact_hello['build']['sha256']==scenario['candidate']['urbit-runner']['sha256']
    report['installed_upgrade'] = {'passed':True,'running_stopped_intent_and_history_preserved':True,'engine_and_container_processes_preserved':True,'access_preserved':True,'hello':exact_hello}
    unit('stop')
    newer = db_projection()
    assert newer['schema']==60
    refused = run([I/'previous/urbit-runner','serve-development','--acknowledge-development'],True,ok=False,timeout=20)
    assert refused.returncode!=0 and 'schema' in (refused.stdout+refused.stderr).lower() and db_projection()==newer
    report['installed_upgrade']['old_reader_refused_unchanged'] = True
    unit('start')
    assert projection('candidate')==before and runtime('candidate')==runtime_before
    # Stop the upgrade witness before another new fake boots; all history stays.
    operation('candidate','upgrade-zod','stop','upgrade-zod-final-stop')
    wait(lambda:instance('candidate','upgrade-zod','stopped'))
    # Explicit fixture cache population from the already rehashed immutable pill.
    cache = HOME/'.local/share/urbit-runner/managed/boot-pills'
    cache.mkdir(mode=0o700,exist_ok=True)
    os.chown(cache,1000,1000)
    dest = cache/(scenario['pill_sha256']+'.pill')
    assert not dest.exists()
    shutil.copy2(I/'boot.pill',dest)
    dest.chmod(0o400)
    os.chown(dest,1000,1000)
    # A freshly authorized restricted SSH connection exercises the guarded bridge
    # and establishes the same actor for Stop, diagnosis, apply and exact replay.
    public = run(['ssh-keyscan','-T','5','-p','22222','127.0.0.1'],True).stdout
    assert '[127.0.0.1]:22222 '+previous_host['host_public_key'] in public
    known = HOME/'issues-known-hosts'
    known.write_text(public)
    config = HOME/'issues-ssh-config'
    config.write_text('Host issues-runner\n  HostName 127.0.0.1\n  Port 22222\n  User review\n  IdentityFile /home/review/operator\n  IdentityAgent none\n  IdentitiesOnly yes\n  UserKnownHostsFile /home/review/issues-known-hosts\n  GlobalKnownHostsFile /dev/null\n  StrictHostKeyChecking yes\n  BatchMode yes\n')
    for path in (known,config):
        path.chmod(0o600)
        os.chown(path,1000,1000)
    run([I/'candidate/urbit-collector','enroll','bridge','--ssh','issues-runner','--ssh-config',config],True)
    report['connection'] = 'bridge'
    create = run([I/'candidate/urbit-collector','--runner','bridge','--expected-runner-id',report['runner_id'],
        'project-create','--label','issues recovery fresh network','--member','issues-original-zod:~zod:19280','--request-id','issues-create','--shared-administration','--confirm-start'],True,timeout=900)
    progress = [json.loads(line) for line in create.stdout.splitlines()]
    assert progress[-1]['phase']=='ready'
    report['project_id'] = progress[-1]['project_id']
    original_before = instance('candidate','issues-original-zod','running')
    original_process = inspect(cid(original_before))['State']
    added = run([I/'candidate/urbit-collector','--runner','bridge','--expected-runner-id',report['runner_id'],'project-add',report['project_id'],
        '--member','issues-added-nec:~nec:19281','--request-id','issues-add-nec','--confirm-start'],True,timeout=900)
    assert json.loads(added.stdout.splitlines()[-1])['phase']=='ready'
    assert inspect(cid(original_before))['State']==original_process,'adding a peer restarted the original'
    added_before = instance('candidate','issues-added-nec','running')
    selected_container = cid(added_before)
    native_before = inspect(selected_container)
    CONTROL.mkdir(mode=0o700)
    os.chown(CONTROL,1000,1000)
    selection = CONTROL/'selection.json'
    selection.write_text(json.dumps({'project':report['project_id']})+'\n')
    selection.chmod(0o600)
    os.chown(selection,1000,1000)
    unit('stop')
    report['injection_helper'] = start_helper('interrupt-stop')
    stop = project_action('member-stop','issues-added-nec','issues-original-stop',selected_container)
    marker = CONTROL/'clean-exit-before-save.json'
    wait(lambda:marker.exists(),360)
    effect = json.loads(marker.read_text())
    assert effect['operation']==stop['operation']['id'] and effect['container']==selected_container
    native_stopped = inspect(selected_container)
    assert not native_stopped['State']['Running'] and native_stopped['State']['ExitCode']==0 and not native_stopped['State']['OOMKilled']
    report['interrupted_helper_status'] = stop_helper(signal.SIGKILL)
    unit('start')
    original = next(x for x in project_status()['operations'] if x['id']==stop['operation']['id'])
    assert original['status']=='interrupted'
    blocked = collector('candidate','project-operate',report['project_id'],'member-start','--member','issues-added-nec','--container-id',selected_container,
        '--request-id','issues-must-stay-fenced','--revision',str(project_status()['project']['revision']),'--review-digest',project_status()['project']['review_digest'],'--confirm-action',pin=True,ok=False)
    assert isinstance(blocked,subprocess.CompletedProcess) and blocked.returncode!=0
    diagnosis = collector('candidate','project-diagnose',report['project_id'],'--member','issues-added-nec','--operation',original['id'],pin=True)
    assert diagnosis['runtime_observation']=='not-performed' and diagnosis['phase']=='stop-dispatched'
    diagnosis = collector('candidate','project-diagnose',report['project_id'],'--member','issues-added-nec','--operation',original['id'],'--probe',pin=True)
    assert diagnosis['reason_code']=='stop_clean_exit_verified' and diagnosis['review']['evidence_basis']=='retained-stop-intent'
    review = HOME/'issues-stop-review.json'
    review.write_text(json.dumps(diagnosis)+'\n')
    review.chmod(0o600)
    os.chown(review,1000,1000)
    reconcile_args = ('project-reconcile-stop',report['project_id'],'--review',str(review),'--request-id','issues-stop-resolution','--confirm-action')
    resolution = collector('candidate',*reconcile_args,pin=True)
    assert resolution['operation']['status']=='succeeded' and resolution['original_operation_id']==original['id'] and resolution['operation']['id']!=original['id']
    assert next(x for x in project_status()['operations'] if x['id']==original['id'])==original
    assert inspect(selected_container)['State']==native_stopped['State'],'reconciliation touched native process'
    unit('stop')
    start_helper('lookup-no-observer')
    replay = collector('candidate',*reconcile_args,'--replay-only',pin=True)
    assert replay==resolution and not (CONTROL/'unexpected-recovery-observer.json').exists()
    assert next(x for x in project_status()['operations'] if x['id']==original['id'])==original
    stop_helper(signal.SIGTERM)
    unit('start')
    assert collector('candidate',*reconcile_args,'--replay-only',pin=True)==resolution
    # Filter only this owned stopped container. EOF/submission is not evidence
    # of a signal; require exactly one native SIGTERM event since its creation.
    events = docker('events','--since',native_before['Created'],'--until',str(int(time.time())+1),'--filter','container='+selected_container,'--filter','event=kill','--format','{{json .}}',timeout=10).stdout
    kills = [json.loads(line) for line in events.splitlines() if line.strip()]
    assert len(kills)==1 and kills[0]['Actor']['Attributes']['signal'] in ('15','SIGTERM')
    report['stop_recovery'] = {'passed':True,'test_only_fault':'after real clean exit, before final phase save; helper killed, no database editing',
        'original':original,'resolution':resolution,'diagnosis':diagnosis,'single_sigterm':True,'no_signal_or_start_on_reconciliation':True,
        'exact_replay_after_manager_and_store_reopen':True,'ssh_guarded_lookup':True,'replay_with_refusing_observer':True,'effect_marker':effect}
    project_action('member-start','issues-added-nec','issues-resume-same-pier',selected_container)
    project_done('issues-resume-same-pier')
    resumed = wait(lambda:instance('candidate','issues-added-nec','running'),180)
    assert cid(resumed)!=selected_container and resumed['ship']==added_before['ship'] and resumed['pier_path']==added_before['pier_path']
    after_start = inspect(cid(resumed))
    before_mounts = [(m['Source'],m['Destination']) for m in native_before['Mounts'] if m['Destination']=='/urbit']
    after_mounts = [(m['Source'],m['Destination']) for m in after_start['Mounts'] if m['Destination']=='/urbit']
    assert before_mounts==after_mounts and before_mounts
    assert inspect(cid(original_before))['State']==original_process,'recovery disturbed peer'
    report['stop_recovery']['same_pier_resume'] = True
    for name in ('issues-added-nec','issues-original-zod'):
        row = instance('candidate',name,'running')
        project_action('member-stop',name,'issues-final-stop-'+name,cid(row))
        project_done('issues-final-stop-'+name)
        wait(lambda:instance('candidate',name,'stopped'),180)
    final = collector('candidate','snapshot')
    assert all(x['observed_state']=='stopped' for x in final['instances'])
    report['final_clean_ship_shutdown'] = True
    report['final_installed_hello'] = hello()
    assert report['final_installed_hello']['build']['sha256']==scenario['candidate']['urbit-runner']['sha256']
    report['schema_after'] = db_projection()['schema']
    report['passed'] = True


if __name__ == '__main__':
    try:
        main()
    except BaseException as error:
        report['error'] = type(error).__name__+': '+str(error)
    finally:
        if helper is not None and helper.poll() is None:
            try:
                report['helper_final_termination'] = stop_helper(signal.SIGTERM)
            except BaseException as error:
                report['helper_cleanup_error'] = type(error).__name__
        if O.is_dir():
            (O/'result.json').write_text(json.dumps(report,indent=2)+'\n')
        print('OMABIT_ISSUES_VM_RESULT '+json.dumps(report,separators=(',',':')),flush=True)
        # This is the freshly booted QEMU guest, after the guard above. Failure
        # evidence remains on its private overlay. Never power off a host.
        try:
            guard()
            subprocess.run(['systemctl','poweroff'],check=True)
        except BaseException:
            pass
