import QtQuick
import QtQuick.Controls
import "Plugin" as P
import "Plugin/Model.js" as Model

ApplicationWindow {
  id:test
  visible:true;width:900;height:1200;color:"#171a1e"
  property int phase:0
  property bool captured:false
  property var reviewed:null
  property var completed:null
  readonly property var connection:svc.connections[0] || null
  readonly property var controller:svc.importRestoration
  readonly property var expected:JSON.parse(bridge.expectedReview)
  @@SERVICE_FIXTURE@@
  ScrollView {
    anchors.fill:parent;anchors.margins:20;contentWidth:availableWidth
    Column {
      id:content;width:parent.width;spacing:16
      P.Label {width:parent.width;text:"Omabit · Disposable VM restoration";font.bold:true}
      P.StoragePane {id:storage;width:parent.width;service:svc;preferredConnection:test.connection}
    }
  }
  Connections {
    target:bridge
    function onInventoryReady(raw){svc.now=Date.now();svc.connections=JSON.parse(raw).connections}
    function onCommandDone(kind,code,output,error){
      check(svc.currentAction===kind,"unexpected subprocess response")
      svc.now=Date.now();svc.output=output;svc.errors=error;svc.finish(code)
    }
  }
  function check(value,message){if(!value)throw new Error(message)}
  function copy(value){return JSON.parse(JSON.stringify(value))}
  function control(object,name,seen){
    seen=seen||[];if(!object||seen.indexOf(object)>=0)return null;seen.push(object)
    if(object.objectName===name)return object
    for(var key of ["children","data","contentItem"]){var list=object[key];if(!list)continue;if(typeof list.length!=="number")list=[list];for(var n=0;n<list.length;n++){var found=control(list[n],name,seen);if(found)return found}}
    return null
  }
  function capture(name){captured=false;content.grabToImage(function(result){captured=result.saveToFile(bridge.outputDirectory+"/"+name+".png")})}
  Timer {interval:1000;repeat:true;running:!bridge.preflight;triggeredOnStart:true;onTriggered:bridge.refresh()}
  Timer {
    interval:100;repeat:true;running:true
    onTriggered:{try{
      var pane=control(storage,"import-source-restoration"),consent=control(pane,"restore-consent"),submit=control(pane,"restore-submit")
      if(bridge.preflight){
        if(phase===0){capture("preflight");phase=1}
        else if(captured)bridge.complete(JSON.stringify({ok:true,preflight:true,calls:svc.calls}))
        return
      }
      if(phase===0 && connection){
        check(connection.name==="guest" && connection.runner_id===expected.runner_id && connection.actor===expected.actor,"inventory identity mismatch")
        var reviewButton=control(storage,"review-source-restoration-"+expected.instance_id)
        check(reviewButton && reviewButton.enabled,"restoration review unavailable")
        reviewButton.clicked();phase=1
      }else if(phase===1 && !svc.busy){
        check(controller.review && pane.current,"real preview refused: "+controller.error)
        reviewed=copy(controller.review)
        for(var key in expected)check(reviewed.plan[key]===expected[key],"preview mismatch: "+key)
        check(!consent.checked && !submit.enabled,"review implicitly approved")
        submit.clicked();check(svc.calls.length===1,"unchecked click dispatched a mutation")
        check(control(pane,"restore-moves").text.indexOf(expected.retain_at)>=0 && control(pane,"restore-moves").text.indexOf(expected.restore_to)>=0,"review missing both moves")
        capture("review");phase=2
      }else if(phase===2 && captured){
        consent.checked=true;check(submit.enabled,"explicit consent did not enable restoration")
        submit.clicked();submit.clicked();check(svc.calls.length===2,"duplicate click submitted another request")
        phase=3
      }else if(phase===3){
        var keys=Object.keys(controller.outcomes);check(keys.length===1,"unexpected restoration count")
        var outcome=controller.outcomes[keys[0]]
        if(controller.terminal(outcome.status) && outcome.status!=="succeeded")throw new Error(outcome.message)
        if(!svc.busy && outcome.status==="succeeded" && Model.instances(connection).length===0){
          completed=copy(outcome)
          check(pane.visible && svc.lastError==="","retirement hid receipt or retained an error")
          check(control(pane,"restore-receipt-message").text.indexOf("Source restored")>=0,"terminal receipt text missing")
          capture("restored");phase=4
        }
      }else if(phase===4 && captured){
        bridge.complete(JSON.stringify({ok:true,review:reviewed,outcome:completed,calls:svc.calls,retirement_visible:true,explicit_consent:true,duplicate_click_suppressed:true}))
      }
    }catch(e){bridge.fail("phase "+phase+": "+e.message)}}
  }
  Timer {interval:180000;running:true;onTriggered:bridge.fail("UI timeout at phase "+phase)}
}
