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
    earlySeconds: 900
  })
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
