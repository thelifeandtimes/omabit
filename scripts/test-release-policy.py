#!/usr/bin/env python3
"""Versioning guard regressions; no repository or release mutations."""
import importlib.util
from pathlib import Path
import unittest

from release_policy import backend_path, validate_master_promotion, validate_publication


class ReleasePolicyTests(unittest.TestCase):
    def test_master_rejects_patch_prerelease_same_version_and_topic_merges(self):
        for target, branch in (("0.1.1", "release/0.1.1"), ("0.2.0-edge.1", "release/0.2.0-edge.1"),
                               ("0.1.0", "release/0.1.0"), ("0.2.0", "fix/tooltip")):
            with self.subTest(target=target, branch=branch), self.assertRaises(ValueError):
                validate_master_promotion("0.1.0", target, branch)
        validate_master_promotion("0.1.0", "0.2.0", "release/0.2.0")
        validate_master_promotion("0.9.0", "1.0.0", "release/1.0.0")
        validate_master_promotion("0.1.0-rc.15", "0.1.0", "release/0.1.0")

    def test_backend_changes_cannot_be_hidden_in_ui_patch_lines(self):
        for previous, target in (("0.1.0", "0.1.1-edge.1"), ("0.1.1-edge.1", "0.1.1-edge.2"), ("0.1.0", "0.1.0")):
            with self.subTest(target=target), self.assertRaises(ValueError):
                validate_publication(target, previous, True, "edge" if "edge" in target else "stable")
        validate_publication("0.1.1-edge.1", "0.1.0", False, "edge")
        validate_publication("0.2.0-edge.1", "0.1.0", True, "edge")
        validate_publication("0.2.0-edge.2", "0.2.0-edge.1", True, "edge")
        validate_publication("0.2.0", "0.2.0-edge.2", True, "stable")

    def test_explicit_guided_upgrade_exception_is_narrow(self):
        validate_publication("0.3.1", "0.3.0", True, "stable")
        validate_master_promotion("0.3.0", "0.3.1", "release/0.3.1")
        for previous, target in (("0.2.0", "0.3.1"), ("0.3.1", "0.3.2"), ("0.3.0", "0.3.2")):
            with self.assertRaises(ValueError):
                validate_publication(target, previous, True, "stable")
            with self.assertRaises(ValueError):
                validate_master_promotion(previous, target, "release/" + target)
        with self.assertRaises(ValueError):
            validate_master_promotion("0.3.0", "0.3.1", "codex/topic")

    def test_channels_and_stable_patch_zero_are_enforced(self):
        for target, channel in (("0.1.1", "stable"), ("0.2.0", "edge"), ("0.2.0-edge.1", "stable"), ("0.2.0-edge.1", "develop")):
            with self.subTest(target=target, channel=channel), self.assertRaises(ValueError):
                validate_publication(target, "0.1.0", False, channel)

    def test_shared_runtime_installer_and_dependencies_are_backend_changes(self):
        for path in ("backend/cmd/urbit-runner/main.go", "backend/cmd/urbit-collector/main.go", "internal/protocol/hello.go",
                     "internal/hostinstall/owner.go", "internal/onboarding/setup.go", "deploy/systemd/runner.service", "go.mod", "go.sum", "scripts/install-services.sh"):
            self.assertTrue(backend_path(path), path)
        self.assertFalse(backend_path("plugins/omabit/Panel.qml"))

    def test_runtime_accepts_only_pinned_edge_prerelease_syntax(self):
        root = Path(__file__).resolve().parents[1]
        spec = importlib.util.spec_from_file_location("edge_runtime", root / "native/bootstrap/runtime.py")
        runtime = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(runtime)
        import json
        pins = json.loads((root / "native/bootstrap/release.json").read_text())
        runtime.validate_manifest(dict(pins, version="0.1.1-edge.1"))
        for bad in ("edge", "latest", "0.1.1-edge", "0.1.1-edge.1/../../other"):
            with self.assertRaises(runtime.BootstrapError):
                runtime.validate_manifest(dict(pins, version=bad))


if __name__ == "__main__":
    unittest.main(verbosity=2)
