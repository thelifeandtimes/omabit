#!/usr/bin/python3
"""Actual Stop observation transition in a fixed offline COW, with two new fakes."""
import contextlib
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import signal
import sqlite3
import subprocess
import sys
import time

import status

I = Path('/opt/omabit-release-input')
R = Path('/run/omabit-runtime-candidate')
O = Path('/opt/omabit-issues-project-race-results')
CONTROL = Path('/home/review/issues-recovery-helper')
REGISTRY = '/home/review/omabit-app-runtime-recovery/registry'
PIN = '0c28d526285e5b4e055e209e9021e5c683505ddabb5f0ce661678052949d141c'
CASE = 'preliminary-exit-transition-v1'
report = {'format': 1, 'passed': False, 'scope': 'fresh offline COW; fixture-only SIGSTOP/SIGCONT with one real SIGTERM; actual preliminary/bracketed native transition, not proof of original incident predicate or installed upgrade'}
common = None
scenario = None


def sha(path):
    with Path(path).open('rb') as f:
        return hashlib.file_digest(f, 'sha256').hexdigest()


def guard():
    if os.geteuid() != 0 or sys.argv[1:]:
        raise RuntimeError('root in fixed offline guest only')
    if Path('/sys/class/dmi/id/sys_vendor').read_text().strip() != 'QEMU' or 'omabit.disposable-guest=release-upgrade-v1' not in Path('/proc/cmdline').read_text().split():
        raise RuntimeError('not isolated guest')
    value = json.loads((I/'scenario.json').read_text())
    if value.get('kind') != 'issues16-native-transition-v1' or value.get('case') != CASE or value.get('predecessor_root_sha256') != PIN or value.get('original_schema') != 65 or value.get('target_schema') != 65 or any(value.get(k) is not True for k in ('development_snapshot', 'require_native_transition', 'preserve_stopped_base_instance')):
        raise RuntimeError('wrong source fixture')
    return value


def retained_state():
    # Only the cloned disposable store: read, never repair or change its marker.
    with contextlib.closing(sqlite3.connect('file:/home/review/.local/share/urbit-runner/managed/runner.sqlite?mode=ro', uri=True)) as db:
        db.execute('BEGIN')
        if db.execute('pragma user_version').fetchone()[0] != 65:
            raise RuntimeError('wrong fixture schema')
        tables = [r[0] for r in db.execute("select name from sqlite_master where type='table' and name not like 'sqlite_%'")]
        app = {table: [[{'bytes_hex': cell.hex()} if isinstance(cell, bytes) else cell for cell in row] for row in db.execute('select * from "'+table+'"')] for table in tables if table.startswith(('grubbery_app_', 'gall_app_'))}
        instances = [json.loads(row[0]) for row in db.execute('select body from instances')]
        old = next(row for row in instances if row['public']['id'] == 'grubbery-zod')
        p = old['public']
        if p['ship'] != '~zod' or p['desired_state'] != 'stopped' or p['observed_state'] != 'stopped' or p.get('active_operation_id'):
            raise RuntimeError('inherited app fake must remain stopped')
        return {'apps': app, 'base': {k: p.get(k) for k in ('id', 'ship', 'mode', 'label', 'config', 'config_revision', 'applied_revision', 'desired_state', 'pier_path', 'bootstrap')}, 'base_runtime': old['runtime'], 'instance_count': len(instances)}


def inflate(value):
    import gzip
    if R.exists() or R.is_symlink():
        raise RuntimeError('fresh memory-backed executable directory required')
    R.mkdir(mode=0o755)
    (R/'candidate').mkdir(mode=0o755)
    if set(value['candidate']) != {'omabit', 'urbit-collector', 'urbit-runner', 'managed.test'}:
        raise RuntimeError('exact matching executable set required')
    for name, pin in value['candidate'].items():
        relative = 'candidate/'+name if name != 'managed.test' else name
        source = I/(relative+'.gz')
        if source.is_symlink() or not source.is_file() or not 0 < pin['bytes'] < 64*1024**2:
            raise RuntimeError('ordinary bounded compressed executable required')
        target = R/relative
        with gzip.open(source, 'rb') as stream, target.open('xb') as output:
            left = pin['bytes']
            while left:
                block = stream.read(min(1<<20, left))
                if not block:
                    raise RuntimeError('truncated executable')
                output.write(block)
                left -= len(block)
            if stream.read(1):
                raise RuntimeError('executable exceeds declared size')
        target.chmod(0o755)
        if sha(target) != pin['sha256']:
            raise RuntimeError('inflated executable differs')


def collector(version, *args, pin=False, ok=True, timeout=90):
    selection = ['--runner', 'fixture', '--expected-runner-id', report['runner_id']] if pin else ['--socket', common.SOCK]
    result = common.run([R/version/'urbit-collector', '--config-dir', REGISTRY, *selection, *args], True, timeout, ok)
    return json.loads(result.stdout) if result.returncode == 0 else result


def process(container):
    data = common.inspect(container)
    if not data['State']['Running']:
        raise RuntimeError('expected exact live fixture process')
    return {k: data['State'][k] for k in ('Pid', 'StartedAt', 'Running')}


def anchor_id():
    with contextlib.closing(sqlite3.connect('file:/home/review/.local/share/urbit-runner/managed/runner.sqlite?mode=ro', uri=True)) as db:
        # The saved ledger is diagnostic only. Its path is never supplied by a caller.
        rows = list(db.execute('select body from project_expansions where project_id=?', (report['project_id'],)))
    if len(rows) != 1:
        raise RuntimeError('one exact expansion ledger required')
    return json.loads(rows[0][0])['anchor']['prepared']['anchor']['id']


def start_helper():
    log = (O/'native-race-helper.log').open('xb')
    try:
        common.helper = subprocess.Popen(common.owner_args(['env', 'OMABIT_PROJECT_RACE_VM='+CASE, R/'managed.test', '-test.run=^TestIssuesProjectRaceDisposableVM$', '-test.v', '-test.timeout=600s']), stdout=log, stderr=subprocess.STDOUT, start_new_session=True)
    finally:
        log.close()
    def ready():
        if common.helper.poll() is not None:
            raise RuntimeError('native race helper exited before readiness')
        path = CONTROL/'native-race-ready.json'
        return json.loads(path.read_text()) if path.exists() else None
    proof = common.wait(ready, 30)
    pid = proof['pid']
    if proof['source_commit'] != scenario['source_commit'] or proof['project'] != report['project_id'] or proof['build_tag'] != 'omabit_disposable_qualification' or '\nUid:\t1000\t1000\t1000\t1000\n' not in Path('/proc/'+str(pid)+'/status').read_text() or sha('/proc/'+str(pid)+'/exe') != scenario['candidate']['managed.test']['sha256']:
        raise RuntimeError('owned tagged helper identity changed')
    common.helper._owned_child = pid
    common.helper._owned_start = Path('/proc/'+str(pid)+'/stat').read_text().rsplit(')', 1)[1].split()[19]
    return proof


def main():
    global common, scenario
    scenario = guard()
    if O.exists() or CONTROL.exists():
        raise RuntimeError('never resume a derived capsule')
    O.mkdir(mode=0o700)
    before = retained_state()
    if before['instance_count'] != 1:
        raise RuntimeError('only the original stopped fixture may be inherited')
    inflate(scenario)
    spec = importlib.util.spec_from_file_location('issues_common', I/'issues-common.py')
    common = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(common)
    common.I, common.O, common.report, common.scenario = R, O, report, scenario
    common.collector = collector
    report.update(inputs=scenario, source_commit=scenario['source_commit'])
    common.run(['systemctl', 'start', 'user@1000.service'])
    properties = common.run(['systemctl', '--user', 'show', 'urbit-runner.service', '-p', 'ActiveState,ExecStart'], True).stdout
    if 'ActiveState=active' in properties or '--resume-running' in properties or str(R/'candidate/urbit-runner') not in properties:
        raise RuntimeError('source manager could auto-resume or select old executable')
    Path('/run/omabit-runtime-runner-ready').touch(mode=0o600)
    common.unit('start')
    hello = common.wait(common.hello, 60)
    if hello['build']['sha256'] != scenario['candidate']['urbit-runner']['sha256']:
        raise RuntimeError('wrong actual service executable')
    report['runner_id'] = hello['runner_id']
    pill = Path('/home/review/.local/share/urbit-runner/managed/boot-pills')/(scenario['pill_sha256']+'.pill')
    if pill.is_symlink() or not pill.is_file() or sha(pill) != scenario['pill_sha256']:
        raise RuntimeError('inherited exact immutable bootstrap cache unavailable')
    if retained_state() != before:
        raise RuntimeError('opening changed inherited app state')
    selection = [R/'candidate/urbit-collector', '--config-dir', REGISTRY, '--runner', 'fixture', '--expected-runner-id', report['runner_id']]
    create = common.run([*selection, 'project-create', '--label', 'issues native transition', '--member', 'issues-race-nec:~nec:19580', '--request-id', 'issues-race-create', '--shared-administration', '--confirm-start'], True, timeout=1200)
    progress = [json.loads(line) for line in create.stdout.splitlines()]
    if progress[-1]['phase'] != 'ready':
        raise RuntimeError('new original member did not become ready')
    report['project_id'] = progress[-1]['project_id']
    peer = common.instance('candidate', 'issues-race-nec', 'running')
    peer_id, peer_process = common.cid(peer), process(common.cid(peer))
    added = common.run([*selection, 'project-add', report['project_id'], '--member', 'issues-race-bud:~bud:19581', '--request-id', 'issues-race-add', '--confirm-start'], True, timeout=1200)
    if json.loads(added.stdout.splitlines()[-1])['phase'] != 'ready' or process(peer_id) != peer_process:
        raise RuntimeError('addition failed or changed original peer')
    target = common.instance('candidate', 'issues-race-bud', 'running')
    target_id = common.cid(target)
    native_before = common.inspect(target_id)
    anchor = anchor_id()
    anchor_process = process(anchor)
    CONTROL.mkdir(mode=0o700)
    os.chown(CONTROL, 1000, 1000)
    path = CONTROL/'selection.json'
    path.write_text(json.dumps({'project': report['project_id']})+'\n')
    path.chmod(0o600)
    os.chown(path, 1000, 1000)
    common.unit('stop')
    report['helper'] = start_helper()
    selected_status = common.project_status()
    stop_args = ('project-operate', report['project_id'], 'member-stop', '--member', 'issues-race-bud', '--container-id', target_id, '--request-id', 'issues-race-stop', '--revision', str(selected_status['project']['revision']), '--review-digest', selected_status['project']['review_digest'], '--confirm-action')
    stop = collector('candidate', *stop_args, pin=True)
    terminal = common.project_done('issues-race-stop')
    if terminal['id'] != stop['operation']['id']:
        raise RuntimeError('terminal Stop lost original operation binding')
    report['terminal_stop_succeeded'] = True
    names = ('pause-intent', 'paused', 'term-dispatched', 'preliminary-running', 'continued', 'exited-before-bracket')
    traces = [json.loads((CONTROL/f'race-{n:02d}-{step}.json').read_text()) for n, step in enumerate(names, 1)]
    if len(list(CONTROL.glob('race-*.json'))) != 6 or [x['step'] for x in traces] != list(names) or any(x['operation_id'] != terminal['id'] or x['container_id'] != target_id or x['started_at'] != traces[0]['started_at'] for x in traces):
        raise RuntimeError('missing/foreign transition trace')
    stopped = common.inspect(target_id)['State']
    if stopped['Running'] or stopped['Paused'] or stopped['OOMKilled'] or stopped['ExitCode'] != 0 or stopped.get('Error'):
        raise RuntimeError('native target did not stop cleanly')
    report['native_transition'] = True
    events = common.docker('events', '--since', native_before['Created'], '--until', str(int(time.time())+1), '--filter', 'container='+target_id, '--filter', 'event=kill', '--format', '{{json .}}', timeout=10).stdout
    kills = [json.loads(line)['Actor']['Attributes']['signal'] for line in events.splitlines() if line.strip()]
    if [dict(SIGSTOP='19', SIGTERM='15', SIGCONT='18').get(x, x) for x in kills] != ['19', '15', '18']:
        raise RuntimeError('unexpected native signal sequence; no repeat TERM or force-kill permitted')
    report['single_sigterm'] = True
    report['helper_exit'] = common.stop_helper(signal.SIGTERM)
    common.unit('start')
    if next(x for x in common.project_status()['operations'] if x['id'] == terminal['id']) != terminal:
        raise RuntimeError('exact terminal receipt changed across manager/store reopen')
    replay = collector('candidate', *stop_args, '--replay-only', pin=True)
    if replay['operation'] != terminal or common.inspect(target_id)['State'] != stopped:
        raise RuntimeError('same-request replay changed receipt or native state')
    report['receipt_replay_after_restart'] = True
    common.project_action('member-start', 'issues-race-bud', 'issues-race-resume', target_id)
    common.project_done('issues-race-resume')
    resumed = common.wait(lambda: common.instance('candidate', 'issues-race-bud', 'running'), 180)
    after = common.inspect(common.cid(resumed))
    mounts = lambda x: [(m['Source'], m['Destination']) for m in x['Mounts'] if m['Destination'] == '/urbit']
    if common.cid(resumed) == target_id or resumed['ship'] != target['ship'] or resumed['pier_path'] != target['pier_path'] or not mounts(after) or mounts(after) != mounts(native_before):
        raise RuntimeError('supported Start did not retain exact same pier')
    report['same_pier_start'] = True
    if process(peer_id) != peer_process or process(anchor) != anchor_process:
        raise RuntimeError('native transition/Start disturbed peer or anchor')
    report['peer_anchor_preserved'] = True
    for name in ('issues-race-bud', 'issues-race-nec'):
        row = common.instance('candidate', name, 'running')
        common.project_action('member-stop', name, 'issues-race-final-stop-'+name, common.cid(row))
        common.project_done('issues-race-final-stop-'+name)
        common.wait(lambda: common.instance('candidate', name, 'stopped'), 180)
    if any(x['observed_state'] != 'stopped' or x['desired_state'] != 'stopped' for x in collector('candidate', 'snapshot')['instances']):
        raise RuntimeError('not every fixture fake is cleanly stopped')
    report['final_clean_stops'] = True
    after = retained_state()
    if after['apps'] != before['apps'] or after['base'] != before['base'] or after['base_runtime'] != before['base_runtime'] or after['instance_count'] != 3:
        raise RuntimeError('inherited stopped app/pier/history changed')
    report['base_stopped_app_preserved'] = True
    proof = {'traces': traces, 'original_target': target, 'native_before': native_before, 'native_stopped': stopped, 'terminal_stop': terminal, 'replay': replay, 'resumed': resumed, 'peer': {'container_id': peer_id, 'state': peer_process}, 'anchor': {'container_id': anchor, 'state': anchor_process}, 'kill_signals': kills, 'retained_before': before, 'retained_after': after}
    path = O/'proof.json'
    path.write_text(json.dumps(proof, indent=2)+'\n')
    report['proof_sha256'] = sha(path)
    report['passed'] = True


if __name__ == '__main__':
    try:
        main()
    except BaseException as error:
        report['error'] = type(error).__name__+': '+str(error)
    finally:
        if common is not None and common.helper is not None and common.helper.poll() is None:
            try:
                report['helper_cleanup_exit'] = common.stop_helper(signal.SIGTERM)
            except BaseException as error:
                report['helper_cleanup_error'] = type(error).__name__
        if O.is_dir():
            (O/'result.json').write_text(json.dumps(report, indent=2)+'\n')
        os.write(1, (status.MARKER+status.serialize(report)+'\n').encode('ascii'))
        try:
            guard()
            subprocess.run(['fstrim', '/'], timeout=60, check=False)
            subprocess.run(['systemctl', 'poweroff'], timeout=30, check=True)
        except BaseException:
            pass
