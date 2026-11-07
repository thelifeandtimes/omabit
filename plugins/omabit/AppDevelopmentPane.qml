import QtQuick
import QtQuick.Layouts
import qs.Commons
import "AppPolicy.js" as Policy

Column {
  id:root
  property var service
  property bool active:false
  property bool detailsOpen:false
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
  readonly property var effective:(scope==="ship" || scope==="desk") && ship && !ship.ready ? ({commit:{value:"block",source:"unresolved identity"},publish:{value:"block",source:"unresolved identity"}}) : Policy.effective(runner && runner.status ? runner.status.policy : null,target)
  spacing:Style.space(12)
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
  Label { width:parent.width;text:"App development and publishing";font.pixelSize:Style.space(19);font.bold:true }
  Label { width:parent.width;text:"Choose where agents may commit to Clay and distribute app updates. Both permissions start blocked.";secondary:true }

  Card {
    objectName:"app-policy-scope-card"
    width:parent.width;implicitHeight:scopeContent.implicitHeight+Style.space(32)
    Column {
      id:scopeContent
      anchors.left:parent.left;anchors.right:parent.right;anchors.top:parent.top;anchors.margins:Style.space(16)
      spacing:Style.space(12)
      RowLayout {
        width:parent.width
        CardTitle { Layout.fillWidth:true;Layout.minimumWidth:0;text:"Scope" }
        ActionButton { objectName:"app-policy-refresh";compact:true;quiet:true;text:controller.busy ? "…" : "↻";hint:"Refresh policy";enabled:controller.available && !controller.busy;onClicked:controller.refresh() }
      }
      GridLayout {
        width:parent.width;columns:width>=Style.space(560) ? 2 : 1;columnSpacing:Style.space(16);rowSpacing:Style.space(12)
        Column {
          Layout.fillWidth:true;Layout.preferredWidth:1;Layout.minimumWidth:0;spacing:Style.space(6)
          Label { width:parent.width;text:"Apply to";caption:true;secondary:true }
          SelectField { objectName:"app-policy-scope";width:parent.width;model:["Global","Runner","Ship","Desk"];currentIndex:["global","runner","ship","desk"].indexOf(root.scope);enabled:!controller.busy;onActivated:root.scope=["global","runner","ship","desk"][currentIndex] }
        }
        Column {
          Layout.fillWidth:true;Layout.preferredWidth:1;Layout.minimumWidth:0;spacing:Style.space(6);visible:root.scope!=="global"
          Label { width:parent.width;text:"Runner";caption:true;secondary:true }
          SelectField { objectName:"app-policy-runner";width:parent.width;model:["Choose a runner…"].concat(root.runners.map(function(r){return r.name}));enabled:!controller.busy;currentIndex:root.runners.findIndex(function(r){return r.runner_id===root.selectedRunner})+1;onActivated:root.selectedRunner=currentIndex>0 ? root.runners[currentIndex-1].runner_id : "" }
        }
        Column {
          Layout.fillWidth:true;Layout.preferredWidth:1;Layout.minimumWidth:0;spacing:Style.space(6);visible:root.scope==="ship" || root.scope==="desk"
          Label { width:parent.width;text:"Ship";caption:true;secondary:true }
          SelectField { objectName:"app-policy-ship";width:parent.width;model:["Choose a ship…"].concat(root.ships.map(function(s){return (s.ship || s.target.instance_id)+" · "+s.label+(s.ready ? "" : " (identity unresolved)")}));enabled:!controller.busy;currentIndex:root.ships.findIndex(function(s){return s.target.instance_id===root.selectedShip})+1;onActivated:root.selectedShip=currentIndex>0 ? root.ships[currentIndex-1].target.instance_id : "" }
        }
        Column {
          Layout.fillWidth:true;Layout.preferredWidth:1;Layout.minimumWidth:0;spacing:Style.space(6);visible:root.scope==="desk"
          Label { width:parent.width;text:"Installed desk";caption:true;secondary:true }
          Field { id:deskName;objectName:"app-policy-desk";width:parent.width;enabled:!controller.busy;placeholderText:"Desk name, without %" }
        }
      }
      Label { width:parent.width;visible:root.scope==="desk";text:"Enter the exact Clay desk name, such as my-app.";caption:true;secondary:true }
      Label { width:parent.width;visible:!!root.policyView && root.runners.length===0;text:"No runners enrolled. Add a runner, then review and apply your global settings.";caption:true;secondary:true }
    }
  }

  Card {
    objectName:"app-policy-permissions-card"
    width:parent.width;implicitHeight:permissionsContent.implicitHeight+Style.space(32)
    Column {
      id:permissionsContent
      anchors.left:parent.left;anchors.right:parent.right;anchors.top:parent.top;anchors.margins:Style.space(16)
      spacing:Style.space(12)
      CardTitle { width:parent.width;text:"Agent permissions" }
      GridLayout {
        width:parent.width;columns:width>=Style.space(560) ? 2 : 1;columnSpacing:Style.space(16);rowSpacing:Style.space(12)
        Column {
          Layout.fillWidth:true;Layout.preferredWidth:1;Layout.minimumWidth:0;spacing:Style.space(6)
          Label { width:parent.width;text:"Commit to Clay";caption:true;secondary:true }
          SelectField { objectName:"app-policy-commit";width:parent.width;model:root.choices.map(function(v){return v[0].toUpperCase()+v.slice(1)});currentIndex:root.choices.indexOf(root.commitValue);enabled:!controller.busy && !!root.policyView;onActivated:root.commitValue=root.choices[currentIndex] }
        }
        Column {
          Layout.fillWidth:true;Layout.preferredWidth:1;Layout.minimumWidth:0;spacing:Style.space(6)
          Label { width:parent.width;text:"Publish or distribute updates";caption:true;secondary:true }
          SelectField { objectName:"app-policy-publish";width:parent.width;model:root.choices.map(function(v){return v[0].toUpperCase()+v.slice(1)});currentIndex:root.choices.indexOf(root.publishValue);enabled:!controller.busy && !!root.policyView;onActivated:root.publishValue=root.choices[currentIndex] }
        }
      }
      Label { width:parent.width;text:"Updating a published desk requires both permissions.";caption:true;secondary:true }
      Label { width:parent.width;visible:root.scope!=="global";text:"Inherit uses the nearest setting above this scope.";caption:true;secondary:true }
      Column {
        objectName:"app-policy-effective"
        width:parent.width;spacing:Style.space(8);visible:root.scope!=="global" && !!root.runner && !!root.runner.status
        Rectangle { width:parent.width;height:1;color:Design.line }
        CardTitle { width:parent.width;text:"Saved effective permissions" }
        Repeater {
          model:[{label:"Commit",value:root.effective.commit},{label:"Publish",value:root.effective.publish}]
          RowLayout {
            required property var modelData
            width:parent.width;spacing:Style.space(8)
            Label { Layout.fillWidth:true;Layout.minimumWidth:0;text:modelData.label;caption:true }
            Label { text:"from "+parent.modelData.value.source;secondary:true;caption:true;Layout.maximumWidth:parent.width/2;elide:Text.ElideRight;wrapMode:Text.NoWrap }
            StatusBadge { text:parent.modelData.value.value==="allow" ? "Allowed" : "Blocked";tone:parent.modelData.value.value==="allow" ? "accent" : "muted" }
          }
        }
      }
      Label { width:parent.width;visible:root.scope!=="global" && !!root.runner && !root.runner.can_edit;text:root.runner && root.runner.error ? root.runner.error : "An available operator connection is required to edit this runner.";color:Design.warning;caption:true }
      Label { width:parent.width;visible:root.scope==="global";text:"Applies to currently enrolled runners. Ship and desk overrides are preserved.";caption:true;secondary:true }
      ActionButton { objectName:"app-policy-save";text:root.scope==="global" ? "Save and apply" : "Save permissions";primary:true;enabled:root.canSave;onClicked:root.save() }
      Label { width:parent.width;visible:!!controller.error;text:controller.error;color:Design.danger;caption:true }
      Label { width:parent.width;visible:!!controller.message;text:controller.message;color:Design.success;caption:true }
    }
  }

  Card {
    objectName:"app-policy-rollout-card"
    width:parent.width;implicitHeight:rolloutContent.implicitHeight+Style.space(32);visible:root.scope==="global" && root.runners.length>0
    Column {
      id:rolloutContent
      anchors.left:parent.left;anchors.right:parent.right;anchors.top:parent.top;anchors.margins:Style.space(16)
      spacing:Style.space(10)
      RowLayout {
        width:parent.width
        CardTitle { Layout.fillWidth:true;Layout.minimumWidth:0;text:"Runner rollout" }
        ActionButton { objectName:"app-policy-retry";compact:true;quiet:true;text:"Retry pending";visible:!!root.policyView && root.policyView.global.runners.some(function(r){return !r.applied});enabled:controller.available && !controller.busy;onClicked:controller.retry() }
      }
      Label { width:parent.width;text:"Offline runners keep their last acknowledged permissions until the change reaches them.";caption:true;secondary:true }
      Repeater {
        model:root.runners
        Column {
          required property var modelData
          width:rolloutContent.width;spacing:Style.space(6)
          RowLayout {
            width:parent.width;spacing:Style.space(8)
            Label { Layout.fillWidth:true;Layout.minimumWidth:0;text:parent.parent.modelData.name;wrapMode:Text.NoWrap;elide:Text.ElideRight }
            StatusBadge { readonly property string rollout:parent.parent.modelData.rollout;text:rollout==="applied" ? "Applied" : rollout==="not enrolled in global rollout" ? "Not applied" : "Pending";tone:rollout==="applied" ? "good" : rollout==="not enrolled in global rollout" ? "muted" : "warn" }
          }
          Label { width:parent.width;visible:!!parent.modelData.error;text:parent.modelData.error || "";caption:true;color:Design.warning }
          Label { width:parent.width;visible:root.detailsOpen;text:parent.modelData.rollout;caption:true;secondary:true }
          Label { width:parent.width;visible:!!parent.modelData.acknowledged;text:parent.modelData.acknowledged ? "Last acknowledged: commit "+parent.modelData.acknowledged.global.commit+", publish "+parent.modelData.acknowledged.global.publish+" · revision "+parent.modelData.acknowledged.revision : "";caption:true;secondary:true }
        }
      }
    }
  }
  ActionButton { objectName:"app-policy-details";compact:true;quiet:true;text:root.detailsOpen ? "Hide details" : "Policy details";onClicked:root.detailsOpen=!root.detailsOpen }
  Column {
    width:parent.width;visible:root.detailsOpen;spacing:Style.space(8)
    Label { width:parent.width;visible:!!root.runner && !!root.runner.status;text:"Runner policy revision "+(root.runner && root.runner.status ? root.runner.status.policy.revision : "");caption:true;secondary:true }
    Label { width:parent.width;text:"Use the installed desk's exact name; this editor does not query Clay. Permissions govern Omabit-managed agent operations. Human Dojo commands and independent Urgit clients keep their own authority.";caption:true;secondary:true }
  }
  Label { width:parent.width;text:"Permissions can be saved now. Desk commit and publishing controls are not available yet.";caption:true;secondary:true }
}
