// Template run by test-rc11-panel-offscreen.py. Application components are copied
// verbatim; the real monitor bindings and service callbacks are inserted below.
import QtQuick
import "Plugin" as P
import "Plugin/Model.js" as Model

Window {
  id: test
  visible: true
  width: 800; height: 600
  property var service: svc
  property string selectedKey: "A/lab"
  @@MONITOR_SELECTION@@
  property int step: 0
  property int openings: 0
  property var saved: null
  property var firstImport: null
  property var recoveryInventory: (@@RECOVERY_INVENTORY@@)
  property var recoveryCompleted: (@@RECOVERY_COMPLETED@@)
  property var recoveryAck: (@@RECOVERY_ACK@@)
  property string recoveryRefusal: @@RECOVERY_REFUSAL@@
  property var roots: [{id:"default",path:"/default"},{id:"ssd",path:"/ssd"},{id:"hdd",path:"/hdd"}]
  property var storageRunner: ({runner_id:"A",name:"A",snapshot:{piers:{roots:roots}}})
  P.StorageLocationSelector { id: location; connection: test.storageRunner; allowCustom: true }
  P.PierImportPane { id: form; width: 600; service: svc; connection: test.connection }
  P.MaintenancePane {
    id: maintenance
    width: 600; service: svc; connection: test.connection; instance: test.instance
    onRetentionTargetChanged: if (retentionTarget) test.openings++
  }
  P.ImportRecoveryPane { id: recovery; width:600; service:svc; connection:test.connection; instance:test.instance }
  QtObject {
    id: svc
    // Match Service's scope for the extracted functions; no Process, native
    // shell module, IPC, settings store or management helper is instantiated.
    readonly property var root: svc
    property var connections: []
    property double now: Date.now()
    property bool observing: true
    property string currentAction: ""
    readonly property bool busy: currentAction!==""
    property var calls: []
    property var inspections: ({})
    property var epochInspection: null
    property var importOutcomes: ({})
    property var externalImportPreview: null
    property var localSetupStatus: null
    property var storageOutcome: null
    property var importRecoveryReview: null
    property var routineOutcome: null
    property var epochPreview: null
    property var previewTarget: null
    property var pendingPreviewTarget: null
    property var importCatalogue: null
    property var pendingImportConnection: null
    property bool canConfirmPrune: false
    property var hostMoveOutcome: null
    property var registrationOutcome: null
    property var bootTarget: null
    property string requestIdentity: ""
    property string pendingInput: ""
    property string output: ""
    property string errors: ""
    property bool outputFailed: false
    property bool refreshPending: false
    property string lastError: ""
    property string lastResult: ""
    property var action: ({stdinEnabled:false})
    property var timeout: ({stop:function(){}})
    function submit(args,kind) {
      if (busy) return false
      calls=calls.concat([{args:args,kind:kind}]);currentAction=kind;return true
    }
    function reply(code,value,error) { output=value ? JSON.stringify(value) : "";errors=error || "";finish(code) }
    @@SERVICE_METHODS@@
    @@CONNECTIONS_CHANGED@@
  }
  function check(value,message) { if (!value) throw new Error(message) }
  function copy(value) { return JSON.parse(JSON.stringify(value)) }
  function control(object,name,visited) {
    visited=visited || [];if(!object || visited.indexOf(object)>=0)return null;visited.push(object)
    if(object.objectName===name)return object
    for(var key of ["children","data","contentItem"]) {
      var children=object[key];if(!children)continue;if(typeof children.length!=="number")children=[children]
      for(var n=0;n<children.length;n++){var found=control(children[n],name,visited);if(found)return found}
    }
    return null
  }
  function runner(name) {
    var stamp=new Date(svc.now).toISOString()
    return {name:name,runner_id:name,access_role:"operator",status:"available",checked_at:stamp,features:["epoch-inventory","epoch-prune-prefix","offline-pack","pier-storage","storage-locations","import-pier","guided-pier-import"],snapshot:{instances:[{id:"lab",mode:"fake",ship:"~zod",config_revision:1,desired_state:"stopped",observed_state:"stopped",observed_at:stamp,config:{http_port:18080,ames_port:34343,loom:31}}],operations:[],piers:{roots:test.roots}}}
  }
  function inventory(digest) {
    return {source:"vere-4.6-epoch-v2",pier_id:"a".repeat(32),digest:digest.repeat(64),observed_at:new Date(svc.now).toISOString(),entries:[{name:"0i0",digest:"a".repeat(64),logical_bytes:100,allocated_bytes:4096,file_count:4,snapshot_bytes:0,shared_files:false,protected:false},{name:"0i1",digest:"b".repeat(64),logical_bytes:100,allocated_bytes:4096,file_count:4,snapshot_bytes:100,shared_files:false,protected:true},{name:"0i2",digest:"c".repeat(64),logical_bytes:100,allocated_bytes:4096,file_count:4,snapshot_bytes:100,shared_files:false,protected:true}]}
  }
  function inspect(request,digest) {
    maintenance.clearReview()
    var c=copy(svc.connections[0]),i=c.snapshot.instances[0]
    svc.rememberInspection({runner:c.runner_id,connection:c.name,instance:i.id,revision:i.config_revision,request:request,status:"queued",message:"Queued"})
    i.epochs=inventory(digest)
    c.snapshot.operations.push({id:"op-"+request,request_id:request,instance_id:i.id,config_revision:i.config_revision,kind:"inspect-epochs",status:"succeeded",created_at:new Date(svc.now).toISOString()})
    svc.connections=[c,runner("B")]
    check(!maintenance.retentionTarget,"inspection opened synchronously against unsettled bindings")
  }
  function reviewed(count,digest) {
    check(openings===count,"wrong review count "+openings+" expected "+count)
    check(!!maintenance.retentionTarget && maintenance.retentionTarget.inventory.digest===digest.repeat(64),"review used missing/old digest")
    check(Model.currentEpochTarget(svc.connections,maintenance.retentionTarget,svc.now),"newly opened review is already stale")
  }
  function beginImport(source) {
    var c=svc.connections[0],s={path:source,fingerprint:"f".repeat(64),files:3,bytes:100}
    var review={runner:c.runner_id,connection:c.name,source:s,loadedAt:svc.now,plan:{runner_id:c.runner_id,source:s,storage_root:"default",instance_id:"zod",destination:"fakeships/zod/zod",ship:"~zod",network:"fakenet",kind:"fake",http_port:18090,in_place:false}}
    form.sourcePath=source
    var target=svc.importExternalPier(c,review,true)
    check(!!target,"actual import dispatch refused")
    form.submission=target
    svc.reply(0,{id:"op-"+target.request,instance_id:"zod",request_id:target.request,kind:"import-pier",config_revision:1,status:"queued"})
    check(form.progress.active && !form.reviewAnother(),"accepted import unlocked before receipt")
    return target
  }
  function importSuccess(target,instance) {
    var c=copy(svc.connections[0])
    c.snapshot.instances=c.snapshot.instances.filter(function(i){return i.id!=="zod"})
    if (instance) c.snapshot.instances.push(instance)
    c.snapshot.operations.push({id:"op-"+target.request,instance_id:"zod",request_id:target.request,kind:"import-pier",config_revision:1,status:"succeeded"})
    svc.connections=[c,runner("B")]
  }
  function replacementImport(revision,source) {
    return {id:"zod",config_revision:revision,external_import:{source:source,stage:"copying",events:[{stage:"removing-source",at:new Date(svc.now).toISOString()}]}}
  }
  Timer {
    interval: 25; running: true; repeat: true
    onTriggered: {
      try {
        switch (test.step++) {
        case 0:
          var selector=control(location,"storage-location-selector")
          check(location.rootID==="default" && selector.currentIndex===0,"wrong initial location")
          location.selectRoot("ssd");check(selector.currentIndex===1,"named location not initialized")
          test.roots=[{id:"hdd",path:"/hdd"},{id:"ssd",path:"/ssd"},{id:"default",path:"/default"}]
          check(location.rootID==="ssd" && selector.currentIndex===2,"inventory reordered selected root")
          test.roots=[{id:"hdd",path:"/hdd"},{id:"default",path:"/default"}]
          check(location.rootID==="" && selector.currentIndex===-1,"removed root silently replaced")
          test.roots=[{id:"default",path:"/default"},{id:"ssd",path:"/ssd"},{id:"hdd",path:"/hdd"}]
          selector.currentIndex=3;selector.activated(3)
          check(location.customPath && control(location,"custom-storage-location").currentIndex===1,"custom path lost selected root")
          test.storageRunner={name:"B",runner_id:"B",snapshot:{piers:{roots:test.roots}}}
          check(location.rootID==="default" && selector.currentIndex===0,"runner switch retained hidden root")
          console.log("PASS: actual offscreen storage selector preserves named/default/custom bindings")
          svc.connections=[runner("A"),runner("B")]
          break
        case 1:
          check(!test.instance.epochs,"first inspection fixture already has inventory")
          inspect("first","b")
          break
        case 2:
          reviewed(1,"b")
          saved=copy(svc.connections[0]);maintenance.clearReview()
          svc.connections=[];svc.connections=[saved,runner("B")]
          svc.rememberInspection({runner:"B",connection:"B",instance:"lab",revision:1,request:"unrelated",status:"queued",message:"Queued"})
          break
        case 3:
          check(!maintenance.retentionTarget && openings===1,"reconnect reopened a consumed review")
          inspect("second","c")
          break
        case 4:
          reviewed(2,"c");maintenance.selectCutoff("0i0")
          svc.connections=copy(svc.connections)
          svc.rememberInspection({runner:"B",connection:"B",instance:"lab",revision:1,request:"unrelated2",status:"queued",message:"Queued"})
          break
        case 5:
          check(openings===3 && maintenance.retentionTarget.cutoff==="0i0","unrelated callback reset edited review")
          inspect("superseded","d")
          svc.rememberInspection({runner:"A",connection:"A",instance:"lab",revision:1,request:"new-request",status:"queued",message:"Queued"})
          break
        case 6:
          check(!maintenance.retentionTarget && openings===3,"superseded request opened")
          inspect("hidden","e");maintenance.visible=false;maintenance.visible=true
          break
        case 7:
          check(!maintenance.retentionTarget && openings===3,"hide/show failed to cancel queued opening")
          maintenance.visible=false;inspect("completed-hidden","f")
          break
        case 8:
          maintenance.visible=true
          saved=copy(svc.connections[0]);svc.connections=[];svc.connections=[saved,runner("B")]
          break
        case 9:
          check(!maintenance.retentionTarget && openings===3,"offscreen completion reopened on return/reconnect")
          inspect("selection-race","d");test.selectedKey="B/lab";test.selectedKey="A/lab"
          break
        case 10:
          check(!maintenance.retentionTarget && openings===3,"selection change did not cancel pending request")
          inspect("revision-race","e")
          saved=copy(svc.connections[0]);saved.snapshot.instances[0].config_revision=2;svc.connections=[saved,runner("B")]
          break
        case 11:
          check(!maintenance.retentionTarget && openings===3,"revision changed before opening")
          inspect("missing-receipt","f")
          saved=copy(svc.connections[0]);saved.snapshot.operations=[];svc.connections=[saved,runner("B")]
          break
        case 12:
          check(!maintenance.retentionTarget && openings===3,"retained ready status substituted for missing receipt")
          inspect("section-race","d");maintenance.section="routine";maintenance.section="manual"
          break
        case 13:
          check(!maintenance.retentionTarget && openings===3,"section navigation did not cancel queued review")
          inspect("final-fresh","e")
          break
        case 14:
          reviewed(4,"e")
          console.log("PASS: actual connectionsChanged opens settled fresh inspections once; stale requests, selection/revision races, hidden panes and reconnects cannot reopen")
          maintenance.visible=false
          svc.connections=[runner("A"),runner("B")]
          firstImport=beginImport("/incoming/first")
          form.sourcePath="/incoming/new-selection";test.selectedKey="B/lab"
          check(form.submitted && form.progress.active,"source/runner selection unlocked unresolved import")
          saved=copy(svc.connections[0]);saved.snapshot.operations.push({id:"op-other",instance_id:"zod",request_id:"other-request",kind:"import-pier",config_revision:1,status:"succeeded"});svc.connections=[saved,runner("B")]
          break
        case 15:
          check(form.progress.active && !form.reviewAnother(),"unrelated history resolved the submission")
          importSuccess(firstImport,replacementImport(2,firstImport.source))
          break
        case 16:
          check(!form.progress.active && form.progress.value===1 && form.progress.log==="","success at revision N stayed active at N+1 or borrowed its history")
          check(form.submission.request===firstImport.request && form.reviewAnother(),"source selection lost terminal request binding")
          test.selectedKey="A/lab"
          firstImport=beginImport("/incoming/deleted")
          importSuccess(firstImport,null)
          break
        case 17:
          check(!form.progress.active && form.reviewAnother(),"deleted instance hid successful request")
          firstImport=beginImport("/incoming/reused-name")
          importSuccess(firstImport,replacementImport(3,"/different/source"))
          break
        case 18:
          check(!form.progress.active && form.progress.log==="","recreated name blocked old success or reused unrelated step history")
          saved=copy(svc.connections[0]);saved.snapshot.operations=[];svc.connections=[saved];svc.connections=[]
          break
        case 19:
          check(!form.submitted && Model.importProgress([],svc.importOutcomes[Model.importRequestKey(firstImport)],svc.now).value===1,"terminal history was lost after journal eviction/disconnection")
          console.log("PASS: actual import callbacks resolve exact success after configure/delete/recreate, preserve source selection binding and retain terminal history")
          test.selectedKey="runner-fixture/alpha";svc.connections=[recoveryInventory]
          break
        case 20:
          check(recovery.available,"G04 explicit capability fixture did not expose recovery")
          saved=copy(recoveryInventory);saved.features=saved.features.filter(function(f){return f!=="recover-import"});svc.connections=[saved]
          break
        case 21:
          check(!recovery.available,"missing recovery capability enabled control")
          svc.connections=[recoveryInventory]
          break
        case 22:
          control(recovery,"review-import-recovery").clicked()
          check(!svc.recoverImport(Object.assign({},recovery.target,{revision:99})),"stale recovery target dispatched")
          check(!svc.recoverImport(Object.assign({},recovery.target,{holding:"/replacement"})),"changed recovery evidence dispatched")
          control(recovery,"submit-import-recovery").clicked();control(recovery,"submit-import-recovery").clicked()
          svc.reply(1,null,recoveryRefusal)
          break
        case 23:
          check(recovery.outcome.status==="failed" && !recovery.receipt,"definite recovery rejection became uncertain")
          check(control(recovery,"review-import-recovery-again").enabled,"refused recovery cannot be reviewed again")
          control(recovery,"review-import-recovery-again").clicked();control(recovery,"review-import-recovery").clicked();control(recovery,"submit-import-recovery").clicked()
          recoveryAck.request_id=svc.storageOutcome.request;svc.reply(0,recoveryAck)
          saved=copy(recoveryCompleted);saved.snapshot.operations[0].request_id=recoveryAck.request_id;svc.connections=[saved]
          break
        case 24:
          check(recovery.receipt && recovery.receipt.status==="succeeded" && !recovery.available,"completed recovery receipt not bound")
          check(control(recovery,"import-recovery-receipt").text.indexOf("ship is stopped")>=0,"stopped receipt message missing")
          control(recovery,"submit-import-recovery").clicked()
          var calls=svc.calls.filter(function(c){return c.kind==="recover-import"})
          check(calls.length===2 && calls[0].args.join(" ").indexOf("--revision 2")>=0,"recovery callbacks duplicated or changed command target")
          check(!svc.calls.some(function(c){return c.args.indexOf("start")>=0}),"fixture dispatched Start")
          console.log("PASS: G04 actual offscreen recovery control uses configured capabilities, rejects stale reviews, handles refusal/retry and binds stopped receipt")
          Qt.quit();break
        }
      } catch (error) { console.error("FAIL: offscreen step "+(test.step-1)+" "+error.message+" "+error.stack);Qt.exit(1) }
    }
  }
  Timer { interval: 15000; running: true; onTriggered: { console.error("FAIL: offscreen regression timed out");Qt.exit(1) } }
}
