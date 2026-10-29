#!/usr/bin/env python3
"""Concurrent actual Theme Eyre clients and private keyring, synthetic loopback peer."""
from pathlib import Path
import argparse,hashlib,json,os,shutil,signal,socket,subprocess,sys,threading,time
CHILD='''from pathlib import Path
import json,os,sys,threading
assert Path.home()==Path('/lab/home') and not Path('/home').exists() and os.environ.get('OMABIT_THEME_TRANSPORT_LAB')=='private-v1'
sys.path.insert(0,'/lab/source')
from client.eyre import Eyre
from client.support import Keyring,Failure
from client.instance import INSTANCE
assert INSTANCE.profile=='qualification-transport'
url=sys.argv[2];client=Eyre(url,Keyring().lookup(url,'~zod'),timeout=15,ack_timeout=15)
def emit(**v):print(json.dumps(v),flush=True)
try:
 if sys.argv[1]=='watch':
  stop=threading.Event();changed=threading.Event();ready=threading.Event()
  def observe():
   if ready.wait(15):emit(event='ready')
  def commands():
   if sys.stdin.readline().strip()=='stop':stop.set()
  threading.Thread(target=observe,daemon=True).start();threading.Thread(target=commands,daemon=True).start()
  client.watch('talon',stop,changed,ready);emit(event='closed')
 elif sys.argv[1]=='publish':
  client.poke('themes',{'privateProbe':'uncertain'});emit(event='published')
 elif sys.argv[1]=='read':
  client.scry();emit(event='read')
 else:raise AssertionError('unknown fixture mode')
except Failure as exc:
 emit(event='error',code=exc.code);raise SystemExit(1)
'''

def sha(path):return hashlib.sha256(path.read_bytes()).hexdigest()
def read(path):return json.loads(path.read_text())
def save(path,value):path.write_text(json.dumps(value,indent=2)+'\n')

def inside():
    root=Path('/lab');assert Path.home()==root/'home' and not Path('/home').exists() and not Path('/run/user').exists()
    assert os.environ.get('OMABIT_THEME_TRANSPORT_LAB')=='private-v1'
    assert all(os.readlink('/proc/self/ns/'+n)!=v for n,v in read(root/'host-namespaces.json').items())
    assert [n for _,n in socket.if_nameindex()]==['lo']
    sys.path[:0]=['/lab/source','/lab'];import connected_fixture as peer
    from client.support import Keyring
    from client.transport_journal import TransportJournal
    from client.instance import INSTANCE
    assert INSTANCE.plugin_id=='thelifeandtimes.urbit-theme.qualification-transport'
    journal=TransportJournal();report={'passed':False,'private_namespaces':True,'actual_theme_clients':True,'real_private_keyring':True,'synthetic_eyre':True,'events':[]};children=[];keyring=None
    def record(event,**data):report['events'].append({'event':event,**data});save(root/'result.json',report)
    def wait(fn,seconds=15):
        end=time.monotonic()+seconds
        while time.monotonic()<end:
            value=fn()
            if value:return value
            time.sleep(.03)
        raise AssertionError('private Theme transport observation timed out')
    class Handler(peer.Handler):
        def reply(self,*args,**kwargs):
            try:return super().reply(*args,**kwargs)
            except (BrokenPipeError,ConnectionResetError):return
        def do_GET(self):
            if self.path=='/~/scry/settings/desk/talon.json':return self.reply({'desk':{'ui-prefs':self.server.settings}})
            return super().do_GET()
        def do_PUT(self):
            s=self.server;owner=self.owner();assert owner=='stable-theme'
            messages=json.loads(self.rfile.read(int(self.headers.get('Content-Length','0'))));hold=False
            row=next((r for r in journal.status()['channels'] if '/~/channel/'+r['channel']==self.path),None);assert row
            with s.cond:
                c=s.channels.setdefault(self.path,{'owner':owner,'frames':[],'subscriptions':0,'active':0,'deleted':False,'delivered':0})
                for m in messages:
                    action=m['action'];assert action in ('poke','subscribe','ack','delete')
                    s.requests.append({'action':action,'channel':self.path});assert len(s.requests)<200
                    if action=='poke':
                        assert row['kind']=='poke' and row['outcome']=='uncertain';put=m['json']['put-entry'];assert (put['desk'],put['bucket-key'],put['entry-key'])==('talon','ui-prefs','themes')
                        assert not s.pokes,'Uncertain settings publication replayed';s.settings['themes']=put['value'];s.pokes.append({'channel':self.path,'accepted_without_ack':True})
                    elif action=='subscribe':
                        assert row['kind']=='watch';c['subscriptions']+=1;s.sequence+=1;c['frames'].append((s.sequence,{'id':m['id'],'response':'subscribe','ok':'ok'}))
                    elif action=='delete':
                        assert row['phase']=='closing';c['deleted']=True
                        if self.path==s.held_path:hold=True;s.accepted_delete.set()
                s.cond.notify_all()
            if hold:
                assert s.release_delete.wait(10)
                self.close_connection=True
                try:self.connection.shutdown(socket.SHUT_RDWR)
                except OSError:pass
                self.connection.close();return
            self.reply(status=204)
    class Hub(peer.Hub):
        def __init__(self):
            self.settings={};self.held_path=None;self.accepted_delete=threading.Event();self.release_delete=threading.Event();super().__init__();self.RequestHandlerClass=Handler
    hub=Hub()
    class Client:
        def __init__(self,name,mode):
            self.name=name;self.events=[];self.out=(root/(name+'.events.jsonl')).open('w');self.err=(root/(name+'.stderr.log')).open('wb')
            self.proc=subprocess.Popen([sys.executable,'-B','/lab/child.py',mode,hub.url],stdin=subprocess.PIPE,stdout=subprocess.PIPE,stderr=self.err,text=True)
            self.start=Path('/proc/'+str(self.proc.pid)+'/stat').read_text().rsplit(') ',1)[1].split()[19];children.append(self)
            def observe():
                for line in self.proc.stdout:self.out.write(line);self.out.flush();self.events.append(json.loads(line))
            self.reader=threading.Thread(target=observe,daemon=True);self.reader.start()
        def kill(self):
            assert self.proc.poll() is None and Path('/proc/'+str(self.proc.pid)+'/stat').read_text().rsplit(') ',1)[1].split()[19]==self.start
            self.proc.send_signal(signal.SIGKILL);assert self.proc.wait(5)==-signal.SIGKILL;record('exact-client-killed',name=self.name,pid=self.proc.pid,start=self.start)
        def finish(self,expected):
            assert self.proc.wait(15)==expected;self.reader.join(2);self.out.close();self.err.close()
        def stop(self):self.proc.stdin.write('stop\n');self.proc.stdin.flush();self.finish(0)
    def snapshot(name):v=journal.status();save(root/'writer-fixtures'/(name+'.json'),v);return v
    try:
        keyring=subprocess.Popen(['/usr/bin/gnome-keyring-daemon','--foreground','--unlock','--components=secrets'],stdin=subprocess.PIPE,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL);keyring.stdin.write(os.urandom(32).hex().encode());keyring.stdin.close()
        session={'url':hub.url,'ship':'~zod','cookieName':'urbauth-~zod','cookieValue':'stable-theme'};Keyring().store(hub.url,session)
        live=Client('live-watch','watch');wait(lambda:any(e['event']=='ready' for e in live.events));first=snapshot('01-live-watch')['channels'][0];firstpath='/~/channel/'+first['channel']
        writer=Client('publisher','publish');wait(lambda:len(hub.pokes)==1)
        rows=snapshot('02-concurrent-uncertain')['channels'];assert len(rows)==2 and all(r['phase']=='open' for r in rows);pending=rows[1];pendingpath='/~/channel/'+pending['channel']
        writer.kill();writer.finish(-signal.SIGKILL)
        assert live.proc.poll() is None and not hub.channels[firstpath]['deleted'];record('live-watch-preserved-after-publisher-death')
        hub.held_path=pendingpath;cleaner=Client('lost-delete','watch');assert hub.accepted_delete.wait(10)
        closing=snapshot('03-held-delete')['channels'][1];assert closing['phase']=='closing' and closing['outcome']=='uncertain'
        cleaner.kill();cleaner.finish(-signal.SIGKILL);hub.release_delete.set();hub.held_path=None
        recovered=Client('recovered-watch','watch');wait(lambda:any(e['event']=='ready' for e in recovered.events))
        rows=snapshot('04-recovered')['channels'];assert len(rows)==3 and rows[1]['phase']=='closed' and rows[1]['outcome']=='uncertain'
        assert rows[1]['cleanup_request']==closing['cleanup_request'] and rows[1]['cleanup_actor']!=closing['cleanup_actor']
        assert rows[0]['owner']==first['owner'] and rows[0]['phase']=='open' and live.proc.poll() is None
        repeat=Client('blocked-repeat','publish');repeat.finish(1);assert repeat.events==[{'event':'error','code':'transport-state'}] and len(hub.pokes)==1
        reader=Client('readback','read');reader.finish(0);assert reader.events==[{'event':'read'}]
        rows=snapshot('05-state-confirmed')['channels'];assert rows[1]['outcome']=='state-confirmed' and len(hub.pokes)==1
        assert Keyring().lookup(hub.url,'~zod')==session;record('exact-readback-confirmed-without-republication')
        recovered.stop();live.stop();wait(lambda:not hub.active());final=snapshot('06-closed')
        assert len(final['channels'])==3 and all(row['phase']=='closed' for row in final['channels'])
        assert all(row['deleted'] for row in hub.snapshot()) and len(hub.pokes)==1
        report.update(passed=True,channels=hub.snapshot(),actions=hub.requests,pokes=hub.pokes,live_watch_preserved=True,cleanup_request_preserved=True,uncertain_write_not_replayed=True,authenticated_state_confirmation=True,private_credential_preserved=True,client_exit_codes={c.name:c.proc.returncode for c in children})
    finally:
        hub.release_delete.set()
        for child in children:
            if child.proc.poll() is None:child.kill();child.finish(-signal.SIGKILL)
        hub.close()
        if keyring:keyring.terminate();keyring.wait(5)
        save(root/'result.json',report)
    print(json.dumps({'passed':report['passed'],'channels':len(hub.channels),'application_pokes':len(hub.pokes),'live_watch_preserved':True}))


def main():
    p=argparse.ArgumentParser();p.add_argument('--inside',action='store_true');p.add_argument('--source');p.add_argument('--peer');p.add_argument('--out');a=p.parse_args()
    if a.inside:return inside()
    src=Path(a.source).resolve();peer=Path(a.peer).resolve();out=Path(a.out).absolute()
    assert out.parent==Path('/tmp') and out.name.startswith('omabit-theme-transport-runtime-') and not out.exists();out.mkdir(mode=0o700)
    for name in ('source','home','config','data','state','cache','runtime','etc','writer-fixtures'):(out/name).mkdir(mode=0o700)
    files=list((src/'client').glob('*.py'))+[src/'manifest.json',src/'application-connection.json'];hashes={}
    for file in files:
        assert file.is_file() and not file.is_symlink();relative=file.relative_to(src);target=out/'source'/relative;target.parent.mkdir(parents=True,exist_ok=True);shutil.copy2(file,target);hashes[str(file)]=sha(file)
    m=read(out/'source/manifest.json');m['id']='thelifeandtimes.urbit-theme.qualification-transport';m['name']+=' (transport qualification)';save(out/'source/manifest.json',m);save(out/'source/application-profile.json',{'version':1,'profile':'qualification-transport'})
    shutil.copy2(peer,out/'connected_fixture.py');shutil.copy2(__file__,out/'driver.py');(out/'child.py').write_text(CHILD)
    save(out/'source.json',{'original_sources':hashes,'peer':str(peer),'peer_sha256':sha(peer),'fixture_sha256':sha(Path(__file__)),'child_sha256':sha(out/'child.py'),'actual_theme_transport_unchanged':True})
    save(out/'host-namespaces.json',{n:os.readlink('/proc/self/ns/'+n) for n in ('user','net','pid','mnt')});(out/'etc/passwd').write_text('lab:x:'+str(os.getuid())+':'+str(os.getgid())+':Lab:/lab/home:/bin/sh\n');(out/'etc/group').write_text('lab:x:'+str(os.getgid())+':\n')
    cmd=['bwrap','--unshare-all','--die-with-parent','--clearenv','--ro-bind','/usr','/usr','--symlink','usr/bin','/bin','--symlink','usr/lib','/lib','--symlink','usr/lib','/lib64','--proc','/proc','--dev','/dev','--tmpfs','/tmp','--ro-bind',str(out/'etc'),'/etc','--bind',str(out),'/lab','--ro-bind',str(out/'source'),'/lab/source','--chdir','/lab']
    env={'HOME':'/lab/home','PATH':'/usr/bin:/bin','LANG':'C.UTF-8','PYTHONDONTWRITEBYTECODE':'1','XDG_RUNTIME_DIR':'/lab/runtime','OMABIT_THEME_TRANSPORT_LAB':'private-v1'}
    for name in ('CONFIG','DATA','STATE','CACHE'):env['XDG_'+name+'_HOME']='/lab/'+name.lower()
    for key,value in env.items():cmd+=['--setenv',key,value]
    cmd+=['--','/usr/bin/dbus-run-session','--','/usr/bin/python3','-B','/lab/driver.py','--inside'];save(out/'command.json',cmd)
    q=subprocess.run(cmd,capture_output=True,text=True,timeout=130);(out/'driver.log').write_text(q.stdout+q.stderr);print(q.stdout+q.stderr);raise SystemExit(q.returncode)

if __name__=='__main__':main()
