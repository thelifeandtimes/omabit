# Native Omarchy GitHub installation

Run as your ordinary user inside the Omarchy desktop:

```bash
omarchy plugin add https://github.com/thelifeandtimes/omarchy-omabit.git --enable
```

`omarchy plugin install` is an alias for `add`. This uses the standard clone,
validation, trust confirmation, placement and enable workflow. The plugin ID is
`thelifeandtimes.omabit`. Host setup remains the next explicit GUI action.
See the [publication and verification record](diagnostics/2026-10-15-native-plugin-publication.md).

## Why a generated plugin repository

Omarchy requires `manifest.json` at the root of the cloned repository and checks
its entry points. Its native installer does not invoke arbitrary build/install
hooks. The main Omabit repository is an authoritative Go/QML monorepo; the plugin
is under `plugins/omabit`, and compiled tools are published as release assets.

`omarchy-omabit` contains the root-layout plugin generated from the verified frozen
release archive, plus small Python launchers. No Go compiler, local extracted
bundle, Git LFS or committed binary blobs are needed. The published RC.2 assets
are not rebuilt or replaced to add this installation path.

## Binary preparation

On first use the launchers retrieve the three Linux/AMD64 assets from the pinned
versioned GitHub release. Names, byte sizes, source commit and SHA-256 hashes are
recorded in `bootstrap/release.json`. HTTPS redirects remain HTTPS; truncated,
oversized or hash-mismatched downloads are refused.

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

## Generate a distribution

After committing the distribution code, download/verify a frozen release and run:

```bash
python3 -B scripts/package-plugin.py /path/to/verified-release-assets \
  --output /path/to/new-generated-plugin
omarchy plugin validate /path/to/new-generated-plugin
```

The output must be new. `DISTRIBUTION.json` binds the release-source commit, archive
hash, version and distribution-generator commit. Publish the generated files to
the plugin repository; make future source changes in the monorepo and regenerate.
Binary assets remain in the main repository's GitHub releases.

`scripts/test-native-plugin.py` covers concurrent cold start, verified offline
reuse, modified-cache refusal, failed download/retry, size/hash/HTTPS validation,
platform and path guards, generated launchers and frozen-release integrity.
