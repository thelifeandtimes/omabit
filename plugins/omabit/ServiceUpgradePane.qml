import QtQuick
import QtQuick.Controls as Controls
import QtQuick.Layouts
import qs.Commons

Card {
  id: root
  property var updater: null
  property bool detailsOpen: false
  width: parent.width
  implicitHeight: contents.implicitHeight + Style.space(32)
  Column {
    id: contents
    anchors.left:parent.left;anchors.right:parent.right;anchors.top:parent.top;anchors.margins:Style.space(16)
    spacing:Style.space(10)
    CardTitle { width:parent.width;text:"Service updates" }
    Label { width:parent.width;secondary:true;text:"The plugin updates Omabit and its collector together. Update the local runner here to finish a desktop upgrade." }
    Label { width:parent.width;caption:true;secondary:true;text:"This preserves your ships and access keys. The host step asks for OS authorization. Docker packages and ship runtimes keep their current versions." }
    Label { objectName:"service-upgrade-builds";width:parent.width;visible:root.detailsOpen && !!root.updater && !!root.updater.status;caption:true;secondary:true;text:visible ? "Installed build: " + root.updater.status.installed_build + " · Selected update: " + root.updater.status.target_build : "" }
    RowLayout {
      width:parent.width
      Controls.BusyIndicator { running:!!root.updater && root.updater.busy;visible:running;implicitWidth:Style.space(24);implicitHeight:Style.space(24) }
      Label { objectName:"service-upgrade-progress";Layout.fillWidth:true;text:root.updater ? root.updater.message : "" }
    }
    Label { objectName:"service-upgrade-error";width:parent.width;visible:!!root.updater && !!root.updater.error;text:root.updater ? root.updater.error : "";color:Design.danger }
    Flow {
      width:parent.width;spacing:Style.space(8)
      ActionButton { objectName:"service-upgrade-check";text:"Check services";enabled:!!root.updater && root.updater.canStart;onClicked:root.updater.check() }
      ActionButton { objectName:"service-upgrade-details";text:root.detailsOpen ? "Hide details" : "Technical details";quiet:true;visible:!!root.updater && !!root.updater.status;onClicked:root.detailsOpen=!root.detailsOpen }
      ActionButton { objectName:"service-upgrade-apply";text:root.updater && root.updater.status && root.updater.status.pending ? "Retry service update" : "Update services";primary:true;visible:!!root.updater && !!root.updater.status && root.updater.status.available;enabled:!!root.updater && root.updater.canStart;onClicked:root.updater.upgrade() }
    }
  }
}
