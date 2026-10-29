#!/usr/bin/env python3
"""Private Linux subreaper qualification; no collection recovery authority yet."""
from pathlib import Path
import argparse,ctypes,errno,hashlib,json,os,select,shutil,signal,socket,subprocess,sys,time,uuid
ROOT=Path('/lab')

def read(path):return json.loads(path.read_text())
def save(path,value):
    temp=path.with_name('.'+path.name+'-'+uuid.uuid4().hex)
    with temp.open('x') as f:
        os.chmod(temp,0o600);f.write(json.dumps(value,indent=2)+'\n');f.flush();os.fsync(f.fileno())
    temp.replace(path);fd=os.open(path.parent,os.O_DIRECTORY|os.O_NOFOLLOW)
    try:os.fsync(fd)
    finally:os.close(fd)
def sha(path):return hashlib.sha256(path.read_bytes()).hexdigest()
def guard():
    assert Path.home()==ROOT/'home' and not Path('/home').exists() and not Path('/run/user').exists()
    assert os.environ.get('OMABIT_SUPERVISOR_LAB')=='private-v1'
    assert all(os.readlink('/proc/self/ns/'+n)!=v for n,v in read(ROOT/'host-namespaces.json').items())
    assert [n for _,n in socket.if_nameindex()]==['lo']
def subreaper():
    libc=ctypes.CDLL(None,use_errno=True)
    assert libc.prctl(36,1,0,0,0)==0
    value=ctypes.c_int();assert libc.prctl(37,ctypes.byref(value),0,0,0)==0 and value.value==1

def identity(pid):
    try:
        fields=(Path('/proc')/str(pid)/'stat').read_text().rsplit(') ',1)[1].split()
        return {'pid':pid,'start':fields[19],'state':fields[0],'parent':int(fields[1])}
    except (FileNotFoundError,ProcessLookupError):return None

def children():
    return [int(p) for p in (Path('/proc')/str(os.getpid())/'task'/str(os.getpid())/'children').read_text().split()]

def owned_signal(pid,signum):
    # Never signal a numeric PID based only on a prior snapshot. A pidfd binds
    # the kernel object; parent/start checks limit authority to this guardian.
    before=identity(pid)
    if before is None or before['parent']!=os.getpid():return False
    try:fd=os.pidfd_open(pid)
    except ProcessLookupError:return False
    try:
        now=identity(pid)
        if now is None or now['parent']!=os.getpid() or now['start']!=before['start']:return False
        try:signal.pidfd_send_signal(fd,signum)
        except ProcessLookupError:return False
        return True
    finally:os.close(fd)

def guardian(token,mode):
    guard();assert len(token)==32 and all(c in '0123456789abcdef' for c in token)
    subreaper();path=ROOT/'receipts'/(token+'.json');assert not path.exists()
    owner={'token':token,**identity(os.getpid()),'boot':Path('/proc/sys/kernel/random/boot_id').read_text().strip(),'namespace':os.readlink('/proc/self/ns/pid')}
    state={'version':1,'owner':owner,'phase':'ready','request':None,'root':None,'reaped':[],'signals':[],'proof':None}
    def emit(event):
        save(path,state)
        try:print(json.dumps({'event':event,'owner':owner,'phase':state['phase']}),flush=True)
        except BrokenPipeError:pass
    emit('ready')
    def reap():
        while True:
            try:pid,status=os.waitpid(-1,os.WNOHANG)
            except ChildProcessError:return True
            if pid==0:return False
            state['reaped'].append({'pid':pid,'status':os.waitstatus_to_exitcode(status)})
    def stop(request):
        state['phase']='draining';state['request']=request;emit('draining')
        deadline=time.monotonic()+5;force=time.monotonic()+.3
        while True:
            # ECHILD is authoritative here only because this unchanged guardian
            # became a subreaper BEFORE launch and has now stopped admission.
            if reap():
                assert not children();state['phase']='closed';state['proof']='waitpid-echild';emit('closed');return
            for pid in children():
                signum=signal.SIGKILL if time.monotonic()>=force else signal.SIGTERM
                if owned_signal(pid,signum):state['signals'].append({'pid':pid,'signal':int(signum)})
            if time.monotonic()>deadline:
                state['phase']='unconfirmed';emit('unconfirmed');return
            time.sleep(.02)
    try:
        while True:
            raw=sys.stdin.readline(2049)
            if not raw:
                stop('controller-eof');return
            assert len(raw)<2049;request=json.loads(raw);assert set(request)=={'action','request'}
            if request['action']=='start':
                assert state['phase']=='ready';state['phase']='launching';state['request']=request['request'];emit('launching')
                command=['/usr/bin/quickshell','--no-color','--path','/lab/shell.qml'] if mode=='qml' else [sys.executable,'-B','/lab/fork-fixture.py',token,mode]
                log_path=Path(os.environ.get('OMABIT_SUPERVISOR_QML_LOG','/lab/shell.log'))
                assert log_path.parent==ROOT and (log_path.name=='shell.log' or log_path.name.startswith('shell-') and log_path.suffix=='.log')
                log=log_path.open('ab') if mode=='qml' else None
                try:child=subprocess.Popen(command,stdin=subprocess.DEVNULL,stdout=log or subprocess.DEVNULL,stderr=subprocess.STDOUT if log else subprocess.DEVNULL,close_fds=True,start_new_session=True)
                finally:
                    if log:log.close()
                state['root']=identity(child.pid);assert state['root'];state['phase']='running';emit('started')
            elif request['action']=='kill-root':
                assert mode=='qml' and state['phase']=='running' and state['root'] is not None
                assert owned_signal(state['root']['pid'],signal.SIGKILL);emit('root-signaled')
            elif request['action']=='status':
                reap();emit('status')
            elif request['action']=='stop':
                stop(request['request']);return
            else:raise AssertionError('unknown private request')
    except BaseException:
        state['phase']='unconfirmed';save(path,state);raise

FORK='''from pathlib import Path
import json,os,signal,sys,time
root=Path('/lab/descendants');token,mode=sys.argv[1:]
def record(label):
 p=Path('/proc/self/stat').read_text().rsplit(') ',1)[1].split()
 (root/(token+'-'+label+'.json')).write_text(json.dumps({'pid':os.getpid(),'start':p[19]}))
if os.fork():raise SystemExit(0)
if os.fork():raise SystemExit(0)
os.setsid()
def late(_signal,_frame):
 if mode=='late-fork' and os.fork()==0:
  signal.signal(signal.SIGTERM,signal.SIG_IGN);record('late')
  while True:time.sleep(1)
 raise SystemExit(0)
signal.signal(signal.SIGTERM,late)
if os.fork()==0:
 signal.signal(signal.SIGTERM,signal.SIG_IGN);record('leaf')
 while True:time.sleep(1)
record('daemon')
while True:time.sleep(1)
'''

def inside():
    guard();subreaper();report={'passed':False,'private_namespaces':True,'cases':[],'guardian_loss_authorizes_recovery':False};handles=[]
    def wait(fn,seconds=8):
        end=time.monotonic()+seconds
        while time.monotonic()<end:
            value=fn()
            if value:return value
            time.sleep(.02)
        raise AssertionError('private supervisor observation timed out')
    class Owner:
        def __init__(self,mode):
            self.token=uuid.uuid4().hex;self.path=ROOT/'receipts'/(self.token+'.json');self.log=(ROOT/(self.token+'.events.jsonl')).open('w');self.err=(ROOT/(self.token+'.stderr.log')).open('w')
            self.proc=subprocess.Popen([sys.executable,'-B','/lab/driver.py','--guardian',self.token,'--mode',mode],stdin=subprocess.PIPE,stdout=self.log,stderr=self.err,text=True);handles.append(self)
            wait(lambda:self.path.exists());self.owner=read(self.path)['owner'];assert self.owner['pid']==self.proc.pid
        def send(self,action):
            ident=uuid.uuid4().hex;self.proc.stdin.write(json.dumps({'action':action,'request':ident})+'\n');self.proc.stdin.flush();return ident
        def start(self):
            self.send('start');wait(lambda:read(self.path)['phase']=='running')
            wait(lambda:all((ROOT/'descendants'/(self.token+'-'+n+'.json')).exists() for n in ('daemon','leaf')))
            return [read(ROOT/'descendants'/(self.token+'-'+n+'.json')) for n in ('daemon','leaf')]
        def finish(self):
            assert self.proc.wait(8)==0;self.log.close();self.err.close();return read(self.path)
    try:
        for mode in ('double-fork','late-fork','eof'):
            owner=Owner(mode);descendants=owner.start();root=read(owner.path)['root']
            wait(lambda:identity(root['pid']) is None or identity(root['pid'])['state']=='Z')
            owner.send('status');wait(lambda:any(r['pid']==root['pid'] for r in read(owner.path)['reaped']))
            assert read(owner.path)['phase']=='running' and all(identity(r['pid']) is not None for r in descendants)
            if mode=='eof':owner.proc.stdin.close();request='controller-eof'
            else:request=owner.send('stop')
            receipt=owner.finish();assert receipt['phase']=='closed' and receipt['proof']=='waitpid-echild' and receipt['request']==request
            assert receipt['owner']==owner.owner and all(identity(r['pid']) is None for r in descendants)
            if mode=='late-fork':
                late=read(ROOT/'descendants'/(owner.token+'-late.json'));assert identity(late['pid']) is None
                assert any(row['pid']==late['pid'] for row in receipt['reaped'])
            assert any(row['signal']==signal.SIGKILL for row in receipt['signals'])
            report['cases'].append({'mode':mode,'owner':owner.owner,'request':request,'reaped_count':len(receipt['reaped']),'closed_receipt_sha256':sha(owner.path)})
        lost=Owner('double-fork');descendants=lost.start();before=read(lost.path)
        os.kill(lost.proc.pid,signal.SIGKILL);assert lost.proc.wait(5)==-signal.SIGKILL;lost.log.close();lost.err.close()
        wait(lambda:identity(descendants[0]['pid']) is not None and identity(descendants[0]['pid'])['parent']==os.getpid())
        assert all(identity(r['pid']) is not None for r in descendants)
        replacement=Owner('double-fork');replacement.send('stop');closed=replacement.finish()
        assert closed['proof']=='waitpid-echild' and closed['owner']!=before['owner']
        assert read(lost.path)['phase']=='running' and read(lost.path)['proof'] is None
        assert all(identity(r['pid']) is not None for r in descendants)
        report['cases'].append({'mode':'guardian-loss','old_owner':before['owner'],'replacement_owner':closed['owner'],'old_receipt_remains_unconfirmed':True,'new_empty_supervisor_is_not_old_cleanup':True})
        # This fixture is the next subreaper and owns the deliberately orphaned
        # test descendants. Exact owned-child cleanup does not resolve the fence.
        deadline=time.monotonic()+5
        while True:
            try:
                pid,status=os.waitpid(-1,os.WNOHANG)
                if pid:continue
            except ChildProcessError:break
            for pid in children():owned_signal(pid,signal.SIGKILL)
            assert time.monotonic()<deadline;time.sleep(.02)
        assert all(identity(r['pid']) is None for r in descendants)
        assert read(lost.path)['phase']=='running' and read(lost.path)['proof'] is None
        report['passed']=True
    finally:
        for owner in handles:
            if owner.proc.poll() is None:owner.proc.kill();owner.proc.wait(5)
        save(ROOT/'result.json',report)
    print(json.dumps({'passed':report['passed'],'cases':len(report['cases']),'production_adapter':False,'collection_fence_unchanged':True}))

def main():
    p=argparse.ArgumentParser();p.add_argument('--inside',action='store_true');p.add_argument('--guardian');p.add_argument('--mode');p.add_argument('--out');a=p.parse_args()
    if a.guardian:return guardian(a.guardian,a.mode)
    if a.inside:return inside()
    out=Path(a.out).absolute();assert out.parent==Path('/tmp') and out.name.startswith('omabit-collection-supervisor-') and not out.exists();out.mkdir(mode=0o700)
    for n in ('home','config','data','state','cache','runtime','etc','receipts','descendants'):(out/n).mkdir(mode=0o700)
    shutil.copy2(__file__,out/'driver.py');(out/'fork-fixture.py').write_text(FORK)
    save(out/'source.json',{'fixture_sha256':sha(Path(__file__)),'child_sha256':sha(out/'fork-fixture.py')})
    save(out/'host-namespaces.json',{n:os.readlink('/proc/self/ns/'+n) for n in ('user','net','pid','mnt')})
    (out/'etc/passwd').write_text('lab:x:'+str(os.getuid())+':'+str(os.getgid())+':Lab:/lab/home:/bin/sh\n');(out/'etc/group').write_text('lab:x:'+str(os.getgid())+':\n')
    cmd=['bwrap','--unshare-all','--die-with-parent','--clearenv','--ro-bind','/usr','/usr','--symlink','usr/bin','/bin','--symlink','usr/lib','/lib','--symlink','usr/lib','/lib64','--proc','/proc','--dev','/dev','--tmpfs','/tmp','--ro-bind',str(out/'etc'),'/etc','--bind',str(out),'/lab','--chdir','/lab']
    env={'HOME':'/lab/home','PATH':'/usr/bin:/bin','LANG':'C.UTF-8','PYTHONDONTWRITEBYTECODE':'1','XDG_RUNTIME_DIR':'/lab/runtime','OMABIT_SUPERVISOR_LAB':'private-v1'}
    for n in ('CONFIG','DATA','STATE','CACHE'):env['XDG_'+n+'_HOME']='/lab/'+n.lower()
    for k,v in env.items():cmd+=['--setenv',k,v]
    cmd+=['--','/usr/bin/python3','-B','/lab/driver.py','--inside'];save(out/'command.json',cmd)
    q=subprocess.run(cmd,capture_output=True,text=True,timeout=60);(out/'driver.log').write_text(q.stdout+q.stderr);print(q.stdout+q.stderr);raise SystemExit(q.returncode)
if __name__=='__main__':main()
