#!/usr/bin/env python3
"""Continuation desktop cleanup retains linked ownership, state and reply-loss evidence."""
import copy,importlib.util,json,os,sys,unittest,uuid
from pathlib import Path
from unittest.mock import patch
ROOT=Path(__file__).resolve().parents[1];sys.path.insert(0,str(ROOT/'native/collections'))
import continuation_cleanup as cleanup
import successor_cleanup as previous
import session_cleanup as original
import continuation_inspection as inspection
import journal as j
import runtime_authority as authority
spec=importlib.util.spec_from_file_location('continued_cleanup_fixture',Path(__file__).with_name('test-collection-continuation-inspection.py'))
f=importlib.util.module_from_spec(spec);spec.loader.exec_module(f)
spec=importlib.util.spec_from_file_location('original_cleanup_fixture',Path(__file__).with_name('test-collection-session-cleanup.py'))
a=importlib.util.module_from_spec(spec);spec.loader.exec_module(a)

class ContinuationCleanupTests(unittest.TestCase):
    def setUp(self):
        f.ContinuationInspectionTests.setUp(self);self.cleanup_current_loaded=self.bound_continued()
        self.cleanup_ancestor_checkpoint=self.linked_checkpoint_path();self.continued_cleanup_request=uuid.uuid4().hex;self.cleanup_adapter=a.Adapter()
    def clean_continued_step(self,request=None):
        with self.store.lease() as lease:return cleanup.step(self.continued_authority.root,self.store,lease,self.cleanup_adapter,request or self.continued_cleanup_request)
    def ready_continued_cleanup(self):self.stop_display();self.close_guardian()
    def continued_cleanup_path(self):return self.store.root/cleanup.filename(self.continued_cleanup_request)
    def continued_core_snapshot(self):
        return {k:v for k,v in self.snapshot().items() if cleanup.PREFIX not in k and '.pending-desktop-chain-cleanup-' not in k}
    def refuse_continued_cleanup(self):
        before=self.snapshot()
        with self.assertRaises((ValueError,OSError)):self.clean_continued_step()
        self.assertEqual(before,self.snapshot())

    def test_prepare_verify_replay_preserves_ancestors_and_current_history(self):
        self.ready_continued_cleanup();before=self.continued_core_snapshot();prepared=self.clean_continued_step()
        self.assertEqual(prepared['format'],3);self.assertEqual(prepared['plan']['loaded'],self.cleanup_current_loaded)
        self.assertEqual(prepared['plan']['authority'],self.continued_authority.descriptor['id']);self.assertIsNone(prepared['verified'])
        self.assertEqual(self.cleanup_adapter.recovers,0);verified=self.clean_continued_step();path=self.continued_cleanup_path();saved=(path.stat().st_ino,path.read_bytes())
        self.assertEqual(self.clean_continued_step(),verified);self.assertEqual(self.cleanup_adapter.recovers,1);self.assertEqual(saved,(path.stat().st_ino,path.read_bytes()))
        self.assertEqual(before,self.continued_core_snapshot());snap=self.inspect();self.assertFalse(snap['activation_authorized']);self.assertFalse(snap['transport_cleanup_verified'])
        if ROOT==Path('/lab/source'):
            f.f.save(Path('/lab/continuation-cleanup-result.json'),{'passed':True,'prepared':prepared,'verified':verified,
                'inspection':snap,'original_runtime_object_replaced':True,'core_unchanged':True,
                'replay_unchanged':True,'application_observations':'synthetic','new_admission':False})

    def test_live_compositor_refused_before_adapter(self):
        self.refuse_continued_cleanup();self.assertEqual(self.cleanup_adapter.prepares,0)
    def test_guardian_closure_required_after_compositor_loss(self):
        self.stop_display();self.refuse_continued_cleanup();self.assertEqual(self.cleanup_adapter.prepares,0)
    def test_pending_switch_blocks_prepare(self):
        self.ready_continued_cleanup();self.record({'type':'begin','request':uuid.uuid4().hex,'candidate':j.digest(self.continued_spec),'expected':j.digest(self.cleanup_current_loaded)})
        self.refuse_continued_cleanup();self.assertEqual(self.cleanup_adapter.prepares,0)
    def test_unknown_format_refuses_unchanged(self):
        self.ready_continued_cleanup();self.clean_continued_step();p=self.continued_cleanup_path();v=json.loads(p.read_text());v['format']=99;f.f.save(p,v)
        self.refuse_continued_cleanup();self.assertEqual(self.cleanup_adapter.recovers,0)
    def test_original_reader_refuses_new_record_and_authority(self):
        self.ready_continued_cleanup();v=self.clean_continued_step()
        for reader in (original,previous):
            with self.assertRaises(ValueError):reader.record_shape(v)
        before=self.snapshot()
        with self.store.lease() as lease,self.assertRaises(ValueError):original.step(self.continued_authority.root,self.store,lease,self.cleanup_adapter,self.continued_cleanup_request)
        self.assertEqual(before,self.snapshot());self.assertEqual(self.cleanup_adapter.recovers,0)
    def test_second_intent_cannot_take_same_activation(self):
        self.ready_continued_cleanup();self.clean_continued_step();before=self.snapshot()
        with self.assertRaises(ValueError):self.clean_continued_step(uuid.uuid4().hex)
        self.assertEqual(before,self.snapshot());self.assertEqual(self.cleanup_adapter.prepares,1)
    def test_transport_failure_preserves_prepared_record(self):
        self.ready_continued_cleanup();self.clean_continued_step();self.cleanup_adapter.fail=True;before=self.snapshot()
        with self.assertRaises(RuntimeError):self.clean_continued_step()
        self.assertEqual(before,self.snapshot());self.cleanup_adapter.fail=False;self.assertIsNotNone(self.clean_continued_step()['verified'])
    def test_pending_application_change_cannot_be_verified(self):
        self.ready_continued_cleanup();self.clean_continued_step();self.cleanup_adapter.changed=True;self.refuse_continued_cleanup()
        self.assertIsNone(cleanup.read_record(self.store,self.continued_cleanup_request)['verified'])
    def test_missing_ancestor_checkpoint_blocks_even_completed_retry(self):
        self.ready_continued_cleanup();self.clean_continued_step();self.clean_continued_step();self.cleanup_ancestor_checkpoint.unlink();self.refuse_continued_cleanup();self.assertEqual(self.cleanup_adapter.recovers,1)
    def test_missing_grandparent_checkpoint_blocks_completed_retry(self):
        self.ready_continued_cleanup();self.clean_continued_step();self.clean_continued_step();Path(self.second_archived['ancestor_checkpoint']['object']['path']).unlink();self.refuse_continued_cleanup();self.assertEqual(self.cleanup_adapter.recovers,1)

    def test_modified_ancestor_cleanup_blocks_before_adapter(self):
        self.ready_continued_cleanup();p=Path(self.second_archived['cleanup']['object']['path']);v=json.loads(p.read_text());v['verified']['members']['dms']['pending_ids']=[];f.f.save(p,v)
        self.refuse_continued_cleanup();self.assertEqual(self.cleanup_adapter.prepares,0)
    def test_missing_current_binding_blocks_retry(self):
        self.ready_continued_cleanup();self.clean_continued_step();(self.root/'continued-binding.json').unlink();self.refuse_continued_cleanup();self.assertEqual(self.cleanup_adapter.recovers,0)
    def test_later_canonical_append_invalidates_prepared_plan(self):
        self.ready_continued_cleanup();self.clean_continued_step();changed=copy.deepcopy(self.continued_spec)
        for role in j.ROLES:changed['members'][role]['artifact_sha256']=j.digest(['later-cleanup',role])
        self.record({'type':'register','spec':changed});self.refuse_continued_cleanup();self.assertEqual(self.cleanup_adapter.recovers,0)
    def test_lost_prepare_reply_keeps_one_original_plan(self):
        self.ready_continued_cleanup();save=cleanup.save_record
        def lost(*args):save(*args);raise OSError('lost reply')
        with patch.object(cleanup,'save_record',lost),self.assertRaises(OSError):self.clean_continued_step()
        prepared=cleanup.read_record(self.store,self.continued_cleanup_request);self.assertEqual(self.clean_continued_step()['plan'],prepared['plan']);self.assertEqual(self.cleanup_adapter.prepares,1)
    def test_lost_verified_reply_does_not_repeat_transport_cleanup(self):
        self.ready_continued_cleanup();self.clean_continued_step();save=cleanup.save_record
        def lost(*args):save(*args);raise OSError('lost reply')
        with patch.object(cleanup,'save_record',lost),self.assertRaises(OSError):self.clean_continued_step()
        self.assertIsNotNone(self.clean_continued_step()['verified']);self.assertEqual(self.cleanup_adapter.recovers,1)
    def test_presync_failure_cannot_dispatch_cleanup(self):
        self.ready_continued_cleanup();self.clean_continued_step()
        with patch.object(cleanup,'sync_record',side_effect=OSError('fsync failed')),self.assertRaises(OSError):self.clean_continued_step()
        self.assertEqual(self.cleanup_adapter.recovers,0)
    def test_prepare_publication_failure_has_no_transport_effect(self):
        self.ready_continued_cleanup()
        with patch.object(cleanup.os,'replace',side_effect=OSError('publication failed')),self.assertRaises(OSError):self.clean_continued_step()
        self.assertIsNone(cleanup.read_record(self.store,self.continued_cleanup_request));self.assertEqual(self.cleanup_adapter.recovers,0)
        self.assertIsNone(self.clean_continued_step()['verified'])
    def test_new_cleanup_refuses_original_descriptor(self):
        with patch.object(inspection,'inspect',return_value={'format':1,'descriptor':{'format':1}}):
            with self.store.lease() as lease,self.assertRaisesRegex(ValueError,'requires authority3'):cleanup.context(self.continued_authority.root,self.store,lease)
    def test_foreign_or_expired_lease_cannot_prepare(self):
        self.ready_continued_cleanup()
        with self.store.lease() as lease:pass
        with self.assertRaises(ValueError):cleanup.step(self.continued_authority.root,self.store,lease,self.cleanup_adapter,self.continued_cleanup_request)
        other=j.Store(self.root/'unrelated-coordinator',create=True)
        try:
            with other.lease() as lease,self.assertRaises(ValueError):cleanup.step(self.continued_authority.root,self.store,lease,self.cleanup_adapter,self.continued_cleanup_request)
        finally:other.close()
        self.assertEqual(self.cleanup_adapter.prepares,0)

for name in dir(f.ContinuationInspectionTests):
    value=getattr(f.ContinuationInspectionTests,name)
    if callable(value) and not name.startswith(('test','_')) and not hasattr(ContinuationCleanupTests,name):setattr(ContinuationCleanupTests,name,value)
if __name__=='__main__':unittest.main()
