import QtQuick
import QtQuick.Controls as Controls
import qs.Commons
import "AppPolicy.js" as Policy

Item {
  id:root
  property var model:null
  property string gate:"commit"
  property string filter:"all"
  property string query:""
  property string selectedKey:""
  readonly property var graph:Policy.layoutTree(model,gate,filter,query)
  readonly property bool filtering:filter!=="all" || !!query.trim()
  readonly property real nodeWidth:Math.max(Style.space(122),Math.min(Style.space(166),(width-Style.space(96))/4))
  readonly property real stride:nodeWidth+Style.space(32)
  readonly property real rowHeight:Style.space(64)
  readonly property real nodeHeight:Style.space(50)
  signal selected(var node)
  implicitHeight:Math.min(Style.space(560),graph.rows*rowHeight+Style.space(38))
  Controls.ScrollView {
    id:scroll
    anchors.fill:parent;clip:true
    contentWidth:Math.max(availableWidth,root.stride*3+root.nodeWidth)
    contentHeight:root.graph.rows*root.rowHeight+Style.space(38)
    Item {
      id:map
      width:scroll.contentWidth;height:scroll.contentHeight
      Repeater {
        model:["GLOBAL","RUNNERS","SHIPS","DESKS"]
        Label { required property int index;required property string modelData;x:index*root.stride;width:root.nodeWidth;caption:true;secondary:true;horizontalAlignment:Text.AlignHCenter;text:modelData;font.letterSpacing:1 }
      }
      Canvas {
        id:paths
        anchors.fill:parent
        onPaint:{
          var ctx=getContext("2d");ctx.reset()
          root.graph.edges.forEach(function(edge){
            var x1=edge.from.depth*root.stride+root.nodeWidth,y1=Style.space(30)+edge.from.row*root.rowHeight+root.nodeHeight/2
            var x2=edge.to.depth*root.stride,y2=Style.space(30)+edge.to.row*root.rowHeight+root.nodeHeight/2
            ctx.strokeStyle=root.filtering && edge.highlight ? Design.accent : Design.line
            ctx.globalAlpha=root.filtering && !edge.highlight ? 0.35 : 1
            ctx.lineWidth=root.filtering && edge.highlight ? 2 : 1.5
            ctx.beginPath();ctx.moveTo(x1,y1);ctx.bezierCurveTo((x1+x2)/2,y1,(x1+x2)/2,y2,x2,y2);ctx.stroke()
          })
        }
      }
      Repeater {
        model:root.graph.nodes
        ActionButton {
          id:node
          required property var modelData
          objectName:"policy-node-"+modelData.key
          x:modelData.depth*root.stride;y:Style.space(30)+modelData.row*root.rowHeight
          width:root.nodeWidth;height:root.nodeHeight
          readonly property bool current:modelData.key===root.selectedKey
          opacity:!current && !activeFocus && root.filtering && !modelData.onPath ? 0.35 : 1
          hint:modelData.title+" · "+modelData.subtitle+"\n"+(!modelData.ready ? "Permissions unavailable" : modelData.decision.value+" · from "+modelData.decision.source)
          Accessible.name:modelData.title+", "+modelData.scope+", "+(modelData.ready ? modelData.decision.value : "unavailable")
          background:Rectangle {
            radius:Style.space(8)
            color:node.current ? Design.fade(Design.accent,0.16) : root.filtering && node.modelData.match ? Design.fade(Design.accent,0.08) : Design.surface
            border.width:node.current || node.activeFocus ? 2 : 1
            border.color:node.current || node.activeFocus ? Design.accent : root.filtering && node.modelData.onPath ? Design.fade(Design.accent,0.5) : Design.line
          }
          contentItem:Column {
            spacing:Style.space(6)
            Label { width:parent.width;text:node.modelData.title;font.bold:node.current;wrapMode:Text.NoWrap;elide:Text.ElideRight }
            Row {
              spacing:Style.space(5)
              Rectangle { anchors.verticalCenter:parent.verticalCenter;width:5;height:5;radius:3;color:!node.modelData.ready ? Design.warning : node.modelData.decision.value==="allow" ? Design.accent : Design.muted }
              Label { caption:true;secondary:true;text:!node.modelData.ready ? "Unavailable" : root.gate==="create_desk" && node.modelData.scope==="desk" ? "Set on ship" : node.modelData.decision.value==="allow" ? "Allowed" : "Blocked" }
              Label { caption:true;color:Design.accent;text:node.current ? "‹" : "" }
            }
          }
          onClicked:root.selected(modelData)
        }
      }
    }
  }
  onGraphChanged:paths.requestPaint()
  onWidthChanged:paths.requestPaint()
  onFilteringChanged:paths.requestPaint()
}
