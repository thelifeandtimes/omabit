#!/usr/bin/env python3
"""Exercise the actual schema58 writer, schema59 migration and old-reader refusal.
Private synthetic state only; no ships, Docker or services. Not release qualification.
"""
import argparse,gzip,hashlib,importlib.util,io,json,os,shutil,subprocess,tarfile
from pathlib import Path
ROOT=Path(__file__).resolve().parents[1]
OLD='66c641bb41771c03391de3057cbf62c096f3ae51'
def module(name,path):
 spec=importlib.util.spec_from_file_location(name,path);m=importlib.util.module_from_spec(spec);spec.loader.exec_module(m);return m
previous=module('routes_upgrade',ROOT/'scripts/test-ship-routes-upgrade.py')
OLD_TEST=previous.OLD_TEST.replace('StateSchemaVersion!=57','StateSchemaVersion!=58').replace('original schema57','original schema58')
OLD_TEST=OLD_TEST.replace('"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"','"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork";"git.thechristophgroup.org/thelifeandtimes/omabit/internal/shiproute"')
OLD_TEST=OLD_TEST.replace('func TestDeskOlderReaderRefusesUnchanged', '''func TestResourceOriginalRoutes(t *testing.T) {
 s,err:=testOpenStore("/tmp/schema-state");g02Check(t,err);defer s.close()
 u:=shiproute.Update{Binding:shiproute.Binding{RunnerID:s.runnerID,Target:apppolicy.Target{InstanceID:"alpha"},Ship:"~zod",Continuity:strings.Repeat("c",64)},Settings:shiproute.DefaultSettings()};u.Settings.DisabledAutomatic=[]string{"managed-http"}
 _,err=s.putShipRoutes("fixture",u);g02Check(t,err)
}
func TestDeskOlderReaderRefusesUnchanged''')
NEW_TEST=r'''package managed
import("database/sql";"reflect";"testing";"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol")
func resourceOriginalRows(t *testing.T,db *sql.DB)map[string][]string{
 rows:=originalShipRouteRows(t,db)
 for _,table:=range []string{"ship_routes","ship_routes_history"}{
  r,err:=db.Query("SELECT body FROM "+table+" ORDER BY body");g02Check(t,err)
  for r.Next(){var b []byte;g02Check(t,r.Scan(&b));rows[table]=append(rows[table],string(b))};g02Check(t,r.Err());g02Check(t,r.Close())
 };return rows
}
func TestResourceCurrentMigration(t *testing.T){
 if StateSchemaVersion!=59{t.Fatal("not schema59")}
 db:=g02RawDB(t,"/tmp/schema-state");before:=resourceOriginalRows(t,db);metadata:=originalDeskRows(t,db);g02Check(t,db.Close())
 for n:=0;n<2;n++{
  s,err:=testOpenStore("/tmp/schema-state");g02Check(t,err)
  if !reflect.DeepEqual(before,resourceOriginalRows(t,s.db))||!reflect.DeepEqual(metadata,originalDeskRows(t,s.db)){t.Fatal("migration rewrote original rows")}
  v,err:=loadInstance(s.db,"alpha");g02Check(t,err)
  if protocol.EffectiveMemoryGiB(v.Public.Config.MemoryGiB)!=4||protocol.EffectiveCPUCores(v.Runtime.CPUCores)!=2{t.Fatal("historical defaults changed")}
  g02Check(t,s.close())
 }
 s,err:=testOpenStore("/tmp/schema-state");g02Check(t,err)
 v,err:=loadInstance(s.db,"alpha");g02Check(t,err);config:=v.Public.Config;config.MemoryGiB,config.CPUCores=16,8
 _,err=s.configure("resource-owner","alpha",protocol.ConfigureRequest{ExpectedRevision:v.Public.ConfigRevision,Config:config});g02Check(t,err);g02Check(t,s.close())
 s,err=testOpenStore("/tmp/schema-state");g02Check(t,err);defer s.close()
 v,err=loadInstance(s.db,"alpha");g02Check(t,err);if v.Public.Config.MemoryGiB!=16||v.Public.Config.CPUCores!=8{t.Fatal("reopen lost limits")}
 if !reflect.DeepEqual(before,resourceOriginalRows(t,s.db)){t.Fatal("resize rewrote app or URL receipts")}
}
'''
def sha(path):
 with path.open('rb') as f:return hashlib.file_digest(f,'sha256').hexdigest()
def main():
 p=argparse.ArgumentParser(description=__doc__);p.add_argument('--output',type=Path,required=True);a=p.parse_args()
 # Sparse copies omit old source archives that these selected tests never open.
 # All actual .go source and test fixtures remain pinned to their original commit.
 storage=module('storage',ROOT/'scripts/qualification/app-development/storage.py')
 warm=bool(os.environ.get('GOCACHE'));admission=storage.admit((450 if warm else 1200)*1024**2)
 out=a.output.resolve();out.mkdir(mode=0o700);scratch=out/'scratch';scratch.mkdir(mode=0o700)
 (out/'storage-admission.json').write_text(json.dumps(admission,indent=2)+'\n')
 head=previous.git('rev-parse','HEAD').decode().strip();report={'original_source':OLD,'current_base':head,'overlays':{},'binaries':{},'omitted_archive_inputs':[]}
 for name,commit,extra in [('writer58',OLD,OLD_TEST),('current',head,NEW_TEST)]:
  source=out/(name+'-source');source.mkdir()
  with tarfile.open(fileobj=io.BytesIO(previous.git('archive',commit))) as archive:
   members=[]
   for member in archive.getmembers():
    if member.name.startswith('internal/managed/testdata/historical-upgrades/') and member.name.endswith('-source.tar.gz'):
     report['omitted_archive_inputs'].append({'source':name,'path':member.name});continue
    members.append(member)
   archive.extractall(source,members=members,filter='data')
  if name=='current':
   paths=set(previous.git('diff','HEAD','--name-only','--diff-filter=ACMR','-z').decode().split('\0')+previous.git('ls-files','--others','--exclude-standard','-z').decode().split('\0'))-{''}
   for path in sorted(paths):
    origin=ROOT/path
    if origin.is_symlink() or not origin.is_file():raise ValueError('non-regular overlay: '+path)
    target=source/path;target.parent.mkdir(parents=True,exist_ok=True);shutil.copyfile(origin,target);report['overlays'][path]=sha(target)
  helper=source/'internal/managed/zz_resource_upgrade_test.go';helper.write_text(extra)
  report[name+'_helper_sha256']=sha(helper)
  binary=out/name;subprocess.run(['go','test','-c','-o',str(binary),'./internal/managed'],cwd=source,check=True)
  report['binaries'][name]=sha(binary)
 capsule=out/'capsule.py';capsule.write_text('''import subprocess,sqlite3
from pathlib import Path
for binary,case in [("/writer58","TestDeskOriginalWriter"),("/writer58","TestResourceOriginalRoutes"),("/current","TestResourceCurrentMigration"),("/writer58","TestDeskOlderReaderRefusesUnchanged")]:
 print(case,flush=True);subprocess.run([binary,"-test.run","^"+case+"$","-test.timeout","1m"],check=True,cwd="/tmp")
 if case=="TestResourceOriginalRoutes":
  with sqlite3.connect("file:/tmp/schema-state/runner.sqlite?mode=ro",uri=True) as db:
   assert db.execute("pragma user_version").fetchone()[0]==58
   Path("/tmp/original-schema58.sql").write_text("\\n".join(db.iterdump())+"\\nPRAGMA user_version=58;\\n")
''')
 command=['bwrap','--unshare-all','--die-with-parent','--clearenv','--ro-bind','/usr','/usr','--symlink','usr/bin','/bin','--symlink','usr/lib','/lib','--symlink','usr/lib','/lib64','--proc','/proc','--dev','/dev','--dir','/etc','--tmpfs','/run','--bind',str(scratch),'/tmp','--ro-bind',str(out/'writer58'),'/writer58','--ro-bind',str(out/'current'),'/current','--ro-bind',str(capsule),'/capsule.py','--setenv','HOME','/tmp','--setenv','PATH','/usr/bin:/bin','--setenv','LANG','C.UTF-8','--chdir','/tmp']
 for name in ['passwd','group','nsswitch.conf','hosts']:command+=['--ro-bind','/etc/'+name,'/etc/'+name]
 command+=['--','/usr/bin/python3','-B','/capsule.py'];(out/'command.json').write_text(json.dumps(command,indent=2)+'\n')
 with (out/'test.log').open('w') as log:result=subprocess.run(command,stdout=log,stderr=subprocess.STDOUT,timeout=240)
 report.update(exit=result.returncode,scope='synthetic migration only; no installed upgrade or release qualification')
 for name in report['binaries']:
  binary=out/name
  with binary.open('rb') as src,gzip.open(str(binary)+'.gz','wb',compresslevel=6) as dst:shutil.copyfileobj(src,dst)
  with gzip.open(str(binary)+'.gz','rb') as src:assert hashlib.file_digest(src,'sha256').hexdigest()==report['binaries'][name]
  binary.unlink()
 (out/'result.json').write_text(json.dumps(report,indent=2)+'\n');print((out/'test.log').read_text());print('Evidence:',out);return result.returncode
if __name__=='__main__':raise SystemExit(main())
