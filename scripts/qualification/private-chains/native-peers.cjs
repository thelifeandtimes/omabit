'use strict';
const http=require('node:http');
const fs=require('node:fs'),crypto=require('node:crypto'),cp=require('node:child_process'),assert=require('node:assert/strict');
assert.equal(process.env.OMABIT_PRIVATE_CHAIN,'private-chain-peers-v1');assert.equal(process.env.HOME,'/lab/home');
assert.equal(fs.readFileSync('/lab/marker','utf8'),'private-chain-peers-v1\n');assert.equal(JSON.parse(fs.readFileSync('/lab/scope-admission.json','utf8')).admitted,true);assert(!fs.existsSync('/home'));assert(!fs.existsSync('/run/user'));
const original=JSON.parse(fs.readFileSync('/lab/host-namespaces.json','utf8'));for(const n of ['user','net','pid','mnt'])assert.notEqual(fs.readlinkSync('/proc/self/ns/'+n),original[n]);
const binaryHash=crypto.createHash('sha256').update(fs.readFileSync('/vere')).digest('hex');assert.equal(binaryHash,'b5c54439ae5b9f0ff94d75319e67e04f04369776fa5e2180a5ab1a5ebd4c2ff0');
const ganache=require('/tools/node_modules/ganache'),ethers=require('/tools/node_modules/ethers');
const foundation=JSON.parse(fs.readFileSync('/deployment/result.json','utf8')),compiled=JSON.parse(fs.readFileSync('/compiled/compiler-output.json','utf8')),qualified=JSON.parse(fs.readFileSync('/compiled/result.json','utf8'));
assert(foundation.passed&&qualified.passed);assert.equal(crypto.createHash('sha256').update(fs.readFileSync('/compiled/compiler-output.json')).digest('hex'),qualified.compilerOutputSha256);
fs.mkdirSync('/lab/secrets',{mode:0o700});
function evaluate(expression){const r=cp.spawnSync('/vere',['eval','--loom','29'],{input:expression+'\n',encoding:'utf8',timeout:30000,maxBuffer:1024*1024,cwd:'/lab',env:{PATH:'/usr/bin:/bin',HOME:'/lab/home'}});if(r.error||r.status!==0)throw new Error('private evaluator failed');return (r.stdout||'').replace(/\x1b\[[0-9;]*m/g,'').trim();}
function groupHex(h){assert.match(h,/^[0-9a-f]+$/);h=h.replace(/^0+/,'')||'0';const n=h.length%4||4;return h.slice(0,n)+(h.length>n?'.'+h.slice(n).match(/.{4}/g).join('.'):'');}
assert.equal(groupHex('0001'),'1');assert.equal(groupHex('000012345'),'1.2345');assert.equal(groupHex('0000'),'0');assert(evaluate('0x0001').includes('syntax error'));assert.equal(evaluate('0x'+groupHex('0001')),'0x1');console.log('stage: canonical Hoon hex regression passed');
const keys=[];
for(const [number,ship] of [[0,'~zod'],[1,'~nec']]){
 const seed=groupHex(crypto.randomBytes(32).toString('hex'));
 const expression='=+  cic=(pit:nu:crub:crypto 256 0x'+seed+')\n:*  (render-hex-bytes:ethereum 32 (rsh 8 (rsh 3 pub:ex:cic)))\n    (render-hex-bytes:ethereum 32 (end 8 (rsh 3 pub:ex:cic)))\n    (scot %uw (jam [[%2 ~] '+ship+' 0 [1 sec:ex:cic]~]))\n==';
 const out=evaluate(expression);fs.writeFileSync('/lab/secrets/'+number+'.generation-output',out,{mode:0o600});
 const publicKeys=[...out.matchAll(/['"]([0-9a-f]{64})['"]/g)].map(a=>'0x'+a[1]),key=out.match(/~\.(0w[0-9a-zA-Z.~_-]+)/);
 if(publicKeys.length!==2||!key)throw new Error('fresh key output format refused; raw output retained only in private secrets directory');
 fs.writeFileSync('/lab/secrets/'+number+'.key',key[1]+'\n',{mode:0o600});keys.push({number,ship,crypt:publicKeys[0],auth:publicKeys[1],keyfile:key[1]});
}
console.log('stage: fresh-keys-generated');
assert.notEqual(keys[0].auth,keys[1].auth);assert.notEqual(keys[0].crypt,keys[1].crypt);
const publicCommand=JSON.parse(fs.readFileSync('/source/bridge/package.json','utf8')).scripts['pilot:ganache'];const mnemonic=publicCommand.match(/ -m '([a-z ]+)'/)[1];
fs.cpSync('/deployment/chain','/lab/chain',{recursive:true});let provider;const rpc=(method,params=[])=>provider.request({method,params});
const interfaces={};for(const name of ['Azimuth','Ecliptic'])interfaces[name]=new ethers.Interface(compiled.contracts['azimuth/contracts/'+name+'.sol'][name].abi);
let block='latest';async function call(name,method,args=[]){return interfaces[name].decodeFunctionResult(method,await rpc('eth_call',[{to:foundation.contracts[name].address,data:interfaces[name].encodeFunctionData(method,args)},block]));}
async function mutate(name,method,args=[]){const id=await rpc('eth_sendTransaction',[{from:qualified.deployer,to:foundation.contracts[name].address,data:interfaces[name].encodeFunctionData(method,args),gas:'0x5b8d80'}]);const receipt=await rpc('eth_getTransactionReceipt',[id]);assert.equal(receipt.status,'0x1');return receipt;}
function octs(value){const bytes=Buffer.from(JSON.stringify(value));return '['+String(bytes.length).replace(/\B(?=(\d{3})+(?!\d))/g,'.')+' 0x'+groupHex(Buffer.from(bytes).reverse().toString('hex'))+']';}
function verdict(ship,key,reply){const expression='-:(veri:dawn '+ship+' ;;(feed:jael (cue '+key+')) (need (point:take:dawn '+ship+' '+octs(reply)+')) ~)';return evaluate(expression);}
async function getPoint(number,id){const p=(await call('Azimuth','points',[number])),rights=await call('Azimuth','rights',[number]);return {jsonrpc:'2.0',id,result:{ownership:{owner:{address:rights.owner},managementProxy:{address:rights.managementProxy},spawnProxy:{address:rights.spawnProxy},votingProxy:{address:rights.votingProxy},transferProxy:{address:rights.transferProxy}},network:{rift:p.continuityNumber.toString(),keys:{life:p.keyRevisionNumber.toString(),crypt:p.encryptionKey,auth:p.authenticationKey,suite:p.cryptoSuiteVersion.toString()},sponsor:{has:p.hasSponsor,who:Number(p.sponsor)}}}};}
async function bootProbe(){
 const dnsReceipt=await mutate('Ecliptic','setDnsDomains',['galaxies.omabit.invalid','galaxies.omabit.invalid','galaxies.omabit.invalid']);block=await rpc('eth_blockNumber');assert.equal(block,'0x10');
 const ships=new Map();
 const expr='(scot %ux q:czar:give:dawn)',raw=evaluate(expr).match(/0x[0-9a-f.]+/)[0].slice(2).replaceAll('.','');const generated=JSON.parse(Buffer.from(raw.length%2?'0'+raw:raw,'hex').reverse().toString('utf8'));assert.equal(generated.length,256);
 generated.forEach((r,n)=>{assert.equal(r.id,'gal-'+n);ships.set(r.params.ship,n);});
 const calls=[],refused=[],chainCalls=[],watcherProof={ok:false,commands:[]};
 async function respond(request){
  if(!request||request.jsonrpc!=='2.0'||typeof request.id!=='string')throw new Error('unsupported private gateway request');
  calls.push({id:request.id,method:request.method,params:request.params});
  if(request.method==='getPoint'&&request.params&&Object.keys(request.params).length===1&&ships.has(request.params.ship))return getPoint(ships.get(request.params.ship),request.id);
  if(request.method==='getDns'&&(!request.params||Object.keys(request.params).length===0)){const domains=[];for(let n=0;n<3;n++)domains.push((await call('Azimuth','dnsDomains',[n]))[0]);return {jsonrpc:'2.0',id:request.id,result:domains};}
  refused.push({method:request.method,id:request.id});return {jsonrpc:'2.0',id:request.id,error:{code:-32601,message:'Method not supported by private boot fixture'}};
 }
 const server=http.createServer(async(req,res)=>{try{
  if(req.method!=='POST'||!['/','/rpc'].includes(req.url))throw new Error('private POST root or RPC required');let bytes=0,chunks=[];for await(const chunk of req){bytes+=chunk.length;if(bytes>1024*1024)throw new Error('request too large');chunks.push(chunk);}const value=JSON.parse(Buffer.concat(chunks));if(Array.isArray(value)&&value.length>256)throw new Error('batch too large');async function chainReply(v){assert(v&&v.jsonrpc==='2.0'&&['string','number'].includes(typeof v.id));const allowed=['web3_clientVersion','net_version','eth_chainId','eth_blockNumber','eth_getLogs','eth_getBlockByNumber','eth_getBlockByHash','eth_getTransactionReceipt','eth_getTransactionByHash','eth_call','eth_getCode'];assert(allowed.includes(v.method),'read-only chain method admission');assert(chainCalls.length<10000);const row={id:v.id,method:v.method,params:v.params||[]};chainCalls.push(row);try{const result=await rpc(v.method,v.params||[]);row.ok=true;return {jsonrpc:'2.0',id:v.id,result};}catch(e){row.ok=false;return {jsonrpc:'2.0',id:v.id,error:{code:-32000,message:'private chain read failed'}};}}const reply=req.url==='/rpc'?chainReply:respond;const answer=Array.isArray(value)?await Promise.all(value.map(reply)):await reply(value);res.writeHead(200,{'Content-Type':'application/json'});res.end(JSON.stringify(answer));
 }catch(e){res.writeHead(400);res.end(JSON.stringify({error:e.message}));}});
 await new Promise((resolve,reject)=>{server.once('error',reject);server.listen(0,'127.0.0.1',resolve);});const address=server.address();assert.equal(address.address,'127.0.0.1');
 for(let n=0;n<64;n++)await rpc('evm_mine');
 const peers=[];let bytes=0;const output=fs.openSync('/lab/peer-driver.log','w',0o600);
 const child=cp.spawn('/usr/bin/python3',['-B','-c',"import os,json;from pathlib import Path;import peer_probe;raise SystemExit(peer_probe.exercise({'unit':json.loads(Path('/lab/command.json').read_text())['unit'],'application_state':True}))"],{cwd:'/lab',env:{...process.env,OMABIT_PRIVATE_GATEWAY:'http://127.0.0.1:'+address.port},stdio:['ignore','pipe','pipe']});
 function capture(chunk){bytes+=chunk.length;if(bytes>8*1024*1024){child.kill('SIGTERM');return;}fs.writeSync(output,chunk);const s=chunk.toString('utf8');if(s.startsWith('PEER_PROBE '))console.log(s.trim());}
 child.stdout.on('data',capture);child.stderr.on('data',capture);
 try{
  const code=await new Promise((resolve,reject)=>{child.once('error',reject);child.once('exit',(code,signal)=>resolve({code,signal}));});assert.deepEqual(code,{code:0,signal:null});assert(bytes<=8*1024*1024);
  const proof=JSON.parse(fs.readFileSync('/lab/peer-result.json'));assert(proof.ok&&proof.transport_passed&&proof.restart_same_pier&&proof.restart_kept_sessions&&proof.cross_ship_sessions_refused);assert.equal(proof.attempts.length,7);assert.equal(proof.attempts.filter(x=>x.state==='acknowledged').length,6);assert.equal(proof.attempts.filter(x=>x.state==='rejected').length,1);assert.deepEqual(proof.clean_stops,['nec','nec','zod']);assert.deepEqual(proof.stop_records.map(x=>[x.ship,x.mode,x.passed,x.exit]),[['nec','lens-exit',true,0],['nec','sigterm',true,0],['zod','lens-exit',true,0]]);assert(proof.stop_records.every(x=>!x.forced));const app=proof.application_state;assert(app.ok&&app.state_survived&&app.queued_update_survived&&app.existing_subscription_survived&&app.fresh_subscription_reconciled&&!app.writes_replayed);assert.equal(app.writes.length,4);assert(app.writes.every(x=>x.state==='acknowledged'));assert.equal(new Set(app.writes.map(x=>x.entry)).size,4);assert.equal(app.channels.length,2);assert.deepEqual(app.channels_deleted,app.channels);assert.equal(app.observations.filter(x=>x.via==='subscription').length,6);assert.equal(refused.length,0);
  const databases=[];for(const ship of ['zod','nec']){const root='/lab/piers/private-'+ship+'/'+ship+'/.urb/log';for(const ent of fs.readdirSync(root,{withFileTypes:true})){const path=ent.isDirectory()&&/^0i[0-9]+$/.test(ent.name)?root+'/'+ent.name+'/data.mdb':ent.name==='data.mdb'?root+'/data.mdb':null;if(path){const raw=fs.readFileSync(path);databases.push({path:path.replace('/lab/',''),bytes:raw.length,sha256:crypto.createHash('sha256').update(raw).digest('hex')});}}}
  return {passed:true,scope:'two fresh native keyed private-chain peers; authenticated messages and receiver restart',proof,databases,calls,refused,chainCalls,dnsReceipt,privateGatewayAddress:address,peerNetworkingQualified:true,dockerDevnetQualified:false,experimentalRuntimeOnly:false,officialRuntimeLensShutdown:true,applicationStateQualified:true,normalFreshKeyedSigtermQualified:false};
 }finally{fs.closeSync(output);fs.writeFileSync('/lab/gateway-calls.json',JSON.stringify({calls,refused,chainCalls},null,2)+'\n');await new Promise(resolve=>server.close(resolve));}

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
const batch=[];for(let n=0;n<256;n++)batch.push(await getPoint(n,'gal-'+n));console.log('stage: galaxy-chain-reads-complete');assert.equal(evaluate('(lent ~(tap by (need (czar:take:dawn '+octs(batch)+'))))'),'2');
const dns={jsonrpc:'2.0',id:'turf',result:[]};for(let n=0;n<3;n++)dns.result.push((await call('Azimuth','dnsDomains',[n]))[0]);assert.equal(evaluate('?=(^ (turf:take:dawn '+octs(dns)+'))'),'%.y');
assert.equal(await rpc('eth_blockNumber'),block);const finalBlock=await rpc('eth_getBlockByNumber',[block,false]);const native=await bootProbe();await provider.disconnect();provider=null;
fs.writeFileSync('/lab/public-responses.json',JSON.stringify({responses,batch,dns},null,2)+'\n');fs.writeFileSync('/lab/result.json',JSON.stringify({passed:true,scope:'official runtime private chain native peers with explicit Lens shutdown; Docker not qualified',native,binaryHash,foundationGenesis:foundation.before.genesis,foundationBlock:foundation.before.latest,block,blockHash:finalBlock.hash,cases,publicKeys:keys.map(({keyfile,...v})=>v),receipts,galaxies:256,keyedGalaxies:2,dnsAccepted:true,secretDirectoryExcludedFromReports:'secrets'},null,2)+'\n');console.log('PASS: official keyed peers and persistent state/subscription recovery with explicit Lens initial shutdown');}
run().catch(async e=>{fs.writeFileSync('/lab/failure.txt',e.stack+'\n');console.error(e.message);if(provider)await provider.disconnect();process.exitCode=1;});
