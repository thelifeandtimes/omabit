import QtQuick
import qs.Commons
import "Model.js" as Model

Column {
  id: root
  property bool supported: false
  property int selectedLoom: 31
  onSelectedLoomChanged: loom.currentIndex=selectedLoom-30
  readonly property string problem: selectedLoom < 30 || selectedLoom > 34 ? "Choose a loom exponent in 30..34 (1–16 GiB)." : ""
  signal edited()
  spacing: Style.space(8)
  Label { text: "Loom address space"; caption: true; secondary: true }
  SelectField {
    id: loom; objectName: "runtime-loom"; width: parent.width; enabled: root.supported
    model: { var options=[]; for(var n=30;n<=34;n++) options.push(Model.loomLabel(n)); return options }
    currentIndex: root.selectedLoom - 30
    onActivated: { root.selectedLoom=30+currentIndex; root.edited() }
  }
  Label { width: parent.width; text: "Address space is not resident memory (RSS). The container memory cap remains 4 GiB; check available RAM and container usage separately."; caption: true; secondary: true }
  Label { width: parent.width; visible: !root.supported; text: "Update this runner for configurable loom (runtime-settings). Existing settings are preserved."; caption: true; color: Design.warning }
}
