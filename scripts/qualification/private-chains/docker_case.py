"""Fresh no-NIC VM only: chain gateway plus two official keyed Vere containers."""
import hashlib,json,os,re,subprocess,time,uuid
from pathlib import Path
from docker_runtime import Engine,Ship,IMAGE,IMAGE_ID,LABEL,identity,require_guest,image_matches,receipt_name
import docker_peer
ROOT=Path('/opt/omabit-lab')
INPUT=ROOT/'private-chain'

def initial_publication_ports(ports):
 if ports is not None and (type(ports) is not tuple or len(ports)!=2 or any(type(p) is not int or not 1024<=p<=65535 for p in ports) or ports[0]==ports[1]):raise ValueError('explicit two-port initial publication required')
 return ports

def exercise(environment,peer_checkpoint=None,*,label=None,publication_ports=None):
 require_guest();receipt_name("chain",label);initial_publication_ports(publication_ports);engine=Engine(environment);project=uuid.uuid4().hex
 root=Path('/home/lab')/('omabit-private-chain-'+project)
 report={'ok':False,'project':project,'cleanup_errors':[]};anchor=None;stopping=False;publication=None
 def save():
  p=ROOT/'results'/receipt_name('chain',label);q=p.with_suffix('.tmp')
  with q.open('w') as f:json.dump(report,f,indent=2);f.write('\n');f.flush();os.fsync(f.fileno())
  q.replace(p)
  fd=os.open(p.parent,os.O_DIRECTORY);os.fsync(fd);os.close(fd)
 def emit(stage):print('OMABIT_PRIVATE_CHAIN '+json.dumps({'stage':stage}),flush=True)
 def checked():
  row=engine.inspect(anchor);h=row['HostConfig'];c=row['Config']
  expected={str(root):('/lab',True)}
  expected.update({str(INPUT/n):(target,False) for n,target in [('node','/node'),('tools','/tools'),('source','/source'),('compiled','/compiled'),('deployment','/deployment'),('vere','/vere'),('src/docker_gateway.cjs','/gateway.cjs')]})
  actual={m.get('Source'):(m.get('Destination'),m.get('RW')) for m in row['Mounts'] if m.get('Type')=='bind'}
  if (not image_matches(row) or row['Name']!='/omabit-private-'+project+'-gateway' or c['Labels'].get(LABEL)!=project
   or c['Entrypoint']!=['/node/usr/lib/ld-linux-x86-64.so.2'] or c['Cmd']!=['--library-path','/node/usr/lib','/node/usr/bin/node','/gateway.cjs']
   or (publication is None and (h['NetworkMode']!='none' or h.get('PortBindings'))) or h['Privileged'] or not h['ReadonlyRootfs'] or h['Init'] is not True
   or h['Memory']!=1536*1024**2 or h['MemorySwap']!=1536*1024**2 or h['NanoCpus']!=2000000000 or h['PidsLimit']!=128
   or h.get('CapAdd') or h['CapDrop']!=['ALL'] or h['RestartPolicy']['Name']!='no' or c['User']!='0:0' or c['WorkingDir']!='/lab'
   or sorted(x for x in c.get('Env',[]) if x.startswith(('LIBC=','PREBUILDS_ONLY=')))!=['LIBC=glibc','PREBUILDS_ONLY=1']
   or h['CgroupnsMode']!='private' or h['IpcMode']!='private' or h['SecurityOpt'] not in (['no-new-privileges'],['no-new-privileges=true'])
   or actual!=expected or sum(m.get('Type')=='bind' for m in row['Mounts'])!=len(expected)):
   raise RuntimeError('gateway identity/isolation binding changed')
  if h.get('Tmpfs')!={'/tmp':'rw,nosuid,nodev,mode=1777,size=64m','/urbit':'ro,nosuid,nodev,noexec,mode=0555,size=64k'}:raise RuntimeError('gateway temporary mounts differ')
  if any(m.get('Type')=='tmpfs' and m.get('Destination')=='/urbit' and m.get('RW') is not False for m in row['Mounts']):raise RuntimeError('gateway image volume is writable')
  if any(m.get('Type')!='bind' and not (m.get('Type')=='tmpfs' and m.get('Destination') in ('/tmp','/urbit')) for m in row['Mounts']):raise RuntimeError('gateway filesystem differs')
  selected=[v for v in c.get('Env',[]) if v.startswith('OMABIT_PRIVATE_CHAIN_NETWORK=')]
  if selected!=(['OMABIT_PRIVATE_CHAIN_NETWORK=published-v1'] if publication else []):raise RuntimeError('initial gateway network environment differs')
  if publication is not None:
   publication.check_gateway_configuration(row,anchor)
   if row['State']['Running']:
    publication.check_gateway_network(row,anchor)
    networks=json.loads(engine.execute(['network','inspect',publication.network]))
    if len(networks)!=1:raise RuntimeError('initial publication bridge observation differs')
    publication.check_network(networks[0],anchor)
    observed={'network':networks[0],'gateway':{'Id':row['Id'],'Image':row['Image'],'Config':{k:c[k] for k in ('Image','Labels')},'HostConfig':{k:h[k] for k in ('NetworkMode','PortBindings')},'NetworkSettings':{k:row['NetworkSettings'][k] for k in ('Ports','Networks')},'State':{k:row['State'][k] for k in ('Running','OOMKilled','Status')}}}
    report['publication'].setdefault('first_observation',observed);report['publication']['last_observation']=observed
   elif row['State']['Status']=='created':publication.check_gateway_network(row,anchor,running=False)
  return row
 try:
  # Only this generated project root is writable by the gateway; no host engine,
  # credential store, existing pier or installation is mounted in any container.
  root.mkdir(mode=0o700);os.chown(root,1000,1000)
  for name in ('home','piers'):
   p=root/name;p.mkdir(mode=0o700);os.chown(p,1000,1000)
  for name,data in [('marker','private-chain-docker-gateway-v1\n'),('guest-namespaces.json',json.dumps({n:os.readlink('/proc/self/ns/'+n) for n in ('net','pid','mnt')})+'\n')]:
   p=root/name;p.write_text(data);p.chmod(0o600);os.chown(p,1000,1000)
  if publication_ports is not None:
   from docker_publication import Publication,network_create_command
   report['network_create_dispatched']=True;save()
   network=identity(engine.execute(network_create_command(project)).decode().strip())
   publication=Publication(project,network,*publication_ports)
   report['publication']={'network':network,'name':publication.name,'ports':publication.ports,'loopback_only':True,'initial_start':True};save()
   networks=json.loads(engine.execute(['network','inspect',network]))
   if len(networks)!=1:raise RuntimeError('initial bridge creation observation differs')
   publication.check_network(networks[0])
  args=['create','--name','omabit-private-'+project+'-gateway','--label',LABEL+'='+project,'--network','none','--pull','never','--init','--read-only','--tmpfs','/tmp:rw,nosuid,nodev,mode=1777,size=64m','--tmpfs','/urbit:ro,nosuid,nodev,noexec,mode=0555,size=64k','--workdir','/lab','--user','0:0','--cap-drop','ALL','--security-opt','no-new-privileges','--cgroupns','private','--ipc','private','--restart','no','--memory','1536m','--memory-swap','1536m','--pids-limit','128','--cpus','2','--env','LIBC=glibc','--env','PREBUILDS_ONLY=1','--env','HOME=/lab/home','--env','OMABIT_PRIVATE_CHAIN=private-chain-docker-gateway-v1','--mount','type=bind,src='+str(root)+',dst=/lab']
  if publication is not None:
   at=args.index('--network');args[at:at+2]=publication.gateway_network_arguments();args+=['--env','OMABIT_PRIVATE_CHAIN_NETWORK=published-v1']
  for name,target in [('node','/node'),('tools','/tools'),('source','/source'),('compiled','/compiled'),('deployment','/deployment'),('vere','/vere'),('src/docker_gateway.cjs','/gateway.cjs')]:
   source=INPUT/name
   if source.resolve(strict=True)!=source:raise RuntimeError('gateway input is indirect')
   args+=['--mount','type=bind,src='+str(source)+',dst='+target+',readonly']
  args+=['--entrypoint','/node/usr/lib/ld-linux-x86-64.so.2',IMAGE,'--library-path','/node/usr/lib','/node/usr/bin/node','/gateway.cjs']
  anchor=identity(engine.execute(args).decode().strip());report['anchor']=anchor;save()
  if checked()['State']['Status']!='created':raise RuntimeError('gateway creation not observed')
  report['start_dispatched']=True;save()
  if engine.execute(['start',anchor]).decode().strip()!=anchor:raise RuntimeError('gateway start reply differs')
  emit('gateway-started');deadline=time.monotonic()+300
  while not (root/'gateway-ready.json').exists():
   row=checked()
   if not row['State']['Running']:raise RuntimeError('gateway exited before readiness')
   if time.monotonic()>deadline:raise RuntimeError('gateway readiness deadline')
   time.sleep(2)
  ready=json.loads((root/'gateway-ready.json').read_text());foundation=json.loads((INPUT/'deployment/result.json').read_text())
  if (not ready.get('passed') or ready['chainId']!='0x539' or ready['block']!='0x50' or ready['genesis']!=foundation['before']['genesis']
   or ready.get('initialNetwork')!=('published-v1' if publication else 'none') or not re.fullmatch('0x[0-9a-f]{64}',ready.get('headHash','')) or ready['headHash']=='0x'+'0'*64
   or ready['externalRpcMutationsAllowed'] is not False or ready['address']['address']!='127.0.0.1' or ready['address']['port']!=25432):raise RuntimeError('gateway chain/readiness differs')
  if len(ready['publicKeys'])!=2 or ready['publicKeys'][0]['auth']==ready['publicKeys'][1]['auth']:raise RuntimeError('fresh public key proof differs')
  report['ready']=ready;save();emit('gateway-ready')
  peers=docker_peer.exercise(environment,project,anchor,checkpoint=peer_checkpoint,label=label,publication=publication);report['peers_ok']=peers['ok'];save()
  if not peers['ok']:raise RuntimeError('keyed Docker peer proof failed')
 except Exception as error:report['error']=type(error).__name__+': '+str(error)
 finally:
  if anchor:
   try:
    row=checked()
    # Save opaque diagnostics privately. No bootstrap logs are emitted over serial.
    logs=engine.execute(['logs','--tail','200',anchor],capture_stderr=True);(ROOT/'results'/receipt_name('log',label)).write_bytes(logs)
    if row['State']['Running']:
     report['stop_dispatched']=True;save();stopping=True
     if engine.execute(['kill','--signal','TERM',anchor]).decode().strip()!=anchor:raise RuntimeError('gateway stop acknowledgment differs')
    code=engine.execute(['wait',anchor],timeout=75).decode().strip();state=checked()['State']
    report['gateway_stop']={k:state[k] for k in ('Running','ExitCode','OOMKilled')};save()
    if code!='0' or state['Running'] or state['OOMKilled'] or not stopping:raise RuntimeError('gateway has no owned clean shutdown')
    proof=json.loads((root/'result.json').read_text())
    if not proof['passed'] or not proof['gateway']['signalShutdown'] or proof['gateway']['head']!='0x50' or proof['gateway'].get('headHash')!=report.get('ready',{}).get('headHash'):raise RuntimeError('gateway final proof differs')
    report['gateway_proof']=proof;save()
   except Exception as error:report['cleanup_errors'].append(type(error).__name__+': '+str(error))
  report['ok']=report.get('peers_ok',False) and 'gateway_proof' in report and 'error' not in report and not report['cleanup_errors'];save()
  emit('complete' if report['ok'] else 'failed')
 return report
