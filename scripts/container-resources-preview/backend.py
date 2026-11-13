#!/usr/bin/env python3
"""Resource UI simulation only. Fixed preview data path; no host/runtime helpers."""
import copy,fcntl,json,os,sys
from pathlib import Path

def initial():
    ships={}
    for runner,instance,running in [('preview-local-040','preview-publisher',False),('preview-remote-040','preview-remote-bus',True)]:
        ships[runner+'/'+instance]={'revision':1,'applied_revision':1,'running':running,'launch':1,
            'config':{'http_port':18080 if not running else 18082,'ames_port':34343,'loom':31,'memory_gib':4,'cpu_cores':2},
            'launch_config':{'loom':31,'memory_gib':4,'cpu_cores':2}}
    return {'version':1,'ships':ships}

def integer(value,minimum,maximum):
    return type(value) is int and minimum<=value<=maximum

def apply(state,mode,action,request):
    if mode not in ('online','offline','observer') or state.get('version')!=1 or set(state.get('ships',{}))!=set(initial()['ships']):
        raise ValueError('Invalid preview resource state')
    if action=='status':return state
    if action not in ('configure','start','stop'):raise ValueError('Unsupported resource preview action')
    target=request['target'];key=target['runner']+'/'+target['instance']
    if key not in state['ships'] or target['connection']+'-040'!=target['runner']:
        raise ValueError('Unknown simulated ship')
    if target['runner']=='preview-remote-040' and mode!='online':raise ValueError('An online operator fixture is required')
    ship=state['ships'][key]
    if target['revision']!=ship['revision']:raise ValueError('Configuration changed; reload the editor')
    if action=='configure':
        if ship['running']:raise ValueError('Stop the simulated ship before changing its resources')
        config=request['config']
        bounds={'memory_gib':(1,16),'cpu_cores':(1,16),'http_port':(1024,65535),'ames_port':(34343,34343),'loom':(30,34)}
        if set(config)!=set(bounds) or any(not integer(config[k],*v) for k,v in bounds.items()):raise ValueError('Invalid simulated resource configuration')
        if config!=ship['config']:ship['config']=dict(config);ship['revision']+=1
    else:
        if (action=='start')==ship['running']:raise ValueError('Simulated ship state changed; refresh')
        ship['running']=action=='start'
        if ship['running']:
            ship['launch']+=1;ship['applied_revision']=ship['revision'];ship['launch_config']=dict(ship['config'])
    return state

def main():
    mode,action,*args=sys.argv[1:]
    if action not in ('status','configure','start','stop') or len(args)!=1:raise ValueError('Unsupported resource preview request')
    request=json.loads(args[0]);path=Path('/data/state/resources.json')
    with path.with_suffix('.lock').open('a') as lock:
        fcntl.flock(lock,fcntl.LOCK_EX)
        state=json.loads(path.read_text()) if path.exists() else initial()
        result=apply(copy.deepcopy(state),mode,action,request)
        if not path.exists() or action!='status':
            temporary=path.with_suffix('.tmp')
            with temporary.open('w') as f:json.dump(result,f,indent=2);f.write('\n');f.flush();os.fsync(f.fileno())
            temporary.replace(path)
        print(json.dumps(result))

if __name__=='__main__':
    try:main()
    except (ValueError,KeyError,TypeError) as error:print(str(error),file=sys.stderr);sys.exit(1)
