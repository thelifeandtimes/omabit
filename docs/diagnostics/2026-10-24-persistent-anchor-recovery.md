# Persistent coordinated recovery and private queue

Unreleased schema38 persists the aggregate recovery under the existing project
and management-update fences. Immutable source history, actual operation actors,
original/current readiness references and exact operation mirrors are checked
against SQLite. Parent and distinct member creator operations commit atomically.
Every Stop/remove/Create/Start follows a saved checkpoint; reopening or actor replay
retains uncertainty without dispatching effects. Completed recovery becomes an
explicit historical boundary for later ordinary Stop/retire/resume/readiness.

Synthetic tests cover interrupted boundaries, mirror/reference tampering, stale
transitions, transaction rollback, two completed recoveries, and an ordinary
replacement between them. Initial empty-list comparisons and causal ordinary-resume
ordering failures were reproduced and fixed; the failed logs remain retained.
A validation-pass-only read cache and shared validated retirement selection remove
repeated recursive work. Cache aliases and new-pass mutation detection are tested;
no engine observation is cached and no cache survives a write.

The private queue and concrete replacement planner are wired. Lost replies remain
fenced, including a returned Create ID. Eyre warmup now retries only complete,
fresh, read-only readiness brackets for at most180 seconds, never runtime actions.
Focused queue race122.986s, planner race2.927s, full pure race228.892s, mixed lifecycle
8.627s and historical migration tests5.557s pass. Original schema37 reproduces its
531102-byte fixture and refuses genuine populated schema38 unchanged. Static checks,
43 Python harness tests and host refusal of the new VM entrypoint pass.

The first full managed race invocation exited143 with no output; its cause is
unknown, and it is not a pass. A new invocation is running. Actual coordinated
Docker repair remains unqualified; the new VM fixture is prepared separately.
There is no public lifecycle route or interrupted-recovery continuation queue yet.
No installed database, production ship, plugin or service was touched.
