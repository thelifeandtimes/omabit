# Exact stopped-container retirement — 2026-10-23

Resume must use a replacement command without bootstrap flags. Its first separate
step is an explicit retirement intent for one exact cleanly exited container.
The model retains the original Stop and exit observation unchanged, rechecks the
full current project/process and initialized pier, and commits dispatch before
removal. Acknowledgement and complete absence are separate phases. Cancellation,
lost replies and failed commits preserve their last durable boundary; there is no
automatic replay, adoption, removal retry or replacement launch.

The concrete adapter verifies original contracts, exact exit metadata, unchanged
peers/anchor and initialized pier/parent identities before and after inspection.
It rejects Vere lock files and unsupported state files, and rechecks fresh complete
ownership immediately before `container rm` with one full ID. It passes neither
force nor volume/link options. Docker documents those options separately in its
[container removal reference](https://docs.docker.com/reference/cli/docker/container/rm/).
After acknowledgement, a complete bracketed inventory must omit exactly that
member and retain every peer/project field and full inspected process metadata.
The pier, state bytes and historical evidence remain retained. Peer/namespace
changes require separate recovery; clean exit observation never schedules removal.

Thirty-four pure-model cases and nineteen injected-engine/temporary-file cases
pass, together with affected race suites and vet. They cover each commit boundary,
lost/bad replies, cancellation, stale evidence, restarted processes, changed peers,
unknown attachments, changed piers, lock/hard-link refusal and duplicate calls.
The Docker wrapper discards cancellation-time output: that case stays dispatched
and uncertain, even when the fixture engine already removed its synthetic target.
Only an acknowledgement actually returned to the controller can reach removed.
Each fixture verifies the retained state file's inode and bytes.

No schema change, persistence, queue or public route is added. No real container
was removed by this code. VM17 is frozen before this model/adapter; its separate
read-only Stop recovery trial cannot qualify retirement or resume.
