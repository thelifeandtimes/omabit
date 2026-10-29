"""Coordinated fresh-fake member replacement after a private anchor restart."""
from pathlib import Path


def endpoint_probe(exit_code, stdout, stderr):
    """Bounded nonsecret evidence; absence of an HTTP response, not curl success.

    Rootless forwarding can return an empty reply/reset after its backend netns
    changes. Require status000 and no body as well as a recognized transport
    failure; an HTTP error or truncated body is never a refusal proof.
    """
    body, separator, status = stdout.rpartition(b'\n')
    valid_status = bool(separator) and len(status) == 3 and status.isdigit()
    facts = {'exit': exit_code, 'status': status.decode('ascii') if valid_status else 'invalid',
             'response_bytes': len(body) if separator else len(stdout),
             'stderr_bytes': len(stderr)}
    facts['unavailable'] = (exit_code in (7, 28, 52, 56) and valid_status and status == b'000'
                            and body == b'' and len(stdout) <= 132 and len(stderr) <= 8192)
    return facts


def recover(context):
    r=context['report'];client=context['published'];state=context['recovery']
    if r['project']!='omabit-network-a' or not client or state is None or state.channel:
        raise RuntimeError('anchor recovery requires the completed published state gate')
    execute,inspect,netns=context['execute'],context['inspect'],context['netns']
    anchor=r['anchor'];old_namespace=netns(anchor)
    if old_namespace!=r['network_namespace']:raise RuntimeError('anchor changed before recovery admission')
    from endpoints_case import published_bindings
    published_bindings(inspect(anchor),client.bindings)
    members={};piers={}
    for ship,item in r['ships'].items():
        current=inspect(item['container'])
        if not current['State']['Running'] or current['State']['OOMKilled'] or netns(item['container'])!=old_namespace:
            raise RuntimeError('anchor recovery requires both original live members')
        members[ship]=item['container'];p=Path(item['pier']);piers[ship]=(p.stat().st_dev,p.stat().st_ino)
    result={'ok':False,'anchor':anchor,'old_namespace':old_namespace,'old_members':members,
            'endpoint_refusals':[],'old_members_retained':[],'new_members':{}}
    r['anchor_recovery']=result;context['save']()
    state.subscribe();state.write('queued-before-anchor-restart')
    result.update(channel=state.channel,cursor_before=state.cursor,state_before=state.expected())
    context['save']()
    execute(['stop','--timeout','15',anchor],timeout=30)
    current=inspect(anchor)['State']
    if current['Running'] or current['OOMKilled'] or current['ExitCode'] not in (0,143):raise RuntimeError('anchor did not stop cleanly')
    result['anchor_stopped']={k:current[k] for k in ('Running','OOMKilled','ExitCode')};context['save']()
    # No HTTP mutation is attempted while publications are absent. The bounded
    # probe is read-only and must produce a bounded failure with no HTTP response.
    for port in client.bindings:
        status=context['refuse_endpoint'](port)
        result['endpoint_refusals'].append(dict(phase='stopped',port=port,**status))
    execute(['start',anchor]);new_namespace=netns(anchor)
    if new_namespace==old_namespace:raise RuntimeError('anchor restart did not produce the expected distinct namespace')
    published_bindings(inspect(anchor),client.bindings)
    result['new_namespace']=new_namespace;context['save']()
    for ship,identity in members.items():
        current=inspect(identity)
        if not current['State']['Running'] or netns(identity)!=old_namespace:raise RuntimeError('old member unexpectedly migrated with anchor')
        result['old_members_retained'].append(ship)
    for port in client.bindings:
        status=context['refuse_endpoint'](port)
        result['endpoint_refusals'].append(dict(phase='restarted-without-members',port=port,**status))
    context['save']()
    # Explicitly stop every old member before replacing any. The parent piers and
    # old containers remain recorded; no boot flags or new logins are used.
    for ship in ('nec','zod'):context['stop'](ship)
    if any(inspect(identity)['State']['Running'] for identity in members.values()):raise RuntimeError('old member remains live')
    for ship in ('zod','nec'):
        item=r['ships'][ship];p=Path(item['pier'])
        replacement=context['replacement'](ship,item['http_port'],p.parent)
        if replacement in members.values():raise RuntimeError('member container was reused')
        item['container']=replacement;result['new_members'][ship]=replacement;context['save']()
        execute(['start',replacement]);context['ready'](ship,180)
        current=inspect(replacement)
        if current['HostConfig']['NetworkMode']!='container:'+anchor or current['HostConfig']['PortBindings'] or netns(replacement)!=new_namespace or (p.stat().st_dev,p.stat().st_ino)!=piers[ship]:raise RuntimeError('replacement membership/pier mismatch')
    published_bindings(inspect(anchor),client.bindings)
    state.verify_state(*context['state_http']('nec','/~/scry/omabit-state/state.json'))
    state.receive()
    result['cursor_recovered']=state.cursor
    if state.cursor<=result['cursor_before']:raise RuntimeError('queued subscription did not advance after anchor recovery')
    state.write('existing-subscription-after-anchor-restart');state.receive()
    state.close()
    result.update(ok=True,same_piers=True,original_sessions=True,existing_subscription=True,
                  final_state=state.expected(),all_old_members_stopped=True,publications_verified=True)
    context['save']()
