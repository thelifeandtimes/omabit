# Bind retained startup to the predecessor's final head hash

Review of the pending resume fixture found a specific evidence gap: it checked
the original head height and public keys, then compared hashes before and after
the reopened service. It did not have a final head hash recorded by the original
gateway. The legacy top-level `blockHash` was captured at0xf before gateway/DNS
setup and mining to0x50; it is not a final-head proof.

The next initializer now records `gateway.headHash` through a read-only lookup
after service shutdown. The next resume contract requires this exact hash at
the selected height before readiness, as well as the existing genesis/contracts/
keys and before/after checks. Missing, malformed or changed hashes refuse. The
independent reader reports `original_head_hash_matched` explicitly, requires it
for the next peer-recovery variant, and preserves the older VM10 case as limited
evidence rather than silently overstating it.

Thirty-nine JS checks and110 Go checks including subcases plus vet pass. VM10's
sealed initializer/resume inputs remain unchanged, and its still-pending audit
will explicitly report the older coverage. The fresh peer-recovery preparation
has not run since the quota refusal; its planned inventory comparison now includes
these two source changes. Actual exact-predecessor-head recovery remains pending.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-25-private-gateway-original-head`.
