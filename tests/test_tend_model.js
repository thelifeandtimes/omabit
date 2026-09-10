const test = require("node:test")
const assert = require("node:assert/strict")
const model = require("../omarchy-plugin/TendModel.js")

test("snapshot is normalized and incomplete reminders are counted", () => {
  const result = model.reduce([], {
    snapshot: {
      lists: [{
        id: 2,
        title: "Work",
        revision: 3,
        reminders: [
          { id: 4, title: "Done", completed: true, revision: 2 },
          { id: 3, title: "Open", completed: false, revision: 1 }
        ]
      }]
    }
  })

  assert.equal(result.error, "")
  assert.deepEqual(result.lists[0].reminders.map((item) => item.id), [4, 3])
  assert.equal(model.incompleteCount(result.lists), 1)
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
  assert.deepEqual(result.lists, initial)
  assert.match(result.error, /stale-list/)
})

