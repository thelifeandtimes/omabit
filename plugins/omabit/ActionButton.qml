import QtQuick
import QtQuick.Controls as Controls
import qs.Commons

Controls.Button {
  id: root
  property bool primary: false
  property bool quiet: false
  property bool danger: false
  property string hint: ""
  property bool compact: false
  hoverEnabled: true
  implicitHeight: Style.space(compact ? 28 : 36)
  leftPadding: Style.space(compact ? 9 : 14)
  rightPadding: leftPadding
  topPadding: Style.space(6)
  bottomPadding: topPadding
  contentItem: Label {
    text: root.text
    wrapMode: Text.NoWrap
    horizontalAlignment: Text.AlignHCenter
    verticalAlignment: Text.AlignVCenter
    color: root.primary ? Design.background : root.danger ? Design.danger : Design.ink
    opacity: root.enabled ? 1 : 0.35
    font.bold: root.primary
  }
  background: Rectangle {
    radius: Style.space(6)
    color: root.primary ? Design.accent : root.down ? Design.raised : root.hovered ? Design.raised : root.quiet ? "transparent" : Design.surface
    opacity: root.enabled ? 1 : 0.5
    border.width: root.activeFocus ? 2 : 1
    border.color: root.activeFocus ? Design.accent : root.quiet && !root.hovered ? "transparent" : Design.line
    Behavior on color { ColorAnimation { duration: 100 } }
  }
  Hint { text: root.hint; visible: root.hovered && root.hint !== "" }
  Accessible.name: text
  Accessible.description: hint
}
