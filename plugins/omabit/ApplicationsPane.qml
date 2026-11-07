import QtQuick
import QtQuick.Layouts
import qs.Commons
import "Model.js" as Model
import "ShipRoutes.js" as Routes

Column {
  id: root
  property var service
  property bool active: true
  property string requestedPlugin: ""
  signal showHosted()
  readonly property var connections: service ? service.applicationConnections : null
  readonly property var choices: connections ? Model.applicationShips(service,connections.ships) : []
  readonly property var selectedShip: ship.currentIndex>=0 ? choices[ship.currentIndex] || null : null
  readonly property string selectionKey: Model.applicationShipKey(selectedShip)
  readonly property bool automatic:!!selectedShip && !!selectedShip.managed && !!selectedShip.managed.fleet
  readonly property bool available: !!connections && !service.busy && !connections.busy && connections.fresh
  readonly property var otherPermissions:connections && connections.permissions ? connections.permissions.entries.filter(function(g){return g.allowed && !(root.automatic && connections.review && Routes.equal(g.binding,connections.review.routes.record.binding) && connections.plugins.some(function(p){return Routes.equal(p.target,g.target) && p.status==="available"}))}) : []
  readonly property var otherAttempts:connections ? connections.attempts.filter(function(a){return !connections.plugins.some(function(p){return root.attempt(p)===a})}) : []
  property var approvals: ({})
  spacing: Style.space(10)
  function approvalKey(plugin) { return selectionKey+"|"+plugin.target.plugin_id+"|"+plugin.target.profile+"|"+plugin.target.artifact_sha256 }
  function attempt(plugin) {
    if (!selectedShip || !connections) return null
    var matches=connections.attempts.filter(function(a) {
      var same=root.automatic ? connections.review && a.fleet && Routes.equal(a.fleet.request.binding,connections.review.routes.record.binding) : a.origin===root.selectedShip.origin && a.ship===root.selectedShip.ship
      return same && a.target.plugin_id===plugin.target.plugin_id && a.target.profile===plugin.target.profile && (a.phase!=="closed" || a.receiver_reconciliation)
    })
    return matches.length ? matches[matches.length-1] : null
  }
  function approve(plugin,value) {var next=Object.assign({},approvals);next[approvalKey(plugin)]=value;approvals=next}
  function resetReview() {approvals=({})}
  function select(){if(connections && typeof connections.selectShip==="function")connections.selectShip(selectedShip,active)}
  onSelectionKeyChanged: {resetReview();select()}
  onActiveChanged: {resetReview();select();if(active && connections)connections.refresh()}
  Component.onCompleted: {select();if(active && connections)connections.refresh()}
  Connections { target:root.connections; function onSnapshotRevisionChanged() {root.resetReview()} }

  Label { width:parent.width;text:"Application Connections";font.pixelSize:Style.space(19);font.bold:true }
  Label { width:parent.width;secondary:true;text:"Choose a ship and the plugins that may connect to it. Each gets its own sign-in. Fleet permissions apply across the ship's verified URLs." }
  RowLayout {
    width:parent.width
    Label { Layout.fillWidth:true;text:"Ship";font.bold:true }
    ActionButton { objectName:"apps-refresh";compact:true;quiet:true;text:root.connections&&root.connections.busy?"Working…":"Refresh";enabled:!!root.connections&&!root.service.busy;onClicked:root.connections.refresh() }
  }
  SelectField {
    id:ship;objectName:"apps-ship";width:parent.width
    model:root.choices.map(function(s){return {label:(s.label||s.name)+" · "+s.ship+(s.managed ? " · "+s.managed.runner : "")}})
    textRole:"label";enabled:root.available
  }
  Label { width:parent.width;caption:true;secondary:true;text:root.automatic ? "Omabit chooses a reachable URL for this ship. Registered addresses are in Setup → Ship URLs." : root.selectedShip?root.selectedShip.origin:"No accessible ships yet. Start a fleet ship or add a remotely hosted Urbit." }
  Label {width:parent.width;caption:true;secondary:true;visible:root.automatic && !!connections && connections.operation==="read-review";text:"Verifying this ship and its saved permissions…"}
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
      readonly property var grant:root.automatic && root.connections ? root.connections.grant(modelData) : null
      readonly property bool readyShip:!!root.selectedShip&&(root.automatic ? !!root.connections.review : !!root.selectedShip.managed||root.selectedShip.status==="session_stored")
      readonly property bool approved:root.automatic ? !!grant && grant.allowed : connected||root.approvals[root.approvalKey(modelData)]===true
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
            text:root.automatic ? "Allowed" : "Approved";checked:row.approved
            enabled:root.available&&row.modelData.status==="available"&&row.readyShip&&(root.automatic||!row.current||row.connected)
            onClicked:{if(root.automatic){var choice=checked;checked=row.approved;root.connections.permit(row.modelData,root.selectedShip,choice)}else if(row.connected&&!checked)root.connections.act(row.current,"app-disconnect");else root.approve(row.modelData,checked)}
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
          text:row.modelData.status!=="available"?"Plugin connection helper needs an update.":!row.readyShip?(root.automatic ? "Refresh to verify this ship and its permission." : "Sign in once in Remotely Hosted Urbits."):row.action==="app-reconcile"?"Session closed; local sign-in cleanup remains.":"Connection needs recovery before another sign-in."
        }
        Label {width:parent.width;caption:true;secondary:true;visible:root.automatic;text:row.connected ? "Connected via "+row.current.origin+(row.approved ? "" : ". Future sign-ins are blocked.") : row.approved ? "Allowed for this ship across its verified URLs." : "Allow this plugin to establish owner sessions on this ship.";wrapMode:Text.WrapAnywhere}
      }
    }
  }
  Label {width:parent.width;font.bold:true;visible:root.otherPermissions.length>0;text:"Other saved permissions"}
  Label {width:parent.width;caption:true;secondary:true;visible:root.otherPermissions.length>0;text:"You can revoke permission even when a ship is offline or its plugin was removed. Disconnect existing sessions separately."}
  Repeater {
    model:root.otherPermissions
    delegate:Card {
      required property var modelData
      width:root.width;implicitHeight:savedContent.implicitHeight+Style.space(24)
      RowLayout {
        id:savedContent;anchors.left:parent.left;anchors.right:parent.right;anchors.top:parent.top;anchors.margins:Style.space(12);spacing:Style.space(10)
        Column {Layout.fillWidth:true;Layout.minimumWidth:0;spacing:Style.space(4)
          Label {width:parent.width;text:modelData.binding.ship+" · "+modelData.target.plugin_id;font.bold:true;wrapMode:Text.WrapAnywhere}
          Label {width:parent.width;text:modelData.binding.runner_id+" / "+modelData.binding.target.instance_id+" · "+modelData.target.profile;caption:true;secondary:true;wrapMode:Text.WrapAnywhere}
        }
        ActionButton {objectName:"apps-revoke-"+modelData.binding.target.instance_id+"-"+modelData.target.plugin_id;text:"Revoke";compact:true;enabled:root.available;onClicked:root.connections.revoke(modelData)}
      }
    }
  }
  Label {width:parent.width;font.bold:true;visible:root.otherAttempts.length>0;text:"Other connections and recovery"}
  Repeater {
    model:root.otherAttempts
    delegate:Card {
      required property var modelData
      readonly property string next:(modelData.next_actions||[])[0]||""
      width:root.width;implicitHeight:recoveryContent.implicitHeight+Style.space(24)
      RowLayout {
        id:recoveryContent;anchors.left:parent.left;anchors.right:parent.right;anchors.top:parent.top;anchors.margins:Style.space(12);spacing:Style.space(10)
        Column {Layout.fillWidth:true;Layout.minimumWidth:0;spacing:Style.space(4)
          Label {width:parent.width;text:modelData.ship+" · "+modelData.target.plugin_id;font.bold:true;wrapMode:Text.WrapAnywhere}
          Label {width:parent.width;text:modelData.origin+" · "+modelData.phase;caption:true;secondary:true;wrapMode:Text.WrapAnywhere}
        }
        ActionButton {objectName:"apps-recovery-"+modelData.request_id;text:next==="app-disconnect" ? "Disconnect" : next==="app-reconcile" ? "Finish disconnect" : next==="app-archive" ? "Archive" : "Recover";compact:true;enabled:root.available && !!next;onClicked:root.connections.act(modelData,next)}
      }
    }
  }
}
