#!/usr/bin/env python3
"""Run original schema53 writer -> current migration -> original older-reader refusal.
Synthetic private state only; no services, engines, sockets, piers or network.
Requires cached Go dependencies and bubblewrap. Does not qualify a release.
"""
import argparse
import hashlib
import io
import json
import os
from pathlib import Path
import subprocess
import tarfile

ROOT = Path(__file__).resolve().parents[1]
OLD = '3a5a5c52a396089231d12998f7316a8d24a48ac9'
OLD_TEST = r'''package managed
import("bytes";"os";"strings";"testing";"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy";"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork")
func TestDeskOriginalWriter(t *testing.T) {
 if StateSchemaVersion!=53 {t.Fatal("not the original schema53 writer")}
 if err:=os.Mkdir("/tmp/schema-state",0700);err!=nil{t.Fatal(err)}
 s,err:=testOpenStore("/tmp/schema-state");if err!=nil{t.Fatal(err)};defer s.close()
 if _,_,err=s.register("fixture",registration("alpha","original-register",18080));err!=nil{t.Fatal(err)}
 _,err=s.putAppPolicy("fixture",apppolicy.Update{Scope:"desk",Target:apppolicy.Target{InstanceID:"alpha",Desk:"example"},Gates:apppolicy.Gates{CreateDesk:apppolicy.Inherit,Commit:apppolicy.Allow,Publish:apppolicy.Block}});if err!=nil{t.Fatal(err)}
 r,_,err:=s.createDeskOperation("fixture",deskIntentFixture());if err!=nil{t.Fatal(err)}
 _,err=s.appendDeskStep(r,deskStep(r,"dispatched"));if err!=nil{t.Fatal(err)}
 target:=apppolicy.Target{InstanceID:"alpha",Desk:"example"};setting:=deskwork.DefaultSetting(target);setting.Desk.Instructions="Compile and check the receiver before release";_,err=s.putDeskSetting("desktop",deskwork.SettingUpdate{Target:target,Desk:setting.Desk});if err!=nil{t.Fatal(err)}
 baseline:=sourceBaselineFixture();baseline.Binding.SettingsRevision=1;if err=s.saveSourceBaseline(baseline);err!=nil{t.Fatal(err)}
 record:=deploymentRecordFixture(t);record.Intent.Intent.ID=strings.Repeat("e",64);record.Intent.Intent.Target.Desk="preview";record.Intent.SourceName="preview"
 op,_,err:=s.createDeskDeployment(record);if err!=nil{t.Fatal(err)};completeDeploymentFixture(t,s,record,op)

}
func TestDeskOlderReaderRefusesUnchanged(t *testing.T) {
 path:="/tmp/schema-state/runner.sqlite";before,err:=os.ReadFile(path);if err!=nil{t.Fatal(err)}
 if s,err:=testOpenStore("/tmp/schema-state");err==nil{s.close();t.Fatal("older reader accepted newer state")}
 after,err:=os.ReadFile(path);if err!=nil||!bytes.Equal(before,after){t.Fatal("refused reader changed state",err)}
}
'''
NEW_TEST = r'''package managed
import("reflect";"testing";"strings";"time";"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy";"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork")
func TestDeskCurrentMigration(t *testing.T) {
 if StateSchemaVersion!=54{t.Fatal("not schema54 release writer")}
 target:=apppolicy.Target{InstanceID:"alpha",Desk:"example"};preview:=apppolicy.Target{InstanceID:"alpha",Desk:"preview"}
 db:=g02RawDB(t,"/tmp/schema-state");before:=originalDeskRows(t,db);receipt,err:=loadDeskOperation(db,deskIntentFixture().ID);g02Check(t,err);setting,err:=loadDeskSetting(db,deskSettingsKey(target));g02Check(t,err)
 owned,err:=loadOwnedSource(db,deskSettingsKey(preview));g02Check(t,err);deployed,err:=loadDeskDeployment(db,owned.OperationID);g02Check(t,err)
 var baselineBody []byte;g02Check(t,db.QueryRow("SELECT body FROM desk_source_baselines WHERE target=?",deskSettingsKey(target)).Scan(&baselineBody));baseline,err:=decodeSourceBaseline(baselineBody);g02Check(t,err);g02Check(t,db.Close())
 for pass:=0;pass<2;pass++ {
  s,err:=testOpenStore("/tmp/schema-state");g02Check(t,err)
  if !reflect.DeepEqual(before,originalDeskRows(t,s.db)){t.Fatal("migration changed original records")}
  migrated,err:=loadDeskOperation(s.db,receipt.Intent.ID);g02Check(t,err);if !reflect.DeepEqual(receipt,migrated){t.Fatal("migration changed effect receipt")}
  currentSetting,err:=s.deskSetting(target);g02Check(t,err);if !reflect.DeepEqual(setting,currentSetting){t.Fatal("migration changed workflow")}
  currentBaseline,err:=s.sourceBaseline(baseline.Binding);g02Check(t,err);if currentBaseline==nil || !reflect.DeepEqual(*currentBaseline,baseline){t.Fatal("migration changed canonical baseline")}
  currentOwned,err:=loadOwnedSource(s.db,deskSettingsKey(preview));g02Check(t,err);if !reflect.DeepEqual(currentOwned,owned){t.Fatal("migration changed verified ownership")}
  currentDeployment,err:=loadDeskDeployment(s.db,owned.OperationID);g02Check(t,err);if !reflect.DeepEqual(currentDeployment,deployed){t.Fatal("migration changed deployment")}
  if pass==0 {
   var count int;g02Check(t,s.db.QueryRow("SELECT count(*) FROM desk_releases").Scan(&count));if count!=0{t.Fatal("migration invented release evidence")}
   record:=releaseRecordFixture(t);record.Intent.Intent.ID=strings.Repeat("f",64);record.Intent.Intent.Target.Desk="release"
   record.Review.DeskSetting=deskwork.DefaultSetting(record.Intent.Intent.Target);record.Review.DeskSetting.Desk.Publication="source";g02Check(t,record.Review.Seal());record.Intent.ReviewDigest=record.Review.Digest
   op,_,err:=s.createDeskRelease(record);g02Check(t,err);op,err=s.appendDeskStep(op,deskStep(op,"dispatched"));g02Check(t,err);op,err=s.appendDeskStep(op,deskStep(op,"acknowledged"));g02Check(t,err)
   after:=record.Review.Observation;after.Publication="published";app:=record.Review.Application
   _,err=s.appendDeskStep(op,deskwork.Step{Kind:op.Intent.Action,State:"observed",At:time.Now().UTC(),Observation:&after,Application:&app});g02Check(t,err)
  } else {
   record,err:=loadDeskRelease(s.db,strings.Repeat("f",64));g02Check(t,err);op,err:=loadDeskOperation(s.db,record.Intent.Intent.ID);g02Check(t,err);if op.Steps[len(op.Steps)-1].State!="observed"{t.Fatal("release evidence lost")}
  }
  g02Check(t,s.close())
 }
}
'''

CAPSULE = '''import subprocess
for binary,case in [("/writer53","TestDeskOriginalWriter"),("/current","TestDeskCurrentMigration"),("/writer53","TestDeskOlderReaderRefusesUnchanged")]:
 print(case,flush=True)
 subprocess.run([binary,"-test.run","^"+case+"$","-test.timeout","1m"],check=True,cwd="/tmp")
 if case=="TestDeskOriginalWriter":
  import sqlite3
  from pathlib import Path
  with sqlite3.connect("file:/tmp/schema-state/runner.sqlite?mode=ro",uri=True) as db:
   assert db.execute("pragma user_version").fetchone()[0]==53
   Path("/tmp/original-schema53.sql").write_text("\\n".join(db.iterdump())+"\\nPRAGMA user_version=53;\\n")
'''
def git(*args):return subprocess.check_output(['git',*args],cwd=ROOT)
def digest(data):return hashlib.sha256(data).hexdigest()
def main():
 p=argparse.ArgumentParser(description=__doc__);p.add_argument('--output',type=Path,required=True);args=p.parse_args()
 out=args.output.resolve();out.mkdir(mode=0o700);scratch=out/'scratch';scratch.mkdir(mode=0o700)
 head=git('rev-parse','HEAD').decode().strip();manifest={'original_source':OLD,'current_base':head,'overlays':{},'binaries':{}}
 for name,commit,extra in [('writer53',OLD,OLD_TEST),('current',head,NEW_TEST)]:
  source=out/(name+'-source');source.mkdir()
  with tarfile.open(fileobj=io.BytesIO(git('archive',commit))) as archive:archive.extractall(source,filter='data')
  if name=='current':
   paths=set(git('diff','HEAD','--name-only','--diff-filter=ACMR','-z').decode().split('\0')+git('ls-files','--others','--exclude-standard','-z').decode().split('\0'))-{''}
   for path in sorted(paths):
    origin=ROOT/path
    if origin.is_symlink() or not origin.is_file():raise ValueError('non-regular source overlay: '+path)
    data=origin.read_bytes();target=source/path;target.parent.mkdir(parents=True,exist_ok=True);target.write_bytes(data);manifest['overlays'][path]=digest(data)
  helper=source/'internal/managed/zz_desk_release_upgrade_probe_test.go';helper.write_text(extra)
  manifest[name+'_helper_sha256']=digest(extra.encode())
  binary=out/name
  subprocess.run(['go','test','-c','-o',str(binary),'./internal/managed'],cwd=source,check=True)
  manifest['binaries'][name]=digest(binary.read_bytes())
 capsule=out/'capsule.py';capsule.write_text(CAPSULE)
 command=['bwrap','--unshare-all','--die-with-parent','--clearenv','--ro-bind','/usr','/usr','--symlink','usr/bin','/bin','--symlink','usr/lib','/lib','--symlink','usr/lib','/lib64','--proc','/proc','--dev','/dev','--dir','/etc','--tmpfs','/run','--bind',str(scratch),'/tmp','--ro-bind',str(out/'writer53'),'/writer53','--ro-bind',str(out/'current'),'/current','--ro-bind',str(capsule),'/capsule.py','--setenv','HOME','/tmp','--setenv','PATH','/usr/bin:/bin','--setenv','LANG','C.UTF-8','--chdir','/tmp']
 for name in ['passwd','group','nsswitch.conf','hosts']:
  path=Path('/etc')/name
  if path.is_file():command+=['--ro-bind',str(path),str(path)]
 command+=['--','/usr/bin/python3','-B','/capsule.py']
 (out/'command.json').write_text(json.dumps(command,indent=2)+'\n')
 with (out/'test.log').open('w') as log:
  result=subprocess.run(command,stdout=log,stderr=subprocess.STDOUT,timeout=240)
 manifest.update(exit=result.returncode,isolation='private user/network/PID/mount/IPC/UTS; synthetic state only',scope='functional migration; not release qualification')
 (out/'result.json').write_text(json.dumps(manifest,indent=2)+'\n');print((out/'test.log').read_text());print('evidence:',out)
 return result.returncode
if __name__=='__main__':raise SystemExit(main())
