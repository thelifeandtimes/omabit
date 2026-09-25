import QtQuick
import Quickshell.Io
import "Model.js" as Model

Item {
  id: root
  property var shell: null
  property var manifest: null
  property var settings: ({})
  property var connections: []
  property var ships: []
  property string lastError: ""
  property string lastResult: ""
  property string requestIdentity: ""
  property double now: Date.now()
  property bool refreshing: false
  property bool refreshPending: false
  readonly property bool busy: action.running
  readonly property bool observing: feed.running && !feedError
  property string feedError: ""
  property string output: ""
  property string errors: ""
  property string currentAction: ""
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
  readonly property string omabit: settings.omabitPath || Model.localPath(Qt.resolvedUrl("bin/omabit"))
  readonly property bool canLogin: !busy && currentAction === ""
  readonly property string collector: settings.collectorPath || Model.localPath(Qt.resolvedUrl("bin/urbit-collector"))

  function command(args) {
    var prefix = [collector, "--format", "json"]
    if (settings.configDir) prefix = prefix.concat(["--config-dir", settings.configDir])
    return prefix.concat(args)
  }

  function refresh() {
    if (busy) { refreshPending = true; return }
    refreshPending = false
    refreshing = true
    if (feed.running) feed.running = false
    else Qt.callLater(startFeed)
    loadShips()
    loadTheme()
  }

  function themeCommand(args) {
    var prefix = [omabit, "--format", "json"]
    if (settings.configDir) prefix = prefix.concat(["--config-dir", settings.configDir])
    return prefix.concat(args)
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
    connections = []
    feedError = ""
    refreshing = false
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
    generation++
    currentAction = name
    output = ""; errors = ""; lastError = ""; lastResult = ""; outputFailed = false
    pendingInput = input || ""
    action.stdinEnabled = pendingInput !== ""
    action.command = useThemeHelper ? themeCommand(args) : command(args)
    timeout.interval = useThemeHelper ? 60000 : 25000
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
    if (!observing || !Model.canOperate(connection, instance, kind, now)) return
    requestIdentity = Model.requestId()
    submit(Model.operate(connection, instance, kind, requestIdentity), kind)
  }

  function configure(target, port) {
    if (!canConfigure(target)) return false
    try { return submit(Model.configure(target, port), "configure") }
    catch (e) { lastError = e.message; return false }
  }

  function canConfigure(target) {
    if (!target || busy || !observing) return false
    return connections.some(function(c) {
      return c.name === target.connection && c.runner_id === target.runner && Model.writable(c, root.now)
        && c.snapshot.instances.some(function(i) { return i.id === target.instance && !i.active_operation_id && !i.recovery_required })
    })
  }

  function enroll(name, host, config) {
    if (!Model.validId(name) || !host) { lastError = "Enter a connection name and SSH host alias."; return false }
    var args = ["enroll", name, "--ssh", host]
    if (config) args = args.concat(["--ssh-config", config])
    return submit(args, "enroll")
  }

  function registerFake(connection, name, port) {
    if (!observing || !Model.writable(connection, now) || !Model.validId(name)
        || Number(port) < 1024 || Number(port) > 65535) { lastError = "Select an available operator connection, a valid name, and port 1024–65535."; return false }
    requestIdentity = Model.requestId()
    return submit(Model.selection(connection).concat(["create-fake", name, "--http-port", String(port), "--request-id", requestIdentity]), "register")
  }

  function finish(code) {
    if (!currentAction) return
    timeout.stop()
    var kind = currentAction
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
        if (kind.indexOf("ship-") === 0 && result.id) {
          var next = Object.assign({}, shipStatus)
          next[result.id] = result
          shipStatus = next
        }
        lastResult = result.id ? kind + ": " + result.id + (result.status ? " — " + result.status : "") : kind + " completed"
      } catch (_) { lastError = "Invalid collector result; inspect a fresh snapshot before retrying." }
    }
    output = ""; errors = ""
    if (kind.indexOf("ship-") === 0) Qt.callLater(loadShips)
    if (kind.indexOf("theme-") === 0) { Qt.callLater(loadTheme); if (!lastError) Qt.callLater(syncThemes) }
    if (refreshPending || (kind === "enroll" && !lastError)) Qt.callLater(refresh)
  }

  Component.onCompleted: Qt.callLater(refresh)
  Component.onDestruction: pendingInput = ""
  Timer { interval: 1000; repeat: true; running: true; onTriggered: root.now = Date.now() }
  Timer { interval: 30000; repeat: true; running: true; onTriggered: root.syncThemes() }
  Timer { interval: 1000; running: true; onTriggered: root.syncThemes() }
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
          if (root.settings.notifications !== false) {
            root.notificationQueue = root.notificationQueue.concat(Model.notices(previous, next)).slice(-8)
            root.pumpNotifications()
          }
          root.connections = Model.merge(root.connections, next); root.feedError = ""
        }
        catch (e) { root.feedError = e.message; feed.running = false }
      }
    }
    stderr: SplitParser { onRead: function(line) { if (line.length < 8192) root.feedError = "Collector inventory is unavailable. Check enrollment and SSH access." } }
    onExited: function(code, status) {
      if (root.refreshing) Qt.callLater(root.startFeed)
      else if (!root.feedError) root.feedError = "Inventory polling stopped. Refresh to reconnect."
    }
    onRunningChanged: if (!running) Qt.callLater(function() {
      if (!feed.running && !root.refreshing && !root.feedError) root.feedError = "Inventory process is unavailable. Check the collector executable and refresh."
    })
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
