function gates(g, global) {
  return g && ["allow","block"].concat(global ? [] : ["inherit"]).indexOf(g.commit)>=0
    && ["allow","block"].concat(global ? [] : ["inherit"]).indexOf(g.publish)>=0
}
function revision(n) { return typeof n==="number" && isFinite(n) && n>=0 && Math.floor(n)===n && n<=9007199254740991 }
function identifier(s){return typeof s==="string" && /^[A-Za-z0-9_-]{1,64}$/.test(s)}
function acknowledged(a){return !a || revision(a.revision) && gates(a.global,true)}
function same(a,b) {
  return (a.instance_id||"")===(b.instance_id||"") && (a.project_id||"")===(b.project_id||"")
    && (a.generation||0)===(b.generation||0) && (a.creator||"")===(b.creator||"") && (a.desk||"")===(b.desk||"")
}
function target(t) {
  return t && identifier(t.instance_id)
    && (t.project_id ? identifier(t.project_id) && revision(t.generation||0) && (t.generation ? identifier(t.creator) : !t.creator) : !t.generation && !t.creator)
    && (!t.desk || /^[a-z][a-z0-9-]{0,63}$/.test(t.desk))
}
function valid(v) {
  return v && v.version===1 && v.global && v.global.version===1 && revision(v.global.revision)
    && gates(v.global.gates,true) && Array.isArray(v.global.runners) && v.global.runners.every(function(r){return r && identifier(r.name) && identifier(r.runner_id) && typeof r.applied==="boolean" && (r.expected_revision===null || revision(r.expected_revision)) && acknowledged(r.acknowledged)}) && Array.isArray(v.runners)
    && v.runners.every(function(r) {
      if (!r || !identifier(r.name) || !identifier(r.runner_id) || typeof r.rollout!=="string" || typeof r.can_edit!=="boolean" || !acknowledged(r.acknowledged)) return false
      if (!r.status) return typeof r.error==="string" && !r.can_edit
      var s=r.status,p=s.policy
      return s.version===1 && s.runner_id===r.runner_id && s.desk_operations===false && isFinite(Date.parse(s.read_at))
        && p && p.version===1 && revision(p.revision) && gates(p.global,true) && gates(p.runner,false)
        && Array.isArray(p.entries) && p.entries.every(function(e){return e && target(e.target) && gates(e.gates,false)})
        && Array.isArray(s.targets) && s.targets.every(function(t){return t && target(t.target) && typeof t.label==="string" && typeof t.ready==="boolean"})
    })
}
function configured(p,scope,t) {
  if (scope==="runner") return p.runner
  for(var i=0;i<p.entries.length;i++) if(same(p.entries[i].target,t)) return p.entries[i].gates
  return {commit:"inherit",publish:"inherit"}
}
function effective(p,t) {
  var out={commit:{value:"block",source:"unavailable"},publish:{value:"block",source:"unavailable"}}
  if(!p)return out
  function apply(g,s){["commit","publish"].forEach(function(k){if(g[k]!=="inherit")out[k]={value:g[k],source:s}})}
  apply(p.global,"global");apply(p.runner,"runner")
  var ship=JSON.parse(JSON.stringify(t||{}));delete ship.desk
  p.entries.forEach(function(e){if(same(e.target,ship))apply(e.gates,"ship")})
  if(t && t.desk)p.entries.forEach(function(e){if(same(e.target,t))apply(e.gates,"desk")})
  return out
}
