# Admit disk conversion by virtual size

VM21's12GiB virtual disk exceeded the extractor's fixed4GiB temporary-space
estimate. Extraction and a concurrent race suite exhausted the50GiB user quota;
SQLite tests reported disk I/O/disk-quota failures, and the first offline verifier
exited120 without a writable error log. The failed test output and partial verifier
dump are retained. Original runtime disks/results were unchanged. After verified
relocation of the conversion, the unchanged independent verifier passed.

The extractor now reads the completed qcow2 geometry in its read-only namespace
before creating output, reserves every virtual byte plus2GiB for extraction and
verification, and rejects malformed geometry. Three focused tests pass. The actual
completed overlay's12GiB geometry was read independently; its14GiB requirement
correctly refuses current temporary headroom before output creation. A new full
conversion under this check remains separate from these source/read-only tests.

Do not overlap large conversions/verifications with the broad Go regression suite.
Admission is a point-in-time quota check, not a reservation against other processes.
When necessary, copy and hash exact inactive lab artifacts into durable evidence
before relocating temporary paths; preserve original disks and failed attempts.
Never clear unrelated temporary directories or omit evidence to reclaim space.
