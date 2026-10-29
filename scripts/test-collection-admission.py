#!/usr/bin/env python3
"""One-use native admission with actual guard processes and process loss."""
import copy
import json
import os
from pathlib import Path
import select
import signal
import subprocess
import sys
import tempfile
import unittest
import uuid
from unittest.mock import patch

sys.path.insert(0,str(Path(__file__).resolve().parents[1]/'native/collections'))
import admission as a
import journal as j


def spec():
    return {'format':1,'name':'stable','members':{
        role:{'plugin_id':a.CANONICAL[role]+'.qualification-stable','profile':'qualification-stable',
              'artifact_sha256':j.digest(['fixture',role]),'endpoint_ref':None,
              'paths':{kind:'/isolated/'+role+'/'+kind for kind in j.PATHS}}
        for role in j.ROLES}}


def binding(permit,role):
    member=permit['specification']['members'][role]
    return {'permit':permit['id'],'epoch':permit['epoch'],'collection':j.digest(permit['specification']),
            **{k:member[k] for k in ('plugin_id','profile','artifact_sha256')}}


class AdmissionTests(unittest.TestCase):
    def setUp(self):
        self.temp=tempfile.TemporaryDirectory(prefix='omabit-admission-unit-');self.root=Path(self.temp.name)/'gate';self.root.mkdir(mode=0o700)
        self.gate=a.Gate(self.root);self.children=[];self.claims=[]
        self.addCleanup(self.cleanup)

    def cleanup(self):
        for child in self.children:
            if child.poll() is None:child.kill();child.wait(timeout=4)
            for stream in (child.stdin,child.stdout,child.stderr):
                if stream:stream.close()
        for claim in self.claims:
            if claim.fd is not None:
                try:claim.end()
                except (OSError,ValueError):pass
        self.gate.close();self.temp.cleanup()

    def permit(self,pid=None):
        return {'id':uuid.uuid4().hex,'request':uuid.uuid4().hex,'epoch':uuid.uuid4().hex,
                'specification':spec(),'shell':a.identity(os.getpid() if pid is None else pid)}

    def issue(self,pid=None):
        p=self.permit(pid);self.gate.issue(p);return p

    def claim(self,p,role='theme',token=None):
        c=self.gate.claim(binding(p,role),role,token or uuid.uuid4().hex);self.claims.append(c);return c

    def start_guard(self,p,role='theme',token=None,expected=None,root=None):
        child=subprocess.Popen([sys.executable,'-B',str(Path(a.__file__).with_name('admission_guard.py')),'--root',str(root or self.root)],
                               stdin=subprocess.PIPE,stdout=subprocess.PIPE,stderr=subprocess.PIPE)
        self.children.append(child)
        request={'expected':expected or binding(p,role),'role':role,'instance':token or uuid.uuid4().hex}
        child.stdin.write(j.canonical(request)+b'\n');child.stdin.flush();return child

    def receive(self,child):
        self.assertTrue(select.select([child.stdout],[],[],5)[0],'guard response deadline')
        data=child.stdout.readline();self.assertTrue(data,'guard closed without response');return json.loads(data)

    def test_missing_permit_and_missing_root_never_bootstrap(self):
        before=list(self.root.iterdir())
        with self.assertRaises(FileNotFoundError):self.gate.status()
        self.assertEqual(list(self.root.iterdir()),before)
        child=self.start_guard(self.permit());self.assertFalse(self.receive(child)['ok']);self.assertEqual(child.wait(timeout=3),1)
        self.assertFalse((self.root/'admission.json').exists())
        missing=self.root/'missing';child=self.start_guard(self.permit(),root=missing)
        self.assertFalse(self.receive(child)['ok']);self.assertEqual(child.wait(timeout=3),1);self.assertFalse(missing.exists())

    def test_exact_issue_retry_preserves_bytes_and_does_not_reopen(self):
        p=self.issue();before=(self.root/'admission.json').read_bytes();self.gate.issue(p)
        self.assertEqual((self.root/'admission.json').read_bytes(),before)
        request=uuid.uuid4().hex;closed=self.gate.stop_admission(p['id'],request)
        self.assertEqual(self.gate.issue(p),closed)
        with self.assertRaises(j.Conflict):self.gate.issue(self.permit())
        with self.assertRaises(j.Conflict):self.gate.stop_admission(p['id'],uuid.uuid4().hex)

    def test_three_actual_guards_get_distinct_member_claims_and_retained_endings(self):
        p=self.issue();guards=[self.start_guard(p,role) for role in j.ROLES]
        rows=[self.receive(child)['admission'] for child in guards]
        self.assertEqual(len({row['instance'] for row in rows}),3)
        self.assertEqual({row['owner']['pid'] for row in rows},{child.pid for child in guards})
        self.assertTrue(all(row['admitted'] and row['state']=='open' for row in rows))
        self.assertEqual(set(self.gate.status()['record']['claims']),set(j.ROLES))
        for child in guards:child.stdin.close();self.assertEqual(child.wait(timeout=3),0)
        self.assertTrue(all(c['state']=='ended' for c in self.gate.status()['record']['claims'].values()))
        replacement=self.start_guard(p);self.assertFalse(self.receive(replacement)['ok']);self.assertEqual(replacement.wait(timeout=3),1)

    def test_duplicate_live_component_never_gets_a_second_grant(self):
        p=self.issue();first=self.start_guard(p);self.assertTrue(self.receive(first)['ok'])
        before=(self.root/'admission.json').read_bytes();second=self.start_guard(p)
        self.assertFalse(self.receive(second)['ok']);self.assertEqual(second.wait(timeout=3),1)
        self.assertEqual((self.root/'admission.json').read_bytes(),before);self.assertIsNone(first.poll())

    def test_actual_guard_sigkill_retains_claim_after_lock_disappears(self):
        p=self.issue();token=uuid.uuid4().hex;child=self.start_guard(p,token=token);row=self.receive(child)['admission']
        before=(self.root/'admission.json').read_bytes();child.kill();self.assertEqual(child.wait(timeout=3),-signal.SIGKILL)
        self.assertFalse(self.gate.status()['claim_owners_live']['theme']);self.assertEqual((self.root/'admission.json').read_bytes(),before)
        replacement=self.start_guard(p,token=token);self.assertFalse(self.receive(replacement)['ok']);replacement.wait(timeout=3)
        self.assertEqual(self.gate.status()['record']['claims']['theme']['owner'],row['owner'])

    def test_durable_close_notifies_old_guard_and_blocks_new_member(self):
        p=self.issue();old=self.start_guard(p);first=self.receive(old)['admission']
        request=uuid.uuid4().hex;self.gate.stop_admission(p['id'],request)
        closing=self.receive(old)['admission'];self.assertEqual(closing['state'],'closing');self.assertEqual(closing['instance'],first['instance'])
        self.assertTrue(closing['admitted']);self.assertIsNone(old.poll())
        new=self.start_guard(p,'dms');self.assertFalse(self.receive(new)['ok']);new.wait(timeout=3)
        self.assertNotIn('dms',self.gate.status()['record']['claims'])
        # No claim is marked transport-closed, and the old application is not
        # killed just because new admission was closed.
        self.assertEqual(self.gate.status()['record']['claims']['theme']['state'],'admitted')

    def test_wrong_artifact_epoch_profile_and_permit_never_claim(self):
        p=self.issue()
        for key,replacement in (('permit','a'*32),('epoch','a'*32),('collection','a'*64),
                                ('artifact_sha256','a'*64),('profile','qualification-dev'),('plugin_id',a.CANONICAL['theme']+'.qualification-dev')):
            with self.subTest(key=key):
                expected=dict(binding(p,'theme'),**{key:replacement});child=self.start_guard(p,expected=expected)
                self.assertFalse(self.receive(child)['ok']);self.assertEqual(child.wait(timeout=3),1)
                self.assertFalse(self.gate.status()['record']['claims'])

    def test_guard_from_wrong_parent_cannot_adopt_live_shell_permit(self):
        p=self.issue(os.getppid());child=self.start_guard(p)
        self.assertFalse(self.receive(child)['ok']);self.assertEqual(child.wait(timeout=3),1)
        self.assertFalse(self.gate.status()['record']['claims'])

    def test_dead_shell_cannot_issue_and_same_path_new_process_cannot_claim(self):
        first=subprocess.Popen([sys.executable,'-B','-c','import time;time.sleep(60)']);self.children.append(first)
        p=self.issue(first.pid);first.terminate();first.wait(timeout=3)
        self.assertFalse(self.gate.status()['shell_live'])
        replacement=subprocess.Popen([sys.executable,'-B','-c','import time;time.sleep(60)']);self.children.append(replacement)
        self.assertNotEqual(a.identity(replacement.pid),p['shell'])
        child=self.start_guard(p);self.assertFalse(self.receive(child)['ok']);child.wait(timeout=3)
        other=self.root/'fresh';other.mkdir(mode=0o700);gate=a.Gate(other)
        try:
            with self.assertRaises(j.Conflict):gate.issue(p)
            self.assertFalse((other/'admission.json').exists())
        finally:gate.close()

    def test_claim_persists_before_return_and_retry_after_lost_sync_is_same_owner_only(self):
        p=self.issue(os.getppid());token=uuid.uuid4().hex
        with patch.object(self.gate,'_sync',side_effect=OSError('lost directory fsync')),self.assertRaises(OSError):self.claim(p,token=token)
        saved=self.gate.status()['record']['claims']['theme'];self.assertEqual(saved['instance'],token)
        with patch.object(self.gate,'_save',side_effect=AssertionError('must not create a second claim')):
            claim=self.claim(p,token=token);self.assertEqual(claim.status()['instance'],token)
        claim.end()
        with self.assertRaises(j.Conflict):self.claim(p,token=token)

    def test_failed_claim_write_returns_no_grant(self):
        p=self.issue(os.getppid());before=(self.root/'admission.json').read_bytes()
        with patch.object(self.gate,'_save',side_effect=OSError('write failed')),self.assertRaises(OSError):self.claim(p)
        self.assertEqual((self.root/'admission.json').read_bytes(),before);self.assertFalse(self.gate.status()['record']['claims'])

    def test_close_lost_sync_still_fences_and_retry_is_exact(self):
        p=self.issue();request=uuid.uuid4().hex
        with patch.object(self.gate,'_sync',side_effect=OSError('lost fsync')),self.assertRaises(OSError):self.gate.stop_admission(p['id'],request)
        child=self.start_guard(p);self.assertFalse(self.receive(child)['ok']);child.wait(timeout=3)
        before=(self.root/'admission.json').read_bytes();self.gate.stop_admission(p['id'],request)
        self.assertEqual((self.root/'admission.json').read_bytes(),before)

    def test_unknown_format_and_missing_issued_record_refuse_unchanged(self):
        p=self.issue();path=self.root/'admission.json';value=json.loads(path.read_text());value['format']=99;path.write_text(json.dumps(value));before=path.read_bytes()
        with self.assertRaises(j.Conflict):self.gate.status()
        with self.assertRaises(j.Conflict):self.gate.issue(p)
        self.assertEqual(path.read_bytes(),before)
        path.unlink()
        with self.assertRaises(j.Conflict):self.gate.issue(p)
        self.assertFalse(path.exists())

    def test_unsealed_claim_refuses_but_same_initial_issue_can_finish_seal(self):
        p=self.permit(os.getppid())
        with patch.object(self.gate,'_seal',side_effect=OSError('seal interruption')),self.assertRaises(OSError):self.gate.issue(p)
        with self.assertRaises(FileNotFoundError):self.claim(p)
        self.gate.issue(p);claim=self.claim(p);self.assertEqual(claim.status()['state'],'open')
        (self.root/'issued.json').unlink()
        with self.assertRaises(j.Conflict):self.gate.issue(p)

    def test_linked_state_and_closed_store_refuse(self):
        p=self.issue();path=self.root/'admission.json';retained=self.root/'retained.json';path.rename(retained);path.symlink_to(retained)
        with self.assertRaises(OSError):self.gate.status()
        path.unlink();os.link(retained,path)
        with self.assertRaises(j.Conflict):self.gate.status()
        self.gate.close()
        with self.assertRaises(j.Conflict):self.gate.status()

    def test_canonical_default_identity_is_never_admitted(self):
        p=self.permit();p['specification']['members']['theme']['profile']='default';p['specification']['members']['theme']['plugin_id']=a.CANONICAL['theme']
        with self.assertRaises(j.Conflict):self.gate.issue(p)
        self.assertFalse((self.root/'admission.json').exists())

    def test_snapshot_reopens_after_atomic_publication(self):
        p=self.issue();writer=a.Gate(self.root);self.addCleanup(writer.close)
        original=self.gate._open;observed=[];request=uuid.uuid4().hex
        def opening(name,flags,mode=0o600):
            fd=original(name,flags,mode)
            if name=='admission.json' and not observed:
                writer.stop_admission(p['id'],request)
                observed.append(os.fstat(fd).st_nlink)
            return fd
        with patch.object(self.gate,'_open',side_effect=opening):row=self.gate.status()['record']
        self.assertEqual(observed,[0]);self.assertEqual(row['state'],'closing')
        self.assertEqual(row['close_request'],request)

    def test_snapshot_disappearance_cannot_reuse_open_old_record(self):
        self.issue();original=self.gate._open;removed=[]
        def opening(name,flags,mode=0o600):
            fd=original(name,flags,mode)
            if name=='admission.json' and not removed:
                (self.root/name).unlink();removed.append(True)
            return fd
        with patch.object(self.gate,'_open',side_effect=opening):
            with self.assertRaises((FileNotFoundError,j.Conflict)):self.gate.status()
        self.assertFalse((self.root/'admission.json').exists())

    def test_snapshot_continual_publication_has_a_finite_bound(self):
        self.issue();writer=a.Gate(self.root);self.addCleanup(writer.close)
        row=writer.status()['record'];original=self.gate._open;opened=[]
        def opening(name,flags,mode=0o600):
            fd=original(name,flags,mode)
            if name=='admission.json':
                with writer.transaction():writer._save(row)
                opened.append(fd)
                self.assertLessEqual(len(opened),8)
            return fd
        with patch.object(self.gate,'_open',side_effect=opening):
            with self.assertRaises(j.Conflict):self.gate.status()
        self.assertTrue(opened);self.assertEqual(writer.status()['record'],row)


if __name__=='__main__':unittest.main()
