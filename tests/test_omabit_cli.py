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
    def test_version_matches_release_and_plugin_manifest(self):
        root = Path(__file__).parents[1]
        version = (root / "VERSION").read_text(encoding="utf-8").strip()
        manifest = json.loads((root / "omarchy-plugin" / "manifest.json").read_text(encoding="utf-8"))
        self.assertEqual(omabit.VERSION, version)
        self.assertEqual(manifest["version"], version)

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
                redirect_stdout(io.StringIO()),
            ):
                self.assertEqual(omabit.run_tend(args), 0)

            exported = json.loads(destination.read_text(encoding="utf-8"))
            self.assertEqual(exported["format"], "tend-backup-1")
            self.assertEqual(exported["sourceShip"], "~zod")
            self.assertEqual(exported["snapshot"], SAMPLE)
            self.assertNotIn("cookie", json.dumps(exported).casefold())
            self.assertEqual(os.stat(destination).st_mode & 0o777, 0o600)
            with self.assertRaisesRegex(omabit.CliError, "already exists"):
                omabit.write_private_json(destination, exported)

    def test_export_refuses_symbolic_link_target(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            destination = root / "backup.json"
            destination.symlink_to(root / "elsewhere.json")
            with self.assertRaisesRegex(omabit.CliError, "symbolic-link"):
                omabit.write_private_json(destination, {"format": "tend-backup-1"}, force=True)

    def test_open_capture_uses_omarchy_shell_payload(self):
        args = SimpleNamespace(tend_command="open", json=False, list_id=0, reminder_id=0, capture=True)
        with mock.patch.object(omabit.subprocess, "run") as run:
            self.assertEqual(omabit.run_tend(args), 0)
        command = run.call_args.args[0]
        self.assertEqual(command[:4], ["omarchy-shell", "shell", "summon", "io.omabit.tend"])
        self.assertEqual(json.loads(command[4])["mode"], "capture")


if __name__ == "__main__":
    unittest.main()
