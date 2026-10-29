#!/usr/bin/env python3
"""Pinned inert hosts retain original artifact history and refuse drift."""
import hashlib,importlib.util,json,os,select,signal,subprocess,sys,tempfile,unittest
from pathlib import Path
from unittest.mock import patch
ROOT=Path(__file__).resolve().parents[1]
sys.path.insert(0,str(ROOT/'native/collections'))
import artifacts as a
import guarded_artifacts as g
import journal as j
spec=importlib.util.spec_from_file_location('artifact_test_inputs',Path(__file__).with_name('test-collection-artifacts.py'))
base=importlib.util.module_from_spec(spec);spec.loader.exec_module(base)

class GuardedArtifactTests(unittest.TestCase):
 def setUp(self):
  self.temp=tempfile.TemporaryDirectory(prefix='omabit-guarded-unit-');self.root=Path(self.temp.name)
  self.source=self.root/'input';self.pin=base.inputs(self.source);self.store=self.root/'generations';self.store.mkdir(mode=0o700)
  self.support=ROOT/'native/collections';self.support_pin=a.inventory(a.snapshot(self.support,list(g.SUPPORT)));self.addCleanup(self.clean)
 def clean(self):
  for root,_,_ in os.walk(self.root):os.chmod(root,0o700)
  self.temp.cleanup()
 def prepare(self,**kw):
  args=dict(role='theme',profile='qualification-stable',environment=None,admission_source=self.support,expected_admission=self.support_pin);args.update(kw)
  return g.prepare(self.source,self.pin,self.store,**args)
 def test_guard_and_widgets_are_pinned_in_same_immutable_generation(self):
  v=self.prepare();root=Path(v['path']);self.assertEqual(v,self.prepare());self.assertTrue(v['admission_capable']);self.assertEqual(v['descriptor']['format'],2)
  for kind,name in [('service',g.HOST),('barWidget',g.WIDGET)]:self.assertEqual(v['manifest']['entryPoints'][kind],'generations/'+v['artifact_sha256']+'/'+name)
  self.assertEqual(a.inventory(a.snapshot(root/'collection',list(g.SUPPORT))),self.support_pin)
  self.assertNotIn('Runtime.Panel',(root/g.WIDGET).read_text());self.assertNotIn('Runtime.Service',(root/g.HOST).read_text())
  self.assertFalse((self.store.parent/'manifest.json').exists())
 def test_original_writer_reproduces_and_old_reader_refuses_successor(self):
  f=ROOT/'tests/fixtures/collection-artifact-v1';meta=json.loads((f/'provenance.json').read_text())
  for name,h in meta['sha256'].items():self.assertEqual(hashlib.sha256((f/name).read_bytes()).hexdigest(),h)
  spec=importlib.util.spec_from_file_location('original_artifact_writer',f/'writer.py');old=importlib.util.module_from_spec(spec);spec.loader.exec_module(old)
  v=old.prepare(f/'source',meta['source'],self.store,role=meta['role'],profile=meta['profile'],environment=meta['environment'])
  self.assertEqual(v['artifact_sha256'],meta['artifact_sha256']);root=Path(v['path'])
  before=a.snapshot(root,sorted(str(p.relative_to(root)) for p in root.rglob('*') if p.is_file()))
  self.assertEqual(before,a.snapshot(f/'generation',list(before)))
  self.assertFalse(g.verify(root,v['artifact_sha256'])['admission_capable'])
  with self.assertRaises(j.Conflict):g.observation(root,v['artifact_sha256'],{})
  self.assertEqual(before,a.snapshot(root,list(before)))
  new=self.prepare();newroot=Path(new['path']);newbefore=a.snapshot(newroot,sorted(str(p.relative_to(newroot)) for p in newroot.rglob('*') if p.is_file()))
  with self.assertRaises(j.Conflict):old.verify(newroot,new['artifact_sha256'])
  self.assertEqual(newbefore,a.snapshot(newroot,list(newbefore)));g.verify(newroot,new['artifact_sha256'])
 def test_changed_admission_pin_refuses_before_any_publication(self):
  pin=json.loads(json.dumps(self.support_pin));pin['Host.qml']['sha256']='0'*64
  with self.assertRaises(j.Conflict):self.prepare(expected_admission=pin)
  self.assertEqual(list(self.store.iterdir()),[])
  del pin['Host.qml']
  with self.assertRaises(j.Conflict):self.prepare(expected_admission=pin)
 def test_generated_primary_widget_helper_or_registry_drift_refuse(self):
  for n,name in enumerate((g.HOST,g.WIDGET,a.WRAPPER,'collection/admission_guard.py','collection/Registry.qml','collection/qmldir')):
   v=self.prepare(profile='drift-'+str(n));p=Path(v['path'])/name;p.chmod(0o600);p.write_text('changed')
   with self.subTest(name=name),self.assertRaises(j.Conflict):g.verify(v['path'],v['artifact_sha256'])
 def test_support_change_creates_distinct_generation_retains_original(self):
  old=self.prepare();other=self.root/'support';other.mkdir()
  for name in g.SUPPORT:(other/name).write_bytes((self.support/name).read_bytes())
  p=other/'Host.qml';p.write_text(p.read_text()+'\n// revised host\n')
  pin=a.inventory(a.snapshot(other,list(g.SUPPORT)));new=self.prepare(admission_source=other,expected_admission=pin)
  self.assertNotEqual(old['artifact_sha256'],new['artifact_sha256']);g.verify(old['path'],old['artifact_sha256'])
 def test_reserved_paths_extra_entries_and_unknown_format_refuse(self):
  (self.source/g.HOST).write_text('untrusted override');self.pin=a.inventory(a.snapshot(self.source,list(self.pin)+[g.HOST]))
  with self.assertRaises(j.Conflict):self.prepare()
  del self.pin[g.HOST]
  m=json.loads((self.source/'manifest.json').read_text());m['entryPoints']['shortcut']='Panel.qml';(self.source/'manifest.json').write_text(json.dumps(m));self.pin=a.inventory(a.snapshot(self.source,list(self.pin)))
  with self.assertRaises(j.Conflict):self.prepare()
 def test_publication_lost_reply_reverifies_without_new_generation(self):
  original=a._publish
  def fail(stage,final):original(stage,final);raise OSError('lost reply')
  with patch.object(a,'_publish',fail),self.assertRaises(OSError):self.prepare()
  with patch.object(a,'_publish',side_effect=AssertionError('republished')),patch.object(a,'_sync',wraps=a._sync) as sync:
   v=self.prepare();sync.assert_called_with(self.store)
  self.assertTrue(g.verify(v['path'],v['artifact_sha256'])['admission_capable'])
 def test_process_death_before_publish_retains_stage_and_retry(self):
  code='''import json,sys,time
from pathlib import Path
sys.path.insert(0,sys.argv[1]);import artifacts as a;import guarded_artifacts as g
def stop(stage,final):
 print(str(stage),flush=True)
 while True:time.sleep(.03)
a._publish=stop
g.prepare(sys.argv[2],json.loads(sys.argv[3]),sys.argv[4],role='theme',profile='qualification-stable',environment=None,admission_source=sys.argv[1],expected_admission=json.loads(sys.argv[5]))
'''
  child=subprocess.Popen([sys.executable,'-B','-c',code,str(self.support),str(self.source),json.dumps(self.pin),str(self.store),json.dumps(self.support_pin)],stdout=subprocess.PIPE,text=True)
  try:
   self.assertTrue(select.select([child.stdout],[],[],5)[0]);stage=Path(child.stdout.readline().strip());self.assertTrue(stage.name.startswith('.preparing-'))
   child.send_signal(signal.SIGKILL);self.assertEqual(child.wait(timeout=3),-signal.SIGKILL)
  finally:
   if child.poll() is None:child.kill();child.wait(timeout=3)
   child.stdout.close()
  self.assertFalse(any(p.name[0]!='.' for p in self.store.iterdir()));v=self.prepare();self.assertTrue(stage.is_dir());g.verify(stage,v['artifact_sha256'])
 def test_omabit_environment_and_all_supported_roles(self):
  for role in ('omabit','dms'):
   source=self.root/role;pin=base.inputs(source,role)
   v=g.prepare(source,pin,self.store,role=role,profile='qualification-dev',environment=base.ENV if role=='omabit' else None,admission_source=self.support,expected_admission=self.support_pin)
   self.assertTrue(g.verify(v['path'],v['artifact_sha256'])['admission_capable'])
   if role=='omabit':self.assertIn(j.canonical(base.ENV).decode(),(Path(v['path'])/a.WRAPPER).read_text())
 def test_observation_binds_actual_admitted_runtime_path(self):
  v=self.prepare();root=Path(v['path']);value=dict(format=1,plugin_id=v['manifest']['id'],profile=v['descriptor']['profile'],artifact_sha256=v['artifact_sha256'],instance='a'*32,source=(root/a.WRAPPER).as_uri())
  self.assertEqual(g.observation(root,v['artifact_sha256'],value)['instance'],'a'*32)
  with self.assertRaises(j.Conflict):g.observation(root,v['artifact_sha256'],dict(value,source=(root/g.HOST).as_uri()))

if __name__=='__main__':unittest.main()
