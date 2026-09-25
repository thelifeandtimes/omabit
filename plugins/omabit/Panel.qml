import QtQuick
import QtQuick.Controls as Controls
import QtQuick.Layouts
import qs.Commons
import qs.Ui as Ui
import "Model.js" as Model

Ui.Panel {
  id: root
  moduleName: "thelifeandtimes.omabit"
  ipcTarget: "thelifeandtimes.omabit"
  property var service: bar && bar.shell && typeof bar.shell.serviceFor === "function" ? bar.shell.serviceFor(moduleName) : null
  property var editTarget: null
  property bool addingRunner: false
  property bool addingFake: false
  property bool addingShip: false
  property var loginTarget: null
  readonly property var connections: service ? service.connections : []
  readonly property var readyConnections: connections.filter(function(c) {
    return root.service && root.service.observing && Model.writable(c, root.service.now)
      && Array.isArray(c.features) && c.features.indexOf("create-fake") >= 0
  })
  implicitWidth: button.implicitWidth
  implicitHeight: button.implicitHeight
  onOpenedChanged: { clearLogin(); if (opened && service) service.refresh() }
  function clearLogin() { loginCode.text = ""; loginTarget = null }
  Component.onDestruction: loginCode.text = ""
  Connections { target: root.service; function onCanLoginChanged() { if (!root.service.canLogin) loginCode.text = "" } }
  function syncSettings() {
    // Omarchy injects inline settings into the bar entry, not its service.
    if (service && service.settings !== settings) { service.settings = settings; service.refresh() }
  }
  onServiceChanged: Qt.callLater(syncSettings)
  onSettingsChanged: Qt.callLater(syncSettings)

  Ui.BarIconButton {
    id: button
    anchors.fill: parent
    bar: root.bar
    text: "~+"
    active: root.opened
    tooltipText: "Omabit — " + root.connections.length + " enrolled runners"
    onPressed: function(code) { if (code === Qt.RightButton) { if (root.service) root.service.refresh() } else root.toggle() }
  }

  Ui.KeyboardPanel {
    owner: root
    anchorItem: button
    bar: root.bar
    open: root.opened
    focusTarget: form
    contentWidth: fittedContentWidth(Style.space(560))
    contentHeight: fittedContentHeight(content.implicitHeight, Style.space(760))
    FocusScope {
      id: form
      anchors.fill: parent
      Keys.onEscapePressed: root.close()
      Flickable {
        id: scroll
        anchors.fill: parent
        contentWidth: width
        contentHeight: content.implicitHeight
        clip: true
        boundsBehavior: Flickable.StopAtBounds
        Controls.ScrollBar.vertical: Controls.ScrollBar {}
        Column {
          id: content
          width: scroll.width
          spacing: Style.space(10)
          Ui.PanelSectionHeader { text: "OMABIT / DEVELOPMENT" }
          Body { text: "Runner control and ship application sessions are separate connections." }
          RowLayout {
            width: parent.width
            Action { text: "Refresh"; enabled: !!root.service && !root.service.busy; onClicked: root.service.refresh() }
            Action { text: "Enroll runner"; enabled: !!root.service && !root.service.busy; onClicked: root.addingRunner = !root.addingRunner }
            Action { text: "Register fake"; enabled: root.readyConnections.length > 0 && !root.service.busy; onClicked: root.addingFake = !root.addingFake }
          }
          Body { visible: text !== ""; color: Color.urgent; text: root.service ? root.service.feedError : "Collector service is not available." }
          Body { visible: text !== ""; color: Color.urgent; text: root.service ? root.service.lastError : "" }
          Body { visible: text !== ""; text: root.service ? root.service.lastResult : "" }
          Body { visible: root.service && root.service.busy; text: "Submitting " + (root.service ? root.service.currentAction : "") + "…" }
          Body { visible: text !== ""; font.pixelSize: Style.font.caption; text: root.service && root.service.requestIdentity ? "Last request: " + root.service.requestIdentity : "" }

          Column {
            width: parent.width
            spacing: Style.space(6)
            visible: root.addingRunner
            Body { text: "Use a host alias from your SSH configuration. SSH access must already be installed." }
            Input { id: connectionName; placeholderText: "Connection name, e.g. workstation" }
            Input { id: hostAlias; placeholderText: "SSH host alias, e.g. omabit-local-4b" }
            Input { id: sshConfig; placeholderText: "Optional absolute SSH config path" }
            Action { text: "Enroll"; enabled: root.service && !root.service.busy; onClicked: if (root.service.enroll(connectionName.text.trim(), hostAlias.text.trim(), sshConfig.text.trim())) root.addingRunner = false }
          }
          Column {
            width: parent.width
            spacing: Style.space(6)
            visible: root.addingFake
            Body { text: "Registers a fresh fake ~zod without starting it. This prototype supports loom 31 and loopback HTTP only." }
            Controls.ComboBox { id: registrationRunner; width: parent.width; model: root.readyConnections; textRole: "name" }
            Input { id: registrationName; placeholderText: "Instance name" }
            Input { id: registrationPort; placeholderText: "Unused loopback HTTP port, e.g. 18085"; validator: IntValidator { bottom: 1024; top: 65535 } }
            Action {
              text: "Register"
              enabled: root.service && !root.service.busy && registrationRunner.currentIndex >= 0
              onClicked: if (root.service.registerFake(root.readyConnections[registrationRunner.currentIndex], registrationName.text.trim(), registrationPort.text)) root.addingFake = false
            }
          }
          Ui.PanelSeparator { width: parent.width }
          Ui.PanelSectionHeader { text: "RUNNERS" }
          Body { visible: root.connections.length === 0; text: "No current runner observations. Enroll a runner or refresh to reconnect." }
          Repeater {
            model: root.connections
            Column {
              id: connectionRow
              required property var modelData
              width: content.width
              spacing: Style.space(6)
              readonly property bool available: root.service && root.service.observing && Model.available(modelData, root.service.now)
              Body { font.bold: true; text: connectionRow.modelData.name + " — " + (connectionRow.available ? "available" : connectionRow.modelData.status === "available" ? "observation stale" : connectionRow.modelData.status) }
              Body { font.pixelSize: Style.font.caption; text: connectionRow.modelData.runner_id + " / " + (connectionRow.modelData.access_role || "local access") }
              Body { visible: !!connectionRow.modelData.error; text: connectionRow.modelData.error ? connectionRow.modelData.error.message : "" }
              Repeater {
                model: connectionRow.available && connectionRow.modelData.snapshot ? connectionRow.modelData.snapshot.instances : []
                Column {
                  id: instanceRow
                  required property var modelData
                  width: connectionRow.width
                  spacing: Style.space(4)
                  Body { text: instanceRow.modelData.id + (instanceRow.modelData.ship ? " / " + instanceRow.modelData.ship : "") + " — " + instanceRow.modelData.observed_state }
                  Body { font.pixelSize: Style.font.caption; text: "Desired: " + instanceRow.modelData.desired_state + " · health: " + (instanceRow.modelData.health || "unknown") + " · revision/applied: " + instanceRow.modelData.config_revision + "/" + instanceRow.modelData.applied_revision }
                  Body { visible: !!instanceRow.modelData.endpoint; text: instanceRow.modelData.endpoint || "" }
                  Body { visible: !!instanceRow.modelData.active_operation_id; text: "Active: " + (instanceRow.modelData.active_operation_id || "") }
                  Body { visible: !!instanceRow.modelData.recovery_required; color: Color.urgent; text: "Recovery required — inspect the runner snapshot before further operations." }
                  RowLayout {
                    width: parent.width
                    Action { objectName: "start-" + instanceRow.modelData.id; text: "Start runtime"; enabled: root.service && !root.service.busy && Model.canOperate(connectionRow.modelData, instanceRow.modelData, "start", root.service.now); onClicked: root.service.operate(connectionRow.modelData, instanceRow.modelData, "start") }
                    Action { objectName: "stop-" + instanceRow.modelData.id; text: "Stop runtime"; enabled: root.service && !root.service.busy && Model.canOperate(connectionRow.modelData, instanceRow.modelData, "stop", root.service.now); onClicked: root.service.operate(connectionRow.modelData, instanceRow.modelData, "stop") }
                    Action {
                      text: "HTTP port"
                      enabled: root.service && !root.service.busy && Model.writable(connectionRow.modelData, root.service.now) && !instanceRow.modelData.active_operation_id && !instanceRow.modelData.recovery_required
                      onClicked: { root.editTarget = Model.edit(connectionRow.modelData, instanceRow.modelData); editPort.text = String(root.editTarget.port) }
                    }
                  }
                }
              }
              Repeater {
                model: connectionRow.available && connectionRow.modelData.snapshot ? connectionRow.modelData.snapshot.incidents.filter(function(i) { return i.active }) : []
                Body { required property var modelData; color: Color.urgent; text: modelData.code + ": " + modelData.message }
              }
              Ui.PanelSeparator { width: parent.width }
            }
          }
          Column {
            width: parent.width
            spacing: Style.space(6)
            visible: !!root.editTarget
            Body { text: root.editTarget ? "Stage HTTP port for " + root.editTarget.connection + "/" + root.editTarget.instance + " at revision " + root.editTarget.revision : "" }
            Body { text: "The port applies on the next explicit runtime stop/start. A concurrent edit returns a conflict." }
            Input { id: editPort; validator: IntValidator { bottom: 1024; top: 65535 } }
            RowLayout {
              Action { text: "Save configuration"; enabled: root.service && root.service.canConfigure(root.editTarget); onClicked: if (root.service.configure(root.editTarget, editPort.text)) root.editTarget = null }
              Action { text: "Cancel"; onClicked: root.editTarget = null }
            }
          }
          Ui.PanelSectionHeader { text: "APPLICATION CONNECTIONS" }
          Body { text: "Application login uses Eyre directly. Sessions are stored in your desktop keyring; codes are never saved." }
          Body { text: root.service && root.service.palette ? "Current palette: " + root.service.palette.name : "Omarchy palette preview unavailable." }
          RowLayout {
            width: parent.width
            Repeater {
              model: ["primary", "secondary", "tertiary", "background", "surface"]
              Rectangle { required property string modelData; Layout.fillWidth: true; height: Style.space(22); color: root.service && root.service.palette ? root.service.palette[modelData] : "transparent"; border.color: Color.foreground; radius: Style.cornerRadius }
            }
          }
          Body { text: "Enabling theme sync selects this palette on all Talon clients for that ship and disables Talon's separate accent override." }
          Action { text: "Add ship"; enabled: root.service && !root.service.busy; onClicked: root.addingShip = !root.addingShip }
          Column {
            width: parent.width; spacing: Style.space(6); visible: root.addingShip
            Input { id: appName; placeholderText: "Connection name" }
            Input { id: appOrigin; placeholderText: "HTTPS origin or loopback development URL" }
            Input { id: appShip; placeholderText: "Expected ~ship" }
            Action { text: "Enroll application connection"; enabled: root.service && !root.service.busy; onClicked: if (root.service.enrollShip(appName.text.trim(), appOrigin.text.trim(), appShip.text.trim())) root.addingShip = false }
          }
          Repeater {
            model: root.service ? root.service.ships : []
            Column {
              id: appRow
              required property var modelData
              readonly property var theme: root.service.themeState.ships[modelData.id] || ({automatic: false})
              width: content.width; spacing: Style.space(5)
              Body { text: appRow.modelData.name + " / " + appRow.modelData.ship + " — " + (root.service.shipStatus[appRow.modelData.id] ? root.service.shipStatus[appRow.modelData.id].status : appRow.modelData.status) + "\n" + appRow.modelData.origin }
              RowLayout {
                Action { text: "Sign in"; enabled: root.service.canLogin; onClicked: { root.clearLogin(); root.loginTarget = Object.assign({}, appRow.modelData); loginCode.forceActiveFocus() } }
                Action { text: "Check"; enabled: !root.service.busy; onClicked: root.service.shipAction(appRow.modelData, "ship-status", "") }
                Action { text: "Log out"; enabled: !root.service.busy; onClicked: root.service.shipAction(appRow.modelData, "ship-logout", "") }
                Action { text: "Forget"; enabled: !root.service.busy && appRow.modelData.status === "login_required"; onClicked: root.service.shipAction(appRow.modelData, "ship-forget", "") }
              }
              RowLayout {
                Action { text: appRow.theme.automatic ? "Pause theme" : "Sync theme"; enabled: !root.service.busy && appRow.modelData.status !== "login_required"; onClicked: root.service.themeAuto(appRow.modelData, !appRow.theme.automatic) }
                Action { text: "Publish now"; enabled: !root.service.busy && appRow.modelData.status !== "login_required"; onClicked: root.service.publishTheme(appRow.modelData) }
              }
              Body { visible: !!appRow.theme.last_error; text: appRow.theme.last_error || ""; color: Color.urgent }
            }
          }
          Body { visible: root.service && root.service.ships.length === 0; text: "No application connections enrolled." }
          Column {
            width: parent.width; spacing: Style.space(6); visible: !!root.loginTarget
            Body { text: root.loginTarget ? "Sign in to " + root.loginTarget.name + " at " + root.loginTarget.origin : "" }
            Input { id: loginCode; objectName: "shipLoginCode"; placeholderText: "Web login code"; echoMode: TextInput.Password; maximumLength: 128; enabled: root.service && root.service.canLogin }
            RowLayout {
              Action { text: "Sign in"; enabled: root.service && root.service.canLogin && loginCode.text.length > 0; onClicked: { try { root.service.shipAction(root.loginTarget, "ship-login", loginCode.text) } finally { root.clearLogin() } } }
              Action { text: "Cancel"; onClicked: root.clearLogin() }
            }
          }
        }
      }
    }
  }

  component Body: Text {
    width: parent.width
    textFormat: Text.PlainText
    wrapMode: Text.Wrap
    color: Color.foreground
    font.family: Style.font.family
    font.pixelSize: Style.font.bodySmall
  }
  component Input: Controls.TextField {
    width: parent.width
    color: Color.foreground
    font.family: Style.font.family
    font.pixelSize: Style.font.bodySmall
    background: Rectangle { color: Color.background; border.color: Color.foreground; radius: Style.cornerRadius }
  }
  component Action: Controls.Button {
    id: control
    implicitHeight: Style.space(34)
    padding: Style.space(7)
    contentItem: Text { text: control.text; textFormat: Text.PlainText; color: Color.foreground; opacity: control.enabled ? 1 : 0.45; font.family: Style.font.family; font.pixelSize: Style.font.bodySmall; horizontalAlignment: Text.AlignHCenter; verticalAlignment: Text.AlignVCenter }
    background: Rectangle { color: control.down ? Color.accent : Color.background; border.color: Color.foreground; opacity: control.enabled ? 1 : 0.45; radius: Style.cornerRadius }
  }
}
