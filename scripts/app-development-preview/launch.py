#!/usr/bin/env python3
"""Launch only this standalone, networkless policy preview."""
import argparse
import fcntl
import json
import os
import signal
import stat
import subprocess
import sys
import time
from pathlib import Path

ROOT = Path(__file__).resolve().parent
DATA = ROOT / "data"
PID = DATA / "state/process.json"
LOG = DATA / "state/window.log"


def start_time(pid):
    try:
        return Path(f"/proc/{pid}/stat").read_text().rsplit(")", 1)[1].split()[19]
    except (FileNotFoundError, ProcessLookupError):
        return None


def running():
    try:
        record = json.loads(PID.read_text())
        return record if start_time(record["pid"]) == record["start_time"] else None
    except FileNotFoundError:
        return None


def command(smoke=False):
    # Code is read-only. The only writable host mount is this preview's data.
    # /run, /home, host IPC, D-Bus, SSH, Docker and runner sockets are absent.
    result = ["/usr/bin/bwrap", "--unshare-all", "--die-with-parent", "--clearenv",
              "--ro-bind", "/usr", "/usr", "--symlink", "usr/bin", "/bin",
              "--symlink", "usr/lib", "/lib", "--symlink", "usr/lib", "/lib64",
              "--ro-bind", "/etc/fonts", "/etc/fonts", "--proc", "/proc",
              "--dev", "/dev", "--tmpfs", "/tmp", "--perms", "0700", "--dir", "/runtime",
              "--ro-bind", str(ROOT), "/preview", "--bind", str(DATA), "/data",
              "--chdir", "/preview"]
    env = {"PATH": "/usr/bin:/bin", "LANG": "C.UTF-8", "HOME": "/data/home",
           "XDG_CONFIG_HOME": "/data/config", "XDG_STATE_HOME": "/data/state",
           "XDG_DATA_HOME": "/data/share", "XDG_CACHE_HOME": "/data/cache",
           "XDG_RUNTIME_DIR": "/runtime", "QT_QUICK_BACKEND": "software",
           "QSG_RHI_BACKEND": "software", "QT_QUICK_CONTROLS_STYLE": "Basic",
           "QML_DISABLE_DISK_CACHE": "1", "NO_COLOR": "1"}
    if smoke:
        env.update(QT_QPA_PLATFORM="offscreen", OMABIT_PREVIEW_SMOKE="1")
    else:
        display = os.environ.get("WAYLAND_DISPLAY", "")
        runtime = os.environ.get("XDG_RUNTIME_DIR", "")
        if not display or not runtime:
            raise RuntimeError("Launch from your Wayland desktop session.")
        socket = Path(display) if display.startswith("/") else Path(runtime) / display
        if not stat.S_ISSOCK(socket.stat().st_mode):
            raise RuntimeError("The Wayland display is not a socket.")
        # One rendering socket only, not the host runtime directory.
        result += ["--ro-bind", str(socket), "/runtime/wayland-preview"]
        env.update(QT_QPA_PLATFORM="wayland", WAYLAND_DISPLAY="wayland-preview")
    for key, value in env.items():
        result += ["--setenv", key, value]
    return result + ["--", "/usr/bin/quickshell", "--no-color", "-p", "/preview/shell.qml"]


def run(smoke=False):
    with (DATA / "state/window.lock").open("a") as lock:
        try:
            fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
        except BlockingIOError:
            print("The preview is already open.")
            return 0
        invocation = command(smoke)
        if LOG.exists():
            LOG.replace(LOG.with_suffix(".previous.log"))
        child = None
        def stop(*_):
            if child is not None:
                child.terminate()
        signal.signal(signal.SIGTERM, stop)
        signal.signal(signal.SIGINT, stop)
        try:
            temporary = PID.with_suffix(".tmp")
            temporary.write_text(json.dumps({"pid": os.getpid(), "start_time": start_time(os.getpid())}))
            temporary.replace(PID)
            with LOG.open("w") as log:
                child = subprocess.Popen(invocation, stdin=subprocess.DEVNULL, stdout=log, stderr=subprocess.STDOUT)
                return child.wait(timeout=25 if smoke else None)
        finally:
            if child is not None and child.poll() is None:
                child.terminate()
                child.wait(timeout=5)
            PID.unlink(missing_ok=True)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("action", choices=("start", "stop", "status", "_run", "smoke"), nargs="?", default="start")
    args = parser.parse_args()
    os.umask(0o077)
    if args.action in ("_run", "smoke"):
        return run(args.action == "smoke")
    record = running()
    if args.action == "status":
        print("Preview is open." if record else "Preview is closed.")
        print(f"Saved fixture policies: {DATA / 'state/policy.json'}")
    elif args.action == "stop":
        if record:
            # Pin the process before checking its identity, avoiding PID reuse.
            fd = os.pidfd_open(record["pid"])
            try:
                if start_time(record["pid"]) == record["start_time"]:
                    signal.pidfd_send_signal(fd, signal.SIGTERM)
            finally:
                os.close(fd)
            print("Closed this preview; saved policies retained.")
        else:
            print("The preview is already closed.")
    elif record:
        print("The preview is already open. Switch to ‘Omabit 0.4.0 · Setup Preview’. ")
    else:
        # Detach only this supervisor; its sandbox is tied to its lifetime.
        env = {key: os.environ[key] for key in ("PATH", "WAYLAND_DISPLAY", "XDG_RUNTIME_DIR") if key in os.environ}
        with (DATA / "state/launcher.log").open("w") as log:
            child = subprocess.Popen([sys.executable, "-B", str(ROOT / "launch.py"), "_run"],
                                     stdin=subprocess.DEVNULL, stdout=log, stderr=subprocess.STDOUT,
                                     start_new_session=True, close_fds=True, env=env)
        for _ in range(80):
            time.sleep(.1)
            if child.poll() is not None:
                raise RuntimeError(f"Preview could not start. See {LOG} and {DATA / 'state/launcher.log'}")
            if running() and LOG.exists() and "READY: isolated policy preview" in LOG.read_text():
                print("Opened Omabit 0.4.0 · Setup Preview.")
                return 0
        print(f"Preview is starting. Log: {LOG}")
    return 0


if __name__ == "__main__":
    try:
        sys.exit(main())
    except (OSError, RuntimeError, subprocess.TimeoutExpired) as error:
        print(str(error), file=sys.stderr)
        sys.exit(1)
