#!/usr/bin/env python3
"""Reusable checkpoint envelope excludes every ancestor body and survives retirement."""
import copy,importlib.util,json,os,sys,unittest,uuid
from pathlib import Path
from unittest.mock import patch
ROOT=Path(__file__).resolve().parents[1];sys.path.insert(0,str(ROOT/'native/collections'))
import repeated_checkpoint as checkpoint
import successor_checkpoint as original
import continuation_cleanup as cleanup
import continuation_inspection as inspection
import journal as j
import runtime_authority as authority
spec=importlib.util.spec_from_file_location('repeated_checkpoint_fixture',Path(__file__).with_name('test-collection-continuation-cleanup.py'))
f=importlib.util.module_from_spec(spec);spec.loader.exec_module(f)

def save(p,v):p.write_bytes(j.canonical(v)+b'\n');p.chmod(0o600)
def redigest(v):v['sha256']=j.digest({k:x for k,x in v.items() if k!='sha256'});return v

class RepeatedCheckpointTests(unittest.TestCase):
    def setUp(self):f.ContinuationCleanupTests.setUp(self)
    def clean_repeated_checkpoint(self):self.ready_continued_cleanup();self.clean_continued_step();self.clean_continued_step()
    def capture_repeated_checkpoint(self,root=None):
        with self.store.lease() as lease:return checkpoint.capture(root or self.continued_authority.root,self.store,lease,self.continued_cleanup_request)
    def read_repeated_checkpoint(self):
        with self.store.lease() as lease:return checkpoint.read(self.store,self.continued_cleanup_request,lease)
    def repeated_checkpoint_path(self):return self.store.root/checkpoint.filename(self.continued_cleanup_request)
    def lose_repeated_runtime(self):
        self.continued_authority.close();self.continued_authority.fd=None;self.runtime.rename(self.root/'third-retained-runtime');self.runtime.mkdir(mode=0o700)
    def repeated_metadata(self):
        return {k:v for k,v in self.snapshot().items() if checkpoint.PREFIX not in k and '.pending-desktop-repeat-checkpoint-' not in k}
    def refuse_repeated_read(self):
        before=self.snapshot()
        with self.assertRaises((ValueError,OSError)):self.read_repeated_checkpoint()
        self.assertEqual(before,self.snapshot())

    def test_capture_replay_then_runtime_loss_export(self):
        self.clean_repeated_checkpoint();before=self.repeated_metadata();value=self.capture_repeated_checkpoint();path=self.repeated_checkpoint_path();saved=(path.stat().st_ino,path.read_bytes())
        self.assertEqual(value['format'],3);ancestor=value['ancestor_checkpoint'];self.assertNotIn(ancestor['object']['path'],value['files']);self.assertEqual(len(value['inspection']['lineage']['ancestors']),1)
        for reference in [ancestor]+value['inspection']['lineage']['ancestors']:self.assertNotIn(reference['object']['path'],value['files'])
        self.assertEqual(value['inspection']['lineage']['checkpoint'],ancestor);self.assertEqual(self.capture_repeated_checkpoint(),value);self.assertEqual(self.repeated_metadata(),before)
        self.lose_repeated_runtime();before=self.snapshot();self.assertEqual(self.read_repeated_checkpoint(),value);self.assertEqual(self.capture_repeated_checkpoint(),value)
        self.assertEqual(before,self.snapshot());self.assertEqual(saved,(path.stat().st_ino,path.read_bytes()))
        with self.assertRaises(OSError):inspection.inspect(self.continued_authority.root,self.store.root)
        if ROOT==Path('/lab/source'):
            save(Path('/lab/repeated-checkpoint-result.json'),{'passed':True,'checkpoint':value,'runtime_retired':True,'ancestor_embedded':False,'all_ancestor_bodies_excluded':True,'ancestor_count':2,
                'completed_retry_unchanged':True,'new_admission':False,'application_observations':'synthetic'})
    def test_unverified_cleanup_cannot_capture(self):
        self.ready_continued_cleanup();self.clean_continued_step();before=self.snapshot()
        with self.assertRaises(ValueError):self.capture_repeated_checkpoint()
        self.assertEqual(before,self.snapshot())
    def test_missing_cleanup_is_not_created(self):
        self.ready_continued_cleanup();before=self.snapshot()
        with self.assertRaises(ValueError):self.capture_repeated_checkpoint()
        self.assertEqual(before,self.snapshot())
    def test_runtime_loss_before_capture_refuses_reconstruction(self):
        self.clean_repeated_checkpoint();self.lose_repeated_runtime();before=self.snapshot()
        with self.assertRaises(OSError):self.capture_repeated_checkpoint()
        self.assertEqual(before,self.snapshot())
    def test_original_reader_refuses_new_checkpoint(self):
        self.clean_repeated_checkpoint();value=self.capture_repeated_checkpoint()
        with self.assertRaises(ValueError):original.validate(value)
    def test_lost_publication_reply_preserves_original_checkpoint(self):
        self.clean_repeated_checkpoint();rename=checkpoint.provisioning.rename_no_replace
        def lost(*args):rename(*args);raise OSError('reply lost')
        with patch.object(checkpoint.provisioning,'rename_no_replace',lost),self.assertRaises(OSError):self.capture_repeated_checkpoint()
        path=self.repeated_checkpoint_path();saved=(path.stat().st_ino,path.read_bytes());self.lose_repeated_runtime();self.capture_repeated_checkpoint()
        self.assertEqual(saved,(path.stat().st_ino,path.read_bytes()))
    def test_failed_publication_retains_unbound_stage(self):
        self.clean_repeated_checkpoint()
        with patch.object(checkpoint.provisioning,'rename_no_replace',side_effect=OSError('publication failed')),self.assertRaises(OSError):self.capture_repeated_checkpoint()
        self.assertFalse(self.repeated_checkpoint_path().exists());stages=list(self.store.root.glob('.pending-desktop-repeat-checkpoint-*'));self.assertEqual(len(stages),1)
        saved=stages[0].read_bytes();self.capture_repeated_checkpoint();self.assertEqual(saved,stages[0].read_bytes())
    def test_foreign_destination_never_overwritten(self):
        self.clean_repeated_checkpoint();rename=checkpoint.provisioning.rename_no_replace
        def conflict(fd,source,target):authority.write_new(fd,target,{'foreign':True});rename(fd,source,target)
        with patch.object(checkpoint.provisioning,'rename_no_replace',conflict),self.assertRaises(ValueError):self.capture_repeated_checkpoint()
        self.assertEqual(json.loads(self.repeated_checkpoint_path().read_text()),{'foreign':True})
    def test_missing_ancestor_refuses_after_runtime_loss(self):
        self.clean_repeated_checkpoint();self.capture_repeated_checkpoint();self.lose_repeated_runtime();self.cleanup_ancestor_checkpoint.unlink();self.refuse_repeated_read()
    def test_identical_bytes_new_ancestor_inode_refuses(self):
        self.clean_repeated_checkpoint();self.capture_repeated_checkpoint();p=self.cleanup_ancestor_checkpoint;raw=p.read_bytes();p.rename(p.with_suffix('.retained'));p.write_bytes(raw);p.chmod(0o600);self.refuse_repeated_read()
    def test_missing_current_cleanup_refuses_after_runtime_loss(self):
        self.clean_repeated_checkpoint();self.capture_repeated_checkpoint();self.lose_repeated_runtime();self.continued_cleanup_path().unlink();self.refuse_repeated_read()
    def test_missing_predecessor_cleanup_refuses(self):
        self.clean_repeated_checkpoint();self.capture_repeated_checkpoint();Path(self.second_archived['cleanup']['object']['path']).unlink();self.refuse_repeated_read()
    def test_missing_canonical_publication_or_mirror_refuses(self):
        self.clean_repeated_checkpoint();self.capture_repeated_checkpoint();self.activation_record_path().unlink();self.refuse_repeated_read()
    def test_changed_canonical_prefix_refuses(self):
        self.clean_repeated_checkpoint();self.capture_repeated_checkpoint();p=self.store.root/'journal.json';v=json.loads(p.read_text());v['events']=v['events'][:1];save(p,v);self.refuse_repeated_read()
    def test_later_journal_append_preserves_checkpoint(self):
        self.clean_repeated_checkpoint();value=self.capture_repeated_checkpoint();self.record({'type':'begin','request':uuid.uuid4().hex,'candidate':j.digest(self.continued_spec),'expected':j.digest(self.cleanup_current_loaded)})
        self.assertEqual(self.read_repeated_checkpoint(),value)
    def test_unknown_format_preserved(self):
        self.clean_repeated_checkpoint();value=self.capture_repeated_checkpoint();value['format']=99;save(self.repeated_checkpoint_path(),redigest(value));self.refuse_repeated_read()
    def test_modified_copied_bytes_refuse_even_with_outer_digest(self):
        self.clean_repeated_checkpoint();value=self.capture_repeated_checkpoint();name=next(iter(value['files']));value['files'][name]['text']='{}'
        with self.assertRaises(ValueError):checkpoint.validate(redigest(value))
    def test_ancestor_cannot_be_embedded_as_copied_source(self):
        self.clean_repeated_checkpoint();value=self.capture_repeated_checkpoint();ref=value['ancestor_checkpoint'];value['files'][ref['object']['path']]=dict(ref,text=self.cleanup_ancestor_checkpoint.read_text())
        with self.assertRaises(ValueError):checkpoint.validate(redigest(value))
    def test_inventory_bound_refuses_before_publication(self):
        self.clean_repeated_checkpoint()
        with patch.object(checkpoint,'MAX_SOURCE_BYTES',1),self.assertRaises(ValueError):self.capture_repeated_checkpoint()
        self.assertFalse(self.repeated_checkpoint_path().exists())
    def test_wrong_root_or_expired_lease_refuses_retry(self):
        self.clean_repeated_checkpoint();self.capture_repeated_checkpoint();before=self.snapshot()
        with self.assertRaises(ValueError):self.capture_repeated_checkpoint(self.root/'wrong'/authority.ANCHOR)
        with self.store.lease() as lease:pass
        with self.assertRaises(ValueError):checkpoint.read(self.store,self.continued_cleanup_request,lease)
        self.assertEqual(before,self.snapshot())

    def test_grandparent_body_cannot_be_embedded(self):
        self.clean_repeated_checkpoint();value=self.capture_repeated_checkpoint();reference=value['inspection']['lineage']['ancestors'][0]
        value['files'][reference['object']['path']]=dict(reference,text=Path(reference['object']['path']).read_text())
        with self.assertRaises(ValueError):checkpoint.validate(redigest(value))
    def test_missing_grandparent_refuses_after_runtime_loss(self):
        self.clean_repeated_checkpoint();value=self.capture_repeated_checkpoint();self.lose_repeated_runtime()
        Path(value['inspection']['lineage']['ancestors'][0]['object']['path']).unlink();self.refuse_repeated_read()
    def test_common_reader_traverses_all_three_checkpoint_versions(self):
        self.clean_repeated_checkpoint();value=self.capture_repeated_checkpoint()
        chain=checkpoint.checkpoint_lineage.read(checkpoint.successor.reference(self.repeated_checkpoint_path()),self.store.root)
        self.assertEqual([r['format'] for r in chain['records']],[3,2,1])
        self.assertEqual(chain['head'],value);self.assertEqual(chain['references'][2:],value['inspection']['lineage']['ancestors'])

for name in dir(f.ContinuationCleanupTests):
    value=getattr(f.ContinuationCleanupTests,name)
    if callable(value) and not name.startswith(('test','_')) and not hasattr(RepeatedCheckpointTests,name):setattr(RepeatedCheckpointTests,name,value)
if __name__=='__main__':unittest.main()
