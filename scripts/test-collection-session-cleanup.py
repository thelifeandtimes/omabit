#!/usr/bin/env python3
"""Prepared desktop cleanup records preserve all original history and intent."""
import copy,importlib.util,json,os
from pathlib import Path
import sys,unittest,uuid
from unittest.mock import patch
ROOT=Path(__file__).resolve().parents[1];sys.path.insert(0,str(ROOT/'native/collections'))
import authority_inspection as inspection
import session_cleanup as cleanup
import journal as j
import runtime_authority as authority
spec=importlib.util.spec_from_file_location('desktop_inspection_fixture',Path(__file__).with_name('test-collection-authority-inspection.py'))
f=importlib.util.module_from_spec(spec);spec.loader.exec_module(f)

class Adapter:
    def __init__(self):self.prepares=0;self.recovers=0;self.fail=False;self.changed=False;self.expected=None
    def prepare_desktop(self,expected):
        self.prepares+=1;self.expected=expected
        return {role:{'binding':expected['loaded']['members'][role],'application_sha256':j.digest(['application',role]),
           'pending_ids':[uuid.uuid4().hex] if role=='dms' else [],'credential_refs':[j.digest(['session',role])] if role!='omabit' else [],
           'transport_sha256':j.digest(['open',role]),'channel_ids':[uuid.uuid4().hex] if role!='omabit' else []} for role in j.ROLES}
    def recover_desktop(self,plan):
        self.recovers+=1
        if self.fail:raise RuntimeError('remote cleanup refused')
        members=copy.deepcopy(plan['members'])
        for role,row in members.items():row.update(closed=True,transport_sha256=j.digest(['closed',role]))
        if self.changed:members['dms']['pending_ids']=[]
        return {'plan_sha256':j.digest(plan),'guardian':plan['guardian'],'closure_request':plan['closure_request'],
          'closure_receipt_sha256':plan['closure_receipt_sha256'],'proof':'waitpid-echild','members':members}

class DesktopCleanupTests(unittest.TestCase):
    def setUp(self):
        f.InspectionTests.setUp(self);self.request=uuid.uuid4().hex;self.adapter=Adapter()
    def ready(self):self.stop_display();self.close_guardian()
    def step(self,request=None):
        with self.store.lease() as lease:return cleanup.step(self.authority.root,self.store,lease,self.adapter,request or self.request)
    def core_snapshot(self):
        paths=[self.store.root/'journal.json',self.store.root/authority.MIRROR]
        paths += [x for x in self.authority.root.rglob('*') if x.is_file()]
        return {str(x):(x.stat().st_ino,x.read_bytes()) for x in paths}

    def test_prepare_precedes_cleanup_and_verified_retry_has_no_effects(self):
        self.ready();before=self.core_snapshot();prepared=self.step();self.assertIsNone(prepared['verified']);self.assertEqual(self.adapter.recovers,0)
        verified=self.step();self.assertIsNotNone(verified['verified']);path=self.store.root/cleanup.filename(self.request);raw=path.read_bytes();inode=path.stat().st_ino
        self.assertEqual(self.step(),verified);self.assertEqual(self.adapter.recovers,1);self.assertEqual((path.read_bytes(),path.stat().st_ino),(raw,inode));self.assertEqual(before,self.core_snapshot())

    def test_live_compositor_refused_before_adapter(self):
        before=self.snapshot()
        with self.assertRaises(ValueError):self.step()
        self.assertEqual(self.adapter.prepares,0);self.assertEqual(before,self.snapshot())

    def test_missing_original_closure_refused_before_adapter(self):
        self.stop_display()
        with self.assertRaises(ValueError):self.step()
        self.assertEqual(self.adapter.prepares,0)

    def test_unknown_receipt_format_is_preserved(self):
        self.ready();self.step();path=self.store.root/cleanup.filename(self.request);value=json.loads(path.read_text());value['format']=2;f.h.save(path,value);before=self.snapshot()
        with self.assertRaises(ValueError):self.step()
        self.assertEqual(before,self.snapshot());self.assertEqual(self.adapter.recovers,0)

    def test_other_recovery_cannot_take_same_activation(self):
        self.ready();self.step();before=self.snapshot()
        with self.assertRaises(ValueError):self.step(uuid.uuid4().hex)
        self.assertEqual(before,self.snapshot());self.assertEqual(self.adapter.prepares,1)

    def test_failed_cleanup_preserves_prepared_intent(self):
        self.ready();self.step();before=self.snapshot();self.adapter.fail=True
        with self.assertRaises(RuntimeError):self.step()
        self.assertEqual(before,self.snapshot());self.assertIsNone(cleanup.read_record(self.store,self.request)['verified'])
        self.adapter.fail=False;self.assertIsNotNone(self.step()['verified'])

    def test_application_intent_change_cannot_be_verified(self):
        self.ready();self.step();before=self.snapshot();self.adapter.changed=True
        with self.assertRaises(ValueError):self.step()
        self.assertEqual(before,self.snapshot());self.assertIsNone(cleanup.read_record(self.store,self.request)['verified'])

    def test_canonical_history_change_blocks_cleanup(self):
        self.ready();self.step()
        path=self.store.root/'journal.json';value=json.loads(path.read_text());value['format']=1;f.h.save(path,value);before=self.snapshot()
        with self.assertRaises(ValueError):self.step()
        self.assertEqual(before,self.snapshot());self.assertEqual(self.adapter.recovers,0)

    def test_pending_switch_blocks_cleanup(self):
        self.ready();self.record({'type':'begin','request':uuid.uuid4().hex,'candidate':j.digest(self.spec),'expected':j.digest(self.loaded)})
        with self.assertRaises(ValueError):self.step()
        self.assertEqual(self.adapter.prepares,0)

    def test_missing_bound_original_evidence_blocks_completed_replay(self):
        self.ready();self.step();self.step();self.binding_path.unlink();before=self.snapshot()
        with self.assertRaises((OSError,ValueError)):self.step()
        self.assertEqual(before,self.snapshot());self.assertEqual(self.adapter.recovers,1)

    def test_prepare_publication_failure_never_dispatches_cleanup(self):
        self.ready()
        with patch.object(cleanup.os,'replace',side_effect=OSError('publication failed')),self.assertRaises(OSError):self.step()
        self.assertIsNone(cleanup.read_record(self.store,self.request));self.assertEqual(self.adapter.recovers,0)
        self.assertIsNone(self.step()['verified']);self.assertEqual(self.adapter.recovers,0)

    def test_lost_prepare_reply_keeps_original_plan(self):
        self.ready();original=cleanup.save_record
        def lost(store,value):original(store,value);raise OSError('lost reply')
        with patch.object(cleanup,'save_record',lost),self.assertRaises(OSError):self.step()
        prepared=cleanup.read_record(self.store,self.request);self.assertIsNone(prepared['verified']);self.assertEqual(self.step()['plan'],prepared['plan']);self.assertEqual(self.adapter.prepares,1)

    def test_lost_verified_reply_never_repeats_cleanup(self):
        self.ready();self.step();original=cleanup.save_record
        def lost(store,value):original(store,value);raise OSError('lost reply')
        with patch.object(cleanup,'save_record',lost),self.assertRaises(OSError):self.step()
        self.assertIsNotNone(self.step()['verified']);self.assertEqual(self.adapter.recovers,1)

    def test_failed_presync_never_calls_cleanup(self):
        self.ready();self.step()
        with patch.object(cleanup,'sync_record',side_effect=OSError('fsync failed')),self.assertRaises(OSError):self.step()
        self.assertEqual(self.adapter.recovers,0)

    def test_completed_replay_rechecks_original_context_before_adapter(self):
        self.ready();self.step();self.step();path=self.store.root/authority.MIRROR;value=json.loads(path.read_text());value['id']=uuid.uuid4().hex;f.h.save(path,value)
        with self.assertRaises(ValueError):self.step()
        self.assertEqual(self.adapter.recovers,1)

    def test_explicit_canonical_lease_allows_read_only_inspection(self):
        before=self.snapshot()
        with self.store.lease() as lease:value=inspection.inspect(self.authority.root,self.store.root,lease=lease)
        self.assertFalse(value['activation_authorized']);self.assertEqual(before,self.snapshot())

    def test_expired_and_foreign_inspection_leases_refused(self):
        with self.store.lease() as lease:pass
        with self.assertRaises(ValueError):inspection.inspect(self.authority.root,self.store.root,lease=lease)
        other=j.Store(self.root/'other',create=True)
        try:
            with other.lease() as lease:
                with self.assertRaises(ValueError):inspection.inspect(self.authority.root,self.store.root,lease=lease)
        finally:other.close()

for name in ('cleanup','provision','receive','launch_guardian','send','start_claims','record','write_binding','bind','start_display','stop_display','close_guardian','snapshot'):
    setattr(DesktopCleanupTests,name,getattr(f.InspectionTests,name))

if __name__=='__main__':unittest.main()
