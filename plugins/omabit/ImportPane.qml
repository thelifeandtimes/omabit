import QtQuick
import QtQuick.Controls as Controls
import QtQuick.Layouts
import qs.Commons
import "Model.js" as Model

Column {
  id: root
  required property var service
  property var connection: null
  property var target: null
  readonly property string selection: connection ? connection.runner_id + "/" + connection.name : ""
  readonly property var catalogue: service && service.importCatalogue && connection && service.importCatalogue.runner === connection.runner_id && service.importCatalogue.connection === connection.name ? service.importCatalogue : null
  readonly property var sources: catalogue ? catalogue.sources.filter(function(s) { return !s.claimed_by }) : []
  spacing: Style.space(10)
  onSelectionChanged: { target = null; consent.checked = false }
  onVisibleChanged: if (!visible) { target = null; consent.checked = false }
  onCatalogueChanged: { target = null; consent.checked = false }
  Label { width: parent.width; text: "Import a qualified development pier"; font.bold: true }
  Label { width: parent.width; text: "This first import stage uses stopped fake piers verified by the runner's copy/resume qualification. External paths, uploads and real identities are not accepted yet."; secondary: true; caption: true }
  ActionButton {
    text: "Load runner-owned sources"; enabled: root.connection && root.service.observing && !root.service.busy && Model.importAvailable(root.service.connections,{connection:root.connection.name,runner:root.connection.runner_id},root.service.now)
    onClicked: root.service.loadImportSources(root.connection)
  }
  Label { width: parent.width; visible: !root.connection || !Model.hasFeature(root.connection,"import-qualified-fake"); text: "Requires the upgraded runner's import-qualified-fake capability."; color: Design.warning; caption: true }
  Label { width: parent.width; visible: !!root.catalogue && root.sources.length === 0; text: "No unclaimed completed sources. Run the disposable import qualification on this runner first."; secondary: true }
  Column {
    width: parent.width; spacing: Style.space(8); visible: root.sources.length > 0
    SelectField { id: source; width: parent.width; model: root.sources; textRole: "run_id"; onActivated: { root.target = null; consent.checked = false } }
    Field { id: instance; width: parent.width; placeholderText: "New instance name, e.g. import-check"; onTextEdited: root.target = null }
    Field { id: port; width: parent.width; placeholderText: "Unused HTTP port, e.g. 18086"; validator: IntValidator { bottom:1024; top:65535 } onTextEdited: root.target = null }
    ActionButton { text: "Review import"; enabled: source.currentIndex >= 0 && !root.service.busy; onClicked: {
      consent.checked = false
      try { root.target = Model.importTarget(root.catalogue,root.sources[source.currentIndex],instance.text.trim(),port.text,root.service.now) }
      catch(e) { root.service.lastError = e.message }
    } }
  }
  Card {
    width: parent.width; visible: !!root.target; implicitHeight: review.implicitHeight + Style.space(24)
    Column {
      id: review
      anchors.left: parent.left; anchors.right: parent.right; anchors.top: parent.top; anchors.margins: Style.space(12); spacing: Style.space(10)
      Label { width: parent.width; text: root.target ? root.target.connection + " / " + root.target.instance + " · HTTP " + root.target.port : ""; font.bold: true }
      Label { width: parent.width; text: root.target ? "Source " + root.target.source.run_id + "\n" + root.target.source.files + " files · " + Model.bytesText(root.target.source.bytes) + "\nDigest " + root.target.source.content_digest : ""; secondary: true; caption: true }
      Label { width: parent.width; text: "Copies into fresh runner-owned storage and reserves this source once. The imported ship remains stopped; Start is a separate action."; secondary: true }
      CheckField { id: consent; width: parent.width; enabled: root.service.canImport(root.target); text: "I confirm this selected source is stopped." }
      ActionButton { text: "Import stopped fake"; primary: true; enabled: consent.checked && root.service.canImport(root.target); onClicked: if (root.service.importFake(root.target,consent.checked)) { root.target = null; consent.checked = false } }
    }
  }
}
