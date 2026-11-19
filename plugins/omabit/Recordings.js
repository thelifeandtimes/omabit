function hex(value,n) { return typeof value==="string" && new RegExp("^[a-f0-9]{"+n+"}$").test(value) && !/^0+$/.test(value) }
function id(value) { return typeof value==="string" && /^[A-Za-z0-9_-]{1,128}$/.test(value) }
function integer(value,max) { return Number.isSafeInteger(value) && value>=0 && value<=max }
function text(value,max) { return typeof value==="string" && value.length<=max }
function fail() { throw new Error("Recording reply does not match this selection.") }
function descriptor(s,runner,instance) {
  return !!s && s.runner_id===runner && s.instance_id===instance && hex(s.job_id,32) && hex(s.job_sha256,64)
    && hex(s.runtime_id,32) && hex(s.container_id,64) && id(s.operation_id) && id(s.request_id)
    && text(s.created_at,64) && isFinite(Date.parse(s.created_at))
}
function catalog(v,runner,instance,after) {
  if (!v || v.version!==1 || v.runner_id!==runner || v.instance_id!==instance || !Array.isArray(v.sessions) || v.sessions.length>32 || !isFinite(Date.parse(v.read_at))) fail()
  var seen={}
  v.sessions.forEach(function(s){if(!descriptor(s,runner,instance) || seen[s.job_id] || s.job_id===after)fail();seen[s.job_id]=true})
  if(v.next_job_id && (v.sessions.length!==32 || v.next_job_id!==v.sessions[31].job_id))fail()
  return v
}
function page(v,runner,instance,job,representation,cursor) {
  if (!v || v.version!==1 || !descriptor(v.session,runner,instance) || !job || v.session.job_id!==job.job_id || v.session.job_sha256!==job.job_sha256
      || v.representation!==representation || v.complete_export!==false || !integer(v.frame_count,64) || !v.status || !text(v.status.state,64)
      || typeof v.status.writer_present!=="boolean" || !text(v.status.session_storage,64) || !text(v.status.session_id_basis,64)
      || JSON.stringify(v.from||null)!==JSON.stringify(cursor||null) || !isFinite(Date.parse(v.read_at)))fail()
  if(v.frame_count>0 && !v.metadata)fail()
  if(v.metadata && (v.metadata.runner_id!==runner || v.metadata.instance_id!==instance || v.metadata.container_id!==v.session.container_id || v.metadata.operation_id!==v.session.operation_id || v.metadata.request_id!==v.session.request_id || v.metadata.session_id!==v.status.session_id || !hex(v.metadata_sha256,64)))fail()
  if(v.next && (!v.metadata || v.next.version!==1 || v.next.session_id!==v.metadata.session_id || v.next.metadata_sha256!==v.metadata_sha256 || !integer(v.next.sequence,33554432) || !v.next.sequence || !hex(v.next.digest,64)))fail()
  if(representation==="original") {
    if(v.redacted || !Array.isArray(v.original||[]) || (v.original||[]).length!==v.frame_count)fail()
    ;(v.original||[]).forEach(function(f){if(!f.frame || !integer(f.frame.Sequence,33554432) || !integer(f.frame.Kind,5) || !f.frame.Kind || !text(f.original_bytes_base64,100000) || !/^[A-Za-z0-9+/]*={0,2}$/.test(f.original_bytes_base64))fail()})
  } else {
    if((v.original||[]).length || v.frame_count>0 && (!v.redacted || v.redacted.representation!=="escaped-literal-plain-text" || !Array.isArray(v.redacted.frames) || v.redacted.frames.length!==v.frame_count))fail()
    ;(v.redacted && v.redacted.frames || []).forEach(function(f){
      if(!f.frame || !integer(f.frame.Sequence,33554432) || !integer(f.frame.Kind,5) || !f.frame.Kind || !Array.isArray(f.spans||[]))fail()
      ;(f.spans||[]).forEach(function(s){if(!integer(s.offset,65536) || !integer(s.bytes,65536) || !text(s.escaped_text||"",400000) || /[^\x20-\x7e]/.test(s.escaped_text||"") || !text(s.omission_reason||"",128))fail()})
    })
  }
  return v
}
function controls(v,runner) {
  if(!v || v.version!==1 || !v.state || v.state.version!==1 || v.state.runner_id!==runner || !integer(v.state.revision,4096)
      || v.state.revision>0 && !hex(v.state.digest,64) || !v.state.policy || v.state.policy.version!==1 || !Array.isArray(v.state.pins) || v.state.pins.length>256
      || typeof v.pilot_enabled!=="boolean" || ["available","unknown","pilot-disabled"].indexOf(v.capacity_status)<0 || !isFinite(Date.parse(v.read_at)))fail()
  if(v.capacity_status==="available" && (!v.capacity || v.capacity.runner_id!==runner || v.capacity.version!==1 || !integer(v.capacity.jobs,4096) || !integer(v.capacity.committed_bytes,4398046511104) || !Array.isArray(v.capacity.instances)))fail()
  if(v.capacity_status!=="available" && v.capacity)fail()
  return v
}
function bytes(n) { return Number.isFinite(n) && n>=0 ? (n/1048576).toFixed(1)+" MiB" : "Unknown" }
function frameLabel(f) { var at=Number.isFinite(f.UnixNano) && f.UnixNano>0 ? new Date(Math.floor(f.UnixNano/1000000)).toISOString() : "receive time unknown";return at+" · #"+f.Sequence+" · "+({1:"stdout",2:"stderr",3:"combined terminal",4:"coverage gap",5:"observed exit"}[f.Kind]||"unknown") }
function windowText(v) {
  if(!v)return ""
  var lines=[]
  if(v.representation==="original") (v.original||[]).forEach(function(f){lines.push(frameLabel(f.frame)+" · original base64\n"+f.original_bytes_base64)})
  else (v.redacted && v.redacted.frames || []).forEach(function(f){
    lines.push(frameLabel(f.frame))
    ;(f.spans||[]).forEach(function(s){lines.push(s.omission_reason ? "["+s.bytes+" bytes omitted: "+s.omission_reason+"]" : s.escaped_text||"")})
    if(f.gap)lines.push("[Coverage gap: "+f.gap.reason+"; missing bytes may be unknown]")
    if(f.exit)lines.push("[Independent process exit · code "+f.exit.code+"]")
  })
  return lines.join("\n")
}
