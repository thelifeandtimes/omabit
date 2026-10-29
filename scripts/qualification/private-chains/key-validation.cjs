'use strict';
const fs=require('node:fs'),crypto=require('node:crypto'),cp=require('node:child_process'),assert=require('node:assert/strict');
assert.equal(process.env.OMABIT_PRIVATE_CHAIN,'private-chain-key-validation-v1');assert.equal(process.env.HOME,'/lab/home');
assert.equal(fs.readFileSync('/lab/marker','utf8'),'private-chain-key-validation-v1\n');assert.equal(JSON.parse(fs.readFileSync('/lab/scope-admission.json','utf8')).admitted,true);assert(!fs.existsSync('/home'));assert(!fs.existsSync('/run/user'));
const original=JSON.parse(fs.readFileSync('/lab/host-namespaces.json','utf8'));for(const n of ['user','net','pid','mnt'])assert.notEqual(fs.readlinkSync('/proc/self/ns/'+n),original[n]);
const binaryHash=crypto.createHash('sha256').update(fs.readFileSync('/vere')).digest('hex');assert.equal(binaryHash,'b5c54439ae5b9f0ff94d75319e67e04f04369776fa5e2180a5ab1a5ebd4c2ff0');
const ganache=require('/tools/node_modules/ganache'),ethers=require('/tools/node_modules/ethers');
const foundation=JSON.parse(fs.readFileSync('/deployment/result.json','utf8')),compiled=JSON.parse(fs.readFileSync('/compiled/compiler-output.json','utf8')),qualified=JSON.parse(fs.readFileSync('/compiled/result.json','utf8'));
assert(foundation.passed&&qualified.passed);assert.equal(crypto.createHash('sha256').update(fs.readFileSync('/compiled/compiler-output.json')).digest('hex'),qualified.compilerOutputSha256);
fs.mkdirSync('/lab/secrets',{mode:0o700});
function evaluate(expression){const r=cp.spawnSync('/vere',['eval','--loom','29'],{input:expression+'\n',encoding:'utf8',timeout:30000,maxBuffer:1024*1024,cwd:'/lab',env:{PATH:'/usr/bin:/bin',HOME:'/lab/home'}});if(r.error||r.status!==0)throw new Error('private evaluator failed');return (r.stdout||'').replace(/\x1b\[[0-9;]*m/g,'').trim();}
function groupHex(h){assert.match(h,/^[0-9a-f]+$/);h=h.replace(/^0+/,'')||'0';const n=h.length%4||4;return h.slice(0,n)+(h.length>n?'.'+h.slice(n).match(/.{4}/g).join('.'):'');}
assert.equal(groupHex('0001'),'1');assert.equal(groupHex('000012345'),'1.2345');assert.equal(groupHex('0000'),'0');assert(evaluate('0x0001').includes('syntax error'));assert.equal(evaluate('0x'+groupHex('0001')),'0x1');
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
function octs(value){const bytes=Buffer.from(JSON.stringify(value));return '['+String(bytes.length).replace(/\B(?=(\d{3})+(?!\d))/g,'.')+' 0x'+groupHex(Buffer.from(bytes).reverse().toString('hex'))+']';}
function verdict(ship,key,reply){const expression='-:(veri:dawn '+ship+' ;;(feed:jael (cue '+key+')) (need (point:take:dawn '+ship+' '+octs(reply)+')) ~)';return evaluate(expression);}
async function getPoint(number,id){const p=(await call('Azimuth','points',[number])),rights=await call('Azimuth','rights',[number]);return {jsonrpc:'2.0',id,result:{ownership:{owner:{address:rights.owner},managementProxy:{address:rights.managementProxy},spawnProxy:{address:rights.spawnProxy},votingProxy:{address:rights.votingProxy},transferProxy:{address:rights.transferProxy}},network:{rift:p.continuityNumber.toString(),keys:{life:p.keyRevisionNumber.toString(),crypt:p.encryptionKey,auth:p.authenticationKey,suite:p.cryptoSuiteVersion.toString()},sponsor:{has:p.hasSponsor,who:Number(p.sponsor)}}}};}
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
assert.equal(await rpc('eth_blockNumber'),block);const finalBlock=await rpc('eth_getBlockByNumber',[block,false]);await provider.disconnect();provider=null;
fs.writeFileSync('/lab/public-responses.json',JSON.stringify({responses,batch,dns},null,2)+'\n');fs.writeFileSync('/lab/result.json',JSON.stringify({passed:true,hexCanonicalization:true,scope:'fresh key and chain-backed dawn response validation via pinned Vere evaluator; no HTTP server or ships',binaryHash,foundationGenesis:foundation.before.genesis,foundationBlock:foundation.before.latest,block,blockHash:finalBlock.hash,cases,publicKeys:keys.map(({keyfile,...v})=>v),receipts,galaxies:256,keyedGalaxies:2,dnsAccepted:true,secretDirectoryExcludedFromReports:'secrets'},null,2)+'\n');console.log('PASS: fresh local-chain keys accepted; wrong key/life refused; galaxy/DNS responses accepted');}
run().catch(async e=>{fs.writeFileSync('/lab/failure.txt',e.stack+'\n');console.error(e.message);if(provider)await provider.disconnect();process.exitCode=1;});
