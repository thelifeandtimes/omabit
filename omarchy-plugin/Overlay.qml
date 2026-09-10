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
    property string viewMode: "list"
    property string sortMode: "manual"
    property bool sortDescending: false
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
        reminderDue.text = reminder.schedule ? urbitToInput(reminder.schedule.dueAt) : "";
        reminderAllDay.checked = reminder.schedule ? reminder.schedule.allDay : false;
        reminderTimezone.text = reminder.schedule ? reminder.schedule.timezone : localTimezone();
        reminderEarly.text = reminder.schedule ? reminder.schedule.earlySeconds.map(function(seconds) {
            return seconds / 60;
        }).join(", ") : "";
        reminderRepeat.currentIndex = reminder.schedule && reminder.schedule.recurrence ? Math.max(0, reminderRepeat.model.indexOf(reminder.schedule.recurrence.frequency)) : 0;
        reminderInterval.value = reminder.schedule && reminder.schedule.recurrence ? reminder.schedule.recurrence.interval : 1;
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
                                            text: modelData.charAt(0).toUpperCase() + modelData.slice(1)
                                            checkable: true
                                            checked: root.viewMode === modelData
                                            onClicked: {
                                                root.viewMode = modelData;
                                                root.selectedReminderId = 0;
                                            }
                                        }

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
                                    model: service ? service.lists : []

                                    delegate: Button {
                                        required property var modelData

                                        width: parent.width
                                        text: modelData.title
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

                            Text {
                                text: root.viewTitle
                                color: Color.menu.text
                                font.family: Style.font.menuFamily
                                font.pixelSize: Style.font.title
                                font.bold: true
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
                                    placeholderText: root.selectedList ? "Add a reminder" : "Create a list first"
                                    enabled: root.selectedList && service && service.connectionState === "online" && !service.mutationPending
                                    onAccepted: {
                                        if (text.trim() && service.addReminder(root.selectedList.id, text, root.selectedList.revision))
                                            text = "";

                                    }
                                }

                                TextField {
                                    id: newSection

                                    width: parent.width - quickAdd.width - parent.spacing
                                    placeholderText: "Add section"
                                    visible: root.viewMode === "list"
                                    enabled: visible && root.selectedList && service && service.connectionState === "online" && !service.mutationPending
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

                            Rectangle {
                                width: parent.width
                                height: root.selectedReminder ? Style.space(330) : 0
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
                                            enabled: root.selectedReminder && reminderDue.text.trim() && reminderTimezone.text.trim() && service && service.connectionState === "online" && !service.mutationPending
                                            onClicked: {
                                                var early = reminderEarly.text.split(",").map(function(minutes) {
                                                    return Math.round(Number(minutes.trim()) * 60);
                                                }).filter(function(seconds) {
                                                    return isFinite(seconds) && seconds >= 0;
                                                });
                                                var recurrence = reminderRepeat.currentText === "none" ? null : {
                                                    frequency: reminderRepeat.currentText,
                                                    interval: reminderInterval.value,
                                                    weekdays: [],
                                                    monthDays: [],
                                                    monthWeek: null,
                                                    endAt: null,
                                                    maxOccurrences: null
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
                                            enabled: root.selectedReminder && root.selectedReminder.schedule && service && service.connectionState === "online" && !service.mutationPending
                                            onClicked: service.setSchedule(root.selectedList.id, root.selectedReminder.id, null, root.selectedList.revision)
                                        }

                                    }

                                    Row {
                                        spacing: Style.space(8)

                                        Button {
                                            text: service && service.mutationPending ? "Saving…" : "Save"
                                            enabled: root.selectedReminder && reminderTitle.text.trim() && service && service.connectionState === "online" && !service.mutationPending
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
                                                    tags: tags
                                                }, root.selectedList.revision);
                                            }
                                        }

                                        Button {
                                            text: "Delete"
                                            enabled: root.selectedReminder && service && service.connectionState === "online" && !service.mutationPending
                                            onClicked: service.deleteReminder(root.selectedList.id, root.selectedReminder.id, root.selectedList.revision)
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
                                            enabled: service && service.connectionState === "online" && !service.mutationPending
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
