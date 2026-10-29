# Platform goal — paused

Complete the [platform roadmap](platform-roadmap.md) through staged acceptance
gates. Goal set by the user on 2026-10-21; not complete. Do not turn progress on one
gate into a release-wide completion claim.

The latest user constraints are in [experiment isolation](experiment-isolation.md)
and take precedence over historical deployment/reset permissions. In particular,
preserve managed `~lismyl-lismyl`, exclude tmux ships and `~/temp-ur`, and keep
experimental plugins separate from currently installed ones.

## Current position

The current native DEV feedback pass is ready for manual review. The latest installed
build is **0.2.0-dev.20261028.2**; see [review and roadmap checklist](native-dev-review.md).
M465–M467 supersede the earlier missing project join/auth/lifecycle statements:
existing-project additions, empty networks, live fleet projection, code/log access,
clean member stop/start, matching Theme/DMs sign-in and key input are implemented.
Live `demo-zod` was preserved; added `demo-nec` passed clean restart and both peers
passed acknowledged Gall messaging. Expanded engine/anchor recovery and broader
release gates remain unfinished. No new VM batches or evidence copies were made.

The table below retains earlier qualification distinctions; the linked current
review is the concise implemented/open checklist for this handoff.

| Plan area | Demonstrated so far | Still to finish |
| --- | --- | --- |
| Imports | Identity-based layout, guided import, cross-filesystem copy, interrupted-copy restoration and explicit Start/Stop through the real CLI/UI in disposable VMs | Remaining failure/upgrade cases and full native desktop qualification |
| CLI and skills | Operational/developer skills, public project commands, capability checks, saved history and explicit recovery receipts | Complete command walkthroughs and release integration |
| Fake projects | Two-ship messaging, persistent state/subscriptions, duplicate-project isolation, private queued lifecycle/recovery; public initial/member lifecycle passes with application recovery | Mutating project UI and release qualification; corrected full-run/service-restart acceptance passes in M243 |
| Theme/DMs connections | Separate-session approval and direct reads/SSE; copied native plugins preserve sessions and an unresolved DM through two package transitions and third-desktop loss cleanup (M329–M331) | Broader hosted/installed and real-backend updates, locked-keyring/reboot paths |
| Backend setup and compatibility | Source-verified plugin-owned install/readiness guidance from Urbit, Talon, Omart and Lattice | Full native desktop/public publisher and broader update-order support; standalone Tend flow/runtime/private release checks pass M252 |
| Developer kit | Separate plugin IDs/state/hooks; private collection activation and recovery; one reusable recovery family passes608 core checks across three later sessions, plus18 package and2 historical CLI checks; official keyed native and Docker peers pass state/subscription recovery (M239/M346) | Claimed/unbound recovery, public/installed profiles, runner private-devnet integration and supported runtime delivery |

Public member lifecycle passes independently (M220). VM41's public interrupted
recovery and application subgates also pass an independent audit (M227): six
requests/six lookups, three lost replies,16 writes and eight subscriptions across
four cycles with the original sessions. Its overall result remains failed at a
wrong-actor test-helper check; a copy of its actual database proves the narrow
fixture correction. Fresh VM42 passes that correction and three actual owner-service restart cycles
(M243). Mutating UI remains separate.

Private chain/key validation passes (M216–M218). An experimental runtime correction
passes fresh keyed shutdown/reopen (M224), then actual key rotation and Jael
persistence across restart with independent chain/pier audit (M229). Two fresh
keyed peers now pass messaging and receiver restart with independent chain/pier
audit, first with the experimental build (M233), then the official binary with
explicit Lens initial shutdown (M236). The official runtime's fresh-key shutdown
failure remains (M219/M222); correct Lens shutdown and same-pier reopen now pass
with that unchanged official binary (M235). The experimental candidate is not
installed or released. Official keyed Docker peers pass persistent state and
subscription recovery (M346); reusable source-entry execution and independent
stopped-disk auditing also pass (M352). Schema41 public reservation/inventory is
implemented and source-tested (M353/M354). Whole-gateway reopen/reconnection passes M367. Schema43 now retains private workspace/gateway operations and separate recovery
observations (M378); public runner runtime dispatch and simultaneous
private projects remain open. Direct HTTP publication and retained client recovery pass the independent
stopped-disk audit (M372). First-start publication independently passes VM15/audit30 (M384).

The checkpoint entries below are chronological evidence. Newer IDs explicitly
supersede older running/pending statements; those older statements are retained
for provenance, not current task status.

## Latest checkpoint — 2026-10-26

- **M447:** user pause; schema46 is integrated but full validation/runtime remain
  incomplete after a bounded test-harness timeout. Storage cleanup removes382
  superseded generated files (378.94→30.38 GiB). Prior full-bundle retention is
  superseded by bounded retention; old pruned archives are not runnable. See the
  [current review](development-review-2026-10-26.md).

- Ordinary run11 remains interrupted at started; audit/disk retention complete. The staged capture adapter passes156 race checks; schema46 admission/integration follows ([M446](diagnostics/2026-10-26-member-capture-adapter.md)).

- Staged capture persistence passes47 selected race checks and genuine original45 downgrade refusal; concrete adapter and integration follow ([M445](diagnostics/2026-10-26-member-capture-store.md)).

- A separate capture-before-observation prototype passes237 race checks; persistence/migration and adapter integration remain, while the ordinary M443 VM continues ([M444](diagnostics/2026-10-26-member-capture-model.md)).

- Combined inventories and overlapping final verification pass791 race checks with unchanged deadlines; a fresh ordinary VM is now testing the result ([M443](diagnostics/2026-10-26-member-observer-inventory.md)).

- Diagnostic VM trace proves capture succeeds before the gateway observation deadline expires. Disk retained; redundant observation work is the next target, with deadlines unchanged ([M442](diagnostics/2026-10-26-member-observation-deadline.md)).

- Source review identifies a capture-to-running persistence gap: a later observation failure can lose the in-memory pier binding. The next format must save capture first and qualify recovery; no behavior changed ([M441](diagnostics/2026-10-26-private-member-capture-gap.md)).

- A fresh diagnostic VM is running with fixed observation-return locations and unchanged checks.579 affected race/20 audit checks pass; its runtime/audit result remains pending ([M440](diagnostics/2026-10-26-managed-member-observation-trace.md)).

- The fixed VM reaches acknowledged member Start, then interrupts during observation. Audits preserve all history and the intact pill; `%term` prevents treating the later parse error as root cause. Disk retained; precise fresh-run diagnostics follow ([M439](diagnostics/2026-10-26-managed-member-started-diagnosis.md)).

- Concrete first-member HTTP observation now composes retained pier/key/pill and full gateway/process checks. 64 targeted/225 affected race checks pass; persistence and real readiness runtime remain ([M438](diagnostics/2026-10-26-private-member-http-driver.md)).

- A read-only first-member HTTP readiness model binds current observations to the captured launch and returns no partial proof. 54 targeted/159 affected race checks pass; concrete observation and persistence remain ([M437](diagnostics/2026-10-26-private-member-http-readiness.md)).

- First-member VM reached the queue but exposed a ready-client reuse mismatch before launch effects. The failed disk is retained; the reproduced source fix passes 198 race checks. A fresh VM remains required ([M436](diagnostics/2026-10-26-managed-member-ready-client.md)).

- Original schema45 is frozen and reproducible, with durable birth/interruption history preserved across three opens. Successor migration/refusal await the next format ([M435](diagnostics/2026-10-26-original45-member-birth.md)).

- A first-member VM fixture and schema45 disk auditor are prepared and source-tested; the fresh isolated VM is running, with runtime evidence pending ([M434](diagnostics/2026-10-26-managed-member-boot-fixture.md)).

- The first-member queue now uses the shared work/update fence and a read-only concrete session factory, with replay before backend lookup. 95 targeted/213 affected race checks pass. Readiness, second-member admission and actual VM lifecycle remain ([M433](diagnostics/2026-10-26-private-member-queue.md)).

- Schema45 now retains first-boot capture for new member launches while preserving original44 history exactly. 1,216 affected race checks and original44 migration/refusal pass. Scheduler, readiness and actual member runtime remain ([M432](diagnostics/2026-10-26-private-member-durable-birth.md)).

- The original44 writer now reproduces completed/interrupted member history exactly in two runs; 23 selected race checks preserve its rows, replay and sidecars. Durable birth/readiness migration is next ([M431](diagnostics/2026-10-26-original44-member-launch.md)).

- The concrete first-member driver now composes artifact/key preflight, exact Create/Start, held birth capture and full member-aware observation. 90 new/1,104 affected race checks pass. Durable capture and managed runtime qualification remain ([M430](diagnostics/2026-10-26-private-member-driver.md)).

- First-boot metadata capture now holds the empty mount across Start and brackets pier/.urb identities with exact process observations: 49 new/1,014 affected race checks pass. Durable binding and actual runtime remain ([M429](diagnostics/2026-10-26-private-member-birth.md)).

- The first private member launch now has schema44 durable storage, actor-scoped replay and interruption-only restart handling. All 536 affected race checks pass; the original43 writer reproduces its fixture and refuses populated successor state unchanged. Concrete runtime, readiness and second-member admission remain ([M428](diagnostics/2026-10-26-private-member-store.md)).

- Initial private member launch now records every dispatch and known returned object, and refuses repeat boot after uncertainty:91 new/965 affected race checks pass. Persistence and actual managed member runtime remain ([M427](diagnostics/2026-10-26-private-member-launch.md)).

- Fresh VM10 and independent audit pass installation, gateway recovery and actual two-key/four-negative proof. Disk/base and original history are preserved; no ships booted, all units inactive, evidence retained. Member lifecycle and M425 runtime remain next ([M426](diagnostics/2026-10-26-managed-member-key-proof.md)).

- The private member-aware gateway observer composes held pier identities, complete member ownership and fresh running-member kernel namespaces:35 new/874 affected race checks, vet and diff pass. Durable lifecycle and real member runtime remain ([M425](diagnostics/2026-10-26-private-member-observer.md)).

- Member container/mount ownership passes72 new/839 affected race checks, vet and diff. Initial gateway checks remain unchanged; complete observer and lifecycle wiring remain. Frozen VM10 is testing M423 installation and M420 key proof ([M424](diagnostics/2026-10-26-private-member-engine.md)).

- The reproduced management-start output-drain defect is corrected in source:400 installer race checks, vet and diff pass. Probe/output and activation checks remain strict. A fresh VM must qualify installation and then member key proof ([M423](diagnostics/2026-10-26-management-command-drain.md)).

- VM09’s cleanly stopped installation failure is independently diagnosed as successful systemctl exit with incomplete output drain. Disk retained; key proof was not reached. A narrow command-acknowledgement fix and fresh VM follow ([M422](diagnostics/2026-10-26-managed-key-proof-install-failure.md)).

- Member-aware gateway filesystem verification passes633 affected race checks with retained pier identities and opaque live payload. Engine integration remains. Separately, VM09 stopped cleanly at host installation before key proof; investigation is ongoing ([M421](diagnostics/2026-10-26-private-member-gateway-state.md)).

- Guest-only key proof and mandatory independent audit are integrated:170 affected race checks, three ordinary-reader checks, vet and host refusals pass. Fresh real-key VM follows; no member launch is qualified ([M420](diagnostics/2026-10-26-private-member-key-fixture.md)).

- Member key-proof source integrates with174 combined Go race checks,46 new/47 existing JS checks, pinned ABI decoding and host refusal. Real evaluator/chain and full member admission remain; fresh VM qualification follows ([M419](diagnostics/2026-10-26-private-member-key-proof.md)).

- Retained member pier/.urb and stopped-state metadata verification is integrated:36 targeted/87 combined race checks and vet pass. Process-exit proof, lifecycle and runtime remain separate ([M418](diagnostics/2026-10-26-private-member-pier.md)).

- Ordinary-source gateway initialization/lost-reply recovery and independent audit pass, preserving the original history across two manager reopens. Disk retained; all units terminal. Member lifecycle and gateway Stop/restart remain ([M417](diagnostics/2026-10-26-managed-gateway-recovery.md)).

- Pinned bootstrap-pill verification is integrated:32 targeted/109 combined member race checks and vet pass with synthetic files. Held objects detect changes after hashing; runtime/staging remain separate ([M416](diagnostics/2026-10-26-private-member-pill.md)).

- Diagnostic managed gateway initialization/lost-reply recovery passes, with independent immutable audit and two manager reopens. Ordinary-source VM08 follows; member launches and whole-gateway restart remain separate ([M415](diagnostics/2026-10-26-managed-gateway-recovery-diagnostic.md)).

- Member bootstrap-key verification is integrated:34 targeted race checks/vet pass using synthetic files. Exact accepted bytes and held objects are checked; digest acquisition, key/chain proof and runtime admission remain ([M414](diagnostics/2026-10-26-private-member-key.md)).

- Held member-mount verification is integrated:17 targeted race checks/vet pass. Replaced mounts and partial first boots refuse without modifying data. Resume/input/gateway checks remain; diagnostic VM07 runs independently ([M413](diagnostics/2026-10-26-private-member-mount.md)).

- Private member container binding is integrated:77 targeted race checks/vet pass. Boot/resume mounts and exact gateway ownership are checked; managed member launch remains pending ([M412](diagnostics/2026-10-26-private-member-container.md)).

- Ordinary VM06 fails before gateway dispatch; its clean, unchanged disk is retained. M410 was not exercised. Logging-only preflight diagnostics pass355 checks; a fresh diagnostic VM follows ([M411](diagnostics/2026-10-26-managed-gateway-preflight-failure.md)).

- RPC rejection is reproduced as a code-hash encoding mismatch. The integrated correction preserves foundation pins and passes47 JS/355 Go race checks plus vet. Fresh ordinary-source readiness/recovery VM is next ([M410](diagnostics/2026-10-26-private-gateway-code-hash.md)).

- Diagnostic VM/audit locate repeated fast failures in the live RPC probe after successful runtime/state checks. Disk retained, units terminal; exact RPC substage and ordinary runtime qualification remain next ([M409](diagnostics/2026-10-26-managed-gateway-rpc-diagnosis.md)).

- Original43 initialization/recovery history now has a frozen, twice-reproduced synthetic fixture and passing repeated-reopen/actor-replay race checks. Schema remains43; successor migration and managed runtime remain separate gates ([M408](diagnostics/2026-10-26-original43-private-lifecycle.md)).

- Fresh M405 VM still fails managed readiness although the initializer publishes ready. Disk/base are unchanged and retained; recovery never ran. A tested logging-only overlay will locate the failing stage in a new VM ([M407](diagnostics/2026-10-26-managed-gateway-observation-followup.md)).

- Private keyed-member command construction is integrated with 21 race checks and parity against the qualified fixture. Managed member lifecycle remains next; the separate current gateway VM is still running ([M406](diagnostics/2026-10-26-private-member-command.md)).

- The initializer reaches GATEWAY_READY, but managed observation hits the fixture deadline. Read-only timings identify costly input preparation; the observer now separates it from the unchanged25s freshness window, passing355 race checks. Fresh runtime qualification follows ([M404](diagnostics/2026-10-26-managed-gateway-readiness-timing.md), [M405](diagnostics/2026-10-26-private-gateway-observation-window.md)).

## Latest checkpoint — 2026-10-25

- Corrected run02 ended unexpectedly during initialization with no final report. Read-only audit04 refused uncheckpointed SQLite and preserved the disk. Next: fresh supervised run using the same corrected appliance ([M403](diagnostics/2026-10-25-managed-private-launch-interruption.md)).

- Docker tmpfs inspection correction reproduces the original failure and passes354 gateway race checks plus vet. A new current-source VM will test readiness/recovery; the failed disk remains privately retained ([M402](diagnostics/2026-10-25-private-gateway-tmpfs-inspection.md)).

- Current managed gateway VM24 reaches Start but fails readiness; three read-only audits preserve its disk. Docker tmpfs inspection mismatch identified; regression/fix and a fresh VM follow. No recovery attempt yet ([M401](diagnostics/2026-10-25-managed-private-gateway-first-runtime.md)).

- First-use private workspace parent provisioning is integrated:452 privatechain and95 workspace race checks pass. Retained claims refuse missing/replaced parents; metadata retry after a failed receipt preserves identity. Delivery and fresh current-source VM qualification remain ([M400](diagnostics/2026-10-25-private-workspace-parent-provision.md)).

- Real private-gateway backend/client composition is integrated:353 gateway race checks,83 queue checks and runner compilation pass. Actual generation07 preflight uses an inert engine; storage provisioning, delivery and fresh VM qualification remain ([M399](diagnostics/2026-10-25-private-gateway-managed-session.md)).

- New sealed generation07 independently verifies, and bounded first-start readiness is integrated:312 gateway race checks pass. Actual generation/workspace preflight uses an inert engine. Managed session wiring and fresh no-NIC runtime qualification remain ([M397](diagnostics/2026-10-25-private-gateway-input-generation-07.md), [M398](diagnostics/2026-10-25-private-gateway-first-start-wait.md)).

- Pre-start initializer admission now matches readiness: actual generation06 is rejected before journal dispatch or engine access; 287 gateway race checks pass ([M396](diagnostics/2026-10-25-private-gateway-preflight-pin.md)). New sealed inputs and full runtime integration remain next.

- Complete gateway observer source integrated: 434 privatechain and 285 private-gateway race checks pass; actual generation06 is refused before engine access because its initializer is obsolete ([M395](diagnostics/2026-10-25-private-gateway-observer.md)). Enforce this pin before startup next, then wire lifecycle and qualify fresh inputs in a no-NIC VM.

- Initialized gateway metadata verifier integrated: 27 focused race checks/vet and all 422 privatechain race checks pass ([M394](diagnostics/2026-10-25-private-gateway-state-verifier.md)). It reads no key/database contents and grants no durability; complete live observer composition remains next.

- External gateway RPC probe integrated: 46 JS checks, 40 focused Go race checks/vet and 251 integrated private-gateway race checks pass ([M393](diagnostics/2026-10-25-private-gateway-rpc-probe.md)). Complete engine/filesystem observation and fresh VM qualification remain; no live transport was used.

- Gateway startup checkpoint integrated: close/sync/reopen/reobserve/sync, private new files and durable exclusive readiness publication pass 114 source checks ([M392](diagnostics/2026-10-25-private-gateway-startup-checkpoint.md)). Fresh VM qualification and the external read-only observer remain next.

- Sealed-input startup copy defect fixed: original0400 copy reproduces EACCES; fresh private writable copy passes with the actual unkeyed foundation, and58 source checks pass ([M391](diagnostics/2026-10-25-private-gateway-fresh-copy.md)). New sealed generation and runtime qualification are required; live readiness/durability remains next.

- Concrete private gateway create/inspect/Start resource methods integrate:45 focused race checks/vet, real generation06 constructor qualification with an inert engine, and211 integrated private-gateway checks pass ([M390](diagnostics/2026-10-25-private-gateway-resources.md)). Fresh readiness observation and managed backend wiring are next; no live Docker/public dispatch yet.

- Gateway startup metadata now has a journal-gated filesystem helper:31 focused race checks/vet and395 complete privatechain checks pass ([M389](diagnostics/2026-10-25-private-gateway-bootstrap.md)). Lost commits and partial writes preserve intent without repeating startup. Concrete Docker adapter/observation remains next.

- Held runtime-input verification integrates: identical-byte root replacement refuses;65 focused race checks/vet, two complete generation06 scans,364 privatechain checks and3 collector checks pass ([M388](diagnostics/2026-10-25-private-input-verifier.md)). Concrete startup adapter remains next.

- Read-only completed-workspace verification integrates; same-shaped directory replacements refuse.62 focused race checks/vet, identity negative control and354 full privatechain package checks pass ([M387](diagnostics/2026-10-25-private-workspace-verifier.md)). Concrete runtime handles/dispatch remain.

- Complete current gateway input generation06 is sealed and independently inspected through the built offline collector:12,352 files, no runtime authority, older generation unchanged ([M386](diagnostics/2026-10-25-private-gateway-input-generation.md)). Concrete backend and delivery remain.

- Gateway container and exact loopback publication checks integrate with168 race checks plus vet and166 overlapping actual-checkout checks ([M385](diagnostics/2026-10-25-private-gateway-container-contract.md)). This remains source evidence; concrete backend/public lifecycle and new input-generation inspection remain.

- First-start publication VM15 and independent audit30 pass: original sessions, two new writes, four subscription reads, three scries, closed channels and refused cross-cookie/admin/RPC/IPv6 access ([M384](diagnostics/2026-10-25-private-initial-publication-runtime.md)). Both scopes stopped cleanly; disk privately retained unchanged. Supersedes M380–M383 runtime-pending statements; concrete runner/public lifecycle and concurrent projects remain.

- Private gateway network ownership/allocation checks integrate with76 race checks plus vet and74 overlapping actual-checkout checks ([M383](diagnostics/2026-10-25-private-gateway-network-contract.md)). This is source evidence; VM15/audit30 and concrete runtime/public dispatch remain pending.

- Older workspace queue now replays retained receipts without a backend and rejects typed-nil/rebound authority; original-source negative control and91 race checks/vet pass, followed by28 integrated checks ([M382](diagnostics/2026-10-25-private-workspace-queue-parity.md)). VM15/audit30 continues separately.

- Full unchanged43 baseline passes1,839 race checks; integrated lifecycle queues then pass132 affected race checks and vet ([M381](diagnostics/2026-10-25-private-lifecycle-integrated.md)). This supersedes M378/M379 pending statements. Fresh initial-publication VM15/audit30 continues under isolation.

- First-start HTTP publication fixture and mandatory independent reader pass4 JavaScript/74 Python/198 Go race checks plus vet and host refusal ([M380](diagnostics/2026-10-25-private-initial-publication-source.md)). Runtime remains pending; storage reserve is preserved through verified appliance16 relocation.

- Internal lifecycle queues pass63 staged race checks and vet, including actual lost-layout recovery through an observer-only filesystem factory ([M379](diagnostics/2026-10-25-private-lifecycle-queues.md)). Queue integration awaits the unchanged full43 baseline; concrete gateway/runtime and public dispatch remain.

- Schema43 migration, startup/global validation and separately attributed lifecycle receipts pass280 focused race checks, vet, original42 reproduction and real successor downgrade refusal ([M378](diagnostics/2026-10-25-private-lifecycle-schema43.md)). Full managed/privatechain regression is running; no public runtime dispatch. This supersedes M377’s staged-only status.

- Staged gateway/workspace persistence and separately attributed recovery pass161 focused race checks plus vet ([M377](diagnostics/2026-10-25-private-lifecycle-store-proposal.md)). The candidate files remain outside the checkout; combined schema43 migration, startup/global validation and original42 refusal are next.

- Original42 reproduces completed and interrupted workspace fixtures twice; repeated reopen/replay and21 affected race checks plus vet pass ([M376](diagnostics/2026-10-25-original42-private-workspaces.md)). This establishes the baseline for combined private lifecycle persistence; no schema/runtime change.

- First gateway startup and observer-only uncertainty recovery pass327 privatechain race checks plus vet ([M375](diagnostics/2026-10-25-private-gateway-start-model.md)). This is the source model; combined persistence and the concrete initial/publication adapter remain. Schema42 is unchanged; appliance16 still awaits memory admission.

- Internal private workspace scheduling and real temporary-directory integration pass focused race/vet checks ([M374](diagnostics/2026-10-25-private-workspace-queue.md)). The pre-queue schema42 full managed regression passes1416 checks; public runtime dispatch and durable observation recovery remain open. Appliance16 still awaits memory admission.

- Concurrent private-project source/auditor checks pass; appliance16 is verified and unbooted ([M373](diagnostics/2026-10-25-private-project-isolation-source.md)). Runtime waits for the32GiB memory-admission floor; no limits or isolation constraints were relaxed. Full schema42 regression remains running.

- Direct private HTTP publication and whole-gateway client recovery now pass independent audit29 on VM14 ([M372](diagnostics/2026-10-25-private-http-publication.md)). The verifier now counts all four event acknowledgements; the runtime disk was unchanged. All VM/audit scopes stopped and evidence retained. Next: simultaneous private projects and durable public runner lifecycle.

- Schema42 workspace journals pass focused race/API/replay and actual original41 downgrade-refusal checks ([M371](diagnostics/2026-10-25-private-workspace-store.md)). Full managed regression is running; public queues and durable observation recovery follow.

- Private workspace creation and interrupted-layout observation now pass source/race and real temporary-filesystem tests ([M370](diagnostics/2026-10-25-private-workspace.md)). Parent/root identities and partial trees are retained. Persistence/runner scheduling follow; schema41 and the frozen publication VM remain unchanged.

- Original41 source and populated private reservations now reproduce twice and pass repeated reopen/replay with race checks ([M369](diagnostics/2026-10-25-original41-private-reservations.md)). This prepares the next lifecycle migration; current schema stays41. VM14 publication remains pending.

- Direct private HTTP publication passes source checks and is running in fresh VM14, with a shutdown-gated independent audit ([M368](diagnostics/2026-10-25-private-http-publication-source.md)). Runtime publication remains pending; M367’s whole-gateway recovery pass is retained.

- Whole private-chain gateway and peer recovery now passes its independent VM audit ([M367](diagnostics/2026-10-25-private-gateway-peer-recovery.md)): original sessions, queued subscription, exact piers and chain/head survive; no writes replay. All scopes stopped. Next: isolated HTTP publication, then concurrent projects and durable runner dispatch.

- Reusable inert input preparation reproduces M362’s exact manifest and passes the independent reader ([M366](diagnostics/2026-10-25-runtime-input-preparation.md)). Runtime delivery/dispatch remain separate; VM12 and its gated disk audit continue.

- Theme/DMs credential stores pass real locked-keyring refusal and original-session recovery through a private daemon restart ([M365](diagnostics/2026-10-25-plugin-keyring-lock.md)). UI unlock/reboot remain separate. Namespace-corrected VM12 is running with frozen inputs.

- VM11 reaches retained-gateway readiness but peer recovery refuses a repeated historical namespace number ([M364](diagnostics/2026-10-25-private-gateway-peer11-refusal.md)). The source correction passes117 Go/54 Python checks; a fresh isolated runtime follows. Original peer recovery passes, disk privately retained, all scopes inactive.

- The unchanged input reader passes an actual same-filesystem bind-mount refusal between two clean inspections ([M363](diagnostics/2026-10-25-private-runtime-input-mounts.md)); all three isolated scopes are inactive. VM11 and its gated read-only audit remain pending.

- Local input verification passes against12,352 actual files, with nine built collector cases and468 affected race checks ([M362](diagnostics/2026-10-25-private-runtime-inputs.md)). Skills/help expose the boundary; schema41 stays reservation-only. VM11 continues independently with frozen inputs.

- Private retention resolves temporary quota admission; fresh corrected gateway/peer VM11 is running ([M361](diagnostics/2026-10-25-private-gateway-peer-preparation11.md)). Runtime recovery and independent audit remain pending; production and installed state remain excluded.

- VM10 independently passes the original peer/state case but refuses gateway resume before creation on a0755/0700 chain-copy mismatch ([M360](diagnostics/2026-10-25-private-gateway-resume10-refusal.md)). Strict guard retained; fresh-only initialization corrected. Disk privately retained, all scopes inactive; corrected fresh preparation follows.

- Exact predecessor final-head binding passes39 JS/110 Go checks ([M359](diagnostics/2026-10-25-private-gateway-original-head.md)). This narrows the older height-only precursor claim; VM10 remains frozen and the stronger peer runtime is pending.

- Whole-gateway client/peer recovery source passes55 Python and109 Go audit checks ([M358](diagnostics/2026-10-25-private-gateway-peer-source.md)). Its preparation was refused before output creation by quota; VM10 continues with unchanged sealed inputs.

- Independent resume auditing passes69 Go checks,50 Python checks,vet and host refusal ([M357](diagnostics/2026-10-25-private-gateway-resume-auditor.md)); VM10 and its actual stopped-disk audit remain pending.

- Ownership-bound gateway resume preparation passes49 inert Python checks; fresh isolated VM10 is running ([M356](diagnostics/2026-10-25-private-gateway-resume-preparation.md)). Runtime and independent audit remain pending.

- Separate retained gateway source passes36 inert checks and host refusal ([M355](diagnostics/2026-10-25-private-gateway-resume-source.md)). The existing initializer must not be reused for restart; actual Ganache reopening/VM ownership checks remain next.

- Private-chain registration/inventory API and CLI pass capability, replay, observer, refusal and regression checks ([M354](diagnostics/2026-10-25-private-chain-public-reservations.md)); schema41 remains reservation-only. M353 full-store regression also passes. Verified runtime inputs/lifecycle follow.

- Private-chain persistence and actual predecessor refusal pass synthetic source gates ([M353](diagnostics/2026-10-25-private-chain-project-store.md)); full managed regression is running. Public reservation API and lifecycle follow.

- Repository preparation, Docker recovery and the new read-only audit now pass end to end with fresh project/keys and unchanged disk hashes ([M352](diagnostics/2026-10-25-private-docker-source-runtime.md)). Private runner lifecycle, publication and simultaneous-project isolation remain next.

- Operator/developer skills now route M349’s offline private-chain review separately from fake registration, with validator/reference/help checks passing ([M351](diagnostics/2026-10-25-private-chain-skill.md)). VM09 and its queued read-only audit continue independently.

- Reusable stopped-disk auditing passes8 inert admission checks,40 Python regressions and25 reader tests/subcases, with fixed metadata output ([M350](diagnostics/2026-10-25-private-docker-auditor-source.md)). Its runtime audit follows source-entry VM09’s shutdown.

- Distinct offline private-chain planning now passes188 top-level regressions/1,392 including subcases and a built CLI walkthrough ([M349](diagnostics/2026-10-25-private-chain-plan.md)). It changes no saved schema and cannot enter fake registration. Persistent runner lifecycle remains next; source-entry VM09 continues.

- Reusable private Docker preparation passes32 inert Python checks and18 Go tests/subcases, sealing exactly M346’s expanded inputs ([M348](diagnostics/2026-10-25-private-docker-preparation.md)). Source-entry VM09 is running; public runner devnet support remains separate.

- The shared VM launcher now carries M346’s qualified private-chain boundary, with11 inert regression checks ([M347](diagnostics/2026-10-25-private-chain-vm-launcher.md)). Reusable preparation and full source-entry execution remain next.

- Official keyed Docker peers now pass state/subscription recovery with independently audited receipts and clean shutdowns ([M346](diagnostics/2026-10-25-keyed-docker-recovery.md)). Reusable launcher and runner/devnet integration remain separate.

- Exact active Docker fixture sources are integrated with21 inert host/lifecycle checks; the bounded metadata auditor passes14 checks ([M345](diagnostics/2026-10-25-keyed-docker-entry.md)). Fresh VM08 is booting its first keyed ship; runtime result remains pending.

- Actual Docker gateway loads the full native runtime and reaches public galaxy parsing; a read-only diagnostic reproduces the30s limit and succeeds in71.45s with120s ([M344](diagnostics/2026-10-25-keyed-docker-evaluator-budget.md)). Fresh corrected gateway/ship qualification is running; key-bearing failed state remains private.

- Closed the native-addon library packaging gap with an exact-runtime probe that forbids host-library fallback; the baseline fails and the complete closure passes ([M343](diagnostics/2026-10-25-keyed-docker-node-closure.md)). Fresh Docker ship qualification remains next.

- Fresh Docker bootstrap and gateway mounts pass; fixed the diagnosed glibc/musl addon selection in the next gateway draft, with native module/persistence and17 helper tests passing ([M342](diagnostics/2026-10-25-keyed-docker-gateway-libc.md)). Ship execution remains pending.

- Fixed a reproduced native-package verification timeout by streaming authenticated archive payloads ([M341](diagnostics/2026-10-25-package-payload-stream.md)); full source suite and read-only VM verification pass. Fresh Docker peer qualification continues.

- Second Docker-chain run admits the pin, then refuses the image’s automatic anonymous volume before Start ([M340](diagnostics/2026-10-25-keyed-docker-gateway-mount.md)). Explicit read-only gateway mount is staged for a fresh run; both zero-ship failed VMs retain source/audits and have their machine disks retired.

- First actual Docker-chain VM refuses before gateway Start on the image-ID representation; exact-pin fix passes15 inert checks ([M339](diagnostics/2026-10-25-keyed-docker-image-pin.md)). Failed runtime and read-only disk audit are retained; fresh runtime rerun is next.

- Private keyed Docker lifecycle source passes14 inert checks ([M338](diagnostics/2026-10-25-keyed-docker-lifecycle-source.md)); actual chain/gateway/container integration is next. Profile recovery edge cases remain tracked separately.

- Three repeated actual native sessions pass with identical packages and45 controller interruptions ([M337](diagnostics/2026-10-25-native-repeated-sessions.md)). Claimed/unpublished and pending/unbound loss, public profiles and broader roadmap gates remain.

- Artifact7 and public preflight pass18 package and2 historical CLI checks ([M336](diagnostics/2026-10-25-reusable-runtime-packages.md)); repeated actual native plugins remain next.

- One recovery family survives three further sessions with608 full tests and nine actual guards ([M335](diagnostics/2026-10-25-reusable-runtime-core.md)); immutable packaging and native plugins remain next.

- Reusable intent3 and old-writer exclusion pass20 private tests ([M334](diagnostics/2026-10-25-reusable-session-ownership.md)); repeated authority and application startup remain next.

- Reference-only checkpoint3 and bounded ancestry pass42 focused checks ([M333](diagnostics/2026-10-25-reusable-checkpoint-envelope.md)); reusable intent/admission remains next.

- Bounded iterative legacy ancestry passes19 focused private tests ([M332](diagnostics/2026-10-25-bounded-checkpoint-ancestry.md)); reusable recovery formats and repeated same-package sessions remain next.

- Authority3 transport cleanup passes532 tests and third-compositor loss with43 injected controller kills ([M331](diagnostics/2026-10-25-checkpoint2-cleanup.md)). Next: reusable bounded lineage.

- Authority3 retained inspection passes512 tests and five native lifecycle snapshots ([M330](diagnostics/2026-10-25-checkpoint2-inspection.md)). Next: bounded reusable repeated-session recovery.

- Actual native checkpoint2 activation passes with package5-to6,41 injected controller kills and preserved sessions/subscriptions/pending DM ([M329](diagnostics/2026-10-25-native-checkpoint2-activation.md)). Next: reusable repeated-session recovery and public profile integration.

- Artifact6 and authority1/2/3 compatibility pass493 tests ([M328](diagnostics/2026-10-25-continuation-runtime-bundles.md)). Next: actual native activation from checkpoint2.

- Authority3 publication/admission from checkpoint2 passes475 tests with owned guards ([M327](diagnostics/2026-10-25-checkpoint2-authority.md)). Next: compatible immutable bundles and actual native plugin activation.

- Checkpoint2 desktop ownership passes454 tests and actual native cohort verification through three controller kills ([M326](diagnostics/2026-10-25-linked-session-ownership.md)). Next: authority/guard publication and application admission.

- Linked checkpoint passes430 tests and native publication/second-runtime retirement ([M325](diagnostics/2026-10-25-linked-desktop-checkpoint.md)). Next: explicit successor ownership/admission from checkpoint2.

- Linked desktop cleanup passes410 tests and actual second-compositor-loss cleanup with sessions/pending DM preserved ([M324](diagnostics/2026-10-25-linked-desktop-cleanup.md)). Next: bounded durable checkpoint and repeated continuation.

- Linked authority inspection passes391 tests and five actual native snapshots through repeated compositor loss ([M323](diagnostics/2026-10-25-linked-authority-inspection.md)). Next: explicit second-session cleanup and durable checkpoint/admission.

- Native artifact4-to5 successor continuation passes all seven new controller interruptions while preserving sessions and pending DM ([M322](diagnostics/2026-10-25-native-successor-upgrade.md)). Next: review repeated desktop-session recovery and public profile integration.

- Actual native successor activation survives seven new controller kills; sessions, pending DM and canonical history persist ([M321](diagnostics/2026-10-25-native-successor-activation.md)). Next: qualify different immutable generations and review repeated-session/public profile integration.

- Complete successor bundles and authority1/2 preflight pass373 tests ([M320](diagnostics/2026-10-25-successor-runtime-bundles.md)). Next: native successor application activation and subscription restoration through the canonical transition.

- Linked authority2 and canonical replacement admission pass355 tests, four publication kills and three real guards ([M319](diagnostics/2026-10-25-linked-successor-authority.md)). Next: complete immutable generation capability declarations and native successor application activation.

- Actual Theme/DMs successor cohort checks pass three controller kills without HTTP or application activation ([M318](diagnostics/2026-10-25-native-successor-ownership.md)). Next: versioned successor descriptor/admission and compatible immutable artifacts.

- Linked successor ownership passes336 tests and three actual controller kills per final run ([M317](diagnostics/2026-10-25-desktop-successor-ownership.md)). Next: native Theme/DMs cohort checks, then versioned successor admission without resetting canonical history.

- Verified desktop evidence survives runtime-path retirement:314 tests and actual private-plugin crash checks ([M316](diagnostics/2026-10-25-desktop-runtime-checkpoint.md)). Next: linked successor ownership and current-cohort validation before new-session admission.

- Explicit old-desktop transport cleanup passes297 tests and native crash/rejected-deletion checks ([M315](diagnostics/2026-10-25-desktop-transport-cleanup.md)). Next: durable linked successor/bootstrap ownership, broader interrupted states and supported public profiles.

- Retained-authority inspection passes280 tests and native live/dead/replaced-compositor and reservation3 checks ([M314](diagnostics/2026-10-25-retained-authority-inspection.md)). Next: explicit old-cohort transport cleanup, then durable linked cross-session bootstrap.

- Physical compositor replacement is characterized and safely fenced ([M313](diagnostics/2026-10-25-compositor-loss.md)); next implement retained-authority inspection and explicit cross-session cleanup/continuation.

- Interrupted authority setup passes260 tests and native reservation2/3 recovery ([M312](diagnostics/2026-10-25-authority-provisioning.md)). Next: cross-session bootstrap and supported public profile activation/upgrade.

- Protocol4 preparation and read-only compatibility preflight pass233 tests and native mixed-version/recovery checks ([M311](diagnostics/2026-10-25-runtime-protocol-preflight.md)). Next: interrupted provisioning and persistent bootstrap; public activation/upgrade remains separate.

- Early/partial initial recovery passes five new native cases plus full-set/adopted regressions; original state and ownership retained ([M310](diagnostics/2026-10-25-initial-startup-cohorts.md)). Next: compatible generation preflight and persistent bootstrap; no user action required.

- Post-adoption/pre-binding owner loss now recovers;217 tests, seven-crash native roundtrip and pre-adoption regression pass ([M309](diagnostics/2026-10-25-closed-initial-binding.md)). Continuing early startup and persistent bootstrap; no user action required.

- Pre-adoption initial owner loss now recovers with retained original cleanup/history;205 tests and actual five-crash recovery pass ([M308](diagnostics/2026-10-25-initial-owner-recovery.md)). Next: remaining startup boundaries and persistent bootstrap; no user action required.

- Initial activation survives four controller crashes with one original reservation and no duplicate plugins ([M307](diagnostics/2026-10-25-initial-activation-replay.md)). Owner/compositor loss and persistent bootstrap remain; no user action required.

- Theme/all-member candidate cleanup recovery and DMs regression pass with stable appearance and state restored ([M306](diagnostics/2026-10-25-candidate-startup-cohorts.md)). Next: initial activation and persistent bootstrap recovery; no user action required.

- Failed DMs candidate cleanup now recovers with original-owner proof and retained application state;179 tests and full shared regression pass ([M305](diagnostics/2026-10-25-candidate-cleanup-recovery.md)). Next: other candidate cohorts and startup recovery. No user action is required.

- Native partial rollback passes before issuance and after DMs-only startup, retaining all state and original closure evidence ([M304](diagnostics/2026-10-25-native-partial-rollback.md)). Failed cleanup and persistent startup recovery remain; no user action is required.

- Shared-authority handoff and connected recovery pass158 tests and five controller crashes; clean guard shutdown is fixed ([M303](diagnostics/2026-10-25-authority-handoff.md)). Next: native partial-candidate rollback and persistent startup recovery. No user action is required.

- Shared native format3 packaging passes133 tests and six loader cases; original format2 bytes and connected recovery remain valid ([M302](diagnostics/2026-10-25-shared-native-artifacts.md)). Next: verified authority handoff and connected shared recovery. No user action is required.

## Prior checkpoints — 2026-10-24

- Shared initial runtime authority and format2 guards pass124 tests plus a private compositor run ([M301](diagnostics/2026-10-24-shared-authority.md)); native packaging and verified handoff remain. No user action is required.

- Two-root characterization confirms the shared-runtime-authority gap; same-root exclusion works, cross-root exclusion is not yet implemented ([M300](diagnostics/2026-10-24-admission-roots.md)). No user action is required to continue.

- Connected sessions/subscriptions and unresolved DM now pass alongside real native Wayland/UI/API lifecycle ([M299](diagnostics/2026-10-24-connected-native-wayland.md)). Next: shared runtime ownership across independent coordinator roots.

- Native API destruction/recreation and real-bar unloads pass ([M298](diagnostics/2026-10-24-native-api-lifecycle.md)); next combine connected state with these surfaces, then shared runtime ownership.

- Actual native bar and Wayland surfaces pass stable/dev/stable; two demonstrated UI lifecycle fixes and connected recovery regression pass ([M297](diagnostics/2026-10-24-native-wayland.md)). Full shell/API lifecycle and cross-root ownership remain.

- Native scoped shell API injection and routing pass across nine supported payloads ([M296](diagnostics/2026-10-24-native-shell-api.md)). Full native shell/bar/compositor and ownership remain.

- Fixed the guarded-slot/native-panel ownership mismatch; actual bar-method regression,88 tests and connected recovery pass ([M295](diagnostics/2026-10-24-native-bar-ownership.md)). Next: native shell capability facades and full bar/compositor qualification.

- Guarded native profiles now pass original-guardian recovery and resumed dev/stable, preserving sessions and unresolved DM ([M294](diagnostics/2026-10-24-native-recovery.md)). Cross-root ownership and native bar/plugin API are the next profile gates.

- Connected guarded native profiles preserve saved sessions and unresolved DM across stable/dev/stable;15 subscriptions pass ([M293](diagnostics/2026-10-24-native-connected.md)). Next: native original-guardian/coordinator recovery.

- Closing-time UI/IPC now fences actual widgets and monitor/overlay reopening even during held work; negative control and88 tests pass ([M292](diagnostics/2026-10-24-closing-ui-admission.md)). Connected native recovery is next.

- Actual supported apps/widgets now pass guarded native admission with empty accounts ([M291](diagnostics/2026-10-24-guarded-supported-set.md)); closing-time UI and connected recovery are next.

- Actual Omabit Service/Panel now pass guarded native construction, service access, scoped IPC and drain ([M290](diagnostics/2026-10-24-guarded-omabit.md)); actual Theme/DMs are next.

- Guarded artifacts pass88 private tests and five native cases with synthetic app/widget bodies ([M289](diagnostics/2026-10-24-guarded-artifacts.md)); actual components are next.

- One-use native admission passes78 private tests and five guarded native-loader cases ([M288](diagnostics/2026-10-24-native-admission.md)); immutable host packaging and actual app integration remain.

- Actual Omabit Panel/Service pass five scoped IPC and reader-drain cycles ([M287](diagnostics/2026-10-24-scoped-omabit-ipc.md)). Native creation/admission and gated widget integration remain.

- Pinned artifact preparation and native loaded identity pass62 private tests and five native-registry profile roundtrips ([M286](diagnostics/2026-10-24-native-artifacts.md)). Real wrappers/widgets and durable loader admission remain next.

- Recovered dev/stable roundtrip passes10 restored subscriptions, retained pending DM/session state and no Theme replay;287 Theme tests pass ([M285](diagnostics/2026-10-24-recovery-resume.md)). Native loader/admission and deployment remain.

- Return after recovery exposed passive Theme re-publication and a swallowed HTTP test assertion ([M284](diagnostics/2026-10-24-recovery-resume-finding.md)); generated pass rejected. Fix passive versus explicit publication and rerun.

- Actual private plugin recovery passes two coordinator kills and failed cleanup, preserving pending writes/credentials ([M283](diagnostics/2026-10-24-explicit-collection-recovery.md)). Next continue activation and return after recovery.

- Explicit collection recovery model passes48 private tests and original-writer migration/refusal ([M282](diagnostics/2026-10-24-collection-recovery-model.md)). Actual plugin/runtime recovery integration remains next.

- Two unresolved actual plugin writes survive supervised shell loss without replay; all17 channels close ([M281](diagnostics/2026-10-24-collection-recovery-cohort.md)). The collection fence remains pending explicit recovery.

- Actual private QML supervision passes four guardian closures, six coordinator kills and16 channel deletions ([M280](diagnostics/2026-10-24-supervised-collection.md)). Recovery still requires explicit old-instance transport/application binding; the fence remains.

- External supervision prerequisite passes four private orphan/late-fork/EOF/owner-loss cases ([M279](diagnostics/2026-10-24-collection-supervisor.md)). Bind it to the actual QML collection before introducing independent recovery authority.

- Theme real-Urbit crash recovery and safe disconnect pass ([M278](diagnostics/2026-10-24-theme-urbit-transport.md)); two positive subscriptions, two channel404s, no app writes and clean fresh-fake shutdown. Independent collection recovery is next.

- Theme safe disconnect passes284 private tests, actual CLI crash/retry and full collection regression ([M277](diagnostics/2026-10-24-theme-safe-disconnect.md)). Fresh actual-Urbit Theme transport is the next separate gate.

- Theme concurrent durable transport passes268 Python tests, client kills/lost-delete/readback without replay, and collection regression ([M276](diagnostics/2026-10-24-theme-durable-transport.md)). Safe credential-preserving disconnect and actual-Urbit Theme recovery remain before coordinator recovery.

- Source follow-up M275 identifies Theme watch/poke cleanup as the next prerequisite: current ownership is in memory and multiple Eyre instances must remain supported. Add durable Theme evidence before coordinator recovery; DMs M274 remains passed.

- Fresh actual Urbit passes DMs helper-kill/lost-delete recovery and shutdown after fixing owned stream cancellation;150 private checks and collection regression pass ([M274](diagnostics/2026-10-24-dms-urbit-transport.md)). Independent collection recovery remains next; credential expiry, archival and native identity gates stay open.

- DMs durable channel recovery passes147 tests, actual helper SIGKILL/lost-delete/reconnect with no app replay, and final-source collection regression ([M273](diagnostics/2026-10-24-dms-durable-transport.md)). Six writer fixtures retained. Actual Urbit, credential expiry/history archival and independent coordinator recovery remain.

- DMs now drains on EOF/broken pipe;118 private checks and actual shell-loss channel deletion pass, with collection uncertainty retained ([M272](diagnostics/2026-10-24-dms-eof-drain.md)). Durable helper-crash cleanup evidence remains next.

- Private shell replacement resumes journaled activation/rollback; unexpected death leaves helpers/channel and correctly persists uncertainty ([M271](diagnostics/2026-10-24-collection-shell-recovery.md)). Plugin-owned cleanup evidence and production binding remain.

- Journal-driven private collection transitions and rollback survive six coordinator kills;30 private checks and the baseline pass ([M270](diagnostics/2026-10-24-collection-coordinator.md)). Whole-shell recovery and production adapter binding remain.

- Persistent collection journal passes 24 private tests, including actual coordinator kill/resume and durable uncertainty ([M269](diagnostics/2026-10-24-collection-journal.md)). Binding real runtime observations/effects is next.

- Combined Omabit/Theme/DMs native-loader transitions pass six drains each,12 deleted channels and preserved/reconciled intent ([M268](diagnostics/2026-10-24-supported-collection.md)). Persistent collection transition/restart/rollback is next.

- Signal-killed helpers now retain unresolved descendant cleanup; actual receiver lifetime and real-QML crash fences pass ([M267](diagnostics/2026-10-24-helper-exit-fence.md)). Combined Omabit/Theme/DMs transition testing is next.

- Actual Omabit monitor/setup readers now drain across five profiles; late SSH results are fixed and missing live owners refuse closure ([M266](diagnostics/2026-10-24-omabit-reader-drain.md)). Collector/auth/clipboard descendants and durable activation remain next.

- Actual offline Theme restore survives native font/shell replacement, post-effect failure and verified retry with unchanged sync intent ([M265](diagnostics/2026-10-24-native-font-restore.md)). Remaining Omabit monitor/launcher/auth helper shutdown is next.

- Native font restart and command fanout are characterized: a shell-owned helper loses its result, an external helper survives, and native exit0 can hide retint failure ([M264](diagnostics/2026-10-24-native-appearance-effects.md)). Actual offline Theme CLI integration across restart is next.

- Copied native headless theme staging/templates pass the offline restore and helper-kill matrix, with exact frozen-file readback ([M263](diagnostics/2026-10-24-native-theme-staging.md)). Native fanout and font-triggered shell restart remain.

- Offline Theme checkpoint/restore passes actual helper failure, exact retry and forced helper exit, with237 Python checks ([M262](diagnostics/2026-10-24-theme-offline-checkpoint.md)). Native staging/fanout and shell restart are the next collection boundary; the goal remains active.

- Real-service appearance rollback now passes four partial-effect failures and failed-restoration retry; a temporary-theme startup stall is fixed, with222 Python checks ([M261](diagnostics/2026-10-24-theme-partial-rollback.md)). Offline retained appearance and durable collection rollback remain.

- Evidence-backed legacy Theme overrides now migrate with original bytes retained;218 Python checks and actual connected restart pass ([M260](diagnostics/2026-10-24-theme-legacy-adoption.md)). Partial-apply restoration and collection rollback remain.

- Theme now has one cooperative appearance owner and selection-bound hooks;213 Python/native/profile checks and connected lifecycle pass ([M259](diagnostics/2026-10-24-theme-appearance-owner.md)). Legacy migration and partial-apply rollback remain.

- Authenticated state reconciliation resolves invitation/read effects without replay and survives another restart;112 Python tests and private lifecycle pass ([M258](diagnostics/2026-10-24-dms-effect-reconciliation.md)). Exclusive appearance ownership is next.

- DMs message, invitation and read-marker uncertainty survives stable/dev/edge replacement without replay;101 Python and6 profile QML tests pass ([M257](diagnostics/2026-10-24-dms-conversation-effects.md)). Observation reconciliation and collection activation remain.

- Applied Theme/DMs drain test passes all three profiles; Omabit service waits for delayed helper/read exit and blocks reconnects in the private inert-tool fixture ([M256](diagnostics/2026-10-24-collection-transport-drains.md)). Complete collection activation remains.

- Theme and DMs explicit drains delete all six channels before replacement while preserving appearance state and unknown sends; all Theme regressions pass ([M255](diagnostics/2026-10-24-theme-transport-drain.md)). Omabit lifecycle and complete collection activation remain.

- DMs explicit transport drain deletes all three owned channels before replacement and preserves unknown sends;92 tests pass ([M254](diagnostics/2026-10-24-dms-transport-drain.md)). Theme cleanup remains next.

- DMs unknown sends now survive actual stable/dev/stable helper replacement with no replay;86 Python+JS/QML checks pass ([M253](diagnostics/2026-10-24-dms-outbound-recovery.md)). Graceful channel cleanup remains next.

- Standalone Tend client install/readiness/restart and private deterministic release checks pass; actual setup card/service separately qualified ([M252](diagnostics/2026-10-24-tend-client14-runtime.md)).

- Standalone Tend backend transport and real setup-card/service pass private tests; native13 client subgate passes but overall fixture counting fails, with fresh14 next ([M251](diagnostics/2026-10-24-tend-client13-ui.md)).

- Tend direct backend install, separate readiness, exact publisher and receiver restart pass on fresh private ships; independent source/pier/chain audits retained ([M250](diagnostics/2026-10-24-tend-backend-runtime.md)). Standalone plugin UI/durable setup intent is next.

- Actual DMs lost-receipt characterization confirms no replay but clears busy while outcome remains unknown; durable transition handoff must retain unresolved writes ([M249](diagnostics/2026-10-24-uncertain-plugin-lifecycle.md)).

- Tend11 verifies publisher read permission changed; a display-only fixture assertion is corrected in fresh12 ([M248](diagnostics/2026-10-24-tend-publisher11-permissions.md)).

- Tend10 proves publisher readiness but receiver acknowledgement remains unready; fresh11 explicitly publishes the test desk ([M247](diagnostics/2026-10-24-tend-receiver10-readiness.md)).

- Actual connected Theme/DMs helper/stream teardown passes private synthetic-Eyre profile roundtrips; all remote fixture channels remain undeleted, so graceful cleanup/pending-write work stays open ([M246](diagnostics/2026-10-24-connected-plugin-lifecycle.md)).

- Tend09 revealed a generic noun-mark rejection; fresh10 uses the explicit Lens mark wrapper ([M245](diagnostics/2026-10-24-tend-publisher09-mark-failure.md)).

- Tend08 identified a fixture path-list type error, now reproduced/fixed by offline compilation; fresh09 is running ([M244](diagnostics/2026-10-24-tend-publisher08-type-failure.md)).

- VM42 passes actual owner-service restart, public interrupted recovery and four application/subscription recovery cycles; original disk/receipts independently verified and retained ([M243](diagnostics/2026-10-24-service-restart42-runtime.md)). Public project UI and release qualification remain.

- Tend07 stopped cleanly after an uncertain publisher-setup acknowledgement; failed data/receipts are retained. A fresh fixture correction is prepared ([M242](diagnostics/2026-10-24-tend-publisher07-failure.md)).

- Fresh private Tend publisher/receiver trial07 is admitted; direct backend installation and protocol readiness are pending ([M241](diagnostics/2026-10-24-tend-trial-admission.md)).

- Tend backend-install inputs are frozen with generated-mark/Kelvin preservation and protocol2 readiness documented; actual private distribution/UI remains next ([M240](diagnostics/2026-10-24-tend-backend-inputs.md)).

- Applied `native-peers` stage passes fresh official keyed application/reconnect runtime and independent source/chain/pier audit ([M239](diagnostics/2026-10-24-native-keyed-recovery-runtime.md)). Docker/runner integration remains.

- Reproducible native keyed recovery stage is applied; guard/argv checks pass and fresh source run06 is admitted ([M238](diagnostics/2026-10-24-native-keyed-source.md)).

- Official keyed application/subscription recovery passes: four writes, seven scries and six snapshots/two channels across restart, independently audited ([M237](diagnostics/2026-10-24-keyed-application-recovery.md)). Reproducible source-stage integration is next.

- Official-runtime private keyed peers pass messaging/restart and independent audit; fresh keyed persistent-state/subscription trial05 is running ([M236](diagnostics/2026-10-24-official-private-peers.md)).

- Correct Lens exit and same-pier SIGTERM reopening pass with the unchanged official runtime ([M235](diagnostics/2026-10-24-official-lens-recovery.md)). Fresh-key SIGTERM itself remains defective; official peers/Docker remain separate.

- Actual Theme/DMs helpers pass isolated empty-account collection roundtrips with verified teardown and per-profile identity preservation ([M234](diagnostics/2026-10-24-real-plugin-lifecycle.md)). Connected/busy/rollback gates remain.

- Two keyed private peers pass messaging/restart and independent chain/pier audit with the experimental runtime ([M233](diagnostics/2026-10-24-private-peers-recovery.md)). Correct official-runtime Lens exit is running separately.

- Private keyed peers exchange messages; a creation-only reopen flag makes trial02 fail. Failed evidence retained and fresh corrected03 running ([M232](diagnostics/2026-10-24-private-peers-reopen.md)).

- First keyed-peer test hits its isolated4GiB cap; failed evidence retained. Fresh6GiB/no-swap/one-core retry is admitted and running ([M231](diagnostics/2026-10-24-private-peers-oom.md)).

- Fresh service-restart VM42 is running with independently verified inputs and a prepared stopped-disk audit; runtime result pending ([M230](diagnostics/2026-10-24-service-restart42-preparation.md)).

- Private-chain key rotation and Jael state survive clean keyed restart with the experimental runtime; independent retained-chain/pier audit passes ([M229](diagnostics/2026-10-24-private-watcher-recovery.md)). Official runtime, two peers and Docker devnet remain separate.

- Actual owner-service restart fixture is applied and source/admission checks pass; fresh VM and independent runtime audit remain pending ([M228](diagnostics/2026-10-24-service-restart-source.md)).

- VM41 public recovery/application subgates pass independent audit, but its final wrong-actor fixture check fails overall. The actual copied database validates the source fixture correction ([M227](diagnostics/2026-10-24-public-recovery41-actor-audit.md)). Real service restart is next.

- Private watcher setup now receives Gall acknowledgments; confirmation-lag fixture failure retained and fresh trial pending ([M226](diagnostics/2026-10-24-private-watcher-setup.md)). VM41 final actor-replay check failed; independent audit remains pending.

- Canonical Hoon seed formatting fixes an intermittent pre-boot fixture refusal; actual evaluator and full key-validation checks pass ([M225](diagnostics/2026-10-24-private-chain-hex.md)). Private watcher runtime trial continues separately.

- Experimental Vere correction passes a failing-original/passing-candidate signal regression and fresh keyed stop/reopen ([M224](diagnostics/2026-10-24-private-fixed-keyed.md)). No installed runtime changes; chain watcher and peer qualification remain.

- Private Lens exit fails to stop the fresh keyed process; both trials are retained ([M223](diagnostics/2026-10-24-private-lens-shutdown.md)). An isolated source-build correction is next; no runtime has been installed.

- Pinned keyed shutdown failure is traced to duplicate signal cleanup; same-launcher fake control passes ([M222](diagnostics/2026-10-24-keyed-signal-diagnosis.md)). Runtime fix and private Lens/reopen qualification remain.

- Fresh public interrupted-recovery VM41 is verified and running in its private two-core/no-NIC scope ([M221](diagnostics/2026-10-24-public-recovery41-preparation.md)). Independent runtime audit remains pending.

- Public member lifecycle passes actual CLI/Docker and independent state/recovery verification ([M220](diagnostics/2026-10-24-public-member-lifecycle-runtime.md)). Full public interrupted recovery is next.

- Private keyed HTTP boot reaches identity but fails clean shutdown ([M219](diagnostics/2026-10-24-private-keyed-boot-shutdown.md)); isolated probes are stopped and retained. Launcher signal handling is next.

- Two fresh private-chain keys and chain-backed dawn responses pass the pinned validator, including wrong-key/revision refusal ([M218](diagnostics/2026-10-24-private-key-validation.md)). HTTP gateway and actual ship boot remain.

- Pinned Vere confirms the private gateway method contract through isolated evaluation ([M217](diagnostics/2026-10-24-private-boot-requests.md)); actual gateway/ship boot remains.

- Private chain foundation now passes pinned contract deployment, chain/genesis/code admission and database reopening ([M216](diagnostics/2026-10-24-private-chain-foundation.md)). Gateway/ship boot and full devnet qualification remain.

- VM40 is running the identical frozen member fixture with an explicit two-core emulation cap; production checks remain unchanged ([M215](diagnostics/2026-10-24-public-members40-preparation.md)). This is a resource-sensitivity experiment, not a runtime pass.

- VM39 retained an interrupted first Start after an engine API read failure; six public dispatches and the stopped-disk audit pass, but the member/application gate failed ([M214](diagnostics/2026-10-24-public-members39-failure.md)). A bounded emulation-resource experiment is next.

- Private native-loader characterization confirms service overlap and selected/loaded generation differences; explicit teardown works with synthetic helpers ([M213](diagnostics/2026-10-24-collection-loader.md)). Real plugin collection activation remains to implement and qualify.

- Verified public member-lifecycle VM39 is running with887 frozen source inputs, actual CLI dispatch and direct application recovery checks ([M211](diagnostics/2026-10-24-public-members39-preparation.md)). No runtime pass yet.

- Observation refusals now distinguish ownership changes from an expired observation without changing the freshness bound ([M210](diagnostics/2026-10-24-member-observation-diagnostics.md)). Applied source/race/vet checks pass; runtime qualification remains.

- VM38 failed before recovery at the first member post-Start observation; the exact interrupted receipt and original disk are retained and independently audited ([M209](diagnostics/2026-10-24-public-recovery38-failure.md)). Specific refusal cause remains unknown; diagnostics and a fresh isolated run are next.

- Read-only project history UI passes actual QML/Process with16 private inert reads and no mutations ([M208](diagnostics/2026-10-24-project-history-ui.md)), including late old-profile response rejection. Real-runner UI and mutating controls remain separate.

- Public member lifecycle fixture passes applied source/race/entry gates ([M207](diagnostics/2026-10-24-public-members-vm-source.md)); all lifecycle dispatch now has an actual-CLI test path. Its runtime gate remains pending and VM38 excludes it.

- Historical M206 preparation/running record is superseded by M209 above.

- Public initial lifecycle now passes actual CLI/rootless-Docker and independent stopped-disk checks ([M205](diagnostics/2026-10-24-public-initial-lifecycle-runtime.md)): five new/five lookup requests, three duplicate refusals, two empty parents, clean anchor Stop, no ship boot.

- Plugin-owned backend setup guidance now includes verified kiln payloads and source-preserving readiness decisions from Urbit/Talon/Omart/Lattice ([M204](diagnostics/2026-10-24-backend-setup-references.md)). Skill validation passes; the actual one-click installer gate remains.

- Verified public initial-lifecycle VM37 is running ([M203](diagnostics/2026-10-24-public-initial37-preparation.md)): five setup actions through the actual CLI, with no ship boot. Independent runtime qualification remains pending.

- VM36 was interrupted with outer143 before any final report ([M202](diagnostics/2026-10-24-public-recovery36-interrupted.md)). Original disk and WAL-aware partial audit are retained; no runtime pass. Initial public lifecycle is next, then fresh full public recovery.

- Applied recovery validation broader managed race passes696.430s ([M201](diagnostics/2026-10-24-recovery-read-validation.md)); the earlier aggregate timeout is retained. Full public Docker recovery still needs qualification.

- Public initial-lifecycle fixture passes source/admission checks ([M200](diagnostics/2026-10-24-public-initial-vm-source.md)): actual CLI reservation/preparation/anchor/empty storage, no ship boot. Runtime qualification remains; frozen VM36 excludes these fixture edits.

- Fresh public recovery VM36 is verified and running with the read-cost fix ([M199](diagnostics/2026-10-24-public-recovery36-preparation.md)). Full runtime and independent verification remain pending; VM35 remains failed.

- Recovery read timeout reproduced and validation cost reduced ([M198](diagnostics/2026-10-24-recovery-read-validation.md)). Both immutable interrupted/completed histories now meet the unchanged five-second deadline with identical projections; model race/vet pass and broader managed rerun is pending. VM35 remains failed. Fresh full public recovery is next.

- Public recovery VM35 failed a checkpoint-read timeout after its third deliberate lost reply ([M197](diagnostics/2026-10-24-public-recovery35-failure.md)). The independent failed-run audit and retention pass; uncertainty and partial effects remain intact. Investigate the read before a fresh retry; public-initial lifecycle draft is parked. Earlier VM35 running statements below are historical.

- Theme hook routing and instance-owned appearance files pass ([M196](diagnostics/2026-10-24-theme-profile-files.md)), including a reproduced/fixed dev notification deleting stable's synthetic override. Visible desktop effects remain shared and require exclusive collection activation; current installations are unchanged.

- Theme profile UI and actual private Quickshell IPC pass ([M195](diagnostics/2026-10-24-theme-profile-ipc.md)): identity injection, distinct targets/state and address removal preserve stable. Hooks, appearance ownership, native desktop and full switching remain.

- DMs stable/dev/edge QML routing now passes with native controls and inert I/O ([M194](diagnostics/2026-10-24-dms-profile-ui.md)). Labels, overlay/popup routes and singleton state are separate; native compositor/IPC/hooks and full switching remain. VM35 continues independently.

- Actual public recovery appliance35 is independently verified and running in a fresh no-NIC VM ([M193](diagnostics/2026-10-24-public-recovery35-preparation.md)). It tests public selection/dispatch/lookup atop M192's qualified private recovery. Runtime verification remains pending.

- **Interrupted recovery now passes the independent Docker runtime gate**: three lost replies/store reopens, two observations/three continuations, sixteen writes/four subscription cycles, retained piers and original sessions, plus actual saved-status CLI ([M192](diagnostics/2026-10-24-anchor-continuation-runtime.md)). VM33 remains failed. Public mutation CLI, OS service restart and UI remain separate.

- The actual-collector public recovery VM fixture passes compilation, inert helper checks and host/preparation refusals ([M191](diagnostics/2026-10-24-public-recovery-vm-source.md)). It has not booted. VM34 remains the earlier private continuation run.

- Public recovery start/observe/continue and actual-collector lost-reply reconciliation pass source/inert gates ([M190](diagnostics/2026-10-24-public-recovery-actions.md)). Exact checkpoints and unknown-Create refusals remain enforced. VM34 predates the commands; guarded public-command Docker coverage is next.

- Saved recovery source and exact member receipt references pass observer/client/CLI gates ([M189](diagnostics/2026-10-24-public-recovery-source.md)). Public recovery actions and runtime qualification remain; VM34 predates this read.

- Observer-readable saved recovery checkpoint and CLI pass source gates ([M188](diagnostics/2026-10-24-public-recovery-checkpoint.md)); reads preserve uncertainty and issue no runtime effects. Frozen VM34 predates this route. Public recovery actions and their runtime qualification remain.

- Corrected continuation appliance34 is frozen, independently checked and running in a fresh no-NIC VM ([M187](diagnostics/2026-10-24-anchor-continuation-rerun-preparation.md)). Runtime and stopped-disk verification remain pending; VM33 remains failed.

- Public Stop/exit/retire/resume/current-observation commands and corrected prerequisite diagnostics pass expanded/applied source gates ([M186](diagnostics/2026-10-24-public-member-actions.md)). Coordinated recovery APIs, actual public-mutation Docker tests and QML remain.

- VM33 failed-run disk audit and retention now pass ([M185](diagnostics/2026-10-24-anchor-continuation-trial33.md)): one uncertain Stop retained, no continuation dispatched, twelve earlier writes/three cycles, both final pier states hashed. The corrected full runtime rerun remains pending.

- VM33 failed on a fixture calling an acknowledged-dispatch wait after a lost reply/store reopen. Its disk is retained and the fixture correction passes source gates ([M184](diagnostics/2026-10-24-anchor-continuation-trial33.md)); independent inspection and a fresh continuation/saved-status rerun remain. This supersedes the running status below.

- Five initial public project actions and an actual-collector lost-reply gate now pass with an inert backend ([M183](diagnostics/2026-10-24-public-initial-project-actions.md)). Member Stop/resume/recovery APIs, public-command Docker qualification and QML remain pending.

- Public project reservation and lookup-only CLI now pass source/Unix-RPC/CLI gates ([M182](diagnostics/2026-10-24-public-project-reservation.md)); runtime lifecycle routes remain pending. Frozen schema40 VM33 is running interrupted continuation and public saved-status tests ([M181](diagnostics/2026-10-24-anchor-continuation-preparation.md)).

- Public saved project status now passes focused source/CLI/observer tests and skill validation ([M180](diagnostics/2026-10-24-project-saved-status.md)). Runtime CLI qualification is included in the next schema40 continuation VM; mutation routes remain pending.

- Coordinated anchor recovery now independently passes in VM32/schema39, including final cleanup, sixteen writes and four subscription cycles ([M179](diagnostics/2026-10-24-anchor-recovery-runtime.md)). Original disks are retained and the offline VM is stopped. Schema40 interrupted continuation is next.

- VM29/schema37 now independently passes direct-client application/subscription recovery and healthy/stopped/restarted anchor review ([M159](diagnostics/2026-10-23-anchor-review-runtime.md)). Original disks, journals and final pier states are retained; the offline VM is stopped. This is diagnostic anchor review, not coordinated repair.

- Read-only recovery observations were added in unreleased schema39 for
  interrupted coordinated recovery ([M169](diagnostics/2026-10-24-persistent-recovery-observation.md)).
  Store/queue race334.200s, checkpoint ancestry149.324s, migration and original38
  refusal pass. Full39 regression now passes in1722.999s with vet (M171).
  Active source now uses schema40 for explicit continuation; restart/rollback and
  migration tests pass. Focused race861.164s and the guarded continuation fixture
  checks pass (M175); full40 race2164.666s and vet now pass (M178).
  Runtime continuation is pending. Public routes
  remain open. Frozen VM31 tested
  schema38 coordinated repair and failed final fixture cleanup. Independent disk
  checks verify the operation and sixteen-write application cycle (M172), but
  the later clean VM32 pass (M179) supersedes that pending rerun.
  Schema38 full baseline race1307.093s and later focused changes pass (M167).
- Earlier schema35 repeated member replacements through generations3/4,
  migration/replay and the full managed race suite pass ([M113 evidence](diagnostics/2026-10-23-member-repeated-resume-store.md)).
- VM26/schema37 remains the fully queued final-Stop reference: queued sender
  lifecycle after peer changes, twelve remote writes and three subscription
  recovery cycles pass with original sessions and exact receiver continuity
  ([M137](diagnostics/2026-10-23-current-context-lifecycle-runtime.md)).
- Later read-only anchor review passes pure and concrete source gates
  ([M135](diagnostics/2026-10-23-anchor-recovery-review-model.md),
  [M136](diagnostics/2026-10-23-anchor-recovery-review-adapter.md)). These changes
  and M134 direct published transport are not included in frozen VM26.
- VM27/schema37 failed at sender replacement readiness before the anchor fault
  ([M143](diagnostics/2026-10-23-anchor-review-trial27.md)). Five direct-client phases
  and three replacements were recorded; originals and clean final piers are retained.
  Timing remains a hypothesis; the fixture diagnostics/audit placement pass source checks ([M144](diagnostics/2026-10-23-readiness-timing-fixture.md)).
- VM28 completes direct-client twelve-write/three-cycle recovery and readiness, then fails restarted-anchor inspection on retained historical FinishedAt plus a secondary fixture reopen panic ([M151](diagnostics/2026-10-23-anchor-review-trial28.md)). Original disks and both final piers are independently retained. Recovery-specific timestamp/fixture corrections are under regression; full runtime qualification remains open.
- Pure coordinated drain now passes full race13.805s/vet ([M139](diagnostics/2026-10-23-anchor-drain-model.md)); the concrete adapter also passes57 inert race cases ([M140](diagnostics/2026-10-23-anchor-drain-adapter.md)). Drain persistence, namespace succession and coordinated runtime remain pending.
- Explicit namespace succession passes its pure race/vet gate ([M141](diagnostics/2026-10-23-anchor-namespace-model.md)); its separate Docker adapter passes22 inert race cases ([M142](diagnostics/2026-10-23-anchor-namespace-adapter.md)). Coordinated retirement/replacement and persistence remain next.
- Coordinated retirement passes pure and concrete source gates ([M146](diagnostics/2026-10-23-anchor-retirement-model.md), [M147](diagnostics/2026-10-23-anchor-retirement-adapter.md)). Same-pier successor contracts and replacement are next; runtime and persistence remain pending.
- Explicit successor member contracts pass pure/concrete source checks without changing ordinary digests ([M148](diagnostics/2026-10-23-recovery-member-contract.md)); serial replacement execution is next.
- Pure serial replacement passes race/vet with uncertainty and peer continuity checks ([M149](diagnostics/2026-10-23-anchor-replacement-model.md)); concrete adapter tests are running. Running still does not imply fresh readiness or recovery completion.
- Concrete successor replacement passes inert race/vet ([M150](diagnostics/2026-10-23-anchor-replacement-adapter.md)). Fresh recovery readiness, causal history and persistence are next; the runtime recovery gate remains open.
- Recovery-specific historical timestamp handling and failed-review cleanup now pass exact VM28 regression, full pure race and related adapter race ([M152](diagnostics/2026-10-23-anchor-history-fix.md)). A fresh full VM rerun is next.
- Corrected appliance29/schema37 preparation ([M153](diagnostics/2026-10-23-anchor-history-rerun-preparation.md)) is now followed by the verified M159 runtime pass.
- Complete successor identity/readiness passes pure race/vet ([M154](diagnostics/2026-10-23-anchor-recovery-readiness-model.md)); concrete adapter checks are running.
- Complete successor readiness also passes its concrete observation-only adapter gate ([M155](diagnostics/2026-10-23-anchor-recovery-readiness-adapter.md)). Later causal history, persistence and coordinated runtime remain.
- Ordinary replacements after one complete recovery pass pure history/prefix checks ([M156](diagnostics/2026-10-23-recovered-member-generations.md)). Multiple recovery boundaries and concrete/persistent integration remain.
- Two recovery boundaries with ordinary replacements between/after them now pass pure history tests and existing adapter regressions ([M157](diagnostics/2026-10-23-recovery-epochs.md)). Concrete later contracts/readers and persistence remain next.
- Ordinary post-recovery creation contracts pass pure/concrete and legacy-format checks ([M158](diagnostics/2026-10-23-recovered-resume-contract.md)); generalized concrete readers and durable coordination remain.
- Concrete current-history readers pass recovery-only and mixed-creator source gates ([M160](diagnostics/2026-10-23-recovery-history-reader.md)); second concrete recovery and global historical-ID checks throughout its phases are next.
- Second coordinated recovery now passes concrete fixture gates, including stopped/restarted anchors, mixed creators and historical-ID guards at every phase ([M161](diagnostics/2026-10-23-second-recovery-adapters.md)); ordinary post-recovery lifecycle and persistence remain next.
- Ordinary post-recovery current readiness, Stop, exit, retirement, resume and fresh readiness pass source/legacy regressions ([M162](diagnostics/2026-10-23-recovered-current-lifecycle.md)); durable recovery coordination is next.
- Original schema37 writer and populated current-lifecycle fixtures are frozen and reproducible ([M163](diagnostics/2026-10-23-schema37-original-writer.md)); actual successor migration/refusal tests remain pending.
- After those gates: coordinated anchor recovery, runner project public
  capabilities/CLI/UI, then remaining roadmap acceptance. The entries below retain
  earlier evidence; their older pending statements are historical.

| Area | Evidence / remaining work |
| --- | --- |
| Import diagnostics | Source diagnostics and [nine-case guided layout/runtime fixtures](diagnostics/2026-10-22-import-layout.md) passed; [actual guided UI and private VM boundary](diagnostics/2026-10-22-import-ui.md) passed; [offline installer/engine/runtime-version gate](diagnostics/2026-10-22-import-guest.md) passed; [real same-filesystem import/Start/Stop](diagnostics/2026-10-22-import-runtime.md) passed; [real cross-filesystem import/Start/Stop](diagnostics/2026-10-22-import-copy.md) passed; [partial-copy preservation](diagnostics/2026-10-22-import-interruption.md) passed; [runner/CLI source restoration](diagnostics/2026-10-22-import-restore-integration.md) passes source tests; [real restoration and re-import](diagnostics/2026-10-22-import-restore-runtime.md) and [guided UI fixtures](diagnostics/2026-10-22-import-restore-ui.md) passed; [actual UI-to-VM restoration](diagnostics/2026-10-22-import-restore-ui-runtime.md) passed; additional failure modes and native desktop integration remain |
| CLI and operational/developer skills | [Initial operator skill/offline help](diagnostics/2026-10-22-operator-entry.md) passed; [offline task families](diagnostics/2026-10-22-task-help.md) passed; [five scripted CLI tasks](diagnostics/2026-10-22-operator-walkthrough.md) passed; [initial plugin-developer guidance](diagnostics/2026-10-22-developer-skill.md) added and validated; [lost-reply reconciliation](diagnostics/2026-10-22-cli-lost-reply.md) and [structured error guidance](diagnostics/2026-10-22-cli-recovery-guidance.md) passed; import-recovery walkthrough expansion, other runtime next actions, skill distribution and developer acceptance remain |
| Fake project networking | [Pinned-runtime source notes](project-fake-networks.md) recorded; [Docker peer messaging and session recovery passed](diagnostics/2026-10-22-docker-peer-recovery.md); [Docker application state/subscription recovery passed](diagnostics/2026-10-22-application-state-docker.md); [duplicate-project isolation passed](diagnostics/2026-10-22-docker-project-isolation.md); [HTTP publication/anchor fixture passed](diagnostics/2026-10-23-endpoint-publication.md); [actual published Eyre state/subscription recovery passed](diagnostics/2026-10-23-published-state.md); [published duplicate-project isolation passed](diagnostics/2026-10-23-published-project-isolation.md); [actual-ship anchor state/subscription recovery passed](diagnostics/2026-10-23-anchor-state-recovery.md); [Public runner lifecycle and real service recovery](diagnostics/2026-10-24-service-restart42-runtime.md) pass; public UI and release qualification remain |
| Independent sessions | Synthetic and [fresh-fake runtime](diagnostics/2026-10-21-isolated-auth.md) gates passed |
| Theme/DMs integration | [Private receiver, real keyring, direct read/SSE, independent logout and standalone helper gates passed](diagnostics/2026-10-21-plugin-handoff.md); [durable issuer fixture gate](diagnostics/2026-10-21-durable-issuer.md) passed; [verified launcher and real-pair recovery gate](diagnostics/2026-10-21-receiver-recovery.md) passed; [installed declarations/CLI and offscreen approval panel](diagnostics/2026-10-21-approval-flow.md) passed; [real CLI/panel/receiver and running-consumer gate](diagnostics/2026-10-21-public-auth-flow.md) passed; [completed archival and replay checkpoint](diagnostics/2026-10-22-application-history.md) passed; [three stale-stream regressions fixed](diagnostics/2026-10-22-dms-stream-generation.md); [actual writes/reconnect passed](diagnostics/2026-10-22-plugin-transport-native.md); [strict acknowledgement validation also passed](diagnostics/2026-10-22-plugin-receipts-native.md); retained-helper pruning, hosted compatibility and full profile gates remain |
| Backend installation/compatibility | [Tend native distribution/readiness/restart](diagnostics/2026-10-24-tend-backend-runtime.md) passes direct Eyre, exact publisher and preserved protocol2. Standalone plugin UI and durable install intent remain; discovery stays in Omart. |
| Private Ethereum chain | [Applied native keyed peers](diagnostics/2026-10-24-native-keyed-recovery-runtime.md) pass private chain, messages, persistent state/subscription recovery and copied-chain audit. Official first boot uses explicit Lens exit. Docker/runner integration remains. |
| Stable/dev/edge collections | [Actual Theme/DMs empty-account teardown](diagnostics/2026-10-24-real-plugin-lifecycle.md) passes isolated roundtrips; connected subscriptions, busy/uncertain writes, Omabit service and durable activation/rollback remain. |
| Wallet/key storage | [Research notes](platform-security-notes.md) retained; implementation out of scope |

## Current development locations

- Omabit: `/home/thelifeandtimes/repos/omabit-platform-overhaul`, branch
  `codex/platform-overhaul-plan`, base `b233128`; changes uncommitted.
- Theme: `/home/thelifeandtimes/repos/omarchy-urbit-theme-auth`, branch
  `codex/omabit-auth`, base `1719e9f`; private receiver/profile and transport changes uncommitted.
- DMs: `/home/thelifeandtimes/repos/omarchy-urbit-dms-auth`, branch
  `codex/omabit-auth`, base `cf236fd`; private receiver/profile and transport changes uncommitted.

The normal checkouts and installed plugins have not been edited. The retired
Omabit/Tend monorepo in the chat's original directory is not the source to extend.
No release, installation or live-fleet upgrade has been made.

## Current work

The user resumed the full goal on 2026-10-22 and requested continued work without
stopping at each checkpoint. Persistent application-state and subscription recovery now pass in fresh native
and Docker fakes. Duplicate-identity project isolation now passes as well. The ordinary-bridge HTTP publication/anchor fixture now passes. Current work is
runner project integration; the combined published duplicate-project and actual-ship anchor recovery gates now pass.
Published Eyre application/subscription recovery has passed. [Real runner login
across same-pier restart](diagnostics/2026-10-23-login-restart.md) also passes, with
its original disks and independently verified receipts retained.
[Durable project reservations](diagnostics/2026-10-23-project-store.md) pass
source/migration checks. [Project runtime assessments](diagnostics/2026-10-23-project-observation.md)
now distinguish stopped members, unavailable resources and changed namespaces in
source tests. [Schema24 preparation journals](diagnostics/2026-10-23-project-journal.md)
retain each resource intent and returned ID before further work, with source and
original-reader migration checks passing. [Private controller/SQLite wiring](diagnostics/2026-10-23-project-dispatch.md) passes
inert-engine race tests. [Private Docker adapter and fresh session setup](diagnostics/2026-10-23-project-adapter.md)
pass inert-engine tests. [Private server admission/fencing](diagnostics/2026-10-23-project-fence.md)
also passes source checks; [real private-adapter preparation](diagnostics/2026-10-23-project-preparation-runtime.md)
now passes. Public routes and member lifecycle remain to be connected; initial anchor Start now has the private source path described below.
[Published duplicate projects passed in VM23-03](diagnostics/2026-10-23-published-project-isolation.md).
[Actual-ship anchor trial05 failed at its post-restart endpoint probe](diagnostics/2026-10-23-anchor-trial05.md);
the corrected [VM23-08 recovery case now passes independent verification](diagnostics/2026-10-23-anchor-state-recovery.md), with clean shutdown and retained originals. A separate
no-ship real-adapter VM23-07 passed and its original evidence is retained. [Creation uncertainty refinements](diagnostics/2026-10-23-project-refinement.md)
pass source checks; the frozen appliance predates them and its inspections now
pass current pure validators. [The initial anchor Start model](diagnostics/2026-10-23-anchor-start-model.md)
and complete namespace-membership resolver pass source checks. [Schema25 durable
Start and its private worker](diagnostics/2026-10-23-anchor-start-journal.md) now pass
source and original-reader checks. [The actual adapter and private server scheduling](diagnostics/2026-10-23-anchor-start-adapter.md)
pass source tests. [VM23-09](diagnostics/2026-10-23-anchor-start-trial09.md) reached
Start but refused a missing helper path; its uncertainty/cleanup and originals are
retained. [Fresh VM23-10](diagnostics/2026-10-23-anchor-start-runtime.md) now passes
real initial anchor Start with independently verified journals and clean cleanup.
[Schema26 member storage](diagnostics/2026-10-23-member-storage.md) now passes
source, temporary-filesystem and original-reader tests. [Private server scheduling](diagnostics/2026-10-23-member-storage-queue.md)
now passes source checks. [Fresh VM23-11](diagnostics/2026-10-23-member-storage-runtime.md)
passed real queued empty-storage preparation and independent disk verification.
[Read-only member namespace observation](diagnostics/2026-10-23-member-observation.md)
now passes source checks, together with the [negative mount-conflict preflight](diagnostics/2026-10-23-member-mounts.md).
[The initial member launch controller](diagnostics/2026-10-23-member-launch-model.md)
now passes source tests. [Schema27 SQLite wiring](diagnostics/2026-10-23-member-launch-store.md)
passes interruption, worker, full managed race and original-reader checks. [The concrete workspace/pill/Docker adapter and private queue](diagnostics/2026-10-23-member-launch-adapter.md)
now pass source tests. [Fresh VM23-12](diagnostics/2026-10-23-member-launch-trial12.md)
retained one running launch and correctly refused the second stale dispatch, but
failed clean member shutdown. [Fresh VM23-13](diagnostics/2026-10-23-member-launch-trial13.md)
advanced to second-member Create, but refused stale Start and again failed early
member cleanup. [Exact-container batching](diagnostics/2026-10-23-member-observation-batch.md)
passes source tests; [trial14](diagnostics/2026-10-23-member-launch-trial14.md)
completed both running receipts but failed its cold-boot HTTP/cleanup gate. The next
[fixture now restores serial identity warmup](diagnostics/2026-10-23-member-readiness-preparation.md)
with time based on prior measured boots. [VM15 now passes](diagnostics/2026-10-23-member-readiness-runtime.md)
both launches, durable readiness, replay/reopen and clean fixture shutdown, with
independently verified database/filesystem evidence and retained originals.
[The pure readiness model](diagnostics/2026-10-23-member-readiness-model.md) passes
source tests; [the concrete collector](diagnostics/2026-10-23-member-readiness-adapter.md)
now passes inert-engine/temporary-filesystem checks. [Schema28 readiness persistence](diagnostics/2026-10-23-member-readiness-store.md)
passes source and original-reader checks; [private readiness scheduling](diagnostics/2026-10-23-member-readiness-queue.md)
also passes source checks. The guarded real readiness gate now passes in VM15.
Durable Stop/resume/recovery and public integration remain. Neither transport test qualifies the new lifecycle by itself. These are implementation tasks, not a need for
user action.
[The pure initial member Stop model](diagnostics/2026-10-23-member-stop-model.md)
now passes inert-driver tests; [schema29 Stop persistence](diagnostics/2026-10-23-member-stop-store.md)
passes source/original-reader checks. [The concrete Stop adapter and private queue](diagnostics/2026-10-23-member-stop-preparation.md)
now pass source tests. [VM16 durable Stop](diagnostics/2026-10-23-member-stop-runtime.md)
passes both queued graceful exits, replay/reopen and independent disk verification.
Recovery, resume and public integration remain pending.
[Read-only recovery observation](diagnostics/2026-10-23-member-stop-exit-source.md)
now passes model/adapter source tests. [Schema30 recovery persistence](diagnostics/2026-10-23-member-stop-exit-store.md)
passes source and original-reader checks. [Private recovery queue and appliance17](diagnostics/2026-10-23-member-stop-exit-preparation.md)
pass source/preparation checks. [VM17 recovery](diagnostics/2026-10-23-member-stop-exit-runtime.md)
now passes independent runtime verification; its outer-shell143 discrepancy remains recorded.
[Exact stopped-container retirement](diagnostics/2026-10-23-member-retirement-source.md)
passes source model/adapter tests. [Schema31 retirement persistence](diagnostics/2026-10-23-member-retirement-store.md)
passes source and original-reader checks. [Private retirement scheduling and project gates](diagnostics/2026-10-23-member-retirement-queue.md)
pass source checks. [Appliance18](diagnostics/2026-10-23-member-retirement-preparation.md)
adds a guarded real retirement/state-byte comparison. [VM18 retirement](diagnostics/2026-10-23-member-retirement-runtime.md)
now passes independent disk/byte/receipt verification.
[Replacement intent](diagnostics/2026-10-23-member-resume-model.md) and
[the concrete first-replacement adapter](diagnostics/2026-10-23-member-resume-adapter.md)
pass source tests. [Schema32 persistence](diagnostics/2026-10-23-member-resume-store.md) and
[private scheduling](diagnostics/2026-10-23-member-resume-queue.md) also pass source checks.
[The read-only replacement-readiness model and adapter](diagnostics/2026-10-23-member-resumed-readiness-source.md)
and [schema33 persistence/private queue](diagnostics/2026-10-23-member-resumed-readiness-store.md)
pass source and original-reader checks. [VM19's first replacement now passes](diagnostics/2026-10-23-member-resume-runtime.md),
with original disks retained. Runtime readiness and general generation handling
remain; VM19 is frozen schema32 and cannot qualify the newer readiness collector.
No production access or additional permission is needed for these experiments.
The goal service reports active after the user resumed; there is no current
implementation blocker. See [development memory](development-memory.md)
for decisions, supersession links and evidence limits.

## Earlier checkpoint record

The [completed history checkpoint](diagnostics/2026-10-22-application-history.md)
passed on final frozen run `09`: real CLI + approval pane + declared Theme/DMs
receivers + private Secret Service, with direct reads/SSE and independent cleanup.
Fully reconciled rows can be archived without replaying old approvals. Fixture
suites cover 96 cycles, interrupted publication and older helper cleanup. The old
experimental collector refuses the new issuer format. The fresh fake shut down
cleanly. Permanent replay receipts and all retained helpers remain preserved;
reference-coordinated helper-cache pruning is still unimplemented.

Core import work now has both the [layout fixture gate](diagnostics/2026-10-22-import-layout.md)
and the [actual guided UI checkpoint](diagnostics/2026-10-22-import-ui.md). Real
components/callbacks passed reviewed destination/consent, uncertain acknowledgements,
progress, retained failure, revision-bound recovery and explicit stopped completion.
They do not qualify a real pier/runtime import.

A signed private QEMU 11.1.2 bundle is ready at
`/tmp/omabit-import-vm-tools-20261022`. The reusable minimal guest driver passed on
`/tmp/omabit-import-vm-smoke-20261022-04` with software emulation and clean poweroff;
this environment has no accessible KVM. That smoke run completed; the later live
import run is tracked below.
The [offline full-system guest checkpoint](diagnostics/2026-10-22-import-guest.md)
now passed on fresh guest run `02` using appliance `04`: real signed native
packages, systemd, Omabit installer, rootless Docker 29.9.1, verified package
provenance, exact offline OCI loading and Vere 4.6 version execution. It powered
off cleanly in 200.58 seconds. No ship was created or operated. An earlier 4 GiB
trial hit its lab-only memory cap; the successful VM used an 8 GiB scope because
temporary disk writes count alongside guest RAM. Failed disks/logs are preserved.
The [first real trials](diagnostics/2026-10-22-import-runtime-trials.md) now proved
fresh fake boot/clean stop, unchanged guided preview, named same-filesystem import,
stopped completion and terminal-request replay through the actual collector. Trials
03 and 04 failed during explicit Start because the minimal guest first exposed only
one CPU, then lacked a TCP networking module. Both failures retain their receipts,
piers and clean guest shutdown evidence. Those trials did not establish the full start/identity/mount gate; the later pass is recorded below.
[Trial 05 passed](diagnostics/2026-10-22-import-runtime.md) using appliance 06:
real import preserved bytes/metadata/inode and stayed stopped, explicit Start
verified identity and the exclusive parent mount, and explicit Stop completed.
The fresh guest powered off cleanly in 727.05 seconds. Its disk was extracted
offline into `/tmp/omabit-import-inspection-20261022-05`; no live job remains.
The [copy-case preparation](diagnostics/2026-10-22-import-copy-trials.md) now uses
a fresh separate guest disk. Trial 06 failed during source boot on the host user's
temporary-storage quota, before importing. Its failed original disks are retained;
redundant converted inspection disks were removed only after checking their originals.
The driver now admits and monitors quota headroom. Trial 07 exited before launch
on a corrected evidence-path typo. [Trial 08 passed](diagnostics/2026-10-22-import-copy.md):
the actual runner copied 7 files / 1,016,823,820 bytes across distinct filesystems,
verified every byte, removed the holding source and completed stopped. Explicit
Start/identity/mount/Stop all passed, with clean guest poweroff in 785.44 seconds.
Offline receipts are in `/tmp/omabit-import-inspection-20261022-08`. Its redundant
converted disk and the earlier appliances' regenerable staging trees were removed
after verifying retained original images; original disks/receipts remain.

[Trial 09 passed](diagnostics/2026-10-22-import-interruption.md) the narrower
partial-copy preservation checkpoint using appliance 08. It stopped only the
private guest runner after progress, restarted it, and verified a complete held
source, a 44,564,480-byte partial destination, the interrupted-operation fence,
no implicit runtime and an unchanged `recover-import` refusal. The guest powered
off cleanly in 728.66 seconds; no live VM job remains. Original disks and offline
receipts in `/tmp/omabit-import-inspection-20261022-09` are retained. Its redundant
conversion was removed after original-image and extracted-result verification.

The [runner/CLI restoration checkpoint](diagnostics/2026-10-22-import-restore-integration.md)
now passes source tests. A read-only preview exposes the two moves and digest;
`restore-import` requires operator access, pinned runner, current revision and
explicit reviewed confirmation. Schema22 persists its journal and fences older
readers. Completion atomically retires the failed registration while preserving
replay receipts and the tombstone revision. Six affected race suites passed, along
with original schema21 writer reproduction and downgrade refusal, client/RPC
validation and twelve private built-CLI smoke cases. No production install occurred.

[Trial10 passed actual restoration and re-import](diagnostics/2026-10-22-import-restore-runtime.md)
in the private VM: both retained filesystem objects survived, terminal replay
preserved retirement and the new revision-2 registration, and explicit
Start/identity/exclusive-parent-mount/Stop passed. Clean poweroff took 849.99
seconds. The 69 offline receipts and original disks are retained; no live job
remains. Earlier run09's originals were moved to persistent task evidence with
verified hashes and compatibility directory links to free temporary quota.

[Guided restoration UI now passes](diagnostics/2026-10-22-import-restore-ui.md)
actual offscreen QML with inert dispatch. It requires a fresh actor/runner/revision
and reviewed moves, explicit consent, and exact receipts; uncertain acknowledgements
stay pending. Completion remains visible after the failed instance disappears,
and late callbacks cannot erase terminal proof. The affected collector/collection
race suites, standard model checks, Qt6 parsing and operator skill validation pass.
The final private screenshots/dispatches are in `/tmp/omabit-restore-ui-20261022-03`.

[Trial11 passed actual UI-to-VM restoration](diagnostics/2026-10-22-import-restore-ui-runtime.md)
with appliance10 and the real collector/runner: one QML preview, explicit consent,
one restoration despite a duplicate signal, and a retained receipt after retirement.
Independent checks confirmed both original filesystem objects survived, terminal
replays preserved both retirement and revision2, and fresh re-import plus explicit
Start/identity/exclusive-parent-mount/Stop passed. Clean poweroff took 865.45 seconds.
The private PySide6/Qt package set has 210 signed packages; the host received no
installation. All 74 offline result files and the original disks remain; no live
VM job remains. Qt produced no warnings/errors. The read-only database check
passed after correcting an offline verifier's schema/instance-shape assumptions.

Trial10's originals were relocated with verified hashes to the task evidence
directory, retaining compatibility links. Regenerable packaging trees and the
verified trial11 inspection conversion were removed for quota headroom.
Empty targets, full disks, process crash/power loss, device renumbering and native
Quickshell lifecycle remain open. The preceding reports keep their original scope
and hashes. No production source or service was accessed.

The next bounded feature checkpoint is the first project fake-network transport
proof: two fresh fake galaxies in one private namespace, with distinct HTTP ports,
bidirectional authenticated application messages, then restart and duplicate
identity isolation in another project. Use the [pinned runtime source notes](project-fake-networks.md)
and the existing disposable VM boundary. Native integration and remaining import
failure gates remain explicit acceptance work; this is not release completion.

The [initial operator entry point](diagnostics/2026-10-22-operator-entry.md) now
has portable source instructions and eleven checked help paths. `logs` and
`login-code` help no longer demand live authority. The collector race suite and vet
passed. [Offline task-family guidance](diagnostics/2026-10-22-task-help.md) now passes
both JSON/text namespace smoke checks and the collector race suite. The built-CLI
simulator also passed five tasks: pinned selection, stale-revision review, exact
asynchronous receipt, unavailable-state reporting and replacement-identity refusal.
Transport-loss/import-recovery walkthroughs, structured runtime next actions,
skill distribution and remaining roadmap gates stay in scope. The new
[plugin developer skill](diagnostics/2026-10-22-developer-skill.md) records standalone
sign-in, optional independent approval, direct data flow, named backend installation,
Omart discovery, private chains and complete profile isolation. It is source guidance;
using it to build a qualified application and shipping it remain open.

Full native desktop/profile integration remains distinct: the actual approval UI
was offscreen with inert style values; Theme's desktop adapter and DMs' native
launcher were disabled. Full plugin overlay/hooks/IPC isolation, writes/reconnect,
hosted compatibility and stable/dev/edge switching remain. Next stages should use
the private desktop or inert adapters, never the active desktop or existing ships.

Keep every production exclusion. Plugins retain independent storage, direct data
transport and standalone login; no shared client package or app-data proxy has
been introduced. Wallet work remains research notes only.

The [first private fake-network trial](diagnostics/2026-10-22-fake-network-trials.md)
confirmed namespace isolation, distinct verified HTTP/admin endpoints and independent
sessions, but the first cross-ship message did not complete. Both test ships and
the VM stopped cleanly. Original disks and offline receipts are retained. No
transport/restart pass is claimed. The next fresh run adds same-ship controls and
credential-free failure diagnostics to separate request compatibility from peer
transport; production exclusions remain unchanged.

[The second private network trial](diagnostics/2026-10-22-fake-network-controls.md)
reproduced a 90-second curl timeout in the same-ship control, before peer transport.
Both fresh ships and the VM stopped cleanly; original disks and verified receipts
remain. Diagnose the HTTP/thread path with the exact binary before another Docker
networking trial. The current CLI/skill now explicitly distinguish shared storage
folders from communicating fake networks; offline guidance checks pass.

[The exact-runtime Eyre channel control now passes](diagnostics/2026-10-22-eyre-channel-control.md):
a fresh native fake accepted a nonce-bearing poke and returned the expected deliberate
application rejection through matched SSE receipts. Both channels closed and the
ship stopped cleanly. All diagnostic scopes are inactive. The Docker probe now uses
this tested protocol; its three receipt-matching tests pass, but its revised real
transport/restart trial remains the next gate. No production installation occurred.

[The Docker peer messaging/session-recovery checkpoint now passes](diagnostics/2026-10-22-docker-peer-recovery.md).
A fresh private VM verified both message directions, an intentional remote Gall
rejection, and both directions after cleanly resuming the same pier in a replacement
container with the original sessions. Two local controls and all five peer outcomes
were independently checked; both ships and the VM stopped cleanly. Original disk
and receipt hashes were verified. Six focused offline tests pass.

The direct remote Eyre probe was disproved by HTTP500; the corrected built-in local
Khan thread was first proven with two fresh native peers. The first native pair's
4 GiB OOM remains failed evidence; a fresh pair at 8 GiB passed. Current experiments
are stopped. This completes only the first transport rung and session/process
recovery. Next: persistent application state, subscriptions and client reconnect,
then duplicate-identity project isolation. Runner integration, publication, anchor
recovery, private chains and the rest of the platform roadmap remain unfinished.
Production ships, installed plugins, tmux and `~/temp-ur` remained excluded.


[Fresh native application-state/subscription recovery now passes](diagnostics/2026-10-22-application-state-native.md).
Four unique peer writes persisted, the queued update and original channel survived
the receiver restart, and a fresh subscription reconciled the full state without
replaying writes. Both ships stopped cleanly. Eleven offline checks pass. The same
case is being prepared in a fresh no-NIC Docker VM; Docker acceptance and duplicate
project isolation remain open. Development memories track the fixture corrections
and explicitly superseded hypotheses in development-memory.md.


[Docker application-state/subscription recovery passed](diagnostics/2026-10-22-application-state-docker.md)
on guest16/appliance15 with independent receipt/hash verification and clean VM
poweroff. Original evidence is retained. The next private VM repeats recovery in
two concurrent projects with the same identities and ports, then tests crossed
sessions and independent writes. Runner integration/publication remain separate.

The [four-fake project isolation run](diagnostics/2026-10-22-docker-project-isolation.md)
passed and its verified originals are retained. No project VM remains live. The
next separate HTTP fixture will qualify publication and anchor recovery.

[Login-selection trial21](diagnostics/2026-10-23-login-selection-trial21.md) stopped
before backend code retrieval because the fixture had not waited for a fresh
resource/container observation. Import/healthy Start passed; the fixture is
corrected and the original failed evidence retained. The direct published Urbit
case from [M034](diagnostics/2026-10-23-published-preparation.md) is now running in
its separate fresh VM. Neither case is yet a runtime pass for administrative
discovery or actual publication.

[Current replacement Stop intent and historical validation](diagnostics/2026-10-23-member-resumed-stop-model.md) now pass pure source checks. Adapter/store integration remains.

[Replacement Stop adapter](diagnostics/2026-10-23-member-resumed-stop-adapter.md) now passes source checks; current-generation persistence/scheduling remains in progress.

[Schema34 replacement Stop persistence and private scheduling](diagnostics/2026-10-23-member-current-stop-store.md) pass source, full managed race and original-reader checks. Actual queued replacement Stop and later generations remain pending.

[VM20 readiness trial](diagnostics/2026-10-23-member-resumed-readiness-trial20.md) was refused before admission because the fixture reused an earlier request ID. Its original disks and clean cleanup evidence are retained. A corrected new run is being prepared; actual resumed readiness remains unqualified.

[Later-generation lifecycle adapters](diagnostics/2026-10-23-member-later-generation-adapters.md) pass source checks through generation4. Repeated-resume persistence and later-generation real runtime qualification remain pending.

[VM21 replacement readiness and Stop](diagnostics/2026-10-23-member-current-stop-runtime.md) pass independent runtime verification, replay/reopen and clean shutdown. Later-generation and runner application-state recovery remain.

[Conversion storage admission](diagnostics/2026-10-23-extraction-storage.md) now uses full virtual size plus verification reserve. Quota-failed source regression is retained and being rerun after VM evidence retention.

[Repeated member replacements](diagnostics/2026-10-23-member-repeated-resume-store.md) pass schema35 migration, generations3/4 queue/replay, original-reader refusal and the full managed race suite. Fresh VM22 qualification is next; application-state recovery remains separate.

[Appliance22 generation3 preparation](diagnostics/2026-10-23-member-repeated-preparation.md) independently verifies all171 sources and embedded code. Its fresh offline VM is running; runtime and application-state acceptance remain open.

[Runner application recovery source fixture](diagnostics/2026-10-23-runner-application-recovery-source.md) passes offline phase/transport/routing checks and host refusal. It will use a new appliance after the generation3 run; real application recovery remains pending.

[Appliance23 application recovery preparation](diagnostics/2026-10-23-runner-application-recovery-preparation.md) verifies all174 sources and embedded helpers. It remains unbooted while VM22 runs.

[Offline project-plan CLI](diagnostics/2026-10-23-project-plan-cli.md) passes source, collector race and five namespace-isolated built-binary cases. It reserves/starts nothing and does not contact runners; lifecycle/public API acceptance remains separate.

Current-context Stop and following private queues pass the [focused schema37 source gate](diagnostics/2026-10-23-current-context-lifecycle-store.md); full managed regression now passes in972.505s. Frozen VM25 remains schema36.

The pure coordinated retirement gate passes ([M146](diagnostics/2026-10-23-anchor-retirement-model.md)); its concrete adapter is under test. VM28 remains the pending read-only anchor review trial.

M164 (2026-10-24): aggregate journal and complete history assembly passed focused race/static checks. Next is persistent ownership and common-project fencing, then isolated coordinated runtime repair. See diagnostics/2026-10-24-recovery-journal-model.md.

M166: corrected image31 is independently verified and unbooted, awaiting the full managed regression. Image30 remains retained and was never run. New source work on observation-only interrupted checkpoints is outside frozen31.

M167 supersedes the pending full managed regression in M165/M166. One-checkpoint pure observation passes; persistence/public continuation and runtime repair remain open. Frozen31 excludes the new observation dispatcher.

M170 adds source-qualified continuation boundaries and exact created-ID Start. Full39 is running. VM31 ended with a fixture cleanup ownership mismatch; actual repair/app evidence awaits stopped-disk extraction and independent checks. No runtime pass is claimed.

M171 supersedes the pending full39 regression: full managed race1722.999s and vet pass, with unchanged production inputs. Continuation remains a temporary prototype; VM31 remains failed pending independent inspection.

M172 retains the independently checked failed VM31: coordinated repair and sixteen-write application recovery are verified, but the final fixture Stop gate failed due to omitted recovery labels. Corrected fresh runtime qualification remains next.

M174 freezes/reproduces original39 observation history and verifies real populated40 downgrade refusal. Private continuation source is integrated with restart/rollback tests; race/full regression and runtime continuation remain pending. VM32 is the separate frozen39 cleanup rerun.

M344: actual Docker gateway loads its complete native runtime; public batch evaluation reproduces a30s timeout and passes in71.45s with120s. Fresh corrected gateway is next; keyed Docker ships have not started. See [evaluator evidence](diagnostics/2026-10-25-keyed-docker-evaluator-budget.md).


### 2026-10-27 — Human review implementation resumed

See M452/M453 and [the acceptance list](review-pass-2026-10-27.md). Implemented
source changes: responsive Setup cards, configurable new/existing start resource
policies, compact fleet-backed Application Connections, separate hosted sign-in,
Theme/DMs navigation entry points, and revoked-session Theme cleanup. Focused
Go, Python and actual offscreen QML regressions pass. No live acceptance claimed
yet; project creation/member controls, real managed auth and a fresh review VM
remain the next work. Cleared 5.77 GiB of reproducible old Go build cache with a
removal manifest; retained roots now 32.70 GiB / 40 GiB. Human review VM is stopped
and preserved with its original state and credentials.

### 2026-10-27 fresh guest follow-up

Managed fleet authentication passed for labeled Theme and DMs against fresh
`pass-zod`; Theme Disconnect completed local cleanup and archival through the UI.
Both plugin entry points select the right Application Connections row. Setup
wrapping passes the narrow tiled window review. Existing-ship restart passed with
new minimum/recommended RAM deliberately above the VM's total RAM. M454 records
scope, storage and preservation rules. Guided fake-project creation is now wired
in source and has focused failure/replay tests; deploy into the same agent VM next,
then prove messaging/restart through the normal flow. Do not hand this populated
acceptance overlay to the user as a fresh onboarding image.

### 2026-10-27 human review follow-up: onboarding reliability

M454–M460 track implemented UI/auth/resource changes, actual pair messaging and
persistent/subscription recovery, and the outstanding fresh-project onboarding
failure. Latest diagnostic evidence identified a canceled Docker engine API read
during boot; the successor uses a bounded 15-second probe and preserves timeout
identity. Focused Go/race tests pass. Real panel/monitor QML testing reproduced and
fixed stale project groups; all profile/drain checks pass. No public release or
production installation was changed. First-use VM acceptance and project member
authentication/lifecycle integration are next. Classified failed agent disks are
retired with manifests after compact exports; the original human VM is preserved.

### 2026-10-28: fresh project path accepted

M461 qualifies the corrected first-use path in a fresh native VM: zero prior ships,
projects or Docker images, successful dependency download, namespace/storage,
both launches and both HTTP/initialized-pier readiness checks. Total 347.97 seconds;
no action replay. Live narrow-width project navigation also passed. Reuse this
single pair for project auth/lifecycle integration and keep the clean source base
for the next human review overlay. This does not mark the overall goal complete.
