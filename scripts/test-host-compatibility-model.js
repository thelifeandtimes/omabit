const fs = require("node:fs");
const vm = require("node:vm");
const assert = require("node:assert/strict");
const path = require("node:path");
const {execFileSync} = require("node:child_process");
const root = path.join(__dirname,"..");
const model = vm.createContext({Date, Math, Number, JSON, isFinite});
vm.runInContext(fs.readFileSync(path.join(__dirname, "../plugins/omabit/Model.js"), "utf8"), model);
const fixtures=JSON.parse(execFileSync("go",["run","scripts/test-host-compatibility-fixtures.go"],{cwd:root,encoding:"utf8",env:{...process.env,GOFLAGS:""},timeout:120000}));
const connection=fixtures.compatible, report=connection.host_compatibility.report, ship=connection.snapshot.instances[0];
const now=Date.parse(connection.checked_at), at=connection.checked_at;
function wrap(r=report) { return {...connection.host_compatibility,report:r}; }
function observe(changes={}) { return model.observation(JSON.stringify({...connection,host_compatibility:wrap(),host_compatibility_status:"available",...changes})); }
function fixture(name) { return model.observation(JSON.stringify(fixtures[name])); }

const current = observe();
assert.equal(model.hostCompatibility(current).engine.version,"29.9.1");
assert.equal(model.canOperate(current,{...ship,observed_state:"stopped"},"start",now),true);
assert.match(model.compatibilityFacts(report,now),/provider: deb/);
assert.match(model.compatibilityFacts(report,now),/5:29.9.1-1~debian.13/);
assert.match(model.compatibilitySummary(report),/qualification: unqualified/);
assert.match(model.compatibilityChecks(report,false),/No matching full-stack/);
assert.doesNotMatch(model.compatibilityGuidance(report),/Omarchy|pacman/);
assert.equal(model.hostCompatibilityHint(current,now),"");
assert.ok(current.host_compatibility.expires_at);
assert.equal(current.host_compatibility.expiresAt,undefined);
assert.ok(current.host_compatibility.metadata_observed_at);
assert.equal(current.host_compatibility.metadataObservedAt,undefined);
assert.match(model.compatibilityFacts(report,now,current.host_compatibility),/Native package evidence:/);
assert.equal(current.host_compatibility.report.engine.data_root,"");
for(const distribution of [{id:"arch",provider:"pacman",profile:"omarchy"},{id:"other-linux",provider:"unknown",profile:"unqualified"}]) {
  const r={...report,distribution,provenance:"unknown",can_mutate:false,guidance:["Resolve native package evidence on "+distribution.id+"."]};
  assert.match(model.compatibilityFacts(r,now),new RegExp("provider: "+distribution.provider));
  assert.equal(model.compatibilityGuidance(r),r.guidance[0]);
  assert.equal(model.canOperate(observe({host_compatibility:wrap(r)}),ship,"stop",now),true);
}
console.log("PASS: distro-native report facts and qualification survive without UI engine-version policy");

for (const engine of ["30.0.0","29.9.1-rc1","unknown"]) {
  const c=observe({host_compatibility:wrap({...report,engine:{...report.engine,version:engine},compatibility:"unknown",can_mutate:false})});
  assert.equal(model.canOperate(c,ship,"stop",now),true);
  assert.equal(model.canOperate(c,{...ship,observed_state:"stopped"},"start",now),false);
  assert.equal(model.shipStatus(c,ship,now,true).label,"Healthy");
  assert.doesNotMatch(model.runtimeDetail(ship),/Recovery/);
  assert.equal(model.canOperate({...c,access_role:"observer"},ship,"stop",now),false);
  assert.equal(model.canOperate(c,{...ship,active_operation_id:"busy"},"stop",now),false);
  assert.equal(model.canOperate(c,{...ship,delete_pending:true},"stop",now),false);
  assert.equal(model.canOperate(c,{...ship,handoff:{direction:"out",state:"moving"}},"stop",now),false);
  assert.equal(model.canOperate(c,{...ship,recovery_required:true,observed_state:"failed"},"stop",now),true);
}
assert.equal(model.canOperate(observe({host_compatibility:wrap({...report,can_stop:false,can_mutate:false})}),ship,"stop",now),false);
const pending = {...report,restart_pending:true,can_mutate:false,engine:{...report.engine,version:"29.7.2"}};
assert.match(model.compatibilitySummary(pending),/restart pending/);
assert.doesNotMatch(model.compatibilitySummary(pending),/pier corruption|Recovery/);
assert.match(model.compatibilityFacts(pending,now),/Installed engine: 29.9.1 · running engine: 29.7.2/);
assert.equal(model.instances(observe({host_compatibility:wrap({...report,can_observe:false,can_stop:false,can_mutate:false})})).length,1);
console.log("PASS: backend Stop permission is independent of new work; recovery/access/operation fences remain");

const metadata=fixture("metadata"), startable={...ship,observed_state:"stopped"};
assert.equal(model.hostCompatibility(metadata).can_stop,false);
assert.equal(model.liveHostCompatibility(metadata,now),null);
assert.equal(model.canOperate(metadata,ship,"stop",now),true);
assert.equal(model.canOperate(metadata,startable,"start",now),true);
assert.match(model.compatibilityFacts(model.hostCompatibility(metadata),now),/Metadata only/);
assert.match(model.hostCompatibilityHint(metadata,now),/permissions are unresolved/);
for(const name of ["compatible","baseline","restarted"]) {
  const c=fixture(name);
  assert.equal(model.hostCompatibility(c).qualification,"unqualified");
  assert.equal(model.canOperate(c,startable,"start",now),true);
}
for(const name of ["pending","unsupported","uncertain","unverified"]) {
  const c=fixture(name);
  assert.equal(model.canOperate(c,ship,"stop",now),true,name);
  assert.equal(model.canOperate(c,startable,"start",now),c.host_compatibility.report.can_mutate,name);
}
for(const name of ["rootful","cgroup","data-root","api"]) {
  const c=fixture(name);
  assert.equal(model.canOperate(c,ship,"stop",now),false,name);
  assert.equal(model.canOperate(c,startable,"start",now),false,name);
  assert.equal(model.shipStatus(c,ship,now,true).label,"Healthy");
}
let updated=model.merge([fixture("pending")],fixture("restarted"))[0];
assert.equal(model.hostCompatibility(updated).restart_pending,false);
assert.equal(model.hostCompatibility(updated).engine.version,"29.9.1");
assert.equal(model.canOperate(updated,startable,"start",now),true);
updated=model.merge([updated],fixture("metadata"))[0];
assert.equal(model.liveHostCompatibility(updated,now),null);
assert.equal(model.canOperate(updated,ship,"stop",now),true);
console.log("PASS: Go-evaluated metadata, qualification, safety failures, package drift and daemon restart drive distinct UI behavior");

for (const changed of [
  {features:[],host_compatibility:undefined,host_compatibility_status:"unsupported"},
  {host_compatibility:undefined,host_compatibility_error:{code:"host_report_timeout",message:"UNTRUSTED"}},
  {host_compatibility:undefined,host_compatibility_error:{code:"host_report_identity_mismatch",message:"UNTRUSTED"}},
  {host_compatibility:wrap({...report,schema_version:2})},
  {host_compatibility:{...wrap(),runner_id:"runner-other"}},
  {host_compatibility:{...wrap(),mode:"simulation"}},
  {host_compatibility:{...wrap(),protocol_version:2}},
  {host_compatibility:{...wrap(),expires_at:undefined,expiresAt:wrap().expires_at}},
  {host_compatibility:wrap({...report,can_stop:"true"})},
  {host_compatibility:wrap({...report,engine:false})},
  {host_compatibility:wrap({...report,distribution:"debian"})},
  {host_compatibility:wrap({...report,packages:[true]})},
  {host_compatibility:wrap({...report,guidance:["x".repeat(8193)]})},
  {host_compatibility:wrap({...report,checks:Array(129).fill(report.checks[0])})},
  {host_compatibility:wrap({...report,provenance:"\u001bunsafe"})}
]) {
  const c=observe(changed);
  assert.equal(model.instances(c).length,1);
  assert.equal(model.available(c,now),true);
  assert.ok(model.hostCompatibilityHint(c,now));
  assert.doesNotMatch(model.hostCompatibilityHint(c,now),/UNTRUSTED/);
  assert.equal(model.fleet(model.merge([current],c)).length,1);
  assert.equal(model.canOperate(c,ship,"stop",now),true);
}
const older=observe({features:[],host_compatibility:undefined});
assert.match(model.hostCompatibilityHint(older,now),/Update this runner.*host-compatibility-v1/);
assert.equal(model.canOperate(older,{...ship,observed_state:"stopped"},"start",now),true);
const stale=fixture("stale");
assert.match(model.hostCompatibilityHint(stale,now),/stale/);
assert.match(model.compatibilityFacts(model.hostCompatibility(stale),now),/stale/);
for(const c of [fixture("stale"),observe({host_compatibility:wrap(fixtures.rootful.host_compatibility.report),host_compatibility_status:"stale"}),observe({host_compatibility:report})]) {
  assert.equal(model.liveHostCompatibility(c,now),null);
  assert.equal(model.canOperate(c,ship,"stop",now),true);
  assert.equal(model.canOperate(c,startable,"start",now),true);
}
const replaced=observe({runner_id:"runner-replacement",snapshot:{...connection.snapshot,runner_id:"runner-replacement"}});
assert.equal(model.liveHostCompatibility(replaced,now),null);
assert.equal(model.merge([fixture("rootful")],replaced)[0].host_compatibility,null);
assert.equal(model.merge([current],{...replaced,status:"unavailable",snapshot:undefined})[0].last_snapshot,undefined);
for(const c of [
  {...fixture("rootful"),features:[]},
  {...fixture("rootful"),runner_id:"runner-replacement",snapshot:{...connection.snapshot,runner_id:"runner-replacement"}},
  {...fixture("rootful"),host_compatibility_error:{code:"host_report_unavailable"}},
  {...fixture("rootful"),host_compatibility:{...wrap(),report:{...report,can_stop:false},expires_at:new Date(now).toISOString()}},
  {...fixture("rootful"),host_compatibility:{...wrap(),report:{...report,can_stop:false},metadata_observed_at:new Date(now+1000).toISOString()}}
]) assert.equal(model.liveHostCompatibility(c,now),null);
// A fresh collector poll cannot extend the report's own short live expiry.
const expired=fixture("rootful"), expiredAt=Date.parse(expired.host_compatibility.expires_at);
expired.checked_at=new Date(expiredAt).toISOString();
assert.equal(model.liveHostCompatibility(expired,expiredAt),null);
assert.equal(model.canOperate(expired,ship,"stop",expiredAt),true);
assert.match(model.compatibilityEvidence(model.hostCompatibility(expired),expiredAt,expired.host_compatibility),/stale/);
const independent=observe({status:"unavailable",snapshot:undefined});
assert.equal(model.hostCompatibility(independent).installed_engine_version,"29.9.1");
assert.equal(model.instances(model.merge([current],independent)[0]).length,1);
console.log("PASS: old, missing, invalid, stale and independently available reports preserve fleet inventory");

const setup={execution_model:"owner-user",ready:false,can_setup:true,checks:[],compatibility:{...report,engine:null,can_mutate:false}};
assert.equal(model.validateSetupStatus(setup).can_setup,true);
assert.equal(model.validateSetupStatus({...setup,compatibility:undefined}).compatibility,undefined);
const invalid=model.validateSetupStatus({...setup,compatibility:{...report,schema_version:2}});
assert.match(invalid.compatibility_problem,/Update/);
assert.equal(invalid.can_setup,true);
const empty={...report,packages:null,checks:null,guidance:null};
assert.equal(model.validateHostCompatibility(empty),empty);
assert.equal(model.compatibilityChecks(empty,true),"");
assert.equal(model.compatibilityGuidance(empty),"");
assert.match(model.compatibilitySummary({...report,provenance:"metadata-only"}),/provenance: metadata-only/);
console.log("PASS: local checks retain core setup eligibility and tolerate absent optional diagnostics");

// Invoke the shipped dispatch callback, not a reconstruction of its binding
// rules. A queued click cannot use a former runner, config revision or report.
const source=fs.readFileSync(path.join(root,"plugins/omabit/Service.qml"),"utf8");
const method=source.match(/^  function operate\(connection, instance, kind\) \{[\s\S]*?^  \}/m);
assert.ok(method);
const dispatch=vm.createContext({Model:model,now,busy:false,observing:true,connections:[current],inspections:{},lastError:"",requestIdentity:"",submissions:[],submit(args,kind){this.submissions.push({args,kind});return true;}});
// Function calls from QML resolve properties on the service root.
dispatch.submit=(args,kind)=>{dispatch.submissions.push({args,kind});return true;};
dispatch.root=dispatch;
vm.runInContext(method[0],dispatch);
assert.equal(dispatch.operate(current,ship,"stop"),true);
dispatch.connections=[fixture("rootful")];
assert.equal(dispatch.operate(current,ship,"stop"),false);
dispatch.connections=[replaced];
assert.equal(dispatch.operate(current,ship,"stop"),false);
dispatch.connections=[{...current,snapshot:{...current.snapshot,instances:[{...ship,config_revision:2}]}}];
assert.equal(dispatch.operate(current,ship,"stop"),false);
assert.equal(dispatch.submissions.length,1);
console.log("PASS: shipped dispatch rechecks the current runner, ship revision and live host evaluation");
