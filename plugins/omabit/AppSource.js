function valid(v,runner,mode,target,revision,source,Policy){
  if(!v || v.version!==1 || v.runner_id!==runner || v.mode!==mode || v.source_name!==source || !v.request || v.request.settings_revision!==revision || !v.request.selection)return false
  var s=v.request.selection,at=Date.parse(v.read_at)
  if(!Policy.same(s.target,target) || !Policy.target(s.target) || !Policy.revision(s.config_revision) || !s.config_revision || !/^[a-f0-9]{64}$/.test(s.runtime_id) || !isFinite(at) || at>Date.now()+5000 || Date.now()-at>95000)return false
  if(["synchronized","different","unix-changes","clay-changes","diverged"].indexOf(v.state)<0 || ["active","inactive","unknown"].indexOf(v.autocommit)<0 || !v.observation || !v.observation.exists || !Policy.revision(v.observation.revision) || !v.observation.revision)return false
  if(![v.unix_digest,v.canonical_unix_digest,v.clay_digest].every(function(d){return /^[a-f0-9]{64}$/.test(d)}))return false
  if(![v.difference_count,v.unix_change_count,v.clay_change_count,v.conflict_count].every(function(n){return Number.isInteger(n) && n>=0 && n<=16384}))return false
  if(v.state==="synchronized" ? v.difference_count!==0 || v.canonical_unix_digest!==v.clay_digest : v.difference_count===0 || v.canonical_unix_digest===v.clay_digest)return false
  if(!Array.isArray(v.differences) || v.differences.length!==Math.min(256,v.difference_count))return false
  return v.differences.every(function(d,i){return typeof d.path==="string" && d.path.length<=1024 && !/[\x00-\x1f\x7f]/.test(d.path) && ["modified","only-unix","only-clay"].indexOf(d.kind)>=0 && (!i || v.differences[i-1].path<d.path)})
}
function title(v){return v ? ({"synchronized":"Unix and Clay match","different":"Unix and Clay differ","unix-changes":"Uncommitted Unix changes","clay-changes":"Clay changed since the last match","diverged":"Unix and Clay have diverged"})[v.state] : "Source status not yet available"}
