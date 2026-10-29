import QtQuick
import "collection" as Admission
Admission.Host {
  pluginId: "thelifeandtimes.urbit-theme.qualification-stable"
  profile: "qualification-stable"
  artifact: "9f8b1abdf003c19c5daf26945a6cc6ff4fdba6bd5c86df06a9aeff1f5006a06d"
  role: "theme"
  runtimeSource: Qt.resolvedUrl("CollectionRuntime.qml")
  guardSource: Qt.resolvedUrl("collection/admission_guard.py")
  collectionRegistry: Admission.Registry
}
