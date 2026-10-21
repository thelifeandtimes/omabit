// In-memory regression tests of the shipped QML JavaScript. No collector, QML
// process, shell session or filesystem mutation is dispatched by these tests.
const fs=require("node:fs"),vm=require("node:vm"),assert=require("node:assert/strict"),path=require("node:path");
const read=name=>fs.readFileSync(path.join(__dirname,"../plugins/omabit",name),"utf8");
const model=vm.createContext({Date,Math,Number,JSON,isFinite});
vm.runInContext(read("Model.js"),model);
const clone=value=>JSON.parse(JSON.stringify(value));
function scope(file,values,names) {
  const text=read(file),context=vm.createContext({Model:model,Date,Math,Number,JSON,...values});
  context.root=context;
  for(const name of names) {
    const match=text.match(new RegExp("^  function "+name+"\\([^\\n]*\\) \\{[^\\n]*\\}$","m")) || text.match(new RegExp("^  function "+name+"\\([^\\n]*\\) \\{[\\s\\S]*?^  \\}","m"));
    assert.ok(match,"extract actual "+file+" function "+name);
    vm.runInContext(match[0],context);
  }
  return context;
}
function binding(context,file,name,type="var") {
  const expression=read(file).match(new RegExp("readonly property "+type+" "+name+": ([^\\n]+)"))[1];
  Object.defineProperty(context,name,{configurable:true,get(){return vm.runInContext(expression,context);}});
}
function lineHandler(context,file,needle,handler) {
  const line=read(file).split("\n").find(line=>line.includes(needle));
  const body=line.slice(line.indexOf(handler+": ")+handler.length+2).replace(/ \}$/,"");
  return vm.runInContext(body,context);
}
const now=Date.now(),stamp=new Date(now).toISOString();
function runner(name="A",pier="@ssd/fakeships/lab/zod") {
  const i={id:"lab",ship:"~zod",mode:"fake",config_revision:1,applied_revision:1,config:{http_port:18080,ames_port:34343,loom:31},desired_state:"stopped",observed_state:"stopped",observed_at:stamp,pier_path:pier};
  return {name,runner_id:"runner-"+name,mode:"development-managed",status:"available",checked_at:stamp,access_role:"operator",features:["pier-storage","storage-locations","import-pier","guided-pier-import","epoch-inventory","maintenance-schedules"],snapshot:{runner_id:"runner-"+name,instances:[i],operations:[],incidents:[],piers:{runner_id:"runner-"+name,observed_at:stamp,roots:[{id:"default",path:"/fixture/default"},{id:"ssd",path:"/fixture/ssd"},{id:"hdd",path:"/fixture/hdd"}],candidates:[{path:pier,fingerprint:"a".repeat(64),enrolled_by:"lab",busy:false}]}}};
}
function service(connections) {
  const calls=[],s=scope("Service.qml",{connections,now,observing:true,currentAction:"",inspections:{},epochInspection:null,importOutcomes:{},storageOutcome:null,registrationOutcome:null,bootTarget:null,routineOutcome:null,hostMoveOutcome:null,output:"",errors:"",outputFailed:false,lastError:"",lastResult:"",refreshPending:false,
    action:{stdinEnabled:false},timeout:{stop(){}},Qt:{callLater(){}},clearEpochPreview(){},
    submit(args,kind){if(s.busy)return false;calls.push({args:Array.from(args),kind});s.currentAction=kind;return true;}},
    ["storageConnection","movePier","addStorageRoot","operate","rememberInspection","reconcileInspection","importExternalPier","rememberImport","reconcileImports","finish"]);
  Object.defineProperty(s,"busy",{get(){return s.currentAction!=="";}});
  s.calls=calls;
  s.reply=(code,output="",errors="",outputFailed=false)=>{s.output=output;s.errors=errors;s.outputFailed=outputFailed;s.finish(code);};
  return s;
}
function selector(connection) {
  const s=scope("StorageLocationSelector.qml",{connection,selectedRoot:"default",customSelected:false,allowCustom:false,selector:{currentIndex:0},customStorage:{currentIndex:0}},["syncSelection","selectRoot"]);
  for(const [name,type] of [["roots","var"],["customPath","bool"],["rootID","string"]])binding(s,"StorageLocationSelector.qml",name,type);
  return s;
}
function movePane(s,connection) {
  const pane=scope("StoragePane.qml",{service:s,connection,moving:null,selected:null,submitted:false,destination:{text:""},moveLocation:selector(connection)},["invalidateMove","reviewMove","choose"]);
  binding(pane,"StoragePane.qml","moveDestination","string");
  return pane;
}

// RC11-03/28: same ID, revision AND public ship/path on two runners is not a
// transferable review. Drive the same functions used by the visible controls.
const a=runner(),b=runner("B"),moves=service([a,b]),pane=movePane(moves,a);
assert.equal(pane.reviewMove(a.snapshot.instances[0]),true);
const reviewed=pane.moving;
assert.equal(Object.isFrozen(reviewed),true);
assert.equal(pane.moveLocation.selector.currentIndex,1);
assert.equal(pane.moveLocation.rootID,"ssd");
assert.equal(pane.destination.text,"fakeships/lab/zod");
pane.destination.text="fakeships/renamed/zod";
pane.connection=b;
vm.runInContext(read("StoragePane.qml").match(/onConnectionChanged: ([^\n]+)/)[1],pane);
assert.equal(pane.moving,null,"outer selector invalidates the move draft");
lineHandler(pane,"StoragePane.qml",'text: root.submitted ? "Move submitted"',"onClicked");
assert.equal(moves.movePier(b,reviewed,"hdd",pane.destination.text),false);
assert.equal(moves.calls.length,0,"stale A review never emits a B mutation");
const replacedRoot=clone(a);replacedRoot.snapshot.piers.roots[1].identity={device:1,inode:9,birth:stamp};
assert.equal(model.currentPierMove(replacedRoot,reviewed,now),false,"same root path with a replaced directory invalidates review");
assert.match(read("SetupPane.qml"),/StoragePane \{[^\n]+preferredConnection: root.provisioningRunner/);
// Even a retained/stale connection object cannot bypass the fresh inventory.
for(const change of [c=>c.runner_id="replacement",c=>c.name="renamed",c=>c.snapshot.instances=[],c=>c.snapshot.instances[0].config_revision++,c=>c.snapshot.instances[0].pier_path="fakeships/replacement/zod",c=>c.snapshot.instances[0].ship="~nec",c=>c.snapshot.instances[0].mode="planet",c=>c.snapshot.piers.roots[1].path="/replacement",c=>c.snapshot.piers.candidates[0].fingerprint="b".repeat(64),c=>c.snapshot.piers.candidates=[],c=>c.snapshot.piers.candidates[0].busy=true,c=>c.snapshot.instances[0].delete_pending=true,c=>c.snapshot.instances[0].active_operation_id="other",c=>c.snapshot.instances[0].observed_state="running",c=>c.snapshot.instances[0].recovery_required=true,c=>c.status="unavailable",c=>c.access_role="observer",c=>c.features=[],c=>c.checked_at=new Date(now-16000).toISOString()]) {
  const changed=clone(a);change(changed);
  const s=service([a,b]),p=movePane(s,a);assert.equal(p.reviewMove(a.snapshot.instances[0]),true);
  s.connections=[changed,b];p.invalidateMove();assert.equal(p.moving,null);
  assert.equal(s.movePier(a,reviewed,"hdd","fakeships/renamed/zod"),false);
  assert.equal(s.calls.length,0);
}
const fresh=clone(a),valid=service([fresh]),p=movePane(valid,fresh);
assert.equal(p.reviewMove(fresh.snapshot.instances[0]),true);
valid.connections=[clone(fresh)];p.invalidateMove();assert.ok(p.moving,"unchanged refreshed inventory retains review");
p.moveLocation.selectRoot("hdd");p.destination.text="fakeships/renamed/zod";
assert.equal(p.moveDestination,"@hdd/fakeships/renamed/zod");
assert.equal(valid.movePier(fresh,p.moving,"hdd","@ssd/fakeships/renamed/zod"),false,"inconsistent pasted prefix cannot override the selector");
assert.equal(valid.movePier(fresh,p.moving,"missing",p.destination.text),false);
lineHandler(p,"StoragePane.qml",'text: root.submitted ? "Move submitted"',"onClicked");
assert.equal(p.submitted,true);
assert.deepEqual(valid.calls[0].args.slice(0,-1),["--runner","A","--expected-runner-id","runner-A","move-pier","lab","--path","@hdd/fakeships/renamed/zod","--revision","1","--request-id"]);
assert.equal(valid.movePier(fresh,p.moving,"hdd",p.destination.text),false,"double click is one dispatch");
const defaultRunner=runner("D","fakeships/lab/zod"),d=movePane(service([defaultRunner]),defaultRunner);
d.moveLocation.selectRoot("hdd");assert.equal(d.reviewMove(defaultRunner.snapshot.instances[0]),true);
assert.equal(d.moveLocation.rootID,"default");assert.equal(d.moveLocation.selector.currentIndex,0);
assert.equal(d.moveDestination,"fakeships/lab/zod");
const location=selector(a);location.selectRoot("ssd");
location.connection=clone(a);location.connection.snapshot.piers.roots.reverse();location.syncSelection();
assert.equal(location.rootID,"ssd");assert.equal(location.roots[location.selector.currentIndex].id,"ssd");
location.connection.snapshot.piers.roots=location.roots.filter(r=>r.id!=="ssd");location.syncSelection();
assert.equal(location.rootID,"");assert.equal(location.selector.currentIndex,-1,"removed root never silently selects another");
console.log("PASS RC11-03/28: immutable move target, external/inventory invalidation, dispatch recheck and synchronized root/path argv");

// RC11-25: execute the actual toggle and actual Service command builder while
// the ship is running. All downtime/resume settings survive both directions.
for(const [stop_running,resume] of [[false,false],[true,false],[true,true]]) {
  const c=runner(),i=c.snapshot.instances[0];i.observed_state=i.desired_state="running";
  i.schedule={revision:7,instance_revision:1,config:{enabled:false,action:"pack",weekdays:[0,2],hour_utc:2,minute_utc:30,window_minutes:15,stop_running,resume}};
  const before=clone(i.schedule.config),calls=[];
  const s=scope("Service.qml",{canSaveRoutine:t=>model.currentRoutineTarget([c],t,now),submit(args){calls.push(Array.from(args));return true;}},["saveRoutine"]);
  const card=vm.createContext({root:{service:s,connection:c,instance:i,policy:i.schedule},Model:model});
  for(const enabled of [true,false]) {
    lineHandler(card,"RoutineCard.qml","onClicked: root.service.saveRoutine","onClicked");
    assert.deepEqual(clone(s.routineOutcome.config),{...before,enabled});
    assert.ok(calls.at(-1).includes("--stop-running="+stop_running));
    assert.ok(calls.at(-1).includes("--resume="+resume));
    i.schedule.config=clone(s.routineOutcome.config);
  }
}
console.log("PASS RC11-25: real toggle/command round trip changes only enabled, including running stopped-only policies");

// RC11-26: finish() must distinguish no-admission errors from lost/truncated or
// mismatched acknowledgements. A later inventory cannot unlock an unknown ID.
const err=code=>JSON.stringify({error:{code,message:"Refused before admission"}});
for(const [exit,output,error,overflow,retry] of [[1,"",err("invalid_state"),false,true],[1,"",err("revision_conflict"),false,true],[2,"",err("client_error"),false,true],[1,"",err("client_error"),false,false],[1,"",err("state_store_error"),false,false],[-1,"","",false,false],[0,"malformed","",false,false],[1,"",err("invalid_state"),true,false],[0,JSON.stringify({id:"wrong",request_id:"other"}),"",false,false]]) {
  const c=runner(),s=service([c]),i=c.snapshot.instances[0];
  assert.equal(s.operate(c,i,"inspect-epochs"),true);const request=s.epochInspection.request;
  s.reply(exit,output,error,overflow);
  assert.equal(s.epochInspection.status,retry ? "failed" : "uncertain");
  const current=clone(c);current.snapshot.instances[0].config_revision++;
  s.connections=[current];s.reconcileInspection();
  const bound=s.inspections[model.inspectionKey(c,i)];
  assert.equal(model.inspectionPending(bound),!retry);
  assert.equal(s.operate(current,current.snapshot.instances[0],"inspect-epochs"),retry);
  if(retry) {assert.notEqual(s.epochInspection.request,request);continue;}
  // Another ship's inspection must not erase this request's uncertainty.
  const other=runner("other");s.connections=[current,other];
  assert.equal(s.operate(other,other.snapshot.instances[0],"inspect-epochs"),true);
  s.reply(1,"",err("invalid_state"));
  assert.equal(s.operate(current,current.snapshot.instances[0],"inspect-epochs"),false);
  current.snapshot.operations=[{id:"receipt",kind:"inspect-epochs",instance_id:i.id,config_revision:1,request_id:request,status:"failed",message:"Recorded terminal failure"}];
  s.reconcileInspection();
  assert.equal(s.inspections[model.inspectionKey(c,i)].status,"failed");
  assert.equal(s.operate(current,current.snapshot.instances[0],"inspect-epochs"),true);
}
console.log("PASS RC11-26: actual finish/dispatch distinguishes refusals from lost ack and retains per-target journal reconciliation");

function review(c,name="zod") {
  const source={path:"/incoming/"+name,fingerprint:"f".repeat(64),files:3,bytes:100};
  return {runner:c.runner_id,connection:c.name,source,loadedAt:now,plan:{runner_id:c.runner_id,source,storage_root:"default",instance_id:name,destination:"fakeships/"+name+"/"+name,ship:"~"+name,network:"fakenet",kind:"fake",http_port:18090,in_place:false}};
}
function importPane(s,c,r) {
  const pane=scope("PierImportPane.qml",{service:s,connection:c,review:r,current:true,submission:null,visible:true,consent:{checked:true}},["clearReview","reviewAnother","submitImport"]);
  for(const [name,type] of [["submitted","bool"],["outcome","var"],["progress","var"]])binding(pane,"PierImportPane.qml",name,type);
  return pane;
}
function importReceipt(s,c,target,status="succeeded",stage="complete") {
  const op={id:"op-"+target.request,instance_id:target.instance,request_id:target.request,kind:"import-pier",config_revision:1,status};
  c.snapshot.operations.push(op);
  c.snapshot.instances.push({id:target.instance,config_revision:1,external_import:{source:target.source,stage,total_bytes:100,copied_bytes:50,events:[]}});
  s.reconcileImports();return op;
}
// RC11-27: two simultaneously retained forms, last-action replacement, receipt
// arrival before/after ack, and terminal evidence disappearing from inventory.
const ic=runner("import"),imports=service([ic]),form=importPane(imports,ic,review(ic));
const hiddenForm=importPane(imports,ic,review(ic));hiddenForm.visible=false;
imports.externalImportPreview=form.review;
hiddenForm.clearReview();
assert.equal(hiddenForm.consent.checked,false);
assert.equal(imports.externalImportPreview,form.review,"hidden import form must not clear the visible form's review");
form.clearReview();
assert.equal(imports.externalImportPreview,null,"visible import form must invalidate its review");
form.consent.checked=true;
assert.equal(form.submitImport(),true);const first=form.submission;
assert.equal(Object.isFrozen(first),true);assert.equal(form.submitImport(),false);
assert.equal(form.reviewAnother(),false);
imports.reply(-1);assert.equal(form.outcome.status,"uncertain");
assert.equal(form.reviewAnother(),false);assert.equal(form.progress.active,true);
form.connection=b;
vm.runInContext(read("PierImportPane.qml").match(/onSelectionChanged: ([^\n]+)/)[1],form);
assert.equal(form.submission,first,"changing the runner cannot forget an unresolved submission");
assert.equal(imports.importExternalPier(ic,review(ic),true),false,"other surface cannot duplicate unresolved source");
assert.equal(imports.addStorageRoot(ic,"archive","/fixture/archive"),true);
imports.reply(1,"",err("invalid_request"));
assert.equal(form.outcome.request,first.request);assert.equal(form.progress.active,true);
const form2=importPane(imports,ic,review(ic,"nec"));assert.equal(form2.submitImport(),true);
const second=form2.submission,op2=importReceipt(imports,ic,second);
imports.reply(0,JSON.stringify({...op2,status:"queued"}));
assert.equal(form2.progress.active,false);assert.equal(form.progress.active,true);
assert.equal(form2.outcome.request,second.request);assert.equal(form.outcome.request,first.request);
const chooser=movePane(imports,ic);chooser.importer=form;chooser.showImport=true;
form.sourcePath=first.source;
const unchangedSource=form.sourcePath;
chooser.choose({path:"fakeships/other/pier"});
assert.equal(form.submission,first);assert.equal(form.sourcePath,unchangedSource,"discovery selection cannot reset an unresolved form");
const wrong=clone(ic);wrong.runner_id="replacement";importReceipt(imports,wrong,first);
assert.equal(form.progress.active,true,"replacement runner cannot resolve old import");
importReceipt(imports,ic,first);
assert.equal(form.progress.active,false);assert.equal(form.progress.value,1);
assert.equal(imports.addStorageRoot(ic,"another","/fixture/another"),true);
imports.reply(1,"",err("invalid_request"));
assert.equal(form.progress.active,false);
ic.snapshot.operations=[];ic.snapshot.instances=[];imports.connections=[];imports.reconcileImports();
assert.equal(form.progress.value,1,"terminal proof survives later journal eviction/disconnection");
assert.equal(form.reviewAnother(),true);assert.equal(form.submitted,false);
assert.equal(form2.progress.active,false);
// Refusal before admission is a new-review opportunity; a missing, running or
// mismatched receipt is not. Exercise both code paths via actual finish().
for(const refusal of [true,false]) {
  const c=runner("last"),s=service([c]),p=importPane(s,c,review(c));p.submitImport();
  s.reply(1,"",refusal ? err("revision_conflict") : "lost response");
  assert.equal(p.reviewAnother(),refusal);
  if(!refusal) {
    const op=importReceipt(s,c,p.submission,"running","copying");
    assert.equal(p.progress.value,.5);assert.equal(p.reviewAnother(),false);
    op.status="failed";s.reconcileImports();assert.equal(p.reviewAnother(),true);
  }
}
console.log("PASS RC11-27: per-request imports survive unrelated actions/other forms; only resolved requests unlock review");

// A receipt is about the submitted request, not the current generation of its
// instance name. Exercise the real finish/reconcile callbacks for both a bound
// acknowledgement and a lost acknowledgement, including unrelated histories.
for (const acknowledged of [true,false]) for (const changed of ["configured","deleted","recreated","metadata-removed","source-changed","stage-lag"]) {
  const c=runner("receipt-history"),s=service([c]),p=importPane(s,c,review(c));
  assert.equal(p.submitImport(),true);const target=p.submission;
  const receipt={id:"success",instance_id:target.instance,request_id:target.request,kind:"import-pier",config_revision:1,status:"succeeded"};
  s.reply(acknowledged ? 0 : -1,acknowledged ? JSON.stringify({...receipt,status:"queued"}) : "");
  p.sourcePath="/different/selection";p.connection=runner("selected-elsewhere");
  vm.runInContext(read("PierImportPane.qml").match(/onSelectionChanged: ([^\n]+)/)[1],p);
  c.snapshot.operations=[{...receipt,request_id:"historic-request"},{...receipt,kind:"recover-import"},{...receipt,instance_id:"different-instance"}];
  s.reconcileImports();assert.equal(p.progress.active,true);assert.equal(p.reviewAnother(),false);
  const otherRunner=clone(c);otherRunner.runner_id="replacement";otherRunner.snapshot.operations.push(receipt);
  s.connections=[otherRunner];s.reconcileImports();assert.equal(p.progress.active,true,"wrong runner's success is not proof");
  s.connections=[c];
  let current={id:target.instance,config_revision:1,external_import:{source:target.source,stage:"complete",events:[{stage:"copying",at:stamp}],copied_bytes:100,total_bytes:100}};
  if (changed==="configured") current.config_revision=2;
  if (changed==="deleted") current=null;
  if (changed==="recreated") current={id:target.instance,config_revision:3,external_import:{source:"/replacement",stage:"copying",events:[{stage:"removing-source",at:stamp}]}};
  if (changed==="metadata-removed") delete current.external_import;
  if (changed==="source-changed") current.external_import.source="/replacement";
  if (changed==="stage-lag") current.external_import.stage="copying";
  c.snapshot.instances=current ? [current] : [];
  c.snapshot.operations.push(receipt);s.reconcileImports();
  assert.equal(p.progress.active,false,changed);assert.equal(p.progress.value,1,changed);
  if (changed!=="stage-lag") assert.equal(p.progress.log,"","do not borrow current-instance history: "+changed);
  else assert.match(p.progress.detail,/Import complete/);
  c.snapshot.operations=[];s.connections=[];s.reconcileImports();
  assert.equal(p.progress.active,false,"terminal evidence survives eviction/offline");
  assert.equal(p.reviewAnother(),true);
  s.connections=[c];c.snapshot.operations=[receipt];p.connection=c;p.consent.checked=true;
  assert.equal(p.submitImport(),true);assert.notEqual(p.submission.request,target.request);
  s.reply(-1);s.reconcileImports();
  assert.equal(p.progress.active,true,"old receipt must not unlock a new import of the same source/name");
  assert.equal(p.reviewAnother(),false);
}
console.log("PASS RC11-27 follow-up: exact success is terminal across revision/delete/recreate changes; unrelated receipts and histories never unlock new requests");
