#!/usr/bin/env python3
"""Retained bootstrap observer correctness and repeatable process/CPU comparison."""
import importlib.util
import io
import json
import os
from pathlib import Path
import resource
import subprocess
import sys
import tempfile
import threading
import time
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / "native/bootstrap"))
import control
spec = importlib.util.spec_from_file_location("bootstrap_fixture", ROOT / "scripts/test-desktop-bootstrap.py")
fixture = importlib.util.module_from_spec(spec)
spec.loader.exec_module(fixture)


def cpu(kind):
    r = resource.getrusage(kind)
    return r.ru_utime + r.ru_stime


def main():
    with tempfile.TemporaryDirectory(prefix="omabit-bootstrap-observation-") as tmp:
        root = Path(tmp)
        env = {"PATH": "/usr/bin:/bin", "LANG": "C.UTF-8", "HOME": str(root / "home")}
        for key, leaf in (("XDG_CONFIG_HOME", "config"), ("XDG_STATE_HOME", "state"),
                          ("XDG_DATA_HOME", "data"), ("XDG_CACHE_HOME", "cache"), ("XDG_RUNTIME_DIR", "runtime")):
            (root / leaf).mkdir(mode=0o700)
            env[key] = str(root / leaf)
        (root / "home").mkdir(mode=0o700)
        _, _, pins = fixture.bundle(root / "bundle")
        pins_path = root / "release.json"
        pins_path.write_text(json.dumps(pins))
        with patch.dict(os.environ, env, clear=True):
            desktop = control.Desktop(pins_path)
            first = desktop.resolve()
            state_path = desktop.state / "state.json"
            initial_stat = state_path.stat()
            count = 40
            start = cpu(resource.RUSAGE_CHILDREN)
            for _ in range(count):
                result = subprocess.check_output([sys.executable, "-B", str(ROOT / "native/bootstrap/control.py"),
                                                  "--pins", str(pins_path), "resolve"], env=env)
                assert json.loads(result) == first
            repeated_cpu = cpu(resource.RUSAGE_CHILDREN) - start
            read_fd, write_fd = os.pipe()
            reader = os.fdopen(read_fd, "rb", buffering=0)
            writer = os.fdopen(write_fd, "wb", buffering=0)
            lock = threading.Condition()
            class Output(io.StringIO):
                def __init__(self):
                    super().__init__()
                    self.lines = []
                def write(self, text):
                    with lock:
                        super().write(text)
                        if text == "\n":
                            self.lines = self.getvalue().splitlines()
                            lock.notify_all()
                    return len(text)
            output = Output()
            worker = threading.Thread(target=control.watch_selection, args=(desktop, "fixture-session", reader, output, .01))
            def wait_rows(number):
                with lock:
                    assert lock.wait_for(lambda: len(output.lines) >= number, timeout=3), "observer stopped emitting"
                    return [json.loads(line) for line in output.lines]
            start = cpu(resource.RUSAGE_SELF)
            worker.start()
            try:
                rows = wait_rows(count)
                retained_cpu = cpu(resource.RUSAGE_SELF) - start
                assert all(row == {"session": "fixture-session", "plan": first} for row in rows)
                after_stat = state_path.stat()
                assert (initial_stat.st_ino, initial_stat.st_mtime_ns) == (after_stat.st_ino, after_stat.st_mtime_ns), "read-only observation rewrote state"
                # An invalid descriptor must become an error, not a stale success;
                # correcting it must recover without another process or lease.
                pins_path.write_text('{"version":"invalid"}')
                at = len(rows)
                rows = wait_rows(at + 3)
                assert "error" in rows[-1] and "plan" not in rows[-1]
                pins_path.write_text(json.dumps(pins))
                rows = wait_rows(len(rows) + 3)
                assert rows[-1]["plan"] == first
                desktop.create_profile("battery-lab", False)
                desktop.select("battery-lab")
                rows = wait_rows(len(rows) + 3)
                assert rows[-1]["plan"]["profile"] == "battery-lab"
            finally:
                writer.close()
                worker.join(timeout=1)
                reader.close()
            assert not worker.is_alive(), "EOF did not release observer"
            print(json.dumps({"observations": count, "old_additional_processes": count,
                              "new_additional_processes": 0, "old_cpu_seconds": repeated_cpu,
                              "retained_cpu_seconds": retained_cpu,
                              "scope": "isolated descriptor/state resolution; excludes Qt and SSH CPU"}, sort_keys=True))
            print("PASS: retained observer validates each read, preserves no-op state, recovers malformed pins, follows profile selection, and exits on EOF")

if __name__ == "__main__":
    main()
