"""Shared isolation and compositor evidence for opt-in mapped qualification."""
import contextlib
import datetime
import json
import os
from pathlib import Path
import signal
import subprocess
import time

HARNESS_ROOT = Path(__file__).resolve().parents[1]
ROOT = Path(os.environ.get("OMABIT_QUALIFY_ROOT", HARNESS_ROOT)).resolve()
NATIVE = Path(os.environ.get("OMABIT_QUALIFY_NATIVE", ROOT / "native")).resolve()


def hypr(command):
    return json.loads(subprocess.check_output(["hyprctl", "-j", command], text=True, timeout=5))


def focus_window(address):
    # The current Lua provider takes an expression, not legacy dispatch argv.
    assert hypr("status")["configProvider"] == "lua", "this fixture requires the current Lua Hyprland contract"
    expression = "hl.dsp.focus({ window = " + json.dumps("address:" + address) + " })"
    subprocess.run(["hyprctl", "dispatch", expression], check=True, capture_output=True, timeout=5)


@contextlib.contextmanager
def preserve_desktop():
    lock = json.loads(subprocess.check_output(["omarchy", "shell", "lock", "status"], text=True, timeout=5))
    assert not any(lock.get(k) for k in ("locked", "secure", "requested", "pending", "sessionLocked")), "unlock before mapped tests"
    assert hypr("status")["configProvider"] == "lua"
    original = hypr("activewindow")
    workspace = hypr("activeworkspace")
    clients = {w["address"]: (w["pid"], w["workspace"]) for w in hypr("clients")}
    try:
        yield
    finally:
        current = {w["address"]: (w["pid"], w["workspace"]) for w in hypr("clients")}
        if original.get("address") in current:
            focus_window(original["address"])
            deadline = time.monotonic() + 3
            while hypr("activewindow").get("address") != original["address"] and time.monotonic() < deadline:
                time.sleep(.1)
            assert hypr("activewindow").get("address") == original["address"], "focus was not restored"
        else:
            expression = "hl.dsp.focus({ workspace = " + json.dumps("name:" + workspace["name"]) + " })"
            subprocess.run(["hyprctl", "dispatch", expression], check=True, capture_output=True, timeout=5)
        assert all(current.get(address) == value for address, value in clients.items()), "an existing window disappeared or changed workspace"
        print("PASS: existing window PIDs/workspaces preserved and original focus restored")


def environment(directory):
    """Private HOME and every writable XDG root; only the display/Hypr IPC is shared."""
    desktop_runtime = Path(os.environ["XDG_RUNTIME_DIR"])
    display = os.environ["WAYLAND_DISPLAY"]
    if not display.startswith("/"):
        display = str(desktop_runtime / display)
    env = dict(os.environ)
    for key, leaf in (("HOME", "home"), ("XDG_CONFIG_HOME", "config"), ("XDG_DATA_HOME", "data"),
                      ("XDG_CACHE_HOME", "cache"), ("XDG_STATE_HOME", "state"), ("XDG_RUNTIME_DIR", "runtime")):
        target = directory / leaf
        target.mkdir(mode=0o700, exist_ok=True)
        env[key] = str(target)
    runtime = directory / "runtime"
    (runtime / "hypr").mkdir(exist_ok=True)
    (runtime / "hypr/f").symlink_to(desktop_runtime / "hypr" / os.environ["HYPRLAND_INSTANCE_SIGNATURE"])
    env.update(QT_QPA_PLATFORM="wayland", WAYLAND_DISPLAY=display, HYPRLAND_INSTANCE_SIGNATURE="f",
               OMABIT_FIXTURE_HYPRLAND_INSTANCE=os.environ["HYPRLAND_INSTANCE_SIGNATURE"],
               OMABIT_FIXTURE_DESKTOP_RUNTIME=str(desktop_runtime),
               OMABIT_FIXTURE_TIME=datetime.datetime.now(datetime.timezone.utc).isoformat(),
               PYTHONDONTWRITEBYTECODE="1", NO_COLOR="1")
    # Do not allow Qt to reuse a caller's offscreen platform/plugin override.
    env.pop("QT_PLUGIN_PATH", None)
    return env


def stop_process(process):
    """Reap only this test's new process group, including fixture descendants."""
    try:
        os.killpg(process.pid, signal.SIGTERM)
    except ProcessLookupError:
        pass
    try:
        process.wait(timeout=5)
    except subprocess.TimeoutExpired:
        os.killpg(process.pid, signal.SIGKILL)
        process.wait(timeout=5)
    try:
        os.killpg(process.pid, signal.SIGKILL)
    except ProcessLookupError:
        pass


def save_evidence(directory, text, windows):
    output = os.environ.get("OMABIT_QUALIFY_EVIDENCE")
    if not output:
        return
    output = Path(output)
    output.mkdir(mode=0o700, parents=True, exist_ok=True)
    (output / "quickshell.log").write_text(text)
    (output / "mapped-windows.json").write_text(json.dumps(windows, indent=2) + "\n")
    for name in ("calls.jsonl", "events.jsonl"):
        if (directory / name).exists():
            (output / name).write_bytes((directory / name).read_bytes())


def run_qs(directory, *, timeout, mapped=True):
    windows = {}
    with preserve_desktop():
        with (directory / "quickshell.log").open("w+") as log:
            process = subprocess.Popen(["qs", "--no-color", "-p", str(directory)], env=environment(directory),
                                       stdout=log, stderr=subprocess.STDOUT, start_new_session=True)
            try:
                deadline = time.monotonic() + timeout
                while process.poll() is None and time.monotonic() < deadline:
                    for window in hypr("clients"):
                        if window["pid"] == process.pid and window["mapped"] and not window["hidden"]:
                            windows[window["address"]] = window
                    time.sleep(.1)
                timed_out = process.poll() is None
            finally:
                stop_process(process)
                log.seek(0)
                text = log.read()
                save_evidence(directory, text, list(windows.values()))
            assert not timed_out, "Quickshell fixture timed out:\n" + text
            assert not mapped or windows, "No actual fixture toplevel mapped:\n" + text
            if mapped:
                print("PASS: compositor observed fixture-owned mapped toplevel(s)")
            return subprocess.CompletedProcess(process.args, process.returncode, text, "")
