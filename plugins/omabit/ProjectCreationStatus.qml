import QtQuick
import qs.Commons
Column {
  id: root
  property var service
  readonly property var controller: service ? service.projectCreation : null
  spacing: Style.space(8)
  Label { width: parent.width; visible: !!root.controller && root.controller.busy; text: root.controller ? "Creating project · " + (root.controller.progress.step || "preparing") : "" }
  Label { width: parent.width; visible: !!root.controller && root.controller.completed; text: root.controller && root.controller.intent && root.controller.intent.members.length===0 ? "Project network is ready. Add a fake ship to join it." : "Project ship is ready. Open it from your fleet."; color: Design.success }
  Label { width: parent.width; visible: !!root.controller && root.controller.error !== ""; text: root.controller ? root.controller.error : ""; color: Design.danger }
  ActionButton { visible: !!root.controller && !!root.controller.intent && !root.controller.busy && !root.controller.completed && root.controller.error !== ""; text: "Continue the same creation attempt"; enabled: !!root.service && !root.service.busy; onClicked: root.controller.resume() }
}
