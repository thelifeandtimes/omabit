import QtQuick
import Quickshell.Io
import "Model.js" as Model

EditableSelectField {
  id: root
  property var service: null
  ReaderLifecycle { id: lifecycle; service: root.service; process: reader; kind: "ssh-suggestions" }
  property string configFile: ""
  property string output: ""
  property string requestedConfig: ""
  property var suggestions: []
  signal suggestionSelected(int port)
  textRole: "label"
  model: suggestions
  placeholderText: "Choose an SSH alias or type a custom host"
  Accessible.name: "SSH host or alias"
  Accessible.description: "Suggestions from local SSH configuration and readable known hosts. Custom hosts can be typed."
  onActivated: function(index) {
    var entry=suggestions[index]
    if(entry){editText=entry.host;suggestionSelected(entry.port)}
  }
  function acceptSuggestions(value) {
    if(!value || !Array.isArray(value.hosts) || value.hosts.length>256 || !value.hosts.every(function(row){return row && Model.validRemoteHost(row.host) && typeof row.label==="string" && row.label.length<=260 && Number.isInteger(row.port) && row.port>=0 && row.port<=65535}))throw new Error("Invalid SSH host suggestions")
    var typed=editText
    suggestions=value.hosts
    currentIndex=-1;editText=typed
  }
  function refreshSuggestions() {
    if(!visible || !lifecycle.admitted || reader.running)return
    output="";requestedConfig=configFile.trim()
    reader.command=["python3","-B",Model.localPath(Qt.resolvedUrl("ssh-hosts.py"))].concat(requestedConfig ? ["--config",requestedConfig] : [])
    reader.running=true
  }
  onVisibleChanged: {if(visible)refreshSoon.restart();else if(reader.running)reader.signal(15)}
  Connections {
    target: lifecycle
    function onAdmittedChanged() {
      if (lifecycle.admitted) refreshSoon.restart()
      else { refreshSoon.stop(); if (reader.running) reader.signal(15) }
    }
  }
  onConfigFileChanged: refreshSoon.restart()
  Component.onCompleted: refreshSoon.restart()
  Timer { id:refreshSoon;interval:300;onTriggered:root.refreshSuggestions() }
  Timer { id:deadline;interval:5000;onTriggered:if(reader.running)reader.signal(9) }
  Process {
    id:reader
    onRunningChanged: if(running)deadline.restart();else deadline.stop()
    stdout:SplitParser {splitMarker:"";onRead:function(data){if(root.output.length+data.length>262144){reader.signal(9);return}root.output+=data}}
    stderr:SplitParser {onRead:function(line){}}
    onExited:function(code,status) {
      if (!lifecycle.admitted || !root.visible) { root.output=""; return }
      if(root.requestedConfig!==root.configFile.trim()){refreshSoon.restart();return}
      if(code===0&&status===0){try{root.acceptSuggestions(JSON.parse(root.output))}catch(_) {}}
      root.output=""
    }
  }
}
