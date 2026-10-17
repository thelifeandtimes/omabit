# Runtime qualification record

The all-ship successor has source/private tests and a successful separately approved
management-only update. Livenet socket execution remains untested. See
[revision02](runtime-settings-validation.md) for the evidence boundaries.

The configurable loom and development-fake native-control gate passed on fresh
managed fakes. Evidence includes actual loom34, private native replies, Clay bytes
before/after replacement, default-disabled behavior and owned Stop. Historical
default-disabled gates below remain separate. See the
[settings report](runtime-settings-validation.md) and
[acceptance procedure](development-runtime.md#approval-gated-managed-acceptance).

Status: rootless setup, lifecycle, maintenance, host-loopback HTTP, service startup
and clean host reboot/resume are accepted for the development fleet. This is still
partial qualification, not general production-runtime support.

## Accepted engine setup

The user reported successful checkpoint 2A installation and checks:

- Dedicated account `urbit-runner`, UID/GID **959/959** on this host only.
- Service `urbit-docker.service`; socket `/run/user/959/urbit-docker.sock`.
- Installer result: `account_created: true`, `rootless_verified: true`,
  `runtime_qualified: false`, `partial: false`.
- Subsequent read-only metadata inspection found Docker 29.7.2, RootlessKit 3.1.0,
  and slirp4netns 1.3.5 installed. No developer Docker API call was made.

This established the engine setup gate. Container/runtime evidence is recorded
separately below; the subsequent [service/reboot checks](services.md) passed.
Ames/remote networking remains a separate gate.

## Accepted isolated lifecycle (checkpoint 2B)

User-supplied report for run `35aac1083d71188d9782ec64b49ba771`:

- Every phase passed: engine, image, pill, probe, boot, restart, pack, post-pack,
  and cleanup.
- `checkpoint_passed: true`, `cleanup_complete: true`, `limits_observed: true`,
  `pier_metadata_seen: true`; `runtime_qualified: false` as designed.
- All five containers were removed; every create outcome was resolved.
- Docker 29.7.2 reported the pinned manifest as its backend image ID; Vere
  reported `4.6-5a7614f`.
- Available memory at admission: 7,298,469,888 bytes; available disk:
  453,037,428,736 bytes. These are observations from that run, not reservations.
- The verified 217,421,172-byte pill matched the generation and SHA-256 below.
- The fake pier and report remain in the account's private qualification storage.
  No existing pier was adopted.

This is operator-reported execution evidence, not an independent developer rerun.

Checkpoint 1 has been accepted by the user. The qualification pill will initially
be the upstream default bootstrap pill for the selected Vere version, not the
unreleased Omabit pill. Record its resolved URL and content digest for repeatable
checks; verify the expected base, webterm, and groups desks after boot.

Before installation, read-only preflight found x86-64 Linux, systemd, and Docker
available. The rootful Docker socket was enabled but inaccessible to the desktop
user, as intended by Quattro. Rootless setup scripts, RootlessKit, and a networking backend
were absent. No engine was activated and no host configuration was changed by
that preflight.

The user confirmed dedicated-account rootless Docker as the default. Preserve
Quattro's existing system Docker services and the desktop's context and group
membership. Rootless installation has its own reviewed OS-authorization step;
there is no automatic privileged fallback.

Run only after the protocol walkthrough is accepted and the user approves any
host setup. Use disposable development ships. Never boot an existing identity's
copy while its original is running.

## Accepted host HTTP (checkpoint 2C)

User-supplied report for run `e9f766ae9854e04f35eb9991d39fb9ba`:

- Engine, image, pill, probe, network, host-http-boot, host-http-restart, and
  cleanup all passed.
- `checkpoint_passed: true`, `cleanup_complete: true`, `limits_observed: true`,
  and `pier_metadata_seen: true`; `runtime_qualified: false` as intended.
- Host endpoints were `http://127.0.0.1:32768/~/host` and then
  `http://127.0.0.1:32769/~/host`, both verified. Those containers are now removed;
  the addresses are historical evidence, not active servers.
- All three containers and network
  `fda5e86036dc00a5c9ed134ab3c41ac33f162174136bef5d6b84ee49bc8802b0` were removed;
  every create outcome was resolved.
- Admission observed 5,396,324,352 bytes available RAM and 452,211,556,352 bytes
  available disk. Image, Vere, and pill pins matched the accepted 2B candidate.

This confirms local publication/reconnect on the dedicated NAT bridge. It does
not establish remote/tailnet access, application authentication, or Ames.

## Record before testing

- Host distribution, architecture, systemd, kernel, Docker version.
- Dedicated account and rootless engine setup, including unattended startup.
- Image reference/digest, embedded Vere version, entrypoint and supported flags.
- Pill release/version, trusted metadata source, URL, digest, included desks.
- Pier/artifact paths, ownership, limits, port bindings, and network policy.
- Released runtime identity and disabled control socket (`--no-conn`).
- Commands, results, and redacted logs for every check below.

## Checklist

| Area | Required observation | Result |
| --- | --- | --- |
| Isolation | No desktop Docker-group grant or rootful socket exposure | Accepted in checkpoint 2A (operator-reported) |
| Image | Correct x86-64 binary and explicit invocation | Passed in 2B |
| URL pill | HTTPS source downloads and verifies before boot | Passed in 2B |
| Local pill | Explicit import verifies before boot | Not run |
| Download errors | Missing source, redirect policy, oversize, timeout, and bad hash fail closed | Not run |
| Bootstrap | Expected identity mode and desks; no unrelated personal state | Default fake and named `~bus` boot passed; desk/content checks pending |
| Named development fakes | Independent local instance, fake identity and confined pier basename; same-identity replacement | Panel 0.13 / runner accepted live for `bus-lab`, `~bus`, `bus-data`; registration replay, epoch inspection and initialized resume passed; see [evidence](diagnostics/2026-10-01-named-fake.md) |
| Persistence | Stop/start preserves pier; containers are replaceable | Local replacement restart passed in 2B; application data semantics pending |
| Networking | Private HTTP access and intended Ames connectivity | In-container HTTP passed in 2B; host-loopback publication/reconnect passed in 2C; Ames/remote access pending |
| Shutdown | Graceful signal handling and adequate stop timeout | Passed in 2B |
| Reboot | Lingering engine/management and dedicated SSH startup; eligible ship resume | Passed September 30 on boot 6f75f84d-7c19-4783-af44-94286ab79d5f: four successful automatic resumes, preserved identities/revisions, uninitialized registration skipped, fleet healthy |
| Offline pack | Exclusive pier access and successful restart | Passed in 2B; managed pack/replay/resume passed in 6A |
| Offline meld | RAM admission and successful restart | Disposable 6B and managed 6C meld/replay/resume passed; larger-pier budgets pending |
| Container metrics | Exact raw memory/task counters with source and freshness | Passed in 6D for both running development instances |
| Container CPU | Same-container paired counter deltas, per-core percentage and labeled time charts | Panel 0.15 live readings passed on alpha, bus-lab and comet-check with unchanged containers; see [CPU evidence](diagnostics/2026-10-01-cpu-and-layout.md) |
| Loom metrics | Internal counters, utilization and pressure advice | Deferred; control socket disabled under the official-release-only policy |
| Epoch inventory | Pinned-layout metadata, protected suffix, stable preview binding | 6E single-epoch live inspection/guards/replay/resume passed; positive prefix selection fixture-validated |
| Offline roll | One newer snapshot-bearing epoch and successful restart per roll | Disposable 6F and three managed 6H roll/replay/resume cycles passed |
| Epoch pruning | Exact selected prefix and retained-suffix recovery | 6G disposable genesis deletion/latest-epoch recovery and 6I managed genesis prune/replay/resume passed; broader prefixes pending |
| Broader prefix pruning | Snapshot-bearing and repeated deletion with restart/recovery | 6J five-to-three-to-two live qualification passed, including both latest-epoch recoveries |
| Managed prefix pruning | Confirmed cutoff, repeated requests, replay and resume | 6K live five-to-three-to-two, retained metadata, historical replay and healthy resume passed |
| Login-code access | Bound operator retrieval, temporary Eyre login/logout, redacted audit | 6N live code/Eyre checks passed; real desktop clipboard expiry follow-up remains |
| Stopped-pier transfer | Verified private copy, two target resumes, original preserved | 6O live run 185ff2ed58b9abdd6861dfe64d6650f5 passed; 7 files / 1,016,840,204 logical bytes, both resumes and original preservation verified |
| Managed qualified-fake import | Durable reservation/copy, progress, replay and explicit resume | 6P live import-check copy, replay, duplicate-source refusal and healthy 1/1 start/stop/resume passed |
| Fresh comet bootstrap | New internally generated identity and same-identity replacement | 6Q live run cb31b1d255ffe45103cf90ed7458a593 passed; host Ames/peer connectivity remains separate |
| Managed comet lifecycle | Durable registration, bootstrap quarantine, identity-pinned resume | 6R live comet-check creation, replay, same-identity replacement, telemetry and code redaction passed |
| Routine maintenance | Durable slot admission and explicit stop/pack-or-meld/resume | 6S live timed fake-pier stop/pack/resume passed; one slot, healthy return, test policy disabled |
| Comet offline maintenance | Isolated pack/meld and same-identity restarts | 6T passed run 5418c87ccd3aac4ef8a1597adedeae0c; both utilities, all restarts and seven-container/bridge cleanup resolved |
| Managed comet compaction | Durable pack/meld and UTC routines with pinned identity | 6U manual pack/replay/resume and one timed meld/resume passed; exact identity retained, healthy 1/1, test policy disabled |
| Comet epoch lifecycle | Three rolls, snapshot inventories, same-identity restarts and preview | 6V passed e84ffd83afcfbbe71e6cceca2d69c30f; four epochs, unchanged identity, nine-container/bridge cleanup |
| Managed comet epochs | Durable inspection/roll, explicit identity-pinned resume and preview-only retention | 6W three-cycle gate and final inspection/resume passed; four epochs retained, confirmed deletion refused, healthy 1/1 |
| Comet prefix deletion/recovery | Repeated exact prefixes, protected newest-two suffix and latest-epoch replay | 6X passed 198fec0f0e49df2bcb0215789f2b5f1d; two recovery checks, unchanged identity, newest two retained and complete cleanup |
| Managed comet pruning | Confirmed oldest prefixes, durable progress, retained metadata and pinned resume | 6Y passed five-to-three-to-two pruning; 0i7082/0i7847 retained, replay/refusal checks and same-identity resumes passed, healthy 1/1 |
| Resource limits | Live memory, swap, PID, and CPU cgroup values | Observed in 2B; overload/failure enforcement tests pending |
| Runtime interface | No control-socket use or executable overlay | Official image binary only; runtime commands retain --no-conn |
| Failures | Low disk, OOM, killed job, and invalid pier are bounded and diagnosable | Not run |

## Preparation evidence

- New `preflight` and `setup-plan` commands inspected this host without subprocess,
  Docker API, or network use and identified the missing rootless prerequisites.
- `prepare-rootless` fetched the immutable Docker 29.7.2 launcher and legal files
  over HTTPS and verified their compiled-in content pins.
- Generated scripts passed `sh -n`; systemd static verification only complained
  that the intended installed executable paths do not yet exist.
- Fixture tests cover the elevated first-install path without executing actual
  administrative commands. Bundle contents are reverified independently of its
  manifest before any host mutation.
- No packages, accounts, services, Docker contexts, group membership, or piers
  have been changed during developer validation. The existing rootful service
  remains inactive/disabled and its socket remains active/enabled.

Candidate dependency baseline: package-managed Docker 29.7.2, repository
RootlessKit and slirp4netns, and the launcher from Moby commit
`6a43e3d5afddf4111da0f864bbc7cae5d7e95001`. A changed engine version requires a new
reviewed baseline, not a bypass or a system downgrade. Preserve the package
manager's normal security-update workflow.

## Gates

Checkpoint 6J adds a fresh five-epoch disposable workflow with two distinct prefix
deletions (five to three, then three to two). It includes snapshot-file deletion,
exact retained fingerprints, and ordinary restart plus latest-epoch recovery after
each round. Its [live gate](checkpoints/20-repeated-prefix.md) precedes expanding
managed pruning beyond the accepted genesis-only scope.

Checkpoint 6G adds the qualification-only `epoch-prune-recovery` profile. It
recreates a fresh four-epoch fake, removes only the revalidated oldest epoch,
verifies three retained per-epoch metadata fingerprints, and tests ordinary
restart plus latest-epoch `play --full` recovery after deleting the current
checkpoint. [Live qualification](checkpoints/17-prefix-prune-recovery.md) passed with
run `e31441cd9345af32611b0153afc0cff7`.
Source review confirms that pinned `play --full` starts at the latest epoch;
older-retained-epoch recovery and managed pruning are separate gates.

Checkpoint 6F introduces `epoch-lifecycle`: a fresh isolated fake, three explicit
offline rolls with exact epoch-growth checks and snapshot presence, three same-pier
restarts, and a non-executable preview retaining three epochs. Its
[live gate](checkpoints/16-epoch-lifecycle.md) precedes any selected-prefix deletion
or retained-suffix recovery claim. Explicit managed roll is implemented in
[checkpoint 6H](checkpoints/18-managed-roll.md), whose live gate passed.

Checkpoint 6B passed with run `9b5dfcb57a677f7cf1a2b833e6f65274`; see the
[operator-reported evidence](diagnostics/2026-09-25-offline-meld.md). It adds the
separate `offline-meld` profile and readable qualification
output. Its [live walkthrough](checkpoints/12-offline-meld.md) creates a fresh
isolated fake, rechecks RAM/disk before meld, records logical file-size evidence,
and requires same-pier restart. An uncertain utility outcome remains quarantined
after cleanup. The qualification holds the management-process lock through
cleanup; the user must pause management for this gate. This does not enable the
managed meld operation or establish populated-pier memory requirements.

Checkpoint 2C introduces a separate host-HTTP profile. It uses a dedicated normal
NAT bridge (with outbound access), not `--internal`, because the pinned Docker
gateway path is needed for NAT port publication. Only `127.0.0.1` and an ephemeral
TCP host port may be published. HTTP identity is checked directly from the host,
without credentials or redirects. Container replacement rediscovers the binding;
cleanup tracks both container and network intents. The accepted live execution
is recorded above.

Checkpoint 2B has fixed image/pill provenance, injected tests, and the accepted
execution record above. The image is pinned to
`sha256:47fbcf0710ded939931ce3c6408864cee5e8848c5765da0b51a90ca0b1ab1afd`;
the default 4.6 pill is pinned to SHA-256
`f8b08c53dd6d0aa9c4f7654f0a7bcd11e94089222888a3c99c0a10eedcbb377b` and
217,421,172 bytes, using object generation `1790171686269740`. The mutable default
was replaced during development; the old object was no longer publicly
retrievable. The new generation's hash was independently computed from its full
HTTPS content; no separate publisher signature was verified.

Do not enable continuous socket collection on an affected runtime. A draining
client is mitigation, not proof against collector hard-kill/OOM.

If rootless Docker does not meet a required capability, capture the evidence and
ask before installing a privileged alternative. Do not change existing Docker
membership, start real ships, or configure system services as an implicit test.

Epoch pruning needs its own later format/recovery qualification. Ordinary startup
success is not enough to claim retained history remains replayable.
