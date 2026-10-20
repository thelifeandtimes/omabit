# RC.15 sourcebox fresh-setup correction

Published: [v0.1.0-rc.15](https://github.com/thelifeandtimes/omabit/releases/tag/v0.1.0-rc.15),
2026-10-20 at04:58:10UTC. Frozen source:
`55ddde08c323477f78675eafdb01c787fe1c1ca3`.

## Observed regression

The user's RC.14 screenshot showed an approved installation checkbox, a disabled
setup button, verbose metadata-only engine diagnostics and duplicate NSS errors.
Sourcebox inspection established:

- Omarchy4.0.3-1; existing local UID/GID1000 owner.
- Standard `passwd: files systemd` and `group: files [SUCCESS=merge] systemd`.
- Root-owned mode0644 `/etc/subuid` and `/etc/subgid`, both granting100000:65536.
- Native `getsubids` recognizes both existing grants.
- No installed Omabit owner profile, reservation file or Omabit engine/runner unit.

The metadata and privileged setup paths conflated consuming existing OS delegation
with allocating new ranges. The former incorrectly required a prior Omabit profile.
The new shared policy permits reuse for an exactly bound local account while
retaining allocation restrictions and explicit reservation policies.

## Actual frozen-helper comparison

After user-approved interactive SSH authentication, the exact frozen RC.15 desktop
helper was hash-checked and run from a temporary user directory on sourcebox. Only
`omabit --format json setup-status` was executed. The temporary executable was
removed afterward; no setup command, elevation or service operation was submitted.

| Result | Published RC.14 | Frozen RC.15 |
| --- | --- | --- |
| `ready` | false | false |
| `host_present` | false | false |
| `can_setup` | false | **true** |
| `subuid` | blocked | **pass** |
| `subgid` | blocked | **pass** |

All required RC.15 prerequisite checks passed. Configuration hashes, ownership,
mode, device/inode and mtime evidence for passwd/group/NSS/grants were unchanged;
the absent installation files remained absent. All12 observed Urbit PID/start-time
records were unchanged.

Candidate SHA-256:
`a8fff928d8b8a4f954ddfc63c55b184da7fa87eb84378ab5c857f357614fe4aa`.
Private evidence: `build/rc15-release/sourcebox-preflight-2.json`, SHA-256
`1c22dd56f085c62befe268914ba049be398e57bd1c7f9ac4cdef1f57e8784957`.
An earlier SSH attempt failed authentication and is retained separately; it is not
counted as target verification. Credentials were entered in a terminal and not logged.

## Source, UI and artifact checks

[CI39370936447](https://github.com/thelifeandtimes/omabit/actions/runs/39370936447)
passed. Local tests/vet and targeted race checks passed for host setup/install,
onboarding and the desktop helper. Regressions cover unsafe/overlapping/short grants,
changed account binding, reservation conflicts, both-files-before-allocation,
installed-owner preservation and authorization/progress ordering.

Actual offscreen Qt checks verify collapsed technical diagnostics, checked-off
progress through authorization and the completed checklist. The frozen bundle
passed real Quickshell loading/profile/recovery tests with the actual UI/helpers.
Two independent clean builds were byte-identical; archive/provenance/schema and
isolated installer tests passed. Govulncheck v1.9.0 found no vulnerabilities in the
three binaries. Independently downloaded draft assets passed the same checks.

## User retry

Update the native plugin and wait for loaded RC.15. Reopen local setup, approve the
installation checkbox, and choose **Set up this computer**. Sourcebox's observed
configuration requires no manual ID-range or NSS change to pass these prerequisites.

This verifies the fixed prerequisite decision on the actual workstation and the
guided flow in isolated tests. Completion of the real privileged installation and
its password prompt remains the user's next acceptance step; no live setup success
is inferred merely from `can_setup:true`.
