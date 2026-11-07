import QtQuick
import Quickshell.Io
import "AppPolicy.js" as Policy
import "AppSource.js" as Source

Item {
  id:root
  property var service
  property bool active:false
  property string runnerName:""
  property string runnerID:""
  property string mode:""
  property var target:({})
  property var settingsRevision:0
  property string sourceName:""
  property var value:null
  property string error:""
  property string output:""
  property string errorOutput:""
  property int generation:0
  property int requestGeneration:-1
  property string requestKey:""
  property bool pending:false
  property bool pendingRefresh:false
  property bool invalidOutput:false
  readonly property bool available:active && !!service && service.activationEnabled && !service.transportStopping && !!runnerName && !!runnerID && Policy.target(target) && !!target.desk && !!sourceName
  readonly property bool busy:pending || helper.running
  readonly property string binding:JSON.stringify([service ? service.themeCommand([]) : [],runnerName,runnerID,mode,target,settingsRevision,sourceName])
  ReaderLifecycle {service:root.service;process:helper;kind:"app-development-source"}
  onBindingChanged:invalidate()
  onAvailableChanged:invalidate()
  function invalidate(){generation++;value=null;error="";pendingRefresh=available;periodic.stop();if(helper.running){helper.signal(15);hardStop.restart()}debounce.restart()}
  function refresh(){
    if(!available || busy)return false
    pendingRefresh=false;requestGeneration=generation;requestKey=binding;output="";errorOutput="";error="";invalidOutput=false;pending=true;periodic.stop()
    helper.command=service.themeCommand(["desk-source-status","--runner",runnerName,"--expected-runner-id",runnerID,"--instance",target.instance_id,"--desk",target.desk])
    deadline.restart();helper.running=true;return true
  }
  function finish(code,status){
    deadline.stop();hardStop.stop();pending=false
    if(requestGeneration!==generation || requestKey!==binding || !available){if(pendingRefresh)debounce.restart();return}
    periodic.restart()
    var v;try{v=JSON.parse(output)}catch(_){v=null}
    if(code!==0 || status!==0 || invalidOutput || !Source.valid(v,runnerID,mode,target,settingsRevision,sourceName,Policy)){
      error="Source status unavailable. Refresh the workflow if its configuration changed."
      try{var e=JSON.parse(errorOutput);if(typeof e.error==="string")error=e.error}catch(_){}
      return
    }
    value=v
  }
  Timer {id:debounce;interval:200;onTriggered:if(root.pendingRefresh && root.available && !root.busy)root.refresh()}
  Timer {id:periodic;interval:90000;onTriggered:root.refresh()}
  Timer {id:deadline;interval:120000;onTriggered:{root.invalidOutput=true;if(helper.running){helper.signal(15);hardStop.restart()}}}
  Timer {id:hardStop;interval:5000;onTriggered:if(helper.running)helper.signal(9)}
  Process {
    id:helper
    stdout:SplitParser {splitMarker:"";onRead:function(data){if(root.output.length+data.length>1048576){root.invalidOutput=true;helper.signal(9)}else root.output+=data}}
    stderr:SplitParser {splitMarker:"";onRead:function(data){if(root.errorOutput.length+data.length<=65536)root.errorOutput+=data}}
    onExited:function(code,status){root.finish(code,status)}
  }
}
