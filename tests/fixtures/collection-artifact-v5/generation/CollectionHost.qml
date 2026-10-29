import QtQuick
import "collection" as Admission
Admission.SharedHost {
  pluginId: "thelifeandtimes.urbit-theme.qualification-stable"
  profile: "qualification-stable"
  artifact: "1a6a02479549c4ea16f241daf0526622935993fb85080f95293f8e5ccb3e77c3"
  role: "theme"
  runtimeSource: Qt.resolvedUrl("CollectionRuntime.qml")
  guardSource: Qt.resolvedUrl("collection/successor_admission_guard.py")
  collectionRegistry: Admission.Registry
}
