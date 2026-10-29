#!/usr/bin/env python3
"""Shared-authority reservation and filesystem/process failure boundaries."""
import copy
import json
import os
from pathlib import Path
import select
import shutil
import socket
import subprocess
import sys
import tempfile
import unittest
import uuid
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / 'native/collections'))
import admission as admission
import journal as journal
import runtime_authority as authority


def specification():
    return {'format': 1, 'name': 'stable', 'members': {
        role: {'plugin_id': admission.CANONICAL[role] + '.qualification-stable',
               'profile': 'qualification-stable', 'artifact_sha256': journal.digest(['fixture', role]),
               'endpoint_ref': None, 'paths': {kind: '/isolated/' + role + '/' + kind for kind in journal.PATHS}}
        for role in journal.ROLES}}


def save(path, value):
    path.write_bytes(journal.canonical(value) + b'\n')
    path.chmod(0o600)


class AuthorityTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix='omabit-runtime-authority-')
        self.root = Path(self.temp.name)
        self.runtime = self.root / 'runtime'
        self.runtime.mkdir(mode=0o700)
        self.display = self.runtime / 'wayland-test'
        self.server = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
        self.server.bind(str(self.display))
        self.server.listen(128)
        self.store = journal.Store(self.root / 'coordinator', create=True)
        self.gate = self.root / 'gate'
        self.gate.mkdir(mode=0o700)
        self.spec = specification()
        self.permit = {'id': uuid.uuid4().hex, 'request': uuid.uuid4().hex, 'epoch': uuid.uuid4().hex,
                       'specification': self.spec, 'shell': admission.identity(os.getpid())}
        self.authority = None
        self.children = []
        self.addCleanup(self.cleanup)

    def cleanup(self):
        for child in self.children:
            if child.poll() is None:
                child.kill()
                child.wait(timeout=5)
            for stream in (child.stdin, child.stdout, child.stderr):
                if stream:
                    stream.close()
        if self.authority:
            self.authority.close()
        self.store.close()
        self.server.close()
        self.temp.cleanup()

    def provision(self):
        with self.store.lease() as lease:
            self.authority = authority.Authority.provision(self.display, self.store, lease)
            self.store.record(lease, {'type': 'register', 'spec': self.spec})
        return self.authority

    def reserve(self, permit=None, gate=None):
        with self.store.lease() as lease:
            return self.authority.reserve(self.store, lease, gate or self.gate, permit or self.permit)

    def child(self, command, store=None, gate=None, permit=None):
        path = self.root / ('request-' + uuid.uuid4().hex + '.json')
        save(path, {'command': command, 'authority': str(self.authority.root),
                    'store': str(store or self.store.root), 'gate': str(gate or self.gate),
                    'permit': permit or self.permit})
        child = subprocess.Popen([sys.executable, '-B', __file__, '--controller', str(path)],
                                 stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        self.children.append(child)
        return child

    def receive(self, child):
        self.assertTrue(select.select([child.stdout], [], [], 6)[0], 'controller response deadline')
        line = child.stdout.readline()
        self.assertTrue(line, 'controller exited without a result')
        return json.loads(line)

    def test_registered_initial_reservation_replays_byte_identically(self):
        self.provision()
        ticket = self.reserve()
        path = self.authority.root / 'reservation.json'
        before = path.read_bytes()
        self.assertEqual(ticket, self.reserve())
        self.assertEqual(before, path.read_bytes())
        self.assertEqual(self.permit, self.authority.verify_reservation(ticket, self.gate, self.permit)['permit'])

    def test_unregistered_collection_cannot_reserve(self):
        with self.store.lease() as lease:
            self.authority = authority.Authority.provision(self.display, self.store, lease)
            with self.assertRaises(journal.Conflict):
                self.authority.reserve(self.store, lease, self.gate, self.permit)
        self.assertIsNone(self.authority.status()['reservation'])

    def test_live_lease_required(self):
        self.provision()
        with self.store.lease() as lease:
            pass
        with self.assertRaises(journal.Conflict):
            self.authority.reserve(self.store, lease, self.gate, self.permit)

    def test_unrelated_coordinator_cannot_reserve_same_runtime(self):
        self.provision()
        other = journal.Store(self.root / 'other-coordinator', create=True)
        try:
            with other.lease() as lease:
                other.record(lease, {'type': 'register', 'spec': self.spec})
                with self.assertRaises(journal.Conflict):
                    self.authority.reserve(other, lease, self.gate, self.permit)
        finally:
            other.close()
        self.assertIsNone(self.authority.status()['reservation'])

    def test_another_coordinator_cannot_provision_second_anchor(self):
        self.provision()
        other = journal.Store(self.root / 'other-coordinator', create=True)
        try:
            with other.lease() as lease:
                with self.assertRaises((FileExistsError, journal.Conflict)):
                    authority.Authority.provision(self.display, other, lease)
        finally:
            other.close()
        self.assertFalse((other.root / authority.MIRROR).exists())

    def test_missing_anchor_cannot_be_reprovisioned_by_an_unrelated_journal(self):
        self.provision()
        self.reserve()
        self.authority.root.rename(self.runtime / 'retained-authority')
        other = journal.Store(self.root / 'other-coordinator', create=True)
        try:
            with other.lease() as lease:
                with self.assertRaises(journal.Conflict):
                    authority.Authority.provision(self.display, other, lease)
        finally:
            other.close()
        self.assertFalse((self.runtime / authority.ANCHOR).exists())
        self.assertTrue((self.runtime / authority.ANCHOR_SEAL).exists())

    def test_new_permit_or_directory_never_releases_old_reservation(self):
        self.provision()
        ticket = self.reserve()
        other = self.root / 'another-gate'
        other.mkdir(mode=0o700)
        before = (self.authority.root / 'reservation.json').read_bytes()
        for change in ('id', 'request', 'epoch', 'artifact'):
            permit = copy.deepcopy(self.permit)
            if change == 'artifact':
                permit['specification']['members']['theme']['artifact_sha256'] = 'a' * 64
            else:
                permit[change] = uuid.uuid4().hex
            with self.assertRaises(journal.Conflict):
                self.reserve(permit)
        with self.assertRaises(journal.Conflict):
            self.reserve(gate=other)
        self.assertEqual(before, (self.authority.root / 'reservation.json').read_bytes())
        self.authority.verify_reservation(ticket, self.gate, self.permit)

    def test_nonempty_initial_gate_refuses(self):
        self.provision()
        (self.gate / 'issued.json').write_text('{}')
        with self.assertRaises(journal.Conflict):
            self.reserve()
        self.assertIsNone(self.authority.status()['reservation'])

    def test_reservation_survives_controller_sigkill_and_lost_reply(self):
        self.provision()
        child = self.child('reserve-and-hold')
        saved = self.receive(child)
        self.assertTrue(saved['reserved'])
        # A real second process cannot take the live canonical coordinator lease.
        competitor = self.child('reserve')
        self.assertEqual('BlockingIOError', self.receive(competitor)['refused'])
        self.assertEqual(0, competitor.wait(timeout=5))
        before = (self.authority.root / 'reservation.json').read_bytes()
        child.kill()
        self.assertEqual(-9, child.wait(timeout=5))
        # The same intended reservation can be observed/re-synced after loss;
        # a fresh request cannot interpret the now-free process lock as cleanup.
        retry = self.child('reserve')
        self.assertEqual(saved['ticket'], self.receive(retry)['ticket'])
        self.assertEqual(0, retry.wait(timeout=5))
        replacement = copy.deepcopy(self.permit)
        replacement['id'] = uuid.uuid4().hex
        denied = self.child('reserve', permit=replacement)
        self.assertEqual('Conflict', self.receive(denied)['refused'])
        self.assertEqual(0, denied.wait(timeout=5))
        self.assertEqual(before, (self.authority.root / 'reservation.json').read_bytes())

    def test_distinct_coordinator_process_refuses(self):
        self.provision()
        other = journal.Store(self.root / 'other-coordinator', create=True)
        try:
            with other.lease() as lease:
                other.record(lease, {'type': 'register', 'spec': self.spec})
            child = self.child('reserve', store=other.root)
            self.assertEqual('Conflict', self.receive(child)['refused'])
            self.assertEqual(0, child.wait(timeout=5))
        finally:
            other.close()

    def test_reservation_missing_is_not_empty(self):
        self.provision()
        self.reserve()
        (self.authority.root / 'reservation.json').unlink()
        with self.assertRaises(FileNotFoundError):
            self.reserve()
        self.assertFalse((self.authority.root / 'reservation.json').exists())

    def test_replaced_gate_refuses_exact_retry(self):
        self.provision()
        self.reserve()
        self.gate.rename(self.root / 'retained-gate')
        self.gate.mkdir(mode=0o700)
        with self.assertRaises(journal.Conflict):
            self.reserve()

    def test_replaced_authority_directory_refuses_held_and_new_objects(self):
        self.provision()
        ticket = self.reserve()
        root = self.authority.root
        root.rename(self.runtime / 'retained-authority')
        shutil.copytree(self.runtime / 'retained-authority', root)
        with self.assertRaises(journal.Conflict):
            self.authority.verify_reservation(ticket, self.gate, self.permit)
        with self.assertRaises(journal.Conflict):
            authority.Authority(root)

    def test_replaced_canonical_journal_refuses(self):
        self.provision()
        self.store.root.rename(self.root / 'retained-journal')
        shutil.copytree(self.root / 'retained-journal', self.store.root)
        with self.assertRaises(journal.Conflict):
            self.reserve()

    def test_replaced_authority_lock_refuses(self):
        self.provision()
        lock = self.authority.root / 'authority.lock'
        lock.rename(lock.with_name('retained-lock'))
        save(lock, {})
        with self.assertRaises(journal.Conflict):
            self.reserve()

    def test_replaced_coordinator_lock_refuses(self):
        self.provision()
        lock = self.store.root / 'coordinator.lock'
        lock.rename(lock.with_name('retained-lock'))
        save(lock, {})
        with self.assertRaises(journal.Conflict):
            self.reserve()

    def test_replaced_socket_refuses_without_erasing_reservation(self):
        self.provision()
        self.reserve()
        before = (self.authority.root / 'reservation.json').read_bytes()
        self.display.rename(self.runtime / 'retained-socket')
        with socket.socket(socket.AF_UNIX, socket.SOCK_STREAM) as new:
            new.bind(str(self.display))
            new.listen(1)
            with self.assertRaises(journal.Conflict):
                self.reserve()
        self.assertEqual(before, (self.authority.root / 'reservation.json').read_bytes())

    def test_display_directory_symlink_refuses(self):
        alias = self.root / 'alias'
        alias.symlink_to(self.runtime, target_is_directory=True)
        with self.store.lease() as lease:
            with self.assertRaises((OSError, journal.Conflict)):
                authority.Authority.provision(alias / self.display.name, self.store, lease)
        self.assertFalse((self.runtime / authority.ANCHOR).exists())

    def test_wrong_ticket_cannot_verify_reservation(self):
        self.provision()
        ticket = self.reserve()
        with self.assertRaises(journal.Conflict):
            self.authority.verify_reservation(dict(ticket, format=True), self.gate, self.permit)
        for key in ('authority', 'reservation', 'root', 'format'):
            bad = copy.deepcopy(ticket)
            bad[key] = 99 if key == 'format' else 'wrong'
            with self.assertRaises(journal.Conflict):
                self.authority.verify_reservation(bad, self.gate, self.permit)

    def test_changed_mirror_refuses_and_is_preserved(self):
        self.provision()
        path = self.store.root / authority.MIRROR
        value = json.loads(path.read_text())
        value['id'] = uuid.uuid4().hex
        save(path, value)
        before = path.read_bytes()
        with self.assertRaises(journal.Conflict):
            self.reserve()
        self.assertEqual(before, path.read_bytes())

    def test_missing_ready_seal_fences_existing_object(self):
        self.provision()
        (self.authority.root / 'ready.json').unlink()
        with self.assertRaises(FileNotFoundError):
            self.reserve()

    def test_unknown_reservation_format_refuses_unchanged(self):
        self.provision()
        path = self.authority.root / 'reservation.json'
        row = json.loads(path.read_text())
        row['format'] = 999
        save(path, row)
        before = path.read_bytes()
        with self.assertRaises(journal.Conflict):
            self.reserve()
        self.assertEqual(before, path.read_bytes())

    def test_partial_provision_never_resets_or_reprovisions(self):
        original = authority.write_new
        def interrupted(fd, name, value):
            if name == 'ready.json':
                raise OSError('injected before ready seal')
            return original(fd, name, value)
        with self.store.lease() as lease:
            with patch.object(authority, 'write_new', side_effect=interrupted):
                with self.assertRaises(OSError):
                    authority.Authority.provision(self.display, self.store, lease)
            with self.assertRaises(journal.Conflict):
                authority.Authority.provision(self.display, self.store, lease)
        self.assertTrue((self.store.root / authority.MIRROR).exists())
        self.assertTrue((self.runtime / authority.ANCHOR / 'reservation.json').exists())
        with self.assertRaises(FileNotFoundError):
            authority.Authority(self.runtime / authority.ANCHOR)

    def test_lost_reservation_publish_reply_replays_exactly(self):
        self.provision()
        original = self.authority._save_reservation
        def lost(value):
            original(value)
            raise OSError('injected lost durable reply')
        with patch.object(self.authority, '_save_reservation', side_effect=lost):
            with self.assertRaises(OSError):
                self.reserve()
        before = (self.authority.root / 'reservation.json').read_bytes()
        self.reserve()
        self.assertEqual(before, (self.authority.root / 'reservation.json').read_bytes())

    def test_status_never_creates_lock_or_repairs_missing_evidence(self):
        self.provision()
        lock = self.authority.root / 'authority.lock'
        lock.unlink()
        before = sorted(p.name for p in self.authority.root.iterdir())
        with self.assertRaises(FileNotFoundError):
            self.authority.status()
        self.assertEqual(before, sorted(p.name for p in self.authority.root.iterdir()))


def controller(path):
    request = json.loads(Path(path).read_text())
    a = store = None
    try:
        a = authority.Authority(request['authority'])
        store = journal.Store(request['store'])
        with store.lease() as lease:
            ticket = a.reserve(store, lease, request['gate'], request['permit'])
            print(json.dumps({'reserved': True, 'ticket': ticket}), flush=True)
            if request['command'] == 'reserve-and-hold':
                sys.stdin.read(1)
    except (OSError, ValueError) as error:
        print(json.dumps({'refused': type(error).__name__}), flush=True)
    finally:
        if store:
            store.close()
        if a:
            a.close()


if __name__ == '__main__':
    if len(sys.argv) == 3 and sys.argv[1] == '--controller':
        controller(sys.argv[2])
    else:
        unittest.main()
