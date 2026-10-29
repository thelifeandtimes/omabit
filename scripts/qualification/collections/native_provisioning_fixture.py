"""Private same-compositor setup crash fixture; never host/bootstrap code."""
from pathlib import Path
import json,os,select,signal,subprocess,sys,uuid
sys.path.insert(0,'/lab/source')
import authority_provisioning as p
import runtime_authority as a
import journal as j
from journal_runtime_fixture import ROOT,guard,read,save,sha


def snapshot():
 paths=[ROOT/'collection-journal',ROOT/'runtime'/a.ANCHOR]
 files=[ROOT/'runtime'/a.ANCHOR_SEAL,ROOT/'runtime'/p.CLAIM]
 for root in paths:files.extend(x for x in root.rglob('*') if x.is_file())
 return {str(x.relative_to(ROOT)):{'inode':x.stat().st_ino,'sha256':sha(x)} for x in files}


def initialize(store):
 guard();attempt=uuid.uuid4().hex;display=ROOT/'runtime'/os.environ['WAYLAND_DISPLAY']
 with store.lease():
  # Explicitly create this new fixture's empty journal before child ownership.
  a.write_new(store.fd,'journal.json',{'format':2,'events':[]});store.fresh=False
 report={'passed':False,'attempt':attempt,'steps':[],'faults':[],'controller_kills':0}
 def child(fault=None):
  save(ROOT/'provision-control.json',{'attempt':attempt,'display':str(display),'fault':fault})
  proc=subprocess.Popen([sys.executable,'-B','/lab/native_provisioning_fixture.py','--worker'],stdout=subprocess.PIPE,stderr=subprocess.PIPE)
  try:
   if fault:
    output,errors=proc.communicate(timeout=8);assert proc.returncode==-signal.SIGKILL,(proc.returncode,errors[-1200:]);assert not output
    report['faults'].append(fault);result=None
   else:
    assert select.select([proc.stdout],[],[],8)[0],'setup controller response deadline'
    line=proc.stdout.readline()
    if not line:
     proc.wait(timeout=3);raise AssertionError(proc.stderr.read().decode()[-1200:])
    result=json.loads(line);proc.kill();assert proc.wait(timeout=3)==-signal.SIGKILL
    report['steps'].append(result)
   report['controller_kills']+=1
   return result
  finally:
   if proc.poll() is None:proc.kill();proc.wait(timeout=3)
   proc.stdout.close();proc.stderr.close()
 faults={'planned':'after-claim-create','claim-published':'after-root-descriptor','seal-prepared':'after-seal-rename','mirror-published':'after-root-rename'}
 result=child()
 while not result['complete']:
  fault=faults.get(result['phase'])
  if fault:child(fault)
  result=child()
 assert len(report['steps'])==9 and report['controller_kills']==13
 assert len({x['authority'] for x in report['steps']})==1
 record=p.read(store);assert record['complete'] and len(record['attempts'])==6
 abandoned=[x for x in record['attempts'] if x['abandoned']];assert [x['kind'] for x in abandoned]==['claim','root']
 report['abandoned_stages']={x['kind']:{'path':x['path'],'inode':Path(x['path']).stat().st_ino,
  'files':({f.name:sha(f) for f in Path(x['path']).iterdir()} if x['kind']=='root' else {'body':sha(Path(x['path']))})} for x in abandoned}
 assert set(report['abandoned_stages']['root']['files'])=={'authority.lock','authority.json'}
 report['authority']=record['authority'];report['complete_record_sha256']=sha(store.root/p.RECORD)
 report['initial_snapshot']=snapshot();report['passed']=True
 save(ROOT/'provisioning-result.json',report)


def verify_completed():
 guard();report=read(ROOT/'provisioning-result.json');before=snapshot();store=j.Store(ROOT/'collection-journal')
 try:
  with store.lease() as lease:opened=p.provision(ROOT/'runtime'/os.environ['WAYLAND_DISPLAY'],store,lease,report['attempt'])
  opened.close();assert snapshot()==before
  assert sha(store.root/p.RECORD)==report['complete_record_sha256']
  for kind,row in report['abandoned_stages'].items():
   path=Path(row['path']);assert path.stat().st_ino==row['inode']
   files={f.name:sha(f) for f in path.iterdir()} if kind=='root' else {'body':sha(path)}
   assert files==row['files']
  report['post_recovery_retry_unchanged']=True;report['final_snapshot']=before
  save(ROOT/'provisioning-result.json',report)
 finally:store.close()


def worker():
 guard();control=read(ROOT/'provision-control.json');store=j.Store(ROOT/'collection-journal');fault=control['fault']
 original_write=a.write_new;original_rename=p.rename_no_replace
 def write(fd,name,value):
  original_write(fd,name,value)
  if (fault=='after-claim-create' and '-claim-' in name) or (fault=='after-root-descriptor' and name=='authority.json'):os.kill(os.getpid(),signal.SIGKILL)
 def rename(fd,src,dst):
  original_rename(fd,src,dst)
  if (fault=='after-seal-rename' and dst==a.ANCHOR_SEAL) or (fault=='after-root-rename' and dst==a.ANCHOR):os.kill(os.getpid(),signal.SIGKILL)
 a.write_new=write;p.rename_no_replace=rename
 try:
  with store.lease() as lease:result=p.step(control['display'],store,lease,control['attempt'])
  print(json.dumps(result),flush=True);signal.pause()
 finally:store.close()

if __name__=='__main__':worker()
