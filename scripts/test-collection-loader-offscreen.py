#!/usr/bin/env python3
"""Characterize copied Omarchy service-loader methods in an isolated QML process."""
import argparse,hashlib,json,os,re,shutil,signal,subprocess,time
from pathlib import Path
SHELL=Path('/usr/share/omarchy/shell/shell.qml')
FUNCTIONS=('ensureService','_syncServices','serviceKeepLoaded','unloadPluginServices')
QML=r'''
import QtQuick
import Quickshell
import Quickshell.Io
import "AuthServiceStore.js" as AuthServiceStore
ShellRoot {
 id: shell
 property var _services: ({})
 property var events: []
 property string omarchyPath: "/lab"
 property var registry: ({installedPlugins: {}, enabled: {}, isEnabled: function(id){return !!shell.registry.enabled[id]}, entryPointUrl:function(manifest,kind){return Qt.resolvedUrl(manifest.entryPoints.service)}})
 property var pluginRegistry: registry
 Item { id: serviceHost }
 function isAuthenticationService(manifest,id) { return false }
 function pluginShellFor(manifest) { return shell }
 function publicPluginManifest(manifest) { return JSON.parse(JSON.stringify(manifest)) }
 function pluginBarWidgetRegistryFor(manifest) { return null }
 function pluginRegistryFor(manifest) { return null }
 function note(event) { events=events.concat([event]);console.log("EVENT "+event) }
 function select(id,version) {
   var plugins=registry.installedPlugins
   plugins[id]={id:id,kinds:["service"],keepLoaded:true,version:version,entryPoints:{service:version===2?"ServiceV2.qml":"Service.qml"}}
   registry.installedPlugins=plugins;registry.enabled=({});registry.enabled[id]=true
   _syncServices()
 }
 function snapshot() {
  var services=({})
  for(var id in _services) { var item=_services[id];services[id]={loaded:item.buildMarker,selected:item.manifest.version} }
  return JSON.stringify({services:services,events:events})
 }
 IpcHandler {
  target: "omabitCollectionProbe"
  function select(id:string):string { shell.select(id,1);return shell.snapshot() }
  function selectV2(id:string):string { shell.select(id,2);return shell.snapshot() }
  function reloadDev():string {
   var m=shell.registry.installedPlugins["dev"];m.version=2;m.entryPoints.service="ServiceV2.qml"
   shell.unloadPluginServices();shell._syncServices();return shell.snapshot()
  }
  function disableAll():string { shell.registry.enabled=({});shell._syncServices();return shell.snapshot() }
  function state():string { return shell.snapshot() }
  function stop():string { Qt.callLater(Qt.quit);return "stopping" }
 }
 FUNCTIONS
}
'''
SERVICE=r'''
import QtQuick
import Quickshell.Io
Item {
 property var shell: null
 property var manifest: null
 property int buildMarker: BUILD
 property string startedId: ""
 onManifestChanged: {
  if(manifest && startedId==="") {
   startedId=manifest.id;shell.note("create:"+startedId)
   child.command=["python3","-B","/lab/child.py",startedId];child.running=true
  }
 }
 Component.onDestruction: { if(shell && startedId!=="")shell.note("destroy:"+startedId) }
 Process { id:child }
}
'''
CHILD=r'''
import os,signal,sys,time
from pathlib import Path
name=sys.argv[1];assert name in ('stable','dev','edge')
p=Path('/lab')/(name+'.pid');p.write_text(str(os.getpid()))
def end(*_):
 Path('/lab/'+name+'.terminated').write_text(str(os.getpid()));raise SystemExit(0)
signal.signal(signal.SIGTERM,end)
while True:time.sleep(.1)
'''
def inside():
 if (os.environ.get('OMABIT_COLLECTION_LOADER_PROBE')!='private-v1' or os.environ.get('HOME')!='/lab/home' or os.environ.get('XDG_RUNTIME_DIR')!='/lab/runtime' or not Path('/lab/marker').is_file() or Path('/home').exists() or Path('/run/user').exists()):raise RuntimeError('private namespace required')
 root=Path('/lab');events=[]
 def alive(pid):
  try:
   s=Path('/proc/'+str(pid)+'/stat').read_text();return s.split(') ',1)[1][0]!='Z'
  except FileNotFoundError:return False
 with (root/'shell.log').open('wb') as out:
  proc=subprocess.Popen(['/usr/bin/quickshell','--no-color','--path','/lab/shell.qml'],stdout=out,stderr=subprocess.STDOUT,start_new_session=True)
  try:
   def call(method,*args,require=True):
    value=subprocess.run(['/usr/bin/quickshell','ipc','--pid',str(proc.pid),'call','--','omabitCollectionProbe',method,*args],capture_output=True,text=True,timeout=4)
    if require:assert value.returncode==0,(method,value.stderr[-1000:])
    return value
   def wait(fn,seconds=8):
    end=time.monotonic()+seconds
    while time.monotonic()<end:
     assert proc.poll() is None,'private shell exited'
     result=fn()
     if result:return result
     time.sleep(.04)
    raise AssertionError('private loader deadline')
   wait(lambda:call('state',require=False).stdout.strip().startswith('{'))
   def start(name):
    previous=int((root/(name+'.pid')).read_text()) if (root/(name+'.pid')).exists() else 0
    snapshot=json.loads(call('select',name).stdout)
    def ready():
     try:p=int((root/(name+'.pid')).read_text());return p if p!=previous and alive(p) else 0
     except FileNotFoundError:return 0
    return wait(ready),snapshot
   stable,first=start('stable');dev,atomic=start('dev')
   after=json.loads(call('state').stdout);wait(lambda:not alive(stable))
   assert after['events'].index('create:dev')<after['events'].index('destroy:stable')
   events.append({'case':'one-pass-switch','result':'new QML service created before old QML service destroyed','snapshot':after})
   kept=json.loads(call('reloadDev').stdout);assert kept['services']['dev']=={'loaded':1,'selected':2} and int((root/'dev.pid').read_text())==dev and alive(dev)
   events.append({'case':'keepLoaded-reload','result':'manifest changed but original service/helper retained','snapshot':kept})
   call('disableAll');wait(lambda:not alive(dev));assert not json.loads(call('state').stdout)['services']
   replaced=json.loads(call('selectV2','dev').stdout)
   pid=wait(lambda: int((root/'dev.pid').read_text()) if int((root/'dev.pid').read_text())!=dev and alive(int((root/'dev.pid').read_text())) else 0)
   assert replaced['services']=={'dev':{'loaded':2,'selected':2}}
   events.append({'case':'explicit-generation-replacement','old_process_exited_before_enable':True,'snapshot':replaced})
   current='dev'
   for target in ('stable','dev','stable','edge','stable'):
    call('disableAll');wait(lambda:not alive(pid));drained=json.loads(call('state').stdout);assert not drained['services']
    old=pid;pid,snapshot=start(target);assert not alive(old) and snapshot['services']=={target:{'loaded':1,'selected':1}}
    events.append({'case':'separate-disable-and-enable','from':current,'to':target,'old_process_exited_before_enable':True,'snapshot':snapshot});current=target
   call('disableAll');wait(lambda:not alive(pid));assert not json.loads(call('state').stdout)['services']
   assert call('stop').stdout.strip()=='stopping';assert proc.wait(timeout=3)==0
  finally:
   if proc.poll() is None:
    os.killpg(proc.pid,signal.SIGTERM)
    try:proc.wait(timeout=2)
    except subprocess.TimeoutExpired:os.killpg(proc.pid,signal.SIGKILL);proc.wait(timeout=2)
 text=(root/'shell.log').read_text();assert not any(k in text for k in ('ReferenceError','TypeError','Binding loop','ERROR:'))
 (root/'result.json').write_text(json.dumps({'passed':True,'events':events,'full_shell':False,'real_plugins':False,'actual_compositor':False,'copied_native_functions':list(FUNCTIONS),'real_quickshell_processes':True},indent=2)+'\n');print('PASS: loader ordering, retained generation and explicit drain boundary')
def main():
 p=argparse.ArgumentParser(description=__doc__);p.add_argument('--output',type=Path);p.add_argument('--inside',action='store_true');a=p.parse_args()
 if a.inside:return inside()
 if not a.output:p.error('--output is required')
 out=a.output.absolute()
 if out.parent!=Path('/tmp') or not re.fullmatch('omabit-collection-loader-probe-[A-Za-z0-9-]+',out.name) or out.exists():p.error('fresh /tmp/omabit-collection-loader-probe-NAME required')
 out.mkdir(mode=0o700)
 for n in ('home','runtime','cache','config','data','state'):(out/n).mkdir(mode=0o700)
 source=SHELL.read_text();methods=[]
 for name in FUNCTIONS:
  match=re.search(r'^  function '+re.escape(name)+r'\([^\n]*\) \{\n.*?^  \}',source,re.M|re.S);assert match,name;methods.append(match[0])
 (out/'native-methods.qml.txt').write_text('\n\n'.join(methods));(out/'shell.qml').write_text(QML.replace('FUNCTIONS','\n'.join(methods)))
 for version,n in ((1,'Service.qml'),(2,'ServiceV2.qml')):(out/n).write_text(SERVICE.replace('BUILD',str(version)))
 (out/'AuthServiceStore.js').write_text('.pragma library\nfunction has(id){return false}\nfunction ids(){return []}\nfunction destroy(id){throw new Error("unexpected authentication service")}\n')
 (out/'child.py').write_text(CHILD);(out/'marker').write_text('private-v1\n');shutil.copy2(__file__,out/'driver.py')
 files=[SHELL,Path('/usr/bin/quickshell'),Path('/usr/share/omarchy/bin/omarchy-shell')]
 (out/'source.json').write_text(json.dumps({'sha256':{str(f):hashlib.sha256(f.read_bytes()).hexdigest() for f in files},'functions':list(FUNCTIONS),'scope':'copied exact service loader methods; synthetic third-party manifests/services and helpers; no full shell or plugin authority claim'},indent=2)+'\n')
 cmd=['bwrap','--unshare-all','--die-with-parent','--clearenv','--ro-bind','/usr','/usr','--symlink','usr/bin','/bin','--symlink','usr/lib','/lib','--symlink','usr/lib','/lib64','--proc','/proc','--dev','/dev','--tmpfs','/tmp','--bind',str(out),'/lab','--chdir','/lab']
 env={'PATH':'/usr/bin:/bin','HOME':'/lab/home','XDG_RUNTIME_DIR':'/lab/runtime','XDG_CONFIG_HOME':'/lab/config','XDG_DATA_HOME':'/lab/data','XDG_STATE_HOME':'/lab/state','XDG_CACHE_HOME':'/lab/cache','QT_QPA_PLATFORM':'offscreen','QT_QUICK_BACKEND':'software','QML_DISABLE_DISK_CACHE':'1','OMABIT_COLLECTION_LOADER_PROBE':'private-v1','PYTHONDONTWRITEBYTECODE':'1'}
 for k,v in env.items():cmd+=['--setenv',k,v]
 result=subprocess.run(cmd+['--','/usr/bin/python3','-B','/lab/driver.py','--inside'],capture_output=True,text=True,timeout=55);(out/'driver.log').write_text(result.stdout+result.stderr)
 print(result.stdout+result.stderr);raise SystemExit(result.returncode)
if __name__=='__main__':main()
