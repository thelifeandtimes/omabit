# 0.3.2 → 0.3.3 guided patch upgrade

Explicit `--portable`, `--previous`, `--candidate`, `--rootless`, `--output` and
`--run` are required. The previous and candidate paths contain exact published
and frozen release assets, respectively. `--rootless` supplies the verified
launcher bundle used by the previous installer. The new output must be a direct
`patch-upgrade-033-NAME` child of the existing private labs root.

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
