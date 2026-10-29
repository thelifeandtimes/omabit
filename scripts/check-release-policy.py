#!/usr/bin/env python3
"""Inspect PR blobs as data. Never check out or execute an untrusted PR head."""
import argparse
import hashlib
import json
import re
import subprocess

from release_policy import (backend_path, validate_master_promotion,
                            validate_distribution_changes, validate_upgrade_paths)


def git(*args):
    return subprocess.check_output(["git", *args], text=True, timeout=30).strip()


def document(commit, name):
    data = git("show", commit + ":" + name)
    if len(data) > 131072:
        raise ValueError("release policy document exceeds its bound: " + name)
    return json.loads(data)


def frozen_backend_digest(commit):
    files = {}
    raw = subprocess.check_output(["git", "ls-tree", "-rz", "--full-tree", commit], timeout=30)
    for entry in raw.split(b"\0"):
        if not entry:
            continue
        attributes, path = entry.split(b"\t", 1)
        name = path.decode()
        if not backend_path(name):
            continue
        mode, kind, oid = attributes.split()
        if kind != b"blob" or mode not in (b"100644", b"100755"):
            raise ValueError("unsupported backend source input")
        data = subprocess.check_output(["git", "cat-file", "blob", oid.decode()], timeout=30)
        files[name] = {"sha256": hashlib.sha256(data).hexdigest(), "executable": mode == b"100755"}
    return hashlib.sha256(json.dumps(files, sort_keys=True, separators=(",", ":")).encode()).hexdigest()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--base", required=True)
    parser.add_argument("--head", required=True)
    parser.add_argument("--base-branch", required=True)
    parser.add_argument("--head-branch", required=True)
    args = parser.parse_args()
    if args.base_branch != "master":
        print("No stable promotion: edge/develop retain their documented development contracts.")
        return
    if not all(re.fullmatch(r"[0-9a-f]{40}", value) for value in (args.base, args.head)):
        parser.error("use exact source commit IDs")
    base = document(args.base, "manifest.json")
    root = document(args.head, "manifest.json")
    component = document(args.head, "plugins/omabit/manifest.json")
    validate_master_promotion(base["version"], root["version"], args.head_branch)
    if root.get("releaseChannel") != "stable" or component.get("releaseChannel") != "stable" or component["version"] != root["version"]:
        raise ValueError("stable root/component version and channel must agree")
    ui = git("show", args.head + ":plugins/omabit/Service.qml")
    if 'property string applicationVersion: ' + json.dumps(root["version"]) not in ui:
        raise ValueError("compiled UI version differs from stable release")
    pins = document(args.head, "native/bootstrap/release.json")
    if pins["version"] != root["version"] or not re.fullmatch(r"[0-9a-f]{40}", pins["source_commit"]):
        raise ValueError("final release pins must name this stable version and its frozen source")
    subprocess.run(["git", "merge-base", "--is-ancestor", pins["source_commit"], args.head], check=True, timeout=30)
    changed = git("diff", "--name-only", pins["source_commit"], args.head).splitlines()
    validate_distribution_changes(changed)
    record = document(args.head, "docs/releases/" + root["version"] + "/qualification.json")
    if record.get("version") != root["version"] or record.get("source_commit") != pins["source_commit"]:
        raise ValueError("qualification must bind the frozen release source")
    upgrade = record.get("stable_upgrade", {})
    if upgrade.get("passed") is not True or upgrade.get("non_destructive") is not True or not re.fullmatch(r"[0-9a-f]{64}", upgrade.get("evidence_sha256", "")):
        raise ValueError("stable promotion requires recorded non-destructive forward-upgrade evidence")
    validate_upgrade_paths(upgrade, base["version"], root["version"], pins["source_commit"],
        lambda path: subprocess.check_output(["git", "show", args.head + ":" + path], timeout=30))
    if not re.fullmatch(r"[0-9a-f]{64}", pins.get("backend_source_sha256", "")) or record.get("backend_source_sha256") != pins["backend_source_sha256"]:
        raise ValueError("backend source identity must match qualification and release pins")
    if frozen_backend_digest(pins["source_commit"]) != pins["backend_source_sha256"]:
        raise ValueError("backend source digest differs from the frozen commit")
    print("PASS: qualified stable promotion, matching frozen artifacts and tested upgrade path")


if __name__ == "__main__":
    main()
