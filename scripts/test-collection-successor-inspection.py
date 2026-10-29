#!/usr/bin/env python3
"""Linked evidence inspection remains read-only after actual owned process loss."""
import copy,importlib.util,json,os,subprocess,sys,unittest,uuid
from pathlib import Path
from unittest.mock import patch
ROOT=Path(__file__).resolve().parents[1];sys.path.insert(0,str(ROOT/'native/collections'))
import successor_inspection as inspection
import authority_inspection as original
import runtime_authority as authority
import successor_authority as linked
import journal as j
spec=importlib.util.spec_from_file_location('linked_inspection_fixture',Path(__file__).with_name('test-collection-successor-authority.py'))
f=importlib.util.module_from_spec(spec);spec.loader.exec_module(f)


class LinkedInspectionTests(unittest.TestCase):
    def setUp(self):f.LinkedAuthorityTests.setUp(self);self.finish()
    def inspect(self,lease=None):return inspection.inspect(self.next_authority.root,self.store.root,lease=lease)
    def refuse(self):
        before=self.snapshot()
        with self.assertRaises((OSError,ValueError)):self.inspect()
        self.assertEqual(before,self.snapshot())
    def bound_ready(self):self.ready_transition();return self.adopt_new()

    def test_empty_linked_authority_reports_lineage_without_admission(self):
        before=self.snapshot();result=self.inspect();self.assertEqual(before,self.snapshot())
        self.assertEqual(result['format'],2);self.assertTrue(result['context']['live_context'])
        self.assertFalse(result['activation_authorized']);self.assertFalse(result['transport_cleanup_verified'])
        self.assertEqual(result['activations'],[]);self.assertEqual(result['journal']['committed'],self.loaded)
        self.assertEqual(result['lineage']['predecessor'],self.archived['inspection']['authority'])
        for path in (self.intent_path(),self.checkpoint_path(),self.record_path(),self.claim_path()):self.assertIn(str(path),result['files'])

    def test_live_dead_and_closed_bound_snapshots_export(self):
        loaded=self.bound_ready();before=self.snapshot();live=self.inspect();self.assertEqual(before,self.snapshot())
        self.assertEqual(live['journal']['committed'],loaded);self.assertEqual(live['activations'][-1]['loaded'],loaded)
        self.assertFalse(live['activations'][-1]['closure']['verified_process_closure'])
        self.stop_display();dead=self.inspect();self.assertEqual(dead['context']['server'],'gone')
        self.assertFalse(dead['activations'][-1]['closure']['verified_process_closure'])
        self.close_guardian();before=self.snapshot();closed=self.inspect();self.assertEqual(before,self.snapshot())
        self.assertTrue(closed['activations'][-1]['closure']['verified_process_closure'])
        self.assertFalse(closed['transport_cleanup_verified']);self.assertFalse(closed['activation_authorized'])
        with self.assertRaises(ValueError):linked.Authority(self.next_authority.root)
        if ROOT==Path('/lab/source'):
            f.save(Path('/lab/successor-inspection-result.json'),{'passed':True,'snapshots':{'live':live,'dead':dead,'closed':closed},
                'descriptor':self.next_authority.descriptor,'publication':linked.read(self.store,self.attempt),
                'original_runtime_object_replaced':Path(self.archived['inspection']['descriptor']['root']['path']).stat().st_ino!=self.archived['inspection']['descriptor']['root']['inode'],
                'application_observations':'synthetic','readonly':True})

    def test_reused_display_name_does_not_change_original_process_observation(self):
        self.stop_display();self.display.unlink();new=self.start_display()
        result=self.inspect();self.assertEqual(result['context']['server'],'gone')
        self.assertEqual(result['context']['filesystem'],'changed-or-unreadable');self.assertFalse(result['context']['live_context'])
        self.assertNotEqual(new.pid,result['descriptor']['context']['server']['pid'])

    def test_inspection_never_constructs_live_authority(self):
        with patch.object(linked.Authority,'__init__',side_effect=AssertionError('live authority constructed')),patch.object(authority.Authority,'__init__',side_effect=AssertionError('old authority constructed')):
            self.assertEqual(self.inspect()['format'],2)

    def test_original_reader_still_refuses_linked_format(self):
        before=self.snapshot()
        with self.assertRaises(ValueError):original.inspect(self.next_authority.root,self.store.root)
        self.assertEqual(before,self.snapshot())

    def test_explicit_lease_works_and_unpassed_lease_refuses(self):
        with self.store.lease() as lease:
            self.refuse();before=self.snapshot();self.assertEqual(self.inspect(lease)['format'],2);self.assertEqual(before,self.snapshot())

    def test_wrong_coordinator_lease_refuses(self):
        other=j.Store(self.root/'other-coordinator',create=True)
        try:
            with other.lease() as lease,self.assertRaises(ValueError):self.inspect(lease)
        finally:other.close()

    def test_missing_publication_refuses_without_repair(self):
        self.record_path().unlink();self.refuse();self.assertFalse(self.record_path().exists())

    def test_missing_or_changed_ancestor_checkpoint_refuses(self):
        path=self.checkpoint_path();data=path.read_bytes();path.rename(path.with_suffix('.retained'));self.refuse()
        path.write_bytes(data);path.chmod(0o600);self.refuse()

    def test_changed_ancestor_cleanup_refuses(self):
        path=Path(self.archived['cleanup']['object']['path']);value=json.loads(path.read_text());value['verified']['members']['dms']['pending_ids']=[]
        f.save(path,value);self.refuse()

    def test_changed_canonical_prefix_refuses(self):
        path=self.store.root/'journal.json';value=json.loads(path.read_text());value['events']=value['events'][:1];f.save(path,value);self.refuse()

    def test_unknown_descriptor_or_ready_seal_refuses(self):
        path=self.next_authority.root/'ready.json';f.save(path,{'format':1,'descriptor':j.digest(self.next_authority.descriptor)});self.refuse()
        f.save(path,{'format':2,'descriptor':j.digest(self.next_authority.descriptor)})
        path=self.next_authority.root/'authority.json';value=json.loads(path.read_text());value['format']=3;f.save(path,value);self.refuse()

    def test_missing_lock_is_never_created(self):
        path=self.next_authority.root/'authority.lock';path.unlink();self.refuse();self.assertFalse(path.exists())

    def test_same_bytes_replaced_claim_is_not_original_evidence(self):
        path=self.claim_path();data=path.read_bytes();path.rename(path.with_name('retained-claim'));path.write_bytes(data);path.chmod(0o600);self.refuse()

    def test_missing_guardian_is_unconfirmed_not_transport_cleanup(self):
        self.bound_ready();self.close_guardian();self.next_receipt.rename(self.next_receipt.with_suffix('.retained'))
        value=self.inspect();self.assertEqual(value['activations'][-1]['closure']['state'],'receipt-missing')
        self.assertFalse(value['activations'][-1]['closure']['verified_process_closure']);self.assertFalse(value['transport_cleanup_verified'])

    def test_wrong_guardian_or_bound_observation_refuses(self):
        self.bound_ready();self.close_guardian();value=json.loads(self.next_receipt.read_text());value['owner']['token']=uuid.uuid4().hex
        f.save(self.next_receipt,value);self.refuse()

    def test_cli_details_summary_and_error_remain_readonly(self):
        command=[sys.executable,'-B',str(ROOT/'scripts/inspect-collection-authority.py'),'--authority',str(self.next_authority.root),'--coordinator',str(self.store.root)]
        before=self.snapshot();expected=self.inspect()
        for args in ([],['--details']):
            q=subprocess.run(command+args,capture_output=True,text=True,timeout=8);self.assertEqual(0,q.returncode,q.stderr)
            value=json.loads(q.stdout)
            self.assertEqual(value,expected if args else {key:expected[key] for key in value})
            self.assertFalse(value['activation_authorized']);self.assertEqual(before,self.snapshot())
        self.record_path().unlink();before=self.snapshot()
        q=subprocess.run(command,capture_output=True,text=True,timeout=8);self.assertEqual(q.returncode,1);self.assertFalse(json.loads(q.stderr)['activation_authorized']);self.assertEqual(before,self.snapshot())

    def test_later_canonical_append_keeps_snapshot_current(self):
        first=self.inspect();changed=copy.deepcopy(self.spec)
        for role in j.ROLES:changed['members'][role]['artifact_sha256']=j.digest(['later',role])
        self.record({'type':'register','spec':changed})
        next_value=self.inspect();self.assertEqual(next_value['journal']['events'],first['journal']['events']+1)
        self.assertEqual(next_value['lineage'],first['lineage']);self.assertNotEqual(next_value['journal']['sha256'],first['journal']['sha256'])


for name in dir(f.LinkedAuthorityTests):
    value=getattr(f.LinkedAuthorityTests,name)
    if callable(value) and not name.startswith(('test','_')) and not hasattr(LinkedInspectionTests,name):setattr(LinkedInspectionTests,name,value)

if __name__=='__main__':unittest.main()
