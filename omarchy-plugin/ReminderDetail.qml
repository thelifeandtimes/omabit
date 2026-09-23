import QtQuick
import QtQuick.Controls as QQC
import qs.Commons
import qs.Ui
import "TendModel.js" as TendModel

Rectangle {
  id: root

  property var service: null
  property var list: null
  property var reminder: null
  property var assigneeChoices: []
  property bool editable: false
  property string timezone: "UTC"
  signal closeRequested()

  radius: Style.cornerRadius
  color: Qt.rgba(Color.menu.text.r, Color.menu.text.g, Color.menu.text.b, 0.05)
  border.color: Color.menu.border
  border.width: Math.max(1, Style.space(1))

  function loadReminder() {
    if (!reminder)
      return
    titleField.text = reminder.title || ""
    notesField.text = reminder.notes || ""
    urlField.text = reminder.url || ""
    priorityField.currentIndex = Math.max(0, priorityField.model.indexOf(reminder.priority || "none"))
    flaggedField.checked = reminder.flagged === true
    tagsField.text = (reminder.tags || []).join(", ")
    var wantedAssignee = reminder.assignee || ""
    var assigneeIndex = 0
    for (var i = 0; i < assigneeChoices.length; i++) {
      if (String(assigneeChoices[i].ship || "") === String(wantedAssignee)) {
        assigneeIndex = i
        break
      }
    }
    assigneeField.currentIndex = assigneeIndex
    dueField.text = TendModel.scheduleInputValue(reminder.schedule)
    allDayField.checked = reminder.schedule ? reminder.schedule.allDay === true : false
  }

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

        Text {
          width: parent.width - closeButton.width
          anchors.verticalCenter: parent.verticalCenter
          text: "DETAILS"
          color: Color.menu.text
          font.family: Style.font.menuFamily
          font.pixelSize: Style.font.caption
          font.bold: true
        }

        TendButton {
          id: closeButton
          text: "󰅖"
          bordered: false
          tooltipText: "Close details"
          onClicked: root.closeRequested()
        }
      }

      TextField {
        id: titleField
        width: parent.width
        placeholderText: "Reminder title"
        Accessible.name: "Reminder title"
      }

      QQC.TextArea {
        id: notesField
        width: parent.width
        height: Style.space(86)
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

      TendDropdown {
        id: assigneeField
        width: parent.width
        label: "Assigned to"
        model: root.assigneeChoices
        textRole: "title"
        valueRole: "ship"
      }

      TextField {
        id: tagsField
        width: parent.width
        placeholderText: "Tags, separated by commas"
        Accessible.name: "Reminder tags"
      }

      PanelSeparator { width: parent.width }

      Text {
        text: "SCHEDULE"
        color: Color.menu.text
        opacity: 0.68
        font.family: Style.font.menuFamily
        font.pixelSize: Style.font.caption
        font.bold: true
      }

      TextField {
        id: dueField
        width: parent.width
        placeholderText: "YYYY-MM-DDTHH:MM"
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

      PanelSeparator { width: parent.width }

      Row {
        width: parent.width
        spacing: Style.space(8)

        TendButton {
          id: saveButton
          width: parent.width - deleteButton.width - parent.spacing
          text: "Save changes"
          enabled: root.editable && root.reminder && titleField.text.trim() && root.service && root.service.connectionState === "online" && !root.service.mutationPending
          onClicked: {
            var tags = tagsField.text.split(",").map(function(tag) { return tag.trim() }).filter(function(tag) { return tag.length > 0 })
            root.service.updateReminder(root.list.id, root.reminder.id, {
              title: titleField.text,
              notes: notesField.text,
              url: urlField.text.trim() || null,
              priority: priorityField.currentText,
              flagged: flaggedField.checked,
              tags: tags,
              assignee: assigneeField.currentValue || null
            }, root.list.revision)
          }
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
