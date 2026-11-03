#!/usr/bin/env python3
"""Build immutable development snapshots and manage explicit desktop test profiles."""
import argparse
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import re
import shlex
import shutil
import stat
import subprocess
import sys
import tempfile
import time

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


def development_environment(stage, go, packager, cache):
    env = packager.go_environment(stage, go)
    if cache is not None:
        # Only these two caches survive. HOME, telemetry, GOENV, GOWORK and
        # compiler flags remain controlled exactly as in a release build.
        control.runtime.private_directory(cache)
        for name, leaf in (("GOMODCACHE", "modules"), ("GOCACHE", "build")):
            path = cache / leaf
            control.runtime.private_directory(path)
            env[name] = str(path)
    return env


def stage_candidate(argv):
    parser = argparse.ArgumentParser(description="Verify/import a candidate and prepare an inactive desktop profile. Does not select it or update any runner.")
    parser.add_argument("assets", type=Path)
    parser.add_argument("--profile", required=True)
    args = parser.parse_args(argv)
    # Validate before opening desktop state, even for the reserved default name.
    from desktop import NAME
    if not NAME.fullmatch(args.profile) or args.profile == "default":
        parser.error("choose a named development profile, not default")
    assets = args.assets.absolute()
    metadata = assets / "release.json"
    from desktop import descriptor_from_release, read_json
    descriptor = descriptor_from_release(read_json(metadata))
    desktop = control.Desktop()
    existing = desktop.status()["profiles"].get(args.profile)
    if existing and existing["host_setup"]:
        raise ValueError("candidate staging requires a profile with local host setup disabled")
    imported = desktop.import_release(metadata, assets / descriptor["bundle"]["name"])
    if existing is None:
        desktop.create_profile(args.profile)
    prepared = desktop.prepare(release_id=imported["release"], name=args.profile)
    command = ["python3", "-B", str(ROOT / "scripts/omabit-dev.py"), "select", args.profile, "--release", imported["release"]]
    print(json.dumps(dict(imported, profile=args.profile, directory=prepared["directory"],
                         activate=shlex.join(command), selected=desktop.status()["selected"],
                         note="Prepared only. Profiles isolate desktop state, not runner hosts."), indent=2))
    return 0


def build(argv):
    parser = argparse.ArgumentParser(description="Snapshot current source, build isolated tools and UI; does not select or install them.")
    parser.add_argument("--output", type=Path, required=True, help="new output directory")
    parser.add_argument("--label", required=True, help="explicit development label")
    cache_options = parser.add_mutually_exclusive_group()
    cache_options.add_argument("--cache-dir", type=Path, help="private reusable Go cache (default: XDG_CACHE_HOME/omabit/development/go)")
    cache_options.add_argument("--no-cache", action="store_true", help="fresh per-build caches for diagnosis; public release packaging always remains isolated")
    args = parser.parse_args(argv)
    started = time.monotonic()
    cache = None if args.no_cache else (args.cache_dir or Path(os.environ.get("XDG_CACHE_HOME") or Path.home() / ".cache") / "omabit/development/go").absolute()
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
        env = development_environment(stage, go, packager, cache)
        packager.check_modules(source, env, go)
        subprocess.run([go, "build", "-mod=readonly", "-buildvcs=true", "-trimpath", "-o", str(package / "build") + "/",
                        "./cmd/omabit", "./backend/cmd/urbit-collector", "./backend/cmd/urbit-runner"], cwd=source, env=env, check=True)
        # Verify downloaded/reused module contents before publishing this snapshot.
        subprocess.run([go, "mod", "verify"], cwd=source, env=env, check=True, stdout=sys.stderr)
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
        skill_asset = packager.package_skills(source, stage, assets, metadata, epoch)
        if skill_asset:
            metadata["assets"].append(skill_asset)
        (assets / "release.json").write_text(json.dumps(metadata, indent=2) + "\n")
        (assets / "source-inputs.json").write_text(json.dumps(snapshot, indent=2) + "\n")
        packager.checksums(assets)
        if inputs() != snapshot:
            raise ValueError("source changed during build; candidate was not published")
        assets.rename(output)
    print(json.dumps({"directory": str(output), "version": version, "development": metadata["development"],
                      "elapsed_seconds": round(time.monotonic() - started, 2), "cache": str(cache) if cache else None,
                      "stage": ["python3", "-B", "scripts/omabit-dev.py", "stage", str(output), "--profile", "development"],
                      "import": ["python3", "-B", "scripts/omabit-dev.py", "import-release", str(output / "release.json"), str(output / archive.name)]}, indent=2))
    return 0


if __name__ == "__main__":
    try:
        command = sys.argv[1:2]
        if command in (["--help"], ["-h"]):
            print("Development commands: build, stage, check (use COMMAND --help).\n"
                  "Build and stage prepare candidates; select explicitly activates one.\n")
        if command == ["check"]:
            sys.path.insert(0, str(ROOT / "scripts"))
            from dev_checks import main
            sys.exit(main(sys.argv[2:]))
        sys.exit(build(sys.argv[2:]) if command == ["build"] else
                 stage_candidate(sys.argv[2:]) if command == ["stage"] else control.main())
    except (OSError, ValueError, KeyError, TypeError, RuntimeError, control.runtime.BootstrapError, subprocess.SubprocessError) as error:
        print(json.dumps({"error": str(error)}), file=sys.stderr)
        sys.exit(1)
