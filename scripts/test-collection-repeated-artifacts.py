#!/usr/bin/env python3
"""Complete continuation bundles, historical reproduction and explicit preflight."""
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
import repeated_artifacts as artifacts
import repeated_preflight as preflight
import continuation_artifacts as previous
import successor_artifacts as five
import recovery_artifacts as older
import continuation_preflight as old_preflight
import journal as j

loader = importlib.util.spec_from_file_location('repeated_artifact_inputs', Path(__file__).with_name('test-collection-artifacts.py'))
fixture = importlib.util.module_from_spec(loader);loader.loader.exec_module(fixture)


class RepeatedArtifactTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix='omabit-repeated-artifacts-')
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

    def test_complete_set_static_preflight_and_export(self):
        spec, roots = self.collection();before = {role: self.snapshot(root) for role, root in roots.items()}
        for authority in (1, 2, 3, 4):
            result = preflight.require_ready(spec, roots, dict(preflight.REQUIRED, authority=authority))
            self.assertTrue(result['ready']);self.assertEqual(j.digest(spec), result['collection'])
            self.assertTrue(all(row['artifact_format']==7 and row['runtime_protocol']==artifacts.PROTOCOL for row in result['members'].values()))
        for root in roots.values():
            host = (Path(root)/artifacts.HOST).read_bytes()
            self.assertIn(b'Admission.SharedHost',host);self.assertEqual(1,host.count(b'collection/repeated_admission_guard.py'))
            self.assertNotIn(b'collection/shared_admission_guard.py',host)
        self.assertEqual(before, {role: self.snapshot(root) for role, root in roots.items()})
        if Path.home()==Path('/lab/home'):
            dest=Path('/lab/repeated-artifact-export');dest.mkdir(mode=0o700)
            for role,root in roots.items():shutil.copytree(root,dest/role)
            (dest/'specification.json').write_bytes(j.canonical(spec))
            (dest/'preflight.json').write_bytes(j.canonical(result))

    def test_bundled_imports_need_no_checkout_or_site_packages(self):
        value=self.prepare();bundle=Path(value['path'])/'collection'
        code='''import importlib,json,sys
from pathlib import Path
bundle=Path(sys.argv[1]);sys.path.insert(0,str(bundle))
names=sorted(p.stem for p in bundle.glob('*.py'))
for name in names:
 module=importlib.import_module(name)
 assert Path(module.__file__).parent==bundle,(name,module.__file__)
assert sys.modules['repeated_authority'].AUTHORITY_READ_FORMATS==(1,2,3,4)
print(json.dumps({'modules':names,'isolated':sys.flags.isolated,'no_site':sys.flags.no_site}))
'''
        q=subprocess.run([sys.executable,'-I','-S','-B','-c',code,str(bundle)],cwd=self.root,capture_output=True,text=True,timeout=10)
        self.assertEqual(0,q.returncode,q.stderr);result=json.loads(q.stdout)
        self.assertEqual(1,result['isolated']);self.assertEqual(1,result['no_site'])
        self.assertEqual(sorted(Path(n).stem for n in artifacts.SUPPORT if n.endswith('.py')),result['modules'])

    def test_original_format6_reproduces_and_refuses7_without_mutation(self):
        old=ROOT/'tests/fixtures/collection-artifact-v6';meta=json.loads((old/'provenance.json').read_text())
        for name,digest in meta['sha256'].items():self.assertEqual(hashlib.sha256((old/name).read_bytes()).hexdigest(),digest)
        new=self.prepare();before=self.snapshot(new['path'])
        code='''import json,sys
from pathlib import Path
f=Path(sys.argv[1]);sys.path[:0]=[str(f),str(f/'dependencies')];import writer
m=json.loads((f/'provenance.json').read_text());v=writer.prepare(f/'source',m['source'],sys.argv[2],role=m['role'],profile=m['profile'],environment=m['environment'],admission_source=f/'support',expected_admission=m['admission'])
assert v['artifact_sha256']==m['artifact_sha256']
try:writer.verify(sys.argv[3],sys.argv[4])
except ValueError:pass
else:raise AssertionError('original format6 reader accepted7')
print(json.dumps(v))
'''
        q=subprocess.run([sys.executable,'-B','-c',code,str(old),str(self.store),new['path'],new['artifact_sha256']],capture_output=True,text=True,timeout=10)
        self.assertEqual(0,q.returncode,q.stderr);reproduced=json.loads(q.stdout)
        self.assertEqual(self.snapshot(old/'generation'),self.snapshot(reproduced['path']))
        self.assertEqual(before,self.snapshot(new['path']))
        with self.assertRaises(j.Conflict):previous.verify(new['path'],new['artifact_sha256'])

    def test_existing_writer6_preserves_original_bytes(self):
        old=ROOT/'tests/fixtures/collection-artifact-v6';meta=json.loads((old/'provenance.json').read_text())
        value=previous.prepare(old/'source',meta['source'],self.store,role=meta['role'],profile=meta['profile'],
            environment=meta['environment'],admission_source=old/'support',expected_admission=meta['admission'])
        self.assertEqual(meta['artifact_sha256'],value['artifact_sha256'])
        self.assertEqual(self.snapshot(old/'generation'),self.snapshot(value['path']))

    def test_legacy1_to6_readable_without_invented_support(self):
        for version in (1,2,3,4,5,6):
            old=ROOT/('tests/fixtures/collection-artifact-v'+str(version));meta=json.loads((old/'provenance.json').read_text())
            before=self.snapshot(old/'generation');result=artifacts.verify(old/'generation',meta['artifact_sha256'])
            self.assertEqual(previous.PROTOCOL if version==6 else five.PROTOCOL if version==5 else older.PROTOCOL if version==4 else None,result['runtime_protocol'])
            self.assertEqual(before,self.snapshot(old/'generation'))

    def test_mixed6_and7_allows3_refuses4_without_effect(self):
        spec,roots=self.collection();old=ROOT/'tests/fixtures/collection-artifact-v6';meta=json.loads((old/'provenance.json').read_text())
        spec['members']['theme']['artifact_sha256']=meta['artifact_sha256'];roots['theme']=str(old/'generation')
        before={role:self.snapshot(root) for role,root in roots.items()}
        self.assertTrue(preflight.require_ready(spec,roots,dict(preflight.REQUIRED,authority=3))['ready'])
        result=preflight.check(spec,roots);self.assertFalse(result['ready'])
        self.assertEqual('runtime-protocol-unsupported',result['members']['theme']['reason'])
        self.assertTrue(result['members']['omabit']['ready'] and result['members']['dms']['ready'])
        with self.assertRaises(j.Conflict):preflight.require_ready(spec,roots)
        self.assertEqual(before,{role:self.snapshot(root) for role,root in roots.items()})

    def test_original_preflight_refuses7_without_mutation(self):
        spec,roots=self.collection();before={role:self.snapshot(root) for role,root in roots.items()}
        result=old_preflight.check(spec,roots);self.assertFalse(result['ready'])
        self.assertTrue(all(row['reason']=='artifact-invalid' for row in result['members'].values()))
        self.assertEqual(before,{role:self.snapshot(root) for role,root in roots.items()})

    def test_undeclared_legacy_is_not_promoted(self):
        spec,roots=self.collection();old=ROOT/'tests/fixtures/collection-artifact-v3';meta=json.loads((old/'provenance.json').read_text())
        spec['members']['theme']['artifact_sha256']=meta['artifact_sha256'];roots['theme']=str(old/'generation')
        self.assertEqual('runtime-protocol-undeclared',preflight.check(spec,roots)['members']['theme']['reason'])

    def test_missing_support_pin_refuses_before_staging(self):
        for name in ('checkpoint_lineage.py','repeated_session.py','repeated_checkpoint.py','repeated_cleanup.py','repeated_inspection.py','continuation_cleanup.py','continuation_inspection.py',artifacts.PROTOCOL_FILE):
            pin=copy.deepcopy(self.support_pin);del pin[name]
            with self.subTest(name=name),self.assertRaises(j.Conflict):self.prepare(expected_admission=pin)
        self.assertEqual([],list(self.store.iterdir()))

    def test_missing_transitive_import_refuses_before_staging(self):
        source=(self.support/'repeated_checkpoint.py').read_bytes()+b'\nimport unbundled_recovery_dependency\n'
        with self.assertRaisesRegex(j.Conflict,'missing bundled dependency'):self.prepare(**self.modified_support('repeated_checkpoint.py',source))
        self.assertEqual([],list(self.store.iterdir()))

    def test_static_check_does_not_execute_bundled_code(self):
        marker=self.root/'executed'
        source=(self.support/'repeated_admission.py').read_bytes()+('\nopen('+repr(str(marker))+',"w").write("bad")\n').encode()
        result=self.prepare(**self.modified_support('repeated_admission.py',source))
        self.assertEqual(artifacts.PROTOCOL,artifacts.verify(result['path'],result['artifact_sha256'])['runtime_protocol'])
        self.assertFalse(marker.exists())

    def test_incorrect_declarations_refuse_before_staging(self):
        support=a.snapshot(self.support,list(artifacts.SUPPORT))
        for name,old,new in (
            ('repeated_authority.py',b'AUTHORITY_READ_FORMATS = (1, 2, 3, 4)',b'AUTHORITY_READ_FORMATS = (1, 2, 3)'),
            ('repeated_authority.py',b'FORMAT = 4',b'FORMAT = True'),
            ('repeated_admission.py',b'FORMAT = 2',b'FORMAT = 1'),
            ('runtime_authority.py',b'RESERVATION_READ_FORMATS = (1, 2, 3)',b'RESERVATION_READ_FORMATS = (1, 2)'),
            ('journal.py',b'READ_FORMATS = (1, 2, 3)',b'READ_FORMATS = (1, 2)'),
            ('successor_authority.py',b'AUTHORITY_READ_FORMATS = (1, 2)',b'AUTHORITY_READ_FORMATS = (1,)')):
            changed=copy.deepcopy(support);data,executable=changed[name];self.assertIn(old,data);changed[name]=(data.replace(old,new),executable)
            with self.subTest(name=name,old=old),self.assertRaises(j.Conflict):artifacts.protocol(changed)
        self.assertEqual([],list(self.store.iterdir()))

    def test_dynamic_ambiguous_or_future_protocol_refuses(self):
        support=a.snapshot(self.support,list(artifacts.SUPPORT))
        for value in (dict(artifacts.PROTOCOL,authorities=[1,2,3,4,5]),dict(artifacts.PROTOCOL,format=True),dict(previous.PROTOCOL)):
            changed=copy.deepcopy(support);changed[artifacts.PROTOCOL_FILE]=(j.canonical(value),False)
            with self.subTest(value=value),self.assertRaises(j.Conflict):artifacts.protocol(changed)
        for literal in (b'AUTHORITY_READ_FORMATS=tuple([1,2,3,4])',b'AUTHORITY_READ_FORMATS=(1,2,3,4)\nAUTHORITY_READ_FORMATS=(1,2,3)'):
            changed=copy.deepcopy(support);changed['repeated_authority.py']=(b'FORMAT=4\n'+literal,False)
            with self.subTest(literal=literal),self.assertRaises(j.Conflict):artifacts.protocol(changed)

    def test_tampered_payload_and_guard_refuse(self):
        spec,roots=self.collection()
        for role,name in (('theme','collection/repeated_admission_guard.py'),('dms','collection/repeated-protocol.json')):
            path=Path(roots[role])/name;path.chmod(0o600);path.write_text('changed')
        result=preflight.check(spec,roots);self.assertFalse(result['ready'])
        for role in ('theme','dms'):self.assertEqual('artifact-invalid',result['members'][role]['reason'])

    def test_wrong_profile_and_future_requirements_refuse(self):
        spec,roots=self.collection();wrong=copy.deepcopy(spec);wrong['members']['theme']['profile']='qualification-other'
        self.assertEqual('artifact-invalid',preflight.check(wrong,roots)['members']['theme']['reason'])
        for requirement in ('authority','admission','reservation','journal'):
            result=preflight.check(spec,roots,dict(preflight.REQUIRED,**{requirement:9}))
            self.assertTrue(all(row['reason']=='runtime-protocol-unsupported' for row in result['members'].values()))
        for value in (True,0,-1,'2'):
            with self.subTest(value=value),self.assertRaises(j.Conflict):preflight.check(spec,roots,dict(preflight.REQUIRED,authority=value))

    def test_lost_publication_reply_and_exact_retry(self):
        publish=a._publish
        def lost(stage,final):publish(stage,final);raise OSError('lost reply')
        with patch.object(a,'_publish',lost),self.assertRaises(OSError):self.prepare()
        with patch.object(a,'_publish',side_effect=AssertionError('duplicate publication')):value=self.prepare()
        info=Path(value['path']).stat();self.assertEqual(value,self.prepare());self.assertEqual(info.st_ino,Path(value['path']).stat().st_ino)

    def test_observation_binds_actual_wrapper(self):
        value=self.prepare();d=value['descriptor']
        observation=dict(format=1,plugin_id=d['plugin_id'],profile=d['profile'],artifact_sha256=value['artifact_sha256'],instance='8'*32,source=(Path(value['path'])/a.WRAPPER).as_uri())
        member=artifacts.observation(value['path'],value['artifact_sha256'],observation)
        self.assertEqual('8'*32,member['instance'])
        with self.assertRaises(j.Conflict):artifacts.observation(value['path'],value['artifact_sha256'],dict(observation,source=self.source.as_uri()))

    def test_cli_explicit_repeated_authority_and_legacy_default(self):
        spec,roots=self.collection();sp=self.root/'spec.json';rp=self.root/'roots.json'
        sp.write_bytes(j.canonical(spec));rp.write_bytes(j.canonical(roots))
        command=[sys.executable,'-B',str(ROOT/'scripts/check-collection-runtime.py'),'--specification',str(sp),'--generations',str(rp)]
        for args,code,authority in (([],0,1),(['--authority-format','2'],0,2),(['--authority-format','3'],0,3),(['--authority-format','4'],0,4),(['--authority-format','5'],2,5)):
            q=subprocess.run(command+args,capture_output=True,text=True,timeout=10);self.assertEqual(code,q.returncode,q.stderr)
            self.assertEqual(authority,json.loads(q.stdout)['required']['authority'])
        old=ROOT/'tests/fixtures/collection-artifact-v6';meta=json.loads((old/'provenance.json').read_text())
        spec['members']['theme']['artifact_sha256']=meta['artifact_sha256'];roots['theme']=str(old/'generation')
        sp.write_bytes(j.canonical(spec));rp.write_bytes(j.canonical(roots))
        for args,code in (([],0),(['--authority-format','2'],0),(['--authority-format','3'],0),(['--authority-format','4'],2),(['--authority-format','0'],1)):
            q=subprocess.run(command+args,capture_output=True,text=True,timeout=10);self.assertEqual(code,q.returncode,q.stderr)


if __name__=='__main__':unittest.main()
