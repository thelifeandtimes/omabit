#!/usr/bin/env python3
"""Exercise the explicitly approved development import gate through pinned RPC.

Fixed request IDs make a repeated invocation reconcile historical operations,
never retry an uncertain copy/start with a new identity. No production input.
"""
import http.client
import json
import pathlib
import subprocess
import time

ROOT = pathlib.Path(__file__).resolve().parents[1]
RUNNER = "runner-f496feed1f7b08f31a5d82b94ac2f5fd"
SOURCE = "185ff2ed58b9abdd6861dfe64d6650f5"
DIGEST = "7d897a1765a4b0d0b511b2a3d5ec5d721aad07f413d0cdd219aca71a98908ff8"
BASE = [str(ROOT / "build/urbit-collector"), "--runner", "workstation", "--expected-runner-id", RUNNER]
IMPORT = ["import-fake", "import-check", "--source-run", SOURCE, "--expected-digest", DIGEST,
          "--http-port", "18086", "--request-id", "checkpoint-6p-import-check", "--confirm-stopped"]


def invoke(args, refused=False):
    p = subprocess.run(BASE + args, capture_output=True, text=True, timeout=15)
    if refused:
        assert p.returncode != 0 and not p.stdout.strip()
        return json.loads(p.stderr)["error"]["code"]
    if p.returncode:
        raise RuntimeError("Collector refused: " + p.stderr)
    return json.loads(p.stdout)


def snapshot():
    return invoke(["snapshot"])


def wait(op):
    deadline = time.monotonic() + 660
    while time.monotonic() < deadline:
        current = next(o for o in snapshot()["operations"] if o["id"] == op["id"])
        if current["status"] in ("succeeded", "failed", "interrupted"):
            print(json.dumps({"operation": current}), flush=True)
            assert current["status"] == "succeeded", "operation did not succeed; no automatic retry"
            return current
        time.sleep(1)
    raise RuntimeError("Operation deadline elapsed; inspect journal before any further action")


def selected():
    return next(i for i in snapshot()["instances"] if i["id"] == "import-check")


def operate(kind, request):
    i = selected()
    return wait(invoke(["operate", i["id"], kind, "--revision", str(i["config_revision"]), "--request-id", request]))


def healthy():
    i = selected()
    assert i["observed_state"] == "running" and i["health"] == "healthy"
    assert i["config_revision"] == i["applied_revision"] == 1 and not i.get("recovery_required")
    assert i["endpoint"] == "http://127.0.0.1:18086"
    c = http.client.HTTPConnection("127.0.0.1", 18086, timeout=5)
    try:
        c.request("GET", "/~/host")
        response = c.getresponse()
        assert response.status == 200 and response.read(128).strip() == b"~zod"
    finally:
        c.close()
    return i


def main():
    hello = invoke(["hello"])
    assert hello["mode"] in ("development-managed-fake", "development-managed") and hello["access_role"] == "operator"
    assert "import-qualified-fake" in hello["features"]
    source = next(s for s in invoke(["import-sources"])["sources"] if s["run_id"] == SOURCE)
    assert source["content_digest"] == DIGEST and source["ship"] == "~zod"
    assert source["files"] == 7 and source["bytes"] == 1016840204
    before = snapshot()
    existing = [i for i in before["instances"] if i["id"] == "import-check"]
    if existing:
        assert any(o["request_id"] == "checkpoint-6p-import-check" and o["actor"] == hello["actor"] for o in before["operations"])
    else:
        assert not source.get("claimed_by")
    imported = wait(invoke(IMPORT))
    state = selected()
    assert state["import"]["source_run_id"] == SOURCE and state["import"]["content_digest"] == DIGEST
    assert state["import"]["copied_files"] == state["import"]["expected_files"] == 7
    assert state["import"]["copied_bytes"] == state["import"]["expected_bytes"] == 1016840204
    assert not state["import"]["uncertain"] and not state.get("recovery_required")
    if not existing:
        assert state["observed_state"] == state["desired_state"] == "stopped" and state["applied_revision"] == 0
    assert invoke(IMPORT) == imported
    duplicate = IMPORT.copy()
    duplicate[1] = "import-duplicate"
    duplicate[duplicate.index("--http-port") + 1] = "18087"
    duplicate[duplicate.index("--request-id") + 1] = "checkpoint-6p-duplicate-source"
    refusal = invoke(duplicate, refused=True)
    assert refusal == "source_changed_or_claimed"
    source = next(s for s in invoke(["import-sources"])["sources"] if s["run_id"] == SOURCE)
    assert source["claimed_by"] == "import-check"
    print(json.dumps({"import_verified": True, "exact_replay": True, "duplicate_source_refused": refusal}), flush=True)
    operate("start", "checkpoint-6p-start-import-check")
    first = healthy()
    assert invoke(IMPORT) == imported
    operate("stop", "checkpoint-6p-stop-import-check")
    operate("start", "checkpoint-6p-resume-import-check")
    final = healthy()
    assert invoke(IMPORT) == imported
    after = snapshot()
    assert not any(i["id"] == "import-duplicate" for i in after["instances"])
    for name in ("alpha", "panel-check"):
        i = next(i for i in after["instances"] if i["id"] == name)
        assert i["observed_state"] == "running" and i["health"] == "healthy"
    print(json.dumps({"gate_passed": True, "runner_id": RUNNER, "instance": final,
                      "first_runtime_observation": first.get("observed_at"), "incidents": after["incidents"]}, indent=2))


if __name__ == "__main__":
    main()
