#!/usr/bin/env python3
"""Unprivileged host controller for disposable, KVM-only filesystem crash tests.

All downloads, binaries, initramfs files, disks and raw evidence stay under build/.
No host mounts, package installation, services, Docker or live piers are used.
"""

import argparse
import datetime
import hashlib
import json
import os
from pathlib import Path
import re
import selectors
import signal
import stat
import subprocess
import sys
import time
import urllib.request

REPO = Path(__file__).resolve().parents[1]
PREFIX = b"POWERCUT "
CASES = {
    "storage": [
        "copy-file-before-fsync", "copy-file-after-fsync",
        "destination-leaf-before-fsync", "destination-leaf-after-fsync",
        "destination-mount-before-fsync", "destination-mount-after-fsync",
        "destination-group-before-fsync", "destination-group-after-fsync",
        "destination-root-before-fsync", "destination-root-after-fsync",
        "destination-before-fsync", "destination-after-fsync",
        "source-before-unlink", "source-after-unlink",
        "source-parent-before-fsync", "source-parent-after-fsync",
        "source-root-after-unlink", "source-removed",
    ],
    "update": [
        "journal-before-file-fsync", "journal-after-file-fsync",
        "journal-before-rename", "journal-after-rename",
        "journal-before-dir-fsync", "journal-after-dir-fsync",
        "executable-before-file-fsync", "executable-after-file-fsync",
        "executable-before-rename", "executable-after-rename",
        "executable-before-dir-fsync", "executable-after-dir-fsync",
        "starting-after-dir-fsync",
    ],
}


def command(args, **kwargs):
    return subprocess.check_output([str(a) for a in args], cwd=REPO, **kwargs).decode().strip()


def sha(path):
    h = hashlib.sha256()
    with open(path, "rb") as f:
        for data in iter(lambda: f.read(1024 * 1024), b""):
            h.update(data)
    return h.hexdigest()


def write_json(path, value):
    path.write_text(json.dumps(value, indent=2, sort_keys=True) + "\n")


def sources():
    names = subprocess.check_output(
        ["git", "ls-files", "-z", "--cached", "--others", "--exclude-standard"], cwd=REPO
    ).decode().split("\0")
    files = {n: sha(REPO / n) for n in sorted(set(names)) if n and (REPO / n).is_file()}
    digest = hashlib.sha256(json.dumps(files, sort_keys=True).encode()).hexdigest()
    return {"sha256": digest, "files": files, "commit": command(["git", "rev-parse", "HEAD"]),
            "status": command(["git", "status", "--porcelain=v1"])}


def tool_env(work):
    return {**os.environ, "LD_LIBRARY_PATH": str(work / "tools/usr/lib"),
            "QEMU_MODULE_DIR": str(work / "tools/usr/lib/qemu")}


def tool_evidence(work):
    manifest = json.loads((work / "packages.json").read_text())
    for package in manifest:
        archive = work / "packages" / package["url"].rsplit("/", 1)[1]
        if sha(archive) != package["sha256"]:
            raise RuntimeError(f"package differs from verified evidence: {archive}")
    binary = work / "tools/usr/bin/qemu-system-x86_64"
    dependencies = re.sub(r" \(0x[0-9a-f]+\)", "", command(["ldd", binary], env=tool_env(work)))
    paths = {str(binary), str(work / "tools/usr/bin/qemu-img")}
    paths.update(re.findall(r"(?:=>\s+)?(/[^\s()]+)", dependencies))
    for directory in ("usr/share/qemu", "usr/share/seabios", "usr/lib/qemu"):
        paths.update(str(p) for p in (work / "tools" / directory).rglob("*") if p.is_file())
    return {"version": command([binary, "--version"], env=tool_env(work)),
            "executable": str(binary),
            "environment": {k: tool_env(work)[k] for k in ("LD_LIBRARY_PATH", "QEMU_MODULE_DIR")},
            "dependencies": dependencies, "files": {p: sha(p) for p in sorted(paths)}}


def require_frozen(snapshot, commit):
    if commit and (snapshot["commit"] != commit or snapshot["status"]):
        raise RuntimeError("frozen qualification requires a clean checkout of the exact --expect-commit")


def tools(work):
    packages, root, home = work / "packages", work / "tools", work / "gnupg"
    for directory in (packages, root, home):
        directory.mkdir(exist_ok=True, mode=0o700)
    keyrings = []
    revoked = set()
    for name in ("archlinux", "omarchy"):
        source = Path(f"/usr/share/pacman/keyrings/{name}.gpg")
        keyring = home / f"{name}.gpg"
        data = source.read_bytes()
        if data.startswith(b"-----BEGIN"):
            data = subprocess.check_output(["gpg", "--homedir", str(home), "--dearmor"], input=data)
        keyring.write_bytes(data)
        keyrings += ["--keyring", str(keyring)]
        revoked.update(Path(f"/usr/share/pacman/keyrings/{name}-revoked").read_text().split())
    urls = command(["pacman", "-Sp", "qemu-system-x86", "qemu-img"]).splitlines()
    evidence = []
    for url in urls:
        if not url.startswith("https://stable-mirror.omarchy.org/"):
            raise RuntimeError(f"unexpected package origin: {url}")
        archive = packages / url.rsplit("/", 1)[1]
        for suffix in ("", ".sig"):
            dest = Path(str(archive) + suffix)
            if not dest.exists():
                print(f"Downloading {dest.name}", flush=True)
                with urllib.request.urlopen(url + suffix, timeout=120) as response:
                    with dest.open("xb") as out:
                        while data := response.read(1024 * 1024):
                            out.write(data)
        verify = subprocess.run(
            ["gpgv", "--homedir", str(home), "--status-fd=1", *keyrings, str(archive) + ".sig", str(archive)],
            capture_output=True, text=True, check=True,
        )
        valid = [line.split()[2:] for line in verify.stdout.splitlines() if line.startswith("[GNUPG:] VALIDSIG ")]
        if not valid or any(fingerprint in revoked for fields in valid for fingerprint in fields):
            raise RuntimeError(f"missing valid or revoked package signature: {archive}")
        subprocess.run(["bsdtar", "-xf", str(archive), "-C", str(root)], check=True)
        evidence.append({"url": url, "sha256": sha(archive), "signature": verify.stderr, "signature_status": verify.stdout})
    write_json(work / "packages.json", evidence)
    print(command([root / "usr/bin/qemu-system-x86_64", "--version"], env=tool_env(work)))


def cpio_file(out, name, data, mode, inode):
    name = name.encode() + b"\0"
    fields = (inode, mode, 0, 0, 1, 0, len(data), 0, 0, 0, 0, len(name), 0)
    header = b"070701" + b"".join(f"{n:08x}".encode() for n in fields)
    out.write(header + name)
    out.write(b"\0" * (-(len(header) + len(name)) % 4))
    out.write(data)
    out.write(b"\0" * (-len(data) % 4))


def prepare(work, kernel, expect_commit):
    if not (work / "tools/usr/bin/qemu-system-x86_64").exists():
        tools(work)
    before = sources()
    require_frozen(before, expect_commit)
    qemu = tool_evidence(work)
    write_json(work / "source.json", before)
    for directory in (work / "go-cache", work / "go-tmp"):
        directory.mkdir(exist_ok=True)
    env = {**os.environ, "CGO_ENABLED": "0", "GOCACHE": str(work / "go-cache"),
           "GOTMPDIR": str(work / "go-tmp"), "GOMAXPROCS": "2"}
    subprocess.run(["go", "build", "-p=2", "-trimpath", "-o", str(work / "init"),
                    "./scripts/powercut-init"], cwd=REPO, env=env, check=True)
    subprocess.run(["go", "test", "-p=2", "-c", "-tags=powercut", "-trimpath", "-o",
                    str(work / "fixture.test"), "./internal/hostinstall"], cwd=REPO, env=env, check=True)
    if sources() != before:
        raise RuntimeError("source changed during build; rerun prepare on a stable snapshot")
    entries = {"init": (work / "init", 0o100755), "fixture.test": (work / "fixture.test", 0o100755)}
    for binary in ("/usr/bin/mkfs.ext4", "/usr/bin/mkfs.btrfs"):
        entries[binary.lstrip("/")] = (Path(binary), 0o100755)
        output = command(["ldd", binary])
        for name in re.findall(r"(?:=>\s+)?(/[^\s()]+)", output):
            entries[name.lstrip("/")] = (Path(name), 0o100755)
    # Include filesystem-tool configuration, without any host identity/config files.
    entries["etc/mke2fs.conf"] = (Path("/etc/mke2fs.conf"), 0o100644)
    modules = []
    release = kernel.parent.name
    for module in ("virtio_pci", "virtio_blk", "ext4", "btrfs"):
        for line in command(["modprobe", "--set-version", release, "--show-depends", module]).splitlines():
            if line.startswith("insmod "):
                path = Path(line.split()[1])
                if path in modules:
                    continue
                modules.append(path)
                data = subprocess.check_output(["zstd", "-dc", str(path)]) if path.suffix == ".zst" else path.read_bytes()
                entries[f"modules/{len(modules):03d}.ko"] = (data, 0o100644)
    entries["powercut-modules"] = (("\n".join(f"/modules/{n:03d}.ko" for n in range(1, len(modules)+1)) + "\n").encode(), 0o100644)
    entries["powercut-guest"] = (b"isolated-rc12-filesystem-guest-v1\n", 0o100444)
    directories = {"proc", "sys", "dev", "tmp", "source", "destination"}
    for name in entries:
        directories.update(str(p) for p in Path(name).parents if str(p) != ".")
    with (work / "initramfs.cpio").open("wb") as out:
        inode = 1
        for name in sorted(directories, key=lambda x: (x.count("/"), x)):
            cpio_file(out, name, b"", 0o40755, inode)
            inode += 1
        for name, (data, mode) in sorted(entries.items()):
            cpio_file(out, name, data if isinstance(data, bytes) else data.read_bytes(), mode, inode)
            inode += 1
        cpio_file(out, "TRAILER!!!", b"", 0, inode)
    (work / "vmlinuz").write_bytes(kernel.read_bytes())
    write_json(work / "build.json", {
        "source_sha256": before["sha256"], "commit": before["commit"], "dirty": bool(before["status"]),
        "go": command(["go", "version"]), "host_kernel": os.uname().release,
        "kernel_source": str(kernel), "kernel_release": release,
        "artifacts": {n: sha(work / n) for n in ("init", "fixture.test", "initramfs.cpio", "vmlinuz")},
        "packages_sha256": sha(work / "packages.json"),
        "qemu": qemu,
        "guest_inputs": {name: hashlib.sha256(data if isinstance(data, bytes) else data.read_bytes()).hexdigest()
                         for name, (data, _) in sorted(entries.items())},
        "host_disk_filesystem": command(["stat", "-f", "-c", "%T", work]),
    })


def boot(work, run, phase, suite, case, fs, timeout):
    bios = work / "tools/usr/share/qemu/bios-256k.bin"
    if not bios.exists():
        bios = work / "tools/usr/share/seabios/bios.bin"
    args = [str(work / "tools/usr/bin/qemu-system-x86_64"), "-enable-kvm", "-cpu", "host",
            "-machine", "q35", "-m", "768M", "-smp", "1", "-nodefaults", "-no-user-config",
            "-display", "none", "-monitor", "none", "-serial", "stdio", "-nic", "none",
            "-no-reboot", "-L", str(work / "tools/usr/share/qemu"), "-bios", str(bios),
            "-kernel", str(work / "vmlinuz"), "-initrd", str(work / "initramfs.cpio"),
            "-append", f"console=ttyS0 panic=-1 quiet rdinit=/init powercut.phase={phase} powercut.suite={suite} powercut.case={case} powercut.fs={fs}"]
    for index, name in enumerate(("source", "destination")):
        disk = run / f"{name}.raw"
        if not stat.S_ISREG(disk.lstat().st_mode) or disk.is_symlink():
            raise RuntimeError("only generated regular disk files are permitted")
        args += ["-drive", f"file={disk},format=raw,if=none,id=disk{index},cache=none,aio=threads",
                 "-device", f"virtio-blk-pci,drive=disk{index}"]
    record = {"phase": phase, "command": args, "events": [], "started_unix_ns": time.time_ns()}
    proc = subprocess.Popen(args, stdin=subprocess.DEVNULL, stdout=subprocess.PIPE,
                            stderr=subprocess.STDOUT, env=tool_env(work))
    record["pid"] = proc.pid
    selector = selectors.DefaultSelector()
    selector.register(proc.stdout, selectors.EVENT_READ)
    buffer = b""
    deadline = time.monotonic() + timeout
    cut = False
    try:
        with (run / f"{phase}.serial.log").open("xb") as log:
            while time.monotonic() < deadline:
                ready = selector.select(0.2)
                if not ready:
                    if proc.poll() is not None:
                        break
                    continue
                data = os.read(proc.stdout.fileno(), 65536)
                if not data:
                    break
                log.write(data)
                log.flush()
                buffer += data
                while b"\n" in buffer:
                    line, buffer = buffer.split(b"\n", 1)
                    if not line.startswith(PREFIX):
                        continue
                    event = json.loads(line[len(PREFIX):])
                    record["events"].append(event)
                    if event.get("event") == "boundary":
                        if phase != "crash" or event.get("case") != case:
                            raise RuntimeError("unexpected kill boundary")
                        record["boundary_received_unix_ns"] = time.time_ns()
                        # No QMP quit, guest shutdown, sync, unmount or process cleanup.
                        proc.send_signal(signal.SIGKILL)
                        record["sigkill_sent_unix_ns"] = time.time_ns()
                        cut = True
                if cut:
                    break
            if proc.poll() is None and not cut and time.monotonic() >= deadline:
                raise TimeoutError(f"{phase} timed out: {run}")
            record["returncode"] = proc.wait(timeout=20)
        events = record["events"]
        if phase == "crash":
            if not cut or record["returncode"] != -signal.SIGKILL:
                raise RuntimeError(f"VM did not reach requested boundary: {run}")
        elif record["returncode"] != 0 or not any(e.get("event") == "success" for e in events):
            raise RuntimeError(f"guest {phase} failed: {run}")
        record["passed"] = True
        return record
    except BaseException as error:
        record["error"] = f"{type(error).__name__}: {error}"
        raise
    finally:
        if proc.poll() is None:
            proc.kill()
            proc.wait()
        selector.close()
        proc.stdout.close()
        record["finished_unix_ns"] = time.time_ns()
        write_json(run / f"{phase}.json", record)


def run_cases(work, args):
    build = json.loads((work / "build.json").read_text())
    snapshot = json.loads((work / "source.json").read_text())
    if snapshot["sha256"] != build["source_sha256"]:
        raise RuntimeError("source manifest differs from build evidence")
    if tool_evidence(work) != build["qemu"] or sha(work / "packages.json") != build["packages_sha256"]:
        raise RuntimeError("QEMU tools differ from build evidence")
    for name, digest in build["artifacts"].items():
        if sha(work / name) != digest:
            raise RuntimeError(f"artifact changed: {name}")
    if args.expect_commit and (build["commit"] != args.expect_commit or build["dirty"]):
        raise RuntimeError("frozen qualification requires a clean build of --expect-commit")
    if args.expect_commit:
        require_frozen(sources(), args.expect_commit)
        if sources() != snapshot:
            raise RuntimeError("frozen source differs from the built source manifest")
    stamp = datetime.datetime.now(datetime.timezone.utc).strftime("%Y%m%dT%H%M%S.%fZ")
    output = work / f"run-{stamp}"
    output.mkdir()
    report = {"version": 1, "scope": "production piercopy/pierstore import primitives and installer publication with synthetic payloads; no live Vere, management service or SQLite lifecycle",
              "model": "KVM guest RAM loss by host SIGKILL; raw virtio disks cache=none; completed host writes survive",
              "build": build, "cases": [], "passed": False,
              "started_utc": stamp, "frozen_commit": args.expect_commit,
              "matrix": CASES, "selection": {"suite": args.suite, "filesystem": args.filesystem, "case": args.case}}
    write_json(output / "source.json", json.loads((work / "source.json").read_text()))
    write_json(output / "packages.json", json.loads((work / "packages.json").read_text()))
    suites = list(CASES) if args.suite == "all" else [args.suite]
    filesystems = ["ext4", "btrfs"] if args.filesystem == "all" else [args.filesystem]
    try:
        for fs in filesystems:
            for suite in suites:
                cases = CASES[suite] if not args.case else [c for c in CASES[suite] if c in args.case]
                for case in cases:
                    item = {"filesystem": fs, "suite": suite, "case": case, "phases": []}
                    report["cases"].append(item)
                    directory = output / f"{fs}-{case}"
                    directory.mkdir()
                    for name in ("source", "destination"):
                        with (directory / f"{name}.raw").open("xb") as disk:
                            disk.truncate(512 * 1024 * 1024)
                    print(f"{fs} {case}", flush=True)
                    for phase in ("seed", "crash", "verify"):
                        item["phases"].append(boot(work, directory, phase, suite, case, fs, args.timeout))
                    item["passed"] = True
                    # Evidence remains; generated data is optional and never tracked.
                    if not args.keep_disks:
                        for name in ("source", "destination"):
                            (directory / f"{name}.raw").unlink()
                    write_json(output / "report.json", report)
        if not report["cases"]:
            raise RuntimeError("no cases selected")
        if args.expect_commit and sources() != snapshot:
            raise RuntimeError("frozen source changed during qualification")
        report["passed"] = True
    except BaseException as error:
        report["error"] = f"{type(error).__name__}: {error}"
        raise
    finally:
        report["finished_utc"] = datetime.datetime.now(datetime.timezone.utc).isoformat()
        report["passed_cases"] = sum(bool(item.get("passed")) for item in report["cases"])
        write_json(output / "report.json", report)
        print(f"Evidence: {output / 'report.json'}", flush=True)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("action", choices=["tools", "prepare", "run"])
    parser.add_argument("--work", type=Path, default=REPO / "build/rc12-powercut")
    parser.add_argument("--kernel", type=Path, default=Path("/usr/lib/modules") / os.uname().release / "vmlinuz")
    parser.add_argument("--suite", choices=["all", *CASES], default="all")
    parser.add_argument("--filesystem", choices=["all", "ext4", "btrfs"], default="all")
    parser.add_argument("--case", action="append", choices=sum(CASES.values(), []))
    parser.add_argument("--timeout", type=int, default=120)
    parser.add_argument("--keep-disks", action="store_true")
    parser.add_argument("--expect-commit", help="require a clean build of this exact frozen commit")
    args = parser.parse_args()
    work = args.work.resolve()
    if os.geteuid() == 0 or work == REPO / "build" or not work.is_relative_to(REPO / "build") or "," in str(work):
        parser.error("run unprivileged with a work directory strictly beneath this repository's build/")
    if not (REPO / "build").is_dir():
        parser.error("build/ must already exist")
    work.mkdir(exist_ok=True)
    if args.action == "tools":
        tools(work)
    elif args.action == "prepare":
        prepare(work, args.kernel.resolve(), args.expect_commit)
    else:
        if not os.access("/dev/kvm", os.R_OK | os.W_OK):
            parser.error("read/write access to /dev/kvm is required")
        selected = sum((CASES[s] for s in CASES if args.suite in ("all", s)), [])
        if args.case and any(case not in selected for case in args.case):
            parser.error("every --case must belong to the selected suite")
        run_cases(work, args)


if __name__ == "__main__":
    main()
