# Runtime policy: immutable choices per ship

## Current development: selectable owner-installed runtimes (rc.4, schema67)

The owner's manual review expands selection to fresh ordinary fake/comet/key
boots, existing-pier imports, and explicit changes on stopped ordinary ships.
This supersedes the finite-catalogue and no-transition restrictions in the
historical rc.3 section below. Project members and private-chain factories keep
their separate official-runtime contracts. The exact rc.4 artifacts pass the linked [manual-review qualification](runtime-profiles-manual-review.md#evidence);
live DEV remains on rc.3 until the owner applies its separate prepared update.
The rc.3 artifacts remain immutable.

An enrolled ship stores an exact artifact ID. Live/edge/dev identify the latest
**installed** upstream channel build; refreshing a channel does not change any
ship's pin. The owner must first install an artifact on the selected runner.
Upstream channel discovery failed with HTTP403 from this machine during this
change, so no latest upstream build is claimed as verified.

For a local executable, package without executing it:

```sh
python3 -B scripts/runtime-profiles/build-local-image.py \
  --binary /absolute/path/to/vere --version EXACT_BANNER_VERSION \
  --source-revision SOURCE_COMMIT --max-loom 34 \
  --cache /absolute/path/to/base-cache --output /absolute/path/to/new-output
```

Use `--channel live` (or edge/dev) instead of `--binary` to resolve upstream;
`--bits 64` selects the upstream64-bit variant. An optional `--version` supplies
an exact banner if it differs from the channel label. `--sha256` checks an
expected executable digest. The builder accepts Linux x86-64 ELF binaries and
retains the pinned container base. The runtime checks the exact binary digest
and banner before using a pier. Packaging is not runtime qualification.

`--max-loom` declares the supported address-space exponent, at most40. The
ordinary UI offers30..34 and preserves a larger current CLI choice. A larger
limit neither changes the container RAM limit nor proves that a particular build
supports that loom. `--compatible-vere VERSION` can be repeated to declare older
epoch versions that the build can replay. For a successor to the reviewed JSON
fix, use `--requirement vere-json-array-error-1145-v1` only if that correction is
actually retained. These are owner assertions, not independent verification.

Install on the pier-owning runner, without starting or selecting a ship:

```sh
urbit-runner runtime-profile-install \
  --manifest /absolute/path/to/new-output/profile.json \
  --archive /absolute/path/to/new-output/runtime-image.tar.gz
```

For the separate review040 DEV runner, prefix that command with its actual
binary and development root:
`/srv/dev/urbit-040/bin/urbit-runner --development-root /srv/dev/urbit-040`.
This requires the new matching helpers; the currently installed rc.3 does not
support `--manifest`. Preserve the owner-private runtime-profiles directory with
runner backups, including profile.json and image.tar.gz for every enrolled ID.
Never substitute a different manifest or executable under an existing ID.

Creation and import accept `--runtime-profile ID`. To change an enrolled ship,
Stop first, then submit `configure` with its fresh revision, existing ports and
loom, `--runtime-profile ID --confirm-runtime-change`. The runner preflights the
artifact and requires a stopped, resolved state with removed containers. It
journals old/new artifacts, preserves the pier and stays stopped. Start is a
separate operation. Preserve a backup before starting an untested build: runtime
replay changes can be irreversible. Old clients omitting the field preserve the
current pin. Schema67 fences older writers without rewriting original bodies.

## Historical rc.3 behavior and qualification

The official default remains Vere `4.6-5a7614f` from the frozen `tloncorp/vere`
image. Omabit's release channel and a ship's Vere runtime are separate choices.
A manager update never changes the meaning of an existing runtime selection.

## Review development for issue #22

This branch adds a finite reviewed catalogue. **The rc.3 core runtime, local
0.5.0 upgrade and full isolated shell activation pass.** The
[manual review guide](runtime-profiles-manual-review.md) identifies the complete
prerelease delivery; it is not a stable release or an automatic production update.
Follow the [review guide](runtime-profiles-review.md) for evidence limits and
the prerequisites for an eventual owner-operated import.

| Profile | Contract |
| --- | --- |
| omitted, or `vere-4.6-5a7614f` | The same immutable official artifact used by 0.5.0 |
| `vere-4.6-jsonfix-0774857c` | Reviewed same-base #1145 JSON decoder backport; exact binary, image and archive pins |

The authoritative metadata is in `internal/runtimeprofile/profiles.go` and the
runner's offline `runtime-profiles` output. The catalogue binds Linux/AMD64,
Vere's banner, executable SHA256, OCI manifest/config digests, source base, patch
revision, review URL, and the retained distribution archive. Arbitrary paths,
shell commands, Docker tags and automatic PR builds are not runtime choices.
The official default has not been replaced globally.

Choose a profile when registering or importing a pier. Guided import binds the
preview and submission to the same selection and checks artifact availability
before inspecting the source. An existing managed pier's binding is immutable in
this first implementation. A proposed different runtime returns
`runtime_transition_required`; a stock downgrade from the JSON fix returns
`runtime_incompatible`. An old client's omitted setting preserves the pin.

A selection follows the stable managed identity through Start, replacement,
manager restart, host startup resume, maintenance, recovery and stopped-pier
handoff. Container intents and recording jobs retain it before execution.
Runtime details distinguish requested metadata, recorded launch metadata and the
last observed container image ID. A saved request is not proof of a live process.

## Patched JSON decoding and replay compatibility

The backport is based on Vere commit
`5a7614f69bbc783736b08965fdc0a2534bb42e82` and
[PR #1145](https://github.com/urbit/vere/pull/1145) revision
`fcc9dd214ae4af07e683490a6cc3f2ded1ab7d47`. Its compatibility requirement is
`vere-json-array-error-1145-v1`. The banner `4.6-82f4558` alone is not provenance.

Once this runtime commits an event that exercises the corrected malformed-JSON
path, stock Vere 4.6 can crash replay. A numerically newer official version is
not an eligible successor until inclusion of the correction and compatibility
are reviewed. Reverting to a pre-patch backup is a separate recovery with loss
of subsequent events, not an ordinary binary downgrade.

The latest imported epoch must already match the selected runtime. A version
transition with a checkpoint behind the event log is refused before Vere is
executed. The diagnostic requires a reviewed checkpoint/preparation plan; it
never follows Vere's generic “downgrade” wording, edits epoch metadata, disables
mug validation, or tries another runtime. The incident's approved offline
`play --yolo` procedure is specific to its source revision and is not automated.

## Artifact installation and retention

The local owner command `urbit-runner runtime-profile-install --profile ID
--archive FILE` validates the exact compiled archive hash, stages it atomically,
and loads its immutable image into the runner's own rootless engine. It opens no
pier or management database and selects no ship. Run it as the installed
pier-owning user, in that runner's namespace. A collector cannot upload an
arbitrary executable through this interface.

Retain `.local/share/urbit-runner/runtime-profiles/ID/image.tar.gz` under the
runner's home and its Docker image while any active pier or supported recovery
requires it. Start verifies the archive, daemon image/platform, and custom binary
checksum before executing Vere against a pier. Missing or corrupt artifacts fail;
there is no pull of a replacement stock runtime. Existing Stop remains available
if an archive is lost. No new cleanup command removes these artifacts.

The `runtime-profiles-v1` capability advertises exact catalogue metadata. The
shared `runtime.profile-preflight.v1` operation (CLI/MCP) verifies the selected
runner and available image without opening a pier. Signed handoff v3 includes
profile and correction requirement. A capable matching destination must pass
artifact preflight before the collector fences the source; the destination checks
again before reservation. Metadata does not transfer or authorize a binary.
Legacy handoff v1/v2 signatures remain unchanged.

## Persistence and upgrade boundaries

Schema66 adds optional profile fields and an older-reader fence. Original
instance, operation, permission and recording rows are preserved during migration.
An omitted field always resolves to the frozen official 4.6 artifact, even if a
future release adds other choices. Historical receipts are not relabeled.

Older runners must refuse schema66. Do not lower a schema marker, reset state or
re-enroll ships to make an older binary run. The published stable support window remains [0.4.0→0.5.0](releases/0.5.0/qualification.json). The
[0.5.0→rc.3 review rehearsal](diagnostics/2026-11-24-runtime-profiles-upgrade.json)
now passes for the local owner-user case; it is not a published stable support
window. Source tests are not installed-upgrade qualification. See [upgrade policy](upgrade-policy.md).

## Existing runtime settings

Vere's control socket stays disabled by default with `--no-conn`. Explicit owner
opt-in is described in [runtime settings](development-runtime.md); the reviewed
JSON patch does not claim to fix unrelated control-socket issues.

Long-running processes use `--snap-time 60` (minutes in Vere 4.6), and each launch
records that argument. Existing two-minute launches remain verifiable until an
explicit replacement. Offline utilities have separate command contracts.
Configured loom is virtual address space, separate from container RAM limits.
No loom-utilization estimate is inferred from container memory measurements.

The management service owns durable operations and can restart without replacing
running containers. Startup resume still requires its verified clean-shutdown
policy and one journaled attempt per boot. See [services](services.md).
Standalone qualification reports, private-chain factories and project factories
retain their own official-only contracts; the new profile is for ordinary managed
ships. They cannot adopt a custom-runtime pier through a report conversion.
