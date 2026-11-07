import QtQuick
import QtQuick.Layouts
import QtQuick.Controls as Controls
import qs.Commons
import "AppPolicy.js" as Policy

FocusScope {
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
  readonly property real mapWidth:stride*3+nodeWidth
  readonly property real mapHeight:graph.rows*rowHeight+Style.space(38)
  readonly property real minimumZoom:0.25
  readonly property real maximumZoom:2
  property real zoom:1
  signal selected(var node)
  implicitHeight:zoomControls.implicitHeight+Style.space(10)+Math.min(Style.space(560),mapHeight)
  function setZoom(value,anchorX,anchorY) {
    var next=Math.max(minimumZoom,Math.min(maximumZoom,value))
    if(!isFinite(next) || next===zoom)return
    var x=anchorX===undefined ? viewport.width/2 : Math.max(0,Math.min(viewport.width,anchorX))
    var y=anchorY===undefined ? viewport.height/2 : Math.max(0,Math.min(viewport.height,anchorY))
    var mapX=(viewport.contentX+x)/zoom,mapY=(viewport.contentY+y)/zoom
    viewport.cancelFlick()
    zoom=next
    viewport.contentX=Math.max(0,Math.min(mapX*zoom-x,Math.max(0,mapWidth*zoom-viewport.width)))
    viewport.contentY=Math.max(0,Math.min(mapY*zoom-y,Math.max(0,mapHeight*zoom-viewport.height)))
  }
  function fitToView() {
    setZoom(Math.min(1,viewport.width/mapWidth,viewport.height/mapHeight))
    viewport.contentX=0;viewport.contentY=0
  }
  Keys.onPressed:function(event) {
    if(!(event.modifiers & Qt.ControlModifier))return
    if(event.key===Qt.Key_Plus || event.key===Qt.Key_Equal)setZoom(zoom*1.25)
    else if(event.key===Qt.Key_Minus)setZoom(zoom/1.25)
    else if(event.key===Qt.Key_0)setZoom(1)
    else return
    event.accepted=true
  }
  RowLayout {
    id:zoomControls
    width:parent.width;spacing:Style.space(6)
    Label { text:"Zoom";caption:true;secondary:true }
    ActionButton { objectName:"app-policy-zoom-out";compact:true;text:"−";hint:"Zoom out · Ctrl+−";Accessible.name:"Zoom out";enabled:root.zoom>root.minimumZoom;onClicked:root.setZoom(root.zoom/1.25) }
    ActionButton { objectName:"app-policy-zoom-reset";compact:true;text:Math.round(root.zoom*100)+"%";hint:"Reset zoom to 100% · Ctrl+0";Accessible.name:"Reset tree zoom to 100 percent";onClicked:root.setZoom(1) }
    ActionButton { objectName:"app-policy-zoom-in";compact:true;text:"+";hint:"Zoom in · Ctrl++";Accessible.name:"Zoom in";enabled:root.zoom<root.maximumZoom;onClicked:root.setZoom(root.zoom*1.25) }
    ActionButton { objectName:"app-policy-zoom-fit";compact:true;text:"Fit";hint:"Fit the tree in view";onClicked:root.fitToView() }
    Item { Layout.fillWidth:true }
  }
  Controls.ScrollView {
    id:scroll
    anchors.top:zoomControls.bottom;anchors.topMargin:Style.space(10)
    anchors.left:parent.left;anchors.right:parent.right;anchors.bottom:parent.bottom
    Flickable {
      id:viewport;objectName:"app-policy-viewport"
      clip:true;interactive:true;boundsBehavior:Flickable.StopAtBounds
      contentWidth:Math.max(width,root.mapWidth*root.zoom)
      contentHeight:Math.max(height,root.mapHeight*root.zoom)
      DragHandler {
        target:null
        enabled:viewport.contentWidth>viewport.width || viewport.contentHeight>viewport.height
        grabPermissions:PointerHandler.CanTakeOverFromAnything
        property real startX:0
        property real startY:0
        onActiveChanged:if(active){viewport.cancelFlick();startX=viewport.contentX;startY=viewport.contentY}
        onActiveTranslationChanged:if(active){
          viewport.contentX=Math.max(0,Math.min(startX-activeTranslation.x,viewport.contentWidth-viewport.width))
          viewport.contentY=Math.max(0,Math.min(startY-activeTranslation.y,viewport.contentHeight-viewport.height))
        }
      }
      Item {
        id:map;objectName:"app-policy-map"
        width:root.mapWidth;height:root.mapHeight
        scale:root.zoom;transformOrigin:Item.TopLeft
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
  }
  MouseArea {
    anchors.fill:scroll
    acceptedButtons:Qt.NoButton
    onWheel:function(wheel) {
      if(!(wheel.modifiers & Qt.ControlModifier)){wheel.accepted=false;return}
      var delta=wheel.angleDelta.y || wheel.pixelDelta.y
      if(!delta){wheel.accepted=false;return}
      var point=mapToItem(viewport,wheel.x,wheel.y)
      root.setZoom(root.zoom*Math.pow(1.25,delta/120),point.x,point.y)
      wheel.accepted=true
    }
  }
  onGraphChanged:paths.requestPaint()
  onWidthChanged:paths.requestPaint()
  onFilteringChanged:paths.requestPaint()
}
