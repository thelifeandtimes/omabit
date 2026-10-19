#!/usr/bin/env python3
"""Explicitly guarded native-package qualification; no arguments only shows help."""
import argparse
import json
import os
from pathlib import Path
import subprocess
import sys

sys.dont_write_bytecode = True

from guards import Probe, Refusal, digest, require, validate


def parser():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('action', nargs='?', choices=('plan', 'validate', 'run'))
    p.add_argument('--format', choices=('json', 'text'), default='json')
    p.add_argument('--provider', choices=('apt', 'pacman', 'both'), default='apt')
    p.add_argument('--workspace', default='/opt/hostcompat-qualification')
    p.add_argument('--hostname')
    p.add_argument('--vm-uuid')
    p.add_argument('--candidate')
    p.add_argument('--candidate-sha256')
    p.add_argument('--candidate-version', help='exact development build label pinned in the provisioned marker')
    p.add_argument('--candidate-schema', help='exact output of the candidate state-schema command')
    p.add_argument('--hooks', help='directory with the three generated, marker-hash-pinned hook assets')
    p.add_argument('--pacman-root', help='fresh independently verified Arch extraction below workspace, never /')
    p.add_argument('--acknowledge-disposable-guest', action='store_true')
    return p


def output(value, fmt, stream):
    if fmt == 'json':
        print(json.dumps(value, indent=2, sort_keys=True), file=stream)
    else:
        for key, val in value.items():
            print(f'{key}: {json.dumps(val) if isinstance(val, (dict, list)) else val}', file=stream)


def main(argv=None, *, probe=None, execute=None):
    p = parser()
    args = p.parse_args(argv)
    if args.action is None:
        p.print_help()
        return 0
    if args.action == 'plan':
        output({'mutated': False, 'scope': 'synthetic package authentication only; never engine qualification',
                'phases': ['read-only VM/marker/owner/process/hash guards', 'exact state-schema check',
                           'fresh signed synthetic fixture setup', 'native offline/fetch and negative cases',
                           'first-use/replacement transaction hooks and cache cleanup',
                           'owner reads, authority/colon/multisigner checks for APT'],
                'outputs': 'workspace/run; fresh guest only; no automatic reset or provisioning'}, args.format, sys.stdout)
        return 0
    try:
        if args.action == 'run':
            require(args.acknowledge_disposable_guest, 'run requires --acknowledge-disposable-guest')
        admission = validate(args, probe)
        if args.action == 'validate':
            output(admission, args.format, sys.stdout)
            return 0
        if execute is None:
            # Importing modules is inert. No case, write or candidate execution can
            # precede the full guard. Rehash the pinned file before executing it.
            require(digest(args.candidate) == args.candidate_sha256, 'candidate changed after validation')
            r = subprocess.run([args.candidate, 'state-schema'], capture_output=True, timeout=10,
                               env={'PATH': '/usr/bin:/usr/sbin:/bin:/sbin', 'HOME': '/root', 'LC_ALL': 'C'})
            require(r.returncode == 0 and r.stdout.decode().strip() == args.candidate_schema,
                    'candidate state-schema does not match the pinned version')
            require(digest(args.candidate) == args.candidate_sha256, 'candidate changed during version check')
            # Recheck live owner/process boundaries immediately before mutation.
            live = Probe()
            live.processes()
            live.clean_root(Path('/'))
            if args.pacman_root:
                live.clean_root(Path(args.pacman_root), pacman=True)
            from suite import execute
        old_umask = os.umask(0o022)
        try:
            # Fixture setup has ordinary public-directory semantics. The native
            # verification and transaction cases explicitly switch to umask077.
            return execute(args, admission)
        finally:
            os.umask(old_umask)
    except (Refusal, OSError, ValueError, subprocess.SubprocessError) as exc:
        output({'validated': False, 'error': str(exc)}, args.format, sys.stderr)
        return 2


if __name__ == '__main__':
    sys.exit(main())
