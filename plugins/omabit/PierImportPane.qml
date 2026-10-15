import QtQuick
import qs.Commons
import "Model.js" as Model

Column {
  id: root
  objectName: "external-pier-import"
  property var service
  property var connection
  property bool submitted: false
  readonly property string selection: connection ? connection.runner_id+"/"+connection.name : ""
  readonly property var review: service ? service.externalImportPreview : null
  readonly property bool current: !!service && Model.externalReviewCurrent(connection,review,source.text.trim(),service.now)
  readonly property string targetPath: Model.storagePath(location.rootID,destination.text.trim())
  onSelectionChanged: {submitted=false;consent.checked=false}
  spacing: Style.space(9)
  Label { text: "Import an existing pier"; font.pixelSize: Style.space(21); font.bold: true }
  Label { width: parent.width; text: "Enter the existing pier directory on this runner's host (the directory containing .urb). Review it, then approve moving it into the chosen storage location. The imported ship stays stopped until you choose Start."; secondary: true }
  Field { id: source; objectName: "import-pier-source"; width: parent.width; placeholderText: "/absolute/host/path/to/existing-pier"; enabled: !root.service.busy && !root.submitted; onTextEdited: {consent.checked=false;root.service.externalImportPreview=null} }
  ActionButton { objectName: "import-pier-review"; text: "Inspect source directory"; enabled: root.connection && Model.hasFeature(root.connection,"import-pier") && !root.service.busy && !root.submitted; onClicked: root.service.previewExternalPier(root.connection,source.text.trim()) }
  Label { width: parent.width; visible: root.current; text: root.current ? "Reviewed "+root.review.source.files+" files · "+root.review.source.bytes+" bytes" : ""; secondary: true }
  Label { width: parent.width; text: "The source must be stopped and accessible to urbit-runner, including permission to move it out of its parent. Permission errors identify host preparation needed; the desktop never interprets this as a local file path."; secondary: true; caption: true }
  Field { id: name; objectName: "import-pier-name"; width: parent.width; placeholderText: "New instance name"; enabled: !root.submitted && !root.service.busy; onTextEdited: consent.checked=false }
  Field { id: ship; objectName: "import-pier-ship"; width: parent.width; placeholderText: "Expected ~ship identity"; enabled: !root.submitted && !root.service.busy; onTextEdited: consent.checked=false }
  StorageLocationSelector { id: location; width: parent.width; connection: root.connection; enabled: !root.submitted && !root.service.busy; onEdited: consent.checked=false }
  Field { id: destination; objectName: "import-pier-destination"; width: parent.width; placeholderText: "e.g. fakeships/testing/pier or livenet/planets/sampel-palnet/pier"; enabled: !root.submitted && !root.service.busy; onTextEdited: consent.checked=false }
  Label { width: parent.width; text: "Destination: "+Model.absolutePierPath(root.connection,root.targetPath)+"\nIts exclusive parent is mounted at /urbit; the pier is /urbit/pier."; secondary: true; caption: true }
  Field { id: port; objectName: "import-pier-port"; width: parent.width; placeholderText: "Unused host HTTP port"; enabled: !root.submitted && !root.service.busy; onTextEdited: consent.checked=false }
  CheckField { id: consent; objectName: "import-pier-consent"; width: parent.width; text: "The source is stopped. Move this pier out of its old location and enroll it here."; enabled: root.current && !root.submitted && !root.service.busy }
  ActionButton { objectName: "import-pier-submit"; primary: true; text: root.submitted ? "Import submitted — inspect activity" : "Move and import pier"; enabled: root.current && !root.submitted && !root.service.busy && consent.checked && Model.validId(name.text.trim()) && Model.validPierPath(root.targetPath) && ship.text.trim()!=="" && port.text.trim()!=="" && Model.optionalPort(port.text); onClicked: if(root.service.importExternalPier(root.connection,root.review,name.text.trim(),ship.text.trim(),root.targetPath,port.text,consent.checked))root.submitted=true }
  ActionButton { text: "Start a new import review"; quiet: true; visible: root.submitted; enabled: !root.service.busy; onClicked: {root.submitted=false;root.service.externalImportPreview=null;consent.checked=false} }
}
