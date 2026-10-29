"""Durable, one-use native loader admission for an exact shell activation.

The external collection coordinator issues an explicit permit before enabling
native plugins. A restarted shell cannot reuse it, and a recreated component
cannot replace a claim merely because its predecessor's process/lock disappeared.
Closing admission does not assert that applications, channels or descendants have
drained. This module never clears that uncertainty or starts/stops a ship.
"""
import contextlib
import copy
import fcntl
import json
import os
from pathlib import Path
import re
import stat
import uuid

import artifacts as a
import journal as j

FORMAT = 1
MAX_BYTES = 256 << 10
CANONICAL = {role:entry[0] for role,entry in a.ROLES.items()}


def identity(pid):
    """Read only a specifically bound process, never discover other processes."""
    j.require(type(pid) is int and pid > 0)
    root = Path('/proc')/str(pid)
    try:
        row = (root/'stat').read_text().rsplit(') ',1)[1].split()
        if row[0] in ('Z','X') or root.stat().st_uid != os.geteuid(): return None
        return {'pid':pid, 'start':row[19],
                'boot':Path('/proc/sys/kernel/random/boot_id').read_text().strip(),
                'namespace':os.readlink(root/'ns/pid')}
    except (FileNotFoundError,ProcessLookupError):
        return None


def process(value):
    j.exact(value, ('pid','start','boot','namespace'))
    j.process_identity({k:value[k] for k in ('pid','start')})
    j.require(type(value['boot']) is str and re.fullmatch('[0-9a-f-]{36}',value['boot']))
    j.require(type(value['namespace']) is str and re.fullmatch(r'pid:\[[0-9]+\]',value['namespace']))


def permit(value):
    j.exact(value, ('id','request','epoch','specification','shell'))
    for key in ('id','request','epoch'):j.token(value[key])
    process(value['shell']);spec=j.specification(value['specification'])
    for role, member in spec['members'].items():
        j.token(member['profile'],a.PROFILE)
        j.require(member['profile'] != 'default' and member['plugin_id']==CANONICAL[role]+'.'+member['profile'],
                  'native collection admission requires independent supported plugin IDs')
    return copy.deepcopy(value)


def validate(value):
    j.exact(value, ('format','permit','state','close_request','claims'))
    j.require(type(value['format']) is int and value['format']==FORMAT, 'unknown admission format; preserve it')
    p=permit(value['permit'])
    j.require(value['state'] in ('open','closing'))
    if value['state']=='open':j.require(value['close_request'] is None)
    else:j.token(value['close_request'])
    j.require(type(value['claims']) is dict and set(value['claims']) <= set(j.ROLES))
    for role, claim in value['claims'].items():
        j.exact(claim, ('instance','owner','state'))
        j.token(claim['instance']);process(claim['owner'])
        j.require(claim['state'] in ('admitted','ended'))
        j.require(claim['owner']['boot']==p['shell']['boot'] and claim['owner']['namespace']==p['shell']['namespace'])
    return value


class Gate:
    """Explicit existing private directory; no defaults or implicit creation."""
    def __init__(self, root):
        self.root=Path(root);self.fd=a.directory(self.root)
        try:
            info=os.fstat(self.fd)
            j.require(info.st_uid==os.geteuid() and stat.S_IMODE(info.st_mode)==0o700, 'private admission root required')
        except BaseException:
            os.close(self.fd);raise

    def close(self):
        if self.fd is not None:os.close(self.fd);self.fd=None

    def _open(self,name,flags,mode=0o600):
        j.require(self.fd is not None, 'admission store is closed')
        return os.open(name,flags|os.O_NOFOLLOW|os.O_NONBLOCK|os.O_CLOEXEC,mode,dir_fd=self.fd)

    def _regular(self,fd):
        info=os.fstat(fd)
        j.require(stat.S_ISREG(info.st_mode) and info.st_nlink==1 and info.st_uid==os.geteuid()
                  and stat.S_IMODE(info.st_mode)==0o600 and info.st_size<=MAX_BYTES,'unsafe admission file')

    @contextlib.contextmanager
    def transaction(self):
        fd=self._open('state.lock',os.O_RDWR|os.O_CREAT)
        try:
            self._regular(fd);fcntl.flock(fd,fcntl.LOCK_EX);yield
        finally:os.close(fd)

    def _read(self, *, sealed=True):
        fd=self._open('admission.json',os.O_RDONLY)
        with os.fdopen(fd,'rb') as stream:
            self._regular(stream.fileno());data=stream.read(MAX_BYTES+1)
        j.require(len(data)<=MAX_BYTES)
        value=validate(a.object_json(data))
        if sealed:
            fd=self._open('issued.json',os.O_RDONLY)
            with os.fdopen(fd,'rb') as stream:
                self._regular(stream.fileno());seal=a.object_json(stream.read(MAX_BYTES+1))
            j.require(seal=={'format':1,'permit_sha256':j.digest(value['permit'])},'admission issuance seal differs')
        return value

    def _sync(self):
        j.require(self.fd is not None);os.fsync(self.fd)

    def _save(self,value):
        validate(value);data=j.canonical(value)+b'\n';j.require(len(data)<=MAX_BYTES)
        name='.pending-'+uuid.uuid4().hex
        fd=self._open(name,os.O_WRONLY|os.O_CREAT|os.O_EXCL)
        with os.fdopen(fd,'wb') as stream:
            stream.write(data);stream.flush();os.fsync(stream.fileno())
        os.rename(name,'admission.json',src_dir_fd=self.fd,dst_dir_fd=self.fd)
        self._sync()

    def issue(self,value):
        value=permit(value)
        with self.transaction():
            try:old=self._read(sealed=False)
            except FileNotFoundError:
                # A missing formerly-issued record is not a new empty store.
                try:os.stat('issued.json',dir_fd=self.fd,follow_symlinks=False)
                except FileNotFoundError:old=None
                else:raise j.Conflict('issued admission record is missing; preserve the fence')
            if old is not None:
                j.require(old['permit']==value, 'another activation already owns this admission root')
                self._seal(old);return copy.deepcopy(old)
            j.require(identity(value['shell']['pid'])==value['shell'], 'permit shell is not the exact live process')
            row={'format':FORMAT,'permit':value,'state':'open','close_request':None,'claims':{}}
            self._save(row);self._seal(row);return copy.deepcopy(row)

    def _seal(self,row):
        expected={'format':1,'permit_sha256':j.digest(row['permit'])}
        try:fd=self._open('issued.json',os.O_RDONLY)
        except FileNotFoundError:
            j.require(not row['claims'] and row['state']=='open','issued seal is missing from an active permit')
            name='.seal-'+uuid.uuid4().hex
            fd=self._open(name,os.O_WRONLY|os.O_CREAT|os.O_EXCL)
            with os.fdopen(fd,'wb') as stream:
                stream.write(j.canonical(expected)+b'\n');stream.flush();os.fsync(stream.fileno())
            os.rename(name,'issued.json',src_dir_fd=self.fd,dst_dir_fd=self.fd)
        else:
            with os.fdopen(fd,'rb') as stream:
                self._regular(stream.fileno());j.require(a.object_json(stream.read(MAX_BYTES+1))==expected,'admission seal changed')
        self._sync()

    def status(self):
        # Read-only, including unknown formats and missing permits. No lock file
        # creation, state repair, orphan adoption or opportunistic grant.
        row=self._read();p=row['permit']
        return {'record':row,'shell_live':identity(p['shell']['pid'])==p['shell'],
                'claim_owners_live':{r:identity(v['owner']['pid'])==v['owner'] for r,v in row['claims'].items()}}

    def stop_admission(self,permit_id,request):
        j.token(permit_id);j.token(request)
        with self.transaction():
            row=self._read();j.require(row['permit']['id']==permit_id,'wrong activation permit')
            if row['state']=='closing':
                j.require(row['close_request']==request,'another close request owns this permit')
                self._sync();return copy.deepcopy(row)
            row['state']='closing';row['close_request']=request;self._save(row);return copy.deepcopy(row)

    def claim(self,expected,role,instance):
        """A direct shell child acquires a one-use member claim before QML loads.

        ``expected`` binds the immutable artifact/profile and selected activation.
        Lost publication response may retry only in this same process and with the
        same instance token. A dead or cleanly ended guard never grants replacement.
        """
        j.require(role in j.ROLES);j.token(instance)
        j.exact(expected,('permit','epoch','collection','plugin_id','profile','artifact_sha256'))
        for key in ('permit','epoch'):j.token(expected[key])
        for key in ('collection','artifact_sha256'):j.token(expected[key],j.HASH)
        j.token(expected['plugin_id'],j.PLUGIN);j.token(expected['profile'],a.PROFILE)
        fd=self._open(role+'.lock',os.O_RDWR|os.O_CREAT)
        try:
            self._regular(fd)
            try:fcntl.flock(fd,fcntl.LOCK_EX|fcntl.LOCK_NB)
            except BlockingIOError:raise j.Conflict('another component owns this member admission') from None
            with self.transaction():
                row=self._read();p=row['permit'];member=p['specification']['members'][role]
                j.require(row['state']=='open','activation admission is closed')
                actual={'permit':p['id'],'epoch':p['epoch'],'collection':j.digest(p['specification']),
                        **{k:member[k] for k in ('plugin_id','profile','artifact_sha256')}}
                j.require(expected==actual,'selected activation differs from the durable permit')
                j.require(identity(os.getppid())==p['shell'],'only the exact permitted shell may create this component')
                owner=identity(os.getpid());j.require(owner is not None)
                claim={'instance':instance,'owner':owner,'state':'admitted'}
                if role in row['claims']:
                    j.require(row['claims'][role]==claim,'prior component claim is retained; explicit coordinator recovery required')
                    self._sync()
                else:
                    row['claims'][role]=claim;self._save(row)
                return Claim(self,fd,role,copy.deepcopy(expected),copy.deepcopy(claim))
        except BaseException:
            os.close(fd);raise


class Claim:
    def __init__(self,gate,fd,role,expected,claim):
        self.gate,self.fd,self.role,self.expected,self.record=gate,fd,role,expected,claim

    def status(self):
        j.require(self.fd is not None and identity(os.getpid())==self.record['owner'],'not the live admission owner')
        row=self.gate._read();j.require(row['permit']['id']==self.expected['permit'] and row['claims'].get(self.role)==self.record)
        live=identity(os.getppid())==row['permit']['shell']
        return {'format':1,'admitted':live,'state':row['state'] if live else 'owner-lost',
                **self.expected,'instance':self.record['instance'],'owner':self.record['owner']}

    def end(self):
        """End this guard's lease only. This is never transport/descendant closure."""
        j.require(self.fd is not None and identity(os.getpid())==self.record['owner'])
        try:
            with self.gate.transaction():
                row=self.gate._read();j.require(row['claims'].get(self.role)==self.record)
                row['claims'][self.role]['state']='ended';self.gate._save(row)
        finally:
            os.close(self.fd);self.fd=None
