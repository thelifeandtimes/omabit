import QtQuick
import qs.Commons

Item {
  id: root

  property color color: Color.foreground
  property real strokeWidth: 1.5

  implicitWidth: Style.font.icon
  implicitHeight: Style.font.icon

  onColorChanged: arrow.requestPaint()
  onStrokeWidthChanged: arrow.requestPaint()

  Canvas {
    id: arrow
    anchors.fill: parent
    antialiasing: true

    onWidthChanged: requestPaint()
    onHeightChanged: requestPaint()

    onPaint: {
      var context = getContext("2d")
      context.clearRect(0, 0, width, height)
      context.strokeStyle = root.color
      context.lineWidth = root.strokeWidth
      context.lineCap = "round"
      context.lineJoin = "round"

      var left = width * 0.24
      var right = width * 0.76
      var top = height * 0.24
      var bottom = height * 0.76
      var head = width * 0.46

      context.beginPath()
      context.moveTo(left, bottom)
      context.lineTo(right, top)
      context.moveTo(head, top)
      context.lineTo(right, top)
      context.lineTo(right, height * 0.54)
      context.stroke()
    }
  }
}
