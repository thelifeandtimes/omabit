import QtQuick
import Quickshell
import Quickshell.Hyprland

// Activate only our own toplevel, not a same-title window from another process.
// Hyprland's address-specific focus also switches to its workspace/monitor.
Item {
  id: root
  required property var targetWindow
  property int attempts: 0
  signal failed()

  function activate() {
    targetWindow.visible = true
    targetWindow.minimized = false
    attempts = 0
    Hyprland.refreshToplevels()
    focusTimer.restart()
  }

  Timer {
    id: focusTimer
    interval: 80
    repeat: true
    onTriggered: {
      if (!root.targetWindow.visible) { stop(); return }
      var matches = Hyprland.toplevels.values.filter(function(window) {
        var info = window.lastIpcObject || {}
        return info.pid === Quickshell.processId && window.title === root.targetWindow.title
          && info.mapped !== false && /^[0-9a-f]+$/.test(window.address) && window.address !== "0"
      })
      if (root.targetWindow.backingWindowVisible && matches.length === 1) {
        // Defer until the menubar popup has released keyboard focus. Never toggle
        // visibility to raise a window: doing that loses its workspace/reviews.
        var address = "address:0x" + matches[0].address
        Hyprland.dispatch(Hyprland.usingLua
          ? "hl.dsp.focus({ window = \"" + address + "\" })"
          : "focuswindow " + address)
        stop()
        return
      }
      if (++root.attempts >= 25) { stop(); root.failed(); return }
      // A newly mapped toplevel may not have its PID metadata yet.
      if (root.attempts % 4 === 0) Hyprland.refreshToplevels()
    }
  }
}
