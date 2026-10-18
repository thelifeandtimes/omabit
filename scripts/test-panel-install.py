#!/usr/bin/env python3
"""Verify snapshot replacement refuses local changes and unrelated files."""
import contextlib
import importlib.util
import io
import json
import multiprocessing
import os
import pathlib
import signal
import subprocess
import tempfile
import unittest
from unittest.mock import patch

spec = importlib.util.spec_from_file_location("panel_install", pathlib.Path(__file__).with_name("install-panel.py"))
installer = importlib.util.module_from_spec(spec)
spec.loader.exec_module(installer)

with tempfile.TemporaryDirectory(prefix="omabit-install-test-") as directory:
    destination = pathlib.Path(directory) / "plugin"
    source = pathlib.Path(directory) / "source"
    (source / "plugins/omabit").mkdir(parents=True)
    (source / "plugins/omabit/Model.js").write_text("// fixture\n")
    (source / "build").mkdir()
    for command in ("omabit", "urbit-collector", "urbit-runner"):
        (source / "build" / command).write_text("inert fixture\n")
    with patch.object(installer, "ROOT", source), contextlib.redirect_stdout(io.StringIO()):
        installer.install(destination)
        installer.install(destination, replace=True)
    original = (destination / "Model.js").read_text()
    (destination / "Model.js").write_text(original + "\n// user change\n")
    try:
        installer.install(destination, replace=True)
        raise AssertionError("overwrote user work")
    except RuntimeError:
        assert (destination / "Model.js").read_text().endswith("// user change\n")
    (destination / "Model.js").write_text(original)
    (destination / "extra.txt").write_text("unrelated user file")
    try:
        installer.install(destination, replace=True)
        raise AssertionError("removed unrelated file")
    except RuntimeError:
        assert (destination / "extra.txt").read_text() == "unrelated user file"
print("PASS: snapshot install/update, local modifications preserved, unrelated files preserved")


class Clock:
    now = 0.0

    def monotonic(self):
        return self.now

    def sleep(self, delay):
        self.now += delay


def test_enable(discovered_after=None, invalid=False, confirm=True):
    clock = Clock()
    scans = 0
    enables = 0
    enabled = False

    def run(args, **kwargs):
        nonlocal scans, enables, enabled
        assert 0 < kwargs["timeout"] <= 3
        result = ""
        if args[-1] == "rescanPlugins":
            scans += 1
        elif args[-1] == "listPlugins":
            discovered = discovered_after is not None and clock.now >= discovered_after
            result = "not JSON" if invalid else json.dumps(
                [{"id": installer.PLUGIN, "enabled": enabled}] if discovered else [])
        elif args[1:3] == ["plugin", "enable"]:
            assert discovered_after is not None and clock.now >= discovered_after, "enabled before discovery"
            enables += 1
            enabled = confirm
        else:
            raise AssertionError(args)
        return subprocess.CompletedProcess(args, 0, result, "")

    with patch.object(installer.subprocess, "run", side_effect=run), \
            patch.object(installer.time, "monotonic", side_effect=clock.monotonic), \
            patch.object(installer.time, "sleep", side_effect=clock.sleep):
        if discovered_after is not None and not invalid and confirm:
            installer.enable_installed(timeout=3)
            assert enables == 1 and enabled
            assert scans >= 2  # re-request after an older scan ignored the first request
        else:
            try:
                installer.enable_installed(timeout=3)
                raise AssertionError("reported success before enabled status was confirmed")
            except RuntimeError as error:
                assert "panel is installed" in str(error)
                assert "Retry without reinstalling" in str(error)
            assert enables == (1 if discovered_after is not None and not invalid else 0)
            assert clock.now <= 3


test_enable(discovered_after=1.5)
test_enable()
test_enable(invalid=True)
test_enable(discovered_after=0, confirm=False)
print("PASS: asynchronous discovery, ignored rescan, bounded timeout, invalid reply, and enable confirmation")


def test_cold_replace(locked=False, fail_install=False):
    with tempfile.TemporaryDirectory(prefix="omabit-cold-install-") as directory:
        path = pathlib.Path(directory)
        (path / "shell").mkdir()
        (path / "shell/shell.qml").write_text("fixture")
        events = []

        def run(args, **kwargs):
            if args == ["omarchy", "shell", "lock", "status"]:
                events.append("check-lock")
                return subprocess.CompletedProcess(args, 0, json.dumps({"locked": locked}), "")
            if args == ["systemctl", "--user", "show-environment"]:
                return subprocess.CompletedProcess(args, 0, "OMARCHY_PATH=" + str(path), "")
            if args[:2] == ["quickshell", "kill"]:
                assert args[2:] == ["-p", str(path / "shell"), "--any-display"]
                events.append("stop")
            elif args == ["omarchy", "restart", "shell"]:
                events.append("restart")
            else:
                raise AssertionError(args)
            return subprocess.CompletedProcess(args, 0, "", "")

        def install(destination, replace, before_replace):
            assert replace
            events.append("validated-and-staged")
            before_replace()
            events.append("replace")
            if fail_install:
                raise RuntimeError("fixture rename failure")

        with patch.object(installer.subprocess, "run", side_effect=run), \
                patch.object(installer, "install_locked", side_effect=install):
            if locked or fail_install:
                try:
                    installer.replace_enabled(path / "plugin")
                    raise AssertionError("expected refusal/failure")
                except RuntimeError:
                    pass
            else:
                installer.replace_enabled(path / "plugin")
        assert events == (["validated-and-staged", "check-lock"] if locked else
                          ["validated-and-staged", "check-lock", "stop", "replace", "restart"])


test_cold_replace()
test_cold_replace(locked=True)
test_cold_replace(fail_install=True)
print("PASS: enabled updates validate/stage first, refuse locked sessions, cold-replace, and restart even on failure")


class SnapshotRecoveryTests(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory(prefix="omabit-snapshot-recovery-")
        self.addCleanup(self.temporary.cleanup)
        self.root = pathlib.Path(self.temporary.name)
        self.source = self.root / "source"
        (self.source / "plugins/omabit").mkdir(parents=True)
        (self.source / "plugins/omabit/Model.js").write_text("old generation\n")
        (self.source / "build").mkdir()
        for name in ("omabit", "urbit-collector", "urbit-runner"):
            (self.source / "build" / name).write_text("inert fixture; never executed\n")
        self.destination = self.root / "plugin"
        self.addCleanup(patch.stopall)
        patch.object(installer, "ROOT", self.source).start()
        patch("sys.stdout", new=io.StringIO()).start()

    def prepare(self, destination=None):
        destination = destination or self.destination
        (self.source / "plugins/omabit/Model.js").write_text("old generation\n")
        installer.install(destination)
        old = installer.identity(destination)
        (self.source / "plugins/omabit/Model.js").write_text("new generation\n")
        return old

    def assert_retained(self, destination, old):
        trees = [destination, *destination.parent.glob("." + destination.name + ".omabit-recovery-*/new"),
                 *destination.parent.glob("." + destination.name + ".omabit-recovery-*/previous")]
        retained = [p for p in trees if installer.identity(p) == old]
        self.assertEqual(len(retained), 1)
        self.assertEqual((retained[0] / "Model.js").read_text(), "old generation\n")

    def generation(self, name):
        (self.source / "plugins/omabit/Model.js").write_text(name)
        for command in ("omabit", "urbit-collector", "urbit-runner"):
            (self.source / "build" / command).write_text("inert " + command + " from " + name)

    def assert_generation(self, destination, name):
        self.assertEqual((destination / "Model.js").read_text(), name)
        for command in ("omabit", "urbit-collector", "urbit-runner"):
            self.assertEqual((destination / "bin" / command).read_text(), "inert " + command + " from " + name)

    def interrupt_install(self, destination, boundary, interruption, occurrence=1):
        def child():
            count = 0

            def interrupt(name):
                nonlocal count
                if name == boundary:
                    count += 1
                    if count == occurrence:
                        os.kill(os.getpid(), interruption)

            with patch.object(installer, "checkpoint", interrupt), contextlib.redirect_stderr(io.StringIO()):
                try:
                    installer.install(destination, replace=True)
                except KeyboardInterrupt:
                    os._exit(130)

        process = multiprocessing.get_context("fork").Process(target=child)
        process.start()
        process.join(10)
        if process.is_alive():
            process.kill()
            process.join()
            self.fail("generation transition fixture hung")
        self.assertEqual(process.exitcode, 130 if interruption == signal.SIGINT else -signal.SIGKILL)

    def test_recovery_fulfils_requested_c_after_interrupted_a_to_b(self):
        boundaries = ("journal-linked", "journal-synced", "before-publication", "published",
                      "publication-synced", "retired-checked", "retired-renamed", "retired-synced",
                      "completion-synced", "journal-unlinked", "complete")
        for index, boundary in enumerate(boundaries):
            for interruption in (signal.SIGINT, signal.SIGKILL):
                with self.subTest(boundary=boundary, signal=interruption):
                    destination = self.root / f"abc-{index}-{interruption}"
                    self.generation("A")
                    installer.install(destination)
                    self.generation("B")
                    self.interrupt_install(destination, boundary, interruption)
                    self.generation("C")
                    installer.install(destination, replace=True)
                    self.assert_generation(destination, "C")
                    retired = list(self.root.glob("." + destination.name + ".omabit-recovery-*/previous"))
                    self.assertEqual(sorted((tree / "Model.js").read_text() for tree in retired), ["A", "B"])
                    for tree in retired:
                        self.assert_generation(tree, (tree / "Model.js").read_text())
        # An interrupted first install can also resume with a newer package,
        # without requiring a flag the original first-install command did not use.
        destination = self.root / "first-b-to-c"
        self.generation("B")
        self.interrupt_install(destination, "journal-synced", signal.SIGINT)
        self.assertFalse(destination.exists())
        self.generation("C")
        installer.install(destination)
        self.assert_generation(destination, "C")

    def test_recovery_and_requested_update_can_be_interrupted_again(self):
        for index, (boundary, occurrence) in enumerate((("recovered", 1), ("journal-synced", 1),
                                                       ("published", 2), ("journal-unlinked", 2), ("complete", 2))):
            for interruption in (signal.SIGINT, signal.SIGKILL):
                destination = self.root / f"abc-retry-{index}-{interruption}"
                self.generation("A")
                installer.install(destination)
                self.generation("B")
                self.interrupt_install(destination, "journal-synced", signal.SIGINT)
                self.generation("C")
                self.interrupt_install(destination, boundary, interruption, occurrence)
                installer.install(destination, replace=True)
                self.assert_generation(destination, "C")
                retired = list(self.root.glob("." + destination.name + ".omabit-recovery-*/previous"))
                self.assertIn("A", [(tree / "Model.js").read_text() for tree in retired])
                self.assertIn("B", [(tree / "Model.js").read_text() for tree in retired])

    def test_modified_b_is_preserved_during_recovery_and_requested_c(self):
        self.generation("A")
        installer.install(self.destination)
        self.generation("B")
        self.interrupt_install(self.destination, "published", signal.SIGINT)
        (self.destination / "Model.js").write_text("modified active B")
        self.generation("C")
        output = io.StringIO()
        with contextlib.redirect_stdout(output), self.assertRaisesRegex(RuntimeError, "local changes"):
            installer.install(self.destination, replace=True)
        self.assertNotIn("Installed Omabit", output.getvalue(), "refused C must not report success")
        self.assertEqual((self.destination / "Model.js").read_text(), "modified active B")

        # A B editor writes through the exchange to C, then the installer dies.
        # Recovery must preserve this actual retired B, including unrelated data.
        destination = self.root / "edited-b-backup"
        self.generation("A")
        installer.install(destination)
        self.generation("B")
        self.interrupt_install(destination, "journal-synced", signal.SIGINT)
        self.generation("C")
        self.interrupt_install(destination, "published", signal.SIGKILL, occurrence=2)
        journal = self.root / ("." + destination.name + ".omabit-transaction.json")
        plan = json.loads(journal.read_text())
        retired_b = self.root / plan["transaction"] / "new"
        self.assert_generation(retired_b, "B")
        with (retired_b / "Model.js").open("r+") as editor:
            editor.write("modified retired B")
            editor.truncate()
            editor.flush()
            (retired_b / "unrelated").write_text("operator data")
            installer.install(destination, replace=True)
            # The descriptor remains useful after recovery renames the backup.
            editor.write("; later editor write")
            editor.flush()
        backup = retired_b.with_name("previous")
        self.assert_generation(destination, "C")
        self.assertEqual((backup / "Model.js").read_text(), "modified retired B; later editor write")
        self.assertEqual((backup / "unrelated").read_text(), "operator data")
        self.assertTrue(json.loads((backup.parent / "completed.json").read_text())["retired_changes_detected"])
        installer.install(destination, replace=True)
        self.assertEqual((backup / "unrelated").read_text(), "operator data")

    def test_sigint_and_process_death_at_every_boundary_resume(self):
        checkpoints = []
        self.prepare()
        with patch.object(installer, "checkpoint", checkpoints.append):
            installer.install(self.destination, replace=True)
        self.assertIn("published", checkpoints)
        for replacement in (False, True):
            for number, boundary in enumerate(checkpoints):
                if not replacement and boundary.startswith("retired-"):
                    continue
                for interruption in (signal.SIGINT, signal.SIGKILL):
                    with self.subTest(replacement=replacement, boundary=boundary, signal=interruption):
                        destination = self.root / f"fixture-{replacement}-{number}-{interruption}"
                        old = self.prepare(destination) if replacement else None

                        def child():
                            def interrupt(name):
                                if name == boundary:
                                    os.kill(os.getpid(), interruption)
                            with patch.object(installer, "checkpoint", interrupt), contextlib.redirect_stderr(io.StringIO()):
                                try:
                                    installer.install(destination, replace=replacement)
                                except KeyboardInterrupt:
                                    os._exit(130)

                        process = multiprocessing.get_context("fork").Process(target=child)
                        process.start()
                        process.join(10)
                        if process.is_alive():
                            process.kill()
                            process.join()
                            self.fail("interruption fixture hung")
                        self.assertEqual(process.exitcode, 130 if interruption == signal.SIGINT else -signal.SIGKILL)
                        if old:
                            self.assertTrue(destination.is_dir(), "atomic exchange must not leave a gap")
                            self.assert_retained(destination, old)
                        # Ordinary retry recovers the recorded generation and
                        # fulfils the current source package's installation.
                        installer.install(destination, replace=destination.exists())
                        self.assertEqual((destination / "Model.js").read_text(), "new generation\n")
                        if old:
                            self.assert_retained(destination, old)

    def test_fsync_failures_and_rename_failures_preserve_both_generations(self):
        self.prepare()
        real_sync = os.fsync
        with patch.object(installer.os, "fsync", wraps=real_sync) as calls:
            installer.install(self.destination, replace=True)
        count = calls.call_count
        for failure in range(count):
            destination = self.root / f"sync-{failure}"
            old = self.prepare(destination)
            calls = 0

            def fail_once(fd):
                nonlocal calls
                calls += 1
                if calls == failure + 1:
                    raise OSError("injected fsync failure")
                return real_sync(fd)

            with patch.object(installer.os, "fsync", fail_once), self.assertRaises(OSError):
                installer.install(destination, replace=True)
            self.assert_retained(destination, old)
            installer.install(destination, replace=True)
            self.assert_retained(destination, old)
        for boundary in ("before-publication", "retired-checked"):
            destination = self.root / boundary
            old = self.prepare(destination)
            real_rename = installer.rename_atomic
            fail = False

            def mark(name):
                nonlocal fail
                fail = name == boundary

            def rename(*args, **kwargs):
                if fail:
                    raise OSError("injected rename failure")
                return real_rename(*args, **kwargs)

            with patch.object(installer, "checkpoint", mark), patch.object(installer, "rename_atomic", rename), self.assertRaises(OSError):
                installer.install(destination, replace=True)
            self.assert_retained(destination, old)
            installer.install(destination, replace=True)
            self.assert_retained(destination, old)

    def test_edit_during_quiescence_refused_and_edits_after_exchange_retained(self):
        self.prepare()
        edit = "operator customization\n"
        with self.assertRaises(RuntimeError):
            installer.install(self.destination, replace=True,
                              before_replace=lambda: (self.destination / "Model.js").write_text(edit))
        self.assertEqual((self.destination / "Model.js").read_text(), edit)
        self.assertFalse((self.root / ".plugin.omabit-transaction.json").exists())
        for boundary in ("before-publication", "published", "retired-checked", "retired-renamed", "complete"):
            destination = self.root / boundary
            self.prepare(destination)
            # Keep a real editor fd open across atomic exchange and the final
            # validation. This is why even apparently clean backups are retained.
            with (destination / "Model.js").open("r+") as editor:
                def mutate(name):
                    if name == boundary:
                        editor.seek(0)
                        editor.write(edit)
                        editor.truncate()
                        editor.flush()
                with patch.object(installer, "checkpoint", mutate):
                    installer.install(destination, replace=True)
            backup = next(destination.parent.glob("." + destination.name + ".omabit-recovery-*/previous"))
            self.assertEqual((backup / "Model.js").read_text(), edit)
            self.assertEqual((destination / "Model.js").read_text(), "new generation\n")
            installer.install(destination, replace=True)
            self.assertEqual((backup / "Model.js").read_text(), edit, "later updates must not prune edited backups")
            if boundary in ("before-publication", "published"):
                note = json.loads((backup.parent / "completed.json").read_text())
                self.assertTrue(note["retired_changes_detected"])

    def test_unrelated_entries_and_recovery_substitution_are_preserved(self):
        for kind in ("empty-directory", "symlink", "fifo", "extra-marker"):
            destination = self.root / kind
            self.prepare(destination)
            extra = destination / "unrelated"
            if kind == "empty-directory":
                extra.mkdir()
            elif kind == "symlink":
                extra.symlink_to(self.source, target_is_directory=True)
            elif kind == "fifo":
                os.mkfifo(extra)
            else:
                extra.mkdir()
                (extra / installer.MARKER).write_text("unrelated data")
            with self.assertRaises(RuntimeError):
                installer.install(destination, replace=True)
            self.assertTrue(extra.exists())
        self.prepare()
        def stop(name):
            if name == "journal-synced":
                raise KeyboardInterrupt()
        with patch.object(installer, "checkpoint", stop), self.assertRaises(KeyboardInterrupt):
            installer.install(self.destination, replace=True)
        moved = self.root / "moved-by-operator"
        self.destination.rename(moved)
        self.destination.mkdir()
        (self.destination / "unrelated").write_text("do not replace")
        with self.assertRaises(RuntimeError):
            installer.install(self.destination, replace=True)
        self.assertEqual((self.destination / "unrelated").read_text(), "do not replace")
        self.assertEqual((moved / "Model.js").read_text(), "old generation\n")

    def test_actual_retired_tree_is_checked_after_editor_substitution(self):
        self.prepare()
        moved = self.root / "original-moved-by-editor"

        def substitute(name):
            if name == "before-publication":
                self.destination.rename(moved)
                self.destination.mkdir()
                (self.destination / "unrelated-data").write_text("preserve replacement tree")

        with patch.object(installer, "checkpoint", substitute):
            installer.install(self.destination, replace=True)
        backup = next(self.root.glob(".plugin.omabit-recovery-*/previous"))
        self.assertEqual((backup / "unrelated-data").read_text(), "preserve replacement tree")
        self.assertEqual((moved / "Model.js").read_text(), "old generation\n")
        self.assertTrue(json.loads((backup.parent / "completed.json").read_text())["retired_changes_detected"])

    def test_destination_lock_serializes_installers_but_not_other_destinations(self):
        self.prepare()
        context = multiprocessing.get_context("fork")
        entered, release, second_entered = context.Event(), context.Event(), context.Event()

        def first():
            def hold(name):
                if name == "before-publication":
                    entered.set()
                    if not release.wait(10):
                        raise RuntimeError("fixture timeout")
            with patch.object(installer, "checkpoint", hold):
                installer.install(self.destination, replace=True)

        def second():
            with patch.object(installer, "checkpoint", lambda _: second_entered.set()):
                installer.install(self.destination, replace=True)

        processes = [context.Process(target=first), context.Process(target=second)]
        try:
            processes[0].start()
            self.assertTrue(entered.wait(5))
            processes[1].start()
            self.assertFalse(second_entered.wait(0.2))
            installer.install(self.root / "other-destination")
            release.set()
            for process in processes:
                process.join(10)
                self.assertEqual(process.exitcode, 0)
            self.assertTrue(second_entered.is_set())
        finally:
            release.set()
            for process in processes:
                if process.is_alive():
                    process.kill()
                    process.join()

    def test_cold_update_interruption_resumes_shell_restart_under_lock(self):
        shell = self.root / "shell"
        shell.mkdir()
        (shell / "shell.qml").write_text("inert shell fixture")
        stopped = self.root / "shell-stopped"
        calls = self.root / "shell-calls"

        def run(args, **kwargs):
            with calls.open("a") as stream:
                stream.write(" ".join(args) + "\n")
            if args == ["omarchy", "shell", "lock", "status"]:
                if stopped.exists():
                    raise RuntimeError("fixture shell is stopped")
                return subprocess.CompletedProcess(args, 0, '{"locked":false}', "")
            if args == ["systemctl", "--user", "show-environment"]:
                return subprocess.CompletedProcess(args, 0, "OMARCHY_PATH=" + str(self.root), "")
            if args[:2] == ["quickshell", "kill"]:
                stopped.write_text("stopped")
            elif args == ["omarchy", "restart", "shell"]:
                # A competing updater cannot enter while restart is in progress.
                lock = destination.parent / ("." + destination.name + ".omabit-lock")
                with lock.open("r+") as stream:
                    with self.assertRaises(BlockingIOError):
                        installer.fcntl.flock(stream.fileno(), installer.fcntl.LOCK_EX | installer.fcntl.LOCK_NB)
                stopped.unlink(missing_ok=True)
            else:
                raise AssertionError(args)
            return subprocess.CompletedProcess(args, 0, "", "")

        for index, boundary in enumerate(("shell-stop-intent", "shell-stopped", "published", "journal-unlinked")):
            for interruption in (signal.SIGINT, signal.SIGKILL):
                destination = self.root / f"cold-{index}-{interruption}"
                old = self.prepare(destination)

                def child():
                    def interrupt(name):
                        if name == boundary:
                            os.kill(os.getpid(), interruption)
                    with patch.object(installer.subprocess, "run", run), patch.object(installer, "checkpoint", interrupt):
                        try:
                            installer.replace_enabled(destination)
                        except KeyboardInterrupt:
                            os._exit(130)

                process = multiprocessing.get_context("fork").Process(target=child)
                process.start()
                process.join(10)
                if process.is_alive():
                    process.kill()
                    process.join()
                    self.fail("cold update fixture hung")
                self.assertEqual(process.exitcode, 130 if interruption == signal.SIGINT else -signal.SIGKILL)
                self.assert_retained(destination, old)
                self.generation("C")
                previous_calls = len(calls.read_text().splitlines())
                restart_pending = (self.root / ("." + destination.name + ".omabit-shell-restart.json")).exists()
                with patch.object(installer.subprocess, "run", run):
                    installer.replace_enabled(destination)
                self.assertFalse(stopped.exists())
                self.assertFalse((self.root / ("." + destination.name + ".omabit-shell-restart.json")).exists())
                self.assert_generation(destination, "C")
                self.assert_retained(destination, old)
                resumed_calls = calls.read_text().splitlines()[previous_calls:]
                self.assertEqual(sum(line.startswith("quickshell kill ") for line in resumed_calls), 1)
                self.assertEqual(resumed_calls.count("omarchy shell lock status"), 1)
                self.assertEqual(resumed_calls.count("omarchy restart shell"), 1 + int(restart_pending))

        # If B is customized after recovery, refusing C must still restart the
        # shell stopped for the earlier A->B activation in this same invocation.
        destination = self.root / "cold-edited-b"
        self.prepare(destination)
        self.interrupt_install(destination, "journal-synced", signal.SIGINT)
        self.generation("C")

        def edit_recovered(name):
            if name == "recovered":
                (destination / "Model.js").write_text("operator B edit")

        with patch.object(installer.subprocess, "run", run), patch.object(installer, "checkpoint", edit_recovered):
            with self.assertRaisesRegex(RuntimeError, "local changes"):
                installer.replace_enabled(destination)
        self.assertFalse(stopped.exists())
        self.assertFalse((self.root / ".cold-edited-b.omabit-shell-restart.json").exists())
        self.assertEqual((destination / "Model.js").read_text(), "operator B edit")


if __name__ == "__main__":
    unittest.main(verbosity=2)
