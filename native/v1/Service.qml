import QtQuick
import Quickshell
import Quickshell.Io

// Stable bootstrap API v1. Never statically import the mutable application tree.
Item {
  id: root
  property var shell: null
  property var manifest: null
  property var settings: ({})
  property string controlPath: decodeURIComponent(Qt.resolvedUrl("../bootstrap/control.py").toString().replace(/^file:\/\//, ""))
  property var controlPrefix: ["python3", "-B", controlPath]
  property string session: ""
  property string phase: "starting"
  property string error: ""
  property var plan: null
  property var candidate: null
  property var active: null
  property var previous: null
  property var view: null
  property bool published: false
  property bool restoring: false
  property string acceptedRequest: ""
  property string failedRequest: ""
  property string job: ""
  property string output: ""
  property string errors: ""
  readonly property var payload: published ? application.item : null
  readonly property string uiRoot: published && active ? active.ui : ""
  readonly property bool safeToSwitch: !application.item || (!application.item.busy && !application.item.copyingCode && !application.item.checkingSetup && !application.item.pendingFirstBoot && (!("reloadSafe" in application.item) || application.item.reloadSafe))

  function diagnostic() {
    var app = application.item
    return {bootstrap_api:1, phase:phase, error:error, profile:active ? active.profile : plan ? plan.profile : "default",requested_profile:plan ? plan.profile : "default",
      requested_version:plan ? plan.version : "", loaded_version:active && published ? active.version : "",
      requested_release:plan ? plan.release : "", loaded_release:active && published ? active.release : "",
      source_commit:active && published ? active.source_commit : "", development:active ? active.development : null, busy:!!app && app.busy,
      enrollments_loaded:!!app && app.registrationsLoaded, observing:!!app && app.observing,
      runners:app ? app.registrations.length : 0, ships:app ? app.connections.reduce(function(n,c){return n+(c.snapshot && c.snapshot.instances ? c.snapshot.instances.length : 0)},0) : 0}
  }
  function run(kind, args) {
    if (worker.running || !session) return false
    job=kind; output=""; errors=""
    worker.command=controlPrefix.concat(args); worker.running=true
    return true
  }
  function resolve() { if (session && !worker.running && ["starting","ready","error","preparing","waiting"].indexOf(phase)>=0) run("resolve",["resolve"]) }
  function retry() { if (!worker.running) run("retry",["retry"]) }
  function intent(kind, message) {
    return run(kind,[kind,"--session",session,"--request",candidate.request,"--message",message || ""])
  }
  function releaseView() {
    var app=application.item
    if (!app) return null
    var m=app.monitorWindow
    return {profile:active ? active.profile : "",visible:m.visible,page:m.page,selectedKey:m.selectedKey}
  }
  function freeze() {
    if (!application.item) return
    if (active.contract===1) application.item.activationEnabled=false
    if (application.item.monitorWindow.contentItem) application.item.monitorWindow.contentItem.enabled=false
  }
  function thaw() {
    if (!application.item) return
    if (active.contract===1) application.item.activationEnabled=true
    if (application.item.monitorWindow.contentItem) application.item.monitorWindow.contentItem.enabled=true
  }
  function load(record, fallback) {
    restoring=fallback
    published=false
    panelProbe.active=false
    application.active=false
    active=record
    phase=fallback ? "restoring" : "activating"
    // Loader deactivation destroys the previous service/IPC/process objects.
    // Release-specific URLs also separate every nested QML/JS cache entry.
    Qt.callLater(function() {
      var props={shell:root.shell,manifest:record.manifest,settings:root.settings,toolsDirectory:record.tools}
      if (record.contract===1) {
        props.profileName=record.profile
        props.profileEnvironment=record.environment
        props.localSetupAllowed=record.host_setup
        props.bootstrap=root
        props.activationEnabled=false
      }
      application.setSource(record.ui+"/Service.qml",props)
      application.active=true
      readinessTimeout.restart()
    })
  }
  function ready() {
    var app=application.item
    if (!app || !app.registrationsLoaded || panelProbe.status!==Loader.Ready) return false
    return app.registrations.length===0 || app.observing && app.connections.length>0
  }
  function expose() {
    thaw()
    panelProbe.active=false
    published=true
    phase="ready"
    readinessTimeout.stop()
    restoring=false
    if (plan && active.release===plan.release && active.profile===plan.profile) acceptedRequest=plan.request
    if (view && view.profile===active.profile) {
      var m=application.item.monitorWindow
      m.page=view.page; m.selectedKey=view.selectedKey
      if (view.visible) application.item.openMonitor(null,null,view.page)
    }
    view=null
  }
  function failed(message) {
    readinessTimeout.stop()
    error=String(message).slice(0,1024)
    if (restoring) { phase="error"; published=false; application.active=false; return }
    failedRequest=candidate ? candidate.request : ""
    phase="error"
    if (candidate && !worker.running) intent("fail",error)
  }
  function fallback() {
    if (previous) load(previous,true)
    else { published=false; application.active=false; phase="error" }
  }
  function finished(code) {
    var kind=job; job=""
    var result=null
    try { if (code!==0) throw new Error(errors || "bootstrap command failed"); result=JSON.parse(output) }
    catch (e) {
      error=String(e.message).slice(0,1024)
      if (kind==="begin" || kind==="commit") { failed(error); return }
      if (kind==="prepare") { failed(error); return }
      if (kind==="fail") { if(published){thaw();phase="ready"}else fallback();Qt.callLater(resolve);return }
      phase=published ? "ready" : "error"
      return
    }
    if (kind==="resolve") {
      plan=result
      if (result.request===acceptedRequest) {if(!result.blocked)error="";return}
      if (result.request===failedRequest) return
      if (!active && result.active) {run("restore",["prepare","--release",result.active,"--profile",result.profile]);return}
      if (result.blocked) {
        failedRequest=result.request; error=result.failure.message
        if (!published && result.active) run("restore",["prepare","--release",result.active,"--profile",result.profile])
        return
      }
      if (phase==="waiting" && candidate && candidate.request===result.request) return
      candidate=result
      error=""; phase="preparing"
      run("prepare",["prepare","--request",result.request])
    } else if (kind==="prepare") {
      candidate=result; phase="waiting"
    } else if (kind==="begin") {
      previous=active && published ? active : null
      view=releaseView()
      load(candidate,false)
    } else if (kind==="commit") {
      acceptedRequest=candidate.request; failedRequest=""; error=""
      expose()
    } else if (kind==="fail") {
      // Download errors occur before deactivation. Keep the current UI alive.
      if (published) {thaw();phase="ready"}
      else fallback()
    } else if (kind==="restore") {
      previous=result; load(result,true)
    } else if (kind==="retry") {
      failedRequest=""; acceptedRequest=""; phase=published ? "ready" : "starting"
      Qt.callLater(resolve)
    }
  }
  onManifestChanged: Qt.callLater(resolve)
  onSettingsChanged: if(application.item) {application.item.settings=settings;if("refresh" in application.item)application.item.refresh()}
  Component.onCompleted: lease.running=true
  Process {
    id: lease
    command: root.controlPrefix.concat(["lease"])
    stdinEnabled: true
    stdout: SplitParser { onRead: function(line) { try { root.session=JSON.parse(line).session; root.resolve() } catch(e) {root.error=String(e)} } }
    stderr: SplitParser { onRead: function(line) { root.error=line } }
    onExited: { root.session=""; root.published=false; application.active=false; root.phase="error" }
  }
  Process {
    id: worker
    stdout: SplitParser { splitMarker:""; onRead:function(data) { if(root.output.length+data.length<1048576)root.output+=data;else worker.signal(9) } }
    stderr: SplitParser { splitMarker:""; onRead:function(data) { if(root.errors.length+data.length<8192)root.errors+=data } }
    onExited:function(code,status) { root.finished(status===0 ? code : -1) }
  }
  Loader {
    id: application
    active: false
    asynchronous: true
    onLoaded: {
      if(root.active.contract===1 && (item.bootstrapAPI!==1 || item.applicationVersion!==root.active.version)) {root.failed("Loaded application version/API does not match the candidate bundle.");return}
      panelProbe.setSource(root.active.ui+"/Panel.qml",{service:item,bar:null,settings:root.settings,manageIpc:false})
      panelProbe.active=true
    }
    onStatusChanged: if (status===Loader.Error) root.failed("Candidate UI could not load; previous release retained.")
  }
  Loader {
    id:panelProbe; active:false; visible:false; asynchronous:true
    onStatusChanged: if(status===Loader.Error)root.failed("Candidate widget could not load; previous release retained.")
  }
  Timer {
    interval:250; running:true; repeat:true
    onTriggered: {
      if (root.phase==="waiting" && root.safeToSwitch && !worker.running) {root.freeze();root.phase="starting-activation";root.intent("begin","")}
      else if ((root.phase==="activating" || root.phase==="restoring") && root.ready() && !worker.running) {
        if (root.restoring) root.expose()
        else {root.phase="committing";root.intent("commit","")}
      }
    }
  }
  Timer { id: readinessTimeout; interval:30000; onTriggered:root.failed("Candidate did not finish enrollment/collector startup; previous release retained.") }
  Timer { interval:3000; running:true; repeat:true; onTriggered: {if (!lease.running) lease.running=true; else root.resolve()} }
  IpcHandler {
    target:"thelifeandtimes.omabit.bootstrap"
    function status(): string { return JSON.stringify(root.diagnostic()) }
    function retry(): void { root.retry() }
  }
}
