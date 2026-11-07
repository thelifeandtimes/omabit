import QtQuick
import QtQuick.Layouts
import qs.Commons
import "Model.js" as Model
import "ShipRoutes.js" as Routes

Column {
  id:root
  property var service
  property bool active:false
  property string selectedKey:""
  property var draft:null
  property string editingID:""
  property string editingSource:"manual"
  property string formMessage:""
  property string accessMessage:""
  onSelectedKeyChanged:{accessMessage="";accessURL.text=""}
  readonly property alias controller:controller
  readonly property var ships:service ? Model.fleet(service.connections) : []
  readonly property var selected:ships.filter(function(row){return row.key===root.selectedKey})[0]||null
  readonly property var connection:selected ? selected.connection : null
  readonly property var instance:selected ? selected.instance : null
  readonly property bool ready:!!connection && !!instance && Model.available(connection,service.now) && connection.access_role==="operator" && Model.hasFeature(connection,"ship-routes-v1") && instance.observed_state==="running" && !instance.active_operation_id && !instance.recovery_required && !instance.identity_pending && !!instance.resources
  readonly property bool canEdit:controller.available && !controller.busy && !!controller.value && !!draft
  readonly property bool dirty:!!draft && !!controller.value && JSON.stringify(Routes.normalized(draft))!==JSON.stringify(controller.value.record.settings)
  readonly property var automatic:controller.value ? controller.value.registry.routes.filter(function(r){return r.source==="managed" || r.source==="tailscale"}).concat((draft ? draft.disabled_automatic : []).filter(function(id){return !controller.value.registry.routes.some(function(r){return r.id===id})}).map(function(id){return {id:id,origin:"",network:"tailnet"}})) : []
  readonly property string unavailable:!selected || !connection || !instance ? "Choose a ship to review its URLs." : !Model.hasFeature(connection,"ship-routes-v1") ? "This runner needs an Omabit update to manage ship URLs." : connection.access_role!=="operator" ? "An operator connection is required to manage ship URLs." : instance.observed_state!=="running" ? "Start this ship to verify its identity and current addresses." : "The ship or runner is unavailable. Refresh the fleet to check its status."
  spacing:Style.space(12)
  function load(){draft=controller.value ? Routes.copy(controller.value.record.settings) : null;editingID="";editingSource="manual";formMessage="";urlField.text=""}
  function automaticEnabled(id){return !!draft && draft.disabled_automatic.indexOf(id)<0}
  function setAutomatic(id,enabled){if(!canEdit)return;var d=Routes.copy(draft);d.disabled_automatic=d.disabled_automatic.filter(function(v){return v!==id});if(!enabled)d.disabled_automatic.push(id);draft=Routes.normalized(d)}
  function setManual(id,enabled){if(!canEdit)return;var d=Routes.copy(draft);d.routes.forEach(function(r){if(r.id===id)r.enabled=enabled});draft=d}
  function remove(id){if(!canEdit)return;var d=Routes.copy(draft);d.routes=d.routes.filter(function(r){return r.id!==id});draft=d;if(editingID===id)cancelEdit()}
  function cancelEdit(){editingID="";editingSource="manual";urlField.text="";formMessage=""}
  function edit(r){editingID=r.id;editingSource=r.source;urlField.text=r.origin;network.currentIndex=["public","tailnet","lan","local"].indexOf(r.network);scopeField.text=r.scope;formMessage=""}
  function add(){
    if(!canEdit)return false
    var kind=["public","tailnet","lan","local"][network.currentIndex],url=urlField.text.trim().replace(/\/$/,""),scope=kind==="local" ? connection.runner_id : kind==="public" ? "" : scopeField.text.trim()
    formMessage=Routes.formError(url,kind,scope);if(formMessage)return false
    if(draft.routes.some(function(r){return r.origin===url && r.id!==root.editingID})){formMessage="This URL is already registered.";return false}
    var d=Routes.copy(draft),id=editingID
    if(!id){var n=1;while(d.routes.some(function(r){return r.id==="route-"+n}))n++;id="route-"+n}
    var prior=d.routes.filter(function(r){return r.id===id})[0]
    if(!prior && d.routes.length>=29){formMessage="This ship has reached its configured URL limit.";return false}
    d.routes=d.routes.filter(function(r){return r.id!==id});d.routes.push({id:id,origin:url,network:kind,scope:scope,source:editingSource,enabled:prior ? prior.enabled : true});draft=Routes.normalized(d);cancelEdit();return true
  }
  function accessHelp(action){
    if(!canEdit || !service || !service.agentHelp || service.agentHelp.busy)return false
    var origin=accessURL.text.trim().replace(/\/$/,""),provider=["tailscale-serve","tailscale-funnel","cloudflare","custom-domain"][accessProvider.currentIndex]
    accessMessage=""
    if(origin && (!Routes.origin(origin) || origin.indexOf("https://")!==0)){accessMessage="Enter an HTTPS URL without a path or credentials.";return false}
    if(action==="disable" && !origin){accessMessage="Enter the exact URL you want help taking offline.";return false}
    if(origin && provider.indexOf("tailscale-")===0 && !/^https:\/\/[^/:]+\.ts\.net(?::[0-9]+)?$/.test(origin)){accessMessage="Tailscale uses the node's actual ts.net URL.";return false}
    var v=controller.value
    return service.requestAgentHelp("ship-access",{version:1,runner_name:connection.name,binding:Routes.copy(v.record.binding),selection:Routes.copy(v.selection),registry_revision:v.record.revision,provider:provider,action:action,origin:origin})
  }
  ShipRoutes {
    id:controller;service:root.service;active:root.active;ready:root.ready
    runnerName:root.connection ? root.connection.name : "";runnerID:root.connection ? root.connection.runner_id : "";mode:root.connection ? root.connection.mode : "";instanceID:root.instance ? root.instance.id : ""
    runtimeBinding:root.instance ? JSON.stringify([root.instance.config_revision,root.instance.resources ? root.instance.resources.container_id : ""]) : ""
    onValueChanged:root.load()
  }
  Label {text:"Ship URLs";font.pixelSize:Style.space(19);font.bold:true}
  Label {width:parent.width;secondary:true;text:"Keep the addresses for each ship together. These choices are shared with other desktops connected to its runner."}
  Card {
    width:parent.width;implicitHeight:selectionContent.implicitHeight+Style.space(32)
    Column {
      id:selectionContent;anchors.left:parent.left;anchors.right:parent.right;anchors.top:parent.top;anchors.margins:Style.space(16);spacing:Style.space(10)
      Label {text:"Ship";caption:true;secondary:true}
      RowLayout {
        width:parent.width
        SelectField {objectName:"ship-routes-select";Layout.fillWidth:true;Layout.minimumWidth:0;model:["Choose a ship"].concat(root.ships.map(function(row){return (row.instance.ship||row.instance.id)+" · "+row.connection.name+" · "+row.instance.id}));currentIndex:root.selected ? root.ships.map(function(row){return row.key}).indexOf(root.selectedKey)+1 : 0;onActivated:root.selectedKey=currentIndex>0 ? root.ships[currentIndex-1].key : ""}
        ActionButton {objectName:"ship-routes-refresh";text:"Refresh";enabled:controller.available && !controller.busy;onClicked:controller.refresh()}
      }
      Label {width:parent.width;visible:!root.ready;text:root.unavailable;caption:true;secondary:true}
      Label {width:parent.width;visible:controller.busy;text:controller.saving ? "Saving URLs…" : "Checking this ship's addresses…";caption:true;secondary:true}
      Label {width:parent.width;visible:!!controller.error;text:controller.error;caption:true;color:Design.danger}
      Label {width:parent.width;visible:!!controller.message && !root.dirty;text:controller.message;caption:true;color:Design.success}
      RowLayout {
        width:parent.width;spacing:Style.space(10);visible:!!root.draft
        ActionButton {objectName:"ship-routes-save";text:"Save URLs";primary:true;enabled:root.canEdit && root.dirty;onClicked:controller.save(root.draft)}
        Label {Layout.fillWidth:true;text:root.dirty ? "Unsaved changes" : "";color:Design.warning}
      }
    }
  }
  Card {
    width:parent.width;visible:!!root.draft;implicitHeight:autoContent.implicitHeight+Style.space(32)
    Column {
      id:autoContent;anchors.left:parent.left;anchors.right:parent.right;anchors.top:parent.top;anchors.margins:Style.space(16);spacing:Style.space(12)
      CardTitle {text:"Detected addresses"}
      Label {width:parent.width;secondary:true;text:Routes.networkMessage(controller.value ? controller.value.network : null)}
      Repeater {
        model:root.automatic
        Column {
          required property var modelData
          width:autoContent.width;spacing:Style.space(5)
          RowLayout {width:parent.width
            Label {Layout.fillWidth:true;Layout.minimumWidth:0;text:Routes.automaticLabel(modelData.id);font.bold:true}
            StatusBadge {text:Routes.networkLabel(modelData.network);tone:modelData.network==="public" ? "warn" : "muted"}
            SwitchField {objectName:"ship-route-auto-"+modelData.id;text:"Use";checked:root.automaticEnabled(modelData.id);enabled:root.canEdit;onToggled:root.setAutomatic(modelData.id,checked)}
          }
          Label {width:parent.width;text:modelData.origin||"Not currently detected";secondary:true;caption:true;wrapMode:Text.WrapAnywhere}
        }
      }
      Label {width:parent.width;caption:true;secondary:true;text:"Localhost addresses work on the runner's machine. Turning off a route here does not stop an external tunnel or remove DNS."}
    }
  }
  Card {
    width:parent.width;visible:!!root.draft;implicitHeight:manualContent.implicitHeight+Style.space(32)
    Column {
      id:manualContent;anchors.left:parent.left;anchors.right:parent.right;anchors.top:parent.top;anchors.margins:Style.space(16);spacing:Style.space(12)
      CardTitle {text:"Configured addresses"}
      Label {width:parent.width;secondary:true;text:"Add an existing domain, tunnel or network address. Adding a URL does not create a tunnel or expose the ship."}
      Repeater {
        model:root.draft ? root.draft.routes : []
        Column {
          required property var modelData
          width:manualContent.width;spacing:Style.space(6)
          Label {width:parent.width;text:modelData.origin;font.bold:true;wrapMode:Text.WrapAnywhere}
          Flow {
            width:parent.width;spacing:Style.space(8)
            StatusBadge {text:Routes.networkLabel(modelData.network);tone:modelData.network==="public" ? "warn" : "muted"}
            SwitchField {text:"Use";checked:modelData.enabled;enabled:root.canEdit;onToggled:root.setManual(modelData.id,checked)}
            ActionButton {text:"Edit";compact:true;quiet:true;enabled:root.canEdit;onClicked:root.edit(modelData)}
            ActionButton {text:"Remove";compact:true;quiet:true;enabled:root.canEdit;onClicked:root.remove(modelData.id)}
          }
        }
      }
      Rectangle {width:parent.width;height:1;color:Design.line}
      Label {text:root.editingID ? "Edit URL" : "Add URL";font.bold:true}
      Field {id:urlField;objectName:"ship-route-url";width:parent.width;maximumLength:2048;placeholderText:"https://your-domain.example";enabled:root.canEdit}
      GridLayout {
        width:parent.width;columns:width>Style.space(560) ? 2 : 1;columnSpacing:Style.space(12);rowSpacing:Style.space(8)
        Column {Layout.fillWidth:true;Layout.minimumWidth:0;Layout.preferredWidth:1;spacing:Style.space(5)
          Label {text:"Reachable from";caption:true;secondary:true}
          SelectField {id:network;objectName:"ship-route-network";width:parent.width;model:["Public web","Tailnet","Local network","Runner machine"];enabled:root.canEdit;onActivated:if(currentIndex===1 && controller.value && controller.value.network)scopeField.text=controller.value.network.tailnet_suffix||""}
        }
        Column {Layout.fillWidth:true;Layout.minimumWidth:0;Layout.preferredWidth:1;spacing:Style.space(5);visible:network.currentIndex===1 || network.currentIndex===2
          Label {text:network.currentIndex===1 ? "Tailnet DNS suffix" : "Network profile";caption:true;secondary:true}
          Field {id:scopeField;objectName:"ship-route-scope";width:parent.width;maximumLength:256;placeholderText:network.currentIndex===1 ? "example.ts.net" : "home-network";enabled:root.canEdit}
        }
      }
      Label {width:parent.width;visible:!!root.formMessage;text:root.formMessage;color:Design.danger;caption:true}
      Flow {width:parent.width;spacing:Style.space(8)
        ActionButton {objectName:"ship-route-add";text:root.editingID ? "Update URL" : "Add URL";enabled:root.canEdit && !!urlField.text.trim();onClicked:root.add()}
        ActionButton {text:"Cancel edit";visible:!!root.editingID;quiet:true;enabled:root.canEdit;onClicked:root.cancelEdit()}
      }
    }
  }
  Card {
    width:parent.width;visible:!!root.draft;implicitHeight:accessContent.implicitHeight+Style.space(32)
    Column {
      id:accessContent;anchors.left:parent.left;anchors.right:parent.right;anchors.top:parent.top;anchors.margins:Style.space(16);spacing:Style.space(10)
      CardTitle {text:"Remote access help"}
      Label {width:parent.width;secondary:true;text:"Open your default agent with this ship selected. It can help plan access, check an existing setup or take a URL offline. Your Urbit apps control what content is published."}
      SelectField {id:accessProvider;objectName:"ship-access-provider";width:parent.width;model:["Tailscale Serve · private Tailnet","Tailscale Funnel · public web","Cloudflare Tunnel · your domain","Custom domain · reverse proxy"];enabled:root.canEdit}
      Field {id:accessURL;objectName:"ship-access-origin";width:parent.width;maximumLength:2048;placeholderText:"https://your-domain.example (optional for setup)";enabled:root.canEdit}
      Label {width:parent.width;caption:true;secondary:true;text:accessProvider.currentIndex===0 ? "Serve limits access to your Tailnet. The agent will check the exact node and web listener." : accessProvider.currentIndex===1 ? "Funnel makes the ship accessible on the public web using its ts.net name." : "The agent will review your domain, HTTPS and the connection to this ship before suggesting changes."}
      Flow {width:parent.width;spacing:Style.space(8)
        ActionButton {objectName:"ship-access-help";text:"Agent Help";enabled:root.canEdit && !!root.service && !!root.service.agentHelp && !root.service.agentHelp.busy;onClicked:root.accessHelp("configure")}
        ActionButton {objectName:"ship-access-disable-help";text:"Help take URL offline";quiet:true;enabled:root.canEdit && !!root.service && !!root.service.agentHelp && !root.service.agentHelp.busy;onClicked:root.accessHelp("disable")}
      }
      Label {width:parent.width;caption:true;visible:!!text;color:Design.danger;text:root.accessMessage || (root.service && root.service.agentHelp ? root.service.agentHelp.error : "")}
      Label {width:parent.width;caption:true;secondary:true;visible:!!root.service && !!root.service.agentHelp && root.service.agentHelp.busy;text:"Opening your default agent…"}
      Label {width:parent.width;caption:true;secondary:true;text:"Opening help makes no network changes. Turning off a route above only stops Omabit from selecting it; the tunnel or DNS must be changed separately to remove exposure."}
    }
  }
}
