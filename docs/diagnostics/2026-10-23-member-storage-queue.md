# Queued member storage and private VM gate — 2026-10-23

The private server now counts accepted member-storage workers before management
update admission and acquires the common resource gate before opening the filesystem
driver. Cancellation, setup failure and retained maintenance preserve the accepted
claim and original planned receipt. Replays do not queue another worker. Focused
project race regressions and vet pass; no public route or startup scheduling exists.

A separate no-NIC VM case runs the real queue/filesystem adapter for both reserved
members and reads SQLite inside each filesystem callback to verify ordering. It
checks exclusive empty parents, retained runtime IDs, repeated-request refusal and
unchanged receipts after reopen. It also reruns initial anchor Start under schema26;
the anchor stops before the empty storage preparation. No ship is launched.

All three VM entries skip on the host. Thirteen offline harness tests pass. Fresh
appliance23-11 independently verifies61 source hashes, exact artifact/case selection
and the embedded test binary; its verified regenerable staging tree is removed.
The bounded guest is now running. This records preparation only, not a runtime pass.
Member container lifecycle, complete live inventories/readiness and recovery remain.
