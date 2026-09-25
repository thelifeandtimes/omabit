function validId(value) {
  return typeof value === "string" && /^[A-Za-z0-9_-]{1,64}$/.test(value)
}

function observation(text) {
  if (text.length > 2 * 1024 * 1024) throw new Error("Collector observation exceeds limit")
  var v = JSON.parse(text)
  if (!v || !validId(v.name) || !validId(v.runner_id)
      || ["available", "unavailable", "identity_mismatch"].indexOf(v.status) < 0
      || !isFinite(Date.parse(v.checked_at))) throw new Error("Invalid collector observation")
  if (v.status === "available" && (!v.snapshot || v.snapshot.runner_id !== v.runner_id
      || v.snapshot.mode !== v.mode || !Array.isArray(v.snapshot.instances)
      || !Array.isArray(v.snapshot.operations) || !Array.isArray(v.snapshot.incidents)))
    throw new Error("Collector snapshot identity mismatch")
  if (v.snapshot && !v.snapshot.instances.every(function(i) {
    return i && validId(i.id) && Number.isSafeInteger(i.config_revision) && i.config_revision > 0
      && Number.isSafeInteger(i.applied_revision) && i.applied_revision >= 0 && i.config
      && Number.isInteger(i.config.http_port) && Number.isInteger(i.config.ames_port)
      && Number.isInteger(i.config.loom)
  })) throw new Error("Invalid instance configuration in collector observation")
  return v
}

function merge(rows, next) {
  return rows.filter(function(row) { return row.name !== next.name }).concat([next])
    .sort(function(a, b) { return a.name.localeCompare(b.name) })
}

function available(connection, now) {
  return !!connection && connection.status === "available" && !!connection.snapshot
    && now - Date.parse(connection.checked_at) >= -5000 && now - Date.parse(connection.checked_at) < 15000
}

function writable(connection, now) {
  return available(connection, now) && connection.access_role !== "observer"
}

function canOperate(connection, instance, kind, now) {
  if (!writable(connection, now) || !instance || instance.active_operation_id || instance.recovery_required) return false
  return kind === "start" ? ["stopped", "uninitialized"].indexOf(instance.observed_state) >= 0
    : kind === "stop" && instance.observed_state === "running"
}

function selection(connection) {
  return ["--runner", connection.name, "--expected-runner-id", connection.runner_id]
}

function requestId() {
  return "ui-" + Date.now() + "-" + Math.random().toString(16).slice(2)
}

function operate(connection, instance, kind, id) {
  return selection(connection).concat(["operate", instance.id, kind,
    "--revision", String(instance.config_revision), "--request-id", id])
}

function edit(connection, instance) {
  return { connection: connection.name, runner: connection.runner_id, instance: instance.id,
    revision: instance.config_revision, port: instance.config.http_port,
    ames: instance.config.ames_port, loom: instance.config.loom }
}

function configure(target, port) {
  if (!Number.isInteger(Number(port)) || Number(port) < 1024 || Number(port) > 65535)
    throw new Error("HTTP port must be 1024–65535")
  return ["--runner", target.connection, "--expected-runner-id", target.runner,
    "configure", target.instance, "--revision", String(target.revision),
    "--http-port", String(port), "--ames-port", String(target.ames), "--loom", String(target.loom)]
}

function localPath(url) {
  var value = String(url)
  return value.indexOf("file://") === 0 ? decodeURIComponent(value.slice(7)) : ""
}

function errorMessage(text) {
  try { var v = JSON.parse(text); if (v.error && typeof v.error.message === "string") return v.error.message }
  catch (_) {}
  return "Collector request failed; inspect a fresh snapshot before retrying."
}

function notices(previous, next) {
  if (!previous || previous.runner_id !== next.runner_id) return []
  var out = []
  if (previous.status === "available" && next.status !== "available")
    out.push({title: "Runner unavailable", body: next.name + ": connection unavailable; ship state is unknown."})
  if (previous.status !== "available" && next.status === "available")
    out.push({title: "Runner reachable", body: next.name + ": connection restored."})
  if (!previous.snapshot || !next.snapshot) return out
  var known = previous.snapshot.operations.map(function(op) { return op.id + ":" + op.status })
  next.snapshot.operations.forEach(function(op) {
    if (["failed", "interrupted"].indexOf(op.status) >= 0 && known.indexOf(op.id + ":" + op.status) < 0)
      out.push({title: "Runner operation " + op.status, body: next.name + "/" + op.instance_id + ": " + op.kind + ". Inspect the operation history."})
  })
  var incidents = previous.snapshot.incidents.filter(function(i) { return i.active }).map(function(i) { return i.id })
  next.snapshot.incidents.forEach(function(i) {
    if (i.active && incidents.indexOf(i.id) < 0)
      out.push({title: "Runner incident", body: next.name + ": " + i.code + ". Inspect the panel."})
  })
  return out.slice(0, 8)
}
