import QtQuick
import QtQuick.Controls
import "Plugin" as Plugin

ApplicationWindow {
  id:test;visible:true;width:1000;height:1000
  property int phase:0
  property int ticks:0
  property bool captured:false
  property bool usageCaptured:false
  function check(ok,message){if(!ok)throw new Error(message)}
  function control(object,name){if(!object)return null;if(object.objectName===name)return object;for(var key of ["children","data","contentItem"]){var list=object[key];if(!list)continue;if(typeof list.length!=="number")list=[list];for(var i=0;i<list.length;i++){var found=control(list[i],name);if(found)return found}}return null}
  QtObject {
    id:service
    property bool activationEnabled:true
    property bool transportStopping:false
    property string profileName:"one"
    property string scenario:"normal"
    property double now:Date.now()
    function themeCommand(args){return ["/usr/bin/python3","/lab/backend.py",args.length ? scenario : "normal",profileName].concat(args)}
  }
  property var connection:({name:"lab",runner_id:"runner-lab",mode:"development-managed",status:"available",snapshot:{},checked_at:new Date().toISOString(),access_role:"operator",features:["runtime-recording-read-v1","runtime-recording-control-v1"]})
  Plugin.RecordingsPane {id:pane;width:950;service:service;connection:test.connection;instance:({id:"one"});active:true}
  Timer {interval:30;running:true;repeat:true;onTriggered:{
    var c=pane.controller
    try {
      if(test.phase===0 && c.available){
        test.check(!c.consent && !c.history && !c.window,"opening retained logs read private data")
        test.check(!c.readHistory(false),"history read without consent")
        test.check(!control(pane,"recording-read-original").enabled,"original read enabled without consent")
        c.consent=true;test.check(c.readHistory(false),"explicit history did not start");test.phase=1
      }else if(test.phase===1 && c.history && !c.busy){
        test.check(c.history.sessions.length===1,"saved history missing")
        c.selectJob(c.history.sessions[0]);c.readWindow("redacted",false);test.phase=2
      }else if(test.phase===2 && c.window && !c.busy){
        var text=control(pane,"recording-window-text")
        test.check(text.textFormat===TextEdit.PlainText && text.text.indexOf("<b>literal</b>")>=0 && text.text.indexOf("\\u001b")>=0,"derived text became rich text or terminal bytes")
        test.check(control(pane,"recording-coverage").text.indexOf("not established")>=0,"EOF invented an exit")
        test.phase=20
        pane.grabToImage(function(image){test.captured=image.saveToFile("/lab/recording-view.png")})
      }else if(test.phase===20 && test.captured){
        c.readWindow("original",false);test.phase=3
      }else if(test.phase===3 && c.window && c.window.representation==="original" && !c.busy){
        test.check(control(pane,"recording-window-text").text.indexOf("UFJJVkFURS1PUklHSU5BTA==")>=0,"explicit raw encoding changed")
        c.readControls();test.phase=4
      }else if(test.phase===4 && c.controls && !c.busy){
        test.check(c.controls.capacity_status==="unknown" && control(pane,"recording-capacity").text.indexOf("Capacity unknown")>=0,"unknown storage hid saved controls or became available")
        test.check(control(pane,"recording-storage-usage").text.indexOf("does not mean zero")>=0,"legacy omission became zero usage")
        service.scenario="measured";c.readControls();test.phase=40
      }else if(test.phase===40 && c.controls && c.controls.capacity && c.controls.capacity.usage && c.controls.capacity.usage.status==="measured" && !c.busy){
        test.check(control(pane,"recording-storage-usage").text.indexOf("24.0 KiB allocated blocks · 4.0 MiB logical")>=0,"sparse measured usage lost distinction or rounded allocation to zero")
        test.check(control(pane,"recording-storage-usage").text.indexOf("Selected ship recording storage: 16.0 KiB allocated")>=0,"selected ship usage missing")
        test.check(control(pane,"recording-capacity").text.indexOf("64.0 MiB")>=0,"measurement replaced admission commitment")
        test.phase=400
        pane.grabToImage(function(image){test.usageCaptured=image.saveToFile("/lab/recording-usage-view.png")})
      }else if(test.phase===400 && test.usageCaptured){
        service.scenario="usage-unknown";c.readControls();test.phase=41
      }else if(test.phase===41 && c.controls && c.controls.capacity && c.controls.capacity.usage.status==="unknown" && !c.busy){
        test.check(control(pane,"recording-storage-usage").text.indexOf("unknown (changed)")>=0,"changing storage gained zero/partial totals")
        service.scenario="foreign-usage";c.readControls();test.phase=42
      }else if(test.phase===42 && c.error && !c.busy){
        test.check(c.controls.capacity.usage.status==="unknown" && c.controls.capacity.usage.reason==="changed","foreign usage replaced accepted evidence")
        service.scenario="foreign";c.readWindow("redacted",false);test.phase=5
      }else if(test.phase===5 && c.error && !c.busy){
        test.check(c.window.representation==="original","foreign window replaced accepted evidence")
        service.scenario="delay";c.readHistory(false);test.ticks=0;test.phase=6
      }else if(test.phase===6 && ++test.ticks>3){
        service.profileName="two";test.check(!c.consent && !c.history && !c.window && !c.controls,"profile change retained private evidence");test.phase=7
      }else if(test.phase===7 && !c.busy){
        test.check(!c.history && !c.error,"late response crossed profiles")
        service.scenario="normal";c.consent=true;c.readHistory(false);test.phase=8
      }else if(test.phase===8 && c.history && !c.busy){
        c.consent=false;test.check(!c.history && !c.window && !c.controls,"revoking consent left private history")
        test.check(!c.readControls(),"control read ignored revoked consent")
        pane.active=false;test.check(!c.available,"inactive recording pane still admitted reads")
        console.log("PASS: private saved recording UI");Qt.quit()
      }
    }catch(e){console.error("FAIL: recording phase "+test.phase+" "+e.message);Qt.exit(1)}
  }}
  Timer {interval:15000;running:true;onTriggered:{console.error("FAIL: recording UI timeout at "+test.phase+" "+pane.controller.error);Qt.exit(1)}}
}
