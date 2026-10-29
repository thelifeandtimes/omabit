# Runner login backend trial02 — 2026-10-23

The corrected fixture waited through one unavailable resource observation, then
selected the exact fresh container with unchanged runner/instance/revision/ship/
endpoint. Code retrieval reached the backend: the offline database contains one
failed `login-code-audit`. No code or session was obtained. The existing generic
audit cannot distinguish endpoint/identity/command failure from timeout.

Import, replay and explicit healthy Start passed. The case failed before its
explicit Stop/restart sequence; only clean VM poweroff is established for final
shutdown. The VM ended in761.96seconds without OOM, quota, deadline or log failure.
All44 extracted files, original artifacts/disks and66 retained metadata files were
independently verified and preserved at
`/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-23-login02`.
The result JSON matches the earlier trial's generic failure text; its identical
hash is not evidence that the selection/audit outcomes were the same. Their
separate receipts prove the difference.

The next source adds a finite failure-stage vocabulary and an outer-deadline flag
to the existing nonsecret operation audit. It never wraps or stores command errors,
response bodies, code or cookies. API failure bodies remain generic. Tests cover
invalid selections/inventories, before/after identity changes, command errors,
unknown/typed-nil diagnostics and redaction; race checks and vet pass.

The four-second outer timeout, three-second command bound, two-second curl limit
and all selection/identity checks remain unchanged. The observed duration makes
timeout plausible but does not prove it. A fresh diagnostic trial is needed before
choosing a runtime fix. This advances M035's selection issue while keeping M031's
real code-retrieval gate open; it does not supersede the historical failed evidence.
