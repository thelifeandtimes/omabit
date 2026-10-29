#!/usr/bin/env python3
"""Exercise recommended comet submission and one-ship project selection with inert callbacks."""
from pathlib import Path
import os,subprocess,shutil,tempfile,atexit
r=Path(__file__).resolve().parents[1];temporary=tempfile.TemporaryDirectory(prefix='omabit-compact-');atexit.register(temporary.cleanup);d=Path(temporary.name)
p=d/'Plugin';p.mkdir(exist_ok=True)
for n in ('RegistrationPane.qml','ProjectCreationStatus.qml','SwitchField.qml','BootstrapSelector.qml','RuntimeSettingsFields.qml','StorageLocationSelector.qml','SelectField.qml','Label.qml','Design.qml','ActionButton.qml','Field.qml','CheckField.qml','Hint.qml','Model.js'):
 shutil.copy2(r/'plugins/omabit'/n,p/n)
(p/'qmldir').write_text('singleton Design 1.0 Design.qml\n')
c=d/'qs/Commons';c.mkdir(parents=True,exist_ok=True)
(c/'qmldir').write_text('module qs.Commons\nsingleton Style 1.0 Style.qml\nsingleton Color 1.0 Color.qml\n')
(c/'Style.qml').write_text('pragma Singleton\nimport QtQuick\nQtObject { function space(n) { return n } property var font: ({family:"sans-serif",caption:12,bodySmall:14,body:14}) }')
(c/'Color.qml').write_text('pragma Singleton\nimport QtQuick\nQtObject { property color background:"#181922"; property color foreground:"white";property color accent:"blue";property color urgent:"red" }')
(d/'main.qml').write_text('''import QtQuick
import "Plugin" as P
Window { id:w; width:590; height:690; visible:true
 property var c: ({name:"dev",runner_id:"dev",access_role:"operator",status:"available",mode:"development-managed",checked_at:new Date().toISOString(),features:["create-comet","create-fake","runtime-settings","bootstrap-selection","pier-storage","project-dependencies-v1"],snapshot:{instances:[]}})
 P.RegistrationPane {id:form;width:590;compact:true;connection:w.c;service:svc}
 QtObject { id:svc
 property bool busy:false
 property bool observing:true
 property double now:Date.now()
 property var portTarget:null
 property var portSuggestion:null
 property bool checkingPort:false
 property string portError:""
 property var registrationOutcome:null
 property string recommendedPillUrl:"https://urbit-foundation.s3.us-east-2.amazonaws.com/omabit.pill"
 property var received:null
 property var projectCreation: ({busy:false,completed:false,error:"",intent:null,progress:{},start:function(c,n,m,ok,r,p){svc.received={project:n,projectID:p,members:m,confirmed:ok};return true}})
 function suggestPort(c,t){portTarget={token:t,runner:c.runner_id};portSuggestion={runner:c.runner_id,connection:c.name,token:t,port:18082}}
 function editRegistration(){}
 function registerComet(c,n,p,consent,pier,boot,source,loom){received={name:n,port:p,consent:consent,pier:pier,bootstrap:boot,loom:loom};return true}
 }
 function find(o,n){if(o.objectName===n)return o;for(var i=0;i<(o.children||[]).length;i++){var f=find(o.children[i],n);if(f)return f}return null}
 function check(v,m){if(!v)throw Error(m)}
 Timer {interval:300;running:true;onTriggered:{try{
  check(form.useRecommended,"recommended default");check(form.height<300,"compact form too tall: "+form.height)
  check(!find(form,"registration-name").visible,"custom fields visible")
  var b=find(form,"register-instance");check(b.enabled,"boot disabled: "+form.problem);b.clicked()
  check(svc.received.name==="comet-1"&&svc.received.port==="18082"&&svc.received.consent&&svc.received.bootstrap==="recommended"&&svc.received.loom===31,"wrong boot settings")
  find(form,"registration-recommended").checked=false
  check(find(form,"registration-name").visible,"custom controls hidden")
  console.log("PASS compact recommended comet: "+JSON.stringify(svc.received))
  form.kind="fake";find(form,"registration-project").currentIndex=1;find(form,"registration-project-name").text="demo";find(form,"registration-port").text="18083";find(form,"comet-consent").checked=true
  check(b.enabled,"project boot disabled: "+form.problem);b.clicked()
  check(svc.received.project==="demo"&&svc.received.members.length===1&&svc.received.members[0].ship==="~zod"&&svc.received.confirmed,"wrong project settings")
  console.log("PASS single fake project from ordinary form")
  w.c=Object.assign({},w.c,{features:w.c.features.concat(["project-reservations-v1","project-add-member-v1"]),project_inventory:{projects:[{id:"project-demo",label:"demo",revision:1,members:[{instance_id:"demo-zod",ship:"~zod",http_port:18080}]}]}})
  find(form,"registration-project").currentIndex=2;find(form,"registration-fake-ship").text="~nec";find(form,"registration-name").text="demo-nec";find(form,"registration-port").text="18082";find(form,"comet-consent").checked=true
  check(b.enabled,"join disabled: "+form.problem);b.clicked()
  check(svc.received.projectID==="project-demo"&&svc.received.members[0].ship==="~nec","join did not bind existing project")
  svc.projectCreation=Object.assign({},svc.projectCreation,{busy:true,intent:{runner_id:"dev",members:svc.received.members}})
  w.c=Object.assign({},w.c,{snapshot:{instances:[{id:"demo-nec",ship:"~nec",mode:"fake",project_id:"project-demo"}]}})
  check(form.ownProjectAttempt&&form.problem==="","own reservation reported as duplicate")
  svc.projectCreation=Object.assign({},svc.projectCreation,{busy:false,completed:true})
  check(!b.enabled&&form.problem==="","completion allows duplicate boot or false duplicate error")
  console.log("PASS existing-project join and own-reservation feedback");Qt.quit()
 }catch(e){console.error(e);Qt.exit(1)}}}
 Timer {interval:4000;running:true;onTriggered:Qt.exit(2)}
}''')
e=dict(os.environ,QT_QPA_PLATFORM='offscreen',QT_QUICK_BACKEND='software',QML_IMPORT_PATH=str(d),QT_QPA_PLATFORMTHEME='',QT_QUICK_CONTROLS_STYLE='Basic',QT_LOGGING_RULES='*.warning=true;*.critical=true;qml.debug=true',QT_FORCE_STDERR_LOGGING='1');
for name in ('DISPLAY','WAYLAND_DISPLAY','DBUS_SESSION_BUS_ADDRESS','HYPRLAND_INSTANCE_SIGNATURE'):e.pop(name,None)
v=subprocess.run(['qml6',str(d/'main.qml')],env=e,capture_output=True,text=True,timeout=10)
print(v.stdout+v.stderr);assert v.returncode==0 and "PASS compact recommended comet" in v.stderr,(v.returncode,v.stderr);assert not any(x in v.stderr for x in ("TypeError", "ReferenceError", "Unable to assign", "Binding loop")),v.stderr
