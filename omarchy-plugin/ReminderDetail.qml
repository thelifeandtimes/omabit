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
  property var availableTags: []
  property bool editable: false
  property string timezone: "UTC"
  property bool loadingFields: false
  property bool pendingTextSave: false
  signal closeRequested()

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
    duePicker.value = TendModel.scheduleInputValue(reminder.schedule)
    duePicker.allDay = reminder.schedule ? reminder.schedule.allDay === true : false
    allDayField.checked = duePicker.allDay
    loadingFields = false
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
      recurrence: null
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

        Column {
          width: parent.width - closeButton.width
          spacing: Style.space(2)
          Text { width: parent.width; text: "REMINDER DETAILS"; color: Color.popups.text; font.family: Style.font.family; font.pixelSize: Style.font.caption; font.bold: true }
          Text { width: parent.width; text: root.list ? root.list.title : ""; color: Qt.darker(Color.popups.text, 1.45); elide: Text.ElideRight; font.family: Style.font.family; font.pixelSize: Style.font.caption }
        }

        TendButton { id: closeButton; text: "󰅖"; bordered: false; tooltipText: "Close details"; onClicked: root.closeRequested() }
      }

      Row {
        width: parent.width
        spacing: Style.space(8)

        TendCheckbox {
          id: completedBox
          text: "Completed"
          checked: root.service && root.reminder ? root.service.effectiveCompleted(root.list.id, root.reminder.id, root.reminder.completed) : false
          enabled: root.editable && root.reminder && root.service && root.service.connectionState === "online" && !root.service.mutationPending
          onClicked: root.service.toggleCompletedWithGrace(root.list.id, root.reminder.id, root.reminder.completed)
        }

        TendFlagButton {
          id: flaggedField
          flagged: false
          enabled: root.canSave()
          onToggled: function(flagged) { flaggedField.flagged = flagged; root.saveDetailsNow() }
        }

        Text { anchors.verticalCenter: parent.verticalCenter; text: flaggedField.flagged ? "Flagged" : "Flag reminder"; color: flaggedField.flagged ? Color.urgent : Qt.darker(Color.popups.text, 1.45); font.family: Style.font.family; font.pixelSize: Style.font.body }
      }

      TextField { id: titleField; width: parent.width; placeholderText: "Reminder title"; Accessible.name: "Reminder title"; onTextEdited: root.scheduleTextSave(); QQC.ContextMenu.menu: TendEditMenu { target: titleField } }
      TendTextArea { id: notesField; width: parent.width; height: Style.space(96); placeholderText: "Notes"; wrapMode: TextEdit.Wrap; Accessible.name: "Reminder notes"; onTextChanged: root.scheduleTextSave() }
      TextField { id: urlField; width: parent.width; placeholderText: "Link"; Accessible.name: "Reminder URL"; onTextEdited: root.scheduleTextSave(); QQC.ContextMenu.menu: TendEditMenu { target: urlField } }

      TendDropdown { id: priorityField; width: parent.width; label: "Priority"; model: ["none", "low", "medium", "high"]; onActivated: root.saveDetailsNow() }

      TendAssigneePicker { id: assigneeField; width: parent.width; options: root.assigneeOptions; placeholderText: "Assignee"; Accessible.name: "Reminder assignee ship"; onChanged: root.saveDetailsNow() }

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

      PanelSeparator { width: parent.width; foreground: Color.popups.text }
      Text { text: "SCHEDULE"; color: Color.popups.text; opacity: 0.68; font.family: Style.font.family; font.pixelSize: Style.font.caption; font.bold: true }

      TendDateTimePicker {
        id: duePicker
        width: parent.width
        placeholderText: "Add due date"
        onCommitted: function(value, allDay) { allDayField.checked = allDay; root.saveSchedule() }
      }

      TendCheckbox {
        id: allDayField
        text: "All day"
        enabled: root.canSave() && duePicker.value !== ""
        onClicked: { checked = !checked; duePicker.allDay = checked; root.saveSchedule() }
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
