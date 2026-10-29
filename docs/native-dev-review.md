# Native DEV review — reliability and release preparation, October 29

Review build **0.2.0-dev.20261028.3**, sourcebox. Use **Omabit · DEV**, **Network Theme — DEV**, and **Urbit DMs — DEV** in the main desktop. This is unqualified development, not a published release. Theme and DMs now use presentation generation `ui-20261029-cohesion-8`. This guide supersedes the previous feedback-pass state in M467; release preparation is tracked in M470–M473.

## What changed

- Theme and DMs now share native header and sign-in components,440-unit popup width,720-unit height cap,12-unit sections and8-unit controls. Both show **Sign In**, then **Connect with Omabit** and **Enter direct credentials**. Light/dark, narrow, long-identity and sign-in views were compared with actual Qt/native controls.

- Project members now show fresh container RAM, CPU, memory/CPU limits and process counts. Samples remain bounded in memory; polling writes no metrics history. Project observations run concurrently and do not hold the lifecycle mutation lock during network reads.
- Recovered `demo-nec` from its interrupted start and `empty-zod` from its interrupted first addition. Continuation binds the exact saved attempt; it does not reinitialize the pier or repeat an unknown container creation. A consumed, private, one-shot boot-key mount no longer blocks unrelated member restarts.
- Empty projects accept their first ship. New project labels are unique on the runner, case-insensitively. The two already-existing `review-empty` records were preserved; selectors distinguish them by ID suffix. The one containing `empty-zod` ends in `36d39f10`.
- Stop requests no longer let Docker escalate a graceful timeout to SIGKILL. Fresh keyed runtimes use the identity-checked private Eyre hood exit path; resume runtimes use SIGTERM and exact-container observation. A timeout retains the runtime and reports failure.
- Remotely Hosted Urbits shows external-only cards with identity, URL and sign-in state. Add asks for @p, URL and +code; the identity supplies its unique connection name. Managed fleet ships remain in Application Connections. The extra hosted-navigation button is removed there, and approval uses a right-aligned toggle beside Connect/Disconnect.
- Disconnect clears the running plugin's old identity/data and presents its login flow. DMs preserves unknown message outcomes, ignores delayed old-session data, and detects an immediately replaced Omabit session even for the same ship. A replacement first returns to login instead of silently switching an active client.
- Monitor selection shows a loading indicator during the observation gap. Logs preserve @p and +code; boot keys, cookies, authorization headers and unsafe terminal controls remain filtered.

## Current ships and manual checks

| Project/type | Instance | Identity | State after testing | HTTP |
| --- | --- | --- | --- | --- |
| demo | demo-zod | ~zod | Running, healthy | http://127.0.0.1:18080 |
| demo | demo-nec | ~nec | Running, healthy | http://127.0.0.1:18082 |
| review-empty | empty-zod | ~zod | Running, healthy | http://127.0.0.1:18084 |
| Moon | lagbet-bisful-dozzod-sarlev | ~lagbet-bisful-dozzod-sarlev | Cleanly stopped, exit 0 | 18083 when running |
| Moon | marrun-dalmeg-dozzod-sarlev | ~marrun-dalmeg-dozzod-sarlev | Cleanly stopped, exit 0 | 18085 when running |

1. Compare **Network Theme — DEV** and **Urbit DMs — DEV** popups and sign-in views: matching headers, spacing, primary/secondary sign-in choices, narrow layouts and keyboard focus. Both start signed out.
2. In **Omabit · DEV**, select each project ship. Confirm RAM/CPU appear and update, logs remain readable, and Open ship / Copy +code work. A first CPU sample can be unknown until the next observation. Start/Stop was tested on all three retained piers.
3. Open **Setup → Application Connections**. Select a fake, approve both DEV plugins, and connect them. Open each plugin. Disconnect DMs: its login should reappear and Theme should retain its separate connection. The test sessions from this pass have been revoked and archived, leaving both DEV plugins signed out.
4. Open **Remotely Hosted Urbits**. Managed moons should not appear here. The existing hosted `~sarlev` registration is retained; it was not authenticated or operated on during this pass. Check Add's three fields and the card layout.
5. In **Project networks**, entering `demo` or `review-empty` should block a new duplicate. Existing duplicates are distinguished by ID suffix; no project or pier was deleted to repair the display.
6. Start either review moon if desired. Both now restart from their retained piers and stop cleanly. The first restart after the old forced shutdown replayed saved events for several minutes before HTTP became ready; a running container alone does not mean the ship is ready.
7. Recheck the small loading interval after creating/selecting a ship and the layout at narrow widths. The previous key paste/path, external Browse chooser, recommended comet settings and single-runner compact form remain available.

## Verified and remaining limits

Both moons completed retained-pier restart and clean Stop (exit 0). All three project ships completed clean Stop/Start, private code retrieval, bounded log reads and fresh metrics. `demo-zod` and `demo-nec` exchanged matching remote Gall acknowledgements in both directions after restart. Temporary messaging sessions were logged out.

Both DEV plugins accepted independent sessions on the fake `demo-zod`. A revocation overlapping its restart remained explicitly pending; recovery finished that same revocation, reconciled local state, and archived the receipt. DMs closed its exact channel and cleared its account while Theme remained connected. The personal outbound-message journal's hash was unchanged. Theme was then independently disconnected and reconciled. No personal DM was sent or retried.

The full runtime and manager Go suites pass, as do the relevant collector/client/collection regressions. A race-detector test proves slow project observation does not hold the mutation lock. DMs: all 152 Python tests and 7 actual-QML checks pass. Actual Omabit Service/Panel/Monitor and forms, plus Theme's 20 QML/helper checks, pass. Fresh keyed shutdown is covered by identity/lost-response/timeout tests and the earlier qualified hood exit experiment; this pass did not boot another keyed identity solely to repeat that check.

Expanded project anchor/whole-engine recovery, project maintenance/epoch integration and existing-pier project import remain unfinished. Security/privacy review is tracked explicitly: keyring lock/reboot, revocation, same-UID plugin access, metadata/log retention and upgrade behavior. Independent sessions are ordinary Eyre owner sessions, not desk-specific permissions or a malicious-plugin sandbox. Wallet implementation remains out of scope.

## Roadmap status

| Area | Implemented / demonstrated | Still open |
| --- | --- | --- |
| Ship management/import | Existing manager, identity-based import layout, actionable import diagnostics, interrupted-copy restoration, new/existing configurable resource policies | Remaining native failure cases, valuable-pier manual review, release-wide upgrade qualification |
| CLI and skills | Bundled operator/developer skills and style conventions,130 offline help entries,57-command walkthrough, capability checks, project create/add/status/lifecycle commands | Richer guided recovery for newly expanded projects; public release gate still pending |
| Fake projects | Shared namespace networking, project grouping, empty/new/existing-project creation, live member controls/auth/metrics, clean member restart, earlier persistent state/subscription and isolation tests | Expanded anchor/engine recovery, project maintenance, existing-pier project import |
| Shared plugin authentication | Independent Theme/DMs sessions, managed/hosted login, matching plugin entry, direct plugin-to-ship traffic, independent disconnect with visible client reset and interrupted-revocation recovery | Broader hosted environments, real backend update combinations, keyring lock/reboot qualification |
| App setup/distribution | Plugin-owned setup/compatibility guidance and earlier Tend proof; Omart remains marketplace owner | General public-publisher/update-order workflow; desk-served artifacts and Urgit distribution are optional future experiments |
| Developer kit | Separate DEV runtime/plugins and profile state; private collection-switch/recovery prototypes; private-chain keyed peer/state/subscription experiments | Complete public runner private-devnet path, supported turnkey packaging and installed stable/dev/edge collection UX |

Wallet implementation and desk-scoped Eyre permissions are not implemented and remain outside initial scope. Wallet/key-storage research continues; same-UID plugin namespacing is not a malicious-code sandbox.

## Paths, retention and commands

Ship data: `/srv/dev/urbit/piers`. Engine/runner state: `/srv/dev/urbit/state`. Collector/plugin state: `/srv/dev/urbit/desktop`. Binaries: `/srv/dev/urbit/bin`.

```sh
omarchy-shell thelifeandtimes.omabit.dev.monitor open
omarchy-shell thelifeandtimes.omabit.dev.setup addShip
urbit-collector-dev --format json inventory
urbit-collector-dev --format text project-operate --help
urbit-runner-dev resource-policy show
```

Omabit retains `ui-20261028-reliability-4`; Theme and DMs use `ui-20261029-cohesion-8`. DEV-specific state, IPC and keyring identities remain isolated. Production runner/engine PIDs 3050/2166 retain their October 21 start times. No tmux ship or `~/temp-ur` was inspected or operated on.

The earlier native reliability pass created no VM or full evidence bundle. Release upgrade checks now use bounded offline disposable guests, preserving compact evidence and retiring superseded generated disks. Moon replay/snapshots grew the retained DEV piers; historical Zig/npm build caches and inactive npm dependencies were pruned with a manifest, preserving all piers, VM disks/backing chains, sources, lockfiles and verified runtime artifacts. Current combined allocation is recorded in [the compact reliability receipt](diagnostics/2026-10-28-native-reliability.json); do not launch another large experiment without measuring and making room first.
