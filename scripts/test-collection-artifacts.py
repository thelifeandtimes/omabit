#!/usr/bin/env python3
"""Preparation never enables a plugin; interruptions retain old generations."""
import copy
import json
import os
from pathlib import Path
import signal
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch

sys.path.insert(0,str(Path(__file__).resolve().parents[1]/'native/collections'))
import artifacts as a
import journal as j

ENV = {f'XDG_{name.upper()}_HOME':'/isolated/'+name for name in ('config','state','data','cache')}


def inputs(root, role='theme', version='1.0.0'):
    canonical, kind, entry = a.ROLES[role]
    root.mkdir()
    manifest = {'schemaVersion':1,'id':canonical,'name':'Test '+role,'version':version,
                'kinds':[kind,'bar-widget'],'entryPoints':{kind:entry,'barWidget':'Panel.qml'},
                'barWidget':{'displayName':'Test '+role,'defaultSection':'right'}}
    (root/'manifest.json').write_text(json.dumps(manifest))
    for name in (entry,'Panel.qml'):
        (root/name).write_text('import QtQuick\nItem { property var manifest; property var shell; property var transportState: ({closed:true}); function stopTransport() {return "requested"} }\n')
    (root/'helper.py').write_text('print("pinned helper '+version+'")\n')
    if role=='omabit':
        (root/'bin').mkdir()
        for name in ('omabit','urbit-collector','urbit-runner'):
            p=root/'bin'/name;p.write_text('#!/bin/sh\nexit 0\n');p.chmod(0o700)
        (root/'DESKTOP.json').write_text('{"bootstrap_api":1,"profile_state_schema":1}')
    return a.inventory(a.snapshot(root,sorted(str(p.relative_to(root)) for p in root.rglob('*') if p.is_file())))


class ArtifactTests(unittest.TestCase):
    def setUp(self):
        self.temp=tempfile.TemporaryDirectory(prefix='omabit-artifact-unit-')
        self.base=Path(self.temp.name);self.source=self.base/'input';self.store=self.base/'generations';self.store.mkdir(mode=0o700)
        self.pin=inputs(self.source)
        self.addCleanup(self.clean)

    def clean(self):
        for root,dirs,files in os.walk(self.base):
            os.chmod(root,0o700)
        self.temp.cleanup()

    def prepare(self, **changes):
        args=dict(role='theme',profile='qualification-stable',environment=None);args.update(changes)
        return a.prepare(self.source,self.pin,self.store,**args)

    def test_exact_retry_is_same_bytes_and_never_publishes_native_manifest(self):
        value=self.prepare();root=Path(value['path']);before=a.snapshot(root,sorted(str(p.relative_to(root)) for p in root.rglob('*') if p.is_file()))
        self.assertEqual(value,self.prepare());self.assertEqual(before,a.snapshot(root,list(before)))
        self.assertFalse((self.store.parent/'manifest.json').exists())
        self.assertEqual(value['manifest']['id'],'thelifeandtimes.urbit-theme.qualification-stable')
        self.assertEqual(value['manifest']['entryPoints']['service'],'generations/'+value['artifact_sha256']+'/CollectionRuntime.qml')
        self.assertEqual(json.loads((root/'application-profile.json').read_text()),{'version':1,'profile':'qualification-stable'})
        self.assertIn('qualification-stable',value['manifest']['barWidget']['displayName'])

    def test_every_runtime_url_changes_and_old_generation_is_retained(self):
        old=self.prepare();oldbytes=(Path(old['path'])/'helper.py').read_bytes()
        (self.source/'helper.py').write_text('print("new")\n');self.pin=a.inventory(a.snapshot(self.source,list(self.pin)))
        new=self.prepare();self.assertNotEqual(old['artifact_sha256'],new['artifact_sha256'])
        self.assertTrue(all(new['manifest']['entryPoints'][k]!=v for k,v in old['manifest']['entryPoints'].items()))
        self.assertEqual((Path(old['path'])/'helper.py').read_bytes(),oldbytes);a.verify(old['path'],old['artifact_sha256'])

    def test_profile_changes_artifact_names_labels_and_receiver_profile(self):
        stable=self.prepare();dev=self.prepare(profile='qualification-dev')
        self.assertNotEqual(stable['artifact_sha256'],dev['artifact_sha256']);self.assertNotEqual(stable['manifest']['id'],dev['manifest']['id'])
        self.assertNotEqual(stable['manifest']['name'],dev['manifest']['name'])
        with self.assertRaises(j.Conflict):self.prepare(profile='default')

    def test_changed_source_refuses_before_creating_generation(self):
        (self.source/'helper.py').write_text('tampered')
        with self.assertRaises(j.Conflict):self.prepare()
        self.assertEqual(list(self.store.iterdir()),[])

    def test_unlisted_source_data_is_never_copied(self):
        (self.source/'credentials.json').write_text('unrelated sentinel')
        value=self.prepare();self.assertFalse((Path(value['path'])/'credentials.json').exists())

    def test_linked_parent_file_hardlink_and_path_escape_refuse(self):
        original=self.source/'helper.py';original.unlink();original.symlink_to(self.source/'Panel.qml')
        with self.assertRaises(OSError):self.prepare()
        original.unlink();os.link(self.source/'Panel.qml',original)
        with self.assertRaises(j.Conflict):self.prepare()
        original.unlink();original.write_text('print("pinned helper 1.0.0")\n')
        parent=self.base/'alias';parent.symlink_to(self.source,target_is_directory=True)
        with self.assertRaises(OSError):a.snapshot(parent,list(self.pin))
        for name in ('../input/manifest.json','/manifest.json','foo//bar','foo/./bar','foo\\bar','.secret'):
            with self.subTest(name=name),self.assertRaises(j.Conflict):a.snapshot(self.source,[name])

    def test_wrapper_payload_and_extra_file_tampering_refuse(self):
        for part in ('CollectionRuntime.qml','helper.py','extra.qml'):
            with self.subTest(part=part):
                value=self.prepare(profile='test-'+str(len(list(self.store.iterdir()))));root=Path(value['path']);os.chmod(root,0o700)
                target=root/part
                if target.exists():target.chmod(0o600)
                target.write_text('changed')
                with self.assertRaises(j.Conflict):a.verify(root,value['artifact_sha256'])

    def test_descriptor_format_or_pins_cannot_authorize_other_bytes(self):
        value=self.prepare();root=Path(value['path']);meta=root/a.META;meta.chmod(0o600)
        descriptor=json.loads(meta.read_text());descriptor['format']=99;meta.write_text(json.dumps(descriptor))
        with self.assertRaises(j.Conflict):a.verify(root,value['artifact_sha256'])
        with self.assertRaises(j.Conflict):a.verify(root,j.digest(descriptor))

    def test_interrupted_rename_reply_retry_reverifies_and_syncs(self):
        real=a._publish
        def lost(stage,final):real(stage,final);raise OSError('lost publication response')
        with patch.object(a,'_publish',lost),self.assertRaises(OSError):self.prepare()
        with patch.object(a,'_publish',side_effect=AssertionError('must not publish again')),patch.object(a,'_sync',wraps=a._sync) as sync:
            value=self.prepare();sync.assert_called_with(self.store)
        self.assertTrue(Path(value['path']).is_dir())

    def test_actual_prepare_process_death_never_exposes_partial_generation(self):
        code='''import json,sys,time
from pathlib import Path
sys.path.insert(0,sys.argv[1]);import artifacts as a
def stopped(stage,final):
 print(str(stage),flush=True)
 while True:time.sleep(.02)
a._publish=stopped
a.prepare(Path(sys.argv[2]),json.loads(sys.argv[3]),Path(sys.argv[4]),role='theme',profile='qualification-stable',environment=json.loads(sys.argv[5]))
'''
        child=subprocess.Popen([sys.executable,'-B','-c',code,str(Path(a.__file__).parent),str(self.source),json.dumps(self.pin),str(self.store),json.dumps(None)],stdout=subprocess.PIPE,text=True)
        try:
            import select
            self.assertTrue(select.select([child.stdout],[],[],5)[0]);stage=Path(child.stdout.readline().strip());self.assertTrue(stage.name.startswith('.preparing-'))
            child.send_signal(signal.SIGKILL);self.assertEqual(child.wait(timeout=3),-signal.SIGKILL)
        finally:
            if child.poll() is None:child.kill();child.wait(timeout=3)
            child.stdout.close()
        self.assertFalse(any(p.name[0]!='.' for p in self.store.iterdir()))
        value=self.prepare();self.assertTrue(stage.is_dir());self.assertTrue(Path(value['path']).is_dir())
        self.assertEqual(a.verify(stage,value['artifact_sha256'])['descriptor'],value['descriptor'])

    def test_dms_and_omabit_require_supported_layout_and_paired_tools(self):
        for role in ('dms','omabit'):
            source=self.base/role;pin=inputs(source,role)
            result=a.prepare(source,pin,self.store,role=role,profile='qualification-dev',environment=ENV if role=='omabit' else None)
            self.assertIn(a.ROLES[role][1],result['manifest']['entryPoints'])
            if role=='omabit':
                body=(Path(result['path'])/a.WRAPPER).read_text();self.assertIn('localSetupAllowed: false',body);self.assertIn('profileEnvironment:',body)
                del pin['bin/urbit-runner']
                with self.assertRaises(j.Conflict):a.prepare(source,pin,self.store,role=role,profile='qualification-dev',environment=ENV if role=='omabit' else None)

    def test_loaded_observation_requires_exact_artifact_profile_instance_and_url(self):
        record=self.prepare();root=Path(record['path'])
        value=dict(format=1,plugin_id=record['manifest']['id'],profile='qualification-stable',
                   artifact_sha256=record['artifact_sha256'],instance='a'*32,source=(root/a.WRAPPER).as_uri())
        result=a.observation(root,record['artifact_sha256'],value)
        self.assertEqual(result,{k:value[k] for k in ('plugin_id','profile','artifact_sha256','instance')})
        for field,replacement in (('format',True),('plugin_id','other'),('profile','qualification-dev'),
                                  ('artifact_sha256','b'*64),('instance',''),('source',(root/'Service.qml').as_uri())):
            with self.subTest(field=field),self.assertRaises(j.Conflict):
                a.observation(root,record['artifact_sha256'],dict(value,**{field:replacement}))

    def test_literal_template_tokens_in_profile_paths_are_not_rewritten(self):
        source=self.base/'omabit';pin=inputs(source,'omabit')
        env={k:v+'/PROFILE-PLUGIN-TYPE-ARTIFACT' for k,v in ENV.items()}
        result=a.prepare(source,pin,self.store,role='omabit',profile='qualification-dev',environment=env)
        text=(Path(result['path'])/a.WRAPPER).read_text()
        self.assertIn(j.canonical(env).decode(),text)

    def test_bad_contract_or_overlapping_profile_roots_refuse(self):
        with self.assertRaises(j.Conflict):self.prepare(environment=ENV)
        env=dict(ENV,XDG_STATE_HOME=ENV['XDG_CONFIG_HOME'])
        with self.assertRaises(j.Conflict):a.settings('omabit','qualification-dev',env)
        m=json.loads((self.source/'manifest.json').read_text());m['entryPoints']['service']='native/v1/Service.qml';(self.source/'manifest.json').write_text(json.dumps(m))
        self.pin=a.inventory(a.snapshot(self.source,list(self.pin)))
        with self.assertRaises(j.Conflict):self.prepare()


if __name__=='__main__':unittest.main()
