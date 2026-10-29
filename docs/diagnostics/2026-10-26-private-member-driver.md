# Initial private member runtime driver

M430, 2026-10-26. The private first-member driver composes the existing durable
launch controller with concrete artifact publication, key proof, full gateway
observation, exact container Create/Start, held first-boot capture and process
observation. Construction is read-only and requires a fresh accepted launch and
separately staged pinned inputs. The caller must hold the common work/update fence.
Peers are refused until their durable readiness is available.

Artifact publication uses exclusive writes beneath a held managed-root directory.
It hashes the copied bytes, syncs the new file/directories, preserves partial
objects and returns a known directory identity even after a later failure. No
cache download, image pull, overwrite or cleanup is performed. Foreign artifact
mounts are excluded before Create, including ancestors and file inode aliases.

Key/chain proof runs before initial acceptance, Create and Start. Full current
gateway observation follows each potentially slow proof; held key, pill and empty
mount checks then run again. Only an acknowledged Create can advance to Start.
The empty mount is armed after durable Start dispatch, directly before the one
Start call. A lost Start acknowledgement cannot become capture authority. A
bounded read-only wait checks the process and mount while waiting for the initial
pier layout, then captures once. Full member-aware gateway checks and a final
process snapshot must pass before returning running evidence. Created containers
are inspected without exec. Running namespace reads are bracketed by complete
container snapshots. There is no HTTP readiness or clean shutdown claim.

All 90 new checks and 1,104 affected gateway/member race checks pass with no skips;
vet and diff checks pass. Tests use synthetic files and mocked engine calls.
The first failed fixture had inconsistent host/container boot IDs; it was corrected.
A later alias test exposed that directory-only mount exclusion did not detect a
hard-link alias of the pill. The implementation now compares both directory and
pill objects. Both failed results are retained alongside the passing result.

This supersedes the pending concrete first-member effect/observation adapter in
M427/M429 only. It preserves schema44 and adds no public route, scheduler factory
or startup dispatch. Captured pier evidence is still held in memory, so a durable
binding is required before readiness, second-member admission or resume can rely
on it. The actual managed-member VM boot, acknowledged Lens Stop, restart recovery
and public integration remain. Existing fixture messaging tests do not qualify
this driver. No real key, ship, host engine or installed plugin was accessed.
Freeze original44 before changing the persistent capture/readiness format.
