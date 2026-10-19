#!/usr/bin/env python3
"""Stock Omarchy fresh-install/upgrade scenarios in an explicitly provisioned VM.

No arguments prints a plan. No reset/delete procedure is provided. Each run starts
from a clean disposable guest; use hypervisor snapshots for repeated baselines.
"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import pwd
import re
import socket
import stat
import subprocess
import time

PLUGIN = "thelifeandtimes.omabit"
MARKER = Path("/etc/omabit-desktop-qualification.json")


def checked(*args, cwd=None, timeout=30):
    return subprocess.check_output(args, cwd=cwd, text=True, timeout=timeout).strip()


def admit(args):
    if not args.acknowledge_disposable_guest:
        raise ValueError("run requires --acknowledge-disposable-guest")
    if os.geteuid() == 0:
        raise ValueError("run as the guest desktop owner, not root")
    if not checked("systemd-detect-virt", "--vm"):
        raise ValueError("a full-system disposable VM is required")
    info = MARKER.lstat()
    if not stat.S_ISREG(info.st_mode) or info.st_uid != 0 or info.st_nlink != 1 or info.st_mode & 0o022:
        raise ValueError("the VM provisioner must supply a root-owned marker")
    marker = json.loads(MARKER.read_text())
    if (marker.get("format") != 1 or marker.get("hostname") != socket.gethostname()
            or marker.get("owner_uid") != os.getuid()
            or marker.get("vm_uuid", "").lower() != Path("/sys/class/dmi/id/product_uuid").read_text().strip().lower()):
        raise ValueError("VM identity differs from the provisioned marker")
    workspace = Path(marker["workspace"])
    if not workspace.is_absolute() or ".." in workspace.parts:
        raise ValueError("workspace must be a canonical absolute path")
    for path in (workspace, *workspace.parents):
        if path.is_symlink() or not path.is_dir():
            raise ValueError("workspace ancestors must be existing directories")
    if workspace.stat().st_uid != os.getuid() or workspace.stat().st_mode & 0o077:
        raise ValueError("workspace must be private to the test owner")
    output = args.output.absolute()
    if output.parent != workspace or output.exists():
        raise ValueError("choose a new direct child of the marked workspace")
    if args.with_host and marker.get("allow_host_setup") is not True:
        raise ValueError("the marker does not permit host setup")
    # A blank profile alone is not a fresh OS installation. Refuse any retained
    # installation rather than deleting it or guessing where its piers moved.
    for path in (Path.home() / ".config/omarchy/plugins" / PLUGIN, Path.home() / ".config/urbit-collector",
                 Path.home() / ".config/omabit/setup", Path.home() / ".local/share/urbit-runner",
                 Path("/etc/urbit-runner"), Path("/srv/urbit/piers")):
        if os.path.lexists(path):
            raise ValueError("guest is not a fresh baseline: " + str(path))
    processes = checked("ps", "-eo", "comm=").splitlines()
    if any(re.match(r"^(urbit|vere)(-|$)", name.strip()) or name.strip() in {"dockerd", "rootlesskit"} for name in processes):
        raise ValueError("an existing runtime/host process is present")
    lock = json.loads(checked("omarchy", "shell", "lock", "status"))
    if any(lock.get(key) for key in ("locked", "secure", "requested", "pending", "sessionLocked")):
        raise ValueError("unlock the disposable desktop before qualification")
    return output, marker


def projection(snapshot):
    return {"runner_id": snapshot["runner_id"], "mode": snapshot["mode"], "instances": [
        {key: row.get(key) for key in ("id", "ship", "mode", "label", "config", "config_revision", "applied_revision",
                                      "desired_state", "pier_path", "bootstrap")}
        for row in snapshot["instances"]], "operations": snapshot["operations"]}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("action", choices=("plan", "run"), nargs="?", default="plan")
    parser.add_argument("--scenario", choices=("fresh", "upgrade"), default="upgrade")
    parser.add_argument("--repository", type=Path, help="existing source Git repository inside the guest")
    parser.add_argument("--baseline", help="full existing baseline distribution commit")
    parser.add_argument("--candidate", help="full existing candidate distribution commit")
    parser.add_argument("--output", type=Path)
    parser.add_argument("--with-host", action="store_true", help="also authorize host setup, two fake ships and an explicit runner upgrade")
    parser.add_argument("--acknowledge-disposable-guest", action="store_true")
    args = parser.parse_args()
    if args.action == "plan":
        print(json.dumps({"scenario": args.scenario, "with_host": args.with_host,
                          "marker": str(MARKER), "steps": ["verify disposable VM and blank installation",
                          "serve existing pinned commits from a private local Git remote",
                          "run stock omarchy plugin install --enable --yes",
                          "optionally authorize setup and create running/stopped fake ships",
                          "advance only the private remote ref and run stock omarchy plugin update --yes",
                          "verify live bootstrap version/digest, preserved fleet and unchanged shell process",
                          "optionally perform journaled runner upgrade and verify runtime preservation",
                          "retain evidence and guest state; no automatic reset/delete"]}, indent=2))
        return
    if not args.repository or not args.output or not re.fullmatch(r"[0-9a-f]{40}", args.candidate or ""):
        parser.error("run requires --repository, --output and a full --candidate commit")
    baseline = args.baseline if args.scenario == "upgrade" else args.candidate
    if not re.fullmatch(r"[0-9a-f]{40}", baseline or ""):
        parser.error("upgrade requires a full --baseline commit")
    output, marker = admit(args)
    # Resolve and inspect immutable commits before the first mutation.
    for commit in (baseline, args.candidate):
        if checked("git", "rev-parse", commit + "^{commit}", cwd=args.repository) != commit:
            raise ValueError("commit mismatch")
    subprocess.run(["git", "merge-base", "--is-ancestor", baseline, args.candidate], cwd=args.repository, check=True)
    manifest = json.loads(checked("git", "show", args.candidate + ":manifest.json", cwd=args.repository))
    pins = json.loads(checked("git", "show", args.candidate + ":native/bootstrap/release.json", cwd=args.repository))
    if manifest.get("keepLoaded") is not True or manifest["entryPoints"]["service"] != "native/v1/Service.qml" or "bundle" not in pins:
        raise ValueError("candidate must contain the retained bootstrap and paired bundle pins")
    output.mkdir(mode=0o700)
    report = {"baseline": baseline, "candidate": args.candidate, "scenario": args.scenario,
              "with_host": args.with_host, "vm": marker, "omarchy": checked("omarchy", "version"), "passed": False}

    def run(*command, timeout=60, cwd=None):
        result = subprocess.run([str(part) for part in command], cwd=cwd, capture_output=True, text=True, timeout=timeout)
        with (output / "commands.jsonl").open("a") as stream:
            stream.write(json.dumps({"argv": [str(part) for part in command], "returncode": result.returncode,
                                     "stdout": result.stdout, "stderr": result.stderr}) + "\n")
        if result.returncode:
            raise RuntimeError("command failed: " + str(command) + "\n" + result.stderr)
        return result.stdout.strip()

    plugin = Path.home() / ".config/omarchy/plugins" / PLUGIN
    tools = plugin / "native/bin"

    def collector(*command):
        return json.loads(run(tools / "urbit-collector", "--runner", "local", *command, timeout=120))

    def wait_for(predicate, timeout=180):
        deadline = time.monotonic() + timeout
        while time.monotonic() < deadline:
            try:
                value = predicate()
                if value:
                    return value
            except (RuntimeError, ValueError, KeyError):
                pass
            time.sleep(1)
        raise RuntimeError("qualification readiness deadline exceeded")

    def instance_ready(name, state):
        rows = collector("snapshot")["instances"]
        row = next(row for row in rows if row["id"] == name)
        return row if row["observed_state"] == state and not row.get("active_operation_id") else None

    def runtime_identity():
        snapshot = collector("snapshot")
        running = [row["runtime_status"]["container_id"] for row in snapshot["instances"] if row["observed_state"] == "running"]
        engine = run("systemctl", "--user", "show", "urbit-docker.service", "--property=MainPID,InvocationID")
        containers = run("docker", "--host", "unix:///run/user/" + str(os.getuid()) + "/urbit-docker.sock", "inspect",
                         "--format", "{{.Id}} {{.State.Pid}} {{.State.StartedAt}}", *running) if running else ""
        return {"engine": engine, "containers": containers}

    try:
        remote = output / "remote.git"
        run("git", "clone", "--bare", "--no-hardlinks", args.repository.absolute(), remote)
        run("git", "--git-dir", remote, "update-ref", "refs/heads/qualification", baseline)
        run("git", "--git-dir", remote, "symbolic-ref", "HEAD", "refs/heads/qualification")
        run("omarchy", "plugin", "install", remote.as_uri(), "--enable", "--yes", timeout=180)
        wait_for(lambda: json.loads(run("omarchy", "shell", PLUGIN + ".monitor", "status")))
        if args.with_host:
            run(tools / "omabit", "--format", "json", "setup", timeout=900)
            for name, ship in (("running-test", "~zod"), ("stopped-test", "~nec")):
                port = collector("suggest-port")["http_port"]
                collector("create-fake", name, "--ship", ship, "--http-port", str(port), "--request-id", name + "-create", "--bootstrap", "bare")
                collector("operate", name, "start", "--revision", "1", "--request-id", name + "-start")
                wait_for(lambda: instance_ready(name, "running"), timeout=600)
                if name == "stopped-test":
                    collector("operate", name, "stop", "--revision", "1", "--request-id", name + "-stop")
                    wait_for(lambda: instance_ready(name, "stopped"), timeout=180)
            report["before"] = projection(collector("snapshot"))
            report["runtime_before"] = runtime_identity()
            registry = Path.home() / ".config/urbit-collector/runners.json"
            report["registry_before"] = hashlib.sha256(registry.read_bytes()).hexdigest()
        shell_before = run("pgrep", "-f", "^quickshell -n -p .*/shell$")
        if args.scenario == "upgrade":
            run("git", "--git-dir", remote, "update-ref", "refs/heads/qualification", args.candidate)
            run("omarchy", "plugin", "update", PLUGIN, "--yes", timeout=180)
        desired = hashlib.sha256(json.dumps({key: pins[key] for key in ("version", "source_commit", "platform", "binaries", "bundle")}, sort_keys=True, separators=(",", ":")).encode()).hexdigest()

        def ready():
            value = json.loads(run("omarchy", "shell", PLUGIN + ".bootstrap", "status"))
            return value if value["phase"] == "ready" and value["loaded_release"] == desired and value["loaded_version"] == pins["version"] and (not args.with_host or value["ships"] == 2) else None

        report["desktop"] = wait_for(ready, timeout=300)
        if run("pgrep", "-f", "^quickshell -n -p .*/shell$") != shell_before:
            raise RuntimeError("desktop update restarted the stock shell")
        if run("git", "rev-parse", "HEAD", cwd=plugin) != args.candidate:
            raise RuntimeError("stock update did not activate the expected commit")
        if args.with_host:
            bundle = output / "rootless-target"
            run(tools / "urbit-runner", "prepare-rootless", "--output", bundle, timeout=180)
            owner = pwd.getpwuid(os.getuid()).pw_name
            run("pkexec", tools / "urbit-runner", "install-host", "--owner", owner, "--bundle", bundle,
                "--public-key", Path.home() / ".config/omabit/setup/id_ed25519.pub", "--acknowledge-host-changes", timeout=900)
            report["after"] = projection(collector("snapshot"))
            report["runtime_after"] = runtime_identity()
            if report["before"] != report["after"] or report["runtime_before"] != report["runtime_after"]:
                raise RuntimeError("host identity/configuration/history/runtime changed across upgrade")
            if hashlib.sha256(registry.read_bytes()).hexdigest() != report["registry_before"]:
                raise RuntimeError("collector enrollment changed")
            if report["desktop"]["ships"] != 2:
                raise RuntimeError("updated live UI did not recover the populated fleet")
        report["passed"] = True
    finally:
        (output / "report.json").write_text(json.dumps(report, indent=2) + "\n")
        print("Evidence:", output / "report.json")


if __name__ == "__main__":
    main()
