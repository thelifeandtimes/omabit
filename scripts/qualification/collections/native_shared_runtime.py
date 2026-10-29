"""Private connected format3 handoff adapter; no installed-profile commands."""
from contextlib import contextmanager
from pathlib import Path
import json,os,signal,sys
sys.path.insert(0,'/lab/source')
import admission as original
import shared_admission as admission
import shared_artifacts as artifacts
import runtime_authority as authority
import journal as j
import coordinator as c
from journal_runtime_fixture import ROOT,guard,read
from native_recovery_runtime import NativeAdapter as Base,NativeRecoveryAdapter

SHARED=True
PROTOCOL_MODE=(ROOT/'runtime-protocol-mode.json').exists()
if PROTOCOL_MODE:
 import recovery_artifacts as artifacts
 import runtime_preflight
 if (ROOT/'successor-activation-mode.json').exists():
  import successor_artifacts as artifacts
  import successor_preflight as runtime_preflight
 if (ROOT/'continuation-activation-mode.json').exists():
  import continuation_artifacts as artifacts
  import continuation_preflight as runtime_preflight
 if (ROOT/'repeated-sessions-mode.json').exists():
  import repeated_artifacts as artifacts
  import repeated_preflight as runtime_preflight


class NativeAdapter(Base):
 artifact_module=artifacts
 admission_module=admission
 def __init__(self,store=None,lease=None):
  super().__init__();self.store=store;self.lease=lease
 @contextmanager
 def controller(self):
  if self.store is not None:
   yield self.store,self.lease
  else:
   store=j.Store(ROOT/'collection-journal')
   try:
    with store.lease() as lease:yield store,lease
   finally:store.close()
 def preflight(self,spec):
  if not PROTOCOL_MODE:return None
  return runtime_preflight.require_ready(spec,{role:str(self.artifact_path(spec,role)) for role in j.ROLES},dict(runtime_preflight.REQUIRED,authority=1))
 def perform(self,phase,request,spec,role):
  # Compatibility gates forward replacement, never the ability to clean up an
  # already started candidate during rollback.
  if PROTOCOL_MODE and phase=='drain-previous':
   selected={j.digest(value):value for value in self.input['specs'].values()}
   for key in (request['previous']['collection'],request['candidate']):self.preflight(selected[key])
  return super().perform(phase,request,spec,role)
 def guardian_receipt(self):return ROOT/'receipts'/(self.input['supervisor']['token']+'.json')
 def issue(self,spec,epoch,request):
  self.preflight(spec)
  path=ROOT/'gates'/epoch;path.mkdir(mode=0o700,exist_ok=True)
  permit={'id':epoch,'request':request,'epoch':epoch,'specification':spec,'shell':original.identity(self.pid)}
  with self.controller() as (store,lease):
   a=authority.Authority(ROOT/'runtime'/authority.ANCHOR)
   try:
    fn=a.reserve if store.status()['committed'] is None else a.handoff
    ticket=fn(store,lease,path,permit,guardian_receipt=self.guardian_receipt())
   finally:a.close()
  if read(ROOT/'coordinator-fault.json')=={'phase':'after-handoff','role':None}:
   from journal_runtime_fixture import save,sha
   save(ROOT/'lost-handoff.json',{'ticket':ticket,'gate':str(path),'reservation_sha256':sha(ROOT/'runtime'/authority.ANCHOR/'reservation.json')})
   os.kill(os.getpid(),signal.SIGKILL)
  gate=self.gate(path)
  try:gate.issue(permit,ticket)
  finally:gate.close()
  if read(ROOT/'coordinator-fault.json')=={'phase':'after-initial-issue','role':None}:
   assert spec['name']=='stable'
   os.kill(os.getpid(),signal.SIGKILL)
  activation={'root':str(path),'permit':permit['id'],'epoch':epoch,'collection':j.digest(spec),'authority':ticket}
  return path,permit,activation
 def partial_roles(self,spec):
  fault=read(ROOT/'coordinator-fault.json')
  if fault.get('role') is not None:return ()
  if spec['name']=='stable':
   return {'initial-partial-dms':('dms',),'initial-partial-theme':('theme',),'initial-partial-omabit':('omabit',)}.get(fault.get('phase'),())
  if spec['name']!='dev':return ()
  return {'partial-dms':('dms',),'partial-theme':('theme',),'partial-all':j.ROLES}.get(fault.get('phase'),())
 def set_enabled(self,spec,enabled):
  roles=self.partial_roles(spec) if enabled else ()
  if roles:
   for role in roles:assert self.rpc('shell','setPluginEnabled',spec['members'][role]['plugin_id'],'true')=='ok'
  else:super().set_enabled(spec,enabled)
 def remember(self,spec,epoch):
  roles=self.partial_roles(spec)
  if roles:
   from journal_runtime_fixture import save
   self.wait(lambda:set(self.bindings())=={spec['members'][r]['plugin_id'] for r in roles})
   def partial_ready():
    connections=self.connections();channels=self.channels();hosts=self.bindings()
    if not all(hosts[spec['members'][r]['plugin_id']]['phase']=='loaded' for r in roles):return False
    for role,count in (('theme',1),('dms',4)):
     if role not in roles:continue
     if not any(x['owner']==spec['name']+'-'+role and x['active_streams'] and x['subscriptions']==count and x['delivered']>=count for x in channels):return False
     body=connections.get(role,{})
     if role=='dms' and (body.get('ship')!='~zod' or not body.get('outboundReadable')):return False
     if role=='theme' and (not body.get('loaded') or body.get('desktop')!='Following shared appearance' or body.get('error') or body.get('bridgeBusy')):return False
    return True
   self.wait(partial_ready,45)
   observed={}
   for role in roles:
    member=spec['members'][role];host=self.bindings()[member['plugin_id']]
    observed[role]=artifacts.observation(self.artifact_path(spec,role),member['artifact_sha256'],host['loaded'])
   if 'theme' in roles:
    from appearance_fixture import profile
    desktop=read(ROOT/'desktop.json');expected=profile(spec['name'])
    assert desktop['colors']==expected['colors'] and desktop['font']==expected['font'] and desktop['rounding']==expected['windows']['decoration:rounding']['int']
   value=read(ROOT/'runtime'/authority.ANCHOR/'reservation.json')
   save(ROOT/'partial-loaded.json',{'reservation':j.digest(value['reservation']),'bindings':observed,'appearance_applied':'theme' in roles})
   os.kill(os.getpid(),signal.SIGKILL)
  return super().remember(spec,epoch)
 def bind(self,store,lease,epoch):
  a=authority.Authority(ROOT/'runtime'/authority.ANCHOR)
  try:
   value=a.status();ticket=a._ticket(value['reservation'])
   return a.bind_loaded(store,lease,ticket,self.bound(epoch),self.guardian_receipt())
  finally:a.close()
 def healthy(self,spec,loaded):
  with self.controller() as (store,lease):
   # Initial adoption records readiness first, then binds it explicitly. During
   # transitions the enable receipt already names the exact loaded instances.
   state=store.status()
   if state['committed'] is not None:self.bind(store,lease,loaded['epoch'])
  return super().healthy(spec,loaded)

def initialize(store):
 if (ROOT/'provisioning-mode.json').exists():
  from native_provisioning_fixture import initialize as provision
  return provision(store)
 with store.lease() as lease:
  a=authority.Authority.provision(ROOT/'runtime'/os.environ['WAYLAND_DISPLAY'],store,lease)
  a.close()

def adopted(adapter,store):
 loaded=store.status()['committed'];spec=store.status()['collections'][loaded['collection']]
 for member in spec['members'].values():
  assert adapter.rpc(member['plugin_id']+'.collection-host','releaseAdmission')=='busy'
 gate=adapter.gate(ROOT/'gates'/loaded['epoch'])
 try:assert all(row['state']=='admitted' for row in gate.status()['record']['claims'].values())
 finally:gate.close()
 with store.lease() as lease:adapter.bind(store,lease,loaded['epoch'])

def before_transition(adapter,store,probe,count_apps):
 if not PROTOCOL_MODE:return
 import copy,uuid
 from journal_runtime_fixture import save,sha
 stable=adapter.input['specs']['stable'];old=copy.deepcopy(stable)
 old['members']['theme']['artifact_sha256']=read(ROOT/'legacy-preflight.json')['artifact_sha256']
 assert adapter.verify_artifact(old,'theme')['descriptor']['format']==3
 adapter.input['specs']['incompatible-preflight']=old
 def snapshot():
  active=store.status()['committed'];gate=ROOT/'gates'/active['epoch']
  return {'journal':sha(ROOT/'collection-journal/journal.json'),
   'authority':sha(ROOT/'runtime'/authority.ANCHOR/'reservation.json'),
   'gate':{p.name:sha(p) for p in gate.iterdir() if p.is_file()},
   'gates':sorted(p.name for p in (ROOT/'gates').iterdir()),
   'manifests':{r:sha(adapter.artifact_path(stable,r).parent.parent/'manifest.json') for r in j.ROLES},
   'applications':count_apps(),'active_channels':sorted(x['path'] for x in probe.hub.active()),
   'deletions':sum(x['action']=='delete' for x in probe.hub.requests)}
 before=snapshot();blocked=[]
 for operation in ('enable','drain'):
  try:
   if operation=='enable':adapter.enable(old,uuid.uuid4().hex,uuid.uuid4().hex)
   else:adapter.perform('drain-previous',{'id':uuid.uuid4().hex,'previous':store.status()['committed'],'candidate':j.digest(old)},stable,'omabit')
  except j.Conflict as error:
   assert 'collection runtime compatibility refused: theme:runtime-protocol-undeclared' in str(error);blocked.append(operation)
  else:raise AssertionError('mixed runtime contract was allowed to change the live collection')
  assert snapshot()==before
 del adapter.input['specs']['incompatible-preflight']
 reports={name:adapter.preflight(spec) for name,spec in adapter.input['specs'].items()}
 assert all(x['ready'] for x in reports.values())
 save(ROOT/'runtime-preflight-result.json',{'passed':True,'blocked_operations':blocked,'unchanged_live_collection':before,'profiles':reports,'historical_generation_unchanged':True})

def before_enable(worker,count_apps):
 from journal_runtime_fixture import save,sha
 before=count_apps();save(ROOT/'coordinator-fault.json',{'phase':'after-handoff','role':None})
 worker(expected=-signal.SIGKILL)
 proof=read(ROOT/'lost-handoff.json');assert count_apps()==before
 assert not list(Path(proof['gate']).iterdir()),'handoff must precede even gate issuance'
 assert sha(ROOT/'runtime'/authority.ANCHOR/'reservation.json')==proof['reservation_sha256']

def after_enable():
 from journal_runtime_fixture import sha
 proof=read(ROOT/'lost-handoff.json')
 assert sha(ROOT/'runtime'/authority.ANCHOR/'reservation.json')==proof['reservation_sha256']
 assert read(Path(proof['gate'])/'admission.json')['authority']==proof['ticket']

def verify_result():
 a=authority.Authority(ROOT/'runtime'/authority.ANCHOR)
 try:state=a.status()
 finally:a.close()
 assert state['format']==2 and len(state['history'])==2
 assert [row['handoff']['mode'] for row in state['history']]==['loaded-recovery','loaded-clean']
 assert all(row['binding'] and len(row['binding']['claims'])==3 for row in state['history'])
 assert state['binding'] and state['binding']['loaded']['collection']==j.digest(state['reservation']['permit']['specification'])
 from journal_runtime_fixture import save
 save(ROOT/'shared-handoff-result.json',{'passed':True,'reservation_format':2,'artifact_format':4 if PROTOCOL_MODE else 3,'admission_format':2,
  'history':state['history'],'current':state['binding'],'actual_plugins':True,'synthetic_eyre':True,
  'private_wayland':True,'handoff_controller_sigkill_before_issue':True,'handoff_retry_preserved_exact_reservation':True,'full_native_shell':False,'production_adapter':False})

def worker(recovery=False,candidate_recovery=False):
 guard();store=j.Store(ROOT/'collection-journal')
 try:
  with store.lease() as lease:
   adapter=NativeAdapter(store,lease)
   if candidate_recovery:
    from native_candidate_recovery import CandidateRecovery
    control=read(ROOT/'recovery-control.json');request=store.status()['requests'][control['request']]
    with CandidateRecovery(request,control,adapter) as recovery_adapter:
     result=c.recover_candidate_step(store,lease,recovery_adapter,control['request'],control['recovery'])
     if control['fault']=='after-prepared':os.kill(os.getpid(),signal.SIGKILL)
   elif recovery:
    control=read(ROOT/'recovery-control.json');request=store.status()['requests'][control['request']]
    with NativeRecoveryAdapter.create(request,control,runtime=adapter) as recovery_adapter:
     result=c.recover_step(store,lease,recovery_adapter,control['request'],control['recovery'])
     if control['fault']=='after-prepared':os.kill(os.getpid(),signal.SIGKILL)
    request=result['requests'][control['request']]
    if request['phase']=='disable-previous' and request['recovery']['verified'] is not None:
     a=authority.Authority(ROOT/'runtime'/authority.ANCHOR)
     try:
      if a.status()['binding'] is None:a.bind_closed_initial(store,lease,request['id'],adapter.bound(request['previous']['epoch']))
     finally:a.close()
     if control['fault']=='after-binding':os.kill(os.getpid(),signal.SIGKILL)
   else:result=c.step(store,lease,adapter)
  print(json.dumps({'pending':result['pending']}))
 finally:store.close()

if __name__=='__main__':worker('--recover' in sys.argv,'--recover-candidate' in sys.argv)
