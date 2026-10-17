# RC.9 publication and local fresh-setup handoff

The user requested committing the integrated PR work, publishing a release, then
clearing the local machine for one final fresh onboarding test. Stable upgrades
remain a separate release gate; this is an explicitly requested development reset.

## Release

- Published: <https://github.com/thelifeandtimes/omabit/releases/tag/v0.1.0-rc.9>.
- Source: `d9b180fdd1ec5c34084c93109b005b9af4d49c7e`.
- Co-author: wundrellama / ~dinnyt-divsud, whose PRs #2/#3 were adapted to RC.8.
- Source CI: <https://github.com/thelifeandtimes/omabit/actions/runs/39053250523>.
- Eight downloaded GitHub assets passed checksums, archive provenance, executable
  and isolated installer verification. Native bootstrap downloaded and verified
  all three tools from an empty isolated cache, checked clean build revisions,
  exposed recovery/guided-import CLI support, and reused the verified cache.
- Archive SHA-256:
  `95c61612beb136be527597d4f5df29e15a1f0ee8aecc097ae654aa9f03fe8a8f`.

## Reviewed local state

Host: **sourceport**. Runner `runner-a37fca697901780463cba5543e081d2b` had one healthy
running test comet, `a-test-comet-rc8`, revision1, with identity
`~tadnus-pacpex-postep-rabbyl--lavmug-habnub-mildus-marzod`. Only the default storage
root was configured, no operations were active, and the private engine had exactly
the expected container and mount. Its two Vere processes matched that container's
host cgroup. No unrelated Vere processes were found.

The identity was checked against the protected list and prefixes. The active pier
root contained only this test pier; earlier retirement trees were separate. System
Docker was inactive with its socket listening, so cleanup never connected to that
socket or activated the service.

## Retirement

Completed **2026-10-17 17:27:52 UTC**:

- Disabled the plugin and gracefully stopped the selected comet through its pinned
  runner. The Stop receipt succeeded and private containers/Vere processes exited.
- Read the stopped pier's identity and complete metadata-tree fingerprint, then
  retired host units, installation, private engine/state, default pier root,
  enrollment, settings/cache, plugin and transient app directories with same-parent
  no-overwrite-by-preflight renames.
- Preserved existing pier archives, account/group/subordinate-ID files, preexisting
  lingering, the user manager and shared Docker service/socket state.
- The preserved pier has **7 files / 648,405,004 bytes** and fingerprint:
  `6f9598c9cb5862c20bff684376abfdaaee643c8192f31eb78fc3e899210a07d4`.
  Identity and fingerprint matched after retirement and again after shell reload.
  This is a metadata-tree fingerprint, not a full backup-content hash.

Retirement manifest:

```text
/var/lib/omabit-retired/sourceport-rc9-reset-20261017T171609Z/receipt.json
```

Preserved pier:

```text
/srv/urbit/.piers.retired-20261017T171609Z/livenet/comets/a-test-comet-rc8/pier
```

## Fresh-install gate

The unlocked shell was cold-restarted at **17:31:20 UTC**, new instance
`lxc98cb2nt`, PID `436665`. Verification at **17:31:24 UTC** confirmed Omabit absent
from the live registry, its paths/sockets absent and services inactive. Other
plugins/settings and the user manager remained intact; shared Docker stayed inactive.

The frozen RC.9 setup helper, using isolated temporary desktop state, reported:

```text
ready: false
host_present: false
can_setup: true
collector runners: []
fresh_install_ready: true
```

Receipts: `/tmp/opencode/sourceport-rc9-reset-20261017T171609Z/` contains
`reset-result.json`, `verification.json` and the isolated `setup-status.json`.
No setup or enrollment was performed on the user's behalf.

Next action, as the ordinary desktop user:

```bash
omarchy plugin install https://github.com/thelifeandtimes/omabit.git --enable
```

Then open the widget and choose **Start Setup →**.
