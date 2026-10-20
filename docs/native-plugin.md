# Native Omarchy GitHub installation

**RC.14** uses the `native/v1` bootstrap and isolated test profiles described in
[desktop development](desktop-development.md). It uses stock Omarchy's retained-service
mechanism and verified release-specific QML paths. Older static-wrapper recovery
instructions remain below for RC.13 and earlier installations.

Run as your ordinary user inside the Omarchy desktop:

```bash
omarchy plugin install https://github.com/thelifeandtimes/omabit.git --enable
```

`omarchy plugin install` is an alias for `add`. This uses the standard clone,
validation, trust confirmation, placement and enable workflow. The plugin ID is
`thelifeandtimes.omabit`. Host setup remains the next explicit GUI action.
The earlier [generated-repository record](diagnostics/2026-10-15-native-plugin-publication.md)
is historical; the canonical monorepo layout below supersedes that install URL.

## One authoritative, directly installable repository

Omarchy requires `manifest.json` at the repository root and checks its entry points.
The root manifest loads `native/v1/Service.qml` and `native/v1/Panel.qml`. The retained
service prepares and activates a complete verified UI/tool generation; the bar
adapter attaches to that same generation. Source UI lives in `plugins/omabit/`,
but native installations execute its frozen release copy from the cache.
Flat binary-bundle installations retain their existing `bin/` layout and snapshot
installer rather than the native bootstrap/profile lifecycle.

Generic runner/collector entry points are in `backend/cmd/`; the Omarchy-specific
Go helper remains `cmd/omabit/`. Shared libraries remain private to the one Go
module under `internal/`. No Go compiler, local extracted bundle, Git LFS or
committed binary blobs are needed for plugin installation. Published RC.2 assets
are not rebuilt or replaced by this repository organization.

The RC.14 distribution selects **RC.14**, schema21 and owner-user host setup,
including compact setup approval, one machine-use switch, guided identity-aware
imports with progress, named pier leaves, and RC.7's persistent log following.
The updated runner, collector and QML are distributed together. Existing hosts
need a matching runner update for `guided-pier-import`/`named-pier-directories`.
RC.9 additionally publishes Ames for new live enrollments and exposes operator-only
retained import recovery. Its UI recovery action requires an updated runner with
`recover-import`; the three published tools and QML are pinned together.
The new UI rejects helpers without the owner-user setup contract.
RC.10 adds loom/native settings, preserves omitted native values from older
collectors and signs nondefault settings in cross-host transfers. Its frozen
binaries passed real fake/comet native execution and repeated RC.9 management
updates with running/stopped state and identities preserved.
RC.11 puts loom/storage selection under Advanced settings, restricts new loom
choices to30–34 and makes native control CLI-only. Its final runner passed an
in-place update of the user's running comet, preserving container/process IDs,
configuration, storage identities and access/enrollment records.
RC.12 retains that layout and adds the reviewed recovery, replay, credential and
transactional-update fixes. Its frozen bundle passed actual rootless runtime and
management-upgrade qualification, guest power cuts, authenticated Ames roundtrips,
mapped desktop and real clipboard checks. Existing RC.10/RC.11 schema19 hosts use
the [explicit offline bridge](management-offline-upgrade.md); updating the desktop
does not implicitly migrate a runner host.
RC.13 adds distribution-native Pacman/Debian/Ubuntu package authentication, compatible
Docker27.2.0–29.x/API1.47 checks and shared compatibility reporting. Its published
tools passed clean reproducible builds and isolated installation checks; fresh host
acceptance on the two requested Omarchy versions remains pending. See the
[RC.13 notes](release-rc13.md) and [schema21 upgrade guide](host-compatibility-upgrades.md).
RC.14 adds the retained bootstrap, isolated test profiles and shared runtime
version display. See the [RC.14 notes](release-rc14.md) for tested scope and the
planned workstation/Omarchy acceptance sequence.
The service resolves packaged defaults when Omarchy passes only explicit inline
settings. `recommended_pill_default` in monitor status verifies the effective
source, rather than only the value recorded in the manifest.

## Bundle preparation

On first use the bootstrap retrieves the complete Linux/AMD64 desktop archive from
the pinned versioned GitHub release. Archive and binary names, byte sizes, source
commit and SHA-256 hashes are recorded in `native/bootstrap/release.json`. HTTPS
redirects remain HTTPS; truncated, oversized or hash-mismatched downloads are refused.
Tool pins name an already published build and advance after release publication;
they are not a self-referential hash of the Git commit that stores those pins.

The UI and tools are prepared together under a lock and atomically promoted into
`$XDG_CACHE_HOME/omabit/releases/<descriptor-digest>/` (normally under `~/.cache`).
The active UI receives the matching tools from that generation. Failed preparation
does not publish a partial generation. Retained authenticated bundles work offline;
new generations use different paths for nested QML/JavaScript too.

The standalone `native/bin/` launchers retain their separate three-binary verified
cache under `omabit/native-binaries/`. For a selected test profile/release, use
`scripts/omabit-dev.py exec` so the command receives that profile's state roots and
matching tool version. The checkout remains clean for Git updates. Python3.11+
and Linux x86-64 are required.

Downloading these user-owned tools does not create an account, install services,
enroll a host or start a ship. Those remain explicit **Set up this computer** actions
using the normal OS authorization flow.

## Updates

For the native plugin:

```bash
omarchy plugin update thelifeandtimes.omabit
omarchy shell thelifeandtimes.omabit.bootstrap status
```

This uses Omarchy's Git fast-forward workflow. The bootstrap prepares the selected
bundle, waits for busy work, validates activation and resumes observation. The Git
command can finish before background preparation does: check `phase`, `loaded_version`,
`profile` and `error` in bootstrap status. The ordinary default profile follows
published pins; explicitly selected development releases remain pinned until
`python3 -B scripts/omabit-dev.py select default --release follow` is run from the
plugin/source checkout. Runner host upgrades remain separate.
The snapshot/binary-bundle installer is a different ownership model: it does not
replace a git-managed checkout, and native `plugin add` refuses an existing plugin
with the same ID. Choose the native path for a fresh standard Omarchy install.

### Legacy static-wrapper recovery

A successful Git update/rescan does not guarantee that the long-running shell has
loaded the new service and QML. On sourceport's RC.12→RC.13 update, the files and
tool pins were RC.13 but live monitor IPC still reported RC.12, the collector feed
had stopped, and reload-time bindings referenced a null service. Both ships and
the saved runner connection were healthy. See the
[diagnosis and verified recovery](diagnostics/2026-10-19-rc13-stale-update.md).

On an unlocked desktop, after any setup/submission has finished, recover with:

```bash
omarchy restart shell
omarchy shell thelifeandtimes.omabit.monitor status
```

For an unpinned published desktop, check that `version` matches the installed root
manifest, then reopen the fleet. RC.14 bootstrap diagnostics additionally identify
an intentionally selected older/development release.
If it remains empty, inspect the bundled collector's `runners` and `inventory`
results rather than treating the missing display as deleted enrollment.

An existing host's `setup-status` may correctly report `ready:true`,
`host_present:true`, `can_setup:false`: it is already set up. Updating/reloading the
desktop does not turn it into a fresh installation or upgrade the installed runner.
Existing-host runner upgrades follow the [schema21 procedure](host-compatibility-upgrades.md).

### Reinstalling in an existing desktop session

Removing and reinstalling the plugin at the same path can leave the long-running
shell executing its previously loaded QML. This was confirmed on sourcebox: the
checkout and tool pins were RC.4 while the live service still reported RC.3 and
displayed the old welcome text. Disable/remove/rescan alone did not clear it.

After reinstalling into a session that already loaded an older Omabit version,
finish any setup/submission and run on the unlocked desktop:

```bash
omarchy restart shell
omarchy shell thelifeandtimes.omabit.monitor status
```

The reported `version` must match the installed root `manifest.json`. Verify the
actual menubar after this restart; a clean checkout, manifest validation and binary
hashes alone do not establish which UI the running shell is executing. The shell
restart preserves the independent runner/engine services and accepted ship jobs.
See the [live diagnosis](diagnostics/2026-10-15-stale-native-ui.md).

## Legacy flattened exports

The previous `omarchy-omabit` microrepository is a historical distribution. New
installs use the canonical `/omabit` command above. To reproduce a flattened export
from a frozen archive after committing the distribution code:

```bash
python3 -B scripts/package-plugin.py /path/to/verified-release-assets \
  --output /path/to/new-generated-plugin
omarchy plugin validate /path/to/new-generated-plugin
```

The output must be new. Its `DISTRIBUTION.json` binds the release-source commit,
archive hash, version and export-generator commit. This export is not required for
the canonical installation. Binary assets remain in this repository's releases.

`scripts/test-native-plugin.py` covers concurrent cold start, verified offline
reuse, modified-cache refusal, failed download/retry, size/hash/HTTPS validation,
platform and path guards, generated launchers and frozen-release integrity.
