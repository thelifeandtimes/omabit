import QtQuick
import qs.Commons
import qs.Ui

TendButton {
  id: root

  property bool flagged: false
  signal toggled(bool flagged)

  width: Style.spacing.controlHeight
  height: Style.spacing.controlHeight
  text: "󰈾"
  tooltipText: flagged ? "Remove flag" : "Flag reminder"
  Accessible.role: Accessible.CheckBox
  Accessible.name: flagged ? "Remove reminder flag" : "Flag reminder"
  Accessible.checked: flagged
  foreground: flagged ? Color.urgent : Qt.rgba(Color.foreground.r, Color.foreground.g, Color.foreground.b, 0.42)
  accent: flagged ? Color.urgent : Color.accent
  bordered: false
  onClicked: root.toggled(!root.flagged)
}
