#!/usr/bin/env python3
"""Fresh native fake desk protocol probe; no installed runners or host network."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import signal
import socket
import subprocess
import time
import urllib.request
import uuid

VERE_SHA = 'b5c54439ae5b9f0ff94d75319e67e04f04369776fa5e2180a5ab1a5ebd4c2ff0'
PILL_SHA = 'f8b08c53dd6d0aa9c4f7654f0a7bcd11e94089222888a3c99c0a10eedcbb377b'
def sha(path):
    with Path(path).open('rb') as f: return hashlib.file_digest(f,'sha256').hexdigest()

def inside():
    g=json.loads(Path('/guard.json').read_text());lab=Path('/lab')
    for k in ('net','pid','mnt','user'):
        if os.readlink('/proc/self/ns/'+k)==g['namespaces'][k]:raise RuntimeError('host namespace refused')
    if any(Path(p).exists() for p in ('/home','/run/user','/var/run/docker.sock')) or [n for _,n in socket.if_nameindex()]!=['lo']:raise RuntimeError('host access refused')
    end=time.monotonic()+20
    while not (lab/'admitted').exists() and time.monotonic()<end:time.sleep(.1)
    if (lab/'admitted').read_text()!=g['unit']:raise RuntimeError('scope admission absent')
    os.umask(0o077);os.nice(15)
    desk=g.get('desk','omabit-test')
    baseline_alias=desk+'-baseline'
    result={'ok':False,'steps':[],'clean_shutdown':False}
    def save(): (lab/'result.json').write_text(json.dumps(result,indent=2)+'\n')
    class NoRedirect(urllib.request.HTTPRedirectHandler):
        def redirect_request(self,*args,**kwargs):raise RuntimeError('redirect refused')
    requests=json.loads(Path('/artifacts/requests.json').read_text())
    opener=urllib.request.build_opener(urllib.request.ProxyHandler({}),NoRedirect())
    def lens(name,expression,app=None,mark=None,required=True,data_input=None):
        expression=expression.replace('omabit-test',desk).replace('omabit-baseline',baseline_alias)
        if expression.count('(')!=expression.count(')') or expression.count('[')!=expression.count(']'):raise ValueError('unbalanced probe expression '+name)
        row={'name':name,'source':expression,'app':app,'mark':mark,'state':'dispatched'};result['steps'].append(row);save()
        source={'as':{'mark':mark,'next':{'dojo':expression}}} if mark else {'dojo':expression}
        if data_input is not None:source={'hoon':{'code':expression,'next':{'data':data_input}}}
        data=json.dumps({'source':source,'sink':{'app':app} if app else {'stdout':None}}).encode()
        req=urllib.request.Request('http://127.0.0.1:12321/',data=data,headers={'Content-Type':'application/json'})
        try:
            with opener.open(req,timeout=30) as r:body=r.read(262145)
            if len(body)>262144:raise RuntimeError('response bound')
            value=json.loads(body)
            if not isinstance(value,str):raise RuntimeError('unexpected Lens response')
            row.update(state='returned',response=value);save()
            if app and value.strip()!='>=':raise RuntimeError('mutation acknowledgement absent')
            return value.strip()
        except Exception as e:
            row.update(state='unresolved',error=type(e).__name__+': '+str(e));save()
            if required:raise
    def fixed(name,key):
        request=requests[key];source=request['source'];app=request['sink'].get('app');mark=None
        if 'as' in source:mark=source['as']['mark'];source=source['as']['next']
        if 'hoon' in source:return lens(name,source['hoon']['code'],app,mark,data_input=source['hoon']['next']['data'])
        return lens(name,source['dojo'],app,mark)
    def decode_atom(value):
        atom=int(value.strip().removeprefix('0x').replace('.',''),16)
        return json.loads(atom.to_bytes((atom.bit_length()+7)//8,'little'))
    args=['/artifacts/vere','-t','-L','--no-conn','--no-dock','--loom','31','--snap-time','60','-b','127.0.0.1','--http-port','18081','-F','zod','-B','/artifacts/boot.pill','-c','/lab/zod']
    if g.get('resume'):args=args[:args.index('-F')]+['/lab/zod']
    with (lab/'vere.log').open('wb') as log:
        child=subprocess.Popen(args,stdin=subprocess.DEVNULL,stdout=log,stderr=subprocess.STDOUT,start_new_session=True)
        try:
            end=time.monotonic()+240
            while child.poll() is None and time.monotonic()<end:
                try:
                    with opener.open('http://127.0.0.1:18081/~/host',timeout=1) as r:
                        if r.read(128).strip()==b'~zod':break
                except OSError:pass
                time.sleep(1)
            else:raise RuntimeError('fake readiness deadline')
            if g.get('resume'):
                # Only the private fixture's own failed read can remain here.
                # Production adapters must not cancel someone else's Lens job.
                request=urllib.request.Request('http://127.0.0.1:12321/',data=json.dumps({'source':{'cancel':None},'sink':{'stdout':None}}).encode(),headers={'Content-Type':'application/json'})
                with opener.open(request,timeout=10) as response:cancelled=json.loads(response.read(1024))
                if cancelled!='cancelled':raise RuntimeError('private read-job cancellation failed')
                result['private_read_job_cancelled']=True;save()
            assert lens('identity','our')=='~zod' 
            print('DESK_PROBE ready',flush=True)
            json_result=lens('json-output',"`@ux`(en:json:html [%o (malt `(list [@t json])`~[['exists' [%b %.y]] ['revision' [%n '2']]])])")
            assert json_result.startswith('0x')
            assert lens('absence','(~(has in .^((set @tas) %cd /~zod//(scot %da now))) %omabit-test)')=='%.n'
            for name,expression in [
              ('hood-state','=/  v  .^(vase %gx /~zod/hood/(scot %da now)/dbug/state/noun)  q:(slap v (ream \'[ver=- kiln-version=-.kiln timer=commit-timer.kiln]\'))'),
              ('timers','.^((list [@da duct]) %bx /~zod//(scot %da now)/debug/timers)'),
              ('treaty-present','.^(? %cu /~zod/base/(scot %da now)/app/treaty/hoon)'),
              ('desks','.^((set @tas) %cd /~zod//(scot %da now))')]:lens(name,expression,required=False)
            create="=/  wen  (scot %da now)  =/  paths=(list path)  ~[/mar/noun/hoon /mar/hoon/hoon /mar/txt/hoon /mar/kelvin/hoon /sys/kelvin]  =/  files  (turn paths |=(p=path [p [(rear p) .^(* %cx (weld /~zod/base/[wen] p))]]))  (new-desk:cloy %omabit-test ~ (~(gas by *(map path page:clay)) files))"
            fixed('create','create')
            fixed('mount','mount')
            mount=lab/'zod'/desk;end=time.monotonic()+45
            while not (mount/'sys.kelvin').is_file() and time.monotonic()<end:time.sleep(.2)
            assert (mount/'sys.kelvin').is_file()
            result['generated']={str(p.relative_to(mount)):sha(p) for p in mount.rglob('*') if p.is_file()};save()
            baseline=lens('revision-before','let:.^(domo:clay %cv /~zod/omabit-test/(scot %da now))')
            inventory=requests['inventory']['source']['dojo']
            inventory_result=lens('typed-inventory',inventory)
            assert decode_atom(inventory_result)['publication']=='private'
            lens('permissions-before','.^(* %cp /~zod/omabit-test/(scot %da now))')
            (mount/'hello.txt').write_text('first source\n')
            assert lens('uncommitted','.^(? %cu /~zod/omabit-test/(scot %da now)/hello/txt)')=='%.n'
            fixed('commit','commit')
            end=time.monotonic()+45
            while time.monotonic()<end:
                revision=lens('revision-after','let:.^(domo:clay %cv /~zod/omabit-test/(scot %da now))')
                if revision!=baseline:break
                time.sleep(.5)
            else:raise RuntimeError('commit revision absent')
            lens('content','.^(* %cx /~zod/omabit-test/(scot %da now)/hello/txt)')
            lens('snapshot-mount','+hood/mount /~zod/omabit-test/'+revision+' %omabit-baseline','hood')
            snapshot=lab/'zod'/baseline_alias;end=time.monotonic()+20
            while not (snapshot/'hello.txt').is_file() and time.monotonic()<end:time.sleep(.2)
            raw=(mount/'hello.txt').read_bytes()
            encoded=format(int.from_bytes(raw,'little'),'x')
            first=len(encoded)%4 or 4;encoded=encoded[:first]+''.join('.'+encoded[i:i+4] for i in range(first,len(encoded),4))
            compare="=/  convert  .^(tube:clay %cc /~zod/omabit-test/"+revision+"/mime/txt)  =(q:(convert !>(`mime`[/text/plain ["+str(len(raw))+" 0x"+encoded+"]])) .^(* %cx /~zod/omabit-test/"+revision+"/hello/txt))"
            assert lens('mark-equivalence',compare)=='%.y'
            assert g['desk'] in decode_atom(fixed('typed-desks','desks'))
            assert ['hello','txt'] in decode_atom(fixed('typed-files','files'))
            assert fixed('structured-mark-equivalence','equivalent')=='%.y'
            assert fixed('structured-mark-difference','different')=='%.n'
            result['raw_byte_normalization']={'unix_bytes':len(raw),'clay_rendered_bytes':len((snapshot/'hello.txt').read_bytes()),'equivalent':True};save()
            (mount/'hello.txt').write_text('pending second source\n')
            assert (snapshot/'hello.txt').read_bytes()!= (mount/'hello.txt').read_bytes()
            assert fixed('typed-autocommit-inactive','autocommit')=='%.n'
            lens('autocommit-start','+hood/commit %omabit-test, =auto &','hood')
            assert fixed('typed-autocommit-active','autocommit')=='%.y'
            lens('timers-active','.^((list [@da duct]) %bx /~zod//(scot %da now)/debug/timers)',required=False)
            lens('hood-state-active','=/  v  .^(vase %gx /~zod/hood/(scot %da now)/dbug/state/noun)  q:(slap v (ream \'commit-timer.kiln\'))',required=False)
            lens('autocommit-stop','~','hood','kiln-cancel-autocommit')
            lens('timers-cancelled','.^((list [@da duct]) %bx /~zod//(scot %da now)/debug/timers)',required=False)
            assert fixed('typed-autocommit-cancelled','autocommit')=='%.n'
            fixed('publish-source','publish-source')
            assert decode_atom(fixed('inventory-published','inventory'))['publication']=='published'
            result['ok']=True
        except Exception as e:result['error']=type(e).__name__+': '+str(e)
        finally:
            if child.poll() is None:os.killpg(child.pid,signal.SIGTERM)
            try:child.wait(timeout=60)
            except subprocess.TimeoutExpired:os.killpg(child.pid,signal.SIGKILL);child.wait(timeout=5)
            result['clean_shutdown']=child.returncode==0
            result['ok']=result['ok'] and result['clean_shutdown'];save()
    print(json.dumps({'ok':result['ok'],'error':result.get('error')}),flush=True)
    return 0 if result['ok'] else 1

def host(a):
    out=a.output.absolute()
    if out.parent!=Path('/tmp') or not re.fullmatch('omabit-desk-probe-[a-z0-9-]+',out.name) or out.exists():raise ValueError('fresh private output required')
    frozen_requests=json.loads(a.requests.read_text())
    if set(frozen_requests)!={'desks','files','autocommit','inventory','create','mount','commit','publish-source','equivalent','different'}:raise ValueError('incomplete typed request fixture')
    if sha(a.vere)!=VERE_SHA or sha(a.pill)!=PILL_SHA:raise ValueError('pinned input mismatch')
    if shutil.disk_usage('/tmp').free<4*1024**3:raise RuntimeError('storage reserve absent')
    previous=None
    if a.resume_from:
        previous=a.resume_from.absolute()
        if previous.parent!=Path('/tmp') or not re.fullmatch('omabit-desk-probe-[a-z0-9-]+',previous.name) or previous.is_symlink():raise ValueError('invalid prior fixture')
        if not json.loads((previous/'lab/result.json').read_text()).get('clean_shutdown') or not (previous/'exit.json').is_file():raise ValueError('prior fixture not stopped')
        report=json.loads((previous/'lab/result.json').read_text())
        if any(row.get('app') and row.get('state')!='returned' for row in report.get('steps',[])):raise ValueError('prior fixture has an uncertain mutation')
        prior_inputs=json.loads((previous/'inputs.json').read_text())
        if prior_inputs['vere']!=VERE_SHA or prior_inputs['boot.pill']!=PILL_SHA:raise ValueError('prior fixture inputs changed')
        pier=previous/'lab/zod'
        if not pier.is_dir() or pier.is_symlink():raise ValueError('prior fixture pier unavailable')
    os.umask(0o077);out.mkdir();art=out/'artifacts';art.mkdir();lab=out/'lab';lab.mkdir();(lab/'home').mkdir()
    for source,name in [(a.vere,'vere'),(a.pill,'boot.pill'),(Path(__file__),'probe.py'),(a.requests,'requests.json')]:shutil.copyfile(source,art/name)
    (art/'vere').chmod(0o700)
    g={'unit':'omabit-desk-probe-'+uuid.uuid4().hex,'namespaces':{k:os.readlink('/proc/self/ns/'+k) for k in ('net','pid','mnt','user')}}
    g['desk']='omabit-test-'+uuid.uuid4().hex[:8]
    g['resume']=previous is not None
    if previous:
        identity=(previous/'lab/zod').stat()
        moved={'from':str(previous),'to':str(out),'device':identity.st_dev,'inode':identity.st_ino,'previous_result_sha256':sha(previous/'lab/result.json')}
        (out/'resume.json').write_text(json.dumps(moved,indent=2));(previous/'pier-transferred.json').write_text(json.dumps(moved,indent=2))
        os.rename(previous/'lab/zod',lab/'zod')
    (out/'guard.json').write_text(json.dumps(g));(out/'inputs.json').write_text(json.dumps({p.name:sha(p) for p in art.iterdir()},indent=2))
    command=['systemd-run','--user','--scope','--unit='+g['unit'],'-p','MemoryMax=4G','-p','MemorySwapMax=0','-p','CPUQuota=100%','-p','TasksMax=128','--','bwrap','--unshare-all','--die-with-parent','--clearenv','--ro-bind','/usr','/usr','--symlink','usr/bin','/bin','--symlink','usr/lib','/lib','--symlink','usr/lib','/lib64','--proc','/proc','--dev','/dev','--tmpfs','/tmp','--dir','/run','--ro-bind',str(art),'/artifacts','--ro-bind',str(out/'guard.json'),'/guard.json','--bind',str(lab),'/lab','--chdir','/lab','--setenv','HOME','/lab/home','--setenv','PATH','/usr/bin:/bin','--setenv','LANG','C.UTF-8','--','python3','-B','/artifacts/probe.py','--inside']
    (out/'command.json').write_text(json.dumps(command,indent=2))
    with (out/'console.log').open('wb') as log:
        child=subprocess.Popen(command,stdout=log,stderr=subprocess.STDOUT,start_new_session=True)
        try:
            end=time.monotonic()+15
            while child.poll() is None and time.monotonic()<end:
                p=subprocess.run(['systemctl','--user','show',g['unit']+'.scope','--property=ActiveState,MemoryMax,MemorySwapMax,CPUQuotaPerSecUSec,TasksMax'],capture_output=True,text=True,timeout=5)
                values=dict(line.split('=',1) for line in p.stdout.splitlines() if '=' in line)
                if values.get('ActiveState')=='active':
                    assert all(values.get(k)==v for k,v in {'MemoryMax':str(4*1024**3),'MemorySwapMax':'0','CPUQuotaPerSecUSec':'1s','TasksMax':'128'}.items())
                    (out/'admission.json').write_text(json.dumps(values));(lab/'admitted').write_text(g['unit']);break
                time.sleep(.1)
            else:raise RuntimeError('scope admission failed')
            end=time.monotonic()+550
            while child.poll() is None:
                if time.monotonic()>end:raise RuntimeError('outer deadline')
                usage=sum(p.stat().st_blocks*512 for p in out.rglob('*') if p.is_file())
                if usage>2*1024**3:raise RuntimeError('2 GiB scratch budget exceeded')
                try:child.wait(timeout=2)
                except subprocess.TimeoutExpired:pass
            rc=child.returncode
        finally:
            if child.poll() is None:os.killpg(child.pid,signal.SIGTERM);child.wait(timeout=10)
    (out/'exit.json').write_text(json.dumps({'exit':rc}));print(json.dumps({'output':str(out),'exit':rc}));return rc
if __name__=='__main__':
    p=argparse.ArgumentParser(description=__doc__);p.add_argument('--inside',action='store_true');p.add_argument('--output',type=Path);p.add_argument('--resume-from',type=Path);p.add_argument('--requests',type=Path);p.add_argument('--vere',type=Path);p.add_argument('--pill',type=Path);a=p.parse_args();raise SystemExit(inside() if a.inside else host(a))
