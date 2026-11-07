import QtQuick
import Quickshell.Io
import "AppPolicy.js" as Policy
import "AppWorkflow.js" as Workflow

Item {
  id:root
  property var service
  property bool active:false
  property string runnerName:""
  property string runnerID:""
  property string mode:""
  property var target:({})
  property bool canEdit:false
  property var value:null
  property string error:""
  property string message:""
  property string output:""
  property string errorOutput:""
  property int generation:0
  property int requestGeneration:-1
  property string requestKey:""
  property bool invalidOutput:false
  property bool pending:false
  property bool pendingRefresh:false
  property bool saving:false
  readonly property bool available:active && !!service && service.activationEnabled && !service.transportStopping && !!runnerName && !!runnerID && Policy.target(target)
  readonly property bool busy:pending || helper.running
  readonly property string binding:JSON.stringify([service ? service.themeCommand([]) : [],runnerName,runnerID,mode,target])
  ReaderLifecycle {service:root.service;process:helper;kind:"app-development-workflow"}
  onBindingChanged:invalidate()
  onAvailableChanged:invalidate()
  function invalidate(){
    generation++;value=null;error="";message="";pendingRefresh=available
    if(helper.running){helper.signal(15);hardStop.restart()}
    debounce.restart()
  }
  function launch(update){
    if(!available || busy || update && !canEdit)return false
    requestGeneration=generation;requestKey=binding;output="";errorOutput="";error="";message="";invalidOutput=false;saving=!!update;pending=true
    var request={name:runnerName,runner_id:runnerID,target:target};if(update)request.update=update
    helper.command=service.themeCommand([update ? "app-development-configure" : "app-development-workflow","--request-json",JSON.stringify(request)])
    deadline.restart();helper.running=true;return true
  }
  function refresh(){pendingRefresh=false;return launch(null)}
  function save(ship,desk){if(!value)return false;var u={target:target,expected_revision:value.revision};if(ship)u.ship=ship;if(desk)u.desk=desk;return launch(u)}
  function finish(code,status){
    deadline.stop();hardStop.stop();pending=false
    if(requestGeneration!==generation || requestKey!==binding || !available){if(pendingRefresh)debounce.restart();return}
    var v;try{v=JSON.parse(output)}catch(_){v=null}
    if(code!==0 || status!==0 || invalidOutput || !Workflow.valid(v,runnerID,mode,target,Policy)){
      value=null;error=saving ? "Could not confirm the workflow save. Refresh before retrying." : "Workflow settings unavailable on this runner."
      try{var e=JSON.parse(errorOutput);if(typeof e.error==="string")error=e.error}catch(_){}
      return
    }
    value=v.setting;if(saving)message="Workflow saved. Permissions are unchanged."
  }
  Timer {id:debounce;interval:100;onTriggered:if(root.pendingRefresh && root.available && !root.busy)root.refresh()}
  Timer {id:deadline;interval:30000;onTriggered:{root.invalidOutput=true;if(helper.running){helper.signal(15);hardStop.restart()}}}
  Timer {id:hardStop;interval:5000;onTriggered:if(helper.running)helper.signal(9)}
  Process {
    id:helper
    stdout:SplitParser {splitMarker:"";onRead:function(data){if(root.output.length+data.length>65536){root.invalidOutput=true;helper.signal(9)}else root.output+=data}}
    stderr:SplitParser {splitMarker:"";onRead:function(data){if(root.errorOutput.length+data.length<=65536)root.errorOutput+=data}}
    onExited:function(code,status){root.finish(code,status)}
  }
}
