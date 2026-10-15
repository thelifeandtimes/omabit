import QtQuick
import QtQuick.Dialogs
import qs.Commons
import "Model.js" as Model

Column {
  id: root
  objectName: "owned-key-form"
  property var service
  property var connection
  property string keyFile: ""
  property bool advanced: false
  property string submitError: ""
  readonly property string ship: Model.fakeIdentity(identity.text)
  readonly property string instanceName: ship.slice(1)
  readonly property string destination: Model.ownedPierPath(ship) ? Model.storagePath(storageLocation.rootID,Model.ownedPierPath(ship)) : ""
  readonly property string selectedPill: bootstrap.source
  readonly property var currentConnection: service && connection ? service.connections.filter(function(c){return c.name===root.connection.name && c.runner_id===root.connection.runner_id})[0] || null : null
  readonly property var registeredInstance: Model.instances(currentConnection).filter(function(i){return i.id===root.instanceName && i.ship===root.ship && i.mode!=="fake"})[0] || null
  readonly property bool registered: !!registeredInstance
  readonly property var outcome: {
    var o=service ? service.registrationOutcome : null
    return o && o.kind==="owned" && connection && o.runner===connection.runner_id && o.connection===connection.name && o.instance===instanceName ? o : null
  }
  readonly property bool submitted: registered || !!outcome && outcome.status!=="failed"
  readonly property string problem: registered ? "" : !destination ? "Enter a galaxy, star, planet or moon identity." : !root.keyFile ? "Choose the private key file." : !Model.optionalPort(httpPort.text) || !Model.optionalPort(amesPort.text) ? "Advanced ports must be blank or 1024–65535." : bootstrap.problem
  spacing: Style.space(12)
  Label { text: "Boot an owned ship from a key"; font.pixelSize: Style.space(21); font.bold: true }
  Label { width: parent.width; text: "Choose your Urbit ID and private key file. Omabit names the instance, selects its pier directory and allocates available ports. Registration leaves it stopped; Start performs the first boot."; secondary: true }
  BootstrapSelector { id: bootstrap; width: parent.width; service: root.service; visible: !root.registered; enabled: !root.submitted && !!root.service && !root.service.busy; onEdited: root.submitError="" }
  Label { width: parent.width; visible: root.registered && !!root.registeredInstance.bootstrap; text: visible ? "Boot pill: "+(root.registeredInstance.bootstrap.kind==="bare" ? "bare bootstrap" : "Recommended Apps") : ""; caption: true; secondary: true }
  Label { text: "Expected Urbit ID"; caption: true; secondary: true }
  Field { id: identity; objectName: "key-identity"; width: parent.width; placeholderText: "~sampel-palnet"; enabled: !root.submitted && !root.service.busy }
  Label { text: "Private key file location"; caption: true; secondary: true }
  Field { objectName: "key-file-location"; width: parent.width; text: root.keyFile; placeholderText: "/path/to/ship.key"; enabled: !root.submitted && !root.service.busy; onTextEdited: root.keyFile=text }
  ActionButton { objectName: "key-file-browse"; text: "Browse…"; enabled: !root.submitted && !root.service.busy; onClicked: picker.open() }
  FileDialog {
    id: picker
    objectName: "key-file-picker"
    title: "Select private Urbit boot key"
    fileMode: FileDialog.OpenFile
    // Native GTK/GVFS dialogs can abort the shared Quickshell process.
    options: FileDialog.DontUseNativeDialog
    onAccepted: root.keyFile=Model.localPath(selectedFile)
  }
  StorageLocationSelector { id: storageLocation; width: parent.width; connection: root.connection; visible: !root.registered; enabled: !root.submitted && !root.service.busy }
  Label { width: parent.width; visible: root.destination!==""; text: "Instance: "+root.instanceName+"\nPier: "+Model.absolutePierPath(root.connection,root.destination); secondary: true; caption: true }
  ActionButton { objectName: "key-advanced-toggle"; text: root.advanced ? "Advanced configuration −" : "Advanced configuration +"; quiet: true; onClicked: root.advanced=!root.advanced }
  Column {
    objectName: "key-advanced"; width: parent.width; spacing: Style.space(10); visible: root.advanced
    Label { text: "Ames UDP port"; caption: true; secondary: true }
    Field { id: amesPort; objectName: "key-ames-port"; width: parent.width; placeholderText: "Automatic"; enabled: !root.submitted && !root.service.busy; validator: IntValidator {bottom:1024;top:65535} }
    Label { text: "HTTP port"; caption: true; secondary: true }
    Field { id: httpPort; objectName: "key-http-port"; width: parent.width; placeholderText: "Automatic"; enabled: !root.submitted && !root.service.busy; validator: IntValidator {bottom:1024;top:65535} }
    Label { width: parent.width; text: "HTTP is loopback-only; Ames UDP is published on the selected host port."; secondary: true; caption: true }
  }
  Label { width: parent.width; visible: root.problem!=="" && identity.text!==""; text: root.problem; secondary: true; caption: true }
  Flow {
    width: parent.width; spacing: Style.space(8)
    ActionButton { objectName: "key-register"; text: root.registered ? "Registered" : root.submitted ? "Registration submitted" : root.service.busy ? "Preparing boot files…" : "Register ship"; primary: !root.registered; enabled: root.connection && root.connection.access_role==="operator" && Model.hasFeature(root.connection,"key-boot-options") && !root.submitted && !root.service.busy && root.problem===""; onClicked: {root.submitError="";if(!root.service.createFromKey(root.connection,root.ship,root.keyFile,httpPort.text,amesPort.text,root.selectedPill,bootstrap.kind,storageLocation.rootID))root.submitError=root.service.lastError} }
    ActionButton {
      objectName: "key-start"
      text: "▶ Start"
      primary: root.registered
      hint: root.registered ? "Start " + root.ship : "Register the ship to enable Start."
      enabled: !!root.service && root.service.observing && !root.service.busy && Model.canOperate(root.currentConnection,root.registeredInstance,"start",root.service.now)
      onClicked: {
        if (!enabled) return
        if (root.service.operate(root.currentConnection,root.registeredInstance,"start"))
          root.service.openMonitor(root.currentConnection,root.registeredInstance,"overview")
      }
    }
  }
  Label { objectName: "key-registration-feedback"; width: parent.width; visible: root.registered || !!root.outcome || root.submitError!==""; text: root.outcome ? root.outcome.message : root.registered ? "Registered “"+root.instanceName+"”. Choose Start when ready." : root.submitError; color: root.registered || root.outcome && root.outcome.status==="succeeded" ? Design.success : root.outcome && root.outcome.status==="submitting" ? Design.accent : Design.danger }
  ActionButton { objectName: "key-register-another"; text: "Register another ship"; visible: root.registered || !!root.outcome && root.outcome.status==="succeeded"; enabled: !root.service.busy; onClicked: {root.service.editRegistration();root.submitError="";root.keyFile="";identity.text="";httpPort.text="";amesPort.text="";bootstrap.reset()} }
}
