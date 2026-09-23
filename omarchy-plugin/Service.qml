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
    property string localTimezone: "UTC"
    property var lists: []
    property var preferences: TendModel.clonePreferences(null)
    property var snoozes: []
    property var accesses: []
    property var invitations: []
    property var pendingOperations: []
    property var activities: []
    property var localSettings: TendModel.cloneLocalSettings(null)
    property var pendingDetailedAdds: []
    property var pendingPostAddUpdates: []
    property var pendingCompletionChanges: []
    property double completionClock: Date.now()
    property var lastAlert: null
    property var notificationQueue: []
    property var notificationAckQueue: []
    property bool mutationPending: false
    property string lastInvitationUri: ""
    property int lastInvitationListId: 0
    property string lastInvitationTarget: ""
    property int reconnectAttempt: 0
    property bool streamAuthenticationFailed: false
    readonly property int incompleteCount: TendModel.incompleteCount(lists)
    readonly property int badgeCount: TendModel.badgeCount(lists, preferences, ship)
    readonly property var nextReminder: TendModel.nextReminder(lists)
    // Omarchy deliberately removes private registry fields such as
    // `__sourceDir` before exposing a third-party manifest. Resolve the
    // bundled bridge relative to this component instead of depending on host
    // implementation details that are unavailable to the service at runtime.
    readonly property string bridgePath: {
        var resolved = String(Qt.resolvedUrl("transport/eyre_client.py"));
        return resolved.indexOf("file://") === 0 ? decodeURIComponent(resolved.substring(7)) : resolved;
    }
    readonly property string runtimeRoot: (Quickshell.env("XDG_RUNTIME_DIR") || (Quickshell.env("HOME") + "/.cache")) + "/omabit/tend"
    readonly property string configRoot: (Quickshell.env("XDG_CONFIG_HOME") || (Quickshell.env("HOME") + "/.config")) + "/omabit/tend"
    readonly property string cookiePath: runtimeRoot + "/cookies.txt"
    readonly property string connectionPath: configRoot + "/connection.json"
    signal reminderAlert(var alert)
    signal loginSucceeded()

    function operationId() {
        return Date.now().toString(36) + "-" + Math.floor(Math.random() * 2.14748e+09).toString(36);
    }

    function normalizeShip(value) {
        var name = String(value || "").trim().replace(/^~/, "");
        return name ? "~" + name : null;
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
            "end-at": recurrence.endAt || recurrence["end-at"] || null,
            "max-occurrences": recurrence.maxOccurrences === undefined ? (recurrence["max-occurrences"] ?? null) : recurrence.maxOccurrences
        };
    }

    function login(url, code) {
        if (!bridgePath) {
            root.connectionState = "error";
            root.errorMessage = "The bundled Tend transport could not be located";
            return ;
        }
        if (loginProcess.running)
            return ;

        root.errorMessage = "";
        root.connectionState = "authenticating";
        loginProcess.output = "";
        loginProcess.errors = "";
        loginProcess.secret = String(code || "");
        loginProcess.command = ["python3", "-B", bridgePath, "login", "--url", String(url || ""), "--cookie", cookiePath, "--config", connectionPath];
        loginProcess.running = true;
    }

    function restore() {
        if (!bridgePath || statusProcess.running)
            return ;

        statusProcess.command = ["python3", "-B", bridgePath, "status", "--cookie", cookiePath, "--config", connectionPath];
        statusProcess.running = true;
    }

    function startStream() {
        if (!bridgePath || !ship || streamProcess.running)
            return ;

        root.connectionState = "checking";
        root.streamAuthenticationFailed = false;
        streamProcess.command = ["python3", "-B", bridgePath, "stream", "--cookie", cookiePath, "--config", connectionPath];
        streamProcess.running = true;
    }

    function disconnect() {
        if (!bridgePath || disconnectProcess.running)
            return ;

        reconnectTimer.stop();
        streamProcess.running = false;
        disconnectProcess.command = ["python3", "-B", bridgePath, "disconnect", "--cookie", cookiePath, "--config", connectionPath];
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
        pokeProcess.command = ["python3", "-B", bridgePath, "poke", "--cookie", cookiePath, "--config", connectionPath];
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

    function placeSection(listId, sectionId, targetId, after, baseRevision) {
        return submit({
            "place-section": {
                "operation-id": operationId(),
                "list-id": Number(listId),
                "section-id": Number(sectionId),
                "target-id": Number(targetId),
                "after": after === true,
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

    function addReminderWithDetails(listId, title, baseRevision, fields) {
        var list = listById(Number(listId));
        fields = fields || {};
        if (!list)
            return false;
        var opId = operationId();
        var existingIds = list.reminders.map(function(reminder) { return Number(reminder.id); });
        var reminderTitle = String(title || "").trim();
        var accepted = submit({
            "add-reminder": {
                "operation-id": opId,
                "list-id": Number(listId),
                "title": reminderTitle,
                "tags": (fields.tags || []).map(String),
                "base-revision": Number(baseRevision)
            }
        }, listId);
        if (accepted) {
            var queued = pendingDetailedAdds.slice();
            queued.push({
                operationId: opId,
                listId: Number(listId),
                title: reminderTitle,
                existingIds: existingIds,
                assignee: normalizeShip(fields.assignee),
                due: String(fields.due || "").trim(),
                allDay: fields.allDay === true,
                timezone: String(fields.timezone || localTimezone || "UTC")
            });
            pendingDetailedAdds = queued;
        }
        return accepted;
    }

    function addReminderAssignedToMe(listId, title, baseRevision, tags) {
        return addReminderWithDetails(listId, title, baseRevision, {
            tags: tags || [],
            assignee: normalizeShip(ship)
        });
    }

    function queueDetailsForCreatedReminder(update) {
        if (!update || !update["list-upserted"])
            return;
        var body = update["list-upserted"];
        var opId = String(body["operation-id"] || "");
        var pendingIndex = -1;
        for (var i = 0; i < pendingDetailedAdds.length; i++) {
            if (pendingDetailedAdds[i].operationId === opId) {
                pendingIndex = i;
                break;
            }
        }
        if (pendingIndex < 0)
            return;

        var pending = pendingDetailedAdds[pendingIndex];
        var list = listById(pending.listId);
        var created = null;
        if (list) {
            for (var j = 0; j < list.reminders.length; j++) {
                var candidate = list.reminders[j];
                if (pending.existingIds.indexOf(Number(candidate.id)) === -1 && candidate.title === pending.title) {
                    created = candidate;
                    break;
                }
            }
        }
        var remaining = pendingDetailedAdds.slice();
        remaining.splice(pendingIndex, 1);
        pendingDetailedAdds = remaining;
        if (!created)
            return;

        var updates = pendingPostAddUpdates.slice();
        if (pending.assignee) {
            updates.push({
                kind: "metadata",
                listId: pending.listId,
                reminderId: Number(created.id),
                assignee: pending.assignee
            });
        }
        if (pending.due) {
            updates.push({
                kind: "schedule",
                listId: pending.listId,
                reminderId: Number(created.id),
                due: pending.due,
                allDay: pending.allDay,
                timezone: pending.timezone
            });
        }
        pendingPostAddUpdates = updates;
        postAddTimer.restart();
    }

    function pumpPostAddUpdates() {
        if (mutationPending || connectionState !== "online" || pendingPostAddUpdates.length === 0)
            return;
        var queued = pendingPostAddUpdates.slice();
        var update = queued[0];
        var list = listById(update.listId);
        var reminder = reminderById(list, update.reminderId);
        if (!list || !reminder) {
            queued.shift();
            pendingPostAddUpdates = queued;
            return;
        }
        if (listMutationPending(list.id))
            return;
        var accepted = false;
        if (update.kind === "metadata") {
            accepted = updateReminder(list.id, reminder.id, {
                title: reminder.title,
                notes: reminder.notes,
                url: reminder.url,
                priority: reminder.priority,
                flagged: reminder.flagged,
                tags: reminder.tags,
                assignee: update.assignee
            }, list.revision);
        } else if (update.kind === "schedule") {
            accepted = setSchedule(list.id, reminder.id, {
                due: update.due,
                allDay: update.allDay,
                timezone: update.timezone,
                earlySeconds: [],
                recurrence: null
            }, list.revision);
        }
        if (accepted) {
            queued.shift();
            pendingPostAddUpdates = queued;
        }
    }

    function pendingCompletionIndex(listId, reminderId) {
        for (var i = 0; i < pendingCompletionChanges.length; i++) {
            var item = pendingCompletionChanges[i];
            if (item.listId === Number(listId) && item.reminderId === Number(reminderId))
                return i;
        }
        return -1;
    }

    function effectiveCompleted(listId, reminderId, confirmed) {
        var index = pendingCompletionIndex(listId, reminderId);
        return index < 0 ? confirmed === true : pendingCompletionChanges[index].completed;
    }

    function completionOpacity(listId, reminderId) {
        var index = pendingCompletionIndex(listId, reminderId);
        if (index < 0)
            return 1;
        var elapsed = Math.max(0, completionClock - pendingCompletionChanges[index].requestedAt);
        if (elapsed <= 5000)
            return 1;
        return Math.max(0, 1 - (elapsed - 5000) / 5000);
    }

    function scheduleCompleted(listId, reminderId, completed, confirmed) {
        var changes = pendingCompletionChanges.slice();
        var index = pendingCompletionIndex(listId, reminderId);
        if (completed === (confirmed === true)) {
            if (index >= 0) {
                changes.splice(index, 1);
                pendingCompletionChanges = changes;
            }
            return true;
        }
        var change = {
            listId: Number(listId),
            reminderId: Number(reminderId),
            completed: completed === true,
            requestedAt: Date.now(),
            submitted: false
        };
        if (index >= 0)
            changes[index] = change;
        else
            changes.push(change);
        pendingCompletionChanges = changes;
        completionTimer.start();
        return true;
    }

    function toggleCompletedWithGrace(listId, reminderId, confirmed) {
        return scheduleCompleted(listId, reminderId, !effectiveCompleted(listId, reminderId, confirmed), confirmed);
    }

    function pumpCompletionChanges() {
        completionClock = Date.now();
        if (connectionState !== "online" || mutationPending || pendingCompletionChanges.length === 0)
            return;
        for (var i = 0; i < pendingCompletionChanges.length; i++) {
            var change = pendingCompletionChanges[i];
            if (completionClock - change.requestedAt < 10000)
                continue;
            var list = listById(change.listId);
            var reminder = reminderById(list, change.reminderId);
            if (!list || !reminder) {
                var missing = pendingCompletionChanges.slice();
                missing.splice(i, 1);
                pendingCompletionChanges = missing;
                return;
            }
            if (listMutationPending(list.id))
                return;
            if (reminder.completed === change.completed) {
                var remaining = pendingCompletionChanges.slice();
                remaining.splice(i, 1);
                pendingCompletionChanges = remaining;
                return;
            }
            if (change.submitted)
                return;
            if (setCompleted(list.id, reminder.id, change.completed, list.revision)) {
                var submitted = pendingCompletionChanges.slice();
                submitted[i] = {
                    listId: change.listId,
                    reminderId: change.reminderId,
                    completed: change.completed,
                    requestedAt: change.requestedAt,
                    submitted: true
                };
                pendingCompletionChanges = submitted;
            }
            return;
        }
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
            var timezone = String(schedule.timezone || "UTC");
            var recurrence = recurrencePayload(schedule.recurrence);
            value = {
                "due-at": String(dueInput || "").trim(),
                "all-day": allDay,
                "timezone": timezone,
                "_all-day-alert-minute": Number(preferences.allDayAlertMinute || 0),
                "early-seconds": (schedule.earlySeconds || schedule["early-seconds"] || []).map(Number),
                "recurrence": recurrence
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

    function setListOrder(listIds) {
        return submit({
            "set-list-order": {
                "operation-id": operationId(),
                "list-ids": (listIds || []).map(Number)
            }
        });
    }

    function setListPresentation(listId, sort, descending) {
        return submit({
            "set-list-presentation": {
                "operation-id": operationId(),
                "list-id": Number(listId),
                "sort": String(sort || "manual"),
                "descending": descending === true
            }
        });
    }

    function setCollaborationPolicy(listId, notifyAdded, notifyCompleted, notifyAssigned) {
        return submit({
            "set-collaboration-policy": {
                "operation-id": operationId(),
                "list-id": Number(listId),
                "notify-added": notifyAdded === true,
                "notify-completed": notifyCompleted === true,
                "notify-assigned": notifyAssigned === true
            }
        });
    }

    function snoozeReminder(listId, reminderId, seconds) {
        var until = new Date(Date.now() + Math.max(1, Number(seconds || 0)) * 1000);
        return snoozeReminderUntil(listId, reminderId, until.toISOString());
    }

    function snoozeReminderUntil(listId, reminderId, until) {
        var source = String(until || "").trim();
        if (source.charAt(0) !== "~") {
            var instant = new Date(source);
            if (isNaN(instant.getTime()) || instant.getTime() <= Date.now()) {
                root.errorMessage = "Enter a valid future date and time for snooze.";
                return false;
            }
        }
        var encoded = toUrbitDate(source);
        if (!encoded || encoded.charAt(0) !== "~") {
            root.errorMessage = "Enter a valid future date and time for snooze.";
            return false;
        }
        return submit({
            "snooze-reminder": {
                "operation-id": operationId(),
                "list-id": Number(listId),
                "reminder-id": Number(reminderId),
                "until": encoded
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
                "assignee": normalizeShip(fields.assignee),
                "base-revision": Number(baseRevision)
            }
        }, listId);
    }

    function inviteMember(listId, targetShip, canInvite) {
        var opId = operationId();
        var accepted = submit({
            "invite-member": {
                "operation-id": opId,
                "list-id": Number(listId),
                "ship": String(targetShip || "").trim(),
                "can-invite": canInvite === true
            }
        }, listId);
        if (accepted) {
            var access = accessForList(listId);
            var owner = String(access ? access.host : "").replace(/^~/, "");
            root.lastInvitationUri = "omabit://tend/invite/" + owner + "/" + encodeURIComponent(opId);
            root.lastInvitationListId = Number(listId);
            root.lastInvitationTarget = String(targetShip || "").trim();
        }
        return accepted;
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

    function placeReminder(listId, reminderId, targetId, after, baseRevision) {
        return submit({
            "place-reminder": {
                "operation-id": operationId(),
                "list-id": Number(listId),
                "reminder-id": Number(reminderId),
                "target-id": Number(targetId),
                "after": after === true,
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

            if (message.json && message.json["activities-updated"]) {
                root.activities = TendModel.reduceActivities(root.activities, message.json);
                return ;
            }

            if (message.json && message.json["local-settings-updated"]) {
                root.localSettings = TendModel.reduceLocalSettings(root.localSettings, message.json);
                return ;
            }

            var result = TendModel.reduce(root.lists, message.json, root.preferences, root.snoozes, root.accesses, root.invitations, root.pendingOperations);
            root.lists = result.lists;
            root.preferences = result.preferences;
            root.snoozes = result.snoozes;
            root.accesses = result.accesses;
            root.invitations = result.invitations;
            root.pendingOperations = result.pendingOperations;
            root.queueDetailsForCreatedReminder(message.json);
            if (message.json && message.json.rejected) {
                var rejectedOp = String(message.json.rejected["operation-id"] || "");
                root.pendingDetailedAdds = root.pendingDetailedAdds.filter(function(entry) {
                    return entry.operationId !== rejectedOp;
                });
            }
            if (message.json && message.json["list-deleted"]) {
                var deletedListId = Number(message.json["list-deleted"]["list-id"]);
                root.activities = root.activities.filter(function(entry) {
                    return Number(entry.listId) !== deletedListId;
                });
            }
            if (message.json && message.json.snapshot)
                root.connectionState = "online";
            if (message.json && message.json.snapshot)
                root.reconnectAttempt = 0;
            if (message.json && message.json.snapshot)
                Qt.callLater(root.pumpNotificationAckQueue);

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
        var notificationId = String(alert.notificationId || "");
        if (notificationId && notificationProcess.alert && String(notificationProcess.alert.notificationId || "") === notificationId)
            return ;

        var queue = notificationQueue.slice();
        for (var i = 0; notificationId && i < queue.length; i++) {
            if (String(queue[i].notificationId || "") === notificationId)
                return ;
        }
        queue.push(alert);
        notificationQueue = queue;
        pumpNotificationQueue();
    }

    function acknowledgeNotification(notificationId) {
        var id = String(notificationId || "");
        if (!id)
            return ;

        if (notificationAckProcess.notificationId === id)
            return ;

        var queue = notificationAckQueue.slice();
        if (queue.indexOf(id) === -1)
            queue.push(id);
        notificationAckQueue = queue;
        pumpNotificationAckQueue();
    }

    function pumpNotificationAckQueue() {
        if (connectionState !== "online" || notificationAckProcess.running || notificationAckQueue.length === 0)
            return ;

        var queue = notificationAckQueue.slice();
        var notificationId = queue.shift();
        notificationAckQueue = queue;
        notificationAckProcess.notificationId = notificationId;
        notificationAckProcess.errors = "";
        notificationAckProcess.payload = JSON.stringify({
            "ack-notification": {
                "operation-id": "ack-" + notificationId,
                "notification-id": notificationId
            }
        });
        notificationAckProcess.command = ["python3", "-B", bridgePath, "poke", "--cookie", cookiePath, "--config", connectionPath];
        notificationAckProcess.running = true;
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
        if (alert.type === "collaboration") {
            var actor = String(alert.actor || "A collaborator");
            var summary = alert.kind === "completed" ? actor + " completed" : alert.kind === "assigned" ? actor + " assigned to you" : actor + " added";
            notificationProcess.command = ["notify-send", "--app-name=Tend", "--action=open=Open", "Tend · " + summary, title];
        } else {
            notificationProcess.command = ["notify-send", "--app-name=Tend", "--action=complete=Complete", "--action=snooze=Snooze", "--action=open=Open", alert.snoozed ? "Tend · Snoozed reminder" : "Tend", title];
        }
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
                root.localTimezone = status.localTimezone || "UTC";
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
                root.localTimezone = result.localTimezone || "UTC";
                root.loginSucceeded();
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
            root.localTimezone = "UTC";
            root.lists = [];
            root.preferences = TendModel.clonePreferences(null);
            root.snoozes = [];
            root.accesses = [];
            root.invitations = [];
            root.pendingOperations = [];
            root.pendingDetailedAdds = [];
            root.pendingPostAddUpdates = [];
            root.pendingCompletionChanges = [];
            root.activities = [];
            root.localSettings = TendModel.cloneLocalSettings(null);
            root.notificationQueue = [];
            root.notificationAckQueue = [];
            root.lastInvitationUri = "";
            root.lastInvitationListId = 0;
            root.lastInvitationTarget = "";
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
            write(payload + "\n");
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

        onExited: function(exitCode) {
            var action = output.trim();
            var list = root.listById(alert ? alert.listId : 0);
            var reminder = root.reminderById(list, alert ? alert.reminderId : 0);
            if (exitCode === 0 && alert)
                root.acknowledgeNotification(alert.notificationId);
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

    Process {
        id: notificationAckProcess

        property string notificationId: ""
        property string payload: ""
        property string errors: ""

        stdinEnabled: true
        onStarted: {
            write(payload + "\n");
            payload = "";
        }
        onExited: function(exitCode) {
            var failedId = notificationId;
            notificationId = "";
            if (exitCode !== 0 && failedId) {
                var queue = root.notificationAckQueue.slice();
                if (queue.indexOf(failedId) === -1)
                    queue.unshift(failedId);
                root.notificationAckQueue = queue;
                notificationAckRetryTimer.restart();
                return ;
            }
            Qt.callLater(root.pumpNotificationAckQueue);
        }

        stderr: StdioCollector {
            waitForEnd: true
            onStreamFinished: notificationAckProcess.errors = text
        }
    }

    Timer {
        id: postAddTimer
        interval: 150
        repeat: pendingPostAddUpdates.length > 0
        running: repeat
        onTriggered: root.pumpPostAddUpdates()
    }

    Timer {
        id: completionTimer
        interval: 100
        repeat: pendingCompletionChanges.length > 0
        running: repeat
        onTriggered: root.pumpCompletionChanges()
    }

    Timer {
        id: reconnectTimer

        interval: 5000
        repeat: false
        onTriggered: root.startStream()
    }

    Timer {
        id: notificationAckRetryTimer

        interval: 5000
        repeat: false
        onTriggered: root.pumpNotificationAckQueue()
    }

}
