import QtQuick
import "collection" as Admission
Admission.SharedHost {
  pluginId: "thelifeandtimes.urbit-theme.qualification-stable"
  profile: "qualification-stable"
  artifact: "4af432d8af91bfca51c116fccbcb7028830147defe9b9886bf33225c8e2b54eb"
  role: "theme"
  runtimeSource: Qt.resolvedUrl("CollectionRuntime.qml")
  guardSource: Qt.resolvedUrl("collection/shared_admission_guard.py")
  collectionRegistry: Admission.Registry
}
