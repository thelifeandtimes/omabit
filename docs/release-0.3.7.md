# Omabit 0.3.7 — one-click setup and verified identity recovery

The supported upgrade is **0.3.6 → 0.3.7** on Linux/AMD64 with an owner-user runner.

## Changes

- **One-click setup on Arch/Omarchy:** choose **Set up this computer**, then
  authorize the OS prompt. Missing native prerequisites are installed and verified
  as part of that operation. Package details are under **View details**.
- **Clearer first-run flow:** direct local setup, ship creation after a runner is
  connected, and progress through native packages, services and enrollment.
- **Bundled SSH key generation:** initial setup no longer requires `ssh-keygen`
  before native packages are installed. Existing OpenSSH-format Ed25519 desktop
  keys are reused, including after an interrupted attempt.
- **Working key-file selection:** the non-native Qt chooser stays on its owning
  layer-shell surface and supports selection, cancellation and reopening.
- **Verified identity correction:** local owner CLI `repair-boot-identity` can
  recover a mistyped keyed first-boot identity. It verifies public and private
  identity, gracefully stops the exact boot process, corrects registration and
  preserves the same pier and history. See [procedure](boot-identity-repair.md).

This release uses runner schema 47, desktop state format 1, bootstrap API 1,
Vere 4.6 and the same Recommended Apps pill as 0.3.6. Building from source
requires Go 1.26 or later. Initial Arch/Omarchy setup can install missing native
packages. Existing installations use the management upgrade described below.

## Installation and upgrade

```sh
# New desktop
omarchy plugin install https://github.com/thelifeandtimes/omabit.git --enable

# Existing desktop
omarchy plugin update thelifeandtimes.omabit
```

For a fresh local host, select **Set up this computer** and authorize the OS prompt.
For an existing host, allow the verified bundle to load, then use **Update
Dependencies** when offered. Remote runner updates remain explicit. A desktop
update does not itself restart a ship or change its runtime.

From an earlier stable version, follow the [upgrade policy](upgrade-policy.md)
to reach 0.3.6 first. Identity repair requires schema 47 and the pinned keyed
runtime. For stopped or interrupted work, follow the existing recovery procedure
before upgrading.

## Upgrade qualification

The isolated service rehearsal used the published 0.3.6 tools and the 0.3.7
artifacts built from `076f97280960`. A running ship kept its container ID, PID and
start time across an authorization denial, retry, completed update and manager
restart. A second ship stayed stopped. The rehearsal also checked runner identity,
SSH access, storage configuration, operation history and a repeated update check.

The native desktop passed hot activation, recovery from a failed candidate and
cold restart with saved preferences. The key picker passed real pointer and
keyboard input on a private Wayland compositor. The operator also confirmed the
fresh-install trial and same-pier identity repair.

The service rehearsal covered a local Arch/Omarchy owner-user installation.
Remote runner upgrades, other distributions, manual password entry and power
loss are outside its scope. See the [qualification record](releases/0.3.7/qualification.json)
and [upgrade evidence](releases/0.3.7/rehearsal.json).
