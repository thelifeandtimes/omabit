# Omabit 0.3.2 — desktop activation and dependency updates

This patch explicitly registers the service-update QML types omitted in0.3.1 and
adds **Update Dependencies** to the Omabit menubar widget.

## Update

Update the Omabit plugin using Omarchy's normal plugin updater. If0.3.1 previously
reported **Candidate UI could not load; previous release retained**, the verified
0.3.2 bundle provides a new activation attempt. Existing desktop enrollment and
preferences stay in place.

Open the Omabit menubar widget. It checks local service versions without installing
anything. When an update is available, choose **Update Dependencies**. The button
opens local-service progress and starts the verified update; approve the OS prompt
when asked. Cancellation leaves a visible check/retry path. Once the installed
services match, the notice disappears. The same controls remain under
**Setup → Local services & updates**.

No reset, re-enrollment or batches of service commands are needed. Remote runners,
manually copied CLI binaries, Docker/OS package upgrades and Vere runtime changes
remain separate operations. Test profiles with local host setup disabled cannot
use this button to modify the production runner.

## Compatibility and validation

Runner schema47, the installer, Go runtime code, authentication and persistent
formats are unchanged from0.3.1. The bundle contains freshly built, matching tools
with the frozen source identity. The release-policy source digest also includes
release-governance code that changed after the0.3.1 artifact freeze; that difference
is not a runner or installer change.

The exact desktop upgrade evidence and limits are recorded in
[qualification.json](releases/0.3.2/qualification.json). Earlier0.2.0-service guided
upgrade rehearsals remain scoped evidence for the unchanged installer/backend;
this patch does not claim a new live-host migration rehearsal. Users on0.1.0 must
still upgrade through0.2.0. No schema downgrade is supported.

The observed host error named `ServiceUpgradePane is not a type`. Both component
files were present but absent from the QML registry. The correction includes a CI
check for complete component registration. Fresh/simple hot-update fixtures also
load the old0.3.1 bundle, so they do not reproduce every long-lived shell condition
behind the original failure. Failed-candidate recovery is separately exercised
with an explicitly broken type; that synthetic failure is not presented as the
original host failure.
