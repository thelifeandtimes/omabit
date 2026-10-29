import QtQuick
import QtQuick.Controls
import "Plugin" as P
import "Plugin/Model.js" as Model

ApplicationWindow {
  id: test
  visible:true;width:760;height:1060;color:"#171a1e"
  property int phase:0
  property bool captured:false
  property var saved:null
  property var recoveryTarget:null
  readonly property var connection:svc.connections[0] || null
  readonly property var instance:Model.instances(connection)[0] || null
  readonly property var service:svc
  readonly property string output:@@OUTPUT@@
  @@SERVICE_FIXTURE@@
  ScrollView {
    anchors.fill:parent;anchors.margins:20;contentWidth:availableWidth
    Column {
      id:content;width:parent.width;spacing:20
      P.Label {width:parent.width;text:"Omabit · Disposable import fixture";font.bold:true}
      P.PierImportPane {id:form;width:parent.width;service:svc;connection:test.connection}
      P.Label {id:diagnostic;width:parent.width;text:svc.lastError;color:P.Design.danger;visible:text!==""}
      P.ImportRecoveryPane {id:recovery;service:svc;connection:test.connection;instance:test.instance}
    }
  }
  function check(value,message){if(!value)throw new Error(message)}
  function copy(value){return JSON.parse(JSON.stringify(value))}
  function control(object,name,seen){
    seen=seen||[];if(!object||seen.indexOf(object)>=0)return null;seen.push(object)
    if(object.objectName===name)return object
    for(var key of ["children","data","contentItem"]){var items=object[key];if(!items)continue;if(typeof items.length!=="number")items=[items];for(var i=0;i<items.length;i++){var found=control(items[i],name,seen);if(found)return found}}
    return null
  }
  function runner(){return {name:"import-lab",runner_id:"runner-import-lab",status:"available",access_role:"operator",checked_at:new Date(svc.now).toISOString(),features:["guided-pier-import","import-pier","recover-import","pier-storage","storage-locations"],snapshot:{instances:[],operations:[],piers:{roots:[{id:"default",path:"/fixture/piers"},{id:"disk",path:"/fixture/disk"}]}}}}
  function replyPreview(inPlace){
    var pending=svc.pendingExternalPreview,source={path:pending.path,fingerprint:"a".repeat(64),files:3,bytes:100}
    svc.reply(0,{runner_id:pending.runner,source:source,storage_root:pending.storageRoot,instance_id:"friendly",destination:(pending.storageRoot==="default"?"":"@"+pending.storageRoot+"/")+"fakeships/friendly/zod",ship:"~zod",network:"fakenet",kind:"fake",http_port:18090,in_place:!!inPlace})
  }
  function snapshot(status,stage,request,kind,revision){
    var c=copy(connection),stamp=new Date(svc.now).toISOString()
    c.status="available";c.checked_at=stamp
    c.snapshot.instances=[{id:"friendly",ship:"~zod",mode:"fake",config_revision:revision||1,desired_state:"stopped",observed_state:"stopped",observed_at:stamp,pier_path:"@disk/fakeships/friendly/zod",external_import:{source:saved.source,stage:stage,total_bytes:100,copied_bytes:50,events:[{stage:"reviewed",at:stamp},{stage:"copying",at:stamp},{stage:stage,at:stamp}]},runtime_status:{import_recovery_available:stage==="enrolling"},config:{http_port:18090}}]
    c.snapshot.operations.push({id:"op-"+request,instance_id:"friendly",request_id:request,kind:kind||"import-pier",config_revision:revision||1,status:status,message:status==="failed"?"Fixture interruption after retained enrollment":""})
    svc.connections=[c]
  }
  function capture(name){captured=false;content.grabToImage(function(result){captured=result.saveToFile(output+"/"+name+".png")})}
  function inspect(){svc.lastError="";control(form,"import-pier-review").clicked();check(svc.currentAction==="preview-pier","preview was not dispatched")}
  function accept(target){svc.reply(0,{id:"op-"+target.request,instance_id:target.instance,request_id:target.request,kind:target.kind,config_revision:target.revision||1,status:"queued"})}
  Timer {
    interval:70;running:true;repeat:true
    onTriggered:{try{
      var consent=control(form,"import-pier-consent"),submit=control(form,"import-pier-submit")
      if(phase===0){
        svc.connections=[runner()];form.sourcePath="/fixture/incoming/arbitrary-folder"
        check(!control(form,"import-pier-browse").visible,"remote path offered local browsing")
        check(!submit.enabled && !consent.enabled,"unreviewed import enabled");inspect();phase=1
      }else if(phase===1){
        svc.reply(1,null,JSON.stringify({error:{code:"invalid_request",message:'pier entry "base/app.hoon": mode 0602 permits unsupported group or world writes'}}))
        check(!form.current && !submit.enabled && diagnostic.text.indexOf('"base/app.hoon"')>=0,"diagnostic lost or failed review admitted")
        capture("diagnostic");phase=2
      }else if(phase===2 && captured){
        inspect();replyPreview(false);check(form.current && form.plan.destination==="fakeships/friendly/zod","identity layout missing")
        consent.checked=true
        var locations=control(form,"storage-location-selector");locations.currentIndex=1;locations.activated(1)
        check(!consent.checked && !form.current,"storage selection retained consent")
        inspect();replyPreview(false);check(form.current && control(form,"import-pier-plan").text.indexOf("/fixture/disk/fakeships/friendly/zod")>=0,"custom root display missing")
        consent.checked=true;svc.now+=300001
        var c=copy(connection);c.checked_at=new Date(svc.now).toISOString();svc.connections=[c]
        check(!form.current && !submit.enabled,"expired preview can submit")
        inspect();replyPreview(false);check(!consent.checked,"new review inherited consent")
        capture("plan");phase=3
      }else if(phase===3 && captured){
        consent.checked=true;check(submit.enabled,"reviewed import disabled");submit.clicked();submit.clicked()
        saved=form.submission;check(saved && svc.calls.filter(function(c){return c.kind==="import-pier"}).length===1,"duplicate import dispatch")
        svc.reply(-1,null,"Fixture collector acknowledgement timed out")
        check(form.outcome.status==="uncertain" && form.progress.active && !form.reviewAnother(),"uncertain result unlocked source")
        var c=copy(connection);c.status="unavailable";svc.connections=[c]
        check(form.progress.detail.indexOf("may still be importing")>=0,"offline import misreported stopped")
        phase=4
      }else if(phase===4){
        snapshot("running","copying",saved.request)
        check(form.progress.active && form.progress.value===0.5 && control(form,"import-pier-steps").text.indexOf("Copying the pier")>=0,"copy progress missing")
        check(!form.reviewAnother() && !submit.enabled,"active copy unlocked form")
        svc.lastError="";capture("copying");phase=5
      }else if(phase===5 && captured){
        var c=copy(connection);c.snapshot.operations=[];svc.connections=[c]
        snapshot("failed","enrolling",saved.request)
        check(!form.progress.active && form.progress.detail.indexOf("retained")>=0,"failure lost retained evidence")
        check(recovery.available,"recoverable enrollment hidden")
        control(recovery,"review-import-recovery").clicked();recoveryTarget=copy(recovery.target)
        c=copy(connection);c.snapshot.instances[0].config_revision=2;svc.connections=[c]
        check(!svc.recoverImport(recoveryTarget),"stale recovery dispatched")
        svc.importRecoveryReview=null;control(recovery,"review-import-recovery").clicked()
        control(recovery,"submit-import-recovery").clicked();control(recovery,"submit-import-recovery").clicked()
        check(svc.calls.filter(function(c){return c.kind==="recover-import"}).length===1,"recovery dispatched twice")
        svc.reply(1,null,JSON.stringify({error:{code:"revision_conflict",message:"Fixture revision changed"}}));phase=6
      }else if(phase===6){
        check(control(recovery,"review-import-recovery-again").visible,"explicit recovery retry unavailable")
        control(recovery,"review-import-recovery-again").clicked();control(recovery,"review-import-recovery").clicked();control(recovery,"submit-import-recovery").clicked()
        var target=copy(svc.storageOutcome);accept(target)
        snapshot("succeeded","complete",target.request,"recover-import",2)
        check(control(recovery,"import-recovery-receipt").text.indexOf("ship is stopped")>=0,"recovery implied running")
        svc.lastError="";capture("recovered");phase=7
      }else if(phase===7 && captured){
        check(form.reviewAnother(),"terminal import did not allow another review")
        form.sourcePath="/fixture/disk/fakeships/friendly/zod";inspect();replyPreview(true)
        check(submit.text==="Enroll stopped pier" && consent.text.indexOf("Enroll it")>=0,"in-place flow describes a move")
        consent.checked=true;submit.clicked();saved=form.submission;accept(saved)
        var c=copy(connection);c.snapshot.operations=[];svc.connections=[c]
        snapshot("succeeded","complete",saved.request)
        check(!form.progress.active && form.progress.value===1 && form.progress.detail.indexOf("remains stopped")>=0,"successful import not shown stopped")
        check(Object.keys(svc.importOutcomes).length===2,"new review discarded previous request")
        capture("complete");phase=8
      }else if(phase===8 && captured){
        check(!form.submitted && form.sourcePath==="" && !consent.checked && svc.externalImportPreview===null,"successful import retained form or approval")
        check(svc.navigations.length===1 && svc.navigations[0].runner==="runner-import-lab" && svc.navigations[0].instance==="friendly" && svc.navigations[0].page==="overview","successful import did not open its own monitor exactly once")
        console.log("CALLS "+JSON.stringify(svc.calls));console.log("PASS: guided import and recovery UI");Qt.quit()
      }
    }catch(error){console.error("FAIL: phase "+phase+" "+error.message+" "+error.stack);Qt.exit(1)}}
  }
  Timer {interval:25000;running:true;onTriggered:{console.error("FAIL: timed out at "+phase);Qt.exit(1)}}
}
