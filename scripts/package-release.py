#!/usr/bin/env python3
"""Build a versioned Linux/AMD64 binary bundle from a clean source revision."""
import argparse
import gzip
import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import tarfile
import tempfile

ROOT = Path(__file__).resolve().parents[1]
COMMANDS = ("omabit", "urbit-collector", "urbit-runner")


def sha256(path):
    with path.open("rb") as f:
        digest = hashlib.sha256()
        for chunk in iter(lambda: f.read(1024 * 1024), b""):
            digest.update(chunk)
        return digest.hexdigest()


def git(*args):
    return subprocess.check_output(["git", *args], cwd=ROOT, text=True).strip()


def checksums(directory):
    paths = sorted(p for p in directory.rglob("*") if p.is_file() and p.name != "SHA256SUMS")
    (directory / "SHA256SUMS").write_text("".join(f"{sha256(p)}  {p.relative_to(directory).as_posix()}\n" for p in paths))


def artifact(path):
    return {"name": path.name, "bytes": path.stat().st_size, "sha256": sha256(path)}


def archive(source, destination, epoch):
    with destination.open("xb") as raw, gzip.GzipFile(filename="", mode="wb", fileobj=raw, mtime=epoch) as zipped:
        with tarfile.open(fileobj=zipped, mode="w", format=tarfile.PAX_FORMAT) as tar:
            for path in [source, *sorted(source.rglob("*"))]:
                if path.is_symlink():
                    raise RuntimeError("release inputs must not contain symlinks")
                info = tar.gettarinfo(str(path), arcname=(Path(source.name) / path.relative_to(source)).as_posix())
                info.uid = info.gid = 0
                info.uname = info.gname = ""
                info.mtime = epoch
                info.pax_headers = {}
                info.mode = 0o755 if path.is_dir() or path.stat().st_mode & 0o111 else 0o644
                if path.is_file():
                    with path.open("rb") as contents:
                        tar.addfile(info, contents)
                else:
                    tar.addfile(info)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--version", required=True)
    parser.add_argument("--pill", type=Path)
    parser.add_argument("--pill-sha256")
    parser.add_argument("--output", type=Path, default=ROOT / "dist")
    parser.add_argument("--qualification", type=Path, help="attach qualification to an already built version; does not rebuild artifacts")
    args = parser.parse_args()
    if not re.fullmatch(r"[0-9]+\.[0-9]+\.[0-9]+(?:-rc\.[0-9]+)?", args.version):
        parser.error("use a release version such as 0.1.0-rc.1")
    output = args.output.resolve()
    final = output / args.version
    if args.qualification:
        report = json.loads(args.qualification.read_text())
        if report.get("version") != args.version:
            parser.error("qualification version does not match")
        metadata = json.loads((final / "release.json").read_text())
        if report.get("source_commit") != metadata["source_commit"] or report.get("pill_sha256") != metadata["pill"]["sha256"]:
            parser.error("qualification source/pill does not match the built candidate")
        for asset in metadata["assets"]:
            if artifact(final / asset["name"]) != asset:
                parser.error("candidate asset changed before qualification attachment")
        target = final / "qualification.json"
        if target.exists():
            parser.error("qualification already attached; inspect rather than overwrite")
        target.write_text(json.dumps(report, indent=2) + "\n")
        metadata["qualification"] = artifact(target)
        (final / "release.json").write_text(json.dumps(metadata, indent=2) + "\n")
        checksums(final)
        print("Attached qualification and refreshed release checksums:", final)
        return
    if git("status", "--porcelain"):
        parser.error("commit source changes before building a release")
    manifest = json.loads((ROOT / "plugins/omabit/manifest.json").read_text())
    if manifest["version"] != args.version:
        parser.error("plugin version and release version must match")
    if args.pill is None or not args.pill.is_file() or not re.fullmatch(r"[0-9a-f]{64}", args.pill_sha256 or ""):
        parser.error("require a frozen --pill and --pill-sha256")
    if args.pill.is_symlink() or not 0 < args.pill.stat().st_size <= 4_000_000_000 or sha256(args.pill) != args.pill_sha256:
        parser.error("pill does not match its bounded expected content")
    if final.exists():
        parser.error("version output already exists; inspect rather than overwrite")
    output.mkdir(parents=True, exist_ok=True)
    commit = git("rev-parse", "HEAD")
    epoch = int(git("show", "-s", "--format=%ct", "HEAD"))
    prefix = "omabit-" + args.version + "-linux-amd64"
    with tempfile.TemporaryDirectory(prefix=".release-", dir=output) as temporary:
        stage = Path(temporary)
        package = stage / prefix
        binaries = package / "build"
        binaries.mkdir(parents=True)
        env = dict(os.environ, CGO_ENABLED="0", GOOS="linux", GOARCH="amd64")
        subprocess.run(["go", "build", "-trimpath", "-o", str(binaries) + "/", "./cmd/omabit", "./backend/cmd/urbit-collector", "./backend/cmd/urbit-runner"], cwd=ROOT, env=env, check=True)
        (package / "plugins").mkdir()
        shutil.copytree(ROOT / "plugins/omabit", package / "plugins/omabit")
        (package / "scripts").mkdir()
        for name in ("install-panel.py", "install-services.sh"):
            shutil.copy2(ROOT / "scripts" / name, package / "scripts" / name)
        shutil.copytree(ROOT / "deploy", package / "deploy")
        # install-services.sh consumes this file in addition to the unit templates.
        (package / "internal/rootless/assets").mkdir(parents=True)
        shutil.copy2(ROOT / "internal/rootless/assets/daemon.json", package / "internal/rootless/assets/daemon.json")
        shutil.copy2(ROOT / "docs/release-install.md", package / "README.md")
        installer = package / "install.sh"
        installer.write_text('#!/usr/bin/env bash\nset -euo pipefail\nROOT="$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")"\nexec python3 -B "$ROOT/scripts/install-panel.py" "$@"\n')
        installer.chmod(0o755)
        metadata = {"version": args.version, "prerelease": "-rc." in args.version, "source_commit": commit, "platform": "linux/amd64", "go_version": subprocess.check_output(["go", "version"], env=env, text=True).strip(), "schema": 16, "execution_model": "owner-user", "default_pier_root": "/srv/urbit/piers", "runtime": "4.6-5a7614f", "image": "docker.io/tloncorp/vere@sha256:47fbcf0710ded939931ce3c6408864cee5e8848c5765da0b51a90ca0b1ab1afd", "engine": "29.7.2", "pill": {"sha256": args.pill_sha256, "default_url": manifest["barWidget"]["defaults"]["recommendedPillUrl"], "application_inventory": "not_asserted"}}
        (package / "BUILD.json").write_text(json.dumps(metadata, indent=2) + "\n")
        checksums(package)
        assets = stage / "assets"
        assets.mkdir()
        archive(package, assets / (prefix + ".tar.gz"), epoch)
        for name in COMMANDS:
            shutil.copy2(binaries / name, assets / (name + "-linux-amd64"))
        pill = assets / ("omabit-recommended-apps-" + args.version + ".pill")
        shutil.copyfile(args.pill, pill)
        if sha256(pill) != args.pill_sha256:
            raise RuntimeError("pill changed during packaging")
        metadata["pill"].update(artifact(pill))
        metadata["assets"] = [artifact(p) for p in sorted(assets.iterdir())]
        (assets / "release.json").write_text(json.dumps(metadata, indent=2) + "\n")
        checksums(assets)
        if git("status", "--porcelain") or git("rev-parse", "HEAD") != commit:
            raise RuntimeError("source changed during packaging")
        assets.rename(final)
    print("Built release assets:", final)
    print("Source revision:", commit)


if __name__ == "__main__":
    main()
