#!/usr/bin/env python3
"""Private native registry/IPC + immutable artifact identity qualification.

Uses the complete copied PluginRegistry and exact native service/panel loaders.
Small synthetic plugin bodies isolate loader identity; this is not the complete
shell, real plugin application behavior, a compositor or durable admission proof.
"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import socket
import subprocess
import sys
import time

NATIVE=Path('/usr/share/omarchy/shell')
FUNCTIONS=('ensureService','_syncServices','serviceKeepLoaded','unloadPluginServices','computePanelEntries')
QML='''import QtQuick
import Quickshell
import Quickshell.Io
import "AuthServiceStore.js" as AuthServiceStore
ShellRoot {
 id: shell
 property var _services: ({})
 property var panels: ({})
 property var openPanelIds: ({})
 property var panelEntries: []
 property string omarchyPath: "/lab"
 property var config: ({version:1,plugins:[],bar:{layout:{left:[],center:[],right:[]}}})
 property var pluginRegistry: PluginRegistry {
  shellConfigProvider: function(){return shell.config}
  shellConfigMutator: function(fn){var next=JSON.parse(JSON.stringify(shell.config));fn(next);shell.config=next}
 }
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
 Connections {
  target:shell.pluginRegistry
  function onPluginsChanged(){shell._syncServices();shell.panelEntries=shell.computePanelEntries()}
 }
 function describe(){
  var selected=({}),loaded=({})
  for(var id in pluginRegistry.installedPlugins){var m=pluginRegistry.installedPlugins[id];selected[id]={version:m.version,enabled:pluginRegistry.isEnabled(id),entryPoints:m.entryPoints,source:m.__sourceDir}}
  for(var id in _services){var o=_services[id];loaded[id]={identity:o.collectionObservation(),code:o.fixtureVersion,helper:o.fixtureHelper,selected:o.manifest.version}}
  for(var id in panels){var o=panels[id].item;if(o)loaded[id]={identity:o.collectionObservation(),code:o.fixtureVersion,helper:o.fixtureHelper,selected:o.manifest.version}}
  return JSON.stringify({selected:selected,loaded:loaded,scanning:pluginRegistry.scanning})
 }
 IpcHandler {
  target:"shell"
  NATIVE_ENABLE
 }
 IpcHandler {
  target:"artifactProbe"
  function state():string{return shell.describe()}
  function rescan():string{shell.pluginRegistry.rescan();return "requested"}
  function stop():string{Qt.callLater(Qt.quit);return "stopping"}
 }
 METHODS
 PANELS
}
'''
BODY='''import QtQuick
import Quickshell.Io
import "Marker.js" as Marker
Item {
 id: root
 property var manifest: null
 property var shell: null
 property string profileName: "default"
 property var profileEnvironment: ({})
 property url toolsDirectory
 property bool localSetupAllowed: true
 readonly property int fixtureVersion: Marker.version
 property string fixtureHelper: ""
 property var transportState: ({closed:false})
 function stopTransport(){transportState={closed:true};return "requested"}
 Process {
  command:["python3","-B",Qt.resolvedUrl("helper.py").toString().replace(/^file:\\/\\//, "")]
  running:true
  stdout: StdioCollector {onStreamFinished:root.fixtureHelper=text.trim()}
 }
}
'''


def sha(path):return hashlib.sha256(Path(path).read_bytes()).hexdigest()
def save(path,value):Path(path).write_text(json.dumps(value,indent=2)+'\n')


def inside():
 root=Path('/lab');host=json.loads((root/'host-namespaces.json').read_text())
 assert os.environ.get('OMABIT_NATIVE_ARTIFACT_PROBE')=='private-v1' and Path.home()==root/'home'
 assert not Path('/home').exists() and not Path('/run/user').exists()
 assert all(os.readlink('/proc/self/ns/'+k)!=v for k,v in host.items()) and [n for _,n in socket.if_nameindex()]==['lo']
 sys.path.insert(0,str(root/'source'));import artifacts as a
 prepared=json.loads((root/'prepared.json').read_text());events=[]
 with (root/'shell.log').open('wb') as log:
  proc=subprocess.Popen(['/usr/bin/quickshell','--no-color','--path','/lab/shell.qml'],stdout=log,stderr=subprocess.STDOUT)
  try:
   def call(target,method,*args,required=True):
    result=subprocess.run(['/usr/bin/quickshell','ipc','--pid',str(proc.pid),'call','--',target,method,*args],capture_output=True,text=True,timeout=4)
    if required:assert result.returncode==0,(target,method,result.stderr)
    return result.stdout.strip()
   def wait(fn):
    deadline=time.monotonic()+8
    while time.monotonic()<deadline:
     assert proc.poll() is None,'private shell exited'
     value=fn()
     if value:return value
     time.sleep(.04)
    raise AssertionError('native artifact fixture deadline')
   def state():return json.loads(call('artifactProbe','state'))
   wait(lambda:call('artifactProbe','state',required=False).startswith('{'))
   wait(lambda:len(state()['selected'])==9 and not state()['scanning'])
   def enable(id,value):assert call('shell','setPluginEnabled',id,str(value).lower())=='ok'
   def inspect(record,version):
    ident=json.loads(call(record['manifest']['id']+'.collection','identity'))
    row=state()['loaded'].get(record['manifest']['id'])
    assert row and row['identity']==ident and row['code']==version and row['helper']==str(version)
    assert ident['artifact_sha256']==record['artifact_sha256'] and ident['profile']==record['descriptor']['profile']
    assert ident['plugin_id']==record['manifest']['id'] and re.fullmatch('[0-9a-f]{32}',ident['instance'])
    assert ident['source']=='file:///lab/home/.config/omarchy/plugins/'+ident['plugin_id']+'/generations/'+ident['artifact_sha256']+'/CollectionRuntime.qml'
    actual=root/'home/.config/omarchy/plugins'/ident['plugin_id']/'generations'/record['artifact_sha256']
    bound=a.observation(actual,record['artifact_sha256'],ident)
    assert bound=={k:ident[k] for k in ('plugin_id','profile','artifact_sha256','instance')}
    return ident
   # A live kept service must report the old executable generation even after
   # the real native registry has scanned the newer selected manifest.
   old=prepared['stable']['theme']['1'];new=prepared['stable']['theme']['2'];id=old['manifest']['id']
   enable(id,True);wait(lambda:state()['loaded'].get(id,{}).get('helper')=='1');before=inspect(old,1)
   path=root/'home/.config/omarchy/plugins'/id/'manifest.json';save(path,new['manifest'])
   call('artifactProbe','rescan');wait(lambda:state()['selected'][id]['version']=='2.0.0' and not state()['scanning'])
   after=inspect(old,1);assert before==after and state()['loaded'][id]['selected']=='2.0.0'
   try:a.observation(root/'home/.config/omarchy/plugins'/id/'generations'/new['artifact_sha256'],new['artifact_sha256'],after)
   except a.j.Conflict:pass
   else:raise AssertionError('selected manifest falsely accepted as new loaded generation')
   events.append({'case':'native-rescan-keeps-loaded-identity','before':before,'after':after,'selected':state()['selected'][id]})
   assert call(id+'.collection','drain')=='requested';assert json.loads(call(id+'.collection','transport'))['closed']
   enable(id,False);wait(lambda:not state()['loaded'])
   enable(id,True);wait(lambda:state()['loaded'].get(id,{}).get('helper')=='2');replacement=inspect(new,2)
   assert replacement['instance']!=before['instance'];enable(id,False);wait(lambda:not state()['loaded'])
   events.append({'case':'native-disable-enable-distinct-url','old':before,'new':replacement})
   # Separate every outgoing disable from the next enable; panel and service
   # loader paths both use native setEnabled and actual registry discovery.
   prior=[]
   for profile in ('stable','dev','stable','edge','stable'):
    current=[]
    for role in ('omabit','theme','dms'):
     version=2 if profile=='stable' and role=='theme' else 1
     record=prepared[profile][role][str(version)];id=record['manifest']['id'];enable(id,True);current.append((record,version))
    wait(lambda:len(state()['loaded'])==3 and all(state()['loaded'].get(r['manifest']['id'],{}).get('helper')==str(v) for r,v in current))
    observed=[inspect(r,v) for r,v in current];assert not {x['instance'] for x in prior}&{x['instance'] for x in observed}
    for r,v in current:
     id=r['manifest']['id'];assert call(id+'.collection','drain')=='requested';assert json.loads(call(id+'.collection','transport'))['closed'];enable(id,False)
    wait(lambda:not state()['loaded']);events.append({'case':'profile-roundtrip','profile':profile,'loaded':observed});prior=observed
   call('artifactProbe','stop');assert proc.wait(timeout=4)==0
  finally:
   if proc.poll() is None:
    proc.terminate()
    try:proc.wait(timeout=3)
    except subprocess.TimeoutExpired:proc.kill();proc.wait(timeout=3)
 text=(root/'shell.log').read_text()
 assert not any(x in text for x in ('ReferenceError','TypeError','Binding loop','ERROR:','failed to load','returned null','service plugin load failed')),text[-4000:]
 save(root/'result.json',{'passed':True,'events':events,'complete_native_registry':True,'native_set_enabled_ipc':True,'native_service_panel_loaders':True,'synthetic_plugin_bodies':True,'full_native_shell':False,'real_plugins':False,'durable_admission':False})
 print('PASS: native discovery, IPC, immutable loaded identity and five profile roundtrips')


def main():
 p=argparse.ArgumentParser(description=__doc__);p.add_argument('--inside',action='store_true');p.add_argument('--output',type=Path);args=p.parse_args()
 if args.inside:return inside()
 out=args.output.absolute() if args.output else None
 if out is None or out.parent!=Path('/tmp') or not re.fullmatch('omabit-native-artifacts-[A-Za-z0-9-]+',out.name) or out.exists():p.error('fresh /tmp/omabit-native-artifacts-NAME required')
 out.mkdir(mode=0o700)
 for name in ('home','runtime','config','state','data','cache','inputs','source','imports'):(out/name).mkdir(mode=0o700)
 source=Path(__file__).resolve().parents[1];sys.path.insert(0,str(source/'native/collections'));import artifacts as a
 source_hashes={}
 for name in ('artifacts.py','journal.py'):
  path=source/'native/collections'/name;shutil.copy2(path,out/'source'/name);source_hashes[str(path)]=sha(path)
 for path,target in ((NATIVE/'services/PluginRegistry.qml',out/'PluginRegistry.qml'),(NATIVE/'Commons/Util.qml',out/'imports/qs/Commons/Util.qml')):
  target.parent.mkdir(parents=True,exist_ok=True);shutil.copy2(path,target);source_hashes[str(path)]=sha(path)
 (out/'imports/qs/Commons/qmldir').write_text('module qs.Commons\nsingleton Util 1.0 Util.qml\n')
 native=(NATIVE/'shell.qml').read_text();source_hashes[str(NATIVE/'shell.qml')]=sha(NATIVE/'shell.qml');methods=[]
 for name in FUNCTIONS:
  match=re.search(r'^  function '+name+r'\([^\n]*\) \{\n.*?^  \}',native,re.M|re.S);assert match;methods.append(match[0])
 method=re.search(r'^    function setPluginEnabled\([^\n]*\): string \{\n.*?^    \}',native,re.M|re.S);assert method
 start=native.index('  Instantiator {',native.index('  function computePanelEntries()'));end=native.index('\n  // ---------------------------------------------------------- plugin loader',start);panel=native[start:end].strip()
 (out/'native-methods.qml.txt').write_text('\n\n'.join(methods)+panel+method[0])
 (out/'shell.qml').write_text(QML.replace('METHODS','\n\n'.join(methods)).replace('PANELS',panel).replace('NATIVE_ENABLE',method[0]))
 (out/'AuthServiceStore.js').write_text('.pragma library\nfunction has(id){return false}\nfunction ids(){return []}\nfunction destroy(id){throw new Error("unexpected auth service")}\n')
 prepared={}
 for profile in ('stable','dev','edge'):
  prepared[profile]={}
  for role in a.ROLES:
   prepared[profile][role]={};plugin_id=a.ROLES[role][0]+'.qualification-'+profile
   plugin=out/'home/.config/omarchy/plugins'/plugin_id;store=plugin/'generations';store.mkdir(parents=True,mode=0o700)
   for version in (1,2) if role=='theme' and profile=='stable' else (1,):
    base=out/'inputs'/(role+'-'+profile+'-'+str(version));base.mkdir();canonical,kind,entry=a.ROLES[role]
    manifest={'schemaVersion':1,'id':canonical,'name':'Synthetic '+role,'version':str(version)+'.0.0','kinds':[kind],'keepLoaded':True,'entryPoints':{kind:entry}}
    save(base/'manifest.json',manifest);(base/entry).write_text(BODY);(base/'Marker.js').write_text('.pragma library\nvar version = '+str(version)+'\n');(base/'helper.py').write_text('print('+str(version)+')\n')
    if role=='omabit':
     (base/'bin').mkdir();save(base/'DESKTOP.json',{'bootstrap_api':1,'profile_state_schema':1})
     for name in ('omabit','urbit-collector','urbit-runner'):
      path=base/'bin'/name;path.write_text('#!/bin/sh\nexit 0\n');path.chmod(0o700)
    names=sorted(str(x.relative_to(base)) for x in base.rglob('*') if x.is_file());pin=a.inventory(a.snapshot(base,names))
    environment={f'XDG_{k.upper()}_HOME':'/lab/'+k+'/'+role+'/'+profile for k in ('config','state','data','cache')}
    record=a.prepare(base,pin,store,role=role,profile='qualification-'+profile,environment=environment if role=='omabit' else None)
    prepared[profile][role][str(version)]=record
    if version==1:save(plugin/'manifest.json',record['manifest'])
 save(out/'prepared.json',prepared);save(out/'source.json',source_hashes);save(out/'host-namespaces.json',{n:os.readlink('/proc/self/ns/'+n) for n in ('user','net','pid','mnt')});shutil.copy2(__file__,out/'driver.py')
 cmd=['bwrap','--unshare-all','--die-with-parent','--clearenv','--ro-bind','/usr','/usr','--symlink','usr/bin','/bin','--symlink','usr/lib','/lib','--symlink','usr/lib','/lib64','--proc','/proc','--dev','/dev','--tmpfs','/tmp','--bind',str(out),'/lab','--chdir','/lab']
 env={'PATH':'/usr/bin:/bin','HOME':'/lab/home','XDG_RUNTIME_DIR':'/lab/runtime','QML_IMPORT_PATH':'/lab/imports','QT_QPA_PLATFORM':'offscreen','QT_QUICK_BACKEND':'software','QML_DISABLE_DISK_CACHE':'1','OMABIT_NATIVE_ARTIFACT_PROBE':'private-v1','PYTHONDONTWRITEBYTECODE':'1','LANG':'C.UTF-8'}
 for n in ('CONFIG','DATA','STATE','CACHE'):env['XDG_'+n+'_HOME']='/lab/'+n.lower()
 for k,v in env.items():cmd+=['--setenv',k,v]
 result=subprocess.run(cmd+['--','/usr/bin/python3','-B','/lab/driver.py','--inside'],capture_output=True,text=True,timeout=75)
 (out/'driver.log').write_text(result.stdout+result.stderr);print(result.stdout+result.stderr);raise SystemExit(result.returncode)


if __name__=='__main__':main()
