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

The typed requests also exercise desk enumeration, immutable-revision file paths
and actual Behn autocommit timer detection (inactive → active → cancelled).
Unknown query output is an error, never evidence that autocommit is disabled.
Probe14 is the first complete pass for this extended set; probe13 retains the
failed Hoon inference query. See M539 and the discovery diagnostics record.

Probe19 extends the typed set with canonical Clay hashes, batched MIME conversion,
and a revision/hash-bound direct Clay delta. It verifies additions, replacements
and deletions. It also records that raw invalid Hoon source can be accepted into
Clay; this is not an application compilation or readiness pass. Those stages
require their own configured checks. Probe17/18 retain the incorrect old-revision
verification and disproved compilation-rejection assumption respectively.

## Assembled native workflow

Compile the managed test binary and pass it with `--workflow-binary`. The launcher
starts only `TestNativeDeskWorkflow` after private ship readiness. The test refuses
ordinary host execution and checks namespaces, admitted scope, loopback-only
network and the owned child's executable hash/start time before Lens requests.

```sh
go test -c -o /tmp/omabit-desk-workflow-native.test ./internal/managed
```

Probe21 passes the real kernel + API/store + collector/Unix transport composition.
It covers default gates, creation/discovery, source alerts, frozen staging, verified
commits preserving scaffold, add/update/delete, lost acknowledgement, read-only
reconciliation after permission revocation, and both gates on a published desk.
Source publication and mounting are explicit fixture operations, not public API
claims. The lifecycle backend is synthetic; Docker, SSH, compilation, activation,
Treaty, receivers and frontend delivery remain separate checks. See M545 and
`docs/diagnostics/2026-11-07-desk-native-workflow.json`.

Native effect dispatch/acknowledgement and managed receipts are retained. Reuse
refuses unresolved native effects or managed operations, including a harness
failure. Never discard those receipts to permit reuse. Probe20 stopped before
runner state existed because the harness omitted directory creation; it mutated
no desk. The repaired probe21 shut down cleanly.

## Application stages, gated releases and source adoption

With `--release-source` pointing at the frozen fixture041 tree, probes34–36 add
application-stage observations, source/Treaty publication, local activation and
native authenticated Docket upload with exact served HTML. Upload changes the
Clay docket and advertised Treaty hash; the old staged source correctly conflicts.
Probe34 also records an invalid installed-agent update returning HTTP500 with
unchanged subsequent Clay. Its dispatch remains uncertain, so ordinary reuse is
forbidden. This is not proof of rejection or permission to retry.

Probe35 routes activation/source/Treaty publication through the gated public
release API/collector and verifies default blocking and exact replay. Probe36
resumes the clean35 pier, selects one existing scaffold file explicitly, verifies
adoption's default block and scoped grant without any Clay commit, and replays its
receipt after stage discard. Other scaffold files stay unowned; subsequent staged
commits and release checks pass. Both runs stop cleanly, with all effects accounted
for. See M546–M548 and their diagnostics. Frontend upload still uses a private
helper; managed Docker/SSH, receiver rollout and Urgit remain separate acceptance
work. Neither live-agent status nor a source revision proves application health.


## Frontend artifact qualification and read-only diagnostics

Native48/M552 verifies predicted versus actual Docket glob hash, the exact bytes
and MIME of HTML/binary/empty assets, the resulting Clay revision, source ownership
conflict and empty-source normalization/commit/comparison. Its internal upload
client is not yet a public gated frontend command. See the artifact diagnostic.

`--frontend-diagnostic --resume-from PATH` transfers only a verified stopped fake
from this capsule. It retains the original random desk and performs fixed Lens
metadata and authenticated asset reads, followed by logout and clean shutdown.
It refuses workflow binaries, release source, receivers and effect-recovery flags;
it declares1GiB scratch. A failed frontend verification may be inspected this way
without authorizing another mutation. Diagnostic46/47 investigated native45's
acknowledged upload/hash mismatch without retrying it. Fresh48 proves the fix.
The original failure record remains authoritative for that historical run.
