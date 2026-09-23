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
  readonly property var assignedReminders: TendModel.queryReminders(tendService ? tendService.lists : [], {
    view: "assigned",
    ship: tendService ? tendService.ship : "",
    sort: "priority"
  })
  readonly property int taskCount: assignedReminders.length
  readonly property var destinationLists: tendService
    ? TendModel.orderedLists(tendService.lists, tendService.preferences.pinnedLists, tendService.localSettings.listOrder)
    : []
  readonly property color dim: Qt.rgba(barForeground.r, barForeground.g, barForeground.b, 0.6)

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

  function parseQuickEntry(value) {
    var tokens = String(value || "").trim().split(/\s+/)
    var tags = []
    var words = []
    for (var i = 0; i < tokens.length; i++) {
      if (tokens[i].charAt(0) === "#" && tokens[i].length > 1)
        tags.push(tokens[i].substring(1))
      else
        words.push(tokens[i])
    }
    return { title: words.join(" ").trim(), tags: tags }
  }

  function addReminder() {
    if (!tendService || destinationList.currentIndex < 0 || !quickAdd.text.trim()) return
    var list = destinationLists[destinationList.currentIndex]
    if (!list) return
    var parsed = parseQuickEntry(quickAdd.text)
    var title = parsed.title || quickAdd.text.trim()
    var tags = parsed.tags
    if (tendService.addReminderAssignedToMe(list.id, title, list.revision, tags)) quickAdd.text = ""
  }

  visible: true
  implicitWidth: widgetButton.implicitWidth
  implicitHeight: widgetButton.implicitHeight

  onOpenedChanged: if (opened) Qt.callLater(function() { quickAdd.forceActiveFocus() })

  WidgetButton {
    id: widgetButton
    anchors.fill: parent
    bar: root.bar
    labelVisible: false
    hasVisualContent: true
    fixedWidth: vertical ? barSize : widgetContents.implicitWidth + Style.space(16)
    fixedHeight: vertical ? barSize : -1
    tooltipText: "Tend · " + taskCount + " assigned · " + state
    Accessible.role: Accessible.Button
    Accessible.name: "Open Tend assigned reminders; " + taskCount + " incomplete; " + state
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
    contentWidth: popup.fittedContentWidth(Style.space(440))
    contentHeight: popup.cappedContentHeight(root.assignedReminders.length > 0
      ? Style.space(250 + Math.min(8, root.assignedReminders.length) * 46)
      : Style.space(300))

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
          spacing: Style.space(8)

          Column {
            Layout.fillWidth: true
            spacing: Style.space(2)

            Text {
              width: parent.width
              text: "TEND"
              color: root.barForeground
              font.family: root.bar ? root.bar.fontFamily : Style.font.family
              font.pixelSize: Style.font.caption
              font.bold: true
            }

            Text {
              width: parent.width
              text: "~" + (root.tendService ? root.tendService.ship : "") + " · " + root.state
              color: root.dim
              font.family: root.bar ? root.bar.fontFamily : Style.font.family
              font.pixelSize: Style.font.caption
            }
          }

          Button {
            text: "Open Tend"
            iconText: "󰈔"
            focusable: true
            bordered: true
            onClicked: root.openFullPanel(0, 0)
          }
        }

        PanelSeparator { Layout.fillWidth: true; foreground: root.barForeground }

        PanelSectionHeader {
          Layout.fillWidth: true
          text: "QUICK ADD"
          foreground: root.barForeground
        }

        RowLayout {
          Layout.fillWidth: true
          spacing: Style.space(7)

          TextField {
            id: quickAdd
            Layout.fillWidth: true
            placeholderText: "New reminder"
            enabled: destinationList.currentIndex >= 0 && root.tendService && root.tendService.connectionState === "online" && !root.tendService.mutationPending
            onAccepted: root.addReminder()
          }

          TendDropdown {
            id: destinationList
            Layout.preferredWidth: Style.space(132)
            model: root.destinationLists
            textRole: "title"
            Accessible.name: "Destination list"
          }

          Button {
            text: "Add"
            focusable: true
            bordered: true
            enabled: quickAdd.enabled && quickAdd.text.trim() !== ""
            onClicked: root.addReminder()
          }
        }

        PanelSeparator { Layout.fillWidth: true; foreground: root.barForeground }

        PanelSectionHeader {
          Layout.fillWidth: true
          text: "ASSIGNED TO ME · PRIORITY ORDER"
          foreground: root.barForeground
        }

        Column {
          Layout.fillWidth: true
          spacing: Style.space(4)

          Text {
            visible: root.assignedReminders.length === 0
            width: parent.width
            text: root.state === "online" ? "Nothing assigned to you." : "Connect Tend to load assigned reminders."
            color: root.dim
            font.family: root.bar ? root.bar.fontFamily : Style.font.family
            font.pixelSize: Style.font.body
            wrapMode: Text.WordWrap
          }

          Repeater {
            model: root.assignedReminders.slice(0, 8)

            delegate: Rectangle {
              required property var modelData
              width: parent.width
              height: Style.space(46)
              radius: Style.cornerRadius
              color: rowMouse.hovered
                ? Style.hoverFillFor(root.barForeground, Color.accent)
                : Qt.rgba(root.barForeground.r, root.barForeground.g, root.barForeground.b, 0.045)

              Row {
                anchors.fill: parent
                anchors.margins: Style.space(7)
                spacing: Style.space(8)

                Button {
                  anchors.verticalCenter: parent.verticalCenter
                  text: "○"
                  tooltipText: "Complete"
                  focusable: true
                  bordered: false
                  enabled: root.tendService && root.tendService.canEditList(modelData.listId) && root.state === "online" && !root.tendService.mutationPending
                  onClicked: root.tendService.setCompleted(modelData.listId, modelData.id, true, modelData.listRevision)
                }

                Text {
                  width: Style.space(26)
                  anchors.verticalCenter: parent.verticalCenter
                  text: root.priorityGlyph(modelData.priority)
                  color: root.priorityColor(modelData.priority)
                  font.family: root.bar ? root.bar.fontFamily : Style.font.family
                  font.pixelSize: Style.font.caption
                  font.bold: true
                }

                Column {
                  width: parent.width - x
                  anchors.verticalCenter: parent.verticalCenter
                  spacing: 0

                  Text {
                    width: parent.width
                    text: modelData.title
                    color: root.barForeground
                    font.family: root.bar ? root.bar.fontFamily : Style.font.family
                    font.pixelSize: Style.font.body
                    elide: Text.ElideRight
                  }

                  Text {
                    width: parent.width
                    text: modelData.listTitle
                    color: root.dim
                    font.family: root.bar ? root.bar.fontFamily : Style.font.family
                    font.pixelSize: Style.font.caption
                    elide: Text.ElideRight
                  }

                  MouseArea {
                    anchors.fill: parent
                    hoverEnabled: true
                    cursorShape: Qt.PointingHandCursor
                    onClicked: root.openFullPanel(modelData.listId, modelData.id)
                  }
                }
              }

              HoverHandler { id: rowMouse }
            }
          }

          Text {
            visible: root.assignedReminders.length > 8
            width: parent.width
            text: "+ " + (root.assignedReminders.length - 8) + " more"
            color: root.dim
            font.family: root.bar ? root.bar.fontFamily : Style.font.family
            font.pixelSize: Style.font.caption
          }
        }
      }
    }
  }
}
