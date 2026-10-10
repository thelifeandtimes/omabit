import QtQuick
import QtQuick.Layouts
import qs.Commons
import "Model.js" as Model

Card {
  id: root
  property var service
  property var connection
  property var instance
  readonly property var policy: instance ? instance.schedule || null : null
  readonly property var outcome: {
    var o=service ? service.routineOutcome : null
    return o && connection && instance && o.target.runner === connection.runner_id && o.target.instance === instance.id ? o : null
  }
  signal editRequested()
  objectName: "routine-policy-card"
  implicitHeight: body.implicitHeight + Style.space(28)
  border.color: policy && policy.config.enabled ? Design.fade(Design.success,0.45) : Design.line
  Column {
    id: body
    anchors.left: parent.left; anchors.right: parent.right; anchors.top: parent.top; anchors.margins: Style.space(14); spacing: Style.space(10)
    RowLayout {
      width: parent.width
      Label { Layout.fillWidth: true; text: root.policy ? "Scheduled " + root.policy.config.action : "Routine maintenance"; font.bold: true; font.pixelSize: Style.space(18) }
      StatusBadge { objectName: "routine-policy-status"; text: !root.policy ? "Not set" : root.policy.config.enabled ? "Enabled" : "Saved · disabled"; tone: root.policy && root.policy.config.enabled ? "good" : "muted" }
    }
    Label { width: parent.width; text: Model.routineTiming(root.policy); font.bold: !!root.policy }
    Label { width: parent.width; visible: !!root.policy; secondary: true; text: root.policy ? Model.routineBehavior(root.policy) : "" }
    Label {
      width: parent.width; objectName: "routine-next-run"
      text: !root.policy ? "Create a policy to schedule pack or meld." : !root.policy.config.enabled ? "This policy will not run until you enable it."
        : root.policy.instance_revision !== root.instance.config_revision ? "Configuration changed — review and save the policy before it can run."
        : root.policy.next_slot ? "Next run: " + Model.humanTime(root.policy.next_slot,root.service.now) + " (local time)" : "Waiting for the next eligible schedule slot."
      color: root.policy && root.policy.config.enabled ? Design.success : Design.muted
    }
    Label { width: parent.width; visible: !!root.outcome; text: root.outcome ? root.outcome.message : ""; color: root.outcome && root.outcome.status === "saved" ? Design.success : root.outcome && root.outcome.status === "submitting" ? Design.accent : Design.danger }
    Flow {
      width: parent.width; spacing: Style.space(8)
      ActionButton {
        objectName: "toggle-routine"
        visible: !!root.policy
        text: root.policy && root.policy.config.enabled ? "Disable policy" : "Enable policy"
        primary: root.policy && !root.policy.config.enabled
        enabled: root.service && root.policy && root.policy.instance_revision === root.instance.config_revision && root.service.canSaveRoutine(Model.routineTarget(root.connection,root.instance)) && !(root.outcome && root.outcome.status === "uncertain")
        onClicked: root.service.saveRoutine(Model.routineTarget(root.connection,root.instance),Object.assign({},root.policy.config,{enabled:!root.policy.config.enabled,stop_running:root.policy.config.enabled ? root.policy.config.stop_running : true}))
      }
      ActionButton { text: root.policy ? "Edit policy…" : "Create policy…"; enabled: root.service && root.service.canSaveRoutine(Model.routineTarget(root.connection,root.instance)); onClicked: root.editRequested() }
    }
  }
}
