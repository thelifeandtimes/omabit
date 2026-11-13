#!/usr/bin/env python3
"""Private full-panel UI fixture; never contacts a runner or network."""
import json
from datetime import datetime, timezone
from pathlib import Path
import sys


def main():
    mode, command, *args = sys.argv[1:]
    if mode not in ("online", "offline", "observer", "conflict") or command not in ("ship-routes", "ship-routes-configure") or len(args) != 6:
        raise ValueError("Invalid ship URL fixture command")
    options = dict(zip(args[::2], args[1::2]))
    name, runner = options["--runner"], options["--expected-runner-id"]
    if name not in ("preview-local", "preview-remote") or runner != name + "-040":
        raise ValueError("Unknown fixture runner")
    if name == "preview-remote" and mode != "online":
        raise ValueError("The fixture remote runner is unavailable")
    u = json.loads(options["--request-json"]) if command == "ship-routes-configure" else None
    instance = u["selection"]["target"]["instance_id"] if u else options["--instance"]
    ships = {"preview-publisher": "~zod", "preview-project-nec": "~nec", "preview-remote-bus": "~bus"}
    if instance not in ships or (name == "preview-remote") != (instance == "preview-remote-bus"):
        raise ValueError("Wrong fixture ship")
    selection = {"target": {"instance_id": instance}, "runtime_id": "a" * 64, "config_revision": 1}
    binding = {"runner_id": runner, "target": selection["target"], "ship": ships[instance], "continuity": "c" * 64}
    path = Path("/data/state/ship-urls.json")
    state = json.loads(path.read_text()) if path.exists() else {}
    record = state.get(instance, {"version": 1, "binding": binding, "revision": 0, "settings": {"routes": [], "disabled_automatic": []}, "actor": "", "updated_at": "0001-01-01T00:00:00Z"})
    now = datetime.now(timezone.utc).isoformat()
    if u:
        if mode == "conflict":
            raise ValueError("Ship URLs changed; refresh before saving")
        update = u["update"]
        if u["selection"] != selection or update["binding"] != binding or update["expected_revision"] != record["revision"]:
            raise ValueError("Ship URLs changed; refresh before saving")
        record = {"version": 1, "binding": binding, "revision": record["revision"] + 1, "settings": update["settings"], "actor": "preview", "updated_at": now}
        state[instance] = record
        path.write_text(json.dumps(state, indent=2) + "\n")
    routes = list(record["settings"]["routes"])
    automatic = [
        {"id": "managed-http", "origin": "http://127.0.0.1:18080", "network": "local", "scope": runner, "source": "managed", "enabled": True},
        {"id": "tailscale-serve", "origin": "https://runner.preview.ts.net", "network": "tailnet", "scope": "preview.ts.net", "source": "tailscale", "enabled": True},
    ]
    for route in automatic:
        route["enabled"] = route["id"] not in record["settings"]["disabled_automatic"]
        if not any(r["origin"] == route["origin"] for r in routes):
            routes.append(route)
    routes.sort(key=lambda r: r["id"])
    print(json.dumps({"version": 1, "runner_id": runner, "mode": "simulation", "selection": selection, "record": record,
                      "registry": {"version": 1, "revision": record["revision"], "binding": binding, "routes": routes},
                      "network": {"tailscale": "ready", "dns_name": "runner.preview.ts.net", "tailnet_suffix": "preview.ts.net", "serve": "tailnet"}, "read_at": now}))


if __name__ == "__main__":
    try:
        main()
    except (KeyError, ValueError, TypeError) as error:
        print(json.dumps({"error": str(error)}), file=sys.stderr)
        sys.exit(1)
