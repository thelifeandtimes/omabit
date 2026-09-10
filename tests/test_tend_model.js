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
      "list-id": 1,
      "reminder-id": 9,
      "due-at": "~2026.9.10..21.00.00",
      "early-seconds": 900
    }
  })

  assert.deepEqual(result.lists, model.sortedLists(initial))
  assert.deepEqual(result.alert, {
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
    alert: { "list-id": 2, "reminder-id": 8, "due-at": "~2026.9.10..20.00.00", "early-seconds": 0, snoozed: true }
  }, updated.preferences, updated.snoozes)
  assert.equal(fired.alert.snoozed, true)
  assert.equal(fired.snoozes.length, 0)
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
  assert.equal(model.queryReminders(lists, { view: "all" }, now).find((item) => item.id === 1).listRevision, 4)
  assert.deepEqual(model.allTags(lists), ["shop"])
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
