#!/usr/bin/env python3
"""Actual DMs helper death/reconnect in a fresh namespace and private keyring.

Only the HTTP peer is synthetic. No host keyring, desktop, ships or data mount.
"""
from pathlib import Path
import argparse,hashlib,json,os,shutil,signal,socket,subprocess,sys,threading,time

def sha(path):return hashlib.sha256(path.read_bytes()).hexdigest()
def read(path):return json.loads(path.read_text())
def save(path,value):path.write_text(json.dumps(value,indent=2)+'\n')

def inside():
    root=Path('/lab');assert Path.home()==root/'home' and not Path('/home').exists() and not Path('/run/user').exists()
    assert os.environ.get('OMABIT_DMS_TRANSPORT_LAB')=='private-v1'
    assert all(os.readlink('/proc/self/ns/'+n)!=v for n,v in read(root/'host-namespaces.json').items())
    assert [n for _,n in socket.if_nameindex()]==['lo']
    sys.path.insert(0,str(root/'source'));sys.path.insert(0,str(root))
    import connected_fixture as peer
    from session_store import SessionStore
    from transport_journal import TransportJournal
    from outbound_journal import OutboundJournal
    from effects_journal import EffectsJournal
    from instance import INSTANCE
    assert INSTANCE.plugin_id=='urbit.dms.qualification-transport'
    journal=TransportJournal();store=SessionStore();outbound=OutboundJournal();effects=EffectsJournal(outbound.store.root)
    processes=[];report={'passed':False,'private_namespaces':True,'synthetic_eyre':True,'actual_helper_and_keyring':True,'events':[]}
    def record(event,**data):report['events'].append({'event':event,**data});save(root/'result.json',report)
    def wait(fn,seconds=15):
        deadline=time.monotonic()+seconds
        while time.monotonic()<deadline:
            value=fn()
            if value:return value
            time.sleep(.03)
        raise AssertionError('transport runtime observation timed out')
    class Handler(peer.Handler):
        def reply(self,*args,**kwargs):
            if self.command=='PUT' and self.path in self.server.drop_delete and self.server.requests[-1]['action']=='delete':
                self.close_connection=True;self.connection.shutdown(socket.SHUT_RDWR);self.connection.close();return
            return super().reply(*args,**kwargs)
        def do_PUT(self):
            # Assert the actual bridge wrote its intent before each HTTP effect.
            state=journal.status();row=next((r for r in state['channels'] if '/~/channel/'+r['channel']==self.path),None)
            assert row is not None and row['phase'] in ('open','closing')
            super().do_PUT()
        def do_GET(self):
            if not self.path.startswith('/~/channel/'):return super().do_GET()
            s=self.server;owner=self.owner()
            with s.cond:c=s.channels[self.path];assert c['owner']==owner;c['active']+=1
            self.send_response(200);self.send_header('Content-Type','text/event-stream');self.send_header('Connection','close');self.end_headers();self.close_connection=True
            try:
                while not s.stopping:
                    with s.cond:
                        if c['deleted'] or self.path in s.drop_stream:break
                        frames=c['frames'][:];c['frames'].clear()
                    for ident,row in frames:
                        self.wfile.write(('id: '+str(ident)+'\ndata: '+json.dumps(row)+'\n\n').encode());self.wfile.flush()
                        with s.cond:c['delivered']+=1
                    self.wfile.write(b': heartbeat\n\n');self.wfile.flush();time.sleep(.05)
            except (BrokenPipeError,ConnectionResetError):pass
            finally:
                with s.cond:c['active']-=1
    class Hub(peer.Hub):
        def __init__(self):
            self.drop_delete=set();self.drop_stream=set();super().__init__();self.RequestHandlerClass=Handler
    hub=Hub();keyring=None
    class Helper:
        def __init__(self,name):
            self.name=name;self.events=[];self.err=(root/(name+'.stderr.log')).open('wb');self.log=(root/(name+'.events.jsonl')).open('w')
            self.proc=subprocess.Popen([sys.executable,'-B','/lab/source/bridge.py'],stdin=subprocess.PIPE,stdout=subprocess.PIPE,stderr=self.err,text=True)
            self.start=Path('/proc/'+str(self.proc.pid)+'/stat').read_text().split(') ',1)[1].split()[19]
            processes.append(self)
            def collect():
                for line in self.proc.stdout:
                    self.log.write(line);self.log.flush();self.events.append(json.loads(line))
            self.reader=threading.Thread(target=collect,daemon=True);self.reader.start()
        def command(self,action,**fields):
            self.proc.stdin.write(json.dumps({'action':action,**fields})+'\n');self.proc.stdin.flush()
        def kill(self):
            assert self.proc.poll() is None
            assert Path('/proc/'+str(self.proc.pid)+'/stat').read_text().split(') ',1)[1].split()[19]==self.start
            self.proc.send_signal(signal.SIGKILL);assert self.proc.wait(5)==-signal.SIGKILL;self.reader.join(2)
            record('exact-helper-killed',name=self.name,pid=self.proc.pid,start=self.start)
        def ended(self):self.reader.join(2);self.err.close();self.log.close()
    def snapshot(name):
        value=journal.status();save(root/'writer-fixtures'/(name+'.json'),value);return value
    try:
        keyring=subprocess.Popen(['/usr/bin/gnome-keyring-daemon','--foreground','--unlock','--components=secrets'],stdin=subprocess.PIPE,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
        keyring.stdin.write(os.urandom(32).hex().encode());keyring.stdin.close()
        session={'url':hub.url,'ship':'~zod','cookieName':'urbauth-~zod','cookieValue':'stable-dms'}
        store.save(session);account=store.account();record('private-session-seeded')
        first=Helper('first');wait(lambda:len(hub.active())==1 and hub.active()[0]['subscriptions']==4 and hub.active()[0]['delivered']>=4)
        old=snapshot('01-open')['channels'][-1];oldpath='/~/channel/'+old['channel']
        first.command('send',ship='~nec',text='private uncertain message',request=101)
        first.command('accept',ship='~nec',request=102)
        first.command('open',ship='~nec',request=103)
        wait(lambda:len(hub.pokes)==3);wait(lambda:len(outbound.unresolved())==1 and len(effects.unresolved())==2)
        saved=(outbound.path.read_bytes(),effects.path.read_bytes());record('uncertain-application-writes',count=3)
        # A concurrent real helper must refuse without touching its owner's channel.
        competing=Helper('competing');wait(lambda:any(e['event']=='error' for e in competing.events))
        competing.proc.stdin.close();assert competing.proc.wait(10)==1;competing.ended()
        assert journal.status()['channels']==[old] and len(hub.channels)==1
        record('concurrent-helper-refused');first.kill();first.ended()
        wait(lambda:not hub.active());assert not hub.channels[oldpath]['deleted']
        hub.drop_delete.add(oldpath)
        second=Helper('lost-delete');wait(lambda:any(e['event']=='error' for e in second.events))
        wait(lambda:hub.channels[oldpath]['deleted'])
        closing=snapshot('02-closing-lost-reply')['channels'][-1]
        assert closing['phase']=='closing' and closing['channel']==old['channel'] and len(hub.channels)==1
        assert not any(e['event']=='login' for e in second.events)
        second.kill();second.ended();hub.drop_delete.clear()
        third=Helper('recovered');wait(lambda:len(hub.active())==1 and len(hub.channels)==2 and hub.active()[0]['delivered']>=4)
        recovered=snapshot('03-recovered');closed,current=recovered['channels']
        assert closed['phase']=='closed' and closed['cleanup_request']==closing['cleanup_request']
        assert closed['cleanup_actor']!=closing['cleanup_actor'] and closed['owner']==old['owner']
        assert saved==(outbound.path.read_bytes(),effects.path.read_bytes()) and len(hub.pokes)==3
        assert store.account()==account and store.lookup(hub.url,'~zod')==session
        record('killed-cleaner-recovered',same_cleanup_request=True,application_replay=False)
        # A stream disconnect must wait on deletion, without adding subscriptions.
        currentpath='/~/channel/'+current['channel'];hub.drop_delete.add(currentpath);hub.drop_stream.add(currentpath)
        wait(lambda:journal.status()['channels'][-1]['phase']=='closing')
        wait(lambda:any(r['channel']==currentpath and r['action']=='delete' for r in hub.requests))
        reconnect=snapshot('04-reconnect-closing')['channels'][-1]
        assert len(hub.channels)==2 and hub.channels[currentpath]['subscriptions']==4
        time.sleep(.2);assert len(hub.channels)==2
        hub.drop_delete.clear();wait(lambda:len(hub.active())==1 and len(hub.channels)==3 and hub.active()[0]['delivered']>=4,20)
        value=snapshot('05-reconnected');assert value['channels'][1]['phase']=='closed' and value['channels'][1]['cleanup_request']==reconnect['cleanup_request']
        assert len(hub.pokes)==3 and saved==(outbound.path.read_bytes(),effects.path.read_bytes())
        record('reconnect-resubscribed-after-exact-cleanup',application_replay=False)
        third.command('shutdown-transport');wait(lambda:any(e['event']=='transport-state' for e in third.events))
        assert third.proc.wait(10)==0;third.ended()
        receipt=next(e for e in third.events if e['event']=='transport-state')
        assert receipt['closed'] and not receipt['collectionReady'] and len(receipt['unresolvedSends'])==1 and len(receipt['unresolvedEffects'])==2
        final=snapshot('06-closed');assert all(r['phase']=='closed' for r in final['channels'])
        assert all(c['deleted'] and not c['active_streams'] for c in hub.snapshot())
        assert saved==(outbound.path.read_bytes(),effects.path.read_bytes()) and store.lookup(hub.url,'~zod')==session
        report.update(passed=True,channels=hub.snapshot(),requests=hub.requests,pokes=hub.pokes,uncertain_app_bytes_preserved=True,private_credential_preserved=True,final_receipt=receipt,helper_exit_codes={p.name:p.proc.returncode for p in processes})
    finally:
        for p in processes:
            if p.proc.poll() is None:p.kill();p.ended()
        hub.close()
        if keyring is not None:keyring.terminate();keyring.wait(5)
        save(root/'result.json',report)
    print(json.dumps({'passed':report['passed'],'channels':len(hub.channels),'app_pokes':len(hub.pokes),'helper_exit_codes':report['helper_exit_codes']}))


def main():
    parser=argparse.ArgumentParser();parser.add_argument('--source');parser.add_argument('--peer');parser.add_argument('--out');parser.add_argument('--inside',action='store_true');args=parser.parse_args()
    if args.inside:return inside()
    src=Path(args.source).resolve();peer=Path(args.peer).resolve();out=Path(args.out).absolute()
    assert out.parent==Path('/tmp') and out.name.startswith('omabit-dms-transport-runtime-') and not out.exists()
    out.mkdir(mode=0o700)
    for n in ('source','home','config','data','state','cache','runtime','etc','writer-fixtures'):(out/n).mkdir(mode=0o700)
    files=[p for p in src.glob('*.py') if not p.name.startswith('test_')]+[src/'manifest.json',src/'application-connection.json'];hashes={}
    for p in files:
        assert p.is_file() and not p.is_symlink();shutil.copy2(p,out/'source'/p.name);hashes[str(p)]=sha(p)
    m=read(out/'source/manifest.json');m['id']='urbit.dms.qualification-transport';m['name']+=' (transport qualification)';save(out/'source/manifest.json',m);save(out/'source/application-profile.json',{'version':1,'profile':'qualification-transport'})
    shutil.copy2(peer,out/'connected_fixture.py');shutil.copy2(__file__,out/'driver.py')
    save(out/'source.json',{'original_sources':hashes,'peer':str(peer),'peer_sha256':sha(peer),'fixture_sha256':sha(Path(__file__)),'adaptations':['separate profile/label','synthetic HTTP peer'],'actual_dms_helper_unchanged':True})
    save(out/'host-namespaces.json',{n:os.readlink('/proc/self/ns/'+n) for n in ('user','net','pid','mnt')})
    (out/'etc/passwd').write_text('lab:x:'+str(os.getuid())+':'+str(os.getgid())+':Lab:/lab/home:/bin/sh\n');(out/'etc/group').write_text('lab:x:'+str(os.getgid())+':\n')
    cmd=['bwrap','--unshare-all','--die-with-parent','--clearenv','--ro-bind','/usr','/usr','--symlink','usr/bin','/bin','--symlink','usr/lib','/lib','--symlink','usr/lib','/lib64','--proc','/proc','--dev','/dev','--tmpfs','/tmp','--ro-bind',str(out/'etc'),'/etc','--bind',str(out),'/lab','--ro-bind',str(out/'source'),'/lab/source','--chdir','/lab']
    env={'HOME':'/lab/home','PATH':'/usr/bin:/bin','LANG':'C.UTF-8','PYTHONDONTWRITEBYTECODE':'1','XDG_RUNTIME_DIR':'/lab/runtime','OMABIT_DMS_TRANSPORT_LAB':'private-v1','OMABIT_LIFECYCLE_DRAIN':'1','OMABIT_LIFECYCLE_EFFECTS':'1'}
    for n in ('CONFIG','DATA','STATE','CACHE'):env['XDG_'+n+'_HOME']='/lab/'+n.lower()
    for k,v in env.items():cmd+=['--setenv',k,v]
    cmd+=['--','/usr/bin/dbus-run-session','--','/usr/bin/python3','-B','/lab/driver.py','--inside'];save(out/'command.json',cmd)
    q=subprocess.run(cmd,capture_output=True,text=True,timeout=130);(out/'driver.log').write_text(q.stdout+q.stderr);print(q.stdout+q.stderr);raise SystemExit(q.returncode)

if __name__=='__main__':main()
