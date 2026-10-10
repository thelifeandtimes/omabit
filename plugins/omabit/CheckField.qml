import QtQuick
import QtQuick.Controls as Controls
import qs.Commons

Controls.CheckBox {
  id: control
  spacing: Style.space(10)
  padding: Style.space(4)
  implicitHeight: Math.max(indicator.implicitHeight, contentItem.implicitHeight) + topPadding + bottomPadding
  implicitWidth: contentItem.implicitWidth + leftPadding + rightPadding
  indicator: Rectangle {
    implicitWidth: Style.space(19); implicitHeight: Style.space(19)
    x: control.leftPadding
    y: control.topPadding + (control.availableHeight - height) / 2
    radius: Style.space(3)
    color: control.checked ? Design.accent : Design.surface
    border.width: control.visualFocus ? 2 : 1
    border.color: control.visualFocus || control.hovered ? Design.accent : Design.line
    opacity: control.enabled ? 1 : 0.5
    Label { anchors.centerIn: parent; text: "✓"; visible: control.checked; color: Design.surface; font.bold: true }
  }
  contentItem: Label {
    text: control.text
    leftPadding: control.indicator.width + control.spacing
    verticalAlignment: Text.AlignVCenter
    opacity: control.enabled ? 1 : 0.55
  }
}
