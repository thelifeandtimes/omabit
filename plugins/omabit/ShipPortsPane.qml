import QtQuick
import QtQuick.Layouts
import qs.Commons
import "Model.js" as Model

Column {
  id: root
  property var service
  property var connection
  property var instance
  property var target: null
  property string notice: ""
  readonly property string selection: connection && instance ? connection.runner_id+"/"+instance.id : ""
  spacing: Style.space(8)
  onSelectionChanged: {target=null;notice=""}
  onVisibleChanged: if (!visible) target=null
  ActionButton {
    objectName: "edit-ship-ports"
    text: root.target ? "Reload HTTP configuration" : "Edit HTTP port…"
    enabled: root.service && root.service.canConfigure(Model.edit(root.connection,root.instance))
    onClicked: {root.target=Model.edit(root.connection,root.instance);httpPort.text=String(root.target.port);root.notice=""}
  }
  Column {
    width: parent.width; spacing: Style.space(8); visible: !!root.target
    Label { text: "Host HTTP port"; secondary: true; caption: true }
    Field { id: httpPort; objectName: "ship-http-port"; width: parent.width; validator: IntValidator {bottom:1024;top:65535} }
    Label { width: parent.width; text: "Changes apply on the next explicit stop/start. The current revision is checked before saving."; secondary: true; caption: true }
    RowLayout {
      ActionButton { objectName: "save-ship-http-port"; text: "Save HTTP port"; primary: true; enabled: root.target && root.service.canConfigure(root.target) && httpPort.acceptableInput; onClicked: {if(root.service.configure(root.target,httpPort.text)){root.target=null;root.notice="Change submitted. The configuration above and ship activity show the result."}else root.notice=root.service.lastError || "Ship state changed; reload the configuration."} }
      ActionButton { text: "Cancel"; onClicked: root.target=null }
    }
  }
  Label { width: parent.width; visible: root.notice!==""; text: root.notice; caption: true; secondary: true }
}
