'use strict';

// Pure retained-state admission. This module opens no file, process or provider.
const crypto = require('node:crypto');
const MODE = 'private-chain-docker-gateway-resume-v1';
const VERE = 'b5c54439ae5b9f0ff94d75319e67e04f04369776fa5e2180a5ab1a5ebd4c2ff0';
const NODE = '6dcdb9bbe2bf22bd053ba06d5ba7b8a2d92b2d3d8c714f3b2670be3d2728c9c8';
const CONTRACTS = ['Azimuth','Censures','Claims','ConditionalStarRelease','DelegatedSending','Ecliptic','LinearStarRelease','Naive','Polls'];
const READ_METHODS = Object.freeze(['web3_clientVersion','net_version','eth_chainId','eth_blockNumber','eth_getLogs','eth_getBlockByNumber','eth_getBlockByHash','eth_getTransactionReceipt','eth_getTransactionByHash','eth_call','eth_getCode']);
function need(value, code = 'retained_gateway_input_invalid') { if (!value) throw new Error(code); }
function hex(value, digits, prefix = '') { return typeof value === 'string' && new RegExp('^'+prefix+'[0-9a-f]{'+digits+'}$').test(value); }
function hash(data) { return crypto.createHash('sha256').update(data).digest('hex'); }
function networkInterfaces(mode, names) {
  need(mode===undefined||mode==='published-v1','private_gateway_network_mode_invalid');
  need(Array.isArray(names)&&names.every(n=>typeof n==='string')&&[...names].sort().join(',')===(mode==='published-v1'?'eth0,lo':'lo'),'private_gateway_interfaces_invalid');
}
function selection(value) {
  need(value && typeof value === 'object' && !Array.isArray(value));
  const fields=['version','mode','project','operation','previous_gateway','initial_result_sha256','foundation_sha256','compiled_sha256','guest_namespaces'];
  need(Object.keys(value).length===fields.length && fields.every(k=>Object.hasOwn(value,k)));
  need(value.version===1 && value.mode===MODE && hex(value.project,32) && hex(value.operation,32) && hex(value.previous_gateway,64));
  for(const k of ['initial_result_sha256','foundation_sha256','compiled_sha256']) need(hex(value[k],64));
  need(value.guest_namespaces && Object.keys(value.guest_namespaces).sort().join(',')==='mnt,net,pid');
  for(const n of ['net','pid','mnt']) need(new RegExp('^'+n+':\\[[0-9]{1,20}\\]$').test(value.guest_namespaces[n]));
  return structuredClone(value);
}
function inputs(chosen, bytes) {
  selection(chosen);
  need(bytes && Buffer.isBuffer(bytes.proof) && bytes.proof.length<=4*1024*1024 && Buffer.isBuffer(bytes.foundation) && bytes.foundation.length<=4*1024*1024 && Buffer.isBuffer(bytes.compiled) && bytes.compiled.length<=16*1024*1024);
  need(hash(bytes.proof)===chosen.initial_result_sha256 && hash(bytes.foundation)===chosen.foundation_sha256 && hash(bytes.compiled)===chosen.compiled_sha256);
  let proof,foundation,compiled;
  try{proof=JSON.parse(bytes.proof);foundation=JSON.parse(bytes.foundation);compiled=JSON.parse(bytes.compiled);}catch{need(false);}
  need(proof && proof.passed===true && proof.binaryHash===VERE && proof.gateway && proof.gateway.passed===true && proof.gateway.signalShutdown===true && proof.gateway.externalRpcMutationsAllowed===false);
  need(proof.gateway.head==='0x50' && hex(proof.gateway.headHash,64,'0x') && foundation && foundation.passed===true && foundation.before && foundation.after);
  need(JSON.stringify(foundation.before)===JSON.stringify(foundation.after));
  need(foundation.before.chainId==='0x539' && foundation.before.block==='0xd' && hex(foundation.before.genesis,64,'0x') && hex(foundation.before.latest,64,'0x'));
  need(proof.foundationGenesis===foundation.before.genesis && proof.foundationBlock===foundation.before.latest);
  need(foundation.contracts && Object.keys(foundation.contracts).sort().join(',')===[...CONTRACTS].sort().join(','));
  for(const name of CONTRACTS){const c=foundation.contracts[name];need(c && hex(c.address,40,'0x') && hex(c.codeKeccak256,64,'0x'));}
  need(compiled && compiled.contracts && compiled.contracts['azimuth/contracts/Azimuth.sol']?.Azimuth?.abi);
  need(Array.isArray(proof.publicKeys) && proof.publicKeys.length===2 && Array.isArray(proof.cases) && proof.cases.length===2);
  const keys=proof.publicKeys.map((key,n)=>{
    const ship=n===0?'~zod':'~nec', verdict=proof.cases[n];
    need(key && key.number===n && key.ship===ship && hex(key.auth,64,'0x') && hex(key.crypt,64,'0x'));
    need(key.auth!=='0x'+'0'.repeat(64) && key.crypt!=='0x'+'0'.repeat(64));
    need(verdict && verdict.ship===ship && verdict.correct===true && verdict.wrongPublicKeyRefused===true && verdict.wrongLifeRefused===true);
    return {number:n,ship,auth:key.auth,crypt:key.crypt};
  });
  need(keys[0].auth!==keys[1].auth && keys[0].crypt!==keys[1].crypt);
  need(Array.isArray(proof.gateway.initialKeySetup?.publicKeys) && proof.gateway.initialKeySetup.publicKeys.length===2);
  for(let n=0;n<2;n++)for(const field of ['number','ship','auth','crypt'])need(proof.gateway.initialKeySetup.publicKeys[n][field]===keys[n][field]);
  return {expected:{project:chosen.project,operation:chosen.operation,previous_gateway:chosen.previous_gateway,genesis:foundation.before.genesis,head:'0x50',head_hash:proof.gateway.headHash,publicKeys:keys},foundation,compiled};
}
async function observe(expected, foundation, rpc, getPoint, codeHash) {
  const mismatch='retained_gateway_chain_mismatch';
  need(await rpc('eth_chainId')==='0x539',mismatch);
  const genesis=await rpc('eth_getBlockByNumber',['0x0',false]);need(genesis?.hash===expected.genesis,mismatch);
  need(await rpc('eth_blockNumber')===expected.head,mismatch);
  const head=await rpc('eth_getBlockByNumber',[expected.head,false]);need(head && hex(head.hash,64,'0x') && head.hash===expected.head_hash && head.number===expected.head,mismatch);
  for(const name of CONTRACTS){const c=foundation.contracts[name];need(codeHash(await rpc('eth_getCode',[c.address,expected.head]))===c.codeKeccak256,mismatch);}
  for(const key of expected.publicKeys){const p=await getPoint(key.number,'resume-check',expected.head);need(p?.result?.network?.keys?.auth===key.auth && p.result.network.keys.crypt===key.crypt && p.result.network.keys.life==='1' && p.result.network.keys.suite==='1',mismatch);}
  need(await rpc('eth_blockNumber')===expected.head,mismatch);
  return {chain_id:'0x539',genesis:expected.genesis,head:expected.head,head_hash:head.hash,public_keys:structuredClone(expected.publicKeys)};
}
function sameObservation(before,after){need(JSON.stringify(before)===JSON.stringify(after),'retained_gateway_changed_during_read_only_service');}
module.exports={MODE,VERE,NODE,CONTRACTS,READ_METHODS,need,hash,networkInterfaces,selection,inputs,observe,sameObservation};
