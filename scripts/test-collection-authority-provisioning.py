#!/usr/bin/env python3
"""Private-process provisioning restart and preservation boundaries."""
import copy
import importlib.util
import json
import os
from pathlib import Path
import select
import signal
import subprocess
import sys
import unittest
import uuid
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / 'native/collections'))
import authority_provisioning as p
import runtime_authority as a
import journal as j

spec = importlib.util.spec_from_file_location('authority_fixture', ROOT / 'scripts/test-collection-runtime-authority.py')
fixture = importlib.util.module_from_spec(spec);spec.loader.exec_module(fixture)


class ProvisioningTests(unittest.TestCase):
    def setUp(self):
        fixture.AuthorityTests.setUp(self)
        with self.store.lease():pass
    cleanup = fixture.AuthorityTests.cleanup

    def step(self):
        if not hasattr(self, 'attempt'):self.attempt = uuid.uuid4().hex
        with self.store.lease() as lease:return p.step(self.display, self.store, lease, self.attempt)

    def finish(self):
        if not hasattr(self, 'attempt'):self.attempt = uuid.uuid4().hex
        with self.store.lease() as lease:self.authority = p.provision(self.display, self.store, lease, self.attempt)
        return self.authority

    def through(self, phase):
        for _ in range(16):
            record = p.read(self.store)
            if record is not None:
                if phase == 'planned':return record
                if phase == 'complete' and record['complete']:return record
                if '-' in phase:
                    kind, state = phase.split('-')
                    row = p.active(record, kind)
                    if row is not None and (row['object'] is not None if state == 'prepared' else row['published']):return record
            r = self.step()
            if r['phase'] == phase:return r
        self.fail('phase was not reached: ' + phase)

    def snapshot(self):
        out = {}
        for base in (self.runtime, self.store.root, self.gate):
            for path in [base] + sorted(base.rglob('*')):
                st = path.lstat()
                out[str(path.relative_to(self.root))] = (st.st_ino, st.st_mode, st.st_nlink, path.read_bytes() if path.is_file() else None)
        return out

    def refused(self, action=None):
        before = self.snapshot()
        with self.assertRaises((j.Conflict, FileNotFoundError, OSError, ValueError)):(action or self.step)()
        self.assertEqual(before, self.snapshot())

    def row(self, kind):return p.active(p.read(self.store), kind)

    def test_ten_steps_and_completed_retry_preserve_exact_objects(self):
        phases = [self.step()['phase'] for _ in range(10)]
        self.assertEqual(phases, ['planned','claim-prepared','claim-published','root-prepared','seal-prepared','seal-published','mirror-prepared','mirror-published','root-published','complete'])
        before = self.snapshot();self.finish();self.assertEqual(before, self.snapshot())

    def test_completed_retry_preserves_later_registration_and_reservation(self):
        self.finish()
        with self.store.lease() as lease:
            self.store.record(lease, {'type':'register','spec':self.spec})
            self.authority.reserve(self.store, lease, self.gate, self.permit)
        self.authority.close();self.authority = None
        before = self.snapshot();self.finish();self.assertEqual(before, self.snapshot())

    def test_original_reader_opens_completed_descriptor(self):
        self.finish();before = self.snapshot()
        source = ROOT / 'tests/fixtures/runtime-authority-v1/writer.py'
        self.assertTrue(source.is_file())
        loader = importlib.util.spec_from_file_location('original_authority', source)
        original = importlib.util.module_from_spec(loader);loader.loader.exec_module(original)
        opened = original.Authority(self.authority.root);opened.close()
        self.assertEqual(before, self.snapshot())

    def test_child_controllers_are_killed_after_each_durable_step(self):
        self.attempt = uuid.uuid4().hex;identity = self.step()['authority'];phases = ['planned']
        request = self.root/'request.json'
        fixture.save(request, {'display':str(self.display),'store':str(self.store.root),'attempt':self.attempt})
        for _ in range(9):
            child = subprocess.Popen([sys.executable,'-B',__file__,'--worker',str(request)], stdout=subprocess.PIPE, stderr=subprocess.PIPE)
            self.children.append(child)
            self.assertTrue(select.select([child.stdout],[],[],6)[0], 'child response deadline')
            line = child.stdout.readline()
            if not line:
                child.wait(timeout=3);self.fail('child exited before durable result: '+child.stderr.read().decode())
            result = json.loads(line);phases.append(result['phase'])
            if identity is None:identity = result['authority']
            self.assertEqual(identity,result['authority']);child.kill();self.assertEqual(child.wait(timeout=5),-signal.SIGKILL)
        self.assertEqual(phases[-1],'complete');before = self.snapshot();self.finish();self.assertEqual(before,self.snapshot())

    def test_unbound_file_is_retained_after_create_before_binding(self):
        self.through('planned');original = a.write_new
        def broken(fd, name, value):
            original(fd,name,value)
            if '-claim-' in name:raise OSError('lost create response')
        with patch.object(a,'write_new',broken),self.assertRaises(OSError):self.step()
        old = copy.deepcopy(self.row('claim'));path = Path(old['path']);before = path.read_bytes();inode = path.stat().st_ino
        self.assertIsNone(old['object']);self.finish()
        self.assertEqual(path.read_bytes(),before);self.assertEqual(path.stat().st_ino,inode)
        r = p.read(self.store);self.assertTrue(r['attempts'][0]['abandoned']);self.assertNotEqual(self.row('claim')['path'],str(path))

    def test_partial_unbound_root_is_retained_untouched(self):
        self.through('claim-published');original = a.write_new
        def broken(fd,name,value):
            original(fd,name,value)
            if name == 'authority.json':raise OSError('interrupted root construction')
        with patch.object(a,'write_new',broken),self.assertRaises(OSError):self.step()
        old = Path(self.row('root')['path']);files = {str(x):x.read_bytes() for x in old.iterdir()};inode = old.stat().st_ino
        self.finish();self.assertEqual(files,{str(x):x.read_bytes() for x in old.iterdir()});self.assertEqual(inode,old.stat().st_ino)

    def test_all_unbound_stages_retry_after_allocation_without_create(self):
        for kind,phase in [('claim','planned'),('root','claim-published'),('seal','root-prepared'),('mirror','seal-published')]:
            with self.subTest(kind=kind):
                self.through(phase);original = p.save
                def broken(store,value):
                    original(store,value)
                    row = p.active(value,kind)
                    if row is not None and row['object'] is None:raise OSError('allocation saved')
                with patch.object(p,'save',broken),self.assertRaises(OSError):self.step()
                old = self.row(kind)['path'];self.assertFalse(os.path.lexists(old));self.step()
                self.assertNotEqual(old,self.row(kind)['path']);self.assertTrue(p.read(self.store)['attempts'][-2]['abandoned'])
        self.finish()

    def test_all_bound_stage_retries_after_save_lost_response(self):
        for kind,phase in [('claim','planned'),('root','claim-published'),('seal','root-prepared'),('mirror','seal-published')]:
            with self.subTest(kind=kind):
                self.through(phase);original = p.save
                def broken(store,value):
                    original(store,value)
                    row = p.active(value,kind)
                    if row is not None and row['object'] is not None:raise OSError('binding saved')
                with patch.object(p,'save',broken),self.assertRaises(OSError):self.step()
                old = copy.deepcopy(self.row(kind));self.step();self.assertEqual(old['object'],self.row(kind)['object'])
        self.finish()

    def test_all_publications_retry_after_rename_lost_response(self):
        for kind,phase in [('claim','claim-prepared'),('seal','seal-prepared'),('mirror','mirror-prepared'),('root','mirror-published')]:
            with self.subTest(kind=kind):
                self.through(phase);original = p.rename_no_replace
                def broken(*args):original(*args);raise OSError('rename response lost')
                row = copy.deepcopy(self.row(kind))
                with patch.object(p,'rename_no_replace',broken),self.assertRaises(OSError):self.step()
                dest = p.target(p.read(self.store),kind);self.assertEqual(dest.stat().st_ino,row['object']['inode'])
                self.assertFalse(Path(row['path']).exists());self.step();self.assertTrue(self.row(kind)['published'])
        self.finish()

    def test_root_promotion_retry_flushes_runtime_parent(self):
        self.through('mirror-published');original = p.rename_no_replace
        def broken(fd,src,dst):
            with patch.object(p.os,'fsync',side_effect=OSError('lost directory sync')):original(fd,src,dst)
        with patch.object(p,'rename_no_replace',broken),self.assertRaises(OSError):self.step()
        with patch.object(p,'sync',wraps=p.sync) as sync:self.step()
        self.assertIn(((self.runtime,),{}),sync.call_args_list)

    def test_missing_bound_root_refuses_without_recreation(self):
        self.through('root-prepared');path = Path(self.row('root')['path']);path.rename(path.with_name(path.name+'-preserved'));self.refused()

    def test_replaced_bound_stage_refuses_without_adoption(self):
        self.through('claim-prepared');path = Path(self.row('claim')['path']);raw = path.read_bytes();path.rename(path.with_name(path.name+'-old'));path.write_bytes(raw);path.chmod(0o600);self.refused()

    def test_changed_bound_root_contents_refuse(self):
        self.through('root-prepared');fixture.save(Path(self.row('root')['path'])/'reservation.json',{'invented':True});self.refused()

    def test_missing_published_claim_is_not_recreated(self):
        self.through('claim-published');(self.runtime/p.CLAIM).unlink();self.refused()

    def test_missing_published_seal_is_not_recreated(self):
        self.through('seal-published');(self.runtime/a.ANCHOR_SEAL).unlink();self.refused()

    def test_missing_published_mirror_is_not_recreated(self):
        self.through('mirror-published');(self.store.root/a.MIRROR).unlink();self.refused()

    def test_missing_completed_root_is_not_recreated(self):
        self.finish();self.authority.close();self.authority=None
        (self.runtime/a.ANCHOR).rename(self.runtime/'retained-root');self.refused()

    def test_fixed_root_conflict_never_overwrites_empty_directory(self):
        self.through('mirror-published');original = p.rename_no_replace
        def conflict(fd,src,dst):os.mkdir(dst,mode=0o700,dir_fd=fd);original(fd,src,dst)
        with patch.object(p,'rename_no_replace',conflict),self.assertRaises(j.Conflict):self.step()
        self.assertEqual(list((self.runtime/a.ANCHOR).iterdir()),[]);self.refused()

    def test_other_attempt_refuses_original_coordinator(self):
        self.step();self.attempt=uuid.uuid4().hex;self.refused()

    def test_other_coordinator_refuses_published_claim(self):
        self.through('claim-published');other=j.Store(self.root/'other',create=True)
        try:
            with other.lease() as lease:self.refused(lambda:p.step(self.display,other,lease,uuid.uuid4().hex))
            self.assertFalse((other.root/p.RECORD).exists())
        finally:other.close()

    def test_old_provisioner_refuses_record_and_runtime_claim(self):
        self.step()
        with self.store.lease() as lease:self.refused(lambda:a.Authority.provision(self.display,self.store,lease))
        self.through('claim-published');other=j.Store(self.root/'other',create=True)
        try:
            with other.lease() as lease:self.refused(lambda:a.Authority.provision(self.display,other,lease))
        finally:other.close()

    def test_legacy_partial_root_is_not_repaired(self):
        (self.runtime/a.ANCHOR).mkdir(mode=0o700);self.refused();self.assertFalse((self.store.root/p.RECORD).exists())

    def test_unknown_attempt_format_preserved(self):
        self.step();value=p.read(self.store);value['format']=2;fixture.save(self.store.root/p.RECORD,value);self.refused()

    def test_changed_context_socket_refused(self):
        self.step();self.display.rename(self.runtime/'retained-socket');self.refused()

    def test_changed_coordinator_lock_refused(self):
        self.step();path=self.store.root/'coordinator.lock';path.rename(self.store.root/'retained-lock');path.write_bytes(b'');path.chmod(0o600);self.refused()

    def test_unfinished_setup_cannot_reset_registration(self):
        self.through('claim-published')
        with self.store.lease() as lease:self.store.record(lease,{'type':'register','spec':self.spec})
        self.refused()

    def test_history_cap_preserves_unbound_orphans(self):
        self.step()
        original = p.save
        def broken(store,value):
            original(store,value)
            if value['attempts']:raise OSError('allocated')
        with patch.object(p,'save',broken):
            for _ in range(p.MAX_ATTEMPTS):
                with self.assertRaises(OSError):self.step()
        self.refused();self.assertEqual(len(p.read(self.store)['attempts']),p.MAX_ATTEMPTS)


if __name__ == '__main__':
    if len(sys.argv)>1 and sys.argv[1]=='--worker':
        request=json.loads(Path(sys.argv[2]).read_text());store=j.Store(request['store'])
        with store.lease() as lease:result=p.step(request['display'],store,lease,request['attempt'])
        print(json.dumps(result),flush=True);signal.pause()
    else:unittest.main()
