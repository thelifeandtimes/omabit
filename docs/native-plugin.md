# Native Omarchy GitHub installation

Run as your ordinary user inside the Omarchy desktop:

```bash
omarchy plugin add https://github.com/thelifeandtimes/omabit.git --enable
```

`omarchy plugin install` is an alias for `add`. This uses the standard clone,
validation, trust confirmation, placement and enable workflow. The plugin ID is
`thelifeandtimes.omabit`. Host setup remains the next explicit GUI action.
The earlier [generated-repository record](diagnostics/2026-10-15-native-plugin-publication.md)
is historical; the canonical monorepo layout below supersedes that install URL.

## One authoritative, directly installable repository

Omarchy requires `manifest.json` at the repository root and checks its entry points.
The root manifest now loads `native/Service.qml` and `native/Panel.qml`, which wrap
the shared QML under `plugins/omabit/`. The service wrapper selects `native/bin/`
for its tools. Flat binary-bundle installations retain their existing `bin/` layout.

Generic runner/collector entry points are in `backend/cmd/`; the Omarchy-specific
Go helper remains `cmd/omabit/`. Shared libraries remain private to the one Go
module under `internal/`. No Go compiler, local extracted bundle, Git LFS or
committed binary blobs are needed for plugin installation. Published RC.2 assets
are not rebuilt or replaced by this repository organization.

## Binary preparation

On first use the launchers retrieve the three Linux/AMD64 assets from the pinned
versioned GitHub release. Names, byte sizes, source commit and SHA-256 hashes are
recorded in `native/bootstrap/release.json`. HTTPS redirects remain HTTPS; truncated,
oversized or hash-mismatched downloads are refused.
Tool pins name an already published build and advance after release publication;
they are not a self-referential hash of the Git commit that stores those pins.

All three tools are prepared together, under a lock, and the complete verified
directory is promoted atomically into
`$XDG_CACHE_HOME/omabit/native-binaries/<manifest-digest>/` (normally under `~/.cache`).
This keeps the runner next to the Omabit setup binary and coordinates concurrent
startup queries. Failed downloads leave no partially published executable set.
Existing cached tools are checked before use and work offline. New pins get a
different directory. The checkout remains clean for Git updates; global PATH and
Tend's legacy CLI are not modified. Python 3.11+ and Linux x86-64 are required.

Downloading these user-owned tools does not create an account, install services,
enroll a host or start a ship. Those remain explicit **Set up this computer** actions
using the normal OS authorization flow.

## Updates

Finish desktop submissions and unload the plugin before replacing its QML:

```bash
omarchy plugin disable thelifeandtimes.omabit
omarchy plugin update thelifeandtimes.omabit
omarchy plugin enable thelifeandtimes.omabit
```

This uses Omarchy's Git fast-forward workflow. Runner host upgrades remain separate.
The snapshot/binary-bundle installer is a different ownership model: it does not
replace a git-managed checkout, and native `plugin add` refuses an existing plugin
with the same ID. Choose the native path for a fresh standard Omarchy install.

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
