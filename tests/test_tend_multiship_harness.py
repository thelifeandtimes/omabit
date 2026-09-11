import importlib.util
from pathlib import Path
import sys
import unittest


ROOT = Path(__file__).parents[1]
HARNESS_PATH = ROOT / "scripts" / "check-tend-multiship.py"
SPEC = importlib.util.spec_from_file_location("tend_multiship_harness", HARNESS_PATH)
assert SPEC is not None and SPEC.loader is not None
harness = importlib.util.module_from_spec(SPEC)
sys.modules[SPEC.name] = harness
SPEC.loader.exec_module(harness)


class TendMultishipHarnessTests(unittest.TestCase):
    def test_canonical_list_ignores_only_the_local_alias(self):
        first = {"id": 1, "title": "Shared", "revision": 4, "created-at": "~2026.9.1..01.02.03", "reminders": [{"id": 7}]}
        replica = {"id": 88, "title": "Shared", "revision": 4, "created-at": "~2026.09.01..01.02.03", "reminders": [{"id": 7}]}
        changed = {"id": 88, "title": "Shared", "revision": 5, "reminders": [{"id": 7}]}

        self.assertEqual(harness.canonical_list(first), harness.canonical_list(replica))
        self.assertNotEqual(harness.canonical_list(first), harness.canonical_list(changed))

    def test_rejection_reason_distinguishes_commits_and_rejections(self):
        self.assertIsNone(harness.rejection_reason({"list-upserted": {"operation-id": "ok"}}))
        self.assertEqual(
            harness.rejection_reason({"rejected": {"reason": "stale-list"}}),
            "stale-list",
        )

    def test_release_bounds_are_exposed_by_the_parser(self):
        source = HARNESS_PATH.read_text(encoding="utf-8")
        self.assertIn("--backpressure-count must be between 2 and 1000", source)
        self.assertIn("--restart-cycles must be between 0 and 100", source)
        self.assertIn("kiln-suspend-many", source)
        self.assertIn("helm-ames-snub", source)
        self.assertIn('f"[%deny {ships}]"', source)


if __name__ == "__main__":
    unittest.main()
