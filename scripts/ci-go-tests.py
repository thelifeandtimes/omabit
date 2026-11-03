#!/usr/bin/env python3
"""Run Go tests with complete duration-balanced shards and public-name summaries."""
import json
import math
import os
from pathlib import Path
import re
import subprocess
import sys

ROOT = Path(__file__).resolve().parents[1]
NAME = re.compile(r'(?:Test|Example|Fuzz)[A-Za-z0-9_]*\Z')


def shard_groups(names, count, durations):
    """Longest-known-test first; unknown tests get a conservative 30s estimate.

    Timings only affect placement. The executable inventory determines coverage,
    so adding/removing a test never requires editing the timing file to run it.
    """
    if not 1 <= count <= 16 or not names or len(set(names)) != len(names):
        raise ValueError('invalid managed test inventory/shard count')
    if any(not NAME.fullmatch(name) for name in names):
        raise ValueError('invalid managed test name')
    if any(type(value) not in (int, float) or not math.isfinite(value) or value < 0 for value in durations.values()):
        raise ValueError('invalid measured test duration')
    weights = {name: max(0.01, durations.get(name, 30.0)) for name in names}
    groups, totals = [[] for _ in range(count)], [0.0] * count
    for name in sorted(names, key=lambda name: (-weights[name], name)):
        index = min(range(count), key=lambda shard: (totals[shard], shard))
        groups[index].append(name)
        totals[index] += weights[name]
    if (set().union(*map(set, groups)) != set(names) or
            sum(map(len, groups)) != len(names) or any(not group for group in groups)):
        raise ValueError('managed test shards do not cover the inventory exactly once')
    return groups, totals


def inventory():
    module = re.search(r'^module (\S+)', (ROOT / 'go.mod').read_text(), re.M).group(1)
    known = {}
    for path in ROOT.rglob('*_test.go'):
        if '.git' in path.parts:
            continue
        package = module + '/' + path.parent.relative_to(ROOT).as_posix()
        known.setdefault(package, set()).update(re.findall(r'^func ((?:Test|Example|Fuzz)[A-Za-z0-9_]*)\(', path.read_text(), re.M))
    return module, known


def managed_plan(count, module, known):
    listing = subprocess.check_output(['go', 'test', '-list', '.', './internal/managed'], cwd=ROOT, text=True)
    names = [line for line in listing.splitlines() if NAME.fullmatch(line)]
    if not set(names) <= known.get(module + '/internal/managed', set()):
        raise ValueError('managed test inventory differs from declared source')
    durations = json.loads((ROOT / 'scripts/managed-race-durations.json').read_text())['seconds']
    groups, totals = shard_groups(names, count, durations)
    return [{'index': index, 'estimated_seconds': round(totals[index], 2), 'tests': sorted(group)}
            for index, group in enumerate(groups)]


def record_event(row, known, failures, timings):
    package, name = row.get('Package'), row.get('Test', '')
    if package not in known or not isinstance(name, str):
        return
    top = name.split('/', 1)[0]
    if row.get('Action') == 'fail' and (not top or top in known[package]):
        failures.add((package, top))
    elapsed = row.get('Elapsed')
    if (row.get('Action') in ('pass', 'fail') and name in known[package] and
            type(elapsed) in (int, float) and math.isfinite(elapsed) and elapsed >= 0):
        timings[(package, name)] = elapsed


def main(options=None):
    options = list(sys.argv[1:] if options is None else options)
    module, known = inventory()
    if options[:1] == ['--managed-plan']:
        if len(options) != 2:
            raise ValueError('usage: --managed-plan COUNT')
        print(json.dumps(managed_plan(int(options[1]), module, known), indent=2))
        return 0
    if options[:1] == ['--managed-shard']:
        if len(options) < 2:
            raise ValueError('usage: --managed-shard INDEX/COUNT [go test flags]')
        index, count = map(int, options[1].split('/'))
        if not 0 <= index < count <= 16:
            raise ValueError('invalid managed test shard')
        options = options[2:]
        # A second filter could silently drop tests from the exhaustive CI gate.
        if any(arg.split('=', 1)[0] in ('-run', '-skip', '-list', '-shuffle') for arg in options):
            raise ValueError('managed shard owns test selection and order')
        plan = managed_plan(count, module, known)
        selected = plan[index]
        print(f'Managed race shard {index + 1}/{count}: {len(selected["tests"])} tests; '
              f'estimated {selected["estimated_seconds"]:.0f}s from historical timings', flush=True)
        options += ['-run', '^(' + '|'.join(map(re.escape, selected['tests'])) + ')$', './internal/managed']
    process = subprocess.Popen(['go', 'test', '-json', *options], cwd=ROOT, stdout=subprocess.PIPE,
                               stderr=subprocess.STDOUT, text=True)
    failures, timings = set(), {}
    for line in process.stdout:
        print(line, end='', flush=True)
        try:
            row = json.loads(line)
        except ValueError:
            continue
        if isinstance(row, dict):
            record_event(row, known, failures, timings)
    process.stdout.close()
    status = process.wait()
    for package, name in sorted(failures):
        print('::error title=Go fixture failed::' + package + (' ' + name if name else ''), flush=True)
    if os.environ.get('GITHUB_STEP_SUMMARY'):
        # Never put arbitrary Go output/subtest names in the Markdown summary.
        with open(os.environ['GITHUB_STEP_SUMMARY'], 'a') as stream:
            stream.write('\n### Slowest Go tests\n\n| Test | Seconds |\n| --- | ---: |\n')
            for (package, name), elapsed in sorted(timings.items(), key=lambda item: (-item[1], item[0]))[:15]:
                stream.write(f'| `{package} {name}` | {elapsed:.2f} |\n')
    return status


if __name__ == '__main__':
    try:
        sys.exit(main())
    except (ValueError, OSError, subprocess.SubprocessError) as error:
        sys.exit(str(error))
