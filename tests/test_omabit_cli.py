from __future__ import annotations

from contextlib import redirect_stdout
from datetime import datetime, timezone
import importlib.machinery
import importlib.util
import io
import json
import os
from pathlib import Path
import tempfile
from types import SimpleNamespace
import unittest
from unittest import mock


CLI_PATH = Path(__file__).parents[1] / "bin" / "omabit"
LOADER = importlib.machinery.SourceFileLoader("omabit_cli", str(CLI_PATH))
SPEC = importlib.util.spec_from_loader("omabit_cli", LOADER)
assert SPEC and SPEC.loader
omabit = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(omabit)


SAMPLE = {
    "lists": [
        {
            "id": 1,
            "title": "Inbox",
            "revision": 4,
            "reminders": [
                {"id": 7, "title": "Pay rent", "completed": False, "priority": "high", "schedule": {"due-at": "~2026.9.10..18.00.00"}},
                {"id": 8, "title": "Filed", "completed": True},
            ],
        }
    ],
    "preferences": {"default-list": 1},
}


class CliDomainTests(unittest.TestCase):
    def setUp(self):
        self.accesses = mock.patch.object(
            omabit.transport,
            "scry_accesses",
            return_value=[{"alias": 1, "host": "~zod", "owner": True, "status": "online"}],
        )
        self.accesses.start()

    def tearDown(self):
        self.accesses.stop()

    def test_version_matches_release_and_plugin_manifest(self):
        root = Path(__file__).parents[1]
        version = (root / "VERSION").read_text(encoding="utf-8").strip()
        manifest = json.loads((root / "omarchy-plugin" / "manifest.json").read_text(encoding="utf-8"))
        self.assertEqual(omabit.VERSION, version)
        self.assertEqual(manifest["version"], version)

    def test_cli_import_does_not_mutate_the_watched_plugin_tree(self):
        self.assertTrue(omabit.sys.dont_write_bytecode)

    def test_selects_default_or_named_list_and_validates_reminder(self):
        self.assertEqual(omabit.select_list(SAMPLE, None)["id"], 1)
        self.assertEqual(omabit.select_list(SAMPLE, "inbox")["id"], 1)
        self.assertEqual(omabit.find_reminder(SAMPLE["lists"][0], 7)["title"], "Pay rent")
        with self.assertRaisesRegex(omabit.CliError, "does not exist"):
            omabit.find_reminder(SAMPLE["lists"][0], 99)

    def test_urbit_date_round_trip(self):
        value = datetime(2026, 9, 10, 22, 15, 30, tzinfo=timezone.utc)
        encoded = omabit.urbit_date(value)
        self.assertEqual(encoded, "~2026.9.10..22.15.30")
        self.assertEqual(omabit.urbit_date_value(encoded), value)

    def test_machine_readable_list_filters_completed_reminders(self):
        args = SimpleNamespace(tend_command="list", json=True)
        with mock.patch.object(omabit, "snapshot", return_value=SAMPLE), redirect_stdout(io.StringIO()) as output:
            self.assertEqual(omabit.run_tend(args), 0)
        rows = json.loads(output.getvalue())
        self.assertEqual([(row["listId"], row["id"]) for row in rows], [(1, 7)])

    def test_add_uses_current_list_revision(self):
        args = SimpleNamespace(tend_command="add", json=True, title="Buy milk", list_selector=None, tag=["shop"])
        with mock.patch.object(omabit, "snapshot", return_value=SAMPLE), mock.patch.object(omabit, "poke") as poke, redirect_stdout(io.StringIO()):
            self.assertEqual(omabit.run_tend(args), 0)
        action = poke.call_args.args[2]["add-reminder"]
        self.assertEqual(action["list-id"], 1)
        self.assertEqual(action["base-revision"], 4)
        self.assertEqual(action["tags"], ["shop"])

    def test_shared_mutation_is_blocked_before_poke_when_owner_is_offline(self):
        args = SimpleNamespace(tend_command="add", json=True, title="Blocked", list_selector=None, tag=[])
        with (
            mock.patch.object(omabit, "snapshot", return_value=SAMPLE),
            mock.patch.object(omabit.transport, "scry_accesses", return_value=[{"alias": 1, "host": "~bus", "owner": False, "status": "offline"}]),
            mock.patch.object(omabit, "poke") as poke,
        ):
            with self.assertRaisesRegex(omabit.CliError, "read-only"):
                omabit.run_tend(args)
        poke.assert_not_called()

    def test_share_and_unshare_use_selected_list_without_rank_assumptions(self):
        share = SimpleNamespace(
            tend_command="share",
            json=True,
            list_selector="Inbox",
            ship="~sampel-palnet",
            allow_invites=True,
        )
        with mock.patch.object(omabit, "snapshot", return_value=SAMPLE), mock.patch.object(omabit, "poke") as poke, redirect_stdout(io.StringIO()) as output:
            self.assertEqual(omabit.run_tend(share), 0)
        invited = poke.call_args.args[2]["invite-member"]
        self.assertEqual(invited["list-id"], 1)
        self.assertEqual(invited["ship"], "~sampel-palnet")
        self.assertTrue(invited["can-invite"])
        response = json.loads(output.getvalue())
        self.assertEqual(response["invitationUri"], f"omabit://tend/invite/zod/{invited['operation-id']}")

        unshare = SimpleNamespace(tend_command="unshare", json=True, list_selector="1", ship="~sampel-palnet")
        with mock.patch.object(omabit, "snapshot", return_value=SAMPLE), mock.patch.object(omabit, "poke") as poke, redirect_stdout(io.StringIO()):
            self.assertEqual(omabit.run_tend(unshare), 0)
        removed = poke.call_args.args[2]["remove-member"]
        self.assertEqual(removed["list-id"], 1)
        self.assertEqual(removed["ship"], "~sampel-palnet")

    def test_share_accepts_documented_can_invite_flag_and_legacy_alias(self):
        for flag in ("--can-invite", "--allow-invites"):
            args = omabit.parser().parse_args(["tend", "share", "Inbox", "~sampel-palnet", flag])
            self.assertTrue(args.allow_invites)

    def test_leave_uses_local_alias(self):
        args = SimpleNamespace(tend_command="leave", json=True, list_selector="1")
        with mock.patch.object(omabit, "snapshot", return_value=SAMPLE), mock.patch.object(omabit, "poke") as poke, redirect_stdout(io.StringIO()):
            self.assertEqual(omabit.run_tend(args), 0)
        self.assertEqual(poke.call_args.args[2]["leave-shared-list"]["list-id"], 1)

    def test_activity_uses_selected_local_list_alias(self):
        args = SimpleNamespace(tend_command="activity", json=True, list_selector="Inbox")
        events = [{"id": "event-1", "actor": "~bus", "kind": "completed"}]
        with (
            mock.patch.object(omabit, "snapshot", return_value=SAMPLE),
            mock.patch.object(omabit.transport, "scry_activities", return_value=events) as scry,
            redirect_stdout(io.StringIO()) as output,
        ):
            self.assertEqual(omabit.run_tend(args), 0)
        scry.assert_called_once_with(mock.ANY, mock.ANY, 1)
        self.assertEqual(json.loads(output.getvalue()), events)

    def test_invitation_cli_lists_accepts_and_declines(self):
        listing = SimpleNamespace(tend_command="invitations", json=True)
        with (
            mock.patch.object(omabit.transport, "scry_invitations", return_value=[{"token": "invite-1"}]),
            redirect_stdout(io.StringIO()) as output,
        ):
            self.assertEqual(omabit.run_tend(listing), 0)
        self.assertEqual(json.loads(output.getvalue()), [{"token": "invite-1"}])

        for command, action_name in (("accept", "accept-invitation"), ("decline", "decline-invitation")):
            args = SimpleNamespace(tend_command=command, json=True, host="~sampel-palnet", token="invite-1")
            with mock.patch.object(omabit, "snapshot", return_value={"protocol-version": 1}) as snapshot, mock.patch.object(omabit, "poke") as poke, redirect_stdout(io.StringIO()):
                self.assertEqual(omabit.run_tend(args), 0)
            snapshot.assert_called_once()
            body = poke.call_args.args[2][action_name]
            self.assertEqual(body["host"], "~sampel-palnet")
            self.assertEqual(body["token"], "invite-1")

    def test_invitation_uri_round_trips_and_is_accepted_by_cli(self):
        uri = omabit.invitation_uri("~sampel-palnet", "invite token/1")
        self.assertEqual(uri, "omabit://tend/invite/sampel-palnet/invite%20token%2F1")
        self.assertEqual(omabit.invitation_reference(uri), ("~sampel-palnet", "invite token/1"))

        args = SimpleNamespace(tend_command="accept", json=True, host=uri, token=None)
        with mock.patch.object(omabit, "snapshot", return_value={"protocol-version": 1}), mock.patch.object(omabit, "poke") as poke, redirect_stdout(io.StringIO()):
            self.assertEqual(omabit.run_tend(args), 0)
        body = poke.call_args.args[2]["accept-invitation"]
        self.assertEqual((body["host"], body["token"]), ("~sampel-palnet", "invite token/1"))

        for invalid in ("omabit://other/invite/zod/x", "omabit://tend/invite/zod", "omabit://tend/invite/zod/x?leak=1"):
            with self.assertRaises(omabit.CliError):
                omabit.invitation_reference(invalid)

    def test_status_reports_the_negotiated_protocol(self):
        args = SimpleNamespace(tend_command="status", json=True)
        configured = {"ship": "zod", "baseUrl": "http://127.0.0.1:8080"}
        with (
            mock.patch.object(omabit.transport, "connection_status", return_value=configured),
            mock.patch.object(omabit, "snapshot", return_value={"protocol-version": 1}),
            redirect_stdout(io.StringIO()) as output,
        ):
            self.assertEqual(omabit.run_tend(args), 0)
        self.assertEqual(json.loads(output.getvalue())["protocolVersion"], 1)

    def test_complete_checks_entity_before_poking(self):
        args = SimpleNamespace(tend_command="complete", json=True, list_id=1, reminder_ids=[7])
        with mock.patch.object(omabit, "snapshot", return_value=SAMPLE), mock.patch.object(omabit, "poke") as poke, redirect_stdout(io.StringIO()):
            self.assertEqual(omabit.run_tend(args), 0)
        action = poke.call_args.args[2]["set-completed"]
        self.assertEqual(action["base-revision"], 4)
        self.assertTrue(action["completed"])

    def test_multiple_completions_use_one_atomic_batch_action(self):
        sample = json.loads(json.dumps(SAMPLE))
        sample["lists"][0]["reminders"].append({"id": 9, "title": "Call", "completed": False})
        args = SimpleNamespace(tend_command="complete", json=True, list_id=1, reminder_ids=[7, 9])
        with mock.patch.object(omabit, "snapshot", return_value=sample), mock.patch.object(omabit, "poke") as poke, redirect_stdout(io.StringIO()):
            self.assertEqual(omabit.run_tend(args), 0)
        action = poke.call_args.args[2]["batch-set-completed"]
        self.assertEqual(action["reminder-ids"], [7, 9])
        self.assertEqual(action["base-revision"], 4)

    def test_snooze_accepts_an_absolute_iso_time(self):
        args = SimpleNamespace(
            tend_command="snooze",
            json=True,
            list_id=1,
            reminder_id=7,
            minutes=None,
            until="2099-01-02T03:04:05Z",
        )
        with mock.patch.object(omabit, "snapshot", return_value=SAMPLE), mock.patch.object(omabit, "poke") as poke, redirect_stdout(io.StringIO()):
            self.assertEqual(omabit.run_tend(args), 0)
        action = poke.call_args.args[2]["snooze-reminder"]
        self.assertEqual(action["until"], "~2099.1.2..03.04.05")

    def test_snooze_rejects_a_nonpositive_duration_before_poking(self):
        args = SimpleNamespace(
            tend_command="snooze",
            json=True,
            list_id=1,
            reminder_id=7,
            minutes=0,
            until=None,
        )
        with mock.patch.object(omabit, "snapshot", return_value=SAMPLE), mock.patch.object(omabit, "poke") as poke:
            with self.assertRaisesRegex(omabit.CliError, "must be positive"):
                omabit.run_tend(args)
        poke.assert_not_called()

    def test_batch_delete_requires_explicit_confirmation(self):
        args = SimpleNamespace(tend_command="delete", json=True, list_id=1, reminder_ids=[7], yes=False)
        with mock.patch.object(omabit, "snapshot", return_value=SAMPLE):
            with self.assertRaisesRegex(omabit.CliError, "without --yes"):
                omabit.run_tend(args)

    def test_export_writes_private_versioned_snapshot_without_credentials(self):
        with tempfile.TemporaryDirectory() as directory:
            destination = Path(directory) / "tend-backup.json"
            args = SimpleNamespace(tend_command="export", json=True, path=str(destination), force=False)
            status = {"ship": "~zod", "baseUrl": "http://127.0.0.1:8080"}
            with (
                mock.patch.object(omabit, "snapshot", return_value=SAMPLE),
                mock.patch.object(omabit.transport, "connection_status", return_value=status),
                mock.patch.object(omabit.transport, "scry_accesses", return_value=[{"alias": 1, "owner": True}]),
                redirect_stdout(io.StringIO()),
            ):
                self.assertEqual(omabit.run_tend(args), 0)

            exported = json.loads(destination.read_text(encoding="utf-8"))
            self.assertEqual(exported["format"], "tend-backup-1")
            self.assertEqual(exported["sourceShip"], "~zod")
            self.assertEqual(exported["snapshot"]["lists"], SAMPLE["lists"])
            self.assertEqual(exported["snapshot"]["preferences"]["default-list"], 1)
            self.assertEqual(exported["snapshot"]["snoozes"], [])
            self.assertNotIn("cookie", json.dumps(exported).casefold())
            self.assertEqual(os.stat(destination).st_mode & 0o777, 0o600)
            with self.assertRaisesRegex(omabit.CliError, "already exists"):
                omabit.write_private_json(destination, exported)

    def test_export_excludes_non_authoritative_replicas(self):
        state = json.loads(json.dumps(SAMPLE))
        state["lists"].append({"id": 9, "title": "Remote", "revision": 2, "reminders": []})
        state["preferences"]["pinned-lists"] = [9, 1]
        state["snoozes"] = [{"list-id": 9, "reminder-id": 4, "until": "~2099.1.1..00.00.00"}]
        exported = omabit.owned_snapshot(state, [
            {"alias": 1, "owner": True},
            {"alias": 9, "owner": False},
        ])
        self.assertEqual([item["id"] for item in exported["lists"]], [1])
        self.assertEqual(exported["preferences"]["pinned-lists"], [1])
        self.assertEqual(exported["snoozes"], [])

    def test_restore_requires_confirmation_and_submits_one_atomic_action(self):
        backup = {
            "format": "tend-backup-1",
            "exportedAt": "2026-09-10T20:00:00Z",
            "sourceShip": "~sampel-palnet",
            "snapshot": {
                "lists": [],
                "preferences": {
                    "revision": 0,
                    "default-list": None,
                    "pinned-lists": [],
                    "pinned-views": ["today", "all"],
                    "snooze-presets": [300],
                },
                "snoozes": [],
            },
        }
        with tempfile.TemporaryDirectory() as directory:
            source = Path(directory) / "backup.json"
            source.write_text(json.dumps(backup), encoding="utf-8")
            refused = SimpleNamespace(tend_command="restore", json=True, path=str(source), yes=False)
            with self.assertRaisesRegex(omabit.CliError, "without --yes"):
                omabit.run_tend(refused)

            accepted = SimpleNamespace(tend_command="restore", json=True, path=str(source), yes=True)
            with (
                mock.patch.object(omabit, "snapshot", return_value={"lists": []}),
                mock.patch.object(omabit, "poke") as poke,
                mock.patch.object(omabit.transport, "scry_receipt", return_value={"snapshot": {"lists": []}}),
                redirect_stdout(io.StringIO()),
            ):
                self.assertEqual(omabit.run_tend(accepted), 0)
            body = poke.call_args.args[2]["restore-empty"]
            self.assertEqual(body["lists"], [])
            self.assertEqual(body["preferences"]["badge-mode"], "today")
            self.assertEqual(body["preferences"]["all-day-alert-minute"], 540)

    def test_restore_surfaces_a_gall_rejection(self):
        backup = {
            "format": "tend-backup-1",
            "exportedAt": "2026-09-10T20:00:00Z",
            "sourceShip": "~zod",
            "snapshot": {"lists": [], "preferences": {"revision": 0, "default-list": None, "pinned-lists": [], "pinned-views": [], "snooze-presets": []}, "snoozes": []},
        }
        with tempfile.TemporaryDirectory() as directory:
            source = Path(directory) / "backup.json"
            source.write_text(json.dumps(backup), encoding="utf-8")
            args = SimpleNamespace(tend_command="restore", json=True, path=str(source), yes=True)
            with (
                mock.patch.object(omabit, "snapshot", return_value={"lists": []}),
                mock.patch.object(omabit, "poke"),
                mock.patch.object(omabit.transport, "scry_receipt", return_value={"rejected": {"reason": "not-empty"}}),
            ):
                with self.assertRaisesRegex(omabit.CliError, "not-empty"):
                    omabit.run_tend(args)

    def test_restore_refuses_a_nonempty_agent_before_poking(self):
        backup = {
            "format": "tend-backup-1",
            "exportedAt": "2026-09-10T20:00:00Z",
            "sourceShip": "~zod",
            "snapshot": {"lists": [], "preferences": {"revision": 0, "default-list": None, "pinned-lists": [], "pinned-views": [], "snooze-presets": []}, "snoozes": []},
        }
        with tempfile.TemporaryDirectory() as directory:
            source = Path(directory) / "backup.json"
            source.write_text(json.dumps(backup), encoding="utf-8")
            args = SimpleNamespace(tend_command="restore", json=True, path=str(source), yes=True)
            with mock.patch.object(omabit, "snapshot", return_value=SAMPLE), mock.patch.object(omabit, "poke") as poke:
                with self.assertRaisesRegex(omabit.CliError, "requires an empty agent"):
                    omabit.run_tend(args)
            poke.assert_not_called()

    def test_export_refuses_symbolic_link_target(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            destination = root / "backup.json"
            destination.symlink_to(root / "elsewhere.json")
            with self.assertRaisesRegex(omabit.CliError, "symbolic-link"):
                omabit.write_private_json(destination, {"format": "tend-backup-1"}, force=True)

    def test_backup_validation_is_offline_structural_and_cycle_safe(self):
        backup = {
            "format": "tend-backup-1",
            "exportedAt": "2026-09-10T20:00:00Z",
            "sourceShip": "~sampel-palnet",
            "snapshot": {
                "lists": [{
                    "id": 1,
                    "title": "Team",
                    "revision": 5,
                    "sections": [{"id": 2, "title": "Now", "rank": 10}],
                    "reminders": [
                        {
                            "id": 3,
                            "title": "Parent",
                            "notes": "Details",
                            "revision": 2,
                            "rank": 10,
                            "completed": False,
                            "priority": "high",
                            "tags": ["work"],
                            "parent-id": None,
                            "section-id": 2,
                            "url": "https://example.com",
                            "schedule": {
                                "due-at": "~2026.9.11..20.00.00",
                                "all-day": False,
                                "timezone": "America/Los_Angeles",
                                "early-seconds": [900],
                                "recurrence": {
                                    "frequency": "weekly",
                                    "interval": 1,
                                    "weekdays": [3],
                                    "month-days": [],
                                    "month-week": None,
                                    "end-at": None,
                                    "max-occurrences": 4,
                                },
                            },
                        },
                        {
                            "id": 4,
                            "title": "Child",
                            "notes": "",
                            "revision": 1,
                            "rank": 20,
                            "completed": False,
                            "priority": "none",
                            "tags": [],
                            "parent-id": 3,
                            "section-id": 2,
                            "url": None,
                            "schedule": None,
                        },
                    ],
                }],
                "preferences": {
                    "revision": 1,
                    "default-list": 1,
                    "pinned-lists": [1],
                    "pinned-views": ["today", "assigned"],
                    "snooze-presets": [300, 900],
                },
                "snoozes": [{"list-id": 1, "reminder-id": 3, "until": "~2026.9.11..21.00.00"}],
            },
        }
        summary = omabit.validate_backup(backup)
        self.assertEqual(summary["status"], "valid")
        self.assertEqual(summary["lists"], 1)
        self.assertEqual(summary["reminders"], 2)

        backup["snapshot"]["lists"][0]["reminders"][0]["parent-id"] = 4
        with self.assertRaisesRegex(omabit.CliError, "parent cycle"):
            omabit.validate_backup(backup)

    def test_validate_backup_reads_no_ship_state_and_refuses_symlinks(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            source = root / "backup.json"
            source.write_text("{}", encoding="utf-8")
            link = root / "link.json"
            link.symlink_to(source)
            with self.assertRaisesRegex(omabit.CliError, "Could not open Tend backup"):
                omabit.read_backup(link)

            args = SimpleNamespace(tend_command="validate-backup", json=True, path=str(source))
            with mock.patch.object(omabit, "snapshot") as snapshot, redirect_stdout(io.StringIO()):
                with self.assertRaisesRegex(omabit.CliError, "format must be"):
                    omabit.run_tend(args)
            snapshot.assert_not_called()

    def test_open_capture_uses_omarchy_shell_payload(self):
        args = SimpleNamespace(tend_command="open", json=False, list_id=0, reminder_id=0, capture=True)
        with mock.patch.object(omabit.subprocess, "run") as run:
            self.assertEqual(omabit.run_tend(args), 0)
        command = run.call_args.args[0]
        self.assertEqual(command[:4], ["omarchy-shell", "shell", "summon", "io.omabit.tend"])
        self.assertEqual(json.loads(command[4])["mode"], "capture")


if __name__ == "__main__":
    unittest.main()
