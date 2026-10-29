# Independent retained-gateway auditor

The stopped-disk reader now has an explicit, sealed resume-audit mode selected by
the prepared build. A required resume report cannot silently fall back to success
from the original peer case. Duplicate/unknown markers refuse. The host driver
also requires the matching resume success projection before declaring a pass.

The reader compares the original proof file hash, original gateway identity,
replacement identity, operation, exact chain device/inode, ready and final receipt
bytes, unchanged head/public keys, minimum contract/key observations, mutation
refusals, guest completion and clean shutdown. It reads only fixed public receipt
paths and exports an explicit metadata projection. Failed proofs yield opaque
diagnostic hashes; no raw key, application state or log content leaves the disk.
Receipt reads reject symlinks, hardlinks, special files and oversized inputs.

Twelve Go tests/69 including subcases,50 Python tests and vet pass. The uninstalled
reader refuses host execution with exit80 and empty output. Initial Python failures
exposed an unrelated random temporary-directory underscore; deterministic UUID
fixture names now satisfy the real admission grammar. Initial vet could not write
the resolved cache outside sandbox roots; the scoped offline check passed.

VM10 remains in progress. No resume disk audit has yet run, and actual retained
Ganache reopening remains unqualified. M350/M352's original reader evidence is
preserved; this source extension does not retroactively change those results.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-25-private-gateway-resume-auditor`.
