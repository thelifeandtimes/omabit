"""Rootless publication/anchor observations with disposable HTTP responders."""
import json
import os
from pathlib import Path
import re
import subprocess
import time

ROOT=Path('/opt/omabit-lab')


def published_bindings(item, expected):
    """Require exact IPv4 loopback publications; reject wildcard/IPv6/extra ports."""
    for where in ('HostConfig','NetworkSettings'):
        actual=item[where]['PortBindings' if where=='HostConfig' else 'Ports'] or {}
        # Image EXPOSE entries can be present with no mapping; they publish nothing.
        mapped={k:v for k,v in actual.items() if v}
        wanted={str(port)+'/tcp':[{'HostIp':'127.0.0.1','HostPort':str(host)}] for port,host in expected.items()}
        if mapped!=wanted:raise RuntimeError('unexpected anchor publication at '+where)


def exercise(environment, runtime):
    if os.geteuid()!=0 or (ROOT/'guest-marker').read_text()!='offline-import-guest-v1\n':raise RuntimeError('admitted guest required')
    if Path('/sys/class/dmi/id/sys_vendor').read_text().strip()!='QEMU' or sorted(p.name for p in Path('/sys/class/net').iterdir())!=['lo']:raise RuntimeError('private no-NIC guest required')
    case=json.loads((ROOT/'case.json').read_text())
    if not case.get('exercise_endpoints') or case.get('exercise_fake_network') or case.get('exercise_import'):raise RuntimeError('wrong endpoint case')
    mode=case['endpoint_network']
    if mode not in ('internal','egress'):raise RuntimeError('unsupported endpoint network')
    internal=mode=='internal'
    docker=['runuser','-u','lab','--','docker','--host','unix:///run/user/1000/urbit-docker.sock']
    report={'ok':False,'fixture_only':True,'network_mode':mode,'projects':{},'requests':[],'stop_observations':[],'publication_observations':[],'clean_stops':[],'cleanup_errors':[]}
    created=[]
    def save():
        (ROOT/'results/endpoints.json').write_text(json.dumps(report,indent=2)+'\n')
    def emit(stage):print('OMABIT_ENDPOINTS '+json.dumps({'stage':stage,'network_mode':mode}),flush=True)
    def execute(args,timeout=30,allow_failure=False):
        r=subprocess.run(docker+args,capture_output=True,env=environment,timeout=timeout)
        if len(r.stdout)>65536 or len(r.stderr)>65536:raise RuntimeError('endpoint command output bound')
        if r.returncode and not allow_failure:raise RuntimeError('endpoint Docker '+args[0]+' failed (exit '+str(r.returncode)+')')
        return r
    def inspect(identity):
        rows=json.loads(execute(['inspect',identity]).stdout)
        if len(rows)!=1 or rows[0]['Id']!=identity:raise RuntimeError('endpoint container identity mismatch')
        return rows[0]
    def namespace(identity):return execute(['exec',identity,'/usr/bin/readlink','/proc/self/ns/net']).stdout.decode().strip()
    def new(args):
        identity=execute(['create',*args]).stdout.decode().strip()
        if not re.fullmatch('[0-9a-f]{64}',identity):raise RuntimeError('invalid endpoint container ID')
        created.append(identity);save();return identity
    def stop(identity):
        execute(['stop','--timeout','15',identity],timeout=30)
        observed=inspect(identity)
        report['stop_observations'].append({'container':identity,'state':observed['State'],'entrypoint':observed['Config']['Entrypoint']});save()
        allowed=(0,) if observed['Config']['Entrypoint']==['/endpoint-probe'] else (0,143)
        if observed['State']['Running'] or observed['State']['OOMKilled'] or observed['State']['ExitCode'] not in allowed:raise RuntimeError('endpoint fixture did not stop cleanly')
        report['clean_stops'].append({'container':identity,'exit':observed['State']['ExitCode']});save()
    def publication(identity, expected):
        observed=inspect(identity)
        report['publication_observations'].append({'container':identity,'expected':expected,
            'host_bindings':observed['HostConfig']['PortBindings'],'active_ports':observed['NetworkSettings']['Ports'],
            'networks':observed['NetworkSettings']['Networks'],'state':observed['State']});save()
        published_bindings(observed,expected)
        return observed
    def request(host,port,container=None):
        url=('http://['+host+']:' if ':' in host else 'http://'+host+':')+str(port)+'/identity'
        args=['/usr/bin/curl','--disable','--noproxy','*','--proxy','','--silent','--show-error','--max-time','2','--max-filesize','1024','--write-out','\n%{http_code}',url]
        r=execute(['exec',container,*args],timeout=10,allow_failure=True) if container else subprocess.run(['runuser','-u','lab','--',*args],capture_output=True,env=environment,timeout=5)
        if len(r.stdout)>2048 or len(r.stderr)>2048:raise RuntimeError('endpoint HTTP output bound')
        body,_,status=r.stdout.rpartition(b'\n');value=None
        if r.returncode==0:
            if status!=b'200':raise RuntimeError('unexpected endpoint HTTP status')
            value=json.loads(body)
        report['requests'].append({'host':host,'port':port,'from_container':container,'exit':r.returncode,'status':status.decode(),'value':value});save()
        return value
    def member(project,slot,phase):
        p=report['projects'][project];port=8081 if slot=='one' else 8082;name=project+'-'+slot+'-'+phase
        identity=new(['--name','omabit-endpoint-'+name,'--label','org.omabit.endpoint-lab='+project,'--network','container:'+p['anchor'],
            '--read-only','--cap-drop','ALL','--security-opt','no-new-privileges','--restart','no','--memory=64m','--memory-swap=64m','--pids-limit=32','--cpus=0.25',
            '--mount','type=bind,src='+str(ROOT/'endpoint-probe')+',dst=/endpoint-probe,readonly','--entrypoint','/endpoint-probe',runtime['image'],'--name',name,'--port',str(port)])
        p['members'][slot]={'container':identity,'port':port,'fixture':name};save();execute(['start',identity])
        item=inspect(identity)
        if not item['State']['Running'] or item['HostConfig']['NetworkMode']!='container:'+p['anchor'] or item['HostConfig']['PortBindings'] or namespace(identity)!=p['namespace']:raise RuntimeError('endpoint member namespace mismatch')
        expected={'fixture':name,'port':port};deadline=time.monotonic()+15
        while time.monotonic()<deadline:
            if request('127.0.0.1',port,container=identity)==expected:return
            time.sleep(.25)
        raise RuntimeError('fixture local endpoint unavailable')
    try:
        if execute(['ps','--all','--quiet','--no-trunc']).stdout.strip():raise RuntimeError('endpoint engine not empty')
        emit('create-projects')
        for project,ports in [('a',{8081:18281,8082:18282}),('b',{8081:18283,8082:18284})]:
            network=execute(['network','create','--driver','bridge','--internal='+str(internal).lower(),'--ipv4=true','--ipv6=false',
                '--label','org.omabit.endpoint-lab='+project,'--opt','com.docker.network.bridge.host_binding_ipv4=127.0.0.1','--opt','com.docker.network.bridge.gateway_mode_ipv4=nat','omabit-endpoint-'+project]).stdout.decode().strip()
            if not re.fullmatch('[0-9a-f]{64}',network):raise RuntimeError('invalid endpoint network ID')
            p={'network':network,'members':{},'ports':ports};report['projects'][project]=p;save()
            args=['--name','omabit-endpoint-'+project+'-anchor','--label','org.omabit.endpoint-lab='+project,'--network',network,'--read-only','--cap-drop','ALL','--security-opt','no-new-privileges','--restart','no','--memory=64m','--memory-swap=64m','--pids-limit=32','--cpus=0.25']
            for port,host in ports.items():args+=['--publish','127.0.0.1:'+str(host)+':'+str(port)+'/tcp']
            p['anchor']=new(args+['--init','--entrypoint','/bin/sleep',runtime['image'],'1200']);execute(['start',p['anchor']]);p['namespace']=namespace(p['anchor'])
            item=publication(p['anchor'],ports)
            networks=item['NetworkSettings']['Networks']
            if len(networks)!=1:raise RuntimeError('unexpected anchor networks')
            p['ip']=next(iter(networks.values()))['IPAddress']
            net=json.loads(execute(['network','inspect',network]).stdout)[0]
            if net['Id']!=network or net['Internal']!=internal or net['EnableIPv6'] or set(net['Containers'])!={p['anchor']}:raise RuntimeError('unexpected bridge membership')
            p['routes']=execute(['exec',p['anchor'],'/bin/cat','/proc/net/route']).stdout.decode()
            if internal and any(line.split()[1]=='00000000' for line in p['routes'].splitlines()[1:]):raise RuntimeError('internal bridge has default route')
            member(project,'one','initial');member(project,'two','initial')
            for row in p['members'].values():
                expected={'fixture':row['fixture'],'port':row['port']}
                if request('127.0.0.1',ports[row['port']])!=expected:raise RuntimeError('published endpoint unreachable or wrong identity')
                if request('::1',ports[row['port']]) is not None:raise RuntimeError('unexpected IPv6 publication')
        a,b=report['projects']['a'],report['projects']['b']
        if a['namespace']==b['namespace']:raise RuntimeError('endpoint projects share namespace')
        for source,target in ((a,b),(b,a)):
            for port in (8081,8082):
                if request(target['ip'],port,container=source['anchor']) is not None:raise RuntimeError('unpublished endpoint crossed bridge boundary')
        report['initial_publication_passed']=True;save();emit('anchor-stop-with-members-live')
        old_namespace=a['namespace'];stop(a['anchor'])
        for row in a['members'].values():
            if not inspect(row['container'])['State']['Running'] or namespace(row['container'])!=old_namespace:raise RuntimeError('anchor stop changed member runtime')
            if request('127.0.0.1',row['port'],container=row['container'])!={'fixture':row['fixture'],'port':row['port']}:raise RuntimeError('member lost original local endpoint')
        report['anchor_stopped_publication']=[request('127.0.0.1',a['ports'][row['port']]) for row in a['members'].values()];save()
        started=execute(['start',a['anchor']],allow_failure=True)
        restart={'exit':started.returncode,'old_namespace':old_namespace};report['anchor_restart']=restart
        if started.returncode==0:
            restart['new_namespace']=namespace(a['anchor']);publication(a['anchor'],a['ports'])
            reachable=[]
            for row in a['members'].values():
                reachable.append(request('127.0.0.1',a['ports'][row['port']])=={'fixture':row['fixture'],'port':row['port']})
            restart['published_reachable']=reachable
            restart['outcome']='namespace-retained' if restart['new_namespace']==old_namespace else 'namespace-replaced'
            if restart['new_namespace']==old_namespace and reachable!=[True,True]:raise RuntimeError('retained anchor lost publication')
            if restart['new_namespace']!=old_namespace and any(reachable):raise RuntimeError('replacement namespace reached old members')
        else:restart['outcome']='start-refused'
        save()
        # Explicit coordinated repair only after stopping every member.
        for row in a['members'].values():stop(row['container'])
        if not inspect(a['anchor'])['State']['Running']:execute(['start',a['anchor']])
        publication(a['anchor'],a['ports'])
        a['namespace']=namespace(a['anchor']);a['retired_members']=list(a['members'].values());a['members']={}
        member('a','one','recovered');member('a','two','recovered')
        for project,p in report['projects'].items():
            for row in p['members'].values():
                if request('127.0.0.1',p['ports'][row['port']])!={'fixture':row['fixture'],'port':row['port']}:raise RuntimeError('coordinated recovery confused project endpoint')
        report['coordinated_recovery_passed']=True;save()
    except Exception as error:
        report['error']=str(error);save()
    finally:
        for identity in reversed(created):
            try:
                if inspect(identity)['State']['Running']:stop(identity)
            except Exception as error:report['cleanup_errors'].append(str(error))
        report['ok']=report.get('coordinated_recovery_passed',False) and not report['cleanup_errors'] and 'error' not in report
        save();emit('complete')
    if not report['ok']:raise RuntimeError('private endpoint gate failed; evidence retained')
    return report
