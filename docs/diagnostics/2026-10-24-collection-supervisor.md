# External process supervision prerequisite

Source review after M278 found the private collection adapter remembers known
helpers and can inspect shell descendants, but a snapshot cannot prove absence
of later, detached or unrecorded descendants after owner death. Plugin-owned
channel cleanup solves a different problem. Keep reader-owner-lost unresolved
until independent process-tree evidence is bound to the exact old activation.

A guarded qualification prototype now runs an external Linux subreaper before
launching its sole workload. It saves an owner nonce with PID/start/boot/namespace,
records stop intent before signals, stops launch admission, reaps adopted children
and reports closed only after waitpid reports no remaining children. Signals use
pidfds plus current parent/start checks; no global process search or group kill is
used. Controller EOF drains rather than silently abandoning owned descendants.

Four private cases pass: double-fork/setsid orphaning; a new child forked during
SIGTERM; controller EOF; and supervisor SIGKILL. TERM-resistant descendants require
KILL and are reaped. On supervisor loss, its saved record remains running with no
closure proof while descendants survive; a newly created empty supervisor's
closed receipt has another owner and cannot resolve the old record. Exact private
fixture cleanup preserves that uncertainty. Both test scopes are inactive.

The first run passed the three drain cases but failed a fixture assertion that
every surviving descendant immediately becomes a direct child of the next
subreaper. A live intermediate parent still owns its child. The corrected test
checks adoption of the orphaned tree root and continued descendant liveness;
recursive cleanup then verifies every owned process exits. Both runs are retained.

This is a process-ownership prerequisite, not a production supervisor, a journal
migration or a completed collection-recovery adapter. It has not yet supervised
the actual QML/plugin set. Next bind the guardian to each exact loaded activation,
qualify shell death with the real services, freeze plugin transport/application
state before cleanup, then persist an explicit independently verified recovery
receipt. Loss of the guardian itself must remain fenced. Native external desktop
services and detached work outside the owned process tree need their own contract.
No installed plugin, service, credential or existing ship was touched.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-collection-supervisor`.
