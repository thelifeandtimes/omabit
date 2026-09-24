from pathlib import Path
import re
import unittest


ROOT = Path(__file__).parents[1]


class TendAccessibilityTests(unittest.TestCase):
    def test_panel_does_not_advertise_unimplemented_keyboard_shortcuts(self):
        panel = (ROOT / "omarchy-plugin" / "TendPanel.qml").read_text(encoding="utf-8")
        for shortcut in ("Ctrl+N", "Ctrl+F", "Ctrl+Shift+N", "Ctrl+,", "Alt+1", "Alt+6", "Ctrl+[", "Ctrl+]"):
            self.assertNotIn(f'sequence: "{shortcut}"', panel)
        self.assertNotIn("Keys.onReturnPressed", panel)
        self.assertNotIn("Keys.onSpacePressed", panel)
        self.assertNotIn("Keys.on", panel)
        self.assertIn("Accessible.role: Accessible.List", panel)
        self.assertIn('Accessible.name: "Current Urbit plus code"', panel)
        self.assertIn("assigneeChoices: root.assigneeChoices", panel)
        self.assertIn("access.pending", panel)
        self.assertIn('target === "new-list"', panel)
        self.assertIn('target === "invite"', panel)
        self.assertIn('target === "reminder-assignee"', panel)
        self.assertIn('target === "reminder-save"', panel)
        self.assertIn("onAccepted: inviteButton.clicked()", panel)
        self.assertIn("root.toggleReminderCollapsed", panel)

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
        self.assertIn("id: openTendButton", widget)
        self.assertIn('Accessible.name: "Open Tend"', widget)
        self.assertIn("TendOpenGlyph", widget)
        self.assertNotIn('text: "↗"', widget)
        open_button = widget.split("id: openTendButton", 1)[1].split("onClicked:", 1)[0]
        self.assertIn("Layout.preferredWidth: Style.spacing.controlHeight", open_button)
        self.assertIn("Layout.preferredHeight: Style.spacing.controlHeight", open_button)
        self.assertIn("Layout.alignment: Qt.AlignVCenter", open_button)
        self.assertIn("addReminderWithDetails", widget)
        self.assertIn("smartViewKeys", widget)
        self.assertIn("PanelHero", widget)
        self.assertIn('title: "Tend"', widget)
        self.assertIn("meta: root.headerMeta", widget)
        self.assertIn('" · " + taskCount + " · " + selectedViewLabel', widget)
        self.assertNotIn("detail: String(root.taskCount)", widget)
        self.assertIn("quickDue", widget)
        self.assertIn("assigneePicker", widget)
        self.assertIn("TendModel.menubarRows", widget)
        self.assertIn("filtered subitem", widget)
        self.assertIn("id: filteredSummaryButton", widget)
        self.assertIn("id: reminderCard", widget)
        self.assertIn("x: reminderRowDelegate.indentPixels", widget)
        self.assertNotIn("anchors.leftMargin: Style.space(7) + reminderRowDelegate.indentPixels", widget)
        self.assertIn("Math.floor(popup.screenH * 0.5)", widget)

    def test_picker_popups_are_clamped_to_the_window_overlay(self):
        date_picker = (ROOT / "omarchy-plugin" / "TendDateTimePicker.qml").read_text(encoding="utf-8")
        assignee_picker = (ROOT / "omarchy-plugin" / "TendAssigneePicker.qml").read_text(encoding="utf-8")
        self.assertIn("parent: trigger.Window.window ? trigger.Window.window.contentItem : trigger", date_picker)
        self.assertIn("parent: QQC.Overlay.overlay", assignee_picker)
        for picker in (date_picker, assignee_picker):
            self.assertIn("function placePopup()", picker)
            self.assertIn("parent.width", picker)
            self.assertIn("parent.height", picker)

    def test_date_picker_owns_the_closing_click_and_centers_its_icon(self):
        picker = (ROOT / "omarchy-plugin" / "TendDateTimePicker.qml").read_text(encoding="utf-8")
        self.assertIn("modal: true", picker)
        self.assertIn("dim: false", picker)
        self.assertIn("onClicked: root.open()", picker)
        self.assertIn("OpticalGlyph", picker)
        self.assertIn("anchors.centerIn: parent", picker)
        self.assertNotIn("suppressTriggerRelease", picker)

    def test_dropdowns_own_the_closing_click_and_stay_inside_the_window(self):
        dropdown = (ROOT / "omarchy-plugin" / "TendDropdown.qml").read_text(encoding="utf-8")
        self.assertIn("parent: trigger.Window.window ? trigger.Window.window.contentItem : trigger", dropdown)
        self.assertIn("modal: true", dropdown)
        self.assertIn("dim: false", dropdown)
        self.assertIn("function placePopup()", dropdown)
        self.assertIn("parent.width", dropdown)
        self.assertIn("parent.height", dropdown)

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

    def test_panel_uses_omarchy_controls_and_responsive_trays(self):
        panel = (ROOT / "omarchy-plugin" / "TendPanel.qml").read_text(encoding="utf-8")
        details = (ROOT / "omarchy-plugin" / "ReminderDetail.qml").read_text(encoding="utf-8")
        settings = (ROOT / "omarchy-plugin" / "TendSettings.qml").read_text(encoding="utf-8")
        self.assertIn("TendButton", panel)
        self.assertIn("TendDropdown", panel)
        self.assertIn('text: "VIEWS"', panel)
        self.assertIn('text: "Add list"', panel)
        self.assertIn('text: "ADD NEW REMINDER"', panel)
        self.assertIn("visible: root.selectionMode", panel)
        self.assertIn("anchors.left: parent.left", panel)
        self.assertIn("anchors.right: parent.right", panel)
        self.assertIn("singleTrayMode", panel)
        self.assertIn("overlayTrayMode", panel)
        self.assertIn("TendModel.sectionedRows", panel)
        self.assertIn("displayedRows", panel)
        self.assertIn('text: "REMINDER DETAILS"', details)
        self.assertIn("TendAssigneePicker", details)
        self.assertIn("TendDateTimePicker", details)
        self.assertIn("MultiSelect", details)
        self.assertIn("TendTextArea", details)
        self.assertIn("Color.popups.background", details)
        self.assertIn('mode === "app"', settings)
        self.assertIn('mode === "list"', settings)

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

    def test_panel_uses_resizable_navigation_and_scrolling_inspectors(self):
        panel = (ROOT / "omarchy-plugin" / "TendPanel.qml").read_text(encoding="utf-8")
        details = (ROOT / "omarchy-plugin" / "ReminderDetail.qml").read_text(encoding="utf-8")
        settings = (ROOT / "omarchy-plugin" / "TendSettings.qml").read_text(encoding="utf-8")
        self.assertIn("sidebarResizeHandle", panel)
        self.assertIn("Text.ElideRight", panel)
        self.assertIn("ScrollView", details)
        self.assertIn("ScrollView", settings)
        self.assertIn("Color.popups.background", details)

    def test_reminder_detail_exposes_relationships_and_recurrence(self):
        details = (ROOT / "omarchy-plugin" / "ReminderDetail.qml").read_text(encoding="utf-8")
        panel = (ROOT / "omarchy-plugin" / "TendPanel.qml").read_text(encoding="utf-8")
        self.assertIn('text: "ANCESTORS"', details)
        self.assertIn('text: "DESCENDANTS"', details)
        self.assertIn("signal reminderRequested", details)
        self.assertIn("onReminderRequested", panel)
        self.assertIn('model: ["none", "hourly", "daily", "weekly", "monthly", "yearly"]', details)
        self.assertIn("recurrence: recurrenceValue()", details)
        self.assertIn("iconOnly: true", details)
        self.assertIn("TendFlagButton {\n          id: flaggedField", details)

    def test_panel_uses_centered_icon_controls_and_full_width_narrow_surfaces(self):
        panel = (ROOT / "omarchy-plugin" / "TendPanel.qml").read_text(encoding="utf-8")
        settings = (ROOT / "omarchy-plugin" / "TendSettings.qml").read_text(encoding="utf-8")
        icon_button = (ROOT / "omarchy-plugin" / "TendIconButton.qml").read_text(encoding="utf-8")
        self.assertIn("OpticalGlyph", icon_button)
        self.assertIn("anchors.centerIn: parent", icon_button)
        self.assertIn("fullSurfaceMode", panel)
        self.assertIn("root.fullSurfaceMode ? parent.width", panel)
        self.assertIn('text: "+ add"', panel)
        self.assertIn("iconOnly: true", panel)
        self.assertIn('glyph: root.pinned() ? "󰐃" : "󰐄"', settings)
        self.assertIn('text: "CONNECTED SHIP"', settings)
        self.assertNotIn('text: root.pinned() ? "Unpin list" : "Pin list"', settings)

    def test_panel_header_actions_are_unboxed_and_form_rows_share_one_height(self):
        panel = (ROOT / "omarchy-plugin" / "TendPanel.qml").read_text(encoding="utf-8")
        self.assertIn("readonly property real formControlHeight: Style.spacing.controlHeight", panel)
        self.assertIn("height: root.formControlHeight", panel)
        for control_id in ("settingsButton", "closeButton", "editListButton"):
            block = panel.split(f"id: {control_id}", 1)[1].split("onClicked:", 1)[0]
            self.assertIn("bordered: false", block)
        title_row = panel.split("id: viewTitleText", 1)[1].split("id: selectedHostDot", 1)[0]
        self.assertLess(title_row.index("text: root.viewTitle"), title_row.index("id: editListButton"))
        for control_id in ("quickAdd", "quickAssignee", "quickDue", "quickAddButton"):
            block = panel.split(f"id: {control_id}", 1)[1].split("}", 1)[0]
            self.assertIn("root.formControlHeight", block)

    def test_panel_uses_hero_headers_and_subtle_list_host_status(self):
        panel = (ROOT / "omarchy-plugin" / "TendPanel.qml").read_text(encoding="utf-8")
        app_header = panel.split("id: appHeader", 1)[1].split("PanelSeparator", 1)[0]
        list_header = panel.split("id: viewHeader", 1)[1].split("PanelSeparator", 1)[0]

        self.assertIn("PanelHero", app_header)
        self.assertIn('title: "Tend"', app_header)
        self.assertIn("meta: service && service.ship ? root.normalizedShip(service.ship)", app_header)
        self.assertIn("id: selectedListAvatar", list_header)
        self.assertIn("text: root.selectedList ? root.listHost(root.selectedList.id)", list_header)
        self.assertIn("id: selectedHostDot", list_header)
        self.assertIn("width: Style.space(5)", list_header)
        self.assertIn('Accessible.name: root.selectedConnectionAvailable ? "List host online"', list_header)

    def test_native_single_line_fields_share_the_tend_control_height(self):
        plugin = ROOT / "omarchy-plugin"
        field = (plugin / "TendTextField.qml").read_text(encoding="utf-8")
        button = (plugin / "TendButton.qml").read_text(encoding="utf-8")
        self.assertIn("implicitHeight: Style.spacing.controlHeight", field)
        self.assertIn("verticalPadding: Style.spacing.controlPaddingY", field)
        self.assertIn("implicitHeight: Style.spacing.controlHeight", button)
        for path in plugin.glob("*.qml"):
            if path.name == "TendTextField.qml":
                continue
            self.assertIsNone(
                re.search(r"(?m)^\s*TextField\s*\{", path.read_text(encoding="utf-8")),
                path.name,
            )

    def test_sidebar_new_list_controls_have_explicit_shared_heights(self):
        panel = (ROOT / "omarchy-plugin" / "TendPanel.qml").read_text(encoding="utf-8")
        new_list_block = panel.split("id: newList", 1)[1].split("onAccepted: addListButton.clicked()", 1)[0]
        add_list_block = panel.split("id: addListButton", 1)[1].split("onClicked:", 1)[0]

        self.assertIn("height: root.formControlHeight", new_list_block)
        self.assertIn("height: root.formControlHeight", add_list_block)

    def test_sidebar_reveal_controls_reserve_visible_space(self):
        panel = (ROOT / "omarchy-plugin" / "TendPanel.qml").read_text(encoding="utf-8")

        for control_id, noun in (("viewsReveal", "views"), ("listsReveal", "lists")):
            block = panel.split(f"id: {control_id}", 1)[1].split("onClicked:", 1)[0]
            self.assertIn("width: Math.max(Style.space(70), implicitWidth)", block)
            self.assertIn("height: parent.height", block)
            self.assertIn('text: root.unpinned', block)
            self.assertIn(f'"Show unpinned {noun}"', block)
            self.assertIn("Accessible.name: tooltipText", block)

    def test_selection_mode_is_for_organization_not_completion_or_deletion(self):
        panel = (ROOT / "omarchy-plugin" / "TendPanel.qml").read_text(encoding="utf-8")
        selection = panel.split('visible: root.selectionMode && root.viewMode === "list"', 1)[1].split("Rectangle {", 1)[0]
        self.assertIn('text: "Indent"', selection)
        self.assertIn('text: "Outdent"', selection)
        self.assertNotIn('text: "Complete"', selection)
        self.assertNotIn('text: "Uncomplete"', selection)
        self.assertNotIn('Confirm delete', selection)

    def test_panel_tree_indents_the_entire_reminder_card_from_the_left(self):
        panel = (ROOT / "omarchy-plugin" / "TendPanel.qml").read_text(encoding="utf-8")
        self.assertIn("delegate: Item {\n                                    id: reminderRow", panel)
        self.assertIn("width: ListView.view.width", panel)
        self.assertIn("x: reminderRow.indentPixels", panel)
        self.assertIn("width: Math.max(Style.space(120), reminderRow.width - x)", panel)
        self.assertNotIn("\n                                    x: indentPixels", panel)

    def test_menubar_filtered_subitems_are_subtle_and_quick_add_is_labeled(self):
        widget = (ROOT / "omarchy-plugin" / "BarWidget.qml").read_text(encoding="utf-8")
        self.assertIn('text: "+ add"', widget)
        self.assertIn("font.pixelSize: Style.font.caption", widget)
        self.assertIn("font.underline: filteredSummaryHover.hovered", widget)
        self.assertNotIn("TendButton {\n                id: filteredSummaryButton", widget)

    def test_mutation_processes_use_newline_delimited_json(self):
        service = (ROOT / "omarchy-plugin" / "Service.qml").read_text(encoding="utf-8")
        bridge = (ROOT / "omarchy-plugin" / "transport" / "eyre_client.py").read_text(encoding="utf-8")
        self.assertEqual(service.count('write(payload + "\\n")'), 2)
        self.assertIn("json.loads(sys.stdin.readline())", bridge)
        self.assertNotIn("json.load(sys.stdin)", bridge)


if __name__ == "__main__":
    unittest.main()
