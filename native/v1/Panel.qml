import QtQuick
import Quickshell.Io
import qs.Commons

// The shell may rebuild this adapter on every rescan. The retained bootstrap
// owns the selected generation; the widget never independently selects a release.
Item {
  id: root
  property QtObject bar: null
  property var settings: ({})
  property bool manageIpc: true
  property string moduleName: "thelifeandtimes.omabit"
  property var bootstrap: bar && bar.shell ? bar.shell.serviceFor(moduleName) : null
  readonly property var service: bootstrap ? bootstrap.payload : null
  readonly property var implementation: panel.item
  readonly property bool opened: panel.item ? panel.item.opened : false
  implicitWidth: panel.item ? panel.item.implicitWidth : Style.space(40)
  implicitHeight: panel.item ? panel.item.implicitHeight : Style.space(30)
  function open() { if(panel.item)panel.item.open() }
  function close() { if(panel.item)panel.item.close() }
  function toggle() { if(panel.item)panel.item.toggle() }
  function closeForPopoutSwitch() { if(panel.item)panel.item.closeForPopoutSwitch() }
  Connections {
    target:root.bootstrap
    function onPhaseChanged() {if(root.bootstrap.phase==="starting-activation")root.close()}
  }
  function attach() {
    panel.active=false
    if (!service || !bootstrap.uiRoot) return
    panel.setSource(bootstrap.uiRoot+"/Panel.qml",{service:service,bar:bar,settings:pairedSettings(),manageIpc:manageIpc})
    panel.active=true
  }
  function pairedSettings() { var value=Object.assign({},settings);delete value.collectorPath;delete value.omabitPath;return value }
  function configureBootstrap() { if(bootstrap)bootstrap.settings=pairedSettings() }
  onBootstrapChanged: configureBootstrap()
  onServiceChanged: Qt.callLater(attach)
  onBarChanged: if(panel.item)panel.item.bar=bar
  onSettingsChanged: { configureBootstrap();if(panel.item)panel.item.settings=pairedSettings() }
  onManageIpcChanged: if(panel.item)panel.item.manageIpc=manageIpc
  Component.onCompleted: { configureBootstrap();Qt.callLater(attach) }
  Loader { id:panel; anchors.fill:parent; active:false }
  Text {
    anchors.centerIn:parent
    visible:!panel.item
    text:"~+"
    color:Color.foreground
    font.family:Style.font.family
    font.pixelSize:Style.space(18)
  }
  MouseArea {
    anchors.fill:parent; enabled:!panel.item
    onClicked: {
      if (root.bootstrap) {
        var message=root.bootstrap.error || "Preparing the selected release…"
        if(root.bootstrap.error) {root.bootstrap.retry();message="Retrying desktop preparation…\n"+message}
        notice.command=["notify-send","--app-name=Omabit","Omabit desktop",message]
      }
      else notice.command=["notify-send","--app-name=Omabit","Omabit desktop","Waiting for the bootstrap service…"]
      notice.running=true
    }
  }
  Process { id:notice }
}
