"""Reject incomplete continuation-case selection before any preparation effect."""
from types import SimpleNamespace
import unittest
from unittest.mock import patch
import appliance

class ContinuationEntryTest(unittest.TestCase):
    def test_requires_complete_recovery_case_before_storage_or_package_access(self):
        args = SimpleNamespace(exercise_project_anchor_continuation=True, exercise_project_anchor_recovery=False)
        with patch.object(appliance.storage, 'admit', side_effect=AssertionError('unexpected storage admission')):
            with self.assertRaisesRegex(ValueError, 'requires --exercise-project-anchor-recovery'):
                appliance.prepare(args)

if __name__ == '__main__':
    unittest.main()
