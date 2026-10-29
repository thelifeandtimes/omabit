#!/usr/bin/env python3
"""Release/upgrade admission regressions; no remote or runtime mutations."""
import copy
import hashlib
import importlib.util
import json
from pathlib import Path
import unittest

from release_policy import (backend_path, validate_master_promotion,
                            validate_publication, validate_distribution_changes,
                            validate_upgrade_paths)


class ReleasePolicyTests(unittest.TestCase):
    def test_stable_patch_minor_major_and_prerelease_promotions(self):
        for previous, target in (("0.3.0", "0.3.1"), ("0.3.1", "0.3.2"),
                                 ("0.3.2", "0.4.0"), ("0.9.0", "1.0.0"),
                                 ("0.3.1-rc.2", "0.3.1"), ("0.4.0-edge.5", "0.4.0")):
            with self.subTest(previous=previous, target=target):
                validate_master_promotion(previous, target, "release/" + target)
                validate_publication(target, previous, True, "stable")

    def test_master_rejects_prerelease_same_version_downgrade_and_topic(self):
        for target, branch in (("0.3.2-edge.1", "release/0.3.2-edge.1"),
                               ("0.3.2-rc.1", "release/0.3.2-rc.1"),
                               ("0.3.1", "release/0.3.1"), ("0.3.0", "release/0.3.0"),
                               ("0.3.2", "codex/topic")):
            with self.subTest(target=target), self.assertRaises(ValueError):
                validate_master_promotion("0.3.1", target, branch)

    def test_backend_changes_are_qualified_not_forced_to_minor(self):
        for changed in (False, True):
            validate_publication("0.3.1", "0.3.0", changed, "stable")
            validate_publication("0.3.1-edge.2", "0.3.1-edge.1", changed, "edge")

    def test_local_reproduction_is_allowed_but_not_same_version_master(self):
        validate_publication("0.3.1", "0.3.1", True, "stable")
        validate_publication("0.3.1", "0.3.1", False, "stable")
        with self.assertRaises(ValueError):
            validate_master_promotion("0.3.1", "0.3.1", "release/0.3.1")

    def test_packaging_rejects_version_and_prerelease_regression(self):
        for previous, target in (("0.3.1", "0.3.0"), ("0.3.1", "0.3.1-rc.1"),
                                 ("0.3.1-rc.2", "0.3.1-rc.1"),
                                 ("0.3.1-edge.2", "0.3.1-edge.1")):
            with self.subTest(target=target), self.assertRaises(ValueError):
                validate_publication(target, previous, False, "edge" if "edge" in target else "stable")

    def test_channel_boundaries_still_apply(self):
        for target, channel in (("0.3.1", "edge"), ("0.3.1-edge.1", "stable"),
                                ("0.3.1", "develop"), ("0.3.1", "unknown")):
            with self.subTest(channel=channel), self.assertRaises(ValueError):
                validate_publication(target, "0.3.0", True, channel)

    def test_backend_identity_inventory_is_preserved(self):
        for path in ("backend/cmd/urbit-runner/main.go", "internal/hostinstall/owner.go",
                     "deploy/systemd/runner.service", "go.mod", "go.sum",
                     "scripts/package-release.py", "scripts/release_policy.py"):
            self.assertTrue(backend_path(path), path)
        self.assertFalse(backend_path("plugins/omabit/Panel.qml"))

    def test_only_named_governance_and_documentation_can_follow_freeze(self):
        validate_distribution_changes(["docs/releases/0.3.1/qualification.json", "README.md",
            "native/bootstrap/release.json", "AGENTS.md", "CONTRIBUTING.md",
            "scripts/release_policy.py", "scripts/test-release-policy.py", "scripts/check-release-policy.py"])
        for path in ("internal/onboarding/upgrade.go", "plugins/omabit/Service.qml",
                     "native/bootstrap/runtime.py", "scripts/package-release.py",
                     ".github/workflows/checks.yml", "scripts/test-release-source.py",
                     "skills/omabit-operator/SKILL.md", "scripts/release_policy.py/extra"):
            with self.subTest(path=path), self.assertRaises(ValueError):
                validate_distribution_changes([path])

    def upgrade_fixture(self):
        self.source = "a" * 40
        self.evidence = b'{"passed":true,"source_commit":"' + self.source.encode() + b'"}'
        return {"paths": [{"versions": ["0.3.0", "0.3.1"],
                           "evidence": "docs/releases/0.3.1/result.json",
                           "evidence_sha256": hashlib.sha256(self.evidence).hexdigest()}]}

    def validate(self, value, evidence=None):
        validate_upgrade_paths(value, "0.3.0", "0.3.1", self.source,
                               lambda path: self.evidence if evidence is None else evidence)

    def test_direct_and_documented_multistep_upgrade_paths(self):
        value = self.upgrade_fixture()
        self.validate(value)
        value["paths"][0]["versions"] = ["0.2.0", "0.3.0", "0.3.1"]
        self.validate(value)

    def test_missing_previous_stable_path_refuses(self):
        value = self.upgrade_fixture()
        value["paths"][0]["versions"] = ["0.2.0", "0.3.1"]
        with self.assertRaisesRegex(ValueError, "previous stable"):
            self.validate(value)

    def test_invalid_path_versions_and_bounds_refuse(self):
        for versions in ([], ["0.3.1"], ["0.3.0", "0.4.0"],
                         ["0.3.0", "0.2.0", "0.3.1"], ["0.3.0", "0.3.0", "0.3.1"],
                         ["0.3.0-rc.1", "0.3.1"], ["0.3.0"] * 17):
            value=self.upgrade_fixture();value["paths"][0]["versions"]=versions
            with self.subTest(versions=versions), self.assertRaises(ValueError):
                self.validate(value)
        for paths in (None, [], {}, [None], [{}]*65):
            self.upgrade_fixture()
            with self.assertRaises(ValueError):
                self.validate({"paths":paths})

    def test_changed_or_unbound_evidence_refuses(self):
        for data in (b'{}', b'{"passed":false}', b'{"passed":true,"source_commit":"wrong"}', b'null'):
            value=self.upgrade_fixture()
            value["paths"][0]["evidence_sha256"]=hashlib.sha256(data).hexdigest()
            with self.assertRaises(ValueError): self.validate(value,data)
        value=self.upgrade_fixture()
        with self.assertRaisesRegex(ValueError, "recorded hash"):
            self.validate(value,self.evidence+b' ')
        data=b' '*131073;value["paths"][0]["evidence_sha256"]=hashlib.sha256(data).hexdigest()
        with self.assertRaises(ValueError): self.validate(value,data)

    def test_evidence_paths_cannot_escape_release_documents(self):
        for path in ("/tmp/result.json", "docs/releases/../result.json", "docs/releases//result.json",
                     "scripts/result.json", "docs/releases/run.py", "docs/releases/./result.json"):
            value=self.upgrade_fixture();value["paths"][0]["evidence"]=path
            with self.subTest(path=path), self.assertRaises(ValueError):self.validate(value)

    def test_actual_031_rehearsal_evidence_matches_both_paths(self):
        root=Path(__file__).resolve().parents[1]
        q=json.loads((root/'docs/releases/0.3.1/qualification.json').read_text())
        for previous in ('0.2.0','0.3.0'):
            validate_upgrade_paths(q['stable_upgrade'],previous,'0.3.1',q['source_commit'],lambda path:(root/path).read_bytes())
        damaged=copy.deepcopy(q['stable_upgrade']);damaged['paths'][0]['evidence_sha256']='0'*64
        with self.assertRaises(ValueError):
            validate_upgrade_paths(damaged,'0.3.0','0.3.1',q['source_commit'],lambda path:(root/path).read_bytes())

    def test_runtime_accepts_only_pinned_version_syntax(self):
        root=Path(__file__).resolve().parents[1]
        spec=importlib.util.spec_from_file_location('runtime',root/'native/bootstrap/runtime.py')
        runtime=importlib.util.module_from_spec(spec);spec.loader.exec_module(runtime)
        pins=json.loads((root/'native/bootstrap/release.json').read_text())
        for value in ('0.3.1','0.3.2-edge.1'):runtime.validate_manifest(dict(pins,version=value))
        for value in ('latest','0.3.1-edge','0.3.1/../../other'):
            with self.assertRaises(runtime.BootstrapError):runtime.validate_manifest(dict(pins,version=value))


if __name__ == '__main__':
    unittest.main(verbosity=2)
