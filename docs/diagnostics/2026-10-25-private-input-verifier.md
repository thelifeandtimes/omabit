# Held runtime-input verification — 2026-10-25

M388 adds a read-only RuntimeInputVerifier holding the reviewed canonical input
root. Opening checks identifiers and directory ownership/mode; Verify must still
succeed before contents qualify. Each call hashes the complete manifest and tree,
then checks that the original pathname still identifies the held directory.
Missing, replaced, writable or symlinked roots refuse, including an identical-byte
replacement and a symlink introduced into an ancestor. Close is idempotent and
serializes with verification. Neither API executes, repairs or grants startup
permission. Same-UID mutation remains outside the isolation boundary.

The offline InspectRuntimeInputs command now uses the same held-root checks.
Manifest format, pins and schema43 are unchanged. The opening API intentionally
does not turn a directory handle into content or execution authority.

Eleven top-level /65 focused race checks and vet pass. A first run failed during
fixture setup: moving the sealed directory between parents required temporary
write permission to update its parent entry. The corrected fixture restores0500
before opening; the failed result remains retained.

A separate temporary qualification test read generation06 twice through one
handle: each scan verified12,352 files /834,036,904 bytes and returned exactly the
reviewed manifest/foundation with runtime_authorized=false. Close and refusal
checks passed; the manifest remained unchanged. No bundled program ran and no
chain database was opened through a database library.

After byte-identical integration, all51 top-level /364 privatechain race checks
and the three affected offline collector tests pass. Counts overlap and must not
be summed. This is not a full managed-store or live-runtime rerun.

This supersedes the snapshot-only input binding in M386 and the pending held
input handle in M387. The concrete engine/container adapter, journalled guest
initialization, public lifecycle and supported runtime delivery remain open.
Protected ships, tmux, temp-ur and installed plugins/services remain excluded.
