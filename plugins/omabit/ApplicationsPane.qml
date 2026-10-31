import QtQuick
import QtQuick.Layouts
import qs.Commons
import "Model.js" as Model

Column {
  id: root
  property var service
  property bool active: true
  property string requestedPlugin: ""
  signal showHosted()
  readonly property var connections: service ? service.applicationConnections : null
  readonly property var choices: connections ? Model.applicationShips(service,connections.ships) : []
  readonly property var selectedShip: ship.currentIndex>=0 ? choices[ship.currentIndex] || null : null
  readonly property string selectionKey: selectedShip ? selectedShip.origin+"|"+selectedShip.ship : ""
  readonly property bool available: !!connections && !service.busy && connections.fresh
  property var approvals: ({})
  spacing: Style.space(10)
  function approvalKey(plugin) { return selectionKey+"|"+plugin.target.plugin_id+"|"+plugin.target.profile+"|"+plugin.target.artifact_sha256 }
  function attempt(plugin) {
    if (!selectedShip || !connections) return null
    var matches=connections.attempts.filter(function(a) {return a.origin===root.selectedShip.origin && a.ship===root.selectedShip.ship && a.target.plugin_id===plugin.target.plugin_id && a.target.profile===plugin.target.profile && (a.phase!=="closed" || a.receiver_reconciliation)})
    return matches.length ? matches[matches.length-1] : null
  }
  function approve(plugin,value) {var next=Object.assign({},approvals);next[approvalKey(plugin)]=value;approvals=next}
  function resetReview() {approvals=({})}
  onSelectionKeyChanged: resetReview()
  onActiveChanged: {resetReview();if(active && connections)connections.refresh()}
  Component.onCompleted: if(active && connections)connections.refresh()
  Connections { target:root.connections; function onSnapshotRevisionChanged() {root.resetReview()} }

  Label { width:parent.width;text:"Application Connections";font.pixelSize:Style.space(19);font.bold:true }
  Label { width:parent.width;secondary:true;text:"Choose a ship and approve the plugins you want to use. Each gets its own sign-in and connects directly to your ship." }
  RowLayout {
    width:parent.width
    Label { Layout.fillWidth:true;text:"Ship";font.bold:true }
    ActionButton { objectName:"apps-refresh";compact:true;quiet:true;text:root.connections&&root.connections.busy?"Working…":"Refresh";enabled:!!root.connections&&!root.service.busy;onClicked:root.connections.refresh() }
  }
  SelectField {
    id:ship;objectName:"apps-ship";width:parent.width
    model:root.choices.map(function(s){return {label:(s.label||s.name)+" · "+s.ship}})
    textRole:"label";enabled:root.available
  }
  Label { width:parent.width;caption:true;secondary:true;text:root.selectedShip?root.selectedShip.origin:"No accessible ships yet. Start a fleet ship or add a remotely hosted Urbit." }
  Label { objectName:"apps-message";width:parent.width;visible:!!root.connections&&root.connections.message!=="";text:root.connections?root.connections.message:"";color:root.connections&&root.connections.failed?Design.danger:Design.ink }
  Label { width:parent.width;caption:true;secondary:true;visible:!!root.selectedShip;text:"Approved plugins can use your ship identity across apps. Eyre sessions currently have access across desks." }
  Label { width:parent.width;secondary:true;visible:!!root.connections&&root.connections.fresh&&root.connections.plugins.length===0;text:"No installed plugins declare an Omabit connection integration yet. Plugins can also use their own direct sign-in." }
  Repeater {
    model:root.connections?root.connections.plugins:[]
    delegate:Card {
      id:row
      required property var modelData
      objectName:"apps-plugin-row-"+modelData.target.plugin_id
      readonly property var current:root.attempt(modelData)
      readonly property bool connected:!!current&&current.phase==="connected"
      readonly property bool requested:modelData.target.plugin_id===root.requestedPlugin
      readonly property bool readyShip:!!root.selectedShip&&(!!root.selectedShip.managed||root.selectedShip.status==="session_stored")
      readonly property bool approved:connected||root.approvals[root.approvalKey(modelData)]===true
      onApprovedChanged: approval.checked=approved
      readonly property string action:current?(current.next_actions||[])[0]||"":""
      width:root.width;implicitHeight:content.implicitHeight+Style.space(20)
      border.color:requested?Design.accent:Design.line
      Column {
        id:content;anchors.left:parent.left;anchors.right:parent.right;anchors.top:parent.top;anchors.margins:Style.space(10);spacing:Style.space(6)
        RowLayout {
          width:parent.width;spacing:Style.space(8)
          Label {
            objectName:"apps-title-"+row.modelData.target.plugin_id
            Layout.fillWidth:true;Layout.minimumWidth:0
            text:(row.modelData.name||row.modelData.target.plugin_id)+(row.requested?" · requested":"")
            font.bold:true;wrapMode:Text.NoWrap;elide:Text.ElideRight
            HoverHandler { id:titleHover }
            Hint { text:parent.text;visible:titleHover.hovered }
          }
          SwitchField {
            id:approval
            objectName:"apps-consent-"+row.modelData.target.plugin_id
            text:"Approved";checked:row.approved
            enabled:root.available&&row.modelData.status==="available"&&row.readyShip&&(!row.current||row.connected)
            onClicked:{if(row.connected&&!checked)root.connections.act(row.current,"app-disconnect");else root.approve(row.modelData,checked)}
          }
          ActionButton {
            objectName:row.current?"apps-action-"+row.action:"apps-connect-"+row.modelData.target.plugin_id
            compact:true;primary:!row.current;danger:row.connected
            text:row.connected?"Disconnect":row.action==="app-reconcile"?"Finish disconnect":row.current?"Recover":"Connect"
            enabled:root.available&&(row.current?row.action!=="":row.readyShip&&row.modelData.status==="available"&&row.approved)
            onClicked:{if(row.current)root.connections.act(row.current,row.action);else if(root.connections.connect(row.modelData,root.selectedShip,row.approved))root.resetReview()}
          }
        }
        Label {
          width:parent.width;caption:true;secondary:true
          visible:!!root.selectedShip&&!row.connected&&(!!row.current||!row.readyShip||row.modelData.status!=="available")
          text:row.modelData.status!=="available"?"Plugin connection helper needs an update.":!row.readyShip?"Sign in once in Remotely Hosted Urbits.":row.action==="app-reconcile"?"Session closed; local sign-in cleanup remains.":"Connection needs recovery before another sign-in."
        }
      }
    }
  }
}
