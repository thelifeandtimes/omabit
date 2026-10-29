# Explicit persisted recovery observation

Unreleased schema39 adds an actor/request-bound observation receipt for one exact
interrupted recovery checkpoint. Acceptance pins the project revision, source
history, parent operation, journal sequence and checkpoint digest. The private
worker shares lifecycle admission and the management-update fence. It exposes
only read operations; replay and startup reconciliation never schedule work.

The parent checkpoint and reserved member mirrors advance atomically with the
observation receipt. The parent remains interrupted until a separate explicit
continuation; only a fresh complete readiness observation can finish recovery.
Unknown Create outcomes, overlapping requests, stale authority, altered mirrors,
missing ownership and changed history are refused. Failed/cancelled reads retain
the parent unchanged. Historical observation receipts must remain exact ancestors
of the later journal, including whether an acknowledgement was actually received.

Store13.949s, queue8.367s, combined race334.200s, ancestry race149.324s, migration9.239s
and static checks pass. The first store run failed because its injected time.Now
kept monotonic clock metadata that JSON intentionally omits. Using a UTC clock in
the fixture fixed the persisted-versus-in-memory comparison; no evidence validator
was relaxed. The failed log remains retained. Original38 reproduces byte-for-byte
and rejects actual populated39 unchanged. Earlier original fixtures remain intact.

This supersedes M167/M168's pending persisted observation and successor-refusal
status only. The full schema39 managed regression remains pending. Mutating
continuation, a public route and runtime observation qualification remain open.
Frozen VM31 uses schema38 and excludes this work; its runtime result is pending.
