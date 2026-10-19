#!/usr/bin/env python3
"""Explicit desktop/profile controls. Does not upgrade or reset runner hosts."""
import argparse
import json
import os
from pathlib import Path
import subprocess
import sys

sys.dont_write_bytecode = True
from desktop import Desktop
import runtime


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__, epilog="Build a local candidate with: python3 -B scripts/omabit-dev.py build --help")
    parser.add_argument("--pins", type=Path, help="explicit release descriptor for isolated qualification")
    commands = parser.add_subparsers(dest="command", required=True)
    commands.add_parser("status")
    commands.add_parser("resolve")
    commands.add_parser("lease")
    create = commands.add_parser("profile-create")
    create.add_argument("name")
    create.add_argument("--host-setup", action="store_true", help="allow explicit local host setup, e.g. inside a disposable guest")
    select = commands.add_parser("select")
    select.add_argument("profile")
    select.add_argument("--release", help="imported release digest, or follow for installed plugin pins")
    commands.add_parser("retry")
    imported = commands.add_parser("import-release")
    imported.add_argument("metadata", type=Path)
    imported.add_argument("archive", type=Path)
    prepare = commands.add_parser("prepare")
    prepare.add_argument("--request")
    prepare.add_argument("--release")
    prepare.add_argument("--profile")
    for name in ("begin", "commit", "fail"):
        child = commands.add_parser(name)
        child.add_argument("--session", required=True)
        child.add_argument("--request", required=True)
        child.add_argument("--message", default="")
    execute = commands.add_parser("exec")
    execute.add_argument("--profile")
    execute.add_argument("tool", choices=runtime.COMMANDS)
    execute.add_argument("arguments", nargs=argparse.REMAINDER)
    args = parser.parse_args(argv)
    desktop = Desktop(args.pins)
    if args.command == "status":
        result = desktop.status()
    elif args.command == "resolve":
        result = desktop.resolve()
    elif args.command == "profile-create":
        result = desktop.create_profile(args.name, args.host_setup)
    elif args.command == "select":
        desktop.select(args.profile, args.release)
        result = desktop.resolve()
    elif args.command == "retry":
        desktop.select(desktop.status()["selected"])
        result = desktop.resolve()
    elif args.command == "import-release":
        result = desktop.import_release(args.metadata, args.archive)
    elif args.command == "prepare":
        result = desktop.prepare(args.request, args.release, args.profile)
    elif args.command == "lease":
        with desktop.lease() as session:
            print(json.dumps({"session": session}), flush=True)
            for _ in sys.stdin:
                pass
        return 0
    elif args.command in ("begin", "commit", "fail"):
        desktop.activation(args.command, args.session, args.request, args.message)
        result = {"status": args.command, "request": args.request}
    else:
        state = desktop.status()
        name = args.profile or state["selected"]
        item = state["profiles"][name]
        key = item["selection"] if item["selection"] != "follow" else None
        bundle = desktop.prepare(release_id=key, name=name)
        arguments = args.arguments[1:] if args.arguments[:1] == ["--"] else args.arguments
        if args.tool == "omabit" and "setup" in arguments and not bundle["host_setup"]:
            raise ValueError("local host setup is disabled in this test profile")
        return subprocess.call([str(Path(bundle["directory"]) / "bin" / args.tool), *arguments],
                               env=dict(os.environ, **bundle["environment"]))
    print(json.dumps(result, sort_keys=True))
    return 0


if __name__ == "__main__":
    try:
        sys.exit(main())
    except (OSError, ValueError, KeyError, TypeError, runtime.BootstrapError) as error:
        print(json.dumps({"error": str(error)}), file=sys.stderr)
        sys.exit(1)
