function copy(v){return JSON.parse(JSON.stringify(v))}
// JSON object member order can change when a value crosses a QML property.
// Compare structure, not serialization order, for ship/helper identity checks.
function equal(a,b){
  if(a===b)return true
  if(!a || !b || typeof a!=="object" || typeof b!=="object" || Array.isArray(a)!==Array.isArray(b))return false
  var ak=Object.keys(a).sort(),bk=Object.keys(b).sort()
  return ak.length===bk.length && ak.every(function(k,i){return k===bk[i] && equal(a[k],b[k])})
}
function identifier(s){return typeof s==="string" && /^[A-Za-z0-9_-]{1,64}$/.test(s)}
function revision(n){return Number.isSafeInteger(n) && n>=0}
function origin(s){return typeof s==="string" && s.length<=2048 && /^https?:\/\/(?:\[[0-9a-fA-F:]+\]|[A-Za-z0-9][A-Za-z0-9.-]*)(?::[0-9]{1,5})?$/.test(s)}
function route(r){return r && identifier(r.id) && origin(r.origin) && ["local","tailnet","lan","public"].indexOf(r.network)>=0 && typeof r.scope==="string" && r.scope.length<=256 && ["manual","tunnel","managed","tailscale"].indexOf(r.source)>=0 && typeof r.enabled==="boolean"}
function valid(v,runner,mode,instance){
  if(!v || v.version!==1 || v.runner_id!==runner || v.mode!==mode || !v.selection || !v.selection.target || v.selection.target.instance_id!==instance || v.selection.target.desk || !/^[0-9a-f]{64}$/.test(v.selection.runtime_id||"") || !revision(v.selection.config_revision) || !v.selection.config_revision)return false
  var r=v.record,g=v.registry,b=r && r.binding,s=r && r.settings
  if(!r || r.version!==1 || !revision(r.revision) || !b || b.runner_id!==runner || !/^[0-9a-f]{64}$/.test(b.continuity||"") || typeof b.ship!=="string" || !/^~[a-z-]+$/.test(b.ship) || JSON.stringify(b.target)!==JSON.stringify(v.selection.target))return false
  if(!s || !Array.isArray(s.routes) || s.routes.length>29 || !s.routes.every(function(r){return route(r) && ["manual","tunnel"].indexOf(r.source)>=0}) || !Array.isArray(s.disabled_automatic) || s.disabled_automatic.length>3 || !s.disabled_automatic.every(function(id){return ["managed-http","tailscale-http","tailscale-serve"].indexOf(id)>=0}))return false
  if(!g || g.version!==1 || g.revision!==r.revision || JSON.stringify(g.binding)!==JSON.stringify(b) || !Array.isArray(g.routes) || g.routes.length>32 || !g.routes.every(route))return false
  var at=Date.parse(v.read_at),age=Date.now()-at
  return isFinite(at) && age>=-5000 && age<40000
}
function normalized(s){var v=copy(s);v.routes.sort(function(a,b){return a.id<b.id ? -1 : a.id>b.id ? 1 : 0});v.disabled_automatic.sort();return v}
function networkLabel(network){return {local:"This runner",tailnet:"Tailnet",lan:"Local network",public:"Public web"}[network]||network}
function automaticLabel(id){return {"managed-http":"Local connection","tailscale-http":"Direct Tailnet connection","tailscale-serve":"Tailscale proxy"}[id]||id}
function networkMessage(n){
  if(!n)return "Refresh to check this runner's network."
  if(n.tailscale==="not-installed")return "Tailscale is not installed on this runner. You can register an existing URL below."
  if(n.tailscale!=="ready")return "Tailscale is unavailable. Existing configured URLs are kept."
  if(n.serve==="tailnet")return "A Tailscale proxy is configured for this ship."
  if(n.serve==="public")return "Tailscale Funnel makes this ship's proxy accessible on the public web."
  return "Tailnet detected. No matching HTTPS proxy was found for this ship."
}
function formError(url,network,scope){
  if(!origin(url))return "Enter a URL without a path, query or credentials."
  if(network==="public" && url.indexOf("https://")!==0)return "Public web URLs require HTTPS."
  if(["tailnet","lan"].indexOf(network)>=0 && !scope)return network==="tailnet" ? "Enter the Tailnet DNS suffix." : "Enter the local network profile."
  return ""
}
