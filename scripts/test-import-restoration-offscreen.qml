import QtQuick
import QtQuick.Controls
import "Plugin" as P
import "Plugin/Model.js" as Model

ApplicationWindow {
  id:test
  visible:true;width:900;height:1200;color:"#171a1e"
  property int phase:0
  property bool captured:false
  property var saved:null
  property var savedKey:null
  property int sourceInspections:0
  readonly property var connection:svc.connections[0] || null
  readonly property var controller:svc.importRestoration
  readonly property string output:@@OUTPUT@@
  @@SERVICE_FIXTURE@@
  ScrollView {
    anchors.fill:parent;anchors.margins:20;contentWidth:availableWidth
    Column {
      id:content;width:parent.width;spacing:16
      P.Label {width:parent.width;text:"Omabit · Isolated restoration lab";font.bold:true}
      P.StoragePane {id:storage;width:parent.width;service:svc;preferredConnection:test.connection}
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
  function runner(){
    var stamp=new Date(svc.now).toISOString()
    return {name:"restore-lab",runner_id:"runner-restore-lab",actor:"local-uid:1000",status:"available",access_role:"operator",checked_at:stamp,features:["restore-import","recover-import","guided-pier-import","import-pier","pier-storage","storage-locations"],snapshot:{instances:[{id:"zod",ship:"~zod",label:"Fresh lab ship",mode:"fake",config_revision:1,applied_revision:0,desired_state:"stopped",observed_state:"failed",observed_at:stamp,recovery_required:true,pier_path:"fakeships/zod/zod",pier_name:"zod",config:{http_port:18080,ames_port:31337,loom:31},external_import:{stage:"copying",source:"/fixture/source/unclassified-pier",holding_path:"/fixture/source/.omabit-import-"+"1".repeat(32)+"/pier",total_files:7,total_bytes:1016823820,copied_bytes:60030976},runtime_status:{import_restore_available:true}}],operations:[],piers:{root:"/fixture/piers",roots:[{id:"default",path:"/fixture/piers"}],candidates:[]}}}
  }
  function plan(){var t=controller.pendingPreview;return {actor:t.actor,runner_id:t.runner,instance_id:t.instance,config_revision:t.revision,source:t.holding,restore_to:t.source,destination:t.destination,retain_at:t.retainAt,review_digest:"a".repeat(64),source_files:7,source_bytes:1016823820,destination_files:1,destination_bytes:60030976,stage:t.stage,continuation:t.stage!=="review"}}
  function preview(){control(storage,"review-source-restoration-zod").clicked();check(svc.currentAction==="preview-import-restore","review did not dispatch");svc.reply(0,plan())}
  function receipt(o,status){var p=o.plan,stamp=new Date(svc.now).toISOString();return {id:"op-"+o.request,actor:o.target.actor,instance_id:o.target.instance,request_id:o.request,config_revision:o.target.revision,kind:"restore-import",status:status,created_at:stamp,finished_at:status==="queued"?undefined:stamp,import_restoration:status==="succeeded"?{source:p.source,restore_to:p.restore_to,destination:p.destination,retain_at:p.retain_at,review_digest:p.review_digest}:undefined,message:status==="failed"?"Fixture interrupted after retaining the partial copy":""}}
  function capture(name){captured=false;content.grabToImage(function(result){captured=result.saveToFile(output+"/"+name+".png")})}
  function currentOutcome(){var keys=Object.keys(controller.outcomes);savedKey=keys[keys.length-1];saved=copy(controller.outcomes[savedKey]);return saved}
  Timer {
    interval:90;repeat:true;running:true
    onTriggered:{try{
      var pane=control(storage,"import-source-restoration"), consent=control(pane,"restore-consent"), submit=control(pane,"restore-submit")
      if(phase===0){svc.connections=[runner()];phase=1}
      else if(phase===1){
        check(control(storage,"review-source-restoration-zod").enabled,"eligible import has no enabled review")
        control(storage,"review-source-restoration-zod").clicked();var p=plan();p.retain_at="/fixture/unrelated"
        svc.reply(0,p);check(!controller.review && controller.error!=="","mismatched archive accepted")
        capture("refusal");phase=2
      }else if(phase===2 && captured){
        preview();check(pane.current && !consent.checked && !submit.enabled,"preview inherited consent or is unavailable")
        check(control(pane,"restore-moves").text.indexOf("/fixture/piers/archive/import-")>=0,"both moves not presented")
        capture("review");phase=3
      }else if(phase===3 && captured){
        consent.checked=true;var c=copy(connection);c.snapshot.instances[0].config_revision=2;svc.connections=[c]
        check(!pane.current && !consent.checked && !submit.enabled,"revision change kept approval")
        preview();phase=4
      }else if(phase===4){
        consent.checked=true;submit.clicked();submit.clicked();currentOutcome()
        check(svc.calls.filter(function(c){return c.kind==="restore-import"}).length===1,"duplicate restoration dispatched")
        svc.reply(-1,null,"Fixture acknowledgement lost")
        check(controller.outcomes[savedKey].status==="uncertain" && !control(storage,"review-source-restoration-zod").enabled,"uncertain request unlocked another restoration")
        var c=copy(connection),wrong=receipt(saved,"succeeded");wrong.actor="ssh-key:someone-else";c.snapshot.operations=[wrong];svc.connections=[c]
        check(controller.outcomes[savedKey].status==="uncertain","another actor resolved the request")
        capture("uncertain");phase=5
      }else if(phase===5 && captured){
        var c=copy(connection);c.snapshot.instances=[];c.snapshot.operations=[receipt(saved,"succeeded")];svc.connections=[c]
        check(controller.outcomes[savedKey].status==="succeeded" && pane.visible && svc.lastError==="","retirement removed the receipt or left a stale global error")
        check(control(pane,"restore-receipt-message").text.indexOf("Source restored")>=0,"success not visible outside the ship card")
        capture("restored");phase=6
      }else if(phase===6 && captured){
        var c=runner();c.snapshot.instances[0].config_revision=3;c.snapshot.instances[0].runtime_status.import_restore_stage="restoring-source";c.snapshot.operations=[receipt(saved,"succeeded")];svc.connections=[c];phase=7
      }else if(phase===7){
        preview();check(control(pane,"restore-continuation").visible && submit.text==="Continue restoration","continuation not distinguished")
        consent.checked=true;submit.clicked();currentOutcome();svc.reply(0,receipt(saved,"queued"))
        var c=copy(connection);c.snapshot.operations.push(receipt(saved,"failed"));svc.connections=[c]
        check(controller.outcomes[savedKey].status==="failed","terminal failure hidden")
        capture("interrupted");phase=8
      }else if(phase===8 && captured){
        var item=control(pane,"restore-receipt-"+saved.request)
        check(control(item,"restore-review-again").visible,"explicit continuation review unavailable")
        control(item,"restore-review-again").clicked();check(svc.currentAction==="preview-import-restore","review again mutated without a new preview")
        svc.reply(0,plan());capture("continuation");phase=9
      }else if(phase===9 && captured){
        consent.checked=true;submit.clicked();currentOutcome();svc.reply(0,receipt(saved,"queued"))
        var c=copy(connection);c.snapshot.instances=[];c.snapshot.operations.push(receipt(saved,"succeeded"));svc.connections=[c]
        phase=10
      }else if(phase===10){
        var item=control(pane,"restore-receipt-"+saved.request)
        control(item,"restore-inspect-source").clicked();check(svc.currentAction==="preview-pier","inspect restored source bypassed guided review")
        check(svc.pendingExternalPreview.path==="/fixture/source/unclassified-pier","inspect selected another source")
        console.log("CALLS "+JSON.stringify(svc.calls));console.log("PASS: actual restoration review, retired-instance receipts and explicit continuation UI");Qt.quit()
      }
    }catch(e){console.error("FAIL phase "+phase+": "+e.message+" "+e.stack);Qt.exit(1)}}
  }
  Timer {interval:30000;running:true;onTriggered:{console.error("FAIL timeout phase "+phase);Qt.exit(1)}}
}
