# Remote runner setup and host-specific help

Use the intended installation's matched `omabit`, `urbit-collector` and
`urbit-runner` generation. The examples below use those selected executables.
Keep the desktop profile's XDG configuration, state, data and cache roots when
running commands or launching its terminal. A collector `--config-dir` override
alone leaves the remote-setup credential directory selected by XDG configuration.

## Tested coverage

These observations describe the 0.4.0 development line, recorded 2026-11-06.
Check the installed tool generation before assuming it contains these fixes.
A compatible host report permits the corresponding operations. Broader release
qualification still depends on the exact platform, package and runtime tuple.

| Host or scenario | Observed coverage |
| --- | --- |
| Ubuntu 22.04.5 amd64 cloud VM, systemd/cgroup v2 | Native missing-package install, all eight authenticated package roles, restricted-key enrollment, empty inventory, a healthy bare fake ship, second-desktop pairing/revocation during that ship's lifetime, interrupted-install recovery and cold-boot reconnection. Tested Docker 29.1.3 with RootlessKit 0.14.6. |
| Arch Linux x86-64 cloud image `20261101.620452`, systemd/cgroup v2 | Stale-mirror failure, full guest update and reboot, same-attempt recovery, native missing-package install, all eight authenticated package roles, restricted-key enrollment and a healthy bare fake ship. Second-desktop pairing/revocation preserved the running ship and services. Tested Docker 29.9.3 with RootlessKit 3.3.0. |
| Rooftop, recorded Ubuntu 22.04 x86-64 | Operator confirmed the manually driven configuration worked using the separate trial plugin. The report leaves the fresh-install versus existing-runner branch unspecified. |
| Debian 13.7 amd64 cloud VM | Stock Trixie, updates, security and backports offered Docker 26.1.5. Setup refused it before package installation, retained the attempt/key/installer, and supplied actionable guidance on retry. Successful Debian provisioning requires a separately verified compatible package source. |

The tested Linux runner contract uses x86-64, systemd, unified cgroup v2 and an
existing non-root pier owner. The desktop side ran in isolated profiles on the
developer's Omarchy machine. The Arch guest covered the underlying runner path.
A fresh full Omarchy desktop image remains untested in this install matrix.
Other architectures, init systems, containers used as hosts, customized SSH
policy, jump hosts and bespoke native repositories need task-specific
investigation. Rolling Arch packages can change after the recorded run.

## Start or resume

The desktop's **Connect another runner** action launches this same setup flow:

```sh
omabit remote-setup --host SSH_ALIAS --owner OWNER
omabit remote-setup-status --id SETUP_ID
omabit remote-setup --id SETUP_ID
```

Use `--ssh-config FILE` when the intended SSH alias lives in an explicit file.
Remote owner defaults to the SSH login user and must already exist. Existing
runners retain their owner. SSH and sudo handle their prompts inside the setup
terminal. Read the detected action before approving the authorized installation
or access grant.

Resume preserves the original dedicated key and installer. An installed desktop
update leaves the retained attempt bound to its original helper. If that helper
contains the failure being fixed, inspect its status and host journals before
choosing a supported repair or new attempt. Keep existing runner identity,
grants, piers and journals available throughout recovery. An unavailable runner
or a missing response calls for reconciliation before any replacement.

Successful setup verifies a fresh restricted-key connection, expected runner ID,
role and inventory before saving enrollment. Verify those results through
`runners`, then identity-bound `hello`, `snapshot` and `host-check`.
Follow [operations](operations.md) for the explicit runner flags and asynchronous
operation checks.

## What the agent may need to resolve

**Package availability and update state.** On the runner host, read
`/etc/os-release`, `uname -m`, the installed package versions and the configured
candidate (`apt-cache policy docker.io` or `pacman -Si docker`). The current
engine policy admits stable Docker 27.2.0 or newer and below 30.0.0, with API 1.47.
Let the selected helper decide compatibility. It also authenticates package
payloads against the configured native signing authority.

Arch images can retain metadata for packages the rolling mirrors have removed.
For download 404s, finish an authorized full-system update using
`pacman -Syu` on Arch or `omarchy update` on Omarchy, including any required
reboot, then resume the same attempt. Avoid refreshing only the package database
before selectively installing dependencies. Omabit installs missing dependencies
from configured repositories. Repository/key additions, package replacement and
system updates require authority covering those host changes. If current sources
lack a compatible candidate, explain the available host/package-source choices
and verify the chosen source before provisioning.

**SSH and owner policy.** Resolve the user's intended alias, login user, port,
proxy/jump route, SSH config and owner. Host-key changes, MFA, chroots,
`ForceCommand`, alternate authorized-key files and owner permissions can need
administrator help. Check whether the final noninteractive connection can use its
dedicated key. Preserve the existing administrator key and login policy while
resolving the specific rejected requirement.

**Rootless engine startup.** Inspect the selected owner's unit and bounded
journal output, explicit `/run/user/UID/urbit-docker.sock`, subordinate UID/GID
ranges, user namespaces and cgroup delegation. The Jammy fix detects older
RootlessKit options and delegates CPU, memory and pids before initial engine
activation. Preserve administrator-edited units and established stopped intent
when deciding how to apply an authorized repair. Keep package signature checks,
resource limits and the owner-bound socket throughout diagnosis.

**First boot.** Setup completion establishes management connectivity. To verify
runtime readiness, use a disposable fake ship only within the authorized test
scope, retain the start receipt, and wait for that operation to succeed with a
running/healthy instance and the expected HTTP ship identity. Use the default
loom. The VM checks used 7 GiB RAM and 12 GiB virtual disks. A 3 GiB guest hit
admission limits, and an explicitly reduced 1 GiB loom exhausted the bare pill's
heap. Check current admission evidence, disk space, image/pill retrieval, memory
and boot logs before retrying. Scope any cleanup to the identified test instance.

Report the tool generation, distro/kernel, selected native versions, last
completed stage, exact failed check and the next authorized action. Exclude
private keys, login codes and cookies. Preserve the tested versus inferred
distinction when handing the case to the user or another agent.
