#!/usr/bin/env python3
"""Actual Omabit service lifecycle with inert tools in a private no-NIC namespace.

--readers includes the real monitor, runtime logs and separate setup reader controls.
Only window focus, Commons design tokens and exact-argv helper results are inert.
The baseline retains its minimal monitor stub. Neither mode contacts a real runner.
"""
import argparse,hashlib,json,os,re,shutil,signal,socket,subprocess,time
from pathlib import Path
COMPONENTS=('Service.qml','Model.js','ApplicationConnections.qml','ImportRestoration.qml')
MONITOR='''import QtQuick
Item {
 property var service
 property string page: "overview"
 property string selectedKey: ""
 property bool fullscreen: false
 property var contentItem: Item {}
 property var setupPane: QtObject {property string mode:"home"}
 function present() {visible=true}
}
'''
QML='''import QtQuick
import Quickshell
import Quickshell.Io
ShellRoot {
 id:root
 property string selected: ""
 Loader {id:app;active:false}
 Loader {id:readers;active:false}
 Loader {id:panel;active:false}
 function snapshot() {
  var a=app.item
  return JSON.stringify({profile:selected,loaded:!!a,registrationsLoaded:a?a.registrationsLoaded:false,
   authFresh:a?a.applicationConnections.fresh:false,busy:a?a.busy:false,observing:a?a.observing:false,draining:a?a.transportStopping:false,
   receipt:a?a.transportState:{closed:false},inventoryExitPending:a?a.inventoryExitPending:false,readers:readers.item?readers.item.snapshot():null})
 }
 IpcHandler {
  target:"omabitServiceDrainFixture"
  function select(profile:string):string {
   if(app.item)throw new Error("disable before selecting")
   if(["stable","dev","edge"].indexOf(profile)<0)throw new Error("bad profile")
   root.selected=profile
   app.setSource("file:///lab/artifacts/"+profile+"/Service.qml",{profileName:profile,
    profileEnvironment:{XDG_STATE_HOME:"/lab/state/"+profile,XDG_CONFIG_HOME:"/lab/config/"+profile,
      XDG_DATA_HOME:"/lab/data/"+profile,XDG_CACHE_HOME:"/lab/cache/"+profile},
    manifest:{id:"thelifeandtimes.omabit.qualification-"+profile},settings:{notifications:false},
    localSetupAllowed:false,toolsDirectory:"file:///lab/artifacts/"+profile+"/bin"})
   app.active=true
   if(Quickshell.env("OMABIT_SERVICE_PANEL_FIXTURE")==="1") {
    panel.setSource("file:///lab/artifacts/"+profile+"/Panel.qml",{service:app.item,moduleName:"thelifeandtimes.omabit.qualification-"+profile,settings:{notifications:false}})
    panel.active=true
   }
   return root.snapshot()
  }
  function state():string{return root.snapshot()}
  function panelState():string {var p=panel.item;return JSON.stringify({loaded:!!p,module:p?p.moduleName:"",target:p?p.ipcTarget:"",pane:p?p.pane:"",opened:p?p.opened:false})}
  function openReaders():string {
   var a=app.item
   var ship={id:"fixture-zod",config_revision:1,ship:"~zod",mode:"fake",config:{http_port:18381,ames_port:34343,loom:31},observed_state:"stopped",desired_state:"stopped",observed_at:new Date().toISOString()}
   a.connections=[{name:"fixture",runner_id:"fixture-runner",mode:"development-managed",status:"available",access_role:"operator",
    checked_at:new Date().toISOString(),features:["runtime-logs","project-reservations-v1","project-saved-status-v1"],snapshot:{instances:[ship]},project_inventory:{schema_version:1,runner_id:"fixture-runner",mode:"development-managed",read_at:new Date().toISOString(),basis:"stored-reservations",runtime_observation:"not-performed",projects:[{id:"project-fixture",label:"Peer review",revision:1,review_digest:"a".repeat(64),members:[{instance_id:"project-zod",ship:"~zod",http_port:18581},{instance_id:"project-nec",ship:"~nec",http_port:18582}]}]}}]
   readers.setSource("file:///lab/artifacts/"+root.selected+"/Readers.qml",{service:a});readers.active=true
   return readers.item.open() ? "opened" : "failed"
  }
  function failedStart():string {readers.item.failStart();return "requested"}
  function failedState():string {return JSON.stringify(readers.item.failedState())}
  function probeReaders():string {return JSON.stringify(readers.item.probe())}
  function destroyReaders():string {readers.active=false;return "destroyed"}
  function authHold():string{return app.item.applicationConnections.begin(["fixture-auth-hold"],"connect","")}
  function authCrash():string{return app.item.applicationConnections.begin(["fixture-auth-crash"],"connect","")}
  function hold():string{return app.item.submit(["fixture-hold"],"fixture-hold")}
  function lateRead():string{app.item.loadRegistrations();return "requested"}
  function drain():string{return app.item.stopTransport()}
  function probeAdmission():string{
   var a=app.item;a.refresh();a.loadRegistrations();a.loadTheme();a.startFeed();a.loadShips();a.checkRemoteSetup();a.checkLocalSetup();a.pumpNotifications()
   return JSON.stringify({submit:a.submit(["fixture-forbidden"],"fixture-forbidden"),auth:a.applicationConnections.refresh()})
  }
  function disable():string{panel.active=false;readers.active=false;app.active=false;return root.snapshot()}
  function stop():string{Qt.callLater(Qt.quit);return "stopping"}
 }
}
'''
HELPER='''#!/usr/bin/python3 -B
import json,os,signal,subprocess,sys,time
from pathlib import Path
args=sys.argv[1:];profile=Path(os.environ['XDG_STATE_HOME']).name
assert profile in ('stable','dev','edge')
def record(event):
 with Path('/lab/calls.jsonl').open('a') as f:f.write(json.dumps({'event':event,'profile':profile,'pid':os.getpid(),'args':args})+'\\n')
record('start')
assert args[:2]==['--format','json']
if args[2:]==['runners']:
 if Path('/lab/late-runners').exists():
  Path('/lab/late-read-started').write_text(str(os.getpid()))
  while not Path('/lab/release-read').exists():time.sleep(.05)
 print(json.dumps([{'name':'fixture','runner_id':'fixture-runner','transport':'unix'}]))
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
 kind='projects' if tail==['projects'] else 'logs'
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
READERS=r'''import QtQuick
import Quickshell.Io
import "Model.js" as Model
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
 function snapshot() {
  var nav=find(monitor.contentItem,"nav-ship-project-zod"), rep=find(monitor.contentItem,"nav-groups-fixture-runner")
  return {logs:logs.logs,projects:projects.controller.projects,groups:Model.shipGroups(service.connections[0]),
  groupCount:rep?rep.count:-1, groupModel:rep?rep.model:null, projectNav:nav?{visible:nav.visible,width:nav.width,height:nav.height}:null,
  detail:projects.controller.detail,suggestions:ssh.suggestions}}

}
'''
FOCUS=r'''import QtQuick
Item {required property var targetWindow;signal failed();function activate(){targetWindow.visible=true}}
'''
SSH_READER=r'''import json,os,signal,time
from pathlib import Path
profile=Path(__file__).parent.name
def record(event):
 with Path('/lab/calls.jsonl').open('a') as f:f.write(json.dumps({'event':event,'profile':profile,'pid':os.getpid(),'args':['ssh-suggestions']})+'\n')
def stop(*_):
 signal.signal(signal.SIGTERM,signal.SIG_IGN)
 record('term');time.sleep(2.5)
 print(json.dumps({'hosts':[{'host':'late-fixture','label':'late fixture','port':22}]}),flush=True)
 record('exit');raise SystemExit(0)
signal.signal(signal.SIGTERM,stop)
record('start');Path('/lab/'+profile+'-ssh-ready').write_text(str(os.getpid()))
while True:time.sleep(.04)
'''
def sha(p):return hashlib.sha256(p.read_bytes()).hexdigest()
def inside():
 root=Path('/lab')
 assert os.environ.get('OMABIT_SERVICE_DRAIN_FIXTURE')=='private-v1' and os.environ.get('HOME')=='/lab/home' and os.environ.get('XDG_RUNTIME_DIR')=='/lab/runtime'
 assert not Path('/home').exists() and not Path('/run/user').exists() and (root/'marker').read_text()=='private-v1\n'
 ns=json.loads((root/'host-namespaces.json').read_text());assert all(os.readlink('/proc/self/ns/'+n)!=ns[n] for n in ns);assert [n for _,n in socket.if_nameindex()]==['lo']
 reader_mode=(root/'readers-mode').exists();panel_mode=(root/'panel-mode').exists()
 report={'native_panel_mode':panel_mode,'readers_mode':reader_mode,'actual_monitor':reader_mode,'actual_setup_readers':reader_mode,'actual_compositor':False,'passed':False,'actual_service':True,'inert_tools':True,'monitor_stub':not reader_mode,'no_runner_operations':True,'cases':[]}
 def save():(root/'result.json').write_text(json.dumps(report,indent=2)+'\n')
 save()
 with (root/'shell.log').open('wb') as log:
  shell=subprocess.Popen(['/usr/bin/quickshell','--no-color','--path','/lab/shell.qml'],stdout=log,stderr=subprocess.STDOUT,start_new_session=True)
  def call(method,*args,check=True):
   q=subprocess.run(['/usr/bin/quickshell','ipc','--pid',str(shell.pid),'call','--','omabitServiceDrainFixture',method,*args],capture_output=True,text=True,timeout=3)
   if check:assert q.returncode==0,(method,q.stderr[-800:])
   return q.stdout.strip()
  def ipc(target,method):
   return subprocess.run(['/usr/bin/quickshell','ipc','--pid',str(shell.pid),'call','--',target,method],capture_output=True,text=True,timeout=3)
  def wait(fn,seconds=8):
   end=time.monotonic()+seconds
   while time.monotonic()<end:
    assert shell.poll() is None,'private shell exited'
    result=fn()
    if result:return result
    time.sleep(.04)
   raise AssertionError('private service deadline')
  def children(pid):
   try:ids=[int(x) for x in Path('/proc/'+str(pid)+'/task/'+str(pid)+'/children').read_text().split()]
   except FileNotFoundError:return []
   return ids+[p for child in ids for p in children(child)]
  def ident(pid):
   try:fields=Path('/proc/'+str(pid)+'/stat').read_text().split(') ',1)[1].split();return {'pid':pid,'start':fields[19],'state':fields[0]}
   except FileNotFoundError:return None
  def exited(old):
   now=ident(old['pid']);return not now or now['start']!=old['start'] or now['state']=='Z'
  def state():return json.loads(call('state'))
  try:
   wait(lambda:call('state',check=False).startswith('{'))
   for profile in ('stable','dev','stable','edge','stable'):
    before=len((root/'calls.jsonl').read_text().splitlines()) if (root/'calls.jsonl').exists() else 0
    ready_file=root/(profile+'-feed-ready')
    if ready_file.exists():ready_file.unlink()
    call('select',profile);wait(lambda:state()['registrationsLoaded'] and state()['observing'] and ready_file.exists())
    feed=ident(int(ready_file.read_text()));assert feed and feed['pid'] in children(shell.pid)
    module='thelifeandtimes.omabit.qualification-'+profile
    native=ipc(module+'.monitor','status');assert native.returncode==0,native.stderr
    native_status=json.loads(native.stdout);assert native_status['module']==module and native_status['profile']==profile
    canonical=ipc('thelifeandtimes.omabit.monitor','status')
    shown=subprocess.run(['/usr/bin/quickshell','ipc','--pid',str(shell.pid),'show'],capture_output=True,text=True,timeout=3)
    (root/('ipc-'+str(len(report['cases']))+'.json')).write_text(json.dumps({'scoped':{'code':native.returncode,'stdout':native.stdout,'stderr':native.stderr},'canonical':{'code':canonical.returncode,'stdout':canonical.stdout,'stderr':canonical.stderr},'show':shown.stdout},indent=2)+'\n')
    targets=re.findall(r'^target (.+)$',shown.stdout,re.M)
    assert canonical.stdout.strip()=='Target not found.' and 'thelifeandtimes.omabit.monitor' not in targets
    assert module+'.monitor' in targets
    report.setdefault('scoped_ipc',[]).append({'module':module,'profile':profile,'native_status':native_status,'canonical_absent':True})
    if panel_mode:
     wait(lambda:json.loads(call('panelState'))['loaded'])
     for target in (module,module+'.setup'):
      assert target in targets,('missing profile widget IPC',target,shown.stdout)
     assert 'thelifeandtimes.omabit' not in targets and 'thelifeandtimes.omabit.setup' not in targets
     assert ipc(module+'.setup','open').stdout.strip()==''
     wait(lambda:json.loads(call('panelState'))=={'loaded':True,'module':module,'target':module,'pane':'setup','opened':True})
     assert ipc(module,'close').stdout.strip()==''
     wait(lambda:not json.loads(call('panelState'))['opened'])
     report.setdefault('scoped_panel_ipc',[]).append({'module':module,'setup_opened':True,'close_verified':True,'canonical_absent':True})


    if not report['cases']:
     assert call('hold')=='true';wait(lambda:(root/'hold-started').exists());assert call('drain')=='busy'
     assert not state()['draining'] and not exited(feed)
     (root/'release-hold').touch();wait(lambda:not state()['busy'])
     report['foreground_work_refuses_drain']=True;save()
     assert call('authHold')=='true';wait(lambda:(root/'auth-hold-started').exists())
     assert call('drain')=='busy' and not state()['draining']
     (root/'release-auth-hold').touch()
     wait(lambda:not state()['busy'] and state()['authFresh'])
     report['application_connection_work_refuses_drain']=True;save()
     (root/'late-runners').touch();assert call('lateRead')=='requested'
     wait(lambda:(root/'late-read-started').exists())
    owned=[]
    if reader_mode:
     for kind in ('logs','projects','ssh'):
      marker=root/(profile+'-'+kind+'-ready')
      if marker.exists():marker.unlink()
     assert call('openReaders')=='opened'
     wait(lambda:all((root/(profile+'-'+k+'-ready')).exists() for k in ('logs','projects','ssh')))
     navigation=state()['readers'];assert navigation['projectNav'] and navigation['projectNav']['visible'] and navigation['projectNav']['width']>0,navigation
     owned=[ident(int((root/(profile+'-'+k+'-ready')).read_text())) for k in ('logs','projects','ssh')]
     assert all(x and x['pid'] in children(shell.pid) for x in owned)
     if not report['cases']:
      assert call('failedStart')=='requested'
      wait(lambda:json.loads(call('failedState'))=={'running':False,'pending':False,'started':0})
      report['failed_start_does_not_leave_false_pending_exit']=True;save()
     assert sorted(state()['receipt']['pendingReaders'])==['project-history','runtime-logs','ssh-suggestions'],state()
    assert call('drain')=='requested'
    early=state();assert early['draining'] and early['inventoryExitPending'] and not early['receipt']['closed'],early
    if reader_mode:
     assert early['receipt']['auxiliaryReadersTracked'] and len(early['receipt']['pendingReaders'])==3,early
     assert json.loads(call('probeReaders'))=={'projects':False}
     wait(lambda:exited(feed))
     assert not state()['receipt']['closed'] and any(not exited(x) for x in owned),state()
    assert json.loads(call('probeAdmission'))=={'submit':False,'auth':False}
    if not report['cases']:
     wait(lambda:exited(feed));assert not state()['receipt']['closed'] and children(shell.pid)
     (root/'release-read').touch();(root/'late-runners').unlink()
     report['late_read_blocks_closure']=True;save()
    wait(lambda:state()['receipt']['closed']);assert not state()['receipt']['collectionReady']
    wait(lambda:exited(feed));wait(lambda:not children(shell.pid))
    if reader_mode:
     assert all(exited(x) for x in owned)
     assert {k:state()['readers'][k] for k in ('logs','projects','detail','suggestions')}=={'logs':None,'projects':[],'detail':None,'suggestions':[]},state()
     assert json.loads(call('probeReaders'))=={'projects':False}
    calls_before=(root/'calls.jsonl').read_bytes();time.sleep(1.2);assert not children(shell.pid) and (root/'calls.jsonl').read_bytes()==calls_before,'paused helper restarted'
    assert call('drain')=='requested'
    current=state();call('disable');wait(lambda:not state()['loaded']);assert not children(shell.pid)
    assert ipc(module+'.monitor','status').stdout.strip()=='Target not found.','old profile IPC remained after unload'
    if panel_mode:
     assert ipc(module+'.setup','open').stdout.strip()=='Target not found.' and ipc(module,'open').stdout.strip()=='Target not found.'
    report['cases'].append({'profile':profile,'feed':feed,'readers':owned,'receipt':current['receipt'],'early_receipt':early['receipt'],'all_owned_helpers_exited':True,'no_reconnect_or_new_admission':True,'old_helpers_exited_before_replacement':True,'new_call_records':len(calls_before.splitlines())-before});save()
   if reader_mode:
    report['signal_killed_helpers']=[]
    for failure in ('inventory','runtime-logs','application-connections'):
     profile='stable'
     for kind in ('feed','logs','projects','ssh'):
      marker=root/(profile+'-'+kind+'-ready')
      if marker.exists():marker.unlink()
     call('select',profile);wait(lambda:state()['registrationsLoaded'] and state()['observing'] and (root/'stable-feed-ready').exists())
     assert call('openReaders')=='opened'
     wait(lambda:all((root/('stable-'+k+'-ready')).exists() for k in ('logs','projects','ssh')))
     orphan=None
     if failure=='application-connections':
      assert call('authCrash')=='true';wait(lambda:(root/'auth-owner').exists())
      victim=ident(int((root/'auth-owner').read_text()));orphan=json.loads((root/'auth-child.json').read_text())
     else:victim=ident(int((root/('stable-'+('feed' if failure=='inventory' else 'logs')+'-ready')).read_text()))
     assert victim and victim['pid'] in children(shell.pid)
     os.kill(victim['pid'],signal.SIGKILL)
     wait(lambda:failure in state()['receipt']['unconfirmedHelpers'])
     wait(lambda:not state()['busy'])
     assert call('drain')=='requested'
     wait(lambda:not children(shell.pid))
     receipt=state()['receipt']
     assert not receipt['closed'] and receipt['reason']=='helper-descendants-unconfirmed' and receipt['unconfirmedHelpers']==[failure],receipt
     assert receipt['helperExitsTracked'] and not receipt['collectionReady'] and exited(victim)
     if orphan:
      assert not exited(orphan),'detached fixture descendant did not survive owner'
      os.kill(orphan['pid'],signal.SIGKILL);wait(lambda:exited(orphan))
      assert not state()['receipt']['closed'],'uncertainty was silently reset'
     report['signal_killed_helpers'].append({'kind':failure,'victim':victim,'orphan':orphan,'receipt':receipt,'exact_children_cleaned':True});save()
     call('disable');wait(lambda:not state()['loaded'])
   if reader_mode:
    profile='stable'
    for kind in ('feed','logs','projects','ssh'):
     marker=root/(profile+'-'+kind+'-ready')
     if marker.exists():marker.unlink()
    call('select',profile)
    wait(lambda:state()['registrationsLoaded'] and state()['observing'] and (root/'stable-feed-ready').exists())
    assert call('openReaders')=='opened'
    wait(lambda:all((root/('stable-'+k+'-ready')).exists() for k in ('logs','projects','ssh')))
    owned=[ident(int((root/('stable-'+k+'-ready')).read_text())) for k in ('logs','projects','ssh')]
    assert call('destroyReaders')=='destroyed'
    wait(lambda:state()['receipt']['readerOwnerLost'])
    assert call('drain')=='requested'
    wait(lambda:not children(shell.pid))
    lost=state()['receipt']
    assert not lost['closed'] and lost['reason']=='reader-owner-lost' and not lost['pendingReaders'],lost
    assert all(exited(x) for x in owned)
    report['removed_live_reader_refuses_closure']={'receipt':lost,'owned':owned,'all_exact_children_exited':True};save()
    call('disable');wait(lambda:not state()['loaded'])
   assert call('stop')=='stopping'  and shell.wait(timeout=4)==0
   text=(root/'shell.log').read_text();assert not any(x in text for x in ('ReferenceError','TypeError','Binding loop','ERROR:','Unable to assign','Cannot assign')),text[-3000:]
   report.update(passed=True,clean_shell_exit=0);save();print('PASS: actual Omabit service drain, foreground refusal, delayed observer exit and stable/dev/edge replacement'+(' with monitor/setup reader accounting and owner-loss refusal' if reader_mode else ''))
  except Exception as e:report['error']=type(e).__name__+': '+str(e);save();raise
  finally:
   if shell.poll() is None:
    os.killpg(shell.pid,signal.SIGTERM)
    try:shell.wait(timeout=3)
    except subprocess.TimeoutExpired:os.killpg(shell.pid,signal.SIGKILL);shell.wait(timeout=3)
def main():
 p=argparse.ArgumentParser(description=__doc__);p.add_argument('--inside',action='store_true');p.add_argument('--readers',action='store_true');p.add_argument('--panel',action='store_true',help='also load actual Panel with native UI controls; requires --readers');p.add_argument('--source',type=Path);p.add_argument('--output',type=Path);a=p.parse_args()
 if a.inside:return inside()
 if a.panel and not a.readers:p.error('--panel requires --readers')
 if not a.source or not a.output:p.error('--source and --output required')
 source=a.source.resolve(strict=True);out=a.output.absolute();assert out.parent==Path('/tmp') and re.fullmatch('omabit-service-drain-[A-Za-z0-9-]+',out.name) and not out.exists();out.mkdir(mode=0o700)
 for n in ('home','runtime','config','state','cache','data','artifacts','etc'):(out/n).mkdir(mode=0o700)
 (out/'etc/fonts').mkdir();(out/'etc/passwd').write_text('lab:x:'+str(os.getuid())+':'+str(os.getgid())+':Lab:/lab/home:/bin/sh\n');(out/'etc/group').write_text('lab:x:'+str(os.getgid())+':\n');(out/'etc/nsswitch.conf').write_text('passwd: files\ngroup: files\n')
 (out/'marker').write_text('private-v1\n');(out/'host-namespaces.json').write_text(json.dumps({n:os.readlink('/proc/self/ns/'+n) for n in ('user','net','pid','mnt')}));hashes={}
 if a.readers:
  (out/'readers-mode').touch()
  commons=out/'Commons';commons.mkdir()
  (commons/'qmldir').write_text('singleton Style 1.0 Style.qml\nsingleton Color 1.0 Color.qml\n')
  (commons/'Style.qml').write_text('pragma Singleton\nimport QtQuick\nQtObject {function space(n){return n} property var font:({family:"sans-serif",body:14,bodySmall:14,caption:12})}\n')
  (commons/'Color.qml').write_text('pragma Singleton\nimport QtQuick\nQtObject {property color background:"#171a1e";property color foreground:"#e7e9eb";property color accent:"#8ab4f8";property color urgent:"#ed8994"}\n')
 if a.panel:
  (out/'panel-mode').touch()
  native=Path('/usr/share/omarchy/shell');ui=out/'Ui';ui.mkdir()
  names=['Panel','PanelController','BarIconButton','WidgetButton','OpticalGlyph','PanelToolTip','BorderSurface','BorderOverlay','PanelKeyCatcher']
  for name in names:
   src=native/'Ui'/(name+'.qml');shutil.copy2(src,ui/src.name);hashes[str(src)]=sha(src)
  (ui/'KeyboardPanel.qml').write_text('import QtQuick\nItem { property Item anchorItem;property var owner;property var bar;property bool open:false;property Item focusTarget;property int contentWidth:440;property int contentHeight:720;width:contentWidth;height:contentHeight;visible:open;function fittedContentWidth(n){return n} function fittedContentHeight(n,cap){return Math.min(n,cap)} }\n')
  (ui/'qmldir').write_text('module qs.Ui\n'+''.join(n+' 1.0 '+n+'.qml\n' for n in names+['KeyboardPanel']))
  for name in ('Style.qml','Util.qml','Border.qml','BorderGeometry.js'):
   src=native/'Commons'/name;shutil.copy2(src,commons/name);hashes[str(src)]=sha(src)
  (commons/'qmldir').write_text('module qs.Commons\nsingleton Style 1.0 Style.qml\nsingleton Color 1.0 Color.qml\nsingleton Util 1.0 Util.qml\nsingleton Border 1.0 Border.qml\n')
  (commons/'Color.qml').write_text('pragma Singleton\nimport QtQuick\nQtObject {property color background:"#171a1e";property color foreground:"#e7e9eb";property color accent:"#8ab4f8";property color urgent:"#ed8994";property var shellValues:({});property var menu:({background:"#171a1e",text:Qt.rgba(0.9,0.9,0.9,1),border:"#8ab4f8",scrim:"#88000000"});property var tooltip:({background:"#171a1e",text:Qt.rgba(0.9,0.9,0.9,1),border:"#8ab4f8"});property var popups:({background:"#171a1e",text:Qt.rgba(0.9,0.9,0.9,1),border:"#8ab4f8"})}\n')
 for profile in ('stable','dev','edge'):
  dest=out/'artifacts'/profile;dest.mkdir();(dest/'bin').mkdir()
  for name in COMPONENTS:
   src=source/'plugins/omabit'/name;shutil.copy2(src,dest/name);hashes[str(src)]=sha(src)
  if a.readers:
   for src in sorted((source/'plugins/omabit').iterdir()):
    if src.suffix in ('.qml','.js') or src.name in ('qmldir','omabit.svg'):
     shutil.copy2(src,dest/src.name);hashes[str(src)]=sha(src)
   (dest/'WindowActivation.qml').write_text(FOCUS)
   (dest/'Readers.qml').write_text(READERS)
   (dest/'ssh-hosts.py').write_text(SSH_READER)
  else:(dest/'MonitorWindow.qml').write_text(MONITOR)
  for name in ('urbit-collector','omabit'):
   tool=dest/'bin'/name;tool.write_text(HELPER);tool.chmod(0o700)
 (out/'shell.qml').write_text(QML);shutil.copy2(__file__,out/'driver.py');(out/'source.json').write_text(json.dumps({'sha256':hashes,'fixture_sha256':sha(Path(__file__)),'adaptations':['inert exact-argv collector/omabit','offscreen focus and inert SSH suggestions' if a.readers else 'monitor stub','private profiles','inert Commons tokens']},indent=2)+'\n')
 cmd=['bwrap','--unshare-all','--die-with-parent','--clearenv','--ro-bind','/usr','/usr','--symlink','usr/bin','/bin','--symlink','usr/lib','/lib','--symlink','usr/lib','/lib64','--proc','/proc','--dev','/dev','--tmpfs','/tmp','--ro-bind',str(out/'etc'),'/etc','--ro-bind','/etc/fonts','/etc/fonts','--bind',str(out),'/lab','--ro-bind',str(out/'artifacts'),'/lab/artifacts','--chdir','/lab']
 env={'HOME':'/lab/home','XDG_RUNTIME_DIR':'/lab/runtime','XDG_CONFIG_HOME':'/lab/config','XDG_DATA_HOME':'/lab/data','XDG_STATE_HOME':'/lab/state','XDG_CACHE_HOME':'/lab/cache','PATH':'/usr/bin:/bin','LANG':'C.UTF-8','QT_QPA_PLATFORM':'offscreen','QT_QUICK_BACKEND':'software','QML_DISABLE_DISK_CACHE':'1','QT_QUICK_CONTROLS_STYLE':'Basic','QT_QPA_PLATFORMTHEME':'','PYTHONDONTWRITEBYTECODE':'1','OMABIT_SERVICE_DRAIN_FIXTURE':'private-v1','OMABIT_SERVICE_PANEL_FIXTURE':'1' if a.panel else '0'}
 for k,v in env.items():cmd+=['--setenv',k,v]
 cmd+=['--','/usr/bin/python3','-B','/lab/driver.py','--inside'];(out/'command.json').write_text(json.dumps(cmd,indent=2)+'\n');q=subprocess.run(cmd,capture_output=True,text=True,timeout=120);(out/'driver.log').write_text(q.stdout+q.stderr);print(q.stdout+q.stderr);raise SystemExit(q.returncode)
if __name__=='__main__':main()
