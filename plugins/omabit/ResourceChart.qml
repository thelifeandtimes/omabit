import QtQuick
import qs.Commons
import "Model.js" as Model

Item {
  id: root
  property var samples: []
  property string metric: "memory"
  property string emptyText: "Collecting live samples…"
  readonly property var values: samples.filter(function(p) { return p !== null })
  readonly property var range: Model.chartRange(samples,metric)
  readonly property real maximum: range.maximum
  implicitHeight: Style.space(190)
  onSamplesChanged: plot.requestPaint()
  onMaximumChanged: plot.requestPaint()
  Repeater {
    model: 3
    Label {
      required property int index
      objectName: "chart-y-" + index
      width: plot.x-Style.space(8); horizontalAlignment: Text.AlignRight
      y: plot.y+index*plot.height/2-height/2
      text: Model.chartValue(root.maximum*(1-index/2),root.metric); caption: true; secondary: true
    }
  }
  Canvas {
    id: plot
    anchors.fill: parent; anchors.leftMargin: Style.space(76); anchors.topMargin: Style.space(10); anchors.bottomMargin: Style.space(43)
    onWidthChanged: requestPaint()
    onHeightChanged: requestPaint()
    Connections { target: Design; function onAccentChanged() { plot.requestPaint() } function onLineChanged() { plot.requestPaint() } }
    onPaint: {
      var ctx = getContext("2d"); ctx.reset()
      ctx.lineWidth = 1; ctx.strokeStyle = Design.line
      for (var j = 0; j < 3; j++) { var y = j * height / 2; ctx.beginPath(); ctx.moveTo(0,y); ctx.lineTo(width,y); ctx.stroke() }
      if (root.values.length < 2) return
      var points = root.samples, count = points.length
      ctx.lineWidth = 2; ctx.strokeStyle = Design.accent
      var segment = []
      function draw() {
        if (!segment.length) return
        ctx.beginPath(); ctx.moveTo(segment[0].x, height)
        segment.forEach(function(p) { ctx.lineTo(p.x,p.y) })
        ctx.lineTo(segment[segment.length-1].x,height); ctx.closePath()
        var gradient = ctx.createLinearGradient(0,0,0,height)
        gradient.addColorStop(0, Design.fade(Design.accent,0.25)); gradient.addColorStop(1, Design.fade(Design.accent,0.015))
        ctx.fillStyle = gradient; ctx.fill()
        ctx.beginPath(); ctx.moveTo(segment[0].x,segment[0].y)
        segment.forEach(function(p) { ctx.lineTo(p.x,p.y) }); ctx.stroke(); segment = []
      }
      var previous = null
      for (var i = 0; i < count; i++) {
        var p = points[i]
        if (!p) { draw(); previous = null; continue }
        if (previous && p.time - previous.time > 30000) draw()
        segment.push({x:(p.time-root.range.start)*width/(root.range.end-root.range.start), y:height * (1-Math.min(1,p.value/root.maximum))}); previous = p
      }
      draw()
    }
  }
  Label { anchors.centerIn: plot; width: plot.width; horizontalAlignment: Text.AlignHCenter; visible: root.values.length < 2; text: root.emptyText; secondary: true; caption: true }
  Repeater {
    model: 3
    Label {
      required property int index
      objectName: "chart-x-" + index
      width: plot.width/3; x: plot.x+index*plot.width/3; y: plot.y+plot.height+Style.space(5)
      horizontalAlignment: index === 0 ? Text.AlignLeft : index === 1 ? Text.AlignHCenter : Text.AlignRight
      text: Model.chartTime(root.range.start === null ? null : root.range.start+(root.range.end-root.range.start)*index/2); secondary: true; caption: true
    }
  }
  Label { anchors.bottom: parent.bottom; anchors.right: parent.right; text: "Local time · this desktop session"; secondary: true; caption: true }
}
