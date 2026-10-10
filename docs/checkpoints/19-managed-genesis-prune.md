# Checkpoint 6I: confirmed managed genesis pruning

**Live validation passed September 26 UTC.** Alpha's confirmed `0i0` deletion
removed all four files and preserved the three retained epoch records exactly.
Stopped/running replay matched, resume completed in 5.97 s, and alpha was healthy
at revision 2/2. See the [evidence](../diagnostics/2026-09-26-managed-genesis-prune.md).
The commands below remain a reproduction guide; this gate is complete.
Standalone genesis deletion and latest-epoch recovery passed 6G. Managed roll
passed 6H and left alpha with four epochs and healthy revision 2/2.

## Supported scope

This checkpoint exposes exactly the deletion scope already qualified: remove
genesis epoch **0i0** from a stopped, initialized, four-epoch development pier,
retaining all three newer snapshot-bearing epochs. It reuses the 6G rooted,
non-recursive unlink engine with a durable manifest, per-file sync/progress,
fresh revalidation, and retained metadata checks.

The request requires an explicit cutoff, selected inventory digest, configuration
revision, request ID, and history-loss confirmation. A zero/missing confirmation,
stale selection, shared files, unsupported layout, or broader cutoff is refused.
The current state remains represented by retained snapshots/logs; the selected
old epoch's recovery history is irreversibly removed.

Deletion success leaves the pier stopped and refreshes its inventory. A partial
or uncertain deletion blocks resume and further maintenance even after stop.
Exact request replay returns the original operation and cannot repeat deletion.

## 1. Upgrade management — user intervention

The agent has built the binaries. After active work finishes, press **Ctrl+C in
the current `serve-fake` terminal** and wait for it to exit. Existing ship containers
continue independently.

Copy the executable:

```bash
sudo install -D -o root -g root -m 0755 \
  /home/thelifeandtimes/repos/omabit/build/urbit-runner \
  /usr/local/lib/urbit-runner/checkpoints/6i/urbit-runner
```

Start management and leave it running:

```bash
sudo runuser -u urbit-runner -- /usr/bin/env -i \
  HOME=/var/lib/urbit-runner PATH=/usr/bin:/bin \
  /usr/local/lib/urbit-runner/checkpoints/6i/urbit-runner \
  serve-fake --acknowledge-development
```

In another desktop terminal:

```bash
/home/thelifeandtimes/repos/omabit/build/urbit-collector \
  --format text --runner workstation hello
```

Expected: the same runner identity and SSH operator, with **`epoch-prune-genesis`**.
Send that output; the agent can perform the bounded development-pier sequence below.
The existing SSH bridge remains compatible. If its listener needs restarting, use
another terminal with `sudo /usr/bin/sshd -D -e -f /etc/urbit-runner/sshd_config`.

## 2. Live target and validation

The planned target is development alpha, last healthy at revision **2/2**, with:

- Selected: `0i0`.
- Retained: `0i22185`, `0i22199`, `0i22213`.

Check fresh identity, state, available RAM for the later resume, and absence of
active work before stopping. Inspection after stop must produce a new inventory
digest; the old 6H digest must not be inferred or reused after its resume.

The agent's sequence is:

1. Stop alpha with `checkpoint-6i-stop-alpha`.
2. Inspect with `checkpoint-6i-inspect-alpha` and record the exact inventory/digest.
3. Preview cutoff 0i0; verify the selected/retained sets and deletion eligibility.
4. Submit the digest-bound, explicitly confirmed request
   `checkpoint-6i-prune-alpha`; verify all four file removals, epoch removal, three
   retained epochs, and completed non-uncertain evidence.
5. Replay the exact request while stopped; compare the complete operation result.
6. Resume with `checkpoint-6i-resume-alpha`; verify healthy 2/2 state and direct
   HTTP `~zod` identity.
7. Replay the same prune request while running; verify no new deletion and no stale
   inventory resurrection. Preserve the pruning report and operation evidence.

For manual reproduction, define the helper in the terminal where it is used:

```bash
collect() {
  /home/thelifeandtimes/repos/omabit/build/urbit-collector \
    --format text --runner workstation \
    --expected-runner-id runner-f496feed1f7b08f31a5d82b94ac2f5fd "$@"
}
collect snapshot
free -m
```

Use the revision from fresh inspection. For alpha still at revision 2, stop and
wait for success, then inspect and wait again:

```bash
collect operate alpha stop --revision 2 --request-id checkpoint-6i-stop-alpha
collect watch --interval 2s
```

Ctrl+C the watcher after clean stop, then:

```bash
collect operate alpha inspect-epochs --revision 2 --request-id checkpoint-6i-inspect-alpha
collect watch --interval 2s
```

After inspection succeeds, Ctrl+C the watcher and run `collect snapshot`.
Copy the actual digest into the following preview and confirmed request (replace
the placeholder; it is not a shell variable):

```text
collect epoch-preview alpha --revision 2 --cutoff 0i0 --expected-digest <actual-digest>
collect epoch-prune alpha --revision 2 --cutoff 0i0 --expected-digest <actual-digest> --request-id checkpoint-6i-prune-alpha --confirm-history-loss
```

Wait for succeeded/stopped with `collect watch --interval 2s`, then Ctrl+C the
watcher. Preserve the pruning snapshot and replay the exact confirmed command.
Resume only after the completed result has no uncertainty/recovery requirement:

```bash
collect operate alpha start --revision 2 --request-id checkpoint-6i-resume-alpha
collect watch --interval 2s
```

On healthy 2/2 state, Ctrl+C the watcher and replay the original prune command.
The request's original digest remains correct for exact replay even though the
inventory has changed; replay lookup precedes current-state admission.

## Local evidence and remaining work

Passed checks:

- `go test ./...`
- `go vet ./...`
- `go test -race ./internal/epochs ./internal/runtimequalify ./internal/managed ./cmd/urbit-collector ./cmd/urbit-runner`
- `go build -o ./build/ ./cmd/...`
- Linux/ARM64 runner cross-compilation (build-only).
- Built collector's `epoch-prune --help` and `git diff --check`.

Tests cover exact managed unlink scope, durable intent/confirmation, stale disk
and cached inventory, partial deletion, failed persistence, quarantine after
cleanup/restart, host-gate serialization, configuration exclusion, observer
refusal, capability refusal before submission, and replay after resume/restart.
The existing disposable qualification also continues to exercise the shared engine.

General selected-prefix pruning, deletion of non-genesis snapshot-bearing epochs,
older-epoch recovery, automatic scheduling, and production hosting remain separate
gates. Allocated byte counts are not a guarantee of reclaimed disk space.
