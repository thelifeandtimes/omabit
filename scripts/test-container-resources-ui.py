#!/usr/bin/env python3
"""Render and exercise real resource controls with an inert service in private XDG roots."""
from pathlib import Path
import argparse,hashlib,importlib.util,json,shutil,subprocess
ROOT=Path(__file__).resolve().parents[1]
spec=importlib.util.spec_from_file_location('storage',ROOT/'scripts/qualification/app-development/storage.py');storage=importlib.util.module_from_spec(spec);spec.loader.exec_module(storage);storage.admit(8*1024**2)
p=argparse.ArgumentParser(description=__doc__);p.add_argument('--output',type=Path,required=True);out=p.parse_args().output.resolve();out.mkdir(mode=0o700)
plugin=out/'Plugin';plugin.mkdir()
for name in ('ShipPortsPane.qml','ContainerResourceFields.qml','RuntimeSettingsFields.qml','SelectField.qml','Label.qml','Design.qml','ActionButton.qml','Field.qml','Hint.qml','Model.js'):
 shutil.copy2(ROOT/'plugins/omabit'/name,plugin/name)
(plugin/'qmldir').write_text('singleton Design 1.0 Design.qml\n')
commons=out/'qs/Commons';commons.mkdir(parents=True)
(commons/'qmldir').write_text('module qs.Commons\nsingleton Style 1.0 Style.qml\nsingleton Color 1.0 Color.qml\n')
(commons/'Style.qml').write_text('pragma Singleton\nimport QtQuick\nQtObject { function space(n) { return n } property var font: ({family:"sans-serif",caption:12,bodySmall:14,body:14}) }')
(commons/'Color.qml').write_text('pragma Singleton\nimport QtQuick\nQtObject { property color background:"#181922"; property color foreground:"#ececf1";property color accent:"#9ea8f2";property color urgent:"#ed8f8f" }')
(out/'main.qml').write_text('''import QtQuick
import "Plugin" as P
Window {
 id:w;width:640;height:680;visible:true;color:P.Design.background
 property var connection: ({name:"fixture",runner_id:"runner-fixture",features:["runtime-settings","container-resources-v1"]})
 property var ship: ({id:"zod",mode:"fake",config_revision:3,desired_state:"stopped",observed_state:"stopped",config:{http_port:18080,ames_port:34343,loom:31}})
 property bool captured:false
 Column {
  anchors.fill:parent;anchors.margins:24;spacing:18
  P.Label {text:"~zod · Runtime settings";font.pixelSize:22}
  P.Label {text:"Stopped · Isolated UI fixture";secondary:true}
  Rectangle {width:parent.width;height:1;color:P.Design.line}
  P.ShipPortsPane {id:pane;width:parent.width;service:svc;connection:w.connection;instance:w.ship}
 }
 QtObject {
  id:svc;property var submitted:null;property string lastError:""
  function canConfigure(target){return !!target}
  function configure(target,port,loom,memory,cpus){submitted={target:target,port:port,loom:loom,memory:memory,cpus:cpus};return true}
 }
 function find(item,name){if(item.objectName===name)return item;var children=item.children||[];for(var i=0;i<children.length;i++){var v=find(children[i],name);if(v)return v}return null}
 function check(value,message){if(!value)throw new Error(message)}
 Timer {interval:150;running:true;repeat:true;property int phase:0;onTriggered:{try{
  var edit=w.find(pane,"edit-ship-ports"),ram=w.find(pane,"container-memory"),cpu=w.find(pane,"container-cpus"),save=w.find(pane,"save-ship-http-port")
  if(phase===0){edit.clicked();w.check(ram.enabled&&cpu.enabled,"stopped resources disabled");w.check(ram.currentIndex===3&&cpu.currentIndex===1,"historical defaults missing");ram.currentIndex=15;ram.activated(15);cpu.currentIndex=7;cpu.activated(7);phase=1}
  else if(phase===1){w.contentItem.grabToImage(function(image){w.captured=image.saveToFile(Qt.resolvedUrl("resources-wide.png").toString().slice(7))});phase=2}
  else if(phase===2&&w.captured){w.captured=false;w.width=420;phase=3}
  else if(phase===3){w.contentItem.grabToImage(function(image){w.captured=image.saveToFile(Qt.resolvedUrl("resources-narrow.png").toString().slice(7))});phase=4}
  else if(phase===4&&w.captured){save.clicked();w.check(svc.submitted.memory===16&&svc.submitted.cpus===8,"save lost limits");w.ship=Object.assign({},w.ship,{desired_state:"running",observed_state:"running"});edit.clicked();w.check(!ram.enabled&&!cpu.enabled,"running ship resources editable");phase=5}
  else if(phase===5){w.connection=Object.assign({},w.connection,{features:["runtime-settings"]});w.ship=Object.assign({},w.ship,{desired_state:"stopped",observed_state:"stopped"});edit.clicked();w.check(!ram.enabled&&!cpu.enabled,"old runner resources editable");console.log("PASS: real resource pane renders wide/narrow, submits limits and blocks running/old runners");Qt.quit()}
 }catch(e){console.error("FAIL: "+e.message);Qt.exit(1)}}}
 Timer {interval:6000;running:true;onTriggered:Qt.exit(2)}
}''')
env=dict(PATH='/usr/bin:/bin',LANG='C.UTF-8',QT_QPA_PLATFORM='offscreen',QT_QUICK_BACKEND='software',QML_IMPORT_PATH=str(out),QT_QPA_PLATFORMTHEME='',QT_QUICK_CONTROLS_STYLE='Basic',QT_FORCE_STDERR_LOGGING='1')
for key,leaf in [('HOME','home'),('XDG_RUNTIME_DIR','runtime'),('XDG_CONFIG_HOME','config'),('XDG_DATA_HOME','data'),('XDG_STATE_HOME','state'),('XDG_CACHE_HOME','cache')]:
 (out/leaf).mkdir(mode=0o700);env[key]=str(out/leaf)
cmd=['bwrap','--unshare-all','--die-with-parent','--clearenv','--ro-bind','/usr','/usr','--symlink','usr/bin','/bin','--symlink','usr/lib','/lib','--symlink','usr/lib','/lib64','--proc','/proc','--dev','/dev','--tmpfs','/tmp','--dir','/etc','--ro-bind','/etc/fonts','/etc/fonts','--bind',str(out),str(out),'--chdir',str(out)]
for key,value in env.items():cmd+=['--setenv',key,value]
cmd+=['--','qml6',str(out/'main.qml')]
v=subprocess.run(cmd,capture_output=True,text=True,timeout=15);log=v.stdout+v.stderr;(out/'test.log').write_text(log);print(log)
assert v.returncode==0 and 'PASS: real resource pane' in log,(v.returncode,log)
assert not any(x in log for x in ('TypeError','ReferenceError','Unable to assign','Binding loop')),log
(out/'sources.json').write_text(json.dumps({p.name:hashlib.sha256(p.read_bytes()).hexdigest() for p in plugin.iterdir() if p.is_file()},indent=2)+'\n')
