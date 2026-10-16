# D-Bus incident follow-up: persistent runtime-log watching

## Evidence and scope

The sourcebox report supplied on 2026-10-16 measured 511.20 MiB of queued outgoing
D-Bus messages on 16 connections owned by 14 **host-native Vere 4.6** processes.
The shared UID byte budget was 99.978% consumed. Those peers were standalone tmux
ships, rather than managed container runtimes. The report also measured a sustained
three-second runner SSH-login cadence: 1,200 sessions/hour while log following was
active. The precise share of Vere's backlog attributable to runner traffic was not
measured, and byte exhaustion predated that sustained sequence.

Source inspection confirmed that `RuntimePane.qml` launched a fresh collector and
SSH/PAM/logind session for every log refresh. Inventory already reused a pinned
client. Runner observation and log-tail work use the private Docker Unix socket;
those paths do not invoke per-poll D-Bus/systemctl commands.

The upstream defect and fix were checked directly through GitHub on 2026-10-16:

- [Vere #1125](https://github.com/urbit/vere/issues/1125).
- [Commit 752aee3fbeecab697d4f8c0898397d71397ae4e5](https://github.com/urbit/vere/commit/752aee3fbeecab697d4f8c0898397d71397ae4e5),
  dated October 12, keeps polling the registration's DNSServiceRef on success and
  deallocates it on error. Previously the Avahi registration stayed alive but unread.
- The latest-release API still returned `vere-v4.6`, published July 3. A fixed
  executable must be verified to contain the patch; “latest release” is insufficient.

## Implemented in source

- `urbit-collector ... logs INSTANCE --revision N --watch --interval 3s` retains
  one identity-pinned client, emitting immediate and subsequent bounded NDJSON
  tails. Existing one-shot output remains available.
- Healthy follow polls reuse one SSH session. Network failures reconnect with
  fresh identity verification and bounded jittered exponential backoff. Transient
  server errors keep the usable connection. Identity/access/revision failures
  terminate with sanitized, distinct errors; the UI does not auto-respawn them.
- The pane cancels on pause, hide, minimized Qt visibility state, navigation,
  changed runner/instance/revision or changed enrollment/tool selection. It waits
  for the previous process to exit before launching another. One-shot Refresh
  remains available when paused.
- Per-record parsing handles fragments and multiple records per read. It rejects
  oversized or mismatched records, discards late output from canceled selections,
  clears tails on errors and keeps only a bounded latest tail. Transient inventory
  observations do not interrupt the collector's backoff.
- Host keys, forced commands, operator checks, runner identity pins, revision and
  runtime-generation checks, redaction and mutation replay policy remain enforced.

Expected healthy log-follow traffic changes from about **200 SSH logins per ten
minutes to one initial session per uninterrupted view**, alongside the inventory
watcher's separate persistent connection. Polling still performs a bounded log
read every three seconds; this change removes login/process lifecycle churn rather
than claiming to eliminate all observation work.

## Verification

Regression coverage includes:

- Multiple real HTTP/Unix-socket log polls on one accepted connection; retained,
  empty and replacement-container tails.
- The actual SSH RPC bridge serving five log polls on one accepted byte stream,
  then refusing a revoked grant on that same stream.
- Bounded delayed reconnect followed by rejection of a replacement identity before
  any log request reaches it; terminal access/configuration/selection errors.
- Cancellation during handshake, request, backoff and output failure. An actual
  child process with stalled SSH-equivalent pipes is killed and reaped when the
  pinned client closes.
- Collector CLI NDJSON framing, interval validation, unchanged one-shot shape and
  successful cancellation exit.
- Mapped QML process-audit tests: a single watcher across multiple former refresh
  ticks and many megabytes of fragmented/batched records; pause/manual refresh;
  selection, revision and enrollment changes; hide/navigation; transient inventory
  changes; denied, mismatched, oversized and failed collectors; no overlapping or
  surviving collector processes. Minimized visibility is injected because Hyprland
  does not implement native minimization; the production gate uses Qt's state.

Passed locally on 2026-10-16 (Go commands used `GOTOOLCHAIN=go1.25.0`,
`GOMAXPROCS=2`, and `-p 1`):

```text
go test ./...
go vet ./...
go test -race ./internal/managed ./internal/runtimequalify ./internal/client \
  ./internal/access ./internal/collection ./internal/piercopy ./backend/cmd/urbit-collector
CGO_ENABLED=0 go build -trimpath -o build/ ./cmd/... ./backend/cmd/...
node scripts/test-panel-model.js
python3 scripts/test-log-watch.py
python3 scripts/test-onboarding.py
python3 -B scripts/test-panel.py --windows
python3 -B scripts/test-panel-install.py
python3 -B scripts/test-native-plugin.py
```

## Deployment and Vere recovery

This change is prepared for RC.7. Its rollout needs the updated collector and QML
together, with native pins advanced after artifact publication. It needs no
runner database migration or new runner endpoint. Live sourcebox SSH/logind counts
and a ten-minute follow test remain deployment acceptance checks.

Reducing this churn does **not** drain already-stuck Vere queues. Host-native ships
need a separately planned rollout of a verified fixed Vere executable and graceful
restarts; replacing the on-disk executable alone cannot repair running processes.
After that rollout, verify D-Bus quota headroom and desktop recovery, then soak for
recurrence. Updating the managed image alone cannot fix those native processes.
This source work does not claim a native-Vere rollout or a D-Bus recovery result.
