#!/usr/bin/env python3
"""Install an owned Omabit panel snapshot, preserving existing/user-modified files."""
import argparse
import hashlib
import json
import os
import pathlib
import shutil
import subprocess
import tempfile

ROOT = pathlib.Path(__file__).resolve().parents[1]
PLUGIN = "thelifeandtimes.omabit"
MARKER = ".omabit-install.json"


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def install(destination, replace=False):
    destination = destination.absolute()
    destination.parent.mkdir(parents=True, exist_ok=True, mode=0o700)
    if destination.parent.resolve() != destination.parent or destination.is_symlink():
        raise RuntimeError("refusing a symlinked installation path")
    if destination.exists():
        if not replace:
            raise RuntimeError("destination exists; use --replace only for an unmodified owned snapshot")
        marker = destination / MARKER
        if marker.is_symlink() or not marker.is_file():
            raise RuntimeError("refusing an unmanaged installation")
        state = json.loads(marker.read_text())
        actual = {str(p.relative_to(destination)) for p in destination.rglob("*") if p.is_file() and p.name != MARKER}
        if state.get("version") != 1 or actual != set(state.get("files", {})):
            raise RuntimeError("installed snapshot contains unexpected files")
        for name, expected in state["files"].items():
            path = destination / name
            if path.is_symlink() or path.resolve().parent not in [destination, destination / "bin"] or digest(path) != expected:
                raise RuntimeError("installed snapshot has local changes; refusing replacement")
    binary = ROOT / "build/urbit-collector"
    if not binary.is_file():
        raise RuntimeError("build urbit-collector before installing the panel")
    with tempfile.TemporaryDirectory(prefix=".omabit-stage-", dir=destination.parent) as temporary:
        staging = pathlib.Path(temporary) / PLUGIN
        shutil.copytree(ROOT / "plugins/omabit", staging)
        (staging / "bin").mkdir()
        shutil.copy2(binary, staging / "bin/urbit-collector")
        os.chmod(staging / "bin/urbit-collector", 0o755)
        state = {"version": 1, "files": {str(p.relative_to(staging)): digest(p) for p in staging.rglob("*") if p.is_file()}}
        (staging / MARKER).write_text(json.dumps(state, indent=2) + "\n")
        if destination.exists():
            backup = pathlib.Path(temporary) / "previous"
            destination.rename(backup)
            try:
                staging.rename(destination)
            except BaseException:
                backup.rename(destination)
                raise
        else:
            staging.rename(destination)
    print(f"Installed Omabit panel snapshot: {destination}")


def main():
    parser = argparse.ArgumentParser()
    config = pathlib.Path(os.environ.get("XDG_CONFIG_HOME", pathlib.Path.home() / ".config"))
    parser.add_argument("--destination", type=pathlib.Path, default=config / "omarchy/plugins" / PLUGIN)
    parser.add_argument("--replace", action="store_true")
    parser.add_argument("--enable", action="store_true")
    args = parser.parse_args()
    if args.enable and args.destination.absolute() != config / "omarchy/plugins" / PLUGIN:
        raise RuntimeError("--enable requires the normal user plugin directory")
    install(args.destination, args.replace)
    if args.enable:
        subprocess.run(["omarchy", "shell", "shell", "rescanPlugins"], check=True)
        subprocess.run(["omarchy", "plugin", "enable", PLUGIN], check=True)
        print("Enabled Omabit. Click the ~+ bar widget to open it.")


if __name__ == "__main__":
    main()
