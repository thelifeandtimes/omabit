import QtQuick
import QtQuick.Controls as Controls
import QtQuick.Layouts
import QtQuick.Effects
import Quickshell.Io
import qs.Commons
import qs.Ui as Ui
import "Model.js" as Model

Ui.Panel {
  id: root
  moduleName: "thelifeandtimes.omabit"
  ipcTarget: "thelifeandtimes.omabit"
  property var service: bar && bar.shell && typeof bar.shell.serviceFor === "function" ? bar.shell.serviceFor(moduleName) : null
  property string pane: "fleet"
  readonly property var connections: service ? service.connections : []
  implicitWidth: button.implicitWidth
  implicitHeight: button.implicitHeight
  onOpenedChanged: if (opened && service && !service.observing) service.refresh()
  function syncSettings() {
    if (service && service.settings !== settings) { service.settings = settings; service.refresh() }
  }
  onServiceChanged: Qt.callLater(syncSettings)
  onSettingsChanged: Qt.callLater(syncSettings)
  function monitor(connection, instance) { root.close(); service.openMonitor(connection, instance, "overview") }
  IpcHandler {
    enabled: root.manageIpc
    target: "thelifeandtimes.omabit.setup"
    function open(): void { root.pane = "setup"; root.open() }
  }

  Ui.BarIconButton {
    id: button
    anchors.fill: parent; bar: root.bar; text: ""; active: root.opened
    activeColor: Qt.tint(foreground, Design.fade(Color.background, 0.25))
    iconComponent: Component {
      Item {
        anchors.centerIn: parent; width: Math.min(parent.width,parent.height); height: width
        Image {
          id: mark
          anchors.fill: parent; source: "omabit.svg"; visible: false
          sourceSize: Qt.size(width*2,height*2); fillMode: Image.PreserveAspectFit
        }
        MultiEffect { anchors.fill: parent; source: mark; colorization: 1; colorizationColor: button.active ? button.activeColor : button.foreground }
      }
    }
    tooltipText: "Omabit 0.1.0-rc.2 · " + Model.fleet(root.connections).length + " instances"
    onPressed: function(code) { if (code === Qt.RightButton) { if (root.service) root.service.refresh() } else root.toggle() }
  }
  Ui.KeyboardPanel {
    owner: root; anchorItem: button; bar: root.bar; open: root.opened; focusTarget: form
    contentWidth: fittedContentWidth(Style.space(590))
    contentHeight: fittedContentHeight(content.implicitHeight, Style.space(690))
    FocusScope {
      id: form
      anchors.fill: parent
      Keys.onEscapePressed: root.close()
      Toast { service: root.service; anchors.horizontalCenter: parent.horizontalCenter; anchors.bottom: parent.bottom; anchors.bottomMargin: Style.space(12) }
      Flickable {
        id: scroll
        anchors.fill: parent; contentWidth: width; contentHeight: content.implicitHeight
        clip: true; boundsBehavior: Flickable.StopAtBounds
        Controls.ScrollBar.vertical: Controls.ScrollBar {}
        Column {
          id: content
          width: scroll.width; spacing: Style.space(16)
          RowLayout {
            width: parent.width
            Column {
              Layout.fillWidth: true; spacing: Style.space(2)
              Label { text: "omabit"; font.pixelSize: Style.space(23); font.bold: true; font.letterSpacing: -1 }
              Label { text: root.pane === "fleet" ? "YOUR FLEET, AT A GLANCE" : "CONNECTIONS & PROVISIONING"; caption: true; secondary: true; font.letterSpacing: 1 }
            }
            ActionButton { compact: true; quiet: true; text: "↻"; hint: "Refresh runner observations"; enabled: root.service && !root.service.busy; onClicked: root.service.refresh() }
            ActionButton { compact: true; text: root.pane === "fleet" ? "Setup" : "← Fleet"; onClicked: { root.pane = root.pane === "fleet" ? "setup" : "fleet"; scroll.contentY = 0 } }
          }
          Label { width: parent.width; visible: text !== ""; color: Design.danger; text: root.service ? root.service.feedError || (root.pane!=="setup" ? root.service.lastError : "") : "Collector service unavailable" }
          Label { width: parent.width; visible: root.service && root.service.busy; secondary: true; caption: true; text: root.service ? root.service.settingUp ? "Setting up this computer…" : "Submitting " + root.service.currentAction + "…" : "" }
          ActionButton { width: parent.width; visible: Model.discoveredPiers(root.connections).length>0; text: Model.discoveredPiers(root.connections).length+" discovered piers need enrollment or location review →"; onClicked: {root.pane="setup";setup.mode="storage";scroll.contentY=0} }
          Column {
            width: parent.width; spacing: Style.space(12); visible: root.pane === "fleet"
            Repeater {
              model: root.connections
              Card {
                id: runner
                required property var modelData
                readonly property bool online: root.service && root.service.observing && Model.available(modelData, root.service.now)
                width: parent.width; implicitHeight: runnerBody.implicitHeight + Style.space(24)
                Column {
                  id: runnerBody
                  anchors.left: parent.left; anchors.right: parent.right; anchors.top: parent.top
                  anchors.margins: Style.space(12); spacing: Style.space(10)
                  RowLayout {
                    width: parent.width; spacing: Style.space(8)
                    StatusBadge { dotOnly: true; tone: runner.online ? "good" : "warn"; text: runner.online ? "Connected" : "Unavailable"; hint: runner.modelData.runner_id }
                    Label { text: Model.runnerTitle(runner.modelData, root.service.runnerPreferences); font.bold: true; Layout.maximumWidth: runner.width * 0.4; elide: Text.ElideRight; wrapMode: Text.NoWrap }
                    Label { Layout.fillWidth: true; horizontalAlignment: Text.AlignRight; caption: true; secondary: true; elide: Text.ElideMiddle; wrapMode: Text.NoWrap; text: Model.runnerAddress(runner.modelData, root.service.registrations, root.service.runnerPreferences) }
                  }
                  Rectangle { width: parent.width; height: 1; color: Design.line }
                  Repeater {
                    model: Model.shipGroups(runner.modelData)
                    Column {
                      id: shipGroup
                      required property var modelData
                      width: runnerBody.width; spacing: Style.space(9)
                      Label { text: shipGroup.modelData.type.toUpperCase() + " · " + shipGroup.modelData.ships.length; caption: true; secondary: true; font.letterSpacing: 1 }
                      Repeater {
                        model: shipGroup.modelData.ships
                    RowLayout {
                      id: ship
                      required property var modelData
                      readonly property var status: Model.shipStatus(runner.modelData, modelData, root.service.now, root.service.observing)
                      width: runnerBody.width; spacing: Style.space(8)
                      StatusBadge { dotOnly: true; text: ship.status.label; tone: ship.status.tone; hint: ship.status.detail }
                      Column {
                        Layout.fillWidth: true; Layout.minimumWidth: Style.space(100); spacing: Style.space(2)
                        ShipName { width: parent.width; instance: ship.modelData; font.bold: true; compact: true }
                        Label { width: parent.width; text: ship.modelData.id + " · " + ship.status.label; secondary: true; caption: true; elide: Text.ElideRight; wrapMode: Text.NoWrap }
                      }
                      Label { text: ship.modelData.mode === "fake" ? "AMES\nLoopback" : (ship.modelData.ames_published ? "AMES UDP\n" : "GUEST AMES\n") + ship.modelData.config.ames_port; caption: true; secondary: true; horizontalAlignment: Text.AlignRight }
                      CopyCodeButton { objectName: "fleet-copy-code-"+ship.modelData.id; compact: true; service: root.service; runnerID: runner.modelData.runner_id; connectionName: runner.modelData.name; instanceID: ship.modelData.id }
                      ActionButton { compact: true; text: ":" + ship.modelData.config.http_port + " ↗"; hint: Model.shipURL(runner.modelData, ship.modelData, root.service.registrations, root.service.runnerPreferences) || "Set the browser host in Setup → Manage runner"; enabled: runner.online && !!Model.shipURL(runner.modelData, ship.modelData, root.service.registrations, root.service.runnerPreferences); onClicked: root.service.openShip(runner.modelData, ship.modelData) }
                      ActionButton { objectName: "start-" + ship.modelData.id; compact: true; quiet: true; text: "▶"; hint: "Start " + ship.modelData.id; enabled: runner.online && !root.service.busy && Model.canOperate(runner.modelData, ship.modelData, "start", root.service.now); onClicked: root.service.operate(runner.modelData, ship.modelData, "start") }
                      ActionButton { objectName: "stop-" + ship.modelData.id; compact: true; quiet: true; text: "■"; hint: "Stop " + ship.modelData.id; enabled: runner.online && !root.service.busy && Model.canOperate(runner.modelData, ship.modelData, "stop", root.service.now); onClicked: root.service.operate(runner.modelData, ship.modelData, "stop") }
                      ActionButton { compact: true; quiet: true; text: "›"; hint: "Monitor & maintain " + ship.modelData.id; onClicked: root.monitor(runner.modelData, ship.modelData) }
                    }
                      }
                    }
                  }
                  Label { visible: Model.instances(runner.modelData).length === 0; text: runner.online ? "No ships yet. Add your first ship in Setup." : "No cached ship inventory."; secondary: true; caption: true }
                  Label { width: parent.width; visible: runner.modelData.mode.indexOf("development") === 0 && Model.instances(runner.modelData).length > 0; text: "HTTP binds host loopback; remote access is separately routed. Fake Ames is confined to each container's loopback; outbound HTTP is allowed. Independent fakes may share an Urbit name."; secondary: true; caption: true }
                }
              }
            }
            Label { width: parent.width; visible: root.connections.length === 0; text: "Your fleet starts here. Add a prepared runner in Setup to see its ships."; secondary: true }
            RowLayout {
              width: parent.width
              Label { Layout.fillWidth: true; text: root.connections.length + " runners · " + Model.fleet(root.connections).length + " instances"; secondary: true; caption: true }
              ActionButton { objectName: "open-monitor"; primary: true; text: "Monitor & maintenance  ↗"; enabled: !!root.service; onClicked: root.monitor(null, null) }
            }
          }
          SetupPane { id: setup; width: parent.width; visible: root.pane === "setup"; service: root.service; active: root.opened && visible; onShowFleet: { root.pane="fleet";scroll.contentY=0 } }
        }
      }
    }
  }
}
