import contextlib
import importlib.util
import io
from pathlib import Path
import sys
import unittest
from unittest.mock import patch


def host():
    path = Path(__file__).resolve().parents[2]/'test-recording-store-upgrade.py'
    spec = importlib.util.spec_from_file_location('recording_store_upgrade', path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


class RecordingStoreSourceGuards(unittest.TestCase):
    def test_without_run_never_queries_or_creates(self):
        module = host()
        with patch.object(sys, 'argv', ['test-recording-store-upgrade.py']), patch.object(module.subprocess, 'check_output', side_effect=AssertionError('no source query')), patch.object(Path, 'mkdir', side_effect=AssertionError('no output')), contextlib.redirect_stdout(io.StringIO()):
            module.main()

    def test_other_capsule_refuses_before_git_storage_or_creation(self):
        module = host()
        with patch.object(sys, 'argv', ['test-recording-store-upgrade.py', '--run', '--output', '/tmp/omabit-issues-app-vm-05']), patch.object(module.subprocess, 'check_output', side_effect=AssertionError('no source query')), patch.object(Path, 'mkdir', side_effect=AssertionError('no output')), contextlib.redirect_stderr(io.StringIO()):
            with self.assertRaises(SystemExit) as error:
                module.main()
        self.assertEqual(error.exception.code, 2)

    def test_dirty_source_refuses_before_storage_or_creation(self):
        module = host()
        with patch.object(sys, 'argv', ['test-recording-store-upgrade.py', '--run', '--output', '/tmp/omabit-recording-store-upgrade-99']), patch.object(Path, 'exists', return_value=False), patch.object(Path, 'is_symlink', return_value=False), patch.object(module.subprocess, 'check_output', return_value=b' M changed'), patch.object(Path, 'mkdir', side_effect=AssertionError('no output')):
            with self.assertRaisesRegex(RuntimeError, 'clean candidate'):
                module.main()


if __name__ == '__main__':
    unittest.main()
