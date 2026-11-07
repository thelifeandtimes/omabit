import json
from pathlib import Path
import tempfile
from types import SimpleNamespace
import unittest
from unittest.mock import patch
import uuid

import storage


class StorageAdmission(unittest.TestCase):
    def test_launcher_refuses_before_output_copy_transfer_or_spawn(self):
        import probe
        with tempfile.TemporaryDirectory() as directory:
            requests = Path(directory) / 'requests.json'
            requests.write_text(json.dumps(dict.fromkeys(['commit-files','commit-invalid','committed-equal','clay-files','normalized-files','normalized-different','desks','files','autocommit','inventory','create','mount','commit','publish-source','equivalent','different'], {})))
            output = Path('/tmp') / ('omabit-desk-probe-storage-test-' + uuid.uuid4().hex)
            args = SimpleNamespace(output=output, requests=requests, vere=Path('/not-read/vere'), pill=Path('/not-read/pill'), recover_receiver_boot=None, resume_from=None, receivers=False)
            with patch('probe.sha', side_effect=[probe.VERE_SHA, probe.PILL_SHA]), patch('probe.shutil.disk_usage', return_value=SimpleNamespace(free=8*storage.GIB)), patch('storage.admit', side_effect=RuntimeError('budget refused')), patch('probe.subprocess.Popen') as spawn, patch('probe.shutil.copyfile') as copy:
                with self.assertRaisesRegex(RuntimeError, 'budget refused'):
                    probe.host(args)
                self.assertFalse(output.exists())
                spawn.assert_not_called()
                copy.assert_not_called()

    def test_failed_budget_refuses_including_partial_measurement_reserve(self):
        rows = [{'allocated_bytes': 38 * storage.GIB, 'unmeasured_reserve': storage.GIB}]
        self.assertEqual(storage.admit(storage.GIB, rows)['total_with_reserves'], storage.BUDGET)
        with self.assertRaises(RuntimeError):
            storage.admit(storage.GIB + 1, rows)

    def test_unexpected_or_expanding_unreadable_state_refuses(self):
        error = "du: cannot read directory '/srv/dev/urbit/state/.local/share/urbit-runner/docker/containerd/daemon/io.containerd.snapshotter.v1.overlayfs/snapshots/42/work/work': Permission denied\n"
        for path, stderr, admitted in [(storage.DEV, error, True), (storage.DEV, error * 19, False),
                                      (Path('/tmp/omabit-fixture'), error, False),
                                      (storage.DEV, 'du: other failure', False)]:
            with self.subTest(path=path, stderr=stderr), patch('storage.subprocess.run', return_value=SimpleNamespace(returncode=1, stdout='4096\tpath\n', stderr=stderr)):
                if admitted:
                    self.assertEqual(storage.measure(path)['unmeasured_reserve'], storage.GIB)
                else:
                    with self.assertRaises(RuntimeError):
                        storage.measure(path)


if __name__ == '__main__':
    unittest.main()
