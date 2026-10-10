import QtQuick
import QtQuick.Controls as Controls
import QtQuick.Layouts
import qs.Commons
import "Model.js" as Model

Column {
  id: root
  property var service
  signal addShip()
  width: parent.width
  spacing: Style.space(14)
  readonly property var status: service ? service.localSetupStatus : null
  readonly property bool ready: !!status && status.ready
  readonly property bool running: !!service && service.settingUp
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
      Label { width: parent.width; secondary: true; text: root.ready ? "The background services and private desktop connection are ready. Your ships can keep running when this panel is closed." : "A private Urbit host for this computer, using the official Vere release. Omabit handles the services and connects your desktop automatically." }
      Label { width: parent.width; caption: true; secondary: true; visible: !root.ready; text: "Setup creates or verifies the dedicated urbit-runner account, rootless Docker, restricted SSH access and reboot services. Host changes ask for your OS password; setup does not boot a ship." }
    }
  }
  Column {
    width: parent.width; spacing: Style.space(9)
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
    ActionButton { text: root.ready ? "Add a ship →" : root.running ? "Setting up…" : root.status && root.status.host_present ? "Connect this computer" : "Set up this computer"; primary: true; enabled: root.service && !root.service.busy && !!root.status && (root.ready || root.status.can_setup); onClicked: root.ready ? root.addShip() : root.service.setUpLocal() }
    ActionButton { text: "Check again"; quiet: true; enabled: root.service && !root.running && !root.service.checkingSetup; onClicked: root.service.checkLocalSetup() }
  }
  Label { width: parent.width; caption: true; secondary: true; text: "First ship startup needs at least 5 GiB available RAM and 8 GiB free disk. New comets are supported; owned-ship keys and ordinary pier import are still upcoming." }
}
