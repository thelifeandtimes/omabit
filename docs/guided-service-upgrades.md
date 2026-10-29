# Guided local service updates

Available in matching **0.3.1 or later** desktop and runner bundles. Updating only QML
against older helpers does not enable this flow. Use the ordinary native plugin
update, then the service-update button described below.

## Desktop flow

1. Update the Omabit plugin through Omarchy. Its verified bundle activates matching
   Omabit and collector tools together.
2. In **0.3.2 or later**, open the Omabit menubar widget. It checks local service
   versions and offers **Update Dependencies** when an update is needed. Select it
   to open progress and begin the verified update; approve the OS prompt when shown.
   A cancelled attempt remains visible for checking and retry. The notice clears
   when installed services match.

   The same controls are available under **Setup → Local services & updates**.
   For this alternate route, continue below.
3. In **Service updates**, choose **Check services**. This reads the existing
   local enrollment and executable metadata without downloading or installing.
4. Choose **Update services** if offered. The panel shows progress while it
   verifies the pinned rootless launcher dependencies (reusing matching installed
   upstream files, or downloading them when missing/changed), then asks for
   OS authorization to update the existing owner installation.
5. Wait for verified completion and the refreshed fleet. A cancelled or interrupted
   attempt offers **Retry service update** after **Check services**. Retrying keeps
   the original target even if a newer desktop bundle has arrived. A subsequent
   update to that newer bundle is a separate explicit action.

The installer preserves runner identity, pier roots, ship settings, access grants
and stopped/running intent. Busy management refuses rather than cancelling pending
work. It verifies the actual runner identity under the host setup lock. A healthy
rootless engine is not restarted for a management-only update. OS packages and
Vere runtime updates remain separate operations, not implicit side effects.

This card handles the local owner-user installation created by Omabit Setup.
Remote runners and the historical offline schema19 bridge are not offered through
it. An unavailable installation without its matching pending activation journal
requires diagnosis, not a reset. DEV profiles with local setup disabled cannot
use these controls to upgrade the production host.

## Retained state and command interface

The desktop writes a version1 `service-upgrade.json` alongside the existing local
setup receipt. It binds owner metadata, runner identity, enrollment and access-file
digests to one target executable and the original installed generation. An unexpected
third generation refuses instead of replacing another updater's work. `service-upgrade/` holds only that executable,
its verified launcher bundle and the existing public key. Private keys are never
copied into the attempt. Completed staging is replaced only when a different
explicitly approved update starts; retries do not accumulate new bundles.
Unknown or changed state refuses without erasing the original attempt. The
privileged installer keeps its existing root-owned activation journal and schema
migration protocol unchanged. Never edit either journal to force a retry.

The UI uses `omabit service-upgrade-status` and
`omabit service-upgrade --expected-runner-id ID --target-sha256 SHA --acknowledge-host-changes`.
Both expose offline `--help`; the status command has no installation effects.
`urbit-runner install-host --expected-runner-id ID` fences the existing host and
accepts an unavailable manager only for a matching retained activation attempt.

## Optional live rehearsal

After the matching 0.3.1 plugin is installed on an owner-user host:

1. Open **Setup → Local services & updates** and select **Check services**.
   Confirm an update is offered for the existing local runner.
2. Select **Update services** and cancel the OS password dialog once. The error
   should leave the fleet available. Check again: the action should read
   **Retry service update**.
3. Retry, approve the OS prompt, and watch progress until verified completion.
   Confirm the running ship is still usable and a stopped ship stays stopped.
4. Check again. The panel should say the local services match this desktop bundle
   and hide the update action. Reopen Omabit and confirm the same fleet appears.

The native plugin supplies the matching Omabit/collector executables; the explicit
button updates the local host runner. Separately copied command-line binaries and
remote runners are not silently overwritten. Docker/OS package changes are outside
this button. Review the real authentication dialog and Omarchy update/rescan on a
disposable machine; automated guest authorization does not exercise human input.

## Validation and limits

Fixture tests cover read-only status, verified no-op, cancellation, lost replies,
new desktop generations during retry, changed owner/access/target refusal, missing
keys, readiness failure and the privileged identity fence. Real offscreen Qt tests
exercise the buttons, progress, busy controls, error display, retry and DEV guard.
The existing owner-installer and setup regressions pass, including race checks.
The frozen review candidate additionally passes both requested disposable-host
paths: published0.2.0 desktop/services → 0.3.1, and published0.3.0 desktop with
0.2.0 services → 0.3.1. Both preserve running engine/container processes and
stopped intent through actual QML-button authorization denial and retained retry.
See [the rehearsal evidence](releases/0.3.1/rehearsal.json) and memories M484/M485
for fixture boundaries and the test-only screenshot-export correction. This does not claim coverage of remote upgrades, arbitrary copied CLI
binaries, human password entry, power cuts or other OS/package stacks.

## Desktop activation recovery in0.3.2

If0.3.1 reports “Candidate UI could not load; previous release retained,” update
the plugin to0.3.2 through Omarchy. The new bundle explicitly registers the service
update QML components. Normal activation retains enrollment and preferences; no
ship reset or service reinstallation is needed to recover the desktop. A running
old UI is a retained fallback, not evidence that underlying services were upgraded.

The dependency notice handles the local Omabit runner, using the matching bundled
CLI and collector. It does not update remote hosts, manually copied CLI binaries,
OS packages or Vere. Opening the menubar only checks; it never installs automatically.
