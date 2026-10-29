import QtQuick
import "collection" as Admission
Admission.SharedHost {
  pluginId: "thelifeandtimes.urbit-theme.qualification-stable"
  profile: "qualification-stable"
  artifact: "99739cf551a126d85f3c157efaa67bd090844f60ca96c7af700479b3b8126825"
  role: "theme"
  runtimeSource: Qt.resolvedUrl("CollectionRuntime.qml")
  guardSource: Qt.resolvedUrl("collection/shared_admission_guard.py")
  collectionRegistry: Admission.Registry
}
