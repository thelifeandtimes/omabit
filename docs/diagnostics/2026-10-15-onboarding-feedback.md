# RC.4 onboarding feedback and sourcebox reset

The first real native-install pass on sourcebox exposed ten onboarding issues.
RC.4 implements the requested welcome, first-check wording, automatic destinations,
HTTP explanation, host-checked port suggestion, explicit confirmed first boot,
quieter fleet cards, persisted laptop/always-on choice, healthy-only browser/code
access and working monitor navigation. See [release notes](../release-rc4.md).

## Window and port findings

The running sourcebox shell returned `visible:false` after the old monitor open
action, with no mapped window. A native Qt window with `transientParent: null`
successfully opened in that same shell. Its address was `0x55ab6bab5c10`, mapped
on workspace1; it was then closed. The local focus test now loads the native
service dynamically with no visual parent, matching the third-party loader, and
checks first open, reuse, workspace/special-workspace focus and close/reopen.

The previous port suggestion only considered Omabit's inventory. RC.4's
`GET /v1/port-suggestion` checks the selected runner's registered/prepared port
reservations and actual TCP host binding. The result is admission evidence, not
a socket reservation; registration and Start continue checking conflicts.

Confirmed GUI creation now consumes a verified registration receipt, requires its
positive configuration revision, and submits one independently identified Start.
Unknown/invalid registration acknowledgement never authorizes a boot. The Start
result is matched to runner selection, instance, request and revision; uncertain
results are shown for review rather than blindly retried. Backend CLI registration
semantics remain stopped and explicit.

## Checks

- Go tests/vet, targeted transport/managed race checks and actual bound-listener
  tests; collector tests reject missing pins and mismatched port-reply identities.
- Model checks for placeholder increments, healthy/revision/identity access gating,
  existing conflict, maintenance, storage and replay behavior.
- `scripts/test-onboarding.py`: empty collector welcome, Start Setup route, hidden
  initial Check again, saved laptop choice, remote guide, ship chevron selection,
  disabled access during startup, required name, host-supplied port and custom-only
  pier-path field. Browsing/checking issued no host or ship mutations.
- `scripts/test-panel.py --windows --native`: verified new comet/fake/owned
  registration→Start sequences, request-count audit, rejected registration without
  boot, duplicate prevention, progress view and existing mapped desktop workflows.
- Native-wrapper focus test, snapshot installer and binary-bootstrap fixtures.

## Sourcebox app-owned retirement

Retired runner `runner-0dc9bb1ff3a882d3de7bb32dec55c9c0` had exactly one instance,
`comet`, revision1, at `/srv/urbit/piers/livenet/comets/comet/pier`, with ship
`~nilpur-savdyn-wormel-pitlyd--dozdeb-filner-masbud-wanzod`. The reviewed graceful
Stop completed before its separate owner Docker engine was disabled.

The root-owned receipt and retired system artifacts/pier are in:

`/var/lib/omabit-retired/sourcebox-rc3-feedback-20261015/`

The first helper attempt rejected an invalid collector flag combination before
changing anything. The second completed Stop/plugin/service removal but encountered
EXDEV when copying home-directory Docker overlay state into `/var`. A separate
completion helper verified the stopped state and original unrelated processes,
then used same-parent renames to preserve the original special files intact:

- `~/.local/share/.urbit-runner.retired-rc3-feedback-20261015`
- `~/.config/.omabit.retired-rc3-feedback-20261015`
- `~/.config/.urbit-collector.retired-rc3-feedback-20261015`
- `~/.local/state/.omabit.retired-rc3-feedback-20261015`
- `~/.local/state/.omabit-feedback-window.retired-rc3-feedback-20261015`
- `~/.cache/.omabit.retired-rc3-feedback-20261015`

The partial `rootfs/home/.../urbit-runner` copy under the root retirement directory
is **incomplete**; the same-filesystem retired tree above is the complete original.
Both were retained and identified in the receipt rather than erasing failure evidence.

Completion was verified at **17:57:24 UTC**. All **10 unmanaged Urbit processes**
retained their original PIDs, start ticks and command hashes. Shared Docker state/PID,
passwd/groups/subIDs and `Linger=yes` were preserved. Only the exact Omabit plugin ID
was removed from bar/plugin settings; Tend, theme, Omachan, Omonitor and other plugins
remain. No user account, home, user manager or unrelated Urbit was stopped/deleted.

The plugin, active runner state, owner profile, collector enrollment and native
binary cache are absent for the next user-driven fresh install. The sanitized
completion receipt is `/tmp/sourcebox-omabit-feedback-reset-result.json` on sourcebox.
