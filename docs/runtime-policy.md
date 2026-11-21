# Runtime policy: immutable choices per ship

The official default remains Vere `4.6-5a7614f` from the frozen `tloncorp/vere`
image. Omabit's release channel and a ship's Vere runtime are separate choices.
A manager update never changes the meaning of an existing runtime selection.

## Review development for issue #22

This branch adds a finite reviewed catalogue. **Native qualification and the
0.5.0→review upgrade rehearsal are still pending. This source is not yet a
production install recommendation.** Follow the [review guide](runtime-profiles-review.md)
for evidence status and the prerequisites for an eventual owner-operated import.

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
re-enroll ships to make an older binary run. The tested previous-stable support
window remains [0.4.0→0.5.0](releases/0.5.0/qualification.json) until a matching
review upgrade rehearsal is recorded. Source tests are not installed-upgrade
qualification. See [upgrade policy](upgrade-policy.md).

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
