# Strict DMs receipts and real plugin recovery

Fresh native control02 passed the complete Theme/DMs transport case after tightening
DM acknowledgement validation. Four new regression tests first reproduced eleven
failed subcases: absent/unrecognized success fields, bool/float request IDs matching
integers, and a late duplicate replacing the first result. The affected 28 tests
now pass. Unknown replies remain uncertain; no write is automatically replayed.

The runtime control independently matches each of four sent DM IDs/texts to its
exact wire poke, channel and positive Gall receipt, receiver live event, and both
histories. The fifth poke accepts the initial invitation. Theme retains its two
exact profiles with positive receipts and matching settings facts. Restart opens
fresh subscription channels using original sessions; DMs logout leaves Theme usable.
Both sessions and Theme's watcher clean up, and all three ship stops are clean.

The [evidence JSON](2026-10-22-plugin-receipts-native.json) records 37 frozen input
hashes and the independent checks. Result SHA256 is
`e588a33ba26f17acac5ed15eca614bcc19be681b373085c509e85d845a35bce2`. Original piers/artifacts remain at
`/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-22-plugin-transport/native02`
with a verified relocation manifest. Native01 is retained separately.

This supersedes the pending newer-artifact runtime gate in the
[first transport report](2026-10-22-plugin-transport-native.md). It retains the same
limits: fresh native fakes, private namespaces and capped resources, in-memory
credential adapter, disabled desktop application/launchers, no production access.
Native desktop/profile activation, hosted compatibility and Docker plugin
transport are separate gates. The four-ship Docker namespace test is independent.
