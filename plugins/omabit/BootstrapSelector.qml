import QtQuick
import qs.Commons
import "Model.js" as Model

Column {
  id: root
  objectName: "bootstrap-selector"
  property var service
  readonly property string kind: choice.currentIndex === 1 ? "recommended" : "bare"
  readonly property string source: kind === "recommended" && service ? String(service.settings.recommendedPillUrl || service.settings.defaultBootPill || "").trim() : ""
  readonly property string problem: Model.bootstrapProblem(kind, source)
  signal edited()
  spacing: Style.space(8)
  function reset() { choice.currentIndex=0 }
  Label { text: "Boot pill"; caption: true; secondary: true }
  SelectField {
    id: choice
    objectName: "bootstrap-choice"
    width: parent.width
    model: ["bare bootstrap", "Recommended Apps"]
    onActivated: root.edited()
  }
  Label {
    width: parent.width; caption: true; secondary: true
    text: root.kind === "bare" ? "Use Vere's default bootstrap.urbit.org pill." : root.problem || "Download the configured Recommended Apps release pill for this ship's first boot."
    color: root.problem ? Design.warning : Design.muted
  }
}
