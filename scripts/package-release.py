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
SYSTEM_PATH = "/usr/bin:/bin"


def source_environment():
    # In particular, do not inherit GIT_DIR, replacement objects, global filters,
    # GOFLAGS overlays, go env -w, workspaces, compiler wrappers or cache helpers.
    return {"PATH": SYSTEM_PATH, "LC_ALL": "C", "TZ": "UTC",
            "GIT_CONFIG_NOSYSTEM": "1", "GIT_CONFIG_GLOBAL": os.devnull,
            "GIT_NO_REPLACE_OBJECTS": "1", "GIT_OPTIONAL_LOCKS": "0"}


def source_git(root, *args, text=True):
    return subprocess.check_output(["git", "-c", "core.fsmonitor=false", *args],
                                   cwd=root, env=source_environment(), text=text)


def clean_commit(root):
    if source_git(root, "status", "--porcelain=v1", "--untracked-files=all").strip():
        raise RuntimeError("commit source changes before building a release")
    return source_git(root, "rev-parse", "HEAD").strip()


def export_source(root, destination, commit):
    """Materialize only commit blobs, without checkout filters/export attributes.

    Fetching existing objects and populating an index creates no new commits. The
    private repository lets Go retain its normal, truthful clean VCS build stamp.
    Never copy the caller's index, config, ignored files or working tree.
    """
    destination.mkdir()
    source_git(destination, "init", "--quiet", "--template=")
    source_git(destination, "fetch", "--quiet", "--depth=1", "--no-tags", str(root), commit)
    source_git(destination, "update-ref", "HEAD", commit)
    source_git(destination, "read-tree", commit)
    entries = source_git(destination, "ls-tree", "-rz", "--full-tree", commit, text=False)
    with subprocess.Popen(["git", "cat-file", "--batch"], cwd=destination,
                          env=source_environment(), stdin=subprocess.PIPE, stdout=subprocess.PIPE) as blobs:
        for entry in entries.split(b"\0"):
            if not entry:
                continue
            attributes, raw_name = entry.split(b"\t", 1)
            mode, kind, oid = attributes.split()
            name = os.fsdecode(raw_name)
            path = Path(name)
            if (mode not in (b"100644", b"100755") or kind != b"blob" or path.is_absolute()
                    or ".." in path.parts or ".git" in path.parts):
                raise RuntimeError("unsupported tracked release input: " + name)
            blobs.stdin.write(oid + b"\n")
            blobs.stdin.flush()
            actual_oid, actual_kind, size = blobs.stdout.readline().split()
            if actual_oid != oid or actual_kind != b"blob":
                raise RuntimeError("invalid source object")
            contents = blobs.stdout.read(int(size))
            if len(contents) != int(size) or blobs.stdout.read(1) != b"\n":
                raise RuntimeError("incomplete source object")
            target = destination / path
            target.parent.mkdir(parents=True, exist_ok=True)
            target.write_bytes(contents)
            target.chmod(0o755 if mode == b"100755" else 0o644)
        blobs.stdin.close()
        if blobs.wait() != 0:
            raise RuntimeError("source export failed")
    if clean_commit(destination) != commit:
        raise RuntimeError("export does not match the committed source")


def go_tool_environment(go):
    # Probes and binary inspection have no HOME/XDG_CONFIG_HOME, so Go cannot
    # locate telemetry configuration and defaults to off. Adding a private HOME
    # requires explicitly disabling telemetry first; see go_environment.
    return dict(source_environment(), PATH=str(go.parent) + os.pathsep + SYSTEM_PATH,
                GOENV="off", GOWORK="off", GOFLAGS="", GOEXPERIMENT="", GOTOOLCHAIN="local")


def resolve_go():
    """Trust the operator's PATH selection (including actions/setup-go), once.

    Resolve it before constructing the allowlisted subprocess environment. Every
    Go invocation then uses this absolute executable, never a system-Go fallback
    or an ambient toolchain/config override.
    """
    selected = shutil.which("go")
    if selected is None:
        raise RuntimeError("select an installed Go toolchain on PATH before packaging")
    go = Path(selected).resolve(strict=True)
    if not go.is_file() or not os.access(go, os.X_OK):
        raise RuntimeError("selected Go toolchain is not an executable file: " + str(go))
    try:
        version = subprocess.check_output([go, "version"], cwd=go.parent,
                                          env=go_tool_environment(go), text=True, timeout=10).strip()
    except (OSError, subprocess.SubprocessError) as error:
        raise RuntimeError("cannot run selected Go toolchain: " + str(go)) from error
    if not re.fullmatch(r"go version (?:go\d+\.\d+\S*|devel .+) \S+/\S+", version):
        raise RuntimeError("selected executable did not identify a Go toolchain: " + str(go))
    return go


def go_environment(stage, go):
    home, temporary = stage / "home", stage / "tmp"
    home.mkdir()
    temporary.mkdir()
    env = dict(go_tool_environment(go), HOME=str(home), TMPDIR=str(temporary),
               XDG_CONFIG_HOME=str(stage / "config"),
               CGO_ENABLED="0", GOOS="linux", GOARCH="amd64", GOAMD64="v1",
               GOPATH=str(stage / "gopath"),
               GOMODCACHE=str(stage / "modules"), GOCACHE=str(stage / "cache"),
               GOPROXY="https://proxy.golang.org", GOSUMDB="sum.golang.org")
    # GOTELEMETRY is a read-only go env value in Go 1.25/1.27, not an environment
    # control. A fresh config otherwise defaults to local collection, whose
    # background child can outlive Go and race stage cleanup. This command skips
    # telemetry startup and persists off in our private config before any queries
    # or builds, without reading or changing the operator's telemetry settings.
    subprocess.run([go, "telemetry", "off"], cwd=stage, env=env, check=True, timeout=10)
    return env


def check_modules(source, env, go):
    modules = subprocess.check_output([go, "list", "-mod=readonly", "-m", "-json", "all"],
                                      cwd=source, env=env, text=True)
    decoder = json.JSONDecoder()
    while modules.strip():
        module, end = decoder.raw_decode(modules.lstrip())
        modules = modules.lstrip()[end:]
        if "Replace" in module and not module["Replace"].get("Version"):
            raise RuntimeError("release modules must not use local filesystem replacements")


def sha256(path):
    with path.open("rb") as f:
        digest = hashlib.sha256()
        for chunk in iter(lambda: f.read(1024 * 1024), b""):
            digest.update(chunk)
        return digest.hexdigest()


def git(*args):
    return source_git(ROOT, *args).strip()


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
    try:
        commit = clean_commit(ROOT)
    except RuntimeError as error:
        parser.error(str(error))
    if args.pill is None or not args.pill.is_file() or not re.fullmatch(r"[0-9a-f]{64}", args.pill_sha256 or ""):
        parser.error("require a frozen --pill and --pill-sha256")
    if args.pill.is_symlink() or not 0 < args.pill.stat().st_size <= 4_000_000_000 or sha256(args.pill) != args.pill_sha256:
        parser.error("pill does not match its bounded expected content")
    if final.exists():
        parser.error("version output already exists; inspect rather than overwrite")
    try:
        go = resolve_go()
    except RuntimeError as error:
        parser.error(str(error))
    output.mkdir(parents=True, exist_ok=True)
    epoch = int(git("show", "-s", "--format=%ct", commit))
    prefix = "omabit-" + args.version + "-linux-amd64"
    with tempfile.TemporaryDirectory(prefix=".release-", dir=output) as temporary:
        stage = Path(temporary)
        source = stage / "source"
        export_source(ROOT, source, commit)
        manifest = json.loads((source / "plugins/omabit/manifest.json").read_text())
        if manifest["version"] != args.version:
            parser.error("plugin version and release version must match")
        package = stage / prefix
        binaries = package / "build"
        binaries.mkdir(parents=True)
        env = go_environment(stage, go)
        check_modules(source, env, go)
        subprocess.run([go, "build", "-mod=readonly", "-buildvcs=true", "-trimpath", "-o", str(binaries) + "/", "./cmd/omabit", "./backend/cmd/urbit-collector", "./backend/cmd/urbit-runner"], cwd=source, env=env, check=True)
        schema = int(subprocess.check_output([binaries / "urbit-runner", "state-schema"], cwd=source,
                                             env=env, text=True, timeout=10).strip())
        (package / "plugins").mkdir()
        shutil.copytree(source / "plugins/omabit", package / "plugins/omabit")
        (package / "scripts").mkdir()
        for name in ("install-panel.py", "install-services.sh"):
            shutil.copy2(source / "scripts" / name, package / "scripts" / name)
        shutil.copytree(source / "deploy", package / "deploy")
        # install-services.sh consumes this file in addition to the unit templates.
        (package / "internal/rootless/assets").mkdir(parents=True)
        shutil.copy2(source / "internal/rootless/assets/daemon.json", package / "internal/rootless/assets/daemon.json")
        shutil.copy2(source / "docs/release-install.md", package / "README.md")
        installer = package / "install.sh"
        installer.write_text('#!/usr/bin/env bash\nset -euo pipefail\nROOT="$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")"\nexec python3 -B "$ROOT/scripts/install-panel.py" "$@"\n')
        installer.chmod(0o755)
        metadata = {"version": args.version, "prerelease": "-rc." in args.version, "source_commit": commit, "platform": "linux/amd64", "go_version": subprocess.check_output([go, "version"], env=env, text=True).strip(), "schema": schema, "execution_model": "owner-user", "default_pier_root": "/srv/urbit/piers", "runtime": "4.6-5a7614f", "image": "docker.io/tloncorp/vere@sha256:47fbcf0710ded939931ce3c6408864cee5e8848c5765da0b51a90ca0b1ab1afd", "engine": "distribution-native", "pill": {"sha256": args.pill_sha256, "default_url": manifest["barWidget"]["defaults"]["recommendedPillUrl"], "application_inventory": "not_asserted"}}
        metadata["engine_policy"] = {
            "id": "docker-rootless-v1", "stable_minimum_inclusive": "27.2.0",
            "stable_maximum_exclusive": "30.0.0", "required_api": "1.47",
            "native_providers": ["pacman", "deb"], "rootless_launcher_version": "29.7.2",
            "qualification": "exact-stack-evidence-required",
        }
        metadata.update(source_tree=source_git(source, "rev-parse", "HEAD^{tree}").strip(),
                        source_epoch=epoch, source_inputs="git-commit-blobs-v1",
                        build_environment="isolated-go-v1")
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
        if clean_commit(source) != commit or clean_commit(ROOT) != commit:
            raise RuntimeError("source changed during packaging")
        assets.rename(final)
    print("Built release assets:", final)
    print("Source revision:", commit)


if __name__ == "__main__":
    main()
