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

function orderedLists(lists, pinnedIds) {
  const normalized = sortedLists(lists)
  const order = new Map((pinnedIds || []).map(function(id, index) { return [Number(id), index] }))
  return normalized.sort(function(a, b) {
    const aPinned = order.has(Number(a.id))
    const bPinned = order.has(Number(b.id))
    if (aPinned && bPinned) return order.get(Number(a.id)) - order.get(Number(b.id))
    if (aPinned) return -1
    if (bPinned) return 1
    return Number(a.id) - Number(b.id)
  })
}

function clonePreferences(value) {
  value = value || {}
  return {
    revision: Number(value.revision || 0),
    defaultList: value["default-list"] === undefined ? (value.defaultList ?? null) : value["default-list"],
    pinnedLists: (value["pinned-lists"] || value.pinnedLists || []).map(Number),
    pinnedViews: (value["pinned-views"] || value.pinnedViews || ["today", "scheduled", "all", "flagged", "completed"]).map(String),
    snoozePresets: (value["snooze-presets"] || value.snoozePresets || [300, 900, 3600]).map(Number)
  }
}

function cloneSnoozes(values) {
  return (values || []).map(function(value) {
    return {
      listId: Number(value["list-id"] === undefined ? value.listId : value["list-id"]),
      reminderId: Number(value["reminder-id"] === undefined ? value.reminderId : value["reminder-id"]),
      until: String(value.until || "")
    }
  }).sort(function(a, b) {
    return a.until.localeCompare(b.until) || a.listId - b.listId || a.reminderId - b.reminderId
  })
}

function reduce(currentLists, update, currentPreferences, currentSnoozes) {
  var lists = sortedLists(currentLists)
  var preferences = clonePreferences(currentPreferences)
  var snoozes = cloneSnoozes(currentSnoozes)
  if (!update || typeof update !== "object") return { lists: lists, preferences: preferences, snoozes: snoozes, error: "Invalid Tend update", alert: null }

  if (update.snapshot) return {
    lists: sortedLists(update.snapshot.lists),
    preferences: clonePreferences(update.snapshot.preferences),
    snoozes: cloneSnoozes(update.snapshot.snoozes),
    error: "",
    alert: null
  }

  if (update.alert) {
    var isSnoozedAlert = update.alert.snoozed === true
    var alertListId = Number(update.alert["list-id"])
    var alertReminderId = Number(update.alert["reminder-id"])
    return {
      lists: lists,
      preferences: preferences,
      snoozes: isSnoozedAlert ? snoozes.filter(function(item) {
        return item.listId !== alertListId || item.reminderId !== alertReminderId
      }) : snoozes,
      error: "",
      alert: {
        listId: alertListId,
        reminderId: alertReminderId,
        dueAt: String(update.alert["due-at"] || ""),
        earlySeconds: Number(update.alert["early-seconds"] || 0),
        snoozed: isSnoozedAlert
      }
    }
  }

  if (update["list-upserted"]) {
    var replacement = cloneList(update["list-upserted"].list)
    return {
      lists: sortedLists(lists.filter(function(item) { return item.id !== replacement.id }).concat([replacement])),
      preferences: clonePreferences(update["list-upserted"].preferences || preferences),
      snoozes: snoozes,
      error: "",
      alert: null
    }
  }

  if (update["list-deleted"]) {
    var deletedId = update["list-deleted"]["list-id"]
    return {
      lists: lists.filter(function(item) { return item.id !== deletedId }),
      preferences: clonePreferences(update["list-deleted"].preferences || preferences),
      snoozes: snoozes.filter(function(item) { return item.listId !== deletedId }),
      error: "",
      alert: null
    }
  }

  if (update["preferences-updated"]) {
    return { lists: lists, preferences: clonePreferences(update["preferences-updated"].preferences), snoozes: snoozes, error: "", alert: null }
  }

  if (update.snoozed) {
    var snoozed = cloneSnoozes([update.snoozed])[0]
    var nextSnoozes = snoozes.filter(function(item) {
      return item.listId !== snoozed.listId || item.reminderId !== snoozed.reminderId
    })
    nextSnoozes.push(snoozed)
    return { lists: lists, preferences: preferences, snoozes: cloneSnoozes(nextSnoozes), error: "", alert: null }
  }

  // Milestone 0 events remain readable during a rolling desk/plugin upgrade.
  if (update["list-created"]) {
    var created = cloneList(update["list-created"].list)
    return { lists: sortedLists(lists.filter(function(item) { return item.id !== created.id }).concat([created])), preferences: preferences, snoozes: snoozes, error: "", alert: null }
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
    return { lists: next, preferences: preferences, snoozes: snoozes, error: "", alert: null }
  }

  if (update.rejected) {
    return { lists: lists, preferences: preferences, snoozes: snoozes, error: "Action rejected: " + String(update.rejected.reason || "unknown"), alert: null }
  }

  return { lists: lists, preferences: preferences, snoozes: snoozes, error: "Unknown Tend update", alert: null }
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

function allTags(lists) {
  const found = new Set()
  ;(lists || []).forEach(function(list) {
    ;(list.reminders || []).forEach(function(reminder) {
      ;(reminder.tags || []).forEach(function(tag) { found.add(String(tag)) })
    })
  })
  return Array.from(found).sort(function(a, b) { return a.localeCompare(b) })
}

function urbitDateMs(value) {
  const match = /^~(\d+)\.(\d+)\.(\d+)\.\.(\d+)\.(\d+)\.(\d+)/.exec(String(value || ""))
  if (!match) return NaN
  return Date.UTC(Number(match[1]), Number(match[2]) - 1, Number(match[3]), Number(match[4]), Number(match[5]), Number(match[6]))
}

function queryReminders(lists, options, nowMs) {
  options = options || {}
  const view = String(options.view || "list")
  const search = String(options.search || "").trim().toLocaleLowerCase()
  const selectedListId = options.listId
  const normalized = sortedLists(lists)
  const now = new Date(nowMs === undefined ? Date.now() : nowMs)
  const tomorrow = new Date(now.getFullYear(), now.getMonth(), now.getDate() + 1).getTime()
  const items = []

  normalized.forEach(function(list) {
    if (view === "list" && list.id !== selectedListId) return
    list.reminders.forEach(function(reminder) {
      const dueMs = reminder.schedule ? urbitDateMs(reminder.schedule.dueAt) : NaN
      var matchesView = true
      if (view === "today") matchesView = !reminder.completed && Number.isFinite(dueMs) && dueMs < tomorrow
      else if (view === "scheduled") matchesView = !reminder.completed && reminder.schedule !== null
      else if (view === "all") matchesView = !reminder.completed
      else if (view === "flagged") matchesView = !reminder.completed && reminder.flagged
      else if (view === "completed") matchesView = reminder.completed
      if (!matchesView) return

      if (search) {
        const haystack = [reminder.title, reminder.notes, reminder.url || "", reminder.tags.join(" "), list.title].join("\n").toLocaleLowerCase()
        if (haystack.indexOf(search) === -1) return
      }

      reminder.listId = list.id
      reminder.listRevision = list.revision
      reminder.listTitle = list.title
      reminder.dueMs = dueMs
      items.push(reminder)
    })
  })

  const sort = String(options.sort || "manual")
  const direction = options.descending === true ? -1 : 1
  const priority = { high: 0, medium: 1, low: 2, none: 3 }
  items.sort(function(a, b) {
    var compared = 0
    if (sort === "due") {
      const aDue = Number.isFinite(a.dueMs) ? a.dueMs : Number.MAX_SAFE_INTEGER
      const bDue = Number.isFinite(b.dueMs) ? b.dueMs : Number.MAX_SAFE_INTEGER
      compared = aDue - bDue
    } else if (sort === "created") {
      const aCreated = urbitDateMs(a.createdAt)
      const bCreated = urbitDateMs(b.createdAt)
      compared = (Number.isFinite(aCreated) ? aCreated : Number.MAX_SAFE_INTEGER) - (Number.isFinite(bCreated) ? bCreated : Number.MAX_SAFE_INTEGER)
    } else if (sort === "priority") {
      compared = priority[a.priority] - priority[b.priority]
    } else if (sort === "title") {
      compared = a.title.localeCompare(b.title)
    } else {
      compared = a.rank - b.rank
    }
    return compared * direction || Number(a.listId) - Number(b.listId) || Number(a.id) - Number(b.id)
  })
  return items
}

if (typeof module !== "undefined") {
  module.exports = { cloneRecurrence: cloneRecurrence, cloneSchedule: cloneSchedule, cloneList: cloneList, sortedLists: sortedLists, orderedLists: orderedLists, clonePreferences: clonePreferences, cloneSnoozes: cloneSnoozes, reduce: reduce, incompleteCount: incompleteCount, allTags: allTags, urbitDateMs: urbitDateMs, queryReminders: queryReminders }
}
