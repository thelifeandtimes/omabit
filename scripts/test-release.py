#!/usr/bin/env python3
"""Verify built assets and exercise the extracted bundle's isolated installer."""
import argparse
import datetime
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import subprocess
import tarfile
import tempfile

spec = importlib.util.spec_from_file_location("release_source", Path(__file__).with_name("package-release.py"))
release_source = importlib.util.module_from_spec(spec)
spec.loader.exec_module(release_source)


def digest(path):
    value = hashlib.sha256()
    with path.open("rb") as stream:
        for chunk in iter(lambda: stream.read(1024 * 1024), b""):
            value.update(chunk)
    return value.hexdigest()


def verify(directory):
    recorded = {}
    for line in (directory / "SHA256SUMS").read_text().splitlines():
        expected, name = line.split("  ", 1)
        assert name not in recorded and not Path(name).is_absolute() and ".." not in Path(name).parts
        recorded[name] = expected
        assert digest(directory / name) == expected, name
    actual = {p.relative_to(directory).as_posix() for p in directory.rglob("*")
              if p.is_file() and p.name != "SHA256SUMS"}
    assert set(recorded) == actual, "unrecorded or missing package content"


def verify_build(binary, metadata, go):
    output = subprocess.check_output([go, "version", "-m", binary],
                                     env=release_source.go_tool_environment(go), text=True)
    settings = dict(line.strip().split("\t", 1)[1].split("=", 1)
                    for line in output.splitlines() if line.strip().startswith("build\t"))
    assert settings["vcs"] == "git" and settings["vcs.revision"] == metadata["source_commit"]
    assert settings["vcs.modified"] == "false", "binary certifies modified source"
    assert settings["-trimpath"] == "true" and settings["CGO_ENABLED"] == "0"
    assert settings["GOOS"] == "linux" and settings["GOARCH"] == "amd64" and settings["GOAMD64"] == "v1"
    assert "-overlay" not in settings and "-modfile" not in settings
    timestamp = datetime.datetime.fromisoformat(settings["vcs.time"].replace("Z", "+00:00"))
    assert int(timestamp.timestamp()) == metadata["source_epoch"]


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("assets", type=Path)
    args = parser.parse_args()
    go = release_source.resolve_go()
    assets = args.assets.resolve()
    verify(assets)
    metadata = json.loads((assets / "release.json").read_text())
    assert metadata["source_inputs"] == "git-commit-blobs-v1"
    assert metadata["build_environment"] == "isolated-go-v1"
    version = metadata["version"]
    for entry in metadata["assets"]:
        path = assets / entry["name"]
        assert digest(path) == entry["sha256"] and path.stat().st_size == entry["bytes"]
    prefix = "omabit-" + version + "-linux-amd64"
    with tempfile.TemporaryDirectory(prefix="omabit-release-check-") as temporary:
        root = Path(temporary)
        with tarfile.open(assets / (prefix + ".tar.gz")) as archive:
            for member in archive.getmembers():
                path = Path(member.name)
                assert path.parts[0] == prefix and ".." not in path.parts and not path.is_absolute()
                assert member.isdir() or member.isfile(), "unexpected link/special file"
            archive.extractall(root, filter="data")
        package = root / prefix
        verify(package)
        build = json.loads((package / "BUILD.json").read_text())
        assert build["source_commit"] == metadata["source_commit"] and build["version"] == version
        for key in ("source_tree", "source_epoch", "source_inputs", "build_environment", "go_version"):
            assert build[key] == metadata[key], "archive/release provenance mismatch: " + key
        schema = int(subprocess.check_output([package / "build/urbit-runner", "state-schema"], text=True, timeout=10))
        assert schema == build["schema"] == metadata["schema"], "binary/package/release schema mismatch"
        if schema >= 21:
            assert build["engine"] == metadata["engine"] == "distribution-native"
            assert build["engine_policy"] == metadata["engine_policy"], "engine policy metadata mismatch"
            policy = metadata["engine_policy"]
            assert policy["id"] == "docker-rootless-v1" and policy["required_api"] == "1.47"
            assert policy["stable_minimum_inclusive"] == "27.2.0" and policy["stable_maximum_exclusive"] == "30.0.0"
            assert policy["qualification"] == "exact-stack-evidence-required"
        assert build["pill"]["sha256"] == metadata["pill"]["sha256"]
        manifest = json.loads((package / "plugins/omabit/manifest.json").read_text())
        assert manifest["version"] == version
        for name in ("Service.qml", "MonitorWindow.qml", "SetupPane.qml", "Panel.qml"):
            assert version in (package / "plugins/omabit" / name).read_text(), "stale packaged UI version: " + name
        if "default_url" in metadata["pill"]:
            assert manifest["barWidget"]["defaults"]["recommendedPillUrl"] == build["pill"]["default_url"] == metadata["pill"]["default_url"]
            assert metadata["pill"]["default_url"].startswith("https://")
        else:
            assert manifest["barWidget"]["defaults"]["recommendedPillUrl"].endswith(
                "/v" + version + "/" + metadata["pill"]["name"])
        for command in ("omabit", "urbit-collector", "urbit-runner"):
            binary = package / "build" / command
            assert digest(binary) == digest(assets / (command + "-linux-amd64"))
            verify_build(binary, metadata, go)
            result = subprocess.run([binary, "--help"], capture_output=True, timeout=10)
            assert result.returncode == 0, (command, result.stderr)
        for path in ("internal/rootless/assets/daemon.json", "deploy/systemd/urbit-runner.service",
                     "deploy/systemd/urbit-runner-ssh.service", "deploy/systemd/user-manager-shutdown.conf",
                     "deploy/systemd/urbit-docker.service.d/shutdown.conf"):
            assert (package / path).is_file(), "service installer dependency missing: " + path
        assert json.loads((package / "internal/rootless/assets/daemon.json").read_text()) == {"shutdown-timeout": 310}
        env = dict(os.environ, HOME=str(root / "home"), XDG_CONFIG_HOME=str(root / "config"))
        destination = root / "installed"

        def install(*flags, succeeds=True):
            result = subprocess.run([package / "install.sh", "--destination", destination, *flags],
                                    env=env, capture_output=True, text=True, timeout=30)
            assert (result.returncode == 0) == succeeds, result.stderr

        install()
        for command in ("omabit", "urbit-collector", "urbit-runner"):
            assert digest(destination / "bin" / command) == digest(package / "build" / command)
        install(succeeds=False)
        install("--replace")
        modified = destination / "Panel.qml"
        modified.write_text(modified.read_text() + "\n// operator customization\n")
        expected = digest(modified)
        install("--replace", succeeds=False)
        assert digest(modified) == expected, "installer overwrote operator customization"
    print("PASS: asset/archive checksums, provenance, executable smoke checks, installer dependencies, fresh install, replacement and customization refusal")


if __name__ == "__main__":
    main()
