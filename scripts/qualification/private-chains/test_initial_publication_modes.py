"""Only source selection, launch capture and refusal checks; no runtime effects."""
import importlib.util
import os
from pathlib import Path
import tempfile
import types
import unittest
from unittest.mock import patch

import docker_appliance as prep
import docker_initial_publication as case
import docker_initial_publication_guest as guest
import docker_inspect as reader
import docker_runtime

class InitialModes(unittest.TestCase):
 def test_initial_payload_has_complete_python_closure_and_separate_entry(self):
  files=prep.gateway_sources(False,initial_publication=True)
  self.assertEqual(files['guest.py'],prep.HERE/'docker_initial_publication_guest.py')
  self.assertEqual(set(files)-set(prep.gateway_sources(False)),{'docker_initial_publication.py','docker_publication.py','docker_published_client.py','docker_gateway_peers.py'})
  self.assertNotIn('docker_gateway_resume.cjs',files)
  for args in [(True,False,False,False,True),(True,True,True,False,True),(False,False,False,True,True),(False,False,False,False,1)]:
   with self.subTest(args=args),self.assertRaises(ValueError):prep.gateway_sources(*args)

 def test_callback_refuses_host_before_reading_state_or_using_engine(self):
  with patch.dict(os.environ,{},clear=True),patch.object(docker_runtime.os,'geteuid',return_value=1000),patch.object(Path,'mkdir') as mkdir,patch.object(case,'PublishedTransport') as transport,patch.object(case,'private_json') as save:
   with self.assertRaisesRegex(RuntimeError,'guest admission'):case.capture(None,None,None,None,None)
   mkdir.assert_not_called();transport.assert_not_called();save.assert_not_called()

 def test_initial_guest_refuses_before_packages_or_services(self):
  with patch.object(guest.os,'geteuid',return_value=1000),patch.object(guest.subprocess,'run') as run:
   with self.assertRaisesRegex(RuntimeError,'wrong guest marker/uid'):guest.main()
   run.assert_not_called()

 def test_reader_mode_is_mandatory_without_replacement_flags(self):
  cmd=reader.command(Path('/tools'),Path('/artifacts'),Path('/appliance'),{'base_source':'/base','initial_http_publication_qualification':True},Path('/run'),'omabit-private-inspect-'+'a'*32)
  kernel=cmd[cmd.index('-append')+1]
  self.assertIn('omabit.initial-publication-audit=required-v1',kernel)
  for other in ('omabit.gateway-resume-audit=','omabit.gateway-peer-audit=','omabit.private-publication-audit=','omabit.private-isolation-audit='):self.assertNotIn(other,kernel)
  self.assertEqual(cmd[cmd.index('-nic')+1],'none')
  self.assertIn('readonly=on',cmd[cmd.index('-drive')+1])

 def test_reader_rejects_conflicting_build_before_artifact_access(self):
  for value in [1,'yes',True]:
   for other in ['gateway_resume_qualification','private_project_isolation_qualification']:
    with self.subTest(value=value,other=other),tempfile.TemporaryDirectory(prefix='omabit-import-guest-run-private-chain-mode-') as name:
     root=Path(name);build={'exercise_private_chain':True,'initial_http_publication_qualification':value,other:True}
     inputs={'unit':'omabit-import-guest-'+'a'*32,'build':build}
     completed={'scope_admitted':True,'exit':0,'timed_out':False,'clean_shutdown':True}
     with patch.object(reader,'read_json',side_effect=[inputs,completed]),patch.object(reader,'inactive',return_value=True),patch.object(reader,'sha') as sha:
      with self.assertRaises(ValueError):reader.admit_run(root)
      sha.assert_not_called()

if __name__=='__main__':unittest.main()
