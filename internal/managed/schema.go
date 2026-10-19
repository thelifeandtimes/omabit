package managed

// StateSchemaVersion is also queried from the built runner by release tooling.
// Schema21 fences per-container engine-version evidence and portable host-engine
// contracts from schema20 and older runners. The version-only migration accepts
// schemas11–20, preserving instance bodies, historical signatures and replay
// records; it does not infer launch evidence from the currently installed engine.
const StateSchemaVersion = 21
