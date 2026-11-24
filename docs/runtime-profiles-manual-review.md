# Omabit 0.6.0-rc.3 manual review

This is the complete Linux/AMD64 prerelease candidate for issue22, including the
native UI, matching collector/runner/CLI, skills, recommended pill and reviewed
Vere archive. It preserves the official default runtime and adds an explicit
immutable runtime choice for each ordinary managed ship.

Download from [v0.6.0-rc.3](https://github.com/thelifeandtimes/omabit/releases/tag/v0.6.0-rc.3).
The release notes identify the exact **distribution checkout** to use below.
The product source is `747dd810c10b74b4fbcdc10f617e6b15dd545a45`; that source tag
predates the distribution pins. Do not install the source tag as the plugin.
Stable/latest remains0.5.0; publishing this candidate changes no installed service.

## Download and verify

Use a new directory. Keep the archive and checksums with your review materials.
The commands need Git, curl and Python3; they do not install or update services.
Replace `DISTRIBUTION_REVISION` with the exact revision in the release notes.

```bash
review_dir="$HOME/Downloads/omabit-0.6.0-rc.3-review"
mkdir -m 700 "$review_dir"
git clone --no-checkout --single-branch --branch codex/issue-22-runtime-profiles \
  https://github.com/thelifeandtimes/omabit.git "$review_dir/source"
git -C "$review_dir/source" checkout --detach DISTRIBUTION_REVISION
mkdir "$review_dir/assets"
(
  set -eu
  cd "$review_dir/assets"
  release_url=https://github.com/thelifeandtimes/omabit/releases/download/v0.6.0-rc.3
  for asset in SHA256SUMS release.json omabit-0.6.0-rc.3-linux-amd64.tar.gz \
    omabit-linux-amd64 urbit-collector-linux-amd64 urbit-runner-linux-amd64 \
    omabit-skills-0.6.0-rc.3.tar.gz omabit-recommended-apps-0.6.0-rc.3.pill \
    vere-4.6-jsonfix-0774857c.tar.gz; do
    curl --fail --location --output "$asset" "$release_url/$asset"
  done
  printf '%s  SHA256SUMS\n' c3220d8db6e798ad06a2257c254c65031318a1281b2ac7c50efaea0a2e6660c7 | sha256sum --check -
  sha256sum --check SHA256SUMS
)
```

Stop if a command or checksum fails. These nine files are the original qualified
bytes; no rebuild is needed. Bundle SHA256:
`f1676a51bede3302327917c9e80b8661bf482e47a8e40633903e60ac435037f7`.

## Review the desktop with empty separate state

For an existing normal0.5.0 Omabit plugin, its compatible bootstrap can load this
sealed release explicitly. This switches the normal widget to a new desktop
profile; it leaves the previous profile's data intact. It does not modify any
separately named DEV widget or upgrade a runner. Use the existing normal plugin,
not an older/custom DEV wrapper with different state paths.

```bash
cd "$review_dir/source"
python3 -B scripts/omabit-dev.py status > "$review_dir/desktop-before.json"
python3 -B scripts/omabit-dev.py import-release \
  "$review_dir/assets/release.json" \
  "$review_dir/assets/omabit-0.6.0-rc.3-linux-amd64.tar.gz"
python3 -B scripts/omabit-dev.py profile-create review060
python3 -B scripts/omabit-dev.py select review060 \
  --release b83468b7724020d344a1221b9d3093f3f562e46639c94a39f02d2e4c43d2bc52
omarchy shell thelifeandtimes.omabit.bootstrap status
```

If `review060` already exists, retain it and use a different new name in both
commands. Activation may take a few seconds. Require `phase:"ready"`, an empty
`error`, `loaded_version:"0.6.0-rc.3"`, source `747dd810…` and the full loaded
release digest above. Open Omabit and confirm **PRERELEASE**, Setup and the empty
Fleet. Inspect Add a ship/import's runtime choices without submitting an import.
Local service setup is disabled in this profile. Profiles isolate desktop data;
they do not isolate host services or piers.

To restore the prior desktop selection exactly, without modifying its release
pin or data:

```bash
python3 -B - "$review_dir/desktop-before.json" <<'PY'
import json, subprocess, sys
state = json.load(open(sys.argv[1]))
name = state['selected']
subprocess.run([sys.executable, '-B', 'scripts/omabit-dev.py', 'select', name,
                '--release', state['profiles'][name]['selection']], check=True)
PY
omarchy shell thelifeandtimes.omabit.bootstrap status
```

Keep the new review profile and downloads; there is no reset/cleanup prerequisite.

## Optional review against an existing local0.5.0 runner

This is a separate owner-operated upgrade. The tested starting state is the
published0.5.0 local owner-user installation, schema65. Earlier stable users must
first follow their documented chain to0.5.0. Remote service updates, active
recording-worker upgrades, power-loss recovery and custom DEV service layouts are
outside the qualified window. Do not apply the normal setup updater to a custom
`/srv/dev` installation.

Restore the original desktop profile first. To use rc.3 with that profile, select
its existing name with the exact release digest above instead of creating another
profile. This retains its enrollment and credentials. In **Setup**, review the
selected runner's identity, existing installation and proposed service update.
Then explicitly authorize that update if desired. A fresh-install/reset offer is
not the tested path. A desktop selection alone never authorizes a service update.

The update advances runner state to schema66. Preserve its database, identities,
credentials, piers, settings and operation history. Compare Fleet before/after;
stopped ships must stay stopped, running peers must retain their processes, and
existing ships retain the official pin. An uncertain update must continue its
original attempt/receipt. Do not force a downgrade or edit the schema marker;
the actual0.5.0 writer refuses the newer state.

For runtime operations, use a fresh disposable fake on a deliberately selected
review runner. Stage the exact reviewed archive as that runner's owner, using the
matching rc.3 tool, then require successful artifact preflight before creating or
importing anything. The [runtime review guide](runtime-profiles-review.md#stage-the-artifact-before-inspecting-a-patched-pier)
contains the commands and exact runtime metadata. Use both runtime choices in
separate fakes, check runtime details, Stop/Start and Pack, and retain their pins.
Never change a ship's runtime to test a downgrade.

Protected existing ships and the owner's later `~sarlev` import are separate
decisions with the [identity safeguards](protected-ships.md). Preparing this
candidate did not inspect or operate those ships.

## Evidence and feedback

- [Native core scenario](diagnostics/2026-11-24-runtime-profiles-native.json):
  all14 checks, both runtime pins, replacement, maintenance, controlled recovery,
  stopped fake import and16-container lifecycle/utility audit.
- [Published0.5.0 upgrade](diagnostics/2026-11-24-runtime-profiles-upgrade.json):
  actual native QML flow, denial/retry, schema65→66, preserved identity/history/access,
  existing processes, manager reopen and genuine older-writer refusal.
- [Full Omarchy shell](diagnostics/2026-11-24-runtime-profiles-full-shell.json):
  cold/hot/rescan/restart with unchanged archives and preserved preferences, in a
  private headless Sway compositor. Host Hyprland integration is not covered.
- [Draft downloads](diagnostics/2026-11-24-runtime-profiles-draft-downloads.json):
  all nine assets match original sizes, SHA256 and GitHub digests.

The frozen source passed core, all four managed-race shards, QML and aggregate
source CI. The protected stable-release policy intentionally rejects this topic
prerelease. Wider native handoff/recording and malformed-input decoder regression
tests are not claimed. Review feedback belongs on
[PR24](https://github.com/thelifeandtimes/omabit/pull/24), with loaded version,
profile, operation and error text; omit credentials. Keep issue22 open until owner
review is complete.
