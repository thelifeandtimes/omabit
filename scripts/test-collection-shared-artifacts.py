#!/usr/bin/env python3
"""Versioned shared-host packaging, original-reader refusal and publication loss."""
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / 'native/collections'))
import artifacts as a
import guarded_artifacts as previous
import shared_artifacts as shared
import journal as j

spec = importlib.util.spec_from_file_location('shared_artifact_inputs', Path(__file__).with_name('test-collection-artifacts.py'))
fixture = importlib.util.module_from_spec(spec)
spec.loader.exec_module(fixture)


class SharedArtifactTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix='omabit-shared-artifact-')
        self.root = Path(self.temp.name)
        self.source = self.root / 'inputs'
        self.pin = fixture.inputs(self.source)
        self.store = self.root / 'generations'
        self.store.mkdir(mode=0o700)
        self.support = ROOT / 'native/collections'
        self.support_pin = a.inventory(a.snapshot(self.support, list(shared.SUPPORT)))
        self.addCleanup(self.cleanup)

    def cleanup(self):
        for directory, _, _ in os.walk(self.root):
            os.chmod(directory, 0o700)
        self.temp.cleanup()

    def prepare(self, **changes):
        options = dict(role='theme', profile='qualification-stable', environment=None,
                       admission_source=self.support, expected_admission=self.support_pin)
        options.update(changes)
        return shared.prepare(self.source, self.pin, self.store, **options)

    def snapshot(self, root):
        root = Path(root)
        return a.snapshot(root, sorted(str(p.relative_to(root)) for p in root.rglob('*') if p.is_file()))

    def test_shared_host_complete_bundle_and_exact_retry(self):
        result = self.prepare()
        root = Path(result['path'])
        self.assertEqual(3, result['descriptor']['format'])
        self.assertTrue(result['shared_authority_capable'])
        self.assertEqual(result, self.prepare())
        self.assertEqual(self.support_pin, a.inventory(a.snapshot(root / 'collection', list(shared.SUPPORT))))
        host = (root / shared.HOST).read_text()
        self.assertIn('Admission.SharedHost {', host)
        self.assertIn('collection/shared_admission_guard.py', host)
        self.assertIn('SharedHost 1.0 SharedHost.qml', (root / 'collection/qmldir').read_text())
        self.assertNotIn('Runtime.Service', host)
        self.assertFalse((self.store.parent / 'manifest.json').exists())

    def test_frozen_format2_writer_reproduces_and_refuses_successor(self):
        old = ROOT / 'tests/fixtures/collection-artifact-v2'
        meta = json.loads((old / 'provenance.json').read_text())
        for name, digest in meta['sha256'].items():
            self.assertEqual(hashlib.sha256((old / name).read_bytes()).hexdigest(), digest)
        result = self.prepare()
        before = self.snapshot(result['path'])
        code = '''import json,sys
from pathlib import Path
f=Path(sys.argv[1]);sys.path[:0]=[str(f),str(f/'dependencies')];import writer
m=json.loads((f/'provenance.json').read_text())
v=writer.prepare(f/'source',m['source'],sys.argv[2],role=m['role'],profile=m['profile'],environment=m['environment'],admission_source=f/'support',expected_admission=m['admission'])
assert v['artifact_sha256']==m['artifact_sha256']
try:writer.verify(sys.argv[3],sys.argv[4])
except ValueError:pass
else:raise AssertionError('original format2 reader accepted format3')
print(json.dumps(v))
'''
        completed = subprocess.run([sys.executable, '-B', '-c', code, str(old), str(self.store),
                                    result['path'], result['artifact_sha256']], capture_output=True, text=True, timeout=10)
        self.assertEqual(0, completed.returncode, completed.stderr)
        reproduced = json.loads(completed.stdout)
        self.assertEqual(self.snapshot(old / 'generation'), self.snapshot(reproduced['path']))
        self.assertFalse(shared.verify(reproduced['path'], reproduced['artifact_sha256'])['shared_authority_capable'])
        self.assertEqual(before, self.snapshot(result['path']))
        with self.assertRaises(j.Conflict):
            previous.verify(result['path'], result['artifact_sha256'])
        self.assertEqual(before, self.snapshot(result['path']))

    def test_refactored_format2_writer_preserves_original_bytes(self):
        old = ROOT / 'tests/fixtures/collection-artifact-v2'
        meta = json.loads((old / 'provenance.json').read_text())
        value = previous.prepare(old / 'source', meta['source'], self.store, role=meta['role'],
                                 profile=meta['profile'], environment=meta['environment'],
                                 admission_source=old / 'support', expected_admission=meta['admission'])
        self.assertEqual(meta['artifact_sha256'], value['artifact_sha256'])
        self.assertEqual(self.snapshot(old / 'generation'), self.snapshot(value['path']))

    def test_old_generation_readability_never_grants_shared_admission(self):
        for version in (1, 2):
            old = ROOT / ('tests/fixtures/collection-artifact-v' + str(version))
            meta = json.loads((old / 'provenance.json').read_text())
            before = self.snapshot(old / 'generation')
            result = shared.verify(old / 'generation', meta['artifact_sha256'])
            self.assertFalse(result['shared_authority_capable'])
            with self.assertRaises(j.Conflict):
                shared.observation(old / 'generation', meta['artifact_sha256'], {})
            self.assertEqual(before, self.snapshot(old / 'generation'))

    def test_missing_or_changed_shared_support_refuses_before_publication(self):
        for name in ('SharedHost.qml', 'runtime_authority.py', 'shared_admission.py', 'shared_admission_guard.py'):
            pin = json.loads(json.dumps(self.support_pin))
            del pin[name]
            with self.assertRaises(j.Conflict):
                self.prepare(expected_admission=pin)
            pin = json.loads(json.dumps(self.support_pin))
            pin[name]['sha256'] = '0' * 64
            with self.assertRaises(j.Conflict):
                self.prepare(expected_admission=pin)
        self.assertEqual([], list(self.store.iterdir()))

    def test_shared_contract_files_and_generated_entry_tampering_refuses(self):
        names = (shared.HOST, shared.WIDGET, 'collection/SharedHost.qml', 'collection/runtime_authority.py',
                 'collection/shared_admission.py', 'collection/shared_admission_guard.py', 'collection/qmldir')
        for number, name in enumerate(names):
            result = self.prepare(profile='shared-drift-' + str(number))
            path = Path(result['path']) / name
            path.chmod(0o600)
            path.write_text('tampered')
            with self.subTest(name=name), self.assertRaises(j.Conflict):
                shared.verify(result['path'], result['artifact_sha256'])

    def test_lost_publication_response_replays_existing_generation(self):
        original = a._publish
        def lost(stage, final):
            original(stage, final)
            raise OSError('lost publication reply')
        with patch.object(a, '_publish', lost), self.assertRaises(OSError):
            self.prepare()
        with patch.object(a, '_publish', side_effect=AssertionError('unexpected republish')):
            result = self.prepare()
        self.assertTrue(shared.verify(result['path'], result['artifact_sha256'])['shared_authority_capable'])

    def test_shared_runtime_observation_binds_generation(self):
        result = self.prepare()
        root = Path(result['path'])
        value = dict(format=1, plugin_id=result['manifest']['id'], profile=result['descriptor']['profile'],
                     artifact_sha256=result['artifact_sha256'], instance='a' * 32, source=(root / a.WRAPPER).as_uri())
        self.assertEqual('a' * 32, shared.observation(root, result['artifact_sha256'], value)['instance'])
        with self.assertRaises(j.Conflict):
            shared.observation(root, result['artifact_sha256'], dict(value, source=(root / shared.HOST).as_uri()))

    def test_all_supported_roles_have_distinct_shared_generations(self):
        for role in ('omabit', 'dms'):
            source = self.root / role
            pin = fixture.inputs(source, role)
            result = shared.prepare(source, pin, self.store, role=role, profile='qualification-dev',
                                    environment=fixture.ENV if role == 'omabit' else None,
                                    admission_source=self.support, expected_admission=self.support_pin)
            self.assertTrue(shared.verify(result['path'], result['artifact_sha256'])['shared_authority_capable'])


if __name__ == '__main__':
    unittest.main()
