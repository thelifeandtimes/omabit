"""Version2 desktop authority linked to an immutable predecessor checkpoint.

The original authority/mirror remain intact. Publication binds every staged inode
before promotion. Initial admission on this authority requires an existing canonical
replacement transition and a freshly verified closed predecessor cohort.
"""
import copy
import os
from pathlib import Path
import uuid
import admission
import artifacts
import authority_inspection as inspection
import authority_provisioning as provisioning
import journal as j
import runtime_authority as old
import session_checkpoint as checkpoint
import session_successor as successor

FORMAT = 2
AUTHORITY_READ_FORMATS = (1, 2)
PREFIX = 'desktop-authority-'
KINDS = ('seal','mirror','root')


def filename(attempt):
    j.token(attempt);return PREFIX+attempt+'.json'


def descriptor(value):
    j.exact(value, ('format','id','root','context','coordinator','lock','coordinator_lock','successor','publication','mirror'))
    j.require(type(value['format']) is int and value['format'] == FORMAT, 'unsupported linked desktop authority')
    j.token(value['id']);successor.reference_shape(value['successor'])
    for key,kind in (('root','directory'),('coordinator','directory'),('lock','file'),('coordinator_lock','file'),('publication','file')):inspection.reference_shape(value[key],kind)
    root=Path(value['root']['path']);coordinator=Path(value['coordinator']['path']);j.require(root.name==old.ANCHOR)
    j.require(root.parent==Path(value['context']['directory']['path']) and value['lock']['path']==str(root/'authority.lock'))
    j.require(value['coordinator_lock']['path']==str(coordinator/'coordinator.lock'))
    j.require(value['mirror']==str(coordinator/('runtime-successor-'+value['id']+'.json')))
    j.require(Path(value['publication']['path']).parent==coordinator)
    return value


def shape(value):
    j.exact(value, ('format','id','descriptor','rows'));j.require(type(value['format']) is int and value['format']==1,'unknown successor publication; preserve it')
    j.token(value['id']);d=descriptor(value['descriptor']);j.exact(value['rows'],KINDS)
    j.require(d['publication']['path']==str(Path(d['coordinator']['path'])/filename(value['id'])))
    j.require(d['successor']['object']['path']==str(Path(d['coordinator']['path'])/successor.filename(value['id'])))
    targets={'root':d['root']['path'],'seal':str(Path(d['root']['path']).parent/old.ANCHOR_SEAL),'mirror':d['mirror']}
    for kind,row in value['rows'].items():
        j.exact(row, ('staging','target'));typ='directory' if kind=='root' else 'file'
        for key in ('staging','target'):inspection.reference_shape(row[key],typ)
        source=Path(row['staging']['path']);target=Path(targets[kind]);prefix='.pending-desktop-authority-'+value['id']+'-'+kind+'-'
        j.require(source.parent==target.parent and source.name.startswith(prefix));j.token(source.name.removeprefix(prefix))
        j.require(row['target']==dict(row['staging'],path=str(target)))
    j.require(value['rows']['root']['target']==d['root'])
    return value


def read(store, attempt):
    try:value=old.read(store.fd,filename(attempt))
    except FileNotFoundError:return None
    shape(value);j.require(value['id']==attempt);old.check_object(value['descriptor']['publication'],'file');return value


def linked(d):
    """Read historical links and the current journal without claiming a lease."""
    descriptor(d);reader=inspection.Reader()
    def bound(ref,maximum=old.MAX_BYTES):
        old.check_object(ref['object'],'file');body=reader.read(ref['object']['path'],maximum=maximum)
        j.require(reader.files[ref['object']['path']]['sha256']==ref['sha256'],'linked predecessor evidence changed');return body
    intent=successor.shape(bound(d['successor']))
    j.require(intent['authority']==d['id'] and intent['context']==d['context'] and intent['coordinator']==d['coordinator'] and intent['coordinator_lock']==d['coordinator_lock'])
    old.check_object(intent['claim']['target'],'file')
    j.require(reader.read(intent['claim']['target']['path'])==successor.claim_body(intent))
    j.require(reader.files[intent['claim']['target']['path']]['sha256']==intent['claim']['sha256'] and not os.path.lexists(intent['claim']['staging']['path']))
    archived=checkpoint.validate(bound(intent['checkpoint'],checkpoint.MAX_BYTES))
    j.require(archived['id']==intent['recovery'] and archived['inspection']['authority']==intent['predecessor'])
    receipt=bound(archived['cleanup']);j.require(receipt==checkpoint.parsed(archived,archived['cleanup']['object']['path']))
    j.require(intent['members']==receipt['verified']['members'] and intent['specification']==receipt['plan']['specification'] and intent['loaded']==receipt['plan']['loaded'])
    mirror=archived['files'][str(Path(d['coordinator']['path'])/old.MIRROR)];bound({key:mirror[key] for key in ('object','sha256')})
    # Journal publication is atomic and mutable. Do not treat its inode as an
    # immutable link: a concurrent legitimate append must not fence live guards.
    store=j.Store(d['coordinator']['path'])
    try:events,state=store._load()
    finally:store.close()
    prefix=intent['journal'];j.require(prefix=={key:archived['inspection']['journal'][key] for key in ('events','sha256')})
    j.require(len(events)>=prefix['events'] and j.digest(events[:prefix['events']])==prefix['sha256'])
    reader.verify();return intent,events,state


def check_staged(value,kind):
    d=value['descriptor'];row=value['rows'][kind];source=Path(row['staging']['path']);old.check_object(row['staging'],'directory' if kind=='root' else 'file')
    if kind!='root':j.require(old.read_private_json(source)[0]==d);return
    fd=artifacts.directory(source)
    try:
        j.require(set(os.listdir(fd))=={'authority.lock','authority.json','ready.json','reservation.json'})
        old.check_object(dict(d['lock'],path=str(source/'authority.lock')),'file')
        j.require(old.read(fd,'authority.lock')=={} and old.read(fd,'authority.json')==d)
        j.require(old.read(fd,'ready.json')=={'format':FORMAT,'descriptor':j.digest(d)})
        j.require(old.read(fd,'reservation.json')=={'format':1,'authority':d['id'],'reservation':None})
    finally:os.close(fd)


def allocate(store,intent):
    attempt=intent['id'];runtime=Path(intent['context']['directory']['path']);root=runtime/old.ANCHOR
    temporary='.pending-desktop-authority-record-'+attempt+'-'+uuid.uuid4().hex
    file=os.open(temporary,os.O_WRONLY|os.O_CREAT|os.O_EXCL|os.O_NOFOLLOW|os.O_CLOEXEC,0o600,dir_fd=store.fd)
    try:
        publication=dict(old.object_identity(store.root/temporary,'file'),path=str(store.root/filename(attempt)))
        stage=runtime/('.pending-desktop-authority-'+attempt+'-root-'+uuid.uuid4().hex);stage.mkdir(mode=0o700);provisioning.sync(runtime)
        fd=artifacts.directory(stage)
        try:
            old.write_new(fd,'authority.lock',{})
            d={'format':FORMAT,'id':intent['authority'],'root':dict(old.object_identity(stage,'directory'),path=str(root)),
               'context':intent['context'],'coordinator':intent['coordinator'],'coordinator_lock':intent['coordinator_lock'],
               'lock':dict(old.object_identity(stage/'authority.lock','file'),path=str(root/'authority.lock')),
               'successor':successor.reference(store.root/successor.filename(attempt)),'publication':publication,
               'mirror':str(store.root/('runtime-successor-'+intent['authority']+'.json'))}
            descriptor(d);old.write_new(fd,'authority.json',d);old.write_new(fd,'ready.json',{'format':FORMAT,'descriptor':j.digest(d)})
            old.write_new(fd,'reservation.json',{'format':1,'authority':d['id'],'reservation':None})
        finally:os.close(fd)
        rows={'root':{'staging':old.object_identity(stage,'directory'),'target':d['root']}}
        for kind,target in (('seal',runtime/old.ANCHOR_SEAL),('mirror',Path(d['mirror']))):
            name='.pending-desktop-authority-'+attempt+'-'+kind+'-'+uuid.uuid4().hex;parent=artifacts.directory(target.parent)
            try:old.write_new(parent,name,d)
            finally:os.close(parent)
            ref=old.object_identity(target.parent/name,'file');rows[kind]={'staging':ref,'target':dict(ref,path=str(target))}
        value={'format':1,'id':attempt,'descriptor':d,'rows':rows};shape(value)
        with os.fdopen(file,'wb') as stream:
            file=None;stream.write(j.canonical(value)+b'\n');stream.flush();os.fsync(stream.fileno())
        os.fsync(store.fd);provisioning.rename_no_replace(store.fd,temporary,filename(attempt));return value
    finally:
        if file is not None:os.close(file)


def published(value,kind):
    row=value['rows'][kind];old.check_object(row['target'],'directory' if kind=='root' else 'file')
    j.require(not os.path.lexists(row['staging']['path']),'both staged and published successor objects exist')
    if kind!='root':j.require(old.read_private_json(row['target']['path'])[0]==value['descriptor'])


def check_missing_record(store,intent):
    # A complete stage can point to a publication record that was later lost.
    # Only the still-present original temporary record inode proves that stage
    # never reached its fixed record pathname. Keep it, but do not adopt it.
    runtime=Path(intent['context']['directory']['path']);attempt=intent['id']
    roots=list(runtime.glob('.pending-desktop-authority-'+attempt+'-root-*'))
    temporary=list(store.root.glob('.pending-desktop-authority-record-'+attempt+'-*'))
    j.require(len(roots)<=128 and len(temporary)<=128,'successor staging history is full; preserve it')
    refs=[old.object_identity(path,'file') for path in temporary]
    for root in roots:
        path=root/'authority.json'
        if not os.path.lexists(path):continue  # Incomplete, never-published stage.
        d=descriptor(old.read_private_json(path)[0]);reference=d['publication']
        j.require(reference['path']==str(store.root/filename(attempt)))
        j.require(any(dict(row,path=reference['path'])==reference for row in refs),
                  'bound publication record is missing; never reconstruct its authority')


def step(display,store,lease,adapter,attempt):
    old.lease_check(store,lease);value=read(store,attempt)
    if value is None:
        intent=successor.read(store,attempt);j.require(intent is not None,'successor ownership must precede authority publication')
        check_missing_record(store,intent)
        result=successor.step(display,store,lease,adapter,intent['recovery'],attempt);j.require(result['phase']=='claimed')
        j.require(not os.path.lexists(intent['context']['directory']['path']+'/'+old.ANCHOR))
        value=allocate(store,intent);return {'phase':'prepared','root':value['descriptor']['root']['path']}
    d=value['descriptor'];j.require(d['coordinator']==old.lease_check(store,lease) and str(display)==d['context']['socket']['path'])
    if os.path.lexists(d['root']['path']):
        for kind in KINDS:published(value,kind)
        opened=Authority(d['root']['path'])
        try:j.require(opened.descriptor==d)
        finally:opened.close()
        provisioning.sync(d['root']['path']);provisioning.sync(Path(d['root']['path']).parent)
        return {'phase':'complete','root':d['root']['path']}
    old.check_context(d['context']);j.require(old.context(display)==d['context'])
    intent,events,state=linked(d);j.require(len(events)==intent['journal']['events'] and state['pending'] is None,
                                         'unfinished successor publication cannot follow new journal activity')
    observed=adapter.inspect_successor(copy.deepcopy({key:intent[key] for key in ('specification','loaded','members')}))
    j.require(observed==intent['members'],'predecessor cohort changed during authority publication')
    old.check_context(d['context']);j.require(linked(d)[1]==events)
    provisioning.sync(store.root/filename(attempt));provisioning.sync(store.root)
    for kind in KINDS:
        row=value['rows'][kind];target=Path(row['target']['path']);source=Path(row['staging']['path'])
        if os.path.lexists(target):published(value,kind);continue
        check_staged(value,kind);fd=artifacts.directory(target.parent)
        try:provisioning.rename_no_replace(fd,source.name,target.name)
        finally:os.close(fd)
        published(value,kind);provisioning.sync(target);provisioning.sync(target.parent)
        return {'phase':kind+'-published','root':d['root']['path']}
    raise j.Conflict('incomplete successor publication; preserve evidence')


class Authority(old.Authority):
    def __init__(self,root):
        self.root=j.absolute(str(root));j.require(self.root.name==old.ANCHOR);self.fd=artifacts.directory(self.root)
        try:
            self.descriptor=descriptor(old.read(self.fd,'authority.json'));self._verify();self._read_reservation()
        except BaseException:self.close();raise

    @classmethod
    def provision(cls,*args,**kwargs):
        raise j.Conflict('linked successor publication requires its explicit intent and step contract')

    def _verify(self):
        j.require(self.fd is not None,'authority is closed');d=descriptor(self.descriptor)
        for key,kind in (('root','directory'),('coordinator','directory'),('lock','file'),('coordinator_lock','file'),('publication','file')):old.check_object(d[key],kind)
        held=os.fstat(self.fd);j.require((held.st_dev,held.st_ino)==(d['root']['device'],d['root']['inode']))
        old.check_context(d['context']);j.require(self.root==Path(d['root']['path']))
        j.require(old.read(self.fd,'authority.json')==d and old.read(self.fd,'ready.json')=={'format':FORMAT,'descriptor':j.digest(d)})
        record=shape(old.read_private_json(d['publication']['path'])[0]);j.require(record['descriptor']==d)
        for kind in KINDS:published(record,kind)
        linked(d)

    def _gate(self,reservation):
        from successor_admission import Gate
        old.check_object(reservation['gate'],'directory');gate=Gate(reservation['gate']['path'])
        try:
            row=gate.status()['record'];j.require(row['permit']==reservation['permit'] and row['authority']==self._ticket(reservation));return row
        finally:gate.close()

    def reserve(self,store,lease,gate,permit,*,guardian_receipt,adapter):
        # The caller keeps the adapter's plugin admission locks through this
        # operation; immutable-generation preflight is a separate required gate.
        self._canonical(store,lease);permit=admission.permit(permit)
        candidate={'permit':permit,'gate':old.object_identity(gate,'directory'),'intent':'initial','owner':old.owner_for_shell(permit['shell'],guardian_receipt)}
        with self.transaction():
            value=self._read_reservation()
            if value['reservation'] is not None:
                j.require(value['reservation']==candidate,'successor is already reserved');os.fsync(self.fd);return self._ticket(candidate)
            intent,events,state=linked(self.descriptor);request=state['requests'].get(state['pending'])
            j.require(request is not None and request['phase']=='enable-candidate' and request['direction']=='forward' and not request['uncertain'], 'successor needs a canonical replacement ready for construction')
            j.require(request['previous']==intent['loaded'] and request['id']==permit['request'] and j.effect(request,'enable-candidate')==permit['epoch'])
            j.require(request['candidate']==j.digest(permit['specification']) and state['collections'][request['candidate']]==permit['specification'])
            # This first cross-session contract resumes the same logical profile.
            # Compatible immutable generations may differ; profile paths/identity
            # and endpoints must not silently migrate with desktop recovery.
            j.require(permit['specification']['name']==intent['specification']['name'])
            for role in j.ROLES:
                before=intent['specification']['members'][role];after=permit['specification']['members'][role]
                j.require(all(after[k]==v for k,v in before.items() if k!='artifact_sha256'),'successor changed profile or endpoint ownership')
            observed=adapter.inspect_successor(copy.deepcopy({k:intent[k] for k in ('specification','loaded','members')}))
            j.require(observed==intent['members'],'predecessor cohort changed before successor reservation')
            fd=artifacts.directory(gate)
            try:j.require(not os.listdir(fd),'successor admission root must be fresh and empty')
            finally:os.close(fd)
            old.owner_live(candidate['owner'],permit['shell']);self._verify();j.require(store._load()[0]==events,'canonical transition changed during reservation')
            self._save_reservation({'format':2,'authority':self.descriptor['id'],'reservation':candidate,'binding':None,'history':[]})
            return self._ticket(candidate)


def open_authority(root):
    value=old.read_private_json(Path(root)/'authority.json')[0];version=value.get('format')
    j.require(type(version) is int and version in AUTHORITY_READ_FORMATS,'unknown authority; preserve it')
    return old.Authority(root) if version==1 else Authority(root)
