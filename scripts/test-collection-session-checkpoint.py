#!/usr/bin/env python3
"""Durable checkpoint publication preserves metadata after runtime loss."""
import copy,importlib.util,json,os
from pathlib import Path
import sys,unittest,uuid
from unittest.mock import patch
ROOT=Path(__file__).resolve().parents[1];sys.path.insert(0,str(ROOT/'native/collections'))
import session_checkpoint as checkpoint
import session_cleanup
import authority_inspection
import runtime_authority as authority
import journal as j
spec=importlib.util.spec_from_file_location('checkpoint_cleanup_fixture',Path(__file__).with_name('test-collection-session-cleanup.py'))
f=importlib.util.module_from_spec(spec);spec.loader.exec_module(f)

class CheckpointTests(unittest.TestCase):
    def setUp(self):f.DesktopCleanupTests.setUp(self)
    def completed_cleanup(self):self.ready();self.step();self.step()
    def capture(self,root=None):
        with self.store.lease() as lease:return checkpoint.capture(root or self.authority.root,self.store,lease,self.request)
    def read_checkpoint(self):
        with self.store.lease() as lease:return checkpoint.read(self.store,self.request,lease)
    def lose_runtime(self):
        self.authority.close();self.authority.fd=None
        self.runtime.rename(self.root/'retained-runtime');self.runtime.mkdir(mode=0o700)
    def checkpoint_path(self):return self.store.root/checkpoint.filename(self.request)

    def test_capture_and_replay_preserve_original_and_checkpoint_bytes(self):
        self.completed_cleanup();before=self.core_snapshot();record=self.capture();path=self.checkpoint_path();original=(path.stat().st_ino,path.read_bytes())
        self.assertEqual(self.capture(),record);self.assertEqual(self.read_checkpoint(),record);self.assertEqual((path.stat().st_ino,path.read_bytes()),original);self.assertEqual(before,self.core_snapshot())
        self.assertFalse(record['inspection']['activation_authorized'])

    def test_runtime_loss_keeps_checkpoint_readable_but_no_live_authority(self):
        self.completed_cleanup();record=self.capture();self.lose_runtime();before=self.snapshot()
        self.assertEqual(self.read_checkpoint(),record);self.assertEqual(self.capture(),record);self.assertEqual(before,self.snapshot())
        with self.assertRaises(OSError):authority.Authority(self.runtime/authority.ANCHOR)
        with self.assertRaises(OSError):authority_inspection.inspect(self.runtime/authority.ANCHOR,self.store.root)

    def test_unverified_cleanup_cannot_be_checkpointed(self):
        self.ready();self.step();before=self.snapshot()
        with self.assertRaises(ValueError):self.capture()
        self.assertEqual(before,self.snapshot());self.assertFalse(self.checkpoint_path().exists())

    def test_missing_cleanup_is_not_created(self):
        self.ready();before=self.snapshot()
        with self.assertRaises(ValueError):self.capture()
        self.assertEqual(before,self.snapshot())

    def test_runtime_loss_before_capture_cannot_invent_checkpoint(self):
        self.completed_cleanup();self.lose_runtime();before=self.snapshot()
        with self.assertRaises(OSError):self.capture()
        self.assertEqual(before,self.snapshot())

    def test_atomic_publication_failure_retains_stage_without_checkpoint(self):
        self.completed_cleanup();before=self.core_snapshot()
        with patch.object(checkpoint.provisioning,'rename_no_replace',side_effect=OSError('publication unavailable')),self.assertRaises(OSError):self.capture()
        self.assertFalse(self.checkpoint_path().exists());stages=list(self.store.root.glob('.pending-desktop-checkpoint-*'));self.assertEqual(len(stages),1)
        retained=stages[0].read_bytes();self.capture();self.assertEqual(stages[0].read_bytes(),retained);self.assertEqual(before,self.core_snapshot())

    def test_lost_publication_reply_reuses_published_checkpoint(self):
        self.completed_cleanup();original=checkpoint.provisioning.rename_no_replace
        def lost(*args):original(*args);raise OSError('reply lost')
        with patch.object(checkpoint.provisioning,'rename_no_replace',lost),self.assertRaises(OSError):self.capture()
        path=self.checkpoint_path();before=(path.stat().st_ino,path.read_bytes());self.lose_runtime();self.capture();self.assertEqual((path.stat().st_ino,path.read_bytes()),before)

    def test_conflicting_target_is_never_overwritten(self):
        self.completed_cleanup();original=checkpoint.provisioning.rename_no_replace
        def conflict(fd,source,destination):
            authority.write_new(fd,destination,{'foreign':True});original(fd,source,destination)
        with patch.object(checkpoint.provisioning,'rename_no_replace',conflict),self.assertRaises(ValueError):self.capture()
        self.assertEqual(json.loads(self.checkpoint_path().read_text()),{'foreign':True})

    def test_changed_checkpoint_payload_refused(self):
        self.completed_cleanup();self.capture();path=self.checkpoint_path();value=json.loads(path.read_text());value['inspection']['authority']=uuid.uuid4().hex;f.f.h.save(path,value);before=self.snapshot()
        with self.assertRaises(ValueError):self.read_checkpoint()
        self.assertEqual(before,self.snapshot())

    def test_changed_original_bytes_refused_even_with_new_outer_digest(self):
        self.completed_cleanup();value=self.capture();name=next(iter(value['files']));value['files'][name]['text']='{}';value['sha256']=j.digest({k:v for k,v in value.items() if k!='sha256'})
        with self.assertRaises(ValueError):checkpoint.validate(value)

    def test_unknown_checkpoint_format_refused_unchanged(self):
        self.completed_cleanup();self.capture();path=self.checkpoint_path();value=json.loads(path.read_text());value['format']=2;f.f.h.save(path,value);before=self.snapshot()
        with self.assertRaises(ValueError):self.read_checkpoint()
        self.assertEqual(before,self.snapshot())

    def test_missing_original_cleanup_refused_after_runtime_loss(self):
        self.completed_cleanup();self.capture();self.lose_runtime();(self.store.root/session_cleanup.filename(self.request)).unlink();before=self.snapshot()
        with self.assertRaises(OSError):self.read_checkpoint()
        self.assertEqual(before,self.snapshot())

    def test_changed_canonical_prefix_refused(self):
        self.completed_cleanup();self.capture();path=self.store.root/'journal.json';value=json.loads(path.read_text());value['events']=value['events'][:1];f.f.h.save(path,value)
        with self.assertRaises(ValueError):self.read_checkpoint()

    def test_append_only_canonical_history_preserves_old_checkpoint(self):
        self.completed_cleanup();record=self.capture();self.record({'type':'begin','request':uuid.uuid4().hex,'candidate':j.digest(self.spec),'expected':j.digest(self.loaded)})
        self.assertEqual(self.read_checkpoint(),record)

    def test_wrong_original_root_refuses_completed_retry(self):
        self.completed_cleanup();self.capture();before=self.snapshot()
        with self.assertRaises(ValueError):self.capture(self.root/'wrong'/authority.ANCHOR)
        self.assertEqual(before,self.snapshot())

    def test_source_inventory_bound_refuses_without_publication(self):
        self.completed_cleanup()
        with patch.object(checkpoint,'MAX_SOURCE_BYTES',1),self.assertRaises(ValueError):self.capture()
        self.assertFalse(self.checkpoint_path().exists())

    def test_live_authority_validation_and_archive_validator_agree(self):
        self.completed_cleanup();record=self.capture();value=record['inspection']['reservation'];descriptor=record['inspection']['descriptor']
        self.assertEqual(authority.reservation_value(value,descriptor),value)
        changed=copy.deepcopy(value);changed['format']=99
        with self.assertRaises(ValueError):authority.reservation_value(changed,descriptor)

for name in ('cleanup','provision','receive','launch_guardian','send','start_claims','record','write_binding','bind','start_display','stop_display','close_guardian','snapshot','ready','step','core_snapshot'):
    setattr(CheckpointTests,name,getattr(f.DesktopCleanupTests,name))
if __name__=='__main__':unittest.main()
