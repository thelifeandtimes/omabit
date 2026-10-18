#!/usr/bin/env python3
"""Install an owned Omabit panel snapshot, preserving existing/user-modified files."""
import argparse
import contextlib
import ctypes
import fcntl
import hashlib
import json
import os
import pathlib
import re
import shutil
import stat
import subprocess
import sys
import tempfile
import time

ROOT = pathlib.Path(__file__).resolve().parents[1]
PLUGIN = "thelifeandtimes.omabit"
MARKER = ".omabit-install.json"


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def checkpoint(name):
    """Filesystem boundary used by the isolated interruption fixtures."""


def identity(path):
    try:
        value = path.lstat()
    except FileNotFoundError:
        return None
    return [value.st_dev, value.st_ino]


def sync_directory(path):
    fd = os.open(path, os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW)
    try:
        os.fsync(fd)
    finally:
        os.close(fd)


def rename_atomic(source, destination, exchange=False):
    # Linux renameat2 keeps the installation continuously present on replacement,
    # and refuses to overwrite an unexpected object on first publication.
    libc = ctypes.CDLL(None, use_errno=True)
    rename = getattr(libc, "renameat2", None)
    if rename is None:
        raise RuntimeError("atomic snapshot publication requires Linux renameat2 support")
    rename.argtypes = [ctypes.c_int, ctypes.c_char_p, ctypes.c_int, ctypes.c_char_p, ctypes.c_uint]
    rename.restype = ctypes.c_int
    if rename(-100, os.fsencode(source), -100, os.fsencode(destination), 2 if exchange else 1):
        error = ctypes.get_errno()
        raise OSError(error, os.strerror(error), str(destination))


def snapshot(path):
    """Validate the v1 ownership marker and capture identities, modes and content.

    Include every entry, including directories and the marker itself. Never follow
    links or read special files. The retired tree is checked against this capture,
    rather than against a possibly edited ownership marker.
    """
    root = path.lstat()
    if not stat.S_ISDIR(root.st_mode):
        raise RuntimeError("refusing a non-directory installation")
    entries = {}
    for parent, directories, files in os.walk(path, followlinks=False):
        for name in sorted(directories + files):
            child = pathlib.Path(parent) / name
            info = child.lstat()
            entry = [info.st_dev, info.st_ino, info.st_mode, info.st_uid, info.st_gid, info.st_mtime_ns]
            if stat.S_ISREG(info.st_mode):
                fd = os.open(child, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK)
                with os.fdopen(fd, "rb") as stream:
                    before = os.fstat(stream.fileno())
                    def stamp(value):
                        return (value.st_dev, value.st_ino, value.st_mode, value.st_uid,
                                value.st_gid, value.st_size, value.st_mtime_ns, value.st_ctime_ns)

                    if not stat.S_ISREG(before.st_mode) or stamp(before) != stamp(info):
                        raise RuntimeError("snapshot changed during validation")
                    entry.append(hashlib.file_digest(stream, "sha256").hexdigest())
                    if stamp(os.fstat(stream.fileno())) != stamp(before) or stamp(child.lstat()) != stamp(before):
                        raise RuntimeError("snapshot changed during validation")
            elif not stat.S_ISDIR(info.st_mode):
                raise RuntimeError("snapshot contains a link or special file")
            entries[str(child.relative_to(path))] = entry
    marker = entries.get(MARKER)
    if marker is None or not stat.S_ISREG(marker[2]):
        raise RuntimeError("refusing an unmanaged installation")
    state = json.loads((path / MARKER).read_text())
    if not isinstance(state, dict):
        raise RuntimeError("unsupported snapshot marker")
    files = state.get("files")
    if state.get("version") != 1 or not isinstance(files, dict):
        raise RuntimeError("unsupported snapshot marker")
    expected_directories = set()
    for name in files:
        relative = pathlib.PurePosixPath(name)
        if relative.is_absolute() or ".." in relative.parts or str(relative) != name or name == MARKER:
            raise RuntimeError("unsafe snapshot marker path")
        expected_directories.update(str(p) for p in relative.parents if str(p) != ".")
    actual_files = {name for name, entry in entries.items() if stat.S_ISREG(entry[2])}
    actual_directories = set(entries) - actual_files
    if actual_files != set(files) | {MARKER} or actual_directories != expected_directories:
        raise RuntimeError("installed snapshot contains unexpected files or directories")
    if any(entries[name][-1] != expected for name, expected in files.items()):
        raise RuntimeError("installed snapshot has local changes; refusing replacement")
    return {"identity": [root.st_dev, root.st_ino], "mode": root.st_mode,
            "owner": [root.st_uid, root.st_gid], "entries": entries}


def sync_tree(path):
    for parent, directories, files in os.walk(path, topdown=False, followlinks=False):
        for name in files:
            fd = os.open(pathlib.Path(parent) / name, os.O_RDONLY | os.O_NOFOLLOW)
            try:
                os.fsync(fd)
            finally:
                os.close(fd)
        sync_directory(parent)


@contextlib.contextmanager
def destination_lock(destination):
    lock = destination.parent / ("." + destination.name + ".omabit-lock")
    fd = os.open(lock, os.O_CREAT | os.O_RDWR | os.O_CLOEXEC | os.O_NOFOLLOW, 0o600)
    try:
        info = os.fstat(fd)
        if not stat.S_ISREG(info.st_mode) or info.st_uid != os.getuid() or info.st_nlink != 1:
            raise RuntimeError("refusing an unowned installation lock")
        fcntl.flock(fd, fcntl.LOCK_EX)
        if identity(lock) != [info.st_dev, info.st_ino]:
            raise RuntimeError("installation lock was replaced")
        yield
    finally:
        os.close(fd)


def write_record(path, value):
    with path.open("x") as stream:
        json.dump(value, stream, indent=2)
        stream.write("\n")
        stream.flush()
        os.fsync(stream.fileno())


def publish_record(path, value):
    # Temporary files contain only journal data, never an installed generation.
    fd, temporary = tempfile.mkstemp(prefix=".record-", dir=path.parent)
    try:
        with os.fdopen(fd, "w") as stream:
            json.dump(value, stream, indent=2)
            stream.write("\n")
            stream.flush()
            os.fsync(stream.fileno())
        rename_atomic(pathlib.Path(temporary), path)
        sync_directory(path.parent)
    finally:
        if os.path.exists(temporary):
            os.unlink(temporary)


def finish_transaction(destination, journal, before_replace=None):
    if journal.is_symlink() or not journal.is_file():
        raise RuntimeError("invalid snapshot recovery journal")
    plan = json.loads(journal.read_text())
    prefix = "." + destination.name + ".omabit-recovery-"
    if (plan.get("version") != 1 or plan.get("destination") != destination.name
            or not re.fullmatch(re.escape(prefix) + r"[a-z0-9_]+", plan.get("transaction", ""))):
        raise RuntimeError("unrecognized snapshot recovery journal")
    transaction = destination.parent / plan["transaction"]
    if transaction.is_symlink() or identity(transaction) != plan["transaction_identity"]:
        raise RuntimeError("snapshot recovery directory was replaced; preserving all data")
    staging = transaction / "new"
    previous = transaction / "previous"
    old = plan["old"]
    expected_old = old["identity"] if old else None
    if identity(destination) != plan["new"]["identity"]:
        if identity(destination) != expected_old or snapshot(staging) != plan["new"]:
            raise RuntimeError("snapshot publication paths changed; preserving recovery data")
        def revalidate_old():
            if old:
                try:
                    if snapshot(destination) == old:
                        return
                except (OSError, ValueError, RuntimeError):
                    pass
                # Nothing has been published. Abandon just this intent so that
                # restoring an operator edit permits an ordinary future retry.
                # Keep all files, including the unused staged generation.
                journal.unlink()
                sync_directory(destination.parent)
                raise RuntimeError("installed snapshot changed before publication; preserving all files")

        revalidate_old()
        if before_replace is not None:
            before_replace()
        revalidate_old()
        checkpoint("before-publication")
        rename_atomic(staging, destination, exchange=old is not None)
        checkpoint("published")
    # Both rename parents must be durable before the journal is removed. On any
    # failure the persistent journal and both generations remain for a normal retry.
    sync_directory(destination.parent)
    sync_directory(transaction)
    checkpoint("publication-synced")
    changed = False
    if old:
        retired = previous if previous.exists() else staging
        try:
            changed = snapshot(retired) != old
        except (OSError, ValueError, RuntimeError):
            changed = True
        checkpoint("retired-checked")
        # Keep even an unchanged retired tree. Editors with open descriptors are
        # not fenced by flock, and can still write after this check. Never rmtree
        # a retired generation, including during recovery or a later update.
        if retired == staging:
            rename_atomic(staging, previous)
        checkpoint("retired-renamed")
        sync_directory(transaction)
        checkpoint("retired-synced")
    if not (transaction / "completed.json").exists():
        # A torn record is harmless: recovery is derived from directory identities,
        # not this informational completion note.
        write_record(transaction / "completed.json", {"retired_changes_detected": changed})
    sync_directory(transaction)
    checkpoint("completion-synced")
    journal.unlink()
    checkpoint("journal-unlinked")
    sync_directory(destination.parent)
    checkpoint("complete")
    if old:
        label = " (post-validation changes detected)" if changed else ""
        print(f"Preserved previous panel snapshot{label}: {previous}")


def install_locked(destination, replace, before_replace):
    journal = destination.parent / ("." + destination.name + ".omabit-transaction.json")
    if journal.exists() or journal.is_symlink():
        finish_transaction(destination, journal, before_replace)
        # Recovery finishes the recorded generation, which may differ from this
        # invocation's source package. Fulfil the current request under the same
        # lock, revalidating the recovered installation before another update.
        # This also covers retrying an interrupted first installation.
        replace = True
        checkpoint("recovered")
    old = None
    if destination.exists() or destination.is_symlink():
        if not replace:
            raise RuntimeError("destination exists; use --replace only for an unmodified owned snapshot")
        old = snapshot(destination)
    checkpoint("validated")
    binary = ROOT / "build/urbit-collector"
    desktop = ROOT / "build/omabit"
    runner = ROOT / "build/urbit-runner"
    if not all(path.is_file() for path in (binary, desktop, runner)):
        raise RuntimeError("build urbit-runner, urbit-collector and omabit before installing the panel")
    # Deliberately persistent, including incomplete staging after process death.
    # Only an atomic, synced journal admits publication; unexplained leftovers are
    # never adopted or recursively removed.
    transaction = pathlib.Path(tempfile.mkdtemp(prefix="." + destination.name + ".omabit-recovery-",
                                              dir=destination.parent))
    sync_directory(destination.parent)
    checkpoint("stage-created")
    staging = transaction / "new"
    shutil.copytree(ROOT / "plugins/omabit", staging, symlinks=True)
    (staging / "bin").mkdir()
    for source in (binary, desktop, runner):
        shutil.copy2(source, staging / "bin" / source.name)
        os.chmod(staging / "bin" / source.name, 0o755)
    checkpoint("stage-copied")
    if any(p.is_symlink() for p in staging.rglob("*")):
        raise RuntimeError("snapshot source contains symlinks")
    state = {"version": 1, "files": {str(p.relative_to(staging)): digest(p) for p in staging.rglob("*") if p.is_file()}}
    (staging / MARKER).write_text(json.dumps(state, indent=2) + "\n")
    new = snapshot(staging)
    sync_tree(staging)
    sync_directory(transaction)
    checkpoint("stage-synced")
    plan = {"version": 1, "destination": destination.name, "transaction": transaction.name,
            "transaction_identity": identity(transaction), "old": old, "new": new}
    write_record(transaction / "plan.json", plan)
    checkpoint("plan-written")
    sync_directory(transaction)
    os.link(transaction / "plan.json", journal, follow_symlinks=False)
    checkpoint("journal-linked")
    sync_directory(destination.parent)
    checkpoint("journal-synced")
    finish_transaction(destination, journal, before_replace)


def installation_path(destination):
    destination = destination.absolute()
    missing = []
    parent = destination.parent
    while not parent.exists():
        missing.append(parent)
        parent = parent.parent
    if parent.resolve() != parent:
        raise RuntimeError("refusing a symlinked installation path")
    for parent in reversed(missing):
        parent.mkdir(mode=0o700, exist_ok=True)
        sync_directory(parent)
        sync_directory(parent.parent)
    if destination.parent.resolve() != destination.parent or destination.is_symlink():
        raise RuntimeError("refusing a symlinked installation path")
    return destination


def install(destination, replace=False, before_replace=None):
    destination = installation_path(destination)
    with destination_lock(destination):
        install_locked(destination, replace, before_replace)
    print(f"Installed Omabit panel snapshot: {destination}")


def replace_enabled(destination):
    """Cold replacement avoids incubating new QML while old services are dying."""
    destination = installation_path(destination)
    restart_intent = destination.parent / ("." + destination.name + ".omabit-shell-restart.json")
    restart = False

    def restart_shell():
        subprocess.run(["omarchy", "restart", "shell"], check=True, timeout=30)
        restart_intent.unlink()
        sync_directory(destination.parent)

    def stop_shell():
        nonlocal restart
        # Recovery and the requested update can each activate a generation in
        # one invocation. Keep the shell stopped across both, with one durable
        # restart intent and one final restart.
        if restart:
            return
        result = subprocess.run(["omarchy", "shell", "lock", "status"],
                                capture_output=True, text=True, check=True, timeout=5)
        state = json.loads(result.stdout)
        if not isinstance(state, dict) or "locked" not in state or any(
                state.get(key) for key in ("locked", "secure", "requested", "pending", "sessionLocked")):
            raise RuntimeError("Unlock the desktop before replacing the enabled panel")
        # Match the environment used by Omarchy's own restart command.
        result = subprocess.run(["systemctl", "--user", "show-environment"],
                                capture_output=True, text=True, check=True, timeout=5)
        path = next((line.split("=", 1)[1] for line in result.stdout.splitlines()
                     if line.startswith("OMARCHY_PATH=")), os.environ.get("OMARCHY_PATH", ""))
        config = pathlib.Path(path) / "shell"
        if not path or not (config / "shell.qml").is_file():
            raise RuntimeError("Cannot identify the running Omarchy shell configuration")
        # The launch supervisor treats this clean exit as a deliberate stop.
        # Persist restart intent before dispatch: SIGKILL cannot run finally.
        publish_record(restart_intent, {"version": 1, "destination": destination.name})
        restart = True
        checkpoint("shell-stop-intent")
        subprocess.run(["quickshell", "kill", "-p", str(config), "--any-display"],
                       capture_output=True, text=True, check=True, timeout=10)
        checkpoint("shell-stopped")

    with destination_lock(destination):
        if restart_intent.exists() or restart_intent.is_symlink():
            if (restart_intent.is_symlink() or not restart_intent.is_file()
                    or json.loads(restart_intent.read_text()) != {"version": 1, "destination": destination.name}):
                raise RuntimeError("unrecognized shell restart intent; preserving it")
            restart_shell()
        try:
            # Keep the destination lock through shell restart as well as staging,
            # quiescence and activation. Another updater must not interleave them.
            install_locked(destination, replace=True, before_replace=stop_shell)
        finally:
            if restart:
                restart_shell()
        print(f"Installed Omabit panel snapshot: {destination}")


def enable_installed(timeout=10):
    """Wait for the live registry, not just the asynchronous rescan IPC reply."""
    deadline = time.monotonic() + timeout

    def command(args):
        remaining = deadline - time.monotonic()
        if remaining <= 0:
            raise RuntimeError("timed out waiting for the shell plugin registry")
        result = subprocess.run(["omarchy", *args], capture_output=True, text=True,
                                timeout=min(3, remaining))
        if result.returncode != 0:
            raise RuntimeError(result.stderr.strip() or "Omarchy shell command failed")
        return result.stdout

    try:
        command(["shell", "shell", "rescanPlugins"])
        next_scan = time.monotonic() + 1
        requested_enable = False
        while time.monotonic() < deadline:
            plugins = json.loads(command(["shell", "shell", "listPlugins"]))
            if not isinstance(plugins, list) or any(not isinstance(p, dict) for p in plugins):
                raise RuntimeError("invalid shell plugin registry response")
            entry = next((p for p in plugins if p.get("id") == PLUGIN), None)
            if entry is not None:
                if entry.get("enabled") is True:
                    return
                if not requested_enable:
                    command(["plugin", "enable", PLUGIN])
                    requested_enable = True
            elif not requested_enable and time.monotonic() >= next_scan:
                # Omarchy ignores rescan requests while another scan is active.
                # That scan may have started before the snapshot was installed.
                command(["shell", "shell", "rescanPlugins"])
                next_scan = time.monotonic() + 1
            remaining = deadline - time.monotonic()
            if remaining > 0:
                time.sleep(min(0.1, remaining))
        raise RuntimeError("timed out waiting for plugin discovery or enabled status")
    except (OSError, ValueError, subprocess.SubprocessError, RuntimeError) as error:
        raise RuntimeError(
            f"The panel is installed, but enabling was not confirmed: {error}\n"
            "Retry without reinstalling (from the checkout):\n"
            "  python3 -B scripts/install-panel.py --enable-only"
        ) from None


def main():
    parser = argparse.ArgumentParser()
    config = pathlib.Path(os.environ.get("XDG_CONFIG_HOME", pathlib.Path.home() / ".config"))
    parser.add_argument("--destination", type=pathlib.Path, default=config / "omarchy/plugins" / PLUGIN)
    parser.add_argument("--replace", action="store_true")
    parser.add_argument("--enable", action="store_true")
    parser.add_argument("--enable-only", action="store_true", help="enable an existing installation without copying files")
    args = parser.parse_args()
    if args.enable_only and (args.enable or args.replace):
        parser.error("--enable-only cannot be combined with --enable or --replace")
    if (args.enable or args.enable_only) and args.destination.absolute() != config / "omarchy/plugins" / PLUGIN:
        raise RuntimeError("--enable requires the normal user plugin directory")
    if args.enable_only:
        manifest = args.destination / "manifest.json"
        if not manifest.is_file() or json.loads(manifest.read_text()).get("id") != PLUGIN:
            raise RuntimeError("Omabit is not installed at the requested destination")
    elif args.replace and args.enable:
        replace_enabled(args.destination)
    else:
        install(args.destination, args.replace)
    if args.enable or args.enable_only:
        enable_installed()
        print("Enabled Omabit. Click the ~+ bar widget to open it.")


if __name__ == "__main__":
    try:
        main()
    except (OSError, ValueError, subprocess.SubprocessError, RuntimeError) as error:
        print(f"Omabit panel installation: {error}", file=sys.stderr)
        sys.exit(1)
