"""Inert regression checks for qualification evidence and source boundaries."""
import base64
import hashlib
import importlib.util
import json
from pathlib import Path
import unittest

spec = importlib.util.spec_from_file_location('runtime_profiles_host', Path(__file__).with_name('run.py'))
host = importlib.util.module_from_spec(spec)
spec.loader.exec_module(host)


class EvidenceTests(unittest.TestCase):
    def frames(self):
        # Larger than a typical console line: the earlier upgrade lost this.
        raw = json.dumps({'passed': True, 'history': 'x' * 18000}).encode()
        text = base64.b64encode(raw).decode()
        chunks = [text[i:i + 3000] for i in range(0, len(text), 3000)]
        lines = ['console: OMABIT_PROFILE_REPORT_CHUNK ' + json.dumps({'i': i, 'data': chunk})
                 for i, chunk in enumerate(chunks)]
        lines.append('console: OMABIT_PROFILE_REPORT_END ' + json.dumps(
            {'chunks': len(chunks), 'sha256': hashlib.sha256(raw).hexdigest()}))
        return lines

    def test_long_report_roundtrip(self):
        self.assertEqual(len(host.read_report(self.frames())['history']), 18000)

    def test_truncated_console_is_not_a_pass(self):
        lines = self.frames()
        for incomplete in (lines[:-1], lines[1:]):
            with self.assertRaises(RuntimeError):
                host.read_report(incomplete)

    def test_duplicate_or_corrupt_frames_are_not_a_pass(self):
        lines = self.frames()
        with self.assertRaises(RuntimeError):
            host.read_report([lines[0], *lines])
        corrupt = list(lines)
        corrupt[0] = corrupt[0].replace('eyJ', 'eCJ', 1)
        with self.assertRaises(RuntimeError):
            host.read_report(corrupt)

    def test_only_harness_and_distribution_changes_are_allowed(self):
        host.validate_frozen_changes(['docs/runtime-profiles-review.md',
                                     'scripts/qualification/runtime-profiles/guest.py'])
        for path in ('internal/managed/store.go', 'scripts/package-release.py',
                     'native/bootstrap/runtime.py', 'scripts/qualification/patch-upgrade/guest.py'):
            with self.subTest(path=path), self.assertRaises(ValueError):
                host.validate_frozen_changes([path])


if __name__ == '__main__':
    unittest.main()
