#!/usr/bin/env python3
"""Verify snapshot replacement refuses local changes and unrelated files."""
import contextlib
import importlib.util
import io
import json
import pathlib
import subprocess
import tempfile
from unittest.mock import patch

spec = importlib.util.spec_from_file_location("panel_install", pathlib.Path(__file__).with_name("install-panel.py"))
installer = importlib.util.module_from_spec(spec)
spec.loader.exec_module(installer)

with tempfile.TemporaryDirectory(prefix="omabit-install-test-") as directory:
    destination = pathlib.Path(directory) / "plugin"
    with contextlib.redirect_stdout(io.StringIO()):
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
                patch.object(installer, "install", side_effect=install):
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
