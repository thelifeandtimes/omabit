# RC.8 setup/import feedback and protection policy

The user requested a single initial-setup machine toggle, compact install approval
with expandable details, a source/location-only guided import with progress, and
discovery/storage using ship-named pier leaves. These changes are implemented in
RC.8 source. `AGENTS.md` and [protected ships](../protected-ships.md) record the
user's exact important identities, moon prefixes and data-preserving reset rules.

## Implementation and validation

The guided preview reads bounded public META records from 64-bit little-endian
LMDB v1 / Vere disk-v3, formats the ship with the standard @p transform, and checks
the selected root, epoch compatibility, free instance name and host HTTP port.
Confirmation and the worker recheck identity/network and the tree fingerprint.
No Vere process, mutable LMDB environment, metadata repair or lock removal is used.
The server still owns authorization, root identity, reservation and content checks.

The worker records bounded phase events and total/copied bytes/files. Large-file
updates are throttled and carried by the existing inventory watcher. The QML pane
binds progress to runner, request, instance, revision and source; it does not retry
an uncertain mutation. The previous log-watcher transport behavior is retained.

## Sourcebox findings (read-only)

The fresh RC.7 installation has runner
`runner-6921f2202ba2cc8377194f443a8b6c5a`, a healthy test comet `a-test-comet`, and a
stale `lismyl-lismyl` registration. Its last Stop succeeded, its enrolled path is
missing and it has no runtime containers. The protected actual pier was located
at `/home/thelifeandtimes/temp-ur/lismyl-lismyl`, outside all cleanup targets.
Its identity was independently read as `~lismyl-lismyl`, live planet.

The failed enrollment receipt reported `incomplete, changed or unsupported epoch
layout`. The retained epochs span Vere 3.0/3.1/3.2/3.3/3.4, 4.3, 4.5 and current 4.6.
Older epochs use layout1; the newest uses layout2. Import accepts this intact
history but labels its inventory separately from qualified maintenance data.

The source also contains a normal two-link docked executable pair: `.run` and
`.bin/live/vere-v4.6-linux-x86_64`. Only a closed internal dock-link set is now
accepted; private state and external/shared links remain fenced.

The completed read-only source review measured **130 files / 49,556,773,510 bytes**,
fingerprint `06f6814bf829fa293274b7b838abc804e6db0d0f8a9366059a4402b1ec80f46f`.
That protected pier was not imported, renamed, started, repaired or deleted.
All active native Urbits and the shared Docker engine remain separate from reset.

## Checks

Passed during development: full Go tests/vet; stateful transport, runtime, managed,
pier-store/copy/metadata race checks; official @p reference vectors and malformed
metadata fixtures; a 15-second metadata fuzz run (722,199 executions); mapped
onboarding, full panel/import progress, and persistent log lifecycle checks;
installer and native-bootstrap fixtures. A real stopped qualification comet also
matched its known identity through the read-only metadata reader.

Publication and protected reset receipts are recorded after their final gates.
