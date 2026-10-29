# Guided local service updates (unreleased)

This workflow is implemented on `codex/guided-service-upgrades`, after the
published 0.3.0 release. It requires the new matching desktop and runner bundle;
updating only QML against the published 0.3.0 helpers does not enable it. The user has explicitly requested 0.3.1 for this installer change.

## Desktop flow

1. Update the Omabit plugin through Omarchy. Its verified bundle activates matching
   Omabit and collector tools together.
2. Open **Omabit → Setup → Local services & updates**.
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

## Validation and limits

Fixture tests cover read-only status, verified no-op, cancellation, lost replies,
new desktop generations during retry, changed owner/access/target refusal, missing
keys, readiness failure and the privileged identity fence. Real offscreen Qt tests
exercise the buttons, progress, busy controls, error display, retry and DEV guard.
The existing owner-installer and setup regressions pass, including race checks.
These are source-level checks. A fresh disposable host upgrade with the final new
release artifacts remains required before publishing 0.3.1;
production ships are not upgrade-test fixtures.
