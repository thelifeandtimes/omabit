function validId(value) {
  return typeof value === "string" && /^[A-Za-z0-9_-]{1,64}$/.test(value)
}

function validateSetupStatus(v) {
  if (!v || typeof v.ready !== "boolean" || typeof v.can_setup !== "boolean" || !Array.isArray(v.checks) || v.checks.length > 64)
    throw new Error("Invalid local setup status")
  if (!v.checks.every(function(c) { return c && typeof c.id === "string" && typeof c.detail === "string" && c.detail.length <= 8192 && ["pass","missing","blocked","unknown","info"].indexOf(c.status) >= 0 }))
    throw new Error("Invalid prerequisite checks")
  if (v.ready && (!v.connection || !validId(v.connection.name) || !validId(v.connection.runner_id) || v.connection.mode !== "development-managed" || v.browser_base !== "http://127.0.0.1"))
    throw new Error("Invalid local runner identity")
  return v
}
function setupStepIndex(step) {
  var groups = {check:0,key:1,download:2,authorize:3,engine:3,access:3,services:3,verify:3,host:3,connect:4,ready:5}
  return groups[step] === undefined ? -1 : groups[step]
}
function setupStepLabel(step) {
  return {check:"Checking this computer",key:"Preparing desktop access",download:"Verifying host helper",authorize:"Waiting for OS authorization",engine:"Preparing the dedicated engine",access:"Configuring private SSH access",services:"Installing background services",verify:"Checking runner readiness",host:"Host setup complete",connect:"Connecting your desktop",ready:"Ready to add a ship"}[step] || "Local setup"
}
function setupEvent(text) {
  if (text.length > 65536) throw new Error("Setup response exceeds limit")
  var v = JSON.parse(text)
  if (!v || (setupStepIndex(v.step) < 0 && v.step !== "setup") || ["running","complete","failed"].indexOf(v.status) < 0 || (v.message !== undefined && (typeof v.message !== "string" || v.message.length > 16384))) throw new Error("Invalid setup progress")
  if (v.result) validateSetupStatus(v.result)
  return v
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
  var previous = rows.filter(function(row) { return row.name === next.name && row.runner_id === next.runner_id })[0]
  if (previous && next.status !== "available") next.last_snapshot = previous.snapshot || previous.last_snapshot
  return rows.filter(function(row) { return row.name !== next.name }).concat([next])
    .sort(function(a, b) { return a.name.localeCompare(b.name) })
}

function instances(connection) { return ((connection && (connection.snapshot || connection.last_snapshot)) || {}).instances || [] }

function fakeIdentity(value) { value=String(value || "").trim(); return value[0] === "~" ? value : "~"+value }
function ownedPierPath(ship) {
  ship=fakeIdentity(ship)
  if (!/^~(?:[a-z]{3}|[a-z]{6}(?:-[a-z]{6}){0,3})$/.test(ship)) return ""
  var folder={Galaxies:"galaxies",Stars:"stars",Planets:"planets",Moons:"moons"}[shipType({ship:ship})]
  return folder ? "livenet/"+folder+"/"+ship.slice(1)+"/pier" : ""
}
function optionalPort(value) {return String(value).trim()==="" || /^\d+$/.test(String(value)) && Number(value)>=1024 && Number(value)<=65535}
function bootstrapProblem(kind, source) {
  if (kind === "bare") return ""
  if (kind !== "recommended") return "Choose bare bootstrap or Recommended Apps."
  if (!String(source || "").trim()) return "The Recommended Apps pill is not configured yet. Set its direct release download URL in Omabit's plugin settings."
  if (!/^https:\/\/[^\s/@?#]+(?:[/?][^\s]*)?$/.test(String(source))) return "Recommended Apps requires a direct HTTPS pill download URL."
  return ""
}
function bootstrapArgs(kind, source) {
  var problem=bootstrapProblem(kind,source)
  if (problem) throw new Error(problem)
  return kind === "bare" ? ["--bootstrap","bare"] : ["--bootstrap","recommended","--pill",String(source).trim()]
}
function fakeOptionsProblem(connection, ship, pier) {
  if (!hasFeature(connection,"fake-ship-options")) return "Update this runner to choose a fake identity and pier directory."
  if (fakeIdentity(ship).length>28) return "The official runtime creates fake galaxies, stars, planets or moons; fake comet creation is unsupported."
  if (!/^~(?:[a-z]{3}|[a-z]{6}(?:-{1,2}[a-z]{6}){0,7})$/.test(fakeIdentity(ship))) return "Enter a fake Urbit identity such as ~bus or ~sampel-palnet."
  if (!validPierPath(pierPath("fake","",ship,pier))) return "Use a ship folder name or fakeships/project/ship/pier path with letters, digits, underscores and hyphens."
  return ""
}
function validPierPath(value) {
  value=String(value || "")
  if (value[0]==="@") {
    var root=value.slice(1).split("/")[0]
    if (!/^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$/.test(root) || root==="default") return false
    value=value.slice(root.length+2)
  }
  var p=value.split("/")
  return String(value).length<=384 && p.length<=8 && p.every(function(s){return /^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$/.test(s)})
    && p[p.length-1]==="pier" && (p[0]==="fakeships" && p.length>=3 || p[0]==="livenet" && p.length>=4 && ["galaxies","stars","planets","moons","comets"].indexOf(p[1])>=0)
}
function storageRoots(connection) {
  var c=connection && connection.snapshot && connection.snapshot.piers
  return c && c.roots && c.roots.length ? c.roots : [{id:"default",path:c && c.root || "/var/lib/urbit-runner/piers"}]
}
function storagePath(root,path) { return path[0]==="@" || !root || root==="default" ? path : "@"+root+"/"+path }
function absolutePierPath(connection,path) {
  var parts=String(path || "").split("/"), id=parts[0][0]==="@" ? parts.shift().slice(1) : "default"
  var root=storageRoots(connection).filter(function(r){return r.id===id})[0]
  return root ? root.path+"/"+parts.join("/") : path
}
function hostPath(value) { return typeof value==="string" && value.length>1 && value.length<=2048 && value[0]==="/" && !/[\u0000\r\n,\"]/.test(value) && value.split("/").slice(1).every(function(s){return s && s!=="." && s!==".."}) }
function canDeletePier(connection,instance,now) {
  return writable(connection,now) && connection.access_role==="operator" && hasFeature(connection,"delete-pier") && !!instance && validPierPath(instance.pier_path) && !handoffFenced(instance) && (!instance.external_import || ["complete","enrolling"].indexOf(instance.external_import.stage)>=0 || instance.external_import.stage==="reviewed" && !instance.external_import.holding_path) && !instance.active_operation_id && instance.desired_state==="stopped" && (instance.delete_pending || ["stopped","uninitialized"].indexOf(instance.observed_state)>=0)
}
function externalReviewCurrent(connection,review,path,now) {
  return writable(connection,now) && connection.access_role==="operator" && hasFeature(connection,"import-pier") && !!review && review.runner===connection.runner_id && review.connection===connection.name && review.source.path===path && validDigest(review.source.fingerprint) && now-review.loadedAt>=0 && now-review.loadedAt<300000
}
function pierPath(kind,name,ship,input) {
  input=String(input || "").trim()
  if (input.indexOf("/")>=0) return input
  return (kind==="fake" ? "fakeships/" : "livenet/comets/")+(input || (kind==="fake" ? fakeIdentity(ship).slice(1) : name))+"/pier"
}
function discoveredPiers(connections) {
  var out=[]
  connections.forEach(function(c){var catalogue=c.snapshot && c.snapshot.piers;if(c.status==="available" && catalogue && !catalogue.error && Array.isArray(catalogue.candidates))catalogue.candidates.forEach(function(p){if(!p.enrolled_by || p.moved)out.push({connection:c,source:p})})})
  return out
}
function currentPierSource(connection,source,now) {
  var catalogue=connection && connection.snapshot && connection.snapshot.piers
  return !!source && validPierPath(source.path) && validDigest(source.fingerprint) && available(connection,now) && hasFeature(connection,"pier-storage") && !!catalogue && catalogue.runner_id===connection.runner_id && !catalogue.error && fresh(catalogue.observed_at,now,30000)
    && Array.isArray(catalogue.candidates) && catalogue.candidates.some(function(p){return p.path===source.path && p.fingerprint===source.fingerprint && !p.busy && !p.problem && p.enrolled_by===source.enrolled_by && p.moved===source.moved})
}
function httpPortOwner(connection, port) {
  return instances(connection).filter(function(i) {
    var observed = String(i.endpoint || "").match(/:([0-9]+)\/?$/)
    return i.config.http_port === Number(port) || observed && Number(observed[1]) === Number(port)
  })[0] || null
}
function registrationProblem(connection, name, port, kind, now) {
  if (!writable(connection,now) || connection.access_role !== "operator" || !hasFeature(connection,kind === "comet" ? "create-comet" : "create-fake")) return "Select an available operator runner."
  if (!validId(name)) return "Use 1–64 letters, numbers, hyphens or underscores for the instance name."
  if (instances(connection).some(function(i){return i.id === name})) return "Instance “"+name+"” already exists on this runner. Choose another name or open it in the fleet."
  if (!Number.isInteger(Number(port)) || Number(port)<1024 || Number(port)>65535) return "Choose a host HTTP port from 1024 to 65535."
  var owner=httpPortOwner(connection,port)
  return owner ? "HTTP port "+port+" is already reserved by “"+owner.id+"”." : ""
}
function suggestInstance(connection, kind) {
  var base=kind === "comet" ? "comet" : "development", name=base, n=2
  while(instances(connection).some(function(i){return i.id === name})) name=base+"-"+(n++)
  return name
}
function suggestHTTPPort(connection) {
  for(var port=18080;port<=65535;port++) if(!httpPortOwner(connection,port)) return String(port)
  return ""
}
function instanceTitle(instance) {
  if (!instance) return "Ship identity unavailable"
  return instance.mode === "fake" ? "[" + (instance.ship || "~zod") + "]" : instance.ship || "Identity pending"
}
function shipType(instance) {
  if (instance.mode === "fake") return "Fakeships"
  if (instance.mode === "comet") return "Comets"
  var name=String(instance.ship || "").replace(/^~/,"").replace(/-/g,"")
  if (!/^[a-z]+$/.test(name)) return "Unassigned"
  return name.length === 3 ? "Galaxies" : name.length === 6 ? "Stars" : name.length === 12 ? "Planets"
    : name.length > 12 && name.length <= 24 && name.length % 6 === 0 ? "Moons"
    : name.length > 24 && name.length <= 48 && name.length % 6 === 0 ? "Comets" : "Unassigned"
}
function shipGroups(connection) {
  return ["Galaxies","Stars","Planets","Moons","Comets","Fakeships","Unassigned"].map(function(type) {
    return {type:type,ships:instances(connection).filter(function(i){return shipType(i) === type}).slice().sort(function(a,b) {
      return Number(b.observed_state === "running")-Number(a.observed_state === "running")
        || String(a.ship || "").localeCompare(String(b.ship || "")) || a.id.localeCompare(b.id)
    })}
  }).filter(function(group){return group.ships.length > 0})
}
function registrationFailure(text) {
  var code=errorCode(text)
  return {message:errorMessage(text),uncertain:["instance_exists","port_conflict","pier_path_conflict","pier_changed_or_busy","pier_not_found","pier_already_enrolled","relink_conflict","invalid_state","revision_conflict","recovery_required","ship_already_enrolled","key_staging_failed","development_capacity","invalid_request","comet_mode_required","unauthorized_peer","operator_required","ssh_access_denied"].indexOf(code)<0}
}
function errorCode(text) { try { return JSON.parse(text).error.code || "" } catch (_) { return "" } }
function runnerPreference(preferences, connection) { return connection ? (preferences || {})[connection.runner_id] || {} : {} }
function runnerTitle(connection, preferences) { return runnerPreference(preferences, connection).label || connection.name }
function runnerAddress(connection, registrations, preferences) {
  var pref = runnerPreference(preferences, connection)
  var registration = (registrations || []).filter(function(r) { return r.name === connection.name && r.runner_id === connection.runner_id })[0]
  if (pref.location) return pref.location
  return !registration ? "Connection details loading" : registration.transport === "unix" ? "Local · Unix socket" : "SSH · " + registration.ssh.host
}

function shipStatus(connection, instance, now, observing) {
  if (!observing || !available(connection, now)) return {label: "Unknown", tone: "muted", detail: "Runner unavailable; showing last known ship."}
  if (handoffFenced(instance)) return {label:instance.handoff.state==="moved" ? "Moved" : instance.handoff.error ? "Move paused" : "Moving",tone:instance.handoff.error ? "warn" : "muted",detail:instance.handoff.error || instance.handoff.message || "Stopped host move; Start is fenced."}
  if (instance.observed_state === "missing") return {label:"Missing pier",tone:"warn",detail:"The registered directory is missing or replaced. Review discovered piers in Setup → Pier storage; storage will not be recreated automatically."}
  if (instance.recovery_required) return {label: "Recovery", tone: "bad", detail: runtimeDetail(instance)}
  if (instance.active_operation_id) return {label: runtimeStage(instance.runtime_status && instance.runtime_status.stage) || "Working", tone: "accent", detail: runtimeDetail(instance)}
  if (!fresh(instance.observed_at,now,30000)) return {label: "Stale", tone: "warn", detail: "The runner is reachable, but the runtime observation is stale."}
  if (instance.observed_state === "running") return instance.health === "healthy"
    ? {label: "Healthy", tone: "good", detail: "Running · healthy"}
    : {label: "Check", tone: "warn", detail: "Running · health " + (instance.health || "unknown")}
  if (instance.observed_state === "uninitialized") return {label: "Not booted", tone: "muted", detail: "Registered; first explicit Start initializes the ship."}
  if (instance.identity_pending) return {label:"Verify identity",tone:"muted",detail:"Start the enrolled pier to verify its expected identity. Mutating maintenance stays unavailable until verification succeeds."}
  if (instance.observed_state === "stopped") return {label: "Stopped", tone: "muted", detail: "Runtime stopped"}
  return {label: instance.observed_state || "Unknown", tone: "warn", detail: "Observed state: " + (instance.observed_state || "unknown")}
}

function fleet(connections) {
  var out = []
  connections.forEach(function(c) { instances(c).forEach(function(i) {
    out.push({key: c.runner_id + "/" + i.id, connection: c, instance: i})
  }) })
  return out
}

function browserBase(value) {
  value = String(value || "").trim().replace(/\/$/, "")
  if (!value) return ""
  if (!/^https?:\/\/(\[[0-9a-fA-F:]+\]|[A-Za-z0-9](?:[A-Za-z0-9.-]*[A-Za-z0-9])?)$/.test(value))
    throw new Error("Use http://host or https://host without a port, path, or credentials.")
  return value
}

function connectionEditArgs(target, name, transport, address, config) {
  if (!target || !validId(target.name) || !validId(target.runner_id) || !Number.isSafeInteger(target.revision) || target.revision < 1 || !validId(name))
    throw new Error("Select a current enrollment and a valid connection name.")
  if (transport !== "ssh" && transport !== "unix") throw new Error("Select SSH or local socket.")
  var args = ["runner-edit",target.name,"--runner-id",target.runner_id,"--revision",String(target.revision),"--name",name,transport === "ssh" ? "--ssh" : "--socket",address]
  if (config && transport === "ssh") args = args.concat(["--ssh-config",config])
  return args
}

function canCopyCode(connection, instance, now) {
  return available(connection,now) && connection.access_role === "operator" && hasFeature(connection,"login-code")
    && !!instance && instance.observed_state === "running" && !instance.active_operation_id && !instance.recovery_required
    && fresh(instance.observed_at,now,30000) && !!instance.resources && /^[0-9a-f]{64}$/.test(instance.resources.container_id || "")
}
function codeUnavailable(connection,instance,now,observing) {
  if (!observing || !available(connection,now)) return "Connect to the runner for +code access."
  if (connection.access_role !== "operator") return "Operator access is required to copy +code."
  if (!hasFeature(connection,"login-code")) return "Update this runner to enable +code access."
  if (instance && instance.active_operation_id) return "Wait for the ship's current operation to finish."
  if (instance && instance.recovery_required) return "Review the recovery details and runtime logs before restarting or retrieving +code."
  if (!instance || instance.observed_state !== "running") return "Start the ship to retrieve its +code."
  return canCopyCode(connection,instance,now) ? "" : "Waiting for a fresh runtime identity."
}

function importAvailable(connections, target, now) {
  return !!target && connections.some(function(c) { return c.name === target.connection && c.runner_id === target.runner
    && available(c,now) && c.access_role === "operator" && hasFeature(c,"import-qualified-fake") })
}
function validImportSource(source) {
  return !!source && /^[0-9a-f]{32}$/.test(source.run_id || "") && validDigest(source.content_digest)
    && source.ship === "~zod" && Number.isInteger(source.files) && source.files > 0 && source.files <= 8192
    && byteCount(source.bytes) && source.bytes <= 100000000000 && isFinite(Date.parse(source.qualified_at))
}
function importTarget(catalogue, source, name, port, now) {
  if (!catalogue || now-catalogue.loadedAt > 300000 || now < catalogue.loadedAt-5000 || !validImportSource(source) || source.claimed_by
      || !validId(name) || !Number.isInteger(Number(port)) || Number(port)<1024 || Number(port)>65535)
    throw new Error("Select a fresh unclaimed source, a valid instance name and an HTTP port.")
  if (!catalogue.sources.some(function(s) { return s.run_id === source.run_id && s.content_digest === source.content_digest && !s.claimed_by }))
    throw new Error("The source does not belong to the selected catalogue.")
  return {connection:catalogue.connection,runner:catalogue.runner,loadedAt:catalogue.loadedAt,
    source:JSON.parse(JSON.stringify(source)),instance:name,port:Number(port)}
}
function importArgs(target,id) {
  return ["--runner",target.connection,"--expected-runner-id",target.runner,"import-fake",target.instance,
    "--source-run",target.source.run_id,"--expected-digest",target.source.content_digest,"--http-port",String(target.port),"--request-id",id,"--confirm-stopped"]
}

function shipURL(connection, instance, registrations, preferences) {
  if (!connection || !instance) return ""
  var url = instance.endpoint || ""
  if (!/^https?:\/\/(\[[0-9a-fA-F:]+\]|[A-Za-z0-9.-]+):[0-9]{1,5}\/?$/.test(url)) return ""
  var base = runnerPreference(preferences, connection).browserBase
  // Use the observed endpoint port: a staged configuration may not be applied yet.
  if (base) { try { return browserBase(base) + ":" + url.match(/:([0-9]+)\/?$/)[1] } catch (_) { return "" } }
  var registration = (registrations || []).filter(function(r) { return r.runner_id === connection.runner_id && r.name === connection.name })[0]
  // A remote runner's loopback URL addresses that host, not this browser.
  if (/^https?:\/\/(127\.[0-9.]+|localhost|\[::1\]):/.test(url) && (!registration || registration.transport !== "unix")) return ""
  return url
}

function activity(connections, runner, ship, filter, preferences) {
  var out = []
  connections.forEach(function(c) {
    if (runner && c.runner_id !== runner) return
    var snapshot = c.snapshot || c.last_snapshot
    if (!snapshot) return
    function subject(id) { return (snapshot.instances || []).filter(function(i){return i.id === id})[0] || {id:id} }
    ;(snapshot.operations || []).forEach(function(o) {
      if (ship && c.runner_id + "/" + o.instance_id !== ship) return
      if (filter === "incidents" || (filter === "attention" && ["failed", "interrupted"].indexOf(o.status) < 0)) return
      out.push({key: c.runner_id + "/" + o.id, runner: runnerTitle(c,preferences), instance: o.instance_id, subject:subject(o.instance_id),
        kind: o.kind, status: o.status, time: o.finished_at || o.created_at, message: o.message || "",
        tone: o.status === "succeeded" ? "good" : o.status === "skipped" ? "muted" : ["failed", "interrupted"].indexOf(o.status) >= 0 ? "bad" : "accent",
        detail: "Runner: " + c.runner_id + "\nOperation: " + o.id + "\nRequest: " + o.request_id + "\nActor: " + o.actor + "\nCreated: " + o.created_at + (o.finished_at ? "\nFinished: " + o.finished_at : "")})
    })
    ;(snapshot.incidents || []).forEach(function(i) {
      if (ship && c.runner_id + "/" + i.instance_id !== ship) return
      if (filter === "operations" || (filter === "attention" && !i.active)) return
      out.push({key: c.runner_id + "/" + i.id, runner: runnerTitle(c,preferences), instance: i.instance_id, subject:subject(i.instance_id),
        kind: i.code, status: i.active ? "incident" : "resolved", time: i.observed_at, message: i.message || "",
        tone: i.active ? "bad" : "muted", detail: "Runner: " + c.runner_id + "\nIncident: " + i.id})
    })
  })
  return out.sort(function(a, b) { return Date.parse(b.time) - Date.parse(a.time) || a.key.localeCompare(b.key) }).slice(0, 300)
}

function memorySample(instance, now) {
  var r = instance && instance.resources
  if (!r || r.source !== "docker-stats-v1.47" || r.status !== "fresh" || !fresh(r.observed_at, now, 30000) || !byteCount(r.memory_current_bytes)
      || !byteCount(r.memory_limit_bytes) || r.memory_limit_bytes <= 0 || !/^[0-9a-f]{64}$/.test(r.container_id || "")) return null
  return {time: Date.parse(r.observed_at), value: r.memory_current_bytes, limit: r.memory_limit_bytes, container: r.container_id}
}

function cpuSample(instance,now) {
  var r=instance && instance.resources
  if (!r || r.source !== "docker-stats-v1.47" || r.status !== "fresh" || !fresh(r.observed_at,now,30000)
      || typeof r.cpu_usage_percent !== "number" || !isFinite(r.cpu_usage_percent) || r.cpu_usage_percent<0
      || typeof r.cpu_limit_cores !== "number" || !isFinite(r.cpu_limit_cores) || r.cpu_limit_cores<=0
      || !/^[0-9a-f]{64}$/.test(r.container_id || "")) return null
  return {time:Date.parse(r.observed_at),value:r.cpu_usage_percent,limit:r.cpu_limit_cores*100,container:r.container_id}
}
function cpuView(connection,instance,now,observing) {
  var base=resourceView(connection,instance,now,observing)
  if (base.inactive) return {value:"N/A",note:base.note,sample:null,inactive:true}
  var sample=observing && available(connection,now) && instance && fresh(instance.observed_at,now,30000) ? cpuSample(instance,now) : null
  return {value:sample ? sample.value.toFixed(2)+"%" : "Unknown",sample:sample,inactive:false,
    note:sample ? "100% = one CPU · limit "+sample.limit+"%" : !hasFeature(connection,"container-cpu") ? "CPU metrics unavailable on this runner"
      : instance && instance.resources && instance.resources.status === "fresh" && fresh(instance.resources.observed_at,now,30000) && instance.resources.cpu_limit_cores ? "Waiting for two fresh CPU samples" : "No fresh CPU sample"}
}

function recordSamples(history, connection, now, metric) {
  var next = Object.assign({}, history)
  instances(connection).forEach(function(i) {
    var key = connection.runner_id + "/" + i.id, points = next[key] || []
    var last = points.filter(function(p) { return p !== null }).slice(-1)[0]
    var currentContainer=i.resources && i.resources.container_id
    if (last && currentContainer && currentContainer !== last.container) { points=[]; next[key]=[]; last=null }
    var sample = available(connection, now) && i.observed_state === "running" ? (metric === "cpu" ? cpuSample(i,now) : memorySample(i, now)) : null
    if (!sample) { if (points.length && points[points.length - 1] !== null) next[key] = points.concat([null]).slice(-120); return }
    if (last && last.container !== sample.container) points = []
    else if (last && sample.time <= last.time) return
    next[key] = points.concat([sample]).slice(-120)
  })
  Object.keys(next).slice(0,Math.max(0,Object.keys(next).length-256)).forEach(function(key) { delete next[key] })
  return next
}

function chartRange(samples,metric) {
  var points=samples.filter(function(p){return p !== null}), unit=metric === "cpu" ? 1 : 1048576
  var target=Math.max(1,points.reduce(function(n,p){return Math.max(n,p.value/unit)},0)*1.15)
  var power=Math.pow(10,Math.floor(Math.log10(target))), fraction=target/power
  var rounded=fraction<=1 ? 1 : fraction<=2 ? 2 : fraction<=2.5 ? 2.5 : fraction<=5 ? 5 : 10
  var end=points.length ? points[points.length-1].time : null
  return {maximum:rounded*power*unit,start:points.length ? Math.min(points[0].time,end-10000) : null,end:end}
}
function chartValue(value,metric) {
  if (metric === "cpu") return value.toFixed(value>0 && value<10 ? 1 : 0)+"%"
  var unit=value>=1073741824 ? 1073741824 : 1048576, number=value/unit
  return number.toFixed(number>0 && number<10 ? 1 : 0)+(unit===1048576 ? " MiB" : " GiB")
}
function chartTime(value) {
  if (value === null) return "—"
  var d=new Date(value)
  return [d.getHours(),d.getMinutes(),d.getSeconds()].map(function(n){return String(n).padStart(2,"0")}).join(":")
}

function epochDisplay(instance, now) {
  if (instance && instance.epochs) return {entries: instance.epochs.entries,
    label: fresh(instance.epochs.observed_at, now, 300000) ? "Last inspection" : "Inspection expired", time: instance.epochs.observed_at, measured: true}
  var p = instance && instance.pruning, roll=lastRoll(instance)
  function retained(names) { return (names || []).map(function(name,index){return {name:name,protected:index >= names.length-2}}) }
  if (roll && roll.finished_at && (!p || !p.finished_at || Date.parse(roll.finished_at)>Date.parse(p.finished_at)))
    return {entries:retained(roll.epochs_after),label:"Recorded at last roll",time:roll.finished_at,measured:false}
  if (p && p.finished_at && !p.uncertain) return {entries: retained(p.retained_epochs),
    label: "Retained at last prune", time: p.finished_at, measured: false}
  return {entries: [], label: "Not inspected", time: "", measured: false}
}

function lastRoll(instance) {
  return instance && (instance.last_roll || instance.maintenance && instance.maintenance.kind === "roll" && instance.maintenance.finished_at && instance.maintenance) || null
}
function epochDate(instance,name,now) {
  var evidence=instance && (instance.epoch_history || []).filter(function(e){return e.name === name})[0], roll=lastRoll(instance)
  var time=evidence ? evidence.rolled_at : roll && roll.created_epoch === name ? roll.finished_at : ""
  return time ? "Rolled "+humanTime(time,now) : "Creation time not recorded"
}
function epochBoundary(name) {
  return name === "0i0" ? "Initial event history" : "Event boundary "+String(name).slice(2).replace(/\B(?=(\d{3})+(?!\d))/g,",")
}
function resourceView(connection,instance,now,observing) {
  var current=observing && available(connection,now) && instance && fresh(instance.observed_at,now,30000)
  if (current && !instance.active_operation_id && !instance.recovery_required && instance.desired_state === "stopped" && ["stopped","uninitialized"].indexOf(instance.observed_state)>=0)
    return {value:"N/A",note:instance.observed_state === "uninitialized" ? "Ship has not booted" : "Container stopped",sample:null,inactive:true}
  var sample=current ? memorySample(instance,now) : null
  return {value:sample ? bytesText(sample.value) : "Unknown",note:sample ? "of "+bytesText(sample.limit) : "No fresh sample",sample:sample,inactive:false}
}
function inspectionStatus(connections,target,now) {
  if (!target) return null
  var c=connections.filter(function(c){return c.name === target.connection && c.runner_id === target.runner})[0]
  if (!available(c,now)) return target
  var op=(c.snapshot.operations || []).filter(function(o){return o.instance_id === target.instance && o.kind === "inspect-epochs" && o.request_id === target.request && o.config_revision === target.revision && (!target.operation || o.id === target.operation)})[0]
  if (!op) return target
  var next=Object.assign({},target,{operation:op.id}), i=instances(c).filter(function(i){return i.id === target.instance})[0]
  if (op.status === "queued" || op.status === "running") {
    next.status=op.status; next.message="Inspecting the stopped pier's event history…"
  } else if (op.status === "succeeded") {
    if (i && i.config_revision === target.revision && canReviewEpochs(c,i,now) && Date.parse(i.epochs.observed_at)>=Date.parse(op.created_at)) {
      next.status="ready"; next.message="Inspection complete · "+i.epochs.entries.length+" retained epochs. No history was changed."
    } else { next.status="unavailable"; next.message="Inspection finished, but the ship state or inventory changed. Stop and inspect again for a fresh review." }
  } else { next.status="failed"; next.message=op.message || "Inspection did not complete. See ship activity for details." }
  return next
}

function shortTime(value) {
  var d = new Date(value)
  if (!isFinite(d.getTime())) return "Unknown time"
  function pad(n) { return (n < 10 ? "0" : "") + n }
  return d.getFullYear() + "-" + pad(d.getMonth()+1) + "-" + pad(d.getDate()) + "  " + pad(d.getHours()) + ":" + pad(d.getMinutes())
}

function humanTime(value, now) {
  var d=new Date(value), today=new Date(now), previous=new Date(now)
  if (!isFinite(d.getTime())) return "Not recorded"
  previous.setDate(previous.getDate()-1)
  var clock=(d.getHours()<10 ? "0" : "")+d.getHours()+":"+(d.getMinutes()<10 ? "0" : "")+d.getMinutes()
  if (d.toDateString() === today.toDateString()) return "Today at "+clock
  if (d.toDateString() === previous.toDateString()) return "Yesterday at "+clock
  return ["Jan","Feb","Mar","Apr","May","Jun","Jul","Aug","Sep","Oct","Nov","Dec"][d.getMonth()]+" "+d.getDate()
    +(d.getFullYear() !== today.getFullYear() ? ", "+d.getFullYear() : "")+" at "+clock
}

function operationTitle(kind) {
  return {"inspect-epochs":"Inspect epochs","prune-epochs":"Prune old epochs","roll":"Roll epoch","pack":"Pack","meld":"Meld",
    "start":"Start","stop":"Stop","startup-resume":"Resume after host startup","register-fake":"Register fake ship","register-comet":"Register comet",
    "configure-routine":"Update maintenance policy","routine-pack":"Scheduled pack","routine-meld":"Scheduled meld","login-code":"Get +code"}[kind] || String(kind || "Event").replace(/-/g," ")
}

function runtimeStage(stage) {
  return {"preparing-runtime":"Preparing runtime","preparing-pill":"Preparing boot pill","starting-runtime":"Starting runtime","waiting-for-identity":"Booting · checking identity","moving-pier":"Moving stopped pier","stopping":"Stopping runtime","stopped":"Stopped","ready":"Ready"}[stage] || ""
}
function runtimeDetail(instance) {
  if (!instance) return "Select a ship."
  var d=instance.runtime_status || {}, op=d.last_operation, parts=[]
  if (instance.active_operation_id) parts.push(runtimeStage(d.stage) || "Runner operation in progress")
  var exited=d.running===false && d.exit_code!==undefined
  if (exited) parts.push("Current runtime: container exited with code "+d.exit_code+" · OOM killed: "+(d.oom_killed===true ? "yes" : d.oom_killed===false ? "no" : "unknown"))
  if (op) {
    parts.push("Last operation: "+operationTitle(op.kind)+" · "+op.status)
    if (op.message) parts.push(op.message)
  }
  var laterExit=exited && d.exit_code!==0 && op && op.status==="succeeded" && ["start","startup-resume"].indexOf(op.kind)>=0
  if (laterExit) parts.push("That start completed earlier; the container exited afterward.")
  if (d.launch_error) parts.push(d.launch_error)
  if (d.snapshot_minutes) parts.push("Container snapshot interval: "+d.snapshot_minutes+" minutes."+(d.default_snapshot_minutes && d.snapshot_minutes!==d.default_snapshot_minutes ? " New starts use "+d.default_snapshot_minutes+" minutes." : ""))
  else if (d.default_snapshot_minutes) parts.push("New runtime snapshot interval: "+d.default_snapshot_minutes+" minutes.")
  if (instance.recovery_required) parts.push(laterExit && d.initialized && instance.observed_state==="failed"
    ? "Recovery needs review. After reviewing the logs, use Stop to reconcile the exited container without deleting its pier, then Start when recovery clears."
    : "Recovery needs review. Check the operation and runtime logs. Stop reconciles retained containers; interrupted provisioning or maintenance may need further review.")
  return parts.join("\n") || "No lifecycle operation recorded yet."
}
function runtimeNarration(lines) {
  var out=[], previous=""
  ;(lines || []).forEach(function(raw) {
    var line=raw.replace(/^\d{4}-\d\d-\d\dT\S+\s+/,"")
    if (!/^[a-z0-9%]/.test(line)) return
    var match=line.match(/^([a-z-]{2,12}):/),prefix=match ? match[1] : ""
    if (prefix && previous===prefix && out.length) out[out.length-1]=line
    else out.push(line)
    previous=prefix
  })
  return out.slice(-4).join("\n")
}
function validateRuntimeLogs(v,runner,instance,revision) {
  if (!v || v.runner_id!==runner || v.instance_id!==instance || v.config_revision!==revision || !Array.isArray(v.lines) || v.lines.length>200 || !isFinite(Date.parse(v.observed_at)) || ["docker","retained","none"].indexOf(v.source)<0 || typeof v.truncated!=="boolean") throw new Error("Log response does not match the selected ship.")
  var size=0
  if (!v.lines.every(function(line){if(typeof line!=="string")return false;size+=line.length+1;return line.length<=4096}) || size>65536) throw new Error("Log response exceeds the display limit.")
  return v
}

function available(connection, now) {
  return !!connection && connection.status === "available" && !!connection.snapshot
    && now - Date.parse(connection.checked_at) >= -5000 && now - Date.parse(connection.checked_at) < 15000
}

function writable(connection, now) {
  return available(connection, now) && connection.access_role !== "observer"
}

function canOperate(connection, instance, kind, now) {
  if (!writable(connection, now) || !instance || instance.active_operation_id || instance.delete_pending || handoffFenced(instance)) return false
  if (kind === "stop") return instance.observed_state === "running" || instance.recovery_required === true
  if (instance.recovery_required) return false
  if (kind === "start") return ["stopped", "uninitialized"].indexOf(instance.observed_state) >= 0
  if (instance.identity_pending && kind!=="inspect-epochs") return false
  if (instance.mode === "comet") {
    var cometCapability = {pack:"comet-compaction",meld:"comet-compaction",roll:"comet-epochs","inspect-epochs":"comet-epochs"}[kind]
    if (!cometCapability || !hasFeature(connection,cometCapability)) return false
  }
  var capability = {pack: "offline-pack", meld: "offline-meld", roll: "offline-roll", "inspect-epochs": "epoch-inventory"}[kind]
  return !!capability && connection.access_role === "operator" && hasFeature(connection, capability)
    && stopped(instance, now) && (kind !== "roll" || !instance.roll_requires_run)
}

function handoffFenced(instance) {return !!instance && !!instance.handoff && (instance.handoff.direction!=="in" || instance.handoff.state!=="complete")}
function canMoveHost(connection,instance,now) {
  return available(connection,now) && connection.access_role==="operator" && hasFeature(connection,"host-handoff")
    && !!instance && !instance.active_operation_id && instance.desired_state==="stopped"
    && (!!instance.handoff && instance.handoff.direction==="out" || !handoffFenced(instance) && !instance.recovery_required && stopped(instance,now) && !!instance.runtime_status && instance.runtime_status.initialized===true)
}
function hostMoveProblem(connections,target,destination,name,httpPort,amesPort,now) {
  if (!target || !destination) return "Choose a destination runner."
  var source=connections.filter(function(c){return c.name===target.connection&&c.runner_id===target.runner})[0]
  var ship=instances(source).filter(function(i){return i.id===target.instance&&i.config_revision===target.revision})[0]
  var dest=connections.filter(function(c){return c.name===destination.connection&&c.runner_id===destination.runner})[0]
  if (!canMoveHost(source,ship,now)) return "Source changed or is unavailable. Stop it cleanly and review a fresh selection."
  if (!dest || !available(dest,now) || dest.access_role!=="operator" || !hasFeature(dest,"host-handoff") || source.runner_id===dest.runner_id) return "Choose another available operator runner with host-move support."
  if (!validId(name) || !validPierPath("fakeships/"+name+"/pier") || !optionalPort(httpPort) || !optionalPort(amesPort)) return "Choose an alphanumeric-leading destination name and optional ports from 1024 to 65535."
  var h=ship.handoff
  if (h&&h.direction==="out" && (h.destination_runner!==dest.runner_id || h.destination_instance!==name)) return "Continue this ship's already selected move. Its destination is fixed."
  var existing=instances(dest).filter(function(i){return i.id===name || ship.mode!=="fake"&&i.ship===ship.ship})[0]
  if (existing && (!h || !existing.handoff || existing.handoff.id!==h.id)) return "The destination already has this instance or ship identity."
  return ""
}
function hostMoveArgs(target,destination,name,httpPort,amesPort,request) {
  var args=["--runner",target.connection,"--expected-runner-id",target.runner,"move-host",target.instance,"--revision",String(target.revision),"--to-runner",destination.connection,"--to-runner-id",destination.runner,"--to-instance",name,"--request-id",request]
  if (String(httpPort).trim()) args=args.concat(["--http-port",String(httpPort)])
  if (String(amesPort).trim()) args=args.concat(["--ames-port",String(amesPort)])
  return args
}

function routineTarget(connection,instance) {
  if (!connection || !instance) return null
  return {connection:connection.name,runner:connection.runner_id,instance:instance.id,
    instanceRevision:instance.config_revision,revision:instance.schedule ? instance.schedule.revision : 0}
}
function routineTiming(policy) {
  if (!policy || !policy.config) return "No policy saved"
  var c=policy.config, names=["Sun","Mon","Tue","Wed","Thu","Fri","Sat"]
  return c.weekdays.map(function(d){return names[d]}).join(", ")+" at "+String(c.hour_utc).padStart(2,"0")+":"+String(c.minute_utc).padStart(2,"0")+" UTC"
}
function routineBehavior(policy) {
  var c=policy.config
  return (c.stop_running ? "May stop a running ship" : "Runs only when already stopped")
    +" · "+(c.resume ? "resumes it after success" : "leaves it stopped")+" · "+c.window_minutes+" minute dispatch window"
}
function sameRoutineConfig(a,b) {
  return !!a && !!b && ["enabled","action","hour_utc","minute_utc","window_minutes","stop_running","resume"].every(function(k){return a[k] === b[k]})
    && Array.isArray(a.weekdays) && Array.isArray(b.weekdays) && a.weekdays.slice().sort().join(",") === b.weekdays.slice().sort().join(",")
}
function currentRoutineTarget(connections,target,now) {
  return !!target && connections.some(function(c) { return c.name === target.connection && c.runner_id === target.runner
    && available(c,now) && c.access_role === "operator" && hasFeature(c,"maintenance-schedules")
    && c.snapshot.instances.some(function(i) { return i.id === target.instance && (i.mode !== "comet" || hasFeature(c,"comet-compaction")) && !i.active_operation_id
      && i.config_revision === target.instanceRevision && (i.schedule ? i.schedule.revision : 0) === target.revision }) })
}
function routineArgs(target,c) {
  if (!target || ["pack","meld"].indexOf(c.action)<0 || !Array.isArray(c.weekdays) || c.weekdays.length<1 || c.weekdays.length>7
      || c.weekdays.some(function(d,index) { return !Number.isInteger(d) || d<0 || d>6 || c.weekdays.indexOf(d)!==index })
      || !Number.isInteger(c.hour_utc) || c.hour_utc<0 || c.hour_utc>23 || !Number.isInteger(c.minute_utc) || c.minute_utc<0 || c.minute_utc>59
      || !Number.isInteger(c.window_minutes) || c.window_minutes<5 || c.window_minutes>180
      || typeof c.enabled !== "boolean" || typeof c.stop_running !== "boolean" || typeof c.resume !== "boolean" || c.resume && !c.stop_running)
    throw new Error("Choose pack/meld, UTC weekdays/time, a 5–180 minute window and explicit downtime/resume settings.")
  return ["--runner",target.connection,"--expected-runner-id",target.runner,"routine-set",target.instance,
    "--revision",String(target.revision),"--instance-revision",String(target.instanceRevision),"--enabled="+c.enabled,
    "--action",c.action,"--days",c.weekdays.join(","),"--hour-utc",String(c.hour_utc),"--minute-utc",String(c.minute_utc),
    "--window-minutes",String(c.window_minutes),"--stop-running="+c.stop_running,"--resume="+c.resume]
}

function hasFeature(connection, feature) {
  return !!connection && Array.isArray(connection.features) && connection.features.indexOf(feature) >= 0
}

function maintenanceTarget(connection, instance, kind, now) {
  if (["pack", "meld", "roll"].indexOf(kind) < 0 || !canOperate(connection, instance, kind, now))
    throw new Error("Select an available, clean stopped pier for maintenance.")
  return {connection: connection.name, runner: connection.runner_id, instance: instance.id,
    revision: instance.config_revision, kind: kind}
}

function currentMaintenanceTarget(connections, target, now) {
  if (!target || ["pack", "meld", "roll"].indexOf(target.kind) < 0) return false
  return connections.some(function(c) {
    return c.name === target.connection && c.runner_id === target.runner && available(c, now)
      && c.snapshot.instances.some(function(i) {
        return i.id === target.instance && i.config_revision === target.revision && canOperate(c, i, target.kind, now)
      })
  })
}

function maintenanceArgs(target, id) {
  return ["--runner", target.connection, "--expected-runner-id", target.runner,
    "operate", target.instance, target.kind, "--revision", String(target.revision), "--request-id", id]
}

function fresh(time, now, age) {
  var observed = Date.parse(time)
  return typeof time === "string" && isFinite(observed) && now - observed >= -5000 && now - observed <= age
}

function stopped(instance, now) {
  return !!instance && instance.observed_state === "stopped" && instance.desired_state === "stopped"
    && !instance.active_operation_id && !instance.recovery_required && fresh(instance.observed_at, now, 30000)
}

// Preserve full uint64 event names without relying on QML BigInt support.
function validEpoch(name) {
  if (typeof name !== "string" || !/^0i(0|[1-9][0-9]{0,19})$/.test(name)) return false
  var digits = name.slice(2)
  return digits.length < 20 || digits <= "18446744073709551615"
}

function epochLess(a, b) {
  var x = a.slice(2), y = b.slice(2)
  return x.length !== y.length ? x.length < y.length : x < y
}

function validDigest(value) { return typeof value === "string" && /^[0-9a-f]{64}$/.test(value) }
function byteCount(value) { return Number.isSafeInteger(value) && value >= 0 }

function validInventory(inv, now) {
  if (!inv || inv.source !== "vere-4.6-epoch-v2" || !/^[0-9a-f]{32}$/.test(inv.pier_id || "")
      || !validDigest(inv.digest) || !fresh(inv.observed_at, now, 300000)
      || !Array.isArray(inv.entries) || inv.entries.length < 1 || inv.entries.length > 128) return false
  return inv.entries.every(function(e, i) {
    return e && validEpoch(e.name) && (i === 0 || epochLess(inv.entries[i - 1].name, e.name))
      && byteCount(e.logical_bytes) && byteCount(e.allocated_bytes)
      && Number.isInteger(e.file_count) && e.file_count >= 1 && e.file_count <= 32
      && (e.snapshot_bytes === undefined || byteCount(e.snapshot_bytes))
      && typeof e.shared_files === "boolean" && e.protected === (i >= inv.entries.length - 2)
  })
}

function canReviewEpochs(connection, instance, now) {
  return available(connection, now) && !!instance && (instance.mode !== "comet" || hasFeature(connection,"comet-epochs")) && hasFeature(connection, "epoch-inventory")
    && stopped(instance, now) && validInventory(instance.epochs, now)
}

function epochTarget(connection, instance, now) {
  if (!canReviewEpochs(connection, instance, now)) throw new Error("Inspect a fresh, clean stopped pier before reviewing retention.")
  return {connection: connection.name, runner: connection.runner_id, instance: instance.id,
    revision: instance.config_revision, role: connection.access_role, mode: instance.mode || "fake",
    inventory: JSON.parse(JSON.stringify(instance.epochs)), cutoff: ""}
}

function selectedEpochTarget(target, cutoff) {
  if (!target || !validEpoch(cutoff)) throw new Error("Select a listed, unprotected epoch.")
  var found = target.inventory.entries.filter(function(e) { return e.name === cutoff && !e.protected })
  if (found.length !== 1) throw new Error("The selected cutoff includes a protected or absent epoch.")
  var copy = JSON.parse(JSON.stringify(target)); copy.cutoff = cutoff
  return copy
}

function currentEpochTarget(connections, target, now) {
  if (!target) return false
  return connections.some(function(c) {
    return c.name === target.connection && c.runner_id === target.runner && c.access_role === target.role
      && available(c, now) && c.snapshot.instances.some(function(i) {
        return i.id === target.instance && i.config_revision === target.revision && (i.mode || "fake") === target.mode
          && canReviewEpochs(c, i, now) && i.epochs.pier_id === target.inventory.pier_id
          && i.epochs.digest === target.inventory.digest && i.epochs.observed_at === target.inventory.observed_at
          && i.epochs.entries.length === target.inventory.entries.length
          && i.epochs.entries.every(function(e, index) { return sameEpochEntry(e, target.inventory.entries[index]) })
      })
  })
}

function epochPreviewArgs(target) {
  return ["--runner", target.connection, "--expected-runner-id", target.runner,
    "epoch-preview", target.instance, "--revision", String(target.revision),
    "--expected-digest", target.inventory.digest, "--cutoff", target.cutoff]
}

function sameEpochEntry(a, b) {
  return a && b && a.name === b.name && a.digest === b.digest && a.logical_bytes === b.logical_bytes
    && a.allocated_bytes === b.allocated_bytes && a.file_count === b.file_count
    && (a.snapshot_bytes || 0) === (b.snapshot_bytes || 0) && a.protected === b.protected && a.shared_files === b.shared_files
}

function validateEpochPreview(result, target) {
  if (!target || !result || result.runner_id !== target.runner || result.instance_id !== target.instance
      || result.config_revision !== target.revision || result.inventory_digest !== target.inventory.digest
      || result.observed_at !== target.inventory.observed_at || result.cutoff !== target.cutoff
      || typeof result.deletion_supported !== "boolean" || result.requires_stopped_revalidation !== true
      || !Array.isArray(result.selected) || !Array.isArray(result.retained)) throw new Error("Pruning preview identity or selection changed.")
  var entries = target.inventory.entries, index = entries.findIndex(function(e) { return e.name === target.cutoff })
  if (index < 0 || index >= entries.length - 2 || result.selected.length !== index + 1
      || result.retained.length !== entries.length - index - 1) throw new Error("Pruning preview violates the protected prefix policy.")
  var logical = 0, allocated = 0, shared = false
  for (var i = 0; i < entries.length; i++) {
    var actual = i <= index ? result.selected[i] : result.retained[i - index - 1]
    if (!sameEpochEntry(entries[i], actual)) throw new Error("Pruning preview differs from the selected inventory.")
    if (i <= index) { logical += entries[i].logical_bytes; allocated += entries[i].allocated_bytes; shared = shared || entries[i].shared_files }
  }
  if (!byteCount(logical) || !byteCount(allocated) || result.selected_logical_bytes !== logical
      || result.selected_allocated_bytes !== allocated || result.shared_files !== shared)
    throw new Error("Pruning preview size evidence is inconsistent.")
  return result
}

function canPrunePreview(connections, target, preview, now) {
  if (!target || target.role !== "operator" || !currentEpochTarget(connections, target, now)
      || !preview || preview.deletion_supported !== true) return false
  try { validateEpochPreview(preview, target) } catch (_) { return false }
  if (!target.inventory.entries.every(function(e) {
    return validDigest(e.digest) && !e.shared_files && e.file_count === (e.name === "0i0" ? 4 : 5)
      && (e.name === "0i0" ? !e.snapshot_bytes : e.snapshot_bytes > 0)
  })) return false
  return connections.some(function(c) {
    return c.name === target.connection && c.runner_id === target.runner
      && (target.mode === "comet" ? hasFeature(c,"comet-prune-prefix")
        : hasFeature(c, "epoch-prune-prefix") || (hasFeature(c, "epoch-prune-genesis")
          && target.cutoff === "0i0" && preview.selected.length === 1 && preview.retained.length === 3))
  })
}

function epochPruneArgs(target, id) {
  return ["--runner", target.connection, "--expected-runner-id", target.runner,
    "epoch-prune", target.instance, "--revision", String(target.revision), "--cutoff", target.cutoff,
    "--expected-digest", target.inventory.digest, "--request-id", id, "--confirm-history-loss"]
}

function bytesText(value) { return byteCount(value) ? (value / 1048576).toFixed(2) + " MiB" : "unknown" }

function resourceText(instance, now) {
  var r = instance.resources
  if (!r) return "Container resources: not sampled"
  if (r.source !== "docker-stats-v1.47" || !/^[0-9a-f]{64}$/.test(r.container_id || "")
      || ["fresh", "stale"].indexOf(r.status) < 0 || !byteCount(r.memory_current_bytes)
      || !byteCount(r.memory_limit_bytes) || !byteCount(r.pids_current) || !isFinite(Date.parse(r.observed_at)))
    return "Container resources: unavailable"
  var status = r.status === "fresh" && fresh(r.observed_at, now, 30000) ? "fresh" : "stale"
  return "Container RAM (cache included): " + bytesText(r.memory_current_bytes) + " / " + bytesText(r.memory_limit_bytes)
    + " · tasks: " + r.pids_current + " · " + status + "\nSample: " + r.observed_at + " · " + r.source
    + (typeof r.cpu_usage_percent === "number" && isFinite(r.cpu_usage_percent) ? "\nContainer CPU: "+r.cpu_usage_percent.toFixed(2)+"% (100% = one CPU)" : "")
    + "\nContainer: " + r.container_id
}

function maintenanceText(report) {
  if (!report) return ""
  return "Last " + report.kind + ": " + (report.finished_at || "unfinished")
    + "\nPier logical size: " + bytesText(report.before_pier_bytes) + " → " + (report.finished_at ? bytesText(report.after_pier_bytes) : "pending")
    + (report.created_epoch ? " · created " + report.created_epoch : "")
}

function pruningText(report) {
  if (!report) return ""
  return "Last prune through " + report.cutoff + ": " + (report.uncertain || !report.finished_at ? "uncertain / incomplete" : "completed " + report.finished_at)
    + "\nRemoved files: " + report.removed_files + "/" + report.expected_files
    + " · removed epochs: " + (report.removed_epochs || []).join(", ")
    + "\nRetained at completion: " + (report.retained_epochs || []).join(", ")
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
  if (!connection || !instance) return null
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
