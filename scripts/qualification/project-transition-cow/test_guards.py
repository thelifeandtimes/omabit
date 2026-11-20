import contextlib
import importlib.util
import io
import json
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch


def module(name):
    spec = importlib.util.spec_from_file_location('project_transition_'+name, Path(__file__).with_name(name+'.py'))
    value = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(value)
    return value


class TransitionGuards(unittest.TestCase):
    def test_no_run_has_no_effect(self):
        host = module('run')
        with patch.object(sys, 'argv', ['run.py']), patch.object(host.subprocess, 'check_output', side_effect=AssertionError('no process')), patch.object(host.budget, 'charge', side_effect=AssertionError('no admission')), contextlib.redirect_stdout(io.StringIO()):
            host.main()

    def test_existing_or_wrong_capsule_refuses_before_inputs(self):
        host = module('run')
        args = ['run.py', '--run', '--output', '/tmp/omabit-issues-runtime-cow-05']
        for key in ('portable', 'candidate', 'rootless', 'runtime', 'helper', 'init'):
            args += ['--'+key, '/missing']
        with patch.object(sys, 'argv', args), patch.object(host.os, 'geteuid', return_value=1000), patch.object(host.subprocess, 'check_output', side_effect=AssertionError('refuse before reading inputs')), contextlib.redirect_stderr(io.StringIO()):
            with self.assertRaises(SystemExit):
                host.main()

    def test_normal_products_omit_tag_and_test_manager_requires_it(self):
        host = module('run')
        normal = '\tbuild\tCGO_ENABLED=0\n'
        tagged = normal+'\tbuild\t-tags=omabit_disposable_qualification\n'
        for name in ('omabit', 'urbit-collector', 'urbit-runner'):
            host.validate_build_info(name, normal)
            with self.assertRaises(RuntimeError):
                host.validate_build_info(name, tagged)
        host.validate_build_info('managed.test', tagged)
        with self.assertRaises(RuntimeError):
            host.validate_build_info('managed.test', normal)

    def test_embedded_and_runtime_changes_refuse(self):
        host = module('run')
        for path in ('go.mod', 'internal/service/legacy-inventory.json', 'contents/ui/Panel.qml', 'docs/example.go', 'docs/../internal/x', '/docs/x', 'internal/managed/server.go'):
            self.assertFalse(host.artifact_change_allowed(path))
        for path in ('docs/development-memory.md', host.BOOT_SOURCE_PATH, 'scripts/qualification/project-transition-cow/status.py'):
            self.assertTrue(host.artifact_change_allowed(path))

    def test_source_and_binary_boot_pin_required_before_probe(self):
        host = module('run')
        with tempfile.TemporaryDirectory() as d:
            init = Path(d)/'init'; init.write_bytes(b'inert')
            with patch.object(host.subprocess, 'check_output', side_effect=AssertionError('pins first')):
                with self.assertRaisesRegex(RuntimeError, 'boot helper or source differs'):
                    host.frozen_boot(init)

    def test_exact_three_level_backing_chain_no_self_cycle(self):
        host = module('run')
        # The successful app recovery fixture is an input, not a prior run of
        # this new transition capsule. Preserve its exact provenance binding.
        self.assertEqual(host.PREDECESSOR, Path('/tmp/omabit-issues-runtime-cow-05/root.qcow2'))
        self.assertEqual(host.PREDECESSOR_SHA, '0c28d526285e5b4e055e209e9021e5c683505ddabb5f0ce661678052949d141c')
        good = [{'format': 'qcow2', 'backing-filename': x, 'backing-filename-format': 'qcow2', 'dirty-flag': False} for x in ('/predecessor.qcow2', '/base.qcow2')]
        with patch.object(host.subprocess, 'check_output', side_effect=[json.dumps(x) for x in good]):
            host.verify_chain(Path('/tools'), {})
        for bad in ({**good[0], 'backing-filename': '/current-app.qcow2'}, {**good[0], 'dirty-flag': True}, {**good[0], 'backing-filename-format': 'raw'}):
            with patch.object(host.subprocess, 'check_output', return_value=json.dumps(bad)):
                with self.assertRaises(RuntimeError):
                    host.verify_chain(Path('/tools'), {})
        source = Path(__file__).with_name('run.py').read_text()
        self.assertIn("'-b', '/current-app.qcow2'", source)
        self.assertIn("str(ORIGINAL), '/predecessor.qcow2'", source)
        self.assertIn("str(PREDECESSOR), '/current-app.qcow2'", source)

    def test_guest_refuses_host_and_foreign_scenario_before_state(self):
        guest = module('guest')
        with patch.object(guest.os, 'geteuid', return_value=1000), patch.object(Path, 'read_text', side_effect=AssertionError('no host reads')):
            with self.assertRaises(RuntimeError):
                guest.guard()
        scenario = {'kind': 'issues16-native-transition-v1', 'case': guest.CASE, 'predecessor_root_sha256': guest.PIN, 'original_schema': 65, 'target_schema': 65, 'development_snapshot': True, 'require_native_transition': True, 'preserve_stopped_base_instance': True}
        values = {'/sys/class/dmi/id/sys_vendor': 'QEMU', '/proc/cmdline': 'omabit.disposable-guest=release-upgrade-v1', '/opt/omabit-release-input/scenario.json': json.dumps(scenario)}
        with patch.object(guest.os, 'geteuid', return_value=0), patch.object(sys, 'argv', ['guest.py']), patch.object(Path, 'read_text', lambda p, **k: values[str(p)]):
            self.assertEqual(guest.guard(), scenario)
            for change in ({'require_native_transition': False}, {'preserve_stopped_base_instance': False}, {'predecessor_root_sha256': '0'*64}, {'original_schema': 62}, {'case': 'other'}):
                values['/opt/omabit-release-input/scenario.json'] = json.dumps({**scenario, **change})
                with self.assertRaises(RuntimeError):
                    guest.guard()

    def test_input_inflation_refuses_size_and_hash(self):
        import gzip
        guest = module('guest')
        for content in (b'bad', b'wrong-too-long'):
            with tempfile.TemporaryDirectory() as d:
                root = Path(d); (root/'input/candidate').mkdir(parents=True)
                candidates = {name: {'bytes': 3, 'sha256': '0'*64} for name in ('omabit', 'urbit-collector', 'urbit-runner', 'managed.test')}
                (root/'input/candidate/omabit.gz').write_bytes(gzip.compress(content))
                with patch.object(guest, 'I', root/'input'), patch.object(guest, 'R', root/'runtime'):
                    with self.assertRaises(RuntimeError):
                        guest.inflate({'candidate': candidates})

    def test_whole_status_bounded_private_input_free(self):
        status = module('status')
        report = {k: True for k in status.GATES}
        report.update(passed=True, source_commit='a'*40, proof_sha256='b'*64, inputs={'private': 'PRIVATE'*1000000}, error='PRIVATE'*1000000)
        raw = status.serialize(report)
        self.assertLess(len((status.MARKER+raw+'\n').encode()), 4096)
        self.assertNotIn('PRIVATE', raw)
        self.assertTrue(status.read_log('prefix '+status.MARKER+raw, 'a'*40)['passed'])
        for log in (status.MARKER+raw+'\n'+status.MARKER+raw, status.MARKER+raw[:80], status.MARKER+raw.replace('a'*40, 'c'*40), status.MARKER+raw[:-1]+',"extra":true}'):
            with self.assertRaises((RuntimeError, ValueError)):
                status.read_log(log, 'a'*40)
        report['single_sigterm'] = False
        self.assertFalse(status.read_log(status.MARKER+status.serialize(report), 'a'*40)['passed'])

    def test_resource_and_deadline_bounds(self):
        host = module('run')
        self.assertLessEqual(host.ROOT_FILE_MAX+64*1024**2+32*1024**2+8*1024**2, host.SCRATCH)
        self.assertGreater(host.SUPERVISOR_DEADLINE_SECONDS, host.VM_DEADLINE_SECONDS+120)
        source = (Path(__file__).parent/'init/main.go').read_text()
        self.assertIn('TimeoutStartSec=6000', source)
        self.assertGreater(host.VM_DEADLINE_SECONDS, 6000+30+60)
        self.assertNotIn('--resume-running', source)

    def test_root_ceiling_detects_live_qemu_io_failure(self):
        host = module('run')
        with tempfile.TemporaryDirectory() as d, patch.object(host, 'ROOT_FILE_MAX', 128):
            root = Path(d)/'root.qcow2'
            root.write_bytes(b'x'*127)
            self.assertFalse(host.root_limit_reached(root))
            with root.open('ab') as stream:
                stream.write(b'x')
            self.assertTrue(host.root_limit_reached(root))
            with root.open('ab') as stream:
                stream.write(b'x')
            self.assertTrue(host.root_limit_reached(root))


if __name__ == '__main__':
    unittest.main()
