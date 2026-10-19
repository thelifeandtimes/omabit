"""Desktop release/profile protocol v1. No runner installation or database opens.

Published and explicit local candidates use the same immutable bundle loader.
Only a live bootstrap lease may acknowledge activation; selection is not success.
"""
import contextlib
import copy
import fcntl
import hashlib
import json
import os
from pathlib import Path, PurePosixPath
import re
import shutil
import stat
import tarfile
import tempfile
import time
import uuid

import runtime

FORMAT = 1
MAX_ARCHIVE = 256 << 20
MAX_EXPANDED = 512 << 20
MAX_FILES = 4096
NAME = re.compile(r"[a-z][a-z0-9-]{0,47}\Z")
HASH = re.compile(r"[0-9a-f]{64}\Z")


def canonical(value):
    return json.dumps(value, sort_keys=True, separators=(",", ":")).encode()


def identity(value):
    return hashlib.sha256(canonical(value)).hexdigest()


def read_json(path, maximum=1 << 20):
    fd = os.open(path, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK)
    with os.fdopen(fd, "rb") as stream:
        info = os.fstat(stream.fileno())
        if not stat.S_ISREG(info.st_mode) or info.st_size > maximum:
            raise ValueError("JSON must be a bounded regular file")
        data = stream.read(maximum + 1)
    if len(data) > maximum:
        raise ValueError("JSON exceeds its size limit")
    def unique(pairs):
        result = {}
        for key, value in pairs:
            if key in result:
                raise ValueError("duplicate JSON key")
            result[key] = value
        return result
    return json.loads(data, object_pairs_hook=unique)


def atomic_json(path, value):
    data = canonical(value) + b"\n"
    if len(data) > 1 << 20:
        raise ValueError("desktop state exceeds its size limit")
    fd, temporary = tempfile.mkstemp(prefix=".pending-", dir=path.parent)
    try:
        with os.fdopen(fd, "wb") as stream:
            stream.write(data)
            stream.flush()
            os.fsync(stream.fileno())
        os.replace(temporary, path)
        sync_dir(path.parent)
    finally:
        if os.path.exists(temporary):
            os.unlink(temporary)


def sync_dir(path):
    fd = os.open(path, os.O_RDONLY | os.O_DIRECTORY)
    try:
        os.fsync(fd)
    finally:
        os.close(fd)


@contextlib.contextmanager
def lock(path, blocking=True):
    fd = os.open(path, os.O_RDWR | os.O_CREAT | os.O_NOFOLLOW | os.O_CLOEXEC, 0o600)
    with os.fdopen(fd, "r+b") as stream:
        info = os.fstat(stream.fileno())
        if not stat.S_ISREG(info.st_mode) or info.st_uid != os.geteuid() or info.st_nlink != 1 or stat.S_IMODE(info.st_mode) != 0o600:
            raise ValueError("unsafe desktop lock")
        fcntl.flock(fd, fcntl.LOCK_EX | (0 if blocking else fcntl.LOCK_NB))
        yield


def descriptor(value):
    runtime.validate_manifest(value)
    entry = value.get("bundle")
    if (not isinstance(entry, dict) or set(entry) != {"name", "bytes", "sha256"}
            or entry["name"] != "omabit-" + value["version"] + "-linux-amd64.tar.gz"
            or type(entry["bytes"]) is not int or not 0 < entry["bytes"] <= MAX_ARCHIVE
            or not isinstance(entry["sha256"], str) or not HASH.fullmatch(entry["sha256"])):
        raise ValueError("release requires a bounded, checksum-pinned desktop bundle")
    return {key: copy.deepcopy(value[key]) for key in ("version", "source_commit", "platform", "binaries", "bundle")}


def descriptor_from_release(value):
    assets = value["assets"]
    names = {entry["name"]: entry for entry in assets}
    if len(names) != len(assets):
        raise ValueError("duplicate release asset")
    version = value["version"]
    return descriptor({"version": version, "source_commit": value["source_commit"], "platform": value["platform"],
                       "binaries": {name: names[name + "-linux-amd64"] for name in runtime.COMMANDS},
                       "bundle": names["omabit-" + version + "-linux-amd64.tar.gz"]})


def profile():
    return {"selection": "follow", "revision": 1, "state_schema": 1,
            "active": None, "previous": None, "pending": None, "last_commit": None, "failure": None, "host_setup": False}


class Desktop:
    def __init__(self, pins=None):
        home = Path.home()
        self.state = Path(os.environ.get("XDG_STATE_HOME") or home / ".local/state") / "omabit/bootstrap"
        self.cache = Path(os.environ.get("XDG_CACHE_HOME") or home / ".cache") / "omabit/releases"
        self.pins = Path(pins) if pins else Path(__file__).with_name("release.json")
        runtime.private_directory(self.state)
        runtime.private_directory(self.cache)

    @contextlib.contextmanager
    def transaction(self):
        with lock(self.state / "state.lock"):
            path = self.state / "state.json"
            if os.path.lexists(path):
                info = path.lstat()
                if not stat.S_ISREG(info.st_mode) or info.st_uid != os.geteuid() or info.st_nlink != 1 or stat.S_IMODE(info.st_mode) != 0o600:
                    raise ValueError("unsafe desktop state file; preserve it for review")
                value = read_json(path)
                if (value.get("format") != FORMAT or not isinstance(value.get("profiles"), dict)
                        or value.get("selected") not in value["profiles"] or "default" not in value["profiles"]
                        or not isinstance(value.get("releases"), dict)):
                    raise ValueError("unsupported desktop state; preserve it for a compatible bootstrap")
                for name, item in value["profiles"].items():
                    if not NAME.fullmatch(name) or item.get("state_schema") != 1:
                        raise ValueError("unsupported profile state")
            else:
                default = profile()
                default["host_setup"] = True
                value = {"format": FORMAT, "selected": "default", "profiles": {"default": default}, "releases": {}, "session": None}
            original = canonical(value)
            yield value
            if canonical(value) != original or not path.exists():
                atomic_json(path, value)

    def create_profile(self, name, host_setup=False):
        if not NAME.fullmatch(name) or name == "default":
            raise ValueError("use a new lowercase profile name")
        with self.transaction() as state:
            if name in state["profiles"]:
                raise ValueError("profile exists; create a new name for clean-install testing")
            state["profiles"][name] = dict(profile(), host_setup=host_setup)
        return self.profile_paths(name)

    def profile_paths(self, name):
        if name == "default":
            return {}
        if not NAME.fullmatch(name):
            raise ValueError("invalid profile")
        root = self.state / "profiles" / name
        env = {}
        for variable, leaf in (("XDG_CONFIG_HOME", "config"), ("XDG_STATE_HOME", "state"),
                               ("XDG_DATA_HOME", "data"), ("XDG_CACHE_HOME", "cache")):
            path = root / leaf
            runtime.private_directory(path)
            env[variable] = str(path)
        return env

    def select(self, name, release=None):
        with self.transaction() as state:
            if name not in state["profiles"]:
                raise ValueError("unknown profile")
            selected = state["profiles"][name]
            if release is not None:
                if release != "follow" and release not in state["releases"]:
                    raise ValueError("unknown release; import a verified bundle first")
                selected["selection"] = release
            selected["revision"] += 1
            selected["failure"] = None
            state["selected"] = name

    def resolve(self):
        with self.transaction() as state:
            name = state["selected"]
            item = copy.deepcopy(state["profiles"][name])
            if item["selection"] == "follow":
                release = descriptor(read_json(self.pins, 16384))
                key = identity(release)
                state["releases"][key] = release
            else:
                key = item["selection"]
                release = descriptor(state["releases"][key])
                if identity(release) != key:
                    raise ValueError("release identity changed")
        request = identity({"profile": name, "revision": item["revision"], "release": key})
        return {"format": FORMAT, "profile": name, "release": key, "version": release["version"],
                "request": request, "environment": self.profile_paths(name), "host_setup": item["host_setup"],
                "active": item["active"], "previous": item["previous"], "pending": item["pending"],
                "failure": item["failure"], "blocked": bool(item["failure"] and item["failure"]["request"] == request)}

    def import_release(self, metadata, archive):
        release = descriptor_from_release(read_json(metadata))
        key = identity(release)
        archive = Path(archive)
        entry = release["bundle"]
        if archive.is_symlink() or archive.stat().st_size != entry["bytes"] or runtime.digest(archive) != entry["sha256"]:
            raise ValueError("candidate archive differs from release metadata")
        with lock(self.cache / "prepare.lock"):
            self._prepare(release, archive)
        with self.transaction() as state:
            state["releases"][key] = release
        return {"release": key, "version": release["version"], "source_commit": release["source_commit"]}

    def prepare(self, request=None, release_id=None, name=None):
        plan = self.resolve()
        if request is not None and plan["request"] != request:
            raise ValueError("release selection changed during preparation")
        key = release_id or plan["release"]
        name = name or plan["profile"]
        with self.transaction() as state:
            if name not in state["profiles"] or key not in state["releases"]:
                raise ValueError("unknown release/profile")
            release = descriptor(state["releases"][key])
            item = copy.deepcopy(state["profiles"][name])
        if identity(release) != key:
            raise ValueError("release identity changed")
        with lock(self.cache / "prepare.lock"):
            package = self._prepare(release)
        contract = package["contract"]
        if name != "default" and contract == 0:
            raise ValueError("this legacy UI cannot isolate test profiles; use a desktop-contract-v1 candidate")
        if request is not None and self.resolve()["request"] != request:
            raise ValueError("release selection changed during preparation")
        return dict(package, profile=name, release=key, version=release["version"], request=plan["request"],
                    source_commit=release["source_commit"], environment=self.profile_paths(name), host_setup=item["host_setup"])

    def _prepare(self, release, archive=None):
        key = identity(release)
        destination = self.cache / key
        if os.path.lexists(destination):
            return self.verify(destination, release)
        with tempfile.TemporaryDirectory(prefix=".prepare-", dir=self.cache) as temporary:
            stage = Path(temporary)
            sealed = stage / "bundle.tar.gz"
            if archive:
                shutil.copyfile(archive, sealed)
            else:
                runtime.download(runtime.RELEASES + "v" + release["version"] + "/" + release["bundle"]["name"], sealed, release["bundle"])
            if sealed.stat().st_size != release["bundle"]["bytes"] or runtime.digest(sealed) != release["bundle"]["sha256"]:
                raise ValueError("bundle changed before extraction")
            output = stage / "payload"
            output.mkdir(mode=0o700)
            prefix = "omabit-" + release["version"] + "-linux-amd64"
            seen, total = set(), 0
            with tarfile.open(sealed) as tar:
                for member in tar:
                    path = PurePosixPath(member.name)
                    total += member.size
                    if (path.is_absolute() or ".." in path.parts or not path.parts or path.parts[0] != prefix
                            or member.name != path.as_posix() or member.name in seen or len(seen) >= MAX_FILES
                            or not (member.isdir() or member.isfile()) or member.size < 0 or total > MAX_EXPANDED):
                        raise ValueError("unsafe, duplicate or oversized bundle member")
                    seen.add(member.name)
                    relative = path.relative_to(prefix)
                    if member.isdir():
                        continue
                    if relative.parts[:2] == ("plugins", "omabit"):
                        target = output / "ui" / Path(*relative.parts[2:])
                        mode = 0o444
                    elif relative.as_posix() in {"build/" + name for name in runtime.COMMANDS}:
                        target = output / "bin" / relative.name
                        mode = 0o555
                    elif relative.as_posix() == "BUILD.json":
                        target, mode = output / "BUILD.json", 0o444
                    else:
                        continue
                    target.parent.mkdir(mode=0o700, parents=True, exist_ok=True)
                    with tar.extractfile(member) as source, target.open("xb") as stream:
                        shutil.copyfileobj(source, stream)
                        stream.flush()
                        os.fchmod(stream.fileno(), mode)
                        os.fsync(stream.fileno())
            shutil.copyfile(sealed, output / "bundle.tar.gz")
            (output / "bundle.tar.gz").chmod(0o400)
            with (output / "bundle.tar.gz").open("rb") as stream:
                os.fsync(stream.fileno())
            inventory = {p.relative_to(output).as_posix(): runtime.digest(p) for p in output.rglob("*") if p.is_file()}
            atomic_json(output / "inventory.json", {"format": FORMAT, "release": key, "files": inventory})
            self.verify(output, release)
            for directory in sorted((p for p in output.rglob("*") if p.is_dir()), reverse=True):
                sync_dir(directory)
            sync_dir(output)
            output.rename(destination)
            sync_dir(self.cache)
        return self.verify(destination, release)

    def verify(self, directory, release):
        runtime.private_directory(directory)
        inventory = read_json(directory / "inventory.json")
        if inventory.get("format") != FORMAT or inventory.get("release") != identity(release):
            raise ValueError("unsupported cached inventory")
        actual = {}
        for path in directory.rglob("*"):
            info = path.lstat()
            if info.st_uid != os.geteuid() or not (stat.S_ISREG(info.st_mode) or stat.S_ISDIR(info.st_mode)):
                raise ValueError("unsafe cached payload")
            if path.is_file() and path != directory / "inventory.json":
                if info.st_nlink != 1 or info.st_mode & 0o022:
                    raise ValueError("unsafe cached file permissions")
                actual[path.relative_to(directory).as_posix()] = runtime.digest(path)
        if actual != inventory["files"]:
            raise ValueError("cached desktop payload changed")
        archive = directory / "bundle.tar.gz"
        if actual.get("bundle.tar.gz") != release["bundle"]["sha256"] or archive.stat().st_size != release["bundle"]["bytes"]:
            raise ValueError("cached archive differs from its trusted pin")
        # The public inventory is an optimization/diagnostic, not trust authority.
        # Rebind every UI byte to the retained, checksum-pinned archive as well.
        expected = {"bundle.tar.gz": release["bundle"]["sha256"]}
        prefix = "omabit-" + release["version"] + "-linux-amd64/"
        with tarfile.open(archive) as tar:
            for member in tar:
                if not member.isfile():
                    continue
                relative = member.name.removeprefix(prefix)
                target = None
                if relative.startswith("plugins/omabit/"):
                    target = "ui/" + relative.removeprefix("plugins/omabit/")
                elif relative in {"build/" + name for name in runtime.COMMANDS}:
                    target = "bin/" + relative.removeprefix("build/")
                elif relative == "BUILD.json":
                    target = relative
                if target:
                    with tar.extractfile(member) as stream:
                        expected[target] = hashlib.file_digest(stream, "sha256").hexdigest()
        if actual != expected:
            raise ValueError("cached UI does not match its authenticated archive")
        for name, entry in release["binaries"].items():
            binary = directory / "bin" / name
            if actual.get("bin/" + name) != entry["sha256"] or binary.stat().st_size != entry["bytes"] or not os.access(binary, os.X_OK):
                raise ValueError("cached binary does not match its release")
        build = read_json(directory / "BUILD.json")
        manifest = read_json(directory / "ui/manifest.json")
        if (build["version"] != release["version"] or build["source_commit"] != release["source_commit"]
                or manifest["version"] != release["version"] or manifest["id"] != "thelifeandtimes.omabit"):
            raise ValueError("UI/tool provenance differs from release descriptor")
        for name in ("Service.qml", "Panel.qml", "Model.js"):
            if "ui/" + name not in actual:
                raise ValueError("incomplete desktop payload")
        contract = 0
        if (directory / "ui/DESKTOP.json").exists():
            schema = read_json(directory / "ui/DESKTOP.json")
            if schema != {"bootstrap_api": 1, "profile_state_schema": 1}:
                raise ValueError("desktop/profile contract requires a different bootstrap")
            contract = 1
        return {"directory": str(directory), "ui": (directory / "ui").as_uri(),
                "tools": (directory / "bin").as_uri(), "manifest": manifest, "contract": contract,
                "development": build.get("development")}

    @contextlib.contextmanager
    def lease(self):
        with lock(self.state / "bootstrap.lock", blocking=False):
            session = uuid.uuid4().hex
            with self.transaction() as state:
                state["session"] = session
            try:
                yield session
            finally:
                with self.transaction() as state:
                    if state["session"] == session:
                        state["session"] = None

    def activation(self, action, session, request, message=""):
        plan = self.resolve()
        with self.transaction() as state:
            if not session or state["session"] != session or plan["request"] != request:
                raise ValueError("stale activation acknowledgement")
            # Selection can change between resolve and this lock acquisition.
            current = state["profiles"][state["selected"]]
            selected_key = identity(descriptor(read_json(self.pins, 16384))) if current["selection"] == "follow" else current["selection"]
            if identity({"profile": state["selected"], "revision": current["revision"], "release": selected_key}) != request:
                raise ValueError("selection changed before activation")
            candidate = {"release": plan["release"], "request": request}
            if action == "begin":
                current["pending"] = dict(candidate, from_release=current["active"])
            elif action == "commit":
                pending = current["pending"]
                if not pending or any(pending.get(key) != value for key, value in candidate.items()):
                    raise ValueError("activation has no matching durable intent")
                current["last_commit"] = {"request": request, "from_release": pending["from_release"], "previous": current["previous"]}
                if current["active"] != plan["release"]:
                    current["previous"] = current["active"]
                current["active"], current["pending"], current["failure"] = plan["release"], None, None
            elif action == "fail":
                committed = current.get("last_commit")
                if committed and committed["request"] == request:
                    # A commit can succeed before its stdout acknowledgement is
                    # lost. Restore only this exact UI transaction's prior state.
                    current["active"] = committed["from_release"]
                    current["previous"] = committed["previous"]
                    current["last_commit"] = None
                current["pending"] = None
                current["failure"] = {"request": request, "message": str(message)[:1024], "at": int(time.time())}
            else:
                raise ValueError("unknown activation action")

    def status(self):
        with self.transaction() as state:
            result = copy.deepcopy(state)
        result["paths"] = {name: self.profile_paths(name) for name in result["profiles"]}
        return result
