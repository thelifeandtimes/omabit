# Project member recovery

Project lifecycle receipts and current process observations are separate evidence.
A queued acknowledgement does not establish shutdown, and an unavailable HTTP
endpoint does not establish a clean exit. Retain the original request and exact
inputs after a transport failure.

## Stop observation and receipt lookup

The development source binds a running member's process start timestamp before
bracketed ownership inspection. A running-to-exited transition may trigger up to
three complete observations with that same binding. It never rebinds a restarted
process or retries a generic ownership, image, namespace or engine mismatch.
Nonzero/OOM exits are observable failures, not clean-stop authority.

Restricted SSH now forwards the replay-only header on guarded project
registration, member action and anchor-recovery endpoints, as well as ship
creation. The bridge rechecks the selected runner's replay capability on the
actual forwarded connection. Unsupported endpoints still refuse the header.
Expanded-member receipt lookup precedes runtime-backend and mutation-admission
checks. It returns only an existing actor/request/fingerprint match and cannot
schedule another signal or worker.

The regression fixture deterministically reproduces the old failure shape at
both preliminary/bracketed read boundaries and during a namespace probe. This
supports the race explanation in issue #16; it does not prove which predicate
failed in that historical incident. Isolated inert fixture and race tests pass
for these paths. No real-runtime, installed upgrade or production qualification
is implied by those results.

An interrupted `stop-dispatched` member still requires an explicit supported
reconciliation before ordinary Start. Never repair a runner database, repeat
SIGTERM with a new request, reset a pier, or reinterpret an interrupted receipt
as a successful shutdown.
