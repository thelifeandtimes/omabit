# First-use private workspace parent provisioning

The internal workspace queue now provisions its shared `private-projects`
grouping directory after actor replay and fresh project/revision/review checks,
only when no workspace claims exist. The host work gate and maintenance checks
cover this metadata step. Retained workspace claims always require inspection of
the same parent identity; a missing, replaced or unsafe parent refuses with
`private_storage_recovery_required`, without recreation or permission repair.

The filesystem helper holds the private state-root identity and uses exclusive
mkdirat plus no-symlink/no-mount child opening. It accepts only an empty, owned,
0700 same-device directory, syncs child and parent, then rechecks identities and
contents. Existing unknown data is retained. It creates no project root, chain,
key or pier. Those effects still require their durable workspace receipt.

Failure before a receipt can leave an empty grouping directory. A later reviewed
request can verify and sync that same empty object; it cannot adopt project data.
Injected journal/mirror failures leave no workspace or operation row and launch
no worker. Their valid retries retain the parent identity. Exact receipt replay
uses no filesystem/backend, including after the original parent has moved.

All27 focused filesystem and50 focused queue race checks plus vet pass. After
integration,452 privatechain and95 workspace race checks pass, including stored
replay, rollback, corruption, observation and original-schema fixtures. The first
fixture attempt correctly refused the testing framework's0755 temporary root;
the corrected fixture explicitly creates its private0700 root. Failed evidence
is retained. Diff checking passes. Schema43, public routes and startup dispatch
are unchanged. No Docker, actual ship, key, database payload or installed state
was accessed; these are synthetic filesystem/store tests.

This supersedes M399's missing shared-parent provisioning only. Supported input
delivery and fresh current-source no-NIC VM qualification remain, along with the
later public lifecycle surfaces. Preparation23 remains historical and unbooted;
it is not evidence for the new managed adapter. Same-UID mutation protection and
wallet-safe key storage are not claimed.
