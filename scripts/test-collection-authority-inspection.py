#!/usr/bin/env python3
"""Retained-evidence inspection uses actual owned test process identities."""
import copy,importlib.util,json,os
from pathlib import Path
import select,signal,socket,subprocess,sys,unittest,uuid
from unittest.mock import patch
ROOT=Path(__file__).resolve().parents[1];sys.path.insert(0,str(ROOT/'native/collections'))
import authority_inspection as inspect
import runtime_authority as a
import journal as j
import admission
spec=importlib.util.spec_from_file_location('inspection_handoff_fixture',Path(__file__).with_name('test-collection-authority-handoff.py'))
h=importlib.util.module_from_spec(spec);spec.loader.exec_module(h)

class InspectionTests(unittest.TestCase):
    provision=h.HandoffTests.provision
    receive=h.HandoffTests.receive
    cleanup=h.HandoffTests.cleanup
    launch_guardian=h.HandoffTests.launch_guardian
    send=h.HandoffTests.send
    start_claims=h.HandoffTests.start_claims
    record=h.HandoffTests.record
    write_binding=h.HandoffTests.write_binding
    bind=h.HandoffTests.bind

    def setUp(self):
        h.fixture.AuthorityTests.setUp(self);self.gates=[];self.guardians=[]
        self.server.close();self.display.unlink();self.compositor=self.start_display()
        self.provision();self.guardian,self.owner,self.shell,self.receipt=self.launch_guardian();self.permit['shell']=self.shell
        with self.store.lease() as lease:self.ticket=self.authority.reserve(self.store,lease,self.gate,self.permit,guardian_receipt=self.receipt)
        self.current=h.shared.Gate(self.gate);self.gates.append(self.current);self.current.issue(self.permit,self.ticket);self.start_claims()
        self.loaded=h.model.loaded(self.spec,self.permit['epoch']);self.record({'type':'adopt','observation':h.model.ready(self.spec,self.loaded)})
        self.binding_path=self.write_binding();self.bind()

    def start_display(self):
        child=subprocess.Popen([sys.executable,'-B',__file__,'--display',str(self.display)],stdout=subprocess.PIPE,stderr=subprocess.PIPE)
        self.children.append(child);self.assertTrue(select.select([child.stdout],[],[],5)[0]);self.assertEqual(child.stdout.readline(),b'ready\n');return child

    def stop_display(self):self.compositor.terminate();self.assertEqual(self.compositor.wait(timeout=3),-signal.SIGTERM)

    def close_guardian(self):
        self.send(self.guardian,{'command':'stop','request':uuid.uuid4().hex});self.assertEqual(self.guardian.wait(timeout=5),0)

    def run_inspection(self):return inspect.inspect(self.authority.root,self.store.root)

    def snapshot(self):
        return {str(p.relative_to(self.root)):(p.stat().st_ino,p.read_bytes()) for p in self.root.rglob('*') if p.is_file()}

    def refused(self):
        before=self.snapshot()
        with self.assertRaises((OSError,ValueError)):self.run_inspection()
        self.assertEqual(before,self.snapshot())

    def test_live_snapshot_is_read_only_and_grants_no_capability(self):
        before=self.snapshot();result=self.run_inspection();self.assertEqual(before,self.snapshot())
        self.assertEqual(result['context'],{'server':'live','filesystem':'original','live_context':True})
        self.assertEqual(result['journal']['committed'],self.loaded)
        self.assertFalse(result['activation_authorized']);self.assertFalse(result['transport_cleanup_verified'])
        self.assertFalse(result['activations'][0]['closure']['verified_process_closure'])

    def test_dead_compositor_does_not_require_repair_or_live_authority(self):
        self.stop_display();before=self.snapshot();result=self.run_inspection()
        self.assertEqual(result['context']['server'],'gone');self.assertFalse(result['context']['live_context']);self.assertEqual(before,self.snapshot())
        with self.assertRaises(ValueError):a.Authority(self.authority.root)

    def test_replaced_socket_same_name_preserves_original_descriptor(self):
        original=self.authority.descriptor;self.stop_display();self.display.unlink();new=self.start_display()
        result=self.run_inspection();self.assertEqual(result['descriptor'],original);self.assertEqual(result['context']['server'],'gone')
        self.assertEqual(result['context']['filesystem'],'changed-or-unreadable');self.assertNotEqual(new.pid,original['context']['server']['pid'])

    def test_original_closure_verifies_separately_from_transport(self):
        self.stop_display();self.close_guardian();result=self.run_inspection();closure=result['activations'][0]['closure']
        self.assertTrue(closure['verified_process_closure']);self.assertFalse(result['transport_cleanup_verified']);self.assertFalse(result['activation_authorized'])

    def test_missing_guardian_receipt_is_unknown_not_verified(self):
        self.stop_display();self.close_guardian();self.receipt.rename(self.receipt.with_suffix('.retained'));result=self.run_inspection()
        self.assertEqual(result['activations'][0]['closure']['state'],'receipt-missing');self.assertFalse(result['activations'][0]['closure']['verified_process_closure'])

    def test_wrong_guardian_closed_receipt_refused(self):
        self.close_guardian();body=json.loads(self.receipt.read_text());body['owner']['token']=uuid.uuid4().hex;h.save(self.receipt,body);self.refused()

    def test_missing_bound_gate_seal_refused(self):
        (self.gate/'issued.json').unlink();self.refused()

    def test_changed_bound_claim_refused(self):
        path=self.gate/'admission.json';body=json.loads(path.read_text());body['claims']['theme']['instance']=uuid.uuid4().hex;h.save(path,body);self.refused()

    def test_replaced_runtime_binding_with_identical_bytes_refused(self):
        raw=self.binding_path.read_bytes();self.binding_path.rename(self.binding_path.with_suffix('.retained'));self.binding_path.write_bytes(raw);self.binding_path.chmod(0o600);self.refused()

    def test_changed_runtime_body_refused(self):
        body=json.loads(self.binding_path.read_text());body['helpers']['theme']=[];h.save(self.binding_path,body);self.refused()

    def test_missing_history_refused_without_recreation(self):
        (self.store.root/'journal.json').unlink();self.refused()

    def test_unknown_reservation_marker_refused(self):
        path=self.authority.root/'reservation.json';body=json.loads(path.read_text());body['format']=99;h.save(path,body);self.refused()

    def test_mirror_change_refused(self):
        path=self.store.root/a.MIRROR;body=json.loads(path.read_text());body['id']=uuid.uuid4().hex;h.save(path,body);self.refused()

    def test_conflicting_coordinator_lease_does_not_modify_files(self):
        with self.store.lease():self.refused()

    def test_conflicting_authority_transaction_does_not_modify_files(self):
        with self.authority.transaction():self.refused()

    def test_missing_lock_is_not_created(self):
        (self.store.root/'coordinator.lock').unlink();self.refused()

    def test_other_pid_namespace_is_unobservable(self):
        value=admission.identity(os.getpid());value['namespace']='pid:[1]'
        self.assertEqual(inspect.process_state(value),'unobservable')
        value['boot']='00000000-0000-0000-0000-000000000000';self.assertEqual(inspect.process_state(value),'previous-boot')

    def test_snapshot_replacement_during_inspection_refused(self):
        original=inspect.Reader.verify
        def changed(reader):
            path=self.authority.root/'ready.json';h.save(path,json.loads(path.read_text()));original(reader)
        with patch.object(inspect.Reader,'verify',changed),self.assertRaises(ValueError):self.run_inspection()

    def test_different_coordinator_or_relative_root_refused(self):
        with self.assertRaises(ValueError):inspect.inspect(self.authority.root,self.root/'other')
        with self.assertRaises(ValueError):inspect.inspect(Path('.'),self.store.root)

    def test_cli_summary_details_and_invalid_inputs_are_read_only(self):
        command=[sys.executable,'-B',str(ROOT/'scripts/inspect-collection-authority.py'),'--authority',str(self.authority.root),'--coordinator',str(self.store.root)]
        before=self.snapshot()
        for options in ([],['--details']):
            q=subprocess.run(command+options,capture_output=True,text=True,timeout=5);self.assertEqual(q.returncode,0,q.stderr);value=json.loads(q.stdout)
            self.assertEqual('descriptor' in value,bool(options));self.assertFalse(value['activation_authorized'])
        q=subprocess.run(command[:-1]+[str(self.root/'missing')],capture_output=True,text=True,timeout=5);self.assertEqual(q.returncode,1);self.assertFalse(json.loads(q.stderr)['activation_authorized'])
        self.assertEqual(before,self.snapshot())

if __name__=='__main__':
    if len(sys.argv)>1 and sys.argv[1]=='--display':
        server=socket.socket(socket.AF_UNIX,socket.SOCK_STREAM);server.bind(sys.argv[2]);server.listen(128);print('ready',flush=True);signal.pause()
    else:unittest.main()
