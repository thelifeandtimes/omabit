import importlib.util
from pathlib import Path
from types import SimpleNamespace
import unittest
from unittest.mock import patch
import uuid

spec = importlib.util.spec_from_file_location('recording_probe', Path(__file__).with_name('probe.py'))
probe = importlib.util.module_from_spec(spec); spec.loader.exec_module(probe)

class RecordingAdmissionTests(unittest.TestCase):
    def test_budget_refusal_precedes_copy_scope_and_output_creation(self):
        output=Path('/tmp')/('omabit-recording-probe-test-'+uuid.uuid4().hex)
        args=SimpleNamespace(output=output, storage_budget_gib=48)
        import sys
        sys.path.insert(0,str(Path(__file__).resolve().parents[1]/'app-development'))
        with patch('storage.roots',return_value=[]), patch('storage.measure',return_value={}), patch('storage.admit',side_effect=RuntimeError('budget refused')), patch('shutil.copyfile') as copy, patch('subprocess.Popen') as spawn:
            with self.assertRaisesRegex(RuntimeError,'budget refused'):probe.host(args)
            self.assertFalse(output.exists());copy.assert_not_called();spawn.assert_not_called()

    def test_non_capsule_output_refuses_before_any_measurement(self):
        with patch('subprocess.run') as run, patch('subprocess.Popen') as spawn:
            with self.assertRaises(ValueError):probe.host(SimpleNamespace(output=Path('/tmp/unrelated-fixture'),storage_budget_gib=48))
            run.assert_not_called();spawn.assert_not_called()

if __name__=='__main__':unittest.main()
