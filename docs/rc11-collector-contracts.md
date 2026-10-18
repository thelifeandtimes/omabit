# RC.11 collector remediation contracts

## Historical small-loom creation replay (RC11-15)

The collector preserves the original creation JSON and caller-supplied flags for
recorded loom20–29 requests. It first requires Hello feature
`creation-replay-only`, then sends `X-Omabit-Replay-Only: true` on the creation
POST. Without that capability it refuses before reading a boot key, downloading
a pill, uploading an artifact or submitting a creation request. New selections
and configuration remain30–34; default31 and native-control flags are retained.
Newly dialed client connections recheck this capability on the same connection
before handing it to HTTP, including an unpinned explicit Unix-socket selection.
For a historical request with a selected pill, the collector recomputes/verifies
the original content descriptor but skips runner upload. Even a refused fresh
small-loom request cannot populate the runner's artifact cache before lookup.

**Server integration is complete.** Hello advertises the feature, and all three
creation endpoints (`/v1/instances`, `/v1/comets`, `/v1/ships`) enforce the guard.
The header accepts exactly one literal `true`; empty, malformed, combined and
duplicate values are refused. Under the ordinary actor-scoped journal transaction,
guarded requests return only an existing receipt with the exact historical
fingerprint. A missing receipt returns `replay_not_found`; changed input returns
`request_conflict`. Both are409 responses without new registration, port probing
or allocation, artifact checks, key persistence or journal acceptance. Exact replay
returns the original receipt with HTTP200 before consulting current storage or
provisioning state. The header is transport metadata, never part of the historical
fingerprint. Existing unguarded creation APIs retain their signatures and behavior.

**SSH integration is complete.** `internal/access/bridge.go` validates and preserves
`X-Omabit-Replay-Only` through its explicit request-header allowlist. It requires
`creation-replay-only` in the same-connection delegated Hello before forwarding
the guarded request or body to the runner. Older servers and capability-losing
reconnections to the same runner identity are refused with zero forwarded
mutations. Malformed/duplicate guards are refused before contacting the runner.

Qualification includes frozen schema19 bodies and receipts produced by the actual
RC.10 registration code at `f6981e6ce4b9bb1bddc1334ff23a3f1ae9e9794a`, accepting
loom29 for all three ship types. Tests reconstruct that historical SQLite schema,
open it with the current store/schema migration, exercise the real HTTP handler,
and build/run the actual collector CLI through a temporary Unix socket. Exact
receipts survive reopen/retry; fresh IDs and a different authenticated actor are
refused; original instance bodies, fingerprint bytes, receipt bodies and synthetic
key files remain unchanged. Port/storage callbacks assert zero replay admission
work. Bridge tests cover all three endpoints, header preservation, malformed and
multiple values, old servers and same-identity capability loss. See
`internal/managed/testdata/rc10_creation_replay.md` for fixture provenance.
The schema19→20 reopen and actual collector CLI retry tests also passed after
the parent schema20 integration landed.

## Credential recovery (RC11-16)

`collection.Registry.SaveShipSession` persists the allocated reference in
`retired_sessions` before invoking bounded credential Save. It holds the registry
lock until atomic promotion to `session_ref`, excluding cleanup, forget and a
competing publication. A CAS loser is rejected before Save. Process interruption
or an ambiguous Save/publication failure leaves either an active reference or a
durable cleanup reference. HTTP login/logout happens outside the registry lock.
Remote logout, local deletion and cleanup-journal completion have independent
background deadlines. Local deletion failure retains retry intent for logout.
Logout first drains the selected retired/pending references, including missing
files left by interrupted saves, so a full32-entry cleanup journal cannot strand
an active session. It then rereads the enrollment, checks the original identity,
session reference and revision, and CAS-detaches only that selected session.
A competing login during the initial cleanup yields a conflict instead of
retargeting logout; a later login cannot enter the detached session's cleanup
snapshot. Interrupted cleanup retains completed progress and the active credential
for a reopened retry. Isolated command tests cover missing/mixed/persisted full
journals, deletion and journal-completion failures, cancellation and concurrent
login both before and after detachment.

This reuses the version-1 registry representation and its existing cleanup
semantics, including an absent credential. **No collector schema marker change is
needed.** Version-1 registries remain readable without rewriting them on open.
Legacy theme credential migration uses the same Save seam and removes abandoned
local copies without revoking the shared remote session.

## Provisioning and terminal boundaries (RC11-17/18, G03)

Network pills require HTTPS, including every redirect. Explicit custom HTTPS and
local files remain supported. `--pill-sha256` plus `--pill-size` supplies a trusted
expected descriptor for custom content; verification precedes runner upload.
The built-in Recommended Apps selection always uses trusted release metadata and
the versioned RC.11 artifact, including when selected through the old S3 alias.
Download requests have no cookie jar, user credentials, runner authorization or
environment proxy. Runtime text logs sanitize control and Unicode formatting
characters at the collector terminal boundary.

## Theme state (RC11-19)

Every update validates the complete post-callback schema,64-preference limit and
actual indented JSON byte size (256KiB) before replacement. Migration projects
its eventual preferences and recovery maps before pausing the old publisher or
copying/clearing credentials. Sync and migration prune unchanged preferences for
forgotten enrollment IDs; unfinished migration identities remain recoverable.
Theme schema1 remains unchanged. Failed updates retain the previous readable
generation and migration failures retain resumable progress.
