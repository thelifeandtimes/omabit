import QtQuick
import QtQuick.Controls as Controls
import qs.Commons

Controls.ComboBox {
  id: root
  implicitHeight: Style.space(38)
  leftPadding: Style.space(12)
  rightPadding: Style.space(32)
  hoverEnabled: true
  contentItem: Label { text: root.displayText; verticalAlignment: Text.AlignVCenter; elide: Text.ElideRight; wrapMode: Text.NoWrap }
  indicator: Label { text: "⌄"; x: root.width - width - Style.space(12); y: (root.height - height) / 2; secondary: true }
  background: Rectangle { radius: Style.space(6); color: root.hovered ? Design.raised : Design.surface; border.color: root.activeFocus ? Design.accent : Design.line }
  delegate: Controls.ItemDelegate {
    id: choice
    required property int index
    required property var modelData
    width: root.width
    implicitHeight: Style.space(36)
    text: root.textRole ? modelData[root.textRole] : modelData
    highlighted: root.highlightedIndex === index
    contentItem: Label { text: choice.text; elide: Text.ElideRight; wrapMode: Text.NoWrap; verticalAlignment: Text.AlignVCenter }
    background: Rectangle { color: choice.highlighted ? Design.raised : Design.background }
  }
  popup: Controls.Popup {
    y: root.height + Style.space(4); width: root.width; padding: Style.space(4)
    implicitHeight: Math.min(contentItem.implicitHeight + topPadding + bottomPadding, Style.space(260))
    contentItem: ListView {
      clip: true; implicitHeight: contentHeight
      model: root.popup.visible ? root.delegateModel : null
      currentIndex: root.highlightedIndex
      Controls.ScrollIndicator.vertical: Controls.ScrollIndicator {}
    }
    background: Rectangle { color: Design.background; radius: Style.space(6); border.color: Design.line }
  }
}
