import QtQuick
import "Model.js" as Model

QtObject {
  id: root
  property var service
  property var selection: null
  property var pendingPreview: null
  property var review: null
  property string pendingRequest: ""
  property var outcomes: ({})
  property string error: ""
  property var observer: Connections {
    target: root.service
    function onConnectionsChanged() { root.reconcile() }
  }

  function connection(target) {
    return target && service ? service.connections.filter(function(c){return c.runner_id===target.runner && c.name===target.connection})[0] || null : null
  }
  function actorValid(actor) { return typeof actor==="string" && actor.length>0 && actor.length<=256 && !/[\u0000-\u001f\u007f]/.test(actor) }
  function eligible(c,i) {
    return !!service && service.observing && Model.available(c,service.now) && c.access_role==="operator" && actorValid(c.actor) && Model.hasFeature(c,"restore-import") && !!i && Model.validId(i.id) && typeof i.ship==="string" && /^~[a-z-]+$/.test(i.ship) && Number.isSafeInteger(i.config_revision) && i.config_revision>0 && Model.validPierPath(i.pier_path) && i.desired_state==="stopped" && !i.active_operation_id && !i.delete_pending && !Model.handoffFenced(i) && !!i.external_import && ["copying","verifying"].indexOf(i.external_import.stage)>=0 && !!i.runtime_status && i.runtime_status.import_restore_available===true
  }
  function targetFor(c,i) {
    if (!eligible(c,i)) return null
    var parts=Model.splitStoragePath(i.pier_path), location=Model.storageRoots(c).filter(function(r){return r.id===parts.root})[0]
    var held=i.external_import.holding_path, source=i.external_import.source, match=typeof held==="string" && held.match(/\/\.omabit-import-([0-9a-f]{32})\/pier$/)
    if (!location || !Model.hostPath(location.path) || !Model.hostPath(source) || !Model.hostPath(held) || !match || held.slice(0,match.index)!==source.slice(0,source.lastIndexOf("/"))) return null
    var destination=Model.absolutePierPath(c,i.pier_path).split("/").slice(0,-1).join("/")
    return {runner:c.runner_id,connection:c.name,actor:c.actor,instance:i.id,revision:i.config_revision,ship:i.ship,path:i.pier_path,source:source,holding:held,destination:destination,retainAt:location.path+"/archive/import-"+match[1],stage:i.runtime_status.import_restore_stage || "review"}
  }
  function sameTarget(a,b) {
    return !!a && !!b && ["runner","connection","actor","instance","revision","ship","path","source","holding","destination","retainAt","stage"].every(function(k){return a[k]===b[k]})
  }
  function currentTarget(t) {
    var c=connection(t), i=Model.instances(c).filter(function(i){return t && i.id===t.instance})[0]
    return sameTarget(t,targetFor(c,i))
  }
  function terminal(status) { return ["succeeded","failed","interrupted","canceled","rejected"].indexOf(status)>=0 }
  function unresolved(t) {
    return !!t && Object.keys(outcomes).some(function(k){var o=outcomes[k];return o.target.runner===t.runner && o.target.instance===t.instance && o.target.revision===t.revision && !terminal(o.status)})
  }
  function planValid(p,t) {
    if (!p || !t || p.runner_id!==t.runner || p.actor!==t.actor || p.instance_id!==t.instance || p.config_revision!==t.revision || p.source!==t.holding || p.restore_to!==t.source || p.destination!==t.destination || p.retain_at!==t.retainAt || !Model.validDigest(p.review_digest) || p.stage!==t.stage || ["review","retaining-destination","restoring-source","complete"].indexOf(p.stage)<0 || p.continuation!==(p.stage!=="review")) return false
    if (![p.source_files,p.destination_files].every(function(n){return Number.isInteger(n)&&n>=0&&n<=8192}) || ![p.source_bytes,p.destination_bytes].every(function(n){return Number.isSafeInteger(n)&&n>=0&&n<=100000000000})) return false
    var paths=[p.source,p.restore_to,p.destination,p.retain_at]
    return paths.every(function(path,index){return Model.hostPath(path) && paths.slice(0,index).every(function(other){return path!==other && path.indexOf(other+"/")!==0 && other.indexOf(path+"/")!==0})})
  }
  function currentReview(value) {
    return !!value && !!service && service.now-value.loadedAt>=0 && service.now-value.loadedAt<300000 && currentTarget(value.target) && planValid(value.plan,value.target)
  }
  function preview(c,i) {
    if (!service || service.busy) return false
    var retained=targetFor(c,i), current=connection(retained), selected=Model.instances(current).filter(function(v){return i && v.id===i.id})[0], target=targetFor(current,selected)
    if (!sameTarget(retained,target) || unresolved(target)) {error="Refresh the stopped import and resolve its recorded request before reviewing restoration.";return false}
    selection=target;review=null;error="";pendingPreview=target
    if (service.submit(Model.selection(current).concat(["preview-import-restore",target.instance,"--revision",String(target.revision),"--expected-actor",target.actor]),"preview-import-restore")) return true
    pendingPreview=null;return false
  }
  function remember(key,value) { var next=Object.assign({},outcomes);next[key]=value;outcomes=next }
  function restore(value,confirmed) {
    if (!service || service.busy) return false
    if (!confirmed || !currentReview(value) || !review || JSON.stringify(value)!==JSON.stringify(review) || unresolved(value.target)) {error="Review the current restoration paths and explicitly approve both moves.";return false}
    var t=value.target, request=Model.requestId(), key=Model.importRequestKey({runner:t.runner,connection:t.connection,request:request})
    var outcome={target:JSON.parse(JSON.stringify(t)),plan:JSON.parse(JSON.stringify(value.plan)),request:request,status:"submitting",message:"Submitting the reviewed restoration…"}
    remember(key,outcome);pendingRequest=key;error=""
    if (service.submit(Model.selection(connection(t)).concat(["restore-import",t.instance,"--revision",String(t.revision),"--expected-actor",t.actor,"--restore-digest",value.plan.review_digest,"--request-id",request,"--confirm-restore"]),"restore-import")) {review=null;return true}
    pendingRequest="";remember(key,Object.assign({},outcome,{status:"rejected",message:"The collector was busy; no restoration was submitted."}));return false
  }
  function receiptValid(o,receipt) {
    var t=o.target
    if (!receipt || !Model.validId(receipt.id) || receipt.kind!=="restore-import" || receipt.instance_id!==t.instance || receipt.config_revision!==t.revision || receipt.request_id!==o.request || receipt.actor!==t.actor || o.operation && o.operation!==receipt.id || ["queued","running","succeeded","failed","interrupted","canceled"].indexOf(receipt.status)<0) return false
    if (terminal(receipt.status) && !isFinite(Date.parse(receipt.finished_at))) return false
    if (receipt.status==="succeeded") {
      var r=receipt.import_restoration
      if (!r || !["source","restore_to","destination","retain_at","review_digest"].every(function(k){return r[k]===o.plan[k]})) return false
    }
    return true
  }
  function withReceipt(o,receipt) {
    return Object.assign({},o,{operation:receipt.id,status:receipt.status,receipt:JSON.parse(JSON.stringify(receipt)),message:receipt.status==="succeeded" ? "Source restored. The partial copy is retained in archive and the failed registration is removed." : terminal(receipt.status) ? "Restoration needs review: "+String(receipt.message || receipt.status).slice(0,512) : "Restoration is recorded. Waiting for completion…"})
  }
  function finish(kind,code,output,errors,outputFailed) {
    var p=pendingPreview, key=pendingRequest, o=outcomes[key], value=null
    if (kind==="preview-import-restore") pendingPreview=null
    else pendingRequest=""
    // A delayed collector exit cannot undo an already verified terminal receipt.
    if (kind==="restore-import" && o && terminal(o.status) && o.receipt) {error="";return ""}
    var problem=outputFailed ? "Collector output exceeded its limit." : code!==0 ? Model.errorMessage(errors) : ""
    if (!problem) {try{value=JSON.parse(output)}catch(_){problem="Invalid collector response; inspect its recorded result before retrying."}}
    if (kind==="preview-import-restore") {
      if (!problem && (!currentTarget(p) || !planValid(value,p))) problem="The restoration selection changed or the returned paths do not match. Refresh and review again."
      if (!problem) review={target:p,plan:value,loadedAt:service.now}
    } else if (o) {
      if (!problem && !receiptValid(o,value)) problem="Restoration acknowledgement does not match the reviewed request. Wait for its recorded result."
      if (!problem) remember(key,withReceipt(o,value))
      else remember(key,Object.assign({},o,{status:Model.inspectionRejected(code,errors,outputFailed) ? "rejected" : "uncertain",message:problem}))
      reconcile()
      if (outcomes[key].receipt) problem=""
    } else problem="The restoration request binding is missing. Inspect Activity before issuing another request."
    error=kind==="restore-import" && o ? "" : problem
    return problem
  }
  function reconcile() {
    var next=Object.assign({},outcomes)
    Object.keys(next).forEach(function(key){
      var o=next[key], c=connection(o.target)
      if (terminal(o.status) || !Model.available(c,service.now)) return
      var matches=(c.snapshot.operations || []).filter(function(r){return root.receiptValid(o,r)})
      if (matches.length===1) {next[key]=withReceipt(o,matches[0]);if (sameTarget(selection,o.target)) error=""}
    })
    if (JSON.stringify(next)!==JSON.stringify(outcomes)) outcomes=next
  }
  function canReviewAgain(o) {
    var c=connection(o.target), i=Model.instances(c).filter(function(v){return v.id===o.target.instance && v.config_revision===o.target.revision})[0]
    return terminal(o.status) && o.status!=="succeeded" && !!targetFor(c,i) && !unresolved(o.target)
  }
  function reviewAgain(o) {
    var c=connection(o.target), i=Model.instances(c).filter(function(v){return v.id===o.target.instance && v.config_revision===o.target.revision})[0]
    return canReviewAgain(o) && preview(c,i)
  }
}
