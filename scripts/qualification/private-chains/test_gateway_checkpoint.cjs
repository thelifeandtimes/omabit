'use strict';
const assert=require('node:assert/strict'),fs=require('node:fs'),os=require('node:os'),path=require('node:path'),test=require('node:test'),vm=require('node:vm');
const source=fs.readFileSync(path.join(__dirname,'docker_gateway.cjs'),'utf8');
function section(begin,end){assert.equal(source.split(begin).length,2);assert.equal(source.split(end).length,2);return source.split(begin)[1].split(end)[0];}
// Extract definitions only: never run the entrypoint, chain libraries or keys.
const definitions=section('// BEGIN pure initial gateway contract: independently exercised without startup.\n','// END pure initial gateway contract.')+section('// BEGIN startup checkpoint: filesystem and lifecycle tested without chain code.\n','// END startup checkpoint.');
function contract(filesystem=fs){return vm.runInNewContext(definitions+'\n({observe:observeGatewayStartup,sync:syncGatewayStartup,checkpoint:checkpointGatewayStartup,status:gatewayStartupStatus,reply:gatewayStartupReply,publish:publishGatewayStartup,serve:serveGatewayStartup})',Object.freeze({fs:filesystem,assert,process:Object.freeze({getuid:()=>process.getuid()}),require:name=>{assert.equal(name,'node:path');return path;}}),{timeout:1000});}
function plain(value){return JSON.parse(JSON.stringify(value));}
function fixture(t){const root=fs.mkdtempSync(path.join(os.tmpdir(),'omabit-gateway-checkpoint-'));fs.chmodSync(root,0o700);t.after(()=>fs.rmSync(root,{recursive:true,force:true}));for(const name of ['chain','secrets','piers'])fs.mkdirSync(path.join(root,name),{mode:0o700});fs.writeFileSync(path.join(root,'chain','CURRENT'),'synthetic database',{mode:0o600});fs.writeFileSync(path.join(root,'secrets','test.key'),'synthetic, not a key',{mode:0o600});return root;}
function recording(){const api=Object.create(fs),open=new Set(),syncs=[];api.openSync=(...args)=>{const fd=fs.openSync(...args);open.add(fd);return fd;};api.closeSync=fd=>{fs.closeSync(fd);open.delete(fd);};api.fsyncSync=fd=>{syncs.push(fs.readlinkSync('/proc/self/fd/'+fd));return fs.fsyncSync(fd);};return {api,open,syncs};}
const hash='0x'+'ab'.repeat(32);
function observed(){return {head:'0x50',head_hash:hash,genesis:'synthetic genesis',public_keys:[{ship:'~zod',crypt:'synthetic crypt',auth:'synthetic auth',life:1,suite:1}]};}
const checked=Object.freeze({head:'0x50',head_hash:hash,durable:true,reopened:true});

function rpcFixture(failure){
 const calls=[],keys=[{number:0,ship:'~zod',crypt:'crypt0',auth:'auth0'},{number:1,ship:'~nec',crypt:'crypt1',auth:'auth1'}],foundation={before:{genesis:'genesis'},contracts:{}};
 for(const name of ['Azimuth','B','C','D','E','F','G','H','I'])foundation.contracts[name]={address:name,codeKeccak256:'hash-'+name};
 let heights=0,headers=0;const rpc=async(method,params=[])=>{calls.push({method,params:plain(params)});
  if(method==='eth_chainId')return failure==='chain'?'0x1':'0x539';
  if(method==='eth_blockNumber')return ++heights===2&&failure==='later-height'?'0x51':'0x50';
  if(method==='eth_getBlockByNumber'){if(params[0]==='0x0')return {hash:failure==='genesis'?'different':'genesis'};headers++;return {number:'0x50',hash:failure==='head'||(failure==='later-hash'&&headers===2)?'0x'+'cd'.repeat(32):hash};}
  if(method==='eth_getCode'){assert.equal(params[1],'0x50');return failure==='code'&&params[0]==='I'?'different':params[0];}
  if(method==='eth_call'){assert.equal(params[0].to,'Azimuth');assert.equal(params[1],'0x50');const key=keys[Number(params[0].data)],value={crypt:key.crypt,auth:key.auth,suite:1n,revision:1n};if(key.number===1&&['crypt','auth','suite','revision'].includes(failure))value[failure]=failure==='suite'||failure==='revision'?2n:'different';return value;}
  throw new Error('unexpected RPC: '+method);
 };
 const interfaces={Azimuth:{encodeFunctionData:(method,args)=>{assert.equal(method,'getKeys');return String(args[0]);},decodeFunctionResult:(method,value)=>{assert.equal(method,'getKeys');return value;}}};
 return {calls,keys,run:()=>contract().observe(rpc,foundation,interfaces,{keccak256:bytes=>'hash-'+bytes},keys,hash)};
}
test('startup observation reads every contract and both key rows at the checked head',async()=>{
 const f=rpcFixture(),value=plain(await f.run());assert.equal(value.head,'0x50');assert.equal(value.head_hash,hash);assert.equal(value.genesis,'genesis');assert.deepEqual(value.public_keys,f.keys.map(({number,...key})=>({...key,life:1,suite:1})));assert.equal(f.calls.filter(c=>c.method==='eth_getCode').length,9);assert.equal(f.calls.filter(c=>c.method==='eth_call').length,2);assert.deepEqual(f.calls.slice(-2).map(c=>c.method),['eth_blockNumber','eth_getBlockByNumber']);
});
test('startup observation rejects wrong chain, code, public keys and moving heads',async t=>{
 for(const failure of ['chain','genesis','head','code','crypt','auth','suite','revision','later-height','later-hash'])await t.test(failure,async()=>{await assert.rejects(rpcFixture(failure).run());});
});

test('startup sync covers files before directories without changing bytes or modes',t=>{
 const root=fixture(t),r=recording();contract(r.api).sync(root);assert.equal(r.open.size,0);
 for(const [name,bytes]of [['chain/CURRENT','synthetic database'],['secrets/test.key','synthetic, not a key']]){assert.equal(fs.readFileSync(path.join(root,name),'utf8'),bytes);assert.equal(fs.statSync(path.join(root,name)).mode&0o7777,0o600);assert(r.syncs.indexOf(path.join(root,name))<r.syncs.indexOf(path.dirname(path.join(root,name))));}
 assert.equal(r.syncs.at(-1),root);assert.equal(r.syncs.length,6);assert.equal(fs.statSync(root).mode&0o7777,0o700);
});
test('startup sync refuses unsafe or missing objects without repairing them',async t=>{
 for(const kind of ['symlink','hardlink','public-file','public-directory','missing-root','aliased-root','foreign-owner','foreign-device'])await t.test(kind,t=>{
  const root=fixture(t),file=path.join(root,'chain','CURRENT'),r=recording();let selected=root;
  if(kind==='symlink')fs.symlinkSync('CURRENT',path.join(root,'chain','linked'));
  if(kind==='hardlink')fs.linkSync(file,path.join(root,'chain','linked'));
  if(kind==='public-file')fs.chmodSync(file,0o644);
  if(kind==='public-directory')fs.chmodSync(path.join(root,'chain'),0o755);
  if(kind==='missing-root')selected=path.join(root,'missing');
  if(kind==='aliased-root'){selected=path.join(root,'alias');fs.symlinkSync('chain',selected);}
  if(kind==='foreign-owner'||kind==='foreign-device')r.api.lstatSync=(name,...args)=>{const s=fs.lstatSync(name,...args);if(name===file)s[kind==='foreign-owner'?'uid':'dev']+=1n;return s;};
  assert.throws(()=>contract(r.api).sync(selected));assert.equal(r.open.size,0);assert.equal(fs.readFileSync(file,'utf8'),'synthetic database');
  if(kind==='public-file')assert.equal(fs.statSync(file).mode&0o777,0o644);
  if(kind==='public-directory')assert.equal(fs.statSync(path.join(root,'chain')).mode&0o777,0o755);
 });
});
test('startup sync detects a replaced open file and closes every descriptor',t=>{
 const root=fixture(t),r=recording(),file=path.join(root,'chain','CURRENT');
 r.api.fsyncSync=fd=>{if(fs.readlinkSync('/proc/self/fd/'+fd)===file){fs.renameSync(file,file+'.retained');fs.writeFileSync(file,'replacement',{mode:0o600});}return fs.fsyncSync(fd);};
 assert.throws(()=>contract(r.api).sync(root));assert.equal(r.open.size,0);assert.equal(fs.readFileSync(file+'.retained','utf8'),'synthetic database');assert.equal(fs.readFileSync(file,'utf8'),'replacement');
});
test('startup sync fails on changing directory membership or fsync errors',async t=>{
 for(const kind of ['membership','fsync'])await t.test(kind,t=>{const root=fixture(t),r=recording();r.api.fsyncSync=fd=>{const name=fs.readlinkSync('/proc/self/fd/'+fd);if(name===path.join(root,'chain','CURRENT')){if(kind==='fsync')throw new Error('injected disk failure');fs.writeFileSync(path.join(root,'chain','late'),'retained',{mode:0o600});}return fs.fsyncSync(fd);};assert.throws(()=>contract(r.api).sync(root));assert.equal(r.open.size,0);assert.equal(fs.readFileSync(path.join(root,'chain','CURRENT'),'utf8'),'synthetic database');});
});
test('checkpoint closes, syncs, reopens and freshly observes before the last sync',async()=>{
 const events=[],c=contract();let reads=0;const result=await c.checkpoint(async()=>{events.push('observe'+(++reads));return observed();},async()=>events.push('close'),async()=>events.push('reopen'),()=>events.push('sync'));
 assert.deepEqual(events,['observe1','close','sync','reopen','observe2','sync']);assert.deepEqual(plain(result),checked);assert(Object.isFrozen(result));
});
test('each checkpoint failure prevents every later lifecycle operation',async t=>{
 const order=['observe1','close','sync1','reopen','observe2','sync2'];
 for(const failure of order)await t.test(failure,async()=>{const events=[];let reads=0,syncs=0;function step(name){events.push(name);if(name===failure)throw new Error('injected '+name);}
  await assert.rejects(contract().checkpoint(async()=>{step('observe'+(++reads));return observed();},async()=>step('close'),async()=>step('reopen'),()=>step('sync'+(++syncs))),/injected/);assert.deepEqual(events,order.slice(0,order.indexOf(failure)+1));
 });
});
test('reopened head, genesis and public key changes refuse readiness',async t=>{
 for(const field of ['head','head_hash','genesis','crypt','auth','life','suite'])await t.test(field,async()=>{let reads=0,syncs=0;await assert.rejects(contract().checkpoint(async()=>{const value=observed();if(++reads===2){if(['head','head_hash','genesis'].includes(field))value[field]='changed';else value.public_keys[0][field]='changed';}return value;},async()=>{},async()=>{},()=>syncs++),/reopened startup state differs/);assert.equal(syncs,1);});
});
test('checkpoint snapshots its initial observation before awaited lifecycle work',async()=>{
 const first=observed();let reads=0;const c=await contract().checkpoint(async()=>++reads===1?first:observed(),async()=>{first.head='changed';first.head_hash='changed';first.public_keys[0].auth='changed';},async()=>{},()=>{});assert.deepEqual(plain(c),checked);
});
test('invalid initial head refuses before provider close',async()=>{let closed=false;await assert.rejects(contract().checkpoint(async()=>({...observed(),head:'0x51'}),async()=>{closed=true;},async()=>{},()=>{}));assert.equal(closed,false);});
test('status only publishes checked public metadata',()=>{
 const c=contract();assert.deepEqual(plain(c.status(null)),{format:'omabit-private-gateway-startup-v1',ready:false});assert.deepEqual(plain(c.status({...checked,private:'never returned'})),{format:'omabit-private-gateway-startup-v1',ready:true,...checked});
 for(const value of [{...checked,durable:false},{...checked,reopened:false},{...checked,head:'0x51'}])assert.throws(()=>c.status(value));
 const request={jsonrpc:'2.0',id:'startup',method:'omabit_getStartupStatus',params:{}};assert.equal(c.reply(request,checked).result.ready,true);assert.equal(c.reply({...request,params:undefined},null).result.ready,false);
 for(const params of [null,[],0,'',false,{extra:true}])assert.throws(()=>c.reply({...request,params},checked));
 for(const value of [{...request,jsonrpc:'1.0'},{...request,id:1}])assert.throws(()=>c.reply(value,checked));
 for(const method of ['evm_mine','eth_sendTransaction','getPoint','unknown'])assert.equal(c.reply({...request,method},checked),null);
});
test('publication is private, exclusive and syncs both the file and parent',t=>{
 const root=fixture(t),r=recording(),c=contract(r.api),ready={startupCheckpoint:checked};c.publish(root,ready);const file=path.join(root,'gateway-ready.json');assert.deepEqual(JSON.parse(fs.readFileSync(file,'utf8')),ready);assert.equal(fs.statSync(file).mode&0o7777,0o600);assert.deepEqual(r.syncs,[file,root]);assert.equal(r.open.size,0);assert.throws(()=>c.publish(root,{replacement:true}));assert.deepEqual(JSON.parse(fs.readFileSync(file,'utf8')),ready);
});
test('publication failure retains output but cannot publish in-memory readiness',async t=>{
 for(const target of ['file','parent'])await t.test(target,t=>{const root=fixture(t),r=recording(),c=contract(r.api);r.api.fsyncSync=fd=>{const name=fs.readlinkSync('/proc/self/fd/'+fd);if((target==='parent')===(name===root))throw new Error('injected durability failure');return fs.fsyncSync(fd);};let state=null;assert.throws(()=>{c.publish(root,{startupCheckpoint:checked});state=checked;},/injected/);assert.equal(c.status(state).ready,false);assert.equal(r.open.size,0);assert(fs.existsSync(path.join(root,'gateway-ready.json')));assert.throws(()=>c.publish(root,{}));});
});
test('serving returns readiness only after publication and closes on publication failure',async t=>{
 for(const failure of [null,'listen','file','parent','address'])await t.test(failure||'success',async t=>{
  const root=fixture(t),r=recording(),events=[],c=contract(r.api);let listener;
  const server={listening:false,once:(name,callback)=>{assert.equal(name,'error');listener=callback;},listen:(port,address,callback)=>{assert.equal(port,25432);assert.equal(address,'127.0.0.1');events.push('listen');if(failure==='listen'){listener(new Error('injected listen failure'));return;}server.listening=true;callback();},address:()=>({address:failure==='address'?'0.0.0.0':'127.0.0.1',port:25432}),close:callback=>{events.push('close');server.listening=false;queueMicrotask(callback);},closeAllConnections:()=>events.push('close-connections')};
  r.api.fsyncSync=fd=>{const name=fs.readlinkSync('/proc/self/fd/'+fd);events.push(name===root?'sync-parent':'sync-file');if((failure==='parent'&&name===root)||(failure==='file'&&name!==root))throw new Error('injected publication failure');return fs.fsyncSync(fd);};
  let state=null;const start=async()=>{state=await c.serve(server,root,{startupCheckpoint:checked},checked);};
  if(failure){await assert.rejects(start());assert.equal(state,null);assert.equal(server.listening,false);assert.deepEqual(events.slice(-2),failure==='listen'?['listen']:['close','close-connections']);}
  else{await start();assert.equal(state,checked);assert.equal(server.listening,true);assert.deepEqual(events,['listen','sync-file','sync-parent']);}
  assert.equal(r.open.size,0);
 });
});
test('entrypoint makes newly written files private and checks explicit-head keys',()=>{
 const guard="for(const n of ['net','pid','mnt'])assert.notEqual(fs.readlinkSync('/proc/self/ns/'+n),original[n]);";
 assert(source.indexOf(guard)<source.indexOf('process.umask(0o077)'));assert(source.indexOf('process.umask(0o077)')<source.indexOf("fs.mkdirSync('/lab/secrets'"));
 assert(source.includes("interfaces.Azimuth.encodeFunctionData('getKeys',[key.number])},head]"));assert(source.includes('wallet:{totalAccounts:0}'));assert(source.indexOf('const checkpoint=await checkpointGatewayStartup')<source.indexOf("startupCheckpoint=await serveGatewayStartup(server,'/lab',ready,checkpoint)"));
});
