import QtQuick
import QtQuick.Controls as QQC
import QtQuick.Layouts
import Quickshell
import Quickshell.Hyprland
import qs.Commons
import qs.Ui
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
    property string rightTrayMode: ""
    property bool selectionMode: false
    property bool unpinnedViewsVisible: false
    property bool unpinnedListsVisible: false
    property bool compactSidebarOpen: false
    property bool confirmDeleteList: false
    property bool captureMode: false
    property bool switchingShip: false
    property var hiddenInvitationKeys: []
    property var selectedReminderIds: []
    property var collapsedReminderIds: []
    property bool confirmBatchDelete: false
    property real sidebarWidth: Style.space(230)
    readonly property real detailTrayWidth: Math.min(Style.space(360), workspace ? workspace.width * 0.88 : Style.space(360))
    readonly property real settingsTrayWidth: Math.min(Style.space(390), workspace ? workspace.width * 0.9 : Style.space(390))
    readonly property bool singleTrayMode: window.width < Style.space(1200)
    readonly property bool overlayTrayMode: window.width < Style.space(980)
    readonly property bool detailTrayOpen: selectedReminder !== null
    readonly property bool settingsTrayOpen: rightTrayMode !== ""
    readonly property var visibleInvitations: service ? service.invitations.filter(function(invitation) {
        return root.hiddenInvitationKeys.indexOf(root.invitationKey(invitation)) === -1;
    }) : []
    readonly property color opaqueMenuBackground: Qt.rgba(Color.menu.background.r, Color.menu.background.g, Color.menu.background.b, 1)
    readonly property bool compactMode: window.width < Style.space(820)
    property var selectedList: null
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
    readonly property bool selectedListHasPendingOperation: service && selectedList ? service.listMutationPending(selectedList.id) : false
    readonly property bool selectedConnectionAvailable: service && service.connectionState === "online" && selectedAccess && (selectedAccess.owner || selectedAccess.status === "online")
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

    function invitationKey(invitation) {
        return [invitation && invitation.host, invitation && invitation.token, invitation && invitation.title].join(":");
    }

    function hideInvitation(invitation) {
        var keys = hiddenInvitationKeys.slice();
        var key = invitationKey(invitation);
        if (keys.indexOf(key) === -1)
            keys.push(key);
        hiddenInvitationKeys = keys;
    }

    function listAccess(listId) {
        return service ? service.accessForList(listId) : null;
    }

    function listHost(listId) {
        var access = listAccess(listId);
        if (access && access.host)
            return root.normalizedShip(access.host);
        return service && service.ship ? root.normalizedShip(service.ship) : "";
    }

    function listHostOnline(listId) {
        var access = listAccess(listId);
        return service && service.connectionState === "online" && access && (access.owner || access.status === "online");
    }

    function sameShip(left, right) {
        return root.normalizedShip(left).replace(/^~/, "") === root.normalizedShip(right).replace(/^~/, "");
    }

    function movableListChoicesForList(list) {
        if (!service || !list)
            return [];
        var sourceAccess = service.accessForList(list.id);
        if (!sourceAccess)
            return [];
        var sourceHost = sourceAccess.host || service.ship;
        return service.lists.filter(function(candidate) {
            var access = service.accessForList(candidate.id);
            return service.canEditList(candidate.id) && access && root.sameShip(access.host || service.ship, sourceHost);
        }).map(function(candidate) {
            return { id: candidate.id, title: candidate.title };
        });
    }
    readonly property var availableViews: ["today", "scheduled", "all", "flagged", "assigned", "completed"]
    readonly property var orderedViews: TendModel.orderedValues(availableViews, service ? service.preferences.pinnedViews : [])
    readonly property var availableTags: TendModel.allTags(service ? service.lists : [])
    property int quickDestinationId: 0
    readonly property var addDestination: {
        if (!service || !service.lists.length)
            return null;

        var wanted = quickDestinationId || (viewMode === "list" && selectedList ? selectedList.id : service.preferences.defaultList);
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
    readonly property var displayedRows: viewMode === "list" && selectedList
        ? TendModel.sectionedRows(displayedReminders, selectedList, true)
        : displayedReminders
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
    function normalizedShip(value) {
        var name = String(value || "").replace(/^~/, "");
        return name ? "~" + name : "";
    }

    function assigneeChoicesForList(list) {
        var choices = [{
            ship: "",
            label: "Unassigned",
            role: "",
            status: "unassigned"
        }];
        if (!service || !list)
            return choices;
        var access = service.accessForList(list.id);
        if (!access)
            return choices;
        var seen = {};
        function append(ship, status) {
            var normalized = root.normalizedShip(ship);
            var key = normalized.replace(/^~/, "");
            if (!key || seen[key])
                return;
            seen[key] = true;
            var self = key === String(service.ship || "").replace(/^~/, "");
            choices.push({
                ship: normalized,
                label: self ? "me" : normalized,
                role: status,
                status: status || "member"
            });
        }
        append(access.host, "admin");
        (access.members || []).forEach(function(member) { append(member.ship, member.policy && member.policy.canInvite ? "admin" : "editor"); });
        (access.pending || []).forEach(function(ship) { append(ship, "invited"); });
        if (!seen[String(service.ship || "").replace(/^~/, "")]) append(service.ship, access.owner ? "admin" : "editor");
        return choices;
    }
    readonly property var assigneeChoices: assigneeChoicesForList(selectedList)
    readonly property var addAssigneeChoices: assigneeChoicesForList(addDestination)
    readonly property var movableListChoices: movableListChoicesForList(selectedList)
    readonly property var addAssigneeOptions: addAssigneeChoices.map(function(choice) {
        return { value: choice.ship, label: choice.label, role: choice.role };
    })
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

    function selectSelfForQuickAdd() {
        if (service)
            quickAssignee.value = normalizedShip(service.ship);
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

    onSelectedListIdChanged: {
        ensureSelectedList();
        loadSelectedPresentation();
    }
    onServiceChanged: Qt.callLater(root.ensureSelectedList)

    Connections {
        target: root.service
        function onLocalSettingsChanged() { root.loadSelectedPresentation(); }
        function onListsChanged() { Qt.callLater(root.ensureSelectedList); }
        function onLoginSucceeded() { loginCode.text = ""; }
        function onConnectionStateChanged() {
            if (root.service && root.service.connectionState === "online")
                root.switchingShip = false;
        }
    }

    function movePinnedView(view, delta) {
        if (!service)
            return;
        var pins = movePinnedValue(service.preferences.pinnedViews, view, delta);
        if (pins !== service.preferences.pinnedViews)
            updatePreferences(service.preferences.defaultList, service.preferences.pinnedLists, pins);
    }

    function ensureSelectedList() {
        if (!service || !service.lists.length) {
            selectedListId = 0;
            selectedList = null;
            return;
        }
        var next = null;
        for (var i = 0; i < service.lists.length; i++) {
            if (service.lists[i].id === selectedListId) {
                next = service.lists[i];
                break;
            }
        }
        if (!next) {
            next = service.lists[0];
            selectedListId = next.id;
        }
        selectedList = next;
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
        policyEditorOpen = false;
        rightTrayMode = "";
    }

    function openListEditor() {
        if (!selectedList)
            return ;
        listEditorOpen = false;
        policyEditorOpen = false;
        rightTrayMode = rightTrayMode === "list" ? "" : "list";
        if (rightTrayMode)
            selectedReminderId = 0;
    }

    function openAppSettings() {
        listEditorOpen = false;
        policyEditorOpen = false;
        rightTrayMode = rightTrayMode === "app" ? "" : "app";
        if (rightTrayMode)
            selectedReminderId = 0;
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

    function moveReminderRelative(reminder, delta) {
        if (!selectedList || !reminder || !service)
            return;

        var siblings = selectedList.reminders.filter(function(item) {
            return item.parentId === reminder.parentId && item.sectionId === reminder.sectionId;
        }).sort(function(a, b) {
            return a.rank - b.rank || a.id - b.id;
        });
        var index = siblings.findIndex(function(item) {
            return Number(item.id) === Number(reminder.id);
        });
        var target = index + delta;
        if (index < 0 || target < 0 || target >= siblings.length)
            return;

        service.placeReminder(selectedList.id, reminder.id, siblings[target].id, delta > 0, selectedList.revision);
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

    function selectedBatchReminder() {
        return selectedReminderIds.length === 1 ? reminderFromSelectedList(selectedReminderIds[0]) : null;
    }

    function indentReminder(reminder) {
        if (!selectedListEditable || !reminder || !service)
            return;
        var index = displayedReminders.findIndex(function(item) { return Number(item.id) === Number(reminder.id); });
        if (index <= 0)
            return;
        var parent = displayedReminders[index - 1];
        if (parent.listId !== selectedList.id || Number(parent.id) === Number(reminder.id))
            return;
        service.moveReminder(selectedList.id, reminder.id, parent.id, parent.sectionId, reminder.rank, selectedList.revision);
    }

    function outdentReminder(reminder) {
        if (!selectedListEditable || !reminder || reminder.parentId === null || !service)
            return;
        var parent = reminderFromSelectedList(reminder.parentId);
        if (!parent)
            return;
        service.moveReminder(selectedList.id, reminder.id, parent.parentId, parent.sectionId, reminder.rank, selectedList.revision);
    }

    function indentSelectedReminder() {
        indentReminder(selectedReminder);
    }

    function outdentSelectedReminder() {
        outdentReminder(selectedReminder);
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
            root.selectSelfForQuickAdd();
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
            Qt.callLater(function() {
                root.focusOpenTarget(String(payload.focus || ""));
            });
        });
    }

    function focusOpenTarget(target) {
        if (!service || !service.ship) {
            shipUrl.forceActiveFocus();
            return ;
        }
        if (target === "new-list") {
            newList.forceActiveFocus();
        } else if (target === "invite") {
            root.openListEditor();
            Qt.callLater(function() {
                inviteShip.forceActiveFocus();
            });
        } else if (target === "reminder-assignee") {
            detailSidebar.focusAssignee();
        } else if (target === "reminder-save") {
            detailSidebar.focusSave();
        } else if (root.captureMode) {
            captureEntry.forceActiveFocus();
        } else {
            quickAdd.forceActiveFocus();
        }
    }

    function close() {
        root.opened = false;
        root.captureMode = false;
    }

    function diagnostics(unused) {
        return JSON.stringify({
            connectionState: service ? service.connectionState : "unavailable",
            ship: service ? service.ship : "",
            baseUrl: service ? service.baseUrl : "",
            bridgeReady: service ? service.bridgePath !== "" : false,
            error: service ? service.errorMessage : "Tend service unavailable"
        });
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

    onSelectedListChanged: {
        if (selectedList) {
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

    onAddDestinationChanged: Qt.callLater(root.selectSelfForQuickAdd)

    FloatingWindow {
        id: window

        title: service && service.ship ? "Tend · ~" + service.ship : "Tend"
        visible: root.opened
        color: root.opaqueMenuBackground
        implicitWidth: Style.space(1180)
        implicitHeight: Style.space(780)
        minimumSize: Qt.size(Style.space(840), Style.space(600))

        onVisibleChanged: {
            if (!visible && root.opened) {
                root.opened = false;
                root.captureMode = false;
                if (root.shell && typeof root.shell.hide === "function")
                    root.shell.hide("io.omabit.tend");
            }
        }

        Rectangle {
            anchors.fill: parent
            color: root.opaqueMenuBackground

            MouseArea {
                anchors.fill: parent
                enabled: false
            }

            Rectangle {
                id: card

                anchors.fill: parent
                radius: 0
                color: root.opaqueMenuBackground
                border.width: 0

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

                        Row {
                            width: parent.width - closeButton.width - settingsButton.width - Style.space(8)
                            spacing: Style.space(8)

                            Text {
                                anchors.verticalCenter: parent.verticalCenter
                                text: service && service.ship ? "Tend · ~" + service.ship : "Tend"
                                color: Color.menu.text
                                font.family: Style.font.menuFamily
                                font.pixelSize: Style.font.heading
                                font.bold: true
                            }

                            Rectangle {
                                anchors.verticalCenter: parent.verticalCenter
                                width: Style.space(9)
                                height: width
                                radius: width / 2
                                color: service && service.connectionState === "online" ? "#22c55e" : "#ef4444"
                                Accessible.role: Accessible.Indicator
                                Accessible.name: service && service.connectionState === "online" ? "Connected to Tend" : "Tend connection unavailable"
                            }
                        }

                        TendButton {
                            id: settingsButton
                            visible: service && service.ship !== ""
                            text: "󰒓"
                            tooltipText: "Tend settings"
                            Accessible.name: "Open Tend settings"
                            onClicked: root.openAppSettings()
                        }

                        TendButton {
                            id: closeButton

                            text: "󰅖"
                            tooltipText: "Close Tend"
                            Accessible.name: "Close Tend"
                            bordered: false
                            onClicked: root.dismiss()
                        }

                    }

                    TextEdit {
                        width: parent.width
                        visible: service && service.errorMessage !== ""
                        text: service ? service.errorMessage : ""
                        readOnly: true
                        selectByMouse: true
                        color: "#ef4444"
                        wrapMode: Text.Wrap
                        textFormat: TextEdit.PlainText
                        font.family: Style.font.menuFamily
                        font.pixelSize: Style.font.body
                        Accessible.name: "Tend error message; selectable for copying"
                    }

                    BorderSurface {
                        visible: service && service.ship !== "" && !root.captureMode && root.visibleInvitations.length > 0
                        width: parent.width
                        height: visible ? invitationColumn.implicitHeight + Style.space(20) : 0
                        color: Qt.rgba(Color.accent.r, Color.accent.g, Color.accent.b, 0.09)
                        borderSpec: Border.controlSpec("normal", Color.menu.text, Color.accent)
                        radius: Style.cornerRadius

                        Column {
                            id: invitationColumn
                            anchors.left: parent.left
                            anchors.right: parent.right
                            anchors.verticalCenter: parent.verticalCenter
                            anchors.margins: Style.space(10)
                            spacing: Style.space(6)

                        Text {
                            text: "Shared-list invitations"
                            color: Color.menu.text
                            font.family: Style.font.menuFamily
                            font.pixelSize: Style.font.caption
                            font.bold: true
                        }

                        Repeater {
                            model: root.visibleInvitations

                            delegate: Row {
                                required property var modelData

                                width: parent.width
                                spacing: Style.space(6)

                                Text {
                                    width: parent.width - acceptInvitationButton.width - declineInvitationButton.width - hideInvitationButton.width - parent.spacing * 3
                                    anchors.verticalCenter: parent.verticalCenter
                                    text: modelData.title + " from " + modelData.host + (modelData.canInvite ? " · may invite" : "")
                                    elide: Text.ElideRight
                                    color: Color.menu.text
                                    font.family: Style.font.menuFamily
                                    font.pixelSize: Style.font.body
                                }

                                TendButton {
                                    id: acceptInvitationButton

                                    text: "Accept"
                                    enabled: service && service.connectionState === "online" && !service.mutationPending
                                    Accessible.name: "Accept invitation to " + modelData.title
                                    onClicked: service.acceptInvitation(modelData)
                                }

                                TendButton {
                                    id: declineInvitationButton

                                    text: "Decline"
                                    enabled: service && service.connectionState === "online" && !service.mutationPending
                                    Accessible.name: "Decline invitation to " + modelData.title
                                    onClicked: service.declineInvitation(modelData)
                                }

                                TendButton {
                                    id: hideInvitationButton

                                    text: "Hide"
                                    bordered: false
                                    Accessible.name: "Hide invitation to " + modelData.title
                                    onClicked: root.hideInvitation(modelData)
                                }
                            }
                        }
                        }
                    }

                    Column {
                        visible: root.switchingShip || !service || !service.ship || service.connectionState === "authentication-required"
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
                            text: root.switchingShip ? "Connect another Urbit. The current ship remains saved unless you explicitly remove it from Settings." : (service && service.connectionState === "authentication-required" ? "Your Eyre session expired. Enter the current +code to reconnect to this ship." : "Enter your ship domain and the current code printed by +code. Tend accepts planets, moons, and comets.")
                            color: Color.menu.text
                            opacity: 0.72
                            wrapMode: Text.Wrap
                            font.family: Style.font.menuFamily
                            font.pixelSize: Style.font.body
                        }

                        TendButton {
                            visible: root.switchingShip && service && service.ship !== ""
                            width: parent.width
                            text: "Cancel switching"
                            bordered: false
                            onClicked: root.switchingShip = false
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

                        TendButton {
                            id: connectButton

                            text: service && service.connectionState === "authenticating" ? "Connecting…" : "Connect"
                            enabled: service && service.connectionState !== "authenticating" && shipUrl.text.trim() !== "" && loginCode.text.trim() !== ""
                            onClicked: {
                                service.login(shipUrl.text, loginCode.text);
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

                            TendDropdown {
                                id: captureList

                                width: captureControls.columns === 1 ? parent.width : parent.width - captureEntry.width - captureAdd.width - captureControls.columnSpacing * 2
                                model: service ? TendModel.orderedLists(service.lists, service.preferences.pinnedLists, service.localSettings.listOrder) : []
                                textRole: "title"
                                Accessible.name: "Destination list"
                            }

                            TendButton {
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
                        spacing: 0

                        Rectangle {
                            id: sidebar

                            visible: !root.compactMode || root.compactSidebarOpen
                            width: visible ? root.sidebarWidth : 0
                            height: parent.height
                            radius: Style.cornerRadius
                            color: Qt.rgba(Color.menu.text.r, Color.menu.text.g, Color.menu.text.b, 0.05)

                            Flickable {
                                anchors.fill: parent
                                anchors.margins: Style.space(10)
                                contentWidth: width
                                contentHeight: sidebarContent.implicitHeight
                                clip: true
                                boundsBehavior: Flickable.StopAtBounds

                                Column {
                                    id: sidebarContent
                                    width: parent.width
                                    spacing: Style.space(6)

                                Row {
                                    width: parent.width
                                    height: Style.space(28)

                                    Text {
                                        width: parent.width - viewsReveal.width
                                        anchors.verticalCenter: parent.verticalCenter
                                        text: "VIEWS"
                                        color: Color.menu.text
                                        opacity: 0.68
                                        font.family: Style.font.menuFamily
                                        font.pixelSize: Style.font.caption
                                        font.bold: true
                                    }

                                    TendButton {
                                        id: viewsReveal
                                        text: root.unpinnedViewsVisible ? "Hide" : "Show all"
                                        bordered: false
                                        onClicked: root.unpinnedViewsVisible = !root.unpinnedViewsVisible
                                    }
                                }

                                Repeater {
                                    model: root.orderedViews

                                    delegate: Row {
                                        required property var modelData
                                        readonly property bool pinned: service && service.preferences.pinnedViews.indexOf(modelData) !== -1

                                        visible: pinned || root.unpinnedViewsVisible
                                        width: parent.width
                                        height: visible ? Style.space(34) : 0
                                        spacing: Style.space(4)

                                        TendButton {
                                            width: parent.width - viewPin.width - parent.spacing
                                            height: parent.height
                                            text: modelData.charAt(0).toUpperCase() + modelData.slice(1)
                                            leftAlign: true
                                            bordered: false
                                            checkable: true
                                            checked: root.viewMode === modelData
                                            onClicked: {
                                                root.viewMode = modelData;
                                                root.selectedReminderId = 0;
                                            }
                                        }

                                        TendButton {
                                            id: viewPin
                                            width: Style.space(34)
                                            height: parent.height
                                            text: parent.pinned ? "󰐃" : "󰐄"
                                            tooltipText: parent.pinned ? "Unpin view" : "Pin view"
                                            bordered: false
                                            enabled: service && service.connectionState === "online" && !service.mutationPending
                                            onClicked: root.toggleViewPin(modelData)
                                        }
                                    }
                                }

                                PanelSeparator { width: parent.width }

                                Row {
                                    width: parent.width
                                    height: Style.space(28)

                                    Text {
                                        width: parent.width - listsReveal.width
                                        anchors.verticalCenter: parent.verticalCenter
                                        text: "LISTS"
                                        color: Color.menu.text
                                        opacity: 0.68
                                        font.family: Style.font.menuFamily
                                        font.pixelSize: Style.font.caption
                                        font.bold: true
                                    }

                                    TendButton {
                                        id: listsReveal
                                        text: root.unpinnedListsVisible ? "Hide" : "Show all"
                                        bordered: false
                                        onClicked: root.unpinnedListsVisible = !root.unpinnedListsVisible
                                    }
                                }

                                Repeater {
                                    model: root.orderedLists

                                    delegate: Row {
                                        id: listRow
                                        required property var modelData
                                        readonly property bool pinned: service && service.preferences.pinnedLists.indexOf(modelData.id) !== -1
                                        readonly property var access: root.listAccess(modelData.id)

                                        visible: pinned || root.unpinnedListsVisible || (root.selectedList && root.selectedList.id === modelData.id)
                                        width: parent.width
                                        height: visible ? Style.space(46) : 0
                                        spacing: Style.space(4)

                                        TendButton {
                                            width: parent.width - listPin.width - parent.spacing
                                            height: parent.height
                                            text: ""
                                            leftAlign: true
                                            bordered: false
                                            checkable: true
                                            clip: true
                                            checked: root.viewMode === "list" && root.selectedList && root.selectedList.id === modelData.id
                                            Accessible.name: modelData.title + (service && service.preferences.defaultList === modelData.id ? ", default list" : "")

                                            Row {
                                                anchors.left: parent.left
                                                anchors.right: parent.right
                                                anchors.verticalCenter: parent.verticalCenter
                                                anchors.leftMargin: Style.spacing.controlPaddingX
                                                anchors.rightMargin: Style.spacing.controlPaddingX
                                                spacing: Style.space(7)

                                                Rectangle {
                                                    width: Style.space(24)
                                                    height: width
                                                    radius: width / 2
                                                    color: listRow.modelData.color || "#3b82f6"
                                                    Text { anchors.centerIn: parent; text: listRow.modelData.symbol && listRow.modelData.symbol !== "list" ? listRow.modelData.symbol : "≡"; color: "white"; font.family: Style.font.menuFamily; font.pixelSize: Style.font.caption; font.bold: true }
                                                }

                                                Column {
                                                    width: parent.width - x - hostDot.width - parent.spacing
                                                    anchors.verticalCenter: parent.verticalCenter
                                                    spacing: 0
                                                    Text { width: parent.width; text: listRow.modelData.title + (service && service.preferences.defaultList === listRow.modelData.id ? " · default" : ""); textFormat: Text.PlainText; elide: Text.ElideRight; color: Color.menu.text; font.family: Style.font.menuFamily; font.pixelSize: Style.font.body }
                                                    Text { width: parent.width; text: root.listHost(listRow.modelData.id); textFormat: Text.PlainText; elide: Text.ElideRight; color: Color.menu.text; opacity: 0.58; font.family: Style.font.menuFamily; font.pixelSize: Style.font.caption }
                                                }

                                                Rectangle {
                                                    id: hostDot
                                                    anchors.verticalCenter: parent.verticalCenter
                                                    width: Style.space(7)
                                                    height: width
                                                    radius: width / 2
                                                    color: root.listHostOnline(listRow.modelData.id) ? "#22c55e" : "#ef4444"
                                                }
                                            }

                                            onClicked: {
                                                root.viewMode = "list";
                                                root.selectedListId = modelData.id;
                                                root.selectedReminderId = 0;
                                            }
                                        }

                                        TendButton {
                                            id: listPin
                                            width: Style.space(34)
                                            height: parent.height
                                            text: parent.pinned ? "󰐃" : "󰐄"
                                            tooltipText: parent.pinned ? "Unpin list" : "Pin list"
                                            bordered: false
                                            enabled: service && service.connectionState === "online" && !service.mutationPending
                                            onClicked: {
                                                root.selectedListId = modelData.id;
                                                root.toggleListPin();
                                            }
                                        }
                                    }
                                }

                                Rectangle {
                                    width: parent.width
                                    height: addListColumn.implicitHeight + Style.space(16)
                                    radius: Style.cornerRadius
                                    color: Qt.rgba(Color.menu.text.r, Color.menu.text.g, Color.menu.text.b, 0.05)

                                    Column {
                                        id: addListColumn
                                        anchors.left: parent.left
                                        anchors.right: parent.right
                                        anchors.verticalCenter: parent.verticalCenter
                                        anchors.margins: Style.space(8)
                                        spacing: Style.space(6)

                                        Text {
                                            text: "NEW LIST"
                                            color: Color.menu.text
                                            opacity: 0.68
                                            font.family: Style.font.menuFamily
                                            font.pixelSize: Style.font.caption
                                            font.bold: true
                                        }

                                        TextField {
                                            id: newList
                                            width: parent.width
                                            placeholderText: "List name"
                                            Accessible.name: "New list title"
                                            enabled: service && service.connectionState === "online" && !service.mutationPending
                                            onAccepted: addListButton.clicked()
                                        }

                                        TendButton {
                                            id: addListButton
                                            width: parent.width
                                            text: "Add list"
                                            enabled: newList.enabled && newList.text.trim() !== ""
                                            onClicked: {
                                                if (newList.text.trim() && service.createList(newList.text))
                                                    newList.text = "";
                                            }
                                        }
                                    }
                                }

                                }
                            }

                            TapHandler {
                                acceptedDevices: PointerDevice.Mouse | PointerDevice.TouchPad
                                onTapped: if (root.selectedReminderId !== 0) root.selectedReminderId = 0
                            }

                            }

                        Item {
                            id: sidebarResizeHandle
                            visible: sidebar.visible && !root.compactMode
                            width: visible ? Style.space(14) : 0
                            height: parent.height

                            Rectangle {
                                anchors.horizontalCenter: parent.horizontalCenter
                                width: Math.max(1, Style.space(1))
                                height: parent.height
                                color: resizeHover.hovered || resizeDrag.active ? Color.accent : Qt.rgba(Color.menu.text.r, Color.menu.text.g, Color.menu.text.b, 0.18)
                            }

                            HoverHandler {
                                id: resizeHover
                                cursorShape: Qt.SizeHorCursor
                            }

                            DragHandler {
                                id: resizeDrag
                                target: null
                                xAxis.enabled: true
                                yAxis.enabled: false
                                property real startingWidth: root.sidebarWidth
                                onActiveChanged: if (active) startingWidth = root.sidebarWidth
                                onTranslationChanged: root.sidebarWidth = Math.max(Style.space(180), Math.min(Style.space(420), startingWidth + translation.x))
                            }
                        }


                        Item {
                            id: workspace
                            width: parent.width - sidebar.width - sidebarResizeHandle.width
                            height: parent.height

                            Column {
                                id: mainContent
                                x: 0
                                width: parent.width - (!root.overlayTrayMode && (root.detailTrayOpen || root.settingsTrayOpen)
                                  ? (root.detailTrayOpen ? detailSidebar.width : settingsSidebar.width) + Style.space(10)
                                  : 0)
                                height: parent.height
                                spacing: Style.space(8)

                                TapHandler {
                                    acceptedDevices: PointerDevice.Mouse | PointerDevice.TouchPad
                                    onTapped: if (root.selectedReminderId !== 0) root.selectedReminderId = 0
                                }

                            Row {
                                width: parent.width
                                spacing: Style.space(8)

                                TendButton {
                                    id: compactSidebarButton
                                    visible: root.compactMode
                                    text: root.compactSidebarOpen ? "Hide lists" : "Lists"
                                    onClicked: root.compactSidebarOpen = !root.compactSidebarOpen
                                }

                                Text {
                                    width: parent.width - compactSidebarButton.width - editListButton.width - parent.spacing * 2
                                    text: root.viewTitle
                                    elide: Text.ElideRight
                                    color: Color.menu.text
                                    font.family: Style.font.menuFamily
                                    font.pixelSize: Style.font.title
                                    font.bold: true
                                }

                                TendButton {
                                    id: alertSettingsButton
                                    visible: false
                                }

                                TendButton {
                                    id: editListButton

                                    visible: root.viewMode === "list" && root.selectedList !== null
                                    text: "✎"
                                    tooltipText: root.rightTrayMode === "list" ? "Close list settings" : "Edit list"
                                    Accessible.name: tooltipText
                                    onClicked: {
                                        root.openListEditor();
                                    }
                                }

                            }

                            Rectangle {
                                visible: root.viewMode === "list" && root.selectedList !== null
                                width: Style.space(9)
                                height: width
                                radius: width / 2
                                color: root.selectedConnectionAvailable ? "#22c55e" : "#ef4444"
                                Accessible.role: Accessible.Indicator
                                Accessible.name: root.selectedConnectionAvailable ? "List host connection available" : "List host connection unavailable; editing is read-only"
                            }

                            Text {
                                visible: root.selectedListHasPendingOperation
                                text: "PREVIOUS EDIT STILL IN FLIGHT"
                                color: Color.menu.text
                                opacity: 0.68
                                font.family: Style.font.menuFamily
                                font.pixelSize: Style.font.caption
                                Accessible.name: text
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

                                TendButton {
                                    text: "Save"
                                    enabled: root.selectedListEditable && listTitle.text.trim() && listColor.text.trim() && listSymbol.text.trim() && service && service.connectionState === "online" && !service.mutationPending
                                    onClicked: service.updateList(root.selectedList.id, listTitle.text, listColor.text, listSymbol.text, root.selectedList.revision)
                                }

                                TendButton {
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

                                TendButton {
                                    text: service && service.preferences.defaultList === root.selectedListId ? "Default list" : "Make default"
                                    enabled: service && root.selectedListId > 0 && service.preferences.defaultList !== root.selectedListId && service.connectionState === "online" && !service.mutationPending
                                    onClicked: root.updatePreferences(root.selectedListId, service.preferences.pinnedLists, service.preferences.pinnedViews)
                                }

                                TendButton {
                                    text: service && service.preferences.pinnedLists.indexOf(root.selectedListId) !== -1 ? "Unpin list" : "Pin list"
                                    enabled: service && root.selectedListId > 0 && service.connectionState === "online" && !service.mutationPending
                                    onClicked: root.toggleListPin()
                                }

                                TendButton {
                                    text: "Pinned ↑"
                                    enabled: service && root.selectedListId > 0 && service.preferences.pinnedLists.indexOf(root.selectedListId) > 0 && service.connectionState === "online" && !service.mutationPending
                                    onClicked: root.movePinnedList(-1)
                                }

                                TendButton {
                                    text: "Pinned ↓"
                                    enabled: service && root.selectedListId > 0 && service.preferences.pinnedLists.indexOf(root.selectedListId) >= 0 && service.preferences.pinnedLists.indexOf(root.selectedListId) < service.preferences.pinnedLists.length - 1 && service.connectionState === "online" && !service.mutationPending
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

                                TendButton {
                                    text: "List ↑"
                                    enabled: root.canMoveList(-1) && service.connectionState === "online" && !service.mutationPending
                                    Accessible.name: "Move selected unpinned list up"
                                    onClicked: root.moveList(-1)
                                }

                                TendButton {
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
                                        onAccepted: inviteButton.clicked()
                                    }

                                    TendCheck {
                                        id: inviteCanInvite

                                        text: "May invite"
                                        Accessible.name: "Allow invited member to invite others"
                                    }

                                    TendButton {
                                        id: inviteButton

                                        text: "Invite"
                                        enabled: root.selectedListEditable && inviteShip.text.trim() !== "" && service && service.connectionState === "online" && !service.mutationPending
                                        Accessible.name: "Invite Urbit ship to selected list"
                                        onClicked: {
                                            if (service.inviteMember(root.selectedList.id, inviteShip.text.trim(), inviteCanInvite.checked))
                                                inviteShip.text = "";
                                        }
                                    }
                                }

                                Column {
                                    visible: service && service.lastInvitationUri !== "" && service.lastInvitationListId === root.selectedList.id
                                    width: parent.width
                                    spacing: Style.space(4)

                                    Text {
                                        width: parent.width
                                        text: "Invitation link for " + service.lastInvitationTarget + " · bound to that Urbit ship"
                                        color: Color.menu.text
                                        opacity: 0.72
                                        wrapMode: Text.Wrap
                                        font.family: Style.font.menuFamily
                                        font.pixelSize: Style.font.caption
                                    }

                                    TextField {
                                        width: parent.width
                                        text: service ? service.lastInvitationUri : ""
                                        readOnly: true
                                        selectByMouse: true
                                        Accessible.name: "Copy recipient-bound Tend invitation link"
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

                                        TendButton {
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

                                        TendButton {
                                            id: removeMemberButton

                                            visible: root.selectedAccess && root.selectedAccess.owner
                                            text: "Remove"
                                            enabled: root.selectedListEditable && service && service.connectionState === "online" && !service.mutationPending
                                            Accessible.name: "Remove " + modelData.ship + " from selected list"
                                            onClicked: service.removeMember(root.selectedList.id, modelData.ship)
                                        }
                                    }
                                }

                                TendButton {
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

                                    TendCheck {
                                        id: notifyAdded
                                        text: "Items added"
                                        checked: root.selectedCollaborationPolicy.notifyAdded
                                        enabled: service && service.connectionState === "online" && !service.mutationPending
                                        Accessible.name: "Notify when another collaborator adds an item"
                                        onClicked: service.setCollaborationPolicy(root.selectedList.id, checked, notifyCompleted.checked, notifyAssigned.checked)
                                    }

                                    TendCheck {
                                        id: notifyCompleted
                                        text: "Items completed"
                                        checked: root.selectedCollaborationPolicy.notifyCompleted
                                        enabled: service && service.connectionState === "online" && !service.mutationPending
                                        Accessible.name: "Notify when another collaborator completes an item"
                                        onClicked: service.setCollaborationPolicy(root.selectedList.id, notifyAdded.checked, checked, notifyAssigned.checked)
                                    }

                                    TendCheck {
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

                                    width: root.compactMode ? parent.width : parent.width * 0.34
                                    placeholderText: "Search reminders"
                                    Accessible.name: "Search reminders"
                                }

                                TendDropdown {
                                    id: reminderTagFilter

                                    visible: !root.compactMode
                                    width: parent.width * 0.2
                                    model: ["All tags"].concat(root.availableTags)
                                    Accessible.name: "Filter reminders by tag"
                                    onCurrentIndexChanged: root.tagFilter = currentIndex > 0 ? currentText : ""
                                }

                                TendDropdown {
                                    id: reminderSort

                                    visible: !root.compactMode
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

                                TendCheck {
                                    visible: !root.compactMode
                                    text: root.sortDescending ? "Descending" : "Ascending"
                                    checked: root.sortDescending
                                    onClicked: {
                                        root.sortDescending = checked;
                                        if (root.viewMode === "list")
                                            root.saveSelectedPresentation();
                                    }
                                }

                            }

                            RowLayout {
                                width: parent.width
                                spacing: Style.space(6)

                                Text {
                                    text: "ADD NEW REMINDER"
                                    color: Color.menu.text
                                    opacity: 0.72
                                    font.family: Style.font.menuFamily
                                    font.pixelSize: Style.font.caption
                                    font.weight: Font.DemiBold
                                }

                                Item {
                                    Layout.fillWidth: true
                                }

                                TendDropdown {
                                    id: quickDestination

                                    Layout.preferredWidth: Math.min(Style.space(240), parent.width * 0.42)
                                    model: root.orderedLists
                                    textRole: "title"
                                    valueRole: "id"
                                    currentIndex: root.addDestination ? Math.max(0, root.indexForId(model, root.addDestination.id)) : -1
                                    Accessible.name: "New reminder list"
                                    onActivated: root.quickDestinationId = Number(currentValue)
                                }
                            }

                            RowLayout {
                                width: parent.width
                                spacing: Style.space(8)

                                TextField {
                                    id: quickAdd

                                    Layout.fillWidth: true
                                    Layout.minimumWidth: Style.space(180)
                                    placeholderText: root.addDestination ? "Describe a new reminder" : "Create a list first"
                                    Accessible.name: "New reminder description"
                                    enabled: root.addDestination && service && service.canEditList(root.addDestination.id) && service.connectionState === "online" && !service.mutationPending
                                    onAccepted: quickAddButton.clicked()
                                }

                                TendAssigneePicker {
                                    id: quickAssignee

                                    Layout.preferredWidth: Style.space(190)
                                    Layout.minimumWidth: Style.space(150)
                                    options: root.addAssigneeOptions
                                    placeholderText: "Assignee"
                                    Accessible.name: "New reminder assignee"
                                }

                                TendDateTimePicker {
                                    id: quickDue

                                    Layout.preferredWidth: Style.space(210)
                                    Layout.minimumWidth: Style.space(170)
                                    enabled: quickAdd.enabled
                                    Accessible.name: "New reminder due date"
                                }

                                TendButton {
                                    id: quickAddButton
                                    text: "+"
                                    enabled: quickAdd.enabled && quickAdd.text.trim() !== ""
                                    Accessible.name: "Add reminder"
                                    onClicked: {
                                        if (service.addReminderWithDetails(root.addDestination.id, quickAdd.text.trim(), root.addDestination.revision, {
                                            tags: [],
                                            due: quickDue.value,
                                            allDay: quickDue.allDay,
                                            timezone: root.localTimezone(),
                                            assignee: quickAssignee.value || root.normalizedShip(service.ship)
                                        })) {
                                            quickAdd.text = "";
                                            quickDue.value = "";
                                            quickDue.allDay = false;
                                        }
                                    }
                                }
                            }

                            RowLayout {
                                visible: root.viewMode === "list" && !root.compactMode
                                width: parent.width
                                spacing: Style.space(6)

                                TextField {
                                    id: newSection

                                    Layout.preferredWidth: Style.space(220)
                                    placeholderText: "Section title"
                                    Accessible.name: "New section title"
                                    enabled: visible && root.selectedListEditable && root.selectedList && service && service.connectionState === "online" && !service.mutationPending
                                    onAccepted: addSectionButton.clicked()
                                }

                                TendButton {
                                    id: addSectionButton
                                    text: "+ Add section"
                                    bordered: false
                                    enabled: newSection.enabled && newSection.text.trim() !== ""
                                    onClicked: {
                                        if (!newSection.text.trim())
                                            return;
                                        var sections = root.selectedList.sections;
                                        var rank = sections.length ? sections[sections.length - 1].rank + 1024 : 1024;
                                        if (service.addSection(root.selectedList.id, newSection.text, rank, root.selectedList.revision))
                                            newSection.text = "";
                                    }
                                }

                                Item {
                                    Layout.fillWidth: true
                                }

                                TendButton {
                                    id: selectModeButton
                                    text: root.selectionMode ? "Done selecting" : "Select"
                                    onClicked: {
                                        root.selectionMode = !root.selectionMode;
                                        if (!root.selectionMode)
                                            root.selectedReminderIds = [];
                                    }
                                }
                            }

                            Row {
                                visible: false
                                width: parent.width
                                spacing: Style.space(6)

                                TendDropdown {
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

                                TendButton {
                                    text: "Up"
                                    enabled: root.selectedListEditable && root.selectedSectionId !== 0 && sectionChooser.currentIndex > 0 && service && service.connectionState === "online" && !service.mutationPending
                                    Accessible.name: "Move selected section up"
                                    onClicked: root.moveSelectedSectionRelative(-1)
                                }

                                TendButton {
                                    text: "Down"
                                    enabled: root.selectedListEditable && root.selectedSectionId !== 0 && sectionChooser.currentIndex >= 0 && sectionChooser.currentIndex < sectionChooser.count - 1 && service && service.connectionState === "online" && !service.mutationPending
                                    Accessible.name: "Move selected section down"
                                    onClicked: root.moveSelectedSectionRelative(1)
                                }

                                TendButton {
                                    text: "Save section"
                                    enabled: root.selectedListEditable && root.selectedSectionId !== 0 && sectionTitleEditor.text.trim() && service && service.connectionState === "online" && !service.mutationPending
                                    onClicked: {
                                        var section = root.selectedSectionValue();
                                        if (section)
                                            service.updateSection(root.selectedList.id, root.selectedSectionId, sectionTitleEditor.text, section.rank, root.selectedList.revision);
                                    }
                                }

                                TendButton {
                                    text: "Delete"
                                    enabled: root.selectedListEditable && root.selectedSectionId !== 0 && service && service.connectionState === "online" && !service.mutationPending
                                    onClicked: {
                                        service.deleteSection(root.selectedList.id, root.selectedSectionId, root.selectedList.revision);
                                        root.selectedSectionId = 0;
                                        sectionTitleEditor.text = "";
                                    }
                                }

                            }

                            Flow {
                                visible: root.selectionMode && root.viewMode === "list" && root.selectedList !== null
                                width: parent.width
                                spacing: Style.space(6)

                                Text {
                                    height: Style.space(34)
                                    verticalAlignment: Text.AlignVCenter
                                    text: "SELECT MODE · " + root.selectedReminderIds.length + " selected"
                                    color: Color.menu.text
                                    font.family: Style.font.menuFamily
                                    font.pixelSize: Style.font.caption
                                }

                                TendDropdown {
                                    id: batchSection

                                    width: Style.space(190)
                                    model: root.sectionChoices
                                    textRole: "title"
                                    Accessible.name: "Batch destination section"
                                }

                                TendButton {
                                    text: "Move"
                                    enabled: root.selectedListEditable && service && service.connectionState === "online" && !service.mutationPending
                                    onClicked: {
                                        var sectionId = batchSection.currentIndex > 0 ? root.sectionChoices[batchSection.currentIndex].id : null;
                                        if (service.batchMoveReminders(root.selectedList.id, root.selectedReminderIds, sectionId, root.batchStartingRank(), root.selectedList.revision))
                                            root.selectedReminderIds = [];
                                    }
                                }

                                TendButton {
                                    text: "↑"
                                    tooltipText: "Move selected reminder up"
                                    enabled: root.selectedReminderIds.length === 1 && root.selectedListEditable && service && service.connectionState === "online" && !service.mutationPending
                                    onClicked: root.moveReminderRelative(root.selectedBatchReminder(), -1)
                                }

                                TendButton {
                                    text: "↓"
                                    tooltipText: "Move selected reminder down"
                                    enabled: root.selectedReminderIds.length === 1 && root.selectedListEditable && service && service.connectionState === "online" && !service.mutationPending
                                    onClicked: root.moveReminderRelative(root.selectedBatchReminder(), 1)
                                }

                                TendButton {
                                    text: "Indent"
                                    enabled: root.selectedReminderIds.length === 1 && root.selectedListEditable && service && service.connectionState === "online" && !service.mutationPending
                                    onClicked: root.indentReminder(root.selectedBatchReminder())
                                }

                                TendButton {
                                    text: "Outdent"
                                    enabled: root.selectedReminderIds.length === 1 && root.selectedBatchReminder() && root.selectedBatchReminder().parentId !== null && root.selectedListEditable && service && service.connectionState === "online" && !service.mutationPending
                                    onClicked: root.outdentReminder(root.selectedBatchReminder())
                                }

                                TendButton {
                                    text: "Complete"
                                    enabled: root.selectedListEditable && service && service.connectionState === "online" && !service.mutationPending
                                    onClicked: {
                                        if (service.batchSetCompleted(root.selectedList.id, root.selectedReminderIds, true, root.selectedList.revision))
                                            root.selectedReminderIds = [];
                                    }
                                }

                                TendButton {
                                    text: "Uncomplete"
                                    enabled: root.selectedListEditable && service && service.connectionState === "online" && !service.mutationPending
                                    onClicked: {
                                        if (service.batchSetCompleted(root.selectedList.id, root.selectedReminderIds, false, root.selectedList.revision))
                                            root.selectedReminderIds = [];
                                    }
                                }

                                TendButton {
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
                                height: 0
                                visible: false
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

                                    QQC.TextArea {
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

                                        TendButton {
                                            text: "Open link"
                                            enabled: TendModel.safeExternalUrl(reminderUrl.text)
                                            Accessible.name: "Open reminder link in the default application"
                                            onClicked: Qt.openUrlExternally(reminderUrl.text.trim())
                                        }

                                        TendDropdown {
                                            id: reminderPriority

                                            width: parent.width * 0.2
                                            model: ["none", "low", "medium", "high"]
                                            Accessible.name: "Reminder priority"
                                        }

                                        TendCheck {
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

                                    TendDropdown {
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

                                        TendCheck {
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

                                        TendDropdown {
                                            id: reminderParent

                                            width: parent.width * 0.3
                                            model: root.parentChoices
                                            textRole: "title"
                                            Accessible.name: "Parent reminder"
                                        }

                                        TendDropdown {
                                            id: reminderSection

                                            width: parent.width * 0.3
                                            model: root.sectionChoices
                                            textRole: "title"
                                            Accessible.name: "Reminder section"
                                        }

                                        TendNumber {
                                            id: reminderRank

                                            from: 0
                                            to: 2147483647
                                            value: 0
                                            editable: true
                                            Accessible.name: "Reminder rank"
                                        }

                                        TendButton {
                                            text: "Move"
                                            enabled: root.selectedListEditable && root.selectedReminder && service && service.connectionState === "online" && !service.mutationPending
                                            onClicked: {
                                                var parentId = reminderParent.currentIndex > 0 ? root.parentChoices[reminderParent.currentIndex].id : null;
                                                var sectionId = reminderSection.currentIndex > 0 ? root.sectionChoices[reminderSection.currentIndex].id : null;
                                                service.moveReminder(root.selectedList.id, root.selectedReminder.id, parentId, sectionId, reminderRank.value, root.selectedList.revision);
                                            }
                                        }

                                        TendButton {
                                            text: "↑"
                                            Accessible.name: "Move reminder up"
                                            enabled: root.selectedListEditable && root.selectedReminder && service && service.connectionState === "online" && !service.mutationPending
                                            onClicked: root.moveSelectedRelative(-1)
                                        }

                                        TendButton {
                                            text: "↓"
                                            Accessible.name: "Move reminder down"
                                            enabled: root.selectedListEditable && root.selectedReminder && service && service.connectionState === "online" && !service.mutationPending
                                            onClicked: root.moveSelectedRelative(1)
                                        }

                                    }

                                    Row {
                                        spacing: Style.space(8)

                                        TendButton {
                                            text: "Indent"
                                            enabled: root.selectedListEditable && root.selectedReminder && service && service.connectionState === "online" && !service.mutationPending
                                            onClicked: root.indentSelectedReminder()
                                        }

                                        TendButton {
                                            text: "Outdent"
                                            enabled: root.selectedListEditable && root.selectedReminder && root.selectedReminder.parentId !== null && service && service.connectionState === "online" && !service.mutationPending
                                            onClicked: root.outdentSelectedReminder()
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

                                        TendCheck {
                                            id: reminderOrdinal

                                            text: "Ordinal"
                                        }

                                        TendNumber {
                                            id: reminderOrdinalIndex

                                            from: 1
                                            to: 5
                                            value: 1
                                            enabled: reminderOrdinal.checked
                                            Accessible.name: "Ordinal week index"
                                        }

                                        TendNumber {
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

                                        TendNumber {
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

                                        TendDropdown {
                                            id: reminderRepeat

                                            width: parent.width * 0.25
                                            model: ["none", "hourly", "daily", "weekly", "monthly", "yearly"]
                                            Accessible.name: "Repeat frequency"
                                        }

                                        TendNumber {
                                            id: reminderInterval

                                            from: 1
                                            to: 999
                                            value: 1
                                            editable: true
                                            Accessible.name: "Repeat interval"
                                        }

                                        TendButton {
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

                                        TendButton {
                                            text: "Clear"
                                            enabled: root.selectedListEditable && root.selectedReminder && root.selectedReminder.schedule && service && service.connectionState === "online" && !service.mutationPending
                                            onClicked: service.setSchedule(root.selectedList.id, root.selectedReminder.id, null, root.selectedList.revision)
                                        }

                                    }

                                    Row {
                                        spacing: Style.space(8)

                                        TendButton {
                                            id: reminderSave

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

                                        TendButton {
                                            text: "Delete"
                                            enabled: root.selectedListEditable && root.selectedReminder && service && service.connectionState === "online" && !service.mutationPending
                                            onClicked: service.deleteReminder(root.selectedList.id, root.selectedReminder.id, root.selectedList.revision)
                                        }

                                        TendDropdown {
                                            id: snoozePreset

                                            model: service ? service.preferences.snoozePresets : []
                                            displayText: currentValue ? Math.round(Number(currentValue) / 60) + " min" : "Snooze"
                                            Accessible.name: "Snooze duration"
                                        }

                                        TendButton {
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

                                        TendButton {
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
                                model: root.displayedRows
                                Accessible.role: Accessible.List
                                Accessible.name: "Reminders"

                                delegate: Item {
                                    id: reminderRow

                                    required property var modelData
                                    required property int index
                                    readonly property bool sectionRow: modelData.kind === "section"
                                    readonly property var reminderData: sectionRow ? null : (modelData.reminder || modelData)
                                    readonly property real indentPixels: sectionRow ? 0 : Math.max(0, Number(reminderData.depth || 0)) * Style.space(22)
                                    readonly property bool currentRow: ListView.isCurrentItem

                                    width: ListView.view.width
                                    height: sectionRow ? Style.space(38) : Style.space(44)
                                    opacity: !sectionRow && service ? service.completionOpacity(reminderData.listId, reminderData.id) : 1
                                    Accessible.role: sectionRow ? Accessible.Heading : Accessible.ListItem
                                    Accessible.name: sectionRow
                                        ? modelData.section.title + ", " + modelData.count + " reminders"
                                        : (Number(reminderData.depth || 0) > 0 ? "Subtask, " : "") + reminderData.title + (root.reminderHasChildren(reminderData.id) ? (root.reminderIsCollapsed(reminderData.id) ? ", collapsed" : ", expanded") : "")

                                    Rectangle {
                                        id: reminderCard

                                        x: reminderRow.indentPixels
                                        width: Math.max(Style.space(120), reminderRow.width - x)
                                        height: parent.height
                                        radius: Style.cornerRadius
                                        color: reminderRow.sectionRow ? "transparent" : (root.selectionMode && root.reminderIsSelected(reminderRow.reminderData.id)
                                            ? Style.selectedFillFor(Color.menu.text, Color.accent)
                                            : (rowHover.hovered || reminderRow.currentRow
                                                ? Qt.rgba(Color.menu.text.r, Color.menu.text.g, Color.menu.text.b, 0.14)
                                                : Qt.rgba(Color.menu.text.r, Color.menu.text.g, Color.menu.text.b, 0.04)))
                                        border.width: !reminderRow.sectionRow && reminderDropArea.containsDrag ? Math.max(1, Style.space(1)) : 0
                                        border.color: reminderDropArea.containsDrag ? Color.menu.text : "transparent"

                                        Row {
                                            visible: reminderRow.sectionRow
                                            anchors.left: parent.left
                                            anchors.right: parent.right
                                            anchors.verticalCenter: parent.verticalCenter
                                            anchors.leftMargin: Style.space(8)
                                            anchors.rightMargin: Style.space(8)
                                            spacing: Style.space(8)

                                            Text {
                                                width: parent.width - sectionCount.width - parent.spacing
                                                text: modelData.section ? modelData.section.title : "Section"
                                                color: Color.menu.text
                                                opacity: 0.72
                                                elide: Text.ElideRight
                                                font.family: Style.font.menuFamily
                                                font.pixelSize: Style.font.caption
                                                font.bold: true
                                            }

                                            Text {
                                                id: sectionCount
                                                text: String(Number(modelData.count || 0))
                                                color: Color.menu.text
                                                opacity: 0.5
                                                font.family: Style.font.menuFamily
                                                font.pixelSize: Style.font.caption
                                            }
                                        }

                                        DropArea {
                                            id: reminderDropArea

                                            anchors.fill: parent
                                            enabled: !reminderRow.sectionRow && root.viewMode === "list" && root.sortMode === "manual" && root.selectedListEditable && service && service.connectionState === "online" && !service.mutationPending
                                            keys: ["tend-reminder"]
                                            onDropped: function(drop) {
                                                if (drop.source && root.placeReminder(drop.source.modelData, reminderRow.reminderData, drop.y >= height / 2))
                                                    drop.acceptProposedAction();
                                            }
                                        }

                                        Row {
                                            visible: !reminderRow.sectionRow
                                            anchors.fill: parent
                                            anchors.margins: Style.space(8)
                                            spacing: Style.space(10)

                                        TendCheckbox {
                                            checked: !reminderRow.sectionRow && (service ? service.effectiveCompleted(reminderRow.reminderData.listId, reminderRow.reminderData.id, reminderRow.reminderData.completed) : reminderRow.reminderData.completed)
                                            enabled: !reminderRow.sectionRow && service && service.canEditList(reminderRow.reminderData.listId) && service.connectionState === "online" && !service.mutationPending
                                            Accessible.name: reminderRow.sectionRow ? "" : (checked ? "Mark open " : "Complete ") + reminderRow.reminderData.title
                                            onClicked: service.toggleCompletedWithGrace(reminderRow.reminderData.listId, reminderRow.reminderData.id, reminderRow.reminderData.completed)
                                        }

                                        TendFlagButton {
                                            flagged: !reminderRow.sectionRow && reminderRow.reminderData.flagged === true
                                            enabled: !reminderRow.sectionRow && service && service.canEditList(reminderRow.reminderData.listId) && service.connectionState === "online" && !service.mutationPending
                                            onToggled: function(flagged) { service.setReminderFlagged(reminderRow.reminderData.listId, reminderRow.reminderData.id, flagged) }
                                        }

                                        TendButton {
                                            visible: !reminderRow.sectionRow && root.viewMode === "list" && root.reminderHasChildren(reminderRow.reminderData.id)
                                            text: reminderRow.sectionRow ? "" : (root.reminderIsCollapsed(reminderRow.reminderData.id) ? "▶" : "▼")
                                            Accessible.name: reminderRow.sectionRow ? "" : (root.reminderIsCollapsed(reminderRow.reminderData.id) ? "Expand " : "Collapse ") + reminderRow.reminderData.title
                                            onClicked: root.toggleReminderCollapsed(reminderRow.reminderData.id)
                                        }

                                        Text {
                                            anchors.verticalCenter: parent.verticalCenter
                                            width: parent.width - x
                                            text: {
                                                if (reminderRow.sectionRow)
                                                    return "";
                                                var list = root.viewMode === "list" ? "" : "  ·  " + reminderRow.reminderData.listTitle;
                                                var section = root.sectionTitleFor(reminderRow.reminderData.listId, reminderRow.reminderData.sectionId);
                                                var due = reminderRow.reminderData.schedule ? "  ·  " + TendModel.scheduleInputValue(reminderRow.reminderData.schedule).replace("T", " ") : "";
                                                return reminderRow.reminderData.title + list + (section ? "  ·  " + section : "") + due;
                                            }
                                            color: Color.menu.text
                                            opacity: !reminderRow.sectionRow && service && service.effectiveCompleted(reminderRow.reminderData.listId, reminderRow.reminderData.id, reminderRow.reminderData.completed) ? 0.5 : 1
                                            font.family: Style.font.menuFamily
                                            font.pixelSize: Style.font.body
                                            font.strikeout: !reminderRow.sectionRow && (service ? service.effectiveCompleted(reminderRow.reminderData.listId, reminderRow.reminderData.id, reminderRow.reminderData.completed) : reminderRow.reminderData.completed)

                                            MouseArea {
                                                anchors.fill: parent
                                                cursorShape: Qt.PointingHandCursor
                                                onClicked: {
                                                    reminderList.currentIndex = index;
                                                    if (root.selectionMode && root.viewMode === "list")
                                                        root.toggleReminderSelection(reminderRow.reminderData.id);
                                                    else
                                                        Qt.callLater(function() { root.editReminder(reminderRow.reminderData); });
                                                }
                                            }
                                        }

                                        }

                                        HoverHandler { id: rowHover }

                                    }

                                }

                            }

                            }

                            ReminderDetail {
                                id: detailSidebar
                                anchors.top: parent.top
                                anchors.bottom: parent.bottom
                                anchors.right: parent.right
                                width: root.detailTrayOpen ? root.detailTrayWidth : 0
                                visible: root.detailTrayOpen
                                z: root.overlayTrayMode ? 20 : 2
                                service: root.service
                                list: root.selectedList
                                reminder: root.selectedReminder
                                assigneeChoices: root.assigneeChoices
                                listChoices: root.movableListChoices
                                availableTags: root.availableTags
                                editable: root.selectedListEditable
                                timezone: root.localTimezone()
                                onCloseRequested: root.selectedReminderId = 0
                                onMoveRequested: function(destinationListId) {
                                    if (!root.selectedList || !root.selectedReminder || !service)
                                        return;
                                    var destination = null;
                                    for (var i = 0; i < service.lists.length; i++)
                                        if (Number(service.lists[i].id) === Number(destinationListId)) destination = service.lists[i];
                                    if (!destination || Number(destination.id) === Number(root.selectedList.id))
                                        return;
                                    if (service.moveReminderToList(root.selectedList.id, destination.id, root.selectedReminder.id, root.selectedList.revision, destination.revision)) {
                                        root.selectedListId = destination.id;
                                        root.selectedReminderId = 0;
                                    }
                                }
                            }

                            TendSettings {
                                id: settingsSidebar
                                anchors.top: parent.top
                                anchors.bottom: parent.bottom
                                anchors.right: parent.right
                                width: root.settingsTrayOpen ? root.settingsTrayWidth : 0
                                visible: root.settingsTrayOpen
                                z: root.overlayTrayMode ? 20 : 2
                                service: root.service
                                mode: root.rightTrayMode
                                list: root.selectedList
                                access: root.selectedAccess
                                editable: root.selectedListEditable
                                mayInvite: root.selectedMayInvite
                                invitations: root.service ? root.service.invitations : []
                                activities: root.selectedActivities
                                collaborationPolicy: root.selectedCollaborationPolicy
                                onCloseRequested: root.rightTrayMode = ""
                                onSwitchShipRequested: {
                                    root.rightTrayMode = "";
                                    root.switchingShip = true;
                                    Qt.callLater(function() { shipUrl.forceActiveFocus(); });
                                }
                                onRemoveShipRequested: {
                                    root.rightTrayMode = "";
                                    root.switchingShip = false;
                                    root.service.disconnect();
                                }
                                onListDeleted: {
                                    root.rightTrayMode = "";
                                    root.selectedListId = 0;
                                }
                            }

                        }

                    }

                }

            }

        }

    }

}
