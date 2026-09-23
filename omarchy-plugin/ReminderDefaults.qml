import QtQuick
import QtQuick.Controls as QQC
import qs.Commons
import qs.Ui

BorderSurface {
  id: root

  property var service: null
  signal closeRequested()

  color: Color.popups.background
  borderSpec: Border.localOrSurfaceSpec("popups", "border", Color.popups.border, Color.menu.border, Style.normalBorderWidth)
  radius: Style.cornerRadius

  function loadPreferences() {
    if (!service) return
    badgeMode.currentIndex = Math.max(0, badgeMode.model.indexOf(service.preferences.badgeMode))
    allDayHour.value = Math.floor(service.preferences.allDayAlertMinute / 60)
    allDayMinute.value = service.preferences.allDayAlertMinute % 60
    allDayOverdue.checked = service.preferences.allDayOverdue
  }

  onVisibleChanged: if (visible) loadPreferences()
  onServiceChanged: loadPreferences()

  Connections {
    target: root.service
    function onPreferencesChanged() { root.loadPreferences() }
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
            text: "REMINDER DEFAULTS"
            color: Color.popups.text
            font.family: Style.font.family
            font.pixelSize: Style.font.caption
            font.bold: true
          }

          Text {
            width: parent.width
            text: "Applies to notifications and the menubar badge."
            color: Qt.darker(Color.popups.text, 1.45)
            wrapMode: Text.WordWrap
            font.family: Style.font.family
            font.pixelSize: Style.font.caption
          }
        }

        TendButton {
          id: closeButton
          text: "󰅖"
          bordered: false
          tooltipText: "Close reminder defaults"
          onClicked: root.closeRequested()
        }
      }

      PanelSeparator { width: parent.width; foreground: Color.popups.text }

      TendDropdown {
        id: badgeMode
        width: parent.width
        label: "Menubar badge"
        model: ["today", "all", "assigned", "none"]
        Accessible.name: "Menubar badge count"
      }

      Row {
        width: parent.width
        spacing: Style.space(8)

        TendNumber {
          id: allDayHour
          width: (parent.width - parent.spacing) / 2
          label: "All-day alert hour"
          from: 0
          to: 23
          Accessible.name: "All-day reminder hour"
        }

        TendNumber {
          id: allDayMinute
          width: (parent.width - parent.spacing) / 2
          label: "Minute"
          from: 0
          to: 59
          Accessible.name: "All-day reminder minute"
        }
      }

      TendCheck {
        id: allDayOverdue
        width: parent.width
        text: "Keep all-day reminders overdue"
      }

      TendButton {
        width: parent.width
        text: root.service && root.service.mutationPending ? "Saving…" : "Save defaults"
        enabled: root.service && root.service.connectionState === "online" && !root.service.mutationPending
        onClicked: root.service.setReminderPolicy({
          badgeMode: badgeMode.currentText,
          allDayAlertMinute: allDayHour.value * 60 + allDayMinute.value,
          allDayOverdue: allDayOverdue.checked
        })
      }
    }
  }
}
