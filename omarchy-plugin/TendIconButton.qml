import QtQuick
import qs.Commons
import qs.Ui

Item {
  id: root

  property string glyph: ""
  property string tooltipText: ""
  property string accessibleName: tooltipText
  property color foreground: Color.foreground
  property color accent: Color.accent
  property bool bordered: true
  property bool selected: false
  property real glyphSize: Style.font.icon
  signal clicked()

  implicitWidth: Style.spacing.controlHeight
  implicitHeight: Style.spacing.controlHeight

  TendButton {
    anchors.fill: parent
    text: ""
    tooltipText: root.tooltipText
    Accessible.name: root.accessibleName
    bordered: root.bordered
    foreground: root.foreground
    accent: root.accent
    checkable: root.selected
    checked: root.selected
    enabled: root.enabled
    onClicked: root.clicked()

    OpticalGlyph {
      anchors.centerIn: parent
      width: root.glyphSize
      height: root.glyphSize
      text: root.glyph
      color: root.foreground
      fontSize: root.glyphSize
    }
  }
}
