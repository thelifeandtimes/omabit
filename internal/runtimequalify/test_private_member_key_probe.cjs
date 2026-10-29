'use strict';
const assert=require('node:assert/strict'),test=require('node:test'),fs=require('node:fs'),os=require('node:os'),path=require('node:path'),crypto=require('node:crypto');
const keyProbe=require('./private_member_key_probe.cjs'),gateway=require('./private_gateway_probe.cjs');
const digest=x=>crypto.createHash('sha256').update(x).digest('hex'),synthetic='0w123456789abcdef123456789\n';
function input(){return {gateway:{foundation:{version:1,profile:'azimuth-local-1337-v1',chain_id:1337,genesis:'0x'+'a'.repeat(64),contracts:['Azimuth','Censures','Claims','ConditionalStarRelease','DelegatedSending','Ecliptic','LinearStarRelease','Naive','Polls'].map((name,n)=>({name,address:'0x'+String(n+1).repeat(40),code_sha256:digest('0x01')}))},members:[{ship:'~nec',number:1},{ship:'~zod',number:0}]},ship:'~zod',binding_sha256:'b'.repeat(64)};}
function point(){return {jsonrpc:'2.0',id:'omabit-member-key',result:{network:{keys:{auth:'0x'+'c'.repeat(64),crypt:'0x'+'d'.repeat(64),life:'1',suite:'1'}}}};}
function fixture(t){const root=fs.mkdtempSync(path.join(os.tmpdir(),'omabit-synthetic-key-proof-'));fs.mkdirSync(root+'/secrets',{mode:0o700});fs.writeFileSync(root+'/secrets/zod.key',synthetic,{mode:0o600});t.after(()=>fs.rmSync(root,{recursive:true,force:true}));return root;}
function fakeSpawn(events,mode='ok'){return(file,args,options)=>{
 events.push('evaluate');assert.equal(file,'/vere');assert.deepEqual(args,['eval','--loom','29']);assert.equal(options.cwd,'/tmp');assert.deepEqual(options.env,{PATH:'/usr/bin:/bin',HOME:'/tmp'});assert.equal(options.timeout,30000);assert.equal(options.maxBuffer,1024*1024);assert.equal(options.encoding,'utf8');assert(options.input.startsWith('-:(veri:dawn ~zod '));assert(options.input.endsWith('\n'));
 if(mode==='throw')throw new Error('secret raw failure');
 return {status:mode==='status'?1:0,signal:mode==='signal'?'SIGTERM':null,error:mode==='timeout'?new Error('raw secret timeout'):null,stdout:mode==='negative'?'%.n':mode==='extra'?'%.y\nprivate output':mode==='oversize'?'x'.repeat(1024*1024+1):'\x1b[32m%.y\x1b[0m\n',stderr:'private stderr never exported'};
};}
test('held synthetic key stays private and uses stdin-only bounded evaluation',t=>{
 const root=fixture(t),events=[],v=keyProbe.openKey(root,'~zod',fakeSpawn(events));assert.equal(v.sha256,digest(synthetic));assert.deepEqual(Object.keys(v).sort(),['close','evaluate','sha256','verify']);
 v.verify();v.evaluate(point());v.verify();assert.deepEqual(events,['evaluate']);assert.equal(fs.readFileSync(root+'/secrets/zod.key','utf8'),synthetic);v.close();v.close();assert.throws(()=>v.verify(),/private member key proof refused/);
});
test('key metadata and bytes refuse unsafe inputs without repair',async t=>{
 for(const kind of ['empty','oversize','invalid','newline','public','hardlink','symlink','directory','public-parent','linked-parent'])await t.test(kind,t=>{
  const root=fixture(t),file=root+'/secrets/zod.key';
  switch(kind){case'empty':fs.writeFileSync(file,'');break;case'oversize':fs.writeFileSync(file,'0w'+'a'.repeat(2048));break;case'invalid':fs.writeFileSync(file,'0w123456789ab[malicious]');break;case'newline':fs.writeFileSync(file,synthetic+'\n');break;case'public':fs.chmodSync(file,0o644);break;case'hardlink':fs.linkSync(file,root+'/retained');break;case'symlink':fs.renameSync(file,root+'/retained');fs.symlinkSync(root+'/retained',file);break;case'directory':fs.unlinkSync(file);fs.mkdirSync(file);break;case'public-parent':fs.chmodSync(root+'/secrets',0o755);break;case'linked-parent':fs.renameSync(root+'/secrets',root+'/retained');fs.symlinkSync(root+'/retained',root+'/secrets');break;}
  assert.throws(()=>keyProbe.openKey(root,'~zod',()=>assert.fail('must not evaluate')),e=>e.message==='private member key proof refused');assert(fs.lstatSync(file));
 });
});
test('held key detects replacement and modification before further evaluation',async t=>{
 for(const kind of ['root','secrets','file','contents','mode'])await t.test(kind,t=>{
  const root=fixture(t),events=[],v=keyProbe.openKey(root,'~zod',fakeSpawn(events));t.after(()=>v.close());
  const file=root+'/secrets/zod.key';
  if(kind==='root'){const kept=root+'-retained';fs.renameSync(root,kept);t.after(()=>fs.rmSync(kept,{recursive:true,force:true}));fs.mkdirSync(root,{mode:0o700});fs.renameSync(kept+'/secrets',root+'/secrets');}
  if(kind==='secrets'){fs.renameSync(root+'/secrets',root+'/retained');fs.mkdirSync(root+'/secrets',{mode:0o700});fs.renameSync(root+'/retained/zod.key',file);}
  if(kind==='file'){fs.renameSync(file,root+'/retained');fs.writeFileSync(file,synthetic,{mode:0o600});}
  if(kind==='contents'){const before=fs.statSync(file);fs.writeFileSync(file,synthetic.replace('abc','def'));fs.utimesSync(file,before.atime,before.mtime);}
  if(kind==='mode')fs.chmodSync(file,0o644);
  assert.throws(()=>v.evaluate(point()),e=>e.message==='private member key proof refused');assert.deepEqual(events,[]);assert(fs.lstatSync(file));
 });
});
test('evaluator rejects negative, ambiguous, truncated and failed results',async t=>{
 for(const mode of ['negative','extra','status','signal','timeout','oversize','throw'])await t.test(mode,t=>{
  const root=fixture(t),v=keyProbe.openKey(root,'~zod',fakeSpawn([],mode));t.after(()=>v.close());assert.throws(()=>v.evaluate(point()),e=>e.message==='private member key proof refused');
 });
});
test('key mutation during evaluator execution invalidates the proof',t=>{
 const root=fixture(t),v=keyProbe.openKey(root,'~zod',(file,args,options)=>{const result=fakeSpawn([])(file,args,options);fs.writeFileSync(root+'/secrets/zod.key',synthetic.replace('abc','def'));return result;});t.after(()=>v.close());assert.throws(()=>v.evaluate(point()),e=>e.message==='private member key proof refused');
});
function proofFixture(change){
 const events=[],value=input(),p=point();let verifies=0;
 const deps={openKey:ship=>{events.push('open');assert.equal(ship,'~zod');return {sha256:change==='hash'?'0'.repeat(64):digest(synthetic),verify:()=>{events.push('verify');if(change==='changed-key'&&++verifies===2)throw new Error('private failure');},evaluate:actual=>{events.push('evaluate');assert.deepEqual(actual,p);if(change==='negative')throw new Error('private failed key');},close:()=>{events.push('close');if(change==='close')throw new Error('private close failure');}};},readPoint:async member=>{events.push('point');assert.deepEqual(member,{ship:'~zod',number:0});return p;},observe:async actual=>{events.push('observe');assert.deepEqual(actual,value.gateway);if(change==='rpc')throw new Error('private RPC diagnostic');return {head:change==='head'?81:80,public_keys:[{ship:'~zod',auth:change==='public-key'?'0x'+'e'.repeat(64):p.result.network.keys.auth,crypt:p.result.network.keys.crypt,life:change==='life'?2:1,suite:1}]};}};
 return {value,events,run:()=>keyProbe.prove(value,gateway,deps)};
}
test('chain observation follows evaluation and returns only a bound public proof',async()=>{
 const f=proofFixture(),out=await f.run();assert.deepEqual(f.events,['open','verify','point','evaluate','observe','verify','close']);assert.equal(out.format,'omabit-private-member-key-proof-v1');assert.equal(out.binding_sha256,f.value.binding_sha256);assert.equal(out.key_sha256,digest(synthetic));assert.equal(out.verified,true);assert(!JSON.stringify(out).includes(synthetic.trim()));
});
test('proof refuses mismatched chain, key and lifecycle result without raw errors',async t=>{
 for(const mode of ['hash','negative','rpc','head','public-key','life','changed-key','close'])await t.test(mode,async()=>{const f=proofFixture(mode);await assert.rejects(f.run(),e=>e.message==='private member key proof refused');assert.equal(f.events.at(-1),'close');});
});
test('malformed input refuses before key or runtime access',async t=>{
 for(const mode of ['extra','binding','ship','members','foundation'])await t.test(mode,async()=>{const f=proofFixture();switch(mode){case'extra':f.value.key='forbidden';break;case'binding':f.value.binding_sha256='0'.repeat(64);break;case'ship':f.value.ship='../../other';break;case'members':f.value.gateway.members.pop();break;case'foundation':f.value.gateway.foundation.chain_id=1;break;}await assert.rejects(f.run());assert.deepEqual(f.events,[]);});
});
test('point reads use only exact Azimuth methods and fixed head',async()=>{
 const events=[],i=input(),m=i.gateway.members[1],address='0x'+'1'.repeat(40),p={continuityNumber:0n,keyRevisionNumber:1n,cryptoSuiteVersion:1n,sponsor:0n,hasSponsor:false,authenticationKey:'0x'+'c'.repeat(64),encryptionKey:'0x'+'d'.repeat(64)},rights={owner:address,managementProxy:address,spawnProxy:address,votingProxy:address,transferProxy:address};
 const abi={encodeFunctionData:(name,args)=>{events.push(name);assert.deepEqual(args,[0]);return '0x12345678'+'0'.repeat(64);},decodeFunctionResult:name=>name==='points'?p:rights};
 const read=async(path,method,args)=>{assert.equal(path,'/rpc');assert.equal(method,'eth_call');assert.equal(args[0].to,address);assert.equal(args[1],'0x50');return '0x'+'0'.repeat(64);};
 const out=await keyProbe.readPoint(m,i.gateway.foundation,read,abi);assert.deepEqual(events,['points','rights']);assert.equal(out.result.network.keys.auth,p.authenticationKey);assert.equal(out.result.network.keys.life,'1');
 p.keyRevisionNumber=2n;await assert.rejects(keyProbe.readPoint(m,i.gateway.foundation,read,abi));
});
test('Hoon expression preserves canonical hex and rejects injected key text',()=>{
 assert.equal(keyProbe.groupHex('0001'),'1');assert.equal(keyProbe.groupHex('000012345'),'1.2345');assert.equal(keyProbe.groupHex('0000'),'0');
 const value=keyProbe.expression('~zod',synthetic.trim(),point());assert(value.includes(';;(feed:jael (cue '+synthetic.trim()+'))'));assert(value.includes('(point:take:dawn ~zod ['));assert.throws(()=>keyProbe.expression('~bud',synthetic.trim(),point()));assert.throws(()=>keyProbe.expression('~zod',synthetic.trim()+')',point()));
});
test('production entry refuses this host before reading key or runtime inputs',async()=>{assert.notEqual(process.env.OMABIT_PRIVATE_CHAIN,'private-chain-docker-gateway-v1');await assert.rejects(keyProbe.main(gateway,JSON.stringify(input())));});
