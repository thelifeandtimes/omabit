#!/usr/bin/env python3
"""Exercise compact provisioning and the real key picker with inert callbacks."""
from pathlib import Path
import subprocess,shutil,tempfile,atexit
r=Path(__file__).resolve().parents[1];temporary=tempfile.TemporaryDirectory(prefix='omabit-compact-');atexit.register(temporary.cleanup);d=Path(temporary.name)
p=d/'Plugin';p.mkdir(exist_ok=True)
for n in ('RegistrationPane.qml','KeyBootPane.qml','ProjectCreationStatus.qml','SwitchField.qml','BootstrapSelector.qml','RuntimeSettingsFields.qml','StorageLocationSelector.qml','SelectField.qml','Label.qml','Design.qml','ActionButton.qml','Field.qml','CheckField.qml','Hint.qml','Model.js'):
 shutil.copy2(r/'plugins/omabit'/n,p/n)
(p/'qmldir').write_text('singleton Design 1.0 Design.qml\n')
c=d/'qs/Commons';c.mkdir(parents=True,exist_ok=True)
(c/'qmldir').write_text('module qs.Commons\nsingleton Style 1.0 Style.qml\nsingleton Color 1.0 Color.qml\n')
(c/'Style.qml').write_text('pragma Singleton\nimport QtQuick\nQtObject { function space(n) { return n } property var font: ({family:"sans-serif",caption:12,bodySmall:14,body:14}) }')
(c/'Color.qml').write_text('pragma Singleton\nimport QtQuick\nQtObject { property color background:"#181922"; property color foreground:"white";property color accent:"blue";property color urgent:"red" }')
(d/'picker-fixtures').mkdir()
(d/'picker-fixtures/owned key λ #1.key').write_text('synthetic picker fixture; never submit or boot\n')
(d/'main.qml').write_text('''import QtQuick
import QtQuick.Dialogs
import QtQuick.Controls as Controls
import "Plugin" as P
Window { id:w; width:590; height:690; visible:true
 property var c: ({name:"dev",runner_id:"dev",access_role:"operator",status:"available",mode:"development-managed",checked_at:new Date().toISOString(),features:["create-comet","create-fake","runtime-settings","bootstrap-selection","pier-storage","project-dependencies-v1"],snapshot:{instances:[]}})
 P.RegistrationPane {id:form;width:590;compact:true;connection:w.c;service:svc}
 P.KeyBootPane {id:keyForm;width:590;visible:false;connection:w.c;service:svc}
 QtObject { id:svc
 property bool busy:false
 property bool observing:true
 property double now:Date.now()
 property var portTarget:null
 property var portSuggestion:null
 property bool checkingPort:false
 property string portError:""
 property var registrationOutcome:null
 property var connections:[w.c]
 property int keyRequests:0
 property string recommendedPillUrl:"https://urbit-foundation.s3.us-east-2.amazonaws.com/omabit.pill"
 property var received:null
 property var projectCreation: ({busy:false,completed:false,error:"",intent:null,progress:{},start:function(c,n,m,ok,r,p){svc.received={project:n,projectID:p,members:m,confirmed:ok};return true}})
 function suggestPort(c,t){portTarget={token:t,runner:c.runner_id};portSuggestion={runner:c.runner_id,connection:c.name,token:t,port:18082}}
 function editRegistration(){}
 function registerComet(c,n,p,consent,pier,boot,source,loom){received={name:n,port:p,consent:consent,pier:pier,bootstrap:boot,loom:loom};return true}
 function createFromKey(){keyRequests++;throw Error("Browsing must not submit a key")}
 }
 function find(o,n,seen){seen=seen||[];if(!o||seen.indexOf(o)>=0)return null;seen.push(o);if(o.objectName===n)return o;for(var key of ["children","data","contentItem"]){var children=o[key];if(!children)continue;if(typeof children.length!=="number")children=[children];for(var i=0;i<children.length;i++){var f=find(children[i],n,seen);if(f)return f}}return null}
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
  console.log("PASS existing-project join and own-reservation feedback");keyTest.running=true
 }catch(e){console.error(e);Qt.exit(1)}}}
  Timer {id:keyTest;interval:150;repeat:true;property int phase:0;onTriggered:{try{
   var picker=find(keyForm,"key-file-picker"),browse=find(keyForm,"key-file-browse"),consent=find(keyForm,"key-boot-consent")
   if(phase===0){
    form.visible=false;keyForm.visible=true;keyForm.keyFile="/previous/key.key";consent.checked=true
    check(!!(picker.options & FileDialog.DontUseNativeDialog),"key picker can enter native GTK/GVFS")
    check(picker.popupType===Controls.Popup.Item,"key picker can open below a layer-shell menu")
    check(picker.parentWindow===w,"key picker lost its owning surface")
    check(picker.fileMode===FileDialog.OpenFile,"key picker does not select an existing file")
    picker.currentFolder=Qt.resolvedUrl("picker-fixtures/")
    browse.clicked();phase=1
   }else if(phase===1){
    check(picker.visible&&!browse.enabled&&browse.text==="Choosing…","Browse did not open a file chooser")
    picker.selectedFile=Qt.resolvedUrl("picker-fixtures/owned%20key%20%CE%BB%20%231.key");picker.reject();phase=2
   }else if(phase===2){
    check(!picker.visible&&keyForm.keyFile==="/previous/key.key"&&consent.checked,"cancel changed the selected key or consent")
    check(browse.enabled,"Browse stayed disabled after cancellation");browse.clicked();phase=3
   }else if(phase===3){
    check(picker.visible,"key picker did not reopen")
    picker.selectedFile=Qt.resolvedUrl("picker-fixtures/owned%20key%20%CE%BB%20%231.key");picker.accept();phase=4
   }else if(phase===4){
    check(!picker.visible&&browse.enabled,"accepted picker stayed open")
    check(keyForm.keyFile===decodeURIComponent(String(Qt.resolvedUrl("picker-fixtures/owned%20key%20%CE%BB%20%231.key")).slice(7)),"selected filename lost spaces, Unicode or #")
    check(find(keyForm,"key-file-location").text===keyForm.keyFile&&!consent.checked,"selected path was not shown or boot consent was reused")
    check(svc.keyRequests===0,"browsing submitted a key")
    console.log("PASS key Browse opens, cancels, reopens and accepts a local path without submitting");Qt.quit()
   }
  }catch(e){console.error(e);Qt.exit(1)}}}
  Timer {interval:8000;running:true;onTriggered:Qt.exit(2)}
}''')
e=dict(PATH='/usr/bin:/bin',LANG='C.UTF-8',QT_QPA_PLATFORM='offscreen',QT_QUICK_BACKEND='software',QML_IMPORT_PATH=str(d),QT_QPA_PLATFORMTHEME='',QT_QUICK_CONTROLS_STYLE='Basic',QT_LOGGING_RULES='*.warning=true;*.critical=true;qml.debug=true',QT_FORCE_STDERR_LOGGING='1')
for key,leaf in [('HOME','home'),('XDG_RUNTIME_DIR','runtime'),('XDG_CONFIG_HOME','config'),('XDG_DATA_HOME','data'),('XDG_STATE_HOME','state'),('XDG_CACHE_HOME','cache')]:
 (d/leaf).mkdir(mode=0o700);e[key]=str(d/leaf)
cmd=['bwrap','--unshare-all','--die-with-parent','--clearenv','--ro-bind','/usr','/usr','--symlink','usr/bin','/bin','--symlink','usr/lib','/lib','--symlink','usr/lib','/lib64','--proc','/proc','--dev','/dev','--tmpfs','/tmp','--dir','/etc','--ro-bind','/etc/fonts','/etc/fonts','--bind',str(d),str(d),'--chdir',str(d)]
for key,value in e.items():cmd+=['--setenv',key,value]
v=subprocess.run(cmd+['--','qml6',str(d/'main.qml')],capture_output=True,text=True,timeout=15)
print(v.stdout+v.stderr);assert v.returncode==0 and "PASS key Browse opens" in v.stderr,(v.returncode,v.stderr);assert not any(x in v.stderr for x in ("TypeError", "ReferenceError", "Unable to assign", "Binding loop")),v.stderr
