"""Direct Eyre/Gall recovery on two newly keyed guest-only containers."""
import hashlib,ipaddress,json,os,re,subprocess,time,uuid
from pathlib import Path
from docker_runtime import Engine,Ship,DOCKER,identity,require_guest,receipt_name
from network_case import thread_poke,thread_receipt
from state_case import Recovery,parse_sse,matching_state
ROOT=Path('/opt/omabit-lab')


class Transport:
 def __init__(self,engine,specs):self.engine=engine;self.specs=specs;self.ports={};self.admin={}
 def call(self,ship,path,data=None,content=None,accept=None,admin=False,method=None,auth=None,login=False,timeout=90):
  require_guest();spec=self.specs[ship];self.engine.check_anchor(spec)
  if not isinstance(path,str) or not path.startswith('/') or '\n' in path or '\r' in path:raise ValueError('private path required')
  if method not in (None,'PUT','DELETE'):raise ValueError('unsupported private HTTP method')
  port=self.admin[ship] if admin else 18081+spec.number
  args=['/usr/bin/curl','--disable','--noproxy','*','--proxy','','--silent','--show-error','--max-time',str(timeout),'--max-filesize','65536','--write-out','\n%{http_code}']
  if login:args+=['--cookie-jar','/tmp/omabit-'+ship+'-cookie']
  elif not admin:
   other=auth or ship
   if other not in ('zod','nec'):raise ValueError('fixture cookie owner required')
   args+=['--cookie','/tmp/omabit-'+other+'-cookie']
  if method:args+=['--request',method]
  if content:args+=['--header','Content-Type: '+content]
  if accept:args+=['--header','Accept: '+accept]
  if data is not None:args+=['--data-urlencode' if login else '--data-binary','password@-' if login else '@-']
  args+=['http://127.0.0.1:'+str(port)+path]
  q=subprocess.run(DOCKER+['exec','-i',spec.anchor,'/bin/sh','-c','umask 077; exec "$@"','private-chain-curl',*args],input=data,capture_output=True,env=self.engine.environment,timeout=timeout+15)
  if q.returncode or len(q.stdout)>65536 or len(q.stderr)>8192:raise RuntimeError('private HTTP request failed; curl_exit='+str(q.returncode))
  body,separator,status=q.stdout.rpartition(b'\n')
  if not separator or not re.fullmatch(b'[0-9]{3}',status):raise RuntimeError('private HTTP status missing')
  return int(status),body
 def events(self,channel,cursor,expected,initial):
  require_guest();spec=self.specs['nec'];self.engine.check_anchor(spec)
  if not re.fullmatch('/~/channel/omabit-state-[0-9a-f]{32}',channel):raise ValueError('fixture channel required')
  rows=[];ack=not initial;fact=False;deadline=time.monotonic()+60
  while time.monotonic()<deadline:
   args=['exec',spec.anchor,'/usr/bin/curl','--disable','--noproxy','*','--proxy','','--silent','--show-error','--no-buffer','--max-time','3','--max-filesize','65536','--cookie','/tmp/omabit-nec-cookie','--header','Accept: text/event-stream','--write-out','\n%{http_code}']
   if cursor is not None:
    if type(cursor) is not int or cursor<0:raise ValueError('nonnegative cursor required')
    args+=['--header','Last-Event-ID: '+str(cursor)]
   args+=['http://127.0.0.1:18082'+channel]
   q=subprocess.run(DOCKER+args,capture_output=True,env=self.engine.environment,timeout=15)
   if q.returncode not in (0,28) or len(q.stdout)>65536 or len(q.stderr)>8192:raise RuntimeError('private SSE connection failed')
   body,separator,status=q.stdout.rpartition(b'\n')
   if not separator or status!=b'200':raise RuntimeError('private SSE reconnect refused')
   for event_id,event in parse_sse(body.splitlines(keepends=True)):
    rows.append((event_id,event));cursor=event_id;fact=matching_state(event,expected) or fact
    ack=ack or (event.get('response')=='subscribe' and event.get('id')==1 and event.get('ok')=='ok')
   if len(rows)>100:raise RuntimeError('private SSE event bound')
   if fact and ack:return rows
  raise RuntimeError('private SSE observation deadline')


def peer_runtime(environment,project,anchor,gateway_port,publication=None):
 if publication is None:engine=Engine(environment);ship_type=Ship
 else:
  from docker_publication import Publication,PublishedEngine,PublishedShip
  if not isinstance(publication,Publication) or publication.project!=project:raise ValueError('initial peer publication belongs to another project')
  engine=PublishedEngine(environment,publication);ship_type=PublishedShip
 specs={s:ship_type(project,s,identity(anchor),gateway_port) for s in ('zod','nec')}
 return engine,specs,ship_type

def check_peer_network(routes,devices,publication=None,network=None,anchor=None):
 names=[x.split(':')[0].strip() for x in devices.splitlines() if ':' in x]
 lines=routes.strip().splitlines()
 if publication is None:
  if len(lines)!=1 or names!=['lo']:raise RuntimeError('non-loopback project network')
  return
 publication.check_network(network,anchor)
 if sorted(names)!=['eth0','lo'] or len(lines)!=3:raise RuntimeError('initial bridge interfaces/routes differ')
 config=network['IPAM']['Config'][0];subnet=ipaddress.ip_network(config['Subnet'])
 def little(value):return ipaddress.ip_address(value).packed[::-1].hex().upper()
 expected={('00000000',little(config['Gateway']),'00000000',3),(little(subnet.network_address),'00000000',little(subnet.netmask),1)}
 seen=set()
 for line in lines[1:]:
  fields=line.split()
  if len(fields)!=11 or fields[0]!='eth0' or any(not re.fullmatch('[0-9A-Fa-f]{8}',fields[n]) for n in (1,2,7)) or not re.fullmatch('[0-9A-Fa-f]{4}',fields[3]):raise RuntimeError('initial bridge route shape differs')
  seen.add((fields[1].upper(),fields[2].upper(),fields[7].upper(),int(fields[3],16)))
 if seen!=expected:raise RuntimeError('initial bridge routes do not match owned network')

def exercise(environment,project,anchor,gateway_port=25432,checkpoint=None,*,label=None,publication=None):
 require_guest();receipt_name("peers",label);engine,specs,ship_type=peer_runtime(environment,project,anchor,gateway_port,publication)
 transport=Transport(engine,specs);root=specs['zod'].root;result={'ok':False,'project':project,'ships':{},'attempts':[],'stop_records':[],'cleanup_errors':[]};recovery=None
 def save():
  path=ROOT/'results'/receipt_name('peers',label);temp=path.with_suffix('.tmp')
  with temp.open('w') as stream:json.dump(result,stream,indent=2);stream.write('\n');stream.flush();os.fsync(stream.fileno())
  temp.replace(path);fd=os.open(path.parent,os.O_DIRECTORY);os.fsync(fd);os.close(fd)
 def emit(stage,**values):print('OMABIT_PRIVATE_CHAIN '+json.dumps(dict(stage=stage,**values)),flush=True)
 def namespace(container):return engine.execute(['exec',container,'/usr/bin/readlink','/proc/self/ns/net']).decode().strip()
 def ready(ship):
  spec=specs[ship];item=result['ships'][ship];deadline=time.monotonic()+(240 if spec.resumed else 900);last_notice=0
  while time.monotonic()<deadline:
   row=spec.check(engine.inspect(item['container']),item['container'])
   if not row['State']['Running'] or row['State']['OOMKilled']:raise RuntimeError('keyed container exited before readiness')
   try:status,body=transport.call(ship,'/~/host',timeout=3)
   except (RuntimeError,subprocess.TimeoutExpired):status,body=0,b''
   if status==200 and body.strip()==('~'+ship).encode():break
   if time.monotonic()-last_notice>30:emit('booting',ship=ship,resumed=spec.resumed);last_notice=time.monotonic()
   time.sleep(3)
  else:raise RuntimeError('fresh keyed container readiness deadline')
  inventory=(spec.pier/'.http.ports').read_text()
  if len(inventory)>1024:raise RuntimeError('port inventory bound')
  ports=[int(row[0]) for row in (line.split() for line in inventory.splitlines()) if row[1:]==['insecure','loopback']]
  if len(ports)!=1:raise RuntimeError('private administrative port missing')
  transport.admin[ship]=ports[0]
  status,body=transport.call(ship,'/',json.dumps({'source':{'dojo':'our'},'sink':{'stdout':None}}).encode(),content='application/json',admin=True)
  if status!=200 or json.loads(body).strip()!='~'+ship:raise RuntimeError('administrative identity mismatch')
  if namespace(item['container'])!=result['network_namespace']:raise RuntimeError('keyed ship left private namespace')
  item.update(admin=ports[0],port=18081+spec.number,ready=True);save();emit('ready',ship=ship,resumed=spec.resumed)
 def login(ship):
  status,body=transport.call(ship,'/',json.dumps({'source':{'dojo':'+code'},'sink':{'stdout':None}}).encode(),content='application/json',admin=True)
  if status!=200:raise RuntimeError('private code unavailable')
  code=json.loads(body).strip().removeprefix('~')
  if not re.fullmatch('[a-z]{6}(?:-[a-z]{6}){3}',code):raise RuntimeError('private code shape refused')
  status,_=transport.call(ship,'/~/login',code.encode(),login=True);del code,body
  if not 200<=status<300:raise RuntimeError('private login refused')
  status,_=transport.call(ship,'/~/scry/hood/kiln/pikes.json')
  if status!=200:raise RuntimeError('authenticated inventory refused')
 def watch(ship):
  peer='nec' if ship=='zod' else 'zod';gateway='http://127.0.0.1:'+str(gateway_port)
  setup=[("+azimuth/watch '"+gateway+"/rpc' %local",'azimuth'),('&eth-watcher-poke [%clear /azimuth]','eth-watcher'),('[%refresh-rate ~s1]','azimuth'),('%resub','azimuth'),('&azimuth-poke [%kick ~]','azimuth')]
  commands=[];result['ships'][ship]['watcher_commands']=commands
  for source,app in setup:
   row={'source':source,'app':app,'dispatched':True};commands.append(row);save()
   status,body=transport.call(ship,'/',json.dumps({'source':{'dojo':source},'sink':{'app':app}}).encode(),content='application/json',admin=True)
   if status!=200 or json.loads(body).strip()!='>=':raise RuntimeError('watcher acknowledgment unavailable')
   row['acknowledged']=True;save()
  expression='.^(@ud %j /(scot %p our)/life/(scot %da now)/(scot %p ~'+peer+'))';deadline=time.monotonic()+180
  while time.monotonic()<deadline:
   status,body=transport.call(ship,'/',json.dumps({'source':{'dojo':expression},'sink':{'stdout':None}}).encode(),content='application/json',admin=True)
   if status==200 and json.loads(body).strip()=='1':result['ships'][ship]['peer_jael_life']=1;save();return
   time.sleep(2)
  raise RuntimeError('private peer key observation deadline')
 def poke(sender,receiver,phase,reject=False):
  nonce='fail' if reject else 'omabit-'+phase+'-'+uuid.uuid4().hex[:24];expected='omabit-receipt-'+uuid.uuid4().hex
  row={'sender':sender,'receiver':receiver,'phase':phase,'nonce':nonce,'expected_receipt':expected,'state':'pending'};result['attempts'].append(row);save()
  status,body=transport.call(sender,'/spider/base/ted-eval/khan-eval/noun',thread_poke(receiver,nonce,expected),content='application/x-urb-jam',accept='application/json' if reject else 'application/x-urb-jam')
  row.update(status=status,bytes=len(body));save();row['receipt']=thread_receipt(status,body,expected,reject);row['state']='rejected' if reject else 'acknowledged';save();emit('message',sender=sender,receiver=receiver,phase=phase)
 def stop(ship):
  spec=specs[ship];container=result['ships'][ship]['container']
  record=next((row for row in result['stop_records'] if row.get('container')==container),None)
  if record is None:record={};result['stop_records'].append(record)
  def exit_ship():
   spec.check(engine.inspect(container),container)
   status,body=transport.call(ship,'/',json.dumps({'source':{'dojo':'our'},'sink':{'stdout':None}}).encode(),content='application/json',admin=True)
   if status!=200 or json.loads(body).strip()!='~'+ship:raise RuntimeError('shutdown endpoint identity changed')
   status,body=transport.call(ship,'/',json.dumps({'source':{'dojo':'+hood/exit'},'sink':{'app':'hood'}}).encode(),content='application/json',admin=True,timeout=20)
   return status==200 and json.loads(body).strip()=='>='
  engine.stop(spec,container,record,save,exit_ship);emit('stopped',ship=ship,mode=record['mode'])
 try:
  result['network_namespace']=namespace(anchor)
  if result['network_namespace']==os.readlink('/proc/self/ns/net'):raise RuntimeError('anchor shares guest network')
  routes=engine.execute(['exec',anchor,'/bin/cat','/proc/net/route']).decode();devices=engine.execute(['exec',anchor,'/bin/cat','/proc/net/dev']).decode()
  network=None
  if publication is not None:
   networks=json.loads(engine.execute(['network','inspect',publication.network]))
   if len(networks)!=1:raise RuntimeError('initial bridge observation count differs')
   network=networks[0]
  check_peer_network(routes,devices,publication,network,anchor)
  result.update(ipv4_routes=routes,network_devices=devices)
  for ship in ('zod','nec'):
   spec=specs[ship];spec.pier_parent.mkdir(mode=0o700,parents=True);os.chown(spec.pier_parent,1000,1000)
   # The gateway creates fresh keys as the rootless owner; ownership is verified
   # by the guest before the read-only key mount is handed to a ship container.
   if spec.key.stat().st_uid!=1000 or spec.key.stat().st_mode&0o077:raise RuntimeError('private key ownership/mode differs')
   container=engine.create(spec);result['ships'][ship]={'container':container,'pier':str(spec.pier),'ready':False};save()
   engine.start(spec,container);ready(ship);login(ship);watch(ship)
  for receiver,other in (('zod','nec'),('nec','zod')):
   status,_=transport.call(receiver,'/~/scry/hood/kiln/pikes.json',auth=other)
   if status not in (401,403):raise RuntimeError('cross-ship cookie accepted')
  result['cross_ship_sessions_refused']=True
  for sender,receiver,phase,reject in [('zod','zod','local-check',False),('nec','nec','local-check',False),('zod','nec','first',False),('nec','zod','first',False),('zod','nec','nack',True)]:poke(sender,receiver,phase,reject)
  recovery=Recovery(transport.call,transport.events,save,emit);result['application_state']=recovery.report
  recovery.install(specs['nec'].pier,ROOT/'omabit-state.hoon');recovery.before_restart()
  pier=specs['nec'].pier;before=(pier.stat().st_dev,pier.stat().st_ino);old=result['ships']['nec']['container'];stop('nec')
  specs['nec']=ship_type(project,'nec',anchor,gateway_port,True);new=engine.create(specs['nec'])
  if new==old:raise RuntimeError('container identity reused')
  result['restart']={'original':old,'replacement':new,'pier_device':before[0],'pier_inode':before[1]};result['ships']['nec']['container']=new;save()
  engine.start(specs['nec'],new);ready('nec')
  if before!=(pier.stat().st_dev,pier.stat().st_ino) or engine.inspect(old)['State']['Running']:raise RuntimeError('pier or stopped ownership changed')
  poke('zod','nec','after-restart');poke('nec','zod','after-restart');recovery.after_restart()
  result.update(transport_passed=True,restart_same_pier=True,restart_kept_sessions=True)
  if checkpoint is not None:checkpoint(engine,specs,transport,recovery,result)
 except Exception as error:result['error']=type(error).__name__+': '+str(error);emit('failure',error=result['error'])
 finally:
  if recovery and recovery.channel:
   try:recovery.close()
   except Exception as error:result['cleanup_errors'].append('subscription:'+type(error).__name__)
  for ship in reversed(list(result['ships'])):
   try:stop(ship)
   except Exception as error:result['cleanup_errors'].append('stop:'+ship+':'+type(error).__name__)
  result['ok']=result.get('transport_passed',False) and not result['cleanup_errors'] and 'error' not in result;save()
 return result
