"""Refuse incomplete or mixed public-member fixtures before filesystem effects."""
from types import SimpleNamespace
import unittest
from unittest.mock import patch
import appliance
class PublicMembersEntryTest(unittest.TestCase):
    def test_requires_lifecycle_and_excludes_mixed_entries(self):
        for lifecycle,initial,review in [(False,False,False),(True,True,False),(True,False,True)]:
            args=SimpleNamespace(exercise_project_public_members=True,exercise_project_context_lifecycle=lifecycle,exercise_project_public_initial=initial,exercise_project_anchor_review=review)
            with self.subTest(lifecycle=lifecycle,initial=initial,review=review), patch.object(appliance.storage,'admit',side_effect=AssertionError('unexpected storage')):
                with self.assertRaisesRegex(ValueError,'requires --exercise-project-context-lifecycle'):
                    appliance.prepare(args)
if __name__=='__main__':unittest.main()
