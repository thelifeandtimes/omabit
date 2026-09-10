import QtQuick
import QtQuick.Controls
import Quickshell
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
    property bool sortDescending: false
    property bool listEditorOpen: false
    property bool confirmDeleteList: false
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
    readonly property string selectedHostStatus: {
        if (!selectedAccess)
            return "CHECKING OWNER";
        if (selectedAccess.owner)
            return "OWNED HERE · ONLINE";
        return "HOSTED BY " + selectedAccess.host.toUpperCase() + " · " + selectedAccess.status.toUpperCase();
    }
    readonly property var orderedLists: TendModel.orderedLists(service ? service.lists : [], service ? service.preferences.pinnedLists : [])
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
    readonly property var displayedReminders: TendModel.queryReminders(service ? service.lists : [], {
        view: viewMode,
        listId: selectedList ? selectedList.id : 0,
        search: reminderSearch.text,
        sort: sortMode,
        descending: sortDescending
    })
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
        reminderAssignee.text = reminder.assignee || "";
        reminderDue.text = reminder.schedule ? urbitToInput(reminder.schedule.dueAt) : "";
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
        reminderRepeatEnd.text = reminder.schedule && reminder.schedule.recurrence && reminder.schedule.recurrence.endAt ? urbitToInput(reminder.schedule.recurrence.endAt) : "";
        reminderRepeatCount.value = reminder.schedule && reminder.schedule.recurrence && reminder.schedule.recurrence.maxOccurrences ? reminder.schedule.recurrence.maxOccurrences : 0;
        reminderRank.value = reminder.rank;
        Qt.callLater(function() {
            reminderParent.currentIndex = root.indexForId(root.parentChoices, reminder.parentId);
            reminderSection.currentIndex = root.indexForId(root.sectionChoices, reminder.sectionId);
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
        sectionRank.value = section.rank;
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

        var rank;
        if (delta < 0) {
            var lower = target > 0 ? siblings[target - 1].rank : 0;
            rank = Math.floor((lower + siblings[target].rank) / 2);
        } else {
            var upper = target + 1 < siblings.length ? siblings[target + 1].rank : siblings[target].rank + 2048;
            rank = Math.floor((siblings[target].rank + upper) / 2);
        }
        service.moveReminder(selectedList.id, selectedReminder.id, selectedReminder.parentId, selectedReminder.sectionId, rank, selectedList.revision);
    }

    function localTimezone() {
        try {
            return Intl.DateTimeFormat().resolvedOptions().timeZone || "UTC";
        } catch (error) {
            return "UTC";
        }
    }

    function pad2(value) {
        return Number(value) < 10 ? "0" + Number(value) : String(value);
    }

    function urbitToInput(value) {
        var match = /^~(\d+)\.(\d+)\.(\d+)\.\.(\d+)\.(\d+)/.exec(String(value || ""));
        if (!match)
            return String(value || "");

        return match[1] + "-" + pad2(match[2]) + "-" + pad2(match[3]) + "T" + pad2(match[4]) + ":" + pad2(match[5]);
    }

    function open(payloadJson) {
        root.opened = true;
        Qt.callLater(function() {
            var payload = {};
            try {
                payload = JSON.parse(String(payloadJson || "{}"));
            } catch (error) {
                payload = {};
            }
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
            if (service && service.ship)
                quickAdd.forceActiveFocus();
            else
                shipUrl.forceActiveFocus();
        });
    }

    function close() {
        root.opened = false;
    }

    function dismiss() {
        root.opened = false;
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
            selectedListId = selectedList.id;
            listEditorOpen = false;
            confirmDeleteList = false;
            selectedSectionId = 0;
        }
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
                width: Math.min(1100, parent.width - Style.space(48))
                height: Math.min(760, parent.height - Style.space(48))
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
                            width: parent.width - closeButton.width
                            text: service && service.ship ? "Tend · ~" + service.ship : "Tend"
                            color: Color.menu.text
                            font.family: Style.font.menuFamily
                            font.pixelSize: Style.font.heading
                            font.bold: true
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
                        visible: !service || !service.ship
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
                            text: "Enter your ship domain and the current code printed by +code. Tend accepts planets, moons, and comets."
                            color: Color.menu.text
                            opacity: 0.72
                            wrapMode: Text.Wrap
                            font.family: Style.font.menuFamily
                            font.pixelSize: Style.font.body
                        }

                        TextField {
                            id: shipUrl

                            width: parent.width
                            placeholderText: "https://sampel-palnet.arvo.network"
                        }

                        TextField {
                            id: loginCode

                            width: parent.width
                            placeholderText: "lidlut-tabwed-pillex-ridrup"
                            echoMode: TextInput.Password
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

                    Row {
                        visible: service && service.ship !== ""
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
                                        model: ["today", "scheduled", "all", "flagged", "completed"]

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
                                        model: ["today", "scheduled", "all", "flagged", "completed"]
                                    }

                                    Button {
                                        width: parent.width - viewPinChoice.width - parent.spacing
                                        text: service && service.preferences.pinnedViews.indexOf(viewPinChoice.currentText) !== -1 ? "Unpin" : "Pin"
                                        enabled: service && service.connectionState === "online" && !service.mutationPending
                                        onClicked: root.toggleViewPin(viewPinChoice.currentText)
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
                                    enabled: service && service.connectionState === "online" && !service.mutationPending
                                    onAccepted: {
                                        if (text.trim() && service.createList(text))
                                            text = "";

                                    }
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
                                    width: parent.width - editListButton.width - parent.spacing
                                    text: root.viewTitle
                                    color: Color.menu.text
                                    font.family: Style.font.menuFamily
                                    font.pixelSize: Style.font.title
                                    font.bold: true
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
                                }

                                TextField {
                                    id: tagReplacement

                                    width: parent.width * 0.3
                                    placeholderText: "Rename or merge tag"
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
                                }

                                TextField {
                                    id: listColor

                                    width: parent.width * 0.2
                                    placeholderText: "#3b82f6"
                                }

                                TextField {
                                    id: listSymbol

                                    width: parent.width * 0.16
                                    placeholderText: "list or emoji"
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
                                width: parent.width
                                spacing: Style.space(8)

                                TextField {
                                    id: reminderSearch

                                    width: parent.width * 0.5
                                    placeholderText: "Search reminders"
                                }

                                ComboBox {
                                    id: reminderSort

                                    width: parent.width * 0.25
                                    model: ["manual", "due", "created", "priority", "title"]
                                    onCurrentTextChanged: root.sortMode = currentText
                                }

                                CheckBox {
                                    text: "Descending"
                                    checked: root.sortDescending
                                    onToggled: root.sortDescending = checked
                                }

                            }

                            Row {
                                width: parent.width
                                spacing: Style.space(8)

                                TextField {
                                    id: quickAdd

                                    width: parent.width * 0.66
                                    placeholderText: root.addDestination ? "Add to " + root.addDestination.title : "Create a list first"
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
                                    onActivated: function(index) {
                                        root.chooseSection(index);
                                    }
                                }

                                TextField {
                                    id: sectionTitleEditor

                                    width: parent.width * 0.3
                                    placeholderText: "Section title"
                                }

                                SpinBox {
                                    id: sectionRank

                                    from: 0
                                    to: 2147483647
                                    value: 0
                                    editable: true
                                }

                                Button {
                                    text: "Save section"
                                    enabled: root.selectedListEditable && root.selectedSectionId !== 0 && sectionTitleEditor.text.trim() && service && service.connectionState === "online" && !service.mutationPending
                                    onClicked: service.updateSection(root.selectedList.id, root.selectedSectionId, sectionTitleEditor.text, sectionRank.value, root.selectedList.revision)
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
                                    }

                                    TextArea {
                                        id: reminderNotes

                                        width: parent.width
                                        height: Style.space(54)
                                        placeholderText: "Notes"
                                        wrapMode: TextEdit.Wrap
                                    }

                                    Row {
                                        width: parent.width
                                        spacing: Style.space(6)

                                        TextField {
                                            id: reminderUrl

                                            width: parent.width * 0.5
                                            placeholderText: "https://…"
                                        }

                                        ComboBox {
                                            id: reminderPriority

                                            width: parent.width * 0.24
                                            model: ["none", "low", "medium", "high"]
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
                                    }

                                    TextField {
                                        id: reminderAssignee

                                        width: parent.width
                                        placeholderText: "Assignee ship, for example ~sampel-palnet"
                                    }

                                    Row {
                                        width: parent.width
                                        spacing: Style.space(6)

                                        TextField {
                                            id: reminderDue

                                            width: parent.width * 0.34
                                            placeholderText: "2026-09-10T17:30"
                                        }

                                        TextField {
                                            id: reminderTimezone

                                            width: parent.width * 0.28
                                            placeholderText: "America/Los_Angeles"
                                        }

                                        CheckBox {
                                            id: reminderAllDay

                                            text: "All day"
                                        }

                                        TextField {
                                            id: reminderEarly

                                            width: parent.width - x
                                            placeholderText: "Early min"
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
                                        }

                                        ComboBox {
                                            id: reminderSection

                                            width: parent.width * 0.3
                                            model: root.sectionChoices
                                            textRole: "title"
                                        }

                                        SpinBox {
                                            id: reminderRank

                                            from: 0
                                            to: 2147483647
                                            value: 0
                                            editable: true
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
                                            enabled: root.selectedListEditable && root.selectedReminder && service && service.connectionState === "online" && !service.mutationPending
                                            onClicked: root.moveSelectedRelative(-1)
                                        }

                                        Button {
                                            text: "↓"
                                            enabled: root.selectedListEditable && root.selectedReminder && service && service.connectionState === "online" && !service.mutationPending
                                            onClicked: root.moveSelectedRelative(1)
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
                                        }

                                        TextField {
                                            id: reminderMonthDays

                                            width: parent.width * 0.15
                                            placeholderText: "Month days"
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
                                        }

                                        SpinBox {
                                            id: reminderOrdinalWeekday

                                            from: 0
                                            to: 6
                                            value: 0
                                            enabled: reminderOrdinal.checked
                                        }

                                        TextField {
                                            id: reminderRepeatEnd

                                            width: parent.width * 0.2
                                            placeholderText: "End date/time"
                                        }

                                        SpinBox {
                                            id: reminderRepeatCount

                                            from: 0
                                            to: 9999
                                            value: 0
                                            editable: true
                                        }

                                    }

                                    Row {
                                        width: parent.width
                                        spacing: Style.space(6)

                                        ComboBox {
                                            id: reminderRepeat

                                            width: parent.width * 0.25
                                            model: ["none", "hourly", "daily", "weekly", "monthly", "yearly"]
                                        }

                                        SpinBox {
                                            id: reminderInterval

                                            from: 1
                                            to: 999
                                            value: 1
                                            editable: true
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
                                                    assignee: reminderAssignee.text.trim() || null
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
                                        }

                                        Button {
                                            text: "Snooze"
                                            enabled: root.selectedReminder && root.selectedReminder.schedule && snoozePreset.currentIndex >= 0 && service && service.connectionState === "online" && !service.mutationPending
                                            onClicked: service.snoozeReminder(root.selectedList.id, root.selectedReminder.id, Number(snoozePreset.currentValue))
                                        }

                                    }

                                }

                            }

                            ListView {
                                width: parent.width
                                height: parent.height - y
                                clip: true
                                spacing: Style.space(4)
                                model: root.displayedReminders

                                delegate: Rectangle {
                                    required property var modelData

                                    width: ListView.view.width
                                    height: Style.space(44)
                                    radius: Style.cornerRadius
                                    color: Qt.rgba(Color.menu.text.r, Color.menu.text.g, Color.menu.text.b, 0.04)

                                    Row {
                                        anchors.fill: parent
                                        anchors.margins: Style.space(8)
                                        spacing: Style.space(10)

                                        CheckBox {
                                            checked: modelData.completed
                                            enabled: service && service.canEditList(modelData.listId) && service.connectionState === "online" && !service.mutationPending
                                            onClicked: service.setCompleted(modelData.listId, modelData.id, checked, modelData.listRevision)
                                        }

                                        Text {
                                            anchors.verticalCenter: parent.verticalCenter
                                            width: parent.width - x
                                            text: {
                                                var list = root.viewMode === "list" ? "" : "  ·  " + modelData.listTitle;
                                                var section = root.sectionTitleFor(modelData.listId, modelData.sectionId);
                                                var prefix = modelData.parentId === null ? "" : "↳ ";
                                                var due = modelData.schedule ? "  ·  " + root.urbitToInput(modelData.schedule.dueAt).replace("T", " ") : "";
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
                                                onClicked: root.editReminder(modelData)
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
