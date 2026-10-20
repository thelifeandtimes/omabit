#!/usr/bin/env python3
"""Opt in to an explicit native plugin branch on unmodified stock Omarchy."""
import argparse
import json
import os
from pathlib import Path
import subprocess

CHANNELS = {"stable": "master", "edge": "edge", "develop": "develop"}
REPOSITORIES = {"https://github.com/thelifeandtimes/omabit.git", "https://github.com/thelifeandtimes/omabit"}
PLUGIN = "thelifeandtimes.omabit"


def command(argv, directory=None, check=True):
    env = dict(os.environ, GIT_TERMINAL_PROMPT="0")
    result = subprocess.run(argv, cwd=directory, env=env, capture_output=True, text=True, timeout=120)
    if check and result.returncode:
        raise RuntimeError(result.stderr.strip() or "command failed: " + " ".join(argv))
    return result


def git(directory, *args, run=command):
    return run(["git", *args], directory).stdout.strip()


def current(directory, run=command):
    if git(directory, "status", "--porcelain=v1", "--untracked-files=all", run=run):
        raise ValueError("plugin has local changes; preserve and review them before a channel update")
    if git(directory, "remote", "get-url", "origin", run=run) not in REPOSITORIES:
        raise ValueError("this helper expects the canonical Omabit origin; it never rewrites remotes")
    manifest = json.loads(git(directory, "show", "HEAD:manifest.json", run=run))
    if manifest.get("id") != PLUGIN:
        raise ValueError("checkout is not the Omabit native plugin")
    return git(directory, "branch", "--show-current", run=run)


def ensure_idle(run=command):
    plugins = json.loads(run(["omarchy", "shell", "shell", "listPlugins"]).stdout)
    if not isinstance(plugins, list):
        raise ValueError("invalid plugin registry response")
    enabled = any(row.get("id") == PLUGIN and row.get("enabled") is True for row in plugins)
    if not enabled:
        return
    state = json.loads(run(["omarchy", "shell", PLUGIN + ".bootstrap", "status"]).stdout)
    if state.get("busy") is not False or state.get("phase") != "ready":
        raise ValueError("wait for the active desktop setup/submission/activation before changing channels")


def update(directory, channel=None, run=command):
    branch = current(directory, run)
    target = CHANNELS.get(channel) if channel else branch
    if target not in CHANNELS.values():
        raise ValueError("select stable, edge or develop explicitly; detached/topic branches are preserved")
    ensure_idle(run)
    # Explicit remote branch, never remote HEAD. Stock Omarchy's updater uses
    # HEAD and therefore cannot be repurposed by changing tracking configuration.
    git(directory, "fetch", "--no-tags", "origin", "refs/heads/" + target + ":refs/remotes/origin/" + target, run=run)
    expected = next(name for name, value in CHANNELS.items() if value == target)
    manifest = json.loads(git(directory, "show", "refs/remotes/origin/" + target + ":manifest.json", run=run))
    if manifest.get("id") != PLUGIN or manifest.get("releaseChannel", "stable") != expected:
        raise ValueError("remote branch does not identify the selected distribution channel")
    if branch != target:
        exists = run(["git", "show-ref", "--verify", "--quiet", "refs/heads/" + target], directory, check=False).returncode == 0
        if exists:
            # Preflight the FF before switching, retaining the current branch if
            # a local target has diverged from the channel's remote history.
            run(["git", "merge-base", "--is-ancestor", target, "refs/remotes/origin/" + target], directory)
            git(directory, "switch", target, run=run)
        else:
            git(directory, "switch", "--create", target, "--track", "origin/" + target, run=run)
    git(directory, "merge", "--ff-only", "refs/remotes/origin/" + target, run=run)
    run(["omarchy", "plugin", "validate", str(directory)])
    run(["omarchy", "shell", "shell", "rescanPlugins"])
    return {"channel": expected, "branch": target, "commit": git(directory, "rev-parse", "HEAD", run=run),
            "activation": "requested; inspect bootstrap status for the actually loaded release"}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest="action", required=True)
    sub.add_parser("status")
    sub.add_parser("update")
    switch = sub.add_parser("switch")
    switch.add_argument("channel", choices=CHANNELS)
    args = parser.parse_args()
    directory = Path.home() / ".config/omarchy/plugins" / PLUGIN
    if directory.is_symlink() or not directory.is_dir():
        parser.error("install the native plugin checkout first")
    if args.action == "status":
        branch = current(directory)
        result = {"branch": branch, "channel": next((key for key, value in CHANNELS.items() if value == branch), "unqualified-topic")}
    else:
        result = update(directory, args.channel if args.action == "switch" else None)
    print(json.dumps(result, indent=2))


if __name__ == "__main__":
    try:
        main()
    except (ValueError, RuntimeError, OSError, subprocess.TimeoutExpired) as error:
        raise SystemExit(str(error))
