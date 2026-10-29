"""Private native registry adapter for connected recovery qualification only."""
from pathlib import Path
import json,os,signal,subprocess,sys,time
sys.path.insert(0,'/lab/source')
import admission as admission
import guarded_artifacts as artifacts
import journal as j
import coordinator as c
from journal_runtime_fixture import ROOT,Adapter as PriorAdapter,guard,read,save,sha,children,identity,gone

class NativeAdapter(PriorAdapter):
 artifact_module=artifacts
 admission_module=admission
 def gate(self,path):return self.admission_module.Gate(path)
 def rpc(self,target,method,*args):
  if gone(self.input['shell']):raise c.Problem('reader-owner-lost')
  q=subprocess.run(['/usr/bin/quickshell','ipc','--pid',str(self.pid),'call','--',target,method,*args],capture_output=True,text=True,timeout=5)
  assert q.returncode==0 and q.stdout.strip()!='Target not found.',(target,method,q.stderr[-500:])
  return q.stdout.strip()
 def snapshot(self):return json.loads(self.rpc('artifactProbe','state'))
 def connections(self):return json.loads(self.rpc('connectedProbe','state'))
 def bindings(self):return {h['host']['loaded']['plugin_id']:h['host'] for h in self.snapshot()['loaded'].values() if h['host']['loaded']}
 def artifact_path(self,spec,role):
  m=spec['members'][role]
  return ROOT/'home/.config/omarchy/plugins'/m['plugin_id']/'generations'/m['artifact_sha256']
 def verify_artifact(self,spec,role):
  m=spec['members'][role];v=self.artifact_module.verify(self.artifact_path(spec,role),m['artifact_sha256'])
  assert v['admission_capable'] and v['descriptor']['profile']==m['profile'] and v['descriptor']['role']==role
  return v
 def loaded(self,spec,epoch):
  observed=self.bindings()
  if not observed:return None
  if not set(observed)<={x['plugin_id'] for x in spec['members'].values()}:raise c.Problem('artifact-mismatch')
  members={}
  for role,m in spec['members'].items():
   if m['plugin_id'] not in observed:continue
   host=observed[m['plugin_id']];g=host['grant']
   if not g or g['epoch']!=epoch or g['collection']!=j.digest(spec):raise c.Problem('artifact-mismatch')
   self.verify_artifact(spec,role)
   members[role]=self.artifact_module.observation(self.artifact_path(spec,role),m['artifact_sha256'],host['loaded'])
  if len(members)!=len(j.ROLES):return None
  value={'collection':j.digest(spec),'epoch':epoch,'members':members};j.loaded(value,spec,epoch);return value
 def remember(self,spec,epoch):
  loaded=self.wait(lambda:self.loaded(spec,epoch));path=self.bound(epoch)
  if path.exists():assert read(path)['loaded']==loaded;return loaded
  def helpers():
   rows={r:[] for r in j.ROLES}
   for pid in children(self.pid):
    row=identity(pid)
    if not row or row['state']=='Z':continue
    args=row.pop('args')
    for role in j.ROLES:
     prefix=(str(self.artifact_path(spec,role))+'/').encode()
     if any(a.startswith(prefix) for a in args):rows[role].append(row)
   # Each role has its admission guard and long-lived real helper. Additional
   # collector readers are retained too; the original guardian covers all forks.
   return rows if all(len(rows[r])>=2 for r in j.ROLES) else None
  owned=self.wait(helpers)
  gate=ROOT/'gates'/epoch
  save(path,{'loaded':loaded,'helpers':owned,'shell':self.input['shell'],'supervisor':self.input['supervisor'],'admission':{'root':str(gate),'permit':read(gate/'admission.json')['permit']}})
  return loaded
 def item(self,role):
  rows=self.snapshot()['loaded']
  row=next((v for k,v in rows.items() if v['host']['loaded'] and v['host']['loaded']['plugin_id'] in {s['members'][role]['plugin_id'] for s in self.input['specs'].values()}),None)
  return {'transportState':row['host']['transport']} if row else None
 def healthy(self,spec,loaded):
  assert self.loaded(spec,loaded['epoch'])==loaded
  def ready():
   s=self.connections();t=s.get('theme');d=s.get('dms');rows=self.channels();active=[x for x in rows if x['active_streams']]
   all_hosts=self.snapshot()['loaded']
   if t and t['desktop']=='Needs attention':raise c.Problem('readiness-failed')
   return (t and d and t['loaded'] and t['ships']==1 and t['desktop']=='Following shared appearance' and not t['error'] and not t['bridgeBusy'] and d['ship']=='~zod' and d['outboundReadable']
    and len(all_hosts)==3 and all(v['host']['phase']=='loaded' and v['helper']=='1' for v in all_hosts.values())
    and len(active)==2 and {x['owner'] for x in active}=={spec['name']+'-theme',spec['name']+'-dms'} and all(x['delivered']>=x['subscriptions']>0 and not x['deleted'] for x in active))
  self.wait(ready,45)
  from appearance_fixture import profile
  v=read(ROOT/'desktop.json');expected=profile(spec['name'])
  assert v['font']==expected['font'] and v['colors']==expected['colors'] and v['rounding']==expected['windows']['decoration:rounding']['int']
  value={'loaded':loaded,'ready':True,'endpoints':{r:spec['members'][r]['endpoint_ref'] for r in j.ROLES}};j.ready(value,spec,loaded);return value
 def set_enabled(self,spec,enabled):
  for m in spec['members'].values():assert self.rpc('shell','setPluginEnabled',m['plugin_id'],str(enabled).lower())=='ok'
 def issue(self,spec,epoch,request):
  path=ROOT/'gates'/epoch;path.mkdir(mode=0o700,exist_ok=True);gate=self.gate(path)
  permit={'id':epoch,'request':request,'epoch':epoch,'specification':spec,'shell':admission.identity(self.pid)}
  try:gate.issue(permit)
  finally:gate.close()
  return path,permit,{'root':str(path),'permit':permit['id'],'epoch':epoch,'collection':j.digest(spec)}
 def enable(self,spec,epoch,request):
  for role in j.ROLES:self.verify_artifact(spec,role)
  path,permit,activation=self.issue(spec,epoch,request)
  changed=False
  for role,m in spec['members'].items():
   selected=self.verify_artifact(spec,role)['manifest']
   selected['collectionActivation']=activation
   p=self.artifact_path(spec,role).parent.parent/'manifest.json'
   if read(p)!=selected:save(p,selected);changed=True
  if changed:self.rpc('artifactProbe','rescan');self.wait(lambda:not self.snapshot()['scanning'])
  self.set_enabled(spec,True);return self.remember(spec,epoch)
 def release_guards(self,spec,old):
  hosts=self.snapshot()['loaded']
  if hosts:
   assert self.loaded(spec,old['loaded']['epoch'])==old['loaded']
   for member in spec['members'].values():
    assert self.rpc(member['plugin_id']+'.collection-host','releaseAdmission') in ('ending','ended')
   self.wait(lambda:all(row['host']['admissionEnded'] for row in self.snapshot()['loaded'].values()))
  gate=self.gate(old['admission']['root'])
  try:
   row=gate.status()['record'];assert row['permit']==old['admission']['permit']
   assert set(row['claims'])==set(j.ROLES) and all(claim['state']=='ended' for claim in row['claims'].values())
  finally:gate.close()
 def perform(self,phase,request,spec,role):
  if gone(self.input['shell']):raise c.Problem('reader-owner-lost')
  epoch=request['previous']['epoch'] if phase.endswith('previous') else j.effect(request,'enable-candidate')
  if phase.startswith('drain-'):
   old=read(self.bound(epoch));assert self.loaded(spec,epoch)==old['loaded']
   gate=self.gate(old['admission']['root'])
   try:
    gate.stop_admission(old['admission']['permit']['id'],j.effect(request,phase))
    guard_owner=gate.status()['record']['claims'][role]['owner']
   finally:gate.close()
   def closed():
    v=self.item(role)['transportState']
    if v.get('unconfirmedHelpers'):raise c.Problem('helper-descendants-unconfirmed')
    if v.get('readerOwnerLost'):raise c.Problem('reader-owner-lost')
    return v if v.get('closed') else None
   receipt=self.wait(closed,75)
   # Guards intentionally survive transport drain until native unload. Every
   # non-guard owned helper must exit; unclean closure uses the original subreaper.
   def transport_exited():
    for p in old['helpers'][role]:
     row=identity(p['pid'])
     if row and row['start']==p['start'] and row['state']!='Z' and (p['pid'],p['start'])!=(guard_owner['pid'],guard_owner['start']):return False
    return True
   self.wait(transport_exited)
   if role!='omabit':self.wait(lambda:all(x['deleted'] and not x['active_streams'] for x in self.channels() if x['owner']==spec['name']+'-'+role))
   base=Path(spec['members'][role]['paths']['state']);names={'omabit':[],'theme':['desktop.json'],'dms':['outbound-messages.json','conversation-effects.json']}[role]
   retained={name:sha(base/name) if (base/name).exists() else None for name in names}
   pending=[x['id'] for key in ('unresolvedSends','unresolvedEffects') for x in receipt.get(key,[])]
   value={'epoch':epoch,'binding':old['loaded']['members'][role],'absent':False,'closed':True,'retained_state':j.digest(retained),'pending_ids':pending}
  elif phase.startswith('disable-'):
   old=read(self.bound(epoch))
   if old['shell']!=self.input['shell']:
    verified=request['recovery']['verified'];assert verified and gone(old['shell'])
    proof=read(ROOT/Path(read(ROOT/'recovery-control.json')['closure']['receipt']).relative_to('/lab'))
    assert proof['proof']=='waitpid-echild' and proof['owner']==old['supervisor'] and sha(ROOT/Path(read(ROOT/'recovery-control.json')['closure']['receipt']).relative_to('/lab'))==verified['closure_receipt_sha256']
   else:self.release_guards(spec,old)
   self.set_enabled(spec,False);self.wait(lambda:not self.snapshot()['loaded'])
   self.wait(lambda:all(gone(p) for rows in old['helpers'].values() for p in rows))
   value={'epoch':epoch,'absent':True,'helpers_exited':True}
  elif phase=='checkpoint-appearance':
   assert not self.snapshot()['loaded'];value={'checkpoint':self.local(spec,'checkpoint-appearance')['appearanceCheckpoint']['id']}
  elif phase=='restore-appearance':
   assert not self.snapshot()['loaded'];result=self.local(spec,'restore-appearance',request['checkpoint'],j.effect(request,phase));assert result['appearanceRestoration']['phase']=='verified';value={'checkpoint':request['checkpoint'],'verified':True}
  elif phase.startswith('enable-'):
   value=self.enable(spec,j.effect(request,phase),request['id'])
  else:value=self.healthy(spec,request['loaded'])
  if read(ROOT/'coordinator-fault.json')=={'phase':phase,'role':role}:
   save(ROOT/'lost-reply.json',{'phase':phase,'role':role,'observation':value});os.kill(os.getpid(),signal.SIGKILL)
  return value

class NativeRecoveryAdapter:
 @staticmethod
 def create(request,control,*,runtime=None):
  from recovery_runtime import RecoveryAdapter
  class GuardedRecovery(RecoveryAdapter):
   def verify_owner(self):
    result=super().verify_owner();saved=self.binding['admission'];gate=self.runtime.gate(saved['root'])
    try:
     state=gate.status()['record'];assert state['state']=='closing' and state['permit']==saved['permit']
     assert state['permit']['specification']==self.spec and state['permit']['epoch']==self.request['previous']['epoch']
     assert state['permit']['shell']['pid']==self.closure['root']['pid'] and state['permit']['shell']['start']==self.closure['root']['start']
    finally:gate.close()
    return result
  return GuardedRecovery(request,control,runtime=runtime if runtime is not None else NativeAdapter())

def worker(recovery=False):
 guard();store=j.Store(ROOT/'collection-journal')
 try:
  with store.lease() as lease:
   if recovery:
    control=read(ROOT/'recovery-control.json');request=store.status()['requests'][control['request']]
    with NativeRecoveryAdapter.create(request,control) as adapter:
     result=c.recover_step(store,lease,adapter,control['request'],control['recovery'])
     if control['fault']=='after-prepared':os.kill(os.getpid(),signal.SIGKILL)
   else:result=c.step(store,lease,NativeAdapter())
  print(json.dumps({'pending':result['pending']}))
 finally:store.close()

if __name__=='__main__':worker('--recover' in sys.argv)
