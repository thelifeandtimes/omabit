const test = require("node:test")
const assert = require("node:assert/strict")
const { performance } = require("node:perf_hooks")
const model = require("../omarchy-plugin/TendModel.js")

const REMINDER_COUNT = 10_000
const OPERATION_BUDGET_MS = 5_000
const TRANSPORT_LIMIT_BYTES = 32 * 1024 * 1024

function scaleSnapshot() {
  const reminders = []
  for (let id = 1; id <= REMINDER_COUNT; id += 1) {
    reminders.push({
      id,
      title: id === REMINDER_COUNT ? "Needle reminder" : `Reminder ${id}`,
      notes: `Deterministic scale fixture ${id}`,
      priority: id % 10 === 0 ? "high" : "none",
      flagged: id % 17 === 0,
      tags: [`tag-${id % 20}`],
      assignee: id % 3 === 0 ? "~zod" : null,
      rank: id * 1024,
      completed: id % 11 === 0,
      revision: 1,
      schedule: id % 2 === 0 ? {
        "due-at": id % 4 === 0 ? "~2026.9.10..18.00.00" : "~2026.9.12..18.00.00",
        "all-day": id % 8 === 0,
        timezone: "UTC",
        "early-seconds": [],
        occurrence: 0,
        recurrence: null
      } : null
    })
  }
  return {
    snapshot: {
      lists: [{ id: 1, title: "Scale", revision: 1, sections: [], reminders }],
      preferences: { revision: 0, "badge-mode": "today", "all-day-overdue": true },
      snoozes: []
    }
  }
}

test("10,000-reminder snapshot and queries stay within release budgets", (context) => {
  const update = scaleSnapshot()
  const payloadBytes = Buffer.byteLength(JSON.stringify(update), "utf8")
  assert.ok(payloadBytes < TRANSPORT_LIMIT_BYTES, `fixture is ${payloadBytes} bytes`)

  const normalizeStarted = performance.now()
  const state = model.reduce([], update)
  const normalizeMs = performance.now() - normalizeStarted
  assert.equal(state.lists[0].reminders.length, REMINDER_COUNT)
  assert.ok(normalizeMs < OPERATION_BUDGET_MS, `normalization took ${normalizeMs.toFixed(1)} ms`)

  const todayStarted = performance.now()
  const today = model.queryReminders(state.lists, { view: "today", sort: "due" }, Date.UTC(2026, 8, 10, 12))
  const todayMs = performance.now() - todayStarted
  assert.ok(today.length > 0)
  assert.ok(todayMs < OPERATION_BUDGET_MS, `Today query took ${todayMs.toFixed(1)} ms`)

  const searchStarted = performance.now()
  const search = model.queryReminders(state.lists, { view: "all", search: "needle reminder" })
  const searchMs = performance.now() - searchStarted
  assert.deepEqual(search.map((item) => item.id), [REMINDER_COUNT])
  assert.ok(searchMs < OPERATION_BUDGET_MS, `search took ${searchMs.toFixed(1)} ms`)

  context.diagnostic(`snapshot=${payloadBytes} bytes normalize=${normalizeMs.toFixed(1)}ms today=${todayMs.toFixed(1)}ms search=${searchMs.toFixed(1)}ms`)
})
