# 0.3.3 → 0.3.4 guided patch upgrade

Explicit `--portable`, `--previous`, `--candidate`, `--rootless`, `--output` and
`--run` are required. The previous and candidate paths contain exact published
and frozen release assets, respectively. `--rootless` supplies the verified
launcher bundle used by the previous installer. The new output must be a direct
`patch-upgrade-034-NAME` child of the existing private labs root.

This specializes the retained0.3.1 QML-button rehearsal for schema47 →47. It uses
actual sealed bundles, a pristine immutable base, offline fresh fakes, actual
service installation, authorization denial/retry, and a manager restart. It checks
unchanged running container/engine processes, stopped intent, fleet history,
identities and access. The guest alone gets an exact-program polkit fixture.

The launcher enforces one4 GiB scratch allowance under the combined40 GiB storage
budget. No host Docker, runner, keyring, desktop, SSH or network is exposed.
It never reuses an overlay. Keep only compact results after a successful powered-
off run; classify and record generated disks before retirement. This fixture does
not test human password input, the stock plugin updater, remote upgrades or power
cuts. Desktop preferences and cold/hot QML activation have a separate native test.

The 0.3.4 rehearsal also makes a separate copy of its own stopped fake, adds
synthetic macOS metadata, reproduces the old validator's refusal, then verifies
guided identity discovery and a complete stopped import with the new runner.
All companion bytes survive and the unrelated running fake retains its process.
No production pier or prior manual review state is used.

If owner-only accounting cannot descend into rootless overlay work directories,
the launcher records that limitation and reserves an additional 4 GiB. Any other
measurement failure refuses the run. Reported charged usage includes this reserve;
it is not a claim that inaccessible directories were fully measured.
