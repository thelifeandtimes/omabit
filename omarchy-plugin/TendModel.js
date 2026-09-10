function cloneList(list) {
  return {
    id: list.id,
    title: String(list.title || ""),
    color: String(list.color || "#3b82f6"),
    symbol: String(list.symbol || "list"),
    revision: Number(list.revision || 0),
    createdAt: String(list["created-at"] || list.createdAt || ""),
    modifiedAt: String(list["modified-at"] || list.modifiedAt || ""),
    sections: (list.sections || []).map(function(section) {
      return {
        id: section.id,
        title: String(section.title || ""),
        rank: Number(section.rank || 0)
      }
    }).sort(function(a, b) {
      return a.rank - b.rank || Number(a.id) - Number(b.id)
    }),
    reminders: (list.reminders || []).map(function(reminder) {
      return {
        id: reminder.id,
        title: String(reminder.title || ""),
        notes: String(reminder.notes || ""),
        url: reminder.url === null || reminder.url === undefined ? null : String(reminder.url),
        priority: String(reminder.priority || "none"),
        flagged: reminder.flagged === true,
        tags: (reminder.tags || []).map(String).sort(),
        parentId: reminder["parent-id"] === undefined ? (reminder.parentId ?? null) : reminder["parent-id"],
        sectionId: reminder["section-id"] === undefined ? (reminder.sectionId ?? null) : reminder["section-id"],
        rank: Number(reminder.rank || 0),
        completed: reminder.completed === true,
        revision: Number(reminder.revision || 0),
        createdAt: String(reminder["created-at"] || reminder.createdAt || ""),
        modifiedAt: String(reminder["modified-at"] || reminder.modifiedAt || "")
      }
    }).sort(function(a, b) {
      return a.rank - b.rank || Number(a.id) - Number(b.id)
    })
  }
}

function sortedLists(lists) {
  return (lists || []).map(cloneList).sort(function(a, b) { return Number(a.id) - Number(b.id) })
}

function reduce(currentLists, update) {
  var lists = sortedLists(currentLists)
  if (!update || typeof update !== "object") return { lists: lists, error: "Invalid Tend update" }

  if (update.snapshot) return { lists: sortedLists(update.snapshot.lists), error: "" }

  if (update["list-upserted"]) {
    var replacement = cloneList(update["list-upserted"].list)
    return {
      lists: sortedLists(lists.filter(function(item) { return item.id !== replacement.id }).concat([replacement])),
      error: ""
    }
  }

  if (update["list-deleted"]) {
    var deletedId = update["list-deleted"]["list-id"]
    return { lists: lists.filter(function(item) { return item.id !== deletedId }), error: "" }
  }

  // Milestone 0 events remain readable during a rolling desk/plugin upgrade.
  if (update["list-created"]) {
    var created = cloneList(update["list-created"].list)
    return { lists: sortedLists(lists.filter(function(item) { return item.id !== created.id }).concat([created])), error: "" }
  }

  var payload = update["reminder-added"] || update["reminder-completed"]
  if (payload) {
    var next = lists.map(function(item) {
      if (item.id !== payload["list-id"]) return item
      var copy = cloneList(item)
      var reminder = cloneList({ id: 0, title: "", revision: 0, reminders: [payload.reminder] }).reminders[0]
      copy.reminders = copy.reminders.filter(function(existing) { return existing.id !== reminder.id })
      copy.reminders.push(reminder)
      copy.reminders.sort(function(a, b) { return Number(a.id) - Number(b.id) })
      copy.revision = Number(payload["list-revision"] || copy.revision)
      return copy
    })
    return { lists: next, error: "" }
  }

  if (update.rejected) {
    return { lists: lists, error: "Action rejected: " + String(update.rejected.reason || "unknown") }
  }

  return { lists: lists, error: "Unknown Tend update" }
}

function incompleteCount(lists) {
  var total = 0
  ;(lists || []).forEach(function(list) {
    ;(list.reminders || []).forEach(function(reminder) {
      if (!reminder.completed) total += 1
    })
  })
  return total
}

if (typeof module !== "undefined") {
  module.exports = { cloneList: cloneList, sortedLists: sortedLists, reduce: reduce, incompleteCount: incompleteCount }
}
