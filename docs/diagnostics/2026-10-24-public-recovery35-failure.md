# Public recovery VM35: failed checkpoint read

The real collector recovery trial failed after2310.85s with a clean QEMU exit0
and outer exit1. The checkpoint GET after the third injected reply loss exceeded
the production client's5-second timeout. The saved parent remains interrupted,
phase `replacing`, sequence22. Timeout cause is not yet established; increasing
the timeout is not a diagnosis. No completed public recovery claim is made.

Independent audit of the stopped disk,19 extracted files, schema40 database and
operation mirrors passes. Four new public requests and three exact lookup-only
requests were accepted before the failure. Three lost replies/store reopens
retain two stop signals, two removals, one creation and one start, with no repeated
effect through the last recorded interruption. One observation and two
continuations are retained. Twelve writes and three successful state/subscription
cycles use the same two original Eyre sessions. Both final pier directory
identities and state files were independently hashed from the stopped disk.

Original QCOW2, prepared artifacts, report, logs, database, verifier and metadata
hashes are retained in:
`/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-public-recovery35`.
Only the hash-verified regenerable raw conversion was removed. The public success
verifier was not run and is retained as an unexecuted draft. Explicit final
fixture cleanup, fourth application cycle and final saved-status CLI were not
reached. The public-initial lifecycle draft is parked while this failure is
investigated. No production ship, installed plugin or host engine was touched.

This supersedes M193's running/pending status. M192's separate private recovery
pass remains valid; VM35 remains a failed qualification.
