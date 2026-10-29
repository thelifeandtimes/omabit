# Real queued member-storage preparation — 2026-10-23

Fresh no-NIC VM23-11 passed schema26's real storage queue/driver for both reserved
members. SQLite readback inside mkdir/inspection confirmed committed ordering.
Each runtime ID retained its own empty `<instance>` parent, with its planned ship
leaf still absent. Repeated dispatch was refused and reopen preserved both journals
and receipts. Initial anchor Start also passed under schema26 and cleaned up before
the storage work. No member container, pier or ship was launched.

Independent read-only inspection of the stopped ext4 image verifies both directory
inodes and nanosecond birth times against their journals, owner/group1000, mode0700,
and exactly `.`/`..` entries. The private engine's only retained container directory
belongs to the stopped anchor. Extracted SQLite bodies/mirrors and all17
receipts agree. Source/artifact and original disk hashes were verified and retained.
Clean poweroff took217.13 seconds without deadline, quota, or OOM failure.

Storage completion does not establish a ready member. Next is durable member
container Create/Start with fresh complete mount/namespace observations and identity
readiness, followed by independent Stop/resume and coordinated anchor recovery.
Future recovery must retain the original anchor Start receipt and record its new
namespace separately; it must not overwrite historical Start evidence. No public
route, capability or production installation is enabled by this private pass.
