import QtQuick
import QtQuick.Controls as Controls
import Quickshell.Io
import qs.Commons
import "Model.js" as Model

Column {
  id: root
  objectName: "owned-key-form"
  property var service
  property var connection
  property string keyFile: ""
  property bool pasteKey: false
  onPasteKeyChanged: {keyInput.clear();bootConsent.checked=false}
  onVisibleChanged: if(!visible) keyInput.clear()
  Component.onDestruction: keyInput.clear()
  property bool advanced: false
  property string submitError: ""
  onKeyFileChanged: if (bootConsent) bootConsent.checked=false
  readonly property string ship: Model.fakeIdentity(identity.text)
  readonly property string instanceName: ship.slice(1)
  readonly property string runtimeSelection: (connection ? connection.runner_id+"/"+connection.name : "")+"/"+ship
  onRuntimeSelectionChanged: {if(runtimeSettings) runtimeSettings.selectedLoom=31; if(bootConsent) bootConsent.checked=false}
  readonly property string destination: Model.ownedPierPath(ship) ? Model.storagePath(storageLocation.rootID,storageLocation.customPath && customPier.text.trim() ? customPier.text.trim() : Model.ownedPierPath(ship)) : ""
  readonly property string selectedPill: bootstrap.source
  readonly property var currentConnection: service && connection ? service.connections.filter(function(c){return c.name===root.connection.name && c.runner_id===root.connection.runner_id})[0] || null : null
  readonly property var registeredInstance: Model.instances(currentConnection).filter(function(i){return i.id===root.instanceName && i.ship===root.ship && i.mode!=="fake"})[0] || null
  readonly property bool registered: !!registeredInstance
  readonly property var outcome: {
    var o=service ? service.registrationOutcome : null
    return o && o.kind==="owned" && connection && o.runner===connection.runner_id && o.connection===connection.name && o.instance===instanceName ? o : null
  }
  readonly property bool submitted: registered || !!outcome && outcome.status!=="failed"
  readonly property string problem: submitted ? "" : !destination || !Model.validPierPath(destination) ? "Enter a galaxy, star, planet or moon identity and valid destination." : (root.pasteKey ? !keyInput.text.trim() : !root.keyFile) ? "Paste a boot key or choose its file." : !Model.optionalPort(httpPort.text) || !Model.optionalPort(amesPort.text) ? "Advanced ports must be blank or 1024–65535." : runtimeSettings.problem || bootstrap.problem
  spacing: Style.space(12)
  Label { text: "Your Urbit ID and boot key"; font.pixelSize: Style.space(21); font.bold: true }
  Label { width: parent.width; text: "Enter your Urbit ID and paste its boot key or choose a key file. Omabit uses your Urbit ID to name the instance, chooses its data directory and available ports, and boots it with your confirmation."; secondary: true }
  ActionButton { objectName: "get-urbit-id"; property url url: "https://urbit.org/overview/running-urbit/get-urbit-id"; text: "Get an Urbit ID ↗"; quiet: true; compact: true; hint: url; onClicked: Qt.openUrlExternally(url) }
  BootstrapSelector { id: bootstrap; width: parent.width; service: root.service; visible: !root.registered; enabled: !root.submitted && !!root.service && !root.service.busy; onEdited: {root.submitError="";bootConsent.checked=false} }
  Label { width: parent.width; visible: root.registered && !!root.registeredInstance.bootstrap; text: visible ? "Boot pill: "+(root.registeredInstance.bootstrap.kind==="bare" ? "bare bootstrap" : "Recommended Apps") : ""; caption: true; secondary: true }
  Label { text: "Expected Urbit ID"; caption: true; secondary: true }
  Field { id: identity; objectName: "key-identity"; width: parent.width; placeholderText: "~sampel-palnet"; enabled: !root.submitted && !root.service.busy; onTextEdited: bootConsent.checked=false }
  SelectField { objectName: "key-input-method"; width: parent.width; model: ["Key file", "Paste key"]; enabled: !root.submitted && !root.service.busy; onActivated: { root.pasteKey=currentIndex===1; keyInput.clear(); bootConsent.checked=false } }
  Column {
    width: parent.width; spacing: Style.space(8); visible: !root.pasteKey
    Label { text: "Key file location"; caption: true; secondary: true }
    Field { objectName: "key-file-location"; width: parent.width; text: root.keyFile; placeholderText: "/path/to/ship.key"; enabled: !root.submitted && !root.service.busy; onTextEdited: root.keyFile=text }
    ActionButton { objectName: "key-file-browse"; text: picker.running ? "Choosing…" : "Browse…"; enabled: !root.submitted && !root.service.busy && !picker.running; onClicked: {root.submitError=""; picker.running=true} }
    Label { width: parent.width; text: "Before boot, Omabit makes this owned key file private to your user (permissions 600)."; caption: true; secondary: true }
  }
  Field { id: keyInput; objectName: "key-pasted"; width: parent.width; visible: root.pasteKey; placeholderText: "Paste your 0w boot key"; maximumLength: 4096; echoMode: TextInput.Password; inputMethodHints: Qt.ImhSensitiveData | Qt.ImhNoPredictiveText; enabled: !root.submitted && !root.service.busy; onTextEdited: bootConsent.checked=false }
  // An external picker cannot crash the shared desktop shell. No key contents
  // cross its stdout: only the selected local filename is returned.
  Process {
    id: picker
    command: ["zenity", "--file-selection", "--title=Select Urbit boot key"]
    stdout: StdioCollector { onStreamFinished: { if (text.trim() && text.length<=4096) root.keyFile=text.trim() } }
    stderr: StdioCollector {}
    onExited: function(code,status) { if(code!==0 && code!==1) root.submitError="File picker could not open. Enter the file path or paste the key." }
  }
  Label { width: parent.width; visible: root.destination!==""; text: "Instance: "+root.instanceName; secondary: true; caption: true }
  ActionButton { objectName: "key-advanced-toggle"; text: root.advanced ? "Advanced settings −" : "Advanced settings +"; quiet: true; onClicked: root.advanced=!root.advanced }
  Column {
    objectName: "key-advanced"; width: parent.width; spacing: Style.space(10); visible: root.advanced
    RuntimeSettingsFields { id: runtimeSettings; width: parent.width; supported: Model.hasFeature(root.connection,"runtime-settings"); enabled: !root.submitted && !!root.service && !root.service.busy; onEdited: {bootConsent.checked=false;root.submitError=""} }
    StorageLocationSelector { id: storageLocation; allowCustom: true; width: parent.width; connection: root.connection; visible: !root.registered; enabled: !root.submitted && !root.service.busy; onEdited: bootConsent.checked=false }
    Field { id: customPier; objectName: "key-pier-path"; width: parent.width; visible: storageLocation.customPath; placeholderText: Model.ownedPierPath(root.ship); enabled: !root.submitted && !root.service.busy; onTextEdited: bootConsent.checked=false }
    Label { width: parent.width; visible: root.destination!==""; text: "Pier: "+Model.absolutePierPath(root.connection,root.destination); secondary: true; caption: true }
    Label { text: "Ames UDP port"; caption: true; secondary: true }
    Field { id: amesPort; objectName: "key-ames-port"; width: parent.width; placeholderText: "Automatic"; enabled: !root.submitted && !root.service.busy; validator: IntValidator {bottom:1024;top:65535} onTextEdited: bootConsent.checked=false }
    Label { text: "HTTP port"; caption: true; secondary: true }
    Field { id: httpPort; objectName: "key-http-port"; width: parent.width; placeholderText: "Automatic"; enabled: !root.submitted && !root.service.busy; validator: IntValidator {bottom:1024;top:65535} onTextEdited: bootConsent.checked=false }
    Label { width: parent.width; text: "Your Urbit ship will run in Docker at port 8080 and be exposed on this host at the selected HTTP port. Ames UDP is published on the selected host port."; secondary: true; caption: true }
  }
  Label { width: parent.width; visible: root.problem!=="" && identity.text!==""; text: root.problem; secondary: true; caption: true }
  CheckField { id: bootConsent; objectName: "key-boot-consent"; width: parent.width; visible: !root.registered; enabled: !root.submitted && !root.service.busy; text: "Boot this identity now with these settings." }
  Flow {
    width: parent.width; spacing: Style.space(8)
    ActionButton { objectName: "key-register"; text: root.registered ? "Boot submitted" : root.submitted ? "Starting…" : root.service.busy ? "Preparing boot files…" : "Boot ship"; primary: !root.registered; enabled: root.connection && root.connection.access_role==="operator" && Model.hasFeature(root.connection,"key-boot-options") && !root.submitted && !root.service.busy && root.problem==="" && bootConsent.checked; onClicked: {root.submitError="";if(!root.service.createFromKey(root.connection,root.ship,root.keyFile,httpPort.text,amesPort.text,root.selectedPill,bootstrap.kind,storageLocation.rootID,root.destination,runtimeSettings.selectedLoom,root.pasteKey ? keyInput.text.trim() : ""))root.submitError=root.service.lastError;else keyInput.clear()} }
    ActionButton {
      objectName: "key-start"
      text: "▶ Start"
      visible: root.registered && !root.service.pendingFirstBoot && !root.service.bootTarget && root.registeredInstance.desired_state==="stopped"
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
  ActionButton { objectName: "key-register-another"; text: "Register another ship"; visible: root.registered || !!root.outcome && root.outcome.status==="succeeded"; enabled: !root.service.busy; onClicked: {root.service.editRegistration();root.submitError="";root.keyFile="";identity.text="";httpPort.text="";amesPort.text="";customPier.text="";storageLocation.reset();root.advanced=false;bootstrap.reset()} }
}
