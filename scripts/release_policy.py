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


def version_order(value):
    core, kind, iteration = version(value)
    return (*core, {"edge": 0, "rc": 1, None: 2}[kind], iteration)


def validate_publication(target, previous, backend_changed, channel="stable"):
    # This is packaging admission, including reproducibility of an existing
    # version into a fresh output directory. Stable promotion separately requires
    # a NEW version and qualified upgrades, regardless of which component changed.
    new, kind, _ = version(target)
    version(previous)
    if channel not in ("stable", "edge", "develop"):
        raise ValueError("unknown distribution channel")
    if channel == "develop":
        raise ValueError("develop is unqualified; use the development snapshot builder, not public release packaging")
    if channel == "edge" and kind != "edge":
        raise ValueError("edge artifacts must use immutable -edge.N prerelease versions")
    if channel == "stable" and kind == "edge":
        raise ValueError("edge artifacts cannot be published as the stable channel")
    if version_order(target) < version_order(previous):
        raise ValueError("release version regressed")


def validate_master_promotion(previous, target, head_branch):
    _, kind, _ = version(target)
    if kind is not None:
        raise ValueError("master accepts only qualified stable releases X.Y.Z")
    if head_branch != "release/" + target:
        raise ValueError("master changes must come through the matching release/X.Y.Z PR")
    if version_order(target) <= version_order(previous):
        raise ValueError("master requires a newer release; published versions are immutable")


# These govern review/publication and are not shipped UI, runtime or installer
# inputs. Packaging, bootstrap, workflow and skill changes still require a freeze.
GOVERNANCE_FILES = frozenset({"AGENTS.md", "CONTRIBUTING.md",
    "scripts/release_policy.py", "scripts/check-release-policy.py",
    "scripts/test-release-policy.py"})


def validate_distribution_changes(paths):
    for path in paths:
        if path not in GOVERNANCE_FILES | {"native/bootstrap/release.json", "README.md"} and not path.startswith("docs/"):
            raise ValueError("runtime/UI/build inputs changed after artifact source freeze: " + path)


def validate_upgrade_paths(upgrade, previous, target, source, read_blob):
    paths = upgrade.get("paths")
    if not isinstance(paths, list) or not 0 < len(paths) <= 64:
        raise ValueError("declare tested upgrade paths, including the previous stable release")
    supports_previous = False
    for path in paths:
        if not isinstance(path, dict):
            raise ValueError("invalid upgrade path")
        versions = path.get("versions")
        if not isinstance(versions, list) or not 2 <= len(versions) <= 16 or versions[-1] != target:
            raise ValueError("upgrade path must finish at the proposed release")
        for value in versions:
            if version(value)[1] is not None:
                raise ValueError("upgrade path must name stable versions")
        if any(version_order(a) >= version_order(b) for a, b in zip(versions, versions[1:])):
            raise ValueError("upgrade path must progress without resets or downgrades")
        evidence = path.get("evidence", "")
        digest = path.get("evidence_sha256", "")
        if not re.fullmatch(r"docs/releases/[A-Za-z0-9._/-]+\.json", evidence) or any(part in (".", "..", "") for part in evidence.split("/")):
            raise ValueError("upgrade evidence must be a repository release JSON document")
        if not re.fullmatch(r"[0-9a-f]{64}", digest):
            raise ValueError("upgrade evidence needs an exact SHA256")
        data = read_blob(evidence)
        if len(data) > 131072 or hashlib.sha256(data).hexdigest() != digest:
            raise ValueError("upgrade evidence differs from its recorded hash")
        result = json.loads(data)
        if not isinstance(result, dict) or result.get("passed") is not True:
            raise ValueError("upgrade evidence must record a passed rehearsal")
        if result.get("source_commit") != source and result.get("sources", {}).get(target) != source:
            raise ValueError("upgrade evidence must bind the frozen target source")
        supports_previous |= previous in versions[:-1]
    if not supports_previous:
        raise ValueError("no tested upgrade path from the previous stable release")
