import QtQuick
import Quickshell.Io
import "Model.js" as Model
import "Recordings.js" as RecordingModel

Item {
  id:root
  property var service
  property var connection
  property var instance
  property bool active:false
  property bool consent:false
  property var history:null
  property var controls:null
  property var selected:null
  property var window:null
  property string error:""
  property string output:""
  property bool invalidOutput:false
  property bool pending:false
  property int generation:0
  property int requestGeneration:-1
  property string requestBinding:""
  property string operation:""
  property string representation:""
  property var cursor:null
  property string afterJob:""
  readonly property bool busy:pending || helper.running
  readonly property bool available:active && !!service && service.activationEnabled && !service.transportStopping && lifecycle.admitted && !!instance && !!connection && connection.access_role==="operator" && Model.hasFeature(connection,"runtime-recording-read-v1") && Model.available(connection,service.now)
  readonly property bool controlsAvailable:available && Model.hasFeature(connection,"runtime-recording-control-v1")
  readonly property string binding:JSON.stringify([service ? service.themeCommand([]) : [],connection ? [connection.name,connection.runner_id,connection.mode] : [],instance ? instance.id : ""])
  ReaderLifecycle {id:lifecycle;service:root.service;process:helper;kind:"saved-recording-read"}
  onConsentChanged:if(!consent){generation++;history=null;controls=null;selected=null;window=null;output="";if(helper.running){helper.signal(15);hardStop.restart()}}
  onBindingChanged:invalidate()
  onAvailableChanged:invalidate()
  function invalidate(){
    generation++;history=null;controls=null;selected=null;window=null;consent=false;error="";output=""
    if(helper.running){helper.signal(15);hardStop.restart()}
  }
  function launch(kind,fields){
    if(!available || !consent || busy)return false
    operation=kind;requestGeneration=generation;requestBinding=binding;output="";invalidOutput=false;error="";pending=true
    var input=Object.assign({name:connection.name,runner_id:connection.runner_id,include_private_data:true},fields)
    helper.command=service.themeCommand(["call",kind,"--input-json",JSON.stringify(input)])
    deadline.restart();helper.running=true;return true
  }
  function readHistory(next){
    if(next && (!history || !history.next_job_id))return false
    afterJob=next ? history.next_job_id : "";selected=null;window=null
    return launch("runtime.recordings.v1",{instance_id:instance.id,after_job_id:afterJob})
  }
  function selectJob(job){if(busy || !history || history.sessions.indexOf(job)<0)return false;selected=job;window=null;error="";return true}
  function readWindow(choice,next){
    if(!selected || ["redacted","original"].indexOf(choice)<0 || next && (!window || !window.next || window.representation!==choice))return false
    representation=choice;cursor=next ? window.next : null
    return launch("runtime.recording-page.v1",{instance_id:instance.id,job_id:selected.job_id,job_sha256:selected.job_sha256,representation:choice,cursor:cursor})
  }
  function readControls(){return Model.hasFeature(connection,"runtime-recording-control-v1") && launch("runtime.recording-controls.v1",{})}
  function finish(code,status){
    deadline.stop();hardStop.stop();pending=false
    if(requestGeneration!==generation || requestBinding!==binding || !available)return
    var result;try{result=JSON.parse(output)}catch(_){result=null}
    if(code!==0 || status!==0 || invalidOutput || !result || result.version!==1 || result.operation!==operation || result.error || !result.data){error="Recording read could not be confirmed. Refresh this exact selection to try again.";return}
    try {
      if(operation==="runtime.recordings.v1")history=RecordingModel.catalog(result.data,connection.runner_id,instance.id,afterJob)
      else if(operation==="runtime.recording-page.v1")window=RecordingModel.page(result.data,connection.runner_id,instance.id,selected,representation,cursor)
      else if(operation==="runtime.recording-controls.v1")controls=RecordingModel.controls(result.data,connection.runner_id)
      else throw new Error("Unexpected read")
    }catch(_){error="Recording reply does not match this selection; no new content was accepted."}
    output=""
  }
  Timer {id:deadline;interval:30000;onTriggered:{root.invalidOutput=true;if(helper.running){helper.signal(15);hardStop.restart()}}}
  Timer {id:hardStop;interval:5000;onTriggered:if(helper.running)helper.signal(9)}
  Process {
    id:helper
    stdout:SplitParser {splitMarker:"";onRead:function(data){if(root.requestGeneration!==root.generation || root.requestBinding!==root.binding || !root.available || !root.consent)return;if(root.output.length+data.length>2097152){root.invalidOutput=true;helper.signal(15);hardStop.restart()}else root.output+=data}}
    stderr:SplitParser {splitMarker:"";onRead:function(data){}}
    onExited:function(code,status){root.finish(code,status)}
  }
}
