import QtQuick
import qs.Commons
import "Model.js" as Model

Column {
  id: root
  objectName: "host-move-pane"
  property var service
  property var connection
  property var instance
  property var target: null
  property var destination: null
  readonly property string selection: connection && instance ? connection.runner_id+"/"+instance.id : ""
  readonly property var handoff: instance && instance.handoff || null
  readonly property bool outgoing: !!handoff && handoff.direction==="out"
  readonly property var candidates: service ? service.connections.filter(function(c){return root.connection && c.runner_id!==root.connection.runner_id && c.access_role==="operator" && Model.hasFeature(c,"host-handoff")}) : []
  readonly property var pinnedDestination: outgoing ? candidates.filter(function(c){return c.runner_id===root.handoff.destination_runner})[0] || null : null
  readonly property var selectedDestination: outgoing ? pinnedDestination ? {connection:pinnedDestination.name,runner:pinnedDestination.runner_id} : null : destination
  readonly property string destinationName: outgoing ? handoff.destination_instance : name.text.trim()
  readonly property var outcome: service && service.hostMoveOutcome && connection && instance && service.hostMoveOutcome.runner===connection.runner_id && service.hostMoveOutcome.instance===instance.id ? service.hostMoveOutcome : null
  readonly property string problem: target ? Model.hostMoveProblem(service.connections,target,selectedDestination,destinationName,httpPort.text,amesPort.text,service.now) : ""
  visible: Model.hasFeature(connection,"host-handoff") || !!handoff
  spacing: Style.space(10)
  onSelectionChanged: {target=null;destination=null}
  function review() {target=Model.edit(connection,instance);destination=null;name.text=instance.id;httpPort.text="";amesPort.text="";chooser.currentIndex=0}

  Label { text: "Move to another host"; font.bold: true }
  Label { width: parent.width; text: root.handoff ? root.handoff.error || root.handoff.message || "Host move in progress." : "Stop this ship first. The move transfers its current pier, permanently fences the source, and leaves the destination stopped."; secondary: true }
  ActionButton { objectName: "review-host-move"; text: root.outgoing ? "Review / continue move…" : "Choose destination…"; enabled: root.service && !root.service.busy && root.service.observing && Model.canMoveHost(root.connection,root.instance,root.service.now); onClicked: root.review() }
  Column {
    width: parent.width; spacing: Style.space(10); visible: !!root.target
    SelectField {
      id: chooser; objectName: "host-move-destination"; width: parent.width; visible: !root.outgoing
      enabled: !root.service.busy
      model: ["Choose a destination runner…"].concat(root.candidates.map(function(c){return Model.runnerTitle(c,root.service.runnerPreferences)}))
      onActivated: {var c=root.candidates[currentIndex-1];root.destination=c ? {connection:c.name,runner:c.runner_id} : null}
    }
    Label { width: parent.width; visible: !!root.selectedDestination || root.outgoing; text: root.selectedDestination ? "Destination: "+root.selectedDestination.connection+" · "+root.selectedDestination.runner : "The selected destination runner is unavailable."; caption: true; secondary: true }
    Field { id: name; objectName: "host-move-instance"; width: parent.width; visible: !root.outgoing; placeholderText: "Destination instance name"; enabled: !root.service.busy }
    Label { width: parent.width; visible: root.outgoing; text: "Destination instance: "+root.destinationName; caption: true; secondary: true }
    Field { id: httpPort; objectName: "host-move-http"; width: parent.width; placeholderText: "Destination HTTP port (blank allocates / retains)"; enabled: !root.service.busy }
    Field { id: amesPort; width: parent.width; visible: !!root.instance && root.instance.ames_published===true; placeholderText: "Destination Ames UDP port (blank allocates / retains)"; enabled: !root.service.busy }
    Label { width: parent.width; visible: root.problem!==""; text: root.problem; color: Design.warning }
    Label { width: parent.width; text: "Networking routes and maintenance schedules are host-specific. Review them on the destination before Start. Interrupted moves continue forward from their recorded phase."; caption: true; secondary: true }
    ActionButton { objectName: "submit-host-move"; text: root.outgoing ? "Continue stopped move" : "Move stopped ship"; primary: true; enabled: root.target && root.problem==="" && root.service.observing && !root.service.busy; onClicked: root.service.moveHost(root.target,root.selectedDestination,root.destinationName,httpPort.text,amesPort.text) }
    ActionButton { text: "Close review"; quiet: true; onClicked: root.target=null }
  }
  Label { width: parent.width; visible: !!root.outcome; text: root.outcome ? root.outcome.message : ""; secondary: true }
}
