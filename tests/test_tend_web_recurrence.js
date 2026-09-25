const test = require("node:test")
const assert = require("node:assert/strict")
const fs = require("node:fs")
const path = require("node:path")

const html = fs.readFileSync(path.join(__dirname, "..", "desk", "app", "tend.html"), "utf8")
const start = html.indexOf("      function instantFromValue")
const end = html.indexOf("      function reminderAncestors")

if (start < 0 || end < 0 || end <= start) throw new Error("Could not locate the web recurrence helpers")

const helpers = new Function(`${html.slice(start, end)}\nreturn { recurrenceAdvance };`)()

test("web daily recurrence preserves wall time across daylight saving", () => {
  const result = helpers.recurrenceAdvance({
    "due-at": "~2026.3.7..17.00.00",
    timezone: "America/Los_Angeles",
    occurrence: 0,
    recurrence: {frequency: "daily", interval: 1},
  }, new Date("2026-03-07T18:00:00Z"))

  assert.deepEqual(result, {"due-at": "~2026.3.8..16.00.00", occurrence: 1})
})

test("web hourly recurrence uses exact elapsed time and catches up", () => {
  const result = helpers.recurrenceAdvance({
    "due-at": "~2026.3.8..09.30.00",
    timezone: "America/Los_Angeles",
    occurrence: 0,
    recurrence: {frequency: "hourly", interval: 1},
  }, new Date("2026-03-08T11:45:00Z"))

  assert.deepEqual(result, {"due-at": "~2026.3.8..12.30.00", occurrence: 3})
})

test("web recurrence supports selected weekdays and ordinal month rules", () => {
  const weekly = helpers.recurrenceAdvance({
    "due-at": "~2026.9.24..16.00.00",
    timezone: "America/Los_Angeles",
    occurrence: 0,
    recurrence: {frequency: "weekly", interval: 2, weekdays: [1, 4]},
  }, new Date("2026-09-24T17:00:00Z"))
  assert.deepEqual(weekly, {"due-at": "~2026.10.5..16.00.00", occurrence: 1})

  const ordinal = helpers.recurrenceAdvance({
    "due-at": "~2026.1.30..09.00.00",
    timezone: "UTC",
    occurrence: 0,
    recurrence: {frequency: "monthly", interval: 1, "month-week": {index: 5, weekday: 1}},
  }, new Date("2026-01-30T10:00:00Z"))
  assert.deepEqual(ordinal, {"due-at": "~2026.2.23..09.00.00", occurrence: 1})
})

test("web recurrence stops at count and date boundaries", () => {
  const count = helpers.recurrenceAdvance({
    "due-at": "~2028.2.29..17.00.00",
    timezone: "America/Los_Angeles",
    occurrence: 1,
    recurrence: {frequency: "yearly", interval: 1, "max-occurrences": 2},
  }, new Date("2028-02-29T18:00:00Z"))
  assert.deepEqual(count, {"due-at": null, occurrence: 2})

  const ending = helpers.recurrenceAdvance({
    "due-at": "~2026.9.25..16.00.00",
    timezone: "America/Los_Angeles",
    occurrence: 1,
    recurrence: {frequency: "daily", interval: 1, "end-at": "~2026.9.25..16.00.00"},
  }, new Date("2026-09-25T17:00:00Z"))
  assert.deepEqual(ending, {"due-at": null, occurrence: 2})
})
