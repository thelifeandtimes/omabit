# Stream authenticated package payloads

The mount-corrected Docker run stopped before host installation or any gateway/ship
start. Seven native package roles authenticated; the engine role did not. Its
91,034,480-byte installed dockerd matches the authenticated package manifest.
Two fresh RAM-only runs of the original verifier reproduce the refusal against
the unchanged, read-only stopped disk. A small instrumented verifier passes, but
that does not establish full-runner success. The full-runner trace identifies the
actual failure: the native tar conversion is killed after its twenty-second
context deadline while buffering 100,823,040 bytes of a 125,327,360-byte stream.

The development verifier now hashes tar members directly from native output. It
keeps signature and installed-byte checks, archive/header validation, all original
size/aggregate/deadline limits, partial-failure revocation and receipt format.
Trailing tar output is drained inside aggregate accounting. A failed decoder never
publishes partial hashes or retries using the buffered path. Existing injected
command-only fixtures remain supported.

The full candidate verifies all eight roles on the same read-only disk in27.25s;
the entire VM probe takes54.66s. Receipt writes occur only in guest RAM. Base and
overlay hashes remain unchanged. Four focused tests (12 including subtests), all
60 top-level hostcompat tests (125 including subtests), and go vet pass. The first
focused run caught omitted trailing-output accounting; a second over-specified
budget-flag assertion was corrected to check the established errBudget refusal.
The first full-suite harness lacked a private GnuPG runtime directory, causing
long fixture socket paths to fail; with a new private runtime all tests pass.
Retain those failures. No installed executable or service changed.

The empty pre-install VM disk is retired after retaining source, audits and hashes;
no ship or pier existed. The next fresh appliance will use the candidate runner
and an explicit offline verification phase before owner installation. It stops on
any refusal without retry. Gateway mount admission, two keyed Docker peers and
application/subscription recovery remain pending. The M340 mount fix is unchanged.
No production, tmux or temp-ur ship was inspected or operated on.

Operational follow-up: the fresh appliance was prepared, but its first boot request
refused before QEMU launch because temporary quota was32MB below the unchanged
8GiB admission reserve. Nine inactive diagnostic initramfs archives (139,655,348
bytes of ELF code/modules only) were copied and hash-verified in
`2026-10-25-private-inspector-code-cache`; their original paths remain symlinks.
No machine disk, chain state, credential or pier was relocated. The subsequent
launch admitted8,697,573,376 available bytes and is running appliance04/run05.
The2GiB running reserve remains unchanged; no outcome is implied by admission.
