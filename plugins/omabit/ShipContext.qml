import QtQuick
import qs.Commons

Flow {
  id: root
  property string host: ""
  property string instanceName: ""
  property string status: ""
  property string annotation: ""
  property string tone: "muted"
  property string hint: ""
  readonly property real naturalWidth: pathLabel.implicitWidth + spacing + badge.implicitWidth
  spacing: Style.space(7)
  Label { id: pathLabel; width: Math.min(implicitWidth,root.width); text: root.host + " / " + root.instanceName + " /"; caption: true; secondary: true }
  StatusBadge { id: badge; objectName: "ship-context-status"; text: root.status.toLowerCase()+(root.annotation ? " ("+root.annotation+")" : ""); tone: root.tone; hint: root.hint; flat: true }
}
