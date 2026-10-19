#!/usr/bin/env python3
"""Build immutable development snapshots and manage explicit desktop test profiles."""
import argparse
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import re
import shutil
import stat
import subprocess
import sys
import tempfile

ROOT = Path(__file__).resolve().parents[1]
sys.dont_write_bytecode = True
sys.path.insert(0, str(ROOT / "native/bootstrap"))
import control


def load_packager():
    spec = importlib.util.spec_from_file_location("release_packager", ROOT / "scripts/package-release.py")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def inputs():
    names = subprocess.check_output(["git", "ls-files", "--cached", "--others", "--exclude-standard", "-z"], cwd=ROOT).split(b"\0")
    result = {}
    for raw in sorted(set(names) - {b""}):
        name = os.fsdecode(raw)
        path = ROOT / name
        if not path.exists() and not path.is_symlink():
            continue  # Deliberately deleted tracked source.
        info = path.lstat()
        if not stat.S_ISREG(info.st_mode) or info.st_size > 16 << 20:
            raise ValueError("unsupported development source input: " + name)
        data = path.read_bytes()
        result[name] = {"sha256": hashlib.sha256(data).hexdigest(), "mode": 0o755 if info.st_mode & 0o111 else 0o644}
    return result


def build(argv):
    parser = argparse.ArgumentParser(description="Snapshot current source, build isolated tools and UI; does not select or install them.")
    parser.add_argument("--output", type=Path, required=True, help="new output directory")
    parser.add_argument("--label", required=True, help="explicit development label")
    args = parser.parse_args(argv)
    if not re.fullmatch(r"[a-zA-Z0-9][a-zA-Z0-9._-]{0,63}", args.label):
        parser.error("use a short alphanumeric development label")
    output = args.output.absolute()
    if output.exists() or not output.parent.is_dir():
        parser.error("choose a new output directory under an existing parent")
    packager = load_packager()
    commit = packager.source_git(ROOT, "rev-parse", "HEAD").strip()
    snapshot = inputs()
    digest = hashlib.sha256(json.dumps(snapshot, sort_keys=True).encode()).hexdigest()
    go = packager.resolve_go()
    with tempfile.TemporaryDirectory(prefix=".omabit-dev-", dir=output.parent) as temporary:
        stage = Path(temporary)
        source = stage / "source"
        packager.export_source(ROOT, source, commit)
        # Overlay exact worktree bytes into the private Git export. Go records
        # truthful vcs.modified=true for local changes; no synthetic commit.
        for path in source.rglob("*"):
            if path.is_file() and ".git" not in path.relative_to(source).parts and path.relative_to(source).as_posix() not in snapshot:
                path.unlink()
        for name, entry in snapshot.items():
            data = (ROOT / name).read_bytes()
            if hashlib.sha256(data).hexdigest() != entry["sha256"]:
                raise ValueError("source changed while snapshotting: " + name)
            target = source / name
            target.parent.mkdir(parents=True, exist_ok=True)
            target.write_bytes(data)
            target.chmod(entry["mode"])
        version = json.loads((source / "plugins/omabit/manifest.json").read_text())["version"]
        if 'property string applicationVersion: ' + json.dumps(version) not in (source / "plugins/omabit/Service.qml").read_text():
            raise ValueError("application version must match the source manifest before building")
        package = stage / ("omabit-" + version + "-linux-amd64")
        (package / "build").mkdir(parents=True)
        env = packager.go_environment(stage, go)
        packager.check_modules(source, env, go)
        subprocess.run([go, "build", "-mod=readonly", "-buildvcs=true", "-trimpath", "-o", str(package / "build") + "/",
                        "./cmd/omabit", "./backend/cmd/urbit-collector", "./backend/cmd/urbit-runner"], cwd=source, env=env, check=True)
        shutil.copytree(source / "plugins/omabit", package / "plugins/omabit")
        schema = int(subprocess.check_output([package / "build/urbit-runner", "state-schema"], env=env, text=True))
        metadata = {"version": version, "platform": "linux/amd64", "source_commit": commit, "schema": schema,
                    "development": {"label": args.label, "snapshot_sha256": digest,
                                    "modified": bool(packager.source_git(source, "status", "--porcelain").strip())},
                    "go_version": subprocess.check_output([go, "version"], env=env, text=True).strip()}
        (package / "BUILD.json").write_text(json.dumps(metadata, indent=2) + "\n")
        packager.checksums(package)
        assets = stage / "assets"
        assets.mkdir()
        archive = assets / (package.name + ".tar.gz")
        epoch = int(packager.source_git(ROOT, "show", "-s", "--format=%ct", commit))
        packager.archive(package, archive, epoch)
        for name in packager.COMMANDS:
            shutil.copy2(package / "build" / name, assets / (name + "-linux-amd64"))
        metadata["assets"] = [packager.artifact(assets / (name + "-linux-amd64")) for name in packager.COMMANDS]
        metadata["assets"].append(packager.artifact(archive))
        (assets / "release.json").write_text(json.dumps(metadata, indent=2) + "\n")
        (assets / "source-inputs.json").write_text(json.dumps(snapshot, indent=2) + "\n")
        packager.checksums(assets)
        if inputs() != snapshot:
            raise ValueError("source changed during build; candidate was not published")
        assets.rename(output)
    print(json.dumps({"directory": str(output), "version": version, "development": metadata["development"],
                      "import": ["python3", "-B", "scripts/omabit-dev.py", "import-release", str(output / "release.json"), str(output / archive.name)]}, indent=2))
    return 0


if __name__ == "__main__":
    try:
        sys.exit(build(sys.argv[2:]) if sys.argv[1:2] == ["build"] else control.main())
    except (OSError, ValueError, KeyError, TypeError, subprocess.SubprocessError) as error:
        print(json.dumps({"error": str(error)}), file=sys.stderr)
        sys.exit(1)
