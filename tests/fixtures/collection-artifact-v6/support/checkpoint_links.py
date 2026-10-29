"""Read immutable checkpoint2 ancestry without acquiring runtime/canonical locks.

Guard validation may run while the canonical journal is atomically appended.
All immutable references are rechecked; only journal prefix checks use its current
snapshot. This reader never opens an ancestor runtime or grants admission.
"""
from pathlib import Path
import authority_inspection as inspection
import journal as j
import runtime_authority as authority
import session_checkpoint as original
import session_successor as first_successor
import successor_checkpoint as checkpoint


def read(reference, coordinator):
    reader=inspection.Reader();coordinator=j.absolute(str(coordinator))
    def bound(ref,maximum=authority.MAX_BYTES):
        authority.check_object(ref['object'],'file');body=reader.read(ref['object']['path'],maximum=maximum)
        row=reader.files[ref['object']['path']]
        j.require(row['object']==ref['object'] and row['sha256']==ref['sha256'],'checkpoint ancestry reference changed')
        return body
    record=checkpoint.validate(bound(reference,checkpoint.MAX_BYTES))
    canonical=authority.object_identity(coordinator,'directory')
    j.require(record['coordinator']==canonical and reference['object']['path']==str(coordinator/checkpoint.filename(record['id'])))
    authority.check_object(record['coordinator_lock'],'file')
    parent=original.validate(bound(record['ancestor_checkpoint'],original.MAX_BYTES))
    intent=first_successor.shape(checkpoint.parsed(record,record['inspection']['descriptor']['successor']['object']['path']))
    j.require(parent['coordinator']==canonical and parent['coordinator_lock']==record['coordinator_lock'])
    j.require(parent['id']==intent['recovery'] and parent['inspection']['authority']==intent['predecessor'])
    j.require(parent['cleanup']==record['inspection']['lineage']['cleanup'])
    receipt=bound(parent['cleanup']);j.require(receipt==original.parsed(parent,parent['cleanup']['object']['path'],reference=parent['cleanup']))
    j.require(intent['members']==receipt['verified']['members'] and intent['loaded']==receipt['plan']['loaded'] and intent['specification']==receipt['plan']['specification'])
    j.require(intent['journal']=={k:parent['inspection']['journal'][k] for k in ('events','sha256')})
    for item in (parent,record):
        for path,row in item['files'].items():
            if Path(path).parent==coordinator and path!=str(coordinator/'journal.json'):
                bound({k:row[k] for k in ('object','sha256')})
    store=j.Store(coordinator)
    try:events,state=store._load()
    finally:store.close()
    for item in (parent,record):
        prefix=item['inspection']['journal'];j.require(len(events)>=prefix['events'] and j.digest(events[:prefix['events']])==prefix['sha256'],'checkpoint canonical prefix changed')
    reader.verify();authority.check_object(canonical,'directory');authority.check_object(record['coordinator_lock'],'file');return record,events,state
