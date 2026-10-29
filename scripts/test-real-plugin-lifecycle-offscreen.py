#!/usr/bin/env python3
"""Fresh no-NIC Theme/DMs lifecycle, optionally connected and combined with Omabit.

--omabit-source requires connected drain mode. It adds real Omabit QML and the
native service loader with inert exact-argv runner/SSH readers and offscreen focus.
Theme/DMs helpers are unchanged. --collection-journal also exercises actual offline
appearance commands against inert private desktop controls; real runner effects
and the production desktop stay excluded.
"""
import argparse,hashlib,json,os,re,shutil,signal,socket,subprocess,time
from pathlib import Path
THEME=None
DMS=None
OMABIT=None
NATIVE=Path('/usr/share/omarchy/shell')
UI=['Panel','PanelController','BarWidget','BarIconButton','WidgetButton','OpticalGlyph','Button','TextField','ToggleSwitch','PanelHero','PanelSeparator','PanelSectionHeader','BorderSurface','BorderOverlay','PanelKeyCatcher','PanelActionButton','PanelToolTip']
FUNCTIONS=['ensureService','_syncServices','serviceKeepLoaded','unloadPluginServices','computePanelEntries']
QML='''
import QtQuick
import Quickshell
import Quickshell.Io
import "AuthServiceStore.js" as AuthServiceStore
ShellRoot {
 id:shell
 property var _services: ({})
 property var panels: ({})
 property var openPanelIds: ({})
 property var panelEntries: []
 property string omarchyPath: "/lab"
 property var registry: ({installedPlugins:MANIFESTS, enabled:{},isEnabled:function(id){return !!shell.registry.enabled[id]},entryPointUrl:function(m,kind){return m.entryPoints[kind]}})
 property var pluginRegistry: registry
 Item {id:serviceHost}
 function isAuthenticationService(m,id){return false}
 function pluginShellFor(m){return shell}
 function publicPluginManifest(m){return JSON.parse(JSON.stringify(m))}
 function pluginBarWidgetRegistryFor(m){return null}
 function pluginRegistryFor(m){return null}
 function serviceFor(id){return _services[id] || null}
 function registerPanelLoader(id,loader){var next=Object.assign({},panels);next[id]=loader;panels=next}
 function unregisterPanelLoader(id){var next=Object.assign({},panels);delete next[id];panels=next}
 function hide(id){}
 function snapshot(){
  var s=({}),p=({}),o=null;for(var id in _services){var a=_services[id];if(a.fixtureCollectionOmabit===true){o={profile:a.profileName,observing:a.observing,registrationsLoaded:a.registrationsLoaded,transportState:a.transportState};continue}s[id]={loaded:a.loaded,ships:a.account.ships.length,busy:a.busy,desktop:a.desktopState,transportState:a.transportState}}
  for(var id in panels){var a=panels[id].item;p[id]={loaded:!!a,ship:a?a.ship:"",status:a?a.status:"",busy:a?a.busy:false,pendingSend:a?a.pendingSend:-1,pendingText:a?a.pendingText:"",statusError:a?a.statusError:false,unresolved:a?a.unresolvedOutbound:[],outboundReadable:a?a.outboundReadable:false,unresolvedEffects:a?a.unresolvedEffects:[],transportState:a?a.transportState:{closed:false}}}
  var value={services:s,panels:p};if(o)value.omabit=o;return JSON.stringify(value)
 }
 function select(name){
  var e=({});if(name!=="none")for(var id in registry.installedPlugins)if(id.endsWith(".qualification-"+name))e[id]=true
  registry.enabled=e;_syncServices();panelEntries=computePanelEntries();return snapshot()
 }
 IpcHandler {
  target:"realPluginLifecycle"
  function select(name:string):string{return shell.select(name)}
  function state():string{return shell.snapshot()}
  function drainTheme():string{for(var id in shell._services)if(shell._services[id].fixtureCollectionOmabit!==true)return shell._services[id].stopTransport();return "no service"}
  function publishTheme(value:string):string{
   for(var id in shell._services){var a=shell._services[id];if(a.fixtureCollectionOmabit===true)continue
    if(a.busy||!a.loaded||a.account.ships.length!==1)return "busy"
    var row=a.account.ships[0],expected={id:row.id,url:row.url,ship:row.ship},palette=JSON.parse(value)
    var q=JSON.parse(JSON.stringify(a.queue));q.running="sync";q.run={action:"sync",generation:q.generation,expectedAccount:expected,palette:palette};a.queue=q
    a.launch("sync",{force:true,expectedAccount:expected,palette:palette});return "requested"
   }return "absent"
  }
  function probeThemeBusy():string{for(var id in shell._services)if(shell._services[id].fixtureCollectionOmabit!==true){shell._services[id].refresh();return shell._services[id].stopTransport()}return "absent"}
  function openOmabit():string{for(var id in shell._services)if(shell._services[id].fixtureCollectionOmabit===true){shell._services[id].fixtureOpenReaders();return "opened"}return "absent"}
  function drainOmabit():string{for(var id in shell._services)if(shell._services[id].fixtureCollectionOmabit===true)return shell._services[id].stopTransport();return "absent"}
  function drainDms():string{for(var id in shell.panels){shell.panels[id].item.stopTransport();return "requested"}return "no panel"}
  function send():string{for(var id in shell.panels){var a=shell.panels[id].item;return a.send("~nec","synthetic pending lifecycle probe","bar")}return "no panel"}
  function effects():string{for(var id in shell.panels){var a=shell.panels[id].item;a.command("accept",{ship:"~nec"});a.command("open",{ship:"~nec"});return "requested"}return "no panel"}
  function refresh():string{for(var id in shell.panels){shell.panels[id].item.command("refresh");return "requested"}return "no panel"}
  function stop():string{Qt.callLater(Qt.quit);return "stopping"}
 }
 METHODS
 PANELS
}
'''
OMABIT_FOCUS=r'''import QtQuick
Item {required property var targetWindow;signal failed();function activate(){targetWindow.visible=true}}
'''
OMABIT_READERS=r'''import QtQuick
import Quickshell.Io
Item {
 id:root
 property var service
 property int failedStarts:0
 ReaderLifecycle {id:failedTracker;service:root.service;process:failed;kind:"fixture-failed-start"}
 Process {id:failed;command:["/lab/missing-fixture-executable"];onStarted:root.failedStarts++}
 function failStart(){failed.running=true}
 function failedState(){return {running:failed.running,pending:failedTracker.pending,started:failedStarts}}
 width:600; height:800
 MonitorWindow {id:monitor;service:root.service}
 RemoteRunnerPane {id:remote;service:root.service;width:600}
 ProjectNetworksPane {id:projects;service:root.service;active:true;width:600}
 function find(item,name) {
  if(item.objectName===name)return item
  var children=item.children || []
  for(var i=0;i<children.length;i++){var hit=find(children[i],name);if(hit)return hit}
  return null
 }
 readonly property var logs:find(monitor.contentItem,"runtime-pane")
 readonly property var ssh:find(remote,"remote-setup-host")
 function open() {monitor.present();return projects.controller.refresh()}
 function probe() {
  logs.refresh();logs.following=false;logs.following=true
  ssh.visible=false;ssh.visible=true;ssh.refreshSuggestions()
  return {projects:projects.controller.refresh()}
 }
 function snapshot() {return {logs:logs.logs,projects:projects.controller.projects,
  detail:projects.controller.detail,suggestions:ssh.suggestions}}
}
'''
OMABIT_HELPER=r'''#!/usr/bin/python3 -B
import json,os,signal,subprocess,sys,time
from pathlib import Path
args=sys.argv[1:];profile=Path(os.environ['XDG_STATE_HOME']).name
assert profile in ('stable','dev','edge')
def record(event):
 with Path('/lab/calls.jsonl').open('a') as f:f.write(json.dumps({'event':event,'profile':profile,'pid':os.getpid(),'args':args})+'\n')
record('start')
assert args[:2]==['--format','json']
if args[2:]==['runners']:
 if Path('/lab/late-runners').exists():
  Path('/lab/late-read-started').write_text(str(os.getpid()))
  while not Path('/lab/release-read').exists():time.sleep(.05)
 print(json.dumps([{'name':'fixture','runner_id':'fixture-runner'}]))
elif args[2:]==['inventory','--watch','--interval','2s']:
 def stop(*_):
  record('term');time.sleep(.75);record('exit');raise SystemExit(0)
 signal.signal(signal.SIGTERM,stop)
 Path('/lab/'+profile+'-feed-ready').write_text(str(os.getpid()))
 while True:time.sleep(.05)
elif args[2:]==['remote-setup-status']:
 print('{}')
elif args[2:6]==['--runner','fixture','--expected-runner-id','fixture-runner']:
 tail=args[6:]
 assert tail in (['projects'],['logs','fixture-zod','--revision','1','--watch','--interval','3s']),tail
 if tail==['projects']:
  print(json.dumps({'schema_version':1,'runner_id':'fixture-runner','mode':'development-managed','read_at':'2026-10-24T00:00:00Z','runtime_observation':'not-performed','basis':'stored-reservations','projects':[]}));record('exit');raise SystemExit(0)
 kind='logs'
 def stop(*_):
  signal.signal(signal.SIGTERM,signal.SIG_IGN)
  record('term');time.sleep(2 if kind=='projects' else 1.4)
  print('{}',flush=True);record('exit');raise SystemExit(0)
 signal.signal(signal.SIGTERM,stop)
 Path('/lab/'+profile+'-'+kind+'-ready').write_text(str(os.getpid()))
 while True:time.sleep(.05)
elif args[2:] in (['app-plugins'],['ships'],['app-connections']):
 print('[]')
elif args[2:]==['fixture-auth-hold']:
 Path('/lab/auth-hold-started').write_text(str(os.getpid()))
 while not Path('/lab/release-auth-hold').exists():time.sleep(.04)
 print('{}')
elif args[2:]==['fixture-auth-crash']:
 child=subprocess.Popen(['/usr/bin/python3','-B','-c',"import time;time.sleep(60)"],stdin=subprocess.DEVNULL,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL,start_new_session=True)
 Path('/lab/auth-child.json').write_text(json.dumps({'pid':child.pid,'start':Path('/proc/'+str(child.pid)+'/stat').read_text().split(') ',1)[1].split()[19]}))
 Path('/lab/auth-owner').write_text(str(os.getpid()))
 while True:time.sleep(.04)
elif args[2:]==['fixture-hold']:
 Path('/lab/hold-started').write_text(str(os.getpid()))
 while not Path('/lab/release-hold').exists():time.sleep(.05)
 print('{}')
else:raise AssertionError('Unexpected fixture command: '+repr(args))
record('exit')
'''
OMABIT_WRAPPER=r'''import QtQuick
Service {
 id:root
 readonly property bool fixtureCollectionOmabit:true
 profileName:"PROFILE"
 profileEnvironment:({XDG_STATE_HOME:"/lab/omabit/state/PROFILE",XDG_CONFIG_HOME:"/lab/omabit/config/PROFILE",XDG_DATA_HOME:"/lab/omabit/data/PROFILE",XDG_CACHE_HOME:"/lab/omabit/cache/PROFILE"})
 toolsDirectory:Qt.resolvedUrl("bin")
 localSetupAllowed:false
 settings:({notifications:false})
 Readers {id:views;service:root;visible:false}
 function fixtureInventory() {
  var ship={id:"fixture-zod",config_revision:1,ship:"~zod",mode:"fake",config:{http_port:18381,ames_port:34343,loom:31},observed_state:"stopped",desired_state:"stopped",observed_at:new Date().toISOString()}
  connections=[{name:"fixture",runner_id:"fixture-runner",mode:"development-managed",status:"available",access_role:"operator",checked_at:new Date().toISOString(),features:["runtime-logs","project-reservations-v1","project-saved-status-v1"],snapshot:{instances:[ship]}}]
 }
 function fixtureOpenReaders(){fixtureInventory();views.visible=true;views.open()}
 Timer {interval:1000;repeat:true;running:views.visible&&root.observing;onTriggered:root.fixtureInventory()}
}
'''
def sha(p):return hashlib.sha256(p.read_bytes()).hexdigest()
def inside():
 root=Path('/lab')
 if os.environ.get('OMABIT_REAL_PLUGIN_LIFECYCLE')!='private-v1' or os.environ.get('HOME')!='/lab/home' or os.environ.get('XDG_RUNTIME_DIR')!='/lab/runtime' or Path('/home').exists() or Path('/run/user').exists() or (root/'marker').read_text()!='private-v1\n':raise RuntimeError('private lifecycle namespace required')
 original=json.loads((root/'host-namespaces.json').read_text());assert all(os.readlink('/proc/self/ns/'+n)!=original[n] for n in original);assert [n for _,n in socket.if_nameindex()]==['lo']
 if os.environ.get('OMABIT_COLLECTION_JOURNAL')=='private-v1':
  from journal_runtime_fixture import run
  return run(root)
 if os.environ.get('OMABIT_LIFECYCLE_APPEARANCE')=='1':
  from appearance_fixture import run
  return run(root)
 combined=os.environ.get('OMABIT_LIFECYCLE_OMABIT')=='1'
 connected=os.environ.get('OMABIT_LIFECYCLE_CONNECTED')=='1'
 drain=os.environ.get('OMABIT_LIFECYCLE_DRAIN')=='1'
 effects=os.environ.get("OMABIT_LIFECYCLE_EFFECTS")=="1"
 assert not drain or connected
 reconcile=os.environ.get("OMABIT_LIFECYCLE_RECONCILE")=="1"
 assert not effects or drain
 assert not reconcile or effects
 report={'passed':False,'cases':[],'real_plugins':['Theme','DMs','Omabit'] if combined else ['Theme','DMs'],'real_processes':True,'empty_accounts_only':not connected,'connected_subscriptions_qualified':False,'synthetic_eyre':connected,'real_gall':False,'appearance_disabled':True,'busy_writes_qualified':False,'full_shell':False,'actual_compositor':False}
 def save():(root/'result.json').write_text(json.dumps(report,indent=2)+'\n')
 save()
 keyring=hub=None
 if connected:
  from connected_fixture import begin
  keyring,hub=begin(root)
 with (root/'shell.log').open('wb') as out:
  proc=subprocess.Popen(['/usr/bin/quickshell','--no-color','--path','/lab/shell.qml'],stdout=out,stderr=subprocess.STDOUT,start_new_session=True)
  def call(method,*args,require=True):
   r=subprocess.run(['/usr/bin/quickshell','ipc','--pid',str(proc.pid),'call','--','realPluginLifecycle',method,*args],capture_output=True,text=True,timeout=4)
   if require:assert r.returncode==0,(method,r.stderr[-600:])
   return r
  def wait(fn,seconds=14):
   end=time.monotonic()+seconds
   while time.monotonic()<end:
    assert proc.poll() is None,'private shell exited'
    result=fn()
    if result:return result
    time.sleep(.05)
   raise AssertionError('lifecycle deadline')
  def children(pid):
   # Traverse only descendants of our exact private Quickshell child.
   path=Path('/proc/'+str(pid)+'/task/'+str(pid)+'/children')
   try:ids=[int(n) for n in path.read_text().split()]
   except FileNotFoundError:return []
   return ids+[v for child in ids for v in children(child)]
  def identity(pid):
   try:
    p=Path('/proc')/str(pid);stat=(p/'stat').read_text().split(') ',1)[1].split();argv=(p/'cmdline').read_bytes().split(b'\0');script=next((v.decode() for v in argv if v.startswith(b'/lab/artifacts/') and v.endswith(b'.py')),'')
    return {'pid':pid,'start_ticks':int(stat[19]),'state':stat[0],'script':script}
   except (FileNotFoundError,ProcessLookupError):return None
  def helpers(name):
   rows=[identity(p) for p in children(proc.pid)];return [r for r in rows if r and r['state']!='Z' and r['script'].endswith(('/bridge.py','/client/sync.py')) and '/'+name+'/' in r['script']]
  def exited(old):
   now=identity(old['pid']);return not now or now['start_ticks']!=old['start_ticks'] or now['state']=='Z'
  previous=[];devices={}
  def omabit_helpers():
   rows=[]
   for pid in children(proc.pid):
    row=identity(pid)
    if not row or row['state']=='Z':continue
    try:argv=Path('/proc/'+str(pid)+'/cmdline').read_bytes()
    except FileNotFoundError:continue
    if b'/omabit/bin/urbit-collector' in argv:rows.append(row)
   return rows
  def open_omabit():
   if not combined:return
   wait(lambda:json.loads(call('state').stdout).get('omabit',{}).get('observing'))
   assert call('openOmabit').stdout.strip()=='opened'
   wait(lambda:len(omabit_helpers())==2)
   report.setdefault('omabit_ready',[]).append({'snapshot':json.loads(call('state').stdout)['omabit'],'helpers':omabit_helpers()});save()
  def drain_omabit():
   if not combined:return
   old=omabit_helpers();assert len(old)==2
   assert call('drainOmabit').stdout.strip()=='requested'
   def closed():
    receipt=json.loads(call('state').stdout)['omabit']['transportState']
    return receipt if receipt.get('closed') else None
   receipt=wait(closed)
   assert receipt['auxiliaryReadersTracked'] and receipt['helperExitsTracked'] and not receipt['collectionReady']
   assert not receipt['unconfirmedHelpers'] and not receipt['pendingReaders'] and not receipt['readerOwnerLost']
   wait(lambda:all(exited(p) for p in old));assert not omabit_helpers()
   report.setdefault('omabit_drains',[]).append({'receipt':receipt,'old_helpers':old,'all_exited':True});save()
  def drain_dms():
   drain_omabit()
   assert call('drainDms').stdout.strip()=='requested'
   def finished():
    state=json.loads(call('state').stdout);item=next(iter(state['panels'].values()))
    return item['transportState'] if item['transportState'].get('closed') else None
   receipt=wait(finished,seconds=35)
   wait(lambda:all(exited(p) for p in previous if p['script'].endswith('/bridge.py')))
   wait(lambda:not any(x['owner'].endswith('-dms') for x in hub.active()))
   rows=[x for x in hub.snapshot() if x['owner'].endswith('-dms')]
   assert rows and all(x['deleted'] for x in rows)
   assert not receipt['collectionReady']
   report.setdefault('dms_transport_drains',[]).append({'receipt':receipt,'all_dms_channels_deleted':True,'old_dms_helpers_exited':True});save()
   assert call('send').stdout.strip()=='false' and len(hub.pokes)==(3 if effects else 1)
   theme_state=next(iter(json.loads(call('state').stdout)['services'].items()))
   profile=theme_state[0].split('.qualification-')[-1]
   disk=root/'state'/('omarchy-urbit-theme.qualification-'+profile)/'desktop.json';before=sha(disk)
   requests=[]
   if combined and not report.get('theme_drain_admission'):
    assert call('probeThemeBusy').stdout.strip()=='busy'
    report['theme_busy_refusal_probed']=True;requests.append('busy');save()
   def request_theme_drain():
    result=call('drainTheme').stdout.strip();requests.append(result)
    assert result in ('busy','requested'),result
    return result=='requested'
   wait(request_theme_drain)
   report.setdefault('theme_drain_admission',[]).append({'profile':profile,'busy_replies':requests.count('busy'),'requested':True});save()
   def theme_finished():
    state=json.loads(call('state').stdout);item=next(iter(state['services'].values()))
    return item['transportState'] if item['transportState'].get('closed') else None
   theme_receipt=wait(theme_finished,seconds=75)
   wait(lambda:all(exited(p) for p in previous));wait(lambda:not hub.active())
   assert all(x['deleted'] for x in hub.snapshot()) and sha(disk)==before
   assert theme_receipt['workersStopped'] and not theme_receipt['collectionReady']
   report.setdefault('theme_transport_drains',[]).append({'receipt':theme_receipt,'all_channels_deleted':True,'all_helpers_exited':True,'desktop_state_unchanged':True,'profile':profile});save()


  try:
   wait(lambda:call('state',require=False).stdout.strip().startswith('{'))
   for target in (['stable'] if drain else ['stable','dev','stable','edge','stable']):
    call('select','none');wait(lambda:all(exited(p) for p in previous));wait(lambda:not hub or not hub.active());wait(lambda:json.loads(call('state').stdout)=={'services':{},'panels':{}})
    assert not helpers('stable') and not helpers('dev') and not helpers('edge')
    call('select',target)
    def ready():
     s=json.loads(call('state').stdout);hs=helpers(target)
     if len(s['services'])!=1 or len(s['panels'])!=1 or len(hs)!=2:return None
     t=next(iter(s['services'].values()));d=next(iter(s['panels'].values()))
     return (s,hs) if t['loaded'] and d['loaded'] and ((t['ships']==1 and d['ship']=='~zod' and len(hub.active())==2 and {r['owner'] for r in hub.active()}=={target+'-theme',target+'-dms'} and all(r['delivered']>=r['subscriptions']>0 for r in hub.active())) if connected else (t['ships']==0 and t['desktop']['status']=='Add a ship to start desktop sync' and not d['ship'])) else None
    snap,current=wait(ready);assert all(exited(p) for p in previous)
    open_omabit()
    disk=root/'state'/('omarchy-urbit-theme.qualification-'+target)/'desktop.json';device=json.loads(disk.read_text())['deviceId']
    if target in devices:assert devices[target]==device
    else:devices[target]=device
    report['cases'].append({'target':target,'old_helpers_exited_before_enable':True,'previous_helpers':previous,'helpers':current,'snapshot':snap,'same_profile_device_preserved':True,'old_streams_closed_before_enable':True,'active_streams':hub.active() if hub else []});save();previous=current
   if drain:
    assert call('send').stdout.strip()==''
    def panel():return next(iter(json.loads(call('state').stdout)['panels'].values()))
    wait(lambda:len(hub.pokes)==1 and panel()['busy'])
    before=panel();assert before['pendingSend']>0 and before['pendingText']=='synthetic pending lifecycle probe'
    wait(lambda:panel()['statusError'] and not panel()['busy'] and panel()['pendingSend']==-1,seconds=18)
    after=panel();assert after['pendingText']==before['pendingText'] and 'unknown' in after['status'].lower() and len(hub.pokes)==1
    assert helpers('stable')==previous and len(hub.active())==2
    report['uncertain_write']={'pending':before,'after_timeout':after,'pokes':list(hub.pokes),'helpers_unchanged':True,'active_streams':hub.active(),'busy_flag_alone_is_insufficient':True};save()
    wait(lambda:len(panel()['unresolved'])==1 and panel()['outboundReadable'])
    intent=panel()['unresolved'][0];journal=root/'state/omarchy-urbit-dms.qualification-stable/outbound-messages.json'
    digest=sha(journal);record=json.loads(journal.read_text());assert len(record['messages'])==1 and record['messages'][0]['phase']=='uncertain' and record['messages'][0]['message']['text']=='synthetic pending lifecycle probe'
    report['persistent_unknown']={'before_reload':intent,'journal_sha256':digest,'profile_roundtrip':[]};save()
    effect_intents=[];effect_journal=root/'state/omarchy-urbit-dms.qualification-stable/conversation-effects.json'
    if effects:
     assert call('effects').stdout.strip()=='requested'
     wait(lambda:len(hub.pokes)==3 and len(panel()['unresolvedEffects'])==2,seconds=20)
     effect_intents=panel()['unresolvedEffects'];effect_digest=sha(effect_journal)
     assert {r['kind'] for r in effect_intents}=={'accept-invitation','mark-read'}
     report['persistent_effects']={'before_reload':effect_intents,'journal_sha256':effect_digest,'profile_roundtrip':[]};save()
    for target in ('dev','stable','edge','stable'):
     drain_dms()
     call('select','none');wait(lambda:all(exited(p) for p in previous));wait(lambda:not hub.active())
     call('select',target)
     def loaded():
      state=json.loads(call('state').stdout);hs=helpers(target)
      if len(state['panels'])!=1 or len(hs)!=2 or len(hub.active())!=2:return None
      item=next(iter(state['panels'].values()))
      if item['ship']!='~zod' or not item['outboundReadable']:return None
      expected=[intent] if target=='stable' else []
      return (state,hs,item) if item['unresolved']==expected and (not effects or item['unresolvedEffects']==(effect_intents if target=='stable' else [])) else None
     state,hs,item=wait(loaded)
     open_omabit()
     device=json.loads((root/'state'/('omarchy-urbit-theme.qualification-'+target)/'desktop.json').read_text())['deviceId']
     if target in devices:assert devices[target]==device
     else:devices[target]=device
     assert all(exited(p) for p in previous) and sha(journal)==digest and len(hub.pokes)==(3 if effects else 1)
     if effects:
      assert sha(effect_journal)==effect_digest
      report['persistent_effects']['profile_roundtrip'].append({'target':target,'unresolved':item['unresolvedEffects'],'total_pokes':len(hub.pokes)})
     report['persistent_unknown']['profile_roundtrip'].append({'target':target,'snapshot':item,'old_helpers_exited':True,'new_helpers':hs,'total_send_pokes':len(hub.pokes)});save();previous=hs
    if reconcile:
     assert len(panel()['unresolvedEffects'])==2 and len(hub.pokes)==3
     hub.confirmed=True
     assert call('refresh').stdout.strip()=='requested'
     wait(lambda:panel()['outboundReadable'] and panel()['unresolvedEffects']==[])
     confirmed=json.loads(effect_journal.read_text());confirmed_digest=sha(effect_journal)
     assert confirmed['version']==2 and {row['phase'] for row in confirmed['effects']}=={'state-confirmed'}
     assert {row['id'] for row in confirmed['effects']}=={row['id'] for row in effect_intents}
     assert sha(journal)==digest and panel()['unresolved']==[intent] and len(hub.pokes)==3
     report['effect_reconciliation']={'desired_state_confirmed':True,'original_acknowledgement_received':False,'total_pokes':len(hub.pokes),'journal_sha256':confirmed_digest,'send_still_unknown':True};save()
     drain_dms()
     call('select','none');wait(lambda:all(exited(p) for p in previous));wait(lambda:not hub.active())
     call('select','stable')
     def restored():
      state=json.loads(call('state').stdout);current=next(iter(state['panels'].values()),None);hs=helpers('stable')
      if current is None:return None
      return hs if len(hs)==2 and len(hub.active())==2 and current['outboundReadable'] and current['unresolvedEffects']==[] and current['unresolved']==[intent] else None
     old=previous;previous=wait(restored)
     open_omabit()
     assert all(exited(p) for p in old) and sha(effect_journal)==confirmed_digest and sha(journal)==digest and len(hub.pokes)==3
     report['effect_reconciliation']['survives_helper_restart']=True;save()
    drain_dms()
   call('select','none');wait(lambda:all(exited(p) for p in previous));wait(lambda:not hub or not hub.active());assert not children(proc.pid)
   assert call('stop').stdout.strip()=='stopping' and proc.wait(timeout=4)==0
   text=(root/'shell.log').read_text();assert not any(s in text for s in ('ReferenceError','TypeError','Binding loop','ERROR:'))
   report.update(passed=True,explicit_transport_drain_qualified=drain,connected_subscriptions_qualified=connected,channels=hub.snapshot() if hub else [],actions=hub.requests if hub else [],zero_application_writes=not hub or all(x['action']!='poke' for x in hub.requests),clean_shell_exit=0,final_helpers_exited=True,distinct_profile_devices=len(set(devices.values()))==3);assert report['distinct_profile_devices'];save();print('PASS: actual '+('Omabit/Theme/DMs' if combined else 'Theme/DMs')+' lifecycle ('+('connected synthetic Eyre' if connected else 'empty accounts')+')')
  except Exception as e:
   report['error']=type(e).__name__+': '+str(e);report['actions']=hub.requests if hub else []
   try:report['failure_snapshot']=json.loads(call('state').stdout)
   except Exception:pass
   save();raise
  finally:
   if hub:hub.close()
   if keyring:
    keyring.terminate()
    try:keyring.wait(timeout=3)
    except subprocess.TimeoutExpired:keyring.kill();keyring.wait(timeout=3)
   if proc.poll() is None:
    os.killpg(proc.pid,signal.SIGTERM)
    try:proc.wait(timeout=3)
    except subprocess.TimeoutExpired:os.killpg(proc.pid,signal.SIGKILL);proc.wait(timeout=3)
def main():
 global THEME,DMS,OMABIT
 p=argparse.ArgumentParser(description=__doc__);p.add_argument('--inside',action='store_true');p.add_argument('--collection-explicit-recovery',action='store_true',help='versioned old-activation recovery through real plugin journals; requires recovery-state');p.add_argument('--collection-recovery-state',action='store_true',help='actual lost application receipts at supervised owner loss');p.add_argument('--collection-supervisor',action='store_true',help='own QML descendants in an external private subreaper; requires shell recovery');p.add_argument('--collection-shell-recovery',action='store_true',help='controlled shell replacement and unclean-owner fencing; requires --collection-journal');p.add_argument('--collection-journal',action='store_true',help='journal-driven private supported-set activation/rollback; requires connected drain and Omabit');p.add_argument('--connected',action='store_true',help='use synthetic Eyre and an isolated real keyring; desktop appearance disabled');p.add_argument('--drain',action='store_true',help='preserve an unknown send and explicitly drain before stable/dev/edge replacement; requires --connected');p.add_argument('--effects',action='store_true',help='also preserve uncertain invitation/read actions; requires --drain');p.add_argument('--reconcile-effects',action='store_true',help='confirm effect state by authenticated reads and reload; requires --effects');p.add_argument('--native-theme-staging',action='store_true',help='use copied native headless theme staging/templates; requires --appearance-offline');p.add_argument('--appearance-offline',action='store_true',help='explicit checkpoint and offline restore with failed retry; requires --connected');p.add_argument('--appearance-rollback',action='store_true',help='real helpers/private simulated desktop failure and rollback; requires --connected, separate from drain scenarios');p.add_argument('--output',type=Path);p.add_argument('--omabit-source',type=Path,help='also load real Omabit service/monitor with inert collector; requires connected drain');p.add_argument('--theme-source',type=Path);p.add_argument('--dms-source',type=Path);a=p.parse_args()
 if a.inside:return inside()
 if a.collection_explicit_recovery and not a.collection_recovery_state:p.error('--collection-explicit-recovery requires --collection-recovery-state')
 if a.collection_recovery_state and not a.collection_supervisor:p.error('--collection-recovery-state requires --collection-supervisor')
 if a.collection_supervisor and not a.collection_shell_recovery:p.error('--collection-supervisor requires --collection-shell-recovery')
 if a.collection_shell_recovery and not a.collection_journal:p.error('--collection-shell-recovery requires --collection-journal')
 if a.collection_journal and (not a.omabit_source or not a.connected or not a.drain or a.effects or a.reconcile_effects or a.appearance_rollback or a.appearance_offline):p.error('--collection-journal requires --connected --drain --omabit-source, without other scenario flags')
 if a.omabit_source and not (a.connected and a.drain):p.error('--omabit-source requires --connected --drain')
 if (a.appearance_rollback or a.appearance_offline) and (not a.connected or a.drain or a.effects or a.reconcile_effects or a.appearance_rollback and a.appearance_offline):p.error('choose one appearance mode with --connected and no uncertainty flags')
 if a.native_theme_staging and not a.appearance_offline:p.error('--native-theme-staging requires --appearance-offline')
 if a.drain and not a.connected:p.error("--drain requires --connected")
 if a.effects and not a.drain:p.error("--effects requires --drain")
 if a.reconcile_effects and not a.effects:p.error("--reconcile-effects requires --effects")
 if not a.output or not a.theme_source or not a.dms_source:p.error('--output, --theme-source and --dms-source required')
 THEME=a.theme_source.resolve(strict=True);DMS=a.dms_source.resolve(strict=True);OMABIT=a.omabit_source.resolve(strict=True) if a.omabit_source else None
 work=a.output.absolute();assert work.parent==Path('/tmp') and re.fullmatch('omabit-real-plugin-lifecycle-[A-Za-z0-9-]+',work.name) and not work.exists();work.mkdir(mode=0o700)
 for n in ['home','runtime','cache','config','data','state','artifacts','imports','etc']:(work/n).mkdir(mode=0o700)
 (work/'etc/fonts').mkdir()
 (work/'etc/passwd').write_text('lab:x:'+str(os.getuid())+':'+str(os.getgid())+':Private lab:/lab/home:/bin/sh\n');(work/'etc/group').write_text('lab:x:'+str(os.getgid())+':\n');(work/'etc/nsswitch.conf').write_text('passwd: files\ngroup: files\n');(work/'marker').write_text('private-v1\n');(work/'host-namespaces.json').write_text(json.dumps({n:os.readlink('/proc/self/ns/'+n) for n in ('user','net','pid','mnt')}));shutil.copy2(__file__,work/'driver.py');shutil.copy2(Path(__file__).parent/'qualification/collections/connected_fixture.py',work/'connected_fixture.py')
 if a.appearance_rollback or a.appearance_offline or a.collection_journal:
  for name in ('appearance_fixture.py','appearance_desktop.py'):shutil.copy2(Path(__file__).parent/'qualification/collections'/name,work/name)
 if a.collection_explicit_recovery:shutil.copy2(Path(__file__).parent/'qualification/collections/recovery_runtime.py',work/'recovery_runtime.py')
 if a.collection_supervisor:
  for name in ('process_supervisor.py','supervised_shell.py'):shutil.copy2(Path(__file__).parent/'qualification/collections'/name,work/name)
 if a.collection_journal:
  shutil.copy2(Path(__file__).parent/'qualification/collections/journal_runtime_fixture.py',work/'journal_runtime_fixture.py')
  (work/'collection-source').mkdir()
  for name in ('journal.py','coordinator.py'):shutil.copy2(Path(__file__).resolve().parents[1]/'native/collections'/name,work/'collection-source'/name)
 (work/'bin').mkdir()
 for name in ('omarchy','omarchy-shell','hyprctl','notify-send','xdg-open'):
  f=work/'bin'/name;f.write_text('#!/bin/sh\nexit 69\n');f.chmod(0o700)
 shutil.copytree(DMS/'tests/qml/imports/qs',work/'imports/qs')
 for name in UI:shutil.copy2(NATIVE/'Ui'/(name+'.qml'),work/'imports/qs/Ui'/(name+'.qml'))
 (work/'imports/qs/Ui/qmldir').write_text('module qs.Ui\n'+''.join(n+' 1.0 '+n+'.qml\n' for n in UI+['KeyboardPanel']))
 for name in ['Style.qml','Util.qml','Border.qml','BorderGeometry.js']:shutil.copy2(NATIVE/'Commons'/name,work/'imports/qs/Commons'/name)
 manifests={};hashes={}
 for profile in ('stable','dev','edge'):
  for name,src in [('theme',THEME),('dms',DMS)]:
   dst=work/'artifacts'/profile/name;dst.mkdir(parents=True)
   files=[p for p in src.iterdir() if p.is_file() and (p.suffix in ('.qml','.js','.py') or p.name in ('qmldir','manifest.json')) and not p.name.startswith('test_')]
   if name=='theme':files +=[p for p in (src/'client').rglob('*.py') if '__pycache__' not in p.parts]
   for f in files:
    rel=f.relative_to(src);target=dst/rel;target.parent.mkdir(parents=True,exist_ok=True);shutil.copy2(f,target);hashes[str(f)]=sha(f)
   if name=='dms':
    f=dst/'Dms.qml';s=f.read_text();assert s.count('  PanelWindow {')==1;s=s.replace('import Quickshell.Wayland\n','').replace('  PanelWindow {','  Rectangle {\n    width: 1024; height: 768').replace('    anchors { top: true; bottom: true; left: true; right: true }\n','');s='\n'.join(l for l in s.split('\n') if not l.strip().startswith(('WlrLayershell.','exclusionMode:')));f.write_text(s)
   m=json.loads((dst/'manifest.json').read_text());m['id']+='.'+'qualification-'+profile;m['name']+=' (qualification '+profile+')';(dst/'manifest.json').write_text(json.dumps(m,indent=2)+'\n');(dst/'application-profile.json').write_text(json.dumps({'version':1,'profile':'qualification-'+profile})+'\n')
   m['entryPoints']={k:'file:///lab/'+str(dst.relative_to(work))+'/'+v for k,v in m['entryPoints'].items()};manifests[m['id']]=m
 if OMABIT:
  for profile in ('stable','dev','edge'):
   dst=work/'artifacts'/profile/'omabit';dst.mkdir();(dst/'bin').mkdir()
   for src in sorted((OMABIT/'plugins/omabit').iterdir()):
    if src.suffix in ('.qml','.js') or src.name in ('qmldir','manifest.json'):
     shutil.copy2(src,dst/src.name);hashes[str(src)]=sha(src)
   (dst/'WindowActivation.qml').write_text(OMABIT_FOCUS)
   (dst/'Readers.qml').write_text(OMABIT_READERS)
   (dst/'CollectionFixture.qml').write_text(OMABIT_WRAPPER.replace('PROFILE',profile))
   (dst/'ssh-hosts.py').write_text('print(\'{"hosts":[]}\')\n')
   for name in ('urbit-collector','omabit'):
    tool=dst/'bin'/name;tool.write_text(OMABIT_HELPER);tool.chmod(0o700)
   m=json.loads((dst/'manifest.json').read_text());m['id']+='.'+'qualification-'+profile;m['name']+=' (qualification '+profile+')';m['entryPoints']={'service':'file:///lab/artifacts/'+profile+'/omabit/CollectionFixture.qml'}
   (dst/'manifest.json').write_text(json.dumps(m,indent=2)+'\n');manifests[m['id']]=m
 if a.collection_journal:
  for profile in ('stable','dev','edge'):
   for role,filename in [('omabit','CollectionFixture.qml'),('theme','Service.qml'),('dms','Dms.qml')]:
    file=work/'artifacts'/profile/role/filename;body=file.read_text()
    marker=' id:root' if role=='omabit' else '  id: root'
    assert body.count(marker)==1
    extra='\n  property var qualificationIdentity: null\n  Timer {\n    interval:1;repeat:true;running:root.qualificationIdentity===null && !!root.shell && !!root.manifest\n    onTriggered: {\n      var token="";for(var n=0;n<4;n++)token+=("00000000"+Math.floor(Math.random()*4294967296).toString(16)).slice(-8)\n      root.qualificationIdentity={epoch:root.shell.qualificationEpoch,source:Qt.resolvedUrl("FILENAME").toString(),profile:"qualification-PROFILE",instance:token,plugin_id:root.manifest.id}\n    }\n  }\n'.replace('FILENAME',filename).replace('PROFILE',profile)
    file.write_text(body.replace(marker,marker+extra,1))
 native=(NATIVE/'shell.qml').read_text();methods=[]
 for name in FUNCTIONS:
  match=re.search(r'^  function '+re.escape(name)+r'\([^\n]*\) \{\n.*?^  \}',native,re.M|re.S);assert match;methods.append(match[0])
 start=native.index('  Instantiator {',native.index('  function computePanelEntries()'));end=native.index('\n  // ---------------------------------------------------------- plugin loader',start);panel=native[start:end].strip();assert panel.count('Instantiator {')==1
 (work/'native-methods.qml.txt').write_text('\n\n'.join(methods)+'\n'+panel);(work/'shell.qml').write_text(QML.replace('MANIFESTS',json.dumps(manifests)).replace('METHODS','\n\n'.join(methods)).replace('PANELS',panel));(work/'AuthServiceStore.js').write_text('.pragma library\nfunction has(id){return false}\nfunction ids(){return []}\nfunction destroy(id){throw new Error("unexpected authentication service")}\n')
 if a.native_theme_staging:
  native=work/'native';(native/'bin').mkdir(parents=True)
  native_hashes={}
  for name in ('omarchy-theme-set','omarchy-theme-set-templates','omarchy-theme-color'):
   source=Path('/usr/share/omarchy/bin')/name;shutil.copy2(source,native/'bin'/name);native_hashes[str(source)]=sha(source)
  shutil.copytree('/usr/share/omarchy/default/themed',native/'default/themed')
  for source in Path('/usr/share/omarchy/default/themed').iterdir():
   assert source.is_file() and not source.is_symlink();native_hashes[str(source)]=sha(source)
  (work/'native-staging.json').write_text(json.dumps({'source_sha256':native_hashes,'headless':True,'native_filesystem_and_templates':True,'native_app_retint':False,'actual_compositor':False},indent=2)+'\n')
 if a.collection_journal:
  path=work/'shell.qml';body=path.read_text();body=body.replace(' id:shell',' id:shell\n property string qualificationEpoch: ""',1)
  body=body.replace(' function snapshot(){',""" function observedBindings(){
  var result=({});for(var id in _services){var a=_services[id];result[a.fixtureCollectionOmabit===true?"omabit":"theme"]=a.qualificationIdentity}
  for(var id in panels)if(panels[id].item)result.dms=panels[id].item.qualificationIdentity
  return JSON.stringify(result)
 }
 function snapshot(){""",1)
  body=body.replace('  target:"realPluginLifecycle"',"""  target:"realPluginLifecycle"
  function bindings():string{return shell.observedBindings()}
  function selectEpoch(name:string,epoch:string):string{if(Object.keys(shell._services).length||Object.keys(shell.panels).length)return "occupied";shell.qualificationEpoch=epoch;return shell.select(name)}""",1)
  path.write_text(body)
  (work/'coordinator-source.json').write_text(json.dumps({name:sha(work/name) for name in ('journal_runtime_fixture.py','collection-source/journal.py','collection-source/coordinator.py')},indent=2)+'\n')
 (work/'source.json').write_text(json.dumps({'original_sources':hashes,'native_shell_sha256':sha(NATIVE/'shell.qml'),'quickshell_sha256':sha(Path('/usr/bin/quickshell')),'adaptations':['private labels/profile declarations','Dms compositor window replaced by Rectangle','private test registry and synthetic Eyre with real private keyring','desktop commands refused by private shims','native UI with fixture colors/KeyboardPanel'],'no_plugin_helper_replacements':not bool(OMABIT),'theme_dms_helpers_unchanged':True,'omabit_inert_collector':bool(OMABIT),'fixture_sha256':sha(Path(__file__))},indent=2)+'\n')
 cmd=['bwrap','--unshare-all','--die-with-parent','--clearenv','--ro-bind','/usr','/usr','--symlink','usr/bin','/bin','--symlink','usr/lib','/lib','--symlink','usr/lib','/lib64','--proc','/proc','--dev','/dev','--tmpfs','/tmp','--ro-bind',str(work/'etc'),'/etc','--ro-bind','/etc/fonts','/etc/fonts','--bind',str(work),'/lab','--ro-bind',str(work/'artifacts'),'/lab/artifacts','--chdir','/lab']
 if a.native_theme_staging:cmd+=['--ro-bind',str(work/'native'),'/lab/native']
 env={'HOME':'/lab/home','XDG_RUNTIME_DIR':'/lab/runtime','XDG_CONFIG_HOME':'/lab/config','XDG_DATA_HOME':'/lab/data','XDG_STATE_HOME':'/lab/state','XDG_CACHE_HOME':'/lab/cache','PATH':'/lab/bin:/usr/bin:/bin','LANG':'C.UTF-8','QT_QPA_PLATFORM':'offscreen','QT_QUICK_BACKEND':'software','QML_DISABLE_DISK_CACHE':'1','QT_QUICK_CONTROLS_STYLE':'Basic','QT_QPA_PLATFORMTHEME':'','QML2_IMPORT_PATH':'/lab/imports','OMABIT_REAL_PLUGIN_LIFECYCLE':'private-v1','OMABIT_LIFECYCLE_OMABIT':'1' if OMABIT else '0','OMABIT_COLLECTION_SHELL_RECOVERY':'1' if a.collection_shell_recovery else '0','OMABIT_COLLECTION_JOURNAL':'private-v1' if a.collection_journal else '0','OMABIT_LIFECYCLE_APPEARANCE':'1' if a.appearance_rollback or a.appearance_offline or a.collection_journal else '0','OMABIT_LIFECYCLE_APPEARANCE_OFFLINE':'1' if a.appearance_offline else '0','OMABIT_NATIVE_THEME_STAGING':'1' if a.native_theme_staging else '0','OMABIT_LIFECYCLE_CONNECTED':'1' if a.connected else '0','OMABIT_LIFECYCLE_DRAIN':'1' if a.drain else '0','OMABIT_LIFECYCLE_EFFECTS':'1' if a.effects else '0','OMABIT_LIFECYCLE_RECONCILE':'1' if a.reconcile_effects else '0','PYTHONDONTWRITEBYTECODE':'1'}
 if a.collection_explicit_recovery:
  env['OMABIT_COLLECTION_EXPLICIT_RECOVERY']='1'
  (work/'recovery-source.json').write_text(json.dumps({'recovery_runtime.py':sha(work/'recovery_runtime.py')},indent=2)+'\n')
 if a.collection_recovery_state:env['OMABIT_COLLECTION_RECOVERY_STATE']='1'
 if a.collection_supervisor:
  env['OMABIT_COLLECTION_SUPERVISOR']='1';env['OMABIT_SUPERVISOR_LAB']='private-v1'
  (work/'supervisor-source.json').write_text(json.dumps({name:sha(work/name) for name in ('process_supervisor.py','supervised_shell.py')},indent=2)+'\n')
 for k,v in env.items():cmd+=['--setenv',k,v]
 inner=['--']+(['/usr/bin/dbus-run-session','--'] if a.connected else [])+['/usr/bin/python3','-B','/lab/driver.py','--inside']
 (work/'command.json').write_text(json.dumps(cmd+inner,indent=2)+'\n');result=subprocess.run(cmd+inner,capture_output=True,text=True,timeout=300 if a.collection_journal else 200);(work/'driver.log').write_text(result.stdout+result.stderr);print(result.stdout+result.stderr);raise SystemExit(result.returncode)
if __name__=='__main__':main()
