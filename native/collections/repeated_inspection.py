"""Read-only inspection through reusable bounded desktop ancestry.

The original reader is unchanged. Linked snapshots validate the immutable
predecessor chain independently of the current compositor's liveness; they grant
neither transport cleanup nor application admission.
"""
import contextlib
import copy
from pathlib import Path

import authority_inspection as original
import continuation_inspection as previous
import artifacts
import journal as j
import runtime_authority as authority

FORMAT = 4


def inspect(root, coordinator, *, lease=None):
    root = j.absolute(str(root));coordinator = j.absolute(str(coordinator))
    j.require(root.name == authority.ANCHOR, 'explicit canonical authority root required')
    reader = original.Reader();d = reader.read(root/'authority.json')
    if type(d.get('format')) is int and d['format'] in (1, 2, 3):
        return previous.inspect(root, coordinator, lease=lease)
    # Keep original-only callers independent of successor module availability.
    import repeated_authority as linked
    import repeated_session as successor
    import repeated_checkpoint as checkpoint
    import checkpoint_lineage
    linked.descriptor(d)
    j.require(d['root']['path'] == str(root) and d['coordinator']['path'] == str(coordinator))
    reader.directory(d['root']);reader.directory(d['coordinator'])
    if lease is not None:
        j.require(str(lease.store.root) == str(coordinator) and
                  authority.lease_check(lease.store, lease) == d['coordinator'], 'inspection lease belongs to another coordinator')
        authority.check_object(d['coordinator_lock'], 'file')
    lock = contextlib.nullcontext() if lease is not None else original.read_lock(d['coordinator_lock'])
    with lock, original.read_lock(d['lock']):
        before_context = original.context_state(d['context'])
        j.require(reader.read(root/'authority.json') == d)
        j.require(reader.read(root/'ready.json') == {'format': 4, 'descriptor': j.digest(d)}, 'linked ready seal changed')
        j.require(reader.read(root.parent/authority.ANCHOR_SEAL) == d and reader.read(d['mirror']) == d,
                  'linked authority mirrors disagree')
        reader.read(root/'authority.lock')
        authority.check_object(d['publication'], 'file')
        publication = linked.shape(reader.read(d['publication']['path']))
        j.require(publication['descriptor'] == d, 'linked publication changed')
        for kind in linked.KINDS:linked.published(publication, kind)
        # Validate the original checkpoint/cleanup and journal prefix using the
        # unchanged linked contract, without constructing any live Authority.
        intent, linked_events, _ = linked.linked(d)

        def bound(reference, maximum=authority.MAX_BYTES):
            authority.check_object(reference['object'], 'file')
            value = reader.read(reference['object']['path'], maximum=maximum)
            row = reader.files[reference['object']['path']]
            j.require(row['object'] == reference['object'] and row['sha256'] == reference['sha256'], 'linked inspection reference changed')
            return value

        j.require(bound(d['successor']) == intent)
        chain = checkpoint_lineage.read(intent['checkpoint'], coordinator)
        archived = chain['head']
        for reference in chain['references']:bound(reference, checkpoint.MAX_BYTES)
        for row in intent['fences'].values():bound({'object':row['target'],'sha256':row['sha256']})
        j.require(bound(archived['cleanup']) == checkpoint.parsed(archived, archived['cleanup']['object']['path']))
        prior_mirror = archived['files'][archived['inspection']['descriptor'].get('mirror',str(coordinator/authority.MIRROR))]
        bound({key:prior_mirror[key] for key in ('object', 'sha256')})
        claim = intent['claim'];authority.check_object(claim['target'], 'file')
        j.require(reader.read(claim['target']['path']) == successor.claim_body(intent))
        j.require(reader.files[claim['target']['path']]['sha256'] == claim['sha256'])
        body = reader.read(coordinator/'journal.json', maximum=j.MAX_BYTES)
        store = j.Store(coordinator)
        try:events, state = store._load()
        finally:store.close()
        j.require(body['events'] == events == linked_events, 'canonical history changed during linked inspection')
        replay = j.initial();recorded_loaded = set()
        for event in events:
            j.require(j.apply(replay, event))
            for loaded in [replay['committed']] + [r[k] for r in replay['requests'].values() for k in ('previous', 'loaded')]:
                if loaded is not None:recorded_loaded.add(j.digest(loaded))
        value = authority.reservation_value(reader.read(root/'reservation.json'), d)
        authority.history_value(value, events, d, lambda ref:(bound(ref), ref['sha256']))
        rows = list(value.get('history', []))
        if value['reservation'] is not None:rows.append({'reservation': value['reservation'], 'binding': value.get('binding')})
        observations = []
        for entry in rows:
            reservation = entry['reservation'];spec = reservation['permit']['specification']
            j.require(state['collections'].get(j.digest(spec)) == spec, 'linked reservation registration changed')
            observed = original.binding(reader, entry['binding'], reservation, recorded_loaded)
            admission = original.gate(reader, d, reservation, entry['binding'])
            closure = original.closure(reader, reservation)
            if 'handoff' in entry:
                proof = entry['handoff'];reference = proof.get('initial_recovery')
                if reference is not None:bound(reference)
                if proof['closure'] is not None:
                    reference = proof['closure'];reader.read(reference['path'])
                    j.require(reader.files[reference['path']]['sha256'] == reference['sha256'], 'historical closure changed')
            observations.append({'reservation_sha256': j.digest(reservation), 'admission': admission, 'closure': closure,
                                 'loaded': entry['binding']['loaded'] if entry['binding'] else None,
                                 'runtime_observation_sha256': entry['binding']['runtime']['sha256'] if observed else None})
        reader.verify()
        j.require(original.context_state(d['context']) == before_context, 'compositor identity changed during linked inspection')
        authority.check_object(d['coordinator_lock'], 'file')
        if lease is not None:j.require(authority.lease_check(lease.store, lease) == d['coordinator'])
        return {'format': FORMAT, 'mode': 'retained-evidence-only', 'authority': d['id'], 'context': before_context,
                'descriptor': copy.deepcopy(d), 'reservation': value,
                'journal': {'format': body['format'], 'events': len(events), 'sha256': j.digest(events),
                            'committed': state['committed'], 'pending': state['pending']}, 'activations': observations,
                'files': {path:{k:row[k] for k in ('object', 'sha256')} for path,row in reader.files.items()},
                'missing_files': sorted(reader.absent), 'activation_authorized': False, 'transport_cleanup_verified': False,
                'lineage': {'predecessor': intent['predecessor'], 'intent': copy.deepcopy(d['successor']),
                            'checkpoint': copy.deepcopy(intent['checkpoint']), 'cleanup': copy.deepcopy(archived['cleanup']),
                            'journal_prefix': copy.deepcopy(intent['journal']),
                            'ancestors': copy.deepcopy(chain['references'][1:])}}
