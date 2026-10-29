"""Reserve repeated desktops using one intent family and legacy-reader fences.

An immutable intent binds one checkpoint, its unchanged canonical history and
closed plugin cohort, and an explicitly selected new live display. Its staged
runtime claim is object-bound before no-replace publication. Neither the intent
nor that claim is a descriptor, ticket, application grant or reconstructed owner.
"""
import copy
import hashlib
import os
from pathlib import Path
import uuid
import artifacts
import authority_inspection as inspection
import authority_provisioning as provisioning
import journal as j
import runtime_authority as authority
import repeated_checkpoint as checkpoint
import checkpoint_lineage
import session_checkpoint as checkpoint1
import successor_checkpoint as checkpoint2
import session_successor as legacy1
import session_continuation as legacy2

FORMAT = 3
PREFIX = 'desktop-repeat-intent-'
MAX_RECORDS = 128


def filename(attempt):
    j.token(attempt);return PREFIX + attempt + '.json'


def reference(path):
    reader = inspection.Reader();reader.read(path, maximum=checkpoint.MAX_BYTES);reader.verify()
    return {key:copy.deepcopy(reader.files[str(path)][key]) for key in ('object','sha256')}


def check_reference(value):
    authority.check_object(value['object'], 'file')
    j.require(reference(value['object']['path']) == value, 'bound successor evidence changed')


def reference_shape(value):
    j.exact(value, ('object', 'sha256'));inspection.reference_shape(value['object'], 'file');j.token(value['sha256'], j.HASH)


def shape(value):
    j.exact(value, ('format','id','authority','coordinator','coordinator_lock','checkpoint','recovery','predecessor',
                    'journal','context','specification','loaded','members','claim','fences'))
    j.require(type(value['format']) is int and value['format'] == FORMAT, 'unknown desktop successor intent; preserve it')
    for name in ('id','authority','recovery','predecessor'):j.token(value[name])
    j.require(value['authority'] != value['predecessor'])
    inspection.reference_shape(value['coordinator'], 'directory');inspection.reference_shape(value['coordinator_lock'], 'file')
    reference_shape(value['checkpoint'])
    j.require(value['checkpoint']['object']['path'] in {
        str(Path(value['coordinator']['path'])/module.filename(value['recovery']))
        for module in (checkpoint1, checkpoint2, checkpoint)})
    j.exact(value['journal'], ('events','sha256'));j.token(value['journal']['sha256'], j.HASH)
    j.require(type(value['journal']['events']) is int and 0 < value['journal']['events'] <= j.MAX_EVENTS)
    context = value['context'];j.exact(context, ('directory','socket','server'))
    inspection.reference_shape(context['directory'], 'directory');inspection.reference_shape(context['socket'], 'socket');authority.admission.process(context['server'])
    runtime = Path(context['directory']['path']);j.require(Path(context['socket']['path']).parent == runtime)
    j.require(not Path(value['coordinator']['path']).is_relative_to(runtime), 'successor coordinator must survive runtime loss')
    j.specification(value['specification']);j.loaded(value['loaded'], value['specification'], value['loaded']['epoch'])
    j.exact(value['members'], j.ROLES)
    for role in j.ROLES:j.recovery_member(value['members'][role], value['loaded']['members'][role], closed=True)
    claim = value['claim'];j.exact(claim, ('staging','target','sha256'))
    for name in ('staging','target'):inspection.reference_shape(claim[name], 'file')
    j.token(claim['sha256'], j.HASH)
    stage = Path(claim['staging']['path']);j.require(stage.parent == runtime and stage.name.startswith('.pending-desktop-repeat-intent-'+value['id']+'-'))
    j.token(stage.name.removeprefix('.pending-desktop-repeat-intent-'+value['id']+'-'))
    j.require(claim['target'] == dict(claim['staging'], path=str(runtime/authority.PROVISION_CLAIM)))
    j.require(claim['sha256'] == hashlib.sha256(j.canonical(claim_body(value))+b'\n').hexdigest())
    j.exact(value['fences'], ('successor', 'continuation'))
    for role, module in (('successor', legacy1), ('continuation', legacy2)):
        row=value['fences'][role];j.exact(row, ('staging','target','sha256'))
        for key in ('staging','target'):inspection.reference_shape(row[key], 'file')
        stage=Path(row['staging']['path']);prefix='.pending-repeat-fence-'+value['id']+'-'+role+'-'
        j.require(stage.parent==Path(value['coordinator']['path']) and stage.name.startswith(prefix))
        j.token(stage.name.removeprefix(prefix))
        j.require(row['target']==dict(row['staging'],path=str(stage.parent/module.filename(value['id']))))
        j.require(row['sha256']==hashlib.sha256(j.canonical(fence_body(value,role))+b'\n').hexdigest())
    return value


def fence_body(value, role):
    return {'format': FORMAT, 'kind': 'repeated-session-ownership-fence', 'legacy': role,
            'attempt': value['id'], 'authority': value['authority'], 'predecessor': value['predecessor'],
            'coordinator': value['coordinator'], 'checkpoint': value['checkpoint'],
            'record': str(Path(value['coordinator']['path'])/filename(value['id']))}


def check_ownership(store, value):
    names=os.listdir(store.fd)
    j.require(len(names)<=checkpoint_lineage.MAX_FILES, 'coordinator evidence inventory is full; preserve it')
    for module, role in ((legacy1,'successor'), (legacy2,'continuation')):
        for name in names:
            if not name.startswith(module.PREFIX) or not name.endswith('.json'):continue
            attempt=name[len(module.PREFIX):-5]
            if not j.ID.fullmatch(attempt):continue  # Other historical record families share these prefixes.
            body=authority.read(store.fd,name)
            if body.get('kind')=='repeated-session-ownership-fence':
                owner=read(store,attempt)
                j.require(owner is not None, 'bound repeated intent is missing; preserve its ownership fence')
                j.require(body==fence_body(owner,role), 'legacy ownership fence changed')
                row=owner['fences'][role]
                check_reference({'object':row['target'],'sha256':row['sha256']})
                other=owner
            else:
                other=module.shape(body)
            j.require(other['predecessor']!=value['predecessor'] or
                      (other['format']==FORMAT and other==value), 'another intent owns this predecessor')
    own=[name for name in names if name.startswith(PREFIX)]
    j.require(len(own)<=MAX_RECORDS, 'repeated intent history is full; preserve it')
    for name in own:
        j.require(name.endswith('.json'), 'unknown repeated intent evidence')
        other=read(store,name[len(PREFIX):-5]);j.require(other is not None)
        j.require(other['predecessor']!=value['predecessor'] or other==value,
                  'another intent owns this predecessor')


def claim_body(value):
    # This deliberately has no authority-descriptor schema. Existing fresh-setup
    # paths refuse any occupied provisioning claim instead of interpreting it.
    return {'format': FORMAT, 'kind': 'desktop-repeat-intent', 'attempt': value['id'],
            'authority': value['authority'], 'coordinator': value['coordinator'],
            'record': str(Path(value['coordinator']['path'])/filename(value['id'])),
            'checkpoint': value['checkpoint'], 'context': value['context']}


def read(store, attempt):
    try:value = authority.read(store.fd, filename(attempt))
    except FileNotFoundError:return None
    shape(value);j.require(value['id'] == attempt);return value


def predecessor(store, lease, checkpoint_reference):
    authority.lease_check(store,lease)
    record = checkpoint_lineage.read(checkpoint_reference,store.root)['head']
    snapshot = record['inspection'];events, state = store._load()
    prefix = {key:snapshot['journal'][key] for key in ('events','sha256')}
    j.require(len(events) == prefix['events'] and j.digest(events) == prefix['sha256'] and state['pending'] is None,
              'successor preparation requires the unchanged checkpoint journal')
    j.require(state['committed'] == snapshot['reservation']['binding']['loaded'])
    mirror = record['files'][snapshot['descriptor'].get('mirror',str(store.root/authority.MIRROR))];check_reference({k:mirror[k] for k in ('object','sha256')})
    receipt = checkpoint.parsed(record, record['cleanup']['object']['path'], reference=record['cleanup'])
    return record, receipt, prefix


def _verify_context(display, store, lease, adapter, value):
    canonical = authority.lease_check(store, lease)
    j.require(canonical == value['coordinator']);authority.check_object(value['coordinator_lock'], 'file')
    check_reference(value['checkpoint']);record, receipt, prefix = predecessor(store, lease, value['checkpoint'])
    j.require(record['inspection']['authority'] == value['predecessor'] and prefix == value['journal'])
    j.require(receipt['plan']['specification'] == value['specification'] and receipt['plan']['loaded'] == value['loaded'] and
              receipt['verified']['members'] == value['members'], 'successor no longer matches original closed cohort')
    check_ownership(store,value)
    current = authority.context(display);j.require(current == value['context'], 'successor display changed; preserve pending ownership')
    old = record['inspection']['descriptor']['context']
    j.require(current['server'] != old['server'] and current['directory'] != old['directory'], 'successor requires a new live runtime and display')
    runtime = Path(current['directory']['path'])
    j.require(not any(os.path.lexists(runtime/name) for name in (authority.ANCHOR,authority.ANCHOR_SEAL)), 'successor runtime already contains authority evidence')
    observed = adapter.inspect_successor(copy.deepcopy({'specification':value['specification'],'loaded':value['loaded'],'members':value['members']}))
    j.exact(observed, j.ROLES)
    for role in j.ROLES:j.recovery_member(observed[role], value['loaded']['members'][role], closed=True)
    j.require(observed == value['members'], 'current plugin state, credentials, pending intent or transport differs from closed checkpoint')
    # Recheck external observations before publishing any ownership evidence.
    check_reference(value['checkpoint']);predecessor(store, lease, value['checkpoint']);authority.check_context(current)
    return value


def verify(display, store, lease, adapter, value):
    shape(value);return _verify_context(display, store, lease, adapter, value)


def save(store, value):
    shape(value)
    name = '.pending-desktop-repeat-intent-record-'+value['id']+'-'+uuid.uuid4().hex
    authority.write_new(store.fd, name, value)
    provisioning.rename_no_replace(store.fd, name, filename(value['id']))


def step(display, store, lease, adapter, checkpoint_reference, attempt):
    """Prepare or claim the same explicit successor, without starting anything."""
    canonical = authority.lease_check(store, lease);filename(attempt);reference_shape(checkpoint_reference)
    value = read(store, attempt)
    if value is None:
        record, receipt, prefix = predecessor(store, lease, checkpoint_reference)
        recovery=record['id']
        names = [name for name in os.listdir(store.fd) if name.startswith(PREFIX)]
        j.require(len(names) < MAX_RECORDS, 'successor history is full; preserve it')
        for name in names:
            j.require(name.endswith('.json'), 'unknown successor evidence')
            other = read(store, name[len(PREFIX):-5]);j.require(other is not None)
            j.require(other['predecessor'] != record['inspection']['authority'], 'another successor intent owns this predecessor')
        current = authority.context(display);runtime = Path(current['directory']['path'])
        j.require(not os.path.lexists(runtime/authority.PROVISION_CLAIM), 'new runtime already has an ownership claim')
        value = {'format':FORMAT,'id':attempt,'authority':uuid.uuid4().hex,'coordinator':canonical,
                 'coordinator_lock':authority.object_identity(store.root/'coordinator.lock','file'),
                 'checkpoint':copy.deepcopy(checkpoint_reference),'recovery':recovery,'predecessor':record['inspection']['authority'],
                 'journal':prefix,'context':current,'specification':receipt['plan']['specification'],
                 'loaded':receipt['plan']['loaded'],'members':receipt['verified']['members']}
        _verify_context(display,store,lease,adapter,value)
        # The runtime claim's inode must be known before publishing the immutable
        # intent. A crash before that intent leaves an unbound, retained stage.
        name = '.pending-desktop-repeat-intent-'+attempt+'-'+uuid.uuid4().hex
        fd = artifacts.directory(runtime)
        try:
            authority.write_new(fd, name, claim_body(value))
            staging = authority.object_identity(runtime/name, 'file')
            value['claim'] = {'staging':staging,'target':dict(staging,path=str(runtime/authority.PROVISION_CLAIM)),
                              'sha256':reference(runtime/name)['sha256']}
            value['fences']={}
            for role,module in (('successor',legacy1),('continuation',legacy2)):
                name='.pending-repeat-fence-'+attempt+'-'+role+'-'+uuid.uuid4().hex
                authority.write_new(store.fd,name,fence_body(value,role))
                staging=authority.object_identity(store.root/name,'file')
                value['fences'][role]={'staging':staging,'target':dict(staging,path=str(store.root/module.filename(attempt))),
                                      'sha256':reference(store.root/name)['sha256']}
            verify(display,store,lease,adapter,value);save(store,value)
        finally:os.close(fd)
        return {'phase':'prepared','intent':value,'activation_authorized':False}
    j.require(value['checkpoint'] == checkpoint_reference, 'successor retry names another checkpoint')
    verify(display,store,lease,adapter,value)
    provisioning.sync(store.root/filename(attempt));provisioning.sync(store.root)
    # Legacy writers scan their own intent prefixes and refuse the new fence
    # shape. Publish both fences before claiming a runtime. A competing legacy
    # intent after an interrupted preparation is detected by verify, never ignored.
    for role in ('successor','continuation'):
        row=value['fences'][role];target=Path(row['target']['path']);stage=Path(row['staging']['path'])
        if os.path.lexists(target):
            check_reference({'object':row['target'],'sha256':row['sha256']})
            j.require(not os.path.lexists(stage), 'both staged and published ownership fences exist')
            continue
        check_reference({'object':row['staging'],'sha256':row['sha256']})
        provisioning.rename_no_replace(store.fd,stage.name,target.name)
        check_reference({'object':row['target'],'sha256':row['sha256']})
        provisioning.sync(target);provisioning.sync(store.root)
        return {'phase':role+'-fenced','intent':value,'activation_authorized':False}
    claim = value['claim'];target = Path(claim['target']['path']);stage = Path(claim['staging']['path'])
    if os.path.lexists(target):
        check_reference({'object':claim['target'],'sha256':claim['sha256']})
        j.require(not os.path.lexists(stage), 'successor stage remains after publication')
    else:
        check_reference({'object':claim['staging'],'sha256':claim['sha256']})
        fd = artifacts.directory(target.parent)
        try:provisioning.rename_no_replace(fd,stage.name,target.name)
        finally:os.close(fd)
        check_reference({'object':claim['target'],'sha256':claim['sha256']})
    provisioning.sync(target);provisioning.sync(target.parent)
    return {'phase':'claimed','intent':value,'activation_authorized':False}


def check_claimed(value):
    """Verify original published fences and claim, without granting admission."""
    shape(value)
    for row in list(value['fences'].values()) + [value['claim']]:
        check_reference({'object':row['target'],'sha256':row['sha256']})
        j.require(not os.path.lexists(row['staging']['path']), 'published ownership stage remains')
    return value
