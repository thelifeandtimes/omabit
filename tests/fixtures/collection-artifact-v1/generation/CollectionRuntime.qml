import QtQuick
import Quickshell.Io
import "." as Runtime
Runtime.Service {
  id: collectionRoot
  readonly property string collectionArtifact: "8637fcd71ade6ad9e043c89d566fec30fbabf84eccc6b6f5df613decf5b02620"
  readonly property string collectionProfile: "qualification-stable"
  readonly property string collectionPlugin: "thelifeandtimes.urbit-theme.qualification-stable"
  readonly property string collectionInstance: {
    var value = ""
    for (var i = 0; i < 4; i++) value += ("00000000" + Math.floor(Math.random()*4294967296).toString(16)).slice(-8)
    return value
  }
  function collectionObservation() {
    return {format:1, plugin_id:collectionPlugin, profile:collectionProfile,
      artifact_sha256:collectionArtifact, instance:collectionInstance,
      source:Qt.resolvedUrl("CollectionRuntime.qml").toString()}
  }
  IpcHandler {
    target: "thelifeandtimes.urbit-theme.qualification-stable" + ".collection"
    function identity(): string { return JSON.stringify(collectionRoot.collectionObservation()) }
    function drain(): string { return String(collectionRoot.stopTransport()) }
    function transport(): string { return JSON.stringify(collectionRoot.transportState) }
  }
}
