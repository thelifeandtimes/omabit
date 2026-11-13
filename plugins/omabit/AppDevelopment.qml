import QtQuick
import Quickshell.Io
import "AppPolicy.js" as Policy

Item {
  id: root
  property var service
  property bool active: false
  property var value: null
  property var inventories: []
  property var discoveryRequest: null
  property string requestCommand: ""
  property string error: ""
  property string message: ""
  property string output: ""
  property string errorOutput: ""
  property bool pending: false
  property int generation: 0
  property int requestGeneration: -1
  property int requestTimeoutMs: 120000
  property bool invalidOutput: false
  property bool mutating: false
  readonly property bool busy: pending || helper.running
  readonly property bool available: active && !!service && service.activationEnabled && !service.transportStopping
  readonly property string binding: service ? JSON.stringify(service.themeCommand([])) : ""
  property string requestBinding: ""
  ReaderLifecycle { service:root.service; process:helper; kind:"app-development-policy" }
  onBindingChanged: invalidate()
  onAvailableChanged: { if(!available) invalidate(); else Qt.callLater(function(){root.refresh()}) }
  function invalidate() {
    generation++;value=null;inventories=[];error="";message=""
    if(helper.running){helper.signal(15);hardStop.restart()}
  }
  function launch(command,request) {
    if(!available || busy)return false
    var args=[command]
    discoveryRequest=command==="desk-list" ? JSON.parse(JSON.stringify(request)) : null
    requestCommand=command
    if(discoveryRequest)args=args.concat(["--runner",request.name,"--expected-runner-id",request.runner_id,"--instance",request.target.instance_id])
    else if(request)args=args.concat(["--request-json",JSON.stringify(request)])
    requestGeneration=generation;requestBinding=binding;output="";errorOutput="";error="";message="";invalidOutput=false
    mutating=command==="app-development-save" || command==="app-development-retry";pending=true
    helper.command=service.themeCommand(args);deadline.restart();helper.running=true;return true
  }
  function refresh(){if(!available || busy)return false;inventories=[];return launch("app-development-status",null)}
  function discover(name,runner,target){
    if(!value || !Policy.target(target) || target.desk)return false
    return launch("desk-list",{name:name,runner_id:runner,target:target})
  }
  function inventory(runner,target){return Policy.inventory(inventories,runner,target)}
  function save(request){if(!value)return false;return launch("app-development-save",request)}
  function retry(){if(!value || value.global.revision===0)return false;return launch("app-development-retry",{revision:value.global.revision})}
  function finish(code,status) {
    deadline.stop();hardStop.stop();pending=false
    if(requestGeneration!==generation || requestBinding!==binding || !available)return
    if(code!==0 || status!==0 || invalidOutput){
      if(!discoveryRequest)value=null
      var message=discoveryRequest ? "Could not read installed desks. Your permissions are unchanged." : "Could not complete the policy request. Refresh to reconcile saved permissions before retrying."
      try{var e=JSON.parse(errorOutput);if(typeof e.error==="string")message=e.error}catch(_){}
      error=message;return
    }
    var v;try{v=JSON.parse(output)}catch(_){v=null}
    if(discoveryRequest){
      var request=discoveryRequest,r=value && value.runners.filter(function(r){return r.name===request.name && r.runner_id===request.runner_id})[0]
      if(!Policy.discovery(v,request) || !r || !r.status || v.mode!==r.status.mode || !r.status.targets.some(function(t){return t.ready && Policy.same(t.target,request.target)})){
        error="Desk discovery did not match the selected ship. Refresh and try again.";return
      }
      inventories=inventories.filter(function(i){return i.runner_id!==v.runner_id || !Policy.same(i.selection.target,v.selection.target)}).concat([v])
      root.message=v.desks.length+" installed desks discovered.";return
    }
    if(!Policy.valid(v)){value=null;error="The helper returned an unsupported policy response. Refresh before editing.";return}
    value=v
    if(mutating)root.message="Permissions saved."
  }
  Timer {id:deadline;interval:root.requestTimeoutMs;onTriggered:{root.invalidOutput=true;if(helper.running){helper.signal(15);hardStop.restart()}}}
  Timer {id:hardStop;interval:5000;onTriggered:if(helper.running)helper.signal(9)}
  Process {
    id:helper
    stdout:SplitParser {splitMarker:"";onRead:function(data){if(root.output.length+data.length>8388608){root.invalidOutput=true;helper.signal(9)}else root.output+=data}}
    stderr:SplitParser {splitMarker:"";onRead:function(data){if(root.errorOutput.length+data.length<=65536)root.errorOutput+=data}}
    onExited:function(code,status){root.finish(code,status)}
  }
}
