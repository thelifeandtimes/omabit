import QtQuick
import qs.Commons

Column {
  property var service
  property bool active: true
  spacing: Style.space(10)
  Label { text: "Application Connections"; font.pixelSize: Style.space(19); font.bold: true }
  Label { objectName: "applications-coming-soon"; width: parent.width; text: "Coming Soon"; secondary: true }
}
