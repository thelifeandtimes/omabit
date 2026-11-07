import QtQuick
import QtQuick.Layouts
import qs.Commons
import "AppPolicy.js" as Policy

Column {
  id:root
  property var service
  property bool active:false
  property alias controller:controller
  property string scope:"global"
  property string selectedRunner:""
  property string selectedShip:""
  property string commitValue:"block"
  property string publishValue:"block"
  readonly property var policyView:controller.value
  readonly property var runners:policyView ? policyView.runners : []
  readonly property var runner:runners.filter(function(r){return r.runner_id===root.selectedRunner})[0] || null
  readonly property var ships:runner && runner.status ? runner.status.targets : []
  readonly property var ship:ships.filter(function(s){return s.target.instance_id===root.selectedShip})[0] || null
  readonly property var target: {
    var t=(scope==="ship" || scope==="desk") && ship ? JSON.parse(JSON.stringify(ship.target)) : {}
    if(scope==="desk")t.desk=deskName.text.trim()
    return t
  }
  readonly property var choices:scope==="global" ? ["block","allow"] : ["inherit","block","allow"]
  readonly property bool canSave:controller.available && !controller.busy && !!policyView && (scope==="global" || !!runner && runner.can_edit && !!runner.status && (scope==="runner" || !!ship && ship.ready && (scope!=="desk" || /^[a-z][a-z0-9-]{0,63}$/.test(deskName.text.trim()))))
  readonly property var effective:ship && !ship.ready ? ({commit:{value:"block",source:"unresolved identity"},publish:{value:"block",source:"unresolved identity"}}) : Policy.effective(runner && runner.status ? runner.status.policy : null,target)
  spacing:Style.space(10)
  AppDevelopment {id:controller;service:root.service;active:root.active;onValueChanged:root.load()}
  function load() {
    var g={commit:"block",publish:"block"}
    if(policyView) {
      if(scope==="global")g=policyView.global.gates
      else if(runner && runner.status)g=Policy.configured(runner.status.policy,scope,target)
    }
    commitValue=g.commit;publishValue=g.publish
  }
  onScopeChanged:load()
  onSelectedRunnerChanged:{selectedShip="";load()}
  onTargetChanged:load()
  function save() {
    if(!canSave)return false
    var request={revision:policyView.global.revision,update:{scope:scope,target:target,gates:{commit:commitValue,publish:publishValue},expected_revision:runner && runner.status ? runner.status.policy.revision : 0}}
    if(scope==="global")request.runners=runners.map(function(r){return {name:r.name,runner_id:r.runner_id,expected_revision:r.status ? r.status.policy.revision : null,applied:false}})
    else {request.name=runner.name;request.runner_id=runner.runner_id}
    return controller.save(request)
  }
  Label {width:parent.width;text:"App development and publishing";font.pixelSize:Style.space(19);font.bold:true}
  Label {width:parent.width;text:"Choose where agents may commit to Clay and distribute app updates. Both permissions start blocked, including development ships.";secondary:true}
  RowLayout {
    ActionButton {objectName:"app-policy-refresh";text:controller.busy ? "Working…" : "Refresh policy";enabled:controller.available && !controller.busy;onClicked:controller.refresh()}
    ActionButton {objectName:"app-policy-retry";text:"Retry pending rollout";visible:!!root.policyView && root.policyView.global.runners.some(function(r){return !r.applied});enabled:controller.available && !controller.busy;onClicked:controller.retry()}
  }
  Label {width:parent.width;visible:!!controller.error;text:controller.error;color:Design.danger}
  Label {width:parent.width;visible:!!controller.message;text:controller.message;color:Design.success;caption:true}
  Label {width:parent.width;visible:!!root.policyView && root.runners.length===0;text:"No runners enrolled. Global choices are saved here; enroll a runner, then review and apply them.";secondary:true}
  SelectField {objectName:"app-policy-scope";width:parent.width;model:["Global","Runner","Ship","Desk"];currentIndex:["global","runner","ship","desk"].indexOf(root.scope);enabled:!controller.busy;onActivated:root.scope=["global","runner","ship","desk"][currentIndex]}
  SelectField {objectName:"app-policy-runner";width:parent.width;visible:root.scope!=="global";model:["Choose a runner…"].concat(root.runners.map(function(r){return r.name}));enabled:!controller.busy;currentIndex:root.runners.findIndex(function(r){return r.runner_id===root.selectedRunner})+1;onActivated:root.selectedRunner=currentIndex>0 ? root.runners[currentIndex-1].runner_id : ""}
  SelectField {objectName:"app-policy-ship";width:parent.width;visible:root.scope==="ship" || root.scope==="desk";model:["Choose a ship…"].concat(root.ships.map(function(s){return (s.ship || s.target.instance_id)+" · "+s.label+(s.ready ? "" : " (identity unresolved)")}));enabled:!controller.busy;currentIndex:root.ships.findIndex(function(s){return s.target.instance_id===root.selectedShip})+1;onActivated:root.selectedShip=currentIndex>0 ? root.ships[currentIndex-1].target.instance_id : ""}
  Field {id:deskName;objectName:"app-policy-desk";width:parent.width;visible:root.scope==="desk";enabled:!controller.busy;placeholderText:"Installed desk name, without % (for example, my-app)"}
  Label {width:parent.width;visible:root.scope==="desk";text:"Use the desk's exact Clay name. This policy editor does not query installed desks.";caption:true;secondary:true}
  Label {text:"Commit to Clay";font.bold:true}
  SelectField {objectName:"app-policy-commit";width:parent.width;model:root.choices.map(function(v){return v[0].toUpperCase()+v.slice(1)});currentIndex:root.choices.indexOf(root.commitValue);enabled:!controller.busy && !!root.policyView;onActivated:root.commitValue=root.choices[currentIndex]}
  Label {text:"Publish or distribute updates";font.bold:true}
  SelectField {objectName:"app-policy-publish";width:parent.width;model:root.choices.map(function(v){return v[0].toUpperCase()+v.slice(1)});currentIndex:root.choices.indexOf(root.publishValue);enabled:!controller.busy && !!root.policyView;onActivated:root.publishValue=root.choices[currentIndex]}
  Label {width:parent.width;text:"A commit to a published desk can update subscribers, so it requires both permissions. Inherit uses the nearest setting above this scope.";caption:true;secondary:true}
  Label {objectName:"app-policy-effective";width:parent.width;visible:root.scope!=="global" && !!root.runner && !!root.runner.status;text:"Saved effective policy: commit "+root.effective.commit.value+" ("+root.effective.commit.source+"); publish "+root.effective.publish.value+" ("+root.effective.publish.source+").\nRunner policy revision "+(root.runner && root.runner.status ? root.runner.status.policy.revision : "")+".";caption:true}
  Label {width:parent.width;visible:root.scope!=="global" && !!root.runner && !root.runner.can_edit;text:root.runner && root.runner.error ? root.runner.error : "An available operator connection is required to edit this runner.";color:Design.warning;caption:true}
  ActionButton {objectName:"app-policy-save";text:root.scope==="global" ? "Save and apply to enrolled runners" : "Save permissions";primary:true;enabled:root.canSave;onClicked:root.save()}
  Label {width:parent.width;visible:root.scope==="global";text:"Global changes preserve ship and desk overrides. Offline runners keep their last acknowledged policy. New enrollments start blocked until reviewed here.";caption:true;secondary:true}
  Repeater {
    model:root.runners
    Label {required property var modelData;width:root.width;caption:true;text:modelData.name+": "+modelData.rollout+(modelData.error ? "\n"+modelData.error : "")+(modelData.acknowledged ? "\nLast acknowledged revision "+modelData.acknowledged.revision+": commit "+modelData.acknowledged.global.commit+", publish "+modelData.acknowledged.global.publish : "");color:modelData.rollout==="applied" ? Design.success : Design.warning}
  }
  Label {width:parent.width;text:"These permissions govern Omabit-managed agent operations. Human Dojo commands and independent Urgit clients keep their own authority. Desk commit and publishing commands are still being implemented.";caption:true;secondary:true}
}
