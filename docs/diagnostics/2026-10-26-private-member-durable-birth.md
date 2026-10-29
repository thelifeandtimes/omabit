# Durable first-boot pier capture

M432, 2026-10-26. Schema45 stores first-boot pier/.urb capture inside the new
version2 member launch's running observation and its operation mirror. A new
launch cannot complete without matching capture identity, process, namespace,
Start timing and post-capture gateway observation. The driver returns a defensive
copy of the held capture. Persisted ownership checks reject overlap with other
known private workspace, gateway, artifact and captured-pier objects.

Original version1 launch records remain readable and byte-for-byte unchanged,
with no invented capture. They cannot execute under the new version2 contract.
Fresh admissions use version2; restart still only interrupts unfinished jobs.
Birth data/validation now live in the portable model file, while actual held
filesystem capture remains Linux-specific. Captured evidence is validated against
its historical timestamps; new final observations still require fresh commit-time
checks. This does not establish HTTP readiness, a clean Stop or resume authority.

All 1,216 affected gateway/member/store/support race checks pass, with vet and
diff checks and no skips. The migration tests preserve every original44 row and
sidecar while adding a separate new project with a captured member, then reopen
three times and verify actor replay. Corrupt original launch data refuses migration
without changing the original database or schema marker. The frozen original44
writer reproduces its exact fixture, and its reader refuses a real populated
schema45 export unchanged. The prior-reader manifest now includes schema44 while
published RC.12/schema20 evidence remains frozen.

An earlier overlay test process terminated with exit143 after its original44
preservation test passed, without an assertion failure or completed wrapper
result. The cause was not determined; its log and termination record are retained.
The new migration tests passed in a separate partition, and the complete integrated
selection subsequently passed. No production database, real engine, key, ship,
installed plugin or profile was used.

M432 supersedes M429/M430's pending durable capture binding and M431's pending
successor migration/old-reader refusal. It keeps one initial member per project.
Scheduler factory/queue integration, application readiness, second-member admission,
actual managed-member VM boot, acknowledged Lens Stop and restart recovery remain.
The existing schema43 VM audit must be explicitly advanced before the next VM;
its prior successful gateway/key results do not qualify this new member lifecycle.
Freeze original45 before the next persistent format change.
