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
If the move completed but enrollment failed, first review
[explicit enrollment recovery](#epoch-compatibility-and-interrupted-enrollment).
Stop followed by Delete can remove the sole destination copy. It is not recovery.

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

### Epoch compatibility and interrupted enrollment

Preview and the import worker both validate epochs **before moving the source**,
using the same read-only scanner as enrollment. Supported epochs have format `2`
and provenance exactly `4.6` (stock release) or `4.6-5a7614f` (pinned build).
Trailing metadata whitespace remains allowed. Other releases, arbitrary build
suffixes and other epoch formats are rejected. This does not relax the exact
runtime image/executable pins and never rewrites `epoc.txt` or `vere.txt`.
This is **header compatibility, not boot qualification**: Vere can still reject a
stale snapshot or incompatible persisted state. Do not rewrite metadata to make
such a pier pass. Before a runtime change, stop the source and run its original
Vere binary's `prep` utility on that pier. Require successful completion before
import or Start. This updates the snapshot through Vere, not by editing metadata.

An older runner could move/copy the source, remove the holding directory, and then
fail epoch validation. The narrow `recover-import` operation finishes only a
**prepared, uninitialized external import at `enrolling`** with its original
storage/mount identities and matching retained adoption fingerprint. It requires
uncertain, unverified adoption, no prior adoption completion, no container/network
records, and no independent provisioning, handoff or maintenance uncertainty.
The existing private runtime directories must still be safe and present. An
explicit Stop that only recorded `stopped` does not erase this evidence and does
not prevent recovery. Stop is neither required nor a way to clear the quarantine.

Recovery checks destination identity, ownership, links, mount boundaries, lock
markers, state files, stable tree metadata and compatible epochs. It also requires
**both historical source and holding paths to be absent**, before and after the
destination checks. A duplicate pier (locked or not), unrelated replacement or
inaccessible evidence all require review; recovery never deletes or changes them.
It does not repeat copying/removal, bootstrap, contact Docker, or start Vere.

Absence is checked through descriptor-relative, no-symlink traversal. After
reaching the recorded source filesystem, subsequent mount crossings (including
same-device bind mounts) are refused. Unavailable `openat2`, inaccessible parents,
or a changed source device fail closed; device renumbering can therefore require
manual review. The historical source parent must still be accessible. The legacy
journal has no parent inode/mount ID, so this does not prove historical ancestry
or detect an ordinary same-filesystem parent replacement. Keep external writers
and supervisors stopped; these are bounded checks, not a filesystem freeze.

The original private provisioning fence remains durable through validation and
the completion callback. Only the terminal SQLite transaction publishes completed
import/adoption state, the stopped observation and the succeeded receipt together.
Failure or restart before that commit leaves the original fence intact, including
after polling and Stop. Success keeps `identity_pending` true. A later Start is a
**separate explicit action**, not part of recovery.

The retained adoption fingerprint binds directory identity, not historical file
contents. Legacy journals do not contain a destination content hash; recovery
cannot prove that previously edited LMDB bytes are original. Stable metadata
checks detect changes during validation, not every historical content edit.
Runtime working directories are checked for their expected path, ownership and
safe modes; they are **not bound to historical inodes**. A safely recreated private
workspace passes those checks and is not evidence of historical integrity. Keep
external supervisors stopped and resolve any independent integrity concern before
requesting recovery. Locks are refused, never removed automatically.

#### Reviewed recovery procedure (not deployment instructions)

Use an independently reviewed build advertising `recover-import`. Installing it
or operating a real pier requires separate authorization. All names below are
placeholders.

1. Inspect the identity-pinned runner's snapshot. Confirm the target instance,
   current configuration revision, desired state `stopped`, no active operation,
   and external-import stage `enrolling`. Review the retained enrollment failure;
   do not repeat `import-pier`, edit SQLite, rewrite epoch metadata, or use Delete
   to bypass it. Earlier copy/move phases, unprepared or already initialized
   registrations are deliberately unsupported.
2. With operator access, submit exactly one revision-bound recovery request:

   ```text
   urbit-collector --runner HOST --expected-runner-id RUNNER_ID \
     operate INSTANCE recover-import --revision CURRENT_REVISION \
     --request-id reviewed-import-recovery
   ```

3. Read the same runner's snapshot again. Match the returned operation ID and wait
   for its terminal result. `queued` is not success. A successful result must show
   desired/observed `stopped`, import stage `complete`, `recovery_required: false`,
   and `identity_pending: true`; no runtime was started. Stop here.
4. If the reply is lost, repeat the **identical** request to retrieve its receipt,
   not re-execute recovery. A failed/interrupted request retains its result. After
   reviewing the cause and a fresh revision, a new request ID permits another
   explicit attempt only if the narrow eligibility checks still hold. A completed
   import rejects a new recovery request. Do not widen recovery by changing state
   records or deleting files.

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
