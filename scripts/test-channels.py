#!/usr/bin/env python3
"""Explicit branch routing and preservation; no live checkout is modified."""
import importlib.util
import json
from pathlib import Path
import subprocess
import unittest

spec = importlib.util.spec_from_file_location("channels", Path(__file__).with_name("omabit-channel.py"))
channels = importlib.util.module_from_spec(spec)
spec.loader.exec_module(channels)


class ChannelTests(unittest.TestCase):
    def fixture(self, branch="edge", dirty=False, busy=False, remote_channel="edge", target_exists=False, diverged=False):
        calls = []
        def run(argv, directory=None, check=True):
            calls.append(argv)
            result = ""
            code = 0
            if argv[:3] == ["git", "status", "--porcelain=v1"]:
                result = " M user-work.qml" if dirty else ""
            elif argv[:3] == ["git", "remote", "get-url"]:
                result = "https://github.com/thelifeandtimes/omabit.git"
            elif argv[:3] == ["git", "branch", "--show-current"]:
                result = branch
            elif argv[:2] == ["git", "show"]:
                result = json.dumps({"id": channels.PLUGIN, "releaseChannel": remote_channel})
            elif argv[:2] == ["git", "show-ref"]:
                code = 0 if target_exists else 1
            elif argv[:3] == ["git", "merge-base", "--is-ancestor"]:
                code = 1 if diverged else 0
            elif argv[-1] == "listPlugins":
                result = json.dumps([{"id": channels.PLUGIN, "enabled": True}])
            elif argv[-1] == "status":
                result = json.dumps({"busy": busy, "phase": "ready"})
            elif argv[:2] == ["git", "rev-parse"]:
                result = "a" * 40
            if check and code:
                raise RuntimeError("fixture divergence")
            return subprocess.CompletedProcess(argv, code, result, "")
        return calls, run

    def test_edge_update_fetches_edge_not_remote_head(self):
        calls, run = self.fixture()
        result = channels.update(Path("/fixture"), run=run)
        self.assertEqual(result["channel"], "edge")
        self.assertIn(["git", "fetch", "--no-tags", "origin", "refs/heads/edge:refs/remotes/origin/edge"], calls)
        self.assertIn(["git", "merge", "--ff-only", "refs/remotes/origin/edge"], calls)
        self.assertFalse(any(call[:2] in (["git", "reset"], ["git", "config"]) for call in calls))

    def test_dirty_or_busy_checkout_never_fetches(self):
        for options in ({"dirty": True}, {"busy": True}):
            calls, run = self.fixture(**options)
            with self.assertRaises(ValueError):
                channels.update(Path("/fixture"), run=run)
            self.assertFalse(any(call[:2] == ["git", "fetch"] for call in calls))

    def test_channel_mismatch_and_divergence_do_not_switch(self):
        for options in ({"remote_channel": "stable"}, {"target_exists": True, "diverged": True}):
            calls, run = self.fixture(branch="master", **options)
            with self.assertRaises((ValueError, RuntimeError)):
                channels.update(Path("/fixture"), "edge", run)
            self.assertFalse(any(call[:2] == ["git", "switch"] for call in calls))

    def test_explicit_new_channel_creates_tracking_branch(self):
        calls, run = self.fixture(branch="master")
        channels.update(Path("/fixture"), "edge", run)
        self.assertIn(["git", "switch", "--create", "edge", "--track", "origin/edge"], calls)
        self.assertIn(["omarchy", "shell", "shell", "rescanPlugins"], calls)


if __name__ == "__main__":
    unittest.main(verbosity=2)
