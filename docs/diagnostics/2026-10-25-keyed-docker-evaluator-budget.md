# Keyed Docker evaluator allowance — 2026-10-25

Run07 passes package verification, installation, pinned image/runtime checks, native
addon loading, fresh key generation and point/key verdicts, then fails at the public
256-galaxy evaluator call. No peer result or ship startup occurs. The gateway exits1
without OOM; the VM shuts down cleanly in466.04s. The original generic evaluator
error did not retain exit details, so its exact termination cause is not directly
observed.

A separate no-NIC read-only VM reproduces the same public response-format parse:
the small control passes in6.64s; the417,055-byte batch expression times out at30s;
the identical batch returns the expected two keyed galaxies in71.45s with a120s
allowance. Both disk hashes remain unchanged. The first diagnostic attempt fails
before evaluation because its RAM-only temporary mount target is absent; retain it
as failed setup evidence.

The next gateway extends only this public batch evaluation to120s. Other evaluator
calls retain30s, and gateway/VM/storage/resource limits remain unchanged. Five inert
checks verify default/extended limits, unsupported-limit refusal and diagnostics
that omit expression and stdout/stderr text. Runtime validation of the new gateway
is still pending; this reproduction strongly supports a timeout diagnosis without
claiming a recorded original ETIMEDOUT.

The run generated fresh disposable keys. Its complete disk is preserved in the
owner-only private lab, with SHA256
`63ac47ae8c6c933c53e2f55fec640a0ce0557faf0efa72821d34a91acdb361b8`.
Its old lab path resolves to that verified copy. No keys, chain database, cookies or
piers enter this code/public-metadata evidence export. All four scopes are inactive.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-25-keyed-docker-evaluator-budget`.
This supersedes M343's pending actual addon-loading proof only. Docker keyed peer
messaging/recovery, runner integration and the broader roadmap remain open.
