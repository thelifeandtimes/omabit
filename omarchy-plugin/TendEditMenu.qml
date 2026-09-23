import QtQuick
import QtQuick.Controls as QQC
import qs.Commons
import qs.Ui

QQC.Menu {
  id: root

  required property var target

  width: Style.space(190)
  padding: Style.space(4)

  background: BorderSurface {
    color: Qt.rgba(Color.menu.background.r, Color.menu.background.g, Color.menu.background.b, 1)
    borderSpec: Border.localOrSurfaceSpec("popups", "border", Color.popups.border, Color.menu.border, Style.normalBorderWidth)
    radius: Style.cornerRadius
  }

  TendMenuItem { text: "Undo"; enabled: root.target && root.target.canUndo; onTriggered: root.target.undo() }
  TendMenuItem { text: "Redo"; enabled: root.target && root.target.canRedo; onTriggered: root.target.redo() }
  QQC.MenuSeparator {}
  TendMenuItem { text: "Cut"; enabled: root.target && !root.target.readOnly && root.target.selectedText.length > 0; onTriggered: root.target.cut() }
  TendMenuItem { text: "Copy"; enabled: root.target && root.target.selectedText.length > 0; onTriggered: root.target.copy() }
  TendMenuItem { text: "Paste"; enabled: root.target && !root.target.readOnly && root.target.canPaste; onTriggered: root.target.paste() }
  TendMenuItem {
    text: "Delete"
    enabled: root.target && !root.target.readOnly && root.target.selectedText.length > 0
    onTriggered: root.target.remove(root.target.selectionStart, root.target.selectionEnd)
  }
  QQC.MenuSeparator {}
  TendMenuItem { text: "Select all"; enabled: root.target && root.target.length > 0; onTriggered: root.target.selectAll() }
}
