import QtQuick
import Quickshell.Io
import "Model.js" as Model

Item {
  id: root
  property var service
  ReaderLifecycle { service: root.service; process: helper; kind: "project-history" }
  property var connection: null
  property bool active: false
  property var projects: []
  property var detail: null
  property string readAt: ""
  property string error: ""
  property string operation: ""
  property string output: ""
  property int generation: 0
  property var request: null
  property bool overflow: false
  property bool timedOut: false
  property int requestTimeoutMs: 15000
  readonly property bool busy: operation !== ""
  readonly property string commandBinding: service ? JSON.stringify(service.command([])) : ""
  readonly property string targetBinding: connection ? connection.name + "/" + connection.runner_id + "/" + connection.mode + "/" + commandBinding : ""
  readonly property bool available: !!service && service.activationEnabled && service.observing && Model.available(connection, service.now)
  readonly property bool canList: active && available && Model.hasFeature(connection, "project-reservations-v1")
  readonly property bool canReadStatus: canList && Model.hasFeature(connection, "project-saved-status-v1")

  onTargetBindingChanged: invalidate()
  onActiveChanged: if (!active) invalidate()
  onAvailableChanged: if (!available) invalidate()
  onCanListChanged: if (!canList) invalidate()
  onCanReadStatusChanged: if (!canReadStatus) clearDetail()

  function invalidate() {
    generation++; projects = []; detail = null; readAt = ""; error = ""
    if (helper.running) { helper.signal(15); hardStop.restart() }
  }
  function launch(kind, project) {
    if (busy || !canList || kind === "status" && !canReadStatus) return false
    var args = ["--runner", connection.name, "--expected-runner-id", connection.runner_id,
      kind === "list" ? "projects" : "project-status"]
    if (project) args.push(project.id)
    request = {generation:generation, binding:targetBinding, runner:connection.runner_id,
      mode:connection.mode, project:project ? JSON.parse(JSON.stringify(project)) : null}
    operation = kind; output = ""; error = ""; detail = null; overflow = false; timedOut = false
    helper.command = service.command(args)
    deadline.interval = requestTimeoutMs; deadline.restart(); helper.running = true
    return true
  }
  function refresh() { return launch("list", null) }
  function clearDetail() {
    detail = null
    if (operation === "status") { generation++; if (helper.running) { helper.signal(15); hardStop.restart() } }
  }
  function validProject(p) {
    return p && typeof p.id === "string" && /^[A-Za-z0-9_-]{1,64}$/.test(p.id) && typeof p.label === "string" && p.label.length > 0 && p.label.length <= 128 && p.revision === 1 && typeof p.review_digest === "string" && /^[0-9a-f]{64}$/.test(p.review_digest) && Array.isArray(p.members) && p.members.length <= 256 && p.members.every(function(m) { return m && typeof m.ship === "string" && typeof m.instance_id === "string" && typeof m.http_port === "number" })
  }
  function inspect(project) {
    if (!project || !projects.some(function(p) { return p.id === project.id && p.revision === project.revision && p.review_digest === project.review_digest })) return false
    return launch("status", project)
  }
  function finish(code) {
    deadline.stop(); hardStop.stop()
    var kind = operation, expected = request, text = output
    operation = ""; output = ""; request = null
    if (!expected || !active || !canList || expected.generation !== generation || expected.binding !== targetBinding) return
    if (code !== 0 || overflow || timedOut) {
      error = timedOut ? "The read timed out. Refresh to try again." : overflow ? "The saved response is too large for this view. Use the CLI for details." : "Could not read saved projects. Check the runner connection and refresh."
      return
    }
    var value
    try { value = JSON.parse(text) } catch (_) { error = "The collector returned an unreadable response."; return }
    if (!value || value.schema_version !== 1 || value.runner_id !== expected.runner || value.mode !== expected.mode || value.runtime_observation !== "not-performed" || !isFinite(Date.parse(value.read_at))) {
      error = "The saved response does not match this runner or a supported format."; return
    }
    if (kind === "list") {
      if (value.basis !== "stored-reservations" || !Array.isArray(value.projects) || !value.projects.every(validProject)) { error = "The collector did not return saved project reservations."; return }
      projects = value.projects; readAt = value.read_at
    } else {
      var p = value.project, want = expected.project
      if (value.basis !== "stored-operation-history" || !validProject(p) || p.id !== want.id || p.revision !== want.revision || p.review_digest !== want.review_digest || !Array.isArray(value.operations) || value.operations.length > 128 || typeof value.has_older_operations !== "boolean" || !value.operations.every(function(o) { return o && typeof o.kind === "string" && ["queued","running","succeeded","failed","interrupted"].indexOf(o.status) >= 0 && typeof o.created_at === "string" })) {
        error = "The project changed or its history could not be verified. Refresh projects."; return
      }
      detail = value
    }
  }
  Timer { id: deadline; onTriggered: { root.timedOut = true; if (helper.running) { helper.signal(15); hardStop.restart() } else root.finish(-1) } }
  Timer { id: hardStop; interval: 5000; onTriggered: if (helper.running) helper.signal(9) }
  Process {
    id: helper
    stdout: SplitParser {
      splitMarker: ""
      onRead: function(data) {
        if (root.output.length + data.length > 1048576) { root.overflow = true; helper.signal(9) }
        else root.output += data
      }
    }
    stderr: SplitParser { splitMarker: ""; onRead: function(data) {} }
    onExited: function(code, status) { root.finish(status === 0 ? code : -1) }
  }
}
