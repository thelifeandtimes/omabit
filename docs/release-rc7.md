# Omabit 0.1.0-rc.7 — persistent log following

RC.7 removes the runtime-log viewer's repeated SSH/PAM/logind login lifecycle.
Following output now keeps one collector and one private, identity-pinned connection
alive, polling the existing bounded endpoint every three seconds.

## Changes

- New `urbit-collector ... logs INSTANCE --revision N --watch --interval 3s` emits
  immediate and subsequent bounded NDJSON updates over a reused connection.
- Pause, hide, minimized Qt visibility, navigation and changed selection/enrollment
  cancel the reader; the next reader waits for it to exit. Refresh works as a
  one-shot read while paused.
- Transient failures use jittered exponential backoff capped at 30 seconds.
  Identity mismatch, denied access and configuration conflicts stop polling for
  review. Transient inventory failures do not reset the watcher's backoff.
- Fragmented/multiple records are parsed incrementally. Oversized and mismatched
  records are refused, and late output cannot cross a changed selection.

Healthy log following should use one SSH session per uninterrupted view instead
of roughly 200 sessions per ten minutes. Inventory retains its separate persistent
connection. Operator access, identity pins, runtime-generation/revision checks,
redaction and line/byte limits continue to apply.

## Qualification and installation

Full Go tests/vet, stateful transport/runtime race tests, static builds, model tests,
mapped onboarding/panel/log-lifecycle tests, and installer/bootstrap fixtures passed.
The bridge regression serves repeated log reads on one byte stream and verifies
live revocation. Cancellation tests cover pending handshakes, requests and backoff,
including child reaping. See the [D-Bus follow-up](diagnostics/2026-10-16-dbus-log-watch.md)
for scope and deployment evidence.

Install the matching collector and plugin together. This release retains schema16,
owner-user hosting, the qualified Docker/Vere image and the frozen supplied pill
from RC.5/RC.6. A user-driven fresh installation begins with:

```bash
omarchy plugin install https://github.com/thelifeandtimes/omabit.git --enable
```

## Native Vere D-Bus recovery

The reported D-Bus queues belonged to standalone native Vere 4.6 processes.
The upstream [mDNS fix](https://github.com/urbit/vere/commit/752aee3fbeecab697d4f8c0898397d71397ae4e5)
keeps the registration connection serviced. RC.7 reduces Omabit's contribution to
host traffic; it does not apply that patch to native ships or drain their existing
queues. Those ships need a verified fixed executable and graceful restarts as a
separate rollout. The latest published Vere release was still 4.6 on October 16.
