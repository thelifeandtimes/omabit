import QtQuick
import Quickshell.Io
import "ShipRoutes.js" as Routes

Item {
  id:root
  property var service
  property bool active:false
  property bool ready:false
  property string runnerName:""
  property string runnerID:""
  property string mode:""
  property string instanceID:""
  property string runtimeBinding:""
  property var value:null
  property string error:""
  property string message:""
  property int generation:0
  property int requestGeneration:-1
  property string requestKey:""
  property string output:""
  property string errorOutput:""
  property bool pending:false
  property bool pendingRefresh:false
  property bool invalidOutput:false
  property bool saving:false
  property var savedRequest:null
  readonly property bool busy:pending || worker.running
  readonly property bool available:active && ready && !!service && service.activationEnabled && !service.transportStopping && !!runnerName && !!runnerID && !!instanceID
  readonly property string binding:JSON.stringify([service ? service.themeCommand([]) : [],runnerName,runnerID,mode,instanceID,runtimeBinding])
  ReaderLifecycle {service:root.service;process:worker;kind:"ship-routes"}
  onBindingChanged:invalidate()
  onAvailableChanged:invalidate()
  function invalidate(){
    generation++;value=null;error="";message="";pendingRefresh=available
    if(worker.running){worker.signal(15);hardStop.restart()}
    debounce.restart()
  }
  function launch(settings){
    if(!available || busy || settings && !value)return false
    requestGeneration=generation;requestKey=binding;output="";errorOutput="";error="";message="";invalidOutput=false;pending=true;saving=!!settings
    var args=[settings ? "ship-routes-configure" : "ship-routes","--runner",runnerName,"--expected-runner-id",runnerID]
    savedRequest=null
    if(settings){savedRequest={selection:Routes.copy(value.selection),update:{binding:Routes.copy(value.record.binding),expected_revision:value.record.revision,settings:Routes.normalized(settings)}};args=args.concat(["--request-json",JSON.stringify(savedRequest)])}
    else args=args.concat(["--instance",instanceID])
    worker.command=service.themeCommand(args);deadline.restart();worker.running=true;return true
  }
  function refresh(){pendingRefresh=false;return launch(null)}
  function save(settings){return launch(settings)}
  function finish(code,status){
    deadline.stop();hardStop.stop();pending=false
    if(requestGeneration!==generation || requestKey!==binding || !available){if(pendingRefresh)debounce.restart();return}
    var v;try{v=JSON.parse(output)}catch(_){v=null}
    var valid=code===0 && status===0 && !invalidOutput && Routes.valid(v,runnerID,mode,instanceID)
    if(valid && saving)valid=v.record.revision===savedRequest.update.expected_revision+1 && JSON.stringify(v.record.binding)===JSON.stringify(savedRequest.update.binding) && JSON.stringify(v.record.settings)===JSON.stringify(savedRequest.update.settings)
    if(!valid){
      value=null;error=saving ? "Could not confirm the URL changes. Refresh before editing again." : "Could not read this ship's URLs. Refresh to try again."
      try{var e=JSON.parse(errorOutput);if(typeof e.error==="string")error=e.error}catch(_){}
      return
    }
    value=v;if(saving)message="Ship URLs saved."
  }
  Timer {id:debounce;interval:100;onTriggered:if(root.pendingRefresh && root.available && !root.busy)root.refresh()}
  Timer {id:deadline;interval:60000;onTriggered:{root.invalidOutput=true;if(worker.running){worker.signal(15);hardStop.restart()}}}
  Timer {id:hardStop;interval:5000;onTriggered:if(worker.running)worker.signal(9)}
  Process {
    id:worker
    stdout:SplitParser {splitMarker:"";onRead:function(data){if(root.output.length+data.length>262144){root.invalidOutput=true;worker.signal(9)}else root.output+=data}}
    stderr:SplitParser {splitMarker:"";onRead:function(data){if(root.errorOutput.length+data.length<=65536)root.errorOutput+=data}}
    onExited:function(code,status){root.finish(code,status)}
  }
}
