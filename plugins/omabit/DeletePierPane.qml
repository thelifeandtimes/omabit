import QtQuick
import qs.Commons
import "Model.js" as Model

Column {
  id: root
  property var service
  property var connection
  property var instance
  property var target: null
  property string targetRunner: ""
  property bool submitted: false
  spacing: Style.space(8)
  readonly property bool current: !!target && !!connection && connection.runner_id===targetRunner && !!instance && instance.id===target.id && instance.pier_path===target.pier_path && instance.config_revision===target.config_revision && instance.ship===target.ship
  ActionButton { objectName: "delete-pier-review"; text: root.instance && root.instance.delete_pending ? "Continue pier deletion…" : "Delete pier…"; enabled: root.service && !root.service.busy && Model.canDeletePier(root.connection,root.instance,root.service.now); onClicked: {root.target=JSON.parse(JSON.stringify(root.instance));root.targetRunner=root.connection.runner_id;root.submitted=false;confirmation.text=""} }
  Column {
    width: parent.width; spacing: Style.space(8); visible: !!root.target
    Label { width: parent.width; color: Design.danger; text: root.target ? "Permanently delete "+root.target.id+" ("+(root.target.ship || "uninitialized")+") on "+root.targetRunner+"?\n"+Model.absolutePierPath(root.connection,root.target.pier_path)+"\nAll pier data and this registration will be removed. Operation history is retained. This cannot be undone." : "" }
    Field { id: confirmation; objectName: "delete-pier-name"; width: parent.width; placeholderText: "Type the exact instance name"; enabled: !root.submitted && !root.service.busy }
    ActionButton { objectName: "delete-pier-submit"; text: root.submitted ? "Deletion submitted — inspect activity" : "Permanently delete pier"; enabled: root.current && !root.submitted && !root.service.busy && confirmation.text===root.target.id && Model.canDeletePier(root.connection,root.instance,root.service.now); onClicked: if(root.service.deletePier(root.connection,root.target,confirmation.text))root.submitted=true }
    ActionButton { text: "Close deletion review"; quiet: true; onClicked: root.target=null }
    Label { width: parent.width; visible: !root.current; text: "The selected instance changed or was removed. Open a fresh review before deleting."; secondary: true }
  }
}
