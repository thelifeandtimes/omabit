#!/usr/bin/env python3
"""Original 0.4.0/schema59 writer, successor preservation and old-reader refusal.

Private synthetic state only. No installed service, Docker engine or ship is used.
This does not replace an installed artifact upgrade or live runtime qualification.
"""
import argparse
import hashlib
import importlib.util
import io
import json
import os
from pathlib import Path
import shutil
import subprocess
import tarfile

ROOT = Path(__file__).resolve().parents[1]
OLD = "a9623329a538ebf83d63365053d2d1011ade091e"


def module(name, path):
    spec = importlib.util.spec_from_file_location(name, path)
    loaded = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(loaded)
    return loaded


previous = module("resource_upgrade", ROOT / "scripts/test-container-resources-upgrade.py")
OLD_TEST = previous.OLD_TEST.replace("StateSchemaVersion!=58", "StateSchemaVersion!=59").replace("original schema58", "original schema59")
OLD_TEST += r'''
func TestStopOriginal59Project(t *testing.T) {
 if StateSchemaVersion!=59 {t.Fatal("not original59")}
 s,err:=testOpenStore("/tmp/schema-state");g02Check(t,err);defer s.close()
 p,_,_:=readyStopProject(t,s)
 selected,err:=s.projectFleetSelection(p.ID,"");g02Check(t,err)
 state,err:=runtimequalify.SeedProjectExpansion(selected);g02Check(t,err)
 state.Members[0].Initialized=memberReadinessProof(t,s,p.ID,state.Members[0].Member.InstanceID).Storage
 state.Members[0].Phase="stop-dispatched"
 now:=time.Now().UTC()
 op:=protocol.Operation{ID:"op-original59-stop",ProjectID:p.ID,InstanceID:state.Members[0].Member.InstanceID,Actor:"actor",RequestID:"original59-stop",Kind:"stop-project-ship",Status:"interrupted",ConfigRevision:p.Revision,CreatedAt:now,FinishedAt:&now,Message:"Original59 interrupted stop diagnostic retained"}
 body,err:=json.Marshal(state);g02Check(t,err)
 tx,err:=s.db.Begin();g02Check(t,err);defer tx.Rollback()
 _,err=tx.Exec("INSERT INTO project_expansions(project_id,body) VALUES(?,?)",p.ID,body);g02Check(t,err)
 digest,err:=p.Plan.Digest();g02Check(t,err)
 input:=protocol.ProjectActionRequest{Kind:"member-stop",InstanceID:op.InstanceID,ContainerID:state.Members[0].Creator.Container.ID,ExpectedRevision:p.Revision,ReviewDigest:digest,RequestID:op.RequestID,ConfirmAction:true}
 fingerprint,err:=json.Marshal(struct{Project string;Request protocol.ProjectActionRequest}{p.ID,input});g02Check(t,err)
 g02Check(t,insertOperation(tx,op,string(fingerprint)));g02Check(t,tx.Commit())
}
'''
OLD_TEST = OLD_TEST.replace('import(', 'import("encoding/json";"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol";"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify";', 1)

NEW_TEST = r'''package managed
import("reflect";"testing";"time";"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol")
func TestStopSchema60Migration(t *testing.T) {
 if StateSchemaVersion<60 || StateSchemaVersion>62 {t.Fatal("unsupported successor schema")}
 for n:=0;n<3;n++{s,err:=testOpenStore("/tmp/schema-state");g02Check(t,err);g02Check(t,s.close())}
}
func TestStopSchema60Resolution(t *testing.T) {
 s,err:=testOpenStore("/tmp/schema-state");g02Check(t,err)
 original,err:=loadOperation(s.db,"op-original59-stop");g02Check(t,err)
 selected,err:=s.projectStopSelection(s.db,"actor",original.ProjectID,original.InstanceID,original.ID);g02Check(t,err)
 m,a:=selected.member,selected.state.Anchor
 ns:=a.Namespace
 o:=protocol.ProjectStopObservation{ContainerID:m.Creator.Container.ID,ContractDigest:m.Creator.Container.ContractDigest,EngineID:a.Prepared.EngineID,AnchorID:a.Prepared.Anchor.ID,BridgeID:a.Prepared.Bridge.ID,BootID:ns.BootID,NamespaceDevice:ns.Device,NamespaceInode:ns.Inode,Storage:m.Initialized,StartedAt:original.CreatedAt.Add(-time.Minute),FinishedAt:*original.FinishedAt,ObservedAt:time.Now().UTC()}
 review,err:=selected.observedReview(o);g02Check(t,err)
 if review.EvidenceBasis!="legacy-admission-window"{t.Fatal("invented old process binding")}
 req:=protocol.ProjectStopReconcileRequest{RequestID:"schema60-reconcile",Review:review,ConfirmAction:true}
 result,err:=s.commitProjectStopResolution("actor",original.ProjectID,req,review);g02Check(t,err)
 retained,err:=loadOperation(s.db,original.ID);g02Check(t,err);if !reflect.DeepEqual(original,retained){t.Fatal("original receipt changed")}
 g02Check(t,s.close());s,err=testOpenStore("/tmp/schema-state");g02Check(t,err);defer s.close()
 replay,hit,err:=s.replayProjectStopResolution("actor",original.ProjectID,req);g02Check(t,err)
 if !hit||!reflect.DeepEqual(result,replay){t.Fatal("reopen changed resolution")}
}
'''

CAPSULE = r'''
import json,shutil,sqlite3,subprocess
from pathlib import Path
def run(binary,name):
 print(name,flush=True)
 subprocess.run([binary,'-test.run','^'+name+'$','-test.timeout','90s'],check=True,cwd='/tmp')
def rows():
 with sqlite3.connect('file:/tmp/schema-state/runner.sqlite?mode=ro',uri=True) as db:
  tables=[r[0] for r in db.execute("select name from sqlite_master where type='table' and name not like 'sqlite_%' order by name")]
  return {t:sorted(json.dumps([{'bytes':v.hex()} if isinstance(v,bytes) else v for v in r],sort_keys=True) for r in db.execute('select * from "'+t+'"')) for t in tables}
run('/writer59','TestDeskOriginalWriter')
run('/writer59','TestResourceOriginalRoutes')
run('/writer59','TestStopOriginal59Project')
with sqlite3.connect('file:/tmp/schema-state/runner.sqlite?mode=ro',uri=True) as db:
 assert db.execute('pragma user_version').fetchone()[0]==59
 Path('/tmp/original-schema59.sql').write_text('\n'.join(db.iterdump())+'\nPRAGMA user_version=59;\n')
before=rows()
run('/current','TestStopSchema60Migration')
after=rows()
with sqlite3.connect('file:/tmp/schema-state/runner.sqlite?mode=ro',uri=True) as db:
 target_schema=db.execute('pragma user_version').fetchone()[0]
assert target_schema in (60,61,62)
expected={'project_stop_reconciliations'}
if target_schema>=61:
 expected.update({'grubbery_app_clock','grubbery_app_operations','grubbery_app_events','grubbery_app_leases'})
assert set(after)-set(before)==expected
assert all(before[t]==after[t] for t in before),'migration rewrote original state'
run('/current','TestStopSchema60Resolution')
run('/writer59','TestDeskOlderReaderRefusesUnchanged')
Path('/tmp/verification.json').write_text(json.dumps({'passed':True,'original_schema':59,'target_schema':target_schema,'original_tables_preserved':sorted(before),'migration_reopens':3,'resolution_reopen_and_exact_replay':True,'actual_old_reader_refusal':True,'scope':'synthetic writer/migration/interface evidence; no installed or live-runtime qualification'},indent=2)+'\n')
'''


def sha(path):
    with path.open("rb") as stream:
        return hashlib.file_digest(stream, "sha256").hexdigest()


def git(*args):
    return subprocess.check_output(["git", *args], cwd=ROOT)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--budget-gib", type=int, choices=(40,48), default=40,
                        help="48 requires a separately authorized temporary qualification budget")
    args = parser.parse_args()
    storage = module("stop_upgrade_storage", ROOT / "scripts/qualification/app-development/storage.py")
    admission = storage.admit(192 * 1024**2, budget_bytes=args.budget_gib * storage.GIB)
    out = args.output.resolve()
    out.mkdir(mode=0o700)
    scratch = out / "scratch"
    scratch.mkdir(mode=0o700)
    head = git("rev-parse", "HEAD").decode().strip()
    report = {"original_source": OLD, "current_base": head, "overlays": {}, "binaries": {}, "storage_admission": admission}
    for name, commit, helper in [("writer59", OLD, OLD_TEST), ("current", head, NEW_TEST)]:
        source = out / (name + "-source")
        source.mkdir()
        with tarfile.open(fileobj=io.BytesIO(git("archive", commit))) as archive:
            members = [m for m in archive.getmembers() if not (m.name.startswith("internal/managed/testdata/historical-upgrades/") and m.name.endswith("-source.tar.gz"))]
            archive.extractall(source, members=members, filter="data")
        if name == "current":
            paths = set(git("diff", "HEAD", "--name-only", "--diff-filter=ACMR", "-z").decode().split("\0") + git("ls-files", "--others", "--exclude-standard", "-z").decode().split("\0")) - {""}
            for path in sorted(paths):
                origin = ROOT / path
                if origin.is_symlink() or not origin.is_file():
                    raise ValueError("unsafe source overlay: " + path)
                target = source / path
                target.parent.mkdir(parents=True, exist_ok=True)
                shutil.copyfile(origin, target)
                report["overlays"][path] = sha(target)
        helper_path = source / "internal/managed/zz_stop_schema_upgrade_test.go"
        helper_path.write_text(helper)
        report[name + "_helper_sha256"] = sha(helper_path)
        binary = out / name
        subprocess.run(["go", "test", "-c", "-p", "2", "-ldflags=-s -w", "-o", str(binary), "./internal/managed"], cwd=source, check=True)
        report["binaries"][name] = sha(binary)
    capsule = out / "capsule.py"
    capsule.write_text(CAPSULE)
    command = ["bwrap", "--unshare-all", "--die-with-parent", "--clearenv", "--ro-bind", "/usr", "/usr", "--symlink", "usr/bin", "/bin", "--symlink", "usr/lib", "/lib", "--symlink", "usr/lib", "/lib64", "--proc", "/proc", "--dev", "/dev", "--dir", "/etc", "--tmpfs", "/run", "--bind", str(scratch), "/tmp", "--ro-bind", str(out / "writer59"), "/writer59", "--ro-bind", str(out / "current"), "/current", "--ro-bind", str(capsule), "/capsule.py", "--setenv", "HOME", "/tmp", "--setenv", "PATH", "/usr/bin:/bin", "--setenv", "LANG", "C.UTF-8", "--chdir", "/tmp"]
    for name in ["passwd", "group", "nsswitch.conf", "hosts"]:
        command += ["--ro-bind", "/etc/" + name, "/etc/" + name]
    command += ["--", "/usr/bin/python3", "-B", "/capsule.py"]
    (out / "command.json").write_text(json.dumps(command, indent=2) + "\n")
    with (out / "test.log").open("w") as log:
        result = subprocess.run(command, stdout=log, stderr=subprocess.STDOUT, timeout=300)
    report["exit"] = result.returncode
    (out / "result.json").write_text(json.dumps(report, indent=2) + "\n")
    print((out / "test.log").read_text())
    print("Evidence:", out)
    return result.returncode


if __name__ == "__main__":
    raise SystemExit(main())
