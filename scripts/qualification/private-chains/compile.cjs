'use strict';
const fs=require('node:fs'),path=require('node:path'),crypto=require('node:crypto');
const assert=require('node:assert/strict');
assert.equal(process.env.OMABIT_PRIVATE_CHAIN,'private-chain-compile-v1');
assert.equal(process.env.HOME,'/lab/home');
assert.equal(fs.readFileSync('/lab/marker','utf8'),'private-chain-compile-v1\n');
assert.equal(JSON.parse(fs.readFileSync('/lab/scope-admission.json','utf8')).admitted,true);assert(!fs.existsSync('/home'));assert(!fs.existsSync('/run/user'));
const original=JSON.parse(fs.readFileSync('/lab/host-namespaces.json','utf8'));
for(const n of ['user','net','pid','mnt'])assert.notEqual(fs.readlinkSync('/proc/self/ns/'+n),original[n]);
const sources={};
function collect(root,prefix){for(const e of fs.readdirSync(root,{withFileTypes:true})){const p=path.join(root,e.name),k=prefix+'/'+e.name;if(e.isDirectory())collect(p,k);else if(e.isFile()&&p.endsWith('.sol'))sources[k]={content:fs.readFileSync(p,'utf8')};else assert(!e.isSymbolicLink());}}
collect('/source/azimuth/contracts','azimuth/contracts');collect('/source/bridge/contracts','bridge/contracts');collect('/tools/node_modules/openzeppelin-solidity/contracts','openzeppelin-solidity/contracts');
const solc=require('/tools/node_modules/solc');const ethers=require('/tools/node_modules/ethers');
assert.equal(solc.version(),'0.4.24+commit.e67f0147.Emscripten.clang');
const input={language:'Solidity',sources,settings:{optimizer:{enabled:true,runs:200},evmVersion:'byzantium',outputSelection:{'*':{'*':['abi','evm.bytecode.object','evm.deployedBytecode.object']}}}};
fs.writeFileSync('/lab/compiler-input.json',JSON.stringify(input));
const out=JSON.parse(solc.compileStandardWrapper(JSON.stringify(input)));
fs.writeFileSync('/lab/compiler-output.json',JSON.stringify(out));
assert.deepEqual((out.errors||[]).filter(e=>e.severity==='error'),[]);
const script=JSON.parse(fs.readFileSync('/source/bridge/package.json','utf8')).scripts['pilot:ganache'];
const match=script.match(/ -m '([a-z ]+)'/);assert(match);
const deployer=ethers.HDNodeWallet.fromPhrase(match[1]).address.toLowerCase();
assert.equal(deployer,'0x6deffb0cafdb11d175f123f6891aa64f01c24f7d');
const addresses=Array.from({length:16},(_,nonce)=>({nonce,address:ethers.getCreateAddress({from:deployer,nonce}).toLowerCase()}));
const expected={Azimuth:'0x863d9c2e5c4c133596cfac29d55255f0d0f86381',Ecliptic:'0x56db68f29203ff44a803faa2404a44ecbb7a7480',DelegatedSending:'0xb71c0b6cee1bcae56dfe95cd9d3e41ddd7eafc43',ConditionalStarRelease:'0x35eb3b102d9c1b69ac1469c1b1fe1799850cd3eb',LinearStarRelease:'0x03c3dc12be658158d1d7f9e66e08ec4099c568e4',Naive:'0x6bb88a9bbd82be7a997feb01929c6ec78988fe12'};
const nonces={};for(const [name,address] of Object.entries(expected)){const a=addresses.find(v=>v.address===address);assert(a,name+' unmatched');nonces[name]=a.nonce;}
const digest=b=>crypto.createHash('sha256').update(b).digest('hex');
const result={passed:true,scope:'offline compiler and deterministic CREATE addresses only; no chain or ships started',compiler:solc.version(),node:process.version,sources:Object.fromEntries(Object.entries(sources).map(([k,v])=>[k,digest(v.content)])),compilerInputSha256:digest(fs.readFileSync('/lab/compiler-input.json')),compilerOutputSha256:digest(fs.readFileSync('/lab/compiler-output.json')),warnings:(out.errors||[]).filter(e=>e.severity!=='error').map(e=>e.formattedMessage),deployer,addresses,expected,nonces,contracts:Object.values(out.contracts).reduce((n,v)=>n+Object.keys(v).length,0)};
fs.writeFileSync('/lab/result.json',JSON.stringify(result,null,2)+'\n');console.log(JSON.stringify({passed:true,contracts:result.contracts,nonces}));
