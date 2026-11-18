import contextlib
import importlib.util
import io
from pathlib import Path
import unittest
from unittest.mock import patch


def load(name):
    spec = importlib.util.spec_from_file_location('issues_'+name, Path(__file__).with_name(name+'.py'))
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


class AdmissionTests(unittest.TestCase):
    def test_no_run_and_existing_capsule_cannot_reach_effects(self):
        host = load('run')
        with patch.object(host, 'charge', side_effect=AssertionError('storage/effect admission reached')), patch.object(host.subprocess,'check_output',side_effect=AssertionError('subprocess reached')):
            with patch('sys.argv',['run.py']), contextlib.redirect_stdout(io.StringIO()):
                host.main()
            with patch('sys.argv',['run.py','--run','--portable','/input','--previous','/input','--candidate','/input','--rootless','/input','--runtime','/input','--helper','/input','--output','/tmp']), contextlib.redirect_stderr(io.StringIO()):
                with self.assertRaises(SystemExit):
                    host.main()

    def test_guest_rejects_host_before_commands_or_filesystem_mutation(self):
        guest = load('guest')
        with patch.object(guest.os,'geteuid',return_value=1000), patch.object(guest.subprocess,'run',side_effect=AssertionError('guest effect reached')), patch.object(guest.Path,'mkdir',side_effect=AssertionError('guest state write reached')):
            with self.assertRaisesRegex(RuntimeError,'refused outside'):
                guest.main()


if __name__ == '__main__':
    unittest.main()
