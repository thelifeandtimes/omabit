import QtQuick
import QtQuick.Controls
import Quickshell
import Quickshell.Hyprland
import Quickshell.Wayland
import qs.Commons
import "TendModel.js" as TendModel

Item {
    id: root

    property var shell: null
    property var manifest: null
    property var service: null
    property bool opened: false
    property int selectedListId: 0
    property int selectedReminderId: 0
    property int selectedSectionId: 0
    property string viewMode: "list"
    property string sortMode: "manual"
    property string tagFilter: ""
    property bool sortDescending: false
    property bool listEditorOpen: false
    property bool policyEditorOpen: false
    property bool confirmDeleteList: false
    property bool captureMode: false
    property var selectedReminderIds: []
    property var collapsedReminderIds: []
    property bool confirmBatchDelete: false
    readonly property var selectedList: {
        var available = service ? service.lists : [];
        for (var i = 0; i < available.length; i++) {
            if (available[i].id === selectedListId)
                return available[i];

        }
        return available.length ? available[0] : null;
    }
    readonly property var selectedReminder: {
        var reminders = selectedList ? selectedList.reminders : [];
        for (var i = 0; i < reminders.length; i++) {
            if (reminders[i].id === selectedReminderId)
                return reminders[i];

        }
        return null;
    }
    readonly property var selectedAccess: service && selectedList ? service.accessForList(selectedList.id) : null
    readonly property bool selectedListEditable: service && selectedList ? service.canEditList(selectedList.id) : false
    readonly property bool selectedMayInvite: {
        if (!selectedAccess || !service)
            return false;
        if (selectedAccess.owner)
            return true;
        var mine = String(service.ship || "").replace(/^~/, "");
        var members = selectedAccess.members || [];
        for (var i = 0; i < members.length; i++) {
            var member = members[i];
            if (String(member.ship || "").replace(/^~/, "") === mine)
                return member.policy && member.policy.canInvite === true;
        }
        return false;
    }
    readonly property string selectedHostStatus: {
        if (!selectedAccess)
            return "CHECKING OWNER";
        if (selectedAccess.owner)
            return "OWNED HERE · ONLINE";
        return "HOSTED BY " + selectedAccess.host.toUpperCase() + " · " + selectedAccess.status.toUpperCase();
    }
    readonly property var selectedActivities: TendModel.activitiesForList(service ? service.activities : [], selectedList ? selectedList.id : 0)
    readonly property var selectedPresentation: TendModel.presentationForList(service ? service.localSettings : null, selectedList ? selectedList.id : 0)
    readonly property var selectedCollaborationPolicy: TendModel.collaborationPolicyForList(service ? service.localSettings : null, selectedList ? selectedList.id : 0)
    readonly property var orderedLists: TendModel.orderedLists(service ? service.lists : [], service ? service.preferences.pinnedLists : [], service ? service.localSettings.listOrder : [])
    readonly property var availableViews: ["today", "scheduled", "all", "flagged", "assigned", "completed"]
    readonly property var orderedViews: TendModel.orderedValues(availableViews, service ? service.preferences.pinnedViews : [])
    readonly property var availableTags: TendModel.allTags(service ? service.lists : [])
    readonly property var addDestination: {
        if (!service || !service.lists.length)
            return null;

        var wanted = viewMode === "list" && selectedList ? selectedList.id : service.preferences.defaultList;
        for (var i = 0; i < service.lists.length; i++) {
            if (service.lists[i].id === wanted)
                return service.lists[i];

        }
        return service.lists[0];
    }
    readonly property var queriedReminders: TendModel.queryReminders(service ? service.lists : [], {
        view: viewMode,
        listId: selectedList ? selectedList.id : 0,
        search: reminderSearch.text,
        tag: tagFilter,
        sort: sortMode,
        descending: sortDescending,
        ship: service ? service.ship : "",
        allDayOverdue: service ? service.preferences.allDayOverdue : true
    })
    readonly property var displayedReminders: viewMode === "list" ? TendModel.visibleReminders(queriedReminders, collapsedReminderIds) : queriedReminders
    readonly property string viewTitle: {
        if (viewMode === "list")
            return selectedList ? selectedList.title : "Create your first list";

        return viewMode.charAt(0).toUpperCase() + viewMode.slice(1);
    }
    readonly property var parentChoices: {
        var choices = [{
            id: 0,
            title: "No parent"
        }];
        if (!selectedList)
            return choices;

        for (var i = 0; i < selectedList.reminders.length; i++) {
            if (selectedList.reminders[i].id !== selectedReminderId)
                choices.push({
                    id: selectedList.reminders[i].id,
                    title: selectedList.reminders[i].title
                });

        }
        return choices;
    }

    function activityLabel(event) {
        var subject = "";
        if (selectedList && event.reminderId !== null) {
            for (var i = 0; i < selectedList.reminders.length; i++) {
                if (Number(selectedList.reminders[i].id) === Number(event.reminderId)) {
                    subject = " · " + selectedList.reminders[i].title;
                    break;
                }
            }
        }
        var action = String(event.kind || "changed").replace(/-/g, " ");
        var occurrence = event.occurredAt ? " · occurrence " + event.occurredAt : "";
        return String(event.actor || "Unknown ship") + " · " + action + subject + occurrence;
    }
    readonly property var assigneeChoices: {
        var choices = [{
            ship: "",
            title: "Unassigned"
        }];
        if (!selectedAccess)
            return choices;
        var seen = {};
        var values = [{ ship: selectedAccess.host }].concat(selectedAccess.members || []);
        for (var i = 0; i < values.length; i++) {
            var ship = String(values[i].ship || "");
            var key = ship.replace(/^~/, "");
            if (!key || seen[key])
                continue;
            seen[key] = true;
            choices.push({ ship: ship, title: ship.charAt(0) === "~" ? ship : "~" + ship });
        }
        return choices;
    }
    readonly property var sectionChoices: {
        var choices = [{
            id: 0,
            title: "No section"
        }];
        return selectedList ? choices.concat(selectedList.sections) : choices;
    }

    function indexForId(values, id) {
        var wanted = id === null || id === undefined ? 0 : id;
        for (var i = 0; i < values.length; i++) {
            if (values[i].id === wanted)
                return i;

        }
        return 0;
    }

    function indexForShip(values, ship) {
        var wanted = String(ship || "").replace(/^~/, "");
        for (var i = 0; i < values.length; i++) {
            if (String(values[i].ship || "").replace(/^~/, "") === wanted)
                return i;
        }
        return 0;
    }

    function reminderIsSelected(reminderId) {
        return selectedReminderIds.indexOf(Number(reminderId)) !== -1;
    }

    function toggleReminderSelection(reminderId) {
        var ids = selectedReminderIds.slice();
        var wanted = Number(reminderId);
        var index = ids.indexOf(wanted);
        if (index === -1)
            ids.push(wanted);
        else
            ids.splice(index, 1);
        selectedReminderIds = ids;
        confirmBatchDelete = false;
    }

    function reminderIsCollapsed(reminderId) {
        return collapsedReminderIds.indexOf(Number(reminderId)) !== -1;
    }

    function reminderHasChildren(reminderId) {
        return TendModel.reminderHasChildren(queriedReminders, reminderId);
    }

    function setReminderCollapsed(reminderId, collapsed) {
        var ids = collapsedReminderIds.slice();
        var wanted = Number(reminderId);
        var index = ids.indexOf(wanted);
        if (collapsed && index === -1)
            ids.push(wanted);
        else if (!collapsed && index !== -1)
            ids.splice(index, 1);
        collapsedReminderIds = ids;
    }

    function toggleReminderCollapsed(reminderId) {
        setReminderCollapsed(reminderId, !reminderIsCollapsed(reminderId));
    }

    function batchStartingRank() {
        var highest = 0;
        if (selectedList) {
            for (var i = 0; i < selectedList.reminders.length; i++)
                highest = Math.max(highest, Number(selectedList.reminders[i].rank || 0));
        }
        return highest + 1024;
    }

    function updatePreferences(defaultList, pinnedLists, pinnedViews) {
        if (!service)
            return false;

        return service.setPreferences({
            defaultList: defaultList,
            pinnedLists: pinnedLists,
            pinnedViews: pinnedViews,
            snoozePresets: service.preferences.snoozePresets
        });
    }

    function parseQuickEntry(value) {
        var words = String(value || "").trim().split(/\s+/);
        var title = [];
        var tags = [];
        for (var i = 0; i < words.length; i++) {
            if (words[i].length > 1 && words[i].charAt(0) === "#")
                tags.push(words[i].slice(1));
            else
                title.push(words[i]);

        }
        return {
            title: title.join(" ").trim(),
            tags: tags
        };
    }

    function toggleListPin() {
        if (!selectedList || !service)
            return ;

        var pins = service.preferences.pinnedLists.slice();
        var index = pins.indexOf(selectedList.id);
        if (index === -1)
            pins.push(selectedList.id);
        else
            pins.splice(index, 1);
        updatePreferences(service.preferences.defaultList, pins, service.preferences.pinnedViews);
    }

    function toggleViewPin(view) {
        if (!service)
            return ;

        var pins = service.preferences.pinnedViews.slice();
        var index = pins.indexOf(view);
        if (index === -1)
            pins.push(view);
        else
            pins.splice(index, 1);
        updatePreferences(service.preferences.defaultList, service.preferences.pinnedLists, pins);
    }

    function movePinnedValue(values, wanted, delta) {
        var next = values.slice();
        var index = next.indexOf(wanted);
        var target = index + delta;
        if (index < 0 || target < 0 || target >= next.length)
            return values;
        var swap = next[target];
        next[target] = next[index];
        next[index] = swap;
        return next;
    }

    function movePinnedList(delta) {
        if (!selectedList || !service)
            return;
        var pins = movePinnedValue(service.preferences.pinnedLists, selectedList.id, delta);
        if (pins !== service.preferences.pinnedLists)
            updatePreferences(service.preferences.defaultList, pins, service.preferences.pinnedViews);
    }

    function loadSelectedPresentation() {
        if (!selectedList)
            return ;
        var presentation = TendModel.presentationForList(service ? service.localSettings : null, selectedList.id);
        sortMode = presentation.sort;
        sortDescending = presentation.descending;
    }

    function saveSelectedPresentation() {
        if (service && selectedList)
            service.setListPresentation(selectedList.id, sortMode, sortDescending);
    }

    function unpinnedListIds() {
        if (!service)
            return [];
        var pins = service.preferences.pinnedLists || [];
        return TendModel.orderedLists(service.lists, [], service.localSettings.listOrder).filter(function(list) {
            return pins.indexOf(list.id) === -1;
        }).map(function(list) { return list.id; });
    }

    function canMoveList(delta) {
        if (!selectedList || !service || service.preferences.pinnedLists.indexOf(selectedList.id) !== -1)
            return false;
        var ids = unpinnedListIds();
        var index = ids.indexOf(selectedList.id);
        return index >= 0 && index + delta >= 0 && index + delta < ids.length;
    }

    function moveList(delta) {
        if (!canMoveList(delta))
            return ;
        var order = TendModel.orderedLists(service.lists, [], service.localSettings.listOrder).map(function(list) { return list.id; });
        var unpinned = unpinnedListIds();
        var index = unpinned.indexOf(selectedList.id);
        var targetId = unpinned[index + delta];
        var sourceIndex = order.indexOf(selectedList.id);
        var targetIndex = order.indexOf(targetId);
        var swap = order[targetIndex];
        order[targetIndex] = order[sourceIndex];
        order[sourceIndex] = swap;
        service.setListOrder(order);
    }

    onSelectedListIdChanged: loadSelectedPresentation()

    Connections {
        target: root.service
        function onLocalSettingsChanged() { root.loadSelectedPresentation(); }
    }

    function movePinnedView(view, delta) {
        if (!service)
            return;
        var pins = movePinnedValue(service.preferences.pinnedViews, view, delta);
        if (pins !== service.preferences.pinnedViews)
            updatePreferences(service.preferences.defaultList, service.preferences.pinnedLists, pins);
    }

    function sectionTitle(sectionId) {
        if (!selectedList || sectionId === null || sectionId === undefined)
            return "";

        for (var i = 0; i < selectedList.sections.length; i++) {
            if (selectedList.sections[i].id === sectionId)
                return selectedList.sections[i].title;

        }
        return "";
    }

    function sectionTitleFor(listId, sectionId) {
        if (sectionId === null || sectionId === undefined || !service)
            return "";

        for (var i = 0; i < service.lists.length; i++) {
            if (service.lists[i].id !== listId)
                continue;

            for (var j = 0; j < service.lists[i].sections.length; j++) {
                if (service.lists[i].sections[j].id === sectionId)
                    return service.lists[i].sections[j].title;
            }
        }
        return "";
    }

    function editReminder(reminder) {
        selectedListId = reminder.listId || selectedListId;
        selectedReminderId = reminder.id;
        reminderTitle.text = reminder.title;
        reminderNotes.text = reminder.notes;
        reminderUrl.text = reminder.url || "";
        reminderPriority.currentIndex = Math.max(0, reminderPriority.model.indexOf(reminder.priority));
        reminderFlagged.checked = reminder.flagged;
        reminderTags.text = reminder.tags.join(", ");
        reminderDue.text = TendModel.scheduleInputValue(reminder.schedule);
        reminderAllDay.checked = reminder.schedule ? reminder.schedule.allDay : false;
        reminderTimezone.text = reminder.schedule ? reminder.schedule.timezone : localTimezone();
        reminderEarly.text = reminder.schedule ? reminder.schedule.earlySeconds.map(function(seconds) {
            return seconds / 60;
        }).join(", ") : "";
        reminderRepeat.currentIndex = reminder.schedule && reminder.schedule.recurrence ? Math.max(0, reminderRepeat.model.indexOf(reminder.schedule.recurrence.frequency)) : 0;
        reminderInterval.value = reminder.schedule && reminder.schedule.recurrence ? reminder.schedule.recurrence.interval : 1;
        reminderWeekdays.text = reminder.schedule && reminder.schedule.recurrence ? reminder.schedule.recurrence.weekdays.join(", ") : "";
        reminderMonthDays.text = reminder.schedule && reminder.schedule.recurrence ? reminder.schedule.recurrence.monthDays.join(", ") : "";
        reminderOrdinal.checked = reminder.schedule && reminder.schedule.recurrence && reminder.schedule.recurrence.monthWeek !== null;
        reminderOrdinalIndex.value = reminderOrdinal.checked ? reminder.schedule.recurrence.monthWeek.index : 1;
        reminderOrdinalWeekday.value = reminderOrdinal.checked ? reminder.schedule.recurrence.monthWeek.weekday : 0;
        reminderRepeatEnd.text = reminder.schedule && reminder.schedule.recurrence && reminder.schedule.recurrence.endAt ? (reminder.schedule.recurrence.localEnd || reminder.schedule.recurrence.endAt) : "";
        reminderRepeatCount.value = reminder.schedule && reminder.schedule.recurrence && reminder.schedule.recurrence.maxOccurrences ? reminder.schedule.recurrence.maxOccurrences : 0;
        reminderRank.value = reminder.rank;
        Qt.callLater(function() {
            reminderParent.currentIndex = root.indexForId(root.parentChoices, reminder.parentId);
            reminderSection.currentIndex = root.indexForId(root.sectionChoices, reminder.sectionId);
            reminderAssignee.currentIndex = root.indexForShip(root.assigneeChoices, reminder.assignee);
        });
    }

    function openListEditor() {
        if (!selectedList)
            return ;

        listTitle.text = selectedList.title;
        listColor.text = selectedList.color;
        listSymbol.text = selectedList.symbol;
        listEditorOpen = true;
        confirmDeleteList = false;
    }

    function chooseSection(index) {
        if (!selectedList || index < 0 || index >= selectedList.sections.length) {
            selectedSectionId = 0;
            sectionTitleEditor.text = "";
            return ;
        }

        var section = selectedList.sections[index];
        selectedSectionId = section.id;
        sectionTitleEditor.text = section.title;
    }

    function selectedSectionValue() {
        if (!selectedList)
            return null;
        for (var i = 0; i < selectedList.sections.length; i++) {
            if (Number(selectedList.sections[i].id) === Number(selectedSectionId))
                return selectedList.sections[i];
        }
        return null;
    }

    function moveSelectedSectionRelative(delta) {
        var section = selectedSectionValue();
        if (!section || !selectedList || !service)
            return false;
        var sections = selectedList.sections.slice().sort(function(left, right) {
            return Number(left.rank) - Number(right.rank) || Number(left.id) - Number(right.id);
        });
        var index = sections.findIndex(function(value) {
            return Number(value.id) === Number(section.id);
        });
        var target = index + delta;
        if (index < 0 || target < 0 || target >= sections.length)
            return false;
        return service.placeSection(selectedList.id, section.id, sections[target].id, delta > 0, selectedList.revision);
    }

    function moveSelectedRelative(delta) {
        if (!selectedList || !selectedReminder)
            return ;

        var siblings = selectedList.reminders.filter(function(item) {
            return item.parentId === selectedReminder.parentId && item.sectionId === selectedReminder.sectionId;
        }).sort(function(a, b) {
            return a.rank - b.rank || a.id - b.id;
        });
        var index = -1;
        for (var i = 0; i < siblings.length; i++) {
            if (siblings[i].id === selectedReminder.id) {
                index = i;
                break;
            }
        }
        var target = index + delta;
        if (index < 0 || target < 0 || target >= siblings.length)
            return ;

        service.placeReminder(selectedList.id, selectedReminder.id, siblings[target].id, delta > 0, selectedList.revision);
    }

    function placeReminder(source, target, after) {
        if (!selectedList || !source || !target || !service || service.mutationPending)
            return false;
        var placement = TendModel.manualDropPlacement(selectedList.reminders, source.id, target.id, after);
        if (!placement)
            return false;
        selectedReminderId = Number(source.id);
        return service.placeReminder(selectedList.id, source.id, placement.targetId, placement.after, selectedList.revision);
    }

    function reminderFromSelectedList(reminderId) {
        if (!selectedList)
            return null;
        for (var i = 0; i < selectedList.reminders.length; i++) {
            if (selectedList.reminders[i].id === Number(reminderId))
                return selectedList.reminders[i];
        }
        return null;
    }

    function indentSelectedReminder() {
        if (!selectedListEditable || !selectedReminder || !service)
            return;
        var index = -1;
        for (var i = 0; i < displayedReminders.length; i++) {
            if (displayedReminders[i].id === selectedReminder.id) {
                index = i;
                break;
            }
        }
        if (index <= 0)
            return;
        var parent = displayedReminders[index - 1];
        if (parent.listId !== selectedList.id)
            return;
        service.moveReminder(selectedList.id, selectedReminder.id, parent.id, parent.sectionId, selectedReminder.rank, selectedList.revision);
    }

    function outdentSelectedReminder() {
        if (!selectedListEditable || !selectedReminder || selectedReminder.parentId === null || !service)
            return;
        var parent = reminderFromSelectedList(selectedReminder.parentId);
        if (!parent)
            return;
        service.moveReminder(selectedList.id, selectedReminder.id, parent.parentId, parent.sectionId, selectedReminder.rank, selectedList.revision);
    }

    function localTimezone() {
        return service ? service.localTimezone : "UTC";
    }

    function screenNamed(name) {
        var wanted = String(name || "");
        if (!wanted)
            return null;
        var screens = Quickshell.screens || [];
        for (var i = 0; i < screens.length; i++) {
            if (String(screens[i].name || "") === wanted)
                return screens[i];
        }
        return null;
    }

    function chooseOpenScreen(requestedName) {
        var focused = Hyprland.focusedMonitor;
        var focusedName = focused ? String(focused.name || "") : "";
        var chosen = screenNamed(requestedName) || screenNamed(focusedName);
        if (chosen)
            window.screen = chosen;
    }

    function open(payloadJson) {
        var payload = {};
        try {
            payload = JSON.parse(String(payloadJson || "{}"));
        } catch (error) {
            payload = {};
        }
        if (!payload || typeof payload !== "object")
            payload = {};
        root.chooseOpenScreen(payload.screen);
        root.captureMode = payload.mode === "capture";
        root.opened = true;
        Qt.callLater(function() {
            if (payload.listId && service) {
                root.viewMode = "list";
                root.selectedListId = Number(payload.listId);
                root.selectedReminderId = Number(payload.reminderId || 0);
                for (var i = 0; i < service.lists.length; i++) {
                    if (service.lists[i].id !== root.selectedListId)
                        continue;

                    for (var j = 0; j < service.lists[i].reminders.length; j++) {
                        if (service.lists[i].reminders[j].id === root.selectedReminderId)
                            root.editReminder(service.lists[i].reminders[j]);

                    }
                }
            }
            if (service && service.ship && root.captureMode)
                captureEntry.forceActiveFocus();
            else if (service && service.ship)
                quickAdd.forceActiveFocus();
            else
                shipUrl.forceActiveFocus();
        });
    }

    function close() {
        root.opened = false;
        root.captureMode = false;
    }

    function dismiss() {
        root.opened = false;
        root.captureMode = false;
        if (root.shell)
            root.shell.hide("io.omabit.tend");

    }

    function toggle() {
        if (root.opened)
            root.dismiss();
        else
            root.open("{}");
    }

    Shortcut {
        sequence: "Ctrl+N"
        enabled: root.opened && !root.captureMode && service && service.ship !== ""
        onActivated: quickAdd.forceActiveFocus()
    }

    Shortcut {
        sequence: "Ctrl+F"
        enabled: root.opened && !root.captureMode && service && service.ship !== ""
        onActivated: reminderSearch.forceActiveFocus()
    }

    Shortcut {
        sequence: "Ctrl+Shift+N"
        enabled: root.opened && !root.captureMode && service && service.ship !== ""
        onActivated: newList.forceActiveFocus()
    }

    Shortcut {
        sequence: "Ctrl+Comma"
        enabled: root.opened && !root.captureMode && service && service.ship !== ""
        onActivated: root.policyEditorOpen = !root.policyEditorOpen
    }

    Shortcut {
        sequence: "Alt+1"
        enabled: root.opened && !root.captureMode
        onActivated: root.viewMode = "today"
    }

    Shortcut {
        sequence: "Alt+2"
        enabled: root.opened && !root.captureMode
        onActivated: root.viewMode = "scheduled"
    }

    Shortcut {
        sequence: "Alt+3"
        enabled: root.opened && !root.captureMode
        onActivated: root.viewMode = "all"
    }

    Shortcut {
        sequence: "Alt+4"
        enabled: root.opened && !root.captureMode
        onActivated: root.viewMode = "flagged"
    }

    Shortcut {
        sequence: "Alt+5"
        enabled: root.opened && !root.captureMode
        onActivated: root.viewMode = "assigned"
    }

    Shortcut {
        sequence: "Alt+6"
        enabled: root.opened && !root.captureMode
        onActivated: root.viewMode = "completed"
    }

    Shortcut {
        sequence: "Ctrl+]"
        enabled: root.opened && !root.captureMode && root.selectedReminder !== null
        onActivated: root.indentSelectedReminder()
    }

    Shortcut {
        sequence: "Ctrl+["
        enabled: root.opened && !root.captureMode && root.selectedReminder !== null
        onActivated: root.outdentSelectedReminder()
    }

    onSelectedListChanged: {
        if (selectedList) {
            selectedListId = selectedList.id;
            listEditorOpen = false;
            confirmDeleteList = false;
            selectedSectionId = 0;
            selectedReminderIds = [];
            collapsedReminderIds = [];
            confirmBatchDelete = false;
        }
    }

    onViewModeChanged: {
        selectedReminderIds = [];
        confirmBatchDelete = false;
    }

    PanelWindow {
        id: window

        visible: root.opened
        color: "transparent"
        WlrLayershell.namespace: "omabit-tend"
        WlrLayershell.layer: WlrLayer.Overlay
        WlrLayershell.keyboardFocus: WlrKeyboardFocus.Exclusive
        exclusionMode: ExclusionMode.Ignore

        anchors {
            top: true
            bottom: true
            left: true
            right: true
        }

        Rectangle {
            anchors.fill: parent
            color: Color.menu.scrim

            MouseArea {
                anchors.fill: parent
                onClicked: root.dismiss()
            }

            Rectangle {
                id: card

                anchors.centerIn: parent
                width: root.captureMode ? Math.min(Style.space(640), parent.width - Style.space(48)) : Math.min(Style.space(1100), parent.width - Style.space(48))
                height: root.captureMode ? Math.min(captureControls.columns === 1 ? Style.space(300) : Style.space(210), parent.height - Style.space(48)) : Math.min(Style.space(760), parent.height - Style.space(48))
                radius: Style.cornerRadius
                color: Color.menu.background
                border.color: Color.menu.border
                border.width: Math.max(1, Style.space(1))
                Keys.onEscapePressed: root.dismiss()

                MouseArea {
                    anchors.fill: parent
                    onClicked: {
                    }
                }

                Column {
                    anchors.fill: parent
                    anchors.margins: Style.spacing.panelPadding
                    spacing: Style.space(12)

                    Row {
                        width: parent.width
                        height: Style.space(38)

                        Text {
                            width: parent.width - closeButton.width - switchShipButton.width - Style.space(8)
                            text: service && service.ship ? "Tend · ~" + service.ship : "Tend"
                            color: Color.menu.text
                            font.family: Style.font.menuFamily
                            font.pixelSize: Style.font.heading
                            font.bold: true
                        }

                        Button {
                            id: switchShipButton

                            visible: service && service.ship !== ""
                            text: "Switch ship"
                            Accessible.name: "Disconnect and switch Urbit ship"
                            onClicked: service.disconnect()
                        }

                        Button {
                            id: closeButton

                            text: "Close"
                            onClicked: root.dismiss()
                        }

                    }

                    Text {
                        width: parent.width
                        visible: service && service.errorMessage !== ""
                        text: service ? service.errorMessage : ""
                        color: "#ef4444"
                        wrapMode: Text.Wrap
                        font.family: Style.font.menuFamily
                        font.pixelSize: Style.font.body
                    }

                    Column {
                        visible: service && service.ship !== "" && !root.captureMode && service.invitations.length > 0
                        width: parent.width
                        spacing: Style.space(6)

                        Text {
                            text: "Shared-list invitations"
                            color: Color.menu.text
                            font.family: Style.font.menuFamily
                            font.pixelSize: Style.font.caption
                            font.bold: true
                        }

                        Repeater {
                            model: service ? service.invitations : []

                            delegate: Row {
                                required property var modelData

                                width: parent.width
                                spacing: Style.space(6)

                                Text {
                                    width: parent.width - acceptInvitationButton.width - declineInvitationButton.width - parent.spacing * 2
                                    anchors.verticalCenter: parent.verticalCenter
                                    text: modelData.title + " from " + modelData.host + (modelData.canInvite ? " · may invite" : "")
                                    elide: Text.ElideRight
                                    color: Color.menu.text
                                    font.family: Style.font.menuFamily
                                    font.pixelSize: Style.font.body
                                }

                                Button {
                                    id: acceptInvitationButton

                                    text: "Accept"
                                    enabled: service && service.connectionState === "online" && !service.mutationPending
                                    Accessible.name: "Accept invitation to " + modelData.title
                                    onClicked: service.acceptInvitation(modelData)
                                }

                                Button {
                                    id: declineInvitationButton

                                    text: "Decline"
                                    enabled: service && service.connectionState === "online" && !service.mutationPending
                                    Accessible.name: "Decline invitation to " + modelData.title
                                    onClicked: service.declineInvitation(modelData)
                                }
                            }
                        }
                    }

                    Column {
                        visible: !service || !service.ship || service.connectionState === "authentication-required"
                        width: Math.min(560, parent.width)
                        anchors.horizontalCenter: parent.horizontalCenter
                        spacing: Style.space(12)

                        Text {
                            width: parent.width
                            text: "Connect your Urbit"
                            color: Color.menu.text
                            font.family: Style.font.menuFamily
                            font.pixelSize: Style.font.title
                            font.bold: true
                        }

                        Text {
                            width: parent.width
                            text: service && service.connectionState === "authentication-required" ? "Your Eyre session expired. Enter the current +code to reconnect to this ship." : "Enter your ship domain and the current code printed by +code. Tend accepts planets, moons, and comets."
                            color: Color.menu.text
                            opacity: 0.72
                            wrapMode: Text.Wrap
                            font.family: Style.font.menuFamily
                            font.pixelSize: Style.font.body
                        }

                        TextField {
                            id: shipUrl

                            width: parent.width
                            text: service ? service.baseUrl : ""
                            placeholderText: "https://sampel-palnet.arvo.network"
                            Accessible.name: "Urbit Eyre URL"
                        }

                        TextField {
                            id: loginCode

                            width: parent.width
                            placeholderText: "lidlut-tabwed-pillex-ridrup"
                            echoMode: TextInput.Password
                            Accessible.name: "Current Urbit plus code"
                            onAccepted: connectButton.clicked()
                        }

                        Button {
                            id: connectButton

                            text: service && service.connectionState === "authenticating" ? "Connecting…" : "Connect"
                            enabled: service && service.connectionState !== "authenticating" && shipUrl.text.trim() !== "" && loginCode.text.trim() !== ""
                            onClicked: {
                                service.login(shipUrl.text, loginCode.text);
                                loginCode.text = "";
                            }
                        }

                    }

                    Column {
                        visible: service && service.ship !== "" && service.connectionState !== "authentication-required" && root.captureMode
                        width: parent.width
                        spacing: Style.space(10)

                        Text {
                            width: parent.width
                            text: "Quick capture · " + (service ? service.connectionState.toUpperCase() : "DISCONNECTED")
                            color: service && service.connectionState === "online" ? "#22c55e" : "#f59e0b"
                            font.family: Style.font.menuFamily
                            font.pixelSize: Style.font.caption
                            font.bold: true
                        }

                        Grid {
                            id: captureControls

                            width: parent.width
                            columns: width < Style.space(520) ? 1 : 3
                            columnSpacing: Style.space(8)
                            rowSpacing: Style.space(8)

                            TextField {
                                id: captureEntry

                                width: captureControls.columns === 1 ? parent.width : parent.width * 0.64
                                placeholderText: "Reminder title with optional #tags"
                                enabled: captureList.currentIndex >= 0 && captureList.model[captureList.currentIndex] && service && service.canEditList(captureList.model[captureList.currentIndex].id) && service.connectionState === "online" && !service.mutationPending
                                onAccepted: captureAdd.clicked()
                                Accessible.name: "Quick reminder title and tags"
                            }

                            ComboBox {
                                id: captureList

                                width: captureControls.columns === 1 ? parent.width : parent.width - captureEntry.width - captureAdd.width - captureControls.columnSpacing * 2
                                model: service ? TendModel.orderedLists(service.lists, service.preferences.pinnedLists, service.localSettings.listOrder) : []
                                textRole: "title"
                                Accessible.name: "Destination list"
                            }

                            Button {
                                id: captureAdd

                                width: captureControls.columns === 1 ? parent.width : implicitWidth
                                text: service && service.mutationPending ? "Adding…" : "Add"
                                enabled: captureEntry.enabled && captureEntry.text.trim() !== ""
                                Accessible.name: "Add reminder"
                                onClicked: {
                                    var parsed = root.parseQuickEntry(captureEntry.text);
                                    var destination = captureList.currentIndex >= 0 ? captureList.model[captureList.currentIndex] : null;
                                    if (destination && parsed.title && service.addReminder(destination.id, parsed.title, destination.revision, parsed.tags)) {
                                        captureEntry.text = "";
                                        root.dismiss();
                                    }
                                }
                            }

                        }

                        Text {
                            width: parent.width
                            text: "Enter adds · Escape closes · use #tag to classify"
                            color: Color.menu.text
                            opacity: 0.68
                            font.family: Style.font.menuFamily
                            font.pixelSize: Style.font.caption
                        }

                    }

                    Row {
                        visible: service && service.ship !== "" && service.connectionState !== "authentication-required" && !root.captureMode
                        width: parent.width
                        height: parent.height - y
                        spacing: Style.space(16)

                        Rectangle {
                            width: Style.space(230)
                            height: parent.height
                            radius: Style.cornerRadius
                            color: Qt.rgba(Color.menu.text.r, Color.menu.text.g, Color.menu.text.b, 0.05)

                            Column {
                                anchors.fill: parent
                                anchors.margins: Style.space(10)
                                spacing: Style.space(6)

                                Text {
                                    text: service ? service.connectionState.toUpperCase() : "DISCONNECTED"
                                    color: service && service.connectionState === "online" ? "#22c55e" : "#f59e0b"
                                    font.family: Style.font.menuFamily
                                    font.pixelSize: Style.font.caption
                                    font.bold: true
                                }

                                Grid {
                                    width: parent.width
                                    columns: 2
                                    spacing: Style.space(4)

                                    Repeater {
                                        model: root.orderedViews

                                        delegate: Button {
                                            required property var modelData

                                            width: (parent.width - parent.spacing) / 2
                                            text: (service && service.preferences.pinnedViews.indexOf(modelData) !== -1 ? "★ " : "") + modelData.charAt(0).toUpperCase() + modelData.slice(1)
                                            checkable: true
                                            checked: root.viewMode === modelData
                                            onClicked: {
                                                root.viewMode = modelData;
                                                root.selectedReminderId = 0;
                                            }
                                        }

                                    }

                                }

                                Row {
                                    width: parent.width
                                    spacing: Style.space(4)

                                    ComboBox {
                                        id: viewPinChoice

                                        width: parent.width * 0.62
                                        model: ["today", "scheduled", "all", "flagged", "assigned", "completed"]
                                        Accessible.name: "Smart view to pin"
                                    }

                                    Button {
                                        width: parent.width - viewPinChoice.width - parent.spacing
                                        text: service && service.preferences.pinnedViews.indexOf(viewPinChoice.currentText) !== -1 ? "Unpin" : "Pin"
                                        enabled: service && service.connectionState === "online" && !service.mutationPending
                                        onClicked: root.toggleViewPin(viewPinChoice.currentText)
                                    }

                                }

                                Row {
                                    width: parent.width
                                    spacing: Style.space(6)

                                    Button {
                                        width: (parent.width - parent.spacing) / 2
                                        text: "Pinned view ↑"
                                        enabled: service && service.preferences.pinnedViews.indexOf(viewPinChoice.currentText) > 0 && service.connectionState === "online" && !service.mutationPending
                                        onClicked: root.movePinnedView(viewPinChoice.currentText, -1)
                                    }

                                    Button {
                                        width: (parent.width - parent.spacing) / 2
                                        text: "Pinned view ↓"
                                        enabled: service && service.preferences.pinnedViews.indexOf(viewPinChoice.currentText) >= 0 && service.preferences.pinnedViews.indexOf(viewPinChoice.currentText) < service.preferences.pinnedViews.length - 1 && service.connectionState === "online" && !service.mutationPending
                                        onClicked: root.movePinnedView(viewPinChoice.currentText, 1)
                                    }

                                }

                                Text {
                                    text: "My Lists"
                                    color: Color.menu.text
                                    opacity: 0.68
                                    font.family: Style.font.menuFamily
                                    font.pixelSize: Style.font.caption
                                    font.bold: true
                                }

                                Repeater {
                                    model: root.orderedLists

                                    delegate: Button {
                                        required property var modelData

                                        width: parent.width
                                        text: {
                                            var pinned = service && service.preferences.pinnedLists.indexOf(modelData.id) !== -1 ? "★ " : "";
                                            var primary = service && service.preferences.defaultList === modelData.id ? " · default" : "";
                                            return pinned + modelData.title + primary;
                                        }
                                        checkable: true
                                        checked: root.viewMode === "list" && root.selectedList && root.selectedList.id === modelData.id
                                        onClicked: {
                                            root.viewMode = "list";
                                            root.selectedListId = modelData.id;
                                            root.selectedReminderId = 0;
                                        }
                                    }

                                }

                                TextField {
                                    id: newList

                                    width: parent.width
                                    placeholderText: "New list"
                                    Accessible.name: "New list title"
                                    enabled: service && service.connectionState === "online" && !service.mutationPending
                                    onAccepted: {
                                        if (text.trim() && service.createList(text))
                                            text = "";

                                    }
                                }

                                Text {
                                    width: parent.width
                                    text: "Keys: Ctrl+N add · Ctrl+F search · Ctrl+Shift+N list · Alt+1…6 views · Enter edit · Space complete"
                                    color: Color.menu.text
                                    opacity: 0.68
                                    wrapMode: Text.Wrap
                                    font.family: Style.font.menuFamily
                                    font.pixelSize: Style.font.caption
                                }

                            }

                        }

                        Column {
                            width: parent.width - Style.space(246)
                            height: parent.height
                            spacing: Style.space(8)

                            Row {
                                width: parent.width
                                spacing: Style.space(8)

                                Text {
                                    width: parent.width - editListButton.width - alertSettingsButton.width - parent.spacing * 2
                                    text: root.viewTitle
                                    color: Color.menu.text
                                    font.family: Style.font.menuFamily
                                    font.pixelSize: Style.font.title
                                    font.bold: true
                                }

                                Button {
                                    id: alertSettingsButton

                                    text: root.policyEditorOpen ? "Close alerts" : "Alerts"
                                    onClicked: root.policyEditorOpen = !root.policyEditorOpen
                                }

                                Button {
                                    id: editListButton

                                    visible: root.viewMode === "list" && root.selectedList !== null
                                    text: root.listEditorOpen ? "Close list settings" : "List settings"
                                    onClicked: {
                                        if (root.listEditorOpen)
                                            root.listEditorOpen = false;
                                        else
                                            root.openListEditor();
                                    }
                                }

                            }

                            Row {
                                visible: root.policyEditorOpen
                                width: parent.width
                                spacing: Style.space(6)

                                ComboBox {
                                    id: badgeMode

                                    width: parent.width * 0.22
                                    model: ["today", "all", "assigned", "none"]
                                    currentIndex: service ? Math.max(0, model.indexOf(service.preferences.badgeMode)) : 0
                                    Accessible.name: "Bar badge count"
                                }

                                SpinBox {
                                    id: allDayHour

                                    width: parent.width * 0.18
                                    from: 0
                                    to: 23
                                    value: service ? Math.floor(service.preferences.allDayAlertMinute / 60) : 9
                                    Accessible.name: "All-day reminder hour"
                                }

                                SpinBox {
                                    id: allDayMinute

                                    width: parent.width * 0.18
                                    from: 0
                                    to: 59
                                    value: service ? service.preferences.allDayAlertMinute % 60 : 0
                                    Accessible.name: "All-day reminder minute"
                                }

                                CheckBox {
                                    id: allDayOverdue

                                    text: "Keep overdue"
                                    checked: service ? service.preferences.allDayOverdue : true
                                }

                                Button {
                                    text: "Save"
                                    enabled: service && service.connectionState === "online" && !service.mutationPending
                                    onClicked: service.setReminderPolicy({
                                        badgeMode: badgeMode.currentText,
                                        allDayAlertMinute: allDayHour.value * 60 + allDayMinute.value,
                                        allDayOverdue: allDayOverdue.checked
                                    })
                                }

                            }

                            Text {
                                visible: root.viewMode === "list" && root.selectedList !== null
                                width: parent.width
                                text: root.selectedHostStatus + (root.selectedListEditable ? "" : " · READ-ONLY UNTIL HOST RETURNS")
                                color: root.selectedListEditable ? "#22c55e" : "#f59e0b"
                                font.family: Style.font.menuFamily
                                font.pixelSize: Style.font.caption
                                font.bold: true
                            }

                            Row {
                                visible: root.availableTags.length > 0
                                width: parent.width
                                spacing: Style.space(6)

                                ComboBox {
                                    id: tagChoice

                                    width: parent.width * 0.25
                                    model: root.availableTags
                                    Accessible.name: "Tag to rename or delete"
                                }

                                TextField {
                                    id: tagReplacement

                                    width: parent.width * 0.3
                                    placeholderText: "Rename or merge tag"
                                    Accessible.name: "Replacement tag"
                                }

                                Button {
                                    text: "Rename tag"
                                    enabled: tagChoice.currentIndex >= 0 && tagReplacement.text.trim() && service && service.connectionState === "online" && !service.mutationPending
                                    onClicked: {
                                        if (service.replaceTag(tagChoice.currentText, tagReplacement.text))
                                            tagReplacement.text = "";
                                    }
                                }

                                Button {
                                    text: "Delete tag"
                                    enabled: tagChoice.currentIndex >= 0 && service && service.connectionState === "online" && !service.mutationPending
                                    onClicked: service.replaceTag(tagChoice.currentText, null)
                                }

                                Text {
                                    anchors.verticalCenter: parent.verticalCenter
                                    text: "Use #tag in quick add"
                                    color: Color.menu.text
                                    opacity: 0.68
                                    font.family: Style.font.menuFamily
                                    font.pixelSize: Style.font.caption
                                }

                            }

                            Row {
                                visible: root.listEditorOpen && root.selectedList !== null
                                width: parent.width
                                spacing: Style.space(6)

                                TextField {
                                    id: listTitle

                                    width: parent.width * 0.34
                                    placeholderText: "List title"
                                    Accessible.name: "List title"
                                }

                                TextField {
                                    id: listColor

                                    width: parent.width * 0.2
                                    placeholderText: "#3b82f6"
                                    Accessible.name: "List color"
                                }

                                TextField {
                                    id: listSymbol

                                    width: parent.width * 0.16
                                    placeholderText: "list or emoji"
                                    Accessible.name: "List symbol or emoji"
                                }

                                Button {
                                    text: "Save"
                                    enabled: root.selectedListEditable && listTitle.text.trim() && listColor.text.trim() && listSymbol.text.trim() && service && service.connectionState === "online" && !service.mutationPending
                                    onClicked: service.updateList(root.selectedList.id, listTitle.text, listColor.text, listSymbol.text, root.selectedList.revision)
                                }

                                Button {
                                    text: root.confirmDeleteList ? "Confirm delete" : "Delete"
                                    enabled: root.selectedListEditable && service && service.connectionState === "online" && !service.mutationPending
                                    onClicked: {
                                        if (!root.confirmDeleteList) {
                                            root.confirmDeleteList = true;
                                            return ;
                                        }
                                        service.deleteList(root.selectedList.id, root.selectedList.revision);
                                        root.confirmDeleteList = false;
                                        root.listEditorOpen = false;
                                    }
                                }

                            }

                            Row {
                                visible: root.listEditorOpen && root.selectedList !== null
                                width: parent.width
                                spacing: Style.space(6)

                                Button {
                                    text: service && service.preferences.defaultList === root.selectedList.id ? "Default list" : "Make default"
                                    enabled: service && service.preferences.defaultList !== root.selectedList.id && service.connectionState === "online" && !service.mutationPending
                                    onClicked: root.updatePreferences(root.selectedList.id, service.preferences.pinnedLists, service.preferences.pinnedViews)
                                }

                                Button {
                                    text: service && service.preferences.pinnedLists.indexOf(root.selectedList.id) !== -1 ? "Unpin list" : "Pin list"
                                    enabled: service && service.connectionState === "online" && !service.mutationPending
                                    onClicked: root.toggleListPin()
                                }

                                Button {
                                    text: "Pinned ↑"
                                    enabled: service && service.preferences.pinnedLists.indexOf(root.selectedList.id) > 0 && service.connectionState === "online" && !service.mutationPending
                                    onClicked: root.movePinnedList(-1)
                                }

                                Button {
                                    text: "Pinned ↓"
                                    enabled: service && service.preferences.pinnedLists.indexOf(root.selectedList.id) >= 0 && service.preferences.pinnedLists.indexOf(root.selectedList.id) < service.preferences.pinnedLists.length - 1 && service.connectionState === "online" && !service.mutationPending
                                    onClicked: root.movePinnedList(1)
                                }

                                Text {
                                    anchors.verticalCenter: parent.verticalCenter
                                    text: "Default list receives quick adds from smart views."
                                    color: Color.menu.text
                                    opacity: 0.68
                                    font.family: Style.font.menuFamily
                                    font.pixelSize: Style.font.caption
                                }

                            }

                            Row {
                                visible: root.listEditorOpen && root.selectedList !== null && service && service.preferences.pinnedLists.indexOf(root.selectedList.id) === -1
                                width: parent.width
                                spacing: Style.space(6)

                                Button {
                                    text: "List ↑"
                                    enabled: root.canMoveList(-1) && service.connectionState === "online" && !service.mutationPending
                                    Accessible.name: "Move selected unpinned list up"
                                    onClicked: root.moveList(-1)
                                }

                                Button {
                                    text: "List ↓"
                                    enabled: root.canMoveList(1) && service.connectionState === "online" && !service.mutationPending
                                    Accessible.name: "Move selected unpinned list down"
                                    onClicked: root.moveList(1)
                                }

                                Text {
                                    anchors.verticalCenter: parent.verticalCenter
                                    text: "List order is private to this ship."
                                    color: Color.menu.text
                                    opacity: 0.68
                                    font.family: Style.font.menuFamily
                                    font.pixelSize: Style.font.caption
                                }
                            }

                            Column {
                                visible: root.listEditorOpen && root.selectedList !== null && root.selectedAccess !== null
                                width: parent.width
                                spacing: Style.space(6)

                                Text {
                                    width: parent.width
                                    text: root.selectedAccess && root.selectedAccess.owner ? "Sharing · hosted by this ship" : "Sharing · hosted by " + (root.selectedAccess ? root.selectedAccess.host : "")
                                    color: Color.menu.text
                                    font.family: Style.font.menuFamily
                                    font.pixelSize: Style.font.caption
                                    font.bold: true
                                }

                                Row {
                                    visible: root.selectedMayInvite
                                    width: parent.width
                                    spacing: Style.space(6)

                                    TextField {
                                        id: inviteShip

                                        width: parent.width * 0.38
                                        placeholderText: "~sampel-palnet, moon, or comet"
                                        Accessible.name: "Urbit ship to invite"
                                    }

                                    CheckBox {
                                        id: inviteCanInvite

                                        text: "May invite"
                                        Accessible.name: "Allow invited member to invite others"
                                    }

                                    Button {
                                        text: "Invite"
                                        enabled: root.selectedListEditable && inviteShip.text.trim() !== "" && service && service.connectionState === "online" && !service.mutationPending
                                        Accessible.name: "Invite Urbit ship to selected list"
                                        onClicked: {
                                            if (service.inviteMember(root.selectedList.id, inviteShip.text.trim(), inviteCanInvite.checked))
                                                inviteShip.text = "";
                                        }
                                    }
                                }

                                Repeater {
                                    visible: root.selectedAccess && root.selectedAccess.pending && root.selectedAccess.pending.length > 0
                                    model: root.selectedAccess ? root.selectedAccess.pending : []

                                    delegate: Row {
                                        required property string modelData

                                        width: parent.width
                                        spacing: Style.space(6)

                                        Text {
                                            width: parent.width - revokeInviteButton.width - parent.spacing
                                            anchors.verticalCenter: parent.verticalCenter
                                            text: modelData + " · awaiting response"
                                            elide: Text.ElideRight
                                            color: Color.menu.text
                                            opacity: 0.72
                                            font.family: Style.font.menuFamily
                                            font.pixelSize: Style.font.caption
                                        }

                                        Button {
                                            id: revokeInviteButton

                                            visible: root.selectedAccess && root.selectedAccess.owner
                                            text: "Revoke"
                                            enabled: root.selectedListEditable && service && service.connectionState === "online" && !service.mutationPending
                                            Accessible.name: "Revoke invitation for " + modelData
                                            onClicked: service.removeMember(root.selectedList.id, modelData)
                                        }
                                    }
                                }

                                Repeater {
                                    model: root.selectedAccess ? root.selectedAccess.members : []

                                    delegate: Row {
                                        required property var modelData

                                        width: parent.width
                                        spacing: Style.space(6)

                                        Text {
                                            width: parent.width - removeMemberButton.width - parent.spacing
                                            anchors.verticalCenter: parent.verticalCenter
                                            text: modelData.ship + (modelData.policy && modelData.policy.canInvite ? " · may invite" : " · member")
                                            elide: Text.ElideRight
                                            color: Color.menu.text
                                            font.family: Style.font.menuFamily
                                            font.pixelSize: Style.font.body
                                        }

                                        Button {
                                            id: removeMemberButton

                                            visible: root.selectedAccess && root.selectedAccess.owner
                                            text: "Remove"
                                            enabled: root.selectedListEditable && service && service.connectionState === "online" && !service.mutationPending
                                            Accessible.name: "Remove " + modelData.ship + " from selected list"
                                            onClicked: service.removeMember(root.selectedList.id, modelData.ship)
                                        }
                                    }
                                }

                                Button {
                                    visible: root.selectedAccess && !root.selectedAccess.owner
                                    text: "Leave shared list"
                                    enabled: service && service.connectionState === "online" && !service.mutationPending
                                    Accessible.name: "Leave selected shared list"
                                    onClicked: service.leaveSharedList(root.selectedList.id)
                                }

                                Text {
                                    text: "My collaboration notifications"
                                    color: Color.menu.text
                                    font.family: Style.font.menuFamily
                                    font.pixelSize: Style.font.caption
                                    font.bold: true
                                }

                                Row {
                                    width: parent.width
                                    spacing: Style.space(8)

                                    CheckBox {
                                        id: notifyAdded
                                        text: "Items added"
                                        checked: root.selectedCollaborationPolicy.notifyAdded
                                        enabled: service && service.connectionState === "online" && !service.mutationPending
                                        Accessible.name: "Notify when another collaborator adds an item"
                                        onClicked: service.setCollaborationPolicy(root.selectedList.id, checked, notifyCompleted.checked, notifyAssigned.checked)
                                    }

                                    CheckBox {
                                        id: notifyCompleted
                                        text: "Items completed"
                                        checked: root.selectedCollaborationPolicy.notifyCompleted
                                        enabled: service && service.connectionState === "online" && !service.mutationPending
                                        Accessible.name: "Notify when another collaborator completes an item"
                                        onClicked: service.setCollaborationPolicy(root.selectedList.id, notifyAdded.checked, checked, notifyAssigned.checked)
                                    }

                                    CheckBox {
                                        id: notifyAssigned
                                        text: "Assigned to me"
                                        checked: root.selectedCollaborationPolicy.notifyAssigned
                                        enabled: service && service.connectionState === "online" && !service.mutationPending
                                        Accessible.name: "Notify when another collaborator assigns an item to me"
                                        onClicked: service.setCollaborationPolicy(root.selectedList.id, notifyAdded.checked, notifyCompleted.checked, checked)
                                    }
                                }

                                Text {
                                    visible: root.selectedActivities.length > 0
                                    text: "Recent activity"
                                    color: Color.menu.text
                                    font.family: Style.font.menuFamily
                                    font.pixelSize: Style.font.caption
                                    font.bold: true
                                }

                                Repeater {
                                    model: root.selectedActivities.slice(0, 20)

                                    delegate: Text {
                                        required property var modelData

                                        width: parent.width
                                        text: root.activityLabel(modelData)
                                        wrapMode: Text.Wrap
                                        color: Color.menu.text
                                        opacity: 0.75
                                        font.family: Style.font.menuFamily
                                        font.pixelSize: Style.font.caption
                                        Accessible.name: "Tend activity: " + text
                                    }
                                }
                            }

                            Row {
                                width: parent.width
                                spacing: Style.space(8)

                                TextField {
                                    id: reminderSearch

                                    width: parent.width * 0.34
                                    placeholderText: "Search reminders"
                                    Accessible.name: "Search reminders"
                                }

                                ComboBox {
                                    id: reminderTagFilter

                                    width: parent.width * 0.2
                                    model: ["All tags"].concat(root.availableTags)
                                    Accessible.name: "Filter reminders by tag"
                                    onCurrentIndexChanged: root.tagFilter = currentIndex > 0 ? currentText : ""
                                }

                                ComboBox {
                                    id: reminderSort

                                    width: parent.width * 0.2
                                    model: ["manual", "due", "created", "priority", "title"]
                                    currentIndex: Math.max(0, model.indexOf(root.sortMode))
                                    Accessible.name: "Reminder sort order"
                                    onActivated: {
                                        root.sortMode = currentText;
                                        if (root.viewMode === "list")
                                            root.saveSelectedPresentation();
                                    }
                                }

                                CheckBox {
                                    text: "Descending"
                                    checked: root.sortDescending
                                    onClicked: {
                                        root.sortDescending = checked;
                                        if (root.viewMode === "list")
                                            root.saveSelectedPresentation();
                                    }
                                }

                            }

                            Row {
                                width: parent.width
                                spacing: Style.space(8)

                                TextField {
                                    id: quickAdd

                                    width: parent.width * 0.66
                                    placeholderText: root.addDestination ? "Add to " + root.addDestination.title : "Create a list first"
                                    Accessible.name: "Quick reminder title and tags"
                                    enabled: root.addDestination && service && service.canEditList(root.addDestination.id) && service.connectionState === "online" && !service.mutationPending
                                    onAccepted: {
                                        var parsed = root.parseQuickEntry(text);
                                        if (parsed.title && service.addReminder(root.addDestination.id, parsed.title, root.addDestination.revision, parsed.tags))
                                            text = "";

                                    }
                                }

                                TextField {
                                    id: newSection

                                    width: parent.width - quickAdd.width - parent.spacing
                                    placeholderText: "Add section"
                                    Accessible.name: "New section title"
                                    visible: root.viewMode === "list"
                                    enabled: visible && root.selectedListEditable && root.selectedList && service && service.connectionState === "online" && !service.mutationPending
                                    onAccepted: {
                                        if (!text.trim())
                                            return ;

                                        var sections = root.selectedList.sections;
                                        var rank = sections.length ? sections[sections.length - 1].rank + 1024 : 1024;
                                        if (service.addSection(root.selectedList.id, text, rank, root.selectedList.revision))
                                            text = "";

                                    }
                                }

                            }

                            Row {
                                visible: root.viewMode === "list" && root.selectedList !== null && root.selectedList.sections.length > 0
                                width: parent.width
                                spacing: Style.space(6)

                                ComboBox {
                                    id: sectionChooser

                                    width: parent.width * 0.24
                                    model: root.selectedList ? root.selectedList.sections : []
                                    textRole: "title"
                                    currentIndex: root.indexForId(root.selectedList ? root.selectedList.sections : [], root.selectedSectionId)
                                    Accessible.name: "Section to edit"
                                    onActivated: function(index) {
                                        root.chooseSection(index);
                                    }
                                }

                                TextField {
                                    id: sectionTitleEditor

                                    width: parent.width * 0.32
                                    placeholderText: "Section title"
                                    Accessible.name: "Section title"
                                }

                                Button {
                                    text: "Up"
                                    enabled: root.selectedListEditable && root.selectedSectionId !== 0 && sectionChooser.currentIndex > 0 && service && service.connectionState === "online" && !service.mutationPending
                                    Accessible.name: "Move selected section up"
                                    onClicked: root.moveSelectedSectionRelative(-1)
                                }

                                Button {
                                    text: "Down"
                                    enabled: root.selectedListEditable && root.selectedSectionId !== 0 && sectionChooser.currentIndex >= 0 && sectionChooser.currentIndex < sectionChooser.count - 1 && service && service.connectionState === "online" && !service.mutationPending
                                    Accessible.name: "Move selected section down"
                                    onClicked: root.moveSelectedSectionRelative(1)
                                }

                                Button {
                                    text: "Save section"
                                    enabled: root.selectedListEditable && root.selectedSectionId !== 0 && sectionTitleEditor.text.trim() && service && service.connectionState === "online" && !service.mutationPending
                                    onClicked: {
                                        var section = root.selectedSectionValue();
                                        if (section)
                                            service.updateSection(root.selectedList.id, root.selectedSectionId, sectionTitleEditor.text, section.rank, root.selectedList.revision);
                                    }
                                }

                                Button {
                                    text: "Delete"
                                    enabled: root.selectedListEditable && root.selectedSectionId !== 0 && service && service.connectionState === "online" && !service.mutationPending
                                    onClicked: {
                                        service.deleteSection(root.selectedList.id, root.selectedSectionId, root.selectedList.revision);
                                        root.selectedSectionId = 0;
                                        sectionTitleEditor.text = "";
                                    }
                                }

                            }

                            Row {
                                visible: root.viewMode === "list" && root.selectedList !== null && root.selectedReminderIds.length > 0
                                width: parent.width
                                spacing: Style.space(6)

                                Text {
                                    anchors.verticalCenter: parent.verticalCenter
                                    text: root.selectedReminderIds.length + " selected"
                                    color: Color.menu.text
                                    font.family: Style.font.menuFamily
                                    font.pixelSize: Style.font.caption
                                }

                                ComboBox {
                                    id: batchSection

                                    width: parent.width * 0.24
                                    model: root.sectionChoices
                                    textRole: "title"
                                    Accessible.name: "Batch destination section"
                                }

                                Button {
                                    text: "Move"
                                    enabled: root.selectedListEditable && service && service.connectionState === "online" && !service.mutationPending
                                    onClicked: {
                                        var sectionId = batchSection.currentIndex > 0 ? root.sectionChoices[batchSection.currentIndex].id : null;
                                        if (service.batchMoveReminders(root.selectedList.id, root.selectedReminderIds, sectionId, root.batchStartingRank(), root.selectedList.revision))
                                            root.selectedReminderIds = [];
                                    }
                                }

                                Button {
                                    text: "Complete"
                                    enabled: root.selectedListEditable && service && service.connectionState === "online" && !service.mutationPending
                                    onClicked: {
                                        if (service.batchSetCompleted(root.selectedList.id, root.selectedReminderIds, true, root.selectedList.revision))
                                            root.selectedReminderIds = [];
                                    }
                                }

                                Button {
                                    text: "Uncomplete"
                                    enabled: root.selectedListEditable && service && service.connectionState === "online" && !service.mutationPending
                                    onClicked: {
                                        if (service.batchSetCompleted(root.selectedList.id, root.selectedReminderIds, false, root.selectedList.revision))
                                            root.selectedReminderIds = [];
                                    }
                                }

                                Button {
                                    text: root.confirmBatchDelete ? "Confirm delete" : "Delete"
                                    enabled: root.selectedListEditable && service && service.connectionState === "online" && !service.mutationPending
                                    onClicked: {
                                        if (!root.confirmBatchDelete) {
                                            root.confirmBatchDelete = true;
                                            return ;
                                        }
                                        if (service.batchDeleteReminders(root.selectedList.id, root.selectedReminderIds, root.selectedList.revision)) {
                                            root.selectedReminderIds = [];
                                            root.confirmBatchDelete = false;
                                        }
                                    }
                                }

                            }

                            Rectangle {
                                width: parent.width
                                height: root.selectedReminder ? Style.space(reminderRepeat.currentText === "none" ? 422 : 470) : 0
                                visible: root.selectedReminder !== null
                                radius: Style.cornerRadius
                                color: Qt.rgba(Color.menu.text.r, Color.menu.text.g, Color.menu.text.b, 0.05)

                                Column {
                                    anchors.fill: parent
                                    anchors.margins: Style.space(8)
                                    spacing: Style.space(6)

                                    TextField {
                                        id: reminderTitle

                                        width: parent.width
                                        placeholderText: "Reminder title"
                                        Accessible.name: "Reminder title"
                                    }

                                    TextArea {
                                        id: reminderNotes

                                        width: parent.width
                                        height: Style.space(54)
                                        placeholderText: "Notes"
                                        wrapMode: TextEdit.Wrap
                                        Accessible.name: "Reminder notes"
                                    }

                                    Row {
                                        width: parent.width
                                        spacing: Style.space(6)

                                        TextField {
                                            id: reminderUrl

                                            width: parent.width * 0.42
                                            placeholderText: "https://…"
                                            Accessible.name: "Reminder URL"
                                        }

                                        Button {
                                            text: "Open link"
                                            enabled: TendModel.safeExternalUrl(reminderUrl.text)
                                            Accessible.name: "Open reminder link in the default application"
                                            onClicked: Qt.openUrlExternally(reminderUrl.text.trim())
                                        }

                                        ComboBox {
                                            id: reminderPriority

                                            width: parent.width * 0.2
                                            model: ["none", "low", "medium", "high"]
                                            Accessible.name: "Reminder priority"
                                        }

                                        CheckBox {
                                            id: reminderFlagged

                                            text: "Flag"
                                        }

                                    }

                                    TextField {
                                        id: reminderTags

                                        width: parent.width
                                        placeholderText: "tags, separated, by commas"
                                        Accessible.name: "Reminder tags"
                                    }

                                    ComboBox {
                                        id: reminderAssignee

                                        width: parent.width
                                        model: root.assigneeChoices
                                        textRole: "title"
                                        valueRole: "ship"
                                        Accessible.name: "Reminder assignee ship"
                                    }

                                    Row {
                                        width: parent.width
                                        spacing: Style.space(6)

                                        TextField {
                                            id: reminderDue

                                            width: parent.width * 0.34
                                            placeholderText: "2026-09-10T17:30"
                                            Accessible.name: "Reminder due date and time"
                                        }

                                        TextField {
                                            id: reminderTimezone

                                            width: parent.width * 0.28
                                            placeholderText: "America/Los_Angeles"
                                            Accessible.name: "Reminder time zone"
                                        }

                                        CheckBox {
                                            id: reminderAllDay

                                            text: "All day"
                                        }

                                        TextField {
                                            id: reminderEarly

                                            width: parent.width - x
                                            placeholderText: "Early min"
                                            Accessible.name: "Early reminder minutes"
                                        }

                                    }

                                    Row {
                                        width: parent.width
                                        spacing: Style.space(6)

                                        ComboBox {
                                            id: reminderParent

                                            width: parent.width * 0.3
                                            model: root.parentChoices
                                            textRole: "title"
                                            Accessible.name: "Parent reminder"
                                        }

                                        ComboBox {
                                            id: reminderSection

                                            width: parent.width * 0.3
                                            model: root.sectionChoices
                                            textRole: "title"
                                            Accessible.name: "Reminder section"
                                        }

                                        SpinBox {
                                            id: reminderRank

                                            from: 0
                                            to: 2147483647
                                            value: 0
                                            editable: true
                                            Accessible.name: "Reminder rank"
                                        }

                                        Button {
                                            text: "Move"
                                            enabled: root.selectedListEditable && root.selectedReminder && service && service.connectionState === "online" && !service.mutationPending
                                            onClicked: {
                                                var parentId = reminderParent.currentIndex > 0 ? root.parentChoices[reminderParent.currentIndex].id : null;
                                                var sectionId = reminderSection.currentIndex > 0 ? root.sectionChoices[reminderSection.currentIndex].id : null;
                                                service.moveReminder(root.selectedList.id, root.selectedReminder.id, parentId, sectionId, reminderRank.value, root.selectedList.revision);
                                            }
                                        }

                                        Button {
                                            text: "↑"
                                            Accessible.name: "Move reminder up"
                                            enabled: root.selectedListEditable && root.selectedReminder && service && service.connectionState === "online" && !service.mutationPending
                                            onClicked: root.moveSelectedRelative(-1)
                                        }

                                        Button {
                                            text: "↓"
                                            Accessible.name: "Move reminder down"
                                            enabled: root.selectedListEditable && root.selectedReminder && service && service.connectionState === "online" && !service.mutationPending
                                            onClicked: root.moveSelectedRelative(1)
                                        }

                                    }

                                    Row {
                                        spacing: Style.space(8)

                                        Button {
                                            text: "Indent"
                                            enabled: root.selectedListEditable && root.selectedReminder && service && service.connectionState === "online" && !service.mutationPending
                                            onClicked: root.indentSelectedReminder()
                                        }

                                        Button {
                                            text: "Outdent"
                                            enabled: root.selectedListEditable && root.selectedReminder && root.selectedReminder.parentId !== null && service && service.connectionState === "online" && !service.mutationPending
                                            onClicked: root.outdentSelectedReminder()
                                        }

                                        Text {
                                            anchors.verticalCenter: parent.verticalCenter
                                            text: "Ctrl+] / Ctrl+["
                                            color: Color.menu.text
                                            opacity: 0.7
                                            font.family: Style.font.menuFamily
                                            font.pixelSize: Style.font.caption
                                        }

                                    }

                                    Row {
                                        visible: reminderRepeat.currentText !== "none"
                                        width: parent.width
                                        spacing: Style.space(5)

                                        TextField {
                                            id: reminderWeekdays

                                            width: parent.width * 0.15
                                            placeholderText: "Weekdays 0-6"
                                            Accessible.name: "Repeat weekdays from zero through six"
                                        }

                                        TextField {
                                            id: reminderMonthDays

                                            width: parent.width * 0.15
                                            placeholderText: "Month days"
                                            Accessible.name: "Repeat month days"
                                        }

                                        CheckBox {
                                            id: reminderOrdinal

                                            text: "Ordinal"
                                        }

                                        SpinBox {
                                            id: reminderOrdinalIndex

                                            from: 1
                                            to: 5
                                            value: 1
                                            enabled: reminderOrdinal.checked
                                            Accessible.name: "Ordinal week index"
                                        }

                                        SpinBox {
                                            id: reminderOrdinalWeekday

                                            from: 0
                                            to: 6
                                            value: 0
                                            enabled: reminderOrdinal.checked
                                            Accessible.name: "Ordinal weekday"
                                        }

                                        TextField {
                                            id: reminderRepeatEnd

                                            width: parent.width * 0.2
                                            placeholderText: "End date/time"
                                            Accessible.name: "Repeat end date and time"
                                        }

                                        SpinBox {
                                            id: reminderRepeatCount

                                            from: 0
                                            to: 9999
                                            value: 0
                                            editable: true
                                            Accessible.name: "Maximum repeat occurrences"
                                        }

                                    }

                                    Row {
                                        width: parent.width
                                        spacing: Style.space(6)

                                        ComboBox {
                                            id: reminderRepeat

                                            width: parent.width * 0.25
                                            model: ["none", "hourly", "daily", "weekly", "monthly", "yearly"]
                                            Accessible.name: "Repeat frequency"
                                        }

                                        SpinBox {
                                            id: reminderInterval

                                            from: 1
                                            to: 999
                                            value: 1
                                            editable: true
                                            Accessible.name: "Repeat interval"
                                        }

                                        Button {
                                            text: "Save schedule"
                                            enabled: root.selectedListEditable && root.selectedReminder && reminderDue.text.trim() && reminderTimezone.text.trim() && service && service.connectionState === "online" && !service.mutationPending
                                            onClicked: {
                                                var early = reminderEarly.text.split(",").map(function(minutes) {
                                                    return Math.round(Number(minutes.trim()) * 60);
                                                }).filter(function(seconds) {
                                                    return isFinite(seconds) && seconds >= 0;
                                                });
                                                var weekdays = reminderWeekdays.text.split(",").map(function(day) {
                                                    return Number(day.trim());
                                                }).filter(function(day) {
                                                    return isFinite(day) && day >= 0 && day <= 6;
                                                });
                                                var monthDays = reminderMonthDays.text.split(",").map(function(day) {
                                                    return Number(day.trim());
                                                }).filter(function(day) {
                                                    return isFinite(day) && day >= 1 && day <= 31;
                                                });
                                                var recurrence = reminderRepeat.currentText === "none" ? null : {
                                                    frequency: reminderRepeat.currentText,
                                                    interval: reminderInterval.value,
                                                    weekdays: weekdays,
                                                    monthDays: monthDays,
                                                    monthWeek: reminderOrdinal.checked ? {
                                                        index: reminderOrdinalIndex.value,
                                                        weekday: reminderOrdinalWeekday.value
                                                    } : null,
                                                    endAt: reminderRepeatEnd.text.trim() || null,
                                                    maxOccurrences: reminderRepeatCount.value > 0 ? reminderRepeatCount.value : null
                                                };
                                                service.setSchedule(root.selectedList.id, root.selectedReminder.id, {
                                                    dueAt: reminderDue.text.trim(),
                                                    allDay: reminderAllDay.checked,
                                                    timezone: reminderTimezone.text.trim(),
                                                    earlySeconds: early,
                                                    recurrence: recurrence
                                                }, root.selectedList.revision);
                                            }
                                        }

                                        Button {
                                            text: "Clear"
                                            enabled: root.selectedListEditable && root.selectedReminder && root.selectedReminder.schedule && service && service.connectionState === "online" && !service.mutationPending
                                            onClicked: service.setSchedule(root.selectedList.id, root.selectedReminder.id, null, root.selectedList.revision)
                                        }

                                    }

                                    Row {
                                        spacing: Style.space(8)

                                        Button {
                                            text: service && service.mutationPending ? "Saving…" : "Save"
                                            enabled: root.selectedListEditable && root.selectedReminder && reminderTitle.text.trim() && service && service.connectionState === "online" && !service.mutationPending
                                            onClicked: {
                                                var tags = reminderTags.text.split(",").map(function(tag) {
                                                    return tag.trim();
                                                }).filter(function(tag) {
                                                    return tag !== "";
                                                });
                                                service.updateReminder(root.selectedList.id, root.selectedReminder.id, {
                                                    title: reminderTitle.text,
                                                    notes: reminderNotes.text,
                                                    url: reminderUrl.text.trim() || null,
                                                    priority: reminderPriority.currentText,
                                                    flagged: reminderFlagged.checked,
                                                    tags: tags,
                                                    assignee: reminderAssignee.currentValue || null
                                                }, root.selectedList.revision);
                                            }
                                        }

                                        Button {
                                            text: "Delete"
                                            enabled: root.selectedListEditable && root.selectedReminder && service && service.connectionState === "online" && !service.mutationPending
                                            onClicked: service.deleteReminder(root.selectedList.id, root.selectedReminder.id, root.selectedList.revision)
                                        }

                                        ComboBox {
                                            id: snoozePreset

                                            model: service ? service.preferences.snoozePresets : []
                                            displayText: currentValue ? Math.round(Number(currentValue) / 60) + " min" : "Snooze"
                                            Accessible.name: "Snooze duration"
                                        }

                                        Button {
                                            text: "Snooze"
                                            enabled: root.selectedReminder && root.selectedReminder.schedule && snoozePreset.currentIndex >= 0 && service && service.connectionState === "online" && !service.mutationPending
                                            onClicked: service.snoozeReminder(root.selectedList.id, root.selectedReminder.id, Number(snoozePreset.currentValue))
                                        }

                                    }

                                    Row {
                                        spacing: Style.space(8)

                                        TextField {
                                            id: customSnooze

                                            width: Style.space(180)
                                            placeholderText: "Snooze until YYYY-MM-DDTHH:MM"
                                            Accessible.name: "Custom snooze date and time"
                                        }

                                        Button {
                                            text: "Snooze until"
                                            enabled: root.selectedReminder && root.selectedReminder.schedule && customSnooze.text.trim() && service && service.connectionState === "online" && !service.mutationPending
                                            onClicked: service.snoozeReminderUntil(root.selectedList.id, root.selectedReminder.id, customSnooze.text.trim())
                                        }

                                    }

                                }

                            }

                            ListView {
                                id: reminderList

                                width: parent.width
                                height: parent.height - y
                                clip: true
                                spacing: Style.space(4)
                                model: root.displayedReminders
                                activeFocusOnTab: true
                                keyNavigationEnabled: true
                                Accessible.role: Accessible.List
                                Accessible.name: "Reminders"
                                Keys.onReturnPressed: {
                                    if (currentIndex >= 0 && currentIndex < root.displayedReminders.length)
                                        root.editReminder(root.displayedReminders[currentIndex]);
                                }
                                Keys.onSpacePressed: {
                                    if (currentIndex < 0 || currentIndex >= root.displayedReminders.length || !service)
                                        return ;
                                    var reminder = root.displayedReminders[currentIndex];
                                    if (service.canEditList(reminder.listId) && service.connectionState === "online" && !service.mutationPending)
                                        service.setCompleted(reminder.listId, reminder.id, !reminder.completed, reminder.listRevision);
                                }
                                Keys.onRightPressed: {
                                    if (currentIndex < 0 || currentIndex >= root.displayedReminders.length)
                                        return;
                                    var reminder = root.displayedReminders[currentIndex];
                                    if (root.reminderHasChildren(reminder.id))
                                        root.setReminderCollapsed(reminder.id, false);
                                }
                                Keys.onLeftPressed: {
                                    if (currentIndex < 0 || currentIndex >= root.displayedReminders.length)
                                        return;
                                    var reminder = root.displayedReminders[currentIndex];
                                    if (root.reminderHasChildren(reminder.id))
                                        root.setReminderCollapsed(reminder.id, true);
                                }

                                delegate: Rectangle {
                                    id: reminderRow

                                    required property var modelData
                                    required property int index

                                    width: ListView.view.width
                                    height: Style.space(44)
                                    radius: Style.cornerRadius
                                    color: ListView.isCurrentItem ? Qt.rgba(Color.menu.text.r, Color.menu.text.g, Color.menu.text.b, 0.14) : Qt.rgba(Color.menu.text.r, Color.menu.text.g, Color.menu.text.b, 0.04)
                                    border.width: reminderDropArea.containsDrag ? Math.max(1, Style.space(1)) : 0
                                    border.color: reminderDropArea.containsDrag ? Color.menu.text : "transparent"
                                    Accessible.role: Accessible.ListItem
                                    Accessible.name: (Number(modelData.depth || 0) > 0 ? "Subtask, " : "") + modelData.title + (root.reminderHasChildren(modelData.id) ? (root.reminderIsCollapsed(modelData.id) ? ", collapsed" : ", expanded") : "")

                                    DropArea {
                                        id: reminderDropArea

                                        anchors.fill: parent
                                        enabled: root.viewMode === "list" && root.sortMode === "manual" && root.selectedListEditable && service && service.connectionState === "online" && !service.mutationPending
                                        keys: ["tend-reminder"]
                                        onDropped: function(drop) {
                                            if (drop.source && root.placeReminder(drop.source.modelData, modelData, drop.y >= height / 2))
                                                drop.acceptProposedAction();
                                        }
                                    }

                                    Row {
                                        anchors.fill: parent
                                        anchors.margins: Style.space(8)
                                        spacing: Style.space(10)

                                        CheckBox {
                                            checked: modelData.completed
                                            enabled: service && service.canEditList(modelData.listId) && service.connectionState === "online" && !service.mutationPending
                                            Accessible.name: (modelData.completed ? "Uncomplete " : "Complete ") + modelData.title
                                            onClicked: service.setCompleted(modelData.listId, modelData.id, checked, modelData.listRevision)
                                        }

                                        CheckBox {
                                            visible: root.viewMode === "list"
                                            checked: root.reminderIsSelected(modelData.id)
                                            enabled: visible && service && service.canEditList(modelData.listId) && service.connectionState === "online" && !service.mutationPending
                                            Accessible.name: "Select " + modelData.title + " for batch action"
                                            onClicked: root.toggleReminderSelection(modelData.id)
                                        }

                                        Rectangle {
                                            id: reminderDragHandle

                                            visible: root.viewMode === "list" && root.sortMode === "manual"
                                            width: visible ? Style.space(24) : 0
                                            height: Style.space(24)
                                            radius: Style.cornerRadius
                                            color: reminderDragArea.drag.active ? Qt.rgba(Color.menu.text.r, Color.menu.text.g, Color.menu.text.b, 0.2) : "transparent"
                                            z: reminderDragArea.drag.active ? 100 : 0
                                            Drag.active: reminderDragArea.drag.active
                                            Drag.source: reminderRow
                                            Drag.keys: ["tend-reminder"]
                                            Drag.supportedActions: Qt.MoveAction
                                            Drag.proposedAction: Qt.MoveAction
                                            Drag.hotSpot.x: width / 2
                                            Drag.hotSpot.y: height / 2
                                            Accessible.role: Accessible.Button
                                            Accessible.name: "Drag to reorder " + modelData.title

                                            Text {
                                                anchors.centerIn: parent
                                                text: "↕"
                                                color: Color.menu.text
                                                opacity: 0.7
                                                font.family: Style.font.menuFamily
                                                font.pixelSize: Style.font.body
                                            }

                                            MouseArea {
                                                id: reminderDragArea

                                                anchors.fill: parent
                                                enabled: reminderDragHandle.visible && service && service.canEditList(modelData.listId) && service.connectionState === "online" && !service.mutationPending
                                                cursorShape: drag.active ? Qt.ClosedHandCursor : Qt.OpenHandCursor
                                                drag.target: reminderDragHandle
                                                drag.axis: Drag.YAxis
                                                onPressed: reminderList.currentIndex = index
                                                onReleased: {
                                                    reminderDragHandle.Drag.drop();
                                                    reminderDragHandle.y = 0;
                                                }
                                                onCanceled: {
                                                    reminderDragHandle.Drag.cancel();
                                                    reminderDragHandle.y = 0;
                                                }
                                            }
                                        }

                                        Button {
                                            visible: root.viewMode === "list" && root.reminderHasChildren(modelData.id)
                                            text: root.reminderIsCollapsed(modelData.id) ? "▶" : "▼"
                                            Accessible.name: (root.reminderIsCollapsed(modelData.id) ? "Expand " : "Collapse ") + modelData.title
                                            onClicked: root.toggleReminderCollapsed(modelData.id)
                                        }

                                        Text {
                                            anchors.verticalCenter: parent.verticalCenter
                                            width: parent.width - x
                                            text: {
                                                var list = root.viewMode === "list" ? "" : "  ·  " + modelData.listTitle;
                                                var section = root.sectionTitleFor(modelData.listId, modelData.sectionId);
                                                var depth = Math.max(0, Number(modelData.depth || 0));
                                                var prefix = depth === 0 ? "" : Array(depth + 1).join("  ") + "↳ ";
                                                var due = modelData.schedule ? "  ·  " + TendModel.scheduleInputValue(modelData.schedule).replace("T", " ") : "";
                                                return prefix + modelData.title + list + (section ? "  ·  " + section : "") + due;
                                            }
                                            color: Color.menu.text
                                            opacity: modelData.completed ? 0.5 : 1
                                            font.family: Style.font.menuFamily
                                            font.pixelSize: Style.font.body
                                            font.strikeout: modelData.completed

                                            MouseArea {
                                                anchors.fill: parent
                                                cursorShape: Qt.PointingHandCursor
                                                onClicked: {
                                                    reminderList.currentIndex = index;
                                                    root.editReminder(modelData);
                                                }
                                            }
                                        }

                                    }

                                }

                            }

                        }

                    }

                }

            }

        }

    }

}
