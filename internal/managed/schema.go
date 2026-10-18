package managed

// StateSchemaVersion is also queried from the built runner by release tooling.
// Schema20 fences durable management-update admission, object-bound destructive
// recovery and actor-scoped handoff receipts from older runners. The migration
// preserves existing instance bodies, historical signatures and replay records.
const StateSchemaVersion = 20
