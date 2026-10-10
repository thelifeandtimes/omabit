# Moon recovery, runtime logs and cold panel updates — 0.19.1

## Confirmed failure

First Start `op-6b342f7e86ae47a1f465eff49c4003ab` of
`~dibpen-fogweg-dozzod-sarlev` began at 15:40:51 UTC and verified the expected HTTP
identity at 15:45:32 UTC. Omabit then requested shutdown to move the staging pier
into the conventional tree. Shutdown did not finish within 300 seconds. Docker
killed the container at 15:50:32 UTC: exit **137**, **OOMKilled=false**.
The retained container was
`5bc31859b8f336d37a3acec7131b423ad621482fd33f339a90af0d398832c7a3`.

Output also contained HTTP 504 / Azimuth snapshot download retries and Hoon thread
traces. There was no Vere coredump or kernel OOM-kill entry. These facts distinguish
management-triggered forced shutdown from the runtime's network/application
messages; they do not establish why Vere was slow to stop.

## Changes

- Snapshots expose runtime stage, initialized/storage-pending state, container
  identity, exit/OOM evidence, observation time and the latest lifecycle operation
  with its actual failure message.
- Operator-only, revision/runner-bound `runtime-logs` reads authenticate the recorded
  container and read both stdout and stderr through Docker's framed API. Reads are
  bounded to 1 MiB transport and 200 lines / 64 KiB display output. Terminal controls
  and known key/code/credential patterns are removed. Logs stay out of fleet snapshots.
- A sanitized tail is retained before an owned container is removed.
- Ship overviews show status/error details, compact narration and selectable
  plain-text output with three-second polling, Pause and Refresh. Selection changes
  clear old output; mismatched replies are rejected.
- Automatic storage finalization sends SIGTERM and waits without force-killing.
  Explicit operator Stop retains its documented grace behavior.
- A stopped initialized staging pier moves before resume, avoiding another staging
  boot/stop cycle. Stale `.vere.lock` recovery requires private staging, verified
  identity and removed-container evidence. It validates file metadata and acquires
  Vere's byte-zero POSIX lock before preserving the PID file outside the pier.
  Unsafe files and locks held by another process are refused.

The UI approach was informed by [Hawk's boot narration](https://hawk.computer/-/try)
and [Groundseg's Docker output](https://github.com/Native-Planet/GroundSeg/blob/master/goseg/ws/logs.go).
Official Vere and `--no-conn` remain; this is read-only output, not an interactive dojo.

## Quickshell update failure

The first panel replacement triggered a separate SIGSEGV (dump PID 1251064 at
16:24:39 UTC). Its stack was in `__dynamic_cast`, Quickshell and QML object
finalization/incubation, unlike the earlier GTK/GVFS Browse abort. The journal
showed service teardown and a deferred registration callback in an invalid QML
context. Internal Quickshell frames remained unsymbolized.

`install-panel.py --replace --enable` now validates/stages first, refuses a locked
desktop, cleanly stops the Omarchy shell configuration, replaces the plugin and
restarts even after a replacement failure. It avoids hot-replacing live QML and
needs no second manual restart. Registration/log deferrals use component-owned
timers. Authorization prompts and shell updates are serialized. Loaded version
**0.19.1** passed subsequent checks without another observed coredump.

## Live recovery

The user completed a stopped-pier copy and SQLite online backup:

- `/var/lib/urbit-runner/piers/archive/dibpen-fogweg-dozzod-sarlev.before-recovery-19`
- `/var/lib/urbit-runner/.local/share/urbit-runner/managed/before-moon-recovery-19.sqlite`

Stop `op-6e2177d370be38b18970063145e3419d` removed the already-exited container and
retained 200 sanitized lines. Resume `op-d62d466714dfd3e3d1308eb43b47dfca` safely
refused the stale Vere lock. After that fix, request
`recovery19-resume-moon-stale-lock`, operation
`op-28bbb764caaba89046d9bc89f962dac4`, moved/resumed the initialized pier and succeeded
at **17:52:30 UTC**, using the initialized-resume path rather than bootstrap/key flags.

Verified at 17:56 UTC:

- Moon running/healthy at **1/1**, storage pending false, no recovery/active operation.
- `http://127.0.0.1:18080/~/host` returns `~dibpen-fogweg-dozzod-sarlev`.
- Container: `eaadda72dc7c576aa9649b47c85be4c02a7b10959d823ed5895b23d28429cf79`.
- Pier: `/var/lib/urbit-runner/piers/livenet/moons/dibpen-fogweg-dozzod-sarlev`.
- Alpha, bus-lab and comet-check healthy; import-check/panel-check stopped;
  rus/ssh-check uninitialized; no incidents.
- Logs readable over real pinned SSH. Later output showed Azimuth snapshot
  loading/processing and `l2-sig-failed`; local HTTP health does not establish full
  application or inbound Ames acceptance.

Final installed runner SHA256:
`823dbba5ff5b0290f4416a70f5bbb772a978dc21db07b29c75bb43860911b396`.

## Checks

Full Go tests/vet, targeted managed/runtime/client/collector race tests, native
builds and the unsupported ARM64 stub build passed. Runtime regressions cover
framing/stderr/redaction/limits, container ownership, retained logs, non-forced
promotion timeout, stale-lock preservation, live-lock refusal and move-before-resume.
API tests cover operator access, revisions, replacement races and snapshot isolation.

Model, installer and mapped QML tests pass, including log selection changes,
plain-text rendering, pause/follow, component teardown, cold-update ordering,
locked-session refusal and restart-after-failure. Browse and registration/Start
checks also pass. `git diff --check` passes.
