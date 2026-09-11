from pathlib import Path
import unittest


ROOT = Path(__file__).parents[1]


class TendGallSourceContractTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.agent = (ROOT / "desk" / "app" / "tend.hoon").read_text(encoding="utf-8")
        cls.migration = (ROOT / "desk" / "lib" / "tend-migrate.hoon").read_text(encoding="utf-8")
        cls.migration_fixtures = (ROOT / "desk" / "gen" / "tend-migrations.hoon").read_text(encoding="utf-8")
        cls.surface = (ROOT / "desk" / "sur" / "tend.hoon").read_text(encoding="utf-8")

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

    def test_manual_placement_is_atomic_and_normalizes_sibling_ranks(self):
        self.assertIn("%place-reminder", self.agent)
        self.assertIn("(ordered-siblings reminders.u.old parent-id.u.source section-id.u.source)", self.agent)
        self.assertIn("(rerank-siblings placed reminders.u.old)", self.agent)
        self.assertIn("next-rank (add next-rank 1.024)", self.agent)

    def test_section_placement_is_atomic_and_normalizes_ranks(self):
        self.assertIn("%place-section", self.agent)
        self.assertIn("(place-section-in-order ordered u.source target-id.act after.act)", self.agent)
        self.assertIn("(rerank-sections placed sections.u.old)", self.agent)

    def test_moving_a_parent_cascades_section_to_descendants(self):
        self.assertIn("(descendant id.item reminder-id.act reminders.u.old)", self.agent)
        self.assertIn("item(section-id section-id.act, revision +(revision.item), modified-at now.bowl)", self.agent)

    def test_batch_completion_records_each_scheduled_occurrence(self):
        self.assertIn("%batch-set-completed -.act", self.agent)
        self.assertIn("(scot %uv (sham [op-id.act id]))", self.agent)
        self.assertIn("?:(completed.act (activity-occurrence-for u.target id before) ~)", self.agent)

    def test_state_ten_migrates_private_presentation_and_notification_defaults(self):
        self.assertIn("(migrate:tend-migrate now.bowl old-state)", self.agent)
        self.assertIn("++  migrate", self.migration)
        self.assertIn("++  upgrade-9", self.migration)
        self.assertIn("+$  saved-state", self.surface)
        self.assertIn("*list-presentations:t", self.migration)
        self.assertIn("*collaboration-policies:t", self.migration)
        self.assertIn("*collaboration-notifications:t", self.migration)

    def test_every_saved_schema_has_a_nonempty_executable_fixture(self):
        for version in range(11):
            self.assertIn(f"[{version} [%{version} ", self.migration_fixtures)
        self.assertIn("'fixture-list'", self.migration_fixtures)
        self.assertIn("(~(has by reminders.u.migrated-list) 7)", self.migration_fixtures)
        self.assertIn("Tend saved-state migrations %0 through %10 passed", self.migration_fixtures)

    def test_private_settings_are_not_routed_as_shared_mutations(self):
        for source_contract in (
            "%set-list-order           ~",
            "%set-list-presentation    ~",
            "%set-collaboration-policy  ~",
        ):
            self.assertIn(source_contract, self.agent)

    def test_collaboration_alerts_are_local_durable_and_actor_filtered(self):
        self.assertIn("?:  =(actor.event our.bowl)  $(events t.events)", self.agent)
        self.assertIn("(collaboration-policy-enabled u.kind policy)", self.agent)
        self.assertIn("nex(collaboration-notification-map notices)", self.agent)
        self.assertIn("(queue-collaboration-activities-fact new-activities visible nex)", self.agent)

    def test_removed_lists_prune_all_local_alert_state(self):
        self.assertIn("notification-map  (without-notifications alias notification-map.st)", self.agent)
        self.assertIn("replica-alert-set  (without-replica-alerts alias replica-alert-set.st)", self.agent)
        self.assertIn("(prune-notifications-load notification-map.st visible)", self.agent)
        self.assertIn("(prune-replica-alerts-load replica-alert-set.st visible)", self.agent)


if __name__ == "__main__":
    unittest.main()
