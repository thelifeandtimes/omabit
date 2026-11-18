"""Storage admission for this host's disposable development capsules.

Measurements are allocated bytes, never apparent LMDB lengths. Symlink aliases
are not counted twice. The DEV count may be incomplete only for the previously
reviewed rootless overlay work directories; its reserve is explicitly an estimate.
"""
import os
from pathlib import Path
import re
import subprocess
import stat

GIB = 1024**3
BUDGET = 40 * GIB
DEV = Path('/srv/dev/urbit')
NAMED_DEV = Path('/srv/dev/urbit-040')


def measure_development_namespace(path):
    """Read allocated blocks with the existing rootless engine's file authority.

    This joins only its user namespace for a fixed read-only du command. It does
    not join its mount/process namespace, contact Docker, or change permissions.
    A stale PID, changed namespace or incomplete read refuses the measurement.
    """
    if path not in (DEV, NAMED_DEV):
        raise RuntimeError('namespace measurement requires a known DEV root')
    name = 'omabit-dev' + str(path).removeprefix('/srv/dev/urbit')
    pidfile = Path('/run/user') / str(os.geteuid()) / name / 'rootlesskit/child_pid'
    try:
        node = pidfile.lstat()
        if not stat.S_ISREG(node.st_mode) or node.st_uid != os.geteuid() or node.st_nlink != 1 or node.st_mode & 0o022:
            raise RuntimeError('unsafe DEV engine PID evidence')
        pid = pidfile.read_text().strip()
        if not pid.isascii() or not pid.isdigit() or not 1 < int(pid) < 2**31:
            raise RuntimeError('invalid DEV engine PID')
        namespace = Path('/proc') / pid / 'ns/user'
        before = os.readlink(namespace)
        result = subprocess.run(['/usr/bin/nsenter', '--user=' + str(namespace), '--', '/usr/bin/du', '-s', '-B1', '--', str(path)],
                                capture_output=True, text=True, check=False,
                                env={**os.environ, 'LC_ALL': 'C'}, timeout=60)
        if result.returncode or result.stderr or before != os.readlink(namespace) or pidfile.read_text().strip() != pid:
            raise RuntimeError('DEV namespace allocation measurement incomplete or changed')
        fields = result.stdout.split()
        if len(fields) != 2 or fields[1] != str(path) or not fields[0].isdigit():
            raise RuntimeError('invalid DEV namespace allocation result')
        return {'path': str(path), 'allocated_bytes': int(fields[0]), 'unmeasured_reserve': 0,
                'measurement_source': 'existing-rootless-user-namespace', 'namespace': before}
    except OSError as error:
        raise RuntimeError('DEV namespace allocation evidence unavailable') from error


def measure(path):
    result = subprocess.run(['/usr/bin/du', '-s', '-B1', '--', str(path)],
                            capture_output=True, text=True, check=False,
                            env={**os.environ, 'LC_ALL': 'C'}, timeout=60)
    if not result.stdout:
        raise RuntimeError('storage measurement unavailable: ' + str(path))
    row = {'path': str(path), 'allocated_bytes': int(result.stdout.split()[0]),
           'unmeasured_reserve': 0}
    if result.returncode:
        errors = result.stderr.splitlines()
        pattern = r"du: cannot read directory '/srv/dev/urbit/state/\.local/share/urbit-runner/docker/containerd/daemon/io\.containerd\.snapshotter\.v1\.overlayfs/snapshots/[0-9]+/work/work': Permission denied"
        pattern = pattern.replace('/srv/dev/urbit/', re.escape(str(path)) + '/')
        if path not in (DEV, NAMED_DEV) or not errors or not all(re.fullmatch(pattern, e) for e in errors):
            raise RuntimeError('incomplete storage measurement: ' + str(path))
        try:
            return measure_development_namespace(path)
        except RuntimeError:
            if len(errors) > 18:
                # Do not expand the historical partial-measurement allowance.
                raise
        row.update(unreadable_work_directories=len(errors), unmeasured_reserve=GIB)
    return row


def roots():
    base = Path.home() / 'Documents/ChatGPT'
    required = [base / 'omabit-private-docker-labs', base / 'omabit-qualification-evidence', DEV]
    if any(not p.is_dir() or p.is_symlink() for p in required):
        raise RuntimeError('expected physical storage roots unavailable')
    if NAMED_DEV.exists():
        if NAMED_DEV.is_symlink() or not NAMED_DEV.is_dir():
            raise RuntimeError("unsafe named development storage root")
        required.append(NAMED_DEV)
    # Include installed native-review artifacts and locally retired trial profiles.
    for path in (Path.home()/'.config/omarchy/plugins/thelifeandtimes.omabit.review040',
                 Path.home()/'.local/share/omabit-review-040-retired-20261112'):
        if path.exists():
            if path.is_symlink() or not path.is_dir():
                raise RuntimeError('unsafe native review artifact root')
            required.append(path)
    return required + sorted(p for p in Path('/tmp').glob('omabit-*') if not p.is_symlink())


def admit(additional_bytes, measurements=None, *, budget_bytes=None):
    # A larger budget is an explicit, separately authorized qualification input;
    # ordinary development retains the original 40 GiB limit.
    budget = BUDGET if budget_bytes is None else budget_bytes
    if budget not in (40 * GIB, 48 * GIB):
        raise ValueError('unsupported experiment storage budget')
    if not 0 <= additional_bytes <= 4 * GIB:
        raise ValueError('invalid additional scratch allowance')
    rows = measurements if measurements is not None else [measure(p) for p in roots()]
    used = sum(row['allocated_bytes'] + row['unmeasured_reserve'] for row in rows)
    report = {'measurements': rows, 'charged_bytes': used, 'budget_bytes': budget,
              'additional_scratch_bytes': additional_bytes,
              'total_with_reserves': used + additional_bytes}
    if report['total_with_reserves'] > budget:
        raise RuntimeError('storage budget exceeded: ' + str(report['total_with_reserves']))
    return report
