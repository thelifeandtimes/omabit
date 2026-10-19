import QtQuick
import qs.Commons
import "Model.js" as Model

// Diagnostic rows within the existing setup/runtime cards. They never execute
// guidance or reinterpret version support as pier recovery.
Column {
  id: root
  property var report: null
  property var envelope: null
  property string problem: ""
  property double now: Date.now()
  property bool details: false
  spacing: Style.space(7)
  Label {
    objectName: "host-compatibility-summary"
    width: parent.width
    text: (root.problem ? root.problem+(root.report ? "\n" : "") : "")+(root.report || !root.problem ? Model.compatibilitySummary(root.report) : "")
    caption: true
    color: root.problem || root.report && (root.report.restart_pending || !root.report.can_mutate) ? Design.warning : Design.muted
  }
  Label {
    objectName: "host-compatibility-facts"
    width: parent.width; caption: true; secondary: true
    visible: !!root.report
    text: root.details ? Model.compatibilityFacts(root.report,root.now,root.envelope) : root.report ? "Installed engine: "+(root.report.installed_engine_version || "unknown")+" · running engine: "+(root.report.engine && root.report.engine.version || "not observed")+"\n"+Model.compatibilityEvidence(root.report,root.now,root.envelope) : ""
  }
  Label {
    objectName: "host-compatibility-checks"
    width: parent.width; caption: true; secondary: true
    text: Model.compatibilityChecks(root.report,root.details); visible: text !== ""
  }
  Label {
    objectName: "host-compatibility-guidance"
    width: parent.width; caption: true; secondary: true
    text: Model.compatibilityGuidance(root.report); visible: text !== ""
  }
}
