#!/usr/bin/env python3
"""Run original schema49 writer -> current migration -> original older-reader refusal.
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
OLD = 'e2f6377da3706ceff54ddc33915ea29b15f397c9'
OLD_TEST = r'''package managed
import("bytes";"os";"testing";"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy")
func TestDeskOriginalWriter(t *testing.T) {
 if StateSchemaVersion!=49 {t.Fatal("not the original writer")}
 if err:=os.Mkdir("/tmp/schema-state",0700);err!=nil{t.Fatal(err)}
 s,err:=testOpenStore("/tmp/schema-state");if err!=nil{t.Fatal(err)};defer s.close()
 if _,_,err=s.register("fixture",registration("alpha","original-register",18080));err!=nil{t.Fatal(err)}
 _,err=s.putAppPolicy("fixture",apppolicy.Update{Scope:"desk",Target:apppolicy.Target{InstanceID:"alpha",Desk:"example"},Gates:apppolicy.Gates{CreateDesk:apppolicy.Inherit,Commit:apppolicy.Allow,Publish:apppolicy.Block}});if err!=nil{t.Fatal(err)}
}
func TestDeskOlderReaderRefusesUnchanged(t *testing.T) {
 path:="/tmp/schema-state/runner.sqlite";before,err:=os.ReadFile(path);if err!=nil{t.Fatal(err)}
 if s,err:=testOpenStore("/tmp/schema-state");err==nil{s.close();t.Fatal("older reader accepted newer state")}
 after,err:=os.ReadFile(path);if err!=nil||!bytes.Equal(before,after){t.Fatal("refused reader changed state",err)}
}
'''
NEW_TEST = r'''package managed
import("reflect";"testing")
func TestDeskCurrentMigration(t *testing.T) {
 if StateSchemaVersion<50{t.Fatal("new journal migration absent")}
 db:=g02RawDB(t,"/tmp/schema-state");before:=originalDeskRows(t,db);g02Check(t,db.Close())
 s,err:=testOpenStore("/tmp/schema-state");g02Check(t,err)
 if !reflect.DeepEqual(before,originalDeskRows(t,s.db)){t.Fatal("migration changed original records")}
 r,_,err:=s.createDeskOperation("fixture",deskIntentFixture());g02Check(t,err)
 r,err=s.appendDeskStep(r,deskStep(r,"dispatched"));g02Check(t,err);g02Check(t,s.close())
 s,err=testOpenStore("/tmp/schema-state");g02Check(t,err);defer s.close()
 g02Check(t,s.reconcileDeskOperations());after,err:=loadDeskOperation(s.db,r.Intent.ID);g02Check(t,err)
 if after.Steps[len(after.Steps)-1].State!="uncertain"{t.Fatal("effect was replayed or lost")}
 if !reflect.DeepEqual(before,originalDeskRows(t,s.db)){t.Fatal("reopen changed original records")}
}
'''
CAPSULE = '''import subprocess
for binary,case in [("/writer49","TestDeskOriginalWriter"),("/current","TestDeskCurrentMigration"),("/writer49","TestDeskOlderReaderRefusesUnchanged")]:
 print(case,flush=True)
 subprocess.run([binary,"-test.run","^"+case+"$","-test.timeout","1m"],check=True,cwd="/tmp")
'''
def git(*args):return subprocess.check_output(['git',*args],cwd=ROOT)
def digest(data):return hashlib.sha256(data).hexdigest()
def main():
 p=argparse.ArgumentParser(description=__doc__);p.add_argument('--output',type=Path,required=True);args=p.parse_args()
 out=args.output.resolve();out.mkdir(mode=0o700);scratch=out/'scratch';scratch.mkdir(mode=0o700)
 head=git('rev-parse','HEAD').decode().strip();manifest={'original_source':OLD,'current_base':head,'overlays':{},'binaries':{}}
 for name,commit,extra in [('writer49',OLD,OLD_TEST),('current',head,NEW_TEST)]:
  source=out/(name+'-source');source.mkdir()
  with tarfile.open(fileobj=io.BytesIO(git('archive',commit))) as archive:archive.extractall(source,filter='data')
  if name=='current':
   paths=set(git('diff','HEAD','--name-only','--diff-filter=ACMR','-z').decode().split('\0')+git('ls-files','--others','--exclude-standard','-z').decode().split('\0'))-{''}
   for path in sorted(paths):
    origin=ROOT/path
    if origin.is_symlink() or not origin.is_file():raise ValueError('non-regular source overlay: '+path)
    data=origin.read_bytes();target=source/path;target.parent.mkdir(parents=True,exist_ok=True);target.write_bytes(data);manifest['overlays'][path]=digest(data)
  helper=source/'internal/managed/zz_desk_upgrade_probe_test.go';helper.write_text(extra)
  manifest[name+'_helper_sha256']=digest(extra.encode())
  binary=out/name
  subprocess.run(['go','test','-c','-o',str(binary),'./internal/managed'],cwd=source,check=True)
  manifest['binaries'][name]=digest(binary.read_bytes())
 capsule=out/'capsule.py';capsule.write_text(CAPSULE)
 command=['bwrap','--unshare-all','--die-with-parent','--clearenv','--ro-bind','/usr','/usr','--symlink','usr/bin','/bin','--symlink','usr/lib','/lib','--symlink','usr/lib','/lib64','--proc','/proc','--dev','/dev','--dir','/etc','--tmpfs','/run','--bind',str(scratch),'/tmp','--ro-bind',str(out/'writer49'),'/writer49','--ro-bind',str(out/'current'),'/current','--ro-bind',str(capsule),'/capsule.py','--setenv','HOME','/tmp','--setenv','PATH','/usr/bin:/bin','--setenv','LANG','C.UTF-8','--chdir','/tmp']
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
