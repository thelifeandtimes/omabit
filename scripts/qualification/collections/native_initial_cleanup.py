"""Private initial-owner cleanup using actual supported plugin transport APIs."""
import contextlib,copy
from pathlib import Path
import journal as j
import runtime_authority as authority
import startup_recovery
from native_candidate_recovery import CandidateRecovery
from journal_runtime_fixture import ROOT,guard,read,gone

class InitialCleanup(CandidateRecovery):
 effect_name='initial-cleanup-effect.json'
 def __init__(self,control,runtime,store,lease):
  guard();self.control=control;self.runtime=runtime;self.store=store;self.lease=lease
  self.request={'recovery':control['recovery']};self.stack=contextlib.ExitStack();self.spec=runtime.input['specs']['stable']
  self.closure=control['closure'];self.original=None;self.admission=None;self.case=read(ROOT/'initial-owner-mode.json').get('cohort','all')
  self.roles={'all':set(j.ROLES),'before-issue':set(),'after-issue':set(),'omabit-only':{'omabit'},'theme-only':{'theme'},'dms-only':{'dms'}}[self.case]
 def verify_owner(self):
  a=authority.Authority(ROOT/'runtime'/authority.ANCHOR)
  try:
   expected,closure,spec=startup_recovery.context(a,self.store,self.lease)
   assert spec==self.spec and closure=={'path':self.closure['receipt'],'sha256':self.closure['receipt_sha256'],'request':self.closure['request']}
   proof=read(Path(closure['path']));assert proof['owner']==self.closure['owner'] and proof['root']==self.closure['root']
   assert gone(proof['owner']) and gone(proof['root']) and not self.runtime.bindings()
   original=a.status()['reservation'];row,snapshot=a._partial_gate(original)
   if self.case=='before-issue':assert row is None and snapshot['missing']==['admission.json','issued.json']
   else:assert row is not None and not snapshot['missing'] and set(row['claims'])==self.roles
   if self.original is None:self.original=original;self.admission=row;self.authority_id=a.descriptor['id']
   else:assert self.original==original and self.admission==row and self.authority_id==a.descriptor['id']
   for role in j.ROLES:self.runtime.verify_artifact(self.spec,role)
   return copy.deepcopy(original['owner']['guardian'])
  finally:a.close()
 def prepare_initial(self,expected,spec):
  assert expected['guardian']==self.guardian and spec==self.spec
  assert expected['authority']==self.authority_id and expected['reservation_sha256']==j.digest(self.original)
  return self.snapshot()
 def recover_initial(self,plan,spec):
  assert spec==self.spec
  return self.recover(self.request,plan)
