# Upgrade to Omabit 0.3.0

**For 0.3.1, use the [button-based local upgrade](guided-service-upgrades.md).**
The command procedure below describes the older0.3.0 release.

The [published qualification record](releases/0.3.0/qualification.json) identifies
the frozen binaries and the completed non-destructive upgrade rehearsal.

The supported stable sources are **0.1.0 and 0.2.0**, both owner-user installations
with schema21. The live-qualified path is0.1.0→0.2.0→0.3.0; users still
on0.1.0 should first follow the0.2.0 upgrade procedure. Directly skipping0.2.0 is
not newly live-qualified. The target uses **runner schema47**, management protocol1 and
desktop/bootstrap state1. The new project journals migrate transactionally;
opening a database does not start ships or alter existing settings, ports,
identities, grants, storage roots or operation receipts. Older runners refuse
schema47. Returning the desktop to an older version cannot downgrade that state.

## Desktop and host are separate updates

Update the native plugin through Omarchy's usual plugin updater. Omabit's retained
bootstrap verifies the pinned bundle and activates matching UI/CLI generations.
It preserves the selected profile, fleet enrollments and preferences. CLI docs and
operator/developer skills are included in the verified bundle.

A desktop update does **not** silently replace a local or remote runner. Setup
shows its actual capabilities. Finish pending management work before updating the
runner; a busy manager must refuse quiescence without cancelling that work.

Use the newly downloaded, verified runner executable and a new matching launcher
bundle. Keep the existing owner and operator public key:

```sh
/absolute/path/to/0.3.0/urbit-runner prepare-rootless \
  --output /absolute/path/to/new/rootless-bundle

sudo /absolute/path/to/0.3.0/urbit-runner install-host \
  --owner YOUR_EXISTING_OWNER \
  --bundle /absolute/path/to/new/rootless-bundle \
  --public-key /absolute/path/to/existing/operator.pub \
  --acknowledge-host-changes
```

The management activation journal retains exact executable generations and fences
new mutations during replacement. A management-only upgrade should preserve the
healthy engine and running containers, while stopped ships stay stopped. Package
verification may require downloads of the exact installed native packages; it
does not authorize changing OS packages or an engine restart.

Retry an interrupted update with the **same target executable and bundle**. Do not
lower SQLite's schema marker, delete journals, recreate profiles, or re-enroll
piers to work around an activation error. Preserve the installation and inspect
the reported phase. Once migration commits, recovery must proceed with compatible
new code; an older binary is not a valid rollback target.

Older RC19 deployments have the separate historical
[offline bridge](management-offline-upgrade.md). Unpublished prototype schemas and
other Linux distributions are not newly qualified by this release.

## Application sessions and companion plugins

Theme and DMs continue to talk directly to the ship with independent sessions.
Omabit assists approval and cleanup. Their receiver journals retain pending and
uncertain actions; an upgrade never retries a personal message automatically.
Application history version2 is a forward migration. Keep archived receipts and
retained receiver generations; lowering the version cannot safely undo it.

Standalone URL/+code sign-in remains available without Omabit. Eyre sessions are
not desk-scoped authorization; the security/privacy audit remains open. These
interfaces do not claim malicious-plugin isolation or wallet-grade key storage.

## Evidence

The release qualification links the original-writer migration/replay/interruption
suite, same-transaction corruption regressions, mixed-client observation tests,
and an offline guest using the actual published0.1.0/0.2.0 binaries. That guest
creates only new fake identities and verifies running/stopped intent, runtime
process continuity, repeat activation and old-reader refusal against the frozen
0.3.0 artifacts. The successful guest then stopped its own fake and powered off.
Source fixtures cover interruption boundaries; this guest does not claim power-cut
qualification or other distribution/package combinations.
