import QtQuick
import QtQuick.Controls as Controls
import QtQuick.Layouts
import Quickshell
import qs.Commons
import "Model.js" as Model

FloatingWindow {
  id: root
  required property var service
  property string selectedKey: ""
  property string page: "overview"
  property bool technical: false
  property var revealTarget: null
  readonly property alias maintenancePane: maintenance
  readonly property alias activityPane: activity
  readonly property alias setupPane: windowSetup
  readonly property var fleet: service ? Model.fleet(service.connections) : []
  readonly property var selected: fleet.filter(function(s) { return s.key === root.selectedKey })[0] || (!selectedKey && fleet.length ? fleet[0] : null)
  readonly property var connection: selected ? selected.connection : null
  readonly property var instance: selected ? selected.instance : null
  readonly property var status: instance ? Model.shipStatus(connection,instance,service.now,service.observing) : ({label:"Select a ship",tone:"muted",detail:""})
  readonly property var resources: Model.resourceView(connection,instance,service.now,service.observing)
  readonly property var sample: resources.sample
  readonly property var cpu: Model.cpuView(connection,instance,service.now,service.observing)
  readonly property var epochs: Model.epochDisplay(instance,service ? service.now : Date.now())
  readonly property var roll: Model.lastRoll(instance)
  readonly property bool shipPage: page === "overview" || page === "maintenance"
  title: "Omabit · Monitor & maintenance"
  color: Design.background
  implicitWidth: Style.space(1100)
  implicitHeight: Style.space(770)
  minimumSize: Qt.size(Style.space(780),Style.space(540))
  visible: false
  function present() { activation.activate() }
  WindowActivation {
    id: activation
    targetWindow: root
    onFailed: if (root.service) root.service.lastError = "Could not focus the monitor window. Try opening it again."
  }
  onVisibleChanged: if (!visible) maintenance.clearReview()
  onSelectedKeyChanged: { technical = false; bodyScroll.contentY = 0; revealTimer.stop(); revealTarget=null }
  onPageChanged: { bodyScroll.contentY = 0; revealTimer.stop(); revealTarget=null }
  function revealItem(item) {
    revealTarget=item; revealTimer.restart()
  }
  Timer {
    id: revealTimer
    interval: 50
    onTriggered: {
      // Let positioners finish the new section's layout before clamping scroll.
      var item=root.revealTarget
      if (item && item.visible) bodyScroll.contentY=Math.max(0,Math.min(item.mapToItem(body,0,0).y,bodyScroll.contentHeight-bodyScroll.height))
      root.revealTarget=null
    }
  }
  FocusScope {
    objectName: "monitor-content"
    anchors.fill: parent; focus: true
    Keys.onEscapePressed: root.visible = false
    Toast { service: root.service; anchors.horizontalCenter: parent.horizontalCenter; anchors.top: parent.top; anchors.topMargin: Style.space(86) }
    ColumnLayout {
      anchors.fill: parent; spacing: 0
      Rectangle {
        Layout.fillWidth: true; implicitHeight: Style.space(80); color: Design.surface
        RowLayout {
          anchors.fill: parent; anchors.leftMargin: Style.space(24); anchors.rightMargin: Style.space(24); spacing: Style.space(18)
          Column { spacing: Style.space(2)
            Label { text: "omabit"; font.pixelSize: Style.space(27); font.bold: true; font.letterSpacing: -1 }
            Label { objectName: "monitor-version"; text: "0.1.0-rc.2 / DEVELOPMENT"; caption: true; secondary: true }
          }
          Rectangle { width: 1; height: Style.space(28); color: Design.line }
          Column { Layout.fillWidth: true; spacing: Style.space(3)
            Label { text: "Monitor & maintenance"; font.bold: true }
            Label { text: "Your fleet, in focus"; caption: true; secondary: true }
          }
          ActionButton { objectName: "monitor-refresh"; text: "↻"; quiet: true; hint: "Refresh runner observations"; enabled: !root.service.busy; onClicked: root.service.refresh() }
          ActionButton { text: "×"; quiet: true; hint: "Close monitoring window"; onClicked: root.visible = false }
        }
      }
      Rectangle { Layout.fillWidth: true; height: 1; color: Design.line }
      RowLayout {
        Layout.fillWidth: true; Layout.fillHeight: true; spacing: 0
        Rectangle {
          Layout.fillHeight: true; Layout.preferredWidth: Style.space(235); color: Design.surface
          ColumnLayout {
            anchors.fill: parent; anchors.margins: Style.space(16); spacing: Style.space(16)
            Label { text: "FLEET"; caption: true; secondary: true; font.letterSpacing: 2 }
            Controls.ScrollView {
              Layout.fillWidth: true; Layout.fillHeight: true; clip: true
              Controls.ScrollBar.horizontal.policy: Controls.ScrollBar.AlwaysOff
              Column {
                width: parent.width; spacing: Style.space(16)
                Repeater {
                  model: root.service.connections
                  Column {
                    id: navRunner
                    required property var modelData
                    width: Style.space(203); spacing: Style.space(6)
                    Label { width: parent.width; text: Model.runnerTitle(navRunner.modelData,root.service.runnerPreferences).toUpperCase(); secondary: true; caption: true; elide: Text.ElideRight; wrapMode: Text.NoWrap }
                    Repeater {
                      model: Model.shipGroups(navRunner.modelData)
                      Column {
                        id: navGroup
                        required property var modelData
                        width: parent.width; spacing: Style.space(5)
                        Label { width: parent.width; text: navGroup.modelData.type; caption: true; secondary: true }
                        Repeater {
                          model: navGroup.modelData.ships
                      ActionButton {
                        id: navShip
                        required property var modelData
                        readonly property string key: navRunner.modelData.runner_id + "/" + modelData.id
                        readonly property var state: Model.shipStatus(navRunner.modelData,modelData,root.service.now,root.service.observing)
                        readonly property bool selectedInView: root.shipPage && !!root.selected && root.selected.key === key
                        width: parent.width; text: Model.instanceTitle(modelData); quiet: true
                        implicitHeight: navLabels.implicitHeight + topPadding + bottomPadding
                        hint: (modelData.ship || "") + " · " + state.detail
                        contentItem: RowLayout {
                          spacing: Style.space(8)
                          StatusBadge { dotOnly: true; tone: navShip.state.tone; text: navShip.state.label }
                          Column {
                            id: navLabels
                            Layout.fillWidth: true; Layout.minimumWidth: 0; spacing: Style.space(3)
                            ShipName { objectName: "nav-ship-" + navShip.modelData.id; width: parent.width; instance: navShip.modelData; compact: true; hintEnabled: false; font.bold: navShip.selectedInView }
                            Label { width: parent.width; text: navShip.modelData.id; caption: true; secondary: true; elide: Text.ElideRight; wrapMode: Text.NoWrap }
                          }
                          Label { objectName: "fleet-marker-" + navShip.modelData.id; text: navShip.selectedInView ? "›" : ""; color: Design.accent }
                        }
                        onClicked: { root.selectedKey = key; if (root.page === "activity" || root.page === "setup") root.page = "overview" }
                      }
                        }
                      }
                    }
                  }
                }
              }
            }
            Rectangle { Layout.fillWidth: true; height: 1; color: Design.line }
            ActionButton { Layout.fillWidth: true; text: "Activity feed"; primary: root.page === "activity"; onClicked: { root.page = "activity"; activity.runnerFilter = ""; activity.shipFilter = "" } }
            ActionButton { objectName: "monitor-setup"; Layout.fillWidth: true; text: "Setup"; primary: root.page === "setup"; onClicked: root.page = "setup" }
            Label { text: root.fleet.length + " ships · " + root.service.connections.length + " runners"; secondary: true; caption: true }
          }
        }
        Rectangle { Layout.fillHeight: true; width: 1; color: Design.line }
        Flickable {
          id: bodyScroll
          objectName: "monitor-scroll"
          Layout.fillWidth: true; Layout.fillHeight: true
          contentWidth: width; contentHeight: body.implicitHeight + Style.space(48)
          clip: true; boundsBehavior: Flickable.StopAtBounds
          Controls.ScrollBar.vertical: Controls.ScrollBar {}
          Column {
            id: body
            x: (bodyScroll.width - width) / 2; y: Style.space(24); width: Math.min(bodyScroll.width - Style.space(48), Style.space(1150)); spacing: Style.space(20)
            Label { width: parent.width; visible: text !== ""; text: root.service.feedError || (root.page!=="setup" ? root.service.lastError : ""); color: Design.danger }
            Label { width: parent.width; visible: root.service.busy; text: root.service.settingUp ? "Setting up this computer…" : "Submitting " + root.service.currentAction + "…"; color: Design.accent }
            ActionButton { width: parent.width; visible: Model.discoveredPiers(root.service.connections).length>0 && root.page!=="setup"; text: Model.discoveredPiers(root.service.connections).length+" discovered piers need enrollment or location review →"; onClicked: {root.page="setup";windowSetup.mode="storage"} }
            Column {
              width: parent.width; spacing: Style.space(20); visible: root.page !== "activity" && root.page !== "setup" && !!root.instance
              Column {
                width: parent.width
                spacing: Style.space(10)
                ShipContext { objectName: "monitor-ship-context"; width: parent.width; host: root.connection ? Model.runnerTitle(root.connection,root.service.runnerPreferences) : ""; instanceName: root.instance ? root.instance.id : ""; status: root.status.label; tone: root.status.tone; hint: root.status.detail }
                ShipName { objectName: "monitor-ship-name"; width: parent.width; instance: root.instance; font.pixelSize: Style.space(30); font.bold: true }
                Flow {
                  width: parent.width; spacing: Style.space(8)
                ActionButton { text: "▶ Start"; enabled: root.service.observing && !root.service.busy && Model.canOperate(root.connection,root.instance,"start",root.service.now); onClicked: root.service.operate(root.connection,root.instance,"start") }
                ActionButton { text: "■ Stop"; enabled: root.service.observing && !root.service.busy && Model.canOperate(root.connection,root.instance,"stop",root.service.now); onClicked: root.service.operate(root.connection,root.instance,"stop") }
                CopyCodeButton { objectName: "monitor-copy-code"; service: root.service; runnerID: root.connection ? root.connection.runner_id : ""; connectionName: root.connection ? root.connection.name : ""; instanceID: root.instance ? root.instance.id : "" }
                ActionButton { objectName: "monitor-open-ship"; text: "Open ship ↗"; hint: Model.shipURL(root.connection,root.instance,root.service.registrations,root.service.runnerPreferences) || "Set the browser host in Setup → Manage runners"; enabled: !!root.connection && root.service.observing && Model.available(root.connection,root.service.now) && !!Model.shipURL(root.connection,root.instance,root.service.registrations,root.service.runnerPreferences); onClicked: root.service.openShip(root.connection,root.instance) }
                }
                Label { width: parent.width; caption: true; secondary: true; text: Model.codeUnavailable(root.connection,root.instance,root.service.now,root.service.observing); visible: text !== "" }
              }
              RowLayout {
                ActionButton { text: "Overview"; primary: root.page === "overview"; onClicked: root.page = "overview" }
                ActionButton { text: "Maintenance"; primary: root.page === "maintenance"; onClicked: root.page = "maintenance" }
                ActionButton { text: "Ship activity ↗"; quiet: true; onClicked: { activity.runnerFilter = root.connection.runner_id; activity.shipFilter = root.selected.key; root.page = "activity" } }
              }
              Column {
                width: parent.width; spacing: Style.space(16); visible: root.page === "overview"
                RuntimePane { width:parent.width;service:root.service;connection:root.connection;instance:root.instance;active:root.visible&&root.page==="overview" }
                GridLayout {
                  width: parent.width; columns: width < Style.space(590) ? 2 : 3; columnSpacing: Style.space(10); rowSpacing: Style.space(10)
                  Metric { objectName: "container-memory"; Layout.fillWidth: true; label: "CONTAINER RAM"; value: root.resources.value; note: root.resources.note; ratio: root.sample ? root.sample.value / root.sample.limit : -1 }
                  Metric { Layout.fillWidth: true; label: "RETAINED EPOCHS"; value: root.epochs.entries.length ? String(root.epochs.entries.length) : "—"; note: root.epochs.label; ratio: -1 }
                  Metric { Layout.fillWidth: true; label: "Task Records"; value: root.resources.inactive ? "N/A" : root.sample && root.instance.resources.pids_current !== undefined ? String(root.instance.resources.pids_current) : "—"; note: root.resources.inactive ? root.resources.note : "Processes + threads"; ratio: -1 }
                }
                GridLayout {
                  width: parent.width; columns: width < Style.space(560) ? 1 : 2; columnSpacing: Style.space(10); rowSpacing: Style.space(10)
                  Card {
                    Layout.fillWidth: true; implicitHeight: rollSummary.implicitHeight + Style.space(28)
                    Column { id: rollSummary; anchors.left: parent.left; anchors.right: parent.right; anchors.top: parent.top; anchors.margins: Style.space(14); spacing: Style.space(8)
                      Label { text: "Last roll"; font.bold: true }
                      Label { width: parent.width; text: root.roll ? Model.humanTime(root.roll.finished_at,root.service.now) : "No completed roll recorded"; color: root.roll ? Design.accent : Design.muted }
                      Label { width: parent.width; text: root.roll ? "Created epoch " + root.roll.created_epoch : "Rolling creates a new history segment and snapshot."; caption: true; secondary: true }
                    }
                  }
                  Card {
                    Layout.fillWidth: true; implicitHeight: pruneSummary.implicitHeight + Style.space(28)
                    Column { id: pruneSummary; anchors.left: parent.left; anchors.right: parent.right; anchors.top: parent.top; anchors.margins: Style.space(14); spacing: Style.space(8)
                      Label { text: "Last prune"; font.bold: true }
                      Label { width: parent.width; text: root.instance && root.instance.pruning ? root.instance.pruning.uncertain || !root.instance.pruning.finished_at ? "Incomplete · review required" : Model.humanTime(root.instance.pruning.finished_at,root.service.now) : "No history removal recorded"; color: root.instance && root.instance.pruning ? Design.accent : Design.muted }
                      Label { width: parent.width; text: root.instance && root.instance.pruning ? "Removed " + (root.instance.pruning.removed_epochs || []).length + " old epochs through " + root.instance.pruning.cutoff : "Old history is removed only after explicit review."; caption: true; secondary: true }
                    }
                  }
                }
                RoutineCard { width: parent.width; visible: root.instance && !!root.instance.schedule; service: root.service; connection: root.connection; instance: root.instance; onEditRequested: { root.page="maintenance"; maintenance.section="routine"; Qt.callLater(function(){maintenance.routinePane.edit()}) } }
                Card {
                  objectName: "cpu-chart-card"
                  width: parent.width; implicitHeight: cpuBody.implicitHeight + Style.space(32)
                  Column {
                    id: cpuBody
                    anchors.left: parent.left; anchors.right: parent.right; anchors.top: parent.top; anchors.margins: Style.space(16); spacing: Style.space(10)
                    RowLayout { width: parent.width
                      Label { text: "Container CPU"; font.bold: true; Layout.fillWidth: true }
                      StatusBadge { objectName: "container-cpu-value"; text: root.cpu.value; tone: root.cpu.sample ? "good" : "muted" }
                    }
                    ResourceChart { objectName: "cpu-chart"; width: parent.width; metric: "cpu"; samples: root.selected ? root.service.cpuHistory[root.selected.key] || [] : []; emptyText: root.cpu.inactive ? root.cpu.note : "Collecting CPU samples…" }
                    Label { width: parent.width; text: root.cpu.note; secondary: true; caption: true }
                  }
                }
                Card {
                  objectName: "memory-chart-card"
                  width: parent.width; implicitHeight: ram.implicitHeight + Style.space(32)
                  Column {
                    id: ram
                    anchors.left: parent.left; anchors.right: parent.right; anchors.top: parent.top; anchors.margins: Style.space(16); spacing: Style.space(10)
                    RowLayout { width: parent.width
                      Label { text: "Container Memory"; font.bold: true; Layout.fillWidth: true }
                      StatusBadge { text: root.resources.inactive ? "Stopped · historical chart" : root.sample ? "Live" : "No fresh sample"; tone: root.sample ? "good" : "muted" }
                    }
                    ResourceChart { objectName: "memory-chart"; width: parent.width; samples: root.selected ? root.service.memoryHistory[root.selected.key] || [] : []; emptyText: root.resources.inactive ? root.resources.note : "Collecting memory samples…" }
                    Label { width: parent.width; text: "Container RAM includes cache and kernel memory. This is not loom utilization."; secondary: true; caption: true }
                  }
                }
                Card {
                  width: parent.width; implicitHeight: epochBody.implicitHeight + Style.space(32)
                  Column {
                    id: epochBody
                    anchors.left: parent.left; anchors.right: parent.right; anchors.top: parent.top; anchors.margins: Style.space(16); spacing: Style.space(12)
                    RowLayout { width: parent.width
                      Label { text: "Epoch history · oldest to newest"; font.bold: true; Layout.fillWidth: true }
                      ActionButton { text: "Manage retention →"; compact: true; quiet: true; onClicked: root.page = "maintenance" }
                    }
                    Flow {
                      width: parent.width; spacing: Style.space(6)
                      Repeater {
                        model: root.epochs.entries
                        Rectangle {
                          required property var modelData
                          width: epochBody.width < Style.space(480) ? epochBody.width : (epochBody.width-Style.space(6))/2
                          height: epochTile.implicitHeight + Style.space(24); radius: Style.space(6)
                          color: Design.fade(modelData.protected ? Design.success : Design.accent,0.09)
                          border.color: Design.fade(modelData.protected ? Design.success : Design.accent,0.25)
                          Column { id: epochTile; anchors.centerIn: parent; width: parent.width - Style.space(24); spacing: Style.space(6)
                            Label { width: parent.width; text: "Epoch " + modelData.name; font.bold: true; elide: Text.ElideMiddle; wrapMode: Text.NoWrap }
                            Label { width: parent.width; text: Model.epochBoundary(modelData.name); caption: true; secondary: true }
                            Label { width: parent.width; text: Model.epochDate(root.instance,modelData.name,root.service.now); caption: true; secondary: true }
                            Label { width: parent.width; text: (modelData.protected ? "PROTECTED" : "RETAINED") + (root.epochs.measured ? " · " + Model.bytesText(modelData.logical_bytes) : ""); caption: true; color: modelData.protected ? Design.success : Design.muted }
                          }
                        }
                      }
                    }
                    Label { width: parent.width; visible: root.epochs.entries.length === 0; text: "Inspect a stopped pier to see its epoch inventory."; secondary: true }
                    Label { width: parent.width; text: root.epochs.time ? root.epochs.label + " · " + Model.humanTime(root.epochs.time,root.service.now) + (root.epochs.measured ? "" : "\nHistorical result; inspect the stopped ship for a fresh inventory.") : "Newest epochs are protected by retention policy."; secondary: true; caption: true }
                  }
                }
                Card {
                  width: parent.width; implicitHeight: network.implicitHeight + Style.space(32)
                  Column {
                    id: network
                    anchors.left: parent.left; anchors.right: parent.right; anchors.top: parent.top; anchors.margins: Style.space(16); spacing: Style.space(10)
                    Label { text: "Connectivity & runtime"; font.bold: true }
                    Label { width: parent.width; text: root.instance ? "Host HTTP  " + root.instance.config.http_port + (root.instance.mode === "fake" ? "     Ames: container loopback" : (root.instance.ames_published ? "     Ames UDP  " : "     Guest Ames  ") + root.instance.config.ames_port) + "     Loom  " + root.instance.config.loom : "" }
                    Label { width: parent.width; visible: !!root.instance; text: root.instance && root.connection && root.instance.pier_path ? "Pier: "+Model.absolutePierPath(root.connection,root.instance.pier_path) : "Pier location unavailable"; secondary: true; caption: true }
                    Label { width: parent.width; text: root.instance ? "Desired / observed: " + root.instance.desired_state + " / " + root.instance.observed_state + "\nConfiguration / applied: " + root.instance.config_revision + " / " + root.instance.applied_revision : ""; secondary: true; caption: true }
                    Label { width: parent.width; text: root.instance && root.instance.ames_published ? "Ames UDP is published on host port "+root.instance.config.ames_port+". HTTP is loopback-only." : root.instance && root.instance.mode !== "fake" ? "Outbound Ames is enabled; the guest UDP port is not published on the host." : "Development fake: Ames is restricted to this container's loopback, with no host UDP publication. Outbound HTTP remains available. A second isolated fake with the same name is allowed; sharing the same pier is not."; secondary: true; caption: true }
                    ShipPortsPane { width: parent.width; service: root.service; connection: root.connection; instance: root.instance }
                    HostMovePane { width: parent.width; service: root.service; connection: root.connection; instance: root.instance }
                    DeletePierPane { objectName: "overview-delete-pier"; width: parent.width; service: root.service; connection: root.connection; instance: root.instance }
                  }
                }
                ActionButton { text: root.technical ? "Technical details −" : "Technical details +"; quiet: true; onClicked: root.technical = !root.technical }
                Label { width: parent.width; visible: root.technical && !!root.roll; text: Model.maintenanceText(root.roll); secondary: true; caption: true }
                Label { width: parent.width; visible: root.technical && root.instance && !!root.instance.maintenance && root.instance.maintenance.kind !== "roll"; text: Model.maintenanceText(root.instance ? root.instance.maintenance : null); secondary: true; caption: true }
                Label { width: parent.width; visible: root.technical && root.instance && !!root.instance.pruning; text: Model.pruningText(root.instance ? root.instance.pruning : null); secondary: true; caption: true }
                Label { width: parent.width; visible: root.technical && root.instance && !!root.instance.import; secondary: true; caption: true; text: root.instance && root.instance.import ? "Import: " + (root.instance.import.uncertain ? "incomplete / in progress" : "complete") + " · " + root.instance.import.copied_files + "/" + root.instance.import.expected_files + " files\n" + Model.bytesText(root.instance.import.copied_bytes) + " / " + Model.bytesText(root.instance.import.expected_bytes) + " · source " + root.instance.import.source_run_id : "" }
                Label { width: parent.width; visible: root.technical; caption: true; secondary: true; text: root.instance && root.connection ? "Runner: " + root.connection.runner_id + "\nAccess: " + root.connection.access_role + "\nEndpoint reported by runner: " + (root.instance.endpoint || "none") + "\n" + Model.resourceText(root.instance,root.service.now) + (root.instance.active_operation_id ? "\nOperation: " + root.instance.active_operation_id : "") : "" }
                Label { width: parent.width; visible: root.technical && root.service.lastResult !== ""; text: "Last submitted result: " + root.service.lastResult; caption: true; secondary: true }
              }
              MaintenancePane { id: maintenance; width: parent.width; service: root.service; connection: root.connection; instance: root.instance; visible: root.page === "maintenance"; onRevealRequested: function(item) { root.revealItem(item) } }
            }
            Label { width: parent.width; visible: root.page !== "activity" && root.page !== "setup" && !root.instance; text: "Select a ship from the fleet to inspect its runtime. Your selection will not silently switch to a replacement runner."; secondary: true }
            ActivityPane { id: activity; width: parent.width; service: root.service; visible: root.page === "activity" }
            Column {
              width: parent.width; spacing: Style.space(16); visible: root.page === "setup"
              Label { text: "Setup"; font.pixelSize: Style.space(26); font.bold: true }
              SetupPane { id: windowSetup; width: parent.width; service: root.service; active: root.visible && root.page === "setup"; onShowFleet: root.page="overview" }
            }
            Label { width: parent.width; visible: root.technical && root.service.requestIdentity !== ""; text: "Last request: " + root.service.requestIdentity; secondary: true; caption: true }
          }
        }
      }
    }
  }
  component Metric: Card {
    property string label
    property string value
    property string note
    property real ratio: -1
    implicitHeight: Style.space(118)
    Column {
      anchors.fill: parent; anchors.margins: Style.space(14); spacing: Style.space(8)
      Label { text: parent.parent.label; secondary: true; caption: true; font.letterSpacing: 0.6 }
      Label { width: parent.width; text: parent.parent.value; font.pixelSize: Style.space(23); font.bold: true; elide: Text.ElideRight; wrapMode: Text.NoWrap }
      Rectangle {
        width: parent.width; height: Style.space(4); radius: 2; color: Design.line; visible: parent.parent.ratio >= 0
        Rectangle { width: parent.width * Math.min(1,Math.max(0,parent.parent.parent.ratio)); height: parent.height; radius: 2; color: Design.accent }
      }
      Label { width: parent.width; text: parent.parent.note; caption: true; secondary: true; elide: Text.ElideRight; wrapMode: Text.NoWrap }
    }
  }
}
