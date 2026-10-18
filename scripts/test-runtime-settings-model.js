const fs=require('node:fs'),vm=require('node:vm'),assert=require('node:assert/strict'),path=require('node:path');
const model=vm.createContext({Date,Math,Number,JSON,isFinite});
vm.runInContext(fs.readFileSync(path.join(__dirname,'../plugins/omabit/Model.js'),'utf8'),model);
const c={name:'local',runner_id:'runner-test',features:['runtime-settings']};
const i={id:'dev',mode:'fake',config_revision:7,config:{http_port:18080,ames_port:34343,loom:34,native_control:true}};
const target=model.edit(c,i);
assert.ok(!Array.from(model.configure(target,18081)).some(a=>a.startsWith('--native-control')),'HTTP-only edit preserves native control by omission');
assert.equal(Array.from(model.configure(target,18081,32)).at(-1),'32');
for (const loom of [19,20,29,35,31.5]) assert.throws(()=>model.configure(target,18081,loom),/30.*34/);
assert.throws(()=>model.configure({...target,runtimeSettings:false},18081,34),/Update/);
for (const [exp,label] of [[20,'20 · 1 MiB'],[29,'29 · 512 MiB'],[30,'30 · 1 GiB'],[31,'31 · 2 GiB'],[34,'34 · 16 GiB']]) assert.equal(model.loomLabel(exp),label);
assert.equal(model.loomLabel(35),'Unsupported loom');
for (let loom=30;loom<=34;loom++) assert.deepEqual(Array.from(model.runtimeSettingsArgs(c,loom)),['--loom',String(loom)]);
for (const loom of [0,20,29,35,31.5]) assert.throws(()=>model.runtimeSettingsArgs(c,loom),/30.*34/);
assert.throws(()=>model.runtimeSettingsArgs({...c,features:[]},34),/Update/);
assert.deepEqual(Array.from(model.runtimeSettingsArgs({...c,features:[]},31)),[]);
const detail=model.runtimeDetail({...i,runtime_status:{launch_loom:31,launch_native_control:false}});
assert.match(detail,/Recorded launch.*31.*2 GiB.*disabled/);
assert.match(detail,/Requested.*34.*16 GiB.*enabled/);
assert.match(model.runtimeDetail({...i,runtime_status:{launch_loom:34,launch_native_control:true,native_control_socket:'/private/pier/.urb/conn.sock'}}),/\/private\/pier\/\.urb\/conn.sock/);
const all={...c,features:['runtime-settings','native-control-all-ships']};
for (const mode of ['fake','comet','moon','planet','star','galaxy','']) {
  const edit=model.edit(all,{...i,mode});
  assert.ok(!Array.from(model.configure(edit,18081)).some(a=>a.startsWith('--native-control')),'UI preserves '+mode+' CLI opt-in by omission');
}
assert.deepEqual(Array.from(model.runtimeSettingsArgs(c,31)),['--loom','31'],'creation uses the default-off native contract');
assert.doesNotThrow(()=>model.configure({...target,runtimeSettings:false,loom:32,nativeControl:false},18081));
assert.equal(model.storageRoots(null)[0].path,'/srv/urbit/piers');
const roots=[{id:'ssd',path:'/custom'},{id:'default',path:'/srv/urbit/piers'}];
assert.equal(model.storageRoots({snapshot:{piers:{roots}}})[0].id,'default');
assert.equal(roots[0].id,'ssd','default ordering must not mutate the snapshot');
for (const [root,relative,stored] of [['default','fakeships/lab/zod','fakeships/lab/zod'],['ssd','fakeships/lab/zod','@ssd/fakeships/lab/zod'],['hdd','livenet/planets/sampel-palnet/sampel-palnet','@hdd/livenet/planets/sampel-palnet/sampel-palnet']]) {
  const parts=model.splitStoragePath(stored);
  assert.equal(parts.root,root);assert.equal(parts.path,relative);
  assert.equal(model.storagePath(root,relative),stored);
  assert.equal(model.storagePath(root,stored),stored);
}
assert.equal(model.storagePath('hdd','@ssd/fakeships/renamed/zod'),'');
assert.equal(model.storagePath('default','@ssd/fakeships/renamed/zod'),'');
assert.equal(model.storagePath('','fakeships/renamed/zod'),'','removed/unselected storage cannot silently mean default');
console.log('PASS: UI offers loom30–34 and default storage, preserves CLI native settings, and distinguishes historical launches');
