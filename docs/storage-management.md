# Storage locations, import and deletion

For ordinary `|mount`/file-manager/`|commit` access, see the
[owner-user runtime](owner-user-runtime.md). RC.3 runs the runner and rootless
engine as the existing pier-owning user, so desk files have the same host UID.

Included in **0.1.0-rc.2**, after the published rc.1. Update the runner
and desktop together to expose the new capabilities: `storage-locations`,
`import-pier` and `delete-pier`. Storage support was introduced in schema 14; current
source uses **management schema 16**. The
additive upgrade preserves schema 11–15 records, and older runners refuse the new
database. Release publication and live fleet deployment are separate steps.

**Historical October 14 deployment:** both dedicated-account runners received these
controls before retirement. See the
[deployment record](diagnostics/2026-10-14-development-update.md).

## Choose where piers live

Open **Setup → Import a pier & storage locations** and select the runner.
Under **Storage locations**, add a short name and an absolute directory **on that
runner's host**, for example `ssd` → `/mnt/ssd/urbit-piers`.

- The owner-user default is `/srv/urbit/piers/`.
- Additional directories use the owner's normal permissions. Missing directories
  are created as that user, including under their home. Shortcut selections are
  canonicalized before pinning; owner-controlled group-writable paths are supported.
- Each location is scanned using the standard `fakeships/` and classified
  `livenet/` layout. Its `archive/` is excluded.
- Root identities retain inode and nanosecond birth timestamps across reboot-time
  device-number changes (PR #1). Missing/recreated directories are refused rather
  than falling back to another location or automatically creating a new pier.
- Locations cannot overlap each other or private runner-home state, and an existing
  name cannot silently change its path. Adding a location does not move existing ships.

For example, select a new directory under your existing workspace:

```bash
mkdir -p "$HOME/projects/urbit-piers"
```

For another disk, use a normally user-writable mounted directory. No service-account
ownership transfer or ACL repair is part of this workflow.

The new-ship and import forms offer a **Storage location on runner host** selector.
Paths inside additional locations are represented as, for example,
`@ssd/fakeships/test/pier`. Unqualified paths always keep their original default-root
meaning; choosing another location never relocates an existing registration.

The actual host directory `/mnt/ssd/urbit-piers/fakeships/test` is the exclusive
container mount at `/urbit`. It may contain only `pier/`, which Vere sees as
`/urbit/pier`. Sibling piers, project directories and whole disks are never mounted
as a substitute. Runtime disk admission checks the selected storage filesystem as
well as the management/image filesystem.

The existing **Move…** action accepts locations too; registered-pier moves remain
atomic same-filesystem renames. A cross-filesystem move of an already registered
pier is not the same workflow as importing an external pier.

## Import an existing pier from its current host path

Open **Setup → Import a pier & storage locations → Import an existing pier…**:

1. Select the host and enter the directory containing `.urb`, such as
   `/srv/urbit-imports/my-old-pier`. This is a host path, not a desktop upload path.
2. Stop the source Vere process and any supervisor that might restart it.
3. Choose **Inspect source directory**. Omabit checks access, directory identity,
   Vere lock markers and bounded tree metadata, returning a review fingerprint.
4. Enter the new instance name, expected `~ship`, storage location, destination
   convention path and unused HTTP port. For example, `fakeships/test/pier` for a
   fake, or `livenet/planets/sampel-palnet/pier` for that planet.
5. Confirm the source is stopped and approve **Move and import pier**.
6. Wait for the journaled import operation to succeed. The imported pier remains
   stopped. Choose Start explicitly; existing identity verification applies.

There is no need to manually relocate the pier and then separately enroll it.
The review is bound to the selected runner, source and destination. Source changes,
duplicate claims, links, nested mounts, foreign/shared files and occupied targets
are rejected. The current content bound is 100 GB / 8192 entries / depth 32.

Same-filesystem imports rename the actual pier into its new exclusive parent.
Across filesystems, the runner copies with bounded buffers, verifies both source
and destination content, then removes the held source. Space admission requires the
source's logical size plus 8 GiB on the destination. The old launch path is removed
first by renaming the source into a private `.omabit-import-<id>/pier` holding
directory alongside it. This avoids leaving the ordinary source path runnable
while a target is being prepared.

### Host permissions

The management RPC remains unprivileged and operates with the pier owner's normal
access. Source data must belong to that user; its parent needs read/write/search
permission for the move and sync. Ordinary group-writable desk files are accepted;
private `.urb` state, symlink, hardlink, mount and world-write checks still apply.
Permission errors are shown before import submission. The RPC does not elevate to
take over another user's files. Remote management remains RPC-only, while ordinary
editing takes place as the owner on the runner host.

### Interrupted imports

Imports record phase, original path, holding path and copied-byte/file progress.
An interruption preserves the source/partial destination and fences runtime Start;
it does not retry a copy, erase evidence, or bootstrap the partial target. Inspect
the operation receipt and the source/holding paths shown under registered locations.
Automatic recovery of an import that has already moved its source is not offered.
A failed review/preparation that never moved its source can be explicitly stopped
and its empty registration deleted before a new attempt.
If the move completed but the subsequent enrollment/layout validation failed,
explicit Stop followed by Delete can discard the now-owned destination as well.

## Permanently delete a pier

Use **Delete pier…** on the selected ship's Overview page or under its registered
storage location:

1. Stop the ship and wait for Stop to finish. The operation must remove/reconcile
   its retained containers; an active, running or handoff-fenced source is refused.
2. Review the runner, instance, identity and absolute path.
3. Type the exact instance name and select **Permanently delete pier**.
4. Wait for the operation to succeed. The instance disappears from the fleet and
   its ports/path become available again.

This removes the pier and its exclusive parent, per-instance working files,
pending boot key, owned network and maintenance policy. Audit operation records and
shared runtime/pill caches remain. It is not a reversible trash/restore interface.
An uninitialized registration can be removed without ever booting it. Unknown or
replaced directories are not adopted just to delete them.

Deletion journals its intent, atomically moves the owned mount out of discovery,
then removes that exact tree. A partial deletion remains fenced and can be explicitly
continued with a new request after review. Replaying the same request returns its
original receipt rather than repeating a destructive action.

An instance name may be reused after successful deletion. Its new configuration
revision starts **above the previous instance's final revision** and its private
runtime ID is new. Old reviewed commands and delayed polls cannot affect the
replacement; old delete-request replay cannot delete it. Always use the revision
from the current snapshot instead of assuming that a reused name starts at 1.

## CLI equivalents

All commands require a named, identity-pinned operator connection:

```text
urbit-collector --runner HOST --expected-runner-id RUNNER_ID \
  storage-root-add ssd --path /mnt/ssd/urbit-piers --request-id add-ssd

urbit-collector --runner HOST --expected-runner-id RUNNER_ID \
  preview-pier --source /srv/urbit-imports/old-pier

urbit-collector --runner HOST --expected-runner-id RUNNER_ID \
  import-pier imported --source /srv/urbit-imports/old-pier \
  --path @ssd/fakeships/imported/pier --ship ~zod --fingerprint REVIEW_FINGERPRINT \
  --http-port 18090 --request-id import-reviewed-pier --confirm-stopped

urbit-collector --runner HOST --expected-runner-id RUNNER_ID \
  delete-pier imported --revision CURRENT_REVISION \
  --request-id delete-reviewed-pier --confirm-delete
```

## Verification scope

Fixtures exercise real temporary-directory moves/deletions, injected cross-device
copy, changed and linked sources, replacement storage identities, interrupted copy
and deletion, private-key cleanup, revision-safe name reuse, replay and stale polls.
Mapped QML tests exercise storage selection, reviewed import, typed deletion and
duplicate-click suppression using isolated CLI fixtures. These are not claims of
live deletion/import qualification on the existing fleet.
