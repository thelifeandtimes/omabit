"""Private recovery for explicitly characterized candidate startup cohorts.

The original complete admission record must match the fixture-selected DMs-only,
Theme-only or full supported set. Missing/extra claims refuse. Every selected
namespace remains in the frozen application/session/channel cohort.
"""
from pathlib import Path
import contextlib,copy,fcntl,json,os,signal,sys
sys.path.insert(0,'/lab/source')
import journal as j
import runtime_authority as authority
from journal_runtime_fixture import ROOT,guard,read,sha,save,gone

class CandidateRecovery:
 def __init__(self,request,control,runtime):
  guard();self.request=request;self.control=control;self.runtime=runtime;self.stack=contextlib.ExitStack()
  self.spec=runtime.input['specs']['dev'];assert j.digest(self.spec)==request['candidate'] and request['loaded'] is None
  assert request['phase']=='drain-candidate' and request['uncertain'] and request['direction']=='rollback'
  self.closure=control['closure'];self.original=None;self.admission=None
  case=read(ROOT/'partial-candidate-mode.json')['case']
  self.roles={'dms-cleanup-refused':{'dms'},'theme-cleanup-refused':{'theme'},'all-cleanup-refused':set(j.ROLES)}[case]
 def verify_owner(self):
  a=authority.Authority(ROOT/'runtime'/authority.ANCHOR)
  try:
   state=a.status();assert state['binding'] is None
   original=state['reservation'];permit=original['permit']
   assert permit['request']==self.request['id'] and permit['epoch']==j.effect(self.request,'enable-candidate') and permit['specification']==self.spec
   closure=a._closure_owner(original['owner'])
   assert closure=={'path':self.closure['receipt'],'sha256':self.closure['receipt_sha256'],'request':self.closure['request']}
   proof=read(Path(closure['path']));assert proof['owner']==self.closure['owner'] and proof['root']==self.closure['root']
   assert gone(proof['owner']) and gone(proof['root']) and not self.runtime.bindings()
   row=a._gate(original);assert set(row['claims'])==self.roles
   if self.original is None:self.original=original;self.admission=row;self.authority_id=a.descriptor['id']
   else:assert self.original==original and self.admission==row and self.authority_id==a.descriptor['id']
   for role in j.ROLES:self.runtime.verify_artifact(self.spec,role)
   return copy.deepcopy(original['owner']['guardian'])
  finally:a.close()
 def __enter__(self):
  try:
   self.guardian=self.verify_owner()
   sys.path.insert(0,str(self.runtime.artifact_path(self.spec,'theme')))
   from client.support import StateStore,Keyring
   from client.transport_journal import TransportJournal as ThemeTransport,session_ref as theme_ref
   self.theme_store=StateStore();assert str(self.theme_store.root)==self.spec['members']['theme']['paths']['state']
   self.stack.enter_context(self.theme_store.locked());rows=self.theme_store.load()['ships'];assert len(rows)==1
   self.theme_row=rows[0];self.keyring=Keyring();self.theme_ref=theme_ref
   self.theme=ThemeTransport(self.theme_store.root)
   value=self.theme.status();self.theme_status=value
   if 'theme' in self.roles:
    from client.eyre import Eyre
    self.theme_session=self.keyring.lookup(self.theme_row['url'],self.theme_row['ship'],check_expiry=False);assert self.theme_session
    assert value and value['channels']
    with self.theme.transaction():fd=self.theme.origin_lock(self.theme_session['url'])
    self.stack.callback(os.close,fd);fcntl.flock(fd,fcntl.LOCK_EX|fcntl.LOCK_NB)
    assert all(self.theme.recoverable(row) for row in value['channels'])
    self.theme_client=Eyre(self.theme_session['url'],self.theme_session,journal=self.theme)
   else:assert value is None or not value['channels']
   sys.path.insert(0,str(self.runtime.artifact_path(self.spec,'dms')))
   from session_store import SessionStore,cookie_for
   from transport_journal import session_ref
   from bridge import Bridge
   self.dms_store=SessionStore();assert str(self.dms_store.root)==self.spec['members']['dms']['paths']['state']
   self.dms=Bridge(session_store=self.dms_store)
   if 'dms' in self.roles:
    self.dms.ensure_transport_owner();self.stack.callback(self.dms.release_transport_owner)
   self.stack.callback(self.dms.pool.shutdown,wait=True);self.stack.enter_context(self.dms_store.locked())
   account=self.dms_store.account();self.dms_session=self.dms_store.lookup(account['url'],account['ship'],check_expiry=False)
   assert self.dms_session and self.dms_session['url']==self.theme_row['url']
   self.dms.transport_session=self.dms_session;self.dms.cookies.set_cookie(cookie_for(self.dms_session));self.dms_ref=session_ref
   value=self.dms.transport.status();self.dms_status=value
   if 'dms' in self.roles:assert value and value['channels']
   else:assert value is None or not value['channels']
   return self
  except BaseException:self.stack.close();raise
 def __exit__(self,*args):return self.stack.__exit__(*args)
 def snapshot(self,closed=False):
  self.verify_owner();theme=self.theme.status();dms=self.dms.transport.status();members={}
  if 'theme' not in self.roles:assert theme==self.theme_status
  if 'dms' not in self.roles:assert dms==self.dms_status
  theme_session=self.keyring.lookup(self.theme_row['url'],self.theme_row['ship'],check_expiry=False);assert theme_session
  dms_session=self.dms_store.lookup(self.dms_session['url'],self.dms_session['ship'],check_expiry=False);assert dms_session
  references={'omabit':[],'theme':[self.theme_ref(theme_session)],'dms':[self.dms_ref(dms_session)]}
  for role in j.ROLES:
   base=Path(self.spec['members'][role]['paths']['state']);transport={'dms':dms,'theme':theme,'omabit':None}[role] or {'channels':[]}
   names={'omabit':[],'theme':['state.json','desktop.json'],'dms':['account.json','outbound-messages.json','conversation-effects.json']}[role]
   files={name:sha(base/name) if (base/name).exists() else None for name in names}
   immutable=[];pending=[]
   for row in transport['channels']:
    fields=('id','origin','ship','session_ref','channel','owner')+(('kind','intent','outcome') if role=='theme' else ())
    immutable.append({k:row[k] for k in fields})
    if role=='theme' and row['kind']=='poke' and row['outcome']=='uncertain':pending.append(row['id'])
    if closed:assert row['phase']=='closed'
   if role=='dms':
    # Account admission is already held across snapshot and cleanup. Use the
    # validated readers under that lock, rather than acquiring it a second time.
    assert self.dms.outbound.store.root==self.dms_store.root and self.dms.effects.store.root==self.dms_store.root
    pending=[row['id'] for row in self.dms.outbound._read()+self.dms.effects._read() if row['phase']=='uncertain']
   value={'binding':None,'application_sha256':j.digest({'files':files,'transport_bindings':immutable,'sessions':transport.get('sessions',[])}),
    'pending_ids':sorted(pending),'credential_refs':sorted(references[role]),'transport_sha256':j.digest(transport),
    'channel_ids':sorted(row['id'] for row in transport['channels'])}
   if closed:value['closed']=True
   members[role]=value
  return members
 def prepare_recovery(self,request,specification,recovery_id):
  assert request==self.request and specification==self.spec
  return {'activation':{'collection':request['candidate'],'epoch':j.effect(request,'enable-candidate')},'loaded':None,
   'fences':request['uncertain'],'guardian':self.guardian,'closure_request':self.closure['request'],
   'authority':self.authority_id,'reservation_sha256':j.digest(self.original),'members':self.snapshot()}
 def recover(self,request,plan):
  assert request==self.request and plan['guardian']==self.guardian and plan['authority']==self.authority_id
  assert plan['reservation_sha256']==j.digest(self.original) and plan['closure_request']==self.closure['request']
  before=self.snapshot()
  for role in j.ROLES:assert all(before[role][key]==value for key,value in plan['members'][role].items() if key!='transport_sha256'),'candidate cohort changed'
  pending=[row for row in (self.theme.status() or {'channels':[]})['channels'] if row['phase']!='closed']
  if pending:
   assert 'theme' in self.roles
   with self.theme.operation() as lease:
    for row in pending:self.theme_client.cleanup_channel(lease,row)
  for row in (self.dms.transport.status() or {'channels':[]})['channels']:
   if row['phase']!='closed':
    assert 'dms' in self.roles;self.dms.delete_transport_record(row)
  value={'plan_sha256':j.digest(plan),'guardian':self.guardian,'closure_request':self.closure['request'],
   'closure_receipt_sha256':sha(Path(self.closure['receipt'])),'proof':'waitpid-echild','members':self.snapshot(closed=True)}
  j.recovered(value,plan)
  if self.control['fault']=='after-cleanup':
   save(ROOT/getattr(self,'effect_name','candidate-cleanup-effect.json'),value);os.kill(os.getpid(),signal.SIGKILL)
  return value
