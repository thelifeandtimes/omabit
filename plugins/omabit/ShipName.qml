import QtQuick
import "Model.js" as Model

Label {
  id: root
  property var instance: null
  property bool compact: false
  property bool hintEnabled: true
  text: Model.instanceTitle(instance)
  font.italic: !!instance && instance.mode === "fake"
  wrapMode: compact ? Text.NoWrap : Text.Wrap
  elide: compact ? Text.ElideMiddle : Text.ElideNone
  HoverHandler { id: hover }
  Hint { text: root.text; visible: root.hintEnabled && root.compact && root.truncated && hover.hovered }
}
