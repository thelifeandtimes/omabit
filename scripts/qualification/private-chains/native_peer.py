"""Fresh native peers control the protocol before the slower private Docker gate."""
from http.cookiejar import CookieJar
import json,os,re,signal,socket,subprocess,threading,time,uuid
from pathlib import Path
import urllib.error,urllib.request,urllib.parse
from thread_probe import NoRedirect,limits
from network_case import thread_poke,thread_receipt
from state_case import Recovery,parse_sse,matching_state

def launch_command(ship,item,resume,gateway):
    args=['/vere','-t','-L','--no-conn','--no-dock','--loom','31','--snap-time','60','-b','127.0.0.1','--http-port',str(item['port']),'-p',str(31337+item['number'])]
    if not resume:args+=['-H','galaxies.omabit.invalid','-w',ship,'-k','/lab/secrets/'+str(item['number'])+'.key','-e',gateway,'-B','/boot.pill','-c']
    return args+[str(item['pier'])]

def exercise(guard):
    if os.environ.get('OMABIT_PRIVATE_CHAIN')!='private-chain-peers-v1' or os.environ.get('HOME')!='/lab/home':raise RuntimeError('private keyed peer admission required')
    original=json.loads(Path('/lab/host-namespaces.json').read_text())
    if any(os.readlink('/proc/self/ns/'+n)==original[n] for n in ('user','net','pid','mnt')):raise RuntimeError('host namespace refused')
    if any(Path(p).exists() for p in ('/home','/run/user','/var/run/docker.sock')):raise RuntimeError('host mounts refused')
    if sorted(name for _,name in socket.if_nameindex())!=['lo']:raise RuntimeError('external network refused')
    if Path('/lab/marker').read_text()!='private-chain-peers-v1\n' or not json.loads(Path('/lab/scope-admission.json').read_text())['admitted']:raise RuntimeError('private scope admission missing')
    gateway=os.environ.get('OMABIT_PRIVATE_GATEWAY','')
    if not re.fullmatch(r'http://127\.0\.0\.1:[1-9][0-9]{0,4}',gateway):raise RuntimeError('private gateway binding invalid')
    lab=Path('/lab');ships={};readers=[];overflow=threading.Event()
    result={'ok':False,'unit':guard['unit'],'transport':'local-khan-thread','ships':{},'attempts':[],'clean_stops':[],'cleanup_errors':[],'stop_records':[]}
    recovery=None;plugins=None
    def save():(lab/'peer-result.json').write_text(json.dumps(result,indent=2)+'\n')
    def emit(stage,**kw):print('PEER_PROBE '+json.dumps(dict(stage=stage,**kw)),flush=True)
    def http(ship,path,data=None,content=None,accept=None,admin=False,auth=None,timeout=60,method=None):
        item=ships[ship];headers={}
        if content:headers['Content-Type']=content
        if accept:headers['Accept']=accept
        port=item['admin'] if admin else item['port'];opener=ships[auth or ship]['opener']
        req=urllib.request.Request('http://127.0.0.1:'+str(port)+path,data=data,headers=headers,method=method)
        try:response=opener.open(req,timeout=timeout)
        except urllib.error.HTTPError as e:response=e
        with response:
            body=response.read(65537)
            if len(body)>65536:raise RuntimeError('HTTP output bound')
            return response.status,body
    def events(channel,cursor,expected,initial):
        headers={'Accept':'text/event-stream'}
        if cursor is not None:headers['Last-Event-ID']=str(cursor)
        request=urllib.request.Request('http://127.0.0.1:'+str(ships['nec']['port'])+channel,headers=headers)
        rows=[];ack=not initial;fact=False
        with ships['nec']['opener'].open(request,timeout=45) as response:
            if response.status!=200:raise RuntimeError('SSE reconnect refused')
            for identity,event in parse_sse(iter(lambda:response.readline(8193),b'')):
                rows.append((identity,event))
                fact=matching_state(event,expected) or fact
                ack=ack or (event.get('response')=='subscribe' and event.get('id')==1 and event.get('ok')=='ok')
                if fact and ack:return rows
        raise RuntimeError('SSE ended before matching application state')
    def start(ship,resume=False):
        item=ships[ship];item['resumed']=resume;args=launch_command(ship,item,resume,gateway)
        child=subprocess.Popen(args,stdin=subprocess.DEVNULL,stdout=subprocess.PIPE,stderr=subprocess.STDOUT,start_new_session=True,preexec_fn=limits);item['child']=child
        def capture():
            total=0
            with (lab/(ship+('-resume' if resume else '')+'.log')).open('xb') as out:
                while chunk:=child.stdout.read(65536):
                    out.write(chunk[:max(0,8*1024**2-total)]);out.flush();total+=len(chunk)
                    if total>8*1024**2:overflow.set()
        reader=threading.Thread(target=capture,daemon=True);reader.start();readers.append(reader)
        deadline=time.monotonic()+(120 if resume else 240)
        while child.poll() is None and time.monotonic()<deadline and not overflow.is_set():
            try:status,body=http(ship,'/~/host',timeout=1)
            except (OSError,urllib.error.URLError):status,body=0,b''
            if status==200 and body.strip()==('~'+ship).encode():break
            time.sleep(1)
        else:raise RuntimeError('fresh '+ship+' not ready; resume='+str(resume)+'; child_exit='+str(child.poll()))
        rows=[line.split() for line in (item['pier']/'.http.ports').read_text().splitlines()]
        ports=[int(row[0]) for row in rows if row[1:]==['insecure','loopback']]
        if len(ports)!=1:raise RuntimeError('admin inventory invalid')
        item['admin']=ports[0]
        status,body=http(ship,'/',json.dumps({'source':{'dojo':'our'},'sink':{'stdout':None}}).encode(),content='application/json',admin=True)
        if status!=200 or json.loads(body).strip()!='~'+ship:raise RuntimeError('admin identity mismatch')
        result['ships'].setdefault(ship,{}).update({'port':item['port'],'admin':item['admin'],'pid':child.pid});save();emit('ready',ship=ship,resume=resume)
    def login(ship):
        status,body=http(ship,'/',json.dumps({'source':{'dojo':'+code'},'sink':{'stdout':None}}).encode(),content='application/json',admin=True)
        if status!=200:raise RuntimeError('code unavailable')
        code=json.loads(body).strip().removeprefix('~')
        if not re.fullmatch('[a-z]{6}(?:-[a-z]{6}){3}',code):raise RuntimeError('invalid code')
        status,_=http(ship,'/~/login',urllib.parse.urlencode({'password':code}).encode(),content='application/x-www-form-urlencoded');del code,body
        if not 200<=status<300:raise RuntimeError('login refused')
        status,_=http(ship,'/~/scry/hood/kiln/pikes.json')
        if status!=200:raise RuntimeError('authenticated inventory refused')
    def configure_watcher(ship):
        peer='nec' if ship=='zod' else 'zod';commands=[];result['ships'][ship]['watcher_commands']=commands;save()
        setup=[("+azimuth/watch '"+gateway+"/rpc' %local",'azimuth'),('&eth-watcher-poke [%clear /azimuth]','eth-watcher'),('[%refresh-rate ~s1]','azimuth'),('%resub','azimuth'),('&azimuth-poke [%kick ~]','azimuth')]
        for source,app in setup:
            entry={'source':source,'app':app,'dispatched':True};commands.append(entry);save()
            status,body=http(ship,'/',json.dumps({'source':{'dojo':source},'sink':{'app':app}}).encode(),content='application/json',admin=True)
            if status!=200 or json.loads(body).strip()!='>=':raise RuntimeError('watcher configuration acknowledgment missing')
            entry['acknowledged']=True;save()
        deadline=time.monotonic()+120
        expression='.^(@ud %j /(scot %p our)/life/(scot %da now)/(scot %p ~'+peer+'))'
        while time.monotonic()<deadline:
            status,body=http(ship,'/',json.dumps({'source':{'dojo':expression},'sink':{'stdout':None}}).encode(),content='application/json',admin=True)
            if status==200 and json.loads(body).strip()=='1':
                result['ships'][ship]['peer_jael_life']=1;save();emit('private-key-peer-observed',ship=ship,peer=peer);return
            time.sleep(1)
        raise RuntimeError('private peer Jael observation deadline')
    def poke(sender,receiver,phase,reject=False):
        nonce='fail' if reject else 'omabit-'+phase+'-'+uuid.uuid4().hex[:24];expected='omabit-receipt-'+uuid.uuid4().hex
        attempt={'sender':sender,'receiver':receiver,'phase':phase,'nonce':nonce,'expected_receipt':expected,'state':'pending'};result['attempts'].append(attempt);save()
        status,body=http(sender,'/spider/base/ted-eval/khan-eval/noun',thread_poke(receiver,nonce,expected),content='application/x-urb-jam',accept='application/json' if reject else 'application/x-urb-jam')
        attempt.update(status=status,bytes=len(body));save();attempt['receipt']=thread_receipt(status,body,expected,reject);attempt['state']='rejected' if reject else 'acknowledged';save();emit('message',sender=sender,receiver=receiver,phase=phase)
    def stop(ship):
        item=ships[ship];child=item['child'];mode='sigterm' if item['resumed'] else 'lens-exit'
        row={'ship':ship,'pid':child.pid,'mode':mode,'passed':False};result['stop_records'].append(row);save();started=time.monotonic()
        if child.poll() is None:
            signals=dict(line.split(':',1) for line in Path('/proc/'+str(child.pid)+'/status').read_text().splitlines() if line.startswith('SigIgn:'))
            row['sigterm_ignored']=bool(int(signals['SigIgn'].strip(),16)&(1<<14));save()
            if row['sigterm_ignored'] != (not item['resumed']):raise RuntimeError('official-runtime signal control differs from qualified first-boot/reopen behavior')
            if mode=='lens-exit':
                row['lens_exit_dispatched']=True;save()
                try:
                    status,body=http(ship,'/',json.dumps({'source':{'dojo':'+hood/exit'},'sink':{'app':'hood'}}).encode(),content='application/json',admin=True,timeout=15)
                    row['lens_status']=status;row['lens_acknowledged']=status==200 and json.loads(body).strip()=='>=';save()
                except (OSError,urllib.error.URLError):row['lens_reply_unavailable']=True;save()
            else:os.killpg(child.pid,signal.SIGTERM)
        try:child.wait(timeout=60)
        except subprocess.TimeoutExpired:
            row['forced']=True;save();os.killpg(child.pid,signal.SIGKILL);child.wait(timeout=5)
        row['exit']=child.returncode;row['milliseconds']=int((time.monotonic()-started)*1000);save()
        if child.returncode!=0:raise RuntimeError('fresh '+ship+' did not stop cleanly')
        row['passed']=True;result['clean_stops'].append(ship);save()

    try:
        for ship,port in [('zod',18081),('nec',18082)]:
            jar=CookieJar();pier=lab/'piers'/('private-'+ship)/ship;pier.parent.mkdir(parents=True,mode=0o700);ships[ship]={'number':0 if ship=='zod' else 1,'pier':pier,'port':port,'opener':urllib.request.build_opener(urllib.request.ProxyHandler({}),NoRedirect(),urllib.request.HTTPCookieProcessor(jar))}
            start(ship);login(ship);configure_watcher(ship)
        for ship,wrong in [('zod','nec'),('nec','zod')]:
            status,_=http(ship,'/~/scry/hood/kiln/pikes.json',auth=wrong)
            if status not in (401,403):raise RuntimeError('cross-ship cookie accepted')
        result['cross_ship_sessions_refused']=True
        for sender,receiver,phase,reject in [('zod','zod','local-check',False),('nec','nec','local-check',False),('zod','nec','first',False),('nec','zod','first',False),('zod','nec','nack',True)]:poke(sender,receiver,phase,reject)
        if guard.get('application_state'):
            recovery=Recovery(http,events,save,emit);result['application_state']=recovery.report
            recovery.install(ships['nec']['pier'],Path('/artifacts/omabit-state.hoon'));recovery.before_restart()
        if guard.get('plugin_transport'):
            from plugin_case import PluginRecovery
            plugins=PluginRecovery(http,ships,save,emit);result['plugins']=plugins.report;plugins.before_restart()
        pier=ships['nec']['pier'];before=(pier.stat().st_dev,pier.stat().st_ino);old_pid=ships['nec']['child'].pid
        stop('nec');start('nec',True)
        if before!=(pier.stat().st_dev,pier.stat().st_ino) or old_pid==ships['nec']['child'].pid:raise RuntimeError('restart pier/process invariant failed')
        poke('zod','nec','after-restart');poke('nec','zod','after-restart')
        if recovery:recovery.after_restart()
        if plugins:plugins.after_restart()
        result.update(transport_passed=True,restart_same_pier=True,restart_kept_sessions=True)
    except Exception as e:result['error']=type(e).__name__+': '+str(e);emit('failure',error=result['error'])
    finally:
        if plugins:
            try:plugins.close()
            except Exception as e:result['cleanup_errors'].append('plugins: '+type(e).__name__+': '+str(e))
        if recovery and recovery.channel:
            try:recovery.close()
            except Exception as e:result['cleanup_errors'].append('subscription: '+type(e).__name__)
        for ship in reversed(list(ships)):
            if 'child' not in ships[ship]:continue
            try:
                if ships[ship]['child'].poll() is None:stop(ship)
                elif ships[ship]['child'].returncode!=0:raise RuntimeError('child exited uncleanly')
            except Exception as e:result['cleanup_errors'].append(str(e))
        for reader in readers:reader.join(timeout=5)
        result['log_truncated']=overflow.is_set();result['ok']=result.get('transport_passed',False) and not result['cleanup_errors'] and not overflow.is_set();save();emit('complete',ok=result['ok'],clean_stops=result['clean_stops'])
    return 0 if result['ok'] else 1
