import QtQuick
import qs.Commons

Text {
  property bool secondary: false
  property bool caption: false
  color: secondary ? Design.muted : Design.ink
  font.family: Style.font.family
  font.pixelSize: caption ? Style.font.caption : Style.font.bodySmall
  textFormat: Text.PlainText
  wrapMode: Text.Wrap
}
