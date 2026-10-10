# Omabit release pill

## 0.1.0: choose an existing pill

The automated pill builder is deferred to **0.2.0**. For 0.1.0 the user provides
the Recommended Apps pill. New-ship forms expose exactly two choices:

- **bare bootstrap**: no `-B`/`-u` override; Vere selects its normal
  bootstrap.urbit.org pill.
- **Recommended Apps**: use the configured direct HTTPS release-asset URL. Omabit
  downloads, stages and verifies the artifact, then supplies `-B` on first boot.

Bare is the initial selection. Choosing Recommended Apps without a configured URL
blocks registration with a clear message; it never falls back to another pill.
These choices apply to new comets, owned-key ships and development fakes. Existing
piers and subsequent runtime starts use initialized resume, not pill provisioning.

### Artifact handoff

Upload **`omabit-recommended-apps-0.1.0.pill`** to a versioned release, for example
`v0.1.0-rc.1`, and provide the **direct asset download URL**, not the release's HTML
page. A GitHub URL would have this shape:

```text
https://github.com/OWNER/REPO/releases/download/v0.1.0-rc.1/omabit-recommended-apps-0.1.0.pill
```

Set that URL in Omabit's **Recommended Apps pill URL** plugin setting
(`recommendedPillUrl`). The old `defaultBootPill` value is accepted as a fallback
source for this explicit choice only; it cannot override bare bootstrap.
No URL is bundled yet because the supplied artifact has not been published or
boot-qualified.

### Supplied artifact

The user supplied `omabit-brass.pill`, **57,011,409 bytes** (about 54.4 MiB), with
SHA-256 `51f0be723918ab7feaaf28b3c599f36f0f0fca6e68453f84517ba1659f07d57e`.
This fits both Omabit's bound and GitHub's release-asset size bound. Keep the binary
outside Git; its proposed release asset name remains the one above.

Reported contents: `%base`, `%webterm`, `%groups`, `%trunk`, `%grubbery`, `%omart`,
`%pals`, `%hawk`, `%obelisk`, `%jars`, `%silo`, `%landscape`, from the intended
distributors. These contents are provider-reported, not independently boot-verified.
The supplied list includes `%landscape` and does not list `%mcp`; the intended
composition below retains that distinction for release review.

CLI equivalents (all require a selected runner; comets/owned ships also require
their usual network acknowledgment):

```text
create-comet INSTANCE ... --bootstrap bare
create-comet INSTANCE ... --bootstrap recommended --pill HTTPS_ASSET_URL
create-ship --ship ~SHIP --key-file FILE ... --bootstrap bare
create-ship --ship ~SHIP --key-file FILE ... --bootstrap recommended --pill HTTPS_ASSET_URL
```

`create-fake` supports the same bootstrap flags. CLI `--pill` also accepts a local
file for artifact validation. Omitting bootstrap flags defaults new registrations
on updated runners to bare. The selector is fixed at registration and the staged
content hash/size becomes part of its durable identity/replay checks.

## Confirmed composition

The user-confirmed first-release composition is recorded in
[`pill-plan.json`](pill-plan.json). This is the user's intended composition for the
provided artifact, not a verified release manifest. Omabit 0.1.0 provisions the
supplied pill; automated composition and build provenance tooling belongs to 0.2.0.

| Desk | Source / publisher |
| --- | --- |
| `%webterm` | Upstream default |
| `%groups` | Upstream default |
| `%base` | Upstream default |
| `%grubbery` | `~ricsul-bilwyt` |
| `%jars` | `~nattyv` |
| `%hawk` | `~dister-migrev-dolseg` |
| `%omart` | `~dister-dozzod-sarlev` |
| `%pals` | `~tocwex` |
| `%obelisk` | `~dister-nomryg-nilref` |
| `%mcp` | `~daplyd` |
| `%trunk` | `~ricsul-bilwyt` |
| `%silo` | `~dister-nocsyx-lassul` |

These 12 desks are required for the first release. This list supersedes the earlier
draft pill bundle in the architecture plan. `%omachan` is explicitly deferred from
the initial 0.1.0 pill at the user's direction.

`%pals` stays on `~tocwex` for now. Moving its publisher to `~paldev` is a future
explicit release change after that ship's issues are resolved, not an automatic
fallback or a reason to block the current `%pals` source.

**`%omachan` does not exist yet and does not block 0.1.0.** Its intended publisher
remains `~dister-dozzod-sarlev`; implementation/publication and later inclusion are
separate work. It is recorded under `deferred_desks`, not in the initial build set.

The pill supplies a working bundled baseline. New ships should follow the declared
publishers and receive normal Urbit OTA updates after connecting. Recording the
versions baked into a pill is build provenance, not a runtime version lock or an
Omabit-managed replacement for desk OTA.

## Release assets and hosting

Recommended distribution: attach the supplied pill as a versioned release asset,
with a checksum alongside it. A richer build manifest can follow with the 0.2.0
builder. Example artifact names:

```text
omabit-recommended-apps-0.1.0.pill
SHA256SUMS
```

The pill has its own version and compatibility metadata. Omabit software releases
can reference an already-published pill without rebaking it or embedding a large
pill inside every desktop/runner package. Pin immutable artifact URLs and hashes;
a default-release pointer can select a newer version without replacing old assets.

Release attachments are sufficient when the selected host accepts the actual file
size. Check the forge and reverse-proxy limits before publication. If the artifact
exceeds those limits, store it in versioned HTTPS object storage and link the same
digest/URL from the release manifest. Hosting location does not change the pill's
identity or the verification contract.

For comparison, [GitHub release assets](https://docs.github.com/en/repositories/releasing-projects-on-github/about-releases#storage-and-bandwidth-quotas)
must each be under 2 GiB. That is a hosting constraint, independent of Omabit's
4 GB provisioning allowance. The project's release host/location is not yet fixed
by this composition plan.

## Provisioning state

From source version 0.22.2, Omabit's own boot-pill byte bound is **4 GB
(4,000,000,000 bytes)**. The common bound covers URL/local staging, upload metadata,
SSH framing, runner reception and the pinned candidate downloader. It is a resource
bound imposed by Omabit, not a limit inherited from Vere or Arvo. Boundary tests use
sparse-file metadata and short failing streams, not a full 4 GB live upload/boot.

The collector currently resolves a selected URL or local file, stages/hashes it and
uploads it to the selected pinned runner. The runner verifies and caches the bytes;
registration stores the artifact descriptor and explicit Start uses `-B`. Boot keys
are supplied separately at ship creation. Initialized ships resume their existing
pier without applying the pill again.

Source version 0.23 implements this selection for all new ship types. Capability
`bootstrap-selection` identifies supporting runners. The additive schema-13 update
preserves old registrations' recorded behavior and stops older runners from
ignoring a new selection. The supplied Recommended Apps artifact still awaits
publication and live qualification; fixtures verify selection, byte binding and resume.

## Deferred automated builder (0.2.0)

1. Obtain the required desks from their declared publishers, check the combined
   starting state and dependencies, and record the versions actually baked.
2. Define the first-boot agent state and preserve or establish each desk's normal
   publisher/OTA relationship. `%omachan` is not a prerequisite for this build.
3. Build in a clean, disposable environment with a recorded upstream base and Vere
   compatibility target. The current tested runtime candidate is `4.6-5a7614f`;
   compatibility with this full bundle still needs verification.
4. Boot disposable ships and verify the actual desk contents, agent startup, web
   assets and expected application behavior, followed by initialized stop/resume
   and normal publisher-driven OTA after network connection.
5. Generate the release manifest from the resulting bytes: version, size, SHA-256,
   immutable URL, base/desk revisions, publishers, runtime compatibility and source
   build revision. Publish through the trusted release channel.
6. Qualify automated composition/manifest-driven release provisioning for comets
   and owned ships. This is not required for 0.1.0's two-choice supplied-pill flow.

Pill publication affects new boots. Existing ships continue through their separate
Urbit OS/desk update paths; a new pill is not an in-place update to an existing pier.
