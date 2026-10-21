import QtQuick
import QtQuick.Controls as Controls
import qs.Commons

SelectField {
  id: root
  editable: true
  currentIndex: -1
  property alias text: root.editText
  property string placeholderText: ""
  contentItem: Controls.TextField {
    text: root.editText
    placeholderText: root.placeholderText
    color: Design.ink; placeholderTextColor: Design.muted
    selectionColor: Design.accent; selectedTextColor: Design.background
    font.family: Style.font.family; font.pixelSize: Style.font.bodySmall
    verticalAlignment: Text.AlignVCenter
    selectByMouse: true; autoScroll: true
    inputMethodHints: Qt.ImhNoPredictiveText
    leftPadding: 0; rightPadding: 0
    background: null
    onTextEdited: root.editText=text
  }
}
