import QtQuick
import QtQuick.Controls as QQC
import qs.Commons
import qs.Ui
import "TendModel.js" as TendModel

BorderSurface {
  id: root

  property var service: null
  property var list: null
  property var reminder: null
  property var assigneeChoices: []
  property var listChoices: []
  property var availableTags: []
  property bool editable: false
  property string timezone: "UTC"
  property bool loadingFields: false
  property bool pendingTextSave: false
  property var recurrenceBase: null
  signal closeRequested()
  signal moveRequested(int destinationListId)
  signal reminderRequested(int reminderId)

  readonly property var assigneeOptions: (assigneeChoices || []).map(function(choice) {
    return {
      value: String(choice.ship || ""),
      label: String(choice.label || choice.ship || "Unassigned"),
      role: String(choice.role || choice.status || "")
    }
  })
  readonly property var tagOptions: {
    var values = (availableTags || []).concat(tagPicker.values || [])
    var seen = ({})
    return values.filter(function(tag) {
      var value = String(tag || "").trim()
      if (!value || seen[value]) return false
      seen[value] = true
      return true
    })
  }

  color: Qt.rgba(Color.popups.background.r, Color.popups.background.g, Color.popups.background.b, 1)
  borderSpec: Border.localOrSurfaceSpec("popups", "border", Color.popups.border, Color.menu.border, Style.normalBorderWidth)
  radius: Style.cornerRadius

  function normalizedShip(value) {
    var name = String(value || "").replace(/^~/, "")
    return name ? "~" + name : ""
  }

  function reminderById(reminderId) {
    var reminders = list ? (list.reminders || []) : []
    for (var i = 0; i < reminders.length; i++)
      if (Number(reminders[i].id) === Number(reminderId)) return reminders[i]
    return null
  }

  function parentIdFor(item) {
    if (!item) return null
    return item.parentId === undefined ? item["parent-id"] : item.parentId
  }

  function ancestorRows() {
    if (!reminder || !list) return []
    var result = []
    var seen = ({})
    var parentId = parentIdFor(reminder)
    while (parentId !== null && parentId !== undefined && result.length < 24) {
      var numeric = Number(parentId)
      if (seen[numeric]) break
      seen[numeric] = true
      var parent = reminderById(numeric)
      if (!parent) break
      result.unshift({ id: Number(parent.id), title: String(parent.title || "Untitled reminder"), depth: result.length })
      parentId = parentIdFor(parent)
    }
    for (var i = 0; i < result.length; i++) result[i].depth = i
    return result
  }

  function descendantRows() {
    if (!reminder || !list) return []
    var reminders = list.reminders || []
    var result = []
    var seen = ({})
    function appendChildren(parentId, depth) {
      if (depth > 24) return
      var children = reminders.filter(function(item) {
        var value = root.parentIdFor(item)
        return value !== null && value !== undefined && Number(value) === Number(parentId)
      }).sort(function(left, right) {
        return Number(left.rank || 0) - Number(right.rank || 0) || Number(left.id) - Number(right.id)
      })
      for (var i = 0; i < children.length; i++) {
        var child = children[i]
        if (seen[Number(child.id)]) continue
        seen[Number(child.id)] = true
        result.push({ id: Number(child.id), title: String(child.title || "Untitled reminder"), depth: depth })
        appendChildren(child.id, depth + 1)
      }
    }
    appendChildren(reminder.id, 0)
    return result
  }

  function recurrenceEndInput(recurrence) {
    if (!recurrence) return ""
    var value = recurrence.localEnd || recurrence["local-end"] || recurrence.endAt || recurrence["end-at"] || ""
    return String(value)
  }

  function recurrenceValue() {
    if (repeatField.currentText === "none") return null
    var source = recurrenceBase || ({})
    return {
      frequency: repeatField.currentText,
      interval: Math.max(1, Number(repeatInterval.value || 1)),
      weekdays: (source.weekdays || []).map(Number),
      monthDays: (source.monthDays || source["month-days"] || []).map(Number),
      monthWeek: source.monthWeek || source["month-week"] || null,
      endAt: repeatEndField.currentText === "on date" && repeatEndPicker.value ? repeatEndPicker.value : null,
      maxOccurrences: repeatEndField.currentText === "after count" ? Math.max(1, Number(repeatCount.value || 1)) : null
    }
  }

  function repeatUnitLabel() {
    var singular = repeatField.currentText === "hourly" ? "hour" : repeatField.currentText === "daily" ? "day" : repeatField.currentText === "weekly" ? "week" : repeatField.currentText === "monthly" ? "month" : "year"
    return Number(repeatInterval.value || 1) === 1 ? singular : singular + "s"
  }

  function loadReminder() {
    if (!reminder) return
    loadingFields = true
    titleField.text = reminder.title || ""
    notesField.text = reminder.notes || ""
    urlField.text = reminder.url || ""
    priorityField.currentIndex = Math.max(0, priorityField.model.indexOf(reminder.priority || "none"))
    flaggedField.flagged = reminder.flagged === true
    tagPicker.values = (reminder.tags || []).slice()
    assigneeField.value = normalizedShip(reminder.assignee)
    listField.currentIndex = indexForList(list ? list.id : 0)
    duePicker.value = TendModel.scheduleInputValue(reminder.schedule)
    duePicker.allDay = reminder.schedule ? reminder.schedule.allDay === true : false
    allDayField.checked = duePicker.allDay
    var recurrence = reminder.schedule ? reminder.schedule.recurrence : null
    recurrenceBase = recurrence ? {
      weekdays: (recurrence.weekdays || []).slice(),
      monthDays: (recurrence.monthDays || recurrence["month-days"] || []).slice(),
      monthWeek: recurrence.monthWeek || recurrence["month-week"] || null
    } : null
    repeatField.currentIndex = Math.max(0, repeatField.model.indexOf(recurrence ? String(recurrence.frequency || "daily") : "none"))
    repeatInterval.value = recurrence ? Math.max(1, Number(recurrence.interval || 1)) : 1
    repeatEndPicker.value = recurrenceEndInput(recurrence)
    repeatEndPicker.allDay = false
    repeatCount.value = recurrence && (recurrence.maxOccurrences !== null && recurrence.maxOccurrences !== undefined)
      ? Math.max(1, Number(recurrence.maxOccurrences)) : 1
    repeatEndField.currentIndex = recurrence && recurrenceEndInput(recurrence)
      ? repeatEndField.model.indexOf("on date")
      : recurrence && recurrence.maxOccurrences !== null && recurrence.maxOccurrences !== undefined
        ? repeatEndField.model.indexOf("after count") : 0
    loadingFields = false
  }

  function indexForList(listId) {
    for (var i = 0; i < (listChoices || []).length; i++)
      if (Number(listChoices[i].id) === Number(listId)) return i
    return (listChoices || []).length ? 0 : -1
  }

  function canSave() {
    return editable && reminder && list && service && service.connectionState === "online" && titleField.text.trim() !== ""
  }

  function showSaved() {
    savedToast.visible = true
    savedToast.opacity = 1
    savedToastTimer.restart()
  }

  function scheduleTextSave() {
    if (loadingFields) return
    pendingTextSave = true
    textSaveTimer.restart()
  }

  function saveDetailsNow() {
    if (loadingFields || !canSave()) return false
    if (service.mutationPending) {
      pendingTextSave = true
      retrySaveTimer.restart()
      return false
    }
    pendingTextSave = false
    var accepted = service.updateReminder(list.id, reminder.id, {
      title: titleField.text,
      notes: notesField.text,
      url: urlField.text.trim() || null,
      priority: priorityField.currentText,
      flagged: flaggedField.flagged,
      tags: tagPicker.values || [],
      assignee: assigneeField.value || null
    }, list.revision)
    if (accepted) showSaved()
    return accepted
  }

  function saveSchedule() {
    if (loadingFields || !canSave()) return false
    if (service.mutationPending) {
      scheduleRetryTimer.restart()
      return false
    }
    var value = String(duePicker.value || "").trim()
    var accepted = service.setSchedule(list.id, reminder.id, value ? {
      due: value,
      allDay: allDayField.checked,
      timezone: timezone,
      earlySeconds: [],
      recurrence: recurrenceValue()
    } : null, list.revision)
    if (accepted) showSaved()
    return accepted
  }

  function addTag() {
    var tag = newTagField.text.trim()
    if (!tag) return
    var tags = (tagPicker.values || []).slice()
    if (tags.indexOf(tag) === -1) tags.push(tag)
    tagPicker.values = tags
    newTagField.text = ""
    saveDetailsNow()
  }

  function focusAssignee() { assigneeField.open() }
  function focusSave() { titleField.forceActiveFocus() }

  onReminderChanged: loadReminder()
  onAssigneeChoicesChanged: loadReminder()
  onListChoicesChanged: loadReminder()
  Component.onCompleted: loadReminder()

  Timer { id: textSaveTimer; interval: 300; repeat: false; onTriggered: root.saveDetailsNow() }
  Timer {
    id: retrySaveTimer
    interval: 250
    repeat: false
    onTriggered: if (root.pendingTextSave) root.saveDetailsNow()
  }
  Timer { id: scheduleRetryTimer; interval: 250; repeat: false; onTriggered: root.saveSchedule() }
  Timer {
    id: savedToastTimer
    interval: 1350
    repeat: false
    onTriggered: { savedToast.opacity = 0; savedToastHideTimer.restart() }
  }
  Timer { id: savedToastHideTimer; interval: 160; repeat: false; onTriggered: savedToast.visible = false }

  QQC.ScrollView {
    anchors.fill: parent
    anchors.margins: Style.space(12)
    clip: true
    QQC.ScrollBar.horizontal.policy: QQC.ScrollBar.AlwaysOff

    Column {
      width: root.width - Style.space(26)
      spacing: Style.space(10)

      Row {
        width: parent.width
        spacing: Style.space(6)

        Column {
          width: parent.width - flaggedField.width - closeButton.width - parent.spacing * 2
          spacing: Style.space(2)
          Text { width: parent.width; text: "REMINDER DETAILS"; color: Color.popups.text; font.family: Style.font.family; font.pixelSize: Style.font.caption; font.bold: true }
          Text { width: parent.width; text: root.list ? root.list.title : ""; color: Qt.darker(Color.popups.text, 1.45); elide: Text.ElideRight; font.family: Style.font.family; font.pixelSize: Style.font.caption }
        }

        TendFlagButton {
          id: flaggedField
          flagged: false
          enabled: root.canSave()
          onToggled: function(flagged) { flaggedField.flagged = flagged; root.saveDetailsNow() }
        }

        TendIconButton { id: closeButton; glyph: "󰅖"; bordered: false; tooltipText: "Close details"; accessibleName: "Close reminder details"; onClicked: root.closeRequested() }
      }

      TendCheckbox {
        id: completedBox
        text: "Completed"
        checked: root.service && root.reminder ? root.service.effectiveCompleted(root.list.id, root.reminder.id, root.reminder.completed) : false
        enabled: root.editable && root.reminder && root.service && root.service.connectionState === "online" && !root.service.mutationPending
        onClicked: root.service.toggleCompletedWithGrace(root.list.id, root.reminder.id, root.reminder.completed)
      }

      TextField { id: titleField; width: parent.width; placeholderText: "Reminder title"; Accessible.name: "Reminder title"; onTextEdited: root.scheduleTextSave(); QQC.ContextMenu.menu: TendEditMenu { target: titleField } }
      TendTextArea { id: notesField; width: parent.width; height: Style.space(96); placeholderText: "Notes"; wrapMode: TextEdit.Wrap; Accessible.name: "Reminder notes"; onTextChanged: root.scheduleTextSave() }
      TextField { id: urlField; width: parent.width; placeholderText: "Link"; Accessible.name: "Reminder URL"; onTextEdited: root.scheduleTextSave(); QQC.ContextMenu.menu: TendEditMenu { target: urlField } }

      TendDropdown { id: priorityField; width: parent.width; label: "Priority"; model: ["none", "low", "medium", "high"]; onActivated: root.saveDetailsNow() }

      TendAssigneePicker { id: assigneeField; width: parent.width; options: root.assigneeOptions; placeholderText: "Assignee"; Accessible.name: "Reminder assignee ship"; onChanged: root.saveDetailsNow() }

      TendDropdown {
        id: listField
        width: parent.width
        label: "List"
        model: root.listChoices || []
        textRole: "title"
        valueRole: "id"
        enabled: root.editable && count > 1 && root.service && !root.service.mutationPending
        onActivated: {
          var destinationId = Number(currentValue)
          if (root.list && destinationId !== Number(root.list.id)) root.moveRequested(destinationId)
        }
      }

      MultiSelect {
        id: tagPicker
        width: parent.width
        label: "Tags"
        options: root.tagOptions
        placeholderText: "Find a tag"
        emptyText: "No matching tags"
        noSelectionText: "No tags"
        Accessible.name: "Reminder tags"
        onChanged: root.saveDetailsNow()
      }

      Row {
        width: parent.width
        spacing: Style.space(8)
        TextField { id: newTagField; width: parent.width - addTagButton.width - parent.spacing; placeholderText: "Add a new tag"; Accessible.name: "New reminder tag"; onAccepted: root.addTag(); QQC.ContextMenu.menu: TendEditMenu { target: newTagField } }
        TendButton { id: addTagButton; text: "Add tag"; enabled: newTagField.text.trim() !== ""; onClicked: root.addTag() }
      }

      Column {
        width: parent.width
        spacing: Style.space(6)
        visible: root.ancestorRows().length > 0 || root.descendantRows().length > 0

        PanelSeparator { width: parent.width; foreground: Color.popups.text }

        Text {
          visible: root.ancestorRows().length > 0
          text: "ANCESTORS"
          color: Color.popups.text
          opacity: 0.68
          font.family: Style.font.family
          font.pixelSize: Style.font.caption
          font.bold: true
        }

        Repeater {
          model: root.ancestorRows()
          delegate: Item {
            required property var modelData
            width: parent.width
            height: Style.space(26)
            Text {
              id: ancestorLabel
              anchors.left: parent.left
              anchors.right: parent.right
              anchors.verticalCenter: parent.verticalCenter
              leftPadding: Number(modelData.depth || 0) * Style.space(14)
              text: "↳ " + modelData.title
              color: Color.popups.text
              opacity: ancestorHover.hovered ? 1 : 0.7
              elide: Text.ElideRight
              font.family: Style.font.family
              font.pixelSize: Style.font.caption
              font.underline: ancestorHover.hovered
            }
            HoverHandler { id: ancestorHover }
            TapHandler { onTapped: root.reminderRequested(Number(modelData.id)) }
          }
        }

        Text {
          visible: root.descendantRows().length > 0
          text: "DESCENDANTS"
          color: Color.popups.text
          opacity: 0.68
          font.family: Style.font.family
          font.pixelSize: Style.font.caption
          font.bold: true
        }

        Repeater {
          model: root.descendantRows()
          delegate: Item {
            required property var modelData
            width: parent.width
            height: Style.space(26)
            Text {
              anchors.left: parent.left
              anchors.right: parent.right
              anchors.verticalCenter: parent.verticalCenter
              leftPadding: (Number(modelData.depth || 0) + 1) * Style.space(14)
              text: "↳ " + modelData.title
              color: Color.popups.text
              opacity: descendantHover.hovered ? 1 : 0.7
              elide: Text.ElideRight
              font.family: Style.font.family
              font.pixelSize: Style.font.caption
              font.underline: descendantHover.hovered
            }
            HoverHandler { id: descendantHover }
            TapHandler { onTapped: root.reminderRequested(Number(modelData.id)) }
          }
        }
      }

      PanelSeparator { width: parent.width; foreground: Color.popups.text }
      Text { text: "SCHEDULE"; color: Color.popups.text; opacity: 0.68; font.family: Style.font.family; font.pixelSize: Style.font.caption; font.bold: true }

      Row {
        width: parent.width
        spacing: Style.space(8)

        Text {
          width: parent.width - duePicker.width - parent.spacing
          anchors.verticalCenter: parent.verticalCenter
          text: duePicker.value ? duePicker.displayValue() : "No due date"
          color: Color.popups.text
          opacity: duePicker.value ? 0.86 : 0.56
          elide: Text.ElideRight
          font.family: Style.font.family
          font.pixelSize: Style.font.body
        }

        TendDateTimePicker {
          id: duePicker
          iconOnly: true
          placeholderText: "Add due date"
          onCommitted: function(value, allDay) { allDayField.checked = allDay; root.saveSchedule() }
        }
      }

      TendCheckbox {
        id: allDayField
        text: "All day"
        enabled: root.canSave() && duePicker.value !== ""
        onClicked: { checked = !checked; duePicker.allDay = checked; root.saveSchedule() }
      }

      Column {
        width: parent.width
        spacing: Style.space(8)
        visible: duePicker.value !== ""

        Text { text: "REPEAT"; color: Color.popups.text; opacity: 0.68; font.family: Style.font.family; font.pixelSize: Style.font.caption; font.bold: true }

        TendDropdown {
          id: repeatField
          width: parent.width
          label: "Repeat"
          model: ["none", "hourly", "daily", "weekly", "monthly", "yearly"]
          Accessible.name: "Repeat frequency"
          onActivated: root.saveSchedule()
        }

        Row {
          width: parent.width
          spacing: Style.space(8)
          visible: repeatField.currentText !== "none"

          TendNumber {
            id: repeatInterval
            width: Math.min(Style.space(120), parent.width * 0.38)
            label: "Every"
            from: 1
            to: 999
            value: 1
            editable: true
            Accessible.name: "Repeat interval"
            onValueChanged: if (activeFocus && !root.loadingFields) root.saveSchedule()
          }

          Text {
            anchors.verticalCenter: parent.verticalCenter
            text: root.repeatUnitLabel()
            color: Color.popups.text
            opacity: 0.72
            font.family: Style.font.family
            font.pixelSize: Style.font.body
          }
        }

        TendDropdown {
          id: repeatEndField
          width: parent.width
          visible: repeatField.currentText !== "none"
          label: "Ends"
          model: ["never", "on date", "after count"]
          Accessible.name: "Repeat ending"
          onActivated: if (currentText === "never") root.saveSchedule()
        }

        Row {
          width: parent.width
          spacing: Style.space(8)
          visible: repeatField.currentText !== "none" && repeatEndField.currentText === "on date"

          Text {
            width: parent.width - repeatEndPicker.width - parent.spacing
            anchors.verticalCenter: parent.verticalCenter
            text: repeatEndPicker.value ? repeatEndPicker.displayValue() : "Choose an end date"
            color: Color.popups.text
            opacity: repeatEndPicker.value ? 0.86 : 0.56
            elide: Text.ElideRight
            font.family: Style.font.family
            font.pixelSize: Style.font.body
          }

          TendDateTimePicker {
            id: repeatEndPicker
            iconOnly: true
            placeholderText: "Repeat end date"
            onCommitted: root.saveSchedule()
          }
        }

        TendNumber {
          id: repeatCount
          width: parent.width
          visible: repeatField.currentText !== "none" && repeatEndField.currentText === "after count"
          label: "Occurrences"
          from: 1
          to: 9999
          value: 1
          editable: true
          Accessible.name: "Maximum repeat occurrences"
          onValueChanged: if (activeFocus && !root.loadingFields) root.saveSchedule()
        }
      }

      PanelSeparator { width: parent.width; foreground: Color.popups.text }

      TendButton {
        width: parent.width
        text: "Delete reminder"
        foreground: Color.urgent
        accent: Color.urgent
        bordered: true
        enabled: root.editable && root.reminder && root.service && root.service.connectionState === "online" && !root.service.mutationPending
        onClicked: root.service.deleteReminder(root.list.id, root.reminder.id, root.list.revision)
      }
    }
  }

  BorderSurface {
    id: savedToast
    visible: false
    opacity: 0
    anchors.top: parent.top
    anchors.right: parent.right
    anchors.margins: Style.space(10)
    width: savedLabel.implicitWidth + Style.space(22)
    height: Style.space(32)
    z: 20
    color: Style.selectedFillFor(Color.popups.text, Color.accent)
    borderSpec: Border.controlSpec("selected", Color.popups.text, Color.accent)
    radius: Style.cornerRadius
    Text { id: savedLabel; anchors.centerIn: parent; text: "✓ Details saved"; color: Style.selectedStateColor(Color.popups.text, Color.accent); font.family: Style.font.family; font.pixelSize: Style.font.caption; font.bold: true }
  }
}
