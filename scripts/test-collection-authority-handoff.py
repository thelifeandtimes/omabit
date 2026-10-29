#!/usr/bin/env python3
"""Handoff journal model with actual private guards and an owned subreaper.

Application drain/appearance observations are synthetic here. This verifies
admission/evidence sequencing, not real application or desktop cleanup.
"""
import copy
import ctypes
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import select
import signal
import subprocess
import sys
import time
import unittest
import uuid
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / 'native/collections'))
import admission as old
import journal as j
import runtime_authority as authority
import shared_admission as shared


def module(name, file):
    spec = importlib.util.spec_from_file_location(name, file)
    value = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(value)
    return value


fixture = module('handoff_authority_fixture', Path(__file__).with_name('test-collection-runtime-authority.py'))
model = module('handoff_journal_fixture', Path(__file__).with_name('test-collection-journal.py'))


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def save(path, value):
    temporary = path.with_name('.pending-' + uuid.uuid4().hex)
    with temporary.open('xb') as stream:
        os.fchmod(stream.fileno(), 0o600)
        stream.write(j.canonical(value) + b'\n')
        stream.flush()
        os.fsync(stream.fileno())
    temporary.replace(path)
    fd = os.open(path.parent, os.O_RDONLY | os.O_DIRECTORY)
    try:
        os.fsync(fd)
    finally:
        os.close(fd)


class HandoffTests(unittest.TestCase):
    provision = fixture.AuthorityTests.provision
    receive = fixture.AuthorityTests.receive

    def setUp(self):
        fixture.AuthorityTests.setUp(self)
        self.gates = []
        self.guardians = []
        self.provision()
        self.dev = copy.deepcopy(self.spec)
        self.dev['name'] = 'dev'
        for role, member in self.dev['members'].items():
            member['plugin_id'] = member['plugin_id'].replace('qualification-stable', 'qualification-dev')
            member['profile'] = 'qualification-dev'
            member['artifact_sha256'] = j.digest(['dev', role])
            member['paths'] = {k: v.replace('/isolated/', '/isolated-dev/') for k, v in member['paths'].items()}
        self.record({'type': 'register', 'spec': self.dev})
        self.guardian, self.owner, self.shell, self.receipt = self.launch_guardian()
        self.permit['shell'] = self.shell
        with self.store.lease() as lease:
            self.ticket = self.authority.reserve(self.store, lease, self.gate, self.permit, guardian_receipt=self.receipt)
        self.current = shared.Gate(self.gate)
        self.gates.append(self.current)
        self.current.issue(self.permit, self.ticket)
        self.start_claims()
        self.loaded = model.loaded(self.spec, self.permit['epoch'])
        self.record({'type': 'adopt', 'observation': model.ready(self.spec, self.loaded)})
        self.binding_path = self.write_binding()

    def cleanup(self):
        for child, _, _, _ in getattr(self, 'guardians', []):
            if child.poll() is None:
                try:
                    self.send(child, {'command': 'stop', 'request': uuid.uuid4().hex})
                    child.wait(timeout=5)
                except (OSError, ValueError, AssertionError, subprocess.TimeoutExpired):
                    pass
        for gate in getattr(self, 'gates', []):
            gate.close()
        fixture.AuthorityTests.cleanup(self)

    def send(self, child, value):
        child.stdin.write(j.canonical(value) + b'\n')
        child.stdin.flush()
        return self.receive(child)

    def launch_guardian(self):
        token = uuid.uuid4().hex
        receipt = self.root / ('guardian-' + token + '.json')
        env = dict(os.environ, WAYLAND_DISPLAY=self.display.name, XDG_RUNTIME_DIR=str(self.runtime))
        child = subprocess.Popen([sys.executable, '-B', __file__, '--guardian', token, str(receipt)], env=env,
                                 stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        self.children.append(child)
        value = self.receive(child)
        self.assertEqual(child.pid, value['owner']['pid'])
        item = (child, value['owner'], value['shell'], receipt)
        self.guardians.append(item)
        return item

    def start_claims(self):
        rows = self.send(self.guardian, {'command': 'start', 'permit': self.permit, 'gate': str(self.gate)})['guards']
        self.assertEqual(3, len(rows))
        self.assertTrue(all(row['ok'] for row in rows), rows)

    def record(self, event):
        with self.store.lease() as lease:
            return self.store.record(lease, event)

    def write_binding(self):
        path = self.root / ('loaded-' + self.permit['epoch'] + '.json')
        row = self.current.status()['record']
        save(path, {'loaded': self.loaded, 'helpers': {role: [claim['owner']] for role, claim in row['claims'].items()},
                    'shell': self.shell, 'supervisor': self.owner,
                    'admission': {'root': str(self.gate), 'permit': self.permit}})
        return path

    def bind(self):
        with self.store.lease() as lease:
            return self.authority.bind_loaded(self.store, lease, self.ticket, self.binding_path, self.receipt)

    def begin(self, specification=None):
        self.request = uuid.uuid4().hex
        self.record({'type': 'begin', 'request': self.request, 'candidate': j.digest(specification or self.dev),
                     'expected': j.digest(self.store.status()['committed'])})

    def pending(self):
        return self.store.status()['requests'][self.request]

    def envelope(self, kind='completed', **fields):
        request = self.pending()
        return dict(type=kind, request=self.request, phase=request['phase'], effect=j.effect(request, request['phase']), **fields)

    def close_claims(self, request=None):
        self.current.stop_admission(self.permit['id'], request or j.effect(self.pending(), self.pending()['phase']))
        exits = self.send(self.guardian, {'command': 'end'})['exits']
        self.assertTrue(all(code == 0 for code in exits))

    def drain_events(self):
        request = self.pending()
        outgoing = request['previous'] if request['phase'] == 'drain-previous' else request['loaded']
        for role in j.ROLES:
            self.record(self.envelope('drained', role=role, observation={
                'epoch': outgoing['epoch'], 'binding': outgoing['members'][role], 'absent': False,
                'closed': True, 'retained_state': j.digest([role, 'retained']), 'pending_ids': ['a' * 32] if role == 'dms' else []}))

    def unload_and_appearance(self):
        self.record(self.envelope(observation={'epoch': self.permit['epoch'], 'absent': True, 'helpers_exited': True}))
        observation = {'checkpoint': 'b' * 64}
        if self.pending()['phase'] == 'restore-appearance':
            observation['verified'] = True
        self.record(self.envelope(observation=observation))

    def candidate(self, specification=None, shell=None):
        request = self.pending()
        phase = 'enable-previous' if request['direction'] == 'rollback' else 'enable-candidate'
        spec = specification or self.store.status()['collections'][j.target(request, phase)]
        path = self.root / ('gate-' + uuid.uuid4().hex)
        path.mkdir(mode=0o700)
        return path, {'id': uuid.uuid4().hex, 'request': request['id'], 'epoch': j.effect(request, phase),
                      'specification': spec, 'shell': shell or self.shell}

    def handoff(self, candidate):
        with self.store.lease() as lease:
            return self.authority.handoff(self.store, lease, *candidate, guardian_receipt=getattr(self, 'next_receipt', None))

    def ready_handoff(self):
        self.bind()
        self.begin()
        self.close_claims()
        self.drain_events()
        self.unload_and_appearance()
        return self.candidate()

    def activate(self, candidate, ticket, *, commit=True):
        self.gate, self.permit = candidate
        self.shell = self.permit['shell']
        self.ticket = ticket
        self.current = shared.Gate(self.gate)
        self.gates.append(self.current)
        self.current.issue(self.permit, ticket)
        self.start_claims()
        self.loaded = model.loaded(self.permit['specification'], self.permit['epoch'])
        self.record(self.envelope(observation=self.loaded))
        self.binding_path = self.write_binding()
        self.bind()
        if commit:
            self.record(self.envelope(observation=model.ready(self.permit['specification'], self.loaded)))

    def bytes(self):
        return (self.authority.root / 'reservation.json').read_bytes()

    def test_clean_same_shell_handoff_and_return_preserve_history(self):
        first = self.permit
        next = self.ready_handoff()
        ticket = self.handoff(next)
        self.activate(next, ticket)
        self.begin(self.spec)
        self.close_claims()
        self.drain_events()
        self.unload_and_appearance()
        back = self.candidate()
        self.activate(back, self.handoff(back))
        state = self.authority.status()
        self.assertEqual(2, len(state['history']))
        self.assertEqual(first, state['history'][0]['reservation']['permit'])
        self.assertEqual(self.spec, state['reservation']['permit']['specification'])
        self.assertIn('a' * 32, (self.store.root / 'journal.json').read_text())

    def test_full_candidate_rollback_has_new_permit_and_old_profile(self):
        candidate = self.ready_handoff()
        self.activate(candidate, self.handoff(candidate), commit=False)
        self.record(self.envelope('rollback', reason='readiness-failed'))
        self.close_claims()
        self.drain_events()
        self.unload_and_appearance()
        back = self.candidate()
        self.activate(back, self.handoff(back))
        self.assertEqual('rolled-back', self.pending()['outcome'])
        self.assertEqual(self.spec, self.authority.status()['reservation']['permit']['specification'])

    def test_bind_replay_is_byte_identical_and_late_rebinding_refuses(self):
        self.bind()
        before = self.bytes()
        self.bind()
        self.assertEqual(before, self.bytes())
        other = self.root / 'new-binding.json'
        save(other, json.loads(self.binding_path.read_text()))
        with self.store.lease() as lease, self.assertRaises(j.Conflict):
            self.authority.bind_loaded(self.store, lease, self.ticket, other, self.receipt)
        self.assertEqual(before, self.bytes())

    def test_unbound_old_activation_cannot_handoff(self):
        self.begin()
        self.close_claims()
        self.drain_events()
        self.unload_and_appearance()
        before = self.bytes()
        with self.assertRaises(j.Conflict):
            self.handoff(self.candidate())
        self.assertEqual(before, self.bytes())

    def test_unbound_activation_cannot_be_bound_after_guard_loss(self):
        self.send(self.guardian, {'command': 'kill-one'})
        with self.assertRaises(j.Conflict):
            self.bind()
        self.assertIsNone(self.authority.status()['binding'])

    def test_caller_success_receipts_do_not_override_live_guards(self):
        self.bind()
        self.begin()
        self.current.stop_admission(self.permit['id'], j.effect(self.pending(), 'drain-previous'))
        self.drain_events()
        self.unload_and_appearance()
        with self.assertRaises(j.Conflict):
            self.handoff(self.candidate())

    def test_handoff_before_cleanup_or_appearance_receipt_refuses(self):
        self.bind()
        self.begin()
        with self.assertRaises(j.Conflict):
            self.handoff(self.candidate())
        self.close_claims()
        self.drain_events()
        with self.assertRaises(j.Conflict):
            self.handoff(self.candidate())
        self.record(self.envelope(observation={'epoch': self.permit['epoch'], 'absent': True, 'helpers_exited': True}))
        with self.assertRaises(j.Conflict):
            self.handoff(self.candidate())

    def test_wrong_close_request_preserves_reservation(self):
        self.bind()
        self.begin()
        self.close_claims(uuid.uuid4().hex)
        self.drain_events()
        self.unload_and_appearance()
        before = self.bytes()
        with self.assertRaises(j.Conflict):
            self.handoff(self.candidate())
        self.assertEqual(before, self.bytes())

    def test_changed_original_binding_refuses(self):
        candidate = self.ready_handoff()
        body = json.loads(self.binding_path.read_text())
        body['loaded']['members']['theme']['instance'] = uuid.uuid4().hex
        save(self.binding_path, body)
        with self.assertRaises(j.Conflict):
            self.handoff(candidate)

    def test_candidate_must_match_canonical_effect_spec_and_display(self):
        candidate = self.ready_handoff()
        for field in ('request', 'epoch', 'specification'):
            changed = copy.deepcopy(candidate[1])
            changed[field] = self.spec if field == 'specification' else uuid.uuid4().hex
            with self.assertRaises(j.Conflict):
                self.handoff((candidate[0], changed))
        self.assertEqual(0, len(self.authority.status()['history']))

    def test_lost_durable_handoff_reply_and_reopen_are_exact(self):
        candidate = self.ready_handoff()
        original = self.authority._save_reservation
        def lost(value):
            original(value)
            raise OSError('lost handoff publication reply')
        with patch.object(self.authority, '_save_reservation', side_effect=lost), self.assertRaises(OSError):
            self.handoff(candidate)
        before = self.bytes()
        self.authority.close()
        self.authority = authority.Authority(self.runtime / authority.ANCHOR)
        self.handoff(candidate)
        self.assertEqual(before, self.bytes())
        self.assertEqual(1, len(self.authority.status()['history']))

    def test_published_handoff_rejects_replaced_target_gate(self):
        candidate = self.ready_handoff()
        self.handoff(candidate)
        candidate[0].rename(candidate[0].with_name('retained-target'))
        candidate[0].mkdir(mode=0o700)
        with self.assertRaises(j.Conflict):
            self.handoff(candidate)

    def test_journal_prefix_change_cannot_replay_handoff(self):
        candidate = self.ready_handoff()
        self.handoff(candidate)
        path = self.store.root / 'journal.json'
        row = json.loads(path.read_text())
        extra = copy.deepcopy(self.spec)
        extra['name'] = 'edge'
        for member in extra['members'].values():
            member['plugin_id'] += '.edge'
            member['profile'] = 'qualification-edge'
            member['paths'] = {k: v.replace('/isolated/', '/isolated-edge/') for k, v in member['paths'].items()}
        row['events'].insert(0, {'type': 'register', 'spec': extra})
        save(path, row)
        self.store.status()  # Still a valid journal; the previously bound prefix differs.
        with self.assertRaises(j.Conflict):
            self.handoff(candidate)

    def recovery_ready(self):
        self.bind()
        self.begin()
        self.current.stop_admission(self.permit['id'], j.effect(self.pending(), 'drain-previous'))
        self.assertEqual(-9, self.send(self.guardian, {'command': 'kill-one'})['exit'])
        self.send(self.guardian, {'command': 'kill-shell'})
        self.record(self.envelope('problem', reason='reader-owner-lost'))
        closure_request = uuid.uuid4().hex
        result = self.send(self.guardian, {'command': 'stop', 'request': closure_request})
        self.assertTrue(result['closed'])
        self.assertEqual(0, self.guardian.wait(timeout=5))
        binding = self.authority.status()['binding']
        request = self.pending()
        plan = {'loaded': request['previous'], 'fences': request['uncertain'], 'guardian': binding['guardian'],
                'closure_request': closure_request, 'runtime_binding_sha256': sha(self.binding_path),
                'members': {role: {'binding': self.loaded['members'][role], 'application_sha256': j.digest([role, 'app']),
                          'pending_ids': ['a' * 32] if role == 'dms' else [], 'credential_refs': [],
                          'transport_sha256': j.digest([role, 'before']), 'channel_ids': []} for role in j.ROLES}}
        recovered = {'plan_sha256': j.digest(plan), 'guardian': plan['guardian'], 'closure_request': closure_request,
                     'closure_receipt_sha256': sha(self.receipt), 'proof': 'waitpid-echild',
                     'members': {role: dict(value, closed=True, transport_sha256=j.digest([role, 'closed']))
                                 for role, value in plan['members'].items()}}
        rid = uuid.uuid4().hex
        self.record(self.envelope('recovery-prepared', recovery=rid, plan=plan))
        self.record(self.envelope('recovery-verified', recovery=rid, observation=recovered))
        self.unload_and_appearance()
        self.guardian, self.owner, self.shell, next_receipt = self.launch_guardian()
        self.next_receipt = next_receipt
        candidate = self.candidate(shell=self.shell)
        return candidate, next_receipt

    def test_original_subreaper_recovery_allows_new_shell_handoff(self):
        candidate, next_receipt = self.recovery_ready()
        ticket = self.handoff(candidate)
        proof = self.authority.status()['history'][0]['handoff']
        self.assertEqual(sha(self.receipt), proof['closure']['sha256'])
        self.receipt = next_receipt
        self.activate(candidate, ticket)

    def test_replacement_guardian_cannot_satisfy_original_recovery(self):
        candidate, next_receipt = self.recovery_ready()
        wrong = json.loads(next_receipt.read_text())
        wrong['phase'], wrong['proof'] = 'closed', 'waitpid-echild'
        save(self.receipt, wrong)
        before = self.bytes()
        with self.assertRaises(j.Conflict):
            self.handoff(candidate)
        self.assertEqual(before, self.bytes())

    def test_changed_recovery_receipt_digest_refuses(self):
        candidate, _ = self.recovery_ready()
        row = json.loads(self.receipt.read_text())
        row['request'] = uuid.uuid4().hex
        save(self.receipt, row)
        with self.assertRaises(j.Conflict):
            self.handoff(candidate)

    def test_original_reader_refuses_history_and_legacy_reserved_state_is_unchanged(self):
        f = ROOT / 'tests/fixtures/runtime-authority-v1'
        meta = json.loads((f / 'provenance.json').read_text())
        for name, digest in meta['sha256'].items():
            self.assertEqual(digest, sha(f / name))
        original = module('original_authority_writer', f / 'writer.py')
        before = self.bytes()
        with self.assertRaises(j.Conflict):
            original.Authority(self.authority.root)
        self.assertEqual(before, self.bytes())
        # A separate old writer creates a genuine format1 reservation. New code
        # may read it but cannot add a live-binding/handoff contract retroactively.
        other_runtime = self.root / 'old-runtime'
        other_runtime.mkdir(mode=0o700)
        import socket
        with socket.socket(socket.AF_UNIX, socket.SOCK_STREAM) as server:
            display = other_runtime / 'wayland-old'
            server.bind(str(display));server.listen(8)
            store = j.Store(self.root / 'old-journal', create=True)
            prior = current = None
            try:
                with store.lease() as lease:
                    prior = original.Authority.provision(display, store, lease)
                    store.record(lease, {'type': 'register', 'spec': self.spec})
                    gate = self.root / 'old-gate';gate.mkdir(mode=0o700)
                    permit = dict(self.permit, shell=old.identity(os.getpid()))
                    ticket = prior.reserve(store, lease, gate, permit)
                data = (prior.root / 'reservation.json').read_bytes()
                current = authority.Authority(prior.root)
                self.assertEqual(1, current.status()['format'])
                with store.lease() as lease, self.assertRaises(j.Conflict):
                    current.bind_loaded(store, lease, ticket, self.binding_path, self.receipt)
                self.assertEqual(data, (prior.root / 'reservation.json').read_bytes())
            finally:
                if prior:prior.close()
                if current:current.close()
                store.close()

    def recovered_partial_candidate(self, *, changed_reservation=False, changed_guardian=False):
        recovery_module = module('handoff_candidate_recovery_fixture', Path(__file__).with_name('test-collection-candidate-recovery.py'))
        next = self.ready_handoff()
        ticket = self.handoff(next)
        self.gate, self.permit, self.ticket = next[0], next[1], ticket
        self.current = shared.Gate(self.gate);self.gates.append(self.current)
        self.current.issue(self.permit, ticket);self.start_claims()
        self.record(self.envelope('problem', reason='reader-owner-lost'))
        self.record(self.envelope('rollback-fenced', reason='loader-unavailable'))
        closure_request = uuid.uuid4().hex
        self.assertTrue(self.send(self.guardian, {'command':'stop','request':closure_request})['closed'])
        self.assertEqual(0,self.guardian.wait(5))
        original = self.authority.status()['reservation']
        plan = recovery_module.plan_for(self.pending())
        plan.update(guardian=copy.deepcopy(original['owner']['guardian']),closure_request=closure_request,
                    authority=self.authority.descriptor['id'],reservation_sha256=j.digest(original))
        if changed_reservation:plan['reservation_sha256']='f'*64
        if changed_guardian:plan['guardian']['owner']['token']='f'*32
        recovery = uuid.uuid4().hex
        self.record(self.envelope('candidate-recovery-prepared',recovery=recovery,plan=plan))
        observed = recovery_module.observed(plan);observed['closure_receipt_sha256']=sha(self.receipt)
        self.record(self.envelope('candidate-recovery-verified',recovery=recovery,observation=observed))
        self.unload_and_appearance()
        self.guardian,self.owner,self.shell,self.next_receipt=self.launch_guardian()
        return self.candidate(shell=self.shell)

    def test_candidate_recovery_handoff_requires_exact_original_reservation(self):
        candidate=self.recovered_partial_candidate(changed_reservation=True);before=self.bytes()
        with self.assertRaises(j.Conflict):self.handoff(candidate)
        self.assertEqual(self.bytes(),before)

    def test_candidate_recovery_handoff_refuses_replacement_guardian(self):
        candidate=self.recovered_partial_candidate(changed_guardian=True);before=self.bytes()
        with self.assertRaises(j.Conflict):self.handoff(candidate)
        self.assertEqual(self.bytes(),before)

    def test_candidate_recovery_handoff_preserves_original_closure(self):
        candidate=self.recovered_partial_candidate();ticket=self.handoff(candidate)
        proof=self.authority.status()['history'][-1]['handoff']
        self.assertEqual('partial-rollback',proof['mode']);self.assertEqual(sha(self.receipt),proof['closure']['sha256'])
        self.receipt=self.next_receipt;self.activate(candidate,ticket)
        self.assertEqual('rolled-back',self.pending()['outcome'])

    def test_guardian_is_pinned_before_loaded_binding(self):
        state = self.authority.status()
        self.assertEqual(2, state['format'])
        self.assertIsNone(state['binding'])
        self.assertEqual(self.owner, state['reservation']['owner']['guardian']['owner'])
        self.assertEqual(str(self.receipt), state['reservation']['owner']['receipt'])

    def test_guardian_admission_loss_fences_real_guards(self):
        receipt = json.loads(self.receipt.read_text())
        receipt['phase'] = 'draining'
        save(self.receipt, receipt)
        result = self.send(self.guardian, {'command': 'await-refusal'})
        self.assertEqual([1, 1, 1], result['exits'])
        self.assertTrue(all(not row['ok'] for row in result['responses']))
        self.assertTrue(all(row['state'] == 'ended' for row in self.current.status()['record']['claims'].values()))

    def partial_rollback(self, *, issue=True, claims=0, missing=None):
        next = self.ready_handoff()
        ticket = self.handoff(next)
        self.gate, self.permit = next
        self.ticket = ticket
        self.current = shared.Gate(self.gate)
        self.gates.append(self.current)
        if issue:
            self.current.issue(self.permit, ticket)
        if claims == 1:
            response = self.send(self.guardian, {'command': 'replace-one', 'permit': self.permit, 'gate': str(self.gate)})
            self.assertTrue(response['grant']['ok'])
            self.assertEqual(0, response['exit'])
        elif claims == 3:
            self.start_claims()
        partial = model.loaded(self.dev, self.permit['epoch'])
        self.record(self.envelope('rollback', reason='loader-unavailable'))
        self.assertEqual('drain-candidate', self.pending()['phase'])
        self.assertTrue(self.send(self.guardian, {'command': 'stop', 'request': uuid.uuid4().hex})['closed'])
        self.assertEqual(0, self.guardian.wait(timeout=5))
        roles = set(j.ROLES) if claims == 3 else {'omabit'} if claims else set()
        for role in j.ROLES:
            self.record(self.envelope('drained', role=role, observation={
                'epoch': self.permit['epoch'], 'binding': partial['members'][role] if role in roles else None,
                'absent': role not in roles, 'closed': True, 'retained_state': j.digest([role, 'partial-retained']),
                'pending_ids': ['a' * 32] if role == 'dms' else []}))
        self.unload_and_appearance()
        if missing:
            (self.gate / missing).rename(self.gate / ('retained-' + missing))
        original = {p.name: p.read_bytes() for p in self.gate.iterdir() if p.is_file()}
        self.guardian, self.owner, self.shell, self.next_receipt = self.launch_guardian()
        candidate = self.candidate(shell=self.shell)
        ticket = self.handoff(candidate)
        proof = self.authority.status()['history'][-1]['handoff']
        self.assertEqual('partial-rollback', proof['mode'])
        self.assertEqual(sha(self.receipt), proof['closure']['sha256'])
        self.assertEqual(original, {p.name: p.read_bytes() for p in self.gate.iterdir() if p.is_file()})
        self.receipt = self.next_receipt
        self.activate(candidate, ticket)
        self.assertEqual('rolled-back', self.pending()['outcome'])
        return proof

    def test_crash_after_reservation_before_issue_rolls_back_without_reset(self):
        proof = self.partial_rollback(issue=False)
        self.assertEqual(['admission.json', 'issued.json'], proof['abandoned_admission']['missing'])

    def test_crash_after_issue_before_construction_rolls_back(self):
        proof = self.partial_rollback()
        self.assertEqual([], proof['abandoned_admission']['missing'])

    def test_partial_one_member_construction_rolls_back(self):
        self.partial_rollback(claims=1)

    def test_all_guards_before_loaded_receipt_can_roll_back(self):
        self.partial_rollback(claims=3)

    def test_lost_candidate_admission_record_retains_evidence_on_rollback(self):
        proof = self.partial_rollback(claims=1, missing='admission.json')
        self.assertEqual(['admission.json'], proof['abandoned_admission']['missing'])

    def test_interrupted_candidate_seal_retains_evidence_on_rollback(self):
        proof = self.partial_rollback(missing='issued.json')
        self.assertEqual(['issued.json'], proof['abandoned_admission']['missing'])



    def closed_unbound_initial(self, *, verify=True):
        self.assertIsNone(self.authority.status()['binding'])
        self.begin(self.spec)
        self.current.stop_admission(self.permit['id'], j.effect(self.pending(), 'drain-previous'))
        self.send(self.guardian, {'command': 'kill-shell'})
        self.record(self.envelope('problem', reason='reader-owner-lost'))
        closure_request = uuid.uuid4().hex
        self.send(self.guardian, {'command': 'stop', 'request': closure_request})
        self.assertEqual(0, self.guardian.wait(timeout=5))
        guardian = self.authority.status()['reservation']['owner']['guardian']
        request = self.pending()
        plan = {'loaded': request['previous'], 'fences': request['uncertain'], 'guardian': guardian,
                'closure_request': closure_request, 'runtime_binding_sha256': sha(self.binding_path),
                'members': {role: {'binding': self.loaded['members'][role], 'application_sha256': j.digest([role, 'app']),
                          'pending_ids': ['a' * 32] if role == 'dms' else [], 'credential_refs': [],
                          'transport_sha256': j.digest([role, 'before']), 'channel_ids': []} for role in j.ROLES}}
        observation = {'plan_sha256': j.digest(plan), 'guardian': guardian, 'closure_request': closure_request,
                       'closure_receipt_sha256': sha(self.receipt), 'proof': 'waitpid-echild',
                       'members': {role: dict(value, closed=True, transport_sha256=j.digest([role, 'closed'])) for role, value in plan['members'].items()}}
        rid = uuid.uuid4().hex
        self.record(self.envelope('recovery-prepared', recovery=rid, plan=plan))
        if verify:self.record(self.envelope('recovery-verified', recovery=rid, observation=observation))

    def bind_closed(self):
        with self.store.lease() as lease:
            return self.authority.bind_closed_initial(self.store, lease, self.request, self.binding_path)

    def test_closed_initial_binding_requires_verified_cleanup(self):
        self.closed_unbound_initial(verify=False);before = self.bytes()
        with self.assertRaises(j.Conflict):self.bind_closed()
        self.assertEqual(before, self.bytes())

    def test_closed_initial_binding_preserves_adoption_and_original_owner(self):
        self.closed_unbound_initial();before = (self.store.root / 'journal.json').read_bytes()
        original = copy.deepcopy(self.authority.status()['reservation']);binding = self.bind_closed()
        self.assertEqual(self.loaded, binding['loaded']);self.assertEqual(original['owner']['guardian'], binding['guardian'])
        self.assertEqual(original, self.authority.status()['reservation']);self.assertEqual(2, self.authority.status()['format'])
        self.assertEqual(before, (self.store.root / 'journal.json').read_bytes())
        saved = self.bytes();self.assertEqual(binding, self.bind_closed());self.assertEqual(saved, self.bytes())

    def test_closed_initial_binding_changed_runtime_observation_refuses(self):
        self.closed_unbound_initial();body = json.loads(self.binding_path.read_text());body['loaded']['members']['dms']['instance'] = 'e' * 32
        save(self.binding_path, body);before = self.bytes()
        with self.assertRaises(j.Conflict):self.bind_closed()
        self.assertEqual(before, self.bytes())

    def test_closed_initial_binding_changed_original_closure_refuses(self):
        self.closed_unbound_initial();body = json.loads(self.receipt.read_text());body['request'] = 'e' * 32
        save(self.receipt, body)
        with self.assertRaises(j.Conflict):self.bind_closed()
        self.assertIsNone(self.authority.status()['binding'])

    def test_closed_initial_binding_wrong_claim_cohort_refuses(self):
        self.closed_unbound_initial();path = self.gate / 'admission.json';row = json.loads(path.read_text())
        row['claims']['dms']['owner']['start'] = str(int(row['claims']['dms']['owner']['start']) + 1);save(path, row)
        with self.assertRaises(j.Conflict):self.bind_closed()
        self.assertIsNone(self.authority.status()['binding'])

    def test_closed_initial_binding_after_unload_refuses(self):
        self.closed_unbound_initial();self.unload_and_appearance()
        with self.assertRaises(j.Conflict):self.bind_closed()
        self.assertIsNone(self.authority.status()['binding'])

    def test_closed_initial_binding_lost_reply_is_exact(self):
        self.closed_unbound_initial();write = self.authority._save_reservation
        def lost(value):write(value);raise OSError('reply lost')
        with patch.object(self.authority, '_save_reservation', side_effect=lost):
            with self.assertRaises(OSError):self.bind_closed()
        before = self.bytes();self.bind_closed();self.assertEqual(before, self.bytes())

    def test_closed_initial_binding_supports_original_reader_and_recovery_handoff(self):
        self.closed_unbound_initial();self.bind_closed()
        original = module('closed_binding_original2', ROOT / 'tests/fixtures/runtime-authority-v2/writer.py')
        old = original.Authority(self.authority.root)
        try:self.assertEqual(self.authority.status(), old.status())
        finally:old.close()
        self.unload_and_appearance()
        self.guardian, self.owner, self.shell, self.receipt = self.launch_guardian();self.next_receipt = self.receipt
        candidate = self.candidate(shell=self.shell);ticket = self.handoff(candidate);self.activate(candidate, ticket)
        self.assertEqual('loaded-recovery', self.authority.status()['history'][0]['handoff']['mode'])
        self.assertEqual('committed', self.pending()['outcome'])


def guardian_worker(token, path):
    path = Path(path)
    libc = ctypes.CDLL(None, use_errno=True)
    assert libc.prctl(36, 1, 0, 0, 0) == 0
    flag = ctypes.c_int()
    assert libc.prctl(37, ctypes.byref(flag), 0, 0, 0) == 0 and flag.value == 1
    child = subprocess.Popen([sys.executable, '-B', str(Path(__file__).with_name('test-collection-shared-admission.py')), '--shell'],
                             stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    assert select.select([child.stdout], [], [], 5)[0]
    shell = json.loads(child.stdout.readline())['identity']
    owner = dict(old.identity(os.getpid()), token=token)
    receipt = {'version': 1, 'owner': owner, 'phase': 'running', 'request': uuid.uuid4().hex,
               'root': authority.process_pair(shell), 'reaped': [], 'signals': [], 'proof': None}
    save(path, receipt)
    def emit(value):
        print(json.dumps(value), flush=True)
    emit({'owner': owner, 'shell': shell})
    def close(request):
        receipt['phase'], receipt['request'] = 'draining', request
        save(path, receipt)
        if not child.stdin.closed:
            child.stdin.close()
        try:
            code = child.wait(timeout=4)
        except subprocess.TimeoutExpired:
            child.kill();code = child.wait(timeout=4)
        receipt['reaped'].append({'pid': child.pid, 'status': code})
        until = time.monotonic() + 5
        while True:
            try:
                pid, status = os.waitpid(-1, os.WNOHANG)
            except ChildProcessError:
                break
            if pid:
                receipt['reaped'].append({'pid': pid, 'status': os.waitstatus_to_exitcode(status)})
            else:
                assert time.monotonic() < until, 'owned descendant has not exited'
                time.sleep(.02)
        receipt['phase'], receipt['proof'] = 'closed', 'waitpid-echild'
        save(path, receipt)
        emit({'closed': True})
    try:
        for raw in sys.stdin.buffer:
            request = json.loads(raw)
            if request['command'] == 'stop':
                close(request['request']);return
            if request['command'] == 'kill-shell':
                child.kill()
                receipt['signals'].append({'pid': child.pid, 'signal': int(signal.SIGKILL)})
                save(path, receipt);emit({'signaled': True});continue
            child.stdin.write(j.canonical(request) + b'\n');child.stdin.flush()
            assert select.select([child.stdout], [], [], 6)[0]
            emit(json.loads(child.stdout.readline()))
        close(uuid.uuid4().hex)
    finally:
        if child.poll() is None:
            child.kill();child.wait(timeout=4)
        for stream in (child.stdin, child.stdout, child.stderr):
            if stream and not stream.closed:stream.close()


if __name__ == '__main__':
    if len(sys.argv) == 4 and sys.argv[1] == '--guardian':
        guardian_worker(sys.argv[2], sys.argv[3])
    else:
        unittest.main()
