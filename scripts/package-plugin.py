#!/usr/bin/env python3
"""Reproduce a legacy flattened plugin export from a frozen binary release."""
import argparse
import hashlib
import json
from pathlib import Path
import shutil
import subprocess
import tarfile
import tempfile

ROOT = Path(__file__).resolve().parents[1]
COMMANDS = ("omabit", "urbit-collector", "urbit-runner")


def digest(path):
    with path.open("rb") as stream:
        return hashlib.file_digest(stream, "sha256").hexdigest()


def generate(assets, destination, distribution_commit):
    release = json.loads((assets / "release.json").read_text())
    version = release["version"]
    names = {}
    for line in (assets / "SHA256SUMS").read_text().splitlines():
        expected, name = line.split("  ", 1)
        if name in names or Path(name).name != name or name in (".", "..") or digest(assets / name) != expected:
            raise RuntimeError("Invalid release checksum: " + name)
        names[name] = expected
    archive_name = "omabit-" + version + "-linux-amd64.tar.gz"
    entries = {entry["name"]: entry for entry in release["assets"]}
    if archive_name not in entries or len(entries) != len(release["assets"]):
        raise RuntimeError("Release must contain one unambiguous binary archive")
    for name, entry in entries.items():
        if name not in names or entry["sha256"] != names[name] or (assets / name).stat().st_size != entry["bytes"]:
            raise RuntimeError("Release metadata differs from checksums: " + name)
    pins = {"version": version, "source_commit": release["source_commit"], "platform": release["platform"],
            "binaries": {name: entries[name + "-linux-amd64"] for name in COMMANDS}}
    # Validate the generated pins with the same code used on the desktop.
    import importlib.util
    spec = importlib.util.spec_from_file_location("native_runtime", ROOT / "native/bootstrap/runtime.py")
    runtime = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(runtime)
    runtime.validate_manifest(pins)
    if destination.exists() or destination.is_symlink():
        raise RuntimeError("Output already exists; inspect rather than overwrite")
    with tempfile.TemporaryDirectory(prefix=".native-plugin-", dir=destination.parent) as temporary:
        working = Path(temporary)
        prefix = "omabit-" + version + "-linux-amd64"
        with tarfile.open(assets / archive_name) as archive:
            for member in archive.getmembers():
                path = Path(member.name)
                if (path.is_absolute() or ".." in path.parts or path.parts[0] != prefix
                        or not (member.isdir() or member.isfile())):
                    raise RuntimeError("Unsafe archive member")
            archive.extractall(working, filter="data")
        package = working / prefix
        build = json.loads((package / "BUILD.json").read_text())
        if build["source_commit"] != release["source_commit"] or build["version"] != version:
            raise RuntimeError("Archive source/version mismatch")
        output = working / "plugin"
        shutil.copytree(package / "plugins/omabit", output)
        manifest = json.loads((output / "manifest.json").read_text())
        if manifest["version"] != version or manifest["id"] != "thelifeandtimes.omabit":
            raise RuntimeError("Plugin identity/version mismatch")
        (output / "bootstrap").mkdir()
        shutil.copy2(ROOT / "native/bootstrap/runtime.py", output / "bootstrap/runtime.py")
        (output / "bootstrap/release.json").write_text(json.dumps(pins, indent=2) + "\n")
        (output / "bin").mkdir()
        for name in COMMANDS:
            shutil.copy2(ROOT / "native/bootstrap/launch.py", output / "bin" / name)
            (output / "bin" / name).chmod(0o755)
        provenance = {"generated": True, "version": version, "release_source_commit": release["source_commit"],
                      "distribution_source_commit": distribution_commit, "archive_sha256": names[archive_name],
                      "release_url": "https://github.com/thelifeandtimes/omabit/releases/tag/v" + version}
        (output / "DISTRIBUTION.json").write_text(json.dumps(provenance, indent=2) + "\n")
        (output / ".gitignore").write_text("__pycache__/\n*.pyc\n")
        (output / "README.md").write_text(f'''# Omabit for Omarchy — {version}

Generated installation repository. Authoritative source and issues:
<https://github.com/thelifeandtimes/omabit>. See `DISTRIBUTION.json` for provenance.
Make source changes there; do not hand-maintain this generated copy.

## Install

Run in your logged-in Omarchy desktop as your ordinary user:

```bash
omarchy plugin add https://github.com/thelifeandtimes/omabit.git --enable
```

`omarchy plugin install` is an alias for `add`. Omarchy handles cloning, validation,
trust confirmation, bar placement and enabling. No sudo, local release directory,
Go compiler or shell install hook is required.

On first use, the plugin downloads the three prebuilt **{version}** release tools
over HTTPS and verifies their exact sizes and SHA-256 hashes. This is approximately
38 MB. They are cached under `$XDG_CACHE_HOME/omabit/native-binaries/` (normally
`~/.cache/omabit/native-binaries/`), outside the git checkout and global PATH.
Later starts use that verified cache, including offline starts.

**Pause after installation to inspect the UI.** When ready, choose
**Setup → Set up this computer**. Host setup and its OS authorization are separate;
merely installing/enabling the plugin does not create a runner or start any ships.
Linux x86-64 and Python 3.11+ are required for these pinned tools.

If downloading fails, check network access and use Refresh, or run
`~/.config/omarchy/plugins/thelifeandtimes.omabit/bin/urbit-collector --help`
to see the diagnostic. A failed download does not publish a partial binary cache.

## Update

Finish active desktop submissions, then unload the plugin while Git replaces QML:

```bash
omarchy plugin disable thelifeandtimes.omabit
omarchy plugin update thelifeandtimes.omabit
omarchy plugin enable thelifeandtimes.omabit
```

New binary pins use a separate cache directory. Plugin update does not itself
upgrade a runner host or restart ships. Keep host updates explicit.

## Candidate scope

This is the {version} development plugin; its runner contract is recorded in the release metadata. Its Recommended
Apps source and frozen artifact provenance are documented by that release.
Release notes: {provenance['release_url']}.
''')
        output.rename(destination)
    return provenance


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("assets", type=Path)
    parser.add_argument("--output", type=Path, required=True)
    args = parser.parse_args()
    if subprocess.check_output(["git", "status", "--porcelain"], cwd=ROOT, text=True).strip():
        parser.error("commit distribution source before generating a publishable plugin")
    commit = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=ROOT, text=True).strip()
    result = generate(args.assets.resolve(), args.output.absolute(), commit)
    if (subprocess.check_output(["git", "status", "--porcelain"], cwd=ROOT, text=True).strip()
            or subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=ROOT, text=True).strip() != commit):
        raise RuntimeError("Distribution source changed during generation; do not publish this output")
    print(json.dumps(result, indent=2))


if __name__ == "__main__":
    main()
