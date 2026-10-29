# Real private adapter preparation — 2026-10-23

Fresh no-NIC VM23-07 passed the real adapter's bridge and stopped-anchor gate in
207.38seconds including guest setup. No ship or anchor process was started. The
actual private engine ID and exact returned resource IDs were bound to the
schema24 journal. Each create/inspection callback read back its required prior
SQLite phase. All six transitions completed; duplicate worker/session setup was
refused, and closing/reopening the separate qualification database preserved the
original receipt and journal. No instance row or pier was created.

The anchor remained in Docker's `created` state with zero PID and never-started
timestamps. Its explicit `/urbit` tmpfs has1MiB size/mode0700; persistent-volume
inventory stayed empty. The original17 extracted receipts, source artifacts and
disks were hash-verified. Independent read-only SQLite checks matched the receipt,
project/member reservations and complete operation-runtime mirror. The guest
powered off cleanly without OOM, timeout or quota failure.

The appliance predates M051's additional source checks. Replaying its exact
inspection receipt through the current pure validators passes with identical
creation-contract digests, including the never-started and strict tmpfs checks.
That replay makes no engine call. Error/cancellation ID retention remains covered
by source/inert-engine tests, not this happy-path VM run.

The application receipt SHA256 is
`c9e30614a48b08b3d91f4ad5c7a223c201918dbef2e2fa6e4368c978450d0366`;
its database SHA256 is
`4c6e5fb19a345d06f6f43ec287b57437548a27d6bdb7955edc05e5d7d1877add`.
Original images remain retained with temporary-path compatibility links. This
qualifies creation/inspection through the private store/controller/real adapter.
It does not qualify public RPC, server concurrency on a real engine, anchor Start,
member launch, readiness or recovery. Installed systems remain untouched.
