import QtQuick

Column {
  id: root
  property var service: null
  property var report: null
  visible: !!service && !!service.bootstrap
  spacing: 6
  Label { width:parent.width;caption:true;secondary:true;text:root.service ? "Distribution channel: "+root.service.distributionChannel+" · "+root.service.releaseLabel : "" }
  Label { width:parent.width;caption:true;secondary:true;visible:!!root.service && root.service.distributionChannel!=="stable";text:"Update this channel with scripts/omabit-channel.py update from the installed checkout. Stock plugin update follows master." }
  function refresh() { report=service && service.bootstrap ? service.bootstrap.diagnostic() : null }
  onServiceChanged: refresh()
  Component.onCompleted: refresh()
  Timer { interval:1000; repeat:true; running:root.visible; onTriggered:root.refresh() }
  Label {
    width:parent.width; caption:true; secondary:true
    text:root.report ? "Desktop profile: "+root.report.profile+" · loaded: "+(root.report.loaded_version || "starting")+" · requested: "+root.report.requested_version+" · "+root.report.phase : ""
  }
  Label {
    width:parent.width; caption:true; secondary:true
    text:root.report ? (root.report.development ? "Development build: "+root.report.development.label+"\n" : "")+"Release: "+(root.report.loaded_release || "pending")+"\nSource: "+(root.report.source_commit || "pending")+"\nCollector: "+(root.report.observing ? "observing" : "waiting/reconnecting")+" · "+root.report.runners+" runners · "+root.report.ships+" ships" : ""
  }
  Label { width:parent.width; color:Design.danger; visible:text!==""; text:root.report ? root.report.error : "" }
  ActionButton { visible:root.report && root.report.error!==""; text:"Retry desktop activation"; onClicked:root.service.bootstrap.retry() }
}
