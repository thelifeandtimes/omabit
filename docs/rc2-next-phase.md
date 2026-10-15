# RC.2, clean installation and multiple-collector plan

Review date: **2026-10-15 UTC**. Publication and retirement have not been executed
by this review. The recommended next phase follows the user's proposed sequence,
with release preparation first and an explicit fresh remote installation before
multi-collector testing.

**Execution direction received afterward:** publish RC.2 and retire the old
installations, then stop with the verified bundle staged on **sourcebox**. The user
will run its plugin installer and first-time setup manually. Sourcebox's desktop
will be the first fresh collector; sourceport can follow as the independent second
collector. The steps below retain the original review's technical scope.

## Current position

- Both runners and the desktop bundle contain `0.1.0-rc.2-dev`, schema 15.
- The [real hard-reboot check passed](diagnostics/2026-10-15-hard-reboot-acceptance.md):
  three automatic resumes, preserved identities/configurations and stable health;
  the intentional stop and moved-source fence were preserved.
- Full Go tests/vet and focused reboot race checks passed before deployment.
  Storage/import/delete have filesystem and mapped-UI fixtures; actual cross-device
  import/destructive live qualification is not implied by those tests.
- GitHub still publishes only **v0.1.0-rc.1**. HEAD is the PR #1 merge; subsequent
  storage/UI/reboot source and diagnostic records remain uncommitted. Latest green
  master CI covers that committed merge, not the deployed dirty development tree.
- The release builder already produces the three binaries, desktop archive,
  provenance, checksums and a frozen pill, and refuses a dirty source tree.
- The release/clean-install guides still pin rc.1. The sourcebox retirement recipe
  is reviewed but not a supported/live-qualified uninstaller. General installation
  text also needs current schema, startup and networking descriptions.

## 1. Prepare and publish v0.1.0-rc.2

1. Preserve the new reboot evidence before resetting either host.
2. Review the accumulated changes; align manifest/displayed version, pill defaults,
   packaged README and release notes to `0.1.0-rc.2` / schema 15. The package verifier
   requires the default pill URL to reference the candidate's own versioned asset.
3. Choose pill contents explicitly. The existing frozen draft lacks MCP/Omachan and
   has recorded Grubbery/Azimuth issues. It can remain an explicitly draft RC asset
   while lifecycle tests use bare bootstrap. A provider replacement needs a new
   frozen hash and qualification. Preserve published rc.1 assets.
4. Run a small disposable import/delete smoke gate and outbound DNS/HTTPS checks,
   then source/race/model/install checks and the real-OpenSSH multi-key fixture.
   Keep qualification claims scoped to exercised paths.
5. Commit intended source/docs, push to GitHub and obtain green checks. Build from
   the clean committed tree with the pinned Go toolchain, run `scripts/test-release.py`
   and attach matching qualification metadata.
6. Tag/publish a **prerelease** with the archive, standalone binaries, pill, release
   and qualification metadata, and `SHA256SUMS`. Download and verify the published
   bytes. Use those exact artifacts for the fresh-install exercises; later fixes
   become a distinct candidate.

Management qualification need not wait for an all-apps-qualified pill. The release
candidate must not be described as already having passed the future clean-install
and multi-collector experience gates.

## 2. Retire the sourcebox runner installation

Update the existing retirement recipe against current paths, services, routes and
the healthy fleet. Stop dedicated Omabit entry points and its dedicated engine
cleanly, retain an exact root-only diagnostic retirement copy, then remove app-owned
account/services/install paths from active locations.

An empty target means no active old database, pier catalogue, container store,
access grants or host identity to adopt. Preserve unrelated native/tmux ships,
Tend, shared packages/system Docker, administrator SSH and unrelated Tailscale routes.
Keep the local moved-source pier inactive. Verify sourcebox is empty and ready for
a fresh headless install; retained data is diagnostic history, not automatic restore.

## 3. Exercise a genuinely fresh workstation install

Prepare a sourceport-specific retirement checklist. A clean install of **all three
components** requires both host and per-user state retired:

- Dedicated local runner account/home, engine, database, services and access state.
- Owned Omabit plugin/bundled binaries, collector enrollments and app-owned setup
  keys/browser/display preferences.

This retires the local managed fleet, including the owned-moon pier. Agree the pier
disposition before executing the reset. Preserve the checkout, unrelated desktop
configuration and Tend's legacy command compatibility.

Install solely from verified GitHub assets, without development binary/path
overrides, using the ordinary first-time setup. Confirm new runner and SSH host
identities, empty inventory/history and current schema/capabilities. Test bare
fake/comet boot, Stop/Start, browser access, state/resource display and management
restart. Replacing just the plugin or keeping the old database tests an upgrade,
not a clean installation. Shared OS prerequisites may remain installed.

## 4. Freshly install sourcebox and enroll it

This explicit step is needed before remote multi-collector testing. Install the
same candidate's headless runner, pin the new management SSH and runner identities,
and enroll it from the freshly installed workstation collector. Remote provisioning
is an administrator CLI flow; **Connect another runner** connects an existing host.

Confirm empty remote state before creating test ships. Verify RPC, local ship HTTP
and tailnet/browser access independently: a healthy runner does not automatically
create a Tailscale Serve route for each ship port.

## 5. Qualify multiple independent collectors against one runner

The runner already owns authoritative configuration, jobs, identity and history.
Collectors own independent connection registries/SSH credentials; there is no
shared collector database or replicated `runner.sqlite` to introduce.

Use two distinct operator keys/profiles and one observer profile. CLI profiles can
use separate `--config-dir` directories; independent GUI sessions use separate
desktop users or machines. Reusing the same default profile/key does not test
independent actors. Grants are currently administered on the host, without a full
desktop grant-management workflow. Roles apply runner-wide, not per ship.

Check common state/revisions/operation IDs; cross-client action visibility and key
attribution; one-winner stale-edit conflicts; overlapping action exclusion; accepted
jobs surviving collector disconnect; exact request replay; observer restrictions;
revocation on persistent sessions; reboot/reconnect convergence; and persistent
watches without SSH/PAM/session churn.

Existing tests include concurrent configuration/replay, real OpenSSH with two
operators and an observer, revocation, and persistent watch identity/role refresh.
There is also live remote caller-loss acceptance. This phase qualifies the complete
current-release, real-ship, independent-collector experience. Findings can drive RC.3.

**Next action:** RC.2 preparation/freeze, publication/download verification, the
two-host clean-install walkthrough, then the multi-collector matrix. No manual
recovery is needed for the latest reboot.
