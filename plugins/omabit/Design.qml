pragma Singleton
import QtQuick
import qs.Commons as Commons

QtObject {
  function fade(c, opacity) { return Qt.rgba(c.r, c.g, c.b, opacity) }
  readonly property color background: Commons.Color.background
  readonly property color ink: Commons.Color.foreground
  readonly property color accent: Commons.Color.accent
  readonly property color muted: fade(Commons.Color.foreground, 0.62)
  readonly property color line: fade(Commons.Color.foreground, 0.12)
  readonly property color surface: Qt.tint(Commons.Color.background, fade(Commons.Color.foreground, 0.035))
  readonly property color raised: Qt.tint(Commons.Color.background, fade(Commons.Color.foreground, 0.065))
  readonly property color success: "#8fbc9b"
  readonly property color warning: "#d9b778"
  readonly property color danger: Commons.Color.urgent
  function tone(value) {
    return value === "good" ? success : value === "warn" ? warning : value === "bad" ? danger : value === "accent" ? accent : muted
  }
}
