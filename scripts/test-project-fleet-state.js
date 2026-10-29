const fs=require('node:fs'),vm=require('node:vm'),assert=require('node:assert/strict'),path=require('node:path');
const m=vm.createContext({Date,Math,Number,JSON,isFinite});vm.runInContext(fs.readFileSync(path.join(__dirname,'../plugins/omabit/Model.js'),'utf8'),m);
const now=Date.now(),stamp=new Date(now).toISOString(),copy=v=>JSON.parse(JSON.stringify(v));
const c={name:'local',runner_id:'runner',mode:'development-managed',status:'available',access_role:'operator',checked_at:stamp,features:[],snapshot:{instances:[],operations:[],incidents:[]},project_inventory_status:'available',project_inventory:{schema_version:1,runner_id:'runner',mode:'development-managed',read_at:stamp,basis:'stored-reservations',runtime_observation:'not-performed',projects:[{id:'project-demo',label:'Demo',revision:1,review_digest:'a'.repeat(64),members:[{instance_id:'demo-zod',ship:'~zod',http_port:18080},{instance_id:'demo-nec',ship:'~nec',http_port:18081}]}]}};
assert.equal(m.instances(c).length,0);assert.equal(m.fleet([c]).length,2);assert.equal(m.shipGroups(c)[0].type,'Project · Demo');
const ship=m.fleet([c])[0].instance;
assert.equal(m.shipStatus(c,ship,now,true).label,'Project');
for(const action of ['start','stop','pack','meld','roll','inspect-epochs'])assert.equal(m.canOperate(c,ship,action,now),false,action);
assert.equal(m.canCopyCode(c,ship,now),false);assert.equal(m.shipLive(c,ship,now),false);
assert.equal(m.applicationShips({connections:[c],now,observing:true,registrations:[],runnerPreferences:{}},[]).length,0,'saved reservation is not a live auth selection');
assert.equal(m.suggestHTTPPort(c),'18082');assert.equal(m.httpPortOwner(c,18080).id,'demo-zod');
for(const mutate of [v=>v.runner_id='other',v=>v.schema_version=2,v=>v.runtime_observation='ready',v=>v.projects[0].members[1].instance_id='demo-zod',v=>v.projects[0].members[0].http_port=0]){const bad=copy(c);mutate(bad.project_inventory);assert.equal(m.projectMembers(bad).length,0)}
const offline=m.merge([c],{name:'local',runner_id:'runner',mode:c.mode,status:'unavailable',checked_at:stamp})[0];assert.equal(m.projectMembers(offline).length,2);assert.equal(m.shipStatus(offline,ship,now,true).label,'Unknown');
const other=m.merge([c],{name:'local',runner_id:'runner',mode:c.mode,status:'identity_mismatch',checked_at:stamp})[0];assert.equal(m.projectMembers(other).length,0);
console.log('PASS project fleet: pinned reservation navigation, no invented live health/auth, ordinary operations excluded, offline retention and port reservations');
