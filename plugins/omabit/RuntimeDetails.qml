import QtQuick
import qs.Commons
import "Model.js" as Model

Column {
  id: root
  property var connection
  property var instance
  property double now: Date.now()
  readonly property var diagnostic: instance && instance.runtime_status || ({})
  readonly property var operation: diagnostic.last_operation || null
  spacing: Style.space(8)
  CardTitle { text: "Runtime details" }
  Label {
    objectName:"runtime-profile-details";width:parent.width;caption:true;secondary:true
    text:"Requested runtime: "+(root.diagnostic.requested_runtime ? root.diagnostic.requested_runtime.id : "unavailable")+"\nRecorded launch: "+(root.diagnostic.launch_runtime ? root.diagnostic.launch_runtime.id : "none recorded")+"\nObserved image: "+(root.diagnostic.observed_image_id || "unavailable")
  }
  Label { width:parent.width;caption:true;color:Design.warning;visible:!!root.diagnostic.requested_runtime&&!!root.diagnostic.requested_runtime.requirement;text:"This pier requires the reviewed JSON decoder fix. Stock Vere can crash replay; retain the runtime artifact and recovery backups." }
  Label { objectName: "runtime-status-detail"; width: parent.width; text: Model.runtimeDetail(root.instance); caption: true; secondary: true }
  Label {
    width: parent.width; caption: true; secondary: true; visible: !!root.operation
    text: root.operation ? "Operation: "+root.operation.id+"\nStarted: "+Model.humanTime(root.operation.created_at,root.now)+(root.operation.finished_at ? " · Finished: "+Model.humanTime(root.operation.finished_at,root.now) : " · Elapsed: "+Math.max(0,Math.floor((root.now-Date.parse(root.operation.created_at))/1000))+"s") : ""
  }
  Label {
    width: parent.width; caption: true; secondary: true; visible: !!root.diagnostic.container_id
    text: "Container: "+root.diagnostic.container_id+"\n"+(root.diagnostic.container_kind || "")+(root.diagnostic.observed_at ? " · Observed "+Model.humanTime(root.diagnostic.observed_at,root.now) : "")
  }
  HostCompatibility {
    objectName: "runtime-host-compatibility"; width: parent.width; details: true
    report: Model.hostCompatibility(root.connection)
    envelope: root.connection ? root.connection.host_compatibility || null : null
    problem: Model.hostCompatibilityHint(root.connection,root.now)
    now: root.now
  }
}
