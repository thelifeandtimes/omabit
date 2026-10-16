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

The updated guided flow additionally advertises `guided-pier-import` and
`named-pier-directories`. Update collector, runner and desktop together for these
capabilities. Operational work on the user's hosts follows [protected ships](protected-ships.md).

**Historical October 14 deployment:** both dedicated-account runners received these
controls before retirement. See the
[deployment record](diagnostics/2026-10-14-development-update.md).

## Choose where piers live

Open **Setup → Import a pier & storage locations** and select the runner.
Expand **Manage storage locations** to add a short name and an absolute directory **on that
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
`@ssd/fakeships/test/zod`. Unqualified paths always keep their original default-root
meaning; choosing another location never relocates an existing registration.

The actual host directory `/mnt/ssd/urbit-piers/fakeships/test` is the exclusive
container mount at `/urbit`. It may contain only its selected pier child (here
`zod/`, which Vere sees as `/urbit/zod`). Sibling piers, project directories and whole disks are never mounted
as a substitute. Runtime disk admission checks the selected storage filesystem as
well as the management/image filesystem.

The existing **Move…** action accepts locations too; registered-pier moves remain
atomic same-filesystem renames. A cross-filesystem move of an already registered
pier is not the same workflow as importing an external pier.

## Import an existing pier from its current host path

Open **Setup → Import a pier & storage locations → Import an existing pier…**:

1. Stop the source Vere process and any supervisor that might restart it.
2. Choose the directory containing `.urb`, such as `/srv/urbit-imports/my-old-pier`.
   Local hosts offer Browse; remote paths are entered on the selected runner host.
3. Choose an already configured storage location (default `/srv/urbit/piers`).
4. Choose **Inspect pier**. Omabit reads public identity and fake/live metadata,
   checks the stopped tree and compatible epoch history, derives an unused instance
   name and `<instance>/<ship-name>` destination, and suggests an available HTTP port.
   Review these read-only details; no identity/path/port transcription is required.
5. Confirm the source is stopped everywhere and choose **Move and import pier**
   (or **Enroll stopped pier** if it is already in place).
6. Follow the loading bar, copied-byte totals and durable step log. The imported pier remains
   stopped. Choose Start explicitly; existing identity verification applies.

There is no need to manually relocate the pier and then separately enroll it.
The review is bound to the selected runner, source and destination. Source changes,
duplicate claims, changed identities/networks, links, nested mounts, foreign/shared files and occupied targets
are rejected. The current content bound is 100 GB / 8192 entries / depth 32.

Identity is read without booting Vere or opening a writable LMDB environment: a
bounded reader follows only public `META` keys in the supported 64-bit little-endian
LMDB v1 / Vere disk-v3 format. Folder names are not identity evidence. Complete
historical epoch-v1 (`north.bin`/`south.bin`) and epoch-v2 (`image.bin`) history can
be retained when the newest epoch already uses Vere 4.6/v2. Mixed history receives
an unqualified inventory source and does not inherit the qualified pruning/roll
contract. Missing or unsupported current snapshots are reported before moving.

**Testnet is a stub:** it means an alternate Ethereum/Azimuth RPC endpoint, not a
fakeship. Fake/live metadata cannot identify that endpoint. The UI explains this;
testnet configuration and detection are not currently supported.

Same-filesystem imports rename the actual pier into its new exclusive parent.
An existing exclusive parent may be reused to rename a legacy `pier` leaf to its
detected ship name; a pier already at the derived destination is enrolled in place.
Across filesystems, the runner copies with bounded buffers, verifies both source
and destination content, then removes the held source. Same-filesystem moves require
8 GiB headroom; copies additionally require the source's logical size. The old launch path is removed
first by renaming the source into a private `.omabit-import-<id>/pier` holding
directory alongside it. This avoids leaving the ordinary source path runnable
while a target is being prepared.

Progress is carried by the existing inventory connection. Phase receipts survive
fast renames, and large-file copy updates are throttled rather than spawning new
collectors or writing a journal record for every buffer. Errors retain the source/
holding/partial target for explicit review; automatic retries are not performed.

### Host permissions

The management RPC remains unprivileged and operates with the pier owner's normal
access. Source data must belong to that user; its parent needs read/write/search
permission for the move and sync. Ordinary group-writable desk files are accepted;
private `.urb` state, symlink, hardlink, mount and world-write checks still apply.
Vere's inert `.run` / `.bin/...` regular-file hardlinks are accepted only when all
links are proven to be inside the reviewed pier. Cross-filesystem copies preserve
their bytes as independent private files; the docked executables are never used.
Shared `.urb` state and links to anything outside the pier remain refused.
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
  preview-pier --source /srv/urbit-imports/old-pier --storage-root ssd

urbit-collector --runner HOST --expected-runner-id RUNNER_ID \
  import-pier imported --source /srv/urbit-imports/old-pier \
  --path @ssd/fakeships/imported/zod --ship ~zod --fingerprint REVIEW_FINGERPRINT \
  --http-port 18090 --request-id import-reviewed-pier --confirm-stopped --identify

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
