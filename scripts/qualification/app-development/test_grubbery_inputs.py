import hashlib
import json
import os
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

import grubbery_inputs


class FrameworkSourceAdmission(unittest.TestCase):
    def test_exact_files_and_unsafe_or_changed_source(self):
        with tempfile.TemporaryDirectory(prefix='omabit-grubbery-source-test-') as name:
            root = Path(name)
            source = root/'desk'; source.mkdir()
            data = b'owned source\n'
            (source/'sample.hoon').write_bytes(data)
            definition = root/'definition.json'
            frozen = {'commit': 'a'*40, 'source': {'digest': 'b'*64, 'files': [
                {'path': 'sample.hoon', 'bytes': len(data), 'sha256': hashlib.sha256(data).hexdigest()}]}}
            definition.write_text(json.dumps(frozen))
            pin = hashlib.sha256(definition.read_bytes()).hexdigest()
            with self.assertRaises(ValueError):
                grubbery_inputs.verify_framework_source(source, definition)
            # Exercise traversal and byte checks with a tiny test-only pin; the
            # production caller never supplies or changes DEFINITION_SHA.
            with patch.object(grubbery_inputs, 'DEFINITION_SHA', pin):
                self.assertEqual(grubbery_inputs.verify_framework_source(source, definition)['files'], 1)
                (source/'extra.txt').write_text('unexpected')
                with self.assertRaises(ValueError): grubbery_inputs.verify_framework_source(source, definition)
                (source/'extra.txt').unlink()
                (source/'sample.hoon').write_bytes(b'changed byte\n')
                with self.assertRaises(ValueError): grubbery_inputs.verify_framework_source(source, definition)
                (source/'sample.hoon').unlink()
                with self.assertRaises(ValueError): grubbery_inputs.verify_framework_source(source, definition)
                outside = root/'outside.hoon'; outside.write_bytes(data)
                (source/'sample.hoon').symlink_to(outside)
                with self.assertRaises(ValueError): grubbery_inputs.verify_framework_source(source, definition)
                (source/'sample.hoon').unlink(); os.link(outside, source/'sample.hoon')
                with self.assertRaises(ValueError): grubbery_inputs.verify_framework_source(source, definition)
                (source/'sample.hoon').unlink(); (source/'sample.hoon').write_bytes(data)
                (source/'linked').symlink_to(root, target_is_directory=True)
                with self.assertRaises(ValueError): grubbery_inputs.verify_framework_source(source, definition)


if __name__ == '__main__':
    unittest.main()
