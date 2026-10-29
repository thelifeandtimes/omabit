"""Private real-plugin cohort checks for a non-admitting successor claim."""
from pathlib import Path
import contextlib,copy,json,os,signal,subprocess,sys,time,uuid
sys.path.insert(0,'/lab/source')
import journal as j
import runtime_authority as authority
import session_checkpoint as checkpoint
import session_successor as successor
from native_candidate_recovery import CandidateRecovery
from native_shared_runtime import NativeAdapter
from journal_runtime_fixture import ROOT,guard,read,save,sha

class SuccessorCohort(CandidateRecovery):
 def __init__(self,control,runtime,store,lease):
  self.control=control;self.runtime=runtime;self.store=store;self.lease=lease
  self.stack=contextlib.ExitStack();self.roles=set(j.ROLES);self.spec=runtime.input['specs']['stable']
  self.original=None;self.admission=None
 def verify_owner(self):
  archived=checkpoint.read(self.store,self.control['recovery'],self.lease)
  receipt=checkpoint.parsed(archived,archived['cleanup']['object']['path'],reference=archived['cleanup'])
  assert receipt['plan']['specification']==self.spec
  self.loaded=receipt['plan']['loaded'];self.authority_id=receipt['plan']['authority']
  for role in j.ROLES:self.runtime.verify_artifact(self.spec,role)
  return copy.deepcopy(receipt['plan']['guardian'])
 def inspect_successor(self,plan):
  assert plan['specification']==self.spec and plan['loaded']==self.loaded
  value=self.snapshot(closed=True)
  for role in j.ROLES:value[role]['binding']=self.loaded['members'][role]
  if self.control.get('wrong_cohort'):value['dms']['pending_ids']=[]
  return value

class Display:
 def start(self):
  guard();self.root=ROOT/'successor-runtime';self.root.mkdir(mode=0o700)
  env=dict(os.environ,XDG_RUNTIME_DIR=str(self.root),WLR_BACKENDS='headless',WLR_RENDERER='pixman',WLR_HEADLESS_OUTPUTS='1',XKB_CONFIG_ROOT='/tools/usr/share/X11/xkb')
  env.pop('WAYLAND_DISPLAY',None);env.pop('SWAYSOCK',None)
  self.log=(ROOT/'sway-successor.log').open('wb')
  self.proc=subprocess.Popen(['/tools/usr/lib/ld-linux-x86-64.so.2','--library-path','/tools/usr/lib','/tools/usr/bin/sway','--config','/lab/sway.conf'],env=env,stdout=self.log,stderr=subprocess.STDOUT)
  deadline=time.monotonic()+15
  while time.monotonic()<deadline:
   assert self.proc.poll() is None,'private successor compositor exited'
   paths=[p for p in self.root.glob('wayland-*') if p.is_socket()]
   if len(paths)==1:self.path=paths[0];self.context=authority.context(self.path);return
   time.sleep(.05)
  raise AssertionError('private successor compositor deadline')
 def close(self):
  if getattr(self,'proc',None):
   if self.proc.poll() is None:
    self.proc.terminate()
    try:self.proc.wait(timeout=8)
    except subprocess.TimeoutExpired:self.proc.kill();self.proc.wait(timeout=4)
   save(ROOT/'compositor-successor-exit.json',{'pid':self.proc.pid,'exit':self.proc.returncode,'renderer':'pixman','backend':'headless'})
  if getattr(self,'log',None):self.log.close()


def worker():
 guard();control=read(ROOT/'successor-control.json');fault=control['fault'];store=j.Store(ROOT/'collection-journal')
 original=successor.save
 if fault in ('before-intent','after-intent'):
  def interrupted(*args):
   if fault=='after-intent':original(*args)
   os.kill(os.getpid(),signal.SIGKILL)
  successor.save=interrupted
 if fault=='after-claim':
  rename=successor.provisioning.rename_no_replace
  def interrupted(*args):rename(*args);os.kill(os.getpid(),signal.SIGKILL)
  successor.provisioning.rename_no_replace=interrupted
 try:
  with store.lease() as lease:
   runtime=NativeAdapter(store,lease)
   with SuccessorCohort(control,runtime,store,lease) as adapter:
    result=successor.step(control['display'],store,lease,adapter,control['recovery'],control['attempt'])
  print(json.dumps({'phase':result['phase'],'authority':result['intent']['authority'],'activation_authorized':False}))
 finally:store.close()


def run(recovery,probe):
 guard();attempt=uuid.uuid4().hex;display=Display();report={'passed':False,'status':'starting'}
 try:
  display.start();path=ROOT/'collection-journal'/successor.filename(attempt);claim=display.root/authority.PROVISION_CLAIM
  old=read(ROOT/'collection-journal'/checkpoint.filename(recovery));canonical=sha(ROOT/'collection-journal/journal.json')
  original=sha(ROOT/'collection-journal'/authority.MIRROR);requests=len(probe.hub.requests);cohort=copy.deepcopy(probe.hub.snapshot())
  def child(fault='',wrong=False,expected=0):
   save(ROOT/'successor-control.json',{'fault':fault,'wrong_cohort':wrong,'display':str(display.path),'recovery':recovery,'attempt':attempt})
   q=subprocess.run([sys.executable,'-B',__file__,'--worker'],capture_output=True,text=True,timeout=35)
   with (ROOT/'successor-worker.log').open('a') as stream:stream.write(json.dumps({'fault':fault,'wrong_cohort':wrong,'exit':q.returncode})+'\n'+q.stdout+q.stderr)
   assert q.returncode==expected,(q.returncode,q.stderr[-2200:]);probe.check()
   assert len(probe.hub.requests)==requests and probe.hub.snapshot()==cohort
  child(wrong=True,expected=1);assert not path.exists() and not list(display.root.glob('.pending-desktop-successor-*'))
  child('before-intent',expected=-signal.SIGKILL);assert not path.exists() and not claim.exists()
  stages=list(display.root.glob('.pending-desktop-successor-*'));assert len(stages)==1;abandoned=stages[0];abandoned_hash=sha(abandoned)
  child('after-intent',expected=-signal.SIGKILL);intent_hash=sha(path);intent_inode=path.stat().st_ino;assert not claim.exists()
  child('after-claim',expected=-signal.SIGKILL);claim_hash=sha(claim);claim_inode=claim.stat().st_ino
  child();assert sha(path)==intent_hash and path.stat().st_ino==intent_inode and sha(claim)==claim_hash and claim.stat().st_ino==claim_inode
  child();assert sha(path)==intent_hash and path.stat().st_ino==intent_inode and sha(claim)==claim_hash and claim.stat().st_ino==claim_inode
  value=successor.shape(read(path));assert value['claim']['target']==authority.object_identity(claim,'file') and value['predecessor']==old['inspection']['authority']
  assert sha(abandoned)==abandoned_hash and sha(ROOT/'collection-journal/journal.json')==canonical and sha(ROOT/'collection-journal'/authority.MIRROR)==original
  assert not (display.root/authority.ANCHOR).exists() and not (display.root/authority.ANCHOR_SEAL).exists()
  assert probe.credentials()==probe.initial_credentials and sha(ROOT/'state/omarchy-urbit-dms.qualification-stable/outbound-messages.json')==probe.pending['sha256']
  assert sum(x['action']=='post' for x in probe.hub.requests)==probe.initial_logins
  assert (ROOT/'shell-0.log').read_text().count('APPLICATION_CREATED ')==3 and 'APPLICATION_CREATED ' not in (ROOT/'shell-1.log').read_text()
  report={'passed':True,'status':'claimed-no-admission','recovery':recovery,'attempt':attempt,'authority':value['authority'],
   'controller_kills':3,'intent':{'inode':intent_inode,'sha256':intent_hash},'claim':{'inode':claim_inode,'sha256':claim_hash},
   'context':display.context,'wrong_cohort_refused_before_staging':True,'retry_preserved_exact_objects':True,'no_http_requests':True,
   'canonical_journal_sha256':canonical,'original_mirror_sha256':original,'credential_fingerprints':probe.initial_credentials,
   'sessions_preserved':6,'unknown_dm_preserved':True,'pending':probe.pending,'channels':cohort,'application_creations':3,
   'abandoned_stage':{'name':abandoned.name,'sha256':abandoned_hash},'original_checkpoint_unchanged':sha(ROOT/'collection-journal'/checkpoint.filename(recovery))==value['checkpoint']['sha256'],
   'actual_plugin_cohort_verified':True,'successor_activation_implemented':False}
  if (ROOT/'successor-activation-mode.json').exists():
   from native_successor_activation_fixture import run as activate
   report['activation']=activate(recovery,attempt,display,probe);report['successor_activation_implemented']=True;report['status']='activated-and-closed'
  else:
   print('PASS: actual Theme/DMs closed cohorts reserve one successor through three controller kills; no login, HTTP, message replay or application activation')
 finally:
  display.close()
  if report['passed'] and (ROOT/'successor-inspection-mode.json').exists():
   from native_successor_inspection_fixture import observe
   result=observe('compositor-closed');assert result['context']['server']=='gone' and result['activations'][-1]['closure']['verified_process_closure']
  save(ROOT/'session-successor-result.json',report)
 if report['passed']:assert read(ROOT/'compositor-successor-exit.json')['exit']==0

if __name__=='__main__':worker()
