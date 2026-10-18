import QtQuick
import QtQuick.Controls as Controls
import QtQuick.Dialogs as Dialogs
import QtQuick.Layouts
import qs.Commons
import "Model.js" as Model

Column {
  id: root
  objectName: "external-pier-import"
  property var service
  property var connection
  property var submission: null
  readonly property bool submitted: !!submission
  property alias sourcePath: source.text
  readonly property string selection: connection ? connection.runner_id+"/"+connection.name : ""
  readonly property var review: service ? service.externalImportPreview : null
  readonly property bool supported: !!connection && Model.hasFeature(connection,"guided-pier-import")
  readonly property bool localHost: !!service && !!service.localSetupStatus && service.localSetupStatus.ready && !!connection && service.localSetupStatus.connection.runner_id===connection.runner_id
  readonly property bool current: !!service && Model.externalReviewCurrent(connection,review,source.text.trim(),service.now) && !!review.plan && review.plan.storage_root===location.rootID
  readonly property var plan: current ? review.plan : null
  readonly property bool inspecting: !!service && service.currentAction==="preview-pier"
  readonly property var outcome: submission ? (service && service.importOutcomes[Model.importRequestKey(submission)]) || submission : null
  readonly property var progress: Model.importProgress(service ? service.connections : [],outcome,service ? service.now : Date.now())
  onSelectionChanged: { reviewAnother();clearReview() }
  function clearReview() { consent.checked=false;if(service)service.externalImportPreview=null }
  function reviewAnother() {
    if (service && service.busy || submitted && progress.active) return false
    submission=null;clearReview();return true
  }
  function inspect() { if (submitted) return false;clearReview();return service.previewExternalPier(connection,source.text.trim(),location.rootID) }
  function submitImport() {
    if (submitted || !current) return false
    var target=service.importExternalPier(connection,review,consent.checked)
    if (!target) return false
    submission=target;return true
  }
  spacing: Style.space(10)
  Label { text: "Import an existing pier"; font.pixelSize: Style.space(21); font.bold: true }
  Label { width: parent.width; text: "Choose the stopped pier directory (the folder containing .urb) and a storage location. Omabit reads its identity and fake/live mode, chooses the instance path and HTTP port, and moves it safely. The ship stays stopped."; secondary: true }
  Label { width:parent.width;visible:!root.supported;text:"Update this runner to enable guided pier identification and import.";color:Design.warning }
  RowLayout {
    width:parent.width
    Field { id:source;objectName:"import-pier-source";Layout.fillWidth:true;placeholderText:"Existing pier directory on this host";enabled:!root.service.busy&&!root.submitted;onTextEdited:root.clearReview() }
    ActionButton { objectName:"import-pier-browse";text:"Browse…";visible:root.localHost;enabled:!root.service.busy&&!root.submitted;onClicked:folderPicker.open() }
  }
  Dialogs.FolderDialog { id:folderPicker;title:"Choose the existing pier directory";options:Dialogs.FolderDialog.DontUseNativeDialog;onAccepted:{source.text=Model.localPath(selectedFolder);root.clearReview()} }
  Label { width:parent.width;caption:true;secondary:true;visible:!root.localHost;text:"Enter the source path on the selected remote runner host." }
  StorageLocationSelector { id:location;width:parent.width;connection:root.connection;enabled:!root.submitted&&!root.service.busy;onEdited:root.clearReview() }
  ActionButton { objectName:"import-pier-review";text:root.inspecting ? "Inspecting pier…" : "Inspect pier";enabled:root.supported&&!root.service.busy&&!root.submitted&&Model.hostPath(source.text.trim());onClicked:root.inspect() }
  Controls.ProgressBar { width:parent.width;visible:root.inspecting;indeterminate:true }
  Label { objectName:"import-pier-plan";width:parent.width;visible:root.current;text:root.plan ? root.plan.ship+" · "+(root.plan.network==="fakenet" ? "Fakenet" : "Livenet (Vere live mode)")+"\nInstance: "+root.plan.instance_id+" · HTTP "+root.plan.http_port+"\nDestination: "+Model.absolutePierPath(root.connection,root.plan.destination)+"\n"+root.plan.source.files+" files · "+Model.bytesText(root.plan.source.bytes)+(root.plan.in_place ? " · already in place" : "") : "" }
  Label { width:parent.width;caption:true;secondary:true;text:"Testnet — coming soon: alternate Ethereum/Azimuth RPC endpoints. Offline fake/live metadata does not identify a live pier's Ethereum endpoint." }
  CheckField { id:consent;objectName:"import-pier-consent";width:parent.width;text:root.plan && root.plan.in_place ? "This pier is stopped everywhere. Enroll it at this location." : "This pier is stopped everywhere. Move it to the reviewed location.";enabled:root.current&&!root.submitted&&!root.service.busy }
  ActionButton { objectName:"import-pier-submit";primary:true;text:root.submitted ? "Import submitted" : root.plan && root.plan.in_place ? "Enroll stopped pier" : "Move and import pier";enabled:root.current&&!root.submitted&&!root.service.busy&&consent.checked;onClicked:root.submitImport() }
  Column {
    width:parent.width;spacing:Style.space(8);visible:root.submitted && !!root.outcome
    Label { objectName:"import-pier-progress-detail";width:parent.width;text:root.progress.detail }
    Controls.ProgressBar { objectName:"import-pier-progress";width:parent.width;from:0;to:1;value:root.progress.value;indeterminate:root.progress.indeterminate }
    Controls.ScrollView {
      width:parent.width;height:Style.space(140);visible:root.progress.log!==""
      Controls.TextArea { objectName:"import-pier-steps";text:root.progress.log;textFormat:TextEdit.PlainText;readOnly:true;selectByMouse:true;wrapMode:TextEdit.Wrap;color:Design.ink;font.pixelSize:Style.space(13);background:Rectangle {color:Design.background;radius:Style.space(5)} }
    }
  }
  ActionButton { text:"Review another pier";quiet:true;visible:root.submitted;enabled:!root.service.busy&&!root.progress.active;onClicked:root.reviewAnother() }
}
