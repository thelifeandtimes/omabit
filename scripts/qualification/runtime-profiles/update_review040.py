#!/usr/bin/env python3
"""Owner-operated, one-time review040 DEV update. No live changes without --apply.

This is a private delivery adapter, not a release installer. A failed apply keeps
its checkpoint and refuses a second attempt; it never downgrades migrated state.
"""
import argparse
import fcntl
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import shutil
import sqlite3
import subprocess
import tarfile
import time

REPO = Path(__file__).resolve().parents[3]
DEV = Path('/srv/dev/urbit-040')
PLUGIN_ID = 'thelifeandtimes.omabit.review040'
OLD = Path.home() / '.config/omarchy/plugins/thelifeandtimes.omabit.review040-v050'
NEW = OLD.with_name('thelifeandtimes.omabit.review040-v060rc3')
CONFIG = Path.home() / '.config/omarchy/shell.json'
UNIT = 'omabit-dev-040-runner.service'
ENGINE = 'omabit-dev-040-docker.service'
RUNNER = 'runner-a9a208a18445179f05df5644ed6d245c'
SOURCE = '747dd810c10b74b4fbcdc10f617e6b15dd545a45'
VERSION = '0.6.0-rc.3'
SOCKET = '/run/user/1000/omabit-dev-040/urbit-runner/runner.sock'
DOCKER = '/run/user/1000/omabit-dev-040/urbit-docker.sock'
MANAGED = DEV / 'state/.local/share/urbit-runner/managed'
BUNDLE = 'omabit-0.6.0-rc.3-linux-amd64.tar.gz'
ARCHIVE = 'vere-4.6-jsonfix-0774857c.tar.gz'
PINS = {BUNDLE: 'f1676a51bede3302327917c9e80b8661bf482e47a8e40633903e60ac435037f7',
        ARCHIVE: 'c8a412c42d53fa0c9a8c97aa55a580ddf2864bbdf32f80074dfb6bdfd837f510'}
TOOLS = {'omabit': '355f897a80db93a7e26dd25e0c1e688ef55d4f077e83862d2327ebb633afb6a3',
         'urbit-collector': '0168cf02b79d247acb834704ea5dfeb7c25a94fd243ac76bfc2b1eac07bf5c58',
         'urbit-runner': 'bfd62c7be2d3a4ce513ecb425ea16129df2eb1218d99f5474a2d41072d386ee1'}
OLD_TOOLS = {'omabit': 'c7ec141df5cc27b2e53f01c2cceca838336bc66be7b232454cce7d4e49d21a1d', 'urbit-collector': '81ea1990acafb847e14a5093fabc67dc15fed3370ad4f5bf9b9043c3fdcf5be5', 'urbit-runner': '5a995481fae3e0642c2eb254e37bd439abc2dd02a40e1d2a7509db3dabd094b4'}


def require(value, message):
    if not value:
        raise RuntimeError(message)


def sha(path):
    with Path(path).open('rb') as f:
        return hashlib.file_digest(f, 'sha256').hexdigest()


def run(args, timeout=30):
    env = {k: v for k, v in os.environ.items() if k not in ('DOCKER_HOST', 'DOCKER_CONTEXT')}
    return subprocess.check_output(list(map(str, args)), text=True, env=env, timeout=timeout)


def write(path, value):
    with path.open('x') as f:
        json.dump(value, f, indent=2)
        f.write('\n')
        f.flush()
        os.fsync(f.fileno())
    path.chmod(0o600)


def tree(path):
    require(path.is_dir() and path.resolve() == path.absolute(), 'missing/aliased directory: ' + str(path))
    result = {}
    for p in sorted(path.rglob('*')):
        require(not p.is_symlink() and (p.is_file() or p.is_dir()), 'unexpected file type: ' + str(p))
        if p.is_file():
            result[str(p.relative_to(path))] = {'sha256': sha(p), 'bytes': p.stat().st_size, 'mode': p.stat().st_mode & 0o777}
    return result


def properties(unit):
    return dict(line.split('=', 1) for line in run(['systemctl', '--user', 'show', unit,
                '-p', 'ActiveState', '-p', 'MainPID', '-p', 'InvocationID', '-p', 'ExecStart', '-p', 'FragmentPath']).splitlines())


def hello(expected_source, schema):
    h = json.loads(run([DEV / 'bin/urbit-collector', '--socket', SOCKET, 'hello']))
    require(h['runner_id'] == RUNNER and h['mode'] == 'development-managed', 'unexpected DEV runner identity')
    require(h['state_schema'] == schema and h['build']['source_revision'] == expected_source and not h['build']['modified'], 'unexpected DEV runner build/schema')
    return h


def projection(snapshot):
    require(snapshot['runner_id'] == RUNNER, 'snapshot runner identity differs')
    require(not any(i.get('active_operation_id') for i in snapshot.get('instances', [])), 'a ship operation is active')
    require(not any(o.get('status') in ('queued', 'pending', 'running') for o in snapshot.get('operations', [])), 'an operation is active')
    keys = ('id', 'ship', 'mode', 'label', 'config', 'config_revision', 'applied_revision', 'desired_state', 'pier_path', 'bootstrap', 'recovery_required')
    return {'instances': [{k: i.get(k) for k in keys} for i in snapshot.get('instances', [])], 'operations': snapshot.get('operations', [])}


def snapshot():
    return projection(json.loads(run([DEV / 'bin/urbit-collector', '--socket', SOCKET, 'snapshot'], 60)))


def engine_state():
    ids = run(['/usr/bin/docker', '--host', 'unix://' + DOCKER, 'ps', '-aq', '--no-trunc']).splitlines()
    containers = {}
    for cid in ids:
        v = json.loads(run(['/usr/bin/docker', '--host', 'unix://' + DOCKER, 'inspect', cid]))[0]
        containers[cid] = {k: v[k] for k in ('Id', 'Image', 'RestartCount', 'Mounts')}
        containers[cid]['State'] = {k: v['State'][k] for k in ('Running', 'Paused', 'Restarting', 'Pid', 'ExitCode', 'StartedAt', 'FinishedAt')}
    return {'service': properties(ENGINE), 'containers': containers}


def budget(additional=1024**3):
    spec = importlib.util.spec_from_file_location('review_budget', Path(__file__).with_name('run.py'))
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module.full_charge(additional, 48 * 1024**3)['charged_bytes']


def copy_bytes(path):
    """Conservative destination allocation, including sparse files and directories."""
    require(path.resolve() == path.absolute(), 'aliased backup input: ' + str(path))
    entries = [path, *path.rglob('*')] if path.is_dir() else [path]
    total = 0
    for item in entries:
        require(not item.is_symlink() and (item.is_file() or item.is_dir()), 'unexpected backup input: ' + str(item))
        total += 4096 if item.is_dir() else ((item.stat().st_size + 4095) // 4096) * 4096
    return total


def storage_plan(work):
    # Retain every management file, including existing boot pills. An arbitrary
    # per-directory threshold is not a substitute for the aggregate budget.
    sqlite_bytes = sum(copy_bytes(MANAGED / name) for name in
                       ('runner.sqlite', 'runner.sqlite-wal', 'runner.sqlite-shm')
                       if (MANAGED / name).exists())
    archive = work / ARCHIVE
    with tarfile.open(archive) as tar:
        payload = sum(member.size for member in tar if member.isfile())
    parts = {
        'management_backup': copy_bytes(MANAGED),
        'sqlite_analysis_copy': sqlite_bytes,
        'desktop_backups': sum(copy_bytes(DEV / 'desktop' / name) for name in ('config', 'data', 'state')),
        'old_binaries_backup': sum(copy_bytes(DEV / 'bin' / name) for name in TOOLS),
        'candidate_binaries_staging': sum(copy_bytes(work / 'plugin/bin' / name) for name in TOOLS),
        # Reserve for the retained runtime archive, Docker import staging, blobs
        # and unpacked layers. Existing engine layers are not assumed reusable.
        'runtime_archive_and_image_reserve': copy_bytes(archive) + 4 * payload,
        'growth_and_metadata_reserve': 128 * 1024**2,
    }
    required = max(1024**3, ((sum(parts.values()) + 1024**2 - 1) // 1024**2) * 1024**2)
    return {'breakdown': parts, 'required_scratch_bytes': required, 'aggregate_cap_bytes': 48 * 1024**3}


def preflight():
    require(os.getuid() == 1000 and Path.home() == Path('/home/thelifeandtimes'), 'run as the existing DEV owner, without sudo')
    owner = json.loads((DEV / 'owner.json').read_text())
    require(owner == {'version': 1, 'name': 'thelifeandtimes', 'uid': 1000, 'gid': 1000, 'home': str(Path.home()), 'piers': str(DEV / 'piers')}, 'DEV owner binding differs')
    require(DEV.resolve() == DEV and not NEW.exists(), 'DEV path differs or candidate already exists')
    manifest = json.loads((OLD / 'manifest.json').read_text())
    require(manifest['id'] == PLUGIN_ID and manifest['version'] == '0.5.0-dev.review-040', 'unexpected installed DEV widget')
    for name, digest in OLD_TOOLS.items():
        require(sha(DEV / 'bin' / name) == sha(OLD / 'bin' / name) == digest, 'unexpected old helper: ' + name)
    unit = properties(UNIT)
    require(unit['ActiveState'] == 'active' and properties(ENGINE)['ActiveState'] == 'active', 'DEV manager and engine must already be active')
    require(Path(unit['FragmentPath']).resolve() == Path.home() / '.config/systemd/user' / UNIT, 'DEV unit source differs')
    expected = str(DEV / 'bin/urbit-runner') + ' --development-root ' + str(DEV) + ' serve-development --acknowledge-development ;'
    require(expected in unit['ExecStart'] and '--resume-running' not in unit['ExecStart'] and '--record-runtime' not in unit['ExecStart'], 'DEV service arguments differ')
    h = hello('59d83aab49384d2f26531320bb5ab68e1257aeb5', 65)
    status = json.loads(run(['omarchy', 'shell', PLUGIN_ID + '.review', 'status']))
    require(status['version'] == '0.5.0' and status['profile'] == 'review-040' and status['loaded'] and not status['local_setup'] and not status['busy'], 'close pending DEV UI work before updating')
    return {'runner_id': h['runner_id'], 'unit_sha256': sha(unit['FragmentPath']), 'owner_sha256': sha(DEV / 'owner.json')}


def stage_plugin(archive, installed, target):
    target.mkdir(mode=0o700)
    prefix = 'omabit-' + VERSION + '-linux-amd64'
    with tarfile.open(archive) as tar:
        for member in tar:
            p = Path(member.name)
            require(not p.is_absolute() and '..' not in p.parts and p.parts[0] == prefix, 'unsafe archive path')
            require(member.isdir() or member.isfile(), 'special archive member')
            if not member.isfile():
                continue
            relative = Path(*p.parts[1:])
            dest = None
            if relative.is_relative_to('plugins/omabit'):
                dest = target / 'app060rc3' / relative.relative_to('plugins/omabit')
            elif relative.is_relative_to('skills') or str(relative) in ('BUILD.json', 'SKILLS.json', 'SKILLSET.json'):
                dest = target / relative
            elif relative.parts[0] == 'build' and relative.name in TOOLS:
                dest = target / 'bin' / relative.name
            if dest:
                dest.parent.mkdir(parents=True, exist_ok=True)
                with dest.open('xb') as out:
                    shutil.copyfileobj(tar.extractfile(member), out)
                dest.chmod(0o755 if member.mode & 0o111 else 0o600)
    build = json.loads((target / 'BUILD.json').read_text())
    require(build['source_commit'] == SOURCE and build['version'] == VERSION and build['schema'] == 66, 'bundle metadata differs')
    for name, digest in TOOLS.items():
        require(sha(target / 'bin' / name) == digest, 'candidate helper differs: ' + name)
    skills = json.loads((target / 'SKILLS.json').read_text())
    for name, digest in skills['files'].items():
        require(sha(target / name) == digest, 'skill digest differs')
    manifest = json.loads((installed / 'manifest.json').read_text())
    manifest.update(name='Omabit 0.6.0-rc.3 DEV', version='0.6.0-rc.3-dev.review-040', description='Preserved review040 DEV environment with the rc.3 runtime catalogue.')
    manifest['entryPoints'] = {'service': 'ReviewService060.qml', 'barWidget': 'ReviewPanel060.qml'}
    manifest['barWidget']['displayName'] = manifest['name']
    (target / 'manifest.json').write_text(json.dumps(manifest, indent=2) + '\n')
    for old, new in [('ReviewService050.qml', 'ReviewService060.qml'), ('ReviewPanel050.qml', 'ReviewPanel060.qml')]:
        text = (installed / old).read_text().replace('"app050"', '"app060rc3"').replace('Omabit 0.5.0 DEV', 'Omabit 0.6.0-rc.3 DEV')
        (target / new).write_text(text)
    panel = target / 'app060rc3/Panel.qml'
    text = panel.read_text()
    start, end = text.index('  Ui.BarIconButton {'), text.index('  Ui.KeyboardPanel {')
    old = (installed / 'app050/Panel.qml').read_text()
    button = old[old.index('  Ui.WidgetButton {'):old.index('  Ui.KeyboardPanel {')]
    panel.write_text(text[:start] + button.replace('Omabit 0.5.0 DEV', 'Omabit 0.6.0-rc.3 DEV') + text[end:])


def prepare(work, assets):
    require(not work.exists(), 'work directory exists; retain it and inspect its plan/receipt')
    binding = preflight()
    charged = budget()
    for name, digest in PINS.items():
        require(sha(assets / name) == digest, 'release asset differs: ' + name)
    work.mkdir(parents=True, mode=0o700)
    stage_plugin(assets / BUNDLE, OLD, work / 'plugin')
    shutil.copy2(assets / ARCHIVE, work / ARCHIVE)
    (work / ARCHIVE).chmod(0o600)
    capacity = storage_plan(work)
    charged = budget(capacity['required_scratch_bytes'])
    plan = dict(binding, storage=capacity, phase='prepared-only', version=VERSION, source=SOURCE, work=str(work),
                old_plugin=str(OLD), new_plugin=str(NEW), plugin_before=tree(OLD), plugin_candidate=tree(work / 'plugin'),
                charged_bytes=charged, script_sha256=sha(__file__), backup=str(DEV / 'state/upgrade-060rc3-owner'))
    write(work / 'plan.json', plan)
    print(json.dumps({k: v for k, v in plan.items() if k not in ('plugin_before', 'plugin_candidate')}, indent=2))


def wait_for(fn, seconds=60):
    end = time.monotonic() + seconds
    while True:
        try:
            return fn()
        except (RuntimeError, subprocess.SubprocessError, ValueError, KeyError):
            if time.monotonic() >= end:
                raise
            time.sleep(.5)


def restore_config(original, expected):
    require(CONFIG.read_bytes() == expected, 'shell configuration changed during update; retained backup requires review')
    temp = CONFIG.with_name('.omabit-review040-restore-' + str(os.getpid()))
    with temp.open('xb') as f:
        f.write(original)
        f.flush()
        os.fsync(f.fileno())
    temp.chmod(CONFIG.stat().st_mode & 0o777)
    require(CONFIG.read_bytes() == expected, 'shell configuration changed before restoration')
    os.replace(temp, CONFIG)


def validate_plan(work, require_script=True):
    plan = json.loads((work / 'plan.json').read_text())
    require(plan['phase'] == 'prepared-only', 'plan phase differs')
    if require_script:
        require(plan['script_sha256'] == sha(__file__), 'prepared updater differs')
    require(not (work / 'apply-started.json').exists(), 'apply already started; inspect retained receipt, do not rerun or downgrade')
    binding = preflight()
    require(all(binding[k] == plan[k] for k in binding), 'DEV owner/service binding changed')
    require(tree(OLD) == plan['plugin_before'] and tree(work / 'plugin') == plan['plugin_candidate'], 'plugin inputs changed')
    require(sha(work / ARCHIVE) == PINS[ARCHIVE], 'reviewed archive changed')
    require(work.stat().st_dev == OLD.stat().st_dev, 'plugin staging must share the installed filesystem')
    require(plan['work'] == str(work) and plan['version'] == VERSION and plan['source'] == SOURCE, 'plan identity differs')
    backup = DEV / 'state/upgrade-060rc3-owner'
    require(plan['backup'] == str(backup), 'backup target differs')
    require(not backup.exists(), 'backup already exists; do not reuse an earlier attempt')
    capacity = storage_plan(work)
    charged = budget(capacity['required_scratch_bytes'])
    return plan, backup, capacity, charged


def check(work):
    _, _, capacity, charged = validate_plan(work)
    print(json.dumps({'status': 'ready', 'live_changes': False, 'charged_bytes': charged,
                      'total_with_scratch': charged + capacity['required_scratch_bytes'], 'storage': capacity}, indent=2))


def refresh_plan(work):
    # Explicitly renew only a still-unstarted preparation. Keep the original
    # plan and refuse any attempt marker, backup or changed staged/installed bytes.
    plan, _, capacity, charged = validate_plan(work, require_script=False)
    original = (work / 'plan.json').read_bytes()
    saved = work / ('plan-before-' + hashlib.sha256(original).hexdigest() + '.json')
    with saved.open('xb') as f:
        f.write(original)
        f.flush()
        os.fsync(f.fileno())
    updated = dict(plan, script_sha256=sha(__file__), storage=capacity, charged_bytes=charged,
                   supersedes_plan=saved.name)
    temporary = work / ('plan-refresh-' + str(os.getpid()) + '.json')
    write(temporary, updated)
    require((work / 'plan.json').read_bytes() == original, 'plan changed during refresh')
    os.replace(temporary, work / 'plan.json')
    check(work)


def apply(work):
    _, backup, capacity, charged = validate_plan(work)
    before = snapshot()
    runtime = engine_state()
    original_config = CONFIG.read_bytes()
    backup.mkdir(mode=0o700)
    for label in ('bin-before', 'bin-candidate'):
        (backup / label).mkdir(mode=0o700)
        for name in TOOLS:
            shutil.copy2((DEV / 'bin' if label == 'bin-before' else work / 'plugin/bin') / name, backup / label / name)
    write(work / 'apply-started.json', {'backup': str(backup), 'before': before, 'runtime': runtime, 'storage': capacity, 'charged_bytes': charged})
    (backup / 'shell-before.json').write_bytes(original_config)
    phase = 'disable-dev-widget'
    try:
        require(run(['omarchy', 'shell', 'shell', 'setPluginEnabled', PLUGIN_ID, 'false']).strip() == 'ok', 'DEV widget disable refused')
        def unloaded():
            p = subprocess.run(['omarchy', 'shell', PLUGIN_ID + '.review', 'status'], capture_output=True, timeout=10)
            require(p.returncode != 0, 'DEV widget still loaded')
        wait_for(unloaded)
        disabled_config = CONFIG.read_bytes()
        (backup / 'shell-disabled.json').write_bytes(disabled_config)
        for name in ('config', 'data', 'state'):
            shutil.copytree(DEV / 'desktop' / name, backup / ('desktop-' + name))
        require(snapshot() == before, 'DEV fleet changed while closing the widget')
        phase = 'stop-dev-manager'
        run(['systemctl', '--user', 'stop', UNIT], 390)
        require(properties(UNIT)['ActiveState'] == 'inactive', 'DEV manager did not stop')
        require(engine_state() == runtime, 'DEV runtime changed before update')
        tree(MANAGED)  # Refuse links/special files before copying management state.
        shutil.copytree(MANAGED, backup / 'managed-before')
        require(tree(MANAGED) == tree(backup / 'managed-before'), 'management backup differs')
        # SQLite can create WAL/SHM sidecars even for mode=ro. Inspect a separate
        # copy so the original offline management backup remains byte-for-byte.
        analysis = backup / 'analysis-before'
        analysis.mkdir(mode=0o700)
        for name in ('runner.sqlite', 'runner.sqlite-wal', 'runner.sqlite-shm'):
            src = backup / 'managed-before' / name
            if src.exists():
                shutil.copy2(src, analysis / name)
        db = sqlite3.connect('file:' + str(analysis / 'runner.sqlite') + '?mode=ro', uri=True)
        try:
            require(db.execute('pragma user_version').fetchone()[0] == 65, 'backup schema differs')
            require(db.execute('pragma integrity_check').fetchone()[0] == 'ok', 'backup integrity failed')
        finally:
            db.close()
        write(backup / 'stopped.json', {'management_backup': tree(backup / 'managed-before'), 'runtime_preserved': True})
        phase = 'install-matching-dev-tools'
        for name, digest in TOOLS.items():
            require(sha(DEV / 'bin' / name) == OLD_TOOLS[name] and sha(backup / 'bin-candidate' / name) == digest, 'binary changed during update')
            os.replace(backup / 'bin-candidate' / name, DEV / 'bin' / name)
        run([DEV / 'bin/urbit-runner', '--development-root', DEV, 'runtime-profile-install', '--profile', 'vere-4.6-jsonfix-0774857c', '--archive', work / ARCHIVE], 180)
        phase = 'start-dev-manager'
        run(['systemctl', '--user', 'start', UNIT], 60)
        wait_for(lambda: hello(SOURCE, 66))
        require(snapshot() == before, 'original fleet configuration/intent/history differs')
        require(engine_state() == runtime, 'existing DEV containers or engine changed')
        phase = 'activate-versioned-dev-widget'
        require(not NEW.exists() and not (work / 'plugin-before').exists(), 'plugin destination changed')
        OLD.rename(work / 'plugin-before')
        (work / 'plugin').rename(NEW)
        run(['omarchy', 'shell', 'shell', 'rescanPlugins'])
        restore_config(original_config, disabled_config)
        run(['omarchy', 'shell', 'shell', 'reloadConfig'])
        run(['omarchy', 'shell', 'shell', 'rescanPlugins'])
        def loaded():
            s = json.loads(run(['omarchy', 'shell', PLUGIN_ID + '.review', 'status']))
            require(s['loaded'] and s['version'] == VERSION and s['profile'] == 'review-040' and not s['local_setup'], 'DEV widget not ready')
            require(s['collector'] == str(NEW / 'bin/urbit-collector') and s['omabit'] == str(NEW / 'bin/omabit'), 'DEV helper paths differ')
            require(any(r['id'] == RUNNER for r in s['runners']), 'existing DEV enrollment not loaded')
            return {k: s[k] for k in ('version', 'profile', 'collector', 'omabit', 'loaded', 'local_setup')}
        status = wait_for(loaded)
        require(CONFIG.read_bytes() == original_config, 'original shell configuration changed')
        write(work / 'complete.json', {'phase': 'complete', 'loaded': status, 'backup': str(backup), 'old_plugin': str(work / 'plugin-before'), 'engine_and_containers_preserved': True, 'original_intent_config_operations_preserved': True})
        print(json.dumps({'status': 'complete', 'version': VERSION, 'backup': str(backup), 'receipt': str(work / 'complete.json')}, indent=2))
    except BaseException as error:
        write(work / 'failed.json', {'phase': phase, 'error': str(error), 'backup': str(backup), 'action': 'Stop and inspect this checkpoint. Do not rerun, reset state, or restore older binaries after migration.'})
        raise


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--prepare', action='store_true')
    p.add_argument('--apply', action='store_true')
    p.add_argument('--check', action='store_true', help='verify the prepared plan and current storage without live changes')
    p.add_argument('--refresh-plan', action='store_true', help='retain and renew an unstarted preparation after adapter corrections')
    p.add_argument('--assets', type=Path, default=Path('/tmp/omabit-runtime-profiles-review-assets/0.6.0-rc.3'))
    p.add_argument('--work', type=Path, default=REPO / 'build/review040-update-060rc3-owner')
    args = p.parse_args()
    if sum((args.prepare, args.apply, args.check, args.refresh_plan)) != 1:
        p.error('choose --prepare, --check, --refresh-plan, or --apply')
    os.umask(0o077)
    lock = os.open('/tmp/omabit-issues-vm-runtime.lock', os.O_RDONLY | os.O_NOFOLLOW)
    with os.fdopen(lock, 'r') as f:
        fcntl.flock(f, fcntl.LOCK_EX | fcntl.LOCK_NB)
        work = args.work.absolute()
        if args.prepare:
            prepare(work, args.assets.absolute())
        elif args.check:
            check(work)
        elif args.refresh_plan:
            refresh_plan(work)
        else:
            apply(work)


if __name__ == '__main__':
    try:
        main()
    except Exception as error:
        raise SystemExit(str(error))
