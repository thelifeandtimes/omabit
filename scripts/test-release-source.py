#!/usr/bin/env python3
"""Release provenance fixtures: existing commit objects only; no test commits.

Run with --build for two independent full builds plus the existing bundle smoke.
The current packaging implementation operates on an isolated committed baseline;
this does NOT certify the uncommitted application/installer changes in a checkout.
"""
import argparse
import contextlib
import importlib.util
import io
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tarfile
import tempfile
import unittest
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[1]


def load(name, path):
    spec = importlib.util.spec_from_file_location(name, path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


packager = load("release_packager", ROOT / "scripts/package-release.py")
plugin_packager = load("plugin_packager", ROOT / "scripts/package-plugin.py")
BUILD = False


class SourceTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.temporary = tempfile.TemporaryDirectory(prefix="omabit-release-source-")
        cls.root = Path(cls.temporary.name)
        cls.baseline = cls.root / "baseline"
        cls.commit = packager.source_git(ROOT, "rev-parse", "HEAD").strip()
        packager.export_source(ROOT, cls.baseline, cls.commit)

    @classmethod
    def tearDownClass(cls):
        cls.temporary.cleanup()

    def hosted_toolchain(self, root):
        real_go = packager.resolve_go()
        binary = root / "opt/hostedtoolcache/go/1.25.0/x64/bin/go"
        binary.parent.mkdir(parents=True)
        calls = root / "toolchain-calls.jsonl"
        # Forward to the selected real compiler so the full build fixture proves
        # both tool selection and reproducibility, without downloading a compiler.
        binary.write_text(f'''#!{sys.executable}
import json, os, sys
assert os.environ["PATH"] == {str(binary.parent) + os.pathsep + packager.SYSTEM_PATH!r}
assert os.environ["GOENV"] == "off"
assert os.environ["GOWORK"] == "off"
assert os.environ["GOFLAGS"] == ""
assert os.environ["GOTOOLCHAIN"] == "local"
assert "GOROOT" not in os.environ and "GOCACHEPROG" not in os.environ
with open({str(calls)!r}, "a") as stream:
    stream.write(json.dumps(sys.argv) + "\\n")
os.execv({str(real_go)!r}, [{str(real_go)!r}, *sys.argv[1:]])
''')
        binary.chmod(0o755)
        system = root / "system/bin/go"
        system.parent.mkdir(parents=True)
        system.write_text(f'''#!{sys.executable}
import pathlib, sys
pathlib.Path({str(root / "wrong-go-used")!r}).touch()
if sys.argv[1:] == ["version"]:
    print("go version go1.24.0 linux/amd64")
else:
    sys.exit("fixture module requires Go 1.25, default Go is 1.24")
''')
        system.chmod(0o755)
        return binary, calls, os.pathsep.join((str(binary.parent), str(system.parent), packager.SYSTEM_PATH))

    def test_hosted_toolcache_selected_before_environment_is_sanitized(self):
        with tempfile.TemporaryDirectory(dir=self.root) as temporary:
            root = Path(temporary)
            binary, calls, path = self.hosted_toolchain(root)
            with patch.dict(os.environ, {"PATH": path, "GOENV": "/outside/go.env",
                                         "GOFLAGS": "-overlay=/outside/overlay.json", "GOWORK": "/outside/go.work",
                                         "GOROOT": "/outside/go", "GOTOOLCHAIN": "invalid"}):
                go = packager.resolve_go()
                self.assertEqual(go, binary)
                env = packager.go_environment(root, go)
            # Subsequent calls use the selected absolute path even if the ambient
            # PATH changes after selection.
            with patch.dict(os.environ, {"PATH": str(root / "system/bin")}):
                subprocess.run([go, "env", "GOVERSION"], env=env, check=True, capture_output=True)
            invocations = [json.loads(line) for line in calls.read_text().splitlines()]
            self.assertEqual([args[1:] for args in invocations], [["version"], ["env", "GOVERSION"]])
            self.assertFalse((root / "wrong-go-used").exists())

    def test_missing_or_invalid_selected_toolchain_is_refused(self):
        with tempfile.TemporaryDirectory(dir=self.root) as temporary:
            root = Path(temporary)
            with patch.dict(os.environ, {"PATH": str(root)}):
                with self.assertRaisesRegex(RuntimeError, "select an installed Go"):
                    packager.resolve_go()
                (root / "go").write_text("#!/bin/sh\nprintf 'not a Go compiler\\n'\n")
                (root / "go").chmod(0o755)
                with self.assertRaisesRegex(RuntimeError, "did not identify a Go toolchain"):
                    packager.resolve_go()

    def test_export_excludes_ignored_files_and_checkout_attributes(self):
        ignored = self.baseline / "plugins/omabit/outside.pill"
        ignored.write_bytes(b"ignored packaging input")
        attributes = self.baseline / ".git/info/attributes"
        attributes.parent.mkdir(exist_ok=True)
        attributes.write_text("plugins/omabit/Model.js export-ignore\n*.qml export-subst\n")
        try:
            self.assertEqual(packager.clean_commit(self.baseline), self.commit)
            destination = self.root / "export"
            packager.export_source(self.baseline, destination, self.commit)
            self.assertFalse((destination / "plugins/omabit/outside.pill").exists())
            self.assertEqual((destination / "plugins/omabit/Model.js").read_bytes(),
                             (self.baseline / "plugins/omabit/Model.js").read_bytes())
            self.assertEqual(packager.clean_commit(destination), self.commit)
            self.assertEqual(packager.source_git(destination, "rev-parse", "HEAD^{tree}"),
                             packager.source_git(self.baseline, "rev-parse", "HEAD^{tree}"))
        finally:
            ignored.unlink()
            attributes.unlink()

    def test_dirty_and_untracked_source_refused(self):
        for path in (self.baseline / "plugins/omabit/untracked.qml", self.baseline / "plugins/omabit/Model.js"):
            original = path.read_bytes() if path.exists() else None
            path.write_bytes(b"uncommitted fixture\n")
            try:
                with self.assertRaisesRegex(RuntimeError, "commit source"):
                    packager.clean_commit(self.baseline)
                output = self.root / "dirty-output"
                with patch.object(packager, "ROOT", self.baseline), patch("sys.argv", [
                        "package-release.py", "--version", "0.1.0-rc.11", "--output", str(output)]):
                    with contextlib.redirect_stderr(io.StringIO()), self.assertRaises(SystemExit) as failure:
                        packager.main()
                    self.assertEqual(failure.exception.code, 2)
                with patch.object(plugin_packager, "ROOT", self.baseline), patch("sys.argv", [
                        "package-plugin.py", str(self.root / "missing-assets"), "--output", str(output)]):
                    with contextlib.redirect_stderr(io.StringIO()), self.assertRaises(SystemExit) as failure:
                        plugin_packager.main()
                    self.assertEqual(failure.exception.code, 2)
                self.assertFalse(output.exists(), "dirty-source refusal must not publish output")
            finally:
                if original is None:
                    path.unlink()
                else:
                    path.write_bytes(original)

    def test_environment_is_allowlisted_and_local_module_replacements_refused(self):
        with tempfile.TemporaryDirectory(dir=self.root) as temporary:
            stage = Path(temporary)
            ambient = {"GOFLAGS": "-overlay=/outside/overlay.json -toolexec=/outside/compiler",
                       "GOWORK": "/outside/go.work", "GOENV": "/outside/go.env", "GOROOT": "/outside/go",
                       "GOEXPERIMENT": "arenas", "GOCACHEPROG": "/outside/cache", "GOTOOLCHAIN": "evil",
                       "GIT_DIR": "/outside/repository", "LD_PRELOAD": "/outside/library", "GOMODCACHE": "/outside/modules"}
            with patch.dict(os.environ, ambient):
                go = packager.resolve_go()
                env = packager.go_environment(stage, go)
                self.assertEqual(packager.clean_commit(self.baseline), self.commit)
            self.assertEqual(env["GOFLAGS"], "")
            self.assertEqual(env["GOWORK"], "off")
            self.assertEqual(env["GOENV"], "off")
            self.assertEqual(env["GOTOOLCHAIN"], "local")
            self.assertEqual(env["GOMODCACHE"], str(stage / "modules"))
            for key in ("GOROOT", "GOCACHEPROG", "GIT_DIR", "LD_PRELOAD"):
                self.assertNotIn(key, env)
            project, outside = stage / "project", stage / "outside"
            project.mkdir()
            outside.mkdir()
            (project / "go.mod").write_text("module example.invalid/fixture\n\ngo 1.25.0\n"
                                           "require example.invalid/outside v0.0.0\n"
                                           "replace example.invalid/outside => ../outside\n")
            (outside / "go.mod").write_text("module example.invalid/outside\n\ngo 1.25.0\n")
            with self.assertRaisesRegex(RuntimeError, "local filesystem replacements"):
                packager.check_modules(project, env, go)

    def test_full_reproduction_with_negative_environment_and_existing_smoke(self):
        if not BUILD:
            self.skipTest("use --build for independent binary builds and package smoke")
        pill = self.root / "inert.pill"
        pill.write_bytes(b"inert release test pill; never booted\n")
        version = json.loads((self.baseline / "plugins/omabit/manifest.json").read_text())["version"]
        schema_source = self.baseline / "internal/managed/schema.go"
        outside = self.root / "outside.go"
        # A syntactically invalid replacement must break a contaminated build.
        outside.write_text("THIS IS NOT GO SOURCE\n")
        overlay = self.root / "overlay.json"
        overlay.write_text(json.dumps({"Replace": {str(schema_source): str(outside)}}))
        workspace = self.root / "go.work"
        workspace.write_text("invalid external workspace\n")
        config = self.root / "go.env"
        config.write_text("GOFLAGS=-overlay=" + str(overlay) + "\nGOWORK=" + str(workspace) + "\n")
        ignored = self.baseline / "plugins/omabit/outside.pill"
        outputs = []
        hosted = self.root / "hosted-build"
        selected_go, toolchain_calls, hosted_path = self.hosted_toolchain(hosted)
        export = packager.export_source

        def export_with_overlay_target(root, destination, commit):
            export(root, destination, commit)
            # Also target the actual private build directory. Otherwise an
            # overlay keyed only to the caller's checkout would be ineffective
            # even if GOFLAGS accidentally leaked into the build environment.
            overlay.write_text(json.dumps({"Replace": {
                str(schema_source): str(outside),
                str(destination / "internal/managed/schema.go"): str(outside)}}))

        for number in (1, 2):
            output = self.root / f"build-{number}"
            argv = ["package-release.py", "--version", version, "--pill", str(pill),
                    "--pill-sha256", packager.sha256(pill), "--output", str(output)]
            ambient = {} if number == 1 else {"GOFLAGS": "-overlay=" + str(overlay), "GOWORK": str(workspace),
                                            "GOENV": str(config), "GOCACHEPROG": "/nonexistent/cache-program",
                                            "GOMODCACHE": "/nonexistent/modules", "GOTOOLCHAIN": "invalid-toolchain",
                                            "GOROOT": "/nonexistent/go", "PATH": hosted_path}
            if number == 2:
                ignored.write_bytes(b"must not enter certified archive\n")
            with patch.object(packager, "ROOT", self.baseline), patch("sys.argv", argv), \
                    patch.dict(os.environ, ambient), patch.object(packager, "export_source", export_with_overlay_target):
                packager.main()
            outputs.append(output / version)
        first, second = outputs
        self.assertEqual({p.name: packager.sha256(p) for p in first.iterdir()},
                         {p.name: packager.sha256(p) for p in second.iterdir()},
                         "independent builds must reproduce every artifact and metadata byte")
        metadata = json.loads((first / "release.json").read_text())
        self.assertEqual(metadata["source_commit"], self.commit)
        archive_name = "omabit-" + version + "-linux-amd64.tar.gz"
        with tarfile.open(first / archive_name) as archive:
            self.assertFalse(any(p.name.endswith("/outside.pill") for p in archive.getmembers()))
            tracked = packager.source_git(self.baseline, "ls-tree", "-r", "--name-only", self.commit,
                                          "plugins/omabit", "deploy").splitlines()
            for name in tracked:
                self.assertEqual(archive.extractfile("omabit-" + version + "-linux-amd64/" + name).read(),
                                 packager.source_git(self.baseline, "show", self.commit + ":" + name, text=False))
        subprocess.run([sys.executable, "-B", str(ROOT / "scripts/test-release.py"), first], check=True,
                       env=dict(os.environ, **ambient), timeout=120)
        invocations = [json.loads(line) for line in toolchain_calls.read_text().splitlines()]
        self.assertTrue(all(args[0] == str(selected_go) for args in invocations))
        self.assertTrue(any(args[1] == "build" for args in invocations), "build bypassed selected toolchain")
        self.assertEqual(sum(args[1:3] == ["version", "-m"] for args in invocations), 3,
                         "release smoke bypassed selected toolchain")
        self.assertFalse((hosted / "wrong-go-used").exists())
        # Integration-only fixture: exercise the NEW uncommitted installer against
        # a real old bundle and its v1 ownership marker. Do not repackage, rehash or
        # certify this edited extraction as the baseline commit.
        extraction = self.root / "development-extraction"
        extraction.mkdir()
        with tarfile.open(first / archive_name) as archive:
            archive.extractall(extraction, filter="data")
        package = extraction / ("omabit-" + version + "-linux-amd64")
        destination = self.root / "development-install"
        env = dict(os.environ, HOME=str(self.root / "home"), XDG_CONFIG_HOME=str(self.root / "config"))
        command = [str(package / "install.sh"), "--destination", str(destination)]
        subprocess.run(command, env=env, check=True, timeout=30)
        old_model = (destination / "Model.js").read_bytes()
        shutil.copy2(ROOT / "scripts/install-panel.py", package / "scripts/install-panel.py")
        subprocess.run([*command, "--replace"], env=env, check=True, timeout=30)
        backup = next(self.root.glob(".development-install.omabit-recovery-*/previous"))
        self.assertEqual((backup / "Model.js").read_bytes(), old_model)
        subprocess.run([*command, "--replace"], env=env, check=True, timeout=30)
        (destination / "Model.js").write_bytes(old_model + b"\n// development fixture customization\n")
        result = subprocess.run([*command, "--replace"], env=env, capture_output=True, text=True, timeout=30)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("local changes", result.stderr)
        self.assertTrue((destination / "Model.js").read_bytes().endswith(b"// development fixture customization\n"))
        print("PASS: current uncommitted installer upgrades an isolated baseline bundle; fixture is not a certified release")
        # Exercise the publishable plugin entrypoint with the same frozen source;
        # ignored native/bootstrap files and caller Git/Go configuration stay out.
        native_ignored = self.baseline / "native/bootstrap/outside.pyc"
        native_ignored.write_bytes(b"ignored distribution input")
        for number, assets in enumerate(outputs):
            destination = self.root / f"plugin-{number}"
            with patch.object(plugin_packager, "ROOT", self.baseline), patch("sys.argv", [
                    "package-plugin.py", str(assets), "--output", str(destination)]):
                plugin_packager.main()
            self.assertFalse((destination / "bootstrap/outside.pyc").exists())
            for name in ("runtime.py",):
                self.assertEqual((destination / "bootstrap" / name).read_bytes(),
                                 packager.source_git(self.baseline, "show", self.commit + ":native/bootstrap/" + name, text=False))
        def content(directory):
            return {str(p.relative_to(directory)): packager.sha256(p) for p in directory.rglob("*") if p.is_file()}
        self.assertEqual(content(self.root / "plugin-0"), content(self.root / "plugin-1"))


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--build", action="store_true")
    args, remaining = parser.parse_known_args()
    BUILD = args.build
    unittest.main(argv=[__file__, *remaining], verbosity=2)
