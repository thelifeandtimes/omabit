// Synthetic sample-history microbenchmark; no desktop or runner access.
const fs = require('node:fs');
const vm = require('node:vm');
const assert = require('node:assert/strict');
const files = process.argv.slice(2);
assert.equal(files.length,2,'provide baseline and candidate Model.js');
const now = Date.now(), stamp = new Date(now).toISOString(), container = 'a'.repeat(64);
const ship = {id:'fake',observed_state:'running',resources:{source:'docker-stats-v1.47',status:'fresh',observed_at:stamp,container_id:container,memory_current_bytes:100,memory_limit_bytes:200}};
const connection = {name:'fixture',runner_id:'runner',status:'available',checked_at:stamp,snapshot:{instances:[]}};
const cases = [
  {name:'empty',count:0,loops:20000,stopped:false},
  {name:'stopped',count:128,loops:2000,stopped:true},
  {name:'unchanged-history',count:128,loops:2000,stopped:false},
];
const results=[];
for (const file of files) {
  const model=vm.createContext({Date,Math,Number,JSON,isFinite});
  vm.runInContext(fs.readFileSync(file,'utf8'),model);
  const rows=[];
  for (const spec of cases) {
    const input={...connection,snapshot:{instances:Array.from({length:spec.count},(_,n)=>({...ship,id:'fake-'+n,observed_state:spec.stopped?'stopped':'running'}))}};
    const history={};
    if(!spec.stopped)for(let n=0;n<spec.count;n++)history['runner/fake-'+n]=Array.from({length:120},(_,i)=>({time:now-(119-i)*2000,value:100,limit:200,container}));
    const before=JSON.stringify(history), started=process.cpuUsage();let replacements=0, result;
    for(let i=0;i<spec.loops;i++){result=model.recordSamples(history,input,now);if(result!==history)replacements++}
    const cpu=process.cpuUsage(started);
    assert.equal(JSON.stringify(history),before,'source map was mutated');
    assert.equal(JSON.stringify(result),before,'unchanged samples changed their data');
    rows.push({case:spec.name,loops:spec.loops,map_replacements:replacements,cpu_ms:(cpu.user+cpu.system)/1000});
  }
  results.push({file,cases:rows});
}
console.log(JSON.stringify({scope:'Node sample-model microbenchmark; not a desktop CPU measurement',results},null,2));
