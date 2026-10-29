import QtQuick
import "collection" as Admission
Admission.SharedHost {
  pluginId: "thelifeandtimes.urbit-theme.qualification-stable"
  profile: "qualification-stable"
  artifact: "ed20c91e931071c489f72d252bdc615f902778e62b489595dcaf5c8f987d9030"
  role: "theme"
  runtimeSource: Qt.resolvedUrl("CollectionRuntime.qml")
  guardSource: Qt.resolvedUrl("collection/continuation_admission_guard.py")
  collectionRegistry: Admission.Registry
}
