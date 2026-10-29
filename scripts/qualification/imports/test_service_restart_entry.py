"""Refuse mixed or incomplete service-restart fixtures before filesystem effects."""
from types import SimpleNamespace
import unittest
from unittest.mock import patch
import appliance
class ServiceRestartEntryTest(unittest.TestCase):
    def test_requires_only_public_recovery_before_effects(self):
        for recovery,initial,members in [(False,False,False),(True,True,False),(True,False,True)]:
            args=SimpleNamespace(exercise_project_service_restart=True,exercise_project_public_recovery=recovery,exercise_project_public_initial=initial,exercise_project_public_members=members)
            with self.subTest(recovery=recovery,initial=initial,members=members), patch.object(appliance.storage,'admit',side_effect=AssertionError('unexpected storage')):
                with self.assertRaisesRegex(ValueError,'requires public-recovery'):
                    appliance.prepare(args)
if __name__=='__main__':unittest.main()
