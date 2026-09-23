from pathlib import Path
import unittest


ROOT = Path(__file__).parents[1]


class TendAccessibilityTests(unittest.TestCase):
    def test_panel_keeps_keyboard_navigation_contract(self):
        panel = (ROOT / "omarchy-plugin" / "TendPanel.qml").read_text(encoding="utf-8")
        for shortcut in ("Ctrl+N", "Ctrl+F", "Ctrl+Shift+N", "Ctrl+,", "Alt+1", "Alt+6", "Ctrl+[", "Ctrl+]"):
            self.assertIn(f'sequence: "{shortcut}"', panel)
        self.assertIn("Keys.onReturnPressed", panel)
        self.assertIn("Keys.onSpacePressed", panel)
        self.assertIn("Accessible.role: Accessible.List", panel)
        self.assertIn('Accessible.name: "Current Urbit plus code"', panel)
        self.assertIn("assigneeChoices: root.assigneeChoices", panel)
        self.assertIn("access.pending", panel)
        self.assertIn('target === "new-list"', panel)
        self.assertIn('target === "invite"', panel)
        self.assertIn('target === "reminder-assignee"', panel)
        self.assertIn('target === "reminder-save"', panel)
        self.assertIn("onAccepted: inviteButton.clicked()", panel)
        self.assertIn("root.setReminderCollapsed", panel)

    def test_bar_exposes_count_and_connection_state(self):
        widget = (ROOT / "omarchy-plugin" / "BarWidget.qml").read_text(encoding="utf-8")
        self.assertIn("Accessible.role: Accessible.Button", widget)
        self.assertIn("taskCount", widget)
        self.assertIn("state", widget)
        self.assertIn("KeyboardPanel", widget)
        self.assertIn('property string selectedView: "assigned"', widget)
        self.assertIn("view: selectedView", widget)
        self.assertIn('sort: "priority"', widget)
        self.assertIn("openFullPanel", widget)
        self.assertIn("addReminderWithDetails", widget)
        self.assertIn("smartViewKeys", widget)
        self.assertIn("quickDue", widget)
        self.assertIn("assigneePicker", widget)

    def test_plugin_adds_no_custom_motion(self):
        plugin_text = "\n".join(path.read_text(encoding="utf-8") for path in (ROOT / "omarchy-plugin").glob("*.qml"))
        for animation in ("NumberAnimation", "PropertyAnimation", "SmoothedAnimation", "SpringAnimation"):
            self.assertNotIn(animation, plugin_text)

    def test_panel_is_a_normal_window_and_bar_popup_opens_it(self):
        panel = (ROOT / "omarchy-plugin" / "TendPanel.qml").read_text(encoding="utf-8")
        widget = (ROOT / "omarchy-plugin" / "BarWidget.qml").read_text(encoding="utf-8")
        manifest = (ROOT / "omarchy-plugin" / "manifest.json").read_text(encoding="utf-8")
        self.assertIn("FloatingWindow", panel)
        self.assertNotIn("WlrLayershell", panel)
        self.assertIn('"panel": "TendPanel.qml"', manifest)
        self.assertNotIn('"overlay"', manifest)
        self.assertIn("bar.shell.summon", widget)
        self.assertIn("columns: width < Style.space(520) ? 1 : 3", panel)

    def test_panel_uses_omarchy_controls_and_right_side_details(self):
        panel = (ROOT / "omarchy-plugin" / "TendPanel.qml").read_text(encoding="utf-8")
        details = (ROOT / "omarchy-plugin" / "ReminderDetail.qml").read_text(encoding="utf-8")
        self.assertIn("TendButton", panel)
        self.assertIn("TendDropdown", panel)
        self.assertIn('text: "VIEWS"', panel)
        self.assertIn('text: "Add list"', panel)
        self.assertIn('text: "Add reminder"', panel)
        self.assertIn("visible: root.selectionMode", panel)
        self.assertIn("anchors.right: parent.right", panel)
        self.assertIn('text: "REMINDER DETAILS"', details)
        self.assertIn("SearchableDropdown", details)
        self.assertIn("MultiSelect", details)
        self.assertIn("TendTextArea", details)
        self.assertIn("Color.popups.background", details)

    def test_recipient_bound_invitation_link_is_selectable(self):
        panel = (ROOT / "omarchy-plugin" / "TendPanel.qml").read_text(encoding="utf-8")
        service = (ROOT / "omarchy-plugin" / "Service.qml").read_text(encoding="utf-8")
        self.assertIn('root.lastInvitationUri = "omabit://tend/invite/"', service)
        self.assertIn('Accessible.name: "Copy recipient-bound Tend invitation link"', panel)
        self.assertIn("selectByMouse: true", panel)

    def test_shared_list_status_exposes_in_flight_edits_as_text(self):
        panel = (ROOT / "omarchy-plugin" / "TendPanel.qml").read_text(encoding="utf-8")
        self.assertIn("service.listMutationPending(selectedList.id)", panel)
        self.assertIn("PREVIOUS EDIT STILL IN FLIGHT", panel)
        self.assertIn("Accessible.name: text", panel)

    def test_login_keeps_the_code_visible_until_authentication_succeeds(self):
        panel = (ROOT / "omarchy-plugin" / "TendPanel.qml").read_text(encoding="utf-8")
        service = (ROOT / "omarchy-plugin" / "Service.qml").read_text(encoding="utf-8")
        self.assertIn("signal loginSucceeded()", service)
        self.assertIn("loginProcess.output = \"\"", service)
        self.assertIn("loginProcess.errors = \"\"", service)
        self.assertIn("root.loginSucceeded()", service)
        self.assertIn('function onLoginSucceeded() { loginCode.text = ""; }', panel)
        self.assertNotIn('service.login(shipUrl.text, loginCode.text);\n                                loginCode.text = "";', panel)

    def test_transport_path_is_component_relative(self):
        panel = (ROOT / "omarchy-plugin" / "TendPanel.qml").read_text(encoding="utf-8")
        service = (ROOT / "omarchy-plugin" / "Service.qml").read_text(encoding="utf-8")
        self.assertIn('Qt.resolvedUrl("transport/eyre_client.py")', service)
        self.assertNotIn("manifest.__sourceDir", service)
        self.assertIn('root.errorMessage = "The bundled Tend transport could not be located"', service)
        self.assertIn("function diagnostics(unused)", panel)
        self.assertIn('bridgeReady: service ? service.bridgePath !== "" : false', panel)

    def test_transport_processes_do_not_mutate_the_watched_plugin_tree(self):
        service = (ROOT / "omarchy-plugin" / "Service.qml").read_text(encoding="utf-8")
        self.assertEqual(service.count('["python3", "-B", bridgePath,'), 6)
        self.assertNotIn('["python3", bridgePath,', service)

    def test_quick_add_normalizes_self_assignment_and_supports_details(self):
        service = (ROOT / "omarchy-plugin" / "Service.qml").read_text(encoding="utf-8")
        self.assertIn("function addReminderAssignedToMe", service)
        self.assertIn("function addReminderWithDetails", service)
        self.assertIn("function queueDetailsForCreatedReminder", service)
        self.assertIn("normalizeShip(fields.assignee)", service)
        self.assertIn('kind: "schedule"', service)

    def test_completion_uses_checkbox_and_ten_second_grace_period(self):
        service = (ROOT / "omarchy-plugin" / "Service.qml").read_text(encoding="utf-8")
        panel = (ROOT / "omarchy-plugin" / "TendPanel.qml").read_text(encoding="utf-8")
        widget = (ROOT / "omarchy-plugin" / "BarWidget.qml").read_text(encoding="utf-8")
        self.assertIn("function toggleCompletedWithGrace", service)
        self.assertIn("elapsed <= 5000", service)
        self.assertIn("< 10000", service)
        self.assertIn("TendCheckbox", panel)
        self.assertIn("TendCheckbox", widget)

    def test_panel_uses_resizable_sidebar_and_opaque_inspectors(self):
        panel = (ROOT / "omarchy-plugin" / "TendPanel.qml").read_text(encoding="utf-8")
        defaults = (ROOT / "omarchy-plugin" / "ReminderDefaults.qml").read_text(encoding="utf-8")
        self.assertIn("sidebarResizeHandle", panel)
        self.assertIn("Text.ElideRight", panel)
        self.assertIn("ReminderDefaults", panel)
        self.assertIn("Color.popups.background", defaults)

    def test_mutation_processes_use_newline_delimited_json(self):
        service = (ROOT / "omarchy-plugin" / "Service.qml").read_text(encoding="utf-8")
        bridge = (ROOT / "omarchy-plugin" / "transport" / "eyre_client.py").read_text(encoding="utf-8")
        self.assertEqual(service.count('write(payload + "\\n")'), 2)
        self.assertIn("json.loads(sys.stdin.readline())", bridge)
        self.assertNotIn("json.load(sys.stdin)", bridge)


if __name__ == "__main__":
    unittest.main()
