function gates(g, global) {
  return g && ["allow","block"].concat(global ? [] : ["inherit"]).indexOf(g.commit)>=0
    && ["allow","block"].concat(global ? [] : ["inherit"]).indexOf(g.publish)>=0
    && ["allow","block"].concat(global ? [] : ["inherit"]).indexOf(g.create_desk)>=0
}
function revision(n) { return typeof n==="number" && isFinite(n) && n>=0 && Math.floor(n)===n && n<=9007199254740991 }
function identifier(s){return typeof s==="string" && /^[A-Za-z0-9_-]{1,64}$/.test(s)}
function deskNames(a){return a===undefined || Array.isArray(a) && a.length<=2048 && a.every(function(d,i){return typeof d==="string" && /^[a-z][a-z0-9-]{0,63}$/.test(d) && a.indexOf(d)===i})}
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
  return v && v.version===1 && v.global && v.global.version===2 && revision(v.global.revision)
    && gates(v.global.gates,true) && Array.isArray(v.global.runners) && v.global.runners.every(function(r){return r && identifier(r.name) && identifier(r.runner_id) && typeof r.applied==="boolean" && (r.expected_revision===null || revision(r.expected_revision)) && acknowledged(r.acknowledged)}) && Array.isArray(v.runners)
    && v.runners.every(function(r) {
      if (!r || !identifier(r.name) || !identifier(r.runner_id) || typeof r.rollout!=="string" || typeof r.can_edit!=="boolean" || !acknowledged(r.acknowledged)) return false
      if (!r.status) return typeof r.error==="string" && !r.can_edit
      var s=r.status,p=s.policy
      return s.version===1 && s.runner_id===r.runner_id && s.desk_operations===false && isFinite(Date.parse(s.read_at))
        && p && p.version===2 && revision(p.revision) && gates(p.global,true) && gates(p.runner,false)
        && Array.isArray(p.entries) && p.entries.every(function(e){return e && target(e.target) && gates(e.gates,false) && (!e.target.desk || e.gates.create_desk==="inherit")})
        && Array.isArray(s.targets) && s.targets.every(function(t){return t && target(t.target) && typeof t.label==="string" && typeof t.ready==="boolean" && deskNames(t.desks)})
    })
}
function configured(p,scope,t) {
  if (scope==="runner") return p.runner
  for(var i=0;i<p.entries.length;i++) if(same(p.entries[i].target,t)) return p.entries[i].gates
  return {commit:"inherit",publish:"inherit",create_desk:"inherit"}
}
function effective(p,t) {
  var out={commit:{value:"block",source:"unavailable"},publish:{value:"block",source:"unavailable"},create_desk:{value:"block",source:"unavailable"}}
  if(!p)return out
  function apply(g,s){["commit","publish","create_desk"].forEach(function(k){if(!(s==="desk" && k==="create_desk") && g[k]!=="inherit")out[k]={value:g[k],source:s}})}
  apply(p.global,"global");apply(p.runner,"runner")
  var ship=JSON.parse(JSON.stringify(t||{}));delete ship.desk
  p.entries.forEach(function(e){if(same(e.target,ship))apply(e.gates,"ship")})
  if(t && t.desk)p.entries.forEach(function(e){if(same(e.target,t))apply(e.gates,"desk")})
  return out
}

function nodeKey(scope,runner,t) {
  t=t||{}
  return [scope,runner||"",t.instance_id||"",t.project_id||"",t.generation||0,t.creator||"",t.desk||""].join("/")
}

// This map never derives runner permissions from an unacknowledged global
// rollout. Each runner contributes its own authoritative saved policy.
function tree(v, extra, inventories) {
  if(!v)return null
  function decisions(g) {
    var out={}
    ;["commit","publish","create_desk"].forEach(function(k){out[k]={value:g[k],source:"global"}})
    return out
  }
  function node(scope,runner,t,title,subtitle,permission,ready) {
    return {key:nodeKey(scope,runner,t),scope:scope,runner:runner||"",target:t||{},title:title,subtitle:subtitle,
      permissions:permission,ready:ready!==false,children:[]}
  }
  var root=node("global","",{},"Global","Fleet defaults",decisions(v.global.gates))
  v.runners.forEach(function(r) {
    var p=r.status ? r.status.policy : null
    var runner=node("runner",r.runner_id,{},r.name,!p ? "Unavailable" : !r.can_edit ? "Observer" : "Runner",effective(p,{}),!!p)
    root.children.push(runner)
    if(!r.status)return
    r.status.targets.forEach(function(s) {
      var ship=node("ship",r.runner_id,s.target,s.ship||s.target.instance_id,s.label,effective(p,s.target),s.ready)
      runner.children.push(ship)
      var names=[]
      // Observations never grant permissions. Saved scopes and draft names stay
      // visible even if the last discovery did not find that desk.
      var observed=inventory(inventories,r.runner_id,s.target)
      ;(observed ? observed.desks : s.desks||[]).forEach(function(d){if(/^[a-z][a-z0-9-]{0,63}$/.test(d) && names.indexOf(d)<0)names.push(d)})
      p.entries.forEach(function(e){if(e.target.desk && same(e.target, Object.assign({},s.target,{desk:e.target.desk})) && names.indexOf(e.target.desk)<0)names.push(e.target.desk)})
      if(extra && extra.runner===r.runner_id && extra.target && extra.target.desk && same(extra.target,Object.assign({},s.target,{desk:extra.target.desk})) && /^[a-z][a-z0-9-]{0,63}$/.test(extra.target.desk) && names.indexOf(extra.target.desk)<0)names.push(extra.target.desk)
      names.sort().forEach(function(name) {
        var target=Object.assign({},s.target,{desk:name})
        var own=configured(p,"desk",target),overridden=own.commit!=="inherit" || own.publish!=="inherit"
        var child=node("desk",r.runner_id,target,"%"+name,(overridden ? "Desk override" : "Inherited"),effective(p,target),s.ready)
        child.installed=!!observed && observed.desks.indexOf(name)>=0
        if(child.installed)child.subtitle="Installed · "+child.subtitle
        ship.children.push(child)
      })
    })
  })
  return root
}

// Stable left-to-right tree; ancestors on a match's path are context, not
// permission matches. A blocked parent must never appear to be allowed.
function layoutTree(root,gate,filter,query) {
  var nodes=[],edges=[],row=0,matches=0
  query=(query||"").trim().toLowerCase()
  function walk(n,depth,ancestors) {
    var out=Object.assign({},n,{depth:depth}),childRows=[],path=ancestors.concat([n.title,n.subtitle]).join(" ").toLowerCase()
    nodes.push(out)
    var applicable=!(gate==="create_desk" && n.scope==="desk")
    out.decision=n.permissions[gate]
    out.match=n.ready && applicable && (filter==="all" || out.decision.value===filter) && (!query || path.indexOf(query)>=0)
    if(out.match)matches++
    out.onPath=out.match
    n.children.forEach(function(child) {
      var placed=walk(child,depth+1,ancestors.concat([n.title]))
      childRows.push(placed.row);out.onPath=out.onPath||placed.onPath
      edges.push({from:out,to:placed,highlight:placed.onPath})
    })
    out.row=childRows.length ? (childRows[0]+childRows[childRows.length-1])/2 : row++
    return out
  }
  if(root)walk(root,0,[])
  return {nodes:nodes,edges:edges,rows:Math.max(1,row),matches:matches}
}

function inventory(inventories,runner,t){
  return (inventories||[]).filter(function(i){return i.runner_id===runner && same(i.selection.target,t)})[0] || null
}
function discovery(v,request){
  return v && v.version===1 && v.runner_id===request.runner_id && identifier(v.mode)
    && v.selection && target(v.selection.target) && !v.selection.target.desk && same(v.selection.target,request.target)
    && typeof v.selection.runtime_id==="string" && /^[0-9a-f]{64}$/.test(v.selection.runtime_id)
    && revision(v.selection.config_revision) && v.selection.config_revision>0
    && Array.isArray(v.desks) && deskNames(v.desks)
    && isFinite(Date.parse(v.read_at)) && Date.now()-Date.parse(v.read_at)<=95000 && Date.parse(v.read_at)<=Date.now()+5000
}
