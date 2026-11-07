import QtQuick
import Quickshell.Io
import "Model.js" as Model

Item {
  id:root
  property var service:null
  property var value:null
  property string target:""
  property string error:""
  onErrorChanged:if(error && service && service.showToast)service.showToast(error,"bad")
  property string message:""
  property string operation:""
  property string output:""
  property string errors:""
  property var commandPrefix:["python3","-B",Model.localPath(Qt.resolvedUrl("agent-help.py"))]
  readonly property bool busy:worker.running
  readonly property bool available:!!service && service.activationEnabled && !service.transportStopping
  ReaderLifecycle {service:root.service;process:worker;kind:"agent-help"}
  function run(action,args) {
    if(!available || busy)return false
    error="";message="";output="";errors="";operation=action
    worker.command=commandPrefix.concat([action]).concat(args || [])
    worker.running=true;return true
  }
  function refresh(){return run("status",target ? ["--target",target] : [])}
  function launch(context){return run("launch",["--context-json",JSON.stringify(context)])}
  function install(row){return row && row.action!=="none" && run("install",["--target",target,"--skill",row.name,"--review",row.review])}
  onAvailableChanged:if(!available){value=null;if(worker.running)worker.signal(15)}
  Timer {id:deadline;interval:20000;onTriggered:if(worker.running)worker.signal(9)}
  Process {
    id:worker
    onRunningChanged:if(running)deadline.restart();else deadline.stop()
    stdout:SplitParser {splitMarker:"";onRead:function(data){if(root.output.length+data.length>131072){worker.signal(9);return}root.output+=data}}
    stderr:SplitParser {splitMarker:"";onRead:function(data){if(root.errors.length+data.length<=8192)root.errors+=data}}
    onExited:function(code,status) {
      if(!root.available)return
      var action=root.operation;root.operation=""
      if(code!==0 || status!==0){
        root.error="Could not complete the agent request. Refresh to inspect its current state."
        try{var failure=JSON.parse(root.errors);if(typeof failure.error==="string")root.error=failure.error}catch(_){}
        return
      }
      try {
        var response=JSON.parse(root.output)
        if(!response || response.version!==1)throw new Error("Unsupported agent response")
        if(action==="status"){
          if(typeof response.agent!=="string" || typeof response.target!=="string" || !Array.isArray(response.skills) || response.skills.length>2)throw new Error("Invalid agent status")
          root.value=response;root.target=response.target
        }else if(action==="install"){
          if(!response.installed || response.installed.state!=="installed")throw new Error("Skill installation needs review")
          Qt.callLater(function(){root.refresh();root.message="Skill installed. New agent sessions can discover it."})
        }else if(action==="launch"){
          if(typeof response.launched!=="string")throw new Error("Agent launch was not acknowledged")
          root.message="Opened "+response.launched+" with Omabit's bundled instructions."
          if(root.service.showToast)root.service.showToast(root.message)
        }
      }catch(e){root.error=e.message}
    }
  }
}
