#!/usr/bin/env python3
"""Fresh fake Urbit + real Theme clients/CLI/keyring; private relay drops one delete reply."""
from pathlib import Path
import argparse,hashlib,http.client,http.server,http.cookiejar,json,os,re,resource,shutil,signal,socket,subprocess,sys,threading,time,urllib.error,urllib.request
CLIENT="from pathlib import Path\nimport json,os,sys,threading\nassert Path.home()==Path('/lab/home') and not Path('/home').exists() and os.environ.get('OMABIT_THEME_URBIT_LAB')=='private-v1'\nsys.path.insert(0,'/lab/source')\nfrom client.eyre import Eyre\nfrom client.support import Keyring,Failure\nfrom client.instance import INSTANCE\nassert INSTANCE.profile=='qualification-urbit'\nurl='http://127.0.0.1:18082';session=Keyring().lookup(url,'~zod');eyre=Eyre(url,session,timeout=15)\nready=threading.Event()\ndef report():\n if ready.wait(20):print(json.dumps({'event':'ready'}),flush=True)\nthreading.Thread(target=report,daemon=True).start()\ntry:eyre.watch('talon',threading.Event(),threading.Event(),ready)\nexcept Failure as exc:print(json.dumps({'event':'error','code':exc.code}),flush=True);raise SystemExit(1)\n"
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
    assert os.environ.get('OMABIT_THEME_URBIT_LAB')=='private-v1'
    assert all(os.readlink('/proc/self/ns/'+n)!=v for n,v in read(root/'host-namespaces.json').items())
    assert [n for _,n in socket.if_nameindex()]==['lo']
    assert sha(root/'runtime-inputs/urbit')==VERE_SHA and sha(root/'runtime-inputs/boot.pill')==PILL_SHA
    sys.path.insert(0,str(root/'source'))
    from client.support import StateStore,Keyring,Failure
    from client.transport_journal import TransportJournal
    from client.instance import INSTANCE
    from client.eyre import Eyre
    assert INSTANCE.plugin_id=='thelifeandtimes.urbit-theme.qualification-urbit'
    store=StateStore();journal=TransportJournal();helpers=[];keyring=None;child=None;hub=None
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
            super().__init__(('127.0.0.1',18082),Handler);self.loss=set();self.accepted_delete=threading.Event();self.release_delete=threading.Event();self.deleted_404=[];self.actions=[];self.frames=[];self.errors=[];self.lock=threading.RLock();self.thread=threading.Thread(target=self.serve_forever,daemon=True);self.thread.start()
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
            if self.path=='/~/logout':
                for row in journal.status()['channels']:
                    assert row['phase']=='closed'
                    check=http.client.HTTPConnection('127.0.0.1',18081,timeout=3)
                    try:
                        check.request('GET','/~/channel/'+row['channel'],headers=headers)
                        verified=check.getresponse();assert verified.status==404;verified.read(16384)
                        s.deleted_404.append(row['channel'])
                    finally:check.close()
            try:
                connection.request(self.command,self.path,body=body,headers=headers);response=connection.getresponse()
                with s.lock:
                    for m in messages:s.actions.append({'path':self.path,'action':m['action'],'id':m.get('id'),'status':response.status})
                streaming='text/event-stream' in response.getheader('Content-Type','')
                if not streaming:
                    data=response.read(131073);assert len(data)<=131072
                    if self.path in s.loss and any(m['action']=='delete' for m in messages):
                        assert response.status==204 and data==b''
                        s.accepted_delete.set();assert s.release_delete.wait(10)
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
        def __init__(self,name,mode='watch'):
            self.name=name;self.events=[];self.err=(root/(name+'.stderr.log')).open('wb');self.log=(root/(name+'.events.jsonl')).open('w')
            command=[sys.executable,'-B','/lab/source/client/main.py','disconnect'] if mode=='disconnect' else [sys.executable,'-B','/lab/theme-client.py']
            self.proc=subprocess.Popen(command,stdin=subprocess.PIPE,stdout=subprocess.PIPE,stderr=self.err,text=True)
            if mode=='disconnect':self.proc.stdin.write(json.dumps({'expectedAccount':account}));self.proc.stdin.close()
            self.start=Path('/proc/'+str(self.proc.pid)+'/stat').read_text().rsplit(') ',1)[1].split()[19];helpers.append(self)
            def reader():
                for line in self.proc.stdout:self.log.write(line);self.log.flush();self.events.append(json.loads(line))
            self.thread=threading.Thread(target=reader,daemon=True);self.thread.start()
        def kill(self):
            assert self.proc.poll() is None and Path('/proc/'+str(self.proc.pid)+'/stat').read_text().rsplit(') ',1)[1].split()[19]==self.start
            self.proc.send_signal(signal.SIGKILL);assert self.proc.wait(5)==-signal.SIGKILL;record('exact-client-killed',name=self.name,pid=self.proc.pid,start=self.start)
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
        hub=Relay()
        logged=subprocess.run([sys.executable,'-B','/lab/source/client/main.py','login'],input=json.dumps({'url':'http://127.0.0.1:18082','code':code}),capture_output=True,text=True,timeout=20);del code
        result=json.loads(logged.stdout);assert logged.returncode==0 and result['ok']
        account={k:result['state']['ships'][0][k] for k in ('id','url','ship')};session=Keyring().lookup(account['url'],'~zod')
        first=Helper('first');wait(lambda:any(e.get('event')=='ready' for e in first.events) and live_path() and len(hub.subscriptions(live_path()))==1)
        opened=snapshot('01-open')['channels'][0];oldpath='/~/channel/'+opened['channel']
        record('first-real-subscription-confirmed');first.kill();first.close_logs()
        hub.loss.add(oldpath);second=Helper('lost-delete');assert hub.accepted_delete.wait(15)
        closing=snapshot('02-closing-lost-reply')['channels'][0]
        assert closing['phase']=='closing' and len(journal.status()['channels'])==1
        assert any(a['action']=='delete' and a['path']==oldpath and a['status']==204 for a in hub.actions)
        second.kill();second.close_logs();hub.release_delete.set();hub.loss.clear();third=Helper('recovered')
        wait(lambda:any(e.get('event')=='ready' for e in third.events) and live_path()!=oldpath and len(hub.subscriptions(live_path()))==1)
        recovered=snapshot('03-recovered');closed,new=recovered['channels']
        assert closed['phase']=='closed' and closed['cleanup_request']==closing['cleanup_request'] and closed['cleanup_actor']!=closing['cleanup_actor']
        assert Keyring().lookup(session['url'],'~zod')==session
        assert {k:store.load()['ships'][0][k] for k in ('id','url','ship')}==account
        assert isinstance(Eyre(session['url'],session).scry('talon'),dict)
        record('real-delete-retry-and-subscription-recovered',same_session=True,same_cleanup_request=True)
        disconnect=Helper('disconnect','disconnect');assert disconnect.proc.wait(60)==0;disconnect.close_logs()
        assert disconnect.events[0]['ok'] and disconnect.events[0]['warning'] is None and disconnect.events[0]['state']=={'ships':[]}
        assert third.proc.wait(5)==1;third.close_logs()
        final=snapshot('04-closed');assert all(row['phase']=='closed' for row in final['channels'])
        assert all(row['retired'] for row in final['sessions']) and store.load()['ships']==[]
        try:Keyring().lookup(session['url'],'~zod')
        except Failure:pass
        else:raise AssertionError('Private credential was not removed')
        assert len(hub.deleted_404)==2 and not hub.errors and not any(a['action']=='poke' for a in hub.actions)
        report.update(passed=True,actions=hub.actions,frames=hub.frames,deleted_channels_404=hub.deleted_404,application_pokes=0,
            session_preserved_through_recovery=True,private_session_revoked=True,actual_disconnect_cli=True,
            helper_exit_codes={p.name:p.proc.returncode for p in helpers},final_receipt=disconnect.events[0])
    except Exception as exc:
        report['error']=type(exc).__name__+': '+str(exc);raise
    finally:
        for helper in helpers:
            if helper.proc.poll() is None:helper.kill();helper.close_logs()
        if hub:
            hub.release_delete.set()
            report['actions']=hub.actions;report['frames']=hub.frames;report['relay_errors']=hub.errors;hub.shutdown();hub.server_close();hub.thread.join(3)
        if child:
            running=child.poll() is None
            if running:child.send_signal(signal.SIGTERM)
            try:exit_code=child.wait(45);report['clean_shutdown']=running and exit_code==0
            except subprocess.TimeoutExpired:child.kill();child.wait(5);report['clean_shutdown']=False
            capture_thread.join(3);report['urbit_exit']=child.returncode;report['log_truncated']=overflow.is_set()
        if keyring:keyring.terminate();keyring.wait(5)
        report['passed']=report['passed'] and report['clean_shutdown'] and not report.get('log_truncated',False);save(root/'result.json',report)
    assert report['passed'];print(json.dumps({'passed':True,'real_subscriptions':2,'deleted_channels_404':len(report['deleted_channels_404']),'clean_shutdown':report['clean_shutdown']}))


def main():
    p=argparse.ArgumentParser();p.add_argument('--inside',action='store_true');p.add_argument('--source');p.add_argument('--vere');p.add_argument('--pill');p.add_argument('--out');a=p.parse_args()
    if a.inside:return inside()
    src=Path(a.source).resolve();vere=Path(a.vere).resolve();pill=Path(a.pill).resolve();out=Path(a.out).absolute()
    assert sha(vere)==VERE_SHA and sha(pill)==PILL_SHA
    assert out.parent==Path('/tmp') and out.name.startswith('omabit-theme-urbit-transport-') and not out.exists()
    memory=int(next(row.split()[1] for row in Path('/proc/meminfo').read_text().splitlines() if row.startswith('MemAvailable:')))*1024;free=shutil.disk_usage(out.parent).free
    assert memory>10*1024**3 and free>4*1024**3
    out.mkdir(mode=0o700)
    for n in ('source','runtime-inputs','home','config','data','state','cache','runtime','etc','writer-fixtures'):(out/n).mkdir(mode=0o700)
    hashes={}
    for f in list((src/'client').glob('*.py'))+[src/'manifest.json',src/'application-connection.json']:
        assert f.is_file() and not f.is_symlink();target=out/'source'/f.relative_to(src);target.parent.mkdir(parents=True,exist_ok=True);shutil.copy2(f,target);hashes[str(f)]=sha(f)
    m=read(out/'source/manifest.json');m['id']='thelifeandtimes.urbit-theme.qualification-urbit';m['name']+=' (Urbit transport qualification)';save(out/'source/manifest.json',m);save(out/'source/application-profile.json',{'version':1,'profile':'qualification-urbit'})
    (out/'theme-client.py').write_text(CLIENT)
    shutil.copy2(vere,out/'runtime-inputs/urbit');shutil.copy2(pill,out/'runtime-inputs/boot.pill');shutil.copy2(__file__,out/'driver.py')
    save(out/'source.json',{'original_sources':hashes,'fixture_sha256':sha(Path(__file__)),'client_sha256':sha(out/'theme-client.py'),'vere_sha256':VERE_SHA,'pill_sha256':PILL_SHA,'actual_theme_clients_unchanged':True,'memory_available':memory,'storage_free':free})
    save(out/'host-namespaces.json',{n:os.readlink('/proc/self/ns/'+n) for n in ('user','net','pid','mnt')});(out/'etc/passwd').write_text('lab:x:'+str(os.getuid())+':'+str(os.getgid())+':Lab:/lab/home:/bin/sh\n');(out/'etc/group').write_text('lab:x:'+str(os.getgid())+':\n')
    cmd=['bwrap','--unshare-all','--die-with-parent','--clearenv','--ro-bind','/usr','/usr','--symlink','usr/bin','/bin','--symlink','usr/lib','/lib','--symlink','usr/lib','/lib64','--proc','/proc','--dev','/dev','--tmpfs','/tmp','--ro-bind',str(out/'etc'),'/etc','--bind',str(out),'/lab','--ro-bind',str(out/'source'),'/lab/source','--ro-bind',str(out/'runtime-inputs'),'/lab/runtime-inputs','--chdir','/lab']
    env={'HOME':'/lab/home','PATH':'/usr/bin:/bin','LANG':'C.UTF-8','PYTHONDONTWRITEBYTECODE':'1','XDG_RUNTIME_DIR':'/lab/runtime','OMABIT_THEME_URBIT_LAB':'private-v1'}
    for n in ('CONFIG','DATA','STATE','CACHE'):env['XDG_'+n+'_HOME']='/lab/'+n.lower()
    for k,v in env.items():cmd+=['--setenv',k,v]
    cmd+=['--','/usr/bin/dbus-run-session','--','/usr/bin/python3','-B','/lab/driver.py','--inside'];save(out/'command.json',cmd)
    q=subprocess.run(cmd,capture_output=True,text=True,timeout=420);(out/'driver.log').write_text(q.stdout+q.stderr);print(q.stdout+q.stderr);raise SystemExit(q.returncode)

if __name__=='__main__':main()
