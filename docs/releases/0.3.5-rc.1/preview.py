#!/usr/bin/env python3
"""Open the frozen candidate in a separate desktop process and private profile.

Uses private review state and a short-lived /tmp runtime. Never changes a host
service, plugin, enrollment or existing DEV profile. --check opens no windows.
"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import tempfile
import subprocess
import time

ROOT = Path(__file__).resolve().parent
VERSION = '0.3.5-rc.1'
SOURCE = 'cce7cd017bf33b08594a82f0dd8286caed6d8811'


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--check', action='store_true')
    args = parser.parse_args()
    assets = ROOT / VERSION
    manifest = json.loads((assets / 'release.json').read_text())
    assert manifest['version'] == VERSION and manifest['source_commit'] == SOURCE
    archive = assets / ('omabit-' + VERSION + '-linux-amd64.tar.gz')
    entry = next(a for a in manifest['assets'] if a['name'] == archive.name)
    assert archive.stat().st_size == entry['bytes']
    assert hashlib.sha256(archive.read_bytes()).hexdigest() == entry['sha256']
    private = ROOT / 'preview-state'
    if private.is_symlink():
        raise RuntimeError('Refusing a linked review state directory')
    private.mkdir(mode=0o700, exist_ok=True)
    env = {'PATH': '/usr/bin:/bin', 'LANG': 'C.UTF-8', 'QT_QUICK_CONTROLS_STYLE': 'Basic',
           'QML_DISABLE_DISK_CACHE': '1', 'OMABIT_REVIEW_CHECK': '1' if args.check else '0'}
    for key, leaf in (('HOME', 'home'), ('XDG_CONFIG_HOME', 'config'),
                      ('XDG_STATE_HOME', 'state'), ('XDG_CACHE_HOME', 'cache'),
                      ('XDG_DATA_HOME', 'data')):
        p = private / leaf
        if p.is_symlink():
            raise RuntimeError('Refusing a linked review state path')
        p.mkdir(mode=0o700, exist_ok=True)
        env[key] = str(p)
    # AF_UNIX socket names are limited to 108 bytes; the artifact path is longer.
    with tempfile.TemporaryDirectory(prefix='omabit-review035-') as runtime:
        env['XDG_RUNTIME_DIR'] = runtime
        display = os.environ.get('WAYLAND_DISPLAY', '')
        if not display:
            raise RuntimeError('Run this preview from your Omarchy desktop terminal')
        env['WAYLAND_DISPLAY'] = display if display.startswith('/') else str(Path(os.environ['XDG_RUNTIME_DIR']) / display)
        env['QT_QPA_PLATFORM'] = 'wayland'
        # No host D-Bus, SSH agent, Hyprland control, production HOME/XDG or credentials.
        shell = ROOT / 'preview-shell' / 'shell.qml'
        control = shell.parent / 'native/bootstrap/control.py'
        pins = control.parent / 'release.json'
        def call(*argv):
            return json.loads(subprocess.check_output(['/usr/bin/python3', '-B', str(control), '--pins', str(pins), *map(str, argv)], env=env, text=True))
        imported = call('import-release', assets / 'release.json', archive)['release']
        status = call('status')
        if 'review035' not in status['profiles']:
            call('profile-create', 'review035')
        else:
            assert not status['profiles']['review035']['host_setup']
        call('select', 'review035', '--release', imported)
        call('prepare')
        with (private / 'quickshell.log').open('a') as log:
            process = subprocess.Popen(['/usr/bin/quickshell', '--no-color', '-p', str(shell)], env=env, stdout=log, stderr=subprocess.STDOUT)
            try:
                if args.check:
                    until = time.monotonic() + 40
                    while time.monotonic() < until:
                        assert process.poll() is None, 'Review process exited; inspect preview-state/quickshell.log'
                        r = subprocess.run(['/usr/bin/quickshell', 'ipc', '-p', str(shell), 'call', 'review035', 'status'], env=env, capture_output=True, text=True, timeout=3)
                        if r.returncode == 0:
                            data = json.loads(r.stdout)
                            if data['phase'] == 'ready':
                                assert data['loaded_version'] == VERSION and data['source_commit'] == SOURCE
                                assert data['profile'] == 'review035' and data['runners'] == data['ships'] == 0
                                print(json.dumps({'passed': True, 'diagnostic': data, 'private_state': str(private)}))
                                return
                        time.sleep(.2)
                    raise RuntimeError('Review readiness timeout; inspect preview-state/quickshell.log')
                print('Opened Omabit ' + VERSION + ' in an isolated review process. Close its launcher window to exit.', flush=True)
                process.wait()
            finally:
                if process.poll() is None:
                    process.terminate()
                    process.wait(timeout=15)


if __name__ == '__main__':
    main()
