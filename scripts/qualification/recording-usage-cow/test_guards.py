import configparser
import copy
import contextlib
import gzip
import importlib.util
import io
import json
from pathlib import Path
import re
import sqlite3
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch


def module(name):
    spec = importlib.util.spec_from_file_location('usage_cow_'+name, Path(__file__).with_name(name+'.py'))
    value = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(value)
    return value


class UsageGuards(unittest.TestCase):
    def test_only_one_fresh_observation_can_differ_from_original_rows(self):
        guest = module('guest')
        state = {'public': {'id': 'recording-zod', 'ship': '~zod', 'desired_state': 'stopped', 'observed_at': '2026-11-19T10:00:00.000000001Z'}, 'runtime_revision': 75, 'runtime': {'storage': {'fixture': 'same-pier'}, 'initialized': True}}
        before = {'instances': [('recording-zod', json.dumps(state))], 'history': [('original', b'unchanged')]}
        fresh = copy.deepcopy(state); fresh['public']['observed_at'] = '2026-11-19T11:00:00.000000001Z'; fresh['runtime_revision'] = 76
        after = {'instances': [('recording-zod', json.dumps(fresh))], 'history': [('original', b'unchanged')]}
        original = copy.deepcopy(before)
        self.assertNotEqual(before, after)  # reproduces the first whole-case assertion failure
        self.assertTrue(guest.verify_preserved_rows(before, after)['fresh_observation_only'])
        self.assertEqual(before, original)
        for change in ('intent', 'storage', 'missing-field', 'revision', 'stale-time', 'history', 'extra-row', 'extra-table'):
            with self.subTest(change=change):
                changed = copy.deepcopy(after); body = copy.deepcopy(fresh)
                if change == 'intent': body['public']['desired_state'] = 'running'
                elif change == 'storage': body['runtime']['storage'] = {'fixture': 'other-pier'}
                elif change == 'missing-field': del body['runtime']['initialized']
                elif change == 'revision': body['runtime_revision'] = 77
                elif change == 'stale-time': body['public']['observed_at'] = state['public']['observed_at']
                elif change == 'history': changed['history'] = [('original', b'changed')]
                elif change == 'extra-row': changed['history'].append(('new', b'not-allowed'))
                elif change == 'extra-table': changed['new-table'] = []
                changed['instances'] = [('recording-zod', json.dumps(body))]
                with self.assertRaises(RuntimeError):
                    guest.verify_preserved_rows(before, changed)

    def test_no_run_has_no_source_storage_or_process_effect(self):
        host = module('run')
        with patch.object(sys, 'argv', ['run.py']), patch.object(host.subprocess, 'check_output', side_effect=AssertionError('no process')), patch.object(host.budget, 'charge', side_effect=AssertionError('no admission')), contextlib.redirect_stdout(io.StringIO()):
            host.main()

    def test_only_fresh_specific_output_is_accepted(self):
        host = module('run')
        args = ['run.py', '--run', '--output', '/tmp/omabit-issues-recording-vm-07']
        for name in ('portable', 'candidate', 'rootless', 'runtime', 'helper', 'init'):
            args += ['--'+name, '/missing']
        with patch.object(sys, 'argv', args), patch.object(host.os, 'geteuid', return_value=1000), patch.object(host.subprocess, 'check_output', side_effect=AssertionError('no source read')), contextlib.redirect_stderr(io.StringIO()):
            with self.assertRaises(SystemExit) as result:
                host.main()
            self.assertEqual(result.exception.code, 2)

    def test_product_or_embedded_changes_are_not_harness_exceptions(self):
        host = module('run')
        for path in ('internal/service/legacy-inventory.json', 'internal/managed/issues_recording_usage_vm_test.go', 'internal/runtimequalify/recording_usage_linux.go', 'go.mod', 'go.sum', 'plugins/omabit/Recordings.js', 'docs/example.go', host.BOOT_SOURCE_PATH+'/../other.go'):
            self.assertFalse(host.artifact_change_allowed(path), path)
        for path in ('docs/diagnostics/result.json', host.BOOT_SOURCE_PATH, 'scripts/qualification/recording-usage-cow/guest.py'):
            self.assertTrue(host.artifact_change_allowed(path), path)

    def test_wrong_boot_bytes_refuse_before_provenance_or_admission(self):
        host = module('run')
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory)/'init'; path.write_bytes(b'unqualified')
            with patch.object(host.subprocess, 'check_output', side_effect=AssertionError('must refuse first')):
                with self.assertRaisesRegex(RuntimeError, 'boot helper or source differs'):
                    host.frozen_boot(path)

    def test_actual_unit_deadline_and_hard_scratch_bounds(self):
        host, guest = module('run'), module('guest')
        source = (Path(__file__).parent/'init/main.go').read_text()
        config = configparser.ConfigParser(interpolation=None)
        config.read_string(re.search(r'service := `([^`]+)`', source).group(1))
        self.assertEqual(config['Service']['Type'], 'oneshot')
        self.assertGreater(guest.HELPER_SECONDS, 120)
        self.assertGreater(guest.HELPER_PROCESS_SECONDS, guest.HELPER_SECONDS)
        self.assertGreater(config.getint('Service', 'TimeoutStartSec'), guest.HELPER_PROCESS_SECONDS+60)
        self.assertGreater(host.VM_DEADLINE_SECONDS, config.getint('Service', 'TimeoutStartSec')+config.getint('Service', 'TimeoutStopSec')+60)
        self.assertGreater(host.SUPERVISOR_DEADLINE_SECONDS, host.VM_DEADLINE_SECONDS+120)
        self.assertLess(host.ROOT_FILE_MAX+64*1024**2+32*1024**2+16*1024**2, host.SCRATCH)
        self.assertEqual(host.SCRATCH+host.HOST_BUILD_RESERVE, 320*1024**2)

    def test_candidate_override_sorts_after_original_without_removing_it(self):
        source = (Path(__file__).parent/'init/main.go').read_text()
        path = re.search(r'os.WriteFile\("(/newroot/etc/systemd/user/urbit-runner.service.d/[^"/]+)"', source).group(1)
        self.assertGreater(Path(path).name, 'recording-pilot.conf')
        override = re.search(r'override := `([^`]+)`', source).group(1)
        self.assertIn('ConditionPathExists=/run/omabit-runtime-runner-ready', override)
        self.assertIn('ExecStart=\nExecStart=/usr/bin/env -i', override)
        self.assertIn('/run/omabit-runtime-candidate/candidate/urbit-runner', override)
        self.assertIn('--record-runtime', override)
        self.assertNotIn('--resume-running', override)
        self.assertNotIn('os.Remove', source)

    def test_guest_refuses_host_and_wrong_original_schema_or_case(self):
        guest = module('guest')
        with patch.object(guest.os, 'geteuid', return_value=1000), patch.object(Path, 'read_text', side_effect=AssertionError('host read forbidden')):
            with self.assertRaises(RuntimeError):
                guest.guard()
        scenario = {'kind': 'issues20-recording-usage-v1', 'case': guest.CASE, 'predecessor_root_sha256': guest.PIN, 'original_schema': 65, 'target_schema': 65, 'development_snapshot': True, 'require_original_exports': True, 'require_derived_exports': True}
        values = {'/sys/class/dmi/id/sys_vendor': 'QEMU', '/proc/cmdline': 'omabit.disposable-guest=release-upgrade-v1', '/opt/omabit-release-input/scenario.json': json.dumps(scenario)}
        with patch.object(guest.os, 'geteuid', return_value=0), patch.object(sys, 'argv', ['guest.py']), patch.object(Path, 'read_text', lambda path, **kwargs: values[str(path)]):
            self.assertEqual(guest.guard(), scenario)
            for change in ({'case': 'other'}, {'predecessor_root_sha256': '0'*64}, {'original_schema': 64}, {'target_schema': 64}, {'development_snapshot': False}, {'require_original_exports': False}, {'require_derived_exports': False}):
                values['/opt/omabit-release-input/scenario.json'] = json.dumps({**scenario, **change})
                with self.assertRaises(RuntimeError):
                    guest.guard()

    def test_large_private_proof_never_enters_serial_status(self):
        guest = module('guest')
        proof = {'passed': True, **{key: True for key in guest.PROOF_GATES}, 'original_exports': [{'private_archive': 'x'*131072}, {'private_query': 'y'*131072}]}
        summary = guest.product_summary(proof)
        self.assertLess(len(json.dumps(summary)), 2048)
        self.assertNotIn('original_exports', summary)
        self.assertEqual(summary['closed_jobs'], 2)
        self.assertTrue(summary['complete_private_proof_retained'])
        self.assertEqual(len(proof['original_exports'][0]['private_archive']), 131072)
        for key in ('passed', *guest.PROOF_GATES):
            with self.assertRaises(RuntimeError):
                guest.product_summary({**proof, key: False})
        for value in (None, [], [{}], [{}, {}, {}]):
            with self.assertRaises(RuntimeError):
                guest.product_summary({**proof, 'original_exports': value})

    def test_sqlite_readers_close_and_preserve_rows_on_refusal(self):
        guest = module('guest'); connect = sqlite3.connect
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory)/'fixture.sqlite'
            with contextlib.closing(connect(path)) as db:
                db.execute('pragma user_version=65')
                db.execute('create table instances (body text)')
                db.execute('insert into instances values (?)', (json.dumps({'public': {'id': 'recording-zod', 'ship': '~zod', 'desired_state': 'stopped'}}),))
                db.commit()
            readers = []
            def readonly(filename, **kwargs):
                self.assertEqual(filename, 'file:/home/review/.local/share/urbit-runner/managed/runner.sqlite?mode=ro')
                self.assertEqual(kwargs, {'uri': True})
                db = connect('file:'+str(path)+'?mode=ro', uri=True); readers.append(db); self.addCleanup(db.close); return db
            with patch.object(guest.sqlite3, 'connect', side_effect=readonly):
                self.assertEqual(list(guest.saved_rows()), ['instances'])
                with contextlib.closing(connect(path)) as db:
                    db.execute('pragma user_version=64')
                with self.assertRaisesRegex(RuntimeError, 'wrong original/candidate schema'):
                    guest.saved_rows()
            for db in readers:
                with self.assertRaises(sqlite3.ProgrammingError):
                    db.execute('select 1')

    def test_inflation_is_bounded_and_hash_verified(self):
        guest = module('guest')
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory); (root/'input/candidate').mkdir(parents=True)
            (root/'input/candidate/omabit.gz').write_bytes(gzip.compress(b'wrong'))
            with patch.object(guest, 'I', root/'input'), patch.object(guest, 'R', root/'runtime'):
                with self.assertRaisesRegex(RuntimeError, 'executable differs'):
                    guest.inflate({'candidate': {'omabit': {'bytes': 5, 'sha256': '0'*64}}})

    def test_actual_kernel_file_limit(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory)/'bounded-output'
            code = "import errno,pathlib,sys; p=pathlib.Path(sys.argv[1]); f=p.open('xb',buffering=0); f.write(b'A'*1048576)\ntry: f.write(b'B')\nexcept OSError as e: assert e.errno==errno.EFBIG\nelse: raise RuntimeError('file bound ignored')\nf.close(); assert p.read_bytes()==b'A'*1048576"
            result = subprocess.run(['/usr/bin/prlimit', '--fsize=1048576:1048576', '--', sys.executable, '-B', '-c', code, str(path)], capture_output=True, text=True, timeout=10)
            self.assertEqual(result.returncode, 0, result.stderr)


if __name__ == '__main__':
    unittest.main()
