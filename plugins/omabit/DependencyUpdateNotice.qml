import QtQuick
import QtQuick.Controls as Controls
import QtQuick.Layouts
import qs.Commons

Card {
  id: root
  property var service: null
  property bool active: false
  property bool inspected: false
  readonly property var updater: service ? service.serviceUpgrade : null
  readonly property bool eligible: !!service && service.localSetupAllowed && !!service.localSetupStatus
    && (service.localSetupStatus.host_present || service.localSetupStatus.ready)
  signal openRequested()
  visible: eligible && !!updater && (updater.busy || !!updater.error || !updater.status || updater.status.available)
  implicitHeight: body.implicitHeight + Style.space(24)
  function inspect() {
    if (active && eligible && !inspected && updater && updater.check()) inspected=true
  }
  onActiveChanged: { if (active) { inspected=false;inspect() } }
  onEligibleChanged: if (eligible) inspect()
  Connections {
    target:root.service
    function onCheckingSetupChanged() { if (!root.service.checkingSetup) root.inspect() }
  }
  Connections {
    target:root.updater
    function onCanStartChanged() { if (root.updater.canStart) root.inspect() }
  }
  Column {
    id: body
    anchors.left:parent.left;anchors.right:parent.right;anchors.top:parent.top
    anchors.margins:Style.space(12);spacing:Style.space(8)
    Label {
      width:parent.width;secondary:true
      text:root.updater && root.updater.error ? "A dependency update needs attention. Open it to check or retry."
        : root.updater && root.updater.busy ? root.updater.message
        : root.updater && root.updater.status && root.updater.status.available
          ? "Updated Omabit tools are ready. Finish updating this computer’s runner."
          : "Check that this computer’s runner matches your Omabit plugin."
    }
    RowLayout {
      width:parent.width
      Controls.BusyIndicator { visible:!!root.updater && root.updater.busy;running:visible;implicitWidth:Style.space(24);implicitHeight:Style.space(24) }
      ActionButton {
        objectName:"menubar-update-dependencies"
        Layout.fillWidth:true;text:"Update Dependencies";primary:true
        enabled:!!root.updater && root.updater.canStart
        onClicked: {
          root.openRequested()
          if (root.updater.status && root.updater.status.available) root.updater.upgrade()
          else root.updater.check()
        }
      }
    }
    Label { width:parent.width;caption:true;secondary:true;text:"Preserves your ships. Updating the local service asks for OS authorization." }
  }
}
