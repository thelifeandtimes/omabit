from pathlib import Path
import unittest


ROOT = Path(__file__).parents[1]


class TendGallSourceContractTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.agent = (ROOT / "desk" / "app" / "tend.hoon").read_text(encoding="utf-8")

    def test_mutation_resource_ceilings_remain_server_side(self):
        for source_contract in (
            "(lent ~(tap by list-map.state)) 10.000",
            "(lent ~(tap by sections.u.old)) 10.000",
            "(lent ~(tap by reminders.u.old)) 100.000",
            "(met 3 notes.act) 65.536",
            "(met 3 u.url) 8.192",
            "(lent offsets) 64",
            "(lent snooze-presets.prefs) 32",
        ):
            self.assertIn(source_contract, self.agent)

    def test_assignment_is_checked_against_owner_or_member(self):
        self.assertIn("(valid-assignee list-id.act assignee.act state)", self.agent)
        self.assertIn("=(u.assignee our.bowl)", self.agent)
        self.assertIn("(~(has by members.u.sharing) u.assignee)", self.agent)


if __name__ == "__main__":
    unittest.main()
