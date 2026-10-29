"""Private linked checkpoint publication and second runtime retirement."""
from pathlib import Path
import json,os,signal,subprocess,sys
sys.path.insert(0,'/lab/source')
import successor_checkpoint as checkpoint
import successor_inspection as inspection
from successor_authority import Authority
import runtime_authority as authority
import journal as j
from journal_runtime_fixture import ROOT,guard,read,save,sha


def worker():
 guard();control=read(ROOT/'successor-checkpoint-control.json');store=j.Store(ROOT/'collection-journal')
 try:
  with store.lease() as lease:
   if control['fault']=='before-publication':
    def interrupted(*args):os.kill(os.getpid(),signal.SIGKILL)
    checkpoint.provisioning.rename_no_replace=interrupted
   record=checkpoint.capture(ROOT/'successor-runtime'/authority.ANCHOR,store,lease,control['recovery'])
   if control['fault']=='after-publication':os.kill(os.getpid(),signal.SIGKILL)
   print(json.dumps({'id':record['id'],'sha256':record['sha256'],'activation_authorized':False}))
 finally:store.close()


def child(recovery,fault='',expected=0):
 save(ROOT/'successor-checkpoint-control.json',{'recovery':recovery,'fault':fault})
 q=subprocess.run([sys.executable,'-B',__file__,'--worker'],capture_output=True,text=True,timeout=20)
 with (ROOT/'successor-checkpoint-worker.log').open('a') as stream:stream.write(json.dumps({'fault':fault,'exit':q.returncode})+'\n'+q.stdout+q.stderr)
 assert q.returncode==expected,(q.returncode,q.stderr[-1800:]);return q


def capture(recovery):
 guard();path=ROOT/'collection-journal'/checkpoint.filename(recovery)
 child(recovery,'before-publication',-signal.SIGKILL);assert not path.exists()
 stages=list(path.parent.glob('.pending-desktop-successor-checkpoint-*'));assert len(stages)==1;stage=stages[0]
 child(recovery,'after-publication',-signal.SIGKILL);before={'inode':path.stat().st_ino,'sha256':sha(path)}
 child(recovery);assert {'inode':path.stat().st_ino,'sha256':sha(path)}==before
 value=read(path);checkpoint.validate(value);assert value['ancestor_checkpoint']['object']['path'] not in value['files']
 assert sha(stage)==before['sha256']
 save(ROOT/'successor-checkpoint-result.json',{'passed':True,'status':'captured','recovery':recovery,'controller_kills':2,
  'checkpoint':before,'content_sha256':value['sha256'],'files':len(value['files']),'source_bytes':sum(len(x['text'].encode()) for x in value['files'].values()),
  'unpublished_stage':{'name':stage.name,'inode':stage.stat().st_ino,'sha256':sha(stage)},'original_root':value['inspection']['descriptor']['root']['path'],
  'completed_capture_retry_unchanged':True,'activation_authorized':False})


def retire_runtime():
 guard();report=read(ROOT/'successor-checkpoint-result.json');assert report['status']=='captured'
 assert read(ROOT/'compositor-successor-exit.json')['exit']==0
 # The fixture's probe and keyring have already closed before this hook. Keep
 # every old file for inspection, while making its original runtime path absent.
 old=ROOT/'successor-runtime';retained=ROOT/'retained-successor-runtime';assert not retained.exists()
 old.rename(retained);old.mkdir(mode=0o700)
 store=j.Store(ROOT/'collection-journal');path=store.root/checkpoint.filename(report['recovery'])
 try:
  before=sha(path)
  with store.lease() as lease:record=checkpoint.read(store,report['recovery'],lease)
  assert record['sha256']==report['content_sha256'] and sha(path)==before
  for inspect_original in (lambda:Authority(old/authority.ANCHOR),lambda:inspection.inspect(old/authority.ANCHOR,store.root)):
   try:inspect_original()
   except OSError:pass
   else:raise AssertionError('retired runtime unexpectedly opened as active authority')
  child(report['recovery']);assert sha(path)==before and path.stat().st_ino==report['checkpoint']['inode']
  assert list(old.iterdir())==[]
  report.update({'status':'runtime-retired','checkpoint_read_after_runtime_loss':True,'new_controller_retry_after_runtime_loss_unchanged':True,
   'original_live_authority_and_inspector_refused':True,'original_runtime_retained_at':str(retained),'replacement_runtime_empty':True,
   'ancestor_embedded':False,'ancestor_checkpoint':record['ancestor_checkpoint'],'private_keyring_stopped_before_retirement':True,'successor_activation_implemented':(ROOT/'successor-activation-mode.json').exists()})
  if (ROOT/'continuation-inspection-mode.json').exists():
   from native_continuation_inspection_fixture import observe
   result=observe('ancestors-runtime-retired');assert result['context']['server']=='gone' and result['activations'][-1]['closure']['verified_process_closure']
  save(ROOT/'successor-checkpoint-result.json',report)
  print('PASS: linked checkpoint survives two publication kills and second runtime retirement; ancestor referenced without copying')
 finally:store.close()

def run():
 guard();report=read(ROOT/'successor-cleanup-result.json');assert report['passed']
 if not (ROOT/'successor-checkpoint-result.json').exists():capture(report['recovery'])
 else:
  prior=read(ROOT/'successor-checkpoint-result.json');assert prior['passed'] and prior['status']=='captured' and prior['recovery']==report['recovery']
 retire_runtime()

if __name__=='__main__':worker()
