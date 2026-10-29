#!/usr/bin/env python3
"""Actual format2 guards, canonical reservation, fencing and old-reader refusal."""
import copy
import importlib.util
import json
import os
from pathlib import Path
import select
import subprocess
import sys
import unittest
import uuid
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / 'native/collections'))
import admission as legacy
import journal as journal
import runtime_authority as authority
import shared_admission as shared

module = importlib.util.spec_from_file_location('authority_fixture', Path(__file__).with_name('test-collection-runtime-authority.py'))
fixture = importlib.util.module_from_spec(module)
module.loader.exec_module(fixture)


def expected(permit, role):
    return {'permit': permit['id'], 'epoch': permit['epoch'], 'collection': journal.digest(permit['specification']),
            **{k: permit['specification']['members'][role][k] for k in ('plugin_id', 'profile', 'artifact_sha256')}}


class SharedAdmissionTests(unittest.TestCase):
    # Reuse only the setup/cleanup helpers, not the authority unit suite.
    setUp = fixture.AuthorityTests.setUp
    cleanup = fixture.AuthorityTests.cleanup
    provision = fixture.AuthorityTests.provision
    reserve = fixture.AuthorityTests.reserve
    receive = fixture.AuthorityTests.receive

    def launch(self, wrong_display=False):
        env = dict(os.environ, XDG_RUNTIME_DIR=str(self.runtime), WAYLAND_DISPLAY='wrong-display' if wrong_display else self.display.name)
        child = subprocess.Popen([sys.executable, '-B', __file__, '--shell'], env=env,
                                 stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        self.children.append(child)
        identity = self.receive(child)['identity']
        self.assertEqual(identity, legacy.identity(child.pid))
        return child, identity

    def send(self, shell, value):
        shell.stdin.write(journal.canonical(value) + b'\n')
        shell.stdin.flush()
        return self.receive(shell)

    def issued(self):
        self.provision()
        shell, identity = self.launch()
        self.permit['shell'] = identity
        reservation = self.reserve()
        gate = shared.Gate(self.gate)
        self.addCleanup(gate.close)
        gate.issue(self.permit, reservation)
        return shell, reservation, gate

    def start(self, shell):
        return self.send(shell, {'command': 'start', 'gate': str(self.gate), 'permit': self.permit})

    def test_three_actual_guards_share_one_reservation_and_end(self):
        shell, ticket, gate = self.issued()
        rows = self.start(shell)['guards']
        self.assertEqual(len(rows), 3)
        self.assertTrue(all(r['ok'] and r['admission']['format'] == 2 and r['admission']['authority'] == ticket for r in rows))
        self.assertEqual(set(gate.status()['record']['claims']), set(journal.ROLES))
        self.assertTrue(all(gate.status()['claim_owners_live'].values()))
        self.assertEqual([0, 0, 0], self.send(shell, {'command': 'end'})['exits'])
        self.assertTrue(all(v['state'] == 'ended' for v in gate.status()['record']['claims'].values()))

    def test_guard_death_and_free_lock_do_not_allow_replacement(self):
        shell, ticket, gate = self.issued()
        self.start(shell)
        self.assertEqual(-9, self.send(shell, {'command': 'kill-one'})['exit'])
        denied = self.send(shell, {'command': 'replace-one', 'gate': str(self.gate), 'permit': self.permit})
        self.assertFalse(denied['grant']['ok'])
        self.assertEqual(1, denied['exit'])
        self.assertEqual('admitted', gate.status()['record']['claims']['omabit']['state'])
        self.assertEqual(ticket, self.reserve())
        self.send(shell, {'command': 'end'})

    def test_shell_environment_cannot_select_a_different_runtime_authority(self):
        self.provision()
        shell, identity = self.launch(wrong_display=True)
        self.permit['shell'] = identity
        ticket = self.reserve()
        gate = shared.Gate(self.gate)
        try:
            with self.assertRaises(journal.Conflict):
                gate.issue(self.permit, ticket)
        finally:
            gate.close()
        self.assertFalse((self.gate / 'admission.json').exists())
        self.assertIsNotNone(self.authority.status()['reservation'])

    def test_copying_permit_to_another_gate_cannot_issue_or_claim(self):
        shell, ticket, gate = self.issued()
        other = self.root / 'unreserved-gate'
        other.mkdir(mode=0o700)
        for name in ('admission.json', 'issued.json'):
            fixture.save(other / name, json.loads((self.gate / name).read_text()))
        copied = shared.Gate(other)
        try:
            with self.assertRaises(journal.Conflict):
                copied.issue(self.permit, ticket)
        finally:
            copied.close()
        denied = self.send(shell, {'command': 'replace-one', 'gate': str(other), 'permit': self.permit})
        self.assertFalse(denied['grant']['ok'])
        self.assertEqual(1, denied['exit'])
        self.assertEqual({}, json.loads((other / 'admission.json').read_text())['claims'])
        self.assertTrue(all(v['ok'] for v in self.start(shell)['guards']))
        self.send(shell, {'command': 'end'})

    def test_authority_loss_fences_live_guards_but_allows_ending(self):
        shell, ticket, gate = self.issued()
        self.start(shell)
        (self.authority.root / 'ready.json').unlink()
        stopped = self.send(shell, {'command': 'await-refusal'})
        self.assertEqual([1, 1, 1], stopped['exits'])
        self.assertTrue(all(not row['ok'] for row in stopped['responses']))
        self.assertTrue(all(v['state'] == 'ended' for v in gate.status()['record']['claims'].values()))
        # Closing is irreversible local cleanup and remains available.
        self.assertEqual('closing', gate.stop_admission(self.permit['id'], uuid.uuid4().hex)['state'])
        self.assertIsNotNone(json.loads((self.authority.root / 'reservation.json').read_text())['reservation'])

    def test_irreversible_closing_preserves_authority_and_denies_new_claims(self):
        shell, ticket, gate = self.issued()
        request = uuid.uuid4().hex
        gate.stop_admission(self.permit['id'], request)
        before = (self.gate / 'admission.json').read_bytes()
        gate.issue(self.permit, ticket)
        self.assertEqual(before, (self.gate / 'admission.json').read_bytes())
        denied = self.send(shell, {'command': 'replace-one', 'gate': str(self.gate), 'permit': self.permit})
        self.assertFalse(denied['grant']['ok'])
        self.assertEqual('closing', gate.status()['record']['state'])

    def test_original_reader_refuses_format2_without_mutation(self):
        shell, ticket, gate = self.issued()
        before = {p.name: p.read_bytes() for p in self.gate.iterdir()}
        old = legacy.Gate(self.gate)
        try:
            with self.assertRaises(journal.Conflict):
                old.status()
            with self.assertRaises(journal.Conflict):
                old.issue(self.permit)
        finally:
            old.close()
        self.assertEqual(before, {p.name: p.read_bytes() for p in self.gate.iterdir()})

    def test_new_reader_refuses_legacy_without_migration_old_cleanup_works(self):
        old = legacy.Gate(self.gate)
        try:
            old.issue(self.permit)
            before = {p.name: p.read_bytes() for p in self.gate.iterdir()}
            new = shared.Gate(self.gate)
            try:
                with self.assertRaises(journal.Conflict):
                    new.status()
            finally:
                new.close()
            self.assertEqual(before, {p.name: p.read_bytes() for p in self.gate.iterdir()})
            self.assertEqual('closing', old.stop_admission(self.permit['id'], uuid.uuid4().hex)['state'])
        finally:
            old.close()

    def test_issue_interruption_before_seal_resumes_exact_reservation(self):
        self.provision()
        shell, identity = self.launch()
        self.permit['shell'] = identity
        ticket = self.reserve()
        gate = shared.Gate(self.gate)
        try:
            with patch.object(gate, '_seal', side_effect=OSError('before seal')):
                with self.assertRaises(OSError):
                    gate.issue(self.permit, ticket)
            denied = self.send(shell, {'command': 'replace-one', 'gate': str(self.gate), 'permit': self.permit})
            self.assertFalse(denied['grant']['ok'])
            before = (self.gate / 'admission.json').read_bytes()
            gate.issue(self.permit, ticket)
            self.assertEqual(before, (self.gate / 'admission.json').read_bytes())
            self.assertTrue(all(v['ok'] for v in self.start(shell)['guards']))
            self.send(shell, {'command': 'end'})
        finally:
            gate.close()

    def test_lost_shell_does_not_release_reservation_or_reissue(self):
        shell, ticket, gate = self.issued()
        shell.kill()
        self.assertEqual(-9, shell.wait(timeout=5))
        before = (self.authority.root / 'reservation.json').read_bytes()
        self.assertEqual(ticket, self.reserve())
        with self.assertRaises(journal.Conflict):
            gate.issue(self.permit, ticket)
        replacement, identity = self.launch()
        new_permit = copy.deepcopy(self.permit)
        new_permit['shell'] = identity
        with self.assertRaises(journal.Conflict):
            self.reserve(new_permit)
        self.assertEqual(before, (self.authority.root / 'reservation.json').read_bytes())

    def test_second_shell_and_gate_cannot_compete_with_live_three_guards(self):
        shell, ticket, gate = self.issued()
        self.assertTrue(all(v['ok'] for v in self.start(shell)['guards']))
        other, identity = self.launch()
        new_permit = copy.deepcopy(self.permit)
        new_permit['shell'] = identity
        new_permit['id'] = uuid.uuid4().hex
        another_gate = self.root / 'competitor-gate'
        another_gate.mkdir(mode=0o700)
        with self.assertRaises(journal.Conflict):
            self.reserve(new_permit, another_gate)
        competing = shared.Gate(another_gate)
        try:
            with self.assertRaises(journal.Conflict):
                competing.issue(new_permit, ticket)
        finally:
            competing.close()
        self.assertFalse((another_gate / 'admission.json').exists())
        self.assertTrue(all(gate.status()['claim_owners_live'].values()))
        self.assertEqual([0, 0, 0], self.send(shell, {'command': 'end'})['exits'])


    def test_shared_snapshot_reopens_after_atomic_publication(self):
        shell,ticket,gate=self.issued();writer=shared.Gate(self.gate);self.addCleanup(writer.close)
        original=gate._open;observed=[];request=uuid.uuid4().hex
        def opening(name,flags,mode=0o600):
            fd=original(name,flags,mode)
            if name=='admission.json' and not observed:
                writer.stop_admission(self.permit['id'],request)
                observed.append(os.fstat(fd).st_nlink)
            return fd
        with patch.object(gate,'_open',side_effect=opening):row=gate.status()['record']
        self.assertEqual(observed,[0]);self.assertEqual(row['state'],'closing')
        self.assertEqual(row['close_request'],request);self.assertEqual(row['authority'],ticket)


def shell_worker():
    children = []
    def emit(value):
        print(json.dumps(value), flush=True)
    def receive(child):
        if not select.select([child.stdout], [], [], 5)[0]:
            raise RuntimeError('guard response deadline')
        return json.loads(child.stdout.readline())
    def launch(request, role):
        child = subprocess.Popen([sys.executable, '-B', str(Path(shared.__file__).with_name('shared_admission_guard.py')),
                                  '--root', request['gate']], stdin=subprocess.PIPE,
                                 stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        children.append(child)
        child.stdin.write(journal.canonical({'expected': expected(request['permit'], role), 'role': role,
                                            'instance': uuid.uuid4().hex}) + b'\n')
        child.stdin.flush()
        return child
    emit({'identity': legacy.identity(os.getpid())})
    try:
        for line in sys.stdin:
            request = json.loads(line)
            command = request['command']
            if command == 'start':
                group = [launch(request, role) for role in journal.ROLES]
                emit({'guards': [receive(c) for c in group]})
            elif command == 'replace-one':
                child = launch(request, 'omabit')
                result = receive(child)
                if result['ok']:
                    child.stdin.close()
                emit({'grant': result, 'exit': child.wait(timeout=5)})
            elif command == 'kill-one':
                children[0].kill()
                emit({'exit': children[0].wait(timeout=5)})
            elif command == 'await-refusal':
                live = [c for c in children if c.returncode is None]
                responses = [receive(c) for c in live]
                emit({'responses': responses, 'exits': [c.wait(timeout=5) for c in live]})
            elif command == 'end':
                for child in children:
                    if child.poll() is None:
                        child.stdin.close()
                emit({'exits': [c.wait(timeout=5) for c in children]})
            else:
                raise ValueError('unknown fixture command')
    finally:
        for child in children:
            if child.poll() is None:
                child.kill()
                child.wait(timeout=5)
            for stream in (child.stdin, child.stdout, child.stderr):
                if stream and not stream.closed:
                    stream.close()


if __name__ == '__main__':
    if sys.argv[1:] == ['--shell']:
        shell_worker()
    else:
        unittest.main()
