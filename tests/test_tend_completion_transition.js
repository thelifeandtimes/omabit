const assert = require("node:assert/strict")
const fs = require("node:fs")
const path = require("node:path")
const test = require("node:test")

const service = fs.readFileSync(path.join(__dirname, "..", "omarchy-plugin", "Service.qml"), "utf8")
const start = service.indexOf("function pendingCompletionIndex")
const end = service.indexOf("function scheduleCompleted", start)
if (start < 0 || end < 0 || end <= start) throw new Error("Could not locate completion transition helpers")
const helpers = service.slice(start, end)

function collapseAt(view, change, clock) {
  return new Function(
    "pendingCompletionChanges",
    "completionClock",
    `${helpers}\nreturn completionShouldCollapse(${JSON.stringify(view)}, ${change.listId}, ${change.reminderId});`,
  )([change], clock)
}

test("completed reminders collapse only after the fade finishes", () => {
  const change = { listId: 3, reminderId: 7, completed: true, requestedAt: 1000 }
  assert.equal(collapseAt("all", change, 10999), false)
  assert.equal(collapseAt("all", change, 11000), true)
})

test("list views retain completed reminders after the transition", () => {
  const change = { listId: 3, reminderId: 7, completed: true, requestedAt: 1000 }
  assert.equal(collapseAt("list", change, 11000), false)
  assert.equal(collapseAt("completed", change, 11000), false)
})

test("reopened reminders collapse out of the completed view", () => {
  const change = { listId: 3, reminderId: 7, completed: false, requestedAt: 1000 }
  assert.equal(collapseAt("completed", change, 11000), true)
  assert.equal(collapseAt("all", change, 11000), false)
})
