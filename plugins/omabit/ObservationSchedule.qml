import QtQuick

// Observation demand is separate from visual content and process ownership.
// Multiple bars/panels may share one service; closing one must not idle another.
QtObject {
  id: root
  property bool monitorOpen: false
  property bool workPending: false
  property var panels: []
  readonly property bool foreground: monitorOpen || panels.length > 0
  readonly property bool active: foreground || workPending
  readonly property string interval: active ? "2s" : "30s"
  signal changed(bool refresh)
  onIntervalChanged: changed(active)
  onForegroundChanged: if (foreground && workPending) changed(true)
  function setPanelOpen(panel, opened) {
    var index = panels.indexOf(panel)
    if (opened && index < 0) panels = panels.concat([panel])
    else if (!opened && index >= 0) panels = panels.filter(function(p) { return p !== panel })
  }
}
