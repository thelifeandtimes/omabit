import QtQuick
import QtQuick.Controls as QQC
import qs.Commons
import qs.Ui

BorderSurface {
  id: root

  property alias text: editor.text
  property alias placeholderText: editor.placeholderText
  property alias wrapMode: editor.wrapMode
  property color foreground: Color.foreground
  property color accent: Color.accent

  radius: Style.cornerRadius
  color: Style.controlFill(editor.activeFocus, hover.hovered, foreground, accent)
  borderSpec: Border.controlSpec(editor.activeFocus ? "focus" : (hover.hovered ? "hover-cursor" : "normal"), foreground, accent)

  QQC.TextArea {
    id: editor
    anchors.fill: parent
    anchors.leftMargin: root.borderLeft
    anchors.rightMargin: root.borderRight
    anchors.topMargin: root.borderTop
    anchors.bottomMargin: root.borderBottom
    leftPadding: Style.spacing.controlPaddingX
    rightPadding: Style.spacing.controlPaddingX
    topPadding: Style.spacing.controlPaddingY
    bottomPadding: Style.spacing.controlPaddingY
    color: root.foreground
    placeholderTextColor: Qt.darker(root.foreground, 1.5)
    selectionColor: root.accent
    selectedTextColor: Style.selectedStateColor(root.foreground, root.accent)
    font.family: Style.font.family
    font.pixelSize: Style.font.body
    background: Item {}
  }

  HoverHandler { id: hover }
}
