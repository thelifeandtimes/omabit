# Replacement contracts explicitly reference namespace succession

Complete clean retirement admits a pure per-member creation selection: original
runtime and initialized pier, next member generation, original AnchorStart,
separate successor/retirement operation IDs and the successor namespace. It rejects
partial retirement, changed piers/authority, unknown members and reused operations.
The side-effect-free Docker contract builder always uses resume mode without a
bootstrap pill and retains the original anchor receipt unchanged.

Recovery contracts have distinct digests and namespace/retirement labels. Ordinary
v1 serialization and labels remain unchanged; tests independently reconstruct the
original digest and reject recovery containers under an ordinary contract. Input
aliases and changed authority are refused. The constructor does not inspect a live
engine, prove mount exclusivity or authorize Create/Start; those remain concrete
adapter dispatch gates. Future group admission must also ensure unique operation
IDs across every planned replacement.

Pure focused race3.909s, concrete contract race4.543s, full projectnetwork
race56.377s, related ordinary contract/launch/resume/retirement race50.161s and
vet/diff pass. No persistent schema or current-generation resolver changed.
Replacement execution, fresh recovery readiness, causal history across subsequent
ordinary restarts/second recovery, persistence and coordinated runtime remain open.
