import contextlib
import importlib.util
import io
import json
from pathlib import Path
import sys
import unittest
from unittest.mock import patch


def module(name):
    spec = importlib.util.spec_from_file_location('product_'+name, Path(__file__).with_name(name+'.py'))
    value = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(value)
    return value


class ProductGuards(unittest.TestCase):
    def test_no_run_has_no_io_or_process_effect(self):
        host = module('run')
        with patch.object(sys, 'argv', ['run.py']), patch.object(host.subprocess, 'check_output', side_effect=AssertionError('no process permitted')), patch.object(host.budget, 'charge', side_effect=AssertionError('no storage admission permitted')), contextlib.redirect_stdout(io.StringIO()):
            host.main()

    def test_bad_output_refuses_before_input_or_process_access(self):
        host = module('run')
        args = ['run.py', '--run', '--output', '/tmp/not-an-omabit-product-capsule']
        for key in ('portable', 'candidate', 'rootless', 'runtime', 'framework-source', 'helper'):
            args += ['--'+key, '/missing-fixture']
        with patch.object(sys, 'argv', args), patch.object(host.os, 'geteuid', return_value=1000), patch.object(host.subprocess, 'check_output', side_effect=AssertionError('input/process access before output guard')), contextlib.redirect_stderr(io.StringIO()):
            with self.assertRaises(SystemExit) as stopped:
                host.main()
        self.assertEqual(stopped.exception.code, 2)

    def test_guest_guard_requires_root_and_exact_fresh_marker(self):
        guest = module('guest')
        with patch.object(guest.os, 'geteuid', return_value=1000), patch.object(Path, 'read_text', side_effect=AssertionError('host reads after owner refusal')):
            with self.assertRaises(RuntimeError):
                guest.guard()
        scenario = {'kind': 'issues18-product-grubbery-v1', 'development_snapshot': True, 'target_schema': 62}
        values = {'/sys/class/dmi/id/sys_vendor': 'QEMU\n', '/proc/cmdline': 'quiet omabit.disposable-guest=release-upgrade-v1\n', '/opt/omabit-release-input/scenario.json': json.dumps(scenario)}
        with patch.object(guest.os, 'geteuid', return_value=0), patch.object(sys, 'argv', ['guest.py']), patch.object(Path, 'read_text', lambda p, **kwargs: values[str(p)]):
            self.assertEqual(guest.guard(), scenario)
            values['/sys/class/dmi/id/sys_vendor'] = 'Physical host'
            with self.assertRaises(RuntimeError):
                guest.guard()

    def test_wrong_scenario_cannot_select_other_schema_or_workflow(self):
        guest = module('guest')
        for change in ({'kind': 'issues16-21-development-v1'}, {'target_schema': 60}, {'development_snapshot': False}):
            scenario = {'kind': 'issues18-product-grubbery-v1', 'development_snapshot': True, 'target_schema': 62, **change}
            values = {'/sys/class/dmi/id/sys_vendor': 'QEMU\n', '/proc/cmdline': 'omabit.disposable-guest=release-upgrade-v1', '/opt/omabit-release-input/scenario.json': json.dumps(scenario)}
            with patch.object(guest.os, 'geteuid', return_value=0), patch.object(sys, 'argv', ['guest.py']), patch.object(Path, 'read_text', lambda p, **kwargs: values[str(p)]):
                with self.assertRaises(RuntimeError):
                    guest.guard()


if __name__ == '__main__':
    unittest.main()
