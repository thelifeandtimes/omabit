import QtQuick
import qs.Commons
import "Model.js" as Model

Column {
  id:root
  property var service
  property var connection
  property var instance
  readonly property string selection: connection && instance ? connection.runner_id+"/"+instance.id+"/"+instance.config_revision : ""
  readonly property bool stopped: !!instance && !instance.project_id && instance.desired_state==="stopped" && ["stopped","uninitialized"].indexOf(instance.observed_state)>=0 && !instance.recovery_required && !instance.active_operation_id
  readonly property bool changed: !!instance && choice.selectedProfile!==(instance.config.runtime_profile || "")
  spacing:Style.space(10)
  onSelectionChanged: {choice.selectedProfile=instance ? instance.config.runtime_profile || "" : "";consent.checked=false}
  RuntimeProfileChoice {id:choice;width:parent.width;connection:root.connection;enabled:root.stopped && !!root.service && !root.service.busy;onEdited:consent.checked=false}
  Label {width:parent.width;secondary:true;caption:true;text:root.stopped ? "Changing this selection keeps the pier stopped. Preserve a backup before starting an untested build; runtime changes can affect replay and cannot always be reversed." : "Stop this ship and resolve recovery before changing its runtime."}
  CheckField {id:consent;width:parent.width;visible:root.changed && root.stopped;text:"Use this runtime on the next Start."}
  ActionButton {text:root.service && root.service.currentAction==="runtime-change" ? "Selecting runtime…" : "Select runtime";visible:root.changed;enabled:root.stopped && consent.checked && !!root.service && !root.service.busy && Model.hasFeature(root.connection,"runtime-transition-v1");onClicked:root.service.selectRuntime(root.connection,root.instance,choice.selectedProfile || "vere-4.6-5a7614f")}
}
