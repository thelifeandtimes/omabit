import QtQuick
import Quickshell.Io
import "Model.js" as Model

Item {
  id:root
  property var service
  readonly property bool busy:helper.running
  property var intent:null
  property var progress:({})
  property string error:""
  property bool completed:false
  property string errors:""
  readonly property string binding:service?JSON.stringify(service.command([])):""
  signal finished()
  onBindingChanged:if(helper.running)helper.signal(15)
  function start(connection,label,members,consent,resumeRequest,projectID) {
    if(!service||service.busy||!service.observing||!Model.writable(connection,service.now)||!consent)return false
    intent={runner:connection.name,runner_id:connection.runner_id,label:label,project:projectID||"",members:members,request:resumeRequest||Model.requestId(),binding:binding}
    return launch(!!resumeRequest)
  }
  function resume() {
    if(!intent||!service||service.busy||intent.binding!==binding)return false
    var connection=service.connections.filter(function(c){return c.name===root.intent.runner&&c.runner_id===root.intent.runner_id})[0]
    return !!connection&&Model.writable(connection,service.now)&&launch(true)
  }
  function launch(resuming) {
    var args=["--runner",intent.runner,"--expected-runner-id",intent.runner_id,"project-create","--label",intent.label,"--request-id",intent.request,"--shared-administration","--confirm-start"]
    if(intent.project)args=["--runner",intent.runner,"--expected-runner-id",intent.runner_id,"project-add",intent.project,"--request-id",intent.request,"--confirm-start"]
    intent.members.forEach(function(m){args.push("--member",m.instance_id+":"+m.ship+":"+m.http_port)})
    if(resuming)args.push("--resume")
    error="";errors="";completed=false;progress=({phase:"working",step:"reserve"})
    helper.command=service.command(args);helper.running=true;deadline.restart();return true
  }
  Timer {id:deadline;interval:1560000;onTriggered:{root.error="Creation deadline reached. Read saved project history before continuing.";helper.signal(15);hardStop.restart()}}
  Timer {id:hardStop;interval:5000;onTriggered:if(helper.running)helper.signal(9)}
  Process {
    id:helper
    stdout:SplitParser {onRead:function(data){if(data.length>8192)return;try{var p=JSON.parse(data);if(["working","completed","ready"].indexOf(p.phase)>=0)root.progress=p}catch(_){}}}
    stderr:SplitParser {splitMarker:"";onRead:function(data){if(root.errors.length+data.length<=8192)root.errors+=data}}
    onExited:function(code,status){
      deadline.stop();hardStop.stop()
      if(root.service&&typeof root.service.recordHelperExit==="function")root.service.recordHelperExit("project-create",status)
      root.completed=code===0&&status===0&&root.progress.phase==="ready"
      if(!root.completed){var p=null;try{p=JSON.parse(root.errors).error}catch(_){};root.error=p&&p.message?p.message:"Creation stopped. Saved ships and receipts remain; inspect project history before continuing."}
      root.errors="";root.finished()
    }
  }
}
