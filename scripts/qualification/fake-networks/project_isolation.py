"""Cross-project assertions while both task-created fake pairs are live."""
import json


def verify(a, b):
    left, right = a['report'], b['report']
    if left['project'] != 'omabit-network-a' or right['project'] != 'omabit-network-b':
        raise RuntimeError('wrong project pair')
    if left['anchor'] == right['anchor'] or left['network_namespace'] == right['network_namespace']:
        raise RuntimeError('projects share a namespace/anchor')
    published=bool(a.get('published'))
    if published != bool(b.get('published')):raise RuntimeError('mixed project publication modes')
    if published:
        from endpoints_case import published_bindings
        if left['publication']['bridge']==right['publication']['bridge']:raise RuntimeError('projects share a bridge')
        if set(a['published'].bindings.values()) & set(b['published'].bindings.values()):raise RuntimeError('projects share a public endpoint')
        for context in (a,b):published_bindings(context['inspect'](context['report']['anchor']),context['published'].bindings)
    for context in (a,b):
        for ship, item in context['report']['ships'].items():
            current = context['inspect'](item['container'])
            if not current['State']['Running'] or current['State']['OOMKilled']:
                raise RuntimeError('comparison requires both live fake pairs')
            if current['HostConfig']['PortBindings'] or current['HostConfig']['NetworkMode'] != 'container:'+context['report']['anchor']:
                raise RuntimeError('cross-project endpoint/membership mismatch')
    report={'ok':False,'same_identities':['zod','nec'],'namespaces':{'a':left['network_namespace'],'b':right['network_namespace']},'cross_cookie_checks':[],'state_checks':[]}
    left['project_isolation']=report
    for source,target in ((a,b),(b,a)):
        for ship in ('zod','nec'):
            if published:
                status=target['published'].reject_foreign_session(source['published'],ship)
            else:
                cookie=source['execute'](['exec',source['report']['anchor'],'/bin/cat','/tmp/omabit-'+ship+'-cookie'])
                if not cookie or len(cookie)>8192:raise RuntimeError('invalid private cookie transfer')
                path='/tmp/omabit-cross-'+ship+'-cookie'
                target['execute'](['exec','-i',target['report']['anchor'],'/bin/sh','-c','umask 077; cat > '+path],data=cookie)
                del cookie
                status,_=target['http'](target['report']['ships'][ship]['http_port'],'/~/scry/hood/kiln/pikes.json',cookie=path)
                target['execute'](['exec',target['report']['anchor'],'/bin/rm',path])
            row={'from':source['report']['project'],'to':target['report']['project'],'ship':ship,'status':status};report['cross_cookie_checks'].append(row)
            if status not in (401,403):raise RuntimeError('same-identity session crossed project boundary')
    def observe(context):
        current=context['recovery'].expected()
        context['recovery'].verify_state(*context['state_http']('nec','/~/scry/omabit-state/state.json'))
        report['state_checks'].append({'project':context['report']['project'],'state':current})
        return current
    original_a,original_b=observe(a),observe(b)
    if set(original_a['entries']) & set(original_b['entries']):raise RuntimeError('project mutation identities overlap')
    a['recovery'].write('project-a-isolation')
    changed_a=observe(a)
    if observe(b)!=original_b or changed_a['revision']!=original_a['revision']+1:raise RuntimeError('A mutation crossed project boundary')
    b['recovery'].write('project-b-isolation')
    changed_b=observe(b)
    if observe(a)!=changed_a or changed_b['revision']!=original_b['revision']+1:raise RuntimeError('B mutation crossed project boundary')
    report.update(ok=True,independent_state=True,no_published_ports=not published,direct_publication=published)
    return report
