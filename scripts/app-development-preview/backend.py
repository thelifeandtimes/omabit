#!/usr/bin/env python3
"""Inert manual-review data adapter. Never imports or invokes an Omabit helper."""
import copy
import fcntl
import json
import os
import re
import sys
from datetime import datetime, timezone
from pathlib import Path


def initial_state():
    blocked = {"commit": "block", "publish": "block"}
    runners = []
    for name, targets in (
        ("preview-local", [
            {"target": {"instance_id": "preview-publisher"}, "ship": "~zod", "label": "Publisher fixture", "ready": True},
            {"target": {"instance_id": "preview-project-nec", "project_id": "preview-project", "generation": 1, "creator": "preview-owner"}, "ship": "~nec", "label": "Project member fixture", "ready": True},
        ]),
        ("preview-remote", [
            {"target": {"instance_id": "preview-remote-bus"}, "ship": "~bus", "label": "Remote fixture", "ready": True},
        ]),
    ):
        runner_id = name + "-040"
        runners.append({
            "name": name, "runner_id": runner_id, "can_edit": True,
            "rollout": "not enrolled in global rollout",
            "status": {
                "version": 1, "runner_id": runner_id, "mode": "simulation",
                "read_at": datetime.now(timezone.utc).isoformat(), "desk_operations": False,
                "policy": {"version": 1, "revision": 0, "global": dict(blocked),
                           "runner": {"commit": "inherit", "publish": "inherit"}, "entries": []},
                "targets": targets,
            },
        })
    return {"version": 1, "global": {"version": 1, "revision": 0, "gates": blocked, "runners": []}, "runners": runners}


def available(runner, mode):
    return runner["name"] == "preview-local" or mode == "online"


def view(state, mode):
    result = copy.deepcopy(state)
    for runner in result["runners"]:
        runner["status"]["read_at"] = datetime.now(timezone.utc).isoformat()
        if runner["name"] == "preview-remote" and mode != "online":
            runner["can_edit"] = False
            runner["error"] = "Preview runner is offline." if mode == "offline" else "Preview connection has observer access."
            if mode == "offline":
                runner["status"] = None
    return result


def rollout(state, mode):
    for entry in state["global"]["runners"]:
        if entry["applied"]:
            continue
        runner = next(r for r in state["runners"] if r["runner_id"] == entry["runner_id"])
        policy = runner["status"]["policy"]
        error = ""
        if not available(runner, mode):
            error = "Preview runner is offline." if mode == "offline" else "Preview connection has observer access."
        elif entry["expected_revision"] not in (None, policy["revision"]):
            error = "Preview policy changed. Refresh and save a new global rollout."
        if error:
            entry["error"] = runner["error"] = error
            runner["rollout"] = "pending"
            continue
        policy["global"] = dict(state["global"]["gates"])
        policy["revision"] += 1
        entry["applied"] = True
        entry["acknowledged"] = runner["acknowledged"] = {
            "revision": policy["revision"], "global": dict(policy["global"])}
        runner["rollout"] = "applied"
        entry.pop("error", None)
        runner.pop("error", None)


def apply(state, mode, command, request=None):
    if mode not in ("online", "offline", "observer"):
        raise ValueError("Unknown preview connection mode")
    if command == "app-development-status":
        return view(state, mode)
    if command not in ("app-development-save", "app-development-retry"):
        raise ValueError("Unsupported preview command")
    if not isinstance(request, dict) or request.get("revision") != state["global"]["revision"]:
        raise ValueError("Preview policy changed. Refresh before saving.")
    if command == "app-development-retry":
        rollout(state, mode)
        return view(state, mode)

    update = request["update"]
    scope, gates = update["scope"], update["gates"]
    if scope not in ("global", "runner", "ship", "desk"):
        raise ValueError("Unknown preview scope")
    allowed = ("allow", "block") if scope == "global" else ("allow", "block", "inherit")
    if set(gates) != {"commit", "publish"} or any(v not in allowed for v in gates.values()):
        raise ValueError("Invalid preview permissions")
    if scope == "global":
        entries = []
        for runner in state["runners"]:
            candidate = next((r for r in request["runners"] if r["runner_id"] == runner["runner_id"] and r["name"] == runner["name"]), None)
            if candidate is None:
                raise ValueError("Unknown preview runner")
            entry = {"name": runner["name"], "runner_id": runner["runner_id"],
                     "expected_revision": candidate["expected_revision"], "applied": False}
            if "acknowledged" in runner:
                entry["acknowledged"] = copy.deepcopy(runner["acknowledged"])
            entries.append(entry)
        state["global"].update(revision=request["revision"] + 1, gates=dict(gates), runners=entries)
        rollout(state, mode)
    else:
        runner = next((r for r in state["runners"] if r["runner_id"] == request.get("runner_id") and r["name"] == request.get("name")), None)
        if runner is None or not available(runner, mode):
            raise ValueError("An online preview operator connection is required")
        policy = runner["status"]["policy"]
        if update["expected_revision"] != policy["revision"]:
            raise ValueError("Preview runner policy changed. Refresh before saving.")
        if scope == "runner":
            policy["runner"] = dict(gates)
        else:
            target = copy.deepcopy(update["target"])
            ship = {k: v for k, v in target.items() if k != "desk"}
            if not any(ship == t["target"] for t in runner["status"]["targets"]):
                raise ValueError("Unknown preview ship")
            if scope == "desk" and not re.fullmatch(r"[a-z][a-z0-9-]{0,63}", target.get("desk", "")):
                raise ValueError("Invalid preview desk")
            if scope == "ship" and "desk" in target:
                raise ValueError("Unexpected desk for ship scope")
            policy["entries"] = [e for e in policy["entries"] if e["target"] != target]
            if any(value != "inherit" for value in gates.values()):
                policy["entries"].append({"target": target, "gates": dict(gates)})
        policy["revision"] += 1
    return view(state, mode)


def main():
    # A fixed path inside the preview namespace; no production configuration or
    # fallback transport. Even unknown commands fail before touching this file.
    mode, command, *args = sys.argv[1:]
    if command not in ("app-development-status", "app-development-save", "app-development-retry"):
        raise ValueError("Unsupported preview command")
    request = None
    if command != "app-development-status":
        if len(args) != 2 or args[0] != "--request-json":
            raise ValueError("Expected a preview request")
        request = json.loads(args[1])
    elif args:
        raise ValueError("Unexpected status arguments")
    path = Path("/data/state/policy.json")
    with path.with_suffix(".lock").open("a") as lock:
        fcntl.flock(lock, fcntl.LOCK_EX)
        state = json.loads(path.read_text()) if path.exists() else initial_state()
        result = apply(state, mode, command, request)
        if not path.exists() or command != "app-development-status":
            temporary = path.with_suffix(".tmp")
            with temporary.open("w") as output:
                json.dump(state, output, indent=2)
                output.write("\n")
                output.flush()
                os.fsync(output.fileno())
            temporary.replace(path)
        print(json.dumps(result))


if __name__ == "__main__":
    try:
        main()
    except (ValueError, KeyError, StopIteration, TypeError) as error:
        print(str(error), file=sys.stderr)
        sys.exit(1)
