# Omabit 0.3.6 — local import preview correction

Local import and source-restoration previews now use the runner's existing 8 TiB
logical-size limit. Previously the collector applied the 100 GB cross-host limit,
so Inspect pier could reject an otherwise valid local source with the misleading
message “guided import identity, bounds or destination changed.” Size, identity
and destination failures now have separate diagnostics.

This is a client validation correction. It does not raise the runner's local
limit, change schemas, alter piers, start a ship or weaken identity, stopped-state,
ownership, destination, entry-count or full-content verification. Sparse file
lengths count toward the limit. Cross-host handoff remains capped at 100 GB.
Runner schema 47, desktop state 1, bootstrap API 1, runtime image and pill remain
unchanged.

## Install from 0.3.5

```sh
omarchy plugin update thelifeandtimes.omabit
```

Allow the verified 0.3.6 bundle to load. In Omabit, use **Update Dependencies**
if it is offered, authorize the local upgrade and wait for completion. The
collector in the new bundle contains the preview fix; the qualified release also
supplies matching runner and Omabit binaries. A local update does not upgrade
remote runners automatically.

Verify that Setup reports **loaded: 0.3.6**. If the shell still holds an earlier
service object after the update has completed, reload it:

```sh
omarchy restart shell
```

Then inspect your stopped pier again and review the newly generated import plan.
Import remains an explicit manual action. Do not reuse a prior preview or import
while the same ship is running elsewhere. No workstation installation or user
ship import is performed by the release process.

Earlier installations use the documented intermediate paths through
[0.3.5](release-0.3.5.md). The exact qualified path and its limits are recorded in
[qualification](releases/0.3.6/qualification.json).

## Qualification scope

Synthetic client and runner tests cover 100 GB + 1 byte, 512 GiB, the exact 8 TiB
limit, rejection above it and preservation of identity/destination validation.
Restoration tests cover source and destination independently.

The isolated upgrade rehearsal uses published 0.3.5 and frozen 0.3.6 binaries,
fresh disposable fake ships and an offline VM. A sparse fixture demonstrates the
old rejection and corrected preview with both the old and upgraded runner. The
fixture is removed before a separate small stopped-fake import, which also checks
macOS metadata preservation. This is not a full 100 GB copy benchmark or a claim
about the time/free space needed for a particular production pier. No production
pier or human DEV state is used.
