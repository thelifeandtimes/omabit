#!/usr/bin/env python3
"""Bounded, passive SSH suggestions. Never invokes SSH, Match exec or a host."""
import argparse
import glob
import json
import os
from pathlib import Path
import re
import shlex
import stat

HOST = re.compile(r"[A-Za-z0-9_][A-Za-z0-9._-]{0,252}\Z")
DIRECTIVE = re.compile(r"^\s*([A-Za-z][A-Za-z0-9]*)\s*(?:=\s*|\s+)(.*)$")
MAX_FILES, MAX_BYTES, MAX_FILE, MAX_HOSTS = 128, 1 << 20, 256 << 10, 256


def discover(home=None, config=None, system_config=Path("/etc/ssh/ssh_config"), system_known=None):
    home = Path.home() if home is None else Path(home)
    hosts, seen_files, known = {}, set(), []
    remaining = MAX_BYTES

    def add(host, port, source):
        if not HOST.fullmatch(host) or not 0 <= port <= 65535 or len(hosts) >= MAX_HOSTS:
            return
        key = (host.lower(), port)
        if key not in hosts:
            hosts[key] = {"host": host, "port": port, "source": source,
                          "label": host + (":" + str(port) if port else "")}

    def read(path):
        nonlocal remaining
        if len(seen_files) >= MAX_FILES or remaining <= 0:
            return ""
        try:
            fd = os.open(path, os.O_RDONLY | os.O_NONBLOCK | os.O_CLOEXEC)
            with os.fdopen(fd, "rb") as stream:
                info = os.fstat(stream.fileno())
                identity = (info.st_dev, info.st_ino)
                if not stat.S_ISREG(info.st_mode) or identity in seen_files:
                    return ""
                seen_files.add(identity)
                bound = min(MAX_FILE, remaining)
                data = stream.read(bound)
                remaining -= len(data)
                if info.st_size > len(data):
                    data = data.rsplit(b"\n", 1)[0] if b"\n" in data else b""
                return data.decode("utf-8", "replace")
        except OSError:
            return ""

    def path_for(value, base):
        value = value.replace("%d", str(home))
        if value.startswith("~/"):
            value = str(home / value[2:])
        if any(c in value for c in ("%", "$", "\x00", "\r", "\n")):
            return None
        path = Path(value)
        return path if path.is_absolute() else base / path

    def configuration(path, base, depth=0):
        if depth > 8:
            return
        for line in read(path).splitlines():
            if len(line) > 4096:
                continue
            match = DIRECTIVE.match(line)
            if not match:
                continue
            try:
                values = shlex.split(match[2], comments=True)
            except ValueError:
                continue
            directive = match[1].lower()
            if directive == "host":
                for value in values:
                    add(value, 0, "SSH config")  # Wildcards/negations fail HOST.
            elif directive == "include":
                for value in values:
                    pattern = path_for(value, base)
                    if pattern is None:
                        continue
                    for index, child in enumerate(glob.iglob(str(pattern))):
                        if index >= MAX_FILES or len(seen_files) >= MAX_FILES:
                            break
                        configuration(Path(child), base, depth + 1)
            elif directive in ("userknownhostsfile", "globalknownhostsfile"):
                for value in values:
                    path = path_for(value, base)
                    if path is not None and value != "none" and len(known) < MAX_FILES:
                        known.append(path)
            # Other directives, including Match exec and KnownHostsCommand, are
            # deliberately not evaluated. These are suggestions, not resolution.

    configuration(Path(config) if config else home / ".ssh/config", home / ".ssh")
    if config is None and system_config is not None:
        configuration(Path(system_config), Path("/etc/ssh"))
    known += [home / ".ssh/known_hosts", home / ".ssh/known_hosts2"]
    known += [Path(p) for p in (system_known if system_known is not None else ("/etc/ssh/ssh_known_hosts", "/etc/ssh/ssh_known_hosts2"))]
    for path in known:
        for line in read(path).splitlines():
            if len(line) > 8192:
                continue
            fields = line.split()
            if fields and fields[0] == "@cert-authority":
                fields = fields[1:]
            if len(fields) < 3 or not fields[1].startswith(("ssh-", "ecdsa-", "sk-")):
                continue
            for value in fields[0].split(","):
                port = 0
                match = re.fullmatch(r"\[([^\]]+)\]:(\d{1,5})", value)
                if match:
                    value, port = match[1], int(match[2])
                    if port == 0:
                        continue
                add(value, port, "Known host")  # Hashed names cannot be recovered.
    return {"hosts": list(hosts.values())}


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--config", type=Path)
    args = parser.parse_args()
    print(json.dumps(discover(config=args.config)))
