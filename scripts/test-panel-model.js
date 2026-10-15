const fs = require("node:fs");
const vm = require("node:vm");
const assert = require("node:assert/strict");
const path = require("node:path");
const model = vm.createContext({Date, Math, Number, JSON, isFinite});
vm.runInContext(fs.readFileSync(path.join(__dirname, "../plugins/omabit/Model.js"), "utf8"), model);
const setupReady = {execution_model:"owner-user",ready:true,can_setup:true,checks:[],connection:{name:"local",runner_id:"runner-local",mode:"development-managed"},browser_base:"http://127.0.0.1"};
assert.throws(()=>model.validateSetupStatus({...setupReady,execution_model:undefined}),/owner-user release/);
assert.equal(model.setupEvent(JSON.stringify({step:"ready",status:"complete",result:setupReady})).result.connection.runner_id,"runner-local");
assert.throws(() => model.validateSetupStatus({...setupReady,browser_base:"https://unrelated.example"}));
assert.throws(() => model.validateSetupStatus({...setupReady,connection:{...setupReady.connection,mode:"simulation"}}));
assert.throws(() => model.setupEvent(JSON.stringify({step:"execute-arbitrary",status:"complete"})));
assert.throws(() => model.setupEvent("x".repeat(65537)));
assert.equal(model.setupStepIndex("services"),model.setupStepIndex("authorize"));
console.log("PASS: local setup progress is bounded and ready state binds the local development runner");
const registerConnection={name:"local",runner_id:"runner-local",mode:"development-managed",status:"available",checked_at:new Date().toISOString(),access_role:"operator",features:["create-fake","fake-ship-options","create-comet"],snapshot:{instances:[{id:"comet",mode:"comet",config:{http_port:18080},endpoint:"http://127.0.0.1:18081"},{id:"development",mode:"fake",config:{http_port:18082}}]}};
assert.match(model.registrationProblem(registerConnection,"comet",18083,"comet",Date.now()),/already exists/);
assert.match(model.registrationProblem(registerConnection,"new",18081,"comet",Date.now()),/reserved/);
assert.equal(model.registrationProblem(registerConnection,"new",18083,"comet",Date.now()),"");
assert.equal(model.suggestInstance(registerConnection,"comet"),"comet-1");
assert.equal(model.suggestInstance({...registerConnection,snapshot:{instances:[{id:"comet-1"}]}},"comet"),"comet-2");
assert.equal(model.suggestHTTPPort(registerConnection),"18083");
assert.equal(model.fakeOptionsProblem(registerConnection,"bus","my-pier"),"");
assert.notEqual(model.fakeOptionsProblem(registerConnection,"~bus","../outside"),"");
assert.equal(model.instanceTitle({mode:"fake",ship:"~bus"}),"[~bus]");
assert.equal(model.registrationFailure(JSON.stringify({error:{code:"instance_exists",message:"duplicate"}})).uncertain,false);
assert.equal(model.registrationFailure("lost response").uncertain,true);
console.log("PASS: registration validates current names and both configured/observed host ports, and separates fake identity from local names");
assert.deepEqual(Array.from(model.bootstrapArgs("bare","https://example.invalid/apps.pill")),["--bootstrap","bare"]);
assert.deepEqual(Array.from(model.bootstrapArgs("recommended","https://example.invalid/apps.pill")),["--bootstrap","recommended","--pill","https://example.invalid/apps.pill"]);
assert.throws(()=>model.bootstrapArgs("recommended",""));
assert.throws(()=>model.bootstrapArgs("recommended","http://example.invalid/apps.pill"));
assert.throws(()=>model.bootstrapArgs("recommended","https://user:password@example.invalid/apps.pill"));
console.log("PASS: bare bootstrap ignores recommendations; Recommended Apps requires its own configured HTTPS source");
const failedRuntime={recovery_required:true,runtime_status:{initialized:true,running:false,exit_code:137,oom_killed:false,last_operation:{kind:"start",status:"failed",message:"graceful shutdown exceeded 300s"}}};
assert.match(model.runtimeDetail(failedRuntime),/exit.*137/);
assert.match(model.runtimeDetail(failedRuntime),/OOM killed: no/);
assert.match(model.runtimeDetail(failedRuntime),/graceful shutdown exceeded/);
assert.match(model.runtimeDetail(failedRuntime),/Recovery needs review/);
const exitedAfterStart={observed_state:"failed",recovery_required:true,runtime_status:{initialized:true,running:false,exit_code:255,oom_killed:false,last_operation:{kind:"start",status:"succeeded"}}};
const exitedDetail=model.runtimeDetail(exitedAfterStart);
assert.ok(exitedDetail.indexOf("Current runtime:")<exitedDetail.indexOf("Last operation:"));
assert.match(exitedDetail,/start completed earlier/);
assert.match(exitedDetail,/use Stop to reconcile/);
assert.doesNotMatch(model.codeUnavailable({...registerConnection,features:["login-code"]},exitedAfterStart,Date.now(),true),/Start the ship to retrieve/);
assert.equal(model.runtimeStage("waiting-for-identity"),"Booting · checking identity");
assert.match(model.runtimeDetail({runtime_status:{snapshot_minutes:2,default_snapshot_minutes:60}}),/2 minutes.*New starts use 60 minutes/);
assert.match(model.runtimeDetail({runtime_status:{snapshot_minutes:60,default_snapshot_minutes:60}}),/snapshot interval: 60 minutes/);
assert.equal(model.runtimeNarration(["2026-10-02T15:00:00Z kiln: downloading","2026-10-02T15:00:01Z kiln: done"," /sys/trace","http: fail 504"]),"kiln: done\nhttp: fail 504");
const logReply={runner_id:"runner-log",instance_id:"moon",config_revision:1,source:"docker",observed_at:new Date().toISOString(),lines:["boot: fixture"],truncated:false};
assert.equal(model.validateRuntimeLogs(logReply,"runner-log","moon",1),logReply);
assert.throws(()=>model.validateRuntimeLogs(logReply,"replacement","moon",1));
assert.throws(()=>model.validateRuntimeLogs(logReply,"runner-log","different",1));
assert.throws(()=>model.validateRuntimeLogs({...logReply,lines:["x".repeat(4097)]},"runner-log","moon",1));
console.log("PASS: runtime diagnostics show the failure/exit/OOM cause, and logs bind identity/revision with bounded narration");
const now = Date.now();
const storageConnection={...registerConnection,features:["storage-locations","import-pier","delete-pier"],snapshot:{...registerConnection.snapshot,piers:{roots:[{id:"default",path:"/var/lib/urbit-runner/piers"},{id:"ssd",path:"/mnt/ssd/ships"}]}}};
assert.equal(model.validPierPath("@ssd/fakeships/test/pier"),true);
for(const invalid of ["@/fakeships/test/pier","@default/fakeships/test/pier","@ssd/../test/pier","@ssd/archive/test/pier"]) assert.equal(model.validPierPath(invalid),false);
assert.equal(model.absolutePierPath(storageConnection,"@ssd/fakeships/test/pier"),"/mnt/ssd/ships/fakeships/test/pier");
const deleteShip={id:"test",pier_path:"@ssd/fakeships/test/pier",desired_state:"stopped",observed_state:"stopped"};
assert.equal(model.canDeletePier(storageConnection,deleteShip,now),true);
assert.equal(model.canDeletePier(storageConnection,{...deleteShip,observed_state:"running"},now),false);
assert.equal(model.canDeletePier({...storageConnection,access_role:"observer"},deleteShip,now),false);
assert.equal(model.canDeletePier(storageConnection,{...deleteShip,handoff:{direction:"out",state:"moved"}},now),false);
assert.equal(model.canDeletePier(storageConnection,{...deleteShip,external_import:{stage:"copying"}},now),false);
assert.equal(model.canDeletePier(storageConnection,{...deleteShip,delete_pending:true,observed_state:"unknown",recovery_required:true},now),true);
assert.equal(model.canOperate(storageConnection,{...deleteShip,delete_pending:true,recovery_required:true},"stop",now),false);
const externalReview={runner:storageConnection.runner_id,connection:storageConnection.name,source:{path:"/srv/incoming/zod",fingerprint:"a".repeat(64)},loadedAt:now};
assert.equal(model.externalReviewCurrent(storageConnection,externalReview,"/srv/incoming/zod",now),true);
assert.equal(model.externalReviewCurrent(storageConnection,externalReview,"/srv/incoming/another",now),false);
assert.equal(model.externalReviewCurrent({...storageConnection,runner_id:"replacement"},externalReview,"/srv/incoming/zod",now),false);
assert.equal(model.externalReviewCurrent({...storageConnection,checked_at:new Date(now+300001).toISOString()},externalReview,"/srv/incoming/zod",now+300001),false);
console.log("PASS: custom storage paths preserve layout; deletion and external-import reviews reject running, fenced, replaced, stale and observer selections");
const instance = {id: "alpha", config_revision: 2, applied_revision: 1, observed_state: "stopped", config: {http_port: 18081, ames_port: 34343, loom: 31}};
const connection = {name: "workstation", runner_id: "runner-original", mode: "development-managed-fake", status: "available", access_role: "operator", checked_at: new Date(now).toISOString(), snapshot: {runner_id: "runner-original", mode: "development-managed-fake", instances: [instance], operations: [], incidents: []}};
assert.equal(model.observation(JSON.stringify(connection)).runner_id, "runner-original");
assert.equal(model.canOperate(connection, instance, "start", now), true);
assert.equal(model.canOperate(connection, {...instance, recovery_required: true}, "start", now), false);
assert.equal(model.canOperate({...connection, access_role: "observer"}, instance, "start", now), false);
assert.equal(model.canOperate(connection, instance, "start", now + 16000), false);
assert.equal(model.canOperate({...connection, status: "unavailable"}, instance, "start", now), false);
const target = model.edit(connection, instance);
instance.config_revision = 3;
const args = Array.from(model.configure(target, 18082));
assert.equal(args[args.indexOf("--revision") + 1], "2");
assert.equal(args[args.indexOf("--expected-runner-id") + 1], "runner-original");
assert.throws(() => model.observation(JSON.stringify({...connection, snapshot: {...connection.snapshot, runner_id: "replacement"}})));
assert.throws(() => model.configure(target, "18081; id"));
console.log("PASS: panel model preserves runner identity/revision and disables unsafe or stale actions");
const moveShip={...instance,desired_state:"stopped",observed_at:new Date(now).toISOString(),runtime_status:{initialized:true}};
const moveSource={...connection,features:["host-handoff"],snapshot:{...connection.snapshot,instances:[moveShip]}};
const moveDest={...moveSource,name:"remote",runner_id:"remote-id",snapshot:{...moveSource.snapshot,runner_id:"remote-id",instances:[]}};
const moveTarget=model.edit(moveSource,moveShip), moveDestination={connection:"remote",runner:"remote-id"};
assert.equal(model.hostMoveProblem([moveSource,moveDest],moveTarget,moveDestination,"arrived","","",now),"");
const populatedMoveDest={...moveDest,snapshot:{...moveDest.snapshot,instances:Array.from({length:32},(_,n)=>({id:"existing-"+n,mode:"fake",ship:"~zod"}))}};
assert.equal(model.hostMoveProblem([moveSource,populatedMoveDest],moveTarget,moveDestination,"arrived","","",now),"");
const largeImportSource={run_id:"a".repeat(32),content_digest:"b".repeat(64),ship:"~zod",files:7,bytes:100000000000,qualified_at:new Date(now).toISOString()};
assert.equal(model.validImportSource(largeImportSource),true);
assert.equal(model.validImportSource({...largeImportSource,bytes:100000000001}),false);
assert.notEqual(model.hostMoveProblem([moveSource,{...moveDest,runner_id:"replacement"}],moveTarget,moveDestination,"arrived","","",now),"");
assert.notEqual(model.hostMoveProblem([moveSource,moveDest],moveTarget,moveDestination,"arrived","22","",now),"");
assert.notEqual(model.hostMoveProblem([moveSource,moveDest],{...moveTarget,revision:0},moveDestination,"arrived","","",now),"");
const fence={id:"a".repeat(32),direction:"out",state:"ready",destination_runner:"remote-id",destination_instance:"arrived"};
const fenced={...moveShip,handoff:fence,recovery_required:true};
for (const action of ["start","stop","pack","meld","roll","inspect-epochs"]) assert.equal(model.canOperate(moveSource,fenced,action,now),false);
assert.equal(model.canMoveHost(moveSource,fenced,now),true);
const fencedSource={...moveSource,snapshot:{...moveSource.snapshot,instances:[fenced]}};
assert.notEqual(model.hostMoveProblem([fencedSource,moveDest],moveTarget,moveDestination,"changed","","",now),"");
assert.equal(model.hostMoveProblem([fencedSource,moveDest],moveTarget,moveDestination,"arrived","","",now),"");
assert.equal(model.shipStatus(moveSource,{...fenced,handoff:{...fence,state:"moved"}},now,true).label,"Moved");
const hostMoveArgs=Array.from(model.hostMoveArgs(moveTarget,moveDestination,"arrived","18090","","attempt"));
assert.equal(hostMoveArgs[hostMoveArgs.indexOf("--to-runner-id")+1],"remote-id");
assert.equal(hostMoveArgs[hostMoveArgs.indexOf("--revision")+1],String(moveTarget.revision));
console.log("PASS: host moves bind both runners and source revision, retain destination on retry, and fence ordinary actions");
assert.equal(model.notices(null, connection).length, 0);
const offline = {...connection, status: "unavailable", snapshot: undefined};
assert.equal(model.notices(connection, offline).length, 1);
assert.equal(model.notices(offline, offline).length, 0);
assert.equal(model.notices(offline, connection).length, 1);
console.log("PASS: notifications suppress initial history and deduplicate unchanged connection failures");

const clone = value => JSON.parse(JSON.stringify(value));
const stamp = new Date(now).toISOString();
connection.features = ["offline-pack", "offline-meld", "offline-roll", "epoch-inventory", "epoch-prune-prefix"];
Object.assign(instance, {desired_state: "stopped", observed_at: stamp});
for (const kind of ["pack", "meld", "roll", "inspect-epochs"]) {
  assert.equal(model.canOperate(connection, instance, kind, now), true, kind);
  for (const changed of [{observed_state: "running"}, {desired_state: "running"}, {active_operation_id: "busy"},
    {recovery_required: true}, {observed_at: new Date(now - 31000).toISOString()}, {observed_at: undefined}])
    assert.equal(model.canOperate(connection, {...instance, ...changed}, kind, now), false);
  for (const changed of [{features: []}, {access_role: "observer"}, {access_role: undefined}, {status: "unavailable"}])
    assert.equal(model.canOperate({...connection, ...changed}, instance, kind, now), false);
}
assert.equal(model.canOperate(connection, {...instance, roll_requires_run: true}, "roll", now), false);
assert.equal(model.canOperate(connection, {...instance, roll_requires_run: true}, "inspect-epochs", now), true);
assert.equal(model.canOperate(connection, {...instance, recovery_required: true}, "stop", now), true);
const maintenance = model.maintenanceTarget(connection, instance, "meld", now);
assert.equal(model.currentMaintenanceTarget([connection], maintenance, now), true);
for (const changed of [{revision: 999}, {runner: "replacement"}, {connection: "another"}, {kind: "prune-epochs"}])
  assert.equal(model.currentMaintenanceTarget([connection], {...maintenance, ...changed}, now), false);
assert.deepEqual(Array.from(model.maintenanceArgs(maintenance, "ui-one")), ["--runner", "workstation", "--expected-runner-id", "runner-original", "operate", "alpha", "meld", "--revision", "3", "--request-id", "ui-one"]);
console.log("PASS: maintenance requires live capability, operator, stopped/recovery state, and original revision");

const names = ["0i0", "0i9007199254740992", "0i9007199254740993", "0i18446744073709551615"];
for (const name of names) assert.equal(model.validEpoch(name), true);
for (const name of ["0i01", "0i-1", "0i18446744073709551616", "0i999999999999999999999", "../log", 123])
  assert.equal(model.validEpoch(name), false);
assert.equal(model.epochLess(names[1], names[2]), true);
instance.epochs = {source: "vere-4.6-epoch-v2", pier_id: "a".repeat(32), digest: "d".repeat(64), observed_at: stamp,
  entries: names.map((name, i) => ({name, digest: String(i + 1).repeat(64), logical_bytes: 100 + i, allocated_bytes: 4096,
    file_count: i === 0 ? 4 : 5, snapshot_bytes: i === 0 ? 0 : 90, shared_files: false, protected: i >= 2}))};
assert.equal(model.canReviewEpochs(connection, instance, now), true);
const review = model.epochTarget(connection, instance, now);
const selection = model.selectedEpochTarget(review, names[1]);
assert.equal(review.cutoff, "");
assert.notEqual(review.inventory, instance.epochs);
assert.throws(() => model.selectedEpochTarget(review, names[2]));
assert.throws(() => model.selectedEpochTarget(review, "0i12"));
const preview = {runner_id: connection.runner_id, instance_id: instance.id, config_revision: instance.config_revision,
  inventory_digest: instance.epochs.digest, observed_at: stamp, cutoff: names[1], selected: clone(instance.epochs.entries.slice(0, 2)),
  retained: clone(instance.epochs.entries.slice(2)), selected_logical_bytes: 201, selected_allocated_bytes: 8192,
  shared_files: false, deletion_supported: true, requires_stopped_revalidation: true};
assert.equal(model.canPrunePreview([connection], selection, preview, now), true);
assert.deepEqual(Array.from(model.epochPruneArgs(selection, "ui-prune")), ["--runner", "workstation", "--expected-runner-id", "runner-original",
  "epoch-prune", "alpha", "--revision", "3", "--cutoff", names[1], "--expected-digest", "d".repeat(64), "--request-id", "ui-prune", "--confirm-history-loss"]);
for (const changed of [{runner_id: "replacement"}, {instance_id: "other"}, {config_revision: 4}, {inventory_digest: "f".repeat(64)},
  {observed_at: new Date(now + 1).toISOString()}, {cutoff: names[2]}, {selected: preview.selected.slice(1)},
  {retained: preview.retained.slice(1)}, {selected_logical_bytes: 202}, {shared_files: true}, {requires_stopped_revalidation: false},
  {selected: [preview.selected[0], {...preview.selected[1], digest: "f".repeat(64)}]}])
  assert.equal(model.canPrunePreview([connection], selection, {...preview, ...changed}, now), false);
assert.equal(model.canPrunePreview([connection], selection, {...preview, deletion_supported: false}, now), false);
for (const change of [c => c.access_role = "observer", c => c.runner_id = "replacement", c => c.features = [],
  c => c.status = "unavailable", c => c.snapshot.instances[0].config_revision++, c => c.snapshot.instances[0].active_operation_id = "new-op",
  c => c.snapshot.instances[0].observed_state = "running", c => c.snapshot.instances[0].recovery_required = true,
  c => c.snapshot.instances[0].epochs.digest = "e".repeat(64), c => c.snapshot.instances[0].epochs.entries[0].logical_bytes++,
  c => c.snapshot.instances[0].epochs.observed_at = new Date(now - 300001).toISOString()]) {
  const changed = clone(connection); change(changed);
  assert.equal(model.canPrunePreview([changed], selection, preview, now), false);
}
const unsupported = {...connection, features: ["epoch-inventory"]};
assert.equal(model.currentEpochTarget([unsupported], selection, now), true);
assert.equal(model.canPrunePreview([unsupported], selection, preview, now), false);
const observer = {...connection, access_role: "observer"};
const observerTarget = model.selectedEpochTarget(model.epochTarget(observer, instance, now), names[1]);
assert.equal(model.canPrunePreview([observer], observerTarget, preview, now), false);
for (const change of [{digest: undefined}, {shared_files: true}, {file_count: 6}, {snapshot_bytes: 0}]) {
  const changed = clone(connection);
  Object.assign(changed.snapshot.instances[0].epochs.entries[1], change);
  const selected = model.selectedEpochTarget(model.epochTarget(changed, changed.snapshot.instances[0], now), names[1]);
  const result = {...preview, selected: clone(selected.inventory.entries.slice(0, 2)), shared_files: !!change.shared_files};
  assert.equal(model.canPrunePreview([changed], selected, result, now), false);
}
for (const entries of [[instance.epochs.entries[1], instance.epochs.entries[0], ...instance.epochs.entries.slice(2)],
  instance.epochs.entries.map(e => ({...e, protected: false})), instance.epochs.entries.map(e => ({...e, logical_bytes: Number.MAX_SAFE_INTEGER + 1}))])
  assert.equal(model.validInventory({...instance.epochs, entries}, now), false);
console.log("PASS: prefix review preserves uint64 names and exact deletion evidence; stale, replaced, altered or protected selections fail closed");

const resource = {status: "fresh", source: "docker-stats-v1.47", container_id: "c".repeat(64), observed_at: stamp,
  memory_current_bytes: 0, memory_limit_bytes: 4294967296, pids_current: 0};
assert.match(model.resourceText({resources: resource}, now), /0.00 MiB.*tasks: 0.*fresh/);
assert.match(model.resourceText({resources: resource}, now + 31000), /stale/);
for (const change of [{memory_current_bytes: undefined}, {pids_current: null}, {status: "unavailable"}, {container_id: undefined}])
  assert.equal(model.resourceText({resources: {...resource, ...change}}, now), "Container resources: unavailable");
assert.match(model.resourceText({}, now), /not sampled/);
console.log("PASS: missing resource readings remain unknown, real zero is displayed, and old samples become stale");

const offlineHistory = model.merge([connection], {...connection, status:"unavailable", snapshot:undefined})[0];
assert.equal(model.instances(offlineHistory).length, 1);
assert.equal(model.shipStatus(offlineHistory,instance,now,true).label, "Unknown");
assert.equal(model.canOperate(offlineHistory,instance,"start",now), false);
assert.equal(model.instances(model.merge([connection], {...offline,runner_id:"replaced"})[0]).length, 0);
const local = [{name:connection.name,runner_id:connection.runner_id,transport:"unix",socket:"/tmp/runner.sock"}];
const remote = [{name:connection.name,runner_id:connection.runner_id,transport:"ssh",ssh:{host:"home-alias"}}];
const webShip = {...instance, endpoint:"http://127.0.0.1:18081"};
assert.equal(model.shipURL(connection,webShip,local,{}), webShip.endpoint);
assert.equal(model.shipURL(connection,webShip,remote,{}), "");
assert.equal(model.shipURL(connection,webShip,remote,{[connection.runner_id]:{browserBase:"http://home.tailnet"}}), "http://home.tailnet:18081");
assert.equal(model.shipURL(connection,webShip,remote,{[connection.runner_id]:{browserBase:"http://[::1]"}}), "http://[::1]:18081");
assert.equal(model.shipURL(connection,{...webShip,config:{...webShip.config,http_port:18099}},remote,{[connection.runner_id]:{browserBase:"http://home"}}), "http://home:18081");
assert.equal(model.shipURL(connection,{...webShip,endpoint:""},remote,{[connection.runner_id]:{browserBase:"http://home"}}), "");
assert.equal(model.shipStatus(connection,{...instance,observed_at:new Date(now-31000).toISOString()},now,true).label,"Stale");
for (const base of ["file:///etc/passwd","javascript:alert(1)","http://user:secret@host","http://host/path","http://host:1234","http://host\ninvalid"])
  assert.throws(() => model.browserBase(base));
const journal = clone(connection);
journal.snapshot.operations = [{id:"op-one",request_id:"one",instance_id:"alpha",kind:"pack",status:"succeeded",created_at:stamp},
  {id:"op-two",request_id:"two",instance_id:"beta",kind:"start",status:"failed",created_at:stamp}];
journal.snapshot.incidents = [{id:"incident",instance_id:"alpha",code:"recovery",active:true,observed_at:stamp}];
assert.equal(model.activity([journal],"","","all").length, 3);
assert.equal(model.activity([journal],journal.runner_id,journal.runner_id+"/alpha","all").length, 2);
assert.equal(model.activity([journal],"","","attention").length, 2);
assert.equal(model.activity([journal],"other","","all").length, 0);
assert.equal(model.activity([journal],"","","operations").length, 2);
assert.equal(model.activity([journal],"","","incidents").length, 1);
const sampling = clone(connection); sampling.snapshot.instances[0].resources = resource; sampling.snapshot.instances[0].observed_state="running";
const key = connection.runner_id + "/alpha";
const history = model.recordSamples({},sampling,now);
assert.equal(history[key].length, 1);
assert.equal(model.recordSamples(history,sampling,now)[key].length, 1);
assert.equal(model.recordSamples(history,{...sampling,status:"unavailable"},now)[key][1], null);
sampling.snapshot.instances[0].resources = {...resource,container_id:"a".repeat(64),observed_at:new Date(now+1).toISOString()};
assert.equal(model.recordSamples(history,sampling,now)[key].length, 1);
assert.equal(model.memorySample({...instance,resources:resource},now+31000), null);
assert.equal(model.memorySample({...instance,resources:{...resource,source:"unrecognized"}},now), null);
console.log("PASS: fleet status, remote browser routing, activity filters and sample discontinuities preserve identity and freshness");

const editConnection={name:"home",runner_id:"runner-pinned",revision:4};
const updateArgs=Array.from(model.connectionEditArgs(editConnection,"renamed","ssh","new-alias","/private/ssh"));
assert.equal(updateArgs[updateArgs.indexOf("--revision")+1],"4");
assert.equal(updateArgs[updateArgs.indexOf("--runner-id")+1],"runner-pinned");
assert.throws(()=>model.connectionEditArgs({...editConnection,revision:undefined},"renamed","ssh","host",""));
const codeConnection={...connection,features:["login-code"]};
const codeInstance={...instance,observed_state:"running",health:"healthy",applied_revision:instance.config_revision,observed_at:new Date(now).toISOString(),resources:resource};
assert.equal(model.canCopyCode(codeConnection,codeInstance,now),true);
assert.equal(model.canCopyCode({...codeConnection,access_role:"observer"},codeInstance,now),false);
assert.equal(model.canCopyCode(codeConnection,{...codeInstance,recovery_required:true},now),false);
assert.equal(model.canCopyCode(codeConnection,{...codeInstance,active_operation_id:"busy"},now),false);
assert.equal(model.canCopyCode(codeConnection,codeInstance,now+31000),false);
for(const change of [{health:"starting"},{health:"unknown"},{identity_pending:true},{applied_revision:0},{observed_state:"uninitialized"}]) {
  assert.equal(model.canCopyCode(codeConnection,{...codeInstance,...change},now),false);
  assert.equal(model.shipLive(codeConnection,{...codeInstance,...change},now),false);
}
console.log("PASS: connection edits bind original enrollment revision and code access requires a fresh operator selection");

const importSource={run_id:"a".repeat(32),content_digest:"b".repeat(64),files:7,bytes:1024,ship:"~zod",qualified_at:stamp};
const catalogue={connection:connection.name,runner:connection.runner_id,loadedAt:now,sources:[importSource]};
const imported=model.importTarget(catalogue,importSource,"copy",18086,now);
assert.notEqual(imported.source,importSource);
const importArguments=Array.from(model.importArgs(imported,"import-one"));
assert.equal(importArguments[importArguments.indexOf("--expected-digest")+1],"b".repeat(64));
assert.equal(importArguments[importArguments.indexOf("--expected-runner-id")+1],connection.runner_id);
assert.ok(importArguments.includes("--confirm-stopped"));
assert.throws(()=>model.importTarget(catalogue,{...importSource,claimed_by:"taken"},"copy",18086,now));
assert.throws(()=>model.importTarget(catalogue,{...importSource,content_digest:"c".repeat(64)},"copy",18086,now));
assert.throws(()=>model.importTarget(catalogue,importSource,"copy",18086,now+300001));
assert.equal(model.importAvailable([{...connection,features:["import-qualified-fake"]}],imported,now),true);
assert.equal(model.importAvailable([{...connection,features:["import-qualified-fake"],access_role:"observer"}],imported,now),false);
console.log("PASS: import reviews retain runner/source/digest binding and reject stale or claimed sources");

const comet = {...instance,mode:"comet"};
assert.equal(model.canOperate(connection,comet,"start",now),true);
for (const kind of ["pack","meld","roll","inspect-epochs"])
  assert.equal(model.canOperate(connection,comet,kind,now),false);
assert.equal(model.canReviewEpochs(connection,comet,now),false);
const cometConnection={...connection,features:connection.features.concat(["comet-compaction","maintenance-schedules"]),snapshot:{...connection.snapshot,instances:[comet]}};
for (const kind of ["pack","meld"]) {
  assert.equal(model.canOperate(cometConnection,comet,kind,now),true);
  assert.equal(model.canOperate(cometConnection,{...comet,observed_state:"running"},kind,now),false);
  assert.equal(model.canOperate({...cometConnection,access_role:"observer"},comet,kind,now),false);
}
for (const kind of ["roll","inspect-epochs"])
  assert.equal(model.canOperate(cometConnection,comet,kind,now),false);
assert.equal(model.canReviewEpochs(cometConnection,comet,now),false);
const cometPolicy=model.routineTarget(cometConnection,comet);
assert.equal(model.currentRoutineTarget([cometConnection],cometPolicy,now),true);
assert.equal(model.currentRoutineTarget([{...cometConnection,features:["maintenance-schedules"]}],cometPolicy,now),false);
assert.equal(model.currentRoutineTarget([{...cometConnection,features:["comet-compaction"]}],cometPolicy,now),false);
const cometMaintenance=model.maintenanceTarget(cometConnection,comet,"pack",now);
assert.equal(model.currentMaintenanceTarget([{...cometConnection,features:connection.features}],cometMaintenance,now),false);
console.log("PASS: comet compaction and routine controls require their capability; retention and stale confirmations stay gated");

const cometEpochConnection={...cometConnection,features:cometConnection.features.concat(["comet-epochs"])};
for (const kind of ["roll","inspect-epochs"])
  assert.equal(model.canOperate(cometEpochConnection,comet,kind,now),true);
assert.equal(model.canOperate(cometEpochConnection,{...comet,roll_requires_run:true},"roll",now),false);
assert.equal(model.canOperate(cometEpochConnection,{...comet,roll_requires_run:true},"inspect-epochs",now),true);
const cometEpochReview=model.selectedEpochTarget(model.epochTarget(cometEpochConnection,comet,now),"0i0");
const cometPreview={...preview,cutoff:"0i0",selected:clone(comet.epochs.entries.slice(0,1)),
  retained:clone(comet.epochs.entries.slice(1)),selected_logical_bytes:100,selected_allocated_bytes:4096,deletion_supported:false};
assert.equal(model.validateEpochPreview(cometPreview,cometEpochReview),cometPreview);
assert.equal(model.currentEpochTarget([cometEpochConnection],cometEpochReview,now),true);
assert.equal(model.currentEpochTarget([cometConnection],cometEpochReview,now),false);
assert.equal(model.canPrunePreview([cometEpochConnection],cometEpochReview,cometPreview,now),false);
assert.equal(model.canPrunePreview([cometEpochConnection],cometEpochReview,{...cometPreview,deletion_supported:true},now),false);
assert.equal(model.canPrunePreview([cometEpochConnection],{...cometEpochReview,mode:"fake"},{...cometPreview,deletion_supported:true},now),false);
console.log("PASS: comet epoch controls are capability-bound and cannot inherit fake-pier deletion permission");

const cometPruneConnection={...cometEpochConnection,features:cometEpochConnection.features.concat(["comet-prune-prefix"])};
assert.equal(model.canPrunePreview([cometPruneConnection],cometEpochReview,{...cometPreview,deletion_supported:true},now),true);
assert.equal(model.canPrunePreview([cometPruneConnection],cometEpochReview,cometPreview,now),false);
assert.equal(model.canPrunePreview([{...cometPruneConnection,access_role:"observer"}],cometEpochReview,{...cometPreview,deletion_supported:true},now),false);
assert.equal(model.canPrunePreview([cometEpochConnection],cometEpochReview,{...cometPreview,deletion_supported:true},now),false);
assert.equal(model.canPrunePreview([cometPruneConnection],cometEpochReview,{...cometPreview,deletion_supported:true},now+300001),false);
console.log("PASS: confirmed comet pruning additionally requires its own capability, exact preview and fresh operator review");

const routineConnection={...connection,features:["maintenance-schedules"]};
const routineSelection=model.routineTarget(routineConnection,instance);
const config={enabled:true,action:"pack",weekdays:[0,2],hour_utc:2,minute_utc:30,window_minutes:15,stop_running:true,resume:true};
assert.equal(model.currentRoutineTarget([routineConnection],routineSelection,now),true);
assert.equal(model.currentRoutineTarget([routineConnection],{...routineSelection,instanceRevision:99},now),false);
const policyArgs=Array.from(model.routineArgs(routineSelection,config));
assert.ok(policyArgs.includes("--stop-running=true") && policyArgs.includes("--resume=true"));
assert.throws(()=>model.routineArgs(routineSelection,{...config,stop_running:false}));
assert.throws(()=>model.routineArgs(routineSelection,{...config,weekdays:[]}));
assert.throws(()=>model.routineArgs(routineSelection,{...config,action:"prune-epochs"}));
console.log("PASS: routine drafts bind revisions and require explicit timing/downtime policy");

const grouped={snapshot:{instances:[
  {id:"fake",mode:"fake",ship:"~zod",observed_state:"running"},
  {id:"comet",mode:"comet",ship:"~doznec-salbud-salwex-sogref--tasweb-navfun-batwet-sigfet"},
  {id:"planet",ship:"~sampel-palnet"},{id:"moon",ship:"~doznec-salbud-salwex-sogref"},
  {id:"star",ship:"~marzod"},{id:"galaxy",ship:"~zod"},
  {id:"fake-stopped",mode:"fake",ship:"~bus",observed_state:"stopped"}
]}};
const groups=Array.from(model.shipGroups(grouped));
assert.deepEqual(groups.map(g=>g.type),["Galaxies","Stars","Planets","Moons","Comets","Fakeships"]);
assert.deepEqual(Array.from(groups[5].ships,s=>s.id),["fake","fake-stopped"]);
assert.equal(model.instanceTitle({mode:"comet"}),"Identity pending");
journal.snapshot.instances[0].ship="~bus"; journal.snapshot.instances[0].mode="fake";
const events=model.activity([journal],"","","all",{[journal.runner_id]:{label:"This computer"}});
assert.equal(events.filter(e=>e.instance==="alpha")[0].subject.ship,"~bus");
assert.equal(events[0].runner,"This computer");
assert.equal(model.activity([journal],"","","all").filter(e=>e.instance==="beta")[0].subject.ship,undefined);
console.log("PASS: fleet grouping follows ship rank, distinguishes fakes, and activity resolves @p within the correct runner");

assert.equal(model.resourceView(connection,instance,now,true).value,"N/A");
assert.equal(model.resourceView({...connection,status:"unavailable"},instance,now,true).value,"Unknown");
assert.equal(model.resourceView(connection,instance,now+31000,true).value,"Unknown");
assert.equal(model.resourceView(connection,{...instance,recovery_required:true},now,true).value,"Unknown");
assert.equal(model.resourceView(connection,{...instance,observed_state:"running",resources:resource},now,true).value,"0.00 MiB");
assert.match(model.routineTiming({config}),/Sun, Tue at 02:30 UTC/);
assert.match(model.routineBehavior({config}),/resumes it after success/);
assert.equal(model.sameRoutineConfig(config,{...config,enabled:false}),false);
console.log("PASS: confirmed stopped resources are N/A, stale/offline resources remain unknown, and policy summaries expose timing and downtime");

const latestRoll={kind:"roll",created_epoch:"0i300",finished_at:stamp,epochs_after:["0i100","0i200","0i300"]};
const historic={last_roll:latestRoll,maintenance:{kind:"pack",finished_at:stamp},pruning:{finished_at:new Date(now-60000).toISOString(),retained_epochs:["0i100","0i200"]},epoch_history:[{name:"0i100",rolled_at:stamp}]};
assert.equal(model.epochDisplay(historic,now).entries.length,3);
assert.equal(model.epochDisplay(historic,now).label,"Recorded at last roll");
assert.equal(model.epochDisplay({...historic,pruning:{...historic.pruning,finished_at:new Date(now+1).toISOString()}},now).entries.length,2);
assert.match(model.epochDate(historic,"0i100",now),/^Rolled Today at/);
assert.equal(model.epochDate(historic,"0i200",now),"Creation time not recorded");
assert.equal(model.epochBoundary("0i18446744073709551615"),"Event boundary 18,446,744,073,709,551,615");
const inspected=clone(connection);
inspected.snapshot.operations=[{id:"inspect-one",request_id:"inspect-request",instance_id:instance.id,config_revision:instance.config_revision,kind:"inspect-epochs",status:"succeeded",created_at:stamp,finished_at:stamp}];
const inspection={runner:connection.runner_id,connection:connection.name,instance:instance.id,revision:instance.config_revision,request:"inspect-request",status:"queued"};
assert.equal(model.inspectionStatus([inspected],inspection,now).status,"ready");
assert.equal(model.inspectionStatus([inspected],{...inspection,request:"different"},now).status,"queued");
assert.equal(model.inspectionStatus([{...inspected,runner_id:"replacement"}],inspection,now).status,"queued");
inspected.snapshot.instances[0].observed_state="running";
assert.equal(model.inspectionStatus([inspected],inspection,now).status,"unavailable");
console.log("PASS: roll evidence supersedes stale prune summaries, dates are sourced honestly, and inspection completion binds request, runner and current inventory");

const cpuResource={...resource,cpu_usage_percent:0,cpu_limit_cores:2};
const cpuShip={...instance,observed_state:"running",resources:cpuResource};
assert.equal(model.cpuSample(cpuShip,now).value,0);
assert.equal(model.cpuSample(cpuShip,now).limit,200);
for (const invalid of [null,undefined,-1,NaN,Infinity,"0"])
  assert.equal(model.cpuSample({...cpuShip,resources:{...cpuResource,cpu_usage_percent:invalid}},now),null);
assert.equal(model.cpuSample(cpuShip,now+31000),null);
assert.equal(model.cpuView(connection,instance,now,true).value,"N/A");
assert.equal(model.cpuView({...connection,status:"unavailable"},cpuShip,now,true).value,"Unknown");
const cpuConnection={...connection,snapshot:{...connection.snapshot,instances:[cpuShip]}};
const cpuHistory=model.recordSamples({},cpuConnection,now,"cpu");
assert.equal(cpuHistory[key][0].value,0);
cpuShip.resources={...cpuResource,container_id:"b".repeat(64),cpu_usage_percent:undefined};
assert.equal(model.recordSamples(cpuHistory,cpuConnection,now,"cpu")[key].length,0);
const points=[{time:now-25000,value:104857600},null,{time:now,value:157286400}];
const range=model.chartRange(points,"memory");
assert.equal(range.start,now-25000); assert.equal(range.end,now); assert.equal(range.maximum,209715200);
assert.equal(model.chartValue(range.maximum,"memory"),"200 MiB");
assert.equal(model.chartValue(50,"cpu"),"50%");
assert.match(model.chartTime(now),/^\d\d:\d\d:\d\d$/);
assert.equal(model.chartTime(null),"—");
assert.equal(model.chartRange([],"cpu").start,null);
console.log("PASS: CPU preserves true zero and sample gaps, replacement resets history, and axes use real times plus explicit units");

assert.equal(model.validPierPath("fakeships/project/zod/pier"),true);
assert.equal(model.validPierPath("livenet/moons/planet-moons/dister-dozzod-sampel-palnet/pier"),true);
for (const p of ["archive/zod","/fakeships/zod","fakeships/../zod","livenet/unknown/zod","fakeships/.private/zod"])
  assert.equal(model.validPierPath(p),false);
const discoverySource={path:"fakeships/project/zod/pier",kind:"fake",fingerprint:"f".repeat(64),busy:false};
const discovery={...connection,features:["pier-storage"],snapshot:{...connection.snapshot,piers:{runner_id:connection.runner_id,observed_at:stamp,candidates:[discoverySource]}}};
assert.equal(model.currentPierSource(discovery,discoverySource,now),true);
assert.equal(model.currentPierSource(discovery,{...discoverySource,fingerprint:"a".repeat(64)},now),false);
assert.equal(model.currentPierSource({...discovery,snapshot:{...discovery.snapshot,piers:{...discovery.snapshot.piers,runner_id:"replaced"}}},discoverySource,now),false);
assert.equal(model.currentPierSource(discovery,discoverySource,now+31000),false);
assert.equal(model.discoveredPiers([discovery]).length,1);
console.log("PASS: convention paths exclude archive/traversal and discovery selections bind runner, fingerprint and freshness");

assert.equal(model.ownedPierPath("sampel-palnet"),"livenet/planets/sampel-palnet/pier");
assert.equal(model.ownedPierPath("~zod"),"livenet/galaxies/zod/pier");
assert.equal(model.ownedPierPath("~marzod"),"livenet/stars/marzod/pier");
assert.equal(model.ownedPierPath("~dibpen-fogweg-dozzod-sarlev"),"livenet/moons/dibpen-fogweg-dozzod-sarlev/pier");
assert.equal(model.ownedPierPath("~dister-dozzod-sampel-palnet"),"livenet/moons/dister-dozzod-sampel-palnet/pier");
assert.equal(model.ownedPierPath("~doznec-salbud-salwex-sogref--tasweb-navfun-batwet-sigfet"),"");
for (const value of ["", "1024", "65535"]) assert.equal(model.optionalPort(value),true);
for (const value of ["0", "80", "65536", "18080; id"]) assert.equal(model.optionalPort(value),false);
console.log("PASS: key boot derives names/tier paths and accepts optional, validated port overrides");
