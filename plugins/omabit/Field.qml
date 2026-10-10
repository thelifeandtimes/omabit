import QtQuick
import QtQuick.Controls as Controls
import qs.Commons

Controls.TextField {
  id: root
  implicitHeight: Style.space(38)
  color: Design.ink
  placeholderTextColor: Design.muted
  selectionColor: Design.accent
  selectedTextColor: Design.background
  font.family: Style.font.family
  font.pixelSize: Style.font.bodySmall
  leftPadding: Style.space(12)
  rightPadding: leftPadding
  background: Rectangle { radius: Style.space(6); color: Design.background; border.color: root.activeFocus ? Design.accent : Design.line }
}
