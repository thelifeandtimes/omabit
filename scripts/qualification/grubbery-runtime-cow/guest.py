#!/usr/bin/python3
"""Fixed read-only-backed pending app fixture, supported lifecycle and audit APIs."""
import gzip
import contextlib
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import sqlite3
import subprocess
import sys
import time

I = Path('/opt/omabit-release-input')
R = Path('/run/omabit-runtime-candidate')
O = Path('/opt/omabit-issues-runtime-results')
REVIEW = Path('/home/review')
PIN = 'a23f10ddc89657489d8e55eaf5e4362603848366a447dbf9f51ce192837a08f9'
CASE = 'cow-recover-pending-zod-v1'
HELPER_PROCESS_TIMEOUT_SECONDS = 5700
HELPER_DEADLINE_SECONDS = 5750
report = {'version': 1, 'passed': False, 'scope': 'actual source service Docker/curl and CLI/stdio on a fresh COW copy of the fixed pending app fixture; not installed upgrade/DEV/production qualification'}


def sha(path):
    with Path(path).open('rb') as stream:
        return hashlib.file_digest(stream, 'sha256').hexdigest()


def guard():
    if os.geteuid() != 0 or sys.argv[1:]:
        raise RuntimeError('root in fixed offline guest only')
    if Path('/sys/class/dmi/id/sys_vendor').read_text().strip() != 'QEMU' or 'omabit.disposable-guest=release-upgrade-v1' not in Path('/proc/cmdline').read_text().split():
        raise RuntimeError('not isolated guest')
    scenario = json.loads((I/'scenario.json').read_text())
    if scenario.get('kind') != 'issues18-product-grubbery-runtime-v1' or scenario.get('case') != CASE or scenario.get('predecessor_root_sha256') != PIN or scenario.get('original_schema') != 62 or scenario.get('target_schema') != 65 or scenario.get('development_snapshot') is not True:
        raise RuntimeError('wrong source fixture')
    return scenario


def original_rows(schema):
    # Read-only diagnostic, never state repair/admission. Only the cloned fixed
    # task store is present; no host state/pier/socket is mounted in this guest.
    with contextlib.closing(sqlite3.connect('file:/home/review/.local/share/urbit-runner/managed/runner.sqlite?mode=ro', uri=True)) as db:
        db.execute('BEGIN')
        if db.execute('pragma user_version').fetchone()[0] != schema:
            raise RuntimeError('unexpected task store schema')
        return {name: list(db.execute('select * from '+name)) for name in ('grubbery_app_operations', 'grubbery_app_events', 'grubbery_app_leases')}


def initialized_fixture():
    with contextlib.closing(sqlite3.connect('file:/home/review/.local/share/urbit-runner/managed/runner.sqlite?mode=ro', uri=True)) as db:
        rows = list(db.execute('select body from instances'))
    if len(rows) != 1:
        raise RuntimeError('one fixed disposable instance required')
    state = json.loads(rows[0][0])
    public, runtime = state['public'], state['runtime']
    if public['id'] != 'grubbery-zod' or public['ship'] != '~zod' or runtime.get('prepared') is not True or runtime.get('initialized') is not True or runtime.get('maintenance_uncertain') or runtime.get('fake', {}).get('uncertain'):
        raise RuntimeError('exact initialized disposable pier required; no bootstrap permitted')
    return hashlib.sha256(json.dumps(runtime['storage'], sort_keys=True).encode()).hexdigest()


def inflate(scenario):
    if R.exists() or R.is_symlink():
        raise RuntimeError('fresh memory-backed executable directory required')
    R.mkdir(mode=0o755)
    (R/'candidate').mkdir(mode=0o755)
    for name, pin in scenario['candidate'].items():
        relative = ('candidate/'+name) if name != 'managed.test' else name
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


def main():
    scenario = guard()
    if O.exists():
        raise RuntimeError('fresh derived results required')
    O.mkdir(mode=0o700)
    before = original_rows(62)

    if len(before['grubbery_app_operations']) != 1 or len(before['grubbery_app_events']) != 17 or len(before['grubbery_app_leases']) != 1:
        raise RuntimeError('wrong original pending journal')
    report['original_rows_sha256'] = hashlib.sha256(repr(before).encode()).hexdigest()
    report['initialized_pier_storage_sha256'] = initialized_fixture()
    inflate(scenario)
    spec = importlib.util.spec_from_file_location('common', I/'issues-common.py')
    common = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(common)
    common.I, common.O, common.report, common.scenario = R, O, report, scenario
    report.update(inputs=scenario, source_commit=scenario['source_commit'])
    common.run(['systemctl', 'start', 'user@1000.service'])
    properties = common.run(['systemctl', '--user', 'show', 'urbit-runner.service', '-p', 'ActiveState,ExecStart'], True).stdout
    if 'ActiveState=active' in properties or '--resume-running' in properties or str(R/'candidate/urbit-runner') not in properties:
        raise RuntimeError('source service could auto-resume or select old executable')
    Path('/run/omabit-runtime-runner-ready').touch(mode=0o600)
    common.run(['systemctl', '--user', 'start', 'urbit-runner.service'], True)
    hello = common.wait(common.hello, 60)
    if hello['build']['sha256'] != scenario['candidate']['urbit-runner']['sha256']:
        raise RuntimeError('wrong actual service executable')
    report['runner_id'] = hello['runner_id']
    if original_rows(65) != before:
        raise RuntimeError('opening/migration rewrote original app uncertainty')
    report['migration_original_app_rows_preserved'] = True
    original_cid = '5ff5f1b73b3baeef0203297e8747202573ca0577918e48bd79b0332201e7b6fa'
    old = common.inspect(original_cid)
    if any(old['State'].get(key) is not False for key in ('Running', 'Paused', 'Restarting')) or old['HostConfig']['RestartPolicy']['Name'] != 'no':
        raise RuntimeError('fixed original disposable process is not stopped')
    helper = subprocess.Popen(common.owner_args(['env', 'OMABIT_APP_RUNTIME_VM='+CASE, R/'managed.test', '-test.run=^TestIssuesGrubberyRuntimeRecoveryDisposableVM$', '-test.v', '-test.timeout='+str(HELPER_PROCESS_TIMEOUT_SECONDS)+'s']))
    deadline = time.monotonic()+HELPER_DEADLINE_SECONDS
    try:
        while helper.poll() is None:
            if time.monotonic() > deadline:
                raise RuntimeError('pending recovery helper deadline')
            common.run(['fstrim', '/'])
            time.sleep(5)
    finally:
        if helper.poll() is None:
            helper.terminate()
            try:
                helper.wait(timeout=15)
            except subprocess.TimeoutExpired:
                helper.kill()
                helper.wait()
    report['helper_exit'] = helper.returncode
    after = original_rows(65)
    # Original header/body/stamp stay exact; its revision and lease may change
    # only through later supported original phase completion. New events append.
    original_header = before['grubbery_app_operations'][0]
    current_header = next(row for row in after['grubbery_app_operations'] if row[0] == original_header[0])
    if current_header[:-1] != original_header[:-1] or not all(row in after['grubbery_app_events'] for row in before['grubbery_app_events']):
        raise RuntimeError('original app header or17-event prefix changed')
    report['original_header_and_event_prefix_preserved'] = True
    if helper.returncode:
        raise RuntimeError('source recovery helper failed; exact pending evidence retained')
    proof = json.loads((REVIEW/'omabit-app-runtime-recovery/proof.json').read_text())
    required = ('audited_replacement', 'unknown_phase_observed_without_redispatch', 'original_source_completed', 'second_source_state_and_code', 'second_same_pier_replacement_state', 'manager_restart_audit_and_original_replay')
    if not all(proof.get('qualification', {}).get(key) is True for key in required):
        raise RuntimeError('not every declared native recovery gate passed')
    report['product'] = proof['qualification']
    report['proof_sha256'] = sha(REVIEW/'omabit-app-runtime-recovery/proof.json')
    common.operation('candidate', 'grubbery-zod', 'stop', 'runtime-recovery-final-clean-stop')
    report['final_clean_stop'] = True
    report['passed'] = True


if __name__ == '__main__':
    try:
        main()
    except BaseException as error:
        report['error'] = type(error).__name__+': '+str(error)
    finally:
        if O.is_dir():
            try:
                (O/'result.json').write_text(json.dumps(report, indent=2)+'\n')
            except OSError as error:
                report['result_persistence_error'] = type(error).__name__
        print('OMABIT_GRUBBERY_VM_RESULT '+json.dumps(report, separators=(',', ':')), flush=True)
        try:
            guard()
            subprocess.run(['fstrim', '/'], timeout=60, check=False)
            subprocess.run(['systemctl', 'poweroff'], check=True)
        except BaseException:
            pass
