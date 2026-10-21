import QtQuick
import qs.Commons
import "Model.js" as Model

Rectangle {
  id: root
  property var service
  readonly property string messageText: !service ? "" : service.toastText || (service.actionProgressVisible ? service.settingUp ? "Setting up this computer…" : "Submitting "+Model.operationTitle(service.currentAction)+"…" : "")
  readonly property string tone: service && service.toastText ? service.toastTone || "accent" : "accent"
  visible: messageText!==""
  z: 100
  width: Math.min(Style.space(520), parent ? parent.width-Style.space(24) : Style.space(520))
  implicitHeight: message.implicitHeight+Style.space(24)
  radius: Style.space(7)
  color: Design.raised
  border.color: Design.tone(tone)
  Label { id: message; objectName: "toast-message"; anchors.left:parent.left;anchors.right:parent.right;anchors.top:parent.top;anchors.margins: Style.space(12); text: root.messageText; caption: true; horizontalAlignment: Text.AlignHCenter }
}
