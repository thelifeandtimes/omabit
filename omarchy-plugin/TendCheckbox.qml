import QtQuick
import qs.Commons
import qs.Ui

Item {
  id: root

  property string text: ""
  property bool checked: false
  property color foreground: Color.foreground
  property color accent: Color.accent
  signal clicked()

  implicitWidth: row.implicitWidth
  implicitHeight: Math.max(Style.spacing.controlHeight, row.implicitHeight)
  activeFocusOnTab: true
  opacity: enabled ? 1 : 0.45

  Keys.onReturnPressed: root.clicked()
  Keys.onEnterPressed: root.clicked()
  Keys.onSpacePressed: root.clicked()

  Row {
    id: row
    anchors.verticalCenter: parent.verticalCenter
    spacing: Style.space(7)

    BorderSurface {
      id: box
      width: Style.space(18)
      height: Style.space(18)
      anchors.verticalCenter: parent.verticalCenter
      radius: Math.max(2, Style.cornerRadius / 2)
      color: root.checked
        ? Style.selectedFillFor(root.foreground, root.accent)
        : Style.controlFill(root.activeFocus, hover.hovered, root.foreground, root.accent)
      borderSpec: Border.controlSpec(root.activeFocus ? "focus" : (hover.hovered ? "hover-cursor" : (root.checked ? "selected" : "normal")), root.foreground, root.accent)

      Text {
        anchors.centerIn: parent
        visible: root.checked
        text: "✓"
        textFormat: Text.PlainText
        color: Style.selectedStateColor(root.foreground, root.accent)
        font.family: Style.font.family
        font.pixelSize: Math.round(box.height * 0.85)
        font.bold: true
      }
    }

    Text {
      visible: root.text !== ""
      anchors.verticalCenter: parent.verticalCenter
      text: root.text
      textFormat: Text.PlainText
      color: root.foreground
      font.family: Style.font.family
      font.pixelSize: Style.font.bodySmall
    }
  }

  HoverHandler { id: hover }

  MouseArea {
    anchors.fill: parent
    enabled: root.enabled
    cursorShape: Qt.PointingHandCursor
    onClicked: {
      root.forceActiveFocus()
      root.clicked()
    }
  }
}
