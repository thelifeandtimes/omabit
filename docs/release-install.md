# Omabit 0.6.0 bundle

This Linux x86-64 release contains matching `omabit`, `urbit-collector` and
`urbit-runner` binaries, the Omarchy interface, CLI references and three agent
skills. It introduces immutable per-ship runtime selection, owner-installed local
builds and schema67. Read the [release notes](releases/0.6.0/notes.md) for the
upgrade support window and tested limits.

## Update an existing Omarchy installation

Update an unmodified stable Git installation with:

```sh
omarchy plugin update thelifeandtimes.omabit
```

This follows protected stable master and its verified artifact pins. Preserve local
plugin edits before updating. A desktop update retains profiles, enrollments and preferences
and does not automatically update a local or remote runner.

Once the verified 0.6.0 desktop has loaded, use **Setup → Local services &
updates → Check services → Update services** and authorize the OS prompt. The
supported stable starting releases are0.4.0 and0.5.0. Older installations must
follow their documented intermediate upgrades first. Finish pending management
work before updating; do not reset, delete or re-enroll ships to perform an update.

The upgrade retains the existing owner, piers, runner identity, access grants,
configuration, operation history and intended running/stopped state. Existing
ships keep their original official runtime. Selecting the reviewed patch for a
new import is a separate explicit operation.

## Inspect or install an extracted archive

Verify the downloaded archive against the release's published SHA256, then verify
its extracted contents:

```sh
sha256sum --check SHA256SUMS
./build/omabit --help
./build/urbit-collector help
./build/urbit-runner --help
```

For a new desktop installation, the bundle provides `./install.sh --enable`.
For an existing unmodified archive installation, it provides
`./install.sh --replace --enable`. Replacement uses a cold desktop reload; unlock
the desktop first. Edited or incompatible installations refuse replacement and
need review. Follow the qualified release's specific update route for a Git-based
plugin installation.

## Hosting and standalone service updates

Hosting requires Linux/AMD64, systemd/cgroup v2, supported distribution-native
Docker27.2.0–29.x with API1.47, RootlessKit, slirp4netns and OpenSSH. The engine
runs rootlessly under the pier-owning user. A desktop managing only a remote
runner needs no local Docker engine. Compatibility is not proof that every
operating-system/engine combination has been tested; consult the release record.

Guided Setup requests authorization for fresh host installation. Existing-host
management upgrades do not replace unrelated Docker installations or upgrade OS
packages. Standalone/headless hosts can prepare a new rootless launcher bundle
with the new runner and apply `install-host` using the **existing owner and public
operator key**; see [guided service upgrades](guided-service-upgrades.md) for the
reviewed commands and access-preservation contract.

Retry interrupted activation with the same target executable and bundle. After
schema67 commits, older runners refuse the database. Use compatible new code for
recovery; never lower the schema marker or erase journals to force a downgrade.
Switching the desktop back to stable does not downgrade runner state.

## Reviewed runtime and existing piers

The optional `vere-4.6-jsonfix-0774857c.tar.gz` asset packages the exact reviewed
Vere4.6 JSON-decoder backport. It has its own immutable profile, image digests,
executable checksum and compatibility requirement. Stage it with the owner-only
`runtime-profile-install` command and verify runner preflight before selecting it
for an import. A normal Start never builds a patch or substitutes a missing
artifact. Read [runtime policy](runtime-policy.md) and the
[owner-operated import prerequisites](runtime-profiles-review.md).

A pier that has used the corrected error path must remain on the fixed runtime
or a reviewed compatible successor. Stock4.6 can fail during replay. A stopped
pier backup, verified native-process shutdown and an explicit runtime selection
are prerequisites to importing an existing live identity. Never boot two copies
of the same ship. This bundle does not authorize or perform a live move.

## Pills, agents and applications

Recommended Apps retains the existing59,351,751-byte pill with SHA256
`85635c0b17a138a6473580ca362475faadf281e09cc9fa3746cfc698cc054f62`.
The compatibility URL is a frozen release alias; changed bytes refuse verification.
Initialized piers resume without reapplying a pill or identity key.

Agent Help opens the default Omarchy agent with matching guidance. The bundle
includes `omabit-operator`, `omabit-app-developer` and `omabit-plugin-developer`,
each with `SKILL.md` and supporting references. `omabit operations` lists the
shared CLI/MCP catalog. See [README](../README.md), [CLI help](cli/README.md),
[Agent Help](agent-skills.md) and [application preflight](app-development-preflight.md).
