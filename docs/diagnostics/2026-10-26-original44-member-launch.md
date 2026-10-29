# Original schema44 first-member launch baseline

M431, 2026-10-26. The current schema44 development writer was frozen before
adding durable pier-birth/readiness records. Its source archive contains 2,745
files, including the first-member store and M430 concrete driver, and excludes
nested source archives and runtime data. This is an unreleased development source
snapshot, not a stable release or managed-runtime qualification.

Two isolated exports from that exact writer produce identical synthetic fixtures:
1,440,462 bytes, SHA256 `ba8b79265136a789efc7bf730f71f215a3fd23f9eadd401c63001e3579ccebda`.
The source archive SHA256 is `80e34af6c6569e9c2bd37a0589cb674261524f6dc1eec176f480afca2aece1a6`.
Each fixture contains both original private projects, original/recovered workspace
and gateway histories, one completed first-member launch and one interrupted at
Start dispatch, alongside existing fleet/project histories and sidecars.

All 23 selected preservation/support/migration race checks pass, along with
vet and diff checks, without skips. The new preservation test opens the original44
fixture three times, checks every original table cell and sidecar, preserves runner
identity and separate gateway attribution, verifies actor-scoped member replay,
and refuses restarting completed or uncertain launch intent. No launch driver,
engine, ship, credential, production database or installed plugin was accessed.

M431 supersedes the pending original44 freeze in M429/M430. It does not qualify a
successor migration or original44 refusal of successor state; both must be tested
when that real format exists. The historical generator extension is retained in
the evidence/staging directory and will join the supported prior-writer manifest
when a successor schema is introduced. The archive's inherited purpose sentence
mentions pre-member lifecycle work; this record corrects that wording: schema44
already persists initial member launches and precedes durable birth/readiness.
