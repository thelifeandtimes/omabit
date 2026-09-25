#!/usr/bin/env python3
"""Explicit live validation against Omabit's known localhost development fake."""
import argparse
import json
import pathlib
import subprocess
import tempfile

ROOT = pathlib.Path(__file__).resolve().parents[1]
COLLECTOR = ROOT / "build/urbit-collector"
# This is the public fake ~zod tutorial code, never a real ship credential.
FAKE_CODE = "lidlut-tabwed-pillex-ridrup"


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--acknowledge-development-fake", action="store_true", required=True)
    parser.add_argument("--port", type=int, default=18081)
    args = parser.parse_args()
    assert 1024 <= args.port <= 65535
    origin = f"http://127.0.0.1:{args.port}"
    for adapter in ("file", "secret-service"):
        with tempfile.TemporaryDirectory(prefix="omabit-eyre-") as directory:
            profile = str(pathlib.Path(directory) / "collector")

            def invoke(*args, code=0, stdin=None):
                result = subprocess.run([str(COLLECTOR), "--config-dir", profile, *args], input=stdin, capture_output=True, text=True, timeout=25)
                assert FAKE_CODE not in result.stdout + result.stderr, "login code leaked"
                assert "cookie_value" not in result.stdout + result.stderr, "session credential leaked"
                assert result.returncode == code, f"{args[0]} failed (exit {result.returncode}); diagnostic: {result.stderr}"
                return result.stdout

            logged_in = False
            try:
                invoke("ship-enroll", "test", "--url", origin, "--ship", "~zod", "--credentials", adapter)
                invoke("ship-login", "test", "--code-stdin", stdin=FAKE_CODE)
                logged_in = True
                assert json.loads(invoke("ship-status", "test"))["status"] == "authenticated"
                # Session replacement must not let cleanup of the old session
                # log out the newly stored one.
                invoke("ship-login", "test", "--code-stdin", stdin=FAKE_CODE)
                assert json.loads(invoke("ship-status", "test"))["status"] == "authenticated"
                data = json.loads(invoke("ship-scry", "test", "--app", "hood", "--path", "/kiln/pikes"))["data"]
                assert "base" in data and "webterm" in data and "groups" in data
                events = [json.loads(line) for line in invoke("ship-watch", "test", "--app", "hood", "--path", "/kiln/updates", "--events", "1", "--duration", "10s").splitlines()]
                assert any(event.get("response") == "diff" for event in events)
                print(f"PASS [{adapter}]: persistent login, replacement, status, SSE fact/ack/delete; desks: {', '.join(sorted(data))}")
                readable = invoke("--format", "text", "ship-status", "test")
                print(readable, end="")
                result = json.loads(invoke("ship-logout", "test"))
                assert result["status"] == "logged_out" and not result.get("message"), "remote logout was not confirmed"
                logged_in = False
                assert json.loads(invoke("ship-status", "test", code=1))["status"] == "login_required"
                invoke("ship-forget", "test", "--url", origin)
                print(f"PASS [{adapter}]: server logout and local credential/enrollment cleanup")
            finally:
                if logged_in:
                    invoke("ship-logout", "test")


if __name__ == "__main__":
    main()
