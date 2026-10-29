# Remote application recovery with the original peer running

VM24/schema35 independently passes two queued replacements of the same ~zod
pier while the original ~nec process remains running. All eight acknowledged
application writes come from ~nec. Both queued-update and existing-subscription
recovery cycles pass; four channels yield12 matched snapshots. The original two
Eyre sessions are reused without later login or replay of uncertain writes.

The peer's exact container, process start, PID, restart count and configuration
match across all four application phases and through the receiver's final queued
Stop. The peer then exits cleanly through exact-ID fixture cleanup. That cleanup
is not a runner Stop authorized by stale readiness after a peer replacement.

Independent stopped-disk verification matches17 extracted files, runtime and
SQLite journals, pier/application state hashes, replay and reopened receipts.
Outer execution and QEMU exit0, clean poweroff and the inactive exact scope are
verified after1757.56s. Original disks and46 metadata files are retained;
only the verified regenerable raw conversion is removed. Cookies remained in the
anchor's private tmpfs and are not retained as evidence.

This supersedes the pending live-peer runtime status in M120/M121 and the remote
messaging limitation of M124. Schema36 current-context readiness is a separate
fixture: VM24 used frozen schema35. Changed-peer lifecycle admission, runner
namespace recovery, public APIs and desktop integration remain open.

No installed plugin, production ship, host Docker service, tmux ship or ~/temp-ur
was inspected or operated.
