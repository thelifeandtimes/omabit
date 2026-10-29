import QtQuick
import Quickshell.Io
import "." as Runtime
Runtime.Service {
  id: collectionRoot
  readonly property string collectionArtifact: "9f8b1abdf003c19c5daf26945a6cc6ff4fdba6bd5c86df06a9aeff1f5006a06d"
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
