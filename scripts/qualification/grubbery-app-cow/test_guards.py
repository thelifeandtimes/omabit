import ast
import contextlib
import importlib.util
import io
import json
from pathlib import Path
import sys
import unittest
from unittest.mock import patch


def module(name):
 spec=importlib.util.spec_from_file_location('cow_'+name,Path(__file__).with_name(name+'.py'))
 value=importlib.util.module_from_spec(spec);spec.loader.exec_module(value);return value


class COWGuards(unittest.TestCase):
 def test_no_run_does_not_open_predecessor_or_spawn(self):
  host=module('run')
  with patch.object(sys,'argv',['run.py']),patch.object(host.subprocess,'check_output',side_effect=AssertionError('no process')),patch.object(host.budget,'charge',side_effect=AssertionError('no admission')),contextlib.redirect_stdout(io.StringIO()):host.main()

 def test_original_capsule_name_cannot_be_output(self):
  host=module('run');args=['run.py','--run','--output','/tmp/omabit-issues-app-vm-05']
  for key in ('portable','candidate','rootless','runtime','framework-source','helper'):args+=['--'+key,'/missing']
  with patch.object(sys,'argv',args),patch.object(host.os,'geteuid',return_value=1000),patch.object(host.subprocess,'check_output',side_effect=AssertionError('must refuse before source/input access')),contextlib.redirect_stderr(io.StringIO()):
   with self.assertRaises(SystemExit) as stopped:host.main()
  self.assertEqual(stopped.exception.code,2)

 def test_guest_refuses_host_and_wrong_binding_before_operations(self):
  guest=module('guest')
  with patch.object(guest.os,'geteuid',return_value=1000),patch.object(Path,'read_text',side_effect=AssertionError('host read forbidden')):
   with self.assertRaises(RuntimeError):guest.guard()
  scenario={'kind':'issues18-product-grubbery-v1','case':'cow-fresh-nec-v1','predecessor_root_sha256':guest.PIN,'target_schema':62,'development_snapshot':True}
  values={'/sys/class/dmi/id/sys_vendor':'QEMU','/proc/cmdline':'omabit.disposable-guest=release-upgrade-v1','/opt/omabit-release-input/scenario.json':json.dumps(scenario)}
  with patch.object(guest.os,'geteuid',return_value=0),patch.object(sys,'argv',['guest.py']),patch.object(Path,'read_text',lambda p,**kwargs:values[str(p)]):
   self.assertEqual(guest.guard(),scenario)
   for change in ({'predecessor_root_sha256':'0'*64},{'case':'source-state-restart-v1'},{'target_schema':60},{'development_snapshot':False}):
    values['/opt/omabit-release-input/scenario.json']=json.dumps({**scenario,**change})
    with self.assertRaises(RuntimeError):guest.guard()

 def test_bootstrap_does_not_replace_original_or_enable_resume(self):
  source=Path(__file__).with_name('init').joinpath('main.go').read_text()
  self.assertIn('ConditionPathExists=/run/omabit-nec-runner-ready',source)
  self.assertIn('serve-development --acknowledge-development\n',source)
  self.assertNotIn('--resume-running',source)
  host=Path(__file__).with_name('run.py').read_text()
  self.assertIn("'--ro-bind', str(PREDECESSOR), '/predecessor.qcow2'",host)
  self.assertIn("'-b', '/predecessor.qcow2'",host)
  self.assertEqual(module('run').SCRATCH,896*1024**2)

if __name__=='__main__':unittest.main()
