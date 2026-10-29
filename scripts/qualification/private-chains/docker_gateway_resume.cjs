'use strict';

// Guest-only candidate for reopening a stopped qualified gateway. No initial
// chain copy, fresh key generation, transaction submission or mining exists here.
async function main(){
  if(process.env.OMABIT_PRIVATE_CHAIN!=='private-chain-docker-gateway-resume-v1'||process.env.HOME!=='/resume/home')throw new Error('private_gateway_resume_guest_required');
  const fs=require('node:fs'),cp=require('node:child_process'),http=require('node:http');
  const c=require('./gateway_resume_contract.cjs');
  const read=(path,limit)=>{const fd=fs.openSync(path,fs.constants.O_RDONLY|fs.constants.O_NOFOLLOW|fs.constants.O_NONBLOCK);try{const s=fs.fstatSync(fd);c.need(s.isFile()&&s.nlink===1&&s.size<=limit);const bytes=fs.readFileSync(fd);c.need(bytes.length===s.size);return bytes;}finally{fs.closeSync(fd);}};
  c.need(fs.existsSync('/.dockerenv')&&fs.readFileSync('/sys/class/dmi/id/sys_vendor','utf8').trim()==='QEMU');
  const networkMode=process.env.OMABIT_PRIVATE_CHAIN_NETWORK;
  c.networkInterfaces(networkMode,fs.readdirSync('/sys/class/net'));
  // Publication is enabled only by the admitted no-NIC guest controller, which
  // verifies the exact owned bridge and IPv4 loopback bindings before peers run.
  // The chain RPC listener below remains on127.0.0.1 and is never published.
  const chosen=c.selection(JSON.parse(read('/selection.json',16*1024)));
  for(const n of ['net','pid','mnt'])c.need(fs.readlinkSync('/proc/self/ns/'+n)!==chosen.guest_namespaces[n]);
  c.need(process.env.LIBC==='glibc'&&process.env.PREBUILDS_ONLY==='1');
  c.need(c.hash(read('/node/usr/bin/node',256*1024*1024))===c.NODE);
  c.need(c.hash(read('/vere',256*1024*1024))===c.VERE);
  const {expected,foundation,compiled}=c.inputs(chosen,{proof:read('/initial-result.json',4*1024*1024),foundation:read('/deployment/result.json',4*1024*1024),compiled:read('/compiled/compiler-output.json',16*1024*1024)});
  const chain=fs.lstatSync('/chain');c.need(chain.isDirectory()&&!chain.isSymbolicLink());
  const files=fs.readdirSync('/chain');c.need(files.includes('CURRENT')&&files.some(n=>n.startsWith('MANIFEST-'))&&files.length<=64);
  for(const name of files){c.need(/^[A-Za-z0-9._-]+$/.test(name));const st=fs.lstatSync('/chain/'+name);c.need(st.isFile()&&!st.isSymbolicLink()&&st.nlink===1);}
  const current=read('/chain/CURRENT',256).toString();c.need(/^MANIFEST-[0-9]+\n$/.test(current)&&files.includes(current.trim()));
  c.need(fs.statSync('/chain/'+current.trim()).size>0&&files.some(n=>/^[0-9]+\.ldb$/.test(n)&&fs.statSync('/chain/'+n).size>0));
  // Dependency loading occurs only after the guest/input/retained-directory gates.
  const loader=require('/tools/node_modules/ganache/node_modules/node-gyp-build');
  const addon=loader.path('/tools/node_modules/ganache/node_modules/leveldown');
  c.need(addon==='/tools/node_modules/ganache/node_modules/leveldown/prebuilds/linux-x64/node.napi.glibc.node');
  c.need(c.hash(read(addon,16*1024*1024))==='35164284059b66d0de99a33674d99ff2fafeb27b928e9014adb6a06838d7c05a');
  const ganache=require('/tools/node_modules/ganache'),ethers=require('/tools/node_modules/ethers');
  const azimuth=new ethers.Interface(compiled.contracts['azimuth/contracts/Azimuth.sol'].Azimuth.abi);
  let provider,server,before;
  const calls={};
  const rpc=async(method,params=[])=>{c.need(c.READ_METHODS.includes(method),'private_gateway_read_method_required');calls[method]=(calls[method]||0)+1;return provider.request({method,params});};
  const call=async(method,args=[],block='latest')=>azimuth.decodeFunctionResult(method,await rpc('eth_call',[{to:foundation.contracts.Azimuth.address,data:azimuth.encodeFunctionData(method,args)},block]));
  const getPoint=async(number,id,block='latest')=>{const p=await call('points',[number],block),rights=await call('rights',[number],block);return {jsonrpc:'2.0',id,result:{ownership:{owner:{address:rights.owner},managementProxy:{address:rights.managementProxy},spawnProxy:{address:rights.spawnProxy},votingProxy:{address:rights.votingProxy},transferProxy:{address:rights.transferProxy}},network:{rift:p.continuityNumber.toString(),keys:{life:p.keyRevisionNumber.toString(),crypt:p.encryptionKey,auth:p.authenticationKey,suite:p.cryptoSuiteVersion.toString()},sponsor:{has:p.hasSponsor,who:Number(p.sponsor)}}}};};
  const write=(name,value)=>{const path='/resume/'+name,tmp=path+'.tmp';const fd=fs.openSync(tmp,fs.constants.O_CREAT|fs.constants.O_EXCL|fs.constants.O_WRONLY,0o600);try{fs.writeFileSync(fd,JSON.stringify(value,null,2)+'\n');fs.fsyncSync(fd);}finally{fs.closeSync(fd);}fs.renameSync(tmp,path);const dir=fs.openSync('/resume',fs.constants.O_RDONLY|fs.constants.O_DIRECTORY);try{fs.fsyncSync(dir);}finally{fs.closeSync(dir);}};
  const stopped=new Promise(resolve=>process.once('SIGTERM',resolve));
  try{
    provider=ganache.provider({chain:foundation.chainSettings,wallet:{totalAccounts:0},database:{dbPath:'/chain'},miner:foundation.miner,logging:{quiet:true}});
    before=await c.observe(expected,foundation,rpc,getPoint,ethers.keccak256);
    // Public galaxy names only; no key/private-file evaluation or first boot.
    const evaluated=cp.spawnSync('/vere',['eval','--loom','29'],{input:'(scot %ux q:czar:give:dawn)\n',encoding:'utf8',timeout:30000,maxBuffer:1024*1024,cwd:'/resume',env:{PATH:'/usr/bin:/bin',HOME:'/resume/home'}});
    c.need(!evaluated.error&&evaluated.status===0,'private_gateway_public_names_failed');
    const match=evaluated.stdout.replace(/\x1b\[[0-9;]*m/g,'').match(/0x[0-9a-f.]+/);c.need(match,'private_gateway_public_names_failed');
    const hex=match[0].slice(2).replaceAll('.','');const names=JSON.parse(Buffer.from(hex.length%2?'0'+hex:hex,'hex').reverse().toString('utf8'));c.need(names.length===256);
    const ships=new Map();names.forEach((v,n)=>{c.need(v.id==='gal-'+n&&typeof v.params?.ship==='string'&&!ships.has(v.params.ship));ships.set(v.params.ship,n);});
    const reply=async(route,v)=>{
      c.need(v&&v.jsonrpc==='2.0'&&['string','number'].includes(typeof v.id),'private_gateway_request_invalid');
      if(route==='/rpc'){c.need(c.READ_METHODS.includes(v.method),'private_gateway_read_method_required');return {jsonrpc:'2.0',id:v.id,result:await rpc(v.method,v.params||[])};}
      if(v.method==='getPoint'&&v.params&&Object.keys(v.params).length===1&&ships.has(v.params.ship))return getPoint(ships.get(v.params.ship),v.id);
      if(v.method==='getDns'&&(!v.params||Object.keys(v.params).length===0)){const domains=[];for(let n=0;n<3;n++)domains.push((await call('dnsDomains',[n]))[0]);return {jsonrpc:'2.0',id:v.id,result:domains};}
      return {jsonrpc:'2.0',id:v.id,error:{code:-32601,message:'Unsupported private gateway method'}};
    };
    server=http.createServer(async(req,res)=>{try{c.need(req.method==='POST'&&['/','/rpc'].includes(req.url));let size=0,chunks=[];for await(const chunk of req){size+=chunk.length;c.need(size<=1024*1024);chunks.push(chunk);}const v=JSON.parse(Buffer.concat(chunks));c.need(!Array.isArray(v)||v.length>0&&v.length<=256);const answer=Array.isArray(v)?await Promise.all(v.map(x=>reply(req.url,x))):await reply(req.url,v);res.writeHead(200,{'Content-Type':'application/json'});res.end(JSON.stringify(answer));}catch{res.writeHead(400,{'Content-Type':'application/json'});res.end('{"error":"private_gateway_request_refused"}');}});
    server.requestTimeout=10000;server.headersTimeout=10000;
    await new Promise((resolve,reject)=>{server.once('error',reject);server.listen(25432,'127.0.0.1',resolve);});
    write('resume-ready.json',{version:1,mode:c.MODE,project:expected.project,operation:expected.operation,previous_gateway:expected.previous_gateway,observation:before,read_only_rpc:true,initialization_performed:false});
    process.stdout.write('PRIVATE_GATEWAY_RESUME_READY\n');
    await stopped;await new Promise(resolve=>server.close(resolve));server=null;
    const after=await c.observe(expected,foundation,rpc,getPoint,ethers.keccak256);c.sameObservation(before,after);
    await provider.disconnect();provider=null;
    write('resume-result.json',{version:1,mode:c.MODE,passed:true,project:expected.project,operation:expected.operation,previous_gateway:expected.previous_gateway,before,after,read_only_rpc:true,initialization_performed:false,signal_shutdown:true,method_counts:calls});
    process.stdout.write('PRIVATE_GATEWAY_RESUME_STOPPED\n');
  }finally{if(server)await new Promise(resolve=>server.close(resolve));if(provider)await provider.disconnect();}
}
if(require.main===module)main().catch(()=>{process.stderr.write('private_gateway_resume_refused\n');process.exitCode=1;});
module.exports={main};
