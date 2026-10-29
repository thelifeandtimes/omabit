"""Native shell API pruning inside the private real-bar qualification fixture."""
from pathlib import Path
import hashlib,json,re,shutil

METHODS=('publicBarWidgetSnapshot','pluginApiActive','prunePluginApis','syncPluginApis')
QML=r'''
 property var _pluginRegistryApis:({})
 property var _pluginBarWidgetRegistryApis:({})
 NATIVE_API_METHODS
 Connections {target:shell.barWidgetRegistry;function onChanged(){shell.syncPluginApis()}}
 IpcHandler {
  target:"apiLifecycleProbe"
  function state():string {
   var apis={};for(var key in shell._pluginShellApis){var a=shell._pluginShellApis[key];apis[key]={pluginId:a.pluginId,instance:a.fixtureInstance}}
   var references={};for(var k in nativeBar.pluginBarApis){var a=nativeBar.pluginBarApis[k],ref=a.shell,row={usable:false,absent:!ref,deletedReference:false};try{if(ref){row.id=String(ref.pluginId);row.usable=typeof ref.serviceFor==="function"}}catch(error){row.deletedReference=true}references[k]=row}
   return JSON.stringify({references:references,apis:apis,descriptors:Object.keys(shell._pluginShellApiDescriptors),barState:Object.keys(shell._pluginBarStateApis),nativeBar:Object.keys(nativeBar.pluginBarApis),placeholderOnly:Object.keys(nativeBar.pluginBarApis).every(function(key){return key.indexOf("bar-entry:")===0 && !references[key].usable}),objects:nativeBar.pluginObjectOwners.size,clickTargets:nativeBar.clickTargets.length,popout:!!nativeBar.activePopout})
  }
  function pruneUnusedBar():void{nativeBar.prunePluginBarApis()}
 }
'''

def prepare(out,hashes):
 native=Path('/usr/share/omarchy/shell/shell.qml');text=native.read_text();methods=[]
 for name in METHODS:
  found=re.search(r'^  function '+name+r'\([^\n]*\) \{\n.*?^  \}',text,re.M|re.S);assert found,name;methods.append(found[0])
 p=out/'shell.qml';s=p.read_text();old='function onPluginsChanged(){shell._syncServices();shell.panelEntries=shell.computePanelEntries()}'
 assert old in s;s=s.replace(old,'function onPluginsChanged(){shell.syncPluginApis();shell._syncServices();shell.panelEntries=shell.computePanelEntries()}')
 s=s.rstrip();assert s.endswith('}');p.write_text(s[:-1]+QML.replace('NATIVE_API_METHODS','\n\n'.join(methods))+'\n}\n')
 p=out/'shell-support/PluginShellApi.qml';s=p.read_text();assert 'Component.onCompleted' not in s and 'Component.onDestruction' not in s
 old='  id: api';assert old in s;s=s.replace(old,old+'''
  readonly property string fixtureInstance: Date.now().toString(16)+"-"+Math.random().toString(16).slice(2)
  Component.onCompleted: console.log("SCOPED_API_CREATED "+pluginId+" "+fixtureInstance)
  Component.onDestruction: console.log("SCOPED_API_DESTROYED "+pluginId+" "+fixtureInstance)
''');p.write_text(s)
 hashes[str(native)]=hashlib.sha256(native.read_bytes()).hexdigest()
 (out/'native-api-lifecycle-mode.json').write_text(json.dumps({'methods':list(METHODS),'full_native_shell':False,'diagnostic_api_lifetime_markers':True})+'\n')
