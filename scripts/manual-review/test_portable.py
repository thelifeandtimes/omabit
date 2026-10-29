#!/usr/bin/env python3
"""Small contract checks; no VM, network, installer or persistent test disks."""
import hashlib,importlib.util,json,os,subprocess,sys,tempfile,unittest
from pathlib import Path
from unittest.mock import patch
HERE=Path(__file__).resolve().parent
def load(name,path):
 s=importlib.util.spec_from_file_location(name,path);m=importlib.util.module_from_spec(s);s.loader.exec_module(m);return m
vm=load('portable_vm',HERE/'vm.py');pull=load('pull_vm',HERE/'get-omabit-vm.py')
class Contracts(unittest.TestCase):
 def test_seal_detects_file_and_symlink_changes(self):
  with tempfile.TemporaryDirectory() as d:
   root=Path(d);(root/'input').write_bytes(b'good');(root/'alias').symlink_to('input')
   (root/'bundle.json').write_text(json.dumps({'format':1,'sha256':{'input':hashlib.sha256(b'good').hexdigest()},'symlinks':{'alias':'input'}}))
   with patch.object(vm,'ROOT',root):
    vm.verify();(root/'alias').unlink();(root/'alias').symlink_to('/tmp/unrelated')
    with self.assertRaisesRegex(RuntimeError,'symlink'):vm.verify()
    (root/'alias').unlink();(root/'alias').symlink_to('input');(root/'input').write_bytes(b'bad')
    with self.assertRaisesRegex(RuntimeError,'checksum'):vm.verify()
 def test_guest_mount_and_network_boundary(self):
  for network in ['off','internet']:
   args=vm.qemu_command({'memory':6,'cpus':2,'network':network,'accelerator':'tcg'})
   self.assertEqual('--share-net' in args,network=='internet')
   self.assertNotIn('hostfwd',' '.join(args));self.assertNotIn('/home',args);self.assertNotIn('/run/user/1000',args)
   i=args.index(str(vm.ROOT/'base.qcow2'));self.assertEqual(args[i-1],'--ro-bind')
   self.assertNotIn('--dev-bind',args)
 def run_pull(self,dest):
  with patch.object(sys,'argv',['get-omabit-vm.py','--to',str(dest),'--no-start']),patch.object(pull.os,'geteuid',return_value=1000),patch.object(pull.shutil,'which',return_value='/usr/bin/tool'):
   pull.main()
 def test_existing_session_never_refetches(self):
  with tempfile.TemporaryDirectory() as d:
   root=Path(d);(root/'state').mkdir();(root/'state/session.qcow2').write_bytes(b'preserve');(root/'bundle.json').write_text(json.dumps({'build':pull.BUILD}))
   with patch.object(pull.subprocess,'run') as run,patch.object(pull.subprocess,'check_output') as remote:
    self.run_pull(root);remote.assert_not_called();self.assertEqual((root/'state/session.qcow2').read_bytes(),b'preserve');self.assertEqual(len(run.call_args_list),2)
 def test_partial_transfer_resumes_but_excludes_sessions(self):
  with tempfile.TemporaryDirectory() as d:
   dest=Path(d)/'review'
   def interrupted(args,**kw):
    if args[0]=='rsync':(dest/'base.qcow2').write_bytes(b'partial');raise subprocess.CalledProcessError(23,args)
   remote=json.dumps({'build':pull.BUILD})
   with patch.object(pull.subprocess,'check_output',return_value=remote),patch.object(pull.subprocess,'run',side_effect=interrupted):
    with self.assertRaises(subprocess.CalledProcessError):self.run_pull(dest)
   self.assertTrue((dest/'.omabit-pull.json').exists())
   def resumed(args,**kw):
    if args[0]=='rsync':
     self.assertIn('--exclude=/state/',args);self.assertIn('--checksum',args);self.assertNotIn('--delete',args)
     (dest/'bundle.json').write_text(remote)
   with patch.object(pull.subprocess,'check_output',return_value=remote),patch.object(pull.subprocess,'run',side_effect=resumed):self.run_pull(dest)
   self.assertFalse((dest/'.omabit-pull.json').exists())
 def test_unrelated_destination_is_untouched(self):
  with tempfile.TemporaryDirectory() as d:
   dest=Path(d);(dest/'important').write_text('keep')
   with self.assertRaisesRegex(RuntimeError,'not empty'):self.run_pull(dest)
   self.assertEqual((dest/'important').read_text(),'keep')
 def test_different_build_is_not_overwritten(self):
  with tempfile.TemporaryDirectory() as d:
   dest=Path(d);(dest/'bundle.json').write_text(json.dumps({'build':'other'}))
   with self.assertRaisesRegex(RuntimeError,'different build'):self.run_pull(dest)
if __name__=='__main__':unittest.main()
