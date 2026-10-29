"""Real Theme/DMs services with private desktop failure/recovery characterization."""
from pathlib import Path
import hashlib,http.server,json,os,signal,subprocess,threading,time,traceback
from connected_fixture import Hub,Handler,begin

def profile(name):
 return {'schemaVersion':1,'updateId':{'stable':'a','dev':'b','edge':'c'}[name]*32,'deviceId':'d'*32,
 'theme':'fixture-'+name,'font':'Stable Mono' if name=='stable' else 'Candidate Mono',
 'colors':{'mode':'dark','background':{'stable':'#112233','dev':'#445566','edge':'#778899'}[name],'foreground':'#EEEEEE','blue':'#123ABC','green':'#33CC66','lighter_background':'#223344','muted':'#888888','selection':'#334455','red':'#EE3333'},
 'shell':{'font.base-size':'12' if name=='stable' else '16'},'windows':{'decoration:rounding':{'int':{'stable':8,'dev':14,'edge':22}[name]}},'animations':{'curves':[],'rules':[]},'files':{'btop.theme':'# '+name+' fixture\n'}}
class AppearanceHub(Hub):
 def __init__(self):
  http.server.ThreadingHTTPServer.__init__(self,('127.0.0.1',0),AppearanceHandler)
  self.errors=[];self.idle_disconnects=[]
  self.url='http://127.0.0.1:'+str(self.server_port);self.cond=threading.Condition();self.channels={};self.sequence=0;self.requests=[];self.pokes=[];self.confirmed=False;self.stopping=False;self.talon={};self.profiles={n:profile(n) for n in ('stable','dev','edge')}
  self.thread=threading.Thread(target=self.serve_forever,daemon=True);self.thread.start()
 def handle_error(self,request,client_address):
  failure=traceback.format_exc()
  with self.cond:
   self.errors.append(failure)
   Path('/lab/http-server-errors.json').write_text(json.dumps(self.errors,indent=2)+'\n')
  super().handle_error(request,client_address)
class AppearanceHandler(Handler):
 def handle_one_request(self):
  # A peer can reset a keep-alive socket after its process is deliberately
  # stopped. Ignore only resets before parsing another request; once parsing
  # begins, all handler failures must still reach AppearanceHub.handle_error.
  self._request_line_received=False
  try:return super().handle_one_request()
  except ConnectionResetError as error:
   if self._request_line_received:raise
   self.close_connection=True
   with self.server.cond:
    self.server.idle_disconnects.append({'stage':'before-request-parse','errno':error.errno})
    Path('/lab/http-idle-disconnects.json').write_text(json.dumps(self.server.idle_disconnects,indent=2)+'\n')
 def parse_request(self):
  self._request_line_received=True
  return super().parse_request()
 def do_POST(self):
  with self.server.cond:self.server.requests.append({'action':'post','path':self.path})
  return super().do_POST()
 def do_GET(self):
  with self.server.cond:self.server.requests.append({'action':'get','owner':self.owner(),'path':self.path})
  if '/~/scry/settings/desk/omarchy-urbit-theme.json'==self.path:
   return self.reply({'desk':{'appearance':{'current':json.dumps(self.server.profiles[self.owner().split('-')[0]])}}})
  if '/~/scry/settings/desk/talon.json'==self.path:return self.reply(self.server.talon.get(self.owner(),{'desk':{}}))
  if getattr(self.server,'complete_visual_reads',False) and self.path=='/~/scry/groups-ui/v4/heads.json':return self.reply({'dms':[]})
  return super().do_GET()
 def do_PUT(self):
  s=self.server;owner=self.owner();raw=self.rfile.read(int(self.headers.get('Content-Length','0')));assert len(raw)<65536
  with s.cond:
   if (getattr(s,'block_cleanup',False) and owner.startswith('stable-') or owner in getattr(s,'blocked_cleanup_owners',[])) and any(m['action']=='delete' for m in json.loads(raw)):
    s.requests.append({'owner':owner,'action':'delete','channel':self.path,'refused':True})
    return self.reply(status=503)
   c=s.channels.setdefault(self.path,{'owner':owner,'frames':[],'subscriptions':0,'active':0,'deleted':False,'delivered':0});assert c['owner']==owner
   for m in json.loads(raw):
    action=m['action'];assert action in ('subscribe','ack','delete','poke')
    s.requests.append({'owner':owner,'action':action,'channel':self.path});assert len(s.requests)<4000
    if action=='poke':
     if getattr(s,'uncertain_mode',False) and owner=='stable-dms':
      assert (m['app'],m['mark'])==('chat','chat-dm-action-2')
      assert not any(p.get('uncertain') and p['owner']==owner for p in s.pokes),'Uncertain DM replayed'
      s.pokes.append({'owner':owner,'mark':m['mark'],'uncertain':True});continue
     put=m['json']['put-entry'];assert owner.endswith('-theme') and m['app']=='settings' and m['mark']=='settings-event'
     assert put['desk']=='talon' and put['bucket-key']=='ui-prefs','stale desktop publication to the ship'
     s.talon.setdefault(owner,{'desk':{'ui-prefs':{}}})['desk']['ui-prefs'][put['entry-key']]=put['value']
     if getattr(s,'uncertain_mode',False) and owner=='stable-theme' and put['entry-key']=='themes':
      assert not any(p.get('uncertain') and p['owner']==owner for p in s.pokes),'Uncertain Theme write replayed'
      s.pokes.append({'owner':owner,'desk':put['desk'],'entry':put['entry-key'],'uncertain':True});continue
     s.pokes.append({'owner':owner,'desk':put['desk'],'entry':put['entry-key']});s.sequence+=1;c['frames'].append((s.sequence,{'id':m['id'],'response':'poke','ok':'ok'}))
    elif action=='subscribe':
     s.sequence+=1;c['subscriptions']+=1;c['frames'].append((s.sequence,{'id':m['id'],'response':'subscribe','ok':'ok'}))
    elif action=='delete':c['deleted']=True
   s.cond.notify_all()
  self.reply()
def prepare(root):
 h=root/'home';current=h/'.local/state/omarchy/current';(current/'theme').mkdir(parents=True)
 value={'font':'Fixture Mono','rounding':0,'colors':profile('edge')['colors'],'shell':{'font':{'base-size':'10'}},'shellApplied':{}}
 (current/'theme/colors.toml').write_text(''.join(k+'='+json.dumps(v)+'\n' for k,v in value['colors'].items()))
 (root/'desktop.json').write_text(json.dumps(value));(root/'fault.json').write_text('{}');(current/'theme.name').write_text('fixture-edge\n');(current/'theme/shell.toml').write_text('[font]\nbase-size="10"\n')
 config=h/'.config/hypr/hyprland.lua';config.parent.mkdir(parents=True);config.write_text('-- private rollback fixture; retain this text\n')
 for name in ('stable','dev','edge'):
  theme=h/'.config/omarchy/themes'/('fixture-'+name);theme.mkdir(parents=True);(theme/'btop.theme').write_text('# source '+name+'\n');(theme/'neovim.lua').write_text('-- local ancillary '+name+'\n')
 for name in ('omarchy','omarchy-shell','hyprctl','fc-list'):
  p=root/'bin'/name;p.write_text('#!/usr/bin/python3\n'+(root/'appearance_desktop.py').read_text());p.chmod(0o700)
def sha(p):return hashlib.sha256(p.read_bytes()).hexdigest()
def run(root):
 assert os.environ.get('OMABIT_LIFECYCLE_APPEARANCE')=='1' and Path.home()==root/'home' and not Path('/home').exists()
 prepare(root);keyring,hub=begin(root,AppearanceHub)
 offline=os.environ.get('OMABIT_LIFECYCLE_APPEARANCE_OFFLINE')=='1';hub_closed=False
 report={'passed':False,'real_helpers':True,'real_theme_service':True,'native_loader_methods':True,'synthetic_eyre':True,'real_private_keyring':True,'actual_lua':True,'inert_desktop_commands':os.environ.get('OMABIT_NATIVE_THEME_STAGING')!='1','inert_desktop_controls':True,'native_theme_staging':os.environ.get('OMABIT_NATIVE_THEME_STAGING')=='1','actual_compositor':False,'automatic_rollback':False,'offline_mode':offline,'cases':[],'transitions':[]}
 def save():(root/'result.json').write_text(json.dumps(report,indent=2)+'\n')
 save()
 with (root/'shell.log').open('wb') as out:
  proc=subprocess.Popen(['/usr/bin/quickshell','--no-color','--path','/lab/shell.qml'],stdout=out,stderr=subprocess.STDOUT,start_new_session=True)
  def call(method,*args,require=True):
   q=subprocess.run(['/usr/bin/quickshell','ipc','--pid',str(proc.pid),'call','--','realPluginLifecycle',method,*args],capture_output=True,text=True,timeout=4)
   if require:assert q.returncode==0,(method,q.stderr[-500:])
   return q.stdout.strip()
  def state():return json.loads(call('state'))
  def wait(fn,timeout=18):
   deadline=time.monotonic()+timeout
   while time.monotonic()<deadline:
    assert proc.poll() is None,'private shell exited'
    v=fn()
    if v:return v
    time.sleep(.075)
   raise AssertionError('deadline: '+str(state()))
  def children(pid):
   try:ids=[int(x) for x in Path('/proc/'+str(pid)+'/task/'+str(pid)+'/children').read_text().split()]
   except FileNotFoundError:return []
   return ids+[v for child in ids for v in children(child)]
  def identity(pid):
   try:
    p=Path('/proc')/str(pid);stats=(p/'stat').read_text().split(') ',1)[1].split();args=(p/'cmdline').read_bytes().split(b'\0');script=next((x.decode() for x in args if x.startswith(b'/lab/artifacts/') and x.endswith(b'.py')),'')
    return {'pid':pid,'start_ticks':stats[19],'state':stats[0],'script':script}
   except (FileNotFoundError,ProcessLookupError):return None
  def helpers():return [v for pid in children(proc.pid) if (v:=identity(pid)) and v['state']!='Z' and v['script'].endswith(('/bridge.py','/client/sync.py'))]
  def gone(old):
   now=identity(old['pid']);return not now or now['state']=='Z' or now['start_ticks']!=old['start_ticks']
  def service():return next(iter(state()['services'].values()))
  def disk(name):return root/'state'/('omarchy-urbit-theme.qualification-'+name)/'desktop.json'
  def healthy(name):
   s=state();h=helpers()
   if len(s['services'])!=1 or len(s['panels'])!=1 or len(h)!=2:return False
   t=next(iter(s['services'].values()));p=next(iter(s['panels'].values()))
   return t['desktop']['status']=='Following shared appearance' and not t['desktop']['error'] and not t['desktop'].get('bridgeBusy') and p['ship']=='~zod' and len(hub.active())==2 and {r['owner'] for r in hub.active()}=={name+'-theme',name+'-dms'}
  def visual(name):
   actual=json.loads((root/'desktop.json').read_text());expected=profile(name)
   assert actual['font']==expected['font'] and actual['rounding']==expected['windows']['decoration:rounding']['int'] and actual['colors']==expected['colors']
   assert actual['shellApplied']=={'font':{'base-size':expected['shell']['font.base-size']}}
   assert (root/'home/.local/state/omarchy/current/theme.name').read_text().strip()==expected['theme']
   assert (root/'home/.local/state/omarchy-urbit-theme-ownership/owner').read_text().splitlines()[1]=='omarchy-urbit-theme.qualification-'+name
   assert (root/'home/.local/state/omarchy/current/theme/btop.theme').read_text()==expected['files']['btop.theme']
   return actual
  def drain(name):
   old=helpers();assert len(old)==2
   # A failed apply remains durable through shutdown; never clear it to make a
   # busy flag look ready. Retained user intent is checked again after exit.
   prior=json.loads(disk(name).read_text());pending=prior['applying']
   wait(lambda:call('drainDms')=='requested')
   wait(lambda:next(iter(state()['panels'].values()))['transportState'].get('closed'),35)
   wait(lambda:call('drainTheme')=='requested')
   receipt=wait(lambda:service()['transportState'] if service()['transportState'].get('closed') else None,75)
   wait(lambda:all(gone(p) for p in old) and not hub.active());assert all(x['deleted'] for x in hub.snapshot())
   assert json.loads(disk(name).read_text())['applying']==pending
   call('select','none');wait(lambda:state()=={'services':{},'panels':{}} and not children(proc.pid))
   report['transitions'].append({'profile':name,'receipt':receipt,'all_helpers_exited':True,'channels_deleted':True,'pending_apply_preserved':True});save()
  def local_helper(*args, killed=False):
   # Run the actual CLI entrypoint. The audit tripwire forbids network and
   # credential tools even if a future refactor accidentally adds a dependency.
   script=root/'artifacts/stable/theme/client/sync.py'
   guard="""import os,runpy,sys
from pathlib import Path
script=sys.argv[1];sys.argv=sys.argv[1:];sys.path.insert(0,str(Path(script).parent))
def audit(event,args):
    if event in ('socket.connect','socket.getaddrinfo','socket.bind') or event=='subprocess.Popen' and Path(args[0]).name in ('secret-tool','busctl'):
        Path('/lab/offline-forbidden-access').write_text(event)
        raise RuntimeError('Offline restore attempted network or credentials')
sys.addaudithook(audit)
runpy.run_path(script,run_name='__main__')
"""
   env=dict(os.environ);env.pop('DBUS_SESSION_BUS_ADDRESS',None)
   result=subprocess.run(['/usr/bin/python3','-B','-c',guard,str(script),*args],stdin=subprocess.DEVNULL,capture_output=True,text=True,timeout=15,env=env)
   assert not (root/'offline-forbidden-access').exists()
   lines=result.stdout.splitlines()
   if killed:
    assert result.returncode==-signal.SIGKILL and not lines
    value={'terminated_after_effect':True}
   else:
    assert len(lines)==1,(args,result.stderr[-500:])
    value=json.loads(lines[0])
   report.setdefault('local_commands',[]).append({'command':list(args),'exit':result.returncode,'result':value});save()
   return result.returncode,value
  def offline_restore(baseline):
   nonlocal hub_closed
   # Seed retained unknown publication intent only after the exact helper exits.
   # It must survive checkpoint, failed candidate, failed restore and retry bytewise.
   state=json.loads(disk('stable').read_text());intent=profile('stable');intent['updateId']='a'*32
   state.update(pending=intent,attempted=True,baseId='b'*32);disk('stable').write_text(json.dumps(state))
   original=sha(disk('stable'))
   code,value=local_helper('checkpoint-appearance');assert code==0
   checkpoint=value['appearanceCheckpoint']['id'];assert len(checkpoint)==64 and not value['collectionReady']
   (root/'fault.json').write_text(json.dumps({'profile':'dev','stage':'font'}))
   call('select','dev')
   def failed():
    snap=json.loads(call('state'))
    return len(snap['services'])==1 and len(helpers())==2 and next(iter(snap['services'].values()))['desktop']['status']=='Needs attention' and '73' in next(iter(snap['services'].values()))['desktop']['error']
   wait(failed);assert json.loads(disk('dev').read_text())['applying']==profile('dev');drain('dev')
   candidate=sha(disk('dev'));assert sha(disk('stable'))==original
   hub.close();hub_closed=True;keyring.terminate();assert keyring.wait(timeout=4)==0
   before=list(hub.requests)
   theme=root/'home/.config/omarchy/themes/fixture-stable';theme.rename(theme.with_name('retired-fixture-stable'))
   generated=root/'home/.config/omarchy/themes/urbit-synced.qualification-stable/neovim.lua';generated.write_text('-- mutable source changed after checkpoint\n')
   request='c'*32
   (root/'fault.json').write_text(json.dumps({'profile':'stable','stage':'shell'}))
   code,value=local_helper('restore-appearance',checkpoint,request);assert code==1 and value['status']=='Needs attention'
   receipt=root/'home/.local/state/omarchy-urbit-theme.qualification-stable/restore-receipts'/(request+'.json')
   assert json.loads(receipt.read_text())['phase']=='applying'
   code,value=local_helper('serve');assert code==1 and 'restoration is incomplete' in value['error']
   (root/'fault.json').write_text('{}')
   code,value=local_helper('restore-appearance',checkpoint,request);assert code==0 and value['appearanceRestoration']['phase']=='verified' and not value['collectionReady']
   assert visual('stable')==baseline and (root/'home/.local/state/omarchy/current/theme/neovim.lua').read_text()=='-- local ancillary stable\n'
   effects=sha(root/'desktop-events.jsonl')
   code,value=local_helper('restore-appearance',checkpoint,request);assert code==0 and sha(root/'desktop-events.jsonl')==effects
   interrupted='d'*32
   (root/'fault.json').write_text(json.dumps({'profile':'stable','stage':'theme','kill_parent':True}))
   code,value=local_helper('restore-appearance',checkpoint,interrupted,killed=True)
   interrupted_receipt=receipt.with_name(interrupted+'.json')
   assert json.loads(interrupted_receipt.read_text())['phase']=='applying'
   assert (root/'home/.local/state/omarchy/current/theme.name').read_text().strip()=='urbit-restored-'+checkpoint
   (root/'fault.json').write_text('{}')
   code,value=local_helper('restore-appearance',checkpoint,interrupted);assert code==0 and value['appearanceRestoration']['phase']=='verified'
   assert visual('stable')==baseline
   assert hub.requests==before and sha(disk('stable'))==original and sha(disk('dev'))==candidate
   assert generated.read_text()=='-- mutable source changed after checkpoint\n'
   if os.environ.get('OMABIT_NATIVE_THEME_STAGING')=='1':report['native_staging_result']={'current_files':sorted(p.name for p in (root/'home/.local/state/omarchy/current/theme').iterdir()),'copied_scripts_and_templates':True,'headless':True,'native_app_retint':False}
   report['offline_restoration']={'checkpoint':checkpoint,'request':request,'failed_then_verified':True,'killed_helper_after_native_effect_then_verified':True,'interrupted_request':interrupted,'pending_startup_refused':True,'verified_replay_has_no_effects':True,'network_and_keyring_stopped':True,'audit_forbids_network_and_credentials':True,'stable_pending_and_candidate_applying_preserved':True,'source_theme_unavailable':True,'mutable_generated_source_ignored':True,'background_policy':'preserve-current','readback':visual('stable')};save()
  try:
   wait(lambda:call('state',require=False).startswith('{'))
   call('select','stable');wait(lambda:healthy('stable'));baseline=visual('stable');drain('stable')
   if offline:
    offline_restore(baseline)
   else:
    for index,stage in enumerate(('theme','windows','shell','font')):
     target='dev' if index%2==0 else 'edge'
     (root/'fault.json').write_text(json.dumps({'profile':target,'stage':stage}))
     call('select',target)
     def failed():
      s=state()
      if len(s['services'])!=1 or len(helpers())!=2:return False
      t=next(iter(s['services'].values()))
      return t['desktop']['status']=='Needs attention' and '73' in t['desktop']['error']
     wait(failed)
     partial=json.loads((root/'desktop.json').read_text());pending=json.loads(disk(target).read_text())['applying'];assert pending==profile(target)
     events=[json.loads(s) for s in (root/'desktop-events.jsonl').read_text().splitlines()]
     assert any(e['profile']==target and e['stage']==stage and e['failed_after_effect'] for e in events)
     report['cases'].append({'candidate':target,'failure_stage':stage,'partial':partial,'pending_apply_retained':True,'restored':False});save()
     drain(target);(root/'fault.json').write_text('{}');call('select','stable');wait(lambda:healthy('stable'))
     restored=visual('stable');assert restored==baseline
     report['cases'][-1].update(restored=True,readback=restored);save();drain('stable')
    # A failed restoration must remain unresolved even if the failing command
    # already changed the visible property. Only a later verified retry succeeds.
    call('select','dev');wait(lambda:healthy('dev'));visual('dev');drain('dev')
    (root/'fault.json').write_text(json.dumps({'profile':'stable','stage':'font'}))
    call('select','stable');wait(failed)
    time.sleep(.5)
    assert service()['desktop']['status']=='Needs attention'
    pending=json.loads(disk('stable').read_text())['applying'];assert pending==profile('stable')
    report['failed_restoration']={'reported_success':False,'pending_retained':True};save()
    drain('stable');assert report['transitions'][-1]['receipt']['pendingAppearance']
    (root/'fault.json').write_text('{}');call('select','stable');wait(lambda:healthy('stable'))
    assert visual('stable')==baseline and json.loads(disk('stable').read_text())['applying'] is None
    report['failed_restoration']['verified_retry']=True;save();drain('stable')
   assert not children(proc.pid) and not hub.active()
   assert call('stop')=='stopping' and proc.wait(timeout=4)==0
   report.update(passed=True,clean_shell_exit=0,final_helpers_exited=True,channels=hub.snapshot(),actions=hub.requests,talon_pokes=hub.pokes,no_theme_publications=True);save();print('PASS: offline checkpoint restoration preserves pending intent and verifies retry' if offline else 'PASS: four partial Theme failures restore the previous profile; failed restoration retains intent until verified retry')
  except Exception as error:
   report.update(error=type(error).__name__+': '+str(error),channels=hub.snapshot(),actions=hub.requests);save();raise
  finally:
   if not hub_closed:hub.close()
   keyring.terminate()
   try:keyring.wait(timeout=3)
   except subprocess.TimeoutExpired:keyring.kill();keyring.wait(timeout=3)
   if proc.poll() is None:
    os.killpg(proc.pid,signal.SIGTERM)
    try:proc.wait(timeout=3)
    except subprocess.TimeoutExpired:os.killpg(proc.pid,signal.SIGKILL);proc.wait(timeout=3)
