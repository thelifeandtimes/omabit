'use strict';
const fs=require('node:fs'),crypto=require('node:crypto'),cp=require('node:child_process'),assert=require('node:assert/strict');
assert.equal(process.env.OMABIT_PRIVATE_CHAIN,'private-chain-boot-requests-v1');assert.equal(process.env.HOME,'/lab/home');
assert.equal(fs.readFileSync('/lab/marker','utf8'),'private-chain-boot-requests-v1\n');assert.equal(JSON.parse(fs.readFileSync('/lab/scope-admission.json','utf8')).admitted,true);assert(!fs.existsSync('/home'));assert(!fs.existsSync('/run/user'));
const original=JSON.parse(fs.readFileSync('/lab/host-namespaces.json','utf8'));for(const n of ['user','net','pid','mnt'])assert.notEqual(fs.readlinkSync('/proc/self/ns/'+n),original[n]);
const hash=crypto.createHash('sha256').update(fs.readFileSync('/vere')).digest('hex');assert.equal(hash,'b5c54439ae5b9f0ff94d75319e67e04f04369776fa5e2180a5ab1a5ebd4c2ff0');
const requests={};
for(const [name,expression] of Object.entries({point:'(scot %ux q:(point:give:dawn ~zod))',galaxies:'(scot %ux q:czar:give:dawn)',dns:'(scot %ux q:turf:give:dawn)'})){
 const result=cp.spawnSync('/vere',['eval','--loom','29'],{input:expression+'\n',encoding:'utf8',timeout:30000,maxBuffer:1024*1024,cwd:'/lab',env:{PATH:'/usr/bin:/bin',HOME:'/lab/home'}});
 fs.writeFileSync('/lab/'+name+'.stdout',result.stdout||'');fs.writeFileSync('/lab/'+name+'.stderr',result.stderr||'');assert(!result.error);assert.equal(result.status,0);
 const match=result.stdout.match(/0x[0-9a-f.]+/);assert(match,name+' output');const hex=match[0].slice(2).replaceAll('.','');const even=hex.length%2?'0'+hex:hex;const text=Buffer.from(even,'hex').reverse().toString('utf8');requests[name]=JSON.parse(text);
}
assert.equal(requests.point.method,'getPoint');assert.deepEqual(requests.point.params,{ship:'~zod'});assert.equal(requests.dns.method,'getDns');assert(Array.isArray(requests.galaxies));assert.equal(requests.galaxies.length,256);for(const r of requests.galaxies)assert.equal(r.method,'getPoint');
fs.writeFileSync('/lab/requests.json',JSON.stringify(requests,null,2)+'\n');fs.writeFileSync('/lab/result.json',JSON.stringify({passed:true,scope:'pinned binary ivory request generation via eval; no HTTP calls, keys or ship boot',binarySha256:hash,node:process.version,pointMethod:requests.point.method,dnsMethod:requests.dns.method,galaxyBatch:requests.galaxies.length},null,2)+'\n');console.log('PASS: pinned Vere ivory generates getPoint/getDns and 256-galaxy batch');
