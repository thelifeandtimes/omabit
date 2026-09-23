import QtQuick
import qs.Commons
import qs.Ui

TendButton {
  id: root

  property bool flagged: false
  signal toggled(bool flagged)

  width: Style.spacing.controlHeight
  height: Style.spacing.controlHeight
  text: flagged ? "󰈿" : "󰈾"
  tooltipText: flagged ? "Remove flag" : "Flag reminder"
  Accessible.role: Accessible.CheckBox
  Accessible.name: flagged ? "Remove reminder flag" : "Flag reminder"
  Accessible.checked: flagged
  foreground: flagged ? Color.urgent : Color.foreground
  accent: flagged ? Color.urgent : Color.accent
  bordered: false
  onClicked: root.toggled(!root.flagged)
}
