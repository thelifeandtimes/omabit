import QtQuick
import QtQuick.Controls as Controls
import QtQuick.Layouts
import qs.Commons
import "Model.js" as Model

Column {
  id: root
  property var service
  property bool active: true
  property string mode: "home"
  property string transport: "ssh"
  property string shipKind: "comet"
  property bool developmentOptions: false
  readonly property var provisioningMethods: [
    {key:"comet",label:"Boot Ephemeral Identity",description:"Generate and boot a ‘comet’ identity. Comets are ephemeral or temporary identities which self-attest their own keys, but cannot rotate them. Great for just getting started and experimentation."},
    {key:"key",label:"Boot Permanent Identity",description:"Boot a fresh instance of your Urbit. Requires cryptographic ownership of an Urbit ID. Once you own your own Urbit ID, you can set networking keys in Urbit’s ‘Azimuth’ Public Key Infrastructure (PKI)."},
    {key:"import",label:"Import existing pier",description:"Already booted your ship and want to import it to Omabit? Bring its existing pier—the folder holding its data and history—under Omabit’s management. Stop the ship, place the pier on the selected runner, then choose its directory below."}
  ]
  readonly property var provisioningMethod: provisioningMethods.filter(function(method){return method.key===root.shipKind})[0] || null
  onShipKindChanged: if (shipKind==="fake" || shipKind==="pier") developmentOptions=true
  property var editing: null
  property var connectionTarget: null
  property string editTransport: "ssh"
  signal showFleet()
  spacing: Style.space(12)
  readonly property var runners: service ? service.connections : []
  readonly property var managedRunners: service ? service.registrations : []
  readonly property var readyRunners: runners.filter(function(c) { return root.service.observing && Model.writable(c, root.service.now) && Model.hasFeature(c, "create-fake") })
  readonly property var provisioningRunner: targetRunner.currentIndex >= 0 ? readyRunners[targetRunner.currentIndex] : null
  readonly property string provisioningSelection: provisioningRunner ? provisioningRunner.runner_id + "/" + provisioningRunner.name : ""
  onProvisioningSelectionChanged: if (provisioningRunner && shipKind === "comet" && !Model.hasFeature(provisioningRunner,"create-comet")) shipKind = "fake"
  onModeChanged: if (service && !service.busy) { service.lastError=""; service.lastResult="" }
  function addShip() { mode="ship" }
  function editRunner(c) {
    editing = {name: c.name, runner_id: c.runner_id}
    var p = Model.runnerPreference(service.runnerPreferences, c)
    displayName.text = p.label || ""; location.text = p.location || ""; browserHost.text = p.browserBase || ""
    mode = "edit"
  }
  function editConnection(entry) {
    if (!entry) entry = service.registrations.filter(function(r) { return r.name === root.editing.name && r.runner_id === root.editing.runner_id })[0]
    if (!entry || !entry.revision) { service.lastError = "Refresh enrollment with the updated collector before editing."; return }
    connectionTarget = JSON.parse(JSON.stringify(entry)); editTransport = entry.transport
    newConnectionName.text = entry.name
    newAddress.text = entry.transport === "ssh" ? entry.ssh.host : entry.socket
    newSSHConfig.text = entry.ssh ? entry.ssh.config_file || "" : ""
    mode = "connection"
  }
  RowLayout {
    width: parent.width; visible: root.mode !== "home"
    ActionButton { compact: true; quiet: true; text: ["runner","edit","connection"].indexOf(root.mode)>=0 ? "← Runners" : "← Setup"; onClicked: root.mode = ["runner","edit","connection"].indexOf(root.mode)>=0 ? "manage" : "home" }
    Label { Layout.fillWidth: true; horizontalAlignment: Text.AlignRight; secondary: true; caption: true; text: "OMABIT "+(root.service ? root.service.applicationVersion : "Loading")+" · SETUP" }
  }
  Column {
    width: parent.width; spacing: Style.space(8); visible: root.mode === "home"
    ActionButton { text: "Set up remote server →"; visible: !!root.service && root.service.machineUse==="laptop"; onClicked: root.mode="remote" }
    Card {
      width: root.width; implicitHeight: localIntro.implicitHeight + Style.space(28)
      Column { id: localIntro; anchors.left: parent.left; anchors.right: parent.right; anchors.top: parent.top; anchors.margins: Style.space(14); spacing: Style.space(9)
        CardTitle { text: root.service && root.service.localSetupStatus && root.service.localSetupStatus.ready ? "This computer is connected" : "Start here: your local Urbit host" }
        Label { width: parent.width; secondary: true; text: root.service && root.service.localSetupStatus && root.service.localSetupStatus.ready ? "Using the existing “" + root.service.localSetupStatus.connection.name + "” runner. Its ships and data are already available in the fleet." : "Set up background services and private desktop access in one guided flow." }
        RowLayout {
          ActionButton { objectName: "setup-add-ship"; text: "Add a ship →"; visible: root.readyRunners.length>0; primary: true; onClicked: root.addShip() }
          ActionButton { text: "Set up this computer →"; visible: root.service && (!root.service.localSetupStatus || !root.service.localSetupStatus.ready); primary: root.readyRunners.length===0; onClicked: { root.mode="local";root.service.checkLocalSetup() } }
          ActionButton { text: "Local setup details"; quiet: true; visible: root.service && root.service.localSetupStatus && root.service.localSetupStatus.ready; onClicked: { root.mode="local";root.service.checkLocalSetup() } }
        }
      }
    }
    Repeater {
      model: [
        {key:"storage",title:"▣  Import a pier & storage locations",detail:"Move an existing pier into management; choose storage or delete stopped piers"},
        {key:"manage", title:"↔  Manage runners", detail:"Add and edit the runner connections enrolled in this collector"},
        {key:"apps", title:"◇  Application Connections", detail:"Coming Soon"}]
      Card {
        required property var modelData
        objectName: "setup-card-"+modelData.key
        width: root.width; implicitHeight: Style.space(70)
        Column { anchors.left: parent.left; anchors.right: parent.right; anchors.verticalCenter: parent.verticalCenter; anchors.margins: Style.space(16); spacing: Style.space(5)
          CardTitle { text: modelData.title }
          Label { text: modelData.detail; caption: true; secondary: true }
        }
        MouseArea { anchors.fill: parent; cursorShape: Qt.PointingHandCursor; onClicked: root.mode = parent.modelData.key }
        ActionButton { anchors.right: parent.right; anchors.verticalCenter: parent.verticalCenter; anchors.rightMargin: Style.space(12); text: "›"; quiet: true; hint: modelData.title; onClicked: root.mode = parent.modelData.key }
      }
    }
  }
  HostSetupPane { width: parent.width; visible: root.mode === "local"; service: root.service; onAddShip: { root.addShip(); root.shipKind = "comet" } }
  DesktopDiagnostics { width:parent.width; service:root.service; visible:root.mode==="home" && !!root.service && !!root.service.bootstrap }
  RemoteRunnerPane { width:parent.width;visible:root.mode==="remote"||root.mode==="runner";service:root.service;onViewRunners:root.mode="manage" }
  StoragePane { width: parent.width; visible: root.mode === "storage"; service: root.service }
  Column {
    objectName: "manage-runners"
    width: parent.width; spacing: Style.space(10); visible: root.mode === "manage"
    Label { text: "Manage runners"; font.pixelSize: Style.space(19); font.bold: true }
    Label { width: parent.width; text: "Runner connections saved by this computer's collector. Connection edits verify the same pinned runner; ship controls live in the fleet and ship views."; secondary: true }
    ActionButton { objectName: "add-runner-connection"; text: "+ Connect another runner"; primary: true; enabled: root.service && !root.service.busy; onClicked: root.mode="runner" }
    Label { visible: root.managedRunners.length===0; text: "No runners are enrolled in this collector."; secondary: true }
    Repeater {
      model: root.managedRunners
      Card {
        id: row
        required property var modelData
        objectName: "managed-runner-"+modelData.name
        readonly property var observation: root.runners.filter(function(c){return c.name===row.modelData.name && c.runner_id===row.modelData.runner_id})[0] || null
        property bool detailsOpen: false
        readonly property bool online: !!row.observation && root.service.observing && Model.available(row.observation,root.service.now)
        readonly property var compatibility: Model.liveHostCompatibility(row.observation,root.service.now)
        width: root.width; implicitHeight: body.implicitHeight + Style.space(24)
        Column {
          id: body
          anchors.left: parent.left; anchors.right: parent.right; anchors.top: parent.top; anchors.margins: Style.space(12); spacing: Style.space(8)
          RowLayout {
            width: parent.width
            CardTitle { Layout.fillWidth: true; text: Model.runnerTitle(row.modelData, root.service.runnerPreferences) }
            StatusBadge { text: row.online ? "Connected" : row.observation && row.observation.status==="identity_mismatch" ? "Needs review" : "Unavailable"; tone: row.online ? "good" : "muted" }
          }
          Label { objectName:"runner-summary-"+row.modelData.name;width:parent.width;secondary:true;text:(row.observation && row.observation.snapshot ? (row.online ? "" : "Last seen · ")+Model.instances(row.observation).length+(Model.instances(row.observation).length===1 ? " ship" : " ships") : "Inventory not loaded")+(row.observation && row.observation.access_role ? " · "+(row.observation.access_role==="operator" ? "Operator access" : "Read-only access") : "")+(Model.runnerPreference(root.service.runnerPreferences,row.modelData).location ? " · "+Model.runnerPreference(root.service.runnerPreferences,row.modelData).location : "") }
          Label { width:parent.width;caption:true;color:Design.warning;visible:text!=="";text:row.observation&&row.observation.status==="identity_mismatch" ? "Connection identity or mode needs review in Technical details." : row.compatibility&&row.compatibility.restart_pending ? "Engine restart pending · see Technical details." : row.compatibility&&!row.compatibility.can_mutate ? "Host checks need attention · see Technical details." : "" }
          Flow {
            width: parent.width; spacing: Style.space(8)
            ActionButton { compact: true; text: "Connection…"; enabled: !root.service.busy; onClicked: root.editConnection(row.modelData) }
            ActionButton { compact: true; text: "Display settings…"; enabled: !root.service.busy; onClicked: root.editRunner(row.modelData) }
            ActionButton { objectName:"runner-details-toggle-"+row.modelData.name;compact:true;quiet:true;text:row.detailsOpen ? "Technical details −" : "Technical details +";onClicked:row.detailsOpen=!row.detailsOpen }
          }
          Column {
          objectName:"runner-details-"+row.modelData.name;width:parent.width;spacing:Style.space(8);visible:row.detailsOpen
          Rectangle { width:parent.width;height:1;color:Design.line }
          Label { width: parent.width; text: "Connection: "+row.modelData.name+" · "+(row.modelData.transport || "unknown")+"\nEndpoint: "+(row.modelData.transport==="ssh" ? row.modelData.ssh.host : row.modelData.socket)+"\nPinned runner: "+row.modelData.runner_id; secondary: true; caption: true }
          Label { width: parent.width; visible: !!row.observation && row.observation.status === "identity_mismatch"; text: "If this same runner was switched to serve-development, explicitly accept its broader mode below. Runner identity remains pinned."; caption: true; color: Design.warning }
          ActionButton { visible: !!row.observation && row.observation.status === "identity_mismatch"; text: "Accept development mode"; enabled: !root.service.busy; onClicked: root.service.acceptDevelopmentMode(row.observation) }
          HostCompatibility { objectName: "runner-host-compatibility-"+row.modelData.name; width: parent.width; report: Model.hostCompatibility(row.observation); envelope: row.observation ? row.observation.host_compatibility || null : null; problem: Model.hostCompatibilityHint(row.observation,root.service.now); now: root.service.now; details: true }
          }
        }
      }
    }
  }
  Column {
    width: parent.width; spacing: Style.space(10); visible: root.mode === "edit"
    Label { text: root.editing ? "Edit “" + root.editing.name + "”" : "Edit runner"; font.pixelSize: Style.space(19); font.bold: true }
    Label { text: "Display name"; caption: true; secondary: true }
    Field { id: displayName; width: parent.width; placeholderText: "Uses the connection name when empty" }
    Label { text: "Location / host label"; caption: true; secondary: true }
    Field { id: location; width: parent.width; placeholderText: "e.g. Home · Berlin, or this laptop" }
    Label { text: "Browser host"; caption: true; secondary: true }
    Field { id: browserHost; width: parent.width; placeholderText: "e.g. http://127.0.0.1 or http://host.tailnet" }
    Label { width: parent.width; text: "The ship's HTTP port is appended to this host. Set this explicitly for SSH runners: their loopback endpoint does not identify this browser's host."; secondary: true; caption: true }
    ActionButton { text: "Save preferences"; primary: true; onClicked: if (root.service.saveRunnerPresentation(root.editing, displayName.text, location.text, browserHost.text)) root.mode = "manage" }
    ActionButton { text: "Edit pinned connection…"; onClicked: root.editConnection() }
  }
  Column {
    width: parent.width; spacing: Style.space(10); visible: root.mode === "connection"
    Label { text: "Edit runner connection"; font.pixelSize: Style.space(19); font.bold: true }
    Label { width: parent.width; text: root.connectionTarget ? "Pinned runner: " + root.connectionTarget.runner_id + "\nConnection revision: " + root.connectionTarget.revision : ""; secondary: true; caption: true }
    Field { id: newConnectionName; width: parent.width; placeholderText: "Connection name" }
    RowLayout {
      ActionButton { text: "SSH"; primary: root.editTransport === "ssh"; onClicked: root.editTransport = "ssh" }
      ActionButton { text: "Local socket"; primary: root.editTransport === "unix"; onClicked: root.editTransport = "unix" }
    }
    Field { id: newAddress; width: parent.width; placeholderText: root.editTransport === "ssh" ? "SSH alias or host" : "Absolute socket path" }
    Field { id: newSSHConfig; width: parent.width; visible: root.editTransport === "ssh"; placeholderText: "Optional SSH config path" }
    Label { width: parent.width; text: "The new endpoint must verify as this same runner. Concurrent edits are rejected; existing enrollment is kept if verification fails."; secondary: true; caption: true }
    ActionButton { text: "Verify & save connection"; primary: true; enabled: root.service && !root.service.busy; onClicked: if (root.service.updateRunner(root.connectionTarget,newConnectionName.text.trim(),root.editTransport,newAddress.text.trim(),newSSHConfig.text.trim())) root.mode = "manage" }
  }
  Column {
    width: parent.width; spacing: Style.space(10); visible: root.mode === "ship"
    Label { text: "Add a ship"; font.pixelSize: Style.space(19); font.bold: true }
    Label { text: "1 / Choose a runner"; secondary: true; caption: true }
    SelectField { id: targetRunner; width: parent.width; model: root.readyRunners.map(function(c){return Model.runnerTitle(c,root.service.runnerPreferences)}) }
    Label { visible: root.readyRunners.length === 0; text: "Connect an available operator runner first."; color: Design.warning }
    Label { text: "2 / Provisioning method"; secondary: true; caption: true }
    Flow {
      width: parent.width; spacing: Style.space(6)
      Repeater {
        model: root.provisioningMethods
        ActionButton { required property var modelData; objectName: "add-ship-method-"+modelData.key; text: modelData.label; primary: root.shipKind === modelData.key; compact: true; onClicked: root.shipKind = modelData.key }
      }
    }
    Card {
      objectName: "provisioning-explainer"
      width: parent.width; visible: !!root.provisioningMethod
      implicitHeight: methodExplanation.implicitHeight+Style.space(24)
      Label {
        id: methodExplanation; objectName: "provisioning-explanation"
        anchors.left: parent.left; anchors.right: parent.right; anchors.top: parent.top; anchors.margins: Style.space(12)
        text: root.provisioningMethod ? root.provisioningMethod.description : ""
        secondary: true
      }
    }
    ActionButton { text: root.developmentOptions ? "Development options −" : "Development options +"; quiet: true; compact: true; onClicked: root.developmentOptions=!root.developmentOptions }
    Flow {
      width: parent.width; spacing: Style.space(6); visible: root.developmentOptions
      ActionButton { text: "Development fake"; primary: root.shipKind==="fake"; compact: true; onClicked: root.shipKind="fake" }
      ActionButton { text: "Qualified development import"; primary: root.shipKind==="pier"; compact: true; onClicked: root.shipKind="pier" }
    }
    KeyBootPane { width: parent.width; visible: root.shipKind === "key"; service: root.service; connection: root.provisioningRunner }
    StoragePane { objectName: "provisioning-pier-import"; width: parent.width; visible: root.shipKind === "import"; provisioning: true; service: root.service; preferredConnection: root.provisioningRunner }
    ImportPane { width: parent.width; visible: root.shipKind === "pier"; service: root.service; connection: targetRunner.currentIndex >= 0 ? root.readyRunners[targetRunner.currentIndex] : null }
    RegistrationPane { width: parent.width; visible: root.shipKind === "fake" || root.shipKind === "comet"; service: root.service; connection: root.provisioningRunner; kind: root.shipKind; onShowFleet: root.showFleet() }
  }
  ApplicationsPane { width: parent.width; visible: root.mode === "apps"; active: visible && root.active; service: root.service }
  Column {
    width: parent.width; spacing: Style.space(10); visible: root.mode === "gaps"
    Label { text: "Secure +code access"; font.pixelSize: Style.space(19); font.bold: true }
    Label { width: parent.width; text: "Code copying requires an operator grant and a runner advertising login-code. The desktop helper hands it to a sensitive clipboard provider for up to 30 seconds."; secondary: true }
    Label { width: parent.width; text: "The code is not displayed or stored by the panel. Clipboard-history protection depends on the manager honoring the sensitive-content hint."; secondary: true; caption: true }
  }
  Label { width: parent.width; visible: root.service && root.mode!=="home" && root.mode!=="ship" && root.mode!=="storage" && !!root.service.lastError; text: root.service ? root.service.lastError : ""; color: Design.danger; caption: true }
  Label { width: parent.width; visible: root.service && root.mode!=="home" && root.mode!=="ship" && !!root.service.lastResult; text: root.service ? root.service.lastResult : ""; color: Design.success; caption: true }
}
