pragma Singleton
import QtQuick
import qs.Commons

QtObject {
  function fade(c, opacity) { return Qt.rgba(c.r, c.g, c.b, opacity) }
  readonly property color background: Color.background
  readonly property color ink: Color.foreground
  readonly property color accent: Color.accent
  readonly property color muted: fade(Color.foreground, 0.62)
  readonly property color line: fade(Color.foreground, 0.12)
  readonly property color surface: Qt.tint(Color.background, fade(Color.foreground, 0.035))
  readonly property color raised: Qt.tint(Color.background, fade(Color.foreground, 0.065))
  readonly property color success: "#8fbc9b"
  readonly property color warning: "#d9b778"
  readonly property color danger: Color.urgent
  function tone(value) {
    return value === "good" ? success : value === "warn" ? warning : value === "bad" ? danger : value === "accent" ? accent : muted
  }
}
