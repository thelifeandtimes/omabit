import QtQuick
import QtCore as Core
import Quickshell
import Quickshell.Io
import "Model.js" as Model

Item {
  id: root
  readonly property int bootstrapAPI: 1
  readonly property string applicationVersion: "0.1.0-rc.14"
  property var shell: null
  property var manifest: null
  property var settings: ({})
  property var bootstrap: null
  property string profileName: "default"
  property var profileEnvironment: ({})
  property bool localSetupAllowed: true
  property bool activationEnabled: true
  readonly property bool reloadSafe: !busy && !copyingCode && !checkingSetup && !themeSync.running && !pendingFirstBoot
  property int reconnectAttempts: 0
  property bool desktopViewLoaded: false
  property var connections: []
  property var ships: []
  property var registrations: []
  property bool registrationsLoaded: false
  readonly property bool firstRun: registrationsLoaded && registrations.length===0 && connections.length===0
  readonly property string machineUse: preferences.machineUse
  readonly property string recommendedPillUrl: Model.recommendedPillSource(settings,manifest)
  function setMachineUse(value) {
    if (["always-on","laptop"].indexOf(value)<0) return
    preferences.machineUse=value;preferences.sync()
  }
  property bool setupCheckRequested: false
  property var portSuggestion: null
  property var portTarget: null
  property string portError: ""
  readonly property bool checkingPort: currentAction === "suggest-port"
  readonly property bool actionProgressVisible: settingUp || currentAction !== "" && !checkingPort
  property var pendingFirstBoot: null
  property var bootTarget: null
  property string registrationOutput: ""
  property var memoryHistory: ({})
  property var cpuHistory: ({})
  property var importCatalogue: null
  property var pendingImportConnection: null
  property string codeNotice: ""
  property string toastText: ""
  function showToast(text) { toastText=text; toastExpiry.restart() }
  property string codeOutput: ""
  property string codeSubject: ""
  readonly property bool copyingCode: codeCopy.running
  readonly property alias monitorWindow: monitor
  readonly property var runnerPreferences: {
    try { var v = JSON.parse(preferences.runners); return v && !Array.isArray(v) && typeof v === "object" ? v : ({}) }
    catch (_) { return ({}) }
  }
  property string lastError: ""
  property string lastResult: ""
  property string requestIdentity: ""
  property var registrationOutcome: null
  property var storageOutcome: null
  property var importOutcomes: ({})
  property var externalImportPreview: null
  property var pendingExternalPreview: null
  property var importRecoveryReview: null
  property var hostMoveOutcome: null
  property var routineOutcome: null
  property var epochInspection: null
  property var inspections: ({})
  property double now: Date.now()
  property bool refreshing: false
  property bool refreshPending: false
  readonly property bool busy: !activationEnabled || action.running || currentAction !== "" || settingUp
  readonly property bool observing: feed.running && !feedError
  property string feedError: ""
  property string output: ""
  property string errors: ""
  property string currentAction: ""
  property var localSetupStatus: null
  property string localSetupStep: ""
  property string localSetupMessage: ""
  property string localSetupError: ""
  property string localStatusOutput: ""
  property bool localSetupPending: false
  property bool localSetupFinished: false
  readonly property bool settingUp: localSetup.running || localSetupPending
  readonly property bool checkingSetup: localStatus.running
  property var epochPreview: null
  property var previewTarget: null
  property var pendingPreviewTarget: null
  readonly property bool canConfirmPrune: observing && !busy && Model.canPrunePreview(connections, previewTarget, epochPreview, now)
  property bool outputFailed: false
  property int generation: 0
  property string shipOutput: ""
  property string pendingInput: ""
  property var shipStatus: ({})
  property var notificationQueue: []
  property var themeState: ({ships: {}})
  property var palette: null
  property string themeOutput: ""
  property string paletteOutput: ""
  property string themeSyncOutput: ""
  property url toolsDirectory: Qt.resolvedUrl("bin")
  readonly property string omabit: (profileName==="default" && settings.omabitPath) || Model.localPath(toolsDirectory) + "/omabit"
  readonly property bool canLogin: !busy && currentAction === ""
  readonly property string collector: (profileName==="default" && settings.collectorPath) || Model.localPath(toolsDirectory) + "/urbit-collector"

  function command(args) {
    var prefix = [collector, "--format", "json"]
    if (profileName==="default" && settings.configDir) prefix = prefix.concat(["--config-dir", settings.configDir])
    return profileCommand(prefix.concat(args))
  }

  function profileCommand(args) {
    var keys=Object.keys(profileEnvironment)
    if (!keys.length) return args
    return ["/usr/bin/env"].concat(keys.map(function(key){return key+"="+profileEnvironment[key]})).concat(args)
  }

  function checkLocalSetup(manual) {
    if (!localSetupAllowed) { localSetupError="Local host setup is disabled in test profile “"+profileName+"”. Enable it explicitly only on the intended test host.";return }
    if (checkingSetup || settingUp) return
    if (manual===true) setupCheckRequested=true
    localStatusOutput = ""; localSetupError = ""
    localStatus.command = themeCommand(["setup-status"]); localStatus.running = true
  }
  function acceptLocalSetup(value) {
    localSetupStatus = Model.validateSetupStatus(value)
    if (value.ready) {
      var next = Object.assign({}, runnerPreferences)
      var old = next[value.connection.runner_id] || {}
      next[value.connection.runner_id] = {label:old.label || "This computer",location:old.location || "Local",browserBase:old.browserBase || value.browser_base}
      preferences.runners = JSON.stringify(next); preferences.sync()
    }
  }
  function setUpLocal(approved) {
    if (approved!==true || busy || !localSetupStatus || !localSetupStatus.can_setup || localSetupStatus.ready) return false
    if (!machineUse) setMachineUse("always-on")
    setupCheckRequested=true
    localSetupError = ""; localSetupMessage = "Preparing local setup"; localSetupStep = "check"
    localSetupFinished = false; localSetupPending = true
    localSetup.command = themeCommand(["setup"]); localSetup.running = true
    return true
  }
  function finishLocalSetup(code) {
    if (!localSetupPending) return
    localSetupPending = false
    if (code !== 0 || !localSetupFinished) {
      if (!localSetupError) localSetupError = "Setup did not finish. Check the current step, then retry when ready. You can also run omabit --format text setup in a terminal."
    } else {
      localSetupMessage = "This computer is connected. Add a comet to get started."
      Qt.callLater(refresh)
    }
  }

  function refresh() {
    if (activationEnabled && busy) { refreshPending = true; return }
    checkLocalSetup()
    clearEpochPreview()
    importCatalogue = null; pendingImportConnection = null
    refreshPending = false
    refreshing = true
    if (feed.running) feed.running = false
    else Qt.callLater(startFeed)
    loadRegistrations()
  }

  function loadRegistrations() {
    if (registrationList.running) return
    registrationOutput = ""; registrationList.command = command(["runners"]); registrationList.running = true
  }
  function saveRunnerPresentation(connection, label, location, base) {
    if (!connection || !registrations.some(function(r) { return r.name === connection.name && r.runner_id === connection.runner_id })) return false
    try {
      var next = Object.assign({}, runnerPreferences)
      next[connection.runner_id] = {label: label.trim().slice(0, 80), location: location.trim().slice(0, 100), browserBase: Model.browserBase(base)}
      preferences.runners = JSON.stringify(next); preferences.sync()
      lastError = ""; lastResult = "Runner display preferences saved."
      return true
    } catch (e) { lastError = e.message; return false }
  }
  function openMonitor(connection, instance, tab) {
    if (connection && instance) monitor.selectedKey = connection.runner_id + "/" + instance.id
    monitor.page = tab || "overview"
    monitor.present()
  }
  function openSetup(mode) { monitor.page="setup";monitor.setupPane.mode=mode || "home";monitor.present() }
  function openShip(connection, instance) {
    if (!observing || !Model.shipLive(connection, instance, now)) return false
    var url = Model.shipURL(connection, instance, registrations, runnerPreferences)
    if (!url) { lastError = "Set this runner's browser host in Setup before opening a loopback endpoint over SSH."; return false }
    return Qt.openUrlExternally(url)
  }

  function themeCommand(args) {
    var prefix = [omabit, "--format", "json"]
    if (profileName==="default" && settings.configDir) prefix = prefix.concat(["--config-dir", settings.configDir])
    return profileCommand(prefix.concat(args))
  }
  function loadTheme() {
    if (!themeInfo.running) { themeOutput = ""; themeInfo.command = themeCommand(["theme-status"]); themeInfo.running = true }
    if (!paletteInfo.running) { paletteOutput = ""; paletteInfo.command = themeCommand(["theme-preview"]); paletteInfo.running = true }
  }
  function syncThemes() {
    if (busy || themeSync.running) return
    themeSyncOutput = ""
    themeSync.command = themeCommand(["theme-sync"])
    themeSync.running = true
  }
  function themeAuto(row, enabled) { return submit(["theme-auto", row.name, "--expected-ship-id", row.id, "--enabled=" + String(enabled)], "theme-auto", "", true) }
  function publishTheme(row) { return submit(["theme-publish", row.name, "--expected-ship-id", row.id], "theme-publish", "", true) }

  function startFeed() {
    if (feed.running) return
    if (!registrationsLoaded) return
    // Retain observations across reconnects; observing=false keeps actions gated.
    // Only explicitly removed enrollments are removed from the displayed fleet.
    connections = connections.filter(function(c){return registrations.some(function(r){return r.runner_id===c.runner_id && r.name===c.name})})
    feedError = ""
    refreshing = false
    if (registrations.length===0) return
    feed.command = command(["inventory", "--watch", "--interval", "2s"])
    feed.running = true
  }

  function loadShips() {
    if (shipList.running) return
    shipOutput = ""
    shipList.command = command(["ships"])
    shipList.running = true
  }

  function pumpNotifications() {
    if (notification.running || notificationQueue.length === 0) return
    var item = notificationQueue[0]
    notificationQueue = notificationQueue.slice(1)
    // Normal urgency lets the desktop notification service enforce DND.
    notification.command = ["notify-send", "--app-name=Omabit", "--urgency=normal", "--", item.title, item.body]
    notification.running = true
  }

  function submit(args, name, input, useThemeHelper) {
    if (busy) return false
    if (name !== "epoch-preview") clearEpochPreview()
    generation++
    currentAction = name
    output = ""; errors = ""; lastError = ""; lastResult = ""; outputFailed = false
    pendingInput = input || ""
    action.stdinEnabled = pendingInput !== ""
    action.command = useThemeHelper ? themeCommand(args) : command(args)
    timeout.interval = name === "move-host" ? 120*60*1000 : ["register-ship","register-comet","register"].indexOf(name)>=0 ? 20*60*1000 : ["preview-pier","import-pier"].indexOf(name)>=0 ? 60000 : useThemeHelper ? 60000 : 25000
    timeout.restart()
    action.running = true
    var current = generation
    Qt.callLater(function() {
      if (root.generation === current && root.currentAction && !action.running)
        root.finish(-1)
    })
    return true
  }

  function enrollShip(name, url, ship) {
    if (!Model.validId(name) || !url || !ship) { lastError = "Enter a connection name, origin, and expected ~ship."; return false }
    return submit(["ship-enroll", name, "--url", url, "--ship", ship], "ship-enroll")
  }

  function shipAction(row, kind, secret) {
    if (!row || !row.id || busy || ["ship-login", "ship-status", "ship-logout", "ship-forget"].indexOf(kind) < 0) return false
    var args = [kind, row.name, "--expected-ship-id", row.id]
    if (kind === "ship-login") {
      if (!canLogin || !secret || secret.length > 128) return false
      args.push("--code-stdin")
    }
    if (kind === "ship-forget") args = args.concat(["--url", row.origin])
    return submit(args, kind, kind === "ship-login" ? secret : "")
  }

  function operate(connection, instance, kind) {
    if (busy || !observing || !Model.canOperate(connection, instance, kind, now)) return false
    if (kind === "inspect-epochs" && Model.inspectionPending(inspections[Model.inspectionKey(connection,instance)])) return false
    var selected = connections.some(function(c) {
      return c.name === connection.name && c.runner_id === connection.runner_id
        && c.snapshot && c.snapshot.instances.some(function(i) {
          return i.id === instance.id && i.config_revision === instance.config_revision && Model.canOperate(c, i, kind, root.now)
        })
    })
    if (!selected) { lastError = "The selected runtime changed; inspect a fresh observation."; return false }
    requestIdentity = Model.requestId()
    if (kind === "inspect-epochs") rememberInspection({connection:connection.name,runner:connection.runner_id,instance:instance.id,revision:instance.config_revision,request:requestIdentity,status:"submitting",message:"Requesting an inspection of the stopped pier…"})
    return submit(Model.operate(connection, instance, kind, requestIdentity), kind)
  }

  function rememberInspection(target) {
    var next=Object.assign({},inspections)
    next[Model.inspectionKey({runner_id:target.runner,name:target.connection},{id:target.instance})]=target
    inspections=next;epochInspection=target
  }
  function reconcileInspection() {
    var next=Object.assign({},inspections)
    Object.keys(next).forEach(function(key){next[key]=Model.inspectionStatus(connections,next[key],now)})
    if (JSON.stringify(next)!==JSON.stringify(inspections)) inspections=next
    var latest=Model.inspectionStatus(connections,epochInspection,now)
    if (JSON.stringify(latest)!==JSON.stringify(epochInspection)) epochInspection=latest
  }

  function clearEpochPreview() { epochPreview = null; previewTarget = null; pendingPreviewTarget = null }
  function canMaintain(target) { return observing && !busy && Model.currentMaintenanceTarget(connections, target, now) }
  function canSaveRoutine(target) { return observing && !busy && Model.currentRoutineTarget(connections,target,now) }
  function saveRoutine(target,config) {
    if (!canSaveRoutine(target)) return false
    try {
      var args=Model.routineArgs(target,config)
      routineOutcome={target:Object.assign({},target),config:JSON.parse(JSON.stringify(config)),status:"submitting",message:"Saving policy…"}
      return submit(args,"routine-set")
    }
    catch(e) { lastError=e.message;return false }
  }
  function maintain(target) {
    if (!canMaintain(target)) return false
    requestIdentity = Model.requestId()
    return submit(Model.maintenanceArgs(target, requestIdentity), target.kind)
  }
  function canReviewTarget(target) { return observing && !busy && Model.currentEpochTarget(connections, target, now) }
  function invalidateEpochPreview() {
    var target = previewTarget || pendingPreviewTarget
    if (target && (!observing || !Model.currentEpochTarget(connections, target, now))) clearEpochPreview()
  }
  onConnectionsChanged: { invalidateEpochPreview(); invalidateImportCatalogue(); reconcileInspection(); reconcileImports() }
  onNowChanged: invalidateEpochPreview()
  onObservingChanged: invalidateEpochPreview()

  function invalidateImportCatalogue() {
    var target = importCatalogue || pendingImportConnection
    if (target && !Model.importAvailable(connections,target,now)) { importCatalogue = null; pendingImportConnection = null }
  }
  function loadImportSources(connection) {
    var target = connection ? {connection:connection.name,runner:connection.runner_id} : null
    if (!observing || busy || !Model.importAvailable(connections,target,now)) return false
    importCatalogue = null; pendingImportConnection = target
    return submit(Model.selection(connection).concat(["import-sources"]),"import-sources")
  }
  function canImport(target) {
    return observing && !busy && !!target && now-target.loadedAt <= 300000 && now >= target.loadedAt-5000
      && Model.importAvailable(connections,target,now) && Model.validImportSource(target.source) && !target.source.claimed_by
  }
  function importFake(target,confirmed) {
    if (!confirmed || !canImport(target)) return false
    requestIdentity = Model.requestId()
    var submitted = submit(Model.importArgs(target,requestIdentity),"import-fake")
    if (submitted) importCatalogue = null
    return submitted
  }

  function previewEpochs(target) {
    if (!canReviewTarget(target) || !target.cutoff) return false
    clearEpochPreview()
    pendingPreviewTarget = JSON.parse(JSON.stringify(target))
    return submit(Model.epochPreviewArgs(target), "epoch-preview")
  }

  function prunePreview(confirmed) {
    if (!confirmed || !canConfirmPrune) return false
    var target = previewTarget
    requestIdentity = Model.requestId()
    var accepted = submit(Model.epochPruneArgs(target, requestIdentity), "epoch-prune")
    if (accepted) clearEpochPreview()
    return accepted
  }

  function configure(target, port, loom) {
    if (!canConfigure(target)) return false
    try { return submit(Model.configure(target, port, loom), "configure") }
    catch (e) { lastError = e.message; return false }
  }

  function canConfigure(target) {
    if (!target || busy || !observing) return false
    return connections.some(function(c) {
      return c.name === target.connection && c.runner_id === target.runner && Model.writable(c, root.now)
        && c.snapshot.instances.some(function(i) { return i.id === target.instance && i.config_revision===target.revision && !i.active_operation_id && !i.recovery_required && !Model.handoffFenced(i) })
    })
  }

  function enroll(name, host, config, transport) {
    if (!Model.validId(name) || !host) { lastError = "Enter a connection name and SSH host alias."; return false }
    var args = ["enroll", name, transport === "unix" ? "--socket" : "--ssh", host]
    if (config && transport !== "unix") args = args.concat(["--ssh-config", config])
    return submit(args, "enroll")
  }

  function updateRunner(target,name,transport,address,config) {
    try { return submit(Model.connectionEditArgs(target,name,transport,address,config),"runner-edit") }
    catch (e) { lastError = e.message; return false }
  }
  function copyCode(connection,instance) {
    if (busy || copyingCode || !observing || !Model.canCopyCode(connection,instance,now)) {showToast(Model.codeUnavailable(connection,instance,now,observing) || "Wait for the current clipboard or management action.");return false}
    var current = connections.filter(function(c) { return c.name === connection.name && c.runner_id === connection.runner_id })[0]
    if (!current || !current.snapshot.instances.some(function(i) { return i.id === instance.id && i.config_revision === instance.config_revision && Model.canCopyCode(current,i,root.now) && i.resources.container_id === instance.resources.container_id })) {showToast("Ship state changed. Refresh before copying +code.");return false}
    codeSubject=Model.instanceTitle(instance)+" ("+instance.id+")"
    codeOutput = ""; codeNotice = "Requesting +code for "+codeSubject+"…"
    codeCopy.command = themeCommand(["copy-code","--runner",connection.name,"--expected-runner-id",connection.runner_id,"--instance",instance.id,"--revision",String(instance.config_revision),"--container-id",instance.resources.container_id])
    codeCopy.running = true
    Qt.callLater(function() { if (!codeCopy.running && root.codeNotice.indexOf("Requesting +code") === 0) root.showToast("Code helper unavailable; check the installed desktop binary.") })
    return true
  }

  function registerFake(connection, name, port, fakeShip, pierName, bootstrap, pillSource, loom) {
    return registerInstance(connection,name,port,"fake",true,fakeShip,pierName,bootstrap,pillSource,loom)
  }

  function canRegisterComet(connection) {
    return observing && !busy && !!connection && connection.mode === "development-managed"
      && connection.access_role === "operator" && Model.hasFeature(connection,"create-comet") && Model.available(connection,now)
  }
  function registerComet(connection,name,port,confirmed,pierPath,bootstrap,pillSource,loom) {
    return registerInstance(connection,name,port,"comet",confirmed,undefined,pierPath,bootstrap,pillSource,loom)
  }
  function registerInstance(connection,name,port,kind,confirmed,fakeShip,pierName,bootstrap,pillSource,loom) {
    if (busy || !observing || !connection || !confirmed) return false
    var current=connections.filter(function(c){return c.name===connection.name && c.runner_id===connection.runner_id})[0]
    var problem=Model.registrationProblem(current,name,port,kind,now)
    bootstrap=bootstrap || "bare";pillSource=pillSource || ""
    if (!problem) problem=!Model.hasFeature(current,"bootstrap-selection") ? "Update this runner for boot-pill selection." : Model.bootstrapProblem(bootstrap,pillSource)
    if (!problem && kind === "comet" && !canRegisterComet(current)) problem="Comet creation is not available on this runner."
    loom=loom===undefined ? 31 : Number(loom)
    var runtimeArgs=[]
    try {runtimeArgs=Model.runtimeSettingsArgs(current,loom)} catch(e) {problem=e.message}
    var options=kind === "fake" && (fakeShip !== undefined || pierName !== undefined)
    if (!problem && options) problem=Model.fakeOptionsProblem(current,fakeShip,pierName)
    if (problem) { lastError=problem; return false }
    requestIdentity = Model.requestId()
    var args=Model.selection(current).concat([kind === "comet" ? "create-comet" : "create-fake",name,"--http-port",String(port),"--request-id",requestIdentity])
    args=args.concat(Model.bootstrapArgs(bootstrap,pillSource),runtimeArgs)
    if (kind === "comet") args.push("--confirm-network")
    if (options) {pierName=Model.pierPath("fake",name,fakeShip,pierName);args=args.concat(["--ship",Model.fakeIdentity(fakeShip),"--pier-name",pierName.split("/").slice(-2,-1)[0]])}
    if (pierName !== undefined) args=args.concat(["--pier-path",pierName])
    registrationOutcome={runner:current.runner_id,connection:current.name,instance:name,port:Number(port),kind:kind,bootstrap:bootstrap,pillSource:pillSource,loom:loom,nativeControl:false,ship:options ? Model.fakeIdentity(fakeShip) : "",pier:options ? pierName : "",request:requestIdentity,status:"submitting",message:"Registering “"+name+"”…"}
    return submit(args,kind === "comet" ? "register-comet" : "register")
  }
  function suggestPort(connection, token) {
    if (busy) return false
    portSuggestion=null;portError="";portTarget={runner:connection ? connection.runner_id : "",connection:connection ? connection.name : "",token:token}
    var c=storageConnection(connection,"host-port-suggestion")
    if (!c) {portError="Update and connect the runner to check its host ports.";return false}
    return submit(Model.selection(c).concat(["suggest-port"]),"suggest-port")
  }
  function dispatchFirstBoot() {
    var target=pendingFirstBoot
    if (!target) return
    if (action.running || busy) { firstBootDispatch.restart();return }
    pendingFirstBoot=null // consume once; never retry a lost Start acknowledgement
    bootTarget=target
    var args=["--runner",target.connection,"--expected-runner-id",target.runner,"operate",target.instance,"start","--revision",String(target.revision),"--request-id",target.request]
    registrationOutcome=Object.assign({},registrationOutcome,{message:"Booting “"+target.instance+"”…"})
    if (!submit(args,"first-boot")) {bootTarget=null;lastError="Registration completed, but boot was not submitted. Inspect the ship before starting it."}
    var c=connections.filter(function(c){return c.name===target.connection && c.runner_id===target.runner})[0]
    if (c) openMonitor(c,{id:target.instance},"overview")
  }
  Timer { id: firstBootDispatch; interval: 20; onTriggered: root.dispatchFirstBoot() }
  function editRegistration() {
    if (!busy && registrationOutcome) { registrationOutcome=null;lastError="";lastResult="" }
  }
  function enrollDiscovered(connection,source,name,ship,port,confirmed) {
    if (source && source.enrolled_by && source.moved!==true) {lastError="This pier is already enrolled as "+source.enrolled_by+".";return false}
    if (busy || !observing || !confirmed || !connection || connection.access_role!=="operator" || !Model.currentPierSource(connection,source,now) || !Model.validId(name)) {lastError="Confirm a stopped pier and review a fresh operator connection before enrolling.";return false}
    var current=connections.filter(function(c){return c.name===connection.name && c.runner_id===connection.runner_id})[0]
    if (!current || !Model.currentPierSource(current,source,now)) {lastError="Discovery changed; refresh and review this pier again.";return false}
    requestIdentity=Model.requestId()
    var args=Model.selection(current).concat(["enroll-pier",name,"--path",source.path,"--ship",ship,"--fingerprint",source.fingerprint,"--request-id",requestIdentity,"--confirm-stopped"])
    if (source.enrolled_by) {
      var i=Model.instances(current).filter(function(i){return i.id===source.enrolled_by})[0]
      if (source.moved!==true || !i || i.id!==name || i.desired_state!=="stopped" || i.active_operation_id) {lastError="Stop the enrolled ship before reconnecting its moved directory.";return false}
      args=args.concat(["--relink","--revision",String(i.config_revision)])
    } else args=args.concat(["--http-port",String(port)])
    storageOutcome={runner:current.runner_id,instance:name,path:source.path,request:requestIdentity,kind:source.enrolled_by ? "relink-pier" : "enroll-pier",status:"submitting",message:"Submitting pier enrollment…"}
    return submit(args,"enroll-pier")
  }
  function movePier(connection,target,storageRoot,relativePath) {
    var c=storageConnection(connection,"pier-storage"), path=Model.storagePath(storageRoot,relativePath)
    if (!Model.currentPierMove(c,target,now) || !storageRoot || !Model.validPierPath(path) || path===target.path || !Model.storageRoots(c).some(function(r){return r.id===storageRoot})) {lastError="The move selection changed. Review the current stopped pier and destination.";return false}
    requestIdentity=Model.requestId()
    storageOutcome={runner:target.runner,connection:target.connection,instance:target.instance,path:path,request:requestIdentity,kind:"move-pier",status:"submitting",message:"Submitting stopped-pier move…"}
    return submit(["--runner",target.connection,"--expected-runner-id",target.runner,"move-pier",target.instance,"--path",path,"--revision",String(target.revision),"--request-id",requestIdentity],"move-pier")
  }
  function storageConnection(connection,feature) {
    if (busy || !observing || !connection) return null
    var c=connections.filter(function(c){return c.name===connection.name && c.runner_id===connection.runner_id})[0]
    return c && Model.writable(c,now) && c.access_role==="operator" && Model.hasFeature(c,feature) ? c : null
  }
  function addStorageRoot(connection,name,path) {
    if (busy) return false
    var c=storageConnection(connection,"storage-locations")
    if (!c || !/^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$/.test(name) || name==="default" || !Model.hostPath(path)) {lastError="Choose an available operator runner, a storage name and an absolute host directory.";return false}
    requestIdentity=Model.requestId()
    storageOutcome={runner:c.runner_id,instance:"",path:path,request:requestIdentity,kind:"storage-root",status:"submitting",message:"Adding storage location…"}
    return submit(Model.selection(c).concat(["storage-root-add",name,"--path",path,"--request-id",requestIdentity]),"storage-root-add")
  }
  function previewExternalPier(connection,path,storageRoot) {
    if (busy) return false
    var c=storageConnection(connection,"guided-pier-import")
    storageRoot=storageRoot===undefined ? "default" : storageRoot
    if (!c || !Model.hostPath(path) || !Model.storageRoots(c).some(function(r){return r.id===storageRoot})) {lastError="Enter the absolute existing pier directory and choose a storage location on the selected runner host.";return false}
    externalImportPreview=null
    pendingExternalPreview={runner:c.runner_id,connection:c.name,path:path,storageRoot:storageRoot}
    return submit(Model.selection(c).concat(["preview-pier","--source",path,"--storage-root",pendingExternalPreview.storageRoot]),"preview-pier")
  }
  function importExternalPier(connection,review,confirmed) {
    if (busy) return false
    var c=storageConnection(connection,"guided-pier-import")
    if (!c || !confirmed || !Model.externalReviewCurrent(c,review,review && review.source.path,now) || !review.plan) {lastError="Inspect the source and confirm the reviewed import.";return false}
    var plan=review.plan
    try { Model.validatePierImportPreview(plan,c.runner_id,review.source.path,plan.storage_root) } catch(e) {lastError=e.message;return false}
    if (Object.keys(importOutcomes).some(function(key){var o=importOutcomes[key];return o.runner===c.runner_id && o.source===review.source.path && Model.importProgress(connections,o,now).active})) {lastError="This source already has an unresolved import request. Wait for its receipt.";return false}
    requestIdentity=Model.requestId()
    var target=Object.freeze({runner:c.runner_id,connection:c.name,instance:plan.instance_id,source:review.source.path,path:plan.destination,ship:plan.ship,request:requestIdentity,kind:"import-pier",status:"submitting",message:"Submitting reviewed pier import…"})
    storageOutcome=target
    if (!submit(Model.selection(c).concat(["import-pier",plan.instance_id,"--source",review.source.path,"--path",plan.destination,"--ship",plan.ship,"--fingerprint",review.source.fingerprint,"--http-port",String(plan.http_port),"--request-id",requestIdentity,"--confirm-stopped","--identify"]),"import-pier")) return false
    rememberImport(target)
    return target
  }
  function rememberImport(target) {
    var key=Model.importRequestKey(target), old=importOutcomes[key], next=Object.assign({},importOutcomes)
    var retained=Object.assign({},target,{progress:old && old.progress})
    retained.progress=Model.importProgress(connections,retained,now)
    next[key]=retained;importOutcomes=next
  }
  function reconcileImports() {
    var next=Object.assign({},importOutcomes)
    Object.keys(next).forEach(function(key){var o=next[key];next[key]=Object.assign({},o,{progress:Model.importProgress(connections,o,now)})})
    if (JSON.stringify(next)!==JSON.stringify(importOutcomes)) importOutcomes=next
  }
  function deletePier(connection,target,typed) {
    if (busy) return false
    var c=storageConnection(connection,"delete-pier")
    var i=Model.instances(c).filter(function(i){return target && i.id===target.id})[0]
    if (!Model.canDeletePier(c,i,now) || typed!==target.id || i.config_revision!==target.config_revision || i.pier_path!==target.pier_path || i.ship!==target.ship) {lastError="The deletion review changed. Review the current stopped instance and type its exact name.";return false}
    requestIdentity=Model.requestId()
    storageOutcome={runner:c.runner_id,instance:i.id,path:i.pier_path,request:requestIdentity,kind:"delete-pier",status:"submitting",message:"Submitting permanent pier deletion…"}
    return submit(Model.selection(c).concat(["delete-pier",i.id,"--revision",String(i.config_revision),"--request-id",requestIdentity,"--confirm-delete"]),"delete-pier")
  }
  function recoverImport(target) {
    if (busy || !observing || !target) return false
    var c=connections.filter(function(c){return c.runner_id===target.runner && c.name===target.connection})[0]
    var i=Model.instances(c).filter(function(i){return i.id===target.instance})[0]
    if (!Model.currentImportRecovery(c,i,target,now)) {lastError="The recovery selection changed. Review a fresh stopped import.";return false}
    requestIdentity=Model.requestId()
    storageOutcome=Object.assign({},target,{request:requestIdentity,kind:"recover-import",status:"submitting",message:"Validating the retained import…"})
    return submit(Model.selection(c).concat(["operate",target.instance,"recover-import","--revision",String(target.revision),"--request-id",requestIdentity]),"recover-import")
  }
  function moveHost(target,destination,name,httpPort,amesPort) {
    if (busy || !observing) return false
    var problem=Model.hostMoveProblem(connections,target,destination,name,httpPort,amesPort,now)
    if (problem) {lastError=problem;return false}
    requestIdentity=Model.requestId()
    hostMoveOutcome={runner:target.runner,instance:target.instance,destination:destination.runner,name:name,status:"moving",message:"Moving the stopped pier. Both hosts retain durable phase receipts."}
    return submit(Model.hostMoveArgs(target,destination,name,httpPort,amesPort,requestIdentity),"move-host")
  }
  function createFromKey(connection,ship,keyFile,httpPort,amesPort,pill,bootstrap,storageRoot,customPath,loom) {
    bootstrap=bootstrap || (pill ? "recommended" : "bare")
    var bootProblem=Model.bootstrapProblem(bootstrap,pill || "")
    if (bootProblem) {lastError=bootProblem;return false}
    ship=Model.fakeIdentity(ship)
    var name=ship.slice(1),path=Model.ownedPierPath(ship)
    if (path) path=Model.storagePath(storageRoot,path)
    if (customPath) path=customPath
    if (busy || !observing || !Model.available(connection,now) || connection.access_role!=="operator" || !Model.hasFeature(connection,"key-boot-options") || !Model.validId(name) || !path || !keyFile || !Model.optionalPort(httpPort) || !Model.optionalPort(amesPort)) {lastError="Choose an owned Urbit ID, private key file and valid optional ports on an updated operator runner.";return false}
    if (Model.instances(connection).some(function(i){return i.id===name || i.mode!=="fake"&&i.ship===ship})) {lastError="This Urbit identity or instance is already registered.";return false}
    loom=loom===undefined ? 31 : Number(loom)
    var runtimeArgs=[]
    try {runtimeArgs=Model.runtimeSettingsArgs(connection,loom)} catch(e) {lastError=e.message;return false}
    requestIdentity=Model.requestId()
    registrationOutcome={runner:connection.runner_id,connection:connection.name,instance:name,kind:"owned",ship:ship,pier:path,loom:loom,nativeControl:false,request:requestIdentity,status:"submitting",message:"Preparing boot files and registering the ship…"}
    var args=Model.selection(connection).concat(["create-ship","--ship",ship,"--key-file",keyFile,"--request-id",requestIdentity,"--confirm-network"])
    if (String(httpPort).trim()) args=args.concat(["--http-port",String(httpPort)])
    if (String(amesPort).trim()) args=args.concat(["--ames-port",String(amesPort)])
    if (customPath || storageRoot && storageRoot!=="default") args=args.concat(["--pier-path",path])
    args=args.concat(Model.bootstrapArgs(bootstrap,pill || ""),runtimeArgs)
    return submit(args,"register-ship")
  }
  function acceptDevelopmentMode(connection) {
    var entry = registrations.filter(function(r) { return r.name === connection.name && r.runner_id === connection.runner_id })[0]
    if (busy || !entry || entry.mode !== "development-managed-fake" || !Number.isSafeInteger(entry.revision)) return false
    return submit(["runner-accept-mode",entry.name,"--runner-id",entry.runner_id,"--revision",String(entry.revision),"--mode","development-managed"],"runner-accept-mode")
  }

  function finish(code) {
    if (!currentAction) return
    timeout.stop()
    var kind = currentAction
    var hostMove=kind==="move-host" ? hostMoveOutcome : null, hostMoved=false
    var registration=(kind === "register" || kind === "register-comet" || kind === "register-ship") ? registrationOutcome : null
    var registered=false
    var boot=kind==="first-boot" ? bootTarget : null, bootAccepted=false
    var storage=(["enroll-pier","move-pier","delete-pier","import-pier","recover-import","storage-root-add"].indexOf(kind)>=0) ? storageOutcome : null, storageAccepted=false
    var routine=kind === "routine-set" ? routineOutcome : null, routineSaved=false
    var inspecting=kind === "inspect-epochs" ? epochInspection : null, inspectionAccepted=false, inspectionOperation=""
    currentAction = ""
    pendingInput = ""
    action.stdinEnabled = false
    if (kind === "ship-status" && !outputFailed && output) {
      try { var statusResult = JSON.parse(output); if (statusResult.id) { var statuses = Object.assign({}, shipStatus); statuses[statusResult.id] = statusResult; shipStatus = statuses; lastResult = statusResult.name + ": " + statusResult.status } } catch (_) {}
    }
    if (code !== 0 || outputFailed) lastError = outputFailed ? "Collector output exceeded its limit." : Model.errorMessage(errors)
    else {
      try {
        var result = JSON.parse(output)
        if (kind==="suggest-port") {
          var pt=portTarget
          if (!pt || result.runner_id!==pt.runner || !Number.isInteger(result.http_port) || result.http_port<1024 || result.http_port>65535 || result.container_port!==8080) throw new Error("Invalid port suggestion")
          portSuggestion=Object.assign({},pt,{port:result.http_port})
        }
        if (boot) {
          if (!result.id || result.instance_id!==boot.instance || result.request_id!==boot.request || result.kind!=="start" || result.config_revision!==boot.revision) throw new Error("Invalid first boot acknowledgement")
          bootAccepted=true
        }
        if (kind==="preview-pier") {
          var preview=pendingExternalPreview
          pendingExternalPreview=null
          if (!preview) throw new Error("Missing import selection")
          Model.validatePierImportPreview(result,preview.runner,preview.path,preview.storageRoot)
          externalImportPreview={runner:preview.runner,connection:preview.connection,source:result.source,plan:result,loadedAt:now}
        }
        if (hostMove) {
          if (result.source_runner!==hostMove.runner || result.source_instance!==hostMove.instance || result.destination_runner!==hostMove.destination || result.destination_instance!==hostMove.name || result.state!=="complete" || result.direction!=="in") throw new Error("Unverified host move receipt")
          hostMoved=true
        }
        if (storage) {
          if (!result.id || result.instance_id!==storage.instance || result.request_id!==storage.request || result.kind!==storage.kind) throw new Error("Unverified storage operation acknowledgement")
          if (kind==="recover-import" && result.config_revision!==storage.revision) throw new Error("Recovery revision changed")
          storageAccepted=true
        }
        if (registration) {
          if (!result.id || result.instance_id !== registration.instance || result.request_id !== registration.request || result.kind !== (registration.kind === "comet" ? "register-comet" : registration.kind === "owned" ? "register-ship" : "register-fake") || result.status !== "succeeded") throw new Error("Unverified registration result")
          if (!Number.isSafeInteger(result.config_revision) || result.config_revision<1) throw new Error("Missing registration revision")
          registered=true
          pendingFirstBoot={runner:registration.runner,connection:registration.connection,instance:registration.instance,revision:result.config_revision,request:Model.requestId()}
        }
        if (routine) {
          if (!result.id || result.instance_id !== routine.target.instance || result.instance_revision !== routine.target.instanceRevision || !Number.isSafeInteger(result.revision) || result.revision < Math.max(1,routine.target.revision) || result.revision > routine.target.revision+1 || !Model.sameRoutineConfig(result.config,routine.config)) throw new Error("Unverified policy result")
          routineSaved=true
        }
        if (inspecting) {
          if (!result.id || result.instance_id !== inspecting.instance || result.kind !== "inspect-epochs" || result.request_id !== inspecting.request || result.config_revision !== inspecting.revision) throw new Error("Unverified inspection acknowledgement")
          inspectionAccepted=true; inspectionOperation=result.id
        }
        if (kind.indexOf("ship-") === 0 && result.id) {
          var next = Object.assign({}, shipStatus)
          next[result.id] = result
          shipStatus = next
        }
        if (kind === "import-sources") {
          var selected = pendingImportConnection; pendingImportConnection = null
          if (!selected || !observing || !Model.importAvailable(connections,selected,now) || result.runner_id !== selected.runner
              || !Array.isArray(result.sources) || result.sources.length > 256 || !result.sources.every(Model.validImportSource)) throw new Error("Invalid import catalogue")
          importCatalogue = {connection:selected.connection,runner:selected.runner,loadedAt:now,sources:result.sources}
          lastResult = "Qualified source catalogue loaded."
        } else if (kind === "epoch-preview") {
          var target = pendingPreviewTarget
          pendingPreviewTarget = null
          if (target && observing && Model.currentEpochTarget(connections, target, now)) {
            Model.validateEpochPreview(result, target)
            previewTarget = target; epochPreview = result
            lastResult = "Epoch prefix preview ready for review."
          } else lastError = "The preview selection changed; review a fresh stopped inventory."
        } else lastResult = result.id ? kind + ": " + result.id + (result.status ? " — " + result.status : "") : kind + " completed"
      } catch (_) { lastError = "Invalid collector result; inspect a fresh snapshot before retrying." }
    }
    if (registration) {
      var outcome=Object.assign({},registration)
      // Collector exit 2 is argument validation before any mutation. Other
      // failures need an explicit no-effect API rejection to permit a retry.
      outcome.status=registered ? "succeeded" : !outputFailed && (code === 2 || code !== 0 && !Model.registrationFailure(errors).uncertain) ? "failed" : "uncertain"
      outcome.message=registered ? "Registered “"+registration.instance+"”. Starting its first boot…" : lastError
      registrationOutcome=outcome
      if (registered && pendingFirstBoot) firstBootDispatch.restart()
    }
    if (boot) {
      bootTarget=null
      registrationOutcome=Object.assign({},registrationOutcome,{message:bootAccepted ? "Boot requested. Follow startup in the ship's management view." : "Registered, but the boot result needs review: "+lastError})
    }
    if (storage) {
      var stored=Object.assign({},storage)
      if (storageAccepted) {stored.operation=result.id;stored.revision=result.config_revision}
      stored.status=storageAccepted ? "accepted" : !outputFailed && (code===2 || code!==0 && !Model.registrationFailure(errors).uncertain) ? "failed" : "uncertain"
      stored.message=storageAccepted ? "Request accepted. The fleet and activity show completion; no automatic Start was requested." : lastError
      storageOutcome=stored
      if (stored.kind==="import-pier") rememberImport(stored)
    }
    if (hostMove) hostMoveOutcome=Object.assign({},hostMove,{status:hostMoved ? "complete" : "paused",message:hostMoved ? "Move complete. The destination is stopped; choose Start there when ready." : lastError+" Read the recorded phase and continue the same move."})
    if (routine) {
      var saved=Object.assign({},routine)
      saved.status=routineSaved ? "saved" : !outputFailed && (code === 2 || code !== 0 && ["revision_conflict","invalid_request","ssh_access_denied"].indexOf(Model.errorCode(errors))>=0) ? "failed" : "uncertain"
      saved.message=routineSaved ? (routine.config.enabled ? "Policy saved and enabled." : "Policy saved, disabled. Use Enable policy when ready.") : lastError
      routineOutcome=saved
    }
    if (inspecting) {
      var rejected=!inspectionAccepted && Model.inspectionRejected(code,errors,outputFailed)
      rememberInspection(Object.assign({},inspecting,{operation:inspectionOperation,status:inspectionAccepted ? "queued" : rejected ? "failed" : "uncertain",message:inspectionAccepted ? "Inspection accepted. Waiting for the runner's inventory…" : rejected ? lastError+" Review the current stopped ship and inspect again." : "Inspection acknowledgement unavailable. Checking the journal; do not resubmit blindly."}))
      reconcileInspection()
    }
    output = ""; errors = ""
    if (kind==="suggest-port") {portError=lastError;lastError="";lastResult=""}
    if (kind === "epoch-preview" && lastError) clearEpochPreview()
    if (kind === "import-sources" && lastError) { importCatalogue = null; pendingImportConnection = null }
    if (kind.indexOf("ship-") === 0) Qt.callLater(loadShips)
    if (kind.indexOf("theme-") === 0) { Qt.callLater(loadTheme); if (!lastError) Qt.callLater(syncThemes) }
    if (refreshPending || ((kind === "enroll" || kind === "runner-edit" || kind === "runner-accept-mode") && !lastError)) Qt.callLater(refresh)
  }

  function saveDesktopView() {
    if (!desktopViewLoaded) return
    preferences.desktopPage=monitor.page
    preferences.desktopSelection=monitor.selectedKey
    // Settings' property-change writeback is debounced. Flush the explicit keys
    // before a bootstrap/process restart, rather than only syncing its old map.
    preferences.setValue("desktopPage",monitor.page)
    preferences.setValue("desktopSelection",monitor.selectedKey)
    preferences.sync()
  }
  Component.onCompleted: {
    if (["overview","maintenance","activity","setup"].indexOf(preferences.desktopPage)>=0) monitor.page=preferences.desktopPage
    if (preferences.desktopSelection.length<=512) monitor.selectedKey=preferences.desktopSelection
    desktopViewLoaded=true
    Qt.callLater(refresh)
  }
  Component.onDestruction: pendingInput = ""
  Timer { interval: 1000; repeat: true; running: true; onTriggered: root.now = Date.now() }
  Timer {
    id: timeout
    interval: 25000
    onTriggered: { root.pendingInput = ""; root.errors = ""; if (action.running) action.signal(9); else root.finish(-1) }
  }
  Process {
    id: feed
    objectName: "inventoryFeed"
    stdout: SplitParser {
      onRead: function(line) {
        if (!line.trim()) return
        try {
          var next = Model.observation(line)
          var previous = root.connections.filter(function(c) { return c.name === next.name })[0]
          if (root.settings.notifications !== false && !root.settingUp) {
            root.notificationQueue = root.notificationQueue.concat(Model.notices(previous, next)).slice(-8)
            root.pumpNotifications()
          }
          root.connections = Model.merge(root.connections, next); root.feedError = ""; root.reconnectAttempts=0
           root.memoryHistory = Model.recordSamples(root.memoryHistory, next, root.now)
           root.cpuHistory = Model.recordSamples(root.cpuHistory, next, root.now,"cpu")
        }
        catch (e) { root.feedError = e.message; feed.running = false }
      }
    }
    stderr: SplitParser { onRead: function(line) { if (!root.firstRun && line.length < 8192) root.feedError = "Collector inventory is unavailable. Check enrollment and SSH access." } }
    onExited: function(code, status) {
      if (root.refreshing) Qt.callLater(root.startFeed)
      else if (!root.firstRun) { if (!root.feedError) root.feedError = "Inventory polling stopped. Reconnecting…"; reconnect.restart() }
    }
    onRunningChanged: if (!running) Qt.callLater(function() {
      if (!feed.running && !root.firstRun && !root.refreshing && !root.feedError) root.feedError = "Inventory process is unavailable. Check the collector executable and refresh."
    })
  }
  Timer {
    id: reconnect
    interval:Math.min(30000,1000*Math.pow(2,Math.min(root.reconnectAttempts,5)))
    onTriggered: { if (!feed.running && root.registrationsLoaded && root.registrations.length) {root.reconnectAttempts++;root.startFeed()} }
  }
  Process {
    id: action
    objectName: "collectorAction"
    onStarted: { if (root.pendingInput) write(root.pendingInput); root.pendingInput = ""; stdinEnabled = false }
    stdout: SplitParser { splitMarker: ""; onRead: function(data) { if (root.output.length + data.length > 2 * 1024 * 1024) { root.outputFailed = true; action.signal(9) } else root.output += data } }
    stderr: SplitParser { splitMarker: ""; onRead: function(data) { if (root.errors.length + data.length < 16384) root.errors += data } }
    onExited: function(code, status) { root.finish(status === 0 ? code : -1) }
    onRunningChanged: if (!running && root.currentAction) {
      root.pendingInput = ""
      var current = root.generation
      Qt.callLater(function() { if (root.generation === current && root.currentAction && !action.running) root.finish(-1) })
    }
  }
  Process {
    id: localStatus
    objectName: "localSetupStatus"
    stdout: SplitParser { splitMarker: ""; onRead: function(data) { if (root.localStatusOutput.length + data.length > 131072) { localStatus.signal(9); return } root.localStatusOutput += data } }
    stderr: SplitParser { onRead: function(line) {} }
    onExited: function(code,status) {
      if (code === 0 && status === 0) {
        try { root.acceptLocalSetup(JSON.parse(root.localStatusOutput)) }
        catch (_) { root.localSetupError = "Local setup status could not be read. Check that the bundled helpers are installed." }
      } else root.localSetupError = "Local setup check is unavailable. Rebuild/reinstall the panel helpers, then check again."
      root.localStatusOutput = ""
    }
  }
  Process {
    id: localSetup
    objectName: "localSetupAction"
    stdout: SplitParser { onRead: function(line) {
      if (!line.trim()) return
      try {
        var event = Model.setupEvent(line)
        if (event.step !== "setup") root.localSetupStep = event.step
        root.localSetupMessage = event.message || Model.setupStepLabel(event.step)
        if (event.status === "failed") root.localSetupError = event.message || "Host setup stopped."
        if (event.result) { root.acceptLocalSetup(event.result); root.localSetupFinished = event.result.ready }
      } catch (_) { root.localSetupError = "Unexpected setup response. Inspect setup status before retrying."; localSetup.signal(9) }
    } }
    stderr: SplitParser { onRead: function(line) {} }
    onExited: function(code,status) { root.finishLocalSetup(status === 0 ? code : -1) }
    onRunningChanged: if (!running && root.localSetupPending) Qt.callLater(function(){ if (!localSetup.running && root.localSetupPending) root.finishLocalSetup(-1) })
  }
  Process {
    id: themeInfo
    stdout: SplitParser { splitMarker: ""; onRead: function(data) { if (root.themeOutput.length + data.length < 262144) root.themeOutput += data } }
    stderr: SplitParser { splitMarker: ""; onRead: function(data) {} }
    onExited: function(code, status) { if (code === 0 && status === 0) { try { var state = JSON.parse(root.themeOutput); if (state.version === 1 && state.ships) root.themeState = state } catch (_) {} } root.themeOutput = "" }
  }
  Process {
    id: paletteInfo
    stdout: SplitParser { splitMarker: ""; onRead: function(data) { if (root.paletteOutput.length + data.length < 16384) root.paletteOutput += data } }
    stderr: SplitParser { splitMarker: ""; onRead: function(data) {} }
    onExited: function(code, status) { if (code === 0 && status === 0) { try { var value = JSON.parse(root.paletteOutput); if (value.id === "omarchy-urbit-theme") root.palette = value } catch (_) {} } root.paletteOutput = "" }
  }
  Process {
    id: themeSync
    stdout: SplitParser { splitMarker: ""; onRead: function(data) { if (root.themeSyncOutput.length + data.length < 65536) root.themeSyncOutput += data } }
    stderr: SplitParser { splitMarker: ""; onRead: function(data) {} }
    onExited: { root.themeSyncOutput = ""; Qt.callLater(root.loadTheme) }
  }
  Process {
    id: notification
    stdout: SplitParser { splitMarker: ""; onRead: function(data) {} }
    stderr: SplitParser { splitMarker: ""; onRead: function(data) {} }
    onExited: Qt.callLater(root.pumpNotifications)
  }
  Process {
    id: codeCopy
    stdout: SplitParser { splitMarker: ""; onRead: function(data) {
      if (root.codeOutput.length + data.length > 4096) { codeCopy.signal(9); return }
      root.codeOutput += data
       try { if (JSON.parse(root.codeOutput).status === "clipboard_handoff_started") {root.codeNotice="Copied to sensitive clipboard for 30s";root.showToast(root.codeNotice)} } catch (_) {}
    } }
    stderr: SplitParser { splitMarker: ""; onRead: function(data) {} }
    onExited: function(code,status) { root.codeOutput = ""; if (code!==0 || status!==0) root.showToast("Could not copy +code. Refresh and check operator access."); root.codeNotice="" }
  }
  Timer { id: toastExpiry; interval: 4500; onTriggered: root.toastText="" }
  Process {
    id: registrationList
    stdout: SplitParser { splitMarker: ""; onRead: function(data) { if (root.registrationOutput.length + data.length < 262144) root.registrationOutput += data } }
    stderr: SplitParser { onRead: function(line) {} }
    onExited: function(code, status) {
      if (code === 0 && status === 0) {
        try {
          var rows = JSON.parse(root.registrationOutput)
          if (!Array.isArray(rows)) throw new Error("Invalid enrollments")
          root.registrations=rows;root.registrationsLoaded=true
          if (!feed.running) Qt.callLater(root.startFeed)
        } catch (_) { root.feedError="Collector enrollment could not be read. Check the collector configuration." }
      } else {
        root.feedError="Collector enrollment could not be read. Check the collector executable and configuration."
      }
      root.registrationOutput = ""
    }
  }
  Core.Settings {
    id: preferences
    location: "file://" + (root.profileEnvironment.XDG_STATE_HOME || Quickshell.env("XDG_STATE_HOME") || Quickshell.env("HOME") + "/.local/state") + "/omabit/panel.ini"
    property string runners: "{}"
    property string machineUse: ""
    property string desktopPage: "overview"
    property string desktopSelection: ""
  }
  MonitorWindow { id: monitor; service: root }
  IpcHandler {
    target: "thelifeandtimes.omabit.monitor"
    function open(): void { root.openMonitor(null, null, "overview") }
    function activity(): void { root.openMonitor(null, null, "activity") }
    function maintenance(): void { root.openMonitor(null, null, "maintenance") }
    function setup(): void { root.openMonitor(null, null, "setup") }
    function refresh(): void { root.refresh() }
    function status(): string { return JSON.stringify({version:root.applicationVersion,visible:monitor.visible,page:monitor.page,fullscreen:monitor.fullscreen,busy:root.busy,first_run:root.firstRun,recommended_pill_default:root.recommendedPillUrl==="https://urbit-foundation.s3.us-east-2.amazonaws.com/omabit.pill"}) }
    function close(): void { monitor.visible = false }
  }
  Process {
    id: shipList
    objectName: "shipList"
    stdout: SplitParser { splitMarker: ""; onRead: function(data) { if (root.shipOutput.length + data.length < 262144) root.shipOutput += data } }
    stderr: SplitParser { splitMarker: ""; onRead: function(data) {} }
    onExited: function(code, status) {
      if (code === 0 && status === 0) {
        try { var list = JSON.parse(root.shipOutput); if (Array.isArray(list)) root.ships = list }
        catch (_) { root.lastError = "Ship enrollment list could not be read." }
      }
      root.shipOutput = ""
    }
  }
}
