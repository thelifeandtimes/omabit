"""Fresh no-NIC VM: direct HTTP against the first initialized gateway.

All cookies and application values stay in this disposable guest. The separate
stopped-disk reader exports only fixed metadata and hashes.
"""
import hashlib
import json
import os
from pathlib import Path

from docker_runtime import require_guest,identity
from docker_publication import PublishedEngine,PublishedShip
from docker_published_client import PublishedTransport,cookie_header
from docker_gateway_peers import private_json
from state_case import Recovery

ROOT=Path('/opt/omabit-lab')


def capture(engine,specs,previous_transport,previous_recovery,initial_result):
    require_guest()
    if not isinstance(engine,PublishedEngine) or set(specs)!={'zod','nec'}:
        raise ValueError('initial published project required')
    publication=engine.publication
    project=publication.project
    anchor=identity(specs['zod'].anchor)
    for ship,spec in specs.items():
        if not isinstance(spec,PublishedShip) or spec.project!=project or spec.anchor!=anchor or spec.ship!=ship:
            raise ValueError('initial publication member selection differs')
        engine.check_anchor(spec)
        spec.check(engine.inspect(initial_result['ships'][ship]['container']),initial_result['ships'][ship]['container'])
    if initial_result.get('project')!=project or previous_recovery.report.get('ok') is not True or previous_recovery.channel is not None or len(previous_recovery.entries)!=4:
        raise ValueError('completed initial application recovery required')
    root=ROOT/'results'/('initial-publication-clients-'+project)
    root.mkdir(mode=0o700)
    report={'version':1,'ok':False,'project':project,'gateway':anchor,'initial_start':True,'original_sessions_reused':False,'cleanup_errors':[],'baseline_entries':previous_recovery.entries.copy(),'cookie_sha256':{}}
    transport=PublishedTransport(engine,specs,publication,root)
    transport.admin=previous_transport.admin.copy()
    report['publication']={'network':publication.network,'ports':publication.ports,'direct_client':True,'requests':transport.requests}
    recovery=None
    def save():private_json(ROOT/'results/initial-publication.json',report)
    def emit(stage,**unused):print('OMABIT_PRIVATE_CHAIN '+json.dumps({'stage':'initial-publication-'+stage}),flush=True)
    try:
        for ship in ('zod','nec'):
            raw=engine.execute(['exec',anchor,'/bin/cat','/tmp/omabit-'+ship+'-cookie'])
            cookie_header(raw,ship)
            with (root/(ship+'.cookie')).open('xb') as stream:
                os.chmod(stream.name,0o600);stream.write(raw);stream.flush();os.fsync(stream.fileno())
            report['cookie_sha256'][ship]=hashlib.sha256(raw).hexdigest();del raw
        fd=os.open(root,os.O_DIRECTORY)
        try:os.fsync(fd)
        finally:os.close(fd)
        save()
        for ship,other in [('zod','nec'),('nec','zod')]:
            status,body=transport.call(ship,'/~/host')
            if status!=200 or body.strip()!=('~'+ship).encode():raise RuntimeError('initial direct ship identity differs')
            status,_=transport.call(ship,'/~/scry/hood/kiln/pikes.json')
            if status!=200:raise RuntimeError('original initial session refused')
            status,_=transport.call(ship,'/~/scry/hood/kiln/pikes.json',auth=other)
            if status not in (401,403):raise RuntimeError('crossed initial session accepted')
        report['original_sessions_reused']=True
        report['publication']['refusals']=transport.check_unpublished();save()
        recovery=Recovery(transport.call,transport.events,save,emit)
        recovery.entries=previous_recovery.entries.copy()
        report['application_state']=recovery.report
        recovery.subscribe()
        recovery.write('initial-published-connected')
        recovery.receive()
        recovery.close()
        recovery.subscribe()
        recovery.write('initial-published-fresh-subscription')
        recovery.receive()
        recovery.verify_state(*transport.call('nec','/~/scry/omabit-state/state.json'))
        recovery.close()
        recovery.report.update(ok=True,writes_replayed=False,initial_publication=True)
        report['ok']=True;save();emit('passed')
    except Exception as error:
        report['error_type']=type(error).__name__;report['error_sha256']=hashlib.sha256(str(error).encode()).hexdigest();save()
        raise
    finally:
        if recovery and recovery.channel:
            try:recovery.close()
            except Exception as error:report['cleanup_errors'].append(type(error).__name__)
        report['ok']=report['ok'] and not report['cleanup_errors'];save()
        if report['cleanup_errors']:raise RuntimeError('initial publication channel cleanup failed')
