#!/usr/bin/env python3
"""6U: manual pack/replay/resume and one timed meld on the approved comet-check.

Refuses an existing policy or previous 6U work. Mutations are never automatically
retried. Disable only our exact temporary policy once its operation is idle.
"""
import datetime
import http.client
import json
import pathlib
import subprocess
import time

ROOT = pathlib.Path(__file__).resolve().parents[1]
RUNNER = "runner-f496feed1f7b08f31a5d82b94ac2f5fd"
INSTANCE = "comet-check"
SHIP = "~nolpel-modsyd-livsep-novmeg--sallen-tassul-padlud-litzod"
PREFIX = "checkpoint-6u-"
BASE = [str(ROOT / "build/urbit-collector"), "--runner", "workstation", "--expected-runner-id", RUNNER]


def invoke(args, refusal=False):
    p = subprocess.run(BASE + args, capture_output=True, text=True, timeout=15)
    if refusal:
        assert p.returncode != 0 and not p.stdout.strip()
        return json.loads(p.stderr)["error"]["code"]
    if p.returncode:
        raise RuntimeError("Collector refused; inspect the journal before any further mutation: " + p.stderr)
    return json.loads(p.stdout)


def snapshot():
    return invoke(["snapshot"])


def selected(s):
    return next(i for i in s["instances"] if i["id"] == INSTANCE)


def policy():
    return invoke(["routine", INSTANCE])["policy"]


def healthy():
    deadline = time.monotonic() + 35
    while time.monotonic() < deadline:
        row = selected(snapshot())
        assert row["mode"] == "comet" and row["ship"] == SHIP
        assert row["observed_state"] == row["desired_state"] == "running" and row["health"] == "healthy"
        assert row["config_revision"] == row["applied_revision"] == 1
        assert not row.get("recovery_required") and not row.get("active_operation_id")
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


def operate(kind):
    args = ["operate", INSTANCE, kind, "--revision", "1", "--request-id", PREFIX + kind + "-comet-check"]
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
            return op
        time.sleep(1)
    raise RuntimeError("Operation deadline elapsed; reconcile its fixed request ID before proceeding")


def maintenance(row, kind):
    m = row["maintenance"]
    assert m["kind"] == kind and m.get("finished_at")
    assert m["before_pier_bytes"] > 0 and m["after_pier_bytes"] > 0
    assert m["mem_available_bytes"] >= 5 << 30 and m["disk_available_bytes"] >= 8 << 30
    return m


def save(config, revision):
    return invoke(["routine-set", INSTANCE, "--revision", str(revision), "--instance-revision", "1",
                   "--enabled=" + str(config["enabled"]).lower(), "--action", config["action"],
                   "--days", ",".join(map(str, config["weekdays"])), "--hour-utc", str(config["hour_utc"]),
                   "--minute-utc", str(config["minute_utc"]), "--window-minutes", str(config["window_minutes"]),
                   "--stop-running=" + str(config["stop_running"]).lower(),
                   "--resume=" + str(config["resume"]).lower()])


def main():
    hello = invoke(["hello"])
    assert hello["mode"] == "development-managed" and hello["access_role"] == "operator"
    assert {"comet-compaction", "maintenance-schedules", "offline-pack", "offline-meld"} <= set(hello["features"])
    assert policy() is None, "Existing policy must not be replaced by this qualification"
    before = snapshot()
    assert not any(o["request_id"].startswith(PREFIX) for o in before["operations"]), "Previous 6U work exists; review instead of rerunning"
    assert not any(i.get("active_operation_id") or i.get("recovery_required") for i in before["instances"])
    first = healthy()
    for kind in ("roll", "inspect-epochs"):
        assert invoke(["operate", INSTANCE, kind, "--revision", "1", "--request-id", PREFIX + "refuse-" + kind], refusal=True) == "unsupported_for_instance"
    assert invoke(["operate", INSTANCE, "pack", "--revision", "1", "--request-id", PREFIX + "refuse-running-pack"], refusal=True) == "invalid_state"
    operate("stop")
    stopped = selected(snapshot())
    assert stopped["desired_state"] == stopped["observed_state"] == "stopped" and stopped["ship"] == SHIP
    packed = operate("pack")
    stopped = selected(snapshot())
    assert stopped["observed_state"] == "stopped" and not stopped.get("recovery_required")
    print(json.dumps({"manual_pack": maintenance(stopped, "pack")}), flush=True)
    operate("start")
    resumed = healthy()
    assert resumed["resources"]["container_id"] != first["resources"]["container_id"]
    assert invoke(["operate", INSTANCE, "pack", "--revision", "1", "--request-id", PREFIX + "pack-comet-check"]) == packed

    slot = datetime.datetime.now(datetime.timezone.utc).replace(second=0, microsecond=0) + datetime.timedelta(minutes=2)
    config = dict(enabled=True, action="meld", weekdays=[(slot.weekday()+1) % 7],
                  hour_utc=slot.hour, minute_utc=slot.minute, window_minutes=5, stop_running=True, resume=True)
    created = save(config, 0)
    try:
        assert created["revision"] == 1 and created["owner"] == hello["actor"]
        print(json.dumps({"policy_created": created, "expected_slot": slot.isoformat()}), flush=True)
        request = "schedule-" + created["id"] + "-" + str(int(slot.timestamp()))
        deadline = time.monotonic() + 1950
        stages = []
        operation = None
        while time.monotonic() < deadline:
            current = snapshot()
            row = selected(current)
            stage = row.get("routine", {}).get("stage")
            if stage and (not stages or stages[-1] != stage):
                stages.append(stage)
                print(json.dumps({"routine_stage": stage, "observed_state": row["observed_state"]}), flush=True)
            matches = [o for o in current["operations"] if o["request_id"] == request]
            assert len(matches) <= 1
            if matches and matches[0]["status"] in ("succeeded", "failed", "skipped", "interrupted"):
                operation = matches[0]
                print(json.dumps({"operation": operation}), flush=True)
                assert operation["status"] == "succeeded", "routine failed; inspect before any further work"
                assert operation["actor"] == hello["actor"] and operation["kind"] == "routine-meld"
                break
            time.sleep(1)
        assert operation is not None, "routine deadline exceeded"
        time.sleep(12)
        after = snapshot()
        assert [o for o in after["operations"] if o["request_id"] == request] == [operation]
        final = healthy()
        assert final["routine"]["stage"] == "complete" and final["routine"]["succeeded"]
        assert final["routine"]["was_running"] and final["routine"]["resume"]
        assert final["resources"]["container_id"] != resumed["resources"]["container_id"]
        maintenance(final, "meld")
        for name in ("alpha", "panel-check", "import-check"):
            peer = next(i for i in after["instances"] if i["id"] == name)
            assert peer["observed_state"] == "running" and peer["health"] == "healthy"
        print(json.dumps({"gate_passed": True, "observed_stages": stages, "instance": final,
                          "incidents": after["incidents"]}, indent=2), flush=True)
    finally:
        current = policy()
        row = selected(snapshot())
        if current and current["id"] == created["id"] and current["revision"] == created["revision"] and not row.get("active_operation_id"):
            disabled = save(dict(config, enabled=False), current["revision"])
            assert not disabled["config"]["enabled"] and not disabled.get("next_slot")
            print(json.dumps({"test_policy_disabled": disabled}), flush=True)
        else:
            raise RuntimeError("Test policy could not be safely disabled; inspect active work and policy before editing")


if __name__ == "__main__":
    main()
