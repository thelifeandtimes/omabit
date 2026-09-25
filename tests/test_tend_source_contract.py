from pathlib import Path
import unittest


ROOT = Path(__file__).parents[1]


class TendGallSourceContractTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.agent = (ROOT / "desk" / "app" / "tend.hoon").read_text(encoding="utf-8")
        cls.migration = (ROOT / "desk" / "lib" / "tend-migrate.hoon").read_text(encoding="utf-8")
        cls.migration_fixtures = (ROOT / "desk" / "lib" / "tend-migration-fixtures.hoon").read_text(encoding="utf-8")
        cls.migration_generator = (ROOT / "desk" / "gen" / "tend-migrations.hoon").read_text(encoding="utf-8")
        cls.migration_thread = (ROOT / "desk" / "ted" / "tend-migrations.hoon").read_text(encoding="utf-8")
        cls.json = (ROOT / "desk" / "lib" / "tend-json.hoon").read_text(encoding="utf-8")
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

    def test_assignment_is_checked_against_owner_member_or_pending_invitee(self):
        self.assertIn("(valid-assignee list-id.act assignee.act state)", self.agent)
        self.assertIn("=(u.assignee our.bowl)", self.agent)
        self.assertIn("(~(has by members.u.sharing) u.assignee)", self.agent)
        self.assertIn("(~(has by pending.u.sharing) u.assignee)", self.agent)

    def test_local_list_lifecycle_refreshes_access_state(self):
        create_branch = self.agent.split("%create-list", 1)[1].split("%rename-list", 1)[0]
        delete_branch = self.agent.split("%delete-list", 1)[1].split("%add-section", 1)[0]
        update = "(give [%accesses (accesses-for nex)])"
        self.assertIn(update, create_branch)
        self.assertIn(update, delete_branch)

    def test_new_lists_receive_a_varied_server_generated_appearance(self):
        create_branch = self.agent.split("%create-list", 1)[1].split("%rename-list", 1)[0]
        self.assertIn("(initial-appearance op-id.act now.bowl next-id.state)", create_branch)
        self.assertIn("++  initial-appearance", self.agent)
        self.assertIn("(mod (mug [operation now list-id]) 12)", self.agent)
        self.assertNotIn("'#3b82f6'\n            'list'", create_branch)
        for color in ("#3b82f6", "#ec4899", "#f97316", "#22c55e", "#06b6d4", "#a855f7"):
            self.assertIn(color, self.agent)

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

    def test_cross_list_move_is_a_durable_cross_host_gall_primitive(self):
        self.assertIn("%move-reminder-to-list", self.surface)
        self.assertIn("%move-reminder-to-list", self.json)
        self.assertIn("source-base-revision=@ud", self.surface)
        self.assertIn("destination-base-revision=@ud", self.surface)
        for peer_message in (
            "%transfer-request",
            "%transfer-payload",
            "%transfer-import",
            "%transfer-imported",
            "%transfer-finalize",
            "%transfer-finalized",
            "%transfer-restore",
            "%transfer-restored",
            "%transfer-rejected",
        ):
            self.assertIn(peer_message, self.surface)
        self.assertIn("++  start-cross-host-transfer", self.agent)
        self.assertIn("++  maintain-transfers", self.agent)
        self.assertIn("transfer-reservation-map", self.agent)
        self.assertIn("transfer-import-map", self.agent)
        self.assertIn("(retry-transfer-card-fact operation transfer)", self.agent)
        self.assertIn("(broadcast-list source-list-id.act", self.agent)
        self.assertIn("(broadcast-list destination-list-id.act", self.agent)
        self.assertIn("(peer-list-result sender source-list-id.act", self.agent)
        self.assertIn("(peer-list-result sender destination-list-id.act", self.agent)
        web = (ROOT / "desk" / "app" / "tend.html").read_text(encoding="utf-8")
        service = (ROOT / "omarchy-plugin" / "Service.qml").read_text(encoding="utf-8")
        self.assertIn('"move-reminder-to-list"', web)
        self.assertIn('"move-reminder-to-list"', service)
        self.assertNotIn("same host", web)
        self.assertNotIn("copyReminderToList", web)

    def test_web_navigation_remains_available_at_narrow_widths(self):
        web = (ROOT / "desk" / "app" / "tend.html").read_text(encoding="utf-8")
        self.assertIn('id="open-navigation"', web)
        self.assertIn("workspace.nav-open .nav", web)
        self.assertIn("mobileNavigationOpen", web)
        self.assertIn('aria-label="Open lists and views"', web)
        self.assertIn('setAttribute("aria-expanded"', web)

    def test_web_compact_controls_and_selection_mode_match_the_design(self):
        web = (ROOT / "desk" / "app" / "tend.html").read_text(encoding="utf-8")
        self.assertIn("--control-height: 44px", web)
        self.assertIn("height: var(--control-height); min-height: var(--control-height)", web)
        self.assertNotIn(".section-adder .field { width: 210px; min-height: 32px; }", web)
        self.assertIn('class="filter-options"', web)
        self.assertIn('class="compact-date"', web)
        self.assertIn('>+ add</button>', web)
        self.assertIn('class="tray-actions"', web)
        selection_markup = web.split('state.selectionMode && state.view === "list" && list ?', 1)[1].split('</div>` : ""}', 1)[0]
        self.assertNotIn('data-selection-action="complete"', selection_markup)
        self.assertNotIn('data-selection-action="open"', selection_markup)
        self.assertNotIn('data-selection-action="delete"', selection_markup)

    def test_web_owns_select_and_datetime_popovers_inside_the_viewport(self):
        web = (ROOT / "desk" / "app" / "tend.html").read_text(encoding="utf-8")
        self.assertIn('class="control-popover-layer"', web)
        self.assertIn("function openSelectPopover", web)
        self.assertIn("function openDatePopover", web)
        self.assertIn("function placeControlPopover", web)
        self.assertIn("window.innerWidth - margin * 2", web)
        self.assertIn("window.innerHeight - margin * 2", web)
        self.assertIn('panel.className = "control-popover date-picker"', web)
        self.assertIn('event.target.closest("#control-popover-layer")', web)
        self.assertIn("!state.controlPopover", web)

    def test_web_detail_exposes_relationships_recurrence_and_local_due_time(self):
        web = (ROOT / "desk" / "app" / "tend.html").read_text(encoding="utf-8")
        self.assertIn("function reminderAncestors", web)
        self.assertIn("function reminderDescendants", web)
        self.assertIn('class="relationship-group"', web)
        self.assertIn("function recurrencePayload", web)
        self.assertIn("function recurrenceAdvance", web)
        self.assertIn('data-repeat-weekday="${index}"', web)
        self.assertIn('data-field="repeatMonthMode"', web)
        self.assertIn("timezone:draft.timezone", web)
        self.assertIn('data-field="repeatFrequency"', web)
        self.assertIn('data-field="repeatEndMode"', web)
        self.assertIn("new Date(Date.UTC(+urbit[1]", web)
        self.assertIn("date.getTime() - date.getTimezoneOffset() * 60000", web)
        self.assertIn("@media (max-width: 520px)", web)
        self.assertIn(".tray { width: 100vw; max-width: none; }", web)

    def test_batch_completion_records_each_scheduled_occurrence(self):
        self.assertIn("%batch-set-completed -.act", self.agent)
        self.assertIn("(scot %uv (sham [op-id.act id]))", self.agent)
        self.assertIn("?:(completed.act (activity-occurrence-for u.target id before) ~)", self.agent)

    def test_recurrence_completion_uses_bounded_timezone_aware_client_hints(self):
        self.assertIn("+$  recurrence-advance", self.surface)
        self.assertIn("advance=(unit recurrence-advance)", self.surface)
        self.assertIn("advances=(list reminder-advance)", self.surface)
        self.assertIn("(advance-schedule sch now.bowl advance.act)", self.agent)
        self.assertIn("(sub occurrence.u.hint occurrence.sch) 100.000", self.agent)
        self.assertIn("(gth due-at.u.advanced now)", self.agent)
        self.assertIn("%invalid-schedule", self.agent)
        service = (ROOT / "omarchy-plugin" / "Service.qml").read_text(encoding="utf-8")
        self.assertIn('"_schedule": reminder ? reminder.schedule : null', service)
        self.assertIn('"_schedules": schedules', service)

    def test_state_ten_migrates_private_presentation_and_notification_defaults(self):
        self.assertIn("(migrate:tend-migrate now.bowl old-state)", self.agent)
        self.assertIn("++  migrate", self.migration)
        self.assertIn("++  upgrade-9", self.migration)
        self.assertIn("+$  saved-state", self.surface)
        self.assertIn("*list-presentations:t", self.migration)
        self.assertIn("*collaboration-policies:t", self.migration)
        self.assertIn("*collaboration-notifications:t", self.migration)

    def test_every_saved_schema_has_a_nonempty_executable_fixture(self):
        for version in range(12):
            self.assertIn(f"[{version} [%{version} ", self.migration_fixtures)
        self.assertIn("'fixture-list'", self.migration_fixtures)
        self.assertIn("(~(has by reminders.u.migrated-list) 7)", self.migration_fixtures)
        self.assertIn("Tend saved-state migrations %0 through %11 passed", self.migration_generator)
        self.assertIn("[%tend-migrations %11 %.y]", self.migration_thread)

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

    def test_snapshot_publishes_the_desktop_protocol_version(self):
        self.assertIn("[%protocol-version (numb 2)]", self.json)

    def test_agent_serves_the_authenticated_landscape_application(self):
        docket = (ROOT / "desk" / "desk.docket-0").read_text(encoding="utf-8")
        web_ui = (ROOT / "desk" / "app" / "tend.html").read_text(encoding="utf-8")
        self.assertIn("%connect `/apps/tend %tend", self.agent)
        self.assertIn("[%eyre %bound *]", self.agent)
        self.assertIn("%handle-http-request", self.agent)
        self.assertIn("require-authorization:app:server", self.agent)
        self.assertIn("site+/apps/tend", docket)
        self.assertIn("/~/scry/tend/state.json", web_ui)
        self.assertIn("/~/channel/", web_ui)
        self.assertIn("const hostWritable = list ? canEdit(list.id) : false;", web_ui)
        self.assertIn("!hostWritable && list && !owner", web_ui)
        self.assertIn('class=\"tray detail-tray\"', web_ui)
        self.assertIn('class=\"tray settings-tray\"', web_ui)
        self.assertIn('type=\"datetime-local\"', web_ui)
        self.assertIn("scheduleDetailSave", web_ui)
        self.assertIn('const list = state.view === "list" ? selectedList() : null;', web_ui)
        self.assertIn('state.selectionMode && state.view === "list" && list', web_ui)
        self.assertIn("const selectedReminder = () => {\n        const list = selectedList();", web_ui)
        self.assertNotIn(".task-actions { display: none; }", web_ui)

    def test_live_plugin_updates_invalidate_qml_cache_and_restart_the_shell(self):
        installer = (ROOT / "packaging" / "tend" / "install.sh").read_text(encoding="utf-8")
        enable_guard = 'if [[ "$enable_plugin" == true ]]'

        self.assertIn('default_plugin_dir="${XDG_CONFIG_HOME:-$HOME/.config}/omarchy/plugins/io.omabit.tend"', installer)
        self.assertIn('if [[ "$plugin_dir" == "$default_plugin_dir" ]]', installer)
        self.assertIn('find "$plugin_dir" -type f -exec touch -- {} +', installer)
        self.assertIn('omarchy-shell -q shell listPlugins', installer)
        self.assertIn('omarchy restart shell', installer)
        self.assertIn('omarchy-shell -q shell rescanPlugins', installer)
        self.assertGreater(installer.index('omarchy restart shell'), installer.index(enable_guard))


if __name__ == "__main__":
    unittest.main()
