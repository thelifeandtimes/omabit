import QtQuick
import QtQuick.Layouts
import qs.Commons

Column {
  id: root
  property bool supported: false
  property bool stopped: false
  property bool initialBoot: false
  signal edited()
  property int memoryGiB: 4
  property int cpuCores: 2
  readonly property string problem: memoryGiB<1 || memoryGiB>16 || cpuCores<1 || cpuCores>16 ? "Choose RAM and CPU limits in 1–16." : ""
  spacing: Style.space(8)
  RowLayout {
    width: parent.width; spacing: Style.space(12)
    Column {
      Layout.fillWidth: true; Layout.preferredWidth: 1; spacing: Style.space(6)
      Label { text: "RAM limit"; caption: true; secondary: true }
      SelectField {
        objectName: "container-memory"; width: parent.width
        enabled: root.supported && root.stopped
        model: { var values=[]; for(var n=1;n<=16;n++) values.push(n+" GiB"+(n===4 ? " · default" : "")); return values }
        currentIndex: root.memoryGiB-1
        onActivated: {root.memoryGiB=currentIndex+1;root.edited()}
      }
    }
    Column {
      Layout.fillWidth: true; Layout.preferredWidth: 1; spacing: Style.space(6)
      Label { text: "CPU limit"; caption: true; secondary: true }
      SelectField {
        objectName: "container-cpus"; width: parent.width
        enabled: root.supported && root.stopped
        model: { var values=[]; for(var n=1;n<=16;n++) values.push(n+(n===1 ? " core" : " cores")+(n===2 ? " · default" : "")); return values }
        currentIndex: root.cpuCores-1
        onActivated: {root.cpuCores=currentIndex+1;root.edited()}
      }
    }
  }
  Label {
    width: parent.width; caption: true; secondary: true
    text: !root.supported ? "Update this runner to configure RAM and CPU limits. Existing limits are preserved."
      : root.initialBoot ? "Limits apply on first boot. They are ceilings, not reserved host capacity; CPU limits do not pin cores."
      : !root.stopped ? "Stop the ship to change resource limits, then reload these settings."
      : "Save the limits, then Start the ship. The existing pier is retained. Limits are ceilings, not reserved host capacity; CPU limits do not pin cores."
  }
}
