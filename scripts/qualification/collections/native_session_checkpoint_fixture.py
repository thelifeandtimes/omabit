"""Private durable evidence capture and original runtime-directory retirement."""
from pathlib import Path
import json,os,signal,subprocess,sys
sys.path.insert(0,'/lab/source')
import session_checkpoint as checkpoint
import authority_inspection as inspection
import runtime_authority as authority
import journal as j
from journal_runtime_fixture import ROOT,guard,read,save,sha


def worker():
 guard();control=read(ROOT/'checkpoint-control.json');store=j.Store(ROOT/'collection-journal')
 try:
  with store.lease() as lease:
   if control['fault']=='before-publication':
    def interrupted(*args):os.kill(os.getpid(),signal.SIGKILL)
    checkpoint.provisioning.rename_no_replace=interrupted
   record=checkpoint.capture(ROOT/'runtime'/authority.ANCHOR,store,lease,control['recovery'])
   if control['fault']=='after-publication':os.kill(os.getpid(),signal.SIGKILL)
   print(json.dumps({'id':record['id'],'sha256':record['sha256'],'activation_authorized':False}))
 finally:store.close()


def child(recovery,fault='',expected=0):
 save(ROOT/'checkpoint-control.json',{'recovery':recovery,'fault':fault})
 q=subprocess.run([sys.executable,'-B',__file__,'--worker'],capture_output=True,text=True,timeout=20)
 with (ROOT/'checkpoint-worker.log').open('a') as stream:stream.write(json.dumps({'fault':fault,'exit':q.returncode})+'\n'+q.stdout+q.stderr)
 assert q.returncode==expected,(q.returncode,q.stderr[-1800:]);return q


def capture(recovery):
 guard();path=ROOT/'collection-journal'/checkpoint.filename(recovery)
 child(recovery,'before-publication',-signal.SIGKILL);assert not path.exists()
 stages=list(path.parent.glob('.pending-desktop-checkpoint-*'));assert len(stages)==1;stage=stages[0]
 child(recovery,'after-publication',-signal.SIGKILL);before={'inode':path.stat().st_ino,'sha256':sha(path)}
 child(recovery);assert {'inode':path.stat().st_ino,'sha256':sha(path)}==before
 value=read(path);checkpoint.validate(value)
 assert sha(stage)==before['sha256']
 save(ROOT/'session-checkpoint-result.json',{'passed':True,'status':'captured','recovery':recovery,'controller_kills':2,
  'checkpoint':before,'content_sha256':value['sha256'],'files':len(value['files']),'source_bytes':sum(len(x['text'].encode()) for x in value['files'].values()),
  'unpublished_stage':{'name':stage.name,'inode':stage.stat().st_ino,'sha256':sha(stage)},'original_root':value['inspection']['descriptor']['root']['path'],
  'completed_capture_retry_unchanged':True,'activation_authorized':False})


def retire_runtime(display):
 guard();report=read(ROOT/'session-checkpoint-result.json');assert report['status']=='captured'
 display.close();assert read(ROOT/'compositor-exit.json')['exit']==0
 # The fixture's probe and keyring have already closed before this hook. Keep
 # every old file for inspection, while making its original runtime path absent.
 old=ROOT/'runtime';retained=ROOT/'retained-runtime';assert not retained.exists()
 old.rename(retained);old.mkdir(mode=0o700)
 store=j.Store(ROOT/'collection-journal');path=store.root/checkpoint.filename(report['recovery'])
 try:
  before=sha(path)
  with store.lease() as lease:record=checkpoint.read(store,report['recovery'],lease)
  assert record['sha256']==report['content_sha256'] and sha(path)==before
  for inspect_original in (lambda:authority.Authority(old/authority.ANCHOR),lambda:inspection.inspect(old/authority.ANCHOR,store.root)):
   try:inspect_original()
   except OSError:pass
   else:raise AssertionError('retired runtime unexpectedly opened as active authority')
  child(report['recovery']);assert sha(path)==before and path.stat().st_ino==report['checkpoint']['inode']
  assert list(old.iterdir())==[]
  if (ROOT/'successor-inspection-mode.json').exists():
   from native_successor_inspection_fixture import observe
   result=observe('predecessor-runtime-retired');assert result['context']['server']=='gone' and result['activations'][-1]['closure']['verified_process_closure']
  report.update({'status':'runtime-retired','checkpoint_read_after_runtime_loss':True,'new_controller_retry_after_runtime_loss_unchanged':True,
   'original_live_authority_and_inspector_refused':True,'original_runtime_retained_at':str(retained),'replacement_runtime_empty':True,
   'private_keyring_stopped_before_retirement':True,'successor_activation_implemented':(ROOT/'successor-activation-mode.json').exists()})
  save(ROOT/'session-checkpoint-result.json',report)
  print('PASS: immutable checkpoint survives two publication kills and retired original runtime; no authority recreated')
  if (ROOT/'successor-checkpoint-mode.json').exists():
   from native_successor_checkpoint_fixture import run as linked_checkpoint
   linked_checkpoint()
 finally:store.close()

if __name__=='__main__':worker()
