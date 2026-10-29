"""Private supported-collection adapter qualification; never a host adapter.

The native loader functions and real plugins/helpers are exercised. Component
identity observation is injected into private copies only. Omabit tools and
compositor controls remain inert; Eyre and endpoint ownership are synthetic.
"""
from pathlib import Path
import hashlib,json,os,signal,socket,subprocess,sys,threading,time,uuid
sys.path.insert(0,'/lab/collection-source')
import journal as j
import coordinator as c
ROOT=Path('/lab')


def guard():
    assert os.environ.get('OMABIT_COLLECTION_JOURNAL')=='private-v1' and Path.home()==ROOT/'home'
    assert not Path('/home').exists() and not Path('/run/user').exists()
    original=json.loads((ROOT/'host-namespaces.json').read_text())
    assert all(os.readlink('/proc/self/ns/'+n)!=v for n,v in original.items())
    assert [n for _,n in socket.if_nameindex()]==['lo']


def read(path):return json.loads(path.read_text())
def sha(path):return hashlib.sha256(path.read_bytes()).hexdigest()
def save(path,value):
    temporary=path.with_name('.'+path.name+'-'+uuid.uuid4().hex)
    with temporary.open('x') as f:
        os.chmod(temporary,0o600);f.write(json.dumps(value,indent=2)+'\n');f.flush();os.fsync(f.fileno())
    temporary.replace(path)
    fd=os.open(path.parent,os.O_DIRECTORY);os.fsync(fd);os.close(fd)


def tree(path):
    files={}
    for p in sorted(path.rglob('*')):
        assert not p.is_symlink()
        if p.is_file():files[str(p.relative_to(path))]=sha(p)
        else:assert p.is_dir()
    return j.digest(files)


def children(pid):
    try:ids=[int(x) for x in Path('/proc/'+str(pid)+'/task/'+str(pid)+'/children').read_text().split()]
    except FileNotFoundError:return []
    return ids+[v for child in ids for v in children(child)]


def identity(pid):
    try:
        p=Path('/proc')/str(pid);row=(p/'stat').read_text().split(') ',1)[1].split()
        return {'pid':pid,'start':row[19],'state':row[0],'args':(p/'cmdline').read_bytes().split(b'\0')}
    except (FileNotFoundError,ProcessLookupError):return None


def gone(old):
    now=identity(old['pid'])
    return now is None or now['start']!=old['start'] or now['state']=='Z'


class Adapter:
    def __init__(self):
        guard();self.input=read(ROOT/'coordinator-input.json');self.pid=self.input['shell']['pid']
    def call(self,method,*args):
        if gone(self.input['shell']):raise c.Problem('reader-owner-lost')
        p=subprocess.run(['/usr/bin/quickshell','ipc','--pid',str(self.pid),'call','--','realPluginLifecycle',method,*args],capture_output=True,text=True,timeout=5)
        assert p.returncode==0,(method,p.stderr[-500:])
        return p.stdout.strip()
    def snapshot(self):return json.loads(self.call('state'))
    def bindings(self):return json.loads(self.call('bindings'))
    def wait(self,fn,seconds=18):
        deadline=time.monotonic()+seconds
        while time.monotonic()<deadline:
            if gone(self.input['shell']):raise c.Problem('reader-owner-lost')
            value=fn()
            if value:return value
            time.sleep(.07)
        raise AssertionError('runtime observation deadline')
    def spec(self,name):return self.input['specs'][name]
    def artifact_path(self,spec,role):return ROOT/'artifacts'/spec['name']/role
    def verify_artifact(self,spec,role):assert tree(self.artifact_path(spec,role))==spec['members'][role]['artifact_sha256']
    def loaded(self,spec,epoch):
        observed=self.bindings()
        if set(observed)!=set(j.ROLES) or any(row is None for row in observed.values()):return None
        members={}
        for role in j.ROLES:
            row=observed[role];member=spec['members'][role]
            expected=ROOT/'artifacts'/spec['name']/role
            filename={'omabit':'CollectionFixture.qml','theme':'Service.qml','dms':'Dms.qml'}[role]
            if row['epoch']!=epoch or row['source']!=(expected/filename).as_uri() or row['profile']!=member['profile'] or row['plugin_id']!=member['plugin_id']:
                raise c.Problem('artifact-mismatch')
            if tree(expected)!=member['artifact_sha256']:raise c.Problem('artifact-mismatch')
            members[role]={**{k:member[k] for k in ('plugin_id','profile','artifact_sha256')},'instance':row['instance']}
        value={'collection':j.digest(spec),'epoch':epoch,'members':members};j.loaded(value,spec,epoch)
        return value
    def bound(self,epoch):return ROOT/'runtime-observations'/(epoch+'.json')
    def remember(self,spec,epoch):
        loaded=self.wait(lambda:self.loaded(spec,epoch))
        path=self.bound(epoch)
        if path.exists():
            prior=read(path);assert prior['loaded']==loaded
            return loaded
        assert self.call('openOmabit')=='opened'
        def helpers():
            rows={r:[] for r in j.ROLES}
            for pid in children(self.pid):
                row=identity(pid)
                if not row or row['state']=='Z':continue
                args=row.pop('args');prefix=('/lab/artifacts/'+spec['name']+'/').encode()
                for role,suffix in (('omabit',b'/omabit/bin/urbit-collector'),('theme',b'/theme/client/sync.py'),('dms',b'/dms/bridge.py')):
                    if any(a.startswith(prefix) and a.endswith(suffix) for a in args):rows[role].append(row)
            return rows if [len(rows[r]) for r in j.ROLES]==[2,1,1] else None
        owned=self.wait(helpers)
        saved={'loaded':loaded,'helpers':owned,'shell':self.input['shell']}
        if 'supervisor' in self.input:saved['supervisor']=self.input['supervisor']
        save(path,saved)
        return loaded
    def item(self,role):
        s=self.snapshot()
        return s.get('omabit') if role=='omabit' else next(iter(s['services' if role=='theme' else 'panels'].values()),None)
    def channels(self):return read(ROOT/'hub-observation.json')['channels']
    def healthy(self,spec,loaded):
        assert self.loaded(spec,loaded['epoch'])==loaded
        def ready():
            t=self.item('theme');d=self.item('dms');o=self.item('omabit')
            if t and t['desktop']['status']=='Needs attention':raise c.Problem('readiness-failed')
            channels=[x for x in self.channels() if x['active_streams']]
            return (t and d and o and t['loaded'] and t['desktop']['status']=='Following shared appearance' and not t['desktop']['error'] and not t['desktop'].get('bridgeBusy') and d['ship']=='~zod' and d['outboundReadable'] and o['observing'] and len(channels)==2 and {x['owner'] for x in channels}=={spec['name']+'-theme',spec['name']+'-dms'} and all(x['delivered']>=x['subscriptions']>0 for x in channels))
        self.wait(ready)
        from appearance_fixture import profile
        value=read(ROOT/'desktop.json');expected=profile(spec['name'])
        assert value['font']==expected['font'] and value['colors']==expected['colors'] and value['rounding']==expected['windows']['decoration:rounding']['int']
        assert (ROOT/'home/.local/state/omarchy-urbit-theme-ownership/owner').read_text().splitlines()[1]=='omarchy-urbit-theme.qualification-'+spec['name']
        observed_url=read(ROOT/'hub-observation.json')['url']
        endpoints={r:None if r=='omabit' else j.digest({'url':observed_url,'ship':'~zod','profile':spec['name'],'role':r}) for r in j.ROLES}
        observation={'loaded':loaded,'ready':True,'endpoints':endpoints}
        j.ready(observation,spec,loaded)
        return observation
    def local(self,spec,*args):
        script=self.artifact_path(spec,'theme')/'client/sync.py'
        # Exact copied helper, external to Quickshell; forbid any network or
        # credential access even if the offline CLI later regresses.
        code="""import runpy,sys
from pathlib import Path
script=sys.argv[1];sys.argv=sys.argv[1:];sys.path.insert(0,str(Path(script).parent))
def audit(event,args):
 if event in ('socket.connect','socket.getaddrinfo','socket.bind') or event=='subprocess.Popen' and Path(args[0]).name in ('secret-tool','busctl'):raise RuntimeError('offline command attempted network/credentials')
sys.addaudithook(audit);runpy.run_path(script,run_name='__main__')
"""
        env=dict(os.environ);env.pop('DBUS_SESSION_BUS_ADDRESS',None)
        q=subprocess.run([sys.executable,'-B','-c',code,str(script),*args],capture_output=True,text=True,timeout=18,env=env)
        lines=q.stdout.splitlines();assert len(lines)==1,(q.returncode,q.stderr[-500:])
        value=json.loads(lines[0])
        with (ROOT/'offline-commands.jsonl').open('a') as f:f.write(json.dumps({'profile':spec['name'],'args':args,'exit':q.returncode,'result':value})+'\n')
        if q.returncode:raise c.Problem('appearance-failed')
        return value
    def perform(self,phase,request,spec,role):
        if gone(self.input['shell']):raise c.Problem('reader-owner-lost')
        epoch=request['previous']['epoch'] if phase.endswith('previous') else j.effect(request,'enable-candidate')
        if phase.startswith('drain-'):
            observed=read(self.bound(epoch));assert self.loaded(spec,epoch)==observed['loaded']
            old=observed['helpers'][role]
            receipt=self.item(role)['transportState']
            if not receipt.get('closed'):
                result=self.call({'omabit':'drainOmabit','theme':'drainTheme','dms':'drainDms'}[role])
                if result=='busy':raise c.Busy()
                assert result=='requested',result
            def closed():
                v=self.item(role)['transportState']
                if v.get('unconfirmedHelpers'):raise c.Problem('helper-descendants-unconfirmed')
                if v.get('readerOwnerLost'):raise c.Problem('reader-owner-lost')
                return v if v.get('closed') else None
            receipt=self.wait(closed,75)
            self.wait(lambda:all(gone(p) for p in old))
            if role=='omabit':assert receipt['auxiliaryReadersTracked'] and receipt['helperExitsTracked'] and not receipt['pendingReaders']
            else:
                self.wait(lambda:all(x['deleted'] and not x['active_streams'] for x in self.channels() if x['owner']==spec['name']+'-'+role))
            assert not receipt['collectionReady']
            names={'omabit':[],'theme':['desktop.json'],'dms':['outbound-messages.json','conversation-effects.json']}[role]
            base=Path(spec['members'][role]['paths']['state'])
            retained={name:sha(base/name) if (base/name).exists() else None for name in names}
            pending=[x['id'] for key in ('unresolvedSends','unresolvedEffects') for x in receipt.get(key,[])]
            value={'epoch':epoch,'binding':observed['loaded']['members'][role],'absent':False,'closed':True,'retained_state':j.digest(retained),'pending_ids':pending}
        elif phase.startswith('disable-'):
            observed=read(self.bound(epoch))
            assert all(gone(p) for rows in observed['helpers'].values() for p in rows)
            if self.bindings():
                assert self.loaded(spec,epoch)==observed['loaded']
                self.call('select','none')
            self.wait(lambda:not self.bindings() and not children(self.pid))
            value={'epoch':epoch,'absent':True,'helpers_exited':True}
        elif phase=='checkpoint-appearance':
            assert not self.bindings() and not children(self.pid)
            value={'checkpoint':self.local(spec,'checkpoint-appearance')['appearanceCheckpoint']['id']}
        elif phase=='restore-appearance':
            assert not self.bindings() and not children(self.pid)
            result=self.local(spec,'restore-appearance',request['checkpoint'],j.effect(request,phase))
            assert result['appearanceRestoration']['phase']=='verified'
            value={'checkpoint':request['checkpoint'],'verified':True}
        elif phase.startswith('enable-'):
            epoch=j.effect(request,phase)
            if not self.bindings():self.call('selectEpoch',spec['name'],epoch)
            value=self.remember(spec,epoch)
        else:value=self.healthy(spec,request['loaded'])
        fault=read(ROOT/'coordinator-fault.json')
        if fault=={'phase':phase,'role':role}:
            save(ROOT/'lost-reply.json',{'phase':phase,'role':role,'effect':j.effect(request,phase),'observation':value})
            os.kill(os.getpid(),signal.SIGKILL)
        return value


def worker():
    guard();adapter=Adapter();store=j.Store(ROOT/'collection-journal')
    try:
        with store.lease() as lease:state=c.step(store,lease,adapter)
        print(json.dumps({'pending':state['pending'],'phase':state['requests'][state['pending']]['phase'] if state['pending'] else None}))
    finally:store.close()


def run(root):
    guard()
    from appearance_fixture import prepare,AppearanceHub
    from connected_fixture import begin
    prepare(root);keyring,hub=begin(root,AppearanceHub)
    shell_recovery=os.environ.get('OMABIT_COLLECTION_SHELL_RECOVERY')=='1'
    supervised=os.environ.get('OMABIT_COLLECTION_SUPERVISOR')=='1'
    recovery_state=os.environ.get('OMABIT_COLLECTION_RECOVERY_STATE')=='1'
    explicit_recovery=os.environ.get('OMABIT_COLLECTION_EXPLICIT_RECOVERY')=='1'
    guardians=[]
    def launch_shell():
        if supervised:
            from supervised_shell import SupervisedShell
            proc=SupervisedShell();guardians.append(proc);return proc
        log=(root/'shell.log').open('ab')
        return subprocess.Popen(['/usr/bin/quickshell','--no-color','--path','/lab/shell.qml'],stdout=log,stderr=subprocess.STDOUT,start_new_session=True)
    report={'shell_recovery':shell_recovery,'passed':False,'private_runtime':True,'synthetic_eyre':True,'inert_omabit_tools':True,'inert_desktop_controls':True,'private_identity_instrumentation':True,'cases':[],'killed_workers':[]}
    def report_save():save(root/'result.json',report)
    report_save();stop=threading.Event()
    def observe_hub():
        while not stop.is_set():
            save(root/'hub-observation.json',{'url':hub.url,'channels':hub.snapshot()});stop.wait(.05)
    thread=threading.Thread(target=observe_hub,daemon=True);thread.start()
    proc=None;store=None
    try:
        specs={}
        for name in ('stable','dev','edge'):
            members={}
            for role in j.ROLES:
                artifact=root/'artifacts'/name/role;manifest=read(artifact/'manifest.json')
                base='omarchy-urbit-'+role+'.qualification-'+name
                paths={k:str(root/'omabit'/k/name) if role=='omabit' else str(root/k/base) for k in j.PATHS}
                members[role]={'plugin_id':manifest['id'],'profile':'qualification-'+name,'artifact_sha256':tree(artifact),'paths':paths,'endpoint_ref':None if role=='omabit' else j.digest({'url':hub.url,'ship':'~zod','profile':name,'role':role})}
            specs[name]={'format':1,'name':name,'members':members};j.specification(specs[name])
        (root/'runtime-observations').mkdir(mode=0o700)
        proc=launch_shell()
        row=identity(proc.pid);row.pop('args');initial={'shell':row,'specs':specs}
        if supervised:initial['supervisor']=proc.owner
        save(root/'coordinator-input.json',initial);save(root/'coordinator-fault.json',{})
        a=Adapter();a.wait(lambda:subprocess.run(['/usr/bin/quickshell','ipc','--pid',str(proc.pid),'call','--','realPluginLifecycle','state'],capture_output=True,text=True,timeout=4).stdout.strip().startswith('{'))
        epoch=uuid.uuid4().hex;a.call('selectEpoch','stable',epoch);loaded=a.remember(specs['stable'],epoch);ready=a.healthy(specs['stable'],loaded)
        # A live component from another activation cannot supply this receipt.
        try:a.loaded(specs['stable'],uuid.uuid4().hex)
        except c.Problem as error:assert error.reason=='artifact-mismatch'
        else:raise AssertionError('stale activation identity was accepted')
        report['stale_runtime_binding_refused']=True
        store=j.Store(root/'collection-journal',create=True)
        def record(event):
            with store.lease() as lease:return store.record(lease,event)
        for s in specs.values():record({'type':'register','spec':s})
        record({'type':'adopt','observation':ready})
        def replace_shell(reason,clean=True):
            nonlocal proc,a
            old=read(root/'coordinator-input.json')['shell']
            if clean:
                assert not a.bindings() and not children(proc.pid)
                assert a.call('stop')=='stopping' and proc.wait(timeout=5)==0
                if supervised:proc.close_owned()
            else:assert proc.poll()==-signal.SIGKILL
            assert gone(old)
            proc=launch_shell()
            row=identity(proc.pid);row.pop('args');value=read(root/'coordinator-input.json');value['shell']=row
            if supervised:value['supervisor']=proc.owner
            save(root/'coordinator-input.json',value)
            a=Adapter();a.wait(lambda:subprocess.run(['/usr/bin/quickshell','ipc','--pid',str(proc.pid),'call','--','realPluginLifecycle','state'],capture_output=True,text=True,timeout=4).stdout.strip().startswith('{'))
            assert not a.bindings() and not children(proc.pid)
            report.setdefault('shell_replacements',[]).append({'reason':reason,'prior':old,'new':row,'old_exited':True,'starts_empty':True});report_save()
        def pending():
            s=store.status();return s['requests'][s['pending']] if s['pending'] else None
        def begin_switch(name):
            s=store.status();request=uuid.uuid4().hex
            record({'type':'begin','request':request,'candidate':j.digest(specs[name]),'expected':j.digest(s['committed'])})
            return request
        def step(kill=False):
            before=pending();phase=before['phase'];effect=j.effect(before,phase)
            role=next((r for r in j.ROLES if effect+'/'+r not in before['receipts']),None) if phase.startswith('drain-') else None
            save(root/'coordinator-fault.json',{'phase':phase,'role':role} if kill else {})
            q=subprocess.run([sys.executable,'-B','/lab/journal_runtime_fixture.py','worker'],capture_output=True,text=True,timeout=90)
            with (root/'worker.log').open('a') as f:f.write(json.dumps({'phase':phase,'role':role,'exit':q.returncode})+'\n'+q.stdout+q.stderr)
            if kill:
                assert q.returncode==-signal.SIGKILL,(q.returncode,q.stderr[-700:]);lost=read(root/'lost-reply.json');assert lost['phase']==phase and pending()['phase']==phase
                report['killed_workers'].append({'phase':phase,'role':role,'effect':effect});report_save()
                return lost['observation']
            assert q.returncode==0,q.stderr[-1000:]
        def finish():
            for _ in range(28):
                if pending() is None:return
                assert not pending()['problem'],pending()['problem'];step()
            raise AssertionError('too many coordinator steps')
        first=begin_switch('dev')
        for phase in ('drain-previous','disable-previous','checkpoint-appearance','enable-candidate','verify-candidate'):
            while pending()['phase']!=phase:step()
            lost=step(kill=True)
            if shell_recovery and phase=='checkpoint-appearance':replace_shell('retained-checkpoint-before-candidate')
            step()
            current=store.status()['requests'][first]
            if phase=='enable-candidate':assert current['loaded']==lost
        assert pending() is None and store.status()['requests'][first]['outcome']=='committed'
        report['cases'].append({'name':'stable-to-dev','outcome':'committed','lost_replies':5});report_save()
        second=begin_switch('stable');finish();assert store.status()['requests'][second]['outcome']=='committed'
        save(root/'fault.json',{'profile':'edge','stage':'font'})
        third=begin_switch('edge')
        while pending()['phase']!='verify-candidate':step()
        step();assert pending()['problem']=='readiness-failed'
        p=pending();record({'type':'rollback','request':p['id'],'phase':p['phase'],'effect':j.effect(p,p['phase']),'reason':'readiness-failed'})
        while pending()['phase']!='restore-appearance':step()
        save(root/'fault.json',{'profile':'stable','stage':'font'});step();assert pending()['phase']=='restore-appearance' and pending()['problem']=='appearance-failed'
        save(root/'fault.json',{});step(kill=True);events=sha(root/'desktop-events.jsonl')
        if shell_recovery:replace_shell('verified-restore-before-journal-receipt')
        step();assert sha(root/'desktop-events.jsonl')==events
        finish();assert store.status()['requests'][third]['outcome']=='rolled-back'
        report['cases'].append({'name':'edge-partial-apply','outcome':'rolled-back','failed_restore_retained':True,'verified_restore_replay_no_effects':True});report_save()
        # Explicitly drain the last collection through the same journal for
        # teardown, retaining a pending checkpoint phase instead of pretending
        # the now-unloaded desktop is still a verified running collection.
        cleanup=begin_switch('dev')
        cohort=None
        if recovery_state:
            assert supervised and shell_recovery
            hub.uncertain_mode=True
            before_pokes=len(hub.pokes)
            assert a.call('send')==''
            a.wait(lambda:len(a.item('dms')['unresolved'])==1 and not a.item('dms')['busy'] and a.item('dms')['pendingSend']==-1,seconds=45)
            dms_base=root/'state/omarchy-urbit-dms.qualification-stable'
            theme_base=root/'state/omarchy-urbit-theme.qualification-stable'
            messages=read(dms_base/'outbound-messages.json')['messages']
            assert len(messages)==1 and messages[0]['phase']=='uncertain'
            # Use the unchanged plugin's profile-to-palette function in a separate
            # read-only child; no credentials/network are needed for this conversion.
            code="import json,sys;sys.path.insert(0,'/lab/artifacts/stable/theme');from client.profile import palette;print(json.dumps(palette(json.load(sys.stdin))))"
            from appearance_fixture import profile
            q=subprocess.run([sys.executable,'-B','-c',code],input=json.dumps(profile('stable')),capture_output=True,text=True,timeout=5)
            assert q.returncode==0
            a.wait(lambda:a.call('publishTheme',q.stdout.strip())=='requested')
            def unknown_theme():
                saved=read(theme_base/'transport/channels.json')
                rows=[row for row in saved['channels'] if row['kind']=='poke' and row['outcome']=='uncertain']
                return rows if len(rows)==1 and rows[0]['phase']=='closed' and rows[0]['owner_released'] else None
            theme_rows=a.wait(unknown_theme,seconds=20)
            cohort={'loaded':pending()['previous'],'dms_outbound_sha256':sha(dms_base/'outbound-messages.json'),
                    'dms_pending_ids':[row['id'] for row in messages],
                    'theme_pending':[{k:row[k] for k in ('id','origin','ship','session_ref','intent','outcome')} for row in theme_rows],
                    'accepted_uncertain_writes':[row for row in hub.pokes[before_pokes:] if row.get('uncertain')]}
            assert len(cohort['accepted_uncertain_writes'])==2
            save(root/'recovery-cohort-before.json',cohort)
            if explicit_recovery:hub.block_cleanup=True
        if shell_recovery:
            lineage=[]
            for pid in children(proc.pid):
                row=identity(pid)
                if row:
                    row.pop('args');lineage.append(row)
            assert len(lineage)>=4
            lost_supervisor=proc if supervised else None
            proc.kill();assert proc.wait(timeout=5)==-signal.SIGKILL
            step();p=pending();assert p['phase']=='drain-previous' and p['uncertain']==[{'phase':'drain-previous','reason':'reader-owner-lost'}]
            before=sha(root/'collection-journal/journal.json')
            replace_shell('ungraceful-owner-loss',clean=False)
            def refuses():
                q=subprocess.run([sys.executable,'-B','/lab/journal_runtime_fixture.py','worker'],capture_output=True,text=True,timeout=10)
                assert q.returncode!=0 and 'unconfirmed helper lineage prevents runtime effects' in q.stderr
                assert sha(root/'collection-journal/journal.json')==before and not a.bindings() and not children(proc.pid)
            refuses()
            survivors=[row for row in lineage if not gone(row)]
            # Cleanup belongs to the private fixture, not the coordinator. Exact
            # PID/start identities only; this does not authorize clearing its fence.
            if supervised:
                closure=lost_supervisor.close_owned()
                observed=read(a.bound(p['previous']['epoch']));assert observed['supervisor']==closure['owner']
                assert closure['root']['pid']!=proc.pid and closure['owner']!=proc.owner
                report['supervised_owner_closure']=closure
            else:
                for row in survivors:
                    if not gone(row):os.kill(row['pid'],signal.SIGTERM)
                deadline=time.monotonic()+3
                while any(not gone(row) for row in survivors) and time.monotonic()<deadline:time.sleep(.05)
                for row in survivors:
                    if not gone(row):os.kill(row['pid'],signal.SIGKILL)
            a.wait(lambda:all(gone(row) for row in lineage));refuses()
            if cohort is not None:
                assert sha(dms_base/'outbound-messages.json')==cohort['dms_outbound_sha256']
                rows=read(theme_base/'transport/channels.json')['channels']
                after=[{k:row[k] for k in ('id','origin','ship','session_ref','intent','outcome')} for row in rows if row['kind']=='poke' and row['outcome']=='uncertain']
                assert after==cohort['theme_pending']
                assert len([p for p in hub.pokes if p.get('uncertain')])==2
                save(root/'recovery-cohort-after.json',cohort)
                report['recovery_application_cohort']={'real_helper_writes':2,'same_pending_intent_after_owner_loss':True,'no_replay':True,'before_sha256':sha(root/'recovery-cohort-before.json'),'after_sha256':sha(root/'recovery-cohort-after.json')}

            if explicit_recovery:
                assert cohort is not None and report['supervised_owner_closure']==closure
                assert len([x for x in hub.snapshot() if not x['deleted']])==2
                recovery=uuid.uuid4().hex
                control={'request':cleanup,'recovery':recovery,'closure':closure,'fault':''}
                def recovery_step(fault='',expected=0,wrong_owner=False):
                    value=dict(control);value['fault']=fault
                    if wrong_owner:
                        value['closure']={'owner':proc.owner,'root':proc.root,'request':uuid.uuid4().hex,'receipt':str(proc.path),'receipt_sha256':sha(proc.path)}
                    save(root/'recovery-control.json',value)
                    q=subprocess.run([sys.executable,'-B','/lab/recovery_runtime.py'],capture_output=True,text=True,timeout=30)
                    with (root/'recovery-worker.log').open('a') as f:f.write(json.dumps({'fault':fault,'wrong_owner':wrong_owner,'exit':q.returncode})+'\n'+q.stdout+q.stderr)
                    assert q.returncode==expected,(fault,q.returncode,q.stderr[-1200:])
                    return q
                initial=sha(root/'collection-journal/journal.json');recovery_step(expected=1,wrong_owner=True)
                assert sha(root/'collection-journal/journal.json')==initial
                recovery_step('after-prepared',-signal.SIGKILL)
                prepared=sha(root/'collection-journal/journal.json');assert pending()['recovery']['verified'] is None and pending()['uncertain']
                save(root/'recovery-plan.json',pending()['recovery']['plan'])
                recovery_step(expected=1)
                assert sha(root/'collection-journal/journal.json')==prepared and pending()['uncertain']
                assert len([x for x in hub.snapshot() if not x['deleted']])==2
                hub.block_cleanup=False
                recovery_step('after-cleanup',-signal.SIGKILL)
                assert sha(root/'collection-journal/journal.json')==prepared and pending()['uncertain']
                a.wait(lambda:all(x['deleted'] and not x['active_streams'] for x in hub.snapshot()))
                effects=len(hub.requests);recovery_step();assert pending()['phase']=='disable-previous' and not pending()['uncertain']
                assert len(hub.requests)==effects
                verified=sha(root/'collection-journal/journal.json');recovery_step()
                assert sha(root/'collection-journal/journal.json')==verified and len(hub.requests)==effects
                assert sha(dms_base/'outbound-messages.json')==cohort['dms_outbound_sha256']
                current=read(theme_base/'transport/channels.json')['channels']
                assert [{k:row[k] for k in ('id','origin','ship','session_ref','intent','outcome')} for row in current if row['kind']=='poke' and row['outcome']=='uncertain']==cohort['theme_pending']
                assert len([p for p in hub.pokes if p.get('uncertain')])==2
                report['explicit_recovery']={'recovery':recovery,'coordinator_kills':2,'wrong_guardian_refused':True,'failed_delete_retained':True,'plugin_owned_cleanup':True,'same_application_and_credentials':True,'replay_no_http':True,'verified_phase':pending()['phase']}
                save(root/'recovery-verified.json',pending()['recovery']['verified'])
                step();assert pending()['phase']=='checkpoint-appearance'
                # Finish the requested switch after independent recovery, then
                # return to the original profile with fresh positive subscriptions.
                recovered_switch=cleanup;finish()
                assert store.status()['requests'][recovered_switch]['outcome']=='committed'
                assert sha(dms_base/'outbound-messages.json')==cohort['dms_outbound_sha256']
                current=read(theme_base/'transport/channels.json')['channels']
                assert [{k:row[k] for k in ('id','origin','ship','session_ref','intent','outcome')} for row in current if row['kind']=='poke' and row['outcome']=='uncertain']==cohort['theme_pending']
                def active_receipts(name):
                    rows=[x for x in hub.snapshot() if x['active_streams']]
                    assert len(rows)==2 and {x['owner'] for x in rows}=={name+'-theme',name+'-dms'}
                    assert all(x['delivered']>=x['subscriptions']>0 and not x['deleted'] for x in rows)
                    return {'loaded':store.status()['committed'],'channels':rows}
                resumed={'dev':active_receipts('dev')}
                returned=begin_switch('stable');finish()
                assert store.status()['requests'][returned]['outcome']=='committed'
                resumed['stable']=active_receipts('stable')
                assert [row['id'] for row in a.item('dms')['unresolved']]==cohort['dms_pending_ids']
                assert sha(dms_base/'outbound-messages.json')==cohort['dms_outbound_sha256']
                original=cohort['theme_pending'][0]
                row=next(row for row in read(theme_base/'transport/channels.json')['channels'] if row['id']==original['id'])
                # State confirmation can occur through the resumed helper's normal
                # authenticated read; it must not be reported as an original ACK.
                assert row['outcome']=='state-confirmed',row['outcome']
                assert {k:row[k] for k in ('id','origin','ship','session_ref','intent')}=={k:original[k] for k in ('id','origin','ship','session_ref','intent')}
                assert len([p for p in hub.pokes if p.get('uncertain')])==2
                save(root/'recovery-resumed.json',{'connections':resumed,'dms_pending_ids':cohort['dms_pending_ids'],'dms_outbound_sha256':sha(dms_base/'outbound-messages.json'),'theme_state_confirmation':{k:row[k] for k in ('id','origin','ship','session_ref','intent','outcome')}})
                report['recovery_resume']={'candidate_request':recovered_switch,'return_request':returned,'both_committed':True,'new_positive_subscriptions':True,'dms_pending_visible':True,'theme_state_confirmed_without_replay':True}
                cleanup=begin_switch('dev')
                while pending()['phase']!='checkpoint-appearance':step()
                save(root/'http-requests.json',hub.requests)
                report_save()
            a.wait(lambda:all(x['deleted'] for x in hub.snapshot() if x['owner']=='stable-dms'),seconds=5)
            report['dms_eof_cleanup']=not explicit_recovery
            report['unclean_shell']={'old_lineage':lineage,'survived_owner_exit':survivors,'exact_fixture_cleanup':True,'journal_fence_survives_new_shell_and_cleanup':True,'unconfirmed_channels':[x for x in hub.snapshot() if not x['deleted']]};report_save()
        else:
            while pending()['phase']!='checkpoint-appearance':step()
            assert all(x['deleted'] and not x['active_streams'] for x in hub.snapshot())
        assert not a.bindings() and not children(proc.pid)
        assert a.call('stop')=='stopping' and proc.wait(timeout=5)==0
        if supervised:
            proc.close_owned();report['supervisor_closures']=[g.close_owned() for g in guardians]
            assert len(report['supervisor_closures'])==4
        assert not any(x in (root/'shell.log').read_text() for x in ('ReferenceError','TypeError','Binding loop','ERROR:'))
        assert not hub.errors,'HTTP fixture worker failed; see http-server-errors.json'
        report.update(passed=True,coordinator_process_per_step=True,committed_requests=[first,second]+([recovered_switch,returned] if explicit_recovery else []),rolled_back_request=third,teardown_request=cleanup,teardown_phase=pending()['phase'],channels=hub.snapshot(),clean_shell_exit=0,full_native_shell=False,production_adapter=False)
        report_save();print('PASS: journal-driven private activation, lost replies and rollback')
    except Exception as error:
        report['error']=type(error).__name__+': '+str(error);report_save();raise
    finally:
        if store:store.close()
        stop.set();thread.join(2);hub.close();keyring.terminate()
        try:keyring.wait(timeout=3)
        except subprocess.TimeoutExpired:keyring.kill();keyring.wait(timeout=3)
        if supervised:
            for guardian in guardians:guardian.close_owned()
        elif proc and proc.poll() is None:
            os.killpg(proc.pid,signal.SIGTERM)
            try:proc.wait(timeout=3)
            except subprocess.TimeoutExpired:os.killpg(proc.pid,signal.SIGKILL);proc.wait(timeout=3)

if __name__=='__main__':worker()
