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
  readonly property bool needsFirstCheck: !status || (!status.can_setup && !!service && !service.setupCheckRequested)
  readonly property int step: ready ? 5 : Model.setupStepIndex(service ? service.localSetupStep : "")
  Card {
    width: parent.width; implicitHeight: intro.implicitHeight + Style.space(32)
    Column {
      id: intro
      anchors.left: parent.left; anchors.right: parent.right; anchors.top: parent.top; anchors.margins: Style.space(16)
      spacing: Style.space(10)
      RowLayout {
        width: parent.width
        Label { Layout.fillWidth: true; text: root.ready ? "Your local runner is ready" : "Set up this computer"; font.pixelSize: Style.space(23); font.bold: true }
        StatusBadge { text: root.ready ? "Connected" : root.running ? "Setting up" : "Local host"; tone: root.ready ? "good" : root.running ? "accent" : "muted" }
      }
      Label { width: parent.width; secondary: true; text: root.ready ? "The background services and private desktop connection are ready. Your ships can keep running when this panel is closed." : "Setup will install Urbit software on this machine. Please check the box to approve." }
      RowLayout {
        width: parent.width
        CheckField { id: approval; objectName: "setup-install-approval"; Layout.fillWidth: true; visible: !root.ready; text: "I approve installing Urbit software"; enabled: !root.running }
        ActionButton { objectName: "setup-view-details"; compact: true; quiet: true; text: root.detailsOpen ? "Hide details ▾" : "View details ›"; onClicked: root.detailsOpen=!root.detailsOpen }
      }
      Column {
        objectName: "setup-install-details"
        width: parent.width; spacing: Style.space(10); visible: root.detailsOpen
      Label { width: parent.width; caption: true; secondary: true; text: "The runner and its separate rootless Docker engine run as your Unix user. Setup creates /srv/urbit/piers for your ordinary file access, plus restricted management SSH and reboot services. Your login shell and personal SSH settings are preserved. Host changes ask for OS authorization; setup does not boot a ship." }
      Label { objectName: "setup-binaries"; width: parent.width; caption: true; secondary: true; text: "Software: omabit (desktop/setup helper), urbit-collector (runner client), and urbit-runner (host service). The plugin downloads checksum-verified release binaries into ~/.cache/omabit/native-binaries/. Authorized setup installs urbit-runner at /usr/local/lib/urbit-runner/current/urbit-runner and Docker's verified rootless launcher under /usr/local/lib/urbit-runner/rootless/29.7.2/." }
      Label { width: parent.width; caption: true; secondary: true; text: "Docker, RootlessKit, slirp4netns and OpenSSH come from your OS packages. Ships use the pinned official Vere 4.6 container image. You can build Omabit's three tools from source with Go 1.25+ and install them yourself." }
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
  MachineUsePane { width: parent.width; service: root.service; visible: !root.ready }
  Column {
    width: parent.width; spacing: Style.space(9); visible: root.running
    Repeater {
      model: ["Check prerequisites","Create private desktop access","Verify host helper","Set up background services","Connect and pin the runner","Ready to add your first ship"]
      RowLayout {
        required property int index
        required property string modelData
        width: root.width; spacing: Style.space(10)
        StatusBadge { text: root.step > index || root.ready ? "✓" : String(index+1); tone: root.step > index || root.ready ? "good" : root.step === index && root.running ? "accent" : "muted" }
        Label { Layout.fillWidth: true; text: modelData; font.bold: root.step === index; secondary: root.step !== index && !root.ready }
        Controls.BusyIndicator { running: root.running && root.step === index; visible: running; implicitWidth: Style.space(22); implicitHeight: Style.space(22) }
      }
    }
  }
  Label { width: parent.width; visible: root.running; text: root.service ? root.service.localSetupMessage : ""; color: Design.accent }
  Label { width: parent.width; visible: root.service && !!root.service.localSetupError; text: root.service ? root.service.localSetupError : ""; color: Design.danger }
  Label { width: parent.width; visible: !!root.status && !!root.status.hint && !root.ready; text: root.status ? root.status.hint || "" : ""; secondary: true }
  Column {
    width: parent.width; spacing: Style.space(7); visible: !!root.status && !root.status.can_setup && !root.ready
    Label { text: "A few prerequisites need attention"; font.bold: true }
    Repeater {
      model: root.status ? root.status.checks.filter(function(c){ return c.status !== "pass" }) : []
      Label { required property var modelData; width: root.width; text: modelData.detail; caption: true; color: Design.warning }
    }
    Label { width: parent.width; secondary: true; caption: true; text: "Use Omarchy's package manager for Docker, rootlesskit, slirp4netns and OpenSSH, then Check again. Setup will not downgrade packages or enable system Docker." }
  }
  RowLayout {
    width: parent.width
    ActionButton { objectName: "local-setup-primary"; text: root.ready ? "Add a ship →" : root.running ? "Setting up…" : root.needsFirstCheck ? "Check prerequisites" : root.status.host_present ? "Connect this computer" : "Set up this computer"; primary: true; enabled: !!root.service && !root.service.busy && !root.service.checkingSetup && (root.ready || root.needsFirstCheck || root.status.can_setup && approval.checked); onClicked: root.ready ? root.addShip() : root.needsFirstCheck ? root.service.checkLocalSetup(true) : root.service.setUpLocal(approval.checked) }
    ActionButton { objectName: "local-check-again"; text: "Check again"; visible: !!root.service && root.service.setupCheckRequested && !root.ready; quiet: true; enabled: !!root.service && !root.running && !root.service.checkingSetup; onClicked: root.service.checkLocalSetup(true) }
  }
}
