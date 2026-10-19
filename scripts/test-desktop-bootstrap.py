#!/usr/bin/env python3
"""Unprivileged release/profile/activation tests; never opens an installed runner."""
import hashlib
import io
import json
import os
from pathlib import Path
import subprocess
import sys
import tarfile
import tempfile
import unittest
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / "native/bootstrap"))
import desktop


def bundle(directory, version="0.1.0-rc.13", *, contract=1, bad=None, qml=None, collector=None, panel=None):
    directory.mkdir(parents=True)
    prefix = "omabit-" + version + "-linux-amd64"
    files = {
        "BUILD.json": json.dumps({"version": version, "source_commit": "a" * 40, "schema": 21}).encode(),
        "plugins/omabit/manifest.json": json.dumps({"version": version, "id": "thelifeandtimes.omabit"}).encode(),
        "plugins/omabit/Service.qml": qml or b"import QtQuick\nItem {}\n",
        "plugins/omabit/Panel.qml": b'''import QtQuick
import "Model.js" as Model
Item {
  property var service; property QtObject bar; property var settings; property bool manageIpc:true
  property bool opened:false
  property string implementationVersion:Model.release()
  implicitWidth:40;implicitHeight:30
  function open(){opened=true} function close(){opened=false} function toggle(){opened=!opened}
  function closeForPopoutSwitch(){close()}
}
''',
        "plugins/omabit/Model.js": b"function release() {return '" + version.encode() + b"'}\n",
    }
    if contract:
        files["plugins/omabit/DESKTOP.json"] = json.dumps({"bootstrap_api": contract, "profile_state_schema": 1}).encode()
    for name in desktop.runtime.COMMANDS:
        files["build/" + name] = b"#!/bin/sh\n# " + name.encode() + b" " + version.encode() + b"\nexit 0\n"
    if collector:
        files["build/urbit-collector"] = collector
    if panel:
        files["plugins/omabit/Panel.qml"] = panel
    archive = directory / (prefix + ".tar.gz")
    with tarfile.open(archive, "w:gz") as stream:
        for name, content in files.items():
            member = tarfile.TarInfo(prefix + "/" + name)
            member.size = len(content)
            member.mode = 0o755 if name.startswith("build/") else 0o644
            stream.addfile(member, io.BytesIO(content))
        if bad:
            member = tarfile.TarInfo(prefix + "/" + ("plugins/omabit/Panel.qml" if bad == "duplicate" else bad))
            if bad == "link":
                member.type = tarfile.SYMTYPE
                member.linkname = "/outside"
            stream.addfile(member)
    asset = lambda name, data: {"name": name, "bytes": len(data), "sha256": hashlib.sha256(data).hexdigest()}
    metadata = {"version": version, "platform": "linux/amd64", "source_commit": "a" * 40,
                "assets": [asset(name + "-linux-amd64", files["build/" + name]) for name in desktop.runtime.COMMANDS]
                + [asset(archive.name, archive.read_bytes())]}
    release = directory / "release.json"
    release.write_text(json.dumps(metadata))
    return release, archive, desktop.descriptor_from_release(metadata)


class DesktopTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix="omabit-bootstrap-")
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.env = {"HOME": str(self.root / "home"), "XDG_CONFIG_HOME": str(self.root / "config"),
                    "XDG_CACHE_HOME": str(self.root / "cache"), "XDG_STATE_HOME": str(self.root / "state")}
        self.environment = patch.dict(os.environ, self.env)
        self.environment.start()
        self.addCleanup(self.environment.stop)
        metadata, archive, pins = bundle(self.root / "one")
        self.pins = self.root / "pins.json"
        self.pins.write_text(json.dumps(pins))
        self.desktop = desktop.Desktop(self.pins)
        self.first = self.desktop.import_release(metadata, archive)["release"]

    def test_profiles_are_isolated_and_default_paths_are_preserved(self):
        self.assertEqual(self.desktop.profile_paths("default"), {})
        a = self.desktop.create_profile("dev")
        b = self.desktop.create_profile("fresh")
        self.assertFalse(set(a.values()) & set(b.values()))
        for value in a.values():
            self.assertEqual(Path(value).stat().st_mode & 0o777, 0o700)
        self.desktop.select("dev", self.first)
        self.assertFalse(self.desktop.resolve()["host_setup"])
        self.assertEqual(self.desktop.resolve()["environment"], a)
        with self.assertRaises(ValueError):
            self.desktop.create_profile("dev")
        with self.assertRaises(ValueError):
            self.desktop.create_profile("../outside")

    def test_selection_is_not_activation_and_stale_acks_cannot_commit(self):
        plan = self.desktop.resolve()
        self.assertIsNone(plan["active"])
        with self.desktop.lease() as session:
            with self.assertRaises(ValueError):
                self.desktop.activation("commit", session, plan["request"])
            self.desktop.activation("begin", session, plan["request"])
            self.desktop.select("default", self.first)
            with self.assertRaises(ValueError):
                self.desktop.activation("commit", session, plan["request"])
            fresh = self.desktop.resolve()
            self.desktop.activation("begin", session, fresh["request"])
            self.desktop.activation("commit", session, fresh["request"])
        self.assertEqual(self.desktop.resolve()["active"], self.first)
        with self.assertRaises(ValueError):
            self.desktop.activation("commit", session, fresh["request"])

    def test_failed_upgrade_retains_previous_and_retry_has_new_identity(self):
        with self.desktop.lease() as session:
            first = self.desktop.resolve()
            self.desktop.activation("begin", session, first["request"])
            self.desktop.activation("commit", session, first["request"])
            meta, archive, _ = bundle(self.root / "two", "0.1.0-rc.14")
            second = self.desktop.import_release(meta, archive)["release"]
            self.desktop.select("default", second)
            plan = self.desktop.resolve()
            self.desktop.activation("begin", session, plan["request"])
            self.desktop.activation("fail", session, plan["request"], "broken QML")
            self.assertTrue(self.desktop.resolve()["blocked"])
            self.assertEqual(self.desktop.resolve()["active"], self.first)
            self.desktop.select("default")
            retry = self.desktop.resolve()
            self.assertNotEqual(plan["request"], retry["request"])
            self.desktop.activation("begin", session, retry["request"])
            self.desktop.activation("commit", session, retry["request"])
            self.assertEqual(self.desktop.resolve()["previous"], self.first)

    def test_interrupted_activation_retains_intent_and_requires_new_lease(self):
        plan = self.desktop.resolve()
        with self.desktop.lease() as old:
            self.desktop.activation("begin", old, plan["request"])
            with self.assertRaises(BlockingIOError), self.desktop.lease():
                pass
        self.assertIsNotNone(self.desktop.resolve()["pending"])
        with self.desktop.lease() as new:
            self.assertNotEqual(old, new)
            with self.assertRaises(ValueError):
                self.desktop.activation("commit", old, plan["request"])
            self.desktop.activation("begin", new, plan["request"])
            self.desktop.activation("commit", new, plan["request"])

    def test_lost_commit_acknowledgement_restores_exact_previous_ui(self):
        with self.desktop.lease() as session:
            first = self.desktop.resolve()
            self.desktop.activation("begin", session, first["request"])
            self.desktop.activation("commit", session, first["request"])
            meta, archive, _ = bundle(self.root / "next", "0.1.0-rc.14")
            second = self.desktop.import_release(meta, archive)["release"]
            self.desktop.select("default", second)
            next_release = self.desktop.resolve()
            self.desktop.activation("begin", session, next_release["request"])
            self.desktop.activation("commit", session, next_release["request"])
            self.desktop.activation("fail", session, next_release["request"], "commit acknowledgement lost")
            self.assertEqual(self.desktop.resolve()["active"], self.first)
            self.assertTrue(self.desktop.resolve()["blocked"])

    def test_cache_cleanup_network_failure_and_modified_inventory_fail_closed(self):
        prepared = self.desktop.prepare()
        path = Path(prepared["directory"])
        ui = path / "ui/Model.js"
        ui.chmod(0o600)
        ui.write_text("malicious replacement")
        inventory = desktop.read_json(path / "inventory.json")
        inventory["files"]["ui/Model.js"] = desktop.runtime.digest(ui)
        desktop.atomic_json(path / "inventory.json", inventory)
        with self.assertRaisesRegex(ValueError, "authenticated archive"):
            self.desktop.prepare()
        meta, archive, pins = bundle(self.root / "network", "0.1.0-rc.15")
        self.pins.write_text(json.dumps(pins))
        with patch.object(desktop.runtime, "download", side_effect=OSError("offline")):
            with self.assertRaises(OSError):
                self.desktop.prepare()
        self.assertFalse((self.desktop.cache / desktop.identity(pins)).exists())

    def test_distinct_generations_have_distinct_nested_qml_urls(self):
        first = self.desktop.prepare()
        meta, archive, _ = bundle(self.root / "two", "0.1.0-rc.14")
        second = self.desktop.import_release(meta, archive)["release"]
        self.desktop.select("default", second)
        next_release = self.desktop.prepare()
        self.assertNotEqual(first["ui"], next_release["ui"])
        self.assertTrue((Path(first["directory"]) / "ui/Model.js").exists())

    def test_legacy_default_is_observable_but_cannot_claim_profile_isolation(self):
        meta, archive, _ = bundle(self.root / "legacy", "0.1.0-rc.12", contract=0)
        key = self.desktop.import_release(meta, archive)["release"]
        self.desktop.select("default", key)
        self.assertEqual(self.desktop.prepare()["contract"], 0)
        self.desktop.create_profile("fresh")
        self.desktop.select("fresh", key)
        with self.assertRaisesRegex(ValueError, "cannot isolate"):
            self.desktop.prepare()

    def test_archive_path_link_duplicate_and_future_contract_are_rejected(self):
        for bad in ("../escape", "link", "duplicate"):
            with self.subTest(bad=bad):
                meta, archive, _ = bundle(self.root / bad.replace("/", "_"), bad=bad)
                with self.assertRaises(ValueError):
                    self.desktop.import_release(meta, archive)
        meta, archive, _ = bundle(self.root / "future", contract=2)
        with self.assertRaisesRegex(ValueError, "contract"):
            self.desktop.import_release(meta, archive)

    def test_real_cli_profile_exec_receives_private_environment(self):
        meta, archive, _ = bundle(self.root / "environment", collector=b'#!/usr/bin/env python3\nimport json,os\nprint(json.dumps({k:v for k,v in os.environ.items() if k.startswith("XDG_")}))\n')
        key = self.desktop.import_release(meta, archive)["release"]
        self.desktop.create_profile("fresh")
        self.desktop.select("fresh", key)
        self.desktop.select("default")
        result = subprocess.run([sys.executable, "-B", ROOT / "native/bootstrap/control.py", "--pins", self.pins,
                                 "exec", "--profile", "fresh", "urbit-collector", "runners"], text=True, capture_output=True)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(json.loads(result.stdout)["XDG_CONFIG_HOME"], self.desktop.profile_paths("fresh")["XDG_CONFIG_HOME"])
        self.assertEqual(self.desktop.status()["selected"], "default")
        result = subprocess.run([sys.executable, "-B", ROOT / "native/bootstrap/control.py", "--pins", self.pins,
                                 "exec", "--profile", "fresh", "omabit", "setup"], text=True, capture_output=True)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("disabled", result.stderr)

    def test_future_desktop_state_is_refused_without_rewriting_it(self):
        state = self.desktop.status()
        state.pop("paths")
        state["format"] = 2
        path = self.desktop.state / "state.json"
        desktop.atomic_json(path, state)
        original = path.read_bytes()
        with self.assertRaisesRegex(ValueError, "unsupported desktop state"):
            self.desktop.status()
        self.assertEqual(path.read_bytes(), original)

    def test_symlinked_state_and_cache_destinations_are_never_replaced(self):
        path = self.desktop.state / "state.json"
        original = path.read_bytes()
        saved = path.with_name("preserved.json")
        path.rename(saved)
        path.symlink_to(saved)
        with self.assertRaisesRegex(ValueError, "unsafe desktop state"):
            self.desktop.status()
        self.assertTrue(path.is_symlink())
        self.assertEqual(saved.read_bytes(), original)
        path.unlink()
        saved.rename(path)
        meta, archive, pins = bundle(self.root / "symlink-target", "0.1.0-rc.14")
        destination = self.desktop.cache / desktop.identity(pins)
        destination.symlink_to(self.root / "absent-target")
        with self.assertRaises(desktop.runtime.BootstrapError):
            self.desktop.import_release(meta, archive)
        self.assertTrue(destination.is_symlink())


if __name__ == "__main__":
    unittest.main(verbosity=2)
