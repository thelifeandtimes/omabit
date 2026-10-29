# Held private member pier and stopped-state metadata

M418, 2026-10-26. The read-only verifier binds the selected member's retained
mount, ship-named pier and .urb identities. It refuses replacements, missing
objects, extra pier children, symlinks and cross-device children. Held handles
prevent object reuse during an operation; reopening never adopts a replacement.

Identity verification works separately from stopped-state metadata checks.
The latter refuse all three established Vere lock names, including dangling
links or special files, and require owned log/checkpoint trees without shared
writes, special modes, linked regular files or special file types. Traversal is
bounded to8192 entries/depth128 and uses metadata-only handles for regular files.
It reads neither database payloads nor peer/key/chain contents. No lock removal,
repair, staging, sync, engine command or runtime dispatch is exposed.

All36 targeted and87 combined pier/mount/key race checks pass, plus package vet
and diff checks. Synthetic cases cover retained-object replacement, lock forms,
unsafe metadata, caller rebinding, cancellation, concurrent close and both exact
traversal limits. An unreadable regular file remains metadata-checkable; a FIFO
in the peer mount is not visited. No actual pier or engine was accessed.

Lock absence and valid metadata do not establish process exit, acknowledged
Lens shutdown, content integrity or durability. The lifecycle driver must bind
those separately and hold the runner mutation fence. Same-UID code is trusted.
This extends M413 and supersedes its pending retained-pier metadata helper only.
Member-aware gateway observation, accepted key/chain proof, durable member
lifecycle and real runtime qualification remain. Schema43 is unchanged.
