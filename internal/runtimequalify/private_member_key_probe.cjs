// Importing this module neither reads a key nor connects to a runtime.
'use strict';
const assert=require('node:assert/strict'),crypto=require('node:crypto');
const FORMAT='omabit-private-member-key-proof-v1';
const fail=()=>new Error('private member key proof refused');
const hash=v=>typeof v==='string'&&/^[0-9a-f]{64}$/.test(v)&&v!=='0'.repeat(64);
function fields(v,names){assert(v&&typeof v==='object'&&!Array.isArray(v));assert.deepEqual(Object.keys(v).sort(),[...names].sort());}
function validate(input,probe){
 fields(input,['gateway','ship','binding_sha256']);probe.validateInput(input.gateway);
 assert(hash(input.binding_sha256));assert(input.ship==='~zod'||input.ship==='~nec');
 const member=input.gateway.members.find(m=>m.ship===input.ship);assert(member);return member;
}
function groupHex(value){assert(/^[0-9a-f]+$/.test(value));value=value.replace(/^0+/,'')||'0';const n=value.length%4||4;return value.slice(0,n)+(value.length>n?'.'+value.slice(n).match(/.{4}/g).join('.'):'');}
function expression(ship,key,point){
 assert(ship==='~zod'||ship==='~nec');assert(typeof key==='string'&&key.length>=18&&key.length<=2048&&/^0w[0-9a-zA-Z.~-]+$/.test(key));
 const bytes=Buffer.from(JSON.stringify(point));assert(bytes.length>0&&bytes.length<=16384);
 const length=String(bytes.length).replace(/\B(?=(\d{3})+(?!\d))/g,'.');
 const octs='['+length+' 0x'+groupHex(Buffer.from(bytes).reverse().toString('hex'))+']';
 return '-:(veri:dawn '+ship+' ;;(feed:jael (cue '+key+')) (need (point:take:dawn '+ship+' '+octs+')) ~)\n';
}
// Read the point and rights from the selected fixed historical head. The final
// full observer independently verifies the foundation/head and the same keys.
async function readPoint(member,foundation,read,abi){
 assert((member.ship==='~zod'&&member.number===0)||(member.ship==='~nec'&&member.number===1));
 const address=foundation.contracts.find(c=>c.name==='Azimuth').address;
 async function call(method){
  const data=abi.encodeFunctionData(method,[member.number]);assert(/^0x[0-9a-f]{72}$/.test(data));
  const result=await read('/rpc','eth_call',[{to:address,data},'0x50']);assert(typeof result==='string'&&/^0x(?:[0-9a-f]{64})+$/.test(result)&&result.length<=4098);
  return abi.decodeFunctionResult(method,result);
 }
 const p=await call('points'),r=await call('rights');
 for(const n of [p.continuityNumber,p.keyRevisionNumber,p.cryptoSuiteVersion])assert(typeof n==='bigint'&&n>=0n&&n<=0xffffffffn);
 assert.equal(p.keyRevisionNumber,1n);assert.equal(p.cryptoSuiteVersion,1n);
 assert(typeof p.sponsor==='bigint'&&p.sponsor>=0n&&p.sponsor<=0xffffffffn);assert.equal(typeof p.hasSponsor,'boolean');
 for(const value of [p.authenticationKey,p.encryptionKey])assert(typeof value==='string'&&/^0x[0-9a-f]{64}$/.test(value)&&value!=='0x'+'0'.repeat(64));
 for(const value of [r.owner,r.managementProxy,r.spawnProxy,r.votingProxy,r.transferProxy])assert(typeof value==='string'&&/^0x[0-9a-fA-F]{40}$/.test(value));
 return {jsonrpc:'2.0',id:'omabit-member-key',result:{ownership:{owner:{address:r.owner},managementProxy:{address:r.managementProxy},spawnProxy:{address:r.spawnProxy},votingProxy:{address:r.votingProxy},transferProxy:{address:r.transferProxy}},network:{rift:p.continuityNumber.toString(),keys:{life:p.keyRevisionNumber.toString(),crypt:p.encryptionKey,auth:p.authenticationKey,suite:p.cryptoSuiteVersion.toString()},sponsor:{has:p.hasSponsor,who:Number(p.sponsor)}}}};
}
function runEvaluator(input,spawn){
 const result=spawn('/vere',['eval','--loom','29'],{input,encoding:'utf8',timeout:30000,maxBuffer:1024*1024,cwd:'/tmp',env:{PATH:'/usr/bin:/bin',HOME:'/tmp'}});
 if(result.error||result.status!==0||result.signal||typeof result.stdout!=='string'||Buffer.byteLength(result.stdout)>1024*1024)throw fail();
 if(result.stdout.replace(/\x1b\[[0-9;]*m/g,'').trim()!=='%.y')throw fail();
}
// Fixed root is selected by production main; tests use an inert temporary root.
// No key/point/evaluator output is returned. The accepted whole-file hash remains
// private journal material; it is not wallet authorization or secure erasure.
function openKey(root,ship,spawn,fs=require('node:fs')){
 assert(ship==='~zod'||ship==='~nec');assert(root.startsWith('/')&&!root.endsWith('/'));
 const paths=[root,root+'/secrets',root+'/secrets/'+ship.slice(1)+'.key'],fds=[],uid=BigInt(process.geteuid());
 let closed=false,initial,accepted;
 const snapshot=(st,directory)=>{
  assert.equal(st.uid,uid);assert(st.ino>0n&&st.birthtimeNs>0n);assert.equal(st.mode,BigInt(directory?0o40700:0o100600));
  if(!directory){assert.equal(st.nlink,1n);assert(st.size>=18n&&st.size<=2049n);}
  return directory?{dev:st.dev,ino:st.ino,birth:st.birthtimeNs,mode:st.mode,uid:st.uid}:{dev:st.dev,ino:st.ino,birth:st.birthtimeNs,mode:st.mode,uid:st.uid,gid:st.gid,nlink:st.nlink,size:st.size,mtime:st.mtimeNs,ctime:st.ctimeNs};
 };
 const metadata=()=>{
  assert(!closed&&fds.length===3);
  for(let i=0;i<3;i++){const held=snapshot(fs.fstatSync(fds[i],{bigint:true}),i<2),named=snapshot(fs.lstatSync(paths[i],{bigint:true}),i<2);assert.deepEqual(held,initial[i]);assert.deepEqual(named,held);}
 };
 const read=()=>{
  metadata();const raw=Buffer.alloc(2050);let used=0;
  try{while(used<raw.length){const n=fs.readSync(fds[2],raw,used,raw.length-used,used);if(n===0)break;used+=n;}
   assert.equal(BigInt(used),initial[2].size);const bytes=raw.subarray(0,used),key=bytes.toString('utf8').replace(/\n$/,'');
   assert(key.length>=18&&key.length<=2048&&/^0w[0-9a-zA-Z.~-]+$/.test(key));const sha=crypto.createHash('sha256').update(bytes).digest('hex');metadata();return {key,sha};
  }finally{raw.fill(0);}
 };
 const close=()=>{if(closed)return;closed=true;let failed=false;for(const fd of fds.reverse()){try{fs.closeSync(fd);}catch{failed=true;}}if(failed)throw fail();};
 try{
  for(let i=0;i<paths.length;i++)fds.push(fs.openSync(paths[i],fs.constants.O_RDONLY|fs.constants.O_NOFOLLOW|fs.constants.O_NONBLOCK|(i<2?fs.constants.O_DIRECTORY:0)));
  initial=fds.map((fd,i)=>snapshot(fs.fstatSync(fd,{bigint:true}),i<2));assert(initial.every(s=>s.dev===initial[0].dev));assert.equal(new Set(initial.map(s=>String(s.ino))).size,3);accepted=read().sha;
  const verify=()=>{try{assert.equal(read().sha,accepted);}catch{throw fail();}};
  return {sha256:accepted,verify,evaluate:point=>{try{const value=read();assert.equal(value.sha,accepted);runEvaluator(expression(ship,value.key,point),spawn);verify();}catch{throw fail();}},close};
 }catch{close();throw fail();}
}
async function prove(input,probe,dependencies){
 let key;
 try{
  const member=validate(input,probe);key=dependencies.openKey(member.ship);assert(hash(key.sha256));key.verify();
  const point=await dependencies.readPoint(member,input.gateway.foundation);key.evaluate(point);
  // The evaluator may take longer than a live freshness window. Gather the full
  // chain observation after it finishes, then recheck the same held key bytes.
  const gateway=await dependencies.observe(input.gateway);assert.equal(gateway.head,80);
  const publicKey=gateway.public_keys.find(k=>k.ship===member.ship),pointKey=point.result.network.keys;
  assert(publicKey);assert.equal(publicKey.auth,pointKey.auth);assert.equal(publicKey.crypt,pointKey.crypt);assert.equal(publicKey.life,1);assert.equal(publicKey.suite,1);assert.equal(pointKey.life,'1');assert.equal(pointKey.suite,'1');
  key.verify();return {format:FORMAT,binding_sha256:input.binding_sha256,ship:member.ship,key_sha256:key.sha256,verified:true,gateway};
 }catch{throw fail();}finally{if(key){try{key.close();}catch{throw fail();}}}
}
async function main(probe,text=process.argv[1]){
 assert(typeof text==='string'&&Buffer.byteLength(text)<=12288);const input=JSON.parse(text);validate(input,probe);
 const fs=require('node:fs'),cp=require('node:child_process');
 assert.equal(process.env.OMABIT_PRIVATE_CHAIN,'private-chain-docker-gateway-v1');assert.equal(process.env.OMABIT_PRIVATE_CHAIN_NETWORK,'published-v1');assert(fs.existsSync('/.dockerenv'));assert.equal(fs.readFileSync('/sys/class/dmi/id/sys_vendor','utf8').trim(),'QEMU');assert.deepEqual(fs.readdirSync('/sys/class/net').sort(),['eth0','lo']);
 const compiled=JSON.parse(fs.readFileSync('/compiled/compiler-output.json','utf8')),ethers=require('/tools/node_modules/ethers');
 const abi=new ethers.Interface(compiled.contracts['azimuth/contracts/Azimuth.sol'].Azimuth.abi);
 const namespace=()=>{const s=fs.statSync('/proc/self/ns/net',{bigint:true});return {device:String(s.dev),inode:String(s.ino)};};
 let sequence=0;const reader=()=>{const deadline=Date.now()+12000;let total=0;return async(path,method,params)=>{const r=await probe.request(path,{jsonrpc:'2.0',id:'omabit-key-'+(++sequence),method,params},deadline);total+=r.bytes;assert(total<=4*1024*1024);return r.result;};};
 const value=await prove(input,probe,{openKey:ship=>openKey('/lab',ship,cp.spawnSync),readPoint:(m,f)=>readPoint(m,f,reader(),abi),observe:g=>probe.observe(g,reader(),abi,namespace)});
 const output=JSON.stringify(value);assert(Buffer.byteLength(output)<=24576);process.stdout.write(output+'\n');
}
module.exports={validate,groupHex,expression,readPoint,runEvaluator,openKey,prove,main};
