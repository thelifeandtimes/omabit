"""Private actual-plugin recovery adapter. No host loader or production command.

Only the fresh lab's already-closed original guardian can supply process closure.
Per-plugin locks exclude cooperating work during each snapshot/effect/receipt;
changed cohorts between retries refuse. Durable global loader admission is still
required before offering this contract on a user's desktop.
"""
import contextlib,copy,fcntl,json,os,signal,subprocess,sys
from pathlib import Path
from journal_runtime_fixture import ROOT,Adapter,guard,read,sha,save,tree,gone
import journal as j
import coordinator as c


class RecoveryAdapter:
    def __init__(self,request,control,*,runtime=None):
        guard();assert os.environ.get('OMABIT_COLLECTION_EXPLICIT_RECOVERY')=='1'
        self.request=request;self.control=control;self.runtime=runtime if runtime is not None else Adapter();self.stack=contextlib.ExitStack()
        self.spec=self.runtime.input['specs']['stable']
        assert request['previous']['collection']==j.digest(self.spec)
        self.path=self.runtime.bound(request['previous']['epoch']);self.binding=read(self.path)
        self.closure=control['closure'];self.proof_path=ROOT/Path(self.closure['receipt']).relative_to('/lab')
    def verify_owner(self):
        assert read(self.path)==self.binding and self.binding['loaded']==self.request['previous']
        assert self.binding['supervisor']==self.closure['owner']
        assert {k:self.binding['shell'][k] for k in ('pid','start')}=={k:self.closure['root'][k] for k in ('pid','start')}
        proof=read(self.proof_path)
        assert sha(self.proof_path)==self.closure['receipt_sha256']
        assert proof['owner']==self.closure['owner'] and proof['root']==self.closure['root'] and proof['request']==self.closure['request']
        assert proof['phase']=='closed' and proof['proof']=='waitpid-echild'
        assert any(row['pid']==proof['root']['pid'] for row in proof['reaped'])
        assert proof['owner']['namespace']==os.readlink('/proc/self/ns/pid') and proof['owner']['boot']==Path('/proc/sys/kernel/random/boot_id').read_text().strip()
        assert gone(proof['owner']) and gone(proof['root'])
        assert all(gone(row) for rows in self.binding['helpers'].values() for row in rows)
        assert self.runtime.input['supervisor']!=proof['owner'] and not self.runtime.bindings()
        for role in j.ROLES:self.runtime.verify_artifact(self.spec,role)
        return {'owner':{k:proof['owner'][k] for k in ('token','pid','start','boot','namespace')},'root':{k:proof['root'][k] for k in ('pid','start')}}
    def __enter__(self):
        try:
            self.guardian=self.verify_owner()
            sys.path.insert(0,str(self.runtime.artifact_path(self.spec,'theme')))
            from client.support import StateStore,Keyring
            from client.transport_journal import TransportJournal,session_ref
            from client.eyre import Eyre
            self.theme_store=StateStore();self.theme=TransportJournal(self.theme_store.root)
            assert self.theme_store.root==Path(self.spec['members']['theme']['paths']['state'])
            self.stack.enter_context(self.theme_store.locked());rows=self.theme_store.load()['ships'];assert len(rows)==1
            self.theme_row=rows[0];self.keyring=Keyring();self.theme_session=self.keyring.lookup(rows[0]['url'],rows[0]['ship'],check_expiry=False)
            with self.theme.transaction():fd=self.theme.origin_lock(self.theme_session['url'])
            self.stack.callback(os.close,fd);fcntl.flock(fd,fcntl.LOCK_EX|fcntl.LOCK_NB)
            assert all(self.theme.recoverable(row) for row in self.theme.status()['channels'])
            self.theme_ref=session_ref;self.theme_client=Eyre(self.theme_session['url'],self.theme_session,journal=self.theme)
            sys.path.insert(0,str(self.runtime.artifact_path(self.spec,'dms')))
            from session_store import SessionStore,cookie_for
            from bridge import Bridge
            from transport_journal import session_ref as dms_ref
            self.dms_store=SessionStore();assert self.dms_store.root==Path(self.spec['members']['dms']['paths']['state'])
            self.dms=Bridge(session_store=self.dms_store);self.dms.ensure_transport_owner();self.stack.callback(self.dms.release_transport_owner)
            self.stack.callback(self.dms.pool.shutdown,wait=True)
            self.stack.enter_context(self.dms_store.locked());self.dms_account=self.dms_store.account()
            self.dms_session=self.dms_store.lookup(self.dms_account['url'],self.dms_account['ship'],check_expiry=False)
            self.dms.transport_session=self.dms_session;self.dms.cookies.set_cookie(cookie_for(self.dms_session));self.dms_ref=dms_ref
            # A separate process must fail to obtain every selected admission lock.
            lock_paths=[self.theme_store.root/'lock',Path('/proc/self/fd/'+str(fd)).resolve(),self.dms_store.root/'account.lock',self.dms.transport.root/'owner.lock']
            code="import fcntl,os,sys; f=os.open(sys.argv[1],os.O_RDWR); blocked=False; exec('try: fcntl.flock(f,fcntl.LOCK_SH|fcntl.LOCK_NB)\\nexcept BlockingIOError: blocked=True'); os.close(f); raise SystemExit(0 if blocked else 1)"
            for path in lock_paths:
                q=subprocess.run([sys.executable,'-B','-c',code,str(path)],capture_output=True,timeout=4);assert q.returncode==0,'plugin admission not exclusive'
            save(ROOT/'recovery-admission.json',{'locks_exclusively_held':len(lock_paths),'owner':self.guardian,'request':self.request['id']})
            return self
        except BaseException:self.stack.close();raise
    def __exit__(self,*args):return self.stack.__exit__(*args)
    def snapshot(self,closed=False):
        self.verify_owner()
        theme=self.theme.status();dms=self.dms.transport.status();members={}
        credentials={'theme':[self.theme_ref(self.keyring.lookup(self.theme_row['url'],self.theme_row['ship'],check_expiry=False))],
                     'dms':[self.dms_ref(self.dms_store.lookup(self.dms_account['url'],self.dms_account['ship'],check_expiry=False))],'omabit':[]}
        for role in j.ROLES:
            base=Path(self.spec['members'][role]['paths']['state']);transport={'theme':theme,'dms':dms,'omabit':{'channels':[]}}[role]
            names={'theme':['state.json','desktop.json'],'dms':['account.json','outbound-messages.json','conversation-effects.json'],'omabit':[]}[role]
            files={name:sha(base/name) if (base/name).exists() else None for name in names}
            immutable=[];pending=[]
            for row in transport['channels']:
                fields=('id','origin','ship','session_ref','channel','owner')+(('kind','intent','outcome') if role=='theme' else ())
                immutable.append({k:row[k] for k in fields})
                if role=='theme' and row['kind']=='poke' and row['outcome']=='uncertain':pending.append(row['id'])
                if closed:assert row['phase']=='closed'
            if role=='dms':
                value=read(base/'outbound-messages.json');pending.extend(row['id'] for row in value['messages'] if row['phase']=='uncertain')
                # This fixture's only outbound application intent is the saved DM;
                # unknown effect files require a separate characterized adapter.
                assert not (base/'conversation-effects.json').exists()
            value={'binding':self.request['previous']['members'][role],
                   'application_sha256':j.digest({'files':files,'transport_bindings':immutable,'sessions':transport.get('sessions',[])}),
                   'pending_ids':sorted(pending),'credential_refs':sorted(credentials[role]),'transport_sha256':j.digest(transport),'channel_ids':sorted(row['id'] for row in transport['channels'])}
            if closed:value['closed']=True
            members[role]=value
        return members
    def prepare_recovery(self,request,specification,recovery_id):
        assert request==self.request and specification==self.spec
        return {'loaded':request['previous'],'fences':request['uncertain'],'guardian':self.guardian,'closure_request':self.closure['request'],
                'runtime_binding_sha256':sha(self.path),'members':self.snapshot()}
    def recover(self,request,plan):
        assert request==self.request and plan['guardian']==self.guardian and plan['runtime_binding_sha256']==sha(self.path)
        assert plan['closure_request']==self.closure['request']
        before=self.snapshot()
        for role in j.ROLES:
            assert all(before[role][key]==value for key,value in plan['members'][role].items() if key!='transport_sha256'),'retained plugin cohort changed'
        # Existing plugin cleanup functions perform only exact channel deletion.
        # No disconnect, logout, account removal, scry or application publication.
        pending=[row for row in self.theme.status()['channels'] if row['phase']!='closed']
        if pending:
            with self.theme.operation() as lease:
                for row in pending:self.theme_client.cleanup_channel(lease,row)
        for row in self.dms.transport.status()['channels']:
            if row['phase']!='closed':self.dms.delete_transport_record(row)
        observation={'plan_sha256':j.digest(plan),'guardian':self.guardian,'closure_request':self.closure['request'],
                     'closure_receipt_sha256':sha(self.proof_path),'proof':'waitpid-echild','members':self.snapshot(closed=True)}
        j.recovered(observation,plan)
        if self.control['fault']=='after-cleanup':
            save(ROOT/'recovery-effect.json',observation);os.kill(os.getpid(),signal.SIGKILL)
        return observation


def worker():
    guard();control=read(ROOT/'recovery-control.json');store=j.Store(ROOT/'collection-journal')
    try:
        with store.lease() as lease:
            request=store.status()['requests'][control['request']]
            with RecoveryAdapter(request,control) as adapter:
                result=c.recover_step(store,lease,adapter,control['request'],control['recovery'])
                if control['fault']=='after-prepared':
                    save(ROOT/'recovery-prepared-boundary.json',{'request':control['request'],'recovery':control['recovery']});os.kill(os.getpid(),signal.SIGKILL)
        print(json.dumps({'phase':result['requests'][control['request']]['phase']}))
    finally:store.close()

if __name__=='__main__':worker()
