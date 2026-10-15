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
  property string suggested: ""
  property string submitError: ""
  signal showFleet()
  spacing: Style.space(9)
  readonly property string selection: connection ? connection.runner_id + "/" + connection.name : ""
  readonly property string pier: Model.storagePath(storageLocation.rootID,Model.pierPath(kind,instanceName.text.trim(),fakeShip.text,pierName.text))
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
  function suggest(force) {
    if (!service || !connection || !instanceName || !httpPort || service.busy) return
    if (force || !instanceName.text || instanceName.text === suggested) {
      suggested=Model.suggestInstance(connection,kind);instanceName.text=suggested
    }
    if (force || !httpPort.text) httpPort.text=Model.suggestHTTPPort(connection)
    if (force) { pierName.text="";service.editRegistration() }
  }
  onSelectionChanged: { consent.checked=false; suggestion.restart() }
  onKindChanged: { consent.checked=false; suggestion.restart() }
  Timer { id:suggestion;interval:0;running:true;onTriggered:root.suggest(false) }

  Label { width: parent.width; secondary: true; text: root.kind === "fake" ? "A fake is a development identity with container-local Ames and outbound HTTP access. Independent fakes can share an Urbit name, but need separate piers and host HTTP ports." : "A comet's Urbit identity is generated on first Start. The instance name below is your local label; registration leaves it stopped." }
  BootstrapSelector { id: bootstrap; width: parent.width; service: root.service; enabled: !!root.service && !root.service.busy && !root.registered; onEdited: {root.service.editRegistration();root.submitError=""} }
  Label { text: "Instance name · unique on this runner"; caption: true; secondary: true }
  Field { id: instanceName; objectName: "registration-name"; width: parent.width; enabled: !!root.service && !root.service.busy; placeholderText: "e.g. my-comet"; onTextEdited: root.service.editRegistration() }
  Column {
    width: parent.width; spacing: Style.space(8); visible: root.kind === "fake"
    Label { text: "Fake Urbit identity"; caption: true; secondary: true }
    Field { id: fakeShip; objectName: "registration-fake-ship"; width: parent.width; text: "~zod"; placeholderText: "e.g. ~bus"; enabled: !!root.service && !root.service.busy; onTextEdited: root.service.editRegistration() }
  }
  StorageLocationSelector { id: storageLocation; width: parent.width; connection: root.connection; enabled: !!root.service && !root.service.busy && !root.registered; onEdited: root.service.editRegistration() }
  Label { text: "Pier path · below selected storage location"; caption: true; secondary: true }
  Field { id: pierName; objectName: "registration-pier"; width: parent.width; placeholderText: root.kind === "fake" ? "e.g. fakeships/project-one/zod/pier" : "e.g. livenet/comets/testing-comet-1/pier"; enabled: !!root.service && !root.service.busy; onTextEdited: root.service.editRegistration() }
  Label { width: parent.width; secondary: true; caption: true; text: "Destination: " + Model.absolutePierPath(root.connection,root.pier) + ". Its parent is the exclusive /urbit mount; archive/ is excluded from discovery." }
  Label { text: "Host HTTP port"; caption: true; secondary: true }
  Field { id: httpPort; objectName: "registration-port"; width: parent.width; placeholderText: "Unused port, 1024–65535"; validator: IntValidator { bottom:1024; top:65535 } enabled: !!root.service && !root.service.busy; onTextEdited: root.service.editRegistration() }
  ActionButton { text: "Suggest an available name and port"; quiet: true; compact: true; enabled: !!root.connection && !!root.service && !root.service.busy; onClicked: root.suggest(true) }
  Label { width: parent.width; visible: root.problem !== ""; text: root.problem; color: Design.warning }
  CheckField {
    id: consent; objectName: "comet-consent"; width: parent.width
    visible: root.kind === "comet"
    enabled: !!root.service && !root.service.busy && root.service.canRegisterComet(root.connection)
    text: "Create a new networked comet when I choose Start."
  }
  Label { width: parent.width; visible: !!root.outcome; text: root.outcome ? root.outcome.message + (root.outcome.status === "uncertain" ? " Refresh and inspect the fleet before retrying." : "") : ""; color: root.registered ? Design.success : root.outcome && root.outcome.status === "submitting" ? Design.accent : Design.danger }
  Label { width: parent.width; visible: !root.outcome && root.submitError!==""; text: root.submitError; color: Design.danger }
  RowLayout {
    width: parent.width
    ActionButton {
      objectName: "register-instance"
      text: root.service && root.service.busy && root.outcome ? "Registering…" : root.kind === "fake" ? "Register fake ship" : "Register comet"
      primary: true
      enabled: !!root.service && root.service.observing && !root.service.busy && !root.registered && root.problem === "" && (root.kind !== "comet" || consent.checked) && !(root.outcome && root.outcome.status === "uncertain")
      onClicked: {
        root.submitError=""
        var accepted=root.kind === "fake" ? root.service.registerFake(root.connection,instanceName.text.trim(),httpPort.text,fakeShip.text,root.pier,bootstrap.kind,bootstrap.source) : root.service.registerComet(root.connection,instanceName.text.trim(),httpPort.text,consent.checked,root.pier,bootstrap.kind,bootstrap.source)
        if (!accepted) root.submitError=root.service.lastError || "The selection changed; review a fresh runner observation."
      }
    }
    ActionButton { text: "Go to fleet →"; visible: root.registered; onClicked: root.showFleet() }
  }
}
