import QtQuick
import QtQuick.Controls as Controls
import qs.Commons

Controls.ToolTip {
  id: root
  objectName: "omabit-tooltip"
  delay: 450
  padding: Style.space(10)
  margins: Style.space(8)
  implicitWidth: Math.min(Style.space(360), contentItem.implicitWidth + leftPadding + rightPadding)
  contentItem: Label { text: root.text; caption: true; wrapMode: Text.Wrap }
  background: Rectangle {
    color: Design.raised
    radius: Style.space(6)
    border.color: Design.accent
    border.width: 1
  }
}
