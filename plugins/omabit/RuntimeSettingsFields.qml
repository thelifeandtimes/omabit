import QtQuick
import qs.Commons
import "Model.js" as Model

Column {
  id: root
  property string mode: "fake"
  property bool supported: false
  property bool allShipsSupported: false
  property int selectedLoom: 31
  onSelectedLoomChanged: loom.currentIndex=selectedLoom-20
  property alias nativeControl: nativeToggle.checked
  readonly property bool nativeSupported: supported && (mode === "fake" || allShipsSupported)
  readonly property string problem: selectedLoom < 20 || selectedLoom > 34 ? "Choose a loom exponent in 20..34." : nativeControl && !nativeSupported ? "Update this runner for " + (mode === "fake" ? "runtime-settings" : "native-control-all-ships") + "." : ""
  signal edited()
  spacing: Style.space(8)
  Label { text: "Loom address space"; caption: true; secondary: true }
  SelectField {
    id: loom; objectName: "runtime-loom"; width: parent.width; enabled: root.supported
    model: { var options=[]; for(var n=20;n<=34;n++) options.push(Model.loomLabel(n)); return options }
    currentIndex: root.selectedLoom - 20
    onActivated: { root.selectedLoom=20+currentIndex; root.edited() }
  }
  Label { objectName: "runtime-loom-size"; width: parent.width; text: Model.loomLabel(root.selectedLoom); caption: true }
  Label { width: parent.width; text: "Address space is not resident memory (RSS). The container memory cap remains 4 GiB; check available RAM and container usage separately."; caption: true; secondary: true }
  Label { width: parent.width; visible: !root.supported; text: "Update this runner for configurable loom and native control (runtime-settings). Existing settings are preserved."; caption: true; color: Design.warning }
  Label { width: parent.width; visible: root.supported && !root.nativeSupported; text: "Update this runner for native-control-all-ships to enable conn.sock on this ship. This backend supports native control only for fakes. Existing settings are preserved."; caption: true; color: Design.warning }
  CheckField { id: nativeToggle; objectName: "runtime-native-control"; width: parent.width; enabled: root.nativeSupported; checked: false; text: "Enable local-owner native control"; onToggled: root.edited() }
  Label { width: parent.width; text: "Warning · known ship-crash bug"; font.bold: true; color: Design.warning }
  Label {
    objectName: "runtime-native-warning"
    width: parent.width
    text: "Primarily for development. Leave disabled for routine use: it is unnecessary and adds risk.\n\nThis pinned Vere build has a known crash bug when a client disconnects with replies pending. Keep clients connected until every reply has been received. Do not kill clients or use short timeouts.\n\nconn.sock grants full ship-control authority, not health-only access."
    color: Design.warning
  }
}
