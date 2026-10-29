#!/usr/bin/env python3
"""Second desktop cleanup retains linked ownership, state and reply-loss evidence."""
import copy,importlib.util,json,os,sys,unittest,uuid
from pathlib import Path
from unittest.mock import patch
ROOT=Path(__file__).resolve().parents[1];sys.path.insert(0,str(ROOT/'native/collections'))
import successor_cleanup as cleanup
import session_cleanup as original
import successor_inspection as inspection
import journal as j
import runtime_authority as authority
spec=importlib.util.spec_from_file_location('linked_cleanup_fixture',Path(__file__).with_name('test-collection-successor-inspection.py'))
f=importlib.util.module_from_spec(spec);spec.loader.exec_module(f)
spec=importlib.util.spec_from_file_location('original_cleanup_fixture',Path(__file__).with_name('test-collection-session-cleanup.py'))
a=importlib.util.module_from_spec(spec);spec.loader.exec_module(a)

class LinkedCleanupTests(unittest.TestCase):
    def setUp(self):
        f.LinkedInspectionTests.setUp(self);self.replacement=self.bound_ready()
        self.ancestor_checkpoint=self.checkpoint_path();self.request=uuid.uuid4().hex;self.adapter=a.Adapter()
    def clean_step(self,request=None):
        with self.store.lease() as lease:return cleanup.step(self.next_authority.root,self.store,lease,self.adapter,request or self.request)
    def ready_cleanup(self):self.stop_display();self.close_guardian()
    def cleanup_path(self):return self.store.root/cleanup.filename(self.request)
    def core_snapshot(self):
        return {k:v for k,v in self.snapshot().items() if cleanup.PREFIX not in k and '.pending-desktop-successor-recovery-' not in k}
    def refuse_cleanup(self):
        before=self.snapshot()
        with self.assertRaises((ValueError,OSError)):self.clean_step()
        self.assertEqual(before,self.snapshot())

    def test_prepare_verify_replay_preserves_ancestors_and_current_history(self):
        self.ready_cleanup();before=self.core_snapshot();prepared=self.clean_step()
        self.assertEqual(prepared['format'],2);self.assertEqual(prepared['plan']['loaded'],self.replacement)
        self.assertEqual(prepared['plan']['authority'],self.next_authority.descriptor['id']);self.assertIsNone(prepared['verified'])
        self.assertEqual(self.adapter.recovers,0);verified=self.clean_step();path=self.cleanup_path();saved=(path.stat().st_ino,path.read_bytes())
        self.assertEqual(self.clean_step(),verified);self.assertEqual(self.adapter.recovers,1);self.assertEqual(saved,(path.stat().st_ino,path.read_bytes()))
        self.assertEqual(before,self.core_snapshot());snap=self.inspect();self.assertFalse(snap['activation_authorized']);self.assertFalse(snap['transport_cleanup_verified'])
        if ROOT==Path('/lab/source'):
            f.f.save(Path('/lab/successor-cleanup-result.json'),{'passed':True,'prepared':prepared,'verified':verified,
                'inspection':snap,'original_runtime_object_replaced':True,'core_unchanged':True,
                'replay_unchanged':True,'application_observations':'synthetic','new_admission':False})

    def test_live_compositor_refused_before_adapter(self):
        self.refuse_cleanup();self.assertEqual(self.adapter.prepares,0)
    def test_guardian_closure_required_after_compositor_loss(self):
        self.stop_display();self.refuse_cleanup();self.assertEqual(self.adapter.prepares,0)
    def test_pending_switch_blocks_prepare(self):
        self.ready_cleanup();self.record({'type':'begin','request':uuid.uuid4().hex,'candidate':j.digest(self.next_spec),'expected':j.digest(self.replacement)})
        self.refuse_cleanup();self.assertEqual(self.adapter.prepares,0)
    def test_unknown_format_refuses_unchanged(self):
        self.ready_cleanup();self.clean_step();p=self.cleanup_path();v=json.loads(p.read_text());v['format']=99;f.f.save(p,v)
        self.refuse_cleanup();self.assertEqual(self.adapter.recovers,0)
    def test_original_reader_refuses_new_record_and_authority(self):
        self.ready_cleanup();v=self.clean_step()
        with self.assertRaises(ValueError):original.record_shape(v)
        before=self.snapshot()
        with self.store.lease() as lease,self.assertRaises(ValueError):original.step(self.next_authority.root,self.store,lease,self.adapter,self.request)
        self.assertEqual(before,self.snapshot());self.assertEqual(self.adapter.recovers,0)
    def test_second_intent_cannot_take_same_activation(self):
        self.ready_cleanup();self.clean_step();before=self.snapshot()
        with self.assertRaises(ValueError):self.clean_step(uuid.uuid4().hex)
        self.assertEqual(before,self.snapshot());self.assertEqual(self.adapter.prepares,1)
    def test_transport_failure_preserves_prepared_record(self):
        self.ready_cleanup();self.clean_step();self.adapter.fail=True;before=self.snapshot()
        with self.assertRaises(RuntimeError):self.clean_step()
        self.assertEqual(before,self.snapshot());self.adapter.fail=False;self.assertIsNotNone(self.clean_step()['verified'])
    def test_pending_application_change_cannot_be_verified(self):
        self.ready_cleanup();self.clean_step();self.adapter.changed=True;self.refuse_cleanup()
        self.assertIsNone(cleanup.read_record(self.store,self.request)['verified'])
    def test_missing_ancestor_checkpoint_blocks_even_completed_retry(self):
        self.ready_cleanup();self.clean_step();self.clean_step();self.ancestor_checkpoint.unlink();self.refuse_cleanup();self.assertEqual(self.adapter.recovers,1)
    def test_modified_ancestor_cleanup_blocks_before_adapter(self):
        self.ready_cleanup();p=Path(self.archived['cleanup']['object']['path']);v=json.loads(p.read_text());v['verified']['members']['dms']['pending_ids']=[];f.f.save(p,v)
        self.refuse_cleanup();self.assertEqual(self.adapter.prepares,0)
    def test_missing_current_binding_blocks_retry(self):
        self.ready_cleanup();self.clean_step();(self.root/'new-binding.json').unlink();self.refuse_cleanup();self.assertEqual(self.adapter.recovers,0)
    def test_later_canonical_append_invalidates_prepared_plan(self):
        self.ready_cleanup();self.clean_step();changed=copy.deepcopy(self.next_spec)
        for role in j.ROLES:changed['members'][role]['artifact_sha256']=j.digest(['later-cleanup',role])
        self.record({'type':'register','spec':changed});self.refuse_cleanup();self.assertEqual(self.adapter.recovers,0)
    def test_lost_prepare_reply_keeps_one_original_plan(self):
        self.ready_cleanup();save=cleanup.save_record
        def lost(*args):save(*args);raise OSError('lost reply')
        with patch.object(cleanup,'save_record',lost),self.assertRaises(OSError):self.clean_step()
        prepared=cleanup.read_record(self.store,self.request);self.assertEqual(self.clean_step()['plan'],prepared['plan']);self.assertEqual(self.adapter.prepares,1)
    def test_lost_verified_reply_does_not_repeat_transport_cleanup(self):
        self.ready_cleanup();self.clean_step();save=cleanup.save_record
        def lost(*args):save(*args);raise OSError('lost reply')
        with patch.object(cleanup,'save_record',lost),self.assertRaises(OSError):self.clean_step()
        self.assertIsNotNone(self.clean_step()['verified']);self.assertEqual(self.adapter.recovers,1)
    def test_presync_failure_cannot_dispatch_cleanup(self):
        self.ready_cleanup();self.clean_step()
        with patch.object(cleanup,'sync_record',side_effect=OSError('fsync failed')),self.assertRaises(OSError):self.clean_step()
        self.assertEqual(self.adapter.recovers,0)
    def test_prepare_publication_failure_has_no_transport_effect(self):
        self.ready_cleanup()
        with patch.object(cleanup.os,'replace',side_effect=OSError('publication failed')),self.assertRaises(OSError):self.clean_step()
        self.assertIsNone(cleanup.read_record(self.store,self.request));self.assertEqual(self.adapter.recovers,0)
        self.assertIsNone(self.clean_step()['verified'])
    def test_new_cleanup_refuses_original_descriptor(self):
        with patch.object(inspection,'inspect',return_value={'format':1,'descriptor':{'format':1}}):
            with self.store.lease() as lease,self.assertRaisesRegex(ValueError,'requires authority2'):cleanup.context(self.next_authority.root,self.store,lease)
    def test_foreign_or_expired_lease_cannot_prepare(self):
        self.ready_cleanup()
        with self.store.lease() as lease:pass
        with self.assertRaises(ValueError):cleanup.step(self.next_authority.root,self.store,lease,self.adapter,self.request)
        other=j.Store(self.root/'unrelated-coordinator',create=True)
        try:
            with other.lease() as lease,self.assertRaises(ValueError):cleanup.step(self.next_authority.root,self.store,lease,self.adapter,self.request)
        finally:other.close()
        self.assertEqual(self.adapter.prepares,0)

for name in dir(f.LinkedInspectionTests):
    value=getattr(f.LinkedInspectionTests,name)
    if callable(value) and not name.startswith(('test','_')) and not hasattr(LinkedCleanupTests,name):setattr(LinkedCleanupTests,name,value)
if __name__=='__main__':unittest.main()
