import QtQuick
import QtQuick.Window
import QtQuick.Controls as Controls
import QtQuick.Layouts
import Quickshell
import Quickshell.Io
import "native/v1" as Bootstrap
ShellRoot {
 id: root
 readonly property bool checking: Quickshell.env("OMABIT_REVIEW_CHECK")==="1"
 Bootstrap.Service {id: host}
 Window {
  visible: !root.checking; width: 720; height: 96
  title: "Omabit 0.3.5-rc.1 · isolated review"; color: "#18201c"
  onClosing: Qt.quit()
  RowLayout {
   anchors.fill:parent; anchors.margins:16; spacing:12
   Bootstrap.Panel {id:widget; bootstrap:host; manageIpc:false; moduleName:"omabit.review035"; bar:null; Layout.preferredWidth:40;Layout.preferredHeight:40}
   Column {
    Layout.fillWidth:true
    Text {text:"Omabit 0.3.5-rc.1 · REVIEW";color:"#e7eee8";font.bold:true;font.pixelSize:17}
    Text {text:host.phase==="ready" ? "Private review profile · "+host.diagnostic().runners+" runners" : host.error || host.phase;color:"#b6c8bc";font.pixelSize:12}
   }
   Controls.Button {text:"Monitor";enabled:!!host.payload;onClicked:host.payload.openMonitor(null,null,"overview")}
   Controls.Button {text:"Quit preview";onClicked:Qt.quit()}
  }
 }
 IpcHandler {target:"review035";function status():string{return JSON.stringify(host.diagnostic())}}
}
