#!/usr/bin/env python3
"""Process-level checkpoint 4A validation using disposable simulator runners."""
import json
import pathlib
import signal
import subprocess
import tempfile
import time

ROOT = pathlib.Path(__file__).resolve().parents[1]
RUNNER = ROOT / "build/urbit-runner"
COLLECTOR = ROOT / "build/urbit-collector"


def main():
    processes = []
    with tempfile.TemporaryDirectory(prefix="omabit-4a-") as directory:
        lab = pathlib.Path(directory)

        def invoke(profile, *args, status=0, parse=True):
            result = subprocess.run([str(COLLECTOR), "--config-dir", str(lab / profile), *args], capture_output=True, text=True, timeout=10)
            assert result.returncode == status, (args, result.returncode, result.stderr)
            return json.loads(result.stdout or result.stderr) if parse else result.stdout

        def start(name):
            folder = lab / name
            folder.mkdir(mode=0o700, exist_ok=True)
            socket = folder / "rpc.sock"
            proc = subprocess.Popen([str(RUNNER), "demo", "--socket", str(socket)], stdout=subprocess.DEVNULL, stderr=subprocess.PIPE)
            processes.append(proc)
            for _ in range(100):
                if socket.exists():
                    return proc, str(socket)
                assert proc.poll() is None, proc.stderr.read().decode()
                time.sleep(0.02)
            raise AssertionError("simulator did not become ready")

        def stop(proc):
            proc.send_signal(signal.SIGINT)
            assert proc.wait(timeout=5) == 0

        try:
            a, sa = start("a")
            b, sb = start("b")
            ea = invoke("one", "enroll", "a", "--socket", sa)
            eb = invoke("one", "enroll", "b", "--socket", sb)
            invoke("two", "enroll", "a", "--socket", sa)
            assert invoke("one", "enroll", "a", "--socket", sa) == ea
            inventory = invoke("one", "inventory")
            assert inventory["complete"] and len(inventory["connections"]) == 2
            assert ea["runner_id"] != eb["runner_id"]
            print("PASS: two processes, persistent enrollment, distinct runner identities")
            changed = invoke("one", "--runner", "a", "configure", "alpha", "--revision=1", "--http-port=19080", "--ames-port=34343", "--loom=31")
            assert changed["config_revision"] == 2
            conflict = invoke("two", "--runner", "a", "configure", "alpha", "--revision=1", "--http-port=19081", "--ames-port=34343", "--loom=31", status=1)
            assert conflict["error"]["code"] == "revision_conflict"
            assert invoke("one", "--runner", "b", "snapshot")["instances"][0]["config_revision"] == 1
            print("PASS: selected-runner routing and independent collector revision conflict")
            stop(b)
            inventory = invoke("one", "inventory", status=1)
            assert [v["status"] for v in inventory["connections"]] == ["available", "unavailable"]
            assert "snapshot" not in inventory["connections"][1]
            print("PASS: unavailable runner preserves healthy inventory")
            b, _ = start("b")
            rejected = invoke("one", "--runner", "b", "operate", "alpha", "start", "--revision=1", "--request-id=must-not-run", status=1)
            assert rejected["error"]["code"] == "runner_identity_mismatch"
            invoke("one", "forget", "b", "--runner-id", eb["runner_id"])
            invoke("one", "enroll", "b", "--socket", sb)
            assert not invoke("one", "--runner", "b", "snapshot")["operations"]
            print("PASS: replacement rejected before mutation; explicit re-enrollment recovers")
            text = invoke("one", "--format=text", "inventory", parse=False)
            assert "Inventory complete: true" in text and "INSTANCE" in text
            print(text)
        finally:
            for proc in processes:
                if proc.poll() is None:
                    proc.send_signal(signal.SIGINT)
                    try:
                        proc.wait(timeout=5)
                    except subprocess.TimeoutExpired:
                        proc.kill()
                        proc.wait()
                proc.stderr.close()


if __name__ == "__main__":
    main()
