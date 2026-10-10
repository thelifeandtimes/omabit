#!/usr/bin/env python3
"""Validate the enrolled development fake's code using an ephemeral HTTP session.

Secrets stay in memory: no credential adapter, clipboard, environment or files.
Only public selection metadata, audit records and assertion outcomes are printed.
"""
import datetime
import http.client
import http.cookies
import json
import pathlib
import re
import subprocess
import urllib.parse

ROOT = pathlib.Path(__file__).resolve().parents[1]
RUNNER = "runner-f496feed1f7b08f31a5d82b94ac2f5fd"
BASE = [str(ROOT / "build/urbit-collector"), "--runner", "workstation", "--expected-runner-id", RUNNER]


def request(*args):
    result = subprocess.run(BASE + list(args), capture_output=True, text=True, timeout=15)
    if result.returncode:
        raise RuntimeError("collector request failed (response withheld)")
    return json.loads(result.stdout)


def main():
    hello, before = request("hello"), request("snapshot")
    assert hello["mode"] in ("development-managed-fake", "development-managed") and hello["access_role"] == "operator"
    assert "login-code" in hello["features"]
    ship = next(i for i in before["instances"] if i["id"] == "alpha")
    assert ship["mode"] == "fake" and ship["ship"] == "~zod" and ship["health"] == "healthy" and ship["observed_state"] == "running"
    assert not ship.get("active_operation_id") and not ship.get("recovery_required")
    # This is an explicitly acknowledged local development fixture, not arbitrary
    # runner-supplied routing or an external/production application enrollment.
    assert ship["endpoint"] == "http://127.0.0.1:18081"
    code = request("login-code", "alpha", "--revision", str(ship["config_revision"]),
                   "--container-id", ship["resources"]["container_id"], "--reveal")
    assert code["runner_id"] == RUNNER and code["instance_id"] == "alpha"
    assert code["container_id"] == ship["resources"]["container_id"]
    assert re.fullmatch(r"[a-z]{6}(?:-[a-z]{6}){3}", code["code"])

    cookie = ""
    checked = logged_out = False
    connection = http.client.HTTPConnection("127.0.0.1", 18081, timeout=5)

    def exchange(method, path, body=None, authenticated=False):
        headers = {"Content-Type": "application/x-www-form-urlencoded"} if body is not None else {}
        if authenticated:
            headers["Cookie"] = cookie
        connection.request(method, path, body=body, headers=headers)
        response = connection.getresponse()
        payload = response.read(65537)
        assert len(payload) <= 65536
        return response.status, response.getheaders(), payload

    try:
        status, _, body = exchange("GET", "/~/host")
        assert status == 200 and body.strip() == b"~zod"
        status, headers, _ = exchange("POST", "/~/login", urllib.parse.urlencode({"password": code["code"]}))
        # Capture a returned session before further assertions so finally can
        # revoke it even if an unexpected login status is received.
        for key, value in headers:
            if key.lower() == "set-cookie":
                parsed = http.cookies.SimpleCookie(value)
                if "urbauth-~zod" in parsed:
                    cookie = "urbauth-~zod=" + parsed["urbauth-~zod"].value
        assert status in (200, 204) and cookie
        status, _, body = exchange("GET", "/~/name", authenticated=True)
        checked = status == 200 and body.strip() == b"~zod"
        assert checked
    finally:
        if cookie:
            status, _, _ = exchange("POST", "/~/logout", authenticated=True)
            logged_out = status in (200, 204, 303)
            status, _, body = exchange("GET", "/~/name", authenticated=True)
            logged_out = logged_out and not (status == 200 and body.strip() == b"~zod")
        connection.close()
    assert logged_out
    after = request("snapshot")
    assert code["code"] not in json.dumps(after) and cookie not in json.dumps(after)
    audits = [o for o in after["operations"] if o["kind"] == "login-code"]
    assert audits and audits[0]["status"] == "succeeded"
    assert audits[0]["actor"] == hello["actor"]
    print(json.dumps({"checked_at": datetime.datetime.now(datetime.timezone.utc).isoformat(),
                      "runner_id": RUNNER, "instance": "alpha", "revision": ship["config_revision"],
                      "identity_bound_retrieval": True, "authenticated_name_verified": checked,
                      "temporary_session_revoked": logged_out, "snapshot_secret_free": True,
                      "latest_audit": audits[0]}, indent=2))


if __name__ == "__main__":
    try:
        main()
    except Exception:
        raise SystemExit("Code-access validation failed; secret-bearing details withheld.") from None
