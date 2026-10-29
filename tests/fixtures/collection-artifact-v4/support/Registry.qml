pragma Singleton
import QtQml

// One QML engine, one immutable generation. Never discovers another plugin.
QtObject {
  property var host: null
  function claim(value) {
    if (host && host !== value) return false
    host = value
    return true
  }
  function release(value) { if (host === value) host = null }
}
