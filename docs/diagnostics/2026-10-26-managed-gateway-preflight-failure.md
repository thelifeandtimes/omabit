# Managed gateway pre-start failure

M411, 2026-10-26. Fresh ordinary-source VM06 contains the M410 RPC encoding
correction but fails before network/container dispatch. Its original gateway
remains planned with three operations, zero instances and zero observations.
The fixture reports that the expected started resources were absent. The
initializer never ran, so this result neither verifies nor disproves M410.

The VM shuts down cleanly after 495.317 seconds. Read-only audit04 preserves the
disk and base hashes, and all three owned units are terminal. The disk is retained
privately at its verified compatibility path. The auditor's input checks pass at
39.281 and 11.084 seconds under its separate one-CPU/2GiB limit. No private key,
chain payload or raw log is exported. No existing ship or installation is accessed.

The manager intentionally records a generic interruption receipt; it does not
retain the precise driver-opening failure. Further disk reads cannot establish
that lost error. A logging-only overlay adds 26 fixed preflight stages to the
prior observer diagnostics. All355 gateway race checks and vet pass, with the
same acceptance rules/deadlines. Production sources remain unchanged by the
overlay. The next fresh VM will use it; do not retry the uncertain original.

This supersedes only M410's pending ordinary-VM status. M410's hash correction
and source evidence remain valid; managed readiness/recovery remain unqualified.
