# Architecture

Status: agreed design, September 22, 2026. This describes the target, not features
already implemented. See the implementation plan for progress and gates.

## Roles and boundaries

```text
Omabit plugin ------> urbit-collector <------ urbit-operator-gateway
                            |                  personal, management only
                            |
                            +-- SSH / local socket --> urbit-runner
                            |                              +-- Docker + piers
                            |
                            +-- Eyre --> ship applications

%omarchy-gossip frontend --> %settings (Talon theme schema)
```

`urbit-runner` is a host service independent of Omarchy. It owns container
lifecycle, desired configuration, bootstrap, durable operations, resource
admission, telemetry, incidents, epoch retention, and monitoring peers. It must
continue working without collectors or a logged-in desktop.

`urbit-collector` is a per-user client service independent of Omarchy. It owns
explicitly enrolled runner connections, combined inventory and observations,
generic Eyre authentication, origin-bound credential storage, and subscription
transport. CLI and interface clients use the same contracts. One unavailable
host or ship must not stall unrelated connections.

`urbit-operator-gateway` is an optional personal web UI served by a collector. It
exposes only runner-provided management, metrics, and tooling. It does not expose
ship application APIs, proxy Eyre, or deliver SSH/Eyre credentials to browser
code. It is not a shared-user service or required central coordinator. Private
deployments still need authentication, origin/CSRF protections, and secure
transport.

The Omabit plugin owns all Omarchy-specific behavior: the Omaship experience,
dependency explanations, notifications, theme publishing, social panels, and
agent setup. Generic binaries must not depend on Quickshell, Omarchy directories,
or desktop theme files. Tend is independently enableable and depends on a ship
connection, not on a locally managed Docker ship.

Use Go for core services/CLI, QML for Omarchy interfaces, and Hoon for ship-side
functionality. Share libraries without conflating process privileges.

## Collection model

- Host: a machine running a runner, identified independently of its address.
- Managed instance: a pier/container owned by a particular runner, with its own
  stable ID and verified ship identity.
- Ship connection: authenticated application access, including externally hosted
  ships added by URL and login code without runner management.

Discover inventory only through explicitly enrolled runners. Do not scan for or
adopt arbitrary containers. Authentication for application access is separate
from inventory discovery. An Eyre timeout establishes unreachability, not a
confirmed process crash.

A ship has one active home. No automatic cross-host failover, pier replication,
or unattended restore/start. Per-host locks cannot prevent an administrator from
starting a copy elsewhere; document this limitation.

## Transport and access

Use a permission-restricted Unix socket locally and structured RPC over a fixed
SSH command remotely. Verify SSH host keys and use normal SSH configuration. Do
not expose a public runner management port or accept arbitrary shell commands,
mount paths, Docker arguments, or runtime arguments through the API.

Several independent collectors, potentially belonging to different operators,
can manage the same runner. Attribute actions to authenticated access keys. SSH
fingerprints must come from trusted authentication context, not client-supplied
JSON. Local access records the operating-system identity. Support individually
revocable credentials and restricted observation access.

Full host/Dojo operators can obtain the ship's login code and access its data.
We do not claim to isolate the ship from its administrator. Operation history is
useful attribution, not tamper-proof evidence against a host administrator.

The initial threat model covers accidental disclosure and agent mistakes, not
malicious same-user desktop code. Omarchy plugins are unsandboxed. Secret Service
reduces accidental exposure but is not a same-user isolation boundary. Desktop
and headless credential stores need separate adapters. Owner Eyre sessions are
not application-scoped; future ship-enforced capabilities are a separate feature.

## Operations and configuration

Use a durable operation journal with request identity, authenticated actor,
instance, kind, progress, outcome, timestamps, and bounded/redacted diagnostics.
Disconnecting a client must not cancel or duplicate work. Retried requests with
the same idempotency key and content return the same operation; reuse for a
different request is an error.

Serialize incompatible operations per instance. Admit expensive jobs against
host-wide RAM/disk budgets. Configuration writes require an expected revision;
stale writes return a conflict rather than silently overwrite another operator.
Reconcile interrupted operations after restart; never blindly repeat a
destructive action.

Keep desired state, observed process state, health, current operation, and
configuration drift separate. Systemd supervises the runner; the runner is the
single ship-lifecycle authority. Avoid competing automatic restart policies.

## Runtime and bootstrap releases

Start with a qualified, digest-pinned `tloncorp/vere` image. Invoke the image's
Vere binary with explicit arguments rather than rely on its convenience wrapper,
which does not pass through all bootstrap options. The image binary is the
runtime authority; do not independently auto-update a docked pier binary.

The Omabit pill is a separately versioned release artifact. Include the agreed
opinionated desk bundle: base, webterm, groups, pals, hawk, trunk, omabit,
omarchy-gossip, jars, and mcp, with auspex, orrery, and lattice resolved as part of
pill composition. Tend remains optional as a user experience. Validate the actual
desks/agents and their compatibility in release tests.

Release metadata identifies image digest, Vere version, pill version, pill URL,
pill SHA-256, included desks/publishers, and compatibility. A hash proves content
identity, not publisher authenticity; metadata must come from a trusted release
source. Test that pills do not contain publisher secrets or personal state.

Pill provisioning supports both:

- URL (the default): the runner downloads the selected release to a bounded,
  private staging area, verifies its expected digest, and caches immutable
  content. Enforce HTTPS and controlled redirects; treat configurable sources
  as privileged input and prevent unintended access to local services. Never
  forward ship/SSH credentials to the source.
- Local file: explicitly import a pill into runner-controlled storage and verify
  its digest. A desktop path is not implicitly a path on a remote runner.

Vere supports URL bootstrap with `-u URL` and file bootstrap with `-B FILE`.
Prefer resolving URL input to verified content and using `-B` for reproducible
boots. The user still provisions by URL; no manual download is required. A direct
`-u` execution path, if introduced, must not bypass the release integrity policy.
Never silently fall back to Vere's default pill when a selected download fails.

Boot keys use temporary restricted files/streams, not arguments, environment,
logs, or image layers. Existing piers do not require the original boot file.
Piers and backups remain secret-bearing even after bootstrap-file deletion.

Changing the published pill changes new boots only. Existing ships have separate
kernel/desk update policies. Runtime upgrades can migrate pier formats; a prior
container image is not necessarily a valid rollback. Require release-specific
preparation and consistent backups.

## Host deployment and storage

Initial support: x86-64 Linux, systemd, Docker. Prefer a dedicated-account
rootless deployment after qualification. Do not enable Sudoless Docker or grant
the desktop account access to the rootful socket. If required, a privileged
runner installation is explicit, with its root-equivalent authority explained.

Example dedicated-account locations:

```text
$XDG_CONFIG_HOME/urbit-runner/config.toml
$XDG_DATA_HOME/urbit/piers/<instance-id>/
$XDG_DATA_HOME/urbit-runner/artifacts/
$XDG_STATE_HOME/urbit-runner/runner.sqlite
$XDG_RUNTIME_DIR/urbit-runner/runner.sock
```

Use equivalent service directories for system installations. The production
journal will be transactional; a simulator may be memory-only if clearly marked.
Keep collector configuration/state separate. Plugin checkout directories never
contain piers, credentials, or authoritative runner state. Uninstall does not
delete piers.

Laptops are supported. Require acknowledgement that sleep/shutdown makes ships
unavailable, allow reminders, and never silently alter suspend/lid behavior.
Optional sleep inhibition for an explicitly requested maintenance job is visible.

## Maintenance and retention

Begin with offline pack/meld/roll/pruning jobs. Measure loom capacity, contiguous
open space, allocator free-list space, process/container RAM, and disk separately.
Each metric carries freshness and source; unknown values are not zeroes. Deep
mass profiling is expensive and is not a frequent heartbeat.

Use sustained pressure and hysteresis for scheduling. After pack/meld, restart,
take a fresh validated measurement, and suggest a larger loom if the defined
utilization remains above 60%. Explain the metric and required resources. Do not
promise compaction fixes large live state, automatically resize, or repeatedly
compact without improvement. Vere 4.6 meld already includes compaction.

Roll is off by default. There is no automatic chop timer. The "Free up space"
action previews epoch inventory, protected entries, estimated allocated space,
and lost recovery history. Selecting an epoch selects it and every older epoch:
only an oldest contiguous prefix can be deleted. Initially protect at least the
newest two, plus any additional runtime/recovery dependencies.

Lock and stop the pier, then revalidate the approved cutoff and inventory. Stock
chop cannot implement arbitrary cutoffs; qualify a runtime-supported pruning
operation or equivalently tested implementation. Never hide directories to trick
chop or delete unvalidated paths. A retained suffix is not proof of backup
integrity; test the recovery paths the UI claims remain available.

## Monitoring

Runners own persistent incidents and observations; collectors combine streams,
snapshots, and polling. Persistent UI indicators supplement transient desktop
notifications. Acknowledgement is separate from resolution. Respect DND and
deduplicate repeated failures.

Monitoring peers are explicit, directional, read-only grants. Preserve observer,
endpoint, freshness, evidence, and maintenance windows. Unreachable does not mean
confirmed dead. Avoid recursive incident forwarding. A healthy independent
observer plus a healthy relay ship is required for immediate Ames outage alerts.

## Social applications and themes

Keep upstream Pals. Tags are local and not exchanged by its relationship
protocol. Omarchy gossip is a separate application from Rumors. Gate incoming
subscriptions, direct messages, backlog, proxy selection, and every outgoing hop
on the local outgoing-pal `omarchy` tag. Cooperating peers each enforce their own
selection. Reconcile tag-only changes and revoke eligibility with bounded delay.
Include retention, rate limits, payload limits, and duplicate suppression.

Anonymous means no author attribution in the rumor UI/payload, not cryptographic
anonymity. A recipient can copy content elsewhere; this is routing policy, not a
confidentiality guarantee.

Use Contacts for profiles/private nicknames, Chat for DMs, Groups for membership,
Channels for group-channel content, and Settings for preferences. Pals, contacts,
and group membership remain distinct. Follow Talon client patterns but validate
wire formats against the backend's codecs.

Omabit publishes the designated device's Omarchy palette to Talon's existing
`talon/ui-prefs/themes` settings entry. The gossip frontend reads/subscribes to
Settings directly and follows that schema. No Omabit browser round trip or new
competing palette schema. Validate colors and supply a fallback; do not sync
arbitrary CSS. Use one designated automatic theme publisher per ship.

## Distribution and exclusions

The monorepo is authoritative. Generate installable microrepos/artifacts with
source revision and compatible API versions. Omarchy plugin repos need their own
root manifest. Do not hand-maintain copies of shared code.

Ship prebuilt binaries, service packages, pill metadata, explicit setup steps,
and a reviewed CLI skill installation. Agent configuration does not contain raw
Eyre cookies. Qualify the actual MCP contract before integrating it.

Out of scope initially: clearweb ship exposure, planet acquisition, identity
ownership-key management, malicious same-user isolation, automatic cross-host
failover, Tend application design, ARM64, and arbitrary NAS platforms.
