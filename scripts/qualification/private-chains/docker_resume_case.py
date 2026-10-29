"""Fresh no-NIC guest only: replace a cleanly stopped gateway over its own chain."""
import hashlib,json,os,re,stat,time,uuid
from pathlib import Path
from docker_runtime import Engine,IMAGE,LABEL,identity,image_matches,require_guest

ROOT=Path('/opt/omabit-lab')
INPUT=ROOT/'private-chain'
MODE='private-chain-docker-gateway-resume-v1'
LOADER='/node/usr/lib/ld-linux-x86-64.so.2'
NODE=[LOADER,'--library-path','/node/usr/lib','/node/usr/bin/node']

def digest(path):
 with path.open('rb') as f:return hashlib.file_digest(f,'sha256').hexdigest()

def directory(path):
 st=path.lstat()
 if path.resolve(strict=True)!=path or not stat.S_ISDIR(st.st_mode) or st.st_uid!=1000 or stat.S_IMODE(st.st_mode)!=0o700:raise RuntimeError('selected private directory differs')
 return (st.st_dev,st.st_ino)

def check_gateway(row,container,project,name,mounts,command,variables,publication=None):
 identity(container)
 if row.get('Id')!=container or not image_matches(row) or row.get('Name')!='/'+name:raise RuntimeError('selected gateway identity differs')
 h=row.get('HostConfig',{});c=row.get('Config',{});state=row.get('State',{})
 if (c.get('Labels',{}).get(LABEL)!=project or c.get('Entrypoint')!=[LOADER] or c.get('Cmd')!=command
  or (publication is None and (h.get('NetworkMode')!='none' or h.get('PortBindings'))) or h.get('Privileged') or h.get('ReadonlyRootfs') is not True or h.get('Init') is not True
  or h.get('Memory')!=1536*1024**2 or h.get('MemorySwap')!=1536*1024**2 or h.get('NanoCpus')!=2000000000 or h.get('PidsLimit')!=128
  or h.get('CapAdd') or h.get('CapDrop')!=['ALL'] or h.get('RestartPolicy',{}).get('Name')!='no' or c.get('User')!='0:0'
  or h.get('CgroupnsMode')!='private' or h.get('IpcMode')!='private' or h.get('SecurityOpt') not in (['no-new-privileges'],['no-new-privileges=true'])
  or h.get('Tmpfs')!={'/tmp':'rw,nosuid,nodev,mode=1777,size=64m','/urbit':'ro,nosuid,nodev,noexec,mode=0555,size=64k'}):raise RuntimeError('gateway resource contract differs')
 actual={m.get('Source'):(m.get('Destination'),m.get('RW')) for m in row.get('Mounts',[]) if m.get('Type')=='bind'}
 if actual!=mounts or sum(m.get('Type')=='bind' for m in row['Mounts'])!=len(mounts):raise RuntimeError('gateway mount contract differs')
 if any(m.get('Type') not in ('bind','tmpfs') or m.get('Type')=='tmpfs' and (m.get('Destination') not in ('/tmp','/urbit') or m.get('Destination')=='/urbit' and m.get('RW') is not False) for m in row['Mounts']):raise RuntimeError('unexpected gateway filesystem')
 environment=[v for v in c.get('Env',[]) if '=' in v and v.split('=',1)[0] in variables]
 if sorted(environment)!=sorted(k+'='+v for k,v in variables.items()) or c.get('WorkingDir')!=variables['HOME'].rsplit('/',1)[0]:raise RuntimeError('gateway process environment differs')
 if state.get('OOMKilled') is not False or state.get('Paused') or state.get('Restarting') or state.get('Dead') or state.get('Error'):raise RuntimeError('gateway process state differs')
 if publication is not None:
  publication.check_gateway_configuration(row,container)
  if state.get('Running'):publication.check_gateway_network(row,container)
 return state

def retained_mounts(work,chain,proof,selected):
 mounts={str(work):('/resume',True),str(chain):('/chain',True),str(proof):('/initial-result.json',False),str(selected):('/selection.json',False)}
 mounts.update({str(INPUT/n):(target,False) for n,target in [('node','/node'),('tools','/tools'),('compiled','/compiled'),('deployment','/deployment'),('vere','/vere'),('src','/gateway')]})
 return mounts

def exercise(environment,initial,peer_resume=None,publication_ports=None):
 require_guest()
 if publication_ports is not None and (type(publication_ports) is not tuple or len(publication_ports)!=2 or any(type(p) is not int or not 1024<=p<=65535 for p in publication_ports) or publication_ports[0]==publication_ports[1] or peer_resume is None):raise ValueError('explicit two-port peer publication required')
 if initial.get('ok') is not True or not re.fullmatch('[0-9a-f]{32}',initial.get('project','')):raise RuntimeError('completed initial gateway required')
 project=initial['project'];previous=identity(initial['anchor']);operation=uuid.uuid4().hex
 oldroot=Path('/home/lab')/('omabit-private-chain-'+project);root_identity=directory(oldroot)
 chain=oldroot/'chain';chain_identity=directory(chain)
 proof=oldroot/'result.json';proof_sha=digest(proof)
 engine=Engine(environment)
 oldmounts={str(oldroot):('/lab',True)}
 oldmounts.update({str(INPUT/n):(target,False) for n,target in [('node','/node'),('tools','/tools'),('source','/source'),('compiled','/compiled'),('deployment','/deployment'),('vere','/vere'),('src/docker_gateway.cjs','/gateway.cjs')]})
 oldvars={'HOME':'/lab/home','OMABIT_PRIVATE_CHAIN':'private-chain-docker-gateway-v1','LIBC':'glibc','PREBUILDS_ONLY':'1'}
 def oldcheck():
  state=check_gateway(engine.inspect(previous),previous,project,'omabit-private-'+project+'-gateway',oldmounts,['--library-path','/node/usr/lib','/node/usr/bin/node','/gateway.cjs'],oldvars)
  if state.get('Running') is not False or state.get('Status')!='exited' or state.get('ExitCode')!=0:raise RuntimeError('prior gateway is not cleanly stopped')
  if directory(oldroot)!=root_identity or directory(chain)!=chain_identity or digest(proof)!=proof_sha:raise RuntimeError('retained project input changed')
 oldcheck()
 work=Path('/home/lab')/('omabit-private-resume-'+project+'-'+operation)
 work.mkdir(mode=0o700);os.chown(work,1000,1000);(work/'home').mkdir(mode=0o700);os.chown(work/'home',1000,1000)
 selected=work/'selection.json'
 selected.write_text(json.dumps({'version':1,'mode':MODE,'project':project,'operation':operation,'previous_gateway':previous,'initial_result_sha256':proof_sha,'foundation_sha256':digest(INPUT/'deployment/result.json'),'compiled_sha256':digest(INPUT/'compiled/compiler-output.json'),'guest_namespaces':{n:os.readlink('/proc/self/ns/'+n) for n in ('net','pid','mnt')}})+'\n');selected.chmod(0o600);os.chown(selected,1000,1000)
 mounts=retained_mounts(work,chain,proof,selected)
 name='omabit-private-'+project+'-gateway-resume-'+operation
 variables={'HOME':'/resume/home','OMABIT_PRIVATE_CHAIN':MODE,'LIBC':'glibc','PREBUILDS_ONLY':'1'}
 command=['--library-path','/node/usr/lib','/node/usr/bin/node','/gateway/docker_gateway_resume.cjs']
 report={'version':1,'ok':False,'project':project,'operation':operation,'previous_gateway':previous,'chain_identity':list(chain_identity),'initial_result_sha256':proof_sha,'initialization_performed':False,'cleanup_errors':[]}
 container=None;publication=None
 def save():
  p=ROOT/'results/gateway-resume.json';q=p.with_suffix('.tmp')
  with q.open('w') as f:json.dump(report,f,indent=2);f.write('\n');f.flush();os.fsync(f.fileno())
  q.replace(p);fd=os.open(p.parent,os.O_DIRECTORY);os.fsync(fd);os.close(fd)
 def current():
  row=engine.inspect(container)
  state=check_gateway(row,container,project,name,mounts,command,variables,publication)
  if publication is not None and state.get('Running'):
   networks=json.loads(engine.execute(['network','inspect',publication.network]))
   if len(networks)!=1:raise RuntimeError('published bridge observation differs')
   publication.check_network(networks[0],container)
   selected={'network':networks[0],'gateway':{'Id':row['Id'],'Image':row['Image'],'Config':{k:row['Config'][k] for k in ('Image','Labels')},'HostConfig':{k:row['HostConfig'][k] for k in ('NetworkMode','PortBindings')},'NetworkSettings':{k:row['NetworkSettings'][k] for k in ('Ports','Networks')},'State':{k:row['State'][k] for k in ('Running','OOMKilled','Status')}}}
   report['publication'].setdefault('first_observation',selected);report['publication']['last_observation']=selected
  return state
 try:
  if publication_ports is not None:
   from docker_publication import Publication,network_create_command
   report['network_create_dispatched']=True;save()
   network=identity(engine.execute(network_create_command(project)).decode().strip())
   publication=Publication(project,network,*publication_ports)
   rows=json.loads(engine.execute(['network','inspect',network]))
   if len(rows)!=1:raise RuntimeError('created bridge observation differs')
   publication.check_network(rows[0]);report['publication']={'network':network,'name':publication.name,'ports':publication.ports,'loopback_only':True};save()
   variables['OMABIT_PRIVATE_CHAIN_NETWORK']='published-v1'
  args=['create','--name',name,'--label',LABEL+'='+project,'--network','none','--pull','never','--init','--read-only','--tmpfs','/tmp:rw,nosuid,nodev,mode=1777,size=64m','--tmpfs','/urbit:ro,nosuid,nodev,noexec,mode=0555,size=64k','--workdir','/resume','--user','0:0','--cap-drop','ALL','--security-opt','no-new-privileges','--cgroupns','private','--ipc','private','--restart','no','--memory','1536m','--memory-swap','1536m','--pids-limit','128','--cpus','2']
  if publication is not None:
   at=args.index('--network');args[at:at+2]=publication.gateway_network_arguments()
  for k,v in variables.items():args+=['--env',k+'='+v]
  for source,(target,writable) in mounts.items():
   if Path(source).resolve(strict=True)!=Path(source):raise RuntimeError('indirect selected input')
   args+=['--mount','type=bind,src='+source+',dst='+target+('' if writable else ',readonly')]
  args+=['--entrypoint',LOADER,IMAGE]+command
  report['create_dispatched']=True;save()
  container=identity(engine.execute(args).decode().strip());report['container']=container;save()
  if container==previous or current().get('Status')!='created':raise RuntimeError('fresh gateway not observed')
  oldcheck();report['start_dispatched']=True;save()
  if engine.execute(['start',container]).decode().strip()!=container:raise RuntimeError('gateway start acknowledgment differs')
  print('OMABIT_PRIVATE_CHAIN '+json.dumps({'stage':'retained-gateway-started'}),flush=True)
  deadline=time.monotonic()+180
  while not (work/'resume-ready.json').exists():
   if current().get('Running') is not True:raise RuntimeError('retained gateway exited before readiness')
   if time.monotonic()>deadline:raise RuntimeError('retained gateway readiness deadline')
   time.sleep(2)
  ready=json.loads((work/'resume-ready.json').read_text())
  if ready.get('mode')!=MODE or ready.get('project')!=project or ready.get('operation')!=operation or ready.get('previous_gateway')!=previous or ready.get('read_only_rpc') is not True or ready.get('initialization_performed') is not False:raise RuntimeError('retained readiness binding differs')
  report['ready']=ready;save();current();oldcheck()
  probe="""(async()=>{const send=async(method)=>{const r=await fetch('http://127.0.0.1:25432/rpc',{method:'POST',body:JSON.stringify({jsonrpc:'2.0',id:'probe',method,params:[]})});return {status:r.status,body:await r.json()};};const read=await send('eth_chainId'),tx=await send('eth_sendTransaction'),mine=await send('evm_mine');if(read.status!==200||read.body.result!=='0x539'||tx.status!==400||mine.status!==400)throw Error('read-only probe failed');process.stdout.write(JSON.stringify({read:true,transaction_refused:true,mining_refused:true}));})().catch(()=>{process.exitCode=1;});"""
  probe_result=json.loads(engine.execute(['exec',container]+NODE+['-e',probe],timeout=30))
  if probe_result!={'read':True,'transaction_refused':True,'mining_refused':True}:raise RuntimeError('read-only gateway probe differs')
  report['rpc_probe']=probe_result;save()
  if peer_resume is not None:
   options={'publication':publication} if publication is not None else {}
   report['peer_recovery']=peer_resume(environment,project,previous,container,operation,**options);save();current();oldcheck()
  report['stop_dispatched']=True;save()
  if engine.execute(['kill','--signal','TERM',container]).decode().strip()!=container:raise RuntimeError('owned gateway stop acknowledgment differs')
  code=engine.execute(['wait',container],timeout=75).decode().strip();state=current()
  if code!='0' or state.get('Running') is not False or state.get('ExitCode')!=0:raise RuntimeError('owned retained gateway stop failed')
  result=json.loads((work/'resume-result.json').read_text())
  if result.get('passed') is not True or result.get('project')!=project or result.get('operation')!=operation or result.get('previous_gateway')!=previous or result.get('signal_shutdown') is not True or result.get('initialization_performed') is not False or result.get('before')!=result.get('after') or result.get('before')!=ready.get('observation'):raise RuntimeError('retained state changed during gateway service')
  report['result']=result;oldcheck();report['ok']=True;save()
  print('OMABIT_PRIVATE_CHAIN '+json.dumps({'stage':'retained-gateway-stopped'}),flush=True)
 except Exception as error:
  report['error_type']=type(error).__name__;report['error_sha256']=hashlib.sha256(str(error).encode()).hexdigest()
 finally:
  if container:
   try:
    state=current()
    if state.get('Running'):
     report['cleanup_stop_dispatched']=True;save()
     engine.execute(['kill','--signal','TERM',container]);engine.execute(['wait',container],timeout=75)
    logs=engine.execute(['logs','--tail','200',container],capture_stderr=True);(ROOT/'results/gateway-resume.log').write_bytes(logs);report['diagnostic_logs_sha256']=hashlib.sha256(logs).hexdigest()
    state=current();report['final_exit']={k:state.get(k) for k in ('Running','ExitCode','OOMKilled')}
    if state.get('Running') is not False or state.get('ExitCode')!=0:raise RuntimeError('retained gateway cleanup not clean')
   except Exception as error:report['cleanup_errors'].append({'type':type(error).__name__,'sha256':hashlib.sha256(str(error).encode()).hexdigest()})
  report['ok']=report['ok'] and not report['cleanup_errors'];save()
 return report
