"""Versioned supported-collection intent journal; no loader or ship effects.

Observations must come from a bound runtime adapter. This module validates their
sequence/identity and persists them; booleans supplied here are not attestations.
The explicit root is independent of Omabit bootstrap/profile and runner state.
"""
import contextlib
import copy
import fcntl
import hashlib
import json
import os
from pathlib import Path
import re
import stat
import uuid

ROLES = ("omabit", "theme", "dms")
PATHS = ("config", "state", "data", "cache")
FORWARD = ("drain-previous", "disable-previous", "checkpoint-appearance", "enable-candidate", "verify-candidate")
ROLLBACK = ("drain-candidate", "disable-candidate", "restore-appearance", "enable-previous", "verify-previous")
MAX_BYTES = 16 << 20
MAX_EVENTS = 4096
HASH = re.compile(r"[0-9a-f]{64}\Z")
ID = re.compile(r"[0-9a-f]{32}\Z")
NAME = re.compile(r"[a-z][a-z0-9-]{0,47}\Z")
PLUGIN = re.compile(r"[a-zA-Z0-9][a-zA-Z0-9_.-]{0,159}\Z")
PROBLEMS = {"loader-unavailable", "artifact-mismatch", "readiness-failed", "appearance-failed", "helper-descendants-unconfirmed", "reader-owner-lost"}


class Conflict(ValueError):
    pass


def require(ok, message="invalid collection journal input"):
    if not ok:
        raise Conflict(message)


def exact(value, keys):
    require(type(value) is dict and set(value) == set(keys))


def token(value, pattern=ID):
    require(type(value) is str and pattern.fullmatch(value))
    return value


def canonical(value):
    return json.dumps(value, sort_keys=True, separators=(",", ":"), allow_nan=False).encode()


def digest(value):
    return hashlib.sha256(canonical(value)).hexdigest()


def absolute(value):
    require(type(value) is str and 1 < len(value) <= 4096 and "\x00" not in value)
    path = Path(value)
    require(path.is_absolute() and str(path) == value and ".." not in path.parts)
    return path


def overlaps(a, b):
    a, b = Path(a), Path(b)
    return a == b or a in b.parents or b in a.parents


def specification(value):
    exact(value, ("format", "name", "members"))
    require(type(value["format"]) is int and value["format"] == 1)
    token(value["name"], NAME)
    exact(value["members"], ROLES)
    roots, identifiers = [], set()
    for role in ROLES:
        member = value["members"][role]
        exact(member, ("plugin_id", "profile", "artifact_sha256", "paths", "endpoint_ref"))
        token(member["plugin_id"], PLUGIN); token(member["profile"], NAME)
        token(member["artifact_sha256"], HASH)
        require(member["plugin_id"] not in identifiers); identifiers.add(member["plugin_id"])
        exact(member["paths"], PATHS)
        for path in member["paths"].values():
            absolute(path)
            require(not any(overlaps(path, old) for old in roots), "collection namespaces overlap")
            roots.append(path)
        if member["endpoint_ref"] is not None:
            token(member["endpoint_ref"], HASH)
    return copy.deepcopy(value)


def binding(value, member):
    exact(value, ("plugin_id", "profile", "artifact_sha256", "instance"))
    token(value["instance"])
    require(all(value[key] == member[key] for key in ("plugin_id", "profile", "artifact_sha256")), "loaded component differs from selected artifact/profile")


def loaded(value, spec, epoch=None):
    exact(value, ("collection", "epoch", "members"))
    require(value["collection"] == digest(spec))
    token(value["epoch"])
    require(epoch is None or value["epoch"] == epoch, "wrong activation epoch")
    exact(value["members"], ROLES)
    for role in ROLES:
        binding(value["members"][role], spec["members"][role])
    require(len({v["instance"] for v in value["members"].values()}) == 3)


def ready(value, spec, expected=None, epoch=None):
    exact(value, ("loaded", "endpoints", "ready"))
    require(value["ready"] is True)
    loaded(value["loaded"], spec, epoch)
    require(expected is None or value["loaded"] == expected, "readiness came from a different loaded generation")
    exact(value["endpoints"], ROLES)
    require(all(value["endpoints"][r] == spec["members"][r]["endpoint_ref"] for r in ROLES), "ready endpoint differs from the selected connection")


def initial():
    return {"collections": {}, "committed": None, "pending": None, "requests": {}}


def effect(request, phase):
    return digest({"request": request["id"], "fingerprint": request["fingerprint"], "phase": phase})[:32]


def target(request, phase):
    return request["candidate"] if phase.endswith("candidate") else request["previous"]["collection"]


def advance(state, request):
    phases = ROLLBACK if request["direction"] == "rollback" else FORWARD
    index = phases.index(request["phase"])
    if index + 1 == len(phases):
        state["committed"] = copy.deepcopy(request["loaded"])
        request["outcome"] = "rolled-back" if request["direction"] == "rollback" else "committed"
        request["phase"] = None
        state["pending"] = None
    else:
        request["phase"] = phases[index + 1]
    request["problem"] = None


def apply(state, event):
    """Reduce one validated event. False means an exact completed-event replay."""
    require(type(event) is dict and type(event.get("type")) is str)
    kind = event["type"]
    if kind == "register":
        exact(event, ("type", "spec"))
        spec = specification(event["spec"]); key = digest(spec)
        if key in state["collections"]:
            return False
        require(len(state["collections"]) < 128, "collection catalog full; preserve existing records")
        for old in state["collections"].values():
            if old["name"] == spec["name"]:
                continue
            for a in spec["members"].values():
                for b in old["members"].values():
                    require(a["plugin_id"] != b["plugin_id"], "different profiles share a plugin identity")
                    require(not any(overlaps(x, y) for x in a["paths"].values() for y in b["paths"].values()), "different profiles share state paths")
        state["collections"][key] = spec
        return True
    if kind == "adopt":
        exact(event, ("type", "observation"))
        observation = event["observation"]
        require(type(observation) is dict and type(observation.get("loaded")) is dict)
        key = observation["loaded"].get("collection")
        require(key in state["collections"])
        ready(observation, state["collections"][key])
        if state["committed"] == observation["loaded"] and not state["requests"]:
            return False
        require(state["committed"] is None and not state["requests"], "initial adoption cannot reset history")
        state["committed"] = copy.deepcopy(observation["loaded"])
        return True
    if kind == "begin":
        exact(event, ("type", "request", "candidate", "expected"))
        token(event["request"]); token(event["candidate"], HASH); token(event["expected"], HASH)
        fingerprint = digest({"candidate": event["candidate"], "expected": event["expected"]})
        if event["request"] in state["requests"]:
            require(state["requests"][event["request"]]["fingerprint"] == fingerprint, "request identity reused for another transition")
            return False
        require(state["pending"] is None, "another transition is unfinished")
        require(state["committed"] is not None and event["expected"] == digest(state["committed"]), "committed collection changed; inspect before switching")
        require(event["candidate"] in state["collections"], "candidate is not registered")
        request = {"id": event["request"], "fingerprint": fingerprint, "candidate": event["candidate"], "previous": copy.deepcopy(state["committed"]), "phase": FORWARD[0], "direction": "forward", "outcome": None, "receipts": {}, "uncertain": [], "checkpoint": None, "loaded": None, "problem": None}
        state["requests"][request["id"]] = request; state["pending"] = request["id"]
        return True
    require(kind in ("drained", "completed", "problem", "rollback"))
    expected_keys = ("type", "request", "phase", "effect", "role", "observation") if kind == "drained" else ("type", "request", "phase", "effect", "reason") if kind in ("problem", "rollback") else ("type", "request", "phase", "effect", "observation")
    exact(event, expected_keys)
    token(event["request"]); token(event["effect"])
    require(event["request"] in state["requests"])
    request = state["requests"][event["request"]]
    phase = event["phase"]
    require(phase in FORWARD + ROLLBACK and event["effect"] == effect(request, phase), "wrong effect identity")
    key = event["effect"] + ("/" + str(event["role"]) if kind == "drained" else "/problem/" + str(event["reason"]) if kind == "problem" else "/" + kind)
    if key in request["receipts"]:
        require(request["receipts"][key] == event, "completed effect replay changed")
        return False
    require(state["pending"] == request["id"] and request["phase"] == phase, "stale transition callback")
    if kind in ("problem", "rollback"):
        require(event["reason"] in PROBLEMS)
        if event["reason"] in ("helper-descendants-unconfirmed", "reader-owner-lost"):
            request["uncertain"].append({"phase": phase, "reason": event["reason"]})
        request["problem"] = event["reason"]
        if kind == "rollback":
            require(phase in ("enable-candidate", "verify-candidate") and request["checkpoint"] is not None and not request["uncertain"], "rollback needs a retained checkpoint and confirmed old shutdown")
            request["direction"] = "rollback"; request["phase"] = ROLLBACK[0]
        request["receipts"][key] = copy.deepcopy(event)
        return True
    require(not request["uncertain"], "unconfirmed helper lineage requires independent recovery; reload is not cleanup")
    observation = event["observation"]
    if kind == "drained":
        require(phase in ("drain-previous", "drain-candidate") and event["role"] in ROLES)
        exact(observation, ("epoch", "binding", "absent", "closed", "retained_state", "pending_ids"))
        require(observation["closed"] is True and type(observation["absent"]) is bool)
        role = event["role"]
        old = request["previous"] if phase == "drain-previous" else request["loaded"]
        epoch = request["previous"]["epoch"] if phase == "drain-previous" else effect(request, "enable-candidate")
        require(observation["epoch"] == epoch, "drain receipt came from another activation")
        spec = state["collections"][target(request, phase)]
        if observation["absent"]:
            require(phase == "drain-candidate" and old is None and observation["binding"] is None, "missing old UI is not a closed receipt")
        else:
            binding(observation["binding"], spec["members"][role])
            require(old is None or observation["binding"] == old["members"][role], "drain receipt came from another component instance")
        token(observation["retained_state"], HASH)
        require(type(observation["pending_ids"]) is list and len(observation["pending_ids"]) <= 1024)
        for pending in observation["pending_ids"]: token(pending)
        require(len(set(observation["pending_ids"])) == len(observation["pending_ids"]))
        request["receipts"][key] = copy.deepcopy(event)
        if all(event["effect"] + "/" + r in request["receipts"] for r in ROLES): advance(state, request)
        return True
    require(phase not in ("drain-previous", "drain-candidate"))
    if phase in ("disable-previous", "disable-candidate"):
        exact(observation, ("epoch", "absent", "helpers_exited"))
        expected_epoch = request["previous"]["epoch"] if phase == "disable-previous" else effect(request, "enable-candidate")
        require(observation["epoch"] == expected_epoch and observation["absent"] is True and observation["helpers_exited"] is True)
    elif phase == "checkpoint-appearance":
        exact(observation, ("checkpoint",)); token(observation["checkpoint"], HASH)
        request["checkpoint"] = observation["checkpoint"]
    elif phase == "restore-appearance":
        exact(observation, ("checkpoint", "verified"))
        require(observation["checkpoint"] == request["checkpoint"] and observation["verified"] is True)
    elif phase in ("enable-candidate", "enable-previous"):
        spec = state["collections"][target(request, phase)]
        loaded(observation, spec, event["effect"])
        request["loaded"] = copy.deepcopy(observation)
    else:
        spec = state["collections"][target(request, phase)]
        ready(observation, spec, request["loaded"])
    request["receipts"][key] = copy.deepcopy(event)
    advance(state, request)
    return True


class Lease:
    def __init__(self, store):
        self.store, self.pid, self.token, self.live = store, os.getpid(), uuid.uuid4().hex, True


class Store:
    """Explicit private directory, serialized coordinator lease, atomic journal."""
    def __init__(self, root, *, create=False):
        self.root = absolute(str(root))
        require(self.root.resolve() == self.root, "journal path must not contain symlinks")
        self.fresh = bool(create)
        if create:
            self.root.mkdir(mode=0o700)  # Existing paths are never reinitialized.
            parent = os.open(self.root.parent, os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW | os.O_CLOEXEC)
            try: os.fsync(parent)
            finally: os.close(parent)
        info = self.root.lstat()
        require(stat.S_ISDIR(info.st_mode) and info.st_uid == os.geteuid() and stat.S_IMODE(info.st_mode) == 0o700, "journal directory must be private and owned")
        self.fd = os.open(self.root, os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW | os.O_CLOEXEC)

    def close(self):
        if self.fd is not None: os.close(self.fd); self.fd = None

    def _open(self, name, flags):
        fd = os.open(name, flags | os.O_NOFOLLOW | os.O_NONBLOCK | os.O_CLOEXEC, 0o600, dir_fd=self.fd)
        info = os.fstat(fd)
        if not (stat.S_ISREG(info.st_mode) and info.st_uid == os.geteuid() and info.st_nlink == 1 and stat.S_IMODE(info.st_mode) == 0o600 and info.st_size <= MAX_BYTES):
            os.close(fd); raise Conflict("journal files must be bounded private owned regular files")
        return fd

    @contextlib.contextmanager
    def lease(self):
        fd = self._open("coordinator.lock", os.O_RDWR | os.O_CREAT)
        lease = None
        try:
            fcntl.flock(fd, fcntl.LOCK_EX | fcntl.LOCK_NB)
            lease = Lease(self)
            yield lease
        finally:
            if lease: lease.live = False
            os.close(fd)

    def _load(self):
        def unique(pairs):
            value = {}
            for key, entry in pairs:
                require(key not in value, "duplicate journal JSON key"); value[key] = entry
            return value
        try: fd = self._open("journal.json", os.O_RDONLY)
        except FileNotFoundError:
            require(self.fresh, "journal is missing; never recreate prior collection history")
            require(not any(n.startswith(".pending-") for n in os.listdir(self.fd)), "missing journal with retained interrupted write")
            return [], initial()
        with os.fdopen(fd, "rb") as stream: data = stream.read(MAX_BYTES + 1)
        require(len(data) <= MAX_BYTES)
        value = json.loads(data, object_pairs_hook=unique, parse_constant=lambda _: (_ for _ in ()).throw(Conflict("non-finite JSON")))
        exact(value, ("format", "events")); require(type(value["format"]) is int and value["format"] == 1, "unsupported collection journal; preserve it")
        require(type(value["events"]) is list and len(value["events"]) <= MAX_EVENTS)
        state = initial()
        for event in value["events"]: require(apply(state, event), "journal contains a redundant event")
        return value["events"], state

    def status(self):
        return self._load()[1]

    def record(self, lease, event):
        require(lease.store is self and lease.live and lease.pid == os.getpid(), "a live coordinator lease is required")
        events, state = self._load()
        if not apply(state, copy.deepcopy(event)):
            os.fsync(self.fd)
            return state
        require(len(events) < MAX_EVENTS, "journal full; preserve history before another transition")
        data = canonical({"format": 1, "events": events + [event]}) + b"\n"
        require(len(data) <= MAX_BYTES, "journal size limit; preserve existing state")
        temporary = ".pending-" + uuid.uuid4().hex
        fd = self._open(temporary, os.O_WRONLY | os.O_CREAT | os.O_EXCL)
        # Keep a failed temporary for diagnosis. The prior durable journal stays
        # authoritative; no recovery path deletes or guesses from temporary bytes.
        with os.fdopen(fd, "wb") as stream:
            stream.write(data); stream.flush(); os.fsync(stream.fileno())
        os.replace(temporary, "journal.json", src_dir_fd=self.fd, dst_dir_fd=self.fd)
        self.fresh = False
        os.fsync(self.fd)
        return state
