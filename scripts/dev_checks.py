"""Named local feedback lanes. These do not replace release qualification."""
import argparse
import os
from pathlib import Path
import re
import shlex
import subprocess
import sys
import time

ROOT = Path(__file__).resolve().parents[1]
PYTHON = [sys.executable, '-B']
DESKTOP = [
    ['node', 'scripts/test-panel-model.js'],
    ['node', 'scripts/test-runtime-settings-model.js'],
    *[PYTHON + ['scripts/' + name] for name in (
        'test-panel-install.py', 'test-native-plugin.py', 'test-desktop-bootstrap.py',
        'test-bootstrap-observation.py', 'test-observation-schedule.py')],
]
TOOLING = [PYTHON + ['scripts/' + name] for name in (
    'test-dev-loop.py', 'test-channels.py', 'test-release-policy.py',
    'test-skill-package.py', 'test-release-source.py')]
QML = [PYTHON + ['scripts/' + name, '--require-qt'] for name in (
    'test-rc11-panel-offscreen.py', 'test-host-compatibility-offscreen.py',
    'test-service-upgrade-offscreen.py')]


def commands(scope, packages=(), run=None, race=False):
    if (packages or run or race) and scope != 'go':
        raise ValueError('--package, --run and --race require --scope go')
    if any(not re.fullmatch(r'\./[A-Za-z0-9_./-]+', package) or '..' in package.split('/')
           for package in packages):
        raise ValueError('use repository-relative Go package paths, e.g. ./internal/client')
    if scope == 'desktop':
        return DESKTOP
    if scope == 'tooling':
        return TOOLING
    if scope == 'fast':
        return DESKTOP + TOOLING
    if scope == 'qml':
        return QML
    if scope == 'go':
        selected = list(packages) or ['./...']
        flags = ['-timeout=30m'] + (['-race'] if race else []) + (['-run', run] if run else [])
        return [PYTHON + ['scripts/ci-go-tests.py', *flags, *selected], ['go', 'vet', *selected]]
    raise ValueError('unknown check scope')


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--scope', choices=('fast', 'desktop', 'tooling', 'qml', 'go'), default='fast')
    parser.add_argument('--package', action='append', default=[], help='Go package; repeat to check several (default: ./...)')
    parser.add_argument('--run', help='explicit Go test-name filter for focused feedback')
    parser.add_argument('--race', action='store_true')
    parser.add_argument('--plan', action='store_true', help='show commands without running tests')
    args = parser.parse_args(argv)
    try:
        plan = commands(args.scope, args.package, args.run, args.race)
    except ValueError as error:
        parser.error(str(error))
    print(f'Local {args.scope} feedback; full CI and upgrade/fleet qualification remain separate.', flush=True)
    started = time.monotonic()
    for command in plan:
        print('+ ' + shlex.join(command), flush=True)
        if not args.plan:
            subprocess.run(command, cwd=ROOT, env=dict(os.environ, TZ='UTC'), check=True)
    if not args.plan:
        print(f'{args.scope} checks passed in {time.monotonic() - started:.1f}s', flush=True)
    return 0
