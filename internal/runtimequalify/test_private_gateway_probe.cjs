'use strict';
const assert=require('node:assert/strict'),crypto=require('node:crypto'),test=require('node:test'),{EventEmitter}=require('node:events');
const probe=require('./private_gateway_probe.cjs');
const digest=bytes=>crypto.createHash('sha256').update(bytes).digest('hex'),head='0x'+'a'.repeat(64),genesis='0x'+'b'.repeat(64);
const ready=()=>({format:'omabit-private-gateway-startup-v1',ready:true,head:'0x50',head_hash:head,durable:true,reopened:true});
function fixture(change){
 const names=['Azimuth','Censures','Claims','ConditionalStarRelease','DelegatedSending','Ecliptic','LinearStarRelease','Naive','Polls'];
 const input={foundation:{version:1,profile:'azimuth-local-1337-v1',chain_id:1337,genesis,contracts:names.map((name,n)=>({name,address:'0x'+String(n+1).repeat(40),code_sha256:digest('0x'+(n+1).toString(16).padStart(2,'0'))}))},members:[{ship:'~nec',number:1},{ship:'~zod',number:0}]};
 if(change==='decoded-code-hash')input.foundation.contracts.forEach((c,n)=>{c.code_sha256=digest(Buffer.from([n+1]));});
 let statuses=0,heights=0,blocks=0,namespaces=0;const calls=[];
 const encoded=n=>'0x'+(n?'3':'1').repeat(64)+(n?'4':'2').repeat(64)+'0'.repeat(63)+'1'+'0'.repeat(63)+'1';
 const read=async(path,method,params)=>{calls.push({path,method,params});if(method==='omabit_getStartupStatus'){const r=ready();if(change==='unready')r.ready=false;if(change==='unsynced')r.durable=false;if(change==='unreopened')r.reopened=false;if(change==='status-change'&&++statuses===2)r.head_hash='0x'+'c'.repeat(64);return r;}
  if(method==='eth_chainId')return change==='chain'?'0x1':'0x539';
  if(method==='eth_blockNumber')return change==='height'?'0x4f':++heights===2&&change==='later-height'?'0x51':'0x50';
  if(method==='eth_getBlockByNumber'){if(params[0]==='0x0')return {hash:change==='genesis'?'0x'+'c'.repeat(64):genesis};blocks++;assert.equal(params[0],'0x50');return {number:'0x50',hash:change==='head'||(change==='later-hash'&&blocks===2)?'0x'+'c'.repeat(64):head};}
  if(method==='eth_getCode'){assert.equal(params[1],'0x50');const n=input.foundation.contracts.findIndex(c=>c.address===params[0]);if(change==='code'&&n===8)return '0xff';if(change==='malformed-code')return '0x1';return '0x'+(n+1).toString(16).padStart(2,'0');}
  if(method==='eth_call'){assert.equal(params[0].to,input.foundation.contracts[0].address);assert.equal(params[1],'0x50');return change==='malformed-keys'?'0x12':encoded(Number(BigInt('0x'+params[0].data.slice(10))));}
  throw new Error('unexpected request '+method);
 };
 const abi={encodeFunctionData:(method,args)=>{assert.equal(method,'getKeys');return '0x12345678'+args[0].toString(16).padStart(64,'0');},decodeFunctionResult:(method,bytes)=>{assert.equal(method,'getKeys');const v={crypt:bytes.slice(0,66),auth:'0x'+bytes.slice(66,130),suite:1n,revision:1n};if(change==='life')v.revision=2n;if(change==='suite')v.suite=2n;if(change==='zero-key')v.crypt='0x'+'0'.repeat(64);if(change==='same-key')v.auth=v.crypt;if(change==='reused-key')v.auth='0x'+'f'.repeat(64);return v;}};
 const namespace=()=>({device:'4',inode:change==='namespace'&&++namespaces===2?'124':'123'});
 return {input,calls,run:()=>probe.observe(input,read,abi,namespace)};
}
test('probe freshly reads nine code hashes and both key rows in selected member order',async()=>{
 const f=fixture(),o=await f.run();assert.equal(o.format,'omabit-private-gateway-observation-v1');assert.equal(o.head,80);assert.equal(o.head_hash,head);assert.deepEqual(o.foundation,f.input.foundation);assert.deepEqual(o.public_keys.map(k=>k.ship),['~nec','~zod']);assert.equal(o.public_keys[0].crypt,'0x'+'3'.repeat(64));assert.equal(f.calls.length,19);assert.equal(f.calls.filter(c=>c.method==='eth_getCode').length,9);assert.equal(f.calls.filter(c=>c.method==='eth_call').length,2);assert.equal(f.calls[0].method,'omabit_getStartupStatus');assert.equal(f.calls.at(-1).method,'omabit_getStartupStatus');assert(f.calls.every(c=>c.method==='omabit_getStartupStatus'||probe.READ_METHODS.includes(c.method)));
});
test('probe rejects incomplete readiness, changed state and malformed or reused keys',async t=>{
 for(const kind of ['unready','unsynced','unreopened','status-change','chain','genesis','height','later-height','head','later-hash','code','malformed-code','malformed-keys','life','suite','zero-key','same-key','reused-key','namespace'])await t.test(kind,async()=>{await assert.rejects(fixture(kind).run());});
});
test('probe validates public input before any request',async t=>{
 for(const kind of ['extra','chain','foundation-extra','missing-contract','duplicate-name','duplicate-address','bad-hash','member','duplicate-member','member-extra'])await t.test(kind,async()=>{const f=fixture();switch(kind){case'extra':f.input.private='must not read';break;case'chain':f.input.foundation.chain_id=1;break;case'foundation-extra':f.input.foundation.private=true;break;case'missing-contract':f.input.foundation.contracts.pop();break;case'duplicate-name':f.input.foundation.contracts[1].name='Azimuth';break;case'duplicate-address':f.input.foundation.contracts[1].address=f.input.foundation.contracts[0].address;break;case'bad-hash':f.input.foundation.contracts[1].code_sha256='bad';break;case'member':f.input.members[0].number=0;break;case'duplicate-member':f.input.members[1]=f.input.members[0];break;case'member-extra':f.input.members[0].key='forbidden';break;}await assert.rejects(f.run());assert.equal(f.calls.length,0);});
});
function transport(mode,body){
 const events=[],res=new EventEmitter();res.statusCode=mode==='redirect'?302:200;res.headers={'content-type':mode==='content-type'?'text/html':'application/json'};res.destroy=()=>events.push('destroy-response');
 const req=new EventEmitter();req.destroy=()=>events.push('destroy-request');req.end=bytes=>{events.push(['body',JSON.parse(bytes)]);queueMicrotask(()=>{if(mode==='request-error'){req.emit('error',new Error('private untrusted error'));return;}if(mode==='timeout')return;callback(res);if(mode==='aborted'){res.emit('aborted');return;}if(mode==='response-error'){res.emit('error',new Error('private untrusted response'));return;}let value={jsonrpc:'2.0',id:body.id,result:{ok:true}};if(mode==='wrong-id')value.id='other';if(mode==='rpc-error')value={jsonrpc:'2.0',id:body.id,error:{message:'do not export'}};if(mode==='extra')value.extra='do not export';const data=Buffer.from(mode==='invalid-json'?'{':mode==='oversize'?'x'.repeat(1024*1024+1):JSON.stringify(value));res.emit('data',data);res.emit('end');});};let callback;
 const api={request:(options,fn)=>{callback=fn;events.push(['options',options]);return req;}};return {api,events};
}
test('HTTP adapter uses only fixed loopback POST and verifies its matching response',async()=>{
 const body={jsonrpc:'2.0',id:'read-1',method:'eth_chainId',params:[]},t=transport('ok',body);const reply=await probe.request('/rpc',body,Date.now()+12000,t.api);assert.deepEqual(reply.result,{ok:true});assert(reply.bytes>0);const options=t.events[0][1];assert.equal(options.host,'127.0.0.1');assert.equal(options.port,25432);assert.equal(options.method,'POST');assert.equal(options.agent,false);assert.equal(options.path,'/rpc');assert.equal(t.events.length,2);
});
test('HTTP adapter refuses redirects, malformed responses, size limits and deadlines',async t=>{
 for(const mode of ['redirect','content-type','request-error','response-error','aborted','wrong-id','rpc-error','extra','invalid-json','oversize','timeout'])await t.test(mode,async()=>{const body={jsonrpc:'2.0',id:'read-2',method:'eth_chainId',params:[]},x=transport(mode,body);await assert.rejects(probe.request('/rpc',body,Date.now()+(mode==='timeout'?15:12000),x.api),error=>error.message==='private gateway read failed');assert(x.events.includes('destroy-request'));});
});
test('mutation methods and alternate paths fail before transport creation',()=>{
 for(const [path,method]of [['/rpc','evm_mine'],['/rpc','eth_sendTransaction'],['/rpc','omabit_getStartupStatus'],['/','eth_call'],['http://external.invalid/','eth_chainId']])assert.throws(()=>probe.request(path,{jsonrpc:'2.0',id:'r',method,params:[]},Date.now()+12000,{request:()=>assert.fail('transport must remain unused')}));
});

// This hash convention is fixed by the original foundation deployment writer.
test('foundation v1 uses the prefixed UTF-8 code hash and refuses decoded-byte pins',async()=>{
 const f=fixture();assert.equal(f.input.foundation.contracts[0].code_sha256,digest('0x01'));
 assert.notEqual(f.input.foundation.contracts[0].code_sha256,digest(Buffer.from([1])));
 await f.run();await assert.rejects(fixture('decoded-code-hash').run());
});
