'use strict';
const test=require('node:test'),assert=require('node:assert/strict'),path=require('node:path'),cp=require('node:child_process');
const c=require('./gateway_resume_contract.cjs');
function fixture(){
  const base={chainId:'0x539',block:'0xd',genesis:'0x'+'1'.repeat(64),latest:'0x'+'2'.repeat(64)};
  const foundation={passed:true,before:base,after:structuredClone(base),contracts:Object.fromEntries(c.CONTRACTS.map((name,n)=>[name,{address:'0x'+(n+1).toString(16).repeat(40),codeKeccak256:'0x'+(n+1).toString(16).repeat(64)}]))};
  const publicKeys=[0,1].map(n=>({number:n,ship:n?'~nec':'~zod',crypt:'0x'+(n+3).toString(16).repeat(64),auth:'0x'+(n+5).toString(16).repeat(64)}));
  const proof={passed:true,binaryHash:c.VERE,foundationGenesis:base.genesis,foundationBlock:base.latest,publicKeys,cases:publicKeys.map(k=>({ship:k.ship,correct:true,wrongPublicKeyRefused:true,wrongLifeRefused:true})),gateway:{passed:true,signalShutdown:true,externalRpcMutationsAllowed:false,head:'0x50',headHash:'0x'+'e'.repeat(64),initialKeySetup:{publicKeys:structuredClone(publicKeys)}}};
  const compiled={contracts:{'azimuth/contracts/Azimuth.sol':{Azimuth:{abi:[]}}}};
  const bytes={proof:Buffer.from(JSON.stringify(proof)),foundation:Buffer.from(JSON.stringify(foundation)),compiled:Buffer.from(JSON.stringify(compiled))};
  const chosen={version:1,mode:c.MODE,project:'a'.repeat(32),operation:'b'.repeat(32),previous_gateway:'c'.repeat(64),initial_result_sha256:c.hash(bytes.proof),foundation_sha256:c.hash(bytes.foundation),compiled_sha256:c.hash(bytes.compiled),guest_namespaces:{net:'net:[101]',pid:'pid:[102]',mnt:'mnt:[103]'}};
  return {chosen,bytes,proof,foundation,compiled};
}
function backend(f,change){
  const calls=[];
  const rpc=async(method,params=[])=>{
    calls.push(method);assert(c.READ_METHODS.includes(method),'mutation forwarded');
    if(method==='eth_chainId')return change==='chain'?'0x1':'0x539';
    if(method==='eth_blockNumber')return change==='head'?'0x51':'0x50';
    if(method==='eth_getBlockByNumber')return {number:params[0],hash:params[0]==='0x0'?(change==='genesis'?'0x'+'a'.repeat(64):f.foundation.before.genesis):'0x'+(change==='head-hash'?'d':'e').repeat(64)};
    if(method==='eth_getCode'){const contract=Object.values(f.foundation.contracts).find(c=>c.address===params[0]);return change==='code'?'changed':contract.codeKeccak256;}
    throw Error('unplanned read');
  };
  const getPoint=async(n,id)=>({id,result:{network:{keys:{auth:change==='key'?'0x'+'0'.repeat(64):f.proof.publicKeys[n].auth,crypt:f.proof.publicKeys[n].crypt,life:change==='life'?'2':'1',suite:change==='suite'?'2':'1'}}}});
  return {rpc,getPoint,calls};
}
test('retained inputs bind prior clean proof without keys or initialization',()=>{
  const f=fixture(),original=structuredClone(f.chosen),p=c.inputs(f.chosen,f.bytes);
  assert.equal(p.expected.project,f.chosen.project);assert.equal(p.expected.head,'0x50');
  assert.deepEqual(f.chosen,original);assert(!JSON.stringify(p.expected).includes('keyfile'));
  const copy=c.selection(f.chosen);copy.guest_namespaces.net='net:[9]';assert.notEqual(copy.guest_namespaces.net,f.chosen.guest_namespaces.net);
});
test('retained input selection refuses aliases, missing bindings and changed bytes',async t=>{
  for(const kind of ['version','mode','project','operation','container','namespace','extra','hash','missing-proof','proof-bytes','foundation-bytes','compiler-bytes'])await t.test(kind,()=>{
    const f=fixture();
    switch(kind){case'version':f.chosen.version++;break;case'mode':f.chosen.mode='initialize';break;case'project':f.chosen.project='other';break;case'operation':f.chosen.operation='';break;case'container':f.chosen.previous_gateway='';break;case'namespace':f.chosen.guest_namespaces.net='pid:[101]';break;case'extra':f.chosen.keyfile='DO-NOT-REFLECT';break;case'hash':f.chosen.foundation_sha256='';break;case'missing-proof':delete f.bytes.proof;break;case'proof-bytes':f.bytes.proof=Buffer.from('{}');break;case'foundation-bytes':f.bytes.foundation=Buffer.from('{}');break;case'compiler-bytes':f.bytes.compiled=Buffer.from('{}');break;}
    assert.throws(()=>c.inputs(f.chosen,f.bytes),e=>e.message==='retained_gateway_input_invalid');
  });
});
test('freshly hashed but unsupported predecessor proof still refuses',async t=>{
  for(const kind of ['not-passed','not-stopped','rpc-mutations','head','missing-head-hash','invalid-head-hash','genesis','foundation-state','contracts','binary','same-keys','wrong-verdict','ready-keys'])await t.test(kind,()=>{
    const f=fixture();
    switch(kind){case'not-passed':f.proof.passed=false;break;case'not-stopped':f.proof.gateway.signalShutdown=false;break;case'rpc-mutations':f.proof.gateway.externalRpcMutationsAllowed=true;break;case'head':f.proof.gateway.head='0xd';break;case'missing-head-hash':delete f.proof.gateway.headHash;break;case'invalid-head-hash':f.proof.gateway.headHash='bad';break;case'genesis':f.proof.foundationGenesis='0x'+'a'.repeat(64);break;case'foundation-state':f.foundation.after.block='0xe';break;case'contracts':delete f.foundation.contracts.Azimuth;break;case'binary':f.proof.binaryHash='a'.repeat(64);break;case'same-keys':f.proof.publicKeys[1].auth=f.proof.publicKeys[0].auth;break;case'wrong-verdict':f.proof.cases[0].correct=false;break;case'ready-keys':f.proof.gateway.initialKeySetup.publicKeys[0].auth='0x'+'f'.repeat(64);break;}
    f.bytes.proof=Buffer.from(JSON.stringify(f.proof));f.bytes.foundation=Buffer.from(JSON.stringify(f.foundation));f.chosen.initial_result_sha256=c.hash(f.bytes.proof);f.chosen.foundation_sha256=c.hash(f.bytes.foundation);
    assert.throws(()=>c.inputs(f.chosen,f.bytes),e=>e.message==='retained_gateway_input_invalid');
  });
});
test('fresh retained-chain observation performs only reads and matches exact keys',async t=>{
  for(const change of [undefined,'chain','genesis','head','head-hash','code','key','life','suite'])await t.test(change||'valid',async()=>{
    const f=fixture(),p=c.inputs(f.chosen,f.bytes),b=backend(f,change);
    const run=()=>c.observe(p.expected,p.foundation,b.rpc,b.getPoint,x=>x);
    if(change)await assert.rejects(run,e=>e.message==='retained_gateway_chain_mismatch');
    else{const before=await run(),after=await run();c.sameObservation(before,after);assert.equal(before.head_hash,'0x'+'e'.repeat(64));assert.equal(b.calls.filter(x=>x==='eth_getCode').length,18);after.head_hash='0x'+'d'.repeat(64);assert.throws(()=>c.sameObservation(before,after));}
    assert(b.calls.every(m=>c.READ_METHODS.includes(m)));
  });
});
test('host entry refuses before dependencies or chain access; import is inert',()=>{
  const env={...process.env};delete env.OMABIT_PRIVATE_CHAIN;
  const r=cp.spawnSync(process.execPath,[path.join(__dirname,'docker_gateway_resume.cjs')],{env,encoding:'utf8',timeout:5000});
  assert.ifError(r.error);assert.equal(r.status,1);assert.equal(r.stdout,'');assert.equal(r.stderr,'private_gateway_resume_refused\n');
  assert.equal(typeof require('./docker_gateway_resume.cjs').main,'function');
});

test('network publication requires exact opt-in and interfaces',()=>{
  c.networkInterfaces(undefined,['lo']);c.networkInterfaces('published-v1',['lo','eth0']);
  for(const [mode,names] of [[undefined,['lo','eth0']],['published-v1',['lo']],['published-v1',['lo','eth0','eth1']],['published-v1',['eth0','eth0']],['host',['lo']],['',['lo']],['published-v1',null]])assert.throws(()=>c.networkInterfaces(mode,names));
});
