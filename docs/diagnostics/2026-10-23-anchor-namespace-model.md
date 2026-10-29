# Explicit successor namespace after clean draining

The pure namespace intent requires complete validated clean exits for every old
member and preserves the entire drain and original history. An already restarted
anchor is observed without another Start. A stopped anchor receives one committed
Start intent; lost acknowledgements require read-only inspection and are never
silently relabeled as acknowledged. Pending or completed work cannot replay Start.

Before/after evidence pins every stopped process/pier, resource contract and
publication, with a fresh namespace on the same host boot. The successor is a
separate identity, not a rewritten AnchorStart or authority to start old members.
No member retirement, replacement or readiness is implied.

Focused cases pass, followed by full projectnetwork race26.833s, vet and diff.
Tests cover both paths, failed commits, uncertainty, no redundant Start, copied
inputs/history and changed evidence. An initial test compile typo referenced a
nonexistent pier field; the corrected tests use its path and the failed log is
retained. The concrete adapter has only been drafted/compiled and is not covered
by this gate. Persistence, runtime and public routing remain pending. VM27 is
unchanged and contains neither the drain nor namespace succession code.
