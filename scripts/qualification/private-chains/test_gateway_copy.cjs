'use strict';
const assert=require('node:assert/strict'),fs=require('node:fs'),os=require('node:os'),path=require('node:path'),test=require('node:test'),vm=require('node:vm');
const source=fs.readFileSync(path.join(__dirname,'docker_gateway.cjs'),'utf8');
const begin='// BEGIN fresh chain copy: exercised with disposable filesystem fixtures only.\n',end='// END fresh chain copy.';
assert.equal(source.split(begin).length,2);assert.equal(source.split(end).length,2);
const definitions=source.split(begin)[1].split(end)[0];
function copier(filesystem=fs){return vm.runInNewContext(definitions+'\ncopyFreshGatewayChain',Object.freeze({fs:filesystem,assert,require:(name)=>{assert.equal(name,'node:path');return path;}}),{timeout:1000});}
function fixture(t){
 const base=fs.mkdtempSync(path.join(os.tmpdir(),'omabit-gateway-copy-'));fs.chmodSync(base,0o700);
 t.after(()=>{function writable(p){const s=fs.lstatSync(p);if(!s.isDirectory())return;fs.chmodSync(p,0o700);for(const n of fs.readdirSync(p))writable(path.join(p,n));}writable(base);fs.rmSync(base,{recursive:true});});
 const input=path.join(base,'sealed'),output=path.join(base,'chain');fs.mkdirSync(input,{mode:0o700});
 fs.writeFileSync(path.join(input,'CURRENT'),'MANIFEST-000001\n',{mode:0o400});fs.writeFileSync(path.join(input,'MANIFEST-000001'),'synthetic database bytes\n',{mode:0o400});
 fs.chmodSync(input,0o500);return {base,input,output};
}
test('sealed foundation becomes a fresh writable private copy without modifying inputs',t=>{
 const {input,output}=fixture(t);const before=fs.readFileSync(path.join(input,'MANIFEST-000001'));
 copier()(input,output);
 assert.equal(fs.statSync(input).mode&0o777,0o500);assert.equal(fs.statSync(output).mode&0o777,0o700);
 for(const n of fs.readdirSync(input)){assert.equal(fs.statSync(path.join(input,n)).mode&0o777,0o400);assert.equal(fs.statSync(path.join(output,n)).mode&0o777,0o600);assert.deepEqual(fs.readFileSync(path.join(input,n)),fs.readFileSync(path.join(output,n)));}
 const fd=fs.openSync(path.join(output,'MANIFEST-000001'),'r+');fs.writeSync(fd,Buffer.from('changed copy'));fs.closeSync(fd);
 assert.deepEqual(fs.readFileSync(path.join(input,'MANIFEST-000001')),before);
});
test('copy refuses existing destination and never repairs or truncates it',t=>{
 const {input,output}=fixture(t);fs.mkdirSync(output,{mode:0o700});fs.writeFileSync(path.join(output,'keep'),'original',{mode:0o600});
 assert.throws(()=>copier()(input,output));assert.equal(fs.readFileSync(path.join(output,'keep'),'utf8'),'original');assert.deepEqual(fs.readdirSync(output),['keep']);
});
test('copy refuses linked inputs and retains partial output for review',async t=>{
 for(const kind of ['symbolic-file','hardlink','symbolic-directory'])await t.test(kind,t=>{
  const {base,input,output}=fixture(t);fs.chmodSync(input,0o700);
  if(kind==='symbolic-file')fs.symlinkSync('CURRENT',path.join(input,'ZZ-link'));
  else if(kind==='hardlink')fs.linkSync(path.join(input,'CURRENT'),path.join(input,'ZZ-link'));
  else fs.symlinkSync(input,path.join(input,'ZZ-link'));
  fs.chmodSync(input,0o500);
  assert.throws(()=>copier()(input,output));assert(fs.existsSync(output));
  const after=fs.readdirSync(output);assert.throws(()=>copier()(input,output));assert.deepEqual(fs.readdirSync(output),after);
  assert(!fs.existsSync(path.join(base,'outside')));
 });
});
test('copy refuses aliased roots and source/destination nesting',async t=>{
 for(const kind of ['root-link','same','nested','relative'])await t.test(kind,t=>{
  const {base,input,output}=fixture(t);let a=input,b=output;
  if(kind==='root-link'){a=path.join(base,'alias');fs.symlinkSync(input,a);}
  if(kind==='same')b=input;
  if(kind==='nested')b=path.join(input,'nested');
  if(kind==='relative')a='relative';
  assert.throws(()=>copier()(a,b));assert(!fs.existsSync(output));assert.equal(fs.statSync(input).mode&0o777,0o500);
 });
});
test('failed sync retains the fresh copy and prevents automatic retry',t=>{
 const {input,output}=fixture(t);const calls=[];
 const failing=Object.create(fs);failing.fsyncSync=(fd)=>{calls.push(fd);throw Error('synthetic sync failure');};
 assert.throws(()=>copier(failing)(input,output),/synthetic sync failure/);assert.equal(calls.length,1);
 assert.equal(fs.statSync(output).mode&0o777,0o700);const names=fs.readdirSync(output);assert.equal(names.length,1);
 assert.equal(fs.statSync(path.join(output,names[0])).mode&0o777,0o600);
 assert.throws(()=>copier()(input,output));assert.deepEqual(fs.readdirSync(output),names);
});
test('nested directories are private and synced after their copied files',t=>{
 const {input,output}=fixture(t);fs.chmodSync(input,0o700);fs.mkdirSync(path.join(input,'nested'),{mode:0o700});fs.writeFileSync(path.join(input,'nested','file'),'data',{mode:0o400});fs.chmodSync(path.join(input,'nested'),0o500);fs.chmodSync(input,0o500);
 const synced=[];const recording=Object.create(fs);recording.fsyncSync=(fd)=>{synced.push(fs.readlinkSync('/proc/self/fd/'+fd));return fs.fsyncSync(fd);};
 copier(recording)(input,output);assert.equal(fs.statSync(path.join(output,'nested')).mode&0o777,0o700);
 assert(synced.indexOf(path.join(output,'nested','file'))<synced.indexOf(path.join(output,'nested')));
 assert(synced.indexOf(path.join(output,'nested'))<synced.indexOf(output));assert.equal(synced.at(-1),path.dirname(output));
});
test('initializer uses the fresh copy before opening its provider',()=>{
 const call="copyFreshGatewayChain('/deployment/chain','/lab/chain');";
 assert.equal(source.split(call).length,2);assert(source.indexOf(call)<source.indexOf('provider=ganache.provider'));
 assert(!source.includes("fs.cpSync('/deployment/chain'"));
});
