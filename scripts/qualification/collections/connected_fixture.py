"""Synthetic Eyre on namespace loopback; real private Secret Service, no desktop."""
from pathlib import Path
import http.server,json,threading,subprocess,sys,secrets,time,os
from urllib.parse import parse_qs
class Hub(http.server.ThreadingHTTPServer):
    daemon_threads=True
    def __init__(self):
        super().__init__(('127.0.0.1',0),Handler);self.url='http://127.0.0.1:'+str(self.server_port);self.cond=threading.Condition();self.channels={};self.sequence=0;self.requests=[];self.pokes=[];self.confirmed=False;self.stopping=False;self.thread=threading.Thread(target=self.serve_forever,daemon=True);self.thread.start()
    def snapshot(self):
        with self.cond:return [{'path':path,'owner':c['owner'],'subscriptions':c['subscriptions'],'active_streams':c['active'],'deleted':c['deleted'],'delivered':c['delivered']} for path,c in self.channels.items()]
    def active(self):return [r for r in self.snapshot() if r['active_streams']]
    def close(self):
        with self.cond:self.stopping=True;self.cond.notify_all()
        self.shutdown();self.server_close();self.thread.join(2)
class Handler(http.server.BaseHTTPRequestHandler):
    protocol_version='HTTP/1.1'
    def log_message(self,*_):pass
    def owner(self):
        owner=self.headers.get('Cookie','').removeprefix('urbauth-~zod=')
        if owner not in [p+'-'+a for p in ('stable','dev','edge') for a in ('theme','dms')]:raise ValueError('foreign fixture session')
        return owner
    def reply(self,value=None,status=200,cookie=None):
        body=b'' if value is None else json.dumps(value).encode();self.send_response(status);self.send_header('Content-Length',str(len(body)));self.send_header('Content-Type','application/json')
        if cookie:self.send_header('Set-Cookie','urbauth-~zod='+cookie+'; Path=/; HttpOnly')
        self.end_headers();self.wfile.write(body)
    def do_POST(self):
        body=parse_qs(self.rfile.read(int(self.headers.get('Content-Length','0'))).decode());owner=body.get('password',[''])[0]
        if self.path!='/~/login' or owner not in [p+'-theme' for p in ('stable','dev','edge')]:return self.reply(status=403)
        self.reply(cookie=owner)
    def do_GET(self):
        s=self.server;owner=self.owner()
        if self.path.startswith('/~/scry/'):
            if '/settings/' in self.path:return self.reply({'desk':{}})
            if os.environ.get('OMABIT_LIFECYCLE_RECONCILE')=='1':
                if self.path=='/~/scry/groups-ui/v4/heads.json':return self.reply({'dms':[]})
                if s.confirmed and owner=='stable-dms':
                    if self.path=='/~/scry/chat/dm.json':return self.reply(['~nec'])
                    if self.path=='/~/scry/activity/v4/activity.json':return self.reply({'ship/~nec':{'recency':0,'count':4,'notify-count':1,'notify':True,'unread':None}})

            if '/chat/' in self.path and '/writs/' in self.path:return self.reply({'writs':{},'older':None})
            if '/chat/' in self.path:return self.reply([])
            return self.reply({})
        if not self.path.startswith('/~/channel/'):return self.reply(status=404)
        with s.cond:
            c=s.channels[self.path];assert c['owner']==owner;c['active']+=1
        self.send_response(200);self.send_header('Content-Type','text/event-stream');self.send_header('Connection','close');self.end_headers();self.close_connection=True
        try:
            while not s.stopping:
                with s.cond:
                    if c['deleted']:break
                    frames=c['frames'][:];c['frames'].clear()
                if frames:
                    for ident,row in frames:
                        self.wfile.write(('id: '+str(ident)+'\ndata: '+json.dumps(row)+'\n\n').encode());self.wfile.flush()
                        with s.cond:c['delivered']+=1
                else:self.wfile.write(b': heartbeat\n\n');self.wfile.flush()
                time.sleep(.05)
        except (BrokenPipeError,ConnectionResetError):pass
        finally:
            with s.cond:c['active']-=1
    def do_PUT(self):
        s=self.server;owner=self.owner();raw=self.rfile.read(int(self.headers.get('Content-Length','0')));assert len(raw)<65536;messages=json.loads(raw)
        with s.cond:
            c=s.channels.setdefault(self.path,{'owner':owner,'frames':[],'subscriptions':0,'active':0,'deleted':False,'delivered':0});assert c['owner']==owner
            for m in messages:
                action=m['action'];assert action in ('subscribe','ack','delete','poke'), 'Unexpected action in synthetic uncertainty fixture'
                s.requests.append({'owner':owner,'action':action,'channel':self.path});assert len(s.requests)<2000
                if action=='poke':
                    assert os.environ.get('OMABIT_LIFECYCLE_DRAIN')=='1', 'Application mutations forbidden outside explicit uncertainty fixture'
                    allowed={('chat','chat-dm-action-2')}
                    if os.environ.get('OMABIT_LIFECYCLE_EFFECTS')=='1':allowed|={('chat','chat-dm-rsvp'),('activity','activity-action')}
                    assert owner=='stable-dms' and (m['app'],m['mark']) in allowed
                    assert not any(x['mark']==m['mark'] for x in s.pokes), 'Uncertain synthetic write replayed'
                    s.pokes.append({'owner':owner,'channel':self.path,'id':m['id'],'app':m['app'],'mark':m['mark'],'accepted_without_ack':True})
                elif action=='subscribe':
                    s.sequence+=1;c['subscriptions']+=1;c['frames'].append((s.sequence,{'id':m['id'],'response':'subscribe','ok':'ok'}))
                elif action=='delete':c['deleted']=True
            s.cond.notify_all()
        self.reply()
def seed(root,hub):
    for profile in ('stable','dev','edge'):
        theme=root/'artifacts'/profile/'theme';dms=root/'artifacts'/profile/'dms'
        r=subprocess.run([sys.executable,'-B',str(theme/'client/main.py'),'login'],input=json.dumps({'url':hub.url,'code':profile+'-theme'}).encode(),capture_output=True,timeout=15)
        assert r.returncode==0 and json.loads(r.stdout)['ok'],'Theme fixture login failed'
        code='import sys,json;sys.path.insert(0,sys.argv[1]);from session_store import SessionStore;SessionStore().save(json.load(sys.stdin))'
        session={'url':hub.url,'ship':'~zod','cookieName':'urbauth-~zod','cookieValue':profile+'-dms'}
        r=subprocess.run([sys.executable,'-B','-c',code,str(dms)],input=json.dumps(session).encode(),capture_output=True,timeout=15);assert r.returncode==0,'DMs private fixture session seed failed'
def begin(root, hub_factory=Hub):
    assert os.environ.get('OMABIT_REAL_PLUGIN_LIFECYCLE')=='private-v1' and os.environ.get('HOME')=='/lab/home' and not Path('/home').exists() and os.environ.get('DBUS_SESSION_BUS_ADDRESS')
    keyring=subprocess.Popen(['/usr/bin/gnome-keyring-daemon','--foreground','--unlock','--components=secrets'],stdin=subprocess.PIPE,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL);keyring.stdin.write(secrets.token_hex(32).encode());keyring.stdin.close()
    # Do not let the first secret-tool call race manual unlock and auto-activate
    # a second locked daemon. Query the session bus itself without activating secrets.
    deadline=time.monotonic()+5
    while True:
     q=subprocess.run(['/usr/bin/busctl','--user','call','org.freedesktop.DBus','/org/freedesktop/DBus','org.freedesktop.DBus','NameHasOwner','s','org.freedesktop.secrets'],capture_output=True,text=True,timeout=3)
     if q.returncode==0 and q.stdout.strip()=='b true':break
     assert keyring.poll() is None and time.monotonic()<deadline,'Private keyring did not acquire its bus name'
     time.sleep(.05)
    q=subprocess.run(['/usr/bin/busctl','--user','call','org.freedesktop.DBus','/org/freedesktop/DBus','org.freedesktop.DBus','GetConnectionUnixProcessID','s','org.freedesktop.secrets'],capture_output=True,text=True,timeout=3)
    assert q.returncode==0 and q.stdout.strip()=='u '+str(keyring.pid),'Private keyring ownership differs from the launched daemon'
    (root/'keyring-startup.json').write_text(json.dumps({'manual_daemon_owned_secrets':True,'pid':keyring.pid})+'\n')
    r=subprocess.run(['/usr/bin/secret-tool','store','--label=Private lifecycle fixture','application','omabit-connected-lifecycle'],input=b'synthetic-value',capture_output=True,timeout=10);assert r.returncode==0,'Private keyring unavailable'
    hub=hub_factory();seed(root,hub);return keyring,hub
