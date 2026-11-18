import contextlib
import importlib.util
import io
from pathlib import Path
import unittest
import copy
from unittest.mock import patch
from types import SimpleNamespace


def load(name):
    spec = importlib.util.spec_from_file_location('issues_'+name, Path(__file__).with_name(name+'.py'))
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


class AdmissionTests(unittest.TestCase):
    def test_recovery_prerequisite_binds_exact_artifacts_without_promoting_whole_scenario(self):
        host=load('run'); builds={n:{'sha256':n+'-frozen'} for n in ('omabit','urbit-collector','urbit-runner','managed.test')}
        prior={'source_commit':'frozen','schema_before':59,'passed':False,'inputs':{'candidate':copy.deepcopy(builds)},'installed_upgrade':{k:True for k in ('passed','running_stopped_intent_and_history_preserved','engine_and_container_processes_preserved','access_preserved','old_reader_refused_unchanged')}}
        host.verify_upgrade_section(prior,'frozen',builds)
        self.assertFalse(prior['passed'])
        for change in ('artifact','source','section','old-reader'):
            altered=copy.deepcopy(prior)
            if change=='artifact':altered['inputs']['candidate']['urbit-runner']['sha256']='different'
            if change=='source':altered['source_commit']='different'
            if change=='section':altered['installed_upgrade']['passed']=False
            if change=='old-reader':altered['installed_upgrade']['old_reader_refused_unchanged']=False
            with self.assertRaisesRegex(RuntimeError,'exact passed installed-upgrade'):
                host.verify_upgrade_section(altered,'frozen',builds)

    def test_fixture_switches_existing_access_with_exact_identity_and_revision(self):
        guest=load('guest');guest.report['runner_id']='original-runner'
        with patch.object(guest,'run',side_effect=[SimpleNamespace(stdout='[{"name":"local","runner_id":"original-runner","revision":7}]'),SimpleNamespace(stdout='{}')]) as calls:
            guest.switch_fixture_access(Path('/home/review/fixture-config'))
        args=calls.call_args_list[1].args[0]
        self.assertIn('runner-edit',args)
        self.assertIn('7',args)
        self.assertIn('original-runner',args)
        self.assertNotIn('enroll',args)
        self.assertEqual(guest.report['connection'],'bridge')
        with patch.object(guest,'run',return_value=SimpleNamespace(stdout='[{"name":"local","runner_id":"other","revision":7}]')) as calls:
            with self.assertRaisesRegex(RuntimeError,'enrollment binding changed'):
                guest.switch_fixture_access(Path('/home/review/fixture-config'))
            self.assertEqual(calls.call_count,1)

    def test_baseline_inspects_only_live_ids_once_and_preserves_stopped_history_elsewhere(self):
        guest=load('guest')
        rows={'instances':[{'id':'stopped','observed_state':'stopped','runtime_status':{'container_id':'historical-removed'}},{'id':'running','observed_state':'running','runtime_status':{'container_id':'owned-live'}}]}
        with patch.object(guest,'collector',return_value=rows), patch.object(guest,'run',return_value=SimpleNamespace(stdout='same-engine')), patch.object(guest,'inspect',return_value={'State':{'Running':True,'Pid':123,'StartedAt':'fixed-start'}}) as native:
            observed=guest.runtime('previous')
        native.assert_called_once_with('owned-live')
        self.assertEqual(set(observed['containers']),{'running'})
        self.assertEqual(observed['containers']['running']['container_id'],'owned-live')
        self.assertEqual(rows['instances'][0]['runtime_status']['container_id'],'historical-removed')

    def test_no_run_and_existing_capsule_cannot_reach_effects(self):
        host = load('run')
        with patch.object(host, 'charge', side_effect=AssertionError('storage/effect admission reached')), patch.object(host.subprocess,'check_output',side_effect=AssertionError('subprocess reached')):
            with patch('sys.argv',['run.py']), contextlib.redirect_stdout(io.StringIO()):
                host.main()
            with patch('sys.argv',['run.py','--run','--portable','/input','--previous','/input','--candidate','/input','--rootless','/input','--runtime','/input','--helper','/input','--output','/tmp']), contextlib.redirect_stderr(io.StringIO()):
                with self.assertRaises(SystemExit):
                    host.main()

    def test_guest_rejects_host_before_commands_or_filesystem_mutation(self):
        guest = load('guest')
        with patch.object(guest.os,'geteuid',return_value=1000), patch.object(guest.subprocess,'run',side_effect=AssertionError('guest effect reached')), patch.object(guest.Path,'mkdir',side_effect=AssertionError('guest state write reached')):
            with self.assertRaisesRegex(RuntimeError,'refused outside'):
                guest.main()


if __name__ == '__main__':
    unittest.main()
