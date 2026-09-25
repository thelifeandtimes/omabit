#!/usr/bin/env python3
"""Exercise the desktop publisher CLI only on the acknowledged development fake."""
import argparse
import json
import pathlib
import subprocess
import tempfile

ROOT = pathlib.Path(__file__).resolve().parents[1]


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--acknowledge-development-fake", action="store_true", required=True)
    parser.add_argument("--port", type=int, default=18085)
    args = parser.parse_args()
    assert 1024 <= args.port <= 65535
    origin = f"http://127.0.0.1:{args.port}"
    with tempfile.TemporaryDirectory(prefix="omabit-theme-") as temporary:
        directory = pathlib.Path(temporary)
        config, state = str(directory / "collector"), str(directory / "omabit")

        def command(binary, *args, stdin=None):
            prefix = [str(ROOT / "build" / binary), "--config-dir", config]
            if binary == "omabit":
                prefix += ["--state-dir", state]
            result = subprocess.run(prefix + list(args), input=stdin, capture_output=True, text=True, timeout=60)
            assert result.returncode == 0, f"{binary} failed: {result.stderr}"
            assert "cookie_value" not in result.stdout and "lidlut-tabwed" not in result.stdout
            return json.loads(result.stdout)

        entry = command("urbit-collector", "ship-enroll", "fixture", "--url", origin, "--ship", "~zod", "--credentials", "file")
        selection = ["fixture", "--expected-ship-id", entry["id"]]
        command("urbit-collector", "ship-login", *selection, "--code-stdin", stdin="lidlut-tabwed-pillex-ridrup")
        try:
            command("omabit", "theme-auto", *selection, "--enabled=true")
            assert command("omabit", "theme-sync")["complete"]
            first = command("omabit", "theme-status")["ships"][entry["id"]]
            assert first["last_published"] and not first["pending"]
            assert command("omabit", "theme-sync")["complete"]
            second = command("omabit", "theme-status")["ships"][entry["id"]]
            assert second["last_published"] == first["last_published"]
            assert not command("omabit", "theme-publish", *selection)["pending"]
            assert not command("omabit", "theme-auto", *selection, "--enabled=false")["automatic"]
            print("PASS: real CLI enable/sync, unchanged read-only check, explicit resend, pause, and credential cleanup")
        finally:
            command("urbit-collector", "ship-logout", *selection)
            command("urbit-collector", "ship-forget", *selection, "--url", origin)


if __name__ == "__main__":
    main()
