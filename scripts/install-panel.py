#!/usr/bin/env python3
"""Install an owned Omabit panel snapshot, preserving existing/user-modified files."""
import argparse
import hashlib
import json
import os
import pathlib
import shutil
import subprocess
import sys
import tempfile
import time

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
    desktop = ROOT / "build/omabit"
    if not binary.is_file() or not desktop.is_file():
        raise RuntimeError("build urbit-collector and omabit before installing the panel")
    with tempfile.TemporaryDirectory(prefix=".omabit-stage-", dir=destination.parent) as temporary:
        staging = pathlib.Path(temporary) / PLUGIN
        shutil.copytree(ROOT / "plugins/omabit", staging)
        (staging / "bin").mkdir()
        shutil.copy2(binary, staging / "bin/urbit-collector")
        shutil.copy2(desktop, staging / "bin/omabit")
        os.chmod(staging / "bin/urbit-collector", 0o755)
        os.chmod(staging / "bin/omabit", 0o755)
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


def enable_installed(timeout=10):
    """Wait for the live registry, not just the asynchronous rescan IPC reply."""
    deadline = time.monotonic() + timeout

    def command(args):
        remaining = deadline - time.monotonic()
        if remaining <= 0:
            raise RuntimeError("timed out waiting for the shell plugin registry")
        result = subprocess.run(["omarchy", *args], capture_output=True, text=True,
                                timeout=min(3, remaining))
        if result.returncode != 0:
            raise RuntimeError(result.stderr.strip() or "Omarchy shell command failed")
        return result.stdout

    try:
        command(["shell", "shell", "rescanPlugins"])
        next_scan = time.monotonic() + 1
        requested_enable = False
        while time.monotonic() < deadline:
            plugins = json.loads(command(["shell", "shell", "listPlugins"]))
            if not isinstance(plugins, list) or any(not isinstance(p, dict) for p in plugins):
                raise RuntimeError("invalid shell plugin registry response")
            entry = next((p for p in plugins if p.get("id") == PLUGIN), None)
            if entry is not None:
                if entry.get("enabled") is True:
                    return
                if not requested_enable:
                    command(["plugin", "enable", PLUGIN])
                    requested_enable = True
            elif not requested_enable and time.monotonic() >= next_scan:
                # Omarchy ignores rescan requests while another scan is active.
                # That scan may have started before the snapshot was installed.
                command(["shell", "shell", "rescanPlugins"])
                next_scan = time.monotonic() + 1
            remaining = deadline - time.monotonic()
            if remaining > 0:
                time.sleep(min(0.1, remaining))
        raise RuntimeError("timed out waiting for plugin discovery or enabled status")
    except (OSError, ValueError, subprocess.SubprocessError, RuntimeError) as error:
        raise RuntimeError(
            f"The panel is installed, but enabling was not confirmed: {error}\n"
            "Retry without reinstalling (from the checkout):\n"
            "  python3 -B scripts/install-panel.py --enable-only"
        ) from None


def main():
    parser = argparse.ArgumentParser()
    config = pathlib.Path(os.environ.get("XDG_CONFIG_HOME", pathlib.Path.home() / ".config"))
    parser.add_argument("--destination", type=pathlib.Path, default=config / "omarchy/plugins" / PLUGIN)
    parser.add_argument("--replace", action="store_true")
    parser.add_argument("--enable", action="store_true")
    parser.add_argument("--enable-only", action="store_true", help="enable an existing installation without copying files")
    args = parser.parse_args()
    if args.enable_only and (args.enable or args.replace):
        parser.error("--enable-only cannot be combined with --enable or --replace")
    if (args.enable or args.enable_only) and args.destination.absolute() != config / "omarchy/plugins" / PLUGIN:
        raise RuntimeError("--enable requires the normal user plugin directory")
    if args.enable_only:
        manifest = args.destination / "manifest.json"
        if not manifest.is_file() or json.loads(manifest.read_text()).get("id") != PLUGIN:
            raise RuntimeError("Omabit is not installed at the requested destination")
    else:
        install(args.destination, args.replace)
    if args.enable or args.enable_only:
        enable_installed()
        print("Enabled Omabit. Click the ~+ bar widget to open it.")


if __name__ == "__main__":
    try:
        main()
    except (OSError, ValueError, subprocess.SubprocessError, RuntimeError) as error:
        print(f"Omabit panel installation: {error}", file=sys.stderr)
        sys.exit(1)
