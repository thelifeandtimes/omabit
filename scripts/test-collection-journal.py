#!/usr/bin/env python3
"""Durable collection sequencing tests. Only fresh temporary files are touched."""
import copy
import json
import os
from pathlib import Path
import signal
import subprocess
import sys
import tempfile
import unittest
import uuid
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / 'native/collections'))
import journal as j


def spec(name, generation=1):
    return {'format': 1, 'name': name, 'members': {
        role: {'plugin_id': 'fixture.'+role+'.'+name, 'profile': 'qualification-'+name,
               'artifact_sha256': j.digest([name, role, generation]),
               'paths': {kind: '/fixture/'+name+'/'+role+'/'+kind for kind in j.PATHS},
               'endpoint_ref': None if role == 'omabit' else j.digest([name, role, 'endpoint'])}
        for role in j.ROLES}}


def loaded(specification, epoch):
    return {'collection': j.digest(specification), 'epoch': epoch, 'members': {
        role: {**{k: member[k] for k in ('plugin_id','profile','artifact_sha256')}, 'instance': uuid.uuid4().hex}
        for role, member in specification['members'].items()}}


def ready(specification, observed):
    return {'loaded': observed, 'ready': True, 'endpoints': {r: specification['members'][r]['endpoint_ref'] for r in j.ROLES}}


class JournalTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix='omabit-collection-journal-')
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)/'journal'
        self.store = j.Store(self.root, create=True)
        self.addCleanup(lambda: self.store.close())
        self.stable, self.dev = spec('stable'), spec('dev')
        self.old = loaded(self.stable, uuid.uuid4().hex)
        self.write({'type':'register','spec':self.stable})
        self.write({'type':'register','spec':self.dev})
        self.write({'type':'adopt','observation':ready(self.stable,self.old)})
        self.request = uuid.uuid4().hex
        self.begin = {'type':'begin','request':self.request,'candidate':j.digest(self.dev),'expected':j.digest(self.old)}

    def write(self, event):
        with self.store.lease() as lease:
            return self.store.record(lease, event)

    def reopen(self):
        self.store.close(); self.store = j.Store(self.root)
        return self.store.status()

    def pending(self):
        state = self.store.status()
        return state['requests'][self.request]

    def envelope(self, kind='completed', **fields):
        p = self.pending()
        return dict(type=kind, request=self.request, phase=p['phase'], effect=j.effect(p,p['phase']), **fields)

    def next_events(self):
        state = self.store.status(); p = state['requests'][self.request]; phase = p['phase']
        token = j.effect(p,phase)
        if phase.startswith('drain-'):
            old = p['previous'] if phase == 'drain-previous' else p['loaded']
            epoch = p['previous']['epoch'] if phase == 'drain-previous' else j.effect(p,'enable-candidate')
            return [self.envelope('drained', role=r, observation={'epoch':epoch,'binding':old['members'][r] if old else None,
                'absent':old is None,'closed':True,'retained_state':j.digest([r,'retained-state']),
                'pending_ids':['a'*32] if r=='dms' else []}) for r in j.ROLES]
        if phase.startswith('disable-'):
            observation = {'epoch':p['previous']['epoch'] if phase=='disable-previous' else j.effect(p,'enable-candidate'), 'absent':True, 'helpers_exited':True}
        elif phase == 'checkpoint-appearance': observation = {'checkpoint':'b'*64}
        elif phase == 'restore-appearance': observation = {'checkpoint':p['checkpoint'],'verified':True}
        elif phase.startswith('enable-'):
            observation = loaded(state['collections'][j.target(p,phase)], token)
        else: observation = ready(state['collections'][j.target(p,phase)], p['loaded'])
        return [self.envelope(observation=observation)]

    def finish(self):
        events = []
        while self.pending()['phase']:
            for event in self.next_events():
                self.write(event); self.reopen(); events.append(event)
        return events

    def until(self, phase):
        while self.pending()['phase'] != phase:
            for event in self.next_events(): self.write(event); self.reopen()

    def test_status_does_not_create_missing_store_and_creation_never_resets(self):
        absent = Path(self.temp.name)/'absent'
        with self.assertRaises(FileNotFoundError): j.Store(absent)
        self.assertFalse(absent.exists())
        before = (self.root/'journal.json').read_bytes()
        with self.assertRaises(FileExistsError): j.Store(self.root, create=True)
        self.assertEqual((self.root/'journal.json').read_bytes(), before)

    def test_initial_adoption_lost_ack_is_exact_read_only_replay(self):
        before = (self.root/'journal.json').read_bytes()
        self.write({'type':'adopt','observation':ready(self.stable,self.old)})
        self.assertEqual((self.root/'journal.json').read_bytes(), before)
        with self.assertRaises(j.Conflict):
            self.write({'type':'adopt','observation':ready(self.stable,loaded(self.stable,uuid.uuid4().hex))})

    def test_commit_reopens_each_boundary_and_replays_without_writes(self):
        self.write(self.begin); events = self.finish()
        state = self.store.status()
        self.assertEqual(state['committed']['collection'], j.digest(self.dev))
        self.assertEqual(self.pending()['outcome'],'committed')
        self.assertIsNone(state['pending'])
        prior = (self.root/'journal.json').read_bytes()
        self.write(self.begin)
        for event in events: self.write(event)
        self.assertEqual((self.root/'journal.json').read_bytes(),prior)
        self.assertIn('a'*32,prior.decode())

    def test_rollback_reopens_each_boundary_and_restores_exact_previous_spec(self):
        self.write(self.begin); self.until('verify-candidate')
        stale_ready = self.next_events()[0]
        self.write(self.envelope('problem',reason='readiness-failed'))
        self.write(self.envelope('rollback',reason='readiness-failed'))
        self.finish(); state = self.store.status()
        self.assertEqual(self.pending()['outcome'],'rolled-back')
        self.assertEqual(state['committed']['collection'],j.digest(self.stable))
        self.assertNotEqual(state['committed']['epoch'],self.old['epoch'])
        with self.assertRaises(j.Conflict): self.write(stale_ready)
        self.assertEqual(self.pending()['checkpoint'],'b'*64)

    def test_failed_enable_can_rollback_with_observed_absence(self):
        self.write(self.begin); self.until('enable-candidate')
        self.write(self.envelope('rollback',reason='loader-unavailable'))
        self.finish(); self.assertEqual(self.pending()['outcome'],'rolled-back')

    def test_partial_candidate_can_drain_present_and_absent_roles(self):
        self.write(self.begin); self.until('enable-candidate')
        self.write(self.envelope('rollback',reason='loader-unavailable'))
        events = self.next_events(); observed = loaded(self.dev,j.effect(self.pending(),'enable-candidate'))
        events[0]['observation'].update(absent=False,binding=observed['members']['omabit'])
        for event in events: self.write(event)
        self.finish(); self.assertEqual(self.pending()['outcome'],'rolled-back')

    def test_same_profile_new_artifact_preserves_namespace_and_supports_return(self):
        updated = spec('stable',2); self.write({'type':'register','spec':updated})
        self.begin['candidate'] = j.digest(updated); self.write(self.begin); self.finish()
        current = self.store.status()['committed']; self.request = uuid.uuid4().hex
        self.write({'type':'begin','request':self.request,'candidate':j.digest(self.stable),'expected':j.digest(current)})
        self.finish(); self.assertEqual(self.store.status()['committed']['collection'],j.digest(self.stable))

    def test_request_conflicts_and_stale_expected_collection_refuse(self):
        bad = dict(self.begin,expected='0'*64)
        with self.assertRaises(j.Conflict): self.write(bad)
        self.write(self.begin)
        with self.assertRaises(j.Conflict): self.write(dict(self.begin,candidate=j.digest(self.stable)))
        with self.assertRaises(j.Conflict): self.write(dict(self.begin,request=uuid.uuid4().hex))

    def test_wrong_epoch_instance_profile_artifact_and_role_do_not_advance(self):
        self.write(self.begin); event = self.next_events()[0]; before = (self.root/'journal.json').read_bytes()
        variants=[]
        for key in ('epoch','binding'):
            row=copy.deepcopy(event)
            if key=='epoch':row['observation'][key]='0'*32
            else:row['observation'][key]['instance']='0'*32
            variants.append(row)
        for key,value in [('profile','other'),('artifact_sha256','0'*64),('plugin_id','other.plugin')]:
            row=copy.deepcopy(event);row['observation']['binding'][key]=value;variants.append(row)
        row=copy.deepcopy(event);row['role']='other';variants.append(row)
        row=copy.deepcopy(event);row['effect']='0'*32;variants.append(row)
        for row in variants:
            with self.assertRaises(j.Conflict):self.write(row)
        self.assertEqual((self.root/'journal.json').read_bytes(),before)

    def test_missing_previous_component_is_not_closed(self):
        self.write(self.begin); event=self.next_events()[0]
        event['observation'].update(absent=True,binding=None)
        with self.assertRaises(j.Conflict):self.write(event)

    def test_partial_drain_is_durable_and_never_unloads_early(self):
        self.write(self.begin); events=self.next_events()
        self.write(events[0]);self.reopen()
        self.assertEqual(self.pending()['phase'],'drain-previous')
        with self.assertRaises(j.Conflict):self.write(self.envelope(observation={'absent':True,'helpers_exited':True,'epoch':self.old['epoch']}))
        self.write(events[1]);self.write(events[2]);self.assertEqual(self.pending()['phase'],'disable-previous')

    def test_unclean_helper_lineage_survives_restart_and_cannot_be_reset_by_receipt(self):
        self.write(self.begin); self.write(self.envelope('problem',reason='loader-unavailable'))
        self.write(self.envelope('problem',reason='helper-descendants-unconfirmed'))
        self.reopen(); before=(self.root/'journal.json').read_bytes()
        with self.assertRaises(j.Conflict):self.write(self.next_events()[0])
        with self.assertRaises(j.Conflict):self.write({'type':'adopt','observation':ready(self.stable,self.old)})
        with self.assertRaises(j.Conflict):self.write(self.envelope('rollback',reason='loader-unavailable'))
        self.assertEqual((self.root/'journal.json').read_bytes(),before)
        self.assertEqual(self.pending()['uncertain'][0]['reason'],'helper-descendants-unconfirmed')

    def test_restore_requires_exact_checkpoint_and_verified_readback(self):
        self.write(self.begin);self.until('verify-candidate');self.write(self.envelope('rollback',reason='readiness-failed'));self.until('restore-appearance')
        for value in ({'checkpoint':'c'*64,'verified':True},{'checkpoint':'b'*64,'verified':False}):
            with self.assertRaises(j.Conflict):self.write(self.envelope(observation=value))
        self.finish()

    def test_wrong_ready_endpoint_cannot_commit(self):
        self.write(self.begin);self.until('verify-candidate');event=self.next_events()[0]
        event['observation']['endpoints']['dms']=self.stable['members']['dms']['endpoint_ref']
        with self.assertRaises(j.Conflict):self.write(event)
        self.assertEqual(self.store.status()['committed'],self.old)

    def test_namespace_aliases_and_nested_roots_are_refused(self):
        bad=spec('edge');bad['members']['dms']['paths']['state']='/fixture/stable/theme/state/inside'
        with self.assertRaises(j.Conflict):self.write({'type':'register','spec':bad})
        bad=spec('edge');bad['members']['dms']['plugin_id']=self.stable['members']['dms']['plugin_id']
        with self.assertRaises(j.Conflict):self.write({'type':'register','spec':bad})
        bad=spec('edge');bad['members']['dms']['paths']['state']=bad['members']['theme']['paths']['state']
        with self.assertRaises(j.Conflict):self.write({'type':'register','spec':bad})

    def test_unknown_and_sensitive_fields_are_not_persisted(self):
        before=(self.root/'journal.json').read_bytes()
        bad=spec('edge');bad['members']['dms']['cookie']='must-not-persist'
        with self.assertRaises(j.Conflict):self.write({'type':'register','spec':bad})
        self.assertEqual((self.root/'journal.json').read_bytes(),before)

    def test_unknown_format_duplicate_keys_and_illegal_history_preserved(self):
        path=self.root/'journal.json'; original=path.read_bytes()
        cases=[b'{"format":99,"events":[]}', b'{"format":1,"format":1,"events":[]}',b'{"format":true,"events":[]}']
        data=json.loads(original);data['events'].append(data['events'][0]);cases.append(j.canonical(data))
        for data in cases:
            path.write_bytes(data)
            with self.assertRaises((j.Conflict,ValueError)):self.store.status()
            self.assertEqual(path.read_bytes(),data)
        path.write_bytes(original)

    def test_missing_journal_never_reinitializes_existing_store(self):
        path=self.root/'journal.json';path.rename(self.root/'retained.json')
        with self.assertRaises(j.Conflict):self.store.status()
        self.store.close();self.store=j.Store(self.root)
        with self.assertRaises(j.Conflict):self.store.status()

    def test_unsafe_file_and_lock_forms_refuse_intact(self):
        path=self.root/'journal.json';original=path.read_bytes();os.chmod(path,0o644)
        with self.assertRaises(j.Conflict):self.store.status()
        os.chmod(path,0o600);other=self.root/'extra';os.link(path,other)
        with self.assertRaises(j.Conflict):self.store.status()
        other.unlink();path.rename(other);path.symlink_to(other)
        with self.assertRaises(OSError):self.store.status()
        path.unlink();other.rename(path);self.assertEqual(path.read_bytes(),original)

    def test_simultaneous_coordinators_and_closed_lease_refuse(self):
        second=j.Store(self.root)
        try:
            with self.store.lease() as lease:
                with self.assertRaises(BlockingIOError):
                    with second.lease():pass
            with self.assertRaises(j.Conflict):self.store.record(lease,self.begin)
        finally:second.close()

    def test_interrupted_write_before_replace_preserves_old_journal_and_temporary(self):
        before=(self.root/'journal.json').read_bytes()
        with patch.object(j.os,'replace',side_effect=OSError('injected rename failure')):
            with self.assertRaises(OSError):self.write(self.begin)
        self.assertEqual((self.root/'journal.json').read_bytes(),before)
        self.assertEqual(len(list(self.root.glob('.pending-*'))),1)
        self.reopen();self.write(self.begin);self.assertEqual(self.pending()['phase'],'drain-previous')

    def test_lost_ack_after_replace_reopens_and_replays_same_intent(self):
        real=j.os.fsync
        def sync(fd):
            if fd==self.store.fd:raise OSError('injected directory sync failure')
            return real(fd)
        with patch.object(j.os,'fsync',side_effect=sync):
            with self.assertRaises(OSError):self.write(self.begin)
        self.reopen();before=(self.root/'journal.json').read_bytes();self.write(self.begin)
        self.assertEqual((self.root/'journal.json').read_bytes(),before)
        self.assertEqual(self.pending()['phase'],'drain-previous')

    def test_history_limit_refuses_without_discarding_receipts(self):
        before=(self.root/'journal.json').read_bytes()
        with patch.object(j,'MAX_EVENTS',3):
            with self.assertRaises(j.Conflict):self.write(self.begin)
        self.assertEqual((self.root/'journal.json').read_bytes(),before)

    def test_actual_coordinator_kill_releases_lease_and_preserves_transition(self):
        script="""import json,sys;from pathlib import Path;sys.path.insert(0,sys.argv[1]);import journal
store=journal.Store(sys.argv[2])
with store.lease() as lease:
 store.record(lease,json.loads(sys.argv[3]));print('persisted',flush=True);sys.stdin.read()
"""
        proc=subprocess.Popen([sys.executable,'-B','-c',script,str(ROOT/'native/collections'),str(self.root),json.dumps(self.begin)],stdin=subprocess.PIPE,stdout=subprocess.PIPE,stderr=subprocess.PIPE,text=True)
        try:
            import select
            self.assertTrue(select.select([proc.stdout],[],[],5)[0]);self.assertEqual(proc.stdout.readline().strip(),'persisted')
            with self.assertRaises(BlockingIOError):
                with self.store.lease():pass
            proc.kill();self.assertEqual(proc.wait(timeout=5),-signal.SIGKILL)
            self.reopen();self.assertEqual(self.pending()['phase'],'drain-previous');self.finish()
            self.assertEqual(self.pending()['outcome'],'committed')
        finally:
            if proc.poll() is None:proc.kill();proc.wait(timeout=5)
            for stream in (proc.stdin,proc.stdout,proc.stderr):stream.close()


if __name__=='__main__':unittest.main(verbosity=2)
