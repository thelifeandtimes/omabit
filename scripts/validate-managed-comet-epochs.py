#!/usr/bin/env python3
"""6W: three explicit managed comet roll/replay/resume cycles and safe previews.

Fixed development instance/identity/request IDs. Refuses prior 6W work and active
policies. No retry, policy edit, deletion, or automatic resume after a failed step.
"""
import datetime
import http.client
import json
import pathlib
import re
import subprocess
import time

ROOT = pathlib.Path(__file__).resolve().parents[1]
RUNNER = "runner-f496feed1f7b08f31a5d82b94ac2f5fd"
INSTANCE = "comet-check"
SHIP = "~nolpel-modsyd-livsep-novmeg--sallen-tassul-padlud-litzod"
PREFIX = "checkpoint-6w-"
BASE = [str(ROOT / "build/urbit-collector"), "--runner", "workstation", "--expected-runner-id", RUNNER]


def invoke(args, refusal=False):
    p = subprocess.run(BASE + args, capture_output=True, text=True, timeout=15)
    if refusal:
        assert p.returncode != 0 and not p.stdout.strip()
        return json.loads(p.stderr)["error"]["code"]
    if p.returncode:
        raise RuntimeError("Collector refused; inspect the journal before further mutation: " + p.stderr)
    return json.loads(p.stdout)


def snapshot():
    return invoke(["snapshot"])


def selected(s):
    return next(i for i in s["instances"] if i["id"] == INSTANCE)


def operate(kind, suffix):
    args = ["operate", INSTANCE, kind, "--revision", "1", "--request-id", PREFIX + suffix]
    accepted = invoke(args)
    deadline = time.monotonic() + 1250
    while time.monotonic() < deadline:
        matches = [o for o in snapshot()["operations"] if o["id"] == accepted["id"]]
        assert len(matches) == 1
        op = matches[0]
        if op["status"] in ("succeeded", "failed", "interrupted"):
            print(json.dumps({"operation": op}), flush=True)
            assert op["status"] == "succeeded", "operation failed; no automatic resume or retry"
            assert invoke(args) == op
            return args, op
        time.sleep(1)
    raise RuntimeError("Operation deadline exceeded; reconcile its fixed request ID before proceeding")


def healthy():
    deadline = time.monotonic() + 35
    while time.monotonic() < deadline:
        row = selected(snapshot())
        assert row["mode"] == "comet" and row["ship"] == SHIP
        assert row["desired_state"] == row["observed_state"] == "running" and row["health"] == "healthy"
        assert row["config_revision"] == row["applied_revision"] == 1
        assert not row.get("active_operation_id") and not row.get("recovery_required")
        assert not row.get("epochs") and not row.get("roll_requires_run")
        if row.get("resources", {}).get("status") == "fresh":
            break
        time.sleep(1)
    else:
        raise RuntimeError("Comet telemetry did not become fresh")
    assert row["endpoint"] == "http://127.0.0.1:18087"
    assert row["resources"]["memory_limit_bytes"] == 4 << 30
    c = http.client.HTTPConnection("127.0.0.1", 18087, timeout=5)
    try:
        c.request("GET", "/~/host")
        response = c.getresponse()
        assert response.status == 200 and response.read(128).decode().strip() == SHIP
    finally:
        c.close()
    return row


def inventory(row, count):
    assert row["mode"] == "comet" and row["ship"] == SHIP
    assert row["desired_state"] == row["observed_state"] == "stopped"
    assert not row.get("active_operation_id") and not row.get("recovery_required")
    inv = row["epochs"]
    assert inv["source"] == "vere-4.6-epoch-v2" and re.fullmatch(r"[0-9a-f]{64}", inv["digest"])
    assert re.fullmatch(r"[0-9a-f]{32}", inv["pier_id"])
    age = datetime.datetime.now(datetime.timezone.utc) - datetime.datetime.fromisoformat(inv["observed_at"].replace("Z", "+00:00"))
    assert -5 <= age.total_seconds() <= 300
    entries = inv["entries"]
    assert len(entries) == count and entries[0]["name"] == "0i0"
    previous = -1
    for index, e in enumerate(entries):
        assert re.fullmatch(r"0i(0|[1-9][0-9]*)", e["name"])
        number = int(e["name"][2:])
        assert previous < number <= 2**64 - 1
        previous = number
        assert e["protected"] == (index >= count - 2)
        assert e["logical_bytes"] > 0 and e["allocated_bytes"] >= 0
        assert e["file_count"] == (4 if number == 0 else 5)
        assert re.fullmatch(r"[0-9a-f]{64}", e["digest"])
        assert number == 0 or e.get("snapshot_bytes", 0) > 0
    return inv


def names(inv):
    return [e["name"] for e in inv["entries"]]


def main():
    hello = invoke(["hello"])
    assert hello["mode"] == "development-managed" and hello["access_role"] == "operator"
    assert {"comet-epochs", "offline-roll", "epoch-inventory"} <= set(hello["features"])
    before = snapshot()
    assert not any(o["request_id"].startswith(PREFIX) for o in before["operations"]), "Prior 6W work exists; review instead of rerunning"
    assert not any(i.get("active_operation_id") or i.get("recovery_required") or i.get("schedule", {}).get("config", {}).get("enabled") for i in before["instances"])
    policies = {i["id"]: i.get("schedule") for i in before["instances"]}
    previous_runtime = healthy()["resources"]["container_id"]
    expected = ["0i0"]
    original_pier = None
    replay = []
    for round_number in range(1, 4):
        operate("stop", "stop-" + str(round_number))
        replay.append(operate("inspect-epochs", "inspect-" + str(round_number)))
        initial = inventory(selected(snapshot()), round_number)
        assert names(initial) == expected
        original_pier = original_pier or initial["pier_id"]
        assert initial["pier_id"] == original_pier
        replay.append(operate("roll", "roll-" + str(round_number)))
        row = selected(snapshot())
        rolled = inventory(row, round_number + 1)
        assert rolled["pier_id"] == original_pier and names(rolled)[:-1] == expected
        m = row["maintenance"]
        assert m["kind"] == "roll" and m.get("finished_at") and row["roll_requires_run"]
        assert m["epochs_before"] == expected and m["epochs_after"] == names(rolled) and m["created_epoch"] == names(rolled)[-1]
        assert m["mem_available_bytes"] >= 5 << 30 and m["disk_available_bytes"] >= 8 << 30
        assert invoke(["operate", INSTANCE, "roll", "--revision", "1", "--request-id", PREFIX + "refuse-repeat-" + str(round_number)], refusal=True) == "invalid_state"
        print(json.dumps({"roll": round_number, "maintenance": m, "inventory": rolled}), flush=True)
        expected = names(rolled)
        operate("start", "resume-" + str(round_number))
        current = healthy()
        assert current["resources"]["container_id"] != previous_runtime
        previous_runtime = current["resources"]["container_id"]
        assert invoke(replay[-1][0]) == replay[-1][1]

    operate("stop", "stop-final-inspection")
    replay.append(operate("inspect-epochs", "inspect-final"))
    final_inventory = inventory(selected(snapshot()), 4)
    assert names(final_inventory) == expected and final_inventory["pier_id"] == original_pier
    preview_args = ["epoch-preview", INSTANCE, "--revision", "1", "--expected-digest", final_inventory["digest"], "--cutoff", "0i0"]
    preview = invoke(preview_args)
    assert preview["runner_id"] == RUNNER and preview["instance_id"] == INSTANCE and preview["config_revision"] == 1
    assert preview["selected"] == final_inventory["entries"][:1] and preview["retained"] == final_inventory["entries"][1:]
    assert not preview["deletion_supported"] and preview["requires_stopped_revalidation"]
    assert invoke(preview_args[:-1] + [expected[-1]], refusal=True) == "client_error"
    assert invoke(["epoch-prune", INSTANCE, "--revision", "1", "--expected-digest", final_inventory["digest"],
                   "--cutoff", "0i0", "--request-id", PREFIX + "refuse-prune", "--confirm-history-loss"], refusal=True) == "unsupported_for_instance"
    print(json.dumps({"preview_only": preview, "confirmed_deletion_refused": True}), flush=True)
    operate("start", "resume-final")
    final = healthy()
    assert final["resources"]["container_id"] != previous_runtime
    for args, op in replay:
        assert invoke(args) == op
    after = snapshot()
    assert not selected(after).get("epochs"), "historical replay resurrected stopped inventory"
    assert {i["id"]: i.get("schedule") for i in after["instances"]} == policies
    for name in ("alpha", "panel-check", "import-check"):
        peer = next(i for i in after["instances"] if i["id"] == name)
        assert peer["observed_state"] == "running" and peer["health"] == "healthy"
    print(json.dumps({"gate_passed": True, "retained_epochs": expected, "instance": final,
                      "policies_unchanged": True, "incidents": after["incidents"]}, indent=2), flush=True)


if __name__ == "__main__":
    main()
