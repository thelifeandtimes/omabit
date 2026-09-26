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

function opacityAt(change, clock, role) {
  return new Function(
    "pendingCompletionChanges",
    "completionClock",
    "recurrenceTransitionDuration",
    `${helpers}\nreturn completionOpacity(${change.listId}, ${change.reminderId}, ${JSON.stringify(role || "")});`,
  )([change], clock, 1400)
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

test("recurring successor opacity is the exact inverse of its outgoing occurrence", () => {
  const change = { listId: 3, reminderId: 7, completed: true, requestedAt: 1000, recurring: true, transitionStartedAt: 2000 }
  for (const elapsed of [0, 350, 700, 1050, 1400]) {
    const outgoing = opacityAt(change, 2000 + elapsed, "outgoing")
    const incoming = opacityAt(change, 2000 + elapsed, "")
    assert.ok(Math.abs(outgoing + incoming - 1) < 1e-9)
  }
})
