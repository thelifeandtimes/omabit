# Omabit 0.2.0

Guided remote onboarding and a more compact desktop experience, with the existing
owner-user runtime and non-destructive management-upgrade contract. Runner schema21
and desktop state/bootstrap API1 are retained.

## Highlights

- Connect through an existing SSH alias, user and port. An authentication terminal
  handles OpenSSH and remote sudo; each desktop/runner receives a distinct restricted
  management key. Enrollment follows a fresh unattended identity/role check.
- Pair existing owner-user runners without replacing their binaries, restarting
  services or altering ships. Approved fresh-host setup installs missing supported
  native dependencies and uses the established rootless owner-user installer.
- Resume interrupted setup with its original endpoint, key and installer binding.
  Revoke only that desktop's grant, including authorization on existing sessions.
  Existing dedicated-port connections remain supported.
- Passive local SSH host suggestions, compact runner summaries and expandable
  technical details. Runtime errors remain bound to their ship and configuration.
- Clear permanent/comet identity guidance, Recommended Apps selected by default,
  always-open provisioning imports and compact storage rows. Destructive actions
  retain explicit review and typed confirmation.
- Inline running loom size, contextual priority-log filtering, compact log controls,
  overlay progress and direct pack/meld/roll actions with existing eligibility gates.

## Install and update

```sh
omarchy plugin install https://github.com/thelifeandtimes/omabit.git --enable
```

For an existing stable desktop:

```sh
omarchy plugin update thelifeandtimes.omabit
omarchy shell thelifeandtimes.omabit.bootstrap status
```

Explicit profile release pins remain selected across plugin updates. To follow the
published desktop again, run `scripts/omabit-dev.py select default --release follow`
from the installed checkout. A desktop update does not upgrade a host runner.

## Upgrade and support scope

Stable0.1.0/schema21 owner-user deployments use the existing journaled management
installer. Pairing an existing stable0.1.0 runner needs only access changes; it does
not require upgrading that runner. Preserve running ships, stopped intent, pier
ownership/locations, access grants, enrollment, configuration and operation history.
Do not reset or re-enroll a host to upgrade it. See [upgrade policy](upgrade-policy.md)
and [host upgrade procedure](host-compatibility-upgrades.md).

Hosting targets Linux/AMD64 with systemd, cgroup v2 and the isolated owner-user
rootless engine. Docker27.2.0–29.x/API1.47 remains the compatibility policy, with
native package provenance and exact-stack qualification reported separately.
Official content-pinned Vere4.6 and the existing Recommended Apps pill are retained.

Guided setup supports normal owner SSH public-key authentication; custom forced
commands, chroot or additional authentication requirements may require the prepared
endpoint workflow. APT dependency provisioning and custom bastion policies have not
received the same live coverage as the documented Arch/Pacman flow. Neither a
stable label nor desktop isolation certifies every host/runtime combination.

See [remote setup](remote-runner-setup.md) for access/retry/revocation details and the
frozen release qualification for the exact tested artifacts, upgrades and limits.
