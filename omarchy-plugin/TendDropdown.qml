import QtQuick
import QtQuick.Controls as QQC
import qs.Commons
import qs.Ui

Item {
  id: root

  property var model: []
  property string label: ""
  property bool showLabel: label !== ""
  property string textRole: ""
  property string valueRole: ""
  property int currentIndex: model && model.length ? 0 : -1
  property string displayText: ""
  property color foreground: Color.popups.text
  property color background: Color.popups.background
  property color accent: Color.accent
  readonly property int count: model && model.length ? model.length : 0
  readonly property string currentText: currentIndex >= 0 && currentIndex < count ? labelFor(model[currentIndex]) : ""
  readonly property var currentValue: currentIndex >= 0 && currentIndex < count ? valueFor(model[currentIndex]) : null
  readonly property bool popupOpen: popup.opened
  signal activated(int index)

  function labelFor(entry) {
    if (textRole && entry && typeof entry === "object" && entry[textRole] !== undefined)
      return String(entry[textRole])
    return String(entry === undefined || entry === null ? "" : entry)
  }

  function valueFor(entry) {
    if (valueRole && entry && typeof entry === "object" && entry[valueRole] !== undefined)
      return entry[valueRole]
    return entry
  }

  function chooseIndex(index) {
    if (index < 0 || index >= count) return
    currentIndex = index
    activated(index)
    popup.close()
  }

  function placePopup() {
    if (!popup.parent) return
    var point = root.mapToItem(popup.parent, 0, 0)
    var gap = Style.space(4)
    var margin = Style.space(8)
    var maxX = Math.max(margin, popup.parent.width - popup.width - margin)
    popup.x = Math.max(margin, Math.min(point.x, maxX))
    var below = point.y + trigger.y + trigger.height + gap
    var above = point.y + trigger.y - popup.height - gap
    var roomBelow = popup.parent.height - below - margin
    popup.y = roomBelow >= popup.height || above < margin
      ? Math.max(margin, Math.min(below, popup.parent.height - popup.height - margin))
      : Math.max(margin, above)
  }

  function open() {
    popup.open()
    Qt.callLater(root.placePopup)
  }
  function close() { popup.close() }
  function toggle() { popup.opened ? popup.close() : root.open() }

  implicitWidth: Style.spacing.dropdownWidth
  implicitHeight: showLabel && label !== "" ? Style.spacing.controlHeight + Style.spacing.huge : Style.spacing.controlHeight

  Column {
    anchors.fill: parent
    spacing: Style.spacing.labelGap

    Text {
      visible: root.showLabel && root.label !== ""
      text: root.label
      textFormat: Text.PlainText
      color: Qt.darker(root.foreground, 1.4)
      font.family: Style.font.family
      font.pixelSize: Style.font.caption
      font.bold: true
    }

    BorderSurface {
      id: trigger
      width: parent.width
      height: Style.spacing.controlHeight
      radius: Style.cornerRadius
      color: Style.controlFill(activeFocus, triggerHover.hovered, root.foreground, root.accent)
      borderSpec: Border.controlSpec(activeFocus ? "focus" : (triggerHover.hovered ? "hover-cursor" : "normal"), root.foreground, root.accent)
      activeFocusOnTab: true

      Row {
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.verticalCenter: parent.verticalCenter
        anchors.leftMargin: Style.spacing.controlPaddingX
        anchors.rightMargin: Style.spacing.controlPaddingX
        spacing: Style.space(6)

        Text {
          width: parent.width - chevron.width - parent.spacing
          anchors.verticalCenter: parent.verticalCenter
          text: root.displayText || root.currentText
          textFormat: Text.PlainText
          color: root.foreground
          font.family: Style.font.family
          font.pixelSize: Style.font.body
          elide: Text.ElideRight
        }

        OpticalGlyph {
          id: chevron
          width: Style.font.body
          height: Style.font.body
          anchors.verticalCenter: parent.verticalCenter
          text: "󰅀"
          color: root.foreground
          opacity: 0.65
          fontSize: Style.font.body
        }
      }

      HoverHandler { id: triggerHover }
      MouseArea {
        anchors.fill: parent
        cursorShape: Qt.PointingHandCursor
        onClicked: { trigger.forceActiveFocus(); root.toggle() }
      }
      Keys.onPressed: function(event) {
        if (event.key === Qt.Key_Return || event.key === Qt.Key_Enter || event.key === Qt.Key_Space || event.key === Qt.Key_Down) {
          root.toggle()
          event.accepted = true
        } else if (event.key === Qt.Key_Escape && popup.opened) {
          popup.close()
          event.accepted = true
        }
      }
    }
  }

  QQC.Popup {
    id: popup
    parent: trigger.Window.window ? trigger.Window.window.contentItem : trigger
    width: Math.min(root.width, parent ? Math.max(Style.space(120), parent.width - Style.space(16)) : root.width)
    height: Math.min(contentList.contentHeight + topPadding + bottomPadding, Style.spacing.popupRowHeight * 8 + Style.space(16))
    padding: Style.space(1)
    focus: true
    modal: true
    dim: false
    closePolicy: QQC.Popup.CloseOnEscape | QQC.Popup.CloseOnPressOutside

    background: BorderSurface {
      color: root.background
      borderSpec: Border.localOrSurfaceSpec("popups", "border", Color.popups.border, Color.menu.border, Style.normalBorderWidth)
      radius: Style.cornerRadius
    }

    contentItem: ListView {
      id: contentList
      clip: true
      boundsBehavior: Flickable.StopAtBounds
      spacing: Style.spacing.labelGap
      model: root.model
      currentIndex: root.currentIndex

      Keys.priority: Keys.BeforeItem
      Keys.onPressed: function(event) {
        if (event.key === Qt.Key_Escape) {
          popup.close()
          event.accepted = true
        } else if (event.key === Qt.Key_Down || event.text === "j") {
          currentIndex = Math.min(root.count - 1, currentIndex + 1)
          event.accepted = true
        } else if (event.key === Qt.Key_Up || event.text === "k") {
          currentIndex = Math.max(0, currentIndex - 1)
          event.accepted = true
        } else if (event.key === Qt.Key_Return || event.key === Qt.Key_Enter) {
          root.chooseIndex(currentIndex)
          event.accepted = true
        }
      }

      delegate: Rectangle {
        required property var modelData
        required property int index
        width: ListView.view.width
        height: Style.spacing.popupRowHeight
        radius: Style.cornerRadius
        color: index === contentList.currentIndex ? Style.hoverFillFor(root.foreground, root.accent) : "transparent"

        Text {
          anchors.left: parent.left
          anchors.right: parent.right
          anchors.verticalCenter: parent.verticalCenter
          anchors.leftMargin: Style.spacing.controlPaddingX
          anchors.rightMargin: Style.spacing.controlPaddingX
          text: root.labelFor(modelData)
          textFormat: Text.PlainText
          color: index === contentList.currentIndex ? Style.hoverStateColor(root.foreground, root.accent) : root.foreground
          font.family: Style.font.family
          font.pixelSize: Style.font.body
          elide: Text.ElideRight
        }

        HoverHandler {
          id: rowHover
          onHoveredChanged: if (hovered) contentList.currentIndex = parent.index
        }
        MouseArea {
          anchors.fill: parent
          cursorShape: Qt.PointingHandCursor
          onClicked: root.chooseIndex(parent.index)
        }
      }
    }

    onOpened: {
      contentList.currentIndex = Math.max(0, root.currentIndex)
      Qt.callLater(function() { root.placePopup(); contentList.forceActiveFocus() })
    }
  }
}
