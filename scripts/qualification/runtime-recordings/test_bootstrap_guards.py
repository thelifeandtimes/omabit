import contextlib
import importlib.util
import io
from pathlib import Path
import sys
import unittest
from unittest.mock import patch


def host():
    path = Path(__file__).resolve().parents[2]/'test-recording-bootstrap-upgrade.py'
    spec = importlib.util.spec_from_file_location('recording_bootstrap_upgrade', path)
    value = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(value)
    return value


class BootstrapSourceGuards(unittest.TestCase):
    def test_without_run_never_queries_source_or_creates_output(self):
        module = host()
        with patch.object(sys, 'argv', ['test-recording-bootstrap-upgrade.py']), patch.object(module.subprocess, 'check_output', side_effect=AssertionError('no source query')), patch.object(Path, 'mkdir', side_effect=AssertionError('no output')), contextlib.redirect_stdout(io.StringIO()):
            module.main()

    def test_other_capsule_is_refused_before_git_or_storage(self):
        module = host()
        with patch.object(sys, 'argv', ['test-recording-bootstrap-upgrade.py', '--run', '--output', '/tmp/omabit-issues-app-vm-05']), patch.object(module.subprocess, 'check_output', side_effect=AssertionError('no source query')), patch.object(Path, 'mkdir', side_effect=AssertionError('no output')), contextlib.redirect_stderr(io.StringIO()):
            with self.assertRaises(SystemExit) as error:
                module.main()
        self.assertEqual(error.exception.code, 2)

    def test_original_snapshot_includes_embedded_and_declared_fixture_inputs(self):
        module = host()
        self.assertEqual(module.ORIGINAL, '6e810bc937e57a9f5ff521c8957502fdceb71bb5')
        for path in ('go.mod', 'go.sum', 'internal/recording/metadata_test.go', 'internal/recording/identity_test.go', 'internal/recording/segments_linux.go', 'internal/service/legacy-inventory.json', 'internal/appdev/samples/grubbery-state-v2/nex/state.hoon', 'internal/rootless/assets/example.service', 'internal/runtimequalify/private_gateway_probe.cjs'):
            self.assertTrue(module.selected(path), path)
        for path in ('internal/recording/capture_linux_test.go', 'docs/private.md', 'build/issues-runtime-cow-03/managed.test'):
            self.assertFalse(module.selected(path), path)


if __name__ == '__main__':
    unittest.main()
