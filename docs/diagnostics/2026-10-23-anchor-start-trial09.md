# Anchor Start trial09 and exact-image probe correction — 2026-10-23

The private no-ship VM created both resources, committed Start intent and a
successful acknowledgement, then refused post-Start observation. The Docker guest
journal records `/usr/bin/stat: no such file or directory`. The operation remains
interrupted at `started` with no recorded namespace. Cleanup stopped only the fresh
anchor (exit143, no OOM); clean VM poweroff took229.64 seconds. Nineteen receipts,
the exact SQLite mirror and original disks are independently verified and retained.

Offline replay confirms that the running anchor's fixed configuration, ephemeral
mount and loopback publications match current pure validators. The independent
verifier initially rejected raw HostConfig equality: this engine changed
OomKillDisable false→null and OomScoreAdj0→200 on Start. The corrected verifier
checks those two exact deltas and equality for every other field. That verifier
correction does not change the runtime failure. The first retention attempt refused
its missing verification prerequisite before copying anything; the verified retry
completed. Original runtime receipts were never modified.

Digest-verified OCI layers show `/bin/stat` is a BusyBox link; `/usr/bin/stat` is
absent. The exact BusyBox/musl files successfully ran `stat -L -c %d:%i` against the
private namespace in a no-network Bubblewrap probe. The adapter now uses that path
and syntax. Focused race tests and vet pass; guarded VM entries still skip on host.
The fixture now retains bounded inspection errors to make future refusals precise.
A fresh VM is required before claiming real Start completion.
