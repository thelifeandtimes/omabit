# Enforce the initializer pin before startup — 2026-10-25

M396 makes the resource adapter's shared input precondition require both the
full reviewed generation and the exact M392 initializer source hash. Every
creation preflight and the late recheck immediately before bootstrap/Start uses
this path. The observer now shares it, avoiding a separate policy or duplicate
program hash check. An otherwise valid obsolete bundle cannot initialize first
and only fail later during readiness observation.

All 82 focused race checks and vet pass, including the real generation06 refusal.
The actual generation's 12,352 files/834,036,904 bytes fully verify, and its old
program matches its original hash. Passing it through the real gateway controller
with inert journal/engine adapters leaves the journal planned and dispatches no
engine command, network/container creation, bootstrap metadata or Start. The
late-change test also refuses a newly mismatched initializer after slower startup
checks, preserving the start-dispatched receipt without initializing metadata.
All 287 private-gateway race checks pass after integration; diff checks are clean.

No sealed input bytes were changed or executed, and no actual engine was used.
The prior M390 constructor result remains historical evidence for its source
version, not support for starting generation06 with the current adapter. This
supersedes its weaker program admission and M395's pending pre-start gap only.
Source pinning does not qualify the changed runtime. Bounded first-start waiting,
managed/client/parent wiring, a fresh sealed generation and a no-NIC VM lifecycle
test remain. Protected ships and installed state remain untouched.
