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
    spec = importlib.util.spec_from_file_location('runtime_cow_'+name, Path(__file__).with_name(name+'.py'))
    value = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(value)
    return value


class RecoveryGuards(unittest.TestCase):
    def test_outer_deadlines_leave_time_for_inner_failure_and_cleanup(self):
        host, guest = module('run'), module('guest')
        self.assertGreater(guest.HELPER_PROCESS_TIMEOUT_SECONDS, 90*60)
        self.assertGreater(guest.HELPER_DEADLINE_SECONDS, guest.HELPER_PROCESS_TIMEOUT_SECONDS)
        self.assertGreater(host.VM_DEADLINE_SECONDS, guest.HELPER_DEADLINE_SECONDS+120)
        self.assertGreater(host.SUPERVISOR_DEADLINE_SECONDS, host.VM_DEADLINE_SECONDS+120)

    def test_no_run_has_no_source_storage_or_process_effect(self):
        host = module('run')
        with patch.object(sys, 'argv', ['run.py']), patch.object(host.subprocess, 'check_output', side_effect=AssertionError('no process')), patch.object(host.budget, 'charge', side_effect=AssertionError('no admission')), contextlib.redirect_stdout(io.StringIO()):
            host.main()

    def test_existing_capsule_is_not_an_output(self):
        host = module('run')
        args = ['run.py', '--run', '--output', '/tmp/omabit-issues-app-vm-05']
        for key in ('portable', 'candidate', 'rootless', 'runtime', 'helper'):
            args += ['--'+key, '/missing']
        with patch.object(sys, 'argv', args), patch.object(host.os, 'geteuid', return_value=1000), patch.object(host.subprocess, 'check_output', side_effect=AssertionError('refuse before inputs')), contextlib.redirect_stderr(io.StringIO()):
            with self.assertRaises(SystemExit) as stopped:
                host.main()
        self.assertEqual(stopped.exception.code, 2)

    def test_guest_refuses_host_wrong_predecessor_case_and_schema(self):
        guest = module('guest')
        with patch.object(guest.os, 'geteuid', return_value=1000), patch.object(Path, 'read_text', side_effect=AssertionError('host read forbidden')):
            with self.assertRaises(RuntimeError):
                guest.guard()
        scenario = {'kind': 'issues18-product-grubbery-runtime-v1', 'case': guest.CASE, 'predecessor_root_sha256': guest.PIN, 'original_schema': 62, 'target_schema': 63, 'development_snapshot': True}
        values = {'/sys/class/dmi/id/sys_vendor': 'QEMU', '/proc/cmdline': 'omabit.disposable-guest=release-upgrade-v1', '/opt/omabit-release-input/scenario.json': json.dumps(scenario)}
        with patch.object(guest.os, 'geteuid', return_value=0), patch.object(sys, 'argv', ['guest.py']), patch.object(Path, 'read_text', lambda path, **kwargs: values[str(path)]):
            self.assertEqual(guest.guard(), scenario)
            for change in ({'predecessor_root_sha256': '0'*64}, {'case': 'cow-fresh-nec-v1'}, {'original_schema': 63}, {'target_schema': 62}, {'development_snapshot': False}):
                values['/opt/omabit-release-input/scenario.json'] = json.dumps({**scenario, **change})
                with self.assertRaises(RuntimeError):
                    guest.guard()

    def test_compressed_input_refuses_wrong_bytes_in_fresh_private_output(self):
        import gzip
        guest = module('guest')
        with tempfile.TemporaryDirectory() as root:
            path = Path(root)
            (path/'input/candidate').mkdir(parents=True)
            (path/'input/candidate/omabit.gz').write_bytes(gzip.compress(b'wrong'))
            with patch.object(guest, 'I', path/'input'), patch.object(guest, 'R', path/'runtime'):
                with self.assertRaises(RuntimeError):
                    guest.inflate({'candidate': {'omabit': {'bytes': 5, 'sha256': '0'*64}}})
                self.assertEqual((path/'runtime/candidate/omabit').read_bytes(), b'wrong')

    def test_actual_kernel_file_limit_refuses_growth_and_preserves_first_bytes(self):
        host = module('run')
        self.assertLess(host.ROOT_FILE_MAX+64*1024**2+32*1024**2+16*1024**2, host.SCRATCH)
        self.assertEqual(host.SCRATCH, 1024*1024**2)
        with tempfile.TemporaryDirectory() as root:
            target = Path(root)/'bounded-disposable-file'
            code = "import errno,pathlib,sys; p=pathlib.Path(sys.argv[1]); f=p.open('xb',buffering=0); f.write(b'A'*1048576)\ntry: f.write(b'B')\nexcept OSError as e: assert e.errno==errno.EFBIG\nelse: raise RuntimeError('kernel file bound was ignored')\nf.close(); assert p.stat().st_size==1048576 and p.read_bytes()==b'A'*1048576"
            result = subprocess.run(['/usr/bin/prlimit', '--fsize=1048576:1048576', '--', sys.executable, '-B', '-c', code, str(target)], capture_output=True, text=True, timeout=10)
            self.assertEqual(result.returncode, 0, result.stderr)


if __name__ == '__main__':
    unittest.main()
