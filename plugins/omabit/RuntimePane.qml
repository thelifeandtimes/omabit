import QtQuick
import QtQuick.Controls as Controls
import Quickshell.Io
import qs.Commons
import "Model.js" as Model

Column {
  id: root
  objectName: "runtime-pane"
  property var service
  property var connection
  property var instance
  property bool active: false
  property bool following: true
  property var logs: null
  property string error: ""
  property string output: ""
  property string errors: ""
  property string requested: ""
  property bool oversized: false
  readonly property string selection: connection && instance ? connection.name+"/"+connection.runner_id+"/"+instance.id+"/"+instance.config_revision : ""
  readonly property bool readable: active && !!service && service.observing && Model.available(connection,service.now) && connection.access_role==="operator" && Model.hasFeature(connection,"runtime-logs")
  readonly property var diagnostic: instance && instance.runtime_status || ({})
  readonly property var operation: diagnostic.last_operation || null
  spacing: Style.space(12)
  onSelectionChanged: { logs=null;error="";if (reader.running) reader.signal(15);refreshSoon.restart() }
  onReadableChanged: { if (!readable) {logs=null;if(reader.running)reader.signal(15)} else refreshSoon.restart() }
  Timer { id:refreshSoon;interval:0;onTriggered:root.refresh() }
  function refresh() {
    if (!readable || reader.running) return
    requested=selection;output="";errors="";oversized=false
    reader.command=service.command(Model.selection(connection).concat(["logs",instance.id,"--revision",String(instance.config_revision)]))
    reader.running=true
  }
  Card {
    width: parent.width; implicitHeight: statusBody.implicitHeight+Style.space(28)
    Column {
      id: statusBody
      anchors.left: parent.left;anchors.right:parent.right;anchors.top:parent.top;anchors.margins:Style.space(14);spacing:Style.space(8)
      Label { text: "Runtime status";font.bold:true }
      Label { objectName:"runtime-status-detail";width:parent.width;text:Model.runtimeDetail(root.instance);color:root.instance && root.instance.recovery_required ? Design.danger : Design.ink }
      Label { width:parent.width;caption:true;secondary:true;visible:!!root.operation;text:root.operation ? "Operation: "+root.operation.id+"\nStarted: "+Model.humanTime(root.operation.created_at,root.service.now)+(root.operation.finished_at ? " · Finished: "+Model.humanTime(root.operation.finished_at,root.service.now) : " · Elapsed: "+Math.max(0,Math.floor((root.service.now-Date.parse(root.operation.created_at))/1000))+"s") : "" }
      Label { width:parent.width;caption:true;secondary:true;visible:!!root.diagnostic.container_id;text:"Container: "+root.diagnostic.container_id+"\n"+(root.diagnostic.container_kind || "")+(root.diagnostic.observed_at ? " · Observed "+Model.humanTime(root.diagnostic.observed_at,root.service.now) : "") }
    }
  }
  Card {
    width:parent.width;implicitHeight:logBody.implicitHeight+Style.space(28)
    Column {
      id:logBody
      anchors.left:parent.left;anchors.right:parent.right;anchors.top:parent.top;anchors.margins:Style.space(14);spacing:Style.space(8)
      Label { text:"Runtime / dojo output";font.bold:true }
      Label { width:parent.width;caption:true;secondary:true;text:"Read-only Vere stdout/stderr · updates every 3 seconds · known key/code patterns redacted" }
      Flow {
        width:parent.width;spacing:Style.space(8)
        ActionButton { objectName:"runtime-logs-refresh";text:reader.running ? "Reading…" : "Refresh logs";enabled:root.readable&&!reader.running;onClicked:root.refresh() }
        ActionButton { objectName:"runtime-logs-follow";text:root.following ? "Pause updates" : "Follow output";enabled:root.readable;onClicked:{root.following=!root.following;if(root.following)root.refresh()} }
      }
      Label { objectName:"runtime-log-summary";width:parent.width;text:root.logs ? Model.runtimeNarration(root.logs.lines) : "";visible:text!=="";color:Design.accent }
      Label { width:parent.width;color:Design.danger;text:root.error;visible:text!=="" }
      Label { width:parent.width;caption:true;secondary:true;text:!root.readable ? "Logs need an available operator connection and an updated runner." : !root.logs ? "Waiting for runtime output…" : root.logs.source==="none" ? "No runtime log is available yet." : (root.logs.source==="retained" ? "Retained output from a removed container" : "Latest container output")+" · "+Model.humanTime(root.logs.observed_at,root.service.now)+(root.logs.truncated ? " · tail truncated" : "")+(!root.following ? " · paused" : "") }
      Controls.ScrollView {
        width:parent.width;height:Style.space(250);visible:!!root.logs&&root.logs.lines.length>0
        contentWidth:availableWidth
        Controls.TextArea {
          objectName:"runtime-log-text"
          text:root.logs ? root.logs.lines.join("\n") : ""
          textFormat:TextEdit.PlainText;readOnly:true;selectByMouse:true;wrapMode:TextEdit.WrapAnywhere
          color:Design.ink;selectionColor:Design.accent;selectedTextColor:Design.background
          font.family:"monospace";font.pixelSize:Style.space(12)
          background:Rectangle {color:Design.background;radius:Style.space(5)}
          onTextChanged:if(root.following)cursorPosition=length
        }
      }
    }
  }
  Timer { interval:3000;repeat:true;running:root.readable&&root.following;onTriggered:root.refresh() }
  Process {
    id:reader
    stdout:SplitParser {splitMarker:"";onRead:function(data){if(root.output.length+data.length>262144){root.oversized=true;reader.signal(9)}else root.output+=data}}
    stderr:SplitParser {splitMarker:"";onRead:function(data){if(root.errors.length+data.length<16384)root.errors+=data}}
    onExited:function(code,status) {
      if(root.requested!==root.selection || !root.readable){root.output="";root.errors="";refreshSoon.restart();return}
      if(code!==0 || status!==0 || root.oversized)root.error="Logs could not be read. "+(root.oversized ? "Response exceeded its limit." : Model.errorMessage(root.errors))
      else try {root.logs=Model.validateRuntimeLogs(JSON.parse(root.output),root.connection.runner_id,root.instance.id,root.instance.config_revision);root.error=""}catch(e){root.error=e.message}
      root.output="";root.errors=""
    }
  }
}
