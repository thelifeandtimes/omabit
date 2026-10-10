#!/usr/bin/env python3
"""Run one approved timed fake-pier routine, then disable only our test policy.

Refuses an existing policy instead of overwriting user work. There are no manual
lifecycle submissions: the runner owns the timed stop/pack/resume operation.
"""
import datetime
import http.client
import json
import pathlib
import subprocess
import time

ROOT = pathlib.Path(__file__).resolve().parents[1]
RUNNER = "runner-f496feed1f7b08f31a5d82b94ac2f5fd"
INSTANCE = "import-check"
BASE = [str(ROOT / "build/urbit-collector"), "--runner", "workstation", "--expected-runner-id", RUNNER]


def invoke(args):
    p = subprocess.run(BASE + args, capture_output=True, text=True, timeout=15)
    if p.returncode:
        raise RuntimeError("Collector refused: " + p.stderr)
    return json.loads(p.stdout)


def snapshot():
    return invoke(["snapshot"])


def selected(s):
    return next(i for i in s["instances"] if i["id"] == INSTANCE)


def policy():
    return invoke(["routine", INSTANCE])["policy"]


def save(config, revision, instance_revision):
    return invoke(["routine-set", INSTANCE, "--revision", str(revision),
                   "--instance-revision", str(instance_revision),
                   "--enabled=" + str(config["enabled"]).lower(), "--action", config["action"],
                   "--days", ",".join(map(str, config["weekdays"])),
                   "--hour-utc", str(config["hour_utc"]), "--minute-utc", str(config["minute_utc"]),
                   "--window-minutes", str(config["window_minutes"]),
                   "--stop-running=" + str(config["stop_running"]).lower(),
                   "--resume=" + str(config["resume"]).lower()])


def main():
    hello = invoke(["hello"])
    assert hello["mode"] == "development-managed" and hello["access_role"] == "operator"
    assert "maintenance-schedules" in hello["features"] and policy() is None
    before = snapshot()
    i = selected(before)
    assert i["mode"] == "fake" and i["observed_state"] == "running" and i["health"] == "healthy"
    assert i["config_revision"] == i["applied_revision"] == 1
    assert not i.get("recovery_required") and not i.get("active_operation_id")
    assert i["resources"]["status"] == "fresh"
    original_container = i["resources"]["container_id"]
    slot = datetime.datetime.now(datetime.timezone.utc).replace(second=0, microsecond=0) + datetime.timedelta(minutes=2)
    config = dict(enabled=True, action="pack", weekdays=[(slot.weekday()+1) % 7],
                  hour_utc=slot.hour, minute_utc=slot.minute, window_minutes=5,
                  stop_running=True, resume=True)
    # No automatic re-submission if acceptance of this policy write is uncertain.
    created = save(config, 0, 1)
    stages = []
    try:
        assert created["revision"] == 1 and created["owner"] == hello["actor"]
        print(json.dumps({"policy_created": created, "expected_slot": slot.isoformat()}), flush=True)
        request = "schedule-" + created["id"] + "-" + str(int(slot.timestamp()))
        deadline = time.monotonic() + 1950
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
                assert operation["status"] == "succeeded", "routine did not succeed; inspect before any new work"
                break
            time.sleep(1)
        assert operation is not None, "routine deadline exceeded"
        # Let multiple scheduler ticks and a resource poll run; the consumed slot
        # must not dispatch a second operation.
        time.sleep(12)
        after = snapshot()
        matches = [o for o in after["operations"] if o["request_id"] == request]
        assert matches == [operation]
        final = selected(after)
        assert final["observed_state"] == final["desired_state"] == "running" and final["health"] == "healthy"
        assert final["applied_revision"] == final["config_revision"] == 1 and not final.get("recovery_required")
        assert final["routine"]["stage"] == "complete" and final["routine"]["succeeded"]
        assert final["routine"]["was_running"] and final["routine"]["resume"]
        assert final["maintenance"]["kind"] == "pack" and final["maintenance"].get("finished_at")
        assert final["resources"]["container_id"] != original_container
        c = http.client.HTTPConnection("127.0.0.1", 18086, timeout=5)
        try:
            c.request("GET", "/~/host")
            response = c.getresponse()
            assert response.status == 200 and response.read(128).strip() == b"~zod"
        finally:
            c.close()
        for name in ("alpha", "panel-check", "comet-check"):
            peer = next(i for i in after["instances"] if i["id"] == name)
            assert peer["observed_state"] == "running" and peer["health"] == "healthy"
        print(json.dumps({"cycle_passed": True, "observed_stages": stages, "instance": final,
                          "incidents": after["incidents"]}, indent=2), flush=True)
    finally:
        current = policy()
        row = selected(snapshot())
        if current and current["id"] == created["id"] and current["revision"] == created["revision"] and not row.get("active_operation_id"):
            disabled = save(dict(config, enabled=False), current["revision"], 1)
            assert not disabled["config"]["enabled"] and not disabled.get("next_slot")
            print(json.dumps({"test_policy_disabled": disabled}), flush=True)
        else:
            raise RuntimeError("Test policy could not be safely disabled; inspect active work/policy before editing")


if __name__ == "__main__":
    main()
