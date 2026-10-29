#!/usr/bin/env python3
"""Linked checkpoints survive runtime retirement without embedding ancestors."""
import copy,importlib.util,json,os,sys,unittest,uuid
from pathlib import Path
from unittest.mock import patch
ROOT=Path(__file__).resolve().parents[1];sys.path.insert(0,str(ROOT/'native/collections'))
import successor_checkpoint as checkpoint
import session_checkpoint as original
import successor_cleanup as cleanup
import successor_inspection as inspection
import journal as j
import runtime_authority as authority
spec=importlib.util.spec_from_file_location('linked_checkpoint_fixture',Path(__file__).with_name('test-collection-successor-cleanup.py'))
f=importlib.util.module_from_spec(spec);spec.loader.exec_module(f)

def save(p,v):p.write_bytes(j.canonical(v)+b'\n');p.chmod(0o600)
def redigest(v):v['sha256']=j.digest({k:x for k,x in v.items() if k!='sha256'});return v

class LinkedCheckpointTests(unittest.TestCase):
    def setUp(self):f.LinkedCleanupTests.setUp(self)
    def clean_linked(self):self.ready_cleanup();self.clean_step();self.clean_step()
    def capture_linked(self,root=None):
        with self.store.lease() as lease:return checkpoint.capture(root or self.next_authority.root,self.store,lease,self.request)
    def read_linked(self):
        with self.store.lease() as lease:return checkpoint.read(self.store,self.request,lease)
    def linked_checkpoint_path(self):return self.store.root/checkpoint.filename(self.request)
    def lose_linked_runtime(self):
        self.next_authority.close();self.next_authority.fd=None;self.runtime.rename(self.root/'second-retained-runtime');self.runtime.mkdir(mode=0o700)
    def metadata(self):
        return {k:v for k,v in self.snapshot().items() if checkpoint.PREFIX not in k and '.pending-desktop-successor-checkpoint-' not in k}
    def refuse_read(self):
        before=self.snapshot()
        with self.assertRaises((ValueError,OSError)):self.read_linked()
        self.assertEqual(before,self.snapshot())

    def test_capture_replay_then_runtime_loss_export(self):
        self.clean_linked();before=self.metadata();value=self.capture_linked();path=self.linked_checkpoint_path();saved=(path.stat().st_ino,path.read_bytes())
        self.assertEqual(value['format'],2);ancestor=value['ancestor_checkpoint'];self.assertNotIn(ancestor['object']['path'],value['files'])
        self.assertEqual(value['inspection']['lineage']['checkpoint'],ancestor);self.assertEqual(self.capture_linked(),value);self.assertEqual(self.metadata(),before)
        self.lose_linked_runtime();before=self.snapshot();self.assertEqual(self.read_linked(),value);self.assertEqual(self.capture_linked(),value)
        self.assertEqual(before,self.snapshot());self.assertEqual(saved,(path.stat().st_ino,path.read_bytes()))
        with self.assertRaises(OSError):inspection.inspect(self.next_authority.root,self.store.root)
        if ROOT==Path('/lab/source'):
            save(Path('/lab/successor-checkpoint-result.json'),{'passed':True,'checkpoint':value,'runtime_retired':True,'ancestor_embedded':False,
                'completed_retry_unchanged':True,'new_admission':False,'application_observations':'synthetic'})
    def test_unverified_cleanup_cannot_capture(self):
        self.ready_cleanup();self.clean_step();before=self.snapshot()
        with self.assertRaises(ValueError):self.capture_linked()
        self.assertEqual(before,self.snapshot())
    def test_missing_cleanup_is_not_created(self):
        self.ready_cleanup();before=self.snapshot()
        with self.assertRaises(ValueError):self.capture_linked()
        self.assertEqual(before,self.snapshot())
    def test_runtime_loss_before_capture_refuses_reconstruction(self):
        self.clean_linked();self.lose_linked_runtime();before=self.snapshot()
        with self.assertRaises(OSError):self.capture_linked()
        self.assertEqual(before,self.snapshot())
    def test_original_reader_refuses_new_checkpoint(self):
        self.clean_linked();value=self.capture_linked()
        with self.assertRaises(ValueError):original.validate(value)
    def test_lost_publication_reply_preserves_original_checkpoint(self):
        self.clean_linked();rename=checkpoint.provisioning.rename_no_replace
        def lost(*args):rename(*args);raise OSError('reply lost')
        with patch.object(checkpoint.provisioning,'rename_no_replace',lost),self.assertRaises(OSError):self.capture_linked()
        path=self.linked_checkpoint_path();saved=(path.stat().st_ino,path.read_bytes());self.lose_linked_runtime();self.capture_linked()
        self.assertEqual(saved,(path.stat().st_ino,path.read_bytes()))
    def test_failed_publication_retains_unbound_stage(self):
        self.clean_linked()
        with patch.object(checkpoint.provisioning,'rename_no_replace',side_effect=OSError('publication failed')),self.assertRaises(OSError):self.capture_linked()
        self.assertFalse(self.linked_checkpoint_path().exists());stages=list(self.store.root.glob('.pending-desktop-successor-checkpoint-*'));self.assertEqual(len(stages),1)
        saved=stages[0].read_bytes();self.capture_linked();self.assertEqual(saved,stages[0].read_bytes())
    def test_foreign_destination_never_overwritten(self):
        self.clean_linked();rename=checkpoint.provisioning.rename_no_replace
        def conflict(fd,source,target):authority.write_new(fd,target,{'foreign':True});rename(fd,source,target)
        with patch.object(checkpoint.provisioning,'rename_no_replace',conflict),self.assertRaises(ValueError):self.capture_linked()
        self.assertEqual(json.loads(self.linked_checkpoint_path().read_text()),{'foreign':True})
    def test_missing_ancestor_refuses_after_runtime_loss(self):
        self.clean_linked();self.capture_linked();self.lose_linked_runtime();self.ancestor_checkpoint.unlink();self.refuse_read()
    def test_identical_bytes_new_ancestor_inode_refuses(self):
        self.clean_linked();self.capture_linked();p=self.ancestor_checkpoint;raw=p.read_bytes();p.rename(p.with_suffix('.retained'));p.write_bytes(raw);p.chmod(0o600);self.refuse_read()
    def test_missing_current_cleanup_refuses_after_runtime_loss(self):
        self.clean_linked();self.capture_linked();self.lose_linked_runtime();self.cleanup_path().unlink();self.refuse_read()
    def test_missing_predecessor_cleanup_refuses(self):
        self.clean_linked();self.capture_linked();Path(self.archived['cleanup']['object']['path']).unlink();self.refuse_read()
    def test_missing_canonical_publication_or_mirror_refuses(self):
        self.clean_linked();self.capture_linked();self.record_path().unlink();self.refuse_read()
    def test_changed_canonical_prefix_refuses(self):
        self.clean_linked();self.capture_linked();p=self.store.root/'journal.json';v=json.loads(p.read_text());v['events']=v['events'][:1];save(p,v);self.refuse_read()
    def test_later_journal_append_preserves_checkpoint(self):
        self.clean_linked();value=self.capture_linked();self.record({'type':'begin','request':uuid.uuid4().hex,'candidate':j.digest(self.next_spec),'expected':j.digest(self.replacement)})
        self.assertEqual(self.read_linked(),value)
    def test_unknown_format_preserved(self):
        self.clean_linked();value=self.capture_linked();value['format']=99;save(self.linked_checkpoint_path(),redigest(value));self.refuse_read()
    def test_modified_copied_bytes_refuse_even_with_outer_digest(self):
        self.clean_linked();value=self.capture_linked();name=next(iter(value['files']));value['files'][name]['text']='{}'
        with self.assertRaises(ValueError):checkpoint.validate(redigest(value))
    def test_ancestor_cannot_be_embedded_as_copied_source(self):
        self.clean_linked();value=self.capture_linked();ref=value['ancestor_checkpoint'];value['files'][ref['object']['path']]=dict(ref,text=self.ancestor_checkpoint.read_text())
        with self.assertRaises(ValueError):checkpoint.validate(redigest(value))
    def test_inventory_bound_refuses_before_publication(self):
        self.clean_linked()
        with patch.object(checkpoint,'MAX_SOURCE_BYTES',1),self.assertRaises(ValueError):self.capture_linked()
        self.assertFalse(self.linked_checkpoint_path().exists())
    def test_wrong_root_or_expired_lease_refuses_retry(self):
        self.clean_linked();self.capture_linked();before=self.snapshot()
        with self.assertRaises(ValueError):self.capture_linked(self.root/'wrong'/authority.ANCHOR)
        with self.store.lease() as lease:pass
        with self.assertRaises(ValueError):checkpoint.read(self.store,self.request,lease)
        self.assertEqual(before,self.snapshot())

for name in dir(f.LinkedCleanupTests):
    value=getattr(f.LinkedCleanupTests,name)
    if callable(value) and not name.startswith(('test','_')) and not hasattr(LinkedCheckpointTests,name):setattr(LinkedCheckpointTests,name,value)
if __name__=='__main__':unittest.main()
