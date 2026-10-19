#!/usr/bin/env python3
"""Qualify exact source/package/archive UI on an unlocked Lua Hyprland desktop.

Only fixture collectors receive UI requests. Optional binaries are checked by
hash and harmless help/schema commands, including offline native launchers.
"""
import argparse
import datetime
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tarfile
import tempfile
from unittest.mock import patch

from desktop_fixture import HARNESS_ROOT, environment, preserve_desktop

CASES = {
    "panel": ["test-panel.py", "--windows"],
    "panel-native": ["test-panel.py", "--windows", "--native"],
    "recovery": ["test-import-recovery.py"],
    "recovery-native": ["test-import-recovery.py", "--native"],
    "log-watch": ["test-log-watch.py"],
    "onboarding": ["test-onboarding.py"],
    "focus": ["test-monitor-focus.py"],
    "focus-native": ["test-monitor-focus.py"],
}
COMMANDS = ("omabit", "urbit-collector", "urbit-runner")


def digest(path):
    with path.open("rb") as stream:
        return hashlib.file_digest(stream, "sha256").hexdigest()


def tree(directory):
    return {p.relative_to(directory).as_posix(): digest(p) for p in sorted(directory.rglob("*"))
            if p.is_file() and "__pycache__" not in p.parts and p.suffix != ".pyc"}


def extract(path, destination):
    with tarfile.open(path) as archive:
        seen = set()
        for member in archive.getmembers():
            name = Path(member.name)
            assert not name.is_absolute() and ".." not in name.parts
            assert member.isdir() or member.isfile(), "archive links/special files are not supported"
            assert member.name not in seen, "duplicate archive member"
            seen.add(member.name)
        archive.extractall(destination, filter="data")
    roots = [p.parent.parent for p in destination.glob("*/plugins/omabit")]
    if (destination / "plugins/omabit").is_dir():
        roots.append(destination)
    assert len(roots) == 1, "expected one source/package root containing plugins/omabit"
    return roots[0]


def verify_package(root):
    recorded = {}
    for line in (root / "SHA256SUMS").read_text().splitlines():
        expected, name = line.split("  ", 1)
        assert name not in recorded and not Path(name).is_absolute() and ".." not in Path(name).parts
        assert not (root / name).is_symlink()
        recorded[name] = expected
        assert digest(root / name) == expected, name
    assert set(recorded) == set(tree(root)) - {"SHA256SUMS"}, "package content differs from checksum inventory"
    return json.loads((root / "BUILD.json").read_text())


def module(path):
    spec = importlib.util.spec_from_file_location("qualified_native_runtime", path)
    value = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(value)
    return value


def qualify_binaries(native, binaries, directory, expected_version=None, expected_commit=None):
    runtime = module(native / "bootstrap/runtime.py")
    pins = json.loads((native / "bootstrap/release.json").read_text())
    runtime.validate_manifest(pins)
    if expected_version:
        assert pins["version"] == expected_version, "native pins still select another release"
    if expected_commit:
        assert pins["source_commit"] == expected_commit, "native pins select another source commit"
    env = environment(directory)
    key = hashlib.sha256(json.dumps(pins, sort_keys=True, separators=(",", ":")).encode()).hexdigest()
    cache = Path(env["XDG_CACHE_HOME"]) / "omabit/native-binaries" / key
    runtime.private_directory(cache.parent)
    runtime.private_directory(cache)
    evidence = {"pins": pins, "binaries": {}}
    for name in COMMANDS:
        source = binaries / name
        if not source.exists():
            source = binaries / (name + "-linux-amd64")
        entry = pins["binaries"][name]
        assert source.stat().st_size == entry["bytes"] and digest(source) == entry["sha256"], "pin mismatch: " + name
        shutil.copyfile(source, cache / name)
        (cache / name).chmod(0o755)
        # Query embedded build info without executing any management operation.
        info = subprocess.check_output(["go", "version", "-m", source], text=True, timeout=10)
        settings = dict(line.strip().split("\t", 1)[1].split("=", 1)
                        for line in info.splitlines() if line.strip().startswith("build\t"))
        assert settings.get("vcs.revision") == pins["source_commit"] and settings.get("vcs.modified") == "false"
        assert settings.get("-trimpath") == "true" and settings.get("CGO_ENABLED") == "0"
        evidence["binaries"][name] = {"path": str(source), "sha256": digest(source), "build_info": info}
    runtime.verify_cache(cache, pins)
    with patch.object(runtime, "download", side_effect=AssertionError("offline cache attempted download")):
        assert runtime.ensure_binaries(pins, Path(env["XDG_CACHE_HOME"])) == cache
    for name in COMMANDS:
        result = subprocess.run([native / "bin" / name, "--help"], env=env, capture_output=True, text=True, timeout=10)
        assert result.returncode == 0, result.stderr
        evidence["binaries"][name]["launcher_help"] = result.stdout
    schema = subprocess.check_output([native / "bin/urbit-runner", "state-schema"], env=env, text=True, timeout=10).strip()
    evidence["schema"] = int(schema)
    print("PASS: exact binary hashes/build provenance, isolated verified offline cache and three native launcher help commands")
    return evidence


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--input", type=Path, default=HARNESS_ROOT, help="source/package root or tar archive")
    parser.add_argument("--native-root", type=Path, help="matching canonical source's native directory (needed for packages)")
    parser.add_argument("--binary-dir", type=Path, help="frozen build/ or release-assets directory; defaults to package build/")
    parser.add_argument("--expect-version", help="require both UI manifests to identify this version")
    parser.add_argument("--require-frozen", action="store_true", help="require package checksums or clean source, exact native pins and binary provenance")
    parser.add_argument("--cases", nargs="+", choices=CASES, default=list(CASES))
    parser.add_argument("--output", type=Path, required=True, help="new evidence directory; never overwritten")
    args = parser.parse_args()
    output = args.output.resolve()
    output.mkdir(mode=0o700)  # Deliberately refuse an existing report.
    report = {"started_at": datetime.datetime.now(datetime.timezone.utc).isoformat(),
              "input": str(args.input.resolve()), "require_frozen": args.require_frozen, "results": [],
              "clipboard": "not run: existing copy-code harness substitutes wl-copy; live desktop expiry remains separate"}
    try:
        with tempfile.TemporaryDirectory(prefix="omabit-qualification-") as temporary:
            directory = Path(temporary)
            source = args.input.resolve()
            if source.is_file():
                report["archive_sha256"] = digest(source)
                source = extract(source, directory / "extracted")
            assert (source / "plugins/omabit/Service.qml").is_file(), "input is not a source/package root"
            native = (args.native_root or source / "native").resolve()
            needs_native = any("native" in case for case in args.cases) or args.require_frozen or args.binary_dir or (source / "BUILD.json").exists()
            if needs_native:
                assert (native / "Service.qml").is_file(), "provide --native-root from matching frozen source"
                assert tree(native.parent / "plugins/omabit") == tree(source / "plugins/omabit"), "native wrapper imports different UI bytes"
            before = {"plugin": tree(source / "plugins/omabit"), "native": tree(native)}
            report["input_files"] = before
            report["harness_files"] = {name: digest(HARNESS_ROOT / "scripts" / name) for name in
                                       {"qualify-desktop.py", "desktop_fixture.py", *(v[0] for v in CASES.values())}}
            manifest = json.loads((source / "plugins/omabit/manifest.json").read_text())
            version = manifest["version"]
            report["ui_version"] = version
            version_contract = ('property string applicationVersion: ' if (source / "plugins/omabit/DESKTOP.json").exists() else "version:") + json.dumps(version)
            assert version_contract in (source / "plugins/omabit/Service.qml").read_text(), "loaded-QML status version differs from manifest"
            if args.expect_version:
                assert version == args.expect_version, "unexpected UI version"
            if needs_native:
                root_manifest = json.loads((native.parent / "manifest.json").read_text())
                entries = root_manifest.pop("entryPoints")
                assert entries in ({"service": "native/Service.qml", "barWidget": "native/Panel.qml"},
                                   {"service": "native/v1/Service.qml", "barWidget": "native/v1/Panel.qml"})
                root_manifest.pop("keepLoaded", None)
                component_manifest = dict(manifest)
                component_manifest.pop("entryPoints")
                assert root_manifest == component_manifest, "canonical and packaged manifests disagree"
                report["native_pins"] = json.loads((native / "bootstrap/release.json").read_text())
            build = None
            if (source / "BUILD.json").exists():
                build = verify_package(source)
                assert build["version"] == version
                report["package_build"] = build
            elif (source / ".git").exists():
                report["source_commit"] = subprocess.check_output(["git", "-C", source, "rev-parse", "HEAD"], text=True).strip()
                report["source_status"] = subprocess.check_output(["git", "-C", source, "status", "--porcelain"], text=True)
            if args.require_frozen:
                assert build or report.get("source_commit") and not report["source_status"], "frozen qualification requires a checksummed package or clean committed source"
                if not build:
                    # Native pins normally advance in a later distribution commit.
                    # Permit that commit only when UI, tools and build inputs still
                    # match the pinned binary-source commit exactly.
                    subprocess.run(["git", "-C", source, "diff", "--exit-code",
                                    report["native_pins"]["source_commit"], "HEAD", "--", "cmd", "backend", "internal",
                                    "go.mod", "go.sum", "plugins/omabit", "native/Service.qml", "native/Panel.qml",
                                    "native/bootstrap/runtime.py", "native/bootstrap/launch.py", "native/bin"],
                                   check=True, capture_output=True, timeout=10)
            binary_dir = args.binary_dir or (source / "build" if build else None)
            if args.require_frozen:
                assert binary_dir, "frozen qualification requires exact binaries"
            if binary_dir:
                binary_work = directory / "binaries"
                binary_work.mkdir()
                report["binary_qualification"] = qualify_binaries(native, binary_dir.resolve(), binary_work,
                    version if args.require_frozen else None,
                    (build["source_commit"] if build else report["native_pins"]["source_commit"]) if args.require_frozen else None)
                if build:
                    assert report["binary_qualification"]["schema"] == build["schema"]
            else:
                report["binary_qualification"] = "not run: no frozen binary directory provided"
            report["compositor"] = subprocess.check_output(["hyprctl", "version"], text=True)
            report["quickshell"] = subprocess.check_output(["qs", "--version"], text=True)
            with preserve_desktop():
                for case in args.cases:
                    case_output = output / case
                    case_output.mkdir(mode=0o700)
                    command = [sys.executable, "-B", str(HARNESS_ROOT / "scripts" / CASES[case][0]), *CASES[case][1:]]
                    if case == "focus-native":
                        command += ["--plugin-dir", str(native)]
                    env = dict(os.environ, OMABIT_QUALIFY_ROOT=str(source), OMABIT_QUALIFY_NATIVE=str(native),
                               OMABIT_QUALIFY_EVIDENCE=str(case_output), PYTHONDONTWRITEBYTECODE="1")
                    # Fixtures have their own bounded waits/cleanup. Keep the parent
                    # alive until their finally blocks restore the desktop.
                    result = subprocess.run(command, env=env, capture_output=True, text=True)
                    (case_output / "result.log").write_text(result.stdout + result.stderr)
                    report["results"].append({"case": case, "returncode": result.returncode, "command": command})
                    print(("PASS" if result.returncode == 0 else "FAIL") + ": " + case, flush=True)
                    if result.returncode:
                        print(result.stdout + result.stderr, file=sys.stderr)
            after = {"plugin": tree(source / "plugins/omabit"), "native": tree(native)}
            assert before == after, "qualification inputs changed during execution"
            assert all(digest(HARNESS_ROOT / "scripts" / name) == value for name, value in report["harness_files"].items()), "qualification harness changed during execution"
            if build:
                assert verify_package(source) == build
            if binary_dir:
                assert all(digest(Path(value["path"])) == value["sha256"]
                           for value in report["binary_qualification"]["binaries"].values()), "binaries changed during execution"
            report["inputs_unchanged"] = True
            if args.require_frozen and not build:
                assert subprocess.check_output(["git", "-C", source, "rev-parse", "HEAD"], text=True).strip() == report["source_commit"]
                assert not subprocess.check_output(["git", "-C", source, "status", "--porcelain"], text=True)
            report["passed"] = all(result["returncode"] == 0 for result in report["results"])
    except BaseException as error:
        report["passed"] = False
        report["error"] = str(error)
        raise
    finally:
        report["finished_at"] = datetime.datetime.now(datetime.timezone.utc).isoformat()
        (output / "report.json").write_text(json.dumps(report, indent=2) + "\n")
        print("Evidence:", output / "report.json")
    return 0 if report["passed"] else 1


if __name__ == "__main__":
    sys.exit(main())
