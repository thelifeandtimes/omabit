# Private desk protocol development probe

This is an isolated native fake-ship probe, not a release rehearsal or an
installer. Read `docs/experiment-isolation.md` first. The host entry point accepts
only the recorded Vere/pill hashes and a fresh `/tmp/omabit-desk-probe-*` output.
It creates a private user/network/PID/mount/IPC/UTS namespace and a capped scope:
4 GiB RAM, no swap, one CPU, 128 tasks, a 550-second deadline and 2 GiB scratch.
No host HOME, Docker socket, SSH agent, keyring or external network is mounted.

Generate the actual typed Go requests before starting the probe:

```sh
go run ./scripts/qualification/app-development/requests.go > /tmp/omabit-desk-requests.json
python3 -B scripts/qualification/app-development/probe.py \
  --output /tmp/omabit-desk-probe-FRESH \
  --requests /tmp/omabit-desk-requests.json \
  --vere /absolute/path/to/verified/vere \
  --pill /absolute/path/to/verified/boot.pill
```

Use a lowercase output suffix. Check the retained experiment budget before each
run. The output records input hashes, exact namespace/scope admission, each
submitted expression and its result, runtime logs and clean-shutdown status.
Every run uses a new desk name. No unknown mutation is retried. Results must be
reviewed separately from process exit; compare exact acknowledgements, observed
revisions, mark-equivalence positive/negative controls, timer state and source
publication state.

For development iteration, `--resume-from /tmp/omabit-desk-probe-PREVIOUS` accepts
only a stopped fixture with matching inputs and no uncertain mutation. It records
and moves that exact fake pier into the new run, preserving device/inode and the
prior result hash. It does not copy or find user piers. Only this private fixture's
own failed read job may be cancelled before resuming. A production adapter must
not cancel shared Lens work as automatic error recovery.

The first complete pass is recorded in
`docs/diagnostics/2026-11-06-desk-protocol.json`. Prior failures remain useful:
absent scries can leave Lens waiting; malformed expressions can leave persistent
busy state; raw Unix/Clay bytes can differ due to mark conversion; the pinned
pill lacks the cancellation convenience generator but accepts the typed Kiln
mark. Earlier pruned runs contain receipts, not reusable piers or runtime copies.

The probe does not qualify managed Docker selection, remote transport, policy
persistence, Treaty publication, receivers, frontend delivery or Urgit. Those
remain separate workflow checks.
