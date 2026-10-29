# Correct Lens exit allows official-runtime keyed bootstrap shutdown

Fresh trial03 uses the unchanged official Vere4.6-5a7614f binary (SHA256
b5c54439ae5b9f0ff94d75319e67e04f04369776fa5e2180a5ab1a5ebd4c2ff0).
Its new keyed ship still ignores SIGTERM, confirming the known signal defect.
The correct Lens request uses source.dojo `+hood/exit` and sink.app `hood`;
it returns a Gall acknowledgment and the exact child exits0 in4805ms. The same
pier reopens without additional bootstrap requests, restores normal SIGTERM
handling and exits0 in4277ms. Neither phase requires forced cleanup.

Independent stopped-result audit verifies the official binary binding, both
identity/signal/exit records,257 private bootstrap lookups, final two LMDB hashes
and inactive exact scope. Source, original records and hashed pier/chain trees
are retained; fresh secret inputs and raw bootstrap logs are excluded.

This supersedes M223's conclusion that Lens cannot bypass initial shutdown:
its earlier `|exit` payload was invalid, as M226 already identified. M222's
runtime defect is still present. This is a qualified explicit shutdown workaround,
not a binary fix or normal fresh-keyed SIGTERM qualification. It is not yet a
runner lifecycle implementation, Docker devnet or official two-peer qualification.
The next bounded experiment repeats two-peer messaging/restart with this official
runtime and explicit Lens shutdown for initial keyed processes.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-official-lens-recovery`.
