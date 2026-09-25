const test = require("node:test")
const assert = require("node:assert/strict")
const model = require("../omarchy-plugin/TendModel.js")

test("snapshot is normalized and incomplete reminders are counted", () => {
  const result = model.reduce([], {
    snapshot: {
      lists: [{
        id: 2,
        title: "Work",
        color: "#ef4444",
        symbol: "briefcase",
        revision: 3,
        sections: [
          { id: 8, title: "Later", rank: 20 },
          { id: 7, title: "Now", rank: 10 }
        ],
        reminders: [
          { id: 4, title: "Done", completed: true, revision: 2, rank: 20 },
          {
            id: 3,
            title: "Open",
            notes: "Details",
            priority: "high",
            flagged: true,
            tags: ["work"],
            completed: false,
            revision: 1,
            rank: 10,
            schedule: {
              "due-at": "~2026.9.11..21.00.00",
              "all-day": false,
              timezone: "America/Los_Angeles",
              "early-seconds": [900],
              occurrence: 1,
              recurrence: {
                frequency: "daily",
                interval: 1,
                weekdays: [],
                "month-days": [],
                "month-week": null,
                "end-at": null,
                "max-occurrences": 5
              }
            }
          }
        ]
      }]
    }
  })

  assert.equal(result.error, "")
  assert.deepEqual(result.lists[0].sections.map((item) => item.id), [7, 8])
  assert.deepEqual(result.lists[0].reminders.map((item) => item.id), [3, 4])
  assert.equal(result.lists[0].reminders[0].priority, "high")
  assert.equal(result.lists[0].reminders[0].flagged, true)
  assert.equal(result.lists[0].reminders[0].schedule.timezone, "America/Los_Angeles")
  assert.equal(result.lists[0].reminders[0].schedule.recurrence.maxOccurrences, 5)
  assert.equal(model.incompleteCount(result.lists), 1)
  assert.deepEqual(result.preferences.pinnedViews, ["today", "scheduled", "all", "flagged", "assigned", "completed"])
  assert.equal(result.preferences.badgeMode, "today")
  assert.equal(result.preferences.allDayAlertMinute, 540)
  assert.equal(result.preferences.allDayOverdue, true)
})

test("alerts preserve list state and expose normalized notification data", () => {
  const initial = [{ id: 1, title: "Inbox", revision: 1, reminders: [] }]
  const result = model.reduce(initial, {
    alert: {
      "notification-id": "notice-1",
      "list-id": 1,
      "reminder-id": 9,
      "due-at": "~2026.9.10..21.00.00",
      "early-seconds": 900
    }
  })

  assert.deepEqual(result.lists, model.sortedLists(initial))
  assert.deepEqual(result.alert, {
    notificationId: "notice-1",
    listId: 1,
    reminderId: 9,
    dueAt: "~2026.9.10..21.00.00",
    earlySeconds: 900,
    snoozed: false
  })
})

test("preferences, snoozes, and pinned list order reduce independently", () => {
  const lists = [
    { id: 1, title: "Home", revision: 1, reminders: [] },
    { id: 2, title: "Work", revision: 1, reminders: [] }
  ]
  const snapshot = model.reduce([], {
    snapshot: {
      lists,
      preferences: { revision: 3, "default-list": 2, "pinned-lists": [2], "pinned-views": ["today"], "snooze-presets": [600] },
      snoozes: [{ "list-id": 2, "reminder-id": 8, until: "~2026.9.10..22.00.00" }]
    }
  })

  assert.equal(snapshot.preferences.defaultList, 2)
  assert.deepEqual(model.orderedLists(snapshot.lists, snapshot.preferences.pinnedLists).map((item) => item.id), [2, 1])
  assert.equal(snapshot.snoozes[0].reminderId, 8)

  const updated = model.reduce(snapshot.lists, {
    "preferences-updated": {
      preferences: { revision: 4, "default-list": 1, "pinned-lists": [1], "pinned-views": ["all"], "snooze-presets": [300] }
    }
  }, snapshot.preferences, snapshot.snoozes)
  assert.equal(updated.preferences.defaultList, 1)
  assert.equal(updated.snoozes.length, 1)

  const fired = model.reduce(updated.lists, {
    alert: { "notification-id": "notice-2", "list-id": 2, "reminder-id": 8, "due-at": "~2026.9.10..20.00.00", "early-seconds": 0, snoozed: true }
  }, updated.preferences, updated.snoozes)
  assert.equal(fired.alert.snoozed, true)
  assert.equal(fired.snoozes.length, 0)
})

test("participant-local list order, presentation, and notification policy reduce independently", () => {
  const update = {
    "local-settings-updated": {
      "list-order": [3, 1, 2],
      presentations: [{ "list-id": 2, sort: "priority", descending: true }],
      "collaboration-policies": [{
        "list-id": 2,
        "notify-added": false,
        "notify-completed": true,
        "notify-assigned": false
      }]
    }
  }
  const settings = model.reduceLocalSettings(null, update)
  assert.deepEqual(settings.listOrder, [3, 1, 2])
  assert.deepEqual(model.presentationForList(settings, 2), { listId: 2, sort: "priority", descending: true })
  assert.deepEqual(model.presentationForList(settings, 99), { listId: 99, sort: "manual", descending: false })
  assert.deepEqual(model.collaborationPolicyForList(settings, 2), {
    listId: 2,
    notifyAdded: false,
    notifyCompleted: true,
    notifyAssigned: false
  })
  assert.equal(model.collaborationPolicyForList(settings, 99).notifyAssigned, true)

  const lists = [
    { id: 1, title: "One", revision: 1, reminders: [] },
    { id: 2, title: "Two", revision: 1, reminders: [] },
    { id: 3, title: "Three", revision: 1, reminders: [] }
  ]
  assert.deepEqual(model.orderedLists(lists, [2], settings.listOrder).map((item) => item.id), [2, 3, 1])
})

test("collaboration alerts preserve canonical state and expose actor context", () => {
  const initial = [{ id: 1, title: "Shared", revision: 4, reminders: [{ id: 7, title: "Milk" }] }]
  const result = model.reduce(initial, {
    "collaboration-alert": {
      "notification-id": "collab-1",
      "list-id": 1,
      "reminder-id": 7,
      actor: "~bus",
      kind: "completed"
    }
  })
  assert.equal(result.error, "")
  assert.equal(result.lists[0].revision, 4)
  assert.deepEqual(result.alert, {
    type: "collaboration",
    notificationId: "collab-1",
    listId: 1,
    reminderId: 7,
    actor: "~bus",
    kind: "completed"
  })
})

test("notification acknowledgements are harmless model events", () => {
  const initial = [{ id: 1, title: "Inbox", revision: 1, reminders: [] }]
  const result = model.reduce(initial, {
    "notification-acked": { "operation-id": "ack-1", "notification-id": "notice-1" }
  })

  assert.equal(result.error, "")
  assert.deepEqual(result.lists, model.sortedLists(initial))
  assert.equal(result.alert, null)
})

test("pinned smart views honor personal order and discard invalid duplicates", () => {
  const views = ["today", "scheduled", "all", "flagged", "assigned", "completed"]
  assert.deepEqual(
    model.orderedValues(views, ["assigned", "today", "assigned", "missing"]),
    ["assigned", "today", "scheduled", "all", "flagged", "completed"]
  )
})

test("deltas update one list without mutating the previous snapshot", () => {
  const initial = [{ id: 1, title: "Inbox", revision: 1, reminders: [] }]
  const added = model.reduce(initial, {
    "reminder-added": {
      "list-id": 1,
      "list-revision": 2,
      reminder: { id: 2, title: "Buy milk", completed: false, revision: 1 }
    }
  })

  assert.equal(initial[0].reminders.length, 0)
  assert.equal(added.lists[0].revision, 2)
  assert.equal(added.lists[0].reminders[0].title, "Buy milk")

  const completed = model.reduce(added.lists, {
    "reminder-completed": {
      "list-id": 1,
      "list-revision": 3,
      reminder: { id: 2, title: "Buy milk", completed: true, revision: 2 }
    }
  })
  assert.equal(completed.lists[0].reminders[0].completed, true)
  assert.equal(model.incompleteCount(completed.lists), 0)
})

test("rejections retain confirmed state and expose an error", () => {
  const initial = [{ id: 1, title: "Inbox", revision: 1, reminders: [] }]
  const result = model.reduce(initial, { rejected: { reason: "stale-list" } })
  assert.equal(result.lists[0].title, "Inbox")
  assert.equal(result.lists[0].revision, 1)
  assert.deepEqual(initial, [{ id: 1, title: "Inbox", revision: 1, reminders: [] }])
  assert.match(result.error, /stale-list/)
})

test("canonical list upserts and deletions replace only the addressed list", () => {
  const initial = [
    { id: 1, title: "Inbox", revision: 1, sections: [], reminders: [] },
    { id: 2, title: "Work", revision: 1, sections: [], reminders: [] }
  ]
  const upserted = model.reduce(initial, {
    "list-upserted": {
      "operation-id": "op-upsert",
      list: { id: 1, title: "Home", revision: 2, sections: [], reminders: [] }
    }
  })

  assert.deepEqual(upserted.lists.map((item) => item.title), ["Home", "Work"])
  assert.equal(initial[0].title, "Inbox")

  const deleted = model.reduce(upserted.lists, {
    "list-deleted": { "operation-id": "op-delete", "list-id": 1 }
  })
  assert.deepEqual(deleted.lists.map((item) => item.id), [2])
})

test("stale and duplicate facts cannot roll confirmed desktop state backward", () => {
  const current = [{
    id: 1,
    title: "Current",
    revision: 5,
    sections: [],
    reminders: [{ id: 9, title: "Confirmed", revision: 3, rank: 1 }]
  }]
  const preferences = { revision: 7, "default-list": 1, "pinned-lists": [1] }
  const pending = [{ operationId: "late-op", listId: 1 }]
  const stale = model.reduce(current, {
    "list-upserted": {
      "operation-id": "late-op",
      list: { id: 1, title: "Old", revision: 4, sections: [], reminders: [] },
      preferences: { revision: 6, "default-list": null, "pinned-lists": [] }
    }
  }, preferences, [], [], [], pending)

  assert.equal(stale.lists[0].title, "Current")
  assert.equal(stale.lists[0].reminders[0].title, "Confirmed")
  assert.equal(stale.preferences.revision, 7)
  assert.equal(stale.preferences.defaultList, 1)
  assert.deepEqual(stale.pendingOperations, [])

  const duplicate = model.reduce(stale.lists, {
    "list-upserted": {
      list: { id: 1, title: "Conflicting duplicate", revision: 5, sections: [], reminders: [] }
    }
  }, stale.preferences)
  assert.equal(duplicate.lists[0].title, "Current")

  const staleLegacy = model.reduce(duplicate.lists, {
    "reminder-completed": {
      "list-id": 1,
      "list-revision": 3,
      reminder: { id: 9, title: "Old reminder", completed: true, revision: 1 }
    }
  })
  assert.equal(staleLegacy.lists[0].revision, 5)
  assert.equal(staleLegacy.lists[0].reminders[0].title, "Confirmed")

  const stalePreferences = model.reduce(staleLegacy.lists, {
    "preferences-updated": {
      preferences: { revision: 2, "default-list": null, "pinned-lists": [] }
    }
  }, stale.preferences)
  assert.equal(stalePreferences.preferences.revision, 7)
  assert.equal(stalePreferences.preferences.defaultList, 1)
})

test("built-in views, search, and due sorting work across lists", () => {
  const lists = [
    {
      id: 1,
      title: "Home",
      revision: 4,
      reminders: [
        { id: 1, title: "Overdue milk", tags: ["shop"], rank: 30, flagged: true, completed: false, schedule: { "due-at": "~2026.9.9..18.00.00" } },
        { id: 2, title: "Tomorrow", rank: 20, completed: false, schedule: { "due-at": "~2026.9.11..18.00.00" } }
      ]
    },
    {
      id: 2,
      title: "Work",
      revision: 2,
      reminders: [
        { id: 3, title: "Ship report", notes: "September", rank: 10, completed: false, schedule: { "due-at": "~2026.9.10..20.00.00" } },
        { id: 4, title: "Filed", rank: 40, completed: true }
      ]
    }
  ]
  const now = Date.UTC(2026, 8, 10, 12)

  assert.deepEqual(model.queryReminders(lists, { view: "today", sort: "due" }, now).map((item) => item.id), [1, 3])
  assert.deepEqual(model.queryReminders(lists, { view: "scheduled", sort: "due" }, now).map((item) => item.id), [1, 3, 2])
  assert.deepEqual(model.queryReminders(lists, { view: "flagged" }, now).map((item) => item.id), [1])
  assert.deepEqual(model.queryReminders(lists, { view: "completed" }, now).map((item) => item.id), [4])
  assert.deepEqual(model.queryReminders(lists, { view: "all", search: "september" }, now).map((item) => item.id), [3])
  assert.deepEqual(model.queryReminders(lists, { view: "all", tag: "shop" }, now).map((item) => item.id), [1])
  assert.deepEqual(model.queryReminders(lists, { view: "all", tag: "shop", search: "milk" }, now).map((item) => item.id), [1])
  assert.deepEqual(model.queryReminders(lists, { view: "all", tag: "missing" }, now), [])
  assert.equal(model.queryReminders(lists, { view: "all" }, now).find((item) => item.id === 1).listRevision, 4)
  assert.deepEqual(model.allTags(lists), ["shop"])
})

test("subtasks render as a guarded hierarchy and collapse with their descendants", () => {
  const items = [
    { id: 4, title: "Grandchild", parentId: 3, rank: 1 },
    { id: 3, title: "Child", parentId: 1, rank: 5 },
    { id: 1, title: "Parent", parentId: null, rank: 10 },
    { id: 2, title: "Sibling", parentId: null, rank: 20 }
  ]
  const ordered = model.hierarchyOrder(items)
  assert.deepEqual(ordered.map((item) => item.id), [1, 3, 4, 2])
  assert.deepEqual(ordered.map((item) => item.depth), [0, 1, 2, 0])
  assert.equal(model.reminderHasChildren(ordered, 1), true)
  assert.equal(model.reminderHasChildren(ordered, 4), false)
  assert.deepEqual(model.visibleReminders(items, [1]).map((item) => item.id), [1, 2])
  assert.deepEqual(model.visibleReminders(items, [3]).map((item) => item.id), [1, 3, 2])

  const malformedCycle = [
    { id: 8, title: "A", parentId: 9, rank: 1 },
    { id: 9, title: "B", parentId: 8, rank: 2 }
  ]
  assert.deepEqual(model.visibleReminders(malformedCycle, []).map((item) => item.id), [8, 9])
})

test("cross-list hierarchy keeps duplicate reminder ids isolated", () => {
  const items = [
    { listId: 1, id: 2, title: "Home child", parentId: 1, rank: 1 },
    { listId: 2, id: 1, title: "Work parent", parentId: null, rank: 2 },
    { listId: 1, id: 1, title: "Home parent", parentId: null, rank: 3 },
    { listId: 2, id: 2, title: "Work child", parentId: 1, rank: 4 }
  ]
  const ordered = model.hierarchyOrder(items)
  assert.deepEqual(ordered.map((item) => `${item.listId}:${item.id}`), ["2:1", "2:2", "1:1", "1:2"])
  assert.deepEqual(ordered.map((item) => item.depth), [0, 1, 0, 1])
})

test("list rows expose empty sections and preserve reminders beneath their section headers", () => {
  const list = {
    sections: [
      { id: 8, title: "Later", rank: 20 },
      { id: 7, title: "Now", rank: 10 }
    ]
  }
  const reminders = [
    { id: 1, title: "Loose", sectionId: null },
    { id: 2, title: "In now", sectionId: 7 }
  ]
  const rows = model.sectionedRows(reminders, list, true)
  assert.deepEqual(rows.map((row) => row.kind === "section" ? `section:${row.section.id}:${row.count}` : `reminder:${row.reminder.id}`), [
    "reminder:1",
    "section:7:1",
    "reminder:2",
    "section:8:0"
  ])
})

test("menubar rows keep visible children with parents and summarize filtered subitems", () => {
  const lists = [{
    id: 4,
    reminders: [
      { id: 1, title: "Parent", parentId: null, rank: 1 },
      { id: 2, title: "Visible child", parentId: 1, rank: 2 },
      { id: 3, title: "Filtered child", parentId: 1, rank: 3 },
      { id: 4, title: "Orphaned by filter", parentId: 3, rank: 4 },
      { id: 5, title: "Visible grandchild", parentId: 2, rank: 5 },
      { id: 6, title: "Filtered grandchild", parentId: 3, rank: 6 },
      { id: 7, title: "Filtered under visible child", parentId: 2, rank: 7 }
    ]
  }]
  const visible = [
    { id: 1, listId: 4, title: "Parent", parentId: null, rank: 1 },
    { id: 2, listId: 4, title: "Visible child", parentId: 1, rank: 2 },
    { id: 4, listId: 4, title: "Orphaned by filter", parentId: 3, rank: 4 },
    { id: 5, listId: 4, title: "Visible grandchild", parentId: 2, rank: 5 }
  ]
  const rows = model.menubarRows(lists, visible)
  assert.deepEqual(rows.map((row) => row.kind === "summary" ? `summary:${row.parentReminderId}:${row.count}` : `reminder:${row.reminder.id}:${row.depth}:${row.orphanParentId || 0}`), [
    "reminder:1:0:0",
    "reminder:2:1:0",
    "reminder:5:2:0",
    "summary:2:1",
    "summary:1:2",
    "reminder:4:0:3"
  ])
  assert.deepEqual(rows.filter((row) => row.kind === "summary").map((row) => row.depth), [2, 1])
})

test("assigned view, assignee search, and next reminder use normalized ships and due order", () => {
  const lists = [{
    id: 1,
    title: "Team",
    revision: 2,
    reminders: [
      { id: 1, title: "Later", assignee: "~zod", completed: false, schedule: { "due-at": "~2026.9.12..12.00.00" } },
      { id: 2, title: "Soon", assignee: "zod", completed: false, schedule: { "due-at": "~2026.9.11..12.00.00" } },
      { id: 3, title: "Other", assignee: "~nec", completed: false }
    ]
  }]
  assert.deepEqual(model.queryReminders(lists, { view: "assigned", ship: "~zod" }).map((item) => item.id), [1, 2])
  assert.deepEqual(model.queryReminders(lists, { view: "all", search: "~nec" }).map((item) => item.id), [3])
  assert.equal(model.nextReminder(lists).id, 2)
})

test("reminder policy controls all-day overdue visibility and the bar badge", () => {
  const now = new Date(2026, 8, 10, 12, 0, 0).getTime()
  const lists = [{
    id: 1,
    title: "Team",
    revision: 1,
    reminders: [
      { id: 1, title: "Old all-day", completed: false, assignee: "~zod", schedule: { "due-at": "~2026.9.9..09.00.00", "all-day": true } },
      { id: 2, title: "Old timed", completed: false, assignee: "~nec", schedule: { "due-at": "~2026.9.9..12.00.00", "all-day": false } },
      { id: 3, title: "Today", completed: false, assignee: "zod", schedule: { "due-at": "~2026.9.10..18.00.00", "all-day": true } },
      { id: 4, title: "Unscheduled", completed: false }
    ]
  }]

  assert.deepEqual(model.queryReminders(lists, { view: "today", allDayOverdue: false }, now).map((item) => item.id), [2, 3])
  assert.equal(model.badgeCount(lists, { "badge-mode": "all" }, "~zod", now), 4)
  assert.equal(model.badgeCount(lists, { "badge-mode": "today", "all-day-overdue": false }, "~zod", now), 2)
  assert.equal(model.badgeCount(lists, { "badge-mode": "assigned" }, "~zod", now), 2)
  assert.equal(model.badgeCount(lists, { "badge-mode": "none" }, "~zod", now), 0)
})

test("transport-enriched wall-clock schedule fields survive model normalization", () => {
  const schedule = model.cloneSchedule({
    "due-at": "~2026.9.11..00.30.00",
    "local-due": "2026-09-10T17:30",
    timezone: "America/Los_Angeles",
    recurrence: {
      frequency: "weekly",
      interval: 1,
      weekdays: [4],
      "month-days": [],
      "month-week": null,
      "end-at": "~2026.10.2..00.30.00",
      "local-end": "2026-10-01T17:30",
      "max-occurrences": null
    }
  })
  assert.equal(schedule.localDue, "2026-09-10T17:30")
  assert.equal(schedule.recurrence.localEnd, "2026-10-01T17:30")
  assert.equal(model.scheduleInputValue(schedule), "2026-09-10T17:30")
  schedule.allDay = true
  assert.equal(model.scheduleInputValue(schedule), "2026-09-10")
})

test("Today evaluates enriched all-day reminders by calendar date", () => {
  const now = new Date(2026, 8, 10, 12, 0, 0).getTime()
  const lists = [{
    id: 1,
    title: "Dates",
    revision: 1,
    reminders: [
      { id: 1, title: "Yesterday", completed: false, schedule: { "due-at": "~2026.9.10..23.00.00", "all-day": true, "local-due": "2026-09-09T09:00" } },
      { id: 2, title: "Today", completed: false, schedule: { "due-at": "~2026.9.11..23.00.00", "all-day": true, "local-due": "2026-09-10T09:00" } },
      { id: 3, title: "Tomorrow but early UTC", completed: false, schedule: { "due-at": "~2026.9.10..16.00.00", "all-day": true, "local-due": "2026-09-11T09:00" } }
    ]
  }]
  assert.deepEqual(model.queryReminders(lists, { view: "today", allDayOverdue: true }, now).map(item => item.title), ["Yesterday", "Today"])
  assert.deepEqual(model.queryReminders(lists, { view: "today", allDayOverdue: false }, now).map(item => item.title), ["Today"])
})

test("manual drop placement accepts only changed sibling order", () => {
  const reminders = [
    { id: 1, parentId: null, sectionId: 4, rank: 1 },
    { id: 2, parentId: null, sectionId: 4, rank: 2 },
    { id: 3, parentId: null, sectionId: 4, rank: 3 },
    { id: 4, parentId: 1, sectionId: 4, rank: 1 },
    { id: 5, parentId: null, sectionId: 8, rank: 1 }
  ]
  assert.deepEqual(model.manualDropPlacement(reminders, 3, 1, false), { targetId: 1, after: false })
  assert.deepEqual(model.manualDropPlacement(reminders, 1, 3, true), { targetId: 3, after: true })
  assert.equal(model.manualDropPlacement(reminders, 1, 2, false), null)
  assert.equal(model.manualDropPlacement(reminders, 1, 4, true), null)
  assert.equal(model.manualDropPlacement(reminders, 1, 5, true), null)
  assert.equal(model.manualDropPlacement(reminders, 1, 1, true), null)
})

test("external reminder links match the Gall allowlist", () => {
  assert.equal(model.safeExternalUrl("https://example.com/task"), true)
  assert.equal(model.safeExternalUrl(" http://127.0.0.1/path "), true)
  assert.equal(model.safeExternalUrl("mailto:person@example.com"), true)
  assert.equal(model.safeExternalUrl("javascript:alert(1)"), false)
  assert.equal(model.safeExternalUrl("file:///tmp/private"), false)
  assert.equal(model.safeExternalUrl("HTTPS://example.com"), false)
  assert.equal(model.safeExternalUrl(""), false)
})

test("multiplayer access state gates remote editing while keeping owned lists writable", () => {
  const owned = {
    alias: 1,
    host: "~zod",
    "host-list-id": 1,
    status: "online",
    owner: true,
    members: [{ ship: "~nec", policy: { "can-invite": false, "notify-added": true, "notify-completed": true } }],
    pending: ["~bud"]
  }
  const offline = {
    alias: 2,
    host: "~nec",
    "host-list-id": 7,
    status: "offline",
    owner: false,
    members: [],
    pending: []
  }
  const checking = { ...offline, alias: 3, status: "checking" }
  const online = { ...offline, alias: 4, status: "online" }
  const result = model.reduce([], { accesses: [owned, offline, checking, online] })

  assert.equal(result.accesses[0].members[0].policy.canInvite, false)
  assert.deepEqual(result.accesses[0].pending, ["~bud"])
  assert.equal(model.canEditList(result.accesses, 1), true)
  assert.equal(model.canEditList(result.accesses, 2), false)
  assert.equal(model.canEditList(result.accesses, 3), false)
  assert.equal(model.canEditList(result.accesses, 4), true)
  assert.equal(model.canEditList(result.accesses, 99), false)
})

test("invitations and remote operation lifecycle reduce independently", () => {
  const invited = model.reduce([], {
    "invitations-updated": [{
      token: "invite-1",
      host: "~sampel-palnet",
      "host-list-id": 12,
      title: "House",
      "can-invite": true,
      "received-at": "~2026.9.10..22.00.00"
    }]
  })
  assert.equal(invited.invitations[0].title, "House")
  assert.equal(invited.invitations[0].canInvite, true)

  const pending = model.reduce([], {
    "operation-pending": { "operation-id": "edit-1", "list-id": 8 }
  }, null, null, null, invited.invitations)
  assert.deepEqual(pending.pendingOperations, [{ operationId: "edit-1", listId: 8 }])

  const settled = model.reduce([], {
    "operation-settled": { "operation-id": "edit-1" }
  }, null, null, null, invited.invitations, pending.pendingOperations)
  assert.deepEqual(settled.pendingOperations, [])
})

test("actor-attributed activity replaces one list log and preserves occurrence history", () => {
  const current = [{ listId: 1, events: [{ id: "old", actor: "~zod", kind: "reminder-added", "list-id": 1, "reminder-id": 7, at: "~2026.9.10" }] }]
  const updated = model.reduceActivities(current, {
    "activities-updated": {
      "list-id": 2,
      activities: [{
        id: "complete-1",
        actor: "~bus",
        kind: "completed",
        "list-id": 2,
        "reminder-id": 9,
        "occurred-at": "~2026.9.11..16.00.00",
        at: "~2026.9.10..20.00.00"
      }]
    }
  })
  assert.deepEqual(updated.map(function(entry) { return entry.listId }), [1, 2])
  const events = model.activitiesForList(updated, 2)
  assert.equal(events[0].actor, "~bus")
  assert.equal(events[0].reminderId, 9)
  assert.equal(events[0].occurredAt, "~2026.9.11..16.00.00")

  const replaced = model.reduceActivities(updated, {
    "activities-updated": { "list-id": 2, activities: [] }
  })
  assert.deepEqual(model.activitiesForList(replaced, 2), [])
  assert.equal(model.activitiesForList(replaced, 1)[0].id, "old")
})
