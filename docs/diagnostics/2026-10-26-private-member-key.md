# Held private member bootstrap-key verification

M414, 2026-10-26. The new read-only verifier selects a member key path from the
canonical private project and validates its recorded root/secrets identities.
It requires an independently accepted whole-file SHA256, holds the original
directory/file handles, and compares exact bytes plus ownership, permissions,
link count, birth identity and modification metadata before and after reads.
Keys remain private0600 files and directories remain0700. The initializer's
trailing LF is included in the hash; whitespace is never repaired or normalized
on disk. Partial, replaced, linked, special or changed files refuse.

The API returns only success or a bounded generic error. It does not expose key
bytes, hashes or paths in diagnostics; does not inspect the other member's key;
and supplies no staging, rotation, repair or runtime operation. Tests use invented
format-valid bytes, not real network keys. All34 targeted race-enabled checks,
package vet and diff checks pass.

An accepted hash is a prerequisite, not independent proof that a key belongs to
the selected live chain. No persistent member intent, hash-acquisition flow or
cryptographic public-key correspondence is implemented by this helper. Those
steps, fresh gateway observation and explicit boot dispatch remain driver gates.
A caller-selected hash or a saved gateway receipt alone must not authorize boot.
Resume must not open this helper or regain the initial key mount.

This extends M412/M413 without superseding runtime evidence. Schema43 and public
interfaces remain unchanged; VM07 uses its frozen earlier binaries. Private file
permissions and held hashes prevent accidental substitution under the trusted-UID
model; they do not protect wallet seeds from same-user code or clear every copy
of a secret from process memory. No wallet custody or signing was implemented.
