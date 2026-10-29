#!/usr/bin/env python3
"""Immutable protocol4 packaging, static inspection and complete-set preflight."""
import copy
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / 'native/collections'))
import artifacts as a
import recovery_artifacts as artifacts
import runtime_preflight as preflight
import shared_artifacts as previous
import journal as j

loader = importlib.util.spec_from_file_location('preflight_inputs', Path(__file__).with_name('test-collection-artifacts.py'))
fixture = importlib.util.module_from_spec(loader);loader.loader.exec_module(fixture)


class PreflightTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix='omabit-runtime-preflight-')
        self.root = Path(self.temp.name);self.store = self.root / 'generations';self.store.mkdir(mode=0o700)
        self.support = ROOT / 'native/collections';self.support_pin = a.inventory(a.snapshot(self.support, list(artifacts.SUPPORT)))
        self.source = self.root / 'theme';self.pin = fixture.inputs(self.source)
        self.addCleanup(self.cleanup)

    def cleanup(self):
        for root, _, _ in os.walk(self.root):os.chmod(root, 0o700)
        self.temp.cleanup()

    def prepare(self, **changes):
        args = dict(role='theme', profile='qualification-stable', environment=None,
                    admission_source=self.support, expected_admission=self.support_pin)
        args.update(changes)
        return artifacts.prepare(self.source, self.pin, self.store, **args)

    def snapshot(self, root):
        root = Path(root)
        return a.snapshot(root, sorted(str(p.relative_to(root)) for p in root.rglob('*') if p.is_file()))

    def collection(self):
        values = {'theme': self.prepare()}
        for role in ('omabit', 'dms'):
            source = self.root / role;pin = fixture.inputs(source, role)
            values[role] = artifacts.prepare(source, pin, self.store, role=role, profile='qualification-stable',
                environment=fixture.ENV if role=='omabit' else None, admission_source=self.support, expected_admission=self.support_pin)
        spec = {'format': 1, 'name': 'stable', 'members': {
            role: {'plugin_id': value['descriptor']['plugin_id'], 'profile': value['descriptor']['profile'],
                   'artifact_sha256': value['artifact_sha256'], 'endpoint_ref': None,
                   'paths': {name: '/isolated/' + role + '/' + name for name in j.PATHS}}
            for role, value in values.items()}}
        return spec, {role: value['path'] for role, value in values.items()}

    def modified_support(self, name, data):
        dest = self.root / 'support';dest.mkdir()
        for filename in artifacts.SUPPORT:shutil.copy2(self.support / filename, dest / filename)
        (dest / name).write_bytes(data)
        return dict(admission_source=dest, expected_admission=a.inventory(a.snapshot(dest, list(artifacts.SUPPORT))))

    def test_complete_set_declares_pinned_readers_without_effects(self):
        spec, roots = self.collection();before = {role: self.snapshot(root) for role, root in roots.items()}
        result = preflight.require_ready(spec, roots)
        self.assertTrue(result['ready']);self.assertEqual(j.digest(spec), result['collection'])
        self.assertEqual(preflight.REQUIRED, result['required'])
        self.assertTrue(all(row['artifact_format']==4 and row['runtime_protocol']==artifacts.PROTOCOL for row in result['members'].values()))
        self.assertEqual(before, {role: self.snapshot(root) for role, root in roots.items()})
        self.assertFalse((self.root / 'manifest.json').exists())
        self.assertEqual(self.prepare(), self.prepare())

    def test_original_format3_reproduces_and_refuses_format4(self):
        old = ROOT / 'tests/fixtures/collection-artifact-v3';meta = json.loads((old / 'provenance.json').read_text())
        for name, digest in meta['sha256'].items():self.assertEqual(hashlib.sha256((old/name).read_bytes()).hexdigest(), digest)
        new = self.prepare();before = self.snapshot(new['path'])
        code = '''import json,sys
from pathlib import Path
f=Path(sys.argv[1]);sys.path[:0]=[str(f),str(f/'dependencies')];import writer
m=json.loads((f/'provenance.json').read_text());v=writer.prepare(f/'source',m['source'],sys.argv[2],role=m['role'],profile=m['profile'],environment=m['environment'],admission_source=f/'support',expected_admission=m['admission'])
assert v['artifact_sha256']==m['artifact_sha256']
try:writer.verify(sys.argv[3],sys.argv[4])
except ValueError:pass
else:raise AssertionError('original format3 reader accepted format4')
print(json.dumps(v))
'''
        q = subprocess.run([sys.executable, '-B', '-c', code, str(old), str(self.store), new['path'], new['artifact_sha256']], capture_output=True, text=True, timeout=10)
        self.assertEqual(0, q.returncode, q.stderr);reproduced = json.loads(q.stdout)
        self.assertEqual(self.snapshot(old/'generation'), self.snapshot(reproduced['path']))
        self.assertEqual(before, self.snapshot(new['path']))
        with self.assertRaises(j.Conflict):previous.verify(new['path'], new['artifact_sha256'])

    def test_existing_format3_writer_preserves_original_bytes(self):
        old = ROOT / 'tests/fixtures/collection-artifact-v3';meta = json.loads((old/'provenance.json').read_text())
        value = previous.prepare(old/'source', meta['source'], self.store, role=meta['role'], profile=meta['profile'],
            environment=meta['environment'], admission_source=old/'support', expected_admission=meta['admission'])
        self.assertEqual(meta['artifact_sha256'], value['artifact_sha256'])
        self.assertEqual(self.snapshot(old/'generation'), self.snapshot(value['path']))

    def test_historical_generations_stay_readable_without_inferred_capability(self):
        for version in (1, 2, 3):
            old = ROOT / ('tests/fixtures/collection-artifact-v'+str(version));meta = json.loads((old/'provenance.json').read_text())
            before = self.snapshot(old/'generation')
            self.assertIsNone(artifacts.verify(old/'generation',meta['artifact_sha256'])['runtime_protocol'])
            self.assertEqual(before,self.snapshot(old/'generation'))

    def test_mixed_collection_refuses_before_any_member_effect(self):
        spec, roots = self.collection();old = ROOT/'tests/fixtures/collection-artifact-v3';meta = json.loads((old/'provenance.json').read_text())
        spec['members']['theme']['artifact_sha256'] = meta['artifact_sha256'];roots['theme'] = str(old/'generation')
        before = {role:self.snapshot(root) for role,root in roots.items()};result = preflight.check(spec,roots)
        self.assertFalse(result['ready']);self.assertEqual('runtime-protocol-undeclared',result['members']['theme']['reason'])
        self.assertTrue(result['members']['omabit']['ready'] and result['members']['dms']['ready'])
        with self.assertRaises(j.Conflict):preflight.require_ready(spec,roots)
        self.assertEqual(before,{role:self.snapshot(root) for role,root in roots.items()})

    def test_missing_pin_refuses_without_publication(self):
        pin = copy.deepcopy(self.support_pin);del pin[artifacts.PROTOCOL_FILE]
        with self.assertRaises(j.Conflict):self.prepare(expected_admission=pin)
        self.assertEqual([],list(self.store.iterdir()))

    def test_unsupported_declaration_refuses_without_publication(self):
        value = dict(artifacts.PROTOCOL,reservations=[1,2,3,4])
        with self.assertRaises(j.Conflict):self.prepare(**self.modified_support(artifacts.PROTOCOL_FILE,j.canonical(value)))
        self.assertEqual([],list(self.store.iterdir()))

    def test_old_reader_cannot_be_relabelled_by_new_protocol_file(self):
        source = ROOT/'tests/fixtures/collection-artifact-v3/support/runtime_authority.py'
        with self.assertRaises(j.Conflict):self.prepare(**self.modified_support('runtime_authority.py',source.read_bytes()))
        self.assertEqual([],list(self.store.iterdir()))

    def test_journal_reader_mismatch_refuses_before_publication(self):
        source = (self.support/'journal.py').read_bytes().replace(b'READ_FORMATS = (1, 2, 3)', b'READ_FORMATS = (1, 2)')
        with self.assertRaises(j.Conflict):self.prepare(**self.modified_support('journal.py',source))
        self.assertEqual([],list(self.store.iterdir()))

    def test_dynamic_or_duplicate_declaration_never_executes(self):
        for source in (b'READ_FORMATS = tuple(range(1,4))',b'READ_FORMATS=(1,2,3)\nREAD_FORMATS=(1,2)'):
            with self.subTest(source=source),self.assertRaises(j.Conflict):artifacts.literal(source,'READ_FORMATS')
        marker = self.root/'executed'
        source = b'READ_FORMATS=(1,2,3)\n'+('open('+repr(str(marker))+',"w").write("bad")\n').encode()
        self.assertEqual((1,2,3),artifacts.literal(source,'READ_FORMATS'));self.assertFalse(marker.exists())

    def test_future_required_protocol_reports_unsupported(self):
        spec,roots=self.collection();result=preflight.check(spec,roots,dict(preflight.REQUIRED,reservation=4))
        self.assertFalse(result['ready']);self.assertTrue(all(row['reason']=='runtime-protocol-unsupported' for row in result['members'].values()))

    def test_invalid_requirements_or_incomplete_roots_refuse(self):
        spec,roots=self.collection()
        for value in (True,0,-1,'3'):
            with self.subTest(value=value),self.assertRaises(j.Conflict):preflight.check(spec,roots,dict(preflight.REQUIRED,reservation=value))
        del roots['dms']
        with self.assertRaises(j.Conflict):preflight.check(spec,roots)

    def test_wrong_profile_and_changed_payload_report_invalid(self):
        spec,roots=self.collection();wrong=copy.deepcopy(spec);wrong['members']['theme']['profile']='qualification-other'
        self.assertEqual('artifact-invalid',preflight.check(wrong,roots)['members']['theme']['reason'])
        path=Path(roots['dms'])/'helper.py';path.chmod(0o600);path.write_text('changed')
        result=preflight.check(spec,roots);self.assertEqual('artifact-invalid',result['members']['dms']['reason']);self.assertFalse(result['ready'])

    def test_declared_protocol_tampering_refuses(self):
        value=self.prepare();path=Path(value['path'])/'collection'/artifacts.PROTOCOL_FILE;path.chmod(0o600);path.write_text('{}')
        with self.assertRaises(j.Conflict):artifacts.verify(value['path'],value['artifact_sha256'])

    def test_lost_publication_reply_reuses_exact_generation(self):
        publish=a._publish
        def lost(stage,final):publish(stage,final);raise OSError('lost reply')
        with patch.object(a,'_publish',lost),self.assertRaises(OSError):self.prepare()
        with patch.object(a,'_publish',side_effect=AssertionError('duplicate publication')):value=self.prepare()
        self.assertEqual(artifacts.PROTOCOL,value['runtime_protocol'])

    def test_readonly_cli_reports_ready_blocked_and_invalid(self):
        spec,roots=self.collection();sp=self.root/'spec.json';rp=self.root/'roots.json';sp.write_bytes(j.canonical(spec));rp.write_bytes(j.canonical(roots))
        command=[sys.executable,'-B',str(ROOT/'scripts/check-collection-runtime.py'),'--specification',str(sp),'--generations',str(rp)]
        q=subprocess.run(command,capture_output=True,text=True,timeout=10);self.assertEqual(0,q.returncode,q.stderr);self.assertTrue(json.loads(q.stdout)['ready'])
        q=subprocess.run(command+['--reservation-format','4'],capture_output=True,text=True,timeout=10);self.assertEqual(2,q.returncode,q.stderr);self.assertFalse(json.loads(q.stdout)['ready'])
        q=subprocess.run(command+['--journal-format','0'],capture_output=True,text=True,timeout=10);self.assertEqual(1,q.returncode);self.assertIn('error',json.loads(q.stderr))


if __name__=='__main__':unittest.main()
