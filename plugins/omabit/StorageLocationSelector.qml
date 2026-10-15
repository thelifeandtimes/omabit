import QtQuick
import qs.Commons
import "Model.js" as Model

Column {
  id: root
  property var connection
  readonly property var roots: Model.storageRoots(connection)
  readonly property string rootID: (roots[selector.currentIndex] || roots[0]).id
  signal edited()
  spacing: Style.space(6)
  Label { text: "Storage location on runner host"; caption: true; secondary: true }
  SelectField { id: selector; objectName: "storage-location-selector"; width: parent.width; model: root.roots.map(function(r){return r.id+" · "+r.path}); onActivated: root.edited() }
}
