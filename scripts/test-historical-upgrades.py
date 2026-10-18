#!/usr/bin/env python3
"""G02: reproduce historical writers in isolated git-archive exports, then test upgrades.

Default: run the checked-in fixture regression tests (no Git history required).
--generate: regenerate synthetic JSON/SQLite logical exports using original source.
--check-generated: regenerate and byte-compare, without updating checked-in fixtures.
Requires Go >=1.25 (testing/synctest) and cached historical module dependencies.
Never edits a historical production file, changes a checkout, or uses a live runtime.
The schema14 source bundle retains the original stash and its untracked-file parent;
it is imported only into a private temporary Git repository, never the checkout.
The support manifest distinguishes release database coverage from the narrower
schema19 owner-user host bridge. Unavailable prototype writers are explicit gaps.
"""

import argparse
import base64
import hashlib
import io
import json
import os
from pathlib import Path
import sqlite3
import subprocess
import tarfile
import tempfile

ROOT = Path(__file__).resolve().parents[1]
DATA = ROOT / "internal/managed/testdata/historical-upgrades"
SCHEMA14 = {
    "commit": "9e956bbab517dc9debe8161c09100e3211df0786",
    "tree": "9170c9feea349a678f8153fe106d12e20657353a",
    "base": "87c8dd3fce80a1692436cd61941e2143093aef12",
    "supplement_commit": "043e1b971800f4e4543e6884d20ef625ec55abf4",
    "supplement_tree": "e0e58b7f91a9637920e124191d7e662feb1bb584",
    "bundle_sha256": "09e2f733ae992826ee21aeee32cc2826bed5b46cd8f9db613b747ecf0ca86556",
}
# RC.10's published source is f6981e6, as recorded by 7fe3691's release.json.
SOURCES = [
    ("rc1", "v0.1.0-rc.1", 13), ("preview14", SCHEMA14["commit"], 14),
    ("rc2", "v0.1.0-rc.2", 15),
    *[(f"rc{n}", f"v0.1.0-rc.{n}", 16) for n in range(3, 10)],
    ("preview18", "ba4ba805221a6ef3e29829c9bda1bfeff0c0c7c2", 18),
    ("rc10", "f6981e6ce4b9bb1bddc1334ff23a3f1ae9e9794a", 19),
    ("rc11", "abfff0529a19adc02cf9f4b93eeef78c645dfa8f", 19),
]
# Pin peeled commits independently of mutable refs. RC.2 has an annotated tag;
# its tag object is not the source commit. RC.10/11 also have public release tags,
# but their original commit refs work in local histories without those tags.
RELEASE_COMMITS = [
    "d67ea3ad6d1a3d768bf22bea73e594d8c799742e",
    "691eea7f808237458c1bf0c8865a62cd5cee11dc",
    "1b6b79193c6ac58f378ed5c6c689fe9d89045257",
    "919ba0ce445e3e952d95ab4bff167e938dd9fdda",
    "b973b5a9cc07197531cfbd1a5e8f60444772e5d2",
    "2786969d308da9a5b4c948bfc93e3892449657d8",
    "07e407a4d4c5b4142425547c55094e218bee0697",
    "f972dee22d20070de35b6f26a6ce3d4e1d2497de",
    "d9b180fdd1ec5c34084c93109b005b9af4d49c7e",
    "f6981e6ce4b9bb1bddc1334ff23a3f1ae9e9794a",
    "abfff0529a19adc02cf9f4b93eeef78c645dfa8f",
]
SOURCE_COMMITS = {f"rc{n}": commit for n, commit in enumerate(RELEASE_COMMITS, 1)}
SOURCE_COMMITS.update(preview14=SCHEMA14["commit"], preview18="ba4ba805221a6ef3e29829c9bda1bfeff0c0c7c2")
GAPS = {
    "11": "Unpublished prototype: no original writer found in local objects/reflogs/stash or public omabit/tend history. Consolidated commit 34d8ea55 advances schema1 directly to schema13, accepting 11/12 only as inputs. No RC.1-11 release writes schema11.",
    "12": "Unpublished prototype: no original writer found in local objects/reflogs/stash or public omabit/tend history. The schema13 writer accepts schema12 but does not produce it. No RC.1-11 release writes schema12.",
    "17": "Unpublished prototype: no original writer found in local objects/reflogs/stash or public omabit/tend history. Runtime-settings commit ba4ba805 advances schema16 directly to schema18. No RC.1-11 release writes schema17.",
}


def run(*args, cwd=ROOT, env=None):
    return subprocess.check_output(args, cwd=cwd, env=env)


def sha(data):
    return hashlib.sha256(data).hexdigest()


def encode(data):
    return base64.b64encode(data).decode("ascii")


def document(value):
    return (json.dumps(value, indent=2, ensure_ascii=True) + "\n").encode()


def support_manifest():
    schemas = {name: schema for name, _, schema in SOURCES}
    return document({
        "format": 2,
        "qualification_target": {"release": "v0.1.0-rc.12", "channel": "prerelease", "state_schema": 20,
                                 "evidence": "source-tests-only"},
        "qualification_note": "docs/reviews/2026-10-18-rc12-historical-upgrades.md",
        "accepted_schema_window": list(range(11, 20)),
        "supported_release_scope": "database-reader",
        "supported_releases": [
            {"release": f"v0.1.0-rc.{n}", "schema": schemas[f"rc{n}"], "fixture": f"rc{n}.json",
             "source_commit": commit,
             "host_upgrade": "owner-user-schema19-offline" if schemas[f"rc{n}"] == 19 else "unsupported"}
            for n, commit in enumerate(RELEASE_COMMITS, 1)
        ],
        "host_upgrade": {"initial_schemas": [19], "deployment": "owner-user-linux-amd64",
                         "procedure": "docs/management-offline-upgrade.md", "evidence": "source-tests-only"},
        "prototype_schemas": [11, 12, 14, 17, 18],
        "preview_fixtures": ["preview14.json", "preview18.json"],
        "fixtures": [name + ".json" for name, _, _ in SOURCES],
        "unavailable_writers": GAPS,
    })


def schema14_history(scratch, env):
    bundle = DATA / "schema14-source.bundle"
    assert sha(bundle.read_bytes()) == SCHEMA14["bundle_sha256"], "schema14 source bundle digest changed"
    history = scratch / "schema14-history.git"
    run("git", "init", "--bare", "--initial-branch=provenance", str(history), env=env)
    # The public RC.1 ancestor supplies the bundle's prerequisite. Neither fetch
    # writes to ROOT, and the bundle is sufficient even when ROOT has no stash.
    run("git", "fetch", "--no-tags", "--no-write-fetch-head", str(ROOT),
        SCHEMA14["base"] + ":refs/provenance/base", cwd=history, env=env)
    run("git", "fetch", "--no-tags", "--no-write-fetch-head", str(bundle),
        "refs/stash:refs/provenance/preview14", cwd=history, env=env)
    for ref, expected in ((SCHEMA14["commit"] + "^{tree}", SCHEMA14["tree"]),
                          (SCHEMA14["commit"] + "^3", SCHEMA14["supplement_commit"]),
                          (SCHEMA14["supplement_commit"] + "^{tree}", SCHEMA14["supplement_tree"])):
        assert run("git", "rev-parse", ref, cwd=history, env=env).decode().strip() == expected
    return history


def generate(name, ref, schema, scratch, env):
    history = schema14_history(scratch, env) if name == "preview14" else ROOT
    commit = run("git", "rev-parse", ref + "^{commit}", cwd=history).decode().strip()
    assert commit == SOURCE_COMMITS[name], (name, "original source ref changed", commit)
    tree = run("git", "rev-parse", commit + "^{tree}", cwd=history).decode().strip()
    source = scratch / name
    source.mkdir()
    snapshots = [commit]
    if name == "preview14":
        snapshots.append(SCHEMA14["supplement_commit"])
    for snapshot in snapshots:
        archive = run("git", "archive", "--format=tar", snapshot, cwd=history)
        with tarfile.open(fileobj=io.BytesIO(archive)) as tar:
            # A stash stores originally untracked files in its third parent.
            # Restore those exact blobs without replacing any tracked source.
            for member in tar.getmembers():
                assert member.isdir() or not (source / member.name).exists(), member.name
            tar.extractall(source, filter="data")
    output = source / "g02-output"
    output.mkdir(mode=0o700)
    template = (DATA / "generate_managed_test.go.txt").read_text()
    replacements = {
        "SCHEMA": str(schema),
        "STORAGE_VALIDATOR": "s.validateStorage = func(string) error { return nil }" if schema >= 14 else "",
        "NAMED_ROOT": 'req.PierPath = "@ssd/" + req.PierPath' if schema >= 14 else "",
        "SETTINGS": ('req.Config.NativeControl = true; req.Config.Loom = 34; '
                     + ('if id == "running" { req.Config.Loom = 29 }' if name != "rc11" else "")) if schema >= 18 else "",
        "HANDOFF_SETTINGS": "h.RuntimeSettings = &protocol.RuntimeSettings{Loom: 34, NativeControl: true}" if schema >= 19 else "",
        # Fixed fake filesystem identities, marshaled by the original type.
        # No filesystem identity from this machine is exported or rebound.
        "ROOT_REGISTRY": '''roots := []pierstore.StorageRoot{{ID: "ssd", Path: "/synthetic/g02/custom-piers", Identity: pierstore.DirectoryIdentity{Device: 7, Inode: 90, Birth: "1000000000.000000000"}}}
        body, err := json.Marshal(roots); check(err)
        check(os.WriteFile(filepath.Join(dir, ".omabit-storage.json"), body, 0600))''' if schema >= 14 else "",
    }
    for key, value in replacements.items():
        assert template.count("/*" + key + "*/") == 1
        template = template.replace("/*" + key + "*/", value)
    injected = source / "internal/managed/g02_generate_historical_test.go"
    assert not injected.exists()
    injected.write_text(template)
    collection = (DATA / "generate_collection_test.go.txt").read_bytes()
    target = source / "internal/collection/g02_generate_historical_test.go"
    assert not target.exists()
    target.write_bytes(collection)
    local_env = dict(env, G02_OUTPUT_DIR=str(output))
    for package in ("managed", "collection"):
        subprocess.run(["go", "test", "-count=1", "-timeout=60s", f"./internal/{package}",
                        "-run=^TestG02GenerateHistorical"], cwd=source, env=local_env, check=True)
    db = sqlite3.connect(f"file:{output}/state/runner.sqlite?mode=ro", uri=True)
    actual_schema = db.execute("PRAGMA user_version").fetchone()[0]
    assert actual_schema == schema, (name, actual_schema, schema)
    tables = []
    hashes = {}
    for table, sql in db.execute("SELECT name,sql FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name"):
        assert table.replace("_", "").isalnum()
        cursor = db.execute(f'SELECT * FROM "{table}" ORDER BY rowid')
        columns = [c[0] for c in cursor.description]
        rows = []
        for row in cursor:
            rows.append([{"blob": encode(v)} if isinstance(v, bytes) else v for v in row])
            for column, value in zip(columns, row):
                if column in ("body", "runtime", "fingerprint") and value is not None:
                    hashes[f"{table}/{row[0]}/{column}"] = sha(value if isinstance(value, bytes) else value.encode())
        tables.append({"name": table, "sql": sql, "columns": columns, "rows": rows})
    db.close()
    files = []
    paths = ["authorized_keys", "known_hosts", "collector/runners.json"]
    if schema >= 14:
        paths.append(".omabit-storage.json")
    for path in paths:
        p = output / path
        body = p.read_bytes()
        files.append({"path": path, "mode": 384, "body": encode(body), "sha256": sha(body)})
    fixture = {
        "format": 1, "name": name, "source_commit": commit, "source_tree": tree, "source_ref": ref,
        "schema": schema, "writer_sha256": sha(template.encode() + collection),
        "synthetic": True, "runner_id": (output / "runner-id").read_text(),
        "commitment_bytes": encode((output / "commitment.bin").read_bytes()),
        "tables": tables, "hashes": hashes, "files": files,
    }
    if name == "preview14":
        fixture.update(source_supplement_commit=SCHEMA14["supplement_commit"],
                       source_supplement_tree=SCHEMA14["supplement_tree"],
                       source_bundle_sha256=SCHEMA14["bundle_sha256"])
    result = document(fixture)
    assert len(result) <= 96 * 1024, (name, len(result))
    return result


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    flags = parser.add_mutually_exclusive_group()
    flags.add_argument("--generate", action="store_true")
    flags.add_argument("--check-generated", action="store_true")
    parser.add_argument("--only", choices=[s[0] for s in SOURCES], help="generate/check one historical release")
    args = parser.parse_args()
    if args.only and not (args.generate or args.check_generated):
        parser.error("--only requires generation or reproducibility checking")
    if args.generate or args.check_generated:
        manifest = support_manifest()
        # Exclude ambient overlays, workspace/toolchain configuration and live home state.
        env = {"PATH": os.environ["PATH"], "GOENV": "off", "GOWORK": "off", "GOFLAGS": "-mod=readonly",
               "GOTOOLCHAIN": "local", "GOPROXY": "off", "GOSUMDB": "off", "CGO_ENABLED": "0", "TZ": "UTC"}
        for key in ("GOCACHE", "GOMODCACHE"):
            env[key] = run("go", "env", key, env=dict(os.environ, GOENV="off", GOWORK="off", GOFLAGS="")).decode().strip()
        scratch_parent = "/tmp/opencode" if Path("/tmp/opencode").is_dir() else None
        with tempfile.TemporaryDirectory(prefix="g02-historical-", dir=scratch_parent) as temp:
            scratch = Path(temp)
            (scratch / "home").mkdir()
            env["HOME"] = str(scratch / "home")
            env["TMPDIR"] = str(scratch)
            for name, ref, schema in SOURCES:
                if args.only and args.only != name:
                    continue
                data = generate(name, ref, schema, scratch, env)
                destination = DATA / (name + ".json")
                if args.check_generated:
                    if destination.read_bytes() != data:
                        raise SystemExit(f"non-reproducible fixture: {destination}")
                else:
                    destination.write_bytes(data)
                print(f"{name}: original schema{schema}, {len(data)} bytes", flush=True)
            if not args.only:
                if args.check_generated:
                    assert (DATA / "manifest.json").read_bytes() == manifest
                else:
                    (DATA / "manifest.json").write_bytes(manifest)
                # Export from the real current migrator before invoking original
                # readers, so their refusal tests never relabel a schema marker.
                successor = scratch / "successor"
                successor.mkdir(mode=0o700)
                subprocess.run(["go", "test", "-count=1", "./internal/managed",
                                "-run=^TestG02HistoricalExportForOldReaders$"], cwd=ROOT,
                               env=dict(env, G02_EXPORT_DIR=str(successor)), check=True)
                for name, _, _ in SOURCES:
                    subprocess.run(["go", "test", "-v", "-count=1", "./internal/managed",
                                    "-run=^TestG02HistoricalOldReaderRefusal$"], cwd=scratch / name,
                                   env=dict(env, G02_SUCCESSOR_DIR=str(successor / name)), check=True)
    if not args.only:
        subprocess.run(["go", "test", "-count=1", "./internal/managed", "-run=^TestG02Historical"], cwd=ROOT, check=True)


if __name__ == "__main__":
    main()
