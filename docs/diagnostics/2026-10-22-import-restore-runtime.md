# Real partial-copy restoration and re-import

The private VM passed the complete CLI restoration case on 2026-10-22 and powered
off cleanly in 849.99 seconds. This advances the
[source integration checkpoint](2026-10-22-import-restore-integration.md).
Exact source, artifact, disk and receipt hashes are in the
[machine-readable evidence](2026-10-22-import-restore-runtime.json).
The earlier integration report's 30 source files are unchanged.

## Behavior demonstrated

The test created a new fake `~zod`, verified its HTTP identity, stopped it cleanly
and removed only its source container. Its pier contained 7 files totaling
1,016,823,820 bytes on the separate guest source disk.

1. Guided import selected `fakeships/zod/zod`. The driver stopped only the guest
   runner after observing 31,719,424 copied bytes. At shutdown the partial target
   held 60,030,976 bytes. Restart retained the complete source, failed operation
   and recovery fence, with no implicit runtime.
2. The existing `recover-import` action refused this partial copy without changing
   either tree. The new `preview-import-restore` returned the exact source,
   original path, exclusive target parent, archive destination and review digest.
   The read-only preview's counts matched independent tree observations.
3. Explicit `restore-import` approval returned the **same source inode** and all
   original contents/metadata to its old path. The **same partial destination
   parent inode** and its complete contents moved to the archive. The empty
   holding parent remains. The failed registration was retired; no ship started.
4. Replaying the restoration and original interrupted import returned their exact
   terminal receipts. Neither replay recreated a registration or moved data.
5. A new guided import with a fresh request succeeded under the same public name
   at configuration revision **2**. Replaying both old revision-1 requests again
   left this new registration and its pier unchanged.
6. Explicit Start reached healthy `~zod`, mounting only
   `/srv/urbit/piers/fakeships/zod` at `/urbit` and selecting `/urbit/zod`.
   Explicit Stop completed with no running container. The archived partial copy
   remained unchanged through re-import, Start and Stop.

Offline extraction retained 69 receipt files (110,484 bytes). The read-only
database check reports schema22 and `quick_check=ok`, one new `zod` registration,
the revision-1 retirement tombstone and exactly these durable operations:
interrupted import, successful restoration, successful new import, Start and Stop.
`copy-interruption.json` describes the intermediate preservation-only phase and
the old action's refusal; `import-case.json` is the full successful result.

## Isolation and evidence retention

Appliance09 and run10 use the existing no-NIC/no-host-share QEMU boundary, private
rootless engine and fresh source/root disks. No production installation, desktop
plugin, existing ship, tmux session or `~/temp-ur` was accessed. The guest has
stopped; no live VM job remains. Receipts are at
`/tmp/omabit-import-inspection-20261022-10`; original disks and backing image are
retained. The redundant converted inspection image was removed only after
verifying its hash, all original disk hashes and all extracted receipts.

To admit this run under the temporary-storage quota, the original appliance08
artifacts and completed run09 disks were moved byte-for-byte to
`/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-22-import09`.
Their original directory paths remain as links, and every recorded hash matches.
Failed appliance01's package staging was archived, verified entry-by-entry and
retained in the same persistent folder before removing redundant staging. Its
original failed disks/logs and appliance02's staging remain. No user data was
cleaned up. Relocation and archive receipts are retained with the evidence.

## Next checkpoint

Wire the reviewed restoration into the guided UI, initially using private
offscreen QML with inert dispatch. Require a fresh operator/capability/revision
binding, present both moves and retained archive, require explicit approval, and
retain the exact request after an uncertain response. Completion removes the
failed instance, so its success receipt must remain visible outside that instance
delegate. Test stale and replaced selections, duplicate clicks, uncertain results,
terminal retirement, continuation and old-request replay before UI-to-VM testing.

This run interrupts a copy through a controlled runner service stop/restart.
Full disks, abrupt process/power loss, device renumbering, other import layouts
and the UI-to-runtime gate remain open. Reboot testing also needs a dedicated
resume path: the current fresh-guest initializer must not normalize ownership
again on a populated test disk. The overall platform goal remains active.
