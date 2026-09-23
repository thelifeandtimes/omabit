import QtQuick
import QtQuick.Controls as QQC
import qs.Commons
import qs.Ui

Item {
  id: root

  property string value: ""
  property bool allDay: false
  property string placeholderText: "Due date"
  property bool iconOnly: false
  property color foreground: Color.foreground
  property color background: Color.popups.background
  property color accent: Color.accent
  signal committed(string value, bool allDay)

  property int shownYear: new Date().getFullYear()
  property int shownMonth: new Date().getMonth()
  property int selectedDay: new Date().getDate()
  property int selectedHour: 9
  property int selectedMinute: 0

  implicitWidth: iconOnly ? Style.spacing.controlHeight : Style.space(190)
  implicitHeight: Style.spacing.controlHeight

  function pad2(number) { return number < 10 ? "0" + number : String(number) }

  function normalizedValue() {
    var date = shownYear + "-" + pad2(shownMonth + 1) + "-" + pad2(selectedDay)
    return allDay ? date : date + "T" + pad2(selectedHour) + ":" + pad2(selectedMinute)
  }

  function displayValue() {
    if (!value) return placeholderText
    var parts = String(value).split("T")
    if (parts.length === 1 || allDay) return parts[0] + " · all day"
    return parts[0] + " · " + parts[1]
  }

  function loadValue() {
    var source = String(value || "")
    var parsed = source ? new Date(source.length === 10 ? source + "T09:00:00" : source) : new Date()
    if (isNaN(parsed.getTime())) parsed = new Date()
    shownYear = parsed.getFullYear()
    shownMonth = parsed.getMonth()
    selectedDay = parsed.getDate()
    selectedHour = parsed.getHours()
    selectedMinute = parsed.getMinutes()
    allDay = source.length === 10
  }

  function daysInMonth(year, month) { return new Date(year, month + 1, 0).getDate() }
  function firstWeekday(year, month) { return new Date(year, month, 1).getDay() }
  function monthTitle() {
    return new Date(shownYear, shownMonth, 1).toLocaleString(Qt.locale(), "MMMM yyyy")
  }
  function placePopup() {
    if (!picker.parent) return
    var point = root.mapToItem(picker.parent, 0, 0)
    var gap = Style.space(4)
    var margin = Style.space(8)
    var maxX = Math.max(margin, picker.parent.width - picker.width - margin)
    picker.x = Math.max(margin, Math.min(point.x + root.width - picker.width, maxX))
    var below = point.y + root.height + gap
    var above = point.y - picker.height - gap
    var roomBelow = picker.parent.height - below - margin
    picker.y = roomBelow >= picker.height || above < margin
      ? Math.max(margin, Math.min(below, picker.parent.height - picker.height - margin))
      : Math.max(margin, above)
  }

  function open() {
    loadValue()
    picker.open()
    Qt.callLater(root.placePopup)
  }

  TendButton {
    id: trigger
    anchors.fill: parent
    text: ""
    bordered: true
    Accessible.name: root.value ? "Due " + root.displayValue() : root.placeholderText
    onClicked: root.open()

    Row {
      anchors.left: parent.left
      anchors.right: parent.right
      anchors.verticalCenter: parent.verticalCenter
      anchors.leftMargin: Style.spacing.controlPaddingX
      anchors.rightMargin: Style.spacing.controlPaddingX
      spacing: Style.space(6)

      Text {
        visible: !root.iconOnly
        width: parent.width - calendarGlyph.width - parent.spacing
        anchors.verticalCenter: parent.verticalCenter
        text: root.displayValue()
        textFormat: Text.PlainText
        color: root.value ? root.foreground : Qt.darker(root.foreground, 1.55)
        elide: Text.ElideRight
        font.family: Style.font.family
        font.pixelSize: Style.font.body
      }

      Text {
        id: calendarGlyph
        anchors.verticalCenter: parent.verticalCenter
        text: "󰃭"
        color: root.foreground
        opacity: 0.72
        font.family: Style.font.family
        font.pixelSize: Style.font.icon
      }
    }
  }

  QQC.Popup {
    id: picker
    parent: QQC.Overlay.overlay
    x: 0
    y: root.height + Style.space(4)
    width: Math.min(Style.space(326), parent ? Math.max(Style.space(260), parent.width - Style.space(16)) : Style.space(326))
    height: Math.min(Style.space(394), parent ? Math.max(0, parent.height - Style.space(16)) : Style.space(394))
    padding: Style.space(12)
    focus: true
    closePolicy: QQC.Popup.CloseOnEscape | QQC.Popup.CloseOnPressOutside
    onOpened: Qt.callLater(root.placePopup)

    background: BorderSurface {
      color: root.background
      borderSpec: Border.localOrSurfaceSpec("popups", "border", Color.popups.border, Color.menu.border, Style.normalBorderWidth)
      radius: Style.cornerRadius
    }

    contentItem: Column {
      spacing: Style.space(8)

      Row {
        width: parent.width
        spacing: Style.space(6)

        TendButton {
          text: "‹"
          bordered: false
          onClicked: {
            root.shownMonth--
            if (root.shownMonth < 0) { root.shownMonth = 11; root.shownYear-- }
            root.selectedDay = Math.min(root.selectedDay, root.daysInMonth(root.shownYear, root.shownMonth))
          }
        }

        Text {
          width: parent.width - parent.children[0].width - parent.children[2].width - parent.spacing * 2
          anchors.verticalCenter: parent.verticalCenter
          horizontalAlignment: Text.AlignHCenter
          text: root.monthTitle()
          color: root.foreground
          font.family: Style.font.family
          font.pixelSize: Style.font.body
          font.bold: true
        }

        TendButton {
          text: "›"
          bordered: false
          onClicked: {
            root.shownMonth++
            if (root.shownMonth > 11) { root.shownMonth = 0; root.shownYear++ }
            root.selectedDay = Math.min(root.selectedDay, root.daysInMonth(root.shownYear, root.shownMonth))
          }
        }
      }

      Grid {
        width: parent.width
        columns: 7
        columnSpacing: Style.space(2)
        rowSpacing: Style.space(2)

        Repeater {
          model: ["S", "M", "T", "W", "T", "F", "S"]
          delegate: Text {
            required property string modelData
            width: (parent.width - parent.columnSpacing * 6) / 7
            height: Style.space(24)
            text: modelData
            horizontalAlignment: Text.AlignHCenter
            verticalAlignment: Text.AlignVCenter
            color: root.foreground
            opacity: 0.56
            font.family: Style.font.family
            font.pixelSize: Style.font.caption
          }
        }

        Repeater {
          model: 42
          delegate: TendButton {
            required property int index
            readonly property int dayNumber: index - root.firstWeekday(root.shownYear, root.shownMonth) + 1
            width: (parent.width - parent.columnSpacing * 6) / 7
            height: Style.space(30)
            text: dayNumber > 0 && dayNumber <= root.daysInMonth(root.shownYear, root.shownMonth) ? String(dayNumber) : ""
            bordered: false
            checkable: true
            checked: text !== "" && dayNumber === root.selectedDay
            enabled: text !== ""
            onClicked: root.selectedDay = dayNumber
          }
        }
      }

      PanelSeparator { width: parent.width; foreground: root.foreground }

      Row {
        width: parent.width
        spacing: Style.space(8)

        TendNumber {
          width: (parent.width - allDayBox.width - parent.spacing * 2) * 0.5
          label: "Hour"
          from: 0
          to: 23
          value: root.selectedHour
          enabled: !root.allDay
          onValueChanged: if (activeFocus) root.selectedHour = value
        }

        TendNumber {
          width: (parent.width - allDayBox.width - parent.spacing * 2) * 0.5
          label: "Minute"
          from: 0
          to: 59
          value: root.selectedMinute
          enabled: !root.allDay
          onValueChanged: if (activeFocus) root.selectedMinute = value
        }

        TendCheckbox {
          id: allDayBox
          anchors.verticalCenter: parent.verticalCenter
          text: "All day"
          checked: root.allDay
          onClicked: root.allDay = !root.allDay
        }
      }

      Row {
        width: parent.width
        spacing: Style.space(8)

        TendButton {
          text: "Clear"
          onClicked: {
            root.value = ""
            root.allDay = false
            root.committed("", false)
            picker.close()
          }
        }

        TendButton {
          width: parent.width - parent.children[0].width - parent.spacing
          text: "Set due date"
          onClicked: {
            root.value = root.normalizedValue()
            root.committed(root.value, root.allDay)
            picker.close()
          }
        }
      }
    }
  }
}
