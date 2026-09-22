from pathlib import Path
import unittest


ROOT = Path(__file__).parents[1]


class TendAccessibilityTests(unittest.TestCase):
    def test_overlay_keeps_keyboard_navigation_contract(self):
        overlay = (ROOT / "omarchy-plugin" / "Overlay.qml").read_text(encoding="utf-8")
        for shortcut in ("Ctrl+N", "Ctrl+F", "Ctrl+Shift+N", "Ctrl+Comma", "Alt+1", "Alt+6", "Ctrl+[", "Ctrl+]"):
            self.assertIn(f'sequence: "{shortcut}"', overlay)
        self.assertIn("Keys.onReturnPressed", overlay)
        self.assertIn("Keys.onSpacePressed", overlay)
        self.assertIn("Accessible.role: Accessible.List", overlay)
        self.assertIn('Accessible.name: "Current Urbit plus code"', overlay)
        self.assertIn("model: root.assigneeChoices", overlay)
        self.assertIn("selectedAccess.pending", overlay)
        self.assertIn('target === "new-list"', overlay)
        self.assertIn('target === "invite"', overlay)
        self.assertIn('target === "reminder-assignee"', overlay)
        self.assertIn('target === "reminder-save"', overlay)
        self.assertIn("onAccepted: inviteButton.clicked()", overlay)
        self.assertIn("root.setReminderCollapsed", overlay)

    def test_bar_exposes_count_and_connection_state(self):
        widget = (ROOT / "omarchy-plugin" / "BarWidget.qml").read_text(encoding="utf-8")
        self.assertIn("Accessible.role: Accessible.Button", widget)
        self.assertIn("taskCount", widget)
        self.assertIn("state", widget)

    def test_plugin_adds_no_custom_motion(self):
        plugin_text = "\n".join(path.read_text(encoding="utf-8") for path in (ROOT / "omarchy-plugin").glob("*.qml"))
        for animation in ("NumberAnimation", "PropertyAnimation", "SmoothedAnimation", "SpringAnimation"):
            self.assertNotIn(animation, plugin_text)

    def test_overlay_targets_the_invoking_or_focused_monitor_and_scales(self):
        overlay = (ROOT / "omarchy-plugin" / "Overlay.qml").read_text(encoding="utf-8")
        widget = (ROOT / "omarchy-plugin" / "BarWidget.qml").read_text(encoding="utf-8")
        self.assertIn("Hyprland.focusedMonitor", overlay)
        self.assertIn("window.screen = chosen", overlay)
        self.assertIn("root.chooseOpenScreen(payload.screen)", overlay)
        self.assertIn("root.QsWindow.window", widget)
        self.assertIn('JSON.stringify({ screen: screenName })', widget)
        self.assertIn("Math.min(Style.space(640)", overlay)
        self.assertIn("columns: width < Style.space(520) ? 1 : 3", overlay)

    def test_recipient_bound_invitation_link_is_selectable(self):
        overlay = (ROOT / "omarchy-plugin" / "Overlay.qml").read_text(encoding="utf-8")
        service = (ROOT / "omarchy-plugin" / "Service.qml").read_text(encoding="utf-8")
        self.assertIn('root.lastInvitationUri = "omabit://tend/invite/"', service)
        self.assertIn('Accessible.name: "Copy recipient-bound Tend invitation link"', overlay)
        self.assertIn("selectByMouse: true", overlay)

    def test_shared_list_status_exposes_in_flight_edits_as_text(self):
        overlay = (ROOT / "omarchy-plugin" / "Overlay.qml").read_text(encoding="utf-8")
        self.assertIn("service.listMutationPending(selectedList.id)", overlay)
        self.assertIn("PREVIOUS EDIT STILL IN FLIGHT", overlay)
        self.assertIn("Accessible.name: text", overlay)

    def test_login_keeps_the_code_visible_until_authentication_succeeds(self):
        overlay = (ROOT / "omarchy-plugin" / "Overlay.qml").read_text(encoding="utf-8")
        service = (ROOT / "omarchy-plugin" / "Service.qml").read_text(encoding="utf-8")
        self.assertIn("signal loginSucceeded()", service)
        self.assertIn("loginProcess.output = \"\"", service)
        self.assertIn("loginProcess.errors = \"\"", service)
        self.assertIn("root.loginSucceeded()", service)
        self.assertIn('function onLoginSucceeded() { loginCode.text = ""; }', overlay)
        self.assertNotIn('service.login(shipUrl.text, loginCode.text);\n                                loginCode.text = "";', overlay)

    def test_transport_path_is_component_relative(self):
        overlay = (ROOT / "omarchy-plugin" / "Overlay.qml").read_text(encoding="utf-8")
        service = (ROOT / "omarchy-plugin" / "Service.qml").read_text(encoding="utf-8")
        self.assertIn('Qt.resolvedUrl("transport/eyre_client.py")', service)
        self.assertNotIn("manifest.__sourceDir", service)
        self.assertIn('root.errorMessage = "The bundled Tend transport could not be located"', service)
        self.assertIn("function diagnostics(unused)", overlay)
        self.assertIn('bridgeReady: service ? service.bridgePath !== "" : false', overlay)


if __name__ == "__main__":
    unittest.main()
