#!/usr/bin/env python3
"""Actual native bar/widgets/layer panels in a private software compositor.

Copies native bar, widget registration and shell API code unchanged. Configuration,
colors and appearance/runner tools remain private fixtures; this is not Hyprland.
"""
from pathlib import Path
import argparse,importlib.util,json,re,shutil,subprocess,sys,time

def module(name,path):
 s=importlib.util.spec_from_file_location(name,path);m=importlib.util.module_from_spec(s);s.loader.exec_module(m);return m

def method(text,name):
 found=re.search(r'^  function '+name+r'\([^\n]*\) \{\n.*?^  \}',text,re.M|re.S);assert found,name;return found[0]

QML=r'''
 property NativeShell.BarWidgetRegistry barWidgetRegistry:NativeShell.BarWidgetRegistry{}
 property var pluginWidgetComponents:({})
 property var fixtureLayout:["thelifeandtimes.omabit.qualification-stable","thelifeandtimes.urbit-theme.qualification-stable","urbit.dms.qualification-stable"]
 onConfigChanged:{
  if(!pluginRegistry)return
  var enabled=Object.keys(pluginRegistry.installedPlugins).filter(function(id){return pluginRegistry.isEnabled(id)})
  if(!enabled.length)return
  var id=enabled[0] || "",match=id.match(/\.qualification-(stable|dev|edge)$/)
  if(!match)return
  var suffix=match[1],next=["thelifeandtimes.omabit.qualification-"+suffix,"thelifeandtimes.urbit-theme.qualification-"+suffix,"urbit.dms.qualification-"+suffix]
  if(JSON.stringify(next)!==JSON.stringify(fixtureLayout))fixtureLayout=next
 }

 property bool pluginReloading:false
 WIDGET_METHODS
 Connections { target:shell.pluginRegistry;function onPluginsChanged(){shell.syncPluginWidgets()} }
 NativeBar.Bar {
  id:nativeBar
  omarchyPath:"/lab"
  shell:shell
  barWidgetRegistry:shell.barWidgetRegistry
  pluginRegistry:shell.pluginRegistry
  barConfig:({position:"top",transparent:false,centerAnchor:"",layout:{left:[],center:[],right:shell.fixtureLayout}})
 }
 IpcHandler {
  target:"waylandProbe"
  function state():string {
   var rows={}
   for(var id in shell.widgets){var h=shell.widgets[id],p=h.payload;var windows=[]
    if(p && p.data)for(var i=0;i<p.data.length;i++){var x=p.data[i];if(x && "backingWindowVisible" in x)windows.push({mapped:x.backingWindowVisible,visible:x.visible,width:x.width,height:x.height,screen:x.screen?x.screen.name:null,keyboard:"focusPrimed" in x})}
    rows[id]={admitted:h.admitted,opened:h.opened,visible:h.visible,width:h.width,height:h.height,barId:p && p.bar?p.bar.pluginId:null,popoutOwner:nativeBar.activePopout===h,payloadSeesOwner:!!p&&!!p.bar&&p.bar.activePopout===p,windows:windows}
   }
   return JSON.stringify({widgets:rows,slots:nativeBar.moduleSlots.length,barApis:Object.keys(nativeBar.pluginBarApis),clickTargets:nativeBar.clickTargets.length,popout:!!nativeBar.activePopout,screenCount:Quickshell.screens.length})
  }
  function pruneBar():string{var before=Object.keys(nativeBar.pluginBarApis);nativeBar.prunePluginBarApis();return JSON.stringify({before:before,after:Object.keys(nativeBar.pluginBarApis)})}
  function dmsState(id:string):string {
   var host=shell.panels[id].item,p=host.payload,windows=[]
   for(var i=0;i<p.data.length;i++){var x=p.data[i];if(x && "backingWindowVisible" in x)windows.push({mapped:x.backingWindowVisible,width:x.width,height:x.height})}
   return JSON.stringify({opened:host.opened,windows:windows})
  }
  function focusDmsClose(id:string):string {
   var p=shell.panels[id].item.payload
   function find(x){if(!x)return null;if(x.tooltipText==="Close" && x.focusable)return x;var c=x.children || [];for(var i=0;i<c.length;i++){var y=find(c[i]);if(y)return y}return null}
   for(var i=0;i<p.data.length;i++){var x=p.data[i];if(x && "backingWindowVisible" in x){var b=find(x.contentItem);if(b){b.forceActiveFocus();return String(b.activeFocus)}}}
   return "absent"
  }
  function openDms(id:string):string{return String(shell.summon(id,"{}"))}
  function connectedUi():string {
   var result={};for(var id in shell._services){var h=shell._services[id];if(h.role==="theme" && h.payload)result.theme={error:h.payload.lastError,busy:h.payload.busy,palette:!!h.payload.currentPalette}}
   for(var id in shell.panels){var h=shell.panels[id].item;if(h && h.payload)result.dms={busy:h.payload.busy,status:h.payload.status,error:h.payload.statusError,rows:h.payload.rows.length}}
   return JSON.stringify(result)
  }
  function open(id:string):string{var h=shell.widgets[id];h.open();return String(h.opened)}
  function closeAll():void{for(var id in shell.widgets)shell.widgets[id].close()}
  function navigate(id:string,direction:int):string{var p=shell.widgets[id].payload;return String(p.bar.switchPanelFrom(p,direction))}
  function peers(id:string):string{var p=shell.widgets[id].payload;return String(p.bar.moduleWidgets(id)[0]===p && p.bar.moduleWidgets("foreign.fixture").length===0)}
 }
'''

class Probe:
 def __init__(self,runtime):self.runtime=runtime;self.rows=[];self.closed=[];self.api_epochs=[];self.api_unloads=[];self.connected=None
 def begin(self,*args):
  self.runtime.start()
  if Path('/lab/native-connected-mode.json').exists():
   self.connected=module('native_connected_fixture',Path('/lab/native_connected_fixture.py')).Probe(Path('/lab'))
   self.connected.begin(*args)
   self.connected.hub.complete_visual_reads=True
 def check(self):
  assert self.runtime.proc.poll() is None
  if self.connected:self.connected.check()
 def no_admission(self):
  if self.connected:self.connected.no_admission()

 def wait(self,fn):
  end=time.monotonic()+8
  while time.monotonic()<end:
   value=fn()
   if value:return value
   time.sleep(.05)
  raise AssertionError('native Wayland assertion deadline')
 def admitted(self,profile,proc,call):
  if self.connected:self.connected.admitted(profile,proc,call)
  ids={'omabit':'thelifeandtimes.omabit.qualification-'+profile,'theme':'thelifeandtimes.urbit-theme.qualification-'+profile,'dms':'urbit.dms.qualification-'+profile}
  state=lambda:json.loads(call(proc,'waylandProbe','state'))
  initial=self.wait(lambda:state() if all(x['width']>0 and x['height']>0 for x in state()['widgets'].values()) else None)
  assert initial['screenCount']==1 and initial['slots']==4 and len(initial['widgets'])==3,initial
  pruned=json.loads(call(proc,'waylandProbe','pruneBar'));assert set(pruned['after'])==set(ids.values()),pruned
  assert all(x['admitted'] and x['visible'] and x['barId']==id for id,x in initial['widgets'].items()),initial
  scopes=json.loads(call(proc,'shellFacadeProbe','state'));assert len(scopes)==3 and all(all(v.values()) for v in scopes.values()),scopes
  ui_state=None
  if self.connected:
   def healthy_ui():
    value=json.loads(call(proc,'waylandProbe','connectedUi'));theme=value.get('theme',{})
    Path('/lab/connected-ui-observation.json').write_text(json.dumps(value,indent=2)+'\n')
    return value if not theme.get('error',True) and not theme.get('busy',True) and theme.get('palette') else None
   self.wait(healthy_ui)

  if Path('/lab/native-api-lifecycle-mode.json').exists():
   apis=json.loads(call(proc,'apiLifecycleProbe','state'));assert len(apis['apis'])==3 and len(apis['descriptors'])==3 and len(apis['barState'])==3,apis
   assert {x['pluginId'] for x in apis['apis'].values()}==set(ids.values()),apis
   current={x['instance'] for x in apis['apis'].values()}
   assert all(not current & {x['instance'] for x in old['state']['apis'].values()} for old in self.api_epochs)
   self.api_epochs.append({'profile':profile,'state':apis})

  self.wait(lambda:not any(w['mapped'] for v in state()['widgets'].values() for w in v['windows']))
  shots=[self.runtime.snapshot(profile+'-'+str(len(self.rows))+'-bar')]
  surfaces=[]
  for role in ('omabit','theme'):
   id=ids[role];assert call(proc,'waylandProbe','open',id)=='true'
   value=self.wait(lambda:state()['widgets'][id] if any(w['mapped'] for w in state()['widgets'][id]['windows']) else None)
   assert value['popoutOwner'] and value['payloadSeesOwner'],value
   assert any(w['mapped'] and w['keyboard'] and w['width']==1280 and w['height']==720 and w['screen']=='HEADLESS-1' for w in value['windows']),value
   surfaces.append({'role':role,**value});shots.append(self.runtime.snapshot(profile+'-'+str(len(self.rows))+'-'+role))
   call(proc,'waylandProbe','closeAll');self.wait(lambda:not state()['popout'] and not any(w['mapped'] for v in state()['widgets'].values() for w in v['windows']))
  assert call(proc,'waylandProbe','navigate',ids['theme'],'-1')=='true'
  self.wait(lambda:state()['widgets'][ids['omabit']]['opened']);call(proc,'waylandProbe','closeAll')
  assert call(proc,'waylandProbe','navigate',ids['dms'],'-1')=='true'
  self.wait(lambda:state()['widgets'][ids['theme']]['opened']);call(proc,'waylandProbe','closeAll')
  assert all(call(proc,'waylandProbe','peers',id)=='true' for id in ids.values())
  assert call(proc,'waylandProbe','openDms',ids['dms'])=='true'
  dms_state=lambda:json.loads(call(proc,'waylandProbe','dmsState',ids['dms']))
  mapped=self.wait(lambda:dms_state() if any(w['mapped'] for w in dms_state()['windows']) else None)
  self.wait(lambda:call(proc,'waylandProbe','focusDmsClose',ids['dms'])=='true')
  control=subprocess.run(['/usr/bin/wtype','-s','250','-k','Return','-s','250'],capture_output=True,text=True,timeout=5)
  assert control.returncode==0,(control.stdout,control.stderr)
  self.wait(lambda:not dms_state()['opened'])
  assert call(proc,'waylandProbe','openDms',ids['dms'])=='true'
  self.wait(lambda:any(w['mapped'] for w in dms_state()['windows']))
  self.wait(lambda:call(proc,'waylandProbe','focusDmsClose',ids['dms'])=='true')
  if self.connected:
   def dms_ui():
    value=json.loads(call(proc,'waylandProbe','connectedUi'));dms=value.get('dms',{})
    Path('/lab/connected-ui-observation.json').write_text(json.dumps(value,indent=2)+'\n')
    return value if not dms.get('busy',True) and not dms.get('error',True) and dms.get('status')=='Up to date · 0 conversations' and dms.get('rows')==0 else None
   ui_state=self.wait(dms_ui)
  shots.append(self.runtime.snapshot(profile+'-'+str(len(self.rows))+'-dms'))
  keyboard=subprocess.run(['/usr/bin/wtype','-s','250','-k','Escape','-s','250'],capture_output=True,text=True,timeout=5)
  assert keyboard.returncode==0,(keyboard.stdout,keyboard.stderr)
  try:self.wait(lambda:not dms_state()['opened'] and not any(w['mapped'] for w in dms_state()['windows']))
  except AssertionError:
   Path('/lab/dms-keyboard-failure.json').write_text(json.dumps({'focus':'actual close button','enter_control_passed':True,'escape_sent':True,'state':dms_state()},indent=2)+'\n');raise
  overlay=json.loads(call(proc,'shellFacadeProbe','overlay',ids['dms']));assert all(overlay.values()),overlay
  self.wait(lambda:not state()['popout'])
  assert call(proc,'waylandProbe','open',ids['theme'])=='true'
  self.wait(lambda:any(w['mapped'] for w in state()['widgets'][ids['theme']]['windows']))
  self.rows.append({'profile':profile,'initial':initial,'surfaces':surfaces,'scopes':scopes,'overlay':overlay,'dms_mapped':mapped,'escape_from_button':True,'enter_control_passed':True,'connected_ui':ui_state,'bar_pruning':pruned,'screenshots':shots})
  Path('/lab/native-wayland-result.json').write_text(json.dumps({'passed':False,'status':'running','profiles':self.rows},indent=2)+'\n')
 def unloaded(self,profile,proc,call):
  if self.connected:self.connected.no_admission()
  if not Path('/lab/native-api-lifecycle-mode.json').exists():return
  def clean():
   call(proc,'apiLifecycleProbe','pruneUnusedBar')
   value=json.loads(call(proc,'apiLifecycleProbe','state'))
   Path('/lab/api-unload-state.json').write_text(json.dumps(value,indent=2)+'\n')
   return value if value['placeholderOnly'] and all(not value[k] for k in ('apis','descriptors','barState','objects','clickTargets','popout')) else None
  value=self.wait(clean)
  # The final profile may be a different shell; only require the epoch's
  # actual native QObject destruction marker, not GC or a global process scan.
  epoch=self.api_epochs[-1]
  self.wait(lambda:all(('SCOPED_API_DESTROYED '+x['pluginId']+' '+x['instance']) in '\n'.join(p.read_text() for p in Path('/lab').glob('shell-*.log')) for x in epoch['state']['apis'].values()))
  self.api_unloads.append({'profile':profile,'empty':value,'destroyed':epoch['state']['apis']})
 def closed_ui(self,profile,proc,call,only):
  state=json.loads(call(proc,'waylandProbe','state'))
  ids=['thelifeandtimes.urbit-theme.qualification-'+profile] if only else ['thelifeandtimes.omabit.qualification-'+profile,'thelifeandtimes.urbit-theme.qualification-'+profile]
  self.wait(lambda:all(not json.loads(call(proc,'waylandProbe','state'))['widgets'][id]['opened'] for id in ids))
  state=json.loads(call(proc,'waylandProbe','state'))
  for id in ids:
   row=state['widgets'][id];assert not row['admitted'] and not row['opened'] and not row['visible'],row
  self.wait(lambda:not any(w['mapped'] for id in ids for w in json.loads(call(proc,'waylandProbe','state'))['widgets'][id]['windows']))
  if not only:
   dms='urbit.dms.qualification-'+profile
   self.wait(lambda:not any(w['mapped'] for w in json.loads(call(proc,'waylandProbe','dmsState',dms))['windows']))
  self.closed.append({'profile':profile,'only':only,'widgets':{id:state['widgets'][id] for id in ids}})

def prepare(out,hashes):
 native=Path('/usr/share/omarchy/shell');support=out/'native-bar';support.mkdir()
 for name in ('Bar.qml','BarModel.js'):
  p=native/'plugins/bar'/name;shutil.copy2(p,support/name);hashes[str(p)]=sha(p)
 (support/'qmldir').write_text('Bar 1.0 Bar.qml\n')
 for name in ('KeyboardPanel','ScreenMoveRemap','PluginBarApi'):
  p=native/'Ui'/(name+'.qml');shutil.copy2(p,out/'imports/qs/Ui'/p.name);hashes[str(p)]=sha(p)
  qmldir=out/'imports/qs/Ui/qmldir';s=qmldir.read_text()
  if not re.search(r'^'+name+r' ',s,re.M):qmldir.write_text(s+name+' 1.0 '+name+'.qml\n')
 p=native/'services/BarWidgetRegistry.qml';shutil.copy2(p,out/'shell-support'/p.name);hashes[str(p)]=sha(p)
 p=out/'shell-support/qmldir';p.write_text(p.read_text()+'BarWidgetRegistry 1.0 BarWidgetRegistry.qml\n')
 colors=out/'imports/qs/Commons/Color.qml';colors.write_text(colors.read_text().replace('property var shellValues:','property var bar:({text:"#e7e9eb",background:"#171a1e",active:"#8ab4f8"});property var shellValues:'))
 p=out/'shell.qml';s=p.read_text().replace('import QtQuick\n','import QtQuick\nimport "native-bar" as NativeBar\n',1)
 start=s.index(' property var widgetEntries:');end=s.index('  function ensureService',start)
 assert 'Instantiator {' in s[start:end];s=s[:start]+s[end:]
 s=s.replace(' property var widgets: ({})',' property var widgets:{var rows={};for(var i=0;i<nativeBar.moduleSlots.length;i++){var slot=nativeBar.moduleSlots[i];if(slot.registered && slot.activeItem)rows[slot.moduleName]=slot.activeItem}return rows}')
 s=s.replace(' property var bar:null',' property var bar:nativeBar').replace(' property var barConfig:config.bar',' property var barConfig:nativeBar.barConfig')
 methods='\n\n'.join(method((native/'shell.qml').read_text(),name) for name in ('syncPluginWidgets','setPluginWidgetComponent','loadPluginWidget'))
 s=s.rstrip();assert s.endswith('}');p.write_text(s[:-1]+QML.replace('WIDGET_METHODS',methods)+'\n}\n')
 (out/'native-wayland-result.json').write_text('{"passed":false,"status":"starting"}\n')

def sha(p):
 import hashlib
 return hashlib.sha256(p.read_bytes()).hexdigest()

def main():
 if '--inside' in sys.argv:
  runtime=module('wayland_runtime',Path('/lab/wayland-runtime.py')).Runtime();probe=Probe(runtime)
  try:
   module('admission_fixture',Path('/lab/admission-fixture.py')).inside(hooks=probe)
   logs='\n'.join(p.read_text() for p in Path('/lab').glob('shell-*.log'))
   assert logs.count('APPLICATION_CREATED ')==9 and logs.count('WIDGET_CREATED ')==9,logs[-3000:]
   assert not any(x in logs for x in ('another handler is registered','Could not attach Keys property','ERROR:')),logs[-3000:]
   if probe.connected:probe.connected.finish()
   Path('/lab/native-wayland-result.json').write_text(json.dumps({'passed':True,'profiles':probe.rows,'closed_admission':probe.closed,'full_native_bar':True,'actual_keyboard_panels':True,'actual_dms_window':True,'hyprland':False,'full_native_shell':False,'api_epochs':probe.api_epochs,'api_unloads':probe.api_unloads,'connected':probe.connected is not None},indent=2)+'\n')
  except BaseException as e:
   Path('/lab/native-wayland-result.json').write_text(json.dumps({'passed':False,'error':str(e),'profiles':probe.rows,'closed_admission':probe.closed},indent=2)+'\n');raise
  finally:
   if probe.connected:probe.connected.close()
   runtime.close()
  return
 parser=argparse.ArgumentParser(add_help=False);parser.add_argument('--wayland-runtime',type=Path,required=True);parser.add_argument('--api-lifecycle',action='store_true');parser.add_argument('--connected',action='store_true');opt,left=parser.parse_known_args();sys.argv[1:]=left
 source=Path(__file__).resolve().parents[1];scripts=source/'scripts';actual=module('actual_native',scripts/'test-collection-omabit-guarded-native.py');original=subprocess.run
 if '--actual-pair' not in sys.argv:sys.argv.append('--actual-pair')
 def launch(command,*args,**kw):
  assert command[0]=='bwrap';out=Path(command[command.index('--bind')+1]);hashes=json.loads((out/'source.json').read_text())
  api=scripts/'qualification/collections/native_shell_api_fixture.py';module('shell_api',api).prepare(out,hashes,bar_fixture=False);hashes[str(api)]=sha(api)
  prepare(out,hashes)
  if opt.api_lifecycle:
   api_lifecycle=scripts/'qualification/collections/native_api_lifecycle_fixture.py';module('api_lifecycle',api_lifecycle).prepare(out,hashes);hashes[str(api_lifecycle)]=sha(api_lifecycle)

  helper=scripts/'qualification/collections/wayland_runtime.py';shutil.copy2(helper,out/'wayland-runtime.py');hashes[str(helper)]=sha(helper)
  module('wayland_runtime',helper).prepare(out,opt.wayland_runtime,command,hashes)
  if opt.connected:
   prepared=json.loads((out/'prepared.json').read_text())
   for profile,roles in prepared.items():
    for role in ('theme','dms'):
     dest=out/'artifacts'/profile/role;dest.parent.mkdir(parents=True,exist_ok=True);shutil.copytree(roles[role]['1']['path'],dest)
   for name in ('native_connected_fixture.py','connected_fixture.py','appearance_fixture.py','appearance_desktop.py'):
    src=scripts/'qualification/collections'/name;shutil.copy2(src,out/name);hashes[str(src)]=sha(src)
   driver=scripts/'test-collection-connected-guarded-native.py';extra=module('connected_qml',driver).QML;hashes[str(driver)]=sha(driver)
   path=out/'shell.qml';body=path.read_text().rstrip();assert body.endswith('}');path.write_text(body[:-1]+extra+'\n}\n')
   (out/'native-connected-mode.json').write_text('{"synthetic_eyre":true,"private_keyring":true}\n')
   at=command.index('--');command[at:at]=['--setenv','OMABIT_REAL_PLUGIN_LIFECYCLE','private-v1','--setenv','OMABIT_LIFECYCLE_APPEARANCE','1']
   at=command.index('--');command[at+1:at+1]=['/usr/bin/dbus-run-session','--']
  hashes[str(Path(__file__))]=sha(Path(__file__));(out/'source.json').write_text(json.dumps(hashes,indent=2)+'\n');kw['timeout']=300 if opt.connected else 150
  return original(command,*args,**kw)
 subprocess.run=launch;actual.__file__=__file__;actual.main(wayland=True)

if __name__=='__main__':main()
