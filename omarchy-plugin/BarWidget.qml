import QtQuick
import QtQuick.Layouts
import qs.Commons
import qs.Ui
import "TendModel.js" as TendModel

Panel {
  id: root

  moduleName: "io.omabit.tend"
  ipcTarget: "omabit-tend-widget"
  manageIpc: true

  readonly property var tendService: bar && bar.shell ? bar.shell.serviceFor(moduleName) : null
  readonly property string state: tendService ? tendService.connectionState : "disconnected"
  property string selectedView: "assigned"
  readonly property var smartViewKeys: ["today", "scheduled", "all", "flagged", "assigned", "completed"]
  readonly property var smartViewLabels: ["Today", "Scheduled", "All", "Flagged", "Assigned to me", "Completed"]
  readonly property var visibleReminders: TendModel.queryReminders(tendService ? tendService.lists : [], {
    view: selectedView,
    ship: tendService ? tendService.ship : "",
    sort: "priority",
    allDayOverdue: tendService ? tendService.preferences.allDayOverdue : true
  })
  readonly property int taskCount: visibleReminders.length
  readonly property var destinationLists: tendService
    ? TendModel.orderedLists(tendService.lists, tendService.preferences.pinnedLists, tendService.localSettings.listOrder)
    : []
  readonly property var selectedDestination: destinationList.currentIndex >= 0 && destinationList.currentIndex < destinationLists.length
    ? destinationLists[destinationList.currentIndex]
    : null
  readonly property var assigneeOptions: assigneeOptionsForList(selectedDestination)
  readonly property color dim: Qt.rgba(barForeground.r, barForeground.g, barForeground.b, 0.6)

  function normalizedShip(value) {
    var name = String(value || "").replace(/^~/, "")
    return name ? "~" + name : ""
  }

  function assigneeOptionsForList(list) {
    var choices = [{ value: "", label: "Unassigned", description: "No owner" }]
    if (!tendService || !list) return choices
    var access = tendService.accessForList(list.id)
    var seen = ({})
    function append(ship, suffix, description) {
      var normalized = root.normalizedShip(ship)
      if (!normalized || seen[normalized]) return
      seen[normalized] = true
      choices.push({
        value: normalized,
        label: normalized + (suffix ? " · " + suffix : ""),
        description: description || ""
      })
    }
    append(tendService.ship, "you", "Default assignee")
    if (!access) return choices
    append(access.host, "owner", "List owner")
    ;(access.members || []).forEach(function(member) { append(member.ship, "member", "Can edit this list") })
    ;(access.pending || []).forEach(function(ship) { append(ship, "invited", "Invitation not yet accepted") })
    return choices
  }

  function selectSelfAssignee() {
    var self = normalizedShip(tendService ? tendService.ship : "")
    assigneePicker.value = self
  }

  function dueLabel(reminder) {
    if (!reminder || !reminder.schedule) return "No due date"
    var value = TendModel.scheduleInputValue(reminder.schedule)
    return value ? value.replace("T", " ") : "No due date"
  }

  function priorityGlyph(priority) {
    if (priority === "high") return "!!!"
    if (priority === "medium") return "!!"
    if (priority === "low") return "!"
    return ""
  }

  function priorityColor(priority) {
    if (priority === "high") return Color.urgent
    if (priority === "medium") return "#e7b45f"
    if (priority === "low") return Color.accent
    return dim
  }

  function openFullPanel(listId, reminderId) {
    if (!bar || !bar.shell || typeof bar.shell.summon !== "function") return
    close()
    var payload = JSON.stringify({ listId: Number(listId || 0), reminderId: Number(reminderId || 0) })
    Qt.callLater(function() { root.bar.shell.summon(root.moduleName, payload) })
  }

  function addReminder() {
    if (!tendService || destinationList.currentIndex < 0 || !quickAdd.text.trim()) return
    var list = destinationLists[destinationList.currentIndex]
    if (!list) return
    if (tendService.addReminderWithDetails(list.id, quickAdd.text.trim(), list.revision, {
      tags: [],
      due: quickDue.text.trim(),
      allDay: quickDue.text.trim().length === 10,
      timezone: tendService.localTimezone,
      assignee: assigneePicker.value || normalizedShip(tendService.ship)
    })) {
      quickAdd.text = ""
      quickDue.text = ""
    }
  }

  visible: true
  implicitWidth: widgetButton.implicitWidth
  implicitHeight: widgetButton.implicitHeight

  onOpenedChanged: if (opened) Qt.callLater(function() {
    root.selectSelfAssignee()
    quickAdd.forceActiveFocus()
  })
  onSelectedDestinationChanged: Qt.callLater(root.selectSelfAssignee)

  WidgetButton {
    id: widgetButton
    anchors.fill: parent
    bar: root.bar
    labelVisible: false
    hasVisualContent: true
    fixedWidth: vertical ? barSize : widgetContents.implicitWidth + Style.space(16)
    fixedHeight: vertical ? barSize : -1
    tooltipText: "Tend · " + taskCount + " in " + selectedView + " · " + state
    Accessible.role: Accessible.Button
    Accessible.name: "Open Tend reminders; " + taskCount + " in " + selectedView + "; " + state
    onPressed: function(buttonCode) {
      if (buttonCode === Qt.LeftButton) root.toggle()
    }

    Row {
      id: widgetContents
      anchors.centerIn: parent
      spacing: Style.space(5)

      Text {
        anchors.verticalCenter: parent.verticalCenter
        text: root.state === "online" ? "󰄬" : (root.state === "checking" ? "󰔟" : "󰅖")
        textFormat: Text.PlainText
        color: root.state === "online" ? root.barForeground : root.dim
        font.family: root.bar ? root.bar.fontFamily : Style.font.family
        font.pixelSize: Style.font.body
      }

      Text {
        visible: !widgetButton.vertical
        anchors.verticalCenter: parent.verticalCenter
        text: String(root.taskCount)
        textFormat: Text.PlainText
        color: root.barForeground
        font.family: root.bar ? root.bar.fontFamily : Style.font.family
        font.pixelSize: Style.font.body
        font.bold: root.taskCount > 0
      }
    }
  }

  KeyboardPanel {
    id: popup
    anchorItem: widgetButton
    owner: root
    bar: root.bar
    open: root.opened
    focusTarget: popupKeys
    contentWidth: popup.fittedContentWidth(Style.space(560))
    contentHeight: popup.cappedContentHeight(root.visibleReminders.length > 0
      ? Style.space(350 + Math.min(8, root.visibleReminders.length) * 54)
      : Style.space(400))

    PanelKeyCatcher {
      id: popupKeys
      anchors.fill: parent
      blocked: popupKeys.Window.window && popupKeys.Window.window.activeFocusItem
        && popupKeys.Window.window.activeFocusItem !== popupKeys
      onCloseRequested: root.close()

      ColumnLayout {
        anchors.fill: parent
        spacing: Style.space(10)

        RowLayout {
          Layout.fillWidth: true
          spacing: Style.space(8)

          Column {
            Layout.fillWidth: true
            spacing: Style.space(2)

            Text {
              width: parent.width
              text: "TEND"
              color: root.barForeground
              font.family: root.bar ? root.bar.fontFamily : Style.font.family
              font.pixelSize: Style.font.caption
              font.bold: true
            }

            Text {
              width: parent.width
              text: "~" + (root.tendService ? root.tendService.ship : "") + " · " + root.state
              color: root.dim
              font.family: root.bar ? root.bar.fontFamily : Style.font.family
              font.pixelSize: Style.font.caption
            }
          }

          Button {
            text: "Open Tend"
            iconText: "󰈔"
            focusable: true
            bordered: true
            onClicked: root.openFullPanel(0, 0)
          }
        }

        PanelSeparator { Layout.fillWidth: true; foreground: root.barForeground }

        PanelSectionHeader {
          Layout.fillWidth: true
          text: "QUICK ADD"
          foreground: root.barForeground
        }

        GridLayout {
          columns: 2
          Layout.fillWidth: true
          columnSpacing: Style.space(7)
          rowSpacing: Style.space(7)

          TextField {
            id: quickAdd
            Layout.fillWidth: true
            placeholderText: "Describe a new reminder"
            enabled: destinationList.currentIndex >= 0 && root.tendService && root.tendService.connectionState === "online" && !root.tendService.mutationPending
            onAccepted: root.addReminder()
          }

          TextField {
            id: quickDue
            Layout.fillWidth: true
            placeholderText: "Due date · YYYY-MM-DD or YYYY-MM-DDTHH:MM"
            enabled: quickAdd.enabled
            Accessible.name: "Reminder due date"
            onAccepted: root.addReminder()
          }

          TendDropdown {
            id: destinationList
            Layout.fillWidth: true
            label: "List"
            model: root.destinationLists
            textRole: "title"
            Accessible.name: "Destination list"
          }

          SearchableDropdown {
            id: assigneePicker
            Layout.fillWidth: true
            label: "Assignee"
            options: root.assigneeOptions
            placeholderText: "Find a ship"
            emptyText: "No matching ships"
            Accessible.name: "Reminder assignee"
          }

          Button {
            Layout.columnSpan: 2
            Layout.fillWidth: true
            text: "Add"
            focusable: true
            bordered: true
            enabled: quickAdd.enabled && quickAdd.text.trim() !== ""
            onClicked: root.addReminder()
          }
        }

        PanelSeparator { Layout.fillWidth: true; foreground: root.barForeground }

        RowLayout {
          Layout.fillWidth: true
          spacing: Style.space(8)

          PanelSectionHeader {
            Layout.fillWidth: true
            text: "REMINDERS · PRIORITY ORDER"
            foreground: root.barForeground
          }

          TendDropdown {
            id: smartView
            Layout.preferredWidth: Style.space(165)
            model: root.smartViewLabels
            currentIndex: Math.max(0, root.smartViewKeys.indexOf(root.selectedView))
            Accessible.name: "Smart list"
            onActivated: root.selectedView = root.smartViewKeys[currentIndex]
          }
        }

        Column {
          Layout.fillWidth: true
          spacing: Style.space(4)

          Text {
            visible: root.visibleReminders.length === 0
            width: parent.width
            text: root.state === "online" ? "Nothing in this view." : "Connect Tend to load reminders."
            color: root.dim
            font.family: root.bar ? root.bar.fontFamily : Style.font.family
            font.pixelSize: Style.font.body
            wrapMode: Text.WordWrap
          }

          Repeater {
            model: root.visibleReminders.slice(0, 8)

            delegate: Rectangle {
              required property var modelData
              width: parent.width
              height: Style.space(54)
              opacity: root.tendService ? root.tendService.completionOpacity(modelData.listId, modelData.id) : 1
              radius: Style.cornerRadius
              color: rowMouse.hovered
                ? Style.hoverFillFor(root.barForeground, Color.accent)
                : Qt.rgba(root.barForeground.r, root.barForeground.g, root.barForeground.b, 0.045)

              Row {
                anchors.fill: parent
                anchors.margins: Style.space(7)
                spacing: Style.space(8)

                TendCheckbox {
                  anchors.verticalCenter: parent.verticalCenter
                  checked: root.tendService ? root.tendService.effectiveCompleted(modelData.listId, modelData.id, modelData.completed) : modelData.completed
                  Accessible.name: (checked ? "Mark open " : "Complete ") + modelData.title
                  enabled: root.tendService && root.tendService.canEditList(modelData.listId) && root.state === "online" && !root.tendService.mutationPending
                  onClicked: root.tendService.toggleCompletedWithGrace(modelData.listId, modelData.id, modelData.completed)
                }

                Text {
                  width: Style.space(26)
                  anchors.verticalCenter: parent.verticalCenter
                  text: root.priorityGlyph(modelData.priority)
                  color: root.priorityColor(modelData.priority)
                  font.family: root.bar ? root.bar.fontFamily : Style.font.family
                  font.pixelSize: Style.font.caption
                  font.bold: true
                }

                Column {
                  width: parent.width - x
                  anchors.verticalCenter: parent.verticalCenter
                  spacing: 0

                  Text {
                    width: parent.width
                    text: modelData.title || "Untitled reminder"
                    color: root.barForeground
                    font.family: root.bar ? root.bar.fontFamily : Style.font.family
                    font.pixelSize: Style.font.body
                    elide: Text.ElideRight
                    font.strikeout: root.tendService ? root.tendService.effectiveCompleted(modelData.listId, modelData.id, modelData.completed) : modelData.completed
                  }

                  Text {
                    width: parent.width
                    text: root.dueLabel(modelData) + "  ·  " + modelData.listTitle
                    color: root.dim
                    font.family: root.bar ? root.bar.fontFamily : Style.font.family
                    font.pixelSize: Style.font.caption
                    elide: Text.ElideRight
                  }

                  MouseArea {
                    anchors.fill: parent
                    hoverEnabled: true
                    cursorShape: Qt.PointingHandCursor
                    onClicked: root.openFullPanel(modelData.listId, modelData.id)
                  }
                }
              }

              HoverHandler { id: rowMouse }
            }
          }

          Text {
            visible: root.visibleReminders.length > 8
            width: parent.width
            text: "+ " + (root.visibleReminders.length - 8) + " more"
            color: root.dim
            font.family: root.bar ? root.bar.fontFamily : Style.font.family
            font.pixelSize: Style.font.caption
          }
        }
      }
    }
  }
}
