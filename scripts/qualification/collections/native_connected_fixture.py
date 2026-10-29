"""Connected native-host qualification; synthetic Eyre and private keyring only."""
from pathlib import Path
import hashlib,json,os,subprocess,sys,time,traceback

def read(p):return json.loads(p.read_text())
def save(p,v):p.write_text(json.dumps(v,indent=2)+'\n')
def sha(p):return hashlib.sha256(p.read_bytes()).hexdigest()

class Probe:
 def __init__(self,root):
  self.root=root;self.keyring=None;self.hub=None;self.ready=[];self.pending=None;self.last_observation=None
  save(root/'connected-result.json',{'passed':False,'status':'starting'})
 def begin(self,root,prepared):
  # Called only after the native fixture's HOME, socket and namespace guards.
  from appearance_fixture import prepare,AppearanceHub
  from connected_fixture import begin
  prepare(root);self.keyring,self.hub=begin(root,AppearanceHub)
  self.initial_credentials=self.credentials()
  self.initial_logins=sum(x['action']=='post' for x in self.hub.requests)
  assert not self.hub.active()
 def check(self):
  if self.hub:assert not self.hub.errors,'synthetic HTTP worker failed'
 def credentials(self):
  result={}
  for profile in ('stable','dev','edge'):
   for role in ('theme','dms'):
    path=self.root/'artifacts'/profile/role
    code='import sys,json,hashlib;sys.path.insert(0,sys.argv[1]);'
    code+=('from client.support import Keyring;session=Keyring().lookup(sys.argv[2],"~zod",check_expiry=False);' if role=='theme'
           else 'from session_store import SessionStore;session=SessionStore().lookup(sys.argv[2],"~zod",check_expiry=False);')
    code+='assert session;print(hashlib.sha256(json.dumps(session,sort_keys=True).encode()).hexdigest())'
    q=subprocess.run([sys.executable,'-B','-c',code,str(path),self.hub.url],capture_output=True,text=True,timeout=5)
    assert q.returncode==0,'private credential fingerprint unavailable'
    value=q.stdout.strip();assert len(value)==64;result[profile+'-'+role]=value
  return result
 def wait(self,proc,fn,seconds=45):
  end=time.monotonic()+seconds
  while time.monotonic()<end:
   self.check();assert proc.poll() is None
   value=fn()
   if value:return value
   time.sleep(.05)
  save(self.root/'connected-failure.json',{'observation':self.last_observation,'channels':self.hub.snapshot(),'requests':self.hub.requests,'http_worker_errors':self.hub.errors})
  raise AssertionError('connected native observation deadline')
 def admitted(self,profile,proc,call):
  def snapshot():
   self.last_observation=json.loads(call(proc,'connectedProbe','state'));return self.last_observation
  def healthy():
   v=snapshot();t=v.get('theme');d=v.get('dms');rows=self.hub.active()
   return v if (t and d and t['loaded'] and t['ships']==1 and t['desktop']=='Following shared appearance'
    and not t['error'] and not t['bridgeBusy'] and d['ship']=='~zod' and d['outboundReadable'] and len(rows)==2
    and {x['owner'] for x in rows}=={profile+'-theme',profile+'-dms'}
    and all(x['delivered']>=x['subscriptions']>0 for x in rows)) else None
  value=self.wait(proc,healthy)
  self.ready.append({'profile':profile,'state':value,'channels':self.hub.active()})
  if len(self.ready)==1 and self.pending is None:self.seed_pending(proc,call)
  if len(self.ready)==3:
   assert profile=='stable' and len(value['dms']['unresolved'])==1
   path=self.root/'state/omarchy-urbit-dms.qualification-stable/outbound-messages.json'
   assert sha(path)==self.pending['sha256'],'unresolved DM changed on return'
   assert len([x for x in self.hub.pokes if x.get('uncertain') and x['owner']=='stable-dms'])==1
  save(self.root/'connected-readiness.json',self.ready)
 def seed_pending(self,proc,call):
  assert self.pending is None
  def snapshot():return json.loads(call(proc,'connectedProbe','state'))
  self.hub.uncertain_mode=True
  assert call(proc,'connectedProbe','send')==''
  self.wait(proc,lambda:len(snapshot()['dms']['unresolved'])==1 and not snapshot()['dms']['busy'])
  path=self.root/'state/omarchy-urbit-dms.qualification-stable/outbound-messages.json'
  rows=read(path)['messages'];assert len(rows)==1 and rows[0]['phase']=='uncertain'
  self.pending={'sha256':sha(path),'ids':[x['id'] for x in rows]}
  save(self.root/'connected-pending.json',self.pending)
 def no_admission(self):
  end=time.monotonic()+5
  while self.hub.active() and time.monotonic()<end:self.check();time.sleep(.025)
  self.check();assert not self.hub.active() and all(x['deleted'] for x in self.hub.snapshot())
 def finish(self,*,unclean_recovery=False,coordinator_recovery_integrated=False):
  self.no_admission();assert [x['profile'] for x in self.ready]==['stable','dev','stable']
  assert all(x['deleted'] and not x['active_streams'] for x in self.hub.snapshot())
  assert self.credentials()==self.initial_credentials
  assert sum(x['action']=='post' for x in self.hub.requests)==self.initial_logins
  assert len([x for x in self.hub.pokes if x.get('uncertain') and x['owner']=='stable-dms'])==1
  save(self.root/'connected-result.json',{'passed':True,'profiles':['stable','dev','stable'],'positive_subscription_cycles':3,
   'unresolved_dm_preserved_without_replay':True,'pending':self.pending,'credentials_unchanged':True,
   'credential_fingerprints':self.initial_credentials,'no_new_logins':True,'channels':self.hub.snapshot(),
   'pokes':self.hub.pokes,'http_worker_errors':self.hub.errors,'synthetic_eyre':True,'actual_private_keyring':True,'unclean_recovery':unclean_recovery,
   'coordinator_recovery_integrated':coordinator_recovery_integrated})
 def close(self):
  if self.hub:self.hub.close()
  if self.keyring and self.keyring.poll() is None:
   self.keyring.terminate()
   try:self.keyring.wait(timeout=3)
   except subprocess.TimeoutExpired:self.keyring.kill();self.keyring.wait(timeout=3)

def run(common):
 probe=Probe(Path('/lab'))
 try:
  common.inside(hooks=probe);probe.finish()
 except BaseException:
  save(probe.root/'connected-result.json',{'passed':False,'status':'failed','failure':traceback.format_exc(),'observation':probe.last_observation,'channels':probe.hub.snapshot() if probe.hub else []})
  raise
 finally:probe.close()
