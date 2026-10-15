import QtQuick
import QtQuick.Layouts
import qs.Commons
import "Model.js" as Model

Column {
  id: root
  objectName: "pier-storage"
  property var service
  property var preferredConnection: null
  property var selected: null
  property var moving: null
  property bool submitted: false
  property string draftName: ""
  property string draftShip: ""
  property string draftPort: ""
  property bool confirmed: false
  property string reviewError: ""
  property bool showImport: false
  readonly property var runners: service ? service.connections.filter(function(c){return Model.hasFeature(c,"pier-storage")}) : []
  readonly property var connection: preferredConnection || runners[runner.currentIndex] || null
  readonly property var catalogue: connection && connection.snapshot ? connection.snapshot.piers : null
  spacing: Style.space(12)
  onConnectionChanged: { if (selected && (!connection || selected.runner!==connection.runner_id)) selected=null }
  Connections {
    target: root.service
    function onStorageOutcomeChanged() {
      var o=root.service.storageOutcome
      if (o && root.connection && o.runner===root.connection.runner_id && o.status==="failed" && (root.selected && o.path===root.selected.source.path || root.moving && o.instance===root.moving.id)) root.submitted=false
    }
  }
  function choose(source) {
    if (source.enrolled_by && source.moved !== true) { reviewError="This pier is already enrolled as "+source.enrolled_by+"."; return }
    selected={runner:connection.runner_id,source:JSON.parse(JSON.stringify(source))};moving=null;submitted=false;confirmed=false;reviewError=""
    var i=Model.instances(connection).filter(function(i){return i.id===source.enrolled_by})[0]
    draftName=i ? i.id : source.path.split("/").slice(-2,-1)[0]
    draftShip=i ? i.ship : source.kind === "fake" ? "~"+draftName : ""
    draftPort=i ? String(i.config.http_port) : Model.suggestHTTPPort(connection)
  }
  function submitEnrollment() {
    if (!selected) return
    reviewError=""
    if (service.enrollDiscovered(connection,selected.source,draftName.trim(),draftShip.trim(),draftPort,confirmed)) submitted=true
    else reviewError=service.lastError || "This review is no longer current. Refresh discovery and review the pier again."
  }
  Label { text: "Pier storage"; font.pixelSize: Style.space(22); font.bold: true }
  Label { width: parent.width; text: "Import an existing stopped pier by its host path, choose storage locations, or manage discovered and registered piers."; secondary: true }
  SelectField { id: runner; width: parent.width; visible: !root.preferredConnection; model: root.runners.map(function(c){return Model.runnerTitle(c,root.service.runnerPreferences)}); onActivated: { root.selected=null;root.moving=null } }
  Label { width: parent.width; text: root.catalogue ? "Root: "+root.catalogue.root : "Select an updated runner."; caption: true; secondary: true }
  Label { width: parent.width; visible: !!root.connection && !Model.hasFeature(root.connection,"storage-locations"); text: "Update this runner to enable configurable locations, host-path import and pier deletion."; color: Design.warning }
  ActionButton { objectName: "open-pier-import"; text: root.showImport ? "Close import form" : "+ Import an existing pier…"; primary: !root.showImport; onClicked: root.showImport=!root.showImport }
  PierImportPane { width: parent.width; service: root.service; connection: root.connection; visible: root.showImport }
  StorageRootsPane { width: parent.width; service: root.service; connection: root.connection }
  Label { width: parent.width; text: "fakeships/[project/]<ship-directory>/pier\nlivenet/{galaxies,stars,planets,moons,comets}/[group/]<ship-directory>/pier\nEach ship directory is an exclusive permanent mount. archive/ is excluded from discovery and Start. Cross-host moves are on each ship's Overview page."; caption: true; secondary: true }
  Label { width: parent.width; visible: root.catalogue && !!root.catalogue.error; text: root.catalogue ? root.catalogue.error || "" : ""; color: Design.warning }
  ActionButton { text: "Refresh discovery"; enabled: root.service && !root.service.busy; onClicked: root.service.refresh() }
  Label { text: "Discovered piers"; font.bold: true }
  Repeater {
    model: root.catalogue ? root.catalogue.candidates || [] : []
    Card {
      id: candidate
      required property var modelData
      readonly property bool expanded: !!root.selected && root.selected.source.path===modelData.path
      readonly property bool needsReview: !modelData.enrolled_by || modelData.moved === true
      width: root.width; implicitHeight: contents.implicitHeight+Style.space(24)
      Column {
        id: contents
        anchors.left: parent.left;anchors.right: parent.right;anchors.top: parent.top;anchors.margins: Style.space(12);spacing: Style.space(8)
        Label { width: parent.width; text: candidate.modelData.path; font.bold: true }
        Label { width: parent.width; text: candidate.modelData.problem || (candidate.modelData.enrolled_by ? (candidate.modelData.moved ? "Moved — reconnect ":"Enrolled as ")+candidate.modelData.enrolled_by : "Unenrolled pier")+(candidate.modelData.busy ? " · Vere lock present" : ""); caption: true; secondary: true }
        ActionButton { objectName: "review-pier-"+candidate.modelData.path; text: candidate.modelData.moved === true ? "Reconnect location…" : "Enroll for management…"; visible: candidate.needsReview && !candidate.expanded; enabled: candidate.needsReview && root.connection && root.connection.access_role==="operator" && !root.service.busy && !candidate.modelData.problem && !candidate.modelData.busy; onClicked: root.choose(candidate.modelData) }
        Column {
          objectName: "pier-enrollment-"+candidate.modelData.path
          width: parent.width; visible: candidate.expanded; spacing: Style.space(10)
          Rectangle { width: parent.width; height: 1; color: Design.line }
          Label { text: root.selected && root.selected.source.enrolled_by ? "Reconnect this stopped pier" : "Enroll this stopped pier"; font.bold: true }
          Field { width: parent.width; text: root.draftName; placeholderText: "Local instance name"; enabled: !candidate.modelData.enrolled_by && !root.submitted; onTextEdited: root.draftName=text }
          Field { width: parent.width; text: root.draftShip; placeholderText: "Expected ~ship identity"; enabled: !candidate.modelData.enrolled_by && !root.submitted; onTextEdited: root.draftShip=text }
          Field { width: parent.width; text: root.draftPort; placeholderText: "Host HTTP port"; visible: !candidate.modelData.enrolled_by; enabled: !root.submitted; onTextEdited: root.draftPort=text }
          CheckField { objectName: "enroll-pier-consent"; width: parent.width; checked: root.confirmed; onToggled: root.confirmed=checked; text: "This pier is fully transferred and stopped; it will not run elsewhere."; enabled: !root.submitted }
          Label { width: parent.width; text: "Enrollment leaves it stopped. Start verifies the expected identity."; secondary: true; caption: true }
          Label { width: parent.width; visible: root.reviewError!==""; text: root.reviewError; color: Design.danger }
          Label { width: parent.width; visible: root.selected && !root.submitted && !Model.currentPierSource(root.connection,root.selected.source,root.service.now); text: "The source changed or is unavailable. Close this review and refresh discovery."; color: Design.warning }
          Label { width: parent.width; visible: !!root.service && !!root.service.storageOutcome && root.service.storageOutcome.path===candidate.modelData.path; text: visible ? root.service.storageOutcome.message : ""; secondary: true }
          ActionButton { objectName: "enroll-pier-submit"; text: root.submitted ? "Request submitted" : candidate.modelData.moved === true ? "Reconnect location" : "Enroll stopped pier"; primary: true; enabled: candidate.needsReview && root.selected && !root.submitted && root.confirmed && !root.service.busy && Model.validId(root.draftName.trim()) && root.draftShip.trim()!=="" && Model.currentPierSource(root.connection,root.selected.source,root.service.now); onClicked: root.submitEnrollment() }
          ActionButton { text: "Close review"; quiet: true; onClicked: root.selected=null }
        }
      }
    }
  }
  Label { visible: root.catalogue && root.catalogue.candidates.length===0; text: "No initialized piers found in the discovery tree."; secondary: true }
  Label { text: "Registered locations"; font.bold: true }
  Repeater {
    model: Model.instances(root.connection)
    Column {
      id: registered
      required property var modelData
      width: root.width; spacing: Style.space(6)
      Label { width: parent.width; text: Model.instanceTitle(registered.modelData)+" · "+registered.modelData.id+"\n"+Model.absolutePierPath(root.connection,registered.modelData.pier_path || ""); caption: true }
      Label { width: parent.width; visible: !!registered.modelData.external_import && registered.modelData.external_import.stage!=="complete"; text: visible ? "Import phase: "+registered.modelData.external_import.stage+" · "+(registered.modelData.external_import.copied_bytes || 0)+" bytes copied\nSource: "+registered.modelData.external_import.source+"\nHeld source: "+(registered.modelData.external_import.holding_path || "not moved")+"\nInspect the activity receipt before continuing an interrupted import." : ""; secondary: true; caption: true }
      ActionButton { property var ship: registered.modelData; text: "Move…"; compact: true; enabled: root.connection && root.connection.access_role==="operator" && !root.service.busy && !Model.handoffFenced(ship) && !ship.delete_pending && ship.desired_state==="stopped" && !ship.recovery_required && !ship.active_operation_id; onClicked: {root.moving=JSON.parse(JSON.stringify(ship));root.selected=null;root.submitted=false;destination.text=ship.pier_path || ""} }
      DeletePierPane { width: parent.width; service: root.service; connection: root.connection; instance: registered.modelData }
    }
  }
  Column {
    width: parent.width;spacing: Style.space(8);visible: !!root.moving
    StorageLocationSelector { id: moveLocation; width: parent.width; connection: root.connection; enabled: !root.submitted }
    Field { id: destination; width: parent.width; placeholderText: "Destination path below the selected location"; enabled: !root.submitted }
    ActionButton { text: root.submitted ? "Move submitted" : "Move stopped pier"; enabled: root.moving && Model.storagePath(moveLocation.rootID,destination.text)!==root.moving.pier_path && !root.submitted && !root.service.busy && Model.validPierPath(destination.text); onClicked: if(root.service.movePier(root.connection,root.moving,Model.storagePath(moveLocation.rootID,destination.text)))root.submitted=true }
    Label { width: parent.width; text: "Moves are same-filesystem, no-overwrite renames. Keep external moves stopped, then use discovery to reconnect the same pier."; secondary: true;caption:true }
  }
  Label { width: parent.width; visible: root.service && !!root.service.lastError; text: root.service ? root.service.lastError : ""; color: Design.danger }
  Label { width: parent.width; visible: root.submitted; text: root.service.storageOutcome ? root.service.storageOutcome.message : "Inspect the fleet/activity for completion. Requests are never automatically repeated."; secondary: true;caption:true }
}
