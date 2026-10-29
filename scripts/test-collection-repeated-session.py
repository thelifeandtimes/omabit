#!/usr/bin/env python3
"""Reusable ownership fences old writers before any new runtime admission."""
import copy,importlib.util,json,os,sys,unittest,uuid
from pathlib import Path
from unittest.mock import patch
ROOT=Path(__file__).resolve().parents[1];sys.path.insert(0,str(ROOT/'native/collections'))
import repeated_session as repeated
import repeated_checkpoint as checkpoint
import runtime_authority as authority
import journal as j

def fixture(name):
    spec=importlib.util.spec_from_file_location(name.replace('-','_'),Path(__file__).with_name(name+'.py'))
    module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module);return module

current=fixture('test-collection-repeated-checkpoint')
first=fixture('test-collection-session-checkpoint')
second=fixture('test-collection-successor-checkpoint')

class Observer:
    def __init__(self,members):self.members=copy.deepcopy(members);self.calls=0
    def inspect_successor(self,plan):self.calls+=1;return copy.deepcopy(self.members)

class Flow:
    def initialize_repeat(self,record,path):
        self.repeat_record=record;self.repeat_reference=repeated.reference(path);self.repeat_attempt=uuid.uuid4().hex
        receipt=checkpoint.parsed(record,record['cleanup']['object']['path'])
        self.repeat_observer=Observer(receipt['verified']['members']);self.compositor=self.start_display()
    def repeat_step(self,attempt=None):
        with self.store.lease() as lease:
            return repeated.step(self.display,self.store,lease,self.repeat_observer,self.repeat_reference,attempt or self.repeat_attempt)
    def finish_repeat(self):
        result=None
        for _ in range(4):
            result=self.repeat_step()
            if result['phase']=='claimed':return result
        self.fail('ownership claim did not complete')
    def repeat_path(self):return self.store.root/repeated.filename(self.repeat_attempt)
    def assert_repeat_refused(self,regex='.'):
        before=self.snapshot()
        with self.assertRaisesRegex((ValueError,OSError),regex):self.repeat_step()
        self.assertEqual(before,self.snapshot())
    def old_step(self,module,attempt):
        with self.store.lease() as lease:
            return module.step(self.display,self.store,lease,self.repeat_observer,self.repeat_record['id'],attempt)

class RepeatedIntentTests(Flow,unittest.TestCase):
    def setUp(self):
        current.RepeatedCheckpointTests.setUp(self);self.clean_repeated_checkpoint()
        record=self.capture_repeated_checkpoint();path=self.repeated_checkpoint_path();self.lose_repeated_runtime()
        self.initialize_repeat(record,path)
    def test_four_phases_claim_once_without_authority(self):
        phases=[self.repeat_step()['phase'] for _ in range(4)]
        self.assertEqual(phases,['prepared','successor-fenced','continuation-fenced','claimed'])
        result=self.repeat_step();value=repeated.check_claimed(result['intent']);before=self.snapshot()
        self.assertEqual(self.repeat_step(),result);self.assertEqual(before,self.snapshot())
        self.assertFalse(result['activation_authorized']);self.assertFalse((self.runtime/authority.ANCHOR).exists())
        self.assertEqual(value['checkpoint'],self.repeat_reference)
        self.assertEqual(value['format'],3)
        if ROOT==Path('/lab/source'):
            current.save(Path('/lab/repeated-intent-result.json'),{'passed':True,'phases':phases,'intent':value,
                'checkpoint':self.repeat_record,'exact_retry_unchanged':True,'new_admission':False,'new_application_instances':0})
    def test_both_unchanged_legacy_readers_refuse_their_fence(self):
        result=self.finish_repeat()
        for module in (repeated.legacy1,repeated.legacy2):
            before=self.snapshot()
            with self.assertRaises(ValueError):module.read(self.store,self.repeat_attempt)
            self.assertEqual(before,self.snapshot())
        self.assertEqual(repeated.check_claimed(result['intent']),result['intent'])
    def test_second_new_attempt_cannot_claim_same_predecessor(self):
        self.repeat_step();before=self.snapshot()
        with self.assertRaisesRegex(ValueError,'another successor intent owns'):
            self.repeat_step(uuid.uuid4().hex)
        self.assertEqual(before,self.snapshot())
    def test_four_lost_publication_replies_reuse_one_intent(self):
        original=repeated.provisioning.rename_no_replace
        def lost(*args):original(*args);raise OSError('publication reply lost')
        saved=None
        for _ in range(4):
            with patch.object(repeated.provisioning,'rename_no_replace',lost),self.assertRaises(OSError):self.repeat_step()
            now=(self.repeat_path().stat().st_ino,self.repeat_path().read_bytes())
            if saved is None:saved=now
            self.assertEqual(saved,now)
        self.assertEqual(self.repeat_step()['phase'],'claimed')
    def test_failed_fence_publication_preserves_bound_stage(self):
        self.repeat_step();before=self.snapshot()
        with patch.object(repeated.provisioning,'rename_no_replace',side_effect=OSError('publish failed')),self.assertRaises(OSError):self.repeat_step()
        self.assertEqual(before,self.snapshot());self.assertEqual(self.finish_repeat()['phase'],'claimed')
    def test_missing_bound_fence_stage_never_reconstructed(self):
        value=self.repeat_step()['intent'];Path(value['fences']['successor']['staging']['path']).unlink()
        self.assert_repeat_refused();self.assertFalse((self.runtime/authority.PROVISION_CLAIM).exists())
    def test_missing_published_fence_never_reconstructed(self):
        value=self.finish_repeat()['intent'];Path(value['fences']['successor']['target']['path']).unlink()
        self.assert_repeat_refused()
        with self.assertRaises(OSError):repeated.check_claimed(value)
    def test_same_fence_bytes_at_a_new_inode_refuse(self):
        value=self.finish_repeat()['intent'];p=Path(value['fences']['continuation']['target']['path']);raw=p.read_bytes()
        p.rename(p.with_suffix('.retained'));p.write_bytes(raw);p.chmod(0o600);self.assert_repeat_refused()
    def test_missing_primary_record_with_a_published_fence_refuses(self):
        self.repeat_step();self.repeat_step();self.repeat_path().unlink()
        self.assert_repeat_refused('bound repeated intent is missing')
    def test_changed_cohort_refuses_before_staging(self):
        self.repeat_observer.members['dms']['pending_ids']=[];before=self.snapshot()
        with self.assertRaises(ValueError):self.repeat_step()
        self.assertEqual(before,self.snapshot());self.assertFalse(self.repeat_path().exists())
    def test_changed_journal_refuses_before_claim(self):
        self.repeat_step();spec=copy.deepcopy(self.repeat_record['inspection']['reservation']['reservation']['permit']['specification'])
        for role in j.ROLES:spec['members'][role]['artifact_sha256']=j.digest(['later-repeat',role])
        self.record({'type':'register','spec':spec});self.assert_repeat_refused('unchanged checkpoint journal')
    def test_missing_ancestor_blocks_pending_ownership(self):
        self.repeat_step();Path(self.repeat_record['ancestor_checkpoint']['object']['path']).unlink();self.assert_repeat_refused()
    def test_foreign_fence_destination_is_not_overwritten(self):
        value=self.repeat_step()['intent'];p=Path(value['fences']['successor']['target']['path']);current.save(p,{'foreign':True})
        self.assert_repeat_refused();self.assertEqual(json.loads(p.read_text()),{'foreign':True})
    def test_expired_lease_refuses_without_observation(self):
        with self.store.lease() as lease:pass
        before=self.snapshot()
        with self.assertRaises(ValueError):repeated.step(self.display,self.store,lease,self.repeat_observer,self.repeat_reference,self.repeat_attempt)
        self.assertEqual(before,self.snapshot());self.assertEqual(self.repeat_observer.calls,0)
    def test_unknown_intent_is_preserved(self):
        self.repeat_step();value=json.loads(self.repeat_path().read_text());value['format']=99;current.save(self.repeat_path(),value)
        self.assert_repeat_refused()
    def test_runtime_loss_does_not_rebind_pending_claim(self):
        self.repeat_step();self.stop_display();self.display.unlink();self.compositor=self.start_display();self.assert_repeat_refused('display changed')

class FirstCheckpointImportTests(Flow,unittest.TestCase):
    def setUp(self):
        first.CheckpointTests.setUp(self);self.completed_cleanup();record=self.capture();path=self.checkpoint_path();self.lose_runtime()
        self.initialize_repeat(record,path)
    def test_legacy_checkpoint1_import_and_original_writer_refusal(self):
        result=self.finish_repeat();self.assertEqual(result['intent']['checkpoint'],self.repeat_reference)
        with self.assertRaises(ValueError):self.old_step(repeated.legacy1,uuid.uuid4().hex)
    def test_prior_legacy_intent_prevents_new_preparation(self):
        old=self.old_step(repeated.legacy1,uuid.uuid4().hex);self.assertEqual(old['phase'],'prepared')
        self.assert_repeat_refused('another intent owns');self.assertFalse(self.repeat_path().exists())

class SecondCheckpointImportTests(Flow,unittest.TestCase):
    def setUp(self):
        second.LinkedCheckpointTests.setUp(self);self.clean_linked();record=self.capture_linked();path=self.linked_checkpoint_path();self.lose_linked_runtime()
        self.initialize_repeat(record,path)
    def test_legacy_checkpoint2_import_and_original_writer_refusal(self):
        result=self.finish_repeat();self.assertEqual(result['intent']['checkpoint'],self.repeat_reference)
        with self.assertRaises(ValueError):self.old_step(repeated.legacy2,uuid.uuid4().hex)
    def test_legacy_claim_between_fences_blocks_new_runtime_claim(self):
        self.repeat_step();self.repeat_step()  # The format2 writer is not fenced yet.
        old=self.old_step(repeated.legacy2,uuid.uuid4().hex);self.assertEqual(old['phase'],'prepared')
        self.assert_repeat_refused('another intent owns');self.assertFalse((self.runtime/authority.PROVISION_CLAIM).exists())

for target,source in ((RepeatedIntentTests,current.RepeatedCheckpointTests),(FirstCheckpointImportTests,first.CheckpointTests),
                       (SecondCheckpointImportTests,second.LinkedCheckpointTests)):
    for name in dir(source):
        value=getattr(source,name)
        if callable(value) and not name.startswith(('test','_')) and not hasattr(target,name):setattr(target,name,value)

del target,source  # Do not expose imported TestCase aliases to unittest's module loader.

if __name__=='__main__':unittest.main()
