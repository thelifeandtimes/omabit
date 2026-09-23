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

  function open() { search.text = ""; filterOptions(); popup.open() }

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
      onClicked: root.open()
    }
    Keys.onReturnPressed: root.open()
    Keys.onEnterPressed: root.open()
    Keys.onSpacePressed: root.open()
  }

  QQC.Popup {
    id: popup
    x: Math.max(0, root.width - width)
    y: root.height + Style.space(4)
    width: Math.max(root.width, Style.space(250))
    height: Math.min(Style.space(310), Style.space(64) + results.count * Style.space(38))
    padding: Style.space(1)
    focus: true
    closePolicy: QQC.Popup.CloseOnEscape | QQC.Popup.CloseOnPressOutside

    background: BorderSurface {
      color: root.background
      borderSpec: Border.localOrSurfaceSpec("popups", "border", Color.popups.border, Color.menu.border, Style.normalBorderWidth)
      radius: Style.cornerRadius
    }

    contentItem: Column {
      spacing: Style.space(2)

      TextField {
        id: search
        width: parent.width
        placeholderText: "Find a person"
        onTextChanged: root.filterOptions()
      }

      ListView {
        id: results
        width: parent.width
        height: parent.height - search.height - parent.spacing
        clip: true
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

    onOpened: Qt.callLater(function() { search.forceActiveFocus() })
  }
}
