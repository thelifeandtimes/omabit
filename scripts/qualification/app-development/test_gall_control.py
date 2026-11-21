import copy
import json
import os
from pathlib import Path
import tempfile
import unittest

from gall_control import read_request, validate_restart, write_receipt

class OwnedRestartTests(unittest.TestCase):
    def setUp(self):
        self.request = {'version': 1, 'id': 'a'*32, 'action': 'restart', 'pid': 17,
                        'start': '12345', 'vere_sha256': 'b'*64, 'device': 42, 'inode': 99}
        self.owned = {key: self.request[key] for key in ('pid', 'start', 'vere_sha256', 'device', 'inode')}
        self.owned.update(executable='/artifacts/vere', running=True)

    def test_exact_owned_restart_and_refusals(self):
        original = copy.deepcopy(self.owned)
        validate_restart(self.request, self.owned)
        for key, value in [('version', True), ('action', 'stop'), ('id', '../x'), ('pid', 18),
                           ('start', '54321'), ('vere_sha256', 'c'*64), ('device', 43), ('inode', 100),
                           ('pid', True), ('device', False), ('id', []), ('command', '/bin/sh')]:
            with self.subTest(key=key, value=value):
                changed = dict(self.request); changed[key] = value
                with self.assertRaises(ValueError):
                    validate_restart(changed, self.owned)
        for key, value in [('running', False), ('executable', '/usr/bin/urbit')]:
            with self.subTest(key=key):
                with self.assertRaises(ValueError):
                    validate_restart(self.request, dict(self.owned, **{key: value}))
        self.assertEqual(original, self.owned)

    def test_bounded_private_request_and_retained_history(self):
        with tempfile.TemporaryDirectory(prefix='omabit-gall-control-') as directory:
            root = Path(directory); request = root/'request.json'
            request.write_text(json.dumps(self.request)); request.chmod(0o600)
            value, digest = read_request(request)
            self.assertEqual(value, self.request); self.assertEqual(len(digest), 64)
            for state in ('stop-dispatched', 'restart-dispatched', 'succeeded'):
                write_receipt(root, {'id': self.request['id'], 'state': state, 'request_sha256': digest})
            history = [json.loads(line) for line in (root/'gall-control-history.jsonl').read_text().splitlines()]
            self.assertEqual([row['state'] for row in history], ['stop-dispatched', 'restart-dispatched', 'succeeded'])
            self.assertEqual(json.loads((root/'gall-control-receipt.json').read_text()), history[-1])
            request.write_bytes(b' '*4097)
            with self.assertRaises(ValueError): read_request(request)
            request.unlink(); request.symlink_to(root/'gall-control-receipt.json')
            with self.assertRaises(OSError): read_request(request)

if __name__ == '__main__':
    unittest.main()
