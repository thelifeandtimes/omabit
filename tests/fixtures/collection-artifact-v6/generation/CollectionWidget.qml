import QtQuick
import "collection" as Admission
Admission.WidgetHost {
  pluginId: "thelifeandtimes.urbit-theme.qualification-stable"
  artifact: "ed20c91e931071c489f72d252bdc615f902778e62b489595dcaf5c8f987d9030"
  role: "theme"
  widgetSource: Qt.resolvedUrl("Panel.qml")
  collectionRegistry: Admission.Registry
}
