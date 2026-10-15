import QtQuick
import qs.Commons
import "Model.js" as Model

Column {
  id: root
  property var connection
  property bool allowCustom: false
  readonly property bool customPath: allowCustom && selector.currentIndex===roots.length
  readonly property var roots: Model.storageRoots(connection)
  readonly property string rootID: (roots[customPath ? customStorage.currentIndex : selector.currentIndex] || roots[0]).id
  signal edited()
  spacing: Style.space(6)
  Label { text: "Storage location on runner host"; caption: true; secondary: true }
  SelectField { id: selector; objectName: "storage-location-selector"; width: parent.width; model: root.roots.map(function(r){return r.id+" · "+r.path}).concat(root.allowCustom ? ["Custom path — /custom/path/to/pier"] : []); onActivated: root.edited() }
  SelectField { id: customStorage; objectName: "custom-storage-location"; width: parent.width; visible: root.customPath; model: root.roots.map(function(r){return r.id+" · "+r.path}); onActivated: root.edited() }
  Label { width: parent.width; visible: root.customPath; caption: true; secondary: true; text: "Choose a configured storage location and a pier path within it. Add another host directory in Setup → Storage locations." }
}
