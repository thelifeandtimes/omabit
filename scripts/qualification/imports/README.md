# Isolated import qualification

Read `docs/experiment-isolation.md`. Never use the active runner, Docker daemon,
installed plugins, managed ships, tmux ships or `~/temp-ur` in this suite.

Current drivers:

- `scripts/test-import-ui-offscreen.py` runs actual import/recovery QML and service
  callbacks with inert command dispatch, private HOME/XDG and offscreen Qt.
- `smoke.py` checks the minimal QEMU/TCG boundary without disks or a network card.
- `packages.py` resolves a separate offline guest package set from copied sync
  databases. It downloads and verifies signatures against the installed public
  Arch/Omarchy keyrings and rejects revoked signers. It never installs host packages.
- `runtime.py` downloads the repository-pinned official image manifest, config and
  layers, checks every digest/size and the platform, and builds an OCI archive. It
  rechecks a previously downloaded pinned pill. It never contacts a Docker daemon.
- `appliance.py prepare` constructs a fresh full-system guest disk without mounts
  or elevated host filesystem ownership. Package extraction runs in Bubblewrap;
  only the new root tree is writable. Execute-only staging files gain owner read
  permission so the unprivileged disk builder can copy them. The guest reinstalls
  all signed packages to restore native metadata before testing the installer.
- `appliance.py boot` starts a fresh overlay on that immutable disk. Guest PID 1
  checks its marker and QEMU identity, normalizes ownership **inside the VM only**,
  and starts real systemd. The guest verifies and installs the cached native
  packages, creates its own test account/key, invokes the actual Omabit owner
  installer and checks the rootless engine and package compatibility. An optional
  staged runtime adds OCI loading, exact image identity and a no-network Vere
  version container. No ship is created by the default environment stage.
- Adding `--exercise-import` to appliance preparation includes `import_case.py`.
  After the environment checks it boots a new fake `~zod` in a no-network source
  container, verifies a clean stop, and tests the actual enrolled collector's guided
  preview, import, completed-request replay, explicit Start, identity, parent-only
  mount and explicit Stop. It retains the pier and all receipts in the guest disk.
  `--import-mode copy` adds a fresh 4 GiB disk with an exact lab serial and a
  24 GiB root disk. Before starting the owner services, the admitted guest verifies
  the extra disk's serial, size and empty header, formats/mounts it, and boots the
  source there. Import must cross devices, account for every byte/file, normalize
  private permissions and remove the verified holding source. Formatting and mounts
  occur inside the VM only; no host device is passed through.
  `--interrupt-copy` additionally stops only the guest owner's runner service after
  observing partial-copy progress, restarts it, verifies the full holding source
  and partial destination survive, and checks the current `invalid_state` recovery
  refusal without tree changes or an implicit runtime. Its result is explicitly
  `partial-copy-preservation`, not a successful recovery/import completion.
  Adding `--restore-copy` extends that case with the new runner/CLI restoration
  flow: review both moves, explicitly approve the digest, verify the same source
  inode returns to its original name and the partial destination is retained in
  archive, then replay both terminal requests. It re-imports with a fresh request,
  verifies the next registration revision and old-request replay against that new
  registration, and runs the explicit Start/identity/mount/Stop checks. The empty
  old holding parent and archived partial tree remain untouched as evidence.
  This flag requires `--exercise-import --import-mode copy --interrupt-copy`.
  Adding `--restore-ui` with `--restore-copy` sends that restoration through the
  actual offscreen `StoragePane`, restoration controller, consent button and
  verbatim Service callbacks. The guest-only PySide6 bridge launches the real
  collector and polls its real inventory; it admits only the selected fresh
  guest's preview/restore arguments. Independent CLI and filesystem assertions
  surround the UI phase and retain replay/re-import/Start/identity/mount/Stop
  checks. This requires packages prepared with `packages.py --restore-ui`.
  Only inert style values and the subprocess adapter replace Quickshell desktop
  integration; no host desktop is mounted. Both review and retired-instance
  completion screenshots are retained inside the guest result directory.
  The driver being present is not evidence of a runtime pass; results are recorded
  separately below and in the diagnostics reports.

Use the signed private QEMU bundle prepared by `tools()` in
`scripts/qualify-powercut.py`. Its packages are extracted, never installed. Supply
an explicit x86-64 kernel and use a fresh output for every preparation and run:

```sh
python3 scripts/qualification/imports/packages.py \
  --output /tmp/omabit-import-guest-packages-FRESH_NAME
python3 scripts/qualification/imports/runtime.py \
  --pill /tmp/VERIFIED_ARTIFACTS/boot.pill \
  --output /tmp/omabit-import-runtime-FRESH_NAME
python3 scripts/qualification/imports/appliance.py prepare \
  --packages /tmp/omabit-import-guest-packages-FRESH_NAME \
  --kernel /usr/lib/modules/EXPLICIT_RELEASE/vmlinuz \
  --runtime /tmp/omabit-import-runtime-FRESH_NAME \
  --output /tmp/omabit-import-appliance-FRESH_NAME
python3 scripts/qualification/imports/appliance.py boot \
  --appliance /tmp/omabit-import-appliance-FRESH_NAME \
  --tools /tmp/PRIVATE_QEMU_BUNDLE \
  --output /tmp/omabit-import-guest-run-FRESH_NAME
```

The VM has no NIC, host filesystem shares, engine socket, keyring, desktop IPC or
SSH agent. QEMU runs in a new user/network/mount/PID namespace with read-only OS,
tool and artifact mounts and only its fresh lab directory writable. Software TCG
uses one core. The environment guest has 3 GiB RAM and an 8 GiB scope cap; the
import case has 7 GiB RAM and a 12 GiB scope cap to satisfy the runner's normal
5 GiB available-memory admission. Both disable swap and cap the scope at 128 tasks.
The import guest exposes two virtual CPUs to satisfy the runner's two-CPU container
limit, while the outer scope still caps total execution at one host core. The boot
driver reads back the exact active scope limits before admitting a run. The scope also accounts for writes to the memory-backed
`/tmp` disk, so its cap must exceed guest RAM. A 4 GiB trial hit that cap and is
retained as a failed run, not a successful installer test.

The environment driver has a 25-minute deadline (40 minutes for an import case)
and a 16 MiB console bound. Guest steps have
their own deadlines; normal completion powers off through systemd. Only the
spawned process group is terminated on a driver limit. Observe the terminal
result before launching another run; an observation timeout is not proof of exit.
Successful exit requires the guest result and clean poweroff. Record the scope
result, including an OOM kill, alongside package/tool/source/artifact hashes.
Preserve original disks and logs for failed or uncertain operations.

`storage.py` also reads the current user's filesystem quota, because filesystem
free space alone misses the per-user `/tmp` limit. Prepare/boot require 8 GiB
headroom. The boot driver checks it every two seconds and stops only its own
process group below a 1 GiB reserve or on a quota-query failure. A 1 MiB allocated
diagnostic reserve is released before writing its terminal result. This currently
requires Linux x86-64. Keep concurrent storage use bounded; these checks are not
a reservation of the entire run's disk budget. Redundant converted inspection
images may be removed after verifying their original overlay/backing hashes;
retain the original disks and extracted receipts.

The same-filesystem rename case passed on run 05; its frozen receipts are in
`docs/diagnostics/2026-10-22-import-runtime.md`. The copy variant passed on run 08, recorded in
`docs/diagnostics/2026-10-22-import-copy.md`. Failure/recovery and further
layout scenarios also remain separate runtime gates. The `--restore-copy` variant
passed on run10: source restoration, partial-target archival, terminal replay,
fresh re-import and explicit Start/identity/mount/Stop. See
`docs/diagnostics/2026-10-22-import-restore-runtime.md`.
The `--restore-ui` variant passed on run11 with the real collector/runner and
retained independent filesystem/replay/re-import/Start/Stop checks; see
`docs/diagnostics/2026-10-22-import-restore-ui-runtime.md`.
Each must use newly created fake data and the real runner/collector. An installer or version-container
pass does not establish any of those import behaviors. Do not bypass native
package or pinned runtime validators to make a test pass.

Use `extract_results.py --run COMPLETED_RUN --tools PRIVATE_QEMU_BUNDLE --output
/tmp/omabit-import-inspection-FRESH_NAME` after terminal completion to extract
private receipt files. It refuses an active scope, converts a read-only view of the
disk inside Bubblewrap, and uses debugfs without mounting the guest filesystem.

## Shared VM boundary for fake-network experiments

`--exercise-fake-network` is mutually exclusive with `--exercise-import` and uses
the same fresh signed appliance, no-NIC VM and guest-only rootless engine. See
`../fake-networks/README.md` for its transport/restart assertions and limits. It
uses the runtime resource scope and a 50-minute outer deadline for two fresh boots;
it neither formats the copy source disk nor runs the import case.


## Opt-in administrative endpoint discovery

`prepare --exercise-import --exercise-login-code` adds a real runner/collector
code-read gate before and after same-pier container replacement. It verifies
runner/instance/revision/container binding, uses the returned code for one direct
Eyre login and authenticated scry, and confirms logout. Code and cookies remain
in memory outside the command logger; receipts contain booleans and nonsecret
identities only. `test_login_case.py` covers binding/refusal/redaction/cleanup with
inert transports. This fixture is prepared source, not a runtime pass. It tests
normal administrative allocation; a relocated port remains separately covered by
source tests until shared runner projects are implemented.

## Runner project lifecycle and application recovery

`--exercise-project-member-launch` runs the private queued project lifecycle,
including two same-pier replacements, readiness, Stop, receipt replay/reopen and
exact owned-resource cleanup. `--exercise-project-app-state` additionally preserves
Eyre sessions, application writes and subscription cursors across both replacements.
The default app fixture stops the peer first, so only its initial writes are remote.

`--exercise-project-live-peer` requires both flags and selects a separate test entry:
the original peer remains running and sends all recovery writes. Its process,
container and configuration must remain unchanged. Peer shutdown at the end is
fixture cleanup, not a newly qualified queued Stop after a changed peer. Each mode
has a distinct guest admission token; none runs on the host. Current outcomes and
source-versus-runtime limits are in `docs/goal-progress.md`.


`--exercise-project-current-readiness` requires the live-peer mode and selects a
separate admission token. After the receiver's second replacement, the private
runner queue observes both current members without changing their original
readiness receipts. The fixture checks exact creator/generation bindings, callback
ordering, request replay and reopening both new journals. It retains the existing
application and live-peer checks. This observes readiness only; the original
peer's final shutdown remains exact-ID fixture cleanup. Source compilation and
host-refusal checks do not establish a runtime pass.

`--exercise-project-context-lifecycle` additionally requires the current-readiness
mode. Its separate guest token selects `TestProjectContextLifecycleDisposableVM`.
After two receiver replacements, it uses refreshed readiness to stop the original
sender, observes its exit, retires only its container, resumes its retained pier
and observes replacement readiness. The receiver's exact process/configuration
remains unchanged throughout this sender cycle. Both members are observed again
before final queued Stops. All lifecycle receipts are replayed and reopened.

This mode adds `before-peer`/`after-peer` application phases. Twelve acknowledged
remote writes span three recovery cycles, using the two original Eyre sessions.
The final cycle checks a queued update and existing receiver subscription across
the sender replacement. Phase intent is persisted before effects; interrupted
phases and writes are never automatically replayed. This describes the fixture's
assertions, not a runtime pass; qualification requires the frozen appliance and
independent stopped-disk checks.

Future runner application fixtures use `http_transport: published-loopback` with
exact guest-only public ports and runtime-tmpfs session files outside the anchor.
The older absent field keeps namespace transport for frozen fixtures. Source
checks pass; see `docs/diagnostics/2026-10-23-runner-published-probe.md` for the
separate pending runtime gate and unchanged frozen VM26. A new anchor namespace
still needs its own retained recovery evidence; transport changes do not adopt it.

`--exercise-project-anchor-review` requires the full context-lifecycle case and
selects the separate `TestProjectAnchorReviewDisposableVM` entry/token. After
all six app phases and both current-readiness observations, it verifies a healthy
project, then injects an exact-ID Stop/Start of this guest's own anchor. The
production read-only adapter must classify the stopped anchor and the restarted
anchor's changed namespace while retaining every member process and old binding.
It must leave all operation journals unchanged. This case uses the new direct
published client and session files outside the anchor.

This is diagnostic qualification, not coordinated recovery: the two final members
use the existing exact-ID fixture cleanup instead of queued final Stops. Earlier
queued Stops/replacements and every retained journal still undergo reopen checks.
The ordinary context-lifecycle case retains its final queued Stops. Host execution
and missing flag prerequisites are refused before effects. The new entry has
source checks only until a separately frozen appliance completes and its disk
receipts are independently verified; VM26 predates this entry.


## Private keyed-chain appliance launch

The shared `appliance.py boot` entry point now recognizes an explicitly prepared
`exercise_private_chain` appliance. It refuses mixed runtime cases and changed
pristine backing bytes, uses a separate read-only backing mount and a fresh overlay,
and admits two-core TCG only for an explicit runtime case. This case requires16GiB
available host memory, uses7GiB guest RAM with a12GiB/no-swap/128-task scope, retains
the8GiB storage admission and2GiB running reserve, and has a3600-second deadline.
Existing import/fake cases retain their previous resource/storage policies.

Five inert launch-boundary checks in `test_private_chain_boot.py` replace all
subprocesses. The actual private-chain runtime settings passed M346 using the
frozen experiment driver; the newly integrated validation adds malformed/mixed-case
refusal. The reusable private-chain preparation command is still under development.
The ordinary `prepare` command does not create such an appliance, and a metadata
flag alone does not turn a normal appliance into a private devnet.
