# Omabit platform roadmap

Recorded 2026-10-21. Product direction from the user; the experiments below are
proposed delivery steps, not implemented contracts or live-operation approvals.
Source baseline: Omabit v0.2.0, commit `b233128`.

## Product anchor and repository ownership

**“Multiplayer apps without a subscription.”** This is the user's anchor phrase,
attributed by the user to DHH. Make the everyday experience demonstrate it:
connect your identity once, open an application, and collaborate using state on
your own ship. Hosting, hardware and optional services still have costs.

The three pillars are:

1. Run ships reliably and let independent operators manage a shared fleet.
2. Discover, install, use and develop Omarchy interfaces for Urbit applications,
   including links to web/PWA and native clients.
3. Approve an application's connection to an already authenticated ship instead
   of entering its URL and `+code` again for each plugin.

`thelifeandtimes/omabit` is the runner/collector/Omarchy integration repository.
`thelifeandtimes/tend` is independent. The retired Omabit/Tend monorepo, its
`apps/urbit-host` pilot and its layout-migration plan are historical context, not
the codebase to extend. Theme, DMs and Omachan likewise remain independently
installable. This roadmap does not require creating a new messenger client.

Theme and DMs are the first shared-authentication integration pair. Tend remains
the full multiplayer application proof. Talon and Lattice are reference clients
and discovery targets; learn from their patterns without assuming their security
or licensing choices can be copied wholesale.

## What already exists

| Area | Evidence in the current source | Remaining distinction |
| --- | --- | --- |
| Fleet management | Runner identities, operator/observer grants, revisions and durable operations; several collectors may enroll the same runner | Concurrent operators must not overwrite each other's changes or share credentials implicitly |
| Imports | `internal/managed/import_preview.go` derives the identity-based child; `docs/pier-storage.md` defines the exclusive parent mount | The screenshot reports a combined tree rejection; its actual cause is still unknown |
| Ship authentication | `docs/ship-sessions.md`; collector enrollment, login, status, scry, watch, logout and Secret Service/file adapters | Owner cookies are not app capabilities; generic application mutation transport is not supplied by that documented checkpoint |
| Hosted ships | Collector ship enrollment is independent of runner/SSH enrollment | Desktop onboarding and application approval still need a coherent flow |
| Desk inventory | Documented successful `hood /kiln/pikes` scry and `/kiln/updates` watch | Inventory needs normalization, freshness and separate Grubbery app/permit probes |
| Fake networking | Current storage docs describe container-local Ames; project directories exist | A shared directory label does not establish a communicating project network |
| Ethereum testnet | Storage docs explicitly call it a stub | Network identity, chain configuration and boot/runtime behavior need an explicit supported contract |
| Application Connections | README identifies the UI as Coming Soon | Target: separate ordinary Eyre sessions handed to plugins, with direct plugin-to-ship data flow |

## Execution discipline

One bounded change and its acceptance evidence at a time. Use a topic branch and
isolated worktree for active implementation; keep installed plugin files and
running services separate. Backend changes enter a new minor/major release line
(the next minor after 0.2.0 would be 0.3.0); this document does not publish one.
Preserve the [development policy](development-policy.md) and
[upgrade contract](upgrade-policy.md).

Completion of this roadmap is the overall goal. The user resumed implementation
on 2026-10-27 after manual VM review. The next pass follows
[the review feedback](review-pass-2026-10-27.md); the 40 GiB retention cap remains
in force. The earlier pause is recorded in [the development review](development-review-2026-10-26.md).
Future experiments must follow
[the user's isolation constraints](experiment-isolation.md): preserve managed
`~lismyl-lismyl`, exclude tmux ships and `~/temp-ur`, and use separate experimental
plugin identities, desktop environments, state and credentials.

Each step ends with: the behavior demonstrated, exact source/artifact identity,
automated versus live evidence, limitations, and the next small experiment.
Read-only inspection, package installation, desk installation, application access
and starting a real identity are distinct operations. Approval must describe the
actual changes. No silent retry of an uncertain mutation.

The user has withdrawn the reported import source from testing: it is now running
as a standalone Urbit in `urbit-prod` on sourcebox. Do not inspect, stop, copy,
move, import or otherwise operate on that ship for this work. Use synthetic
fixtures and later explicitly designated disposable ships. Manual testing of the
reported source waits for renewed instructions and confirmation it is stopped.

## Step 1 — import reliability

First slice: replace the ambiguous tree rejection with a quoted relative entry
and the failed check. Preserve ownership, private state, link, mount and file-type
restrictions. Do not suggest recursive chmod/chown or lock deletion as a fix.
See [the diagnostic change](diagnostics/2026-10-21-import-diagnostics.md).
The [partial-copy preservation run](diagnostics/2026-10-22-import-interruption.md)
and [runner/CLI restoration checkpoint](diagnostics/2026-10-22-import-restore-integration.md)
record the earlier interruption and source integration checkpoints.
The [real restoration/re-import run](diagnostics/2026-10-22-import-restore-runtime.md)
now passes. [Guided restoration UI](diagnostics/2026-10-22-import-restore-ui.md)
also passes actual offscreen QML with inert dispatch. The [UI-to-VM trial](diagnostics/2026-10-22-import-restore-ui-runtime.md)
now passes actual collector/runner dispatch, preserved trees, retained receipts,
replay/re-import and explicit Start/Stop. Other failure modes and native desktop
integration remain open.

Next slices, separately:

- Verify a synthetic import retains `<instance-name>/<patp-without-tilde>` and
  mounts only the instance parent. Cover arbitrary source names, collisions,
  an existing legacy `pier` child, custom roots, and a comet with known identity.
- Distinguish newly mined comets (identity initially unknown) from imported
  comets. Decide a stopped, identity-verified rename policy before requiring a
  new comet's provisional `pier` child to become its name; never rename a live
  registered pier as a cosmetic fix.
- Exercise same-filesystem rename and cross-filesystem copy with interruptions,
  full disks, replaced sources, restart/retry and retained enrollment recovery.
- Run the guided UI on a disposable stopped pier. Check preview, progress,
  failure recovery, explicit Start and the resumed identity. Only then schedule
  review of a valuable pier under [protected ships](protected-ships.md).

Exit: actionable refusal, preserved source on failed preview, the required mount
layout, and demonstrated recovery without duplicate starts or lost pier data.

## Step 2 — CLI and operational skills

Inventory the actual commands first. Ship a concise operator `SKILL.md` with
links to task-specific references, separately from contributor instructions.
Agent entry points should progressively disclose:

1. Short help: what can be observed and the available command families.
2. Read-only status/capability discovery: version, runner/ship identity, authority,
   freshness and supported operations.
3. Command help/examples: required inputs, effect, prerequisites and secret input
   handling. Never invent a command in a skill that the shipped CLI lacks.
4. Machine-readable errors and next actions: unsupported capability, expired
   session, revision conflict, uncertain operation, and the operation receipt.
5. Detailed troubleshooting only on request, with redacted output.

Acceptance tasks: identify the intended runner, distinguish unreachable from
stopped, inspect an import rejection, monitor an asynchronous operation, and
recover from a stale revision. Exercise both readable output and structured output
using the built CLI, with simple scripted callers and human walkthroughs. Skills
describe authority limits; they are not enforcement against an unrestricted agent.

## Step 3 — project fake networks

Manual review correction (2026-10-28, M464): fake creation belongs in Add a ship,
with None / New Project + / existing-project selection. Bulk creation stays in
the CLI. M466–M467 supersede the initial join limitation: an append-only member
ledger now adds a fake to an existing namespace with its own loopback forwarding
receipt. The original roster/anchor contract remains unchanged. Expanded anchor
and whole-engine recovery remain separate unfinished gates.

Design a project identifier independent of filesystem labels, ship `@p`, and
instance IDs. Start with one runner, two disposable fake galaxies and one project.
Inspect the pinned Vere/Ames behavior before selecting bridge versus shared
namespace transport. Check administrative loopback exposure as well as Ames. The
[pinned-runtime source notes](project-fake-networks.md) record why a bridge alone
is insufficient and define the first private namespace experiment.

Acceptance ladder:

1. Bidirectional authenticated application messages between two fake ships.
2. Restart one ship and recover subscriptions/state without recreating its pier.
3. Run the same fake identities in a second project and prove no cross-project
   traffic, endpoint/cookie confusion, or administrative access.
4. Prove fakes cannot accidentally participate in the real network.
5. Add a two-runner project only after same-host behavior and trust boundaries
   are demonstrated. SSH fleet access alone is not a ship-network transport.

Checkpoint 2026-10-22: [the private Docker transport test passes](diagnostics/2026-10-22-docker-peer-recovery.md).
Rung 1 is demonstrated. Clean process/container replacement also preserves the
pier, namespace and original Eyre sessions, with messages working afterward.
Rung 2 remains open for persistent application state and subscriptions/client
reconnect; rung 3 and runner integration are still pending. No production rollout
or complete networking-feature qualification is implied.

Subsequent checkpoint: [Docker state/subscription recovery](diagnostics/2026-10-22-application-state-docker.md)
passes rung 2, and [two simultaneous duplicate-identity projects](diagnostics/2026-10-22-docker-project-isolation.md)
pass the isolated state/session portion of rung 3. These later records supersede
the pending status above. Published endpoint/administrative exposure and anchor
lifecycle still require their separate fixture before runner integration.

Record the trust of all ships sharing a namespace. Do not claim a shared
development namespace isolates hostile applications from one another.

[The ordinary-bridge endpoint fixture passed](diagnostics/2026-10-23-endpoint-publication.md)
with exact loopback mappings and coordinated recovery after namespace replacement.
This closes the HTTP-fixture publication/lifecycle experiment, not the actual
Urbit publication or durable runner project gates. The
[project contract draft](project-network-contract.md) records the remaining ownership,
reconciliation and upgrade requirements.

## Step 4 — shared authentication, direct application connections

Omabit assists with authentication and connection management. Each plugin then
talks **directly to Eyre** using its own session and transport. Omabit/collector
must not become the data provider, proxy or permanent dependency for plugin
scries, pokes, watches or reconnects. A plugin remains fully usable without
Omabit, including on a hosted Tlon ship: URL + `+code` login is a first-class
standalone path, not an emergency mode requiring Omabit binaries.

Use **Theme and DMs** as the first pair. They use agents in the standard `%groups`
stack (Theme: `%settings`; DMs: `%chat`, `%contacts`, `%activity`). Probe actual
backend readiness; bare or customized ships may differ. Tend is not part of the
first shared-authentication checkpoint.

The assisted flow is: select a saved ship, see the requesting plugin and profile,
approve, create a **new** ordinary Eyre session, and hand it privately to that
plugin's credential adapter. Separate sessions are required even when two plugins
use the same desk. Do not copy the collector's cookie to all plugins. Retain the
existing standalone login and plugin-owned storage/transport lifecycle.

Current Eyre sessions are not restricted to desks or paths. The first release
does not add inter-desk permission enforcement or an authorization proxy. Separate
sessions provide independent lifecycle/revocation, not capability isolation.
The user's 407k / Gall 2026 work is a future integration point, not an initial
dependency. Lattice's scoped keys remain useful research rather than a prerequisite.

Specify only the boundaries needed for this handoff:

- Binding to plugin ID, profile, connection ID, canonical origin and verified ship
  identity; repeated fake `~zod` identities must remain distinct connections.
- Explicit approval, private delivery, durable acceptance/cleanup, expiry and
  cancellation. Cookies and transient codes stay out of QML, argv, URLs and logs.
- Plugin-owned requests and reconnects after the auth helper exits. No automatic
  fallback to a collector proxy when direct data access fails.
- Logout of one exact plugin session; distinguish local removal, confirmed remote
  revocation and offline cleanup pending. Do not log out all ship sessions.
- Profile changes stop old watchers before activating another account; late
  responses cannot overwrite the new profile's state or credentials.

The next checkpoint is a small authentication experiment: one saved login creates
two distinct sessions; both clients read directly after the authentication client
closes; revoking one leaves the other and the original login usable. Test malformed
responses, wrong identity/origin, missing thread support, and session reuse. This
starts with synthetic HTTP fixtures; real ship compatibility requires a separately
authorized disposable target. See [the checkpoint](application-auth-checkpoint.md).

Then wire the handoff into Theme and DMs, testing assisted and standalone login,
process restart, keyring lock, session expiry, profile switching and operation
with Omabit stopped or absent. Extract shared client code only after both work.
Runner management permission never implicitly enrolls an application session or
shares one operator's session with every fleet operator.

## Step 5 — plugin-owned setup and compatibility, with Omart discovery

Omabit is **not an app store**. Its scope is connected-ship/software observations,
connection status and useful links. Do not build a catalog browsing, publishing,
ranking or marketplace system into it. If work starts to drift toward that scope,
stop and rethink whether it belongs in **Omart**, the existing alpha Urbit-based
Omarchy plugin marketplace. Consult `ahlmark/omart-desk` before designing
discovery infrastructure; do not create a parallel catalog as incidental work.

The default developer pattern belongs inside every standalone plugin:

1. Authenticate, through Omabit or directly using URL + `+code`.
2. Probe the required Clay desk/Gall agents or Grubbery app and permissions.
3. If absent, show an **Install backend** button naming the target ship, exact
   desk/app and publisher. Use the supported install mechanism after approval.
4. Observe installation, boot/readiness and Grubbery permits separately; show
   progress, refusal, retry or an incompatibility explanation.
5. Open the application using its own direct transport once compatible and ready.

This must work with no Omabit installation. A ship being unreachable is not proof
that its backend is missing. Installation cannot silently change an existing
publisher or grant unrelated permissions. Use Tend as a later missing-backend
demonstration; Omart already has a useful missing-`%pals` installation pattern.

Keep dependencies/source links and OS authorization visible when installing
desktop software. Shared conventions should describe backend identity/publisher,
readiness probes, protocol versions, supported client/backend combinations,
immutable artifact identity and update channels. Start with plugin-local metadata;
do not turn a compatibility descriptor into an Omabit catalog schema.

Track partial installation and compatibility explicitly. A desktop install and
ship update are not one atomic transaction. Test both update orders, old/new
client/backend pairs, offline/restart halfway through installation and backend
permission refusal. Preserve a last-known compatible desktop artifact without
promising to reverse ship migrations. Desk OTA must not silently execute new
desktop code outside the selected desktop update/consent policy.

Distribution experiments remain separate: native GitHub installation first,
optional desk-served artifacts second, optional Urgit Git remotes later.
[Urgit documents Git hosting and peer collaboration](https://github.com/yapishu/urgit);
it is not a mandatory dependency. Verify installer support and source provenance
before advertising ship-URL installation commands. Third-party plugin authors must
be able to maintain compatibility metadata independently of backend publishers.

## Step 6 — private Ethereum devnets and developer profiles

The initial happy path is a **private local Ethereum development chain**. Public
Goerli-style testnets are outside initial scope. Pin the chain configuration,
chain ID, Azimuth deployment/contracts, local RPC endpoint, disposable identities,
bootstrap pill and supported Vere options. An alternate RPC URL alone is not a
complete development network; qualify the pinned kernel/runtime and chain together.

Start with two disposable identities on one runner. Prove messaging, restart,
project isolation, wrong-chain rejection and absence of livenet key/port reuse.
Keep fake, private-chain and livenet classifications distinct in storage,
connection selection, UI and operation receipts. Multi-host work follows later.

The developer kit must also define **stable/dev/edge plugin collections**:

- A named profile selects a coherent set of plugin versions/channels and ship
  connections. Distinguish the selected collection from versions actually loaded.
- Keep state, caches, session keys and selected endpoints separate per profile.
  Never silently point a dev plugin at a production ship or copy owner sessions
  across profiles. Profile creation does not create/reset a pier or chain.
- Switch deliberately: stop the old helpers/watchers, activate the collection,
  verify readiness and recover the last working desktop collection on failure.
  Old in-flight results must not mutate the new profile. Returning to stable
  must not imply downgrading a runner database or reversing ship state.
- Test stable → dev → stable and stable → edge → stable with different fake
  endpoints, concurrent stale completions and independent authentication.
- Reuse lessons from [Omabit's existing desktop profiles](desktop-development.md),
  while proving collection-wide behavior against the actual Omarchy loader.
  Existing Omabit-only profiles do not yet establish whole-plugin-set switching.

Provide developer skills and a small working sample covering native design tokens,
keyboard/focus/accessibility, loading/stale/error states, direct Eyre transports,
assisted and standalone authentication, plugin-owned backend installation from a
named publisher, compatibility/reconnect/idempotency, and fake/private-chain
development commands. Include the profile workflow in the sample and skills.
Consult the current Omarchy plugin API and test on two independent ships.

## Core v1.0 release gates

- Import/layout and recovery acceptance, including an explicitly authorized
  manual UI pass; no duplicate real-identity starts.
- Understandable CLI discovery/errors and verified operational/developer skills.
- Demonstrated project fake networking and separately qualified private Ethereum devnet
  networking, with isolation and restart evidence.
- Independent multi-collector operation, stale revision handling, revocation and
  compatibility across the supported runner/collector/UI upgrade window.
- Theme/DMs separate-session handoff, direct data traffic with Omabit stopped,
  standalone hosted-ship login and independent logout. Desk isolation is deferred.
- Plugin-owned missing-backend installation and compatibility/update demonstration,
  with marketplace discovery kept in Omart.
- Stable/dev/edge collection switching with isolated credentials and state.
- Tests on the supported rootless Omarchy host; non-destructive upgrades,
  interruption recovery, diagnostics and documented limitations.

Wallet delivery, a universal app store, automatic failover, and adopting Urgit on
every ship are outside these gates. Retain wallet/key-storage research as part of
the connection architecture so future choices are informed rather than accidental.

## Reference snapshots consulted

These are local source observations, not independent release qualification:

- Omabit `b233128`: `docs/architecture.md`, `docs/ship-sessions.md`, storage docs,
  `internal/managed/import_preview.go`, `internal/pierstore/external_linux.go`.
- Tend `bd99c33`: `docs/TEND_SECURITY.md`, `docs/TEND_COMPATIBILITY.md` and
  `docs/TEND_REPOSITORY_MIGRATION.md` (independent repository and protocol gates).
- Talon `7cbaa15`: `core/.../urbit/UrbitSession.kt`, `UrbitChannel.kt`,
  `composeApp/.../DesktopSessionStore.kt`, `docs/grubbery-apps.md` (session/channel
  ownership, backend availability and installation UX).
- Lattice `5d90b16`: `code/lib/lattice-keys.hoon`, `code/nex/lattice/app.hoon`,
  `docs/releasing.md` (route scopes, revocation and separate update mechanisms).
- Theme `1719e9f`: `client/support.py`, `client/eyre.py` and README (Secret Service,
  multi-ship settings integration and explicit authentication-required state).
- DMs `cf236fd`: `session_store.py`, `bridge.py`, `browser_session.py` (origin-bound
  keyring sessions and private browser handoff).
- Omachan: uncommitted local source, `plugin/bridge.py` and README (memory-only
  session and staged backend onboarding); no commit-backed baseline yet.

## Immediate checkpoint after the 2026-10-21 corrections

Step 4's independent-session primitive and fixture tests are now implemented as
an internal experiment, before a shared SDK or approval UI. See the
[checkpoint evidence and next gate](application-auth-checkpoint.md). The source import diagnostic patch remains retained and tested;
its live acceptance waits for a designated disposable target. CLI, networking and
profile work remain separate checkpoints, not prerequisites for this auth experiment.

## Latest qualified runtime follow-up — 2026-10-24

[VM42](diagnostics/2026-10-24-service-restart42-runtime.md) independently passes
public recovery after lost replies plus three actual owner-service restart cycles,
with unchanged project containers/checkpoints and application/subscription state.
The fake-network runner recovery checkpoint is now qualified in a disposable VM;
public UI and release installation remain. Private keyed native peers also pass
reproducible state/subscription recovery (M239); Docker devnet integration remains.
Theme/DMs empty-account collection teardown passes (M234); connected/busy lifecycle
and durable activation are next. These records supersede earlier pending statements
only for their explicitly demonstrated scopes.


Collection follow-up M285: the private supported Omabit/Theme/DMs set now passes
persistent switch/rollback, supervised shell loss, explicit cleanup recovery and
return with retained credentials/application intent and restored subscriptions.
This is [qualified private evidence](diagnostics/2026-10-24-recovery-resume.md), not
stock Omarchy loader support or a user-facing profile command. Native identity/
admission and external appearance effects remain the profile integration gate.

M286 now supplies immutable artifact preparation and native registry/IPC identity
proof with synthetic plugin bodies. Selected versus loaded versions are distinct;
62 private tests and five profile roundtrips pass. This narrows the preceding native
identity gap. Real supported wrappers/bar widgets, durable admission and external
effects still gate a user-facing collection command. See
[artifact qualification](diagnostics/2026-10-24-native-artifacts.md).

M288 qualifies a sealed one-use shell/member permit and inert host with78 private
tests and five native loader cases. Missing admission loads zero apps; killed guards
and replacement shells cannot reuse old permits. Synthetic bodies only; immutable
host packaging, actual widgets/apps and coordinator recovery remain. Closing claims
does not prove in-flight constructor or transport closure. See
[native admission](diagnostics/2026-10-24-native-admission.md).

M289 packages primary/widget hosts and helpers in pinned format2 generations.
All88 private tests and five native-primary-loader cases pass with synthetic bodies;
original format1 reproduction/readability and old-reader refusal pass. Actual
supported bodies, native bar behavior and coordinator recovery remain. See
[packaged admission](diagnostics/2026-10-24-guarded-artifacts.md).

M290 loads actual Omabit Service/Panel through immutable guarded native hosts.
Stable/dev constructor identity, real service-payload access, scoped IPC and
transport drain pass. Theme/DMs remain synthetic; actual pair/widgets and connected
coordinator recovery are next. See [actual Omabit](diagnostics/2026-10-24-guarded-omabit.md).

M291 qualifies actual Omabit/Theme/DMs bodies and widgets in guarded generations
with empty private profiles. Five native admission cases and actual transport drain
pass; unchanged Python helpers and portable sources are verified. Connected recovery
and closing-time UI/IPC still need integration. See
[actual supported set](diagnostics/2026-10-24-guarded-supported-set.md).

M292 withdraws actual widget/setup IPC and refuses monitor/overlay reopening as
admission closes, including while an Omabit operation is still finishing. Negative
control and actual supported-set regression pass; all88 private tests pass.
Connected native recovery remains. See
[closing UI admission](diagnostics/2026-10-24-closing-ui-admission.md).

M293 qualifies connected stable/dev/stable through guarded native generations:
fifteen positive subscriptions, unchanged sessions, and byte-identical unresolved DM
without replay. Ten channels close. This narrows M292's connected-account gap;
native coordinator/unclean recovery remains. See
[connected native state](diagnostics/2026-10-24-native-connected.md).

M294 integrates original-guardian recovery with guarded native generations and the
durable coordinator. Stale startup stays inert; failed cleanup and three lost-reply
boundaries preserve state and safely resume dev/stable with15 subscriptions.
Cross-root ownership, full bar/plugin API/compositor and public commands remain.
See [native recovery](diagnostics/2026-10-24-native-recovery.md).

M295 fixes a demonstrated native bar ownership mismatch around guarded widgets.
Actual navigation/popout checks, closed-admission refusal,88 private tests and the
connected recovery regression pass. Full bar/compositor and native shell facades
remain separate. See [bar ownership](diagnostics/2026-10-24-native-bar-ownership.md).

M296 qualifies actual supported payloads with native scoped shell API factories and
routing callbacks, including own DMs open/dismiss and foreign-ID refusal. Full
shell/bar/compositor and API pruning remain separate. See
[native shell APIs](diagnostics/2026-10-24-native-shell-api.md).

M297 qualifies actual native bar/registry and real Wayland popup surfaces in a
private software compositor. Controlled tests fix DMs Escape and mapped-popup
closure after guard loss. Stable/dev/stable,88 tests and connected recovery pass.
Full shell/API lifecycle, Hyprland and cross-root ownership remain. See
[native Wayland evidence](diagnostics/2026-10-24-native-wayland.md).

M298 qualifies native API sync/pruning and actual QObject destruction across three
real-bar profile unloads; all13 API lifetimes close and admitted instances are fresh.
Full shell, captured callbacks and cross-root ownership remain. See
[native API lifetime](diagnostics/2026-10-24-native-api-lifecycle.md).

M299 combines connected state, subscriptions, real native bar/Wayland surfaces and
API lifecycle in one stable/dev/stable test. Sessions and unresolved DM persist;
visible empty-inbox/palette readiness and offscreen crash-recovery regression pass.
See [connected native Wayland](diagnostics/2026-10-24-connected-native-wayland.md).

M300 confirms the remaining shared-authority gap: independent coordinator roots can
admit six guards for one collection despite correct same-root exclusion. No apps
were started. The next contract must bind admission across roots before public
profiles. See [characterization](diagnostics/2026-10-24-admission-roots.md).


M301 adds shared initial reservation and format2 admission:124 tests and a private
compositor run verify competing-journal refusal and live guard fencing. Original
formats remain unchanged. Native packaging, verified handoff and persistent desktop
bootstrap remain; no public profile command is enabled. See
[shared authority](diagnostics/2026-10-24-shared-authority.md).

M302 packages shared authority in immutable format3 native generations.133 tests,
six initial-admission native cases and the legacy connected recovery regression
pass, with original format2 reproduction/refusal retained. This supersedes M301's
packaging gap; verified shared handoff and connected recovery remain. See
[shared native artifacts](diagnostics/2026-10-25-shared-native-artifacts.md).

M303 qualifies shared reservation2 handoff and actual connected native recovery:
158 tests and five controller crashes preserve sessions, subscriptions and unknown
DM intent. Explicit guard EOF after transport drain fixes clean unload. This
supersedes M301/M302's handoff gap for the isolated supported set; native partial
rollback, initial/cross-session recovery and public profiles remain. See
[authority handoff](diagnostics/2026-10-25-authority-handoff.md).

M304 adds actual native partial rollback before issuance and after only DMs loads,
with original-guardian proof, measured EOF cleanup, retained gate evidence and
unchanged connected stable state. This narrows M303's partial-native limit; failed
partial cleanup, Theme/all-member partial startup and persistent startup recovery
remain. See [partial rollback](diagnostics/2026-10-25-native-partial-rollback.md).

M305 qualifies journal3 candidate cleanup recovery for DMs-only partial startup:
rejected deletion preserves the fence, original-guardian recovery survives two
controller crashes, and stable sessions/application state/subscriptions return.
179 tests and full shared recovery regression pass. This supersedes M304's failed
DMs cleanup limit only; other candidate cohorts and startup/bootstrap remain.
See [candidate recovery](diagnostics/2026-10-25-candidate-cleanup-recovery.md).

M306 extends candidate startup recovery to Theme-only and the entire supported set,
with DMs regression. Failed deletion stays fenced, original-owner cleanup survives
crashes, and stable appearance/sessions/application state/subscriptions return.
This supersedes M305's cohort limitation; initial adoption/cross-session bootstrap
and public profiles remain. See [startup cohorts](diagnostics/2026-10-25-candidate-startup-cohorts.md).

M307 qualifies same-original-shell initial activation across four controller crashes,
with one reservation, one construction per plugin, unchanged sessions/application
state and effect-free completed replay. This narrows initial adoption only; owner/
compositor loss, interrupted provisioning and persistent bootstrap still need work.
See [initial replay](diagnostics/2026-10-25-initial-activation-replay.md).

M308 qualifies pre-adoption initial-owner recovery with original cleanup intent,
reservation3 retained history and one replacement construction per plugin.205 tests
and actual failed-cleanup/controller-crash recovery pass; ordinary switching remains
compatible. This narrows M307 only within the same physical compositor; later
adoption/binding, early startup, provisioning/cross-session and public upgrade gates
remain. See [initial owner recovery](diagnostics/2026-10-25-initial-owner-recovery.md).

M309 qualifies initial owner loss after adoption and before binding: canonical
verified cleanup reconciles the original historical binding, then ordinary handoff
resumes.217 tests and seven-crash native stable/dev/stable recovery pass, with
pre-adoption regression. A deterministic snapshot publication race is fixed without
format changes. This supersedes M308's adoption/binding gap; early partial startup,
provisioning/cross-session and public profile gates remain. See
[closed initial binding](diagnostics/2026-10-25-closed-initial-binding.md).

M310 extends initial recovery to before/after issuance and each single-plugin
startup cohort. Original missing/partial admission is preserved, cleanup precedes
replacement, and full-set/adopted recovery regress. This narrows M308/M309's early
startup limits; compatible-generation preflight, provisioning/cross-session and
public profile gates remain. See [initial startup cohorts](diagnostics/2026-10-25-initial-startup-cohorts.md).

M311 adds immutable protocol4 generations and a read-only complete-set compatibility
command.233 tests and actual mixed-version refusal plus connected recovery pass;
old generations stay readable and unchanged. This narrows M308–M310's capability
preflight gap; public activation/upgrade, provisioning and cross-session bootstrap
remain. See [runtime preflight](collection-runtime-preflight.md).

M312 qualifies explicit same-compositor interrupted setup:260 tests and two native
runs preserve the original authority through staged publication crashes, then
connected reservation2/3 recovery. Unbound objects remain recorded; historical
partial authorities are not repaired. Cross-session bootstrap and public activation/
upgrade remain. See [provisioning evidence](diagnostics/2026-10-25-authority-provisioning.md).

M313 characterizes real compositor replacement: the old authority stays fenced,
saved selections construct no new apps, and process closure leaves two remote
channels unresolved. Explicit transport cleanup and durable cross-session evidence
remain required. See [compositor loss](diagnostics/2026-10-25-compositor-loss.md).

M314 adds [read-only retained-authority inspection](retained-authority-inspection.md).
280 tests and actual live/dead/replaced-compositor plus reservation3 snapshots pass;
process closure remains distinct from transport cleanup and activation authority.
Explicit cross-session cleanup and durable successor/bootstrap ownership remain.

M315 adds explicit original-desktop transport cleanup after compositor exit:
297 tests and actual rejected-deletion/controller-crash recovery preserve app intent,
sessions and authority while closing old channels. Cleanup grants no new admission.
Durable successor/bootstrap and broader interrupted states remain. See
[desktop cleanup](diagnostics/2026-10-25-desktop-transport-cleanup.md).

M316 retains verified cleanup/authority evidence outside the disposable runtime:
314 tests and native controller-kill/runtime-retirement checks pass. This enables
historical evidence reads after path loss, not successor admission. See
[desktop checkpoint](diagnostics/2026-10-25-desktop-runtime-checkpoint.md).

M317 adds one linked successor intent and exclusive new-runtime claim, preserving
the checkpoint and canonical history.336 tests pass, including three controller
kills. The cohort adapter is synthetic and this grants no application admission;
native cohort verification and versioned successor activation remain. See
[successor ownership](diagnostics/2026-10-25-desktop-successor-ownership.md).

M318 qualifies linked successor ownership against actual private Theme/DMs state:
three controller kills, no additional HTTP/app construction, six preserved sessions
and unresolved DM.336-test core unchanged. Successor admission remains next. See
[native successor ownership](diagnostics/2026-10-25-native-successor-ownership.md).

M319 qualifies linked authority2 and canonical replacement admission with355 tests,
four publication-controller kills and three real guards. Old history/mirror remain;
native application observations are synthetic. Compatible immutable generations and
native activation are next. See [linked authority](diagnostics/2026-10-25-linked-successor-authority.md).

M320 qualifies complete artifact5/protocol2 bundles and authority1/2 preflight with
373 tests. Frozen original4 reproduces and refuses5; mixed4/5 refuses authority2.
Native successor application activation and subscription restoration remain next.
See [successor bundles](diagnostics/2026-10-25-successor-runtime-bundles.md).

M321 qualifies actual native successor activation: seven new controller kills retain
one three-app activation, five restored subscriptions, six sessions and the pending
DM. Canonical history commits without reset;373-test core unchanged. Repeated desktop
loss and broader public/installed profile workflows remain. See
[native continuation](diagnostics/2026-10-25-native-successor-activation.md).

M322 qualifies private native artifact4-to5 successor activation. Old-set refusal,
three new hashes, seven controller kills and direct subscription restoration preserve
sessions, pending intent and canonical history. Repeated desktop loss and public/
installed profiles remain. See [native upgrade](diagnostics/2026-10-25-native-successor-upgrade.md).

M323 adds read-only linked-authority inspection:391 tests and five actual native
CLI/internal snapshots pass through compositor loss and predecessor-runtime
retirement. Repeated-session cleanup/checkpoint/admission remain next. See
[linked inspection](diagnostics/2026-10-25-linked-authority-inspection.md).

M324 qualifies committed linked-session transport cleanup after the second desktop
exits:410 tests and actual private Theme/DMs pass, preserving sessions and unresolved
DM through rejected DELETE and two controller kills. Durable linked checkpoint and
repeated admission remain next; see [cleanup evidence](diagnostics/2026-10-25-linked-desktop-cleanup.md).

M325 adds a durable linked checkpoint:430 tests and actual native retirement pass,
with ancestry referenced instead of recursively copied. Explicit admission from this
checkpoint remains next; see [checkpoint evidence](diagnostics/2026-10-25-linked-desktop-checkpoint.md).

M326 qualifies new desktop ownership from checkpoint2 with actual locked plugin
cohorts:454 tests and private native three-kill replay pass. No new apps or HTTP are
admitted; explicit authority/guard publication remains next. See
[linked ownership](diagnostics/2026-10-25-linked-session-ownership.md).

M327 proves the checkpoint2-to-authority3 core contract with475 tests and actual
owned guard/controller processes. Application observations remain synthetic; complete
compatible bundles and native plugin activation are next. See
[authority evidence](diagnostics/2026-10-25-checkpoint2-authority.md).

M328 qualifies complete artifact6 bundles and explicit authority3 preflight with493
tests; original artifact5 reproduces unchanged. Actual native checkpoint2 activation
is next. See [bundle evidence](diagnostics/2026-10-25-continuation-runtime-bundles.md).

M329 proves actual private native checkpoint2 activation through package5-to6 and
seven additional controller interruptions, preserving sessions/subscriptions/pending DM.
Repeated-session lineage and public installed integration remain. See
[native evidence](diagnostics/2026-10-25-native-checkpoint2-activation.md).

M330 qualifies read-only authority3 inspection with512 tests and five actual native
lifecycle snapshots. Reusable repeated-session recovery remains next. See
[inspection evidence](diagnostics/2026-10-25-checkpoint2-inspection.md).

M331 qualifies authority3 transport cleanup with532 tests and actual third-compositor
loss, failed DELETE and two controller interruptions. Next follow the
[reusable lineage design](repeated-session-recovery.md); see
[cleanup evidence](diagnostics/2026-10-25-checkpoint2-cleanup.md).

M332 qualifies a bounded, non-leasing legacy ancestry reader with19 private tests.
The reusable runtime family remains next; see
[ancestry evidence](diagnostics/2026-10-25-bounded-checkpoint-ancestry.md).

M333 adds reference-only checkpoint3 with23 focused tests and19 ancestry regressions.
Original checkpoint writers remain unchanged; earlier readers refuse3. Reusable
intent/admission and same-format repeated sessions remain unqualified. See
[checkpoint evidence](diagnostics/2026-10-25-reusable-checkpoint-envelope.md).

M334 qualifies reusable intent3 and dual legacy-reader refusal fences with20
private tests; no authority or application admission is granted. See
[ownership evidence](diagnostics/2026-10-25-reusable-session-ownership.md).

M335 qualifies one reusable recovery family across three further sessions with608
core tests and actual guards. Application observations are synthetic; packaging
and copied native plugins remain next. See
[repeated core evidence](diagnostics/2026-10-25-reusable-runtime-core.md).

M336 qualifies artifact7 and explicit authority4 preflight while retaining the
authority1 default, exact old writer6 reproduction and old-reader refusal.
See [package evidence](diagnostics/2026-10-25-reusable-runtime-packages.md).

M337 qualifies three repeated native Theme/DMs sessions with unchanged artifact7
packages,45 controller interruptions and preserved credentials/subscriptions/pending DM.
See [native repeated sessions](diagnostics/2026-10-25-native-repeated-sessions.md).
