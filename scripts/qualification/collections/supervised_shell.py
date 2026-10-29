"""Private QML fixture adapter to an independently owned subreaper.

No production loader API and no journal fence resolution is implemented here.
"""
from pathlib import Path
import json,os,signal,subprocess,sys,time,uuid
from process_supervisor import guard,identity,read,sha
ROOT=Path('/lab')

class SupervisedShell:
    def __init__(self,*,env=None):
        guard();(ROOT/'receipts').mkdir(mode=0o700,exist_ok=True)
        self.token=uuid.uuid4().hex;self.path=ROOT/'receipts'/(self.token+'.json')
        self.log=(ROOT/(self.token+'.supervisor-events.jsonl')).open('w');self.err=(ROOT/(self.token+'.supervisor-stderr.log')).open('w')
        self.guardian=subprocess.Popen([sys.executable,'-B','/lab/process_supervisor.py','--guardian',self.token,'--mode','qml'],stdin=subprocess.PIPE,stdout=self.log,stderr=self.err,text=True,env=env)
        self.closed=None;self.returncode=None
        self._wait(lambda:self.path.exists(),5);self.owner=read(self.path)['owner']
        assert self.owner['pid']==self.guardian.pid and self.owner['token']==self.token
        self.send('start');self._wait(lambda:read(self.path)['phase']=='running',5)
        self.root=read(self.path)['root'];self.pid=self.root['pid']
    def _wait(self,fn,timeout):
        end=time.monotonic()+timeout
        while time.monotonic()<end:
            value=fn()
            if value:return value
            assert self.guardian.poll() is None,'Original supervisor disappeared; closure is unconfirmed'
            time.sleep(.025)
        raise subprocess.TimeoutExpired('private supervisor observation',timeout)
    def send(self,action):
        assert self.guardian.poll() is None,'Original supervisor disappeared; closure is unconfirmed'
        token=uuid.uuid4().hex;self.guardian.stdin.write(json.dumps({'action':action,'request':token})+'\n');self.guardian.stdin.flush();return token
    def poll(self):
        if self.returncode is not None:return self.returncode
        receipt=read(self.path);assert receipt['owner']==self.owner
        row=next((r for r in receipt['reaped'] if r['pid']==self.pid),None)
        if row is not None:self.returncode=row['status'];return self.returncode
        if self.guardian.poll() is not None:raise AssertionError('Supervisor loss is not a root-exit receipt')
        self.send('status');return None
    def wait(self,timeout=None):
        end=time.monotonic()+(timeout or 30)
        while time.monotonic()<end:
            status=self.poll()
            if status is not None:return status
            time.sleep(.03)
        raise subprocess.TimeoutExpired('supervised QML',timeout)
    def kill(self):
        now=identity(self.pid);assert now and now['start']==self.root['start'] and now['parent']==self.owner['pid']
        self.send('kill-root')
    def close_owned(self):
        if self.closed is not None:return self.closed
        request=self.send('stop');assert self.guardian.wait(8)==0
        self.log.close();self.err.close();receipt=read(self.path)
        assert receipt['owner']==self.owner and receipt['phase']=='closed' and receipt['proof']=='waitpid-echild' and receipt['request']==request
        assert receipt['root']==self.root and any(r['pid']==self.pid for r in receipt['reaped'])
        self.closed={'owner':self.owner,'root':self.root,'request':request,'receipt_sha256':sha(self.path),'receipt':str(self.path),'all_owned_descendants_reaped':True}
        return self.closed
