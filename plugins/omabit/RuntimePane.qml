import QtQuick
import QtQuick.Controls as Controls
import QtQuick.Layouts
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
  property bool showReading: false
  property string reportedComet: ""
  property string reportedContainer: ""
  readonly property bool reading: reader.running
  readonly property string selection: connection && instance ? connection.name+"/"+connection.runner_id+"/"+instance.id+"/"+instance.config_revision : ""
  readonly property bool readable: active && !!service && service.observing && Model.available(connection,service.now) && connection.access_role==="operator" && Model.hasFeature(connection,"runtime-logs")
  readonly property var diagnostic: instance && instance.runtime_status || ({})
  readonly property var operation: diagnostic.last_operation || null
  spacing: Style.space(12)
  onSelectionChanged: { logs=null;error="";reportedComet="";reportedContainer="";if (reader.running) reader.signal(15);refreshSoon.restart() }
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
      Label { objectName:"runtime-reported-comet";width:parent.width;visible:root.readable && !!root.reportedComet && !!root.instance && root.instance.mode==="comet" && !root.instance.ship;text:"Vere reported "+root.reportedComet+" · verifying startup";color:Design.accent }
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
      RowLayout {
        width:parent.width;spacing:Style.space(8)
        ActionButton { objectName:"runtime-logs-refresh";text:"Refresh logs";enabled:root.readable;hint:root.reading ? "A log refresh is already in progress" : "Read the latest runtime output";onClicked:root.refresh() }
        ActionButton { objectName:"runtime-logs-follow";text:root.following ? "Pause updates" : "Follow output";enabled:root.readable;onClicked:{root.following=!root.following;if(root.following)root.refresh()} }
        Item { Layout.preferredWidth:Style.space(22);Layout.preferredHeight:Style.space(22)
          Controls.BusyIndicator { objectName:"runtime-log-progress";anchors.fill:parent;running:root.showReading;opacity:root.showReading ? 1 : 0 }
        }
        Item { Layout.fillWidth:true }
      }
      Label { objectName:"runtime-log-summary";width:parent.width;text:root.logs ? Model.runtimeNarration(root.logs.lines) : "";visible:text!=="";color:Design.accent }
      Label { width:parent.width;color:Design.danger;text:root.error;visible:text!=="" }
      Label { width:parent.width;caption:true;secondary:true;wrapMode:Text.NoWrap;elide:Text.ElideRight;text:!root.readable ? "Logs need an available operator connection and an updated runner." : !root.logs ? "Waiting for runtime output…" : root.logs.source==="none" ? "No runtime log is available yet." : (root.logs.source==="retained" ? "Retained output from a removed container" : "Latest container output")+" · "+Model.humanTime(root.logs.observed_at,root.service.now)+(root.logs.truncated ? " · tail truncated" : "")+(!root.following ? " · paused" : "") }
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
  Timer { id:readingDelay;interval:250;onTriggered:root.showReading=reader.running }
  Process {
    id:reader
    onRunningChanged: {if(running)readingDelay.restart();else{readingDelay.stop();root.showReading=false}}
    stdout:SplitParser {splitMarker:"";onRead:function(data){if(root.output.length+data.length>262144){root.oversized=true;reader.signal(9)}else root.output+=data}}
    stderr:SplitParser {splitMarker:"";onRead:function(data){if(root.errors.length+data.length<16384)root.errors+=data}}
    onExited:function(code,status) {
      if(root.requested!==root.selection || !root.readable){root.output="";root.errors="";refreshSoon.restart();return}
      if(code!==0 || status!==0 || root.oversized)root.error="Logs could not be read. "+(root.oversized ? "Response exceeded its limit." : Model.errorMessage(root.errors))
      else try {
        var next=Model.validateRuntimeLogs(JSON.parse(root.output),root.connection.runner_id,root.instance.id,root.instance.config_revision)
        if(next.container_id!==root.reportedContainer){root.reportedComet="";root.reportedContainer=next.container_id || ""}
        var comet=Model.reportedCometName(next.lines)
        if(comet)root.reportedComet=comet
        root.logs=next;root.error=""
      }catch(e){root.error=e.message}
      root.output="";root.errors=""
    }
  }
}
