import QtQuick
import Quickshell

Item {
 id:test
 required property var window
 property bool reloadOnly:false
 property int phase:reloadOnly ? 100 : 0
 property int ticks:0
 property bool captured:false
 function check(v,message){if(!v)throw new Error(message)}
 function find(o,name){if(o.objectName===name)return o;var children=o.children||[];for(var i=0;i<children.length;i++){var v=find(children[i],name);if(v)return v}return null}
 function textButton(o,label){if(o.text===label && typeof o.clicked==="function")return o;var children=o.children||[];for(var i=0;i<children.length;i++){var v=textButton(children[i],label);if(v)return v}return null}
 Timer {interval:100;running:true;repeat:true;onTriggered:{try{
  test.ticks++;test.check(test.ticks<160,"timed out at "+test.phase)
  var w=test.window,s=w.service
  if(!s.resourceReady || s.busy || test.ticks<20)return
  var edit=test.find(w.contentItem,"edit-ship-ports"),ram=test.find(edit.parent,"container-memory"),cpu=test.find(edit.parent,"container-cpus")
  if(test.phase===100){
   test.check(w.instance.config.memory_gib===16 && w.instance.config.cpu_cores===4,"reopen lost resource selection")
   edit.clicked();test.check(ram.currentIndex===15 && cpu.currentIndex===3,"reopen editor lost settings")
   console.log("PASS: resource preview persisted across reopen");Qt.quit()
  }else if(test.phase===0){
   test.check(!!test.find(w.contentItem,"nav-ship-preview-publisher"),"full sidebar missing")
   edit.clicked();test.check(ram.enabled && cpu.enabled,"stopped controls disabled")
   ram.currentIndex=15;ram.activated(15);cpu.currentIndex=3;cpu.activated(3)
   test.find(w.contentItem,"save-ship-http-port").clicked();test.phase=1
  }else if(test.phase===1){
   test.check(w.instance.config.memory_gib===16 && w.instance.config.cpu_cores===4,"Save lost limits")
   test.check(w.instance.config_revision===2 && w.instance.applied_revision===1,"Save changed applied launch")
   var start=test.textButton(w.contentItem,"▶ Start");test.check(start.enabled,"Start unavailable");start.clicked();test.phase=2
  }else if(test.phase===2){
   test.check(w.instance.observed_state==="running" && w.instance.runtime_status.launch_memory_gib===16 && w.instance.resources.cpu_limit_cores===4,"Start lost resource contract")
   edit.clicked();test.check(!ram.enabled && !cpu.enabled,"running controls enabled")
   var stop=test.textButton(w.contentItem,"■ Stop");test.check(stop.enabled,"Stop unavailable");stop.clicked();test.phase=3
  }else if(test.phase===3){
   test.check(w.instance.observed_state==="stopped","Stop failed");edit.clicked();test.check(ram.enabled && cpu.enabled && ram.currentIndex===15 && cpu.currentIndex===3,"reload lost limits")
   w.selectedKey="preview-remote-040/preview-remote-bus";test.phase=4
  }else if(test.phase===4){
   edit.clicked();test.check(!ram.enabled,"remote running controls enabled")
   s.remoteMode="observer";s.refresh();test.check(!edit.enabled,"observer can edit")
   s.remoteMode="offline";s.refresh();test.check(!edit.enabled,"offline can edit")
   s.remoteMode="online";s.refresh();w.selectedKey="preview-local-040/preview-project-nec";test.phase=5
  }else if(test.phase===5){
   test.check(!edit.enabled,"project member inherited standalone resource editing")
   w.page="setup";w.setupPane.mode="development";test.phase=6
  }else if(test.phase===6 && w.setupPane.appDevelopmentPane.policyView){
   test.check(!!test.find(w.setupPane,"app-policy-tree"),"permission tree missing")
   w.selectedKey="preview-local-040/preview-publisher";w.page="overview";edit.clicked();var card=test.find(w.contentItem,"connectivity-runtime-card");w.revealItem(card);test.phase=7
  }else if(test.phase===7){
   test.check(w.page==="overview" && ram.enabled,"RAM & CPU shortcut failed")
   w.contentItem.grabToImage(function(result){test.captured=result.saveToFile("/data/state/manual-resources.png")});test.phase=8
  }else if(test.phase===8 && test.captured){console.log("PASS: full-panel resource save/start/stop, requested/applied limits, scopes and Setup retained");Qt.quit()}
 }catch(e){console.error("FAIL: "+e.message);Qt.exit(1)}}}
}
