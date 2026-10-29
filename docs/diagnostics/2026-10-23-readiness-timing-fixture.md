# Preserve resumed-readiness diagnostics without repeated fixture audit overhead

After failed VM27, the fixture now retains both runtime observations, the canonical
identity result and per-call start/end/error timing. Whole-history fixture audits
bracket the collection instead of running before every driver call inside its
freshness interval. Each individual read still checks its exact committed journal
and running operation. Production validators, 30-second freshness limits and
worker scheduling are unchanged.

The new source test proves reads are refused before the observation begins and
after interruption, with no Docker/host adapter calls. Race3.004s, entry compile,
explicit host refusal, vet and diff pass. This is preparation for diagnosis and
rerun, not proof that timing caused VM27 or that runtime is fixed. Original27
observations cannot be reconstructed from absent diagnostics. M143 remains failed;
a fresh VM must complete the original direct-client and anchor-review assertions.
