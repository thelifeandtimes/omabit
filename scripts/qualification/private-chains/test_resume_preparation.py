"""Inert selection of a fresh gateway-resume experiment; no appliance is built."""
import unittest
from pathlib import Path
import docker_appliance as prep

class ResumePreparationTests(unittest.TestCase):
 def test_original_source_selection_unchanged(self):
  files=prep.gateway_sources(False)
  self.assertEqual(files['guest.py'],prep.HERE/'docker_guest.py')
  self.assertEqual(set(files),{'guest.py','docker_gateway.cjs','docker_peer.py','docker_case.py','docker_runtime.py'})
 def test_resume_adds_only_explicit_retained_gateway_sources(self):
  files=prep.gateway_sources(True)
  self.assertEqual(files['guest.py'],prep.HERE/'docker_resume_guest.py')
  self.assertEqual(set(files)-set(prep.gateway_sources(False)),{'docker_resume_case.py','docker_gateway_resume.cjs','gateway_resume_contract.cjs'})
  for name in set(prep.gateway_sources(False))-{'guest.py'}:self.assertEqual(files[name],prep.gateway_sources(False)[name])
 def test_resume_selection_requires_a_boolean(self):
  for value in ('true',1,None,[]):
   with self.subTest(value=value),self.assertRaises(ValueError):prep.gateway_sources(value)
 def test_guest_variant_retains_original_installer_and_peer_case(self):
  old=(prep.HERE/'docker_guest.py').read_text();new=(prep.HERE/'docker_resume_guest.py').read_text()
  start=new.index('        import docker_resume_case\n');end=new.index("        report['ok']=True",start)
  self.assertEqual(new[:start]+new[end:],old)
  self.assertLess(new.index("if not proof['ok']"),start)

if __name__=='__main__':unittest.main()
