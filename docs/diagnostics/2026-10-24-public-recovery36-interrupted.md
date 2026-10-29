# Public recovery VM36 interruption

The outer execution ended143 before the driver could write result.json. Its exact
scope is inactive; its journal records9min29.702s wall time and7GiB peak. The cause
is unclassified. There is no final guest/project report or confirmed clean guest
shutdown. This is an incomplete trial, not a runtime pass and not evidence that
the recovery read fix failed.

The original QCOW2 and backing artifacts are retained. Read-only extraction
preserved the project database together with its WAL and SHM. A separate copy
was opened for WAL recovery and backed up; both quick-checks pass at schema40,
and hashes of all three original files remain unchanged. It contains one project,
two members, zero ordinary instances and six operations: registration, preparation,
anchor Start and both empty member parents succeeded; the member-launch receipt
is interrupted. No full application or public recovery completion is claimed.
The unclean filesystem may lack data that had not reached disk.

All original disk bytes, logs, source identities, extractor/auditor, SQLite/WAL
and recovered diagnostic snapshot are retained. Only the independently hashed,
regenerable raw conversion was removed. No original disk journal was replayed.
The success-only VM36 verifier was not run. This supersedes M199's running status;
M201's passing selected regression and M192's private runtime reference stand.
Next qualify M200's separate public initial lifecycle, then repeat full public
recovery on a fresh overlay. Protected ships, installed plugins and host services
remain outside the experiments.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-public-recovery36-interrupted`.
