import QtQuick
import QtQuick.Controls as Controls
import qs.Commons

Rectangle {
  id: root
  property string text: "Unknown"
  property string tone: "muted"
  property string hint: ""
  property bool dotOnly: false
  property bool flat: false
  readonly property color tint: Design.tone(tone)
  implicitWidth: dotOnly ? Style.space(10) : row.implicitWidth + (flat ? 0 : Style.space(16))
  implicitHeight: dotOnly ? Style.space(10) : flat ? row.implicitHeight : Style.space(24)
  radius: height / 2
  color: dotOnly ? tint : flat ? "transparent" : Design.fade(tint, 0.1)
  Row {
    id: row
    visible: !root.dotOnly
    anchors.centerIn: parent
    spacing: Style.space(6)
    Rectangle { width: Style.space(6); height: width; radius: width / 2; color: root.tint; anchors.verticalCenter: parent.verticalCenter }
    Label { text: root.text; color: root.tint; caption: true; anchors.verticalCenter: parent.verticalCenter }
  }
  HoverHandler { id: hover }
  Hint { text: root.hint; visible: hover.hovered && root.hint !== "" }
  Accessible.name: text
  Accessible.description: hint
}
