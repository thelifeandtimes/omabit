import QtQuick
import qs.Commons
import "Model.js" as Model

Column {
  id: root
  objectName: "import-recovery-"+(instance ? instance.id : "")
  property var service
  property var connection
  property var instance
  // Fleet snapshots recreate delegates. Keep the review and submitted request
  // in the service so a poll cannot erase a review or lose its receipt.
  readonly property var target: service && matches(service.importRecoveryReview) ? service.importRecoveryReview : null
  readonly property var outcome: service && service.storageOutcome && service.storageOutcome.kind==="recover-import" && matches(service.storageOutcome) ? service.storageOutcome : null
  readonly property string request: outcome ? outcome.request : ""
  function matches(value) { return !!value && !!connection && !!instance && value.runner===connection.runner_id && value.connection===connection.name && value.instance===instance.id && value.revision===instance.config_revision }
  readonly property bool available: !!service && service.observing && Model.canRecoverImport(connection,instance,service.now)
  readonly property bool current: !!service && service.observing && Model.currentImportRecovery(connection,instance,target,service.now)
  readonly property var receipt: request && connection && connection.snapshot ? (connection.snapshot.operations || []).filter(function(o){return o.kind==="recover-import" && o.instance_id===root.instance.id && o.request_id===root.request && o.config_revision===root.instance.config_revision})[0] || null : null
  width:parent.width
  spacing:Style.space(8)
  visible:available || request!==""
  Label { width:parent.width;text:"Finish interrupted enrollment";font.bold:true }
  Label { width:parent.width;caption:true;secondary:true;text:"Revalidate the retained pier and complete its enrollment. The ship remains stopped for a separate Start." }
  ActionButton { objectName:"review-import-recovery";text:"Review recovery…";visible:!root.target && !root.request;enabled:root.available && !root.service.busy;onClicked:root.service.importRecoveryReview=Model.importRecoveryTarget(root.connection,root.instance) }
  Column {
    width:parent.width;spacing:Style.space(8);visible:!!root.target
    Label { width:parent.width;text:root.target ? root.target.ship+" · "+root.target.instance+"\nRetained pier: "+Model.absolutePierPath(root.connection,root.target.path)+"\nSource: "+root.target.source+"\nHolding path: "+(root.target.holding || "none — in-place import") : "";caption:true }
    ActionButton { objectName:"submit-import-recovery";text:root.request ? "Recovery submitted" : "Validate and finish enrollment";primary:true;enabled:root.current && !root.request && !root.service.busy;onClicked:root.service.recoverImport(root.target) }
    Label { width:parent.width;visible:!root.current && !root.request;text:"The selection changed; review it again.";color:Design.warning }
    ActionButton { text:"Close review";quiet:true;visible:!root.request;onClicked:root.service.importRecoveryReview=null }
  }
  Label { objectName:"import-recovery-receipt";width:parent.width;visible:!!root.request;text:root.receipt ? root.receipt.status==="succeeded" ? "Enrollment recovered. The ship is stopped; choose Start to verify its runtime identity." : root.receipt.status==="failed" || root.receipt.status==="interrupted" ? "Recovery needs review: "+(root.receipt.message || root.receipt.status) : "Validating retained import…" : root.outcome && root.outcome.status==="failed" ? "Recovery request refused: "+root.outcome.message : "Waiting for the runner's recorded result. Check Activity before retrying.";color:root.receipt && root.receipt.status==="succeeded" ? Design.success : Design.muted }
  ActionButton { objectName:"review-import-recovery-again";text:"Review recovery again";quiet:true;visible:!!root.request && (root.outcome && root.outcome.status==="failed" || root.receipt && ["failed","interrupted"].indexOf(root.receipt.status)>=0);enabled:root.available && !root.service.busy;onClicked:{root.service.storageOutcome=null;root.service.importRecoveryReview=null} }
}
