#!/usr/bin/env python3
"""Boundary tests for the mapped qualification runner; no desktop access."""
import importlib.util
import io
import json
import os
from pathlib import Path
import tarfile
import tempfile
import unittest
from unittest.mock import patch

from desktop_fixture import environment, HARNESS_ROOT

spec = importlib.util.spec_from_file_location("qualification", Path(__file__).with_name("qualify-desktop.py"))
qualification = importlib.util.module_from_spec(spec)
spec.loader.exec_module(qualification)


class QualificationTests(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory(prefix="omabit-qualification-test-")
        self.addCleanup(self.temporary.cleanup)
        self.root = Path(self.temporary.name)

    def test_archive_selects_its_own_package_and_rejects_escape_links_and_duplicates(self):
        for bad in (None, "../outside", "/outside", "symlink", "duplicate"):
            archive_path = self.root / (str(bad).replace("/", "_") + ".tar")
            with tarfile.open(archive_path, "w") as archive:
                member = tarfile.TarInfo("frozen/plugins/omabit/Service.qml")
                content = b"frozen bytes"
                member.size = len(content)
                archive.addfile(member, io.BytesIO(content))
                if bad:
                    member = tarfile.TarInfo("frozen/link" if bad == "symlink" else
                                             "frozen/plugins/omabit/Service.qml" if bad == "duplicate" else bad)
                    if bad == "symlink":
                        member.type = tarfile.SYMTYPE
                        member.linkname = "/outside"
                    archive.addfile(member)
            destination = self.root / (archive_path.name + "-extract")
            if bad:
                with self.assertRaises(AssertionError):
                    qualification.extract(archive_path, destination)
                self.assertFalse(destination.exists())
            else:
                package = qualification.extract(archive_path, destination)
                self.assertEqual(package, destination / "frozen")
                self.assertEqual((package / "plugins/omabit/Service.qml").read_bytes(), content)

    def test_package_checksums_refuse_changed_and_extra_bytes(self):
        build = self.root / "BUILD.json"
        build.write_text(json.dumps({"version": "0.1.0-rc.12", "schema": 20}))
        (self.root / "SHA256SUMS").write_text(qualification.digest(build) + "  BUILD.json\n")
        self.assertEqual(qualification.verify_package(self.root)["schema"], 20)
        extra = self.root / "outside"
        extra.touch()
        with self.assertRaises(AssertionError):
            qualification.verify_package(self.root)
        extra.unlink()
        build.write_text("{}")
        with self.assertRaises(AssertionError):
            qualification.verify_package(self.root)

    def test_every_writable_home_is_private_but_display_and_hypr_ipc_are_retained(self):
        with patch.dict(os.environ, {"XDG_RUNTIME_DIR": "/fixture/desktop", "WAYLAND_DISPLAY": "wayland-1",
                                     "HYPRLAND_INSTANCE_SIGNATURE": "fixture-signature", "QT_QPA_PLATFORM": "offscreen"}):
            env = environment(self.root)
        for key in ("HOME", "XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_CACHE_HOME", "XDG_STATE_HOME", "XDG_RUNTIME_DIR"):
            self.assertEqual(Path(env[key]).parent, self.root)
            self.assertEqual(Path(env[key]).stat().st_mode & 0o777, 0o700)
        self.assertEqual(env["WAYLAND_DISPLAY"], "/fixture/desktop/wayland-1")
        self.assertEqual(env["QT_QPA_PLATFORM"], "wayland")
        self.assertEqual((self.root / "runtime/hypr/f").readlink(), Path("/fixture/desktop/hypr/fixture-signature"))

    def test_old_pins_cannot_certify_next_release_or_another_source(self):
        native = HARNESS_ROOT / "native"
        pins = json.loads((native / "bootstrap/release.json").read_text())
        for expected in ({"expected_version": "99.0.0"}, {"expected_commit": "0" * 40}):
            with self.assertRaises(AssertionError):
                qualification.qualify_binaries(native, self.root / "absent", self.root, **expected)
        self.assertNotEqual(pins["version"], "99.0.0")


if __name__ == "__main__":
    unittest.main(verbosity=2)
