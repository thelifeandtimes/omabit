"""Refuse incomplete public initial cases before any filesystem effects."""
from types import SimpleNamespace
import unittest
from unittest.mock import patch
import appliance
class PublicInitialEntryTest(unittest.TestCase):
    def test_requires_storage_before_preparation(self):
        args=SimpleNamespace(exercise_project_public_initial=True,exercise_project_member_storage=False)
        with patch.object(appliance.storage,'admit',side_effect=AssertionError('unexpected storage')):
            with self.assertRaisesRegex(ValueError,'requires --exercise-project-member-storage'):
                appliance.prepare(args)
if __name__=='__main__':unittest.main()
