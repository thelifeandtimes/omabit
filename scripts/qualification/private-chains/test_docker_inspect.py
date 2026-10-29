"""Inert admission/command checks; every systemd/QEMU boundary is mocked."""
import argparse
import json
import os
from pathlib import Path
import shutil
import uuid
import unittest
from unittest.mock import patch
import docker_inspect as audit

class AuditEntryTests(unittest.TestCase):
    def setUp(self):
        self.run=Path('/tmp')/('omabit-import-guest-run-private-chain-audit-test-'+uuid.uuid4().hex)
        self.run.mkdir(mode=0o700);self.addCleanup(shutil.rmtree,self.run)
        self.appliance=self.run/'appliance';(self.appliance/'artifacts').mkdir(parents=True)
        (self.appliance/'artifacts/init').write_bytes(b'code');(self.run/'base').write_bytes(b'pristine')
        (self.run/'lab').mkdir();(self.run/'lab/root.qcow2').write_bytes(b'overlay')
        self.build={'exercise_private_chain':True,'base_source':str(self.run/'base'),'base_source_sha256':audit.sha(self.run/'base'),'artifacts':{'init':audit.sha(self.appliance/'artifacts/init')}}
        self.unit='omabit-import-guest-'+'a'*32
        self.write('inputs.json',{'unit':self.unit,'build':self.build,'appliance':str(self.appliance)})
        self.complete={'scope_admitted':True,'exit':0,'timed_out':False,'clean_shutdown':True};self.write('result.json',self.complete)
    def write(self,name,value):(self.run/name).write_text(json.dumps(value))
    def test_stopped_owned_run_admitted(self):
        with patch.object(audit,'inactive',return_value=True) as stopped:
            _,build,appliance=audit.admit_run(self.run)
        stopped.assert_called_once_with(self.unit);self.assertEqual(build,self.build);self.assertEqual(appliance,self.appliance)
    def test_active_run_refuses_before_hash_or_execution(self):
        with patch.object(audit,'inactive',return_value=False),patch.object(audit,'sha') as sha,patch.object(audit.subprocess,'Popen') as process:
            with self.assertRaisesRegex(ValueError,'inactive'):audit.admit_run(self.run)
            sha.assert_not_called();process.assert_not_called()
    def test_incomplete_or_foreign_case_refused(self):
        for key,value in [('scope_admitted',False),('exit',1),('timed_out',True),('clean_shutdown',False)]:
            with self.subTest(key=key),patch.object(audit,'inactive',return_value=True),patch.object(audit,'sha') as sha:
                self.write('result.json',{**self.complete,key:value})
                with self.assertRaises(ValueError):audit.admit_run(self.run)
                sha.assert_not_called()
        self.write('result.json',self.complete);self.build['exercise_private_chain']=False
        self.write('inputs.json',{'unit':self.unit,'build':self.build,'appliance':str(self.appliance)})
        with patch.object(audit,'inactive',return_value=True),self.assertRaises(ValueError):audit.admit_run(self.run)
    def test_changed_inputs_and_indirect_overlay_refused(self):
        with patch.object(audit,'inactive',return_value=True):
            for path in (self.run/'base',self.appliance/'artifacts/init'):
                original=path.read_bytes();path.write_bytes(b'changed')
                with self.assertRaises(ValueError):audit.admit_run(self.run)
                path.write_bytes(original)
            (self.run/'lab/root.qcow2').unlink();(self.run/'lab/root.qcow2').symlink_to(self.run/'base')
            with self.assertRaises(ValueError):audit.admit_run(self.run)
    def test_nonprivate_directory_refused_before_scope(self):
        self.run.chmod(0o755)
        with patch.object(audit,'inactive') as stopped,self.assertRaises(ValueError):audit.admit_run(self.run)
        stopped.assert_not_called()
    def test_only_exact_qualification_scope_is_read(self):
        for unit in ('urbit-runner','omabit-import-guest-wrong','omabit-private-inspect-'+'a'*32+'.scope'):
            with self.subTest(unit=unit),patch.object(audit.subprocess,'run') as run,self.assertRaises(ValueError):audit.inactive(unit)
            run.assert_not_called()
        with patch.object(audit.subprocess,'run',return_value=argparse.Namespace(returncode=0,stdout='inactive\n')) as run:
            self.assertTrue(audit.inactive(self.unit))
        self.assertEqual(run.call_args.args[0],['systemctl','--user','show',self.unit+'.scope','-p','ActiveState','--value'])
    def test_command_is_readonly_and_has_no_nic(self):
        cmd=audit.command(Path('/tmp/tools'),Path('/tmp/artifacts'),self.appliance,self.build,self.run,'omabit-private-inspect-'+'a'*32)
        self.assertNotIn('--bind',cmd);self.assertIn('--unshare-all',cmd);self.assertIn('--clearenv',cmd)
        self.assertEqual(cmd[cmd.index('-nic')+1],'none');self.assertIn('readonly=on',cmd[cmd.index('-drive')+1])
        self.assertIn('--property=MemoryMax=2G',cmd);self.assertIn('--property=MemorySwapMax=0',cmd)
        binds=[cmd[i+1:i+3] for i,value in enumerate(cmd) if value=='--ro-bind']
        self.assertIn([str(self.run/'lab'),'/guest'],binds);self.assertIn([str(self.run/'base'),'/artifacts/base.raw'],binds)
        self.assertFalse(any('/run/user' in part or 'docker.sock' in part for part in cmd))
    def test_resume_marker_is_bound_to_build(self):
        for value in (False,True):
            self.build['gateway_resume_qualification']=value
            cmd=audit.command(Path('/tmp/tools'),Path('/tmp/artifacts'),self.appliance,self.build,self.run,'omabit-private-inspect-'+'a'*32)
            self.assertEqual('omabit.gateway-resume-audit=required-v1' in cmd[cmd.index('-append')+1],value)
        for value in ('true',1,None):
            self.build['gateway_resume_qualification']=value
            self.write('inputs.json',{'unit':self.unit,'build':self.build,'appliance':str(self.appliance)})
            with self.subTest(value=value),patch.object(audit,'inactive',return_value=True),self.assertRaisesRegex(ValueError,'resume audit selection'):
                audit.admit_run(self.run)

    def test_peer_audit_selection_cannot_omit_resume(self):
        for resume,peers in [(False,True),(True,'true'),(True,1),(True,None)]:
            self.build.update(gateway_resume_qualification=resume,gateway_peer_recovery_qualification=peers)
            self.write('inputs.json',{'unit':self.unit,'build':self.build,'appliance':str(self.appliance)})
            with self.subTest(resume=resume,peers=peers),patch.object(audit,'inactive',return_value=True),self.assertRaisesRegex(ValueError,'peer audit selection'):
                audit.admit_run(self.run)
        self.build.update(gateway_resume_qualification=True,gateway_peer_recovery_qualification=True)
        cmd=audit.command(Path('/tmp/tools'),Path('/tmp/artifacts'),self.appliance,self.build,self.run,'omabit-private-inspect-'+'a'*32)
        self.assertIn('omabit.gateway-peer-audit=required-v1',cmd[cmd.index('-append')+1])
        self.assertIn('omabit.gateway-resume-audit=required-v1',cmd[cmd.index('-append')+1])

    def test_private_isolation_audit_cannot_be_silently_omitted_or_mixed(self):
        self.build['private_project_isolation_qualification']=True
        cmd=audit.command(Path('/tmp/tools'),Path('/tmp/artifacts'),self.appliance,self.build,self.run,'omabit-private-inspect-'+'a'*32)
        self.assertIn('omabit.private-isolation-audit=required-v1',cmd[cmd.index('-append')+1])
        for update in ({'private_project_isolation_qualification':1},{'private_project_isolation_qualification':'yes'},{'private_project_isolation_qualification':True,'gateway_resume_qualification':True}):
            build={**self.build,**update}
            self.write('inputs.json',{'unit':self.unit,'build':build,'appliance':str(self.appliance)})
            with self.subTest(update=update),patch.object(audit,'inactive',return_value=True),patch.object(audit,'sha') as sha,self.assertRaisesRegex(ValueError,'isolation audit selection'):
                audit.admit_run(self.run)
            sha.assert_not_called()

    def test_reused_output_refuses_before_run_read(self):
        with patch.object(audit,'admit_run') as admission,self.assertRaises(ValueError):
            audit.inspect(argparse.Namespace(run=self.run,output=self.run,tools=Path('/tmp/tools')))
        admission.assert_not_called()

if __name__=='__main__':unittest.main()
