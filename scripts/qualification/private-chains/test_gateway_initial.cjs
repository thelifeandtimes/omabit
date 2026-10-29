'use strict';
const assert=require('node:assert/strict'),fs=require('node:fs'),vm=require('node:vm'),test=require('node:test'),path=require('node:path');
// Read only the marked pure definitions. Never evaluate the gateway entrypoint,
// load chain/key libraries, or execute its filesystem/process startup code here.
const source=fs.readFileSync(path.join(__dirname,'docker_gateway.cjs'),'utf8');
const begin='// BEGIN pure initial gateway contract: independently exercised without startup.\n';
const end='// END pure initial gateway contract.';
assert.equal(source.split(begin).length,2);assert.equal(source.split(end).length,2);
const definitions=source.split(begin)[1].split(end)[0];
const contract=vm.runInNewContext(definitions+'\n({interfaces:initialGatewayInterfaces,head:initialGatewayHead})',Object.freeze({}),{timeout:1000});
test('initial network defaults to loopback only and bridge mode is explicit',()=>{
 assert.equal(contract.interfaces(undefined,['lo']),'none');
 assert.equal(contract.interfaces('published-v1',['lo','eth0']),'published-v1');
 assert.equal(contract.interfaces('published-v1',['eth0','lo']),'published-v1');
 for(const [mode,names]of [[undefined,['lo','eth0']],['published-v1',['lo']],['published-v1',['lo','eth0','eth1']],['published-v1',['lo','eth0','eth0']],['published-v1',['lo','wlan0']],['none',['lo']],['',['lo']],[null,['lo']],['other',['eth0','lo']]])assert.throws(()=>contract.interfaces(mode,names));
 for(const names of [null,{},'lo',[1],['lo',null]])assert.throws(()=>contract.interfaces(undefined,names));
});
test('network validation does not rewrite supplied observations',()=>{
 const names=Object.freeze(['lo','eth0']);assert.equal(contract.interfaces('published-v1',names),'published-v1');assert.deepEqual(names,['lo','eth0']);
});
test('readiness requires the exact supported initialized block and nonzero hash',()=>{
 const hash='0x'+'ab'.repeat(32),header=Object.freeze({number:'0x50',hash});
 assert.equal(contract.head('0x50',header),hash);
 for(const [block,value]of [['0x4f',header],['0x050',header],[80,header],['0x50',null],['0x50',[]],['0x50',{number:'0x51',hash}],['0x50',{number:'0x50',hash:'0x'+'0'.repeat(64)}],['0x50',{number:'0x50',hash:'0x'+'A'.repeat(64)}],['0x50',{number:'0x50',hash:'0x12'}],['0x50',{number:'0x50'}]])assert.throws(()=>contract.head(block,value));
});
test('entrypoint calls network guard before fresh keys and publishes the checked head',()=>{
 const call="const initialNetwork=initialGatewayInterfaces(process.env.OMABIT_PRIVATE_CHAIN_NETWORK,fs.readdirSync('/sys/class/net'));";
 assert.equal(source.split(call).length,2);assert(source.indexOf(call)<source.indexOf("fs.mkdirSync('/lab/secrets'"));
 assert(source.includes("const readyHead=initialGatewayHead(readyBlock,await rpc('eth_getBlockByNumber',['latest',false]));"));
 assert(source.includes('block:readyBlock,headHash:readyHead,initialNetwork,'));
 // Preserve the existing container/QEMU and namespace guard for either mode.
 assert(source.includes("assert(fs.existsSync('/.dockerenv'))"));
 assert(source.includes("assert.equal(fs.readFileSync('/sys/class/dmi/id/sys_vendor','utf8').trim(),'QEMU')"));
 assert(source.includes("for(const n of ['net','pid','mnt'])assert.notEqual(fs.readlinkSync('/proc/self/ns/'+n),original[n]);"));
});
