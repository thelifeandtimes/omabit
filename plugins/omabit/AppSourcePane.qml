import QtQuick
import QtQuick.Layouts
import qs.Commons
import "AppSource.js" as Source

Column {
  id:root
  property var service
  property bool active:false
  property var runner:null
  property var target:({})
  property var setting:null
  property alias controller:source
  property bool expanded:false
  spacing:Style.space(8)
  AppSource {
    id:source;service:root.service;active:root.active && !!root.setting
    runnerName:root.runner ? root.runner.name : "";runnerID:root.runner ? root.runner.runner_id : ""
    mode:root.runner && root.runner.status ? root.runner.status.mode : "";target:root.target
    settingsRevision:root.setting ? root.setting.revision : 0
    sourceName:root.setting && root.setting.desk ? root.setting.desk.source_name : ""
  }
  Rectangle {width:parent.width;height:1;color:Design.line}
  RowLayout {
    width:parent.width
    Label {Layout.fillWidth:true;text:"Source changes"}
    ActionButton {objectName:"app-source-refresh";compact:true;quiet:true;text:source.busy ? "Checking…" : "Refresh";enabled:source.available && !source.busy;onClicked:source.refresh()}
  }
  Label {objectName:"app-source-summary";width:parent.width;text:source.error ? "Source status unavailable" : source.busy ? "Checking source…" : Source.title(source.value);color:source.error ? Design.danger : source.busy ? Design.muted : source.value && source.value.state==="synchronized" ? Design.success : Design.warning}
  Label {width:parent.width;visible:!!source.error;text:source.error;caption:true;secondary:true}
  Label {width:parent.width;visible:!!source.value;text:source.value ? (source.error || source.busy ? "Previous comparison · " : "Compared · ")+new Date(source.value.read_at).toLocaleString()+" · Clay revision "+source.value.observation.revision : "";caption:true;secondary:true}
  Label {width:parent.width;visible:!!source.value;text:source.value ? source.value.difference_count+" differing file"+(source.value.difference_count===1 ? "" : "s")+(source.value.conflict_count ? " · "+source.value.conflict_count+" conflict"+(source.value.conflict_count===1 ? "" : "s") : "") : "";caption:true;secondary:true}
  Label {width:parent.width;visible:!!source.value && source.value.state==="different";text:"No matching baseline has been observed for this source configuration, so the direction of these changes is not yet known.";caption:true;secondary:true}
  Label {width:parent.width;visible:!!source.value;text:source.value ? (source.value.autocommit==="active" ? "Active autocommit detected. Saving files can commit them to Clay." : source.value.autocommit==="unknown" ? "Autocommit status is unknown." : "No active autocommit timer observed.") : "";color:source.value && source.value.autocommit==="active" ? Design.warning : Design.muted;caption:true}
  ActionButton {objectName:"app-source-details";visible:!!source.value && source.value.difference_count>0;compact:true;quiet:true;text:root.expanded ? "Hide file details" : "Show file details";onClicked:root.expanded=!root.expanded}
  Flickable {
    width:parent.width;height:visible ? Math.min(files.implicitHeight,Style.space(220)) : 0
    visible:root.expanded && !!source.value && source.value.difference_count>0
    clip:true;contentHeight:files.implicitHeight;boundsBehavior:Flickable.StopAtBounds
    Column {
      id:files;width:parent.width;spacing:Style.space(6)
      Repeater {
        model:source.value ? source.value.differences : []
        delegate:Column {
          required property var modelData
          width:files.width;spacing:Style.space(2)
          Label {width:parent.width;text:modelData.path;caption:true}
          Label {width:parent.width;text:({"modified":"Different contents","only-unix":"Only in Unix source","only-clay":"Only in Clay"})[modelData.kind];caption:true;secondary:true}
        }
      }
    }
  }
  Label {width:parent.width;visible:root.expanded && !!source.value && source.value.difference_count>source.value.differences.length;text:source.value ? "Showing the first "+source.value.differences.length+" of "+source.value.difference_count+" differing files." : "";caption:true;secondary:true}
  Label {width:parent.width;caption:true;secondary:true;text:"Comparison does not commit or publish. Source equality does not establish application compilation or readiness."}
}
