import QtQuick
import Quickshell.Io

// Inert until an external coordinator's exact shell/member permit is claimed.
// A generated immutable entry point supplies these pins. No application QML is
// imported statically: helpers and native IPC must not start before admission.
Item {
  id: root
  required property string pluginId
  required property string profile
  required property string artifact
  required property string role
  required property url runtimeSource
  required property url guardSource
  property var shell: null
  property var manifest: null
  property var settings: ({})
  property var collectionRegistry: null
  readonly property bool opened: payload && "opened" in payload ? payload.opened : false
  readonly property int collectionHostAPI: 1
  readonly property var payload: implementation.item
  readonly property var transportState: payload ? payload.transportState : ({closed:false,reason:phase})
  property string phase: "waiting"
  property string error: ""
  property bool attempted: false
  property bool drainRequested: false
  property bool admissionEnding: false
  property bool admissionEnded: false
  property var expected: null
  property var grant: null
  property var frozenManifest: null
  readonly property string instance: {
    var value=""
    for(var i=0;i<4;i++)value+=("00000000"+Math.floor(Math.random()*4294967296).toString(16)).slice(-8)
    return value
  }

  function fence(reason) {
    phase="fenced";error=reason
    if(payload)drainPayload()
  }
  function localPath(url) {
    var value=String(url)
    if(!value.startsWith("file:///"))throw new Error("collection runtime must be local")
    return decodeURIComponent(value.slice(7))
  }
  function activationValid(activation) {return true}
  function grantValid(value) {return value.format===1}
  function start() {
    if(attempted || !manifest || !shell)return
    var activation=manifest.collectionActivation
    if(!activation)return
    if(!root.activationValid(activation) || manifest.id!==pluginId || typeof activation.root!=="string" || !activation.root.startsWith("/")
       || activation.root.length>4096 || /[\x00-\x1f]/.test(activation.root)
       || !/^[0-9a-f]{32}$/.test(activation.permit || "") || !/^[0-9a-f]{32}$/.test(activation.epoch || "")
       || !/^[0-9a-f]{64}$/.test(activation.collection || "")) {
      attempted=true;fence("Invalid collection activation metadata");return
    }
    attempted=true;phase="starting"
    frozenManifest=JSON.parse(JSON.stringify(manifest))
    expected={permit:activation.permit,epoch:activation.epoch,collection:activation.collection,
      plugin_id:pluginId,profile:profile,artifact_sha256:artifact}
    try {guard.command=["python3","-B",localPath(guardSource),"--root",activation.root];guard.running=true}
    catch(e){fence("Admission helper could not be prepared")}
  }
  function sameGrant(value) {
    if(!value || !root.grantValid(value) || value.instance!==instance)return false
    for(var key in expected)if(value[key]!==expected[key])return false
    return !grant || JSON.stringify(value.owner)===JSON.stringify(grant.owner)
  }
  function onAdmission(line) {
    if(phase==="fenced")return
    var response
    try {if(line.length>16384)throw new Error("oversize");response=JSON.parse(line)}
    catch(e){fence("Admission response could not be read");return}
    var value=response.admission
    if(response.ok!==true || !sameGrant(value) || value.admitted!==true || ["open","closing"].indexOf(value.state)<0) {
      fence("Collection admission is unavailable");return
    }
    grant=value
    if(value.state==="closing") {
      phase="draining";if(payload)drainPayload();return
    }
    if(phase!=="starting")return
    phase="loading"
    var props={manifest:frozenManifest,shell:root.shell}
    if(role!=="dms")props.settings=settings
    if(role==="omabit")props.uiAdmissionOpen=Qt.binding(function(){return root.phase==="loaded"})
    implementation.setSource(runtimeSource,props)
    implementation.active=true
  }
  function loaded() {
    if(!payload || typeof payload.collectionObservation!=="function") {fence("Loaded runtime lacks collection identity");return}
    var value=payload.collectionObservation()
    if(value.plugin_id!==pluginId || value.profile!==profile || value.artifact_sha256!==artifact
       || value.source!==String(runtimeSource)) {fence("Loaded runtime differs from the admitted artifact");return}
    if(phase==="loading" && grant && grant.state==="open" && guard.running)phase="loaded"
    else drainPayload()
  }
  function drainPayload() {
    if(!payload)return "not-loaded"
    if(typeof payload.stopTransport!=="function") {phase="fenced";error="Runtime lacks transport drain";return "unsupported"}
    var result=String(payload.stopTransport())
    if(result!=="busy")drainRequested=true
    return result
  }
  function open(value) {if(phase==="loaded" && payload && typeof payload.open==="function")payload.open(value)}
  function close() {if(payload && typeof payload.close==="function")payload.close()}
  function drain() {
    if(phase!=="fenced")phase="draining"
    return drainPayload()
  }
  function releaseAdmission() {
    if(admissionEnding)return admissionEnded ? "ended" : "ending"
    if(phase!=="draining" || !grant || grant.state!=="closing"
       || !drainRequested || !payload || transportState.closed!==true)return "busy"
    // Keep the host alive while EOF lets the exact guard durably end its claim.
    // This is not a transport or descendant-closure receipt for the coordinator.
    admissionEnding=true
    guard.stdinEnabled=false
    return "ending"
  }
  function diagnostic() {
    return {format:1,plugin_id:pluginId,profile:profile,artifact_sha256:artifact,instance:instance,
      phase:phase,error:error,expected:expected,grant:grant,
      loaded:payload && typeof payload.collectionObservation==="function" ? payload.collectionObservation() : null,
      transport:transportState,admissionEnding:admissionEnding,admissionEnded:admissionEnded}
  }
  onPhaseChanged: if(phase!=="loaded")close()
  onManifestChanged: Qt.callLater(start)
  onShellChanged: {if(payload && "shell" in payload)payload.shell=shell;Qt.callLater(start)}
  onSettingsChanged: if(phase==="loaded" && payload && "settings" in payload)payload.settings=settings
  Component.onCompleted: {
    if(collectionRegistry && !collectionRegistry.claim(root)) {attempted=true;fence("Generation already has a host")}
    else Qt.callLater(start)
  }
  Component.onDestruction: if(collectionRegistry)collectionRegistry.release(root)

  Loader {id:implementation;active:false;onLoaded:root.loaded();onStatusChanged:if(status===Loader.Error)root.fence("Admitted runtime failed to load")}
  Timer {interval:100;repeat:true;running:(root.phase==="draining" || root.phase==="fenced") && !!root.payload && !root.drainRequested;onTriggered:root.drainPayload()}
  Process {
    id:guard
    stdinEnabled:true
    onStarted:write(JSON.stringify({expected:root.expected,role:root.role,instance:root.instance})+"\n")
    stdout:SplitParser {onRead:function(line){root.onAdmission(line)}}
    onExited:function(exitCode,exitStatus) {
      if(root.admissionEnding && exitCode===0 && exitStatus===0)root.admissionEnded=true
      else root.fence("Admission owner exited; coordinator recovery is required")
    }
  }
  IpcHandler {
    target:root.pluginId+".collection-host"
    function status():string{return JSON.stringify(root.diagnostic())}
    function drain():string{return root.drain()}
    function releaseAdmission():string{return root.releaseAdmission()}
  }
}
