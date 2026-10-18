import QtQuick
import qs.Commons
import "Model.js" as Model

Column {
  id: root
  property var connection
  property bool allowCustom: false
  property string selectedRoot: "default"
  property bool customSelected: false
  readonly property string selection: connection ? connection.runner_id+"/"+connection.name : ""
  onSelectionChanged: reset()
  onRootsChanged: syncSelection()
  Component.onCompleted: syncSelection()
  function syncSelection() {
    var index=(roots || []).map(function(r){return r.id}).indexOf(selectedRoot)
    if (selector) selector.currentIndex=customPath ? roots.length : index
    if (customStorage) customStorage.currentIndex=index
  }
  function reset() { selectedRoot="default";customSelected=false;syncSelection() }
  function selectRoot(id) {
    selectedRoot=id;customSelected=false;syncSelection()
    return roots.some(function(r){return r.id===id})
  }
  readonly property bool customPath: allowCustom && customSelected
  readonly property var roots: Model.storageRoots(connection)
  readonly property string rootID: roots.some(function(r){return r.id===selectedRoot}) ? selectedRoot : ""
  signal edited()
  spacing: Style.space(6)
  Label { text: "Storage location on runner host"; caption: true; secondary: true }
  SelectField { id: selector; objectName: "storage-location-selector"; width: parent.width; model: root.roots.map(function(r){return r.id+" · "+r.path}).concat(root.allowCustom ? ["Custom path — /custom/path/to/pier"] : []); onModelChanged: root.syncSelection(); onActivated: { root.customSelected=root.allowCustom && currentIndex===root.roots.length;if(!root.customPath)root.selectedRoot=(root.roots[currentIndex] || {}).id || "";root.syncSelection();root.edited() } }
  SelectField { id: customStorage; objectName: "custom-storage-location"; width: parent.width; visible: root.customPath; model: root.roots.map(function(r){return r.id+" · "+r.path}); onModelChanged: root.syncSelection(); onActivated: { root.selectedRoot=(root.roots[currentIndex] || {}).id || "";root.syncSelection();root.edited() } }
  Label { width: parent.width; visible: root.customPath; caption: true; secondary: true; text: "Choose a configured storage location and a pier path within it. Add another host directory in Setup → Storage locations." }
}
