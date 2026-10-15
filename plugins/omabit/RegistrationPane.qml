import QtQuick
import QtQuick.Layouts
import qs.Commons
import "Model.js" as Model

Column {
  id: root
  objectName: "registration-form"
  property var service
  property var connection: null
  property string kind: "comet"
  property string portToken: ""
  property string submitError: ""
  signal showFleet()
  spacing: Style.space(9)
  readonly property string selection: connection ? connection.runner_id + "/" + connection.name : ""
  readonly property string pier: Model.storagePath(storageLocation.rootID,Model.pierPath(kind,instanceName.text.trim(),fakeShip.text,storageLocation.customPath ? pierName.text : ""))
  readonly property var outcome: {
    var o=service ? service.registrationOutcome : null
    return o && connection && o.runner === connection.runner_id && o.instance === instanceName.text.trim()
      && o.kind === kind && o.port === Number(httpPort.text) && o.bootstrap === bootstrap.kind && o.pillSource === bootstrap.source
      && (kind !== "fake" || o.ship === Model.fakeIdentity(fakeShip.text) && o.pier === pier) ? o : null
  }
  readonly property bool registered: !!outcome && outcome.status === "succeeded"
  readonly property string problem: {
    if (!service || registered) return ""
    var error=Model.registrationProblem(connection,instanceName.text.trim(),httpPort.text,kind,service.now)
    return error || (!Model.hasFeature(connection,"bootstrap-selection") ? "Update this runner for boot-pill selection." : bootstrap.problem) || (!Model.hasFeature(connection,"pier-storage") ? "Update this runner for the pier storage convention." : !Model.validPierPath(pier) ? "Choose a convention-relative pier path." : kind === "fake" ? Model.fakeOptionsProblem(connection,fakeShip.text,pier) : "")
  }
  function suggestPort() {
    if (!service || !connection || service.busy) return
    portToken=Model.requestId();consent.checked=false
    service.suggestPort(connection,portToken)
  }
  onSelectionChanged: { consent.checked=false;portToken="";httpPort.text="";suggestion.restart() }
  onKindChanged: { consent.checked=false;instanceName.text=kind==="fake" ? Model.fakeIdentity(fakeShip.text).slice(1) : "";suggestion.restart() }
  onVisibleChanged: if (visible) suggestion.restart()
  Timer { id:suggestion;interval:0;running:true;onTriggered:if(root.visible && !httpPort.text && root.connection)root.suggestPort() }
  Connections {
    target: root.service
    function onPortSuggestionChanged() {
      var p=root.service.portSuggestion
      if(p && root.connection && p.runner===root.connection.runner_id && p.connection===root.connection.name && p.token===root.portToken) {httpPort.text=String(p.port);root.service.editRegistration()}
    }
  }

  Label { width: parent.width; secondary: true; text: root.kind === "fake" ? "Boot a development identity with container-local Ames and outbound HTTP access." : "Boot a new comet identity. Choose a required local name; the Urbit identity is generated during startup." }
  BootstrapSelector { id: bootstrap; width: parent.width; service: root.service; enabled: !!root.service && !root.service.busy && !root.registered; onEdited: {consent.checked=false;root.service.editRegistration();root.submitError=""} }
  Label { text: "Instance name · required, unique on this runner"; caption: true; secondary: true }
  Field { id: instanceName; objectName: "registration-name"; width: parent.width; enabled: !!root.service && !root.service.busy && !root.registered; placeholderText: Model.suggestInstance(root.connection,root.kind); onTextEdited: {consent.checked=false;root.service.editRegistration()} }
  Column {
    width: parent.width; spacing: Style.space(8); visible: root.kind === "fake"
    Label { text: "Fake Urbit identity"; caption: true; secondary: true }
    Field { id: fakeShip; objectName: "registration-fake-ship"; width: parent.width; text: "~zod"; placeholderText: "e.g. ~bus"; enabled: !!root.service && !root.service.busy; onTextEdited: {instanceName.text=Model.fakeIdentity(text).slice(1);consent.checked=false;root.service.editRegistration()} }
  }
  StorageLocationSelector { id: storageLocation; allowCustom: true; width: parent.width; connection: root.connection; enabled: !!root.service && !root.service.busy && !root.registered; onEdited: {consent.checked=false;root.service.editRegistration()} }
  Label { visible: storageLocation.customPath; text: "Custom pier path · below selected storage location"; caption: true; secondary: true }
  Field { id: pierName; objectName: "registration-pier"; width: parent.width; visible: storageLocation.customPath; placeholderText: root.kind === "fake" ? "e.g. fakeships/project-one/zod/pier" : "e.g. livenet/comets/testing-comet-1/pier"; enabled: !!root.service && !root.service.busy; onTextEdited: {consent.checked=false;root.service.editRegistration()} }
  Label { width: parent.width; secondary: true; caption: true; text: "Destination: " + Model.absolutePierPath(root.connection,root.pier) }
  Label { text: "Host HTTP port"; caption: true; secondary: true }
  Field { id: httpPort; objectName: "registration-port"; width: parent.width; placeholderText: "Unused port, 1024–65535"; validator: IntValidator { bottom:1024; top:65535 } enabled: !!root.service && !root.service.busy; onTextEdited: {root.portToken="";consent.checked=false;root.service.editRegistration()} }
  Label { width: parent.width; secondary: true; caption: true; text: "Your Urbit ship will run in Docker at port 8080 and be exposed on this host at the selected port." }
  ActionButton { objectName: "suggest-host-port"; text: "Find an available port"; quiet: true; compact: true; enabled: !!root.connection && !!root.service && !root.service.busy; onClicked: root.suggestPort() }
  Label { width: parent.width; visible: root.problem !== ""; text: root.problem; color: Design.warning }
  CheckField {
    id: consent; objectName: "comet-consent"; width: parent.width
    enabled: !!root.service && !root.service.busy && !root.registered
    text: root.kind==="comet" ? "Boot a new comet identity now with these settings." : "Boot this development identity now with these settings."
  }
  Label { width: parent.width; visible: !!root.outcome; text: root.outcome ? root.outcome.message + (root.outcome.status === "uncertain" ? " Refresh and inspect the fleet before retrying." : "") : ""; color: root.registered ? Design.success : root.outcome && root.outcome.status === "submitting" ? Design.accent : Design.danger }
  Label { width: parent.width; visible: !root.outcome && root.submitError!==""; text: root.submitError; color: Design.danger }
  RowLayout {
    width: parent.width
    ActionButton {
      objectName: "register-instance"
      text: root.service && root.service.busy && root.outcome ? "Starting…" : root.kind === "fake" ? "Boot fake ship" : "Boot comet"
      primary: true
      enabled: !!root.service && root.service.observing && !root.service.busy && !root.registered && root.problem === "" && consent.checked && !(root.outcome && root.outcome.status === "uncertain")
      onClicked: {
        root.submitError=""
        var accepted=root.kind === "fake" ? root.service.registerFake(root.connection,instanceName.text.trim(),httpPort.text,fakeShip.text,root.pier,bootstrap.kind,bootstrap.source) : root.service.registerComet(root.connection,instanceName.text.trim(),httpPort.text,consent.checked,root.pier,bootstrap.kind,bootstrap.source)
        if (!accepted) root.submitError=root.service.lastError || "The selection changed; review a fresh runner observation."
      }
    }
    ActionButton { text: "Go to fleet →"; visible: root.registered; onClicked: root.showFleet() }
  }
}
