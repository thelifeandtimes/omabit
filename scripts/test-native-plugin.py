#!/usr/bin/env python3
"""Isolated tests for native-plugin binary preparation and release generation."""
import concurrent.futures
import hashlib
import importlib.util
import io
import json
import os
from pathlib import Path
import tarfile
import tempfile
import time
import unittest
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[1]


def module(name, path):
    spec = importlib.util.spec_from_file_location(name, path)
    value = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(value)
    return value


runtime = module("native_runtime", ROOT / "native/bootstrap/runtime.py")
packager = module("native_packager", ROOT / "scripts/package-plugin.py")


class NativePluginTests(unittest.TestCase):
    def test_canonical_manifest_launchers_and_release_pins_match(self):
        root = json.loads((ROOT / "manifest.json").read_text())
        component = json.loads((ROOT / "plugins/omabit/manifest.json").read_text())
        entries = root.pop("entryPoints")
        component.pop("entryPoints")
        self.assertEqual(root, component)
        self.assertIn('version:'+json.dumps(root["version"]), (ROOT / "plugins/omabit/Service.qml").read_text(),
                      "loaded-QML status version must identify the release being installed")
        for name in ("MonitorWindow.qml", "SetupPane.qml", "Panel.qml"):
            self.assertIn(root["version"], (ROOT / "plugins/omabit" / name).read_text(),
                          "visible UI version must match the release manifest: " + name)
        self.assertEqual(entries, {"service": "native/Service.qml", "barWidget": "native/Panel.qml"})
        for name in entries.values():
            self.assertTrue((ROOT / name).is_file())
        pins = json.loads((ROOT / "native/bootstrap/release.json").read_text())
        runtime.validate_manifest(pins)
        # Tool pins refer to an already published build. They are advanced after
        # release, independently of the next UI version being prepared in Git.
        template = (ROOT / "native/bootstrap/launch.py").read_bytes()
        for name in runtime.COMMANDS:
            path = ROOT / "native/bin" / name
            self.assertEqual(path.read_bytes(), template)
            self.assertEqual(path.stat().st_mode & 0o777, 0o755)

    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory(prefix="omabit-native-test-")
        self.addCleanup(self.temporary.cleanup)
        self.root = Path(self.temporary.name)
        self.content = {name: ("#!/bin/sh\n# " + name + "\nexit 0\n").encode() for name in runtime.COMMANDS}
        self.pins = {"version": "0.1.0-rc.2", "platform": "linux/amd64", "source_commit": "a" * 40,
                     "binaries": {name: {"name": name + "-linux-amd64", "bytes": len(data),
                                          "sha256": hashlib.sha256(data).hexdigest()} for name, data in self.content.items()}}

    def fake_download(self, url, target, entry):
        self.assertTrue(url.startswith(runtime.RELEASES + "v0.1.0-rc.2/"))
        target.write_bytes(self.content[target.name])
        target.chmod(0o755)

    def test_concurrent_startup_downloads_once_and_offline_cache_is_verified(self):
        calls = []

        def download(*args):
            calls.append(args[0])
            time.sleep(0.02)
            self.fake_download(*args)

        with patch.object(runtime, "download", download):
            with concurrent.futures.ThreadPoolExecutor(max_workers=5) as executor:
                paths = list(executor.map(lambda _: runtime.ensure_binaries(self.pins, self.root), range(5)))
        self.assertEqual(len(calls), 3)
        self.assertEqual(len(set(paths)), 1)
        self.assertEqual({p.name for p in paths[0].iterdir()}, set(runtime.COMMANDS))
        with patch.object(runtime, "download", side_effect=AssertionError("offline cache downloaded")):
            self.assertEqual(runtime.ensure_binaries(self.pins, self.root), paths[0])
            (paths[0] / "urbit-runner").write_bytes(b"tampered")
            with self.assertRaises(runtime.BootstrapError):
                runtime.ensure_binaries(self.pins, self.root)

    def test_partial_download_is_not_published_and_can_be_retried(self):
        def broken(url, target, entry):
            target.write_bytes(b"partial")
            raise OSError("network interrupted")

        with patch.object(runtime, "download", broken), self.assertRaises(OSError):
            runtime.ensure_binaries(self.pins, self.root)
        root = self.root / "omabit/native-binaries"
        self.assertEqual([p.name for p in root.iterdir()], ["download.lock"])
        with patch.object(runtime, "download", self.fake_download):
            runtime.verify_cache(runtime.ensure_binaries(self.pins, self.root), self.pins)

    def test_symlinked_cache_and_lock_are_refused(self):
        link = self.root / "redirect"
        link.symlink_to(self.root, target_is_directory=True)
        with self.assertRaises(runtime.BootstrapError):
            runtime.ensure_binaries(self.pins, link)
        root = self.root / "omabit/native-binaries"
        runtime.private_directory(root)
        (root / "download.lock").symlink_to(self.root / "other")
        with self.assertRaises(OSError):
            runtime.ensure_binaries(self.pins, self.root)
        self.assertFalse((self.root / "other").exists())

    def test_unrecognized_platform_or_manifest_cannot_download(self):
        with patch.object(runtime.platform, "machine", return_value="aarch64"), patch.object(runtime, "download") as download:
            with self.assertRaises(runtime.BootstrapError):
                runtime.ensure_binaries(self.pins, self.root)
            download.assert_not_called()
        for bad in ("../elsewhere", "0.1.0-dev", "latest"):
            with self.assertRaises(runtime.BootstrapError):
                runtime.validate_manifest(dict(self.pins, version=bad))

    def test_download_enforces_hash_length_and_https_redirect(self):
        entry = self.pins["binaries"]["omabit"]
        valid = self.content["omabit"]
        for number, content in enumerate((valid, b"x" * len(valid), valid[:-1], valid + b"extra")):
            response = io.BytesIO(content)
            response.status, response.headers = 200, {}
            target = self.root / str(number)
            with patch.object(runtime.urllib.request, "build_opener") as opener:
                opener.return_value.open.return_value = response
                if content == valid:
                    runtime.download("https://example.invalid/asset", target, entry)
                    self.assertEqual(runtime.digest(target), entry["sha256"])
                    self.assertEqual(target.stat().st_mode & 0o777, 0o755)
                else:
                    with self.assertRaises(runtime.BootstrapError):
                        runtime.download("https://example.invalid/asset", target, entry)
        with self.assertRaises(runtime.BootstrapError):
            runtime.HTTPSRedirect().redirect_request(None, None, 302, "", {}, "http://example.invalid/asset")

    def release_fixture(self):
        assets = self.root / "assets"
        assets.mkdir()
        package = self.root / "omabit-0.1.0-rc.2-linux-amd64"
        plugin = package / "plugins/omabit"
        plugin.mkdir(parents=True)
        manifest = {"schemaVersion": 1, "id": "thelifeandtimes.omabit", "name": "Omabit", "version": self.pins["version"],
                    "kinds": ["service", "bar-widget"], "entryPoints": {"service": "Service.qml", "barWidget": "Panel.qml"}}
        (plugin / "manifest.json").write_text(json.dumps(manifest))
        (plugin / "Service.qml").write_text("import QtQuick\nItem {}\n")
        (plugin / "Panel.qml").write_text("import QtQuick\nItem {}\n")
        (package / "BUILD.json").write_text(json.dumps({"version": self.pins["version"], "source_commit": self.pins["source_commit"]}))
        archive = assets / (package.name + ".tar.gz")
        with tarfile.open(archive, "w:gz") as tar:
            tar.add(package, arcname=package.name)
        for name, content in self.content.items():
            (assets / (name + "-linux-amd64")).write_bytes(content)
        entries = [{"name": p.name, "bytes": p.stat().st_size, "sha256": runtime.digest(p)} for p in assets.iterdir()]
        release = {key: self.pins[key] for key in ("version", "platform", "source_commit")}
        release["assets"] = entries
        (assets / "release.json").write_text(json.dumps(release))
        (assets / "SHA256SUMS").write_text("".join(runtime.digest(p) + "  " + p.name + "\n" for p in sorted(assets.iterdir())))
        return assets, plugin

    def test_generation_uses_frozen_ui_and_executable_launchers(self):
        assets, plugin = self.release_fixture()
        destination = self.root / "native"
        provenance = packager.generate(assets, destination, "b" * 40)
        self.assertEqual((destination / "Service.qml").read_bytes(), (plugin / "Service.qml").read_bytes())
        self.assertEqual(json.loads((destination / "bootstrap/release.json").read_text()), self.pins)
        self.assertEqual(provenance["release_source_commit"], "a" * 40)
        self.assertEqual(provenance["distribution_source_commit"], "b" * 40)
        self.assertFalse(any(p.is_symlink() for p in destination.rglob("*")))
        for name in runtime.COMMANDS:
            self.assertEqual((destination / "bin" / name).stat().st_mode & 0o777, 0o755)
        with self.assertRaises(RuntimeError):
            packager.generate(assets, destination, "b" * 40)

    def test_generation_refuses_tampered_release(self):
        assets, _ = self.release_fixture()
        (assets / "urbit-runner-linux-amd64").write_bytes(b"changed")
        with self.assertRaises(RuntimeError):
            packager.generate(assets, self.root / "native", "b" * 40)
        self.assertFalse((self.root / "native").exists())


if __name__ == "__main__":
    unittest.main(verbosity=2)
