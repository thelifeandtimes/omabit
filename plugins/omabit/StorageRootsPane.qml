import QtQuick
import qs.Commons
import "Model.js" as Model

Column {
  id: root
  property var service
  property var connection
  spacing: Style.space(8)
  Label { text: "Storage locations"; font.bold: true }
  Repeater { model: Model.storageRoots(root.connection); Label { required property var modelData; width: root.width; text: modelData.id+" · "+modelData.path; secondary: true } }
  Label { width: parent.width; text: "Each location uses the same ship-directory/pier layout. Choose a location when creating or importing a ship. Adding one also includes it in discovery; existing ships stay at their recorded locations."; secondary: true; caption: true }
  Field { id: name; objectName: "storage-root-name"; width: parent.width; placeholderText: "Location name, e.g. ssd"; enabled: !root.service.busy }
  Field { id: path; objectName: "storage-root-path"; width: parent.width; placeholderText: "Absolute host directory, e.g. /mnt/ssd/urbit-piers"; enabled: !root.service.busy }
  Label { width: parent.width; text: "The directory must already belong to urbit-runner, with access through its parents. Prepare filesystem permissions on that host before adding it. Locations are identity-pinned and cannot be silently retargeted to a different disk."; secondary: true; caption: true }
  ActionButton { objectName: "storage-root-add"; text: "Add storage location"; enabled: root.connection && Model.hasFeature(root.connection,"storage-locations") && !root.service.busy; onClicked: root.service.addStorageRoot(root.connection,name.text.trim(),path.text.trim()) }
}
