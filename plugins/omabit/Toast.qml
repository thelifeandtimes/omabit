import QtQuick
import qs.Commons

Rectangle {
  id: root
  property var service
  visible: !!service && service.toastText!==""
  z: 100
  width: Math.min(implicitWidth, parent ? parent.width-Style.space(24) : implicitWidth)
  implicitWidth: message.implicitWidth+Style.space(28)
  implicitHeight: message.implicitHeight+Style.space(20)
  radius: Style.space(7)
  color: Design.raised
  border.color: Design.accent
  Label { id: message; anchors.fill: parent; anchors.margins: Style.space(10); text: root.service ? root.service.toastText : ""; caption: true; horizontalAlignment: Text.AlignHCenter }
}
