"""Inert retained gateway ownership/mount checks; never invokes Docker."""
import copy,os,sys,unittest
from pathlib import Path
from unittest.mock import patch
sys.path.insert(0,str(Path(__file__).resolve().parent))
import docker_resume_case as case
import docker_runtime as runtime

PROJECT='a'*32;CONTAINER='b'*64;NAME='resume-gateway'
MOUNTS=case.retained_mounts(Path('/synthetic/work'),Path('/synthetic/project/chain'),Path('/synthetic/project/result.json'),Path('/synthetic/selection.json'))
VARS={'HOME':'/resume/home','OMABIT_PRIVATE_CHAIN':case.MODE,'LIBC':'glibc','PREBUILDS_ONLY':'1'}
COMMAND=['--library-path','/node/usr/lib','/node/usr/bin/node','/gateway/docker_gateway_resume.cjs']

def row():
 return {'Id':CONTAINER,'Image':runtime.IMAGE_ID,'Name':'/'+NAME,
  'Config':{'Image':runtime.IMAGE,'Labels':{runtime.LABEL:PROJECT},'User':'0:0','WorkingDir':'/resume','Entrypoint':[case.LOADER],'Cmd':COMMAND.copy(),'Env':[k+'='+v for k,v in VARS.items()]},
  'HostConfig':{'NetworkMode':'none','Init':True,'ReadonlyRootfs':True,'RestartPolicy':{'Name':'no'},'PortBindings':{},'Privileged':False,'Memory':1536*1024**2,'MemorySwap':1536*1024**2,'NanoCpus':2000000000,'PidsLimit':128,'CapAdd':None,'CapDrop':['ALL'],'CgroupnsMode':'private','IpcMode':'private','SecurityOpt':['no-new-privileges=true'],'Tmpfs':{'/tmp':'rw,nosuid,nodev,mode=1777,size=64m','/urbit':'ro,nosuid,nodev,noexec,mode=0555,size=64k'}},
  'Mounts':[{'Type':'bind','Source':source,'Destination':target,'RW':rw} for source,(target,rw) in MOUNTS.items()],
  'State':{'Running':False,'ExitCode':0,'OOMKilled':False,'Status':'exited'}}

class RetainedGatewayTests(unittest.TestCase):
 def test_host_refuses_before_docker_or_filesystem_selection(self):
  with patch.dict(os.environ,{},clear=True),patch.object(case,'Engine') as engine,patch.object(case,'directory') as directory:
   with self.assertRaisesRegex(RuntimeError,'guest admission'):case.exercise({}, {'ok':True,'project':PROJECT})
   engine.assert_not_called();directory.assert_not_called()

 def test_retained_mounts_exclude_keys_piers_and_original_project_root(self):
  self.assertEqual({src for src,(_,rw) in MOUNTS.items() if rw},{'/synthetic/work','/synthetic/project/chain'})
  self.assertNotIn('/synthetic/project',MOUNTS)
  self.assertFalse(any('secrets' in src or 'piers' in src for src in MOUNTS))
  self.assertEqual(MOUNTS['/synthetic/project/result.json'],('/initial-result.json',False))

 def test_exact_container_contract_is_required(self):
  self.assertEqual(case.check_gateway(row(),CONTAINER,PROJECT,NAME,MOUNTS,COMMAND,VARS)['ExitCode'],0)
  changes={'id':lambda r:r.update(Id='d'*64),'image':lambda r:r.update(Image='bad'),'name':lambda r:r.update(Name='/other'),
   'project':lambda r:r['Config']['Labels'].update({runtime.LABEL:'d'*32}),'entry':lambda r:r['Config'].update(Entrypoint=['/bin/sh']),
   'command':lambda r:r['Config'].update(Cmd=['initializer']),'network':lambda r:r['HostConfig'].update(NetworkMode='host'),
   'published':lambda r:r['HostConfig'].update(PortBindings={'25432/tcp':[{}]}),'privileged':lambda r:r['HostConfig'].update(Privileged=True),
   'root-write':lambda r:r['HostConfig'].update(ReadonlyRootfs=False),'memory':lambda r:r['HostConfig'].update(Memory=0),
   'swap':lambda r:r['HostConfig'].update(MemorySwap=-1),'capability':lambda r:r['HostConfig'].update(CapAdd=['SYS_ADMIN']),
   'namespace':lambda r:r['HostConfig'].update(IpcMode='host'),'environment':lambda r:r['Config']['Env'].append('OMABIT_PRIVATE_CHAIN=initialize'),
   'cwd':lambda r:r['Config'].update(WorkingDir='/lab'),'missing-mount':lambda r:r['Mounts'].pop(),
   'duplicate-mount':lambda r:r['Mounts'].append(r['Mounts'][0]),'changed-mount':lambda r:r['Mounts'][0].update(Source='/other'),
   'volume':lambda r:r['Mounts'].append({'Type':'volume','Destination':'/urbit'}),'writable-volume':lambda r:r['Mounts'].append({'Type':'tmpfs','Destination':'/urbit','RW':True}),
   'oom':lambda r:r['State'].update(OOMKilled=True),'paused':lambda r:r['State'].update(Paused=True),'runtime-error':lambda r:r['State'].update(Error='failed')}
  for name,change in changes.items():
   with self.subTest(name=name):
    r=row();change(r)
    with self.assertRaises(RuntimeError):case.check_gateway(r,CONTAINER,PROJECT,NAME,MOUNTS,COMMAND,VARS)

if __name__=='__main__':unittest.main()
