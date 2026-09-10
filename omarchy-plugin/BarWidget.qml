import QtQuick
import qs.Commons
import qs.Ui

BarWidget {
  id: root
  moduleName: "io.omabit.tend"

  readonly property var tendService: bar?.shell?.serviceFor("io.omabit.tend")
  readonly property int taskCount: tendService ? tendService.incompleteCount : 0
  readonly property string state: tendService ? tendService.connectionState : "disconnected"

  implicitWidth: row.implicitWidth + Style.space(12)
  implicitHeight: barSize

  Row {
    id: row
    anchors.centerIn: parent
    spacing: Style.space(5)

    Text {
      text: root.state === "online" ? "󰄬" : (root.state === "checking" ? "󰔟" : "󰅖")
      color: root.bar.barForeground
      font.family: root.bar.fontFamily
      font.pixelSize: Style.font.body
    }

    Text {
      text: String(root.taskCount)
      color: root.bar.barForeground
      font.family: root.bar.fontFamily
      font.pixelSize: Style.font.body
      font.bold: root.taskCount > 0
    }
  }

  MouseArea {
    anchors.fill: parent
    cursorShape: Qt.PointingHandCursor
    onClicked: if (root.bar && root.bar.shell) root.bar.shell.toggle("io.omabit.tend", "{}")
    onEntered: {
      if (!root.bar) return
      var next = root.tendService ? root.tendService.nextReminder : null
      var suffix = next ? " · Next: " + next.title : ""
      root.bar.showTooltip(root, "Tend · " + root.taskCount + " incomplete · " + root.state + suffix)
    }
    onExited: if (root.bar) root.bar.hideTooltip(root)
  }
}
