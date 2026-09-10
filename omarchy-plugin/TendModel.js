function cloneRecurrence(recurrence) {
  if (!recurrence) return null
  const monthWeek = recurrence["month-week"] || recurrence.monthWeek || null
  return {
    frequency: String(recurrence.frequency || "daily"),
    interval: Number(recurrence.interval || 1),
    weekdays: (recurrence.weekdays || []).map(Number).sort(function(a, b) { return a - b }),
    monthDays: (recurrence["month-days"] || recurrence.monthDays || []).map(Number).sort(function(a, b) { return a - b }),
    monthWeek: monthWeek ? { index: Number(monthWeek.index), weekday: Number(monthWeek.weekday) } : null,
    endAt: recurrence["end-at"] === undefined ? (recurrence.endAt || null) : recurrence["end-at"],
    maxOccurrences: recurrence["max-occurrences"] === undefined ? (recurrence.maxOccurrences ?? null) : recurrence["max-occurrences"]
  }
}

function cloneSchedule(schedule) {
  if (!schedule) return null
  return {
    dueAt: String(schedule["due-at"] || schedule.dueAt || ""),
    allDay: schedule["all-day"] === undefined ? schedule.allDay === true : schedule["all-day"] === true,
    timezone: String(schedule.timezone || "UTC"),
    earlySeconds: (schedule["early-seconds"] || schedule.earlySeconds || []).map(Number).sort(function(a, b) { return a - b }),
    recurrence: cloneRecurrence(schedule.recurrence),
    occurrence: Number(schedule.occurrence || 0)
  }
}

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
        schedule: cloneSchedule(reminder.schedule),
        completed: reminder.completed === true,
        lastCompletedAt: String(reminder["last-completed-at"] || reminder.lastCompletedAt || ""),
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
  if (!update || typeof update !== "object") return { lists: lists, error: "Invalid Tend update", alert: null }

  if (update.snapshot) return { lists: sortedLists(update.snapshot.lists), error: "", alert: null }

  if (update.alert) {
    return {
      lists: lists,
      error: "",
      alert: {
        listId: Number(update.alert["list-id"]),
        reminderId: Number(update.alert["reminder-id"]),
        dueAt: String(update.alert["due-at"] || ""),
        earlySeconds: Number(update.alert["early-seconds"] || 0)
      }
    }
  }

  if (update["list-upserted"]) {
    var replacement = cloneList(update["list-upserted"].list)
    return {
      lists: sortedLists(lists.filter(function(item) { return item.id !== replacement.id }).concat([replacement])),
      error: "",
      alert: null
    }
  }

  if (update["list-deleted"]) {
    var deletedId = update["list-deleted"]["list-id"]
    return { lists: lists.filter(function(item) { return item.id !== deletedId }), error: "", alert: null }
  }

  // Milestone 0 events remain readable during a rolling desk/plugin upgrade.
  if (update["list-created"]) {
    var created = cloneList(update["list-created"].list)
    return { lists: sortedLists(lists.filter(function(item) { return item.id !== created.id }).concat([created])), error: "", alert: null }
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
    return { lists: next, error: "", alert: null }
  }

  if (update.rejected) {
    return { lists: lists, error: "Action rejected: " + String(update.rejected.reason || "unknown"), alert: null }
  }

  return { lists: lists, error: "Unknown Tend update", alert: null }
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
  module.exports = { cloneRecurrence: cloneRecurrence, cloneSchedule: cloneSchedule, cloneList: cloneList, sortedLists: sortedLists, reduce: reduce, incompleteCount: incompleteCount }
}
