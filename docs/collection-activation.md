# Collection activation: lifecycle requirements

This is a source-backed design and isolated loader characterization, not an
implemented collection command. Stable/dev/edge must select exact plugin artifacts
and separate connections/state while preserving the user's current collection.
Omabit's own desktop profiles do not yet coordinate independent plugin services.

## Observed Omarchy behavior

On the inspected installed source snapshot, `shell.qml`'s `_syncServices` creates
new enabled services before destroying disabled ones. A single configuration
replacement can therefore overlap old and new QML services. Destruction itself is
deferred. `keepLoaded` retains a service across reload and updates its manifest:
selected metadata may describe a newer version while the old code/helper runs.
`PluginRegistryApi` exposes a self-scoped view; an ordinary plugin cannot assume
it receives a collection-wide loader-management API. These are API contracts, not
an OS sandbox for same-user plugin code.

The guarded `scripts/test-collection-loader-offscreen.py` copies four exact loader
functions and runs real Quickshell/IPC/Process in a no-network namespace with
synthetic manifests, services and Python children. It confirms object overlap,
manifest/loaded-generation disagreement, and successful explicit disable → wait
for the owned child to exit → enable. A distinct component URL loads generation2
only after generation1 is destroyed. Separate stable → dev → stable and
stable → edge → stable cycles pass. The opted-in inner driver refuses on the host.
The first trial lacked a configured private wrapper path; later trials target the
exact child Quickshell PID. All attempts remain retained.

This is not the entire Omarchy shell, real Theme/DMs subscriptions, a compositor,
keyring or collection activator. The fixture's own child exits on teardown; other
plugins and detached descendants still need their own measured shutdown contract.

## First supported coordinator

The initial developer-kit coordinator should handle the explicitly supported
Theme/DMs/Omabit set, with no marketplace or arbitrary-plugin compatibility claim.
Before implementing a persistent format, prove these steps with their real
services in the private shell fixture:

1. Prepare immutable artifacts and validate manifest/profile/receiver identities.
   Record the last working collection and endpoint references, without secrets.
2. Stop admission of new application work and let each supported plugin report
   its pending requests. Preserve uncertain writes and credential-cleanup journals.
3. Disable the outgoing collection separately. Confirm its services, watchers and
   helper processes have stopped before enabling replacements. A config write or
   a missing registry entry alone is not that confirmation. If shutdown remains
   uncertain, keep the replacement inactive and retain the pending transition.
4. Activate candidates from immutable component/helper paths. Check the actual
   loaded generation, profile, connection and helper readiness, not just the
   selected manifest. Reject stale callbacks from the previous generation.
5. Give only the active Theme instance authority over visible appearance and
   shared hook effects. Reconcile exact owned includes/files; preserve user edits.
6. Commit the collection after readiness. On failure, stop and verify the candidate
   before restoring the exact retained prior collection. Restore prior appearance
   ownership explicitly; never roll back a runner database, ship or chain as a
   side effect of selecting older desktop code.

Durable transition IDs, restart reconciliation, busy/uncertain operations and
failed rollback must be tested before exposing a general switch command. Hooks,
launchers and auth receivers may outlive a QML object and are part of this gate.

Evidence: `docs/diagnostics/2026-10-24-collection-loader.md`.

## Real helper teardown checkpoint

The real Theme Service and DMs overlay now pass separately disabled/drained
stable/dev/edge roundtrips with empty accounts in a private offscreen namespace.
Both Python helpers exit before replacements; Theme device IDs persist per profile.
DMs uses the overlay loader, so service-only coordination is insufficient. The
applied `scripts/test-real-plugin-lifecycle-offscreen.py` preserves actual helper
code and records its source hashes. See
[the exact evidence and limitations](diagnostics/2026-10-24-real-plugin-lifecycle.md).

Next prove connected subscriptions, pending/uncertain work and Omabit's service
before defining a durable switch journal. Empty-account teardown does not show
that a killed helper completed a write or removed a remote channel.

## Connected helper checkpoint and remaining cleanup gap

M246 passes connected Theme/DMs helper and active-stream teardown against synthetic
Eyre with the real private keyring, using the optional `--connected` harness mode.
It supersedes M234's empty-account-only limit for transport lifecycle, while all
ten server channels remain undeleted after each run. A graceful prepare/drain
protocol must preserve uncertain writes and verify remote channel cleanup before
its result can authorize durable activation. Exact process exit is necessary but
insufficient. See [the observations](diagnostics/2026-10-24-connected-plugin-lifecycle.md).

M249 further characterizes pending writes: after a missing Gall acknowledgement,
the actual DMs UI reports unknown and avoids replay but clears busy/pendingSend.
The text remains in its current object. Therefore busy=false cannot authorize
unloading; preserve unresolved operation identity/intent in the transition contract
and reconcile it without resend. See
[the synthetic lost-receipt observation](diagnostics/2026-10-24-uncertain-plugin-lifecycle.md).

M253 implements the send-only durable handoff prerequisite in DMs. The real helper
restores one unknown send after stable→dev→stable with an unchanged journal and no
replay. Exact authenticated history reconciliation has focused tests. This
supersedes M249's in-memory-only send limitation, not M246 channel cleanup or the
remaining invitation/read-marker/Theme/Omabit lifecycle gates. An empty send
journal still cannot authorize unloading. See
[the evidence](diagnostics/2026-10-24-dms-outbound-recovery.md).

M254 qualifies an explicit DMs transport-only drain: three owned channels are
observed deleted before helper replacement, with the uncertain send/session
retained. Failure paths refuse a closed receipt. `collectionReady` remains false.
This supersedes M246 only for that opt-in DMs path; Theme channels and complete
application quiescence remain open. See
[the evidence](diagnostics/2026-10-24-dms-transport-drain.md).

M255 extends the explicit transport drain to Theme. All six Theme/DMs channels are
observed deleted before stable/dev/stable replacement, with saved appearance and
unknown-send state preserved. This supersedes M246 for that opt-in path only.
`collectionReady` remains false: Omabit, all mutable effects, appearance ownership
and durable activation/rollback still require qualification. See
[the evidence](diagnostics/2026-10-24-theme-transport-drain.md).

M256 applies the repeatable plugin fixture as
`scripts/test-real-plugin-lifecycle-offscreen.py --connected --drain` (with explicit
source/output paths). Five stable/dev/edge cycles delete ten channels and preserve
the uncertain send without replay. The original no-write baseline remains intact.
`scripts/test-omabit-service-drain-offscreen.py` tests the actual Omabit Service's
new internal stop against inert tools: foreground work refuses drain, delayed
inventory exit and late read completion block closure, and callbacks do not restart
helpers. It uses a monitor stub, so real collector/monitor effects remain separate.
Bootstrap activation is unchanged; collectionReady remains false. See
[the evidence](diagnostics/2026-10-24-collection-transport-drains.md).

M257 extends the optional fixture with `--effects`: invitation and read-marker
unknowns persist alongside the send across all profiles, with three total pokes
and no replay. DMs reports these exact effect references in its drain receipt.
This supersedes send-only coverage in M253/M256, not full collection readiness.
Observation-based effect reconciliation and external launchers remain separate.
See [the evidence](diagnostics/2026-10-24-dms-conversation-effects.md).

M258 resolves M257's observation-reconciliation gap for invitation/read effects.
Only a fresh, correctly bound pre-read intent cohort can reach state-confirmed;
this is desired-state evidence, not a recovered original ACK. The optional
`--reconcile-effects` fixture proves read-only confirmation, preserved unknown send
and another helper restart with no extra pokes. See
[the evidence](diagnostics/2026-10-24-dms-effect-reconciliation.md).

The remaining Theme ownership problem is concrete: separate appearance files can
still both execute when their guarded theme names match. Transport closure alone
does not remove an older profile's Hyprland include or suppress its installed
notification hook. The next implementation must coordinate one active appearance
owner, make old includes/hooks inert, preserve edited user files and restore the
previous owner's appearance on rollback. Keep this inside private desktop fixtures;
no production profile may be used to prove it.

M259 supersedes the uncoordinated appearance/include/hook condition above for
cooperating development helpers. One HOME lock prevents helper overlap; applied
owner guards make inactive Lua inert; persistent selection IDs reject delayed
hooks; hooks no longer dispatch the secondary UI publication path. Actual Lua,
helper and connected-drain fixtures pass. This is not automatic rollback or a
complete collection switch. Older/unreceipted overrides require preservation and
migration, and old helpers must exit before using the new contract. See
[the evidence](diagnostics/2026-10-24-theme-appearance-owner.md).

M260 narrows M259's legacy-migration gap: exact adapter2 bytes reconstructed from
retained application evidence are backed up and adopted before startup reconcile.
Actual helper migration/restart passes without stale publication. Unknown/edited
files still refuse intact. Automatic partial-apply restoration and the durable
collection transaction remain separate. See
[the evidence](diagnostics/2026-10-24-theme-legacy-adoption.md).

M261 demonstrates explicit online previous-profile restoration with actual services
and helpers after four partial desktop-command effects, plus failed restoration
and verified retry. The temporary managed-theme capture stall is fixed only with
intact ownership evidence and authenticated hub adoption; uncertain intent remains.
This satisfies a private restoration prerequisite, not the durable switch itself.
Immutable offline appearance, native compositor/fanout, external processes and
full transition/restart reconciliation remain. The reproducible mode is
`--connected --appearance-rollback`. See
[the evidence](diagnostics/2026-10-24-theme-partial-rollback.md).

M262 adds explicit offline Theme checkpoints/restore requests. Actual private CLI
recovery succeeds with hub/keyring stopped and source themes changed, retaining
pending work through a failed command and forced helper exit. Success replay is
read-only. This supersedes M261's offline snapshot gap, not automatic collection
rollback. The native theme-set read lock covers snapshot/readback; wallpaper is
excluded. Native template regeneration, app-retint fanout and font-triggered shell
restart remain separate gates. The future durable coordinator must survive outside
plugin QML and cannot equate a per-profile receipt with collection readiness. See
[the evidence](diagnostics/2026-10-24-theme-offline-checkpoint.md).

M263 qualifies copied native headless theme staging/templates in the offline
matrix. Frozen files survive real native staging and match byte-for-byte after
failure and helper-kill recovery. Use --native-theme-staging with the offline mode.
This narrows M262's native-file staging gap; font-triggered shell restart, app-retint
and compositor behavior remain separate. See
[the evidence](diagnostics/2026-10-24-native-theme-staging.md).

M264 confirms the native restart boundary with real private Quickshell: shell-owned
work loses its result during font-set, while an externally owned command runner
survives. Native theme-set exit0 also hides a deliberately failed retint, and its
background cache lifetime differs under the command runner's cleanup. Keep durable
coordination outside plugin QML and require explicit new-shell/appearance evidence.
The actual offline Theme CLI must still be integrated across this boundary. See
[the evidence](diagnostics/2026-10-24-native-appearance-effects.md).

M265 integrates the actual offline restore CLI across copied native font/shell
restart. Independent ownership preserves it through real private shell replacement;
post-effect failure stays applying until a fresh verified retry, with sync journals
unchanged. This supersedes M264's missing CLI integration. Full plugin-service
collection recovery, monitor/launcher/auth quiescence, app/compositor effects and
persistent activation are still separate. See
[the evidence](diagnostics/2026-10-24-native-font-restore.md).

M266 adds shared reader exit accounting for actual monitor/runtime-log and separate
project/SSH setup controls. The --readers fixture passes delayed exits and five
profile cycles; late SSH results are rejected. Removing a still-pending control
retains reader-owner-lost rather than reporting closure. This supersedes M256's
monitor-stub-only limit, not real collector/auth/clipboard child accounting or
full activation. Old receipts without auxiliaryReadersTracked cannot satisfy this
prerequisite. See [the evidence](diagnostics/2026-10-24-omabit-reader-drain.md).

M267 proves the actual receiver adapter can outlive a signal-killed owner. Omabit
therefore retains helper-descendants-unconfirmed after signaled helper exits;
even an empty shell child list cannot override it. The real-QML auth crash leaves
an inert detached child alive and refuses closure correctly. Normal/cancel adapter
cleanup, active-auth drain refusal and five-profile regression pass. This supersedes
M266's untracked abnormal-exit gap. Old receipts without helperExitsTracked cannot
satisfy this prerequisite; a future coordinator must preserve uncertainty across
its own restart. It must not clear the fence merely by reloading the service.
See [the evidence](diagnostics/2026-10-24-helper-exit-fence.md).

M268 combines Omabit/Theme/DMs using the same copied native loader functions and
real plugin components. Six drains each pass across stable/dev/edge and a final
reconciliation reload, deleting12 channels while retaining the unknown send and
confirmed effect journals. Theme's busy refusal and asynchronous overlay readiness
must be awaited explicitly. Omabit tools remain inert and appearance is disabled.
This supersedes the separate-service limitation, permitting work on the first
persistent coordinator contract without claiming full native activation. The next
contract must journal selected/loaded artifact identities, phase and outstanding
intent before effects, survive shell replacement, preserve unconfirmed helper
lineage, and refuse unknown formats/ambiguous rollback. See
[the evidence](diagnostics/2026-10-24-supported-collection.md).

M269 implements the first internal [format1 journal](collection-journal.md), with
24 private storage/transition tests and original-writer fixtures. This supersedes
the absence of durable intent in M268, not the missing runtime coordinator. The
reducer accepts observations from an adapter; it cannot itself prove readiness,
helper exit or artifact identity. Runtime binding and effects are the next gate.

M270 binds the journal to the real supported plugin set in the private fixture.
Stable/dev/stable and failed edge rollback survive six coordinator kills after
effects but before receipts, using the actual offline Theme CLI and retained
component/process identities. All16 channels close;30 private checks and the
zero-write baseline pass. This supersedes M269's model-only limit for that fixture,
not whole-shell replacement or a production loader API. The private adapter's
instrumented identity protocol must not be presented as stock Omarchy support.
See [the evidence](diagnostics/2026-10-24-collection-coordinator.md).

M271 passes two controlled private shell replacements between effects/receipts.
Unexpected owner death instead leaves two helpers and an undeleted DMs channel;
reader-owner-lost persists across new-shell creation and exact fixture cleanup.
This supersedes M270's missing private shell-replacement evidence, not production
orphan/channel recovery. Plugin-owned durable cleanup evidence and explicit fence
resolution are required; a reload cannot clear uncertainty. See
[the evidence](diagnostics/2026-10-24-collection-shell-recovery.md).

M272 fixes DMs EOF handling: actual shell death now drains its channel, and all16
fixture channels delete. All118 private DMs checks pass. The M271 uncertainty fence
still persists; its surviving processes are the inert Omabit test readers. This
supersedes the EOF channel leak only. Durable cleanup evidence after helper SIGKILL
or unsuccessful reconnect remains next. See
[the evidence](diagnostics/2026-10-24-dms-eof-drain.md).

M273 adds plugin-owned DMs transport evidence:147 tests and real private helper
SIGKILL/lost-delete/reconnect scenarios pass without replaying uncertain app work.
Final-source collection regression still preserves reader-owner-lost. Do not treat
newly opened subscriptions as recovery authorization. Independent recovery must
bind the old loaded instance, its journal/closed channels and remaining app intent;
actual Urbit/credential expiry/archive/native-adapter gates remain. See
[durable transport evidence](diagnostics/2026-10-24-dms-durable-transport.md).

M274 qualifies DMs transport cleanup against fresh actual Urbit. Deletion alone
may leave the HTTP reader blocked; explicit owned-socket cancellation plus verified
thread exit is required. Same-session cleanup after two helper kills/lost204 and
four restored subscriptions pass; deleted channels independently return404.
The full private collection regression still preserves reader-owner-lost. This is
transport evidence, not authority to erase coordinator uncertainty. See
[the actual-Urbit gate](diagnostics/2026-10-24-dms-urbit-transport.md).

M276 adds Theme operation leases and durable watch/poke cleanup. Private concurrent
client kills preserve a live watch and an uncertain settings update; exact readback
confirms state with no repeat poke. All268 Python checks and the full collection
regression pass. Keep reader-owner-lost fenced: safe credential-preserving disconnect,
actual-Urbit Theme recovery and independently bound coordinator recovery remain.
See [Theme transport evidence](diagnostics/2026-10-24-theme-durable-transport.md).

M277 qualifies credential-preserving Theme disconnect and handoff cancellation in
private checks. Session retirement fences reads/new channels and survives a killed
CLI; exact cleanup completes before credential removal. Format1 upgrades intact
with original-reader downgrade refusal. Keep collection owner-loss uncertainty
separate; new sessions and these receipts alone do not clear it. See
[disconnect evidence](diagnostics/2026-10-24-theme-safe-disconnect.md).

M278 verifies Theme against a fresh real Urbit: original-session cleanup after two
client kills/lost204, restored subscription and actual CLI disconnect pass, with
two independent channel404s before logout. This extends M277 beyond a synthetic
peer. It still does not erase collection owner-loss uncertainty; recovery must
bind the retained old instance and its unresolved application work explicitly.
See [actual Theme transport evidence](diagnostics/2026-10-24-theme-urbit-transport.md).

M279 qualifies a private external subreaper prerequisite. Known-child snapshots
are insufficient after owner death; same-guardian ECHILD evidence covers adopted
and late-forked descendants. Guardian loss remains uncertain even if a new guardian
is empty. The actual QML set must next run under this owner before any explicit
collection recovery can use its proof. No journal fence is cleared by this work.
See [process-supervision evidence](diagnostics/2026-10-24-collection-supervisor.md).

M280 binds a surviving external guardian to actual private QML activations. Four
guardians certify complete owned-process exit, including surviving readers after
shell SIGKILL; all16 channels close and the same-source process regression passes.
The journal still retains reader-owner-lost. Process closure must next be combined
with the exact old instance's plugin journals in an explicit durable recovery
transaction. This supersedes M279's missing real-component integration only.
See [supervised collection evidence](diagnostics/2026-10-24-supervised-collection.md).

M281 extends the supervised fixture with unresolved actual DMs and Theme writes.
Both intents survive old-owner loss and cleanup unchanged, without replay; all17
channels close. M280 itself lacked that pending application cohort. The collection
fence remains until a separately versioned recovery transaction is implemented.
See [the cohort evidence](diagnostics/2026-10-24-collection-recovery-cohort.md).

M283 integrates format2 explicit recovery with actual private plugin journals and
saved sessions. Two coordinator kills and failed deletes preserve the fence until
original-guardian closure and unchanged application/credential cohorts verify.
Plugin-owned cleanup closes17 channels without app replay. The switch reaches
checkpoint-appearance; candidate activation/return after recovery is next. This
supersedes M281's permanent-fence state only for the characterized private adapter.
See [explicit recovery](diagnostics/2026-10-24-explicit-collection-recovery.md).

M285 qualifies candidate activation and original-profile return after explicit
recovery. Ten new positive subscriptions use the retained sessions; the unknown DM
stays visible and Theme confirms its original intent by readback without another
poke. All21 channels close. Passive versus explicit publication is separated, and
the same corrected harness rejects the old replaying source. This supersedes M283's
remaining return gate and M284's failed condition for the private fixture only.
See [recovery roundtrip](diagnostics/2026-10-24-recovery-resume.md).

M286 adds internal immutable artifact preparation and loaded identity verification
with the complete copied native registry and stock enable IPC. Native service
metadata can refresh while the generated loaded identity correctly stays old.
Five private roundtrips pass with synthetic application bodies; actual wrappers,
bar widgets, profile-scoped Omabit IPC and durable admission remain. In particular,
prepared inherited components must not be enabled before the external coordinator
can fence their creation across shell loss. See
[the artifact evidence](diagnostics/2026-10-24-native-artifacts.md).

M287 makes actual Omabit monitor/widget/setup IPC follow injected profile identity
and passes the real Panel/Service with native controls in five private cycles.
Unknown Quickshell targets return exit0, so absence checks use response content and
inventory. Native creation before property injection and durable gated activation
remain separate requirements. See
[scoped IPC evidence](diagnostics/2026-10-24-scoped-omabit-ipc.md).

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

## Shared runtime authority gate after M300

The [two-root characterization](diagnostics/2026-10-24-admission-roots.md)
confirms that separate coordinator/admission directories can admit six guards for
one identical collection. This is not an application-overlap test; real helper
locks are separate and do not establish a constructor/appearance admission barrier.

The next internal contract must provide one explicitly provisioned shared runtime
authority outside profile-specific XDG directories. Bind it to a canonical
coordinator journal and the actual runtime context; verify physical directory and
resource identity, not a caller's matching display label. Independent controller
processes may share that journal and its existing lease. An unrelated journal
root cannot acquire the same authority by choosing a new permit directory.

Persist reservation before candidate issuance and bind every constructor claim to
that authority, exact coordinator/activation, shell identity and immutable members.
Loss of a controller, guard, lock or socket must not by itself release prior
uncertain effects. Reuse the original guardian plus journal-bound cleanup proof;
a new empty guardian or caller-provided success flag cannot authorize handoff.
Malformed/missing/replaced authority evidence must preserve the fence.

Keep current format1 permit and format2 artifact cleanup/readability. New global
admission requires an explicit versioned contract and old-reader refusal; do not
retroactively claim that old hosts enforce it or rewrite their active seals.
Qualify two controllers with one authority, independent roots targeting the same
runtime, death at reservation/issuance/handoff, exact replay, replacement filesystem
objects, and migration/downgrade refusal before exposing a profile command.
These cooperative integrity controls are not a same-user code or wallet sandbox.


## Initial shared authority implementation (M301)

`runtime_authority.py` provides explicit fresh provisioning, a canonical journal
mirror, physical runtime/lock/socket bindings and durable first reservation.
`shared_admission.py` and its guard use admission format2 and validate the shell's
actual display context before granting or renewing a claim. Another journal,
another gate, a copied permit or a replacement shell cannot acquire the reservation.
Local closing/ending works when external authority is lost; no new effects do.

The [M301 evidence](diagnostics/2026-10-24-shared-authority.md) narrows M300 for this
new internal path only. It does not change legacy/native profiles. No release or
handoff is implemented, so owner loss remains fenced. Native immutable packaging,
journal/original-guardian-backed handoff and persistent bootstrap across desktop
sessions remain mandatory before public profile controls.

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
