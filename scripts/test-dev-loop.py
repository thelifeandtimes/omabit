#!/usr/bin/env python3
"""Offline development-loop regressions; all desktop state is disposable."""
import contextlib
import importlib.util
import io
import itertools
import json
import os
from pathlib import Path
import re
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / 'scripts'))
import dev_checks


def load(name, filename):
    spec = importlib.util.spec_from_file_location(name, ROOT / 'scripts' / filename)
    value = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(value)
    return value


dev = load('dev_candidate', 'omabit-dev.py')
ci = load('ci_tests', 'ci-go-tests.py')
fixtures = load('desktop_fixtures', 'test-desktop-bootstrap.py')


class ShardTests(unittest.TestCase):
    def test_every_test_once_including_new_tests_and_stale_timings(self):
        names = ['Test' + str(index) for index in range(40)]
        weights = {name: 300 / (index + 1) for index, name in enumerate(names[:30])}
        weights['TestRemoved'] = 9000
        groups, totals = ci.shard_groups(names, 8, weights)
        self.assertEqual(sorted(itertools.chain.from_iterable(groups)), sorted(names))
        self.assertEqual((groups, totals), ci.shard_groups(list(reversed(names)), 8, weights))
        self.assertLessEqual(max(totals) - min(totals), 300)
        self.assertTrue(all(groups))

    def test_measured_workload_is_balanced(self):
        weights = json.loads((ROOT / 'scripts/managed-race-durations.json').read_text())['seconds']
        groups, totals = ci.shard_groups(list(weights), 8, weights)
        self.assertLess(max(totals) - min(totals), 5)
        self.assertEqual(sum(map(len, groups)), len(weights))

    def test_invalid_or_incomplete_inventory_refused(self):
        for names, count, weights in (([], 2, {}), (['TestA'] * 2, 1, {}),
                                      (['TestA'], 2, {}), (['bad/name'], 1, {}),
                                      (['TestA'], 0, {}), (['TestA'], 1, {'TestA': float('nan')})):
            with self.subTest(names=names, count=count), self.assertRaises(ValueError):
                ci.shard_groups(names, count, weights)

    def test_second_filter_cannot_reduce_managed_coverage(self):
        with patch.object(ci, 'inventory', return_value=('module', {})):
            for flag in ('-run=TestOne', '-skip=TestOther', '-list', '-shuffle=on'):
                with self.subTest(flag=flag), self.assertRaises(ValueError):
                    ci.main(['--managed-shard', '0/8', flag])

    def test_summary_only_contains_declared_top_level_names(self):
        failures, timings = set(), {}
        known = {'public/package': {'TestPublic'}}
        for package, name in (('public/package', 'TestPublic'), ('private/package', 'TestPublic'),
                              ('public/package', 'TestPublic/private-input'), ('public/package', 'TestSecret')):
            ci.record_event({'Package': package, 'Test': name, 'Action': 'fail', 'Elapsed': 1}, known, failures, timings)
        self.assertEqual(timings, {('public/package', 'TestPublic'): 1})
        self.assertEqual(failures, {('public/package', 'TestPublic')})

    def test_source_gate_rejects_failure_cancel_and_skip(self):
        workflow = (ROOT / '.github/workflows/checks.yml').read_text()
        source = workflow.split('\n  source:\n', 1)[1].split('\n  qml-offscreen:', 1)[0]
        self.assertIn('if: always()', source)
        self.assertIn('needs: [core, transport-race, managed-race]', source)
        gate = re.search(r'^        run: (.*)$', source, re.M).group(1)
        for states in itertools.product(('success', 'failure', 'cancelled', 'skipped'), repeat=3):
            result = subprocess.run(['sh', '-c', gate], env=dict(os.environ, **dict(zip(('CORE', 'TRANSPORT', 'RACE'), states))))
            self.assertEqual(result.returncode == 0, states == ('success',) * 3, states)


class CandidateTests(unittest.TestCase):
    def setUp(self):
        temporary = tempfile.TemporaryDirectory(prefix='omabit-dev-loop-')
        self.addCleanup(temporary.cleanup)
        self.root = Path(temporary.name)
        self.env = {key: str(self.root / leaf) for key, leaf in (
            ('HOME', 'home'), ('XDG_CONFIG_HOME', 'config'), ('XDG_STATE_HOME', 'state'),
            ('XDG_DATA_HOME', 'data'), ('XDG_CACHE_HOME', 'cache'))}
        context = patch.dict(os.environ, self.env)
        context.start()
        self.addCleanup(context.stop)

    def stage(self, assets, profile='review'):
        output = io.StringIO()
        with contextlib.redirect_stdout(output):
            self.assertEqual(dev.stage_candidate([str(assets), '--profile', profile]), 0)
        return json.loads(output.getvalue())

    def test_stage_verifies_candidate_without_selecting_or_resetting_profile(self):
        fixtures.bundle(self.root / 'assets')
        desktop = dev.control.Desktop()
        desktop.create_profile('review')
        before = desktop.status()
        preference = Path(desktop.profile_paths('review')['XDG_STATE_HOME']) / 'retained.txt'
        preference.write_text('human review state')
        first = self.stage(self.root / 'assets')
        self.assertEqual(first['release'], self.stage(self.root / 'assets')['release'])
        after = desktop.status()
        self.assertEqual(before['selected'], after['selected'])
        self.assertEqual(before['profiles'], after['profiles'])
        self.assertEqual(preference.read_text(), 'human review state')
        self.assertIn('select review --release ' + first['release'], first['activate'])
        self.assertTrue(Path(first['directory']).is_dir())

    def test_new_profile_has_no_host_setup_and_remains_inactive(self):
        fixtures.bundle(self.root / 'assets')
        self.stage(self.root / 'assets')
        state = dev.control.Desktop().status()
        self.assertEqual(state['selected'], 'default')
        self.assertFalse(state['profiles']['review']['host_setup'])
        self.assertIsNone(state['profiles']['review']['active'])

    def test_default_and_unsafe_names_refused_before_state_creation(self):
        for name in ('default', '../review', 'Review'):
            with contextlib.redirect_stderr(io.StringIO()), self.assertRaises(SystemExit):
                self.stage(self.root / 'missing', name)
        self.assertFalse(Path(self.env['XDG_STATE_HOME']).exists())

    def test_host_setup_profile_and_tampered_archive_refused(self):
        _, archive, _ = fixtures.bundle(self.root / 'assets')
        desktop = dev.control.Desktop()
        desktop.create_profile('host', True)
        with self.assertRaisesRegex(ValueError, 'host setup disabled'):
            self.stage(self.root / 'assets', 'host')
        archive.write_bytes(archive.read_bytes() + b'tamper')
        with self.assertRaisesRegex(ValueError, 'differs'):
            self.stage(self.root / 'assets')
        self.assertNotIn('review', desktop.status()['profiles'])
        self.assertEqual(desktop.status()['selected'], 'default')

    def test_cached_environment_reuses_only_go_caches(self):
        packager = dev.load_packager()
        go = packager.resolve_go()
        stages = [self.root / name for name in ('one', 'two', 'uncached')]
        for stage in stages:
            stage.mkdir()
        cache = self.root / 'shared'
        one = dev.development_environment(stages[0], go, packager, cache)
        two = dev.development_environment(stages[1], go, packager, cache)
        fresh = dev.development_environment(stages[2], go, packager, None)
        for key in ('GOCACHE', 'GOMODCACHE'):
            self.assertEqual(one[key], two[key])
            self.assertNotEqual(one[key], fresh[key])
        self.assertNotEqual(one['HOME'], two['HOME'])
        self.assertEqual(one['GOENV'], 'off')
        self.assertEqual(one['GOWORK'], 'off')
        self.assertNotIn('GOCACHEPROG', one)
        self.assertEqual(cache.stat().st_mode & 0o777, 0o700)

    def test_shared_or_symlinked_cache_refused(self):
        for name, link in (('shared', False), ('link', True)):
            stage = self.root / (name + '-stage')
            stage.mkdir()
            cache = self.root / name
            if link:
                cache.symlink_to(self.root)
            else:
                cache.mkdir(mode=0o755)
            packager = dev.load_packager()
            with self.assertRaises(dev.control.runtime.BootstrapError):
                dev.development_environment(stage, packager.resolve_go(), packager, cache)


class LocalCheckTests(unittest.TestCase):
    def test_lanes_do_not_accidentally_build_or_operate_on_hosts(self):
        for lane in ('fast', 'desktop', 'tooling', 'qml'):
            for command in dev_checks.commands(lane):
                self.assertNotIn('--build', command)
                self.assertNotIn('docker', command)
                self.assertNotIn('ssh', command)
        self.assertTrue(all('--require-qt' in command for command in dev_checks.commands('qml')))

    def test_go_flags_are_separate_arguments_and_cannot_be_ignored(self):
        plan = dev_checks.commands('go', ['./internal/client'], 'TestGuided.*', True)
        self.assertIn('-race', plan[0])
        self.assertEqual(plan[0][-3:], ['-run', 'TestGuided.*', './internal/client'])
        for lane in ('fast', 'desktop', 'tooling', 'qml'):
            with self.assertRaises(ValueError):
                dev_checks.commands(lane, ['./internal/client'])
        with self.assertRaises(ValueError):
            dev_checks.commands('go', ['../../outside'])


if __name__ == '__main__':
    unittest.main(verbosity=2)
