#!/usr/bin/env python3
"""Run one persistent-app probe phase only in the admitted offline runner VM."""
import argparse
import fcntl
import hashlib
import json
import os
from pathlib import Path
import re
import stat
import subprocess
import tempfile
import time
from runner_app_state import PHASES, PEER_PHASES, ANCHOR_PHASES, execute_phase, restore_recovery, save_recovery
from state_case import Recovery, parse_sse, matching_state

ROOT=Path('/opt/omabit-lab')
SPEC=Path('/home/lab/project-state-spec.json')
STATE=Path('/home/lab/project-app-state.json')
SUCCESSION=Path('/home/lab/project-app-succession.json')
SESSION_ROOT=Path('/run/user/1000/omabit-runner-app-sessions')

def admit_guest():
    if os.geteuid()!=1000 or os.environ.get('HOME')!='/home/lab' or os.environ.get('XDG_RUNTIME_DIR')!='/run/user/1000':raise ValueError('guest owner')
    for path,want in [(ROOT/'guest-marker','offline-import-guest-v1\n'),(Path('/sys/class/dmi/id/sys_vendor'),'QEMU\n'),(Path('/proc/1/comm'),'systemd\n'),(Path('/etc/hostname'),'omabit-disposable-import\n')]:
        if path.read_text()!=want:raise ValueError('guest marker')
    if 'omabit.disposable-guest=offline-import-v1' not in Path('/proc/cmdline').read_text():raise ValueError('guest boot')
    if sorted(p.name for p in Path('/sys/class/net').iterdir())!=['lo']:raise ValueError('guest network')

def private_json(path,limit):
    fd=os.open(path,os.O_RDONLY|os.O_CLOEXEC|os.O_NOFOLLOW)
    with os.fdopen(fd,'rb') as stream:
        info=os.fstat(stream.fileno())
        if not stat.S_ISREG(info.st_mode) or info.st_uid!=1000 or stat.S_IMODE(info.st_mode)!=0o600 or info.st_nlink!=1 or info.st_size>limit:raise ValueError('private fixture file')
        raw=stream.read(limit+1)
    if len(raw)>limit:raise ValueError('fixture file bound')
    return json.loads(raw),hashlib.sha256(raw).hexdigest()

def validate_spec(spec):
    if not isinstance(spec,dict) or type(spec.get('format')) is not int or spec['format']!=1 or not isinstance(spec.get('ships'),dict) or set(spec['ships'])!={'zod','nec'}:raise ValueError('fixture spec')
    for key,pattern in [('runner_id','runner-[0-9a-f]{32}'),('project_id','project-[0-9a-f]{32}'),('anchor_id','[0-9a-f]{64}'),('namespace','net:\\[[1-9][0-9]*\\]')]:
        if not isinstance(spec.get(key),str) or not re.fullmatch(pattern,spec[key]):raise ValueError('fixture binding')
    expected={'org.urbit-runner.project':'true','org.urbit-runner.project.runner':spec['runner_id'],'org.urbit-runner.project.project':spec['project_id']}
    if not isinstance(spec.get('anchor_labels'),dict) or any(spec['anchor_labels'].get(k)!=v for k,v in expected.items()):raise ValueError('anchor owner')
    if spec.get("peer_mode","stopped") not in ("stopped","running"):raise ValueError("fixture peer mode")
    if type(spec.get("peer_cycle",False)) is not bool or spec.get("peer_cycle") and spec.get("peer_mode")!="running":raise ValueError("fixture peer cycle")
    if type(spec.get('anchor_cycle',False)) is not bool or spec.get('anchor_cycle') and not spec.get('peer_cycle'):raise ValueError('fixture anchor cycle')
    transport=spec.get('http_transport','namespace')
    if transport not in ('namespace','published-loopback'):raise ValueError('fixture HTTP transport')
    if transport=='published-loopback':
        for ship,port in [('zod',18381),('nec',18382)]:
            if type(spec['ships'].get(ship,{})) is not dict or type(spec['ships'][ship].get('published_port')) is not int or spec['ships'][ship]['published_port']!=port:raise ValueError('fixture published binding')
    ports=[]
    for ship,item in spec['ships'].items():
        if not isinstance(item,dict) or not isinstance(item.get('pier'),str) or not isinstance(item.get('identity'),dict):raise ValueError('fixture ship spec')
        port=item.get('http_port');path=Path(item['pier'])
        if type(port) is not int or not 1024<=port<=65535 or port in ports:raise ValueError('fixture endpoint')
        ports.append(port)
        if str(path) not in [root+'/fakeships/vm-'+ship+'/'+ship for root in ('/home/lab/piers','/srv/urbit/piers')]:raise ValueError('fixture pier path')
        for key in ('device','inode','urb_device','urb_inode'):
            if type(item.get('identity',{}).get(key)) is not int or item['identity'][key]<=0:raise ValueError('fixture pier identity')
    return spec

def validate_succession(spec,state,receipt):
    # The guarded Go fixture writes this only after validating the completed
    # persisted recovery. Keep the original spec immutable and retain its digest.
    if not spec.get('anchor_cycle') or state.get('attempts',[])[-1:]!=[{'phase':'after-anchor','status':'running'}]:raise ValueError('succession outside recovery phase')
    if not isinstance(receipt,dict) or set(receipt)!={'format','spec_sha256','runner_id','project_id','anchor_id','previous_namespace','namespace','operation_id'} or type(receipt['format']) is not int or receipt['format']!=1:raise ValueError('succession receipt')
    for key in ('runner_id','project_id','anchor_id'):
        if receipt[key]!=spec[key]:raise ValueError('foreign succession')
    if receipt['spec_sha256']!=state.get('spec_sha256') or not re.fullmatch('[0-9a-f]{64}',receipt['spec_sha256']):raise ValueError('succession source changed')
    if receipt['previous_namespace']!=spec['namespace'] or receipt['namespace']==spec['namespace'] or not isinstance(receipt['namespace'],str) or not re.fullmatch(r'net:\[[1-9][0-9]*\]',receipt['namespace']):raise ValueError('invalid namespace succession')
    if not isinstance(receipt['operation_id'],str) or not re.fullmatch('op-[0-9a-f]{32}',receipt['operation_id']):raise ValueError('succession operation')
    return receipt['namespace']

class Probe:
    def __init__(self,spec,state,save):
        self.spec=spec;self.state=state;self.save=save
        self.published=spec.get("http_transport")=="published-loopback"
        self.command=['/usr/bin/docker','--config','/home/lab/.local/share/urbit-runner/managed/project-resources-'+spec['project_id']+'/docker-config','--host','unix:///run/user/1000/urbit-docker.sock']
        self.environment={'HOME':'/home/lab','XDG_RUNTIME_DIR':'/run/user/1000','PATH':'/usr/bin:/bin','LC_ALL':'C'}
    def run(self,args,data=None,timeout=30,allowed=(0,)):
        try:r=subprocess.run(self.command+args,input=data,capture_output=True,env=self.environment,timeout=timeout)
        except subprocess.TimeoutExpired:raise RuntimeError('private command deadline') from None
        if r.returncode not in allowed or len(r.stdout)>65536 or len(r.stderr)>8192:raise RuntimeError('bounded private command failed')
        return r.stdout
    def run_guest_curl(self,args,data=None,timeout=30,allowed=(0,)):
        if not args or args[0]!='/usr/bin/curl':raise ValueError('fixture direct client command')
        try:r=subprocess.run(args,input=data,capture_output=True,env=self.environment,timeout=timeout)
        except subprocess.TimeoutExpired:raise RuntimeError('direct private HTTP deadline; request not replayed') from None
        if r.returncode not in allowed or len(r.stdout)>65536 or len(r.stderr)>8192:raise RuntimeError('direct private HTTP failed; request not replayed')
        return r.stdout
    def session_root(self):
        # Main admits this exact no-NIC guest before constructing Probe. Session
        # bytes live only on its runtime tmpfs, never the retained evidence disk.
        parent=SESSION_ROOT.parent
        info=parent.lstat()
        if not stat.S_ISDIR(info.st_mode) or info.st_uid!=1000 or stat.S_IMODE(info.st_mode)!=0o700:raise ValueError('guest runtime directory')
        mounts=[]
        for line in Path('/proc/self/mountinfo').read_text().splitlines():
            fields=line.split();split=fields.index('-');mount=Path(fields[4])
            if parent==mount or mount in parent.parents:mounts.append((len(mount.parts),fields[split+1]))
        if not mounts or max(mounts)[1]!='tmpfs':raise ValueError('session storage must be guest tmpfs')
        try:SESSION_ROOT.mkdir(mode=0o700)
        except FileExistsError:pass
        info=SESSION_ROOT.lstat()
        if not stat.S_ISDIR(info.st_mode) or info.st_uid!=1000 or stat.S_IMODE(info.st_mode)!=0o700:raise ValueError('private guest session directory')
        return SESSION_ROOT
    def published_cookie(self,ship,create=False):
        if ship not in ('zod','nec'):raise ValueError('fixture session identity')
        path=self.session_root()/(ship+'.cookie')
        if create:
            fd=os.open(path,os.O_WRONLY|os.O_CREAT|os.O_EXCL|os.O_NOFOLLOW|os.O_CLOEXEC,0o600);os.close(fd)
        info=path.lstat()
        if not stat.S_ISREG(info.st_mode) or info.st_uid!=1000 or stat.S_IMODE(info.st_mode)!=0o600 or info.st_nlink!=1 or info.st_size>8192:raise ValueError('private guest session file')
        return str(path)
    def verify_anchor(self):
        rows=json.loads(self.run(['container','inspect',self.spec['anchor_id']]))
        if len(rows)!=1 or rows[0]['Id']!=self.spec['anchor_id'] or rows[0]['Config']['Labels']!=self.spec['anchor_labels'] or not rows[0]['State']['Running']:raise ValueError('anchor changed')
        host=rows[0].get('HostConfig',{});mounts=rows[0].get('Mounts',[])
        if host.get('ReadonlyRootfs') is not True or host.get('Mounts')!=[{'Type':'tmpfs','Target':'/urbit','TmpfsOptions':{'SizeBytes':1048576,'Mode':448}}] or len(mounts)!=1 or mounts[0].get('Type')!='tmpfs' or mounts[0].get('Destination')!='/urbit' or mounts[0].get('RW') is not True:raise ValueError('anchor private cookie mount changed')
        namespace=self.run(['exec',self.spec['anchor_id'],'/usr/bin/readlink','/proc/self/ns/net']).decode().strip()
        expected=self.spec['namespace']
        if self.state.get('attempts',[])[-1:]==[{'phase':'after-anchor','status':'running'}]:
            receipt,digest=private_json(SUCCESSION,65536)
            expected=validate_succession(self.spec,self.state,receipt)
            self.state['succession']={'receipt':receipt,'sha256':digest};self.save()
        if namespace!=expected:raise ValueError('anchor namespace changed')
    def http(self,ship,path,data=None,content=None,accept=None,admin=False,method=None,login=False):
        if ship not in ('zod','nec') or not path.startswith('/') or path.startswith('//') or '\r' in path or '\n' in path:raise ValueError('fixture endpoint')
        if login and (admin or path!='/~/login'):raise ValueError('fixture session capture')
        item=self.spec['ships'][ship];direct=self.published and not admin
        port=item['admin_port'] if admin else item['published_port'] if direct else item['http_port']
        args=['/usr/bin/curl','--disable','--noproxy','*','--proxy','','--silent','--show-error','--max-time','90','--max-filesize','65536','--write-out','\n%{http_code}']
        cookie=self.published_cookie(ship,create=login) if direct else '/urbit/omabit-runner-app-'+ship+'-cookie'
        if login:args+=['--cookie-jar',cookie]
        elif not admin:args+=['--cookie',cookie]
        if method:
            if method!='PUT':raise ValueError('fixture HTTP method')
            args+=['--request',method]
        if content:args+=['--header','Content-Type: '+content]
        if accept:args+=['--header','Accept: '+accept]
        if data is not None:args+=['--data-urlencode' if login else '--data-binary','password@-' if login else '@-']
        args+=['http://127.0.0.1:'+str(port)+path]
        output=self.run_guest_curl(args,data=data,timeout=105) if direct else self.run(['exec','-i',self.spec['anchor_id'],'/bin/sh','-c','umask 077; exec "$@"','omabit-curl',*args],data=data,timeout=105)
        body,sep,status=output.rpartition(b'\n')
        if not sep or not re.fullmatch(b'[0-9]{3}',status):raise ValueError('fixture HTTP status')
        requests=self.state.setdefault('requests',[])
        if len(requests)>=1024:raise ValueError('fixture request bound')
        requests.append({'ship':ship,'path':path,'method':method or ('POST' if data is not None else 'GET'),'admin':admin,'status':int(status),'response_bytes':len(body)})
        if self.published:requests[-1]['transport']='published-loopback' if direct else 'namespace-admin'
        self.save();return int(status),body
    def login_once(self,ship):
        item=self.spec['ships'][ship];pier=Path(item['pier'])
        for path in [pier,*pier.parents]:
            if path.is_symlink():raise ValueError('linked fixture pier')
        inode=pier.stat();urb=(pier/'.urb').lstat();identity=item['identity']
        if (inode.st_dev,inode.st_ino,urb.st_dev,urb.st_ino)!=(identity['device'],identity['inode'],identity['urb_device'],identity['urb_inode']) or not stat.S_ISDIR(urb.st_mode):raise ValueError('fixture pier replaced')
        ports=pier/'.http.ports';info=ports.lstat()
        if not stat.S_ISREG(info.st_mode) or info.st_size>1024:raise ValueError('fixture port inventory')
        rows=[row.split() for row in ports.read_text().splitlines()]
        if not rows or any(len(row)!=3 or not row[0].isdigit() or row[1] not in ('secure','insecure') or row[2] not in ('loopback','public') for row in rows):raise ValueError('fixture port row')
        loop=[int(port) for port,secure,exposure in rows if (secure,exposure)==('insecure','loopback')]
        public=[int(port) for port,secure,exposure in rows if (secure,exposure)==('insecure','public')]
        if len(loop)!=1 or public!=[item['http_port']] or not 1024<=loop[0]<=65535:raise ValueError('fixture port binding')
        item['admin_port']=loop[0]
        def dojo(expression):return self.http(ship,'/',json.dumps({'source':{'dojo':expression},'sink':{'stdout':None}}).encode(),content='application/json',admin=True)
        status,body=dojo('our')
        if status!=200 or json.loads(body).strip()!='~'+ship:raise ValueError('fixture admin identity')
        status,body=dojo('+code')
        if status!=200:raise RuntimeError('fixture code unavailable')
        code=json.loads(body).strip().removeprefix('~')
        if not re.fullmatch('[a-z]{6}(?:-[a-z]{6}){3}',code):raise ValueError('fixture code format')
        status,_=self.http(ship,'/~/login',code.encode(),login=True);del code,body
        if not 200<=status<300:raise RuntimeError('fixture login refused')
        status,body=self.http(ship,'/~/scry/hood/kiln/pikes.json')
        if status!=200 or 'base' not in json.loads(body):raise RuntimeError('fixture authenticated inventory missing')
        self.state.setdefault('logins',[]).append({'ship':ship,'authenticated':True});self.save()
    def events(self,channel,cursor,expected,initial):
        if not re.fullmatch('/~/channel/omabit-state-[0-9a-f]{32}',channel):raise ValueError('fixture channel')
        rows=[];ack=not initial;fact=False;deadline=time.monotonic()+45
        while time.monotonic()<deadline:
            args=['/usr/bin/curl','--disable','--noproxy','*','--proxy','','--silent','--show-error','--no-buffer','--max-time','3','--max-filesize','65536','--cookie',self.published_cookie('zod') if self.published else '/urbit/omabit-runner-app-zod-cookie','--header','Accept: text/event-stream','--write-out','\n%{http_code}']
            if cursor is not None:args+=['--header','Last-Event-ID: '+str(cursor)]
            args+=['http://127.0.0.1:'+str(self.spec['ships']['zod']['published_port' if self.published else 'http_port'])+channel]
            output=self.run_guest_curl(args,timeout=15,allowed=(0,28)) if self.published else self.run(['exec',self.spec['anchor_id'],*args],timeout=15,allowed=(0,28));body,sep,status=output.rpartition(b'\n')
            if not sep or status!=b'200':raise RuntimeError('fixture SSE refused')
            for identity,event in parse_sse(body.splitlines(keepends=True)):
                rows.append((identity,event));cursor=identity
                fact=matching_state(event,expected) or fact
                ack=ack or(event.get('response')=='subscribe' and type(event.get('id')) is int and event['id']==1 and event.get('ok')=='ok')
            if len(rows)>100:raise ValueError('fixture event bound')
            if fact and ack:return rows
        raise RuntimeError('fixture SSE deadline')

def run_phase_work(spec,current,phase,commit):
    recovery=None
    def save():
        if recovery is not None:save_recovery(current,recovery)
        commit(current)
    probe=Probe(spec,current,save);probe.verify_anchor()
    recovery=Recovery(probe.http,probe.events,save,lambda *a,**kw:None,target='zod',sender='nec' if phase==PHASES[0] or spec.get('peer_mode')=='running' else 'zod')
    restore_recovery(current,recovery)
    if phase==PHASES[0]:
        probe.login_once('zod');probe.login_once('nec')
        recovery.install(Path(spec['ships']['zod']['pier']),ROOT/'omabit-state.hoon')
        recovery.before_restart()
    elif phase in (PHASES[2],"before-peer","before-anchor"):
        recovery.report['ok']=False;recovery.before_restart()
    else:
        recovery.after_restart()
        current.setdefault('completed_cycles',[]).append(json.loads(json.dumps(recovery.report)))
    save()

def main():
    parser=argparse.ArgumentParser(description=__doc__);parser.add_argument('phase',choices=ANCHOR_PHASES);args=parser.parse_args()
    admit_guest() # No private file or engine access before all guest guards.
    spec,digest=private_json(SPEC,65536);validate_spec(spec)
    lock=os.open(str(STATE)+'.lock',os.O_CREAT|os.O_RDWR|os.O_CLOEXEC|os.O_NOFOLLOW,0o600)
    with os.fdopen(lock,'wb') as held:
        info=os.fstat(held.fileno())
        if not stat.S_ISREG(info.st_mode) or info.st_uid!=1000 or info.st_nlink!=1 or stat.S_IMODE(info.st_mode)!=0o600:raise ValueError('fixture lock ownership')
        fcntl.flock(held,fcntl.LOCK_EX|fcntl.LOCK_NB)
        if STATE.exists():state,_=private_json(STATE,2*1024**2)
        else:state={'format':1,'spec_sha256':digest,'attempts':[]}
        for key in ('peer_cycle','anchor_cycle'):
            if not state['attempts'] and spec.get(key):state[key]=True
            if state.get(key,False)!=spec.get(key,False):raise ValueError('fixture cycle changed')
        if state.get('spec_sha256')!=digest:raise ValueError('fixture spec changed')
        def commit(current):
            raw=(json.dumps(current,indent=2)+'\n').encode()
            if len(raw)>2*1024**2:raise ValueError('fixture report bound')
            fd,name=tempfile.mkstemp(prefix='.runner-app-',dir=STATE.parent)
            try:
                with os.fdopen(fd,'wb') as out:out.write(raw);out.flush();os.fsync(out.fileno())
                os.replace(name,STATE)
                parent=os.open(STATE.parent,os.O_RDONLY|os.O_DIRECTORY)
                try:os.fsync(parent)
                finally:os.close(parent)
            finally:
                if os.path.exists(name):os.unlink(name)
        def work(current):run_phase_work(spec,current,args.phase,commit)
        result=execute_phase(state,args.phase,commit,work)
        raw=STATE.read_bytes();print(json.dumps({'ok':True,'phase':args.phase,'all_phases_complete':result['ok'],'receipt':{'path':str(STATE),'bytes':len(raw),'sha256':hashlib.sha256(raw).hexdigest()}}))

if __name__=='__main__':
    try:main()
    except Exception as error:
        print(json.dumps({'ok':False,'error_type':type(error).__name__}));raise SystemExit(1)
