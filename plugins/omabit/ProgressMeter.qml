import QtQuick
import qs.Commons

Rectangle {
  id: root
  property real value: 0
  property bool indeterminate: false
  implicitHeight: Style.space(6)
  radius: height / 2
  color: Design.line
  clip: true
  Rectangle {
    id: fill
    height: parent.height
    width: root.indeterminate ? parent.width * 0.25 : parent.width * Math.max(0, Math.min(1, root.value))
    radius: root.radius
    color: Design.accent
    x: 0
    SequentialAnimation on x {
      running: root.visible && root.indeterminate
      loops: Animation.Infinite
      NumberAnimation { from: -fill.width; to: root.width; duration: 1300 }
    }
    Connections { target: root; function onIndeterminateChanged() { if (!root.indeterminate) fill.x=0 } }
  }
}
