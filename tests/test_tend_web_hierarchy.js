const assert = require("node:assert/strict")
const fs = require("node:fs")
const path = require("node:path")
const test = require("node:test")

const html = fs.readFileSync(path.join(__dirname, "..", "desk", "app", "tend.html"), "utf8")
const start = html.indexOf("function hierarchyOrderedReminders")
const end = html.indexOf("function canEdit", start)
if (start < 0 || end < 0 || end <= start) throw new Error("Could not locate the web hierarchy helper")
const { hierarchyOrderedReminders } = new Function(`${html.slice(start, end)}\nreturn { hierarchyOrderedReminders };`)()

const fixtures = () => [
  { id: 11, title: "Child", parentId: 4, rank: 11 },
  { id: 15, title: "Grandchild", parentId: 11, rank: 15 },
  { id: 4, title: "Parent", parentId: null, rank: 4 },
  { id: 20, title: "Sibling", parentId: null, rank: 20 },
]

test("web hierarchy keeps parents before descendants in ascending order", () => {
  const ordered = hierarchyOrderedReminders(fixtures(), (a, b) => a.rank - b.rank || a.id - b.id)
  assert.deepEqual(ordered.map(item => item.id), [4, 11, 15, 20])
})

test("web hierarchy reverses sibling order without reversing parent-child order", () => {
  const ordered = hierarchyOrderedReminders(fixtures(), (a, b) => b.rank - a.rank || a.id - b.id)
  assert.deepEqual(ordered.map(item => item.id), [20, 4, 11, 15])
})

test("web hierarchy handles missing parents and cycles without dropping reminders", () => {
  const reminders = [
    { id: 8, title: "Cycle A", parentId: 9, rank: 8 },
    { id: 9, title: "Cycle B", parentId: 8, rank: 9 },
    { id: 10, title: "Filtered orphan", parentId: 99, rank: 10 },
  ]
  const ordered = hierarchyOrderedReminders(reminders, (a, b) => a.rank - b.rank || a.id - b.id)
  assert.deepEqual(ordered.map(item => item.id), [10, 8, 9])
})

test("web hierarchy isolates duplicate reminder ids from different lists", () => {
  const reminders = [
    { listId: 1, id: 2, title: "Home child", parentId: 1, rank: 1 },
    { listId: 2, id: 1, title: "Work parent", parentId: null, rank: 2 },
    { listId: 1, id: 1, title: "Home parent", parentId: null, rank: 3 },
    { listId: 2, id: 2, title: "Work child", parentId: 1, rank: 4 },
  ]
  const ordered = hierarchyOrderedReminders(reminders, (a, b) => a.rank - b.rank || a.id - b.id)
  assert.deepEqual(ordered.map(item => `${item.listId}:${item.id}`), ["2:1", "2:2", "1:1", "1:2"])
})

test("web All view uses parent-first hierarchy ordering", () => {
  assert.match(html, /state\.view === "list" \|\| state\.view === "all"/)
})
