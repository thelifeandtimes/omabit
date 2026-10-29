# Member capture succeeds; gateway observation expires

M442, 2026-10-26. The M440 logging-only fresh VM reproduces the interrupted
member launch. Independent audit16 observes the stopped disk read-only and
preserves both disk and base hashes. It extracts 1,290 fixed return records with
zero malformed records; only the final128 are retained in the public projection.
Original source hashes are checked before interpreting their line locations.

`member_birth_linux.go:166` returns successful capture at elapsed658,318ms.
Opening the member-aware gateway observer succeeds. The first member/process
observation, chain RPC, and both held filesystem verifications then succeed.
At elapsed712,294ms, `private_member_gateway_observer_linux.go:97` returns from
the second member namespace observation with its context expired. The refusal
propagates through `private_gateway_observer_linux.go:244` and `:182`.
The outer launch context remains live at `private_member_launch_linux.go:408`
and `:388`. Thus the observed launch refusal is the gateway observer's25-second
mutable-observation deadline, not the12-minute fixture deadline, failed capture,
pill corruption, or a storage failure. This identifies the failure in this
diagnostic run; it does not independently establish every earlier run's cause.

The trace also demonstrates M441's persistence gap: successful birth capture
is lost from durable state after the later observation fails. The durable launch
remains `started`, operation `interrupted`. Member exit1/no OOM and `%term`
remain teardown evidence; no complete member boot, readiness or Stop is claimed.
The diagnostic overlay itself cannot qualify the ordinary runtime.

Next, reduce redundant read-only engine inventories while preserving complete
before/after comparisons, object binding, fresh namespace observations and the
existing deadlines. Measure and qualify that change in a fresh VM. Keep M441's
capture-persistence work separate and use original45 for its future migration.
No timeout, freshness requirement, schema, installed plugin or runtime selection
has changed in this checkpoint. This resolves M440's pending diagnostic outcome
and narrows M439's unresolved first-failure question.

All runtime, supervisor and audit units are terminal. The3,635,216,384-byte disk
is synced, hash-verified and retained privately with a compatibility symlink.
SHA256: `f971d2028fa19e1a1168074c6631edf4180e50ea8def6374c0f4dd44b794dff1`.
Evidence: `omabit-qualification-evidence/2026-10-26-member-observation-deadline`.
Keys, raw logs, chain contents and pier payload remain private. Production ships,
tmux ships, temp-ur, installed plugins and host Docker remain excluded.
