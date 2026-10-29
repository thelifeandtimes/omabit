import importlib.util
from pathlib import Path
import unittest

class CPUAdmission(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        spec=importlib.util.spec_from_file_location('appliance_cpu',Path(__file__).with_name('appliance.py'))
        cls.module=importlib.util.module_from_spec(spec);spec.loader.exec_module(cls.module)

    def test_legacy_single_core_boundary(self):
        value=self.module.cpu_settings({},1)
        self.assertEqual(value,{'cores':1,'quota':'100%','scope_quota':'1s','tcg_thread':'single'})

    def test_bounded_runtime_emulation(self):
        for mode in ('exercise_import','exercise_fake_network','exercise_project_member_launch','exercise_private_chain'):
            value=self.module.cpu_settings({mode:True},2)
            self.assertEqual(value,{'cores':2,'quota':'200%','scope_quota':'2s','tcg_thread':'multi'})

    def test_no_implicit_or_unbounded_cpu_increase(self):
        for build in ({},{'exercise_project_public_initial':True},{'exercise_project_member_launch':False}):
            with self.assertRaises(ValueError):self.module.cpu_settings(build,2)
        for count in (0,3,16,True,2.0,'2',None):
            with self.assertRaises(ValueError):self.module.cpu_settings({'exercise_project_member_launch':True},count)

if __name__=='__main__':unittest.main()
