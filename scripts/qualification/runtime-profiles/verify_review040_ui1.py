"""Rehearse the exact UI-only activation in a private Omarchy shell."""
import importlib.util
import argparse
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import time

repo = Path('/repo') if '--inside' in sys.argv else Path(__file__).resolve().parents[3]


def load(name, filename):
    spec = importlib.util.spec_from_file_location(name, repo / 'scripts/qualification/runtime-profiles' / filename)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


if '--inside' not in sys.argv:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--output', required=True)
    parser.add_argument('--recovery', action='store_true')
    options = parser.parse_args()
    launcher = load('shell_launcher', 'full_shell.py')
    original = launcher.subprocess.run

    def launch(args, **kwargs):
        if args[0] == 'bwrap':
            lab = Path(args[args.index('--bind') + 1])
            for name in ('config', 'data', 'state', 'cache'):
                (lab / 'dev/desktop' / name).mkdir(parents=True)
            shutil.copyfile(__file__, lab / 'ui-driver.py')
            if options.recovery:
                (lab / 'recovery-case').touch()
            extra = ['--dir', '/srv', '--dir', '/srv/dev', '--bind', str(lab / 'dev'), '/srv/dev/urbit-040',
                '--ro-bind', str(Path.home() / '.config/omarchy/plugins/thelifeandtimes.omabit.review040-v060rc4'), '/oldplugin',
                '--ro-bind', str(repo / 'build/review040-update-060rc4-ui1/plugin'), '/candidateplugin']
            i = args.index('--')
            args = args[:i] + extra + args[i:]
            args = ['/lab/ui-driver.py' if a == '/lab/driver.py' else a for a in args]
        return original(args, **kwargs)

    launcher.subprocess.run = launch
    sys.argv = [__file__, '--output', options.output]
    launcher.main()
    sys.exit()

update = load('ui_update', 'update_review040_ui1.py')
lab = Path('/lab')
home = lab / 'home'
plugins = home / '.config/omarchy/plugins'
update.OLD = plugins / 'thelifeandtimes.omabit.review040-v060rc4'
update.NEW = plugins / 'thelifeandtimes.omabit.review040-v060rc4-ui1'
update.WORK = lab / 'work'
update.base.CONFIG = home / '.config/omarchy/shell.json'
shutil.copytree('/oldplugin', update.OLD)
shutil.copytree('/candidateplugin', update.WORK / 'plugin')
disabled = [json.loads(p.read_text())['id'] for p in Path('/usr/share/omarchy/shell/plugins').glob('*/manifest.json')
            if json.loads(p.read_text())['id'] != 'omarchy.bar']
update.base.CONFIG.write_text(json.dumps({'version': 1, 'idle': {'screensaver': 0, 'lock': 0},
    'bar': {'position': 'top', 'layout': {k: ([{'id': update.base.PLUGIN_ID}] if k == 'right' else [])
        for k in ['left', 'centerBegin', 'center', 'centerEnd', 'right']}}, 'plugins': [], 'disabledPlugins': disabled}))
original = update.base.CONFIG.read_bytes()
before_tree = update.base.tree(update.OLD)
candidate_tree = update.base.tree(update.WORK / 'plugin')
os.environ.update(LD_LIBRARY_PATH='/tools/usr/lib', WLR_BACKENDS='headless', WLR_RENDERER='pixman', WLR_HEADLESS_OUTPUTS='1')
compositor = shell = None
report = {'passed': False, 'host_access': False, 'network': False, 'checks': {}}
try:
    compositor = subprocess.Popen(['/tools/usr/bin/sway', '--config', '/lab/sway.conf'], stdout=(lab / 'sway.log').open('w'), stderr=subprocess.STDOUT)
    end = time.monotonic() + 15
    while not list((lab / 'runtime').glob('wayland-*')):
        assert compositor.poll() is None and time.monotonic() < end
        time.sleep(.1)
    os.environ['WAYLAND_DISPLAY'] = str(next(p for p in (lab / 'runtime').glob('wayland-*') if p.is_socket()))
    shell = subprocess.Popen(['quickshell', '--no-color', '-p', '/usr/share/omarchy/shell'], stdout=(lab / 'shell.log').open('w'), stderr=subprocess.STDOUT)
    update.base.wait_for(lambda: update.ready(update.base.VERSION, update.OLD))
    report['checks']['previous_rc4_wrapper_loads'] = True
    if (lab / 'recovery-case').exists():
        update.base.write(update.WORK / 'plan.json', {'version': update.VERSION, 'script_sha256': update.FAILED_UPDATER_SHA,
            'base_script_sha256': update.base.sha(update.base.__file__), 'before': before_tree, 'candidate': candidate_tree})
        corrected = update.disabled_settings
        def old_guard(raw):
            value = corrected(raw)
            value['bar']['layout'].pop('tray')
            return value
        update.disabled_settings = old_guard
        try:
            update.activate()
            raise AssertionError('old guard did not reproduce owner failure')
        except RuntimeError as error:
            assert str(error) == 'shell settings changed while disabling the DEV widget'
        update.disabled_settings = corrected
        retained = {n: (update.WORK / n).read_bytes() for n in ('plan.json', 'failed.json', 'apply-started.json', 'shell-before.json', 'shell-disabled.json')}
        assert update.base.tree(update.OLD) == before_tree and not update.NEW.exists()
        # Only runner/owner preflight and budget are inert in the isolated shell;
        # checkpoint validation, retained files and actual IPC are unchanged.
        update.preflight = lambda widget_loaded=True: update.require_unloaded()
        update.base.budget = lambda additional: 0
        update.base.properties = lambda unit: {'ActiveState': 'active', 'unit': unit}
        update.recover_disabled()
        result = json.loads((update.WORK / 'complete.json').read_text())['loaded']
        assert all((update.WORK / n).read_bytes() == value for n, value in retained.items())
        try:
            update.recovery_checkpoint()
            raise AssertionError('recovery repeated')
        except RuntimeError as error:
            assert 'cannot resume' in str(error)
        report['checks'].update(owner_failure_reproduced=True, exact_recovery_files_and_IPC=True,
            original_failure_and_plan_retained=True, repeated_recovery_refused=True)
        report['runner_preflight_stubbed'] = True
    else:
        result = update.activate()
    assert shell.poll() is None and update.base.CONFIG.read_bytes() == original
    assert update.base.tree(update.NEW) == candidate_tree
    assert update.base.tree(update.WORK / 'plugin-before') == before_tree
    assert not update.OLD.exists()
    report['checks'].update(exact_adapter_activation=True, new_version_and_helper_paths=True,
        original_plugin_retained=True, original_shell_settings_preserved=True, no_shell_restart=True,
        missing_tray_normalization_accepted=True)
    # A second attempt must refuse before preflight or any installed-state work.
    update.preflight = lambda: (_ for _ in ()).throw(AssertionError('unexpected second preflight'))
    try:
        update.check()
        raise AssertionError('repeat attempt accepted')
    except RuntimeError as error:
        assert 'already started' in str(error)
    report['checks']['repeat_attempt_refused'] = True
    report.update(passed=True, after=result, updater_sha256=update.base.sha(update.__file__))
finally:
    for child in [shell, compositor]:
        if child and child.poll() is None:
            child.terminate()
            child.wait(timeout=15)
    (lab / 'result.json').write_text(json.dumps(report, indent=2) + '\n')
