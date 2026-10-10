#!/usr/bin/env python3
"""6Y: confirmed five-to-three-to-two pruning on the approved development comet.

Pins the existing 6W identity/pier/epoch baseline, prepares one additional epoch,
and preserves exact reviewed requests. Refuses previous 6Y work; never retries an
uncertain mutation or resumes after a failed step. Does not remove checkpoints.
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
PIER = "ee7b362a3bc23bc78f911ed1e08fb6e9"
BASELINE = ["0i0", "0i7046", "0i7064", "0i7082"]
PREFIX = "checkpoint-6y-"
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


def completed(args):
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


def operate(kind, suffix):
    return completed(["operate", INSTANCE, kind, "--revision", "1", "--request-id", PREFIX + suffix])


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
    assert inv["source"] == "vere-4.6-epoch-v2" and inv["pier_id"] == PIER
    assert re.fullmatch(r"[0-9a-f]{64}", inv["digest"])
    age = datetime.datetime.now(datetime.timezone.utc) - datetime.datetime.fromisoformat(inv["observed_at"].replace("Z", "+00:00"))
    assert -5 <= age.total_seconds() <= 300 and len(inv["entries"]) == count
    previous = -1
    for index, e in enumerate(inv["entries"]):
        assert re.fullmatch(r"0i(0|[1-9][0-9]*)", e["name"])
        number = int(e["name"][2:])
        assert previous < number <= 2**64 - 1
        previous = number
        assert e["protected"] == (index >= count - 2)
        assert not e["shared_files"] and e["logical_bytes"] > 0 and e["allocated_bytes"] >= 0
        assert e["file_count"] == (4 if number == 0 else 5)
        assert re.fullmatch(r"[0-9a-f]{64}", e["digest"])
        assert number == 0 or e.get("snapshot_bytes", 0) > 0
    return inv


def names(inv):
    return [e["name"] for e in inv["entries"]]


def main():
    if not __debug__:
        raise RuntimeError("This destructive development validator requires enabled assertions")
    hello = invoke(["hello"])
    assert hello["mode"] == "development-managed" and hello["access_role"] == "operator"
    assert {"comet-prune-prefix", "comet-epochs", "offline-roll", "epoch-inventory"} <= set(hello["features"])
    before = snapshot()
    assert not any(o["request_id"].startswith(PREFIX) for o in before["operations"]), "Prior 6Y work exists; review instead of rerunning"
    assert not any(i.get("active_operation_id") or i.get("recovery_required") or i.get("schedule", {}).get("config", {}).get("enabled") for i in before["instances"])
    policies = {i["id"]: i.get("schedule") for i in before["instances"]}
    previous_container = healthy()["resources"]["container_id"]
    replay = []
    operate("stop", "stop-prepare")
    replay.append(operate("inspect-epochs", "inspect-baseline"))
    assert names(inventory(selected(snapshot()), 4)) == BASELINE
    replay.append(operate("roll", "roll-prepare"))
    five = inventory(selected(snapshot()), 5)
    assert names(five)[:4] == BASELINE
    expected = names(five)
    operate("start", "resume-prepare")
    row = healthy()
    assert row["resources"]["container_id"] != previous_container
    previous_container = row["resources"]["container_id"]

    for round_number, cutoff in enumerate(("0i7046", "0i7064"), 1):
        operate("stop", "stop-" + str(round_number))
        replay.append(operate("inspect-epochs", "inspect-" + str(round_number)))
        inv = inventory(selected(snapshot()), len(expected))
        assert names(inv) == expected
        selected_count = 2 if round_number == 1 else 1
        preview = invoke(["epoch-preview", INSTANCE, "--revision", "1", "--expected-digest", inv["digest"], "--cutoff", cutoff])
        assert preview["runner_id"] == RUNNER and preview["instance_id"] == INSTANCE and preview["config_revision"] == 1
        assert preview["inventory_digest"] == inv["digest"] and preview["observed_at"] == inv["observed_at"] and preview["cutoff"] == cutoff
        assert preview["selected"] == inv["entries"][:selected_count] and preview["retained"] == inv["entries"][selected_count:]
        assert preview["deletion_supported"] and preview["requires_stopped_revalidation"] and not preview["shared_files"]
        assert preview["selected_logical_bytes"] == sum(e["logical_bytes"] for e in preview["selected"])
        assert preview["selected_allocated_bytes"] == sum(e["allocated_bytes"] for e in preview["selected"])
        args = ["epoch-prune", INSTANCE, "--revision", "1", "--expected-digest", inv["digest"], "--cutoff", cutoff,
                "--request-id", PREFIX + "prune-" + str(round_number), "--confirm-history-loss"]
        invoke(args[:-1], refusal=True)
        for field, value in (("--expected-digest", "0" * 64), ("--cutoff", expected[-1])):
            invalid = args.copy()
            invalid[invalid.index(field) + 1] = value
            invalid[invalid.index("--request-id") + 1] = PREFIX + "refuse-" + field[2:] + "-" + str(round_number)
            assert invoke(invalid, refusal=True) == "stale_or_unqualified_inventory"
        print(json.dumps({"confirmed_development_selection": preview}), flush=True)
        replay.append(completed(args))
        row = selected(snapshot())
        retained = inventory(row, len(expected) - selected_count)
        assert retained["entries"] == inv["entries"][selected_count:]
        p = row["pruning"]
        assert p["cutoff"] == cutoff and p["inventory_digest"] == inv["digest"]
        assert p.get("finished_at") and not p["uncertain"] and p["epoch_removed"]
        assert p["selected_epochs"] == p["removed_epochs"] == expected[:selected_count]
        assert p["retained_epochs"] == expected[selected_count:]
        assert p["removed_files"] == p["expected_files"] == sum(e["file_count"] for e in preview["selected"])
        expected = names(retained)
        print(json.dumps({"round": round_number, "pruning": p, "retained_inventory": retained}), flush=True)
        operate("start", "resume-" + str(round_number))
        row = healthy()
        assert row["resources"]["container_id"] != previous_container
        previous_container = row["resources"]["container_id"]
        assert invoke(args) == replay[-1][1]

    operate("stop", "stop-final-inspection")
    replay.append(operate("inspect-epochs", "inspect-final"))
    final_inventory = inventory(selected(snapshot()), 2)
    assert names(final_inventory) == expected and all(e["protected"] for e in final_inventory["entries"])
    assert invoke(["epoch-prune", INSTANCE, "--revision", "1", "--expected-digest", final_inventory["digest"],
                   "--cutoff", expected[0], "--request-id", PREFIX + "refuse-final-protected", "--confirm-history-loss"], refusal=True) == "stale_or_unqualified_inventory"
    print(json.dumps({"final_stopped_inventory": final_inventory}), flush=True)
    operate("start", "resume-final")
    final = healthy()
    assert final["resources"]["container_id"] != previous_container
    for args, op in replay:
        assert invoke(args) == op
    after = snapshot()
    assert not selected(after).get("epochs") and selected(after)["pruning"]["cutoff"] == "0i7064"
    assert {i["id"]: i.get("schedule") for i in after["instances"]} == policies
    for name in ("alpha", "panel-check", "import-check"):
        peer = next(i for i in after["instances"] if i["id"] == name)
        assert peer["observed_state"] == "running" and peer["health"] == "healthy"
    print(json.dumps({"gate_passed": True, "retained_epochs": expected, "instance": final,
                      "policies_unchanged": True, "incidents": after["incidents"]}, indent=2), flush=True)


if __name__ == "__main__":
    main()
