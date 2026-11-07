import QtQuick
import QtQuick.Controls as Controls
import qs.Commons

Controls.ScrollView {
  id:root
  property alias text:input.text
  property alias placeholderText:input.placeholderText
  property int maximumLength:2048
  signal edited(string value)
  implicitHeight:Style.space(110)
  clip:true
  Controls.TextArea {
    id:input
    textFormat:TextEdit.PlainText
    wrapMode:TextEdit.Wrap
    selectByMouse:true
    color:Design.ink
    placeholderTextColor:Design.muted
    selectionColor:Design.accent
    selectedTextColor:Design.background
    font.family:Style.font.family
    font.pixelSize:Style.font.bodySmall
    padding:Style.space(12)
    onTextChanged:{if(text.length>root.maximumLength){text=text.slice(0,root.maximumLength);return}root.edited(text)}
    background:Rectangle {radius:Style.space(6);color:Design.background;border.color:input.activeFocus ? Design.accent : Design.line}
  }
}
