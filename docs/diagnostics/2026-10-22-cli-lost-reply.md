# CLI lost-reply walkthrough

The private built-CLI simulator passes six tasks / 57 commands, including a lost
reply after the runner accepted a Stop operation. The client exits with uncertainty;
reading the snapshot finds exactly one matching request/operation. Re-presenting
the original terminal request returns that same operation ID. No new mutation ID,
automatic retry, inverse operation or duplicate work is used.

The original selection, stale-revision, asynchronous receipt, unavailable-state
and replacement-identity tasks also pass. [Frozen evidence](2026-10-22-cli-lost-reply.json)
records the binaries, script, transcript and hashes. This is an in-memory simulator
in a private Bubblewrap namespace; it does not qualify runtime restart durability,
real imports or production services. The test proxy drops only the selected first
reply and otherwise preserves the runner's protocol headers and connection framing.

The prior transport01 trial failed before the mutation because the test proxy
omitted a required protocol header. It is retained as a harness failure. This
pass supersedes the earlier transport-loss-pending status, not that failed record.
Structured CLI recovery guidance is subsequent source work, not part of this frozen
binary's evidence.
