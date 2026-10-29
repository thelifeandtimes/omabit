# Verified runner login across pier restart — 2026-10-23

Fresh private VM run23-06 passed the actual collector/runner code-retrieval path
both before and after cleanly stopping and resuming the same imported fake pier in
a different container. Both exact fresh selections authenticated a direct scry and
revoked their temporary session. Two successful access audits match the same
instance and configuration revision. Codes and cookies remained in memory.

Independent inspection confirmed the source tree was preserved before Start, the
exclusive parent mount and correct ship identity, distinct container IDs, successful
operations and final stopped state with no remaining runtime container. The VM
powered off cleanly in 806.97 seconds; no outer deadline, OOM or storage
failure occurred. All 51 extracted files, original
artifacts and disks were verified, with 73 metadata files retained at:
`/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-23-login06`.
Result receipt SHA256: `9e515d41654a9a8e74c0febb0593402d9a2e8067cf33a3fd4d098d6894a23135`.

This supersedes M042's runtime-fix-pending status and closes M031's basic real-runner
code-retrieval/restart gate. The earlier failed trials remain retained. The bounded
12-second read and 15-second response/client fix preserves all endpoint and identity
checks. A separate same-client race test confirms ordinary requests still use their
original timeout, including cancellation without replay.

This run uses the ordinary administrative endpoint. Relocated-port behavior has
source tests; real shared-project discovery and lifecycle integration remain open.
No installed plugin, production service or existing user ship was accessed.
