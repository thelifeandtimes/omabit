import QtQuick
import Quickshell.Io
import "Marker.js" as Marker
Item {
 id: root
 property var manifest: null
 property var shell: null
 property string profileName: "default"
 property var profileEnvironment: ({})
 property url toolsDirectory
 property bool localSetupAllowed: true
 readonly property int fixtureVersion: Marker.version
 property string fixtureHelper: ""
 property var transportState: ({closed:false})
 function stopTransport(){transportState={closed:true};return "requested"}
 Process {
  command:["python3","-B",Qt.resolvedUrl("helper.py").toString().replace(/^file:\/\//, "")]
  running:true
  stdout: StdioCollector {onStreamFinished:root.fixtureHelper=text.trim()}
 }
}
