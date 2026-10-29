// Exercise shipped restoration controller callbacks without QML, RPC or processes.
const fs=require('node:fs'),path=require('node:path'),vm=require('node:vm'),assert=require('node:assert/strict');
const plugin=path.join(__dirname,'../plugins/omabit');
const model=vm.createContext({Date,Math,Number,JSON,isFinite});
vm.runInContext(fs.readFileSync(path.join(plugin,'Model.js'),'utf8'),model);
const source=fs.readFileSync(path.join(plugin,'ImportRestoration.qml'),'utf8'),clone=v=>JSON.parse(JSON.stringify(v));
const now=Date.now(),stamp=new Date(now).toISOString(),runtime='1'.repeat(32);
function runner(){return {name:'lab',runner_id:'runner-lab',actor:'local-uid:1000',access_role:'operator',status:'available',checked_at:stamp,features:['restore-import'],snapshot:{instances:[{id:'friendly',ship:'~zod',config_revision:1,pier_path:'@disk/fakeships/friendly/zod',desired_state:'stopped',observed_state:'failed',external_import:{stage:'copying',source:'/incoming/source',holding_path:'/incoming/.omabit-import-'+runtime+'/pier'},runtime_status:{import_restore_available:true}}],operations:[],piers:{roots:[{id:'disk',path:'/disk'}]}}}}
function controller(c=runner()){
 const calls=[],s={connections:[c],now,observing:true,busy:false,submit(args,kind){if(this.busy)return false;calls.push({args:Array.from(args),kind});this.busy=true;return true}};
 const ctx=vm.createContext({Model:model,Date,Math,Number,JSON,isFinite,service:s,selection:null,pendingPreview:null,review:null,pendingRequest:'',outcomes:{},error:''});ctx.root=ctx;
 for(const name of [...source.matchAll(/^  function (\w+)\(/gm)].map(m=>m[1])){
  const match=source.match(new RegExp('^  function '+name+'\\([^\\n]*\\) \\{[^\\n]*\\}$','m')) || source.match(new RegExp('^  function '+name+'\\([^\\n]*\\) \\{[\\s\\S]*?^  \\}','m'));
  assert.ok(match,name);vm.runInContext(match[0],ctx);
 }
 ctx.calls=calls;ctx.reply=(kind,value,code=0,error='',overflow=false)=>{s.busy=false;return ctx.finish(kind,code,JSON.stringify(value),error,overflow)};
 return ctx;
}
function plan(c,t=c.pendingPreview){return {actor:t.actor,runner_id:t.runner,instance_id:t.instance,config_revision:t.revision,source:t.holding,restore_to:t.source,destination:t.destination,retain_at:t.retainAt,review_digest:'a'.repeat(64),source_files:7,source_bytes:1000,destination_files:1,destination_bytes:40,stage:t.stage,continuation:t.stage!=='review'}}
function inspect(c){const r=c.service.connections[0];assert.equal(c.preview(r,r.snapshot.instances[0]),true);const p=plan(c);assert.equal(c.reply('preview-import-restore',p),'');assert.equal(c.currentReview(c.review),true);return p}
function submit(c){const reviewed=c.review;assert.equal(c.restore(reviewed,true),true);assert.equal(c.restore(reviewed,true),false);return Object.values(c.outcomes).at(-1)}
function receipt(o,status='succeeded'){return {id:'op-'+o.request,actor:o.target.actor,instance_id:o.target.instance,config_revision:o.target.revision,kind:'restore-import',request_id:o.request,status,created_at:stamp,...(['succeeded','failed','interrupted','canceled'].includes(status)?{finished_at:stamp}:{}),...(status==='succeeded'?{import_restoration:Object.fromEntries(['source','restore_to','destination','retain_at','review_digest'].map(k=>[k,o.plan[k]]))}:{})}}

for(const change of [c=>c.status='unavailable',c=>c.access_role='observer',c=>c.actor='',c=>c.features=[],c=>c.checked_at=new Date(now-16000).toISOString(),c=>c.snapshot.instances[0].active_operation_id='op-busy',c=>c.snapshot.instances[0].desired_state='running',c=>c.snapshot.instances[0].delete_pending=true,c=>c.snapshot.instances[0].runtime_status.import_restore_available=false,c=>c.snapshot.instances[0].external_import.stage='enrolling',c=>c.snapshot.instances[0].external_import.holding_path='/incoming/other/pier',c=>c.snapshot.instances[0].external_import.source='/another/source',c=>c.snapshot.piers.roots[0].path='/disk/../changed']){
 const r=runner();change(r);const c=controller(r);assert.equal(c.preview(r,r.snapshot.instances[0]),false);assert.equal(c.calls.length,0);
}
for(const change of [p=>p.actor='ssh-key:other',p=>p.runner_id='replacement',p=>p.instance_id='another',p=>p.config_revision++,p=>p.source='/unrelated',p=>p.restore_to='/unrelated',p=>p.destination='/unrelated',p=>p.retain_at='/unrelated',p=>p.review_digest='invalid',p=>p.source_files=8193,p=>p.destination_bytes=8796093022209,p=>p.source_bytes=-1,p=>p.destination_files=1.5,p=>p.stage='restoring-source',p=>p.continuation=true]){
 const c=controller(),r=c.service.connections[0];c.preview(r,r.snapshot.instances[0]);const p=plan(c);change(p);assert.notEqual(c.reply('preview-import-restore',p),'');assert.equal(c.review,null);assert.equal(c.restore({target:c.selection,plan:p,loadedAt:now},true),false);assert.equal(c.calls.length,1);
}
for(const change of [c=>c.runner_id='replacement',c=>c.actor='ssh-key:replacement',c=>c.snapshot.instances[0].config_revision++,c=>c.snapshot.instances[0].ship='~nec',c=>c.snapshot.instances[0].pier_path='@disk/fakeships/changed/zod',c=>c.snapshot.piers.roots[0].path='/replacement',c=>c.snapshot.instances=[],c=>c.snapshot.instances[0].runtime_status.import_restore_stage='restoring-source']){
 const c=controller();inspect(c);const reviewed=clone(c.review),changed=clone(c.service.connections[0]);change(changed);c.service.connections=[changed];assert.equal(c.currentReview(reviewed),false);assert.equal(c.restore(reviewed,true),false);assert.equal(c.calls.length,1);
}
{
 const c=controller();inspect(c);const r=c.review;c.service.now+=300001;c.service.connections[0].checked_at=new Date(c.service.now).toISOString();assert.equal(c.restore(r,true),false);
 c.service.now=now;assert.equal(c.restore(r,false),false);assert.equal(c.calls.length,1);
}
const c=controller();inspect(c);const o=submit(c),key=c.pendingRequest;
assert.deepEqual(c.calls[1].args,['--runner','lab','--expected-runner-id','runner-lab','restore-import','friendly','--revision','1','--expected-actor','local-uid:1000','--restore-digest','a'.repeat(64),'--request-id',o.request,'--confirm-restore']);
assert.equal(c.review,null,'a submitted review cannot be reused');
c.reply('restore-import',null,-1,'acknowledgement lost');assert.equal(c.outcomes[key].status,'uncertain');
assert.equal(c.preview(c.service.connections[0],c.service.connections[0].snapshot.instances[0]),false,'lost acknowledgement prevents another request');
for(const change of [r=>r.actor='ssh-key:other',r=>r.instance_id='other',r=>r.config_revision=2,r=>r.request_id='another',r=>r.kind='import-pier',r=>r.import_restoration.retain_at='/elsewhere',r=>delete r.finished_at]){
 const bad=receipt(o);change(bad);c.service.connections[0].snapshot.operations=[bad];c.reconcile();assert.equal(c.outcomes[key].status,'uncertain','unrelated receipt cannot resolve request');
}
c.service.connections[0].status='unavailable';c.service.connections[0].snapshot.operations=[receipt(o)];c.reconcile();assert.equal(c.outcomes[key].status,'uncertain');
c.service.connections[0].status='available';c.service.connections[0].snapshot.instances=[];c.reconcile();assert.equal(c.outcomes[key].status,'succeeded','retirement does not erase terminal result');
c.service.connections=[];c.reconcile();assert.equal(c.outcomes[key].status,'succeeded','terminal receipt survives history/connection disappearance');
const reused=runner();reused.snapshot.instances[0].config_revision=2;c.service.connections=[reused];assert.equal(c.canReviewAgain(c.outcomes[key]),false);inspect(c);const newOutcome=submit(c),newKey=c.pendingRequest;
c.reply('restore-import',receipt(newOutcome,'queued'));reused.snapshot.operations=[receipt(o)];c.reconcile();assert.equal(c.outcomes[newKey].status,'queued','old generation receipt cannot settle the new request');

for(const status of ['failed','interrupted','canceled']){
 const c=controller();inspect(c);const o=submit(c),key=c.pendingRequest;
 c.reply('restore-import',receipt(o,status));assert.equal(c.outcomes[key].status,status);assert.equal(c.canReviewAgain(c.outcomes[key]),true);
 c.service.connections[0].snapshot.instances[0].runtime_status.import_restore_stage='restoring-source';assert.equal(c.reviewAgain(c.outcomes[key]),true);
 const p=plan(c);assert.equal(p.continuation,true);c.reply('preview-import-restore',p);assert.equal(c.currentReview(c.review),true);assert.equal(c.calls.filter(x=>x.kind==='restore-import').length,1,'fresh review never implicitly retries');
}
for(const [code,error,overflow,want] of [[2,'validation failed',false,'rejected'],[1,JSON.stringify({error:{code:'revision_conflict',message:'changed'}}),false,'rejected'],[1,'transport closed',false,'uncertain'],[0,'',true,'uncertain']]){
 const c=controller();inspect(c);submit(c);const key=c.pendingRequest;c.reply('restore-import',null,code,error,overflow);assert.equal(c.outcomes[key].status,want);assert.equal(c.canReviewAgain(c.outcomes[key]),want==='rejected');
}
{
 const c=controller();inspect(c);const o=submit(c),key=c.pendingRequest;
 c.service.connections[0].snapshot.operations=[receipt(o)];c.service.connections[0].snapshot.instances=[];c.reconcile();assert.equal(c.outcomes[key].status,'succeeded');
 c.service.connections=[];c.reply('restore-import',null,-1,'late lost acknowledgement');assert.equal(c.outcomes[key].status,'succeeded','late transport callback cannot erase terminal snapshot proof');
}
assert.match(model.importRestorationDetail(receipt(o)),/Restored source: \/incoming\/source/);
assert.match(model.importRestorationDetail(receipt(o)),/Retained partial copy: \/disk\/archive/);
assert.equal(model.importRestorationDetail({...receipt(o),status:'running'}),'');
console.log('PASS: restoration authority, paths, stale reviews, explicit approval, actor/request/revision receipts, uncertainty, retirement and continuation');
