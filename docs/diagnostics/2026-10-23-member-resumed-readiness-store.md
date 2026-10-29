# Durable replacement-readiness observations

Schema33 adds separate read-only observations tied to an exact completed replacement.
Original replacement and lifecycle receipts remain unchanged. Restart interrupts
unfinished observations; a fresh request may observe again while an exact replay
returns its retained attempt. The private queue shares work/update/maintenance gates.

Seventeen persistence cases cover restart boundaries, invalid proofs, worker failure,
concurrent admission and damaged journals. Six queue cases cover shared admission,
replay, cancellation, setup failure, altered evidence and retained maintenance.
Focused/historical and queue race checks pass in80.403/10.046 seconds. The full
managed race suite passes in351.630 seconds; vet is clean.

The frozen original schema32 source contains1334 files; its archive SHA256 is
`3af14bd5559188c9293a23ada1124f246606447935445cd438e6a5fb4305d7eb`.
Two original-writer runs reproduce the288542-byte populated fixture exactly. The
original reader refuses actual schema33 readiness state without modification.
Only schema32 has the320KiB fixture bound; older bounds remain unchanged.

This qualifies source persistence and private scheduling, not real readiness or
release upgrades. The guarded VM fixture was added after full-suite compilation
and checked separately. No production ships, services or installed plugins changed.
