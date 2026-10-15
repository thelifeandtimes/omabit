# Architecture

Design established September 22, 2026; host identity/storage updated October 15 for
RC.3. Some sections describe targets; see project status for implemented gates.

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

Lifecycle words refer to the host runtime unless stated otherwise. Urbit's
persisted state is the continuing computer; it is not reinitialized when Vere
exits and is started again. Use these distinctions in interfaces and diagnostics:

- **Bootstrap/initialize:** create a new pier using a pill and identity inputs
  (or fake-ship flags in development). This is a one-time creation operation.
- **Resume/start runtime:** run Vere against the existing pier, load its
  checkpoint, replay any required logged events, and reconnect host I/O. Do not
  pass creation flags or apply a bootstrap pill to an initialized pier.
- **Stop runtime:** gracefully stop the host process and preserve the pier.
- **Restart management:** restart the runner service; healthy Vere processes
  need not stop. Runtime status and service availability are host observations,
  not a reset or conventional guest-OS reboot of Arvo.

Urbit can observe elapsed time and resumed I/O through events. State continuity
does not imply zero host startup work or instant network availability.

Start with a qualified, digest-pinned `tloncorp/vere` image. Invoke the image's
Vere binary with explicit arguments rather than rely on its convenience wrapper,
which does not pass through all bootstrap options. The image binary is the
runtime authority; do not independently auto-update a docked pier binary.

For 0.1.0, the user supplies a versioned Recommended Apps pill. New-ship forms offer
**bare bootstrap** (Vere's own default) and **Recommended Apps** (the configured
release asset). Automated pill building/composition is deferred to 0.2.0.
The Omabit pill is a separately versioned release artifact. The confirmed intended
bundle is `%webterm`, `%groups`, `%base`, `%grubbery`, `%jars`, `%hawk`, `%omart`,
`%pals`, `%obelisk`, `%mcp`, `%trunk`, and `%silo`. The exact publishers
and release gates are recorded in [pill-release.md](pill-release.md) and
[pill-plan.json](pill-plan.json), superseding the earlier draft composition.
`%pals` currently comes from `~tocwex`; the future `~paldev` switch is deferred.
`%omachan`, intended for `~dister-dozzod-sarlev`, is deferred from the initial pill
and does not block 0.1.0. Validate the actual desks/agents and their compatibility
in release tests. Record the bundled revisions for provenance; preserve the normal
publisher-driven OTA paths for ships after they boot and connect.

Release metadata identifies image digest, Vere version, pill version, pill URL,
pill SHA-256, included desks/publishers, and compatibility. A hash proves content
identity, not publisher authenticity; metadata must come from a trusted release
source. Test that pills do not contain publisher secrets or personal state.

Pill provisioning supports both:

- URL (the default): the collector downloads the selected release to a bounded,
  private staging area, verifies its expected release digest, and uploads immutable
  content to the runner's private cache. The release-metadata verification layer
  remains to be implemented; current custom URLs are hashed after download.
  Enforce HTTPS for release provisioning and controlled redirects; treat configurable sources
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

Initial support: x86-64 Linux, systemd, Docker. The default installation is an
existing-owner-user rootless deployment. Qualification must validate this path;
there is no silent fallback to a privileged engine. Do not enable Sudoless Docker,
grant the desktop account access to the rootful socket, disable existing system
Docker services, or change the desktop's Docker context.

The target onboarding remains: enable Omabit, choose local hosting, review the
installation and permissions, authorize through the OS, then set up a ship.
Rootless complexity belongs in our setup tooling rather than manual user
configuration. Initial development checkpoints may require terminal steps before
the plugin installer exists.

Use the selected existing pier-owning Unix account, preserving its login shell,
groups and personal SSH settings. Use non-overlapping subordinate UID/GID ranges
and lingering for unattended systemd user services. Keep the package-managed Docker engine and
helpers. Supply the missing upstream rootless launcher at a reviewed version and
digest; do not pipe a remote script into a shell or install a second static
Docker engine. Keep launcher code root-owned outside the plugin checkout.

Use a separate user service and explicit owner-private Docker socket. Remote clients
receive only the runner API. Same-owner local processes share Unix authority;
this model does not isolate the desktop user from their own engine/piers.
Prove rootless mode and cgroup limits before using the engine for ship operations.

Current owner-user locations:

```text
/srv/urbit/piers/{fakeships,livenet/<tier>}/[group/]<ship-directory>/pier/
~/.local/share/urbit-runner/managed/runner.sqlite
~/.local/share/urbit-runner/managed/<private-id>/
/run/user/UID/urbit-runner/runner.sock
```

The exclusive ship parent mounts at `/urbit`; Vere always uses `/urbit/pier`.
First boot creates that child in place and keeps the same container running.
The SQLite journal is transactional; a simulator may be memory-only if clearly marked.
Keep collector configuration/state separate. Plugin checkout directories never
contain piers, credentials, or authoritative runner state. Uninstall does not
delete piers.

Laptops are supported. Require acknowledgement that sleep/shutdown makes ships
unavailable, allow reminders, and never silently alter suspend/lid behavior.
Optional sleep inhibition for an explicitly requested maintenance job is visible.

## Maintenance and retention

Use official released runtimes for offline pack/meld/roll/pruning jobs and explicit
UTC maintenance schedules. Measure container RAM and logical/allocated disk sizes
with source and freshness; unknown values are not zeroes. Vere 4.6 meld already
includes compaction.

Live loom-internal telemetry, utilization percentages and pressure-based advice are
deferred. Keep the control socket disabled and avoid custom runtime builds. A larger
runtime feature set is not a prerequisite for reliable ship operation. Future
multi-runtime support, if needed, should be a simple `live` (upstream release) /
`edge` (upstream develop) choice; see [runtime-policy.md](runtime-policy.md).

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

The `/omabit` monorepo is authoritative and directly installable as the Omarchy
plugin, with a root manifest and native wrappers over shared QML. Generic runner
and collector entry points live under `backend/cmd/`; the desktop Go helper lives
under `cmd/omabit/`. Shared private Go packages stay in one module. Release assets
carry source revision and compatibility metadata; native launchers pin their exact
binary contents. Do not hand-maintain copies of shared UI or fork the generic tools
merely to distribute them to non-Omarchy users.

Ship prebuilt binaries, service packages, pill metadata, explicit setup steps,
and a reviewed CLI skill installation. Agent configuration does not contain raw
Eyre cookies. Qualify the actual MCP contract before integrating it.

Out of scope initially: clearweb ship exposure, planet acquisition, identity
ownership-key management, malicious same-user isolation, automatic cross-host
failover, Tend application design, ARM64, and arbitrary NAS platforms.
