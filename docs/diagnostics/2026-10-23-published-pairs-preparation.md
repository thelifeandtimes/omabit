# Published duplicate-project gate — prepared 2026-10-23

The harness now combines `--network-published --network-isolation` with the
existing state/restart sequence. Two fresh zod/nec pairs occupy distinct ordinary
bridges and anchor namespaces, with guest-loopback ports18281/2 and18291/2.
Both pairs must remain live while four same-identity cross-project sessions are
refused and interleaved writes remain independent. Cookies stay in memory; the
foreign test handle is discarded even on failure, preserving original sessions.

All37 offline network tests pass. Four appliance artifacts, case sources and both
Go binaries are independently hash-verified in appliance23-03; regenerable staging
was removed only afterward. Source/verification records are retained under
`/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-23-published-pairs-preparation`.
No VM for this case has run yet. Its admission keeps the existing four-fake limits:
14GiB guest,24GiB scope, no swap, one host CPU and32GiB available-memory minimum.

M037 qualifies only the earlier single published project. Neither that pass nor
these offline checks qualify the combined topology. Actual anchor replacement,
runner resource journals, CLI/UI and hostile/external networking remain separate.
