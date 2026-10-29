import hashlib
import json
from pathlib import Path
import tempfile
import unittest
from guest import project_receipt_summary
from console_events import guest_events

class ProjectReceiptTransport(unittest.TestCase):
    def test_large_complete_receipt_is_retained_and_serial_reference_is_bounded(self):
        with tempfile.TemporaryDirectory(prefix='omabit-receipt-') as root:
            p=Path(root)/'project.json'
            raw=json.dumps({'ok':True,'private_history':'x'*(1024*1024)}).encode()
            p.write_bytes(raw)
            result=project_receipt_summary(p)
            event='OMABIT_GUEST '+json.dumps({'stage':'result','project_preparation':result})
            self.assertLess(len(event),512)
            self.assertEqual(guest_events(event)[0]['project_preparation'],result)
            self.assertEqual(result,{'ok':True,'receipt':{'path':'results/project-preparation.json','bytes':len(raw),'sha256':hashlib.sha256(raw).hexdigest()}})
            self.assertEqual(p.read_bytes(),raw)
    def test_failure_is_not_relabelled_success(self):
        with tempfile.TemporaryDirectory(prefix='omabit-receipt-') as root:
            p=Path(root)/'project.json';p.write_text('{"ok":false,"stage":"failed"}')
            self.assertFalse(project_receipt_summary(p)['ok'])
    def test_invalid_or_oversized_receipts_fail(self):
        for raw in [b'not-json',b'[]',b'{"ok":"true"}',b'x'*(4*1024*1024+1)]:
            with self.subTest(size=len(raw)),tempfile.TemporaryDirectory(prefix='omabit-receipt-') as root:
                p=Path(root)/'project.json';p.write_bytes(raw)
                with self.assertRaises(ValueError):project_receipt_summary(p)

if __name__=='__main__':unittest.main()
