"""Version/channel policy shared by release packaging and trusted PR checks."""
import hashlib
import json
from pathlib import Path
import re

VERSION = re.compile(r"(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-(rc|edge)\.([0-9]+))?\Z")
BACKEND_PREFIXES = ("backend/", "internal/", "deploy/")
BACKEND_FILES = {"go.mod", "go.sum", "scripts/install-services.sh", "scripts/package-release.py", "scripts/release_policy.py",
                 "native/bootstrap/runtime.py", "native/bootstrap/desktop.py", "native/bootstrap/control.py"}


def version(value):
    match = VERSION.fullmatch(value or "")
    if not match:
        raise ValueError("invalid release version")
    return tuple(map(int, match.group(1, 2, 3))), match[4], int(match[5] or 0)


def backend_path(path):
    return path.startswith(BACKEND_PREFIXES) or path in BACKEND_FILES


def backend_digest(root):
    root = Path(root)
    files = {}
    paths = [p for directory in BACKEND_PREFIXES for p in (root / directory).rglob("*") if p.is_file()]
    paths += [root / name for name in BACKEND_FILES if (root / name).is_file()]
    for path in paths:
        if path.is_symlink():
            raise ValueError("backend source must not contain symlinks")
        files[path.relative_to(root).as_posix()] = {"sha256": hashlib.sha256(path.read_bytes()).hexdigest(),
                                                  "executable": bool(path.stat().st_mode & 0o111)}
    return hashlib.sha256(json.dumps(files, sort_keys=True, separators=(",", ":")).encode()).hexdigest()


def validate_publication(target, previous, backend_changed, channel="stable"):
    new, kind, _ = version(target)
    old, old_kind, _ = version(previous)
    if channel not in ("stable", "edge", "develop"):
        raise ValueError("unknown distribution channel")
    if channel == "develop":
        raise ValueError("develop is unqualified; use the development snapshot builder, not public release packaging")
    if channel == "edge" and kind != "edge":
        raise ValueError("edge artifacts must use immutable -edge.N prerelease versions")
    if channel == "stable" and kind == "edge":
        raise ValueError("edge artifacts cannot be published as the stable channel")
    if kind is None and new[2] != 0:
        raise ValueError("stable releases must be full minor/major versions X.Y.0")
    if new < old:
        raise ValueError("release version regressed")
    if backend_changed and new[:2] <= old[:2]:
        # A new minor's RC/edge iterations may converge on that same upcoming
        # line. A UI patch line (nonzero patch) never grants this exception.
        if not (old_kind and old[2] == 0 and new == old):
            raise ValueError("runner/collector-affecting changes require a new minor or major line")


def validate_master_promotion(previous, target, head_branch):
    old, old_kind, _ = version(previous)
    new, kind, _ = version(target)
    if kind is not None or new[2] != 0:
        raise ValueError("master accepts only stable minor/major releases X.Y.0")
    if head_branch != "release/" + target:
        raise ValueError("master changes must come through the matching release/X.Y.0 PR")
    if not (new[:2] > old[:2] or old_kind == "rc" and new == old):
        raise ValueError("master cannot receive a same-version or patch-only update")
