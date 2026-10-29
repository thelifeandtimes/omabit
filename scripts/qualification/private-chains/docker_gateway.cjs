// Disposable Docker gateway only; never a public RPC or production key service.
'use strict';
const http=require('node:http');
const fs=require('node:fs'),crypto=require('node:crypto'),cp=require('node:child_process'),assert=require('node:assert/strict');
assert.equal(process.env.OMABIT_PRIVATE_CHAIN,'private-chain-docker-gateway-v1');assert.equal(process.env.HOME,'/lab/home');
assert.equal(fs.readFileSync('/lab/marker','utf8'),'private-chain-docker-gateway-v1\n');
assert(fs.existsSync('/.dockerenv'));assert.equal(fs.readFileSync('/sys/class/dmi/id/sys_vendor','utf8').trim(),'QEMU');
// BEGIN pure initial gateway contract: independently exercised without startup.
function initialGatewayInterfaces(mode, names) {
 if(!Array.isArray(names)||names.some(n=>typeof n!=='string'))throw new Error('initial gateway interface evidence required');
 const expected=mode===undefined?['lo']:mode==='published-v1'?['eth0','lo']:null;
 if(!expected||JSON.stringify([...names].sort())!==JSON.stringify(expected))throw new Error('initial gateway network mode differs');
 return mode===undefined?'none':'published-v1';
}
function initialGatewayHead(block, header) {
 if(block!=='0x50'||!header||Array.isArray(header)||header.number!==block||typeof header.hash!=='string'||!/^0x[0-9a-f]{64}$/.test(header.hash)||header.hash==='0x'+'0'.repeat(64))throw new Error('initial gateway head observation differs');
 return header.hash;
}
// END pure initial gateway contract.
// BEGIN fresh chain copy: exercised with disposable filesystem fixtures only.
function copyFreshGatewayChain(source,destination) {
 const path=require('node:path');
 assert.equal(path.resolve(source),source);assert.equal(path.resolve(destination),destination);
 assert(source!==destination&&!source.startsWith(destination+'/')&&!destination.startsWith(source+'/'));
 const sync=(name,directory)=>{const fd=fs.openSync(name,fs.constants.O_RDONLY|fs.constants.O_NOFOLLOW|(directory?fs.constants.O_DIRECTORY:0));try{fs.fsyncSync(fd);}finally{fs.closeSync(fd);}};
 let files=0,bytes=0;
 function copy(from,to) {
  const info=fs.lstatSync(from);assert(info.isDirectory()&&!info.isSymbolicLink());
  // Exclusivity is the retry fence. Never repair or reuse an existing chain.
  fs.mkdirSync(to,{mode:0o700});
  for(const name of fs.readdirSync(from).sort()) {
   const input=path.join(from,name),output=path.join(to,name),entry=fs.lstatSync(input);
   if(entry.isDirectory()){copy(input,output);continue;}
   assert(entry.isFile()&&entry.nlink===1,'foundation contains a linked or unsupported file');
   files++;bytes+=entry.size;assert(files<=40000&&bytes<=1024*1024*1024);
   fs.copyFileSync(input,output,fs.constants.COPYFILE_EXCL);
   // Sealed input modes intentionally stay read-only. Only this newly created
   // private database copy receives write permission for LevelDB.
   fs.chmodSync(output,0o600);sync(output,false);
  }
  sync(to,true);
 }
 copy(source,destination);sync(path.dirname(destination),true);
}
// END fresh chain copy.
// BEGIN startup checkpoint: filesystem and lifecycle tested without chain code.
async function observeGatewayStartup(rpc,foundation,interfaces,ethers,keys,readyHead) {
 assert.equal(await rpc('eth_chainId'),'0x539');assert.equal((await rpc('eth_getBlockByNumber',['0x0',false])).hash,foundation.before.genesis);
 const head=await rpc('eth_blockNumber'),header=await rpc('eth_getBlockByNumber',['latest',false]);assert.equal(initialGatewayHead(head,header),readyHead);
 for(const v of Object.values(foundation.contracts))assert.equal(ethers.keccak256(await rpc('eth_getCode',[v.address,head])),v.codeKeccak256);
 const observed=[];
 for(const key of keys){const values=interfaces.Azimuth.decodeFunctionResult('getKeys',await rpc('eth_call',[{to:foundation.contracts.Azimuth.address,data:interfaces.Azimuth.encodeFunctionData('getKeys',[key.number])},head]));assert.equal(values.crypt,key.crypt);assert.equal(values.auth,key.auth);assert.equal(values.suite,1n);assert.equal(values.revision,1n);observed.push({ship:key.ship,crypt:values.crypt,auth:values.auth,life:Number(values.revision),suite:Number(values.suite)});}
 assert.equal(await rpc('eth_blockNumber'),head);assert.equal((await rpc('eth_getBlockByNumber',['latest',false])).hash,header.hash);
 return {head,head_hash:header.hash,genesis:foundation.before.genesis,public_keys:observed};
}
function syncGatewayStartup(root) {
 const path=require('node:path');assert.equal(path.resolve(root),root);
 const base=fs.lstatSync(root,{bigint:true});assert(base.isDirectory());
 const uid=BigInt(process.getuid());let entries=0,bytes=0n;
 const same=(a,b)=>a.dev===b.dev&&a.ino===b.ino&&a.mode===b.mode&&a.uid===b.uid&&a.nlink===b.nlink&&a.size===b.size&&a.mtimeNs===b.mtimeNs;
 function visit(name,depth) {
  assert(depth<=32&&++entries<=40000);
  const before=fs.lstatSync(name,{bigint:true}),directory=before.isDirectory();
  assert(before.dev===base.dev&&before.uid===uid&&!before.isSymbolicLink());
  assert(directory||(before.isFile()&&before.nlink===1n));
  assert.equal(before.mode&0o7777n,directory?0o700n:0o600n);
  if(!directory){bytes+=before.size;assert(bytes<=1024n*1024n*1024n);}
  const fd=fs.openSync(name,fs.constants.O_RDONLY|fs.constants.O_NOFOLLOW|fs.constants.O_NONBLOCK|(directory?fs.constants.O_DIRECTORY:0));
  try {
   assert(same(before,fs.fstatSync(fd,{bigint:true})));
   if(directory){const names=fs.readdirSync(name).sort();for(const child of names)visit(path.join(name,child),depth+1);assert.deepEqual(fs.readdirSync(name).sort(),names);}
   fs.fsyncSync(fd);
   assert(same(before,fs.fstatSync(fd,{bigint:true}))&&same(before,fs.lstatSync(name,{bigint:true})));
  }finally{fs.closeSync(fd);}
 }
 visit(root,0);
}
async function checkpointGatewayStartup(observe,close,reopen,syncState) {
 const before=await observe();initialGatewayHead(before.head,{number:before.head,hash:before.head_hash});
 const expected=JSON.stringify(before),head=before.head,head_hash=before.head_hash;
 await close();syncState();await reopen();
 const after=await observe();assert.equal(JSON.stringify(after),expected,'reopened startup state differs');
 syncState();
 return Object.freeze({head,head_hash,durable:true,reopened:true});
}
function gatewayStartupStatus(checkpoint) {
 if(checkpoint===null)return {format:'omabit-private-gateway-startup-v1',ready:false};
 assert(checkpoint.durable===true&&checkpoint.reopened===true);
 initialGatewayHead(checkpoint.head,{number:checkpoint.head,hash:checkpoint.head_hash});
 return {format:'omabit-private-gateway-startup-v1',ready:true,head:checkpoint.head,head_hash:checkpoint.head_hash,durable:true,reopened:true};
}
function gatewayStartupReply(request,checkpoint) {
 if(request?.method!=='omabit_getStartupStatus')return null;
 assert(request.jsonrpc==='2.0'&&typeof request.id==='string');
 assert(request.params===undefined||(request.params!==null&&typeof request.params==='object'&&!Array.isArray(request.params)&&Object.keys(request.params).length===0));
 return {jsonrpc:'2.0',id:request.id,result:gatewayStartupStatus(checkpoint)};
}
function publishGatewayStartup(root,ready) {
 const path=require('node:path');
 const fd=fs.openSync(path.join(root,'gateway-ready.json'),fs.constants.O_WRONLY|fs.constants.O_CREAT|fs.constants.O_EXCL|fs.constants.O_NOFOLLOW,0o600);
 try{fs.writeFileSync(fd,JSON.stringify(ready,null,2)+'\n');fs.fsyncSync(fd);}finally{fs.closeSync(fd);}
 const directory=fs.openSync(root,fs.constants.O_RDONLY|fs.constants.O_DIRECTORY|fs.constants.O_NOFOLLOW);
 try{fs.fsyncSync(directory);}finally{fs.closeSync(directory);}
}
async function serveGatewayStartup(server,root,ready,checkpoint) {
 try {
  await new Promise((resolve,reject)=>{server.once('error',reject);server.listen(25432,'127.0.0.1',resolve);});
  const address=server.address();assert.equal(address.address,'127.0.0.1');assert.equal(address.port,25432);
  publishGatewayStartup(root,{...ready,address});return checkpoint;
 }catch(error){
  if(server.listening)await new Promise(resolve=>{server.close(resolve);server.closeAllConnections();});
  throw error;
 }
}
// END startup checkpoint.
const initialNetwork=initialGatewayInterfaces(process.env.OMABIT_PRIVATE_CHAIN_NETWORK,fs.readdirSync('/sys/class/net'));
const original=JSON.parse(fs.readFileSync('/lab/guest-namespaces.json','utf8'));
for(const n of ['net','pid','mnt'])assert.notEqual(fs.readlinkSync('/proc/self/ns/'+n),original[n]);
// Apply only to this already-isolated gateway and its children. LevelDB's new
// files must be private too, not just the initial copied foundation.
process.umask(0o077);
const binaryHash=crypto.createHash('sha256').update(fs.readFileSync('/vere')).digest('hex');assert.equal(binaryHash,'b5c54439ae5b9f0ff94d75319e67e04f04369776fa5e2180a5ab1a5ebd4c2ff0');
assert.equal(process.env.LIBC,'glibc');assert.equal(process.env.PREBUILDS_ONLY,'1');
assert.match(process.report.getReport().header.glibcVersionRuntime,/^[0-9]+\.[0-9]+/);
const addon=require('/tools/node_modules/ganache/node_modules/node-gyp-build').path('/tools/node_modules/ganache/node_modules/leveldown');
assert.equal(addon,'/tools/node_modules/ganache/node_modules/leveldown/prebuilds/linux-x64/node.napi.glibc.node');
assert.equal(crypto.createHash('sha256').update(fs.readFileSync(addon)).digest('hex'),'35164284059b66d0de99a33674d99ff2fafeb27b928e9014adb6a06838d7c05a');
const ganache=require('/tools/node_modules/ganache'),ethers=require('/tools/node_modules/ethers');
const foundation=JSON.parse(fs.readFileSync('/deployment/result.json','utf8')),compiled=JSON.parse(fs.readFileSync('/compiled/compiler-output.json','utf8')),qualified=JSON.parse(fs.readFileSync('/compiled/result.json','utf8'));
assert(foundation.passed&&qualified.passed);assert.equal(crypto.createHash('sha256').update(fs.readFileSync('/compiled/compiler-output.json')).digest('hex'),qualified.compilerOutputSha256);
fs.mkdirSync('/lab/secrets',{mode:0o700});
function evaluate(expression,timeout=30000){assert([30000,120000].includes(timeout));const started=performance.now();const r=cp.spawnSync('/vere',['eval','--loom','29'],{input:expression+'\n',encoding:'utf8',timeout,maxBuffer:1024*1024,cwd:'/lab',env:{PATH:'/usr/bin:/bin',HOME:'/lab/home'}});if(r.error||r.status!==0){fs.writeFileSync('/lab/evaluator-failure.json',JSON.stringify({timeout_ms:timeout,elapsed_ms:Math.round(performance.now()-started),status:r.status,signal:r.signal,error_code:r.error?r.error.code:null,stdout_bytes:Buffer.byteLength(r.stdout||''),stderr_bytes:Buffer.byteLength(r.stderr||''),raw_output_exported:false})+'\n',{mode:0o600});throw new Error('private evaluator failed; bounded diagnostic retained');}return (r.stdout||'').replace(/\x1b\[[0-9;]*m/g,'').trim();}
function groupHex(h){assert.match(h,/^[0-9a-f]+$/);h=h.replace(/^0+/,'')||'0';const n=h.length%4||4;return h.slice(0,n)+(h.length>n?'.'+h.slice(n).match(/.{4}/g).join('.'):'');}
assert.equal(groupHex('0001'),'1');assert.equal(groupHex('000012345'),'1.2345');assert.equal(groupHex('0000'),'0');assert(evaluate('0x0001').includes('syntax error'));assert.equal(evaluate('0x'+groupHex('0001')),'0x1');console.log('stage: canonical Hoon hex regression passed');
const keys=[];
for(const [number,ship] of [[0,'~zod'],[1,'~nec']]){
 const seed=groupHex(crypto.randomBytes(32).toString('hex'));
 const expression='=+  cic=(pit:nu:crub:crypto 256 0x'+seed+')\n:*  (render-hex-bytes:ethereum 32 (rsh 8 (rsh 3 pub:ex:cic)))\n    (render-hex-bytes:ethereum 32 (end 8 (rsh 3 pub:ex:cic)))\n    (scot %uw (jam [[%2 ~] '+ship+' 0 [1 sec:ex:cic]~]))\n==';
 const out=evaluate(expression);fs.writeFileSync('/lab/secrets/'+number+'.generation-output',out,{mode:0o600});
 const publicKeys=[...out.matchAll(/['"]([0-9a-f]{64})['"]/g)].map(a=>'0x'+a[1]),key=out.match(/~\.(0w[0-9a-zA-Z.~_-]+)/);
 if(publicKeys.length!==2||!key)throw new Error('fresh key output format refused; raw output retained only in private secrets directory');
 fs.writeFileSync('/lab/secrets/'+ship.slice(1)+'.key',key[1]+'\n',{mode:0o600});keys.push({number,ship,crypt:publicKeys[0],auth:publicKeys[1],keyfile:key[1]});
}
console.log('stage: fresh-keys-generated');
assert.notEqual(keys[0].auth,keys[1].auth);assert.notEqual(keys[0].crypt,keys[1].crypt);
const publicCommand=JSON.parse(fs.readFileSync('/source/bridge/package.json','utf8')).scripts['pilot:ganache'];const mnemonic=publicCommand.match(/ -m '([a-z ]+)'/)[1];
copyFreshGatewayChain('/deployment/chain','/lab/chain');let provider;const rpc=(method,params=[])=>provider.request({method,params});
const interfaces={};for(const name of ['Azimuth','Ecliptic'])interfaces[name]=new ethers.Interface(compiled.contracts['azimuth/contracts/'+name+'.sol'][name].abi);
let block='latest';async function call(name,method,args=[]){return interfaces[name].decodeFunctionResult(method,await rpc('eth_call',[{to:foundation.contracts[name].address,data:interfaces[name].encodeFunctionData(method,args)},block]));}
async function mutate(name,method,args=[]){const id=await rpc('eth_sendTransaction',[{from:qualified.deployer,to:foundation.contracts[name].address,data:interfaces[name].encodeFunctionData(method,args),gas:'0x5b8d80'}]);const receipt=await rpc('eth_getTransactionReceipt',[id]);assert.equal(receipt.status,'0x1');return receipt;}
function octs(value){const bytes=Buffer.from(JSON.stringify(value));return '['+String(bytes.length).replace(/\B(?=(\d{3})+(?!\d))/g,'.')+' 0x'+groupHex(Buffer.from(bytes).reverse().toString('hex'))+']';}
function verdict(ship,key,reply){const expression='-:(veri:dawn '+ship+' ;;(feed:jael (cue '+key+')) (need (point:take:dawn '+ship+' '+octs(reply)+')) ~)';return evaluate(expression);}
async function getPoint(number,id){const p=(await call('Azimuth','points',[number])),rights=await call('Azimuth','rights',[number]);return {jsonrpc:'2.0',id,result:{ownership:{owner:{address:rights.owner},managementProxy:{address:rights.managementProxy},spawnProxy:{address:rights.spawnProxy},votingProxy:{address:rights.votingProxy},transferProxy:{address:rights.transferProxy}},network:{rift:p.continuityNumber.toString(),keys:{life:p.keyRevisionNumber.toString(),crypt:p.encryptionKey,auth:p.authenticationKey,suite:p.cryptoSuiteVersion.toString()},sponsor:{has:p.hasSponsor,who:Number(p.sponsor)}}}};}
async function gateway(){
 const dnsReceipt=await mutate('Ecliptic','setDnsDomains',['galaxies.omabit.invalid','galaxies.omabit.invalid','galaxies.omabit.invalid']);block=await rpc('eth_blockNumber');assert.equal(block,'0x10');
 const ships=new Map();
 const expr='(scot %ux q:czar:give:dawn)',raw=evaluate(expr).match(/0x[0-9a-f.]+/)[0].slice(2).replaceAll('.','');const generated=JSON.parse(Buffer.from(raw.length%2?'0'+raw:raw,'hex').reverse().toString('utf8'));assert.equal(generated.length,256);
 generated.forEach((r,n)=>{assert.equal(r.id,'gal-'+n);ships.set(r.params.ship,n);});
 const calls=[],refused=[],chainCalls=[],watcherProof={ok:false,commands:[]};
 let startupCheckpoint=null;
 async function respond(request){
  if(!request||request.jsonrpc!=='2.0'||typeof request.id!=='string')throw new Error('unsupported private gateway request');
  calls.push({id:request.id,method:request.method,params:request.params});
  const startupReply=gatewayStartupReply(request,startupCheckpoint);if(startupReply)return startupReply;
  if(request.method==='getPoint'&&request.params&&Object.keys(request.params).length===1&&ships.has(request.params.ship))return getPoint(ships.get(request.params.ship),request.id);
  if(request.method==='getDns'&&(!request.params||Object.keys(request.params).length===0)){const domains=[];for(let n=0;n<3;n++)domains.push((await call('Azimuth','dnsDomains',[n]))[0]);return {jsonrpc:'2.0',id:request.id,result:domains};}
  refused.push({method:request.method,id:request.id});return {jsonrpc:'2.0',id:request.id,error:{code:-32601,message:'Method not supported by private boot fixture'}};
 }
 const server=http.createServer(async(req,res)=>{try{
  if(req.method!=='POST'||!['/','/rpc'].includes(req.url))throw new Error('private POST root or RPC required');let bytes=0,chunks=[];for await(const chunk of req){bytes+=chunk.length;if(bytes>1024*1024)throw new Error('request too large');chunks.push(chunk);}const value=JSON.parse(Buffer.concat(chunks));if(Array.isArray(value)&&value.length>256)throw new Error('batch too large');async function chainReply(v){assert(v&&v.jsonrpc==='2.0'&&['string','number'].includes(typeof v.id));const allowed=['web3_clientVersion','net_version','eth_chainId','eth_blockNumber','eth_getLogs','eth_getBlockByNumber','eth_getBlockByHash','eth_getTransactionReceipt','eth_getTransactionByHash','eth_call','eth_getCode'];assert(allowed.includes(v.method),'read-only chain method admission');assert(chainCalls.length<10000);const row={id:v.id,method:v.method,params:v.params||[]};chainCalls.push(row);try{const result=await rpc(v.method,v.params||[]);row.ok=true;return {jsonrpc:'2.0',id:v.id,result};}catch(e){row.ok=false;return {jsonrpc:'2.0',id:v.id,error:{code:-32000,message:'private chain read failed'}};}}const reply=req.url==='/rpc'?chainReply:respond;const answer=Array.isArray(value)?await Promise.all(value.map(reply)):await reply(value);res.writeHead(200,{'Content-Type':'application/json'});res.end(JSON.stringify(answer));
 }catch(e){res.writeHead(400);res.end(JSON.stringify({error:e.message}));}});
 for(let n=0;n<64;n++)await rpc('evm_mine');
 const stopped=new Promise(resolve=>process.once('SIGTERM',resolve));
 const readyBlock=await rpc('eth_blockNumber');
 const readyHead=initialGatewayHead(readyBlock,await rpc('eth_getBlockByNumber',['latest',false]));
 const observeStartup=()=>observeGatewayStartup(rpc,foundation,interfaces,ethers,keys,readyHead);
 const checkpoint=await checkpointGatewayStartup(observeStartup,async()=>{await provider.disconnect();provider=null;},async()=>{provider=ganache.provider({chain:foundation.chainSettings,wallet:{totalAccounts:0},database:{dbPath:'/lab/chain'},miner:foundation.miner,logging:{quiet:true}});},()=>syncGatewayStartup('/lab'));
 const ready={passed:true,chainId:await rpc('eth_chainId'),genesis:foundation.before.genesis,
  block:readyBlock,headHash:readyHead,initialNetwork,publicKeys:keys.map(({keyfile,...v})=>v),dnsReceipt,
  namespace:fs.readlinkSync('/proc/self/ns/net'),externalRpcMutationsAllowed:false,startupCheckpoint:checkpoint};
 assert.equal(ready.chainId,'0x539');assert.equal(ready.block,'0x50');
 startupCheckpoint=await serveGatewayStartup(server,'/lab',ready,checkpoint);const address=server.address();ready.address=address;
 console.log('GATEWAY_READY '+JSON.stringify({port:address.port,chainId:ready.chainId,keys:2}));
 await stopped;await new Promise(resolve=>server.close(resolve));
 fs.writeFileSync('/lab/gateway-calls.json',JSON.stringify({calls,refused,chainCalls},null,2)+'\n');
 return {passed:true,scope:'private Docker chain-backed boot gateway; ship lifecycle tested separately',
  calls,refused,chainCalls,dnsReceipt,privateGatewayAddress:address,head:await rpc('eth_blockNumber'),headHash:(await rpc('eth_getBlockByNumber',['latest',false])).hash,
  initialKeySetup:ready,externalRpcMutationsAllowed:false,signalShutdown:true};
}

async function run(){provider=ganache.provider({chain:foundation.chainSettings,wallet:{mnemonic,totalAccounts:2,defaultBalance:1000},database:{dbPath:'/lab/chain'},miner:foundation.miner,logging:{quiet:true}});
assert.equal(await rpc('eth_chainId'),foundation.before.chainId);assert.equal((await rpc('eth_getBlockByNumber',['0x0',false])).hash,foundation.before.genesis);assert.equal((await rpc('eth_getBlockByNumber',['latest',false])).hash,foundation.before.latest);
for(const v of Object.values(foundation.contracts))assert.equal(ethers.keccak256(await rpc('eth_getCode',[v.address,'latest'])),v.codeKeccak256);
const receipts=[];for(const key of keys){const id=await rpc('eth_sendTransaction',[{from:qualified.deployer,to:foundation.contracts.Ecliptic.address,data:interfaces.Ecliptic.encodeFunctionData('configureKeys',[key.number,key.crypt,key.auth,1,false]),gas:'0x5b8d80'}]);const r=await rpc('eth_getTransactionReceipt',[id]);assert.equal(r.status,'0x1');receipts.push(r);}
console.log('stage: keys-configured-on-private-chain');
block=await rpc('eth_blockNumber');assert.equal(block,'0xf');const cases=[],responses=[];
for(const key of keys){const reply=await getPoint(key.number,'');assert.equal(reply.result.network.keys.crypt,key.crypt);assert.equal(reply.result.network.keys.auth,key.auth);assert.equal(reply.result.network.keys.life,'1');responses.push(reply);
 const correct=verdict(key.ship,key.keyfile,reply);assert.equal(correct,'%.y');
 const badKey=structuredClone(reply);badKey.result.network.keys.auth='0x'+'00'.repeat(32);assert.equal(verdict(key.ship,key.keyfile,badKey),'%.n');
 const badLife=structuredClone(reply);badLife.result.network.keys.life='2';assert.equal(verdict(key.ship,key.keyfile,badLife),'%.n');
 cases.push({ship:key.ship,correct:true,wrongPublicKeyRefused:true,wrongLifeRefused:true});}
console.log('stage: both-key-verdicts-checked');
const batch=[];for(let n=0;n<256;n++)batch.push(await getPoint(n,'gal-'+n));console.log('stage: galaxy-chain-reads-complete');assert.equal(evaluate('(lent ~(tap by (need (czar:take:dawn '+octs(batch)+'))))',120000),'2');
const dns={jsonrpc:'2.0',id:'turf',result:[]};for(let n=0;n<3;n++)dns.result.push((await call('Azimuth','dnsDomains',[n]))[0]);assert.equal(evaluate('?=(^ (turf:take:dawn '+octs(dns)+'))'),'%.y');
assert.equal(await rpc('eth_blockNumber'),block);const finalBlock=await rpc('eth_getBlockByNumber',[block,false]);const gatewayResult=await gateway();await provider.disconnect();provider=null;
fs.writeFileSync('/lab/public-responses.json',JSON.stringify({responses,batch,dns},null,2)+'\n');fs.writeFileSync('/lab/result.json',JSON.stringify({passed:true,scope:'private Docker chain gateway; independent peer proof required',gateway:gatewayResult,binaryHash,foundationGenesis:foundation.before.genesis,foundationBlock:foundation.before.latest,block,blockHash:finalBlock.hash,cases,publicKeys:keys.map(({keyfile,...v})=>v),receipts,galaxies:256,keyedGalaxies:2,dnsAccepted:true,secretDirectoryExcludedFromReports:'secrets'},null,2)+'\n');console.log('PASS: private Docker gateway stopped with retained chain and public key evidence');}
run().catch(async e=>{fs.writeFileSync('/lab/failure.txt',e.stack+'\n');console.error(e.message);if(provider)await provider.disconnect();process.exitCode=1;});
