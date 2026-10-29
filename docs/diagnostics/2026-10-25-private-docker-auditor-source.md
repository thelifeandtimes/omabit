# Reusable private Docker read-only auditor — 2026-10-25

The repository now contains `private-chains/docker_inspect.py` and its VM-only
`inspect-init` reader. Eight inert outer checks verify stopped owned input, refusal
before hashing/execution for an active run, changed artifacts/backing, indirect
and nonprivate paths, exact scope selection and read-only/no-NIC command mounts.
The full Python fixture set passes40 tests. The reader passes25 tests/subcases and
vet; its compiled init exits80 with no output when invoked on the host.

The successful M346 reader is retained unchanged as historical evidence. The new
source version replaces machine-local test data with a synthetic state fixture,
and narrows its output: private error bodies become opaque diagnostics; arbitrary
receipt text is hashed; success rows contain only explicit typed fields; IDs,
channels, public keys and RPC methods are validated before export. Bootstrap and
failure paths no longer copy complete guest reports, Docker metadata or raw logs.
A compile-only unused-variable error during this extraction was corrected and its
failed check is retained alongside the passing run.

The outer command compiles a frozen reader, verifies its pristine modules, waits
for the original exact scope to be inactive, and launches a distinct no-NIC VM.
Both disk bindings are read-only and ext4 mounts with noload. Success requires
message/state/shutdown evidence, unchanged base and overlay hashes, clean VM exit
and an inactive inspection scope. Scope limits remain2GiB/no-swap/one-core/64 tasks,
with240s and2MiB bounds; only the explicitly owned child group can be terminated.

These source tests do not qualify the new reader against a live-produced disk.
Source-entry VM09 remains in progress; audit it only after its clean shutdown.
Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-25-private-docker-auditor-source`.
No production state or installed desktop was inspected or changed.
