#!/usr/bin/env python3
"""Fixed-request-ID live gate for a newly registered development comet.

Does not accept a key, existing identity or pier. Prints public journal/identity
evidence only. Secret code responses stay in memory and are never printed.
"""
import http.client
import json
import pathlib
import re
import subprocess
import time

ROOT = pathlib.Path(__file__).resolve().parents[1]
RUNNER = "runner-f496feed1f7b08f31a5d82b94ac2f5fd"
BASE = [str(ROOT / "build/urbit-collector"), "--runner", "workstation", "--expected-runner-id", RUNNER]
REGISTER = ["create-comet", "comet-check", "--http-port", "18087", "--request-id",
            "checkpoint-6r-register-comet-check", "--confirm-network"]
COMET = re.compile(r"~[a-z]{6}(?:-[a-z]{6}){3}--[a-z]{6}(?:-[a-z]{6}){3}")


def invoke(args, refusal=False):
    p = subprocess.run(BASE + args, capture_output=True, text=True, timeout=15)
    if refusal:
        assert p.returncode != 0 and not p.stdout.strip()
        return json.loads(p.stderr)["error"]["code"]
    if p.returncode:
        raise RuntimeError("Collector request refused; inspect a fresh snapshot.")
    return json.loads(p.stdout)


def snapshot():
    return invoke(["snapshot"])


def selected():
    return next(i for i in snapshot()["instances"] if i["id"] == "comet-check")


def operate(kind, request):
    args = ["operate", "comet-check", kind, "--revision", "1", "--request-id", request]
    op = invoke(args)
    deadline = time.monotonic() + 1250
    while time.monotonic() < deadline:
        current = next(o for o in snapshot()["operations"] if o["id"] == op["id"])
        if current["status"] in ("succeeded", "failed", "interrupted"):
            print(json.dumps({"operation": current}), flush=True)
            assert current["status"] == "succeeded", "terminal operation failed; no automatic retry"
            assert invoke(args) == current
            return current
        time.sleep(2)
    raise RuntimeError("Operation deadline elapsed; reconcile its fixed request ID before proceeding")


def healthy(expected=None):
    deadline = time.monotonic() + 35
    while time.monotonic() < deadline:
        i = selected()
        assert i["mode"] == "comet" and i["observed_state"] == "running" and i["health"] == "healthy"
        assert not i.get("recovery_required") and i["config_revision"] == i["applied_revision"] == 1
        assert COMET.fullmatch(i["ship"]) and (expected is None or i["ship"] == expected)
        assert i["endpoint"] == "http://127.0.0.1:18087"
        if i.get("resources", {}).get("status") == "fresh":
            break
        time.sleep(1)
    else:
        raise RuntimeError("Comet telemetry did not become fresh")
    c = http.client.HTTPConnection("127.0.0.1", 18087, timeout=5)
    try:
        c.request("GET", "/~/host")
        response = c.getresponse()
        assert response.status == 200 and response.read(128).decode().strip() == i["ship"]
    finally:
        c.close()
    return i


def main():
    hello = invoke(["hello"])
    assert hello["mode"] == "development-managed" and hello["access_role"] == "operator"
    assert "create-comet" in hello["features"]
    before = snapshot()
    existing = [i for i in before["instances"] if i["id"] == "comet-check"]
    if existing:
        assert any(o["request_id"] == "checkpoint-6r-register-comet-check" and o["actor"] == hello["actor"] for o in before["operations"])
    registered = invoke(REGISTER)
    assert registered["kind"] == "register-comet" and registered["status"] == "succeeded"
    print(json.dumps({"registration": registered}), flush=True)
    i = selected()
    if not existing:
        assert i["observed_state"] == "uninitialized" and not i.get("ship") and i["applied_revision"] == 0
    assert invoke(REGISTER) == registered
    operate("start", "checkpoint-6r-start-comet-check")
    first = healthy()
    assert invoke(REGISTER) == registered
    print(json.dumps({"first_public_identity": first["ship"], "resources": first["resources"]}), flush=True)
    code = invoke(["login-code", "comet-check", "--revision", "1", "--container-id",
                   first["resources"]["container_id"], "--reveal"])
    assert re.fullmatch(r"[a-z]{6}(?:-[a-z]{6}){3}", code["code"])
    assert code["code"] not in json.dumps(snapshot())
    code.clear()
    assert invoke(["operate", "comet-check", "pack", "--revision", "1", "--request-id",
                   "checkpoint-6r-comet-maintenance-refused"], refusal=True) == "unsupported_for_instance"
    operate("stop", "checkpoint-6r-stop-comet-check")
    operate("start", "checkpoint-6r-resume-comet-check")
    final = healthy(first["ship"])
    assert final["resources"]["container_id"] != first["resources"]["container_id"]
    assert invoke(REGISTER) == registered
    after = snapshot()
    for name in ("alpha", "panel-check", "import-check"):
        peer = next(i for i in after["instances"] if i["id"] == name)
        assert peer["observed_state"] == "running" and peer["health"] == "healthy"
    print(json.dumps({"gate_passed": True, "runner_id": RUNNER, "instance": final,
                      "code_shape_and_redaction_verified": True,
                      "maintenance_refused": True, "incidents": after["incidents"]}, indent=2))


if __name__ == "__main__":
    main()
