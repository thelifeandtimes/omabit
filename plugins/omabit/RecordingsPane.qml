import QtQuick
import QtQuick.Controls as Controls
import QtQuick.Layouts
import qs.Commons
import "Recordings.js" as RecordingModel

Card {
  id:root
  objectName:"recordings-pane"
  property var service
  property var connection
  property var instance
  property bool active:false
  property alias controller:reader
  implicitHeight:content.implicitHeight+Style.space(32)
  Recordings {id:reader;service:root.service;connection:root.connection;instance:root.instance;active:root.active}
  Column {
    id:content
    anchors.left:parent.left;anchors.right:parent.right;anchors.top:parent.top;anchors.margins:Style.space(16)
    spacing:Style.space(12)
    CardTitle {text:"Saved runtime recordings"}
    Label {width:parent.width;secondary:true;text:"Read retained output from earlier launches, including removed containers. Saved intent does not prove capture or complete coverage."}
    Label {width:parent.width;visible:!reader.available;secondary:true;text:"Select an available operator connection with retained-recording support. Reading history never starts capture."}
    SwitchField {
      objectName:"recordings-private-consent";text:"Allow private recording reads for this selection"
      checked:reader.consent;enabled:reader.available
      onToggled:reader.consent=checked
    }
    Label {width:parent.width;caption:true;secondary:true;text:"Application output can contain private data. Redacted views remove known credential fields; review content before sharing. Originals require a separate explicit choice below."}
    Flow {
      width:parent.width;spacing:Style.space(8)
      ActionButton {objectName:"recordings-history";text:"Read history";enabled:reader.available && reader.consent && !reader.busy;onClicked:reader.readHistory(false)}
      ActionButton {objectName:"recordings-next-history";text:"Next saved sessions";visible:!!reader.history && !!reader.history.next_job_id;enabled:reader.available && reader.consent && !reader.busy;onClicked:reader.readHistory(true)}
      ActionButton {objectName:"recordings-controls";text:"Read retention settings";enabled:reader.controlsAvailable && reader.consent && !reader.busy;onClicked:reader.readControls()}
    }
    Label {width:parent.width;visible:reader.busy;text:"Reading saved recording evidence…";caption:true;secondary:true}
    Label {objectName:"recordings-error";width:parent.width;visible:!!reader.error;text:reader.error;color:Design.danger}
    Label {width:parent.width;visible:!!reader.history && !reader.history.sessions.length;text:"No saved recording launch intent in this history window. This does not establish that no output existed.";secondary:true}
    Repeater {
      model:reader.history ? reader.history.sessions : []
      ActionButton {
        required property var modelData
        width:content.width
        objectName:"recording-session-"+modelData.job_id
        text:modelData.created_at+" · "+modelData.operation_id+" · "+modelData.job_id.slice(0,12)
        primary:!!reader.selected && reader.selected.job_id===modelData.job_id
        enabled:reader.available && reader.consent && !reader.busy
        onClicked:reader.selectJob(modelData)
      }
    }
    Column {
      width:parent.width;spacing:Style.space(10);visible:!!reader.selected
      Label {width:parent.width;text:reader.selected ? "Job "+reader.selected.job_id+"\nContainer "+reader.selected.container_id : "";caption:true;secondary:true}
      Flow {
        width:parent.width;spacing:Style.space(8)
        ActionButton {objectName:"recording-read-redacted";text:"Read redacted window";enabled:reader.available && reader.consent && !reader.busy;onClicked:reader.readWindow("redacted",false)}
        ActionButton {objectName:"recording-read-original";text:"Read original bytes as base64";enabled:reader.available && reader.consent && !reader.busy;onClicked:reader.readWindow("original",false)}
        ActionButton {objectName:"recording-next-window";text:"Read following window";visible:!!reader.window && !!reader.window.next;enabled:reader.available && reader.consent && !reader.busy;onClicked:reader.readWindow(reader.window.representation,true)}
      }
    }
    Column {
      width:parent.width;spacing:Style.space(8);visible:!!reader.window
      Label {
        objectName:"recording-coverage";width:parent.width;caption:true;secondary:true
        text:!reader.window ? "" : "Saved state: "+reader.window.status.state+" · storage: "+reader.window.status.session_storage+"\n"+reader.window.frame_count+(reader.window.frame_count===1 ? " frame in this window" : " frames in this window")+" · "+reader.window.tail+"\n"+(reader.window.status.writer_present ? "Writer lock present; health is unverified." : "No writer lock observed; this is not exit evidence.")+"\nObserved exit saved: "+(reader.window.status.exit_saved===true ? "yes" : "not established")+" · closed durably: "+(reader.window.status.closed_durably===true ? "yes" : "not established")
      }
      Label {width:parent.width;text:"Window only · not a complete export. End-of-file is a read snapshot, not process exit. Omitted text and coverage gaps remain explicit.";caption:true;secondary:true}
      Label {width:parent.width;visible:!!reader.window && reader.window.representation==="original";text:"Private originals · unchanged octets encoded as base64. This view does not decode or execute terminal control sequences.";caption:true;color:Design.warning}
      Controls.ScrollView {
        width:parent.width;height:Style.space(300);contentWidth:availableWidth
        Controls.TextArea {
          objectName:"recording-window-text";text:RecordingModel.windowText(reader.window)
          textFormat:TextEdit.PlainText;readOnly:true;selectByMouse:true;wrapMode:TextEdit.WrapAnywhere
          color:Design.ink;selectionColor:Design.accent;selectedTextColor:Design.background
          font.family:"monospace";font.pixelSize:Style.space(12)
          background:Rectangle {color:Design.background;radius:Style.space(5)}
        }
      }
    }
    Column {
      width:parent.width;spacing:Style.space(8);visible:!!reader.controls
      Label {width:parent.width;text:!reader.controls ? "" : "Recording pilot: "+(reader.controls.pilot_enabled ? "enabled for new ordinary launches" : "not enabled")+"\n"+reader.controls.state.policy.max_jobs+" job limit · "+RecordingModel.bytes(reader.controls.state.policy.max_session_bytes)+" per session\nHost limit "+RecordingModel.bytes(reader.controls.state.policy.host_max_raw_bytes)+" · default ship limit "+RecordingModel.bytes(reader.controls.state.policy.instance_max_raw_bytes)+"\nFree-space floor "+RecordingModel.bytes(reader.controls.state.policy.reserve_free_bytes)+" · "+reader.controls.state.pins.length+" saved incident/owner pins";caption:true;secondary:true}
      Label {objectName:"recording-capacity";width:parent.width;text:!reader.controls ? "" : reader.controls.capacity_status==="available" ? "Admission commitments: "+RecordingModel.bytes(reader.controls.capacity.committed_bytes)+" across "+reader.controls.capacity.jobs+" retained jobs. Every job charges its full original maximum, including closed jobs; this is not measured disk use." : reader.controls.capacity_status==="unknown" ? "Capacity unknown · new recording admissions are refused. Preserve the saved jobs and investigate their storage." : "Pilot disabled · saved history may still be available.";caption:true;secondary:true}
      Label {objectName:"recording-storage-usage";width:parent.width;text:RecordingModel.usageText(reader.controls,root.instance ? root.instance.id : "");caption:true;secondary:true}
      Label {width:parent.width;text:"No data is deleted by these reads. Automatic and age-based retirement are unavailable in this version; changing settings does not change existing worker limits.";caption:true;secondary:true}
    }
  }
}
