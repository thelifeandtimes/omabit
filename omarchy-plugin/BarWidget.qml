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
  readonly property var reminderRows: TendModel.menubarRows(tendService ? tendService.lists : [], visibleReminders)
  readonly property int taskCount: visibleReminders.length
  readonly property var destinationLists: tendService
    ? TendModel.orderedLists(tendService.lists, tendService.preferences.pinnedLists, tendService.localSettings.listOrder)
    : []
  readonly property var selectedDestination: destinationList.currentIndex >= 0 && destinationList.currentIndex < destinationLists.length
    ? destinationLists[destinationList.currentIndex]
    : null
  readonly property var assigneeOptions: assigneeOptionsForList(selectedDestination)
  readonly property color dim: Qt.rgba(barForeground.r, barForeground.g, barForeground.b, 0.6)
  readonly property string selectedViewLabel: {
    var index = smartViewKeys.indexOf(selectedView)
    return index >= 0 ? smartViewLabels[index] : "Reminders"
  }
  readonly property string headerMeta: "~" + (tendService ? tendService.ship : "") + " · " + taskCount + " · " + selectedViewLabel

  function normalizedShip(value) {
    var name = String(value || "").replace(/^~/, "")
    return name ? "~" + name : ""
  }

  function assigneeOptionsForList(list) {
    var choices = [{ value: "", label: "Unassigned", role: "" }]
    if (!tendService || !list) return choices
    var access = tendService.accessForList(list.id)
    var seen = ({})
    function append(ship, role) {
      var normalized = root.normalizedShip(ship)
      if (!normalized || seen[normalized]) return
      seen[normalized] = true
      var self = normalized.replace(/^~/, "") === String(tendService.ship || "").replace(/^~/, "")
      choices.push({
        value: normalized,
        label: self ? "me" : normalized,
        role: role || "editor"
      })
    }
    if (!access) return choices
    append(access.host, "admin")
    ;(access.members || []).forEach(function(member) { append(member.ship, member.policy && member.policy.canInvite ? "admin" : "editor") })
    ;(access.pending || []).forEach(function(ship) { append(ship, "invited") })
    if (!seen[root.normalizedShip(tendService.ship)]) append(tendService.ship, access.owner ? "admin" : "editor")
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
      due: quickDue.value,
      allDay: quickDue.allDay,
      timezone: tendService.localTimezone,
      assignee: assigneePicker.value || normalizedShip(tendService.ship)
    })) {
      quickAdd.text = ""
      quickDue.value = ""
      quickDue.allDay = false
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
    contentWidth: popup.fittedContentWidth(Style.space(650))
    contentHeight: Math.min(
      popup.cappedContentHeight(root.visibleReminders.length > 0
        ? Style.space(350 + Math.min(8, root.visibleReminders.length) * 54)
        : Style.space(400)),
      popup.screenH > 0 ? Math.floor(popup.screenH * 0.5) : Style.space(520)
    )

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
          spacing: Style.space(12)

          PanelHero {
            id: tendHero
            Layout.fillWidth: true
            title: "Tend"
            meta: root.headerMeta
            foreground: root.barForeground
            fontFamily: root.bar ? root.bar.fontFamily : Style.font.family
            iconComponent: Component {
              Item {
                implicitWidth: Style.font.display
                implicitHeight: Style.font.display

                Text {
                  anchors.centerIn: parent
                  text: "󰄬"
                  textFormat: Text.PlainText
                  color: root.barForeground
                  font.family: root.bar ? root.bar.fontFamily : Style.font.family
                  font.pixelSize: Style.font.display
                }

                Rectangle {
                  anchors.right: parent.right
                  anchors.bottom: parent.bottom
                  width: Style.space(11)
                  height: width
                  radius: width / 2
                  color: Color.popups.background

                  Rectangle {
                    anchors.centerIn: parent
                    width: Style.space(7)
                    height: width
                    radius: width / 2
                    color: root.state === "online" ? "#22c55e" : "#ef4444"
                    Accessible.role: Accessible.Indicator
                    Accessible.name: root.state === "online" ? "Connected to Tend" : "Tend connection unavailable"
                  }
                }
              }
            }
          }

          TendButton {
            id: openTendButton
            Layout.preferredWidth: Style.spacing.controlHeight
            Layout.preferredHeight: Style.spacing.controlHeight
            Layout.alignment: Qt.AlignVCenter
            text: ""
            tooltipText: "Open Tend"
            bordered: true
            Accessible.name: "Open Tend"
            onClicked: root.openFullPanel(0, 0)

            TendOpenGlyph {
              anchors.centerIn: parent
              width: Style.font.icon
              height: Style.font.icon
              color: root.barForeground
            }
          }
        }

        PanelSeparator { Layout.fillWidth: true; foreground: root.barForeground }

        RowLayout {
          Layout.fillWidth: true

          PanelSectionHeader {
            Layout.fillWidth: true
            text: "QUICK ADD"
            foreground: root.barForeground
          }

          TendDropdown {
            id: destinationList
            Layout.preferredWidth: Style.space(220)
            Layout.preferredHeight: Style.spacing.controlHeight
            showLabel: false
            model: root.destinationLists
            textRole: "title"
            Accessible.name: "Destination list"
          }
        }

        RowLayout {
          Layout.fillWidth: true
          spacing: Style.space(7)

          TendTextField {
            id: quickAdd
            Layout.fillWidth: true
            Layout.preferredHeight: Style.spacing.controlHeight
            placeholderText: "Describe a new reminder"
            enabled: destinationList.currentIndex >= 0 && root.tendService && root.tendService.connectionState === "online" && !root.tendService.mutationPending
            onAccepted: root.addReminder()
          }

          TendAssigneePicker {
            id: assigneePicker
            Layout.preferredWidth: Style.space(170)
            Layout.preferredHeight: Style.spacing.controlHeight
            options: root.assigneeOptions
            placeholderText: "Assignee"
            Accessible.name: "Reminder assignee"
          }

          TendDateTimePicker {
            id: quickDue
            Layout.preferredWidth: Style.spacing.controlHeight
            Layout.preferredHeight: Style.spacing.controlHeight
            placeholderText: "Due date"
            iconOnly: true
            enabled: quickAdd.enabled
            Accessible.name: "Reminder due date"
          }

          Button {
            text: "+ add"
            Layout.preferredWidth: Style.space(62)
            Layout.preferredHeight: Style.spacing.controlHeight
            tooltipText: "Add reminder"
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

        ListView {
          id: menubarReminderList
          Layout.fillWidth: true
          Layout.fillHeight: true
          Layout.minimumHeight: Style.space(64)
          clip: true
          boundsBehavior: Flickable.StopAtBounds
          spacing: Style.space(4)
          model: root.reminderRows

          Text {
            anchors.centerIn: parent
            visible: root.visibleReminders.length === 0
            text: root.state === "online" ? "Nothing in this view." : "Connect Tend to load reminders."
            color: root.dim
            font.family: root.bar ? root.bar.fontFamily : Style.font.family
            font.pixelSize: Style.font.body
            wrapMode: Text.WordWrap
          }

          delegate: Item {
              id: reminderRowDelegate
              required property var modelData
              readonly property bool summaryRow: modelData.kind === "summary"
              readonly property var reminder: summaryRow ? ({}) : (modelData.reminder || {})
              readonly property real indentPixels: Math.min(
                Math.max(0, Number(modelData.depth || 0)) * Style.space(18),
                Math.max(0, width - Style.space(120))
              )
              width: ListView.view.width
              height: summaryRow ? Style.space(30) : Style.space(54)
              opacity: summaryRow || !root.tendService ? 1 : root.tendService.completionOpacity(reminder.listId, reminder.id)
              Accessible.role: Accessible.ListItem
              Accessible.name: summaryRow
                ? Number(modelData.count || 0) + " filtered subitems"
                : (Number(modelData.depth || 0) > 0 ? "Subtask, " : "") + (reminder.title || "Untitled reminder")

              Text {
                id: filteredSummaryButton
                visible: summaryRow
                x: reminderRowDelegate.indentPixels
                anchors.verticalCenter: parent.verticalCenter
                text: Number(modelData.count || 0) + " filtered subitem" + (Number(modelData.count || 0) === 1 ? "" : "s")
                width: Math.min(implicitWidth, Math.max(0, parent.width - x))
                color: root.dim
                font.family: root.bar ? root.bar.fontFamily : Style.font.family
                font.pixelSize: Style.font.caption
                font.underline: filteredSummaryHover.hovered
                elide: Text.ElideRight
                Accessible.name: text + ". Open parent reminder."

                HoverHandler { id: filteredSummaryHover }
                MouseArea {
                  anchors.fill: parent
                  cursorShape: Qt.PointingHandCursor
                  onClicked: root.openFullPanel(modelData.listId, modelData.parentReminderId)
                }
              }

              Loader {
                id: reminderCard

                active: !summaryRow
                x: reminderRowDelegate.indentPixels
                width: Math.max(0, parent.width - x)
                height: parent.height
                sourceComponent: Rectangle {
                  radius: Style.cornerRadius
                  color: rowMouse.hovered
                    ? Style.hoverFillFor(root.barForeground, Color.accent)
                    : Qt.rgba(root.barForeground.r, root.barForeground.g, root.barForeground.b, 0.045)

                  Row {
                    anchors.fill: parent
                    anchors.margins: Style.space(7)
                    spacing: Style.space(8)

                  TendButton {
                    visible: modelData.orphanParentId !== null && modelData.orphanParentId !== undefined
                    width: visible ? Style.space(26) : 0
                    height: Style.space(30)
                    anchors.verticalCenter: parent.verticalCenter
                    text: "↳"
                    bordered: false
                    tooltipText: "Open parent reminder"
                    Accessible.name: "Open parent reminder"
                    onClicked: root.openFullPanel(reminder.listId, modelData.orphanParentId)
                  }

                  TendCheckbox {
                    anchors.verticalCenter: parent.verticalCenter
                    checked: !summaryRow && (root.tendService ? root.tendService.effectiveCompleted(reminder.listId, reminder.id, reminder.completed) : reminder.completed)
                    Accessible.name: summaryRow ? "" : (checked ? "Mark open " : "Complete ") + reminder.title
                    enabled: !summaryRow && root.tendService && root.tendService.canEditList(reminder.listId) && root.state === "online" && !root.tendService.mutationPending
                    onClicked: root.tendService.toggleCompletedWithGrace(reminder.listId, reminder.id, reminder.completed)
                  }

                  TendFlagButton {
                    anchors.verticalCenter: parent.verticalCenter
                    flagged: !summaryRow && reminder.flagged === true
                    enabled: !summaryRow && root.tendService && root.tendService.canEditList(reminder.listId) && root.state === "online" && !root.tendService.mutationPending
                    onToggled: function(flagged) { root.tendService.setReminderFlagged(reminder.listId, reminder.id, flagged) }
                  }

                  Text {
                    width: Style.space(26)
                    anchors.verticalCenter: parent.verticalCenter
                    text: summaryRow ? "" : root.priorityGlyph(reminder.priority)
                    color: root.priorityColor(reminder.priority)
                    font.family: root.bar ? root.bar.fontFamily : Style.font.family
                    font.pixelSize: Style.font.caption
                    font.bold: true
                  }

                  Item {
                    width: parent.width - x
                    height: parent.height

                    Column {
                      anchors.left: parent.left
                      anchors.right: parent.right
                      anchors.verticalCenter: parent.verticalCenter
                      spacing: 0

                      Text {
                        width: parent.width
                        text: summaryRow ? "" : (reminder.title || "Untitled reminder")
                        color: root.barForeground
                        font.family: root.bar ? root.bar.fontFamily : Style.font.family
                        font.pixelSize: Style.font.body
                        elide: Text.ElideRight
                        font.strikeout: !summaryRow && (root.tendService ? root.tendService.effectiveCompleted(reminder.listId, reminder.id, reminder.completed) : reminder.completed)
                      }

                      Text {
                        width: parent.width
                        text: summaryRow ? "" : root.dueLabel(reminder) + (Number(modelData.depth || 0) > 0 ? "" : "  ·  " + reminder.listTitle)
                        color: root.dim
                        font.family: root.bar ? root.bar.fontFamily : Style.font.family
                        font.pixelSize: Style.font.caption
                        elide: Text.ElideRight
                      }
                    }

                    MouseArea {
                      anchors.fill: parent
                      hoverEnabled: true
                      cursorShape: Qt.PointingHandCursor
                      onClicked: root.openFullPanel(reminder.listId, reminder.id)
                    }
                  }
                  }

                  HoverHandler { id: rowMouse }
                }
              }
            }
        }
      }
    }
  }
}
