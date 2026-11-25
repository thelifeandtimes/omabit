import QtQuick
import qs.Commons
import "Model.js" as Model

Column {
  id: root
  property bool supported: false
  property int selectedLoom: 31
  property int preservedLoom: 31
  readonly property var loomChoices: {var values=[30,31,32,33,34];if(preservedLoom>34 && preservedLoom<=40)values.push(preservedLoom);return values}
  onSelectedLoomChanged: loom.currentIndex=loomChoices.indexOf(selectedLoom)
  readonly property string problem: loomChoices.indexOf(selectedLoom)<0 ? "Choose a loom exponent in 30..34 (1–16 GiB), or preserve the current CLI selection." : ""
  signal edited()
  spacing: Style.space(8)
  Label { text: "Loom address space"; caption: true; secondary: true }
  SelectField {
    id: loom; objectName: "runtime-loom"; width: parent.width; enabled: root.supported
    model: root.loomChoices.map(function(n){return Model.loomLabel(n)})
    currentIndex: root.loomChoices.indexOf(root.selectedLoom)
    onActivated: { root.selectedLoom=root.loomChoices[currentIndex]; root.edited() }
  }
  Label { width: parent.width; text: "Address space is not resident memory (RSS). Container RAM has a separate limit (4 GiB by default). Check host capacity and container usage separately."; caption: true; secondary: true }
  Label { width: parent.width; visible: !root.supported; text: "Update this runner for configurable loom (runtime-settings). Existing settings are preserved."; caption: true; color: Design.warning }
}
