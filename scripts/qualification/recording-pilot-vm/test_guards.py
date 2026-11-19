import contextlib
import base64
import hashlib
import importlib.util
import io
import json
from pathlib import Path
import sys
import unittest
import tempfile
from unittest.mock import patch


def module(name):
    spec = importlib.util.spec_from_file_location('product_'+name, Path(__file__).with_name(name+'.py'))
    value = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(value)
    return value


class RecordingGuards(unittest.TestCase):
    def test_no_run_has_no_io_or_process_effect(self):
        host = module('run')
        with patch.object(sys, 'argv', ['run.py']), patch.object(host.subprocess, 'check_output', side_effect=AssertionError('no process permitted')), patch.object(host.budget, 'charge', side_effect=AssertionError('no storage admission permitted')), contextlib.redirect_stdout(io.StringIO()):
            host.main()

    def test_bad_output_refuses_before_input_or_process_access(self):
        host = module('run')
        args = ['run.py', '--run', '--output', '/tmp/not-an-omabit-product-capsule']
        for key in ('portable', 'candidate', 'predecessor', 'init', 'rootless', 'runtime', 'helper'):
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
        scenario = {'kind': 'issues20-recording-policy-upgrade-v1', 'development_snapshot': True, 'target_schema': 65}
        values = {'/sys/class/dmi/id/sys_vendor': 'QEMU\n', '/proc/cmdline': 'quiet omabit.disposable-guest=release-upgrade-v1\n', '/opt/omabit-release-input/scenario.json': json.dumps(scenario)}
        with patch.object(guest.os, 'geteuid', return_value=0), patch.object(sys, 'argv', ['guest.py']), patch.object(Path, 'read_text', lambda p, **kwargs: values[str(p)]):
            self.assertEqual(guest.guard(), scenario)
            values['/sys/class/dmi/id/sys_vendor'] = 'Physical host'
            with self.assertRaises(RuntimeError):
                guest.guard()

    def test_only_exact_fixture_override_can_be_withdrawn_and_restored(self):
        guest=module('guest')
        with tempfile.TemporaryDirectory(prefix='omabit-recording-override-')as root:
            home=Path(root);directory=home/'.config/systemd/user/urbit-runner.service.d';directory.mkdir(mode=0o700,parents=True)
            file=directory/'recording-pilot.conf';file.write_text('exact fixture')
            with patch.object(guest,'HOME',home):
                foreign=directory/'other.conf';foreign.write_text('preserve user setting')
                with self.assertRaises(RuntimeError):guest.withdraw_fixture_override(directory,'exact fixture')
                self.assertEqual(foreign.read_text(),'preserve user setting');self.assertEqual(file.read_text(),'exact fixture')
                foreign.unlink()
                with self.assertRaises(RuntimeError):guest.withdraw_fixture_override(directory,'different fixture')
                self.assertTrue(file.exists())
                mode=guest.withdraw_fixture_override(directory,'exact fixture')
                self.assertFalse(directory.exists())
                guest.restore_fixture_override(directory,'exact fixture',mode)
                self.assertEqual(file.read_text(),'exact fixture');self.assertEqual(file.stat().st_mode&0o777,mode)
                with self.assertRaises(FileExistsError):guest.restore_fixture_override(directory,'replacement',mode)
                self.assertEqual(file.read_text(),'exact fixture')

    def test_api_prefix_checks_actual_original_export_shape(self):
        guest=module('guest')
        payload=b'qualified stdout\n'
        row={'status':{'job_id':'1'*32,'job_sha256':'2'*64,'session_id':'3'*32},'stdout_bytes':len(payload),'stderr_bytes':0,'stdout_sha256':hashlib.sha256(payload).hexdigest(),'stderr_sha256':hashlib.sha256(b'').hexdigest()}
        page={'complete_export':False,'status':{'session_id':'3'*32},'original':[{'frame':{'Kind':1},'original_bytes_base64':base64.b64encode(payload).decode()},{'frame':{'Kind':5},'original_bytes_base64':base64.b64encode(b'{}').decode()}],'tail':'snapshot-eof'}
        with patch.object(guest,'core',return_value=page):
            guest.verify_api_prefix(row)
            row['stdout_bytes']+=1
            with self.assertRaises(RuntimeError):guest.verify_api_prefix(row)

    def test_wrong_scenario_cannot_select_other_schema_or_workflow(self):
        guest = module('guest')
        for change in ({'kind': 'issues16-21-development-v1'}, {'target_schema': 60}, {'target_schema':64}, {'development_snapshot': False}):
            scenario = {'kind': 'issues20-recording-policy-upgrade-v1', 'development_snapshot': True, 'target_schema': 65, **change}
            values = {'/sys/class/dmi/id/sys_vendor': 'QEMU\n', '/proc/cmdline': 'omabit.disposable-guest=release-upgrade-v1', '/opt/omabit-release-input/scenario.json': json.dumps(scenario)}
            with patch.object(guest.os, 'geteuid', return_value=0), patch.object(sys, 'argv', ['guest.py']), patch.object(Path, 'read_text', lambda p, **kwargs: values[str(p)]):
                with self.assertRaises(RuntimeError):
                    guest.guard()


if __name__ == '__main__':
    unittest.main()
