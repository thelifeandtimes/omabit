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
  property bool priorityOnly: false
  property var logs: null
  property string error: ""
  property string output: ""
  property string errors: ""
  property string requested: ""
  property bool watching: false
  property bool received: false
  property bool stopping: false
  property bool pending: false
  property bool blocked: false
  property bool showReading: false
  property string reportedComet: ""
  property string reportedContainer: ""
  readonly property bool reading: reader.running && !received
  readonly property bool running: reader.running
  // Enrollment changes matter; replacing an inventory object every two seconds
  // does not. A transient inventory failure must not reset the watcher's backoff.
  readonly property string selection: connection && instance && service ? JSON.stringify([connection.name,connection.runner_id,connection.mode,instance.id,instance.config_revision,service.collector,service.settings.configDir || "",(service.registrations || []).filter(function(r){return r.name===root.connection.name})]) : ""
  readonly property bool eligible: active && !!service && service.observing && !!instance && !!connection && (connection.status==="unavailable" || connection.access_role==="operator" && connection.status!=="identity_mismatch" && Model.hasFeature(connection,"runtime-logs"))
  readonly property bool readable: eligible && Model.available(connection,service.now)
  readonly property var displayedLines: logs ? (priorityOnly ? Model.priorityRuntimeLogs(logs.lines) : logs.lines) : []
  spacing: Style.space(12)
  onSelectionChanged: { clearLogs();priorityOnly=false;error="";blocked=false;pending=following;stopReader();refreshSoon.restart() }
  onEligibleChanged: {
    if (!eligible) { pending=false;clearLogs();stopReader() }
    else if (following && !blocked) { pending=true;refreshSoon.restart() }
  }
  onReadableChanged: if (readable) refreshSoon.restart()
  onFollowingChanged: {
    pending=following
    if (following) blocked=false
    stopReader();refreshSoon.restart()
  }
  Timer { id:refreshSoon;interval:0;onTriggered:root.startReader() }
  function clearLogs() { logs=null;reportedComet="";reportedContainer="" }
  function stopReader() {
    if (!reader.running || stopping) return
    stopping=true;reader.signal(15);stopDeadline.restart()
  }
  function refresh() {
    if (!readable || reader.running) return
    blocked=false;pending=true;startReader()
  }
  function startReader() {
    if (!pending || blocked || !readable || reader.running) return
    pending=false;stopping=false;received=false
    requested=selection;output="";errors="";error="";watching=following
    var args=Model.selection(connection).concat(["logs",instance.id,"--revision",String(instance.config_revision)])
    if (watching) args=args.concat(["--watch","--interval","3s"])
    reader.command=service.command(args)
    reader.running=true
    Qt.callLater(function() {
      if (!reader.running && !stopping && root.requested===root.selection && !root.received) {
        root.blocked=true;root.error="Log reader could not start. Check the collector and Refresh logs to retry."
      }
    })
  }
  function acceptLogs(next) {
    if(next.container_id!==reportedContainer){reportedComet="";reportedContainer=next.container_id || ""}
    var comet=Model.reportedCometName(next.lines)
    if(comet)reportedComet=comet
    logs=next;error=""
  }
  function failRecord(message) {
    blocked=true;pending=false;clearLogs();error=message;output="";stopReader()
  }
  function acceptRecord(record) {
    try {
      var update=Model.validateRuntimeLogUpdate(JSON.parse(record),connection.runner_id,instance.id,instance.config_revision)
      received=true
      if (update.error) {
        clearLogs();error=update.error.message+(update.retry_after_ms ? " Retry in "+Math.ceil(update.retry_after_ms/1000)+"s." : "")
        if (!update.retry_after_ms) { blocked=true;pending=false;stopReader() }
      } else acceptLogs(update.logs)
    } catch(e) { failRecord(e.message) }
  }
  function readChunk(data) {
    if (stopping || requested!==selection || !eligible) return
    // Split incrementally: bounded per NDJSON record, never per stream lifetime.
    // 512 KiB accommodates JSON escaping of the server's 64 KiB bounded tail.
    var offset=0
    do {
      var end=watching ? data.indexOf("\n",offset) : -1
      var part=data.slice(offset,end<0 ? data.length : end)
      if (output.length+part.length>524288) { failRecord("Log response exceeded its limit.");return }
      output+=part
      if (end<0) return
      var record=output;output=""
      acceptRecord(record)
      if (stopping) return
      offset=end+1
    } while (offset<data.length)
  }
  Card {
    width:parent.width;implicitHeight:logBody.implicitHeight+Style.space(28)
    Column {
      id:logBody
      anchors.left:parent.left;anchors.right:parent.right;anchors.top:parent.top;anchors.margins:Style.space(14);spacing:Style.space(8)
      RowLayout {
        objectName: "runtime-logs-header"; width: parent.width; spacing: Style.space(6)
        CardTitle {
          objectName:"runtime-logs-title";text:"Urbit Logs"; Layout.fillWidth: true
          HoverHandler { id: logTitleHover }
          Hint { text:"Read-only Vere stdout/stderr · one connection while following · known key/code patterns redacted";visible:logTitleHover.hovered }
          Accessible.description: "Read-only Vere output. Known key/code patterns are redacted."
        }
        Item { Layout.preferredWidth:Style.space(22);Layout.preferredHeight:Style.space(22)
          Controls.BusyIndicator { objectName:"runtime-log-progress";anchors.fill:parent;running:root.showReading;opacity:root.showReading ? 1 : 0 }
        }
        ActionButton { objectName:"runtime-logs-follow";text:root.following ? "Ⅱ" : "▶";compact:true;quiet:true;Layout.preferredWidth:Style.space(30);hint:root.following ? "Pause log updates" : "Follow log output";Accessible.name:hint;enabled:root.following || root.readable;onClicked:root.following=!root.following }
        ActionButton { objectName:"runtime-logs-refresh";text:"↻";compact:true;quiet:true;Layout.preferredWidth:Style.space(30);enabled:root.readable;hint:root.running ? "The current reader is updating output; pause for a one-shot refresh" : "Refresh logs";Accessible.name:"Refresh logs";onClicked:root.refresh() }
      }
      Flow {
        width:parent.width;spacing:Style.space(8)
        SwitchField {
          id: priorityToggle
          objectName: "runtime-logs-filter"; text: "Priority logs only"
          height: Style.space(36); checked: root.priorityOnly; hoverEnabled: true
          onToggled: root.priorityOnly=checked
          Accessible.description: "Off shows all available output. On shows error/warning matches with surrounding context."
          Hint { text:"Priority logs match error/warning text in the available tail, with surrounding lines for context. This filter does not determine ship health.";visible:priorityToggle.hovered }
        }
      }
      Label { objectName:"runtime-reported-comet";width:parent.width;visible:root.readable && !!root.reportedComet && !!root.instance && root.instance.mode==="comet" && !root.instance.ship;text:"Vere reported "+root.reportedComet+" · verifying startup";color:Design.accent }
      Label { objectName:"runtime-log-error";width:parent.width;color:Design.danger;text:root.error ? "Log connection: "+root.error : "";visible:text!=="" }
      Label { width:parent.width;caption:true;secondary:true;wrapMode:Text.NoWrap;elide:Text.ElideRight;text:!root.readable ? "Logs need an available operator connection and an updated runner." : root.blocked ? "Log updates stopped · Refresh logs after reviewing the error." : !root.logs ? (root.following ? "Waiting for runtime output…" : "Updates paused · Refresh logs to read once.") : root.logs.source==="none" ? "No runtime log is available yet." : (root.logs.source==="retained" ? "Retained output from a removed container" : "Latest container output")+" · "+Model.humanTime(root.logs.observed_at,root.service.now)+(root.logs.truncated ? " · tail truncated" : "")+(!root.following ? " · paused" : "") }
      Label { objectName:"runtime-log-filter-status";width:parent.width;caption:true;secondary:true;visible:root.priorityOnly && !!root.logs;text:!root.logs ? "" : root.displayedLines.length ? "Priority logs with context · "+root.displayedLines.length+" of "+root.logs.lines.length+" lines" : "No priority entries in the latest "+root.logs.lines.length+" log lines. Turn off Priority logs only to see all available output." }
      Controls.ScrollView {
        width:parent.width;height:Style.space(250);visible:root.displayedLines.length>0
        contentWidth:availableWidth
        Controls.TextArea {
          objectName:"runtime-log-text"
          text:root.displayedLines.join("\n")
          textFormat:TextEdit.PlainText;readOnly:true;selectByMouse:true;wrapMode:TextEdit.WrapAnywhere
          color:Design.ink;selectionColor:Design.accent;selectedTextColor:Design.background
          font.family:"monospace";font.pixelSize:Style.space(12)
          background:Rectangle {color:Design.background;radius:Style.space(5)}
          onTextChanged:if(root.following)cursorPosition=length
        }
      }
    }
  }
  onReadingChanged: { if(reading)readingDelay.restart();else{readingDelay.stop();showReading=false} }
  Timer { id:readingDelay;interval:250;onTriggered:root.showReading=root.reading }
  Timer { id:stopDeadline;interval:6000;onTriggered:if(reader.running && root.stopping)reader.signal(9) }
  Process {
    id:reader
    stdout:SplitParser {splitMarker:"";onRead:function(data){root.readChunk(data)}}
    stderr:SplitParser {splitMarker:"";onRead:function(data){if(root.errors.length+data.length<16384)root.errors+=data}}
    onExited:function(code,status) {
      stopDeadline.stop()
      if (!root.stopping && root.requested===root.selection && root.eligible) {
        if (root.watching || code!==0 || status!==0) {
          root.blocked=true;root.clearLogs()
          if (!root.error) root.error="Log reader stopped. "+Model.errorMessage(root.errors)+" Refresh logs to retry."
        } else try {
          root.acceptLogs(Model.validateRuntimeLogs(JSON.parse(root.output),root.connection.runner_id,root.instance.id,root.instance.config_revision))
          root.received=true
        } catch(e) { root.blocked=true;root.clearLogs();root.error=e.message }
      }
      root.stopping=false
      root.output="";root.errors=""
      refreshSoon.restart()
    }
  }
}
