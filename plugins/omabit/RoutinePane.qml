import QtQuick
import QtQuick.Controls as Controls
import QtQuick.Layouts
import qs.Commons
import "Model.js" as Model

Column {
  id: root
  property var service
  property var connection
  property var instance
  property var target: null
  property var days: [0]
  readonly property string selected: connection && instance ? connection.runner_id + "/" + instance.id : ""
  readonly property var policy: instance ? instance.schedule || null : null
  spacing: Style.space(12)
  onSelectedChanged: target = null
  onVisibleChanged: if (!visible) target = null
  function edit() {
    service.routineOutcome = null
    target = Model.routineTarget(connection,instance)
    var c = policy ? policy.config : {action:"pack",weekdays:[0],hour_utc:2,minute_utc:0,window_minutes:15,resume:false}
    action.currentIndex = c.action === "meld" ? 1 : 0
    days = c.weekdays.slice(); hour.text = String(c.hour_utc); minute.text = String(c.minute_utc); window.text = String(c.window_minutes)
    resume.checked = c.resume
  }
  Label { text: "Routine maintenance"; font.pixelSize: Style.space(21); font.bold: true }
  Label { width: parent.width; text: "Runs on the runner, even when this desktop is closed. Times are UTC. Missed windows are skipped; interrupted jobs are not continued automatically."; secondary: true }
  RoutineCard { width: parent.width; service: root.service; connection: root.connection; instance: root.instance; onEditRequested: root.edit() }
  Label { width: parent.width; visible: root.policy && root.instance && root.policy.instance_revision !== root.instance.config_revision; text: "Configuration changed. Save a reviewed policy to bind the new revision."; color: Design.warning }
  Label { width: parent.width; visible: root.instance && !!root.instance.routine; text: root.instance && root.instance.routine ? "Last routine stage: " + root.instance.routine.stage + "\nScheduled UTC slot: " + root.instance.routine.slot : ""; secondary: true; caption: true }
  Connections {
    target: root.service
    function onRoutineOutcomeChanged() {
      var o=root.service.routineOutcome
      if (o && root.target && o.target.runner === root.target.runner && o.target.instance === root.target.instance && o.status === "saved") root.target=null
    }
  }
  Label { width: parent.width; visible: root.connection && !Model.hasFeature(root.connection,"maintenance-schedules"); text: "Requires the updated runner's maintenance-schedules capability."; color: Design.warning; caption: true }
  Card {
    width: parent.width; visible: !!root.target; implicitHeight: form.implicitHeight + Style.space(28)
    Column {
      id: form
      anchors.left: parent.left; anchors.right: parent.right; anchors.top: parent.top; anchors.margins: Style.space(14); spacing: Style.space(12)
      Label { width: parent.width; text: "Saving enables this policy. Scheduled maintenance stops the ship when needed; choose below whether to resume it after success."; secondary: true; caption: true }
      SelectField { id: action; width: parent.width; model: ["pack","meld"] }
      Label { text: "UTC weekdays"; secondary: true; caption: true }
      Flow {
        width: parent.width; spacing: Style.space(4)
        Repeater {
          model: ["Sun","Mon","Tue","Wed","Thu","Fri","Sat"]
          ActionButton { required property int index; required property string modelData; text: modelData; compact: true; primary: root.days.indexOf(index)>=0; onClicked: root.days = root.days.indexOf(index)>=0 ? root.days.filter(function(d) { return d !== index }) : root.days.concat([index]).sort() }
        }
      }
      RowLayout {
        width: parent.width
        Field { id: hour; Layout.fillWidth: true; placeholderText: "UTC hour (0–23)"; validator: IntValidator { bottom:0; top:23 } }
        Label { text: ":" }
        Field { id: minute; Layout.fillWidth: true; placeholderText: "Minute (0–59)"; validator: IntValidator { bottom:0; top:59 } }
      }
      Label { text: "Dispatch window in minutes (5–180)"; secondary: true; caption: true }
      Field { id: window; width: parent.width; validator: IntValidator { bottom:5; top:180 } }
      CheckField { id: resume; objectName: "routine-resume"; width: parent.width; text: "Resume after success, only if the ship was running beforehand." }
      Label { width: parent.width; text: "After a failed maintenance step, automatic resume is not attempted; inspect the activity feed. Retention deletion stays manual."; secondary: true; caption: true }
      RowLayout {
        ActionButton { objectName: "save-routine"; text: "Save policy"; primary: true; enabled: root.service && root.days.length > 0 && hour.acceptableInput && minute.acceptableInput && window.acceptableInput && root.service.canSaveRoutine(root.target); onClicked: root.service.saveRoutine(root.target,{enabled:true,action:action.currentText,weekdays:root.days.slice(),hour_utc:Number(hour.text),minute_utc:Number(minute.text),window_minutes:Number(window.text),stop_running:true,resume:resume.checked}) }
        ActionButton { text: "Cancel"; onClicked: root.target = null }
      }
    }
  }
}
