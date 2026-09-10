function cloneList(list) {
  return {
    id: list.id,
    title: String(list.title || ""),
    revision: Number(list.revision || 0),
    reminders: (list.reminders || []).map(function(reminder) {
      return {
        id: reminder.id,
        title: String(reminder.title || ""),
        completed: reminder.completed === true,
        revision: Number(reminder.revision || 0)
      }
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

