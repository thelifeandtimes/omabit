# Host compatibility and schema21 upgrades

**RC.13 introduces schema21, with source/fixture regression coverage.** RC.12 remains
schema20. The [RC.13 release notes](release-rc13.md) distinguish artifact checks
from pending live host/distribution qualification. The [upgrade policy](upgrade-policy.md)
and separate stable-release qualification requirement still apply.

## Why a new schema fence

Schema21 protects per-container `EngineVersion` evidence and the portable
host-engine contract from schema20 and older writers. The transactional migration
accepts schemas11–20 without rewriting instance bodies, signed handoffs, receipts
or identities. Historical containers keep their omitted engine-version field;
opening state never fills it from today's installed package or running daemon.
New launches record the observed daemon version independently of original
provisioning evidence. Compatibility, package authentication and live qualification
are separate facts.

## Existing owner-user schema20 → schema21

First, as an ordinary user, prepare a new rootless bundle with the new trusted
executable. Choose a new output directory: the generated guard/unit templates
must match that executable, so a bundle prepared by RC.12 cannot be reused.

```sh
/absolute/path/to/new/urbit-runner prepare-rootless \
  --output /absolute/path/to/new/rootless-bundle
```

Then use the same executable with the ordinary management installer, the matching
bundle, existing owner and enrolled operator public key:

```sh
sudo /absolute/path/to/new/urbit-runner install-host \
  --owner YOUR_EXISTING_OWNER \
  --bundle /absolute/path/to/new/rootless-bundle \
  --public-key /absolute/path/to/existing/operator.pub \
  --acknowledge-host-changes
```

The existing manager stays online until its durable quiescence protocol admits
the update. Busy work is refused without cancellation. Verified generations and
the activation journal survive the management-only replacement; ships, stopped
intent, owner/profile, custom roots, grants and SSH identity are preserved. A
healthy engine is not restarted to update management or its startup guard.
Schema19 installations use the narrower [one-time offline bridge](management-offline-upgrade.md)
instead, with the same new-bundle preparation before stopping management;
accepting an old database schema does not qualify every host deployment.

## Old Docker29.7.2 guard and package authentication

Normal setup/update authenticates native packages before Docker execution or
service changes and migrates only recognized generated29.7.2 guard/unit contents.
Administrator edits remain preserved for review. The pinned rootless launcher
identity/path stays independent of the currently compatible engine version.

If an OS update left the engine stopped under the old exact-version guard, use
the new executable's local guard-only repair first:

```sh
sudo /absolute/path/to/new/urbit-runner repair-engine-guard \
  --acknowledge-host-changes --format text
```

Repair verifies cached package evidence and replaces known guard/unit contents;
it never starts/restarts the engine or activates management. Missing authentication
material can be retrieved with explicit `--fetch-missing`: native download-only
commands must obtain the exact installed builds from configured repositories and
verify trust and installed bytes. Unavailable builds remain unresolved. Normal
OS-authorized setup permits this retrieval when needed. Neither path installs or
upgrades packages, edits repositories, or imports keys. Coordinate any necessary
engine start/restart separately, then finish the management update.

After target activation, supported Pacman/APT hooks refresh authentication evidence
without restarting services. A hook's successful exit is not authentication proof;
failed verification can block new runtime work. Package/daemon drift remains
visible as restart-pending, without relabeling historical launches. See
[package-hook details and limits](../internal/hostinstall/PACKAGE-HOOKS.md). The
schema19 offline bridge remains management-only; follow it with guard repair and
package verification as described above.

## Retry and downgrade rules

Retry interrupted activation with the same retained target executable, command
and activation journal. Once schema21 may have committed, schema20/older binaries
are incompatible rollback targets. All14 original RC.1–12/preview14/preview18
readers refuse actually migrated schema21 databases. Preserve the journal, fence
and generations for forward recovery; lowering `user_version`, resetting state or
re-enrolling ships is not an upgrade or rollback procedure.
