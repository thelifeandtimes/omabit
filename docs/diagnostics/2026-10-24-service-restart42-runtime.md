# Real runner service restart and public recovery pass

VM42 completes with clean shutdown after2591.55 seconds. Independent stopped-disk
verification passes against frozen schema40/source888 inputs. This closes M230's
pending runtime gate and M227's outstanding clean overall rerun; the original
VM41 failure remains retained unchanged.

Three real owner-service restart cycles use the installed runner binary, its
original identity and managed SQLite store. All seven observed PIDs/invocations
are distinct. Saved checkpoints and history match across each cycle (apart from
read timestamps), and exact project container projections remain unchanged.
The service stops cleanly before the fixture reopens its database. Restart does
not replay interrupted effects or start/stop project members.

The actual collector performs six new public recovery requests and six lookup-only
replays. Three lost replies, two explicit observations and three continuations
retain their actor, source/checkpoint bindings and journal mirrors. Exactly two
each of drain, retirement, Create and Start occur; the anchor is not started again.
The final replay uses the saved actor and preserves all36 operation receipts.

The application retains16 unique writes, eight subscriptions and24 subscription
snapshots across four recovery cycles, with the two original sessions and no
later login or replay of uncertain writes. Both final piers and schema40 mirrors
are independently verified. Initial/member public lifecycle was separately
qualified in VM40; this VM exercises public recovery and real service restart.

The first extraction refuses insufficient quota without reducing its14GiB reserve.
After verified stopped test-pier relocation, extraction and audit pass. The original
VM disk, metadata, receipts, verifier and both extraction logs are retained. Only
the verified regenerable raw conversion is removed. Original exec-tool completion
was lost to context truncation; exit0 is taken from the durable outer result and
cross-checked with the exact inactive successful scope, not presented as a newly
observed tool exit.

Scope: fresh disposable fakes inside a no-NIC VM with rootless Docker29.9.1 and
pinned Vere4.6-5a7614f. Final member stop is exact-ID fixture cleanup. Production
ships/plugins were inaccessible. Public project UI, Docker private-chain support,
full installation/upgrade and release qualification remain separate.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-service-restart42`.
