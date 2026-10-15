"""Fetch verified release binaries for a git-installed Omarchy plugin.

Only immutable, explicitly pinned release assets are downloaded. Runtime state is
outside the git checkout; all three executables are installed together so the
Omabit setup helper can find its sibling runner. This never runs host setup.
"""
import fcntl
import hashlib
import http.client
import json
import os
from pathlib import Path
import platform
import re
import stat
import sys
import tempfile
import time
import urllib.parse
import urllib.request

COMMANDS = ("omabit", "urbit-collector", "urbit-runner")
RELEASES = "https://github.com/thelifeandtimes/omabit/releases/download/"
MAX_BINARY_BYTES = 64 << 20


class BootstrapError(Exception):
    pass


def validate_manifest(manifest):
    if (not isinstance(manifest, dict)
            or not re.fullmatch(r"[0-9]+\.[0-9]+\.[0-9]+(?:-rc\.[0-9]+)?", manifest.get("version", ""))
            or not re.fullmatch(r"[0-9a-f]{40}", manifest.get("source_commit", ""))
            or manifest.get("platform") != "linux/amd64"
            or not isinstance(manifest.get("binaries"), dict)
            or set(manifest["binaries"]) != set(COMMANDS)):
        raise BootstrapError("Invalid pinned binary manifest")
    for command in COMMANDS:
        entry = manifest["binaries"][command]
        if (not isinstance(entry, dict) or entry.get("name") != command + "-linux-amd64"
                or not re.fullmatch(r"[0-9a-f]{64}", entry.get("sha256", ""))
                or type(entry.get("bytes")) is not int
                or not 0 < entry["bytes"] <= MAX_BINARY_BYTES):
            raise BootstrapError("Invalid pinned binary: " + command)


def digest(path):
    with path.open("rb") as stream:
        return hashlib.file_digest(stream, "sha256").hexdigest()


def private_directory(path):
    if not path.is_absolute() or ".." in path.parts:
        raise BootstrapError("Binary cache must have a clean absolute path")
    for parent in (path, *path.parents):
        try:
            info = parent.lstat()
        except FileNotFoundError:
            continue
        if not stat.S_ISDIR(info.st_mode):
            raise BootstrapError("Binary cache ancestors must be directories without symlinks")
    path.mkdir(mode=0o700, parents=True, exist_ok=True)
    info = path.lstat()
    if info.st_uid != os.geteuid() or stat.S_IMODE(info.st_mode) != 0o700:
        raise BootstrapError("Binary cache must be owned by this user with mode 0700")


class HTTPSRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        if urllib.parse.urlsplit(newurl).scheme != "https":
            raise BootstrapError("Refusing a non-HTTPS release redirect")
        return super().redirect_request(req, fp, code, msg, headers, newurl)


def download(url, target, entry):
    opener = urllib.request.build_opener(HTTPSRedirect())
    request = urllib.request.Request(url, headers={"User-Agent": "Omabit-native-plugin", "Accept-Encoding": "identity"})
    with opener.open(request, timeout=30) as response, target.open("xb") as output:
        if response.status != 200:
            raise BootstrapError("Release asset did not return HTTP 200")
        if response.headers.get("Content-Encoding", "identity") != "identity":
            raise BootstrapError("Unexpected release asset encoding")
        length = response.headers.get("Content-Length")
        if length is not None and int(length) != entry["bytes"]:
            raise BootstrapError("Release asset size differs from its pin")
        remaining = entry["bytes"]
        hashed = hashlib.sha256()
        deadline = time.monotonic() + 90
        while remaining:
            if time.monotonic() > deadline:
                raise BootstrapError("Release download deadline exceeded")
            chunk = response.read(min(1 << 20, remaining))
            if not chunk:
                raise BootstrapError("Incomplete release asset")
            output.write(chunk)
            hashed.update(chunk)
            remaining -= len(chunk)
        if response.read(1) or hashed.hexdigest() != entry["sha256"]:
            raise BootstrapError("Release asset does not match its SHA-256 pin")
        output.flush()
        os.fchmod(output.fileno(), 0o755)
        os.fsync(output.fileno())


def verify_cache(directory, manifest):
    info = directory.lstat()
    if not stat.S_ISDIR(info.st_mode) or info.st_uid != os.geteuid() or stat.S_IMODE(info.st_mode) != 0o700:
        raise BootstrapError("Unsafe binary cache directory")
    if {p.name for p in directory.iterdir()} != set(COMMANDS):
        raise BootstrapError("Incomplete or modified binary cache")
    for command, entry in manifest["binaries"].items():
        path = directory / command
        info = path.lstat()
        if (not stat.S_ISREG(info.st_mode) or info.st_uid != os.geteuid() or info.st_nlink != 1
                or stat.S_IMODE(info.st_mode) != 0o755 or info.st_size != entry["bytes"]
                or digest(path) != entry["sha256"]):
            raise BootstrapError("Cached binary was modified: " + command)


def ensure_binaries(manifest, cache_home=None):
    validate_manifest(manifest)
    if sys.platform != "linux" or platform.machine() not in ("x86_64", "amd64"):
        raise BootstrapError("This release requires Linux x86-64")
    if cache_home is None:
        cache_home = Path(os.environ.get("XDG_CACHE_HOME") or Path.home() / ".cache")
    root = Path(cache_home) / "omabit" / "native-binaries"
    private_directory(root)
    key = hashlib.sha256(json.dumps(manifest, sort_keys=True, separators=(",", ":")).encode()).hexdigest()
    destination = root / key
    # All startup commands share this lock. Closing it before exec also ensures a
    # long-lived inventory watch cannot block other commands or plugin updates.
    fd = os.open(root / "download.lock", os.O_RDWR | os.O_CREAT | os.O_NOFOLLOW | os.O_CLOEXEC, 0o600)
    with os.fdopen(fd, "r+b") as lock:
        info = os.fstat(lock.fileno())
        if not stat.S_ISREG(info.st_mode) or info.st_uid != os.geteuid() or info.st_nlink != 1 or stat.S_IMODE(info.st_mode) != 0o600:
            raise BootstrapError("Unsafe binary download lock")
        deadline = time.monotonic() + 300
        while True:
            try:
                fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
                break
            except BlockingIOError:
                if time.monotonic() > deadline:
                    raise BootstrapError("Timed out waiting for the binary download")
                time.sleep(0.1)
        if os.path.lexists(destination):
            verify_cache(destination, manifest)
            return destination
        with tempfile.TemporaryDirectory(prefix=".download-", dir=root) as temporary:
            staging = Path(temporary)
            for command in COMMANDS:
                entry = manifest["binaries"][command]
                download(RELEASES + "v" + manifest["version"] + "/" + entry["name"], staging / command, entry)
            verify_cache(staging, manifest)
            staging.rename(destination)
        directory_fd = os.open(root, os.O_RDONLY | os.O_DIRECTORY)
        try:
            os.fsync(directory_fd)
        finally:
            os.close(directory_fd)
    return destination


def main(command, arguments):
    try:
        if command not in COMMANDS:
            raise BootstrapError("Unknown bundled command")
        with Path(__file__).with_name("release.json").open("rb") as stream:
            data = stream.read(16385)
        if len(data) > 16384:
            raise BootstrapError("Binary manifest exceeds its size bound")
        manifest = json.loads(data)
        binary = ensure_binaries(manifest) / command
        os.execv(binary, [str(binary), *arguments])
    except (BootstrapError, OSError, ValueError, TypeError, KeyError, http.client.HTTPException) as error:
        print("Omabit release tools unavailable: " + str(error), file=sys.stderr)
        return 1
