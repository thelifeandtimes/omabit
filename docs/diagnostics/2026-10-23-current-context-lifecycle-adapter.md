# Current-context lifecycle adapters

Read-only exit recovery, stopped-container retirement and retained-pier replacement
now accept a Stop bound to fresh current readiness. The original member and the
replacement member can both use this source. Exact stopped-peer receipts are
accepted through a shared contract check; changed or unrecorded peer exits refuse
before signalling the target. No Stop is replayed after a lost acknowledgement.

Twelve inert lifecycle cases and related Stop/exit/retirement/resume/readiness race
regressions pass in39.285s; vet passes. The tests verify one exact effect per phase,
retained pier bytes/inode, unchanged peer metadata, source-conflict refusal and
changed-process refusal. The first fixture lacked one target's .urb/chk directory
and reused the anchor ID for a replacement. Those failures are retained; corrections
changed the fixture, not the product guards. Generation expectations now follow
the explicitly selected creator rather than an incorrect fixture-label assumption.

This extends M126 through subsequent adapter phases but does not add store/queue
admission. Persistent schema remains36. VM25 runs its earlier frozen schema36
readiness fixture and cannot qualify this later lifecycle code.

Before these adapter edits, original36 source was frozen (1450 files); its populated
fixture with two completed current-readiness receipts reproduces byte-for-byte in
two independent runs (533672 bytes). Future schema migration and original-reader
refusal have not yet been tested. No installed database or runtime was changed.

Follow-up: the concrete resumed-readiness adapter also passes after the current-context lifecycle, race9.981s. The extended test and log are retained separately; product source is unchanged from this report.
