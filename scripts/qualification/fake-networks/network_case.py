"""Two fresh fake galaxies in one guest-only container network namespace."""
import hashlib
import json
import os
from pathlib import Path
import re
import secrets
import stat
import subprocess
import time

ROOT = Path('/opt/omabit-lab')


class CommandFailure(RuntimeError):
    """Keep status metadata without exposing argv, stdin, cookies or raw output."""
    def __init__(self, command, returncode, stdout, stderr):
        self.evidence = {'command': command, 'exit': returncode,
                         'stdout_bytes': len(stdout), 'stderr_bytes': len(stderr)}
        # Curl's exit code identifies timeouts; raw stderr is deliberately omitted.
        super().__init__('private Docker command failed: '+command+' (exit '+str(returncode)+')')


def jam(noun):
    """Encode this probe's small atoms/cells without backreferences (valid jam)."""
    bits = []
    def put(value, width):
        bits.extend((value >> i) & 1 for i in range(width))
    def atom(value):
        if not isinstance(value, int) or value < 0 or value.bit_length() > 4096:
            raise ValueError('probe atom outside bound')
        if value == 0:
            put(1, 1)
        else:
            width = value.bit_length(); size = width.bit_length()
            put(0, size); put(1, 1)
            put(width, size - 1); put(value, width)
    def visit(value):
        if isinstance(value, tuple) and len(value) == 2:
            put(1, 2); visit(value[0]); visit(value[1])
        else:
            put(0, 1); atom(value)
    visit(noun)
    value = sum(bit << i for i, bit in enumerate(bits))
    return value.to_bytes((len(bits) + 7) // 8, 'little')


def cue_atom(data):
    """Decode only a bounded top-level atom, as returned by the standard hi thread."""
    if not 1 <= len(data) <= 1024:
        raise ValueError('unexpected response size')
    value = int.from_bytes(data, 'little'); offset = 0
    def take(width):
        nonlocal offset
        if offset + width > len(data) * 8:
            raise ValueError('truncated jam')
        result = (value >> offset) & ((1 << width) - 1); offset += width
        return result
    if take(1) != 0:
        raise ValueError('expected an atom response')
    size = 0
    while take(1) == 0:
        size += 1
        if size > 13:
            raise ValueError('oversized mat length')
    width = (1 << (size - 1)) + take(size - 1) if size else 0
    result = take(width)
    if value >> offset:
        raise ValueError('trailing jam data')
    return result


def poke_receipt(stream, request_id=1, reject=False):
    """Require one complete matching Gall receipt from a bounded SSE response."""
    if not isinstance(stream, bytes) or len(stream) > 65536:
        raise ValueError('invalid SSE bound')
    data=[]; event_id=None; matches=[]
    for line in stream.decode().splitlines():
        if line.startswith('id:'): event_id=line[3:].strip()
        elif line.startswith('data:'): data.append(line[5:].strip())
        elif not line:
            if data:
                event=json.loads('\n'.join(data))
                if isinstance(event,dict) and type(event.get('id')) is int and event.get('id') == request_id and event.get('response') == 'poke':
                    if event_id is None or not re.fullmatch('0|[1-9][0-9]{0,15}',event_id):
                        raise ValueError('missing or invalid SSE event identity')
                    if reject:
                        if not isinstance(event.get('err'),str) or 'poke-hi-fail' not in event['err'] or 'ok' in event:
                            raise ValueError('expected application rejection absent')
                    elif event.get('ok') != 'ok' or 'err' in event:
                        raise ValueError('application did not accept the poke')
                    matches.append((int(event_id),event))
            event_id=None;data=[]
    if len(matches) != 1: raise ValueError('one complete matching application receipt required')
    return matches[0]


def thread_poke(receiver, message, receipt):
    """Build only this lab's fixed, typed local thread; never interpolate Hoon input."""
    if receiver not in ('zod','nec') or not re.fullmatch('(?:fail|omabit-[a-z-]+-[0-9a-f]{24})',message) or not re.fullmatch('omabit-receipt-[0-9a-f]{32}',receipt):
        raise ValueError('invalid lab poke binding')
    code="=/  m  (strand ,vase)  ;<  ~  bind:m  (poke [~"+receiver+" %hood] %helm-hi !>('"+message+"'))  (pure:m !>('"+receipt+"'))"
    return jam(int.from_bytes(code.encode(),'little'))


def thread_receipt(status, body, receipt, reject=False):
    """Positive receipt exists only after Gall ack; negative must name helm failure."""
    if not isinstance(body,bytes) or len(body)>65536:raise ValueError('invalid receipt bound')
    if reject:
        value=json.loads(body)
        if status!=500 or not isinstance(value,dict) or value.get('term')!='child-failed':raise ValueError('expected thread rejection absent')
        tang=value.get('tang')
        if not isinstance(tang,list) or not all(isinstance(row,list) and all(isinstance(v,str) for v in row) for row in tang):raise ValueError('invalid rejection trace')
        labels={v for row in tang for v in row}
        if not {'thread-fail','poke-fail','poke-hi-fail'}<=labels:raise ValueError('expected Gall rejection absent')
        return value
    if status!=200:raise ValueError('thread submission/receipt failed')
    atom=cue_atom(body);value=atom.to_bytes((atom.bit_length()+7)//8,'little').decode()
    if value!=receipt:raise ValueError('thread receipt binding mismatch')
    return value


def exercise(run, environment, runtime, project="a", allowed_existing=None, on_complete=None):
    if project not in ("a","b") or (project=="b" and (not allowed_existing or on_complete is None)):
        raise RuntimeError("unbound private project comparison")
    project_name="omabit-network-"+project
    if os.geteuid() != 0 or (ROOT/'guest-marker').read_text() != 'offline-import-guest-v1\n':
        raise RuntimeError('admitted guest required')
    if Path('/sys/class/dmi/id/sys_vendor').read_text().strip() != 'QEMU' or sorted(p.name for p in Path('/sys/class/net').iterdir()) != ['lo']:
        raise RuntimeError('private no-NIC QEMU guest required')
    case = json.loads((ROOT/'case.json').read_text())
    if not case.get('exercise_fake_network') or case.get('exercise_import'):
        raise RuntimeError('wrong qualification case')
    owner = ['runuser', '-u', 'lab', '--']
    docker = owner + ['docker', '--host', 'unix:///run/user/1000/urbit-docker.sock']
    results = ROOT/'results'; report = {'ok': False, 'project': project_name, 'ships': {}, 'messages': [], 'attempts': [], 'local_checks': [], 'clean_stops': []}
    created = []; anchor = None; recovery = None; published = None
    publication = bool(case.get('network_published'))
    if publication and project=='b' and not case.get('network_isolation'):raise RuntimeError('second published project requires the isolation case')

    def emit(stage, **values):
        print('OMABIT_NETWORK ' + json.dumps(dict(stage=stage, **values)), flush=True)
    def save():
        (results/('fake-network.json' if project=='a' else 'fake-network-b.json')).write_text(json.dumps(report, indent=2) + '\n')
    def execute(arguments, data=None, timeout=30):
        result = subprocess.run(docker + arguments, input=data, capture_output=True, env=environment, timeout=timeout)
        if result.returncode:
            raise CommandFailure(arguments[0], result.returncode, result.stdout, result.stderr)
        if len(result.stdout) > 4*1024*1024 or len(result.stderr) > 65536:
            raise RuntimeError('private Docker output bound exceeded')
        return result.stdout + result.stderr if arguments[0] == 'logs' else result.stdout
    def inspect(container):
        rows = json.loads(execute(['inspect', container]))
        if len(rows) != 1 or rows[0]['Id'] != container:
            raise RuntimeError('container identity changed')
        return rows[0]
    def new(arguments):
        identity = execute(['create'] + arguments).decode().strip()
        if not re.fullmatch('[0-9a-f]{64}', identity):
            raise RuntimeError('invalid new container identity')
        created.append(identity); return identity
    def netns(container):
        return execute(['exec', container, '/usr/bin/readlink', '/proc/self/ns/net']).decode().strip()
    def http(port, path, data=None, cookie=None, jar=None, content=None, accept=None, timeout=30, form=False, method=None):
        if not isinstance(port, int) or not 1024 <= port <= 65535 or not path.startswith('/'):
            raise RuntimeError('invalid private endpoint')
        if published and port in published.bindings:
            return published.request(port,path,data,cookie,jar,content,accept,timeout,form,method)
        args = ['/usr/bin/curl', '--disable', '--noproxy', '*', '--proxy', '', '--silent', '--show-error', '--max-time', str(timeout), '--max-filesize', '65536', '--write-out', '\n%{http_code}']
        for flag, value in (('--cookie', cookie), ('--cookie-jar', jar)):
            if value:
                if not re.fullmatch('/tmp/omabit-(?:cross-)?(zod|nec)-cookie', value):
                    raise RuntimeError('invalid cookie path')
                args += [flag, value]
        if method:
            if method != 'PUT': raise RuntimeError('unexpected explicit HTTP method')
            args += ['--request', method]
        if content: args += ['--header', 'Content-Type: '+content]
        if accept: args += ['--header', 'Accept: '+accept]
        if data is not None: args += ['--data-urlencode' if form else '--data-binary', 'password@-' if form else '@-']
        args += ['http://127.0.0.1:'+str(port)+path]
        # No credential appears in argv, evidence, or logs. Only the private
        # anchor's tmpfs receives cookie jars, with an explicit private umask.
        try:
            output = execute(['exec', '-i', anchor, '/bin/sh', '-c', 'umask 077; exec "$@"', 'omabit-curl', *args], data=data, timeout=timeout+15)
        except (CommandFailure, subprocess.TimeoutExpired) as error:
            # Readiness connection refusals are routine. Preserve later failures,
            # including only endpoint/stage metadata, never request or response data.
            if path != '/~/host':
                report['http_failure'] = {'port': port, 'path': path,
                    'method': method or ('POST' if data is not None else 'GET'), 'deadline_seconds': timeout,
                    'failure': error.evidence if isinstance(error, CommandFailure) else {'process_timeout': True}}
                save()
            raise
        body, separator, status = output.rpartition(b'\n')
        if not separator or not re.fullmatch(b'[0-9]{3}', status):
            raise RuntimeError('invalid HTTP status trailer')
        return int(status), body
    def state_http(ship, path, data=None, content=None, accept=None, admin=False, method=None):
        item = report['ships'][ship]
        return http(item['loopback_port'] if admin else item['http_port'], path, data,
            cookie=None if admin else '/tmp/omabit-'+ship+'-cookie', content=content,
            accept=accept, method=method, timeout=90)
    def state_events(channel, cursor, expected, initial):
        from state_case import parse_sse, matching_state
        if not re.fullmatch('/~/channel/omabit-state-[0-9a-f]{32}', channel):
            raise RuntimeError('invalid private state channel')
        rows=[]; ack=not initial; fact=False; deadline=time.monotonic()+45
        while time.monotonic()<deadline:
            if published:
                body=published.events(report['ships']['nec']['http_port'],channel,'/tmp/omabit-nec-cookie',cursor)
            else:
                args=['exec',anchor,'/usr/bin/curl','--disable','--noproxy','*','--proxy','',
                    '--silent','--show-error','--no-buffer','--max-time','3','--max-filesize','65536',
                    '--cookie','/tmp/omabit-nec-cookie','--header','Accept: text/event-stream',
                    '--write-out','\n%{http_code}']
                if cursor is not None:args+=['--header','Last-Event-ID: '+str(cursor)]
                args+=['http://127.0.0.1:'+str(report['ships']['nec']['http_port'])+channel]
                response=subprocess.run(docker+args,capture_output=True,env=environment,timeout=15)
                # Curl 28 is the deliberate end of this bounded SSE connection.
                # Preserve complete events only; a timeout without a matching fact fails.
                if response.returncode not in (0,28) or len(response.stdout)>65536 or len(response.stderr)>8192:
                    raise RuntimeError('bounded SSE connection failed')
                body,separator,status=response.stdout.rpartition(b'\n')
                if not separator or status!=b'200':raise RuntimeError('SSE reconnect refused')
            for identity,event in parse_sse(body.splitlines(keepends=True)):
                rows.append((identity,event));cursor=identity
                fact=matching_state(event,expected) or fact
                ack=ack or (event.get('response')=='subscribe' and event.get('id')==1 and event.get('ok')=='ok')
            if len(rows)>100:raise RuntimeError('SSE event count bound')
            if fact and ack:return rows
        raise RuntimeError('SSE deadline without matching state')
    def ports(ship):
        p = Path(report['ships'][ship]['pier'])/'.http.ports'; mode = p.lstat().st_mode
        if not stat.S_ISREG(mode) or p.stat().st_size > 1024:
            raise RuntimeError('invalid runtime port file')
        rows = [line.split() for line in p.read_text().splitlines()]
        if not rows or any(len(row) != 3 or not row[0].isdigit() or row[1] not in ('secure', 'insecure') or row[2] not in ('loopback', 'public') for row in rows):
            raise RuntimeError('invalid runtime port inventory')
        loop = [int(p) for p, security, exposure in rows if security == 'insecure' and exposure == 'loopback']
        public = [int(p) for p, security, exposure in rows if security == 'insecure' and exposure == 'public']
        if len(loop) != 1 or public != [report['ships'][ship]['http_port']]:
            raise RuntimeError('runtime endpoint differs from selected ship')
        payload = json.dumps({'source': {'dojo': 'our'}, 'sink': {'stdout': None}}).encode()
        status, body = http(loop[0], '/', payload, content='application/json')
        if status != 200 or json.loads(body).strip() != '~'+ship:
            raise RuntimeError('loopback administrative identity mismatch')
        report['ships'][ship]['port_inventory'] = rows
        report['ships'][ship]['loopback_port'] = loop[0]
    def ready(ship, deadline_seconds=900):
        item = report['ships'][ship]; deadline = time.monotonic()+deadline_seconds; notice = 0
        while time.monotonic() < deadline:
            try:
                status, body = http(item['http_port'], '/~/host', timeout=3)
            except (OSError, ValueError, RuntimeError, subprocess.TimeoutExpired):
                status, body = 0, b''
            if status == 200:
                if body.strip() != ('~'+ship).encode():
                    raise RuntimeError('HTTP endpoint belongs to a different ship')
                ports(ship); emit('ready', ship='~'+ship); return
            current = inspect(item['container'])
            if not current['State']['Running'] or current['State']['OOMKilled']:
                raise RuntimeError('fresh '+ship+' exited during boot')
            if time.monotonic()-notice > 30:
                emit('booting', ship='~'+ship); notice = time.monotonic()
            time.sleep(3)
        raise RuntimeError('fresh '+ship+' did not become ready within deadline')
    def login(ship):
        item = report['ships'][ship]
        payload = json.dumps({'source': {'dojo': '+code'}, 'sink': {'stdout': None}}).encode()
        status, body = http(item['loopback_port'], '/', payload, content='application/json')
        if status != 200:
            raise RuntimeError('private login code unavailable')
        code = json.loads(body).strip().removeprefix('~')
        if not re.fullmatch('[a-z]{6}(?:-[a-z]{6}){3}', code):
            raise RuntimeError('invalid private login code')
        status, _ = http(item['http_port'], '/~/login', code.encode(), jar='/tmp/omabit-'+ship+'-cookie', form=True)
        del code, body
        if not 200 <= status < 300:
            raise RuntimeError('fresh ship login refused')
        status, body = http(item['http_port'], '/~/scry/hood/kiln/pikes.json', cookie='/tmp/omabit-'+ship+'-cookie')
        if status != 200 or 'base' not in json.loads(body):
            raise RuntimeError('authenticated desk inventory unavailable')
        item['authenticated_inventory'] = True; save()
    def hi(sender, receiver, phase, reject=False):
        message='fail' if reject else 'omabit-'+phase+'-'+secrets.token_hex(12)
        receipt='omabit-receipt-'+secrets.token_hex(16)
        item=report['ships'][sender]
        attempt={'phase':phase,'sender':'~'+sender,'receiver':'~'+receiver,
            'nonce':message,'expected_receipt':receipt,'state':'pending','transport':'local-khan-thread'}
        report['attempts'].append(attempt);save()
        emit('message-start',sender='~'+sender,receiver='~'+receiver,phase=phase)
        status,body=http(item['http_port'],'/spider/base/ted-eval/khan-eval/noun',
            thread_poke(receiver,message,receipt),cookie='/tmp/omabit-'+sender+'-cookie',
            content='application/x-urb-jam',accept='application/json' if reject else 'application/x-urb-jam',timeout=90)
        attempt.update(http_status=status,response_bytes=len(body),response_sha256=hashlib.sha256(body).hexdigest());save()
        outcome=thread_receipt(status,body,receipt,reject=reject)
        attempt.update(state='rejected' if reject else 'acknowledged',receipt=outcome)
        if reject:report['remote_nack_propagated']=True
        else:report['local_checks' if sender==receiver else 'messages'].append({
            'phase':phase,'sender':'~'+sender,'receiver':'~'+receiver,'nonce':message,'receipt':outcome})
        save();emit('message',sender='~'+sender,receiver='~'+receiver,phase=phase,rejected=reject)

    def diagnostic_snapshot():
        """Best-effort public probe facts before stopping only our new containers."""
        snapshot = {'ships': {}, 'errors': []}
        for name in ('udp', 'tcp', 'snmp', 'dev', 'route'):
            try:
                value = execute(['exec', anchor, '/bin/cat', '/proc/net/'+name])
                if len(value) > 32768: raise RuntimeError('network diagnostic size bound')
                snapshot[name] = value.decode()
            except Exception as error:
                snapshot['errors'].append(name+': '+type(error).__name__)
        for ship, item in report['ships'].items():
            facts = {}; snapshot['ships'][ship] = facts
            try:
                current = inspect(item['container'])
                facts['state'] = {key: current['State'][key] for key in ('Running', 'ExitCode', 'OOMKilled')}
                logs = execute(['logs', '--tail', '250', item['container']]).decode(errors='replace')
                # Only public transport/probe lines are retained. Boot output can
                # contain credentials, so never dump it into the report.
                facts['transport_lines'] = [line for line in logs.splitlines()
                    if re.fullmatch(r'(?:ames|mesa): live on [0-9]+ \(localhost only\)', line)
                    or re.fullmatch(r'.*< ~(?:zod|nec): omabit-[a-z-]+-[0-9a-f]{24}', line)
                    or line == '%poke-hi-fail']
                if item.get('authenticated_inventory'):
                    # Eyre adds the output mark from the extension. The Spider
                    # path is /tree, and %noun supplies a binary jam MIME body.
                    status, body = http(item['http_port'], '/~/scry/spider/tree.noun',
                        cookie='/tmp/omabit-'+ship+'-cookie', timeout=10)
                    facts['spider_tree'] = {'status': status, 'bytes': len(body)}
                    if status == 200 and len(body) < 8192:
                        # This endpoint exposes only public thread identifiers.
                        facts['spider_tree']['jam_hex'] = body.hex()
            except Exception as error:
                facts['error'] = type(error).__name__
        report['diagnostic_snapshot'] = snapshot; save()

    def ship_container(ship, port, parent, resume=False, replacement=False):
        arguments = ['--name',project_name+'-'+ship+('-anchor-recovery' if replacement else '-resume' if resume else ''),
            '--label','org.omabit.network-lab='+project,'--network','container:'+anchor,
            '--init','--user','0:0','--cap-drop','ALL','--security-opt','no-new-privileges',
            '--cgroupns','private','--ipc','private','--restart','no','--stop-signal','SIGTERM',
            '--stop-timeout','300','--memory=4g','--memory-swap=4g','--pids-limit=256','--cpus=1',
            '--log-driver','local','--log-opt','max-size=5m','--log-opt','max-file=2',
            '--mount','type=bind,src='+str(parent)+',dst=/urbit']
        if not resume:
            arguments += ['--mount','type=bind,src='+str(ROOT/'runtime/boot.pill')+',dst=/artifacts/boot.pill,readonly']
        arguments += ['--entrypoint','/bin/urbit',runtime['image'],'-t','-L','--no-conn',
            '--no-dock','--loom','31','--snap-time','60','-b','0.0.0.0' if publication else '127.0.0.1','--http-port',str(port)]
        # Vere refuses -c/-F on an existing pier. Resume from the durable pier;
        # preserve the stopped boot container as evidence rather than modifying it.
        if not resume: arguments += ['-F',ship,'-B','/artifacts/boot.pill','-c']
        return new(arguments + ['/urbit/'+ship])

    def refuse_endpoint(port):
        if not published or port not in published.bindings:raise RuntimeError('unbound recovery endpoint')
        from anchor_recovery import endpoint_probe
        check=subprocess.run(owner+['/usr/bin/curl','--disable','--noproxy','*','--proxy','','--silent','--max-time','2','--max-filesize','128','--write-out','\n%{http_code}','http://127.0.0.1:'+str(published.bindings[port])+'/~/host'],capture_output=True,env=environment,timeout=5)
        facts=endpoint_probe(check.returncode,check.stdout,check.stderr)
        report['publication'].setdefault('recovery_probes',[]).append(dict(port=port,published_port=published.bindings[port],**facts));save()
        if not facts['unavailable']:raise RuntimeError('unavailable anchor endpoint did not establish absence of an HTTP response')
        return facts

    def stopped(ship):
        item = report['ships'][ship];execute(['stop', '--timeout', '300', item['container']], timeout=330)
        current = inspect(item['container'])
        if current['State']['Running'] or current['State']['ExitCode'] != 0 or current['State']['OOMKilled']:
            raise RuntimeError('fresh '+ship+' did not stop cleanly')
        report['clean_stops'].append(ship);save()

    try:
        existing=set(execute(['ps','--all','--quiet','--no-trunc']).decode().split())
        if existing!=set(allowed_existing or []):
            raise RuntimeError('network lab engine inventory differs from its own admitted containers')
        root = Path('/home/lab/fake-network-'+project);root.mkdir(mode=0o700);os.chown(root,1000,1000)
        anchor_network=['--network','none']
        if publication:
            from published_client import PublishedClient
            from endpoints_case import published_bindings
            bridge=execute(['network','create','--driver','bridge','--label','org.omabit.network-lab='+project,project_name+'-bridge']).decode().strip()
            if not re.fullmatch('[0-9a-f]{64}',bridge):raise RuntimeError('invalid new project bridge identity')
            bindings={8081:18281,8082:18282} if project=='a' else {8081:18291,8082:18292}
            report['publication']={'bridge':bridge,'bindings':{str(k):v for k,v in bindings.items()},'requests':[],'fixture_only':False}
            anchor_network=['--network',bridge,'--init']
            for port,host in bindings.items():anchor_network+=['--publish','127.0.0.1:'+str(host)+':'+str(port)+'/tcp']
            published=PublishedClient(bindings,report['publication']['requests'],save)
        anchor = new(['--name',project_name+'-anchor','--label','org.omabit.network-lab='+project,*anchor_network,'--read-only','--tmpfs','/tmp:rw,nosuid,nodev,noexec,size=16m','--user','0:0','--cap-drop','ALL','--security-opt','no-new-privileges','--restart','no','--memory=128m','--memory-swap=128m','--pids-limit=32','--cpus=0.25','--entrypoint','/bin/sleep',runtime['image'],'7200'])
        execute(['start',anchor]);namespace=netns(anchor)
        if namespace == os.readlink('/proc/self/ns/net'):
            raise RuntimeError('anchor shares guest host namespace')
        report.update(anchor=anchor,network_namespace=namespace)
        if publication:
            observed=inspect(anchor)
            report['publication']['anchor_observation']={'host_bindings':observed['HostConfig']['PortBindings'],'active_ports':observed['NetworkSettings']['Ports'],'networks':observed['NetworkSettings']['Networks']};save()
            published_bindings(observed,published.bindings)
            attachments=observed['NetworkSettings']['Networks']
            if len(attachments)!=1 or next(iter(attachments.values()))['NetworkID']!=bridge:raise RuntimeError('published anchor bridge mismatch')
            report['publication']['ipv6_refusals']=[]
            for port in published.bindings.values():
                check=subprocess.run(owner+['/usr/bin/curl','--disable','--noproxy','*','--proxy','','--silent','--max-time','2','http://[::1]:'+str(port)+'/~/host'],capture_output=True,env=environment,timeout=5)
                if check.returncode not in (7,28):raise RuntimeError('unexpected published IPv6 reachability')
                report['publication']['ipv6_refusals'].append({'port':port,'exit':check.returncode})
            save()
        for ship, port in [('zod',8081),('nec',8082)]:
            parent=root/ship;parent.mkdir(mode=0o700);os.chown(parent,1000,1000)
            container=ship_container(ship,port,parent)
            report['ships'][ship]={'container':container,'pier':str(parent/ship),'http_port':port};save()
            execute(['start',container]);ready(ship)
            current=inspect(container)
            if current['HostConfig']['NetworkMode']!='container:'+anchor or current['HostConfig']['PortBindings'] or netns(container)!=namespace:
                raise RuntimeError('ship escaped selected private project namespace')
            login(ship)
        if report['ships']['zod']['loopback_port']==report['ships']['nec']['loopback_port']:
            raise RuntimeError('administrative endpoints collided')
        routes=execute(['exec',anchor,'/bin/cat','/proc/net/route']).decode()
        devices=execute(['exec',anchor,'/bin/cat','/proc/net/dev']).decode()
        if not publication and (len(routes.strip().splitlines())!=1 or [line.split(':')[0].strip() for line in devices.splitlines() if ':' in line]!=['lo']):
            raise RuntimeError('project has a non-loopback network route/device')
        if publication:
            from endpoints_case import published_bindings
            published_bindings(inspect(anchor),published.bindings)
            report['publication']['admin_bridge_refusals']=[]
            bridge_ip=next(iter(inspect(anchor)['NetworkSettings']['Networks'].values()))['IPAddress']
            import ipaddress
            if ipaddress.ip_address(bridge_ip).version!=4:raise RuntimeError('invalid bridge endpoint')
            for item in report['ships'].values():
                check=subprocess.run(docker+['exec',anchor,'/usr/bin/curl','--disable','--noproxy','*','--proxy','','--silent','--max-time','2','http://'+bridge_ip+':'+str(item['loopback_port'])+'/'],capture_output=True,env=environment,timeout=10)
                if check.returncode not in (7,28):raise RuntimeError('administrative endpoint reachable on bridge address')
                report['publication']['admin_bridge_refusals'].append({'port':item['loopback_port'],'exit':check.returncode})
            save()
        report['ipv4_routes']=routes;report['network_devices']=devices
        for receiver,wrong in [('zod','nec'),('nec','zod')]:
            status,_=http(report['ships'][receiver]['http_port'],'/~/scry/hood/kiln/pikes.json',cookie='/tmp/omabit-'+wrong+'-cookie')
            if status not in (401,403):raise RuntimeError('cross-ship cookie did not give an authentication refusal')
        report['cross_ship_sessions_refused']=True
        hi('zod','zod','local-check');hi('nec','nec','local-check')
        hi('zod','nec','first');hi('nec','zod','first');hi('zod','nec','nack',reject=True)
        if case.get('network_state'):
            from state_case import Recovery
            recovery=Recovery(state_http,state_events,save,emit);report['application_state']=recovery.report
            recovery.install(Path(report['ships']['nec']['pier']),ROOT/'omabit-state.hoon');recovery.before_restart()
        pier=Path(report['ships']['nec']['pier']);before=(pier.stat().st_dev,pier.stat().st_ino)
        stopped('nec')
        old_container=report['ships']['nec']['container']
        resumed=ship_container('nec',report['ships']['nec']['http_port'],pier.parent,resume=True)
        report['restart']={'stopped_boot_container':old_container,'resume_container':resumed,
            'pier_device':before[0],'pier_inode':before[1],'namespace':namespace}
        report['ships']['nec']['container']=resumed;save()
        execute(['start',resumed]);ready('nec',180)
        resumed_state=inspect(resumed)
        if inspect(old_container)['State']['Running'] or resumed_state['HostConfig']['NetworkMode']!='container:'+anchor or resumed_state['HostConfig']['PortBindings']:
            raise RuntimeError('restart container ownership/network contract changed')
        if before!=(pier.stat().st_dev,pier.stat().st_ino) or netns(report['ships']['nec']['container'])!=namespace or netns(anchor)!=namespace:
            raise RuntimeError('restart replaced the pier or project namespace')
        # Retain the original cookie jars: new login is not part of restart recovery.
        hi('zod','nec','after-restart');hi('nec','zod','after-restart')
        report['restart_same_pier_and_namespace']=True;report['restart_kept_sessions']=True
        if recovery:recovery.after_restart()
        if published:
            from endpoints_case import published_bindings
            published_bindings(inspect(anchor),published.bindings)
            report['publication']['resumed_publications_verified']=True;save()
        report['transport_passed']=True
        context={'report':report,'execute':execute,'inspect':inspect,'http':http,'state_http':state_http,'recovery':recovery,'published':published}
        if case.get('network_anchor_recovery'):
            if not published or case.get('network_isolation') or project!='a':raise RuntimeError('anchor recovery requires its separate published single-project case')
            from anchor_recovery import recover
            context.update(netns=netns,save=save,stop=stopped,ready=ready,refuse_endpoint=refuse_endpoint,
                           replacement=lambda ship,port,parent:ship_container(ship,port,parent,resume=True,replacement=True))
            recover(context)
        if on_complete:on_complete(context)
        if case.get('network_isolation') and project=='a':
            if recovery is None:raise RuntimeError('project isolation requires persistent state fixtures')
            from project_isolation import verify
            def compare(other):
                report['project_isolation']=verify(context,other);save()
            report['second_project']=exercise(run,environment,runtime,project='b',allowed_existing=created.copy(),on_complete=compare)
            if not report.get('project_isolation',{}).get('ok'):raise RuntimeError('cross-project comparison absent')
            save()
    except Exception as error:
        report['error']=str(error);emit('failure',error=str(error)); save()
        if anchor:
            # Diagnostics must not replace the original failure or prevent cleanup.
            original_http_failure = report.get('http_failure')
            try: diagnostic_snapshot()
            except Exception as diagnostic_error:
                report['diagnostic_error'] = type(diagnostic_error).__name__
            if original_http_failure: report['http_failure'] = original_http_failure
    finally:
        cleanup_errors=[]
        if recovery and recovery.channel:
            try:recovery.close()
            except Exception as error:cleanup_errors.append('subscription: '+type(error).__name__)
        if published:
            try:report['publication']['logout_checks']=published.logout()
            except Exception as error:cleanup_errors.append('published sessions: '+type(error).__name__)
        for ship in reversed(list(report['ships'])):
            try:
                if inspect(report['ships'][ship]['container'])['State']['Running']:stopped(ship)
            except Exception as error:cleanup_errors.append(str(error))
        if anchor:
            try:
                execute(['stop','--timeout','15',anchor],timeout=30)
                anchor_state=inspect(anchor)['State']
                report['anchor_stop']={key:anchor_state[key] for key in ('Running','ExitCode','OOMKilled')}
                if anchor_state['Running'] or (publication and (anchor_state['OOMKilled'] or anchor_state['ExitCode'] not in (0,143))):raise RuntimeError('anchor did not stop cleanly')
            except Exception as error:cleanup_errors.append(str(error))
        report['cleanup_errors']=cleanup_errors
        report['ok']=report.get('transport_passed',False) and not cleanup_errors and 'error' not in report
        save();emit('complete',ok=report['ok'],messages=len(report['messages']),clean_stops=report['clean_stops'])
    if not report['ok']:raise RuntimeError('private fake-network gate failed; original disks retained')
    return report
