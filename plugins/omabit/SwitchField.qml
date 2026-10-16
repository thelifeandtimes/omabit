import QtQuick
import QtQuick.Controls as Controls
import qs.Commons

Controls.Switch {
  id: control
  spacing: Style.space(10)
  padding: Style.space(4)
  implicitHeight: Math.max(indicator.implicitHeight, contentItem.implicitHeight) + topPadding + bottomPadding
  indicator: Rectangle {
    implicitWidth: Style.space(38); implicitHeight: Style.space(22)
    x: control.leftPadding; y: control.topPadding + (control.availableHeight-height)/2
    radius: height/2
    color: control.checked ? Design.accent : Design.surface
    border.color: control.visualFocus || control.hovered ? Design.accent : Design.line
    border.width: control.visualFocus ? 2 : 1
    opacity: control.enabled ? 1 : 0.5
    Rectangle {
      width: Style.space(16); height: width; radius: width/2
      x: control.checked ? parent.width-width-Style.space(3) : Style.space(3)
      anchors.verticalCenter: parent.verticalCenter
      color: control.checked ? Design.background : Design.muted
    }
  }
  contentItem: Label {
    text: control.text
    leftPadding: control.indicator.width+control.spacing
    verticalAlignment: Text.AlignVCenter
    opacity: control.enabled ? 1 : 0.55
  }
}
