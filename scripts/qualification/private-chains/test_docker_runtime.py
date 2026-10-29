"""Inert keyed-container boundary checks. No Docker or ship is started."""
import copy
import importlib.util
import os
from pathlib import Path
import sys
import unittest
from unittest.mock import patch

HERE=Path(__file__).resolve().parent
spec=importlib.util.spec_from_file_location('private_docker_runtime',HERE/'docker_runtime.py')
runtime=importlib.util.module_from_spec(spec);sys.modules[spec.name]=runtime;spec.loader.exec_module(runtime)
PROJECT='a'*32;CONTAINER='b'*64;ANCHOR='c'*64


def ship(resumed=False):return runtime.Ship(PROJECT,'nec',ANCHOR,25432,resumed)
def inspected(s, running=True):
 return {'Id':CONTAINER,'Image':runtime.IMAGE_ID,'Name':'/'+s.name,
  'Config':{'Image':runtime.IMAGE,'Labels':{runtime.LABEL:PROJECT},'User':'0:0','Entrypoint':['/bin/urbit'],'Cmd':s.arguments()},
  'HostConfig':{'NetworkMode':'container:'+ANCHOR,'Init':True,'ReadonlyRootfs':True,'RestartPolicy':{'Name':'no'},
   'PortBindings':{},'Privileged':False,'Memory':4*1024**3,'MemorySwap':4*1024**3,'NanoCpus':2000000000,
   'PidsLimit':256,'CapAdd':None,'CapDrop':['ALL'],'CgroupnsMode':'private','IpcMode':'private','SecurityOpt':['no-new-privileges=true']},
  'Mounts':[{'Type':'bind','Source':source,'Destination':target,'RW':rw} for source,(target,rw) in s.mounts().items()],
  'State':{'Running':running,'ExitCode':0,'OOMKilled':False,'Status':'running' if running else 'exited'}}


class FakeEngine(runtime.Engine):
 def __init__(self,s):self.spec=s;self.row=inspected(s);self.commands=[];self.timeline=[];self.wait_exit=b'0\n';self.wait_running=False;self.wait_oom=False
 def inspect(self,container):runtime.identity(container);return copy.deepcopy(self.row)
 def execute(self,args,timeout=30):
  self.commands.append(args);self.timeline.append(('command',args[0]))
  if args[0]=='wait':self.row['State'].update(Running=self.wait_running,OOMKilled=self.wait_oom,ExitCode=int(self.wait_exit));return self.wait_exit
  return (CONTAINER+'\n').encode()


class KeyedContainerTests(unittest.TestCase):
 def test_fresh_keyed_boot_uses_local_gateway_and_named_nested_pier(self):
  s=ship();args=s.arguments();cmd=s.create_command()
  self.assertEqual(args[-1],'/urbit/nec');self.assertEqual(str(s.pier),'/home/lab/omabit-private-chain-'+PROJECT+'/piers/private-nec/nec')
  for flag in ('-w','-k','-e','-B','-c','-L'):self.assertIn(flag,args)
  self.assertNotIn('-F',args);self.assertNotIn('--fake',args)
  self.assertEqual(args[args.index('-e')+1],'http://127.0.0.1:25432')
  self.assertEqual(cmd[cmd.index('--network')+1],'container:'+ANCHOR);self.assertNotIn('--publish',cmd)
  self.assertEqual(s.mounts()[str(s.key)],('/run/omabit-boot/key',False));self.assertIn(runtime.IMAGE,cmd)

 def test_reopen_retains_named_pier_and_omits_boot_inputs(self):
  first=ship();s=ship(True);self.assertEqual(s.pier,first.pier);self.assertNotEqual(s.name,first.name)
  self.assertEqual(s.mounts(),{str(s.pier_parent):('/urbit',True)})
  for flag in ('-w','-k','-e','-B','-c','-H'):self.assertNotIn(flag,s.arguments())
  self.assertNotIn(str(s.key),','.join(s.create_command()))

 def test_refuses_nonfixture_identities_and_selectors(self):
  for values in ((PROJECT,'lismyl-lismyl',ANCHOR,25432),('dev','nec',ANCHOR,25432),(PROJECT,'nec','name',25432),
                 (PROJECT,'nec',ANCHOR,80),(PROJECT,'nec',ANCHOR,True),(PROJECT,'nec',ANCHOR,65536)):
   with self.subTest(values=values),self.assertRaises(ValueError):runtime.Ship(*values)
  for target in ('all','name','-a',CONTAINER[:12]):
   with self.assertRaises(ValueError):runtime.identity(target)

 def test_inspection_requires_exact_container_generation(self):
  s=ship();self.assertEqual(s.check(inspected(s),CONTAINER)['Id'],CONTAINER)
  for key,value in [('Id','d'*64),('Image','sha256:'+'d'*64),('Name','/another')]:
   row=inspected(s);row[key]=value
   with self.subTest(key=key),self.assertRaises(RuntimeError):s.check(row,CONTAINER)
  row=inspected(s);row['Config']['Cmd']=['-F','nec','/urbit/nec']
  with self.assertRaises(RuntimeError):s.check(row,CONTAINER)

 def test_verified_config_or_manifest_digest_still_requires_exact_pin(self):
  s=ship()
  for digest in (runtime.IMAGE_ID,runtime.IMAGE.split('@',1)[1]):
   row=inspected(s);row['Image']=digest
   self.assertTrue(runtime.image_matches(row));s.check(row,CONTAINER)
   for ref in ('tloncorp/vere:latest','docker.io/tloncorp/vere:latest',runtime.IMAGE+'-changed',None):
    wrong=copy.deepcopy(row);wrong['Config']['Image']=ref
    self.assertFalse(runtime.image_matches(wrong))
    with self.assertRaises(RuntimeError):s.check(wrong,CONTAINER)
  row=inspected(s);row['Image']='sha256:'+'f'*64
  self.assertFalse(runtime.image_matches(row))

 def test_refuses_host_network_publication_and_weakened_limits(self):
  s=ship()
  for key,value in [('NetworkMode','host'),('PortBindings',{'8080/tcp':[]}),('Privileged',True),('Init',False),
                    ('ReadonlyRootfs',False),('Memory',0),('MemorySwap',-1),('NanoCpus',0),('PidsLimit',0),
                    ('CapAdd',['SYS_ADMIN']),('SecurityOpt',[]),('RestartPolicy',{'Name':'always'})]:
   row=inspected(s);row['HostConfig'][key]=value
   with self.subTest(key=key),self.assertRaises(RuntimeError):s.check(row,CONTAINER)

 def test_refuses_mount_substitution_or_writable_key(self):
  s=ship()
  for change in ('key-rw','pier-source','extra','duplicate'):
   row=inspected(s)
   if change=='key-rw':row['Mounts'][1]['RW']=True
   elif change=='pier-source':row['Mounts'][0]['Source']='/home/lab/unrelated'
   elif change=='extra':row['Mounts'].append({'Type':'volume','Destination':'/data'})
   else:row['Mounts'].append(copy.deepcopy(row['Mounts'][0]))
   with self.subTest(change=change),self.assertRaises(RuntimeError):s.check(row,CONTAINER)

 def test_host_refuses_before_docker_or_callback(self):
  e=runtime.Engine({});record={}
  with patch.dict(os.environ,{},clear=True),patch.object(runtime.subprocess,'run') as command:
   for invoke in (lambda:e.execute(['info']),lambda:e.create(ship()),lambda:e.start(ship(),CONTAINER),
                  lambda:e.stop(ship(),CONTAINER,record,lambda:None,lambda:self.fail('Lens called on host'))):
    with self.assertRaisesRegex(RuntimeError,'guest admission'):invoke()
   command.assert_not_called();self.assertEqual(record,{})

 def test_engine_uses_only_dedicated_guest_socket(self):
  e=runtime.Engine({'HOME':'/home/lab','XDG_RUNTIME_DIR':'/run/user/1000'})
  result=runtime.subprocess.CompletedProcess([],0,b'{}',b'')
  with patch.object(runtime,'require_guest'),patch.object(runtime.subprocess,'run',return_value=result) as cmd:
   e.execute(['info']);self.assertEqual(cmd.call_args.args[0][:len(runtime.DOCKER)],runtime.DOCKER)
  with patch.object(runtime,'require_guest'),patch.object(runtime.subprocess,'run') as cmd:
   with self.assertRaisesRegex(RuntimeError,'engine environment'):runtime.Engine({'HOME':'/home/other'}).execute(['info'])
   cmd.assert_not_called()

 def test_diagnostic_capture_keeps_container_stderr_private_and_bounded(self):
  e=runtime.Engine({'HOME':'/home/lab','XDG_RUNTIME_DIR':'/run/user/1000'})
  result=runtime.subprocess.CompletedProcess([],0,b'stdout\nstderr\n',None)
  with patch.object(runtime,'require_guest'),patch.object(runtime.subprocess,'run',return_value=result) as cmd:
   self.assertEqual(e.execute(['logs','--tail','200',CONTAINER],capture_stderr=True),result.stdout)
   self.assertEqual(cmd.call_args.kwargs['stderr'],runtime.subprocess.STDOUT)
   self.assertEqual(cmd.call_args.kwargs['stdout'],runtime.subprocess.PIPE)
   result.stdout=b'x'*(4*1024*1024+1)
   with self.assertRaisesRegex(RuntimeError,'private Docker logs failed'):e.execute(['logs','--tail','200',CONTAINER],capture_stderr=True)

 def test_diagnostic_capture_refuses_unbounded_or_nonexact_targets(self):
  e=runtime.Engine({'HOME':'/home/lab','XDG_RUNTIME_DIR':'/run/user/1000'})
  with patch.object(runtime,'require_guest'),patch.object(runtime.subprocess,'run') as cmd:
   for args in (['info'],['logs',CONTAINER],['logs','--tail','200','all'],['logs','--tail','200',CONTAINER[:12]]):
    with self.assertRaises(ValueError):e.execute(args,capture_stderr=True)
   cmd.assert_not_called()

 def stop(self,s,engine,record,lens):
  def persist():engine.timeline.append(('saved',copy.deepcopy(record)))
  with patch.object(runtime,'require_guest'):return engine.stop(s,CONTAINER,record,persist,lens)

 def test_first_stop_requires_lens_ack_and_observed_zero_exit(self):
  s=ship();e=FakeEngine(s);record={}
  def lens():self.assertTrue(record['dispatched']);self.assertEqual(e.timeline[-1][0],'saved');e.timeline.append(('lens',None));return True
  result=self.stop(s,e,record,lens);self.assertTrue(result['passed']);self.assertEqual(e.commands,[['wait',CONTAINER]])
  self.assertEqual([r[0] for r in e.timeline[:3]],['saved','saved','lens'])

 def test_reopened_stop_sends_only_term_to_exact_container(self):
  s=ship(True);e=FakeEngine(s);record={}
  self.stop(s,e,record,lambda:self.fail('Lens on reopened ship'))
  self.assertEqual(e.commands,[['kill','--signal','TERM',CONTAINER],['wait',CONTAINER]])
  self.assertTrue(record['passed'])

 def test_lost_shutdown_reply_never_resends_and_preserves_uncertainty(self):
  s=ship();e=FakeEngine(s);record={};calls=[]
  def uncertain():calls.append(1);raise TimeoutError('lost fixture reply')
  with self.assertRaises(TimeoutError):self.stop(s,e,record,uncertain)
  self.assertTrue(record['dispatched']);self.assertFalse(record['acknowledged'])
  with self.assertRaisesRegex(RuntimeError,'acknowledged clean exit'):self.stop(s,e,record,uncertain)
  self.assertEqual(calls,[1]);self.assertEqual(e.commands,[['wait',CONTAINER]]);self.assertFalse(record['passed'])

 def test_acknowledged_dispatch_is_not_success_without_clean_exit(self):
  for condition in ('exit','running','oom'):
   s=ship();e=FakeEngine(s);record={}
   if condition=='exit':e.wait_exit=b'137\n'
   if condition=='running':e.wait_running=True
   if condition=='oom':e.wait_oom=True
   with self.subTest(condition=condition),self.assertRaisesRegex(RuntimeError,'clean exit'):self.stop(s,e,record,lambda:True)
   self.assertTrue(record['acknowledged']);self.assertFalse(record['passed'])

 def test_completed_retry_observes_without_another_dispatch(self):
  s=ship();e=FakeEngine(s);record={};self.stop(s,e,record,lambda:True);before=copy.deepcopy(record);commands=copy.deepcopy(e.commands)
  self.assertEqual(self.stop(s,e,record,lambda:self.fail('resent')),before);self.assertEqual(e.commands,commands)

 def test_foreign_record_and_unexplained_dead_container_refuse(self):
  s=ship();e=FakeEngine(s);record={'container':'d'*64,'project':PROJECT,'mode':'lens-exit'}
  with self.assertRaisesRegex(RuntimeError,'another container'):self.stop(s,e,record,lambda:self.fail('sent'))
  e.row['State']['Running']=False
  with self.assertRaisesRegex(RuntimeError,'no owned shutdown'):self.stop(s,e,{},lambda:self.fail('sent'))
  self.assertEqual(e.commands,[])

if __name__=='__main__':unittest.main()
