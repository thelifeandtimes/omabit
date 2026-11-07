#!/usr/bin/env python3
"""Original application journal writer -> fleet writer -> original reader refusal.

Only synthetic state in a fresh networkless capsule. No installed profile or
credential service is mounted. This is a feature migration check, not a release.
"""
import argparse
import gzip
import hashlib
import importlib.util
import io
import json
import os
from pathlib import Path
import re
import subprocess
import tarfile

ROOT = Path(__file__).resolve().parents[1]
OLD = "dddb2cbfa5708c68f4664127483b0ce575de46d1"
COMMON = r'''package collection
import("bytes";"context";"encoding/json";"errors";"fmt";"net/http";"net/http/httptest";"os";"path/filepath";"strings";"testing";"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy";"git.thechristophgroup.org/thelifeandtimes/omabit/internal/shiproute")
func upgradeBinding() shiproute.Binding {return shiproute.Binding{RunnerID:"runner-fixture",Target:apppolicy.Target{InstanceID:"alpha"},Ship:"~zod",Continuity:strings.Repeat("c",64)}}
func upgradeTarget() ApplicationTarget {return ApplicationTarget{PluginID:"fixture.plugin",Profile:"default",SHA256:strings.Repeat("a",64)}}
func upgradeWrite(t *testing.T,name string,v any){t.Helper();b,e:=json.Marshal(v);if e!=nil{t.Fatal(e)};if e=os.WriteFile("/tmp/"+name,b,0600);e!=nil{t.Fatal(e)}}
func upgradeRead(t *testing.T,name string,v any){t.Helper();b,e:=os.ReadFile("/tmp/"+name);if e!=nil||json.Unmarshal(b,v)!=nil{t.Fatal("read fixture",name,e)}}
var _ = []any{bytes.Equal,context.Background,errors.New,fmt.Sprint,http.MethodGet,httptest.NewServer,filepath.Join}
'''
OLD_TEST = COMMON + r'''
func TestOriginalApplicationWriter(t *testing.T){
 ctx:=context.Background();r,e:=Open("/tmp/state");if e!=nil{t.Fatal(e)}
 server:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,q *http.Request){if q.URL.Path!="/~/host"{t.Error("unexpected request")};fmt.Fprint(w,"~zod")}));defer server.Close()
 ship,e:=r.EnrollShip(ctx,"legacy",server.URL,"~zod","file");if e!=nil{t.Fatal(e)}
 ship,e=r.SaveShipSession(ctx,ship,NewSessionRef(),func(context.Context)error{return nil});if e!=nil{t.Fatal(e)}
 legacyTarget:=upgradeTarget();legacyTarget.PluginID="fixture.legacy"
 a,e:=r.ApproveApplication(ctx,ship,strings.Repeat("1",32),legacyTarget);if e!=nil{t.Fatal(e)}
 a=closeUnissued(t,r,a.ID);if _,e=r.ArchiveApplication(ctx,a.ID,a.Revision);e!=nil{t.Fatal(e)};upgradeWrite(t,"legacy-archived.json",a)
 pending,e:=r.ApproveApplication(ctx,ship,strings.Repeat("2",32),legacyTarget);if e!=nil{t.Fatal(e)}
 e=r.WithApplication(ctx,pending.ID,func(tx *ApplicationTransaction)error{if e:=tx.Advance(ctx,"issuing","");e!=nil{return e};pending=tx.Current();return nil});if e!=nil{t.Fatal(e)}
 upgradeWrite(t,"legacy-pending.json",pending);upgradeWrite(t,"legacy-ship.json",ship)
 if _,e=r.SetApplicationShipApproval(ctx,0,upgradeBinding(),upgradeTarget(),true);e!=nil{t.Fatal(e)}
 raw,e:=os.ReadFile(filepath.Join(r.dir,"applications.json"));if e!=nil{t.Fatal(e)};var state applicationsFile
 if json.Unmarshal(raw,&state)!=nil||state.Version!=2{t.Fatal("not original format2 writer")}
 for _,name:=range []string{"ships.json","application-ship-approvals.json","application-history/"+a.ID+".json"}{
  data,e:=os.ReadFile(filepath.Join(r.dir,name));if e!=nil{t.Fatal(e)}
  if e=os.WriteFile("/tmp/original-"+strings.ReplaceAll(name,"/","-"),data,0600);e!=nil{t.Fatal(e)}
 }
}
func TestOriginalApplicationReaderRefusesFleetUnchanged(t *testing.T){
 ctx:=context.Background();r,e:=Open("/tmp/state");if e!=nil{t.Fatal(e)}
 before,e:=os.ReadFile(filepath.Join(r.dir,"applications.json"));if e!=nil{t.Fatal(e)}
 if _,e=r.Applications(ctx);!errors.Is(e,ErrApplicationJournal){t.Fatal("old reader accepted format3",e)}
 var ship Ship;upgradeRead(t,"legacy-ship.json",&ship)
 if _,e=r.ApproveApplication(ctx,ship,strings.Repeat("9",32),upgradeTarget());!errors.Is(e,ErrApplicationJournal){t.Fatal("old writer modified format3",e)}
 if _,_,e=r.readApplicationArchive(strings.Repeat("3",32));!errors.Is(e,ErrApplicationJournal){t.Fatal("old archive reader accepted fleet ancestry",e)}
 after,e:=os.ReadFile(filepath.Join(r.dir,"applications.json"));if e!=nil||!bytes.Equal(before,after){t.Fatal("old writer changed refused journal",e)}
}
'''
NEW_TEST = COMMON.replace('"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"', '"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol";"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"') + r'''
func TestFleetApplicationMigrationPreservesLegacyAndReopens(t *testing.T){
 ctx:=context.Background();r,e:=Open("/tmp/state");if e!=nil{t.Fatal(e)}
 before,e:=os.ReadFile(filepath.Join(r.dir,"applications.json"));if e!=nil{t.Fatal(e)}
 var pending,oldArchive ApplicationAttempt;upgradeRead(t,"legacy-pending.json",&pending);upgradeRead(t,"legacy-archived.json",&oldArchive)
 all,e:=r.Applications(ctx);if e!=nil||len(all)!=1||all[0]!=pending||pending.Fleet!=(FleetApplication{}){t.Fatal("original pending receipt changed",e)}
 after,e:=os.ReadFile(filepath.Join(r.dir,"applications.json"));if e!=nil||!bytes.Equal(before,after){t.Fatal("read migrated bytes")}
 binding:=upgradeBinding();fleet:=FleetApplication{Version:1,RunnerName:"fixture",GrantRevision:1,Request:protocol.ShipRouteCodeRequest{Binding:binding,Selection:protocol.ShipRouteSelection{Target:binding.Target,RuntimeID:strings.Repeat("a",64),ConfigRevision:1},Route:shiproute.Route{ID:"web",Origin:"https://ship.example",Network:"public",Source:"manual",Enabled:true}}}
 a,e:=r.ApproveFleetApplication(ctx,strings.Repeat("3",32),upgradeTarget(),fleet,"file");if e!=nil{t.Fatal(e)}
 a=closeUnissued(t,r,a.ID);if _,e=r.ArchiveApplication(ctx,a.ID,a.Revision);e!=nil{t.Fatal(e)}
 current,e:=r.ApproveFleetApplication(ctx,strings.Repeat("4",32),upgradeTarget(),fleet,"file");if e!=nil{t.Fatal(e)}
 e=r.WithApplication(ctx,current.ID,func(tx *ApplicationTransaction)error{if e:=tx.Advance(ctx,"issuing","");e!=nil{return e};current=tx.Current();return nil});if e!=nil{t.Fatal(e)}
 r,e=Open("/tmp/state");if e!=nil{t.Fatal(e)}
 got,archived,e:=r.Application(ctx,a.ID);if e!=nil||!archived||got!=a{t.Fatal("fleet archive lost",e)}
 got,archived,e=r.Application(ctx,current.ID);if e!=nil||archived||got!=current{t.Fatal("fleet pending lost",e)}
 got,archived,e=r.Application(ctx,pending.ID);if e!=nil||archived||got!=pending{t.Fatal("legacy pending lost",e)}
 got,archived,e=r.Application(ctx,oldArchive.ID);if e!=nil||!archived||got!=oldArchive{t.Fatal("legacy archive lost",e)}
 for _,name:=range []string{"ships.json","application-ship-approvals.json","application-history/"+oldArchive.ID+".json"}{
  old,e:=os.ReadFile("/tmp/original-"+strings.ReplaceAll(name,"/","-"));if e!=nil{t.Fatal(e)}
  now,e:=os.ReadFile(filepath.Join(r.dir,name));if e!=nil||!bytes.Equal(old,now){t.Fatal("original state bytes changed",name,e)}
 }
 raw,e:=os.ReadFile(filepath.Join(r.dir,"applications.json"));if e!=nil{t.Fatal(e)};var state applicationsFile
 if json.Unmarshal(raw,&state)!=nil||state.Version!=3{t.Fatal("fleet archive downgraded format")}
}
'''


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--output", type=Path, required=True)
    out = p.parse_args().output.absolute()
    if out.parent != Path("/tmp") or not re.fullmatch(r"omabit-fleet-attempt-upgrade-[A-Za-z0-9-]+", out.name) or out.exists():
        p.error("Use a new /tmp/omabit-fleet-attempt-upgrade-NAME directory")
    spec = importlib.util.spec_from_file_location("storage", ROOT / "scripts/qualification/app-development/storage.py")
    storage = importlib.util.module_from_spec(spec); spec.loader.exec_module(storage)
    # Collection's small dependency closure and two warm-cache binaries avoid
    # the complete managed-runtime source snapshots used by runner migrations.
    admission = storage.admit(400 << 20)
    out.mkdir(mode=0o700); scratch = out / "scratch"; scratch.mkdir(mode=0o700)
    env = {**os.environ, "GOCACHE": "/tmp/omabit-040-go-cache", "GOTOOLCHAIN": "local", "GOPROXY": "off"}
    def git(*args): return subprocess.check_output(["git", *args], cwd=ROOT)
    # Copy exactly the collection package's local dependency/test closure, not
    # unrelated runtime assets. Each copied input is retained with its digest.
    names = subprocess.check_output(["go", "list", "-deps", "-test", "-f", '{{if and (not .Standard) .Module}}{{if eq .Module.Path "git.thechristophgroup.org/thelifeandtimes/omabit"}}{{.Dir}}{{end}}{{end}}', "./internal/collection"], cwd=ROOT, env=env, text=True)
    paths = sorted({str(Path(v).relative_to(ROOT)) for v in names.splitlines() if v})
    manifest = {"original_source": OLD, "current_base": git("rev-parse", "HEAD").decode().strip(), "dependency_paths": paths, "overlays": {}, "binaries": {}}
    (out / "storage-admission.json").write_text(json.dumps(admission, indent=2)+"\n")
    for name, commit, helper in [("original", OLD, OLD_TEST), ("current", manifest["current_base"], NEW_TEST)]:
        source = out / (name+"-source"); source.mkdir()
        with tarfile.open(fileobj=io.BytesIO(git("archive", commit, "go.mod", "go.sum", *paths))) as archive:
            archive.extractall(source, filter="data")
        if name == "current":
            changes = set(git("diff", "HEAD", "--name-only", "--diff-filter=ACMR", "-z").decode().split("\0") + git("ls-files", "--others", "--exclude-standard", "-z").decode().split("\0")) - {""}
            for rel in sorted(changes):
                if not any(rel.startswith(v+"/") for v in paths): continue
                original = ROOT / rel
                if original.is_symlink() or not original.is_file(): raise ValueError("unsafe source overlay")
                data = original.read_bytes(); target = source / rel; target.parent.mkdir(parents=True, exist_ok=True); target.write_bytes(data)
                manifest["overlays"][rel] = hashlib.sha256(data).hexdigest()
        (source / "internal/collection/zz_fleet_upgrade_test.go").write_text(helper)
        manifest[name+"_helper_sha256"] = hashlib.sha256(helper.encode()).hexdigest()
        binary = out / name
        result = subprocess.run(["go", "test", "-race", "-c", "-o", str(binary), "./internal/collection"], cwd=source, env=env, capture_output=True, text=True)
        (out / (name+"-build.log")).write_text(result.stdout+result.stderr)
        if result.returncode: raise RuntimeError(result.stdout+result.stderr)
        manifest["binaries"][name] = hashlib.sha256(binary.read_bytes()).hexdigest()
    capsule = out / "capsule.py"
    capsule.write_text('import subprocess\nfor binary,case in '+repr([("original", "TestOriginalApplicationWriter"), ("current", "TestFleetApplicationMigrationPreservesLegacyAndReopens"), ("original", "TestOriginalApplicationReaderRefusesFleetUnchanged")])+':\n print(case,flush=True)\n subprocess.run(["/"+binary,"-test.run","^"+case+"$","-test.timeout","1m","-test.v"],check=True)\n')
    command = ["bwrap", "--unshare-all", "--die-with-parent", "--clearenv", "--ro-bind", "/usr", "/usr", "--symlink", "usr/bin", "/bin", "--symlink", "usr/lib", "/lib", "--symlink", "usr/lib", "/lib64", "--proc", "/proc", "--dev", "/dev", "--dir", "/etc", "--tmpfs", "/run", "--bind", str(scratch), "/tmp", "--ro-bind", str(out/"original"), "/original", "--ro-bind", str(out/"current"), "/current", "--ro-bind", str(capsule), "/capsule.py", "--setenv", "HOME", "/tmp", "--setenv", "PATH", "/usr/bin:/bin", "--setenv", "LANG", "C.UTF-8", "--chdir", "/tmp"]
    for name in ["passwd", "group", "nsswitch.conf", "hosts"]: command += ["--ro-bind", "/etc/"+name, "/etc/"+name]
    command += ["--", "/usr/bin/python3", "-B", "/capsule.py"]
    (out/"command.json").write_text(json.dumps(command, indent=2)+"\n")
    with (out/"test.log").open("w") as log: result = subprocess.run(command, stdout=log, stderr=subprocess.STDOUT, timeout=210)
    manifest.update(exit=result.returncode, isolation="private network/user/PID/mount/IPC/UTS; synthetic state only", scope="functional format migration, not installed or release qualification")
    (out/"result.json").write_text(json.dumps(manifest, indent=2)+"\n")
    for name, sha in manifest["binaries"].items():
        binary = out/name
        with binary.open("rb") as src, gzip.open(str(binary)+".gz", "wb") as dst:
            while block := src.read(1<<20): dst.write(block)
        with gzip.open(str(binary)+".gz", "rb") as src: assert hashlib.file_digest(src,"sha256").hexdigest() == sha
        binary.unlink()
    print((out/"test.log").read_text()); print("Evidence:",out)
    return result.returncode


if __name__ == "__main__":
    raise SystemExit(main())
