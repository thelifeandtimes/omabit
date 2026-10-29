"""Refuse incomplete public recovery cases before any filesystem effects."""
from types import SimpleNamespace
import unittest
from unittest.mock import patch
import appliance
class PublicRecoveryEntryTest(unittest.TestCase):
    def test_requires_continuation_before_storage_or_package_access(self):
        args=SimpleNamespace(exercise_project_public_recovery=True,exercise_project_anchor_continuation=False)
        with patch.object(appliance.storage,'admit',side_effect=AssertionError('unexpected admission')):
            with self.assertRaisesRegex(ValueError,'requires --exercise-project-anchor-continuation'):
                appliance.prepare(args)
if __name__=='__main__':unittest.main()
