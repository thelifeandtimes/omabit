import QtQuick
import Quickshell.Io
import "Model.js" as Model
import "ShipRoutes.js" as Routes

Item {
  id: root
  property var service
  property var plugins: []
  property var ships: []
  property var attempts: []
  property bool fresh: false
  property string operation: ""
  readonly property bool busy: operation !== ""
  property string message: ""
  property bool failed: false
  property string lastRequestId: ""
  property int snapshotRevision: 0
  property string output: ""
  property string errorOutput: ""
  property string pendingInput: ""
  property string hostedCode: ""
  property bool overflow: false
  property bool timedOut: false
  property var stagedPlugins: []
  property var stagedShips: []
  property var stagedAttempts: []
  property var permissions: null
  property var review: null
  property var selectedFleet: null
  property string selectionKey: ""
  property string requestSelectionKey: ""
  property bool pendingReview: false
  property int generation: 0
  property int requestGeneration: -1
  readonly property string profileBinding: service ? JSON.stringify(service.command([])) : ""
  onProfileBindingChanged: { generation++;fresh=false;permissions=null;review=null }

  function ready() { return service && !service.busy && service.activationEnabled && !busy }
  function launch(args, kind, input) {
    operation = kind; output = ""; errorOutput = ""; overflow = false; timedOut = false
    requestGeneration=generation;requestSelectionKey=selectionKey
    pendingInput = input || ""
    helper.stdinEnabled = pendingInput !== ""
    helper.command = service.command(args)
    deadline.interval = kind === "read-review" ? 45000 : kind === "connect-fleet" ? 120000 : kind.indexOf("read-") === 0 ? 15000 : 90000
    deadline.restart(); helper.running = true
  }
  function refresh() {
    if (!ready()) return false
    fresh = false; review=null
    launch(["app-plugins"], "read-plugins", "")
    return true
  }
  function begin(args, kind, input) {
    if (!ready()) return false
    fresh = false; review=null; message = ""; failed = false
    launch(args, kind, input)
    return true
  }
  function connect(plugin, ship, approved) {
    if(ship && ship.managed && ship.managed.fleet)return connectFleet(plugin,ship)
    if (!approved || !fresh || !plugin || plugin.status !== "available" || !ship || (!ship.managed && ship.status !== "session_stored")) return false
    var target = plugin.target
    // An operation ID is a replay key, not a credential or an authorization.
    var request = ""
    for (var i=0; i<32; i++) request += Math.floor(Math.random()*16).toString(16)
    var args=ship.managed ? ["--runner",ship.managed.runner,"--expected-runner-id",ship.managed.runner_id,
      "app-managed-connect","--instance",ship.managed.instance,"--revision",String(ship.managed.revision),
      "--container-id",ship.managed.container_id,"--url",ship.origin,"--ship",ship.ship] :
      ["app-connect",ship.name,"--expected-ship-id",ship.id,"--expected-ship-revision",String(ship.revision)]
    args=args.concat(["--plugin-id",target.plugin_id,"--profile",target.profile,"--artifact-sha256",target.artifact_sha256,
      "--request-id",request,"--approve-owner-session"])
    var ok = begin(args, "connect", "")
    if (ok) lastRequestId = request
    return ok
  }
  function selectShip(ship,active) {
    var current=active && ship && ship.managed && ship.managed.fleet ? ship : null
    var key=Model.applicationShipKey(current)
    if(key===selectionKey)return
    selectedFleet=current;selectionKey=key;review=null;pendingReview=!!current
    Qt.callLater(ensureReview)
  }
  function ensureReview() {
    if(!pendingReview || !selectedFleet || !fresh || !ready())return
    pendingReview=false
    var m=selectedFleet.managed
    launch(["--runner",m.runner,"--expected-runner-id",m.runner_id,"app-ship-review","--instance",m.instance],"read-review","")
  }
  function validPermissions(v) {
    return v && v.version===1 && Routes.revision(v.revision) && Array.isArray(v.entries) && v.entries.length<=256 && v.entries.every(function(g){
      return g && g.binding && g.binding.target && Routes.identifier(g.binding.runner_id) && Routes.identifier(g.binding.target.instance_id) && /^[0-9a-f]{64}$/.test(g.binding.continuity||"") && typeof g.binding.ship==="string" && g.target && typeof g.target.plugin_id==="string" && typeof g.target.profile==="string" && /^[0-9a-f]{64}$/.test(g.target.artifact_sha256||"") && typeof g.allowed==="boolean" && Routes.revision(g.revision) && g.revision>0 && g.revision<=v.revision
    })
  }
  function grant(plugin) {
    if(!review || !plugin || !permissions)return null
    return permissions.entries.filter(function(g){return Routes.equal(g.binding,review.routes.record.binding) && Routes.equal(g.target,plugin.target)})[0]||null
  }
  function permit(plugin,ship,allowed) {
    if(!fresh || !review || !plugin || plugin.status!=="available" || Model.applicationShipKey(ship)!==selectionKey)return false
    var m=ship.managed,q={version:1,binding:Routes.copy(review.routes.record.binding),selection:Routes.copy(review.routes.selection),target:Routes.copy(plugin.target),expected_revision:permissions.revision,allowed:allowed}
    if(!allowed){var g=grant(plugin);return g ? revoke(g) : false}
    return begin(["--runner",m.runner,"--expected-runner-id",m.runner_id,"app-ship-approve","--request-json",JSON.stringify(q),"--allow-owner-session"],"permit","")
  }
  function revoke(g) {
    if(!fresh || !permissions || !g || !g.allowed || !permissions.entries.some(function(e){return Routes.equal(e,g)}))return false
    var q={version:1,binding:Routes.copy(g.binding),target:Routes.copy(g.target),expected_revision:permissions.revision,allowed:false}
    return begin(["app-ship-approve","--request-json",JSON.stringify(q)],"revoke","")
  }
  function connectFleet(plugin,ship) {
    var g=grant(plugin)
    if(!fresh || !review || !g || !g.allowed || plugin.status!=="available" || Model.applicationShipKey(ship)!==selectionKey)return false
    var id="";for(var i=0;i<32;i++)id+=Math.floor(Math.random()*16).toString(16)
    var m=ship.managed,q={version:1,binding:Routes.copy(g.binding),target:Routes.copy(g.target),grant_revision:g.revision}
    var ok=begin(["--runner",m.runner,"--expected-runner-id",m.runner_id,"app-fleet-connect","--request-json",JSON.stringify(q),"--request-id",id],"connect-fleet","")
    if(ok)lastRequestId=id
    return ok
  }
  function act(row, action) {
    if (!fresh || !row || !row.next_actions || row.next_actions.indexOf(action) < 0) return false
    return begin([action, row.request_id, "--expected-revision", String(row.revision)], action, "")
  }
  function enroll(name, origin, ship) {
    if (!name.trim() || !origin.trim() || !ship.trim()) return false
    return begin(["ship-enroll", name.trim(), "--url", origin.trim(), "--ship", ship.trim()], "enroll", "")
  }
  function enrollHosted(origin,ship,code) {
    ship=ship.trim(); if(ship.charAt(0)!=="~")ship="~"+ship
    if(!origin.trim()||!code||code.length>128)return false
    if(ships.some(function(s){return !s.managed && s.ship===ship})) {message="This hosted ship is already saved.";failed=true;return false}
    if(!begin(["ship-enroll",ship.slice(1),"--url",origin.trim(),"--ship",ship,"--unique-ship"],"enroll-hosted",""))return false
    hostedCode=code; return true
  }
  function login(ship, code) {
    if (!fresh || !ship || !code || code.length > 128) return false
    return begin(["ship-login", ship.name, "--expected-ship-id", ship.id, "--code-stdin"], "login", code)
  }
  function finish(exitCode) {
    deadline.stop(); hardStop.stop(); pendingInput = ""
    var kind = operation
    var value = null
    try { value = JSON.parse(output) } catch (_) {}
    var problem=null
    try { problem=JSON.parse(errorOutput).error } catch (_) {}
    errorOutput=""
    output = ""
    var ok = exitCode === 0 && !overflow && !timedOut
    if(requestGeneration!==generation){operation="";fresh=false;review=null;message="The selected profile changed. Refresh connection state before continuing.";return}
    if(kind === "read-review") {
      operation=""
      if(requestSelectionKey!==selectionKey || !selectedFleet){Qt.callLater(ensureReview);return}
      var m=selectedFleet.managed,r=value && value.routes
      if(!ok || !value || value.version!==1 || !Routes.valid(r,m.runner_id,m.mode,m.instance) || r.selection.config_revision!==m.revision || r.selection.runtime_id!==m.container_id || r.record.binding.ship!==selectedFleet.ship || !validPermissions(value.approvals)) {
        review=null;failed=true;message="Could not verify this ship's application permission. Refresh to review it again.";return
      }
      review=value;permissions=value.approvals
      return
    }
    if (kind.indexOf("read-") === 0) {
      if (!ok || (kind === "read-permissions" ? !validPermissions(value) : !Array.isArray(value))) {
        operation = ""; fresh = false; failed = true
        message = "Could not refresh application connections. No connection was approved."
        return
      }
      if (kind === "read-plugins") {
        stagedPlugins = value
        Qt.callLater(function() { launch(["ships"], "read-ships", "") })
      } else if (kind === "read-ships") {
        stagedShips = value
        Qt.callLater(function() { launch(["app-connections"], "read-connections", "") })
      } else if (kind === "read-connections") {
        stagedAttempts=value
        Qt.callLater(function(){launch(["app-ship-approvals"],"read-permissions","")})
      } else {
        plugins = stagedPlugins; ships = stagedShips; attempts = stagedAttempts;permissions=value
        snapshotRevision++; operation = ""; fresh = true
        pendingReview=!!selectedFleet;Qt.callLater(ensureReview)
      }
      return
    }
    if(kind==="enroll-hosted") {
      var secret=hostedCode; hostedCode=""
      if(ok && value && value.id) {
        Qt.callLater(function(){launch(["ship-login",value.name,"--expected-ship-id",value.id,"--code-stdin"],"login",secret);secret=""})
        return
      }
    }
    // One Disconnect includes both server logout and receiver cleanup. A
    // failed step stays explicit; refresh never repeats a mutation.
    if (ok && value && value.connection && kind === "app-disconnect" && value.connection.phase === "closed" && value.connection.receiver_reconciliation) {
      var closed=value.connection
      Qt.callLater(function(){launch(["app-reconcile",closed.request_id,"--expected-revision",String(closed.revision)],"app-reconcile","")})
      return
    }
    if (ok && value && value.connection && kind === "app-reconcile" && value.connection.phase === "closed" && !value.connection.receiver_reconciliation) {
      var done=value.connection
      Qt.callLater(function(){launch(["app-archive",done.request_id,"--expected-revision",String(done.revision)],"app-archive","")})
      return
    }
    failed = !ok
    if (!ok) {
      message = problem && typeof problem.message==="string" ? problem.message.slice(0,600) : value && value.message ? String(value.message).slice(0,600) :
        "The operation did not complete. Refresh its status before trying again; approval is never retried automatically."
    } else if (kind === "connect" || kind === "connect-fleet") {
      message = "Connected. The plugin now talks directly to your ship."
    } else if(kind === "permit") {
      message="Permission saved for this ship. Omabit can choose a verified URL when connecting."
    } else if(kind === "revoke") {
      message="Future connections are blocked. Existing sessions have their own Disconnect action."
    } else if (kind === "app-archive") {
      message = "Disconnected. The completed connection is saved in history."
    } else if (kind === "login") {
      message = "Ship sign-in saved. Open Application Connections to approve plugins."
    } else if (kind === "enroll") {
      message = "Ship added. Select it above and sign in once."
    } else {
      message = "Connection state updated. Follow any remaining recovery action below."
    }
    operation = ""
    Qt.callLater(refresh)
  }
  Timer {
    id: deadline
    onTriggered: {
      root.timedOut = true; root.pendingInput = ""; root.hostedCode = ""
      if (helper.running) { helper.signal(15); hardStop.restart() }
      else root.finish(-1)
    }
  }
  Timer { id: hardStop; interval: 5000; onTriggered: if (helper.running) helper.signal(9) }
  Process {
    id: helper
    onStarted: { if (root.pendingInput) write(root.pendingInput); root.pendingInput = ""; stdinEnabled = false }
    stdout: SplitParser {
      splitMarker: ""
      onRead: function(data) {
        if (root.output.length + data.length > 262144) { root.overflow = true; helper.signal(9) }
        else root.output += data
      }
    }
    stderr: SplitParser { splitMarker: ""; onRead: function(data) { if(root.errorOutput.length+data.length<=8192)root.errorOutput+=data } }
    onExited: function(code, status) {
      if (root.service && typeof root.service.recordHelperExit === "function")
        root.service.recordHelperExit("application-connections", status)
      root.finish(status === 0 ? code : -1)
    }
  }
}
