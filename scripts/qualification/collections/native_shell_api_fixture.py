"""Copy the native installed-plugin shell facade and its exact callbacks into lab."""
from pathlib import Path
import hashlib,json,re,shutil

METHODS=('publicPluginManifest','publicBarConfig','manifestHasKind','pluginHasBarCapabilities','pluginIsIndicatorsClone','publicIdleConfigFor','pluginCloneMaySummon','pluginOwnsTarget','pluginServiceFor','barEntryConfigured','barPluginMayControl','pluginBarStateFor','pluginShellCapabilityProfile','cacheWithoutKey','cacheWithoutPrefix','revokePluginShellApi','createScopedPluginShell','scopedPluginShellForId','pluginShellForId','pluginShellFor','isBarWidgetPanelPlugin','summon','hide','isPluginOpen','toggle','deliverIfLoaded','invokeIfLoaded')
QML=r'''
 property var _pluginShellApis:({})
 property var _pluginShellApiDescriptors:({})
 property var _pluginBarStateApis:({})
 property var _pluginAppLibraryApis:({})
 property var _pluginFirstPartyServiceApis:({})
 property var _pluginBarEntryShellApis:({})
 property var shellConfig:config
 property var barConfig:config.bar
 property var bar:null
 property var pendingPayloads:({})
 property var panelLoaders:panels
 Component {id:pluginShellApiComponent;NativeShell.PluginShellApi{}}
 Component {id:pluginBarStateApiComponent;NativeShell.PluginBarStateApi{}}
 NATIVE_SHELL_METHODS
 IpcHandler {
  target:"shellFacadeProbe"
  function state():string {
   var result={}
   var hosts=[];for(var id in shell._services)hosts.push(shell._services[id]);for(var id in shell.panels)if(shell.panels[id].item)hosts.push(shell.panels[id].item)
   for(var i=0;i<hosts.length;i++){
    var h=hosts[i],p=h.payload;if(!p)continue;var a=p.shell
    var foreign=hosts[(i+1)%hosts.length].pluginId
    result[h.pluginId]={scoped:a!==shell && a.pluginId===h.pluginId,
     foreignServiceRefused:a.serviceFor(foreign)===null,
     foreignSummonRefused:a.summon(foreign,"{}")===false,
     foreignHideRefused:a.hide(foreign)===false,
     genericConfigRefused:a.mutateShellConfig(function(c){c.unexpected=true})===false,
     ownService:h.role==="dms" ? a.serviceFor(h.pluginId)===null : a.serviceFor(h.pluginId)===h,
     noHostMarkers:!Object.prototype.hasOwnProperty.call(p.manifest,"__isFirstParty") && !Object.prototype.hasOwnProperty.call(p.manifest,"__hostCapabilities")}
   }
   return JSON.stringify(result)
  }
  function overlay(id:string):string {
   var h=shell.panels[id].item,p=h.payload,a=p.shell
   var summoned=a.summon(id,"{}");var opened=h.opened;p.dismiss()
   return JSON.stringify({summoned:summoned,opened:opened,dismissed:!h.opened,registryClosed:shell.openPanelIds[id]!==true})
  }
 }
'''

def prepare(out,hashes,*,bar_fixture=True):
 native=Path('/usr/share/omarchy/shell');path=native/'shell.qml';text=path.read_text();methods=[]
 for name in METHODS:
  found=re.search(r'^  function '+name+r'\([^\n]*\) \{\n.*?^  \}',text,re.M|re.S);assert found,name;methods.append(found[0])
 (out/'native-shell-methods.qml.txt').write_text('\n\n'.join(methods))
 support=out/'shell-support';support.mkdir()
 for name in ('PluginShellApi','PluginBarStateApi'):
  p=native/'services'/(name+'.qml');shutil.copy2(p,support/p.name);hashes[str(p)]=hashlib.sha256(p.read_bytes()).hexdigest()
 (support/'qmldir').write_text('PluginShellApi 1.0 PluginShellApi.qml\nPluginBarStateApi 1.0 PluginBarStateApi.qml\n')
 p=out/'shell.qml';s=p.read_text().replace('import QtQuick\n','import QtQuick\nimport "shell-support" as NativeShell\nimport qs.Commons\n',1)
 for old in (' function pluginShellFor(m){return shell}\n',' function publicPluginManifest(m){return JSON.parse(JSON.stringify(m))}\n',' function hide(id){}\n'):
  assert old in s;s=s.replace(old,'')
 # Actual PluginBarApi receives the same service-capable scoped shell it would
 # receive from the trusted native bar. No host ShellRoot reaches the payload.
 if bar_fixture:
  old='pluginId:id,moduleName:id,fontFamily:';assert old in s;s=s.replace(old,'pluginId:id,moduleName:id,shell:shell.pluginShellForId(id),fontFamily:')
 s=s.rstrip();assert s.endswith('}');p.write_text(s[:-1]+QML.replace('NATIVE_SHELL_METHODS','\n\n'.join(methods))+'\n}\n')
 hashes[str(path)]=hashlib.sha256(path.read_bytes()).hexdigest()
 (out/'native-shell-api-mode.json').write_text(json.dumps({'actual_shell_api':True,'actual_callbacks':list(METHODS),'full_shell':False})+'\n')
