"""Two live private chains with duplicate ship names, inside one fresh offline VM."""
import json
import os
from pathlib import Path
import subprocess

import docker_case
from docker_runtime import DOCKER, identity, require_guest
from state_case import Recovery

ROOT = Path('/opt/omabit-lab')
SHIPS = ('zod', 'nec')
LABELS = ('a', 'b')


def save(report):
    path = ROOT/'results/private-chain-isolation.json'
    temporary = path.with_suffix('.tmp')
    with temporary.open('w') as stream:
        json.dump(report, stream, indent=2)
        stream.write('\n'); stream.flush(); os.fsync(stream.fileno())
    temporary.replace(path)
    fd = os.open(path.parent, os.O_DIRECTORY)
    try: os.fsync(fd)
    finally: os.close(fd)


def context(engine, specs, transport, recovery, peers, persist):
    if not recovery.report.get('ok') or len(recovery.entries) != 4 or recovery.channel is not None:
        raise RuntimeError('complete four-write peer recovery required')
    # Preserve the already-qualified recovery transcript. This extension has its
    # own channel, writes and receipts, and never repeats the first four writes.
    extension = Recovery(transport.call, transport.events, persist, lambda *a, **k: None)
    extension.entries = recovery.entries.copy()
    return dict(engine=engine, specs=specs, transport=transport, peers=peers, recovery=extension)


def snapshot(contexts):
    require_guest()
    output = {}
    for label, ctx in contexts.items():
        engine, specs, peers = ctx['engine'], ctx['specs'], ctx['peers']
        spec = specs['zod']; engine.check_anchor(spec)
        anchor = engine.inspect(spec.anchor)
        if anchor['Name'] != '/omabit-private-'+spec.project+'-gateway':
            raise RuntimeError('gateway name changed')
        namespace = engine.execute(['exec', spec.anchor, '/usr/bin/readlink', '/proc/self/ns/net']).decode().strip()
        if namespace != peers['network_namespace'] or namespace == os.readlink('/proc/self/ns/net'):
            raise RuntimeError('current project namespace differs')
        containers = {}
        for ship in SHIPS:
            item = specs[ship]; container = identity(peers['ships'][ship]['container'])
            row = item.check(engine.inspect(container), container)
            if not row['State']['Running'] or row['State']['OOMKilled']:
                raise RuntimeError('comparison requires all project peers live')
            if engine.execute(['exec',container,'/usr/bin/readlink','/proc/self/ns/net']).decode().strip() != namespace:
                raise RuntimeError('live peer membership differs')
            containers[ship] = row
        objects = {}
        for name, path in [('root',spec.root),('chain',spec.root/'chain'),('zod',specs['zod'].pier),('nec',specs['nec'].pier)]:
            if path.resolve(strict=True) != path or not path.is_dir():
                raise RuntimeError('private project directory differs')
            info = path.stat(); objects[name] = [info.st_dev,info.st_ino]
        output[label] = dict(project=spec.project, anchor=anchor, namespace=namespace, ships=containers, objects=objects)
    if len(contexts) == 2:
        a,b = output['a'],output['b']
        if a['project'] == b['project'] or a['namespace'] == b['namespace'] or a['anchor']['Id'] == b['anchor']['Id']:
            raise RuntimeError('concurrent projects share ownership')
        objects = [tuple(value) for ctx in output.values() for value in ctx['objects'].values()]
        containers = [row['Id'] for ctx in output.values() for row in [ctx['anchor'],*ctx['ships'].values()]]
        if len(set(objects)) != 8 or len(set(containers)) != 6:
            raise RuntimeError('concurrent objects overlap')
    return output


def reject_foreign_session(source, target, ship):
    require_guest()
    if ship not in SHIPS: raise ValueError('fixture ship required')
    left,right = source['specs'][ship],target['specs'][ship]
    if left.project == right.project or left.anchor == right.anchor:
        raise ValueError('distinct private projects required')
    source['engine'].check_anchor(left); target['engine'].check_anchor(right)
    cookie = source['engine'].execute(['exec',left.anchor,'/bin/cat','/tmp/omabit-'+ship+'-cookie'])
    try:
        if not cookie or len(cookie)>8192: raise RuntimeError('private cookie bound')
        # The copied cookie is stdin of this one read-only request. Neither the
        # target's saved session nor a shared host credential store is changed.
        command = DOCKER+['exec','-i',right.anchor,'/usr/bin/curl','--disable','--noproxy','*','--proxy','','--silent','--show-error',
            '--max-time','15','--max-filesize','65536','--cookie','/dev/stdin','--write-out','\n%{http_code}',
            'http://127.0.0.1:'+str(18081+right.number)+'/~/scry/hood/kiln/pikes.json']
        reply = subprocess.run(command,input=cookie,capture_output=True,env=target['engine'].environment,timeout=30)
    finally: del cookie
    if reply.returncode or len(reply.stdout)>65536 or len(reply.stderr)>8192:
        raise RuntimeError('cross-project session probe failed')
    _, separator, status = reply.stdout.rpartition(b'\n')
    if not separator or status not in (b'401',b'403'):
        raise RuntimeError('same-identity session crossed private project')
    return int(status)


def verify(contexts, report, persist):
    require_guest()
    report['before'] = snapshot(contexts); persist()
    public_keys = []
    for label,ctx in contexts.items():
        ready = json.loads((ctx['specs']['zod'].root/'gateway-ready.json').read_text())
        if ready['chainId'] != '0x539' or ready['block'] != '0x50' or ready['externalRpcMutationsAllowed'] is not False:
            raise RuntimeError('private chain identity changed')
        report['ready'][label] = ready
        public_keys.extend(key[field] for key in ready['publicKeys'] for field in ('auth','crypt'))
    if len(public_keys) != 8 or len(set(public_keys)) != 8:
        raise RuntimeError('projects reused public network keys')
    if report['ready']['a']['genesis'] != report['ready']['b']['genesis']:
        raise RuntimeError('shared unkeyed foundation differs')
    persist()
    for src,dst in (('a','b'),('b','a')):
        for ship in SHIPS:
            attempt = dict(source=src,target=dst,ship=ship,dispatched=True)
            report['cross_sessions'].append(attempt); persist()
            attempt['status'] = reject_foreign_session(contexts[src],contexts[dst],ship); persist()
    def observe(label, phase):
        ctx = contexts[label]
        status,body = ctx['transport'].call('nec','/~/scry/omabit-state/state.json')
        expected = ctx['recovery'].expected()
        if status != 200 or json.loads(body) != expected:
            raise RuntimeError('private project state differs')
        report['states'].append(dict(project=label,phase=phase,state=expected));persist()
        return expected
    a,b = observe('a','initial'),observe('b','initial')
    if set(a['entries']) & set(b['entries']):raise RuntimeError('project write identities overlap')
    try:
        for label in LABELS:contexts[label]['recovery'].subscribe()
        for label,other in (('a','b'),('b','a')):
            recovery = contexts[label]['recovery']
            recovery.write('project-'+label+'-isolation'); recovery.receive()
            observe(label,'after-'+label);observe(other,'after-'+label)
        report['after'] = snapshot(contexts);persist()
    finally:
        for label in reversed(LABELS):
            recovery=contexts[label]['recovery']
            if recovery.channel:
                try:recovery.close()
                except Exception as error:report['cleanup_errors'].append('channel:'+label+':'+type(error).__name__)
        persist()
    if report['cleanup_errors']:raise RuntimeError('isolation subscription cleanup incomplete')
    report['verified'] = True; persist()


def exercise(environment):
    require_guest()
    report = dict(ok=False,verified=False,ready={},cross_sessions=[],states=[],extensions={},cleanup_errors=[])
    persist = lambda:save(report)
    contexts = {}
    def second(engine,specs,transport,recovery,peers):
        contexts['b'] = context(engine,specs,transport,recovery,peers,persist)
        report['extensions']['b'] = contexts['b']['recovery'].report
        verify(contexts,report,persist)
    def first(engine,specs,transport,recovery,peers):
        contexts['a'] = context(engine,specs,transport,recovery,peers,persist)
        report['extensions']['a'] = contexts['a']['recovery'].report
        report['second'] = docker_case.exercise(environment,peer_checkpoint=second,label='b')['ok'];persist()
        if not report['second']:raise RuntimeError('second private project did not complete')
        report['after_second_stopped'] = snapshot({'a':contexts['a']})
        for ship in SHIPS:
            row=contexts['b']['engine'].inspect(contexts['b']['peers']['ships'][ship]['container'])
            if row['State']['Running'] or row['State']['OOMKilled'] or row['State']['ExitCode']!=0:
                raise RuntimeError('second project member not cleanly stopped')
        if contexts['b']['engine'].inspect(contexts['b']['specs']['zod'].anchor)['State']['Running']:
            raise RuntimeError('second gateway not stopped')
        state=contexts['a']['recovery'].expected()
        status,body=transport.call('nec','/~/scry/omabit-state/state.json')
        if status!=200 or json.loads(body)!=state:raise RuntimeError('first project affected by second shutdown')
        report['states'].append(dict(project='a',phase='after-b-stopped',state=state));persist()
    try:
        report['first'] = docker_case.exercise(environment,peer_checkpoint=first,label='a')['ok']
        report['ok'] = report['first'] and report.get('second',False) and report['verified'] and not report['cleanup_errors']
    except Exception as error:report['error']=type(error).__name__
    finally:persist()
    return report
