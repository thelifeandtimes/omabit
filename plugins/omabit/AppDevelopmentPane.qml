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
  property string selectedDesk:""
  property string commitValue:"block"
  property string publishValue:"block"
  property string createValue:"block"
  property string filterGate:"commit"
  property string filterValue:"all"
  readonly property var policyView:controller.value
  readonly property var runners:policyView ? policyView.runners : []
  readonly property var runner:runners.filter(function(r){return r.runner_id===root.selectedRunner})[0] || null
  readonly property var ships:runner && runner.status ? runner.status.targets : []
  readonly property var ship:ships.filter(function(s){return s.target.instance_id===root.selectedShip})[0] || null
  readonly property var target: {
    var t=(scope==="ship" || scope==="desk") && ship ? JSON.parse(JSON.stringify(ship.target)) : {}
    if(scope==="desk")t.desk=selectedDesk
    return t
  }
  readonly property var choices:scope==="global" ? ["block","allow"] : ["inherit","block","allow"]
  readonly property bool canSave:controller.available && !controller.busy && !!policyView && (scope==="global" || !!runner && runner.can_edit && !!runner.status && (scope==="runner" || !!ship && ship.ready && (scope!=="desk" || /^[a-z][a-z0-9-]{0,63}$/.test(selectedDesk))))
  readonly property var effective:scope==="global" && policyView ? ({commit:{value:policyView.global.gates.commit,source:"global"},publish:{value:policyView.global.gates.publish,source:"global"},create_desk:{value:policyView.global.gates.create_desk,source:"global"}}) : Policy.effective((scope==="ship" || scope==="desk") && ship && !ship.ready ? null : runner && runner.status ? runner.status.policy : null,target)
  readonly property string selectionKey:Policy.nodeKey(scope,scope==="global" ? "" : selectedRunner,target)
  readonly property string selectionTitle:scope==="global" ? "Global defaults" : scope==="runner" ? runner ? runner.name : "Runner" : scope==="ship" ? ship ? ship.ship || ship.label : "Ship" : "%"+selectedDesk
  readonly property string breadcrumb:scope==="global" ? "Applies to enrolled runners" : "Global / "+(runner ? runner.name : "Runner")+((scope==="ship" || scope==="desk") ? " / "+(ship ? ship.ship || ship.label : "Ship") : "")+(scope==="desk" ? " / %"+selectedDesk : "")
  spacing:Style.space(12)
  AppDevelopment {id:controller;service:root.service;active:root.active;onValueChanged:root.load()}
  function load() {
    var g={commit:"block",publish:"block",create_desk:"block"}
    if(policyView) {
      if(scope==="global")g=policyView.global.gates
      else if(runner && runner.status)g=Policy.configured(runner.status.policy,scope,target)
    }
    commitValue=g.commit;publishValue=g.publish;createValue=g.create_desk
  }
  onScopeChanged:load()
  onSelectedRunnerChanged:{selectedShip="";selectedDesk="";load()}
  onSelectedShipChanged:{selectedDesk="";load()}
  onTargetChanged:load()
  function selectNode(node) {
    scope=node.scope;selectedRunner=node.runner;selectedShip=node.target.instance_id||"";selectedDesk=node.target.desk||""
    load()
  }
  function valueFor(key){return key==="commit" ? commitValue : key==="publish" ? publishValue : createValue}
  function setValue(key,value){if(key==="commit")commitValue=value;else if(key==="publish")publishValue=value;else createValue=value}
  function reviewDesk(name) {
    if(!ship || !/^[a-z][a-z0-9-]{0,63}$/.test(name))return false
    scope="desk";selectedDesk=name;load();return true
  }
  function save() {
    if(!canSave)return false
    var request={revision:policyView.global.revision,update:{scope:scope,target:target,gates:{commit:commitValue,publish:publishValue,create_desk:scope==="desk" ? "inherit" : createValue},expected_revision:runner && runner.status ? runner.status.policy.revision : 0}}
    if(scope==="global")request.runners=runners.map(function(r){return {name:r.name,runner_id:r.runner_id,expected_revision:r.status ? r.status.policy.revision : null,applied:false}})
    else {request.name=runner.name;request.runner_id=runner.runner_id}
    return controller.save(request)
  }
  RowLayout {
    width:parent.width
    Label { Layout.fillWidth:true;Layout.minimumWidth:0;text:"App development and publishing";font.pixelSize:Style.space(19);font.bold:true }
    ActionButton { objectName:"app-policy-refresh";compact:true;quiet:true;text:controller.busy ? "…" : "↻";hint:"Refresh policy";enabled:controller.available && !controller.busy;onClicked:controller.refresh() }
  }
  Label { width:parent.width;text:"Select a branch or desk to set agent permissions. More specific settings override inherited ones.";secondary:true }

  GridLayout {
    id:editorLayout
    width:parent.width;columns:width>=Style.space(1000) ? 2 : 1;columnSpacing:Style.space(12);rowSpacing:Style.space(12)
  Card {
    objectName:"app-policy-map-card"
    Layout.fillWidth:true;Layout.minimumWidth:0;Layout.preferredWidth:740;Layout.alignment:Qt.AlignTop
    implicitHeight:mapContent.implicitHeight+Style.space(32)
    Column {
      id:mapContent
      anchors.left:parent.left;anchors.right:parent.right;anchors.top:parent.top;anchors.margins:Style.space(16)
      spacing:Style.space(12)
      RowLayout {
        width:parent.width
        CardTitle { Layout.fillWidth:true;text:"Permission map" }
        Label { caption:true;secondary:true;text:tree.filtering ? tree.graph.matches+(tree.graph.matches===1 ? " matching node" : " matching nodes") : "Saved permissions" }
      }
      GridLayout {
        width:parent.width;columns:width>Style.space(590) ? 3 : 2;columnSpacing:Style.space(10);rowSpacing:Style.space(10)
        Column {
          Layout.fillWidth:true;Layout.preferredWidth:1;Layout.minimumWidth:0;spacing:Style.space(6)
          Label { width:parent.width;text:"Permission";caption:true;secondary:true }
          SelectField { objectName:"app-policy-filter-gate";width:parent.width;model:["Commit to Clay","Publish updates","Create desks"];currentIndex:["commit","publish","create_desk"].indexOf(root.filterGate);onActivated:root.filterGate=["commit","publish","create_desk"][currentIndex] }
        }
        Column {
          Layout.fillWidth:true;Layout.preferredWidth:1;Layout.minimumWidth:0;spacing:Style.space(6)
          Label { width:parent.width;text:"Highlight";caption:true;secondary:true }
          SelectField { objectName:"app-policy-filter-value";width:parent.width;model:["All permissions","Allowed","Blocked"];currentIndex:["all","allow","block"].indexOf(root.filterValue);onActivated:root.filterValue=["all","allow","block"][currentIndex] }
        }
        Column {
          Layout.fillWidth:true;Layout.preferredWidth:1;Layout.minimumWidth:0;Layout.columnSpan:parent.columns===2 ? 2 : 1;spacing:Style.space(6)
          Label { width:parent.width;text:"Find in tree";caption:true;secondary:true }
          Field { id:search;objectName:"app-policy-search";width:parent.width;placeholderText:"Runner, ship or %desk" }
        }
      }
      AppPolicyTree {
        id:tree;objectName:"app-policy-tree";width:parent.width
        model:Policy.tree(root.policyView,{runner:root.selectedRunner,target:root.target})
        gate:root.filterGate;filter:root.filterValue;query:search.text;selectedKey:root.selectionKey
        onSelected:function(node){root.selectNode(node)}
      }
      Label { width:parent.width;caption:true;secondary:true;text:tree.filtering ? "Highlighted paths lead to matches. Each node shows its own effective permission; unrelated branches stay dimmed." : "Permissions shown here are saved values. Select any node to edit it below." }
      Label { width:parent.width;visible:!!root.policyView && root.runners.length===0;text:"No runners enrolled. Add a runner, then review and apply your global settings.";caption:true;secondary:true }
      Label { width:parent.width;visible:!root.policyView;text:controller.busy ? "Loading permissions…" : "Permissions unavailable. Refresh to try again.";caption:true;secondary:true }
    }
  }

  Card {
    objectName:"app-policy-permissions-card"
    Layout.fillWidth:true;Layout.minimumWidth:0;Layout.preferredWidth:320;Layout.alignment:Qt.AlignTop
    implicitHeight:permissionsContent.implicitHeight+Style.space(32)
    Column {
      id:permissionsContent
      anchors.left:parent.left;anchors.right:parent.right;anchors.top:parent.top;anchors.margins:Style.space(16)
      spacing:Style.space(12)
      RowLayout {
        width:parent.width
        Label { Layout.fillWidth:true;Layout.minimumWidth:0;text:root.selectionTitle;font.bold:true;font.pixelSize:Style.space(16) }
        StatusBadge { text:root.scope[0].toUpperCase()+root.scope.slice(1);tone:"muted" }
      }
      Label { width:parent.width;caption:true;secondary:true;text:root.breadcrumb }
      GridLayout {
        width:parent.width;columns:width>=Style.space(670) ? (root.scope==="desk" ? 2 : 3) : width>=Style.space(430) ? 2 : 1
        columnSpacing:Style.space(16);rowSpacing:Style.space(12)
        Repeater {
          model:(root.scope==="desk" ? [] : [{key:"create_desk",label:"Create new desks"}]).concat([{key:"commit",label:"Commit to Clay"},{key:"publish",label:"Publish or distribute"}])
          Column {
            id:gate
            required property var modelData
            Layout.fillWidth:true;Layout.preferredWidth:1;Layout.minimumWidth:0;spacing:Style.space(6)
            Label { width:parent.width;text:gate.modelData.label;caption:true;secondary:true }
            SelectField { objectName:"app-policy-"+gate.modelData.key;width:parent.width;model:root.choices.map(function(v){return v[0].toUpperCase()+v.slice(1)});currentIndex:root.choices.indexOf(root.valueFor(gate.modelData.key));enabled:root.canSave;onActivated:root.setValue(gate.modelData.key,root.choices[currentIndex]) }
            Label { width:parent.width;caption:true;secondary:true;text:"Saved: "+root.effective[gate.modelData.key].value+" · "+root.effective[gate.modelData.key].source }
          }
        }
      }
      Label { width:parent.width;visible:root.scope!=="global";text:"Inherit uses the nearest explicit setting above this scope.";caption:true;secondary:true }
      Label { width:parent.width;text:root.scope==="desk" ? "Desk creation is controlled by the ship's policy. This desk inherits commit and publishing permissions until you set an override." : "New desks inherit commit and publishing permissions. Allowing desk creation does not grant either permission.";caption:true;secondary:true }
      Label { width:parent.width;text:"Updating a published desk requires both commit and publishing permission.";caption:true;secondary:true }
      Label { width:parent.width;visible:root.scope!=="global" && !!root.runner && !root.runner.can_edit;text:root.runner && root.runner.error ? root.runner.error : "An available operator connection is required to edit this runner.";color:Design.warning;caption:true }
      Label { width:parent.width;visible:root.scope==="global";text:"Applies to currently enrolled runners. Ship and desk overrides are preserved.";caption:true;secondary:true }
      ActionButton { objectName:"app-policy-save";text:root.scope==="global" ? "Save and apply" : "Save permissions";primary:true;enabled:root.canSave;onClicked:root.save() }
      Label { width:parent.width;visible:!!controller.error;text:controller.error;color:Design.danger;caption:true }
      Label { width:parent.width;visible:!!controller.message;text:controller.message;color:Design.success;caption:true }
      Column {
        width:parent.width;spacing:Style.space(8);visible:root.scope==="ship" || root.scope==="desk"
        Rectangle { width:parent.width;height:1;color:Design.line }
        Label { width:parent.width;text:"Review another desk's permissions";caption:true;secondary:true }
        RowLayout {
          width:parent.width;spacing:Style.space(8)
          Field { id:deskName;objectName:"app-policy-desk";Layout.fillWidth:true;Layout.minimumWidth:0;placeholderText:"Desk name, without %" }
          ActionButton { text:"Review desk";enabled:!!root.ship && !controller.busy && /^[a-z][a-z0-9-]{0,63}$/.test(deskName.text.trim());onClicked:root.reviewDesk(deskName.text.trim()) }
        }
        Label { width:parent.width;text:"Adds a name to this policy view. It does not create or install a desk.";caption:true;secondary:true }
      }
    }
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
          Label { width:parent.width;visible:!!parent.modelData.acknowledged;text:parent.modelData.acknowledged ? "Last acknowledged: create desks "+parent.modelData.acknowledged.global.create_desk+", commit "+parent.modelData.acknowledged.global.commit+", publish "+parent.modelData.acknowledged.global.publish+" · revision "+parent.modelData.acknowledged.revision : "";caption:true;secondary:true }
        }
      }
    }
  }
  ActionButton { objectName:"app-policy-details";compact:true;quiet:true;text:root.detailsOpen ? "Hide details" : "Policy details";onClicked:root.detailsOpen=!root.detailsOpen }
  Column {
    width:parent.width;visible:root.detailsOpen;spacing:Style.space(8)
    Label { width:parent.width;visible:!!root.runner && !!root.runner.status;text:"Runner policy revision "+(root.runner && root.runner.status ? root.runner.status.policy.revision : "");caption:true;secondary:true }
    Label { width:parent.width;text:"The tree includes saved desk policies and any supplied desk inventory. Installed-desk discovery is not connected yet. A name entered here is not evidence that a desk exists on the ship.";caption:true;secondary:true }
    Label { width:parent.width;text:"Permissions govern Omabit-managed agent operations. Human Dojo commands and independent Urgit clients keep their own authority.";caption:true;secondary:true }
  }
  Label { width:parent.width;text:"Permissions can be saved now. Desk creation, commit and publishing commands are not available yet.";caption:true;secondary:true }
}
