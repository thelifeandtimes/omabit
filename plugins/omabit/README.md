# Omabit unified desktop plugin — 0.2.0 development

## Project network history (unreleased)

Setup → Project networks reads the selected runner’s saved project reservations
and operation history. The view pins the runner identity, supports observer
access, and requires the advertised project inventory/history capabilities.
It shows interrupted receipts and whether older operations are omitted. Saved
completion is not a current ship-health observation.

This view sends only `projects` and `project-status` reads. It clears data and
cancels pending reads on runner/profile changes, loss of availability or closing
the pane; late replies cannot fill a different profile. Creating networks and
member lifecycle actions remain CLI workflows while their UI is developed.

## Shared card presentation

Card headings use one uppercase monospace caption style. Urbit Logs keeps its
pause/play and refresh icons in the top-right header, with the priority filter
below. Submission/error toasts overlay the view without moving ship context;
ongoing runtime/connection diagnostics and identity-bound request failures live in
Connectivity & runtime. Runner management cards show a compact inventory/access
summary and put endpoint IDs and host compatibility reports under Technical details.

The remote host field is an editable dropdown. Suggestions are read without sudo
from literal SSH Host aliases, bounded Include files and readable known-host entries.
The reader never runs SSH, Match exec or KnownHostsCommand, and does not contact
suggested hosts. Wildcard entries and hashed hostnames are omitted; custom hosts
remain editable, and selecting a known-host port fills the existing port field.

## Guided remote runners

**Connect another runner** and **Set up remote server** now use the same guided
flow. Enter an ordinary SSH alias, authenticate/approve the detected action in the
setup terminal, then return to verified enrollment. Advanced retains prepared SSH
and Unix-socket connections. See [remote onboarding](../../docs/remote-runner-setup.md)
for authentication, dedicated keys, fresh dependencies, resume and revocation.

## Provisioning choices

**Add a ship** offers **Boot Ephemeral Identity**, **Boot Permanent Identity**, and
**Import existing pier**, with a contextual explanation of comet keys, owned
identities/Azimuth, or bringing an existing ship's data into management.
New boot forms default to **Recommended Apps**, including when registering another
owned ship. **bare bootstrap** remains an explicit choice. A missing configured
recommendation is reported rather than silently changing the selection.

The comet form explains automatic first-boot identity/key generation, the local
instance label versus the network's `~name`, and reuse of the same pier on restart.
Import provisioning displays its form directly and always keeps it open for the
selected runner. Discovered piers and registered locations use compact rows with
full paths on hover and inline actions; deletion still opens its typed review.

## UI-only runtime and maintenance polish

- The ship context shows **healthy (2GB loom)** while the ship is running; stopped
  ships have no loom suffix. There is no separate Runtime status card. Recorded/requested
  settings, last operation, snapshot interval and container/host diagnostics are
  under **Technical details**. Current boot progress and recovery notices remain
  visible. The status suffix prefers recorded launch evidence; its tooltip explains any
  different requested size and distinguishes loom address space from RAM usage.
- **Urbit Logs** shows the original timestamped output once. The **Priority logs only**
  toggle filters the available tail for error/warning text with surrounding lines;
  turning it off restores all output. This is a display filter, not a ship-health assessment.
  Hover the heading for read-only/redaction information.
- Manual maintenance has direct **Run pack**, **Run meld** and **Run roll** actions,
  using the existing identity/revision-bound dispatch and stopped-pier checks.
  History removal still has its separate exact-history preview and consent.

These changes use the existing binaries. Automatic runner-update discovery and a
one-click installer need a separate helper/API change: the current local setup
returns immediately for an already connected host, and the collector does not
expose a remote upgrade command. Missing capabilities are not a release-version
comparison, so they are not presented as a verified update-available indicator.

## Host compatibility checks

Local setup, **Manage runners**, and the ship's **Technical details** display the
runner's structured host report: distribution/provider/profile, installed packages
and running engine, compatibility, qualification, provenance, restart state,
permissions, evidence time, checks and native-distribution guidance. Expand the
existing local **View details** control for the full local report. Compatibility
warnings stay separate from pier recovery. Only a fresh, matching live report's
`can_stop` and `can_mutate` flags restrict controls; Stop is independent of admission
for new runtime work. Metadata-only reports and qualification labels are diagnostic.
Every request still goes through the runner's current safety checks.

Host revalidation is automatic through the existing collector inventory polling,
including after native package updates and explicit engine restarts. The runner
owns bounded evidence caching and freshness. **Refresh** also rechecks local setup
metadata using `omabit setup-status` and restarts collector observation. No opt-in
setting is required. These reads do not install/update packages or restart engines;
no update hook, remote executable policy or desktop configuration change is needed.

Older runners retain their ship inventory and controls and show an explicit
`host-compatibility-v1` upgrade hint. A missing or invalid optional report affects
only host diagnostics, not the entire fleet. Local setup status forwards
`hostsetup.Report.compatibility`; collector observations use `host_compatibility`.

Opening **Monitor & maintenance** activates the existing monitor on its workspace
instead of merely marking it visible. Activation matches this process's exact
window and also handles special workspaces. See the
[focus fix and live check](../../docs/diagnostics/2026-10-13-monitor-focus.md).

The development storage update adds **Delete pier…** to ship Overview and registered
locations; **Import an existing pier…** reviews an absolute host source and performs
the move/enrollment; **Storage locations** configures additional runner-owned roots.
New ships and imports choose a location while keeping the exclusive parent mount
and `/urbit/pier` layout. A dedicated, theme-tinted SVG replaces the font-based logo.
These controls require schema 14 or newer and the matching capability
flags. See [storage management](../../docs/storage-management.md) for host permissions,
interruption behavior and CLI equivalents. RC.2 uses schema 15 and includes durable
automatic resume after an unclean host reboot.

This release candidate includes a **Boot pill** selector for new comet, owned-key and development
fake forms: **bare bootstrap** uses Vere's canonical bootstrap.urbit.org pill;
**Recommended Apps** uploads the release asset configured in `recommendedPillUrl`.
The default URL names the frozen rc.1 draft. An empty setting blocks recommended
registration; download failures are explicit. Bare is independent of this setting.
Selection is fixed at registration; initialized piers resume without a pill.
The automated pill builder is deferred to 0.2.0; 0.1.0 uses a user-provided artifact.

Version 0.22.3 keeps an identity-pinned connection per runner during inventory
watching, rather than creating an SSH/PAM/logind session for every two-second poll.
Failed observations discard the connection and back off retries to 30 seconds;
healthy observations retain their normal cadence. Role/capability observations and
per-request grant revocation remain live, and canceled watches close their transports.

Version 0.22.2 raises Omabit's boot-pill download/staging/upload bound to **4 GB
(4,000,000,000 bytes)**. This is separate from the 100 GB pier-transfer bound.
The confirmed 0.1.0 desk/publisher composition and deferred `%omachan` work
are recorded in the [pill release plan](../../docs/pill-release.md).

Version 0.22.1 removes the fixed registration-count cap throughout the runner,
collector and UI. That release capped stopped-pier copy/handoff contents at
**100 GB (100,000,000,000 bytes)**, with a separate 32 MiB archive allowance.
Current polishing source raises local import/copy to 8 TiB while retaining the
100 GB cross-host handoff cap. Published0.3.2 needs an update for the local fix.
Size boundaries and a 512 GiB same-filesystem import are tested using sparse
fixtures; a full transfer above 100 GB has not been qualified.

Version 0.22 adds **Move to another host** on a ship's Overview page. Stop first,
select a second pinned operator runner, and explicitly move its current pier.
The source is permanently fenced; the destination remains stopped until Start.
Interrupted transfers can continue from durable phase receipts. There is no
historical pier restore action. See [stopped host moves](../../docs/host-handoff.md).
Host-port admission probes real listeners; safe Docker launch categories appear
in runtime diagnostics. Both runners need the updated host-handoff capability.

Version 0.21 starts Vere with `--snap-time 60` by default. Runtime status distinguishes
the recorded container interval from the new-start default. Existing processes
adopt the setting on their next explicit stop/start; manager updates do not change
live process arguments. Offline utilities retain their dedicated command lines.

Version 0.20.2 corrects the tilde's optical offsets using fractional spacing,
avoiding the previous rounded horizontal overcorrection.

Version 0.20.1 centers the tilde using its painted glyph bounds and places Open ship
beside Copy +code in the ship header, using the same browser-host/endpoint routing
as the menubar action.

Version 0.20 uses one permanent layout: `.../<ship-directory>/pier`. Each container
mounts only its exclusive parent at `/urbit` and uses `/urbit/pier` for boot,
resume and maintenance. First boot keeps the same container running; there is no
staging/promotion step. Discovery, enrollment and moves use the actual `/pier`
path. The local fleet is migrated offline to schema 11; old layouts are refused.

Version 0.19.1 cold-replaces an enabled panel during `--replace --enable` installs,
avoiding overlapping QML teardown/creation in Quickshell. Deferred registration and
log refreshes use component-owned timers. Updates require an unlocked desktop and
must run separately from OS authorization prompts.

Version 0.19 adds per-ship runtime status, the last lifecycle operation/error,
exit/OOM evidence and bounded read-only Vere/dojo output. Logs refresh every three
seconds on the visible overview, with pause/refresh controls and a short boot
narration summary. Operator-only log reads work during boot and recovery; known
key/code patterns and terminal escapes are removed before output reaches the panel.

Version 0.18.2 adds Start beside owned-ship registration. It is disabled until the
selected runner reports the matching registered ship in a startable state. Starting
opens that ship's overview to follow boot progress. Re-entering an already
registered identity after a panel reload restores Start without needing the key again.

Version 0.18.1 uses Qt's non-native dialogs for key and pill browsing, avoiding the
GTK/GVFS chooser crash inside the shared Quickshell process. Mapped-window fixtures
exercise opening, canceling, reopening and accepting both file pickers.

Version 0.18 consolidates new comet, fresh-key and existing-pier provisioning under
**Add a ship →**. Development fake/qualified-import controls are secondary options
in that same flow; there is no separate key-boot card or outdated upcoming screen.
**Manage runners** lists the local collector's enrollments (including unavailable
ones) and provides connection/presentation editing plus Connect another runner.
Ship HTTP configuration now lives in the ship's Connectivity & runtime section.

Activity cards place time at top right, the host/instance/status context at bottom
right, and Details beneath the action. Key errors stay with their form and no
longer repeat across Setup or unrelated runner pages. Public-name validation now
identifies malformed syllables separately from transport, port or key-file errors.

Version 0.17 fixes the omitted-false discovery flag that exposed enrollment buttons
on already enrolled piers. Reviews expand inside their own cards and preserve
drafts across inventory updates. Menubar and full-window +code use the same fresh
selection component and show a short clipboard toast. Activity identities are
smaller, the task metric reads Task Records, and the menubar mark is a circled tilde.

Owned key boot requires only the expected Urbit ID and private key file. Instance
name and tier path are derived. HTTP and Ames UDP ports are allocated by the runner
inside the registration transaction; Advanced configuration exposes port overrides.
The Boot pill dropdown defaults to Recommended Apps and also offers bare bootstrap. Recommended Apps uses
`recommendedPillUrl`; bare always leaves bootstrap selection to Vere. The CLI also
accepts local pill files. Selected pills are staged by hash/size through an
authenticated bounded upload.

Version 0.16 adds **Pier storage & livenet import** and **Boot from a key** in Setup.
The runner discovers ordinary `.urb` piers below `fakeships/` and classified
`livenet/` directories; project/group folders are supported and `archive/` stays
inert. Fleet prompts offer explicit enrollment or reconnection of moved piers.
Missing/replaced data is reported without recreating it. Stopped moves preserve
the pier and never overwrite a destination.

New registration accepts a convention-relative pier path. Key boot uses a private
file selected by the helper, with no key contents in QML or the operation journal.
Imported livenet piers verify identity with Ames disabled before a networked resume.
See [the storage guide](../../docs/pier-storage.md) for paths, filesystem access,
CLI usage, supported import scope and acceptance status.

Version 0.15 uses matching `host / instance / status` context rows in monitoring
and activity, compact action/date rows, and ship selection markers only on ship
pages. The header puts the version under the title, uses icon-only refresh and
leaves fullscreen control to the window manager.

Saving a routine now enables it and allows the scheduled task to stop a running
ship. The resume checkbox remains an explicit choice; existing policy settings
are untouched until an operator saves or toggles them. Disable remains on the
saved-policy card.

**Container CPU** and **Container Memory** charts have numeric vertical axes and
local-time horizontal axes positioned by actual sample timestamps. CPU follows
Docker's convention: 100% is one logical CPU; the current two-CPU cap is 200%.
Rates require two fresh, increasing counter readings for the same container.
Missing/reset samples are gaps, genuine zero remains zero, and replacement clears
the old container's series. Histories are bounded to this desktop session. The
retention section is labeled **Epoch history**.

Version 0.14 groups each runner's fleet as galaxies, stars, planets, moons, comets
and fakeships, with running instances first inside each group. Ship identities are
primary, instance names secondary; fakes use italic bracketed names. Long identities
wrap in full views and use themed full-name tooltips in compact navigation.

Activity cards lead with the subject ship, followed by event/status and quieter
runner/date metadata. The full window includes Setup and a prominent **Copy +code**
action. Confirmed stopped containers show **N/A**, while unavailable observations
remain unknown. Saved routine cards show schedule, enabled/disabled state, next run,
downtime/resume behavior and explicit enable/disable/edit actions. The current runner
supports one saved pack-or-meld policy per ship.

**Inspect epochs** opens the completed inventory for review. Retention language uses
“oldest history”: an epoch is a segment of the event log; a prefix is a consecutive
group starting at the oldest retained epoch. Roll dates come from completed runner
journal records, never filesystem modification times. Last-roll/prune summaries use
local human-readable dates; full records remain under Technical details. A newer
roll takes precedence over an older pruning result in the historical timeline.

New installations start at **Setup → Set up this computer**. The guided local flow
checks prerequisites, prepares private desktop access, requests OS authorization,
installs host services and enrolls the verified runner. Progress stays visible and
the completed flow offers **Add a ship**, with new comet creation as the default.
See [the installation guide](../../docs/install.md).

Registration distinguishes instance names, fake Urbit identities and pier directory
names, suggests unused names/ports and retains the form on server errors. Fleet
rows label development fakes explicitly and distinguish guest Ames settings from
host HTTP bindings. Consent controls use a shared left-aligned checkbox component.

The desktop has three coordinated surfaces:

- **Fleet menu:** runners and their ships, status dots with descriptive tooltips,
  Ames/HTTP ports, browser links, start/stop, and a shortcut to monitoring.
- **Setup pane:** guided local installation, SSH/socket enrollment, runner
  display/location/browser-host preferences, comet/development-fake registration,
  HTTP configuration and application sessions.
- **Monitor & maintenance window:** ship navigation, live RAM bars and session
  trends, epoch timeline, manual maintenance and confirmed retention review,
  expandable technical details, and a unified operation/incident feed filtered by
  runner, ship and event type. Open it from the menu or
  `omarchy shell thelifeandtimes.omabit.monitor open`.

Runner presentation preferences are local Omabit policy, stored in
`$XDG_STATE_HOME/omabit/panel.ini` (default `~/.local/state/omabit/panel.ini`) keyed
by pinned runner ID. A browser host is a scheme plus host; each ship's configured
HTTP port is appended. SSH loopback endpoints require this explicit host choice.
This does not create tunnels or rewrite collector enrollment. Memory trends are
bounded, session-only, deduplicated by daemon sample time and reset on container
replacement. Unknown/disconnected samples are gaps, never zero usage.

The `+code` control requires the runner's `login-code` capability and operator
access. The desktop helper fetches the code directly, hands it to foreground
`wl-copy --sensitive` over stdin and releases its provider after 30 seconds. Codes
do not enter QML, command arguments, environment, snapshots or journals. The
runner records only an attributed access outcome. The live 6N runner gate is
documented in `docs/checkpoints/24-runner-edit-and-code.md`.

Setup can now rename/rebind a collector enrollment: it verifies the same pinned
runner at the proposed endpoint and rejects stale revisions or collisions.
Setup → Add a ship → Development import supports the `import-qualified-fake`
capability: select an unclaimed source from the runner's completed copy/resume
qualification, review its digest and destination, and explicitly confirm import.
The copy runs durably and leaves the new instance stopped. Progress/provenance are
shown in monitoring. This does not accept arbitrary external paths or real piers;
external uploads/validation and key-based boot remain later work.

Version 0.7 adds **New comet** registration for an operator on the explicitly opted-in
`development-managed` endpoint. The first Start generates an identity internally;
future resumes and health observations enforce the same identity. Registration
requires network acknowledgement and does not start the runtime. Comet maintenance
requires its additional capability. Setup also exposes an explicit same-ID mode
acceptance action when an enrolled fake-only endpoint changes mode. See
`docs/checkpoints/28-managed-comets.md` for activation and compatibility.

Version 0.8 adds runner-owned **Routine maintenance** policy editing for qualified
fake-pier pack/meld. Select UTC weekdays/time and dispatch window; downtime and
resume each require explicit policy values. Policies bind both configuration and
policy revisions. The runner records every due slot durably, skips missed or
ineligible slots, rechecks owner access and resource admission, and executes one
stop/maintain/optional-resume operation under the host work gate. Interrupted work
is not continued after restart. See `docs/checkpoints/29-routine-maintenance.md`.

Version 0.9 extends manual pack/meld and routine editing to initialized comets when
the runner advertises **`comet-compaction`**. Every resume verifies the pinned
identity, and failed utilities do not trigger an automatic resume. Additional epoch
controls require their separate capability; automatic pruning is not enabled. See
`docs/checkpoints/31-managed-comet-compaction.md` for the accepted live gate.

Version 0.10 adds comet epoch inspection, explicit roll and retention previews when
the runner advertises **`comet-epochs`**. Another new roll requires a healthy runtime
cycle and explicit stop. Without the additional pruning capability, comet retention
is preview-only: the collector and panel refuse to inherit fake-pier deletion capabilities. The newest
two epochs stay protected, and review state includes instance mode as well as
runner/revision/inventory identity. See `docs/checkpoints/33-managed-comet-epochs.md`
for the accepted managed gate.

Version 0.11 adds confirmed comet prefix pruning when the runner advertises
**`comet-prune-prefix`**. A fresh exact preview, qualified exclusive-file layout,
operator role and explicit history-loss checkbox are required. The newest two
epochs stay protected. New capability availability alone never upgrades an old
preview. Pruning leaves the ship stopped for explicit identity-pinned resume;
partial deletions require review. See `docs/checkpoints/35-managed-comet-pruning.md`
for the accepted managed live gate.

After updating, use `omarchy restart shell` if the desktop retains cached QML;
an enabled registry entry alone does not verify the loaded UI version.

This initial Omarchy panel observes explicitly enrolled runners through the
bundled `urbit-collector`. It displays connection availability, instance desired/
observed state, health, revision drift, active operations, and incidents. Operators
can register a fake instance, explicitly start/stop its runtime, stage an HTTP-port
change, or enroll another prepared SSH host. Application enrollment, masked sign-in,
status, logout, and origin-bound credential handling are separate from management.
The bundled `omabit` helper owns palette/Talon policy; the collector stays generic.

Every mutation carries the selected runner ID and configuration revision. The UI
does not silently change revisions or retry mutations. Unavailable/stale
observations, observer grants, busy instances, and recovery-required state disable
inappropriate controls. Closing the panel does not stop a ship or cancel an
operation already accepted by the runner.

Version 0.3.0 adds container resource samples, recent operation history, maintenance
results, and explicit stopped-pier pack/meld/roll confirmations. Roll's runtime-cycle
guard is displayed. Resource samples keep their source, container identity and
timestamp; unknown readings remain unknown and container RAM is not loom usage.

Inspect epochs, then open **Review retention** to select an oldest-prefix cutoff.
Epoch names remain strings, including uint64 events beyond JavaScript's safe numeric
range. The newest two are protected. The collector preview must match the selected
runner, instance, revision, sample time, digest, entries and counts. A separate
history-loss checkbox enables **Delete reviewed prefix** only for a qualified layout
and operator capability. Stale or replaced observations clear the preview; changing
the cutoff requires a new preview and confirmation. No action stops or starts the
runtime implicitly. Failed/ambiguous submissions retain their request ID and are
never automatically retried. Logical and allocated sizes are not promises of
reclaimed space; retained metadata is not backup verification.

This is the Omabit-specific frontend. Generic runner/collector code has no
Quickshell dependency. The monorepo is authoritative; generated plugin microrepo
packaging is a later release step. `scripts/install-panel.py` installs a reviewed
snapshot containing these files and the built collector binary, without requiring
root. Replacement refuses modified/unmanaged installed files.

The existing `thelifeandtimes.urbit-theme` plugin has a separate origin-scoped
credential namespace and publication consent. The explicit `omabit theme-migrate`
preview/apply workflow transfers it with a recovery journal. Installing/updating
this panel does not perform that migration. Normal-urgency desktop notifications
report connection changes, new incidents, and failed operations without classifying
unreachability as a ship crash. Host setup and real-key bootstrap retain their gates.

See `docs/checkpoints/09-management-panel.md` in the source monorepo for the visual
acceptance walkthrough and `docs/checkpoints/22-maintenance-panel.md` for the 6L
review gate. No production hosting support is implied by this panel.
