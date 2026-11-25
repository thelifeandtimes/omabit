#!/usr/bin/env python3
"""Owner-run local UI overlay for RC4; never updates a runner or runtime.

This is a separately labelled DEV preview, not a replacement RC4 release.
Preparation/check are read-only with respect to the installed environment.
"""
import argparse
from contextlib import contextmanager
import fcntl
import importlib.util
import json
import os
from pathlib import Path
import shutil
import stat
import subprocess

spec = importlib.util.spec_from_file_location('rc4_adapter', Path(__file__).with_name('update_review040_rc4.py'))
base = importlib.util.module_from_spec(spec)
spec.loader.exec_module(base)
REPO = base.REPO
OLD = base.NEW
NEW = OLD.with_name('thelifeandtimes.omabit.review040-v060rc4-ui1')
WORK = REPO / 'build/review040-update-060rc4-ui1'
LOCK = Path('/tmp/omabit-issues-vm-runtime.lock')
VERSION = '0.6.0-rc.4+dev.ui.1'
FAILED_UPDATER_SHA = 'd6315fa46651840bf99a1a55dc1449882b8288ce95f354d688961adefe6322b2'
FILES = {
    'MonitorWindow.qml': ('ce8cf30dab1f8276d6f8ab64dd64bb7f348daf79e52eabc40a632681dc5ffdd5', '4ae1c94588b3eda67dcc0cd120c8ca871b2535c3502194dfd59bfc44155a4560'),
    'ShipPortsPane.qml': ('bc5a2b0ed54a233b042719cf64e5ac596c8a8ddebcc114fe3c27f54ba88a3a24', 'd2acb073d6c9fc2c194037d71ccd08e2ddea43b5df7aef1332145938c3c7c6d0'),
}
require = base.require


def ipc(target, method, *args):
    return base.run(['omarchy', 'shell', target, method, *args]).strip()


def status():
    return json.loads(ipc(base.PLUGIN_ID + '.review', 'status'))


def disabled_settings(original):
    expected = json.loads(original)
    # PluginRegistry.ensureConfigShape fills absent sections before removing a
    # widget. Older shell.json files commonly omit the newer tray section.
    bar = expected.setdefault('bar', {})
    layout = bar.setdefault('layout', {})
    for section in ('left', 'centerBegin', 'center', 'centerEnd', 'right', 'tray'):
        if not isinstance(layout.get(section), list):
            layout[section] = []
    if not isinstance(expected.get('plugins'), list):
        expected['plugins'] = []
    locations = [entries for entries in expected.get('bar', {}).get('layout', {}).values()
                 if isinstance(entries, list)] + [expected.get('plugins', [])]
    matches = [(entries, i) for entries in locations for i, entry in enumerate(entries)
               if isinstance(entry, dict) and entry.get('id') == base.PLUGIN_ID]
    require(len(matches) == 1, 'expected exactly one DEV widget placement')
    entries, index = matches[0]
    entries.pop(index)
    return expected


def ready(version, plugin):
    s = status()
    require(s['loaded'] and s['version'] == version and s['profile'] == 'review-040'
            and not s['local_setup'] and not s['busy'], 'close pending DEV UI work, then retry the check')
    for name in ('collector', 'omabit'):
        require(s[name] == str(plugin / 'bin' / ('urbit-collector' if name == 'collector' else name)), 'unexpected DEV helper path')
    return {k: s[k] for k in ('version', 'profile', 'collector', 'omabit', 'loaded', 'local_setup')}


def preflight(widget_loaded=True):
    require(os.getuid() == 1000 and Path.home() == Path('/home/thelifeandtimes'), 'run as the DEV owner, without sudo')
    previous = REPO / 'build/review040-update-060rc4-owner'
    require(json.loads((previous / 'complete.json').read_text())['phase'] == 'complete', 'RC4 update has not completed')
    require(base.tree(OLD) == json.loads((previous / 'plan.json').read_text())['plugin_candidate'], 'installed RC4 widget differs from its completed update')
    for name, digest in base.TOOLS.items():
        require(base.sha(OLD / 'bin' / name) == base.sha(base.DEV / 'bin' / name) == digest, 'RC4 helper differs: ' + name)
    base.hello(base.SOURCE, 67)
    if widget_loaded:
        ready(base.VERSION, OLD)
    else:
        require_unloaded()
    require(not NEW.exists(), 'UI preview destination already exists; inspect its receipt')


def require_unloaded():
    # A missing widget target only proves unload when the shell itself responds.
    require(ipc('shell', 'ping') == 'ok', 'Omarchy shell is not available')
    result = subprocess.run(['omarchy', 'shell', base.PLUGIN_ID + '.review', 'status'],
                            capture_output=True, text=True, timeout=10)
    require(result.returncode != 0 and (result.stdout + result.stderr).strip() == 'Target not found.',
            'DEV widget has not verifiably unloaded')


def stage(target):
    before = base.tree(OLD)
    for name, (old, new) in FILES.items():
        require(base.sha(OLD / 'app060rc4' / name) == old, 'original UI changed: ' + name)
        require(base.sha(REPO / 'plugins/omabit' / name) == new, 'reviewed UI source changed: ' + name)
    shutil.copytree(OLD, target)
    for name in FILES:
        shutil.copyfile(REPO / 'plugins/omabit' / name, target / 'app060rc4' / name)
    service = target / 'app060rc4/Service.qml'
    text = service.read_text()
    marker = 'readonly property string applicationVersion: "' + base.VERSION + '"'
    require(text.count(marker) == 1, 'application version binding differs')
    service.write_text(text.replace(marker, 'readonly property string applicationVersion: "' + VERSION + '"'))
    manifest = json.loads((target / 'manifest.json').read_text())
    manifest.update(version=VERSION, name='Omabit RC4 DEV · UI 1', description='Local DEV UI preview: Vere selection inside Runtime Settings. RC4 helpers unchanged.')
    manifest['barWidget']['displayName'] = manifest['name']
    (target / 'manifest.json').write_text(json.dumps(manifest, indent=2) + '\n')
    wrapper = target / 'ReviewService060.qml'
    wrapper.write_text(wrapper.read_text().replace('Omabit 0.6.0-rc.4 DEV', 'Omabit RC4 DEV · UI 1'))
    changed = {name for name, value in base.tree(target).items() if before.get(name) != value}
    require(changed == {'app060rc4/' + n for n in FILES} | {'app060rc4/Service.qml', 'manifest.json', 'ReviewService060.qml'}, 'unexpected overlay file changes')
    base.write(target / 'DEV-UI.json', {'version': VERSION, 'kind': 'local-ui-preview', 'base_release': base.VERSION,
        'base_source': base.SOURCE, 'base_build_sha256': base.sha(target / 'BUILD.json'),
        'description': 'Move Vere runtime selection into Runtime Settings.',
        'overrides': {n: base.sha(target / n) for n in sorted(changed)}})


def prepare():
    require(not WORK.exists(), 'preparation exists; use --check instead')
    preflight()
    base.budget(base.copy_bytes(OLD) + 16 * 1024**2)
    WORK.mkdir(mode=0o700)
    stage(WORK / 'plugin')
    base.write(WORK / 'plan.json', {'version': VERSION, 'script_sha256': base.sha(__file__),
        'base_script_sha256': base.sha(base.__file__), 'before': base.tree(OLD), 'candidate': base.tree(WORK / 'plugin')})
    check()


def check():
    require(not (WORK / 'apply-started.json').exists(), 'update already started; inspect complete.json or failed.json; do not rerun')
    plan = json.loads((WORK / 'plan.json').read_text())
    require(plan['version'] == VERSION and plan['script_sha256'] == base.sha(__file__)
            and plan['base_script_sha256'] == base.sha(base.__file__), 'prepared updater changed')
    preflight()
    require(base.tree(OLD) == plan['before'] and base.tree(WORK / 'plugin') == plan['candidate'], 'prepared UI bytes changed')
    require(WORK.stat().st_dev == OLD.stat().st_dev, 'staging must share the plugin filesystem')
    # Activation uses renames, not a second full plugin copy.
    charged = base.budget(16 * 1024**2)
    print(json.dumps({'status': 'ready', 'version': VERSION, 'runner_update': False, 'charged_bytes': charged}))
    return plan


def activate():
    """The same filesystem/IPC activation is rehearsed in a private full shell."""
    original = base.CONFIG.read_bytes()
    expected = disabled_settings(original)
    base.write(WORK / 'apply-started.json', {'version': VERSION, 'old_plugin': str(OLD), 'new_plugin': str(NEW)})
    (WORK / 'shell-before.json').write_bytes(original)
    phase = 'disable-dev-widget'
    try:
        require(ipc('shell', 'setPluginEnabled', base.PLUGIN_ID, 'false') == 'ok', 'DEV widget disable refused')
        base.wait_for(require_unloaded)
        disabled = base.CONFIG.read_bytes()
        (WORK / 'shell-disabled.json').write_bytes(disabled)
        require(json.loads(disabled) == expected, 'shell settings changed while disabling the DEV widget')
        phase = 'activate-ui-preview'
        return install_widget(original, disabled)
    except BaseException as error:
        base.write(WORK / 'failed.json', {'phase': phase, 'error': str(error),
            'action': 'Retain this directory and inspect the checkpoint. No runner, runtime or database upgrade was attempted.'})
        raise


def install_widget(original, disabled):
    require(base.CONFIG.read_bytes() == disabled, 'shell settings changed before widget activation')
    require_unloaded()
    OLD.rename(WORK / 'plugin-before')
    (WORK / 'plugin').rename(NEW)
    ipc('shell', 'rescanPlugins')
    base.restore_config(original, disabled)
    ipc('shell', 'reloadConfig')
    ipc('shell', 'rescanPlugins')
    loaded = base.wait_for(lambda: ready(VERSION, NEW))
    require(base.CONFIG.read_bytes() == original, 'shell settings changed')
    base.write(WORK / 'activated.json', loaded)
    return loaded


def finish(plan, loaded, manager, engine):
    require(base.properties(base.UNIT) == manager and base.properties(base.ENGINE) == engine, 'DEV service identity changed during UI activation')
    require(base.tree(NEW) == plan['candidate'] and base.tree(WORK / 'plugin-before') == plan['before'], 'installed UI or retained backup changed')
    base.write(WORK / 'complete.json', {'phase': 'complete', 'loaded': loaded,
        'backup': str(WORK / 'plugin-before'), 'runner_and_engine_unchanged': True})
    print(json.dumps({'status': 'complete', 'version': VERSION, 'receipt': str(WORK / 'complete.json')}))


def apply():
    plan = check()
    manager = base.properties(base.UNIT)
    engine = base.properties(base.ENGINE)
    finish(plan, activate(), manager, engine)


def recovery_checkpoint():
    for name in ('recovery-started.json', 'recovery-failed.json', 'activated.json', 'complete.json', 'plugin-before'):
        require(not (WORK / name).exists(), 'recovery cannot resume this checkpoint: ' + name)
    plan = json.loads((WORK / 'plan.json').read_text())
    failure = json.loads((WORK / 'failed.json').read_text())
    started = json.loads((WORK / 'apply-started.json').read_text())
    require(plan['version'] == VERSION and plan['script_sha256'] == FAILED_UPDATER_SHA
            and plan['base_script_sha256'] == base.sha(base.__file__), 'not the reviewed UI1 failed updater')
    require(failure['phase'] == 'disable-dev-widget' and failure['error'] == 'shell settings changed while disabling the DEV widget', 'not the recoverable shell-normalization failure')
    require(started == {'version': VERSION, 'old_plugin': str(OLD), 'new_plugin': str(NEW)}, 'failed attempt paths differ')
    require(not NEW.exists(), 'new plugin already exists; inspect checkpoint')
    require(base.tree(OLD) == plan['before'] and base.tree(WORK / 'plugin') == plan['candidate'], 'original or staged plugin changed')
    require(WORK.stat().st_dev == OLD.stat().st_dev, 'staging must share the plugin filesystem')
    original = (WORK / 'shell-before.json').read_bytes()
    disabled = (WORK / 'shell-disabled.json').read_bytes()
    require(json.loads(disabled) == disabled_settings(original), 'checkpoint contains unrelated shell changes')
    require(base.CONFIG.read_bytes() == disabled, 'live shell settings changed after the failed update')
    return plan, original, disabled


def check_recovery():
    checkpoint = recovery_checkpoint()
    preflight(widget_loaded=False)
    charged = base.budget(16 * 1024**2)
    print(json.dumps({'status': 'ready-to-recover', 'version': VERSION, 'runner_update': False, 'charged_bytes': charged}))
    return checkpoint


def recover_disabled():
    plan, original, disabled = check_recovery()
    manager = base.properties(base.UNIT)
    engine = base.properties(base.ENGINE)
    base.write(WORK / 'recovery-started.json', {'version': VERSION, 'updater_sha256': base.sha(__file__),
        'original_failure_sha256': base.sha(WORK / 'failed.json'), 'original_plan_sha256': base.sha(WORK / 'plan.json')})
    try:
        finish(plan, install_widget(original, disabled), manager, engine)
    except BaseException as error:
        base.write(WORK / 'recovery-failed.json', {'error': str(error), 'action': 'Retain this checkpoint; do not rerun.'})
        raise


@contextmanager
def delivery_lock():
    # /tmp is cleared at reboot. The lock is coordination, not a checkpoint:
    # create it if absent, never truncate/unlink it, and retain existing locks.
    fd = os.open(LOCK, os.O_RDWR | os.O_CREAT | os.O_NOFOLLOW | os.O_CLOEXEC | os.O_NONBLOCK, 0o600)
    try:
        node = os.fstat(fd)
        require(stat.S_ISREG(node.st_mode) and node.st_uid == os.geteuid()
                and node.st_nlink == 1 and not node.st_mode & 0o022, 'unsafe delivery lock file')
        try:
            fcntl.flock(fd, fcntl.LOCK_EX | fcntl.LOCK_NB)
        except BlockingIOError as error:
            raise RuntimeError('another Omabit update or qualification is running') from error
        yield
    finally:
        os.close(fd)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    mode = parser.add_mutually_exclusive_group(required=True)
    for name in ('prepare', 'check', 'apply', 'check-recovery', 'recover-disabled'):
        mode.add_argument('--' + name, action='store_true')
    args = parser.parse_args()
    # Share the existing delivery lock with backend updaters.
    try:
        with delivery_lock():
            (prepare if args.prepare else apply if args.apply else check_recovery if args.check_recovery
             else recover_disabled if args.recover_disabled else check)()
    except (RuntimeError, OSError, ValueError, subprocess.SubprocessError) as error:
        raise SystemExit(str(error))


if __name__ == '__main__':
    main()
