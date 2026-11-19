#!/usr/bin/python3
"""Observe exact copied closed native recordings; no launch/control writes."""
import contextlib
import copy
import datetime
import gzip
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import sqlite3
import subprocess
import sys

I = Path('/opt/omabit-release-input')
R = Path('/run/omabit-runtime-candidate')
O = Path('/opt/omabit-issues-usage-results')
CASE = 'closed-native-capture-allocation-v1'
PIN = 'e3a728a4b1dcc090a10560481dd8414e801c4ef12632164fb05e0f4db5d022d9'
HELPER_SECONDS = 150
HELPER_PROCESS_SECONDS = 180
report = {'format': 1, 'passed': False, 'scope': 'closed native recording allocation on one fresh offline COW overlay; no runtime launch/control write, installed upgrade, DEV or production qualification'}


def sha(path):
    with Path(path).open('rb') as stream:
        return hashlib.file_digest(stream, 'sha256').hexdigest()


def guard():
    if os.geteuid() != 0 or sys.argv[1:]:
        raise RuntimeError('root in fixed offline guest only')
    if Path('/sys/class/dmi/id/sys_vendor').read_text().strip() != 'QEMU' or 'omabit.disposable-guest=release-upgrade-v1' not in Path('/proc/cmdline').read_text().split():
        raise RuntimeError('not isolated guest')
    value = json.loads((I/'scenario.json').read_text())
    if value.get('kind') != 'issues20-recording-usage-v1' or value.get('case') != CASE or value.get('predecessor_root_sha256') != PIN or value.get('original_schema') != 65 or value.get('target_schema') != 65 or value.get('development_snapshot') is not True:
        raise RuntimeError('wrong exact closed-recording fixture')
    return value


def saved_rows():
    # Full logical rows from the known task copy only. Explicitly close this
    # independent reader; no repair, marker changes or WAL removal is allowed.
    path = 'file:/home/review/.local/share/urbit-runner/managed/runner.sqlite?mode=ro'
    with contextlib.closing(sqlite3.connect(path, uri=True)) as db:
        db.execute('BEGIN')
        if db.execute('pragma user_version').fetchone()[0] != 65:
            raise RuntimeError('wrong original/candidate schema')
        names = [row[0] for row in db.execute("select name from sqlite_master where type='table' order by name")]
        if len(names) > 256:
            raise RuntimeError('bounded known schema required')
        rows = {}
        for name in names:
            quoted = '"'+name.replace('"', '""')+'"'
            rows[name] = sorted(db.execute('select * from '+quoted).fetchall(), key=repr)
        instances = list(db.execute('select body from instances'))
        if len(instances) != 1:
            raise RuntimeError('one exact original disposable recording instance required')
        state = json.loads(instances[0][0])
        if state['public']['id'] != 'recording-zod' or state['public']['ship'] != '~zod' or state['public']['desired_state'] != 'stopped':
            raise RuntimeError('fixed stopped original intent required; never auto-resume')
        return rows


def verify_preserved_rows(before, after):
    # A fresh manager reobserves the one stopped instance once. This is current
    # observation evidence, not a rewrite of immutable capture/control history.
    # Permit exactly that timestamp and its single revision increment. Every
    # other instance value, row/cell and table remains exact.
    original, current = copy.deepcopy(before), copy.deepcopy(after)
    if set(original) != set(current) or len(original.get('instances', [])) != 1 or len(current.get('instances', [])) != 1:
        raise RuntimeError('original logical inventory changed')
    old_row, new_row = original['instances'][0], current['instances'][0]
    if len(old_row) != 2 or len(new_row) != 2 or old_row[0] != new_row[0]:
        raise RuntimeError('original instance binding changed')
    old, new = json.loads(old_row[1]), json.loads(new_row[1])
    old_time, new_time = old['public'].pop('observed_at'), new['public'].pop('observed_at')
    old_revision, new_revision = old.pop('runtime_revision'), new.pop('runtime_revision')
    if type(old_revision) is not int or type(new_revision) is not int or old_revision < 1 or new_revision != old_revision+1:
        raise RuntimeError('not exactly one fresh instance observation')
    if not isinstance(old_time, str) or not isinstance(new_time, str) or not old_time.endswith('Z') or not new_time.endswith('Z') or datetime.datetime.fromisoformat(new_time[:-1]+'+00:00') <= datetime.datetime.fromisoformat(old_time[:-1]+'+00:00'):
        raise RuntimeError('fresh observation timestamp not established')
    if old != new:
        raise RuntimeError('original instance intent/configuration/runtime values changed')
    del original['instances'], current['instances']
    if original != current:
        raise RuntimeError('saved original history/table rows changed')
    return {'fields': ['public.observed_at', 'runtime_revision'], 'before_revision': old_revision, 'after_revision': new_revision, 'fresh_observation_only': True}


def inflate(scenario):
    if R.exists() or R.is_symlink():
        raise RuntimeError('fresh tmpfs executables required')
    R.mkdir(mode=0o755)
    (R/'candidate').mkdir(mode=0o755)
    for name, pin in scenario['candidate'].items():
        relative = 'candidate/'+name if name != 'managed.test' else name
        source, target = I/(relative+'.gz'), R/relative
        if source.is_symlink() or not source.is_file() or not 0 < pin['bytes'] < 64*1024**2:
            raise RuntimeError('bounded ordinary compressed executable required')
        with gzip.open(source, 'rb') as stream, target.open('xb') as output:
            left = pin['bytes']
            while left:
                data = stream.read(min(1 << 20, left))
                if not data:
                    raise RuntimeError('truncated executable')
                output.write(data)
                left -= len(data)
            if stream.read(1):
                raise RuntimeError('executable exceeds declared bound')
        target.chmod(0o755)
        if sha(target) != pin['sha256']:
            raise RuntimeError('inflated executable differs')


def main():
    scenario = guard()
    if O.exists():
        raise RuntimeError('fresh derived results required')
    O.mkdir(mode=0o700)
    before = saved_rows()
    report.update(source_commit=scenario['source_commit'], inputs=scenario, original_rows_sha256=hashlib.sha256(repr(before).encode()).hexdigest())
    inflate(scenario)
    spec = importlib.util.spec_from_file_location('common', I/'issues-common.py')
    common = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(common)
    common.I, common.O, common.report, common.scenario = R, O, report, scenario
    common.run(['systemctl', 'start', 'user@1000.service'])
    properties = common.run(['systemctl', '--user', 'show', 'urbit-runner.service', '-p', 'ActiveState,ExecStart'], True).stdout
    if 'ActiveState=active' in properties or '--resume-running' in properties or str(R/'candidate/urbit-runner') not in properties or '--record-runtime' not in properties:
        raise RuntimeError('wrong source service or implicit resume')
    Path('/run/omabit-runtime-runner-ready').touch(mode=0o600)
    common.run(['systemctl', '--user', 'start', 'urbit-runner.service'], True)
    hello = common.wait(common.hello, 30)
    if hello['build']['sha256'] != scenario['candidate']['urbit-runner']['sha256']:
        raise RuntimeError('wrong actual service executable')
    report['runner_id'] = hello['runner_id']
    helper = common.run(['env', 'OMABIT_RECORDING_USAGE_VM='+CASE, R/'managed.test', '-test.run=^TestIssuesRecordingUsageDisposableVM$', '-test.v', '-test.timeout='+str(HELPER_SECONDS)+'s'], True, timeout=HELPER_PROCESS_SECONDS)
    report['helper_exit'] = helper.returncode
    proof = Path('/home/review/omabit-recording-usage-proof/proof.json')
    value = json.loads(proof.read_text())
    if value.get('passed') is not True or not all(value.get(key) is True for key in ('original_recording_bytes_unchanged', 'policy_head_unchanged', 'actual_cli_and_stdio_measurements')):
        raise RuntimeError('not every native allocation/read gate passed')
    observation = verify_preserved_rows(before, saved_rows())
    report.update(product=value, proof_sha256=sha(proof), original_rows_preserved_except_one_observation=True, observation_refresh=observation, passed=True)


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
        print('OMABIT_RECORDING_USAGE_VM_RESULT '+json.dumps(report, separators=(',', ':')), flush=True)
        try:
            guard()
            subprocess.run(['fstrim', '/'], timeout=15, check=False)
            subprocess.run(['systemctl', 'poweroff'], check=True)
        except BaseException:
            pass
