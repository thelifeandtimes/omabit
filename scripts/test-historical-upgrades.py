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
owner-user host paths: schema19 offline and schema20 quiescence-aware online.
RC.12 qualification stays frozen; schema46 is an unreleased development target.
Schema21 has a pinned baseline writer; schemas22/23/24/25/26/27/28/29/30/31/32/33/34/35/36/37/38/39/40/41/42/43/44/45 have frozen unreleased source snapshots.
Unavailable prototype writers are explicit gaps.
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
SCHEMA22_SHA256 = "4f5029ac1469d6cb8b74c99e3fe88eb6932ee2f561460e7a48dd02834b6f8ed5"
SNAPSHOT_SHA256 = {"schema22": SCHEMA22_SHA256, "schema23": "4dcd826f1d3268cfe1eb66abd586cb599266b04aff9b543677cc647db1bfb2dc", "schema24": "69b23e268b73cb87db73b825d43c9692feeced18b30e5c8674d0c2b05f5edf54", "schema25": "ea439418502436ac149c48d01316135834f762b67e629b84c0b410ef89639dd9", "schema26": "a5f0af92d9ab4d504879a732f315e8902d67d0e59312e2b504aaec9c32db5ecd", "schema27": "221f4e7ba757c65339759423c8d5f43232b65cd309755e9f4378ceeeca441904", "schema28": "8386021e437e4d586b06695d69f3877c65d9cb5a7d81105ae04a61c1e3d13be3", "schema29": "8527e8465d3fc2330bc6ba661577b6d3bc776b75da36c301f4557be6550a0331", "schema30": "b8aa477ce568abee0c607b3375686b143fa04e69f62a90c389631b383114dea8", "schema31": "74dcd3b8847474f3d2772af0177e0c85573f5a231fdab958b9a3f9af4829ebf9", "schema32": "3af14bd5559188c9293a23ada1124f246606447935445cd438e6a5fb4305d7eb", "schema33": "ce58e7e73ec6312b34f68ae09d8348a1edcdbd562b61b420d8a09dd408fff30b", "schema34": "0e4893ae8fe8e9fbb486e3a34ccea1a5ec0d6717cfafa2a174331c7a519dceb4"}
SNAPSHOT_SHA256["schema35"] = "ea67857247fccc7b175e6bb4c54fc7110604b0f195850475781fca6b54cc3eba"
SNAPSHOT_SHA256["schema36"] = "17340ebdb296525407a95e004b4779935df2e25f3acec3ae47f861b7bd651c2b"
SNAPSHOT_SHA256["schema37"] = "57ed6461b9cd12f595a20139c3643e76cd79bdfd1d336b3d69c8a86547feec04"
SNAPSHOT_SHA256["schema38"] = "8b363b4c25c8f256b017abbeaedb0bb5b9be5e4d85f52d53778b1df90eea692b"
SNAPSHOT_SHA256["schema40"] = "052de08d5f4aaec592af98396bf457c3b21bc16847e14d59ae1ae6843e58e14f"
SNAPSHOT_SHA256["schema41"] = "eb4bd69f6ac1bb692a83768d84f58ea00afb4c1ba3096a637f215c200b50beb2"
SNAPSHOT_SHA256["schema42"] = "5f743bba5b7df16e4bbaa6ccb3f96582e357c12c313882c154c959a64947ccf3"
SNAPSHOT_SHA256["schema39"] = "e9afaa95aa2d4ad0531baee741925c4fb8e823ce2644118888bcb7633fb3d356"
SNAPSHOT_SHA256["schema43"] = "ff7e9860bc98dbb765e4e2f98535a77a4e8b0898d79327494c1d17b912876222"
SNAPSHOT_SHA256["schema44"] = "80e34af6c6569e9c2bd37a0589cb674261524f6dc1eec176f480afca2aece1a6"
SNAPSHOT_SHA256["schema45"] = "b366caf7df4c488816cb69b24fbeefb082d6b2618283ab76fe2d721a06496da9"
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
    ("rc12", "1529c1335754d15c6efd03d77ce2af7ede3d48f0", 20),
    ("schema21", "b23312876c611e1d083bdcc9468469f7e716d749", 21),
    ("schema22", "unreleased-source-snapshot", 22),
    ("schema23", "unreleased-source-snapshot", 23),
    ("schema24", "unreleased-source-snapshot", 24),
    ("schema25", "unreleased-source-snapshot", 25),
    ("schema26", "unreleased-source-snapshot", 26),
    ("schema27", "unreleased-source-snapshot", 27),
    ("schema28", "unreleased-source-snapshot", 28),
    ("schema29", "unreleased-source-snapshot", 29),
    ("schema30", "unreleased-source-snapshot", 30),
    ("schema31", "unreleased-source-snapshot", 31),
    ("schema32", "unreleased-source-snapshot", 32),
    ("schema33", "unreleased-source-snapshot", 33),
    ("schema34", "unreleased-source-snapshot", 34),
    ("schema35", "unreleased-source-snapshot", 35),
    ("schema36", "unreleased-source-snapshot", 36),
    ("schema37", "unreleased-source-snapshot", 37),
    ("schema38", "unreleased-source-snapshot", 38),
    ("schema39", "unreleased-source-snapshot", 39),
    ("schema40", "unreleased-source-snapshot", 40),
    ("schema41", "unreleased-source-snapshot", 41),
    ("schema42", "unreleased-source-snapshot", 42),
    ("schema43", "unreleased-source-snapshot", 43),
    ("schema44", "unreleased-source-snapshot", 44),
    ("schema45", "unreleased-source-snapshot", 45),
]
# Pin peeled commits independently of mutable refs. RC.2 has an annotated tag;
# its tag object is not the source commit. RC.10–12 also have public release tags,
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
    "1529c1335754d15c6efd03d77ce2af7ede3d48f0",
]
SOURCE_COMMITS = {f"rc{n}": commit for n, commit in enumerate(RELEASE_COMMITS, 1)}
SOURCE_COMMITS["schema21"] = "b23312876c611e1d083bdcc9468469f7e716d749"
SOURCE_COMMITS.update(preview14=SCHEMA14["commit"], preview18="ba4ba805221a6ef3e29829c9bda1bfeff0c0c7c2")
GAPS = {
    "11": "Unpublished prototype: no original writer found in local objects/reflogs/stash or public omabit/tend history. Consolidated commit 34d8ea55 advances schema1 directly to schema13, accepting 11/12 only as inputs. No RC.1-12 release writes schema11.",
    "12": "Unpublished prototype: no original writer found in local objects/reflogs/stash or public omabit/tend history. The schema13 writer accepts schema12 but does not produce it. No RC.1-12 release writes schema12.",
    "17": "Unpublished prototype: no original writer found in local objects/reflogs/stash or public omabit/tend history. Runtime-settings commit ba4ba805 advances schema16 directly to schema18. No RC.1-12 release writes schema17.",
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
        "development_test_target": {"release": "unreleased", "channel": "development", "state_schema": 46,
                                    "evidence": "source-tests-only"},
        "development_test_note": "The reader window and fixture inventory below test the unreleased development target. The RC.12/schema20 qualification target and linked historical evidence remain frozen; these tests do not qualify a new release or artifact.",
        "accepted_schema_window": list(range(11, 46)),
        "supported_release_scope": "database-reader",
        "supported_releases": [
            {"release": f"v0.1.0-rc.{n}", "schema": schemas[f"rc{n}"], "fixture": f"rc{n}.json",
             "source_commit": commit,
             "host_upgrade": {19: "owner-user-schema19-offline", 20: "owner-user-quiescence-online"}.get(schemas[f"rc{n}"], "unsupported")}
            for n, commit in enumerate(RELEASE_COMMITS, 1)
        ],
        "host_upgrade": {"initial_schemas": [19], "deployment": "owner-user-linux-amd64",
                         "procedure": "docs/management-offline-upgrade.md", "evidence": "source-tests-only"},
        "prototype_schemas": [11, 12, 14, 17, 18],
        "preview_fixtures": ["preview14.json", "preview18.json"],
        "baseline_fixtures": ["schema21.json", "schema22.json", "schema23.json", "schema24.json", "schema25.json", "schema26.json", "schema27.json", "schema28.json", "schema29.json", "schema30.json", "schema31.json", "schema32.json", "schema33.json", "schema34.json", "schema35.json", "schema36.json", "schema37.json", "schema38.json", "schema39.json", "schema40.json", "schema41.json", "schema42.json", "schema43.json", "schema44.json", "schema45.json"],
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
    source = scratch / name
    source.mkdir()
    if name in SNAPSHOT_SHA256:
        archive = (DATA / (name + "-source.tar.gz")).read_bytes()
        assert sha(archive) == SNAPSHOT_SHA256[name], "schema22 frozen source changed"
        with tarfile.open(fileobj=io.BytesIO(archive)) as tar:
            tar.extractall(source, filter="data")
        manifest = json.loads((source / "snapshot-manifest.json").read_text())
        assert manifest["schema"] == schema and manifest["unreleased_source_snapshot"] is True
        assert manifest["source_base"] == SOURCE_COMMITS["schema21"]
        for path, digest in manifest["files"].items():
            assert sha((source / path).read_bytes()) == digest, path
        # This was uncommitted source, not a Git release/tree. Preserve that
        # distinction; the tar digest is the reproducible source authority.
        commit, tree = "", ""
    else:
        history = schema14_history(scratch, env) if name == "preview14" else ROOT
        commit = run("git", "rev-parse", ref + "^{commit}", cwd=history).decode().strip()
        assert commit == SOURCE_COMMITS[name], (name, "original source ref changed", commit)
        tree = run("git", "rev-parse", commit + "^{tree}", cwd=history).decode().strip()
        snapshots = [commit]
        if name == "preview14":
            snapshots.append(SCHEMA14["supplement_commit"])
        for snapshot in snapshots:
            archive = run("git", "archive", "--format=tar", snapshot, cwd=history)
            with tarfile.open(fileobj=io.BytesIO(archive)) as tar:
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
                     + ('if id == "running" { req.Config.Loom = 29 }' if name in ("preview18", "rc10") else "")) if schema >= 18 else "",
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
    if name == "schema23":
        # Populate ownership using the frozen original writer, not current types.
        needle = "\tcheck(s.reconcileInterrupted()) // includes an already-interrupted historical receipt"
        assert template.count(needle) == 1
        template = template.replace(needle, 'projectReceipt, _, err := s.registerProject("historical-project-actor", projectRequest(t, "historical-project", "historical", 27080)); check(err); if projectReceipt.ProjectID == "" { t.Fatal("missing project ID") }\n' + needle)
    if name == "schema24":
        # Use the original schema24 store to persist a completed creation journal.
        needle = "\tcheck(s.reconcileInterrupted()) // includes an already-interrupted historical receipt"
        assert template.count(needle) == 1
        template = template.replace(needle, '''projectReceipt, _, err := s.registerProject("historical-project-actor", projectRequest(t, "historical-project", "historical", 27080)); check(err)
        prepared := projectPreparationFixture(t, s, projectReceipt.ProjectID, "pending")
        prepareReceipt, _, err := s.acceptProjectPreparation("historical-project-actor", projectReceipt.ProjectID, projectPrepareRequest{RequestID: "historical-prepare", ExpectedRevision: 1, ReviewDigest: prepared.PlanDigest}); check(err)
        prepared.OperationID = prepareReceipt.ID
        check(s.beginProjectPreparation(projectReceipt.ProjectID, prepareReceipt.ID, prepared))
        for n, step := range []struct{kind,phase,id string}{{"bridge","dispatched",""},{"bridge","created",strings.Repeat("a",64)},{"bridge","observed",strings.Repeat("a",64)},{"anchor","dispatched",""},{"anchor","created",strings.Repeat("b",64)},{"anchor","observed",strings.Repeat("b",64)}} {
            check(s.advanceProjectPreparation(projectReceipt.ProjectID, prepareReceipt.ID, uint64(n), step.kind, step.phase, step.id))
        }
''' + needle)
    if name == "schema25":
        # Preserve the completed original schema25 creation and Start journals.
        needle = "\tcheck(s.reconcileInterrupted()) // includes an already-interrupted historical receipt"
        assert template.count(needle) == 1
        template = template.replace(needle, '''startReceipt, _ := acceptAnchorStartFixture(t, s)
        _, err = s.beginProjectAnchorStart(startReceipt.ProjectID, startReceipt.ID); check(err)
        check(s.advanceProjectAnchorStart(startReceipt.ProjectID, startReceipt.ID, "planned", "dispatched", nil))
        check(s.advanceProjectAnchorStart(startReceipt.ProjectID, startReceipt.ID, "dispatched", "started", nil))
        check(s.advanceProjectAnchorStart(startReceipt.ProjectID, startReceipt.ID, "started", "observed", anchorNamespaceFixture()))
''' + needle)
    if name == "schema26":
        # Use the original writer for completed anchor and both empty parents.
        needle = "\tcheck(s.reconcileInterrupted()) // includes an already-interrupted historical receipt"
        assert template.count(needle) == 1
        template = template.replace(needle, '''startReceipt, _ := acceptAnchorStartFixture(t, s)
        _, err = s.beginProjectAnchorStart(startReceipt.ProjectID, startReceipt.ID); check(err)
        check(s.advanceProjectAnchorStart(startReceipt.ProjectID, startReceipt.ID, "planned", "dispatched", nil))
        check(s.advanceProjectAnchorStart(startReceipt.ProjectID, startReceipt.ID, "dispatched", "started", nil))
        check(s.advanceProjectAnchorStart(startReceipt.ProjectID, startReceipt.ID, "started", "observed", anchorNamespaceFixture()))
        project, err := loadProject(s.db,startReceipt.ProjectID); check(err)
        planDigest, err := project.Plan.Digest(); check(err)
        for i, member := range project.Members {
            storageOp, _, err := s.acceptProjectMemberStorage("historical-member-actor", project.ID, member.InstanceID, projectMemberStorageRequest{RequestID:"storage-"+member.InstanceID,ExpectedRevision:1,ReviewDigest:planDigest}); check(err)
            _, err = s.beginProjectMemberStorage(member.InstanceID,storageOp.ID); check(err)
            check(s.advanceProjectMemberStorage(member.InstanceID,storageOp.ID,"planned","dispatched",pierstore.DirectoryIdentity{}))
            identity := memberMountIdentity; identity.Inode += uint64(i)
            check(s.advanceProjectMemberStorage(member.InstanceID,storageOp.ID,"dispatched","created",identity))
            check(s.advanceProjectMemberStorage(member.InstanceID,storageOp.ID,"created","observed",identity))
        }
''' + needle)
    if name == "schema27":
        # Original writer retains complete serial member launches and their parents.
        needle = "\tcheck(s.reconcileInterrupted()) // includes an already-interrupted historical receipt"
        assert template.count(needle) == 1
        template = template.replace(needle, """project, launchReq := preparedMemberLaunchProject(t, s)
        for i, member := range project.Members {
            launchReq.RequestID = "launch-"+member.InstanceID
            op, _, err := s.acceptProjectMemberLaunch("historical-launch-actor", project.ID, member.InstanceID, launchReq); check(err)
            initial := memberLaunchSelection(t,s,op)
            _, err = s.beginProjectMemberLaunch(member.InstanceID,op.ID,initial); check(err)
            for n := 0; n < len(memberLaunchPhases)-1; n++ { advanceMemberLaunchFixture(t,s,op,n,fmt.Sprintf("%064x",i+100)) }
        }
""" + needle)
    if name == "schema28":
        needle = "\tcheck(s.reconcileInterrupted()) // includes an already-interrupted historical receipt"
        assert template.count(needle) == 1
        template = template.replace(needle, """project, readyReq := launchedReadinessProject(t,s)
        for _,member:=range project.Members {
            readyReq.RequestID="observe-"+member.InstanceID
            op,_,err:=s.acceptProjectMemberReadiness("historical-ready-actor",project.ID,member.InstanceID,readyReq);check(err)
            _,err=s.beginProjectMemberReadiness(op.ID);check(err)
            check(s.completeProjectMemberReadiness(op.ID,memberReadinessProof(t,s,project.ID,member.InstanceID)))
        }
""" + needle)
    if name == "schema29":
        needle = "\tcheck(s.reconcileInterrupted()) // includes an already-interrupted historical receipt"
        assert template.count(needle) == 1
        template = template.replace(needle, """{
        project,stopReq,_:=readyStopProject(t,s)
        instance:=project.Members[0].InstanceID
        op,_,err:=s.acceptProjectMemberStop("historical-stop-actor",project.ID,instance,stopReq);check(err)
        _,err=s.beginProjectMemberStop(op.ID);check(err)
        before,_:=storeStopObservations(t,s,project.ID,instance)
        advanceStopFixture(t,s,project,op.ID,"stop-dispatched",&before)
        }
""" + needle)
    if name == "schema30":
        needle = "\tcheck(s.reconcileInterrupted()) // includes an already-interrupted historical receipt"
        assert template.count(needle) == 1
        template = template.replace(needle, """{
        project,exitReq:=interruptedStopProject(t,s)
        instance:=project.Members[0].InstanceID
        op,_,err:=s.acceptProjectMemberStopExit("historical-exit-actor",project.ID,instance,exitReq);check(err)
        _,err=s.beginProjectMemberStopExit(op.ID);check(err)
        check(s.completeProjectMemberStopExit(op.ID,memberStopExitProof(t,s,op.ID)))
        }
""" + needle)
    if name == "schema31":
        needle = "\tcheck(s.reconcileInterrupted()) // includes an already-interrupted historical receipt"
        assert template.count(needle) == 1
        template = template.replace(needle, """{
        project,retireReq:=readyRetirementProject(t,s)
        instance:=project.Members[0].InstanceID
        op,_,err:=s.acceptProjectMemberRetirement("historical-retirement-actor",project.ID,instance,retireReq);check(err)
        _,err=s.beginProjectMemberRetirement(op.ID);check(err)
        before,after:=retirementStoreObservations(t,s,op.ID)
        advanceRetirementFixture(t,s,project,op.ID,"remove-dispatched",&before,nil)
        advanceRetirementFixture(t,s,project,op.ID,"removed",nil,nil)
        advanceRetirementFixture(t,s,project,op.ID,"absent",nil,&after)
        check(s.finishProjectMemberRetirement(op.ID))
        }
""" + needle)
    if name == "schema32":
        needle = "\tcheck(s.reconcileInterrupted()) // includes an already-interrupted historical receipt"
        assert template.count(needle) == 1
        template = template.replace(needle, """{
        project,resumeReq:=readyResumeProject(t,s)
        instance:=project.Members[0].InstanceID
        op,_,err:=s.acceptProjectMemberResume("historical-resume-actor",project.ID,instance,resumeReq);check(err)
        _,err=s.beginProjectMemberResume(op.ID,initialResumeFixture(t,s,op.ID));check(err)
        for _,phase:=range []string{"create-dispatched","created","start-dispatched","started","running"} {
            j,err:=loadProjectMemberResume(s.db,op.ID);check(err)
            check(s.advanceProjectMemberResume(op.ID,j.Resume.Phase,nextResumeFixture(t,s,op.ID,phase)))
        }
        check(s.finishProjectMemberResume(op.ID))
        }
""" + needle)
    if name == "schema33":
        needle = "\tcheck(s.reconcileInterrupted()) // includes an already-interrupted historical receipt"
        assert template.count(needle) == 1
        template = template.replace(needle, """{
        project,readyReq:=resumedReadinessProject(t,s)
        instance:=project.Members[0].InstanceID
        op,_,err:=s.acceptProjectMemberResumedReadiness("historical-resumed-ready-actor",project.ID,instance,readyReq);check(err)
        _,err=s.beginProjectMemberResumedReadiness(op.ID);check(err)
        check(s.completeProjectMemberResumedReadiness(op.ID,resumedReadinessProof(t,s,project.ID,instance,readyReq.ResumeOperationID)))
        }
""" + needle)
    if name == "schema34":
        needle = "\tcheck(s.reconcileInterrupted()) // includes an already-interrupted historical receipt"
        assert template.count(needle) == 1
        template = template.replace(needle, """{
        project,stopReq,_:=readyCurrentStopProject(t,s)
        instance:=project.Members[0].InstanceID
        op,_,err:=s.acceptProjectMemberStop("historical-current-stop-actor",project.ID,instance,stopReq);check(err)
        _,err=s.beginProjectMemberStop(op.ID);check(err)
        before,after:=currentStopObservations(t,s,project.ID,instance)
        advanceStopFixture(t,s,project,op.ID,"stop-dispatched",&before)
        advanceStopFixture(t,s,project,op.ID,"stop-signalled",nil)
        advanceStopFixture(t,s,project,op.ID,"stopped",&after)
        check(s.finishProjectMemberStop(op.ID))
        }
""" + needle)
    if name == "schema35":
        needle = "\tcheck(s.reconcileInterrupted()) // includes an already-interrupted historical receipt"
        assert template.count(needle) == 1
        template = template.replace(needle, """{
        project,stopReq,_:=readyCurrentStopProject(t,s)
        req:=retireCurrentFixture(t,s,project,stopReq,2)
        instance:=project.Members[0].InstanceID
        op,_,err:=s.acceptProjectMemberResume("historical-second-resume-actor",project.ID,instance,req);check(err)
        resumeThroughFixture(t,s,op.ID,"complete")
        digest,err:=project.Plan.Digest();check(err)
        observeGenerationFixture(t,s,project,op.ID,digest,3)
        }
""" + needle)
    if name == "schema36":
        needle = "\tcheck(s.reconcileInterrupted()) // includes an already-interrupted historical receipt"
        assert template.count(needle) == 1
        template = template.replace(needle, """{
        project,stopReq,_:=readyCurrentStopProject(t,s)
        req:=retireCurrentFixture(t,s,project,stopReq,2)
        instance:=project.Members[0].InstanceID
        op,_,err:=s.acceptProjectMemberResume("historical-second-resume-actor",project.ID,instance,req);check(err)
        resumeThroughFixture(t,s,op.ID,"complete")
        digest,err:=project.Plan.Digest();check(err)
        observeGenerationFixture(t,s,project,op.ID,digest,3)
        for _,member:=range project.Members {
            var ready string
            check(s.db.QueryRow("SELECT operation_id FROM project_member_readiness WHERE project_id=? AND instance_id=?",project.ID,member.InstanceID).Scan(&ready))
            src,err:=projectContextReadinessSource(s.db,project.ID,member.InstanceID,ready,nil);check(err)
            req:=projectMemberContextReadinessRequest{OriginalReadinessOperationID:ready,CreatorOperationID:src.Context.Member.OperationID,RequestID:"current-"+member.InstanceID,ExpectedRevision:project.Revision,ReviewDigest:digest}
            op,_,err:=s.acceptProjectMemberContextReadiness("historical-current-ready-actor",project.ID,member.InstanceID,req);check(err)
            _,err=s.beginProjectMemberContextReadiness(op.ID);check(err)
            check(s.completeProjectMemberContextReadiness(op.ID,contextReadinessProof(t,s,op.ID)))
        }
        }
""" + needle)
    if name == "schema37":
        needle = "\tcheck(s.reconcileInterrupted()) // includes an already-interrupted historical receipt"
        assert template.count(needle) == 1
        template = template.replace(needle, """{
        project,instance,stopReq:=readyContextStopProject(t,s,0)
        stop,_,err:=s.acceptProjectMemberStop("historical-context-actor",project.ID,instance,stopReq);check(err)
        _,err=s.beginProjectMemberStop(stop.ID);check(err)
        before,_:=contextStopObservations(t,s,stopReq.ReadinessOperationID)
        advanceStopFixture(t,s,project,stop.ID,"stop-dispatched",&before)
        check(s.interruptMemberStop(stop.ID))
        retained,err:=loadProjectMemberStop(s.db,stop.ID);check(err)
        resumeReq:=retireStoppedGenerationFixture(t,s,retained)
        resumed,_,err:=s.acceptProjectMemberResume("historical-context-actor",project.ID,instance,resumeReq);check(err)
        resumeThroughFixture(t,s,resumed.ID,"complete")
        digest,err:=project.Plan.Digest();check(err)
        observeGenerationFixture(t,s,project,resumed.ID,digest,3)
        var original string
        check(s.db.QueryRow("SELECT operation_id FROM project_member_readiness WHERE project_id=? AND instance_id=?",project.ID,instance).Scan(&original))
        src,err:=projectContextReadinessSource(s.db,project.ID,instance,original,nil);check(err)
        readyReq:=projectMemberContextReadinessRequest{OriginalReadinessOperationID:original,CreatorOperationID:src.Context.Member.OperationID,RequestID:"historical-current-again",ExpectedRevision:project.Revision,ReviewDigest:digest}
        ready,_,err:=s.acceptProjectMemberContextReadiness("historical-context-actor",project.ID,instance,readyReq);check(err)
        _,err=s.beginProjectMemberContextReadiness(ready.ID);check(err)
        check(s.completeProjectMemberContextReadiness(ready.ID,contextReadinessProof(t,s,ready.ID)))
        pendingReq:=projectMemberStopRequest{RequestID:"historical-final-context-stop",ReadinessOperationID:ready.ID,ExpectedRevision:project.Revision,ReviewDigest:digest}
        pending,_,err:=s.acceptProjectMemberStop("historical-context-actor",project.ID,instance,pendingReq);check(err)
        _,err=s.beginProjectMemberStop(pending.ID);check(err)
        before,after:=contextStopObservations(t,s,ready.ID)
        advanceStopFixture(t,s,project,pending.ID,"stop-dispatched",&before)
        advanceStopFixture(t,s,project,pending.ID,"stop-signalled",nil)
        advanceStopFixture(t,s,project,pending.ID,"stopped",&after)
        check(s.finishProjectMemberStop(pending.ID))
        }
""" + needle)
    if name in ("schema38", "schema39", "schema40", "schema41", "schema42", "schema43", "schema44", "schema45"):
        needle = "\tcheck(s.reconcileInterrupted()) // includes an already-interrupted historical receipt"
        assert template.count(needle) == 1
        template = template.replace(needle, """{
        project,req:=recoveryStoreProject(t,s)
        first,_,err:=s.acceptProjectAnchorRecovery("historical-recovery-actor",project.ID,req);check(err)
        completed,err:=loadProjectAnchorRecovery(s.db,first.ID);check(err)
        completed=completeRecoveryStoreFixture(t,s,completed)
        if completed.Journal.Phase!="complete" {t.Fatal("historical recovery incomplete")}
        history,_,_,err:=projectRecoveryHistory(s.db,project.ID,true);check(err)
        req.RequestID="historical-second-recovery";req.HistoryDigest=recoveryHistoryDigest(history)
        second,_,err:=s.acceptProjectAnchorRecovery("historical-recovery-actor",project.ID,req);check(err)
        pending,err:=loadProjectAnchorRecovery(s.db,second.ID);check(err)
        for _,phase:=range []string{"reviewing","reviewed","draining","stop-dispatched"} {pending=advanceRecoveryStoreFixture(t,s,pending,phase)}
        check(s.interruptAnchorRecovery(second.ID,false))
        check(validateProjectAnchorRecoveries(s.db))
        }
""" + needle)
    if name in ("schema39", "schema40", "schema41", "schema42", "schema43", "schema44", "schema45"):
        template = template.replace('"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"', '"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"\n"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"')
        needle = "        check(s.interruptAnchorRecovery(second.ID,false))"
        assert template.count(needle) == 1
        template = template.replace(needle, needle + """
        observationReq:=observationRequest(project,pending,"historical-observation-interrupted")
        interrupted,_,err:=s.acceptProjectRecoveryObservation("historical-observer",project.ID,observationReq);check(err)
        _,err=s.beginProjectRecoveryObservation(interrupted.ID);check(err)
        check(s.interruptRecoveryObservation(interrupted.ID))
        observationReq.RequestID="historical-observation-complete"
        observed,_,err:=s.acceptProjectRecoveryObservation("historical-observer",project.ID,observationReq);check(err)
        _,err=s.beginProjectRecoveryObservation(observed.ID);check(err)
        after:=recoveryTestClone(t,recoveryStoreDrainObservation(pending))
        after.CapturedAt=time.Now().UTC();after.Runtime.ObservedAt=after.CapturedAt
        instance:=pending.Journal.Drain.Members[0].InstanceID
        for k:=range after.Runtime.Members {if after.Runtime.Members[k].InstanceID==instance {after.Runtime.Members[k].Running=false;after.Runtime.Members[k].Namespace=projectnetwork.NamespaceIdentity{}}}
        for k:=range after.Processes {if after.Processes[k].InstanceID==instance {after.Processes[k].Process.FinishedAt=time.Now().UTC()}}
        drained,err:=pending.Journal.Drain.Advance(pending.Source,0,pending.Journal.Drain.Sequence,"exited",&after);check(err)
        next,err:=pending.Journal.Advance(pending.Source,pending.Journal.Sequence,"draining",projectnetwork.AnchorRecoveryJournal{Drain:&drained});check(err)
        check(s.completeProjectRecoveryObservation(observed.ID,pending.Journal.Sequence,next))
        check(validateProjectRecoveryObservations(s.db))
""")
    if name in ("schema40", "schema41", "schema42", "schema43", "schema44", "schema45"):
        needle = "        check(validateProjectRecoveryObservations(s.db))"
        assert template.count(needle) == 1
        template = template.replace(needle, needle + """
        pending,err=loadProjectAnchorRecovery(s.db,second.ID);check(err)
        continuationReq:=observationRequest(project,pending,"historical-continuation-interrupted")
        continuation,_,err:=s.acceptProjectRecoveryContinuation("historical-continuation-actor",project.ID,continuationReq);check(err)
        _,err=s.beginProjectRecoveryContinuation(continuation.ID);check(err)
        check(s.interruptRecoveryContinuation(continuation.ID))
        pending,err=loadProjectAnchorRecovery(s.db,second.ID);check(err)
        continuationReq=observationRequest(project,pending,"historical-continuation-active")
        active,_,err:=s.acceptProjectRecoveryContinuation("historical-second-continuation-actor",project.ID,continuationReq);check(err)
        _,err=s.beginProjectRecoveryContinuation(active.ID);check(err)
        check(validateProjectRecoveryContinuations(s.db))
""")
    if name in ("schema41", "schema42", "schema43", "schema44", "schema45"):
        needle = "\tcheck(s.reconcileInterrupted()) // includes an already-interrupted historical receipt"
        assert template.count(needle) == 1
        template = template.replace(needle, """{
        for n,prefix := range []string{"historical-private-a","historical-private-b"} {
            req:=privateProjectRequest(t,"historical-register-"+prefix,prefix,uint16(30080+n*10))
            actor:="historical-private-actor-"+prefix
            op,hit,err:=s.registerPrivateProject(actor,req);check(err)
            if hit || op.ProjectID=="" {t.Fatal("missing original private reservation")}
            req.ReplayOnly=true
            replay,hit,err:=s.registerPrivateProject(actor,req);check(err)
            if !hit || replay.ID!=op.ID {t.Fatal("original private replay changed")}
        }
        if _,err:=projectReservations(s.db);err!=nil {t.Fatal(err)}
        }
""" + needle)
    if name in ("schema42", "schema43", "schema44", "schema45"):
        template=template.replace('"bytes"','"bytes"\n    "context"')
        needle='            if !hit || replay.ID!=op.ID {t.Fatal("original private replay changed")}'
        assert template.count(needle)==1
        template=template.replace(needle,needle+"""
            p,err:=loadPrivateProject(s.db,op.ProjectID);check(err)
            reviewed,err:=p.Plan.Digest();check(err)
            workspaceReq:=privateWorkspaceRequest{RequestID:"historical-workspace-"+prefix,ExpectedRevision:1,ReviewDigest:reviewed}
            prepared,hit,err:=s.acceptPrivateWorkspace(actor+"-workspace",op.ProjectID,workspaceReq,privateWorkspaceParent());check(err)
            if hit {t.Fatal("workspace unexpectedly replayed")}
            if n==0 {
                check(s.runPrivateWorkspace(context.Background(),op.ProjectID,prepared.ID,&privateWorkspaceStoreDriver{t:t,s:s}))
            } else {
                w,err:=s.beginPrivateWorkspace(op.ProjectID,prepared.ID);check(err)
                for _,phase:=range []string{"root-dispatched","root-created","layout-dispatched"} {
                    root:=w.Root
                    if phase=="root-created" {root=privateWorkspaceParent();root.Inode=20}
                    next,err:=w.Advance(p.Plan,w.Phase,phase,root,nil);check(err)
                    check(s.advancePrivateWorkspace(op.ProjectID,prepared.ID,w.Phase,next));w=next
                }
            }
""")
    if name in ("schema43", "schema44", "schema45"):
        template=template.replace('"context"','"context"\n    "errors"\n    "git.thechristophgroup.org/thelifeandtimes/omabit/internal/privatechain"')
        needle = "\tcheck(s.reconcileInterrupted()) // includes an already-interrupted historical receipt"
        assert template.count(needle)==1
        template=template.replace(needle,"""{
        for n,prefix := range []string{"historical-private-a","historical-private-b"} {
            actor:="historical-private-actor-"+prefix
            registerReq:=privateProjectRequest(t,"historical-register-"+prefix,prefix,uint16(30080+n*10));registerReq.ReplayOnly=true
            registered,hit,err:=s.registerPrivateProject(actor,registerReq);check(err)
            if !hit {t.Fatal("private project missing")}
            p,err:=loadPrivateProject(s.db,registered.ProjectID);check(err)
            w,err:=loadPrivateWorkspace(s.db,registered.ProjectID);check(err)
            if n==1 {
                check(s.interruptPrivateWorkspace(registered.ProjectID,w.OperationID))
                observe:=privateWorkspaceObservationRequest{RequestID:"historical-workspace-observation-interrupted",ExpectedRevision:1,ReviewDigest:w.PlanDigest,SourceOperationID:w.OperationID,SourceDigest:privateWorkspaceSourceDigest(w)}
                first,_,err:=s.acceptPrivateWorkspaceObservation(actor+"-observer",registered.ProjectID,observe);check(err)
                _,err=s.beginPrivateWorkspaceObservation(first.ID);check(err)
                check(s.interruptPrivateWorkspaceObservation(first.ID))
                observe.RequestID="historical-workspace-observation-complete"
                recovered,_,err:=s.acceptPrivateWorkspaceObservation(actor+"-observer",registered.ProjectID,observe);check(err)
                check(s.runPrivateWorkspaceObservation(context.Background(),recovered.ID,&workspaceObservationOnly{t:t,s:s,operation:recovered.ID}))
            }
            ready,err:=readyPrivateWorkspace(s.db,registered.ProjectID);check(err)
            inputs:=privatechain.RuntimeInputInspection{Format:privatechain.RuntimeInputFormat,ManifestSHA256:strings.Repeat("1",64),FoundationDigest:p.Binding.FoundationDigest,Files:12352,Bytes:834035190}
            prototype,err:=privatechain.NewGateway(p.Plan,ready,"prototype-gateway","private-engine",inputs);check(err)
            req:=privateGatewayRequest{RequestID:"historical-gateway-"+prefix,ExpectedRevision:1,ReviewDigest:prototype.Selection.PlanDigest,WorkspaceOperationID:w.OperationID,WorkspaceDigest:prototype.Selection.WorkspaceDigest,InputManifestSHA256:inputs.ManifestSHA256}
            initialized,hit,err:=s.acceptPrivateGateway(actor+"-gateway",registered.ProjectID,req,"private-engine",inputs);check(err)
            if hit {t.Fatal("gateway unexpectedly replayed")}
            driver:=&privateGatewayStoreDriver{t:t,s:s}
            if n==0 {
                driver.fail="observe-gateway"
                if err=s.runPrivateGateway(context.Background(),registered.ProjectID,initialized.ID,driver);!errors.Is(err,privatechain.ErrGatewayUnresolved){t.Fatal("expected interrupted original initialization",err)}
                original,err:=loadPrivateGateway(s.db,registered.ProjectID);check(err)
                observe:=privateGatewayObservationRequest{RequestID:"historical-gateway-observation-interrupted",ExpectedRevision:1,ReviewDigest:original.Selection.PlanDigest,SourceOperationID:initialized.ID,SourceDigest:privateGatewaySourceDigest(original)}
                first,_,err:=s.acceptPrivateGatewayObservation(actor+"-observer",registered.ProjectID,observe);check(err)
                _,err=s.beginPrivateGatewayObservation(first.ID);check(err)
                check(s.interruptPrivateGatewayObservation(first.ID))
                observe.RequestID="historical-gateway-observation-complete"
                recovered,_,err:=s.acceptPrivateGatewayObservation(actor+"-observer",registered.ProjectID,observe);check(err)
                check(s.runPrivateGatewayObservation(context.Background(),recovered.ID,&gatewayObservationOnly{t:t,s:s,operation:recovered.ID}))
            } else {
                check(s.runPrivateGateway(context.Background(),registered.ProjectID,initialized.ID,&g02PrivateSecondGateway{driver}))
            }
            actual,evidence,err:=readyPrivateGateway(s.db,registered.ProjectID);check(err)
            if actual.Phase!="ready" || evidence.Status!="succeeded" {t.Fatal("gateway readiness missing")}
        }
        check(privateGatewayState(s.db))
        }
"""+needle)
        template+="""
// Synthetic second-project resources must remain distinct from the first.
type g02PrivateSecondGateway struct { *privateGatewayStoreDriver }
func (d *g02PrivateSecondGateway) CreateNetwork(_ context.Context,g privatechain.Gateway)(string,error){return strings.Repeat("4",64),d.check("create-network",g)}
func (d *g02PrivateSecondGateway) CreateGateway(_ context.Context,g privatechain.Gateway)(string,error){return strings.Repeat("5",64),d.check("create-gateway",g)}
"""
    if name in ("schema44", "schema45"):
        needle = "\tcheck(s.reconcileInterrupted()) // includes an already-interrupted historical receipt"
        assert template.count(needle)==1
        template=template.replace(needle, "populatePrivateMember44(t, s)\n"+needle)
    if name == "schema45":
        needle = "\tcheck(s.reconcileInterrupted()) // includes an already-interrupted historical receipt"
        assert template.count(needle)==1
        template=template.replace(needle, "populatePrivateBirth45(t, s)\n"+needle)
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
    if name in SNAPSHOT_SHA256:
        fixture.update(source_snapshot_sha256=SNAPSHOT_SHA256[name], source_base=SOURCE_COMMITS["schema21"], unreleased_source_snapshot=True)
    result = document(fixture)
    # Schema28 retains two complete identity/storage/runtime proofs and mirrors.
    limit = 8 * 1024 * 1024 if name in ("schema39", "schema40", "schema41", "schema42", "schema43", "schema44", "schema45") else 4 * 1024 * 1024 if name == "schema38" else 1536 * 1024 if name == "schema37" else 1024 * 1024 if name == "schema36" else 768 * 1024 if name == "schema35" else 384 * 1024 if name == "schema34" else 320 * 1024 if name == "schema33" else 320 * 1024 if name == "schema32" else 224 * 1024 if name == "schema31" else 192 * 1024 if name == "schema30" else 160 * 1024 if name in ("schema28", "schema29") else 96 * 1024
    assert len(result) <= limit, (name, len(result))
    return result


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    flags = parser.add_mutually_exclusive_group()
    flags.add_argument("--generate", action="store_true")
    flags.add_argument("--check-generated", action="store_true")
    parser.add_argument("--only", choices=[s[0] for s in SOURCES], help="reproduce one writer and check its refusal of actual successor state")
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
            successor_schema = json.loads(manifest)["development_test_target"]["state_schema"]
            for name, _, schema in SOURCES:
                if args.only and args.only != name:
                    continue
                assert schema < successor_schema, (name, "not an old reader")
                output = run("go", "test", "-v", "-count=1", "./internal/managed",
                             "-run=^TestG02HistoricalOldReaderRefusal$", cwd=scratch / name,
                             env=dict(env, G02_SUCCESSOR_DIR=str(successor / name))).decode()
                print(output, end="", flush=True)
                # A skipped/missing test is not downgrade-refusal evidence.
                assert f"original schema{schema} refused real schema{successor_schema} state without modification" in output, (name, "missing original-reader refusal")
    if not args.only:
        subprocess.run(["go", "test", "-count=1", "./internal/managed", "-run=^TestG02Historical"], cwd=ROOT, check=True)


if __name__ == "__main__":
    main()
