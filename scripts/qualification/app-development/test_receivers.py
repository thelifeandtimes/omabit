"""Offline tests for the narrow, pre-install OOM recovery admission."""
import json
from pathlib import Path
import tempfile
from types import SimpleNamespace
import unittest
from unittest.mock import patch

from receivers import interrupted_boot_source


class ReceiverRecoveryAdmission(unittest.TestCase):
    def fixture(self, root):
        def write(name, value):
            path = root / name
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_text(json.dumps(value))
        write('guard.json', {'unit': 'omabit-desk-probe-fixture', 'receivers': True})
        write('scope-termination.json', {'unit': 'omabit-desk-probe-fixture.scope', 'observed': {'Result': 'oom-kill'}})
        write('lab/result.json', {'clean_shutdown': False})
        expected = {'exists': True, 'revision': 7, 'clay_hash': '0v123', 'publication': 'published'}
        write('lab/workflow-proof.json', {'ok': True, 'glob-upload-committed-docket': expected})
        (root / 'lab/workflow-effects.jsonl').write_text('{"id":1,"state":"dispatched"}\n{"id":1,"state":"acknowledged"}\n')
        for ship in ('zod', 'nec', 'bud'):
            p = root / 'lab' / ship / '.urb/log/data.mdb'
            p.parent.mkdir(parents=True)
            p.write_bytes(b'fixture, never booted')
        return write, expected

    def test_only_unused_receivers_and_completed_publisher_are_admitted(self):
        for scenario in ('ok', 'active', 'uncertain', 'receiver-effect', 'receiver-proof', 'publisher-failed', 'symlink', 'not-oom'):
            with self.subTest(scenario=scenario), tempfile.TemporaryDirectory() as directory:
                root = Path(directory)
                write, expected = self.fixture(root)
                state = 'ActiveState=failed\nResult=oom-kill\nControlGroup=\n'
                if scenario == 'active':
                    state = 'ActiveState=active\nResult=oom-kill\nControlGroup=/live\n'
                elif scenario == 'uncertain':
                    (root / 'lab/workflow-effects.jsonl').write_text('{"id":1,"state":"dispatched"}\n')
                elif scenario == 'receiver-effect':
                    (root / 'lab/receiver-nec-effects.jsonl').write_text('{"id":1,"state":"acknowledged"}\n')
                elif scenario == 'receiver-proof':
                    write('lab/receiver-proof.json', {'ok': False})
                elif scenario == 'publisher-failed':
                    write('lab/workflow-proof.json', {'ok': False})
                elif scenario == 'symlink':
                    (root / 'lab/bud/.urb/log/data.mdb').unlink()
                    (root / 'lab/bud/.urb/log/data.mdb').symlink_to(root / 'lab/nec/.urb/log/data.mdb')
                elif scenario == 'not-oom':
                    write('scope-termination.json', {'unit': 'omabit-desk-probe-fixture.scope', 'observed': {'Result': 'exit-code'}})
                with patch('receivers.subprocess.run', return_value=SimpleNamespace(stdout=state)):
                    if scenario == 'ok':
                        self.assertEqual(interrupted_boot_source(root), expected)
                    else:
                        with self.assertRaises(ValueError):
                            interrupted_boot_source(root)


if __name__ == '__main__':
    unittest.main()
