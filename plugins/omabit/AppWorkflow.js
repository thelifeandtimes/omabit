function bytes(s){try{return encodeURIComponent(s).replace(/%[0-9A-F]{2}/g,"x").length}catch(_){return Infinity}}
function validText(s,max,multiline){return typeof s==="string" && bytes(s)<=max && !(multiline ? /[\x00-\x08\x0b-\x1f\x7f-\x9f]/ : /[\x00-\x1f\x7f-\x9f]/).test(s)}
function configError(s){
  if(!s)return ""
  var c=s.desk||s.ship
  if(!c || !validText(c.instructions,2048,true))return "Instructions must be at most 2,048 UTF-8 bytes, without control characters."
  if(!s.desk)return ""
  if(c.workflow==="mounted" && !/^[a-z][a-z0-9-]{0,63}$/.test(c.source_name))return "Use one directory name: lowercase letters, digits and hyphens, starting with a letter."
  if(!validText(c.repository,256,false) || !validText(c.branch,128,false))return "The repository or branch is too long or contains control characters."
  if(c.checks.length>16 || !c.checks.every(function(v){return validText(v,256,false) && !!v.trim()}))return "Use at most 16 checks, each up to 256 UTF-8 bytes."
  return ""
}
function valid(v,runner,mode,target,Policy){
  if(!v || v.version!==1 || v.runner_id!==runner || v.mode!==mode || !isFinite(Date.parse(v.read_at)))return false
  var s=v.setting
  if(!s || s.version!==1 || !Policy.same(s.target,target) || !Policy.target(s.target) || !Policy.revision(s.revision))return false
  if(s.revision && (!validText(s.actor,128,false) || !s.actor || !isFinite(Date.parse(s.updated_at))))return false
  if(!target.desk)return !!s.ship && !s.desk && ["standard","development","distribution"].indexOf(s.ship.purpose)>=0 && validText(s.ship.instructions,2048,true)
  var d=s.desk
  return !!d && !s.ship && ["mounted","urgit"].indexOf(d.workflow)>=0 && (d.workflow==="mounted" ? /^[a-z][a-z0-9-]{0,63}$/.test(d.source_name) : d.source_name==="")
    && ["none","source","treaty"].indexOf(d.publication)>=0 && validText(d.repository,256,false) && validText(d.branch,128,false) && validText(d.instructions,2048,true)
    && Array.isArray(d.checks) && d.checks.length<=16 && d.checks.every(function(c){return validText(c,256,false) && !!c.trim()})
}
