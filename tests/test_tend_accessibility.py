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


if __name__ == "__main__":
    unittest.main()
