#!/usr/bin/env python3
"""Private actual-widget ownership/navigation against copied native bar methods."""
from pathlib import Path
import importlib.util,json,re,shutil,sys

def module(name,path):
 s=importlib.util.spec_from_file_location(name,path);m=importlib.util.module_from_spec(s);s.loader.exec_module(m);return m

QML=r'''
 Item {
  id:root
  property var activePopout:null
  property int runs:0
  property var moduleSlots: {
   var result=[];var widgets=shell.widgets
   Object.keys(widgets).sort().forEach(function(id){var w=widgets[id];result.push({activeItem:w,moduleName:id,region:"right",visible:w.visible,width:24,height:24})})
   return result
  }
  property var apis:({})
  function layoutEntries(region){return moduleSlots.map(function(s){return s.moduleName})}
  function entryId(entry){return entry}
  function slotWindow(slot){return null}
  function sameWindow(a,b){return a===b}
  NATIVE_BAR_METHODS
  Component {id:barApi;NativeBar.PluginBarApi{}}
  function apiFor(id) {
   if(apis[id])return apis[id]
   var a=barApi.createObject(root,{pluginId:id,moduleName:id,fontFamily:"monospace",barSize:24,
    foreground:"#ffffff",barForeground:"#ffffff",background:"#222222",urgent:"#ee3333",
    _run:function(command){root.runs++},
    _switchPanelFrom:function(owner,direction){return root.switchPanelFrom(owner,direction)},
    _requestPopout:function(owner){root.requestPopout(owner)},
    _releasePopout:function(owner){root.releasePopout(owner)},
    _moduleWidgets:function(requested){return requested===id?root.moduleWidgets(id):[]}})
   a.activePopout=Qt.binding(function(){return root.activePopout})
   var next=Object.assign({},apis);next[id]=a;apis=next;return a
  }
 }
 IpcHandler {
  target:"barProbe"
  function state():string {
   var result={}
   for(var id in shell.widgets){var h=shell.widgets[id],p=h.payload;result[id]={loaded:!!p,bar:!!p&&!!p.bar,barId:p&&p.bar?p.bar.pluginId:null,opened:h.opened,admitted:h.admitted}}
   return JSON.stringify(result)
  }
  function navigate(id:string,direction:int):string {
   var h=shell.widgets[id];if(!h || !h.payload)return "absent"
   if(typeof h.payload.switchPanel==="function")return String(h.payload.switchPanel(direction))
   return String(h.payload.bar.switchPanelFrom(h.payload,direction))
  }
  function popout(id:string):string {
   var h=shell.widgets[id];h.open();h.payload.bar.requestPopout(h.payload)
   return JSON.stringify({hostOwns:root.activePopout===h,payloadSeesItself:h.payload.bar.activePopout===h.payload,opened:h.opened})
  }
  function blocked(id:string):string {
   var h=shell.widgets[id],p=h?h.payload:null;if(!h || h.admitted)return "invalid"
   if(!p)return JSON.stringify({inertWithoutPayload:true})
   var before=root.activePopout,count=root.runs;p.bar.requestPopout(p)
   var moved=p.bar.switchPanelFrom(p,1);p.bar.run("inert fixture command")
   return JSON.stringify({noPopout:root.activePopout===before,noNavigation:!moved,noRun:root.runs===count})
  }
  function peers(id:string):string {var h=shell.widgets[id],p=h.payload;return String(p.bar.moduleWidgets(id)[0]===p && p.bar.moduleWidgets("foreign.fixture").length===0)}
  function closeAll():void{for(var id in shell.widgets)shell.widgets[id].close();root.activePopout=null}
 }
'''

class Probe:
 def __init__(self):self.rows=[];self.closed=[];self.facades=[]
 def begin(self,*args):pass
 def check(self):pass
 def no_admission(self):pass
 def closed_ui(self,profile,proc,call,only):
  roles=[only] if only else ['omabit','theme']
  for role in roles:
   id={'omabit':'thelifeandtimes.omabit.qualification-','theme':'thelifeandtimes.urbit-theme.qualification-'}[role]+profile
   value=json.loads(call(proc,'barProbe','blocked',id));assert all(value.values()),value
   self.closed.append({'profile':profile,'role':role,**value})
 def admitted(self,profile,proc,call):
  names={'omabit':'thelifeandtimes.omabit.qualification-'+profile,'theme':'thelifeandtimes.urbit-theme.qualification-'+profile,'dms':'urbit.dms.qualification-'+profile}
  if Path('/lab/native-shell-api-mode.json').exists():
   scopes=json.loads(call(proc,'shellFacadeProbe','state'));assert len(scopes)==3 and all(all(v.values()) for v in scopes.values()),scopes
   overlay=json.loads(call(proc,'shellFacadeProbe','overlay',names['dms']));assert all(overlay.values()),overlay
   self.facades.append({'profile':profile,'scopes':scopes,'overlay':overlay})
  state=json.loads(call(proc,'barProbe','state'));assert len(state)==3 and all(x['loaded'] and x['bar'] and x['barId']==id for id,x in state.items())
  assert call(proc,'barProbe','navigate',names['theme'],'-1')=='true','native bar could not map the actual panel to its guarded slot'
  state=json.loads(call(proc,'barProbe','state'));assert state[names['omabit']]['opened']
  call(proc,'barProbe','closeAll')
  result=json.loads(call(proc,'barProbe','popout',names['theme']));assert all(result.values()),result
  call(proc,'barProbe','closeAll')
  assert call(proc,'barProbe','navigate',names['dms'],'-1')=='true'
  assert json.loads(call(proc,'barProbe','state'))[names['theme']]['opened']
  call(proc,'barProbe','closeAll')
  assert all(call(proc,'barProbe','peers',id)=='true' for id in names.values())
  self.rows.append({'profile':profile,'navigation':True,'popout_owner_maps_to_host':True,'payload_sees_own_popout':True,'actual_native_bar_methods':True})
  Path('/lab/native-bar-result.json').write_text(json.dumps({'passed':False,'status':'running','profiles':self.rows},indent=2)+'\n')

def main():
 if '--inside' in sys.argv:
  probe=Probe();common=module('admission_fixture',Path('/lab/admission-fixture.py'));common.inside(hooks=probe)
  Path('/lab/native-bar-result.json').write_text(json.dumps({'passed':True,'profiles':probe.rows,'closed_admission':probe.closed,'shell_facades':probe.facades,'full_native_bar':False,'actual_native_bar_methods':True,'actual_native_plugin_bar_api':True},indent=2)+'\n');return
 shell_api='--shell-api' in sys.argv
 if shell_api:sys.argv.remove('--shell-api')
 source=Path(__file__).resolve().parents[1];scripts=source/'scripts';actual=module('actual_native',scripts/'test-collection-omabit-guarded-native.py')
 if '--actual-pair' not in sys.argv:sys.argv.append('--actual-pair')
 import subprocess
 original=subprocess.run
 def launch(command,*args,**kw):
  assert command[0]=='bwrap';out=Path(command[command.index('--bind')+1]);common=module('common_hash',scripts/'test-collection-admission-native.py')
  native=Path('/usr/share/omarchy/shell/plugins/bar/Bar.qml');text=native.read_text();methods=[]
  for name in ('panelNavigationSlots','switchPanelFrom','moduleWidgets','requestPopout','releasePopout'):
   found=re.search(r'^  function '+name+r'\([^\n]*\) \{\n.*?^  \}',text,re.M|re.S);assert found;methods.append(found[0])
  (out/'native-bar-methods.qml.txt').write_text('\n'.join(methods))
  support=out/'bar-support';support.mkdir();api=Path('/usr/share/omarchy/shell/Ui/PluginBarApi.qml');shutil.copy2(api,support/api.name);(support/'qmldir').write_text('PluginBarApi 1.0 PluginBarApi.qml\n')
  path=out/'shell.qml';s=path.read_text().replace('import QtQuick\n','import QtQuick\nimport "bar-support" as NativeBar\n',1)
  old='onLoaded:{item.moduleName=entry.modelData.id;var next=';assert old in s;s=s.replace(old,'onLoaded:{item.moduleName=entry.modelData.id;item.bar=root.apiFor(entry.modelData.id);var next=')
  s=s.rstrip();assert s.endswith('}');path.write_text(s[:-1]+QML.replace('NATIVE_BAR_METHODS','\n'.join(methods))+'\n}\n')
  h=common.read(out/'source.json');h[str(native)]=common.sha(native);h[str(api)]=common.sha(api);h[str(Path(__file__))]=common.sha(Path(__file__))
  if shell_api:
   extra=scripts/'qualification/collections/native_shell_api_fixture.py';module('shell_api_fixture',extra).prepare(out,h);h[str(extra)]=common.sha(extra)
  common.save(out/'source.json',h)
  common.save(out/'native-bar-result.json',{'passed':False,'status':'starting'});return original(command,*args,**kw)
 subprocess.run=launch;actual.__file__=__file__;actual.main()

if __name__=='__main__':main()
