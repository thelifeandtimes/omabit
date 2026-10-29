#!/usr/bin/env python3
"""Fresh fake Urbit + real DMs helper/keyring; private relay drops one delete reply."""
from pathlib import Path
import argparse,hashlib,http.client,http.server,http.cookiejar,json,os,re,resource,shutil,signal,socket,subprocess,sys,threading,time,urllib.error,urllib.request
VERE_SHA='b5c54439ae5b9f0ff94d75319e67e04f04369776fa5e2180a5ab1a5ebd4c2ff0'
PILL_SHA='f8b08c53dd6d0aa9c4f7654f0a7bcd11e94089222888a3c99c0a10eedcbb377b'
def sha(path):
    with path.open('rb') as f:return hashlib.file_digest(f,'sha256').hexdigest()
def read(path):return json.loads(path.read_text())
def save(path,v):path.write_text(json.dumps(v,indent=2)+'\n')
def limits():
    os.nice(15);resource.setrlimit(resource.RLIMIT_CORE,(0,0));resource.setrlimit(resource.RLIMIT_CPU,(240,240))

def inside():
    root=Path('/lab');assert Path.home()==root/'home' and not Path('/home').exists() and not Path('/run/user').exists()
    assert os.environ.get('OMABIT_DMS_URBIT_LAB')=='private-v1'
    assert all(os.readlink('/proc/self/ns/'+n)!=v for n,v in read(root/'host-namespaces.json').items())
    assert [n for _,n in socket.if_nameindex()]==['lo']
    assert sha(root/'runtime-inputs/urbit')==VERE_SHA and sha(root/'runtime-inputs/boot.pill')==PILL_SHA
    sys.path.insert(0,str(root/'source'))
    from session_store import SessionStore,cookie_for
    from transport_journal import TransportJournal
    from eyre_session import revoke
    from instance import INSTANCE
    assert INSTANCE.plugin_id=='urbit.dms.qualification-urbit'
    store=SessionStore();journal=TransportJournal();helpers=[];keyring=None;child=None;hub=None
    report={'passed':False,'real_urbit':True,'private_namespaces':True,'events':[],'clean_shutdown':False}
    def record(event,**fields):report['events'].append({'event':event,**fields});save(root/'result.json',report)
    def wait(fn,seconds=30):
        deadline=time.monotonic()+seconds
        while time.monotonic()<deadline:
            value=fn()
            if value:return value
            time.sleep(.05)
        raise AssertionError('fresh Urbit transport observation timed out')
    direct=urllib.request.build_opener(urllib.request.ProxyHandler({}))
    def lens(expr):
        req=urllib.request.Request('http://127.0.0.1:12321/',json.dumps({'source':{'dojo':expr},'sink':{'stdout':None}}).encode(),headers={'Content-Type':'application/json'})
        with direct.open(req,timeout=20) as response:
            assert response.status==200;return json.loads(response.read(16384))
    class Relay(http.server.ThreadingHTTPServer):
        daemon_threads=True
        def __init__(self):
            super().__init__(('127.0.0.1',18082),Handler);self.loss=set();self.actions=[];self.frames=[];self.errors=[];self.lock=threading.RLock();self.thread=threading.Thread(target=self.serve_forever,daemon=True);self.thread.start()
        def subscriptions(self,path):
            with self.lock:
                ids={a['id'] for a in self.actions if a['path']==path and a['action']=='subscribe' and a['status']==204}
                return {f['id'] for f in self.frames if f['path']==path and f.get('response')=='subscribe' and f.get('ok')=='ok'}&ids
    class Handler(http.server.BaseHTTPRequestHandler):
        protocol_version='HTTP/1.1'
        def log_message(self,*a):pass
        def forward(self):
            s=self.server;connection=http.client.HTTPConnection('127.0.0.1',18081,timeout=70)
            body=self.rfile.read(int(self.headers.get('Content-Length','0'))) if self.command in ('PUT','POST') else None
            assert body is None or len(body)<=65536
            messages=json.loads(body) if self.command=='PUT' and self.path.startswith('/~/channel/') else []
            if messages:
                assert all(m['action'] in ('subscribe','ack','delete') for m in messages),'Application write forbidden in real transport gate'
                row=next((r for r in journal.status()['channels'] if '/~/channel/'+r['channel']==self.path),None)
                assert row is not None
                if any(m['action']=='delete' for m in messages):assert row['phase']=='closing'
            headers={k:v for k,v in self.headers.items() if k.lower() not in ('host','connection','content-length')}
            headers['Connection']='close'
            try:
                connection.request(self.command,self.path,body=body,headers=headers);response=connection.getresponse()
                with s.lock:
                    for m in messages:s.actions.append({'path':self.path,'action':m['action'],'id':m.get('id'),'status':response.status})
                streaming='text/event-stream' in response.getheader('Content-Type','')
                if not streaming:
                    data=response.read(131073);assert len(data)<=131072
                    if self.path in s.loss and any(m['action']=='delete' for m in messages):
                        assert response.status==204 and data==b''
                        self.close_connection=True;self.connection.shutdown(socket.SHUT_RDWR);self.connection.close();return
                    self.send_response(response.status)
                    for k,v in response.getheaders():
                        if k.lower() not in ('content-length','transfer-encoding','connection'):self.send_header(k,v)
                    self.send_header('Content-Length',str(len(data)));self.send_header('Connection','close');self.end_headers();self.close_connection=True;self.wfile.write(data)
                else:
                    self.send_response(response.status);self.send_header('Content-Type','text/event-stream');self.send_header('Connection','close');self.end_headers();self.close_connection=True
                    data=[]
                    while True:
                        line=response.readline(65537)
                        if not line:break
                        assert len(line)<=65536;self.wfile.write(line);self.wfile.flush()
                        clean=line.strip()
                        if clean.startswith(b'data:'):data.append(clean[5:].lstrip())
                        elif not clean and data:
                            frame=json.loads(b'\n'.join(data));data=[]
                            with s.lock:s.frames.append({'path':self.path,**{k:frame.get(k) for k in ('id','response','ok','err')}})
            except (BrokenPipeError,ConnectionResetError):pass
            except Exception as exc:
                with s.lock:s.errors.append(type(exc).__name__)
                raise
            finally:connection.close()
        do_GET=do_POST=do_PUT=forward
    class Helper:
        def __init__(self,name):
            self.name=name;self.events=[];self.err=(root/(name+'.stderr.log')).open('wb');self.log=(root/(name+'.events.jsonl')).open('w')
            self.proc=subprocess.Popen([sys.executable,'-B','/lab/source/bridge.py'],stdin=subprocess.PIPE,stdout=subprocess.PIPE,stderr=self.err,text=True)
            self.start=Path('/proc/'+str(self.proc.pid)+'/stat').read_text().split(') ',1)[1].split()[19];helpers.append(self)
            def reader():
                for line in self.proc.stdout:self.log.write(line);self.log.flush();self.events.append(json.loads(line))
            self.thread=threading.Thread(target=reader,daemon=True);self.thread.start()
        def command(self,action,**fields):self.proc.stdin.write(json.dumps({'action':action,**fields})+'\n');self.proc.stdin.flush()
        def kill(self):
            assert self.proc.poll() is None and Path('/proc/'+str(self.proc.pid)+'/stat').read_text().split(') ',1)[1].split()[19]==self.start
            self.proc.send_signal(signal.SIGKILL);assert self.proc.wait(5)==-signal.SIGKILL;record('exact-helper-killed',name=self.name,pid=self.proc.pid,start=self.start)
        def close_logs(self):self.thread.join(3);self.err.close();self.log.close()
    def snapshot(name):v=journal.status();save(root/'writer-fixtures'/(name+'.json'),v);return v
    def live_path():
        v=journal.status()
        if v is None or not v['channels'] or v['channels'][-1]['phase']!='open':return None
        return '/~/channel/'+v['channels'][-1]['channel']
    try:
        keyring=subprocess.Popen(['/usr/bin/gnome-keyring-daemon','--foreground','--unlock','--components=secrets'],stdin=subprocess.PIPE,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL);keyring.stdin.write(os.urandom(32).hex().encode());keyring.stdin.close()
        command=['/lab/runtime-inputs/urbit','-t','-L','--no-conn','--no-dock','--loom','31','--snap-time','60','-b','127.0.0.1','--http-port','18081','-F','zod','-B','/lab/runtime-inputs/boot.pill','-c','/lab/zod']
        save(root/'launch.json',command);child=subprocess.Popen(command,stdin=subprocess.DEVNULL,stdout=subprocess.PIPE,stderr=subprocess.STDOUT,start_new_session=True,preexec_fn=limits)
        overflow=threading.Event()
        def capture():
            total=0
            with (root/'vere.log').open('wb') as f:
                while chunk:=child.stdout.read(65536):
                    f.write(chunk[:max(0,8*1024**2-total)]);f.flush();total+=len(chunk)
                    if total>8*1024**2:overflow.set()
        capture_thread=threading.Thread(target=capture,daemon=True);capture_thread.start()
        def ready():
            assert child.poll() is None and not overflow.is_set()
            try:
                with direct.open('http://127.0.0.1:18081/~/host',timeout=1) as response:return response.read(256).strip()==b'~zod'
            except (OSError,urllib.error.URLError):return False
        wait(ready,240);assert lens('our').strip()=='~zod';record('fresh-fake-ready');print('Fresh isolated fake ready',flush=True)
        code=lens('+code').strip().removeprefix('~');assert re.fullmatch('[a-z]{6}(?:-[a-z]{6}){3}',code)
        hub=Relay();first=Helper('first');wait(lambda:any(e['event']=='ready' for e in first.events));first.command('login',url='http://127.0.0.1:18082',code=code,request=1);del code
        wait(lambda:live_path() and len(hub.subscriptions(live_path()))==4)
        opened=snapshot('01-open')['channels'][0];oldpath='/~/channel/'+opened['channel'];session=store.lookup('http://127.0.0.1:18082','~zod');account=store.account()
        assert any(e['event']=='login' for e in first.events);record('four-real-subscriptions-confirmed');first.kill();first.close_logs()
        hub.loss.add(oldpath);second=Helper('lost-delete');wait(lambda:any(e['event']=='error' for e in second.events))
        closing=snapshot('02-closing-lost-reply')['channels'][0]
        assert closing['phase']=='closing' and len(journal.status()['channels'])==1
        assert any(a['action']=='delete' and a['path']==oldpath and a['status']==204 for a in hub.actions)
        second.kill();second.close_logs();hub.loss.clear();third=Helper('recovered')
        wait(lambda:live_path() and live_path()!=oldpath and len(hub.subscriptions(live_path()))==4)
        recovered=snapshot('03-recovered');closed,new=recovered['channels']
        assert closed['phase']=='closed' and closed['cleanup_request']==closing['cleanup_request'] and closed['cleanup_actor']!=closing['cleanup_actor']
        assert store.lookup(session['url'],'~zod')==session and store.account()==account
        record('real-delete-retry-and-four-subscriptions-recovered',same_session=True,same_cleanup_request=True)
        third.command('shutdown-transport');wait(lambda:any(e['event']=='transport-state' for e in third.events));assert third.proc.wait(10)==0;third.close_logs()
        receipt=next(e for e in third.events if e['event']=='transport-state');assert receipt['closed'] and not receipt['collectionReady']
        final=snapshot('04-closed');assert all(row['phase']=='closed' for row in final['channels'])
        # Only after confirmed DELETE, independent GET must return actual Eyre404.
        jar=http.cookiejar.CookieJar();jar.set_cookie(cookie_for(session));check=urllib.request.build_opener(urllib.request.ProxyHandler({}),urllib.request.HTTPCookieProcessor(jar))
        absent=[]
        for row in final['channels']:
            try:
                with check.open(row['origin']+'/~/channel/'+row['channel'],timeout=3) as response:raise AssertionError('deleted channel still available')
            except urllib.error.HTTPError as error:assert error.code==404;absent.append(row['channel'])
        assert len(absent)==2 and not hub.errors and not any(a['action']=='poke' for a in hub.actions)
        assert revoke(session) in ('revoked','already-revoked')
        report.update(passed=True,actions=hub.actions,frames=hub.frames,deleted_channels_404=absent,application_pokes=0,session_preserved_through_recovery=True,private_session_revoked=True,helper_exit_codes={p.name:p.proc.returncode for p in helpers},final_receipt=receipt)
    except Exception as exc:
        report['error']=type(exc).__name__+': '+str(exc);raise
    finally:
        for helper in helpers:
            if helper.proc.poll() is None:helper.kill();helper.close_logs()
        if hub:
            report['actions']=hub.actions;report['frames']=hub.frames;report['relay_errors']=hub.errors;hub.shutdown();hub.server_close();hub.thread.join(3)
        if child:
            running=child.poll() is None
            if running:child.send_signal(signal.SIGTERM)
            try:exit_code=child.wait(45);report['clean_shutdown']=running and exit_code==0
            except subprocess.TimeoutExpired:child.kill();child.wait(5);report['clean_shutdown']=False
            capture_thread.join(3);report['urbit_exit']=child.returncode;report['log_truncated']=overflow.is_set()
        if keyring:keyring.terminate();keyring.wait(5)
        report['passed']=report['passed'] and report['clean_shutdown'] and not report.get('log_truncated',False);save(root/'result.json',report)
    assert report['passed'];print(json.dumps({'passed':True,'real_subscriptions':8,'deleted_channels_404':len(report['deleted_channels_404']),'clean_shutdown':report['clean_shutdown']}))


def main():
    p=argparse.ArgumentParser();p.add_argument('--inside',action='store_true');p.add_argument('--source');p.add_argument('--vere');p.add_argument('--pill');p.add_argument('--out');a=p.parse_args()
    if a.inside:return inside()
    src=Path(a.source).resolve();vere=Path(a.vere).resolve();pill=Path(a.pill).resolve();out=Path(a.out).absolute()
    assert sha(vere)==VERE_SHA and sha(pill)==PILL_SHA
    assert out.parent==Path('/tmp') and out.name.startswith('omabit-dms-urbit-transport-') and not out.exists()
    memory=int(next(row.split()[1] for row in Path('/proc/meminfo').read_text().splitlines() if row.startswith('MemAvailable:')))*1024;free=shutil.disk_usage(out.parent).free
    assert memory>10*1024**3 and free>4*1024**3
    out.mkdir(mode=0o700)
    for n in ('source','runtime-inputs','home','config','data','state','cache','runtime','etc','writer-fixtures'):(out/n).mkdir(mode=0o700)
    hashes={}
    for f in [p for p in src.glob('*.py') if not p.name.startswith('test_')]+[src/'manifest.json',src/'application-connection.json']:
        assert f.is_file() and not f.is_symlink();shutil.copy2(f,out/'source'/f.name);hashes[str(f)]=sha(f)
    m=read(out/'source/manifest.json');m['id']='urbit.dms.qualification-urbit';m['name']+=' (Urbit transport qualification)';save(out/'source/manifest.json',m);save(out/'source/application-profile.json',{'version':1,'profile':'qualification-urbit'})
    shutil.copy2(vere,out/'runtime-inputs/urbit');shutil.copy2(pill,out/'runtime-inputs/boot.pill');shutil.copy2(__file__,out/'driver.py')
    save(out/'source.json',{'original_sources':hashes,'fixture_sha256':sha(Path(__file__)),'vere_sha256':VERE_SHA,'pill_sha256':PILL_SHA,'actual_dms_helper_unchanged':True,'memory_available':memory,'storage_free':free})
    save(out/'host-namespaces.json',{n:os.readlink('/proc/self/ns/'+n) for n in ('user','net','pid','mnt')});(out/'etc/passwd').write_text('lab:x:'+str(os.getuid())+':'+str(os.getgid())+':Lab:/lab/home:/bin/sh\n');(out/'etc/group').write_text('lab:x:'+str(os.getgid())+':\n')
    cmd=['bwrap','--unshare-all','--die-with-parent','--clearenv','--ro-bind','/usr','/usr','--symlink','usr/bin','/bin','--symlink','usr/lib','/lib','--symlink','usr/lib','/lib64','--proc','/proc','--dev','/dev','--tmpfs','/tmp','--ro-bind',str(out/'etc'),'/etc','--bind',str(out),'/lab','--ro-bind',str(out/'source'),'/lab/source','--ro-bind',str(out/'runtime-inputs'),'/lab/runtime-inputs','--chdir','/lab']
    env={'HOME':'/lab/home','PATH':'/usr/bin:/bin','LANG':'C.UTF-8','PYTHONDONTWRITEBYTECODE':'1','XDG_RUNTIME_DIR':'/lab/runtime','OMABIT_DMS_URBIT_LAB':'private-v1'}
    for n in ('CONFIG','DATA','STATE','CACHE'):env['XDG_'+n+'_HOME']='/lab/'+n.lower()
    for k,v in env.items():cmd+=['--setenv',k,v]
    cmd+=['--','/usr/bin/dbus-run-session','--','/usr/bin/python3','-B','/lab/driver.py','--inside'];save(out/'command.json',cmd)
    q=subprocess.run(cmd,capture_output=True,text=True,timeout=420);(out/'driver.log').write_text(q.stdout+q.stderr);print(q.stdout+q.stderr);raise SystemExit(q.returncode)

if __name__=='__main__':main()
