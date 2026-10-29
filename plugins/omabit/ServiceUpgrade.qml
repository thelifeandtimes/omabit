import QtQuick
import Quickshell.Io

Item {
  id: root
  property var service
  property var status: null
  property string message: ""
  property string error: ""
  property bool pending: false
  property bool completed: false
  property bool invalidOutput: false
  readonly property bool busy: worker.running || pending
  readonly property bool canStart: !busy && allowed()
  property bool checking: false
  property var checkedResult: null
  function allowed() {
    return service && service.localSetupAllowed && service.activationEnabled && !service.transportStopping
      && !service.busy && !service.checkingSetup && !service.remoteSetupBusy
  }
  function valid(value) {
    return value && value.version === 1 && typeof value.runner_id === "string" && value.runner_id.indexOf("runner-") === 0
      && typeof value.target_sha256 === "string" && /^[0-9a-f]{64}$/.test(value.target_sha256)
      && typeof value.available === "boolean" && typeof value.pending === "boolean"
      && typeof value.installed_build === "string" && typeof value.target_build === "string" && typeof value.message === "string"
  }
  function check() {
    if (busy || !allowed()) return false
    error="";message="Checking local services…";checking=true;pending=true;completed=false;invalidOutput=false;checkedResult=null
    worker.command=service.themeCommand(["service-upgrade-status"]);worker.running=true
    return true
  }
  function upgrade() {
    if (busy || !allowed() || !valid(status) || !status.available) return false
    error="";message="Preparing service update…";checking=false;pending=true;completed=false;invalidOutput=false;checkedResult=null
    worker.command=service.themeCommand(["service-upgrade","--expected-runner-id",status.runner_id,"--target-sha256",status.target_sha256,"--acknowledge-host-changes"])
    worker.running=true
    return true
  }
  function accept(data) {
    if (invalidOutput) return
    try {
      if(data.length>65536) throw new Error("oversize response")
      var value=JSON.parse(data)
      if (value.status === "failed" && typeof value.message === "string") {error=value.message;return}
      if (checking) {
        if (!valid(value)) throw new Error("invalid check")
        checkedResult=value;completed=true
      } else {
        if (typeof value.step!=="string" || ["running","complete"].indexOf(value.status)<0) throw new Error("invalid progress")
        if (value.message) message=String(value.message)
        if (value.step==="upgrade" && value.status==="complete") {
          if (!valid(value.result)) throw new Error("invalid result")
          checkedResult=value.result;completed=true
        }
      }
    } catch (_) {
      invalidOutput=true;error="The update helper returned an unexpected response. Check services again before retrying."
      // Drain the helper to its real exit; killing this reader is not proof that
      // a privileged installer has finished.
    }
  }
  function finish(code, exitStatus) {
    if (!pending) return
    if(service && service.recordHelperExit) service.recordHelperExit("service-upgrade",exitStatus)
    pending=false
    if(code!==0 || exitStatus!==0 || !completed || invalidOutput || error) {
      if(!error) error="The service update did not finish. Check again, then retry the retained attempt."
      message="";status=null
    } else {
      status=checkedResult;message=checking ? status.message : "Service update completed. Your fleet is ready."
      if(!checking) Qt.callLater(function(){ if(root.service && !root.service.transportStopping) {root.service.refresh();root.service.checkLocalSetup()} })
    }
  }
  Process {
    id: worker
    objectName: "service-upgrade-process"
    stdout: SplitParser { onRead: data => root.accept(data) }
    stderr: SplitParser { onRead: data => { if(data.length && !root.error) root.error="The service helper reported an error. Check services again for recovery." } }
    onExited: (code, exitStatus) => root.finish(code,exitStatus)
    onRunningChanged: if(!running && root.pending) Qt.callLater(function(){if(!worker.running && root.pending)root.finish(-1,1)})
  }
}
