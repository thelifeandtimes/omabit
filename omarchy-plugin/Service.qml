import QtQuick
import Quickshell
import Quickshell.Io
import "TendModel.js" as TendModel

Item {
    id: root

    property var shell: null
    property var manifest: null
    property string connectionState: "disconnected"
    property string errorMessage: ""
    property string baseUrl: ""
    property string ship: ""
    property var lists: []
    property var preferences: TendModel.clonePreferences(null)
    property var snoozes: []
    property var accesses: []
    property var invitations: []
    property var pendingOperations: []
    property var lastAlert: null
    property var notificationQueue: []
    property bool mutationPending: false
    property int reconnectAttempt: 0
    property bool streamAuthenticationFailed: false
    readonly property int incompleteCount: TendModel.incompleteCount(lists)
    readonly property int badgeCount: TendModel.badgeCount(lists, preferences, ship)
    readonly property var nextReminder: TendModel.nextReminder(lists)
    readonly property string pluginDir: manifest && manifest.__sourceDir ? String(manifest.__sourceDir) : ""
    readonly property string bridgePath: pluginDir ? pluginDir + "/transport/eyre_client.py" : ""
    readonly property string runtimeRoot: (Quickshell.env("XDG_RUNTIME_DIR") || (Quickshell.env("HOME") + "/.cache")) + "/omabit/tend"
    readonly property string configRoot: (Quickshell.env("XDG_CONFIG_HOME") || (Quickshell.env("HOME") + "/.config")) + "/omabit/tend"
    readonly property string cookiePath: runtimeRoot + "/cookies.txt"
    readonly property string connectionPath: configRoot + "/connection.json"
    signal reminderAlert(var alert)

    function operationId() {
        return Date.now().toString(36) + "-" + Math.floor(Math.random() * 2.14748e+09).toString(36);
    }

    function pad2(value) {
        return value < 10 ? "0" + value : String(value);
    }

    function toUrbitDate(value) {
        var text = String(value || "").trim();
        if (text.charAt(0) === "~")
            return text;

        var date = new Date(text);
        if (isNaN(date.getTime()))
            return text;

        return "~" + date.getUTCFullYear() + "." + (date.getUTCMonth() + 1) + "." + date.getUTCDate() + ".." + pad2(date.getUTCHours()) + "." + pad2(date.getUTCMinutes()) + "." + pad2(date.getUTCSeconds());
    }

    function recurrencePayload(recurrence) {
        if (!recurrence)
            return null;

        return {
            "frequency": String(recurrence.frequency || "daily"),
            "interval": Math.max(1, Number(recurrence.interval || 1)),
            "weekdays": (recurrence.weekdays || []).map(Number),
            "month-days": (recurrence.monthDays || recurrence["month-days"] || []).map(Number),
            "month-week": recurrence.monthWeek || recurrence["month-week"] || null,
            "end-at": recurrence.endAt || recurrence["end-at"] ? toUrbitDate(recurrence.endAt || recurrence["end-at"]) : null,
            "max-occurrences": recurrence.maxOccurrences === undefined ? (recurrence["max-occurrences"] ?? null) : recurrence.maxOccurrences
        };
    }

    function login(url, code) {
        if (!bridgePath || loginProcess.running)
            return ;

        root.errorMessage = "";
        root.connectionState = "authenticating";
        loginProcess.secret = String(code || "");
        loginProcess.command = ["python3", bridgePath, "login", "--url", String(url || ""), "--cookie", cookiePath, "--config", connectionPath];
        loginProcess.running = true;
    }

    function restore() {
        if (!bridgePath || statusProcess.running)
            return ;

        statusProcess.command = ["python3", bridgePath, "status", "--cookie", cookiePath, "--config", connectionPath];
        statusProcess.running = true;
    }

    function startStream() {
        if (!bridgePath || !ship || streamProcess.running)
            return ;

        root.connectionState = "checking";
        root.streamAuthenticationFailed = false;
        streamProcess.command = ["python3", bridgePath, "stream", "--cookie", cookiePath, "--config", connectionPath];
        streamProcess.running = true;
    }

    function disconnect() {
        if (!bridgePath || disconnectProcess.running)
            return ;

        reconnectTimer.stop();
        streamProcess.running = false;
        disconnectProcess.command = ["python3", bridgePath, "disconnect", "--cookie", cookiePath, "--config", connectionPath];
        disconnectProcess.running = true;
    }

    function accessForList(listId) {
        return TendModel.accessForList(accesses, listId);
    }

    function canEditList(listId) {
        return TendModel.canEditList(accesses, listId);
    }

    function listMutationPending(listId) {
        for (var i = 0; i < pendingOperations.length; i++) {
            if (pendingOperations[i].listId === Number(listId))
                return true;

        }
        return false;
    }

    function submit(action, listId) {
        if (connectionState !== "online" || mutationPending || pokeProcess.running)
            return false;

        if (listId !== undefined && listId !== null && !canEditList(listId)) {
            var access = accessForList(listId);
            var host = access ? access.host : "the list owner";
            root.errorMessage = host + " is " + (access ? access.status : "checking") + "; this shared list is read-only until its host is online.";
            return false;
        }

        root.mutationPending = true;
        root.errorMessage = "";
        pokeProcess.payload = JSON.stringify(action);
        pokeProcess.command = ["python3", bridgePath, "poke", "--cookie", cookiePath, "--config", connectionPath];
        pokeProcess.running = true;
        return true;
    }

    function createList(title) {
        return submit({
            "create-list": {
                "operation-id": operationId(),
                "title": String(title || "").trim()
            }
        });
    }

    function renameList(listId, title, baseRevision) {
        return submit({
            "rename-list": {
                "operation-id": operationId(),
                "list-id": Number(listId),
                "title": String(title || "").trim(),
                "base-revision": Number(baseRevision)
            }
        }, listId);
    }

    function updateList(listId, title, color, symbol, baseRevision) {
        return submit({
            "update-list": {
                "operation-id": operationId(),
                "list-id": Number(listId),
                "title": String(title || "").trim(),
                "color": String(color || "").trim(),
                "symbol": String(symbol || "").trim(),
                "base-revision": Number(baseRevision)
            }
        }, listId);
    }

    function deleteList(listId, baseRevision) {
        return submit({
            "delete-list": {
                "operation-id": operationId(),
                "list-id": Number(listId),
                "base-revision": Number(baseRevision)
            }
        }, listId);
    }

    function addSection(listId, title, rank, baseRevision) {
        return submit({
            "add-section": {
                "operation-id": operationId(),
                "list-id": Number(listId),
                "title": String(title || "").trim(),
                "rank": Number(rank),
                "base-revision": Number(baseRevision)
            }
        }, listId);
    }

    function updateSection(listId, sectionId, title, rank, baseRevision) {
        return submit({
            "update-section": {
                "operation-id": operationId(),
                "list-id": Number(listId),
                "section-id": Number(sectionId),
                "title": String(title || "").trim(),
                "rank": Number(rank),
                "base-revision": Number(baseRevision)
            }
        }, listId);
    }

    function deleteSection(listId, sectionId, baseRevision) {
        return submit({
            "delete-section": {
                "operation-id": operationId(),
                "list-id": Number(listId),
                "section-id": Number(sectionId),
                "base-revision": Number(baseRevision)
            }
        }, listId);
    }

    function addReminder(listId, title, baseRevision, tags) {
        return submit({
            "add-reminder": {
                "operation-id": operationId(),
                "list-id": Number(listId),
                "title": String(title || "").trim(),
                "tags": (tags || []).map(String),
                "base-revision": Number(baseRevision)
            }
        }, listId);
    }

    function replaceTag(from, to) {
        return submit({
            "replace-tag": {
                "operation-id": operationId(),
                "from": String(from || "").trim(),
                "to": to === null || to === undefined ? null : String(to).trim()
            }
        });
    }

    function setCompleted(listId, reminderId, completed, baseRevision) {
        return submit({
            "set-completed": {
                "operation-id": operationId(),
                "list-id": Number(listId),
                "reminder-id": Number(reminderId),
                "completed": completed === true,
                "base-revision": Number(baseRevision)
            }
        }, listId);
    }

    function batchSetCompleted(listId, reminderIds, completed, baseRevision) {
        return submit({
            "batch-set-completed": {
                "operation-id": operationId(),
                "list-id": Number(listId),
                "reminder-ids": (reminderIds || []).map(Number),
                "completed": completed === true,
                "base-revision": Number(baseRevision)
            }
        }, listId);
    }

    function batchDeleteReminders(listId, reminderIds, baseRevision) {
        return submit({
            "batch-delete-reminders": {
                "operation-id": operationId(),
                "list-id": Number(listId),
                "reminder-ids": (reminderIds || []).map(Number),
                "base-revision": Number(baseRevision)
            }
        }, listId);
    }

    function batchMoveReminders(listId, reminderIds, sectionId, startingRank, baseRevision) {
        return submit({
            "batch-move-reminders": {
                "operation-id": operationId(),
                "list-id": Number(listId),
                "reminder-ids": (reminderIds || []).map(Number),
                "section-id": sectionId === null || sectionId === undefined ? null : Number(sectionId),
                "starting-rank": Number(startingRank || 0),
                "base-revision": Number(baseRevision)
            }
        }, listId);
    }

    function setSchedule(listId, reminderId, schedule, baseRevision) {
        var value = null;
        if (schedule) {
            var allDay = schedule.allDay === true || schedule["all-day"] === true;
            var dueInput = schedule.dueAt || schedule["due-at"];
            if (allDay && String(dueInput || "").charAt(0) !== "~") {
                var localDue = new Date(dueInput);
                if (!isNaN(localDue.getTime())) {
                    var minute = Math.max(0, Math.min(1439, Number(preferences.allDayAlertMinute || 0)));
                    localDue.setHours(Math.floor(minute / 60), minute % 60, 0, 0);
                    dueInput = localDue.toISOString();
                }
            }
            value = {
                "due-at": toUrbitDate(dueInput),
                "all-day": allDay,
                "timezone": String(schedule.timezone || "UTC"),
                "early-seconds": (schedule.earlySeconds || schedule["early-seconds"] || []).map(Number),
                "recurrence": recurrencePayload(schedule.recurrence)
            };
        }
        return submit({
            "set-schedule": {
                "operation-id": operationId(),
                "list-id": Number(listId),
                "reminder-id": Number(reminderId),
                "schedule": value,
                "base-revision": Number(baseRevision)
            }
        }, listId);
    }

    function setPreferences(fields) {
        fields = fields || {
        };
        return submit({
            "set-preferences": {
                "operation-id": operationId(),
                "default-list": fields.defaultList === null || fields.defaultList === undefined ? null : Number(fields.defaultList),
                "pinned-lists": (fields.pinnedLists || []).map(Number),
                "pinned-views": (fields.pinnedViews || []).map(String),
                "snooze-presets": (fields.snoozePresets || []).map(Number),
                "base-revision": Number(preferences.revision || 0)
            }
        });
    }

    function setReminderPolicy(fields) {
        fields = fields || {
        };
        return submit({
            "set-reminder-policy": {
                "operation-id": operationId(),
                "badge-mode": String(fields.badgeMode || "today"),
                "all-day-alert-minute": Math.max(0, Math.min(1439, Number(fields.allDayAlertMinute || 0))),
                "all-day-overdue": fields.allDayOverdue !== false,
                "base-revision": Number(preferences.revision || 0)
            }
        });
    }

    function snoozeReminder(listId, reminderId, seconds) {
        var until = new Date(Date.now() + Math.max(1, Number(seconds || 0)) * 1000);
        return submit({
            "snooze-reminder": {
                "operation-id": operationId(),
                "list-id": Number(listId),
                "reminder-id": Number(reminderId),
                "until": toUrbitDate(until.toISOString())
            }
        });
    }

    function updateReminder(listId, reminderId, fields, baseRevision) {
        fields = fields || {
        };
        return submit({
            "update-reminder": {
                "operation-id": operationId(),
                "list-id": Number(listId),
                "reminder-id": Number(reminderId),
                "title": String(fields.title || "").trim(),
                "notes": String(fields.notes || ""),
                "url": fields.url ? String(fields.url) : null,
                "priority": String(fields.priority || "none"),
                "flagged": fields.flagged === true,
                "tags": (fields.tags || []).map(String),
                "assignee": fields.assignee ? String(fields.assignee) : null,
                "base-revision": Number(baseRevision)
            }
        }, listId);
    }

    function inviteMember(listId, targetShip, canInvite) {
        return submit({
            "invite-member": {
                "operation-id": operationId(),
                "list-id": Number(listId),
                "ship": String(targetShip || "").trim(),
                "can-invite": canInvite === true
            }
        }, listId);
    }

    function acceptInvitation(invitation) {
        return submit({
            "accept-invitation": {
                "operation-id": operationId(),
                "host": String(invitation.host || ""),
                "token": String(invitation.token || "")
            }
        });
    }

    function declineInvitation(invitation) {
        return submit({
            "decline-invitation": {
                "operation-id": operationId(),
                "host": String(invitation.host || ""),
                "token": String(invitation.token || "")
            }
        });
    }

    function removeMember(listId, targetShip) {
        return submit({
            "remove-member": {
                "operation-id": operationId(),
                "list-id": Number(listId),
                "ship": String(targetShip || "")
            }
        }, listId);
    }

    function leaveSharedList(listId) {
        return submit({
            "leave-shared-list": {
                "operation-id": operationId(),
                "list-id": Number(listId)
            }
        });
    }

    function moveReminder(listId, reminderId, parentId, sectionId, rank, baseRevision) {
        return submit({
            "move-reminder": {
                "operation-id": operationId(),
                "list-id": Number(listId),
                "reminder-id": Number(reminderId),
                "parent-id": parentId === null || parentId === undefined ? null : Number(parentId),
                "section-id": sectionId === null || sectionId === undefined ? null : Number(sectionId),
                "rank": Number(rank),
                "base-revision": Number(baseRevision)
            }
        }, listId);
    }

    function deleteReminder(listId, reminderId, baseRevision) {
        return submit({
            "delete-reminder": {
                "operation-id": operationId(),
                "list-id": Number(listId),
                "reminder-id": Number(reminderId),
                "base-revision": Number(baseRevision)
            }
        }, listId);
    }

    function handleStreamLine(line) {
        try {
            var envelope = JSON.parse(String(line || ""));
            var message = envelope.message || {
            };
            if (message.response === "subscribe" && message.ok !== undefined) {
                root.connectionState = "checking";
                return ;
            }
            if (message.response !== "diff")
                return ;

            var result = TendModel.reduce(root.lists, message.json, root.preferences, root.snoozes, root.accesses, root.invitations, root.pendingOperations);
            root.lists = result.lists;
            root.preferences = result.preferences;
            root.snoozes = result.snoozes;
            root.accesses = result.accesses;
            root.invitations = result.invitations;
            root.pendingOperations = result.pendingOperations;
            if (message.json && message.json.snapshot)
                root.connectionState = "online";
            if (message.json && message.json.snapshot)
                root.reconnectAttempt = 0;

            if (result.error)
                root.errorMessage = result.error;

            if (result.alert) {
                root.lastAlert = result.alert;
                root.reminderAlert(result.alert);
                root.showNotification(result.alert);
            }

        } catch (error) {
            root.errorMessage = "Could not parse an Eyre event: " + error;
        }
    }

    function processError(raw, fallback) {
        return parseError(raw, fallback).message;
    }

    function parseError(raw, fallback) {
        try {
            var parsed = JSON.parse(String(raw || ""));
            return {
                code: String(parsed.code || "transport-error"),
                message: String(parsed.message || fallback)
            };
        } catch (error) {
            return {
                code: "transport-error",
                message: String(raw || fallback).trim()
            };
        }
    }

    function recordStreamError(raw) {
        var failure = parseError(raw, "Eyre stream failed");
        root.errorMessage = failure.message;
        if (failure.code === "authentication-required") {
            root.streamAuthenticationFailed = true;
            root.connectionState = "authentication-required";
        }
    }

    function scheduleReconnect() {
        reconnectAttempt += 1;
        reconnectTimer.interval = Math.min(300000, 5000 * Math.pow(2, reconnectAttempt - 1));
        reconnectTimer.restart();
    }

    function showNotification(alert) {
        var queue = notificationQueue.slice();
        queue.push(alert);
        notificationQueue = queue;
        pumpNotificationQueue();
    }

    function listById(listId) {
        for (var i = 0; i < lists.length; i++) {
            if (lists[i].id === listId)
                return lists[i];

        }
        return null;
    }

    function reminderById(list, reminderId) {
        if (!list)
            return null;

        for (var i = 0; i < list.reminders.length; i++) {
            if (list.reminders[i].id === reminderId)
                return list.reminders[i];

        }
        return null;
    }

    function pumpNotificationQueue() {
        if (notificationProcess.running || notificationQueue.length === 0)
            return ;

        var queue = notificationQueue.slice();
        var alert = queue.shift();
        notificationQueue = queue;
        notificationProcess.alert = alert;
        notificationProcess.output = "";
        var title = "Reminder due";
        var list = listById(alert.listId);
        var reminder = reminderById(list, alert.reminderId);
        if (reminder)
            title = reminder.title;

        notificationProcess.command = ["notify-send", "--app-name=Tend", "--action=complete=Complete", "--action=snooze=Snooze", "--action=open=Open", alert.snoozed ? "Tend · Snoozed reminder" : "Tend", title];
        notificationProcess.running = true;
    }

    onManifestChanged: {
        if (manifest) {
            Qt.callLater(restore);
        }
    }

    Process {
        id: statusProcess

        property string output: ""
        property string errors: ""

        onExited: function(exitCode) {
            if (exitCode !== 0)
                return ;

            try {
                var status = JSON.parse(output);
                root.baseUrl = status.baseUrl || "";
                root.ship = status.ship || "";
                if (status.authenticated)
                    root.startStream();

            } catch (error) {
                root.errorMessage = "Could not read the saved Tend connection";
            }
        }

        stdout: StdioCollector {
            waitForEnd: true
            onStreamFinished: statusProcess.output = text
        }

        stderr: StdioCollector {
            waitForEnd: true
            onStreamFinished: statusProcess.errors = text
        }

    }

    Process {
        id: loginProcess

        property string secret: ""
        property string output: ""
        property string errors: ""

        stdinEnabled: true
        onStarted: {
            write(secret + "\n");
            secret = "";
        }
        onExited: function(exitCode) {
            if (exitCode !== 0) {
                root.connectionState = "error";
                root.errorMessage = root.processError(errors, "Could not authenticate to Eyre");
                return ;
            }
            try {
                var result = JSON.parse(output);
                root.baseUrl = result.baseUrl;
                root.ship = result.ship;
                root.startStream();
            } catch (error) {
                root.connectionState = "error";
                root.errorMessage = "Eyre returned an invalid login response";
            }
        }

        stdout: StdioCollector {
            waitForEnd: true
            onStreamFinished: loginProcess.output = text
        }

        stderr: StdioCollector {
            waitForEnd: true
            onStreamFinished: loginProcess.errors = text
        }

    }

    Process {
        id: streamProcess

        onExited: function() {
            if (root.streamAuthenticationFailed) {
                root.connectionState = "authentication-required";
                return ;
            }
            if (root.ship) {
                root.connectionState = "offline";
                root.scheduleReconnect();
            } else {
                root.connectionState = "disconnected";
            }
        }

        stdout: SplitParser {
            onRead: function(line) {
                root.handleStreamLine(line);
            }
        }

        stderr: SplitParser {
            onRead: function(line) {
                root.recordStreamError(line);
            }
        }

    }

    Process {
        id: disconnectProcess

        property string errors: ""

        onExited: function(exitCode) {
            if (exitCode !== 0) {
                root.errorMessage = root.processError(errors, "Could not disconnect Tend");
                return ;
            }
            root.ship = "";
            root.baseUrl = "";
            root.lists = [];
            root.preferences = TendModel.clonePreferences(null);
            root.snoozes = [];
            root.accesses = [];
            root.invitations = [];
            root.pendingOperations = [];
            root.connectionState = "disconnected";
            root.errorMessage = "";
            root.reconnectAttempt = 0;
            root.streamAuthenticationFailed = false;
        }

        stderr: StdioCollector {
            waitForEnd: true
            onStreamFinished: disconnectProcess.errors = text
        }

    }

    Process {
        id: pokeProcess

        property string payload: ""
        property string errors: ""

        stdinEnabled: true
        onStarted: {
            write(payload);
            payload = "";
        }
        onExited: function(exitCode) {
            root.mutationPending = false;
            if (exitCode !== 0)
                root.errorMessage = root.processError(errors, "Tend action failed");

        }

        stderr: StdioCollector {
            waitForEnd: true
            onStreamFinished: pokeProcess.errors = text
        }

    }

    Process {
        id: notificationProcess

        property var alert: null
        property string output: ""

        onExited: function() {
            var action = output.trim();
            var list = root.listById(alert ? alert.listId : 0);
            var reminder = root.reminderById(list, alert ? alert.reminderId : 0);
            if (action === "complete" && list && reminder && root.connectionState === "online")
                root.setCompleted(list.id, reminder.id, true, list.revision);
            else if (action === "snooze" && list && reminder && root.connectionState === "online")
                root.snoozeReminder(list.id, reminder.id, Number(root.preferences.snoozePresets[0] || 300));
            else if (action === "open" && root.shell)
                root.shell.toggle("io.omabit.tend", JSON.stringify({
                    listId: alert.listId,
                    reminderId: alert.reminderId
                }));

            alert = null;
            Qt.callLater(root.pumpNotificationQueue);
        }

        stdout: StdioCollector {
            waitForEnd: true
            onStreamFinished: notificationProcess.output = text
        }
    }

    Timer {
        id: reconnectTimer

        interval: 5000
        repeat: false
        onTriggered: root.startStream()
    }

}
