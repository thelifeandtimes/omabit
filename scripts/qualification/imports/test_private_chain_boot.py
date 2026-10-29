"""Inert VM launch-boundary checks; subprocess execution is replaced entirely."""
import importlib.util,json,hashlib,subprocess,sys,tempfile,types,unittest
from pathlib import Path
from unittest.mock import patch
HERE=Path(__file__).resolve().parent;sys.path.insert(0,str(HERE))
spec=importlib.util.spec_from_file_location('private_boot',HERE/'appliance.py');module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
class CapturedVM(Exception):pass
class PrivateChainBoot(unittest.TestCase):
 def fixture(self,root,private=True):
  appliance=root/'appliance';art=appliance/'artifacts';art.mkdir(parents=True)
  for n in ('base.raw','vmlinuz','initramfs.cpio'):(art/n).write_bytes(b'fixture-'+n.encode())
  base=root/'pristine.raw';base.write_bytes(b'private test pristine bytes')
  build={'exercise_private_chain':True} if private else {'exercise_import':True}
  build.update(import_mode='rename',root_disk_gib=12,artifacts={p.name:module.sha(p) for p in art.iterdir()})
  if private:build.update(base_source=str(base),base_source_sha256=module.sha(base))
  (appliance/'build.json').write_text(json.dumps(build))
  return types.SimpleNamespace(appliance=appliance,tools=root/'tools',output=root/'run',cpu_cores=2),build,base
 def capture(self,args,mem_kib=16*1024**2,expect=CapturedVM):
  original=Path.read_text
  def read(path,*a,**kw):return 'MemAvailable: '+str(mem_kib)+' kB\n' if path==Path('/proc/meminfo') else original(path,*a,**kw)
  def fresh(path,prefix):path.mkdir(mode=0o700);return path
  helper=types.SimpleNamespace(tool_evidence=lambda p:{'fixture':True},tool_env=lambda p:{})
  with patch.object(module.storage,'admit',return_value={'available_bytes':9*1024**3}),patch.object(module,'fresh',side_effect=fresh),patch.object(module,'helper',return_value=helper),patch.object(Path,'read_text',read),patch.object(module.subprocess,'run',return_value=subprocess.CompletedProcess([],0)) as commands,patch.object(module.subprocess,'Popen',side_effect=CapturedVM) as spawn:
   with self.assertRaises(expect):module.boot(args)
   return commands.call_args_list,spawn.call_args_list
 def test_private_launch_matches_qualified_boundary(self):
  with tempfile.TemporaryDirectory(prefix='omabit-private-boot-check-') as d:
   args,build,base=self.fixture(Path(d));commands,spawn=self.capture(args);self.assertEqual(len(commands),1);self.assertEqual(len(spawn),1);cmd=spawn[0].args[0]
   for flag in ('--property=MemoryMax=12G','--property=MemorySwapMax=0','--property=CPUQuota=200%','--property=TasksMax=128','--unshare-all','--die-with-parent'):self.assertIn(flag,cmd)
   self.assertEqual(cmd[cmd.index('-nic')+1],'none');self.assertEqual(cmd[cmd.index('-m')+1],'7168M');self.assertEqual(cmd[cmd.index('-smp')+1],'2');self.assertNotIn('-netdev',cmd)
   index=cmd.index(str(base));self.assertEqual(cmd[index-1:index+2],['--ro-bind',str(base),'/artifacts/base.raw'])
   self.assertEqual(cmd.count('--bind'),1);index=cmd.index('--bind');self.assertEqual(cmd[index+1:index+3],[str(args.output/'lab'),'/lab'])
   inputs=json.loads((args.output/'inputs.json').read_text());self.assertEqual(inputs['deadline_seconds'],3600);self.assertEqual(inputs['memory_available_at_start'],16*1024**3)
 def test_changed_backing_refuses_before_any_process(self):
  with tempfile.TemporaryDirectory(prefix='omabit-private-boot-check-') as d:
   args,build,base=self.fixture(Path(d));base.write_bytes(b'changed');commands,spawn=self.capture(args,expect=ValueError);self.assertEqual(commands,[]);self.assertEqual(spawn,[])
 def test_memory_floor_refuses_vm_start(self):
  with tempfile.TemporaryDirectory(prefix='omabit-private-boot-check-') as d:
   args,_,_=self.fixture(Path(d));commands,spawn=self.capture(args,mem_kib=15*1024**2,expect=RuntimeError);self.assertEqual(len(commands),1);self.assertEqual(spawn,[])
 def test_malformed_or_mixed_case_refuses_before_output(self):
  for update in ({'exercise_private_chain':'yes'},{'exercise_fake_network':True},{'network_isolation':True},{'root_disk_gib':24},{'import_mode':'copy'}):
   with self.subTest(update=update),tempfile.TemporaryDirectory(prefix='omabit-private-boot-check-') as d:
    args,build,_=self.fixture(Path(d));build.update(update);(args.appliance/'build.json').write_text(json.dumps(build));commands,spawn=self.capture(args,expect=ValueError);self.assertFalse(args.output.exists());self.assertEqual(commands,[]);self.assertEqual(spawn,[])
 def test_two_project_boundary_and_memory_floor(self):
  for memory,expect in ((32,CapturedVM),(31,RuntimeError)):
   with self.subTest(memory=memory),tempfile.TemporaryDirectory(prefix='omabit-private-boot-check-') as d:
    args,build,_=self.fixture(Path(d));build['private_project_isolation_qualification']=True;(args.appliance/'build.json').write_text(json.dumps(build))
    _,spawn=self.capture(args,mem_kib=memory*1024**2,expect=expect)
    if memory==31:self.assertEqual(spawn,[]);continue
    cmd=spawn[0].args[0];self.assertIn('--property=MemoryMax=24G',cmd);self.assertEqual(cmd[cmd.index('-m')+1],'14336M')
    self.assertEqual(cmd[cmd.index('-nic')+1],'none');self.assertEqual(cmd.count('--bind'),1)
    self.assertEqual(json.loads((args.output/'inputs.json').read_text())['deadline_seconds'],7200)
 def test_isolation_mode_rejects_ambiguous_build_before_output(self):
  for update in ({'private_project_isolation_qualification':1},{'private_project_isolation_qualification':'yes'},{'private_project_isolation_qualification':True,'gateway_resume_qualification':True},{'private_project_isolation_qualification':True,'exercise_private_chain':False}):
   with self.subTest(update=update),tempfile.TemporaryDirectory(prefix='omabit-private-boot-check-') as d:
    args,build,_=self.fixture(Path(d));build.update(update);(args.appliance/'build.json').write_text(json.dumps(build))
    commands,spawn=self.capture(args,expect=ValueError);self.assertEqual(commands,[]);self.assertEqual(spawn,[]);self.assertFalse(args.output.exists())
 def test_initial_publication_retains_single_pair_resource_boundary(self):
  with tempfile.TemporaryDirectory(prefix='omabit-private-boot-check-') as d:
   args,build,_=self.fixture(Path(d));build['initial_http_publication_qualification']=True;(args.appliance/'build.json').write_text(json.dumps(build))
   _,spawn=self.capture(args);cmd=spawn[0].args[0];self.assertIn('--property=MemoryMax=12G',cmd);self.assertEqual(cmd[cmd.index('-m')+1],'7168M');self.assertEqual(cmd[cmd.index('-nic')+1],'none')
   self.assertEqual(json.loads((args.output/'inputs.json').read_text())['deadline_seconds'],3600)
 def test_initial_publication_refuses_mixed_modes_before_output(self):
  cases=[{'initial_http_publication_qualification':v} for v in [1,'yes',None]]
  cases += [dict(initial_http_publication_qualification=True,**{key:True}) for key in ['gateway_resume_qualification','gateway_peer_recovery_qualification','http_publication_qualification','private_project_isolation_qualification']]
  cases += [dict(initial_http_publication_qualification=True,exercise_private_chain=False)]
  for update in cases:
   with self.subTest(update=update),tempfile.TemporaryDirectory(prefix='omabit-private-boot-check-') as d:
    args,build,_=self.fixture(Path(d));build.update(update);(args.appliance/'build.json').write_text(json.dumps(build));commands,spawn=self.capture(args,expect=ValueError);self.assertEqual(commands,[]);self.assertEqual(spawn,[]);self.assertFalse(args.output.exists())
 def test_existing_import_boundary_is_preserved(self):
  with tempfile.TemporaryDirectory(prefix='omabit-private-boot-check-') as d:
   args,_,base=self.fixture(Path(d),private=False);commands,spawn=self.capture(args);cmd=spawn[0].args[0];self.assertNotIn(str(base),cmd);inputs=json.loads((args.output/'inputs.json').read_text());self.assertEqual(inputs['deadline_seconds'],2400);self.assertIsNone(inputs['memory_available_at_start']);self.assertEqual(inputs['guest_ram'],'7168M');self.assertEqual(inputs['cpu']['cores'],2)
if __name__=='__main__':unittest.main()
