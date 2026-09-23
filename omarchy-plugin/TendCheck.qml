import QtQuick
import qs.Commons
import qs.Ui

Item {
  id: root

  property string text: ""
  property bool checked: false
  signal clicked()

  implicitWidth: row.implicitWidth
  implicitHeight: Math.max(row.implicitHeight, Style.spacing.controlHeight)
  opacity: enabled ? 1 : 0.45

  Row {
    id: row
    anchors.verticalCenter: parent.verticalCenter
    spacing: Style.space(7)

    ToggleSwitch {
      id: switchControl
      anchors.verticalCenter: parent.verticalCenter
      checked: root.checked
      interactive: false
      cursorRing: false
      trackHeight: Style.space(18)
    }

    Text {
      anchors.verticalCenter: parent.verticalCenter
      text: root.text
      textFormat: Text.PlainText
      color: Color.foreground
      font.family: Style.font.family
      font.pixelSize: Style.font.bodySmall
    }
  }

  MouseArea {
    anchors.fill: parent
    enabled: root.enabled
    cursorShape: Qt.PointingHandCursor
    onClicked: {
      root.checked = !root.checked
      root.clicked()
    }
  }
}
