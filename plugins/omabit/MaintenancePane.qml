import QtQuick
import QtQuick.Controls as Controls
import QtQuick.Layouts
import qs.Commons
import "Model.js" as Model

Column {
  id: root
  property var service
  property var connection
  property var instance
  property var maintenanceTarget: null
  property var retentionTarget: null
  property string section: "manual"
  signal revealRequested(var item)
  readonly property alias routinePane: routinePolicy
  readonly property string selectionKey: connection && instance ? connection.runner_id + "/" + instance.id : ""
  readonly property var inspection: service && service.epochInspection && connection && instance && service.epochInspection.runner === connection.runner_id && service.epochInspection.instance === instance.id ? service.epochInspection : null
  readonly property bool inspecting: !!inspection && ["submitting","queued","running","uncertain"].indexOf(inspection.status)>=0
  onSelectionKeyChanged: clearReview()
  onVisibleChanged: if (!visible) clearReview()
  spacing: Style.space(16)
  Label { width: parent.width; visible: root.instance && root.instance.mode === "comet" && !Model.hasFeature(root.connection,"comet-prune-prefix"); text: "This runner supports comet history previews only. Update the runner for confirmed history removal."; color: Design.warning }
  function clearReview() { maintenanceTarget = null; retentionTarget = null; consent.checked = false; if (service) service.clearEpochPreview() }
  function reviewMaintenance(c, i, kind) { clearReview(); maintenanceTarget = Model.maintenanceTarget(c,i,kind,service.now) }
  function reviewRetention(c, i) { clearReview(); retentionTarget = Model.epochTarget(c,i,service.now); Qt.callLater(function(){if (root.visible && root.retentionTarget) root.revealRequested(historyReview)}) }
  function selectCutoff(name) { service.clearEpochPreview(); consent.checked = false; retentionTarget = Model.selectedEpochTarget(retentionTarget,name) }
  Connections {
    target: root.service
    function onEpochPreviewChanged() { consent.checked = false }
    function onCanConfirmPruneChanged() { if (!root.service.canConfirmPrune) consent.checked = false }
    function onEpochInspectionChanged() {
      if (root.visible && root.section === "manual" && root.inspection && root.inspection.status === "ready" && Model.canReviewEpochs(root.connection,root.instance,root.service.now)) root.reviewRetention(root.connection,root.instance)
    }
  }
  RowLayout {
    ActionButton { text: "Manual maintenance"; primary: root.section === "manual"; onClicked: root.section = "manual" }
    ActionButton { text: "Routine maintenance"; primary: root.section === "routine"; onClicked: { root.clearReview(); root.section = "routine"; Qt.callLater(function(){root.revealRequested(routinePolicy)}) } }
  }
  RoutinePane { id: routinePolicy; width: parent.width; visible: root.section === "routine"; service: root.service; connection: root.connection; instance: root.instance }
  Column {
    width: parent.width; spacing: Style.space(14); visible: root.section === "manual"
    Label { width: parent.width; text: "Maintenance operates on a clean stopped pier. Stop and resume remain explicit actions."; secondary: true }
    Repeater {
      model: [{kind:"pack",title:"Pack",description:"Compact the event database."}, {kind:"meld",title:"Meld",description:"Compact loom storage and pack events."}, {kind:"roll",title:"Roll epoch",description:"Create a new snapshot-bearing epoch."}]
      Card {
        id: action
        required property var modelData
        width: root.width; implicitHeight: Style.space(76)
        RowLayout {
          anchors.fill: parent; anchors.margins: Style.space(14)
          Column { Layout.fillWidth: true; spacing: Style.space(6)
            Label { text: action.modelData.title; font.bold: true }
            Label { text: action.modelData.description; secondary: true; caption: true }
          }
          ActionButton { text: "Review…"; enabled: root.service && root.service.observing && !root.service.busy && Model.canOperate(root.connection,root.instance,action.modelData.kind,root.service.now); onClicked: root.reviewMaintenance(root.connection,root.instance,action.modelData.kind) }
        }
      }
    }
    Label { visible: root.instance && !!root.instance.roll_requires_run; text: "Another roll requires a healthy runtime start, followed by an explicit stop."; color: Design.warning; width: parent.width }
    Card {
      width: parent.width; implicitHeight: retain.implicitHeight + Style.space(28)
      Column {
        id: retain
        anchors.left: parent.left; anchors.right: parent.right; anchors.top: parent.top; anchors.margins: Style.space(14); spacing: Style.space(10)
        Label { text: "Epoch history & retention"; font.bold: true }
        Label { width: parent.width; text: "An epoch is a segment of the ship's event log. Rolling starts a new segment with a snapshot. History removal selects the oldest consecutive epochs; the newest two always stay protected."; secondary: true }
        Label { width: parent.width; text: "Inspect reads the stopped pier and opens its inventory below. It does not delete history or start the ship."; secondary: true; caption: true }
        Label { width: parent.width; visible: root.instance && root.instance.observed_state === "running"; text: "Stop the ship to inspect its history."; color: Design.warning }
        Flow {
          width: parent.width; spacing: Style.space(8)
          ActionButton { objectName: "inspect-epochs"; text: root.inspecting ? "Inspecting…" : "Inspect epochs"; enabled: root.service && root.service.observing && !root.service.busy && !root.inspecting && Model.canOperate(root.connection,root.instance,"inspect-epochs",root.service.now); onClicked: { root.clearReview(); root.service.operate(root.connection,root.instance,"inspect-epochs") } }
          ActionButton { text: "Review retention…"; enabled: root.service && root.service.observing && !root.service.busy && Model.canReviewEpochs(root.connection,root.instance,root.service.now); onClicked: root.reviewRetention(root.connection,root.instance) }
        }
        Label { objectName: "inspection-feedback"; width: parent.width; visible: !!root.inspection; text: root.inspection ? root.inspection.message : ""; color: root.inspection && root.inspection.status === "ready" ? Design.success : Design.accent }
      }
    }
  }
  Card {
    width: parent.width; visible: !!root.maintenanceTarget; implicitHeight: review.implicitHeight + Style.space(28); border.color: Design.accent
    Column {
      id: review
      anchors.left: parent.left; anchors.right: parent.right; anchors.top: parent.top; anchors.margins: Style.space(14); spacing: Style.space(12)
      Label { text: root.maintenanceTarget ? "Confirm " + root.maintenanceTarget.kind : ""; font.bold: true; font.pixelSize: Style.space(19) }
      Label { width: parent.width; text: root.maintenanceTarget ? root.maintenanceTarget.connection + " / " + root.maintenanceTarget.instance + " · revision " + root.maintenanceTarget.revision : "" }
      Label { width: parent.width; text: "Only this operation will run. The pier remains stopped afterward; runtime start is a separate action."; secondary: true }
      Label { visible: root.maintenanceTarget && !root.service.canMaintain(root.maintenanceTarget); text: "Selection changed or unavailable. Cancel and review again."; color: Design.warning; width: parent.width }
      RowLayout {
        ActionButton { objectName: "confirm-maintenance"; text: root.maintenanceTarget ? "Run " + root.maintenanceTarget.kind : "Run"; primary: true; enabled: root.service && root.service.canMaintain(root.maintenanceTarget); onClicked: if (root.service.maintain(root.maintenanceTarget)) root.maintenanceTarget = null }
        ActionButton { text: "Cancel"; onClicked: root.clearReview() }
      }
    }
  }
  Column {
    id: historyReview
    width: parent.width; visible: !!root.retentionTarget; spacing: Style.space(12)
    objectName: "history-review"
    Label { text: "Retained epochs · oldest to newest"; font.pixelSize: Style.space(20); font.bold: true }
    Label { width: parent.width; text: root.retentionTarget ? "Inspected " + Model.humanTime(root.retentionTarget.inventory.observed_at,root.service.now) + " · " + root.retentionTarget.inventory.entries.length + " epochs\nChoose the last old epoch to remove. Every earlier epoch is included; protected entries stay." : ""; secondary: true; caption: true }
    Repeater {
      model: root.retentionTarget ? root.retentionTarget.inventory.entries : []
      RowLayout {
        required property var modelData
        width: root.width
        Column {
          Layout.fillWidth: true; Layout.minimumWidth: 0; spacing: Style.space(4)
          Label { width: parent.width; text: "Epoch " + parent.parent.modelData.name; font.bold: true; elide: Text.ElideMiddle; wrapMode: Text.NoWrap }
          Label { width: parent.width; text: Model.epochBoundary(parent.parent.modelData.name); caption: true; secondary: true }
          Label { width: parent.width; text: Model.epochDate(root.instance,parent.parent.modelData.name,root.service.now); caption: true; secondary: true }
        }
        Label { text: Model.bytesText(modelData.logical_bytes); caption: true; secondary: true }
        ActionButton { property var epoch: parent.modelData; objectName: "cutoff-" + epoch.name; text: epoch.protected ? "Protected" : root.retentionTarget && root.retentionTarget.cutoff && !Model.epochLess(root.retentionTarget.cutoff,epoch.name) ? "Selected" : "Remove through here"; compact: true; primary: root.retentionTarget && root.retentionTarget.cutoff && !Model.epochLess(root.retentionTarget.cutoff,epoch.name); enabled: !epoch.protected && root.service.canReviewTarget(root.retentionTarget); onClicked: root.selectCutoff(epoch.name) }
      }
    }
    Label { width: parent.width; visible: root.retentionTarget && !root.service.busy && !root.service.canReviewTarget(root.retentionTarget); text: "Review expired or changed. Inspect again; the selected digest is never refreshed automatically."; color: Design.warning }
    RowLayout {
      ActionButton { objectName: "preview-prefix"; text: "Preview history removal"; enabled: root.retentionTarget && !!root.retentionTarget.cutoff && root.service.canReviewTarget(root.retentionTarget); onClicked: root.service.previewEpochs(root.retentionTarget) }
      ActionButton { text: "Cancel"; onClicked: root.clearReview() }
    }
    Card {
      width: parent.width; visible: !!root.service.epochPreview; implicitHeight: previewBody.implicitHeight + Style.space(28); border.color: Design.warning
      Column {
        id: previewBody
        readonly property var preview: root.service.epochPreview
        anchors.left: parent.left; anchors.right: parent.right; anchors.top: parent.top; anchors.margins: Style.space(14); spacing: Style.space(12)
        Label { width: parent.width; text: previewBody.preview ? "REMOVE  " + previewBody.preview.selected.map(function(e) { return e.name }).join(", ") : ""; color: Design.danger }
        Label { width: parent.width; text: previewBody.preview ? "RETAIN  " + previewBody.preview.retained.map(function(e) { return e.name }).join(", ") : ""; color: Design.success }
        Label { width: parent.width; text: previewBody.preview ? Model.bytesText(previewBody.preview.selected_logical_bytes) + " logical · " + Model.bytesText(previewBody.preview.selected_allocated_bytes) + " allocated. Reclaimed disk space may differ." : ""; secondary: true; caption: true }
        Label { width: parent.width; text: "Deletion permanently removes this history and its snapshots. Retained metadata is not a verified backup." }
        Label { visible: previewBody.preview && !previewBody.preview.deletion_supported; text: "Preview only: unsupported deletion layout or capability."; color: Design.warning; width: parent.width }
        CheckField {
          id: consent; objectName: "confirm-history-loss"; width: parent.width; enabled: root.service.canConfirmPrune
          text: "I accept irreversible loss of the exact history listed above."
        }
        ActionButton { objectName: "delete-prefix"; text: "Delete selected history"; danger: true; enabled: consent.checked && root.service.canConfirmPrune; onClicked: if (root.service.prunePreview(consent.checked)) root.retentionTarget = null }
      }
    }
  }
}
