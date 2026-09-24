# Implementation plan

Work in independently reviewable commits. Automated tests do not replace manual
acceptance gates. When a runnable checkpoint is available, document exact steps,
expected results, limitations, and cleanup, then pause for the user's validation
before moving across that gate. Never use a production pier or identity for an
automated test.

## Progress

- Architecture and sequence: recorded.
- Checkpoint 1 protocol/simulator: implemented; automated tests and user acceptance
  of the [local walkthrough](checkpoints/01-local-protocol.md) passed.
- Protocol contract: [experimental version 1](protocol.md).
- Real Docker/runtime qualification: isolated lifecycle and local host HTTP have
  passed; Ames/remote networking, application health, meld, reboot, and failure
  gates remain.
- Qualification pill: use the selected Vere runtime's upstream default bootstrap
  pill until the Omabit release is available. Verify the expected base, webterm,
  and groups desks after boot; this does not qualify the future Omabit bundle.
- Default deployment is confirmed: dedicated-account rootless Docker, with the
  same intended five-step plugin onboarding. Existing Docker services, contexts,
  and desktop group membership remain unchanged. No automatic privileged fallback.
- Checkpoint 2A rootless setup: accepted by the user. The dedicated account is
  UID/GID 959 on this host, with socket `/run/user/959/urbit-docker.sock` and
  `rootless_verified: true`. Runtime qualification remains false; the UID is not
  a portable configuration default.
- Checkpoint 2B is accepted: run `35aac1083d71188d9782ec64b49ba771` verified
  image/pill pins, live cgroup limits, fake boot, clean shutdown, replacement
  restart, offline pack, and post-pack restart. All five containers were removed.
- Checkpoint 2C is accepted: run `e9f766ae9854e04f35eb9991d39fb9ba` verified
  loopback HTTP on ports 32768/32769 across replacement, then removed all three
  containers and the per-run network with resolved outcomes.
- Checkpoint 3A implements the persistent development runner/collector workflow,
  restricted to fresh fake ships and the qualified image/HTTP/stop behavior.
  SQLite holds registrations, configuration revisions, idempotency, and operation
  history. Unqualified real-key bootstrap, Ames, meld, and control-socket monitoring
  remain disabled rather than delaying the bounded control-plane milestone. The
  [persistent-runner walkthrough](checkpoints/05-persistent-runner.md) is the next
  manual gate; developer tests use real SQLite and an injected fake runtime.
- The user's 3A snapshot confirms alpha registration, first start, a revision-2
  HTTP-port change, stop, and successful initialized-pier resume. A reported
  two-minute delay was traced to the preflight sleep probe's PID-1 signal handling;
  see the [diagnostic record](diagnostics/2026-09-24-resume-delay.md). Two live
  checks after the fix completed resume in 6.81 s and 6.80 s, with stops in 0.70 s
  and 0.75 s. Stable runner identity and prior history remained after management
  restart; alpha stayed healthy at revision 2 with no incidents. The timing-fix
  gate is complete; other 3A scenarios retain their documented acceptance scope.
- No production deployment is supported yet.

## 1. Contracts and safe simulator

Define host/instance/connection identifiers, protocol negotiation, snapshots,
errors, configuration revisions, idempotent requests, operations, and incidents.
Create a small explicitly simulated runner and collector CLI before managing real
resources. Use a private local socket, no Docker, no key input, and no pier access.

Validation: independent clients see the same snapshots; stale configuration
updates fail; duplicate requests do not execute twice; incompatible operations
conflict; disconnecting a client does not cancel the simulated job; unsupported
protocol versions fail explicitly. Be honest about memory-only demo state.

Manual gate: a documented two-terminal walkthrough using only development data.

## 2. Runtime qualification

Qualify a pinned upstream image on x86-64 with dedicated-account rootless Docker.
Verify direct Vere invocation, permissions, persistence, HTTP/Ames networking,
custom URL/local pill boot, clean shutdown, reboot, offline utilities, and safe
control-socket collection. See runtime-qualification.md.

Validation: record exact artifacts and environment; pass lifecycle and resource
tests on disposable ships. Resolve the known socket-disconnect hazard before
enabling continuous control-socket telemetry.

Manual gate: user-approved setup and disposable-ship qualification. If rootless
fails, explain evidence and agree on an explicit alternative; no silent privilege
changes.

## 3. Minimal real runner and bootstrap

Add explicit instance registration, desired configuration, transactional journal,
start/stop/status, diagnostics, locks, and authenticated access attribution. Boot
from URL by default or imported local pill, with digest verification and temporary
key handling. Reconcile incomplete bootstrap and interrupted jobs on restart.

Validation: two disposable instances; existing/partial pier refusal; download
failure/hash mismatch; idempotent bootstrap; clean stop/restart; journal recovery;
no secret-bearing arguments or logs. Test simultaneous configuration changes.

Manual gate: manage a disposable ship using the CLI, including a runner restart.

## 4. Collector, SSH, and generic ship sessions

Add persistent per-user enrollment, verified SSH connections, inventory,
snapshots/events, polling, and isolation between unavailable connections. Add
origin-bound Eyre authentication/subscriptions and desktop/headless credential
adapters. Keep application access distinct from host management.

Validation: two collectors and two runner hosts; host-key errors; reconnects;
cross-operator conflicts; external ship without a runner; origin isolation;
expired sessions; subscription replay/reconciliation; redacted diagnostics.

Manual gate: laptop CLI manages a remote disposable instance and separately
connects to an unmanaged ship.

## 5. Omabit management interface

Build Omaship collection, enrollment, configuration, operations, health, incident
indicators, bootstrap, and reviewed dependency setup. Add laptop acknowledgement.
Migrate the existing theme integration without moving Omarchy behavior into core
services. Preserve shipped settings and credential behavior through an explicit
migration.

Validation: isolated QML fixtures, desktop/mobile-sized layouts as relevant,
sleep/reconnect, inaccessible hosts, multiple instances, notification behavior,
and non-destructive plugin update/removal.

Manual gate: real Omarchy plugin controls a remote disposable instance.

## 6. Maintenance and epoch retention

Add offline pack/meld with schedules, host resource admission, safe interruption,
fresh before/after measurements, and >60% post-maintenance loom advice. Add epoch
inventory, cutoff preview, confirmation, protected suffix, stopped-pier
revalidation, and qualified prefix pruning. Never substitute blanket chop for a
user-selected cutoff.

Validation: stale/unknown telemetry does not trigger work; bounded retry;
insufficient resources; stale previews; protected epochs; exact deletion set;
restart and claimed recovery paths on disposable multi-epoch piers.

Manual gate: maintenance followed by selected-prefix cleanup and recovery checks.

## 7. Monitoring peers

Add persistent incidents, polling plus streams, observer attribution, restricted
directional peer grants, maintenance windows, deduplication, and recovery events.
Integrate independent relay-ship notifications only through a qualified path.

Validation: process failure, runner failure, partition, sleeping laptop, stale
metrics, and recovery. Do not turn unreachability into a confirmed crash.

Manual gate: two hosts observe each other with one deliberately disconnected.

## 8. Personal operator gateway

Serve a management-only web interface from the collector: inventory, metrics,
configuration, jobs, incidents, and storage cleanup. Do not expose application
transport or credentials. Use authenticated sessions and browser-origin/CSRF
protections. Private reachability is not authentication.

Validation: browser and native collector concurrency; session expiry; unauthorized
requests; CSRF; operation continuity after closing the browser; credential absence
from browser payloads. Reuse configuration revision and operation contracts.

Manual gate: a non-Omarchy browser operates the same runner as an Omarchy client.

## 9. Social utilities and Omarchy gossip

Implement small Contacts, Pals, DM, group-join, and channel connectors with Talon
handoff. Build separate Omarchy gossip with tag enforcement on every delivery
path, bounded tag-revocation delay, limits, deduplication, and retention. Read the
Talon theme schema directly from Settings in the gossip frontend.

Validation: wire fixtures plus disposable multi-ship acceptance; first DM, join
completion, contact-name layers, permitted/blocked forwarding, tag removal,
backlog behavior, and direct Settings theme updates. Tend details remain separate.

Manual gate: multi-ship rumor and theme walkthrough; no claims of cryptographic
anonymity or guaranteed confidentiality.

## 10. Agents and release distribution

Deliver stable CLI output, reviewed top-level skill installation, qualified MCP
integration, prebuilt binaries, service packages, generated microrepos, pill
releases, and upgrade/uninstall/recovery documentation.

Validation: clean x86-64 install without a compiler; digest/version compatibility;
upgrade/uninstall preserves user data; no secrets in agent config, logs, or release
artifacts. Generated repositories identify their source revision.

Manual gate: clean-machine installation using only published artifacts.

## First supported release gates

- Clean install, reboot, and explicit privileges.
- Independent access keys/collectors operating the same runner.
- Configuration conflicts and duplicate-request handling.
- Client and runner interruption during work.
- Verified URL/local custom-pill bootstrap.
- Qualified maintenance and post-maintenance advice.
- Prefix-only epoch deletion and tested recovery claims.
- Laptop sleep/reconnect and truthful partition reporting.
- CLI, Omabit, and gateway use compatible contracts.
- Secret-redacted diagnostics and non-destructive uninstall.

## Research baseline

Source research is not execution evidence and must be refreshed when pinning
release artifacts.

- Omarchy Docker policy: https://github.com/omacom/omarchy/pull/8056
- Pending rootless proposal: https://github.com/omacom/omarchy/pull/11386
- Vere Docker: https://github.com/urbit/vere/tree/develop/docker
- Vere runtime reference: https://docs.urbit.org/user-manual/running/vere
- Urbtop reference: https://github.com/nisfeb/urbtop/tree/7f5ba606ef84e9fdeb11f0a9949936f012313290
- Reported socket crash: https://github.com/urbit/vere/issues/1100
- Proposed socket fix: https://github.com/urbit/vere/pull/1101
- Pals/Rumors: https://github.com/Fang-/suite/tree/b69ce154ae6ce906c2598cd46c806158b51b8a0f
- Talon: https://github.com/nisfeb/talon/tree/8dccb37424e961a542784232b8e50c5097e0bfb8
- Tlon backend: https://github.com/tloncorp/tlon-apps/tree/3f0e5e90e280b67311f7932be8ee7c519f7b0cb8/desk
