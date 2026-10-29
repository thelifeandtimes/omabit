"""Immutable linked-runtime checkpoint with a referenced predecessor checkpoint.

Capture requires verified original-desktop cleanup while original evidence still
exists. Reading later verifies copied bytes and the original canonical journal;
it neither reconstructs a runtime authority nor authorizes successor activation.
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
import successor_cleanup as session_cleanup
import successor_inspection
import successor_authority as linked
import session_successor as successor
import session_checkpoint as original_checkpoint

FORMAT = 2
PREFIX = 'desktop-successor-checkpoint-'
MAX_BYTES = 32 << 20
MAX_SOURCE_BYTES = 16 << 20
MAX_FILES = 1024


def filename(recovery):
    j.token(recovery);return PREFIX + recovery + '.json'


def parsed(record, path, *, reference=None):
    path = str(path);j.require(path in record['files'], 'checkpoint lacks required original evidence')
    row = record['files'][path]
    if reference is not None:j.require(row['object'] == reference['object'] and row['sha256'] == reference['sha256'])
    return artifacts.object_json(row['text'].encode('utf-8'))


def validate(record):
    j.exact(record, ('format', 'id', 'coordinator', 'coordinator_lock', 'cleanup', 'inspection', 'files', 'ancestor_checkpoint', 'sha256'))
    j.require(type(record['format']) is int and record['format'] == FORMAT, 'unknown desktop checkpoint; preserve it')
    j.token(record['id']);j.token(record['sha256'], j.HASH)
    payload = {key:value for key,value in record.items() if key != 'sha256'}
    j.require(j.digest(payload) == record['sha256'], 'desktop checkpoint content changed')
    inspection.reference_shape(record['coordinator'], 'directory');inspection.reference_shape(record['coordinator_lock'], 'file')
    j.exact(record['cleanup'], ('object', 'sha256'));inspection.reference_shape(record['cleanup']['object'], 'file');j.token(record['cleanup']['sha256'], j.HASH)
    snap = record['inspection'];j.exact(snap, ('format','mode','authority','context','descriptor','reservation','journal','activations','files','missing_files','activation_authorized','transport_cleanup_verified','lineage'))
    j.require(type(snap['format']) is int and snap['format'] == 2 and snap['mode'] == 'retained-evidence-only')
    j.require(snap['activation_authorized'] is False and snap['transport_cleanup_verified'] is False)
    j.exact(snap['context'], ('server','filesystem','live_context'));j.require(snap['context']['server'] == 'gone' and snap['context']['live_context'] is False)
    j.require(type(record['files']) is dict and 0 < len(record['files']) <= MAX_FILES)
    successor.reference_shape(record['ancestor_checkpoint']);ancestor = record['ancestor_checkpoint'];ancestor_path = ancestor['object']['path']
    j.require(snap['lineage']['checkpoint'] == ancestor and snap['files'].get(ancestor_path) == ancestor)
    j.require(set(record['files']) == (set(snap['files']) | {record['cleanup']['object']['path']}) - {ancestor_path}, 'checkpoint inventory changed')
    size = 0
    for path, row in record['files'].items():
        j.absolute(path);j.exact(row, ('object','sha256','text'));inspection.reference_shape(row['object'], 'file')
        j.require(row['object']['path'] == path and type(row['text']) is str);j.token(row['sha256'], j.HASH)
        raw = row['text'].encode('utf-8');size += len(raw)
        j.require(hashlib.sha256(raw).hexdigest() == row['sha256'], 'copied evidence bytes changed')
        if path in snap['files']:j.require(snap['files'][path] == {k:row[k] for k in ('object','sha256')})
    j.require(size <= MAX_SOURCE_BYTES, 'checkpoint source inventory is too large')
    receipt = parsed(record, record['cleanup']['object']['path'], reference=record['cleanup']);session_cleanup.record_shape(receipt)
    j.require(receipt['id'] == record['id'] and receipt['verified'] is not None, 'checkpoint has no verified original cleanup')
    plan = receipt['plan'];d = snap['descriptor'];root = Path(d['root']['path']);coordinator = Path(record['coordinator']['path'])
    linked.descriptor(d);j.require(root.name == authority.ANCHOR)
    j.require(d['coordinator'] == record['coordinator'] and d['coordinator_lock'] == record['coordinator_lock'])
    j.require(record['cleanup']['object']['path'] == str(coordinator/session_cleanup.filename(record['id'])))
    inspection.reference_shape(d['root'], 'directory');inspection.reference_shape(d['lock'], 'file')
    j.require(root.parent == Path(d['context']['directory']['path']) and not coordinator.is_relative_to(root.parent), 'checkpoint coordinator is inside disposable runtime')
    j.require(parsed(record,root/'authority.json') == d and parsed(record,root.parent/authority.ANCHOR_SEAL) == d and
              parsed(record,d['mirror']) == d)
    j.require(parsed(record,root/'ready.json') == {'format':2,'descriptor':j.digest(d)})
    intent = successor.shape(parsed(record,d['successor']['object']['path'],reference=d['successor']))
    j.require(intent['authority'] == d['id'] and intent['context'] == d['context'] and intent['coordinator'] == d['coordinator'] and intent['coordinator_lock'] == d['coordinator_lock'])
    j.require(intent['checkpoint'] == ancestor and Path(ancestor_path).parent == coordinator)
    j.require(snap['lineage'] == {'predecessor':intent['predecessor'],'intent':d['successor'],'checkpoint':ancestor,
                                'cleanup':snap['lineage']['cleanup'],'journal_prefix':intent['journal']})
    successor.reference_shape(snap['lineage']['cleanup'])
    publication = linked.shape(parsed(record,d['publication']['path']))
    j.require(publication['descriptor'] == d and record['files'][d['publication']['path']]['object'] == d['publication'])
    claim = intent['claim'];j.require(parsed(record,claim['target']['path']) == successor.claim_body(intent))
    j.require(record['files'][claim['target']['path']]['object'] == claim['target'] and record['files'][claim['target']['path']]['sha256'] == claim['sha256'])
    value = parsed(record,root/'reservation.json');j.require(value == snap['reservation']);authority.reservation_value(value,d)
    journal = parsed(record,coordinator/'journal.json');j.exact(journal, ('format','events'))
    j.require(type(journal['format']) is int and journal['format'] in j.READ_FORMATS and type(journal['events']) is list and len(journal['events']) <= j.MAX_EVENTS)
    state = j.initial()
    for event in journal['events']:
        j.require(journal['format'] >= 2 or not str(event.get('type','')).startswith('recovery-'))
        j.require(journal['format'] == 3 or not j.format3_event(event));j.require(j.apply(state,event))
    j.require(snap['journal'] == {'format':journal['format'],'events':len(journal['events']),'sha256':j.digest(journal['events']),
                                'committed':state['committed'],'pending':state['pending']})
    authority.history_value(value,journal['events'],d,lambda ref:(parsed(record,ref['object']['path'],reference=ref),ref['sha256']))
    evidence = {key:snap[key] for key in ('descriptor','reservation','journal','activations','files','missing_files','lineage')}
    j.require(plan['authority'] == d['id'] == snap['authority'] and plan['descriptor_sha256'] == j.digest(d) and
              plan['reservation_sha256'] == j.digest(value['reservation']) and plan['evidence_sha256'] == j.digest(evidence))
    j.require(plan['journal'] == {'events':len(journal['events']),'sha256':j.digest(journal['events'])} and state['pending'] is None)
    j.require(value['binding'] is not None and value['binding']['loaded'] == state['committed'] == plan['loaded'])
    j.require(value['reservation']['permit']['specification'] == plan['specification'])
    owner = value['reservation']['owner'];proof = parsed(record,owner['receipt']);authority.check_guardian_receipt(proof,owner['guardian'])
    j.require(plan['guardian'] == owner['guardian'] and proof['phase'] == 'closed' and proof['proof'] == 'waitpid-echild' and
              proof['request'] == plan['closure_request'] and record['files'][owner['receipt']]['sha256'] == plan['closure_receipt_sha256'])
    j.require(any(row.get('pid') == owner['guardian']['root']['pid'] for row in proof['reaped']))
    return record


def read(store, recovery, lease):
    canonical = authority.lease_check(store,lease);path = store.root/filename(recovery)
    reader = inspection.Reader();record = validate(reader.read(path,maximum=MAX_BYTES))
    j.require(record['id'] == recovery and record['coordinator'] == canonical, 'checkpoint belongs to another coordinator')
    authority.check_object(record['coordinator_lock'],'file')
    # Canonical immutable links must survive; only disposable runtime metadata is
    # supplied by copied bytes. Parent checkpoints are referenced, never embedded.
    ancestor = record['ancestor_checkpoint'];authority.check_object(ancestor['object'],'file')
    parent = reader.read(ancestor['object']['path'],maximum=original_checkpoint.MAX_BYTES)
    j.require(reader.files[ancestor['object']['path']]['sha256'] == ancestor['sha256'])
    intent = parsed(record,record['inspection']['descriptor']['successor']['object']['path'])
    j.require(parent == original_checkpoint.read(store,intent['recovery'],lease))
    j.require(parent['inspection']['authority'] == intent['predecessor'] and parent['cleanup'] == record['inspection']['lineage']['cleanup'])
    receipt = original_checkpoint.parsed(parent,parent['cleanup']['object']['path'],reference=parent['cleanup'])
    j.require(receipt['verified']['members'] == intent['members'] and receipt['plan']['specification'] == intent['specification'] and receipt['plan']['loaded'] == intent['loaded'])
    j.require(intent['journal'] == {k:parent['inspection']['journal'][k] for k in ('events','sha256')})
    for name,row in record['files'].items():
        if Path(name).parent == store.root and name != str(store.root/'journal.json'):
            authority.check_object(row['object'],'file');reader.read(name)
            j.require(reader.files[name]['sha256'] == row['sha256'], 'canonical linked checkpoint evidence changed')
    cleanup = record['cleanup'];authority.check_object(cleanup['object'],'file');reader.read(cleanup['object']['path'])
    j.require(reader.files[cleanup['object']['path']]['sha256'] == cleanup['sha256'], 'original verified cleanup record changed')
    events,_ = store._load();prefix = record['inspection']['journal']
    j.require(len(events) >= prefix['events'] and j.digest(events[:prefix['events']]) == prefix['sha256'], 'canonical history no longer retains checkpoint prefix')
    reader.verify();return record


def capture(root, store, lease, recovery):
    authority.lease_check(store,lease);path = store.root/filename(recovery);root = j.absolute(str(root))
    if os.path.lexists(path):
        record = read(store,recovery,lease)
        j.require(record['inspection']['descriptor']['root']['path'] == str(root), 'checkpoint retry names another original root')
        fd = os.open(path,os.O_RDONLY|os.O_NOFOLLOW|os.O_CLOEXEC)
        try:os.fsync(fd)
        finally:os.close(fd)
        os.fsync(store.fd);return record
    receipt = session_cleanup.read_record(store,recovery)
    j.require(receipt is not None and receipt['verified'] is not None, 'verified original cleanup must precede checkpoint capture')
    session_cleanup.validate_context(root,store,lease,receipt['plan'])
    snapshot = successor_inspection.inspect(root,store.root,lease=lease)
    runtime = Path(snapshot['descriptor']['context']['directory']['path'])
    j.require(not store.root.is_relative_to(runtime), 'checkpoint storage must be outside the runtime directory')
    cleanup_path = store.root/session_cleanup.filename(recovery);body,digest = authority.read_private_json(cleanup_path);j.require(body == receipt)
    cleanup = {'object':authority.object_identity(cleanup_path,'file'),'sha256':digest}
    references = dict(snapshot['files']);references[str(cleanup_path)] = cleanup
    ancestor = copy.deepcopy(snapshot['lineage']['checkpoint']);j.require(references.pop(ancestor['object']['path']) == ancestor)
    j.require(len(references) <= MAX_FILES)
    copied = {};total = 0
    for name, reference in references.items():
        authority.check_object(reference['object'],'file');fd = os.open(name,os.O_RDONLY|os.O_NOFOLLOW|os.O_NONBLOCK|os.O_CLOEXEC)
        with os.fdopen(fd,'rb') as stream:raw = stream.read(MAX_SOURCE_BYTES + 1)
        total += len(raw);j.require(total <= MAX_SOURCE_BYTES and hashlib.sha256(raw).hexdigest() == reference['sha256'])
        authority.check_object(reference['object'],'file');copied[name] = dict(reference,text=raw.decode('utf-8'))
    session_cleanup.validate_context(root,store,lease,receipt['plan'])
    value = {'format':FORMAT,'id':recovery,'coordinator':authority.object_identity(store.root,'directory'),
             'coordinator_lock':authority.object_identity(store.root/'coordinator.lock','file'),'cleanup':cleanup,'inspection':snapshot,'files':copied,'ancestor_checkpoint':ancestor}
    value['sha256'] = j.digest(value);validate(value);raw = j.canonical(value)+b'\n';j.require(len(raw) <= MAX_BYTES)
    temporary = '.pending-desktop-successor-checkpoint-' + recovery + '-' + uuid.uuid4().hex
    fd = os.open(temporary,os.O_WRONLY|os.O_CREAT|os.O_EXCL|os.O_NOFOLLOW|os.O_CLOEXEC,0o600,dir_fd=store.fd)
    with os.fdopen(fd,'wb') as stream:stream.write(raw);stream.flush();os.fsync(stream.fileno())
    os.fsync(store.fd)
    provisioning.rename_no_replace(store.fd,temporary,path.name)
    return read(store,recovery,lease)
