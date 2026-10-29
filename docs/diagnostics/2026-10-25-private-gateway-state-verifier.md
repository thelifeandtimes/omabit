# Initialized gateway workspace verification — 2026-10-25

M394 adds a metadata-only verifier for first-start gateway observation. The
existing WorkspaceVerifier remains limited to an empty workspace before effects.
The new verifier requires the retained start-dispatched or started gateway,
opens the recorded private-projects parent and preserves the recorded root,
piers root and member directory identities. Its first successful observation
binds the chain and secrets directory identities for the handle's lifetime.

Two bounded scans must agree on names, identities, modes, owner, link counts,
sizes, birth/modification/change times and current pathname bindings. Child
opens use openat2 beneath held directory descriptors with no symlinks or mount
crossing. Regular files are opened with O_PATH for metadata only. The verifier
never reads their contents and provides no sync, initialization or repair method.
Read-only directory scans can update access times; those are not treated as
content changes. Close and repeated verification serialize on the held lock.

The expected first-start layout requires home, chain, secrets, piers, marker,
namespace metadata and the ready document. Member directories must still be
empty. The four generated key-related files must be present, private, regular,
single-link, nonempty and bounded; their values are never read. The chain must
contain nonempty regular data. Directory depth, total entries and file bytes
are bounded, and metadata output contains directory identities only. A ready
document's presence is a layout check, never evidence of its claims.

All 27 focused race checks and vet pass with synthetic temporary directories.
They cover preserved contents, replaced parent/root/piers/member/chain/secrets
objects, links, a FIFO without blocking, public modes, missing/partial layouts,
oversized sparse files, cancellation and repeated concurrent verification.
All 422 privatechain package race checks pass after integration, and
git diff --check passes. No gateway, Docker engine or actual ship was accessed.

This result establishes current metadata and held identities, not database
validity, durability or live key agreement. At initial opening there is no prior
chain/secrets identity in the workspace journal; those must be attributed by
the complete live gateway observation. Reopening cannot independently prove
their historical identity. The complete observer must combine this verifier
with the selected engine/container/namespaces, sealed initializer and fresh RPC
probe. Same-UID mutation remains outside this boundary; private permissions do
not establish secure wallet storage. Managed wiring, new sealed inputs and a
fresh no-NIC VM qualification remain outstanding. This supersedes M393's missing
initialized-workspace verifier only; all protected and installed state is unchanged.
