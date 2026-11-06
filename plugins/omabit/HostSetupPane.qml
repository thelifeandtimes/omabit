import QtQuick
import QtQuick.Controls as Controls
import QtQuick.Layouts
import qs.Commons
import "Model.js" as Model

Column {
  id: root
  property var service
  property bool detailsOpen: false
  signal addShip()
  width: parent.width
  spacing: Style.space(14)
  readonly property var status: service ? service.localSetupStatus : null
  readonly property bool ready: !!status && status.ready
  readonly property bool running: !!service && service.settingUp
  readonly property var packages: status && status.packages_to_install || []
  readonly property int step: ready ? 6 : Model.setupStepIndex(service ? service.localSetupStep : "")
  Card {
    width: parent.width; implicitHeight: intro.implicitHeight + Style.space(32)
    Column {
      id: intro
      anchors.left: parent.left; anchors.right: parent.right; anchors.top: parent.top; anchors.margins: Style.space(16)
      spacing: Style.space(10)
      RowLayout {
        width: parent.width
        CardTitle { Layout.fillWidth: true; text: root.ready ? "Your local runner is ready" : "Set up this computer" }
        StatusBadge { text: root.ready ? "Connected" : root.running ? "Setting up" : "Local host"; tone: root.ready ? "good" : root.running ? "accent" : "muted" }
      }
      Label { objectName: "setup-description"; width: parent.width; secondary: true; text: root.ready ? "The background services and private desktop connection are ready. Your ships can keep running when this panel is closed." : root.running ? "Installing and configuring the software needed to run Urbit on your Omarchy computer. Enter your password when the system asks." : "Install and configure the software needed to run Urbit on your Omarchy computer. Choose Set up this computer, then enter your password when the system asks." }
      RowLayout {
        width: parent.width
        ActionButton { objectName: "setup-view-details"; compact: true; quiet: true; text: root.detailsOpen ? "Hide details ▾" : "View details ›"; onClicked: root.detailsOpen=!root.detailsOpen }
      }
      Column {
        objectName: "setup-install-details"
        width: parent.width; spacing: Style.space(10); visible: root.detailsOpen
      Label { width: parent.width; font.bold: true; text: "System dependencies" }
      Label { width: parent.width; caption: true; secondary: true; text: "Docker provides the containers that run your ships. RootlessKit and slirp4netns let those containers run under your user account, and OpenSSH provides private management access. Setup installs missing dependencies from the Arch/Omarchy repositories." }
      Label { objectName: "setup-native-packages"; width: parent.width; caption: true; visible: !root.ready && root.packages.length > 0; text: "Missing dependencies to install: " + root.packages.join(", ") + "." }
      Label { width: parent.width; font.bold: true; text: "Omabit software" }
      Label { objectName: "setup-binaries"; width: parent.width; caption: true; secondary: true; text: "The plugin includes three Omabit tools: omabit handles desktop setup, urbit-runner manages your ships, and urbit-collector connects the desktop to your runners. Setup installs urbit-runner as a background service." }
      Label { width: parent.width; font.bold: true; text: "Urbit software" }
      Label { width: parent.width; caption: true; secondary: true; text: "Vere is the Urbit runtime—the software that boots and runs your ships. Omabit uses the pinned official Vere 4.6 Docker image, running each ship in its own container with its data stored separately." }
      Flow {
        width: parent.width; spacing: Style.space(8)
        ActionButton { objectName: "setup-source"; compact: true; text: "Source code ↗"; onClicked: Qt.openUrlExternally("https://github.com/thelifeandtimes/omabit") }
        ActionButton { objectName: "setup-build-guide"; compact: true; text: "Build & install yourself ↗"; onClicked: Qt.openUrlExternally("https://github.com/thelifeandtimes/omabit/blob/master/docs/install.md#build-and-install-from-source") }
        ActionButton { compact: true; quiet: true; text: "Docker source ↗"; onClicked: Qt.openUrlExternally("https://github.com/moby/moby") }
        ActionButton { compact: true; quiet: true; text: "Vere source ↗"; onClicked: Qt.openUrlExternally("https://github.com/urbit/vere") }
      }
      }
    }
  }
  ServiceUpgradePane {
    width:parent.width
    updater:root.service && root.service.serviceUpgrade ? root.service.serviceUpgrade : null
    visible:!!updater && !!root.status && (root.status.host_present || root.ready) && root.service.localSetupAllowed
  }
  MachineUsePane { width: parent.width; service: root.service; visible: !root.ready }
  Column {
    objectName: "setup-progress-checklist"
    width: parent.width; spacing: Style.space(9); visible: root.running || root.ready
    Repeater {
      model: ["Check prerequisites","Create private desktop access","Verify host helper","Install and verify native packages","Set up background services","Connect and pin the runner","Ready to add your first ship"]
      RowLayout {
        required property int index
        required property string modelData
        objectName: "setup-progress-step-"+index
        width: root.width; spacing: Style.space(10)
        StatusBadge { objectName: "setup-progress-state-"+parent.index; text: root.step > index || root.ready ? "✓" : String(index+1); tone: root.step > index || root.ready ? "good" : root.step === index && root.running ? "accent" : "muted" }
        Label { Layout.fillWidth: true; text: modelData; font.bold: root.step === index; secondary: root.step !== index && !root.ready }
        Controls.BusyIndicator { running: root.running && root.step === index; visible: running; implicitWidth: Style.space(22); implicitHeight: Style.space(22) }
      }
    }
  }
  Label { width: parent.width; visible: root.running; text: root.service ? root.service.localSetupMessage : ""; color: Design.accent }
  Label { width: parent.width; visible: root.service && !!root.service.localSetupError; text: root.service ? root.service.localSetupError : ""; color: Design.danger }
  Label { width: parent.width; visible: !!root.status && !!root.status.hint && !root.ready; text: root.status ? root.status.hint || "" : ""; secondary: true }
  HostCompatibility {
    objectName: "local-host-compatibility"
    width: parent.width; visible: root.detailsOpen && !!root.status
    report: root.status ? root.status.compatibility || null : null
    problem: root.status ? root.status.compatibility_problem || (report ? "" : "Update the local Omabit and runner helpers for structured host compatibility checks.") : ""
    details: root.detailsOpen
    now: root.service ? root.service.now : Date.now()
  }
  Column {
    width: parent.width; spacing: Style.space(7); visible: !!root.status && !root.status.can_setup && !root.ready && !root.running
    Label { text: "A few prerequisites need attention"; font.bold: true }
    Repeater {
      model: root.status && Array.isArray(root.status.checks) ? root.status.checks.filter(function(c){ return c.status !== "pass" }) : []
      Label { required property var modelData; width: root.width; text: modelData.id + ": " + modelData.detail; caption: true; color: Design.warning }
    }
    Label { width: parent.width; secondary: true; caption: true; text: "Resolve the listed prerequisites, then Check again. Additional diagnostics are available in View details." }
  }
  RowLayout {
    width: parent.width
    ActionButton { objectName: "local-setup-primary"; text: root.ready ? "Add a ship →" : root.running ? "Setting up…" : !root.status ? "Checking this computer…" : "Set up this computer"; primary: true; enabled: !!root.service && !root.service.busy && !root.service.checkingSetup && !root.running && !!root.status && (root.ready || root.status.can_setup); onClicked: root.ready ? root.addShip() : root.service.setUpLocal(true) }
    ActionButton { objectName: "local-check-again"; text: "Check again"; visible: !!root.status && !root.ready && !root.running; quiet: true; enabled: !!root.service && !root.service.checkingSetup; onClicked: root.service.checkLocalSetup(true) }
  }
}
