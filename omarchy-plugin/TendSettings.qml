import QtQuick
import QtQuick.Controls as QQC
import qs.Commons
import qs.Ui

BorderSurface {
  id: root

  property var service: null
  property string mode: "app"
  property var list: null
  property var access: null
  property bool editable: false
  property bool mayInvite: false
  property var activities: []
  property var collaborationPolicy: ({ notifyAdded: true, notifyCompleted: true, notifyAssigned: true })
  property bool confirmDelete: false
  signal closeRequested()
  signal switchShipRequested()
  signal listDeleted()

  color: Qt.rgba(Color.popups.background.r, Color.popups.background.g, Color.popups.background.b, 1)
  borderSpec: Border.localOrSurfaceSpec("popups", "border", Color.popups.border, Color.menu.border, Style.normalBorderWidth)
  radius: Style.cornerRadius

  function load() {
    if (mode === "app") {
      if (!service) return
      badgeMode.currentIndex = Math.max(0, badgeMode.model.indexOf(service.preferences.badgeMode))
      allDayHour.value = Math.floor(service.preferences.allDayAlertMinute / 60)
      allDayMinute.value = service.preferences.allDayAlertMinute % 60
      allDayOverdue.checked = service.preferences.allDayOverdue
      return
    }
    if (!list) return
    listTitle.text = list.title || ""
    listColor.text = list.color || "#3b82f6"
    listSymbol.text = list.symbol || "list"
    notifyAdded.checked = collaborationPolicy.notifyAdded === true
    notifyCompleted.checked = collaborationPolicy.notifyCompleted === true
    notifyAssigned.checked = collaborationPolicy.notifyAssigned === true
    confirmDelete = false
  }

  function pinned() {
    return service && list && service.preferences.pinnedLists.indexOf(list.id) !== -1
  }

  function setDefaultList() {
    if (!service || !list) return
    service.setPreferences({
      defaultList: list.id,
      pinnedLists: service.preferences.pinnedLists,
      pinnedViews: service.preferences.pinnedViews,
      snoozePresets: service.preferences.snoozePresets
    })
  }

  function togglePinned() {
    if (!service || !list) return
    var values = service.preferences.pinnedLists.slice()
    var index = values.indexOf(list.id)
    if (index === -1) values.push(list.id)
    else values.splice(index, 1)
    service.setPreferences({
      defaultList: service.preferences.defaultList,
      pinnedLists: values,
      pinnedViews: service.preferences.pinnedViews,
      snoozePresets: service.preferences.snoozePresets
    })
  }

  function activityLabel(event) {
    var actor = String(event.actor || "")
    var action = String(event.action || "activity").replace(/-/g, " ")
    var title = String(event.reminderTitle || event.title || "")
    return actor + " · " + action + (title ? " · " + title : "")
  }

  onVisibleChanged: if (visible) load()
  onModeChanged: load()
  onListChanged: load()
  onCollaborationPolicyChanged: load()

  Connections {
    target: root.service
    function onPreferencesChanged() { if (root.mode === "app") root.load() }
  }

  QQC.ScrollView {
    anchors.fill: parent
    anchors.margins: Style.space(12)
    clip: true
    QQC.ScrollBar.horizontal.policy: QQC.ScrollBar.AlwaysOff

    Column {
      width: root.width - Style.space(26)
      spacing: Style.space(12)

      Row {
        width: parent.width

        Column {
          width: parent.width - closeButton.width
          spacing: Style.space(2)

          Text {
            width: parent.width
            text: root.mode === "list" ? "LIST SETTINGS" : "TEND SETTINGS"
            color: Color.popups.text
            font.family: Style.font.family
            font.pixelSize: Style.font.caption
            font.bold: true
          }

          Row {
            spacing: Style.space(6)
            Rectangle {
              width: Style.space(8)
              height: width
              radius: width / 2
              anchors.verticalCenter: parent.verticalCenter
              color: root.service && root.service.connectionState === "online" ? "#22c55e" : "#ef4444"
            }
            Text {
              text: root.mode === "list" && root.list ? root.list.title : (root.service && root.service.ship ? "~" + root.service.ship.replace(/^~/, "") : "Not connected")
              color: Qt.darker(Color.popups.text, 1.45)
              elide: Text.ElideRight
              font.family: Style.font.family
              font.pixelSize: Style.font.caption
            }
          }
        }

        TendButton { id: closeButton; text: "󰅖"; bordered: false; tooltipText: "Close settings"; onClicked: root.closeRequested() }
      }

      PanelSeparator { width: parent.width; foreground: Color.popups.text }

      Column {
        visible: root.mode === "app"
        width: parent.width
        spacing: Style.space(12)

        Text { text: "SHIP"; color: Color.popups.text; opacity: 0.68; font.family: Style.font.family; font.pixelSize: Style.font.caption; font.bold: true }

        TendButton {
          width: parent.width
          text: "Switch ship"
          Accessible.name: "Disconnect and switch Urbit ship"
          onClicked: root.switchShipRequested()
        }

        PanelSeparator { width: parent.width; foreground: Color.popups.text }
        Text { text: "REMINDER DEFAULTS"; color: Color.popups.text; opacity: 0.68; font.family: Style.font.family; font.pixelSize: Style.font.caption; font.bold: true }

        TendDropdown { id: badgeMode; width: parent.width; label: "Menubar badge"; model: ["today", "all", "assigned", "none"]; Accessible.name: "Menubar badge count" }

        Row {
          width: parent.width
          spacing: Style.space(8)
          TendNumber { id: allDayHour; width: (parent.width - parent.spacing) / 2; label: "All-day alert hour"; from: 0; to: 23; Accessible.name: "All-day reminder hour" }
          TendNumber { id: allDayMinute; width: (parent.width - parent.spacing) / 2; label: "Minute"; from: 0; to: 59; Accessible.name: "All-day reminder minute" }
        }

        TendCheckbox { id: allDayOverdue; width: parent.width; text: "Keep all-day reminders overdue"; onClicked: checked = !checked }

        TendButton {
          width: parent.width
          text: root.service && root.service.mutationPending ? "Saving…" : "Save reminder defaults"
          enabled: root.service && root.service.connectionState === "online" && !root.service.mutationPending
          onClicked: root.service.setReminderPolicy({
            badgeMode: badgeMode.currentText,
            allDayAlertMinute: allDayHour.value * 60 + allDayMinute.value,
            allDayOverdue: allDayOverdue.checked
          })
        }
      }

      Column {
        visible: root.mode === "list" && root.list !== null
        width: parent.width
        spacing: Style.space(12)

        Text { text: "APPEARANCE"; color: Color.popups.text; opacity: 0.68; font.family: Style.font.family; font.pixelSize: Style.font.caption; font.bold: true }
        TextField { id: listTitle; width: parent.width; placeholderText: "List title"; Accessible.name: "List title" }

        Row {
          width: parent.width
          spacing: Style.space(8)
          TextField { id: listColor; width: (parent.width - parent.spacing) * 0.62; placeholderText: "#3b82f6"; Accessible.name: "List color" }
          TextField { id: listSymbol; width: parent.width - listColor.width - parent.spacing; placeholderText: "Symbol"; Accessible.name: "List symbol or emoji" }
        }

        TendButton {
          width: parent.width
          text: "Save list appearance"
          enabled: root.editable && listTitle.text.trim() && listColor.text.trim() && listSymbol.text.trim() && root.service && root.service.connectionState === "online" && !root.service.mutationPending
          onClicked: root.service.updateList(root.list.id, listTitle.text, listColor.text, listSymbol.text, root.list.revision)
        }

        Row {
          width: parent.width
          spacing: Style.space(8)
          TendButton { width: (parent.width - parent.spacing) / 2; text: root.service && root.service.preferences.defaultList === root.list.id ? "Default list" : "Make default"; enabled: root.service && root.service.connectionState === "online" && !root.service.mutationPending; onClicked: root.setDefaultList() }
          TendButton { width: (parent.width - parent.spacing) / 2; text: root.pinned() ? "Unpin list" : "Pin list"; enabled: root.service && root.service.connectionState === "online" && !root.service.mutationPending; onClicked: root.togglePinned() }
        }

        PanelSeparator { width: parent.width; foreground: Color.popups.text }
        Text { text: "SHARING"; color: Color.popups.text; opacity: 0.68; font.family: Style.font.family; font.pixelSize: Style.font.caption; font.bold: true }

        Row {
          visible: root.mayInvite
          width: parent.width
          spacing: Style.space(8)
          TextField { id: inviteShip; width: parent.width - inviteButton.width - parent.spacing; placeholderText: "~sampel-palnet, moon, or comet"; Accessible.name: "Urbit ship to invite"; onAccepted: inviteButton.clicked() }
          TendButton { id: inviteButton; text: "Invite"; enabled: root.editable && inviteShip.text.trim() !== "" && root.service && root.service.connectionState === "online" && !root.service.mutationPending; onClicked: { if (root.service.inviteMember(root.list.id, inviteShip.text.trim(), false)) inviteShip.text = "" } }
        }

        Repeater {
          model: root.access ? root.access.members || [] : []
          delegate: Row {
            required property var modelData
            width: parent.width
            spacing: Style.space(8)
            Text { width: parent.width - removeMember.width - parent.spacing; anchors.verticalCenter: parent.verticalCenter; text: modelData.ship + (modelData.policy && modelData.policy.canInvite ? " (admin)" : " (editor)"); color: Color.popups.text; elide: Text.ElideRight; font.family: Style.font.family; font.pixelSize: Style.font.body }
            TendButton { id: removeMember; visible: root.access && root.access.owner; text: "Remove"; bordered: false; foreground: Color.urgent; enabled: root.editable && root.service && root.service.connectionState === "online" && !root.service.mutationPending; onClicked: root.service.removeMember(root.list.id, modelData.ship) }
          }
        }

        Repeater {
          model: root.access ? root.access.pending || [] : []
          delegate: Row {
            required property string modelData
            width: parent.width
            spacing: Style.space(8)
            Text { width: parent.width - revokeInvite.width - parent.spacing; anchors.verticalCenter: parent.verticalCenter; text: modelData + " (invited)"; color: Qt.darker(Color.popups.text, 1.4); elide: Text.ElideRight; font.family: Style.font.family; font.pixelSize: Style.font.body }
            TendButton { id: revokeInvite; visible: root.access && root.access.owner; text: "Revoke"; bordered: false; foreground: Color.urgent; enabled: root.editable && root.service && root.service.connectionState === "online" && !root.service.mutationPending; onClicked: root.service.removeMember(root.list.id, modelData) }
          }
        }

        PanelSeparator { width: parent.width; foreground: Color.popups.text }
        Text { text: "COLLABORATION NOTIFICATIONS"; color: Color.popups.text; opacity: 0.68; font.family: Style.font.family; font.pixelSize: Style.font.caption; font.bold: true }

        TendCheckbox { id: notifyAdded; text: "Items added"; onClicked: { checked = !checked; root.service.setCollaborationPolicy(root.list.id, checked, notifyCompleted.checked, notifyAssigned.checked) } }
        TendCheckbox { id: notifyCompleted; text: "Items completed"; onClicked: { checked = !checked; root.service.setCollaborationPolicy(root.list.id, notifyAdded.checked, checked, notifyAssigned.checked) } }
        TendCheckbox { id: notifyAssigned; text: "Assigned to me"; onClicked: { checked = !checked; root.service.setCollaborationPolicy(root.list.id, notifyAdded.checked, notifyCompleted.checked, checked) } }

        Text { visible: root.activities.length > 0; text: "RECENT ACTIVITY"; color: Color.popups.text; opacity: 0.68; font.family: Style.font.family; font.pixelSize: Style.font.caption; font.bold: true }
        Repeater {
          model: root.activities.slice(0, 20)
          delegate: Text { required property var modelData; width: parent.width; text: root.activityLabel(modelData); wrapMode: Text.Wrap; color: Qt.darker(Color.popups.text, 1.35); font.family: Style.font.family; font.pixelSize: Style.font.caption }
        }

        PanelSeparator { width: parent.width; foreground: Color.popups.text }

        TendButton {
          visible: root.access && root.access.owner
          width: parent.width
          text: root.confirmDelete ? "Confirm delete list" : "Delete list"
          foreground: Color.urgent
          accent: Color.urgent
          bordered: true
          enabled: root.editable && root.service && root.service.connectionState === "online" && !root.service.mutationPending
          onClicked: {
            if (!root.confirmDelete) { root.confirmDelete = true; return }
            if (root.service.deleteList(root.list.id, root.list.revision)) root.listDeleted()
          }
        }

        TendButton {
          visible: root.access && !root.access.owner
          width: parent.width
          text: "Leave shared list"
          foreground: Color.urgent
          accent: Color.urgent
          bordered: true
          enabled: root.service && root.service.connectionState === "online" && !root.service.mutationPending
          onClicked: if (root.service.leaveSharedList(root.list.id)) root.listDeleted()
        }
      }
    }
  }
}
