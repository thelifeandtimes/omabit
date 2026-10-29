#!/usr/bin/env python3
"""Diagnose the pinned image's HTTP thread path in a fresh private native fake."""
import argparse
import hashlib
from http.cookiejar import CookieJar
import importlib.util
import json
import os
from pathlib import Path
import re
import resource
import shutil
import signal
import socket
import subprocess
import threading
import time
import urllib.error
import urllib.parse
import urllib.request
import uuid

PILL_SHA = 'f8b08c53dd6d0aa9c4f7654f0a7bcd11e94089222888a3c99c0a10eedcbb377b'
VERE_SHA = 'b5c54439ae5b9f0ff94d75319e67e04f04369776fa5e2180a5ab1a5ebd4c2ff0'

def sha(path):
    with path.open('rb') as f:return hashlib.file_digest(f,'sha256').hexdigest()

class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self,*args,**kwargs):raise RuntimeError('redirect refused')

def limits():
    os.nice(15);resource.setrlimit(resource.RLIMIT_CORE,(0,0));resource.setrlimit(resource.RLIMIT_CPU,(240,240))

def inside():
    guard=json.loads(Path('/run/omabit-thread-probe.json').read_text())
    for kind in ('net','pid'):
        if os.readlink('/proc/self/ns/'+kind)==guard['host_'+kind]:raise RuntimeError('host namespace refused')
    if not re.fullmatch('omabit-thread-probe-[0-9a-f]{32}',guard['unit']):raise RuntimeError('invalid private guard')
    if any(Path(p).exists() for p in ('/home','/run/user','/var/run/docker.sock')):raise RuntimeError('host mounts refused')
    if sorted(name for _,name in socket.if_nameindex())!=['lo']:raise RuntimeError('non-loopback device')
    lab=Path('/lab');deadline=time.monotonic()+20
    while not (lab/'admitted').exists() and time.monotonic()<deadline:time.sleep(.1)
    if (lab/'admitted').read_text()!=guard['unit']:raise RuntimeError('scope admission absent')
    os.umask(0o077)
    if guard.get('peer'):
        from peer_probe import exercise
        return exercise(guard)
    spec=importlib.util.spec_from_file_location('codec','/artifacts/network_case.py');codec=importlib.util.module_from_spec(spec);spec.loader.exec_module(codec)
    result={'ok':False,'unit':guard['unit'],'self_messages':[],'clean_shutdown':False}
    def save(): (lab/'result.json').write_text(json.dumps(result,indent=2)+'\n')
    cookie=CookieJar();opener=urllib.request.build_opener(urllib.request.ProxyHandler({}),NoRedirect(),urllib.request.HTTPCookieProcessor(cookie))
    def http(port,path,data=None,content=None,accept=None,timeout=30,method=None):
        headers={}
        if content:headers['Content-Type']=content
        if accept:headers['Accept']=accept
        event={'port':port,'path':path,'method':method or ('POST' if data is not None else 'GET'),'phase':'waiting-headers'}
        result['last_http']=event;save()
        request=urllib.request.Request('http://127.0.0.1:'+str(port)+path,data=data,headers=headers,method=method)
        try:response=opener.open(request,timeout=timeout)
        except urllib.error.HTTPError as error:response=error
        with response:
            event.update(status=response.status,phase='reading-body',headers={name:response.headers.get(name) for name in ('Content-Type','Content-Length','Transfer-Encoding')});save()
            body=response.read(65537)
            event.update(phase='complete',bytes=len(body));save()
            if len(body)>65536:raise RuntimeError('response size bound')
            return response.status,body
    def lens(expression):
        status,body=http(12321,'/',json.dumps({'source':{'dojo':expression},'sink':{'stdout':None}}).encode(),content='application/json')
        if status!=200:raise RuntimeError('Lens unavailable')
        return json.loads(body)
    args=['/artifacts/urbit','-t','-L','--no-conn','--no-dock','--loom','31','--snap-time','60','-b','127.0.0.1','--http-port','18081','-F','zod','-B','/artifacts/boot.pill','-c','/lab/zod']
    result['version']=subprocess.check_output(['/artifacts/urbit','--version'],stderr=subprocess.STDOUT,timeout=10).decode().strip()
    child=subprocess.Popen(args,stdin=subprocess.DEVNULL,stdout=subprocess.PIPE,stderr=subprocess.STDOUT,start_new_session=True,preexec_fn=limits)
    overflow=threading.Event()
    def capture():
        total=0
        with (lab/'vere.log').open('wb') as out:
            while chunk:=child.stdout.read(65536):
                out.write(chunk[:max(0,8*1024**2-total)]);out.flush();total+=len(chunk)
                if total>8*1024**2:overflow.set()
    reader=threading.Thread(target=capture,daemon=True);reader.start()
    try:
        deadline=time.monotonic()+240
        while child.poll() is None and time.monotonic()<deadline and not overflow.is_set():
            try:status,body=http(18081,'/~/host',timeout=1)
            except (OSError,urllib.error.URLError):status,body=0,b''
            if status==200 and body.strip()==b'~zod':break
            time.sleep(1)
        else:raise RuntimeError('fresh fake boot failed')
        if lens('our').strip()!='~zod':raise RuntimeError('admin identity mismatch')
        print('THREAD_PROBE ready',flush=True)
        result['stage']='code';save()
        code=lens('+code').strip().removeprefix('~')
        if not re.fullmatch('[a-z]{6}(?:-[a-z]{6}){3}',code):raise RuntimeError('invalid fake code')
        result['stage']='login';save()
        status,_=http(18081,'/~/login',urllib.parse.urlencode({'password':code}).encode(),content='application/x-www-form-urlencoded');del code
        if not 200<=status<300:raise RuntimeError('login refused')
        result['stage']='inventory';save()
        status,_=http(18081,'/~/scry/hood/kiln/pikes.json');assert status==200
        nonce='omabit-native-'+uuid.uuid4().hex[:24]
        if guard.get('khan_receipt'):
            result['transport']='local-khan-thread';result['thread_receipts']=[]
            for message,reject in [(nonce,False),('fail',True)]:
                reply='omabit-accepted-'+uuid.uuid4().hex
                code="=/  m  (strand ,vase)  ;<  ~  bind:m  (poke [~zod %hood] %helm-hi !>('"+message+"'))  (pure:m !>('"+reply+"'))"
                payload=codec.jam(int.from_bytes(code.encode(),'little'))
                print('THREAD_PROBE khan-poke',flush=True)
                status,body=http(18081,'/spider/base/ted-eval/khan-eval/noun',payload,
                    content='application/x-urb-jam',accept='application/json' if reject else 'application/x-urb-jam',timeout=60)
                item={'reject':reject,'status':status,'bytes':len(body),'nonce':message,'expected':reply}
                (lab/('khan-rejected.bin' if reject else 'khan-accepted.bin')).write_bytes(body)
                result['thread_receipts'].append(item);save()
                if reject:
                    item['response']=json.loads(body)
                    if status!=500 or 'poke-hi-fail' not in json.dumps(item['response']):raise RuntimeError('expected Gall rejection absent')
                else:
                    atom=codec.cue_atom(body);item['response']=atom.to_bytes((atom.bit_length()+7)//8,'little').decode()
                    if status!=200 or item['response']!=reply:raise RuntimeError('matching thread acknowledgement absent')
            result['ok']=True;result['receipt_basis']='typed local khan thread: matched poke acknowledgement and deliberate Gall rejection'
        elif guard.get('channel'):
            result['transport']='eyre-channel'
            for message,reject in [(nonce,False),('fail',True)]:
                channel='/~/channel/omabit-probe-'+uuid.uuid4().hex
                result['attempt']={'sender':'~zod','receiver':'~zod','nonce':message,'state':'pending'};save()
                payload=[{'id':1,'action':'poke','ship':'zod','app':'hood','mark':'helm-hi','json':message}]
                print('THREAD_PROBE channel-poke',flush=True)
                try:
                    status,_=http(18081,channel,json.dumps(payload).encode(),content='application/json',method='PUT')
                    if status not in (200,204):raise RuntimeError('channel submission refused')
                    request=urllib.request.Request('http://127.0.0.1:18081'+channel,headers={'Accept':'text/event-stream'})
                    with opener.open(request,timeout=30) as stream:
                        if stream.status!=200:raise RuntimeError('channel stream refused')
                        event_id=None;data=[];size=0
                        for _ in range(100):
                            line=stream.readline(8193);size+=len(line)
                            if not line or len(line)>8192 or size>65536:raise RuntimeError('invalid stream bound/EOF')
                            line=line.decode().rstrip('\r\n')
                            if line.startswith('id:'):event_id=line[3:].strip()
                            elif line.startswith('data:'):data.append(line[5:].strip())
                            elif not line and data:
                                event=json.loads('\n'.join(data))
                                if event.get('id')==1 and event.get('response')=='poke':
                                    if reject:
                                        if event.get('err') is None or 'ok' in event:raise RuntimeError('deliberate application rejection absent')
                                    elif event.get('ok')!='ok' or event.get('err') is not None:raise RuntimeError('poke rejected')
                                    if event_id is None or not event_id.isdigit():raise RuntimeError('missing SSE identity')
                                    result['attempt']['state']='rejected' if reject else 'acknowledged'
                                    if reject:result['application_nack']={'ack':event,'event_id':event_id}
                                    else:result['self_messages'].append({'nonce':message,'ack':event,'event_id':event_id})
                                    break
                                data=[];event_id=None
                        else:raise RuntimeError('matching acknowledgement absent')
                    status,_=http(18081,channel,json.dumps([{'action':'ack','event-id':int(event_id)}]).encode(),content='application/json',method='PUT')
                    if status not in (200,204):raise RuntimeError('channel ack refused')
                finally:
                    status,_=http(18081,channel,b'[{"action":"delete"}]',content='application/json',method='PUT')
                    result['channel_deleted']=status in (200,204)
                    if not result['channel_deleted']:raise RuntimeError('private channel cleanup failed')
            result['ok']=True;result['receipt_basis']='matched Gall application acknowledgement and deliberate application rejection'
            if guard.get('remote_submission'):
                channel='/~/channel/omabit-remote-'+uuid.uuid4().hex
                payload=[{'id':1,'action':'poke','ship':'nec','app':'hood','mark':'helm-hi','json':'omabit-native-remote-'+uuid.uuid4().hex[:24]}]
                status,body=http(18081,channel,json.dumps(payload).encode(),content='application/json',method='PUT')
                # This new fake carries no private data. Preserve the bounded
                # refusal for protocol diagnosis; this does not prove delivery.
                (lab/'remote-submission-response.bin').write_bytes(body)
                result['remote_submission']={'status':status,'bytes':len(body),'sha256':sha(lab/'remote-submission-response.bin')}
                status,_=http(18081,channel,b'[{"action":"delete"}]',content='application/json',method='PUT')
                result['remote_channel_deleted']=status in (200,204)
                result['ok']=result['remote_submission']['status'] in (200,204) and result['remote_channel_deleted']
                result['receipt_basis']='local receipts plus remote submission only; peer delivery and restart untested'

        else:
            noun=(0,int.from_bytes(nonce.encode(),'little'));encoded=codec.jam(noun)
            digits=format(int.from_bytes(encoded,'little'),'x');first=len(digits)%4 or 4
            grouped=digits[:first]+''.join('.'+digits[i:i+4] for i in range(first,len(digits),4))
            expression="=([~zod '"+nonce+"'] (cue 0x"+grouped+'))'
            result['stage']='codec';save()
            result['codec_equal']=lens(expression).strip();save()
            if result['codec_equal']!='%.y':raise RuntimeError('runtime disagrees with jam encoding')
            for name,expression in [('hi','.^(@t %cx %/ted/hi/hoon)'),('spider','.^(@t %cx %/app/spider/hoon)')]:
                result['stage']='source-'+name;save()
                value=lens(expression)
                (lab/(name+'-source.json')).write_text(json.dumps(value)+'\n')
                result[name+'_source_sha256']=sha(lab/(name+'-source.json'));save()
            result['stage']='self-message';save()
            result['attempt']={'sender':'~zod','receiver':'~zod','nonce':nonce,'state':'pending'};save();print('THREAD_PROBE self-message',flush=True)
            try:
                status,body=http(18081,'/spider/base/noun/hi/noun',encoded,content='application/x-urb-jam',accept='application/x-urb-jam',timeout=30)
                result['thread_response']={'status':status,'bytes':len(body),'body_hex':body[:2048].hex()}
                if status!=200:raise RuntimeError('thread HTTP failure')
                atom=codec.cue_atom(body);reply=atom.to_bytes((atom.bit_length()+7)//8,'little').decode()
                if reply!='hi ~zod successful':raise RuntimeError('unexpected acknowledgement')
                result['attempt']['state']='acknowledged';result['self_messages'].append({'nonce':nonce,'ack':reply});result['ok']=True
            except Exception as error:
                result['thread_error']=type(error).__name__;raise
            finally:
                try:
                    status,body=http(18081,'/~/scry/spider/tree.noun');result['spider_after']={'status':status,'jam_hex':body[:8192].hex()}
                except Exception as error:result['spider_after_error']=type(error).__name__
    except Exception as error:
        result['ok']=False;result['error']=type(error).__name__+': '+str(error);print('THREAD_PROBE failure '+type(error).__name__,flush=True)
    finally:
        if child.poll() is None:os.killpg(child.pid,signal.SIGTERM)
        try:child.wait(timeout=60)
        except subprocess.TimeoutExpired:os.killpg(child.pid,signal.SIGKILL);child.wait(timeout=5)
        reader.join(timeout=5);result['clean_shutdown']=child.returncode==0
        result['log_truncated']=overflow.is_set();result['ok']=result['ok'] and result['clean_shutdown'] and not overflow.is_set();save()
        print('THREAD_PROBE complete '+json.dumps({'ok':result['ok'],'clean_shutdown':result['clean_shutdown']}),flush=True)
    return 0 if result['ok'] else 1

def host(args):
    if args.application_state and not args.peer:raise ValueError('--application-state requires --peer')
    if args.plugin_transport and (not args.peer or args.application_state or not args.theme_source or not args.dms_source):raise ValueError('--plugin-transport requires --peer and both plugin sources, without --application-state')
    if not args.plugin_transport and (args.theme_source or args.dms_source):raise ValueError('plugin sources require --plugin-transport')
    output=args.output.absolute()
    if output.parent!=Path('/tmp') or not re.fullmatch('omabit-thread-probe-[A-Za-z0-9-]+',output.name) or output.exists():raise ValueError('fresh private output required')
    if sha(args.vere)!=VERE_SHA or sha(args.pill)!=PILL_SHA:raise ValueError('pinned input mismatch')
    plugin_admission=None;storage=None
    if args.plugin_transport:
        spec=importlib.util.spec_from_file_location('lab_storage',Path(__file__).parents[1]/'imports/storage.py');storage=importlib.util.module_from_spec(spec);spec.loader.exec_module(storage)
        memory=int(next(row.split()[1] for row in Path('/proc/meminfo').read_text().splitlines() if row.startswith('MemAvailable:')))*1024
        if memory<16*1024**3:raise RuntimeError('insufficient memory for private plugin control')
        plugin_admission={'memory_available_bytes':memory,'storage':storage.admit('/tmp',8*1024**3)}
    os.umask(0o077);output.mkdir();artifacts=output/'artifacts';artifacts.mkdir();lab=output/'lab';lab.mkdir();(lab/'home').mkdir()
    for source,target in [(args.vere,'urbit'),(args.pill,'boot.pill'),(Path(__file__),'thread_probe.py'),(Path(__file__).with_name('network_case.py'),'network_case.py'),(Path(__file__).with_name('peer_probe.py'),'peer_probe.py'),(Path(__file__).with_name('state_case.py'),'state_case.py'),(Path(__file__).with_name('omabit-state.hoon'),'omabit-state.hoon')]:shutil.copyfile(source,artifacts/target)
    (artifacts/'urbit').chmod(0o700)
    plugins={}
    if args.plugin_transport:
        shutil.copyfile(Path(__file__).with_name('plugin_case.py'),artifacts/'plugin_case.py')
        stage_path=Path(__file__).parents[1]/'app-connections/qualify.py'
        spec=importlib.util.spec_from_file_location('plugin_stage',stage_path);staging=importlib.util.module_from_spec(spec);spec.loader.exec_module(staging)
        for kind,source in [('theme',args.theme_source),('dms',args.dms_source)]:plugins[kind]=staging.stage_plugin(source,artifacts/'plugins'/kind,kind)
        plugins['stager_sha256']=sha(stage_path)
    guard={'unit':'omabit-thread-probe-'+uuid.uuid4().hex,'host_net':os.readlink('/proc/self/ns/net'),'host_pid':os.readlink('/proc/self/ns/pid'),'channel':args.channel,'remote_submission':args.remote_submission,'khan_receipt':args.khan_receipt,'peer':args.peer,'application_state':args.application_state,'plugin_transport':args.plugin_transport}
    (output/'guard.json').write_text(json.dumps(guard)+'\n')
    (output/'inputs.json').write_text(json.dumps({'guard':guard,'plugins':plugins,'plugin_admission':plugin_admission,'sha256':{p.relative_to(artifacts).as_posix():sha(p) for p in sorted(artifacts.rglob('*')) if p.is_file()}},indent=2)+'\n')
    command=['systemd-run','--user','--scope','--unit='+guard['unit'],'--property=MemoryMax='+('8G' if args.peer else '4G'),'--property=MemorySwapMax=0','--property=CPUQuota=100%','--property=TasksMax=128','--','bwrap','--unshare-all','--die-with-parent','--clearenv','--ro-bind','/usr','/usr','--symlink','usr/bin','/bin','--symlink','usr/lib','/lib','--symlink','usr/lib','/lib64','--proc','/proc','--dev','/dev','--tmpfs','/tmp','--dir','/run','--ro-bind',str(artifacts),'/artifacts','--ro-bind',str(output/'guard.json'),'/run/omabit-thread-probe.json','--bind',str(lab),'/lab','--chdir','/lab','--setenv','HOME','/lab/home','--setenv','PATH','/usr/bin:/bin','--setenv','LANG','C.UTF-8','--','/usr/bin/python3','/artifacts/thread_probe.py','--inside']
    (output/'command.json').write_text(json.dumps(command,indent=2)+'\n')
    with (output/'console.log').open('wb') as log:
        child=subprocess.Popen(command,stdout=log,stderr=subprocess.STDOUT,start_new_session=True)
        try:
            deadline=time.monotonic()+15;admitted=None
            while child.poll() is None and time.monotonic()<deadline:
                q=subprocess.run(['systemctl','--user','show',guard['unit']+'.scope','--property=ActiveState,MemoryMax,MemorySwapMax,CPUQuotaPerSecUSec,TasksMax'],capture_output=True,text=True,timeout=5)
                values=dict(line.split('=',1) for line in q.stdout.splitlines() if '=' in line)
                if values.get('ActiveState')=='active':
                    if any(values.get(k)!=v for k,v in {'MemoryMax':str((8 if args.peer else 4)*1024**3),'MemorySwapMax':'0','CPUQuotaPerSecUSec':'1s','TasksMax':'128'}.items()):raise RuntimeError('scope limits mismatch')
                    admitted=values;break
                time.sleep(.1)
            if not admitted:raise RuntimeError('scope not admitted')
            (output/'admission.json').write_text(json.dumps(admitted,indent=2)+'\n');(lab/'admitted').write_text(guard['unit'])
            if args.plugin_transport:
                deadline=time.monotonic()+900
                while child.poll() is None:
                    if time.monotonic()>deadline:raise RuntimeError('private plugin control deadline exceeded')
                    if storage.available('/tmp')['available_bytes']<2*1024**3:raise RuntimeError('private plugin control storage reserve reached')
                    try:child.wait(timeout=5)
                    except subprocess.TimeoutExpired:pass
                code=child.returncode
            else:code=child.wait(timeout=900 if args.peer else 440)
        finally:
            if child.poll() is None:os.killpg(child.pid,signal.SIGTERM);child.wait(timeout=10)
    (output/'exit.json').write_text(json.dumps({'exit':code})+'\n');print(json.dumps({'output':str(output),'exit':code}));return code

if __name__=='__main__':
    p=argparse.ArgumentParser(description=__doc__);p.add_argument('--inside',action='store_true');p.add_argument('--channel',action='store_true');p.add_argument('--remote-submission',action='store_true');p.add_argument('--khan-receipt',action='store_true');p.add_argument('--peer',action='store_true');p.add_argument('--application-state',action='store_true');p.add_argument('--plugin-transport',action='store_true');p.add_argument('--theme-source',type=Path);p.add_argument('--dms-source',type=Path);p.add_argument('--vere',type=Path);p.add_argument('--pill',type=Path);p.add_argument('--output',type=Path);a=p.parse_args()
    raise SystemExit(inside() if a.inside else host(a))
