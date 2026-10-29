import QtQuick
import QtQuick.Controls as Controls
import qs.Commons
import "Model.js" as Model

Column {
  id: root
  objectName: "import-source-restoration"
  property var service
  property var connection
  signal inspectRequested(string path)
  readonly property var controller: service ? service.importRestoration : null
  function matches(t) { return !!t && !!connection && t.runner===connection.runner_id && t.connection===connection.name }
  readonly property var review: controller && controller.review && matches(controller.review.target) ? controller.review : null
  readonly property bool inspecting: !!controller && matches(controller.pendingPreview)
  readonly property bool current: !!controller && controller.currentReview(review)
  readonly property bool unresolved: !!review && controller.unresolved(review.target)
  readonly property var history: controller ? Object.keys(controller.outcomes).map(function(k){return controller.outcomes[k]}).filter(function(o){return root.matches(o.target)}) : []
  readonly property string problem: controller && matches(controller.selection) ? controller.error : ""
  onReviewChanged: consent.checked=false
  onCurrentChanged: if (!current) consent.checked=false
  width:parent.width
  spacing:Style.space(10)
  visible:inspecting || !!review || history.length>0 || problem!==""
  Label { width:parent.width;text:"Import restoration";font.bold:true;font.pixelSize:Style.space(20) }
  Label { width:parent.width;visible:root.inspecting;text:"Reviewing the retained source and partial destination…";secondary:true }
  Controls.ProgressBar { width:parent.width;visible:root.inspecting;indeterminate:true }
  Column {
    width:parent.width;spacing:Style.space(8);visible:!!root.review
    Label { objectName:"restore-subject";width:parent.width;text:root.review ? root.review.target.ship+" · "+root.review.target.instance : "";font.bold:true }
    Label { objectName:"restore-moves";width:parent.width;text:root.review ? "Restore source:\n"+root.review.plan.source+"\n→ "+root.review.plan.restore_to+"\n\nRetain partial copy:\n"+root.review.plan.destination+"\n→ "+root.review.plan.retain_at : "" }
    Label { objectName:"restore-counts";width:parent.width;caption:true;secondary:true;text:root.review ? "Source: "+root.review.plan.source_files+(root.review.plan.source_files===1 ? " file · " : " files · ")+Model.bytesText(root.review.plan.source_bytes)+"\nPartial copy: "+root.review.plan.destination_files+(root.review.plan.destination_files===1 ? " file · " : " files · ")+Model.bytesText(root.review.plan.destination_bytes) : "" }
    Label { objectName:"restore-continuation";width:parent.width;visible:!!root.review && root.review.plan.continuation;text:"Continue the recorded restoration. The runner verifies moves already made before completing the remaining steps.";secondary:true }
    Label { width:parent.width;text:"Both pier trees are retained. Completion removes this failed registration. The ship stays stopped; importing it again is a separate review.";secondary:true }
    CheckField { id:consent;objectName:"restore-consent";width:parent.width;enabled:root.current && !root.unresolved && !root.service.busy;text:"Restore the source to its original path and retain the partial copy in the reviewed archive." }
    ActionButton { objectName:"restore-submit";primary:true;text:root.unresolved ? "Restoration submitted" : root.review && root.review.plan.continuation ? "Continue restoration" : "Restore source and retain copy";enabled:root.current && !root.unresolved && !root.service.busy && consent.checked;onClicked:root.controller.restore(root.review,consent.checked) }
    Label { width:parent.width;visible:!root.current && !root.unresolved;text:"This review is no longer current. Refresh the stopped import and review it again.";color:Design.warning }
    ActionButton { objectName:"restore-close-review";quiet:true;text:"Close review";visible:!root.unresolved;enabled:!root.service.busy;onClicked:root.controller.review=null }
  }
  Label { objectName:"restore-error";width:parent.width;visible:root.problem!=="";text:root.problem;color:Design.warning }
  Repeater {
    model:root.history
    Column {
      id:receipt
      required property var modelData
      objectName:"restore-receipt-"+modelData.request
      width:root.width;spacing:Style.space(6)
      Label { width:parent.width;text:receipt.modelData.target.ship+" · "+receipt.modelData.target.instance+" · "+({succeeded:"Restored",queued:"Restoring",running:"Restoring",submitting:"Submitting",uncertain:"Checking result",failed:"Needs review",interrupted:"Needs review",canceled:"Canceled",rejected:"Not submitted"})[receipt.modelData.status];font.bold:true }
      Label { objectName:"restore-receipt-message";width:parent.width;text:receipt.modelData.message; color:receipt.modelData.status==="succeeded" ? Design.success : Design.muted }
      Label { width:parent.width;text:(receipt.modelData.status==="succeeded" ? "Restored source: " : "Reviewed source location: ")+receipt.modelData.plan.restore_to+"\nArchive destination: "+receipt.modelData.plan.retain_at;caption:true }
      Label { width:parent.width;visible:!root.controller.terminal(receipt.modelData.status);text:"Keep this request while its result is uncertain. Refresh the runner or check Activity; another restoration will not be submitted automatically.";caption:true;secondary:true }
      Label { objectName:"restore-request-id";width:parent.width;text:"Request: "+receipt.modelData.request;caption:true;secondary:true }
      ActionButton { objectName:"restore-review-again";text:"Review restoration again";quiet:true;visible:root.controller.canReviewAgain(receipt.modelData);enabled:!root.service.busy;onClicked:root.controller.reviewAgain(receipt.modelData) }
      ActionButton { objectName:"restore-inspect-source";text:"Inspect restored pier…";quiet:true;visible:receipt.modelData.status==="succeeded";enabled:!root.service.busy && root.service.observing && Model.available(root.connection,root.service.now) && root.connection.access_role==="operator" && Model.hasFeature(root.connection,"guided-pier-import");onClicked:root.inspectRequested(receipt.modelData.plan.restore_to) }
    }
  }
}
