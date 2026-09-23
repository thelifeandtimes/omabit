import QtQuick
import QtQuick.Controls as QQC
import qs.Commons
import qs.Ui

QQC.MenuItem {
  id: root

  property color foreground: Color.menu.text
  property color accent: Color.accent

  implicitWidth: Style.space(190)
  implicitHeight: Style.spacing.controlHeight
  leftPadding: Style.spacing.controlPaddingX
  rightPadding: Style.spacing.controlPaddingX
  topPadding: 0
  bottomPadding: 0

  contentItem: Text {
    text: root.text
    color: root.enabled
      ? (root.highlighted ? Style.hoverStateColor(root.foreground, root.accent) : root.foreground)
      : Qt.darker(root.foreground, 1.7)
    font.family: Style.font.family
    font.pixelSize: Style.font.body
    verticalAlignment: Text.AlignVCenter
    elide: Text.ElideRight
  }

  background: Rectangle {
    radius: Style.cornerRadius
    color: root.highlighted ? Style.hoverFillFor(root.foreground, root.accent) : "transparent"
  }
}
