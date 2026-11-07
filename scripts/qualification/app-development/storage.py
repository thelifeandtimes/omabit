"""Storage admission for this host's disposable development capsules.

Measurements are allocated bytes, never apparent LMDB lengths. Symlink aliases
are not counted twice. The DEV count may be incomplete only for the previously
reviewed rootless overlay work directories; its reserve is explicitly an estimate.
"""
import os
from pathlib import Path
import re
import subprocess

GIB = 1024**3
BUDGET = 40 * GIB
DEV = Path('/srv/dev/urbit')


def measure(path):
    result = subprocess.run(['du', '-s', '-B1', '--', str(path)],
                            capture_output=True, text=True, check=False,
                            env={**os.environ, 'LC_ALL': 'C'}, timeout=60)
    if not result.stdout:
        raise RuntimeError('storage measurement unavailable: ' + str(path))
    row = {'path': str(path), 'allocated_bytes': int(result.stdout.split()[0]),
           'unmeasured_reserve': 0}
    if result.returncode:
        errors = result.stderr.splitlines()
        pattern = r"du: cannot read directory '/srv/dev/urbit/state/\.local/share/urbit-runner/docker/containerd/daemon/io\.containerd\.snapshotter\.v1\.overlayfs/snapshots/[0-9]+/work/work': Permission denied"
        if path != DEV or not 1 <= len(errors) <= 18 or not all(re.fullmatch(pattern, e) for e in errors):
            raise RuntimeError('incomplete storage measurement: ' + str(path))
        row.update(unreadable_work_directories=len(errors), unmeasured_reserve=GIB)
    return row


def roots():
    base = Path.home() / 'Documents/ChatGPT'
    required = [base / 'omabit-private-docker-labs', base / 'omabit-qualification-evidence', DEV]
    if any(not p.is_dir() or p.is_symlink() for p in required):
        raise RuntimeError('expected physical storage roots unavailable')
    return required + sorted(p for p in Path('/tmp').glob('omabit-*') if not p.is_symlink())


def admit(additional_bytes, measurements=None):
    if not 0 <= additional_bytes <= 2 * GIB:
        raise ValueError('invalid additional scratch allowance')
    rows = measurements if measurements is not None else [measure(p) for p in roots()]
    used = sum(row['allocated_bytes'] + row['unmeasured_reserve'] for row in rows)
    report = {'measurements': rows, 'charged_bytes': used, 'budget_bytes': BUDGET,
              'additional_scratch_bytes': additional_bytes,
              'total_with_reserves': used + additional_bytes}
    if report['total_with_reserves'] > BUDGET:
        raise RuntimeError('storage budget exceeded: ' + str(report['total_with_reserves']))
    return report
