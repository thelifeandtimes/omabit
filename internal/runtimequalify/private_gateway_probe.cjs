// Embedded read-only probe. Importing this file never connects or reads state.
'use strict';
const assert=require('node:assert/strict'),crypto=require('node:crypto'),http=require('node:http');
const FORMAT='omabit-private-gateway-observation-v1';
const READ_METHODS=Object.freeze(['eth_chainId','eth_blockNumber','eth_getBlockByNumber','eth_getCode','eth_call']);
function fields(value,keys){assert(value&&typeof value==='object'&&!Array.isArray(value));assert.deepEqual(Object.keys(value).sort(),[...keys].sort());}
function hash(value){return typeof value==='string'&&/^0x[0-9a-f]{64}$/.test(value)&&value!=='0x'+'0'.repeat(64);}
function validateInput(input){
 fields(input,['foundation','members']);const f=input.foundation;
 fields(f,['version','profile','chain_id','genesis','contracts']);assert.equal(f.version,1);assert.equal(f.profile,'azimuth-local-1337-v1');assert.equal(f.chain_id,1337);assert(hash(f.genesis));
 assert(Array.isArray(f.contracts)&&f.contracts.length===9);const names=new Set(),addresses=new Set();
 for(const c of f.contracts){fields(c,['name','address','code_sha256']);assert(['Azimuth','Censures','Claims','ConditionalStarRelease','DelegatedSending','Ecliptic','LinearStarRelease','Naive','Polls'].includes(c.name)&&!names.has(c.name));assert(/^0x[0-9a-f]{40}$/.test(c.address)&&!addresses.has(c.address));assert(/^[0-9a-f]{64}$/.test(c.code_sha256));names.add(c.name);addresses.add(c.address);}
 assert(Array.isArray(input.members)&&input.members.length===2);const ships=new Set();
 for(const m of input.members){fields(m,['ship','number']);assert((m.ship==='~zod'&&m.number===0)||(m.ship==='~nec'&&m.number===1));assert(!ships.has(m.ship));ships.add(m.ship);}
 return input;
}
function startup(value){fields(value,['format','ready','head','head_hash','durable','reopened']);assert.equal(value.format,'omabit-private-gateway-startup-v1');assert.equal(value.ready,true);assert.equal(value.head,'0x50');assert(hash(value.head_hash));assert.equal(value.durable,true);assert.equal(value.reopened,true);return value;}
// Fixed endpoint, no proxy, redirects, credentials or state-changing methods.
function request(path,body,deadline,transport=http){
 assert((path==='/'&&body.method==='omabit_getStartupStatus')||(path==='/rpc'&&READ_METHODS.includes(body.method)));
 assert.equal(body.jsonrpc,'2.0');assert.equal(typeof body.id,'string');const remaining=deadline-Date.now();assert(remaining>0&&remaining<=12000);
 const bytes=Buffer.from(JSON.stringify(body));assert(bytes.length<=8192);
 return new Promise((resolve,reject)=>{
  let settled=false,timer,req;const finish=(error,value)=>{if(settled)return;settled=true;clearTimeout(timer);if(error){if(req)req.destroy();reject(new Error('private gateway read failed'));}else resolve(value);};
  req=transport.request({host:'127.0.0.1',port:25432,path,method:'POST',agent:false,headers:{'Content-Type':'application/json','Content-Length':bytes.length}},res=>{
   if(res.statusCode!==200||!/^application\/json(?:\s*;|$)/i.test(res.headers['content-type']||'')){res.destroy();finish(true);return;}
   let size=0;const chunks=[];
   res.on('data',chunk=>{size+=chunk.length;if(size>1024*1024){res.destroy();finish(true);}else chunks.push(chunk);});
   res.on('aborted',()=>finish(true));res.on('error',()=>finish(true));
   res.on('end',()=>{if(settled)return;try{const value=JSON.parse(Buffer.concat(chunks).toString('utf8'));fields(value,['jsonrpc','id','result']);assert.equal(value.jsonrpc,'2.0');assert.equal(value.id,body.id);finish(null,{result:value.result,bytes:size});}catch{finish(true);}});
  });
  req.on('error',()=>finish(true));timer=setTimeout(()=>finish(true),Math.min(remaining,2000));req.end(bytes);
 });
}
async function observe(input,read,abi,namespace){
 validateInput(input);const beforeNS=namespace();fields(beforeNS,['device','inode']);for(const n of Object.values(beforeNS))assert(/^[1-9][0-9]*$/.test(n));
 const status=startup(await read('/','omabit_getStartupStatus',{}));
 assert.equal(await read('/rpc','eth_chainId',[]),'0x539');
 const genesis=(await read('/rpc','eth_getBlockByNumber',['0x0',false])).hash;assert.equal(genesis,input.foundation.genesis);
 const head=await read('/rpc','eth_blockNumber',[]);assert.equal(head,'0x50');
 const block=await read('/rpc','eth_getBlockByNumber',[head,false]);assert.equal(block.number,head);assert.equal(block.hash,status.head_hash);
 // Foundation v1 hashes the canonical 0x-prefixed lowercase code string as UTF-8.
 // Preserve deploy.cjs's original convention; decoding bytes changes every pin.
 const contracts=[];
 for(const c of input.foundation.contracts){const code=await read('/rpc','eth_getCode',[c.address,head]);assert(typeof code==='string'&&/^0x(?:[0-9a-f]{2})+$/.test(code));const sum=crypto.createHash('sha256').update(code,'utf8').digest('hex');assert.equal(sum,c.code_sha256);contracts.push({name:c.name,address:c.address,code_sha256:sum});}
 const azimuth=contracts.find(c=>c.name==='Azimuth'),public_keys=[],seen=new Set();
 for(const m of input.members){const data=abi.encodeFunctionData('getKeys',[m.number]);assert(/^0x[0-9a-f]{72}$/.test(data));const encoded=await read('/rpc','eth_call',[{to:azimuth.address,data},head]);assert(typeof encoded==='string'&&/^0x[0-9a-f]{256}$/.test(encoded));const values=abi.decodeFunctionResult('getKeys',encoded);assert.equal(values.suite,1n);assert.equal(values.revision,1n);for(const key of [values.auth,values.crypt]){assert(hash(key)&&!seen.has(key));seen.add(key);}public_keys.push({ship:m.ship,auth:values.auth,crypt:values.crypt,life:Number(values.revision),suite:Number(values.suite)});}
 assert.equal(await read('/rpc','eth_blockNumber',[]),head);const afterBlock=await read('/rpc','eth_getBlockByNumber',[head,false]);assert.equal(afterBlock.number,head);assert.equal(afterBlock.hash,block.hash);
 const afterStatus=startup(await read('/','omabit_getStartupStatus',{}));assert.deepEqual(afterStatus,status);const afterNS=namespace();assert.deepEqual(afterNS,beforeNS);
 return {format:FORMAT,head:80,head_hash:block.hash,foundation:{version:1,profile:input.foundation.profile,chain_id:1337,genesis,contracts},public_keys,startup:status,namespace:beforeNS};
}
async function main(text=process.argv[1]){
 assert(typeof text==='string'&&Buffer.byteLength(text)<=8192);const input=validateInput(JSON.parse(text));
 const fs=require('node:fs');assert.equal(process.env.OMABIT_PRIVATE_CHAIN,'private-chain-docker-gateway-v1');assert.equal(process.env.OMABIT_PRIVATE_CHAIN_NETWORK,'published-v1');assert(fs.existsSync('/.dockerenv'));assert.equal(fs.readFileSync('/sys/class/dmi/id/sys_vendor','utf8').trim(),'QEMU');assert.deepEqual(fs.readdirSync('/sys/class/net').sort(),['eth0','lo']);
 const compiled=JSON.parse(fs.readFileSync('/compiled/compiler-output.json','utf8')),ethers=require('/tools/node_modules/ethers');
 const abi=new ethers.Interface(compiled.contracts['azimuth/contracts/Azimuth.sol'].Azimuth.abi);
 // This observation reads namespace metadata, never chain files, keyfiles or logs.
 const namespace=()=>{const s=fs.statSync('/proc/self/ns/net',{bigint:true});return {device:String(s.dev),inode:String(s.ino)};};
 const deadline=Date.now()+12000;let sequence=0,total=0;
 const read=async(path,method,params)=>{const reply=await request(path,{jsonrpc:'2.0',id:'omabit-observe-'+(++sequence),method,params},deadline);total+=reply.bytes;assert(total<=4*1024*1024);return reply.result;};
 const value=await observe(input,read,abi,namespace),output=JSON.stringify(value);assert(Buffer.byteLength(output)<=16384);process.stdout.write(output+'\n');
}
module.exports={validateInput,startup,request,observe,main,READ_METHODS};
