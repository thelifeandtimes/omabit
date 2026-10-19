#!/usr/bin/env python3
"""Guest admission and scenario preservation checks, without a VM or live writes."""
import argparse
import importlib.util
import json
from pathlib import Path
import subprocess
import sys
import unittest
from unittest.mock import patch

spec = importlib.util.spec_from_file_location("desktop_scenario", Path(__file__).with_name("qualify.py"))
scenario = importlib.util.module_from_spec(spec)
spec.loader.exec_module(scenario)


class GuardTests(unittest.TestCase):
    def args(self, **values):
        return argparse.Namespace(**dict({"acknowledge_disposable_guest": False, "with_host": False, "output": Path("/unused")}, **values))

    def test_no_ack_has_no_process_or_filesystem_effects(self):
        with patch.object(scenario, "checked", side_effect=AssertionError("probe before acknowledgement")):
            with self.assertRaisesRegex(ValueError, "acknowledge"):
                scenario.admit(self.args())

    def test_root_is_refused_before_probing(self):
        with patch.object(scenario.os, "geteuid", return_value=0), patch.object(scenario, "checked", side_effect=AssertionError("probe as root")):
            with self.assertRaisesRegex(ValueError, "desktop owner"):
                scenario.admit(self.args(acknowledge_disposable_guest=True))

    def test_bare_metal_and_containers_never_reach_marker_or_mutation(self):
        with patch.object(scenario.os, "geteuid", return_value=1000), patch.object(scenario, "checked", return_value=""):
            with self.assertRaisesRegex(ValueError, "full-system"):
                scenario.admit(self.args(acknowledge_disposable_guest=True))

    def test_default_plan_is_inert_and_identifies_the_real_stock_command(self):
        result = subprocess.run([sys.executable, "-B", str(Path(__file__).with_name("qualify.py"))], capture_output=True, text=True)
        self.assertEqual(result.returncode, 0)
        plan = json.loads(result.stdout)
        self.assertIn("stock omarchy plugin update", " ".join(plan["steps"]))
        self.assertFalse(plan["with_host"])

    def test_preservation_projection_ignores_samples_not_identity_or_history(self):
        snapshot = {"runner_id": "runner-test", "mode": "development-managed", "instances": [{"id": "one", "ship": "~zod", "config_revision": 2,
                    "config": {"http_port": 18080}, "desired_state": "running", "resources": {"memory": 1}}], "operations": [{"id": "receipt"}]}
        before = scenario.projection(snapshot)
        snapshot["instances"][0]["resources"]["memory"] = 2
        self.assertEqual(before, scenario.projection(snapshot))
        snapshot["instances"][0]["desired_state"] = "stopped"
        self.assertNotEqual(before, scenario.projection(snapshot))


if __name__ == "__main__":
    unittest.main(verbosity=2)
