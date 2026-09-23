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
  signal closeRequested()

  readonly property var assigneeOptions: (assigneeChoices || []).map(function(choice) {
    return {
      value: String(choice.ship || ""),
      label: String(choice.title || choice.ship || "Unassigned"),
      description: String(choice.description || "")
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

  color: Color.popups.background
  borderSpec: Border.localOrSurfaceSpec("popups", "border", Color.popups.border, Color.menu.border, Style.normalBorderWidth)
  radius: Style.cornerRadius

  function normalizedShip(value) {
    var name = String(value || "").replace(/^~/, "")
    return name ? "~" + name : ""
  }

  function loadReminder() {
    if (!reminder) return
    titleField.text = reminder.title || ""
    notesField.text = reminder.notes || ""
    urlField.text = reminder.url || ""
    priorityField.currentIndex = Math.max(0, priorityField.model.indexOf(reminder.priority || "none"))
    flaggedField.checked = reminder.flagged === true
    tagPicker.values = (reminder.tags || []).slice()
    assigneeField.value = normalizedShip(reminder.assignee)
    dueField.text = TendModel.scheduleInputValue(reminder.schedule)
    allDayField.checked = reminder.schedule ? reminder.schedule.allDay === true : false
  }

  function addTag() {
    var tag = newTagField.text.trim()
    if (!tag) return
    var tags = (tagPicker.values || []).slice()
    if (tags.indexOf(tag) === -1) tags.push(tag)
    tagPicker.values = tags
    newTagField.text = ""
  }

  function focusAssignee() { assigneeField.open() }
  function focusSave() { saveButton.forceActiveFocus() }

  onReminderChanged: loadReminder()
  onAssigneeChoicesChanged: loadReminder()
  Component.onCompleted: loadReminder()

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

          Text {
            width: parent.width
            text: "REMINDER DETAILS"
            color: Color.popups.text
            font.family: Style.font.family
            font.pixelSize: Style.font.caption
            font.bold: true
          }

          Text {
            width: parent.width
            text: root.list ? root.list.title : ""
            color: Qt.darker(Color.popups.text, 1.45)
            elide: Text.ElideRight
            font.family: Style.font.family
            font.pixelSize: Style.font.caption
          }
        }

        TendButton {
          id: closeButton
          text: "󰅖"
          bordered: false
          tooltipText: "Close details"
          onClicked: root.closeRequested()
        }
      }

      TendCheckbox {
        id: completedBox
        text: "Completed"
        checked: root.service && root.reminder
          ? root.service.effectiveCompleted(root.list.id, root.reminder.id, root.reminder.completed)
          : false
        enabled: root.editable && root.reminder && root.service && root.service.connectionState === "online" && !root.service.mutationPending
        onClicked: root.service.toggleCompletedWithGrace(root.list.id, root.reminder.id, root.reminder.completed)
      }

      TextField {
        id: titleField
        width: parent.width
        placeholderText: "Reminder title"
        Accessible.name: "Reminder title"
      }

      TendTextArea {
        id: notesField
        width: parent.width
        height: Style.space(92)
        placeholderText: "Notes"
        wrapMode: TextEdit.Wrap
        Accessible.name: "Reminder notes"
      }

      TextField {
        id: urlField
        width: parent.width
        placeholderText: "Link"
        Accessible.name: "Reminder URL"
      }

      Row {
        width: parent.width
        spacing: Style.space(8)

        TendDropdown {
          id: priorityField
          width: parent.width * 0.54
          label: "Priority"
          model: ["none", "low", "medium", "high"]
        }

        TendCheck {
          id: flaggedField
          anchors.verticalCenter: parent.verticalCenter
          text: "Flagged"
        }
      }

      SearchableDropdown {
        id: assigneeField
        width: parent.width
        label: "Assigned to"
        options: root.assigneeOptions
        placeholderText: "Find a ship"
        emptyText: "No matching ships"
        Accessible.name: "Reminder assignee ship"
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
      }

      Row {
        width: parent.width
        spacing: Style.space(8)

        TextField {
          id: newTagField
          width: parent.width - addTagButton.width - parent.spacing
          placeholderText: "Add a new tag"
          Accessible.name: "New reminder tag"
          onAccepted: root.addTag()
        }

        TendButton {
          id: addTagButton
          text: "Add tag"
          enabled: newTagField.text.trim() !== ""
          onClicked: root.addTag()
        }
      }

      PanelSeparator { width: parent.width; foreground: Color.popups.text }

      Text {
        text: "SCHEDULE"
        color: Color.popups.text
        opacity: 0.68
        font.family: Style.font.family
        font.pixelSize: Style.font.caption
        font.bold: true
      }

      TextField {
        id: dueField
        width: parent.width
        placeholderText: "YYYY-MM-DD or YYYY-MM-DDTHH:MM"
        Accessible.name: "Reminder due date and time"
      }

      TendCheck {
        id: allDayField
        text: "All day"
      }

      Row {
        width: parent.width
        spacing: Style.space(8)

        TendButton {
          text: "Save schedule"
          enabled: root.editable && root.reminder && dueField.text.trim() && root.service && root.service.connectionState === "online" && !root.service.mutationPending
          onClicked: root.service.setSchedule(root.list.id, root.reminder.id, {
            due: dueField.text.trim(),
            allDay: allDayField.checked,
            timezone: root.timezone,
            earlySeconds: [],
            recurrence: null
          }, root.list.revision)
        }

        TendButton {
          text: "Clear"
          enabled: root.editable && root.reminder && root.reminder.schedule && root.service && root.service.connectionState === "online" && !root.service.mutationPending
          onClicked: root.service.setSchedule(root.list.id, root.reminder.id, null, root.list.revision)
        }
      }

      PanelSeparator { width: parent.width; foreground: Color.popups.text }

      Row {
        width: parent.width
        spacing: Style.space(8)

        TendButton {
          id: saveButton
          width: parent.width - deleteButton.width - parent.spacing
          text: root.service && root.service.mutationPending ? "Saving…" : "Save changes"
          enabled: root.editable && root.reminder && titleField.text.trim() && root.service && root.service.connectionState === "online" && !root.service.mutationPending
          onClicked: root.service.updateReminder(root.list.id, root.reminder.id, {
            title: titleField.text,
            notes: notesField.text,
            url: urlField.text.trim() || null,
            priority: priorityField.currentText,
            flagged: flaggedField.checked,
            tags: tagPicker.values || [],
            assignee: assigneeField.value || null
          }, root.list.revision)
        }

        TendButton {
          id: deleteButton
          text: "Delete"
          enabled: root.editable && root.reminder && root.service && root.service.connectionState === "online" && !root.service.mutationPending
          onClicked: root.service.deleteReminder(root.list.id, root.reminder.id, root.list.revision)
        }
      }
    }
  }
}
