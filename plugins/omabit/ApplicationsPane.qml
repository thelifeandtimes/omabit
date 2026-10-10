import QtQuick
import QtQuick.Controls as Controls
import QtQuick.Layouts
import qs.Commons

Column {
  id: root
  property var service
  property bool active: true
  property bool adding: false
  property var loginTarget: null
  spacing: Style.space(10)
  function clearLogin() { code.text = ""; loginTarget = null }
  onActiveChanged: if (!active) clearLogin()
  Component.onDestruction: code.text = ""
  Connections { target: root.service; function onCanLoginChanged() { if (!root.service.canLogin) code.text = "" } }
  Label { text: "Application connections"; font.pixelSize: Style.space(19); font.bold: true }
  Label { width: parent.width; text: "Eyre sessions are independent of runner control. Login codes are sent through stdin; sessions use the desktop keyring."; secondary: true; caption: true }
  ActionButton { text: "+ Add application connection"; onClicked: root.adding = !root.adding }
  Column {
    width: parent.width; spacing: Style.space(8); visible: root.adding
    Field { id: name; width: parent.width; placeholderText: "Connection name" }
    Field { id: origin; width: parent.width; placeholderText: "HTTPS origin or loopback development URL" }
    Field { id: ship; width: parent.width; placeholderText: "Expected ~ship" }
    ActionButton { text: "Enroll"; primary: true; enabled: root.service && !root.service.busy; onClicked: if (root.service.enrollShip(name.text.trim(), origin.text.trim(), ship.text.trim())) root.adding = false }
  }
  Repeater {
    model: root.service ? root.service.ships : []
    Column {
      id: row
      required property var modelData
      readonly property var theme: root.service.themeState.ships[modelData.id] || ({automatic:false})
      width: root.width; spacing: Style.space(8)
      Label { text: row.modelData.name + " / " + row.modelData.ship; font.bold: true }
      Label { width: parent.width; text: row.modelData.origin; secondary: true; caption: true }
      Flow {
        width: parent.width; spacing: Style.space(6)
        ActionButton { text: "Sign in"; compact: true; enabled: root.service.canLogin; onClicked: { root.clearLogin(); root.loginTarget = Object.assign({}, row.modelData); code.forceActiveFocus() } }
        ActionButton { text: "Check"; compact: true; enabled: !root.service.busy; onClicked: root.service.shipAction(row.modelData, "ship-status", "") }
        ActionButton { text: "Log out"; compact: true; enabled: !root.service.busy; onClicked: root.service.shipAction(row.modelData, "ship-logout", "") }
        ActionButton { text: "Forget"; compact: true; enabled: !root.service.busy && row.modelData.status === "login_required"; onClicked: root.service.shipAction(row.modelData, "ship-forget", "") }
        ActionButton { text: row.theme.automatic ? "Pause theme" : "Sync theme"; compact: true; enabled: !root.service.busy && row.modelData.status !== "login_required"; onClicked: root.service.themeAuto(row.modelData, !row.theme.automatic) }
        ActionButton { text: "Publish now"; compact: true; enabled: !root.service.busy && row.modelData.status !== "login_required"; onClicked: root.service.publishTheme(row.modelData) }
      }
      Label { width: parent.width; text: (root.service.shipStatus[row.modelData.id] || row.modelData).status; secondary: true; caption: true }
      Label { width: parent.width; visible: !!row.theme.last_error; text: row.theme.last_error || ""; color: Design.danger }
    }
  }
  Label { visible: root.service && root.service.ships.length === 0; text: "No application sessions enrolled."; secondary: true }
  Column {
    width: parent.width; spacing: Style.space(8); visible: !!root.loginTarget
    Label { text: root.loginTarget ? "Sign in · " + root.loginTarget.name : "" }
    Field { id: code; objectName: "shipLoginCode"; width: parent.width; echoMode: TextInput.Password; maximumLength: 128; placeholderText: "Web login code"; enabled: root.service && root.service.canLogin }
    RowLayout {
      ActionButton { text: "Sign in"; primary: true; enabled: root.service && root.service.canLogin && code.text.length > 0; onClicked: { try { root.service.shipAction(root.loginTarget, "ship-login", code.text) } finally { root.clearLogin() } } }
      ActionButton { text: "Cancel"; onClicked: root.clearLogin() }
    }
  }
}
