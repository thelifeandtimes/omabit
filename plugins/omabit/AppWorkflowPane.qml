import QtQuick
import QtQuick.Layouts
import qs.Commons
import "AppWorkflow.js" as Workflow

Card {
  id:root
  property var service
  property bool active:false
  property var runner:null
  property var target:({})
  property bool ready:false
  property alias controller:controller
  property var draft:null
  property string checksText:""
  readonly property string validationError:Workflow.configError(draft)
  readonly property bool isDesk:!!target.desk
  readonly property var config:draft ? (isDesk ? draft.desk : draft.ship) : null
  readonly property bool canSave:controller.available && controller.canEdit && !controller.busy && !!draft && !!controller.value
  readonly property bool dirty:!!draft && !!controller.value && JSON.stringify([draft.ship,draft.desk])!==JSON.stringify([controller.value.ship,controller.value.desk])
  objectName:"app-workflow-card"
  implicitHeight:content.implicitHeight+Style.space(32)
  function load(){draft=controller.value ? JSON.parse(JSON.stringify(controller.value)) : null;checksText=draft && draft.desk ? draft.desk.checks.join("\n") : ""}
  function setField(key,value){if(!config)return;var d=JSON.parse(JSON.stringify(draft));d[isDesk ? "desk" : "ship"][key]=value;draft=d}
  function setWorkflow(value){if(!draft)return;var d=JSON.parse(JSON.stringify(draft));d.desk.workflow=value;d.desk.source_name=value==="mounted" ? target.desk : "";draft=d}
  function editChecks(value){checksText=value;if(draft && isDesk)setField("checks",value.split("\n").map(function(v){return v.trim()}).filter(function(v){return !!v}))}
  function save(){return canSave && !validationError && controller.save(draft.ship||null,draft.desk||null)}
  AppWorkflow {
    id:controller;service:root.service;active:root.active && root.ready
    runnerName:root.runner ? root.runner.name : "";runnerID:root.runner ? root.runner.runner_id : ""
    mode:root.runner && root.runner.status ? root.runner.status.mode : "";target:root.target
    canEdit:!!root.runner && root.runner.can_edit
    onValueChanged:root.load()
  }
  Column {
    id:content
    anchors.left:parent.left;anchors.right:parent.right;anchors.top:parent.top;anchors.margins:Style.space(16)
    spacing:Style.space(12)
    RowLayout {
      width:parent.width
      CardTitle {Layout.fillWidth:true;Layout.minimumWidth:0;text:root.isDesk ? "%"+root.target.desk+" · Development workflow" : "Ship purpose"}
      Label {visible:root.dirty;text:"Unsaved";caption:true;color:Design.warning}
      ActionButton {objectName:"app-workflow-refresh";compact:true;quiet:true;text:"↻";hint:"Refresh workflow settings";enabled:controller.available && !controller.busy;onClicked:controller.refresh()}
    }
    Label {width:parent.width;secondary:true;text:"Shared by collectors connected to this runner. These settings describe the workflow; agent permissions are configured above."}
    Column {
      width:parent.width;spacing:Style.space(12);visible:!!root.draft
      GridLayout {
        width:parent.width;columns:width>Style.space(620) ? 2 : 1;columnSpacing:Style.space(16);rowSpacing:Style.space(12)
        Column {
          Layout.fillWidth:true;Layout.minimumWidth:0;Layout.preferredWidth:1;spacing:Style.space(6);visible:!root.isDesk
          Label {width:parent.width;text:"Intended use";caption:true;secondary:true}
          SelectField {objectName:"app-workflow-purpose";width:parent.width;model:["Standard ship","Development","Software distribution"];currentIndex:root.draft && root.draft.ship ? ["standard","development","distribution"].indexOf(root.draft.ship.purpose) : 0;enabled:root.canSave;onActivated:root.setField("purpose",["standard","development","distribution"][currentIndex])}
        }
        Column {
          Layout.fillWidth:true;Layout.minimumWidth:0;Layout.preferredWidth:1;spacing:Style.space(6);visible:root.isDesk
          Label {width:parent.width;text:"Source workflow";caption:true;secondary:true}
          SelectField {objectName:"app-workflow-source";width:parent.width;model:["Unix source directory","Urgit"];currentIndex:root.draft && root.draft.desk && root.draft.desk.workflow==="urgit" ? 1 : 0;enabled:root.canSave;onActivated:root.setWorkflow(currentIndex===0 ? "mounted" : "urgit")}
        }
        Column {
          Layout.fillWidth:true;Layout.minimumWidth:0;Layout.preferredWidth:1;spacing:Style.space(6);visible:root.isDesk && !!root.draft && !!root.draft.desk && root.draft.desk.workflow==="mounted"
          Label {width:parent.width;text:"Directory inside the pier";caption:true;secondary:true}
          Field {objectName:"app-workflow-directory";width:parent.width;maximumLength:64;text:root.draft && root.draft.desk ? root.draft.desk.source_name : "";enabled:root.canSave;onTextEdited:root.setField("source_name",text)}
        }
        Column {
          Layout.fillWidth:true;Layout.minimumWidth:0;Layout.preferredWidth:1;spacing:Style.space(6);visible:root.isDesk
          Label {width:parent.width;text:"Publication method";caption:true;secondary:true}
          SelectField {objectName:"app-workflow-publication";width:parent.width;model:["Not configured","Clay source distribution","Treaty app publication"];currentIndex:root.draft && root.draft.desk ? ["none","source","treaty"].indexOf(root.draft.desk.publication) : 0;enabled:root.canSave;onActivated:root.setField("publication",["none","source","treaty"][currentIndex])}
        }
        Column {
          Layout.fillWidth:true;Layout.minimumWidth:0;Layout.preferredWidth:1;spacing:Style.space(6);visible:root.isDesk
          Label {width:parent.width;text:"Release branch (optional)";caption:true;secondary:true}
          Field {objectName:"app-workflow-branch";width:parent.width;maximumLength:128;placeholderText:"For example, master";text:root.draft && root.draft.desk ? root.draft.desk.branch : "";enabled:root.canSave;onTextEdited:root.setField("branch",text)}
        }
        Column {
          Layout.fillWidth:true;Layout.minimumWidth:0;Layout.preferredWidth:1;spacing:Style.space(6);visible:root.isDesk && !!root.draft && !!root.draft.desk && root.draft.desk.workflow==="urgit"
          Label {width:parent.width;text:"Urgit repository (optional)";caption:true;secondary:true}
          Field {objectName:"app-workflow-repository";width:parent.width;maximumLength:256;placeholderText:"Existing repository name";text:root.draft && root.draft.desk ? root.draft.desk.repository : "";enabled:root.canSave;onTextEdited:root.setField("repository",text)}
        }
      }
      Label {width:parent.width;visible:root.isDesk && !!root.draft && !!root.draft.desk && root.draft.desk.workflow==="urgit";text:"Urgit manages its own repository, branch binding and release gates. Record that existing configuration here; tokens stay in Urgit's tooling.";caption:true;secondary:true}
      Column {
        width:parent.width;spacing:Style.space(6);visible:root.isDesk
        Label {width:parent.width;text:"Release checks · one per line";caption:true;secondary:true}
        NotesField {objectName:"app-workflow-checks";width:parent.width;maximumLength:8192;enabled:root.canSave;placeholderText:"Describe the checks agents must complete";text:root.checksText;onEdited:function(value){if(root.checksText!==value)root.editChecks(value)}}
      }
      Label {width:parent.width;text:root.isDesk ? "Release instructions for agents" : "Development instructions for agents";caption:true;secondary:true}
      NotesField {objectName:"app-workflow-instructions";width:parent.width;enabled:root.canSave;placeholderText:"Describe how this ship or desk should be used";text:root.config ? root.config.instructions : "";onEdited:function(value){if(root.config && root.config.instructions!==value)root.setField("instructions",value)}}
      Label {width:parent.width;visible:!!root.validationError;text:root.validationError;color:Design.danger;caption:true}
      ActionButton {objectName:"app-workflow-save";text:"Save workflow";primary:true;enabled:root.canSave && root.dirty && !root.validationError;onClicked:root.save()}
    }
    Label {width:parent.width;visible:!root.draft && !controller.error;text:controller.busy ? "Loading workflow…" : "Select a resolved ship or desk to read its workflow.";caption:true;secondary:true}
    Label {width:parent.width;visible:!!controller.error;text:controller.error;color:Design.danger;caption:true}
    Label {width:parent.width;visible:!!controller.message;text:controller.message;color:Design.success;caption:true}
  }
}
