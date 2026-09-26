import QtQuick
import QtQuick.Controls as QQC
import qs.Commons
import qs.Ui

Item {
  id: root

  property var options: []
  property string value: ""
  property string placeholderText: "Assignee"
  property color foreground: Color.foreground
  property color background: Color.popups.background
  property color accent: Color.accent
  property var filteredOptions: []
  readonly property bool popupOpen: popup.opened
  signal changed(string value)

  implicitWidth: Style.space(180)
  implicitHeight: Style.spacing.controlHeight

  function selectedOption() {
    for (var i = 0; i < options.length; i++)
      if (String(options[i].value || "") === String(value || "")) return options[i]
    return null
  }

  function filterOptions() {
    var query = String(search.text || "").toLowerCase()
    filteredOptions = (options || []).filter(function(option) {
      if (!query) return true
      return [option.label, option.role, option.value].join(" ").toLowerCase().indexOf(query) !== -1
    })
  }

  function placePopup() {
    if (!popup.parent) return
    var point = root.mapToItem(popup.parent, 0, 0)
    var gap = Style.space(4)
    var margin = Style.space(8)
    var maxX = Math.max(margin, popup.parent.width - popup.width - margin)
    popup.x = Math.max(margin, Math.min(point.x, maxX))
    var below = point.y + root.height + gap
    var above = point.y - popup.height - gap
    var roomBelow = popup.parent.height - below - margin
    popup.y = roomBelow >= popup.height || above < margin
      ? Math.max(margin, Math.min(below, popup.parent.height - popup.height - margin))
      : Math.max(margin, above)
  }

  function open() {
    search.text = ""
    filterOptions()
    popup.open()
    Qt.callLater(root.placePopup)
  }
  function close() { popup.close() }
  function toggle() { popup.opened ? popup.close() : root.open() }

  onOptionsChanged: filterOptions()
  Component.onCompleted: filterOptions()

  BorderSurface {
    id: trigger
    anchors.fill: parent
    color: Style.controlFill(activeFocus, hover.hovered, root.foreground, root.accent)
    borderSpec: Border.controlSpec(activeFocus ? "focus" : (hover.hovered ? "hover-cursor" : "normal"), root.foreground, root.accent)
    radius: Style.cornerRadius
    activeFocusOnTab: true

    Row {
      anchors.left: parent.left
      anchors.right: parent.right
      anchors.verticalCenter: parent.verticalCenter
      anchors.margins: Style.spacing.controlPaddingX
      spacing: Style.space(5)

      Text {
        readonly property var option: root.selectedOption()
        text: option ? String(option.label || option.value || "") : root.placeholderText
        color: option ? root.foreground : Qt.darker(root.foreground, 1.55)
        font.family: Style.font.family
        font.pixelSize: Style.font.body
        elide: Text.ElideRight
      }

      Text {
        readonly property var option: root.selectedOption()
        width: parent.width - x - chevron.width - parent.spacing
        text: option && option.role ? "(" + option.role + ")" : ""
        color: Qt.darker(root.foreground, 1.5)
        font.family: Style.font.family
        font.pixelSize: Style.font.caption
        elide: Text.ElideRight
      }

      Text {
        id: chevron
        text: "󰅀"
        color: root.foreground
        opacity: 0.65
        font.family: Style.font.family
        font.pixelSize: Style.font.body
      }
    }

    HoverHandler { id: hover }
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

  QQC.Popup {
    id: popup
    parent: trigger.Window.window ? trigger.Window.window.contentItem : trigger
    x: 0
    y: root.height + Style.space(4)
    width: Math.min(root.width, parent ? Math.max(Style.space(120), parent.width - Style.space(16)) : root.width)
    height: Math.min(contentColumn.implicitHeight + topPadding + bottomPadding, parent ? Math.max(Style.space(80), parent.height - Style.space(16)) : contentColumn.implicitHeight + topPadding + bottomPadding)
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

    contentItem: Column {
      id: contentColumn
      spacing: Style.space(2)

      TendTextField {
        id: search
        width: parent.width
        placeholderText: "Find a person"
        onTextChanged: root.filterOptions()
      }

      ListView {
        id: results
        width: parent.width
        height: implicitHeight
        implicitHeight: Math.min(6, count) * Style.space(36) + Math.max(0, Math.min(6, count) - 1) * spacing
        clip: true
        boundsBehavior: Flickable.StopAtBounds
        model: root.filteredOptions
        spacing: Style.space(2)

        delegate: Rectangle {
          required property var modelData
          width: ListView.view.width
          height: Style.space(36)
          radius: Style.cornerRadius
          color: rowHover.hovered ? Style.hoverFillFor(root.foreground, root.accent) : "transparent"

          Row {
            anchors.left: parent.left
            anchors.right: parent.right
            anchors.verticalCenter: parent.verticalCenter
            anchors.margins: Style.spacing.controlPaddingX
            spacing: Style.space(5)

            Text {
              text: String(modelData.label || modelData.value || "")
              color: root.foreground
              font.family: Style.font.family
              font.pixelSize: Style.font.body
            }

            Text {
              width: parent.width - x
              text: modelData.role ? "(" + modelData.role + ")" : ""
              color: Qt.darker(root.foreground, 1.5)
              font.family: Style.font.family
              font.pixelSize: Style.font.caption
              elide: Text.ElideRight
            }
          }

          HoverHandler { id: rowHover }
          MouseArea {
            anchors.fill: parent
            cursorShape: Qt.PointingHandCursor
            onClicked: {
              root.value = String(modelData.value || "")
              root.changed(root.value)
              popup.close()
            }
          }
        }
      }
    }

    onOpened: Qt.callLater(function() { root.placePopup(); search.forceActiveFocus() })
  }
}
